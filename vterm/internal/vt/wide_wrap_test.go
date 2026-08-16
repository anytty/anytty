package vt

import "testing"

// 宽字符 autowrap 回归：R432 发现两处丢字——
// 1) 行尾剩余列不足一个宽字符时直接落在行尾被 SetCell 丢弃，
//    DECAWM 下应整字软换行（xterm 行为）；
// 2) 宽字符恰好填满行尾时 pending-wrap 判定按起始列（x>=width-1）算不中，
//    cursor 越界 clamp 回行尾，下一个字符擦掉刚写入的宽字符。

func screenTextForWideWrapTest(e *Emulator, y int) string {
	text := ""
	for x := 0; x < e.scr.Width(); x++ {
		if cell := e.CellAt(x, y); cell != nil {
			text += cell.String()
		}
	}
	return text
}

func TestWideCharWrapsWholeCharAtLineEnd(t *testing.T) {
	e := newTestTerminal(t, 6, 3)
	e.WriteString("abcde你")
	if got := screenTextForWideWrapTest(e, 0); got != "abcde " {
		t.Fatalf("row 0 = %q, want %q (wide char must not overwrite line tail)", got, "abcde ")
	}
	if got := screenTextForWideWrapTest(e, 1); got != "你    " {
		t.Fatalf("row 1 = %q, want wide char wrapped whole", got)
	}
	if !e.ScreenLineWrapped(0) {
		t.Fatalf("row 0 must carry soft-wrap flag after wide-char wrap")
	}
}

func TestWideCharExactFitKeepsPendingWrap(t *testing.T) {
	e := newTestTerminal(t, 6, 3)
	// 你好世 恰好填满 6 列；界 必须软换行到下一行，且 世 不能被擦掉。
	e.WriteString("你好世界")
	if got := screenTextForWideWrapTest(e, 0); got != "你好世" {
		t.Fatalf("row 0 = %q, want %q (exact-fit wide char must survive)", got, "你好世")
	}
	if got := screenTextForWideWrapTest(e, 1); got != "界    " {
		t.Fatalf("row 1 = %q, want next wide char soft-wrapped", got)
	}
	if !e.ScreenLineWrapped(0) {
		t.Fatalf("row 0 must carry soft-wrap flag on exact-fit pending wrap")
	}
}

func TestWideCharWrapContinuesRun(t *testing.T) {
	e := newTestTerminal(t, 4, 3)
	e.WriteString("你好世界")
	if got := screenTextForWideWrapTest(e, 0); got != "你好" {
		t.Fatalf("row 0 = %q, want 你好", got)
	}
	if got := screenTextForWideWrapTest(e, 1); got != "世界" {
		t.Fatalf("row 1 = %q, want 世界", got)
	}
}
