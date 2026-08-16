package state

import (
	"testing"
	"time"
)

func TestTerminalSurfaceApplySnapshotIsDetached(t *testing.T) {
	lines := []string{"one", "two"}
	screen := [][]LiveCell{{{Text: "one", Width: 3, FG: "ansi:2"}}}
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   7,
		Cols:       80,
		Rows:       24,
		Lines:      lines,
		Screen:     screen,
		Title:      "shell",
		Cursor:     LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
	})
	lines[0] = "mutated"
	screen[0][0].Text = "mutated"

	if store.TerminalID != "term-1" || store.Cols != 80 || store.Rows != 24 || store.Title != "shell" || !store.Ready {
		t.Fatalf("unexpected surface store %#v", store)
	}
	if store.Revision != 7 || store.Surfaces["term-1"].Revision != 7 {
		t.Fatalf("expected snapshot revision to be projected and cached, got %#v", store)
	}
	if !store.Cursor.Visible || store.Cursor.Row != 1 || store.Cursor.Col != 2 || store.Cursor.Shape != "bar" {
		t.Fatalf("expected detached cursor metadata, got %#v", store.Cursor)
	}
	if store.Lines[0] != "one" {
		t.Fatalf("expected detached lines, got %#v", store.Lines)
	}
	if store.Screen[0][0].Text != "one" || store.Screen[0][0].FG != "ansi:2" {
		t.Fatalf("expected detached live screen cells, got %#v", store.Screen)
	}
	cached := store.Surfaces["term-1"]
	if cached.Screen[0][0].Text != "one" || cached.Screen[0][0].FG != "ansi:2" {
		t.Fatalf("expected detached cached live screen cells, got %#v", cached.Screen)
	}
}

func TestTerminalSurfaceSnapshotReturnsDetachedPayload(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   7,
		Lines:      []string{"one"},
		Screen:     [][]LiveCell{{{Text: "one", Width: 3, FG: "ansi:2"}}},
	})

	snapshot := store.Snapshot()
	snapshot.Lines[0] = "mutated"
	snapshot.Screen[0][0].Text = "mutated"

	if store.Lines[0] != "one" {
		t.Fatalf("snapshot must not expose mutable store lines, got %#v", store.Lines)
	}
	if store.Screen[0][0].Text != "one" || store.Surfaces["term-1"].Screen[0][0].Text != "one" {
		t.Fatalf("snapshot must not expose mutable store cells, store=%#v cached=%#v", store.Screen, store.Surfaces["term-1"].Screen)
	}
}

func TestTerminalSurfaceAppliesSparseRowsOnMatchingRevision(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID:  "term-1",
		Revision:    4,
		FullReplace: true,
		Cols:        8,
		Rows:        3,
		Title:       "build shell",
		State:       TerminalLiveAttached,
		Command:     []string{"zsh", "-l"},
		Screen: [][]LiveCell{
			{{Text: "zero", Width: 4}},
			{{Text: "one", Width: 3}},
			{{Text: "two", Width: 3}},
		},
	})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID:   "term-1",
		BaseRevision: 4,
		Revision:     7,
		Cols:         8,
		Rows:         3,
		ChangedRows:  []int{1},
		Screen:       [][]LiveCell{{{Text: "changed", Width: 7}}},
		Cursor:       LiveCursor{Visible: true, Row: 1, Col: 7},
	})

	if store.Revision != 7 || store.Screen[0][0].Text != "zero" || store.Screen[1][0].Text != "changed" || store.Screen[2][0].Text != "two" {
		t.Fatalf("sparse snapshot did not preserve unchanged rows: %#v", store.Screen)
	}
	if !store.Cursor.Visible || store.Cursor.Row != 1 || store.Cursor.Col != 7 {
		t.Fatalf("sparse snapshot lost cursor metadata: %#v", store.Cursor)
	}
	if store.Title != "build shell" || store.State != TerminalLiveAttached || len(store.Command) != 2 || store.Command[0] != "zsh" {
		t.Fatalf("sparse snapshot must preserve lifecycle metadata: %#v", store)
	}
}

func TestTerminalSurfaceRejectsSparseRowsWithStaleBase(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   5,
		Cols:       8,
		Rows:       1,
		Screen:     [][]LiveCell{{{Text: "current", Width: 7}}},
	})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID:   "term-1",
		BaseRevision: 4,
		Revision:     6,
		Cols:         8,
		Rows:         1,
		ChangedRows:  []int{0},
		Screen:       [][]LiveCell{{{Text: "wrong", Width: 5}}},
	})

	if store.Revision != 5 || store.Screen[0][0].Text != "current" {
		t.Fatalf("stale sparse base must not corrupt current screen: %#v", store)
	}
}

func TestTerminalSurfaceRejectsStaleLiveRevision(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"new"},
	})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   1,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"old"},
	})

	if store.Revision != 2 || store.Lines[0] != "new" {
		t.Fatalf("stale revision must not replace current projection, got %#v", store)
	}
	if cached := store.Surfaces["term-1"]; cached.Revision != 2 || cached.Lines[0] != "new" {
		t.Fatalf("stale revision must not replace cached surface, got %#v", cached)
	}
}

func TestTerminalSurfaceRejectsEmptyBootstrapOverContent(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"prompt"},
		Cursor:     LiveCursor{Visible: true, Row: 0, Col: 6, Shape: "bar"},
	})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
	})

	if store.Revision != 3 || len(store.Lines) != 1 || store.Lines[0] != "prompt" || !store.Cursor.Visible {
		t.Fatalf("empty bootstrap snapshot must not replace current content, got %#v", store)
	}
	if cached := store.Surfaces["term-1"]; cached.Revision != 3 || len(cached.Lines) != 1 || cached.Lines[0] != "prompt" || !cached.Cursor.Visible {
		t.Fatalf("empty bootstrap snapshot must not replace cached content, got %#v", cached)
	}
}

func TestTerminalSurfaceAttachKeepsCachedRevision(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   5,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"connected panel"},
	})
	store = store.Attach("term-2", 80, 24)
	store = store.Attach("term-1", 80, 24)
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
	})

	if store.Revision != 5 || len(store.Lines) != 1 || store.Lines[0] != "connected panel" {
		t.Fatalf("reattach must keep cached content when empty bootstrap arrives, got %#v", store)
	}
}

func TestTerminalSurfaceRestartPreservesCurrentContentAndClearsLifecycle(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   5,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"old live tail"},
		Cursor:     LiveCursor{Visible: true, Row: 0, Col: 13},
		Modes:      LiveTerminalModes{MouseTracking: true, BracketedPaste: true},
	})
	store = store.MarkExitedWithMetadata("term-1", 23, "exited", time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC), []string{"bash", "-lc", "exit 23"})

	store = store.RestartPreservingContent("term-1", 80, 24)

	if store.State != TerminalLiveAttached || store.ExitCode != 0 || !store.ExitedAt.IsZero() || len(store.Command) != 0 || store.Err != "" {
		t.Fatalf("restart should clear lifecycle metadata, got %#v", store)
	}
	if len(store.Lines) != 1 || store.Lines[0] != "old live tail" || !store.Ready {
		t.Fatalf("restart should keep live tail ready, got %#v", store)
	}
	if store.Cursor != (LiveCursor{}) || store.Modes != (LiveTerminalModes{}) {
		t.Fatalf("restart must drop old process cursor/modes, got cursor=%#v modes=%#v", store.Cursor, store.Modes)
	}
}

func TestTerminalSurfaceRestartNonCurrentOnlyUpdatesCache(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
		Lines:      []string{"current"},
	})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-2",
		Cols:       40,
		Rows:       12,
		Lines:      []string{"other old tail"},
	})
	store = store.MarkExited("term-2", 9, "exited")

	store = store.RestartPreservingContent("term-2", 40, 12)

	if store.TerminalID != "term-1" || len(store.Lines) != 1 || store.Lines[0] != "current" {
		t.Fatalf("restart for non-current terminal must not switch projection, got %#v", store)
	}
	cached := store.Surfaces["term-2"]
	if cached.State != TerminalLiveAttached || cached.ExitCode != 0 || len(cached.Lines) != 1 || cached.Lines[0] != "other old tail" {
		t.Fatalf("restart should update only the target cache, got %#v", cached)
	}
}

func TestTerminalSurfaceResizeBoundaryRejectsLateOldSizeFrame(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"before resize"},
	})
	store = store.Resize(100, 40)
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"late old size"},
	})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
		Lines:      []string{"late unknown revision"},
	})

	if store.Cols != 100 || store.Rows != 40 || store.Lines[0] != "before resize" {
		t.Fatalf("late old-size frames must not roll back resized projection, got %#v", store)
	}
	if !store.ResizeBoundary.Active {
		t.Fatalf("resize boundary should remain until matching-size surface arrives, got %#v", store.ResizeBoundary)
	}
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       98,
		Rows:       38,
		Lines:      []string{"accepted non-old size"},
	})
	if store.Cols != 98 || store.Rows != 38 || store.Lines[0] != "accepted non-old size" {
		t.Fatalf("resize boundary should not reject non-old-size live frames, got %#v", store)
	}
	store = store.Resize(100, 40)
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   4,
		Cols:       100,
		Rows:       40,
		Lines:      []string{"after resize"},
	})
	if store.ResizeBoundary.Active || store.Cols != 100 || store.Rows != 40 || store.Lines[0] != "after resize" {
		t.Fatalf("matching-size frame should clear resize boundary, got %#v", store)
	}
}

func TestTerminalSurfaceExitBoundaryRejectsLateOrdinaryFrame(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"last screen"},
	})
	store = store.MarkExitedWithMetadata("term-1", 0, "done", time.Time{}, nil)
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"late ordinary"},
	})

	if store.State != TerminalLiveExited || store.ExitReason != "done" {
		t.Fatalf("late ordinary frame must not clear exit state, got %#v", store)
	}
	if store.Lines[0] != "last screen" {
		t.Fatalf("exit boundary should preserve last accepted screen, got %#v", store.Lines)
	}
}

func TestTerminalSurfaceOrdinaryRunningSnapshotDoesNotClearExitBoundary(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   9,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited"},
	})
	store = store.MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited", "% "},
		Cursor:     LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
		State:      TerminalLiveAttached,
	})

	if store.State != TerminalLiveExited || store.ExitReason != "exited" || store.Revision != 9 {
		t.Fatalf("ordinary live snapshot must not clear exited boundary, got %#v", store)
	}
}

func TestTerminalSurfaceAttachAllowsFreshFrameAfterExitBoundary(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"last screen"},
	})
	store = store.MarkExitedWithMetadata("term-1", 0, "done", time.Time{}, nil)
	store = store.Attach("term-1", 80, 24)
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   1,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"reattached"},
	})

	if store.State != TerminalLiveAttached || store.ExitReason != "" || store.Lines[0] != "reattached" {
		t.Fatalf("explicit attach boundary should accept a fresh ordinary frame, got %#v", store)
	}
}

func TestTerminalSurfaceMarkAttachedClearsBoundaryAndAcceptsCursorSnapshot(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"old live tail"},
	})
	store = store.MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 40, 0, 0, time.UTC), []string{"/bin/zsh"})

	store = store.MarkAttached("term-1")
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"old live tail", "% "},
		Cursor:     LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
	})

	if store.State != TerminalLiveAttached || store.ExitReason != "" || !store.ExitedAt.IsZero() || len(store.Command) != 0 {
		t.Fatalf("running lifecycle should clear exited metadata, got %#v", store)
	}
	if len(store.Lines) != 2 || store.Lines[1] != "% " {
		t.Fatalf("fresh live snapshot should replace old tail projection, got %#v", store.Lines)
	}
	if !store.Cursor.Visible || store.Cursor.Row != 1 || store.Cursor.Col != 2 || store.Cursor.Shape != "bar" {
		t.Fatalf("fresh core cursor should be accepted after lifecycle clear, got %#v", store.Cursor)
	}
}

func TestTerminalSurfaceAttachProjectsCachedSnapshot(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"cached live"},
	})
	store = store.Attach("term-2", 100, 40)
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   4,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"cached update"},
	})
	store = store.Attach("term-1", 80, 24)

	if !store.Ready || store.TerminalID != "term-1" || store.Lines[0] != "cached update" {
		t.Fatalf("attach should immediately project cached live surface, got %#v", store)
	}
}

func TestTerminalSurfaceTracksMouseModesForPassthrough(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
		Modes:      LiveTerminalModes{MouseTracking: true, MouseSGR: true},
	})
	if !store.Modes.MousePassthroughEnabled() || !store.SurfaceForTerminal("term-1").Modes.MouseSGR {
		t.Fatalf("expected mouse modes preserved on surface snapshot, got %#v", store)
	}
	store = store.Attach("term-2", 80, 24)
	if store.Modes.MousePassthroughEnabled() {
		t.Fatalf("attach switch must clear projected mouse modes, got %#v", store.Modes)
	}
	store = store.ApplySnapshot(LiveSurfaceSnapshot{TerminalID: "term-2", Modes: LiveTerminalModes{MouseNormal: true}})
	store = store.MarkExited("term-2", 0, "done")
	if store.Modes.MousePassthroughEnabled() {
		t.Fatalf("exited surface must clear mouse passthrough modes, got %#v", store.Modes)
	}
}

func TestTerminalSessionAttachResizeAndError(t *testing.T) {
	session := (TerminalSessionStore{}).Attach("term-1", 7, 80, 24)
	if !session.Attached || session.Channel != 7 || session.TerminalID != "term-1" || session.State != TerminalLiveAttached {
		t.Fatalf("unexpected attached session %#v", session)
	}
	session = session.Resize(100, 40)
	if session.Cols != 100 || session.Rows != 40 {
		t.Fatalf("unexpected resized session %#v", session)
	}
	session = session.SetError("boom")
	if session.LastError != "boom" || session.State != TerminalLiveError || session.Attached {
		t.Fatalf("expected session error, got %#v", session)
	}
}

func TestTerminalSessionErrorIsNotClearedByLifecycleAttachedProjection(t *testing.T) {
	session := (TerminalSessionStore{}).Attach("term-1", 7, 80, 24).SetError("pty resize failed")
	next := session.MarkAttached("term-1")
	if next.LastError != "pty resize failed" || next.State != TerminalLiveError || next.Attached {
		t.Fatalf("lifecycle attached projection must not clear live session error, got %#v", next)
	}
}

func TestTerminalLiveLifecycleTracksAttachSwitchExitAndError(t *testing.T) {
	surface := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
		Lines:      []string{"old"},
	})
	surface = surface.Attach("term-2", 100, 40)
	if surface.TerminalID != "term-2" || surface.Ready || len(surface.Lines) != 0 || surface.State != TerminalLiveAttached {
		t.Fatalf("attach switch must clear stale surface projection, got %#v", surface)
	}
	surface = surface.MarkExited("term-2", 130, "closed")
	if surface.State != TerminalLiveExited || surface.ExitCode != 130 || surface.ExitReason != "closed" || surface.Cursor.Visible {
		t.Fatalf("expected exited surface lifecycle, got %#v", surface)
	}
	surface = surface.SetError("transport lost")
	if surface.State != TerminalLiveError || surface.Err != "transport lost" {
		t.Fatalf("expected surface error lifecycle, got %#v", surface)
	}

	session := (TerminalSessionStore{}).Attach("term-2", 3, 100, 40).MarkExited("term-2", 0, "done")
	if session.Attached || session.State != TerminalLiveExited || session.ExitReason != "done" || session.LastError != "" {
		t.Fatalf("expected exited session lifecycle, got %#v", session)
	}
}

func TestTerminalSessionTracksRequestedResizeAndRejectsStaleResults(t *testing.T) {
	session := (TerminalSessionStore{}).Attach("term-1", 7, 80, 24)
	session = session.RequestResize(40, 20)
	firstSeq := session.ResizeRequestSeq
	session = session.RequestResize(98, 36)
	if cols, rows := session.DesiredSize(); cols != 98 || rows != 36 {
		t.Fatalf("expected latest desired size, got %dx%d in %#v", cols, rows, session)
	}
	if !session.IsStaleResizeResult(firstSeq) {
		t.Fatalf("expected first resize result to be stale, got %#v", session)
	}
	next, applied := session.ApplyResizeResult(firstSeq, 40, 20)
	if applied || next.Cols != 80 || next.Rows != 24 || next.DesiredCols != 98 || next.DesiredRows != 36 {
		t.Fatalf("stale resize result must not mutate session, next=%#v applied=%v", next, applied)
	}
	next, applied = session.ApplyResizeResult(session.ResizeRequestSeq, 98, 36)
	if !applied || next.Cols != 98 || next.Rows != 36 || next.DesiredCols != 98 || next.DesiredRows != 36 {
		t.Fatalf("latest resize result must apply, next=%#v applied=%v", next, applied)
	}
}

func TestTerminalSessionTracksResizeOwner(t *testing.T) {
	session := (TerminalSessionStore{}).AttachWithResizeOwner("term-1", 7, 80, 24, "owner", "surface-1", "view-1")
	if session.ResizePolicy != "owner" || session.SurfaceID != "surface-1" || session.ViewID != "view-1" {
		t.Fatalf("expected resize owner metadata, got %#v", session)
	}
	if errored := session.SetError("boom"); errored.ResizePolicy != "" || errored.SurfaceID != "" || errored.ViewID != "" {
		t.Fatalf("error state must clear resize owner metadata, got %#v", errored)
	}
	if exited := session.MarkExited("term-1", 0, "done"); exited.ResizePolicy != "" || exited.SurfaceID != "" || exited.ViewID != "" {
		t.Fatalf("exit state must clear resize owner metadata, got %#v", exited)
	}
}

func TestTerminalSurfaceRefreshBackpressureConsumesDirtyAfterFetch(t *testing.T) {
	var store TerminalSurfaceStore
	var fetch bool
	store, fetch = store.RequestRefresh("term-1", 80, 24)
	if !fetch || !store.Refreshes["term-1"].InFlight || store.Refreshes["term-1"].Dirty {
		t.Fatalf("first invalidation should start a fetch, store=%#v fetch=%v", store.Refreshes, fetch)
	}

	store, fetch = store.RequestRefresh("term-1", 96, 30)
	if fetch {
		t.Fatalf("in-flight invalidation should not start a second fetch, store=%#v", store.Refreshes)
	}
	if request := store.Refreshes["term-1"]; !request.InFlight || !request.Dirty || request.Cols != 96 || request.Rows != 30 {
		t.Fatalf("in-flight request should keep latest dirty size, got %#v", request)
	}

	store = store.ApplySnapshot(LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 8, Cols: 80, Rows: 24, Lines: []string{"rev8"}})
	if request := store.Refreshes["term-1"]; request.InFlight || request.Dirty || request.Cols != 96 || request.Rows != 30 {
		t.Fatalf("accepted snapshot should leave dirty refresh consumable, got %#v", request)
	}

	var cols, rows int
	store, cols, rows, fetch = store.ConsumeDirtyRefresh("term-1")
	if !fetch || cols != 96 || rows != 30 {
		t.Fatalf("dirty refresh should schedule one latest fetch, cols=%d rows=%d fetch=%v store=%#v", cols, rows, fetch, store.Refreshes)
	}
	if request := store.Refreshes["term-1"]; !request.InFlight || request.Dirty {
		t.Fatalf("follow-up fetch should be in-flight and clean, got %#v", request)
	}

	store = store.ApplySnapshot(LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 10, Cols: 96, Rows: 30, Lines: []string{"rev10"}})
	if _, ok := store.Refreshes["term-1"]; ok {
		t.Fatalf("fresh snapshot should clear refresh state, store=%#v", store.Refreshes)
	}
}

func TestTerminalSurfaceRejectedSnapshotFinishesRefresh(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   10,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"current"},
	})
	var fetch bool
	store, fetch = store.RequestRefresh("term-1", 96, 30)
	if !fetch {
		t.Fatalf("expected refresh request to start fetch, store=%#v", store.Refreshes)
	}

	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   9,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"stale"},
	})
	if got := store.SurfaceForTerminal("term-1"); got.Revision != 10 || len(got.Lines) != 1 || got.Lines[0] != "current" {
		t.Fatalf("stale snapshot must not replace current surface, got %#v", got)
	}
	if _, ok := store.RefreshStateRef(LocalTerminalRef("term-1")); ok {
		t.Fatalf("rejected stale snapshot must release in-flight refresh, store=%#v", store.Refreshes)
	}

	store = store.Resize(120, 40)
	store.ResizeBoundary = LiveResizeBoundary{Active: true, PreviousCols: 96, PreviousRows: 30, Cols: 120, Rows: 40}
	store, fetch = store.RequestRefresh("term-1", 120, 40)
	if !fetch {
		t.Fatalf("expected resized refresh request to start fetch, store=%#v", store.Refreshes)
	}
	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   11,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"late old size"},
	})
	if got := store.SurfaceForTerminal("term-1"); got.Cols != 120 || got.Rows != 40 || len(got.Lines) != 1 || got.Lines[0] != "current" {
		t.Fatalf("old-size snapshot must not roll back resized surface, got %#v", got)
	}
	if _, ok := store.RefreshStateRef(LocalTerminalRef("term-1")); ok {
		t.Fatalf("rejected old-size snapshot must release in-flight refresh, store=%#v", store.Refreshes)
	}
}

func TestTerminalSurfaceRejectedDirtySnapshotKeepsFollowupConsumable(t *testing.T) {
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   10,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"current"},
	})
	var fetch bool
	store, fetch = store.RequestRefresh("term-1", 96, 30)
	if !fetch {
		t.Fatalf("expected refresh request to start fetch, store=%#v", store.Refreshes)
	}
	store, fetch = store.RequestRefresh("term-1", 120, 40)
	if fetch {
		t.Fatalf("dirty invalidation must not start a parallel fetch, store=%#v", store.Refreshes)
	}

	store = store.ApplySnapshot(LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   9,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"stale"},
	})
	refresh, ok := store.RefreshStateRef(LocalTerminalRef("term-1"))
	if !ok || refresh.InFlight || refresh.Dirty || refresh.Cols != 120 || refresh.Rows != 40 {
		t.Fatalf("rejected dirty snapshot should leave follow-up refresh consumable, got %#v ok=%v", refresh, ok)
	}
	var cols, rows int
	store, cols, rows, fetch = store.ConsumeDirtyRefresh("term-1")
	if !fetch || cols != 120 || rows != 40 {
		t.Fatalf("dirty follow-up must stay schedulable after rejected snapshot, cols=%d rows=%d fetch=%v store=%#v", cols, rows, fetch, store.Refreshes)
	}
}

func TestTerminalSurfaceLifecycleBoundaryClearsRefresh(t *testing.T) {
	ref := LocalTerminalRef("term-1")
	cases := []struct {
		name  string
		apply func(TerminalSurfaceStore) TerminalSurfaceStore
	}{
		{
			name: "attach",
			apply: func(store TerminalSurfaceStore) TerminalSurfaceStore {
				return store.AttachRef(ref, 120, 40)
			},
		},
		{
			name: "cache attach",
			apply: func(store TerminalSurfaceStore) TerminalSurfaceStore {
				return store.CacheAttachRef(ref, 120, 40)
			},
		},
		{
			name: "restart",
			apply: func(store TerminalSurfaceStore) TerminalSurfaceStore {
				return store.RestartPreservingContentRef(ref, 120, 40)
			},
		},
		{
			name: "mark attached",
			apply: func(store TerminalSurfaceStore) TerminalSurfaceStore {
				return store.MarkAttachedRef(ref)
			},
		},
		{
			name: "mark exited",
			apply: func(store TerminalSurfaceStore) TerminalSurfaceStore {
				return store.MarkExitedWithMetadataRef(ref, 0, "done", time.Time{}, nil)
			},
		},
		{
			name: "set error",
			apply: func(store TerminalSurfaceStore) TerminalSurfaceStore {
				return store.SetErrorRef(ref, "transport offline")
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			store := terminalSurfaceStoreWithDirtyRefresh(t, ref)
			store = tt.apply(store)
			if refresh, ok := store.RefreshStateRef(ref); ok {
				t.Fatalf("lifecycle boundary must clear refresh debt, got %#v", refresh)
			}
		})
	}
}

func TestTerminalSurfaceLifecycleBoundaryClearsOnlyMatchingEndpointRefresh(t *testing.T) {
	local := LocalTerminalRef("term-1")
	west := NewTerminalRef("west", "term-1")
	store := terminalSurfaceStoreWithDirtyRefresh(t, local)
	var fetch bool
	store, fetch = store.RequestRefreshRef(west, 120, 40)
	if !fetch {
		t.Fatalf("expected west refresh request to start fetch, store=%#v", store.Refreshes)
	}

	store = store.MarkExitedWithMetadataRef(west, 0, "done", time.Time{}, nil)
	if _, ok := store.RefreshStateRef(west); ok {
		t.Fatalf("west lifecycle boundary must clear west refresh debt, store=%#v", store.Refreshes)
	}
	if refresh, ok := store.RefreshStateRef(local); !ok || !refresh.InFlight || !refresh.Dirty {
		t.Fatalf("west lifecycle boundary must not clear local refresh debt, got %#v ok=%v", refresh, ok)
	}
}

func TestTerminalSurfaceLiveScreenAllowsOneRequestAndOnePendingRevision(t *testing.T) {
	var store TerminalSurfaceStore
	ref := LocalTerminalRef("term-1")
	store, canceled := store.ReconcileLiveScreenDemand([]TerminalRef{ref})
	if len(canceled) != 0 {
		t.Fatalf("new demand should not cancel requests, got %#v", canceled)
	}
	store = store.SubmitLiveScreenRef(ref, 8, 80, 24)
	var request LiveScreenRequestState
	var start bool
	store, request, start = store.BeginLiveScreenRequestRef(ref)
	if !start || !request.RequestInFlight || request.SubmittedRevision != 8 {
		t.Fatalf("first submission should start one request, got %#v start=%v", request, start)
	}
	store, _, start = store.BeginLiveScreenRequestRef(ref)
	if start {
		t.Fatal("request in flight must reject a parallel request")
	}
	store, ok := store.FinishLiveScreenRequestRef(ref, request.Generation, 9)
	if !ok {
		t.Fatal("matching result should release request")
	}
	store, _, start = store.BeginLiveScreenRequestRef(ref)
	if start {
		t.Fatal("received revision must stay pending until renderer submission")
	}
	store = store.SubmitLiveScreenRef(ref, 9, 80, 24)
	store, request, start = store.BeginLiveScreenRequestRef(ref)
	if !start || request.SubmittedRevision != 9 {
		t.Fatalf("submitting pending revision should start next request, got %#v start=%v", request, start)
	}
}

func TestTerminalSurfaceLiveScreenBootstrapBypassesPendingRevision(t *testing.T) {
	var store TerminalSurfaceStore
	ref := LocalTerminalRef("term-1")
	store, _ = store.ReconcileLiveScreenDemand([]TerminalRef{ref})
	store = store.SubmitLiveScreenRef(ref, 7, 80, 24)
	var request LiveScreenRequestState
	var start bool
	store, request, start = store.BeginLiveScreenRequestRef(ref)
	if !start {
		t.Fatal("initial request should start")
	}
	store, ok := store.RequireLiveScreenBootstrap(ref, request.Generation)
	if !ok {
		t.Fatal("matching request should enter bootstrap state")
	}
	store = store.SubmitLiveScreenRef(ref, 7, 80, 24)
	store, request, start = store.BeginLiveScreenRequestRef(ref)
	if !start || !request.NeedsBootstrap || !request.RequestInFlight {
		t.Fatalf("bootstrap must start even while the old revision remains visible, got %#v start=%v", request, start)
	}
	store, ok = store.FinishLiveScreenRequestRef(ref, request.Generation, 9)
	if !ok {
		t.Fatal("full bootstrap result should finish the request")
	}
	request, _ = store.LiveScreenRequestRef(ref)
	if request.NeedsBootstrap || request.ReceivedRevision != 9 {
		t.Fatalf("successful bootstrap should restore steady delivery, got %#v", request)
	}
}

func terminalSurfaceStoreWithDirtyRefresh(t *testing.T, ref TerminalRef) TerminalSurfaceStore {
	t.Helper()
	store := (TerminalSurfaceStore{}).ApplySnapshot(LiveSurfaceSnapshot{
		EndpointID: ref.EndpointID,
		TerminalID: ref.TerminalID,
		Revision:   10,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"current"},
	})
	var fetch bool
	store, fetch = store.RequestRefreshRef(ref, 96, 30)
	if !fetch {
		t.Fatalf("expected refresh request to start fetch, store=%#v", store.Refreshes)
	}
	store, fetch = store.RequestRefreshRef(ref, 120, 40)
	if fetch {
		t.Fatalf("dirty invalidation must not start a parallel fetch, store=%#v", store.Refreshes)
	}
	if refresh, ok := store.RefreshStateRef(ref); !ok || !refresh.InFlight || !refresh.Dirty {
		t.Fatalf("expected dirty refresh state, got %#v ok=%v", refresh, ok)
	}
	return store
}

func TestTerminalSurfaceStoreSeparatesSameTerminalAcrossEndpoints(t *testing.T) {
	var store TerminalSurfaceStore
	store = store.ApplySnapshot(LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 1, Cols: 80, Rows: 24, Lines: []string{"local"}})
	store = store.ApplySnapshot(LiveSurfaceSnapshot{EndpointID: "west", TerminalID: "term-1", Revision: 2, Cols: 100, Rows: 30, Lines: []string{"west"}})

	if got := store.SurfaceForTerminal("term-1").Lines; len(got) != 1 || got[0] != "local" {
		t.Fatalf("local surface must keep local content, got %#v", got)
	}
	if got := store.SurfaceForTerminalRef(NewTerminalRef("west", "term-1")).Lines; len(got) != 1 || got[0] != "west" {
		t.Fatalf("west surface must keep west content, got %#v", got)
	}

	var fetch bool
	store, fetch = store.RequestRefresh("term-1", 80, 24)
	if !fetch {
		t.Fatalf("local refresh should arm independently")
	}
	store, fetch = store.RequestRefreshRef(NewTerminalRef("west", "term-1"), 100, 30)
	if !fetch {
		t.Fatalf("west refresh should not be blocked by local terminal id")
	}
	if _, ok := store.Refreshes["term-1"]; !ok {
		t.Fatalf("local refresh key missing: %#v", store.Refreshes)
	}
	if _, ok := store.Refreshes["west/term-1"]; !ok {
		t.Fatalf("west refresh key missing: %#v", store.Refreshes)
	}
}

func TestTerminalSurfaceResizeRefDoesNotSwitchActiveProjection(t *testing.T) {
	leftRef := NewTerminalRef("west", "term-left")
	rightRef := NewTerminalRef("east", "term-right")
	store := (TerminalSurfaceStore{}).
		AttachRef(leftRef, 100, 30).
		CacheAttachRef(rightRef, 60, 20)

	next := store.ResizeRef(rightRef, 72, 25)
	if !next.TerminalRef().Equal(leftRef) || next.Cols != 100 || next.Rows != 30 {
		t.Fatalf("background resize switched active projection, store=%#v", next)
	}
	if next.ResizeBoundary.Active {
		t.Fatalf("background resize must not create an active resize boundary, boundary=%#v", next.ResizeBoundary)
	}
	right := next.SurfaceForTerminalRef(rightRef)
	if right.Cols != 72 || right.Rows != 25 {
		t.Fatalf("background surface geometry was not cached, surface=%#v", right)
	}

	next = next.ResizeRef(leftRef, 110, 32)
	if next.Cols != 110 || next.Rows != 32 || !next.ResizeBoundary.Active {
		t.Fatalf("active resize must update projection and boundary, store=%#v", next)
	}
}

func TestTerminalSessionStoreSeparatesInputChannelsAcrossEndpoints(t *testing.T) {
	session := (TerminalSessionStore{}).
		AttachWithResizeOwner("term-1", 7, 80, 24, "owner", "surface-local", "view-local").
		AttachRefWithResizeOwner(NewTerminalRef("west", "term-1"), 9, 100, 30, "follower", "surface-west", "view-west")

	if channel, ok := session.InputChannelFor("term-1"); !ok || channel != 7 {
		t.Fatalf("local input channel mismatch, channel=%d ok=%v session=%#v", channel, ok, session.InputChannels)
	}
	if channel, ok := session.InputChannelForRef(NewTerminalRef("west", "term-1")); !ok || channel != 9 {
		t.Fatalf("west input channel mismatch, channel=%d ok=%v session=%#v", channel, ok, session.InputChannels)
	}

	session = session.ClearInputChannelRef(NewTerminalRef("west", "term-1"))
	if _, ok := session.InputChannelForRef(NewTerminalRef("west", "term-1")); ok {
		t.Fatalf("west input channel should be cleared, session=%#v", session.InputChannels)
	}
	if channel, ok := session.InputChannelFor("term-1"); !ok || channel != 7 {
		t.Fatalf("local input channel must survive west clear, channel=%d ok=%v session=%#v", channel, ok, session.InputChannels)
	}
}
