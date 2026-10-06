package runtime

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
)

// Same-process A/B control for the incremental layout path. Both benchmarks
// apply the same one-leaf patch to the same 10k-node tree; the control then
// rebuilds the whole kernel tree and solves it from scratch (exactly what the
// host did before subtree reuse), while BenchmarkHandleViewDeltaLeaf_10000 uses
// the cache. Running them in one process cancels machine load, so the
// difference is the real reuse win (and any overhead it adds).
//
//	b.TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/runtime \
//	    -run '^$' -bench 'DeltaLeaf_10000|DeltaLeafControl_10000' -benchmem -count=5

func BenchmarkDeltaLeafControl_10000(b *testing.B) { benchDeltaLeafControl(b, 10000) }

func benchDeltaLeafControl(b *testing.B, rows int) {
	patch := &pb.Patch{
		Op:   "replace",
		Path: []uint32{uint32(rows / 2), 0},
		Box:  &pb.Box{Content: &pb.Content{Text: "changed"}},
	}
	base := benchTree(rows)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		next, reason := applyPatches(base, []*pb.Patch{patch})
		if reason != "" {
			b.Fatal(reason)
		}
		var scratch Session
		root := scratch.nodeFull(nil, next)
		_ = kernel.Layout(root, 120, 40)
	}
}
