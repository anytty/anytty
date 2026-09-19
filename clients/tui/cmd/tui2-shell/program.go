package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// emitter is the slice of the protocol client the program needs; tests inject
// a fake to drive the binding flow without a host.
type emitter interface {
	Hello() *pb.Hello
	Epoch() uint64
	Commit(root *pb.Box, keys sdk.Keys) error
	Emit(method string, params *pb.MethodParams, onResponse func(*pb.Response)) (uint64, error)
}

// program glues the pure model to the protocol client: events in, requests
// and a fresh view out after every state change.
//
// The mutex serializes model access between protocol handlers (Loop
// goroutine) and the optional clock ticker, so a status-line redraw can never
// race a state transition.
type program struct {
	mu    sync.Mutex
	model *model
	em    emitter
}

func newProgram(m *model, em emitter) *program {
	return &program{model: m, em: em}
}

func (p *program) handlers() sdk.Handlers {
	return sdk.Handlers{
		Hello:        p.onHello,
		Sources:      p.onSources,
		Key:          p.onKey,
		Paste:        p.onPaste,
		Mouse:        p.onMouse,
		Wheel:        p.onWheel,
		Resize:       p.onResize,
		Notice:       p.onNotice,
		ViewRejected: p.onViewRejected,
	}
}

func (p *program) onHello(h *pb.Hello) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model.hello(h)
	// Register configured daemon endpoints with the host so their terminal
	// inventories arrive as sources (ENDPOINTS.zh-CN.md §3).
	p.apply(p.model.syncEndpoints())
}

func (p *program) onSources(items []*pb.Source) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.apply(p.model.sourcesEvent(items))
}

func (p *program) onKey(k *pb.KeyEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.apply(p.model.key(keyEvent{ID: k.GetId(), Key: k.GetKey(), Char: k.GetChar()}))
}

// onPaste: in NORMAL the host writes a paste straight into the focused PTY,
// so a paste reaching the program belongs to a mode without focus. It is
// forwarded as the complete original block (never re-assembled).
func (p *program) onPaste(ev *pb.PasteEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.model.mode == modePrompt {
		p.model.promptText += ev.GetText()
		p.commit()
		return
	}
	slot := p.model.focusSlot()
	if slot.sourceID == "" || ev.GetId() == "" {
		p.commit()
		return
	}
	p.apply([]request{{
		Method: "input.forward",
		Params: &pb.MethodParams{EventId: ev.GetId(), Source: slot.sourceID},
	}})
}

func (p *program) onMouse(m *pb.MouseEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.apply(p.model.mouse(m.GetAction(), m.GetButton(), m.GetNode(), int(m.GetX()), int(m.GetY())))
}

func (p *program) onWheel(w *pb.WheelEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.apply(p.model.wheel(w.GetNode(), int(w.GetDelta())))
}

func (p *program) onResize(cols, rows int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model.resize(cols, rows)
	p.commit()
}

func (p *program) onNotice(level, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model.notice(level, message)
	p.commit()
}

func (p *program) onViewRejected(epoch, rev uint64, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model.status = "view rejected: " + reason
	p.commit()
}

// startClock redraws the status line periodically so the clock keeps
// ticking even without input. The host diffs frames, so a tick that does not
// change the clock costs nothing on the wire.
func (p *program) startClock(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			p.tick()
		}
	}()
}

func (p *program) tick() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.em.Hello() == nil {
		return
	}
	p.commit()
}

func (p *program) apply(reqs []request) {
	for _, req := range reqs {
		if _, err := p.em.Emit(req.Method, req.Params, p.wrap(req.After)); err != nil {
			p.model.status = "emit failed: " + err.Error()
		}
	}
	p.commit()
}

// wrap locks around the response callback: the client invokes it on the
// Loop goroutine, while the clock ticker may be mid-commit.
func (p *program) wrap(after func(*pb.Response)) func(*pb.Response) {
	if after == nil {
		return nil
	}
	return func(resp *pb.Response) {
		p.mu.Lock()
		defer p.mu.Unlock()
		after(resp)
		p.commit()
	}
}

// commit sends the current model as a view. The caller holds p.mu. A pressed
// title-bar button is a one-frame highlight: the frame that carries it was
// just committed, so the mark is cleared right after the redraw.
func (p *program) commit() {
	root, keys := p.model.view()
	if err := p.em.Commit(root.Build(), keys); err != nil {
		fmt.Fprintln(os.Stderr, "tui2-shell: commit:", err)
	}
	p.model.pressed = ""
}
