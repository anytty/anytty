// Package enginehost 提供 Android JNI 与浏览器 WASM 共用的 Go Client Engine composition root。
// 平台只能注入 credential primitive；Endpoint Route、remote-auth、Hello、Proto API 与 generation 真值留在 Go。
package enginehost

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	cloudadapter "github.com/anytty/anytty/client/adapter/cloud"
	"github.com/anytty/anytty/client/adapter/direct"
	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	shareadapter "github.com/anytty/anytty/client/adapter/share"
	sshadapter "github.com/anytty/anytty/client/adapter/ssh"
	systemadapter "github.com/anytty/anytty/client/adapter/system"
	"github.com/anytty/anytty/client/binding"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const cloudPresenceLocatorStoreTimeout = 2 * time.Second
const (
	cloudPresenceLocatorSourceCachedEdge = "cached_edge"
	cloudPresenceLocatorSourceController = "controller"
)

// Options 定义单个平台 generation 的 primitive 依赖。
// Broker 与 peer factory 必须只属于当前 engine；关闭后不得复用到下一代。
type Options struct {
	Broker               *binding.PlatformBroker
	DirectPeers          direct.PeerFactory
	SSHCredentials       port.SSHCredentialSource
	ClientName           string
	CredentialPrefix     string
	Now                  func() time.Time
	SessionAuthority     *clientruntime.SessionGenerationAuthority
	CloudProduct         cloudv1.ClientProduct
	CloudProfileResolve  func(context.Context, string) (*bindingpb.CloudProfileRecord, error)
	EnableLocalDiscovery bool
	ShareReceive         func(context.Context, *remoteauthpb.ShareSessionOffer) (*remoteauthpb.ClientEndpointShareBundleV1, error)
}

// Host 是跨 Android/Web 共用的 binding.Host、PairingHost 与 CredentialHost。
type Host struct {
	options        Options
	owner          *clientruntime.SessionOwner
	supervisor     *clientruntime.EndpointSupervisor
	cloudBootID    string
	registryMu     sync.Mutex
	registry       endpoint.Registry
	registryLoaded bool
	pendingShares  map[string]*remoteauthpb.ClientEndpointShareBundleV1
	closeOnce      sync.Once
}

// New 校验平台依赖并创建共享 managed host。
func New(options Options) (*Host, error) {
	if options.Broker == nil || options.DirectPeers == nil {
		return nil, fmt.Errorf("binding engine host requires broker and Direct peer factory")
	}
	options.ClientName = strings.TrimSpace(options.ClientName)
	if options.ClientName == "" {
		return nil, fmt.Errorf("binding client name is required")
	}
	options.CredentialPrefix = strings.TrimSpace(options.CredentialPrefix)
	if options.CredentialPrefix == "" {
		return nil, fmt.Errorf("binding credential prefix is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.CloudProduct != cloudv1.ClientProduct_CLIENT_PRODUCT_UNSPECIFIED && !validCloudProduct(options.CloudProduct) {
		return nil, fmt.Errorf("binding Cloud client product is invalid")
	}
	if options.SSHCredentials == nil && options.DirectPeers != nil {
		// Android/native binding 默认复用同一 platform broker 的不可导出 SSH signer；浏览器没有 Direct primitive，不会启用该 Route。
		options.SSHCredentials = platformSSHCredential{broker: options.Broker}
	}
	if options.ShareReceive == nil {
		options.ShareReceive = shareadapter.Receive
	}
	host := &Host{options: options, owner: clientruntime.NewSessionOwnerWithAuthority(options.SessionAuthority), cloudBootID: uuid.NewString(), pendingShares: make(map[string]*remoteauthpb.ClientEndpointShareBundleV1)}
	supervisor, err := clientruntime.NewEndpointSupervisor(endpointSupervisorController{host: host}, clientruntime.EndpointSupervisorOptions{})
	if err != nil {
		_ = host.owner.Close()
		return nil, err
	}
	host.supervisor = supervisor
	return host, nil
}

// OpenSession acquires the supervisor-owned endpoint winner. An explicit Route
// override is the diagnostic escape hatch that may establish a direct session.
func (host *Host) OpenSession(ctx context.Context, request *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
	if request == nil {
		return nil, fmt.Errorf("open session request is required")
	}
	endpointID := endpoint.EndpointID(strings.TrimSpace(request.GetEndpointId()))
	if endpointID == "" {
		return nil, fmt.Errorf("open session endpoint_id is required")
	}
	if strings.TrimSpace(request.GetRouteOverride()) == "" {
		return host.supervisor.Acquire(ctx, endpointID)
	}
	return host.openSessionDirect(ctx, request)
}

// AdvanceEndpointIntent records OpenSession acceptance before binding schedules
// its asynchronous host call.
func (host *Host) AdvanceEndpointIntent(endpointID endpoint.EndpointID) error {
	return host.owner.AdvanceEndpointIntent(endpointID)
}

func (host *Host) openSessionDirect(ctx context.Context, request *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
	intent, err := connectIntent(request.GetIntent())
	if err != nil {
		return nil, err
	}
	endpointID := endpoint.EndpointID(strings.TrimSpace(request.GetEndpointId()))
	target, err := host.registryEndpoint(ctx, endpointID)
	if err != nil {
		return nil, err
	}
	routeID := endpoint.RouteID(strings.TrimSpace(request.GetRouteOverride()))
	credentials := newPlatformCredentials(host.options.Broker)
	authorizer := peeradapter.CapabilityAuthorizer{Credentials: credentials, Signers: credentials, Now: host.options.Now}
	profiles := host.cloudProfiles()
	planningTarget, environment, err := routePlanEnvironment(ctx, target, host.options, credentials, profiles)
	if err != nil {
		return nil, err
	}
	dialers, err := clientruntime.NewPeerConnectorMap(host.routeConnectorsWithProfiles(authorizer, profiles))
	if err != nil {
		return nil, err
	}
	wireTarget, err := endpoint.EndpointToProto(planningTarget)
	if err != nil {
		return nil, err
	}
	config := sessionConfig(wireTarget, routeID, request.GetIntent(), environment)
	return host.owner.AcquirePlanned(ctx, planningTarget, routeID, intent, config, environment, systemadapter.Clock{}, dialers)
}

type endpointSupervisorController struct{ host *Host }

func (controller endpointSupervisorController) AdvanceEndpointIntent(endpointID endpoint.EndpointID) error {
	return controller.host.owner.AdvanceEndpointIntent(endpointID)
}

func (controller endpointSupervisorController) AcquireCurrent(endpointID endpoint.EndpointID) (clientruntime.ApplicationReadyPeerSession, error) {
	return controller.host.owner.AcquireCurrent(endpointID)
}

func (controller endpointSupervisorController) Connect(ctx context.Context, endpointID endpoint.EndpointID) (clientruntime.ApplicationReadyPeerSession, error) {
	return controller.host.openSessionDirect(ctx, &bindingpb.OpenSessionRequest{
		EndpointId: string(endpointID),
		Intent:     bindingpb.ConnectIntent_CONNECT_INTENT_BACKGROUND,
	})
}

func (controller endpointSupervisorController) Probe(ctx context.Context, session clientruntime.ApplicationReadyPeerSession) error {
	result, err := session.ExecuteApplication(ctx, &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsCommand{}},
	})
	if err != nil {
		return err
	}
	if result.GetTerminalDefaults() == nil {
		return fmt.Errorf("endpoint supervisor probe returned no terminal defaults")
	}
	return nil
}

func (controller endpointSupervisorController) Invalidate(stamp clientruntime.EndpointSessionStamp, cause error) error {
	return controller.host.owner.InvalidateSessionFast(stamp, cause)
}

// WatchEndpoint exposes the SessionOwner lifecycle to the cross-language
// binding without giving the platform ownership of connection state.
func (host *Host) WatchEndpoint(ctx context.Context, endpointID endpoint.EndpointID) (<-chan clientruntime.EndpointEvent, error) {
	return host.owner.WatchEndpoint(ctx, endpointID)
}

// ReplaceEndpointDemand atomically reconciles the process-owned Android demand
// projection. The supervisor never consumes renderer deltas.
func (host *Host) ReplaceEndpointDemand(snapshot clientruntime.EndpointDemandSnapshot) error {
	return host.supervisor.ReplaceDemand(snapshot)
}

// SignalEndpointHost wakes demanded endpoint workers after a foreground or
// stable network revision. The signal is a hint; only the application probe can
// invalidate a retained winner.
func (host *Host) SignalEndpointHost(signal clientruntime.EndpointHostSignal) error {
	return host.supervisor.Signal(signal)
}

// RepairEndpoint forces a fresh application probe for one demanded endpoint.
func (host *Host) RepairEndpoint(endpointID endpoint.EndpointID) error {
	return host.supervisor.Repair(endpointID)
}

func (host *Host) WaitEndpointDemandReady(ctx context.Context) error {
	return host.supervisor.WaitReady(ctx)
}

func (host *Host) EndpointSupervisorSnapshot() []clientruntime.EndpointSupervisorProjection {
	return host.supervisor.Snapshot()
}

// InvalidateSession removes the exact current generation before the next
// OpenSession samples platform networking again. Stale handles cannot close a
// newer winner because SessionOwner validates the complete stamp.
func (host *Host) InvalidateSession(ctx context.Context, stamp clientruntime.EndpointSessionStamp) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return host.owner.InvalidateSessionFast(stamp, nil)
}

// PrepareEndpointDisconnect captures the endpoint intent before the binding
// schedules its asynchronous physical disconnect.
func (host *Host) PrepareEndpointDisconnect(endpointID endpoint.EndpointID) (binding.EndpointDisconnectOperation, error) {
	return host.owner.PrepareEndpointDisconnect(endpointID)
}

// GetEndpointCloudPresence prefers the endpoint's cached Edge locator, then
// falls back to the Controller when the cached locator is invalid, unreachable,
// or reports offline. The fallback still requires the paired CloudRouteGrant and
// ClientAccessIdentity proof before any current Edge locator is returned.
func (host *Host) GetEndpointCloudPresence(ctx context.Context, request *bindingpb.EndpointCloudPresenceGetRequest) (*bindingpb.EndpointCloudPresenceGetResult, error) {
	startedAt := time.Now()
	if request == nil || strings.TrimSpace(request.GetEndpointId()) == "" {
		log.Printf("anytty cloud presence stage=request_invalid")
		return nil, fmt.Errorf("endpoint Cloud presence request is incomplete")
	}
	log.Printf("anytty cloud presence stage=request")
	target, err := host.registryEndpoint(ctx, endpoint.EndpointID(strings.TrimSpace(request.GetEndpointId())))
	if err != nil {
		log.Printf("anytty cloud presence stage=registry_failed total_ms=%d error_type=%T", time.Since(startedAt).Milliseconds(), err)
		return nil, err
	}
	var cloudRoute *endpoint.AccessRoute
	for _, candidate := range target.RouteList() {
		if candidate.Kind == endpoint.RouteManagedWebRTC && candidate.Enabled {
			value := candidate
			cloudRoute = &value
			break
		}
	}
	if cloudRoute == nil {
		log.Printf("anytty cloud presence stage=route_missing total_ms=%d", time.Since(startedAt).Milliseconds())
		return nil, fmt.Errorf("endpoint %q has no enabled Cloud route", request.GetEndpointId())
	}
	credentials := newPlatformCredentials(host.options.Broker)
	attempt, err := clientruntime.NewAttemptRequest(target, cloudRoute.ID, clientruntime.SessionGeneration(1), clientruntime.ConnectIntentProbe)
	if err != nil {
		log.Printf("anytty cloud presence stage=attempt_failed total_ms=%d error_type=%T", time.Since(startedAt).Milliseconds(), err)
		return nil, err
	}
	prepared, err := (peeradapter.CapabilityAuthorizer{Credentials: credentials, Signers: credentials, Now: host.options.Now}).Prepare(ctx, attempt)
	if err != nil {
		log.Printf("anytty cloud presence stage=authorization_failed total_ms=%d error_type=%T", time.Since(startedAt).Milliseconds(), err)
		return nil, err
	}
	signaling, ok := prepared.(peeradapter.PreparedSignalingAuthorization)
	if !ok || len(signaling.CloudRouteGrant()) == 0 {
		log.Printf("anytty cloud presence stage=credential_incomplete total_ms=%d", time.Since(startedAt).Milliseconds())
		return nil, fmt.Errorf("endpoint Cloud route credential is incomplete")
	}
	protocolClient, err := host.cloudProfiles().Resolve(ctx, cloudRoute.AccountProfileRef)
	if err != nil {
		log.Printf("anytty cloud presence stage=profile_failed total_ms=%d error_type=%T", time.Since(startedAt).Milliseconds(), err)
		return nil, err
	}
	resolution, err := cloudclient.NewCachedCapabilityRoute(signaling.CloudEdgeLocator(), signaling.CloudRouteGrant())
	needsRefresh := false
	if err != nil {
		log.Printf("anytty cloud presence stage=locator_invalid total_ms=%d error_type=%T", time.Since(startedAt).Milliseconds(), err)
		needsRefresh = true
	} else {
		locator := resolution.Locator()
		log.Printf("anytty cloud presence stage=edge_probe")
		online, probeErr := protocolClient.ProbePresence(ctx, resolution, signaling.ClientIdentity(), signaling, host.options.CloudProduct)
		if probeErr != nil {
			needsRefresh = cloudclient.ShouldRefreshEdgeLocator(probeErr)
			log.Printf("anytty cloud presence stage=edge_failed total_ms=%d rpc_code=%s refresh_locator=%t error_type=%T", time.Since(startedAt).Milliseconds(), status.Code(probeErr), needsRefresh, probeErr)
			if !needsRefresh {
				return nil, probeErr
			}
		} else if online {
			log.Printf("anytty cloud presence stage=complete total_ms=%d online=true source=cached_edge", time.Since(startedAt).Milliseconds())
			return endpointCloudPresenceGetResult(request.GetEndpointId(), target, signaling.CloudRouteGrant(), locator, cloudPresenceLocatorSourceCachedEdge, false, true), nil
		} else {
			needsRefresh = true
			log.Printf("anytty cloud presence stage=edge_offline total_ms=%d refresh_locator=true", time.Since(startedAt).Milliseconds())
		}
	}
	if !needsRefresh {
		return endpointCloudPresenceGetResult(request.GetEndpointId(), target, signaling.CloudRouteGrant(), nil, "", false, false), nil
	}
	resolution, err = protocolClient.Resolve(ctx, signaling.CloudRouteGrant(), signaling)
	if err != nil {
		if cloudclient.IsDaemonOffline(err) {
			log.Printf("anytty cloud presence stage=controller_offline total_ms=%d", time.Since(startedAt).Milliseconds())
			return endpointCloudPresenceGetResult(request.GetEndpointId(), target, signaling.CloudRouteGrant(), nil, cloudPresenceLocatorSourceController, true, false), nil
		}
		log.Printf("anytty cloud presence stage=controller_failed total_ms=%d rpc_code=%s error_type=%T", time.Since(startedAt).Milliseconds(), status.Code(err), err)
		return nil, err
	}
	locator := resolution.Locator()
	if locator == nil {
		return nil, fmt.Errorf("Cloud route resolution returned no Edge locator")
	}
	log.Printf("anytty cloud presence stage=controller_resolved")
	online, err := protocolClient.ProbePresence(ctx, resolution, signaling.ClientIdentity(), signaling, host.options.CloudProduct)
	if err != nil {
		log.Printf("anytty cloud presence stage=resolved_edge_failed total_ms=%d rpc_code=%s refresh_locator=%t error_type=%T", time.Since(startedAt).Milliseconds(), status.Code(err), cloudclient.ShouldRefreshEdgeLocator(err), err)
		return nil, err
	}
	if online {
		locatorPayload, err := cloudclient.EncodeEdgeLocator(locator)
		if err != nil {
			return nil, fmt.Errorf("encode authenticated Cloud Edge locator: %w", err)
		}
		storeContext, cancelStore := context.WithTimeout(context.WithoutCancel(ctx), cloudPresenceLocatorStoreTimeout)
		storeErr := signaling.StoreCloudEdgeLocator(storeContext, locatorPayload)
		cancelStore()
		if storeErr != nil {
			log.Printf("anytty cloud presence stage=locator_store_failed total_ms=%d error_type=%T", time.Since(startedAt).Milliseconds(), storeErr)
		}
	}
	log.Printf("anytty cloud presence stage=complete total_ms=%d online=%t source=controller", time.Since(startedAt).Milliseconds(), online)
	return endpointCloudPresenceGetResult(request.GetEndpointId(), target, signaling.CloudRouteGrant(), locator, cloudPresenceLocatorSourceController, true, online), nil
}

func endpointCloudPresenceGetResult(endpointID string, target endpoint.Endpoint, routeGrantPayload []byte, locator *cloudv1.EdgeLocator, source string, refreshed bool, online bool) *bindingpb.EndpointCloudPresenceGetResult {
	result := &bindingpb.EndpointCloudPresenceGetResult{
		EndpointId:              endpointID,
		Online:                  online,
		DeviceId:                target.DaemonIdentity.DeviceID,
		DeviceFingerprint:       target.DaemonIdentity.DeviceFingerprint,
		DaemonId:                cloudRouteGrantDaemonID(routeGrantPayload),
		LocatorSource:           source,
		RefreshedFromController: refreshed,
	}
	if locator != nil {
		result.EdgeId = locator.GetEdgeId()
		result.EdgeName = locator.GetName()
		result.EdgeRegion = locator.GetRegion()
		result.EdgePublicEndpoint = locator.GetPublicEndpoint()
		result.EdgeServerName = locator.GetServerName()
	}
	return result
}

func cloudRouteGrantDaemonID(payload []byte) string {
	envelope := &cloudv1.SignedEnvelope{}
	if len(payload) == 0 || proto.Unmarshal(payload, envelope) != nil {
		return ""
	}
	claims := &cloudv1.CloudRouteGrantClaims{}
	if len(envelope.GetPayload()) == 0 || proto.Unmarshal(envelope.GetPayload(), claims) != nil {
		return ""
	}
	return strings.TrimSpace(claims.GetDaemonId())
}

func (host *Host) routeConnectors(authorizer peeradapter.CapabilityAuthorizer) map[endpoint.RouteKind]clientruntime.PeerConnector {
	return host.routeConnectorsWithProfiles(authorizer, host.cloudProfiles())
}

func (host *Host) routeConnectorsWithProfiles(authorizer peeradapter.CapabilityAuthorizer, profiles platformCloudProfiles) map[endpoint.RouteKind]clientruntime.PeerConnector {
	connectors := make(map[endpoint.RouteKind]clientruntime.PeerConnector, 3)
	if host.options.DirectPeers != nil {
		connectors[endpoint.RouteDirectWebRTCTCP] = &direct.Dialer{
			Peers: host.options.DirectPeers, Authorization: authorizer, ClientName: host.options.ClientName, Now: host.options.Now,
		}
		if host.options.SSHCredentials != nil {
			connectors[endpoint.RouteSSHWebRTCTCP] = sshadapter.NewDialer(sshadapter.Options{
				Peers: host.options.DirectPeers, Authorization: authorizer, Credentials: host.options.SSHCredentials,
				ClientName: host.options.ClientName,
			})
		}
	}
	cloudPeers, cloudSupported := host.options.DirectPeers.(cloudadapter.PeerFactory)
	if validCloudProduct(host.options.CloudProduct) && cloudSupported {
		connectors[endpoint.RouteManagedWebRTC] = &platformCloudConnector{
			profiles: profiles, peers: cloudPeers,
			authorization: authorizer, product: host.options.CloudProduct, clientName: host.options.ClientName,
		}
	}
	return connectors
}

// ImportPairing 验证 bootstrap、使用平台不可导出 signer 完成 PairingExchange，并原子绑定 grant。
func (host *Host) ImportPairing(ctx context.Context, request *bindingpb.ImportPairingRequest) (*bindingpb.ImportPairingResult, error) {
	payload, err := decodeBootstrap(request.GetPortablePayload())
	if err != nil {
		return nil, err
	}
	now := host.options.Now().UTC()
	offer, err := remoteauth.ParsePairingClaimOffer(payload, now)
	if err != nil {
		return nil, fmt.Errorf("parse pairing claim offer at %s: %w", now.Format(time.RFC3339Nano), err)
	}
	pairingCandidate, err := remoteauth.PairingClaimEndpointCandidate(offer)
	if err != nil {
		return nil, err
	}
	endpointID := strings.TrimSpace(request.GetExpectedEndpointId())
	if endpointID == "" {
		endpointID = offer.GetDeviceId()
	}
	deviceFingerprint := remoteauth.Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey()))
	credentialRef := credentialRef(host.options.CredentialPrefix, offer.GetDeviceId(), deviceFingerprint)
	response, err := host.options.Broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CredentialPrepare{
		CredentialPrepare: &bindingpb.CredentialPrepareRequest{EndpointId: endpointID, CredentialRef: credentialRef},
	}})
	if err != nil {
		return nil, err
	}
	record, err := platformCredential(response)
	if err != nil {
		return nil, err
	}
	pairingRoutes, err := pairingClaimRoutes(pairingCandidate, host.options)
	if err != nil {
		return nil, err
	}
	target := pairingTarget(endpointID, pairingCandidate.Identity, pairingRoutes, credentialRef)
	if host.options.EnableLocalDiscovery {
		target = applyPlatformLocalDiscovery(ctx, target, host.options)
	}
	routeList := target.RouteList()
	routeIDs := make([]endpoint.RouteID, 0, len(routeList))
	for _, route := range routeList {
		routeIDs = append(routeIDs, route.ID)
	}
	attempts, err := host.owner.BeginRouteAttempts(target, routeIDs, clientruntime.ConnectIntentInteractive)
	if err != nil {
		return nil, err
	}
	identity := remoteauth.ClientAccessIdentity{
		EndpointID: endpointID, PublicKey: append(ed25519.PublicKey(nil), record.GetPublicKey()...), Fingerprint: record.GetKeyFingerprint(),
	}
	if err := identity.ValidatePublic(); err != nil {
		return nil, fmt.Errorf("pairing identity is invalid: %w", err)
	}
	signer := platformSigner{broker: host.options.Broker, credentialRef: credentialRef, identity: identity}
	pairingRequest := remoteauth.ClientPairingRequest{
		ExpectedDeviceID: offer.GetDeviceId(), ExpectedDeviceFingerprint: deviceFingerprint,
		PairingClaimOffer: payload, Identity: identity, Signer: signer, ClientLabel: host.options.ClientName, ClientProduct: uint32(host.options.CloudProduct),
	}
	paired, err := host.redeemPairingRace(ctx, attempts, pairingRequest)
	if err != nil {
		return nil, err
	}
	// PairingAccepted 只会在 owning daemon 已原子兑换 ticket 后返回；这里重新校验 ticket
	// 本地时钟会把合法的跨设备小幅 clock skew 误判为过期。签名、身份、ticket 对应关系与
	// 带容差的 grant 已由 ClientPairingHandshake 验证，此处只解析已接受的持久 Endpoint 配置。
	bundle, claims, err := remoteauth.ParsePairingBundleForExchange(paired.Bundle)
	if err != nil {
		return nil, fmt.Errorf("parse accepted pairing bundle: %w", err)
	}
	candidate, err := endpoint.EndpointCandidateFromBootstrapBundle(bundle)
	if err != nil {
		return nil, err
	}
	boundResponse, err := host.options.Broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CredentialBind{
		CredentialBind: &bindingpb.CredentialBindRequest{EndpointId: endpointID, CredentialRef: credentialRef, CapabilityGrant: paired.Grant, CloudRouteGrant: paired.CloudRouteGrant, CloudEdgeLocator: paired.CloudEdgeLocator},
	}})
	if err != nil {
		return nil, err
	}
	bound, err := platformCredential(boundResponse)
	if err != nil {
		return nil, err
	}
	if bound.GetKeyFingerprint() != identity.Fingerprint || bound.GetCapabilityGrant() != paired.Grant || !bytes.Equal(bound.GetCloudRouteGrant(), paired.CloudRouteGrant) || !bytes.Equal(bound.GetCloudEdgeLocator(), paired.CloudEdgeLocator) {
		return nil, fmt.Errorf("platform secure store bound a different pairing credential")
	}
	committed, registry, err := host.commitPairingEndpoint(ctx, endpoint.EndpointID(endpointID), candidate, credentialRef)
	if err != nil {
		rollbackContext, cancelRollback := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelRollback()
		rollbackErr := host.rollbackPreparedCredential(rollbackContext, record, paired.Grant, paired.CloudRouteGrant, paired.CloudEdgeLocator)
		if rollbackErr != nil {
			return nil, fmt.Errorf("commit paired endpoint: %v; rollback credential: %w", err, rollbackErr)
		}
		return nil, err
	}
	expiresAtUnixNano := int64(0)
	if !paired.ExpiresAt.IsZero() {
		expiresAtUnixNano = paired.ExpiresAt.UnixNano()
	}
	return &bindingpb.ImportPairingResult{
		Endpoint: committed, Registry: registry, TicketId: claims.TicketID, ClientKeyFingerprint: record.GetKeyFingerprint(),
		ExpiresAtUnixNano: expiresAtUnixNano, AuthorizationRequired: false,
	}, nil
}

// pairingClaimRoutes 返回当前 Go Client Engine 可执行的 claim Route；平台 UI 不参与筛选或排序。
func pairingClaimRoutes(candidate endpoint.EndpointCandidate, options Options) ([]endpoint.AccessRoute, error) {
	routes := make([]endpoint.AccessRoute, 0, len(candidate.Routes))
	for _, route := range candidate.Routes {
		if !route.Enabled {
			continue
		}
		switch route.Kind {
		case endpoint.RouteDirectWebRTCTCP:
			if options.DirectPeers != nil {
				routes = append(routes, route)
			}
		case endpoint.RouteSSHWebRTCTCP:
			if options.DirectPeers != nil && options.SSHCredentials != nil {
				routes = append(routes, route)
			}
		case endpoint.RouteManagedWebRTC:
			_, cloudPeers := options.DirectPeers.(cloudadapter.PeerFactory)
			if cloudPeers && options.Broker != nil && validCloudProduct(options.CloudProduct) {
				routes = append(routes, route)
			}
		}
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("pairing claim has no Route supported by this client")
	}
	return routes, nil
}

type pairingRaceResult struct {
	paired remoteauth.PairingExchangeResult
	err    error
}

// redeemPairingRace 并发尝试同一 generation 的 Route，首个成功立即获胜；失败 Route 只贡献本次诊断，不写入 Endpoint 真值。
func (host *Host) redeemPairingRace(ctx context.Context, attempts []clientruntime.AttemptRequest, request remoteauth.ClientPairingRequest) (remoteauth.PairingExchangeResult, error) {
	raceContext, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan pairingRaceResult, len(attempts))
	for _, attempt := range attempts {
		attempt := attempt
		go func() {
			var paired remoteauth.PairingExchangeResult
			var err error
			switch attempt.Route().Kind {
			case endpoint.RouteDirectWebRTCTCP:
				paired, err = (&direct.PairingConnector{Peers: host.options.DirectPeers, Now: host.options.Now}).Redeem(raceContext, attempt, request)
			case endpoint.RouteSSHWebRTCTCP:
				paired, err = (&sshadapter.PairingConnector{Peers: host.options.DirectPeers, Credentials: host.options.SSHCredentials, Now: host.options.Now}).Redeem(raceContext, attempt, request)
			case endpoint.RouteManagedWebRTC:
				cloudPeers, ok := host.options.DirectPeers.(cloudadapter.PeerFactory)
				if !ok {
					err = fmt.Errorf("Cloud pairing peer factory is unavailable")
					break
				}
				protocolClient, resolveErr := host.cloudProfiles().Resolve(raceContext, attempt.Route().AccountProfileRef)
				if resolveErr != nil {
					err = resolveErr
					break
				}
				paired, err = (&cloudadapter.PairingConnector{Peers: cloudPeers, Cloud: protocolClient, Product: host.options.CloudProduct, Now: host.options.Now}).Redeem(raceContext, attempt, request)
			default:
				err = fmt.Errorf("pairing Route %q is unsupported", attempt.Route().Kind)
			}
			results <- pairingRaceResult{paired: paired, err: err}
		}()
	}
	var failures []error
	for range attempts {
		result := <-results
		if result.err == nil {
			cancel()
			return result.paired, nil
		}
		failures = append(failures, result.err)
	}
	return remoteauth.PairingExchangeResult{}, fmt.Errorf("all pairing Routes failed: %w", errors.Join(failures...))
}

// DeleteCredential 删除当前平台 secure store 中的 credential record。
func (host *Host) DeleteCredential(ctx context.Context, request *bindingpb.DeleteCredentialRequest) error {
	response, err := host.options.Broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CredentialDelete{
		CredentialDelete: &bindingpb.CredentialDeleteRequest{CredentialRef: request.GetCredentialRef()},
	}})
	if err != nil {
		return err
	}
	return platformResponseError(response)
}

// Close 关闭当前 generation 的 peer factory 与 broker，解除冻结中的平台等待。
func (host *Host) Close() error {
	if host == nil {
		return nil
	}
	host.closeOnce.Do(func() {
		host.registryMu.Lock()
		host.pendingShares = make(map[string]*remoteauthpb.ClientEndpointShareBundleV1)
		host.registryMu.Unlock()
		_ = host.supervisor.Close()
		_ = host.owner.Close()
		if closer, ok := host.options.DirectPeers.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		_ = host.options.Broker.Close()
	})
	return nil
}

// Broker 返回当前 engine 独占的平台 broker，供 JNI/WASM wrapper 驱动。
func (host *Host) Broker() *binding.PlatformBroker { return host.options.Broker }

func sessionConfig(config *remoteauthpb.EndpointConfigV1, routeID endpoint.RouteID, intent bindingpb.ConnectIntent, environment clientruntime.RoutePlanEnvironment) string {
	payload, _ := (proto.MarshalOptions{Deterministic: true}).Marshal(config)
	return fmt.Sprintf("%s\x00%d\x00%x\x00%s\x00%s", routeID, intent, sha256.Sum256(payload), joinRouteKinds(environment.SupportedRouteKinds), strings.Join(environment.AvailableCredentialRefs, "\x00"))
}

type clientCredentialAvailability interface {
	Available(context.Context, string, string) bool
}

type sshCredentialAvailability interface {
	Available(string) bool
}

type contextSSHCredentialAvailability interface {
	AvailableContext(context.Context, string) bool
}

type cloudCredentialAvailability interface {
	CloudAvailable(context.Context, string, string) bool
}

// routePlanEnvironment 生成当前调用的平台、凭据与 Cloud profile 能力快照。
// 禁用只影响本次规划，不能删除持久 Endpoint，也不能阻断 Direct/SSH。
func routePlanEnvironment(ctx context.Context, target endpoint.Endpoint, options Options, credentials clientCredentialAvailability, cloudProfiles platformCloudProfiles) (endpoint.Endpoint, clientruntime.RoutePlanEnvironment, error) {
	wireTarget, err := endpoint.EndpointToProto(target)
	if err != nil {
		return endpoint.Endpoint{}, clientruntime.RoutePlanEnvironment{}, err
	}
	planningTarget, err := endpoint.EndpointFromProto(wireTarget)
	if err != nil {
		return endpoint.Endpoint{}, clientruntime.RoutePlanEnvironment{}, err
	}
	if options.EnableLocalDiscovery {
		planningTarget = applyPlatformLocalDiscovery(ctx, planningTarget, options)
	}
	environment := clientruntime.RoutePlanEnvironment{}
	if options.DirectPeers != nil {
		environment.SupportedRouteKinds = append(environment.SupportedRouteKinds, endpoint.RouteDirectWebRTCTCP)
	}
	_, sshContextSupported := options.SSHCredentials.(contextSSHCredentialAvailability)
	_, sshLegacySupported := options.SSHCredentials.(sshCredentialAvailability)
	sshSupported := options.DirectPeers != nil && options.SSHCredentials != nil && (sshContextSupported || sshLegacySupported)
	if sshSupported {
		environment.SupportedRouteKinds = append(environment.SupportedRouteKinds, endpoint.RouteSSHWebRTCTCP)
	}
	_, cloudPeerSupported := options.DirectPeers.(cloudadapter.PeerFactory)
	cloudSupported := cloudPeerSupported && validCloudProduct(options.CloudProduct)
	if cloudSupported {
		environment.SupportedRouteKinds = append(environment.SupportedRouteKinds, endpoint.RouteManagedWebRTC)
	}
	available := make(map[string]struct{})
	credentialChecked := make(map[string]bool)
	for _, route := range planningTarget.RouteList() {
		if route.Kind != endpoint.RouteManagedWebRTC && route.CredentialRef != "" && credentials != nil {
			credentialAvailable, checked := credentialChecked[route.CredentialRef]
			if !checked {
				credentialAvailable = credentials.Available(ctx, string(planningTarget.ID), route.CredentialRef)
				credentialChecked[route.CredentialRef] = credentialAvailable
			}
			if credentialAvailable {
				available[route.CredentialRef] = struct{}{}
			}
		}
		switch route.Kind {
		case endpoint.RouteSSHWebRTCTCP:
			if sshSupported && sshCredentialAvailable(ctx, options.SSHCredentials, route.SSHCredentialRef) {
				available[route.SSHCredentialRef] = struct{}{}
			}
		case endpoint.RouteManagedWebRTC:
			cloudCredentials, credentialSupported := credentials.(cloudCredentialAvailability)
			if !cloudSupported || !credentialSupported || !cloudProfiles.Available(ctx, route.AccountProfileRef) || !cloudCredentials.CloudAvailable(ctx, string(planningTarget.ID), route.CredentialRef) {
				route.Enabled = false
				planningTarget.Routes[route.ID] = route
				continue
			}
			available[route.CredentialRef] = struct{}{}
		}
	}
	for _, route := range planningTarget.RouteList() {
		for _, reference := range []string{route.CredentialRef, route.SSHCredentialRef} {
			if _, ok := available[reference]; ok {
				environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
				delete(available, reference)
			}
		}
	}
	return planningTarget, environment, nil
}

func sshCredentialAvailable(ctx context.Context, source port.SSHCredentialSource, reference string) bool {
	if availability, ok := source.(contextSSHCredentialAvailability); ok {
		return availability.AvailableContext(ctx, reference)
	}
	if availability, ok := source.(sshCredentialAvailability); ok {
		return availability.Available(reference)
	}
	return false
}

func joinRouteKinds(kinds []endpoint.RouteKind) string {
	values := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		values = append(values, string(kind))
	}
	return strings.Join(values, "\x00")
}

func pairingTarget(endpointID string, identity endpoint.DaemonIdentity, routes []endpoint.AccessRoute, credentialRef string) endpoint.Endpoint {
	targetRoutes := make(map[endpoint.RouteID]endpoint.AccessRoute, len(routes))
	for _, route := range routes {
		route.CredentialRef = credentialRef
		targetRoutes[route.ID] = route
	}
	return endpoint.Endpoint{
		ID: endpoint.EndpointID(endpointID), DaemonIdentity: identity,
		Routes: targetRoutes,
	}
}

type platformCloudProfiles struct {
	broker  *binding.PlatformBroker
	resolve func(context.Context, string) (*bindingpb.CloudProfileRecord, error)
	bootID  string
	cache   *platformCloudProfileCache
}

type platformCloudProfileCache struct {
	mu      sync.Mutex
	clients map[string]*cloudclient.Client
}

func (host *Host) cloudProfiles() platformCloudProfiles {
	return platformCloudProfiles{
		broker:  host.options.Broker,
		resolve: host.options.CloudProfileResolve,
		bootID:  host.cloudBootID,
		cache:   &platformCloudProfileCache{clients: make(map[string]*cloudclient.Client)},
	}
}

// Resolve 只把平台 profile 投影成 Go Cloud protocol client；平台不建立网络连接，也不拥有 attempt。
func (source platformCloudProfiles) Resolve(ctx context.Context, reference string) (*cloudclient.Client, error) {
	reference = strings.TrimSpace(reference)
	if (source.broker == nil && source.resolve == nil) || reference == "" {
		return nil, fmt.Errorf("Cloud account profile is unavailable")
	}
	if source.cache != nil {
		source.cache.mu.Lock()
		defer source.cache.mu.Unlock()
		if client := source.cache.clients[reference]; client != nil {
			return client, nil
		}
	}
	var profile *bindingpb.CloudProfileRecord
	var err error
	if source.resolve != nil {
		profile, err = source.resolve(ctx, reference)
	} else {
		var response *bindingpb.PlatformResponse
		response, err = source.broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CloudProfileResolve{
			CloudProfileResolve: &bindingpb.CloudProfileResolveRequest{AccountProfileRef: reference},
		}})
		if err == nil {
			err = platformResponseError(response)
		}
		if err == nil {
			profile = response.GetCloudProfile()
		}
	}
	if err != nil {
		return nil, err
	}
	if profile == nil || profile.GetAccountProfileRef() != reference {
		return nil, fmt.Errorf("platform Cloud profile response does not match %q", reference)
	}
	client, err := cloudclient.NewClient(cloudclient.Config{
		ControllerAddress: profile.GetControllerAddress(), ControllerServerName: profile.GetControllerServerName(),
		ControllerCAPEM: append([]byte(nil), profile.GetControllerCaPem()...), BootID: source.bootID,
	})
	if err != nil {
		return nil, err
	}
	if source.cache != nil {
		source.cache.clients[reference] = client
	}
	return client, nil
}

func (source platformCloudProfiles) Available(ctx context.Context, reference string) bool {
	_, err := source.Resolve(ctx, reference)
	return err == nil
}

// platformCloudConnector 延迟解析当前本地 profile，然后把整个 Cloud attempt 交回 Go adapter。
type platformCloudConnector struct {
	profiles      platformCloudProfiles
	peers         cloudadapter.PeerFactory
	authorization peeradapter.Authorizer
	product       cloudv1.ClientProduct
	clientName    string
}

func (connector *platformCloudConnector) Connect(ctx context.Context, request clientruntime.AttemptRequest) (clientruntime.ReadyPeerSession, error) {
	if connector == nil {
		return nil, fmt.Errorf("Cloud connector is unavailable")
	}
	startedAt := time.Now()
	protocolClient, err := connector.profiles.Resolve(ctx, request.Route().AccountProfileRef)
	if err != nil {
		return nil, err
	}
	log.Printf("anytty cloud connect generation=%d stage=cloud_profile_resolve stage_ms=%d total_ms=%d", request.Stamp().Generation, time.Since(startedAt).Milliseconds(), time.Since(startedAt).Milliseconds())
	return (&cloudadapter.Dialer{
		Peers: connector.peers, Cloud: protocolClient, Authorization: connector.authorization,
		Product: connector.product, ClientName: connector.clientName,
	}).Connect(ctx, request)
}

func validCloudProduct(product cloudv1.ClientProduct) bool {
	return product >= cloudv1.ClientProduct_CLIENT_PRODUCT_TUI && product <= cloudv1.ClientProduct_CLIENT_PRODUCT_DESKTOP_GUI
}

type platformCredentials struct {
	broker *binding.PlatformBroker
	cache  *platformCredentialCache
}

type platformCredentialCache struct {
	mu          sync.Mutex
	credentials map[string]remoteauth.ClientAccessCredential
}

func newPlatformCredentials(broker *binding.PlatformBroker) platformCredentials {
	return platformCredentials{broker: broker, cache: &platformCredentialCache{credentials: make(map[string]remoteauth.ClientAccessCredential)}}
}

func (source platformCredentials) Available(ctx context.Context, endpointID, reference string) bool {
	_, err := source.ResolveClientCredential(ctx, endpointID, reference)
	return err == nil
}

func (source platformCredentials) ResolveClientCredential(ctx context.Context, endpointID, reference string) (remoteauth.ClientAccessCredential, error) {
	cacheKey := endpointID + "\x00" + reference
	if source.cache != nil {
		source.cache.mu.Lock()
		defer source.cache.mu.Unlock()
		if credential, ok := source.cache.credentials[cacheKey]; ok {
			return clonePlatformCredential(credential), nil
		}
	}
	response, err := source.broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CredentialResolve{
		CredentialResolve: &bindingpb.CredentialResolveRequest{EndpointId: endpointID, CredentialRef: reference},
	}})
	if err != nil {
		return remoteauth.ClientAccessCredential{}, err
	}
	record, err := platformCredential(response)
	if err != nil {
		return remoteauth.ClientAccessCredential{}, err
	}
	identity := remoteauth.ClientAccessIdentity{EndpointID: record.GetEndpointId(), Fingerprint: record.GetKeyFingerprint(), PublicKey: append(ed25519.PublicKey(nil), record.GetPublicKey()...)}
	if err := identity.ValidatePublic(); err != nil {
		return remoteauth.ClientAccessCredential{}, err
	}
	credential := remoteauth.ClientAccessCredential{Version: 3, EndpointID: record.GetEndpointId(), Identity: identity, CapabilityGrant: record.GetCapabilityGrant(), CloudRouteGrant: append([]byte(nil), record.GetCloudRouteGrant()...), CloudEdgeLocator: append([]byte(nil), record.GetCloudEdgeLocator()...), UpdatedAt: time.Now().UTC()}
	if source.cache != nil {
		source.cache.credentials[cacheKey] = credential
	}
	return clonePlatformCredential(credential), nil
}

func clonePlatformCredential(credential remoteauth.ClientAccessCredential) remoteauth.ClientAccessCredential {
	credential.Identity.PublicKey = append(ed25519.PublicKey(nil), credential.Identity.PublicKey...)
	credential.Identity.PrivateKey = nil
	credential.CloudRouteGrant = append([]byte(nil), credential.CloudRouteGrant...)
	credential.CloudEdgeLocator = append([]byte(nil), credential.CloudEdgeLocator...)
	return credential
}

func (source platformCredentials) UpdateCloudEdgeLocator(ctx context.Context, endpointID, reference string, locator []byte) error {
	credential, err := source.ResolveClientCredential(ctx, endpointID, reference)
	if err != nil {
		return err
	}
	response, err := source.broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CredentialBind{
		CredentialBind: &bindingpb.CredentialBindRequest{EndpointId: endpointID, CredentialRef: reference, CapabilityGrant: credential.CapabilityGrant, CloudRouteGrant: credential.CloudRouteGrant, CloudEdgeLocator: append([]byte(nil), locator...)},
	}})
	if err != nil {
		return err
	}
	record, err := platformCredential(response)
	if err != nil {
		return err
	}
	if record.GetEndpointId() != endpointID || record.GetCredentialRef() != reference || record.GetKeyFingerprint() != credential.Identity.Fingerprint || record.GetCapabilityGrant() != credential.CapabilityGrant || !bytes.Equal(record.GetCloudRouteGrant(), credential.CloudRouteGrant) || !bytes.Equal(record.GetCloudEdgeLocator(), locator) {
		return errors.New("platform secure store changed the Cloud credential while updating its Edge locator")
	}
	if source.cache != nil {
		source.cache.mu.Lock()
		credential.CloudEdgeLocator = append([]byte(nil), locator...)
		source.cache.credentials[endpointID+"\x00"+reference] = credential
		source.cache.mu.Unlock()
	}
	return nil
}

func (source platformCredentials) CloudAvailable(ctx context.Context, endpointID, reference string) bool {
	credential, err := source.ResolveClientCredential(ctx, endpointID, reference)
	return err == nil && credential.Ready() && len(credential.CloudRouteGrant) != 0
}

func (source platformCredentials) ResolveClientSigner(_ context.Context, endpointID, reference string, identity remoteauth.ClientAccessIdentity) (remoteauth.ClientAccessSigner, error) {
	if identity.EndpointID != endpointID {
		return nil, fmt.Errorf("platform signer endpoint mismatch")
	}
	return platformSigner{broker: source.broker, credentialRef: reference, identity: identity}, nil
}

type platformSigner struct {
	broker        *binding.PlatformBroker
	credentialRef string
	identity      remoteauth.ClientAccessIdentity
}

func (signer platformSigner) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	response, err := signer.broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_CredentialSign{
		CredentialSign: &bindingpb.CredentialSignRequest{CredentialRef: signer.credentialRef, Payload: append([]byte(nil), payload...)},
	}})
	if err != nil {
		return nil, err
	}
	if err := platformResponseError(response); err != nil {
		return nil, err
	}
	signature := append([]byte(nil), response.GetCredentialSign().GetSignature()...)
	if !ed25519.Verify(signer.identity.PublicKey, payload, signature) {
		return nil, fmt.Errorf("platform signer returned an invalid signature")
	}
	return signature, nil
}

func platformCredential(response *bindingpb.PlatformResponse) (*bindingpb.CredentialRecord, error) {
	if err := platformResponseError(response); err != nil {
		return nil, err
	}
	record := response.GetCredential()
	if record == nil || record.GetEndpointId() == "" || record.GetCredentialRef() == "" {
		return nil, fmt.Errorf("platform credential response is incomplete")
	}
	return proto.Clone(record).(*bindingpb.CredentialRecord), nil
}

func platformResponseError(response *bindingpb.PlatformResponse) error {
	if response == nil {
		return fmt.Errorf("platform response is empty")
	}
	if value := response.GetError(); value != nil {
		return &platformAPIError{value: proto.Clone(value).(*apipb.ApiError)}
	}
	return nil
}

// platformAPIError 保留 generated ApiError 的稳定 code/retryable 语义。
// 平台消息只用于诊断，Go route 控制流不得通过英文错误文本推断是否可重试。
type platformAPIError struct {
	value *apipb.ApiError
}

func (err *platformAPIError) Error() string {
	if err == nil || err.value == nil {
		return "platform request failed"
	}
	return fmt.Sprintf("platform request failed: %s", err.value.GetMessage())
}

func connectIntent(value bindingpb.ConnectIntent) (clientruntime.ConnectIntent, error) {
	switch value {
	case bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE:
		return clientruntime.ConnectIntentInteractive, nil
	case bindingpb.ConnectIntent_CONNECT_INTENT_BACKGROUND:
		return clientruntime.ConnectIntentBackground, nil
	case bindingpb.ConnectIntent_CONNECT_INTENT_PROBE:
		return clientruntime.ConnectIntentProbe, nil
	default:
		return "", fmt.Errorf("connect intent is unsupported")
	}
}

func decodeBootstrap(value string) ([]byte, error) {
	encoded := strings.TrimSpace(value)
	if strings.HasPrefix(encoded, remoteauth.PairingClaimCodePrefix) {
		return remoteauth.DecodePairingClaimCode(encoded)
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(payload) == 0 {
		return nil, fmt.Errorf("pairing bootstrap payload is invalid")
	}
	return payload, nil
}

func credentialRef(prefix, deviceID, fingerprint string) string {
	digest := sha256.Sum256([]byte(deviceID + "\n" + fingerprint))
	return prefix + base64.RawURLEncoding.EncodeToString(digest[:])
}

var _ binding.Host = (*Host)(nil)
var _ binding.PairingHost = (*Host)(nil)
var _ binding.CredentialHost = (*Host)(nil)
var _ binding.EndpointRegistryHost = (*Host)(nil)
var _ binding.ConnectionPolicyHost = (*Host)(nil)
var _ binding.SessionInvalidationHost = (*Host)(nil)
var _ binding.EndpointShareHost = (*Host)(nil)
var _ peeradapter.CredentialSource = platformCredentials{}
var _ peeradapter.SignerSource = platformCredentials{}
var _ remoteauth.ClientAccessSigner = platformSigner{}
