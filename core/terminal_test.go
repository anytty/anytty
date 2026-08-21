package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/core/history/linehist"
	"github.com/anytty/anytty/core/live"
	vterm "github.com/anytty/anytty/vterm/vterm"
)

func TestTerminalLifecycleAndLiveSurface(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	info, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-1",
		Command: []string{"shell"},
		Size:    Size{Cols: 40, Rows: 6},
	})
	if err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if info.State != TerminalStateRunning {
		t.Fatalf("unexpected state %q", info.State)
	}
	process := factory.process("term-1")
	if process == nil {
		t.Fatal("expected process to be spawned")
	}
	rows, err := server.LiveRows("term-1")
	if err != nil {
		t.Fatalf("initial live rows: %v", err)
	}
	initial := strings.Join(rows, "\n")
	if !strings.Contains(initial, "terminal started: term-1") || !strings.Contains(initial, "started at: ") {
		t.Fatalf("live rows must include terminal start marker, got %#v", rows)
	}
	if err := server.WriteInput(context.Background(), "term-1", []byte("echo hi\n")); err != nil {
		t.Fatalf("write input: %v", err)
	}
	inputs, _, _, _ := process.snapshot()
	if got := inputs[0]; string(got) != "echo hi\n" {
		t.Fatalf("unexpected input %q", string(got))
	}
	if err := server.IngestOutput(context.Background(), "term-1", "hello\nworld"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	rows, err = server.LiveRows("term-1")
	if err != nil {
		t.Fatalf("live rows: %v", err)
	}
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "hello") || !strings.Contains(joined, "world") {
		t.Fatalf("unexpected live rows %#v", rows)
	}
	if err := server.ResizeTerminal(context.Background(), "term-1", 20, 5); err != nil {
		t.Fatalf("resize: %v", err)
	}
	_, resizes, _, _ := process.snapshot()
	if got := resizes[0]; got != (Size{Cols: 20, Rows: 5}) {
		t.Fatalf("unexpected resize %#v", got)
	}
	info, err = server.GetTerminal("term-1")
	if err != nil {
		t.Fatalf("get terminal after resize: %v", err)
	}
	if info.Size != (Size{Cols: 20, Rows: 5}) {
		t.Fatalf("expected registry size to follow resize, got %#v", info.Size)
	}
}

func TestR373ResizeSerializesProcessOutputWithTapResize(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-resize-order",
		Command: []string{"shell"},
		Size:    Size{Cols: 5, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-r373-resize-order")
	if process == nil {
		t.Fatal("expected process to be spawned")
	}
	process.setResizeHook(func(size Size) {
		process.emitOutput("\x1b[2Jabcdefg")
	})
	if err := server.ResizeTerminal(context.Background(), "term-r373-resize-order", 8, 3); err != nil {
		t.Fatalf("resize terminal: %v", err)
	}
	terminal, err := server.Terminal("term-r373-resize-order")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	assertEventually(t, time.Second, func() bool {
		snapshot := terminal.NativeScreenSnapshot("term-r373-resize-order")
		return snapshot.Size.Cols == 8 && terminalSnapshotContainsRowText(snapshot, "abcdefg")
	}, "resize output should be applied after tap resize")
	snapshot := terminal.NativeScreenSnapshot("term-r373-resize-order")
	if snapshot.Size.Cols != 8 {
		t.Fatalf("tap latest screen should use resized width, got %#v", snapshot.Size)
	}
	rowText, rowCells, ok := terminalSnapshotRowText(snapshot, "abcdefg")
	if !ok {
		t.Fatalf("resize-triggered output row missing, snapshot=%#v", snapshot)
	}
	if got := len(rowCells); got != 7 {
		t.Fatalf("resize-triggered output should not wrap at old width, row cells=%d text=%q cells=%#v", got, rowText, rowCells)
	}
}
func TestTerminalLiveSurfaceRepliesToOSCBackgroundQuery(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-osc-query",
		Command: []string{"shell"},
		Size:    Size{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-osc-query")
	if process == nil {
		t.Fatal("expected process to be spawned")
	}
	if err := server.IngestOutput(context.Background(), "term-osc-query", "\x1b]11;?\x1b\\"); err != nil {
		t.Fatalf("ingest OSC background query: %v", err)
	}
	assertEventually(t, time.Second, func() bool {
		inputs, _, _, _ := process.snapshot()
		for _, input := range inputs {
			if strings.Contains(string(input), "\x1b]11;") {
				return true
			}
		}
		return false
	}, "expected live terminal query response to be written back to process input")
}

func TestTerminalIngestOutputPublishesLiveInvalidatedEvent(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	events := server.Events(context.Background(), EventFilter{Types: []EventType{EventTerminalLiveInvalidated}})
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-1", "live update\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	event := assertEventValue(t, events, EventTerminalLiveInvalidated, "term-1")
	if event.Live == nil || event.Live.Revision == 0 {
		t.Fatalf("expected live invalidation revision, got %#v", event)
	}
}

func TestTerminalListTracksLastNonEmptyPTYOutput(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-output-time", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-output-time")
	process.emitOutput("agent finished\n")
	assertEventually(t, time.Second, func() bool {
		for _, terminal := range server.ListTerminals() {
			if terminal.ID == "term-output-time" {
				return !terminal.LastOutputAt.IsZero()
			}
		}
		return false
	}, "expected terminal list to expose last PTY output time")
}

func TestTerminalHistoryDisabledUsesNativeScreenOnlyWritePath(t *testing.T) {
	historyDir := t.TempDir()
	server := NewServer(
		WithProcessFactory(newRecordingProcessFactory()),
		WithHistoryStorageDir(historyDir),
		WithHistoryDisabled(),
	)
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-live-only",
		Command: []string{"shell"},
		Size:    Size{Cols: 24, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-live-only", "alpha\r\nbeta\r\nlatest"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(historyDir, "term-live-only.history-lines.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("history disabled must not write payload file, err=%v", err)
	}
	snapshot, err := server.LiveSnapshot("term-live-only")
	if err != nil {
		t.Fatalf("live snapshot: %v", err)
	}
	if got := strings.Join(liveSnapshotRows(snapshot), "\n"); !strings.Contains(got, "latest") {
		t.Fatalf("native screen must still update in history disabled mode, got %q", got)
	}
}

func TestProcessOutputUsesSharedLiveEmulatorAndTypedHistoryDelta(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	events := server.Events(context.Background(), EventFilter{
		TerminalID: "term-r373-fanout",
		Types:      []EventType{EventTerminalLiveInvalidated},
	})
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-fanout",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-r373-fanout")
	if process == nil {
		t.Fatal("expected recording process")
	}
	process.emitOutput("alpha\r\nbeta\r\n")
	event := assertEventValue(t, events, EventTerminalLiveInvalidated, "term-r373-fanout")
	if event.Live == nil || event.Live.Revision == 0 {
		t.Fatalf("expected live invalidation from live SurfaceTrack output, got %#v", event)
	}

	rows, err := server.LiveRows("term-r373-fanout")
	if err != nil {
		t.Fatalf("live rows: %v", err)
	}
	if got := strings.Join(rows, "|"); !strings.Contains(got, "beta") {
		t.Fatalf("process output must update latest native screen, rows=%#v", rows)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r373-fanout", history.HistoryWindowRequest{
		TerminalID: "term-r373-fanout",
		Mode:       history.HistoryWindowModeLatest,
		Cols:       30,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("history.window should flush history semantic backlog: %v", err)
	}
	if got := strings.Join(historyRowTexts(window.Rows), "|"); got != "alpha|beta" {
		t.Fatalf("history queue must consume shared-emulator transaction, got %q window=%#v", got, window)
	}
	terminal, err := server.Terminal("term-r373-fanout")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if terminal.tap != nil {
		t.Fatalf("production path allocated a second semantic parser: %#v", terminal.tap)
	}
	if terminal.currentHistoryDeltaQueue() == nil {
		t.Fatal("history-enabled production path did not install typed delta queue")
	}
}

func TestR396HistoryDisabledProcessOutputKeepsLiveOnlyFastPath(t *testing.T) {
	historyDir := t.TempDir()
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryStorageDir(historyDir),
		WithHistoryDisabled(),
	)
	events := server.Events(context.Background(), EventFilter{
		TerminalID: "term-r373-live-only",
		Types:      []EventType{EventTerminalLiveInvalidated},
	})
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-live-only",
		Command: []string{"shell"},
		Size:    Size{Cols: 24, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-r373-live-only")
	if process == nil {
		t.Fatal("expected recording process")
	}
	process.emitOutput("alpha\r\nlive-only-tail")
	event := assertEventValue(t, events, EventTerminalLiveInvalidated, "term-r373-live-only")
	if event.Live == nil || event.Live.Revision == 0 {
		t.Fatalf("expected live invalidation in history disabled mode, got %#v", event)
	}

	snapshot, err := server.LiveSnapshot("term-r373-live-only")
	if err != nil {
		t.Fatalf("live snapshot: %v", err)
	}
	if got := strings.Join(liveSnapshotRows(snapshot), "\n"); !strings.Contains(got, "live-only-tail") {
		t.Fatalf("history disabled process output must still update live native screen, got %q", got)
	}
	terminal, err := server.Terminal("term-r373-live-only")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	if buffer == nil || buffer.consumers[terminalOutputConsumerHistory].active {
		t.Fatal("history disabled live-only mode must not activate a history output consumer")
	}
	if _, err := server.TerminalHistoryWindow(context.Background(), "term-r373-live-only", history.HistoryWindowRequest{
		TerminalID: "term-r373-live-only",
		Mode:       history.HistoryWindowModeLatest,
		Cols:       24,
		Limit:      10,
	}); !errors.Is(err, ErrHistoryDisabled) {
		t.Fatalf("history disabled window must stay disabled, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(historyDir, "term-r373-live-only.history-lines.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("history disabled process path must not write payload file, err=%v", err)
	}
	if terminal.tap != nil {
		t.Fatalf("history disabled live-only mode must not create history SemanticTap, got %#v", terminal.tap.InputRecords())
	}
}

func TestR396ProcessOutputResponseOwnerRemainsLiveSurfaceOnly(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-response",
		Command: []string{"shell"},
		Size:    Size{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-r373-response")
	if process == nil {
		t.Fatal("expected recording process")
	}
	process.emitOutput("\x1b]11;?\x1b\\")
	assertEventually(t, time.Second, func() bool {
		return terminalProcessResponseCount(process, "\x1b]11;") >= 1
	}, "process PTY query must be answered exactly once by live SurfaceTrack")
	time.Sleep(20 * time.Millisecond)
	if got := terminalProcessResponseCount(process, "\x1b]11;"); got != 1 {
		t.Fatalf("process PTY query must be answered exactly once by live SurfaceTrack, got %d responses", got)
	}
	terminal, err := server.Terminal("term-r373-response")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if terminal.tap != nil {
		t.Fatalf("response-producing output allocated a second semantic parser: %#v", terminal.tap)
	}
}

func TestR373SingleTapSynchronizedTransactionPublishesPrimaryFrame(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-sync-one-tx",
		Command: []string{"shell"},
		Size:    Size{Cols: 20, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r373-sync-one-tx", "\x1b[?2026hline1\r\nline2\r\nline3\x1b[?2026l"); err != nil {
		t.Fatalf("ingest synchronized single transaction: %v", err)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r373-sync-one-tx", history.HistoryWindowRequest{
		TerminalID: "term-r373-sync-one-tx",
		Mode:       history.HistoryWindowModeLatest,
		Cols:       20,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("history.window: %v", err)
	}
	if !historyRowsContainSegment(window.Rows, history.HistorySegmentCurrentPrimaryFrame) {
		t.Fatalf("history semantic sync begin/payload/end transaction must publish current frame, rows=%#v", window.Rows)
	}
	if got := strings.Join(historyRowTexts(window.Rows), "|"); !strings.Contains(got, "line3") {
		t.Fatalf("current frame should preserve final screen payload, got %q", got)
	}
}

func TestR373SingleTapSynchronizedScrollOutKeepsFullPayload(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-sync-scrollout",
		Command: []string{"shell"},
		Size:    Size{Cols: 12, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	var output strings.Builder
	output.WriteString("\x1b[?2026h")
	for i := 1; i <= 8; i++ {
		output.WriteString(fmt.Sprintf("line%02d\r\n", i))
	}
	output.WriteString("\x1b[?2026l")
	if err := server.IngestOutput(context.Background(), "term-r373-sync-scrollout", output.String()); err != nil {
		t.Fatalf("ingest long synchronized transaction: %v", err)
	}
	rows, pageCount := r326CollectAllHistoryRows(t, server, "term-r373-sync-scrollout", 12, 2)
	text := strings.Join(historyRowTexts(rows), "\n")
	for _, want := range []string{"line01", "line08"} {
		if !strings.Contains(text, want) {
			t.Fatalf("single tap synchronized scroll-out must keep full payload, missing %q:\n%s\nrows=%#v", want, text, rows)
		}
	}
	if pageCount < 2 {
		t.Fatalf("long synchronized transaction should page beyond latest frame, page_count=%d rows=%#v", pageCount, rows)
	}
}

func TestR373SplitSynchronizedEndScrollOutKeepsPayload(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-split-sync-end",
		Command: []string{"shell"},
		Size:    Size{Cols: 12, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r373-split-sync-end", "\x1b[?2026hline01\r\n"); err != nil {
		t.Fatalf("ingest synchronized begin: %v", err)
	}
	var output strings.Builder
	for i := 2; i <= 8; i++ {
		output.WriteString(fmt.Sprintf("line%02d\r\n", i))
	}
	output.WriteString("\x1b[?2026l")
	if err := server.IngestOutput(context.Background(), "term-r373-split-sync-end", output.String()); err != nil {
		t.Fatalf("ingest synchronized end flush: %v", err)
	}
	rows, _ := r326CollectAllHistoryRows(t, server, "term-r373-split-sync-end", 12, 2)
	text := strings.Join(historyRowTexts(rows), "\n")
	for _, want := range []string{"line01", "line08"} {
		if !strings.Contains(text, want) {
			t.Fatalf("split sync-end transaction must keep full payload, missing %q:\n%s\nrows=%#v", want, text, rows)
		}
	}
}

func TestR373SingleTapAltTransactionDoesNotEnterPrimaryHistory(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r373-alt-one-tx",
		Command: []string{"shell"},
		Size:    Size{Cols: 20, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r373-alt-one-tx", "before\r\n"); err != nil {
		t.Fatalf("seed primary history: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r373-alt-one-tx", "\x1b[?1049hALT\x1b[?1049l"); err != nil {
		t.Fatalf("ingest alt single transaction: %v", err)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r373-alt-one-tx", history.HistoryWindowRequest{
		TerminalID: "term-r373-alt-one-tx",
		Mode:       history.HistoryWindowModeLatest,
		Cols:       20,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("history.window: %v", err)
	}
	for _, row := range window.Rows {
		if strings.Contains(historyCellsText(row.Cells), "ALT") && row.Segment != history.HistorySegmentCurrentAltFrame {
			t.Fatalf("alt enter/write/exit in one tap transaction must not enter primary history, row=%#v rows=%#v", row, window.Rows)
		}
	}
	if got := historyTextCount(window.Rows, "ALT"); got != 0 {
		t.Fatalf("alt transient already exited; latest authoritative primary history must not contain ALT, count=%d rows=%#v", got, window.Rows)
	}
}

func TestServerNextLiveInvalidationReplaysOnlyWhenObservedRevisionIsBehind(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-live-next", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-live-next", "live update\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	terminal, err := server.Terminal("term-live-next")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	currentRevision := terminal.LiveRevision()
	if currentRevision == 0 {
		t.Fatalf("expected live revision after output")
	}
	event, err := server.NextLiveInvalidation(context.Background(), "term-live-next", currentRevision-1)
	if err != nil {
		t.Fatalf("behind observed revision should get immediate wake: %v", err)
	}
	if event.Live == nil || event.Live.Revision != currentRevision {
		t.Fatalf("unexpected immediate wake %#v current=%#v", event, currentRevision)
	}
	if strings.Contains(fmt.Sprintf("%#v", event.Live), "Snapshot") {
		t.Fatalf("immediate wake must not carry screen payload, got %#v", event.Live)
	}
	snapshot := terminal.NativeScreenSnapshot("term-live-next")
	if snapshot.Revision != currentRevision || !strings.Contains(strings.Join(terminalLiveRowsFromNativeSnapshot(snapshot), "\n"), "live update") {
		t.Fatalf("latest native snapshot should remain pull-based, got %#v", snapshot)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if event, err := server.NextLiveInvalidation(ctx, "term-live-next", currentRevision); err == nil {
		t.Fatalf("observed current revision must wait for a future wake instead of replaying current revision: %#v", event)
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected wait timeout without future invalidation, got %v", err)
	}
}

func TestServerNextLiveInvalidationWaitsForNextWake(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-live-wait", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan Event, 1)
	errs := make(chan error, 1)
	go func() {
		terminal, err := server.Terminal("term-live-wait")
		if err != nil {
			errs <- err
			return
		}
		event, err := server.NextLiveInvalidation(ctx, "term-live-wait", terminal.LiveRevision())
		if err != nil {
			errs <- err
			return
		}
		done <- event
	}()
	select {
	case event := <-done:
		t.Fatalf("one-shot arm returned before next wake: %#v", event)
	case err := <-errs:
		t.Fatalf("one-shot arm failed before next wake: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	if err := server.IngestOutput(context.Background(), "term-live-wait", "live update\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	select {
	case err := <-errs:
		t.Fatalf("one-shot arm failed: %v", err)
	case event := <-done:
		if event.Type != EventTerminalLiveInvalidated || event.TerminalID != "term-live-wait" || event.Live == nil || event.Live.Revision == 0 {
			t.Fatalf("expected next live invalidation, got %#v", event)
		}
		if strings.Contains(fmt.Sprintf("%#v", event.Live), "Snapshot") {
			t.Fatalf("waited wake must not carry screen payload, got %#v", event.Live)
		}
		terminal, err := server.Terminal("term-live-wait")
		if err != nil {
			t.Fatalf("terminal: %v", err)
		}
		snapshot := terminal.NativeScreenSnapshot("term-live-wait")
		if !strings.Contains(strings.Join(terminalLiveRowsFromNativeSnapshot(snapshot), "\n"), "live update") {
			t.Fatalf("latest native snapshot should remain pull-based, got %#v", snapshot)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for live invalidation: %v", ctx.Err())
	}
}

func TestServerDefersLiveInvalidationUntilSynchronizedOutputEnds(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	const terminalID = "term-live-synchronized"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := server.Events(ctx, EventFilter{
		TerminalID: terminalID,
		Types:      []EventType{EventTerminalLiveInvalidated},
	})

	if err := server.IngestOutput(context.Background(), terminalID, "\x1b[?2026h\x1b[4;10H\x1b[?25h"); err != nil {
		t.Fatalf("ingest synchronized begin: %v", err)
	}
	if snapshot := terminal.live.Snapshot(); !snapshot.Modes.SynchronizedOutput {
		t.Fatalf("synchronized mode must remain active after the begin chunk, got %#v", snapshot.Modes)
	}
	select {
	case event := <-events:
		t.Fatalf("synchronized intermediate frame must not wake clients: %#v", event)
	case <-time.After(30 * time.Millisecond):
	}

	if err := server.IngestOutput(context.Background(), terminalID, "\x1b[1;2H\x1b[?2026l"); err != nil {
		t.Fatalf("ingest synchronized end: %v", err)
	}
	select {
	case event := <-events:
		if event.Live == nil || event.Live.Revision != terminal.LiveRevision() {
			t.Fatalf("synchronized end must publish the final revision, event=%#v revision=%d", event, terminal.LiveRevision())
		}
		snapshot := terminal.NativeScreenSnapshot(terminalID)
		if snapshot.Cursor.Row != 0 || snapshot.Cursor.Col != 1 || !snapshot.Cursor.Visible {
			t.Fatalf("published snapshot must expose only the final cursor, got %#v", snapshot.Cursor)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for synchronized end invalidation: %v", ctx.Err())
	}
}

func TestServerNextLiveInvalidationStopsWhenTerminalIsRecreated(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	const terminalID = "term-live-recreated"
	if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	oldTerminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, waitErr := server.NextLiveInvalidation(context.Background(), terminalID, oldTerminal.LiveRevision())
		done <- waitErr
	}()
	time.Sleep(20 * time.Millisecond)

	if err := server.RemoveTerminal(terminalID); err != nil {
		t.Fatalf("remove terminal: %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}}); err != nil {
		t.Fatalf("recreate terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), terminalID, "new terminal"); err != nil {
		t.Fatalf("ingest recreated terminal: %v", err)
	}

	select {
	case waitErr := <-done:
		if !errors.Is(waitErr, ErrTerminalNotFound) {
			t.Fatalf("old long poll error = %v, want terminal not found", waitErr)
		}
	case <-time.After(time.Second):
		t.Fatal("old terminal long poll did not stop after removal")
	}
}

func TestProtocolSessionNextLiveScreenReturnsDeltaFromConfirmedBaseline(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-live-screen-next", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 3}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer session.clearLiveScreenBaselines()
	bootstrap, err := session.ApplicationLiveScreenNext(context.Background(), "term-live-screen-next", 0)
	if err != nil || !bootstrap.FullReplace {
		t.Fatalf("bootstrap screen = %#v err=%v", bootstrap, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan NativeScreenSnapshot, 1)
	errs := make(chan error, 1)
	go func() {
		snapshot, nextErr := session.ApplicationLiveScreenNext(ctx, "term-live-screen-next", bootstrap.Revision)
		if nextErr != nil {
			errs <- nextErr
			return
		}
		done <- snapshot
	}()
	select {
	case snapshot := <-done:
		t.Fatalf("long poll returned before output: %#v", snapshot)
	case nextErr := <-errs:
		t.Fatalf("long poll failed before output: %v", nextErr)
	case <-time.After(20 * time.Millisecond):
	}
	if err := server.IngestOutput(context.Background(), "term-live-screen-next", "latest"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	select {
	case nextErr := <-errs:
		t.Fatalf("long poll failed: %v", nextErr)
	case snapshot := <-done:
		if snapshot.FullReplace || snapshot.BaseRevision != bootstrap.Revision || snapshot.Revision <= bootstrap.Revision {
			t.Fatalf("unexpected next screen metadata: %#v", snapshot)
		}
		if got := strings.Join(terminalLiveRowsFromNativeSnapshot(snapshot), "\n"); !strings.Contains(got, "latest") {
			t.Fatalf("next response did not carry latest rows: %q", got)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for next screen: %v", ctx.Err())
	}
}

func TestServerNextLiveInvalidationCoalescesMissedRevisionsToLatestWake(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-live-coalesce", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	for _, output := range []string{"one\r\n", "two\r\n", "three\r\n"} {
		if err := server.IngestOutput(context.Background(), "term-live-coalesce", output); err != nil {
			t.Fatalf("ingest output: %v", err)
		}
	}
	terminal, err := server.Terminal("term-live-coalesce")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	currentRevision := terminal.LiveRevision()
	if currentRevision < 3 {
		t.Fatalf("expected multiple live revisions, got %d", currentRevision)
	}
	event, err := server.NextLiveInvalidation(context.Background(), "term-live-coalesce", 1)
	if err != nil {
		t.Fatalf("missed revisions should coalesce to latest wake: %v", err)
	}
	if event.Live == nil || event.Live.Revision != currentRevision {
		t.Fatalf("expected latest coalesced revision %d, got %#v", currentRevision, event)
	}
	if strings.Contains(fmt.Sprintf("%#v", event.Live), "Snapshot") {
		t.Fatalf("coalesced wake must not carry screen payload, got %#v", event.Live)
	}
	snapshot := terminal.NativeScreenSnapshot("term-live-coalesce")
	if snapshot.Revision != currentRevision || !strings.Contains(strings.Join(terminalLiveRowsFromNativeSnapshot(snapshot), "\n"), "three") {
		t.Fatalf("latest native snapshot should remain pull-based, got %#v", snapshot)
	}
}

func TestNativeScreenSnapshotSinceCoalescesLatestChangedRows(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-live-delta", Command: []string{"shell"}, Size: Size{Cols: 12, Rows: 3}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	terminal, err := server.Terminal("term-live-delta")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	base, _, err := terminal.applyLiveOutput("zero")
	if err != nil {
		t.Fatalf("write base: %v", err)
	}
	bootstrap, clientBase := terminal.nativeScreenSnapshotSinceBaseline("term-live-delta", 0, nil)
	if !bootstrap.FullReplace || bootstrap.Revision != base || len(bootstrap.Rows) != 3 {
		t.Fatalf("bootstrap must return full native screen: %#v", bootstrap)
	}
	if _, _, err := terminal.applyLiveOutput("\x1b[2;1Hone"); err != nil {
		t.Fatalf("write row one: %v", err)
	}
	latest, _, err := terminal.applyLiveOutput("\x1b[3;1Htwo")
	if err != nil {
		t.Fatalf("write row two: %v", err)
	}

	delta, _ := terminal.nativeScreenSnapshotSinceBaseline("term-live-delta", base, clientBase)
	if delta.FullReplace || delta.BaseRevision != base || delta.Revision != latest {
		t.Fatalf("matching base should return sparse latest snapshot: %#v", delta)
	}
	if len(delta.Rows) != 2 || delta.Rows[0].Index != 1 || delta.Rows[1].Index != 2 {
		t.Fatalf("missed revisions must coalesce changed row indexes, got %#v", delta.Rows)
	}
	if got := terminalLiveRowsFromNativeSnapshot(delta); len(got) != 2 || !strings.Contains(got[0], "one") || !strings.Contains(got[1], "two") {
		t.Fatalf("sparse rows do not contain latest contents: %#v", got)
	}
}

func TestR324TerminalHistoryReturnsAuthoritativeWindow(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-history-r324",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-history-r324", "alpha\r\nbeta\r\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-history-r324", history.HistoryWindowRequest{
		TerminalID: "term-history-r324",
		Mode:       history.HistoryWindowModeLatest,
		Cols:       30,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("history.window should return rebuilt authoritative history: %v", err)
	}
	if got := historyRowTexts(window.Rows); strings.Join(got, "|") != "alpha|beta" {
		t.Fatalf("history.window rows mismatch: %v window=%#v", got, window)
	}
	startLineID, ok := historyLineIDForText(window.Rows, "alpha")
	if !ok {
		t.Fatalf("history.window missing alpha row: %#v", window)
	}
	endLineID, ok := historyLineIDForText(window.Rows, "beta")
	if !ok {
		t.Fatalf("history.window missing beta row: %#v", window)
	}
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-history-r324", history.FreezeHistoryRequest{
		TerminalID: "term-history-r324",
		Cols:       30,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("history.freeze should create token: %v", err)
	}
	text, err := server.TerminalHistoryCopy(context.Background(), "term-history-r324", history.HistoryCopyRequest{
		TerminalID: "term-history-r324",
		Token:      snapshot.Token,
		Range:      &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: startLineID}, End: history.HistoryCopyPosition{LineID: endLineID, Col: 4}},
	})
	if err != nil {
		t.Fatalf("history.copy should use authoritative frozen token: %v", err)
	}
	if text != "alpha\nbeta" {
		t.Fatalf("history.copy mismatch: %q", text)
	}
	if err := server.TerminalHistoryRelease(context.Background(), "term-history-r324", snapshot.Token); err != nil {
		t.Fatalf("history.release should release token: %v", err)
	}
}

func TestR370ProtocolLatestReadsFrozenTokenWithoutSecondFlush(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r370-no-second-flush",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r370-no-second-flush", "alpha\r\nbeta\r\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-r370-no-second-flush", history.FreezeHistoryRequest{
		TerminalID: "term-r370-no-second-flush",
		Cols:       30,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("freeze latest: %v", err)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r370-no-second-flush", history.HistoryWindowRequest{
		TerminalID: "term-r370-no-second-flush",
		Mode:       history.HistoryWindowModeLatest,
		Token:      snapshot.Token,
		Cols:       30,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("frozen latest window should not need a second flush: %v", err)
	}
	if got := historyRowTexts(window.Rows); strings.Join(got, "|") != "alpha|beta" {
		t.Fatalf("unexpected frozen latest rows %v window=%#v", got, window)
	}
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	for _, mode := range []history.HistoryWindowMode{
		history.HistoryWindowModeOlder,
		history.HistoryWindowModeNewer,
		history.HistoryWindowModeOldest,
	} {
		request := history.HistoryWindowRequest{
			TerminalID: "term-r370-no-second-flush",
			Mode:       mode,
			Cols:       30,
			Limit:      10,
		}
		if _, err := server.TerminalHistoryWindow(context.Background(), request.TerminalID, request); !errors.Is(err, history.ErrHistoryInvalidMutation) {
			t.Fatalf("server %s pagination must require a frozen token, got %v", mode, err)
		}
		if _, err := session.ApplicationHistoryWindow(context.Background(), request); !errors.Is(err, history.ErrHistoryInvalidMutation) {
			t.Fatalf("application %s pagination must require a frozen token, got %v", mode, err)
		}
	}
	if _, err := server.TerminalHistoryCopy(context.Background(), "term-r370-no-second-flush", history.HistoryCopyRequest{
		TerminalID: "term-r370-no-second-flush",
	}); !errors.Is(err, history.ErrHistoryInvalidMutation) {
		t.Fatalf("copy must require a frozen token, got %v", err)
	}
}

func TestR360TerminalHistoryOldestReturnsReplaceWindow(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-history-r360-oldest",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-history-r360-oldest", "line-1\r\nline-2\r\nline-3\r\nline-4\r\n"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	snapshot, err := server.TerminalHistoryFreeze(context.Background(), "term-history-r360-oldest", history.FreezeHistoryRequest{
		TerminalID: "term-history-r360-oldest",
		Cols:       30,
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("history.freeze should create token: %v", err)
	}
	oldest, err := server.TerminalHistoryWindow(context.Background(), "term-history-r360-oldest", history.HistoryWindowRequest{
		TerminalID: "term-history-r360-oldest",
		Mode:       history.HistoryWindowModeOldest,
		Token:      snapshot.Token,
		Cols:       30,
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("oldest window: %v", err)
	}
	if oldest.Op != history.HistoryWindowReplace {
		t.Fatalf("oldest must be a replace window, got %s", oldest.Op)
	}
	if got := strings.Join(historyRowTexts(oldest.Rows), "|"); got != "line-1|line-2" {
		t.Fatalf("oldest should return frozen head rows, got %q window=%#v", got, oldest)
	}
}

func TestR436TerminalDefaultsToLineHistoryStore(t *testing.T) {
	historyDir := t.TempDir()
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()), WithHistoryStorageDir(historyDir))
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r436-linehist-default",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 3},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	terminal, err := server.Terminal("term-r436-linehist-default")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if terminal.lineHistory == nil {
		t.Fatal("default terminal history path must own a linehist store")
	}
	if _, ok := terminal.historyStore.(*linehist.Store); !ok {
		t.Fatalf("default terminal history store must be linehist, got %T", terminal.historyStore)
	}
	if err := server.IngestOutput(context.Background(), "term-r436-linehist-default", "alpha\r\nbeta\r\ngamma"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	logicalPath := filepath.Join(historyDir, "term-r436-linehist-default.logical-lines.bin")
	if _, err := os.Stat(logicalPath); err != nil {
		t.Fatalf("R436 default storage dir should write logical lines %s: %v", logicalPath, err)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-r436-linehist-default", history.HistoryWindowRequest{
		TerminalID: "term-r436-linehist-default",
		Mode:       history.HistoryWindowModeLatest,
		Limit:      3,
		Cols:       30,
	})
	if err != nil {
		t.Fatalf("history window: %v", err)
	}
	if got := strings.Join(historyRowTexts(window.Rows), "|"); got != "alpha|beta|gamma" {
		t.Fatalf("linehist terminal history window mismatch: %q rows=%#v", got, window.Rows)
	}
}
func TestR436HistoryStorageDirRecoversLineHistoryRows(t *testing.T) {
	historyDir := t.TempDir()
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory), WithHistoryStorageDir(historyDir))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID:      "term-r436-recover-linehist",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 2},
	}); err != nil {
		t.Fatalf("register first terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-r436-recover-linehist", "alpha\r\nbeta\r\ngamma\r\n"); err != nil {
		t.Fatalf("ingest recover rows: %v", err)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown first server: %v", err)
	}

	recovered := NewServer(WithProcessFactory(newRecordingProcessFactory()), WithHistoryStorageDir(historyDir))
	t.Cleanup(func() { _ = recovered.Shutdown(context.Background()) })
	if _, err := recovered.RegisterTerminal(TerminalRecord{
		ID:      "term-r436-recover-linehist",
		Command: []string{"shell"},
		Size:    Size{Cols: 30, Rows: 2},
	}); err != nil {
		t.Fatalf("register recovered terminal: %v", err)
	}
	window, err := recovered.TerminalHistoryWindow(context.Background(), "term-r436-recover-linehist", history.HistoryWindowRequest{
		TerminalID: "term-r436-recover-linehist",
		Mode:       history.HistoryWindowModeLatest,
		Limit:      20,
		Cols:       30,
	})
	if err != nil {
		t.Fatalf("recovered history window: %v", err)
	}
	if got := strings.Join(historyRowTexts(window.Rows), "|"); got != "alpha|beta|gamma" {
		t.Fatalf("recovered linehist rows mismatch: %q rows=%#v", got, window.Rows)
	}
	for _, row := range window.Rows {
		if !row.Committed || row.Segment != history.HistorySegmentCommitted {
			t.Fatalf("shutdown lifecycle tail must recover as committed cold history, row=%#v", row)
		}
	}
}

func TestR436HistoryStorageDirFailsWhenLineHistoryCannotOpen(t *testing.T) {
	blockingFile := filepath.Join(t.TempDir(), "blocking-file")
	badDir := filepath.Join(blockingFile, "child")
	if err := os.WriteFile(blockingFile, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatalf("seed blocking file: %v", err)
	}
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()), WithHistoryStorageDir(badDir))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-r436-bad-linehist", Command: []string{"shell"}}); err == nil {
		t.Fatal("R436 must fail terminal creation when linehist storage cannot open")
	}
}

func TestR324TerminalHistoryRemoveClosesOpenLine(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-history-remove", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 3}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-history-remove", "partial"); err != nil {
		t.Fatalf("ingest output: %v", err)
	}
	terminal, err := server.Terminal("term-history-remove")
	if err != nil {
		t.Fatalf("terminal handle before remove: %v", err)
	}
	if err := server.RemoveTerminal("term-history-remove"); err != nil {
		t.Fatalf("remove terminal: %v", err)
	}
	window, err := terminal.HistoryWindow(history.HistoryWindowRequest{TerminalID: "term-history-remove", Mode: history.HistoryWindowModeLatest, Cols: 20, Limit: 10})
	if err != nil {
		t.Fatalf("history.window after remove should read closed store: %v", err)
	}
	if got := historyRowTexts(window.Rows); strings.Join(got, "|") != "partial" {
		t.Fatalf("remove close should seal open line before process close, got %v", got)
	}
}

func historyRowTexts(rows []history.HistoryRow) []string {
	raw := rawHistoryRowTexts(rows)
	texts := make([]string, 0, len(raw))
	for index, text := range raw {
		if isLifecycleHistoryRowTextAt(raw, index) {
			continue
		}
		texts = append(texts, text)
	}
	return texts
}

func historyRowsContain(rows []history.HistoryRow, needle string) bool {
	texts := rawHistoryRowTexts(rows)
	for index, text := range texts {
		if isLifecycleHistoryRowTextAt(texts, index) {
			continue
		}
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func historyRowsContainSegment(rows []history.HistoryRow, segment history.HistorySegment) bool {
	for _, row := range rows {
		if row.Segment == segment {
			return true
		}
	}
	return false
}

func historyCellsText(cells []history.Cell) string {
	var out string
	for _, cell := range cells {
		out += cell.Text
	}
	return out
}

func liveSnapshotRows(snapshot live.SurfaceSnapshot) []string {
	rows := make([]string, 0, len(snapshot.Screen.Cells))
	for _, row := range snapshot.Screen.Cells {
		var text string
		for _, cell := range row {
			text += cell.Content
		}
		rows = append(rows, strings.TrimRight(text, " "))
	}
	return rows
}

func terminalLiveRowsFromNativeSnapshot(snapshot NativeScreenSnapshot) []string {
	if len(snapshot.Rows) == 0 {
		return nil
	}
	out := make([]string, len(snapshot.Rows))
	for index, row := range snapshot.Rows {
		out[index] = strings.TrimRight(terminalTestVTermRowText(row.Cells), " ")
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func terminalSnapshotContainsRowText(snapshot NativeScreenSnapshot, want string) bool {
	_, _, ok := terminalSnapshotRowText(snapshot, want)
	return ok
}

func terminalSnapshotRowText(snapshot NativeScreenSnapshot, want string) (string, []vterm.Cell, bool) {
	for _, row := range snapshot.Rows {
		text := strings.TrimRight(terminalTestVTermRowText(row.Cells), " ")
		if text == want {
			return text, row.Cells, true
		}
	}
	return "", nil, false
}

func terminalTestVTermRowText(row []vterm.Cell) string {
	var builder strings.Builder
	for _, cell := range row {
		builder.WriteString(cell.Content)
	}
	return builder.String()
}

func terminalProcessResponseCount(process *recordingProcess, needle string) int {
	inputs, _, _, _ := process.snapshot()
	count := 0
	for _, input := range inputs {
		if strings.Contains(string(input), needle) {
			count++
		}
	}
	return count
}

func TestTerminalRestartReplacesProcessAndClearsExitMetadata(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 4}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-1", "before\n"); err != nil {
		t.Fatalf("ingest before: %v", err)
	}
	first := factory.process("term-1")
	first.exit(0)
	waitForTerminalState(t, server, "term-1", TerminalStateExited)
	if err := server.RestartTerminal(context.Background(), "term-1"); err != nil {
		t.Fatalf("restart: %v", err)
	}
	second := factory.process("term-1")
	if second == nil || second == first {
		t.Fatalf("expected new process, first=%p second=%p", first, second)
	}
	_, _, _, firstClosed := first.snapshot()
	if !firstClosed {
		t.Fatal("restart should close old process")
	}
	info, err := server.GetTerminal("term-1")
	if err != nil {
		t.Fatalf("get after restart: %v", err)
	}
	if info.State != TerminalStateRunning || info.ExitCode != nil || !info.ExitedAt.IsZero() {
		t.Fatalf("restart should clear exit metadata, got %#v", info)
	}
}

func TestTerminalRestartSpawnFailurePreservesCurrentOutputGeneration(t *testing.T) {
	factory := newCancelableOutputProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	defer func() { _ = server.Shutdown(context.Background()) }()
	const terminalID = "term-restart-spawn-failure"
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 20},
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	oldProcess := factory.process(terminalID)
	terminal.queueMu.Lock()
	oldBuffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	if oldBuffer == nil {
		t.Fatal("initial output buffer was not installed")
	}
	publishAndFlush := func(process *cancelableOutputProcess, output string) {
		t.Helper()
		observedRevision := terminal.LiveRevision()
		_, published := process.publish([]byte(output))
		if !receiveOutputTest(t, published, "process output handoff") {
			t.Fatal("current process output was canceled")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := server.NextLiveInvalidation(ctx, terminalID, observedRevision); err != nil {
			t.Fatalf("wait for live output: %v", err)
		}
		if err := terminal.FlushHistory(ctx); err != nil {
			t.Fatalf("flush history output: %v", err)
		}
	}
	publishAndFlush(oldProcess, "before failed restart\r\n")

	spawnErr := errors.New("replacement spawn failed")
	spawnStarted, releaseSpawn := factory.controlNextSpawn(spawnErr)
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseSpawn) })
	restarted := make(chan error, 1)
	go func() {
		restarted <- server.RestartTerminal(context.Background(), terminalID)
	}()
	receiveOutputTest(t, spawnStarted, "replacement spawn barrier")
	publishAndFlush(oldProcess, "while replacement spawn blocked\r\n")
	select {
	case <-oldProcess.cancel:
		t.Fatal("old producer was canceled before replacement spawn completed")
	default:
	}
	if status := oldBuffer.Status(terminalOutputConsumerHistory); status.Closed || status.Unavailable {
		t.Fatalf("old buffer changed while replacement spawn was pending: %#v", status)
	}

	releaseOnce.Do(func() { close(releaseSpawn) })
	if err := receiveOutputTest(t, restarted, "failed restart result"); !errors.Is(err, spawnErr) {
		t.Fatalf("restart error=%v want=%v", err, spawnErr)
	}
	terminal.mu.Lock()
	currentProcess := terminal.process
	terminal.mu.Unlock()
	terminal.queueMu.Lock()
	currentBuffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	if currentProcess != oldProcess || currentBuffer != oldBuffer {
		t.Fatalf("failed restart replaced generation: process=%p/%p buffer=%p/%p", currentProcess, oldProcess, currentBuffer, oldBuffer)
	}
	oldBuffer.mu.Lock()
	liveActive := oldBuffer.consumers[terminalOutputConsumerLive].active
	historyActive := oldBuffer.consumers[terminalOutputConsumerHistory].active
	oldBuffer.mu.Unlock()
	if !liveActive || historyActive {
		t.Fatalf("failed restart changed single-consumer generation: live=%t legacy_history=%t", liveActive, historyActive)
	}
	select {
	case <-oldProcess.cancel:
		t.Fatal("failed restart canceled old producer")
	default:
	}
	publishAndFlush(oldProcess, "after failed restart\r\n")
	info, err := server.GetTerminal(terminalID)
	if err != nil || info.State != TerminalStateRunning {
		t.Fatalf("failed restart changed terminal state: info=%#v err=%v", info, err)
	}
	liveRows, err := server.LiveRows(terminalID)
	if err != nil {
		t.Fatal(err)
	}
	liveText := strings.Join(liveRows, "\n")
	for _, marker := range []string{"before failed restart", "while replacement spawn blocked", "after failed restart"} {
		if !strings.Contains(liveText, marker) {
			t.Fatalf("live output lost %q after failed restart: %q", marker, liveText)
		}
	}
	window, err := server.TerminalHistoryWindow(context.Background(), terminalID, history.HistoryWindowRequest{
		TerminalID: terminalID, Mode: history.HistoryWindowModeLatest, Cols: 80, Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	historyText := strings.Join(historyRowTexts(window.Rows), "\n")
	for _, marker := range []string{"before failed restart", "while replacement spawn blocked", "after failed restart"} {
		if !strings.Contains(historyText, marker) {
			t.Fatalf("history lost %q after failed restart: %q", marker, historyText)
		}
	}

	if err := server.RestartTerminal(context.Background(), terminalID); err != nil {
		t.Fatalf("successful restart: %v", err)
	}
	newProcess := factory.process(terminalID)
	if newProcess == oldProcess {
		t.Fatal("successful restart did not replace process generation")
	}
	receiveOutputTest(t, oldProcess.cancel, "old producer cancellation")
	receiveOutputTest(t, oldProcess.done, "old process shutdown")
	if status := oldBuffer.Status(terminalOutputConsumerHistory); !status.Closed {
		t.Fatalf("successful restart left old buffer open: %#v", status)
	}
	terminal.markExited(oldProcess, ProcessExit{Code: 99})
	info, err = server.GetTerminal(terminalID)
	if err != nil || info.State != TerminalStateRunning || info.ExitCode != nil {
		t.Fatalf("old watchExit changed new generation: info=%#v err=%v", info, err)
	}
	publishAndFlush(newProcess, "new generation output\r\n")
	newLiveRows, err := server.LiveRows(terminalID)
	if err != nil || !strings.Contains(strings.Join(newLiveRows, "\n"), "new generation output") {
		t.Fatalf("new generation output unavailable: rows=%q err=%v", newLiveRows, err)
	}

	publisherDone := newProcess.publishUntilCanceled([]byte("shutdown output\r\n"))
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	receiveOutputTest(t, publisherDone, "new producer shutdown")
	receiveOutputTest(t, newProcess.done, "new process shutdown")
	if resident, _ := server.outputBudget.status(); resident != 0 {
		t.Fatalf("shutdown left output budget resident: %d", resident)
	}
}

func TestTerminalOutputGenerationRejectsPriorProcessCallbacksAfterRestart(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: "term-output-generation", Command: []string{"shell"}, Size: Size{Cols: 40, Rows: 4},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	terminal, err := server.Terminal("term-output-generation")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	oldProcess := factory.process("term-output-generation")
	if err := server.RestartTerminal(context.Background(), "term-output-generation"); err != nil {
		t.Fatalf("restart terminal: %v", err)
	}
	newProcess := factory.process("term-output-generation")
	if newProcess == oldProcess {
		t.Fatal("restart did not create a new process generation")
	}
	if err := terminal.ingestProcessLiveOutput(oldProcess, "OLD-GENERATION-LIVE"); err != nil {
		t.Fatalf("stale live callback: %v", err)
	}
	if err := terminal.ingestProcessHistoryTapOutput(oldProcess, "OLD-GENERATION-HISTORY\r\n"); err != nil {
		t.Fatalf("stale history callback: %v", err)
	}
	if err := server.IngestOutput(context.Background(), "term-output-generation", "NEW-GENERATION"); err != nil {
		t.Fatalf("new generation output: %v", err)
	}
	rows, err := server.LiveRows("term-output-generation")
	if err != nil {
		t.Fatalf("live rows: %v", err)
	}
	joined := strings.Join(rows, "\n")
	if strings.Contains(joined, "OLD-GENERATION") || !strings.Contains(joined, "NEW-GENERATION") {
		t.Fatalf("generation isolation failed: %q", joined)
	}
	window, err := server.TerminalHistoryWindow(context.Background(), "term-output-generation", history.HistoryWindowRequest{
		TerminalID: "term-output-generation", Mode: history.HistoryWindowModeLatest, Cols: 40, Limit: 100,
	})
	if err != nil {
		t.Fatalf("history window: %v", err)
	}
	var historyText strings.Builder
	for _, row := range window.Rows {
		for _, cell := range row.Cells {
			historyText.WriteString(cell.Text)
		}
		historyText.WriteByte('\n')
	}
	if got := historyText.String(); strings.Contains(got, "OLD-GENERATION") {
		t.Fatalf("prior process entered new history parser: %q", got)
	}
}

func TestTerminalHistoryIngestFailureIsObservableAndReleasesOutputCapacity(t *testing.T) {
	wantErr := errors.New("line storage write failed")
	storage := &failingTerminalLineStorage{failAfter: 1, err: wantErr}
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithTerminalOutputBufferConfig(TerminalOutputBufferConfig{
			CapacityBytes: MinTerminalOutputBufferCapacityBytes,
			Overflow:      TerminalOutputOverflowBlock,
		}),
		WithTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes),
		WithHistoryStoreFactory(func(id string) (history.HistoryStore, error) {
			return linehist.NewStore(id, linehist.NewEngine(storage)), nil
		}),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: "term-history-output-failure", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 2},
	}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	terminal, err := server.Terminal("term-history-output-failure")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	observed := terminal.LiveRevision()
	factory.process("term-history-output-failure").emitOutput("one\r\ntwo\r\nthree\r\nfour\r\n")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := server.NextLiveInvalidation(ctx, "term-history-output-failure", observed); err != nil {
		t.Fatalf("wait for live output: %v", err)
	}
	_, err = server.TerminalHistoryWindow(ctx, "term-history-output-failure", history.HistoryWindowRequest{
		TerminalID: "term-history-output-failure", Mode: history.HistoryWindowModeLatest, Cols: 20, Limit: 20,
	})
	if !errors.Is(err, wantErr) || !errors.Is(err, ErrTerminalOutputUnavailable) {
		t.Fatalf("history failure was not returned as typed unavailable error: %v", err)
	}
	status, err := server.TerminalHistoryBacklogStatus("term-history-output-failure")
	if err != nil {
		t.Fatalf("history status: %v", err)
	}
	if !status.Unavailable || status.ResidentBytes != 0 || status.AggregateResidentBytes != 0 {
		t.Fatalf("history failure did not release buffer capacity: %#v", status)
	}
	info, err := server.GetTerminal("term-history-output-failure")
	if err != nil || info.State != TerminalStateRunning {
		t.Fatalf("history failure changed healthy process lifecycle: info=%#v err=%v", info, err)
	}
}

func TestTerminalHistoryIngestFailurePersistsGapBeforeRestart(t *testing.T) {
	applyErr := errors.New("fail one history transaction")
	file, err := linehist.OpenCompressedLineFile(t.TempDir(), "term-history-gap-restart", linehist.CompressedLineFileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	storage := &controlledTerminalLineStorage{file: file}
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryStoreFactory(func(id string) (history.HistoryStore, error) {
			return linehist.NewStore(id, linehist.NewEngine(storage)), nil
		}),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: "term-history-gap-restart", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 2},
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal("term-history-gap-restart")
	if err != nil {
		t.Fatal(err)
	}
	storage.failNextAppend(applyErr)
	factory.process("term-history-gap-restart").emitOutput("one\r\ntwo\r\nthree\r\nfour\r\n")
	assertEventually(t, 2*time.Second, func() bool {
		return errors.Is(terminal.FlushHistory(context.Background()), applyErr)
	}, "history transaction failure was not observable")
	if gaps := storage.GapOffsets(); len(gaps) != 1 {
		t.Fatalf("history failure did not persist exactly one gap: %v", gaps)
	}

	if err := server.RestartTerminal(context.Background(), "term-history-gap-restart"); err != nil {
		t.Fatal(err)
	}
	_, err = server.TerminalHistoryWindow(context.Background(), "term-history-gap-restart", history.HistoryWindowRequest{
		Mode: history.HistoryWindowModeLatest, Cols: 20, Limit: 100,
	})
	if !errors.Is(err, history.ErrHistorySyncLost) {
		t.Fatalf("restart allowed a history query to cross the persisted gap: %v", err)
	}
	latest, err := server.TerminalHistoryWindow(context.Background(), "term-history-gap-restart", history.HistoryWindowRequest{
		Mode: history.HistoryWindowModeLatest, Cols: 20, Limit: 1,
	})
	if err != nil || len(latest.Rows) != 1 {
		t.Fatalf("one-sided post-gap history must remain readable: window=%#v err=%v", latest, err)
	}
}

func TestTerminalHistoryGapPersistenceFailureStaysStickyAcrossRestart(t *testing.T) {
	applyErr := errors.New("fail history transaction")
	gapErr := errors.New("fail persistent history gap")
	file, err := linehist.OpenCompressedLineFile(t.TempDir(), "term-history-gap-sticky", linehist.CompressedLineFileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	storage := &controlledTerminalLineStorage{file: file, gapErr: gapErr}
	factory := newRecordingProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryStoreFactory(func(id string) (history.HistoryStore, error) {
			return linehist.NewStore(id, linehist.NewEngine(storage)), nil
		}),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: "term-history-gap-sticky", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 2},
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal("term-history-gap-sticky")
	if err != nil {
		t.Fatal(err)
	}
	storage.failNextAppend(applyErr)
	factory.process("term-history-gap-sticky").emitOutput("one\r\ntwo\r\nthree\r\nfour\r\n")
	assertEventually(t, 2*time.Second, func() bool {
		err := terminal.FlushHistory(context.Background())
		return errors.Is(err, applyErr) && errors.Is(err, gapErr)
	}, "gap persistence failure did not become sticky")
	if gaps := storage.GapOffsets(); len(gaps) != 0 {
		t.Fatalf("failed gap write unexpectedly changed durable offsets: %v", gaps)
	}

	if err := server.RestartTerminal(context.Background(), "term-history-gap-sticky"); err != nil {
		t.Fatal(err)
	}
	_, err = server.TerminalHistoryWindow(context.Background(), "term-history-gap-sticky", history.HistoryWindowRequest{
		Mode: history.HistoryWindowModeLatest, Cols: 20, Limit: 1,
	})
	if !errors.Is(err, applyErr) || !errors.Is(err, gapErr) || !errors.Is(err, ErrTerminalOutputUnavailable) {
		t.Fatalf("restart cleared sticky history failure: %v", err)
	}
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	if buffer == nil {
		t.Fatal("restart did not install a live output buffer")
	}
	buffer.mu.Lock()
	historyActive := buffer.consumers[terminalOutputConsumerHistory].active
	buffer.mu.Unlock()
	if historyActive {
		t.Fatal("history consumer restarted after an unpersisted output gap")
	}
}

func TestTerminalCloseCancelsPublisherBlockedBehindOutputBuffer(t *testing.T) {
	factory := newCancelableOutputProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryDisabled(),
		WithTerminalOutputBufferConfig(TerminalOutputBufferConfig{
			CapacityBytes: MinTerminalOutputBufferCapacityBytes,
			Overflow:      TerminalOutputOverflowBlock,
		}),
		WithTerminalOutputResidentBudget(MinTerminalOutputBufferCapacityBytes),
	)
	if _, err := server.RegisterTerminal(TerminalRecord{
		ID: "term-output-cancel", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 2},
	}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal("term-output-cancel")
	if err != nil {
		t.Fatal(err)
	}
	process := factory.process("term-output-cancel")
	terminal.liveOpMu.Lock()
	locked := true
	defer func() {
		if locked {
			terminal.liveOpMu.Unlock()
		}
	}()

	firstStarted, firstDone := process.publish(bytes.Repeat([]byte("x"), int(MinTerminalOutputBufferCapacityBytes)))
	<-firstStarted
	if !receiveOutputTest(t, firstDone, "first unbuffered publish") {
		t.Fatal("first publish was canceled")
	}
	secondStarted, secondDone := process.publish([]byte("blocked-write"))
	<-secondStarted
	if !receiveOutputTest(t, secondDone, "second unbuffered publish") {
		t.Fatal("second publish was canceled before reaching the watcher")
	}
	terminal.queueMu.Lock()
	buffer := terminal.outputBuffer
	terminal.queueMu.Unlock()
	waitForOutputCondition(t, func() bool {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return buffer.localWaiters == 1
	}, "watcher blocked on local output capacity")
	thirdStarted, thirdDone := process.publish([]byte("blocked-publisher"))
	<-thirdStarted

	closed := make(chan error, 1)
	go func() { closed <- terminal.Close() }()
	if receiveOutputTest(t, thirdDone, "publisher cancellation") {
		t.Fatal("publisher unexpectedly handed off output after terminal close")
	}
	terminal.liveOpMu.Unlock()
	locked = false
	if err := receiveOutputTest(t, closed, "terminal close"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-process.done:
	case <-time.After(2 * time.Second):
		t.Fatal("process wait lifecycle did not finish")
	}
}

func TestTerminalRepeatedRestartAndCloseStopAllOutputGoroutines(t *testing.T) {
	factory := newCancelableOutputProcessFactory()
	server := NewServer(WithProcessFactory(factory), WithHistoryDisabled())
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-output-restart", Command: []string{"shell"}}); err != nil {
		t.Fatal(err)
	}
	const restarts = 20
	for i := 0; i < restarts; i++ {
		old := factory.process("term-output-restart")
		publisherDone := old.publishUntilCanceled([]byte("continuous-output"))
		if err := server.RestartTerminal(context.Background(), "term-output-restart"); err != nil {
			t.Fatalf("restart %d: %v", i, err)
		}
		select {
		case <-publisherDone:
		case <-time.After(2 * time.Second):
			t.Fatalf("restart %d left publisher blocked", i)
		}
		select {
		case <-old.done:
		case <-time.After(2 * time.Second):
			t.Fatalf("restart %d left process wait goroutine blocked", i)
		}
	}
	last := factory.process("term-output-restart")
	publisherDone := last.publishUntilCanceled([]byte("final-output"))
	terminal, err := server.Terminal("term-output-restart")
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-publisherDone:
	case <-time.After(2 * time.Second):
		t.Fatal("terminal close left final publisher blocked")
	}
	select {
	case <-last.done:
	case <-time.After(2 * time.Second):
		t.Fatal("terminal close left final wait goroutine blocked")
	}
}

func TestTerminalOutputGenerationClosesBeforeDiagnosticsAreCached(t *testing.T) {
	t.Run("normal output end", func(t *testing.T) {
		factory := newRecordingProcessFactory()
		server := NewServer(WithProcessFactory(factory))
		t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
		if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-output-end", Command: []string{"shell"}}); err != nil {
			t.Fatal(err)
		}
		factory.process("term-output-end").exit(0)
		waitForTerminalState(t, server, "term-output-end", TerminalStateExited)
		status, err := server.TerminalHistoryBacklogStatus("term-output-end")
		if err != nil || !status.Closed {
			t.Fatalf("normal generation end cached open diagnostics: status=%#v err=%v", status, err)
		}
	})

	t.Run("terminal close", func(t *testing.T) {
		factory := newRecordingProcessFactory()
		server := NewServer(WithProcessFactory(factory))
		terminalInfo, err := server.RegisterTerminal(TerminalRecord{ID: "term-output-close", Command: []string{"shell"}})
		if err != nil {
			t.Fatal(err)
		}
		terminal, err := server.Terminal(terminalInfo.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := terminal.Close(); err != nil {
			t.Fatal(err)
		}
		if status := terminal.HistoryBacklogStatus(); !status.Closed {
			t.Fatalf("terminal close cached open diagnostics: %#v", status)
		}
	})

	t.Run("restart", func(t *testing.T) {
		factory := newRecordingProcessFactory()
		server := NewServer(WithProcessFactory(factory))
		t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
		if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-output-restart-closed", Command: []string{"shell"}}); err != nil {
			t.Fatal(err)
		}
		terminal, err := server.Terminal("term-output-restart-closed")
		if err != nil {
			t.Fatal(err)
		}
		terminal.queueMu.Lock()
		oldBuffer := terminal.outputBuffer
		terminal.queueMu.Unlock()
		if err := server.RestartTerminal(context.Background(), "term-output-restart-closed"); err != nil {
			t.Fatal(err)
		}
		if status := oldBuffer.Status(terminalOutputConsumerHistory); !status.Closed {
			t.Fatalf("restart replaced generation before closing diagnostics: %#v", status)
		}
	})
}

func TestTerminalOutputMinimumCapacityAggregateBudgetHasNoProcessPrequeue(t *testing.T) {
	factory := newCancelableOutputProcessFactory()
	server := NewServer(
		WithProcessFactory(factory),
		WithHistoryDisabled(),
		WithTerminalOutputBufferConfig(TerminalOutputBufferConfig{
			CapacityBytes: MinTerminalOutputBufferCapacityBytes,
			Overflow:      TerminalOutputOverflowBlock,
		}),
		WithTerminalOutputResidentBudget(MinTerminalOutputResidentBudgetBytes),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	for _, id := range []string{"term-aggregate-a", "term-aggregate-b"} {
		if _, err := server.RegisterTerminal(TerminalRecord{ID: id, Command: []string{"shell"}}); err != nil {
			t.Fatal(err)
		}
	}
	firstTerminal, _ := server.Terminal("term-aggregate-a")
	secondTerminal, _ := server.Terminal("term-aggregate-b")
	firstProcess := factory.process("term-aggregate-a")
	secondProcess := factory.process("term-aggregate-b")
	if cap(firstProcess.output) != 0 || cap(secondProcess.output) != 0 {
		t.Fatal("terminal processes added a queue outside the shared output budget")
	}
	firstTerminal.liveOpMu.Lock()
	firstLocked := true
	secondTerminal.liveOpMu.Lock()
	secondLocked := true
	defer func() {
		if firstLocked {
			firstTerminal.liveOpMu.Unlock()
		}
		if secondLocked {
			secondTerminal.liveOpMu.Unlock()
		}
	}()

	_, firstDone := firstProcess.publish(bytes.Repeat([]byte("a"), int(MinTerminalOutputBufferCapacityBytes)))
	if !receiveOutputTest(t, firstDone, "first terminal publish") {
		t.Fatal("first terminal publish was canceled")
	}
	firstTerminal.queueMu.Lock()
	firstBuffer := firstTerminal.outputBuffer
	firstTerminal.queueMu.Unlock()
	waitForOutputCondition(t, func() bool {
		return firstBuffer.Status(terminalOutputConsumerLive).ResidentBytes == MinTerminalOutputBufferCapacityBytes
	}, "first terminal fills aggregate budget")

	_, blockedDone := secondProcess.publish([]byte("b"))
	if !receiveOutputTest(t, blockedDone, "second terminal aggregate handoff") {
		t.Fatal("second terminal handoff was canceled")
	}
	waitForOutputCondition(t, func() bool {
		server.outputBudget.mu.Lock()
		defer server.outputBudget.mu.Unlock()
		return server.outputBudget.waiters == 1
	}, "second terminal aggregate waiter")
	secondStarted, secondDone := secondProcess.publish([]byte("next"))
	<-secondStarted
	resident, limit := server.outputBudget.status()
	if resident != MinTerminalOutputResidentBudgetBytes || resident > limit {
		t.Fatalf("aggregate budget exceeded: resident=%d limit=%d", resident, limit)
	}

	firstTerminal.liveOpMu.Unlock()
	firstLocked = false
	if !receiveOutputTest(t, secondDone, "second terminal resumed publisher") {
		t.Fatal("second terminal publisher was canceled")
	}
	secondTerminal.liveOpMu.Unlock()
	secondLocked = false
	assertEventually(t, 2*time.Second, func() bool {
		resident, _ := server.outputBudget.status()
		return resident == 0
	}, "aggregate resident bytes did not drain")
}

func TestTerminalKillAndRemoveCloseProcess(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"shell"}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	process := factory.process("term-1")
	if err := server.KillTerminal(context.Background(), "term-1"); err != nil {
		t.Fatalf("kill terminal: %v", err)
	}
	_, _, killed, _ := process.snapshot()
	if !killed {
		t.Fatal("expected process to be killed")
	}
	if err := server.RemoveTerminal("term-1"); err != nil {
		t.Fatalf("remove terminal: %v", err)
	}
	_, _, _, closed := process.snapshot()
	if !closed {
		t.Fatal("expected process to be closed")
	}
	if _, err := server.Terminal("term-1"); !errors.Is(err, ErrTerminalNotFound) {
		t.Fatalf("expected ErrTerminalNotFound, got %v", err)
	}
}

func TestServerShutdownRejectsLaterTerminalRegistration(t *testing.T) {
	factory := newRecordingProcessFactory()
	server := NewServer(WithProcessFactory(factory))
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-1", Command: []string{"shell"}}); !errors.Is(err, ErrServerClosed) {
		t.Fatalf("expected ErrServerClosed, got %v", err)
	}
	if process := factory.process("term-1"); process != nil {
		t.Fatal("shutdown server must not spawn process for later registration")
	}
}

type recordingProcessFactory struct {
	mu        sync.Mutex
	processes map[string][]*recordingProcess
	specs     map[string][]ProcessSpec
}

type cancelableOutputProcessFactory struct {
	mu        sync.Mutex
	processes map[string][]*cancelableOutputProcess
	nextSpawn *controlledProcessSpawn
}

type controlledProcessSpawn struct {
	started chan struct{}
	release chan struct{}
	err     error
}

func newCancelableOutputProcessFactory() *cancelableOutputProcessFactory {
	return &cancelableOutputProcessFactory{processes: make(map[string][]*cancelableOutputProcess)}
}

func (factory *cancelableOutputProcessFactory) Spawn(_ context.Context, spec ProcessSpec) (TerminalProcess, error) {
	factory.mu.Lock()
	controlled := factory.nextSpawn
	factory.nextSpawn = nil
	factory.mu.Unlock()
	if controlled != nil {
		close(controlled.started)
		<-controlled.release
		if controlled.err != nil {
			return nil, controlled.err
		}
	}
	process := &cancelableOutputProcess{
		output: make(chan []byte), cancel: make(chan struct{}),
		wait: make(chan ProcessExit, 1), done: make(chan struct{}),
	}
	factory.mu.Lock()
	factory.processes[spec.TerminalID] = append(factory.processes[spec.TerminalID], process)
	factory.mu.Unlock()
	return process, nil
}

func (factory *cancelableOutputProcessFactory) controlNextSpawn(err error) (<-chan struct{}, chan struct{}) {
	controlled := &controlledProcessSpawn{started: make(chan struct{}), release: make(chan struct{}), err: err}
	factory.mu.Lock()
	if factory.nextSpawn != nil {
		factory.mu.Unlock()
		panic("process spawn is already controlled")
	}
	factory.nextSpawn = controlled
	factory.mu.Unlock()
	return controlled.started, controlled.release
}

func (factory *cancelableOutputProcessFactory) process(id string) *cancelableOutputProcess {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	processes := factory.processes[id]
	return processes[len(processes)-1]
}

type cancelableOutputProcess struct {
	output     chan []byte
	cancel     chan struct{}
	wait       chan ProcessExit
	done       chan struct{}
	cancelOnce sync.Once
	finishOnce sync.Once
}

func (process *cancelableOutputProcess) Input([]byte) error { return nil }
func (process *cancelableOutputProcess) Resize(Size) error  { return nil }
func (process *cancelableOutputProcess) Output() <-chan []byte {
	return process.output
}
func (process *cancelableOutputProcess) CancelOutput() {
	process.cancelOnce.Do(func() { close(process.cancel) })
}
func (process *cancelableOutputProcess) Kill() error {
	process.finish(-1)
	return nil
}
func (process *cancelableOutputProcess) Wait() <-chan ProcessExit { return process.wait }
func (process *cancelableOutputProcess) Close() error {
	process.finish(-1)
	return nil
}

func (process *cancelableOutputProcess) publish(payload []byte) (<-chan struct{}, <-chan bool) {
	started := make(chan struct{})
	done := make(chan bool, 1)
	go func() {
		close(started)
		select {
		case process.output <- append([]byte(nil), payload...):
			done <- true
		case <-process.cancel:
			done <- false
		}
		close(done)
	}()
	return started, done
}

func (process *cancelableOutputProcess) publishUntilCanceled(payload []byte) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case process.output <- append([]byte(nil), payload...):
			case <-process.cancel:
				return
			}
		}
	}()
	return done
}

func (process *cancelableOutputProcess) finish(code int) {
	process.finishOnce.Do(func() {
		process.CancelOutput()
		process.wait <- ProcessExit{Code: code}
		close(process.wait)
		close(process.done)
	})
}

type failingTerminalLineStorage struct {
	mu        sync.Mutex
	lines     []linehist.Line
	appends   int
	failAfter int
	err       error
}

func (storage *failingTerminalLineStorage) AppendLines(lines []linehist.Line) error {
	if len(lines) == 0 {
		return nil
	}
	storage.mu.Lock()
	defer storage.mu.Unlock()
	if storage.appends >= storage.failAfter {
		return storage.err
	}
	storage.appends++
	storage.lines = append(storage.lines, lines...)
	return nil
}

func (storage *failingTerminalLineStorage) AppendBoundary() error { return nil }

func (storage *failingTerminalLineStorage) LineCount() int {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return len(storage.lines)
}

func (storage *failingTerminalLineStorage) Base() int { return 0 }

func (storage *failingTerminalLineStorage) Lines(start int, end int) ([]linehist.Line, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return append([]linehist.Line(nil), storage.lines[start:end]...), nil
}

func (storage *failingTerminalLineStorage) Sync() error { return nil }

func (storage *failingTerminalLineStorage) Close() error { return nil }

type controlledTerminalLineStorage struct {
	mu        sync.Mutex
	file      *linehist.CompressedLineFile
	appendErr error
	gapErr    error
}

func (storage *controlledTerminalLineStorage) failNextAppend(err error) {
	storage.mu.Lock()
	storage.appendErr = err
	storage.mu.Unlock()
}

func (storage *controlledTerminalLineStorage) AppendLines(lines []linehist.Line) error {
	storage.mu.Lock()
	err := storage.appendErr
	storage.appendErr = nil
	storage.mu.Unlock()
	if err != nil {
		return err
	}
	return storage.file.AppendLines(lines)
}

func (storage *controlledTerminalLineStorage) AppendBoundary() error {
	return storage.file.AppendBoundary()
}

func (storage *controlledTerminalLineStorage) AppendGap() error {
	storage.mu.Lock()
	err := storage.gapErr
	storage.mu.Unlock()
	if err != nil {
		return err
	}
	return storage.file.AppendGap()
}

func (storage *controlledTerminalLineStorage) GapOffsets() []int {
	return storage.file.GapOffsets()
}

func (storage *controlledTerminalLineStorage) LineCount() int { return storage.file.LineCount() }
func (storage *controlledTerminalLineStorage) Base() int      { return storage.file.Base() }
func (storage *controlledTerminalLineStorage) Lines(start int, end int) ([]linehist.Line, error) {
	return storage.file.Lines(start, end)
}
func (storage *controlledTerminalLineStorage) Sync() error  { return storage.file.Sync() }
func (storage *controlledTerminalLineStorage) Close() error { return storage.file.Close() }

func newRecordingProcessFactory() *recordingProcessFactory {
	return &recordingProcessFactory{processes: make(map[string][]*recordingProcess), specs: make(map[string][]ProcessSpec)}
}

func (factory *recordingProcessFactory) Spawn(_ context.Context, spec ProcessSpec) (TerminalProcess, error) {
	process := &recordingProcess{
		id:       spec.TerminalID,
		outputCh: make(chan []byte, 16),
		waitCh:   make(chan ProcessExit, 1),
	}
	factory.mu.Lock()
	factory.processes[spec.TerminalID] = append(factory.processes[spec.TerminalID], process)
	factory.specs[spec.TerminalID] = append(factory.specs[spec.TerminalID], cloneProcessSpec(spec))
	factory.mu.Unlock()
	return process, nil
}

func (factory *recordingProcessFactory) process(id string) *recordingProcess {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	processes := factory.processes[id]
	if len(processes) == 0 {
		return nil
	}
	return processes[len(processes)-1]
}

func (factory *recordingProcessFactory) spawnedSpecs(id string) []ProcessSpec {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	specs := factory.specs[id]
	out := make([]ProcessSpec, 0, len(specs))
	for _, spec := range specs {
		out = append(out, cloneProcessSpec(spec))
	}
	return out
}

type recordingProcess struct {
	mu         sync.Mutex
	id         string
	inputs     [][]byte
	resizes    []Size
	resizeErr  error
	resizeHook func(Size)
	closeHook  func()
	outputCh   chan []byte
	waitCh     chan ProcessExit
	exitOnce   sync.Once
	outputOnce sync.Once
	killed     bool
	closed     bool
	closeErr   error
}

func (process *recordingProcess) Input(data []byte) error {
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.closed {
		return io.ErrClosedPipe
	}
	process.inputs = append(process.inputs, append([]byte(nil), data...))
	return nil
}

func (process *recordingProcess) Resize(size Size) error {
	process.mu.Lock()
	if process.closed {
		process.mu.Unlock()
		return io.ErrClosedPipe
	}
	if process.resizeErr != nil {
		err := process.resizeErr
		process.mu.Unlock()
		return err
	}
	process.resizes = append(process.resizes, size)
	hook := process.resizeHook
	process.mu.Unlock()
	if hook != nil {
		hook(size)
	}
	return nil
}

func (process *recordingProcess) setResizeHook(hook func(Size)) {
	process.mu.Lock()
	defer process.mu.Unlock()
	process.resizeHook = hook
}

func (process *recordingProcess) Output() <-chan []byte {
	return process.outputCh
}

func (process *recordingProcess) CancelOutput() {
	process.closeOutput()
}

func (process *recordingProcess) emitOutput(output string) {
	process.outputCh <- []byte(output)
}

func (process *recordingProcess) Kill() error {
	process.mu.Lock()
	process.killed = true
	process.mu.Unlock()
	process.exit(-1)
	return nil
}

func (process *recordingProcess) Wait() <-chan ProcessExit {
	return process.waitCh
}

func (process *recordingProcess) Close() error {
	process.mu.Lock()
	process.closed = true
	err := process.closeErr
	hook := process.closeHook
	process.mu.Unlock()
	if hook != nil {
		hook()
	}
	process.closeOutput()
	process.exit(-1)
	return err
}

func (process *recordingProcess) setCloseError(err error) {
	process.mu.Lock()
	process.closeErr = err
	process.mu.Unlock()
}

func (process *recordingProcess) setCloseHook(hook func()) {
	process.mu.Lock()
	process.closeHook = hook
	process.mu.Unlock()
}

func (process *recordingProcess) snapshot() ([][]byte, []Size, bool, bool) {
	process.mu.Lock()
	defer process.mu.Unlock()
	inputs := make([][]byte, len(process.inputs))
	for i, input := range process.inputs {
		inputs[i] = append([]byte(nil), input...)
	}
	resizes := append([]Size(nil), process.resizes...)
	return inputs, resizes, process.killed, process.closed
}

func (process *recordingProcess) exit(code int) {
	process.exitOnce.Do(func() {
		process.closeOutput()
		process.waitCh <- ProcessExit{Code: code}
		close(process.waitCh)
	})
}

func (process *recordingProcess) closeOutput() {
	process.outputOnce.Do(func() {
		close(process.outputCh)
	})
}

func assertEventually(t *testing.T, timeout time.Duration, check func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(message)
}
