package clientruntimeadapter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"google.golang.org/protobuf/proto"
)

func testPluginRoute(t *testing.T, daemon, plugin string, epoch uint64, send func(*apipb.PluginCommand)) *pluginRoute {
	t.Helper()
	address := &apipb.PluginAddress{DaemonId: daemon, TuiInstanceId: "tui", PluginId: plugin, PluginInstanceId: "host", RegistrationEpoch: epoch}
	execute := func(_ context.Context, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		if command.GetRegister() != nil {
			return &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: &apipb.PluginRegistration{Address: address, SourceLease: []byte("host-lease")}}}, nil
		}
		if send != nil {
			send(command)
		}
		return &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{Delivered: 1}}}, nil
	}
	client := sdk.NewClient(execute)
	if _, err := client.Register(context.Background(), &apipb.PluginRegisterRequest{}); err != nil {
		t.Fatal(err)
	}
	return &pluginRoute{client: client, address: address, execute: execute}
}
func TestPluginSendNeverUpgradesOldEpochOrForwardsForeignDaemon(t *testing.T) {
	var sent []*apipb.PluginMessage
	service := NewPluginService(nil, PluginOptions{TUIInstanceID: "tui"})
	route := testPluginRoute(t, "d", "p", 9, func(c *apipb.PluginCommand) { sent = append(sent, c.GetSend().Message) })
	service.hosts[pluginRouteKey("canonical", "p")] = *route
	service.aliases["alias"] = "canonical"
	peer := &apipb.PluginAddress{DaemonId: "d", TuiInstanceId: "tui", PluginId: "p", PluginInstanceId: "ui", RegistrationEpoch: 12}
	service.peers[peerKey("canonical", peer)] = pluginPeer{address: peer}
	old := proto.Clone(peer).(*apipb.PluginAddress)
	old.RegistrationEpoch = 11
	message := &apipb.PluginMessage{Destination: old, Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{}}}
	if err := service.Send(context.Background(), "alias", message); err == nil {
		t.Fatal("old epoch silently upgraded")
	}
	old.DaemonId = "foreign"
	if err := service.Send(context.Background(), "alias", message); err == nil {
		t.Fatal("foreign daemon forwarded")
	}
	old.DaemonId = "d"
	old.RegistrationEpoch = 0
	if err := service.Send(context.Background(), "alias", message); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 1 || sent[0].Destination.RegistrationEpoch != 12 || message.Destination.RegistrationEpoch != 0 {
		t.Fatalf("fresh interaction resolution mutated input or missed peer: %v", sent)
	}
	message.Body = &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: "x"}}
	if err := service.Send(context.Background(), "alias", message); err == nil {
		t.Fatal("reply with unresolved target epoch accepted")
	}
}
func TestPluginExitIsScopedAndCleansOnlyItsPeerLeases(t *testing.T) {
	service := NewPluginService(nil, PluginOptions{TUIInstanceID: "tui", Endpoints: []state.EndpointID{"a", "b"}})
	cleaned := 0
	for _, endpoint := range service.options.Endpoints {
		for _, id := range []string{"one", "two"} {
			route := testPluginRoute(t, string(endpoint), id, 1, nil)
			service.hosts[pluginRouteKey(endpoint, id)] = *route
			peer := proto.Clone(route.address).(*apipb.PluginAddress)
			peer.PluginInstanceId = "ui"
			service.peers[peerKey(endpoint, peer)] = pluginPeer{address: peer, lease: []byte(id), execute: func(_ context.Context, c *apipb.PluginCommand) (*apipb.PluginResult, error) {
				if string(c.GetUnregister().SourceLease) != "one" {
					t.Fatal("unregistered another plugin")
				}
				cleaned++
				return &apipb.PluginResult{}, nil
			}}
		}
	}
	out := make(chan port.PluginDelivery, 4)
	service.processExited(context.Background(), "one", errors.New("exited"), out)
	if cleaned != 2 || len(service.peers) != 2 || len(out) != 2 {
		t.Fatalf("cleanup=%d peers=%d errors=%d", cleaned, len(service.peers), len(out))
	}
	for range 2 {
		d := <-out
		if d.PluginID != "one" || d.EndpointID == "" || d.Err == nil {
			t.Fatalf("unscoped exit: %+v", d)
		}
	}
}
func TestPluginIdentityResolutionRequiresLiveHostAndReleasesFailedClaim(t *testing.T) {
	service := NewPluginService(nil, PluginOptions{})
	service.daemons["d"] = "a"
	service.aliases["alias"] = "a"
	service.claims[pluginRouteKey("a", "p")] = "d"
	if _, ok := service.ResolveDaemon("d"); ok {
		t.Fatal("reservation advertised as live daemon")
	}
	service.hosts[pluginRouteKey("a", "p")] = *testPluginRoute(t, "d", "p", 1, nil)
	if id, ok := service.DaemonIdentity("alias"); !ok || id != "d" {
		t.Fatal("alias did not resolve live identity")
	}
	delete(service.hosts, pluginRouteKey("a", "p"))
	service.releaseClaim("a", "p", "d")
	if _, exists := service.daemons["d"]; exists {
		t.Fatal("failed canonical claim prevents alias takeover")
	}
}

type pluginReadyTest struct {
	*routerReady
	identity      string
	registrations chan state.EndpointID
	mu            sync.Mutex
	epoch         uint64
}

func (r *pluginReadyTest) Readiness() clientruntime.ReadyPeerSessionEvidence {
	return clientruntime.ReadyPeerSessionEvidence{Identity: clientendpoint.DaemonIdentity{DeviceID: r.identity}}
}
func (r *pluginReadyTest) ExecuteApplication(ctx context.Context, envelope *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	select {
	case <-r.Done():
		return nil, errors.New("connection offline")
	default:
	}
	var result *apipb.PluginResult
	command := envelope.GetPlugin()
	switch {
	case command.GetRegister() != nil:
		r.mu.Lock()
		r.epoch++
		epoch := r.epoch
		r.mu.Unlock()
		a := proto.Clone(command.GetRegister().Address).(*apipb.PluginAddress)
		a.DaemonId = r.identity
		a.RegistrationEpoch = epoch
		result = &apipb.PluginResult{Result: &apipb.PluginResult_Registration{Registration: &apipb.PluginRegistration{Address: a, SourceLease: []byte(fmt.Sprintf("%s-%d", r.Stamp().EndpointID, epoch))}}}
		if a.PluginInstanceId == "host" {
			r.registrations <- state.EndpointID(r.Stamp().EndpointID)
		}
	case command.GetReceive() != nil:
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-r.Done():
			return nil, errors.New("connection offline")
		}
	default:
		result = &apipb.PluginResult{Result: &apipb.PluginResult_Ack{Ack: &apipb.PluginAck{}}}
	}
	return &apipb.ResultEnvelope{RequestId: envelope.GetContext().RequestId, OriginSession: envelope.GetContext().Session, Result: &apipb.ResultEnvelope_Plugin{Plugin: result}}, nil
}
func TestPluginWatchDeduplicatesAliasesAndReportsCleanProcessExit(t *testing.T) {
	registrations := make(chan state.EndpointID, 8)
	a := &pluginReadyTest{routerReady: newRouterReady("a", ""), identity: "same", registrations: registrations}
	b := &pluginReadyTest{routerReady: newRouterReady("b", ""), identity: "same", registrations: registrations}
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{"a": a, "b": b}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(a, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("a", initial)
	if err != nil {
		t.Fatal(err)
	}
	defer router.Close()
	endpointCount := make(chan int, 1)
	service := NewPluginService(router, PluginOptions{TUIInstanceID: "tui", Endpoints: []state.EndpointID{"a", "b"}, Plugins: []PluginLaunch{{ID: "p", Run: func(_ context.Context, _ sdk.RoutedExecutor, endpoints []state.EndpointID) error {
		endpointCount <- len(endpoints)
		return nil
	}}}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out, err := service.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case count := <-endpointCount:
		if count != 1 {
			t.Fatalf("aliases launched %d endpoint feeds", count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("plugin did not start")
	}
	canonical := <-registrations
	select {
	case duplicate := <-registrations:
		t.Fatalf("duplicate registered: %s", duplicate)
	default:
	}
	select {
	case d := <-out:
		if d.PluginID != "p" || d.EndpointID != canonical || d.Err == nil {
			t.Fatalf("clean exit not scoped: %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("clean process exit did not notify UI")
	}
	cancel()
	select {
	case <-out:
	case <-time.After(2 * time.Second):
		t.Fatal("watch failed to stop")
	}
}

func TestPluginSupervisedRestartRevokesOldPeerBeforeNewRegistration(t *testing.T) {
	ready := &pluginReadyTest{routerReady: newRouterReady("a", ""), identity: "d", registrations: make(chan state.EndpointID, 4)}
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{"a": ready}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(ready, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("a", initial)
	if err != nil {
		t.Fatal(err)
	}
	defer router.Close()
	checked := make(chan error, 1)
	launch := PluginLaunch{ID: "p", RunWithExit: func(ctx context.Context, execute sdk.RoutedExecutor, _ []state.EndpointID, onExit func(error)) error {
		register := &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: "p", PluginInstanceId: "ui", TuiInstanceId: "tui"}}}}
		first, err := execute(ctx, "a", register)
		if err != nil {
			checked <- err
			return err
		}
		onExit(errors.New("child restarting"))
		_, err = execute(ctx, "a", &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: first.GetRegistration().SourceLease}}})
		if err == nil {
			err = errors.New("old child lease still authorized")
			checked <- err
			return err
		}
		next, err := execute(ctx, "a", register)
		if err == nil && next.GetRegistration().Address.RegistrationEpoch <= first.GetRegistration().Address.RegistrationEpoch {
			err = errors.New("restart did not advance registration epoch")
		}
		checked <- err
		<-ctx.Done()
		return ctx.Err()
	}}
	service := NewPluginService(router, PluginOptions{TUIInstanceID: "tui", Endpoints: []state.EndpointID{"a"}, Plugins: []PluginLaunch{launch}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out, err := service.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case d := <-out:
		if d.EndpointID != "a" || d.PluginID != "p" || d.Err == nil {
			t.Fatalf("restart did not mark own endpoint stale: %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("no restart notification")
	}
	select {
	case err := <-checked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("restart did not complete")
	}
	cancel()
	for range out {
	}
}

func TestPluginAliasTakesOverAfterCanonicalConnectionDies(t *testing.T) {
	registrations := make(chan state.EndpointID, 8)
	a := &pluginReadyTest{routerReady: newRouterReady("a", ""), identity: "same", registrations: registrations}
	b := &pluginReadyTest{routerReady: newRouterReady("b", ""), identity: "same", registrations: registrations}
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{"a": a, "b": b}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(a, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("a", initial)
	if err != nil {
		t.Fatal(err)
	}
	defer router.Close()
	started := make(chan struct{})
	service := NewPluginService(router, PluginOptions{TUIInstanceID: "tui", Endpoints: []state.EndpointID{"a", "b"}, Plugins: []PluginLaunch{{ID: "p", Run: func(ctx context.Context, _ sdk.RoutedExecutor, _ []state.EndpointID) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out, err := service.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("initial host did not start")
	}
	original := <-registrations
	registrationRequest := &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: "p", PluginInstanceId: "ui", TuiInstanceId: "tui"}}}}
	oldPeer, err := service.pluginExecute(ctx, original, "p", registrationRequest)
	if err != nil {
		t.Fatal(err)
	}
	if original == "a" {
		_ = a.Close()
	} else {
		_ = b.Close()
	}
	if _, ok := service.ResolveDaemon("same"); ok {
		t.Fatal("dead canonical connection still exposed as live")
	}
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case endpoint := <-registrations:
			if endpoint == original {
				t.Fatal("dead endpoint registered again")
			}
			// Registration response publication can follow the fixture observation.
			service.mu.RLock()
			alias := service.aliases[endpoint]
			service.mu.RUnlock()
			if alias != endpoint {
				t.Fatal("alias takeover did not establish canonical identity")
			}
			// The original long-lived feed still selects the OLD endpoint name.
			// Wait only for the newly observed host to finish publication.
			deadline := time.Now().Add(time.Second)
			for {
				if _, live := service.DaemonIdentity(original); live {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("old endpoint name cannot resolve new live host")
				}
				time.Sleep(time.Millisecond)
			}
			staleReceive := &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: oldPeer.GetRegistration().SourceLease}}}
			if _, err := service.pluginExecute(ctx, original, "p", staleReceive); err == nil {
				t.Fatal("old lease translated onto takeover connection")
			}
			fresh, err := service.pluginExecute(ctx, original, "p", registrationRequest)
			if err != nil {
				t.Fatalf("old feed could not re-register through new canonical connection: %v", err)
			}
			if string(fresh.GetRegistration().SourceLease) == string(oldPeer.GetRegistration().SourceLease) {
				t.Fatal("takeover reused old lease")
			}
			unregister := &apipb.PluginCommand{Command: &apipb.PluginCommand_Unregister{Unregister: &apipb.PluginUnregisterRequest{SourceLease: fresh.GetRegistration().SourceLease}}}
			if _, err := service.pluginExecute(ctx, original, "p", unregister); err != nil {
				t.Fatal(err)
			}
			cancel()
			for range out {
			}
			return
		case d := <-out:
			if d.Err != nil && (d.PluginID != "p" || d.EndpointID == "") {
				t.Fatalf("unscoped reconnect error: %+v", d)
			}
		case <-timeout.C:
			t.Fatal("live alias never replaced failed canonical connection")
		}
	}
}
