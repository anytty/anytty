package server

import (
	"bytes"
	"compress/flate"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

// benchScreen 造一帧接近真实终端 delta 的 protobuf 载荷：56 行 × 180 cell。
func benchScreen(frame int) []byte {
	screen := &apipb.NativeScreenResult{
		LiveRevision: uint64(frame),
		Size:         &apipb.TerminalSize{Cols: 180, Rows: 60},
	}
	for row := 0; row < 56; row++ {
		line := make([]byte, 0, 180)
		for col := 0; col < 180; col++ {
			line = append(line, byte('a'+(row+col)%26))
		}
		if len(line) > 8 {
			line[8] = byte('0' + frame%10)
		}
		screen.RowReplacements = append(screen.RowReplacements, &apipb.ScreenRowReplace{
			RowIndex: int32(row),
			Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{
				Content: string(line), Width: 180, Style: &apipb.CellStyle{Foreground: "ansi:7"},
			}}},
		})
	}
	payload, err := proto.Marshal(screen)
	if err != nil {
		panic(err)
	}
	return payload
}

func flateCompress(payload []byte) []byte {
	var buffer bytes.Buffer
	writer, _ := flate.NewWriter(&buffer, flate.BestSpeed)
	_, _ = writer.Write(payload)
	_ = writer.Close()
	return buffer.Bytes()
}

func TestCompressionRatios(t *testing.T) {
	previous := benchScreen(1)
	payload := benchScreen(2)

	plain := flateCompress(payload)

	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithEncoderDictRaw(1, previous),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	withDict := encoder.EncodeAll(payload, nil)

	encoderNoDict, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		t.Fatal(err)
	}
	defer encoderNoDict.Close()
	noDict := encoderNoDict.EncodeAll(payload, nil)

	t.Logf("payload=%d flate=%d zstd=%d zstd+dict=%d", len(payload), len(plain), len(noDict), len(withDict))
	// 终端 delta 的行内容在相邻帧之间高度重复：用上一帧做字典应显著优于
	// 逐帧 deflate（实测 645B -> 151B）。
	if len(withDict) >= len(plain)/2 {
		t.Fatalf("zstd with previous-frame dictionary = %d bytes, want < half of flate %d", len(withDict), len(plain))
	}
}

func BenchmarkFlateBestSpeed(b *testing.B) {
	payload := benchScreen(2)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = flateCompress(payload)
	}
}

func BenchmarkZstdDictFastest(b *testing.B) {
	previous := benchScreen(1)
	payload := benchScreen(2)
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithEncoderDictRaw(1, previous),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer encoder.Close()
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = encoder.EncodeAll(payload, nil)
	}
}

func BenchmarkZstdNoDictFastest(b *testing.B) {
	payload := benchScreen(2)
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		b.Fatal(err)
	}
	defer encoder.Close()
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = encoder.EncodeAll(payload, nil)
	}
}
