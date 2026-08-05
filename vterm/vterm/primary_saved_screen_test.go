package vterm

import (
	"strings"
	"testing"
	"time"
)

// R434 准入：alt 期间可读取 primary 屏保存行。linehist 无限历史用它投影
// "被 alt 覆盖但仍未滚出"的主屏时间线尾部，不引入第二份屏幕真值。

func primaryRowTextForTest(cells []Cell) string {
	var out strings.Builder
	for _, cell := range cells {
		out.WriteString(cell.Content)
	}
	return strings.TrimRight(out.String(), " ")
}

func TestPrimarySavedScreenRowsWhileAlt(t *testing.T) {
	vt := New(12, 3, 0, nil)
	defer vt.Close()
	mustWriteForTest := func(raw string) {
		t.Helper()
		if _, err := vt.Write([]byte(raw)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	mustWriteForTest("one\r\ntwo")
	mustWriteForTest("\x1b[?1049h\x1b[2J\x1b[HALT-FRAME")
	if !vt.IsAltScreen() {
		t.Fatalf("expected alt screen active")
	}
	rows, wrapped, timestamps := vt.PrimarySavedScreenRows()
	if len(rows) != 3 || len(wrapped) != 3 {
		t.Fatalf("primary rows/wrapped length = %d/%d, want 3/3", len(rows), len(wrapped))
	}
	if primaryRowTextForTest(rows[0]) != "one" || primaryRowTextForTest(rows[1]) != "two" {
		t.Fatalf("primary saved rows = %q/%q, want one/two", primaryRowTextForTest(rows[0]), primaryRowTextForTest(rows[1]))
	}
	if timestamps[0].IsZero() || timestamps[1].IsZero() {
		t.Fatalf("primary saved timestamps = %v, want non-zero content timestamps", timestamps)
	}
	for y, flag := range wrapped {
		if flag {
			t.Fatalf("hard-ended primary row %d must not be wrapped", y)
		}
	}
	// 退出 alt 后 primary 视图与 active screen 一致。
	mustWriteForTest("\x1b[?1049l")
	beforeExitTimestamps := append([]time.Time(nil), timestamps...)
	rows, _, timestamps = vt.PrimarySavedScreenRows()
	if primaryRowTextForTest(rows[0]) != "one" || primaryRowTextForTest(rows[1]) != "two" {
		t.Fatalf("primary rows after alt exit = %q/%q, want one/two", primaryRowTextForTest(rows[0]), primaryRowTextForTest(rows[1]))
	}
	if !timestamps[0].Equal(beforeExitTimestamps[0]) || !timestamps[1].Equal(beforeExitTimestamps[1]) {
		t.Fatalf("primary timestamps changed across alt exit: before=%v after=%v", beforeExitTimestamps, timestamps)
	}
}

func TestPrimarySavedScreenRowsKeepsSoftWrapFlags(t *testing.T) {
	vt := New(6, 3, 0, nil)
	defer vt.Close()
	if _, err := vt.Write([]byte("abcdefghijkl")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := vt.Write([]byte("\x1b[?1049hALT")); err != nil {
		t.Fatalf("enter alt: %v", err)
	}
	rows, wrapped, _ := vt.PrimarySavedScreenRows()
	if primaryRowTextForTest(rows[0]) != "abcdef" || primaryRowTextForTest(rows[1]) != "ghijkl" {
		t.Fatalf("primary rows = %q/%q, want abcdef/ghijkl", primaryRowTextForTest(rows[0]), primaryRowTextForTest(rows[1]))
	}
	if !wrapped[0] || wrapped[1] {
		t.Fatalf("wrapped flags = %v, want row0 wrapped into row1 only", wrapped)
	}
}
