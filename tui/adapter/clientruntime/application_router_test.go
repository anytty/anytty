package clientruntimeadapter

import (
	"context"
	"sync"
	"testing"
	"time"

	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"google.golang.org/protobuf/proto"
)

func TestEndpointApplicationRouterAcquiresOwningEndpoint(t *testing.T) {
	local := newRouterReady("local", "/Users/local")
	remote := newRouterReady("al", "/root")
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{
		"al": remote,
	}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(local, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("local", initial)
	if err != nil {
		t.Fatal(err)
	}
	defer router.Close()

	defaults, err := router.Defaults(context.Background(), port.PathDefaultsRequest{EndpointID: "al"})
	if err != nil {
		t.Fatal(err)
	}
	if defaults.EndpointID != "al" || defaults.DefaultCWD != "/root" || len(defaults.DefaultCommand) != 1 || defaults.DefaultCommand[0] != "/bin/zsh" {
		t.Fatalf("remote defaults = %#v", defaults)
	}
	if len(runtime.requests) != 1 || runtime.requests[0].EndpointID != "al" {
		t.Fatalf("acquire requests = %#v", runtime.requests)
	}

	if _, err := router.Defaults(context.Background(), port.PathDefaultsRequest{EndpointID: "al"}); err != nil {
		t.Fatal(err)
	}
	if len(runtime.requests) != 1 {
		t.Fatalf("cached remote session was reacquired: %#v", runtime.requests)
	}
	if err := router.Close(); err != nil {
		t.Fatal(err)
	}
	if remote.closeCount != 1 {
		t.Fatalf("remote consumer close count = %d", remote.closeCount)
	}
	if local.closeCount != 0 {
		t.Fatalf("router closed composition-owned initial client: %d", local.closeCount)
	}
}

func TestEndpointApplicationRouterSamplesOnlyCachedSession(t *testing.T) {
	local := newRouterReady("local", "/Users/local")
	local.connection = clientruntime.ConnectionSnapshot{LocalAddress: "192.0.2.10", RemoteAddress: "192.0.2.20", RoundTrip: 20 * time.Millisecond, Connected: true}
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(local, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("local", initial)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, valid, err := router.ConnectionSnapshot(context.Background(), "local")
	if err != nil || !valid || snapshot.RemoteAddress != "192.0.2.20" {
		t.Fatalf("cached snapshot = %#v valid=%t err=%v", snapshot, valid, err)
	}
	if _, valid, err = router.ConnectionSnapshot(context.Background(), "offline"); err != nil || valid {
		t.Fatalf("offline snapshot valid=%t err=%v", valid, err)
	}
	if len(runtime.requests) != 0 {
		t.Fatalf("sampling dialed an endpoint: %#v", runtime.requests)
	}
}

func TestEndpointApplicationRouterDisablesIdleSessionImmediately(t *testing.T) {
	local := newRouterReady("local", "/Users/local")
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(local, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("local", initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.SetEndpointEnabled(context.Background(), "local", false); err != nil {
		t.Fatal(err)
	}
	if len(runtime.disconnects) != 1 || runtime.disconnects[0].Stamp != local.stamp {
		t.Fatalf("disconnects = %#v", runtime.disconnects)
	}
	if _, err := router.Defaults(context.Background(), port.PathDefaultsRequest{EndpointID: "local"}); err == nil {
		t.Fatal("disabled endpoint accepted a new defaults request")
	}
	if len(runtime.requests) != 0 {
		t.Fatalf("disabled endpoint was dialed: %#v", runtime.requests)
	}
}

func TestEndpointApplicationRouterDrainsAttachmentsBeforeDisconnect(t *testing.T) {
	local := newRouterReady("local", "/Users/local")
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(local, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("local", initial)
	if err != nil {
		t.Fatal(err)
	}
	if !router.recordAttachment("local", 7) || !router.recordAttachment("local", 8) {
		t.Fatal("failed to record active attachments")
	}
	if err := router.SetEndpointEnabled(context.Background(), "local", false); err != nil {
		t.Fatal(err)
	}
	if len(runtime.disconnects) != 0 {
		t.Fatalf("active session disconnected early: %#v", runtime.disconnects)
	}
	if err := router.releaseAttachment(context.Background(), "local", 7); err != nil {
		t.Fatal(err)
	}
	if len(runtime.disconnects) != 0 {
		t.Fatalf("session disconnected before last attachment: %#v", runtime.disconnects)
	}
	if err := router.releaseAttachment(context.Background(), "local", 8); err != nil {
		t.Fatal(err)
	}
	if len(runtime.disconnects) != 1 || runtime.disconnects[0].Stamp != local.stamp {
		t.Fatalf("last attachment did not disconnect exact session: %#v", runtime.disconnects)
	}
}

func TestEndpointApplicationRouterReenableCancelsDeferredDisconnect(t *testing.T) {
	local := newRouterReady("local", "/Users/local")
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(local, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("local", initial)
	if err != nil {
		t.Fatal(err)
	}
	if !router.recordAttachment("local", 7) {
		t.Fatal("failed to record active attachment")
	}
	if err := router.SetEndpointEnabled(context.Background(), "local", false); err != nil {
		t.Fatal(err)
	}
	if err := router.SetEndpointEnabled(context.Background(), "local", true); err != nil {
		t.Fatal(err)
	}
	if err := router.releaseAttachment(context.Background(), "local", 7); err != nil {
		t.Fatal(err)
	}
	if len(runtime.disconnects) != 0 {
		t.Fatalf("reenabled endpoint was disconnected: %#v", runtime.disconnects)
	}
}

func TestEndpointApplicationRouterDropsAttachmentsFromClosedGeneration(t *testing.T) {
	local := newRouterReady("local", "/Users/local")
	runtime := &routerRuntime{sessions: map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession{}}
	initial, err := clientprotocol.NewRuntimeApplicationClient(local, runtime)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewEndpointApplicationRouter("local", initial)
	if err != nil {
		t.Fatal(err)
	}
	if !router.recordAttachment("local", 7) {
		t.Fatal("failed to record active attachment")
	}
	if err := local.Close(); err != nil {
		t.Fatal(err)
	}
	if err := router.SetEndpointEnabled(context.Background(), "local", false); err != nil {
		t.Fatal(err)
	}
	router.mu.Lock()
	remaining := len(router.attachments["local"])
	router.mu.Unlock()
	if remaining != 0 || len(runtime.disconnects) != 0 {
		t.Fatalf("closed generation kept lifecycle state: attachments=%d disconnects=%#v", remaining, runtime.disconnects)
	}
}

type routerRuntime struct {
	mu          sync.Mutex
	sessions    map[clientendpoint.EndpointID]clientruntime.ApplicationReadyPeerSession
	requests    []clientruntime.ConnectRequest
	disconnects []clientruntime.DisconnectRequest
}

func (runtime *routerRuntime) AcquireSession(_ context.Context, request clientruntime.ConnectRequest) (clientruntime.ApplicationReadyPeerSession, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	runtime.requests = append(runtime.requests, request)
	return runtime.sessions[request.EndpointID], nil
}

func (*routerRuntime) EnsureSession(context.Context, clientruntime.ConnectRequest) (clientruntime.SessionLease, error) {
	return clientruntime.SessionLease{}, nil
}

func (runtime *routerRuntime) Disconnect(_ context.Context, request clientruntime.DisconnectRequest) error {
	runtime.mu.Lock()
	runtime.disconnects = append(runtime.disconnects, request)
	runtime.mu.Unlock()
	return nil
}

func (*routerRuntime) WatchEndpoint(context.Context, clientendpoint.EndpointID) (<-chan clientruntime.EndpointEvent, error) {
	return make(chan clientruntime.EndpointEvent), nil
}

type routerReady struct {
	stamp      clientruntime.EndpointSessionStamp
	cwd        string
	done       chan struct{}
	closeOnce  sync.Once
	closeCount int
	connection clientruntime.ConnectionSnapshot
}

func newRouterReady(endpointID clientendpoint.EndpointID, cwd string) *routerReady {
	return &routerReady{stamp: clientruntime.EndpointSessionStamp{EndpointID: endpointID, RouteID: "route", Generation: 1}, cwd: cwd, done: make(chan struct{})}
}

func (ready *routerReady) Stamp() clientruntime.EndpointSessionStamp { return ready.stamp }
func (*routerReady) ObservedPath() string                            { return "test" }
func (*routerReady) Readiness() clientruntime.ReadyPeerSessionEvidence {
	return clientruntime.ReadyPeerSessionEvidence{}
}
func (ready *routerReady) Done() <-chan struct{} { return ready.done }
func (*routerReady) Err() error                  { return nil }
func (ready *routerReady) Close() error {
	ready.closeOnce.Do(func() {
		ready.closeCount++
		close(ready.done)
	})
	return nil
}
func (ready *routerReady) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result: &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{
			DefaultCommand: []string{"/bin/zsh"}, DefaultCwd: ready.cwd,
		}}},
	}, nil
}
func (*routerReady) ApplicationEvents(context.Context) (<-chan *apipb.EventEnvelope, error) {
	return make(chan *apipb.EventEnvelope), nil
}
func (ready *routerReady) ConnectionSnapshot(at time.Time) (clientruntime.ConnectionSnapshot, bool) {
	snapshot := ready.connection
	snapshot.SampledAt = at
	return snapshot, snapshot.Connected
}

var _ clientruntime.ApplicationRuntime = (*routerRuntime)(nil)
var _ clientruntime.ApplicationReadyPeerSession = (*routerReady)(nil)
var _ clientruntime.ConnectionSnapshotProvider = (*routerReady)(nil)
