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
const cloudSessionReleaseTimeout = 5 * time.Second
const cloudApplicationProbeTimeout = 4 * time.Second
const cloudApplicationProbeRounds = 2

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
	application, failureStage, err := dialer.prepareConfirmedCloudApplication(ctx, request, prepared, opened, reportTiming)
	if err != nil {
		fallback, retry := cloudApplicationFallback(ctx, request.Route(), opened, failureStage, err)
		if !retry {
			return nil, reportCloudFailure(request.Stamp().Generation, failureStage, errors.Join(err, abandonOpenedCloudPeer(ctx, opened)))
		}
		releaseContext, cancelRelease := context.WithTimeout(ctx, cloudPeerCleanupTimeout)
		releaseErr := opened.AbandonAndWait(releaseContext)
		cancelRelease()
		if releaseErr != nil {
			if ctx.Err() != nil {
				return nil, reportCloudFailure(request.Stamp().Generation, failureStage, context.Cause(ctx))
			}
			return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureEdgeExchange, fmt.Errorf("release provisional Cloud path before fallback: %w", releaseErr))
		}
		reportTiming("provisional_path_abandoned")
		log.Printf("anytty cloud connect generation=%d stage=relay_tcp_fallback", request.Stamp().Generation)
		opened, err = openResolvedCloudPeerAttempt(ctx, request, dialer.Peers, dialer.Cloud, resolved, signaling.ClientIdentity(), signaling, dialer.Product, dialer.report, fallback)
		if err != nil {
			return nil, reportCloudFailure(request.Stamp().Generation, cloudFailureEdgeExchange, cloudConnectionError(err))
		}
		application, failureStage, err = dialer.prepareConfirmedCloudApplication(ctx, request, prepared, opened, reportTiming)
		if err != nil {
			return nil, reportCloudFailure(request.Stamp().Generation, failureStage, errors.Join(err, abandonOpenedCloudPeer(ctx, opened)))
		}
	}
	if err := application.MarkReady(clientruntime.ReadyPeerSessionEvidence{Identity: request.DaemonIdentity(), IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version}); err != nil {
		return nil, errors.Join(err, application.Close(), releaseConfirmedCloudPeer(opened))
	}
	var locatorToStore []byte
	if discovered {
		locatorToStore, err = cloudclient.EncodeEdgeLocator(resolved.Locator())
		if err != nil {
			return nil, errors.Join(fmt.Errorf("encode authenticated Cloud Edge locator: %w", err), application.Close(), releaseConfirmedCloudPeer(opened))
		}
	}
	dialer.report(clientruntime.EndpointPhaseReady)
	confirmedPath := opened.path
	peer, signalSession := opened.Release()
	session := newSession(application, peer, signalSession, confirmedPath)
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

type confirmedCloudPeerReleaser interface {
	ReleaseAndWait(context.Context) error
}

func releaseConfirmedCloudPeer(opened confirmedCloudPeerReleaser) error {
	if opened == nil {
		return nil
	}
	releaseContext, cancelRelease := context.WithTimeout(context.Background(), cloudSessionReleaseTimeout)
	defer cancelRelease()
	return opened.ReleaseAndWait(releaseContext)
}

func (dialer *Dialer) prepareConfirmedCloudApplication(
	ctx context.Context,
	request clientruntime.AttemptRequest,
	prepared peeradapter.PreparedAuthorization,
	opened *openedCloudPeer,
	reportTiming func(string),
) (*protocoladapter.ApplicationClient, cloudFailureStage, error) {
	if opened == nil {
		return nil, cloudFailureEdgeExchange, errors.New("Cloud peer is unavailable")
	}
	setupContext, stopSetup := cloudPeerSetupContext(ctx, opened.signaling)
	defer stopSetup()
	application, failureStage, err := dialer.prepareCloudApplication(setupContext, request, prepared, opened, reportTiming)
	if err != nil {
		if signalErr, ended := cloudSignalTerminalError(opened.signaling); ended {
			return nil, cloudFailureEdgeExchange, cloudSignalingTermination(signalErr)
		}
		return nil, failureStage, err
	}
	initialPath := opened.path
	confirmedPath, err := opened.ConfirmAuthenticatedPath(setupContext)
	if err != nil {
		_ = application.Close()
		if signalErr, ended := cloudSignalTerminalError(opened.signaling); ended {
			return nil, cloudFailureEdgeExchange, cloudSignalingTermination(signalErr)
		}
		return nil, cloudFailureEdgeExchange, cloudConnectionError(err)
	}
	if initialPath != confirmedPath {
		log.Printf("anytty cloud connect generation=%d stage=path_reselected initial=%s confirmed=%s", request.Stamp().Generation, initialPath, confirmedPath)
	}
	if snapshot, ok := opened.peer.Snapshot(time.Now()); ok {
		log.Printf("anytty cloud connect generation=%d stage=path_confirmation_evidence path=%s local_candidate=%s remote_candidate=%s", request.Stamp().Generation, snapshot.Path, snapshot.LocalCandidateType, snapshot.RemoteCandidateType)
	}
	reportTiming("path_confirmed")
	return application, "", nil
}

func (dialer *Dialer) prepareCloudApplication(
	ctx context.Context,
	request clientruntime.AttemptRequest,
	prepared peeradapter.PreparedAuthorization,
	opened *openedCloudPeer,
	reportTiming func(string),
) (*protocoladapter.ApplicationClient, cloudFailureStage, error) {
	fingerprint, err := opened.RemoteCertificateFingerprint()
	if err != nil {
		return nil, cloudFailurePeerFingerprint, err
	}
	dialer.report(clientruntime.EndpointPhaseAuthorizing)
	connection := opened.Transport()
	if _, err = prepared.Authenticate(ctx, connection, fingerprint); err != nil {
		return nil, cloudFailureDataChannelAuth, fmt.Errorf("authenticate Cloud DataChannel: %w", err)
	}
	reportTiming("datachannel_authenticated")
	protocolClient := internalprotocol.NewClient(connection)
	clientName := strings.TrimSpace(dialer.ClientName)
	if clientName == "" {
		clientName = defaultClientName
	}
	if err := protocolClient.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: clientName}); err != nil {
		_ = protocolClient.Close()
		return nil, cloudFailureProtocolHello, fmt.Errorf("Cloud protocol Hello: %w", err)
	}
	reportTiming("protocol_ready")
	application, err := protocoladapter.NewApplicationClientWithObservedPath(protocolClient, request.Stamp(), string(opened.ObservedPath()))
	if err != nil {
		_ = protocolClient.Close()
		return nil, cloudFailureApplicationProbe, err
	}
	probeContext, cancelProbe := context.WithTimeout(ctx, cloudApplicationProbeTimeout)
	err = probeCloudApplication(probeContext, application)
	cancelProbe()
	if err != nil {
		_ = application.Close()
		return nil, cloudFailureApplicationProbe, fmt.Errorf("probe Cloud application session: %w", err)
	}
	reportTiming("application_ready")
	return application, "", nil
}

func cloudApplicationFallback(
	ctx context.Context,
	route endpoint.AccessRoute,
	opened *openedCloudPeer,
	stage cloudFailureStage,
	err error,
) (cloudPeerAttempt, bool) {
	if opened == nil || ctx == nil || ctx.Err() != nil {
		return cloudPeerAttempt{}, false
	}
	if _, ended := cloudSignalTerminalError(opened.signaling); ended {
		return cloudPeerAttempt{}, false
	}
	if stage == cloudFailureDataChannelAuth {
		fallback, ok := planCloudPeerTransportFallback(route, err)
		return fallback, ok && opened.relayTCPAvailable
	}
	if stage != cloudFailureProtocolHello && stage != cloudFailureApplicationProbe {
		return cloudPeerAttempt{}, false
	}
	return planCloudPeerPostAuthFallback(ctx, route, opened.ObservedPath(), opened.relayTCPAvailable, err)
}

type cloudApplicationProber interface {
	TerminalDefaults(context.Context, *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error)
}

func probeCloudApplication(ctx context.Context, application cloudApplicationProber) error {
	if application == nil {
		return errors.New("Cloud application session is unavailable")
	}
	for round := 1; round <= cloudApplicationProbeRounds; round++ {
		result, err := application.TerminalDefaults(ctx, &apipb.TerminalDefaultsCommand{})
		if err != nil {
			return fmt.Errorf("Cloud application probe round %d: %w", round, err)
		}
		if result == nil {
			return fmt.Errorf("Cloud application probe round %d returned no terminal defaults", round)
		}
	}
	return nil
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
		return &clientruntime.Error{Code: clientruntime.ErrorRelayConcurrencyExhausted, Message: cloudEntitlementMessage(failure), Cause: err, Retryable: true}
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
	path      endpoint.Path
	closeOnce sync.Once
	closeErr  error
	done      chan struct{}
	errMu     sync.Mutex
	err       error
}

func newSession(application *protocoladapter.ApplicationClient, peer port.WebRTCPeer, signaling *cloudclient.SignalSession, path endpoint.Path) *Session {
	session := &Session{ApplicationClient: application, peer: peer, signaling: signaling, path: path, done: make(chan struct{})}
	go session.watchApplication(application)
	if signaling != nil {
		go session.watchSignaling(signaling)
	}
	return session
}

type cloudSessionLifecycle interface {
	Done() <-chan struct{}
	Err() error
}

func (session *Session) watchApplication(application cloudSessionLifecycle) {
	<-application.Done()
	terminalErr := application.Err()
	session.finish(cloudSessionClosure{origin: cloudSessionCloseApplication, cause: terminalErr, terminalErr: terminalErr})
}

func (session *Session) watchSignaling(signaling cloudSessionLifecycle) {
	<-signaling.Done()
	terminalErr := signaling.Err()
	session.finish(cloudSessionClosure{origin: cloudSessionCloseSignaling, cause: cloudSignalingTermination(terminalErr), terminalErr: terminalErr})
}

type cloudSessionClosure struct {
	origin      cloudSessionCloseOrigin
	cause       error
	terminalErr error
}

// ObservedPath returns the candidate pair used for the acknowledged path decision.
func (session *Session) ObservedPath() string {
	if session == nil {
		return ""
	}
	return string(session.path)
}

func cloudSignalingTermination(err error) error {
	if cloudclient.IsAdminDisconnect(err) {
		message := strings.TrimSpace(err.Error())
		if message == "" {
			message = "This connection was closed by an administrator"
		}
		return &clientruntime.Error{Code: clientruntime.ErrorUserStopped, Message: message, Cause: err}
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

func (session *Session) finish(closure cloudSessionClosure) {
	if session == nil {
		return
	}
	session.closeOnce.Do(func() {
		reportCloudSessionClose(closure.origin, closure.cause, closure.terminalErr)
		session.errMu.Lock()
		session.err = closure.cause
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
			releaseContext, cancelRelease := context.WithTimeout(context.Background(), cloudSessionReleaseTimeout)
			session.closeErr = errors.Join(session.closeErr, session.signaling.ReleaseAndWait(releaseContext))
			cancelRelease()
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
	session.finish(session.observedClosure())
	return session.closeErr
}

func (session *Session) observedClosure() cloudSessionClosure {
	if session == nil {
		return cloudSessionClosure{origin: cloudSessionCloseLocal}
	}
	var application cloudSessionLifecycle
	if session.ApplicationClient != nil {
		application = session.ApplicationClient
	}
	var signaling cloudSessionLifecycle
	if session.signaling != nil {
		signaling = session.signaling
	}
	return observeCloudSessionClosure(application, signaling)
}

func observeCloudSessionClosure(application, signaling cloudSessionLifecycle) cloudSessionClosure {
	var applicationErr error
	applicationDone := false
	if application != nil {
		select {
		case <-application.Done():
			applicationDone = true
			applicationErr = application.Err()
		default:
		}
	}
	if signaling != nil {
		select {
		case <-signaling.Done():
			terminalErr := signaling.Err()
			signalErr := cloudSignalingTermination(terminalErr)
			if cloudclient.IsAdminDisconnect(terminalErr) {
				return cloudSessionClosure{origin: cloudSessionCloseSignaling, cause: signalErr, terminalErr: terminalErr}
			}
			if applicationDone {
				return cloudSessionClosure{origin: cloudSessionCloseApplication, cause: applicationErr, terminalErr: applicationErr}
			}
			return cloudSessionClosure{origin: cloudSessionCloseSignaling, cause: signalErr, terminalErr: terminalErr}
		default:
		}
	}
	if applicationDone {
		return cloudSessionClosure{origin: cloudSessionCloseApplication, cause: applicationErr, terminalErr: applicationErr}
	}
	return cloudSessionClosure{origin: cloudSessionCloseLocal}
}

var _ clientruntime.PeerConnector = (*Dialer)(nil)
var _ clientruntime.ApplicationReadyPeerSession = (*Session)(nil)
