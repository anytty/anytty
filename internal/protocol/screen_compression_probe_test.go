package protocol

import (
	"bytes"
	"compress/flate"
	"fmt"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
	"google.golang.org/protobuf/proto"
)

// buildTUIScreen 构造一个 opencode 风格的全屏 TUI：每行带边框、多段着色 run。
// 内容按行列变化，避免「同一个字符串重复」把压缩比拉到不真实的高度。
var probeWords = []string{
	"func", "return", "err", "context", "session", "request", "timeout",
	"terminal", "screen", "revision", "buffer", "channel", "payload",
	"connect", "disconnect", "retry", "flush", "batch", "insert", "update",
}

func buildTUIScreen(cols, rows, runsPerRow int) *apipb.NativeScreenResult {
	rowsProto := make([]*apipb.ScreenRow, 0, rows)
	for row := 0; row < rows; row++ {
		cells := make([]*apipb.ScreenCell, 0, runsPerRow)
		for run := 0; run < runsPerRow; run++ {
			content := fmt.Sprintf(
				"│ %02d:%02d %s %s %d  ",
				row, run,
				probeWords[(row*3+run)%len(probeWords)],
				probeWords[(row+run*7)%len(probeWords)],
				(row*31+run*17)%997,
			)
			cells = append(cells, &apipb.ScreenCell{
				Content: content,
				Width:   int32(len(content)),
				Style: &apipb.CellStyle{
					Foreground: []string{"#c0caf5", "#9ece6a", "#e0af68", "#7aa2f7"}[(row+run)%4],
					Background: "#1a1b26",
					Bold:       run%3 == 0,
				},
			})
		}
		rowsProto = append(rowsProto, &apipb.ScreenRow{Cells: cells})
	}
	return &apipb.NativeScreenResult{
		Terminal:     &apipb.TerminalRef{EndpointId: "local", TerminalId: "term"},
		LiveRevision: 42,
		Size:         &apipb.TerminalSize{Cols: uint32(cols), Rows: uint32(rows)},
		RowReplacements: func() []*apipb.ScreenRowReplace {
			out := make([]*apipb.ScreenRowReplace, 0, rows)
			for index, row := range rowsProto {
				out = append(out, &apipb.ScreenRowReplace{RowIndex: int32(index), Row: row})
			}
			return out
		}(),
		FullReplace: true,
	}
}

func deflateBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer, err := flate.NewWriter(&buffer, flate.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestScreenFrameCompressionProbe(t *testing.T) {
	for _, shape := range []struct {
		cols, rows, runs int
	}{
		{80, 24, 8},
		{120, 40, 12},
		{160, 48, 16},
	} {
		envelope := &apipb.ResultEnvelope{
			Result: &apipb.ResultEnvelope_LiveScreen{
				LiveScreen: buildTUIScreen(shape.cols, shape.rows, shape.runs),
			},
		}
		raw, err := proto.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		compressed := deflateBytes(t, raw)
		ratio := float64(len(raw)) / float64(len(compressed))
		t.Logf(
			"%dx%d runs=%d raw=%d deflate=%d ratio=%.1fx",
			shape.cols, shape.rows, shape.runs, len(raw), len(compressed), ratio,
		)
		if ratio < 3 {
			t.Fatalf(
				"%dx%d compression ratio %.1fx is below the 3x design floor",
				shape.cols, shape.rows, ratio,
			)
		}
		if shape.cols == 160 && len(compressed) > 32<<10 {
			t.Fatalf("160x48 deflated frame is %d bytes, above the 32KB cap", len(compressed))
		}
	}
}
