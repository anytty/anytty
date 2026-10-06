package kernel

import (
	"strconv"
	"testing"
)

// Same-process A/B control for the cached solver. Both benchmarks solve the
// same 10k-node tree; the scratch variant solves from a clean tree every
// iteration, the cached variant changes one leaf and invalidates only its path
// (root -> row -> leaf), reusing every untouched row. Running them in one
// process cancels machine load, so the gap is the kernel-only reuse win.
//
//	b.TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test ./clients/tui/kernel \
//	    -run '^$' -bench 'ABKernel' -benchmem -count=5

func BenchmarkABKernelFromScratch_10000(b *testing.B) { benchABKernelScratch(b, 10000) }
func BenchmarkABKernelCached_10000(b *testing.B)      { benchABKernelCached(b, 10000) }

// abKernelTree is benchKernelTree plus a slice of addresses of the row
// containers, so the cached benchmark can invalidate exactly one path.
func abKernelTree(rows int) (*Node, []*Node) {
	root := benchKernelTree(rows)
	rowsNodes := make([]*Node, rows)
	for i := range root.Children {
		rowsNodes[i] = &root.Children[i]
	}
	return root, rowsNodes
}

func benchABKernelScratch(b *testing.B, rows int) {
	root, _ := abKernelTree(rows)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := Layout(root, 120, 40)
		if f.RectCount() == 0 {
			b.Fatal("empty layout")
		}
	}
}

func benchABKernelCached(b *testing.B, rows int) {
	root, rowsNodes := abKernelTree(rows)
	LayoutCached(root, 120, 40)
	row := rowsNodes[rows/2]
	target := &row.Children[0]
	flip := false
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flip = !flip
		if flip {
			target.Content = &Content{Text: "changed"}
		} else {
			target.Content = &Content{Text: "cell-" + strconv.Itoa(rows/2)}
		}
		// Invalidate only the changed path, mirroring the host, which rebuilds
		// kernel nodes along a delta's patched path and reuses the rest.
		root.SetCached(nil)
		row.SetCached(nil)
		target.SetCached(nil)
		f := LayoutCached(root, 120, 40)
		if f.RectCount() == 0 {
			b.Fatal("empty layout")
		}
	}
}
