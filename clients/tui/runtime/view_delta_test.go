package runtime

import (
	"testing"

	gproto "google.golang.org/protobuf/proto"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
)

func boolp(v bool) *bool { return &v }

func (h *harness) sendDelta(d *pb.ViewDelta) { h.send(wire.TypeViewDelta, d) }

func patch(op string, path []uint32, box *pb.Box) *pb.Patch {
	return &pb.Patch{Op: op, Path: path, Box: box}
}

func delta(epoch, rev, base uint64, keys *pb.Keys, patches ...*pb.Patch) *pb.ViewDelta {
	return &pb.ViewDelta{Epoch: epoch, Rev: rev, RevBase: base, Keys: keys, Patches: patches}
}

// tree builds the rev_base used by most delta tests:
//
//	root
//	├── 0: "a"
//	├── 1: container "c" [ "c0", "c1", "c2" ]
//	└── 2: "b"
func deltaTree() *pb.View {
	c := &pb.Box{Id: "c", Flow: "col", Children: []*pb.Box{
		textBox("c0", "zero"), textBox("c1", "one"), textBox("c2", "two"),
	}}
	return &pb.View{
		Epoch: 1, Rev: 1, Root: &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{
			textBox("a", "A"), c, textBox("b", "B"),
		}},
	}
}

func newDeltaHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12})
	h.sendView(deltaTree())
	h.drain()
	return h
}

func TestViewDeltaSetMergesAndPreservesChildren(t *testing.T) {
	h := newDeltaHarness(t)

	// Set merges scalar fields, keeps children, and makes visible=false real.
	box := &pb.Box{Size: &pb.Size{Height: 3}, Visible: boolp(false), Style: "bold"}
	h.sendDelta(delta(1, 2, 1, nil, patch("set", []uint32{1}, box)))
	if h.s.Rev() != 2 {
		t.Fatalf("rev = %d, want 2", h.s.Rev())
	}

	got := h.s.View()
	c := got.Root.Children[1]
	if c.Id != "c" || c.Flow != "col" {
		t.Fatalf("set dropped untouched fields: %+v", c)
	}
	if c.Size.GetHeight() != 3 || c.Size.GetWidth() != 0 {
		t.Fatalf("set did not merge size: %+v", c.Size)
	}
	if c.Visible == nil || c.GetVisible() {
		t.Fatalf("set visible=false not applied: %+v", c.Visible)
	}
	if c.Style != "bold" {
		t.Fatalf("set style = %q, want bold", c.Style)
	}
	if len(c.Children) != 3 || c.Children[0].Id != "c0" || c.Children[2].Id != "c2" {
		t.Fatalf("set clobbered children: %+v", c.Children)
	}

	// The cached tree must not be reachable through the applied tree.
	box.Size.Height = 99
	if h.s.View().Root.Children[1].Size.GetHeight() != 3 {
		t.Fatal("applied tree aliases the patch payload")
	}
}

func TestViewDeltaReplace(t *testing.T) {
	h := newDeltaHarness(t)
	repl := &pb.Box{Id: "c", Children: []*pb.Box{textBox("only", "X")}}
	h.sendDelta(delta(1, 2, 1, nil, patch("replace", []uint32{1}, repl)))
	if h.s.Rev() != 2 {
		t.Fatalf("rev = %d, want 2", h.s.Rev())
	}
	c := h.s.View().Root.Children[1]
	if len(c.Children) != 1 || c.Children[0].Id != "only" {
		t.Fatalf("replace did not swap the subtree: %+v", c.Children)
	}
	// Root-level replace keeps the root addressable.
	h.sendDelta(delta(1, 3, 2, nil, patch("replace", nil, &pb.Box{Id: "root", Children: []*pb.Box{textBox("z", "Z")}})))
	if h.s.Rev() != 3 {
		t.Fatalf("root replace rev = %d, want 3", h.s.Rev())
	}
	if _, ok := h.rect("z"); !ok {
		t.Fatal("root replace not applied")
	}
	if _, ok := h.rect("a"); ok {
		t.Fatal("root replace must drop the old children")
	}
}

func TestViewDeltaInsertRemoveMove(t *testing.T) {
	h := newDeltaHarness(t)

	// insert at the end (index == child count) and in the middle
	h.sendDelta(delta(1, 2, 1, nil, &pb.Patch{Op: "insert", Path: []uint32{1}, Index: 3, Box: textBox("c3", "three")}))
	if ids := childIDs(h.s.View().Root.Children[1]); len(ids) != 4 || ids[3] != "c3" {
		t.Fatalf("append insert = %v", ids)
	}
	h.sendDelta(delta(1, 3, 2, nil, &pb.Patch{Op: "insert", Path: []uint32{1}, Index: 1, Box: textBox("cX", "x")}))
	if ids := childIDs(h.s.View().Root.Children[1]); len(ids) != 5 || ids[1] != "cX" {
		t.Fatalf("middle insert = %v", ids)
	}

	// remove the inserted middle node, then the append
	h.sendDelta(delta(1, 4, 3, nil, &pb.Patch{Op: "remove", Path: []uint32{1, 1}}))
	if ids := childIDs(h.s.View().Root.Children[1]); len(ids) != 4 || ids[0] != "c0" || ids[1] != "c1" {
		t.Fatalf("remove = %v", ids)
	}

	// move within the parent, to is the post-move index
	h.sendDelta(delta(1, 5, 4, nil, &pb.Patch{Op: "move", Path: []uint32{1}, From: 3, To: 0}))
	if ids := childIDs(h.s.View().Root.Children[1]); ids[0] != "c3" {
		t.Fatalf("move = %v, want c3 first", ids)
	}
}

func TestViewDeltaRejectsBaseMismatchAndFirstFrame(t *testing.T) {
	// First frame of an epoch must be a full VIEW; a delta has no base.
	h := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12})
	h.sendDelta(delta(1, 1, 0, nil, patch("set", []uint32{0}, &pb.Box{Id: "x"})))
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("first-frame delta produced %d frames, want 1", len(msgs))
	}
	rej := msgs[0].(*pb.Event).GetViewRejected()
	if rej == nil || rej.GetReason() != "base_mismatch" || rej.GetRev() != 1 {
		t.Fatalf("first-frame rejection = %+v", rej)
	}
	if h.s.Rev() != 0 || h.s.View() != nil {
		t.Fatal("rejected first delta must not install a view")
	}

	// Wrong rev_base against a live cache.
	h2 := newDeltaHarness(t)
	h2.sendDelta(delta(1, 5, 4, nil, patch("set", []uint32{0}, &pb.Box{Id: "a"})))
	msgs = h2.drain()
	if len(msgs) != 1 || msgs[0].(*pb.Event).GetViewRejected().GetReason() != "base_mismatch" {
		t.Fatalf("rev_base mismatch = %v", msgs)
	}
	if h2.s.Rev() != 1 {
		t.Fatalf("rejected delta changed rev to %d", h2.s.Rev())
	}

	// Same (epoch,rev) is rejected once per spec.
	h2.sendDelta(delta(1, 5, 4, nil, patch("set", []uint32{0}, &pb.Box{Id: "a"})))
	if msgs := h2.drain(); len(msgs) != 0 {
		t.Fatalf("duplicate rejection emitted: %v", msgs)
	}
}

func TestViewDeltaStaleRevAndWrongEpoch(t *testing.T) {
	h := newDeltaHarness(t)
	h.sendDelta(delta(1, 1, 1, nil, patch("set", []uint32{0}, &pb.Box{Id: "a"}))) // rev <= current
	if msgs := h.drain(); len(msgs) != 0 {
		t.Fatalf("stale rev produced frames: %v", msgs)
	}
	if h.s.Rev() != 1 {
		t.Fatalf("stale rev changed rev to %d", h.s.Rev())
	}

	h.sendDelta(delta(2, 9, 1, nil, patch("replace", nil, &pb.Box{Id: "other"})))
	if msgs := h.drain(); len(msgs) != 0 {
		t.Fatalf("wrong epoch produced frames: %v", msgs)
	}
	if _, ok := h.rect("other"); ok {
		t.Fatal("wrong-epoch delta was applied")
	}
}

func TestViewDeltaPathInvalid(t *testing.T) {
	cases := []struct {
		name string
		p    *pb.Patch
	}{
		{"out of range path", patch("set", []uint32{9}, &pb.Box{Id: "x"})},
		{"set target nil box", patch("set", []uint32{0}, nil)},
		{"remove root", &pb.Patch{Op: "remove", Path: nil}},
		{"remove out of range", &pb.Patch{Op: "remove", Path: []uint32{9}}},
		{"insert index too large", &pb.Patch{Op: "insert", Path: []uint32{1}, Index: 4, Box: box("x")}},
		{"insert missing box", &pb.Patch{Op: "insert", Path: []uint32{1}, Index: 0}},
		{"replace missing box", &pb.Patch{Op: "replace", Path: []uint32{1}}},
		{"move out of range", &pb.Patch{Op: "move", Path: []uint32{1}, From: 0, To: 7}},
		{"move missing container", &pb.Patch{Op: "move", Path: []uint32{0}}},
		{"unknown op", &pb.Patch{Op: "frobnicate", Path: []uint32{0}, Box: box("x")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newDeltaHarness(t)
			h.sendDelta(delta(1, 2, 1, nil, tc.p))
			msgs := h.drain()
			if len(msgs) != 1 {
				t.Fatalf("produced %d frames, want 1", len(msgs))
			}
			rej := msgs[0].(*pb.Event).GetViewRejected()
			if rej == nil || rej.GetReason() != "path_invalid" || rej.GetRev() != 2 {
				t.Fatalf("rejection = %+v", rej)
			}
			if h.s.Rev() != 1 {
				t.Fatalf("rev changed to %d after rejection", h.s.Rev())
			}
		})
	}
}

func TestViewDeltaAtomicity(t *testing.T) {
	h := newDeltaHarness(t)
	before := gproto.Clone(h.s.View())

	// The first patch is legal, the second is not: nothing may be committed.
	h.sendDelta(delta(1, 2, 1, nil,
		patch("replace", []uint32{0}, textBox("a", "changed")),
		&pb.Patch{Op: "remove", Path: nil}, // remove root is invalid
	))
	msgs := h.drain()
	if len(msgs) != 1 || msgs[0].(*pb.Event).GetViewRejected().GetReason() != "path_invalid" {
		t.Fatalf("rejection = %v", msgs)
	}
	if !gproto.Equal(before, h.s.View()) {
		t.Fatalf("cached view mutated by a rejected delta:\n got %v\nwant %v", h.s.View(), before)
	}
	if _, ok := h.rect("a"); !ok {
		t.Fatal("layout lost the pre-delta nodes")
	}
}

func TestViewDeltaMaxNodes(t *testing.T) {
	h := newHarness(t, Options{
		ViewID: "v", Cols: 40, Rows: 12,
		Limits: Limits{MaxNodes: 3, MaxMessageBytes: 1 << 20},
	})
	h.sendView(view(1, 1, nil, false, textBox("a", "A")))
	h.drain()

	// root + a = 2 nodes at base; two inserts push it to 4 > 3.
	h.sendDelta(delta(1, 2, 1, nil,
		patch("insert", nil, textBox("b", "B")),
		patch("insert", nil, textBox("c", "C")),
	))
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("max_nodes produced %d frames, want 1", len(msgs))
	}
	rej := msgs[0].(*pb.Event).GetViewRejected()
	if rej == nil || rej.GetReason() != "max_nodes" || rej.GetRev() != 2 {
		t.Fatalf("rejection = %+v", rej)
	}
	if h.s.Rev() != 1 {
		t.Fatalf("rev changed to %d after max_nodes rejection", h.s.Rev())
	}

	// A growth that stays within the limit applies.
	h.sendDelta(delta(1, 2, 1, nil, patch("insert", nil, textBox("b", "B"))))
	if h.s.Rev() != 2 {
		t.Fatalf("legal growth rev = %d, want 2", h.s.Rev())
	}
}

func TestViewDeltaKeepsClaimWhenKeysOmitted(t *testing.T) {
	h := newDeltaHarness(t)
	h.drain()
	h.sendView(view(1, 2, []string{"ctrl-p"}, false,
		textBox("a", "A"), box("c"), textBox("b", "B")))
	h.drain()
	if claim, _ := h.s.Claim(); len(claim) != 1 || claim[0] != "ctrl-p" {
		t.Fatalf("setup claim = %v", claim)
	}

	h.sendDelta(delta(1, 3, 2, nil, patch("set", []uint32{0}, &pb.Box{Id: "a"})))
	if claim, all := h.s.Claim(); len(claim) != 1 || claim[0] != "ctrl-p" || all {
		t.Fatalf("claim not retained when keys omitted: %v/%v", claim, all)
	}

	h.sendDelta(delta(1, 4, 3, &pb.Keys{Claim: []string{"ctrl-q"}, All: true}, patch("set", []uint32{0}, &pb.Box{Id: "a"})))
	if claim, all := h.s.Claim(); len(claim) != 1 || claim[0] != "ctrl-q" || !all {
		t.Fatalf("claim not replaced when keys present: %v/%v", claim, all)
	}
}

// TestViewDeltaEquivalentToFullView applies a sequence of deltas and checks
// that the resulting layout matches the equivalent full VIEW frame exactly.
func TestViewDeltaEquivalentToFullView(t *testing.T) {
	h := newDeltaHarness(t)

	h.sendDelta(delta(1, 2, 1, &pb.Keys{Claim: []string{"ctrl-p"}},
		patch("set", []uint32{0}, &pb.Box{Style: "bold", Visible: boolp(true)}),
		patch("replace", []uint32{1}, &pb.Box{Id: "c", Children: []*pb.Box{textBox("only", "X")}}),
		patch("insert", nil, textBox("tail", "tail")),
		&pb.Patch{Op: "move", From: 2, To: 0},
	))
	if h.s.Rev() != 2 {
		t.Fatalf("delta rev = %d, want 2", h.s.Rev())
	}
	deltaRoot := gproto.Clone(h.s.View().Root).(*pb.Box)
	deltaFrame, _ := h.s.Frame()

	full := newHarness(t, Options{ViewID: "v", Cols: 40, Rows: 12})
	full.sendView(&pb.View{
		Epoch: 1, Rev: 2,
		Keys: &pb.Keys{Claim: []string{"ctrl-p"}},
		Root: gproto.Clone(deltaRoot).(*pb.Box),
	})
	fullFrame, _ := full.s.Frame()

	if !gproto.Equal(deltaRoot, full.s.View().Root) {
		t.Fatalf("delta tree != full tree:\n delta %v\n full  %v", deltaRoot, full.s.View().Root)
	}
	if deltaFrame.RectCount() != fullFrame.RectCount() {
		t.Fatalf("rect count = %d, want %d", deltaFrame.RectCount(), fullFrame.RectCount())
	}
	rect := 0
	fullFrame.RectsIterate(func(id string, r kernel.Rect) bool {
		rect++
		if got, ok := deltaFrame.Rect(id); !ok || got != r {
			t.Fatalf("rect[%s] = %+v/%v, want %+v", id, got, ok, r)
		}
		return true
	})
	if rect != fullFrame.RectCount() {
		t.Fatalf("iterated %d rects, want %d", rect, fullFrame.RectCount())
	}

	// Composed framebuffer equivalence (the full view is the same tree).
	if !h.s.ComposeFrame(nil, nil).Equal(full.s.ComposeFrame(nil, nil)) {
		t.Fatal("delta composition != full view composition")
	}
	if claim, all := h.s.Claim(); len(claim) != 1 || claim[0] != "ctrl-p" || all {
		t.Fatalf("delta claim = %v/%v", claim, all)
	}
}

func TestViewDeltaRejectedOncePerRev(t *testing.T) {
	h := newDeltaHarness(t)
	bad := delta(1, 2, 9, nil, patch("set", []uint32{0}, &pb.Box{Id: "a"}))
	h.sendDelta(bad)
	if msgs := h.drain(); len(msgs) != 1 {
		t.Fatalf("first rejection produced %d frames, want 1", len(msgs))
	}
	h.sendDelta(bad)
	if msgs := h.drain(); len(msgs) != 0 {
		t.Fatalf("same (epoch,rev) rejected twice: %v", msgs)
	}
}

func TestViewDeltaOversize(t *testing.T) {
	h := newHarness(t, Options{
		ViewID: "v", Cols: 40, Rows: 12,
		Limits: Limits{MaxNodes: 100, MaxMessageBytes: 1 << 20},
	})
	if err := h.s.HandleOversize(wire.TypeViewDelta); err != nil {
		t.Fatalf("HandleOversize: %v", err)
	}
	msgs := h.drain()
	if len(msgs) != 1 {
		t.Fatalf("oversize delta produced %d frames, want 1", len(msgs))
	}
	rej := msgs[0].(*pb.Event).GetViewRejected()
	if rej == nil || rej.GetReason() != "oversize" || rej.GetRev() != 0 {
		t.Fatalf("oversize rejection = %+v", rej)
	}
}

func childIDs(b *pb.Box) []string {
	out := make([]string, len(b.Children))
	for i, c := range b.Children {
		out[i] = c.Id
	}
	return out
}
