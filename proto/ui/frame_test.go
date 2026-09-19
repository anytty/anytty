package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestRoundTrip(t *testing.T) {
	hello := &pb.Hello{
		Schema:     1,
		ViewId:     "view:client-a:1",
		Epoch:      3,
		Cols:       120,
		Rows:       32,
		Components: []string{"terminal"},
		Events:     []string{"key", "paste"},
		Methods:    []string{"terminal.attach"},
		Features:   map[string]bool{"component": true},
		Limits:     &pb.Limits{MaxNodes: 4096, MaxMessageBytes: 1 << 20},
	}
	view := &pb.View{
		Epoch: 3,
		Rev:   7,
		Keys:  &pb.Keys{Claim: []string{"ctrl-p"}, All: false},
		Root: &pb.Box{
			Id:    "root",
			Flow:  "col",
			Input: []string{"key"},
			Children: []*pb.Box{{
				Id:      "term",
				Focused: true,
				Content: &pb.Content{
					Self:  "terminal:local:main",
					Props: map[string]string{"chrome.border": "fg:#565f89"},
				},
			}},
		},
	}
	event := &pb.Event{Event: &pb.Event_Resize{Resize: &pb.ResizeEvent{Cols: 80, Rows: 24}}}
	result := &pb.Result{
		RequestId: 42,
		Epoch:     3,
		Method:    "terminal.create",
		Params:    &pb.MethodParams{Endpoint: "local", Argv: []string{"zsh"}, Ephemeral: gproto.Bool(true)},
	}
	response := &pb.Response{
		RequestId: 42,
		Epoch:     3,
		Ok:        true,
		Data:      &pb.MethodData{Endpoint: "local", Id: "terminal:local:main"},
	}

	tests := []struct {
		name string
		from Role
		to   Role
		typ  Type
		msg  gproto.Message
	}{
		{"hello", RoleHost, RoleProgram, TypeHello, hello},
		{"view", RoleProgram, RoleHost, TypeView, view},
		{"event", RoleHost, RoleProgram, TypeEvent, event},
		{"result", RoleProgram, RoleHost, TypeResult, result},
		{"response", RoleHost, RoleProgram, TypeResponse, response},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := NewEncoder(&buf, tc.from, 0).Encode(tc.typ, tc.msg); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			wire := buf.Bytes()
			total := binary.BigEndian.Uint32(wire[:4])
			if int(total)+4 != len(wire) {
				t.Fatalf("length prefix = %d, want %d", total, len(wire)-4)
			}
			if Type(wire[4]) != tc.typ {
				t.Fatalf("type byte = %d, want %d", wire[4], tc.typ)
			}
			gotType, gotMsg, err := NewDecoder(&buf, tc.to, 0).DecodeMessage()
			if err != nil {
				t.Fatalf("DecodeMessage: %v", err)
			}
			if gotType != tc.typ {
				t.Fatalf("decoded type = %v, want %v", gotType, tc.typ)
			}
			if !gproto.Equal(gotMsg, tc.msg) {
				t.Fatalf("round trip mismatch:\n got %v\nwant %v", gotMsg, tc.msg)
			}
		})
	}
}

func frameBytes(typ Type, payload []byte) []byte {
	out := make([]byte, 5+len(payload))
	binary.BigEndian.PutUint32(out[:4], uint32(1+len(payload)))
	out[4] = byte(typ)
	copy(out[5:], payload)
	return out
}

func TestZeroLengthFrame(t *testing.T) {
	_, _, err := DecodeFrame([]byte{0, 0, 0, 0}, RoleHost, 0)
	if !IsKind(err, KindZeroLength) {
		t.Fatalf("err = %v, want KindZeroLength", err)
	}
}

func TestOversizeRejectedBeforeDecode(t *testing.T) {
	payload := bytes.Repeat([]byte{0x08}, 64)
	frame := frameBytes(TypeResult, payload)
	_, _, err := DecodeFrame(frame, RoleHost, 32)
	if !IsKind(err, KindOversize) {
		t.Fatalf("err = %v, want KindOversize", err)
	}
	var fe *Error
	if !errors.As(err, &fe) {
		t.Fatalf("err is not *Error: %v", err)
	}
	if fe.Type != TypeResult {
		t.Fatalf("classified type = %v, want RESULT", fe.Type)
	}
	if fe.Limit != 32 {
		t.Fatalf("limit = %d, want 32", fe.Limit)
	}
}

func TestOversizeBeatsDirection(t *testing.T) {
	// A huge HELLO would be a direction error for a program-role decoder,
	// but the oversize verdict must come first (PROTOCOL §0).
	frame := frameBytes(TypeHello, bytes.Repeat([]byte{0x08}, 64))
	_, _, err := DecodeFrame(frame, RoleProgram, 32)
	if !IsKind(err, KindOversize) {
		t.Fatalf("err = %v, want KindOversize", err)
	}
}

func TestHalfPacketNeverPanics(t *testing.T) {
	frame := frameBytes(TypeView, []byte{0x08, 0x01})
	for cut := 0; cut < len(frame); cut++ {
		_, _, err := DecodeFrame(frame[:cut], RoleHost, 0)
		if err == nil {
			t.Fatalf("cut %d: expected error", cut)
		}
		if cut == 0 {
			if !errors.Is(err, io.EOF) {
				t.Fatalf("cut 0: err = %v, want io.EOF", err)
			}
			continue
		}
		if !IsKind(err, KindTruncated) {
			t.Fatalf("cut %d: err = %v, want KindTruncated", cut, err)
		}
	}
	if _, _, err := DecodeFrame(frame, RoleHost, 0); err != nil {
		t.Fatalf("complete frame: %v", err)
	}
}

func TestTruncatedOversizeDrain(t *testing.T) {
	frame := frameBytes(TypeView, bytes.Repeat([]byte{0x08}, 64))
	_, _, err := DecodeFrame(frame[:10], RoleHost, 8)
	if !IsKind(err, KindTruncated) {
		t.Fatalf("err = %v, want KindTruncated", err)
	}
}

func TestUnknownType(t *testing.T) {
	frame := frameBytes(Type(9), nil)
	_, _, err := DecodeFrame(frame, RoleHost, 0)
	if !IsKind(err, KindUnknownType) {
		t.Fatalf("err = %v, want KindUnknownType", err)
	}
}

func TestDirectionValidation(t *testing.T) {
	tests := []struct {
		name string
		role Role
		typ  Type
		ok   bool
	}{
		{"host receives view", RoleHost, TypeView, true},
		{"host receives result", RoleHost, TypeResult, true},
		{"host receives hello", RoleHost, TypeHello, false},
		{"host receives event", RoleHost, TypeEvent, false},
		{"host receives response", RoleHost, TypeResponse, false},
		{"program receives hello", RoleProgram, TypeHello, true},
		{"program receives event", RoleProgram, TypeEvent, true},
		{"program receives response", RoleProgram, TypeResponse, true},
		{"program receives view", RoleProgram, TypeView, false},
		{"program receives result", RoleProgram, TypeResult, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			frame := frameBytes(tc.typ, nil)
			_, _, err := DecodeFrame(frame, tc.role, 0)
			if tc.ok && err != nil {
				t.Fatalf("DecodeFrame: %v", err)
			}
			if !tc.ok && !IsKind(err, KindDirection) {
				t.Fatalf("err = %v, want KindDirection", err)
			}
		})
	}
}

func TestEncodeDirectionValidation(t *testing.T) {
	var buf bytes.Buffer
	if err := NewEncoder(&buf, RoleHost, 0).Encode(TypeView, &pb.View{}); !IsKind(err, KindDirection) {
		t.Fatalf("host sent VIEW: err = %v, want KindDirection", err)
	}
	if err := NewEncoder(&buf, RoleProgram, 0).Encode(TypeHello, &pb.Hello{}); !IsKind(err, KindDirection) {
		t.Fatalf("program sent HELLO: err = %v, want KindDirection", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("illegal frames wrote %d bytes", buf.Len())
	}
}

func TestEncodeOversize(t *testing.T) {
	_, err := Marshal(TypeHello, &pb.Hello{ViewId: "0123456789"}, 4)
	if !IsKind(err, KindOversize) {
		t.Fatalf("err = %v, want KindOversize", err)
	}
}

func TestMarshalUnknownType(t *testing.T) {
	if _, err := Marshal(Type(0), &pb.View{}, 0); !IsKind(err, KindUnknownType) {
		t.Fatalf("err = %v, want KindUnknownType", err)
	}
}

func TestPayloadDecodeError(t *testing.T) {
	frame := frameBytes(TypeResponse, []byte{0xff, 0xff})
	_, _, err := DecodeFrame(frame, RoleProgram, 0)
	if err != nil {
		t.Fatalf("envelope decode: %v", err)
	}
	_, payload, _ := DecodeFrame(frame, RoleProgram, 0)
	if _, err := UnmarshalPayload(TypeResponse, payload); !IsKind(err, KindPayload) {
		t.Fatalf("err = %v, want KindPayload", err)
	}
}

func TestEmptyPayload(t *testing.T) {
	frame, err := Marshal(TypeResponse, nil, 0)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(frame) != 5 {
		t.Fatalf("empty frame length = %d, want 5", len(frame))
	}
	tip, payload, err := DecodeFrame(frame, RoleProgram, 0)
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if tip != TypeResponse || len(payload) != 0 {
		t.Fatalf("tip=%v payload=%d bytes, want RESPONSE/0", tip, len(payload))
	}
	msg, err := UnmarshalPayload(tip, payload)
	if err != nil {
		t.Fatalf("UnmarshalPayload: %v", err)
	}
	if !gproto.Equal(msg, &pb.Response{}) {
		t.Fatalf("payload = %v, want empty Response", msg)
	}
}

func TestStreamEOFBetweenFrames(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf, RoleHost, 0)
	for i := 0; i < 3; i++ {
		if err := enc.Encode(TypeEvent, &pb.Event{Event: &pb.Event_Resize{Resize: &pb.ResizeEvent{Cols: 1, Rows: 1}}}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}
	dec := NewDecoder(&buf, RoleProgram, 0)
	for i := 0; i < 3; i++ {
		if _, _, err := dec.Decode(); err != nil {
			t.Fatalf("Decode %d: %v", i, err)
		}
	}
	if _, _, err := dec.Decode(); !errors.Is(err, io.EOF) {
		t.Fatalf("final Decode err = %v, want io.EOF", err)
	}
}

func TestDefaultLimitApplied(t *testing.T) {
	big := bytes.Repeat([]byte{0x08}, int(DefaultMaxMessageBytes)+1)
	frame := frameBytes(TypeResult, big)
	if _, _, err := DecodeFrame(frame, RoleHost, 0); !IsKind(err, KindOversize) {
		t.Fatalf("err = %v, want KindOversize", err)
	}
}
