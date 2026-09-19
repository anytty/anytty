package runtime

import (
	"fmt"
	"sync"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Request is one whitelisted RESULT call handed to a Handler.
type Request struct {
	Epoch     uint64
	RequestID uint64
	Method    Method
	Params    *pb.MethodParams
}

// Outcome is a completed method execution.
type Outcome struct {
	OK    bool
	Data  *pb.MethodData
	Error string
}

// Handler executes whitelisted methods. Returning pending=true means the
// response will be delivered later through Session.Complete; the request
// stays in flight and counts against max_inflight_requests until then.
//
// The Handler is where a real host plugs in terminal components. This core
// only ships StubHandler.
type Handler interface {
	Handle(Request) (Outcome, bool)
}

// StubHandler is the default no-PTY executor: it validates nothing (the
// session did), fabricates terminal.create ids and serves configured
// scrollback rows and clipboard text.
type StubHandler struct {
	mu        sync.Mutex
	seq       uint64
	Clipboard string
	// Rows maps "endpoint:id" to the rows returned by terminal.scroll and
	// history.window.
	Rows map[string][]string
}

// Handle implements Handler.
func (h *StubHandler) Handle(req Request) (Outcome, bool) {
	switch req.Method.Name {
	case "terminal.create":
		endpoint := req.Params.GetEndpoint()
		h.mu.Lock()
		h.seq++
		id := fmt.Sprintf("terminal:%s:stub-%d", endpoint, h.seq)
		h.mu.Unlock()
		return Outcome{OK: true, Data: &pb.MethodData{Endpoint: endpoint, Id: id}}, false
	case "terminal.scroll", "history.window":
		key := req.Params.GetEndpoint() + ":" + req.Params.GetId()
		h.mu.Lock()
		rows := append([]string(nil), h.Rows[key]...)
		h.mu.Unlock()
		return Outcome{OK: true, Data: &pb.MethodData{Rows: rows}}, false
	case "clipboard.read":
		h.mu.Lock()
		text := h.Clipboard
		h.mu.Unlock()
		return Outcome{OK: true, Data: &pb.MethodData{Text: text}}, false
	default:
		return Outcome{OK: true}, false
	}
}
