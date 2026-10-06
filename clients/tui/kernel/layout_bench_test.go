package kernel

import (
	"strconv"
	"testing"
)

// benchKernelTree builds a deterministic tree of 2*rows+1 nodes that mirrors
// the runtime host-side VIEW benchmark: a column root with `rows` row
// containers, each holding two text leaves. Layout on it is the dominant cost
// of applying a VIEW, so this benchmark isolates kernel allocation behavior.
func benchKernelTree(rows int) *Node {
	children := make([]Node, 0, rows)
	for i := 0; i < rows; i++ {
		children = append(children, Node{
			ID:   "row-" + strconv.Itoa(i),
			Flow: FlowRow,
			Size: Size{Height: 1},
			Children: []Node{
				{Content: &Content{Text: "cell-" + strconv.Itoa(i)}},
				{Content: &Content{Text: "value"}, Size: Size{Width: 8}},
			},
		})
	}
	return &Node{ID: "root", Flow: FlowCol, Children: children}
}

func BenchmarkKernelLayout_100(b *testing.B)   { benchKernelLayout(b, 100) }
func BenchmarkKernelLayout_1000(b *testing.B)  { benchKernelLayout(b, 1000) }
func BenchmarkKernelLayout_10000(b *testing.B) { benchKernelLayout(b, 10000) }

func benchKernelLayout(b *testing.B, rows int) {
	root := benchKernelTree(rows)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := Layout(root, 120, 40)
		if f.RectCount() == 0 {
			b.Fatal("empty layout")
		}
	}
}
