package runtime

import (
	"errors"
	"strconv"

	pb "github.com/anytty/anytty/proto/ui/protobuf"

	"github.com/anytty/anytty/clients/tui/runtime/keys"
)

// ErrNoInputSink means a PTY-bound input was submitted without an InputSink.
var ErrNoInputSink = errors.New("runtime: no input sink")

// EventSink delivers host -> program EVENT frames (PROTOCOL §3). The default
// sink writes through the session encoder; a host may install its own to
// observe or reroute events. Implementations must be safe for concurrent
// use: the session serializes calls under its lock.
type EventSink interface {
	SendEvent(ev *pb.Event) error
}

// InputSink carries host-encoded bytes to a content source and answers the
// terminal capability questions the encoder needs. TerminalHandler
// implements it for PTYs; tests can implement a recorder.
type InputSink interface {
	// WriteInput writes one already-encoded payload to sourceID's PTY.
	WriteInput(sourceID string, data []byte) error
	// BracketPaste reports whether sourceID has DEC 2004 on (§6.8).
	BracketPaste(sourceID string) bool
}

// SetInputSink installs the PTY input sink (PROTOCOL §6.5 priority 5).
func (s *Session) SetInputSink(sink InputSink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputSink = sink
}

// SetEventSink installs the program event sink; nil restores the encoder.
func (s *Session) SetEventSink(sink EventSink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventSink = sink
}

// SetMouseTracking installs the terminal capability probe used by routing.
func (s *Session) SetMouseTracking(fn func(string) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mouseTracker = fn
}

func routeKind(k keys.Kind) (InputKind, bool) {
	switch k {
	case keys.KindKey:
		return InputKey, true
	case keys.KindPaste:
		return InputPaste, true
	case keys.KindMouse:
		return InputMouse, true
	case keys.KindWheel:
		return InputWheel, true
	default:
		return "", false
	}
}

// Input routes one normalized input event through the §6.5 table and
// performs the destination action:
//
//   - DestinationPTY: the keys package encodes the event (paste is chunked
//     per §6.8) and the bytes go to the focused source through the InputSink;
//   - DestinationProgram: key/paste events are retained for input.forward
//     (last 64 per view, §9.3) and delivered as EVENT frames;
//   - DestinationHost / DestinationComponent: returned for the caller, no
//     bytes are written and no event is delivered.
//
// The returned Destination is always the routing result, also on error.
func (s *Session) Input(ev keys.Event) (Destination, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshFocusLocked()
	kind, ok := routeKind(ev.Kind)
	if !ok {
		return DestinationProgram, errors.New("runtime: unknown input kind")
	}
	dst := Route(InputEvent{Kind: kind, Key: keys.Name(ev), HitFocused: ev.HitFocused}, RouteState{
		CoreOverlay: s.coreOverlay != nil,
		KeysAll:     s.keysAll,
		Claim:       s.claim,
		Focus:       s.focus,
	})
	switch ev.Kind {
	case keys.KindPaste:
		return s.inputPasteLocked(ev, dst)
	default:
		if dst != DestinationPTY {
			if dst == DestinationProgram {
				return dst, s.deliverLocked(ev)
			}
			return dst, nil
		}
		data, ok := keys.Encode(ev)
		if !ok {
			return dst, errors.New("runtime: unencodable " + ev.Kind.String() + " event")
		}
		if s.inputSink == nil {
			return dst, ErrNoInputSink
		}
		return dst, s.inputSink.WriteInput(s.focus.ID, data)
	}
}

// inputPasteLocked chunks one paste event and delivers every chunk in order,
// each with its own event id, never merged (§6.8). Chunks destined for the
// PTY are converted to bytes with the terminal's current bracket-paste mode;
// chunks destined for the program keep the normalized text and are retained
// for input.forward.
func (s *Session) inputPasteLocked(ev keys.Event, dst Destination) (Destination, error) {
	if dst != DestinationPTY && dst != DestinationProgram {
		return dst, nil
	}
	if dst == DestinationPTY && s.inputSink == nil {
		return dst, ErrNoInputSink
	}
	bracket := dst == DestinationPTY && s.inputSink.BracketPaste(s.focus.ID)
	chunks := keys.ChunkPaste(ev.Text, int(s.limits.MaxPasteBytes), bracket, s.nextEventIDLocked)
	for _, chunk := range chunks {
		if dst == DestinationPTY {
			if err := s.inputSink.WriteInput(s.focus.ID, chunk.Bytes); err != nil {
				return dst, err
			}
			continue
		}
		s.history.Add(chunk.ID, keys.Event{Kind: keys.KindPaste, Text: chunk.Text})
		if err := s.sendEventLocked(&pb.Event{Event: &pb.Event_Paste{
			Paste: &pb.PasteEvent{Id: chunk.ID, Text: chunk.Text},
		}}); err != nil {
			return dst, err
		}
	}
	return dst, nil
}

// deliverLocked sends one non-paste event to the program and retains
// key/paste events for input.forward.
func (s *Session) deliverLocked(ev keys.Event) error {
	id := s.nextEventIDLocked()
	if ev.Kind == keys.KindKey {
		s.history.Add(id, ev)
	}
	return s.sendEventLocked(eventFor(id, ev))
}

// forwardLocked answers input.forward (PROTOCOL §6.6, §9.3): the event id
// must still be in the 64-entry history and the target source must be an
// attached terminal; past that the host re-encodes the original event and
// writes it to the PTY. Programs never supply bytes.
func (s *Session) forwardLocked(r *pb.Result) error {
	params := r.Params
	ev, ok := s.history.Get(params.GetEventId())
	if !ok {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "event expired")
	}
	src := s.sourceLocked(params.GetSource())
	if src == nil {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "unknown source")
	}
	if src.GetKind() != "terminal" || !src.GetAttached() || src.GetExited() {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "source not attached")
	}
	if s.inputSink == nil {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, ErrNoInputSink.Error())
	}
	var data []byte
	switch ev.Kind {
	case keys.KindKey:
		encoded, ok := keys.EncodeKey(ev)
		if !ok {
			return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "event expired")
		}
		data = encoded
	case keys.KindPaste:
		data = keys.EncodePaste(keys.NormalizeText(ev.Text), s.inputSink.BracketPaste(params.GetSource()))
	default:
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, "event expired")
	}
	if err := s.inputSink.WriteInput(params.GetSource(), data); err != nil {
		return s.sendResponseLocked(r.Epoch, r.RequestId, false, nil, err.Error())
	}
	return s.sendResponseLocked(r.Epoch, r.RequestId, true, nil, "")
}

func (s *Session) nextEventIDLocked() string {
	s.eventSeq++
	return "ev-" + strconv.FormatUint(s.eventSeq, 10)
}

func eventFor(id string, ev keys.Event) *pb.Event {
	switch ev.Kind {
	case keys.KindKey:
		return &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{
			Id:   id,
			Key:  keys.Name(ev),
			Char: ev.Char,
		}}}
	case keys.KindMouse:
		return &pb.Event{Event: &pb.Event_Mouse{Mouse: &pb.MouseEvent{
			Action: ev.Action,
			Button: ev.Button,
			X:      int32(ev.X),
			Y:      int32(ev.Y),
			Node:   ev.Node,
		}}}
	case keys.KindWheel:
		return &pb.Event{Event: &pb.Event_Wheel{Wheel: &pb.WheelEvent{
			Delta: int32(ev.Delta),
			X:     int32(ev.X),
			Y:     int32(ev.Y),
			Node:  ev.Node,
		}}}
	default:
		return nil
	}
}
