package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
)

func corePluginSession(t *testing.T, server *Server) *protocolSession {
	t.Helper()
	return &protocolSession{server: server, scope: fullDaemonTransportScope()}
}
func corePluginCall(t *testing.T, s *protocolSession, c *apipb.PluginCommand) *apipb.PluginResult {
	t.Helper()
	result, err := s.ApplicationPlugin(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func corePluginRegister(t *testing.T, s *protocolSession, instance string, service bool) *apipb.PluginRegistration {
	t.Helper()
	result := corePluginCall(t, s, &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: "org.anytty.agents", PluginInstanceId: instance}, DaemonService: service}}})
	if result.GetRegistration() == nil {
		t.Fatal(result)
	}
	return result.GetRegistration()
}
func corePluginState(t *testing.T, s *protocolSession, reg *apipb.PluginRegistration, request *apipb.PluginStateRequest) *apipb.PluginResult {
	t.Helper()
	request.SourceLease = reg.SourceLease
	request.Collection = "agents"
	return corePluginCall(t, s, &apipb.PluginCommand{Command: &apipb.PluginCommand_State{State: request}})
}
func TestPluginStateAtomicWatchOverflowAndPersistentRecovery(t *testing.T) {
	directory := t.TempDir()
	server := NewServer(WithPluginRuntime("d", directory))
	s := corePluginSession(t, server)
	service := corePluginRegister(t, s, "service", true)
	reader := corePluginRegister(t, s, "reader", false)
	watch := corePluginState(t, s, reader, &apipb.PluginStateRequest{Operation: &apipb.PluginStateRequest_Watch{Watch: &apipb.PluginStateWatch{}}})
	if watch.GetState() == nil || watch.GetState().Revision != 0 {
		t.Fatal(watch)
	}
	put := func(reg *apipb.PluginRegistration, revision uint64) *apipb.PluginResult {
		return corePluginState(t, s, reg, &apipb.PluginStateRequest{Operation: &apipb.PluginStateRequest_Put{Put: &apipb.PluginStatePut{ExpectedRevision: revision, Value: &apipb.PluginPayload{Schema: "agents", Version: 1, Data: []byte(fmt.Sprint(revision + 1))}}}})
	}
	if got := put(reader, 0); got.GetError().GetCode() != "PERMISSION_DENIED" {
		t.Fatal(got)
	}
	for i := 0; i < pluginQueueLimit+2; i++ {
		if got := put(service, uint64(i)); got.GetState().GetRevision() != uint64(i+1) {
			t.Fatal(got)
		}
	}
	if got := put(service, 0); got.GetError().GetCode() != "CONFLICT" {
		t.Fatal(got)
	}
	batch := corePluginCall(t, s, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: reader.SourceLease}}}).GetBatch()
	if !batch.GetResyncRequired() || len(batch.Messages) != 32 {
		t.Fatalf("slow reader must resync, got %v", batch)
	}
	current := corePluginState(t, s, reader, &apipb.PluginStateRequest{Operation: &apipb.PluginStateRequest_Get{Get: &apipb.PluginStateGet{}}}).GetState()
	if current.Revision != pluginQueueLimit+2 {
		t.Fatal(current)
	}
	// A new independent server reads the durable snapshot, but changes its epoch.
	restarted := NewServer(WithPluginRuntime("d", directory))
	s2 := corePluginSession(t, restarted)
	reader2 := corePluginRegister(t, s2, "reader", false)
	restored := corePluginState(t, s2, reader2, &apipb.PluginStateRequest{Operation: &apipb.PluginStateRequest_Watch{Watch: &apipb.PluginStateWatch{}}}).GetState()
	if restored.Revision != current.Revision || string(restored.Value.Data) != string(current.Value.Data) || restored.BootEpoch == current.BootEpoch {
		t.Fatalf("bad durable state recovery: %v", restored)
	}
	if got := corePluginCall(t, s2, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: reader.SourceLease}}}); got.GetError().GetCode() != "STALE_CONTEXT" {
		t.Fatal(got)
	}
}
func TestPluginReceiveCancellationAndSessionCleanup(t *testing.T) {
	server := NewServer(WithPluginRuntime("d", t.TempDir()))
	s := corePluginSession(t, server)
	reg := corePluginRegister(t, s, "reader", false)
	result := make(chan *apipb.PluginResult, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		value, _ := s.ApplicationPlugin(ctx, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: reg.SourceLease, WaitMillis: 25000}}})
		result <- value
	}()
	cancel()
	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("receive ignored cancellation")
	}
	s.releasePluginRegistrations()
	if got := corePluginCall(t, s, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: reg.SourceLease}}}); got.GetError().GetCode() != "STALE_CONTEXT" {
		t.Fatal(got)
	}
}
func TestPluginRemoteServiceImpersonationDenied(t *testing.T) {
	server := NewServer(WithPluginRuntime("d", t.TempDir()))
	remote := corePluginSession(t, server)
	remote.scope.LocalOwner = false
	got := corePluginCall(t, remote, &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: "org.anytty.agents", PluginInstanceId: "fake-service"}, DaemonService: true}}})
	if got.GetError().GetCode() != "PERMISSION_DENIED" {
		t.Fatal(got)
	}
}
