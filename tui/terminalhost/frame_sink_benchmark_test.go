package terminalhost

import (
	"io"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/render"
)

func BenchmarkFrameSinkRepeatedFrame(b *testing.B) {
	frame := benchmarkFrame(180, 60)
	sink := NewFrameSink(io.Discard)
	if err := sink.WriteFrame(frame); err != nil {
		b.Fatalf("write first frame: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sink.WriteFrame(frame); err != nil {
			b.Fatalf("write frame: %v", err)
		}
	}
}

func BenchmarkFrameSinkSingleRowChange(b *testing.B) {
	frame := benchmarkFrame(180, 60)
	sink := NewFrameSink(io.Discard)
	if err := sink.WriteFrame(frame); err != nil {
		b.Fatalf("write first frame: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame.ANSILines[len(frame.ANSILines)/2] = strings.Repeat(string(rune('a'+(i%26))), 180)
		if err := sink.WriteFrame(frame); err != nil {
			b.Fatalf("write frame: %v", err)
		}
	}
}

func benchmarkFrame(width int, height int) render.Frame {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = strings.Repeat("x", width)
	}
	return render.Frame{ANSILines: lines, Metadata: render.RenderMetadata{Width: width, Height: height}}
}
