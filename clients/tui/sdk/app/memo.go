package app

import pb "github.com/anytty/anytty/proto/ui/protobuf"

// A Memo lets a View hand back the SAME *pb.Box pointer for an unchanged
// subtree across frames. The core diff short-circuits on pointer identity
// (core.DiffView's boxesEqual fast path), so a memoized tree turns a one-leaf
// change on a large tree into O(changed) diff work instead of a full-tree
// walk.
//
// A Memo is zero-value ready. It is not safe for concurrent use: call it from
// the loop goroutine, normally inside Model.View.
//
// # Frames
//
// Caching is frame-scoped so memory stays bounded even when keys come and go
// (terminal ids, list rows). BeginFrame starts a frame, Box reuses boxes built
// in the previous frame and builds the rest, and EndFrame drops every key that
// was not requested in the frame. A program run by Program picks up
// Program.Memo and does not manage frames itself: Program calls BeginFrame
// before Model.View and EndFrame after the batch's commit (exactly one commit
// per batch). Without BeginFrame/EndFrame the zero value still works: every
// miss is cached until an EndFrame, so a one-shot View can call Box directly.
//
// # Keys
//
// Keys must be comparable, like a string, an int, or a struct of comparable
// fields. A slice, map or func key panics naturally on the map lookup. Use a
// key that captures everything the subtree depends on: the same key must
// always map to the same tree, or the stale box is returned. A node whose
// descendants can change must derive its key from that state (typically by
// including the child keys) or must not be memoized.
//
// # Immutability
//
// Returned boxes are immutable: a program must not mutate a box after Box
// hands it out. The same pointer is shared with the committed baseline and
// with every later frame that reuses the key, so an in-place mutation would
// corrupt the diff (the pointer fast path would treat the mutated subtree as
// unchanged) and any host cache derived from it.
type Memo struct {
	// prev holds the boxes built in the previous frame. A key requested again
	// in the current frame is moved into cur; a key that is not requested is
	// dropped when the frame ends, so both maps track the live keys only.
	prev map[any]*pb.Box
	// cur holds the boxes built (or reused) in the current frame.
	cur map[any]*pb.Box
}

// BeginFrame starts a memo frame. Box calls after it reuse boxes from the
// previous frame and rebuild the rest; EndFrame drops everything that was not
// built or reused in this frame. Program calls BeginFrame before Model.View,
// so a program driven by Program does not call it.
func (m *Memo) BeginFrame() {
	if m == nil {
		return
	}
	// Discard a partial frame (a previous BeginFrame without EndFrame): only
	// complete frames may become the reuse base.
	if m.cur != nil {
		clear(m.cur)
	}
}

// EndFrame ends the current frame: every key not requested since BeginFrame
// is dropped and the boxes requested in this frame become the reuse base for
// the next one. It must be paired with BeginFrame; Program calls it after the
// batch's commit.
func (m *Memo) EndFrame() {
	if m == nil {
		return
	}
	m.prev, m.cur = m.cur, m.prev
	if m.cur != nil {
		clear(m.cur)
	}
}

// Box returns the box for key. When key was already built in the current frame
// (or, on its first request in the frame, in the previous one) the cached
// pointer is returned and build is not called. Otherwise build runs and its
// result is cached for the rest of the frame. The returned box must not be
// mutated (see Memo).
func (m *Memo) Box(key any, build func() *pb.Box) *pb.Box {
	if m == nil {
		// A nil memo is a no-op, matching Program.Memo == nil.
		return build()
	}
	if box, ok := m.cur[key]; ok {
		return box
	}
	if box, ok := m.prev[key]; ok {
		if m.cur == nil {
			m.cur = make(map[any]*pb.Box)
		}
		m.cur[key] = box
		return box
	}
	box := build()
	if m.cur == nil {
		m.cur = make(map[any]*pb.Box)
	}
	m.cur[key] = box
	return box
}
