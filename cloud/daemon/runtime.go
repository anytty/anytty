package daemon

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	"github.com/anytty/anytty/cloud/securetransport"
	"github.com/anytty/anytty/cloud/ticket"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	remotedaemon "github.com/anytty/anytty/remote/daemon"
	"github.com/anytty/anytty/remote/webrtc"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Config 是 daemon Cloud owner 的稳定 identity、发现记录和真实 P2P answerer 装配。
type Config struct {
	Record                EnrollmentRecord
	RecordPath            string
	Identity              remoteauth.Identity
	Answerer              webrtc.Answerer
	AccessStore           *remoteauth.AccessStore
	SoftwareVersion       string
	ControllerAddress     string
	ControllerServerName  string
	ControllerCAPEM       []byte
	RetryMinimum          time.Duration
	RetryMaximum          time.Duration
	BindingRefreshMinimum time.Duration
}

// Runtime 持有可刷新的 enrollment 路由材料和当前 AgentGateway 在线状态。
type Runtime struct {
	config            Config
	bootID            string
	attemptGeneration atomic.Uint64
	recordMu          sync.RWMutex
	record            EnrollmentRecord
	lifecycleMu       sync.Mutex
	daemonState       *cloudv1.DaemonStateRecord
	readyConnectionID string
	lifecycleAck      uint64
	cloudSessions     map[string]*cloudSession
	enrollmentDeleted bool
	operationMu       sync.Mutex
	attemptMu         sync.Mutex
	activeAttemptID   string
	activeCancel      context.CancelFunc
}

var errEdgeReselected = errors.New("daemon Edge reselection requested")

type authorizedRuntimeOptions struct {
	pionLogger           *slog.Logger
	recordPath           string
	controllerAddress    string
	controllerServerName string
	controllerCAPEM      []byte
}

type cloudSession struct {
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

// AuthorizedRuntimeOption configures process-owned dependencies for the Cloud WebRTC answerer.
type AuthorizedRuntimeOption func(*authorizedRuntimeOptions)

// WithPionLogger routes embedded Pion diagnostics through the daemon logger.
func WithPionLogger(logger *slog.Logger) AuthorizedRuntimeOption {
	return func(options *authorizedRuntimeOptions) {
		options.pionLogger = logger
	}
}

// WithEnrollmentRecordPath allows DELETED to remove only the Cloud enrollment record.
func WithEnrollmentRecordPath(path string) AuthorizedRuntimeOption {
	return func(options *authorizedRuntimeOptions) {
		options.recordPath = strings.TrimSpace(path)
	}
}

// WithControllerEndpoint configures the stable Controller used to refresh a stale Edge binding.
func WithControllerEndpoint(address, serverName string, caPEM []byte) AuthorizedRuntimeOption {
	return func(options *authorizedRuntimeOptions) {
		options.controllerAddress = strings.TrimSpace(address)
		options.controllerServerName = strings.TrimSpace(serverName)
		options.controllerCAPEM = append([]byte(nil), caPEM...)
	}
}

// NewAuthorizedRuntime 把现有 daemon Core/DeviceIdentity/AccessStore 接到真实 WebRTC Answerer。
// cmd composition root 只传 owner 和只读 session 观测回调，不需要依赖 Cloud 内部的 Pion 装配类型；
// 回调不能修改授权、PeerSession 或 Cloud generation 真值。
func NewAuthorizedRuntime(record EnrollmentRecord, identity remoteauth.Identity, accessStore *remoteauth.AccessStore, core remotedaemon.ScopedTransportServer, softwareVersion string, onSessionStart func(), onSessionError func(error), runtimeOptions ...AuthorizedRuntimeOption) (*Runtime, error) {
	if accessStore == nil {
		return nil, errors.New("daemon Cloud runtime requires AccessStore")
	}
	options := authorizedRuntimeOptions{}
	for _, apply := range runtimeOptions {
		if apply != nil {
			apply(&options)
		}
	}
	answerer := webrtc.Answerer{
		Handler:        remotedaemon.SessionAcceptor{Core: core, Identity: identity, AccessStore: accessStore},
		PionLogger:     options.pionLogger,
		OnSessionStart: onSessionStart,
		OnSessionError: onSessionError,
	}
	runtime, err := NewRuntime(Config{
		Record: record, RecordPath: options.recordPath, Identity: identity, Answerer: answerer, AccessStore: accessStore, SoftwareVersion: softwareVersion,
		ControllerAddress: options.controllerAddress, ControllerServerName: options.controllerServerName, ControllerCAPEM: options.controllerCAPEM,
	})
	if err != nil {
		return nil, err
	}
	if err := accessStore.ConfigureManagedRouteGrantIssuer(runtime.issueCloudRouteGrant); err != nil {
		return nil, err
	}
	if err := accessStore.ConfigureManagedPairingBootstrapIssuer(runtime.managedPairingBootstrap); err != nil {
		return nil, err
	}
	return runtime, nil
}

// NewRuntime 验证 Cloud owner 与 DataChannel 端到端授权 handler 已真实接线。
func NewRuntime(config Config) (*Runtime, error) {
	if err := config.Record.Validate(); err != nil {
		return nil, err
	}
	if err := config.Identity.Validate(); err != nil {
		return nil, err
	}
	config.SoftwareVersion = strings.TrimSpace(config.SoftwareVersion)
	if config.SoftwareVersion == "" || config.Answerer.Handler == nil || config.AccessStore == nil {
		return nil, errors.New("daemon Cloud runtime requires version and authorized WebRTC Answerer")
	}
	if config.RetryMinimum <= 0 {
		config.RetryMinimum = 250 * time.Millisecond
	}
	if config.RetryMaximum < config.RetryMinimum {
		config.RetryMaximum = 5 * time.Second
	}
	config.ControllerAddress = strings.TrimSpace(config.ControllerAddress)
	config.ControllerServerName = strings.TrimSpace(config.ControllerServerName)
	if (config.ControllerAddress == "") != (config.ControllerServerName == "") {
		return nil, errors.New("Controller address and server name must be configured together")
	}
	if config.BindingRefreshMinimum <= 0 {
		config.BindingRefreshMinimum = 30 * time.Second
	}
	return &Runtime{
		config: config, bootID: uuid.NewString(),
		record:        cloneEnrollmentRecord(config.Record),
		cloudSessions: make(map[string]*cloudSession),
	}, nil
}

// Run 维持 AgentGateway 长连接；Controller/Edge 失败只撤销 Presence 并有界重新解析。
func (runtime *Runtime) Run(ctx context.Context) error {
	delay := runtime.config.RetryMinimum
	var lastRefresh time.Time
	if runtime.config.ControllerAddress != "" {
		lastRefresh = time.Now()
		if err := runtime.refreshBinding(ctx); err == nil && runtime.daemonDeleted() {
			return nil
		}
	}
	for ctx.Err() == nil {
		err := runtime.connectOnce(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, errEdgeReselected) {
			delay = runtime.config.RetryMinimum
			continue
		}
		if runtime.daemonDeleted() {
			return nil
		}
		if err != nil && runtime.config.ControllerAddress != "" && (lastRefresh.IsZero() || time.Since(lastRefresh) >= runtime.config.BindingRefreshMinimum) {
			lastRefresh = time.Now()
			if refreshErr := runtime.refreshBinding(ctx); refreshErr == nil {
				if runtime.daemonDeleted() {
					return nil
				}
				delay = runtime.config.RetryMinimum
				continue
			}
			if runtime.daemonDeleted() {
				return nil
			}
		}
		if err == nil {
			delay = runtime.config.RetryMinimum
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < runtime.config.RetryMaximum {
			delay *= 2
			if delay > runtime.config.RetryMaximum {
				delay = runtime.config.RetryMaximum
			}
		}
	}
	return ctx.Err()
}

func (runtime *Runtime) connectOnce(ctx context.Context) error {
	record := runtime.currentRecord()
	binding := &cloudv1.SignedEnvelope{}
	locator := &cloudv1.EdgeLocator{}
	if proto.Unmarshal(record.DaemonBinding, binding) != nil || proto.Unmarshal(record.EdgeLocator, locator) != nil {
		return errors.New("Cloud enrollment binding or Edge locator is invalid")
	}
	return runtime.connectEdge(ctx, record.DaemonID, binding, locator)
}

func (runtime *Runtime) connectEdge(ctx context.Context, daemonID string, binding *cloudv1.SignedEnvelope, locator *cloudv1.EdgeLocator) error {
	if binding == nil || locator == nil {
		return errors.New("daemon binding or Edge locator is incomplete")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(locator.GetCaCertificatePem()) {
		return errors.New("Edge CA certificate is invalid")
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, ServerName: locator.GetServerName()}
	connection, err := grpc.NewClient(locator.GetPublicEndpoint(), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return err
	}
	defer connection.Close()
	attemptCtx, cancelAttempt := context.WithCancel(ctx)
	attemptID := uuid.NewString()
	runtime.setActiveAttempt(attemptID, cancelAttempt)
	var workers sync.WaitGroup
	var peers sync.WaitGroup
	defer func() {
		runtime.clearActiveAttempt(attemptID)
		cancelAttempt()
		workers.Wait()
		peers.Wait()
	}()
	stream, err := cloudv1.NewAgentGatewayClient(connection).Connect(attemptCtx)
	if err != nil {
		return err
	}
	challengeCommand, err := stream.Recv()
	if err != nil {
		return err
	}
	challenge, err := validateAgentGatewayChallenge(challengeCommand, locator.GetEdgeId(), time.Now().UTC())
	if err != nil {
		return err
	}
	connectionID := uuid.NewString()
	attemptGeneration := runtime.attemptGeneration.Add(1)
	hello := &cloudv1.AgentEvent{ProtocolVersion: cloudprotocol.AgentGatewayVersion, MessageId: uuid.NewString(), SenderId: daemonID, BootId: runtime.bootID, ConnectionId: connectionID, StreamSeq: 1, SentAt: timestamppb.Now(), Payload: &cloudv1.AgentEvent_Hello{Hello: &cloudv1.AgentHello{DaemonBinding: binding, SoftwareVersion: runtime.config.SoftwareVersion, AttemptGeneration: attemptGeneration}}}
	proof, err := ticket.SignAgentHelloProof(runtime.config.Identity, challenge, hello, time.Now().UTC())
	if err != nil {
		return err
	}
	hello.GetHello().DeviceProof = proof
	if err := stream.Send(hello); err != nil {
		return err
	}
	command, err := stream.Recv()
	if err != nil {
		return err
	}
	if command.GetReady() == nil || command.GetProtocolVersion() != cloudprotocol.AgentGatewayVersion || command.GetSenderId() != challenge.GetEdgeId() || command.GetBootId() != challenge.GetEdgeBootId() || command.GetConnectionId() != connectionID || command.GetStreamSeq() != 2 {
		return errors.New("AgentReady is invalid")
	}
	interval := command.GetReady().GetHeartbeat().GetInterval().AsDuration()
	daemonState := command.GetReady().GetDaemonState()
	if interval <= 0 || validateDaemonState(daemonState, daemonID) != nil {
		return errors.New("AgentReady heartbeat is invalid")
	}
	if err := runtime.applyDaemonState(attemptCtx, daemonState); err != nil {
		return err
	}
	runtime.markAgentReady(connectionID)
	defer runtime.clearAgentReady(connectionID)
	outbound := make(chan *cloudv1.AgentEvent, 32)
	writerErrors := make(chan error, 1)
	receive := make(chan error, 1)
	workers.Add(2)
	go func() {
		defer workers.Done()
		runtime.runAgentWriter(attemptCtx, stream, daemonID, runtime.bootID, connectionID, 1, outbound, writerErrors)
	}()
	go func() {
		defer workers.Done()
		runtime.runEdgeCommands(attemptCtx, stream, command.GetReady().GetGeneration(), outbound, receive, &peers)
	}()
	outbound <- lifecycleResult(command.GetReady().GetGeneration(), daemonState, nil)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case writeErr := <-writerErrors:
			return writeErr
		case recvErr := <-receive:
			if errors.Is(recvErr, io.EOF) {
				return nil
			}
			return recvErr
		case <-ticker.C:
			event := &cloudv1.AgentEvent{Payload: &cloudv1.AgentEvent_Heartbeat{Heartbeat: &cloudv1.AgentHeartbeat{Generation: command.GetReady().GetGeneration()}}}
			select {
			case outbound <- event:
			default:
				return errors.New("AgentGateway writer queue is full")
			}
		}
	}
}

func validateAgentGatewayChallenge(command *cloudv1.EdgeCommand, expectedEdgeID string, now time.Time) (*cloudv1.EdgeChallenge, error) {
	if command == nil || command.GetProtocolVersion() != cloudprotocol.AgentGatewayVersion || command.GetChallenge() == nil || strings.TrimSpace(command.GetMessageId()) == "" ||
		command.GetStreamSeq() != 1 || command.GetSentAt() == nil || command.GetSentAt().CheckValid() != nil {
		return nil, errors.New("AgentGateway EdgeChallenge envelope is invalid")
	}
	challenge := command.GetChallenge()
	if err := ticket.ValidateEdgeChallenge(challenge, cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY, now); err != nil {
		return nil, err
	}
	if challenge.GetEdgeId() != strings.TrimSpace(expectedEdgeID) || command.GetSenderId() != challenge.GetEdgeId() || command.GetBootId() != challenge.GetEdgeBootId() ||
		command.GetConnectionId() != challenge.GetStreamId() || !proto.Equal(command.GetSentAt(), challenge.GetIssuedAt()) {
		return nil, errors.New("AgentGateway EdgeChallenge identity is invalid")
	}
	return proto.Clone(challenge).(*cloudv1.EdgeChallenge), nil
}

func (runtime *Runtime) runAgentWriter(ctx context.Context, stream cloudv1.AgentGateway_ConnectClient, daemonID, bootID, connectionID string, sequence uint64, outbound <-chan *cloudv1.AgentEvent, failures chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-outbound:
			sequence++
			event.ProtocolVersion = cloudprotocol.AgentGatewayVersion
			event.MessageId = uuid.NewString()
			event.SenderId = daemonID
			event.BootId = bootID
			event.ConnectionId = connectionID
			event.StreamSeq = sequence
			event.SentAt = timestamppb.Now()
			if err := stream.Send(event); err != nil {
				select {
				case failures <- err:
				default:
				}
				return
			}
			if result := event.GetLifecycleResult(); result != nil && result.GetApplied() {
				runtime.markLifecycleAcknowledged(connectionID, result.GetDaemonState().GetStateRevision())
			}
		}
	}
}

func (runtime *Runtime) runEdgeCommands(ctx context.Context, stream cloudv1.AgentGateway_ConnectClient, generation uint64, outbound chan<- *cloudv1.AgentEvent, failures chan<- error, peers *sync.WaitGroup) {
	for {
		command, err := stream.Recv()
		if err != nil {
			failures <- err
			return
		}
		var response *cloudv1.AgentEvent
		switch {
		case command.GetAuthorize() != nil:
			authorize := command.GetAuthorize()
			if authorize.GetAgentGeneration() != generation || strings.TrimSpace(authorize.GetCorrelationId()) == "" || strings.TrimSpace(authorize.GetSessionId()) == "" || len(authorize.GetClientPublicKey()) != ed25519.PublicKeySize {
				failures <- errors.New("Edge client authorization command is invalid")
				return
			}
			allowed := runtime.allowsCloudAccess(authorize.GetClientPublicKey(), authorize.GetAccessMode(), authorize.GetPairingClaimSha256(), time.Now().UTC())
			result := &cloudv1.AgentAuthorizationResult{CorrelationId: authorize.GetCorrelationId(), SessionId: authorize.GetSessionId(), Authorized: allowed}
			if !allowed {
				result.Code, result.Message = cloudAccessRejection(authorize.GetAccessMode())
			}
			response = &cloudv1.AgentEvent{Payload: &cloudv1.AgentEvent_Authorization{Authorization: result}}
		case command.GetOffer() != nil:
			offer := command.GetOffer()
			if offer.GetAgentGeneration() != generation || strings.TrimSpace(offer.GetCorrelationId()) == "" || strings.TrimSpace(offer.GetSessionId()) == "" {
				failures <- errors.New("Edge signaling command is invalid")
				return
			}
			response = runtime.answerOffer(ctx, offer, peers)
		case command.GetLifecycle() != nil:
			lifecycle := command.GetLifecycle()
			if lifecycle.GetAgentGeneration() != generation || validateDaemonState(lifecycle.GetDaemonState(), runtime.currentRecord().DaemonID) != nil {
				failures <- errors.New("Edge daemon lifecycle command is invalid")
				return
			}
			applyErr := runtime.applyDaemonState(ctx, lifecycle.GetDaemonState())
			response = lifecycleResult(generation, lifecycle.GetDaemonState(), applyErr)
		case command.GetEdgeReselect() != nil:
			request := command.GetEdgeReselect()
			if request.GetAgentGeneration() != generation {
				failures <- errors.New("Edge reselection command generation is invalid")
				return
			}
			if _, reselectErr := runtime.reselect(ctx, false, "", 0, true); reselectErr != nil {
				failures <- reselectErr
				return
			}
			failures <- errEdgeReselected
			return
		default:
			failures <- errors.New("Edge command payload is unsupported")
			return
		}
		select {
		case <-ctx.Done():
			return
		case outbound <- response:
		default:
			failures <- errors.New("AgentGateway writer queue is full")
			return
		}
	}
}

func (runtime *Runtime) answerOffer(ctx context.Context, offer *cloudv1.AgentOffer, peers *sync.WaitGroup) *cloudv1.AgentEvent {
	reject := func(code, message string) *cloudv1.AgentEvent {
		return &cloudv1.AgentEvent{Payload: &cloudv1.AgentEvent_Rejected{Rejected: &cloudv1.AgentSignalRejected{CorrelationId: offer.GetCorrelationId(), SessionId: offer.GetSessionId(), Code: code, Message: message}}}
	}
	if !runtime.allowsCloudAccess(offer.GetClientPublicKey(), offer.GetAccessMode(), offer.GetPairingClaimSha256(), time.Now().UTC()) {
		code, message := cloudAccessRejection(offer.GetAccessMode())
		return reject(code, message)
	}
	candidates := make([]webrtc.ICECandidate, 0, len(offer.GetCandidates()))
	for _, candidate := range offer.GetCandidates() {
		if candidate != nil {
			candidates = append(candidates, webrtc.ICECandidate{Candidate: candidate.GetCandidate(), SDPMid: candidate.GetSdpMid(), SDPMLineIndex: candidate.GetSdpMlineIndex(), UsernameFragment: candidate.GetUsernameFragment()})
		}
	}
	iceServers := make([]webrtc.ICEServer, 0, 1)
	if relay := offer.GetRelay(); relay != nil {
		if len(relay.GetUrls()) == 0 || strings.TrimSpace(relay.GetUsername()) == "" || strings.TrimSpace(relay.GetCredential()) == "" {
			return reject("RELAY_INVALID", "Edge supplied incomplete Relay ICE material")
		}
		iceServers = append(iceServers, webrtc.ICEServer{URLs: append([]string(nil), relay.GetUrls()...), Username: relay.GetUsername(), Credential: relay.GetCredential()})
	}
	sessionCtx, session, ok := runtime.beginCloudSession(ctx, offer.GetSessionId(), peers)
	if !ok {
		return reject("DAEMON_UNAVAILABLE", "daemon Cloud access is not active")
	}
	answerer := runtime.config.Answerer
	onPeerClosed := answerer.OnPeerClosed
	var peerClosed sync.Once
	answerer.OnPeerClosed = func() {
		peerClosed.Do(func() {
			runtime.finishCloudSession(offer.GetSessionId(), session, peers, onPeerClosed)
		})
	}
	answer, err := answerer.Answer(sessionCtx, &webrtc.SignalingOffer{SessionID: offer.GetSessionId(), SDP: offer.GetOfferSdp(), Candidates: candidates}, iceServers)
	if err != nil {
		peerClosed.Do(func() { runtime.finishCloudSession(offer.GetSessionId(), session, peers, onPeerClosed) })
		return reject("ANSWER_FAILED", "daemon could not establish P2P signaling")
	}
	wireCandidates := make([]*cloudv1.CloudICECandidate, 0, len(answer.Candidates))
	for _, candidate := range answer.Candidates {
		wireCandidates = append(wireCandidates, &cloudv1.CloudICECandidate{Candidate: candidate.Candidate, SdpMid: candidate.SDPMid, SdpMlineIndex: candidate.SDPMLineIndex, UsernameFragment: candidate.UsernameFragment})
	}
	return &cloudv1.AgentEvent{Payload: &cloudv1.AgentEvent_Answer{Answer: &cloudv1.AgentAnswer{CorrelationId: offer.GetCorrelationId(), SessionId: offer.GetSessionId(), AnswerSdp: answer.SDP, Candidates: wireCandidates}}}
}

func validateDaemonState(state *cloudv1.DaemonStateRecord, daemonID string) error {
	if state == nil || state.GetDaemonId() != strings.TrimSpace(daemonID) || state.GetStateRevision() == 0 {
		return errors.New("daemon lifecycle state is invalid")
	}
	switch state.GetState() {
	case cloudv1.DaemonState_DAEMON_STATE_ACTIVE, cloudv1.DaemonState_DAEMON_STATE_BLOCKED, cloudv1.DaemonState_DAEMON_STATE_DELETED:
		return nil
	default:
		return errors.New("daemon lifecycle state is invalid")
	}
}

func lifecycleResult(generation uint64, state *cloudv1.DaemonStateRecord, applyErr error) *cloudv1.AgentEvent {
	result := &cloudv1.DaemonLifecycleResult{DaemonState: proto.Clone(state).(*cloudv1.DaemonStateRecord), AgentGeneration: generation, Applied: applyErr == nil}
	if applyErr != nil {
		result.ErrorMessage = applyErr.Error()
	}
	return &cloudv1.AgentEvent{Payload: &cloudv1.AgentEvent_LifecycleResult{LifecycleResult: result}}
}

func (runtime *Runtime) applyDaemonState(ctx context.Context, state *cloudv1.DaemonStateRecord) error {
	if err := validateDaemonState(state, runtime.currentRecord().DaemonID); err != nil {
		return err
	}
	clone := proto.Clone(state).(*cloudv1.DaemonStateRecord)
	runtime.lifecycleMu.Lock()
	current := runtime.daemonState
	if current != nil && clone.GetStateRevision() < current.GetStateRevision() {
		runtime.lifecycleMu.Unlock()
		return errors.New("daemon lifecycle state is stale")
	}
	if current != nil && clone.GetStateRevision() == current.GetStateRevision() && current.GetStateRevision() != 0 && !proto.Equal(current, clone) {
		runtime.lifecycleMu.Unlock()
		return errors.New("daemon lifecycle revision conflicts with current state")
	}
	if current != nil && current.GetState() == cloudv1.DaemonState_DAEMON_STATE_DELETED && !proto.Equal(current, clone) {
		runtime.lifecycleMu.Unlock()
		return errors.New("deleted daemon lifecycle state is terminal")
	}
	if current == nil || current.GetStateRevision() != clone.GetStateRevision() {
		runtime.lifecycleAck = 0
	}
	runtime.daemonState = clone
	sessions := make([]*cloudSession, 0, len(runtime.cloudSessions))
	if clone.GetState() != cloudv1.DaemonState_DAEMON_STATE_ACTIVE {
		for _, session := range runtime.cloudSessions {
			sessions = append(sessions, session)
		}
	}
	runtime.lifecycleMu.Unlock()

	for _, session := range sessions {
		session.cancel()
	}
	for _, session := range sessions {
		select {
		case <-session.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if clone.GetState() == cloudv1.DaemonState_DAEMON_STATE_DELETED {
		if runtime.config.AccessStore == nil {
			return errors.New("client access store is unavailable")
		}
		if err := runtime.config.AccessStore.DisableManagedCloudRoute(); err != nil {
			return err
		}
		if runtime.config.RecordPath == "" {
			return errors.New("Cloud enrollment record path is unavailable")
		}
		if err := DeleteRecord(runtime.config.RecordPath); err != nil {
			return err
		}
		runtime.lifecycleMu.Lock()
		runtime.enrollmentDeleted = true
		runtime.lifecycleMu.Unlock()
	}
	return nil
}

func (runtime *Runtime) beginCloudSession(parent context.Context, sessionID string, peers *sync.WaitGroup) (context.Context, *cloudSession, bool) {
	runtime.lifecycleMu.Lock()
	defer runtime.lifecycleMu.Unlock()
	if !runtime.cloudActiveLocked() || runtime.cloudSessions[sessionID] != nil {
		return nil, nil, false
	}
	ctx, cancel := context.WithCancel(parent)
	session := &cloudSession{cancel: cancel, done: make(chan struct{})}
	peers.Add(1)
	runtime.cloudSessions[sessionID] = session
	return ctx, session, true
}

func (runtime *Runtime) finishCloudSession(sessionID string, session *cloudSession, peers *sync.WaitGroup, onPeerClosed func()) {
	session.once.Do(func() {
		session.cancel()
		runtime.lifecycleMu.Lock()
		if runtime.cloudSessions[sessionID] == session {
			delete(runtime.cloudSessions, sessionID)
		}
		runtime.lifecycleMu.Unlock()
		close(session.done)
		peers.Done()
		if onPeerClosed != nil {
			onPeerClosed()
		}
	})
}

func (runtime *Runtime) cloudActive() bool {
	runtime.lifecycleMu.Lock()
	defer runtime.lifecycleMu.Unlock()
	return runtime.cloudActiveLocked()
}

func (runtime *Runtime) cloudActiveLocked() bool {
	return runtime.readyConnectionID != "" && runtime.daemonState != nil &&
		runtime.daemonState.GetState() == cloudv1.DaemonState_DAEMON_STATE_ACTIVE &&
		runtime.lifecycleAck == runtime.daemonState.GetStateRevision()
}

func (runtime *Runtime) markAgentReady(connectionID string) {
	runtime.lifecycleMu.Lock()
	runtime.readyConnectionID = connectionID
	runtime.lifecycleAck = 0
	runtime.lifecycleMu.Unlock()
}

func (runtime *Runtime) clearAgentReady(connectionID string) {
	runtime.lifecycleMu.Lock()
	if runtime.readyConnectionID == connectionID {
		runtime.readyConnectionID = ""
		runtime.lifecycleAck = 0
	}
	runtime.lifecycleMu.Unlock()
}

func (runtime *Runtime) markLifecycleAcknowledged(connectionID string, revision uint64) {
	runtime.lifecycleMu.Lock()
	if runtime.readyConnectionID == connectionID && runtime.daemonState != nil && runtime.daemonState.GetStateRevision() == revision {
		runtime.lifecycleAck = revision
	}
	runtime.lifecycleMu.Unlock()
}

func (runtime *Runtime) daemonDeleted() bool {
	runtime.lifecycleMu.Lock()
	defer runtime.lifecycleMu.Unlock()
	return runtime.daemonState != nil && runtime.daemonState.GetState() == cloudv1.DaemonState_DAEMON_STATE_DELETED && runtime.enrollmentDeleted
}

func (runtime *Runtime) allowsCloudAccess(clientPublicKey []byte, mode cloudv1.CloudClientAccessMode, pairingClaimDigest []byte, now time.Time) bool {
	if runtime == nil || !runtime.cloudActive() || runtime.config.AccessStore == nil || len(clientPublicKey) != ed25519.PublicKeySize {
		return false
	}
	publicKey := ed25519.PublicKey(clientPublicKey)
	switch mode {
	case cloudv1.CloudClientAccessMode_CLOUD_CLIENT_ACCESS_MODE_CAPABILITY:
		return len(pairingClaimDigest) == 0 && runtime.config.AccessStore.AllowsClientPublicKey(publicKey, now)
	case cloudv1.CloudClientAccessMode_CLOUD_CLIENT_ACCESS_MODE_PAIRING:
		return runtime.config.AccessStore.AllowsPairingClaimDigest(pairingClaimDigest, publicKey, now)
	default:
		return false
	}
}

func cloudAccessRejection(mode cloudv1.CloudClientAccessMode) (string, string) {
	if mode == cloudv1.CloudClientAccessMode_CLOUD_CLIENT_ACCESS_MODE_CAPABILITY {
		return "CLIENT_REVOKED", "client access is not active"
	}
	return "PAIRING_CLAIM_INVALID", "pairing claim is not active"
}

func cloneEnrollmentRecord(record EnrollmentRecord) EnrollmentRecord {
	record.DaemonBinding = append([]byte(nil), record.DaemonBinding...)
	record.EdgeLocator = append([]byte(nil), record.EdgeLocator...)
	return record
}

func (runtime *Runtime) currentRecord() EnrollmentRecord {
	runtime.recordMu.RLock()
	defer runtime.recordMu.RUnlock()
	return cloneEnrollmentRecord(runtime.record)
}

func (runtime *Runtime) replaceRecord(record EnrollmentRecord) {
	runtime.recordMu.Lock()
	runtime.record = cloneEnrollmentRecord(record)
	runtime.recordMu.Unlock()
}

func (runtime *Runtime) activeRecord() (EnrollmentRecord, error) {
	runtime.lifecycleMu.Lock()
	defer runtime.lifecycleMu.Unlock()
	if !runtime.cloudActiveLocked() {
		return EnrollmentRecord{}, errors.New("Cloud pairing requires an active Edge connection")
	}
	return runtime.currentRecord(), nil
}

func (runtime *Runtime) issueCloudRouteGrant(clientPublicKey ed25519.PublicKey, product uint32, issuedAt, expiresAt time.Time) ([]byte, []byte, error) {
	record, err := runtime.activeRecord()
	if err != nil {
		return nil, nil, err
	}
	clientProduct := cloudv1.ClientProduct(product)
	if clientProduct == cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED || clientProduct > cloudv1.ClientProduct_CLIENT_PRODUCT_DESKTOP_GUI {
		return nil, nil, errors.New("CloudRouteGrant client product is invalid")
	}
	claims := &cloudv1.CloudRouteGrantClaims{GrantId: uuid.NewString(), DaemonId: record.DaemonID, ClientPublicKey: append([]byte(nil), clientPublicKey...), Product: clientProduct, IssuedAt: timestamppb.New(issuedAt.UTC()), ExpiresAt: timestamppb.New(expiresAt.UTC())}
	envelope, err := ticket.SignCloudRouteGrant(runtime.config.Identity, claims)
	if err != nil {
		return nil, nil, err
	}
	grant, err := proto.MarshalOptions{Deterministic: true}.Marshal(envelope)
	return grant, record.EdgeLocator, err
}

func (runtime *Runtime) managedPairingBootstrap() (*remoteauthpb.PairingManagedRouteSeed, error) {
	record, err := runtime.activeRecord()
	if err != nil {
		return nil, err
	}
	locator := &cloudv1.EdgeLocator{}
	if err := proto.Unmarshal(record.EdgeLocator, locator); err != nil {
		return nil, errors.New("daemon Cloud runtime Edge locator is invalid")
	}
	caFingerprint, err := securetransport.EdgeCACertificateDERFingerprint(locator.GetCaCertificatePem())
	if err != nil {
		return nil, err
	}
	return &remoteauthpb.PairingManagedRouteSeed{
		DaemonId: record.DaemonID, EdgeId: locator.GetEdgeId(), PublicEndpoint: locator.GetPublicEndpoint(), ServerName: locator.GetServerName(),
		CaCertificateDerSha256: caFingerprint,
	}, nil
}

func (runtime *Runtime) refreshBinding(ctx context.Context) error {
	_, err := runtime.refreshBindingRequest(ctx, nil, false, "", 0, true)
	return err
}

func (runtime *Runtime) refreshBindingRequest(ctx context.Context, measurements []*cloudv1.DaemonEdgeMeasurement, changePreference bool, preferredEdgeID string, expectedPreferenceRevision uint64, persistBinding bool) (*cloudv1.DaemonEdgeSelection, error) {
	record := runtime.currentRecord()
	var roots *x509.CertPool
	if len(runtime.config.ControllerCAPEM) != 0 {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM(runtime.config.ControllerCAPEM) {
			return nil, errors.New("Cloud Controller CA certificate is invalid")
		}
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, ServerName: runtime.config.ControllerServerName, RootCAs: roots}
	connection, err := grpc.NewClient(runtime.config.ControllerAddress, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	client := cloudv1.NewEnrollmentServiceClient(connection)
	challenge, err := client.BeginDaemonBindingRefresh(ctx, &cloudv1.BeginDaemonBindingRefreshRequest{DaemonId: record.DaemonID})
	if err != nil {
		return nil, err
	}
	proof, err := remoteauth.SignDeviceIdentityProof(runtime.config.Identity, challenge.GetChallenge())
	if err != nil {
		return nil, err
	}
	completed, err := client.CompleteDaemonBindingRefresh(ctx, &cloudv1.CompleteDaemonBindingRefreshRequest{ChallengeId: challenge.GetChallengeId(), DeviceProof: proof, EdgeMeasurements: measurements, ChangePreference: changePreference, PreferredEdgeId: strings.TrimSpace(preferredEdgeID), ExpectedPreferenceRevision: expectedPreferenceRevision})
	if err != nil {
		return nil, err
	}
	daemon := completed.GetDaemon()
	if daemon == nil || daemon.GetDaemonId() != record.DaemonID || daemon.GetAccountId() != record.AccountID ||
		daemon.GetDeviceId() != runtime.config.Identity.DeviceID || daemon.GetDeviceFingerprint() != runtime.config.Identity.Fingerprint {
		return nil, errors.New("daemon binding refresh identity is invalid")
	}
	state := &cloudv1.DaemonStateRecord{DaemonId: daemon.GetDaemonId(), State: daemon.GetState(), StateRevision: daemon.GetStateRevision()}
	if daemon.GetState() == cloudv1.DaemonState_DAEMON_STATE_DELETED {
		if completed.GetDaemonBinding() != nil || completed.GetEdgeLocator() != nil {
			return nil, errors.New("deleted daemon binding refresh returned route material")
		}
		return completed.GetEdgeSelection(), runtime.applyDaemonState(ctx, state)
	}
	if completed.GetDaemonBinding() == nil || completed.GetEdgeLocator() == nil {
		return nil, errors.New("daemon binding refresh response is incomplete")
	}
	bindingClaims := &cloudv1.DaemonBindingClaims{}
	if err := proto.Unmarshal(completed.GetDaemonBinding().GetPayload(), bindingClaims); err != nil ||
		bindingClaims.GetDeviceId() != runtime.config.Identity.DeviceID || !ed25519.PublicKey(bindingClaims.GetDevicePublicKey()).Equal(runtime.config.Identity.PublicKey) {
		return nil, errors.New("daemon binding refresh identity is invalid")
	}
	if !persistBinding {
		return completed.GetEdgeSelection(), runtime.applyDaemonState(ctx, state)
	}
	binding, err := proto.MarshalOptions{Deterministic: true}.Marshal(completed.GetDaemonBinding())
	if err != nil {
		return nil, err
	}
	locatorPayload, err := proto.MarshalOptions{Deterministic: true}.Marshal(completed.GetEdgeLocator())
	if err != nil {
		return nil, err
	}
	updated := EnrollmentRecord{Version: recordVersion, DaemonID: record.DaemonID, AccountID: record.AccountID, DaemonBinding: binding, EdgeLocator: locatorPayload, EnrolledAt: record.EnrolledAt}
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if runtime.config.RecordPath == "" {
		return nil, errors.New("Cloud enrollment record path is unavailable")
	}
	if err := SaveRecord(runtime.config.RecordPath, updated); err != nil {
		return nil, err
	}
	runtime.replaceRecord(updated)
	return completed.GetEdgeSelection(), runtime.applyDaemonState(ctx, state)
}

// EdgeSelection probes every currently advertised Edge and returns the Controller's final ranking.
func (runtime *Runtime) EdgeSelection(ctx context.Context) (*cloudv1.DaemonEdgeSelection, error) {
	return runtime.reselect(ctx, false, "", 0, false)
}

// PreferEdge persists a soft preference, probes candidates, and reconnects the Cloud control stream.
func (runtime *Runtime) PreferEdge(ctx context.Context, edgeID string, expectedRevision uint64) (*cloudv1.DaemonEdgeSelection, error) {
	if expectedRevision == 0 {
		return nil, errors.New("Edge preference revision is required")
	}
	return runtime.reselect(ctx, true, strings.TrimSpace(edgeID), expectedRevision, true)
}

// ReselectEdge keeps the current preference and immediately probes/reconnects without restarting the daemon.
func (runtime *Runtime) ReselectEdge(ctx context.Context) (*cloudv1.DaemonEdgeSelection, error) {
	return runtime.reselect(ctx, false, "", 0, true)
}

func (runtime *Runtime) reselect(ctx context.Context, changePreference bool, preferredEdgeID string, expectedRevision uint64, reconnect bool) (*cloudv1.DaemonEdgeSelection, error) {
	runtime.operationMu.Lock()
	defer runtime.operationMu.Unlock()
	selection, err := runtime.refreshBindingRequest(ctx, nil, changePreference, preferredEdgeID, expectedRevision, false)
	if err != nil {
		return nil, err
	}
	measurements := runtime.probeEdges(ctx, selection.GetCandidates())
	selection, err = runtime.refreshBindingRequest(ctx, measurements, false, "", 0, reconnect)
	if err != nil {
		return nil, err
	}
	if reconnect {
		runtime.interruptActiveAttempt()
	}
	return proto.Clone(selection).(*cloudv1.DaemonEdgeSelection), nil
}

func (runtime *Runtime) probeEdges(ctx context.Context, candidates []*cloudv1.DaemonEdgeCandidate) []*cloudv1.DaemonEdgeMeasurement {
	result := make([]*cloudv1.DaemonEdgeMeasurement, len(candidates))
	var workers sync.WaitGroup
	limit := make(chan struct{}, 4)
	for index, candidate := range candidates {
		index, locator := index, candidate.GetLocator()
		if locator == nil {
			continue
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			select {
			case <-ctx.Done():
				return
			case limit <- struct{}{}:
			}
			defer func() { <-limit }()
			result[index] = probeEdge(ctx, locator)
		}()
	}
	workers.Wait()
	compact := make([]*cloudv1.DaemonEdgeMeasurement, 0, len(result))
	for _, value := range result {
		if value != nil {
			compact = append(compact, value)
		}
	}
	return compact
}

func probeEdge(parent context.Context, locator *cloudv1.EdgeLocator) *cloudv1.DaemonEdgeMeasurement {
	const samples = 3
	var success int
	var latency time.Duration
	for range samples {
		started := time.Now()
		probeCtx, cancel := context.WithTimeout(parent, 2*time.Second)
		roots := x509.NewCertPool()
		validCA := roots.AppendCertsFromPEM(locator.GetCaCertificatePem())
		if validCA {
			connection, err := grpc.NewClient(locator.GetPublicEndpoint(), grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, ServerName: locator.GetServerName()})))
			if err == nil {
				_, err = grpc_health_v1.NewHealthClient(connection).Check(probeCtx, &grpc_health_v1.HealthCheckRequest{}, grpc.WaitForReady(true))
				_ = connection.Close()
			}
			if err == nil {
				success++
				latency += time.Since(started)
			}
		}
		cancel()
		if parent.Err() != nil {
			break
		}
	}
	latencyMS := uint32(0)
	if success > 0 {
		latencyMS = uint32(math.Round(float64(latency/time.Duration(success)) / float64(time.Millisecond)))
	}
	return &cloudv1.DaemonEdgeMeasurement{EdgeId: locator.GetEdgeId(), Reachable: success > 0, ConnectLatencyMs: latencyMS, ConnectionFailureRate: float64(samples-success) / samples, SampleCount: samples, MeasuredAt: timestamppb.Now()}
}

func (runtime *Runtime) setActiveAttempt(id string, cancel context.CancelFunc) {
	runtime.attemptMu.Lock()
	runtime.activeAttemptID, runtime.activeCancel = id, cancel
	runtime.attemptMu.Unlock()
}

func (runtime *Runtime) clearActiveAttempt(id string) {
	runtime.attemptMu.Lock()
	if runtime.activeAttemptID == id {
		runtime.activeAttemptID, runtime.activeCancel = "", nil
	}
	runtime.attemptMu.Unlock()
}

func (runtime *Runtime) interruptActiveAttempt() {
	runtime.attemptMu.Lock()
	cancel := runtime.activeCancel
	runtime.attemptMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Enroll 使用一次性 code 和现有 DeviceIdentity 完成 challenge，并返回可持久化最小记录。
func Enroll(ctx context.Context, controllerAddress, controllerServerName, code string, tlsConfig *tls.Config, identity remoteauth.Identity) (EnrollmentRecord, error) {
	if err := identity.Validate(); err != nil {
		return EnrollmentRecord{}, err
	}
	if tlsConfig == nil {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS13, ServerName: controllerServerName}
	} else {
		tlsConfig = tlsConfig.Clone()
	}
	connection, err := grpc.NewClient(strings.TrimSpace(controllerAddress), grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return EnrollmentRecord{}, err
	}
	defer connection.Close()
	client := cloudv1.NewEnrollmentServiceClient(connection)
	challenge, err := client.BeginDaemonEnrollment(ctx, &cloudv1.BeginDaemonEnrollmentRequest{EnrollmentCode: strings.TrimSpace(code), DeviceId: identity.DeviceID, DeviceFingerprint: identity.Fingerprint, DevicePublicKey: append([]byte(nil), identity.PublicKey...)})
	if err != nil {
		return EnrollmentRecord{}, err
	}
	proof, err := remoteauth.SignDeviceIdentityProof(identity, challenge.GetChallenge())
	if err != nil {
		return EnrollmentRecord{}, err
	}
	completed, err := client.CompleteDaemonEnrollment(ctx, &cloudv1.CompleteDaemonEnrollmentRequest{ChallengeId: challenge.GetChallengeId(), DeviceProof: proof})
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if completed.GetDaemon() == nil || completed.GetDaemonBinding() == nil || completed.GetEdgeLocator() == nil {
		return EnrollmentRecord{}, errors.New("daemon enrollment response is incomplete")
	}
	binding, err := proto.MarshalOptions{Deterministic: true}.Marshal(completed.GetDaemonBinding())
	if err != nil {
		return EnrollmentRecord{}, err
	}
	locator, err := proto.MarshalOptions{Deterministic: true}.Marshal(completed.GetEdgeLocator())
	if err != nil {
		return EnrollmentRecord{}, err
	}
	return EnrollmentRecord{
		Version: recordVersion, DaemonID: completed.GetDaemon().GetDaemonId(), AccountID: completed.GetDaemon().GetAccountId(), DaemonBinding: binding, EdgeLocator: locator, EnrolledAt: time.Now().UTC(),
		DaemonCount: completed.GetDaemonCount(), DaemonLimit: completed.GetDaemonLimit(),
	}, nil
}

// EnrollLocal 加载 daemon 已有 DeviceIdentity、完成注册并原子保存最小 Cloud record。
func EnrollLocal(ctx context.Context, controllerAddress, controllerServerName, code, identityDirectory, recordPath string) (EnrollmentRecord, error) {
	identity, err := remoteauth.LoadOrCreateLocalIdentity(identityDirectory)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	record, err := Enroll(ctx, controllerAddress, controllerServerName, code, &tls.Config{MinVersion: tls.VersionTLS13, ServerName: controllerServerName}, identity)
	if err != nil {
		return EnrollmentRecord{}, err
	}
	if err := SaveRecord(recordPath, record); err != nil {
		return EnrollmentRecord{}, err
	}
	return record, nil
}
