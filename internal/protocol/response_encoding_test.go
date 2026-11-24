package protocol

import (
	"bytes"
	"compress/flate"
	"testing"
)

func TestResponsePayloadRoundTripsCompressedBody(t *testing.T) {
	body := bytes.Repeat([]byte("screen-cell-payload;"), 400)
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	payload, err := EncodeResponsePayload(Response{
		ID:       7,
		Result:   compressed.Bytes(),
		Encoding: ResponseEncodingDeflate,
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeResponsePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ID != 7 || decoded.Encoding != ResponseEncodingDeflate {
		t.Fatalf("decoded response = %#v", decoded)
	}
	inflated, err := inflateApplicationResult(decoded.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inflated, body) {
		t.Fatal("inflated body does not match the original")
	}
}

func TestResponsePayloadDefaultsToRaw(t *testing.T) {
	payload, err := EncodeResponsePayload(Response{ID: 1, Result: []byte("plain")})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeResponsePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Encoding != ResponseEncodingRaw {
		t.Fatalf("default encoding = %v, want raw", decoded.Encoding)
	}
}

func TestInflateRejectsOversizedPayload(t *testing.T) {
	// 压缩一段超过单帧上限的数据，解压时必须被限额拒绝。
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	chunk := make([]byte, 1<<20)
	for index := 0; index < 6; index += 1 {
		if _, err := writer.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := inflateApplicationResult(compressed.Bytes()); err == nil {
		t.Fatal("inflate should reject payloads above the frame limit")
	}
}

func TestHelloCarriesCapabilities(t *testing.T) {
	payload, err := EncodeHelloPayload(Hello{
		Version:      7,
		Client:       "mobile",
		Capabilities: []string{ScreenCompressionCapability},
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeHelloPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Capabilities) != 1 || decoded.Capabilities[0] != ScreenCompressionCapability {
		t.Fatalf("capabilities = %#v", decoded.Capabilities)
	}
}
