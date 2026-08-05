package app

import (
	"context"
	"github.com/anytty/anytty/tui/testkit"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestHostCursorProjectionLiveCopyPromptFloatingAndOverlayPriority(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 7, Cols: 78, Rows: 20},
	}
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-main",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "alpha", LineID: 10}, {Text: "beta", LineID: 11}},
		)}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core, Rows: 20},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-main",
		Revision:   1,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"main live"},
		Cursor:     state.LiveCursor{Visible: true, Row: 0, Col: 4, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain live cursor: %v", err)
	}
	liveFrame := lastFrame(t, host.Frames())
	liveRect := liveFrame.CursorRect
	if !liveFrame.Cursor.Visible || liveFrame.Cursor.Anchor || liveFrame.Cursor.Shape != render.CursorShapeBar || liveRect.W != 1 || liveRect.H != 1 {
		t.Fatalf("live cursor should project to a visible global cursor rect, cursor=%#v rect=%#v", liveFrame.Cursor, liveRect)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyPageUp}); err != nil {
		t.Fatalf("send copy mode entry: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy mode entry: %v", err)
	}
	copyFrame := lastFrame(t, host.Frames())
	if !copyFrame.Cursor.Visible || copyFrame.Cursor.Anchor || copyFrame.Cursor.Shape != render.CursorShapeBlock || copyFrame.CursorRect == liveRect {
		t.Fatalf("copy mode should own visible block cursor, cursor=%#v rect=%#v live=%#v", copyFrame.Cursor, copyFrame.CursorRect, liveRect)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send copy exit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain copy exit: %v", err)
	}
	if runtime.State().CopyMode.Active {
		t.Fatalf("copy mode should exit before prompt/floating cursor checks, got %#v", runtime.State().CopyMode)
	}

	if err := runtime.Post(ShellOpenPromptMsg{Prompt: createTerminalPrompt(state.DefaultPaneID)}); err != nil {
		t.Fatalf("post prompt: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain prompt: %v", err)
	}
	promptFrame := lastFrame(t, host.Frames())
	if !promptFrame.Cursor.Visible || promptFrame.Cursor.Anchor || promptFrame.Cursor.Shape != render.CursorShapeBar || promptFrame.CursorRect == copyFrame.CursorRect {
		t.Fatalf("prompt overlay should own visible bar cursor, cursor=%#v rect=%#v copy=%#v", promptFrame.Cursor, promptFrame.CursorRect, copyFrame.CursorRect)
	}

	if err := runtime.Post(ShellCloseOverlayMsg{}); err != nil {
		t.Fatalf("close prompt: %v", err)
	}
	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 12, Y: 6, W: 32, H: 9},
		Source:   state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("create floating: %v", err)
	}
	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID:       "term-float",
		TargetFloatingID: "floating-1",
		Result:           port.TerminalAttachResult{TerminalID: "term-float", Channel: 9, Cols: 30, Rows: 7},
	}); err != nil {
		t.Fatalf("attach floating terminal: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-float",
		Revision:   1,
		Cols:       30,
		Rows:       7,
		Lines:      []string{"floating live"},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 5, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post floating surface: %v", err)
	}
	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: "floating-1", Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("focus floating: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating cursor: %v", err)
	}
	floatingFrame := lastFrame(t, host.Frames())
	floatingRect := runtime.State().Shell.ActiveFloatings()[0].Rect
	if !floatingFrame.Cursor.Visible || floatingFrame.CursorRect.X < floatingRect.X+1 || floatingFrame.CursorRect.X >= floatingRect.X+floatingRect.W-1 ||
		floatingFrame.CursorRect.Y < floatingRect.Y+1 || floatingFrame.CursorRect.Y >= floatingRect.Y+floatingRect.H-1 {
		t.Fatalf("active floating terminal should own visible cursor inside floating content, floating=%#v cursor=%#v rect=%#v", floatingRect, floatingFrame.Cursor, floatingFrame.CursorRect)
	}

	if err := runtime.Post(ShellOpenHelpMsg{Section: "most-used"}); err != nil {
		t.Fatalf("open help overlay: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain help overlay: %v", err)
	}
	helpFrame := lastFrame(t, host.Frames())
	if helpFrame.Cursor.Visible || !helpFrame.Cursor.Anchor || helpFrame.CursorRect == floatingFrame.CursorRect || helpFrame.CursorRect.W != 1 || helpFrame.CursorRect.H != 1 {
		t.Fatalf("non-input overlay should override floating with hidden anchor cursor, cursor=%#v rect=%#v floating=%#v", helpFrame.Cursor, helpFrame.CursorRect, floatingFrame.CursorRect)
	}
}

func TestHostCursorProjectionHidesPaneCursorCoveredByFloating(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 7, Cols: 78, Rows: 20}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-main",
		Revision:   1,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"main live"},
		Cursor:     state.LiveCursor{Visible: true, Row: 5, Col: 10, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain live cursor: %v", err)
	}
	liveFrame := lastFrame(t, host.Frames())
	if !liveFrame.Cursor.Visible || liveFrame.CursorRect.W != 1 || liveFrame.CursorRect.H != 1 {
		t.Fatalf("live pane should expose visible cursor before floating cover, cursor=%#v rect=%#v", liveFrame.Cursor, liveFrame.CursorRect)
	}

	coverRect := state.FloatingRect{X: liveFrame.CursorRect.X - 2, Y: liveFrame.CursorRect.Y - 1, W: 24, H: 6}
	if coverRect.X < 0 {
		coverRect.X = 0
	}
	if coverRect.Y < 0 {
		coverRect.Y = 0
	}
	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-cover",
		Pane:     state.PaneState{ID: "floating-cover-pane", Title: "float", Kind: state.PaneEmpty},
		Rect:     coverRect,
		Source:   state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("create floating cover: %v", err)
	}
	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action: state.FloatingCommandDeactivate,
		Source: state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("deactivate floating cover: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating cover: %v", err)
	}

	coveredFrame := lastFrame(t, host.Frames())
	if coveredFrame.Cursor.Visible || !coveredFrame.Cursor.Anchor || coveredFrame.CursorRect != liveFrame.CursorRect {
		t.Fatalf("floating should hide covered pane cursor but keep IME anchor, cursor=%#v rect=%#v live=%#v", coveredFrame.Cursor, coveredFrame.CursorRect, liveFrame.CursorRect)
	}
}

func TestHostCursorProjectionStaysInViewportAfterResize(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 3, Cols: 78, Rows: 20}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-main",
		Revision:   1,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"main live"},
		Cursor:     state.LiveCursor{Visible: true, Row: 19, Col: 77, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial cursor: %v", err)
	}
	if err := host.SendResize(40, 12); err != nil {
		t.Fatalf("send host resize: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-main",
		Revision:   2,
		Cols:       38,
		Rows:       8,
		Lines:      []string{"after resize"},
		Cursor:     state.LiveCursor{Visible: true, Row: 7, Col: 37, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post resized surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resized cursor: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	if !frame.Cursor.Visible || frame.CursorRect.X < 0 || frame.CursorRect.Y < 0 || frame.CursorRect.X >= 40 || frame.CursorRect.Y >= 12 {
		t.Fatalf("resized cursor should remain visible inside viewport, cursor=%#v rect=%#v", frame.Cursor, frame.CursorRect)
	}
}
