package core

import (
	"sync"

	wire "github.com/anytty/anytty/proto/ui"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
	"google.golang.org/protobuf/encoding/protowire"
	gproto "google.golang.org/protobuf/proto"
)

// viewRootField is the View.root field number (proto/ui/tui2.proto): the cost
// guard adds its tag and length prefix when it sizes the root separately.
const viewRootField = 4

// Patch op names (PROTOCOL §2.1). The wire protocol carries them as strings.
const (
	OpSet     = "set"
	OpReplace = "replace"
	OpInsert  = "insert"
	OpRemove  = "remove"
)

// ViewDeltaFeature is the HELLO feature a host advertises to accept
// VIEW_DELTA frames (PROTOCOL §1, §2.1).
const ViewDeltaFeature = "view_delta"

// deltaScratch is the reusable CommitDelta envelope: the ViewDelta plus one
// keys message, so a steady-state delta does not allocate envelopes. It lives
// in a pool because CommitDelta runs per frame; like viewScratch it is only
// touched while writeMu is held.
type deltaScratch struct {
	delta pb.ViewDelta
	keys  pb.Keys
}

var deltaScratchPool = sync.Pool{New: func() any { return &deltaScratch{} }}

// frameBufPool reuses the delta frame buffer the cost guard marshals into.
// Buffers larger than frameRetainLimit are dropped instead of retained,
// mirroring wire's encoder pool.
var frameBufPool = sync.Pool{New: func() any {
	buf := make([]byte, 0, 4096)
	return &buf
}}

const frameRetainLimit = wire.DefaultMaxMessageBytes + 5

func getFrameBuf() *[]byte { return frameBufPool.Get().(*[]byte) }

func putFrameBuf(buf *[]byte) {
	if cap(*buf) > int(frameRetainLimit) {
		return
	}
	frameBufPool.Put(buf)
}

// DiffView computes the patches that turn base into next (PROTOCOL §2.1).
//
// Addressing is by child index: a patch path is the sequence of children[i]
// indices from the root (an empty path is the root itself). The walk is
// depth-first:
//
//   - equal subtrees (boxedEquals, pointer fast path) produce no patch;
//   - a node whose non-children fields changed emits one set carrying only the
//     fields that differ (never children). proto3 has no scalar presence, so a
//     change that clears a scalar, a bool to false, or a sub-message is not
//     representable by a merge-only set; the node is replaced instead;
//   - a node whose child count changed emits set(s) for its own fields plus
//     insert/remove for the unmatched middle, where the middle is what lies
//     between a common prefix and a common suffix of deeply equal children
//     (LCS is overkill: matching from both ends is exact for inserts and
//     removes). A child whose own child count changed is diffed recursively
//     when it still shares an equal prefix/suffix child, else replaced whole.
//
// The second result is false when there is nothing to send: base and next are
// identical, or the diff could not be expressed (a nil node, or a nil child
// that changed, which has no addressable index). A caller that gets false must
// either skip the frame (identical) or fall back to a full VIEW. The returned
// patches share no storage with base or next except by pointer for the
// replaced/inserted subtrees (the caller owns next and must not mutate it until
// the frame is written).
func DiffView(base, next *pb.Box) ([]*pb.Patch, bool) {
	if base == nil || next == nil {
		return nil, false
	}
	if boxesEqual(base, next) {
		return nil, false
	}
	var patches []*pb.Patch
	diffNode(base, next, nil, &patches)
	if len(patches) == 0 {
		// A change exists but could not be addressed (a nil child on either
		// side). The caller must fall back to a full VIEW.
		return nil, false
	}
	return patches, true
}

// boxesEqual reports whether two boxes are deeply protocol-equal, ignoring the
// nil-vs-empty slice/map distinction that proto equality also ignores. It is a
// hand-written walk instead of gproto.Equal: the diff calls it on every
// unchanged subtree, and the reflection-based comparison allocates and
// dominates the cost for large trees. The pointer fast path makes shared
// subtrees free.
func boxesEqual(a, b *pb.Box) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.GetId() != b.GetId() ||
		a.GetFlow() != b.GetFlow() ||
		a.GetStyle() != b.GetStyle() ||
		a.GetFocused() != b.GetFocused() ||
		visibleEffective(a) != visibleEffective(b) {
		return false
	}
	if !equalStrings(a.GetInput(), b.GetInput()) {
		return false
	}
	if !equalSize(a.GetSize(), b.GetSize()) ||
		!equalPos(a.GetPos(), b.GetPos()) ||
		!equalCursor(a.GetCursor(), b.GetCursor()) ||
		!equalContent(a.GetContent(), b.GetContent()) {
		return false
	}
	ac, bc := a.GetChildren(), b.GetChildren()
	if len(ac) != len(bc) {
		return false
	}
	for i := range ac {
		if !boxesEqual(ac[i], bc[i]) {
			return false
		}
	}
	return true
}

func equalSize(a, b *pb.Size) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}
	return a.GetWidth() == b.GetWidth() && a.GetHeight() == b.GetHeight() && a.GetFlex() == b.GetFlex()
}

func equalPos(a, b *pb.Pos) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}
	return a.GetX() == b.GetX() && a.GetY() == b.GetY()
}

func equalCursor(a, b *pb.Cursor) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}
	return a.GetRow() == b.GetRow() && a.GetCol() == b.GetCol() && a.GetShape() == b.GetShape()
}

// diffNode appends the patches for next under path. base and next are non-nil;
// their child lists may differ in length.
func diffNode(base, next *pb.Box, path []uint32, patches *[]*pb.Patch) {
	if boxesEqual(base, next) {
		return
	}
	// proto3 has no scalar presence: a set patch cannot clear a string, a bool
	// or a whole sub-message back to its zero value. When any such change is
	// needed, the node is replaced instead (PROTOCOL §2.1 "replace").
	if !setRepresentable(base, next) {
		*patches = append(*patches, replacePatch(path, next))
		return
	}
	bc, nc := base.GetChildren(), next.GetChildren()

	// Same child count: patch this node's scalars, then recurse into every
	// child position. A child whose own child count differs is replaced by
	// diffChild.
	if len(bc) == len(nc) {
		if set := scalarPatch(base, next, path); set != nil {
			*patches = append(*patches, set)
		}
		for i := range nc {
			diffChild(bc[i], nc[i], path, uint32(i), patches)
		}
		return
	}

	if set := scalarPatch(base, next, path); set != nil {
		*patches = append(*patches, set)
	}
	// Match a common prefix and suffix of deeply equal children; the middle is
	// the edit. Removing then inserting at the prefix boundary keeps every other
	// child's index stable.
	prefix := commonPrefix(bc, nc)
	suffix := commonSuffix(bc, nc, prefix)
	for i := prefix; i < len(bc)-suffix; i++ {
		*patches = append(*patches, &pb.Patch{Op: OpRemove, Path: childPath(path, uint32(prefix))})
	}
	for i := prefix; i < len(nc)-suffix; i++ {
		*patches = append(*patches, &pb.Patch{
			Op:    OpInsert,
			Path:  clonePath(path),
			Index: uint32(i),
			Box:   nc[i],
		})
	}
	// The matched prefix/suffix children are deeply equal by definition and
	// need no recursion; only the replaced middle changed.
}

// diffChild patches one pair of children. A child whose own child count changed
// is a shape change: it is replaced as a whole subtree unless it still shares a
// deeply equal prefix or suffix with its base, in which case the recursive
// insert/remove path can express the edit more cheaply. The child path is only
// materialised when the child actually differs, so an unchanged subtree costs
// no allocation.
func diffChild(base, next *pb.Box, path []uint32, index uint32, patches *[]*pb.Patch) {
	if base == nil || next == nil {
		// A nil child has no addressable index; only a corrupt tree reaches
		// this, and DiffView's caller falls back to a full VIEW.
		return
	}
	if boxesEqual(base, next) {
		return
	}
	child := childPath(path, index)
	if sameChildShape(base, next) || anchored(base, next) {
		diffNode(base, next, child, patches)
		return
	}
	*patches = append(*patches, replacePatch(child, next))
}

// sameChildShape reports whether two nodes have the same number of children,
// so they can be diffed in place instead of replaced.
func sameChildShape(a, b *pb.Box) bool {
	return len(a.GetChildren()) == len(b.GetChildren())
}

// anchored reports whether two nodes share a deeply equal prefix or suffix
// child, so a child-count change can be expressed with insert/remove instead of
// a whole-subtree replace.
func anchored(a, b *pb.Box) bool {
	ac, bc := a.GetChildren(), b.GetChildren()
	return commonPrefix(ac, bc) > 0 || commonSuffix(ac, bc, 0) > 0
}

// commonPrefix returns the number of leading children that are deeply equal.
func commonPrefix(a, b []*pb.Box) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && boxesEqual(a[i], b[i]) {
		i++
	}
	return i
}

// commonSuffix returns the number of trailing children that are deeply equal,
// never counting past start (so the prefix and suffix cannot overlap).
func commonSuffix(a, b []*pb.Box, start int) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	limit -= start
	i := 0
	for i < limit && boxesEqual(a[len(a)-1-i], b[len(b)-1-i]) {
		i++
	}
	return i
}

// scalarPatch returns a set patch for next's non-children fields when any of
// them differs from base, or nil when they are all equal. The emitted box
// carries only the fields that differ; protobuf merges them into the cached
// node. Children are never copied (the caller is responsible for them).
func scalarPatch(base, next *pb.Box, path []uint32) *pb.Patch {
	if !scalarsDiffer(base, next) {
		return nil
	}
	set := &pb.Box{
		Id:      next.GetId(),
		Flow:    next.GetFlow(),
		Style:   next.GetStyle(),
		Focused: next.GetFocused(),
	}
	if len(next.GetInput()) > 0 {
		set.Input = append([]string(nil), next.GetInput()...)
	}
	if next.GetSize() != nil {
		set.Size = &pb.Size{Width: next.GetSize().GetWidth(), Height: next.GetSize().GetHeight(), Flex: next.GetSize().GetFlex()}
	}
	if next.GetPos() != nil {
		set.Pos = &pb.Pos{X: next.GetPos().GetX(), Y: next.GetPos().GetY()}
	}
	if next.GetCursor() != nil {
		set.Cursor = &pb.Cursor{Row: next.GetCursor().GetRow(), Col: next.GetCursor().GetCol(), Shape: next.GetCursor().GetShape()}
	}
	if next.GetContent() != nil {
		set.Content = cloneContent(next.GetContent())
	}
	// visible is proto3 optional: always carry it when the effective value
	// changed, so a transition back to the default true is explicit.
	if visibleEffective(base) != visibleEffective(next) {
		visible := visibleEffective(next)
		set.Visible = &visible
	}
	return &pb.Patch{Op: OpSet, Path: clonePath(path), Box: set}
}

// replacePatch returns a replace patch carrying next's whole subtree.
func replacePatch(path []uint32, next *pb.Box) *pb.Patch {
	return &pb.Patch{Op: OpReplace, Path: clonePath(path), Box: next}
}

// setRepresentable reports whether the non-children changes from base to next
// can be expressed by a set patch, whose box only carries non-zero values
// (proto3 fields have no presence). It returns false when:
//
//   - a scalar (id/flow/style) is cleared;
//   - focused flips to false;
//   - visible is cleared;
//   - input shrinks (a shorter list cannot be merged field-by-field);
//   - size/pos/cursor/content is cleared.
//
// Growing input and changed sub-messages are representable because the host
// merges the non-zero fields of the set box into the cached node.
func setRepresentable(base, next *pb.Box) bool {
	if (base.GetId() != "" && next.GetId() == "") ||
		(base.GetFlow() != "" && next.GetFlow() == "") ||
		(base.GetStyle() != "" && next.GetStyle() == "") {
		return false
	}
	if base.GetFocused() && !next.GetFocused() {
		return false
	}
	// visible is a proto3 optional field (presence), so hiding a box with
	// visible=false is representable by a set (PROTOCOL §2.1).
	// Repeated fields merge by concatenation on the host, so a set may only
	// carry them when the base side is empty (append == replace from empty).
	if len(base.GetInput()) > 0 && !equalStrings(base.GetInput(), next.GetInput()) {
		return false
	}
	if base.GetSize() != nil && next.GetSize() == nil {
		return false
	}
	if base.GetPos() != nil && next.GetPos() == nil {
		return false
	}
	if base.GetCursor() != nil && next.GetCursor() == nil {
		return false
	}
	if base.GetContent() != nil && next.GetContent() == nil {
		return false
	}
	return sizeRepresentable(base.GetSize(), next.GetSize()) &&
		posRepresentable(base.GetPos(), next.GetPos()) &&
		cursorRepresentable(base.GetCursor(), next.GetCursor()) &&
		contentRepresentable(base.GetContent(), next.GetContent())
}

// sizeRepresentable reports that no size field is cleared. proto3 omits zero
// scalars, so a set box could not clear a non-zero width/height/flex.
func sizeRepresentable(base, next *pb.Size) bool {
	if base == nil || next == nil {
		return true
	}
	return !(base.GetWidth() != 0 && next.GetWidth() == 0) &&
		!(base.GetHeight() != 0 && next.GetHeight() == 0) &&
		!(base.GetFlex() != 0 && next.GetFlex() == 0)
}

func posRepresentable(base, next *pb.Pos) bool {
	if base == nil || next == nil {
		return true
	}
	return !(base.GetX() != 0 && next.GetX() == 0) && !(base.GetY() != 0 && next.GetY() == 0)
}

func cursorRepresentable(base, next *pb.Cursor) bool {
	if base == nil || next == nil {
		return true
	}
	return !(base.GetRow() != 0 && next.GetRow() == 0) &&
		!(base.GetCol() != 0 && next.GetCol() == 0) &&
		!(base.GetShape() != "" && next.GetShape() == "")
}

// contentRepresentable reports that a content set can merge without leaving a
// stale field (a cleared text/self/lines/props entry cannot be expressed).
func contentRepresentable(base, next *pb.Content) bool {
	if base == nil || next == nil {
		return true
	}
	if base.GetText() != "" && next.GetText() == "" {
		return false
	}
	if base.GetSelf() != "" && next.GetSelf() == "" {
		return false
	}
	if len(base.GetLines()) > 0 && !equalStrings(base.GetLines(), next.GetLines()) {
		return false
	}
	if len(base.GetProps()) > 0 && len(next.GetProps()) < len(base.GetProps()) {
		return false
	}
	for key := range base.GetProps() {
		if _, ok := next.GetProps()[key]; !ok {
			return false
		}
	}
	return true
}

// scalarsDiffer reports whether any non-children field differs between base
// and next, comparing every field directly (no children-stripped clone). A
// nil-vs-empty slice or map is treated as equal because it is equal on the wire
// after a proto round trip.
func scalarsDiffer(base, next *pb.Box) bool {
	if base.GetId() != next.GetId() ||
		base.GetFlow() != next.GetFlow() ||
		base.GetStyle() != next.GetStyle() ||
		base.GetFocused() != next.GetFocused() {
		return true
	}
	if visibleEffective(base) != visibleEffective(next) {
		return true
	}
	if !equalStrings(base.GetInput(), next.GetInput()) {
		return true
	}
	if !equalSize(base.GetSize(), next.GetSize()) ||
		!equalPos(base.GetPos(), next.GetPos()) ||
		!equalCursor(base.GetCursor(), next.GetCursor()) {
		return true
	}
	return !equalContent(base.GetContent(), next.GetContent())
}

// equalContent compares two content messages treating nil and empty maps as
// equal, then compares the remaining fields directly.
func equalContent(a, b *pb.Content) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if a.GetText() != b.GetText() || a.GetSelf() != b.GetSelf() || !equalStrings(a.GetLines(), b.GetLines()) {
		return false
	}
	if len(a.GetProps()) != len(b.GetProps()) {
		return false
	}
	for key, value := range a.GetProps() {
		if b.GetProps()[key] != value {
			return false
		}
	}
	return true
}

// cloneContent deep-copies the sub-messages a set patch carries (props and
// lines), so the patch never aliases the model's view.
func cloneContent(src *pb.Content) *pb.Content {
	out := &pb.Content{Text: src.GetText(), Self: src.GetSelf()}
	if len(src.GetLines()) > 0 {
		out.Lines = append([]string(nil), src.GetLines()...)
	}
	if len(src.GetProps()) > 0 {
		out.Props = make(map[string]string, len(src.GetProps()))
		for key, value := range src.GetProps() {
			out.Props[key] = value
		}
	}
	return out
}

// visibleEffective returns a box's effective visibility: nil means the
// protocol default true (PROTOCOL §9.1).
func visibleEffective(box *pb.Box) bool {
	if box.Visible == nil {
		return true
	}
	return *box.Visible
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// childPath appends one child index to path, reusing path's backing array when
// it has room so a depth-first walk allocates little.
func childPath(path []uint32, index uint32) []uint32 {
	if cap(path) > len(path) {
		return append(path[:len(path):len(path)], index)
	}
	return append(append([]uint32(nil), path...), index)
}

func clonePath(path []uint32) []uint32 {
	if path == nil {
		return nil
	}
	return append([]uint32(nil), path...)
}

// CommitDelta commits next as a VIEW_DELTA against base, falling back to a full
// VIEW when a delta is not possible or not worth it (PROTOCOL §2.1). It returns
// true when a delta frame was sent.
//
// The decision is a byte-cost guard: the delta is marshalled into an existing
// pooled buffer and its size compared with the full VIEW bytes. If the delta is
// at least as large, the full snapshot wins. A full VIEW is also forced when
// there is no baseline, the host did not advertise features["view_delta"], or
// no commit has succeeded in this epoch.
//
// Like Commit, one call advances rev exactly once and takes writeMu before mu
// so write order equals revision order. The baseline is replaced on every
// successful send (full or delta), so the next call diffs against the frame the
// host is known to hold. base is treated as an optimisation hint only: the
// client's own committed baseline is authoritative and is used in its place.
func (c *Client) CommitDelta(base, next *pb.Box, keys Keys) (bool, error) {
	if next == nil {
		return false, ErrNilRoot
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.Lock()
	if c.hello == nil {
		c.mu.Unlock()
		return false, ErrNoHello
	}
	epoch := c.epoch
	prevRev := c.rev
	rev := prevRev + 1
	supported := c.hello.GetFeatures()[ViewDeltaFeature]
	// The client's committed baseline is authoritative: a delta is only valid
	// against the frame the host is known to hold. The caller's base argument
	// is a hint; it is used only when it matches the committed baseline, so a
	// stale hint (or a DropBase) always falls back to a full VIEW.
	if base == nil || base != c.base {
		base = c.base
	}
	c.mu.Unlock()

	delta := supported && base != nil
	var patches []*pb.Patch
	if delta {
		patches, delta = DiffView(base, next)
	}

	// Cost guard: marshal the delta into the pooled scratch, but compare it with
	// the full VIEW's exact frame size from proto.Size (the same byte count
	// wire.Marshal would produce) instead of marshalling the whole tree as
	// well — a 10k-node snapshot marshalled per frame would dominate the cost
	// of the tiny patch it is guarding against.
	deltaBuf := getFrameBuf()
	defer putFrameBuf(deltaBuf)
	var deltaFrame []byte
	if delta {
		frame, err := c.marshalDelta((*deltaBuf)[:0], patches, keys, epoch, rev, prevRev)
		if err != nil {
			delta = false
		} else {
			deltaFrame = frame
		}
	}
	if delta && len(deltaFrame) >= c.fullViewFrameLen(next, keys, epoch, rev) {
		delta = false
	}

	if !delta {
		// The full-VIEW path has no cost guard to run, so wire's pooled
		// Encoder writes it (the same path Commit uses) and we borrow no
		// buffer.
		if err := c.encodeView(next, keys, epoch, rev); err != nil {
			return false, err
		}
	} else {
		if _, err := c.w.Write(deltaFrame); err != nil {
			return false, err
		}
	}

	c.mu.Lock()
	c.rev = rev
	c.base = next
	c.baseRev = rev
	c.mu.Unlock()
	return delta, nil
}

// fullViewFrameLen returns the exact wire size of a full VIEW frame (u32
// length | u8 type | payload) from the pooled envelope, using proto.Size so the
// cost guard never has to marshal the whole tree. It must stay in lockstep with
// encodeView (core.go): both build the same View message.
//
// The envelope (epoch/rev/keys) is sized fresh; the root subtree is sized with
// MarshalOptions.UseCachedSize, so a subtree that did not change since its size
// was last computed (a memoized box keeps its pointer, see sdk/app.Memo) is
// O(1) and only the changed path is walked. UseCachedSize is valid here
// because a cached size is only ever reused for a pointer whose message has
// not changed: program trees are immutable once committed (the delta contract),
// and every box sized by a full VIEW/marshal or by an earlier call stores its
// size in the protobuf size cache. The root field's tag and length prefix are
// added explicitly, which keeps the returned length byte-exact with the frame
// Commit writes (pinned by TestFullViewFrameLenMatchesMarshal).
func (c *Client) fullViewFrameLen(root *pb.Box, keys Keys, epoch, rev uint64) int {
	scratch := viewScratchPool.Get().(*viewScratch)
	scratch.keys.Claim = append(scratch.keys.Claim[:0], keys.Claim...)
	scratch.keys.All = keys.All
	scratch.view.Epoch = epoch
	scratch.view.Rev = rev
	scratch.view.Keys = &scratch.keys
	scratch.view.Root = nil
	envelope := gproto.Size(&scratch.view)
	scratch.view.Keys = nil
	scratch.keys.All = false
	scratch.keys.Claim = scratch.keys.Claim[:0]
	viewScratchPool.Put(scratch)
	rootSize := gproto.MarshalOptions{UseCachedSize: true}.Size(root)
	return 4 + 1 + envelope + protowire.SizeTag(viewRootField) + protowire.SizeBytes(rootSize)
}

// marshalDelta appends one VIEW_DELTA frame to dst from the pooled envelope. It
// does not advance rev; CommitDelta commits after a successful write.
func (c *Client) marshalDelta(dst []byte, patches []*pb.Patch, keys Keys, epoch, rev, revBase uint64) ([]byte, error) {
	scratch := deltaScratchPool.Get().(*deltaScratch)
	scratch.keys.Claim = append(scratch.keys.Claim[:0], keys.Claim...)
	scratch.keys.All = keys.All
	scratch.delta.Epoch = epoch
	scratch.delta.Rev = rev
	scratch.delta.RevBase = revBase
	scratch.delta.Keys = &scratch.keys
	scratch.delta.Patches = patches
	frame, err := wire.MarshalAppend(dst, wire.TypeViewDelta, &scratch.delta, 0)
	scratch.delta.Keys = nil
	scratch.delta.Patches = nil
	scratch.keys.All = false
	scratch.keys.Claim = scratch.keys.Claim[:0]
	deltaScratchPool.Put(scratch)
	return frame, err
}
