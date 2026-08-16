package wire

import "testing"

func BenchmarkDecodeFrame(b *testing.B) {
	frame, err := EncodeFrame(7, TypeScreenUpdate, []byte("payload"))
	if err != nil {
		b.Fatalf("encode frame failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		channel, typ, payload, err := DecodeFrame(frame)
		if err != nil {
			b.Fatalf("decode frame failed: %v", err)
		}
		if channel != 7 || typ != TypeScreenUpdate || len(payload) != len("payload") {
			b.Fatalf("unexpected decode result: %d %d %d", channel, typ, len(payload))
		}
	}
}
