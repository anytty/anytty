package daemon_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	adapter "github.com/anytty/anytty/access/provider/daemon"
	"github.com/anytty/anytty/daemon/core"
	provider "github.com/anytty/anytty/daemon/provider"
	"github.com/anytty/anytty/proto/access/apipb"
)

func startAdapterProvider(t *testing.T) string {
	t.Helper()
	coreServer := core.NewServer(core.WithHistoryStorageDir(t.TempDir()))
	t.Cleanup(func() { _ = coreServer.Shutdown(context.Background()) })
	socketPath := filepath.Join(t.TempDir(), "provider.sock")
	server, err := provider.New(coreServer, provider.Config{Socket: socketPath})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); _ = server.Shutdown(context.Background()) })
	go func() { _ = server.ListenAndServe(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			return socketPath
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("provider socket did not appear")
	return ""
}

func withContext(requestID string, envelope *apipb.CommandEnvelope) *apipb.CommandEnvelope {
	envelope.Context = &apipb.RequestContext{
		RequestId:  requestID,
		ApiVersion: &apipb.ApiVersion{Major: 1},
		Session:    &apipb.EndpointSessionStamp{EndpointId: "local", RouteId: "adapter-test", Generation: 1},
	}
	return envelope
}

// TestAdapterTerminalCommandAndEventFlow 覆盖 Slice B adapter 的终端命令分发、
// provider 错误映射与 terminal event subscription 投影。
func TestAdapterTerminalCommandAndEventFlow(t *testing.T) {
	socketPath := startAdapterProvider(t)
	ctx := context.Background()
	terminal, err := adapter.DialTerminal(ctx, socketPath)
	if err != nil {
		t.Fatalf("dial adapter: %v", err)
	}
	defer func() { _ = terminal.Close() }()

	events, err := terminal.Events(ctx)
	if err != nil {
		t.Fatalf("events: %v", err)
	}

	subscribe, err := terminal.Execute(ctx, withContext("sub-1", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_EventSubscribe{
		EventSubscribe: &apipb.EventSubscribeCommand{},
	}}))
	if err != nil {
		t.Fatalf("subscribe execute: %v", err)
	}
	subscription := subscribe.GetEventSubscription().GetSubscription()
	if len(subscription.GetOpaqueToken()) == 0 || subscription.GetKind() != apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION {
		t.Fatalf("subscription = %#v", subscription)
	}

	created, err := terminal.Execute(ctx, withContext("create-1", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalCreate{
		TerminalCreate: &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
			TerminalId: "term-adapter", Name: "adapter", Command: []string{"/bin/cat"},
			Size: &apipb.TerminalSize{Cols: 20, Rows: 5},
		}},
	}}))
	if err != nil {
		t.Fatalf("create execute: %v", err)
	}
	if created.GetRequestId() != "create-1" || created.GetOriginSession().GetEndpointId() != "local" {
		t.Fatalf("create envelope correlation = %#v", created)
	}
	info := created.GetTerminalCreate().GetTerminal()
	if info.GetRef().GetTerminalId() != "term-adapter" || info.GetRef().GetEndpointId() != "local" || info.GetSize().GetCols() != 20 {
		t.Fatalf("created terminal projection = %#v", info)
	}

	select {
	case event := <-events:
		lifecycle := event.GetTerminalLifecycle()
		if lifecycle == nil || lifecycle.GetTerminal().GetRef().GetTerminalId() != "term-adapter" {
			t.Fatalf("lifecycle event = %#v", event)
		}
		if string(event.GetSubscription().GetOpaqueToken()) != string(subscription.GetOpaqueToken()) {
			t.Fatalf("event subscription token = %q, want %q", event.GetSubscription().GetOpaqueToken(), subscription.GetOpaqueToken())
		}
		if event.GetOriginSession().GetEndpointId() != "local" {
			t.Fatalf("event origin session = %#v", event.GetOriginSession())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for terminal lifecycle event through adapter")
	}

	list, err := terminal.Execute(ctx, withContext("list-1", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}}))
	if err != nil || len(list.GetTerminalList().GetTerminals()) != 1 {
		t.Fatalf("list = %#v err=%v", list, err)
	}

	missing, err := terminal.Execute(ctx, withContext("get-missing", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalGet{
		TerminalGet: &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "missing"}},
	}}))
	if err != nil {
		t.Fatalf("get missing execute: %v", err)
	}
	if got := missing.GetError().GetCode(); got != apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND {
		t.Fatalf("missing terminal error code = %v, want NOT_FOUND (%#v)", got, missing.GetError())
	}

	unsupported, err := terminal.Execute(ctx, withContext("bad-1", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_CancelOperation{CancelOperation: &apipb.CancelOperationCommand{}}}))
	if err != nil {
		t.Fatalf("cancel execute: %v", err)
	}
	if got := unsupported.GetError().GetCode(); got != apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE || !unsupported.GetError().GetRetryable() {
		t.Fatalf("cancel error = %#v, want retryable UNAVAILABLE", unsupported.GetError())
	}

	released, err := terminal.Execute(ctx, withContext("release-1", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{
		ReleaseResource: &apipb.ReleaseResourceCommand{Resource: subscription},
	}}))
	if err != nil || released.GetAcknowledge() == nil {
		t.Fatalf("release subscription = %#v err=%v", released, err)
	}

	// storage-only 订阅由 access 本地处理，adapter 只返回可释放的合成 token。
	storageOnly, err := terminal.Execute(ctx, withContext("sub-storage", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_EventSubscribe{
		EventSubscribe: &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED}},
	}}))
	if err != nil {
		t.Fatalf("storage subscribe execute: %v", err)
	}
	storageSubscription := storageOnly.GetEventSubscription().GetSubscription()
	if len(storageSubscription.GetOpaqueToken()) == 0 {
		t.Fatalf("storage-only subscription token = %#v", storageSubscription)
	}
	if _, err := terminal.Execute(ctx, withContext("release-storage", &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{
		ReleaseResource: &apipb.ReleaseResourceCommand{Resource: storageSubscription},
	}})); err != nil {
		t.Fatalf("release storage subscription: %v", err)
	}
}
