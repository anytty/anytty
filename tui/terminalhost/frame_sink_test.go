package terminalhost

import (
	"bytes"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/render"
)

func TestFrameSinkInitialFrameClearsScreen(t *testing.T) {
	var out bytes.Buffer
	sink := NewFrameSink(&out)
	if err := sink.WriteFrame(testFrameSinkFrame([]string{"aaa", "bbb"}, 3, 2)); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	payload := out.String()
	if !strings.Contains(payload, clearScreen) {
		t.Fatalf("initial frame should clear screen, got %q", payload)
	}
}

func TestFrameSinkSameSizeFrameUsesGlobalRowPresenterWithoutClearingRows(t *testing.T) {
	var out bytes.Buffer
	sink := NewFrameSink(&out)
	first := testFrameSinkFrame([]string{"header", "live-1", "live-2", "footer"}, 12, 4)
	if err := sink.WriteFrame(first); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	out.Reset()

	next := testFrameSinkFrame([]string{"header", "copy-1", "copy-2", "footer"}, 12, 4)
	if err := sink.WriteFrame(next); err != nil {
		t.Fatalf("write next frame: %v", err)
	}
	payload := out.String()
	if strings.Contains(payload, clearScreen) {
		t.Fatalf("same-size frame must not clear full screen, got %q", payload)
	}
	if strings.Contains(payload, clearLine) {
		t.Fatalf("same-size row presenter must not clear every changed row, got %q", payload)
	}
	if !strings.Contains(payload, cursorPosition(2, 1)+"copy-1") || !strings.Contains(payload, cursorPosition(3, 1)+"copy-2") {
		t.Fatalf("same-size presenter should write only changed rows, got %q", payload)
	}
}

func TestFrameSinkConsecutiveCompleteLiveFramesDoNotClearScreen(t *testing.T) {
	var out bytes.Buffer
	sink := NewFrameSink(&out)
	first := testFrameSinkFrame([]string{"header", "live revision 1", "footer"}, 20, 3)
	if err := sink.WriteFrame(first); err != nil {
		t.Fatalf("write first live frame: %v", err)
	}
	out.Reset()

	next := testFrameSinkFrame([]string{"header", "live revision 2", "footer"}, 20, 3)
	if next.Patch != nil {
		t.Fatalf("test requires a complete logical frame, got patch %#v", next.Patch)
	}
	if err := sink.WriteFrame(next); err != nil {
		t.Fatalf("write next live frame: %v", err)
	}
	if payload := out.String(); strings.Contains(payload, clearScreen) {
		t.Fatalf("same-size complete live frame must use row diff, got %q", payload)
	}
}

func TestFrameSinkShrinkingRowPadsStaleTailWithoutClearLine(t *testing.T) {
	var out bytes.Buffer
	sink := NewFrameSink(&out)
	if err := sink.WriteFrame(testFrameSinkFrame([]string{"abcdef"}, 6, 1)); err != nil {
		t.Fatalf("write first frame: %v", err)
	}
	out.Reset()

	if err := sink.WriteFrame(testFrameSinkFrame([]string{"xy"}, 6, 1)); err != nil {
		t.Fatalf("write next frame: %v", err)
	}
	payload := out.String()
	if strings.Contains(payload, clearLine) {
		t.Fatalf("shrinking row should be padded, not clear-line rewritten, got %q", payload)
	}
	if !strings.Contains(payload, "xy"+render.ANSIReset+"    ") {
		t.Fatalf("shrinking row should pad stale tail, got %q", payload)
	}
}

func testFrameSinkFrame(lines []string, width int, height int) render.Frame {
	return render.Frame{
		ANSILines: lines,
		Metadata:  render.RenderMetadata{Width: width, Height: height},
	}
}
