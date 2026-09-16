// anytty-osc-repro checks that repeated Unicode window titles stay out of the
// terminal grid. Run with: go run ./cmd/anytty-osc-repro
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anytty/anytty/core/live"
)

func main() {
	failed := false
	for _, alt := range []bool{false, true} {
		for _, chunkSize := range []int{1, 7, 65536} {
			if !check(alt, chunkSize) {
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

func check(alt bool, chunkSize int) bool {
	surface := live.NewSurfaceTrackWithOptions(live.SurfaceSize{Cols: 80, Rows: 8}, live.SurfaceTrackOptions{
		CaptureLineHistory: true,
	})
	const body = "正文：窗口标题不应出现在这里。"
	var raw strings.Builder
	if alt {
		raw.WriteString("\x1b[?1049h")
	}
	raw.WriteString("\x1b[2J\x1b[H" + body + "\r\n")
	for range 12 {
		// U+2713 encodes as E2 9C 93. Its continuation byte 9C is not ST.
		raw.WriteString("\x1b]0;✓ APP历史模式下拉跳转原因 | anytty\x07")
	}
	data := raw.String()
	for start := 0; start < len(data); start += chunkSize {
		surface.WriteWithResult(data[start:min(start+chunkSize, len(data))])
	}
	got := strings.TrimSpace(strings.Join(surface.Rows(), "\n"))
	if got != body {
		fmt.Printf("FAIL alt=%v chunk=%d: title leaked into the screen\n%s\n", alt, chunkSize, got)
		return false
	}
	fmt.Printf("PASS alt=%v chunk=%d: only the body is visible\n", alt, chunkSize)
	return true
}
