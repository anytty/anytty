package runtime

import "github.com/anytty/anytty/clients/tui/kernel"

// LayerKind identifies one z-order band of a composite (PROTOCOL §6, §9.1).
type LayerKind uint8

const (
	// LayerProgram is the program's regular-flow output.
	LayerProgram LayerKind = iota
	// LayerComponent is a builtin component frame (e.g. a terminal surface).
	LayerComponent
	// LayerOverlay is one program `pos` subtree, in declaration order.
	LayerOverlay
	// LayerNotice is a host-rendered non-request notice.
	LayerNotice
	// LayerCore is the host core overlay; it is always the last layer and
	// can never be covered by the program.
	LayerCore
)

func (k LayerKind) String() string {
	switch k {
	case LayerProgram:
		return "program"
	case LayerComponent:
		return "component"
	case LayerOverlay:
		return "overlay"
	case LayerNotice:
		return "notice"
	case LayerCore:
		return "core"
	default:
		return "unknown"
	}
}

// Layer is one band of a Composite.
type Layer struct {
	Kind  LayerKind
	Frame kernel.Frame
}

// Composite is the fully ordered render stack. Later layers draw on top of
// earlier ones.
type Composite struct {
	Layers []Layer
}

// Compose merges the program frame (regular flow plus its `pos` overlay
// subtrees), the builtin component frames, a host notice and the core
// overlay into the fixed z-order:
//
//	program regular flow
//	component frames (given order)
//	program `pos` overlay subtrees (declaration order)
//	notice
//	core overlay (always last, never coverable)
//
// The program frame's own OverlayFrames are moved into their own layers, so
// a later component frame can never paint over a program overlay and the
// core overlay always wins.
func Compose(program kernel.Frame, components []kernel.Frame, core *kernel.Frame, notice *kernel.Frame) Composite {
	base := program
	overlays := base.OverlayFrames
	base.OverlayFrames = nil

	c := Composite{Layers: make([]Layer, 0, len(components)+len(overlays)+3)}
	c.Layers = append(c.Layers, Layer{Kind: LayerProgram, Frame: base})
	for _, f := range components {
		c.Layers = append(c.Layers, Layer{Kind: LayerComponent, Frame: f})
	}
	for _, f := range overlays {
		c.Layers = append(c.Layers, Layer{Kind: LayerOverlay, Frame: f})
	}
	if notice != nil {
		c.Layers = append(c.Layers, Layer{Kind: LayerNotice, Frame: *notice})
	}
	if core != nil {
		c.Layers = append(c.Layers, Layer{Kind: LayerCore, Frame: *core})
	}
	return c
}

// Compose builds the composite for the last accepted view, the installed
// core overlay and the given component frames and notice.
func (s *Session) Compose(components []kernel.Frame, notice *kernel.Frame) Composite {
	s.mu.Lock()
	program := s.frame
	core := s.coreOverlay
	s.mu.Unlock()
	var corePtr *kernel.Frame
	if core != nil {
		f := *core
		corePtr = &f
	}
	return Compose(program, components, corePtr, notice)
}
