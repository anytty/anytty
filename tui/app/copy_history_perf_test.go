package app

import (
	"context"
	"fmt"
	"github.com/anytty/anytty/tui/testkit"
	"io"
	"testing"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/terminalhost"
)

var copyHistoryPerfFrame render.Frame

func BenchmarkCopyHistoryRenderScrollFrame(b *testing.B) {
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.CopyMode.ViewportTop = 2500 + (i % 200)
		copyHistoryPerfFrame = renderer.Render(builder.Build(root))
	}
}

func BenchmarkCopyHistoryRenderANSIFrame(b *testing.B) {
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.CopyMode.ViewportTop = 2500 + (i % 200)
		copyHistoryPerfFrame = renderer.RenderANSI(builder.Build(root))
	}
}

func BenchmarkCopyHistoryFrameSinkScrollShift(b *testing.B) {
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	sink := terminalhost.NewFrameSink(io.Discard)
	root.CopyMode.ViewportTop = 2500
	if err := sink.WriteFrame(renderer.Render(builder.Build(root))); err != nil {
		b.Fatalf("write initial frame: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.CopyMode.ViewportTop = 2501 + (i % 200)
		if err := sink.WriteFrame(renderer.Render(builder.Build(root))); err != nil {
			b.Fatalf("write scrolled frame: %v", err)
		}
	}
}

func BenchmarkCopyHistoryFrameSinkScrollShiftANSI(b *testing.B) {
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	sink := terminalhost.NewFrameSink(io.Discard)
	root.CopyMode.ViewportTop = 2500
	if err := sink.WriteFrame(renderer.RenderANSI(builder.Build(root))); err != nil {
		b.Fatalf("write initial frame: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.CopyMode.ViewportTop = 2501 + (i % 200)
		if err := sink.WriteFrame(renderer.RenderANSI(builder.Build(root))); err != nil {
			b.Fatalf("write scrolled frame: %v", err)
		}
	}
}

func BenchmarkCopyHistoryRuntimeWheelBatch(b *testing.B) {
	host := NewFakeTerminalHost(4096)
	host.SetSize(180, 60)
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return renderer.Render(builder.Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	benchmarkCopyHistoryWheelRuntime(b, runtime, host)
}

func BenchmarkCopyHistoryRuntimeWheelBatchANSI(b *testing.B) {
	host := NewFakeTerminalHost(4096)
	host.SetSize(180, 60)
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return renderer.RenderANSI(builder.Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	benchmarkCopyHistoryWheelRuntime(b, runtime, host)
}

func BenchmarkCopyHistoryRuntimeWheelBatchIncremental(b *testing.B) {
	host := newCopyHistoryPerfIncrementalHost(4096)
	host.SetSize(180, 60)
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return renderer.RenderANSI(builder.Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	benchmarkCopyHistoryWheelRuntimeWithSender(b, runtime, host.SendInput, func() int {
		return host.sink.patchFrames
	})
}

func BenchmarkCopyHistoryRuntimeWheelAlternatingIncremental(b *testing.B) {
	host := newCopyHistoryPerfIncrementalHost(4096)
	host.SetSize(180, 60)
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return renderer.RenderANSI(builder.Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	benchmarkCopyHistoryAlternatingWheelRuntime(b, runtime, host.SendInput, func() int {
		return host.sink.patchFrames
	})
}

func BenchmarkCopyHistoryRuntimeLoadedLineScrollIncremental(b *testing.B) {
	host := newCopyHistoryPerfIncrementalHost(4096)
	host.SetSize(180, 60)
	root := copyHistoryPerfRoot(180, 60, 5000)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	runtime := NewAppRuntime(
		root,
		copyHistoryPerfLineScrollReducer(),
		func(root state.Root) render.Frame {
			return renderer.RenderANSI(builder.Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	benchmarkCopyHistoryAlternatingWheelRuntime(b, runtime, host.SendInput, func() int {
		return host.sink.patchFrames
	})
}

func BenchmarkCopyHistoryRuntimeOlderResultIncremental(b *testing.B) {
	host := newCopyHistoryPerfIncrementalHost(4096)
	host.SetSize(180, 60)
	builder := render.NewRenderVMBuilder()
	renderer := render.NewRenderer(render.DefaultTheme())
	root := copyHistoryPerfRoot(180, 60, 8192)
	root.CopyMode.ViewportTop = 0
	root.CopyMode.Cursor = state.CopyPosition{Col: 8}
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
	window := copyHistoryPerfOlderWindow(128, root.History)
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return renderer.RenderANSI(builder.Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	cache, ok := buildCopyHistoryPatchCache(root, render.DefaultTheme())
	if !ok {
		b.Fatal("expected copy history patch cache")
	}
	cache.Metadata = render.RenderMetadata{Width: 180, Height: 60}
	msg := CopyModeHistoryResultMsg{Result: port.HistoryResult{RequestID: 1, Window: window}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runtime.state = root
		runtime.copyHistoryPatch = cache
		if err := runtime.Post(msg); err != nil {
			b.Fatalf("post older result: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			b.Fatalf("drain older result: %v", err)
		}
	}
	b.StopTimer()
	if host.sink.patchFrames == 0 {
		b.Fatalf("older result benchmark did not write patch frames")
	}
}

func benchmarkCopyHistoryWheelRuntime(b *testing.B, runtime *AppRuntime, host *FakeTerminalHost) {
	b.Helper()
	benchmarkCopyHistoryWheelRuntimeWithSender(b, runtime, host.SendInput, nil)
}

func benchmarkCopyHistoryWheelRuntimeWithSender(b *testing.B, runtime *AppRuntime, send func(input.InputEvent) error, patchCount func() int) {
	b.Helper()
	for i := 0; i < 100; i++ {
		if err := send(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
			b.Fatalf("seed input: %v", err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		b.Fatalf("seed drain: %v", err)
	}
	beforePatches := 0
	if patchCount != nil {
		beforePatches = patchCount()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := send(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseWheelUp}); err != nil {
			b.Fatalf("send input: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			b.Fatalf("drain: %v", err)
		}
	}
	b.StopTimer()
	if patchCount != nil && patchCount() == beforePatches {
		b.Fatalf("incremental benchmark did not write patch frames")
	}
}

func benchmarkCopyHistoryAlternatingWheelRuntime(b *testing.B, runtime *AppRuntime, send func(input.InputEvent) error, patchCount func() int) {
	b.Helper()
	runtime.renderFrame()
	beforePatches := 0
	if patchCount != nil {
		beforePatches = patchCount()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mouse := input.MouseWheelUp
		if i%2 == 1 {
			mouse = input.MouseWheelDown
		}
		if err := send(input.InputEvent{Kind: input.EventKindMouse, Mouse: mouse}); err != nil {
			b.Fatalf("send input: %v", err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			b.Fatalf("drain: %v", err)
		}
	}
	b.StopTimer()
	if patchCount != nil && patchCount() == beforePatches {
		b.Fatalf("incremental benchmark did not write patch frames")
	}
}

type copyHistoryPerfIncrementalHost struct {
	*FakeTerminalHost
	sink *copyHistoryPerfIncrementalSink
}

func newCopyHistoryPerfIncrementalHost(buffer int) *copyHistoryPerfIncrementalHost {
	fake := NewFakeTerminalHost(buffer)
	return &copyHistoryPerfIncrementalHost{FakeTerminalHost: fake, sink: &copyHistoryPerfIncrementalSink{}}
}

func (host *copyHistoryPerfIncrementalHost) FrameSink() render.FrameSink {
	return host.sink
}

type copyHistoryPerfIncrementalSink struct {
	patchFrames int
	frames      int
}

func (sink *copyHistoryPerfIncrementalSink) NeedsCompleteFrame() bool {
	return false
}

func (sink *copyHistoryPerfIncrementalSink) WriteFrame(frame render.Frame) error {
	sink.frames++
	if frame.Patch != nil {
		sink.patchFrames++
	}
	copyHistoryPerfFrame = frame
	return nil
}

func copyHistoryPerfLineScrollReducer() Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		inputMsg, ok := msg.(InputMsg)
		if !ok || inputMsg.Event.Kind != input.EventKindMouse {
			return root, nil
		}
		switch inputMsg.Event.Mouse {
		case input.MouseWheelUp:
			root.CopyMode = root.CopyMode.ScrollCursor(-1, len(root.History.Rows))
			return root.Advance(), nil
		case input.MouseWheelDown:
			root.CopyMode = root.CopyMode.ScrollCursor(1, len(root.History.Rows))
			return root.Advance(), nil
		default:
			return root, nil
		}
	}
}

func copyHistoryPerfRoot(cols int, rows int, historyRows int) state.Root {
	history := state.HistoryStore{
		PaneID:     state.DefaultPaneID,
		ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID: "term-1",
		Token:      "tok-perf",
		Cols:       cols - 2,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 1},
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 1, LastLineID: uint64(historyRows)},
		HasMore:    true,
	}
	history.SourceLines = make([]state.HistoryLogicalLine, historyRows)
	for i := range history.SourceLines {
		history.SourceLines[i] = state.HistoryLogicalLine{
			Text:   fmt.Sprintf("copy-history-perf row %05d segment segment segment segment segment", i),
			LineID: uint64(i + 1),
		}
	}
	history.Rows, history.Lines = state.ReflowHistoryLogicalLines(history.SourceLines, history.Cols)
	viewportTop := len(history.Rows) / 2
	copyMode := state.CopyModeStore{
		Active:      true,
		PaneID:      state.DefaultPaneID,
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		TerminalID:  "term-1",
		ViewportTop: viewportTop,
		ViewRows:    rows - 4,
		Cursor:      state.CopyPosition{Row: viewportTop},
		BoundToken:  "tok-perf",
		BoundCols:   history.Cols,
	}
	return state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: cols, Rows: rows},
		Shell:    state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		Session: state.TerminalSessionStore{
			TerminalID: "term-1",
			Attached:   true,
			Cols:       cols - 2,
			Rows:       rows - 4,
		},
		Surface:       state.TerminalSurfaceStore{TerminalID: "term-1", Cols: cols - 2, Rows: rows - 4, Lines: []string{"live"}},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 4, cols-2, rows-4, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)),
		History:       history,
		CopyMode:      copyMode,
	}
}

func copyHistoryPerfOlderWindow(olderLines int, base state.HistoryStore) state.HistoryWindow {
	firstLineID := base.Boundary.FirstLineID - uint64(olderLines)
	source := make([]state.HistoryLogicalLine, olderLines)
	for i := range source {
		lineID := firstLineID + uint64(i)
		source[i] = state.HistoryLogicalLine{
			Text:   fmt.Sprintf("copy-history-perf older %05d segment segment segment segment segment", lineID),
			LineID: lineID,
		}
	}
	rows, spans := state.ReflowHistoryLogicalLines(source, base.Cols)
	return state.HistoryWindow{
		PaneID:      base.PaneID,
		ViewID:      base.ViewID,
		TerminalID:  base.TerminalID,
		Token:       base.Token,
		Op:          state.HistoryWindowPrepend,
		Cols:        base.Cols,
		SourceLines: source,
		Rows:        rows,
		Lines:       spans,
		Cursor:      state.HistoryCursor{Valid: true, BeforeLineID: firstLineID},
		Generation:  base.Generation,
		Boundary:    state.HistoryBoundary{FirstLineID: firstLineID, LastLineID: base.Boundary.LastLineID},
		HasMore:     true,
	}
}
