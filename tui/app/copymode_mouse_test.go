package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/testkit"
)

func TestCopyModeMouseDragCopiesOnRelease(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			runtime, host, clipboard, regions := mouseCopyFixture(t)
			start, end := regions[0], regions[1]
			startCol, endCol := 1, 3
			if reverse {
				start, end = end, start
				startCol, endCol = endCol, startCol
			}
			sendCopyMouse(t, runtime, host, input.MouseLeft, start.Rect.Y+1, start.Rect.X+1+startCol)
			anchor := *runtime.State().CopyMode.Mark
			sendCopyMouse(t, runtime, host, input.MouseLeftDrag, end.Rect.Y+1, end.Rect.X+1+endCol)
			if got := runtime.State().CopyMode.Selection; got == nil || got.Anchor != anchor {
				t.Fatalf("drag reset anchor: %+v", got)
			}
			if got := SelectedText(runtime.State().History, runtime.State().CopyMode); got != "lpha\na好" {
				t.Fatalf("selected %q", got)
			}
			if len(clipboard.Writes) != 0 {
				t.Fatal("copied before release")
			}
			sendCopyMouse(t, runtime, host, input.MouseLeftUp, end.Rect.Y+1, end.Rect.X+1+endCol)
			if clipboard.LastCopy() != "lpha\na好" || len(clipboard.Writes) != 1 {
				t.Fatalf("clipboard: %+v", clipboard)
			}
			if !runtime.State().CopyMode.Active {
				t.Fatal("mouse copy exited history")
			}
			selection := *runtime.State().CopyMode.Selection
			sendCopyMouse(t, runtime, host, input.MouseLeftDrag, start.Rect.Y+1, start.Rect.X+1)
			sendCopyMouse(t, runtime, host, input.MouseLeftUp, start.Rect.Y+1, start.Rect.X+1)
			if len(clipboard.Writes) != 1 || *runtime.State().CopyMode.Selection != selection {
				t.Fatal("stale drag changed selection or copied again")
			}
		})
	}
}

func TestCopyModeMouseClickDoesNotCopyAndLineTailClamps(t *testing.T) {
	runtime, host, clipboard, regions := mouseCopyFixture(t)
	row := regions[0]
	sendCopyMouse(t, runtime, host, input.MouseLeft, row.Rect.Y+1, row.Rect.X+1)
	sendCopyMouse(t, runtime, host, input.MouseLeftUp, row.Rect.Y+1, row.Rect.X+1)
	if len(clipboard.Writes) != 0 {
		t.Fatal("single click copied")
	}
	sendCopyMouse(t, runtime, host, input.MouseLeft, row.Rect.Y+1, row.Rect.X+1)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, row.Rect.Y+1, row.Rect.X+30)
	if got := runtime.State().CopyMode.Cursor.Col; got != 5 {
		t.Fatalf("tail not clamped: %d", got)
	}
	sendCopyMouse(t, runtime, host, input.MouseRelease, row.Rect.Y+1, row.Rect.X+30)
	if clipboard.LastCopy() != "alpha" {
		t.Fatalf("copied %q", clipboard.LastCopy())
	}
}

func TestCopyModeMouseDragCancelsOnKeyboardInput(t *testing.T) {
	runtime, host, clipboard, regions := mouseCopyFixture(t)
	row := regions[0]
	sendCopyMouse(t, runtime, host, input.MouseLeft, row.Rect.Y+1, row.Rect.X+1)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, row.Rect.Y+1, row.Rect.X+4)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc, RawSeq: "\x1b"}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	sendCopyMouse(t, runtime, host, input.MouseLeftUp, row.Rect.Y+1, row.Rect.X+4)
	if len(clipboard.Writes) != 0 {
		t.Fatal("cancelled drag copied")
	}
}

func mouseCopyFixture(t *testing.T) (*AppRuntime, *FakeTerminalHost, *testkit.FakeClipboardService, []render.HitRegion) {
	t.Helper()
	core := &testkit.FakeCoreClient{LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(state.HistoryWindowReplace, "term-1", "tok-1", 78, 7, []state.HistoryRow{
		{Text: "alpha", LineID: 10},
		{Text: "a好bc", LineID: 11, Cells: []state.HistoryCell{{Text: "a", Width: 1}, {Text: "好", Width: 2}, {Text: "b", Width: 1}, {Text: "c", Width: 1}}},
	})}}}
	clipboard := &testkit.FakeClipboardService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 8)
	runtime := newCopyModeRuntime(host, core, clipboard)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	var regions []render.HitRegion
	for _, region := range lastFrame(t, host.Frames()).HitRegions {
		if region.Kind == render.HitRegionHistoryRow {
			regions = append(regions, region)
		}
	}
	if len(regions) != 2 {
		t.Fatalf("history regions: %+v", regions)
	}
	return runtime, host, clipboard, regions
}

func sendCopyMouse(t *testing.T, runtime *AppRuntime, host *FakeTerminalHost, button input.MouseButton, row, col int) {
	t.Helper()
	code, final := 0, "M"
	switch button {
	case input.MouseLeftDrag:
		code = 32
	case input.MouseLeftUp, input.MouseRelease:
		final = "m"
	}
	event := input.InputEvent{Kind: input.EventKindMouse, Mouse: button, Row: row, Col: col, RawSeq: fmt.Sprintf("\x1b[<%d;%d;%d%s", code, col, row, final)}
	if err := host.SendInput(event); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCopyModeMouseDragStaysInOriginalPane(t *testing.T) {
	runtime, host, clipboard, regions := mouseCopyFixture(t)
	row := regions[0]
	sendCopyMouse(t, runtime, host, input.MouseLeft, row.Rect.Y+1, row.Rect.X+1)
	// A foreground history region from another pane must not steal this drag.
	foreign := row
	foreign.PaneID = "other-pane"
	foreign.Rect.X += 40
	foreign.Row = 99
	runtime.lastHitRegions = append(runtime.lastHitRegions, foreign)
	sendCopyMouse(t, runtime, host, input.MouseLeftDrag, foreign.Rect.Y+1, foreign.Rect.X+1)
	if got := runtime.State().CopyMode.Cursor; got != (state.CopyPosition{Row: 0, Col: 5}) {
		t.Fatalf("drag escaped original pane: %+v", got)
	}
	sendCopyMouse(t, runtime, host, input.MouseLeftUp, foreign.Rect.Y+1, foreign.Rect.X+1)
	if clipboard.LastCopy() != "alpha" {
		t.Fatalf("copied %q", clipboard.LastCopy())
	}
}

func TestCopyModeMouseReleaseAfterSessionReplacementDoesNotCopy(t *testing.T) {
	runtime, host, clipboard, regions := mouseCopyFixture(t)
	row := regions[0]
	sendCopyMouse(t, runtime, host, input.MouseLeft, row.Rect.Y+1, row.Rect.X+1)
	runtime.mouseDrag.HistoryToken = "expired-token"
	sendCopyMouse(t, runtime, host, input.MouseLeftUp, row.Rect.Y+1, row.Rect.X+4)
	if runtime.mouseDrag.Active || len(clipboard.Writes) != 0 {
		t.Fatal("stale session drag must be discarded")
	}
}
