package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

// R435 准入（server 级回归等价）：旧 screen-backed 验收（r328/r331/r334 系列）
// 的行为期望在 linehist 引擎上重建为等价断言——断言行为而非实现：
// clear 后旧内容仍可翻页、prompt 不插到当前 frame 之前、已滚出的 shell
// 尾部不被 pseudo-TUI 重绘复制、CJK 跨冷热段保真。

func r435ReadFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "fixtures", "history", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(raw)
}

// r328 等价：ED2 把被清的屏幕内容挤进 scrollback，clear 前的会话内容
// 必须完整留在可翻页历史里，且每行恰一次。
func TestR435LinehistED2ClearPreservesPreviousSessionsInHistory(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r435-sessions", 20, 3)
	for session := 1; session <= 3; session++ {
		for i := 1; i <= 4; i++ {
			r433Ingest(t, server, "term-r435-sessions", fmt.Sprintf("s%d-line%d\r\n", session, i))
		}
		if session < 3 {
			r433Ingest(t, server, "term-r435-sessions", "\x1b[2J\x1b[H")
		}
	}
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r435-sessions", 20, 5)
	joined := strings.Join(historyRowTexts(rows), "|")
	var wants []string
	for session := 1; session <= 3; session++ {
		for i := 1; i <= 4; i++ {
			wants = append(wants, fmt.Sprintf("s%d-line%d", session, i))
		}
	}
	for _, want := range wants {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("multi-session history must contain %q exactly once, got %q", want, joined)
		}
	}
	for i := 0; i < len(wants)-1; i++ {
		if strings.Index(joined, wants[i]) > strings.Index(joined, wants[i+1]) {
			t.Fatalf("multi-session history out of order (%q vs %q): %q", wants[i], wants[i+1], joined)
		}
	}
}

// r331 等价：屏内重绘后出现的普通 prompt 不得插到已滚出内容之前；
// 投影顺序恒为 冷段（滚出序）→ 当前屏（屏序）。
func TestR435LinehistPromptAfterRepaintKeepsTimelineOrder(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r435-order", 24, 3)
	for i := 1; i <= 4; i++ {
		r433Ingest(t, server, "term-r435-order", fmt.Sprintf("shell-%d\r\n", i))
	}
	// 屏内 pseudo-TUI 重绘（光标上移 + 整行清除 + 改写），随后普通 prompt。
	r433Ingest(t, server, "term-r435-order", "\x1b[1A\x1b[2KFRAME-A")
	r433Ingest(t, server, "term-r435-order", "\r\nprompt-after\r\n")
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r435-order", 24, 5)
	joined := strings.Join(historyRowTexts(rows), "|")
	for _, want := range []string{"shell-1", "shell-2", "FRAME-A", "prompt-after"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("history must contain %q exactly once, got %q", want, joined)
		}
	}
	if strings.Index(joined, "prompt-after") < strings.Index(joined, "FRAME-A") {
		t.Fatalf("prompt must not interleave before repainted frame, got %q", joined)
	}
	if strings.Index(joined, "FRAME-A") < strings.Index(joined, "shell-2") {
		t.Fatalf("repainted frame must not jump before sealed shell lines, got %q", joined)
	}
}

// r334 等价：已滚出的 shell 尾部是 sealed truth，codex 式 pseudo-TUI
// 重绘不得复制它（真实 fixture 走完整 Server ingest 链路）。
func TestR435LinehistCodexFixtureViaServerIngest(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r435-codex", 24, 3)
	payload := r435ReadFixture(t, "codex_pseudotui.ansi")
	// 按字节切片 ingest，模拟 PTY 任意分包。
	for _, b := range []byte(payload) {
		r433Ingest(t, server, "term-r435-codex", string([]byte{b}))
	}
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r435-codex", 24, 5)
	joined := strings.Join(historyRowTexts(rows), "|")
	for _, want := range []string{"shell prompt 1", "shell prompt 2", "OpenAI Codex frame"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("history must contain %q exactly once, got %q", want, joined)
		}
	}
	if strings.Contains(joined, "codex starts") {
		t.Fatalf("erased on-screen line must not enter history, got %q", joined)
	}
}

func TestR435LinehistCodexStylePrimaryMutationDoesNotDuplicateHistory(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r435-codex-mutation", 40, 5)
	payload := "shell prompt 1\r\nshell prompt 2\r\nshell prompt 3\r\ncodex starts\r\n" +
		"\x1b[1A\x1b[2KOpenAI Codex frame" +
		"\x1b[1A\x1b[6G\x1b[1Pmodified frame"
	r433Ingest(t, server, "term-r435-codex-mutation", payload)

	rows, _ := r326CollectAllHistoryRows(t, server, "term-r435-codex-mutation", 40, 10)
	joined := strings.Join(historyRowTexts(rows), "\n")
	for _, needle := range []string{"shell prompt 1", "shell prompt 2"} {
		if got := historyTextCount(rows, needle); got != 1 {
			t.Fatalf("%q must appear once after primary mutation repaint, count=%d:\n%s\nrows=%#v", needle, got, joined, rows)
		}
	}
	if got := historyTextCount(rows, "OpenAI Codex frame"); got > 1 {
		t.Fatalf("intermediate primary repaint must not be appended repeatedly, count=%d:\n%s\nrows=%#v", got, joined, rows)
	}
	if !strings.Contains(joined, "modified frame") {
		t.Fatalf("latest projection should include final mutated frame state:\n%s\nrows=%#v", joined, rows)
	}
}

// CJK 等价：宽字符行跨滚出/落盘/重投影/复制全程保真。
func TestR435LinehistCJKAcrossEvictionWindowAndCopy(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r435-cjk", 8, 2)
	lines := []string{"中文历史第一行", "中文第二行", "third", "第四行"}
	for _, line := range lines {
		r433Ingest(t, server, "term-r435-cjk", line+"\r\n")
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r435-cjk", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  40,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	joined := strings.Join(historyRowTexts(window.Rows), "|")
	for _, want := range lines {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("CJK line %q must project exactly once at wide cols, got %q", want, joined)
		}
	}
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-r435-cjk", history.FreezeHistoryRequest{Cols: 40, Limit: 100})
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	startLineID, ok := historyLineIDForText(window.Rows, "中文历史第一行")
	if !ok {
		t.Fatalf("latest window missing CJK copy start row, rows=%#v", window.Rows)
	}
	endLineID, ok := historyLineIDForText(window.Rows, "中文第二行")
	if !ok {
		t.Fatalf("latest window missing CJK copy end row, rows=%#v", window.Rows)
	}
	copied, err := server.TerminalHistoryCopy(context.Background(), "term-r435-cjk", history.HistoryCopyRequest{
		Token: snapshot.Token,
		Cols:  40,
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: startLineID}, End: history.HistoryCopyPosition{LineID: endLineID, Col: 12}},
	})
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if copied != "中文历史第一行\n中文第二行" {
		t.Fatalf("CJK copy = %q, want first two lines verbatim", copied)
	}
}
