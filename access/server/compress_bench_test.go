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
	// 线上实际使用的是 zstd(无字典)：比逐帧 deflate 小约 1.5×，服务端 CPU
	// 低一个数量级（见 Benchmark*）。上一帧字典能压到 151B，但 klauspost 的
	// encoder 必须绑定字典，逐帧重建要 448µs/7MB，不划算。
	if len(noDict) >= len(plain) {
		t.Fatalf("zstd = %d bytes, want smaller than flate %d", len(noDict), len(plain))
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

func BenchmarkZstdDictCreatePerFrame(b *testing.B) {
	previous := benchScreen(1)
	payload := benchScreen(2)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		encoder, err := zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.SpeedFastest),
			zstd.WithEncoderDictRaw(1, previous),
		)
		if err != nil {
			b.Fatal(err)
		}
		_ = encoder.EncodeAll(payload, nil)
		encoder.Close()
	}
}

func flateDictCompress(dict, payload []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := flate.NewWriterDict(&buffer, flate.BestSpeed, dict)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(payload); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// TestFlateDictRatio 记录一个反直觉结果：Go 标准库 flate 的 preset
// dictionary 对此类 protobuf delta 没有收益（645B -> 645B），所以方案一
// 最终选 zstd（无字典）。
func TestFlateDictRatio(t *testing.T) {
	previous := benchScreen(1)
	payload := benchScreen(2)
	plain := flateCompress(payload)
	dictCompressed, err := flateDictCompress(previous, payload)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("payload=%d flate=%d flate+dict=%d", len(payload), len(plain), len(dictCompressed))
}

func BenchmarkFlateDictBestSpeed(b *testing.B) {
	previous := benchScreen(1)
	payload := benchScreen(2)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := flateDictCompress(previous, payload); err != nil {
			b.Fatal(err)
		}
	}
}
