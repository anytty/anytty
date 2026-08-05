package core

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestProtocolServerRequestBudgetDefaultsTo512(t *testing.T) {
	server := NewServer()
	if got := cap(server.protocolRequestSlots); got != 512 {
		t.Fatalf("protocol server request budget = %d, want 512", got)
	}
}

func TestProtocolServerRequestBudgetExhaustionAcrossConnectionsReleasesAndReusesID(t *testing.T) {
	server := NewServer(withProtocolRequestBudgetForTest(2))
	firstExecutor := newProtocolBudgetExecutor(2)
	secondExecutor := newProtocolBudgetExecutor(1)
	first, firstClient, firstDaemon := newProtocolBudgetSession(server, firstExecutor)
	second, secondClient, secondDaemon := newProtocolBudgetSession(server, secondExecutor)
	defer firstClient.Close()
	defer firstDaemon.Close()
	defer secondClient.Close()
	defer secondDaemon.Close()
	payload := protocolBudgetCommandPayload(t)

	handleProtocolRequestDirect(t, first, context.Background(), protocol.Request{ID: 1, Method: "api.execute", Params: payload})
	awaitProtocolBudgetStart(t, firstExecutor)
	handleProtocolRequestDirect(t, second, context.Background(), protocol.Request{ID: 2, Method: "api.execute", Params: payload})
	awaitProtocolBudgetStart(t, secondExecutor)

	handleProtocolRequestDirect(t, first, context.Background(), protocol.Request{ID: 3, Method: "api.execute", Params: payload})
	assertProtocolBudgetExhausted(t, firstClient, 3)
	if calls := firstExecutor.calls.Load(); calls != 1 {
		t.Fatalf("first executor calls after rejected request = %d, want 1", calls)
	}
	assertProtocolRequestState(t, first, 1, 1)
	assertProtocolRequestIDInactive(t, first, 3)

	secondExecutor.release <- struct{}{}
	assertProtocolResponseID(t, secondClient, 2)
	second.requests.Wait()
	if got := len(server.protocolRequestSlots); got != 1 {
		t.Fatalf("server slots after release = %d, want 1", got)
	}

	handleProtocolRequestDirect(t, first, context.Background(), protocol.Request{ID: 3, Method: "api.execute", Params: payload})
	awaitProtocolBudgetStart(t, firstExecutor)
	if calls := firstExecutor.calls.Load(); calls != 2 {
		t.Fatalf("first executor calls after ID reuse = %d, want 2", calls)
	}
	firstExecutor.release <- struct{}{}
	firstExecutor.release <- struct{}{}
	assertProtocolResponseIDs(t, firstClient, 1, 3)
	first.requests.Wait()
	assertProtocolBudgetIdle(t, server, first, second)
}

func TestProtocolServerRequestBudgetCancellationAndConnectionCloseRelease(t *testing.T) {
	t.Run("request cancellation", func(t *testing.T) {
		server := NewServer(withProtocolRequestBudgetForTest(1))
		firstExecutor := newProtocolBudgetExecutor(1)
		first, firstClient, firstDaemon := newProtocolBudgetSession(server, firstExecutor)
		defer firstClient.Close()
		defer firstDaemon.Close()
		payload := protocolBudgetCommandPayload(t)

		requestCtx, cancel := context.WithCancel(context.Background())
		handleProtocolRequestDirect(t, first, requestCtx, protocol.Request{ID: 1, Method: "api.execute", Params: payload})
		awaitProtocolBudgetStart(t, firstExecutor)
		cancel()
		assertProtocolResponseID(t, firstClient, 1)
		first.requests.Wait()
		if got := len(server.protocolRequestSlots); got != 0 {
			t.Fatalf("server slots after cancellation = %d, want 0", got)
		}

		secondExecutor := newProtocolBudgetExecutor(1)
		second, secondClient, secondDaemon := newProtocolBudgetSession(server, secondExecutor)
		defer secondClient.Close()
		defer secondDaemon.Close()
		handleProtocolRequestDirect(t, second, context.Background(), protocol.Request{ID: 2, Method: "api.execute", Params: payload})
		awaitProtocolBudgetStart(t, secondExecutor)
		secondExecutor.release <- struct{}{}
		assertProtocolResponseID(t, secondClient, 2)
		second.requests.Wait()
		assertProtocolBudgetIdle(t, server, first, second)
	})

	t.Run("request cancel frame", func(t *testing.T) {
		server := NewServer(withProtocolRequestBudgetForTest(1))
		executor := newProtocolBudgetExecutor(1)
		session, client, daemon := newProtocolBudgetSession(server, executor)
		defer client.Close()
		defer daemon.Close()
		handleProtocolRequestDirect(t, session, context.Background(), protocol.Request{ID: 7, Method: "api.execute", Params: protocolBudgetCommandPayload(t)})
		awaitProtocolBudgetStart(t, executor)
		cancelPayload, err := protocol.EncodeRequestCancelPayload(7)
		if err != nil {
			t.Fatal(err)
		}
		if err := session.handleControlFrame(context.Background(), wire.TypeRequestCancel, cancelPayload); err != nil {
			t.Fatal(err)
		}
		assertProtocolResponseID(t, client, 7)
		session.requests.Wait()
		assertProtocolBudgetIdle(t, server, session)

		if err := session.handleControlFrame(context.Background(), wire.TypeRequestCancel, cancelPayload); err != nil {
			t.Fatalf("late duplicate cancel should be a no-op: %v", err)
		}
	})

	t.Run("connection close", func(t *testing.T) {
		server := NewServer(withProtocolRequestBudgetForTest(1))
		executor := newProtocolBudgetExecutor(1)
		client, daemon := memory.NewPair()
		defer daemon.Close()
		session := newProtocolSession(server, daemon, fullDaemonTransportScope())
		session.application = executor
		done := make(chan error, 1)
		go func() { done <- session.run(context.Background()) }()
		completeServerProtocolHello(t, client)
		sendProtocolRequest(t, client, protocol.Request{ID: 1, Method: "api.execute", Params: protocolBudgetCommandPayload(t)})
		awaitProtocolBudgetStart(t, executor)
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("session exit after connection close = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("session did not stop after connection close")
		}
		if got := len(server.protocolRequestSlots); got != 0 {
			t.Fatalf("server slots after connection close = %d, want 0", got)
		}
		assertProtocolSessionRequestIdle(t, session)
	})
}

func TestProtocolServerRequestBudgetSessionLimit64WinsWithoutConsumingServerSlot(t *testing.T) {
	server := NewServer(withProtocolRequestBudgetForTest(65))
	executor := newProtocolBudgetExecutor(65)
	first, firstClient, firstDaemon := newProtocolBudgetSession(server, executor)
	second, secondClient, secondDaemon := newProtocolBudgetSession(server, executor)
	defer firstClient.Close()
	defer firstDaemon.Close()
	defer secondClient.Close()
	defer secondDaemon.Close()
	payload := protocolBudgetCommandPayload(t)

	for id := uint64(1); id <= 64; id++ {
		handleProtocolRequestDirect(t, first, context.Background(), protocol.Request{ID: id, Method: "api.execute", Params: payload})
		awaitProtocolBudgetStart(t, executor)
	}
	handleProtocolRequestDirect(t, first, context.Background(), protocol.Request{ID: 65, Method: "api.execute", Params: payload})
	assertProtocolBudgetExhausted(t, firstClient, 65)
	if calls := executor.calls.Load(); calls != 64 {
		t.Fatalf("executor calls after session rejection = %d, want 64", calls)
	}
	if got := len(server.protocolRequestSlots); got != 64 {
		t.Fatalf("server slots after session rejection = %d, want 64", got)
	}

	handleProtocolRequestDirect(t, second, context.Background(), protocol.Request{ID: 1, Method: "api.execute", Params: payload})
	awaitProtocolBudgetStart(t, executor)
	if got := len(server.protocolRequestSlots); got != 65 {
		t.Fatalf("server slots after second connection request = %d, want 65", got)
	}
	for range 65 {
		executor.release <- struct{}{}
	}
	for range 64 {
		assertProtocolResponse(t, firstClient)
	}
	assertProtocolResponseID(t, secondClient, 1)
	first.requests.Wait()
	second.requests.Wait()
	assertProtocolBudgetIdle(t, server, first, second)
}

func TestProtocolServerRequestBudgetRejectsExecutionAfterShutdown(t *testing.T) {
	server := NewServer(withProtocolRequestBudgetForTest(1))
	executor := newProtocolBudgetExecutor(1)
	session, client, daemon := newProtocolBudgetSession(server, executor)
	defer client.Close()
	defer daemon.Close()
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	handleProtocolRequestDirect(t, session, context.Background(), protocol.Request{ID: 1, Method: "api.execute", Params: protocolBudgetCommandPayload(t)})
	assertProtocolBudgetExhausted(t, client, 1)
	handleProtocolRequestDirect(t, session, context.Background(), protocol.Request{ID: 1, Method: "api.execute", Params: protocolBudgetCommandPayload(t)})
	assertProtocolBudgetExhausted(t, client, 1)
	if calls := executor.calls.Load(); calls != 0 {
		t.Fatalf("executor calls after shutdown = %d, want 0", calls)
	}
	assertProtocolBudgetIdle(t, server, session)
}

type protocolBudgetExecutor struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func newProtocolBudgetExecutor(capacity int) *protocolBudgetExecutor {
	return &protocolBudgetExecutor{
		started: make(chan struct{}, capacity),
		release: make(chan struct{}, capacity),
	}
}

func (executor *protocolBudgetExecutor) Execute(ctx context.Context, _ *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	executor.calls.Add(1)
	executor.started <- struct{}{}
	select {
	case <-executor.release:
	case <-ctx.Done():
	}
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
}

func withProtocolRequestBudgetForTest(budget int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.protocolRequestBudget = budget
	}
}

func newProtocolBudgetSession(server *Server, executor ApplicationExecutor) (*protocolSession, *memory.Transport, *memory.Transport) {
	client, daemon := memory.NewPair()
	session := newProtocolSession(server, daemon, fullDaemonTransportScope())
	session.helloAccepted = true
	session.application = executor
	return session, client, daemon
}

func protocolBudgetCommandPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func awaitProtocolBudgetStart(t *testing.T, executor *protocolBudgetExecutor) {
	t.Helper()
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("request did not enter executor")
	}
}

func assertProtocolBudgetExhausted(t *testing.T, client *memory.Transport, id uint64) {
	t.Helper()
	_, typ, payload := receiveProtocolFrame(t, client)
	if typ != wire.TypeError {
		t.Fatalf("exhausted response type = %d", typ)
	}
	message, err := protocol.DecodeErrorPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if message.ID != id || message.Error.Code != protocolErrorExhausted || message.Error.Message != protocolRequestCapacityExhaustedMessage {
		t.Fatalf("exhausted response = %#v", message)
	}
}

func assertProtocolResponse(t *testing.T, client *memory.Transport) uint64 {
	t.Helper()
	_, typ, payload := receiveProtocolFrame(t, client)
	if typ != wire.TypeResponse {
		t.Fatalf("response type = %d", typ)
	}
	response, err := protocol.DecodeResponsePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	return response.ID
}

func assertProtocolResponseID(t *testing.T, client *memory.Transport, id uint64) {
	t.Helper()
	if got := assertProtocolResponse(t, client); got != id {
		t.Fatalf("response ID = %d, want %d", got, id)
	}
}

func assertProtocolResponseIDs(t *testing.T, client *memory.Transport, ids ...uint64) {
	t.Helper()
	want := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	for range ids {
		delete(want, assertProtocolResponse(t, client))
	}
	if len(want) != 0 {
		t.Fatalf("missing response IDs: %v", want)
	}
}

func assertProtocolBudgetIdle(t *testing.T, server *Server, sessions ...*protocolSession) {
	t.Helper()
	if got := len(server.protocolRequestSlots); got != 0 {
		t.Fatalf("server request slots = %d, want 0", got)
	}
	for _, session := range sessions {
		assertProtocolSessionRequestIdle(t, session)
	}
}

func assertProtocolSessionRequestIdle(t *testing.T, session *protocolSession) {
	t.Helper()
	assertProtocolRequestState(t, session, 0, 0)
}

func assertProtocolRequestState(t *testing.T, session *protocolSession, slots, active int) {
	t.Helper()
	session.requestMu.Lock()
	activeCount := len(session.activeRequests)
	session.requestMu.Unlock()
	if slotCount := len(session.requestSlots); slotCount != slots || activeCount != active {
		t.Fatalf("session request state = slots:%d active:%d, want %d/%d", slotCount, activeCount, slots, active)
	}
}

func assertProtocolRequestIDInactive(t *testing.T, session *protocolSession, id uint64) {
	t.Helper()
	session.requestMu.Lock()
	_, active := session.activeRequests[id]
	session.requestMu.Unlock()
	if active {
		t.Fatalf("request ID %d remains active", id)
	}
}
