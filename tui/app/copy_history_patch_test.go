package app

import (
	"context"
	"github.com/anytty/anytty/tui/testkit"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

func TestCopyHistoryOlderResultUsesIncrementalPatchWhenVisibleContentOnlyShifts(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	root := copyHistoryPerfRoot(80, 24, 256)
	root.CopyMode.ViewportTop = 0
	root.CopyMode.Cursor = state.CopyPosition{}
	root.History.Pending = &state.HistoryPendingRequest{
		ID:                 1,
		Kind:               state.HistoryRequestOlder,
		PaneID:             root.History.PaneID,
		ViewID:             root.History.ViewID,
		TerminalID:         root.History.TerminalID,
		Cols:               root.History.Cols,
		Token:              root.History.Token,
		Generation:         root.History.Generation,
		Cursor:             root.History.Cursor,
		Boundary:           root.History.Boundary,
		DeferredScrollRows: 1,
	}
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		t.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 80, Height: 24}
	runtime.copyHistoryPatch = cache

	window := copyHistoryPerfOlderWindow(64, root.History)
	if err := runtime.Post(CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 1, Window: window}}); err != nil {
		t.Fatalf("post older result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain older result: %v", err)
	}
	if host.sink.patchFrames != 1 {
		t.Fatalf("older result should use one incremental patch, frames=%d patches=%d", host.sink.frames, host.sink.patchFrames)
	}
	if copyHistoryPerfFrame.Patch == nil || !copyHistoryPerfFrame.Patch.Rewrite || len(copyHistoryPerfFrame.Patch.LinesANSI) != root.CopyMode.ViewRows {
		t.Fatalf("expected content-rect rewrite patch for pane history, got %#v", copyHistoryPerfFrame.Patch)
	}
	if copyHistoryPerfFrame.Patch.LineX != 1 || copyHistoryPerfFrame.Patch.LineWidth != root.History.Cols {
		t.Fatalf("rewrite patch must stay inside pane content rect, got %#v", copyHistoryPerfFrame.Patch)
	}
}

func TestCopyHistoryPatchStableIncludesEndpoint(t *testing.T) {
	previous := copyHistoryPatchCache{
		Valid:       true,
		ViewID:      "pane:main",
		EndpointID:  state.DefaultEndpointID,
		TerminalID:  "term-1",
		Token:       "tok-1",
		Cols:        80,
		RowsLen:     20,
		HistoryGen:  7,
		ViewportTop: 1,
		ViewRows:    10,
		ContentRect: render.Rect{W: 80, H: 10},
		Metadata:    render.RenderMetadata{Width: 100, Height: 30},
		Theme:       render.DefaultTheme().WithFallback(),
	}
	current := previous
	current.EndpointID = "west"
	if copyHistoryPatchStable(previous, current) {
		t.Fatalf("same terminal id on different endpoints must not reuse copy-history patch cache")
	}
}

func TestCopyHistoryPatchDisabledWhenFloatingOverlapsContent(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	root := copyHistoryPerfRoot(80, 24, 256)
	root.CopyMode.ViewportTop = 120
	root.CopyMode.Cursor = state.CopyPosition{Row: 120}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 8, Y: 6, W: 30, H: 8},
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandDeactivate})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("deactivate floating: %#v", result)
	}
	runtime := NewAppRuntime(
		root,
		copyHistoryPerfLineScrollReducer(),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		t.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 80, Height: 24}
	runtime.copyHistoryPatch = cache

	root.CopyMode = root.CopyMode.ScrollViewport(-1, len(root.History.Rows))
	root = root.WithCopyHistorySession(root.CopyMode.ViewID, root.History, root.CopyMode)
	runtime.state = root.Advance()
	if runtime.tryRenderCopyHistoryPatch() {
		t.Fatalf("overlapped floating should force complete frame, got patch %#v", copyHistoryPerfFrame.Patch)
	}
	if host.sink.patchFrames != 0 || host.sink.frames != 0 {
		t.Fatalf("disabled patch should not write partial frame, frames=%d patches=%d", host.sink.frames, host.sink.patchFrames)
	}
}

func TestCopyHistoryExitClearsSessionsAndPatchCache(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	core := &testkit.FakeCoreClient{}
	root := copyHistoryPerfRoot(80, 24, 256)
	viewID := root.CopyMode.ViewID
	root = root.WithCopyHistorySession(viewID, root.History, root.CopyMode)
	runtime := NewAppRuntime(
		root,
		ComposeReducers(
			NewBackNavigationReducer(CopyModeDeps{Core: core, Rows: 20}),
			NewCopyModeReducer(CopyModeDeps{Core: core, Rows: 20}),
		),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	if !runtime.copyHistoryPatch.Valid {
		t.Fatal("expected active copy/history frame to seed incremental patch cache")
	}

	if err := runtime.Post(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}}); err != nil {
		t.Fatalf("post esc: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain esc: %v", err)
	}

	next := runtime.State()
	if next.History.TerminalID != "" || next.History.Token != "" || len(next.History.Rows) != 0 || len(next.History.SourceLines) != 0 {
		t.Fatalf("ESC exit must release current history window, got %#v", next.History)
	}
	if next.CopyMode.Active || next.CopyMode.TerminalID != "" || next.CopyMode.BoundToken != "" {
		t.Fatalf("ESC exit must clear copy mode, got %#v", next.CopyMode)
	}
	if _, ok := next.HistoryByView[viewID]; ok {
		t.Fatalf("ESC exit must remove history session for %s, got %#v", viewID, next.HistoryByView)
	}
	if _, ok := next.CopyModeByView[viewID]; ok {
		t.Fatalf("ESC exit must remove copy session for %s, got %#v", viewID, next.CopyModeByView)
	}
	if runtime.copyHistoryPatch.Valid {
		t.Fatalf("ESC exit must clear copy/history patch cache, got %#v", runtime.copyHistoryPatch)
	}
	if len(core.ReleaseRequests) != 1 || core.ReleaseRequests[0].TerminalID != "term-1" || core.ReleaseRequests[0].Token != "tok-perf" {
		t.Fatalf("ESC exit should release frozen history token, got %#v", core.ReleaseRequests)
	}
	if copyHistoryPerfFrame.Patch != nil {
		t.Fatalf("after copy/history exit renderer must write a full frame, got patch %#v", copyHistoryPerfFrame.Patch)
	}
}

func TestCopyHistoryRewritePatchOffsetsANSIColumnAnchors(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	root := copyHistoryPerfRoot(80, 24, 256)
	root.CopyMode.ViewportTop = 120
	root.CopyMode.Cursor = state.CopyPosition{Row: 120}
	root.History.Rows[119] = state.HistoryRow{
		Text:   "099975",
		LineID: 119,
		Cells: []state.HistoryCell{
			{Text: "0", Width: 1},
			{Text: "99975", Width: 5},
		},
	}
	root.History.Rows[120] = state.HistoryRow{Text: "current", LineID: 120}
	runtime := NewAppRuntime(
		root,
		copyHistoryPerfLineScrollReducer(),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		t.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 80, Height: 24}
	runtime.copyHistoryPatch = cache

	root.CopyMode = root.CopyMode.ScrollViewport(-1, len(root.History.Rows))
	root = root.WithCopyHistorySession(root.CopyMode.ViewID, root.History, root.CopyMode)
	runtime.state = root.Advance()
	if !runtime.tryRenderCopyHistoryPatch() {
		t.Fatal("expected rewrite patch")
	}
	patch := copyHistoryPerfFrame.Patch
	if patch == nil || !patch.Rewrite || len(patch.LinesANSI) == 0 {
		t.Fatalf("expected rewrite patch lines, got %#v", patch)
	}
	got := patch.LinesANSI[0]
	if !strings.Contains(got, "0\x1b[3G99975") || strings.Contains(got, "0\x1b[2G99975") {
		t.Fatalf("rewrite patch must offset content-local ANSI anchors, got %q", got)
	}
}

func TestCopyHistoryCursorOnlyMoveUsesCursorPatch(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	root := copyHistoryPerfRoot(80, 24, 256)
	root.CopyMode.ViewportTop = 120
	root.CopyMode.Cursor = state.CopyPosition{Row: 121, Col: 2}
	runtime := NewAppRuntime(
		root,
		copyHistoryPerfLineScrollReducer(),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		t.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 80, Height: 24}
	runtime.copyHistoryPatch = cache

	root.CopyMode = root.CopyMode.ScrollCursor(-1, len(root.History.Rows))
	runtime.state = root.Advance()
	if !runtime.tryRenderCopyHistoryPatch() {
		t.Fatal("expected cursor-only patch")
	}
	patch := copyHistoryPerfFrame.Patch
	if patch == nil || !patch.CursorOnly || patch.Rewrite || patch.Dir != 0 || patch.LineANSI != "" || len(patch.LinesANSI) != 0 {
		t.Fatalf("cursor-only movement should not rewrite history content, got %#v", patch)
	}
	if !copyHistoryPerfFrame.Cursor.Visible || copyHistoryPerfFrame.CursorRect.W != 1 || copyHistoryPerfFrame.CursorRect.H != 1 {
		t.Fatalf("cursor-only patch should carry visible cursor metadata, frame=%#v", copyHistoryPerfFrame)
	}
}

func TestR410CopyHistoryPatchDisabledWhenVisibleRowWouldWrapTTY(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	root := copyHistoryPerfRoot(80, 24, 256)
	root.CopyMode.ViewportTop = 120
	root.CopyMode.Cursor = state.CopyPosition{Row: 120}
	root.History.Rows[119] = state.HistoryRow{Text: strings.Repeat("x", root.History.Cols+1), LineID: 119}
	runtime := NewAppRuntime(
		root,
		copyHistoryPerfLineScrollReducer(),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		t.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 80, Height: 24}
	runtime.copyHistoryPatch = cache

	root.CopyMode = root.CopyMode.ScrollViewport(-1, len(root.History.Rows))
	root = root.WithCopyHistorySession(root.CopyMode.ViewID, root.History, root.CopyMode)
	runtime.state = root.Advance()
	if runtime.tryRenderCopyHistoryPatch() {
		t.Fatalf("wrapping history row must force full-frame render, got patch %#v", copyHistoryPerfFrame.Patch)
	}
	if host.sink.patchFrames != 0 || host.sink.frames != 0 {
		t.Fatalf("unsafe patch should not write partial frame, frames=%d patches=%d", host.sink.frames, host.sink.patchFrames)
	}
}

func TestR411CopyHistoryPatchAllowsScreenRowMetadataInContinuousViewport(t *testing.T) {
	host := newCopyHistoryPerfIncrementalHost(16)
	host.SetSize(80, 24)
	root := copyHistoryPerfRoot(80, 24, 256)
	root.CopyMode.ViewportTop = 120
	root.CopyMode.Cursor = state.CopyPosition{Row: 123}
	root.History.Rows[120] = state.HistoryRow{
		Text:         "OpenAI Codex",
		LineID:       120,
		Kind:         state.HistoryRowKindScreenFrame,
		Segment:      state.HistoryCursorSegmentCurrentPrimaryFrame,
		SessionID:    1,
		FrameID:      10,
		FixedGrid:    true,
		ScreenCols:   root.History.Cols,
		ScreenRow:    3,
		ScreenRowSet: true,
	}
	runtime := NewAppRuntime(
		root,
		copyHistoryPerfLineScrollReducer(),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).RenderANSI(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		t.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 80, Height: 24}
	runtime.copyHistoryPatch = cache

	root.CopyMode = root.CopyMode.ScrollViewport(-1, len(root.History.Rows))
	root = root.WithCopyHistorySession(root.CopyMode.ViewID, root.History, root.CopyMode)
	runtime.state = root.Advance()
	if !runtime.tryRenderCopyHistoryPatch() {
		t.Fatalf("ScreenRow metadata alone should not disable continuous-viewport patch, got patch %#v", copyHistoryPerfFrame.Patch)
	}
	if host.sink.patchFrames != 1 {
		t.Fatalf("continuous viewport should use one partial patch, frames=%d patches=%d", host.sink.frames, host.sink.patchFrames)
	}
	if runtime.state.CopyMode.ViewportTop != 119 || runtime.state.CopyMode.Cursor.Row != 122 {
		t.Fatalf("viewport patch should keep cursor viewport-relative offset, got %#v", runtime.state.CopyMode)
	}
}

func TestCopyHistoryPatchCursorClampsToVisibleViewport(t *testing.T) {
	history := state.HistoryStore{
		Cols: 10,
		Rows: []state.HistoryRow{
			{Text: "one", LineID: 1},
			{Text: "two", LineID: 2},
			{Text: "three", LineID: 3},
			{Text: "four", LineID: 4},
			{Text: "five", LineID: 5},
		},
	}
	copyMode := state.CopyModeStore{
		Active:      true,
		ViewportTop: 1,
		ViewRows:    2,
		Cursor:      state.CopyPosition{Row: 4, Col: 2},
	}
	cursor := copyHistoryPatchCursor(history, copyMode)
	if !cursor.Visible || cursor.Row != 1 {
		t.Fatalf("patch cursor must stay inside visible viewport, got %#v", cursor)
	}
	rect := copyHistoryPatchCursorRect(history, copyMode, render.Rect{X: 4, Y: 6, W: 10, H: 2})
	if rect != (render.Rect{X: 6, Y: 7, W: 1, H: 1}) {
		t.Fatalf("patch cursor rect should remain inside content rect, got %#v", rect)
	}
}
