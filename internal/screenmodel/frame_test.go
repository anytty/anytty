package screenmodel

import (
	"bytes"
	"testing"
)

// TestFrameRoundTrip 覆盖 TypeScreen 帧的 raw/zstd 两种编码。
func TestFrameRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("row-0123456789abcdef"), 200)
	encoded := EncodeFrame(payload)
	if encoded[0] != frameEncodingZstd {
		t.Fatalf("encoding flag = %d, want zstd for repetitive payload", encoded[0])
	}
	decoded, err := DecodeFrame(encoded)
	if err != nil {
		t.Fatalf("decode zstd frame: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("zstd screen frame round trip mismatch")
	}

	tiny := []byte{1, 2, 3}
	rawEncoded := EncodeFrame(tiny)
	if rawEncoded[0] != frameEncodingRaw {
		t.Fatalf("tiny payload flag = %d, want raw", rawEncoded[0])
	}
	if got, err := DecodeFrame(rawEncoded); err != nil || !bytes.Equal(got, tiny) {
		t.Fatalf("raw round trip = %v err=%v", got, err)
	}
	if _, err := DecodeFrame(nil); err == nil {
		t.Fatal("empty frame must fail")
	}
}
