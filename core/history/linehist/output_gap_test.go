package linehist

import (
	"errors"
	"sync"
	"testing"

	"github.com/anytty/anytty/core/history"
)

type outputEpochTestGate struct {
	sync.Mutex
	held bool
}

func (gate *outputEpochTestGate) Lock() {
	gate.Mutex.Lock()
	gate.held = true
}

func (gate *outputEpochTestGate) Unlock() {
	gate.held = false
	gate.Mutex.Unlock()
}

func TestOutputGapPersistsAndRejectsCrossEpochHistory(t *testing.T) {
	dir := t.TempDir()
	file, err := OpenCompressedLineFile(dir, "gap-terminal", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore("gap-terminal", NewEngine(file))
	if err := store.AppendLifecycleLines([]string{"before"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendGapBoundary(); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendLifecycleLines([]string{"after"}); err != nil {
		t.Fatal(err)
	}
	latest, err := store.LatestWindow(history.HistoryWindowRequest{Mode: history.HistoryWindowModeLatest, Limit: 1})
	if err != nil || len(latest.Rows) != 1 || rowText(latest.Rows[0].Cells) != "after" {
		t.Fatalf("post-gap window failed: window=%#v err=%v", latest, err)
	}
	if _, err := store.LatestWindow(history.HistoryWindowRequest{Mode: history.HistoryWindowModeLatest, Limit: 2}); !errors.Is(err, history.ErrHistorySyncLost) {
		t.Fatalf("cross-gap window error=%v", err)
	} else {
		var gapErr *history.SyncGapError
		if !errors.As(err, &gapErr) || gapErr.GapAfterLine != 1 {
			t.Fatalf("cross-gap error lost durable boundary: %v", err)
		}
	}
	before, err := store.Copy(history.HistoryCopyRequest{
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: 1}, End: history.HistoryCopyPosition{LineID: 1, Col: 6}},
	})
	if err != nil || before != "before" {
		t.Fatalf("pre-gap copy failed: text=%q err=%v", before, err)
	}
	after, err := store.Copy(history.HistoryCopyRequest{
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: 2}, End: history.HistoryCopyPosition{LineID: 2, Col: 5}},
	})
	if err != nil || after != "after" {
		t.Fatalf("post-gap copy failed: text=%q err=%v", after, err)
	}
	if _, err := store.Copy(history.HistoryCopyRequest{
		Range: &history.HistoryCopyRange{Start: history.HistoryCopyPosition{LineID: 1}, End: history.HistoryCopyPosition{LineID: 2, Col: 5}},
	}); !errors.Is(err, history.ErrHistorySyncLost) {
		t.Fatalf("cross-gap copy error=%v", err)
	}
	if _, err := store.Freeze(history.FreezeHistoryRequest{Limit: 10}); !errors.Is(err, history.ErrHistorySyncLost) {
		t.Fatalf("cross-gap freeze error=%v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenCompressedLineFile(dir, "gap-terminal", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	recovered := NewStore("gap-terminal", NewEngine(reopened))
	if gaps := reopened.GapOffsets(); len(gaps) != 1 || gaps[0] != 1 {
		t.Fatalf("recovered gap offsets=%v", gaps)
	}
	if _, err := recovered.LatestWindow(history.HistoryWindowRequest{Mode: history.HistoryWindowModeLatest, Limit: 2}); !errors.Is(err, history.ErrHistorySyncLost) {
		t.Fatalf("recovered cross-gap window error=%v", err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOutputGapCoalescesAtSameLogicalBoundary(t *testing.T) {
	file, err := OpenCompressedLineFile(t.TempDir(), "coalesced-gap", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := file.AppendLines([]Line{{Runs: []Run{{Text: "before"}}, HardEnd: true}}); err != nil {
		t.Fatal(err)
	}
	if err := file.AppendGap(); err != nil {
		t.Fatal(err)
	}
	if err := file.AppendGap(); err != nil {
		t.Fatal(err)
	}
	if gaps := file.GapOffsets(); len(gaps) != 1 || gaps[0] != 1 {
		t.Fatalf("same-offset gap transitions were not coalesced: %v", gaps)
	}
}

func TestTransitionOutputEpochPersistsGapAndResetsParserUnderGate(t *testing.T) {
	file, err := OpenCompressedLineFile(t.TempDir(), "epoch-transition", CompressedLineFileOptions{Compression: compressionNone})
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	store := NewStore("epoch-transition", NewEngine(file))
	gate := &outputEpochTestGate{}
	store.Bind(func() ScreenSnapshot {
		if !gate.held {
			t.Fatal("screen snapshot captured outside ingest gate")
		}
		return ScreenSnapshot{}
	}, gate)
	if err := store.AppendLifecycleLines([]string{"before"}); err != nil {
		t.Fatal(err)
	}
	reset := false
	if err := store.TransitionOutputEpoch(func() {
		if !gate.held {
			t.Fatal("parser reset outside ingest gate")
		}
		reset = true
	}); err != nil {
		t.Fatal(err)
	}
	if !reset {
		t.Fatal("parser reset was not invoked")
	}
	if gaps := file.GapOffsets(); len(gaps) != 1 || gaps[0] != 1 {
		t.Fatalf("epoch transition gap offsets=%v", gaps)
	}
}
