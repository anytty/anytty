package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestApplicationExecutorPanicReturnsGeneric500AndKeepsConnectionAlive(t *testing.T) {
	var logs bytes.Buffer
	executor := &panicOnceExecutor{}
	sessionReady := make(chan *protocolSession, 1)
	server := NewServer(
		WithHistoryDisabled(),
		WithLogger(slog.New(slog.NewTextHandler(&logs, nil))),
		WithApplicationExecutorFactory(func(port ApplicationSessionPort) ApplicationExecutor {
			sessionReady <- port.(*protocolSession)
			return executor
		}),
	)
	client, daemon := memory.NewPair()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.ServeTransport(context.Background(), daemon) }()
	session := awaitApplicationTestResult(t, sessionReady, "protocol session was not created")
	completeServerProtocolHello(t, client)
	payload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}

	sendProtocolRequest(t, client, protocol.Request{ID: 41, Method: "api.execute", Params: payload})
	_, typ, responsePayload := receiveProtocolFrame(t, client)
	if typ != wire.TypeError {
		t.Fatalf("panic response type = %d, want error", typ)
	}
	response, err := protocol.DecodeErrorPayload(responsePayload)
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != 41 || response.Error.Code != protocolErrorInternal || response.Error.Message != errApplicationExecutorPanic.Error() {
		t.Fatalf("panic response = %#v", response)
	}
	if strings.Contains(response.Error.Message, panicOnceSecret) {
		t.Fatal("panic value leaked to client")
	}
	session.requests.Wait()
	assertProtocolSessionRequestStateIdle(t, server, session)

	sendProtocolRequest(t, client, protocol.Request{ID: 42, Method: "api.execute", Params: payload})
	_, typ, responsePayload = receiveProtocolFrame(t, client)
	if typ != wire.TypeResponse {
		t.Fatalf("post-panic response type = %d, want response", typ)
	}
	decoded, err := protocol.DecodeResponsePayload(responsePayload)
	if err != nil || decoded.ID != 42 {
		t.Fatalf("post-panic response = %#v, %v", decoded, err)
	}
	session.requests.Wait()
	assertProtocolSessionRequestStateIdle(t, server, session)
	if executor.calls.Load() != 2 {
		t.Fatalf("executor calls = %d, want 2", executor.calls.Load())
	}
	logOutput := logs.String()
	if !strings.Contains(logOutput, "application executor panicked") || !strings.Contains(logOutput, "goroutine") || strings.Contains(logOutput, panicOnceSecret) {
		t.Fatalf("panic log did not contain fixed message and stack without value: %q", logOutput)
	}

	_ = client.Close()
	if err := awaitApplicationTestResult(t, serveDone, "post-panic connection did not finish"); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("post-panic transport error = %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error = %v", err)
	}
}

func TestOversizedApplicationResultReturnsTyped429AndKeepsConnectionAlive(t *testing.T) {
	executor := &oversizedOnceExecutor{}
	sessionReady := make(chan *protocolSession, 1)
	server := NewServer(
		WithHistoryDisabled(),
		WithApplicationExecutorFactory(func(port ApplicationSessionPort) ApplicationExecutor {
			sessionReady <- port.(*protocolSession)
			return executor
		}),
	)
	client, daemon := memory.NewPair()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.ServeTransport(context.Background(), daemon) }()
	session := awaitApplicationTestResult(t, sessionReady, "protocol session was not created")
	completeServerProtocolHello(t, client)
	payload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}

	sendProtocolRequest(t, client, protocol.Request{ID: 51, Method: "api.execute", Params: payload})
	_, typ, responsePayload := receiveProtocolFrame(t, client)
	if typ != wire.TypeError {
		t.Fatalf("oversized response type = %d, want error", typ)
	}
	response, err := protocol.DecodeErrorPayload(responsePayload)
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != 51 || response.Error.Code != protocolErrorExhausted || !strings.Contains(response.Error.Message, protocol.ErrApplicationResultTooLarge.Error()) {
		t.Fatalf("oversized response = %#v", response)
	}
	session.requests.Wait()
	assertProtocolSessionRequestStateIdle(t, server, session)

	sendProtocolRequest(t, client, protocol.Request{ID: 52, Method: "api.execute", Params: payload})
	_, typ, responsePayload = receiveProtocolFrame(t, client)
	if typ != wire.TypeResponse {
		t.Fatalf("post-oversize response type = %d, want response", typ)
	}
	decoded, err := protocol.DecodeResponsePayload(responsePayload)
	if err != nil || decoded.ID != 52 {
		t.Fatalf("post-oversize response = %#v, %v", decoded, err)
	}
	session.requests.Wait()
	assertProtocolSessionRequestStateIdle(t, server, session)

	_ = client.Close()
	if err := awaitApplicationTestResult(t, serveDone, "post-oversize connection did not finish"); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("post-oversize transport error = %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error = %v", err)
	}
}

func assertProtocolSessionRequestStateIdle(t *testing.T, server *Server, session *protocolSession) {
	t.Helper()
	if got := len(server.protocolRequestSlots); got != 0 {
		t.Fatalf("server request slots = %d, want 0", got)
	}
	if got := len(session.requestSlots); got != 0 {
		t.Fatalf("session request slots = %d, want 0", got)
	}
	session.requestMu.Lock()
	active := len(session.activeRequests)
	session.requestMu.Unlock()
	if active != 0 {
		t.Fatalf("active request IDs = %d, want 0", active)
	}
}

func awaitApplicationTestResult[T any](t *testing.T, result <-chan T, failure string) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal(failure)
		var zero T
		return zero
	}
}

const panicOnceSecret = "panic-secret-must-not-leak"

type panicOnceExecutor struct {
	calls atomic.Int32
}

type oversizedOnceExecutor struct {
	calls atomic.Int32
}

func (executor *oversizedOnceExecutor) Execute(context.Context, *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	if executor.calls.Add(1) == 1 {
		return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryCopy{HistoryCopy: &apipb.HistoryCopyResult{
			Text: strings.Repeat("x", protocol.MaxApplicationResultEnvelopeBytes),
		}}}
	}
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
}

func (executor *panicOnceExecutor) Execute(context.Context, *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	if executor.calls.Add(1) == 1 {
		panic(panicOnceSecret)
	}
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
}
