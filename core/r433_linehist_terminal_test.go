package core

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
)

// R433 准入：linehist store 经 LineHistoryStoreFactory 接入真实 Terminal
// ingest 链路（tap 事务 EvictedRows 落盘 + 查询投影 emulator 当前屏）。
// 断言口径与旧 screen-backed 验收一致：不重不漏、分页闭环、freeze/copy
// 语义、alt 内容不进 committed 历史。

func newR433LinehistServer(t *testing.T) *Server {
	t.Helper()
	server := NewServer(
		WithProcessFactory(newRecordingProcessFactory()),
		WithHistoryStoreFactory(LineHistoryStoreFactory(t.TempDir())),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	return server
}

func r433RegisterTerminal(t *testing.T, server *Server, id string, cols uint16, rows uint16) {
	t.Helper()
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      id,
		Command: []string{"shell"},
		Size:    Size{Cols: cols, Rows: rows},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
}

func r433Ingest(t *testing.T, server *Server, id string, output string) {
	t.Helper()
	if err := server.IngestOutput(context.Background(), id, output); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
}

func TestR433LinehistLatestWindowHasEachLineExactlyOnce(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r433-latest", 12, 3)
	for i := 1; i <= 8; i++ {
		r433Ingest(t, server, "term-r433-latest", fmt.Sprintf("line%02d\r\n", i))
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r433-latest", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	texts := historyRowTexts(window.Rows)
	joined := strings.Join(texts, "|")
	for i := 1; i <= 8; i++ {
		want := fmt.Sprintf("line%02d", i)
		if strings.Count(joined, want) != 1 {
			t.Fatalf("history must contain %q exactly once, got %v", want, texts)
		}
	}
	for i := 1; i < 8; i++ {
		if strings.Index(joined, fmt.Sprintf("line%02d", i)) > strings.Index(joined, fmt.Sprintf("line%02d", i+1)) {
			t.Fatalf("history lines out of order: %v", texts)
		}
	}
}

func TestR433LinehistOlderPagingRoundTrip(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r433-paging", 12, 3)
	const totalLines = 30
	for i := 1; i <= totalLines; i++ {
		r433Ingest(t, server, "term-r433-paging", fmt.Sprintf("line%02d\r\n", i))
	}
	rows, pageCount := r326CollectAllHistoryRows(t, server, "term-r433-paging", 12, 5)
	if pageCount < 3 {
		t.Fatalf("expected multi-page older traversal, pages=%d", pageCount)
	}
	texts := historyRowTexts(rows)
	want := make([]string, 0, totalLines)
	for i := 1; i <= totalLines; i++ {
		want = append(want, fmt.Sprintf("line%02d", i))
	}
	if strings.Join(texts, "|") != strings.Join(want, "|") {
		t.Fatalf("older paging round trip = %v, want %v", texts, want)
	}
}

func TestR433LinehistProjectsSoftWrapByRequestCols(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r433-wrap", 6, 2)
	r433Ingest(t, server, "term-r433-wrap", "abcdefghijkl\r\nx\r\ny\r\nz\r\n")
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r433-wrap", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  20,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	texts := historyRowTexts(window.Rows)
	if len(texts) == 0 || texts[0] != "abcdefghijkl" {
		t.Fatalf("wide query cols must rejoin evicted soft wrap into one row, got %v", texts)
	}
}

func TestR433LinehistFreezeAndCopyAcrossColdAndHot(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r433-copy", 12, 3)
	for i := 1; i <= 6; i++ {
		r433Ingest(t, server, "term-r433-copy", fmt.Sprintf("line%02d\r\n", i))
	}
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-r433-copy", history.FreezeHistoryRequest{
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	for i := 7; i <= 9; i++ {
		r433Ingest(t, server, "term-r433-copy", fmt.Sprintf("line%02d\r\n", i))
	}
	frozen, err := server.TerminalHistoryWindow(context.Background(), "term-r433-copy", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
		Token: snapshot.Token,
	})
	if err != nil {
		t.Fatalf("frozen latest window: %v", err)
	}
	frozenText := strings.Join(historyRowTexts(frozen.Rows), "|")
	if strings.Contains(frozenText, "line07") {
		t.Fatalf("frozen window must not include post-freeze output, got %q", frozenText)
	}
	if !strings.Contains(frozenText, "line06") {
		t.Fatalf("frozen window missing pre-freeze tail, got %q", frozenText)
	}
	startLineID, ok := historyLineIDForText(frozen.Rows, "line02")
	if !ok {
		t.Fatalf("frozen window missing copy start row, rows=%#v", frozen.Rows)
	}
	endLineID, ok := historyLineIDForText(frozen.Rows, "line05")
	if !ok {
		t.Fatalf("frozen window missing copy end row, rows=%#v", frozen.Rows)
	}
	// 冷段（已落盘）到热段（当时屏上）的跨段复制。
	copied, err := server.TerminalHistoryCopy(context.Background(), "term-r433-copy", history.HistoryCopyRequest{
		Token: snapshot.Token,
		Cols:  12,
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: startLineID}, End: history.HistoryCopyPosition{LineID: endLineID, Col: 6}},
	})
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if copied != "line02\nline03\nline04\nline05" {
		t.Fatalf("copy range = %q, want line02..line05", copied)
	}
	if err := server.TerminalHistoryRelease(context.Background(), "term-r433-copy", snapshot.Token); err != nil {
		t.Fatalf("release: %v", err)
	}
}

func TestR433LinehistInitialSearchPrioritizesCurrentScreenThenWrapsColdHistory(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r433-search-frontier", 24, 3)
	for i := 1; i <= 20; i++ {
		r433Ingest(t, server, "term-r433-search-frontier", fmt.Sprintf("cold-line-%02d\r\n", i))
	}
	r433Ingest(t, server, "term-r433-search-frontier", "current-screen-needle")

	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-r433-search-frontier", history.FreezeHistoryRequest{
		Cols:  24,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	t.Cleanup(func() {
		_ = server.TerminalHistoryRelease(context.Background(), "term-r433-search-frontier", snapshot.Token)
	})

	current, err := server.TerminalHistorySearch(context.Background(), "term-r433-search-frontier", history.HistorySearchRequest{
		Token: snapshot.Token, Cols: 24, Limit: 10,
		Query: "current-screen-needle", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatalf("search current screen: %v", err)
	}
	if !current.Found || current.Wrapped {
		t.Fatalf("initial search must inspect the frozen current screen before cold history: %#v", current)
	}

	cold, err := server.TerminalHistorySearch(context.Background(), "term-r433-search-frontier", history.HistorySearchRequest{
		Token: snapshot.Token, Cols: 24, Limit: 10,
		Query: "cold-line-01", Direction: history.HistorySearchForward,
	})
	if err != nil {
		t.Fatalf("search cold history: %v", err)
	}
	if !cold.Found || !cold.Wrapped {
		t.Fatalf("initial search must wrap from the in-memory frontier into cold history: %#v", cold)
	}
}

func TestR433LinehistProcessExitSealsCurrentScreenBeforeRestart(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryStoreFactory(LineHistoryStoreFactory(t.TempDir())),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r433-exit-tail",
		Command: []string{"shell"},
		Size:    Size{Cols: 12, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	first := factory.process("term-r433-exit-tail")
	if first == nil {
		t.Fatal("expected first process")
	}
	for i := 1; i <= 8; i++ {
		first.emitOutput(fmt.Sprintf("line%02d\r\n", i))
	}
	first.exit(0)
	waitForTerminalState(t, server, "term-r433-exit-tail", TerminalStateExited)
	if err := server.RestartTerminal(context.Background(), "term-r433-exit-tail"); err != nil {
		t.Fatalf("restart terminal: %v", err)
	}
	second := factory.process("term-r433-exit-tail")
	if second == nil || second == first {
		t.Fatalf("expected restarted process, first=%p second=%p", first, second)
	}
	second.emitOutput("newprompt\r\n")
	waitForLiveRow(t, server, "term-r433-exit-tail", "newprompt")
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r433-exit-tail", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest history window: %v", err)
	}
	texts := rawHistoryRowTexts(window.Rows)
	joined := strings.Join(texts, "|")
	for _, want := range []string{"line01", "line02", "line03", "line04", "line05", "line06", "line07", "line08", "newprompt"} {
		if strings.Count(joined, want) != 1 {
			t.Fatalf("history after restart must contain %q exactly once, got %v", want, texts)
		}
	}
	if strings.Count(joined, "terminal started: term-r433-exit-tail") != 2 {
		t.Fatalf("initial start and restart markers must be in history, got %v", texts)
	}
	if strings.Count(joined, "started at: ") != 2 {
		t.Fatalf("start time markers must be in history, got %v", texts)
	}
	if strings.Count(joined, "terminal exited: term-r433-exit-tail code:0 exited") != 1 {
		t.Fatalf("exit marker must be in history exactly once, got %v", texts)
	}
	if strings.Count(joined, "exited at: ") != 1 {
		t.Fatalf("exit time marker must be in history exactly once, got %v", texts)
	}
	exitSpacer := -1
	for index, text := range texts {
		if text == "" && index+1 < len(texts) && strings.HasPrefix(texts[index+1], "terminal exited: term-r433-exit-tail") {
			exitSpacer = index
			break
		}
	}
	if exitSpacer < 0 {
		t.Fatalf("history must preserve blank lifecycle spacer before exit marker, got %v", texts)
	}
	firstStart := strings.Index(joined, "terminal started: term-r433-exit-tail")
	lineTail := strings.Index(joined, "line08")
	exitMarker := strings.Index(joined, "terminal exited: term-r433-exit-tail")
	secondStart := strings.LastIndex(joined, "terminal started: term-r433-exit-tail")
	newPrompt := strings.Index(joined, "newprompt")
	spacerMarker := strings.Index(joined, "||terminal exited: term-r433-exit-tail")
	if !(firstStart < lineTail && lineTail < spacerMarker && spacerMarker < exitMarker && exitMarker < secondStart && secondStart < newPrompt) {
		t.Fatalf("lifecycle markers must bracket old/new process history, got %v", texts)
	}
}

func TestR433LinehistAltScreenContentStaysOutOfCommittedHistory(t *testing.T) {
	server := newR433LinehistServer(t)
	r433RegisterTerminal(t, server, "term-r433-alt", 12, 3)
	for i := 1; i <= 5; i++ {
		r433Ingest(t, server, "term-r433-alt", fmt.Sprintf("line%02d\r\n", i))
	}
	r433Ingest(t, server, "term-r433-alt", "\x1b[?1049h\x1b[2J\x1b[HALT-ONLY")
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r433-alt", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest window: %v", err)
	}
	for _, row := range window.Rows {
		text := strings.TrimRight(strings.Join(historyRowTexts([]history.HistoryRow{row}), ""), " ")
		if row.Committed && strings.Contains(text, "ALT-ONLY") {
			t.Fatalf("alt content must not be committed into history, row=%#v", row)
		}
	}
	joined := strings.Join(historyRowTexts(window.Rows), "|")
	for _, want := range []string{"line01", "line02", "line03"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("committed lines missing during alt, got %q", joined)
		}
	}
	// 退出 alt 后主屏时间线恢复，alt 瞬态内容不留在历史里。
	r433Ingest(t, server, "term-r433-alt", "\x1b[?1049l")
	r433Ingest(t, server, "term-r433-alt", "post-alt\r\n")
	after, err := server.TerminalHistoryWindow(context.Background(), "term-r433-alt", history.HistoryWindowRequest{
		Mode:  history.HistoryWindowModeLatest,
		Cols:  12,
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("latest window after alt exit: %v", err)
	}
	afterText := strings.Join(historyRowTexts(after.Rows), "|")
	if strings.Contains(afterText, "ALT-ONLY") {
		t.Fatalf("alt transient content must not survive alt exit, got %q", afterText)
	}
	if !strings.Contains(afterText, "post-alt") {
		t.Fatalf("post-alt primary output missing, got %q", afterText)
	}
}
