package linehist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

// R435 准入（fixture 级）：真实 vterm 事务喂 `fixtures/history/*.ansi`，
// 断言旧 screen-backed 验收的行为期望在 linehist 上等价成立：
// 普通输出每行恰一次；CR/光标重绘不污染历史（屏上修改在滚出前收敛）；
// pseudo-TUI 重绘不重复已滚出的 shell 尾部；alt 瞬态不进历史；宽字符保真。

func readFixtureForTest(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "history", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(raw)
}

func assertExactlyOnceInOrder(t *testing.T, texts []string, wants []string) {
	t.Helper()
	joined := strings.Join(texts, "|")
	for _, want := range wants {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("projection must contain %q exactly once, got %v", want, texts)
		}
	}
	for i := 0; i < len(wants)-1; i++ {
		if strings.Index(joined, wants[i]) > strings.Index(joined, wants[i+1]) {
			t.Fatalf("projection out of order (%q after %q): %v", wants[i], wants[i+1], texts)
		}
	}
}

func TestFixturePlainShellEachLineExactlyOnce(t *testing.T) {
	harness := newStoreHarness(t, 12, 2)
	harness.write(readFixtureForTest(t, "plain-shell.ansi"))
	texts := collectAllTextsForTest(t, harness.store, 12, 100)
	assertExactlyOnceInOrder(t, texts, []string{"one", "two", "three"})
}

func TestFixtureProgressCRRepaintKeepsFinalStateOnly(t *testing.T) {
	harness := newStoreHarness(t, 24, 2)
	harness.write(readFixtureForTest(t, "progress-cr-repaint.ansi"))
	texts := collectAllTextsForTest(t, harness.store, 24, 100)
	joined := strings.Join(texts, "|")
	if strings.Contains(joined, "progress 10%") {
		t.Fatalf("CR-overwritten intermediate state must not enter history, got %v", texts)
	}
	assertExactlyOnceInOrder(t, texts, []string{"progress 90%", "done"})
}

func TestFixturePseudoTUIRepaintKeepsFinalStateOnly(t *testing.T) {
	harness := newStoreHarness(t, 24, 2)
	harness.write(readFixtureForTest(t, "pseudo-tui-repaint.ansi"))
	texts := collectAllTextsForTest(t, harness.store, 24, 100)
	joined := strings.Join(texts, "|")
	if strings.Contains(joined, "status: pending") {
		t.Fatalf("in-place repainted state must not enter history, got %v", texts)
	}
	assertExactlyOnceInOrder(t, texts, []string{"status: done", "answer: ready"})
}

func TestFixtureCodexPseudoTUIDoesNotDuplicateSealedShellTail(t *testing.T) {
	harness := newStoreHarness(t, 24, 3)
	harness.write(readFixtureForTest(t, "codex_pseudotui.ansi"))
	texts := collectAllTextsForTest(t, harness.store, 24, 100)
	joined := strings.Join(texts, "|")
	// 已滚出的 shell 行是 sealed truth：屏内 pseudo-TUI 重绘不得复制或改写它们。
	assertExactlyOnceInOrder(t, texts, []string{"shell prompt 1", "shell prompt 2"})
	if strings.Count(joined, "OpenAI Codex frame") != 1 {
		t.Fatalf("current frame must project exactly once, got %v", texts)
	}
	if strings.Contains(joined, "codex starts") {
		t.Fatalf("erased on-screen line must not enter history, got %v", texts)
	}
	if !strings.Contains(joined, "modified frame") {
		t.Fatalf("final repaint state missing, got %v", texts)
	}
}

func TestFixtureCodexPseudoTUIChunkedWritesMatchSingleWrite(t *testing.T) {
	payload := readFixtureForTest(t, "codex_pseudotui.ansi")
	single := newStoreHarness(t, 24, 3)
	single.write(payload)
	chunked := newStoreHarness(t, 24, 3)
	for _, b := range []byte(payload) {
		chunked.write(string([]byte{b}))
	}
	singleTexts := collectAllTextsForTest(t, single.store, 24, 100)
	chunkedTexts := collectAllTextsForTest(t, chunked.store, 24, 100)
	if strings.Join(singleTexts, "|") != strings.Join(chunkedTexts, "|") {
		t.Fatalf("projection must not depend on write chunking:\nsingle=%v\nchunked=%v", singleTexts, chunkedTexts)
	}
}

func TestFixtureScrollRegionKeepsHistoryConsistent(t *testing.T) {
	harness := newStoreHarness(t, 12, 3)
	harness.write(readFixtureForTest(t, "scroll-region.ansi"))
	texts := collectAllTextsForTest(t, harness.store, 12, 100)
	joined := strings.Join(texts, "|")
	for _, want := range []string{"top", "after"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("scroll-region projection must contain %q exactly once, got %v", want, texts)
		}
	}
	for _, once := range []string{"mid", "bot"} {
		if strings.Count(joined, once) > 1 {
			t.Fatalf("scroll-region must not duplicate %q, got %v", once, texts)
		}
	}
}

func TestFixtureAltScreenTransientStaysOutOfHistory(t *testing.T) {
	harness := newStoreHarness(t, 24, 3)
	harness.write(readFixtureForTest(t, "alt-screen.ansi"))
	texts := collectAllTextsForTest(t, harness.store, 24, 100)
	joined := strings.Join(texts, "|")
	if strings.Contains(joined, "alt screen") {
		t.Fatalf("alt transient content must not stay in history after exit, got %v", texts)
	}
	for _, want := range []string{"primary", "back"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("primary timeline must contain %q exactly once, got %v", want, texts)
		}
	}
}

func TestFixtureWideCharSurvivesEvictionAndReprojection(t *testing.T) {
	payload := readFixtureForTest(t, "wide-char.ansi")
	wantFirst := payload[:strings.Index(payload, "\x1bE")]
	harness := newStoreHarness(t, 12, 2)
	harness.write(payload + "\x1bEx\x1bEy\x1bEz")
	texts := collectAllTextsForTest(t, harness.store, 12, 100)
	assertExactlyOnceInOrder(t, texts, []string{wantFirst, "ascii"})
	// 窄列投影按 grapheme 宽度换行，宽字符不撕裂。
	narrow, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 2, Limit: 100})
	if err != nil {
		t.Fatalf("narrow window: %v", err)
	}
	narrowJoined := strings.Join(windowTextsForTest(narrow), "")
	if !strings.Contains(strings.ReplaceAll(narrowJoined, " ", ""), strings.ReplaceAll(wantFirst, " ", "")) {
		t.Fatalf("narrow reprojection lost wide chars: %v", windowTextsForTest(narrow))
	}
}
