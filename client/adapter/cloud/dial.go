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
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/connecttrace"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const defaultClientName = "anytty-go-cloud"

const cloudLocatorStoreTimeout = 2 * time.Second
const cloudSessionReleaseTimeout = 2 * time.Second

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

// Connect gives the cached Edge locator a short head start, then refreshes through Controller
// concurrently when the cached route is slow or unavailable.
func (dialer *Dialer) Connect(ctx context.Context, request clientruntime.AttemptRequest) (result clientruntime.ReadyPeerSession, resultErr error) {
	ctx, trace := connecttrace.Start(ctx, "cloud_route")
	defer func() { trace.End(resultErr) }()
	startedAt := time.Now()
	lastAt := startedAt
	var timingMu sync.Mutex
	reportTiming := func(phase string) {
		timingMu.Lock()
		defer timingMu.Unlock()
		now := time.Now()
		log.Printf("anytty cloud connect generation=%d stage=%s stage_ms=%d total_ms=%d", request.Stamp().Generation, phase, now.Sub(lastAt).Milliseconds(), now.Sub(startedAt).Milliseconds())
		trace.Mark(phase)
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
	clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseAuthorizing, clientruntime.EndpointStageAuthorizationPreparing)
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
	clientName := strings.TrimSpace(dialer.ClientName)
	if clientName == "" {
		clientName = defaultClientName
	}
	verify := func(attemptCtx context.Context, opened *openedCloudPeer) error {
		fingerprint, err := opened.RemoteCertificateFingerprint()
		if err != nil {
			return reportCloudFailure(request.Stamp().Generation, cloudFailurePeerFingerprint, err)
		}
		dialer.report(clientruntime.EndpointPhaseAuthorizing)
		clientruntime.ReportEndpointProgress(attemptCtx, clientruntime.EndpointPhaseAuthorizing, clientruntime.EndpointStageTransportAuthorizing)
		connection := opened.Transport()
		if _, err := prepared.Authenticate(attemptCtx, connection, fingerprint); err != nil {
			return reportCloudFailure(request.Stamp().Generation, cloudFailureDataChannelAuth, fmt.Errorf("authenticate Cloud DataChannel: %w", err))
		}
		if receiver, ok := connection.(interface{ EnableReceiveBackpressure() }); ok {
			receiver.EnableReceiveBackpressure()
		}
		reportTiming("datachannel_authenticated")
		clientruntime.ReportEndpointProgress(attemptCtx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageProtocolOpening)
		opened.protocolClient = internalprotocol.NewClient(connection)
		if err := opened.protocolClient.Hello(attemptCtx, internalprotocol.Hello{Version: wire.Version, Client: clientName}); err != nil {
			return reportCloudFailure(request.Stamp().Generation, cloudFailureProtocolHello, fmt.Errorf("Cloud protocol Hello: %w", err))
		}
		reportTiming("protocol_ready")
		return nil
	}
	resolved, cachedErr := cloudclient.NewCachedCapabilityRoute(signaling.CloudEdgeLocator(), signaling.CloudRouteGrant())
	if cachedErr != nil {
		resolved = nil
	}
	opened, source, selectedResolution, err := dialCloudRoute(
		ctx,
		resolved,
		func(ctx context.Context) (*cloudclient.RouteResolution, error) {
			clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseResolving, clientruntime.EndpointStageCloudDiscovering)
			fresh, resolveErr := dialer.Cloud.Resolve(ctx, signaling.CloudRouteGrant(), signaling)
			if resolveErr == nil {
				reportTiming("controller_resolved")
				if !sameCloudRoute(resolved, fresh) {
					locator, encodeErr := cloudclient.EncodeEdgeLocator(fresh.Locator())
					if encodeErr != nil {
						return nil, fmt.Errorf("encode authenticated Cloud Edge locator: %w", encodeErr)
					}
					storeCloudEdgeLocatorAsync(ctx, request.Stamp().Generation, signaling, locator, "controller_resolve")
				}
			}
			return fresh, resolveErr
		},
		func(ctx context.Context, route *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
			if source == cloudRouteSourceCached {
				clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseResolving, clientruntime.EndpointStageCloudCachedEdge)
			} else {
				clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseResolving, clientruntime.EndpointStageCloudDiscovering)
			}
			return openResolvedCloudPeer(ctx, request, dialer.Peers, dialer.Cloud, route, signaling.ClientIdentity(), signaling, dialer.Product, dialer.report, verify)
		},
		func(source cloudRouteSource, route *cloudclient.RouteResolution, routeErr error) {
			reportTiming(string(source) + "_edge_failed")
			log.Printf("anytty cloud connect generation=%d route_source=%s route_refresh=%t error_type=%T", request.Stamp().Generation, source, shouldRefreshCloudRoute(routeErr), routeErr)
		},
	)
	if err != nil {
		failureStage := cloudFailureEdgeExchange
		if source == cloudRouteSourceController && selectedResolution == nil {
			failureStage = cloudFailureController
		}
		return nil, reportCloudFailure(request.Stamp().Generation, failureStage, cloudConnectionError(err))
	}
	protocolClient := opened.protocolClient
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
	dialer.report(clientruntime.EndpointPhaseReady)
	peer, signalSession := opened.Release()
	session := newSession(application, peer, signalSession)
	return session, nil
}

type cloudEdgeLocatorStore interface {
	StoreCloudEdgeLocator(context.Context, []byte) error
}

func storeCloudEdgeLocatorAsync(ctx context.Context, generation clientruntime.SessionGeneration, store cloudEdgeLocatorStore, locator []byte, reason string) {
	if ctx == nil || store == nil || len(locator) == 0 {
		return
	}
	go func(locator []byte) {
		storeContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), cloudLocatorStoreTimeout)
		defer cancel()
		if err := store.StoreCloudEdgeLocator(storeContext, locator); err != nil {
			log.Printf("anytty cloud connect generation=%d stage=locator_store_failed reason=%s error_type=%T", generation, reason, err)
			return
		}
		log.Printf("anytty cloud connect generation=%d stage=locator_stored reason=%s", generation, reason)
	}(append([]byte(nil), locator...))
}

type cloudRouteSource string

const (
	cloudRouteSourceCached     cloudRouteSource = "cached"
	cloudRouteSourceController cloudRouteSource = "controller"
)

// dialCloudRoute queries Controller only when the cached locator is absent or
// fails with a refreshable error. A slow but viable cached route is not a reason
// to contact Controller. Authentication/lifecycle rejection remains terminal.
func dialCloudRoute(
	ctx context.Context,
	cached *cloudclient.RouteResolution,
	resolve func(context.Context) (*cloudclient.RouteResolution, error),
	open func(context.Context, *cloudclient.RouteResolution, cloudRouteSource) (*openedCloudPeer, error),
	onFailure func(cloudRouteSource, *cloudclient.RouteResolution, error),
) (*openedCloudPeer, cloudRouteSource, *cloudclient.RouteResolution, error) {
	if ctx == nil || resolve == nil || open == nil {
		return nil, "", nil, errors.New("Cloud route recovery dependencies are incomplete")
	}
	if err := ctx.Err(); err != nil {
		return nil, "", nil, err
	}
	openRoute := func(route *cloudclient.RouteResolution, source cloudRouteSource) (*openedCloudPeer, error) {
		opened, err := open(ctx, route, source)
		if err == nil && opened == nil {
			err = fmt.Errorf("%s Cloud route returned no peer", source)
		}
		if err != nil {
			if opened != nil {
				_ = opened.Close()
				opened = nil
			}
			if onFailure != nil {
				onFailure(source, route, err)
			}
		}
		return opened, err
	}
	var cachedFailure error
	if cached != nil {
		opened, err := openRoute(cached, cloudRouteSourceCached)
		if err == nil {
			return opened, cloudRouteSourceCached, cached, nil
		}
		if ctx.Err() != nil || !shouldRefreshCloudRoute(err) {
			return nil, cloudRouteSourceCached, cached, err
		}
		cachedFailure = err
	}
	fresh, err := resolve(ctx)
	if err == nil && fresh == nil {
		err = errors.New("Cloud route resolver returned no route")
	}
	if err != nil {
		if cachedFailure != nil {
			err = combinedCloudRouteFailure(cachedFailure, err)
		}
		return nil, cloudRouteSourceController, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, cloudRouteSourceController, fresh, err
	}
	opened, err := openRoute(fresh, cloudRouteSourceController)
	if err != nil && cachedFailure != nil {
		err = combinedCloudRouteFailure(cachedFailure, err)
	}
	return opened, cloudRouteSourceController, fresh, err
}

func combinedCloudRouteFailure(cachedErr, controllerErr error) error {
	cause := errors.Join(
		fmt.Errorf("cached Cloud route failed: %w", cachedErr),
		fmt.Errorf("Controller Cloud route failed: %w", controllerErr),
	)
	return &clientruntime.Error{
		Code:      clientruntime.ErrorUnavailable,
		Message:   cause.Error(),
		Cause:     cause,
		Retryable: true,
	}
}

func sameCloudRoute(left, right *cloudclient.RouteResolution) bool {
	if left == nil || right == nil {
		return left == right
	}
	return proto.Equal(left.Locator(), right.Locator())
}

func shouldRefreshCloudRoute(err error) bool {
	if err == nil {
		return false
	}
	// Exchange marks an Edge-local protocol timeout explicitly. It must be
	// refreshable even though the wrapper still unwraps to DeadlineExceeded.
	if cloudclient.ShouldRefreshEdgeLocator(err) {
		return true
	}
	if cloudclient.SignalRejectionCode(err) != "" {
		return false
	}
	var authErr *remoteauth.HandshakeError
	if errors.As(err, &authErr) {
		switch authErr.Code {
		case remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_UNSPECIFIED,
			remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL,
			remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL:
		default:
			// A definitive peer rejection cannot be repaired by rediscovering an Edge.
			return false
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if cloudclient.IsAdminDisconnect(err) || cloudclient.IsDaemonBlocked(err) || cloudclient.IsDaemonDeleted(err) || cloudclient.EntitlementFailure(err) != nil {
		return false
	}
	// Explicit gRPC authentication and permission failures belong to the frozen credential,
	// not to the cached locator. All other non-context setup failures are bounded route failures.
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied:
		return false
	default:
		return true
	}
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
		return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: cloudRPCFailureMessage(err), Cause: err, Retryable: true}
	default:
		return err
	}
}

func cloudRPCFailureMessage(err error) string {
	code := status.Code(err)
	detail := strings.Join(strings.Fields(status.Convert(err).Message()), " ")
	if detail == "" {
		return fmt.Sprintf("Cloud connection failed (RPC %s)", code)
	}
	return fmt.Sprintf("Cloud connection failed (RPC %s): %s", code, detail)
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

type cloudSessionSignaling interface {
	Done() <-chan struct{}
	PathConfirmed() bool
	ReleaseAndWait(context.Context) error
}

func releaseCloudSession(signaling cloudSessionSignaling) (err error) {
	if signaling == nil || !signaling.PathConfirmed() {
		return nil
	}
	select {
	case <-signaling.Done():
		// Edge has already observed the signaling stream end and will run its
		// deferred session cleanup; there is no stream left to send release on.
		return nil
	default:
	}
	ctx, cancel := context.WithTimeout(context.Background(), cloudSessionReleaseTimeout)
	defer cancel()
	started := time.Now()
	defer func() {
		log.Printf("anytty cloud close stage=edge_release elapsed_ms=%d error_type=%T", time.Since(started).Milliseconds(), err)
	}()
	return signaling.ReleaseAndWait(ctx)
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
		return &clientruntime.Error{Code: clientruntime.ErrorConnectionStopped, Message: message, Cause: err}
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
		if err := releaseCloudSession(session.signaling); err != nil {
			session.closeErr = errors.Join(session.closeErr, err)
		}
		if session.ApplicationClient != nil {
			session.closeErr = errors.Join(session.closeErr, session.ApplicationClient.Close())
		}
		if session.peer != nil {
			session.closeErr = errors.Join(session.closeErr, session.peer.Close())
		}
		if session.signaling != nil {
			session.closeErr = errors.Join(session.closeErr, session.signaling.Close())
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
