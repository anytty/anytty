// Package bench measures the SDK + codec hot paths that a layout program
// hits every frame: full-view Commit (100/1000/10000 boxes), VIEW_DELTA diff
// (cloned-tree and memoized per sdk/app.Memo), key-event decode, RESULT
// encode and STREAM frame encode. Run with -benchmem:
//
//	go test ./clients/tui/sdk/bench -bench . -benchmem -count=5
package bench

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
	"github.com/anytty/anytty/clients/tui/sdk/builder"
	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"

	gproto "google.golang.org/protobuf/proto"
)

// repeatReader replays one fixed frame forever without allocating, so a
// decode benchmark measures the Decoder, not the reader.
type repeatReader struct {
	data []byte
	pos  int
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.pos == len(r.data) {
		r.pos = 0
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// buildTree builds a deterministic tree with exactly n boxes (8-way split,
// so the recursion depth stays logarithmic).
func buildTree(n int) *builder.Builder {
	if n <= 1 {
		return builder.Text("leaf")
	}
	root := builder.Col()
	rest := n - 1
	children := 8
	if rest < children {
		children = rest
	}
	base, extra := rest/children, rest%children
	for i := 0; i < children; i++ {
		count := base
		if i < extra {
			count++
		}
		root.Child(buildTree(count))
	}
	return root
}

// helloClient returns a client whose HELLO has been applied and whose writes
// go to io.Discard, the state Commit/Emit run in after the handshake.
func helloClient(b testing.TB) *sdk.Client {
	b.Helper()
	var buf bytes.Buffer
	if err := wire.NewEncoder(&buf, wire.RoleHost, 0).Encode(wire.TypeHello, &pb.Hello{
		Schema: 1, ViewId: "view:bench:1", Epoch: 1, Cols: 120, Rows: 40,
	}); err != nil {
		b.Fatalf("encode hello: %v", err)
	}
	client := sdk.New(bytes.NewReader(buf.Bytes()), io.Discard, sdk.Handlers{})
	if err := client.Loop(); err != nil {
		b.Fatalf("loop: %v", err)
	}
	return client
}

func keyEventFrame(b *testing.B) []byte {
	b.Helper()
	frame, err := wire.Marshal(wire.TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "ev-1", Key: "ctrl-p"}}}, 0)
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}
	return frame
}

func BenchmarkCommit100(b *testing.B)   { benchmarkCommit(b, 100) }
func BenchmarkCommit1000(b *testing.B)  { benchmarkCommit(b, 1000) }
func BenchmarkCommit10000(b *testing.B) { benchmarkCommit(b, 10000) }

func benchmarkCommit(b *testing.B, nodes int) {
	tree := buildTree(nodes).Build()
	client := helloClient(b)
	keys := sdk.Keys{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Commit(tree, keys); err != nil {
			b.Fatalf("commit: %v", err)
		}
	}
}

// BenchmarkKeyEventDecode measures the envelope decode of one key EVENT
// frame (no payload unmarshal), the fastest path the host -> program side
// takes per keystroke.
func BenchmarkKeyEventDecode(b *testing.B) {
	dec := wire.NewDecoder(&repeatReader{data: keyEventFrame(b)}, wire.RoleProgram, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := dec.Decode(); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}

// BenchmarkKeyEventDecodeRoundTrip measures decode + typed unmarshal of one
// key EVENT frame, what sdk/core.Loop does per keystroke.
func BenchmarkKeyEventDecodeRoundTrip(b *testing.B) {
	dec := wire.NewDecoder(&repeatReader{data: keyEventFrame(b)}, wire.RoleProgram, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		typ, payload, err := dec.Decode()
		if err != nil {
			b.Fatalf("decode: %v", err)
		}
		if _, err := wire.UnmarshalPayload(typ, payload); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
	}
}

// BenchmarkKeyEventDecodeReuse is the same envelope decode through the
// scratch-buffer mode added in this change (no per-frame payload buffer).
func BenchmarkKeyEventDecodeReuse(b *testing.B) {
	dec := wire.NewDecoder(&repeatReader{data: keyEventFrame(b)}, wire.RoleProgram, 0).ReuseBuffer(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := dec.Decode(); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}

// BenchmarkKeyEventDecodeRoundTripReuse is the full host -> program
// keystroke path in scratch-buffer mode.
func BenchmarkKeyEventDecodeRoundTripReuse(b *testing.B) {
	dec := wire.NewDecoder(&repeatReader{data: keyEventFrame(b)}, wire.RoleProgram, 0).ReuseBuffer(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		typ, payload, err := dec.Decode()
		if err != nil {
			b.Fatalf("decode: %v", err)
		}
		if _, err := wire.UnmarshalPayload(typ, payload); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}
	}
}

// TestDeltaBytesVsFullView asserts the protocol-level win the benchmarks
// report: a one-leaf change on a large tree costs well under 5% of the full
// VIEW, and a whole-tree change falls back to a full VIEW.
func TestDeltaBytesVsFullView(t *testing.T) {
	const nodes = 10000
	a, btree := deltaPair(nodes)
	client, out := helloDeltaClient(t)

	keys := sdk.Keys{}
	if err := client.Commit(a, keys); err != nil {
		t.Fatalf("commit base: %v", err)
	}
	fullBytes := out.n
	out.n = 0

	sent, err := client.CommitDelta(nil, btree, keys)
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if !sent {
		t.Fatal("one-leaf edit fell back to a full VIEW")
	}
	deltaBytes := out.n
	out.n = 0
	if ratio := float64(deltaBytes) / float64(fullBytes); ratio >= 0.05 {
		t.Fatalf("delta/full = %d/%d = %.4f, want < 0.05", deltaBytes, fullBytes, ratio)
	}

	// Whole-tree change: clearing the root id cannot be a set, so the diff is a
	// replace whose box is the whole tree; the guard must pick the full VIEW.
	huge := cloneTree(a)
	huge.Id = "root-id"
	if err := client.Commit(huge, keys); err != nil {
		t.Fatalf("commit huge: %v", err)
	}
	fullHuge := out.n
	out.n = 0
	hugeNext := cloneTree(huge)
	hugeNext.Id = ""
	sent, err = client.CommitDelta(nil, hugeNext, keys)
	if err != nil {
		t.Fatalf("delta huge: %v", err)
	}
	if sent {
		t.Fatalf("whole-tree change stayed a delta (%d bytes vs %d full)", out.n, fullHuge)
	}
	if out.n > fullHuge {
		t.Fatalf("fallback frame %d bytes exceeds the full VIEW %d bytes", out.n, fullHuge)
	}
	t.Logf("10000-node one-leaf delta = %d bytes, full VIEW = %d bytes (%.3f%%)", deltaBytes, fullBytes, 100*float64(deltaBytes)/float64(fullBytes))
}

func BenchmarkResultEncode(b *testing.B) {
	enc := wire.NewEncoder(io.Discard, wire.RoleProgram, 0)
	result := &pb.Result{
		RequestId: 42,
		Epoch:     1,
		Method:    "terminal.create",
		Params:    &pb.MethodParams{Endpoint: "local", Argv: []string{"zsh"}, Ephemeral: gproto.Bool(true)},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := enc.Encode(wire.TypeResult, result); err != nil {
			b.Fatalf("encode: %v", err)
		}
	}
}

func BenchmarkStreamEncode(b *testing.B) {
	enc := wire.NewEncoder(io.Discard, wire.RoleProgram, 0)
	payload := bytes.Repeat([]byte("x"), 256)
	frame := &pb.StreamFrame{StreamId: 7, Kind: "data", Offset: 3, Payload: payload}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := enc.Encode(wire.TypeStream, frame); err != nil {
			b.Fatalf("encode: %v", err)
		}
	}
}

// countingWriter counts bytes written so a benchmark can report produced
// bytes per frame (which -benchmem does not). It does not retain the frame, so
// its overhead matches io.Discard and ns/op stays comparable with the existing
// Commit benchmarks.
type countingWriter struct {
	n int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

// helloDeltaClient is helloClient with features["view_delta"]=true and a
// counting writer, so the delta path is negotiated and bytes can be measured.
func helloDeltaClient(b testing.TB) (*sdk.Client, *countingWriter) {
	b.Helper()
	var buf bytes.Buffer
	if err := wire.NewEncoder(&buf, wire.RoleHost, 0).Encode(wire.TypeHello, &pb.Hello{
		Schema: 1, ViewId: "view:bench:1", Epoch: 1, Cols: 120, Rows: 40,
		Features: map[string]bool{"view_delta": true},
	}); err != nil {
		b.Fatalf("encode hello: %v", err)
	}
	out := &countingWriter{}
	client := sdk.New(bytes.NewReader(buf.Bytes()), out, sdk.Handlers{})
	if err := client.Loop(); err != nil {
		b.Fatalf("loop: %v", err)
	}
	return client, out
}

// deltaPair builds two trees of n boxes that differ in exactly one text leaf,
// so a diff between them is one set patch. The alternating benchmark commits A
// then B then A, so each iteration diffs against the previous frame without
// cloning the tree.
func deltaPair(n int) (a, b *pb.Box) {
	a = buildTree(n).Build()
	b = cloneTree(a)
	target := lastText(b)
	if target.Content == nil {
		target.Content = &pb.Content{}
	}
	target.Content.Text = "changed"
	return a, b
}

func cloneTree(box *pb.Box) *pb.Box {
	return gproto.Clone(box).(*pb.Box)
}

// lastText returns the last text-bearing leaf, so the edit lands at the deep
// end of the tree (the most expensive address to reach).
func lastText(box *pb.Box) *pb.Box {
	if len(box.GetChildren()) == 0 {
		return box
	}
	return lastText(box.GetChildren()[len(box.GetChildren())-1])
}

func BenchmarkCommitDelta100(b *testing.B)   { benchmarkCommitDelta(b, 100) }
func BenchmarkCommitDelta1000(b *testing.B)  { benchmarkCommitDelta(b, 1000) }
func BenchmarkCommitDelta10000(b *testing.B) { benchmarkCommitDelta(b, 10000) }

func benchmarkCommitDelta(b *testing.B, nodes int) {
	a, btree := deltaPair(nodes)
	client, out := helloDeltaClient(b)
	if err := client.Commit(a, sdk.Keys{}); err != nil {
		b.Fatalf("commit base: %v", err)
	}
	out.n = 0
	keys := sdk.Keys{}
	b.ReportAllocs()
	b.ResetTimer()
	var total int64
	for i := 0; i < b.N; i++ {
		// Alternate the two trees so every call is a real one-leaf diff.
		// nil base: the client diffs against its own committed baseline.
		next := btree
		if i%2 == 1 {
			next = a
		}
		sent, err := client.CommitDelta(nil, next, keys)
		if err != nil {
			b.Fatalf("commit delta: %v", err)
		}
		if !sent {
			b.Fatal("delta path fell back to a full VIEW")
		}
		total += out.n
		out.n = 0
	}
	b.ReportMetric(float64(total)/float64(b.N), "B/frame")
}

// memoTreeKey identifies one node of the memo benchmark tree. rev is the
// node's revision: it changes only along the path to the edited leaf, so
// sibling subtrees keep their key (and therefore their pointer) across
// iterations.
type memoTreeKey struct {
	id  int
	rev int
}

// buildMemoTree builds an n-box tree (same 8-way shape as buildTree) through
// the Memo. id is the preorder index of the subtree root and target the
// preorder index of the leaf whose content changes with rev; rev is nonzero
// only on the root-to-target path, so a new revision rebuilds exactly that
// path and reuses every other subtree pointer.
func buildMemoTree(m *app.Memo, id, n, target, rev int) *pb.Box {
	return m.Box(memoTreeKey{id: id, rev: rev}, func() *pb.Box {
		if n <= 1 {
			text := "leaf"
			if id == target {
				// Fixed-width so B/frame stays comparable with the cloned-tree
				// control regardless of the revision.
				text = fmt.Sprintf("chg%04d", rev%10000)
			}
			return &pb.Box{Content: &pb.Content{Text: text}}
		}
		root := &pb.Box{Flow: "col"}
		rest := n - 1
		children := 8
		if rest < children {
			children = rest
		}
		base, extra := rest/children, rest%children
		childID := id + 1
		for i := 0; i < children; i++ {
			count := base
			if i < extra {
				count++
			}
			childRev := 0
			if childID <= target && target < childID+count {
				childRev = rev
			}
			root.Children = append(root.Children, buildMemoTree(m, childID, count, target, childRev))
			childID += count
		}
		return root
	})
}

// BenchmarkCommitDeltaMemo10000 is the memoized counterpart of
// BenchmarkCommitDelta10000: the tree is built through app.Memo with keys that
// change only on the root-to-target path, so every other subtree keeps its
// *pb.Box pointer and the diff short-circuits there. It is the O(changed)
// control against the existing O(tree) benchmark; frame bytes must stay around
// the 40 B of the cloned-tree delta.
func BenchmarkCommitDeltaMemo10000(b *testing.B) {
	const nodes = 10000
	const target = nodes - 1
	client, out := helloDeltaClient(b)
	keys := sdk.Keys{}

	var memo app.Memo
	memo.BeginFrame()
	base := buildMemoTree(&memo, 0, nodes, target, 0)
	memo.EndFrame()
	if err := client.Commit(base, keys); err != nil {
		b.Fatalf("commit base: %v", err)
	}
	out.n = 0

	b.ReportAllocs()
	b.ResetTimer()
	var total int64
	for i := 0; i < b.N; i++ {
		// One new revision rebuilds only the root-to-target path through the
		// memo; nil base makes the client diff against its committed baseline,
		// exactly like BenchmarkCommitDelta10000.
		memo.BeginFrame()
		tree := buildMemoTree(&memo, 0, nodes, target, i+1)
		memo.EndFrame()
		sent, err := client.CommitDelta(nil, tree, keys)
		if err != nil {
			b.Fatalf("commit delta: %v", err)
		}
		if !sent {
			b.Fatal("memoized one-leaf delta fell back to a full VIEW")
		}
		total += out.n
		out.n = 0
	}
	b.ReportMetric(float64(total)/float64(b.N), "B/frame")
}

// BenchmarkCommitBytes reports the full-VIEW frame bytes per commit, the
// baseline the delta benchmarks compare against.
func BenchmarkCommitBytes100(b *testing.B)   { benchmarkCommitBytes(b, 100) }
func BenchmarkCommitBytes1000(b *testing.B)  { benchmarkCommitBytes(b, 1000) }
func BenchmarkCommitBytes10000(b *testing.B) { benchmarkCommitBytes(b, 10000) }

func benchmarkCommitBytes(b *testing.B, nodes int) {
	tree := buildTree(nodes).Build()
	client, out := helloDeltaClient(b)
	keys := sdk.Keys{}
	b.ReportAllocs()
	b.ResetTimer()
	var total int64
	for i := 0; i < b.N; i++ {
		if err := client.Commit(tree, keys); err != nil {
			b.Fatalf("commit: %v", err)
		}
		total += out.n
		out.n = 0
	}
	b.ReportMetric(float64(total)/float64(b.N), "B/frame")
}

// BenchmarkEmit measures one SDK RESULT emit (request bookkeeping + encode).
func BenchmarkEmit(b *testing.B) {
	client := helloClient(b)
	params := &pb.MethodParams{Endpoint: "local", Id: "main", Delta: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Emit("terminal.scroll", params, nil); err != nil {
			b.Fatalf("emit: %v", err)
		}
	}
}

// BenchmarkFrameRoundTrip is the codec floor: decode one pre-built RESULT
// frame through an in-memory buffer.
func BenchmarkFrameRoundTrip(b *testing.B) {
	frame, err := wire.Marshal(wire.TypeResult, &pb.Result{RequestId: 1, Epoch: 1, Method: "terminal.scroll"}, 0)
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := wire.DecodeFrame(frame, wire.RoleHost, 0); err != nil {
			b.Fatalf("decode: %v", err)
		}
	}
}
