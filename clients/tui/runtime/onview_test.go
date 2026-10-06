package runtime

import (
	"bytes"
	"io"
	"sync/atomic"
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// An accepted VIEW/VIEW_DELTA must notify the host immediately, so the frame
// loop repaints without polling on a ticker.
func TestAcceptedViewFiresOnViewCallback(t *testing.T) {
	var calls atomic.Int64
	s := NewSession(Options{ViewID: "v", Cols: 80, Rows: 24, OnView: func() { calls.Add(1) }}, bytes.NewReader(nil), io.Discard)

	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: box("root")}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("accepted VIEW callback calls = %d, want 1", calls.Load())
	}
	// A stale rev is dropped and must not notify.
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: box("root")}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("stale VIEW must not notify, calls = %d", calls.Load())
	}
}

func TestAcceptedViewDeltaFiresOnViewCallback(t *testing.T) {
	root := &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{textBox("a", "a")}}
	var calls atomic.Int64
	s := NewSession(Options{ViewID: "v", Cols: 80, Rows: 24, OnView: func() { calls.Add(1) }}, bytes.NewReader(nil), io.Discard)
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: root}); err != nil {
		t.Fatal(err)
	}
	base := calls.Load()
	delta := &pb.ViewDelta{Epoch: 1, Rev: 2, RevBase: 1, Patches: []*pb.Patch{
		{Op: "set", Path: []uint32{0}, Box: &pb.Box{Id: "a", Content: &pb.Content{Text: "b"}}},
	}}
	if err := s.HandleViewDelta(delta); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != base+1 {
		t.Fatalf("accepted VIEW_DELTA callback calls = %d, want %d", calls.Load(), base+1)
	}
}
