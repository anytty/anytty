package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/render"
	"github.com/anytty/anytty/clients/tui/runtime"
	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func renderFrameText(frame *render.Frame) string {
	var b strings.Builder
	for y := 0; y < frame.Rows(); y++ {
		for x := 0; x < frame.Cols(); x++ {
			b.WriteString(frame.CellAt(x, y).Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// TestViewRendersThroughTheRuntime lays the committed view out with the real
// kernel/compositor: it catches geometry regressions (negative sizes, boxes
// outside the viewport, a mode bar replacing the tab row) that a box-tree
// assertion cannot see.
func TestViewRendersThroughTheRuntime(t *testing.T) {
	m := seededModel(t)
	alpha := terminalSource("terminal:local:alpha", "alpha", "local")
	bindSource(t, m, alpha)
	prefix(m, "v")

	s := runtime.NewSession(runtime.Options{ViewID: "view-1", Cols: m.cols, Rows: m.rows}, &bytes.Buffer{}, &bytes.Buffer{})
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 1, Root: m.View(), Keys: &pb.Keys{All: true}}); err != nil {
		t.Fatalf("handle view: %v", err)
	}
	text := renderFrameText(s.ComposeFrame(nil, nil))
	for _, want := range []string{"pane 1 · alpha", "Agents", "Spaces", "connected", "1:main"} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered frame is missing %q:\n%s", want, text)
		}
	}
	// The body owns every row under the tab bar: no size/status text and no
	// reserved toast row (the removed status line used to show "100x30").
	if strings.Contains(text, "100x30") || strings.Contains(text, "layout restore failed") {
		t.Fatalf("rendered frame has a persistent status/toast row:\n%s", text)
	}

	// The prefix mode bar replaces the tab row.
	key(m, "ctrl-b")
	if err := s.HandleView(&pb.View{Epoch: 1, Rev: 2, Root: m.View(), Keys: &pb.Keys{All: true}}); err != nil {
		t.Fatalf("handle mode view: %v", err)
	}
	text = renderFrameText(s.ComposeFrame(nil, nil))
	if !strings.Contains(text, "PREFIX") || !strings.Contains(text, "q detach") {
		t.Fatalf("rendered mode bar is missing:\n%s", text)
	}
}
