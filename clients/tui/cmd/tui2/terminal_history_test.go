package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/history"
	"github.com/anytty/anytty/clients/tui/pty"
	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/proto/access/apipb"
	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

type blockingHistoryPTY struct {
	*fakePTY
	history.Backend
	entered chan struct{}
	release chan struct{}
}

// traceHistoryBackend is deliberately small: it provides a stable frozen
// tail and records no UI assumptions, so the test below can isolate routing
// from a real daemon or renderer.
type traceHistoryBackend struct {
	rows []*apipb.HistoryRow
}

func newTraceHistoryBackend() *traceHistoryBackend {
	b := &traceHistoryBackend{}
	for i := 1; i <= 32; i++ {
		b.rows = append(b.rows, &apipb.HistoryRow{
			LogicalLineId: uint64(i),
			Row:           &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: fmt.Sprintf("history-%02d", i), Width: 10}}},
		})
	}
	return b
}

func (b *traceHistoryBackend) Window(ctx context.Context, req *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	end := len(b.rows)
	if req.GetMode() == apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER {
		end = int(req.GetBeforeCursor().GetLineId()) - 1
	}
	if end < 0 {
		end = 0
	}
	if end > len(b.rows) {
		end = len(b.rows)
	}
	start := end - int(req.GetLimit())
	if start < 0 {
		start = 0
	}
	return &apipb.HistoryWindowResult{Token: "trace-token", HistoryGeneration: 1,
		Rows: b.rows[start:end], HasMore: start > 0}, nil
}

func (b *traceHistoryBackend) Search(context.Context, *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	return &apipb.HistorySearchResult{}, nil
}

func (b *traceHistoryBackend) Copy(context.Context, *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	return &apipb.HistoryCopyResult{Done: true}, nil
}

func (b *traceHistoryBackend) Release(context.Context, *apipb.HistoryReleaseCommand) error {
	return nil
}

type traceHistoryPTY struct {
	*fakePTY
	backend history.Backend
}

func (p *traceHistoryPTY) HistoryBackend(context.Context) (history.Backend, *apipb.TerminalRef, error) {
	return p.backend, &apipb.TerminalRef{TerminalId: "trace"}, nil
}

func (p *blockingHistoryPTY) HistoryBackend(context.Context) (history.Backend, *apipb.TerminalRef, error) {
	return p, &apipb.TerminalRef{TerminalId: "slow"}, nil
}

func (p *blockingHistoryPTY) Window(ctx context.Context, _ *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	close(p.entered)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.release:
		return &apipb.HistoryWindowResult{Token: "frozen"}, nil
	}
}

func (p *blockingHistoryPTY) Release(context.Context, *apipb.HistoryReleaseCommand) error { return nil }

func TestTerminalHistoryDoesNotBlockSiblingOrView(t *testing.T) {
	slow := &blockingHistoryPTY{fakePTY: newFakePTY(), entered: make(chan struct{}), release: make(chan struct{})}
	fast := newFakePTY()
	inner := runtime.NewTerminalHandler(runtime.TerminalOptions{Cols: 80, Rows: 12, NewPTY: func(cfg pty.Config) pty.PTY {
		if cfg.ID == "slow" {
			return slow
		}
		return fast
	}})
	defer inner.Close()
	defer close(slow.release)
	for _, id := range []string{"slow", "fast"} {
		out, _ := inner.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.attach"}, Params: &pb.MethodParams{Endpoint: "local", Id: id}})
		if !out.OK {
			t.Fatal(out.Error)
		}
	}
	output := newMemPipe()
	defer output.Close()
	host := &Host{urgentWake: make(chan struct{}, 1)}
	gate := &gateHandler{host: host, inner: inner}
	session := runtime.NewSession(runtime.Options{ViewID: "v", Cols: 80, Rows: 12, Handler: gate}, bytes.NewReader(nil), output)
	host.session = session
	responses := make(chan *pb.Response, 4)
	go func() {
		decoder := wire.NewDecoder(output, wire.RoleProgram, 0)
		for {
			_, msg, err := decoder.DecodeMessage()
			if err != nil {
				return
			}
			if response, ok := msg.(*pb.Response); ok {
				responses <- response
			}
		}
	}()
	// Use the actual RESULT path, not just a call to the background queue.
	returned := make(chan error, 1)
	go func() {
		returned <- session.HandleResult(&pb.Result{Epoch: 1, RequestId: 1, Method: "history.window", Params: &pb.MethodParams{Endpoint: "local", Id: "slow"}})
	}()
	select {
	case <-slow.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("history did not start")
	}
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RESULT reader blocked on history")
	}
	if err := session.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: &pb.Box{Id: "root"}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := session.Frame(); !ok {
		t.Fatal("VIEW was blocked by history")
	}
	if err := session.HandleResult(&pb.Result{Epoch: 1, RequestId: 2, Method: "history.window", Params: &pb.MethodParams{Endpoint: "local", Id: "fast"}}); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-responses:
		if response.GetRequestId() != 2 || !response.GetOk() {
			t.Fatalf("sibling response = %v", response)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("sibling terminal blocked by slow history")
	}
	select {
	case <-host.urgentWake:
		// The host repaints as soon as the viewport is published, before the
		// response round-trip reaches the layout program.
	case <-time.After(2 * time.Second):
		t.Fatal("history completion did not wake the host")
	}
	if session.Pending() != 1 {
		t.Fatalf("pending=%d; slow history should still be running", session.Pending())
	}
}

func TestTerminalHistoryRoutesReverseWheelWhileFirstRequestIsPending(t *testing.T) {
	slow := &blockingHistoryPTY{fakePTY: newFakePTY(), entered: make(chan struct{}), release: make(chan struct{})}
	inner := runtime.NewTerminalHandler(runtime.TerminalOptions{Cols: 80, Rows: 12, NewPTY: func(pty.Config) pty.PTY {
		return slow
	}})
	defer inner.Close()
	outcome, pending := inner.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.attach"}, Params: &pb.MethodParams{Endpoint: "remote", Id: "slow"}})
	if !outcome.OK || pending {
		t.Fatalf("attach = %+v pending=%v", outcome, pending)
	}
	term, ok := inner.TerminalAt("remote", "slow")
	if !ok {
		t.Fatal("attached terminal missing")
	}

	host := &Host{urgentWake: make(chan struct{}, 1)}
	gate := &gateHandler{host: host, inner: inner}
	session := runtime.NewSession(runtime.Options{
		ViewID: "v", Epoch: 1, Handler: gate,
		MouseTracking: func(string) bool { return true },
		HistoryActive: func(id, view string) bool { return id == term.SourceID() && term.HistoryRoutingActive(view) },
	}, bytes.NewReader(nil), io.Discard)
	host.session = session
	session.SetInputSink(inner)
	if err := session.SetSources([]*pb.Source{{Id: term.SourceID(), Kind: "terminal", Attached: true}}); err != nil {
		t.Fatal(err)
	}
	if err := session.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: &pb.Box{Id: "root", Children: []*pb.Box{{
		Id: "term", Focused: true, Input: []string{"wheel"}, Content: &pb.Content{Self: term.SourceID()},
	}}}}); err != nil {
		t.Fatal(err)
	}

	first, firstPending := gate.terminalHistory(runtime.Request{
		Epoch: 1, RequestID: 1, Method: runtime.Method{Name: "terminal.scroll"},
		Params: &pb.MethodParams{Endpoint: "remote", Id: "slow", Delta: 1, Rows: 8, View: "term"},
	})
	if first.OK || !firstPending {
		t.Fatalf("first scroll = %+v pending=%v, want async pending", first, firstPending)
	}
	select {
	case <-slow.entered:
	case <-time.After(time.Second):
		t.Fatal("first history request did not start")
	}
	if !term.HistoryRoutingActive("term") {
		t.Fatal("host routing was live while first remote scroll was pending")
	}
	if got := session.Route(runtime.InputEvent{Kind: runtime.InputWheel}); got != runtime.DestinationProgram {
		t.Fatalf("pending reverse wheel destination = %v, want program", got)
	}

	second, secondPending := gate.terminalHistory(runtime.Request{
		Epoch: 1, RequestID: 2, Method: runtime.Method{Name: "terminal.scroll"},
		Params: &pb.MethodParams{Endpoint: "remote", Id: "slow", Delta: -1, Rows: 8, View: "term"},
	})
	if second.OK || !secondPending {
		t.Fatalf("reverse scroll = %+v pending=%v, want async pending", second, secondPending)
	}
	if !term.HistoryRoutingActive("term") {
		t.Fatal("stale completion handed reverse wheel back to the PTY")
	}
	close(slow.release)
	waitFor(t, "latest reverse scroll completion", func() bool { return !term.HistoryRoutingActive("term") })
}

func TestTerminalHistoryLiveDownwardScrollIsImmediateNoOp(t *testing.T) {
	slow := &blockingHistoryPTY{fakePTY: newFakePTY(), entered: make(chan struct{}), release: make(chan struct{})}
	inner := runtime.NewTerminalHandler(runtime.TerminalOptions{Cols: 80, Rows: 12, NewPTY: func(pty.Config) pty.PTY {
		return slow
	}})
	defer inner.Close()
	outcome, pending := inner.Handle(runtime.Request{Method: runtime.Method{Name: "terminal.attach"}, Params: &pb.MethodParams{Endpoint: "remote", Id: "live"}})
	if !outcome.OK || pending {
		t.Fatalf("attach = %+v pending=%v", outcome, pending)
	}
	term, ok := inner.TerminalAt("remote", "live")
	if !ok {
		t.Fatal("attached terminal missing")
	}
	host := &Host{urgentWake: make(chan struct{}, 1)}
	gate := &gateHandler{host: host, inner: inner}
	host.session = runtime.NewSession(runtime.Options{ViewID: "v"}, bytes.NewReader(nil), io.Discard)

	result, pending := gate.terminalHistory(runtime.Request{
		Epoch: 1, RequestID: 1, Method: runtime.Method{Name: "terminal.scroll"},
		Params: &pb.MethodParams{Endpoint: "remote", Id: "live", Delta: -1, Rows: 8, View: "live"},
	})
	if !result.OK || pending {
		t.Fatalf("live downward scroll = %+v pending=%v, want immediate success", result, pending)
	}
	select {
	case <-slow.entered:
		t.Fatal("live downward scroll opened the persistent history backend")
	default:
	}
	if term.HistoryRoutingActive("live") {
		t.Fatal("live downward scroll left history routing active")
	}
}
