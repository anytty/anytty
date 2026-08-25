// Package cloud 把 AnyTTY Cloud 发现/信令组装成与 Direct/SSH 相同的 Go-owned ReadyPeerSession。
// Endpoint planning 和 generation 属于 client/runtime；Controller/Edge 结果不能替代最终 DataChannel remote auth。
package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/wire"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultClientName = "anytty-go-cloud"

const cloudLocatorStoreTimeout = 2 * time.Second

// PeerFactory 根据本次 Controller/Edge 决策创建 direct 或 single-Relay WebRTC primitive。
type PeerFactory interface {
	OpenCloudPeer(context.Context, port.WebRTCConfig) (port.WebRTCPeer, error)
}

// Dialer 是 managed-webrtc Route 的 Go-owned connector。
type Dialer struct {
	Peers         PeerFactory
	Cloud         *cloudclient.Client
	Authorization peeradapter.Authorizer
	Product       cloudv1.ClientProduct
	ClientName    string
	Phase         func(clientruntime.EndpointPhase)
}

// Connect 优先复用 secure credential 中的 Edge locator；只有 locator 缺失或失效才查询 Controller。
func (dialer *Dialer) Connect(ctx context.Context, request clientruntime.AttemptRequest) (clientruntime.ReadyPeerSession, error) {
	startedAt := time.Now()
	lastAt := startedAt
	reportTiming := func(phase string) {
		now := time.Now()
		log.Printf("anytty cloud connect generation=%d stage=%s stage_ms=%d total_ms=%d", request.Stamp().Generation, phase, now.Sub(lastAt).Milliseconds(), now.Sub(startedAt).Milliseconds())
		lastAt = now
	}
	if dialer == nil || dialer.Peers == nil || dialer.Cloud == nil || dialer.Authorization == nil || dialer.Product == cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED {
		return nil, errors.New("Cloud connector dependencies are incomplete")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if request.Route().Kind != endpoint.RouteManagedWebRTC {
		return nil, fmt.Errorf("route %q is not managed WebRTC", request.Route().ID)
	}
	prepared, err := dialer.Authorization.Prepare(ctx, request)
	if err != nil {
		return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureAuthorization, err)
	}
	reportTiming("authorization_prepared")
	signaling, ok := prepared.(peeradapter.PreparedSignalingAuthorization)
	if !ok || len(signaling.CloudRouteGrant()) == 0 {
		return nil, errors.New("Cloud route credential is missing its signed discovery grant")
	}
	dialer.report(clientruntime.EndpointPhaseSignaling)
	resolved, cachedErr := cloudclient.NewCachedCapabilityRoute(signaling.CloudEdgeLocator(), signaling.CloudRouteGrant())
	discovered := false
	var opened *openedCloudPeer
	if cachedErr == nil {
		opened, err = openResolvedCloudPeer(ctx, request, dialer.Peers, dialer.Cloud, resolved, signaling.ClientIdentity(), signaling, dialer.Product, dialer.report)
		if err == nil {
			reportTiming("cached_edge_ready")
		} else {
			reportTiming("cached_edge_failed")
		}
		if err != nil && !cloudclient.ShouldRefreshEdgeLocator(err) {
			return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureEdgeExchange, cloudConnectionError(err))
		}
	}
	if opened == nil {
		resolved, err = dialer.Cloud.Resolve(ctx, signaling.CloudRouteGrant(), signaling)
		if err != nil {
			return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureController, cloudConnectionError(err))
		}
		reportTiming("controller_resolved")
		opened, err = openResolvedCloudPeer(ctx, request, dialer.Peers, dialer.Cloud, resolved, signaling.ClientIdentity(), signaling, dialer.Product, dialer.report)
		if err != nil {
			return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureEdgeExchange, cloudConnectionError(err))
		}
		discovered = true
	}
	fingerprint, err := opened.RemoteCertificateFingerprint()
	if err != nil {
		_ = opened.Close()
		return nil, reportCloudFailure(request.Stamp().Generation, cloudFailurePeerFingerprint, err)
	}
	dialer.report(clientruntime.EndpointPhaseAuthorizing)
	connection := opened.Transport()
	if _, err := prepared.Authenticate(ctx, connection, fingerprint); err != nil {
		_ = opened.Close()
		return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureDataChannelAuth, fmt.Errorf("authenticate Cloud DataChannel: %w", err))
	}
	reportTiming("datachannel_authenticated")
	protocolClient := internalprotocol.NewClient(connection)
	clientName := strings.TrimSpace(dialer.ClientName)
	if clientName == "" {
		clientName = defaultClientName
	}
	if err := protocolClient.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: clientName}); err != nil {
		_ = protocolClient.Close()
		_ = opened.Close()
		return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureProtocolHello, fmt.Errorf("Cloud protocol Hello: %w", err))
	}
	reportTiming("protocol_ready")
	application, err := protocoladapter.NewApplicationClientWithObservedPath(protocolClient, request.Stamp(), string(opened.ObservedPath()))
	if err != nil {
		_ = protocolClient.Close()
		_ = opened.Close()
		return nil, err
	}
	if err := application.MarkReady(clientruntime.ReadyPeerSessionEvidence{Identity: request.DaemonIdentity(), IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version}); err != nil {
		_ = application.Close()
		_ = opened.Close()
		return nil, err
	}
	var locatorToStore []byte
	if discovered {
		locatorToStore, err = cloudclient.EncodeEdgeLocator(resolved.Locator())
		if err != nil {
			_ = application.Close()
			_ = opened.Close()
			return nil, fmt.Errorf("encode authenticated Cloud Edge locator: %w", err)
		}
	}
	dialer.report(clientruntime.EndpointPhaseReady)
	peer, signalSession := opened.Release()
	session := newSession(application, peer, signalSession)
	if len(locatorToStore) > 0 {
		// Locator is only a public optimization. Disk persistence must not delay a session that
		// already completed end-to-end authentication and protocol Hello.
		go func(locator []byte) {
			storeContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), cloudLocatorStoreTimeout)
			defer cancel()
			_ = signaling.StoreCloudEdgeLocator(storeContext, locator)
		}(append([]byte(nil), locatorToStore...))
	}
	return session, nil
}

func (dialer *Dialer) report(phase clientruntime.EndpointPhase) {
	if dialer != nil && dialer.Phase != nil {
		dialer.Phase(phase)
	}
}

func cloudConnectionError(err error) error {
	failure := cloudclient.EntitlementFailure(err)
	switch {
	case cloudclient.IsAdminDisconnect(err):
		return cloudSignalingTermination(err)
	case cloudclient.IsDaemonBlocked(err):
		return &clientruntime.Error{Code: clientruntime.ErrorDaemonBlocked, Message: "daemon Cloud access is temporarily disabled", Cause: err, Retryable: true}
	case cloudclient.IsDaemonDeleted(err):
		return &clientruntime.Error{Code: clientruntime.ErrorDaemonDeleted, Message: "daemon Cloud enrollment was deleted", Cause: err}
	case failure != nil && failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_QUOTA_EXHAUSTED:
		return &clientruntime.Error{Code: clientruntime.ErrorRelayQuotaExhausted, Message: cloudEntitlementMessage(failure), Cause: err}
	case failure != nil && failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED:
		return &clientruntime.Error{Code: clientruntime.ErrorRelayConcurrencyExhausted, Message: cloudEntitlementMessage(failure), Cause: err}
	case failure != nil && failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SERVICE_UNAVAILABLE:
		return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: cloudEntitlementMessage(failure), Cause: err, Retryable: true}
	case failure != nil && failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_NOT_IN_PLAN:
		return &clientruntime.Error{Code: clientruntime.ErrorRelayNotInPlan, Message: cloudEntitlementMessage(failure), Cause: err}
	case failure != nil && failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE:
		return &clientruntime.Error{Code: clientruntime.ErrorSubscriptionInactive, Message: cloudEntitlementMessage(failure), Cause: err}
	case failure != nil && failure.GetCode() == cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_REGION_UNAVAILABLE:
		return &clientruntime.Error{Code: clientruntime.ErrorRelayRegionUnavailable, Message: cloudEntitlementMessage(failure), Cause: err}
	case failure != nil:
		return &clientruntime.Error{Code: clientruntime.ErrorEntitlement, Message: cloudEntitlementMessage(failure), Cause: err}
	case retryableCloudRPC(status.Code(err)):
		return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: "Cloud signaling is temporarily unavailable", Cause: err, Retryable: true}
	default:
		return err
	}
}

func retryableCloudRPC(code codes.Code) bool {
	switch code {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted, codes.ResourceExhausted, codes.NotFound:
		return true
	default:
		return false
	}
}

func cloudEntitlementMessage(failure *cloudv1.CloudEntitlementFailure) string {
	if failure == nil {
		return "Cloud entitlement denied"
	}
	switch failure.GetCode() {
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED:
		return "Cloud daemon connection limit is reached; stop another Cloud daemon or upgrade the plan. Direct and SSH remain available"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_QUOTA_EXHAUSTED:
		return "Relay traffic quota is exhausted; Direct, P2P, and SSH remain available"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED:
		return "Relay concurrency is full; keep the existing connection or use Direct, P2P, or SSH"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_NOT_IN_PLAN:
		return "Relay is not included in the current AnyTTY Cloud plan"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE:
		return "AnyTTY Cloud subscription is inactive; Direct and SSH remain available"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_REGION_UNAVAILABLE:
		return "Relay is unavailable in the selected region; Direct and SSH remain available"
	default:
		return "Relay authorization is temporarily unavailable"
	}
}

// Session 把 authenticated application client 与获胜的 Cloud P2P peer 绑定到同一 generation。
type Session struct {
	*protocoladapter.ApplicationClient
	peer      port.WebRTCPeer
	signaling *cloudclient.SignalSession
	closeOnce sync.Once
	closeErr  error
	done      chan struct{}
	errMu     sync.Mutex
	err       error
}

func newSession(application *protocoladapter.ApplicationClient, peer port.WebRTCPeer, signaling *cloudclient.SignalSession) *Session {
	session := &Session{ApplicationClient: application, peer: peer, signaling: signaling, done: make(chan struct{})}
	go func() { <-application.Done(); session.finish(application.Err()) }()
	if signaling != nil {
		go func() { <-signaling.Done(); session.finish(cloudSignalingTermination(signaling.Err())) }()
	}
	return session
}

func cloudSignalingTermination(err error) error {
	if cloudclient.IsAdminDisconnect(err) {
		message := strings.TrimSpace(err.Error())
		if message == "" {
			message = "This connection was closed by an administrator"
		}
		return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: message, Cause: err}
	}
	if err != nil {
		return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: "Cloud signaling was interrupted", Cause: err, Retryable: true}
	}
	return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: "Cloud signaling ended", Retryable: true}
}

func (session *Session) Done() <-chan struct{} {
	if session == nil || session.done == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return session.done
}

func (session *Session) Err() error {
	if session == nil {
		return nil
	}
	session.errMu.Lock()
	defer session.errMu.Unlock()
	return session.err
}

func (session *Session) finish(cause error) {
	if session == nil {
		return
	}
	session.closeOnce.Do(func() {
		session.errMu.Lock()
		session.err = cause
		session.errMu.Unlock()
		if session.ApplicationClient != nil {
			session.closeErr = session.ApplicationClient.Close()
		}
		if session.peer != nil {
			if err := session.peer.Close(); session.closeErr == nil {
				session.closeErr = err
			}
		}
		if session.signaling != nil {
			if err := session.signaling.Close(); session.closeErr == nil {
				session.closeErr = err
			}
		}
		close(session.done)
	})
}

// ExecuteApplication 执行 generated Proto application command。
func (session *Session) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.ApplicationSession.Execute(ctx, command)
}

// ExecuteApplicationTerminal 为 resource-producing command 保留 terminal result。
func (session *Session) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.ApplicationSession.ExecuteTerminal(ctx, command)
}

// ConnectionSnapshot 返回 P2P selected pair 的地址与网络计数。
func (session *Session) ConnectionSnapshot(at time.Time) (clientruntime.ConnectionSnapshot, bool) {
	if session == nil || session.ApplicationClient == nil {
		return clientruntime.ConnectionSnapshot{}, false
	}
	result := clientruntime.ConnectionSnapshot{RouteID: session.Stamp().RouteID, RouteKind: endpoint.RouteManagedWebRTC, ObservedPath: session.ObservedPath(), SampledAt: at.UTC(), Connected: true}
	if snapshot, ok := session.peer.Snapshot(at); ok {
		result.ObservedPath, result.PairID = string(snapshot.Path), snapshot.PairID
		result.SampledAt, result.RoundTrip = snapshot.At, snapshot.RoundTrip
		result.LocalCandidateType, result.RemoteCandidateType = snapshot.LocalCandidateType, snapshot.RemoteCandidateType
		result.LocalAddress, result.RemoteAddress = snapshot.LocalAddress, snapshot.RemoteAddress
		result.LocalPort, result.RemotePort = snapshot.LocalPort, snapshot.RemotePort
		result.LocalRelatedAddress, result.RemoteRelatedAddress = snapshot.LocalRelatedAddress, snapshot.RemoteRelatedAddress
		result.LocalRelatedPort, result.RemoteRelatedPort = snapshot.LocalRelatedPort, snapshot.RemoteRelatedPort
		result.LocalProtocol, result.RemoteProtocol, result.RelayTransport = snapshot.LocalProtocol, snapshot.RemoteProtocol, snapshot.RelayProtocol
		result.NetworkClass, result.BytesSent, result.BytesReceived = snapshot.NetworkClass, snapshot.BytesSent, snapshot.BytesRecv
		result.PacketsSent, result.LossEvents, result.Connected = snapshot.PacketsSent, snapshot.LossEvents, snapshot.Connected
	}
	return result, true
}

// Close 幂等释放 protocol、DataChannel、ICE、DTLS 和 Pion peer。
func (session *Session) Close() error {
	if session == nil {
		return nil
	}
	session.finish(session.observedCloseCause())
	return session.closeErr
}

func (session *Session) observedCloseCause() error {
	if session == nil {
		return nil
	}
	var applicationErr error
	applicationDone := false
	if session.ApplicationClient != nil {
		select {
		case <-session.ApplicationClient.Done():
			applicationDone = true
			applicationErr = session.ApplicationClient.Err()
		default:
		}
	}
	if session.signaling != nil {
		select {
		case <-session.signaling.Done():
			signalErr := cloudSignalingTermination(session.signaling.Err())
			if cloudclient.IsAdminDisconnect(signalErr) {
				return signalErr
			}
			if applicationDone {
				return applicationErr
			}
			return signalErr
		default:
		}
	}
	if applicationDone {
		return applicationErr
	}
	return nil
}

var _ clientruntime.PeerConnector = (*Dialer)(nil)
var _ clientruntime.ApplicationReadyPeerSession = (*Session)(nil)
