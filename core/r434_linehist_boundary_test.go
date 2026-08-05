package core

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

// R434 准入（terminal 级）：linehist 在真实 Terminal 链路上的边界语义。
// resize 经 Server.ResizeTerminal（tap Resize 事务的 EvictedRows）；
// ED3 经 IngestOutput（tx.ClearScrollback 软页边界）；alt 期间 primary
// 时间线尾部保持可见（PrimarySavedScreenRows 投影）。

func TestR434LinehistResizeKeepsHistoryExact(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r434-resize", 12, 3)
	const totalLines = 20
	for i := 1; i <= totalLines; i++ {
		r433Ingest(t, server, "term-r434-resize", fmt.Sprintf("line%02d\r\n", i))
	}
	if err := server.ResizeTerminal(context.Background(), "term-r434-resize", 6, 2); err != nil {
		t.Fatalf("resize: %v", err)
	}
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r434-resize", 12, 5)
	joined := strings.Join(historyRowTexts(rows), "|")
	for i := 1; i <= totalLines; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("resize lost/duplicated %q, got %q", want, joined)
		}
	}
	// 变宽变高后继续写入仍不重不漏。
	if err := server.ResizeTerminal(context.Background(), "term-r434-resize", 24, 6); err != nil {
		t.Fatalf("resize taller: %v", err)
	}
	for i := totalLines + 1; i <= totalLines+5; i++ {
		r433Ingest(t, server, "term-r434-resize", fmt.Sprintf("line%02d\r\n", i))
	}
	rows, _ = r326CollectAllHistoryRows(t, server, "term-r434-resize", 12, 5)
	joined = strings.Join(historyRowTexts(rows), "|")
	for i := 1; i <= totalLines+5; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("post-resize history lost/duplicated %q, got %q", want, joined)
		}
	}
	for i := 1; i < totalLines+5; i++ {
		if strings.Index(joined, fmt.Sprintf("line%02d", i)) > strings.Index(joined, fmt.Sprintf("line%02d", i+1)) {
			t.Fatalf("post-resize history out of order: %q", joined)
		}
	}
}

func TestR439LinehistClearScrollbackKeepsHistoryAndToken(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r434-clear", 12, 3)
	for i := 1; i <= 6; i++ {
		r433Ingest(t, server, "term-r434-clear", fmt.Sprintf("line%02d\r\n", i))
	}
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-r434-clear", history.FreezeHistoryRequest{
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	// `clear` 完整形态：ED2 + ED3 + 归位，一次写入。
	r433Ingest(t, server, "term-r434-clear", "\x1b[2J\x1b[3J\x1b[H")
	live, err := server.TerminalHistoryWindow(context.Background(), "term-r434-clear", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("live window after clear: %v", err)
	}
	liveJoined := strings.Join(historyRowTexts(live.Rows), "|")
	for i := 1; i <= 6; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(liveJoined, want) != 1 {
			t.Fatalf("clear must keep live authoritative history %q exactly once, got %q", want, liveJoined)
		}
	}
	frozen, err := server.TerminalHistoryWindow(context.Background(), "term-r434-clear", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
		Token: snapshot.Token,
	})
	if err != nil {
		t.Fatalf("frozen window after clear: %v", err)
	}
	frozenJoined := strings.Join(historyRowTexts(frozen.Rows), "|")
	for i := 1; i <= 6; i++ {
		if !strings.Contains(frozenJoined, fmt.Sprintf("line%02d", i)) {
			t.Fatalf("frozen token must keep pre-clear content, got %q", frozenJoined)
		}
	}
	// clear 之后的新输出从头计入可见历史。
	for i := 7; i <= 9; i++ {
		r433Ingest(t, server, "term-r434-clear", fmt.Sprintf("line%02d\r\n", i))
	}
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r434-clear", 12, 5)
	joined := strings.Join(historyRowTexts(rows), "|")
	for i := 1; i <= 9; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("history across clear must contain %q exactly once, got %q", want, joined)
		}
	}
}

func TestR434LinehistAltKeepsPrimaryTailVisible(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r434-alt", 12, 3)
	for i := 1; i <= 5; i++ {
		r433Ingest(t, server, "term-r434-alt", fmt.Sprintf("line%02d\r\n", i))
	}
	r433Ingest(t, server, "term-r434-alt", "\x1b[?1049h\x1b[2J\x1b[HALT-ONLY")
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r434-alt", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest window during alt: %v", err)
	}
	byText := make(map[string]history.HistoryRow)
	for _, row := range window.Rows {
		byText[strings.TrimRight(strings.Join(historyRowTexts([]history.HistoryRow{row}), ""), " ")] = row
	}
	// line04/line05 在进入 alt 时仍在主屏上：alt 期间必须继续投影，
	// 且保持 mutable（alt 退出后程序仍可改写它们，不能 seal）。
	for _, want := range []string{"line04", "line05"} {
		row, ok := byText[want]
		if !ok {
			t.Fatalf("primary tail %q must stay visible during alt, got %v", want, historyRowTexts(window.Rows))
		}
		if row.Committed || row.Segment != history.HistorySegmentCurrentPrimaryFrame {
			t.Fatalf("primary tail %q must stay mutable primary-frame, got %#v", want, row)
		}
	}
	if _, ok := byText["ALT-ONLY"]; !ok {
		t.Fatalf("alt frame content missing, got %v", historyRowTexts(window.Rows))
	}
	// 退出 alt 后 primary 尾部不重复。
	r433Ingest(t, server, "term-r434-alt", "\x1b[?1049l")
	r433Ingest(t, server, "term-r434-alt", "post-alt\r\n")
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r434-alt", 12, 5)
	joined := strings.Join(historyRowTexts(rows), "|")
	if strings.Contains(joined, "ALT-ONLY") {
		t.Fatalf("alt transient content must not survive alt exit, got %q", joined)
	}
	for _, want := range []string{"line04", "line05", "post-alt"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("primary timeline must contain %q exactly once after alt exit, got %q", want, joined)
		}
	}
}
