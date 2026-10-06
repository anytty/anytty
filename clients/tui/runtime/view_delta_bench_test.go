package runtime

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	wire "github.com/anytty/anytty/proto/ui"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func benchTree(rows int) *pb.Box {
	children := make([]*pb.Box, 0, rows)
	for i := 0; i < rows; i++ {
		row := &pb.Box{
			Id:    "row-" + strconv.Itoa(i),
			Flow:  "row",
			Size:  &pb.Size{Height: 1},
			Style: "muted",
			Children: []*pb.Box{
				{Content: &pb.Content{Text: "cell-" + strconv.Itoa(i)}},
				{Content: &pb.Content{Text: "value"}, Size: &pb.Size{Width: 8}},
			},
		}
		children = append(children, row)
	}
	return &pb.Box{Id: "root", Flow: "col", Children: children}
}

func newBenchSession(rows int) *Session {
	return NewSession(Options{
		ViewID: "bench", Cols: 120, Rows: 40,
		Limits: Limits{MaxNodes: 1 << 20, MaxMessageBytes: 64 << 20},
	}, bytes.NewReader(nil), io.Discard)
}

func BenchmarkHandleViewFull_100(b *testing.B)   { benchHandleViewFull(b, 100) }
func BenchmarkHandleViewFull_1000(b *testing.B)  { benchHandleViewFull(b, 1000) }
func BenchmarkHandleViewFull_10000(b *testing.B) { benchHandleViewFull(b, 10000) }

func benchHandleViewFull(b *testing.B, rows int) {
	base := benchTree(rows)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := newBenchSession(rows)
		if err := s.HandleView(&pb.View{Epoch: 1, Rev: uint64(i + 1), Root: base}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHandleViewDeltaLeaf_100(b *testing.B)   { benchHandleViewDeltaLeaf(b, 100) }
func BenchmarkHandleViewDeltaLeaf_1000(b *testing.B)  { benchHandleViewDeltaLeaf(b, 1000) }
func BenchmarkHandleViewDeltaLeaf_10000(b *testing.B) { benchHandleViewDeltaLeaf(b, 10000) }

func benchHandleViewDeltaLeaf(b *testing.B, rows int) {
	base := benchTree(rows)
	s := newBenchSession(rows)
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: base}); err != nil {
		b.Fatal(err)
	}
	rev := uint64(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rev++
		// Fresh payload per iteration, like the wire path decodes: a hoisted
		// aliased box would be an in-process-only pattern.
		delta := &pb.ViewDelta{
			Epoch: 1, Rev: rev, RevBase: rev - 1,
			Patches: []*pb.Patch{{
				Op:   "replace",
				Path: []uint32{uint32(rows / 2), 0},
				Box:  &pb.Box{Content: &pb.Content{Text: fmt.Sprintf("changed-%d", i)}},
			}},
		}
		if err := s.HandleViewDelta(delta); err != nil {
			b.Fatal(err)
		}
	}
}

func benchFullFrame(rows int) []byte {
	frame, err := wire.Marshal(wire.TypeView, &pb.View{Epoch: 1, Rev: 1, Root: benchTree(rows)}, 0)
	if err != nil {
		panic(err)
	}
	return frame
}

func benchDeltaFrame(rows int) []byte {
	delta := &pb.ViewDelta{
		Epoch: 1, Rev: 2, RevBase: 1,
		Patches: []*pb.Patch{{
			Op:   "replace",
			Path: []uint32{uint32(rows / 2), 0},
			Box:  &pb.Box{Content: &pb.Content{Text: "changed"}},
		}},
	}
	frame, err := wire.Marshal(wire.TypeViewDelta, delta, 0)
	if err != nil {
		panic(err)
	}
	return frame
}

func BenchmarkWireFullRoundTrip_10000(b *testing.B)  { benchWireFull(b, 10000) }
func BenchmarkWireDeltaRoundTrip_10000(b *testing.B) { benchWireDelta(b, 10000) }

func benchWireFull(b *testing.B, rows int) {
	frame := benchFullFrame(rows)
	b.SetBytes(int64(len(frame)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := newBenchSession(rows)
		typ, payload, err := wire.DecodeFrame(frame, wire.RoleHost, 0)
		if err != nil {
			b.Fatal(err)
		}
		m, err := wire.UnmarshalPayload(typ, payload)
		if err != nil {
			b.Fatal(err)
		}
		if err := s.HandleView(m.(*pb.View)); err != nil {
			b.Fatal(err)
		}
	}
}

func benchWireDelta(b *testing.B, rows int) {
	frame := benchDeltaFrame(rows)
	b.SetBytes(int64(len(frame)))
	base := benchTree(rows)
	s := newBenchSession(rows)
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: base}); err != nil {
		b.Fatal(err)
	}
	rev := uint64(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rev++
		typ, payload, err := wire.DecodeFrame(frame, wire.RoleHost, 0)
		if err != nil {
			b.Fatal(err)
		}
		m, err := wire.UnmarshalPayload(typ, payload)
		if err != nil {
			b.Fatal(err)
		}
		d := m.(*pb.ViewDelta)
		d.Rev = rev
		d.RevBase = rev - 1
		if err := s.HandleViewDelta(d); err != nil {
			b.Fatal(err)
		}
	}
}
