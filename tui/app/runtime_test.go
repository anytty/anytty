package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/anytty/anytty/tui/testkit"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

type testMsg struct {
	Name string
}

func (testMsg) isMsg() {}

type skipRenderTestMsg struct {
	Name string
}

func (skipRenderTestMsg) isMsg() {}

func (skipRenderTestMsg) SkipRender() bool { return true }

type recordingRuntimeEffectRunner struct {
	Effects []Effect
}

func (runner *recordingRuntimeEffectRunner) Run(_ context.Context, effect Effect, _ func(Msg)) {
	runner.Effects = append(runner.Effects, effect)
}

func (runner *recordingRuntimeEffectRunner) Cancel(CancelToken) {}

type retainingMsg struct {
	Payload []byte
}

func (retainingMsg) isMsg() {}

type blockingCompletionSink struct {
	frames      []render.Frame
	completions []chan render.FrameWriteCompletion
}

func (sink *blockingCompletionSink) WriteFrame(frame render.Frame) error {
	sink.frames = append(sink.frames, frame.Clone())
	return nil
}

func (sink *blockingCompletionSink) WriteFrameWithCompletion(frame render.Frame) (<-chan render.FrameWriteCompletion, error) {
	done := make(chan render.FrameWriteCompletion, 1)
	sink.frames = append(sink.frames, frame.Clone())
	sink.completions = append(sink.completions, done)
	return done, nil
}

func (sink *blockingCompletionSink) NeedsCompleteFrame() bool { return false }

type blockingCompletionHost struct {
	sink *blockingCompletionSink
}

func (host blockingCompletionHost) Size() (int, int, error)              { return 0, 0, nil }
func (host blockingCompletionHost) InputEvents() <-chan input.InputEvent { return nil }
func (host blockingCompletionHost) EventsReady() <-chan struct{}         { return nil }
func (host blockingCompletionHost) FrameSink() render.FrameSink          { return host.sink }

type staticFrameSinkHost struct {
	sink render.FrameSink
}

func (host staticFrameSinkHost) Size() (int, int, error)              { return 0, 0, nil }
func (host staticFrameSinkHost) InputEvents() <-chan input.InputEvent { return nil }
func (host staticFrameSinkHost) EventsReady() <-chan struct{}         { return nil }
func (host staticFrameSinkHost) FrameSink() render.FrameSink          { return host.sink }

type failingFrameSink struct {
	err    error
	writes int
}

func (sink *failingFrameSink) WriteFrame(render.Frame) error {
	sink.writes++
	return sink.err
}

func waitForRuntimeQueue(t *testing.T, runtime *AppRuntime) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		runtime.mu.Lock()
		queued := len(runtime.queue) > 0
		runtime.mu.Unlock()
		if queued {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for runtime completion message")
}

func TestRuntimeContractsDoNotUseBubbleTea(t *testing.T) {
	var _ Msg = NoopMsg{}
	var _ Effect = NoopEffect{}
}

func TestAppRuntimeBuildsOnlyLatestFrameWhileWriterIsBusy(t *testing.T) {
	sink := &blockingCompletionSink{}
	effectRuns := 0
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, _ Msg) (state.Root, []Effect) {
			root = root.Advance()
			return root, []Effect{FuncEffect{Run: func(context.Context) Msg {
				effectRuns++
				return nil
			}}}
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("revision=%d", root.Generation)}}
		},
		blockingCompletionHost{sink: sink},
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(testMsg{Name: "first"}); err != nil {
		t.Fatalf("post first: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first: %v", err)
	}
	if len(sink.frames) != 1 || !runtime.frameWriteInFlight {
		t.Fatalf("expected one frame in flight, frames=%d in_flight=%v", len(sink.frames), runtime.frameWriteInFlight)
	}

	for index := 0; index < 100; index++ {
		if err := runtime.Post(testMsg{Name: fmt.Sprintf("update-%d", index)}); err != nil {
			t.Fatalf("post update: %v", err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain updates: %v", err)
	}
	if len(sink.frames) != 1 {
		t.Fatalf("writer busy must suppress intermediate frame builds, got %d frames", len(sink.frames))
	}
	if runtime.State().Generation != 101 || effectRuns != 101 {
		t.Fatalf("reducer/effects must continue while writing, generation=%d effects=%d", runtime.State().Generation, effectRuns)
	}

	sink.completions[0] <- render.FrameWriteCompletion{Written: true}
	close(sink.completions[0])
	waitForRuntimeQueue(t, runtime)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain completion: %v", err)
	}
	if len(sink.frames) != 2 {
		t.Fatalf("completion must build exactly one latest frame, got %d", len(sink.frames))
	}
	if got := sink.frames[1].Lines; !reflect.DeepEqual(got, []string{"revision=101"}) {
		t.Fatalf("second frame is not latest state: %#v", got)
	}
}

func TestAppRuntimeDroppedFrameReleasesWriteGate(t *testing.T) {
	sink := &blockingCompletionSink{}
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, _ Msg) (state.Root, []Effect) { return root.Advance(), nil },
		func(root state.Root) render.Frame { return render.Frame{Lines: []string{fmt.Sprint(root.Generation)}} },
		blockingCompletionHost{sink: sink},
		NewSyncEffectRunner(),
	)

	_ = runtime.Post(testMsg{Name: "first"})
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first: %v", err)
	}
	sink.completions[0] <- render.FrameWriteCompletion{Written: false}
	close(sink.completions[0])
	waitForRuntimeQueue(t, runtime)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain dropped completion: %v", err)
	}
	if !runtime.frameWriteInFlight || len(sink.frames) != 2 {
		t.Fatalf("Written=false must retry the latest state with a full frame, in_flight=%v frames=%d", runtime.frameWriteInFlight, len(sink.frames))
	}
	sink.completions[1] <- render.FrameWriteCompletion{Written: true}
	close(sink.completions[1])
	waitForRuntimeQueue(t, runtime)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain retry completion: %v", err)
	}

	_ = runtime.Post(testMsg{Name: "second"})
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain second: %v", err)
	}
	if len(sink.frames) != 3 {
		t.Fatalf("next update must render after dropped completion, got %d frames", len(sink.frames))
	}
}

func TestAppRuntimeReturnsFrameSinkErrorWithoutRetryLoop(t *testing.T) {
	want := errors.New("tty write failed")
	sink := &failingFrameSink{err: want}
	renderCalls := 0
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, _ Msg) (state.Root, []Effect) { return root.Advance(), nil },
		func(state.Root) render.Frame {
			renderCalls++
			return render.Frame{Lines: []string{"frame"}}
		},
		staticFrameSinkHost{sink: sink},
		NewSyncEffectRunner(),
	)
	_ = runtime.Post(testMsg{Name: "render"})
	err := runtime.Run(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Run must return the underlying sink error, got %v", err)
	}
	if sink.writes != 1 || renderCalls != 1 || runtime.frameWriteInFlight || runtime.renderPending {
		t.Fatalf("sink error must terminate without retry loop, writes=%d renders=%d in_flight=%v pending=%v", sink.writes, renderCalls, runtime.frameWriteInFlight, runtime.renderPending)
	}
}

func TestAppRuntimeCommitsVisualBaselineOnlyAfterWriteCompletion(t *testing.T) {
	sink := &blockingCompletionSink{}
	hit := render.HitRegion{PaneID: "pane-1"}
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, _ Msg) (state.Root, []Effect) { return root.Advance(), nil },
		func(state.Root) render.Frame {
			return render.Frame{Lines: []string{"frame"}, HitRegions: []render.HitRegion{hit}}
		},
		blockingCompletionHost{sink: sink},
		NewSyncEffectRunner(),
	)
	_ = runtime.Post(testMsg{Name: "render"})
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	if len(runtime.lastHitRegions) != 0 {
		t.Fatalf("in-flight proposal must not advance presented baseline: hits=%#v", runtime.lastHitRegions)
	}
	sink.completions[0] <- render.FrameWriteCompletion{Written: true}
	close(sink.completions[0])
	waitForRuntimeQueue(t, runtime)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain completion: %v", err)
	}
	if !reflect.DeepEqual(runtime.lastHitRegions, []render.HitRegion{hit}) {
		t.Fatalf("successful completion must commit proposal: hits=%#v", runtime.lastHitRegions)
	}
}

func TestAppRuntimeFrameCompletionOnlyReleasesWriter(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, NewFakeTerminalHost(1), NewSyncEffectRunner())
	runtime.frameWriteInFlight = true
	runtime.renderPending = true

	if runtime.prepareRuntimeMessage(frameWriteCompletedMsg{Written: true}) {
		t.Fatal("frame completion should not run reducers")
	}
	if runtime.frameWriteInFlight {
		t.Fatal("frame completion must release the write gate")
	}
	if _, ok := runtime.dequeue(); ok {
		t.Fatal("physical completion must not control live-screen requests")
	}
}

func TestAppRuntimeContinuousLiveUpdatesBuildCompleteLogicalFrames(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			if live, ok := msg.(LiveScreenNextResultMsg); ok {
				root.Surface = root.Surface.ApplySnapshot(live.Snapshot)
			}
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: append([]string(nil), root.Surface.Lines...), Metadata: render.RenderMetadata{Width: 20, Height: 8}}
		},
		host,
		NewSyncEffectRunner(),
	)

	for revision, line := range []string{"first", "second"} {
		if err := runtime.Post(LiveScreenNextResultMsg{
			EndpointID: state.DefaultEndpointID,
			TerminalID: "term-1",
			Snapshot: state.LiveSurfaceSnapshot{
				EndpointID:  state.DefaultEndpointID,
				TerminalID:  "term-1",
				Revision:    uint64(revision + 1),
				FullReplace: true,
				Cols:        20,
				Rows:        8,
				Lines:       []string{line},
			},
		}); err != nil {
			t.Fatalf("post live update %d: %v", revision, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain live update %d: %v", revision, err)
		}
	}

	frames := host.Frames()
	if len(frames) != 2 {
		t.Fatalf("expected one logical frame per live update, got %#v", frames)
	}
	for index, frame := range frames {
		if frame.Patch != nil {
			t.Fatalf("continuous live frame %d must be complete, got patch %#v", index, frame.Patch)
		}
	}
	if frames[0].Lines[0] != "first" || frames[1].Lines[0] != "second" {
		t.Fatalf("complete frames must carry the latest surfaces, got %#v", frames)
	}
}

func TestAppRuntimeMixedCopyAndLiveDirtyForcesFullFrame(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.copyHistoryPatch = copyHistoryPatchCache{Valid: true}
	runtime.noteRenderPending(testMsg{Name: "copy-scroll"})
	if !runtime.copyHistoryPatch.Valid {
		t.Fatal("copy-only dirty state should keep its incremental baseline")
	}
	runtime.noteRenderPending(LiveScreenNextResultMsg{TerminalID: "term-1", Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1", BaseRevision: 7, Revision: 8, Cols: 80, Rows: 24, ChangedRows: []int{1},
	}})
	if !runtime.forceFullFrame || !runtime.copyHistoryPatch.Valid || !runtime.renderPending {
		t.Fatalf("live dirtiness must select a complete logical frame without destroying the copy baseline: force_full=%v copy=%#v pending=%v", runtime.forceFullFrame, runtime.copyHistoryPatch, runtime.renderPending)
	}
}

func TestAsyncEffectRunnerSerialKeyPreservesEffectOrder(t *testing.T) {
	runner := NewAsyncEffectRunner()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startedFirst := make(chan struct{})
	releaseFirst := make(chan struct{})
	startedSecond := make(chan struct{})
	posted := make(chan string, 2)
	post := func(msg Msg) {
		if test, ok := msg.(testMsg); ok {
			posted <- test.Name
		}
	}

	runner.Run(ctx, FuncEffect{
		Async:     true,
		SerialKey: "terminal.input:term-1:view-1:7",
		Run: func(context.Context) Msg {
			close(startedFirst)
			<-releaseFirst
			return testMsg{Name: "first"}
		},
	}, post)
	select {
	case <-startedFirst:
	case <-time.After(time.Second):
		t.Fatal("first serial effect did not start")
	}

	runner.Run(ctx, FuncEffect{
		Async:     true,
		SerialKey: "terminal.input:term-1:view-1:7",
		Run: func(context.Context) Msg {
			close(startedSecond)
			return testMsg{Name: "second"}
		},
	}, post)
	select {
	case <-startedSecond:
		t.Fatal("second serial effect started before first completed")
	case <-time.After(20 * time.Millisecond):
	}

	close(releaseFirst)
	for _, want := range []string{"first", "second"} {
		select {
		case got := <-posted:
			if got != want {
				t.Fatalf("serial effect order changed, got %q want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s serial effect", want)
		}
	}
	select {
	case <-startedSecond:
	case <-time.After(time.Second):
		t.Fatal("second serial effect did not start after first completed")
	}
}

func TestAppRuntimeDequeClearsProcessedMessageReferences(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.enqueue(retainingMsg{Payload: make([]byte, 1024)})
	runtime.enqueue(retainingMsg{Payload: make([]byte, 2048)})

	if _, ok := runtime.dequeue(); !ok {
		t.Fatal("expected first queued message")
	}
	retained := runtime.queue[:cap(runtime.queue)]
	if len(runtime.queue) != 1 || retained[1] != nil {
		t.Fatalf("dequeue must clear stale tail reference, len=%d retained=%#v", len(runtime.queue), retained)
	}

	if _, ok := runtime.dequeue(); !ok {
		t.Fatal("expected second queued message")
	}
	retained = runtime.queue[:cap(runtime.queue)]
	for i, msg := range retained {
		if msg != nil {
			t.Fatalf("queue backing array kept processed message at index %d: %#v", i, msg)
		}
	}
}

func TestAppRuntimeCoalescedLiveScreenNextClearsStalePayloadReference(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.queue = make([]Msg, 0, 2)
	runtime.enqueue(LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}})
	runtime.enqueue(LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}})

	if len(runtime.queue) != 1 {
		t.Fatalf("live screen nexts should coalesce to one message, queue=%#v", runtime.queue)
	}
	retained := runtime.queue[:cap(runtime.queue)]
	for i := len(runtime.queue); i < len(retained); i++ {
		if retained[i] != nil {
			t.Fatalf("coalesced live screen next kept stale queue slot %d: %#v", i, retained[i])
		}
	}
}

func TestAppRuntimeCoalescedLiveSurfaceClearsStalePayloadReference(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.queue = make([]Msg, 0, 2)
	runtime.enqueue(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 1, Lines: []string{strings.Repeat("old", 1024)}}})
	runtime.enqueue(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 2, Lines: []string{"latest"}}})

	if len(runtime.queue) != 1 {
		t.Fatalf("live surfaces should coalesce to one message, queue=%#v", runtime.queue)
	}
	retained := runtime.queue[:cap(runtime.queue)]
	for i := len(runtime.queue); i < len(retained); i++ {
		if retained[i] != nil {
			t.Fatalf("coalesced live surface kept stale queue slot %d: %#v", i, retained[i])
		}
	}
}

func TestAppRuntimePassiveRefreshDoesNotStartSecondScreenSource(t *testing.T) {
	terminal := &testkit.FakeTerminalService{SurfaceResult: port.TerminalSurfaceResult{
		Ready: true,
		Snapshot: state.LiveSurfaceSnapshot{
			TerminalID: "term-1",
			Revision:   12,
			Lines:      []string{"latest"},
		},
	}}
	root := state.Root{}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   10,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"ready"},
	})
	runtime := NewAppRuntime(root, newLiveReducerPrepared(LiveDeps{Terminal: terminal}), nil, nil, NewSyncEffectRunner())

	if err := runtime.Post(LiveEventMsg{Event: port.TerminalLiveEvent{
		TerminalID: "term-1",
		Refresh:    true,
		Snapshot:   state.LiveSurfaceSnapshot{Revision: 11},
	}}); err != nil {
		t.Fatalf("post wake: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   11,
		Lines:      []string{"side surface"},
	}, RequestedCols: 80, RequestedRows: 24}); err != nil {
		t.Fatalf("post surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Surfaces) != 0 || runtime.State().Surface.Revision != 11 {
		t.Fatalf("passive hint must not fetch; explicit surface result must still reduce, requests=%#v surface=%#v", terminal.Surfaces, runtime.State().Surface)
	}
}

func TestInteractiveRuntimeAppliesConfiguredPanelPresentation(t *testing.T) {
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine),
			Config: state.TUIConfigStore{Chrome: state.TUIChromeConfig{
				PanelPresentation: string(state.PanelPresentationCard),
			}},
		},
		NewFakeTerminalHost(1),
		NewSyncEffectRunner(),
		LiveDeps{},
		CopyModeDeps{},
	)

	if got := runtime.State().Shell.EnsureDefaults().PanelPresentation; got != state.PanelPresentationCard {
		t.Fatalf("runtime should seed shell presentation from config, got %q", got)
	}
}

func TestInteractiveRuntimeAppliesConfiguredPaneChromeGlyphs(t *testing.T) {
	render.ResetPaneChromeGlyphs()
	defer render.ResetPaneChromeGlyphs()

	host := NewFakeTerminalHost(8)
	root := state.Root{
		Shell: state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical),
		Config: state.TUIConfigStore{Chrome: state.TUIChromeConfig{
			PaneGlyphs: state.TUIPaneChromeGlyphsConfig{
				Zoom:            "—",
				SplitVertical:   "□",
				SplitHorizontal: "▭",
				Close:           "⤫",
			},
		}},
	}
	host.SetSize(80, 20)
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{}, CopyModeDeps{})
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}
	frame := lastRuntimeFrame(t, host)
	if !frameContains(frame, "[—]") || !frameContains(frame, "[□]") || !frameContains(frame, "[▭]") || !frameContains(frame, "[⤫]") {
		t.Fatalf("configured pane glyphs should render in pane chrome, got %#v", frame.Lines)
	}
}

func TestAppRuntimePrioritizesInputBeforeQueuedOrdinaryLiveUpdate(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.enqueue(LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}})
	runtime.enqueue(InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x16", Ctrl: true}})

	msg, ok := runtime.dequeue()
	if !ok {
		t.Fatal("expected queued input")
	}
	if _, ok := msg.(InputMsg); !ok {
		t.Fatalf("input should be processed before ordinary live update, got %#v", msg)
	}
}

func TestAppRuntimeCoalescesQueuedWorkbenchStorageRequests(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.enqueue(testMsg{Name: "before"})
	runtime.enqueue(WorkbenchStorageLoadRequestMsg{})
	runtime.enqueue(WorkbenchStoragePersistRequestMsg{Reason: "move"})
	runtime.enqueue(WorkbenchStorageLoadRequestMsg{})
	runtime.enqueue(WorkbenchStoragePersistRequestMsg{Reason: "resize"})
	runtime.enqueue(testMsg{Name: "after"})

	if len(runtime.queue) != 4 {
		t.Fatalf("workbench storage requests should coalesce in queue, got %#v", runtime.queue)
	}
	if _, ok := runtime.queue[0].(testMsg); !ok {
		t.Fatalf("ordinary message before coalesced storage requests should keep order, queue=%#v", runtime.queue)
	}
	if _, ok := runtime.queue[1].(WorkbenchStorageLoadRequestMsg); !ok {
		t.Fatalf("load request should stay at first load position, queue=%#v", runtime.queue)
	}
	persist, ok := runtime.queue[2].(WorkbenchStoragePersistRequestMsg)
	if !ok || persist.Reason != "resize" {
		t.Fatalf("persist request should keep latest reason and one queue slot, queue=%#v", runtime.queue)
	}
	if after, ok := runtime.queue[3].(testMsg); !ok || after.Name != "after" {
		t.Fatalf("ordinary message after coalesced storage requests should keep order, queue=%#v", runtime.queue)
	}
}

func TestAppRuntimeDiagnosticsRespectsEnvironmentToggle(t *testing.T) {
	t.Setenv(tuiDiagnosticsEnv, "")
	t.Setenv(tuiInputTraceEnv, "")
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if runtime.diagnostics == nil || runtime.diagnostics.enabled || runtime.diagnostics.inputTraceEnabled {
		t.Fatalf("diagnostics should be present but disabled by default, got %#v", runtime.diagnostics)
	}

	t.Setenv(tuiDiagnosticsEnv, "1")
	runtime.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if runtime.diagnostics == nil || !runtime.diagnostics.enabled || !runtime.diagnostics.inputTraceEnabled {
		t.Fatalf("diagnostics should be enabled by %s, got %#v", tuiDiagnosticsEnv, runtime.diagnostics)
	}

	t.Setenv(tuiDiagnosticsEnv, "")
	t.Setenv(tuiInputTraceEnv, "1")
	runtime.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if runtime.diagnostics == nil || runtime.diagnostics.enabled || !runtime.diagnostics.inputTraceEnabled {
		t.Fatalf("input trace should be independently enabled by %s, got %#v", tuiInputTraceEnv, runtime.diagnostics)
	}
}

func TestAppRuntimeDiagnosticsWritesRequestedHeapProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tuiHeapProfileDirEnv, dir)
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	runtime.diagnostics.RequestHeapProfile(runtime.State(), "usr1")

	files, err := filepath.Glob(filepath.Join(dir, "tui-usr1-*.pprof"))
	if err != nil {
		t.Fatalf("glob heap profiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one heap profile, got %d files=%#v", len(files), files)
	}
}

func TestAppRuntimeDiagnosticsWritesRequestedMemstats(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tuiMemstatsDirEnv, dir)
	t.Setenv(tuiMemstatsStageEnv, "copy-oldest")
	runtime := NewAppRuntime(state.Root{}, nil, nil, nil, nil)
	runtime.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	runtime.RequestMemstats("usr2")

	data, err := os.ReadFile(filepath.Join(dir, "memstats.tsv"))
	if err != nil {
		t.Fatalf("read memstats: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "process\tstage") || !strings.Contains(text, "\ttui\tcopy-oldest\tusr2\t") {
		t.Fatalf("unexpected memstats tsv:\n%s", text)
	}
}

func TestAppPackageDoesNotImportBubbleTea(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob app files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imported := range parsed.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if path == "github.com/charmbracelet/bubbletea" || strings.Contains(path, "/bubbles") {
				t.Fatalf("%s imports Bubble Tea contract package %s", file, path)
			}
		}
	}
}

func TestComposeReducersStopsAtHandledEffect(t *testing.T) {
	var seen []string
	reducer := ComposeReducers(
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, "first")
			return root.Advance(), []Effect{handledEffect{}}
		},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, "second")
			return root.Advance(), []Effect{NoopEffect{}}
		},
	)

	root, effects := reducer(state.Root{}, NoopMsg{})
	if root.Generation != 1 {
		t.Fatalf("expected only first reducer to run, got generation %d", root.Generation)
	}
	if !reflect.DeepEqual(seen, []string{"first"}) {
		t.Fatalf("unexpected reducer sequence %v", seen)
	}
	if len(effects) != 0 {
		t.Fatalf("handled marker must not leak as effect %#v", effects)
	}
}

func TestAppRuntimeProcessesMessagesInOrderAndRenders(t *testing.T) {
	host := NewFakeTerminalHost(4)
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, msg.(testMsg).Name)
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{string(rune('0' + root.Generation))}}
		},
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(testMsg{Name: "first"}); err != nil {
		t.Fatalf("post first: %v", err)
	}
	if err := runtime.Post(testMsg{Name: "second"}); err != nil {
		t.Fatalf("post second: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(seen, []string{"first", "second"}) {
		t.Fatalf("unexpected message order %v", seen)
	}
	if runtime.State().Generation != 2 {
		t.Fatalf("expected generation 2, got %d", runtime.State().Generation)
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"2"}) {
		t.Fatalf("unexpected rendered frames %v", got)
	}
}

func TestAppRuntimeCoalescesMessageBurstIntoOneFrame(t *testing.T) {
	host := NewFakeTerminalHost(4)
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{string(rune('0' + root.Generation))}}
		},
		host,
		NewSyncEffectRunner(),
	)

	const burstMessages = 128
	for i := 0; i < burstMessages; i++ {
		if err := runtime.Post(testMsg{Name: "burst"}); err != nil {
			t.Fatalf("post burst: %v", err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := len(host.Frames()); got != 1 {
		t.Fatalf("burst should render only the final coalesced frame, got %d", got)
	}
	if runtime.State().Generation != burstMessages {
		t.Fatalf("expected all burst messages processed, got generation %d", runtime.State().Generation)
	}
}

func TestAppRuntimeSkipRenderMessageDoesNotWriteStaleFrame(t *testing.T) {
	host := NewFakeTerminalHost(4)
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			switch msg := msg.(type) {
			case skipRenderTestMsg:
				seen = append(seen, msg.Name)
			case testMsg:
				seen = append(seen, msg.Name)
			default:
				t.Fatalf("unexpected message %T", msg)
			}
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(skipRenderTestMsg{Name: "refresh"}); err != nil {
		t.Fatalf("post refresh: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain refresh: %v", err)
	}
	if runtime.State().Generation != 1 || len(host.Frames()) != 0 {
		t.Fatalf("skip-render message should update state without writing a stale frame, generation=%d frames=%#v", runtime.State().Generation, host.Frames())
	}

	if err := runtime.Post(testMsg{Name: "surface"}); err != nil {
		t.Fatalf("post surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain surface: %v", err)
	}
	if !reflect.DeepEqual(seen, []string{"refresh", "surface"}) {
		t.Fatalf("unexpected message order %v", seen)
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"frame-2"}) {
		t.Fatalf("ordinary message should render latest state after skipped frame, got %v", got)
	}
}

func TestAppRuntimeContinuesDrainForMessagesArrivingDuringFrameWrite(t *testing.T) {
	host := NewFakeTerminalHost(4)
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)
	host.sink.onWrite = func() {
		if len(host.sink.frames) == 0 {
			_ = runtime.Post(testMsg{Name: "during-write"})
		}
	}

	if err := runtime.Post(testMsg{Name: "first"}); err != nil {
		t.Fatalf("post first: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"frame-1", "frame-2"}) {
		t.Fatalf("drain should render next frame after write-time message arrival, got %v", got)
	}
}

func TestAppRuntimeWritesFirstFrameBeforeStartupBurstFinishes(t *testing.T) {
	host := NewFakeTerminalHost(4)
	host.SetSize(100, 30)
	var runtime *AppRuntime
	remaining := 8
	runtime = NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			root = root.Advance()
			if remaining > 0 {
				remaining--
				return root, []Effect{FuncEffect{Run: func(context.Context) Msg { return testMsg{Name: "startup"} }}}
			}
			return root, nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)
	host.sink.onWrite = func() {
		if len(host.sink.frames) == 0 && remaining != 7 {
			t.Fatalf("startup must write first frame before draining startup burst, remaining=%d", remaining)
		}
	}

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := frameLines(host.Frames()); len(got) != 2 || got[0] != "frame-2" || got[1] != "frame-10" {
		t.Fatalf("expected immediate first frame and final coalesced frame, got %v", got)
	}
}

func TestAppRuntimeRendersBoundedBatchBeforeQueueBecomesEmpty(t *testing.T) {
	host := NewFakeTerminalHost(4)
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			root = root.Advance()
			return root, []Effect{FuncEffect{Run: func(context.Context) Msg { return testMsg{Name: "again"} }}}
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)
	runtime.maxMessagesPerBatch = 3
	if err := runtime.Post(testMsg{Name: "start"}); err != nil {
		t.Fatalf("post: %v", err)
	}

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"frame-3"}) {
		t.Fatalf("bounded batch should render before queue is empty, got %v", got)
	}
}

func TestAppRuntimeCoalescesOrdinaryNativeScreenFetchResultsByTerminal(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var revisions []uint64
	var lines []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			surface, ok := msg.(LiveSurfaceMsg)
			if !ok {
				t.Fatalf("expected LiveSurfaceMsg, got %T", msg)
			}
			revisions = append(revisions, surface.Snapshot.Revision)
			if len(surface.Snapshot.Lines) > 0 {
				lines = append(lines, surface.Snapshot.Lines[0])
			}
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)

	for _, snapshot := range []state.LiveSurfaceSnapshot{
		{TerminalID: "term-1", Revision: 1, Lines: []string{"one"}},
		{TerminalID: "term-1", Revision: 2, Lines: []string{"two"}},
		{TerminalID: "term-1", Revision: 3, Lines: []string{"three"}},
	} {
		if err := runtime.Post(LiveSurfaceMsg{Snapshot: snapshot}); err != nil {
			t.Fatalf("post revision %d: %v", snapshot.Revision, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(revisions, []uint64{3}) {
		t.Fatalf("ordinary native screen fetch results should be latest-only, got %v", revisions)
	}
	if !reflect.DeepEqual(lines, []string{"three"}) {
		t.Fatalf("ordinary native screen fetch payload should keep only latest, got %v", lines)
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"frame-1"}) {
		t.Fatalf("coalesced native screen result should render once, got %v", got)
	}
}

func TestAppRuntimeDoesNotCoalesceNativeScreenFetchResultsAcrossOrdinaryBoundary(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			switch msg := msg.(type) {
			case LiveSurfaceMsg:
				seen = append(seen, fmt.Sprintf("surface:%d", msg.Snapshot.Revision))
			case testMsg:
				seen = append(seen, msg.Name)
			default:
				t.Fatalf("unexpected message %T", msg)
			}
			return root.Advance(), nil
		},
		nil,
		host,
		NewSyncEffectRunner(),
	)

	for _, msg := range []Msg{
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 1, Lines: []string{"one"}}},
		testMsg{Name: "ordinary"},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 2, Lines: []string{"two"}}},
	} {
		if err := runtime.Post(msg); err != nil {
			t.Fatalf("post %T: %v", msg, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if want := []string{"surface:1", "ordinary", "surface:2"}; !reflect.DeepEqual(seen, want) {
		t.Fatalf("native screen fetch result order changed, got %v want %v", seen, want)
	}
}

func TestAppRuntimeDoesNotCoalesceLifecycleNativeScreenFetchResult(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var revisions []uint64
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			surface, ok := msg.(LiveSurfaceMsg)
			if !ok {
				t.Fatalf("expected LiveSurfaceMsg, got %T", msg)
			}
			revisions = append(revisions, surface.Snapshot.Revision)
			return root.Advance(), nil
		},
		nil,
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 3, Lines: []string{"new"}}}); err != nil {
		t.Fatalf("post revision 3: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 2, Lines: []string{"old"}}, LifecycleKnown: true}); err != nil {
		t.Fatalf("post revision 2: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(revisions, []uint64{3, 2}) {
		t.Fatalf("lifecycle native screen result must stay as semantic boundary, got %v", revisions)
	}
}

func TestAppRuntimeDoesNotCoalesceReadyLiveEventsByTerminalID(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var revisions []uint64
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			event, ok := msg.(LiveEventMsg)
			if !ok {
				t.Fatalf("expected LiveEventMsg, got %T", msg)
			}
			revisions = append(revisions, event.Event.Snapshot.Revision)
			return root.Advance(), nil
		},
		nil,
		host,
		NewSyncEffectRunner(),
	)

	for _, revision := range []uint64{1, 2, 3} {
		if err := runtime.Post(LiveEventMsg{Event: port.TerminalLiveEvent{
			TerminalID: "term-1",
			Ready:      true,
			Snapshot:   state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: revision, Lines: []string{fmt.Sprintf("r%d", revision)}},
		}}); err != nil {
			t.Fatalf("post revision %d: %v", revision, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(revisions, []uint64{1, 2, 3}) {
		t.Fatalf("ready live events are semantic boundaries and must stay ordered, got %v", revisions)
	}
}

func TestAppRuntimeCoalescesQueuedLiveRefreshEventsByTerminalID(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			event, ok := msg.(LiveEventMsg)
			if !ok {
				t.Fatalf("expected LiveEventMsg, got %T", msg)
			}
			seen = append(seen, fmt.Sprintf("%s:%t", event.Event.TerminalID, event.Event.Refresh))
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)

	for i := 0; i < 3; i++ {
		if err := runtime.Post(LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}}); err != nil {
			t.Fatalf("post refresh %d: %v", i, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(seen, []string{"term-1:true"}) {
		t.Fatalf("expected only latest queued refresh event, got %v", seen)
	}
	if len(host.Frames()) != 0 {
		t.Fatalf("queued refresh invalidation should not render before latest surface arrives, got %#v", host.Frames())
	}
}

func TestAppRuntimeSchedulesDirtyLiveFetchAfterSurfaceReturn(t *testing.T) {
	host := NewFakeTerminalHost(8)
	liveDeps := LiveDeps{Terminal: &testkit.FakeTerminalService{
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Revision:   3,
				Lines:      []string{"latest"},
			},
		},
	}}
	reducer := newLiveReducerPrepared(liveDeps)
	runtime := NewAppRuntime(
		state.Root{Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Refreshes: map[string]state.LiveSurfaceRefreshState{
				"term-1": {InFlight: true, Dirty: true, Cols: 80, Rows: 24},
			},
		}},
		reducer,
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{root.Surface.Lines[0]}}
		},
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Lines:      []string{"middle"},
	}, RequestedCols: 80, RequestedRows: 24}); err != nil {
		t.Fatalf("post middle surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"latest"}) {
		t.Fatalf("dirty live follow-up should coalesce middle surface before frame write, got %v", got)
	}
	if runtime.State().Surface.Revision != 3 {
		t.Fatalf("latest surface should win, got %#v", runtime.State().Surface)
	}
}

func TestAppRuntimeRequestsNextScreenAtFrameSubmission(t *testing.T) {
	host := NewFakeTerminalHost(8)
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	liveDeps := LiveDeps{Terminal: terminal}
	runner := &recordingRuntimeEffectRunner{}
	runtime := NewAppRuntime(
		state.Root{Surface: state.TerminalSurfaceStore{TerminalID: "term-1"}},
		newLiveReducerPrepared(liveDeps),
		func(root state.Root) render.Frame {
			return render.Frame{
				Lines:       []string{root.Surface.Lines[0]},
				LiveTargets: []render.LiveRenderTarget{{TerminalID: root.Surface.TerminalID, Revision: root.Surface.Revision}},
			}
		},
		host,
		runner,
	)

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   12,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"ready"},
	}}); err != nil {
		t.Fatalf("post surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(runner.Effects) != 1 {
		t.Fatalf("frame submission should schedule one async next-screen effect, got %#v", runner.Effects)
	}
	arm, ok := runner.Effects[0].(FuncEffect)
	if !ok || arm.Token != liveScreenNextTokenForRef(state.LocalTerminalRef("term-1")) || !arm.Async {
		t.Fatalf("expected terminal-scoped async next-screen effect, got %#v", runner.Effects[0])
	}
}

func TestAppRuntimeUsesOnlyTargetsSelectedByFrame(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	runner := &recordingRuntimeEffectRunner{}
	root := state.Root{}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 12, Lines: []string{"ready"}})
	runtime := NewAppRuntime(root, newLiveReducerPrepared(LiveDeps{Terminal: terminal}), nil, NewFakeTerminalHost(8), runner)
	done := make(chan render.FrameWriteCompletion, 1)
	done <- render.FrameWriteCompletion{Written: true}
	close(done)

	runtime.enqueueLiveScreenFrameSelected(render.Frame{LiveTargets: []render.LiveRenderTarget{{TerminalID: "term-1", Revision: 12}}}, true)
	runtime.trackFrameCompletion(done, nil)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(runner.Effects) != 1 {
		t.Fatalf("selected live target should request its next screen, got %#v", runner.Effects)
	}
	arm, ok := runner.Effects[0].(FuncEffect)
	if !ok || arm.Token != liveScreenNextTokenForRef(state.LocalTerminalRef("term-1")) || !arm.Async {
		t.Fatalf("expected terminal-scoped async next-screen effect, got %#v", runner.Effects[0])
	}
}

func TestAppRuntimeNetworkRequestDoesNotWaitForPhysicalWriteSuccess(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	runner := &recordingRuntimeEffectRunner{}
	root := state.Root{}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 12, Lines: []string{"ready"}})
	runtime := NewAppRuntime(root, newLiveReducerPrepared(LiveDeps{Terminal: terminal}), nil, NewFakeTerminalHost(8), runner)
	done := make(chan render.FrameWriteCompletion, 1)
	done <- render.FrameWriteCompletion{Written: false}
	close(done)

	runtime.enqueueLiveScreenFrameSelected(render.Frame{LiveTargets: []render.LiveRenderTarget{{TerminalID: "term-1", Revision: 12}}}, true)
	runtime.trackFrameCompletion(done, nil)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(runner.Effects) != 1 {
		t.Fatalf("next-screen wait must overlap physical output and not depend on completion, got %#v", runner.Effects)
	}
}

func TestAppRuntimeRequestsAllVisibleLiveTerminalsAtSubmission(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	runner := &recordingRuntimeEffectRunner{}
	root := state.Root{Shell: state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical)}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 10, Lines: []string{"one"}})
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-2", Revision: 20, Lines: []string{"two"}})
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewPaneTerminalView("pane-2", "term-2", 8, 80, 24, state.TerminalResizeRoleOwner, "surface-2", state.TerminalPaneViewID("pane-2"), true))
	runtime := NewAppRuntime(root, newLiveReducerPrepared(LiveDeps{Terminal: terminal}), nil, NewFakeTerminalHost(8), runner)
	done := make(chan render.FrameWriteCompletion, 1)
	done <- render.FrameWriteCompletion{Written: true}
	close(done)

	runtime.enqueueLiveScreenFrameSelected(render.Frame{LiveTargets: []render.LiveRenderTarget{
		{TerminalID: "term-1", Revision: 10},
		{TerminalID: "term-2", Revision: 20},
	}}, true)
	runtime.trackFrameCompletion(done, nil)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if len(runner.Effects) != 2 {
		t.Fatalf("frame submission should request every visible terminal once, got %#v", runner.Effects)
	}
}

func TestAppRuntimeDirtyLiveFetchIgnoresServiceLifecycleFlagFromOrdinaryRefresh(t *testing.T) {
	host := NewFakeTerminalHost(8)
	liveDeps := LiveDeps{Terminal: &testkit.FakeTerminalService{
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Revision:   3,
				Lines:      []string{"latest"},
			},
			LifecycleKnown: true,
		},
	}}
	reducer := newLiveReducerPrepared(liveDeps)
	runtime := NewAppRuntime(
		state.Root{Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Refreshes: map[string]state.LiveSurfaceRefreshState{
				"term-1": {InFlight: true, Dirty: true, Cols: 80, Rows: 24},
			},
		}},
		reducer,
		func(root state.Root) render.Frame {
			return render.Frame{Lines: append([]string(nil), root.Surface.Lines...)}
		},
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Lines:      []string{"middle"},
	}, RequestedCols: 80, RequestedRows: 24}); err != nil {
		t.Fatalf("post ordinary surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"latest"}) {
		t.Fatalf("ordinary refresh follow-up should coalesce middle surface before frame write, got %v", got)
	}
	if runtime.State().Surface.State == state.TerminalLiveExited {
		t.Fatalf("ordinary refresh fetch must not become lifecycle boundary, got %#v", runtime.State().Surface)
	}
}

func TestAppRuntimeDoesNotSkipLifecycleLiveSurfaceUnderPressure(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runtime := NewAppRuntime(
		state.Root{Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Refreshes: map[string]state.LiveSurfaceRefreshState{
				"term-1": {InFlight: true, Dirty: true, Cols: 80, Rows: 24},
			},
		}},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			switch msg := msg.(type) {
			case LiveSurfaceMsg:
				root.Surface = root.Surface.ApplySnapshotWithLifecycle(msg.Snapshot, msg.LifecycleKnown)
				return root.Advance(), nil
			case LiveScreenFrameSelectedMsg:
				return root, nil
			default:
				t.Fatalf("expected LiveSurfaceMsg, got %T", msg)
			}
			return root, nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{string(root.Surface.State)}}
		},
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(LiveSurfaceMsg{
		Snapshot: state.LiveSurfaceSnapshot{
			TerminalID: "term-1",
			Revision:   2,
			State:      state.TerminalLiveExited,
			ExitReason: "exited",
		},
		LifecycleKnown: true,
	}); err != nil {
		t.Fatalf("post lifecycle surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{string(state.TerminalLiveExited)}) {
		t.Fatalf("lifecycle surface must render even while dirty refresh is pending, got %v", got)
	}
}

func TestAppRuntimeKeepsLiveSemanticBoundariesBetweenQueuedUpdates(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			switch msg := msg.(type) {
			case LiveSurfaceMsg:
				seen = append(seen, fmt.Sprintf("surface:%d", msg.Snapshot.Revision))
			case LiveResizeMsg:
				seen = append(seen, fmt.Sprintf("resize:%dx%d", msg.Cols, msg.Rows))
			case HostResizeMsg:
				seen = append(seen, fmt.Sprintf("host-resize:%dx%d", msg.Cols, msg.Rows))
			case LiveResizeResultMsg:
				seen = append(seen, fmt.Sprintf("resize-result:%dx%d", msg.Cols, msg.Rows))
			case LiveExitMsg:
				seen = append(seen, "exit")
			case LiveAttachMsg:
				seen = append(seen, "attach")
			default:
				t.Fatalf("unexpected message %T", msg)
			}
			return root.Advance(), nil
		},
		nil,
		host,
		NewSyncEffectRunner(),
	)

	for _, msg := range []Msg{
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 1, Lines: []string{"before-resize"}}},
		LiveResizeMsg{Cols: 100, Rows: 40},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 2, Lines: []string{"after-resize"}}},
		HostResizeMsg{Cols: 120, Rows: 48},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 3, Lines: []string{"after-host-resize"}}},
		LiveResizeResultMsg{Cols: 118, Rows: 44},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 4, Lines: []string{"after-resize-result"}}},
		LiveExitMsg{TerminalID: "term-1", ExitCode: 0, Reason: "done"},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 5, Lines: []string{"after-exit"}}},
		LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 6, Lines: []string{"after-attach"}}},
	} {
		if err := runtime.Post(msg); err != nil {
			t.Fatalf("post %T: %v", msg, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	want := []string{"surface:1", "resize:100x40", "surface:2", "host-resize:120x48", "surface:3", "resize-result:118x44", "surface:4", "exit", "surface:5", "attach", "surface:6"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("live coalescing crossed semantic boundary: got %v want %v", seen, want)
	}
}

func TestAppRuntimeLiveCoalescingIsEndpointScoped(t *testing.T) {
	host := NewFakeTerminalHost(8)
	var seen []state.TerminalRef
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			switch msg := msg.(type) {
			case LiveSurfaceMsg:
				seen = append(seen, msg.Snapshot.TerminalRef())
			default:
				t.Fatalf("unexpected message %T", msg)
			}
			return root.Advance(), nil
		},
		nil,
		host,
		NewSyncEffectRunner(),
	)

	for _, msg := range []Msg{
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Revision: 1, Lines: []string{"local"}}},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{EndpointID: "west", TerminalID: "term-1", Revision: 2, Lines: []string{"west"}}},
	} {
		if err := runtime.Post(msg); err != nil {
			t.Fatalf("post %T: %v", msg, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	want := []state.TerminalRef{state.NewTerminalRef(state.DefaultEndpointID, "term-1"), state.NewTerminalRef("west", "term-1")}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("same daemon-local terminal ids on different endpoints must not coalesce, got %#v want %#v", seen, want)
	}
}

func TestLiveScreenSelectedTargetsPreserveEndpoint(t *testing.T) {
	targets := []render.LiveRenderTarget{{EndpointID: string(state.DefaultEndpointID), TerminalID: "term-1", Revision: 3}, {EndpointID: "west", TerminalID: "term-1", Revision: 8}}
	runtime := NewAppRuntime(state.Root{}, nil, nil, NewFakeTerminalHost(1), NewSyncEffectRunner())
	runtime.enqueueLiveScreenFrameSelected(render.Frame{LiveTargets: targets}, true)
	msg, ok := runtime.dequeue()
	selected, selectedOK := msg.(LiveScreenFrameSelectedMsg)
	if !ok || !selectedOK || !selected.Full || !reflect.DeepEqual(selected.Targets, targets) {
		t.Fatalf("selected frame must preserve endpoint refs, got %#v", msg)
	}
}

func TestLiveScreenSubmissionUsesOnlyTargetsPresentedByFrame(t *testing.T) {
	runtime := NewAppRuntime(state.Root{}, nil, nil, NewFakeTerminalHost(1), NewSyncEffectRunner())
	targets := []render.LiveRenderTarget{{
		TerminalID: "term-local",
		Revision:   3,
	}}
	runtime.enqueueLiveScreenFrameSelected(render.Frame{LiveTargets: targets}, true)

	msg, ok := runtime.dequeue()
	if !ok {
		t.Fatal("presented live target should be selected")
	}
	selected, ok := msg.(LiveScreenFrameSelectedMsg)
	if !ok || !reflect.DeepEqual(selected.Targets, targets) {
		t.Fatalf("submission must follow the frame's live targets, got %#v", msg)
	}
	if _, ok := runtime.dequeue(); ok {
		t.Fatal("surfaces absent from the frame must not become live demand")
	}
}

func TestAppRuntimeVisibleLocalInputContinuesAfterHiddenRemoteSurface(t *testing.T) {
	host := NewFakeTerminalHost(8)
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	root := state.Root{Shell: state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-local")}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{EndpointID: state.DefaultEndpointID, TerminalID: "term-local", Revision: 1, Lines: []string{"ready"}})
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{EndpointID: "west", TerminalID: "remote-hidden", Revision: 9, Lines: []string{"hidden"}})
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewEndpointPaneTerminalView(state.DefaultEndpointID, state.DefaultPaneID, "term-local", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-local", state.TerminalPaneViewID(state.DefaultPaneID), true))
	runtime := NewAppRuntime(
		root,
		ComposeReducers(newLiveReducerPrepared(LiveDeps{Terminal: terminal}), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal})),
		func(root state.Root) render.Frame {
			return render.Frame{
				Lines:       append([]string(nil), root.Surface.Lines...),
				LiveTargets: []render.LiveRenderTarget{{EndpointID: string(root.Surface.EndpointID), TerminalID: root.Surface.TerminalID, Revision: root.Surface.Revision}},
			}
		},
		host,
		NewAsyncEffectRunner(),
	)

	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial frame: %v", err)
	}
	if err := waitForLiveScreenNextRequest(context.Background(), runtime, terminal, "term-local"); err != nil {
		t.Fatalf("local frame should arm live screen next: %v", err)
	}
	requests := terminal.LiveScreenNextRequestsSnapshot()
	if len(requests) != 1 || requests[0].EndpointID != state.DefaultEndpointID {
		t.Fatalf("hidden remote cache must not create an arm request, got %#v", requests)
	}

	terminal.SurfaceResult = port.TerminalSurfaceResult{
		Ready:    true,
		Snapshot: state.LiveSurfaceSnapshot{EndpointID: state.DefaultEndpointID, TerminalID: "term-local", Revision: 2, Lines: []string{"typed"}},
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", RawSeq: "x"}); err != nil {
		t.Fatalf("send input: %v", err)
	}
	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{EndpointID: state.DefaultEndpointID, TerminalID: "term-local", Revision: 2, Cols: 80, Rows: 24, Lines: []string{"typed"}, FullReplace: true}}
	if err := drainUntilFrameContains(context.Background(), runtime, host, "typed"); err != nil {
		t.Fatalf("input wake should fetch and render latest local surface: %v", err)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].EndpointID != state.DefaultEndpointID || terminal.Inputs[0].TerminalID != "term-local" {
		t.Fatalf("input must stay on visible local terminal, got %#v", terminal.Inputs)
	}
}

func TestAppRuntimeRoutesEffectResultsThroughMessagePath(t *testing.T) {
	host := NewFakeTerminalHost(4)
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, msg.(testMsg).Name)
			if msg.(testMsg).Name == "start" {
				return root.Advance(), []Effect{FuncEffect{
					Run: func(context.Context) Msg {
						return testMsg{Name: "done"}
					},
				}}
			}
			return root.Advance(), nil
		},
		nil,
		host,
		NewSyncEffectRunner(),
	)

	if err := runtime.Post(testMsg{Name: "start"}); err != nil {
		t.Fatalf("post start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(seen, []string{"start", "done"}) {
		t.Fatalf("effect result did not return through message path: %v", seen)
	}
}

func TestAppRuntimeSupportsBatchAndCancel(t *testing.T) {
	runner := NewSyncEffectRunner()
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, msg.(testMsg).Name)
			if msg.(testMsg).Name == "start" {
				return root, []Effect{
					CancelEffect{Token: "drop"},
					BatchEffect{Effects: []Effect{
						FuncEffect{Token: "keep", Run: func(context.Context) Msg { return testMsg{Name: "kept"} }},
						FuncEffect{Token: "drop", Run: func(context.Context) Msg { return testMsg{Name: "dropped"} }},
					}},
				}
			}
			return root, nil
		},
		nil,
		nil,
		runner,
	)

	if err := runtime.Post(testMsg{Name: "start"}); err != nil {
		t.Fatalf("post start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(seen, []string{"start", "kept"}) {
		t.Fatalf("unexpected batch/cancel sequence %v", seen)
	}
}

func TestAppRuntimeIngestsTerminalHostInput(t *testing.T) {
	host := NewFakeTerminalHost(2)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey}); err != nil {
		t.Fatalf("send input: %v", err)
	}
	var seen []input.EventKind
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			inputMsg, ok := msg.(InputMsg)
			if !ok {
				t.Fatalf("expected InputMsg, got %T", msg)
			}
			seen = append(seen, inputMsg.Event.Kind)
			return root.Advance(), nil
		},
		nil,
		host,
		nil,
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(seen, []input.EventKind{input.EventKindKey}) {
		t.Fatalf("unexpected host input events %v", seen)
	}
}

func TestAppRuntimeBatchesPlainTerminalInputBytes(t *testing.T) {
	host := NewFakeTerminalHost(8)
	root := state.Root{Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(
		state.DefaultPaneID,
		"term-1",
		7,
		80,
		24,
		state.TerminalResizeRoleOwner,
		"surface-1",
		state.TerminalPaneViewID(state.DefaultPaneID),
		true,
	))
	for _, ch := range "abc" {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: string(ch)}); err != nil {
			t.Fatalf("send input %q: %v", ch, err)
		}
	}
	var batches []string
	runtime := NewAppRuntime(
		root,
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			batch, ok := msg.(TerminalInputBytesMsg)
			if !ok {
				t.Fatalf("expected TerminalInputBytesMsg, got %T", msg)
			}
			batches = append(batches, string(batch.Bytes))
			return root, nil
		},
		nil,
		host,
		nil,
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !reflect.DeepEqual(batches, []string{"abc"}) {
		t.Fatalf("plain terminal input should batch by drain, got %v", batches)
	}
}

func TestAppRuntimeDoesNotBatchInputWithoutTerminalBinding(t *testing.T) {
	host := NewFakeTerminalHost(2)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send input: %v", err)
	}
	var sawInput bool
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			if _, ok := msg.(InputMsg); !ok {
				t.Fatalf("expected InputMsg without terminal binding, got %T", msg)
			}
			sawInput = true
			return root, nil
		},
		nil,
		host,
		nil,
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !sawInput {
		t.Fatal("expected unbound key to remain a normal input message")
	}
}

func TestAppRuntimeIngestsHostThemeWithoutTerminalInputLeak(t *testing.T) {
	host := NewFakeTerminalHost(2)
	if err := host.SendInput(input.InputEvent{
		Kind:  input.EventKindHostTheme,
		Theme: input.HostThemeEvent{DefaultFG: "#aabbcc", PaletteIndex: 5, PaletteColor: "#445566"},
	}); err != nil {
		t.Fatalf("send host theme: %v", err)
	}
	var leaked bool
	runtime := NewAppRuntime(
		state.Root{},
		ComposeReducers(NewShellReducer(), func(root state.Root, msg Msg) (state.Root, []Effect) {
			if _, ok := msg.(InputMsg); ok {
				leaked = true
			}
			return root, nil
		}),
		func(state.Root) render.Frame { return render.Frame{} },
		host,
		NewSyncEffectRunner(),
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	got := runtime.State().HostTheme
	if got.DefaultFG != "#aabbcc" {
		t.Fatalf("expected host default fg update, got %#v", got)
	}
	if color, ok := got.PaletteColor(5); !ok || color != "#445566" {
		t.Fatalf("expected palette update, got %#v ok=%v", color, ok)
	}
	if leaked {
		t.Fatal("host theme event must not leak as terminal InputMsg")
	}
}

func TestAppRuntimeIngestsHostKeyboardCapabilityWithoutInputLeak(t *testing.T) {
	host := NewFakeTerminalHost(2)
	if err := host.SendInput(input.InputEvent{
		Kind:       input.EventKindHostCapability,
		Capability: input.HostCapabilityEvent{KeyboardDisambiguation: true},
	}); err != nil {
		t.Fatalf("send host capability: %v", err)
	}
	var leaked bool
	runtime := NewAppRuntime(
		state.Root{},
		ComposeReducers(NewShellReducer(), func(root state.Root, msg Msg) (state.Root, []Effect) {
			if _, ok := msg.(InputMsg); ok {
				leaked = true
			}
			return root, nil
		}),
		func(state.Root) render.Frame { return render.Frame{} },
		host,
		NewSyncEffectRunner(),
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	got := runtime.State().HostCapabilities
	if !got.KeyboardDisambiguation {
		t.Fatalf("expected keyboard disambiguation capability, got %#v", got)
	}
	if leaked {
		t.Fatal("host capability event must not leak as terminal InputMsg")
	}
}

func TestAppRuntimeSuppressesUnknownHostControlWithoutInputLeak(t *testing.T) {
	host := NewFakeTerminalHost(1)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindHostControl, RawSeq: "\x1b]52;ignored\x07"}); err != nil {
		t.Fatalf("send host control: %v", err)
	}
	var leaked bool
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			if _, ok := msg.(InputMsg); ok {
				leaked = true
			}
			return root, nil
		},
		func(state.Root) render.Frame { return render.Frame{} },
		host,
		NewSyncEffectRunner(),
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if leaked {
		t.Fatal("unknown host control must not become InputMsg")
	}
}

func TestAppRuntimeInitializesViewportFromHostSizeAndRenders(t *testing.T) {
	host := NewFakeTerminalHost(4)
	host.SetSize(132, 43)
	var seen []Msg
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, msg)
			if root.Viewport.Cols != 132 || root.Viewport.Rows != 43 {
				t.Fatalf("reducer must see updated viewport before message handling, got %#v", root.Viewport)
			}
			return root, nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{viewportLabel(root)}}
		},
		host,
		nil,
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := runtime.State().Viewport; !got.Valid || got.Cols != 132 || got.Rows != 43 {
		t.Fatalf("expected viewport initialized from host size, got %#v", got)
	}
	if len(seen) != 1 {
		t.Fatalf("expected one initial HostResizeMsg through reducer chain, got %d", len(seen))
	}
	if _, ok := seen[0].(HostResizeMsg); !ok {
		t.Fatalf("expected HostResizeMsg, got %T", seen[0])
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"132x43"}) {
		t.Fatalf("expected initial viewport render, got %v", got)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("second drain: %v", err)
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"132x43"}) {
		t.Fatalf("initial size must be ingested once, got %v", got)
	}
}

func TestAppRuntimeIngestsHostResizeEventsAndDeduplicates(t *testing.T) {
	host := NewFakeTerminalHost(4)
	var seen []Msg
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, msg)
			return root, nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{viewportLabel(root)}}
		},
		host,
		nil,
	)

	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}
	if len(host.Frames()) != 0 {
		t.Fatalf("invalid initial host size must not render, got %#v", host.Frames())
	}
	if err := host.SendResize(90, 30); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := host.SendResize(90, 30); err != nil {
		t.Fatalf("send duplicate resize: %v", err)
	}
	if err := host.SendResize(100, 32); err != nil {
		t.Fatalf("send second resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := runtime.State().Viewport; !got.Valid || got.Cols != 100 || got.Rows != 32 {
		t.Fatalf("expected latest viewport, got %#v", got)
	}
	if len(seen) != 1 {
		t.Fatalf("expected queued host resizes to collapse to latest viewport before reducer, got %d messages", len(seen))
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"100x32"}) {
		t.Fatalf("expected resize drain to render latest viewport once, got %v", got)
	}
}

func TestAppRuntimeRunWakesOnPostedMessageWithoutPollingSleep(t *testing.T) {
	host := NewFakeTerminalHost(4)
	frameWritten := make(chan struct{}, 1)
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			return root.Advance(), nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{fmt.Sprintf("frame-%d", root.Generation)}}
		},
		host,
		NewSyncEffectRunner(),
	)
	host.sink.onWrite = func() {
		select {
		case frameWritten <- struct{}{}:
		default:
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- runtime.Run(ctx)
	}()

	if err := runtime.Post(testMsg{Name: "wake"}); err != nil {
		t.Fatalf("post wake: %v", err)
	}
	select {
	case <-frameWritten:
	case <-time.After(time.Second):
		t.Fatal("runtime.Run should wake and render without outer polling sleep")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation after stopping run loop, got %v", err)
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"frame-1"}) {
		t.Fatalf("unexpected rendered frames %v", got)
	}
}

func TestAppRuntimePollsHostSizeWithoutResizeEvent(t *testing.T) {
	host := NewFakeTerminalHost(4)
	host.SetSize(80, 20)
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			return root, nil
		},
		func(root state.Root) render.Frame {
			return render.Frame{Lines: []string{viewportLabel(root)}}
		},
		host,
		nil,
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}
	host.SetSize(120, 40)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("resize drain: %v", err)
	}

	if got := runtime.State().Viewport; !got.Valid || got.Cols != 120 || got.Rows != 40 {
		t.Fatalf("expected polled host size to update viewport, got %#v", got)
	}
	if got := frameLines(host.Frames()); !reflect.DeepEqual(got, []string{"80x20", "120x40"}) {
		t.Fatalf("expected host size poll to render updated viewport, got %v", got)
	}
}

func TestFakeTerminalHostReportsFullInputQueue(t *testing.T) {
	host := NewFakeTerminalHost(1)
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey}); err != nil {
		t.Fatalf("send first input: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse}); !errors.Is(err, ErrInputQueueFull) {
		t.Fatalf("expected ErrInputQueueFull, got %v", err)
	}
}

func TestAppRuntimeDispatchesMouseHitRegionsToPaneCommands(t *testing.T) {
	focusHost := NewFakeTerminalHost(8)
	focusRoot := state.Root{
		Shell: state.DefaultShell().
			SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
			FocusPane(state.PaneCommandTarget{PaneID: "pane-main"}),
	}
	focusRuntime := newShellHitRuntime(focusRoot, focusHost)
	if err := focusRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post focus initial render: %v", err)
	}
	if err := focusRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("focus initial drain: %v", err)
	}

	content := frameHitRegion(t, lastRuntimeFrame(t, focusHost), render.HitRegionPaneContent, "pane-2")
	if err := focusHost.SendInput(mouseEventAt(content.Rect)); err != nil {
		t.Fatalf("send content click: %v", err)
	}
	if err := focusRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("focus drain: %v", err)
	}
	if focusRuntime.State().Shell.EnsureDefaults().ActivePaneID != "pane-2" {
		t.Fatalf("content click should focus pane-2, got %#v", focusRuntime.State().Shell)
	}

	closeHost := NewFakeTerminalHost(8)
	closeRoot := state.Root{
		Shell: state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical),
	}
	closeRuntime := newShellHitRuntime(closeRoot, closeHost)
	if err := closeRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post close initial render: %v", err)
	}
	if err := closeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("close initial drain: %v", err)
	}

	frame := lastRuntimeFrame(t, closeHost)
	action := frameHitRegionByAction(t, frame, render.HitRegionPaneAction, "pane.close", "pane-2")
	closeMouse := mouseEventAtRenderedTokenInRect(t, frame, action.Rect, render.DefaultPaneChromeGlyphs().Close)
	if !pointInRenderRect(closeMouse, action.Rect) {
		t.Fatalf("visible close token must be inside pane.close hit region, mouse=%#v region=%#v line=%q", closeMouse, action, frame.Lines[action.Rect.Y])
	}
	if err := closeHost.SendInput(closeMouse); err != nil {
		t.Fatalf("send visible close click: %v", err)
	}
	if err := closeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("close drain: %v", err)
	}
	if closeRuntime.State().Shell.HasPane(state.PaneCommandTarget{PaneID: "pane-2"}) {
		t.Fatalf("action click should close pane-2, got %#v", closeRuntime.State().Shell)
	}
}

func TestAppRuntimePaneFocusClickKeepsCopyMode(t *testing.T) {
	host := NewFakeTerminalHost(8)
	root := state.Root{
		Shell: state.DefaultShell().
			SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		History: state.HistoryStore{
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			Token:      "tok-1",
			Cols:       78,
			Rows:       []state.HistoryRow{{Text: "frozen history", LineID: 1}},
		},
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-1",
			BoundToken: "tok-1",
			BoundCols:  78,
			ViewRows:   10,
		},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-1", 7, 78, 10, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewShellReducer(), NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}})),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	content := frameHitRegion(t, lastRuntimeFrame(t, host), render.HitRegionPaneContent, "pane-2")
	if err := host.SendInput(mouseEventAt(content.Rect)); err != nil {
		t.Fatalf("click pane-2: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("focus drain: %v", err)
	}

	if got := runtime.State().Shell.EnsureDefaults().ActivePaneID; got != "pane-2" {
		t.Fatalf("content click should focus pane-2, got %q", got)
	}
	if !runtime.State().CopyMode.Active || runtime.State().CopyMode.PaneID != state.DefaultPaneID {
		t.Fatalf("pane focus click must keep original copy mode, got %#v", runtime.State().CopyMode)
	}
	if len(runtime.State().History.Rows) != 1 || runtime.State().History.Token != "tok-1" {
		t.Fatalf("pane focus click must keep frozen history, got %#v", runtime.State().History)
	}
	frame := lastRuntimeFrame(t, host)
	if !frameContains(frame, "frozen history") {
		t.Fatalf("copy/history panel must keep rendering frozen history after pane focus changes, got %#v", frame.Lines)
	}
}

func TestAppRuntimePaneCloseHitRegionMatchesWideGlyph(t *testing.T) {
	render.SetPaneChromeGlyphs(render.PaneChromeGlyphs{Close: "❌"})
	defer render.ResetPaneChromeGlyphs()

	host := NewFakeTerminalHost(8)
	root := state.Root{
		Shell: state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical),
	}
	runtime := newShellHitRuntime(root, host)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	frame := lastRuntimeFrame(t, host)
	action := frameHitRegionByAction(t, frame, render.HitRegionPaneAction, render.ActionPaneClose.String(), "pane-2")
	closeMouse := mouseEventAtRenderedTokenInRect(t, frame, action.Rect, "❌")
	if !pointInRenderRect(closeMouse, action.Rect) {
		t.Fatalf("wide close glyph must be inside pane.close hit region, mouse=%#v region=%#v line=%q", closeMouse, action, frame.Lines[action.Rect.Y])
	}
	if err := host.SendInput(closeMouse); err != nil {
		t.Fatalf("send wide close click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	if runtime.State().Shell.HasPane(state.PaneCommandTarget{PaneID: "pane-2"}) {
		t.Fatalf("wide glyph close click should close pane-2, got %#v", runtime.State().Shell)
	}
}

func TestAppRuntimeLastPaneCloseClickShowsFeedback(t *testing.T) {
	host := NewFakeTerminalHost(8)
	runtime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, host)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	frame := lastRuntimeFrame(t, host)
	action := frameHitRegionByAction(t, frame, render.HitRegionPaneAction, render.ActionPaneClose.String(), state.DefaultPaneID)
	closeMouse := mouseEventAtRenderedTokenInRect(t, frame, action.Rect, render.DefaultPaneChromeGlyphs().Close)
	if err := host.SendInput(closeMouse); err != nil {
		t.Fatalf("send last pane close click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	shell := runtime.State().Shell.EnsureDefaults()
	if !shell.HasPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}) {
		t.Fatalf("last pane close must not remove final pane, got %#v", shell)
	}
	if len(shell.Toasts) == 0 || shell.Toasts[len(shell.Toasts)-1].Body != "cannot close last pane" {
		t.Fatalf("last pane close click should show feedback, got %#v", shell.Toasts)
	}
}

func TestAppRuntimeTakeResizeOwnerRequiresDoubleClick(t *testing.T) {
	host := NewFakeTerminalHost(8)
	root := state.Root{Shell: state.DefaultShell().SetPanelPresentation(state.PanelPresentationCard)}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 78, 18, state.TerminalResizeRoleFollower, "surface", "view-1", false))
	runtime := newShellHitRuntime(root, host)
	now := time.Unix(1, 0)
	runtime.now = func() time.Time { return now }
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	frame := lastRuntimeFrame(t, host)
	action := frameHitRegionByAction(t, frame, render.HitRegionPaneAction, render.ActionTerminalTakeResizeOwner.String(), state.DefaultPaneID)
	click := mouseEventAtRenderedTokenInRect(t, frame, action.Rect, "◇ follow")
	if err := host.SendInput(click); err != nil {
		t.Fatalf("send first owner click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("first drain: %v", err)
	}
	if binding, _ := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); binding.ResizeRole != state.TerminalResizeRoleFollower || binding.CanResize {
		t.Fatalf("single click must not take resize owner, got %#v", binding)
	}
	if got := runtime.State().Shell.EnsureDefaults().OwnerConfirm.ViewID; got != "view-1" {
		t.Fatalf("single click should arm owner confirmation, got %q", got)
	}
	if frame := lastRuntimeFrame(t, host); !frameContains(frame, "◆ owner?") {
		t.Fatalf("single click should render owner confirmation, got %#v", frame.Lines)
	}
	seq := runtime.State().Shell.EnsureDefaults().OwnerConfirm.Seq
	if err := runtime.Post(ShellClearOwnerConfirmMsg{Seq: seq}); err != nil {
		t.Fatalf("post owner confirm clear: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("clear drain: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().OwnerConfirm.ViewID; got != "" {
		t.Fatalf("owner confirmation should clear after timeout, got %q", got)
	}
	if frame := lastRuntimeFrame(t, host); !frameContains(frame, "◇ follow") {
		t.Fatalf("cleared owner confirmation should render follow, got %#v", frame.Lines)
	}
	now = now.Add(ownerConfirmDelay + time.Millisecond)
	if err := host.SendInput(click); err != nil {
		t.Fatalf("send rearmed owner click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("rearm drain: %v", err)
	}
	if binding, _ := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); binding.ResizeRole != state.TerminalResizeRoleFollower || binding.CanResize {
		t.Fatalf("click after confirmation timeout must only rearm, got %#v", binding)
	}
	if err := host.SendInput(click); err != nil {
		t.Fatalf("send second owner click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("second drain: %v", err)
	}
	if binding, _ := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); binding.ResizeRole != state.TerminalResizeRoleFollower || binding.CanResize || !binding.OwnerAcquirePending {
		t.Fatalf("double click should wait for daemon owner confirmation, got %#v", binding)
	}
}

func TestInteractiveRuntimeTerminalSizeLockChromeButtonUsesDaemonControl(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:     "term-1",
			Channel:        7,
			Cols:           80,
			Rows:           24,
			ResizePolicy:   state.TerminalResizeRoleOwner,
			CanResize:      true,
			SurfaceID:      "surface",
			ViewID:         state.TerminalPaneViewID(state.DefaultPaneID),
			OwnerSurfaceID: "surface",
			OwnerViewID:    state.TerminalPaneViewID(state.DefaultPaneID),
		},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "main", Tags: map[string]string{"role": "shell"}}}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(96, 28)
	root := state.Root{
		Shell:        state.DefaultShell().SetPanelPresentation(state.PanelPresentationCard),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "main", Tags: map[string]string{"role": "shell"}}}},
	}
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}})
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "test-surface", ViewID: state.TerminalPaneViewID(state.DefaultPaneID)}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("attach drain: %v", err)
	}

	frame := lastRuntimeFrame(t, host)
	action := frameHitRegionByAction(t, frame, render.HitRegionPaneAction, render.ActionResizeLayoutLock.String(), state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send size lock click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("click drain: %v", err)
	}
	if len(terminal.ResizeLocks) != 1 || terminal.ResizeLocks[0].TerminalID != "term-1" || !terminal.ResizeLocks[0].Locked {
		t.Fatalf("chrome size lock click should call daemon control, locks=%#v", terminal.ResizeLocks)
	}
	binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || !binding.SizeLocked || binding.CanResize || binding.ResizeRole != state.TerminalResizeRoleOwner {
		t.Fatalf("chrome size lock click should project locked owner view, binding=%#v ok=%v", binding, ok)
	}
	if !frameContains(lastRuntimeFrame(t, host), render.DefaultPaneChromeGlyphs().SizeLock) {
		t.Fatalf("locked frame should render size lock glyph, got %#v", lastRuntimeFrame(t, host).Lines)
	}

	frame = lastRuntimeFrame(t, host)
	action = frameHitRegionByAction(t, frame, render.HitRegionPaneAction, render.ActionResizeLayoutLock.String(), state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send size unlock click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("unlock drain: %v", err)
	}
	if len(terminal.ResizeLocks) != 2 || terminal.ResizeLocks[1].TerminalID != "term-1" || terminal.ResizeLocks[1].Locked {
		t.Fatalf("chrome locked owner should stay clickable for daemon unlock, locks=%#v", terminal.ResizeLocks)
	}
	binding, ok = runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.SizeLocked || !binding.CanResize || binding.ResizeRole != state.TerminalResizeRoleOwner {
		t.Fatalf("chrome unlock should restore owner resize rights, binding=%#v ok=%v", binding, ok)
	}
	if len(terminal.Resizes) != 1 || terminal.Resizes[0].ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("unlock should resize owner panel when size diverged, resizes=%#v", terminal.Resizes)
	}
}

func TestInteractiveRuntimeTerminalSizeLockChromeButtonDoesNotLoadMissingTags(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:     "term-1",
			Channel:        7,
			Cols:           80,
			Rows:           24,
			ResizePolicy:   state.TerminalResizeRoleOwner,
			CanResize:      true,
			SurfaceID:      "surface",
			ViewID:         state.TerminalPaneViewID(state.DefaultPaneID),
			OwnerSurfaceID: "surface",
			OwnerViewID:    state.TerminalPaneViewID(state.DefaultPaneID),
		},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "main", Tags: map[string]string{"role": "shell"}}}},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(96, 28)
	root := state.Root{Shell: state.DefaultShell().SetPanelPresentation(state.PanelPresentationCard)}
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}})
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "test-surface", ViewID: state.TerminalPaneViewID(state.DefaultPaneID)}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("attach drain: %v", err)
	}

	frame := lastRuntimeFrame(t, host)
	action := frameHitRegionByAction(t, frame, render.HitRegionPaneAction, render.ActionResizeLayoutLock.String(), state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send size lock click without cached tags: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("click drain: %v", err)
	}
	if len(terminal.ResizeLocks) != 1 || !terminal.ResizeLocks[0].Locked || len(terminal.TagEdits) != 0 {
		t.Fatalf("chrome lock should call daemon without metadata edits, locks=%#v edits=%#v", terminal.ResizeLocks, terminal.TagEdits)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || !binding.SizeLocked || binding.CanResize {
		t.Fatalf("chrome lock with missing cache should project locked owner, binding=%#v ok=%v", binding, ok)
	}
	if !frameContains(lastRuntimeFrame(t, host), render.DefaultPaneChromeGlyphs().SizeLock) {
		t.Fatalf("locked frame should render size lock glyph after loading tags, got %#v", lastRuntimeFrame(t, host).Lines)
	}
}

func TestInteractiveRuntimeFloatingSizeLockChromeButtonTargetsFloatingTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 32)
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationCard).BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-pane")
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Title:    "float",
		Rect:     state.FloatingRect{X: 8, Y: 4, W: 62, H: 12},
		BoundsW:  110,
		BoundsH:  32,
	})
	shell, _ = shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandDeactivate})
	root := state.Root{
		Shell: shell,
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{
			{TerminalID: "term-pane", Title: "pane", Tags: map[string]string{"role": "pane"}},
			{TerminalID: "term-float", Title: "float", Tags: map[string]string{"role": "float"}},
		}},
	}
	paneBinding := state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 3, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)
	paneBinding.OwnerSurfaceID = "surface"
	paneBinding.OwnerViewID = state.TerminalPaneViewID(state.DefaultPaneID)
	floatingBinding := state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-float", 4, 40, 10, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true)
	floatingBinding.OwnerSurfaceID = "surface"
	floatingBinding.OwnerViewID = state.TerminalFloatingViewID("floating-1")
	root.TerminalViews = root.TerminalViews.BindPane(paneBinding)
	root.TerminalViews = root.TerminalViews.BindFloating(floatingBinding)
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}})
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	frame := lastRuntimeFrame(t, host)
	action := frameActionHitRegion(t, frame, render.ActionResizeLayoutLock.String(), "floating-pane-1")
	if !action.Floating {
		t.Fatalf("floating size lock hit region should carry floating flag, got %#v", action)
	}
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send floating size lock click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("floating click drain: %v", err)
	}
	if len(terminal.ResizeLocks) != 1 || terminal.ResizeLocks[0].TerminalID != "term-float" || !terminal.ResizeLocks[0].Locked {
		t.Fatalf("floating chrome size lock should target floating terminal, locks=%#v", terminal.ResizeLocks)
	}
	if runtime.State().Shell.ActiveFloatingID() != "floating-1" {
		t.Fatalf("floating size lock click should focus floating, shell=%#v", runtime.State().Shell)
	}
	if floating, ok := runtime.State().TerminalViews.FloatingBinding("floating-1"); !ok || !floating.SizeLocked || floating.CanResize {
		t.Fatalf("floating binding should project size lock, binding=%#v ok=%v", floating, ok)
	}
	if pane, _ := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); pane.SizeLocked {
		t.Fatalf("pane terminal must not be locked by floating chrome click, pane=%#v", pane)
	}
}

func TestAppRuntimeDispatchesHeaderTabActionHitRegions(t *testing.T) {
	workspaceHost := NewFakeTerminalHost(8)
	workspaceRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, workspaceHost)
	if err := workspaceRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post workspace initial render: %v", err)
	}
	if err := workspaceRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("workspace initial drain: %v", err)
	}
	workspaceAction := frameActionHitRegion(t, lastRuntimeFrame(t, workspaceHost), "menu.workbench_tree", "")
	if err := workspaceHost.SendInput(mouseEventAt(workspaceAction.Rect)); err != nil {
		t.Fatalf("send workspace navigator click: %v", err)
	}
	if err := workspaceRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("workspace navigator drain: %v", err)
	}
	if overlay := workspaceRuntime.State().Shell.Overlay; !overlay.Open || overlay.Kind != state.OverlayWorkbenchTree {
		t.Fatalf("workspace name click should open workbench navigator, overlay=%#v shell=%#v", overlay, workspaceRuntime.State().Shell)
	}

	closeHost := NewFakeTerminalHost(8)
	closeShell, _ := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	closeRoot := state.Root{
		Shell: closeShell,
	}
	closeRuntime := newShellHitRuntime(closeRoot, closeHost)
	if err := closeRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post close initial render: %v", err)
	}
	if err := closeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("close initial drain: %v", err)
	}
	closeAction := frameActionHitRegion(t, lastRuntimeFrame(t, closeHost), render.ActionTabClose.String(), "tab-2")
	if err := closeHost.SendInput(mouseEventAt(closeAction.Rect)); err != nil {
		t.Fatalf("send tab close click: %v", err)
	}
	if err := closeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("close drain: %v", err)
	}
	if tabs := closeRuntime.State().Shell.EnsureDefaults().Workspace.Tabs; len(tabs) != 1 || tabs[0].Title != "main" {
		t.Fatalf("tab close click should remove active tab and keep main, got %#v", tabs)
	}

	targetHost := NewFakeTerminalHost(8)
	targetShell, _ := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	targetRoot := state.Root{Shell: targetShell}
	targetRuntime := newShellHitRuntime(targetRoot, targetHost)
	if err := targetRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post target close initial render: %v", err)
	}
	if err := targetRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("target close initial drain: %v", err)
	}
	mainCloseAction := frameActionHitRegion(t, lastRuntimeFrame(t, targetHost), render.ActionTabClose.String(), state.DefaultTabID)
	if err := targetHost.SendInput(mouseEventAt(mainCloseAction.Rect)); err != nil {
		t.Fatalf("send inactive tab close click: %v", err)
	}
	if err := targetRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("target close drain: %v", err)
	}
	if tabs := targetRuntime.State().Shell.EnsureDefaults().Workspace.Tabs; len(tabs) != 1 || tabs[0].Title != "logs" || targetRuntime.State().Shell.EnsureDefaults().Workspace.ActiveTabID != tabs[0].ID {
		t.Fatalf("inactive tab close click should close the clicked tab and keep active logs tab, got %#v shell=%#v", tabs, targetRuntime.State().Shell.EnsureDefaults())
	}

	switchHost := NewFakeTerminalHost(8)
	switchShell, _ := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	switchShell, _ = switchShell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabSwitch, TargetID: state.DefaultTabID})
	switchRuntime := newShellHitRuntime(state.Root{Shell: switchShell}, switchHost)
	if err := switchRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post switch initial render: %v", err)
	}
	if err := switchRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("switch initial drain: %v", err)
	}
	switchAction := frameActionHitRegion(t, lastRuntimeFrame(t, switchHost), render.ActionTabSwitch.String(), "tab-2")
	if err := switchHost.SendInput(mouseEventAt(switchAction.Rect)); err != nil {
		t.Fatalf("send tab switch click: %v", err)
	}
	if err := switchRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("switch drain: %v", err)
	}
	if got := switchRuntime.State().Shell.EnsureDefaults().Workspace.ActiveTabID; got != "tab-2" {
		t.Fatalf("tab label click should switch active tab, got %q", got)
	}

	createHost := NewFakeTerminalHost(8)
	createRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, createHost)
	if err := createRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post create initial render: %v", err)
	}
	if err := createRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("create initial drain: %v", err)
	}
	createAction := frameActionHitRegion(t, lastRuntimeFrame(t, createHost), render.ActionTabCreate.String(), "")
	if err := createHost.SendInput(mouseEventAt(createAction.Rect)); err != nil {
		t.Fatalf("send tab create click: %v", err)
	}
	if err := createRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("create drain: %v", err)
	}
	createShell := createRuntime.State().Shell.EnsureDefaults()
	if tabs := createShell.Workspace.Tabs; len(tabs) != 2 || createShell.Workspace.ActiveTabID == state.DefaultTabID || createShell.ActivePaneID == "" {
		t.Fatalf("tab create click should add and activate a tab, got %#v", createRuntime.State().Shell)
	}
	if overlay := createShell.Overlay; !overlay.Open || overlay.Kind != state.OverlayTerminalPicker || overlay.TargetID != createShell.ActivePaneID {
		t.Fatalf("tab create click should open picker for new pane, overlay=%#v shell=%#v", overlay, createShell)
	}

	lastHost := NewFakeTerminalHost(8)
	lastRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, lastHost)
	if err := lastRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post last close initial render: %v", err)
	}
	if err := lastRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("last close initial drain: %v", err)
	}
	lastCloseAction := frameActionHitRegion(t, lastRuntimeFrame(t, lastHost), render.ActionTabClose.String(), state.DefaultTabID)
	if err := lastHost.SendInput(mouseEventAt(lastCloseAction.Rect)); err != nil {
		t.Fatalf("send last tab close click: %v", err)
	}
	if err := lastRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("last close drain: %v", err)
	}
	lastShell := lastRuntime.State().Shell.EnsureDefaults()
	if len(lastShell.Workspace.Tabs) != 0 || lastShell.Workspace.ActiveTabID != "" || lastShell.ActivePaneID != "" {
		t.Fatalf("last tab close click should leave an empty workspace, got %#v", lastShell)
	}
}

func TestInteractiveRuntimeTabRenameFooterAction(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(140, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	shell, result := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create tab: %#v", result)
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   shell.SetInteractionMode(state.InteractionModeTab),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	action := frameActionHitRegion(t, lastRuntimeFrame(t, host), "tab.rename", "")
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send tab rename footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tab rename footer click: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	if !shell.Overlay.Open || shell.Overlay.Prompt.Purpose != "tab.rename" || shell.Overlay.Prompt.Value != "logs" {
		t.Fatalf("tab rename footer action should open active tab rename prompt, got %#v", shell.Overlay)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("tab rename footer action must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeTabSwitchFooterActions(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	shell, result := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create tab: %#v", result)
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   shell.SetInteractionMode(state.InteractionModeTab),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}

	prevAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "tab.previous", "")
	if err := host.SendInput(mouseEventAt(prevAction.Rect)); err != nil {
		t.Fatalf("send tab previous footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tab previous footer click: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().Workspace.ActiveTabID; got != state.DefaultTabID {
		t.Fatalf("tab previous footer action should activate default tab, got %q", got)
	}

	nextAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "tab.next", "")
	if err := host.SendInput(mouseEventAt(nextAction.Rect)); err != nil {
		t.Fatalf("send tab next footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tab next footer click: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().Workspace.ActiveTabID; got == state.DefaultTabID {
		t.Fatalf("tab next footer action should return to logs tab, got %q", got)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("tab switch footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimePaneModeFooterActions(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModePane),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}

	splitAction := frameHitRegionByAction(t, lastRuntimeFrame(t, host), render.HitRegionPaneAction, render.ActionPaneSplitRight.String(), state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(splitAction.Rect)); err != nil {
		t.Fatalf("send pane split chrome click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane split chrome click: %v", err)
	}
	tab := runtime.State().Shell.EnsureDefaults().Workspace.Tabs[0]
	if len(tab.Panes) != 2 || tab.RootSplit.Direction != state.SplitDirectionVertical || runtime.State().Shell.EnsureDefaults().ActivePaneID == state.DefaultPaneID {
		t.Fatalf("pane split chrome action should create and activate vertical split, shell=%#v", runtime.State().Shell.EnsureDefaults())
	}

	assertFrameMissingActionHitRegion(t, lastRuntimeFrame(t, host), "panel.focus_next")
	activePaneID := runtime.State().Shell.EnsureDefaults().ActivePaneID

	zoomAction := frameHitRegionByAction(t, lastRuntimeFrame(t, host), render.HitRegionPaneAction, render.ActionPaneZoom.String(), activePaneID)
	if err := host.SendInput(mouseEventAt(zoomAction.Rect)); err != nil {
		t.Fatalf("send pane zoom footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane zoom footer click: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().ZoomedPaneID; got != activePaneID {
		t.Fatalf("pane zoom footer action should toggle zoom on active pane, got %q", got)
	}

	closeAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "panel.close", "")
	if err := host.SendInput(mouseEventAt(closeAction.Rect)); err != nil {
		t.Fatalf("send pane close footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane close footer click: %v", err)
	}
	shell := runtime.State().Shell.EnsureDefaults()
	if shell.HasPane(state.PaneCommandTarget{PaneID: activePaneID}) || len(shell.Workspace.Tabs[0].Panes) != 1 {
		t.Fatalf("pane close footer action should close active pane through workbench command, shell=%#v", shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("pane mode footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimePaneModeFooterHidesUnavailableLastPaneClose(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModePane),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	assertFrameMissingActionHitRegion(t, lastRuntimeFrame(t, host), "panel.close")
	if len(terminal.Inputs) != 0 {
		t.Fatalf("unavailable last pane footer close must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeResizeModeFooterActions(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}).
		SetInteractionMode(state.InteractionModeResize)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   shell,
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
			Config: state.TUIConfigStore{Shortcuts: state.TUIShortcutConfig{Configured: true, Scenes: map[string]state.TUIShortcutSceneConfig{
				"resize": {Bindings: map[string]state.TUIShortcutBindingConfig{
					"=": {Action: "panel.balance"},
				}},
			}}},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}

	assertFrameMissingActionHitRegion(t, lastRuntimeFrame(t, host), "resize.right")
	split := runtime.State().Shell.EnsureDefaults().Workspace.Tabs[0].RootSplit
	if split.BiasCells != 0 {
		t.Fatalf("hint-only resize aggregate must not mutate split bias, got %#v", split)
	}

	balanceAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "panel.balance", "")
	if err := host.SendInput(mouseEventAt(balanceAction.Rect)); err != nil {
		t.Fatalf("send resize balance footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize balance footer click: %v", err)
	}
	split = runtime.State().Shell.EnsureDefaults().Workspace.Tabs[0].RootSplit
	if split.BiasCells != 0 {
		t.Fatalf("resize balance footer action should clear split bias, got %#v", split)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("resize mode footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeResizeModeFooterHidesUnavailableSinglePaneResize(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModeResize),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	frame := lastRuntimeFrame(t, host)
	assertFrameMissingActionHitRegion(t, frame, "resize.left")
	assertFrameMissingActionHitRegion(t, frame, "resize.right")
	assertFrameMissingActionHitRegion(t, frame, "resize.up")
	assertFrameMissingActionHitRegion(t, frame, "resize.down")
	if len(terminal.Inputs) != 0 {
		t.Fatalf("unavailable single pane resize footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeFloatingFooterActions(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModeFloating),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}

	newAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "floating.new", "")
	if err := host.SendInput(mouseEventAt(newAction.Rect)); err != nil {
		t.Fatalf("send floating new footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating new footer click: %v", err)
	}
	shell := runtime.State().Shell.EnsureDefaults()
	floatings := shell.ActiveFloatings()
	if len(floatings) != 1 || shell.ActiveFloatingID() == "" || !floatings[0].Active {
		t.Fatalf("floating new footer action should create active floating, shell=%#v", shell)
	}

	closeAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), render.ActionFloatingClose.String(), "")
	if err := host.SendInput(mouseEventAt(closeAction.Rect)); err != nil {
		t.Fatalf("send active floating close footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain active floating close footer click: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	if len(shell.ActiveFloatings()) != 0 || shell.ActiveFloatingID() != "" {
		t.Fatalf("floating close footer action should close active floating, shell=%#v", shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("floating footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeFloatingFooterHidesCloseWithoutActive(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModeFloating),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	assertFrameMissingActionHitRegion(t, lastRuntimeFrame(t, host), render.ActionFloatingClose.String())
	if len(terminal.Inputs) != 0 {
		t.Fatalf("unavailable floating close footer action must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeFloatingOverviewKeyboardAndContentActions(t *testing.T) {
	host := NewFakeTerminalHost(24)
	host.SetSize(120, 28)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeFloating),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	for _, command := range []state.FloatingCommand{
		{
			Action:   state.FloatingCommandCreate,
			TargetID: "floating-1",
			Title:    "logs",
			Pane:     state.PaneState{ID: "floating-pane-1", Title: "logs", Kind: state.PaneEmpty},
			Rect:     state.FloatingRect{X: 6, Y: 3, W: 28, H: 9},
			Source:   state.PaneCommandSourceTest,
		},
		{
			Action:   state.FloatingCommandCreate,
			TargetID: "floating-2",
			Title:    "shell",
			Pane:     state.PaneState{ID: "floating-pane-2", Title: "shell", Kind: state.PaneEmpty},
			Rect:     state.FloatingRect{X: 18, Y: 6, W: 30, H: 10},
			Source:   state.PaneCommandSourceTest,
		},
	} {
		if err := runtime.Post(ShellFloatingCommandMsg{Command: command}); err != nil {
			t.Fatalf("post floating setup %#v: %v", command, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating setup: %v", err)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "o"},
		{Kind: input.EventKindKey, Key: input.KeyDown},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send floating overview input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain floating overview input %#v: %v", event, err)
		}
	}

	shell := runtime.State().Shell.EnsureDefaults()
	if shell.Overlay.Kind != state.OverlayFloatingOverview || !shell.Overlay.Open {
		t.Fatalf("floating overview should stay open during overview actions, shell=%#v", shell)
	}
	if shell.Overlay.SelectedIndex != 1 {
		t.Fatalf("floating overview should move selection to second floating, overlay=%#v", shell.Overlay)
	}
	floatings := shell.ActiveFloatings()
	if len(floatings) != 2 || floatings[0].Collapsed || floatings[1].Collapsed {
		t.Fatalf("collapse-all then show-all should end with both floatings expanded, floatings=%#v", floatings)
	}
	if shell.ActiveFloatingID() != "floating-2" || !floatings[1].Active {
		t.Fatalf("enter open should raise selected floating, shell=%#v", shell)
	}
	frame := lastRuntimeFrame(t, host)
	if frameContains(frame, "● open") || frameContains(frame, "Restore, collapse") {
		t.Fatalf("floating overview should not render old title/status/help copy, got %#v", frame.Lines)
	}
	if !frameContains(frame, "Floating Windows") || !frameContains(frame, "terminal") || !frameContains(frame, "No terminal connected") || !frameContains(frame, "empty") {
		t.Fatalf("expected floating overview frame, got %#v", frame.Lines)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"}); err != nil {
		t.Fatalf("send floating collapse-all hotkey: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating collapse-all hotkey: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	floatings = shell.ActiveFloatings()
	if !floatings[0].Collapsed || !floatings[1].Collapsed {
		t.Fatalf("collapse-all hotkey should collapse every floating, floatings=%#v", floatings)
	}

	rowAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), render.ActionFloatingSummon.String(), "floating-1")
	if err := host.SendInput(mouseEventAt(rowAction.Rect)); err != nil {
		t.Fatalf("send floating row summon click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating row summon click: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	floatings = shell.ActiveFloatings()
	if shell.ActiveFloatingID() != "floating-1" || floatings[0].Collapsed {
		t.Fatalf("row summon should activate and expand selected floating, shell=%#v", shell)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "2"}); err != nil {
		t.Fatalf("send floating summon hotkey: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating summon hotkey: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	floatings = shell.ActiveFloatings()
	if shell.ActiveFloatingID() != "floating-2" || floatings[1].Collapsed {
		t.Fatalf("number summon should activate indexed floating, shell=%#v", shell)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send floating close hotkey: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating close hotkey: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	floatings = shell.ActiveFloatings()
	if len(floatings) != 1 || floatings[0].ID != "floating-1" {
		t.Fatalf("overview close should remove selected floating, shell=%#v", shell)
	}
	if !shell.Overlay.Open || shell.Overlay.Kind != state.OverlayFloatingOverview {
		t.Fatalf("overview close should keep overview overlay active for remaining floatings, overlay=%#v", shell.Overlay)
	}
}

func TestInteractiveRuntimeSingleTabSwitchFooterActionsAreHidden(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModeTab),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	frame := lastRuntimeFrame(t, host)
	assertFrameMissingActionHitRegion(t, frame, "tab.next")
	assertFrameMissingActionHitRegion(t, frame, "tab.previous")
	if len(terminal.Inputs) != 0 {
		t.Fatalf("unavailable single tab switch footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestAppRuntimeDispatchesFooterActionHitRegions(t *testing.T) {
	paneHost := NewFakeTerminalHost(8)
	paneRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, paneHost)
	if err := paneRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post pane footer render: %v", err)
	}
	if err := paneRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane footer render: %v", err)
	}
	paneAction := frameActionHitRegion(t, lastRuntimeFrame(t, paneHost), "menu.panel", "")
	if err := paneHost.SendInput(mouseEventAt(paneAction.Rect)); err != nil {
		t.Fatalf("send footer pane click: %v", err)
	}
	if err := paneRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer pane click: %v", err)
	}
	if got := paneRuntime.State().Shell.EnsureDefaults().InteractionMode; got != state.InteractionModePane {
		t.Fatalf("footer pane click should enter pane mode, got %q", got)
	}

	globalModeHost := NewFakeTerminalHost(8)
	globalModeRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, globalModeHost)
	if err := globalModeRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post global footer render: %v", err)
	}
	if err := globalModeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global footer render: %v", err)
	}
	globalModeAction := frameActionHitRegion(t, lastRuntimeFrame(t, globalModeHost), "menu.system", "")
	if err := globalModeHost.SendInput(mouseEventAt(globalModeAction.Rect)); err != nil {
		t.Fatalf("send footer global click: %v", err)
	}
	if err := globalModeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer global click: %v", err)
	}
	if got := globalModeRuntime.State().Shell.EnsureDefaults().InteractionMode; got != state.InteractionModeGlobal {
		t.Fatalf("footer global click should enter global mode, got %q", got)
	}

	globalHost := NewFakeTerminalHost(8)
	globalTerminal := &testkit.FakeTerminalService{
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running"}}},
	}
	globalRuntime := NewInteractiveRuntime(
		state.Root{Shell: state.DefaultShell().SetInteractionMode(state.InteractionModeGlobal).AddToast(state.ToastSpec{ID: "toast-1", Title: "notice"})},
		globalHost,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: globalTerminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	globalHost.SetSize(120, 20)
	if err := globalRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post global footer render: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global footer render: %v", err)
	}
	globalFrame := lastRuntimeFrame(t, globalHost)
	assertFrameMissingActionHitRegion(t, globalFrame, "system.close_toast")
	assertFrameMissingActionHitRegion(t, globalFrame, "system.clear_toasts")

	headerAction := frameActionHitRegion(t, globalFrame, "system.toggle_header", "")
	if err := globalHost.SendInput(mouseEventAt(headerAction.Rect)); err != nil {
		t.Fatalf("send footer header click: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer header click: %v", err)
	}
	if globalRuntime.State().Shell.EnsureDefaults().HeaderVisible {
		t.Fatalf("footer header click should hide header, got %#v", globalRuntime.State().Shell.EnsureDefaults())
	}

	if err := globalRuntime.Post(ShellSetHeaderVisibleMsg{Visible: true}); err != nil {
		t.Fatalf("restore header: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain restore header: %v", err)
	}
	globalModeAction = frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.system", "")
	if err := globalHost.SendInput(mouseEventAt(globalModeAction.Rect)); err != nil {
		t.Fatalf("reenter global mode before footer toggle: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global mode before footer toggle: %v", err)
	}
	footerAction := frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "system.toggle_footer", "")
	if err := globalHost.SendInput(mouseEventAt(footerAction.Rect)); err != nil {
		t.Fatalf("send footer toggle click: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer toggle click: %v", err)
	}
	if globalRuntime.State().Shell.EnsureDefaults().FooterVisible {
		t.Fatalf("footer toggle click should hide footer, got %#v", globalRuntime.State().Shell.EnsureDefaults())
	}
	if err := globalRuntime.Post(ShellSetFooterVisibleMsg{Visible: true}); err != nil {
		t.Fatalf("restore footer: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain restore footer: %v", err)
	}
	globalModeAction = frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.system", "")
	if err := globalHost.SendInput(mouseEventAt(globalModeAction.Rect)); err != nil {
		t.Fatalf("reenter global mode before help: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global mode before help: %v", err)
	}

	helpAction := frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.help", "")
	if err := globalHost.SendInput(mouseEventAt(helpAction.Rect)); err != nil {
		t.Fatalf("send footer help click: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer help click: %v", err)
	}
	if globalRuntime.State().Shell.EnsureDefaults().Overlay.Kind != state.OverlayHelp {
		t.Fatalf("footer help click should open help, got %#v", globalRuntime.State().Shell.EnsureDefaults().Overlay)
	}
	if err := globalHost.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send help esc: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain help esc: %v", err)
	}
	if globalRuntime.State().Shell.EnsureDefaults().Overlay.Open {
		t.Fatalf("help esc should close overlay, got %#v", globalRuntime.State().Shell.EnsureDefaults().Overlay)
	}

	globalModeAction = frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.system", "")
	if err := globalHost.SendInput(mouseEventAt(globalModeAction.Rect)); err != nil {
		t.Fatalf("reenter global mode before pool: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global mode before pool: %v", err)
	}
	poolAction := frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.terminal_pool", "")
	if err := globalHost.SendInput(mouseEventAt(poolAction.Rect)); err != nil {
		t.Fatalf("send footer pool click: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer pool click: %v", err)
	}
	if globalRuntime.State().Shell.EnsureDefaults().Overlay.Kind != state.OverlayTerminalPool {
		t.Fatalf("footer pool click should open terminal pool, got %#v", globalRuntime.State().Shell.EnsureDefaults().Overlay)
	}
	if err := globalHost.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send pool esc: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool esc: %v", err)
	}

	globalModeAction = frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.system", "")
	if err := globalHost.SendInput(mouseEventAt(globalModeAction.Rect)); err != nil {
		t.Fatalf("reenter global mode before tree: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global mode before tree: %v", err)
	}
	treeAction := frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.workbench_tree", "")
	if err := globalHost.SendInput(mouseEventAt(treeAction.Rect)); err != nil {
		t.Fatalf("send footer tree click: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer tree click: %v", err)
	}
	if globalRuntime.State().Shell.EnsureDefaults().Overlay.Kind != state.OverlayWorkbenchTree {
		t.Fatalf("footer tree click should open workbench tree, got %#v", globalRuntime.State().Shell.EnsureDefaults().Overlay)
	}
	if err := globalHost.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEsc}); err != nil {
		t.Fatalf("send tree esc: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tree esc: %v", err)
	}
	globalModeAction = frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "menu.system", "")
	if err := globalHost.SendInput(mouseEventAt(globalModeAction.Rect)); err != nil {
		t.Fatalf("reenter global mode before quit: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain global mode before quit: %v", err)
	}

	globalHost.SetSize(240, 20)
	if err := globalRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post wide global footer render: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain wide global footer render: %v", err)
	}
	quitAction := frameActionHitRegion(t, lastRuntimeFrame(t, globalHost), "system.quit", "")
	if err := globalHost.SendInput(mouseEventAt(quitAction.Rect)); err != nil {
		t.Fatalf("send footer quit click: %v", err)
	}
	if err := globalRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer quit click: %v", err)
	}
	if !globalRuntime.Quit() {
		t.Fatal("footer quit click should stop runtime")
	}
}

func TestInteractiveRuntimeFooterActionDoesNotLeakTerminalInput(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 100, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	action := frameActionHitRegion(t, lastRuntimeFrame(t, host), "menu.panel", "")
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send footer pane click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain footer pane click: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().InteractionMode; got != state.InteractionModePane {
		t.Fatalf("footer pane click should enter pane mode, got %q", got)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("footer action click must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeWorkspaceDeleteFooterAction(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	shell, result := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "remote"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create workspace: %#v", result)
	}
	remoteWorkspaceID := result.ID
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceSwitch, TargetID: remoteWorkspaceID})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("switch workspace: %#v", result)
	}
	shell = shell.SetInteractionMode(state.InteractionModeWorkspace)
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   shell,
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	activeBefore := runtime.State().Shell.EnsureDefaults().Workspace.ID
	if activeBefore != remoteWorkspaceID {
		t.Fatalf("workspace delete harness must start from remote workspace, active=%q remote=%q shell=%#v", activeBefore, remoteWorkspaceID, runtime.State().Shell)
	}
	action := frameActionHitRegion(t, lastRuntimeFrame(t, host), "workspace.delete", "")
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send workspace delete footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace delete footer click: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	if shell.Workspace.ID == activeBefore || len(shell.Workspaces) != 1 || workspaceIDExistsForTest(shell.Workspaces, activeBefore) {
		t.Fatalf("workspace delete footer action should delete active workspace, got %#v", shell)
	}
	if !toastExistsForTest(shell.Toasts, string(state.WorkbenchCommandWorkspaceDelete), activeBefore) {
		t.Fatalf("workspace delete footer action should show success feedback for deleted workspace %q, got %#v", activeBefore, shell.Toasts)
	}
	assertFrameMissingActionHitRegion(t, lastRuntimeFrame(t, host), "workspace.delete")
	shell = runtime.State().Shell.EnsureDefaults()
	if len(shell.Workspaces) != 1 {
		t.Fatalf("last workspace delete footer action should be hidden after delete, got %#v", shell)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("workspace footer delete must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeTerminalPoolDeleteRemovesBindingsAndReloadsInventory(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-logs", Channel: 7, Cols: 80, Rows: 24},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{
			{TerminalID: "term-logs", Title: "logs", State: "running"},
			{TerminalID: "term-shell", Title: "shell", State: "running"},
		}},
	}
	host := NewFakeTerminalHost(48)
	host.SetSize(100, 28)
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-logs"),
		Session: state.TerminalSessionStore{
			TerminalID: "term-logs",
			Channel:    7,
			Attached:   true,
			Cols:       80,
			Rows:       24,
		},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-logs",
			Ready:      true,
			Lines:      []string{"live logs"},
			Surfaces: map[string]state.LiveSurfaceSnapshot{
				"term-logs": {TerminalID: "term-logs", Lines: []string{"live logs"}},
			},
		},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(
		state.DefaultPaneID, "term-logs", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
	))
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x07", Ctrl: true},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "t"},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send pool open input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain pool open input %#v: %v", event, err)
		}
	}

	frame := lastRuntimeFrame(t, host)
	if !frameContains(frame, "Terminal Manager") || !frameContains(frame, "logs") {
		t.Fatalf("expected terminal pool frame, got %#v", frame.Lines)
	}
	selectRegion := frameActionHitRegion(t, frame, render.ActionPoolSelect.String(), "")
	if err := host.SendInput(mouseEventAt(selectRegion.Rect)); err != nil {
		t.Fatalf("send pool select click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool select click: %v", err)
	}
	if runtime.State().Shell.Overlay.SelectedIndex != 0 {
		t.Fatalf("pool row click should select first row, overlay=%#v", runtime.State().Shell.Overlay)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "\x18", Ctrl: true}); err != nil {
		t.Fatalf("send pool delete shortcut: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pool delete shortcut: %v", err)
	}

	if len(terminal.Removes) != 1 || terminal.Removes[0].TerminalID != "term-logs" {
		t.Fatalf("pool delete should call terminal remove service, removes=%#v", terminal.Removes)
	}
	if len(terminal.Lists) < 2 {
		t.Fatalf("pool delete should trigger inventory reload, lists=%#v", terminal.Lists)
	}
	stateAfter := runtime.State()
	if stateAfter.TerminalPool.LastRemovedID != "term-logs" {
		t.Fatalf("pool delete should record removed terminal, pool=%#v", stateAfter.TerminalPool)
	}
	if _, ok := stateAfter.TerminalViews.PaneBinding(state.DefaultPaneID); ok {
		t.Fatal("pool delete should clear pane terminal view binding")
	}
	if pane, ok := stateAfter.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.Kind != state.PaneEmpty || pane.TerminalID != "" {
		t.Fatalf("pool delete should clear shell pane binding, pane=%#v ok=%v", pane, ok)
	}
	if stateAfter.Session.TerminalID != "" || stateAfter.Surface.TerminalID != "" {
		t.Fatalf("pool delete should clear active session/surface, session=%#v surface=%#v", stateAfter.Session, stateAfter.Surface)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("pool delete path must not leak terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeWorkspaceNewRenameFooterActions(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModeWorkspace),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}

	newAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "workspace.create", "")
	if err := host.SendInput(mouseEventAt(newAction.Rect)); err != nil {
		t.Fatalf("send workspace new footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace new footer click: %v", err)
	}
	shell := runtime.State().Shell.EnsureDefaults()
	if len(shell.Workspaces) != 2 || shell.Workspace.ID == state.DefaultWorkspaceID || shell.Workspace.Name != "workspace 2" {
		t.Fatalf("workspace new footer action should create and activate next workspace, got %#v", shell)
	}
	if !toastExistsForTest(shell.Toasts, string(state.WorkbenchCommandWorkspaceCreate), shell.Workspace.ID) {
		t.Fatalf("workspace new footer action should show create feedback, got %#v", shell.Toasts)
	}

	if err := runtime.Post(ShellShortcutActionMsg{Invocation: actiondomain.Invocation{ID: "menu.workspace", SourceActionID: "menu.workspace"}}); err != nil {
		t.Fatalf("reenter workspace mode before rename: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace mode before rename: %v", err)
	}
	renameAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "workspace.rename", "")
	if err := host.SendInput(mouseEventAt(renameAction.Rect)); err != nil {
		t.Fatalf("send workspace rename footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace rename footer click: %v", err)
	}
	shell = runtime.State().Shell.EnsureDefaults()
	if !shell.Overlay.Open || shell.Overlay.Prompt.Purpose != "workspace.rename" || shell.Overlay.Prompt.Value != "workspace 2" {
		t.Fatalf("workspace rename footer action should open active workspace rename prompt, got %#v", shell.Overlay)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("workspace new/rename footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeWorkspaceSwitchFooterActions(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	shell, result := state.DefaultShell().ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceCreate, Name: "remote"})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("create workspace: %#v", result)
	}
	remoteWorkspaceID := result.ID
	shell, result = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandWorkspaceSwitch, TargetID: state.DefaultWorkspaceID})
	if result.Status != state.WorkbenchCommandOK {
		t.Fatalf("switch workspace: %#v", result)
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   shell.SetInteractionMode(state.InteractionModeWorkspace),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}

	nextAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "workspace.next", "")
	if err := host.SendInput(mouseEventAt(nextAction.Rect)); err != nil {
		t.Fatalf("send workspace next footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace next footer click: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().Workspace.ID; got != remoteWorkspaceID {
		t.Fatalf("workspace next footer action should activate remote workspace, got %q", got)
	}

	prevAction := frameActionHitRegion(t, lastRuntimeFrame(t, host), "workspace.previous", "")
	if err := host.SendInput(mouseEventAt(prevAction.Rect)); err != nil {
		t.Fatalf("send workspace previous footer click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain workspace previous footer click: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().Workspace.ID; got != state.DefaultWorkspaceID {
		t.Fatalf("workspace previous footer action should return to default workspace, got %q", got)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("workspace switch footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeSingleWorkspaceSwitchFooterActionsAreHidden(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(110, 24)
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5, Cols: 100, Rows: 24},
	}
	runtime := NewInteractiveRuntime(
		state.Root{
			Shell:   state.DefaultShell().SetInteractionMode(state.InteractionModeWorkspace),
			Session: state.TerminalSessionStore{}.Attach("term-1", 5, 100, 24),
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain render: %v", err)
	}
	frame := lastRuntimeFrame(t, host)
	assertFrameMissingActionHitRegion(t, frame, "workspace.next")
	assertFrameMissingActionHitRegion(t, frame, "workspace.previous")
	if len(terminal.Inputs) != 0 {
		t.Fatalf("unavailable single workspace switch footer actions must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func workspaceIDExistsForTest(workspaces []state.WorkspaceState, id string) bool {
	for _, workspace := range workspaces {
		if workspace.ID == id {
			return true
		}
	}
	return false
}

func toastExistsForTest(toasts []state.ToastState, title string, body string) bool {
	for _, toast := range toasts {
		if toast.Title == title && toast.Body == body {
			return true
		}
	}
	return false
}

func TestAppRuntimeTiledPaneClickDeactivatesFloatingFocus(t *testing.T) {
	host := NewFakeTerminalHost(8)
	root := state.Root{
		Shell: state.DefaultShell().
			SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical).
			FocusPane(state.PaneCommandTarget{PaneID: "pane-main"}),
	}
	var result state.FloatingCommandResult
	root.Shell, result = root.Shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "floating", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 30, Y: 4, W: 24, H: 8},
		BoundsW:  90,
		BoundsH:  28,
	})
	if result.Status != state.FloatingCommandOK || root.Shell.ActiveFloatingID() == "" {
		t.Fatalf("expected active floating setup, result=%#v shell=%#v", result, root.Shell)
	}

	runtime := newShellHitRuntime(root, host)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	vmWithFloating := render.NewRenderVMBuilder().Build(runtime.State())
	if len(vmWithFloating.Shell.Layout.Panels) < 2 || vmWithFloating.Shell.Layout.Panels[1].Active || len(vmWithFloating.Shell.Layout.Floating) != 1 || !vmWithFloating.Shell.Layout.Floating[0].Active {
		t.Fatalf("active floating should dim tiled panes before click, panels=%#v floating=%#v", vmWithFloating.Shell.Layout.Panels, vmWithFloating.Shell.Layout.Floating)
	}

	content := frameHitRegion(t, lastRuntimeFrame(t, host), render.HitRegionPaneContent, "pane-2")
	if err := host.SendInput(mouseEventAt(content.Rect)); err != nil {
		t.Fatalf("send tiled pane click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tiled pane click: %v", err)
	}
	shell := runtime.State().Shell.EnsureDefaults()
	floatings := shell.ActiveFloatings()
	if shell.ActiveFloatingID() != "" || len(floatings) != 1 || floatings[0].Active {
		t.Fatalf("tiled pane click should deactivate floating without closing it, shell=%#v", shell)
	}
	if shell.ActivePaneID != "pane-2" {
		t.Fatalf("tiled pane click should focus pane-2, shell=%#v", shell)
	}
	vmAfterClick := render.NewRenderVMBuilder().Build(runtime.State())
	if len(vmAfterClick.Shell.Layout.Panels) < 2 || !vmAfterClick.Shell.Layout.Panels[1].Active || len(vmAfterClick.Shell.Layout.Floating) != 1 || vmAfterClick.Shell.Layout.Floating[0].Active {
		t.Fatalf("tiled pane should regain active style and floating should render inactive, panels=%#v floating=%#v", vmAfterClick.Shell.Layout.Panels, vmAfterClick.Shell.Layout.Floating)
	}
}

func TestAppRuntimeDispatchesMouseSplitActions(t *testing.T) {
	downHost := NewFakeTerminalHost(8)
	downRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, downHost)
	if err := downRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post split-down initial render: %v", err)
	}
	if err := downRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split-down initial render: %v", err)
	}
	downAction := frameHitRegionByAction(t, lastRuntimeFrame(t, downHost), render.HitRegionPaneAction, "pane.split-down", state.DefaultPaneID)
	if err := downHost.SendInput(mouseEventAt(downAction.Rect)); err != nil {
		t.Fatalf("send split-down click: %v", err)
	}
	if err := downRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split-down click: %v", err)
	}
	downSplit := downRuntime.State().Shell.Workspace.Tabs[0].RootSplit
	if len(downRuntime.State().Shell.Workspace.Tabs[0].Panes) != 2 || downSplit.Direction != state.SplitDirectionHorizontal {
		t.Fatalf("split-down action should create horizontal split, shell=%#v", downRuntime.State().Shell)
	}

	rightHost := NewFakeTerminalHost(8)
	rightRuntime := newShellHitRuntime(state.Root{Shell: state.DefaultShell()}, rightHost)
	if err := rightRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post split-right initial render: %v", err)
	}
	if err := rightRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split-right initial render: %v", err)
	}
	rightAction := frameHitRegionByAction(t, lastRuntimeFrame(t, rightHost), render.HitRegionPaneAction, "pane.split-right", state.DefaultPaneID)
	if err := rightHost.SendInput(mouseEventAt(rightAction.Rect)); err != nil {
		t.Fatalf("send split-right click: %v", err)
	}
	if err := rightRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split-right click: %v", err)
	}
	rightSplit := rightRuntime.State().Shell.Workspace.Tabs[0].RootSplit
	if len(rightRuntime.State().Shell.Workspace.Tabs[0].Panes) != 2 || rightSplit.Direction != state.SplitDirectionVertical {
		t.Fatalf("split-right action should create vertical split, shell=%#v", rightRuntime.State().Shell)
	}
	if rightRuntime.State().Shell.ActivePaneID == state.DefaultPaneID || downRuntime.State().Shell.ActivePaneID == state.DefaultPaneID {
		t.Fatalf("mouse split actions should activate new panes, down=%#v right=%#v", downRuntime.State().Shell.ActivePaneID, rightRuntime.State().Shell.ActivePaneID)
	}
}

func TestAppRuntimeSplitActionOnHorizontalDividerPaneWinsOverResize(t *testing.T) {
	host := NewFakeTerminalHost(8)
	root := state.Root{
		Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			SplitActivePane(state.PaneState{ID: "pane-bottom", Title: "bottom", Kind: state.PaneTerminalLive}, state.SplitDirectionHorizontal),
	}
	runtime := newShellHitRuntime(root, host)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}

	action := frameHitRegionByAction(t, lastRuntimeFrame(t, host), render.HitRegionPaneAction, "pane.split-down", "pane-bottom")
	if err := host.SendInput(mouseEventAt(action.Rect)); err != nil {
		t.Fatalf("send bottom split action: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain bottom split action: %v", err)
	}

	shell := runtime.State().Shell.EnsureDefaults()
	if len(shell.Workspace.Tabs[0].Panes) != 3 || !shell.HasPane(state.PaneCommandTarget{PaneID: "pane-2"}) {
		t.Fatalf("bottom pane split icon should create a new pane instead of starting divider resize, shell=%#v", shell)
	}
	if shell.ActivePaneID != "pane-2" || runtime.mouseDrag.Active {
		t.Fatalf("split action should activate new pane and not leave resize drag state, active=%q drag=%#v", shell.ActivePaneID, runtime.mouseDrag)
	}
	for _, toast := range shell.Toasts {
		if toast.Body == "missing new pane id" || toast.Body == "target pane not found" {
			t.Fatalf("split action should not produce invalid toast, got %#v", shell.Toasts)
		}
	}
}

func TestAppRuntimeDragsPaneResizeHitRegions(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 7, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(32)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split-line: %v", err)
	}
	if err := runtime.Post(ShellSplitActivePaneMsg{Pane: state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive}, Direction: state.SplitDirectionVertical}); err != nil {
		t.Fatalf("post split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split: %v", err)
	}
	beforeInputCount := len(terminal.Inputs)
	beforeResizeCount := len(terminal.Resizes)
	beforeToastCount := len(runtime.State().Shell.Toasts)
	beforeRects := paneLayoutRects(runtime.State())
	resizeRegion := framePaneResizeRegion(t, lastRuntimeFrame(t, host), state.DefaultPaneID, state.PaneResizeRight)
	start := mouseEventAt(resizeRegion.Rect)
	start.Mouse = input.MouseLeft
	if err := host.SendInput(start); err != nil {
		t.Fatalf("send drag start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain drag start: %v", err)
	}
	if runtime.mouseDrag.PaneID != state.DefaultPaneID || runtime.mouseDrag.Direction != state.PaneResizeRight {
		t.Fatalf("expected active pane resize drag state, got %#v", runtime.mouseDrag)
	}

	drag := start
	drag.Mouse = input.MouseLeftDrag
	drag.Col += 5
	if err := host.SendInput(drag); err != nil {
		t.Fatalf("send drag move: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain drag move: %v", err)
	}
	split := runtime.State().Shell.Workspace.Tabs[0].RootSplit
	if split.BiasCells != 5 {
		t.Fatalf("expected horizontal divider drag to resize split bias, got %#v", split)
	}
	grownRects := paneLayoutRects(runtime.State())
	if grownRects[state.DefaultPaneID].X != beforeRects[state.DefaultPaneID].X || grownRects["pane-2"].X+grownRects["pane-2"].W != beforeRects["pane-2"].X+beforeRects["pane-2"].W {
		t.Fatalf("dragging right divider should keep opposite pane edges anchored, before=%#v after=%#v", beforeRects, grownRects)
	}
	reverseDrag := start
	reverseDrag.Mouse = input.MouseLeftDrag
	reverseDrag.Col -= 2
	if err := host.SendInput(reverseDrag); err != nil {
		t.Fatalf("send reverse drag move: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain reverse drag move: %v", err)
	}
	split = runtime.State().Shell.Workspace.Tabs[0].RootSplit
	if split.BiasCells != -2 || runtime.mouseDrag.Direction != state.PaneResizeRight {
		t.Fatalf("dragging the same right edge back must keep the left edge anchored, split=%#v drag=%#v", split, runtime.mouseDrag)
	}
	shrunkRects := paneLayoutRects(runtime.State())
	if shrunkRects[state.DefaultPaneID].X != beforeRects[state.DefaultPaneID].X || shrunkRects["pane-2"].X+shrunkRects["pane-2"].W != beforeRects["pane-2"].X+beforeRects["pane-2"].W {
		t.Fatalf("dragging right divider backward should still keep opposite pane edges anchored, before=%#v after=%#v", beforeRects, shrunkRects)
	}
	if len(terminal.Resizes) <= beforeResizeCount {
		t.Fatalf("pane drag resize should schedule active terminal content resize, got %#v", terminal.Resizes)
	}
	if len(terminal.Inputs) != beforeInputCount {
		t.Fatalf("pane resize drag must not leak to terminal input, got %#v", terminal.Inputs)
	}
	if len(runtime.State().Shell.Toasts) != beforeToastCount {
		t.Fatalf("pane resize drag success should not add toast, before=%d after=%#v", beforeToastCount, runtime.State().Shell.Toasts)
	}

	release := reverseDrag
	release.Mouse = input.MouseLeftUp
	if err := host.SendInput(release); err != nil {
		t.Fatalf("send drag release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain drag release: %v", err)
	}
	if runtime.mouseDrag.Active {
		t.Fatalf("release should clear drag state, got %#v", runtime.mouseDrag)
	}

	afterRelease := release
	afterRelease.Mouse = input.MouseLeftDrag
	afterRelease.Col += 3
	if err := host.SendInput(afterRelease); err != nil {
		t.Fatalf("send drag after release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain drag after release: %v", err)
	}
	if got := runtime.State().Shell.Workspace.Tabs[0].RootSplit.BiasCells; got != -2 {
		t.Fatalf("drag after release must not resize, bias=%d", got)
	}
}

func TestAppRuntimeDragsHorizontalPaneDividerResize(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newShellHitRuntime(
		state.Root{Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			SplitActivePane(state.PaneState{ID: "pane-bottom", Title: "bottom", Kind: state.PaneTerminalLive}, state.SplitDirectionHorizontal)},
		host,
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	resizeRegion := framePaneResizeRegion(t, lastRuntimeFrame(t, host), state.DefaultPaneID, state.PaneResizeDown)
	if resizeRegion.Direction != string(state.PaneResizeDown) {
		t.Fatalf("expected horizontal split divider direction down, got %#v", resizeRegion)
	}
	start := mouseEventAt(resizeRegion.Rect)
	start.Mouse = input.MouseLeft
	drag := start
	drag.Mouse = input.MouseLeftDrag
	drag.Row += 3
	release := drag
	release.Mouse = input.MouseLeftUp
	for _, event := range []input.InputEvent{start, drag, release} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send event %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain event %#v: %v", event, err)
		}
	}
	if got := runtime.State().Shell.Workspace.Tabs[0].RootSplit.BiasCells; got != 3 {
		t.Fatalf("expected vertical drag to update horizontal split bias, got %#v", runtime.State().Shell.Workspace.Tabs[0].RootSplit)
	}
}

func TestAppRuntimeClearsStaleMouseDragWhenReleaseIsMissing(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := newShellHitRuntime(
		state.Root{Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			SplitActivePane(state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive}, state.SplitDirectionVertical)},
		host,
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	resizeRegion := framePaneResizeRegion(t, lastRuntimeFrame(t, host), state.DefaultPaneID, state.PaneResizeRight)
	start := mouseEventAt(resizeRegion.Rect)
	start.Mouse = input.MouseLeft
	if err := host.SendInput(start); err != nil {
		t.Fatalf("send drag start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain drag start: %v", err)
	}
	if !runtime.mouseDrag.Active {
		t.Fatalf("drag start should arm mouse drag state")
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send key after missing release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain key after missing release: %v", err)
	}
	if runtime.mouseDrag.Active {
		t.Fatalf("non-mouse input should clear stale drag state, got %#v", runtime.mouseDrag)
	}

	if err := host.SendInput(start); err != nil {
		t.Fatalf("send second drag start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain second drag start: %v", err)
	}
	if !runtime.mouseDrag.Active {
		t.Fatalf("new mouse down should still start a fresh drag")
	}
}

func TestAppRuntimeDragsNestedPaneResizeOnlyChangesExactDivider(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 9, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(32)
	host.SetSize(90, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 90, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split-line: %v", err)
	}
	if err := runtime.Post(ShellSplitActivePaneMsg{Pane: state.PaneState{ID: "pane-middle", Title: "middle", Kind: state.PaneTerminalLive}, Direction: state.SplitDirectionVertical}); err != nil {
		t.Fatalf("post middle split: %v", err)
	}
	if err := runtime.Post(ShellSplitActivePaneMsg{Pane: state.PaneState{ID: "pane-right", Title: "right", Kind: state.PaneTerminalLive}, Direction: state.SplitDirectionVertical}); err != nil {
		t.Fatalf("post right split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}
	beforeInputCount := len(terminal.Inputs)
	beforeToastCount := len(runtime.State().Shell.Toasts)
	beforeRects := paneLayoutRects(runtime.State())

	resizeRegion := framePaneResizeRegion(t, lastRuntimeFrame(t, host), "pane-middle", state.PaneResizeRight)
	if resizeRegion.SplitPath != "root/1" {
		t.Fatalf("expected nested divider split path, got %#v", resizeRegion)
	}
	start := mouseEventAt(resizeRegion.Rect)
	start.Mouse = input.MouseLeft
	if err := host.SendInput(start); err != nil {
		t.Fatalf("send nested drag start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain nested drag start: %v", err)
	}
	if runtime.mouseDrag.SplitPath != "root/1" || runtime.mouseDrag.PaneID != "pane-middle" {
		t.Fatalf("drag state should keep exact nested split path, got %#v", runtime.mouseDrag)
	}

	drag := start
	drag.Mouse = input.MouseLeftDrag
	drag.Col -= 5
	if err := host.SendInput(drag); err != nil {
		t.Fatalf("send nested drag move: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain nested drag move: %v", err)
	}
	root := runtime.State().Shell.Workspace.Tabs[0].RootSplit
	if root.BiasCells != 0 || len(root.Children) < 2 || root.Children[1].BiasCells != -5 {
		t.Fatalf("nested divider drag should not mutate outer split, got %#v", root)
	}
	afterRects := paneLayoutRects(runtime.State())
	if afterRects[state.DefaultPaneID] != beforeRects[state.DefaultPaneID] {
		t.Fatalf("outer left pane must stay anchored, before=%#v after=%#v", beforeRects[state.DefaultPaneID], afterRects[state.DefaultPaneID])
	}
	if afterRects["pane-middle"].X != beforeRects["pane-middle"].X || afterRects["pane-middle"].W != beforeRects["pane-middle"].W-5 {
		t.Fatalf("left side of nested divider should shrink by drag delta, before=%#v after=%#v", beforeRects["pane-middle"], afterRects["pane-middle"])
	}
	if afterRects["pane-right"].X != beforeRects["pane-right"].X-5 || afterRects["pane-right"].X+afterRects["pane-right"].W != beforeRects["pane-right"].X+beforeRects["pane-right"].W {
		t.Fatalf("right side of nested divider should grow while outer edge stays anchored, before=%#v after=%#v", beforeRects["pane-right"], afterRects["pane-right"])
	}
	if len(terminal.Inputs) != beforeInputCount {
		t.Fatalf("nested pane resize drag must not leak to terminal input, got %#v", terminal.Inputs)
	}
	if len(runtime.State().Shell.Toasts) != beforeToastCount {
		t.Fatalf("nested pane resize drag success should not add toast, before=%d after=%#v", beforeToastCount, runtime.State().Shell.Toasts)
	}
}

func TestAppRuntimeDragsFourColumnPaneResizeOnlyAdjacentColumns(t *testing.T) {
	leftRuntime, leftHost, leftTerminal := newFourColumnPaneRuntime(t)
	beforeLeftRects := paneLayoutRects(leftRuntime.State())
	beforeLeftInputCount := len(leftTerminal.Inputs)
	beforeLeftToastCount := len(leftRuntime.State().Shell.Toasts)

	leftDivider := framePaneResizeRegion(t, lastRuntimeFrame(t, leftHost), state.DefaultPaneID, state.PaneResizeRight)
	if leftDivider.ResizeBeforePaneID != state.DefaultPaneID || leftDivider.ResizeAfterPaneID != "pane-2" {
		t.Fatalf("expected pane-2 left divider to target pane-main/pane-2, got %#v", leftDivider)
	}
	leftStart := mouseEventAt(leftDivider.Rect)
	leftStart.Mouse = input.MouseLeft
	if err := leftHost.SendInput(leftStart); err != nil {
		t.Fatalf("send left divider start: %v", err)
	}
	if err := leftRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain left divider start: %v", err)
	}
	leftDrag := leftStart
	leftDrag.Mouse = input.MouseLeftDrag
	leftDrag.Col -= 4
	if err := leftHost.SendInput(leftDrag); err != nil {
		t.Fatalf("send left divider drag: %v", err)
	}
	if err := leftRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain left divider drag: %v", err)
	}
	afterLeftRects := paneLayoutRects(leftRuntime.State())
	if afterLeftRects[state.DefaultPaneID].W != beforeLeftRects[state.DefaultPaneID].W-4 || afterLeftRects["pane-2"].X != beforeLeftRects["pane-2"].X-4 || afterLeftRects["pane-2"].W != beforeLeftRects["pane-2"].W+4 {
		t.Fatalf("dragging pane-2 left edge left should only trade width with pane-main, before=%#v after=%#v", beforeLeftRects, afterLeftRects)
	}
	if afterLeftRects["pane-3"] != beforeLeftRects["pane-3"] || afterLeftRects["pane-4"] != beforeLeftRects["pane-4"] {
		t.Fatalf("dragging pane-2 left edge must not grow later panes, before=%#v after=%#v", beforeLeftRects, afterLeftRects)
	}
	if len(leftTerminal.Inputs) != beforeLeftInputCount || len(leftRuntime.State().Shell.Toasts) != beforeLeftToastCount {
		t.Fatalf("four-column left drag should not leak input or add toast, inputs=%#v toasts=%#v", leftTerminal.Inputs, leftRuntime.State().Shell.Toasts)
	}

	rightRuntime, rightHost, rightTerminal := newFourColumnPaneRuntime(t)
	beforeRightRects := paneLayoutRects(rightRuntime.State())
	beforeRightInputCount := len(rightTerminal.Inputs)
	beforeRightToastCount := len(rightRuntime.State().Shell.Toasts)

	rightDivider := framePaneResizeRegion(t, lastRuntimeFrame(t, rightHost), "pane-2", state.PaneResizeRight)
	if rightDivider.ResizeBeforePaneID != "pane-2" || rightDivider.ResizeAfterPaneID != "pane-3" {
		t.Fatalf("expected pane-2 right divider to target pane-2/pane-3, got %#v", rightDivider)
	}
	rightStart := mouseEventAt(rightDivider.Rect)
	rightStart.Mouse = input.MouseLeft
	if err := rightHost.SendInput(rightStart); err != nil {
		t.Fatalf("send right divider start: %v", err)
	}
	if err := rightRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain right divider start: %v", err)
	}
	rightDrag := rightStart
	rightDrag.Mouse = input.MouseLeftDrag
	rightDrag.Col += 5
	if err := rightHost.SendInput(rightDrag); err != nil {
		t.Fatalf("send right divider drag: %v", err)
	}
	if err := rightRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain right divider drag: %v", err)
	}
	afterRightRects := paneLayoutRects(rightRuntime.State())
	if afterRightRects["pane-2"].W != beforeRightRects["pane-2"].W+5 || afterRightRects["pane-3"].X != beforeRightRects["pane-3"].X+5 || afterRightRects["pane-3"].W != beforeRightRects["pane-3"].W-5 {
		t.Fatalf("dragging pane-2 right edge right should only trade width with pane-3, before=%#v after=%#v", beforeRightRects, afterRightRects)
	}
	if afterRightRects["pane-4"] != beforeRightRects["pane-4"] {
		t.Fatalf("dragging pane-2 right edge must not shrink pane-4, before=%#v after=%#v", beforeRightRects["pane-4"], afterRightRects["pane-4"])
	}
	if len(rightTerminal.Inputs) != beforeRightInputCount || len(rightRuntime.State().Shell.Toasts) != beforeRightToastCount {
		t.Fatalf("four-column right drag should not leak input or add toast, inputs=%#v toasts=%#v", rightTerminal.Inputs, rightRuntime.State().Shell.Toasts)
	}
}

func TestAppRuntimeDragsStackedRightColumnResizeAsSharedWidthGroup(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 12, Cols: 80, Rows: 20},
	}
	host := NewFakeTerminalHost(64)
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 80, Rows: 20},
		Shell:    stackedRightColumnShellForTest(),
	}
	runtime := newShellHitRuntimeWithTerminal(root, host, terminal)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post stacked initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stacked initial render: %v", err)
	}
	beforeRects := paneLayoutRects(runtime.State())
	beforeInputCount := len(terminal.Inputs)
	beforeToastCount := len(runtime.State().Shell.Toasts)

	divider := framePaneResizeRegion(t, lastRuntimeFrame(t, host), "left", state.PaneResizeRight)
	if divider.ResizeBeforePaneID != "left" || divider.ResizeAfterPaneID != "top" {
		t.Fatalf("expected root divider to target left/top boundary panes, got %#v", divider)
	}
	start := mouseEventAt(divider.Rect)
	start.Mouse = input.MouseLeft
	if err := host.SendInput(start); err != nil {
		t.Fatalf("send stacked drag start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stacked drag start: %v", err)
	}
	drag := start
	drag.Mouse = input.MouseLeftDrag
	drag.Col -= 6
	if err := host.SendInput(drag); err != nil {
		t.Fatalf("send stacked drag move: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stacked drag move: %v", err)
	}

	afterRects := paneLayoutRects(runtime.State())
	if afterRects["left"].W != beforeRects["left"].W-6 {
		t.Fatalf("left column should shrink by drag delta, before=%#v after=%#v", beforeRects["left"], afterRects["left"])
	}
	for _, paneID := range []string{"top", "middle-left", "bottom"} {
		if afterRects[paneID].X != beforeRects[paneID].X-6 || afterRects[paneID].W != beforeRects[paneID].W+6 {
			t.Fatalf("%s should move with the shared right-column boundary, before=%#v after=%#v", paneID, beforeRects[paneID], afterRects[paneID])
		}
	}
	if afterRects["middle-right"].X != beforeRects["middle-right"].X || afterRects["middle-right"].W != beforeRects["middle-right"].W {
		t.Fatalf("nested right child should keep its outer anchor and width, before=%#v after=%#v", beforeRects["middle-right"], afterRects["middle-right"])
	}
	if len(terminal.Inputs) != beforeInputCount || len(runtime.State().Shell.Toasts) != beforeToastCount {
		t.Fatalf("stacked column drag should not leak input or add toast, inputs=%#v toasts=%#v", terminal.Inputs, runtime.State().Shell.Toasts)
	}
}

func TestAppRuntimeDragsFloatingMoveAndResizeHitRegions(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 3, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	shell, result := state.DefaultShell().ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "floating", Kind: state.PaneEmpty},
		Title:    "floating",
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		BoundsW:  80,
		BoundsH:  24,
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating for test: %#v", result)
	}
	root := state.Root{
		Shell: shell,
	}
	runtime := newShellHitRuntimeWithTerminal(root, host, terminal)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post floating initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating initial render: %v", err)
	}
	beforeInputCount := len(terminal.Inputs)
	beforeToastCount := len(runtime.State().Shell.Toasts)

	moveRegion := frameActionHitRegion(t, lastRuntimeFrame(t, host), "floating.move-drag", "floating-pane-1")
	if !moveRegion.Floating {
		t.Fatalf("floating move drag hit region should carry floating flag, got %#v", moveRegion)
	}
	moveStart := mouseEventAt(moveRegion.Rect)
	if err := host.SendInput(moveStart); err != nil {
		t.Fatalf("send floating move start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating move start: %v", err)
	}
	if runtime.mouseDrag.Kind != mouseDragFloatingMove || runtime.mouseDrag.FloatingID != "floating-1" {
		t.Fatalf("expected floating move drag state, got %#v", runtime.mouseDrag)
	}
	framesBeforeMove := len(host.Frames())
	moveDrag := moveStart
	moveDrag.Mouse = input.MouseLeftDrag
	moveDrag.Col += 4
	moveDrag.Row += 3
	if err := host.SendInput(moveDrag); err != nil {
		t.Fatalf("send floating move drag: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating move drag: %v", err)
	}
	moved := runtime.State().Shell.ActiveFloatings()[0].Rect
	if moved.X != 14 || moved.Y != 7 {
		t.Fatalf("floating title drag should move rect, got %#v", moved)
	}
	if lastRuntimeNewFrame(t, host, framesBeforeMove).Metadata.ForceFullRepaint {
		t.Fatalf("floating move frame must use row-diff repaint, got %#v", lastRuntimeNewFrame(t, host, framesBeforeMove).Metadata)
	}
	if len(runtime.State().Shell.Toasts) != beforeToastCount {
		t.Fatalf("floating move drag success should not add toast, before=%d after=%#v", beforeToastCount, runtime.State().Shell.Toasts)
	}
	moveRelease := moveDrag
	moveRelease.Mouse = input.MouseLeftUp
	if err := host.SendInput(moveRelease); err != nil {
		t.Fatalf("send floating move release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating move release: %v", err)
	}
	if runtime.mouseDrag.Active {
		t.Fatalf("floating move release should clear drag state, got %#v", runtime.mouseDrag)
	}

	resizeRegion := frameActionHitRegion(t, lastRuntimeFrame(t, host), "floating.resize-drag", "floating-pane-1")
	if !resizeRegion.Floating {
		t.Fatalf("floating resize drag hit region should carry floating flag, got %#v", resizeRegion)
	}
	resizeStart := mouseEventAt(resizeRegion.Rect)
	if err := host.SendInput(resizeStart); err != nil {
		t.Fatalf("send floating resize start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating resize start: %v", err)
	}
	if runtime.mouseDrag.Kind != mouseDragFloatingResize || runtime.mouseDrag.FloatingID != "floating-1" {
		t.Fatalf("expected floating resize drag state, got %#v", runtime.mouseDrag)
	}
	beforeResize := runtime.State().Shell.ActiveFloatings()[0].Rect
	framesBeforeResize := len(host.Frames())
	resizeDrag := resizeStart
	resizeDrag.Mouse = input.MouseLeftDrag
	resizeDrag.Col += 6
	resizeDrag.Row += 2
	if err := host.SendInput(resizeDrag); err != nil {
		t.Fatalf("send floating resize drag: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating resize drag: %v", err)
	}
	resized := runtime.State().Shell.ActiveFloatings()[0].Rect
	if resized.W != beforeResize.W+6 || resized.H != beforeResize.H+2 {
		t.Fatalf("floating resize drag should resize rect, before=%#v after=%#v", beforeResize, resized)
	}
	if lastRuntimeNewFrame(t, host, framesBeforeResize).Metadata.ForceFullRepaint {
		t.Fatalf("floating resize frame must use row-diff repaint, got %#v", lastRuntimeNewFrame(t, host, framesBeforeResize).Metadata)
	}
	if len(runtime.State().Shell.Toasts) != beforeToastCount {
		t.Fatalf("floating resize drag success should not add toast, before=%d after=%#v", beforeToastCount, runtime.State().Shell.Toasts)
	}
	resizeRelease := resizeDrag
	resizeRelease.Mouse = input.MouseLeftUp
	if err := host.SendInput(resizeRelease); err != nil {
		t.Fatalf("send floating resize release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating resize release: %v", err)
	}
	afterRelease := resizeRelease
	afterRelease.Mouse = input.MouseLeftDrag
	afterRelease.Col += 3
	afterRelease.Row += 3
	if err := host.SendInput(afterRelease); err != nil {
		t.Fatalf("send floating drag after release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating drag after release: %v", err)
	}
	if got := runtime.State().Shell.ActiveFloatings()[0].Rect; got != resized {
		t.Fatalf("drag after release must not resize floating, before=%#v after=%#v", resized, got)
	}
	if len(terminal.Inputs) != beforeInputCount {
		t.Fatalf("floating drag must not leak to terminal input, got %#v", terminal.Inputs)
	}
}

func TestAppRuntimeDragsClipboardHistoryDivider(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 30)
	root := state.Root{
		Shell: state.DefaultShell().OpenClipboardHistory(),
		Clipboard: state.ClipboardStore{Entries: []state.ClipboardEntry{
			{ID: "clip:1", Title: "git commit", Text: "git commit -m fix terminal\nsecond preview line", Preview: "git commit -m fix terminal"},
		}},
	}
	runtime := newShellHitRuntime(root, host)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post clipboard history render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clipboard history render: %v", err)
	}

	divider := frameActionHitRegion(t, lastRuntimeFrame(t, host), render.ActionClipboardHistoryDividerDrag.String(), "")
	start := mouseEventAt(divider.Rect)
	if err := host.SendInput(start); err != nil {
		t.Fatalf("send clipboard divider drag start: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clipboard divider drag start: %v", err)
	}
	if runtime.mouseDrag.Kind != mouseDragClipboardDivider {
		t.Fatalf("expected clipboard divider drag state, got %#v", runtime.mouseDrag)
	}

	drag := start
	drag.Mouse = input.MouseLeftDrag
	drag.Col += 7
	if err := host.SendInput(drag); err != nil {
		t.Fatalf("send clipboard divider drag move: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clipboard divider drag move: %v", err)
	}
	if got := state.ClipboardHistoryNameWidth(runtime.State().Shell.Overlay); got != state.DefaultClipboardHistoryNameWidth+7 {
		t.Fatalf("clipboard divider drag should resize name column, got %d", got)
	}

	release := drag
	release.Mouse = input.MouseLeftUp
	if err := host.SendInput(release); err != nil {
		t.Fatalf("send clipboard divider drag release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain clipboard divider drag release: %v", err)
	}
	if runtime.mouseDrag.Active {
		t.Fatalf("clipboard divider release should clear drag state, got %#v", runtime.mouseDrag)
	}
}

func TestAppRuntimeMouseHitPriorityAndMissFallback(t *testing.T) {
	host := NewFakeTerminalHost(8)
	root := state.Root{
		Shell: state.DefaultShell().
			OpenTerminalPicker().
			AddToast(state.ToastSpec{ID: "toast-1", Title: "notice"}),
	}
	var inputSeen int
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewShellReducer(), func(root state.Root, msg Msg) (state.Root, []Effect) {
			if _, ok := msg.(InputMsg); ok {
				inputSeen++
			}
			return root, nil
		}),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	overlay := frameHitRegion(t, lastRuntimeFrame(t, host), render.HitRegionOverlay, "")
	if err := host.SendInput(mouseEventAt(overlay.Rect)); err != nil {
		t.Fatalf("send overlay click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("overlay drain: %v", err)
	}
	if runtime.State().Shell.Overlay.Open {
		t.Fatalf("overlay hit should close overlay, got %#v", runtime.State().Shell.Overlay)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft, Row: 999, Col: 999}); err != nil {
		t.Fatalf("send miss click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("miss drain: %v", err)
	}
	if inputSeen == 0 {
		t.Fatal("missed mouse hit should continue through InputMsg fallback")
	}
}

func TestInteractiveRuntimeTerminalMouseTrackingPassthroughOnlyFromContent(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	frame := lastRuntimeFrame(t, host)
	content := frameHitRegion(t, frame, render.HitRegionPaneContent, state.DefaultPaneID)
	mouse := mouseEventAt(content.Rect)
	mouse.Mouse = input.MouseRight
	mouse.RawSeq = "\x1b[<2;10;4M"
	if err := host.SendInput(mouse); err != nil {
		t.Fatalf("send mouse without tracking: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain mouse without tracking: %v", err)
	}
	if len(terminal.Inputs) != 0 {
		t.Fatalf("mouse without terminal tracking must not passthrough, got %#v", terminal.Inputs)
	}

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       78,
		Rows:       22,
		Lines:      []string{"tracking"},
		Modes:      state.LiveTerminalModes{MouseTracking: true, MouseSGR: true},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain live surface: %v", err)
	}
	if err := host.SendInput(mouse); err != nil {
		t.Fatalf("send tracked content mouse: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tracked content mouse: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x1b[<2;1;1M" || terminal.Inputs[0].Event.Col != 1 || terminal.Inputs[0].Event.Row != 1 {
		t.Fatalf("tracked content mouse should passthrough content-local SGR, got %#v", terminal.Inputs)
	}

	runtime.lastHitRegions = append([]render.HitRegion{{
		Kind:     render.HitRegionContentAction,
		Rect:     content.Rect,
		PaneID:   state.DefaultPaneID,
		ActionID: render.ActionEmptyAttach.String(),
	}}, runtime.lastHitRegions...)
	actionMouse := mouseEventAt(content.Rect)
	actionMouse.Mouse = input.MouseRight
	actionMouse.RawSeq = "\x1b[<2;11;5M"
	if err := host.SendInput(actionMouse); err != nil {
		t.Fatalf("send content action mouse: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain content action mouse: %v", err)
	}
	if len(terminal.Inputs) != 1 {
		t.Fatalf("foreground content action must not passthrough to terminal, got %#v", terminal.Inputs)
	}

	frame = lastRuntimeFrame(t, host)
	chrome := frameHitRegion(t, frame, render.HitRegionPaneChrome, state.DefaultPaneID)
	chromeMouse := mouseEventAt(chrome.Rect)
	chromeMouse.Mouse = input.MouseRight
	chromeMouse.RawSeq = "\x1b[<2;1;1M"
	if err := host.SendInput(chromeMouse); err != nil {
		t.Fatalf("send chrome mouse: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain chrome mouse: %v", err)
	}
	if len(terminal.Inputs) != 1 {
		t.Fatalf("chrome mouse must not passthrough to terminal, got %#v", terminal.Inputs)
	}

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: state.DefaultPaneID},
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"},
		Source:         state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post split tracked pane: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-2",
		Cols:       38,
		Rows:       22,
		Lines:      []string{"tracking-2"},
		Modes:      state.LiveTerminalModes{MouseTracking: true, MouseSGR: true},
	}}); err != nil {
		t.Fatalf("post second live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split tracked pane: %v", err)
	}
	frame = lastRuntimeFrame(t, host)
	inactiveContent := frameHitRegion(t, frame, render.HitRegionPaneContent, state.DefaultPaneID)
	inactiveMouse := mouseEventAt(inactiveContent.Rect)
	inactiveMouse.Mouse = input.MouseRight
	inactiveMouse.RawSeq = "\x1b[<2;12;6M"
	if err := host.SendInput(inactiveMouse); err != nil {
		t.Fatalf("send inactive pane mouse: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain inactive pane mouse: %v", err)
	}
	if len(terminal.Inputs) != 2 {
		t.Fatalf("inactive pane raw mouse should target clicked terminal without leaking to old active terminal, got %#v", terminal.Inputs)
	}
	if got := terminal.Inputs[1]; got.TerminalID != "term-1" || string(got.Bytes) != "\x1b[<2;1;1M" || got.Event.Col != 1 || got.Event.Row != 1 {
		t.Fatalf("inactive pane raw mouse should focus and passthrough content-local event to clicked terminal, got %#v", got)
	}
	if runtime.State().Shell.EnsureDefaults().ActivePaneID != state.DefaultPaneID {
		t.Fatalf("inactive pane raw mouse should focus clicked pane, got %#v", runtime.State().Shell)
	}
}

func TestInteractiveRuntimeTrackedWheelPassthroughDoesNotEnterCopyMode(t *testing.T) {
	core := &testkit.FakeCoreClient{
		LatestResponses: []port.HistoryResult{{Window: historyWindowForApp(
			state.HistoryWindowReplace,
			"term-1",
			"tok-1",
			78,
			7,
			[]state.HistoryRow{{Text: "history", LineID: 20}},
		)}},
	}
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       78,
		Rows:       22,
		Lines:      []string{"tracked wheel"},
		Modes:      state.LiveTerminalModes{MouseTracking: true, MouseSGR: true},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}
	content := frameHitRegion(t, lastRuntimeFrame(t, host), render.HitRegionPaneContent, state.DefaultPaneID)
	wheel := mouseEventAt(content.Rect)
	wheel.Mouse = input.MouseWheelUp
	wheel.RawSeq = "\x1b[<64;10;5M"

	if err := host.SendInput(wheel); err != nil {
		t.Fatalf("send tracked wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tracked wheel: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x1b[<64;1;1M" || terminal.Inputs[0].Event.Col != 1 || terminal.Inputs[0].Event.Row != 1 {
		t.Fatalf("tracked wheel should passthrough content-local SGR, got %#v", terminal.Inputs)
	}
	if len(core.LatestRequests) != 0 || runtime.State().CopyMode.Active || runtime.State().CopyMode.Entering {
		t.Fatalf("tracked wheel must not enter copy/history, latest=%#v copy=%#v", core.LatestRequests, runtime.State().CopyMode)
	}
}

func TestInteractiveRuntimeTrackedWheelPassthroughUsesLegacyEncodingWithoutSGRMode(t *testing.T) {
	core := &testkit.FakeCoreClient{}
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: core},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       78,
		Rows:       22,
		Lines:      []string{"legacy tracked wheel"},
		Modes:      state.LiveTerminalModes{MouseTracking: true, MouseNormal: true},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}
	content := frameHitRegion(t, lastRuntimeFrame(t, host), render.HitRegionPaneContent, state.DefaultPaneID)
	wheel := mouseEventAt(content.Rect)
	wheel.Mouse = input.MouseWheelUp
	wheel.RawSeq = "\x1b[<64;10;5M"

	if err := host.SendInput(wheel); err != nil {
		t.Fatalf("send legacy tracked wheel: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain legacy tracked wheel: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "\x1b[M`!!" || terminal.Inputs[0].Event.Col != 1 || terminal.Inputs[0].Event.Row != 1 {
		t.Fatalf("legacy tracked wheel should passthrough content-local X10 bytes, got %#v", terminal.Inputs)
	}
	if len(core.LatestRequests) != 0 || runtime.State().CopyMode.Active || runtime.State().CopyMode.Entering {
		t.Fatalf("legacy tracked wheel must not enter copy/history, latest=%#v copy=%#v", core.LatestRequests, runtime.State().CopyMode)
	}
}

func TestInteractiveRuntimeTrackedMouseClickPassthroughUsesContentLocalCoordinates(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 80, Rows: 24},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       78,
		Rows:       22,
		Lines:      []string{"tracked click"},
		Modes:      state.LiveTerminalModes{MouseTracking: true, MouseSGR: true},
	}}); err != nil {
		t.Fatalf("post live surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}
	content := frameHitRegion(t, lastRuntimeFrame(t, host), render.HitRegionPaneContent, state.DefaultPaneID)
	click := input.InputEvent{
		Kind:   input.EventKindMouse,
		Mouse:  input.MouseLeft,
		Row:    content.Rect.Y + 3,
		Col:    content.Rect.X + 5,
		RawSeq: "\x1b[<0;99;88M",
	}
	release := click
	release.Mouse = input.MouseLeftUp
	release.RawSeq = "\x1b[<0;99;88m"

	if err := host.SendInput(click); err != nil {
		t.Fatalf("send tracked click: %v", err)
	}
	if err := host.SendInput(release); err != nil {
		t.Fatalf("send tracked release: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tracked click: %v", err)
	}
	if len(terminal.Inputs) != 2 {
		t.Fatalf("tracked click/release should passthrough to terminal, got %#v", terminal.Inputs)
	}
	if got := string(terminal.Inputs[0].Bytes); got != "\x1b[<0;5;3M" {
		t.Fatalf("tracked click should use content-local SGR coords, got %q", got)
	}
	if got := string(terminal.Inputs[1].Bytes); got != "\x1b[<0;5;3m" {
		t.Fatalf("tracked release should use content-local SGR coords, got %q", got)
	}
	if terminal.Inputs[0].Event.Col != 5 || terminal.Inputs[0].Event.Row != 3 || terminal.Inputs[1].Event.Col != 5 || terminal.Inputs[1].Event.Row != 3 {
		t.Fatalf("tracked mouse events should carry content-local event coords, got %#v", terminal.Inputs)
	}
}

func TestAppRuntimeAutoDismissesToastsOnRuntimeTick(t *testing.T) {
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 20)
	root := state.Root{
		Shell: state.DefaultShell().
			AddToast(state.ToastSpec{ID: "short", Severity: state.ToastInfo, Title: "short notice"}).
			AddToast(state.ToastSpec{ID: "pending", Severity: state.ToastError, Title: "pending notice", Pending: true}),
	}
	var inputSeen int
	runtime := NewAppRuntime(
		root,
		ComposeReducers(NewShellReducer(), func(root state.Root, msg Msg) (state.Root, []Effect) {
			if _, ok := msg.(InputMsg); ok {
				inputSeen++
			}
			return root, nil
		}),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	runtime.toastTickInterval = time.Second

	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}
	initialFrameCount := len(host.Frames())
	if len(runtime.State().Shell.Toasts) != 2 || frameContains(lastRuntimeFrame(t, host), "pending notice") {
		t.Fatalf("toast should remain in state but stay hidden initially, state=%#v frame=%#v", runtime.State().Shell.Toasts, lastRuntimeFrame(t, host).Lines)
	}

	now = now.Add(3 * time.Second)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("first timer drain: %v", err)
	}
	if len(host.Frames()) <= initialFrameCount {
		t.Fatalf("runtime timer should trigger redraw, before=%d after=%d", initialFrameCount, len(host.Frames()))
	}
	if len(runtime.State().Shell.Toasts) != 1 || runtime.State().Shell.Toasts[0].ID != "pending" {
		t.Fatalf("short toast should auto-dismiss while pending remains, got %#v", runtime.State().Shell.Toasts)
	}
	if frameContains(lastRuntimeFrame(t, host), "short notice") || frameContains(lastRuntimeFrame(t, host), "pending notice") {
		t.Fatalf("toast text should stay hidden after first timer, got %#v", lastRuntimeFrame(t, host).Lines)
	}

	now = now.Add(5 * time.Second)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("second timer drain: %v", err)
	}
	if len(runtime.State().Shell.Toasts) != 0 || frameContains(lastRuntimeFrame(t, host), "pending notice") {
		t.Fatalf("pending toast should eventually auto-dismiss, state=%#v frame=%#v", runtime.State().Shell.Toasts, lastRuntimeFrame(t, host).Lines)
	}
	if inputSeen != 0 {
		t.Fatalf("runtime toast timer must not leak through terminal input path, inputSeen=%d", inputSeen)
	}
}

func TestAppRuntimeDispatchesProductContentActions(t *testing.T) {
	pickerHost := NewFakeTerminalHost(8)
	pickerRoot := state.Root{
		Shell: state.DefaultShell().
			SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}).
			OpenTerminalPicker(),
		TerminalPool: state.TerminalPoolStore{Status: state.TerminalPoolReady, Items: []state.TerminalPoolItem{{TerminalID: "term-2", Title: "logs", State: "running", Cols: 80, Rows: 20}}},
	}
	pickerTerminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-2", Channel: 7, Cols: 80, Rows: 20}}
	pickerRuntime := newShellHitRuntimeWithTerminal(pickerRoot, pickerHost, pickerTerminal)
	if err := pickerRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post picker render: %v", err)
	}
	if err := pickerRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain picker render: %v", err)
	}
	pickerAction := frameActionHitRegionRow(t, lastRuntimeFrame(t, pickerHost), "picker.attach", "", 1)
	if err := pickerHost.SendInput(mouseEventAt(pickerAction.Rect)); err != nil {
		t.Fatalf("send picker attach click: %v", err)
	}
	if err := pickerRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain picker attach: %v", err)
	}
	if pickerRuntime.State().Shell.EnsureDefaults().ActivePaneID != state.DefaultPaneID || pickerRuntime.State().Shell.Overlay.Open || len(pickerTerminal.Attaches) != 1 || pickerTerminal.Attaches[0].TerminalID != "term-2" {
		t.Fatalf("picker attach should attach terminal into current pane, shell=%#v attaches=%#v", pickerRuntime.State().Shell, pickerTerminal.Attaches)
	}

	emptyHost := NewFakeTerminalHost(8)
	emptyShell := state.DefaultShell()
	emptyShell.Workspace.Tabs[0].Panes[0] = state.PaneState{ID: state.DefaultPaneID, Title: "slot", Kind: state.PaneEmpty, Active: true}
	emptyRuntime := newShellHitRuntime(state.Root{Shell: emptyShell}, emptyHost)
	if err := emptyRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post empty render: %v", err)
	}
	if err := emptyRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain empty render: %v", err)
	}
	emptyAttach := frameActionHitRegion(t, lastRuntimeFrame(t, emptyHost), "empty.attach", state.DefaultPaneID)
	if err := emptyHost.SendInput(mouseEventAt(emptyAttach.Rect)); err != nil {
		t.Fatalf("send empty attach click: %v", err)
	}
	if err := emptyRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain empty attach: %v", err)
	}
	if overlay := emptyRuntime.State().Shell.EnsureDefaults().Overlay; !overlay.Open || overlay.Kind != state.OverlayTerminalPicker || overlay.TargetID != state.DefaultPaneID {
		t.Fatalf("empty attach should open terminal picker for pane, got %#v", overlay)
	}

	closeHost := NewFakeTerminalHost(8)
	closeShell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "empty", Kind: state.PaneEmpty}, state.SplitDirectionVertical)
	closeRuntime := newShellHitRuntime(state.Root{Shell: closeShell}, closeHost)
	if err := closeRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post close render: %v", err)
	}
	if err := closeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain close render: %v", err)
	}
	closeAction := frameActionHitRegion(t, lastRuntimeFrame(t, closeHost), "empty.close", "pane-2")
	if err := closeHost.SendInput(mouseEventAt(closeAction.Rect)); err != nil {
		t.Fatalf("send empty close click: %v", err)
	}
	if err := closeRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain empty close: %v", err)
	}
	if closeRuntime.State().Shell.HasPane(state.PaneCommandTarget{PaneID: "pane-2"}) {
		t.Fatalf("empty close action should close pane-2, got %#v", closeRuntime.State().Shell)
	}

	exitedHost := NewFakeTerminalHost(8)
	exitedRoot := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-exited", State: state.TerminalLiveExited, ExitCode: 23},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}
	exitedTerminal := &testkit.FakeTerminalService{ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-exited", Title: "done", State: string(state.TerminalLiveExited)}}}}
	exitedRuntime := newShellHitRuntimeWithTerminal(exitedRoot, exitedHost, exitedTerminal)
	if err := exitedRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post exited render: %v", err)
	}
	if err := exitedRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exited render: %v", err)
	}
	restartAction := frameActionHitRegion(t, lastRuntimeFrame(t, exitedHost), render.ActionExitedRestart.String(), state.DefaultPaneID)
	if err := exitedHost.SendInput(mouseEventAt(restartAction.Rect)); err != nil {
		t.Fatalf("send exited restart click: %v", err)
	}
	if err := exitedRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exited restart: %v", err)
	}
	if len(exitedTerminal.Lists) == 0 || len(exitedTerminal.Restarts) != 1 || exitedTerminal.Restarts[0].TerminalID != "term-exited" {
		t.Fatalf("exited restart click should query core then restart bound exited terminal, lists=%#v restarts=%#v", exitedTerminal.Lists, exitedTerminal.Restarts)
	}

	runningHost := NewFakeTerminalHost(8)
	runningTerminal := &testkit.FakeTerminalService{ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-exited", Title: "live", State: "running"}}}}
	runningRuntime := newShellHitRuntimeWithTerminal(exitedRoot, runningHost, runningTerminal)
	if err := runningRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post stale exited render: %v", err)
	}
	if err := runningRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stale exited render: %v", err)
	}
	staleRestartAction := frameActionHitRegion(t, lastRuntimeFrame(t, runningHost), render.ActionExitedRestart.String(), state.DefaultPaneID)
	if err := runningHost.SendInput(mouseEventAt(staleRestartAction.Rect)); err != nil {
		t.Fatalf("send stale exited restart click: %v", err)
	}
	if err := runningRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stale exited restart: %v", err)
	}
	if len(runningTerminal.Lists) == 0 || len(runningTerminal.Restarts) != 0 {
		t.Fatalf("stale exited cache must query core and skip restart for running terminal, lists=%#v restarts=%#v", runningTerminal.Lists, runningTerminal.Restarts)
	}

	pickerExitedHost := NewFakeTerminalHost(8)
	pickerExitedTerminal := &testkit.FakeTerminalService{ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-next", Title: "next", State: "running"}}}}
	pickerExitedRuntime := newShellHitRuntimeWithTerminal(exitedRoot, pickerExitedHost, pickerExitedTerminal)
	if err := pickerExitedRuntime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post exited picker render: %v", err)
	}
	if err := pickerExitedRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exited picker render: %v", err)
	}
	exitedPickerAction := frameActionHitRegion(t, lastRuntimeFrame(t, pickerExitedHost), render.ActionExitedReconnect.String(), state.DefaultPaneID)
	if err := pickerExitedHost.SendInput(mouseEventAt(exitedPickerAction.Rect)); err != nil {
		t.Fatalf("send exited picker click: %v", err)
	}
	if err := pickerExitedRuntime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exited picker: %v", err)
	}
	exitedOverlay := pickerExitedRuntime.State().Shell.EnsureDefaults().Overlay
	if !exitedOverlay.Open || exitedOverlay.Kind != state.OverlayTerminalPicker || len(pickerExitedTerminal.Lists) == 0 {
		t.Fatalf("exited picker click should open terminal picker and list pool, overlay=%#v lists=%#v", exitedOverlay, pickerExitedTerminal.Lists)
	}
}

func TestInteractiveRuntimeExitedSessionRKeyRestartsAndReattaches(t *testing.T) {
	host := NewFakeTerminalHost(8)
	terminal := &testkit.FakeTerminalService{
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{TerminalID: "term-session-exited", Title: "done", State: string(state.TerminalLiveExited)}}},
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-session-exited",
			Channel:      11,
			Cols:         80,
			Rows:         24,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface",
			ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
			CanResize:    true,
		},
	}
	root := state.Root{
		Shell: state.DefaultShell(),
		Session: state.TerminalSessionStore{}.
			AttachWithResizeOwner("term-session-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID)).
			MarkExitedWithMetadata("term-session-exited", 130, "exited", time.Date(2026, 7, 5, 1, 1, 34, 0, time.UTC), []string{"/bin/zsh"}),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-session-exited", State: state.TerminalLiveAttached, Lines: []string{"terminal exited: term-session-exited code:130 exited"}},
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID, "term-session-exited", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true,
		)),
	}
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{})
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial render: %v", err)
	}
	if !frameContains(lastRuntimeFrame(t, host), "restart") {
		t.Fatalf("session-exited terminal should render restart CTA, frame=%#v", lastRuntimeFrame(t, host).Lines)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyEnter}); err != nil {
		t.Fatalf("send selected restart CTA: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain selected restart CTA: %v", err)
	}

	if len(terminal.Lists) == 0 || len(terminal.Restarts) != 1 || terminal.Restarts[0].TerminalID != "term-session-exited" {
		t.Fatalf("selected CTA should query core lifecycle then restart exited terminal, lists=%#v restarts=%#v", terminal.Lists, terminal.Restarts)
	}
	if len(terminal.Attaches) != 1 || terminal.Attaches[0].TerminalID != "term-session-exited" || terminal.Attaches[0].ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("restart should reattach the bound terminal view, attaches=%#v", terminal.Attaches)
	}
	final := runtime.State()
	if final.Session.State == state.TerminalLiveExited || final.Session.Channel != 11 || !final.Session.Attached {
		t.Fatalf("restart reattach should clear exited session and install new channel, session=%#v", final.Session)
	}
	if final.Surface.State == state.TerminalLiveExited || frameContains(lastRuntimeFrame(t, host), "restart") {
		t.Fatalf("restart should remove exited CTA while waiting for fresh live output, surface=%#v frame=%#v", final.Surface, lastRuntimeFrame(t, host).Lines)
	}
}

func TestAppRuntimeContentActionFocusesTargetPane(t *testing.T) {
	host := NewFakeTerminalHost(8)
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "empty", Kind: state.PaneEmpty}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	runtime := newShellHitRuntime(state.Root{Shell: shell}, host)
	if err := runtime.Post(NoopMsg{}); err != nil {
		t.Fatalf("post initial render: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("initial drain: %v", err)
	}

	attach := frameActionHitRegion(t, lastRuntimeFrame(t, host), render.ActionEmptyAttach.String(), "pane-2")
	if err := host.SendInput(mouseEventAt(attach.Rect)); err != nil {
		t.Fatalf("send empty attach click: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	shell = runtime.State().Shell.EnsureDefaults()
	if shell.ActivePaneID != "pane-2" {
		t.Fatalf("content action should focus clicked empty pane, got %q", shell.ActivePaneID)
	}
	if !shell.Overlay.Open || shell.Overlay.TargetID != "pane-2" {
		t.Fatalf("empty attach should target clicked pane, got %#v", shell.Overlay)
	}
}

func newShellHitRuntime(root state.Root, host *FakeTerminalHost) *AppRuntime {
	host.SetSize(80, 20)
	return NewAppRuntime(
		root,
		NewShellReducer(),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
}

func paneLayoutRects(root state.Root) map[string]render.Rect {
	vm := render.NewRenderVMBuilder().Build(root)
	plan := render.MeasureLayout(vm.Shell, vm.Shell.Layout.Viewport)
	rects := make(map[string]render.Rect, len(plan.Panels))
	for _, panel := range plan.Panels {
		rects[panel.Panel.ID] = panel.Rect
	}
	return rects
}

func newFourColumnPaneRuntime(t *testing.T) (*AppRuntime, *FakeTerminalHost, *testkit.FakeTerminalService) {
	t.Helper()
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 12, Cols: 100, Rows: 24},
	}
	host := NewFakeTerminalHost(64)
	host.SetSize(100, 24)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 100, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split-line: %v", err)
	}
	for _, paneID := range []string{"pane-2", "pane-3", "pane-4"} {
		if err := runtime.Post(ShellSplitActivePaneMsg{Pane: state.PaneState{ID: paneID, Title: paneID, Kind: state.PaneTerminalLive}, Direction: state.SplitDirectionVertical}); err != nil {
			t.Fatalf("post split %s: %v", paneID, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain four-column setup: %v", err)
	}
	return runtime, host, terminal
}

func stackedRightColumnShellForTest() state.ShellStore {
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	tab := &shell.Workspace.Tabs[0]
	tab.Panes = []state.PaneState{
		{ID: "left", Title: "left", Kind: state.PaneTerminalLive},
		{ID: "top", Title: "top", Kind: state.PaneTerminalLive},
		{ID: "middle-left", Title: "middle-left", Kind: state.PaneTerminalLive},
		{ID: "middle-right", Title: "middle-right", Kind: state.PaneTerminalLive},
		{ID: "bottom", Title: "bottom", Kind: state.PaneTerminalLive},
	}
	tab.ActivePaneID = "middle-left"
	tab.RootSplit = state.SplitNode{
		Direction: state.SplitDirectionVertical,
		Children: []state.SplitNode{
			{PaneID: "left"},
			{
				Direction: state.SplitDirectionHorizontal,
				Children: []state.SplitNode{
					{PaneID: "top"},
					{
						Direction: state.SplitDirectionHorizontal,
						Children: []state.SplitNode{
							{
								Direction: state.SplitDirectionVertical,
								Children:  []state.SplitNode{{PaneID: "middle-left"}, {PaneID: "middle-right"}},
							},
							{PaneID: "bottom"},
						},
					},
				},
			},
		},
	}
	shell.ActivePaneID = "middle-left"
	return shell.EnsureDefaults()
}

func newShellHitRuntimeWithTerminal(root state.Root, host *FakeTerminalHost, terminal port.TerminalService) *AppRuntime {
	host.SetSize(80, 20)
	return NewAppRuntime(
		root,
		ComposeReducers(NewShellReducer(), newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})),
		func(root state.Root) render.Frame {
			return render.NewRenderer(render.DefaultTheme()).Render(render.NewRenderVMBuilder().Build(root))
		},
		host,
		NewSyncEffectRunner(),
	)
}

func mouseEventAt(rect render.Rect) input.InputEvent {
	return input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft, Row: rect.Y + 1, Col: rect.X + 1}
}

func mouseEventAtRenderedTokenInRect(t *testing.T, frame render.Frame, rect render.Rect, token string) input.InputEvent {
	t.Helper()
	row := rect.Y
	if row < 0 || row >= len(frame.Lines) {
		t.Fatalf("row %d out of frame %#v", row, frame.Lines)
	}
	col := renderedTokenCellIndexInRange(frame.Lines[row], token, rect.X, rect.X+rect.W)
	if col < 0 {
		t.Fatalf("missing token %q inside rect=%#v row=%q", token, rect, frame.Lines[row])
	}
	return input.InputEvent{Kind: input.EventKindMouse, Mouse: input.MouseLeft, Row: row + 1, Col: col + 1}
}

func renderedTokenCellIndexInRange(line string, token string, left int, right int) int {
	width := render.DisplayWidth(token)
	if width <= 0 {
		return -1
	}
	maxCol := minIntForTest(right-width, render.DisplayWidth(line)-width)
	for col := maxIntForTest(0, left); col <= maxCol; col++ {
		if render.SliceCells(line, col, col+width) == token {
			return col
		}
	}
	return -1
}

func minIntForTest(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxIntForTest(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func pointInRenderRect(event input.InputEvent, rect render.Rect) bool {
	col := event.Col - 1
	row := event.Row - 1
	return col >= rect.X && col < rect.X+rect.W && row >= rect.Y && row < rect.Y+rect.H
}

func lastRuntimeFrame(t *testing.T, host *FakeTerminalHost) render.Frame {
	t.Helper()
	frames := host.Frames()
	if len(frames) == 0 {
		t.Fatal("expected at least one rendered frame")
	}
	return frames[len(frames)-1]
}

func lastRuntimeNewFrame(t *testing.T, host *FakeTerminalHost, before int) render.Frame {
	t.Helper()
	frames := host.Frames()
	if len(frames) <= before {
		t.Fatalf("expected new rendered frame after %d, got %d", before, len(frames))
	}
	return frames[len(frames)-1]
}

func frameHitRegion(t *testing.T, frame render.Frame, kind render.HitRegionKind, paneID string) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == kind && (paneID == "" || region.PaneID == paneID) {
			return region
		}
	}
	t.Fatalf("missing hit region kind=%s pane=%s in %#v", kind, paneID, frame.HitRegions)
	return render.HitRegion{}
}

func frameActionHitRegion(t *testing.T, frame render.Frame, actionID string, paneID string) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == render.HitRegionContentAction && region.ActionID == actionID && (paneID == "" || region.PaneID == paneID) {
			return region
		}
	}
	t.Fatalf("missing content action=%s pane=%s in %#v", actionID, paneID, frame.HitRegions)
	return render.HitRegion{}
}

func frameActionHitRegionRow(t *testing.T, frame render.Frame, actionID string, paneID string, row int) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == render.HitRegionContentAction && region.ActionID == actionID && region.Row == row && (paneID == "" || region.PaneID == paneID) {
			return region
		}
	}
	t.Fatalf("missing content action=%s pane=%s row=%d in %#v", actionID, paneID, row, frame.HitRegions)
	return render.HitRegion{}
}

func assertFrameMissingActionHitRegion(t *testing.T, frame render.Frame, actionID string) {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.ActionID == actionID {
			t.Fatalf("action %s should not have hit region in %#v", actionID, frame.HitRegions)
		}
	}
}

func frameHitRegionByAction(t *testing.T, frame render.Frame, kind render.HitRegionKind, actionID string, paneID string) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == kind && region.ActionID == actionID && (paneID == "" || region.PaneID == paneID) {
			return region
		}
	}
	t.Fatalf("missing hit region kind=%s action=%s pane=%s in %#v", kind, actionID, paneID, frame.HitRegions)
	return render.HitRegion{}
}

func framePaneResizeRegion(t *testing.T, frame render.Frame, paneID string, direction state.PaneResizeDirection) render.HitRegion {
	t.Helper()
	for _, region := range frame.HitRegions {
		if region.Kind == render.HitRegionPaneResize && region.ActionID == "pane.resize" && region.PaneID == paneID && region.Direction == string(direction) {
			return region
		}
	}
	t.Fatalf("missing pane resize region pane=%s direction=%s in %#v", paneID, direction, frame.HitRegions)
	return render.HitRegion{}
}

func viewportLabel(root state.Root) string {
	if !root.Viewport.Valid {
		return "unset"
	}
	return fmt.Sprintf("%dx%d", root.Viewport.Cols, root.Viewport.Rows)
}

func TestAppRuntimeQuitStopsQueue(t *testing.T) {
	var seen []string
	runtime := NewAppRuntime(
		state.Root{},
		func(root state.Root, msg Msg) (state.Root, []Effect) {
			seen = append(seen, msg.(testMsg).Name)
			return root.Advance(), nil
		},
		nil,
		nil,
		nil,
	)

	if err := runtime.Post(testMsg{Name: "before"}); err != nil {
		t.Fatalf("post before: %v", err)
	}
	if err := runtime.Post(QuitMsg{}); err != nil {
		t.Fatalf("post quit: %v", err)
	}
	if err := runtime.Post(testMsg{Name: "after"}); err != nil {
		t.Fatalf("post after: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !runtime.Quit() {
		t.Fatal("expected runtime quit")
	}
	if !reflect.DeepEqual(seen, []string{"before"}) {
		t.Fatalf("queue should stop at quit, got %v", seen)
	}
	if err := runtime.Post(testMsg{Name: "late"}); !errors.Is(err, ErrRuntimeStopped) {
		t.Fatalf("expected ErrRuntimeStopped, got %v", err)
	}
}

func TestFakeFrameSinkReturnsDetachedFrames(t *testing.T) {
	sink := &FakeFrameSink{}
	if err := sink.WriteFrame(render.Frame{Lines: []string{"one"}}); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	frames := sink.Frames()
	frames[0].Lines[0] = "mutated"

	got := sink.Frames()
	if got[0].Lines[0] != "one" {
		t.Fatalf("expected detached frames, got %v", got)
	}
}

func frameLines(frames []render.Frame) []string {
	lines := make([]string, len(frames))
	for i, frame := range frames {
		if len(frame.Lines) > 0 {
			lines[i] = frame.Lines[0]
		}
	}
	return lines
}
