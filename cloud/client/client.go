// Package client 实现 Go Client Engine 复用的 Controller 解析与 Edge ClientGateway 协议。
// 它只运输 generated Cloud Proto，不选择 Endpoint Route，也不拥有 PeerSession generation。
package client

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	"github.com/anytty/anytty/cloud/securetransport"
	"github.com/anytty/anytty/cloud/ticket"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Signer 是 Android Keystore 或 desktop secure credential 对 canonical Cloud proof 的异步签名边界。
type Signer interface {
	Sign(context.Context, []byte) ([]byte, error)
}

// Config 是当前 Cloud account/profile 解析出的 Controller endpoint；不属于 Endpoint/Route 真值。
type Config struct {
	ControllerAddress    string
	ControllerServerName string
	ControllerCAPEM      []byte
	BootID               string
	SoftwareVersion      string
	Now                  func() time.Time
}

// Client 持有进程级 boot identity；每次 Resolve/Exchange 仍使用当前调用的 grant 和 generation。
type Client struct {
	config Config
	bootID string
}

type daemonLifecycleError struct {
	code string
}

func (err *daemonLifecycleError) Error() string {
	if err.code == cloudprotocol.DaemonDeletedCode {
		return "daemon Cloud enrollment was deleted"
	}
	return "daemon Cloud access is temporarily disabled"
}

// DaemonLifecycleCode returns the stable ClientGateway state rejection code.
func DaemonLifecycleCode(err error) string {
	var lifecycle *daemonLifecycleError
	if errors.As(err, &lifecycle) {
		return lifecycle.code
	}
	return ""
}

func IsDaemonBlocked(err error) bool {
	return DaemonLifecycleCode(err) == cloudprotocol.DaemonBlockedCode
}
func IsDaemonDeleted(err error) bool {
	return DaemonLifecycleCode(err) == cloudprotocol.DaemonDeletedCode
}

type daemonOfflineError struct {
	cause error
}

func (err *daemonOfflineError) Error() string { return "daemon is offline" }
func (err *daemonOfflineError) Unwrap() error { return err.cause }

func IsDaemonOffline(err error) bool {
	var offline *daemonOfflineError
	return errors.As(err, &offline)
}

// EntitlementError is a stable Cloud commercial-policy rejection returned by signaling.
type EntitlementError struct {
	Failure *cloudv1.CloudEntitlementFailure
}

func (err *EntitlementError) Error() string {
	if err == nil || err.Failure == nil || strings.TrimSpace(err.Failure.GetMessage()) == "" {
		return "Cloud entitlement denied"
	}
	return err.Failure.GetMessage()
}

func EntitlementFailure(err error) *cloudv1.CloudEntitlementFailure {
	var entitlement *EntitlementError
	if errors.As(err, &entitlement) && entitlement.Failure != nil {
		return proto.Clone(entitlement.Failure).(*cloudv1.CloudEntitlementFailure)
	}
	for _, detail := range status.Convert(err).Details() {
		if failure, ok := detail.(*cloudv1.CloudEntitlementFailure); ok {
			return proto.Clone(failure).(*cloudv1.CloudEntitlementFailure)
		}
	}
	return nil
}

// RouteResolution 是一次已认证的目录结果，或者由本机缓存 Edge locator 与原始 daemon grant 重建。
type RouteResolution struct {
	locator          *cloudv1.EdgeLocator
	routeGrant       *cloudv1.SignedEnvelope
	pairingBootstrap *remoteauthpb.PairingManagedRouteSeed
	pairingAdmission *cloudv1.PairingAdmission
	cachedLocator    bool
}

func NewCachedRoute(locator *cloudv1.EdgeLocator, routeGrant *cloudv1.SignedEnvelope) (*RouteResolution, error) {
	return newCapabilityRoute(locator, routeGrant, true)
}

func newCapabilityRoute(locator *cloudv1.EdgeLocator, routeGrant *cloudv1.SignedEnvelope, cached bool) (*RouteResolution, error) {
	if err := validateEdgeLocator(locator); err != nil || routeGrant == nil {
		return nil, errors.New("cached Cloud Route is incomplete")
	}
	return &RouteResolution{
		locator:       proto.Clone(locator).(*cloudv1.EdgeLocator),
		routeGrant:    proto.Clone(routeGrant).(*cloudv1.SignedEnvelope),
		cachedLocator: cached,
	}, nil
}

func (resolution *RouteResolution) Locator() *cloudv1.EdgeLocator {
	if resolution == nil || resolution.locator == nil {
		return nil
	}
	return proto.Clone(resolution.locator).(*cloudv1.EdgeLocator)
}

// SignalSession 持有一个已经完成 offer/answer 的 ClientGateway 流。
// 该流跟随 ReadyPeerSession 存活，使 Edge 的纯内存客户端投影与真实 P2P 生命周期一致；terminal 数据仍只走 DataChannel。
type SignalSession struct {
	answer      *cloudv1.EdgeAnswer
	connection  *grpc.ClientConn
	stream      cloudv1.ClientGateway_ConnectClient
	cancel      context.CancelFunc
	senderID    string
	bootID      string
	edgeID      string
	edgeBootID  string
	sessionID   string
	sendMu      sync.Mutex
	nextSendSeq uint64
	decisionMu  sync.Mutex
	decision    *signalPathDecision
	releaseMu   sync.Mutex
	release     *signalSessionRelease
	closeOnce   sync.Once
	closeErr    error
	done        chan struct{}
	doneOnce    sync.Once
	errMu       sync.Mutex
	err         error
}

const signalPathDecisionRetryInterval = 250 * time.Millisecond
const signalTransportKeepaliveTime = 15 * time.Second
const signalTransportKeepaliveTimeout = 5 * time.Second

type signalStreamOwner struct {
	parent                 context.Context
	cancel                 context.CancelFunc
	stopParentCancellation func() bool
}

func newSignalStreamOwner(parent context.Context) (context.Context, *signalStreamOwner) {
	streamContext, cancel := context.WithCancel(context.WithoutCancel(parent))
	return streamContext, &signalStreamOwner{
		parent:                 parent,
		cancel:                 cancel,
		stopParentCancellation: context.AfterFunc(parent, cancel),
	}
}

// Retain transfers the stream from the route attempt to the returned physical session.
// A cancellation already in flight wins this fence and prevents a stale winner publish.
func (owner *signalStreamOwner) Retain() error {
	if owner == nil || owner.parent == nil || owner.cancel == nil || owner.stopParentCancellation == nil {
		return errors.New("Cloud signaling stream owner is unavailable")
	}
	if !owner.stopParentCancellation() || owner.parent.Err() != nil {
		owner.cancel()
		return context.Cause(owner.parent)
	}
	return nil
}

func (owner *signalStreamOwner) Close() {
	if owner == nil {
		return
	}
	if owner.stopParentCancellation != nil {
		_ = owner.stopParentCancellation()
	}
	if owner.cancel != nil {
		owner.cancel()
	}
}

type signalPathDecision struct {
	id       string
	decision cloudv1.CloudPathDecision
	acked    bool
	ack      chan struct{}
}

type signalSessionRelease struct {
	id    string
	acked bool
	ack   chan struct{}
}

// SignalSessionCloseError is a typed, authenticated Edge decision. Transient
// gRPC failures remain ordinary errors so the runtime can reconnect them.
type SignalSessionCloseError struct {
	Code    cloudv1.SignalSessionCloseCode
	Message string
}

func (err *SignalSessionCloseError) Error() string {
	if err == nil || strings.TrimSpace(err.Message) == "" {
		return "Cloud signaling session was closed"
	}
	return err.Message
}

func IsAdminDisconnect(err error) bool {
	var closed *SignalSessionCloseError
	return errors.As(err, &closed) && closed.Code == cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT
}

// Done closes when Edge ends the signaling stream or the local owner closes it.
func (session *SignalSession) Done() <-chan struct{} {
	if session == nil || session.done == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return session.done
}

func (session *SignalSession) Err() error {
	if session == nil {
		return nil
	}
	session.errMu.Lock()
	defer session.errMu.Unlock()
	return session.err
}

func (session *SignalSession) watch() {
	expectedSequence := uint64(4)
	var err error
	for {
		var event *cloudv1.EdgeSignal
		event, err = session.stream.Recv()
		if err != nil {
			break
		}
		if event.GetProtocolVersion() != cloudprotocol.ClientGatewayVersion || strings.TrimSpace(event.GetMessageId()) == "" ||
			event.GetSenderId() != session.edgeID || event.GetBootId() != session.edgeBootID || event.GetConnectionId() != session.sessionID ||
			event.GetStreamSeq() != expectedSequence || event.GetSentAt() == nil || event.GetSentAt().CheckValid() != nil {
			err = errors.New("Cloud signaling post-answer envelope is invalid")
			break
		}
		expectedSequence++
		if closedErr, closed := signalSessionCloseErrorAtSequence(event, session.edgeID, session.edgeBootID, session.sessionID, event.GetStreamSeq()); closed {
			err = closedErr
			break
		}
		if ack := event.GetPathDecisionAck(); ack != nil {
			if err = session.acceptPathDecisionAck(ack); err != nil {
				break
			}
			continue
		}
		if ack := event.GetSessionReleaseAck(); ack != nil {
			if err = session.acceptSessionReleaseAck(ack); err != nil {
				break
			}
			continue
		}
		err = errors.New("Cloud signaling stream returned an unexpected post-answer message")
		break
	}
	log.Printf(
		"anytty cloud signaling session=%s stage=stream_ended cause=%s grpc_code=%s error=%q",
		session.sessionID,
		signalStreamEndCause(err),
		status.Code(err),
		signalStreamErrorText(err),
	)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		session.errMu.Lock()
		session.err = err
		session.errMu.Unlock()
	}
	session.doneOnce.Do(func() { close(session.done) })
}

func signalStreamEndCause(err error) string {
	switch {
	case err == nil:
		return "nil"
	case errors.Is(err, io.EOF):
		return "eof"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "grpc_error"
	}
}

func signalStreamErrorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Answer 返回 daemon 生成的不可变 SDP answer 投影。
func (session *SignalSession) Answer() *cloudv1.EdgeAnswer {
	if session == nil || session.answer == nil {
		return nil
	}
	return proto.Clone(session.answer).(*cloudv1.EdgeAnswer)
}

// ConfirmPath commits the authenticated ICE winner only after Edge acknowledges
// every release side effect associated with that decision.
func (session *SignalSession) ConfirmPath(ctx context.Context, path cloudv1.SelectedCloudPath) error {
	decision := cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_UNSPECIFIED
	switch path {
	case cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_DIRECT:
		decision = cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_DIRECT
	case cloudv1.SelectedCloudPath_SELECTED_CLOUD_PATH_RELAY:
		decision = cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_RELAY
	default:
		return errors.New("selected Cloud path is invalid")
	}
	return session.decidePath(ctx, decision)
}

// AbandonPath releases a provisional Relay reservation and runtime session.
// EOF and Done never satisfy this barrier; only a matching authenticated ACK does.
func (session *SignalSession) AbandonPath(ctx context.Context) error {
	return session.decidePath(ctx, cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_ABANDON)
}

func (session *SignalSession) decidePath(ctx context.Context, decision cloudv1.CloudPathDecision) error {
	if session == nil || session.stream == nil || strings.TrimSpace(session.sessionID) == "" {
		return errors.New("Cloud signaling session is unavailable")
	}
	if ctx == nil {
		return errors.New("Cloud path decision context is required")
	}
	state, err := session.beginPathDecision(decision)
	if err != nil {
		return err
	}
	stopCancellation := func() bool { return true }
	if session.cancel != nil {
		stopCancellation = context.AfterFunc(ctx, session.cancel)
	}
	defer stopCancellation()
	for {
		if session.pathDecisionAcknowledged(state) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for Cloud path decision acknowledgment: %w", context.Cause(ctx))
		case <-session.Done():
			if session.pathDecisionAcknowledged(state) {
				return nil
			}
			if cause := context.Cause(ctx); cause != nil {
				return fmt.Errorf("wait for Cloud path decision acknowledgment: %w", cause)
			}
			return session.pathDecisionTerminalError()
		default:
		}
		if err := session.sendPathDecision(state); err != nil {
			if session.pathDecisionAcknowledged(state) {
				return nil
			}
			if cause := context.Cause(ctx); cause != nil {
				return fmt.Errorf("wait for Cloud path decision acknowledgment: %w", cause)
			}
			return fmt.Errorf("send Cloud path decision: %w", err)
		}
		timer := time.NewTimer(signalPathDecisionRetryInterval)
		select {
		case <-state.ack:
			stopSignalDecisionTimer(timer)
			return nil
		case <-session.Done():
			stopSignalDecisionTimer(timer)
			if session.pathDecisionAcknowledged(state) {
				return nil
			}
			if cause := context.Cause(ctx); cause != nil {
				return fmt.Errorf("wait for Cloud path decision acknowledgment: %w", cause)
			}
			return session.pathDecisionTerminalError()
		case <-ctx.Done():
			stopSignalDecisionTimer(timer)
			return fmt.Errorf("wait for Cloud path decision acknowledgment: %w", context.Cause(ctx))
		case <-timer.C:
		}
	}
}

func stopSignalDecisionTimer(timer *time.Timer) {
	if timer == nil || timer.Stop() {
		return
	}
	select {
	case <-timer.C:
	default:
	}
}

func (session *SignalSession) beginPathDecision(decision cloudv1.CloudPathDecision) (*signalPathDecision, error) {
	if decision != cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_DIRECT &&
		decision != cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_RELAY &&
		decision != cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_ABANDON {
		return nil, errors.New("Cloud path decision is invalid")
	}
	session.decisionMu.Lock()
	defer session.decisionMu.Unlock()
	if session.decision == nil {
		session.decision = &signalPathDecision{id: uuid.NewString(), decision: decision, ack: make(chan struct{})}
	} else if session.decision.decision != decision {
		return nil, errors.New("Cloud path decision is already committed to another outcome")
	}
	return session.decision, nil
}

func (session *SignalSession) sendPathDecision(state *signalPathDecision) error {
	session.sendMu.Lock()
	defer session.sendMu.Unlock()
	sequence := session.nextSendSeq
	if sequence < 3 {
		sequence = 3
	}
	session.nextSendSeq = sequence + 1
	return session.stream.Send(&cloudv1.ClientSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: uuid.NewString(), SenderId: session.senderID, BootId: session.bootID,
		ConnectionId: session.sessionID, StreamSeq: sequence, SentAt: timestamppb.Now(),
		Payload: &cloudv1.ClientSignal_PathDecision{PathDecision: &cloudv1.ClientPathDecision{
			SessionId: session.sessionID, DecisionId: state.id, Decision: state.decision,
		}},
	})
}

func (session *SignalSession) acceptPathDecisionAck(ack *cloudv1.EdgePathDecisionAck) error {
	session.decisionMu.Lock()
	defer session.decisionMu.Unlock()
	state := session.decision
	if state == nil || ack.GetSessionId() != session.sessionID || ack.GetDecisionId() != state.id || ack.GetDecision() != state.decision {
		return errors.New("Cloud path decision acknowledgment is invalid")
	}
	if !state.acked {
		state.acked = true
		close(state.ack)
	}
	return nil
}

func (session *SignalSession) pathDecisionAcknowledged(state *signalPathDecision) bool {
	session.decisionMu.Lock()
	defer session.decisionMu.Unlock()
	return session.decision == state && state.acked
}

func (session *SignalSession) pathDecisionTerminalError() error {
	if err := session.Err(); err != nil {
		return fmt.Errorf("Cloud signaling ended before path decision acknowledgment: %w", err)
	}
	return errors.New("Cloud signaling ended before path decision acknowledgment")
}

// ReleaseAndWait tears down a path that already received its confirmation ACK.
// It closes the local stream only after Edge acknowledges Relay and runtime cleanup.
func (session *SignalSession) ReleaseAndWait(ctx context.Context) error {
	if session == nil || session.stream == nil || strings.TrimSpace(session.sessionID) == "" {
		return errors.New("Cloud signaling session is unavailable")
	}
	if ctx == nil {
		return errors.New("Cloud session release context is required")
	}
	state, err := session.beginSessionRelease()
	if err != nil {
		return err
	}
	stopCancellation := func() bool { return true }
	if session.cancel != nil {
		stopCancellation = context.AfterFunc(ctx, session.cancel)
	}
	defer stopCancellation()
	for {
		if session.sessionReleaseAcknowledged(state) {
			return session.close()
		}
		select {
		case <-ctx.Done():
			return errors.Join(fmt.Errorf("wait for Cloud session release acknowledgment: %w", context.Cause(ctx)), session.close())
		case <-session.Done():
			if session.sessionReleaseAcknowledged(state) {
				return session.close()
			}
			if cause := context.Cause(ctx); cause != nil {
				return errors.Join(fmt.Errorf("wait for Cloud session release acknowledgment: %w", cause), session.close())
			}
			return errors.Join(session.sessionReleaseTerminalError(), session.close())
		default:
		}
		if err := session.sendSessionRelease(state); err != nil {
			if session.sessionReleaseAcknowledged(state) {
				return session.close()
			}
			if cause := context.Cause(ctx); cause != nil {
				return errors.Join(fmt.Errorf("wait for Cloud session release acknowledgment: %w", cause), session.close())
			}
			return errors.Join(fmt.Errorf("send Cloud session release: %w", err), session.close())
		}
		timer := time.NewTimer(signalPathDecisionRetryInterval)
		select {
		case <-state.ack:
			stopSignalDecisionTimer(timer)
			return session.close()
		case <-session.Done():
			stopSignalDecisionTimer(timer)
			if session.sessionReleaseAcknowledged(state) {
				return session.close()
			}
			if cause := context.Cause(ctx); cause != nil {
				return errors.Join(fmt.Errorf("wait for Cloud session release acknowledgment: %w", cause), session.close())
			}
			return errors.Join(session.sessionReleaseTerminalError(), session.close())
		case <-ctx.Done():
			stopSignalDecisionTimer(timer)
			return errors.Join(fmt.Errorf("wait for Cloud session release acknowledgment: %w", context.Cause(ctx)), session.close())
		case <-timer.C:
		}
	}
}

func (session *SignalSession) beginSessionRelease() (*signalSessionRelease, error) {
	session.decisionMu.Lock()
	confirmed := session.decision != nil && session.decision.acked &&
		(session.decision.decision == cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_DIRECT ||
			session.decision.decision == cloudv1.CloudPathDecision_CLOUD_PATH_DECISION_CONFIRM_RELAY)
	session.decisionMu.Unlock()
	if !confirmed {
		return nil, errors.New("Cloud session cannot be released before path confirmation")
	}
	session.releaseMu.Lock()
	defer session.releaseMu.Unlock()
	if session.release == nil {
		session.release = &signalSessionRelease{id: uuid.NewString(), ack: make(chan struct{})}
	}
	return session.release, nil
}

func (session *SignalSession) sendSessionRelease(state *signalSessionRelease) error {
	session.sendMu.Lock()
	defer session.sendMu.Unlock()
	sequence := session.nextSendSeq
	if sequence < 3 {
		sequence = 3
	}
	session.nextSendSeq = sequence + 1
	return session.stream.Send(&cloudv1.ClientSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: uuid.NewString(), SenderId: session.senderID, BootId: session.bootID,
		ConnectionId: session.sessionID, StreamSeq: sequence, SentAt: timestamppb.Now(),
		Payload: &cloudv1.ClientSignal_SessionRelease{SessionRelease: &cloudv1.ClientSessionRelease{
			SessionId: session.sessionID, ReleaseId: state.id,
		}},
	})
}

func (session *SignalSession) acceptSessionReleaseAck(ack *cloudv1.EdgeSessionReleaseAck) error {
	session.releaseMu.Lock()
	defer session.releaseMu.Unlock()
	state := session.release
	if state == nil || ack.GetSessionId() != session.sessionID || ack.GetReleaseId() != state.id {
		return errors.New("Cloud session release acknowledgment is invalid")
	}
	if !state.acked {
		state.acked = true
		close(state.ack)
	}
	return nil
}

func (session *SignalSession) sessionReleaseAcknowledged(state *signalSessionRelease) bool {
	session.releaseMu.Lock()
	defer session.releaseMu.Unlock()
	return session.release == state && state.acked
}

func (session *SignalSession) sessionReleaseTerminalError() error {
	if err := session.Err(); err != nil {
		return fmt.Errorf("Cloud signaling ended before session release acknowledgment: %w", err)
	}
	return errors.New("Cloud signaling ended before session release acknowledgment")
}

// Close 结束 ClientGateway 观察流并释放 Edge gRPC 连接；可以重复调用。
func (session *SignalSession) Close() error {
	return session.close()
}

// AbandonAndWait explicitly abandons the candidate and waits for its cleanup
// ACK before releasing the underlying gRPC connection.
func (session *SignalSession) AbandonAndWait(ctx context.Context) error {
	if ctx == nil {
		return errors.New("Cloud signaling close context is required")
	}
	decisionErr := session.AbandonPath(ctx)
	return errors.Join(decisionErr, session.close())
}

// CloseAndWait is the ownership-close spelling of AbandonAndWait.
func (session *SignalSession) CloseAndWait(ctx context.Context) error {
	return session.AbandonAndWait(ctx)
}

func (session *SignalSession) close() error {
	if session == nil {
		return nil
	}
	session.closeOnce.Do(func() {
		// Cancel first: an in-flight gRPC Send owns sendMu and may otherwise wait
		// forever on a dead network, preventing CloseSend from ever acquiring it.
		if session.cancel != nil {
			session.cancel()
		}
		if session.stream != nil {
			session.sendMu.Lock()
			session.closeErr = normalizeActiveSignalCloseError(session.stream.CloseSend())
			session.sendMu.Unlock()
		}
		if session.connection != nil {
			if err := normalizeActiveSignalCloseError(session.connection.Close()); session.closeErr == nil {
				session.closeErr = err
			}
		}
		session.doneOnce.Do(func() { close(session.done) })
	})
	return session.closeErr
}

func normalizeActiveSignalCloseError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
		return nil
	}
	return err
}

// NewClient 验证 Controller TLS locator；账号 session 在 R7 接入，但 R5 不允许使用明文或跳过证书校验。
func NewClient(config Config) (*Client, error) {
	config.ControllerAddress = strings.TrimSpace(config.ControllerAddress)
	config.ControllerServerName = strings.TrimSpace(config.ControllerServerName)
	config.BootID = strings.TrimSpace(config.BootID)
	config.SoftwareVersion = strings.TrimSpace(config.SoftwareVersion)
	if config.ControllerAddress == "" || config.ControllerServerName == "" {
		return nil, errors.New("Cloud Controller address and TLS server name are required")
	}
	parsedBootID, err := uuid.Parse(config.BootID)
	if err != nil || parsedBootID == uuid.Nil || parsedBootID.String() != config.BootID {
		return nil, errors.New("Cloud client boot ID must be a canonical non-zero UUID")
	}
	if config.SoftwareVersion == "" {
		config.SoftwareVersion = "development"
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Client{config: config, bootID: config.BootID}, nil
}

// Resolve 只在本机没有 Edge locator 或旧 Edge 失效时查询实时 Presence；返回结果仍使用原始 daemon grant 准入。
func (client *Client) Resolve(ctx context.Context, cloudRouteGrant []byte, signer Signer) (*RouteResolution, error) {
	if client == nil || signer == nil || len(cloudRouteGrant) == 0 {
		return nil, errors.New("Cloud route grant and signer are required")
	}
	startedAt := time.Now()
	lastAt := startedAt
	reportTiming := func(stage string) {
		now := time.Now()
		log.Printf("anytty cloud connect stage=%s stage_ms=%d total_ms=%d", stage, now.Sub(lastAt).Milliseconds(), now.Sub(startedAt).Milliseconds())
		lastAt = now
	}
	grant := &cloudv1.SignedEnvelope{}
	if err := proto.Unmarshal(cloudRouteGrant, grant); err != nil {
		return nil, fmt.Errorf("decode CloudRouteGrant: %w", err)
	}
	connection, err := client.dial(client.config.ControllerAddress, client.config.ControllerServerName, client.config.ControllerCAPEM)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	directory := cloudv1.NewDirectoryServiceClient(connection)
	challenge, err := directory.BeginClientRoute(ctx, &cloudv1.BeginClientRouteRequest{CloudRouteGrant: grant})
	if err != nil {
		return nil, fmt.Errorf("begin Cloud route resolution: %w", classifyDaemonLifecycleError(err))
	}
	reportTiming("controller_challenge")
	requestID := uuid.NewString()
	canonical, err := ticket.ClientRouteProofBytes(challenge.GetChallengeId(), challenge.GetChallenge(), grant, requestID)
	if err != nil {
		return nil, err
	}
	proof, err := signer.Sign(ctx, canonical)
	if err != nil {
		return nil, fmt.Errorf("sign Cloud route challenge: %w", err)
	}
	reportTiming("controller_proof")
	resolved, err := directory.ResolveClientRoute(ctx, &cloudv1.ResolveClientRouteRequest{ChallengeId: challenge.GetChallengeId(), RequestId: requestID, ClientProof: proof})
	if err != nil {
		return nil, fmt.Errorf("resolve Cloud route: %w", classifyDaemonLifecycleError(err))
	}
	reportTiming("controller_resolve")
	if resolved.GetEdgeLocator() == nil {
		return nil, errors.New("Cloud route response is incomplete")
	}
	return newCapabilityRoute(resolved.GetEdgeLocator(), grant, false)
}

func classifyDaemonLifecycleError(err error) error {
	if err == nil {
		return nil
	}
	if failure := EntitlementFailure(err); failure != nil {
		return &EntitlementError{Failure: failure}
	}
	grpcStatus := status.Convert(err)
	switch {
	case grpcStatus.Code() == codes.PermissionDenied && grpcStatus.Message() == cloudprotocol.DaemonBlockedCode:
		return &daemonLifecycleError{code: cloudprotocol.DaemonBlockedCode}
	case grpcStatus.Code() == codes.NotFound && grpcStatus.Message() == cloudprotocol.DaemonDeletedCode:
		return &daemonLifecycleError{code: cloudprotocol.DaemonDeletedCode}
	case grpcStatus.Code() == codes.Unavailable && grpcStatus.Message() == "daemon is offline":
		return &daemonOfflineError{cause: err}
	default:
		return err
	}
}

// PairingRoute 只使用 claim offer 内的紧凑 Edge 入口和 CA pin，不访问 Controller。
// 最终 daemon 身份由端到端 DeviceHello 验证，完整 locator 只接受 PairingAccepted 返回值。
func (client *Client) PairingRoute(pairingClaimOffer []byte) (*RouteResolution, error) {
	if client == nil || len(pairingClaimOffer) == 0 {
		return nil, errors.New("Cloud pairing route input is incomplete")
	}
	offer, err := remoteauth.ParsePairingClaimOfferForExchange(pairingClaimOffer)
	if err != nil {
		return nil, err
	}
	var bootstrap *remoteauthpb.PairingManagedRouteSeed
	for _, route := range offer.GetRoutes() {
		managed := route.GetManagedWebrtc()
		if managed == nil {
			continue
		}
		bootstrap = proto.Clone(managed).(*remoteauthpb.PairingManagedRouteSeed)
		break
	}
	if bootstrap == nil {
		return nil, errors.New("pairing claim offer has no Cloud bootstrap route")
	}
	digest := sha256.Sum256(offer.GetClaim())
	admission := &cloudv1.PairingAdmission{
		DaemonId: bootstrap.GetDaemonId(), DeviceId: offer.GetDeviceId(), DevicePublicKey: append([]byte(nil), offer.GetDevicePublicKey()...),
		PairingClaimSha256: digest[:], ExpiresAtUnixNano: offer.GetExpiresAtUnixNano(),
	}
	return &RouteResolution{pairingBootstrap: bootstrap, pairingAdmission: admission}, nil
}

// ProbePresence asks the cached Edge whether the paired daemon currently owns
// an authenticated AgentGateway connection. It deliberately does not fall back
// to the Controller and never creates a signaling session, Relay reservation,
// WebRTC peer, or application protocol client.
func (client *Client) ProbePresence(ctx context.Context, resolution *RouteResolution, identity remoteauth.ClientAccessIdentity, signer Signer, product cloudv1.ClientProduct) (bool, error) {
	capabilityRoute := resolution != nil && resolution.locator != nil && resolution.routeGrant != nil && resolution.pairingBootstrap == nil && resolution.pairingAdmission == nil
	if client == nil || !capabilityRoute || signer == nil || identity.ValidatePublic() != nil || product == cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED {
		return false, errors.New("Cloud presence probe input is incomplete")
	}
	connection, err := client.dial(resolution.locator.GetPublicEndpoint(), resolution.locator.GetServerName(), resolution.locator.GetCaCertificatePem())
	if err != nil {
		return false, markEdgeLocatorUnavailable(err)
	}
	defer connection.Close()
	if err := waitForEdgeTransport(ctx, connection, resolution.edgeTransportTimeout()); err != nil {
		return false, markEdgeLocatorUnavailable(err)
	}
	stream, err := cloudv1.NewClientGatewayClient(connection).Connect(ctx)
	if err != nil {
		return false, markEdgeLocatorUnavailable(err)
	}
	challengeSignal, err := stream.Recv()
	if err != nil {
		return false, markEdgeLocatorUnavailable(err)
	}
	challenge, err := validateClientGatewayChallenge(challengeSignal, resolution.locator.GetEdgeId(), client.config.Now().UTC())
	if err != nil {
		return false, markEdgeLocatorUnavailable(err)
	}
	sessionID := uuid.NewString()
	helloBody := &cloudv1.ClientHello{
		ClientPublicKey: append([]byte(nil), identity.PublicKey...), Product: product, SoftwareVersion: client.config.SoftwareVersion,
		AttemptGeneration: 1, RelayPreference: cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY, PresenceProbe: true,
		Authorization: &cloudv1.ClientHello_CloudRouteGrant{CloudRouteGrant: proto.Clone(resolution.routeGrant).(*cloudv1.SignedEnvelope)},
	}
	hello := &cloudv1.ClientSignal{
		ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: uuid.NewString(), SenderId: identity.Fingerprint,
		BootId: client.bootID, ConnectionId: sessionID, StreamSeq: 1, SentAt: timestamppb.New(client.config.Now().UTC()),
		Payload: &cloudv1.ClientSignal_Hello{Hello: helloBody},
	}
	canonical, err := ticket.ClientHelloProofBytes(challenge, hello, client.config.Now().UTC())
	if err != nil {
		return false, err
	}
	helloBody.ClientProof, err = signer.Sign(ctx, canonical)
	if err != nil {
		return false, err
	}
	if err := stream.Send(hello); err != nil {
		return false, markEdgeLocatorUnavailable(err)
	}
	response, err := stream.Recv()
	if err != nil {
		return false, err
	}
	if response.GetProtocolVersion() != cloudprotocol.ClientGatewayVersion || strings.TrimSpace(response.GetMessageId()) == "" ||
		response.GetSenderId() != challenge.GetEdgeId() || response.GetBootId() != challenge.GetEdgeBootId() || response.GetConnectionId() != sessionID ||
		response.GetStreamSeq() != 2 || response.GetSentAt() == nil || response.GetSentAt().CheckValid() != nil || response.GetPresence() == nil {
		return false, errors.New("Cloud daemon presence response is invalid")
	}
	return response.GetPresence().GetOnline(), nil
}

// Exchange 连接目标 Edge，并用长期 RouteGrant 或一次性 pairing admission 与本次 client proof 完成 offer/answer。
func (client *Client) Exchange(ctx context.Context, resolution *RouteResolution, identity remoteauth.ClientAccessIdentity, signer Signer, product cloudv1.ClientProduct, attemptGeneration uint64, relayPreference cloudv1.RelayPreference, createOffer func(context.Context, *cloudv1.ClientReady) (string, error)) (*SignalSession, error) {
	capabilityRoute := resolution != nil && resolution.locator != nil && resolution.routeGrant != nil && resolution.pairingBootstrap == nil && resolution.pairingAdmission == nil
	pairingRoute := resolution != nil && resolution.locator == nil && resolution.routeGrant == nil && resolution.pairingBootstrap != nil && resolution.pairingAdmission != nil
	if client == nil || (!capabilityRoute && !pairingRoute) || signer == nil || identity.ValidatePublic() != nil || product == cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED || attemptGeneration == 0 || createOffer == nil {
		return nil, errors.New("Cloud signaling input is incomplete")
	}
	startedAt := time.Now()
	lastAt := startedAt
	reportTiming := func(stage string) {
		now := time.Now()
		log.Printf("anytty cloud connect generation=%d stage=%s stage_ms=%d total_ms=%d", attemptGeneration, stage, now.Sub(lastAt).Milliseconds(), now.Sub(startedAt).Milliseconds())
		lastAt = now
	}
	sessionID := uuid.NewString()
	var prefetchedOffer *offerFuture
	if relayPreference == cloudv1.RelayPreference_RELAY_PREFERENCE_DIRECT_ONLY {
		// Direct offer gathering does not depend on Edge or Relay material. Run it beside
		// TCP/TLS/challenge setup so local interface enumeration is not a serial cold stage.
		prefetchedOffer = newOfferFuture(ctx, createOffer, &cloudv1.ClientReady{SessionId: sessionID, Generation: attemptGeneration})
		defer prefetchedOffer.Close()
		reportTiming("client_offer_started")
	}
	var connection *grpc.ClientConn
	var err error
	if capabilityRoute {
		connection, err = client.dial(resolution.locator.GetPublicEndpoint(), resolution.locator.GetServerName(), resolution.locator.GetCaCertificatePem())
	} else {
		connection, err = client.dialPinned(resolution.pairingBootstrap.GetPublicEndpoint(), resolution.pairingBootstrap.GetServerName(), resolution.pairingBootstrap.GetCaCertificateDerSha256())
	}
	if err != nil {
		return nil, markEdgeLocatorUnavailable(err)
	}
	if err := waitForEdgeTransport(ctx, connection, resolution.edgeTransportTimeout()); err != nil {
		_ = connection.Close()
		return nil, markEdgeLocatorUnavailable(err)
	}
	reportTiming("edge_transport_ready")
	closeConnection := true
	defer func() {
		if closeConnection {
			_ = connection.Close()
		}
	}()
	// route racer 会在 winner 发布后取消 attempt context；ClientGateway 需要在 answer 前响应该取消，
	// 成功后则改由 ReadyPeerSession lifecycle 持有，不能被 winner 自己的 attempt cancel 误关。
	streamContext, streamOwner := newSignalStreamOwner(ctx)
	keepStream := false
	defer func() {
		if !keepStream {
			streamOwner.Close()
		}
	}()
	stream, err := cloudv1.NewClientGatewayClient(connection).Connect(streamContext)
	if err != nil {
		return nil, markEdgeLocatorUnavailable(err)
	}
	challengeSignal, err := stream.Recv()
	if err != nil {
		return nil, markEdgeLocatorUnavailable(err)
	}
	expectedEdgeID := resolution.pairingBootstrap.GetEdgeId()
	if capabilityRoute {
		expectedEdgeID = resolution.locator.GetEdgeId()
	}
	challenge, err := validateClientGatewayChallenge(challengeSignal, expectedEdgeID, client.config.Now().UTC())
	if err != nil {
		return nil, markEdgeLocatorUnavailable(err)
	}
	reportTiming("edge_challenge")
	clientHello := &cloudv1.ClientHello{ClientPublicKey: append([]byte(nil), identity.PublicKey...), Product: product, SoftwareVersion: client.config.SoftwareVersion, AttemptGeneration: attemptGeneration, RelayPreference: relayPreference}
	if capabilityRoute {
		clientHello.Authorization = &cloudv1.ClientHello_CloudRouteGrant{CloudRouteGrant: proto.Clone(resolution.routeGrant).(*cloudv1.SignedEnvelope)}
	} else {
		clientHello.Authorization = &cloudv1.ClientHello_PairingAdmission{PairingAdmission: proto.Clone(resolution.pairingAdmission).(*cloudv1.PairingAdmission)}
	}
	hello := &cloudv1.ClientSignal{ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: uuid.NewString(), SenderId: identity.Fingerprint, BootId: client.bootID, ConnectionId: sessionID, StreamSeq: 1, SentAt: timestamppb.New(client.config.Now().UTC()), Payload: &cloudv1.ClientSignal_Hello{Hello: clientHello}}
	canonical, err := ticket.ClientHelloProofBytes(challenge, hello, client.config.Now().UTC())
	if err != nil {
		return nil, err
	}
	proof, err := signer.Sign(ctx, canonical)
	if err != nil {
		return nil, err
	}
	reportTiming("edge_proof")
	clientHello.ClientProof = proof
	if err := stream.Send(hello); err != nil {
		return nil, err
	}
	ready, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	if closedErr, closed := signalSessionCloseError(ready, challenge.GetEdgeId(), challenge.GetEdgeBootId(), sessionID); closed {
		return nil, closedErr
	}
	if ready.GetProtocolVersion() != cloudprotocol.ClientGatewayVersion || ready.GetSenderId() != challenge.GetEdgeId() || ready.GetBootId() != challenge.GetEdgeBootId() || ready.GetConnectionId() != sessionID || ready.GetStreamSeq() != 2 {
		return nil, errors.New("ClientReady is invalid")
	}
	if rejected := ready.GetRejected(); rejected != nil {
		if rejected.GetSessionId() != sessionID {
			return nil, errors.New("ClientReady rejection is invalid")
		}
		if rejected.GetEntitlementFailure() != nil {
			return nil, &EntitlementError{Failure: proto.Clone(rejected.GetEntitlementFailure()).(*cloudv1.CloudEntitlementFailure)}
		}
		if rejected.GetCode() != cloudprotocol.DaemonBlockedCode && rejected.GetCode() != cloudprotocol.DaemonDeletedCode {
			return nil, errors.New("ClientReady rejection is invalid")
		}
		return nil, &daemonLifecycleError{code: rejected.GetCode()}
	}
	if ready.GetReady() == nil || ready.GetReady().GetGeneration() != attemptGeneration {
		return nil, errors.New("ClientReady is invalid")
	}
	reportTiming("edge_client_ready")
	responseResult := make(chan edgeSignalReceiveResult, 1)
	go func() {
		response, receiveErr := stream.Recv()
		responseResult <- edgeSignalReceiveResult{signal: response, err: receiveErr}
	}()
	var offerSDP string
	if prefetchedOffer != nil {
		offerSDP, err = prefetchedOffer.Await()
	} else {
		offerSDP, err = createOffer(ctx, ready.GetReady())
	}
	if err != nil {
		return nil, fmt.Errorf("create Cloud P2P offer: %w", err)
	}
	reportTiming("client_offer_gathered")
	if strings.TrimSpace(offerSDP) == "" {
		return nil, errors.New("create Cloud P2P offer returned an empty SDP")
	}
	select {
	case result := <-responseResult:
		if result.err != nil {
			return nil, result.err
		}
		if closedErr, closed := signalSessionCloseError(result.signal, challenge.GetEdgeId(), challenge.GetEdgeBootId(), sessionID); closed {
			return nil, closedErr
		}
		return nil, errors.New("Edge signaling response arrived before ClientOffer")
	default:
	}
	offer := &cloudv1.ClientSignal{ProtocolVersion: cloudprotocol.ClientGatewayVersion, MessageId: uuid.NewString(), SenderId: identity.Fingerprint, BootId: client.bootID, ConnectionId: sessionID, StreamSeq: 2, SentAt: timestamppb.New(client.config.Now().UTC()), Payload: &cloudv1.ClientSignal_Offer{Offer: &cloudv1.ClientOffer{SessionId: sessionID, OfferSdp: offerSDP}}}
	if err := stream.Send(offer); err != nil {
		select {
		case result := <-responseResult:
			if result.err == nil {
				if closedErr, closed := signalSessionCloseError(result.signal, challenge.GetEdgeId(), challenge.GetEdgeBootId(), sessionID); closed {
					return nil, closedErr
				}
			}
		case <-time.After(100 * time.Millisecond):
		}
		return nil, err
	}
	received := <-responseResult
	if received.err != nil {
		return nil, received.err
	}
	response := received.signal
	if closedErr, closed := signalSessionCloseError(response, challenge.GetEdgeId(), challenge.GetEdgeBootId(), sessionID); closed {
		return nil, closedErr
	}
	if response.GetProtocolVersion() != cloudprotocol.ClientGatewayVersion || response.GetSenderId() != challenge.GetEdgeId() || response.GetBootId() != challenge.GetEdgeBootId() || response.GetConnectionId() != sessionID || response.GetStreamSeq() != 3 {
		return nil, errors.New("Edge signaling response is invalid")
	}
	if rejected := response.GetRejected(); rejected != nil {
		if rejected.GetEntitlementFailure() != nil {
			return nil, &EntitlementError{Failure: proto.Clone(rejected.GetEntitlementFailure()).(*cloudv1.CloudEntitlementFailure)}
		}
		return nil, fmt.Errorf("Cloud signaling rejected (%s): %s", rejected.GetCode(), rejected.GetMessage())
	}
	if response.GetAnswer() == nil || response.GetAnswer().GetSessionId() != sessionID || strings.TrimSpace(response.GetAnswer().GetAnswerSdp()) == "" {
		return nil, errors.New("Edge signaling answer is invalid")
	}
	reportTiming("edge_answer")
	if err := streamOwner.Retain(); err != nil {
		return nil, err
	}
	closeConnection = false
	keepStream = true
	session := &SignalSession{
		answer: proto.Clone(response.GetAnswer()).(*cloudv1.EdgeAnswer), connection: connection, stream: stream, cancel: streamOwner.cancel,
		senderID: identity.Fingerprint, bootID: client.bootID, edgeID: challenge.GetEdgeId(), edgeBootID: challenge.GetEdgeBootId(), sessionID: sessionID, done: make(chan struct{}),
	}
	go session.watch()
	return session, nil
}

type edgeSignalReceiveResult struct {
	signal *cloudv1.EdgeSignal
	err    error
}

func signalSessionCloseError(signal *cloudv1.EdgeSignal, edgeID, edgeBootID, sessionID string) (error, bool) {
	return signalSessionCloseErrorAtSequence(signal, edgeID, edgeBootID, sessionID, 4)
}

func signalSessionCloseErrorAtSequence(signal *cloudv1.EdgeSignal, edgeID, edgeBootID, sessionID string, expectedSequence uint64) (error, bool) {
	if signal == nil || signal.GetClosed() == nil {
		return nil, false
	}
	closed := signal.GetClosed()
	if signal.GetProtocolVersion() != cloudprotocol.ClientGatewayVersion || strings.TrimSpace(signal.GetMessageId()) == "" ||
		signal.GetSenderId() != edgeID || signal.GetBootId() != edgeBootID || signal.GetConnectionId() != sessionID ||
		signal.GetStreamSeq() != expectedSequence || signal.GetSentAt() == nil || signal.GetSentAt().CheckValid() != nil ||
		closed.GetSessionId() != sessionID || closed.GetCode() != cloudv1.SignalSessionCloseCode_SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT {
		return errors.New("Cloud signaling session close is invalid"), true
	}
	return &SignalSessionCloseError{Code: closed.GetCode(), Message: strings.TrimSpace(closed.GetMessage())}, true
}

type offerResult struct {
	sdp string
	err error
}

// offerFuture owns the cancellation and completion of one prefetched Direct offer.
// Close waits for the producer so callers can safely release callback-owned peer state.
type offerFuture struct {
	cancel context.CancelFunc
	result <-chan offerResult
	once   sync.Once
	value  offerResult
}

func newOfferFuture(ctx context.Context, createOffer func(context.Context, *cloudv1.ClientReady) (string, error), ready *cloudv1.ClientReady) *offerFuture {
	offerContext, cancel := context.WithCancel(ctx)
	result := make(chan offerResult, 1)
	go func() {
		sdp, err := createOffer(offerContext, proto.Clone(ready).(*cloudv1.ClientReady))
		result <- offerResult{sdp: sdp, err: err}
	}()
	return &offerFuture{cancel: cancel, result: result}
}

func (future *offerFuture) Await() (string, error) {
	if future == nil {
		return "", errors.New("Cloud Direct offer future is nil")
	}
	future.once.Do(func() { future.value = <-future.result })
	return future.value.sdp, future.value.err
}

func (future *offerFuture) Close() {
	if future == nil {
		return
	}
	future.cancel()
	_, _ = future.Await()
}

func validateClientGatewayChallenge(signal *cloudv1.EdgeSignal, expectedEdgeID string, now time.Time) (*cloudv1.EdgeChallenge, error) {
	if signal == nil || signal.GetProtocolVersion() != cloudprotocol.ClientGatewayVersion || signal.GetChallenge() == nil || strings.TrimSpace(signal.GetMessageId()) == "" ||
		signal.GetStreamSeq() != 1 || signal.GetSentAt() == nil || signal.GetSentAt().CheckValid() != nil {
		return nil, errors.New("ClientGateway EdgeChallenge envelope is invalid")
	}
	challenge := signal.GetChallenge()
	if err := ticket.ValidateEdgeChallenge(challenge, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY, now); err != nil {
		return nil, err
	}
	if challenge.GetEdgeId() != strings.TrimSpace(expectedEdgeID) || signal.GetSenderId() != challenge.GetEdgeId() || signal.GetBootId() != challenge.GetEdgeBootId() ||
		signal.GetConnectionId() != challenge.GetStreamId() || !proto.Equal(signal.GetSentAt(), challenge.GetIssuedAt()) {
		return nil, errors.New("ClientGateway EdgeChallenge identity is invalid")
	}
	return proto.Clone(challenge).(*cloudv1.EdgeChallenge), nil
}

func (resolution *RouteResolution) edgeTransportTimeout() time.Duration {
	if resolution != nil && resolution.cachedLocator {
		// A cached Edge is an optimization, not an authority. Fail over to the Controller
		// quickly when an old locator no longer reaches its Edge.
		return 1500 * time.Millisecond
	}
	return 3 * time.Second
}

func waitForEdgeTransport(ctx context.Context, connection *grpc.ClientConn, timeout time.Duration) error {
	if connection == nil {
		return errors.New("Edge connection is nil")
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	dialContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	connection.Connect()
	for {
		state := connection.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if state == connectivity.Shutdown {
			return errors.New("Edge transport shut down during connect")
		}
		if !connection.WaitForStateChange(dialContext, state) {
			return context.Cause(dialContext)
		}
	}
}

func (client *Client) dial(address, serverName string, caPEM []byte) (*grpc.ClientConn, error) {
	var roots *x509.CertPool
	if len(caPEM) != 0 {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("Cloud TLS CA certificate is invalid")
		}
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, ServerName: strings.TrimSpace(serverName), RootCAs: roots}
	return grpc.NewClient(
		strings.TrimSpace(address),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(signalClientKeepaliveParameters()),
	)
}

func (client *Client) dialPinned(address, serverName string, caCertificateDERFingerprint []byte) (*grpc.ClientConn, error) {
	tlsConfig, err := securetransport.NewPinnedEdgeClientTLSConfig(serverName, caCertificateDERFingerprint, client.config.Now)
	if err != nil {
		return nil, err
	}
	return grpc.NewClient(
		strings.TrimSpace(address),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(signalClientKeepaliveParameters()),
	)
}

func signalClientKeepaliveParameters() keepalive.ClientParameters {
	return keepalive.ClientParameters{
		Time:                signalTransportKeepaliveTime,
		Timeout:             signalTransportKeepaliveTimeout,
		PermitWithoutStream: false,
	}
}
