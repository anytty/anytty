package app

import (
	"context"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/testkit"
)

func TestCopyModeMouseEdgeLoadsOlderAndKeepsAnchor(t *testing.T) {
	runtime, host, core, rect := mouseAutoScrollFixture(t)
	now := time.Now()
	runtime.now = func() time.Time { return now }
	start := runtime.State().CopyMode.Cursor
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	anchor := runtime.State().CopyMode.Selection.LogicalAnchor
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	if runtime.mouseDrag.HistoryScrollDirection != -1 {
		t.Fatalf("not on top edge: %+v rect=%+v cursor=%+v", runtime.mouseDrag, rect, start)
	}
	before := runtime.State().CopyMode.Selection.LogicalFocus
	for i := 0; i < 30 && len(core.OlderRequests) == 0; i++ {
		now = now.Add(historyMouseScrollInterval)
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(core.OlderRequests) != 1 {
		t.Fatalf("older requests: %d", len(core.OlderRequests))
	}
	selection := runtime.State().CopyMode.Selection
	if selection.LogicalAnchor != anchor {
		t.Fatalf("anchor moved across prepend: %+v -> %+v", anchor, selection.LogicalAnchor)
	}
	if selection.LogicalFocus.LineID >= before.LineID {
		t.Fatalf("stationary edge did not extend into older history: %+v -> %+v", before, selection.LogicalFocus)
	}
	sendCopyMouse(t, runtime, host, input.MouseLeftUp, rect.Y+1, rect.X+2)
	position := runtime.State().CopyMode.Cursor
	now = now.Add(time.Second)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Cursor != position || runtime.nextHistoryMouseScrollWakeDelay() != -1 {
		t.Fatal("autoscroll continued after release")
	}
}

func TestCopyModeMouseEdgeTimerWakesWithoutNewInput(t *testing.T) {
	runtime, host, _, rect := mouseAutoScrollFixture(t)
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	before := runtime.State().CopyMode.Selection.LogicalFocus
	// Clear unrelated wake notifications left by the already-drained input.
	for {
		select {
		case <-runtime.wake:
			continue
		default:
			goto drained
		}
	}
drained:
	for {
		select {
		case <-host.EventsReady():
			continue
		default:
			goto ready
		}
	}
ready:
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !runtime.waitForWake(ctx) {
		t.Fatal("stationary drag never woke the runtime")
	}
	if err := runtime.Drain(ctx); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Selection.LogicalFocus == before {
		t.Fatal("timer wake did not scroll")
	}
}

func TestCopyModeMouseEdgeStopsInInteriorAndRejectsQueuedTicks(t *testing.T) {
	runtime, host, _, rect := mouseAutoScrollFixture(t)
	now := time.Now()
	runtime.now = func() time.Time { return now }
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	tick := CopyModeMouseAutoScrollMsg{GestureID: runtime.mouseDrag.HistoryGestureID, ViewID: runtime.mouseDrag.ViewID, Direction: -1}
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+rect.H/2, rect.X+2)
	pos := runtime.State().CopyMode.Cursor
	if runtime.nextHistoryMouseScrollWakeDelay() != -1 {
		t.Fatal("interior still arms timer")
	}
	now = now.Add(time.Second)
	runtime.enqueue(tick)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Cursor != pos {
		t.Fatal("queued old-direction tick scrolled")
	}
	sendCopyMouse(t, runtime, host, input.MouseLeftUp, rect.Y+rect.H/2, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	pos = runtime.State().CopyMode.Cursor
	runtime.enqueue(tick)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Cursor != pos {
		t.Fatal("previous gesture tick affected new selection")
	}
}

func TestCopyModeMouseBottomEdgeScrollsWithoutLeavingHistory(t *testing.T) {
	runtime, host, _, rect := mouseAutoScrollFixture(t)
	now := time.Now()
	runtime.now = func() time.Time { return now }
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	for i := 0; i < 4; i++ {
		now = now.Add(historyMouseScrollInterval)
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+rect.H, rect.X+2)
	before := runtime.State().CopyMode.Cursor.Row
	now = now.Add(historyMouseScrollInterval)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Cursor.Row <= before || !runtime.State().CopyMode.Active {
		t.Fatal("bottom edge failed to scroll within frozen history")
	}
}

func mouseAutoScrollFixture(t *testing.T) (*AppRuntime, *FakeTerminalHost, *testkit.FakeCoreClient, render.Rect) {
	t.Helper()
	latest := historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 78, 7, historyRowsForEnteringScrollTest("new", 30, 100))
	latest.Cursor = state.HistoryCursor{Valid: true, BeforeLineID: 100}
	older := historyWindowForApp(state.HistoryWindowPrepend, "term-1", "tok-1", 78, 7, historyRowsForEnteringScrollTest("old", 10, 90))
	older.Boundary = state.HistoryBoundary{FirstLineID: 90, LastLineID: 129}
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: latest}}, OlderResponses: []port.HistoryResult{{Window: older}}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 10)
	runtime := newCopyModeRuntime(host, core, &testkit.FakeClipboardService{})
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	rect, ok := copyModeContentRect(runtime.State())
	if !ok {
		t.Fatal("missing copy content rect")
	}
	return runtime, host, core, rect
}

func TestCopyModeMouseEdgePausesForPendingHistory(t *testing.T) {
	runtime, host, core, rect := mouseAutoScrollFixture(t)
	now := time.Now()
	runtime.now = func() time.Time { return now }
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	pending := &state.HistoryPendingRequest{ID: 900, Kind: state.HistoryRequestOlder, DeferredScrollRows: 0}
	runtime.state.History.Pending = pending
	before := runtime.State().CopyMode.Cursor
	now = now.Add(10 * time.Second)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().CopyMode.Cursor != before || pending.DeferredScrollRows != 0 || len(core.OlderRequests) != 0 || runtime.nextHistoryMouseScrollWakeDelay() != -1 {
		t.Fatal("pending history accumulated scrolling/requests")
	}
	runtime.state.History.Pending = nil
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if delta := before.Row - runtime.State().CopyMode.Cursor.Row; delta != 2 {
		t.Fatalf("resume should take one step, not replay 10 seconds: %d", delta)
	}
}

func TestCopyModeMouseEdgeStopsAtOldestAndOnCancel(t *testing.T) {
	runtime, host, core, rect := mouseAutoScrollFixture(t)
	now := time.Now()
	runtime.now = func() time.Time { return now }
	sendCopyMouse(t, runtime, host, input.MouseLeft, rect.Y+rect.H-1, rect.X+2)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+1, rect.X+2)
	for i := 0; i < 50; i++ {
		now = now.Add(historyMouseScrollInterval)
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if runtime.State().CopyMode.Cursor.Row != 0 || len(core.OlderRequests) != 1 || runtime.nextHistoryMouseScrollWakeDelay() != -1 {
		t.Fatal("oldest boundary did not stop autoscroll")
	}
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, rect.Y+rect.H, rect.X+2)
	if runtime.nextHistoryMouseScrollWakeDelay() < 0 {
		t.Fatal("bottom edge not armed")
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.nextHistoryMouseScrollWakeDelay() != -1 {
		t.Fatal("keyboard cancel left timer active")
	}
}

func TestCopyModeMouseEdgeLatePageDoesNotMoveReleasedSelection(t *testing.T) {
	runtime, _, core, _ := mouseAutoScrollFixture(t)
	root := runtime.State()
	viewID := root.CopyMode.ViewID
	root.CopyMode = root.CopyMode.SetMark(state.CopyPosition{Row: 5, Col: 1}).MoveCursor(state.CopyPosition{Row: 0, Col: 1}).RefreshLogicalSelection(root.History)
	root = saveCopyHistorySessionForView(root, viewID)
	reducer := NewCopyModeReducer(CopyModeDeps{Core: core})
	root, effects := reducer(root, CopyModeMouseAutoScrollMsg{ViewID: viewID, Direction: -1})
	if root.History.Pending == nil || len(effects) == 0 {
		t.Fatal("edge did not request older history")
	}
	pending := *root.History.Pending
	if pending.DeferredScrollRows != 0 {
		t.Fatalf("async page carries stale scroll: %d", pending.DeferredScrollRows)
	}
	root, _ = reducer(root, CopyModeMouseSelectMsg{ViewID: viewID, PaneID: root.CopyMode.PaneID, Extend: true, Copy: true, Position: root.CopyMode.Cursor})
	before := *root.CopyMode.Selection
	result := core.OlderResponses[0]
	result.Window.PaneID = pending.PaneID
	result.Window.ViewID = pending.ViewID
	result.Window.EndpointID = pending.EndpointID
	root, _ = reducer(root, CopyModeHistoryResultMsg{ViewID: viewID, PaneID: pending.PaneID, RequestID: pending.ID, TerminalID: pending.TerminalID, Result: result})
	if root.History.Pending != nil || len(root.History.Rows) != 40 {
		t.Fatalf("late page was not applied: pending=%+v rows=%d copy=%+v", root.History.Pending, len(root.History.Rows), root.CopyMode)
	}
	after := root.CopyMode.Selection
	if after.LogicalAnchor != before.LogicalAnchor || after.LogicalFocus != before.LogicalFocus {
		t.Fatalf("late page changed released selection: %+v -> %+v", before, after)
	}
}
