package wire

import (
	"bytes"
	"errors"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.WriteFrame(3, TypeScreenUpdate, []byte("hello")); err != nil {
		t.Fatalf("write frame failed: %v", err)
	}

	dec := NewDecoder(&buf)
	ch, typ, payload, err := dec.ReadFrame()
	if err != nil {
		t.Fatalf("read frame failed: %v", err)
	}
	if ch != 3 || typ != TypeScreenUpdate || string(payload) != "hello" {
		t.Fatalf("unexpected frame: %d %d %q", ch, typ, string(payload))
	}
}

func TestMaxEncodedFrameSizeIncludesWireHeader(t *testing.T) {
	if MaxEncodedFrameSize != MaxFrameSize+7 {
		t.Fatalf("max encoded frame size = %d want=%d", MaxEncodedFrameSize, MaxFrameSize+7)
	}
}

func TestFrameRejectsOversizePayload(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.WriteFrame(1, TypeScreenUpdate, make([]byte, MaxFrameSize+1))
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
}

func TestDecodeFrameRejectsMalformedLength(t *testing.T) {
	frame := []byte{
		0x00, 0x03,
		TypeScreenUpdate,
		0x00, 0x00, 0x00, 0x03,
		'o', 'k',
	}

	if _, _, _, err := DecodeFrame(frame); err == nil {
		t.Fatal("expected malformed frame length error")
	}
}

func TestPayloadHelpersRoundTrip(t *testing.T) {
	cols, rows, err := DecodeResizePayload(EncodeResizePayload(120, 50))
	if err != nil {
		t.Fatalf("decode resize payload failed: %v", err)
	}
	if cols != 120 || rows != 50 {
		t.Fatalf("unexpected resize payload roundtrip: %dx%d", cols, rows)
	}

	dropped, err := DecodeSyncLostPayload(EncodeSyncLostPayload(42))
	if err != nil {
		t.Fatalf("decode sync-lost payload failed: %v", err)
	}
	if dropped != 42 {
		t.Fatalf("unexpected dropped bytes: %d", dropped)
	}

	readySeq, err := DecodeStreamReadyPayload(EncodeStreamReadyPayload(99))
	if err != nil {
		t.Fatalf("decode stream-ready payload failed: %v", err)
	}
	if readySeq != 99 {
		t.Fatalf("unexpected stream-ready sequence: %d", readySeq)
	}

	code, err := DecodeClosedPayload(EncodeClosedPayload(-1))
	if err != nil {
		t.Fatalf("decode closed payload failed: %v", err)
	}
	if code != -1 {
		t.Fatalf("unexpected closed code: %d", code)
	}

	if _, _, err := DecodeResizePayload([]byte{1, 2, 3}); !errors.Is(err, ErrShortPayload) {
		t.Fatalf("expected ErrShortPayload for resize payload, got %v", err)
	}
	if _, err := DecodeSyncLostPayload([]byte{1, 2, 3}); !errors.Is(err, ErrShortPayload) {
		t.Fatalf("expected ErrShortPayload for sync-lost payload, got %v", err)
	}
	if _, err := DecodeStreamReadyPayload([]byte{1, 2, 3}); !errors.Is(err, ErrShortPayload) {
		t.Fatalf("expected ErrShortPayload for stream-ready payload, got %v", err)
	}
	if _, err := DecodeClosedPayload([]byte{1, 2, 3}); !errors.Is(err, ErrShortPayload) {
		t.Fatalf("expected ErrShortPayload for closed payload, got %v", err)
	}
}
