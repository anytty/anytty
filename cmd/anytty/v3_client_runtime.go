package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	cloudadapter "github.com/anytty/anytty/client/adapter/cloud"
	directadapter "github.com/anytty/anytty/client/adapter/direct"
	localadapter "github.com/anytty/anytty/client/adapter/local"
	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	sshadapter "github.com/anytty/anytty/client/adapter/ssh"
	systemadapter "github.com/anytty/anytty/client/adapter/system"
	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/google/uuid"
)

type cliEndpointPlanSource struct {
	registryPath   string
	localOptions   localadapter.Options
	credentials    cliCredentialSource
	sshCredentials sshadapter.AgentCredentialSource
	initialTarget  clientendpoint.Endpoint
	cloudAvailable bool
}

const (
	cliEndpointClientName = "anytty-cli"
	tuiEndpointClientName = "anytty-tui"
)

// Snapshot 每次从共享 Endpoint registry 读取当前配置，再生成本机平台与 credential 索引。
// 当前 Ready session 仍由 SessionOwner 持有；registry priority 变更只会在下一次 EnsureSession 时生效。
func (source cliEndpointPlanSource) Snapshot(ctx context.Context, endpointID clientendpoint.EndpointID) (clientruntime.EndpointPlanSnapshot, error) {
	registry, err := clientendpoint.Load(source.registryPath)
	if err != nil {
		return clientruntime.EndpointPlanSnapshot{}, err
	}
	target, ok := registry.Endpoints[endpointID]
	if !ok {
		// pairing import 在 registry 事务提交前必须先对候选 Endpoint 完成 daemon-authenticated handshake。
		// 该显式 invocation 输入只覆盖“同 ID 尚不存在”；已持久化 Endpoint 永远以 registry 最新值为准。
		if source.initialTarget.ID != endpointID {
			return clientruntime.EndpointPlanSnapshot{}, &clientruntime.Error{Code: clientruntime.ErrorNotFound, Message: fmt.Sprintf("endpoint %q is not configured", endpointID)}
		}
		target = source.initialTarget
	}
	environment := cliRoutePlanEnvironment(ctx, target, source.credentials, source.sshCredentials, source.cloudAvailable)
	configKey, err := cliEndpointConfigKey(target, source.localOptions, environment)
	if err != nil {
		return clientruntime.EndpointPlanSnapshot{}, err
	}
	return clientruntime.EndpointPlanSnapshot{Endpoint: target, Environment: environment, ConfigKey: configKey}, nil
}

type cliCredentialSource struct {
	store *remoteauth.CredentialStore
}

// ResolveClientCredential 从 desktop owner-only store 读取 endpoint-bound grant 与私钥，不访问 Cloud 或 registry fallback。
func (source cliCredentialSource) ResolveClientCredential(ctx context.Context, endpointID, reference string) (remoteauth.ClientAccessCredential, error) {
	credential, err := source.store.ResolveContext(ctx, reference)
	if err != nil {
		return remoteauth.ClientAccessCredential{}, err
	}
	if credential.EndpointID != endpointID {
		return remoteauth.ClientAccessCredential{}, fmt.Errorf("credential endpoint does not match route endpoint")
	}
	return credential, nil
}

// ResolveClientSigner 从已验证 credential 构造进程内 signer；identity 不匹配 Endpoint 时禁止签名。
func (source cliCredentialSource) ResolveClientSigner(_ context.Context, endpointID, _ string, identity remoteauth.ClientAccessIdentity) (remoteauth.ClientAccessSigner, error) {
	if identity.EndpointID != endpointID {
		return nil, fmt.Errorf("credential signer endpoint does not match route endpoint")
	}
	return remoteauth.NewPrivateClientAccessSigner(identity)
}

func (source cliCredentialSource) UpdateCloudEdgeLocator(ctx context.Context, endpointID, reference string, locator []byte) error {
	return source.store.UpdateCloudEdgeLocator(ctx, reference, endpointID, locator)
}

func (source cliCredentialSource) Available(ctx context.Context, endpointID, reference string) bool {
	credential, err := source.store.ResolveContext(ctx, strings.TrimSpace(reference))
	return err == nil && credential.EndpointID == endpointID && credential.Ready()
}

func (source cliCredentialSource) CloudAvailable(ctx context.Context, endpointID, reference string) bool {
	credential, err := source.store.ResolveContext(ctx, strings.TrimSpace(reference))
	return err == nil && credential.EndpointID == endpointID && credential.Ready() && len(credential.CloudRouteGrant) != 0
}

func connectCLIEndpointApplication(ctx context.Context, owner *clientruntime.SessionOwner, target clientendpoint.Endpoint, requested clientendpoint.RouteID, intent clientruntime.ConnectIntent, localOptions localadapter.Options, logger *slog.Logger) (*protocoladapter.ApplicationClient, clientendpoint.AccessRoute, error) {
	runtime, err := newCLIEndpointRuntime(ctx, owner, target, localOptions, logger)
	if err != nil {
		return nil, clientendpoint.AccessRoute{}, err
	}
	return connectWithCLIEndpointRuntime(ctx, owner, runtime, target, requested, intent)
}

func connectCLIEndpointApplicationWithRegistry(ctx context.Context, owner *clientruntime.SessionOwner, target clientendpoint.Endpoint, requested clientendpoint.RouteID, intent clientruntime.ConnectIntent, registryPath string, localOptions localadapter.Options, logger *slog.Logger) (*protocoladapter.ApplicationClient, clientendpoint.AccessRoute, error) {
	runtime, err := newCLIEndpointRuntimeWithRegistry(ctx, owner, target, registryPath, localOptions, logger)
	if err != nil {
		return nil, clientendpoint.AccessRoute{}, err
	}
	return connectWithCLIEndpointRuntime(ctx, owner, runtime, target, requested, intent)
}

func connectWithCLIEndpointRuntime(ctx context.Context, owner *clientruntime.SessionOwner, runtime *clientruntime.ClientRuntime, target clientendpoint.Endpoint, requested clientendpoint.RouteID, intent clientruntime.ConnectIntent) (*protocoladapter.ApplicationClient, clientendpoint.AccessRoute, error) {
	lease, err := runtime.EnsureSession(ctx, clientruntime.ConnectRequest{EndpointID: target.ID, RouteOverride: requested, Intent: intent})
	if err != nil {
		return nil, clientendpoint.AccessRoute{}, err
	}
	ready, err := owner.ApplicationSession(lease)
	if err != nil {
		return nil, clientendpoint.AccessRoute{}, err
	}
	client, err := protocoladapter.NewRuntimeApplicationClient(ready, runtime)
	if err != nil {
		return nil, clientendpoint.AccessRoute{}, err
	}
	route, ok := target.Route(lease.Stamp.RouteID)
	if !ok {
		_ = client.Close()
		return nil, clientendpoint.AccessRoute{}, fmt.Errorf("runtime winner route %q is absent from endpoint %q", lease.Stamp.RouteID, target.ID)
	}
	return client, route, nil
}

func newCLIEndpointRuntime(ctx context.Context, owner *clientruntime.SessionOwner, target clientendpoint.Endpoint, localOptions localadapter.Options, logger *slog.Logger) (*clientruntime.ClientRuntime, error) {
	return newCLIEndpointRuntimeWithRegistry(ctx, owner, target, "", localOptions, logger)
}

func newCLIEndpointRuntimeWithRegistry(ctx context.Context, owner *clientruntime.SessionOwner, target clientendpoint.Endpoint, registryPath string, localOptions localadapter.Options, logger *slog.Logger) (*clientruntime.ClientRuntime, error) {
	if ctx == nil || owner == nil {
		return nil, fmt.Errorf("CLI endpoint runtime requires context and a session owner")
	}
	if err := target.Validate(); err != nil {
		return nil, err
	}
	clientName := endpointRuntimeClientName(localOptions.ClientName)
	cloudProduct := endpointRuntimeCloudProduct(clientName)
	credentials := cliCredentialSource{store: remoteauth.NewCredentialStore(v3RemoteCredentialDir())}
	sshCredentials := sshadapter.AgentCredentialSource{}
	localDialer := localadapter.NewDialer(localOptions)
	peers := pionadapter.Factory{Logger: logger}
	connectors := map[clientendpoint.RouteKind]clientruntime.PeerConnector{
		clientendpoint.RouteLocalUnix: localDialer,
		clientendpoint.RouteDirectWebRTCTCP: &directadapter.Dialer{
			Peers: peers, Authorization: peeradapter.CapabilityAuthorizer{Credentials: credentials},
			ClientName: clientName, Now: time.Now,
		},
		clientendpoint.RouteSSHWebRTCTCP: sshadapter.NewDialer(sshadapter.Options{
			Peers: peers, Authorization: peeradapter.CapabilityAuthorizer{Credentials: credentials},
			Credentials: sshCredentials, ClientName: clientName,
		}),
	}
	cloudProtocol, err := cliCloudClientFromEnvironment(uuid.NewString())
	if err != nil {
		return nil, err
	}
	if cloudProtocol != nil {
		connectors[clientendpoint.RouteManagedWebRTC] = &cloudadapter.Dialer{Peers: peers, Cloud: cloudProtocol, Authorization: peeradapter.CapabilityAuthorizer{Credentials: credentials}, Product: cloudProduct, ClientName: clientName}
	}
	dialers, err := clientruntime.NewPeerConnectorMap(connectors)
	if err != nil {
		return nil, err
	}
	runtime, err := clientruntime.NewClientRuntime(owner, cliEndpointPlanSource{
		registryPath: registryPath, localOptions: localOptions, credentials: credentials, sshCredentials: sshCredentials, initialTarget: target, cloudAvailable: cloudProtocol != nil,
	}, systemadapter.Clock{}, dialers)
	if err != nil {
		return nil, err
	}
	return runtime, nil
}

func endpointRuntimeClientName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return cliEndpointClientName
	}
	return value
}

func endpointRuntimeCloudProduct(clientName string) cloudv1.ClientProduct {
	if endpointRuntimeClientName(clientName) == tuiEndpointClientName {
		return cloudv1.ClientProduct_CLIENT_PRODUCT_TUI
	}
	return cloudv1.ClientProduct_CLIENT_PRODUCT_CLI
}

type cliSSHCredentialAvailability interface {
	Available(string) bool
}

type cliCapabilityAvailability interface {
	Available(context.Context, string, string) bool
}

type cliCloudCredentialAvailability interface {
	CloudAvailable(context.Context, string, string) bool
}

const (
	defaultCloudControllerAddress    = "cloud.anytty.com:443"
	defaultCloudControllerServerName = "cloud.anytty.com"
)

func cliRoutePlanEnvironment(ctx context.Context, target clientendpoint.Endpoint, credentials cliCapabilityAvailability, sshCredentials cliSSHCredentialAvailability, cloudEnabled ...bool) clientruntime.RoutePlanEnvironment {
	cloudAvailable := len(cloudEnabled) != 0 && cloudEnabled[0]
	environment := clientruntime.RoutePlanEnvironment{SupportedRouteKinds: []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix, clientendpoint.RouteDirectWebRTCTCP, clientendpoint.RouteSSHWebRTCTCP,
	}}
	if cloudAvailable {
		environment.SupportedRouteKinds = append(environment.SupportedRouteKinds, clientendpoint.RouteManagedWebRTC)
	}
	for _, route := range target.RouteList() {
		reference := strings.TrimSpace(route.CredentialRef)
		switch route.Kind {
		case clientendpoint.RouteDirectWebRTCTCP:
			if credentials != nil && credentials.Available(ctx, string(target.ID), reference) {
				environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
			}
		case clientendpoint.RouteManagedWebRTC:
			if cloudCredentials, ok := credentials.(cliCloudCredentialAvailability); cloudAvailable && ok {
				if cloudCredentials.CloudAvailable(ctx, string(target.ID), reference) {
					environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
				}
			}
		case clientendpoint.RouteSSHWebRTCTCP:
			if credentials != nil && credentials.Available(ctx, string(target.ID), reference) {
				environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, reference)
				if sshCredentials != nil && sshCredentials.Available(route.SSHCredentialRef) {
					environment.AvailableCredentialRefs = append(environment.AvailableCredentialRefs, route.SSHCredentialRef)
				}
			}
		}
	}
	return environment
}

func cliCloudClientFromEnvironment(bootID string) (*cloudclient.Client, error) {
	endpoint, err := cliCloudControllerEndpointFromEnvironment()
	if err != nil {
		return nil, err
	}
	return cloudclient.NewClient(cloudclient.Config{ControllerAddress: endpoint.address, ControllerServerName: endpoint.serverName, ControllerCAPEM: endpoint.caPEM, BootID: bootID})
}

type cliCloudControllerEndpoint struct {
	address    string
	serverName string
	caPEM      []byte
}

func cliCloudControllerEndpointFromEnvironment() (cliCloudControllerEndpoint, error) {
	address := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_ADDRESS"))
	serverName := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_SERVER_NAME"))
	caFile := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_CA"))
	if address == "" && serverName == "" && caFile == "" {
		address = defaultCloudControllerAddress
		serverName = defaultCloudControllerServerName
	}
	if address == "" || serverName == "" {
		return cliCloudControllerEndpoint{}, fmt.Errorf("ANYTTY_CLOUD_CONTROLLER_ADDRESS and ANYTTY_CLOUD_CONTROLLER_SERVER_NAME must be configured together")
	}
	var caPEM []byte
	if caFile != "" {
		var err error
		caPEM, err = os.ReadFile(caFile)
		if err != nil {
			return cliCloudControllerEndpoint{}, fmt.Errorf("read AnyTTY Cloud Controller CA: %w", err)
		}
	}
	return cliCloudControllerEndpoint{address: address, serverName: serverName, caPEM: caPEM}, nil
}

type v3PairingRaceResult struct {
	index  int
	paired remoteauth.PairingExchangeResult
	err    error
}

func redeemV3RemotePairing(ctx context.Context, endpointID clientendpoint.EndpointID, credentialRef string, candidate clientendpoint.EndpointCandidate, request remoteauth.ClientPairingRequest) (remoteauth.PairingExchangeResult, error) {
	cloudProtocol, cloudInitErr := cliCloudClientFromEnvironment(uuid.NewString())
	sshCredentials := sshadapter.AgentCredentialSource{}
	routes := make(map[clientendpoint.RouteID]clientendpoint.AccessRoute)
	for _, route := range candidate.Routes {
		if !route.Enabled {
			continue
		}
		switch route.Kind {
		case clientendpoint.RouteDirectWebRTCTCP:
		case clientendpoint.RouteSSHWebRTCTCP:
			if route.SSHCredentialRef == "" && route.CredentialDescriptor != nil {
				route.SSHCredentialRef = route.CredentialDescriptor.DescriptorID
			}
		case clientendpoint.RouteManagedWebRTC:
			if cloudProtocol == nil && cloudInitErr == nil {
				continue
			}
		default:
			continue
		}
		route.CredentialRef = credentialRef
		routes[route.ID] = route
	}
	if len(routes) == 0 {
		return remoteauth.PairingExchangeResult{}, errors.New("pairing claim has no Route supported by this CLI")
	}
	target := clientendpoint.Endpoint{ID: endpointID, DaemonIdentity: candidate.Identity, Routes: routes}
	routeIDs := make([]clientendpoint.RouteID, 0, len(routes))
	for _, route := range candidate.Routes {
		if _, ok := routes[route.ID]; ok {
			routeIDs = append(routeIDs, route.ID)
		}
	}
	owner := clientruntime.NewSessionOwner()
	defer owner.Close()
	attempts, err := owner.BeginRouteAttempts(target, routeIDs, clientruntime.ConnectIntentInteractive)
	if err != nil {
		return remoteauth.PairingExchangeResult{}, err
	}
	raceContext, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan v3PairingRaceResult, len(attempts))
	for index, attempt := range attempts {
		index, attempt := index, attempt
		go func() {
			var paired remoteauth.PairingExchangeResult
			var attemptErr error
			switch attempt.Route().Kind {
			case clientendpoint.RouteDirectWebRTCTCP:
				paired, attemptErr = (&directadapter.PairingConnector{Peers: pionadapter.Factory{}, Now: time.Now}).Redeem(raceContext, attempt, request)
			case clientendpoint.RouteSSHWebRTCTCP:
				paired, attemptErr = (&sshadapter.PairingConnector{Peers: pionadapter.Factory{}, Credentials: sshCredentials, Now: time.Now}).Redeem(raceContext, attempt, request)
			case clientendpoint.RouteManagedWebRTC:
				if cloudInitErr != nil {
					attemptErr = cloudInitErr
					break
				}
				paired, attemptErr = (&cloudadapter.PairingConnector{Peers: pionadapter.Factory{}, Cloud: cloudProtocol, Product: cloudv1.ClientProduct_CLIENT_PRODUCT_CLI, Now: time.Now}).Redeem(raceContext, attempt, request)
			default:
				attemptErr = fmt.Errorf("pairing Route %q is unsupported", attempt.Route().Kind)
			}
			results <- v3PairingRaceResult{index: index, paired: paired, err: attemptErr}
		}()
	}
	failures := make([]error, len(attempts))
	for range attempts {
		result := <-results
		if result.err == nil {
			cancel()
			return result.paired, nil
		}
		failures[result.index] = result.err
	}
	return remoteauth.PairingExchangeResult{}, clientruntime.NewAllRoutesUnavailableError(attempts, failures)
}

func cliEndpointConfigKey(target clientendpoint.Endpoint, localOptions localadapter.Options, environment clientruntime.RoutePlanEnvironment) (string, error) {
	payload, err := json.Marshal(struct {
		Endpoint       clientendpoint.Endpoint            `json:"endpoint"`
		SocketOverride string                             `json:"socket_override"`
		DefaultSocket  string                             `json:"default_socket"`
		ClientName     string                             `json:"client_name"`
		ReadyTimeout   time.Duration                      `json:"ready_timeout"`
		RetryInterval  time.Duration                      `json:"retry_interval"`
		Environment    clientruntime.RoutePlanEnvironment `json:"environment"`
	}{target, localOptions.SocketOverride, localOptions.DefaultSocket, localOptions.ClientName, localOptions.ReadyTimeout, localOptions.RetryInterval, environment})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
