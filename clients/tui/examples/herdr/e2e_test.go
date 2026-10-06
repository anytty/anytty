package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	apipb "github.com/anytty/anytty/proto/access/apipb"
	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	gproto "google.golang.org/protobuf/proto"
)

// viewCache is the host side's copy of the committed view. It starts from the
// full VIEW and applies every VIEW_DELTA, so the test asserts on the tree the
// host would actually hold.
type viewCache struct {
	root *pb.Box
	rev  uint64
}

func (c *viewCache) apply(t *testing.T, typ wire.Type, payload []byte) {
	t.Helper()
	switch typ {
	case wire.TypeView:
		view := decodePayload(t, typ, payload).(*pb.View)
		c.root, c.rev = view.GetRoot(), view.GetRev()
	case wire.TypeViewDelta:
		delta := decodePayload(t, typ, payload).(*pb.ViewDelta)
		if delta.GetRevBase() != c.rev {
			t.Fatalf("delta rev_base = %d, cached rev = %d", delta.GetRevBase(), c.rev)
		}
		c.root = applyDelta(t, c.root, delta.GetPatches())
		c.rev = delta.GetRev()
	default:
		t.Fatalf("cannot apply frame %v", typ)
	}
}

func applyDelta(t *testing.T, root *pb.Box, patches []*pb.Patch) *pb.Box {
	t.Helper()
	for _, patch := range patches {
		path := patch.GetPath()
		switch patch.GetOp() {
		case "insert":
			parent := boxAtPath(t, root, path)
			index := int(patch.GetIndex())
			if index > len(parent.Children) {
				index = len(parent.Children)
			}
			parent.Children = append(parent.Children, nil)
			copy(parent.Children[index+1:], parent.Children[index:])
			parent.Children[index] = patch.GetBox()
		case "remove":
			if len(path) == 0 {
				t.Fatal("remove patch on the root")
			}
			parent := boxAtPath(t, root, path[:len(path)-1])
			index := int(path[len(path)-1])
			if index >= len(parent.Children) {
				t.Fatalf("remove index %d out of range", index)
			}
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
		case "replace":
			if len(path) == 0 {
				root = patch.GetBox()
				continue
			}
			parent := boxAtPath(t, root, path[:len(path)-1])
			index := int(path[len(path)-1])
			if index >= len(parent.Children) {
				t.Fatalf("replace index %d out of range", index)
			}
			parent.Children[index] = patch.GetBox()
		case "set":
			mergeBox(boxAtPath(t, root, path), patch.GetBox())
		default:
			t.Fatalf("unknown patch op %q", patch.GetOp())
		}
	}
	return root
}

func boxAtPath(t *testing.T, root *pb.Box, path []uint32) *pb.Box {
	t.Helper()
	node := root
	for _, index := range path {
		if index >= uint32(len(node.GetChildren())) {
			t.Fatalf("patch path %v leaves the tree", path)
		}
		node = node.GetChildren()[index]
	}
	return node
}

// mergeBox applies a "set" patch (non-children fields only) to a node.
func mergeBox(dst, patch *pb.Box) {
	if dst == nil || patch == nil {
		return
	}
	if patch.GetId() != "" {
		dst.Id = patch.GetId()
	}
	if patch.GetFlow() != "" {
		dst.Flow = patch.GetFlow()
	}
	if patch.GetStyle() != "" {
		dst.Style = patch.GetStyle()
	}
	if patch.GetFocused() {
		dst.Focused = true
	}
	if patch.Visible != nil {
		dst.Visible = patch.Visible
	}
	if len(patch.GetInput()) > 0 {
		dst.Input = patch.GetInput()
	}
	if patch.GetSize() != nil {
		dst.Size = patch.GetSize()
	}
	if patch.GetPos() != nil {
		dst.Pos = patch.GetPos()
	}
	if patch.GetCursor() != nil {
		dst.Cursor = patch.GetCursor()
	}
	if patch.GetContent() != nil {
		dst.Content = patch.GetContent()
	}
}

// --- scripted host ---

type scriptHost struct {
	t       *testing.T
	enc     *wire.Encoder
	dec     *wire.Decoder
	cache   *viewCache
	layout  string // storage "layout" value ("" = NOT_FOUND)
	pin     string // storage "pinned" value ("" = NOT_FOUND)
	creates int
	puts    []string
	methods []string
	gets    int
	quit    bool
	errCh   chan error // program exit, for timeout diagnostics
}

func (h *scriptHost) sendHello(epoch uint64) {
	h.t.Helper()
	hello := &pb.Hello{
		Schema: 1, ViewId: fmt.Sprintf("view-%d", epoch), Epoch: epoch, Cols: 100, Rows: 30,
		Features: map[string]bool{"view_delta": true},
		Methods: []string{
			"terminal.create", "terminal.attach", "terminal.restart", "terminal.kill",
			"terminal.remove", "terminal.scroll", "terminal.scrollEnd", "terminal.copy",
			"access.call", "system.quit",
		},
	}
	if err := h.enc.Encode(wire.TypeHello, hello); err != nil {
		h.t.Fatalf("hello: %v", err)
	}
}

func (h *scriptHost) key(id, k string) {
	h.t.Helper()
	ev := &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: id, Key: k}}}
	if err := h.enc.Encode(wire.TypeEvent, ev); err != nil {
		h.t.Fatalf("key %s: %v", k, err)
	}
}

func (h *scriptHost) mouse(action, button, node string, x, y int) {
	h.t.Helper()
	ev := &pb.Event{Event: &pb.Event_Mouse{Mouse: &pb.MouseEvent{
		Action: action, Button: button, Node: node, X: int32(x), Y: int32(y),
	}}}
	if err := h.enc.Encode(wire.TypeEvent, ev); err != nil {
		h.t.Fatalf("mouse %s: %v", node, err)
	}
}

func (h *scriptHost) sources(items ...*pb.Source) {
	h.t.Helper()
	ev := &pb.Event{Event: &pb.Event_Sources{Sources: &pb.SourcesEvent{Items: items}}}
	if err := h.enc.Encode(wire.TypeEvent, ev); err != nil {
		h.t.Fatalf("sources: %v", err)
	}
}

func (h *scriptHost) answer(result *pb.Result, ok bool, errText string, data *pb.MethodData) {
	h.t.Helper()
	resp := &pb.Response{RequestId: result.GetRequestId(), Epoch: result.GetEpoch(), Ok: ok, Error: errText, Data: data}
	if err := h.enc.Encode(wire.TypeResponse, resp); err != nil {
		h.t.Fatalf("response: %v", err)
	}
}

// nextFrame reads one frame or fails on the deadline.
func (h *scriptHost) nextFrame(deadline time.Time) (wire.Type, []byte) {
	h.t.Helper()
	type frame struct {
		typ     wire.Type
		payload []byte
		err     error
	}
	ch := make(chan frame, 1)
	go func() {
		typ, payload, err := h.dec.Decode()
		ch <- frame{typ, payload, err}
	}()
	select {
	case got := <-ch:
		if got.err != nil {
			h.t.Fatalf("read frame: %v", got.err)
		}
		return got.typ, got.payload
	case <-time.After(time.Until(deadline)):
		if h.errCh != nil {
			select {
			case err := <-h.errCh:
				h.t.Fatalf("program exited while waiting for a frame: %v", err)
			default:
			}
		}
		h.t.Fatalf("timeout waiting for a frame\nmethods: %v\nview:\n%s", h.methods, allText(h.cache.root))
		return 0, nil
	}
}

// pump applies frames until cond holds. RESULT frames are answered by the
// script (create/attach/access.call/system.quit).
func (h *scriptHost) pump(what string, cond func() bool) {
	h.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			h.t.Fatalf("timeout waiting for %s\nmethods: %v\nview:\n%s", what, h.methods, allText(h.cache.root))
		}
		typ, payload := h.nextFrame(deadline)
		switch typ {
		case wire.TypeView, wire.TypeViewDelta:
			h.cache.apply(h.t, typ, payload)
		case wire.TypeResult:
			result := decodePayload(h.t, typ, payload).(*pb.Result)
			h.handleResult(result)
		}
	}
}

func (h *scriptHost) handleResult(result *pb.Result) {
	h.methods = append(h.methods, result.GetMethod())
	switch result.GetMethod() {
	case "access.call":
		h.handleAccess(result)
	case "terminal.create":
		h.creates++
		h.answer(result, true, "", &pb.MethodData{Endpoint: "local", Id: fmt.Sprintf("term-%d", h.creates)})
	case "system.quit":
		h.quit = true
		h.answer(result, true, "", nil)
	default:
		h.answer(result, true, "", nil)
	}
}

func (h *scriptHost) handleAccess(result *pb.Result) {
	h.t.Helper()
	var envelope apipb.CommandEnvelope
	if err := gproto.Unmarshal(result.GetParams().GetAccessCommand(), &envelope); err != nil {
		h.t.Fatalf("unmarshal access command: %v", err)
	}
	switch cmd := envelope.GetCommand().(type) {
	case *apipb.CommandEnvelope_StorageGet:
		h.gets++
		key := cmd.StorageGet.GetKey().GetKey()
		value := h.layout
		if key == legacyPinKey {
			value = h.pin
		}
		if value == "" {
			payload, err := gproto.Marshal(&apipb.ResultEnvelope{
				Result: &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{
					Code: apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, Message: "not found",
				}},
			})
			if err != nil {
				h.t.Fatalf("marshal not found: %v", err)
			}
			h.answer(result, true, "", &pb.MethodData{AccessResult: payload})
			return
		}
		payload, err := gproto.Marshal(&apipb.ResultEnvelope{
			Result: &apipb.ResultEnvelope_StorageGet{StorageGet: &apipb.StorageGetResult{
				Entry: &apipb.StorageEntry{Key: cmd.StorageGet.GetKey(), Value: []byte(value)},
			}},
		})
		if err != nil {
			h.t.Fatalf("marshal storage get: %v", err)
		}
		h.answer(result, true, "", &pb.MethodData{AccessResult: payload})
	case *apipb.CommandEnvelope_StoragePut:
		value := string(cmd.StoragePut.GetValue())
		if cmd.StoragePut.GetKey().GetKey() == layoutKey {
			h.layout = value
			h.puts = append(h.puts, value)
		}
		payload, err := gproto.Marshal(&apipb.ResultEnvelope{
			Result: &apipb.ResultEnvelope_StoragePut{StoragePut: &apipb.StoragePutResult{
				Entry: &apipb.StorageEntry{Key: cmd.StoragePut.GetKey()},
			}},
		})
		if err != nil {
			h.t.Fatalf("marshal storage put: %v", err)
		}
		h.answer(result, true, "", &pb.MethodData{AccessResult: payload})
	}
}

func (h *scriptHost) sawMethod(method string) bool {
	for _, m := range h.methods {
		if m == method {
			return true
		}
	}
	return false
}

// --- program harness ---

type programRun struct {
	host       *scriptHost
	errCh      chan error
	hostToProg *pipe
	progToHost *pipe
}

func startProgram(t *testing.T) *programRun {
	t.Helper()
	hostToProgram := newPipe()
	programToHost := newPipe()
	client := sdk.New(hostToProgram, programToHost, sdk.Handlers{})
	memo := &app.Memo{}
	run := &programRun{
		errCh:      make(chan error, 1),
		hostToProg: hostToProgram,
		progToHost: programToHost,
		host: &scriptHost{
			t:     t,
			enc:   wire.NewEncoder(hostToProgram, wire.RoleHost, 0),
			dec:   wire.NewDecoder(programToHost, wire.RoleHost, 0),
			cache: &viewCache{},
		},
	}
	run.host.errCh = run.errCh
	go func() {
		run.errCh <- (&app.Program{Client: client, Model: newModel(client, memo), Memo: memo, Keys: &sdk.Keys{All: true}}).Run()
	}()
	t.Cleanup(func() {
		hostToProgram.Close()
		programToHost.Close()
	})
	return run
}

// pressPrefix sends ctrl+b, waits for the mode bar, then sends the action key.
func (r *programRun) pressPrefix(action string) {
	r.host.t.Helper()
	r.host.key("kb", "ctrl-b")
	r.host.pump("prefix mode bar", func() bool {
		return strings.Contains(allText(r.host.cache.root), "PREFIX")
	})
	r.host.key("k-"+action, action)
}

// --- the end-to-end scenario ---

func TestProgramEndToEnd(t *testing.T) {
	run := startProgram(t)
	host := run.host
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	alpha.ResizeOwner = "view-1"
	beta := terminalSource("terminal:local:beta", "beta", "local")

	// 1. HELLO commits the first view; Init reads `layout` and (not found)
	// falls back to the legacy `pinned` hint.
	host.sendHello(1)
	host.pump("initial view", func() bool { return host.cache.root != nil })
	if !strings.Contains(allText(host.cache.root), "empty pane") {
		t.Fatalf("first view is missing the empty state:\n%s", allText(host.cache.root))
	}
	host.pin = alpha.GetId()
	host.pump("layout + pinned reads", func() bool { return host.gets >= 2 })
	if host.methods[0] != "access.call" {
		t.Fatalf("first method = %q, want access.call", host.methods[0])
	}

	// 2. Sources bind the pinned source to the first empty pane. First run
	// (no saved layout) must not have toasted anything.
	host.sources(alpha, beta)
	host.pump("pane bound to alpha", func() bool {
		term := findBox(host.cache.root, termPrefix+"w1:t1:p1")
		return term != nil && term.GetContent().GetSelf() == alpha.GetId()
	})
	if findBox(host.cache.root, toastID) != nil {
		t.Fatalf("first run rendered a toast:\n%s", allText(host.cache.root))
	}
	text := allText(host.cache.root)
	for _, want := range []string{"alpha", "[you]", "Agents", "Spaces"} {
		if !strings.Contains(text, want) {
			t.Fatalf("sidebar is missing %q:\n%s", want, text)
		}
	}

	// 3. ctrl+b v splits the tab; enter creates a terminal for the new pane.
	run.pressPrefix("v")
	host.pump("split", func() bool {
		return findBox(host.cache.root, dividerPrefix+"w1:t1:0") != nil &&
			findBox(host.cache.root, paneBoxPrefix+"w1:t1:p2") != nil
	})
	host.key("k-enter", "enter")
	host.pump("create response", func() bool {
		return strings.Contains(allText(host.cache.root), "created terminal:local:term-1")
	})
	if host.creates != 1 {
		t.Fatalf("creates = %d, want 1", host.creates)
	}
	// The host publishes the created terminal; the pane turns live and the
	// layout is saved.
	host.sources(alpha, beta, terminalSource("terminal:local:term-1", "term-1", "local"))
	host.pump("pane 2 terminal box", func() bool {
		term := findBox(host.cache.root, termPrefix+"w1:t1:p2")
		return term != nil && term.GetContent().GetSelf() == "terminal:local:term-1"
	})
	if findBox(host.cache.root, termPrefix+"w1:t1:p1") == nil {
		t.Fatal("pane 1 terminal box disappeared")
	}
	host.pump("layout save after create", func() bool { return len(host.puts) >= 1 })

	// 4. ctrl+b c opens a new tab.
	run.pressPrefix("c")
	host.pump("new tab", func() bool { return findBox(host.cache.root, tabPrefix+"w1:t2") != nil })
	if !strings.Contains(allText(host.cache.root), "2:shell") {
		t.Fatalf("tab bar is missing the second tab:\n%s", allText(host.cache.root))
	}

	// 5. A synthesized click on a sidebar pane row switches back to t1 and
	// focuses pane 1.
	host.mouse("press", "left", agentRowPrefix+"w1:t1:p1", 3, 20)
	host.pump("sidebar click focuses pane 1", func() bool {
		term := findBox(host.cache.root, termPrefix+"w1:t1:p1")
		return term != nil && term.GetFocused()
	})

	// 6. ctrl+b q asks the host to quit (detach).
	run.pressPrefix("q")
	host.pump("system.quit", func() bool { return host.quit })
	if !host.sawMethod("system.quit") {
		t.Fatalf("methods = %v, want system.quit", host.methods)
	}

	// 7. The saved layout JSON contains the two tabs and the split.
	if len(host.puts) == 0 {
		t.Fatal("no layout was persisted")
	}
	var stored layoutDoc
	if err := json.Unmarshal([]byte(host.puts[len(host.puts)-1]), &stored); err != nil {
		t.Fatalf("stored layout: %v", err)
	}
	if len(stored.Workspaces) != 1 || len(stored.Workspaces[0].Tabs) != 2 {
		t.Fatalf("stored layout shape = %+v", stored)
	}
	tab1 := stored.Workspaces[0].Tabs[0]
	if len(tab1.Panes) != 2 || tab1.Panes[0].Source != alpha.GetId() ||
		tab1.Panes[1].Source != "terminal:local:term-1" {
		t.Fatalf("stored tab1 = %+v", tab1)
	}
	if !sameInts(tab1.Weights, []int{50, 50}) {
		t.Fatalf("stored weights = %v", tab1.Weights)
	}

	// The host closes the transport once it quits; Run returns cleanly.
	_ = run.hostToProg.Close()
	select {
	case err := <-run.errCh:
		if err != nil && !errors.Is(err, app.ErrQuit) {
			t.Fatalf("Run error = %v, want ErrQuit or nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Run to end")
	}
}

// TestProgramRestoresSavedLayout restarts the program with the layout another
// run persisted and checks the restored structure plus source re-binding.
func TestProgramRestoresSavedLayout(t *testing.T) {
	run := startProgram(t)
	host := run.host
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	alpha.ResizeOwner = "view-1"
	created := "terminal:local:term-1"

	doc := layoutDoc{
		Version:         1,
		ActiveWorkspace: "w1",
		Workspaces: []layoutWorkspace{{
			ID: "w1", Name: "main", ActiveTab: "t2",
			Tabs: []layoutTab{
				{
					ID: "t1", Name: "main", Axis: "row", Weights: []int{40, 60}, Focus: "p2",
					Panes: []layoutPane{
						{ID: "p1", Source: alpha.GetId()},
						{ID: "p2", Name: "second", Source: created},
					},
				},
				{
					ID: "t2", Name: "logs", Axis: "col", Weights: []int{100}, Focus: "p1",
					Panes: []layoutPane{{ID: "p1"}},
				},
			},
		}},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	host.layout = string(data)

	host.sendHello(1)
	host.pump("layout restored", func() bool { return host.gets >= 1 })
	host.sources(alpha, terminalSource(created, "term-1", "local"))

	host.pump("restored panes", func() bool {
		root := host.cache.root
		// The active tab is t2; the sidebar still lists t1's panes.
		return findBox(root, tabPrefix+"w1:t2") != nil && strings.Contains(allText(root), "alpha")
	})
	// t2 is the active tab: its empty pane is shown.
	if !strings.Contains(allText(host.cache.root), "logs") {
		t.Fatalf("restored tab bar:\n%s", allText(host.cache.root))
	}
	if findBox(host.cache.root, paneBoxPrefix+"w1:t2:p1") == nil {
		t.Fatalf("active tab pane is missing:\n%s", allText(host.cache.root))
	}
	// The restored binding for t1:p2 carries the user rename and source.
	host.mouse("press", "left", tabPrefix+"w1:t1", 30, 0)
	host.pump("switch to t1", func() bool {
		return findBox(host.cache.root, termPrefix+"w1:t1:p2") != nil
	})
	if !strings.Contains(allText(host.cache.root), "second") {
		t.Fatalf("pane rename was not restored:\n%s", allText(host.cache.root))
	}
	// Weights survive the round trip.
	var restored layoutDoc
	if err := json.Unmarshal([]byte(host.layout), &restored); err != nil {
		t.Fatalf("stored layout: %v", err)
	}
	if !sameInts(restored.Workspaces[0].Tabs[0].Weights, []int{40, 60}) {
		t.Fatalf("restored weights = %v", restored.Workspaces[0].Tabs[0].Weights)
	}
}
