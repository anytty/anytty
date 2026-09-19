package runtime

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/kernel"
)

func lineFrame(text string) kernel.Frame {
	return kernel.Frame{
		Rects: map[string]kernel.Rect{},
		Lines: []kernel.Line{{Text: text}},
	}
}

func layerKinds(c Composite) []LayerKind {
	out := make([]LayerKind, len(c.Layers))
	for i, l := range c.Layers {
		out[i] = l.Kind
	}
	return out
}

func TestComposeZOrder(t *testing.T) {
	program := lineFrame("base")
	program.OverlayFrames = []kernel.Frame{lineFrame("overlay-1"), lineFrame("overlay-2")}
	component := lineFrame("component")
	notice := lineFrame("notice")
	core := lineFrame("core")

	c := Compose(program, []kernel.Frame{component}, &core, &notice)
	want := []LayerKind{LayerProgram, LayerComponent, LayerOverlay, LayerOverlay, LayerNotice, LayerCore}
	got := layerKinds(c)
	if len(got) != len(want) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("layer %d = %v, want %v (all: %v)", i, got[i], want[i], got)
		}
	}
	if len(c.Layers[0].Frame.OverlayFrames) != 0 {
		t.Fatal("program overlays must be moved out of the base layer")
	}
	if c.Layers[2].Frame.Lines[0].Text != "overlay-1" || c.Layers[3].Frame.Lines[0].Text != "overlay-2" {
		t.Fatalf("overlay declaration order lost: %v %v", c.Layers[2], c.Layers[3])
	}
	last := c.Layers[len(c.Layers)-1]
	if last.Kind != LayerCore || last.Frame.Lines[0].Text != "core" {
		t.Fatalf("core overlay is not last: %+v", last)
	}
}

func TestComposeSkipsNil(t *testing.T) {
	c := Compose(lineFrame("base"), nil, nil, nil)
	if got := layerKinds(c); len(got) != 1 || got[0] != LayerProgram {
		t.Fatalf("layers = %v, want [program]", got)
	}
}

func TestSessionComposeCoreLast(t *testing.T) {
	h := newHarness(t, Options{ViewID: "v", Cols: 80, Rows: 24})
	popup := &pb.Box{Id: "popup", Pos: &pb.Pos{X: 2, Y: 2}, Content: &pb.Content{Text: "pick"}}
	h.sendView(view(1, 1, nil, false, textBox("body", "body"), popup))

	core := lineFrame("confirm")
	h.s.OpenCoreOverlay(core)
	notice := lineFrame("notice")
	c := h.s.Compose([]kernel.Frame{lineFrame("term-surface")}, &notice)

	want := []LayerKind{LayerProgram, LayerComponent, LayerOverlay, LayerNotice, LayerCore}
	got := layerKinds(c)
	if len(got) != len(want) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("layer %d = %v, want %v", i, got[i], want[i])
		}
	}
	if c.Layers[len(c.Layers)-1].Frame.Lines[0].Text != "confirm" {
		t.Fatal("core overlay must be unbeatable")
	}
}
