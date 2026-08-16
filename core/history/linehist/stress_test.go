package linehist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

// R435 准入（stress 级）：大流量下投影精确、分页闭环、冷历史跨重启不丢不重。
// 无限性口径：内存只有 offset/prefix 索引，正文全在文件；两万行量级验证
// 首次查询建索引与随机分页都在秒级内完成。

func TestStressLargeVolumeProjectionStaysExact(t *testing.T) {
	dir := t.TempDir()
	harness := newStoreHarnessInDir(t, dir, "term-stress", 12, 3)
	t.Cleanup(func() { _ = harness.store.Close() })
	const totalLines = 20000
	const linesPerWrite = 100
	var batch strings.Builder
	for i := 1; i <= totalLines; i++ {
		fmt.Fprintf(&batch, "line%05d\r\n", i)
		if i%linesPerWrite == 0 {
			harness.write(batch.String())
			batch.Reset()
		}
	}
	latest, err := harness.store.LatestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 10})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	latestJoined := strings.Join(windowTextsForTest(latest), "|")
	if !strings.Contains(latestJoined, fmt.Sprintf("line%05d", totalLines)) {
		t.Fatalf("latest window missing newest line, got %q", latestJoined)
	}
	oldest, err := harness.store.OldestWindow(history.HistoryWindowRequest{Cols: 12, Limit: 5})
	if err != nil {
		t.Fatalf("oldest window: %v", err)
	}
	oldestTexts := windowTextsForTest(oldest)
	if len(oldestTexts) == 0 || oldestTexts[0] != "line00001" {
		t.Fatalf("oldest window must start at line00001, got %v", oldestTexts)
	}
	texts := collectAllTextsForTest(t, harness.store, 12, history.MaxHistoryWindowLines)
	want := make([]string, 0, totalLines)
	for i := 1; i <= totalLines; i++ {
		want = append(want, fmt.Sprintf("line%05d", i))
	}
	// 屏上可能带一行空行（光标行之前的空内容不投影，尾部裁剪），
	// 只要求整段前缀精确等于全部行序列。
	if got := strings.Join(texts, "|"); !strings.HasPrefix(got, strings.Join(want, "|")) {
		gotHead := got
		if len(gotHead) > 200 {
			gotHead = gotHead[:200] + "..."
		}
		t.Fatalf("large volume projection diverged, head=%q rows=%d want=%d", gotHead, len(texts), len(want))
	}
	if len(texts) < totalLines || len(texts) > totalLines+1 {
		t.Fatalf("large volume row count = %d, want %d..%d", len(texts), totalLines, totalLines+1)
	}
}

func TestStressColdHistorySurvivesReopenExactly(t *testing.T) {
	dir := t.TempDir()
	harness := newStoreHarnessInDir(t, dir, "term-stress-reopen", 12, 3)
	const totalLines = 5000
	var batch strings.Builder
	for i := 1; i <= totalLines; i++ {
		fmt.Fprintf(&batch, "line%05d\r\n", i)
		if i%50 == 0 {
			harness.write(batch.String())
			batch.Reset()
		}
	}
	coldCount := harness.store.engine.LineCount()
	if coldCount == 0 {
		t.Fatalf("expected evicted cold history before reopen")
	}
	if err := harness.store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
	reopened := newStoreHarnessInDir(t, dir, "term-stress-reopen", 12, 3)
	t.Cleanup(func() { _ = reopened.store.Close() })
	if got := reopened.store.engine.LineCount(); got != coldCount {
		t.Fatalf("cold line count after reopen = %d, want %d", got, coldCount)
	}
	// seal-on-eviction 口径：滚出的行跨重启不丢不重；重启时仍在屏上的行
	// 随 vterm 生命周期消失（emulator 是唯一屏幕真值，文件不存屏）。
	texts := collectAllTextsForTest(t, reopened.store, 12, history.MaxHistoryWindowLines)
	want := make([]string, 0, coldCount)
	for i := 1; i <= coldCount; i++ {
		want = append(want, fmt.Sprintf("line%05d", i))
	}
	if strings.Join(texts, "|") != strings.Join(want, "|") {
		t.Fatalf("cold history after reopen diverged: rows=%d want=%d", len(texts), len(want))
	}
}
