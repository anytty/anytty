package protocol

import (
	"bytes"
	"compress/flate"
	"testing"
)

// TestZstdApplicationResultRoundTrip 验证 zstd 编码/解码与 deflate 的互操作：
// 两种编码都能还原同一份 application result，且 zstd 明显更小。
func TestZstdApplicationResultRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("screen-row-0123456789abcdef"), 400)
	compressed, ok := CompressZstdApplicationResult(payload)
	if !ok {
		t.Fatal("zstd compression unavailable")
	}
	decoded, err := inflateZstdApplicationResult(compressed)
	if err != nil {
		t.Fatalf("inflate zstd: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("zstd round trip mismatch")
	}
	deflated := bytes.Buffer{}
	writer, err := flate.NewWriter(&deflated, flate.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if len(compressed) >= deflated.Len() {
		t.Fatalf("zstd = %d bytes, want smaller than deflate %d", len(compressed), deflated.Len())
	}
}

// TestResponseEncodingZstdEnvelope 验证 zstd 编码位在 envelope 编解码里保持。
func TestResponseEncodingZstdEnvelope(t *testing.T) {
	encoded, err := EncodeResponsePayload(Response{ID: 7, Result: []byte("x"), Encoding: ResponseEncodingZstd})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeResponsePayload(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Encoding != ResponseEncodingZstd || decoded.ID != 7 || !bytes.Equal(decoded.Result, []byte("x")) {
		t.Fatalf("decoded = %#v", decoded)
	}
}

// TestScreenStreamPayloadRoundTrip 覆盖 TypeScreen 帧的 raw/zstd 两种编码。
func TestScreenStreamPayloadRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("row-0123456789abcdef"), 200)
	encoded := EncodeScreenStreamPayload(payload)
	if encoded[0] != screenStreamEncodingZstd {
		t.Fatalf("encoding flag = %d, want zstd for repetitive payload", encoded[0])
	}
	decoded, err := DecodeScreenStreamPayload(encoded)
	if err != nil {
		t.Fatalf("decode zstd frame: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("zstd screen stream round trip mismatch")
	}

	tiny := []byte{1, 2, 3}
	rawEncoded := EncodeScreenStreamPayload(tiny)
	if rawEncoded[0] != screenStreamEncodingRaw {
		t.Fatalf("tiny payload flag = %d, want raw", rawEncoded[0])
	}
	if got, err := DecodeScreenStreamPayload(rawEncoded); err != nil || !bytes.Equal(got, tiny) {
		t.Fatalf("raw round trip = %v err=%v", got, err)
	}
	if _, err := DecodeScreenStreamPayload(nil); err == nil {
		t.Fatal("empty frame must fail")
	}
}
