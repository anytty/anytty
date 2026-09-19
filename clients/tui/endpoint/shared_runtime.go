package endpoint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cloudadapter "github.com/anytty/anytty/access/engine/adapter/cloud"
	directadapter "github.com/anytty/anytty/access/engine/adapter/direct"
	localadapter "github.com/anytty/anytty/access/engine/adapter/local"
	peeradapter "github.com/anytty/anytty/access/engine/adapter/peer"
	sshadapter "github.com/anytty/anytty/access/engine/adapter/ssh"
	systemadapter "github.com/anytty/anytty/access/engine/adapter/system"
	pionadapter "github.com/anytty/anytty/access/engine/adapter/webrtc/pion"
	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	cloudclient "github.com/anytty/anytty/access/transport/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/userdirs"
	"github.com/google/uuid"
)

// tuiClientName is the Hello client identity of the TUI host. The shared
// client layer owns the connection; the name is diagnostics only.
const tuiClientName = "anytty-tui"

// fixedPlanSource serves one already-resolved endpoint plan. tui2 dials one
// Config at a time; route selection inside the endpoint still runs the shared
// planner race through ClientRuntime.
type fixedPlanSource struct {
	snapshot clientruntime.EndpointPlanSnapshot
}

func (s fixedPlanSource) Snapshot(_ context.Context, endpointID clientendpoint.EndpointID) (clientruntime.EndpointPlanSnapshot, error) {
	if s.snapshot.Endpoint.ID != endpointID {
		return clientruntime.EndpointPlanSnapshot{}, &clientruntime.Error{
			Code: clientruntime.ErrorNotFound, Message: fmt.Sprintf("endpoint %q is not configured", endpointID),
		}
	}
	return s.snapshot, nil
}

// tuiCredentialSource resolves endpoint-bound grants from the same owner-only
// store the CLI pairing commands write. Read-only: tui2 never pairs or binds.
type tuiCredentialSource struct {
	store *remoteauth.CredentialStore
}

func (source tuiCredentialSource) ResolveClientCredential(ctx context.Context, endpointID, reference string) (remoteauth.ClientAccessCredential, error) {
	credential, err := source.store.ResolveContext(ctx, reference)
	if err != nil {
		return remoteauth.ClientAccessCredential{}, err
	}
	if credential.EndpointID != endpointID {
		return remoteauth.ClientAccessCredential{}, fmt.Errorf("credential endpoint does not match route endpoint")
	}
	return credential, nil
}

func (source tuiCredentialSource) ResolveClientSigner(_ context.Context, endpointID, _ string, identity remoteauth.ClientAccessIdentity) (remoteauth.ClientAccessSigner, error) {
	if identity.EndpointID != endpointID {
		return nil, fmt.Errorf("credential signer endpoint does not match route endpoint")
	}
	return remoteauth.NewPrivateClientAccessSigner(identity)
}

func (source tuiCredentialSource) UpdateCloudEdgeLocator(ctx context.Context, endpointID, reference string, locator []byte) error {
	return source.store.UpdateCloudEdgeLocator(ctx, reference, endpointID, locator)
}

func (source tuiCredentialSource) Available(ctx context.Context, endpointID, reference string) bool {
	credential, err := source.store.ResolveContext(ctx, strings.TrimSpace(reference))
	return err == nil && credential.EndpointID == endpointID && credential.Ready()
}

func (source tuiCredentialSource) CloudAvailable(ctx context.Context, endpointID, reference string) bool {
	credential, err := source.store.ResolveContext(ctx, strings.TrimSpace(reference))
	return err == nil && credential.EndpointID == endpointID && credential.Ready() && len(credential.CloudRouteGrant) != 0
}

// tuiCredentialDir mirrors the CLI credential store path (owner-only).
func tuiCredentialDir() string {
	return filepath.Join(userdirs.StateHome(), "anytty", "remote-v2", "credentials")
}

// newSharedEndpointRuntime assembles the shared connection stack for one
// endpoint: registry/plan source, route connectors and the SessionOwner that
// is the single generation truth (client/runtime). No route is dialed here.
// cloudProtocol nil means the managed Cloud connector is not available and
// managed routes stay out of the plan (readable no-route error).
func newSharedEndpointRuntime(target clientendpoint.Endpoint, environment clientruntime.RoutePlanEnvironment, cloudProtocol *cloudclient.Client) (*clientruntime.SessionOwner, *clientruntime.ClientRuntime, error) {
	credentials := tuiCredentialSource{store: remoteauth.NewCredentialStore(tuiCredentialDir())}
	peers := pionadapter.Factory{}
	connectors := map[clientendpoint.RouteKind]clientruntime.PeerConnector{
		clientendpoint.RouteLocalUnix: localadapter.NewDialer(localadapter.Options{ClientName: tuiClientName}),
		clientendpoint.RouteDirectWebRTCTCP: &directadapter.Dialer{
			Peers: peers, Authorization: peeradapter.CapabilityAuthorizer{Credentials: credentials},
			ClientName: tuiClientName,
		},
		clientendpoint.RouteSSHWebRTCTCP: sshadapter.NewDialer(sshadapter.Options{
			Peers: peers, Authorization: peeradapter.CapabilityAuthorizer{Credentials: credentials},
			Credentials: sshadapter.AgentCredentialSource{}, ClientName: tuiClientName,
		}),
	}
	if cloudProtocol != nil {
		connectors[clientendpoint.RouteManagedWebRTC] = &cloudadapter.Dialer{
			Peers: peers, Cloud: cloudProtocol,
			Authorization: peeradapter.CapabilityAuthorizer{Credentials: credentials},
			Product:       cloudv1.ClientProduct_CLIENT_PRODUCT_TUI, ClientName: tuiClientName,
		}
	}
	dialers, err := clientruntime.NewPeerConnectorMap(connectors)
	if err != nil {
		return nil, nil, err
	}
	owner := clientruntime.NewSessionOwner()
	runtime, err := clientruntime.NewClientRuntime(owner, fixedPlanSource{snapshot: clientruntime.EndpointPlanSnapshot{
		Endpoint: target, Environment: environment, ConfigKey: sharedConfigKey(target, environment),
	}}, systemadapter.Clock{}, dialers)
	if err != nil {
		_ = owner.Close()
		return nil, nil, err
	}
	return owner, runtime, nil
}

// sharedPlanSnapshot resolves the endpoint target and planner environment.
// The shared endpoints.yaml registries are the source of truth written by the
// CLI: a same-ID entry wins over the tui2.json compatibility fields, reading
// ordered registry paths (explicit TUI2_ENDPOINTS/-endpoints first, then the
// default path) with the first file that defines the ID owning it. A missing
// registry entry falls back to the Config projection so legacy tui2.json
// daemon endpoints keep working (migration path).
func sharedPlanSnapshot(ctx context.Context, registryPaths []string, cfg Config, bridgeSocket string, cloudAvailable bool, routes []clientendpoint.RouteKind) (clientruntime.EndpointPlanSnapshot, error) {
	target, ok, err := sharedRegistryEndpoint(registryPaths, cfg.Name)
	if err != nil {
		return clientruntime.EndpointPlanSnapshot{}, err
	}
	if !ok {
		target, err = endpointFromConfig(cfg, bridgeSocket)
		if err != nil {
			return clientruntime.EndpointPlanSnapshot{}, err
		}
	}
	environment := sharedRouteEnvironment(ctx, target, routes, cloudAvailable)
	return clientruntime.EndpointPlanSnapshot{Endpoint: target, Environment: environment, ConfigKey: sharedConfigKey(target, environment)}, nil
}

// sharedRegistryEndpoint locates one endpoint across the ordered registry
// paths and validates it. Missing files are skipped (the tui2.json fallback
// stays available); a corrupt file is reported only when no later registry
// defines the endpoint.
func sharedRegistryEndpoint(paths []string, name string) (clientendpoint.Endpoint, bool, error) {
	target, ok, err := sharedEndpointIn(paths, name)
	if err != nil || !ok {
		return clientendpoint.Endpoint{}, false, err
	}
	if err := target.Validate(); err != nil {
		return clientendpoint.Endpoint{}, false, err
	}
	return target, true, nil
}

// endpointFromConfig synthesizes the shared Endpoint view of one tui2 Config.
// It is only used for endpoints the CLI has not written to the registry yet
// (legacy tui2.json migration). direct/managed routes still require the pin
// and locators the Config carries; missing fields fail with a readable error.
func endpointFromConfig(cfg Config, bridgeSocket string) (clientendpoint.Endpoint, error) {
	label := strings.TrimSpace(cfg.Name)
	target := clientendpoint.Endpoint{
		ID: clientendpoint.EndpointID(strings.TrimSpace(cfg.Name)), Label: label, LabelSource: clientendpoint.SourceUser,
		ConnectMode:    clientendpoint.ConnectAuto,
		Enabled:        true,
		DaemonIdentity: clientendpoint.DaemonIdentity{DeviceID: cfg.DaemonDeviceID, DeviceFingerprint: cfg.DaemonFingerprint},
		Routes:         map[clientendpoint.RouteID]clientendpoint.AccessRoute{},
	}
	source := clientendpoint.EndpointSource(clientendpoint.SourceManual)
	newRoute := func(id, kind string) clientendpoint.AccessRoute {
		return clientendpoint.AccessRoute{
			ID: clientendpoint.RouteID(id), Kind: clientendpoint.RouteKind(kind), Enabled: true,
			Source: source, PolicySource: clientendpoint.SourceUser,
		}
	}
	switch cfg.ConnectModeName() {
	case ConnectLocalUnix:
		route := newRoute("local", string(clientendpoint.RouteLocalUnix))
		route.Socket = strings.TrimSpace(cfg.Socket)
		target.Routes[route.ID] = route
	case ConnectDirectTCP:
		// The tcp compatibility mode terminates at the daemon framed
		// transport. tui2 owns only a byte-transparent relay to a local unix
		// socket; the connection itself still goes through the shared local
		// route adapter (see tcp_bridge.go).
		route := newRoute("local", string(clientendpoint.RouteLocalUnix))
		route.Socket = strings.TrimSpace(bridgeSocket)
		target.Routes[route.ID] = route
	case ConnectDirectWebRTC:
		route := newRoute("direct", string(clientendpoint.RouteDirectWebRTCTCP))
		route.SignalingAddresses = append([]string(nil), trimLocators(cfg.SignalingAddresses)...)
		route.ICETCPAddresses = append([]string(nil), trimLocators(cfg.ICETCPAddresses)...)
		route.CredentialRef = strings.TrimSpace(cfg.CredentialRef)
		target.Routes[route.ID] = route
	case ConnectManagedWebRTC:
		route := newRoute("cloud", string(clientendpoint.RouteManagedWebRTC))
		route.TargetDeviceID = strings.TrimSpace(cfg.DaemonDeviceID)
		route.CredentialRef = strings.TrimSpace(cfg.CredentialRef)
		route.RelayMode = clientendpoint.RelayAuto
		route.RelayTransport = clientendpoint.RelayTransportTCP
		target.Routes[route.ID] = route
	case ConnectSSHWebRTC:
		return clientendpoint.Endpoint{}, fmt.Errorf("endpoint %q: connect_mode ssh-webrtc-tcp is configured by the CLI registry (anytty endpoint add ssh-webrtc-tcp ...) and dialed by the shared SSH adapter; tui2.json cannot carry its tunnel parameters", cfg.Name)
	default:
		return clientendpoint.Endpoint{}, fmt.Errorf("endpoint %q: connect_mode %q has no shared route mapping; pair it with the CLI (anytty endpoint add ...)", cfg.Name, cfg.ConnectModeName())
	}
	if err := target.Validate(); err != nil {
		return clientendpoint.Endpoint{}, err
	}
	return target, nil
}

// sharedRouteEnvironment reports the route kinds the TUI may dial for one
// target plus the credential refs currently present in the shared store. The
// planner uses it to filter routes; a missing credential produces a readable
// no-route error instead of an opaque dial failure.
//
// The enabled policy is the operator's route list (default: local-unix plus
// ssh-webrtc-tcp). local-unix is always dialable; ssh-webrtc-tcp needs an
// available stored credential; direct/managed WebRTC are opt-in and managed
// additionally needs the Cloud client. An old registry that carries webrtc or
// cloud routes therefore cannot stall the picker with an unconfigured dial.
func sharedRouteEnvironment(ctx context.Context, target clientendpoint.Endpoint, routes []clientendpoint.RouteKind, cloudAvailable bool) clientruntime.RoutePlanEnvironment {
	routes = normalizeRouteKinds(routes)
	credentials := tuiCredentialSource{store: remoteauth.NewCredentialStore(tuiCredentialDir())}
	supported := make([]clientendpoint.RouteKind, 0, len(routes))
	addKind := func(kind clientendpoint.RouteKind) {
		if !routeKindEnabled(supported, kind) {
			supported = append(supported, kind)
		}
	}
	if routeKindEnabled(routes, clientendpoint.RouteLocalUnix) {
		addKind(clientendpoint.RouteLocalUnix)
	}
	if routeKindEnabled(routes, clientendpoint.RouteSSHWebRTCTCP) && hasAvailableCredential(ctx, target, clientendpoint.RouteSSHWebRTCTCP, credentials) {
		addKind(clientendpoint.RouteSSHWebRTCTCP)
	}
	if routeKindEnabled(routes, clientendpoint.RouteDirectWebRTCTCP) {
		addKind(clientendpoint.RouteDirectWebRTCTCP)
	}
	if routeKindEnabled(routes, clientendpoint.RouteManagedWebRTC) && cloudAvailable && hasAvailableCredential(ctx, target, clientendpoint.RouteManagedWebRTC, credentials) {
		addKind(clientendpoint.RouteManagedWebRTC)
	}
	environment := clientruntime.RoutePlanEnvironment{SupportedRouteKinds: supported}
	for _, route := range target.RouteList() {
		if !routeKindEnabled(supported, route.Kind) {
			continue
		}
		reference := strings.TrimSpace(route.CredentialRef)
		switch route.Kind {
		case clientendpoint.RouteDirectWebRTCTCP:
			if credentials.Available(ctx, string(target.ID), reference) {
				environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
			}
		case clientendpoint.RouteSSHWebRTCTCP:
			if credentials.Available(ctx, string(target.ID), reference) {
				environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
				if sshRef := strings.TrimSpace(route.SSHCredentialRef); sshRef != "" && credentials.Available(ctx, string(target.ID), sshRef) {
					environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, sshRef)
				}
			}
		case clientendpoint.RouteManagedWebRTC:
			if cloudAvailable && credentials.CloudAvailable(ctx, string(target.ID), reference) {
				environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
			}
		}
	}
	return environment
}

// hasAvailableCredential reports whether the target has an enabled route of
// the given credential-bearing kind whose stored credential resolves.
func hasAvailableCredential(ctx context.Context, target clientendpoint.Endpoint, kind clientendpoint.RouteKind, credentials tuiCredentialSource) bool {
	for _, route := range target.RouteList() {
		if route.Kind != kind || !route.Enabled {
			continue
		}
		if credentials.Available(ctx, string(target.ID), strings.TrimSpace(route.CredentialRef)) {
			return true
		}
		if kind == clientendpoint.RouteSSHWebRTCTCP {
			if sshRef := strings.TrimSpace(route.SSHCredentialRef); sshRef != "" && credentials.Available(ctx, string(target.ID), sshRef) {
				return true
			}
		}
	}
	return false
}

// Cloud controller defaults mirror the CLI composition; the environment
// variables allow an on-prem/test controller.
const (
	defaultCloudControllerAddress    = "cloud.anytty.com:443"
	defaultCloudControllerServerName = "cloud.anytty.com"
)

// tuiCloudClient builds the shared Cloud client for managed-webrtc routes.
// nil/error means the managed connector is unavailable (the endpoint still
// lists, and the planner reports a readable no-route error).
func tuiCloudClient() (*cloudclient.Client, error) {
	address := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_ADDRESS"))
	serverName := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_SERVER_NAME"))
	caFile := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_CA"))
	if address == "" && serverName == "" && caFile == "" {
		address = defaultCloudControllerAddress
		serverName = defaultCloudControllerServerName
	}
	if address == "" || serverName == "" {
		return nil, fmt.Errorf("ANYTTY_CLOUD_CONTROLLER_ADDRESS and ANYTTY_CLOUD_CONTROLLER_SERVER_NAME must be configured together")
	}
	var caPEM []byte
	if caFile != "" {
		contents, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read AnyTTY Cloud Controller CA: %w", err)
		}
		caPEM = contents
	}
	return cloudclient.NewClient(cloudclient.Config{
		ControllerAddress: address, ControllerServerName: serverName, ControllerCAPEM: caPEM, BootID: uuid.NewString(),
	})
}

func sharedConfigKey(target clientendpoint.Endpoint, environment clientruntime.RoutePlanEnvironment) string {
	payload, err := json.Marshal(struct {
		Endpoint    clientendpoint.Endpoint            `json:"endpoint"`
		Environment clientruntime.RoutePlanEnvironment `json:"environment"`
	}{target, environment})
	if err != nil {
		return fmt.Sprintf("tui2-%s-%d", target.ID, len(target.Routes))
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
