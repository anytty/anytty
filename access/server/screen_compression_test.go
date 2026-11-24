package server

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestCompressLiveScreenResultPrefersCompression(t *testing.T) {
	body := bytes.Repeat([]byte("row-cells-with-styles;"), 512)
	compressed, ok := compressLiveScreenResult(body)
	if !ok {
		t.Fatal("large repetitive payload should compress")
	}
	if len(compressed) >= len(body) {
		t.Fatalf("compressed=%d raw=%d", len(compressed), len(body))
	}
	reader := flate.NewReader(bytes.NewReader(compressed))
	round, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(round, body) {
		t.Fatal("round trip mismatch")
	}
}

func TestCompressLiveScreenResultSkipsSmallFrames(t *testing.T) {
	if _, ok := compressLiveScreenResult([]byte("ack")); ok {
		t.Fatal("tiny frames must stay raw")
	}
}

func TestCompressLiveScreenResultSkipsIncompressibleFrames(t *testing.T) {
	// 随机字节压缩后会更大，必须保持明文。
	incompressible := make([]byte, 4096)
	state := uint32(2166136261)
	for index := range incompressible {
		state = state*1664525 + 1013904223
		incompressible[index] = byte(state >> 24)
	}
	if _, ok := compressLiveScreenResult(incompressible); ok {
		t.Fatal("incompressible payload must stay raw")
	}
}
