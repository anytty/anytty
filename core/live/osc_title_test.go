package live

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRepeatedUnicodeTitlesLeaveLiveScreenUnchanged(t *testing.T) {
	for _, history := range []bool{false, true} {
		for _, alt := range []bool{false, true} {
			for _, chunkSize := range []int{1, 7, 65536} {
				t.Run(fmt.Sprintf("history=%v/alt=%v/chunk=%d", history, alt, chunkSize), func(t *testing.T) {
					surface := NewSurfaceTrackWithOptions(SurfaceSize{Cols: 80, Rows: 8}, SurfaceTrackOptions{CaptureLineHistory: history})
					defer surface.vt.Close()
					if alt {
						surface.WriteWithResult("\x1b[?1049h")
					}
					surface.WriteWithResult("\x1b[2J\x1b[H正文：窗口标题不应出现在这里。\r\n")
					before := surface.Snapshot()
					for range 12 {
						for _, title := range []string{"⠋ 正在处理APP历史模式下拉跳转原因 | anytty", "✓ APP历史模式下拉跳转原因 | anytty"} {
							raw := "\x1b]0;" + title + "\x07"
							for start := 0; start < len(raw); start += chunkSize {
								surface.WriteWithResult(raw[start:min(start+chunkSize, len(raw))])
							}
							if after := surface.Snapshot(); !reflect.DeepEqual(after, before) {
								t.Fatalf("title %q changed the live screen or cursor: rows=%q cursor=%+v", title, surface.Rows(), after.Cursor)
							}
						}
					}
				})
			}
		}
	}
}
