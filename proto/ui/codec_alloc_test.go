package proto

import (
	"bytes"
	"io"
	"testing"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// frameRepeater replays one fixed frame forever without allocating.
type frameRepeater struct {
	frame []byte
	pos   int
}

func (r *frameRepeater) Read(p []byte) (int, error) {
	if r.pos == len(r.frame) {
		r.pos = 0
	}
	n := copy(p, r.frame[r.pos:])
	r.pos += n
	return n, nil
}

func TestMarshalAppendMatchesMarshal(t *testing.T) {
	cases := []struct {
		name string
		typ  Type
		msg  gproto.Message
	}{
		{"view", TypeView, &pb.View{
			Epoch: 3, Rev: 7,
			Keys: &pb.Keys{Claim: []string{"ctrl-p"}, All: false},
			Root: &pb.Box{Id: "root", Flow: "col", Children: []*pb.Box{{Id: "a"}, {Id: "b"}}},
		}},
		{"result", TypeResult, &pb.Result{RequestId: 42, Epoch: 1, Method: "terminal.scroll", Params: &pb.MethodParams{Id: "t", Delta: 2}}},
		{"stream", TypeStream, &pb.StreamFrame{StreamId: 7, Kind: "data", Payload: []byte{1, 2, 3}}},
		{"empty response", TypeResponse, &pb.Response{}},
		{"nil event", TypeEvent, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, err := Marshal(tc.typ, tc.msg, 0)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got, err := MarshalAppend(nil, tc.typ, tc.msg, 0)
			if err != nil {
				t.Fatalf("MarshalAppend: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("MarshalAppend = %x, Marshal = %x", got, want)
			}
		})
	}
}

func TestMarshalAppendPrefixAndErrors(t *testing.T) {
	prefix := []byte("keep-me")
	frame, err := MarshalAppend(append([]byte(nil), prefix...), TypeResponse, &pb.Response{RequestId: 1}, 0)
	if err != nil {
		t.Fatalf("MarshalAppend: %v", err)
	}
	if !bytes.HasPrefix(frame, prefix) {
		t.Fatalf("prefix lost: %x", frame[:len(prefix)])
	}
	if typ, _, err := DecodeFrame(frame[len(prefix):], RoleProgram, 0); err != nil || typ != TypeResponse {
		t.Fatalf("appended frame decode = %v/%v", typ, err)
	}

	// Error kinds mirror Marshal and leave dst untouched.
	dst := []byte("base")
	if _, err := MarshalAppend(dst, Type(9), &pb.View{}, 0); !IsKind(err, KindUnknownType) {
		t.Fatalf("unknown type = %v, want KindUnknownType", err)
	}
	if _, err := MarshalAppend(dst, TypeHello, &pb.Hello{ViewId: "0123456789"}, 4); !IsKind(err, KindOversize) {
		t.Fatalf("oversize = %v, want KindOversize", err)
	}
	bad := &pb.Hello{ViewId: string([]byte{0xff, 0xfe})}
	if _, err := MarshalAppend(dst, TypeHello, bad, 0); !IsKind(err, KindPayload) {
		t.Fatalf("payload error = %v, want KindPayload", err)
	}
	if !bytes.Equal(dst, []byte("base")) {
		t.Fatalf("dst mutated on error: %q", dst)
	}
}

// countingWriter records how many Write calls the Encoder makes.
type countingWriter struct {
	writes int
	bytes  int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.writes++
	w.bytes += len(p)
	return len(p), nil
}

func TestEncoderWritesOnce(t *testing.T) {
	var w countingWriter
	enc := NewEncoder(&w, RoleProgram, 0)
	msg := &pb.Result{RequestId: 1, Epoch: 1, Method: "x"}
	for i := 0; i < 3; i++ {
		if err := enc.Encode(TypeResult, msg); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	if w.writes != 3 {
		t.Fatalf("writes = %d, want one per frame", w.writes)
	}
}

// TestEncoderEncodeZeroAlloc pins the pooled-encoder fast path: after the
// pool is warm, Encode must not allocate in steady state.
func TestEncoderEncodeZeroAlloc(t *testing.T) {
	enc := NewEncoder(io.Discard, RoleProgram, 0)
	msg := &pb.Result{RequestId: 42, Epoch: 1, Method: "terminal.scroll", Params: &pb.MethodParams{Id: "main", Delta: 1}}
	if err := enc.Encode(TypeResult, msg); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		if err := enc.Encode(TypeResult, msg); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	if allocs != 0 {
		t.Fatalf("pooled Encode allocs = %v, want 0", allocs)
	}
}

// TestEncoderEncodeOldPathStillWorks keeps the documented error kinds of
// Encode intact (direction, oversize) while the scratch is pooled.
func TestEncoderEncodeOldPathStillWorks(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf, RoleProgram, 0)
	if err := enc.Encode(TypeHello, &pb.Hello{}); !IsKind(err, KindDirection) {
		t.Fatalf("direction = %v, want KindDirection", err)
	}
	if err := enc.Encode(TypeResult, &pb.Result{Method: string(bytes.Repeat([]byte("x"), int(DefaultMaxMessageBytes)))}); !IsKind(err, KindOversize) {
		t.Fatalf("oversize = %v, want KindOversize", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("illegal frames wrote %d bytes", buf.Len())
	}
}

// TestDecodeReuseBufferZeroAlloc pins the decoder reuse mode: after the
// scratch is sized, Decode must not allocate.
func TestDecodeReuseBufferZeroAlloc(t *testing.T) {
	frame, err := Marshal(TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "ev-1", Key: "ctrl-p"}}}, 0)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dec := NewDecoder(&frameRepeater{frame: frame}, RoleProgram, 0).ReuseBuffer(true)
	if typ, _, err := dec.Decode(); err != nil || typ != TypeEvent {
		t.Fatalf("warmup = %v/%v", typ, err)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		typ, payload, err := dec.Decode()
		if err != nil || typ != TypeEvent || len(payload) == 0 {
			t.Errorf("decode = %v/%d/%v", typ, len(payload), err)
		}
	})
	if allocs != 0 {
		t.Fatalf("ReuseBuffer Decode allocs = %v, want 0", allocs)
	}
}

// TestDecodeDefaultStillAllocates pins the unchanged default: NewDecoder
// keeps allocating one payload buffer per frame.
func TestDecodeDefaultStillAllocates(t *testing.T) {
	frame, err := Marshal(TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "ev-1", Key: "ctrl-p"}}}, 0)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dec := NewDecoder(&frameRepeater{frame: frame}, RoleProgram, 0)
	allocs := testing.AllocsPerRun(100, func() {
		if _, _, err := dec.Decode(); err != nil {
			t.Errorf("decode: %v", err)
		}
	})
	if allocs == 0 {
		t.Fatalf("default Decode allocs = 0, want the historical per-frame allocation")
	}
}

// TestDecodeReuseBufferAliasesScratch documents the reuse contract: payloads
// share one scratch buffer and stay valid only until the next Decode.
func TestDecodeReuseBufferAliasesScratch(t *testing.T) {
	frame, err := Marshal(TypeEvent, &pb.Event{Event: &pb.Event_Key{Key: &pb.KeyEvent{Id: "ev-1", Key: "ctrl-p"}}}, 0)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dec := NewDecoder(&frameRepeater{frame: frame}, RoleProgram, 0).ReuseBuffer(true)
	_, first, err := dec.Decode()
	if err != nil {
		t.Fatalf("first decode: %v", err)
	}
	_, second, err := dec.Decode()
	if err != nil {
		t.Fatalf("second decode: %v", err)
	}
	if len(first) == 0 || len(second) == 0 || &first[0] != &second[0] {
		t.Fatalf("payloads must alias the decoder scratch buffer")
	}

	// DecodeMessage decodes a copy into a fresh message before the next call.
	dec2 := NewDecoder(&frameRepeater{frame: frame}, RoleProgram, 0).ReuseBuffer(true)
	_, msg, err := dec2.DecodeMessage()
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}
	if msg.(*pb.Event).GetKey().GetKey() != "ctrl-p" {
		t.Fatalf("message = %v", msg)
	}
}

// TestDecodeReuseBufferErrors keeps the error classification identical in
// reuse mode.
func TestDecodeReuseBufferErrors(t *testing.T) {
	dec := NewDecoder(bytes.NewReader([]byte{0, 0, 0, 0}), RoleProgram, 0).ReuseBuffer(true)
	if _, _, err := dec.Decode(); !IsKind(err, KindZeroLength) {
		t.Fatalf("zero length = %v, want KindZeroLength", err)
	}

	truncated := NewDecoder(bytes.NewReader([]byte{0, 0, 0, 9, 3}), RoleProgram, 0).ReuseBuffer(true)
	if _, _, err := truncated.Decode(); !IsKind(err, KindTruncated) {
		t.Fatalf("truncated = %v, want KindTruncated", err)
	}

	oversize := NewDecoder(&frameRepeater{frame: frameBytes(TypeHello, []byte{1, 1})}, RoleProgram, 2).ReuseBuffer(true)
	if _, _, err := oversize.Decode(); !IsKind(err, KindOversize) {
		t.Fatalf("oversize = %v, want KindOversize", err)
	}
}
