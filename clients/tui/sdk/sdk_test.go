package sdk

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func helloMsg(epoch uint64) *pb.Hello {
	return &pb.Hello{Schema: 1, ViewId: "view:test:1", Epoch: epoch, Cols: 100, Rows: 30}
}

// memPipe is an unbounded in-memory byte stream so a test host can write
// frames without a concurrent drainer and can close its end to unblock the
// client read loop.
type memPipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newMemPipe() *memPipe {
	p := &memPipe{}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *memPipe) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	p.buf = append(p.buf, b...)
	p.cond.Broadcast()
	return len(b), nil
}

func (p *memPipe) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(b, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *memPipe) Close() error {
	p.mu.Lock()
	p.closed = true
	p.cond.Broadcast()
	p.mu.Unlock()
	return nil
}

// pipePair wires a client to a test host: the host writes through enc and
// reads through dec, the client runs Loop on the other ends.
func pipePair(t *testing.T, handlers Handlers) (*Client, *wire.Encoder, *wire.Decoder) {
	t.Helper()
	hostToProg := newMemPipe()
	progToHost := newMemPipe()
	client := New(hostToProg, progToHost, handlers)
	go func() {
		_ = client.Loop()
	}()
	t.Cleanup(func() {
		_ = hostToProg.Close()
		_ = progToHost.Close()
	})
	return client, wire.NewEncoder(hostToProg, wire.RoleHost, 0), wire.NewDecoder(progToHost, wire.RoleHost, 0)
}

func readFrame(t *testing.T, dec *wire.Decoder) (wire.Type, []byte) {
	t.Helper()
	type result struct {
		t       wire.Type
		payload []byte
		err     error
	}
	ch := make(chan result, 1)
	go func() {
		typ, payload, err := dec.Decode()
		ch <- result{typ, payload, err}
	}()
	select {
	case got := <-ch:
		if got.err != nil {
			t.Fatalf("read frame: %v", got.err)
		}
		return got.t, got.payload
	case <-time.After(2 * time.Second):
		t.Fatal("timeout reading frame")
		return 0, nil
	}
}

func decodeView(t *testing.T, payload []byte) *pb.View {
	t.Helper()
	m, err := wire.UnmarshalPayload(wire.TypeView, payload)
	if err != nil {
		t.Fatalf("decode view: %v", err)
	}
	return m.(*pb.View)
}

func TestCommitRoundTrip(t *testing.T) {
	client, enc, dec := pipePair(t, Handlers{})
	if err := enc.Encode(wire.TypeHello, helloMsg(3)); err != nil {
		t.Fatalf("hello: %v", err)
	}
	waitFor(t, "hello applied", func() bool { return client.Epoch() == 3 })

	tree := Row(
		Text("one").ID("a").Width(10),
		Terminal("terminal:local:main").
			ID("term").
			Flex(1).
			Input("key", "paste", "wheel").
			Props(map[string]string{"chrome.border": "fg:#565f89", "chrome.title": "fg:#c0caf5"}).
			Focused(true),
		Stack(
			Text("bg").ID("bg"),
			Text("popup").ID("popup").Pos(5, 2).Style("fg:#a78bfa;bg:#161823;bold"),
		).ID("stack"),
	).ID("root")
	keys := Keys{Claim: []string{"ctrl-p", "?"}, All: false}
	if err := client.Commit(tree.Build(), keys); err != nil {
		t.Fatalf("commit: %v", err)
	}

	typ, payload := readFrame(t, dec)
	if typ != wire.TypeView {
		t.Fatalf("frame type = %v, want VIEW", typ)
	}
	view := decodeView(t, payload)
	if view.GetEpoch() != 3 || view.GetRev() != 1 {
		t.Fatalf("view epoch/rev = %d/%d, want 3/1", view.GetEpoch(), view.GetRev())
	}
	if got := view.GetKeys().GetClaim(); len(got) != 2 || got[0] != "ctrl-p" || got[1] != "?" {
		t.Fatalf("claim = %v", got)
	}
	root := view.GetRoot()
	if root.GetId() != "root" || root.GetFlow() != "row" || len(root.GetChildren()) != 3 {
		t.Fatalf("root = %+v", root)
	}
	term := root.GetChildren()[1]
	if term.GetContent().GetSelf() != "terminal:local:main" || !term.GetFocused() {
		t.Fatalf("terminal node = %+v", term)
	}
	if term.GetSize().GetFlex() != 1 || term.GetInput()[0] != "key" {
		t.Fatalf("terminal size/input = %+v %v", term.GetSize(), term.GetInput())
	}
	if props := term.GetContent().GetProps(); props["chrome.border"] != "fg:#565f89" || props["chrome.title"] != "fg:#c0caf5" {
		t.Fatalf("terminal props = %v, want the declared chrome styles", props)
	}
	popup := findBox(root, "popup")
	if popup == nil || popup.GetPos().GetX() != 5 || popup.GetPos().GetY() != 2 {
		t.Fatalf("popup = %+v", popup)
	}
	if popup.GetStyle() != "fg:#a78bfa;bg:#161823;bold" {
		t.Fatalf("popup style = %q, want the explicit style untouched", popup.GetStyle())
	}

	if err := client.Commit(Box().ID("root2").Build(), Keys{}); err != nil {
		t.Fatalf("second commit: %v", err)
	}
	view = decodeView(t, readFramePayload(t, dec))
	if view.GetRev() != 2 {
		t.Fatalf("second rev = %d, want 2", view.GetRev())
	}
}

func readFramePayload(t *testing.T, dec *wire.Decoder) []byte {
	t.Helper()
	_, payload := readFrame(t, dec)
	return payload
}

func findBox(root *pb.Box, id string) *pb.Box {
	if root == nil {
		return nil
	}
	if root.GetId() == id {
		return root
	}
	for _, child := range root.GetChildren() {
		if found := findBox(child, id); found != nil {
			return found
		}
	}
	return nil
}

func TestEmitCorrelatesResponses(t *testing.T) {
	client, enc, dec := pipePair(t, Handlers{})
	if err := enc.Encode(wire.TypeHello, helloMsg(1)); err != nil {
		t.Fatalf("hello: %v", err)
	}
	waitFor(t, "hello applied", func() bool { return client.Epoch() == 1 })

	var mu sync.Mutex
	got := map[string]string{}
	first, err := client.Emit("terminal.attach", &pb.MethodParams{Endpoint: "local", Id: "a"}, func(resp *pb.Response) {
		mu.Lock()
		got["first"] = resp.GetError()
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("emit first: %v", err)
	}
	second, err := client.Emit("terminal.scroll", &pb.MethodParams{Endpoint: "local", Id: "a", Delta: 3}, func(resp *pb.Response) {
		mu.Lock()
		got["second"] = strings.Join(resp.GetData().GetRows(), ",")
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("emit second: %v", err)
	}
	if second != first+1 {
		t.Fatalf("request ids = %d,%d, want consecutive", first, second)
	}

	for i := 0; i < 2; i++ {
		typ, payload := readFrame(t, dec)
		if typ != wire.TypeResult {
			t.Fatalf("frame %d type = %v, want RESULT", i, typ)
		}
		m, err := wire.UnmarshalPayload(wire.TypeResult, payload)
		if err != nil {
			t.Fatalf("decode result: %v", err)
		}
		result := m.(*pb.Result)
		if result.GetEpoch() != 1 || result.GetMethod() == "" {
			t.Fatalf("result = %+v", result)
		}
	}

	// Responses arrive out of order: the second request is answered first.
	if err := enc.Encode(wire.TypeResponse, &pb.Response{RequestId: second, Epoch: 1, Ok: true, Data: &pb.MethodData{Rows: []string{"x", "y"}}}); err != nil {
		t.Fatalf("response second: %v", err)
	}
	if err := enc.Encode(wire.TypeResponse, &pb.Response{RequestId: first, Epoch: 1, Ok: false, Error: "nope"}); err != nil {
		t.Fatalf("response first: %v", err)
	}
	waitFor(t, "responses delivered", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == 2
	})
	mu.Lock()
	defer mu.Unlock()
	if got["first"] != "nope" || got["second"] != "x,y" {
		t.Fatalf("callbacks = %v", got)
	}
	if client.Pending() != 0 {
		t.Fatalf("pending = %d, want 0", client.Pending())
	}
}

func TestEventDispatch(t *testing.T) {
	var (
		mu       sync.Mutex
		seen     []string
		resized  [2]int
		notice   string
		rejected string
	)
	mark := func(name string) {
		mu.Lock()
		seen = append(seen, name)
		mu.Unlock()
	}
	_, enc, _ := pipePair(t, Handlers{
		Hello: func(h *pb.Hello) { mark("hello") },
		Sources: func(items []*pb.Source) {
			mark("sources")
			if len(items) != 1 || items[0].GetId() != "terminal:local:main" {
				t.Errorf("sources = %+v", items)
			}
		},
		Key:   func(k *pb.KeyEvent) { mark("key:" + k.GetKey()) },
		Paste: func(p *pb.PasteEvent) { mark("paste:" + p.GetText()) },
		Mouse: func(m *pb.MouseEvent) { mark("mouse:" + m.GetNode()) },
		Wheel: func(w *pb.WheelEvent) { mark("wheel") },
		Resize: func(cols, rows int) {
			resized = [2]int{cols, rows}
			mark("resize")
		},
		Notice:       func(level, message string) { notice = message; mark("notice") },
		Component:    func(c *pb.ComponentEvent) { mark("component") },
		ViewRejected: func(epoch, rev uint64, reason string) { rejected = reason; mark("rejected") },
		Response:     func(r *pb.Response) { mark("response") },
	})
	if err := enc.Encode(wire.TypeHello, helloMsg(1)); err != nil {
		t.Fatalf("hello: %v", err)
	}
	events := []*pb.Event{
		{Event: &pb.Event_Sources{Sources: &pb.SourcesEvent{Items: []*pb.Source{{Id: "terminal:local:main", Kind: "terminal"}}}}},
		{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "ev-1", Key: "ctrl-p"}}},
		{Event: &pb.Event_Paste{Paste: &pb.PasteEvent{Id: "ev-2", Text: "hi"}}},
		{Event: &pb.Event_Mouse{Mouse: &pb.MouseEvent{Action: "press", Button: "left", X: 1, Y: 2, Node: "slot:0"}}},
		{Event: &pb.Event_Wheel{Wheel: &pb.WheelEvent{Delta: 1, X: 1, Y: 2, Node: "slot:0"}}},
		{Event: &pb.Event_Resize{Resize: &pb.ResizeEvent{Cols: 120, Rows: 40}}},
		{Event: &pb.Event_Notice{Notice: &pb.NoticeEvent{Level: "info", Message: "hint"}}},
		{Event: &pb.Event_Component{Component: &pb.ComponentEvent{Source: "picker:local:1", Name: "pick", Value: "x"}}},
		{Event: &pb.Event_ViewRejected{ViewRejected: &pb.ViewRejectedEvent{Epoch: 1, Rev: 9, Reason: "max_nodes"}}},
		{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "ev-3", Key: "esc"}}},
	}
	for _, ev := range events {
		if err := enc.Encode(wire.TypeEvent, ev); err != nil {
			t.Fatalf("encode event: %v", err)
		}
	}
	waitFor(t, "last event", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(seen) == len(events)+1
	})
	mu.Lock()
	defer mu.Unlock()
	want := []string{"hello", "sources", "key:ctrl-p", "paste:hi", "mouse:slot:0", "wheel", "resize", "notice", "component", "rejected", "key:esc"}
	if strings.Join(seen, "|") != strings.Join(want, "|") {
		t.Fatalf("dispatch order = %v, want %v", seen, want)
	}
	if resized != [2]int{120, 40} || notice != "hint" || rejected != "max_nodes" {
		t.Fatalf("resize=%v notice=%q rejected=%q", resized, notice, rejected)
	}
}

func TestViewBeforeHelloIsRejected(t *testing.T) {
	client, _, _ := pipePair(t, Handlers{})
	if err := client.Commit(Box().Build(), Keys{}); err != ErrNoHello {
		t.Fatalf("commit before hello = %v, want ErrNoHello", err)
	}
	if _, err := client.Emit("terminal.attach", nil, nil); err != ErrNoHello {
		t.Fatalf("emit before hello = %v, want ErrNoHello", err)
	}
}

func TestNewHelloResetsEpochAndRev(t *testing.T) {
	client, enc, dec := pipePair(t, Handlers{})
	if err := enc.Encode(wire.TypeHello, helloMsg(1)); err != nil {
		t.Fatalf("hello 1: %v", err)
	}
	waitFor(t, "epoch 1", func() bool { return client.Epoch() == 1 })
	if err := client.Commit(Box().Build(), Keys{}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	readFrame(t, dec)

	if err := enc.Encode(wire.TypeHello, helloMsg(2)); err != nil {
		t.Fatalf("hello 2: %v", err)
	}
	waitFor(t, "epoch 2", func() bool { return client.Epoch() == 2 })
	if client.Rev() != 0 {
		t.Fatalf("rev after hello = %d, want 0", client.Rev())
	}
	if err := client.Commit(Box().ID("first").Build(), Keys{}); err != nil {
		t.Fatalf("commit 2: %v", err)
	}
	view := decodeView(t, readFramePayload(t, dec))
	if view.GetEpoch() != 2 || view.GetRev() != 1 {
		t.Fatalf("restarted view epoch/rev = %d/%d, want 2/1", view.GetEpoch(), view.GetRev())
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

// TestBuilderPropsSnapshotIsIndependent checks that Props snapshots the map:
// mutating the caller's map or a previously built node never leaks into the
// next Build.
func TestBuilderPropsSnapshotIsIndependent(t *testing.T) {
	props := map[string]string{"chrome.border": "fg:#111111"}
	builder := Terminal("terminal:local:main").Props(props)
	props["chrome.border"] = "fg:#222222"
	props["chrome.title"] = "fg:#333333"

	node := builder.Build()
	if got := node.GetContent().GetProps()["chrome.border"]; got != "fg:#111111" {
		t.Fatalf("props after caller mutation = %q, want the declared value", got)
	}
	if _, ok := node.GetContent().GetProps()["chrome.title"]; ok {
		t.Fatal("props added after Props() must not appear in the node")
	}

	node.GetContent().Props["chrome.border"] = "fg:#444444"
	if got := builder.Build().GetContent().GetProps()["chrome.border"]; got != "fg:#111111" {
		t.Fatalf("props after node mutation = %q, want the builder snapshot", got)
	}
}
