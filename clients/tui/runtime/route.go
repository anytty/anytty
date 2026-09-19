package runtime

// Input routing per PROTOCOL §6.5 (the single priority table, mirrored by
// SCENARIOS.zh-CN.md §7).
//
//	1. Ctrl-Q is always reserved; Ctrl-C is reserved only while a core
//	   overlay is open.
//	2. An open core overlay owns every input.
//	3. VIEW.keys.all=true sends keys to the program.
//	4. A key named in VIEW.keys.claim goes to the program.
//	5. With a focused content source:
//	     key   -> terminal declaring "key" writes PTY; non-terminal
//	              component declaring "key" gets the component event;
//	     paste -> focused terminal declaring "paste" writes PTY, else program;
//	     mouse -> only a hit on the focused terminal with mouse tracking on
//	              is written to the PTY, else program;
//	     wheel -> only focused + terminal mouse tracking + the focused panel
//	              declaring "wheel" writes PTY, else program.
//	6. Everything else goes to the program; nothing may be silently dropped.

// Destination is where one input event ends up.
type Destination uint8

const (
	// DestinationProgram sends an event frame to the layout program.
	DestinationProgram Destination = iota
	// DestinationHost is consumed by the host (reserved keys, core overlay).
	DestinationHost
	// DestinationPTY is written to the focused terminal's PTY after the host
	// encodes it (PROTOCOL §6.8).
	DestinationPTY
	// DestinationComponent is handed to the focused builtin component.
	DestinationComponent
)

func (d Destination) String() string {
	switch d {
	case DestinationProgram:
		return "program"
	case DestinationHost:
		return "host"
	case DestinationPTY:
		return "pty"
	case DestinationComponent:
		return "component"
	default:
		return "unknown"
	}
}

// InputKind is the normalized input class of an event.
type InputKind string

const (
	// InputKey is a normalized key press.
	InputKey InputKind = "key"
	// InputPaste is one complete paste chunk (PROTOCOL §6.8).
	InputPaste InputKind = "paste"
	// InputMouse is a mouse action (press/drag/release).
	InputMouse InputKind = "mouse"
	// InputWheel is a wheel delta.
	InputWheel InputKind = "wheel"
)

// Reserved key names (PROTOCOL §6.6 / §6 invariant 6).
const (
	KeyCtrlQ = "ctrl-q"
	KeyCtrlC = "ctrl-c"
)

// Focus describes the focused content source of the last accepted view.
// Input lists the box's declared input kinds; IsTerminal and MouseTracking
// come from the host component/source registry, never from the program.
type Focus struct {
	ID            string
	IsTerminal    bool
	MouseTracking bool
	Input         []string
}

// Accepts reports whether the focused box declares the input kind.
func (f *Focus) Accepts(kind InputKind) bool {
	if f == nil {
		return false
	}
	for _, in := range f.Input {
		if in == string(kind) {
			return true
		}
	}
	return false
}

// RouteState is the routing-relevant slice of the session state (VIEW keys
// plus the focused source and the core overlay flag).
type RouteState struct {
	CoreOverlay bool
	KeysAll     bool
	Claim       []string
	Focus       *Focus
}

func (s RouteState) claims(key string) bool {
	for _, c := range s.Claim {
		if c == key {
			return true
		}
	}
	return false
}

// InputEvent is one normalized input to route.
type InputEvent struct {
	Kind InputKind
	// Key is the normalized key name for InputKey (e.g. "ctrl-p").
	Key string
	// HitFocused is true when a mouse event hit the focused source's box.
	HitFocused bool
}

// Route applies the §6.5 priority table and returns the single destination.
// It is a pure function of its arguments.
func Route(ev InputEvent, st RouteState) Destination {
	if ev.Kind == InputKey {
		switch ev.Key {
		case KeyCtrlQ:
			return DestinationHost
		case KeyCtrlC:
			if st.CoreOverlay {
				return DestinationHost
			}
		}
	}

	if st.CoreOverlay {
		return DestinationHost
	}

	if ev.Kind == InputKey {
		if st.KeysAll {
			return DestinationProgram
		}
		if st.claims(ev.Key) {
			return DestinationProgram
		}
	}

	if st.Focus != nil {
		switch ev.Kind {
		case InputKey:
			if st.Focus.Accepts(InputKey) {
				if st.Focus.IsTerminal {
					return DestinationPTY
				}
				return DestinationComponent
			}
		case InputPaste:
			if st.Focus.IsTerminal && st.Focus.Accepts(InputPaste) {
				return DestinationPTY
			}
		case InputMouse:
			if st.Focus.IsTerminal && st.Focus.MouseTracking && ev.HitFocused {
				return DestinationPTY
			}
		case InputWheel:
			if st.Focus.IsTerminal && st.Focus.MouseTracking && st.Focus.Accepts(InputWheel) {
				return DestinationPTY
			}
		}
	}

	return DestinationProgram
}
