package main

import (
	"context"
	"log"
	"time"

	"github.com/anytty/anytty/clients/tui/runtime"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// terminalHistory uses a bounded FIFO owned by the terminal object. A slow
// history request cannot block the protocol reader, VIEW commits or a sibling
// terminal. CompleteForEpoch protects a restarted program's reused IDs.
func (g *gateHandler) terminalHistory(req runtime.Request) (runtime.Outcome, bool) {
	term, ok := g.inner.TerminalBySource(runtime.SourceID(req.Params.GetEndpoint(), req.Params.GetId()))
	if !ok {
		return runtime.Outcome{Error: "no such terminal"}, false
	}
	session := g.host.currentSession()
	if session == nil {
		return runtime.Outcome{Error: "no active session"}, false
	}
	// A live terminal is already at the bottom. Treat a downward scroll as a
	// synchronous no-op until a history request is active. This keeps an
	// accidental wheel event from opening the persistent-history queue and
	// publishing a second "latest" frame that races a mouse-aware child TUI's
	// redraw. When an upward request is pending, keep the async path so a
	// reverse wheel can cancel/supersede that request correctly.
	if req.Method.Name == "terminal.scroll" && req.Params.GetDelta() < 0 && !term.HistoryRoutingActive() {
		rows, offset := term.Window(0, int(req.Params.GetRows()))
		if wheelDebugEnabled() {
			log.Printf("tui2 history no-op endpoint=%s id=%s delta=%d rows=%d offset=%d",
				req.Params.GetEndpoint(), req.Params.GetId(), req.Params.GetDelta(), len(rows), offset)
		}
		return runtime.Outcome{OK: true, Data: &pb.MethodData{Rows: rows, Offset: int32(offset)}}, false
	}
	if wheelDebugEnabled() {
		log.Printf("tui2 history %s endpoint=%s id=%s delta=%d rows=%d routing=%v",
			req.Method.Name, req.Params.GetEndpoint(), req.Params.GetId(),
			req.Params.GetDelta(), req.Params.GetRows(), term.HistoryRoutingActive())
	}
	// A local PTY already has its complete scrollback in the parser. Running
	// this tiny operation inline removes the queue/goroutine hop from the hot
	// wheel path; persistent endpoint history keeps the asynchronous path so a
	// network call can never stall the protocol reader.
	if req.Method.Name == "terminal.scroll" && !term.HasPersistentHistory() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		outcome, _ := g.inner.HandleContext(ctx, req)
		if outcome.OK {
			g.host.signalWakeImmediate()
		}
		return outcome, false
	}
	token := term.BeginHistoryScroll(int(req.Params.GetDelta()))
	work := func(parent context.Context) {
		defer term.EndHistoryScroll(token)
		if session.Epoch() != req.Epoch {
			return
		}
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		outcome, _ := g.inner.HandleContext(ctx, req)
		// HistoryScroll publishes the terminal viewport before the protocol
		// response is written. Wake the host at that point so the content can
		// repaint without waiting for the layout program to receive the
		// response, update its badge, and commit another VIEW.
		if outcome.OK {
			g.host.signalWakeImmediate()
		}
		errMsg := outcome.Error
		if !outcome.OK && errMsg == "" {
			errMsg = "rejected"
		}
		_ = session.CompleteForEpoch(req.Epoch, req.RequestID, outcome.Data, errMsg)
	}
	var accepted bool
	if req.Method.Name == "terminal.scroll" {
		accepted = term.EnqueueLatestHistory(work)
	} else {
		accepted = term.EnqueueHistory(work)
	}
	if !accepted {
		term.EndHistoryScroll(token)
		return runtime.Outcome{Error: "terminal history queue full"}, false
	}
	return runtime.Outcome{}, true
}
