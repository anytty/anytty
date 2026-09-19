package proto

import (
	"errors"
	"fmt"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// Type is the fixed first-level frame type (PROTOCOL §0). Numbers are
// append-only.
type Type uint8

const (
	// TypeHello is sent host -> program on start/restart.
	TypeHello Type = 1
	// TypeView is sent program -> host (full view snapshot).
	TypeView Type = 2
	// TypeEvent is sent host -> program.
	TypeEvent Type = 3
	// TypeResult is sent program -> host (method call).
	TypeResult Type = 4
	// TypeResponse is sent host -> program (exactly one per Result).
	TypeResponse Type = 5
)

// Valid reports whether t is a known frame type.
func (t Type) Valid() bool {
	return t >= TypeHello && t <= TypeResponse
}

func (t Type) String() string {
	switch t {
	case TypeHello:
		return "HELLO"
	case TypeView:
		return "VIEW"
	case TypeEvent:
		return "EVENT"
	case TypeResult:
		return "RESULT"
	case TypeResponse:
		return "RESPONSE"
	default:
		return "UNKNOWN"
	}
}

// Role identifies one endpoint of the connection.
type Role uint8

const (
	// RoleHost is the runtime side: receives VIEW/RESULT, sends
	// HELLO/EVENT/RESPONSE.
	RoleHost Role = iota
	// RoleProgram is the layout-program side: receives HELLO/EVENT/RESPONSE,
	// sends VIEW/RESULT.
	RoleProgram
)

// CanSend reports whether frames of type t may be sent by this endpoint.
func (r Role) CanSend(t Type) bool {
	if !t.Valid() {
		return false
	}
	if r == RoleHost {
		return t == TypeHello || t == TypeEvent || t == TypeResponse
	}
	return t == TypeView || t == TypeResult
}

// CanReceive reports whether frames of type t may be received by this
// endpoint. It is the exact inverse of CanSend.
func (r Role) CanReceive(t Type) bool {
	if !t.Valid() {
		return false
	}
	if r == RoleHost {
		return t == TypeView || t == TypeResult
	}
	return t == TypeHello || t == TypeEvent || t == TypeResponse
}

// DefaultMaxMessageBytes is the default single-frame limit when none is
// configured (PROTOCOL §1 example).
const DefaultMaxMessageBytes uint32 = 1 << 20

// ErrorKind classifies protocol envelope errors. Payload decode errors live
// in the generated protobuf code and are wrapped with KindPayload.
type ErrorKind uint8

const (
	// KindZeroLength: the length prefix was 0.
	KindZeroLength ErrorKind = iota + 1
	// KindOversize: the frame exceeded max_message_bytes.
	KindOversize
	// KindTruncated: the stream ended in the middle of a frame.
	KindTruncated
	// KindUnknownType: the type byte is not one of 1..5.
	KindUnknownType
	// KindDirection: the type is valid but illegal for this endpoint.
	KindDirection
	// KindPayload: the protobuf payload failed to decode.
	KindPayload
)

// Error is a frame envelope error.
type Error struct {
	Kind ErrorKind
	// Type is the frame type when it could be read.
	Type Type
	// Limit is max_message_bytes for KindOversize.
	Limit uint32
	Err   error
}

func (e *Error) Error() string {
	switch e.Kind {
	case KindZeroLength:
		return "tui2/proto: zero-length frame"
	case KindOversize:
		return fmt.Sprintf("tui2/proto: %s frame exceeds max_message_bytes (%d)", e.Type, e.Limit)
	case KindTruncated:
		return "tui2/proto: truncated frame"
	case KindUnknownType:
		return fmt.Sprintf("tui2/proto: unknown frame type %d", uint8(e.Type))
	case KindDirection:
		return fmt.Sprintf("tui2/proto: illegal direction for %s frame", e.Type)
	case KindPayload:
		return fmt.Sprintf("tui2/proto: invalid payload for %s frame: %v", e.Type, e.Err)
	default:
		return "tui2/proto: frame error"
	}
}

// Unwrap exposes the underlying I/O or protobuf error, if any.
func (e *Error) Unwrap() error { return e.Err }

// IsKind reports whether err is a frame *Error of kind k.
func IsKind(err error, k ErrorKind) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind == k
	}
	return false
}

// NewPayload returns a new empty payload message for t, or nil for an
// unknown type.
func NewPayload(t Type) gproto.Message {
	switch t {
	case TypeHello:
		return &pb.Hello{}
	case TypeView:
		return &pb.View{}
	case TypeEvent:
		return &pb.Event{}
	case TypeResult:
		return &pb.Result{}
	case TypeResponse:
		return &pb.Response{}
	default:
		return nil
	}
}

// UnmarshalPayload decodes a frame payload into its typed message. An empty
// payload is valid and yields the zero message.
func UnmarshalPayload(t Type, payload []byte) (gproto.Message, error) {
	m := NewPayload(t)
	if m == nil {
		return nil, &Error{Kind: KindUnknownType, Type: t}
	}
	if len(payload) == 0 {
		return m, nil
	}
	if err := gproto.Unmarshal(payload, m); err != nil {
		return nil, &Error{Kind: KindPayload, Type: t, Err: err}
	}
	return m, nil
}
