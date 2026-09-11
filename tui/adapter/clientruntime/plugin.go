package clientruntimeadapter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"google.golang.org/protobuf/proto"
)

type PluginLaunch struct {
	ID  string
	Run func(context.Context, sdk.RoutedExecutor, []state.EndpointID) error
	// RunWithExit reports each supervised child exit before any restart.
	RunWithExit func(context.Context, sdk.RoutedExecutor, []state.EndpointID, func(error)) error
}
type PluginOptions struct {
	TUIInstanceID string
	Endpoints     []state.EndpointID
	Plugins       []PluginLaunch
}
type pluginRoute struct {
	client  *sdk.Client
	address *apipb.PluginAddress
	execute sdk.Executor
	alive   func() bool
}
type pluginPeer struct {
	address *apipb.PluginAddress
	lease   []byte
	execute sdk.Executor
}

type PluginService struct {
	router          *EndpointApplicationRouter
	options         PluginOptions
	mu              sync.RWMutex
	started         bool
	hosts           map[string]pluginRoute
	peers           map[string]pluginPeer
	daemons         map[string]state.EndpointID
	aliases         map[state.EndpointID]state.EndpointID
	claims          map[string]string
	endpointDaemons map[state.EndpointID]string
}

func NewPluginService(router *EndpointApplicationRouter, options PluginOptions) *PluginService {
	endpoints := make([]state.EndpointID, 0, len(options.Endpoints))
	seenEndpoints := map[state.EndpointID]bool{}
	for _, endpoint := range options.Endpoints {
		endpoint = state.NormalizeEndpointID(endpoint)
		if endpoint != "" && !seenEndpoints[endpoint] {
			endpoints = append(endpoints, endpoint)
			seenEndpoints[endpoint] = true
		}
	}
	options.Endpoints = endpoints
	plugins := make([]PluginLaunch, 0, len(options.Plugins))
	seenPlugins := map[string]bool{}
	for _, plugin := range options.Plugins {
		if plugin.ID != "" && !seenPlugins[plugin.ID] {
			plugins = append(plugins, plugin)
			seenPlugins[plugin.ID] = true
		}
	}
	options.Plugins = plugins

	return &PluginService{router: router, options: options, hosts: map[string]pluginRoute{}, peers: map[string]pluginPeer{}, daemons: map[string]state.EndpointID{}, aliases: map[state.EndpointID]state.EndpointID{}, claims: map[string]string{}, endpointDaemons: map[state.EndpointID]string{}}
}
func pluginRouteKey(endpoint state.EndpointID, pluginID string) string {
	return string(endpoint) + "\x00" + pluginID
}
func peerKey(endpoint state.EndpointID, address *apipb.PluginAddress) string {
	return pluginRouteKey(endpoint, address.GetPluginId()) + "\x00" + address.GetPluginInstanceId()
}
func pluginApplicationExecutor(client *clientprotocol.ApplicationClient, endpoint state.EndpointID) sdk.Executor {
	return func(ctx context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		result, err := client.ApplicationSession.Execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_Plugin{Plugin: command}})
		if err != nil {
			return nil, err
		}
		if result.GetPlugin() == nil {
			return nil, fmt.Errorf("endpoint %s does not support plugins", endpoint)
		}
		return result.GetPlugin(), nil
	}
}
func (service *PluginService) canonicalLocked(endpoint state.EndpointID) state.EndpointID {
	// Keep the verified endpoint-to-daemon association across disconnection.
	// An existing feed can follow that same daemon's new canonical connection,
	// but still has to re-register: leases are never translated here.
	if daemonID := service.endpointDaemons[endpoint]; daemonID != "" {
		if canonical, ok := service.daemons[daemonID]; ok {
			return canonical
		}
		return endpoint
	}
	if canonical, ok := service.aliases[endpoint]; ok {
		return canonical
	}
	return endpoint
}
func (service *PluginService) pluginExecute(ctx context.Context, endpoint state.EndpointID, pluginID string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
	service.mu.RLock()
	endpoint = service.canonicalLocked(endpoint)
	route, ready := service.hosts[pluginRouteKey(endpoint, pluginID)]
	service.mu.RUnlock()
	if !ready || (route.alive != nil && !route.alive()) {
		return nil, fmt.Errorf("plugin host unavailable on endpoint %s", endpoint)
	}
	if command == nil {
		return nil, errors.New("plugin command required")
	}
	if registration := command.GetRegister(); registration != nil {
		a := registration.GetAddress()
		if a.GetPluginId() != pluginID || a.GetTuiInstanceId() != service.options.TUIInstanceID || a.GetPluginInstanceId() == "host" || registration.GetDaemonService() {
			return nil, errors.New("plugin registration exceeds its host identity")
		}
	} else {
		var lease []byte
		switch c := command.GetCommand().(type) {
		case *apipb.PluginCommand_Send:
			lease = c.Send.GetSourceLease()
		case *apipb.PluginCommand_Receive:
			lease = c.Receive.GetSourceLease()
		case *apipb.PluginCommand_State:
			lease = c.State.GetSourceLease()
		case *apipb.PluginCommand_Unregister:
			lease = c.Unregister.GetSourceLease()
		default:
			return nil, errors.New("unsupported plugin command")
		}
		service.mu.RLock()
		authorized := false
		for key, peer := range service.peers {
			if key == peerKey(endpoint, peer.address) && peer.address.GetPluginId() == pluginID && string(peer.lease) == string(lease) {
				authorized = true
				break
			}
		}
		service.mu.RUnlock()
		if !authorized {
			return nil, errors.New("plugin lease belongs to another instance or expired route")
		}
	}
	result, err := route.execute(ctx, command)
	if err != nil {
		return nil, err
	}
	service.mu.Lock()
	current, live := service.hosts[pluginRouteKey(endpoint, pluginID)]
	if !live || current.client != route.client {
		service.mu.Unlock()
		if reg := result.GetRegistration(); reg != nil {
			cleanupPluginPeers([]pluginPeer{{address: reg.Address, lease: reg.SourceLease, execute: route.execute}})
		}
		return nil, errors.New("plugin host changed during request")
	}
	defer service.mu.Unlock()

	if reg := result.GetRegistration(); reg != nil {
		if reg.GetAddress().GetDaemonId() != route.address.GetDaemonId() {
			return nil, errors.New("plugin registration daemon identity mismatch")
		}
		service.peers[peerKey(endpoint, reg.Address)] = pluginPeer{address: proto.Clone(reg.Address).(*apipb.PluginAddress), lease: append([]byte(nil), reg.SourceLease...), execute: route.execute}
	}
	if request := command.GetUnregister(); request != nil && result.GetAck() != nil {
		for key, peer := range service.peers {
			if key == peerKey(endpoint, peer.address) && string(peer.lease) == string(request.SourceLease) {
				delete(service.peers, key)
			}
		}
	}
	return result, nil
}

func (service *PluginService) ResolveDaemon(id string) (state.EndpointID, bool) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	ep, ok := service.daemons[id]
	if !ok {
		return "", false
	}
	for key, route := range service.hosts {
		if route.address.GetDaemonId() == id && key == pluginRouteKey(ep, route.address.GetPluginId()) && (route.alive == nil || route.alive()) {
			return ep, true
		}
	}
	return "", false
}

// DaemonIdentity exposes only a live, daemon-confirmed registration identity;
// endpoint aliases and a pending connection are not resource identities.
func (service *PluginService) DaemonIdentity(endpoint state.EndpointID) (string, bool) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	endpoint = service.canonicalLocked(endpoint)
	for key, route := range service.hosts {
		if key == pluginRouteKey(endpoint, route.address.GetPluginId()) && (route.alive == nil || route.alive()) {
			return route.address.GetDaemonId(), true
		}
	}
	return "", false
}
func (service *PluginService) Watch(ctx context.Context) (<-chan port.PluginDelivery, error) {
	service.mu.Lock()
	if service.started {
		service.mu.Unlock()
		return nil, errors.New("plugin watch already started")
	}
	service.started = true
	service.mu.Unlock()
	deliveries := make(chan port.PluginDelivery, 128)
	var workers sync.WaitGroup
	initial := sync.WaitGroup{}
	for _, endpoint := range service.options.Endpoints {
		for _, plugin := range service.options.Plugins {
			initial.Add(1)
			workers.Add(1)
			go func() { defer workers.Done(); service.hostLoop(ctx, endpoint, plugin.ID, deliveries, initial.Done) }()
		}
	}
	for _, plugin := range service.options.Plugins {
		workers.Add(1)
		go func() {
			defer workers.Done()
			initial.Wait()
			if ctx.Err() != nil || (plugin.Run == nil && plugin.RunWithExit == nil) {
				return
			}
			execute := func(callCtx context.Context, endpointID string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
				endpoint := state.EndpointID(endpointID)
				if endpoint == "" && len(service.options.Endpoints) > 0 {
					endpoint = service.options.Endpoints[0]
				}
				return service.pluginExecute(callCtx, endpoint, plugin.ID, command)
			}
			service.mu.RLock()
			endpoints := make([]state.EndpointID, 0, len(service.options.Endpoints))
			for _, ep := range service.options.Endpoints {
				if canonical := service.canonicalLocked(ep); canonical == ep {
					endpoints = append(endpoints, ep)
				}
			}
			service.mu.RUnlock()
			var reported atomic.Bool
			onExit := func(err error) {
				if ctx.Err() != nil {
					return
				}
				reported.Store(true)
				if err == nil {
					err = errors.New("plugin UI process exited")
				}
				service.processExited(ctx, plugin.ID, err, deliveries)
			}
			var err error
			if plugin.RunWithExit != nil {
				err = plugin.RunWithExit(ctx, execute, endpoints, onExit)
			} else {
				err = plugin.Run(ctx, execute, endpoints)
			}
			if !reported.Load() {
				onExit(err)
			}

		}()
	}
	go func() { workers.Wait(); close(deliveries) }()
	return deliveries, nil
}
func (service *PluginService) releaseClaim(endpoint state.EndpointID, pluginID, daemonID string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	delete(service.claims, pluginRouteKey(endpoint, pluginID))
	for _, id := range service.claims {
		if id == daemonID {
			return
		}
	}
	if service.daemons[daemonID] == endpoint {
		delete(service.daemons, daemonID)
	}
}
func (service *PluginService) clearPeersLocked(endpoint state.EndpointID, pluginID string) []pluginPeer {
	var peers []pluginPeer
	for key, peer := range service.peers {
		if peer.address.GetPluginId() == pluginID && key == peerKey(endpoint, peer.address) {
			peers = append(peers, peer)
			delete(service.peers, key)
		}
	}
	return peers
}
func cleanupPluginPeers(peers []pluginPeer) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, peer := range peers {
		if ctx.Err() != nil {
			return
		}
		_, _ = peer.execute(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Unregister{Unregister: &apipb.PluginUnregisterRequest{SourceLease: peer.lease}}})
	}
}
func (service *PluginService) processExited(ctx context.Context, pluginID string, err error, out chan<- port.PluginDelivery) {
	service.mu.Lock()
	endpoints := map[state.EndpointID]bool{}
	var peers []pluginPeer
	for key, route := range service.hosts {
		if route.address.GetPluginId() != pluginID {
			continue
		}
		for _, endpoint := range service.options.Endpoints {
			if key == pluginRouteKey(endpoint, pluginID) {
				endpoints[endpoint] = true
				peers = append(peers, service.clearPeersLocked(endpoint, pluginID)...)
				break
			}
		}
	}
	service.mu.Unlock()
	for endpoint := range endpoints {
		select {
		case out <- port.PluginDelivery{EndpointID: endpoint, PluginID: pluginID, Err: err}:
		case <-ctx.Done():
		}
	}
	cleanupPluginPeers(peers)
}
func (service *PluginService) hostLoop(ctx context.Context, endpoint state.EndpointID, pluginID string, out chan<- port.PluginDelivery, initialDone func()) {
	var initialOnce sync.Once
	defer initialOnce.Do(initialDone)
	for ctx.Err() == nil {
		registerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		application, err := service.router.application(registerCtx, endpoint)
		var daemonID string
		duplicate := false
		claimed := false
		if err == nil {
			daemonID = application.Readiness().Identity.DeviceID
			if daemonID == "" {
				err = errors.New("endpoint lacks authenticated daemon identity")
			} else {
				service.mu.Lock()
				service.endpointDaemons[endpoint] = daemonID
				canonical, exists := service.daemons[daemonID]
				if !exists {
					canonical = endpoint
					service.daemons[daemonID] = endpoint
				}
				service.aliases[endpoint] = canonical
				duplicate = canonical != endpoint
				if !duplicate {
					service.claims[pluginRouteKey(endpoint, pluginID)] = daemonID
					claimed = true
				}
				service.mu.Unlock()
			}
		}
		if duplicate {
			cancel()
			initialOnce.Do(initialDone)
			if !pluginRetry(ctx) {
				return
			}
			continue
		}
		var client *sdk.Client
		var reg *apipb.PluginRegistration
		var execute sdk.Executor
		if err == nil {
			execute = pluginApplicationExecutor(application, endpoint)
			client = sdk.NewClient(execute)
			reg, err = client.Register(registerCtx, &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{TuiInstanceId: service.options.TUIInstanceID, PluginId: pluginID, PluginInstanceId: "host"}})
		}
		cancel()
		if err == nil && reg.Address.DaemonId != daemonID {
			err = errors.New("registration differs from authenticated daemon identity")
		}
		if err == nil {
			service.mu.Lock()
			service.hosts[pluginRouteKey(endpoint, pluginID)] = pluginRoute{client: client, address: reg.Address, execute: execute, alive: func() bool { return applicationClientReady(application) }}
			service.mu.Unlock()
			initialOnce.Do(initialDone)
			for ctx.Err() == nil {
				var batch *apipb.PluginBatch
				batch, err = client.Receive(ctx, time.Second)
				if err != nil {
					break
				}
				if batch.ResyncRequired {
					err = errors.New("plugin host queue requires resync")
					break
				}
				for _, message := range batch.Messages {
					select {
					case out <- port.PluginDelivery{EndpointID: endpoint, PluginID: pluginID, Message: message}:
					case <-ctx.Done():
					}
				}
			}
			service.mu.Lock()
			current := service.hosts[pluginRouteKey(endpoint, pluginID)]
			var peers []pluginPeer
			if current.client == client {
				delete(service.hosts, pluginRouteKey(endpoint, pluginID))
				peers = service.clearPeersLocked(endpoint, pluginID)
			}
			service.mu.Unlock()
			cleanupPluginPeers(peers)
		}
		if reg != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = client.Unregister(cleanup)
			cancel()
		}
		if claimed {
			service.releaseClaim(endpoint, pluginID, daemonID)
		}
		initialOnce.Do(initialDone)
		if err != nil && ctx.Err() == nil {
			select {
			case out <- port.PluginDelivery{EndpointID: endpoint, PluginID: pluginID, Err: err}:
			case <-ctx.Done():
			}
		}
		if !pluginRetry(ctx) {
			return
		}
	}
}
func pluginRetry(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func (service *PluginService) Send(ctx context.Context, endpoint state.EndpointID, message *apipb.PluginMessage) error {
	if message == nil || message.Destination == nil {
		return errors.New("plugin UI delivery requires destination")
	}
	message = proto.Clone(message).(*apipb.PluginMessage)
	service.mu.RLock()
	endpoint = service.canonicalLocked(endpoint)
	route, ok := service.hosts[pluginRouteKey(endpoint, message.Destination.PluginId)]
	peer, hasPeer := service.peers[peerKey(endpoint, message.Destination)]
	service.mu.RUnlock()
	if !ok || (route.alive != nil && !route.alive()) {
		return errors.New("plugin host is offline")
	}
	if message.Destination.DaemonId != route.address.DaemonId {
		return errors.New("plugin destination does not belong to selected daemon")
	}
	// Epoch zero is only a local request to resolve a NEW cross-endpoint action.
	// Existing exact addresses are never silently upgraded after reconnect.
	if message.Destination.RegistrationEpoch == 0 {
		if message.GetInteraction() == nil || message.Destination.TuiInstanceId != service.options.TUIInstanceID || !hasPeer {
			return errors.New("only a fresh TUI interaction can resolve a peer address")
		}
		message.Destination = proto.Clone(peer.address).(*apipb.PluginAddress)
	} else if hasPeer && !proto.Equal(peer.address, message.Destination) {
		return errors.New("plugin destination registration is stale")
	}
	return route.client.Send(ctx, message)
}
