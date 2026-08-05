package vterm

import (
	"fmt"
	"strings"
	"testing"
)

// EvictedRows 是 logical-line 无限历史引擎的唯一 seal 信号：
// 每个 transaction 无条件携带真正离开 primary 可见区的行，
// 不经过 PrimaryScrollOut 的 sync/alt/clear attach gate。

func TestEvictedRowsCarriesOrdinaryStdoutScrollOut(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	tx, err := source.ApplyPTYWrite([]byte("one\r\ntwo\r\nthree\r\nfour"))
	if err != nil {
		t.Fatalf("apply pty write: %v", err)
	}
	// 契约不变：普通输出的 PrimaryScrollOut 仍被 gate 挡住。
	if len(tx.PrimaryScrollOut) != 0 {
		t.Fatalf("ordinary write must keep PrimaryScrollOut gated: %#v", tx.PrimaryScrollOut)
	}
	// 屏高 2：one/two 已离屏，three/four 仍在屏上。
	got := evictedTextsForTest(tx.EvictedRows)
	want := []string{"one", "two"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("evicted rows = %v, want %v", got, want)
	}
}

func TestEvictedRowsAccumulateInOrderAcrossTransactions(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	var evicted []string
	for i := 1; i <= 6; i++ {
		tx, err := source.ApplyPTYWrite([]byte(fmt.Sprintf("line%02d\r\n", i)))
		if err != nil {
			t.Fatalf("apply write %d: %v", i, err)
		}
		evicted = append(evicted, evictedTextsForTest(tx.EvictedRows)...)
	}
	// 6 行写入后屏上留 line06 与光标行，line01..line05 依次滚出。
	want := []string{"line01", "line02", "line03", "line04", "line05"}
	if strings.Join(evicted, "|") != strings.Join(want, "|") {
		t.Fatalf("evicted sequence = %v, want %v", evicted, want)
	}
}

func TestEvictedRowsKeepSoftWrapFlag(t *testing.T) {
	source := NewSemanticSource(6, 2, 100, nil)
	// 12 列文本在 6 列屏上软换行成两行，再用硬换行推出屏幕。
	tx, err := source.ApplyPTYWrite([]byte("abcdefghijkl\r\nx\r\ny"))
	if err != nil {
		t.Fatalf("apply pty write: %v", err)
	}
	rows := tx.EvictedRows
	if len(rows) < 2 {
		t.Fatalf("expected wrapped rows to scroll out, got %v", evictedTextsForTest(rows))
	}
	if got := semanticScrollOutText(rows[0]); got != "abcdef" {
		t.Fatalf("first evicted row = %q, want abcdef", got)
	}
	if !rows[0].WrappedSet || !rows[0].Wrapped {
		t.Fatalf("first evicted row must carry soft-wrap continuation flag: %#v", rows[0])
	}
	if got := semanticScrollOutText(rows[1]); got != "ghijkl" {
		t.Fatalf("second evicted row = %q, want ghijkl", got)
	}
	if rows[1].Wrapped {
		t.Fatalf("hard line end must not be marked wrapped: %#v", rows[1])
	}
}

func TestEvictedRowsIgnoreAltScreenScroll(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	if _, err := source.ApplyPTYWrite([]byte("shell\r\n")); err != nil {
		t.Fatalf("apply shell write: %v", err)
	}
	raw := "\x1b[?1049h"
	for i := 1; i <= 6; i++ {
		raw += fmt.Sprintf("alt%02d\r\n", i)
	}
	tx, err := source.ApplyPTYWrite([]byte(raw))
	if err != nil {
		t.Fatalf("apply alt write: %v", err)
	}
	for _, text := range evictedTextsForTest(tx.EvictedRows) {
		if strings.HasPrefix(text, "alt") {
			t.Fatalf("alt screen scroll must not evict into primary history: %v", evictedTextsForTest(tx.EvictedRows))
		}
	}
}

func TestEvictedRowsKeepPrimaryScrollOutBeforeAltEnterInSameWrite(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	// 同一次写入内：主屏先滚出 pre1/pre2，再进 alt 并在 alt 内滚动。
	// 进 alt 前的主屏 eviction 不能因 shape-changed 批次丢失，
	// alt 内滚动也不能混进主历史。
	raw := "pre1\r\npre2\r\npre3\r\npre4"
	raw += "\x1b[?1049h"
	for i := 1; i <= 6; i++ {
		raw += fmt.Sprintf("alt%02d\r\n", i)
	}
	tx, err := source.ApplyPTYWrite([]byte(raw))
	if err != nil {
		t.Fatalf("apply mixed write: %v", err)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	want := []string{"pre1", "pre2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("evicted rows = %v, want %v (primary rows before alt enter)", got, want)
	}
}

func TestEvictedRowsExcludeTransientAltSessionInSameWrite(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	// 同一次写入内 enter alt -> alt 滚动 -> exit alt：写入前后都在主屏
	// （same-shape 批次），alt 内滚出的行必须按有序 damage 流剔除。
	raw := "\x1b[?1049h"
	for i := 1; i <= 6; i++ {
		raw += fmt.Sprintf("alt%02d\r\n", i)
	}
	raw += "\x1b[?1049l"
	tx, err := source.ApplyPTYWrite([]byte(raw))
	if err != nil {
		t.Fatalf("apply transient alt write: %v", err)
	}
	if got := evictedTextsForTest(tx.EvictedRows); len(got) != 0 {
		t.Fatalf("transient alt session must not evict into primary history, got %v", got)
	}
}

func TestEvictedRowsPreserveBlankLines(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	// para1 / 空行 / para2 依次滚出：空行必须保留在 eviction 序列里，
	// 否则历史会丢段落间距。
	tx, err := source.ApplyPTYWrite([]byte("para1\r\n\r\npara2\r\nx\r\ny"))
	if err != nil {
		t.Fatalf("apply pty write: %v", err)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	want := []string{"para1", "", "para2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("evicted rows = %v, want %v (blank line preserved)", got, want)
	}
}

func TestEvictedRowsCoverSynchronizedOutput(t *testing.T) {
	source := NewSemanticSource(12, 3, 100, nil)
	raw := "\x1b[?2026h"
	for i := 1; i <= 8; i++ {
		raw += fmt.Sprintf("line%02d\r\n", i)
	}
	raw += "\x1b[?2026l"
	tx, err := source.ApplyPTYWrite([]byte(raw))
	if err != nil {
		t.Fatalf("apply synchronized output: %v", err)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	if len(got) < 6 || got[0] != "line01" {
		t.Fatalf("synchronized output must still evict scrolled rows in order, got %v", got)
	}
}

func TestEvictedRowsWorkWithScrollbackRingDisabled(t *testing.T) {
	// 生产 SemanticTap 用 scrollbackSize=0 创建 vterm（ring 关闭）。
	// EvictedRows 的 seal 信号必须来自 damage 捕获本身，不依赖 ring diff。
	source := NewSemanticSource(12, 2, 0, nil)
	tx, err := source.ApplyPTYWrite([]byte("one\r\ntwo\r\nthree\r\nfour"))
	if err != nil {
		t.Fatalf("apply pty write: %v", err)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	want := []string{"one", "two"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("evicted rows with ring disabled = %v, want %v", got, want)
	}
}

func TestEvictedRowsCoverED2ClearScreen(t *testing.T) {
	// ED2 的滚出行不走 recordScrollbackLine，而是随 ControlDamage("ed").ScrollOut
	// 携带；它们同样真正离开可见区（xterm `clear` 把整屏推进 scrollback），
	// 必须并入 EvictedRows，否则 clear 前的屏幕内容从历史中消失。
	source := NewSemanticSource(12, 3, 0, nil)
	if _, err := source.ApplyPTYWrite([]byte("aaa\r\nbbb\r\n")); err != nil {
		t.Fatalf("apply prelude write: %v", err)
	}
	tx, err := source.ApplyPTYWrite([]byte("\x1b[2J\x1b[H"))
	if err != nil {
		t.Fatalf("apply ED2 write: %v", err)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	want := []string{"aaa", "bbb"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ED2 evicted rows = %v, want %v", got, want)
	}
}

func TestEvictedRowsED2InsideAltStaysOutOfPrimary(t *testing.T) {
	source := NewSemanticSource(12, 3, 0, nil)
	if _, err := source.ApplyPTYWrite([]byte("shell\r\n")); err != nil {
		t.Fatalf("apply shell write: %v", err)
	}
	// alt 内的 ED2 清屏归属 alternate，不得混进主历史 eviction。
	tx, err := source.ApplyPTYWrite([]byte("\x1b[?1049haltrow\r\n\x1b[2J\x1b[H"))
	if err != nil {
		t.Fatalf("apply alt ED2 write: %v", err)
	}
	for _, text := range evictedTextsForTest(tx.EvictedRows) {
		if strings.Contains(text, "altrow") {
			t.Fatalf("alt ED2 scroll-out must not evict into primary history, got %v", evictedTextsForTest(tx.EvictedRows))
		}
	}
}

func evictedTextsForTest(rows []TerminalSemanticScrollOut) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, semanticScrollOutText(row))
	}
	return out
}
