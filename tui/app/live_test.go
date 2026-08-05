package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/testkit"
)

type refreshingInputTerminalService struct {
	testkit.FakeTerminalService
	nextChannel         uint16
	staleChannels       map[uint16]bool
	staleKnownOnAttach  bool
	knownActiveChannels map[uint16]bool
}

func (service *refreshingInputTerminalService) Attach(_ context.Context, req port.TerminalAttachRequest) (port.TerminalAttachResult, error) {
	service.Attaches = append(service.Attaches, req)
	if service.AttachErr != nil {
		return port.TerminalAttachResult{}, service.AttachErr
	}
	if service.staleChannels == nil {
		service.staleChannels = make(map[uint16]bool)
	}
	if service.knownActiveChannels == nil {
		service.knownActiveChannels = make(map[uint16]bool)
	}
	if service.staleKnownOnAttach {
		for channel := range service.knownActiveChannels {
			service.staleChannels[channel] = true
		}
	}
	channel := service.nextChannel
	if channel == 0 {
		channel = 1
	}
	service.nextChannel = channel + 1
	service.knownActiveChannels[channel] = true
	delete(service.staleChannels, channel)
	result := service.AttachResult
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = req.TerminalID
	}
	result.Channel = channel
	if result.Cols == 0 {
		result.Cols = req.Cols
	}
	if result.Rows == 0 {
		result.Rows = req.Rows
	}
	if result.ResizePolicy == "" {
		result.ResizePolicy = req.ResizePolicy
	}
	if result.SurfaceID == "" {
		result.SurfaceID = req.SurfaceID
	}
	if result.ViewID == "" {
		result.ViewID = req.ViewID
	}
	if result.Session == nil {
		result.Session = &apipb.EndpointSessionStamp{EndpointId: string(state.NormalizeEndpointID(req.EndpointID)), RouteId: "test", Generation: 1}
	}
	if result.OperationID == "" {
		result.OperationID = req.OperationID
	}
	if !result.SizeLocked && result.ControlReason == "" && result.ResizePolicy == state.TerminalResizeRoleOwner {
		result.CanResize = true
	}
	return result, nil
}

func (service *refreshingInputTerminalService) SendInput(_ context.Context, req port.TerminalInputRequest) error {
	service.Inputs = append(service.Inputs, req)
	if service.InputErr != nil {
		return service.InputErr
	}
	if service.staleChannels[req.Channel] {
		return fmt.Errorf("stale channel %d", req.Channel)
	}
	return nil
}

type blockingOrderedInputTerminalService struct {
	testkit.FakeTerminalService
	firstStarted chan struct{}
	releaseFirst chan struct{}
	done         chan struct{}
	mu           sync.Mutex
	inputs       []port.TerminalInputRequest
}

func stampedTestTerminalView(binding state.TerminalViewBinding) state.TerminalViewBinding {
	binding.Session = testEndpointSessionStamp(binding.EndpointID)
	return binding
}

func newBlockingOrderedInputTerminalService() *blockingOrderedInputTerminalService {
	return &blockingOrderedInputTerminalService{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		done:         make(chan struct{}),
	}
}

func (service *blockingOrderedInputTerminalService) SendInput(_ context.Context, req port.TerminalInputRequest) error {
	service.mu.Lock()
	isFirst := len(service.inputs) == 0
	service.inputs = append(service.inputs, req)
	service.mu.Unlock()
	if isFirst {
		close(service.firstStarted)
		<-service.releaseFirst
	}
	service.mu.Lock()
	count := len(service.inputs)
	service.mu.Unlock()
	if count >= 2 {
		select {
		case <-service.done:
		default:
			close(service.done)
		}
	}
	return nil
}

func (service *blockingOrderedInputTerminalService) inputText() string {
	service.mu.Lock()
	defer service.mu.Unlock()
	var builder strings.Builder
	for _, req := range service.inputs {
		builder.Write(req.Bytes)
	}
	return builder.String()
}

func (service *blockingOrderedInputTerminalService) inputCount() int {
	service.mu.Lock()
	defer service.mu.Unlock()
	return len(service.inputs)
}

func ownerLiveAttachConfig(terminalID string, cols int, rows int) LiveConfig {
	return ownerLiveAttachConfigForPane(terminalID, cols, rows, state.DefaultPaneID)
}

func ownerLiveAttachConfigForPane(terminalID string, cols int, rows int, paneID string) LiveConfig {
	return LiveConfig{
		TerminalID:   terminalID,
		Cols:         cols,
		Rows:         rows,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "test-surface",
		ViewID:       state.TerminalPaneViewID(paneID),
	}
}

func firstWorkbenchPersistEffect(t *testing.T, effects []Effect) WorkbenchStoragePersistRequestMsg {
	t.Helper()
	for _, effect := range effects {
		funcEffect, ok := effect.(FuncEffect)
		if !ok || funcEffect.Run == nil {
			continue
		}
		if msg, ok := funcEffect.Run(context.Background()).(WorkbenchStoragePersistRequestMsg); ok {
			return msg
		}
	}
	t.Fatalf("expected workbench persist effect, got %#v", effects)
	return WorkbenchStoragePersistRequestMsg{}
}

func hasTerminalPoolListEffect(effects []Effect) bool {
	for _, effect := range effects {
		funcEffect, ok := effect.(FuncEffect)
		if !ok || funcEffect.Run == nil {
			continue
		}
		if _, ok := funcEffect.Run(context.Background()).(TerminalPoolListRequestMsg); ok {
			return true
		}
	}
	return false
}

func TestLiveInputRoutesLSSequenceAcrossTwoTiledPaneBindings(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewPaneTerminalView("pane-2", "term-2", 8, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID("pane-2"), true))
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 30)
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	pane2Content := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionPaneContent, "pane-2")
	if err := host.SendInput(mouseEventAt(pane2Content.Rect)); err != nil {
		t.Fatalf("click pane-2: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send pane-2 input %#v: %v", event, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane-2 input: %v", err)
	}

	pane1Content := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionPaneContent, state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(pane1Content.Rect)); err != nil {
		t.Fatalf("click pane-1: %v", err)
	}
	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send pane-1 input %#v: %v", event, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane-1 input: %v", err)
	}

	if got := compactInputRequests(terminal.Inputs); len(got) != 2 ||
		got[0] != "term-2#8:ls\r" ||
		got[1] != "term-1#7:ls\r" {
		t.Fatalf("input must follow clicked pane binding, got %#v raw=%#v", got, terminal.Inputs)
	}
}

func TestTerminalInputRouterLogsActiveViewRoute(t *testing.T) {
	t.Setenv(tuiInputTraceEnv, "1")
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	terminal := &testkit.FakeTerminalService{}
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true))
	reducer := ComposeReducers(NewUIInputReducer(), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal, Logger: logger}), newLiveReducerPrepared(LiveDeps{Terminal: terminal, Logger: logger}))

	_, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l", RawSeq: "l"}})
	if len(effects) != 1 {
		t.Fatalf("ordinary key should produce terminal input effect, got %#v", effects)
	}
	effect, ok := effects[0].(FuncEffect)
	if !ok || !effect.Async || !effect.ForceSyncInTests {
		t.Fatalf("terminal input send must be async in real runtime and sync-capable in tests, got %#v", effects[0])
	}
	msg, ok := effect.Run(context.Background()).(LiveInputResultMsg)
	if !ok || msg.Err != nil {
		t.Fatalf("expected terminal input result, got %#v ok=%v", msg, ok)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || terminal.Inputs[0].Channel != 7 || string(terminal.Inputs[0].Bytes) != "l" {
		t.Fatalf("input must route through active view binding, got %#v", terminal.Inputs)
	}
	text := logs.String()
	for _, want := range []string{"tui-v3 input route", "tui-v3 terminal input sent", "result=terminal", "target_view=" + state.TerminalPaneViewID(state.DefaultPaneID), "terminal_id=term-1", "channel=7"} {
		if !strings.Contains(text, want) {
			t.Fatalf("input route log missing %q in:\n%s", want, text)
		}
	}
}

func TestBracketedPasteBypassesShortcutDispatchAsOneTerminalSemanticEvent(t *testing.T) {
	for _, tc := range []struct {
		name      string
		bracketed bool
		want      string
	}{
		{name: "plain downstream", want: "\x07danger\n"},
		{name: "bracketed downstream", bracketed: true, want: "\x1b[200~\x07danger\n\x1b[201~"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			terminal := &testkit.FakeTerminalService{}
			root := state.Root{Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")}
			root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView(
				"west", state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true,
			))
			root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
				EndpointID: "west", TerminalID: "term-1", State: state.TerminalLiveAttached,
				Modes: state.LiveTerminalModes{BracketedPaste: tc.bracketed},
			})
			reducer := ComposeReducers(NewUIInputReducer(), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}))

			next, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindPaste, Paste: "\x07danger\n"}})
			if next.Shell.EnsureDefaults().InteractionMode != state.InteractionModeNormal || next.Shell.EnsureDefaults().Overlay.Open {
				t.Fatalf("paste body control bytes must not enter shortcut scenes: %#v", next.Shell)
			}
			if len(effects) != 1 {
				t.Fatalf("paste must produce one terminal send effect, got %#v", effects)
			}
			msg, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveInputResultMsg)
			if !ok || msg.Err != nil {
				t.Fatalf("expected paste input result, got %#v ok=%v", msg, ok)
			}
			if len(terminal.Inputs) != 1 || terminal.Inputs[0].EndpointID != "west" || string(terminal.Inputs[0].Bytes) != tc.want {
				t.Fatalf("paste must use endpoint surface mode in one send, got %#v want=%q", terminal.Inputs, tc.want)
			}
		})
	}
}

func TestTerminalInputRouterRoutesEndpointBinding(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true))
	reducer := ComposeReducers(NewUIInputReducer(), NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}))

	_, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x", RawSeq: "x"}})
	if len(effects) != 1 {
		t.Fatalf("ordinary key should produce terminal input effect, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveInputResultMsg)
	if !ok || msg.Err != nil {
		t.Fatalf("expected terminal input result, got %#v ok=%v", msg, ok)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].EndpointID != "west" || terminal.Inputs[0].TerminalID != "term-1" || string(terminal.Inputs[0].Bytes) != "x" {
		t.Fatalf("input must route through endpoint-aware binding, got %#v", terminal.Inputs)
	}
}

func TestInteractiveRuntimeSerializesAsyncTerminalInputBytes(t *testing.T) {
	terminal := newBlockingOrderedInputTerminalService()
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
	}
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
	host := NewFakeTerminalHost(64)
	host.SetSize(100, 30)
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewAsyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	command := "printf abc"
	for _, ch := range command {
		if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: string(ch)}); err != nil {
			t.Fatalf("send input %q: %v", ch, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain input: %v", err)
	}
	select {
	case <-terminal.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first terminal input did not start")
	}
	// 中文说明：连续普通 key bytes 会在 runtime drain 边界合成一个 PTY
	// input RPC；SerialKey 仍保证同 terminal/view/channel 的后续输入不能越过它。
	if got := terminal.inputText(); got != command {
		t.Fatalf("plain key burst should send as one ordered input batch, got %q", got)
	}
	close(terminal.releaseFirst)
	time.Sleep(20 * time.Millisecond)
	if got := terminal.inputText(); got != command {
		t.Fatalf("terminal input order changed, got %q want %q", got, command)
	}
	if got := terminal.inputCount(); got != 1 {
		t.Fatalf("plain key burst should send exactly one batched request, got %d", got)
	}
}

func TestCopyModeBoundToSiblingPaneDoesNotConsumeActivePaneInput(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-shared").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-shared"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-2"})
	root := state.Root{
		Shell: shell,
		CopyMode: state.CopyModeStore{
			Active:     true,
			PaneID:     state.DefaultPaneID,
			ViewID:     state.TerminalPaneViewID(state.DefaultPaneID),
			TerminalID: "term-shared",
			BoundToken: "tok-copy",
			BoundCols:  80,
			ViewRows:   24,
		},
	}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-shared", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-1", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewPaneTerminalView("pane-2", "term-shared", 8, 80, 24, state.TerminalResizeRoleFollower, "surface-2", state.TerminalPaneViewID("pane-2"), false))
	reducer := ComposeReducers(
		NewUIInputReducer(),
		NewCopyModeReducer(CopyModeDeps{Core: &testkit.FakeCoreClient{}}),
		NewTerminalInputRouterReducer(LiveDeps{Terminal: terminal}),
	)

	_, effects := reducer(root, InputMsg{Event: input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l", RawSeq: "l"}})
	if len(effects) != 1 {
		t.Fatalf("active sibling pane key should reach terminal router, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(LiveInputResultMsg)
	if !ok || msg.Err != nil {
		t.Fatalf("expected terminal input result, got %#v ok=%v", msg, ok)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].Channel != 8 || terminal.Inputs[0].ViewID != state.TerminalPaneViewID("pane-2") || string(terminal.Inputs[0].Bytes) != "l" {
		t.Fatalf("copy mode must not consume active sibling view input, got %#v", terminal.Inputs)
	}
}

func TestLiveInputRoutesBetweenTiledAndFloatingSharedTerminalChannels(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-shared")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-shared"},
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	shell = shell.FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-shared", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-shared", 8, 30, 8, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("floating-1"), false))
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 30)
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	floatingContent := frameActionHitRegion(t, lastFrame(t, host.Frames()), render.ActionFloatingRaise.String(), "floating-pane-1")
	if err := host.SendInput(mouseEventAt(floatingContent.Rect)); err != nil {
		t.Fatalf("click floating: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"}); err != nil {
		t.Fatalf("send floating key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating input: %v", err)
	}

	paneContent := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionPaneContent, state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(paneContent.Rect)); err != nil {
		t.Fatalf("click pane: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"}); err != nil {
		t.Fatalf("send pane key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane input: %v", err)
	}

	if got := compactInputRequests(terminal.Inputs); len(got) != 2 ||
		got[0] != "term-shared#8:f" ||
		got[1] != "term-shared#7:p" {
		t.Fatalf("shared terminal input must use active view channel, got %#v raw=%#v", got, terminal.Inputs)
	}
}

func TestLiveInputRoutesBetweenTiledAndFloatingDifferentTerminals(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-pane")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-pane", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-float", 8, 30, 8, state.TerminalResizeRoleOwner, "surface", state.TerminalFloatingViewID("floating-1"), true))
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 30)
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	floatingContent := frameActionHitRegion(t, lastFrame(t, host.Frames()), render.ActionFloatingRaise.String(), "floating-pane-1")
	if err := host.SendInput(mouseEventAt(floatingContent.Rect)); err != nil {
		t.Fatalf("click floating: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"}); err != nil {
		t.Fatalf("send floating key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating input: %v", err)
	}

	paneContent := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionPaneContent, state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(paneContent.Rect)); err != nil {
		t.Fatalf("click pane: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "p"}); err != nil {
		t.Fatalf("send pane key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane input: %v", err)
	}

	if got := compactInputRequests(terminal.Inputs); len(got) != 2 ||
		got[0] != "term-float#8:f" ||
		got[1] != "term-pane#7:p" {
		t.Fatalf("different terminal input must follow active view binding, got %#v raw=%#v", got, terminal.Inputs)
	}
}

func TestTerminalPoolReattachCurrentPaneDoesNotOverwriteSiblingBinding(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      21,
			Cols:         80,
			Rows:         24,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface",
		},
	}
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(stampedTestTerminalView(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))).
		BindPane(stampedTestTerminalView(state.NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false)))
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})

	root, effects := reducer(root, TerminalPoolAttachRequestMsg{TerminalID: "term-1", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("expected attach effect, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolAttachResultMsg)
	if !ok {
		t.Fatalf("expected attach result, got %#v", msg)
	}
	root, _ = reducer(root, msg)

	pane1, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || pane1.Channel != 21 || pane1.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("current pane should receive fresh channel, binding=%#v ok=%v", pane1, ok)
	}
	pane2, ok := root.TerminalViews.PaneBinding("pane-2")
	if !ok || pane2.Channel != 8 || pane2.ViewID != state.TerminalPaneViewID("pane-2") {
		t.Fatalf("reattach must not overwrite sibling channel, binding=%#v ok=%v", pane2, ok)
	}
}

func TestTerminalPoolAttachRoutesEndpointRef(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      31,
			Cols:         100,
			Rows:         30,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-west",
		},
	}
	root := state.Root{Shell: state.DefaultShell()}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})

	root, effects := reducer(root, TerminalPoolAttachRequestMsg{EndpointID: "west", TerminalID: "term-1", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("expected attach effect, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolAttachResultMsg)
	if !ok {
		t.Fatalf("expected attach result, got %#v", msg)
	}
	if len(terminal.Attaches) != 1 || terminal.Attaches[0].EndpointID != "west" {
		t.Fatalf("attach request must carry endpoint, got %#v", terminal.Attaches)
	}

	root, _ = reducer(root, msg)
	binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || binding.EndpointID != "west" || binding.TerminalID != "term-1" {
		t.Fatalf("pane binding must keep endpoint ref, binding=%#v ok=%v", binding, ok)
	}
	if !root.Session.TerminalRef().Equal(state.NewTerminalRef("west", "term-1")) {
		t.Fatalf("session must attach west ref, got %#v", root.Session.TerminalRef())
	}
	if !root.Surface.TerminalRef().Equal(state.NewTerminalRef("west", "term-1")) {
		t.Fatalf("surface must attach west ref, got %#v", root.Surface.TerminalRef())
	}
	if !root.TerminalPool.LastAttachedRef.Equal(state.NewTerminalRef("west", "term-1")) {
		t.Fatalf("pool attach projection must keep west ref, got %#v", root.TerminalPool.LastAttachedRef)
	}
}

func TestTerminalInputSerialKeyIsEndpointScoped(t *testing.T) {
	local := liveInputTargetInfo{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", ViewID: "view-1", Channel: 7}
	west := liveInputTargetInfo{EndpointID: "west", TerminalID: "term-1", ViewID: "view-1", Channel: 7}
	if terminalInputSerialKey(local) == terminalInputSerialKey(west) {
		t.Fatalf("input serial key must include endpoint, got %q", terminalInputSerialKey(local))
	}
}

func TestTerminalPoolAttachExitedTerminalDoesNotCacheLifecycleBeforeSurface(t *testing.T) {
	exitedAt := time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC)
	exitCode := 23
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-exited",
			Channel:      21,
			Cols:         80,
			Rows:         24,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface",
		},
	}
	root := state.Root{
		Shell: state.DefaultShell().
			BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-old"),
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			TerminalID: "term-exited",
			Title:      "done",
			State:      string(state.TerminalLiveExited),
			ExitCode:   &exitCode,
			ExitedAt:   exitedAt,
			Command:    []string{"bash", "-lc", "exit 23"},
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-old", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})

	next, effects := reducer(root, TerminalPoolAttachRequestMsg{TerminalID: "term-exited", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("expected attach effect, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolAttachResultMsg)
	if !ok {
		t.Fatalf("expected attach result, got %#v", msg)
	}
	next, _ = reducer(next, msg)

	surface := next.Surface.SurfaceForTerminal("term-exited")
	if surface.State == state.TerminalLiveExited || surface.ExitCode != 0 || !surface.ExitedAt.IsZero() || len(surface.Command) != 0 {
		t.Fatalf("picker attach must not copy pool lifecycle into live surface before core surface query, surface=%#v exited_at=%s command=%s", surface, exitedAt, strings.Join([]string{"bash", "-lc", "exit 23"}, " "))
	}
	if next.Session.TerminalID != "term-exited" || !next.Session.Attached || next.Session.Channel != 21 {
		t.Fatalf("attach should still bind the selected terminal view/session, session=%#v", next.Session)
	}
}

func TestLiveEffectsPreserveLifecycleBoundaryOnSurfaceMessage(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Cols:       80,
				Rows:       24,
				State:      state.TerminalLiveAttached,
			},
			LifecycleKnown: true,
		},
	}
	effects := liveEffects("term-1", 80, 24, LiveDeps{Terminal: terminal})
	var surface LiveSurfaceMsg
	found := false
	for _, effect := range effects {
		funcEffect, ok := effect.(FuncEffect)
		if !ok {
			continue
		}
		msg := funcEffect.Run(context.Background())
		candidate, ok := msg.(LiveSurfaceMsg)
		if !ok {
			continue
		}
		surface = candidate
		found = true
		break
	}
	if !found {
		t.Fatalf("expected live surface msg in effects, got %#v", effects)
	}
	if !surface.LifecycleKnown || surface.Snapshot.TerminalID != "term-1" || surface.Snapshot.State != state.TerminalLiveAttached {
		t.Fatalf("live surface effect must preserve lifecycle message boundary, got %#v", surface)
	}
}

func TestInteractionModeContentClickThenKeyUsesTerminalInputRoute(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	root := state.Root{
		Shell: state.DefaultShell().
			BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
			SetInteractionMode(state.InteractionModeResize),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 30)
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}
	content := frameHitRegion(t, lastFrame(t, host.Frames()), render.HitRegionPaneContent, state.DefaultPaneID)
	if err := host.SendInput(mouseEventAt(content.Rect)); err != nil {
		t.Fatalf("click pane content: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}); err != nil {
		t.Fatalf("send key after content activation: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain input: %v", err)
	}

	if runtime.State().Shell.EnsureDefaults().InteractionMode != state.InteractionModeNormal {
		t.Fatalf("content click should exit interaction mode, shell=%#v", runtime.State().Shell)
	}
	if got := compactInputRequests(terminal.Inputs); len(got) != 1 || got[0] != "term-1#7:l" {
		t.Fatalf("key after content click should reach terminal service, got %#v raw=%#v", got, terminal.Inputs)
	}
}

func TestLiveAppAttachRenderInputAndResize(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID: "term-1",
			Channel:    9,
			Cols:       78,
			Rows:       22,
			CanResize:  true,
		},
	}
	host := NewFakeTerminalHost(8)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       80,
		Rows:       24,
		Lines:      []string{"$ echo hi", "hi"},
	}}); err != nil {
		t.Fatalf("post surface: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send input: %v", err)
	}
	if err := runtime.Post(LiveResizeMsg{Cols: 100, Rows: 40}); err != nil {
		t.Fatalf("post resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Attaches) != 1 || terminal.Attaches[0].TerminalID != "term-1" {
		t.Fatalf("unexpected attach requests %#v", terminal.Attaches)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "x" || terminal.Inputs[0].Channel != 9 {
		t.Fatalf("unexpected input requests %#v", terminal.Inputs)
	}
	if len(terminal.Resizes) != 1 || terminal.Resizes[0].Cols != 100 || terminal.Resizes[0].Rows != 40 || terminal.Resizes[0].Channel != 9 {
		t.Fatalf("manual resize must win over stale attach correction, got %#v", terminal.Resizes)
	}
	if runtime.State().Session.Cols != 100 || runtime.State().Surface.Cols != 100 {
		t.Fatalf("resize was not reflected in state %#v", runtime.State())
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.DesiredCols != 100 || binding.DesiredRows != 40 {
		t.Fatalf("manual resize should sync active owner binding desired size, got %#v", binding)
	}
	last := lastFrame(t, host.Frames())
	if len(last.Lines) == 0 || !frameContains(last, "$ echo hi") {
		t.Fatalf("expected live surface frame, got %#v", last.Lines)
	}
}

func TestLiveAttachMarksViewPendingBeforeResult(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell(), RuntimeSurfaceID: "runtime-a"}

	next, effects := reducer(root, LiveAttachMsg{Config: LiveConfig{
		TerminalID:   "term-1",
		Cols:         80,
		Rows:         24,
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    "runtime-a",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
	}})
	if len(effects) != 1 {
		t.Fatalf("expected attach effect, got %#v", effects)
	}
	binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || !binding.AttachPending || binding.Attached || binding.Channel != 0 {
		t.Fatalf("attach request should claim pending pane binding before result, binding=%#v ok=%v", binding, ok)
	}
	if binding.TerminalID != "term-1" || binding.SurfaceID != "runtime-a" || binding.DesiredCols != 78 || binding.DesiredRows != 20 {
		t.Fatalf("pending binding should keep attach request identity, got %#v", binding)
	}
}

func TestLiveAttachResultRefreshesTerminalPoolProjection(t *testing.T) {
	root := state.Root{Shell: state.DefaultShell()}
	next, effects := reduceLiveAttachResultPrepared(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      7,
		Cols:         80,
		Rows:         24,
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    "surface",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
	}}, LiveDeps{})

	if _, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); !ok {
		t.Fatalf("attach result should bind active pane, root=%#v", next)
	}
	if !hasTerminalPoolListEffect(effects) {
		t.Fatalf("attach result should refresh terminal pool projection for xN, effects=%#v", effects)
	}
}

func TestLiveAttachResultForBackgroundViewDoesNotReplaceActiveProjection(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main").
		SplitActivePane(state.PaneState{ID: "pane-remote", Title: "remote", Kind: state.PaneTerminalLive, TerminalID: "remote"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{}.
			AttachRefWithResizeOwner(state.LocalTerminalRef("term-main"), 7, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID)),
	}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: state.DefaultEndpointID,
		TerminalID: "term-main",
		Revision:   10,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"main"},
		State:      state.TerminalLiveAttached,
	})
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewEndpointPaneTerminalView("west", "pane-remote", "remote", 0, 100, 30, state.TerminalResizeRoleFollower, "surface-remote", state.TerminalPaneViewID("pane-remote"), false))

	next, _ := reducer(root, LiveAttachResultMsg{EndpointID: "west", TerminalID: "remote", ViewID: state.TerminalPaneViewID("pane-remote"), Result: port.TerminalAttachResult{
		EndpointID:    "west",
		TerminalID:    "remote",
		Channel:       9,
		Cols:          100,
		Rows:          30,
		ResizePolicy:  state.TerminalResizeRoleOwner,
		SurfaceID:     "surface-remote",
		ViewID:        state.TerminalPaneViewID("pane-remote"),
		CanResize:     false,
		SizeLocked:    true,
		ControlReason: "size_locked",
	}})
	if !next.Session.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Session.Channel != 7 {
		t.Fatalf("background attach must not replace active session, session=%#v", next.Session)
	}
	if !next.Surface.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Surface.Lines[0] != "main" {
		t.Fatalf("background attach must not replace active surface, surface=%#v", next.Surface)
	}
	if channel, ok := next.Session.InputChannelForRef(state.NewTerminalRef("west", "remote")); !ok || channel != 9 {
		t.Fatalf("background attach should record endpoint-scoped channel, channel=%d ok=%v session=%#v", channel, ok, next.Session)
	}
	if binding, ok := next.TerminalViews.PaneBinding("pane-remote"); !ok || binding.Channel != 9 || !binding.TerminalRef().Equal(state.NewTerminalRef("west", "remote")) {
		t.Fatalf("background view binding should receive attach result, binding=%#v ok=%v", binding, ok)
	}

	next, _ = reducer(next, LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		EndpointID: "west",
		TerminalID: "remote",
		Revision:   11,
		Cols:       100,
		Rows:       30,
		Lines:      []string{"remote"},
		State:      state.TerminalLiveAttached,
	}})
	if !next.Surface.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Surface.Lines[0] != "main" {
		t.Fatalf("background surface refresh must not replace active surface, surface=%#v", next.Surface)
	}
	if remote := next.Surface.SurfaceForTerminalRef(state.NewTerminalRef("west", "remote")); remote.TerminalID != "remote" || len(remote.Lines) != 1 || remote.Lines[0] != "remote" {
		t.Fatalf("background surface should be cached for its binding, remote=%#v", remote)
	}
}

func TestBackgroundLiveAttachErrorDoesNotPoisonActiveProjection(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	remoteRef := state.NewTerminalRef("west", "remote")
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main").
		SplitActivePane(state.PaneState{ID: "pane-remote", Title: "remote", Kind: state.PaneTerminalLive, TerminalID: "remote"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{}.
			AttachRefWithResizeOwner(state.LocalTerminalRef("term-main"), 7, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID)),
	}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: state.DefaultEndpointID,
		TerminalID: "term-main",
		Revision:   10,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"main"},
		State:      state.TerminalLiveAttached,
	})
	remoteBinding := state.NewEndpointPaneTerminalView("west", "pane-remote", "remote", 0, 100, 30, state.TerminalResizeRoleFollower, "surface-remote", state.TerminalPaneViewID("pane-remote"), false)
	remoteBinding.AttachPending = true
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(remoteBinding)

	next, _ := reducer(root, LiveAttachResultMsg{EndpointID: "west", TerminalID: "remote", ViewID: state.TerminalPaneViewID("pane-remote"), Err: errors.New("ssh endpoint \"west\" hello: EOF")})
	if !next.Session.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Session.State != state.TerminalLiveAttached || next.Session.Channel != 7 {
		t.Fatalf("background attach error must not replace active session, session=%#v", next.Session)
	}
	if !next.Surface.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Surface.State != state.TerminalLiveAttached || next.Surface.Err != "" || next.Surface.Lines[0] != "main" {
		t.Fatalf("background attach error must not replace active surface, surface=%#v", next.Surface)
	}
	if remote := next.Surface.SurfaceForTerminalRef(remoteRef); remote.State != state.TerminalLiveError || remote.Err == "" {
		t.Fatalf("background attach error should be cached on remote ref only, remote=%#v", remote)
	}
	if binding, ok := next.TerminalViews.PaneBinding("pane-remote"); !ok || binding.AttachPending || binding.LastError == "" {
		t.Fatalf("remote binding should keep scoped attach error, binding=%#v ok=%v", binding, ok)
	}
}

func TestBackgroundLiveSurfaceErrorDoesNotPoisonActiveProjection(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	remoteRef := state.NewTerminalRef("west", "remote")
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main").
		SplitActivePane(state.PaneState{ID: "pane-remote", Title: "remote", Kind: state.PaneTerminalLive, TerminalID: "remote"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{
		Shell: shell,
		Session: state.TerminalSessionStore{}.
			AttachRefWithResizeOwner(state.LocalTerminalRef("term-main"), 7, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID)),
	}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: state.DefaultEndpointID,
		TerminalID: "term-main",
		Revision:   10,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"main"},
		State:      state.TerminalLiveAttached,
	})
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-main", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewEndpointPaneTerminalView("west", "pane-remote", "remote", 0, 100, 30, state.TerminalResizeRoleFollower, "surface-remote", state.TerminalPaneViewID("pane-remote"), false))

	next, _ := reducer(root, LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{EndpointID: "west", TerminalID: "remote"}, Err: errors.New("ssh endpoint \"west\" hello: EOF")})
	if !next.Session.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Session.State != state.TerminalLiveAttached || next.Session.Channel != 7 {
		t.Fatalf("background surface error must not replace active session, session=%#v", next.Session)
	}
	if !next.Surface.TerminalRef().Equal(state.LocalTerminalRef("term-main")) || next.Surface.State != state.TerminalLiveAttached || next.Surface.Err != "" || next.Surface.Lines[0] != "main" {
		t.Fatalf("background surface error must not replace active surface, surface=%#v", next.Surface)
	}
	if remote := next.Surface.SurfaceForTerminalRef(remoteRef); remote.State != state.TerminalLiveError || remote.Err == "" {
		t.Fatalf("background surface error should be cached on remote ref only, remote=%#v", remote)
	}
}

func TestLiveAttachmentStoreSupportsSameTerminalAcrossTwoPanes(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := state.Root{Shell: state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneEmpty}, state.SplitDirectionVertical)}

	var effects []Effect
	root, effects = reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8, Cols: 40, Rows: 12, ResizePolicy: state.TerminalResizeRoleFollower, SurfaceID: "surface", ViewID: state.TerminalPaneViewID("pane-2")}})
	if msg := firstWorkbenchPersistEffect(t, effects); msg.Reason != "terminal.attach" {
		t.Fatalf("expected terminal attach persist request, got %#v", msg)
	}
	root.Shell = root.Shell.FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root, _ = reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-1", Channel: 7, Cols: 80, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "surface", ViewID: state.TerminalPaneViewID(state.DefaultPaneID), CanResize: true}})

	bindings := root.TerminalViews.BindingsForTerminal("term-1")
	if len(bindings) != 2 {
		t.Fatalf("expected two bindings for shared terminal, got %#v", bindings)
	}
	if target, ok := liveInputTarget(root); !ok || target.Channel != 7 || target.TerminalID != "term-1" {
		t.Fatalf("active pane should use its own attachment channel, target=%#v ok=%v", target, ok)
	}
	root.Shell = root.Shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"})
	if target, ok := liveInputTarget(root); !ok || target.Channel != 8 || target.TerminalID != "term-1" {
		t.Fatalf("sibling pane should use its own attachment channel, target=%#v ok=%v", target, ok)
	}

	root, _ = reducePaneCommand(root, state.PaneCommand{Action: state.PaneCommandClose, Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID}, Source: state.PaneCommandSourceTest})
	if _, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); ok {
		t.Fatal("close pane should detach only that view")
	}
	if _, ok := root.TerminalViews.PaneBinding("pane-2"); !ok {
		t.Fatal("close pane should not detach sibling view for same terminal")
	}
	root, _ = reduceTerminalPoolRemoveResult(root, TerminalPoolRemoveResultMsg{TerminalID: "term-1"})
	if bindings := root.TerminalViews.BindingsForTerminal("term-1"); len(bindings) != 0 {
		t.Fatalf("remove terminal should clear all view bindings, got %#v", bindings)
	}
	if pane, ok := root.Shell.Pane(state.PaneCommandTarget{PaneID: "pane-2"}); !ok || pane.TerminalID != "" || pane.Kind != state.PaneEmpty {
		t.Fatalf("remove terminal should clear pane terminal binding, pane=%#v ok=%v", pane, ok)
	}
	if root.Session.TerminalID == "term-1" || root.Surface.TerminalID == "term-1" {
		t.Fatalf("remove terminal should clear active session and surface, session=%#v surface=%#v", root.Session, root.Surface)
	}
}

func TestTerminalPoolKillPreservesAttachedPaneAndFloatingViews(t *testing.T) {
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-1"},
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root := state.Root{
		Shell:   shell,
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       80,
			Rows:       24,
			Lines:      []string{"live"},
		},
	}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-1", 8, 30, 8, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("floating-1"), false))

	next, effects := reduceTerminalPoolKillResult(root, TerminalPoolKillResultMsg{TerminalID: "term-1"})
	if !hasTerminalPoolListEffect(effects) {
		t.Fatalf("kill should refresh terminal inventory, effects=%#v", effects)
	}
	if bindings := next.TerminalViews.BindingsForTerminal("term-1"); len(bindings) != 2 {
		t.Fatalf("kill terminal should preserve attached views, got %#v", bindings)
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != "term-1" || pane.Kind != state.PaneTerminalLive {
		t.Fatalf("kill terminal should keep pane attached to terminal, pane=%#v ok=%v", pane, ok)
	}
	if floating, ok := next.Shell.FloatingByID("floating-1"); !ok || floating.Pane.TerminalID != "term-1" || floating.Pane.Kind != state.PaneTerminalLive {
		t.Fatalf("kill terminal should keep floating attached to terminal, floating=%#v ok=%v", floating, ok)
	}
	if next.Session.TerminalID != "term-1" || next.Surface.TerminalID != "term-1" {
		t.Fatalf("kill terminal should not clear active session or surface, session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestLiveInputAttachesActiveViewWhenChannelMissing(t *testing.T) {
	terminal := &refreshingInputTerminalService{nextChannel: 21}
	host := NewFakeTerminalHost(8)
	host.SetSize(100, 30)
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 0, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID(state.DefaultPaneID), false))
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}); err != nil {
		t.Fatalf("send key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Attaches) != 1 {
		t.Fatalf("missing channel should attach target view once, got %#v", terminal.Attaches)
	}
	if got := terminal.Attaches[0]; got.TerminalID != "term-1" || got.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || got.ResizePolicy != state.TerminalResizeRoleFollower {
		t.Fatalf("attach should target active pane view, got %#v", got)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].Channel != 21 || string(terminal.Inputs[0].Bytes) != "l" {
		t.Fatalf("input should replay through fresh channel, got %#v", terminal.Inputs)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != 21 {
		t.Fatalf("fresh channel should be stored on active view, binding=%#v ok=%v", binding, ok)
	}
}

func TestLiveInputDoesNotReattachWhileViewAttachPending(t *testing.T) {
	terminal := &refreshingInputTerminalService{nextChannel: 21}
	host := NewFakeTerminalHost(8)
	host.SetSize(100, 30)
	root := state.Root{Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1")}
	root.TerminalViews, _ = root.TerminalViews.BeginAttach(state.TerminalViewBinding{
		ViewID:      state.TerminalPaneViewID(state.DefaultPaneID),
		SurfaceID:   "surface",
		TerminalID:  "term-1",
		ResizeRole:  state.TerminalResizeRoleFollower,
		DesiredCols: 80,
		DesiredRows: 24,
		PaneID:      state.DefaultPaneID,
	})
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}); err != nil {
		t.Fatalf("send key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Attaches) != 0 || len(terminal.Inputs) != 0 {
		t.Fatalf("pending attach should consume input without issuing another attach/input, attaches=%#v inputs=%#v", terminal.Attaches, terminal.Inputs)
	}
	binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || !binding.AttachPending || binding.Channel != 0 {
		t.Fatalf("pending binding should remain waiting for original attach result, binding=%#v ok=%v", binding, ok)
	}
}

func TestLiveInputFailureDoesNotReplayAcrossReattach(t *testing.T) {
	terminal := &refreshingInputTerminalService{
		nextChannel:         21,
		staleChannels:       map[uint16]bool{7: true},
		staleKnownOnAttach:  true,
		knownActiveChannels: map[uint16]bool{7: true, 8: true},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(120, 36)
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(stampedTestTerminalView(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))).
		BindPane(stampedTestTerminalView(state.NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false)))
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}); err != nil {
		t.Fatalf("send first pane key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first pane: %v", err)
	}

	if len(terminal.Inputs) != 1 || terminal.Inputs[0].Channel != 7 {
		t.Fatalf("failed input must be attempted exactly once, got %#v", terminal.Inputs)
	}
	if len(terminal.Attaches) != 0 {
		t.Fatalf("failed input must not trigger implicit reattach, got %#v", terminal.Attaches)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != 0 || binding.Attached {
		t.Fatalf("failed input may invalidate the committed channel but must not replay payload, binding=%#v ok=%v", binding, ok)
	}

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{Action: state.PaneCommandFocus, Target: state.PaneCommandTarget{PaneID: "pane-2"}, Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("focus pane-2: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "s"}); err != nil {
		t.Fatalf("send second pane key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain second pane: %v", err)
	}

	if len(terminal.Inputs) != 2 || terminal.Inputs[1].Channel != 8 {
		t.Fatalf("sibling input must also be attempted once without replay, got %#v", terminal.Inputs)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding("pane-2"); !ok || binding.Channel != 8 || !binding.Attached {
		t.Fatalf("successful sibling input must preserve its committed channel, binding=%#v ok=%v", binding, ok)
	}
}

func TestLiveInputProtocolFailureDoesNotReattachOrReplay(t *testing.T) {
	terminal := &refreshingInputTerminalService{
		nextChannel:         21,
		staleChannels:       map[uint16]bool{7: true},
		knownActiveChannels: map[uint16]bool{7: true, 8: true},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(120, 36)
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(stampedTestTerminalView(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))).
		BindPane(stampedTestTerminalView(state.NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false)))
	runtime := NewInteractiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}); err != nil {
		t.Fatalf("send first pane key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain first pane: %v", err)
	}

	if len(terminal.Attaches) != 0 {
		t.Fatalf("failed active view input must not reattach implicitly, got %#v", terminal.Attaches)
	}
	if got := compactInputRequests(terminal.Inputs); len(got) != 1 || got[0] != "term-1#7:l" {
		t.Fatalf("input must be attempted exactly once, got %#v raw=%#v", got, terminal.Inputs)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.Channel != 0 || binding.Attached {
		t.Fatalf("active pane failed input may invalidate its channel without replay, binding=%#v ok=%v", binding, ok)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding("pane-2"); !ok || binding.Channel != 8 {
		t.Fatalf("sibling channel must not be overwritten, binding=%#v ok=%v", binding, ok)
	}
}

func TestLiveAttachResultAcceptsPrefilledSessionBeforeFirstBinding(t *testing.T) {
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Cols: 100, Rows: 30},
		Surface: state.TerminalSurfaceStore{TerminalID: "term-1", Cols: 100, Rows: 30},
	}
	root, effects := reduceLiveAttachResultPrepared(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      7,
		Cols:         100,
		Rows:         30,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "cmd/anytty-v3",
		ViewID:       "cmd/anytty-v3-main",
		CanResize:    true,
	}}, LiveDeps{})

	if msg := firstWorkbenchPersistEffect(t, effects); msg.Reason != "terminal.attach" {
		t.Fatalf("expected terminal attach persist request, got %#v", msg)
	}
	if !root.Session.Attached || root.Session.Channel != 7 || root.Session.ViewID != "cmd/anytty-v3-main" {
		t.Fatalf("prefilled CLI session should accept first attach result, session=%#v", root.Session)
	}
	if binding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.ViewID != "cmd/anytty-v3-main" {
		t.Fatalf("expected first attach result to create active pane binding, binding=%#v ok=%v", binding, ok)
	}
}

func TestLiveAttachResultDoesNotAutoRestartExitedTerminal(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := state.Root{
		Shell:   state.DefaultShell(),
		Surface: state.TerminalSurfaceStore{TerminalID: "term-1", State: state.TerminalLiveExited},
	}

	root, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{
		TerminalID:   "term-1",
		Channel:      7,
		Cols:         80,
		Rows:         24,
		ResizePolicy: state.TerminalResizeRoleOwner,
		SurfaceID:    "cmd/anytty-v3",
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
		CanResize:    true,
	}})

	if _, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID); !ok {
		t.Fatalf("attach result should only bind the active pane, root=%#v", root)
	}
	for _, effect := range effects {
		funcEffect, ok := effect.(FuncEffect)
		if !ok || funcEffect.Run == nil {
			continue
		}
		if msg, ok := funcEffect.Run(context.Background()).(TerminalPoolRestartIfExitedRequestMsg); ok {
			t.Fatalf("attach result must not auto restart or query restart, got %#v", msg)
		}
	}
}

func TestLiveAttachAndInitialSurfaceEffectsAreAsync(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	root := state.Root{Shell: state.DefaultShell()}
	_, effects := reduceLiveAttach(root, LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}, LiveDeps{Terminal: terminal})
	if len(effects) != 1 {
		t.Fatalf("expected one attach effect, got %#v", effects)
	}
	attach, ok := effects[0].(FuncEffect)
	if !ok || !attach.Async || !attach.ForceSyncInTests {
		t.Fatalf("attach must be async in real runtime and sync-capable in harness, got %#v", effects[0])
	}

	effects = liveSurfaceEffect("term-1", 80, 24, true, LiveDeps{Terminal: terminal})
	if len(effects) != 1 {
		t.Fatalf("expected one live surface effect, got %#v", effects)
	}
	surface, ok := effects[0].(FuncEffect)
	if !ok || !surface.Async || !surface.ForceSyncInTests {
		t.Fatalf("initial live surface fetch must be async in real runtime and sync-capable in harness, got %#v", effects[0])
	}
}

func TestTerminalPoolRemoveDeletesInventoryAndBindings(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Shell:        state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		Session:      state.TerminalSessionStore{TerminalID: "term-1", Channel: 7, Attached: true, InputChannels: map[string]uint16{"term-1": 7}},
		Surface:      state.TerminalSurfaceStore{TerminalID: "term-1", Ready: true, Lines: []string{"live"}, Surfaces: map[string]state.LiveSurfaceSnapshot{"term-1": {TerminalID: "term-1"}}},
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "one"}, {TerminalID: "term-2", Title: "two"}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))

	next, effects := reducer(root, TerminalPoolRemoveRequestMsg{TerminalID: "term-1"})
	if len(effects) != 1 {
		t.Fatalf("expected remove effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	if len(terminal.Removes) != 1 || terminal.Removes[0].TerminalID != "term-1" {
		t.Fatalf("remove request should call terminal service remove, got %#v", terminal.Removes)
	}
	next, _ = reducer(next, msg)
	if next.TerminalPool.LastRemovedID != "term-1" || len(next.TerminalPool.Items) != 1 || next.TerminalPool.Items[0].TerminalID != "term-2" {
		t.Fatalf("remove result should update pool inventory, pool=%#v", next.TerminalPool)
	}
	if _, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID); ok {
		t.Fatal("remove result should clear terminal view binding")
	}
	if pane, ok := next.Shell.Pane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}); !ok || pane.TerminalID != "" || pane.Kind != state.PaneEmpty {
		t.Fatalf("remove result should clear shell pane binding, pane=%#v ok=%v", pane, ok)
	}
	if next.Session.TerminalID != "" || next.Surface.TerminalID != "" {
		t.Fatalf("remove result should clear active session and surface, session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestLiveScreenNextEffectIsOneShot(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 2)}
	request := state.LiveScreenRequestState{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Demand: true, RequestInFlight: true, Generation: 3, SubmittedRevision: 7, Cols: 80, Rows: 24}
	effects := liveScreenNextEffectForRef(request, LiveDeps{Terminal: terminal})
	if len(effects) != 1 {
		t.Fatalf("expected one next-screen effect, got %#v", effects)
	}
	next, ok := effects[0].(FuncEffect)
	if !ok || !next.Async || next.ForceSyncInTests || next.Token != liveScreenNextTokenForRef(state.LocalTerminalRef("term-1")) {
		t.Fatalf("expected async terminal-scoped one-shot request, got %#v", effects[0])
	}
	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 8}}
	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 9}}

	msg := next.Run(context.Background())
	result, ok := msg.(LiveScreenNextResultMsg)
	if !ok || result.Snapshot.Revision != 8 || result.Generation != 3 {
		t.Fatalf("expected one next-screen result, got %#v", msg)
	}
	requests := terminal.LiveScreenNextRequestsSnapshot()
	if len(requests) != 1 || requests[0].TerminalID != "term-1" || requests[0].ObservedRevision != 7 {
		t.Fatalf("expected one-shot request with submitted revision, got %#v", requests)
	}
}

func TestLiveScreenNextEffectRequestsFullBootstrapAfterMergeFailure(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	request := state.LiveScreenRequestState{
		EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Demand: true,
		RequestInFlight: true, NeedsBootstrap: true, Generation: 3,
		SubmittedRevision: 7, Cols: 80, Rows: 24,
	}
	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 9, FullReplace: true}}
	effect := liveScreenNextEffectForRef(request, LiveDeps{Terminal: terminal})[0].(FuncEffect)
	if msg := effect.Run(context.Background()); msg == nil {
		t.Fatal("bootstrap request should return its result")
	}
	requests := terminal.LiveScreenNextRequestsSnapshot()
	if len(requests) != 1 || requests[0].ObservedRevision != 0 {
		t.Fatalf("bootstrap request must use observed revision zero, got %#v", requests)
	}
}

func TestLiveScreenDeltaMergeFailureRequiresBootstrap(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	ref := state.LocalTerminalRef("term-1")
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-1", Revision: 7, FullReplace: true, Cols: 80, Rows: 24,
		Screen: make([][]state.LiveCell, 24),
	})
	surface, _ = surface.ReconcileLiveScreenDemand([]state.TerminalRef{ref})
	surface = surface.SubmitLiveScreenRef(ref, 7, 80, 24)
	var request state.LiveScreenRequestState
	var start bool
	surface, request, start = surface.BeginLiveScreenRequestRef(ref)
	if !start {
		t.Fatal("test setup should start live request")
	}
	root, effects := reducer(state.Root{Surface: surface}, LiveScreenNextResultMsg{
		TerminalID: "term-1", Generation: request.Generation,
		Snapshot: state.LiveSurfaceSnapshot{
			TerminalID: "term-1", BaseRevision: 6, Revision: 8, Cols: 80, Rows: 24,
			ChangedRows: []int{0}, Screen: [][]state.LiveCell{{{Text: "stale", Width: 5}}},
		},
	})
	if len(effects) != 0 || root.Surface.Revision != 7 {
		t.Fatalf("unmergeable delta must keep the last valid screen, effects=%#v surface=%#v", effects, root.Surface)
	}
	request, _ = root.Surface.LiveScreenRequestRef(ref)
	if request.RequestInFlight || !request.NeedsBootstrap {
		t.Fatalf("unmergeable delta must release request and require bootstrap, got %#v", request)
	}
}

func TestLiveScreenNextEffectDoesNotUseFixedDelay(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	request := state.LiveScreenRequestState{EndpointID: state.DefaultEndpointID, TerminalID: "term-1", Demand: true, RequestInFlight: true, Generation: 1, SubmittedRevision: 7, Cols: 80, Rows: 24}
	effects := liveScreenNextEffectForRef(request, LiveDeps{Terminal: terminal})
	if len(effects) != 1 {
		t.Fatalf("expected one next-screen effect, got %#v", effects)
	}
	next := effects[0].(FuncEffect)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if msg := next.Run(ctx); msg == nil {
		t.Fatal("canceled request must return its generation-scoped result")
	} else if result, ok := msg.(LiveScreenNextResultMsg); !ok || !errors.Is(result.Err, context.Canceled) {
		t.Fatalf("canceled request should return context error, got %#v", msg)
	}
	if requests := terminal.LiveScreenNextRequestsSnapshot(); len(requests) != 1 {
		t.Fatalf("request cancellation is owned by context, got %#v", requests)
	}

	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 8}}
	msg := next.Run(context.Background())
	result, ok := msg.(LiveScreenNextResultMsg)
	if !ok || result.Snapshot.Revision != 8 {
		t.Fatalf("expected next-screen result without artificial delay, got %#v", msg)
	}
	if requests := terminal.LiveScreenNextRequestsSnapshot(); len(requests) != 2 {
		t.Fatalf("expected immediate request, got %#v", requests)
	}
}

func TestLiveFrameSubmissionStartsNextScreenRequest(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   8,
		Cols:       96,
		Rows:       30,
	})
	root := state.Root{
		Surface: surface,
	}

	_, effects := reducer(root, LiveScreenFrameSelectedMsg{Full: true, Targets: []render.LiveRenderTarget{{TerminalID: "term-1", Revision: 8}}})
	if len(effects) != 1 {
		t.Fatalf("expected one follow-up next-screen effect, got %#v", effects)
	}
	next := effects[0].(FuncEffect)
	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 9}}
	if msg := next.Run(context.Background()); msg == nil {
		t.Fatal("expected one-shot request to return latest screen")
	}
	requests := terminal.LiveScreenNextRequestsSnapshot()
	if len(requests) != 1 || requests[0].ObservedRevision != 8 || requests[0].Cols != 96 || requests[0].Rows != 30 {
		t.Fatalf("frame submission should request at rendered surface size, got %#v", requests)
	}
}

func TestLiveFrameSubmissionUsesOwningEndpoint(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	surface := (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: "west",
		TerminalID: "term-1",
		Revision:   8,
		Cols:       96,
		Rows:       30,
	})
	root := state.Root{Surface: surface}

	_, effects := reducer(root, LiveScreenFrameSelectedMsg{Full: true, Targets: []render.LiveRenderTarget{{EndpointID: "west", TerminalID: "term-1", Revision: 8}}})
	if len(effects) != 1 {
		t.Fatalf("expected one endpoint-scoped next-screen effect, got %#v", effects)
	}
	next := effects[0].(FuncEffect)
	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 9}}
	if msg := next.Run(context.Background()); msg == nil {
		t.Fatal("expected one-shot request to return latest screen")
	}
	requests := terminal.LiveScreenNextRequestsSnapshot()
	if len(requests) != 1 || requests[0].EndpointID != "west" || requests[0].TerminalID != "term-1" || requests[0].ObservedRevision != 8 {
		t.Fatalf("frame submission should request from owning endpoint, got %#v", requests)
	}
}

func TestLiveFrameSubmissionDedupesSharedTerminalViews(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextCh: make(chan port.TerminalSurfaceResult, 1)}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-shared").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-shared"}, state.SplitDirectionVertical)
	root := state.Root{Shell: shell}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-shared",
		Revision:   8,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"shared"},
	})
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-shared", 7, 96, 30, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewPaneTerminalView("pane-2", "term-shared", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false))

	selected := LiveScreenFrameSelectedMsg{Full: true, Targets: []render.LiveRenderTarget{{TerminalID: "term-shared", Revision: 8}}}
	root, effects := reducer(root, selected)
	if len(effects) != 1 {
		t.Fatalf("first submission should start exactly one request, got %#v", effects)
	}
	firstRequest, ok := effects[0].(FuncEffect)
	if !ok || firstRequest.Token != liveScreenNextTokenForRef(state.LocalTerminalRef("term-shared")) {
		t.Fatalf("first submission should use terminal-scoped request, got %#v", effects[0])
	}
	if request, _ := root.Surface.LiveScreenRequestRef(state.LocalTerminalRef("term-shared")); !request.Demand || !request.RequestInFlight || request.Cols != 96 || request.Rows != 30 {
		t.Fatalf("shared terminal should keep one request at owner size, got %#v", request)
	}

	root, effects = reducer(root, selected)
	if len(effects) != 0 {
		t.Fatalf("duplicate submission while request is pending must not start another, got %#v", effects)
	}
	if request, _ := root.Surface.LiveScreenRequestRef(state.LocalTerminalRef("term-shared")); !request.RequestInFlight {
		t.Fatalf("duplicate submission should leave request unchanged, got %#v", request)
	}

	terminal.LiveScreenNextCh <- port.TerminalSurfaceResult{Ready: true, Snapshot: state.LiveSurfaceSnapshot{Revision: 9, Cols: 96, Rows: 30, FullReplace: true}}
	result, ok := firstRequest.Run(context.Background()).(LiveScreenNextResultMsg)
	if !ok {
		t.Fatalf("pending request should return next-screen result")
	}
	root, effects = reducer(root, result)
	if len(effects) != 0 {
		t.Fatalf("received screen must wait for renderer submission, got %#v", effects)
	}
	if request, _ := root.Surface.LiveScreenRequestRef(state.LocalTerminalRef("term-shared")); request.RequestInFlight || request.ReceivedRevision != 9 || request.SubmittedRevision != 8 {
		t.Fatalf("received screen should be the sole pending revision, got %#v", request)
	}
	root, effects = reducer(root, LiveScreenFrameSelectedMsg{Targets: []render.LiveRenderTarget{{TerminalID: "term-shared", Revision: 9}}})
	if len(effects) != 1 {
		t.Fatalf("submitting revision 9 should immediately request the next screen, got %#v", effects)
	}
}

func TestLiveScreenNextContextCanceledDoesNotPostPanelError(t *testing.T) {
	terminal := &testkit.FakeTerminalService{LiveScreenNextErr: context.Canceled}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell()}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 7, Cols: 80, Rows: 24})
	root, effects := reducer(root, LiveScreenFrameSelectedMsg{Full: true, Targets: []render.LiveRenderTarget{{TerminalID: "term-1", Revision: 7}}})
	result := effects[0].(FuncEffect).Run(context.Background())
	next, effects := reducer(root, result)
	if len(effects) != 0 || next.Surface.Err != "" || next.Session.LastError != "" {
		t.Fatalf("context canceled next-screen request must stay silent, root=%#v effects=%#v", next, effects)
	}
}

func TestLiveScreenDemandCancellationDropsLateResult(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   8,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"ready"},
	})

	root, effects := reducer(root, LiveScreenFrameSelectedMsg{Full: true, Targets: []render.LiveRenderTarget{{TerminalID: "term-1", Revision: 8}}})
	if len(effects) != 1 {
		t.Fatalf("frame submission should start one request, got %#v", effects)
	}
	request, _ := root.Surface.LiveScreenRequestRef(state.LocalTerminalRef("term-1"))
	root, effects = reducer(root, LiveScreenFrameSelectedMsg{Full: true})
	if len(effects) != 1 {
		t.Fatalf("hiding the last live view should cancel its request, got %#v", effects)
	}
	if _, ok := effects[0].(CancelEffect); !ok {
		t.Fatalf("expected request cancellation, got %#v", effects[0])
	}
	root, effects = reducer(root, LiveScreenNextResultMsg{TerminalID: "term-1", Generation: request.Generation, Snapshot: state.LiveSurfaceSnapshot{Revision: 9, FullReplace: true}})
	if len(effects) != 0 || root.Surface.Revision != 8 {
		t.Fatalf("late hidden-view result must be ignored, surface=%#v effects=%#v", root.Surface, effects)
	}
}

func TestLiveScreenRequestsProgressIndependentlyAcrossPanes(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	refA := state.LocalTerminalRef("term-a")
	refB := state.LocalTerminalRef("term-b")
	root := state.Root{}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-a", Revision: 10, Cols: 80, Rows: 24, FullReplace: true})
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{TerminalID: "term-b", Revision: 20, Cols: 80, Rows: 24, FullReplace: true})
	root, effects := reducer(root, LiveScreenFrameSelectedMsg{Full: true, Targets: []render.LiveRenderTarget{
		{TerminalID: "term-a", Revision: 10},
		{TerminalID: "term-b", Revision: 20},
	}})
	if len(effects) != 2 {
		t.Fatalf("two visible refs should own independent requests, got %#v", effects)
	}
	requestA, _ := root.Surface.LiveScreenRequestRef(refA)
	root, effects = reducer(root, LiveScreenNextResultMsg{TerminalID: "term-a", Generation: requestA.Generation, Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-a", Revision: 11, Cols: 80, Rows: 24, FullReplace: true}})
	if len(effects) != 0 {
		t.Fatalf("received A must wait for renderer submission, got %#v", effects)
	}
	requestB, _ := root.Surface.LiveScreenRequestRef(refB)
	if !requestB.RequestInFlight {
		t.Fatal("A result must not release or replace B request")
	}
	root, effects = reducer(root, LiveScreenFrameSelectedMsg{Targets: []render.LiveRenderTarget{{TerminalID: "term-a", Revision: 11}}})
	requestA, _ = root.Surface.LiveScreenRequestRef(refA)
	requestB, _ = root.Surface.LiveScreenRequestRef(refB)
	if len(effects) != 1 || !requestA.RequestInFlight || !requestB.RequestInFlight {
		t.Fatalf("submitting A should overlap A next wait with B original wait, A=%#v B=%#v effects=%#v", requestA, requestB, effects)
	}
}

func TestLiveExitClearsPendingRefreshDebt(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := rootWithDirtyRefreshForLiveTest(t, state.LocalTerminalRef("term-1"))

	next, effects := reducer(root, LiveExitMsg{TerminalID: "term-1", ExitCode: 0, Reason: "done"})
	if len(effects) != 0 {
		t.Fatalf("live exit should not schedule refresh effects, got %#v", effects)
	}
	if refresh, ok := next.Surface.RefreshStateRef(state.LocalTerminalRef("term-1")); ok {
		t.Fatalf("live exit must clear pending refresh debt, got %#v", refresh)
	}
	if next.Surface.State != state.TerminalLiveExited || next.Session.State != state.TerminalLiveExited {
		t.Fatalf("live exit should still project exited lifecycle, surface=%#v session=%#v", next.Surface, next.Session)
	}
}

func TestLiveEventBoundaryClearsPendingRefreshDebt(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	cases := []struct {
		name  string
		event port.TerminalLiveEvent
	}{
		{
			name:  "exited",
			event: port.TerminalLiveEvent{TerminalID: "term-1", Exited: true, ExitCode: 0, Reason: "done"},
		},
		{
			name:  "error",
			event: port.TerminalLiveEvent{TerminalID: "term-1", Err: errors.New("transport offline")},
		},
		{
			name:  "terminal exited error",
			event: port.TerminalLiveEvent{TerminalID: "term-1", Err: errors.New("protocol error 400: terminal exited")},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			root := rootWithDirtyRefreshForLiveTest(t, state.LocalTerminalRef("term-1"))
			next, effects := reducer(root, LiveEventMsg{Event: tt.event})
			if len(effects) != 0 {
				t.Fatalf("boundary event should not schedule refresh effects, got %#v", effects)
			}
			if refresh, ok := next.Surface.RefreshStateRef(state.LocalTerminalRef("term-1")); ok {
				t.Fatalf("boundary event must clear pending refresh debt, got %#v", refresh)
			}
		})
	}
}

func TestLiveEventRefreshDoesNotStartAlternateScreenFetch(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Lines:      []string{"latest"},
			},
			LifecycleKnown: true,
		},
	}
	shell := state.DefaultShell().
		BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1").
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: "pane-2"})
	root := state.Root{
		Shell:   shell,
		Session: state.TerminalSessionStore{TerminalID: "term-1", Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-1",
			Cols:       70,
			Rows:       18,
			Lines:      []string{"stale"},
		},
	}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 96, 30, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false))

	next, effects := reduceLiveEvent(root, LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}}, LiveDeps{Terminal: terminal})
	if next.Generation != root.Generation || next.Surface.Lines[0] != "stale" {
		t.Fatalf("refresh invalidation should not mutate live state before surface returns, next=%#v", next.Surface)
	}
	if len(effects) != 0 || len(terminal.Surfaces) != 0 {
		t.Fatalf("passive refresh hint must not start a second screen source, effects=%#v requests=%#v", effects, terminal.Surfaces)
	}
	refreshMsg := LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}}
	if !refreshMsg.SkipRender() {
		t.Fatal("ordinary refresh invalidation should not render stale frame")
	}
}

func rootWithDirtyRefreshForLiveTest(t *testing.T, ref state.TerminalRef) state.Root {
	t.Helper()
	root := state.Root{}
	root.Surface = root.Surface.ApplySnapshot(state.LiveSurfaceSnapshot{
		EndpointID: ref.EndpointID,
		TerminalID: ref.TerminalID,
		Revision:   10,
		Cols:       96,
		Rows:       30,
		Lines:      []string{"current"},
	})
	root.Session = root.Session.AttachRefWithResizeOwner(ref, 7, 96, 30, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID))
	var fetch bool
	root.Surface, fetch = root.Surface.RequestRefreshRef(ref, 96, 30)
	if !fetch {
		t.Fatalf("expected refresh request to start fetch, store=%#v", root.Surface.Refreshes)
	}
	root.Surface, fetch = root.Surface.RequestRefreshRef(ref, 120, 40)
	if fetch {
		t.Fatalf("dirty invalidation must not start a parallel fetch, store=%#v", root.Surface.Refreshes)
	}
	if refresh, ok := root.Surface.RefreshStateRef(ref); !ok || !refresh.InFlight || !refresh.Dirty {
		t.Fatalf("expected dirty refresh state, got %#v ok=%v", refresh, ok)
	}
	return root
}

func TestRepeatedLiveEventRefreshHintsRemainNoops(t *testing.T) {
	terminal := &testkit.FakeTerminalService{SurfaceResult: port.TerminalSurfaceResult{
		Ready:    true,
		Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Lines: []string{"latest"}},
	}}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Surface: state.TerminalSurfaceStore{TerminalID: "term-1", Cols: 80, Rows: 24}}

	root, effects := reducer(root, LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}})
	if len(effects) != 0 {
		t.Fatalf("first passive refresh hint should be ignored, got %#v", effects)
	}
	root, effects = reducer(root, LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}})
	if len(effects) != 0 {
		t.Fatalf("second refresh while in-flight should only mark dirty, got %#v", effects)
	}
	if _, ok := root.Surface.RefreshStateRef(state.LocalTerminalRef("term-1")); ok {
		t.Fatalf("passive hints must not allocate explicit refresh state, got %#v", root.Surface.Refreshes)
	}
}

func TestLiveEventRefreshPayloadCannotReplaceCanonicalScreen(t *testing.T) {
	terminal := &testkit.FakeTerminalService{SurfaceResult: port.TerminalSurfaceResult{
		Ready:    true,
		Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-1", Revision: 11, Lines: []string{"latest"}},
	}}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   10,
		Lines:      []string{"rev10"},
	})}

	root, effects := reducer(root, LiveEventMsg{Event: port.TerminalLiveEvent{
		TerminalID: "term-1",
		Refresh:    true,
		Snapshot:   state.LiveSurfaceSnapshot{Revision: 5},
	}})
	if len(effects) != 0 || len(terminal.Surfaces) != 0 {
		t.Fatalf("ordinary refresh must not fetch or apply event payload, effects=%#v requests=%#v", effects, terminal.Surfaces)
	}
	if root.Surface.SurfaceForTerminal("term-1").Revision != 10 {
		t.Fatalf("refresh event alone must not mutate current surface, got %#v", root.Surface.SurfaceForTerminal("term-1"))
	}
}

func TestLiveEventRefreshDoesNotTriggerLayoutResizeMeasurement(t *testing.T) {
	root := state.Root{Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24}}
	refresh := LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Refresh: true}}
	if terminalLayoutMayNeedResize(root, refresh) {
		t.Fatal("ordinary live refresh must not enter layout resize measurement")
	}
	metadata := LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Metadata: true}}
	if !terminalLayoutMayNeedResize(root, metadata) {
		t.Fatal("metadata event should still allow layout resize checks")
	}
}

func TestTerminalPoolReconnectUsesActiveViewIdentity(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 11, Cols: 80, Rows: 24, CanResize: true}}
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{Shell: state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-logs", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)}

	root, effects := reducer(root, TerminalPoolReconnectRequestMsg{TerminalID: "term-1"})
	if len(effects) != 1 {
		t.Fatalf("expected reconnect effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	if len(terminal.Reconnects) != 1 || terminal.Reconnects[0].ViewID != state.TerminalPaneViewID("pane-logs") || terminal.Reconnects[0].SurfaceID != "tui" {
		t.Fatalf("reconnect should use active pane view identity, requests=%#v", terminal.Reconnects)
	}
	root, _ = reducer(root, msg)
	if binding, ok := root.TerminalViews.PaneBinding("pane-logs"); !ok || binding.Channel != 11 || binding.TerminalID != "term-1" {
		t.Fatalf("reconnect result should bind active pane view, binding=%#v ok=%v", binding, ok)
	}
}

func TestLiveMetadataProjectionUpdatesGlobalOwnerAndAttachmentCount(t *testing.T) {
	rootA := state.Root{
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running", AttachmentCount: 2}}},
	}
	rootA.TerminalViews = rootA.TerminalViews.BindPane(state.NewPaneTerminalView("pane-1", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-a", "pane:main", true))
	rootB := state.Root{
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{TerminalID: "term-1", Title: "shell", State: "running", AttachmentCount: 2}}},
	}
	rootB.TerminalViews = rootB.TerminalViews.BindPane(state.NewPaneTerminalView("pane-1", "term-1", 8, 80, 24, state.TerminalResizeRoleOwner, "surface-b", "pane:main", true))

	event := LiveEventMsg{Event: port.TerminalLiveEvent{
		TerminalID:           "term-1",
		Metadata:             true,
		AttachmentProjection: true,
		AttachmentCount:      4,
		OwnerSurfaceID:       "surface-b",
		OwnerViewID:          "pane:main",
		ResizeEpoch:          9,
	}}
	nextA, _ := reduceLiveEvent(rootA, event, LiveDeps{})
	nextB, _ := reduceLiveEvent(rootB, event, LiveDeps{})
	if got := nextA.TerminalPool.Items[0].AttachmentCount; got != 4 {
		t.Fatalf("metadata projection should update global attachment count, got %d", got)
	}
	if got := nextB.TerminalPool.Items[0].AttachmentCount; got != 4 {
		t.Fatalf("metadata projection should update global attachment count in owner TUI, got %d", got)
	}
	first, _ := nextA.TerminalViews.PaneBinding("pane-1")
	second, _ := nextB.TerminalViews.PaneBinding("pane-1")
	if first.HasResizeOwner() || first.CanResize {
		t.Fatalf("same panel id in other surface must be follower, got %#v", first)
	}
	if !second.HasResizeOwner() || !second.CanResize {
		t.Fatalf("matching surface+panel should remain owner, got %#v", second)
	}
}

func TestLiveAppLayoutResizePreservesAttachResizeOwner(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      9,
			Cols:         80,
			Rows:         24,
			CanResize:    true,
			ResizePolicy: "owner",
			SurfaceID:    "surface-1",
			ViewID:       "view-1",
		},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{
		TerminalID:   "term-1",
		Cols:         80,
		Rows:         24,
		Mode:         "collaborator",
		ResizePolicy: "owner",
		SurfaceID:    "surface-1",
		ViewID:       "view-1",
	}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if len(terminal.Resizes) != 1 {
		t.Fatalf("expected attach correction resize, got %#v", terminal.Resizes)
	}
	got := terminal.Resizes[0]
	if got.Cols != 78 || got.Rows != 20 || got.ResizePolicy != "owner" || got.SurfaceID != "surface-1" || got.ViewID != "view-1" {
		t.Fatalf("layout resize must preserve attach owner metadata, got %#v", got)
	}
}

func TestTerminalLayoutResizeUsesSharedOwnerWhenActiveViewIsFollower(t *testing.T) {
	reducer := NewTerminalLayoutResizeReducer()
	shell := state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	next, effects := reducer(root, HostResizeMsg{Cols: 120, Rows: 40})
	if len(effects) != 1 {
		t.Fatalf("visible shared owner should resize PTY even when active view is follower, got %#v", effects)
	}
	ownerMsg := effects[0].(FuncEffect).Run(context.Background()).(LiveResizeMsg)
	if ownerMsg.ViewID != "view-1" || ownerMsg.Seq != 1 || ownerMsg.Cols <= 0 || ownerMsg.Rows <= 0 {
		t.Fatalf("resize effect should carry shared owner view identity, got %#v", ownerMsg)
	}
	if got, _ := next.TerminalViews.PaneBinding("pane-2"); got.DesiredCols != 40 || got.DesiredRows != 12 || got.RequestSeq != 0 {
		t.Fatalf("follower layout resize must not mutate desired size, got %#v", got)
	}
	if got, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID); got.RequestSeq != 1 || got.DesiredCols != ownerMsg.Cols || got.DesiredRows != ownerMsg.Rows {
		t.Fatalf("owner desired size should track request, got %#v msg=%#v", got, ownerMsg)
	}

	root.TerminalViews = root.TerminalViews.TransferPaneResizeOwner("pane-2")
	next, effects = reducer(root, HostResizeMsg{Cols: 120, Rows: 40})
	if len(effects) != 1 {
		t.Fatalf("active owner should schedule one resize effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background()).(LiveResizeMsg)
	if msg.ViewID != "view-2" || msg.Seq != 1 || msg.Cols <= 0 || msg.Rows <= 0 {
		t.Fatalf("resize effect should carry active owner view identity, got %#v", msg)
	}
	if got, _ := next.TerminalViews.PaneBinding("pane-2"); got.RequestSeq != 1 || got.DesiredCols != msg.Cols || got.DesiredRows != msg.Rows {
		t.Fatalf("owner view desired size should track request, got %#v msg=%#v", got, msg)
	}
}

func TestTerminalLayoutResizeUsesEndpointSharedOwner(t *testing.T) {
	reducer := NewTerminalLayoutResizeReducer()
	shell := state.DefaultShell().SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Session:  state.TerminalSessionStore{EndpointID: "west", TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-west-owner", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", "pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-west-follower", false))

	next, effects := reducer(root, HostResizeMsg{Cols: 120, Rows: 40})
	if len(effects) != 1 {
		t.Fatalf("west follower should resize west owner, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background()).(LiveResizeMsg)
	if msg.EndpointID != "west" || msg.ViewID != "view-west-owner" || msg.TerminalID != "term-1" {
		t.Fatalf("layout resize must preserve owner endpoint, got %#v", msg)
	}
	if got, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID); got.EndpointID != "west" || got.RequestSeq != msg.Seq {
		t.Fatalf("west owner binding should track endpoint-scoped resize, binding=%#v msg=%#v", got, msg)
	}
}

func TestTerminalLayoutResizeOwnerPaneResizeChangesPTYSize(t *testing.T) {
	reducer := ComposeReducers(NewShellReducer(), NewTerminalLayoutResizeReducer())
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	next, effects := reducer(root, ShellPaneCommandMsg{Command: state.PaneCommand{Action: state.PaneCommandResize, Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID}, ResizeDirection: state.PaneResizeRight, Delta: 6, Source: state.PaneCommandSourceTest}})
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok {
		t.Fatalf("owner pane resize should schedule one PTY resize effect, got %#v", effects)
	}
	if msg.ViewID != "view-1" || msg.Seq != 1 || msg.Cols == 80 || msg.Rows == 24 {
		t.Fatalf("resize effect should carry changed owner view size, got %#v", msg)
	}
	owner, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if owner.RequestSeq != 1 || owner.DesiredCols != msg.Cols || owner.DesiredRows != msg.Rows {
		t.Fatalf("owner desired size should track pane resize request, got %#v msg=%#v", owner, msg)
	}
}

func TestTerminalSizeLockBlocksSplitLayoutResize(t *testing.T) {
	reducer := ComposeReducers(NewShellReducer(), NewTerminalLayoutResizeReducer())
	root := state.Root{
		Shell:    state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine).FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 100, Rows: 30},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.ApplyTerminalSizeLock("term-1", true)

	next, effects := reducer(root, ShellSplitActivePaneMsg{
		Pane:      state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive},
		Direction: state.SplitDirectionHorizontal,
	})
	if _, ok := liveResizeMsgFromEffects(effects); ok {
		t.Fatalf("splitting a locked terminal must not emit PTY resize, effects=%#v", effects)
	}
	follower, _ := next.TerminalViews.PaneBinding("pane-2")
	if follower.ResizeRole != state.TerminalResizeRoleFollower || follower.CanResize || !follower.SizeLocked || follower.ControlReason != "size_locked" {
		t.Fatalf("new split view should inherit terminal lock as follower intent, got %#v", follower)
	}
	previous, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if previous.CanResize || !previous.SizeLocked {
		t.Fatalf("previous pane should remain locked after split, got %#v", previous)
	}

	next.TerminalViews = next.TerminalViews.ApplyTerminalSizeLock("term-1", false)
	follower, _ = next.TerminalViews.PaneBinding("pane-2")
	if follower.SizeLocked || follower.CanResize || follower.ResizeRole != state.TerminalResizeRoleFollower {
		t.Fatalf("unlock should clear lock projection without inventing owner authority, got %#v", follower)
	}
}

func TestTerminalSizeUnlockResizesOwnerWhenPanelSizeDiverged(t *testing.T) {
	reducer := ComposeReducers(newTerminalPoolReducerPrepared(LiveDeps{}), NewTerminalLayoutResizeReducer())
	root := state.Root{
		Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "main",
			Tags:       map[string]string{"anytty.size_lock": "lock"},
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.ApplyTerminalSizeLock("term-1", true)

	next, effects := reducer(root, TerminalSizeLockToggleResultMsg{TerminalID: "term-1", Tags: map[string]string{}, Locked: false})
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok {
		t.Fatalf("unlock should trigger owner resize when panel and terminal size diverged, effects=%#v", effects)
	}
	if msg.ViewID != "view-1" || msg.Cols == 80 || msg.Rows == 24 {
		t.Fatalf("unlock resize should use owner view content rect, got %#v", msg)
	}
	owner, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if owner.SizeLocked || !owner.CanResize || owner.RequestSeq != msg.Seq || owner.DesiredCols != msg.Cols || owner.DesiredRows != msg.Rows {
		t.Fatalf("owner should be unlocked and track requested resize, owner=%#v msg=%#v", owner, msg)
	}
}

func TestLiveMetadataEventProjectsTerminalSizeLockAndUnlockResize(t *testing.T) {
	reducer := ComposeReducers(newLiveReducerPrepared(LiveDeps{}), NewTerminalLayoutResizeReducer())
	root := state.Root{
		Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "main",
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 80, 24, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	locked, effects := reducer(root, LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Metadata: true, Tags: map[string]string{"anytty.size_lock": "lock"}}})
	if _, ok := liveResizeMsgFromEffects(effects); ok {
		t.Fatalf("metadata lock must not resize PTY, effects=%#v", effects)
	}
	owner, _ := locked.TerminalViews.PaneBinding(state.DefaultPaneID)
	follower, _ := locked.TerminalViews.PaneBinding("pane-2")
	if !owner.SizeLocked || owner.CanResize || !follower.SizeLocked || follower.CanResize {
		t.Fatalf("metadata lock should broadcast to all terminal views, owner=%#v follower=%#v", owner, follower)
	}
	if locked.TerminalPool.Items[0].Tags["anytty.size_lock"] != "lock" {
		t.Fatalf("metadata lock should update terminal pool tags, pool=%#v", locked.TerminalPool)
	}

	unlocked, effects := reducer(locked, LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Metadata: true, Tags: map[string]string{}}})
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok {
		t.Fatalf("metadata unlock should trigger owner resize when panel and terminal size diverged, effects=%#v", effects)
	}
	owner, _ = unlocked.TerminalViews.PaneBinding(state.DefaultPaneID)
	follower, _ = unlocked.TerminalViews.PaneBinding("pane-2")
	if owner.SizeLocked || !owner.CanResize || follower.SizeLocked || follower.CanResize {
		t.Fatalf("metadata unlock should broadcast unlocked projection, owner=%#v follower=%#v", owner, follower)
	}
	if msg.ViewID != "view-1" {
		t.Fatalf("metadata unlock resize should target owner view, got %#v", msg)
	}
}

func TestTerminalSizeLockBlocksAttachResultResizeToSplitPane(t *testing.T) {
	reducer := ComposeReducers(NewShellReducer(), newTerminalPoolReducerPrepared(LiveDeps{}), NewTerminalLayoutResizeReducer())
	root := state.Root{
		Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 100, Rows: 30, DesiredCols: 100, DesiredRows: 30},
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "main",
			Tags:       map[string]string{"anytty.size_lock": "lock"},
			Cols:       100,
			Rows:       30,
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 100, 30, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.ApplyTerminalSizeLock("term-1", true)

	root, effects := reducer(root, ShellSplitActivePaneMsg{
		Pane:      state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneEmpty},
		Direction: state.SplitDirectionHorizontal,
	})
	if _, ok := liveResizeMsgFromEffects(effects); ok {
		t.Fatalf("splitting an empty sibling for locked terminal must not emit PTY resize, effects=%#v", effects)
	}
	if root.Shell.EnsureDefaults().ActivePaneID != "pane-2" {
		t.Fatalf("new split pane should be active before attach, shell=%#v", root.Shell.EnsureDefaults())
	}

	next, effects := reducer(root, TerminalPoolAttachResultMsg{
		TerminalID: "term-1",
		Result: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      8,
			Cols:         60,
			Rows:         14,
			ResizePolicy: state.TerminalResizeRoleOwner,
			SurfaceID:    "surface",
			ViewID:       state.TerminalPaneViewID("pane-2"),
			CanResize:    true,
		},
	})
	if _, ok := liveResizeMsgFromEffects(effects); ok {
		t.Fatalf("attaching split pane to locked terminal must not emit PTY resize, effects=%#v", effects)
	}
	binding, ok := next.TerminalViews.PaneBinding("pane-2")
	if !ok || binding.CanResize || !binding.SizeLocked || binding.ControlReason != "size_locked" {
		t.Fatalf("attached split pane should inherit terminal lock without resize authority, binding=%#v ok=%v", binding, ok)
	}
	previous, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || previous.CanResize || !previous.SizeLocked {
		t.Fatalf("previous binding should remain locked, binding=%#v ok=%v", previous, ok)
	}
}

func TestTerminalPoolAttachRequestToSameLockedTerminalDoesNotResize(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-1",
			Channel:      8,
			Cols:         60,
			Rows:         14,
			ResizePolicy: state.TerminalResizeRoleOwner,
			CanResize:    true,
			SurfaceID:    "surface",
			ViewID:       state.TerminalPaneViewID("pane-2"),
		},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 40)
	root := state.Root{
		Shell: state.DefaultShell().
			SetPanelPresentation(state.PanelPresentationSplitLine).
			FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 100, Rows: 30, DesiredCols: 100, DesiredRows: 30},
		TerminalPool: state.TerminalPoolStore{Items: []state.TerminalPoolItem{{
			TerminalID: "term-1",
			Title:      "main",
			Tags:       map[string]string{"anytty.size_lock": "lock"},
			Cols:       100,
			Rows:       30,
		}}},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 100, 30, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))
	root.TerminalViews = root.TerminalViews.ApplyTerminalSizeLock("term-1", true)
	runtime := NewInteractiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal}, CopyModeDeps{Core: &testkit.FakeCoreClient{}})

	if err := runtime.Post(ShellSplitActivePaneMsg{
		Pane:      state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneEmpty},
		Direction: state.SplitDirectionHorizontal,
	}); err != nil {
		t.Fatalf("post split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split: %v", err)
	}
	if err := runtime.Post(TerminalPoolAttachRequestMsg{TerminalID: "term-1"}); err != nil {
		t.Fatalf("post attach request: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if len(terminal.Attaches) != 1 {
		t.Fatalf("expected one attach request, got %#v", terminal.Attaches)
	}
	if got := terminal.Attaches[0]; got.ResizePolicy != state.TerminalResizeRoleFollower || got.ViewID != state.TerminalPaneViewID("pane-2") {
		t.Fatalf("same-terminal pool attach should be a follower request for the new pane, got %#v", got)
	}
	if len(terminal.Resizes) != 0 {
		t.Fatalf("locked terminal attach must not resize PTY, got %#v", terminal.Resizes)
	}
	binding, ok := runtime.State().TerminalViews.PaneBinding("pane-2")
	if !ok || binding.CanResize || !binding.SizeLocked || binding.ControlReason != "size_locked" {
		t.Fatalf("attached pane should stay size-locked without resize authority, binding=%#v ok=%v", binding, ok)
	}
}

func TestTakeResizeOwnerAttachResultTriggersOwnerViewResize(t *testing.T) {
	reducer := ComposeReducers(newLiveReducerPrepared(LiveDeps{}), NewTerminalLayoutResizeReducer())
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	next, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8, Cols: 40, Rows: 12, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "surface", ViewID: "view-2", CanResize: true, OwnerSurfaceID: "surface", OwnerViewID: "view-2", ResizeEpoch: 2}})
	owner, _ := next.TerminalViews.PaneBinding("pane-2")
	if !owner.HasAuthoritativeResizeOwner() {
		t.Fatalf("attach result should project clicked view as authoritative owner, got %#v", owner)
	}
	previous, _ := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if previous.ResizeRole != state.TerminalResizeRoleFollower || previous.CanResize {
		t.Fatalf("previous owner should be demoted by authoritative result, got %#v", previous)
	}
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok || msg.ViewID != "view-2" || msg.Seq != 1 || msg.Cols <= 40 || msg.Rows <= 12 {
		t.Fatalf("owner attach result should immediately resize to active content rect, got msg=%#v effects=%#v", msg, effects)
	}
}

func TestTakeResizeOwnerAttachResultTriggersSameSizeOwnerResize(t *testing.T) {
	reducer := ComposeReducers(newLiveReducerPrepared(LiveDeps{}), NewTerminalLayoutResizeReducer())
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
		Viewport: state.ViewportStore{Valid: true, Cols: 122, Rows: 28},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 59, Rows: 24, DesiredCols: 59, DesiredRows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 59, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 59, 24, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	next, effects := reducer(root, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8, Cols: 59, Rows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "surface", ViewID: "view-2", CanResize: true, OwnerSurfaceID: "surface", OwnerViewID: "view-2", ResizeEpoch: 2}})
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok || msg.ViewID != "view-2" || msg.Seq != 1 || msg.Cols != 59 || msg.Rows != 24 {
		t.Fatalf("owner attach result should force same-size resize check, msg=%#v effects=%#v", msg, effects)
	}
	owner, _ := next.TerminalViews.PaneBinding("pane-2")
	if owner.ResizePending || owner.RequestSeq != 1 {
		t.Fatalf("same-size owner resize request should clear pending, got %#v", owner)
	}
}

func TestViewScopedOwnerResizeUsesBindingResizePolicy(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Session: state.TerminalSessionStore{
			TerminalID:   "term-1",
			Channel:      8,
			Attached:     true,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface-follower",
			ViewID:       "view-follower",
		},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-owner", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-owner", "view-owner", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-follower", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface-follower", "view-follower", false))
	var decision state.TerminalViewResizeDecision
	root.TerminalViews, decision = root.TerminalViews.RequestViewResize("view-owner", 100, 30)
	if !decision.Allowed || !decision.Changed {
		t.Fatalf("expected owner view resize decision, got %#v", decision)
	}

	next, effects := reducer(root, LiveResizeMsg{TerminalID: "term-1", Cols: 100, Rows: 30, Seq: decision.Seq, ViewID: "view-owner"})
	if len(effects) != 1 {
		t.Fatalf("view-scoped owner resize should emit terminal resize effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	next, _ = reducer(next, msg)
	if len(terminal.Resizes) != 1 {
		t.Fatalf("expected one terminal resize request, got %#v", terminal.Resizes)
	}
	got := terminal.Resizes[0]
	if got.Channel != 7 || got.ResizePolicy != state.TerminalResizeRoleOwner || got.SurfaceID != "surface-owner" || got.ViewID != "view-owner" {
		t.Fatalf("view-scoped resize must use owner binding identity, got %#v", got)
	}
	if binding, ok := next.TerminalViews.PaneBinding("pane-owner"); !ok || binding.DesiredCols != 100 || binding.DesiredRows != 30 {
		t.Fatalf("owner binding should track requested desired size, binding=%#v ok=%v", binding, ok)
	}
}

func TestViewScopedOwnerResizeRoutesEndpointBinding(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	reducer := newLiveReducerPrepared(LiveDeps{Terminal: terminal})
	root := state.Root{
		Session: state.TerminalSessionStore{
			EndpointID:    "west",
			TerminalID:    "term-1",
			Channel:       8,
			Attached:      true,
			ResizePolicy:  state.TerminalResizeRoleFollower,
			SurfaceID:     "surface-follower",
			ViewID:        "view-follower",
			InputChannels: map[string]uint16{"west/term-1": 8},
		},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewEndpointPaneTerminalView("west", "pane-owner", "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface-owner", "view-owner", true))
	var decision state.TerminalViewResizeDecision
	root.TerminalViews, decision = root.TerminalViews.RequestViewResize("view-owner", 100, 30)
	if !decision.Allowed || !decision.Changed {
		t.Fatalf("expected owner view resize decision, got %#v", decision)
	}

	next, effects := reducer(root, LiveResizeMsg{EndpointID: "west", TerminalID: "term-1", Cols: 100, Rows: 30, Seq: decision.Seq, ViewID: "view-owner"})
	if len(effects) != 1 {
		t.Fatalf("view-scoped owner resize should emit terminal resize effect, got %#v", effects)
	}
	msg := effects[0].(FuncEffect).Run(context.Background())
	next, _ = reducer(next, msg)
	if len(terminal.Resizes) != 1 || terminal.Resizes[0].EndpointID != "west" || terminal.Resizes[0].Channel != 7 {
		t.Fatalf("resize request must route to west owner binding, got %#v", terminal.Resizes)
	}
	if !next.Session.TerminalRef().Equal(state.NewTerminalRef("west", "term-1")) {
		t.Fatalf("resize result must preserve west session ref, got %#v", next.Session.TerminalRef())
	}
}

func TestPaneTakeOwnerCommandCarriesViewScopedResizeSeq(t *testing.T) {
	reducer := NewShellReducer()
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: "pane-2"}),
		Viewport: state.ViewportStore{Valid: true, Cols: 120, Rows: 40},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", "view-2", false))

	next, effects := reducer(root, shortcutTestMessage("panel.take_owner", "pane-2", false, 0))
	msg, ok := liveResizeMsgFromEffects(effects)
	if !ok || msg.ViewID != "view-2" || msg.Seq == 0 {
		t.Fatalf("take owner command should emit view-scoped resize with binding seq, msg=%#v effects=%#v", msg, effects)
	}
	binding, _ := next.TerminalViews.PaneBinding("pane-2")
	if binding.RequestSeq != msg.Seq {
		t.Fatalf("resize message seq must match target binding seq, binding=%#v msg=%#v", binding, msg)
	}
}

func liveResizeMsgFromEffects(effects []Effect) (LiveResizeMsg, bool) {
	for _, effect := range effects {
		funcEffect, ok := effect.(FuncEffect)
		if !ok || funcEffect.Run == nil {
			continue
		}
		msg, ok := funcEffect.Run(context.Background()).(LiveResizeMsg)
		if ok {
			return msg, true
		}
	}
	return LiveResizeMsg{}, false
}

func TestLiveResizeResultUsesViewScopedStaleGuard(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 80, DesiredRows: 24, ResizeRequestSeq: 9},
		Surface: state.TerminalSurfaceStore{TerminalID: "term-1", Cols: 80, Rows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 100, 30, state.TerminalResizeRoleOwner, "surface", "view-1", true))

	next, effects := reducer(root, LiveResizeResultMsg{ViewID: "view-1", Seq: 1, Cols: 100, Rows: 30})
	if len(effects) != 0 {
		t.Fatalf("resize result should not emit effects, got %#v", effects)
	}
	if next.Session.Cols != 100 || next.Surface.Cols != 100 {
		t.Fatalf("view-scoped result should not be rejected by global seq, got session=%#v surface=%#v", next.Session, next.Surface)
	}
}

func TestLiveResizeResultProjectsTerminalSizeLockWithoutResizingSurface(t *testing.T) {
	reducer := newLiveReducerPrepared(LiveDeps{})
	root := state.Root{
		Session: state.TerminalSessionStore{TerminalID: "term-1", Attached: true, Cols: 80, Rows: 24, DesiredCols: 100, DesiredRows: 30},
		Surface: state.TerminalSurfaceStore{TerminalID: "term-1", Cols: 80, Rows: 24},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 100, 30, state.TerminalResizeRoleOwner, "surface", "view-1", true))

	next, effects := reducer(root, LiveResizeResultMsg{
		ViewID: "view-1",
		Seq:    1,
		Cols:   100,
		Rows:   30,
		Result: port.TerminalResizeResult{
			TerminalID:     "term-1",
			Cols:           80,
			Rows:           24,
			Resized:        false,
			CanResize:      false,
			SizeLocked:     true,
			ControlReason:  "size_locked",
			OwnerSurfaceID: "surface",
			OwnerViewID:    "view-1",
			ResizeEpoch:    3,
			ResizePolicy:   state.TerminalResizeRoleOwner,
			SurfaceID:      "surface",
			ViewID:         "view-1",
		},
	})
	if len(effects) != 0 {
		t.Fatalf("locked resize result should not emit effects, got %#v", effects)
	}
	if next.Session.Cols != 80 || next.Surface.Cols != 80 {
		t.Fatalf("locked resize result must not resize session/surface, got session=%#v surface=%#v", next.Session, next.Surface)
	}
	binding, ok := next.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || !binding.SizeLocked || binding.CanResize || binding.ControlReason != "size_locked" || binding.ResizeEpoch != 3 {
		t.Fatalf("expected terminal size lock projection on binding, got %#v", binding)
	}
	if binding.Layout.SizeLocked {
		t.Fatalf("terminal size lock must not mutate view-local layout lock, got %#v", binding.Layout)
	}
}

func TestLiveAppInputDisplaysOnlyAfterSurfaceEventAndExitState(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 4, Cols: 78, Rows: 20},
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send input: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain input: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "x" || terminal.Inputs[0].Channel != 4 {
		t.Fatalf("expected terminal service input, got %#v", terminal.Inputs)
	}
	beforeSurface := lastFrame(t, host.Frames())
	if frameContains(beforeSurface, "typed x") {
		t.Fatalf("runtime must not fake local echo before live surface event, got %#v", beforeSurface.Lines)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       78,
		Rows:       20,
		Lines:      []string{"$ typed x", "echo x"},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 6, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain surface: %v", err)
	}
	afterSurface := lastFrame(t, host.Frames())
	if !frameContains(afterSurface, "$ typed x") || !afterSurface.Cursor.Visible || afterSurface.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("expected service-returned live content and cursor, got lines=%#v cursor=%#v", afterSurface.Lines, afterSurface.Cursor)
	}
	if err := runtime.Post(LiveExitMsg{TerminalID: "term-1", ExitCode: 0, Reason: "shell exited"}); err != nil {
		t.Fatalf("post exit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exit: %v", err)
	}
	if runtime.State().Session.Attached || runtime.State().Session.State != state.TerminalLiveExited {
		t.Fatalf("expected detached exited session, got %#v", runtime.State().Session)
	}
	exitFrame := lastFrame(t, host.Frames())
	if !frameContains(exitFrame, "exited: term-1 code:0 shell exited") || !frameContains(exitFrame, "$ typed x") {
		t.Fatalf("expected exit status with preserved last surface, got %#v", exitFrame.Lines)
	}
}

func TestLiveInputRoutesToFocusedPaneTerminal(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 1, Cols: 78, Rows: 20},
	}
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		SplitActivePane(state.PaneState{ID: "pane-3", Title: "build", Kind: state.PaneTerminalLive, TerminalID: "term-3"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	host := NewFakeTerminalHost(16)
	host.SetSize(90, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: shell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 90, Rows: 24}}); err != nil {
		t.Fatalf("post main attach: %v", err)
	}
	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-2", Channel: 2, Cols: 42, Rows: 20, ViewID: state.TerminalPaneViewID("pane-2")}}); err != nil {
		t.Fatalf("post term-2 attach result: %v", err)
	}
	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-3", Channel: 3, Cols: 42, Rows: 20, ViewID: state.TerminalPaneViewID("pane-3")}}); err != nil {
		t.Fatalf("post term-3 attach result: %v", err)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{Action: state.PaneCommandFocus, Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID}, Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("restore main focus: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	frame := lastFrame(t, host.Frames())
	pane2Content := frameHitRegion(t, frame, render.HitRegionPaneContent, "pane-2")
	if err := host.SendInput(mouseEventAt(pane2Content.Rect)); err != nil {
		t.Fatalf("click pane-2: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "b"}); err != nil {
		t.Fatalf("send pane-2 key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane-2 key: %v", err)
	}
	if got := runtime.State().Shell.EnsureDefaults().ActivePaneID; got != "pane-2" {
		t.Fatalf("click should focus pane-2, got %q", got)
	}

	frame = lastFrame(t, host.Frames())
	pane3Content := frameHitRegion(t, frame, render.HitRegionPaneContent, "pane-3")
	if err := host.SendInput(mouseEventAt(pane3Content.Rect)); err != nil {
		t.Fatalf("click pane-3: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "c"}); err != nil {
		t.Fatalf("send pane-3 key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane-3 key: %v", err)
	}

	if len(terminal.Inputs) != 2 {
		t.Fatalf("expected two routed inputs, got %#v", terminal.Inputs)
	}
	if terminal.Inputs[0].TerminalID != "term-2" || terminal.Inputs[0].Channel != 2 || string(terminal.Inputs[0].Bytes) != "b" {
		t.Fatalf("pane-2 input should route to term-2, got %#v", terminal.Inputs[0])
	}
	if terminal.Inputs[1].TerminalID != "term-3" || terminal.Inputs[1].Channel != 3 || string(terminal.Inputs[1].Bytes) != "c" {
		t.Fatalf("pane-3 input should route to term-3, got %#v", terminal.Inputs[1])
	}
}

func TestMousePaneContentActivationExitsInteractionModeBeforeLiveInput(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	shell := state.DefaultShell().
		SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-2"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}).
		SetInteractionMode(state.InteractionModePane)
	host := NewFakeTerminalHost(16)
	host.SetSize(90, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: shell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-main", Channel: 1, Cols: 42, Rows: 20, ViewID: state.TerminalPaneViewID(state.DefaultPaneID)}}); err != nil {
		t.Fatalf("post main attach result: %v", err)
	}
	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-2", Channel: 2, Cols: 42, Rows: 20, ViewID: state.TerminalPaneViewID("pane-2")}}); err != nil {
		t.Fatalf("post term-2 attach result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	frame := lastFrame(t, host.Frames())
	pane2Content := frameHitRegion(t, frame, render.HitRegionPaneContent, "pane-2")
	if err := host.SendInput(mouseEventAt(pane2Content.Rect)); err != nil {
		t.Fatalf("click pane-2: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"}); err != nil {
		t.Fatalf("send pane-2 key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane-2 key: %v", err)
	}

	shell = runtime.State().Shell.EnsureDefaults()
	if shell.ActivePaneID != "pane-2" || shell.ActiveFloatingID() != "" || shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("content click should activate pane input mode, shell=%#v", shell)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].TerminalID != "term-2" || terminal.Inputs[0].Channel != 2 || string(terminal.Inputs[0].Bytes) != "l" {
		t.Fatalf("pane input should route to clicked pane terminal after mode exit, inputs=%#v", terminal.Inputs)
	}
}

func TestLiveInputTargetsActiveFloatingBeforeTiledPane(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 1, Cols: 78, Rows: 20}}
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: shell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post main attach: %v", err)
	}
	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-float", Channel: 9, Cols: 28, Rows: 6, ViewID: state.TerminalFloatingViewID("floating-1")}}); err != nil {
		t.Fatalf("post floating attach result: %v", err)
	}
	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: "floating-1", Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("focus floating: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"}); err != nil {
		t.Fatalf("send floating key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating key: %v", err)
	}

	if len(terminal.Inputs) != 1 || terminal.Inputs[0].TerminalID != "term-float" || terminal.Inputs[0].Channel != 9 || string(terminal.Inputs[0].Bytes) != "f" {
		t.Fatalf("active floating input should route to floating terminal, got %#v", terminal.Inputs)
	}
}

func TestMouseFloatingContentActivationExitsInteractionModeBeforeLiveInput(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	shell = shell.SetInteractionMode(state.InteractionModeFloating)
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: shell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-main", Channel: 1, Cols: 78, Rows: 20, ViewID: state.TerminalPaneViewID(state.DefaultPaneID)}}); err != nil {
		t.Fatalf("post main attach result: %v", err)
	}
	if err := postPreparedLiveAttachResult(runtime, LiveAttachResultMsg{Result: port.TerminalAttachResult{TerminalID: "term-float", Channel: 9, Cols: 28, Rows: 6, ViewID: state.TerminalFloatingViewID("floating-1")}}); err != nil {
		t.Fatalf("post floating attach result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain setup: %v", err)
	}

	frame := lastFrame(t, host.Frames())
	floatingContent := frameActionHitRegion(t, frame, render.ActionFloatingRaise.String(), "floating-pane-1")
	if err := host.SendInput(mouseEventAt(floatingContent.Rect)); err != nil {
		t.Fatalf("click floating: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"}); err != nil {
		t.Fatalf("send floating key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating key: %v", err)
	}

	shell = runtime.State().Shell.EnsureDefaults()
	if shell.ActiveFloatingID() != "floating-1" || shell.InteractionMode != state.InteractionModeNormal {
		t.Fatalf("content click should activate floating input mode, shell=%#v", shell)
	}
	if len(terminal.Inputs) != 1 || terminal.Inputs[0].TerminalID != "term-float" || terminal.Inputs[0].Channel != 9 || string(terminal.Inputs[0].Bytes) != "f" {
		t.Fatalf("floating input should route to clicked floating terminal after mode exit, inputs=%#v", terminal.Inputs)
	}
}

func TestTerminalPoolPaneReattachDoesNotStealSharedFloatingBinding(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{
			TerminalID:   "term-b",
			Channel:      9,
			Cols:         80,
			Rows:         24,
			ResizePolicy: state.TerminalResizeRoleFollower,
			SurfaceID:    "surface",
		},
	}
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-a")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-a"},
		Rect:     state.FloatingRect{X: 8, Y: 4, W: 40, H: 10},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-a", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindFloating(state.NewFloatingTerminalView("floating-1", "floating-pane-1", "term-a", 8, 80, 24, state.TerminalResizeRoleFollower, "surface", state.TerminalFloatingViewID("floating-1"), false))
	reducer := newTerminalPoolReducerPrepared(LiveDeps{Terminal: terminal})

	root, effects := reducer(root, TerminalPoolAttachRequestMsg{TerminalID: "term-b", TargetPaneID: state.DefaultPaneID})
	if len(effects) != 1 {
		t.Fatalf("expected attach effect, got %#v", effects)
	}
	msg, ok := effects[0].(FuncEffect).Run(context.Background()).(TerminalPoolAttachResultMsg)
	if !ok {
		t.Fatalf("expected attach result, got %#v", msg)
	}
	root, _ = reducer(root, msg)

	paneBinding, ok := root.TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || paneBinding.TerminalID != "term-b" || paneBinding.Channel != 9 {
		t.Fatalf("pane should rebind to term-b, binding=%#v ok=%v", paneBinding, ok)
	}
	floatingBinding, ok := root.TerminalViews.FloatingBinding("floating-1")
	if !ok || floatingBinding.TerminalID != "term-a" || floatingBinding.Channel != 8 {
		t.Fatalf("floating binding must remain on term-a, binding=%#v ok=%v", floatingBinding, ok)
	}
	root.Shell, _ = root.Shell.ApplyFloatingCommand(state.FloatingCommand{Action: state.FloatingCommandFocusRaise, TargetID: "floating-1", Source: state.PaneCommandSourceTest})
	target, ok := liveInputTarget(root)
	if !ok || target.TerminalID != "term-a" || target.Channel != 8 || !target.Floating {
		t.Fatalf("focused floating input must keep its own binding, target=%#v ok=%v", target, ok)
	}
}

func TestLiveInputDoesNotFallbackToSessionForEmptyActivePane(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-main", Channel: 1, Cols: 78, Rows: 20}}
	host := NewFakeTerminalHost(8)
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
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellSplitActivePaneMsg{
		Pane:      state.PaneState{ID: "pane-empty", Title: "empty", Kind: state.PaneEmpty},
		Direction: state.SplitDirectionVertical,
	}); err != nil {
		t.Fatalf("split empty pane: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain empty split: %v", err)
	}
	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"}); err != nil {
		t.Fatalf("send empty pane key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain empty pane key: %v", err)
	}

	if len(terminal.Inputs) != 0 {
		t.Fatalf("empty active pane must not receive old session terminal input, got %#v", terminal.Inputs)
	}
	if toasts := runtime.State().Shell.Toasts; len(toasts) == 0 || toasts[len(toasts)-1].Body != "no terminal bound" {
		t.Fatalf("empty active pane should show explicit input state, got %#v", runtime.State().Shell.Toasts)
	}
}

func TestFloatingEmptyPaneAttachesExistingTerminalFromPicker(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{Channel: 9},
		ListResult: port.TerminalListResult{Items: []port.TerminalPoolItem{{
			TerminalID: "term-float",
			Title:      "floating shell",
			State:      "running",
		}, {
			TerminalID: "term-main",
			Title:      "main",
			State:      "running",
		}}},
	}
	shell := state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-main")
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float slot", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 10, Y: 5, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(90, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: shell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-main", Cols: 90, Rows: 24}}); err != nil {
		t.Fatalf("post main attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain main attach: %v", err)
	}
	terminal.Resizes = nil

	emptyAttach := frameActionHitRegion(t, lastFrame(t, host.Frames()), "empty.attach", "floating-pane-1")
	if err := host.SendInput(mouseEventAt(emptyAttach.Rect)); err != nil {
		t.Fatalf("send floating empty attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating empty attach: %v", err)
	}
	if len(terminal.Lists) == 0 || runtime.State().Shell.Overlay.Kind != state.OverlayTerminalPicker || runtime.State().Shell.Overlay.TargetID != "floating-1" {
		t.Fatalf("empty attach should open picker for floating, lists=%#v overlay=%#v", terminal.Lists, runtime.State().Shell.Overlay)
	}
	if !frameContains(lastFrame(t, host.Frames()), "floating shell") || frameContains(lastFrame(t, host.Frames()), "@pool") {
		t.Fatalf("picker should render pool terminal row, got %#v", lastFrame(t, host.Frames()).Lines)
	}

	for _, event := range []input.InputEvent{
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "l"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "o"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"},
		{Kind: input.EventKindKey, Key: input.KeyChar, Char: "t"},
		{Kind: input.EventKindKey, Key: input.KeyEnter},
	} {
		if err := host.SendInput(event); err != nil {
			t.Fatalf("send picker input %#v: %v", event, err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatalf("drain picker input %#v: %v", event, err)
		}
	}

	if len(terminal.Attaches) < 2 {
		t.Fatalf("expected main attach and floating attach, got %#v", terminal.Attaches)
	}
	floatAttach := terminal.Attaches[len(terminal.Attaches)-1]
	if floatAttach.TerminalID != "term-float" || floatAttach.Cols != 28 || floatAttach.Rows != 6 {
		t.Fatalf("floating attach should use floating content rect, got %#v", floatAttach)
	}
	floating := runtime.State().Shell.ActiveFloatings()[0]
	if !floating.Active || floating.Pane.Kind != state.PaneTerminalLive || floating.Pane.TerminalID != "term-float" || runtime.State().Shell.Overlay.Open {
		t.Fatalf("floating should bind selected terminal and close picker, floating=%#v overlay=%#v", floating, runtime.State().Shell.Overlay)
	}
	if len(terminal.Resizes) != 0 {
		t.Fatalf("attach result already matches floating content rect, resize should dedupe, got %#v", terminal.Resizes)
	}

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-float",
		Revision:   1,
		Cols:       28,
		Rows:       6,
		Lines:      []string{"floating ready"},
	}}); err != nil {
		t.Fatalf("post floating surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating surface: %v", err)
	}
	if !frameContains(lastFrame(t, host.Frames()), "floating ready") {
		t.Fatalf("floating should render terminal live content, got %#v", lastFrame(t, host.Frames()).Lines)
	}

	if err := host.SendInput(input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "f"}); err != nil {
		t.Fatalf("send floating key: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating key: %v", err)
	}
	if got := terminal.Inputs[len(terminal.Inputs)-1]; got.TerminalID != "term-float" || got.Channel != 9 || string(got.Bytes) != "f" {
		t.Fatalf("floating input should route to attached terminal, got %#v all=%#v", got, terminal.Inputs)
	}

	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{Action: state.FloatingCommandClose, TargetID: "floating-1", Source: state.PaneCommandSourceTest}}); err != nil {
		t.Fatalf("post floating close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating close: %v", err)
	}
	if len(runtime.State().Shell.ActiveFloatings()) != 0 || len(terminal.Kills) != 0 {
		t.Fatalf("closing floating should remove window without killing terminal, floatings=%#v kills=%#v", runtime.State().Shell.ActiveFloatings(), terminal.Kills)
	}
}

func TestActiveFloatingResizeCommandResizesAttachedTerminalContentRect(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-float", Channel: 5, Cols: 28, Rows: 6}}
	shell := state.DefaultShell()
	var result state.FloatingCommandResult
	shell, result = shell.ApplyFloatingCommand(state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneTerminalLive, TerminalID: "term-float"},
		Rect:     state.FloatingRect{X: 8, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	})
	if result.Status != state.FloatingCommandOK {
		t.Fatalf("create floating: %#v", result)
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(90, 24)
	runtime := NewInteractiveRuntime(
		state.Root{Shell: shell},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)
	if err := postPreparedTerminalPoolAttachResult(runtime, TerminalPoolAttachResultMsg{
		TerminalID:       "term-float",
		TargetFloatingID: "floating-1",
		Result:           port.TerminalAttachResult{TerminalID: "term-float", Channel: 5, Cols: 28, Rows: 6, ResizePolicy: state.TerminalResizeRoleOwner, CanResize: true},
	}); err != nil {
		t.Fatalf("post floating attach result: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating attach result: %v", err)
	}
	terminal.Resizes = nil

	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action:   state.FloatingCommandResize,
		TargetID: "floating-1",
		DeltaW:   4,
		DeltaH:   2,
		Source:   state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post floating resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating resize: %v", err)
	}
	if len(terminal.Resizes) != 1 {
		t.Fatalf("active floating resize should emit one terminal resize, got %#v", terminal.Resizes)
	}
	if got := terminal.Resizes[0]; got.TerminalID != "term-float" || got.Channel != 5 || got.Cols != 32 || got.Rows != 8 {
		t.Fatalf("floating resize should use updated content rect, got %#v", got)
	}
}

func TestLiveAppAttachSwitchClearsStaleSurfaceRows(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-new", Channel: 5, Cols: 78, Rows: 20},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{
			Surface: state.TerminalSurfaceStore{
				TerminalID: "term-old",
				Ready:      true,
				Lines:      []string{"old terminal output"},
			},
		},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-new", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "old terminal output") {
		t.Fatalf("new attach must not render stale live rows, got %#v", frame.Lines)
	}
	if frameContains(frame, "old terminal output") || !frameContains(frame, "live surface empty") || !runtime.State().Surface.Ready {
		t.Fatalf("expected empty ready surface after terminal switch, frame=%#v state=%#v", frame.Lines, runtime.State().Surface)
	}
}

func TestLiveAppAttachHydratesReadySurfaceFromService(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8, Cols: 78, Rows: 20},
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Cols:       78,
				Rows:       20,
				Lines:      []string{"alpha", "beta 你好🚀"},
				Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 8, Shape: "bar"},
			},
		},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	if len(terminal.Surfaces) != 1 || terminal.Surfaces[0].TerminalID != "term-1" || terminal.Surfaces[0].Rows != 20 {
		t.Fatalf("expected live surface request after attach, got %#v", terminal.Surfaces)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "alpha") || !frameContains(frame, "beta 你好🚀") || !frame.Cursor.Visible || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("expected hydrated live surface in frame, lines=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestLiveAppAttachClearsPendingForEmptySurfaceSnapshot(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-blank", Channel: 8, Cols: 78, Rows: 20},
		SurfaceResult: port.TerminalSurfaceResult{
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-blank",
				Cols:       78,
				Rows:       20,
			},
		},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-blank", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	if !runtime.State().Surface.Ready {
		t.Fatalf("empty snapshot should still mark live surface ready: %#v", runtime.State().Surface)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "live surface pending") {
		t.Fatalf("empty ready snapshot must clear pending fallback, frame=%#v", frame.Lines)
	}
}

func TestLiveRuntimeConsumesBackendLiveEventsAndRedraws(t *testing.T) {
	liveEvents := make(chan port.TerminalSurfaceResult, 2)
	terminal := &testkit.FakeTerminalService{
		AttachResult:     port.TerminalAttachResult{TerminalID: "term-1", Channel: 8, Cols: 78, Rows: 20},
		LiveScreenNextCh: liveEvents,
		SurfaceResult: port.TerminalSurfaceResult{
			Ready: true,
			Snapshot: state.LiveSurfaceSnapshot{
				TerminalID: "term-1",
				Cols:       78,
				Rows:       20,
				Lines:      []string{"backend live update"},
			},
		},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewAsyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := waitForLiveScreenNextRequest(context.Background(), runtime, terminal, "term-1"); err != nil {
		t.Fatal(err)
	}
	liveEvents <- port.TerminalSurfaceResult{
		Ready: true,
		Snapshot: state.LiveSurfaceSnapshot{
			TerminalID: "term-1", Revision: 7, Cols: 78, Rows: 20, Lines: []string{"backend live update"}, FullReplace: true,
		},
	}
	if err := drainUntilFrameContains(context.Background(), runtime, host, "backend live update"); err != nil {
		t.Fatal(err)
	}
}

func TestLiveEventUsesEventTerminalIDWhenSnapshotTerminalIDMissing(t *testing.T) {
	root := state.Root{
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-main",
			Revision:   2,
			Cols:       80,
			Rows:       24,
			Lines:      []string{"main"},
		}),
	}
	reducer := newLiveReducerPrepared(LiveDeps{})
	next, _ := reducer(root, LiveEventMsg{Event: port.TerminalLiveEvent{
		TerminalID: "term-logs",
		Ready:      true,
		Snapshot: state.LiveSurfaceSnapshot{
			Revision: 1,
			Cols:     40,
			Rows:     12,
			Lines:    []string{"logs"},
		},
	}})

	if got := next.Surface.SurfaceForTerminal("term-main").Lines[0]; got != "main" {
		t.Fatalf("event for another terminal must not overwrite current projection, got %q", got)
	}
	logs := next.Surface.SurfaceForTerminal("term-logs")
	if logs.TerminalID != "term-logs" || logs.Lines[0] != "logs" || logs.Revision != 1 {
		t.Fatalf("expected event terminal id to be applied to snapshot, got %#v", logs)
	}
}

func TestLiveSurfaceStoreKeepsPaneTerminalBindingsIsolated(t *testing.T) {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0].TerminalID = "term-main"
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "logs", Kind: state.PaneTerminalLive, TerminalID: "term-logs"}, state.SplitDirectionVertical).
		FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID})
	host := NewFakeTerminalHost(8)
	host.SetSize(100, 24)
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-main", 7, 48, 20, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true)).
		BindPane(state.NewPaneTerminalView("pane-2", "term-logs", 8, 48, 20, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID("pane-2"), false))
	runtime := NewLiveRuntime(
		root,
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
	)

	for _, msg := range []Msg{
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-main", Cols: 48, Rows: 20, Lines: []string{"main-only"}}},
		LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-logs", Cols: 48, Rows: 20, Lines: []string{"logs-only"}}},
		NoopMsg{},
	} {
		if err := runtime.Post(msg); err != nil {
			t.Fatalf("post %T: %v", msg, err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "main-only") || !frameContains(frame, "logs-only") {
		t.Fatalf("expected both pane-bound live surfaces, got %#v", frame.Lines)
	}

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{TerminalID: "term-old", Cols: 48, Rows: 20, Lines: []string{"old-stale"}}}); err != nil {
		t.Fatalf("post stale surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain stale surface: %v", err)
	}
	frame = lastFrame(t, host.Frames())
	if frameContains(frame, "old-stale") {
		t.Fatalf("unbound old terminal update must not render into active panes, got %#v", frame.Lines)
	}
	if got := runtime.State().Surface.SurfaceForTerminal("term-main").Lines[0]; got != "main-only" {
		t.Fatalf("old terminal update polluted main binding, got %q", got)
	}
}

func TestLiveContentRendererKeepsStyleCursorAndChromeSafe(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 9, Cols: 28, Rows: 10},
	}
	host := NewFakeTerminalHost(8)
	host.SetSize(30, 12)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 30, Rows: 12}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Cols:       28,
		Rows:       8,
		Lines: []string{
			"\x1b[31mERR\x1b[0m 你好🚀 output that must clip before chrome",
			"\x1b[32mOK\x1b[0m done",
		},
		Cursor: state.LiveCursor{Visible: true, Row: 1, Col: 7, Shape: "bar"},
	}}); err != nil {
		t.Fatalf("post surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain surface: %v", err)
	}

	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "ERR 你好🚀") || !frameContains(frame, "OK done") {
		t.Fatalf("expected live content in pane content rect, got %#v", frame.Lines)
	}
	if frameContains(frame, "\x1b[") {
		t.Fatalf("plain frame must not leak raw ANSI, got %#v", frame.Lines)
	}
	assertPaneVisualState(t, frame, "ERR", render.StyleDanger)
	assertPaneVisualState(t, frame, "OK", render.StyleSuccess)
	if !ansiFrameContains(frame, "\x1b[38;2;") {
		t.Fatalf("ANSI frame must contain SGR styled live content, got %#v", frame.ANSILines)
	}
	if !frame.Cursor.Visible || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("expected live content cursor metadata, got %#v", frame.Cursor)
	}
	for i, line := range frame.Lines {
		if width := render.DisplayWidth(line); width != 30 {
			t.Fatalf("row %d width=%d want=30 line=%q", i, width, line)
		}
	}
	if right := render.SliceCells(frame.Lines[2], 29, 30); right != "│" {
		t.Fatalf("right pane border must survive live ANSI/wide clipping, got %q in %#v", right, frame.Lines)
	}
}

func TestLiveAttachUsesCardContentRectForInitialTerminalSize(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 7}}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Attaches) != 1 {
		t.Fatalf("expected one attach, got %#v", terminal.Attaches)
	}
	if got := terminal.Attaches[0]; got.Cols != 78 || got.Rows != 20 {
		t.Fatalf("attach must use card content rect, got %#v", got)
	}
}

func TestAttachResultWithExistingSizeIsCorrectedToContentRect(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 7, Cols: 80, Rows: 24}}
	host := NewFakeTerminalHost(8)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if got := terminal.Attaches[0]; got.Cols != 78 || got.Rows != 20 {
		t.Fatalf("attach request must use content rect, got %#v", got)
	}
	if len(terminal.Resizes) != 1 {
		t.Fatalf("expected attach result correction resize, got %#v", terminal.Resizes)
	}
	if got := terminal.Resizes[0]; got.Cols != 78 || got.Rows != 20 {
		t.Fatalf("resize correction must use content rect, got %#v", got)
	}
}

func TestHostResizeUsesActiveContentRectAndDeduplicates(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send duplicate resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize: %v", err)
	}

	if len(terminal.Resizes) != 1 {
		t.Fatalf("expected one deduplicated content resize, got %#v", terminal.Resizes)
	}
	if got := terminal.Resizes[0]; got.Cols != 98 || got.Rows != 36 {
		t.Fatalf("host resize must use card content rect, got %#v", got)
	}
}

func TestLiveResizeKeepsLatestContentRectAndIgnoresOldSizeSurface(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   1,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"before resize"},
	}}); err != nil {
		t.Fatalf("post initial surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain initial surface: %v", err)
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send host resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain host resize: %v", err)
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 98 || got.Rows != 36 {
		t.Fatalf("host resize must use latest content rect, got %#v", got)
	}
	if runtime.State().Surface.Cols != 98 || runtime.State().Surface.Rows != 36 {
		t.Fatalf("surface resize boundary should project latest content rect, got %#v", runtime.State().Surface)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       78,
		Rows:       20,
		Lines:      []string{"late old size"},
	}}); err != nil {
		t.Fatalf("post old-size surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain old-size surface: %v", err)
	}
	oldSizeFrame := lastFrame(t, host.Frames())
	if frameContains(oldSizeFrame, "late old size") || runtime.State().Surface.Cols != 98 || runtime.State().Surface.Rows != 36 {
		t.Fatalf("late old-size surface must not roll back resized frame/state, frame=%#v state=%#v", oldSizeFrame.Lines, runtime.State().Surface)
	}

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       98,
		Rows:       36,
		Lines:      []string{"after resize"},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 5, Shape: "bar"},
		Modes:      state.LiveTerminalModes{MouseTracking: true, MouseSGR: true},
	}}); err != nil {
		t.Fatalf("post resized surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resized surface: %v", err)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "after resize") || frameContains(frame, "late old size") {
		t.Fatalf("expected resized live surface only, got %#v", frame.Lines)
	}
	if runtime.State().Surface.Cols != 98 || runtime.State().Surface.Rows != 36 || runtime.State().Surface.ResizeBoundary.Active {
		t.Fatalf("matching resized surface should clear resize boundary, got %#v", runtime.State().Surface)
	}
	if !frame.Cursor.Visible || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("resized live surface should preserve cursor, got %#v", frame.Cursor)
	}
	vm := render.NewRenderVMBuilder().Build(runtime.State())
	panel, ok := activePanelVMForAppTest(vm.Shell)
	if !ok {
		t.Fatalf("expected active panel VM, got %#v", vm.Shell.Layout.Panels)
	}
	if panel.Content.Extent != (render.ContentExtent{Known: true, Cols: 98, Rows: 36}) {
		t.Fatalf("active content should expose resized live extent, got %#v", panel.Content.Extent)
	}
	layout := render.MeasureLayout(vm.Shell, vm.Shell.Layout.Viewport)
	contentRect, ok := activeContentRectForAppTest(layout)
	if !ok || contentRect.W != 98 || contentRect.H != 36 {
		t.Fatalf("layout should allocate resized active content rect, rect=%#v ok=%v layout=%#v", contentRect, ok, layout)
	}
	wantCursorRect := render.Rect{X: contentRect.X + 5, Y: contentRect.Y + 1, W: 1, H: 1}
	if frame.CursorRect != wantCursorRect || layout.CursorRect != wantCursorRect {
		t.Fatalf("cursor should stay content-local after resize, frame=%#v layout=%#v want=%#v", frame.CursorRect, layout.CursorRect, wantCursorRect)
	}
	if !runtime.State().Surface.Modes.MousePassthroughEnabled() {
		t.Fatalf("resized live surface should preserve mouse modes, got %#v", runtime.State().Surface.Modes)
	}
	for i, line := range frame.Lines {
		if width := render.DisplayWidth(line); width != 100 {
			t.Fatalf("resized frame row %d width=%d want=100 line=%q", i, width, line)
		}
	}
}

func TestLiveResizeFallbacksDoNotUseResizeBoundaryDots(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send host resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain host resize: %v", err)
	}

	emptyFrame := lastFrame(t, host.Frames())
	if !frameContains(emptyFrame, "live surface empty") {
		t.Fatalf("expected empty fallback after resize without surface, got %#v", emptyFrame.Lines)
	}
	emptyLayer, ok := firstPanelLayerForAppTest(render.NewRenderer(render.DefaultTheme()).RenderResult(render.NewRenderVMBuilder().Build(runtime.State())))
	if !ok {
		t.Fatalf("expected empty panel layer")
	}
	if panelLayerContainsPlainForAppTest(emptyLayer, "·") {
		t.Fatalf("empty fallback should not be filled by resize-boundary dots, got %#v", emptyLayer.Lines)
	}

	if err := runtime.Post(LiveEventMsg{Event: port.TerminalLiveEvent{TerminalID: "term-1", Exited: true, ExitCode: 0, Reason: "exited"}}); err != nil {
		t.Fatalf("post exit: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain exit: %v", err)
	}
	exitFrame := lastFrame(t, host.Frames())
	if !frameContains(exitFrame, "terminal exited: term-1 code:0") {
		t.Fatalf("expected exited fallback after resize, got %#v", exitFrame.Lines)
	}
	exitLayer, ok := firstPanelLayerForAppTest(render.NewRenderer(render.DefaultTheme()).RenderResult(render.NewRenderVMBuilder().Build(runtime.State())))
	if !ok {
		t.Fatalf("expected exited panel layer")
	}
	if panelLayerContainsPlainForAppTest(exitLayer, "·") {
		t.Fatalf("exited fallback should not be filled by resize-boundary dots, got %#v", exitLayer.Lines)
	}
}

func TestLiveSurfaceProtocolTerminalExitedIsNotErrorUI(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5},
		SurfaceErr:   errors.New("protocol error 400: terminal exited"),
	}
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 40)
	runtime := NewLiveRuntime(
		state.Root{Surface: state.TerminalSurfaceStore{TerminalID: "term-1", Lines: []string{"last output"}}},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 100, 40)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}

	root := runtime.State()
	if root.Session.LastError != "" || root.Surface.Err != "" {
		t.Fatalf("terminal exited should not be stored as live error, session=%q surface=%q", root.Session.LastError, root.Surface.Err)
	}
	if root.Session.State != state.TerminalLiveExited || root.Surface.State != state.TerminalLiveExited {
		t.Fatalf("expected exited state, session=%s surface=%s", root.Session.State, root.Surface.State)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "last output") ||
		!frameContains(frame, "► restart ◄") ||
		!frameContains(frame, "[ reconnect ]") ||
		frameContains(frame, "protocol error 400") {
		t.Fatalf("expected exited content with restart hints and no protocol error, got %#v", frame.Lines)
	}
}

func TestLiveSurfaceAuthoritativeRunningClearsExitedSessionAndSurface(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 40)
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID,
			"term-1",
			7,
			80,
			24,
			state.TerminalResizeRoleOwner,
			"surface-1",
			state.TerminalPaneViewID(state.DefaultPaneID),
			true,
		)),
		Session: state.TerminalSessionStore{}.
			Attach("term-1", 7, 80, 24).
			MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-1",
			Revision:   9,
			Cols:       80,
			Rows:       24,
			Lines:      []string{"terminal exited: term-1 code:0 exited"},
		}).MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
	}
	runtime := NewLiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited", "% "},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
		State:      state.TerminalLiveAttached,
	}, LifecycleKnown: true}); err != nil {
		t.Fatalf("post running surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain running surface: %v", err)
	}

	final := runtime.State()
	if final.Surface.State != state.TerminalLiveAttached || final.Session.State == state.TerminalLiveExited || final.Session.ExitReason != "" || final.Surface.ExitReason != "" {
		t.Fatalf("authoritative running surface should clear stale exited session/surface, session=%#v surface=%#v", final.Session, final.Surface)
	}
	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "% ") || frameContains(frame, "restart") {
		t.Fatalf("running terminal with old exit marker should render live prompt without restart CTA, got %#v", frame.Lines)
	}
	if !frame.Cursor.Visible || frame.Cursor.Shape != render.CursorShapeBar {
		t.Fatalf("running surface should project core cursor, got %#v", frame.Cursor)
	}
}

func TestLiveQueueKeepsAuthoritativeRunningLifecycleWhenOrdinaryFrameFollows(t *testing.T) {
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 40)
	root := state.Root{
		Shell: state.DefaultShell().BindPaneTerminal(state.PaneCommandTarget{PaneID: state.DefaultPaneID}, "term-1"),
		TerminalViews: state.TerminalViewStore{}.BindPane(state.NewPaneTerminalView(
			state.DefaultPaneID,
			"term-1",
			7,
			80,
			24,
			state.TerminalResizeRoleOwner,
			"surface-1",
			state.TerminalPaneViewID(state.DefaultPaneID),
			true,
		)),
		Session: state.TerminalSessionStore{}.
			Attach("term-1", 7, 80, 24).
			MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
		Surface: (state.TerminalSurfaceStore{}).ApplySnapshot(state.LiveSurfaceSnapshot{
			TerminalID: "term-1",
			Revision:   9,
			Cols:       80,
			Rows:       24,
			Lines:      []string{"terminal exited: term-1 code:0 exited"},
		}).MarkExitedWithMetadata("term-1", 0, "exited", time.Date(2026, 6, 17, 12, 45, 0, 0, time.UTC), []string{"/bin/zsh"}),
	}
	runtime := NewLiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: &testkit.FakeTerminalService{}})

	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   3,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited", "% "},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
		State:      state.TerminalLiveAttached,
	}, LifecycleKnown: true}); err != nil {
		t.Fatalf("post authoritative running surface: %v", err)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   4,
		Cols:       80,
		Rows:       24,
		Lines:      []string{"terminal exited: term-1 code:0 exited", "% "},
		Cursor:     state.LiveCursor{Visible: true, Row: 1, Col: 2, Shape: "bar"},
		State:      state.TerminalLiveAttached,
	}}); err != nil {
		t.Fatalf("post ordinary surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain surfaces: %v", err)
	}

	final := runtime.State()
	if final.Surface.State != state.TerminalLiveAttached || final.Session.State == state.TerminalLiveExited {
		t.Fatalf("authoritative running lifecycle must survive live queue coalescing, session=%#v surface=%#v", final.Session, final.Surface)
	}
	frame := lastFrame(t, host.Frames())
	if frameContains(frame, "restart") || !frameContains(frame, "% ") || !frame.Cursor.Visible {
		t.Fatalf("running lifecycle should remove restart CTA after queued ordinary frame, lines=%#v cursor=%#v", frame.Lines, frame.Cursor)
	}
}

func TestLiveResizeOverflowMarkersStayOnChrome(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5}}
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 40)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 100, 40)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := host.SendResize(80, 24); err != nil {
		t.Fatalf("send shrink resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain shrink resize: %v", err)
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 78 || got.Rows != 20 {
		t.Fatalf("shrunk viewport should resize terminal to content rect, got %#v", got)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       120,
		Rows:       30,
		Lines:      []string{"terminal output should clip along right edge after resize"},
	}}); err != nil {
		t.Fatalf("post oversized surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain oversized surface: %v", err)
	}

	frame := lastFrame(t, host.Frames())
	if !frameContains(frame, "terminal output should clip") {
		t.Fatalf("expected oversized live surface content, got %#v", frame.Lines)
	}
	result := render.NewRenderer(render.DefaultTheme()).RenderResult(render.NewRenderVMBuilder().Build(runtime.State()))
	panelLayer, ok := firstPanelLayerForAppTest(result)
	if !ok || panelLayer.ContentOverflow != (render.ContentOverflow{Right: true, Bottom: true}) {
		t.Fatalf("live resize mismatch should expose chrome overflow, layer=%#v ok=%v", panelLayer, ok)
	}
	rightMarkerRow := panelLayer.Rect.Y + panelLayer.Rect.H - 2
	rightMarkerCol := panelLayer.Rect.X + panelLayer.Rect.W - 1
	if got := render.SliceCells(frame.Lines[rightMarkerRow], rightMarkerCol, rightMarkerCol+1); got != ">" {
		t.Fatalf("right overflow marker should be shown for live resize mismatch, got %q frame=%#v", got, frame.Lines)
	}
	bottomMarkerRow := panelLayer.Rect.Y + panelLayer.Rect.H - 1
	bottomMarkerCol := panelLayer.Rect.X + panelLayer.Rect.W - 2
	if got := render.SliceCells(frame.Lines[bottomMarkerRow], bottomMarkerCol, bottomMarkerCol+1); got != "v" {
		t.Fatalf("bottom overflow marker should be shown for live resize mismatch, got %q frame=%#v", got, frame.Lines)
	}
	cornerCol := panelLayer.Rect.X + panelLayer.Rect.W - 1
	if got := render.SliceCells(frame.Lines[bottomMarkerRow], cornerCol, cornerCol+1); got != "┘" {
		t.Fatalf("overflow marker should keep pane corner, got %q frame=%#v", got, frame.Lines)
	}
	for _, line := range panelLayer.Lines {
		if strings.Contains(line.PlainString(), ">") || strings.Contains(line.PlainString(), "v") {
			t.Fatalf("overflow markers must stay out of panel content layer, got %#v", panelLayer.Lines)
		}
	}
	for i, line := range frame.Lines {
		if width := render.DisplayWidth(line); width != 80 {
			t.Fatalf("shrunk frame row %d width=%d want=80 line=%q", i, width, line)
		}
	}
}

func TestHostResizeUsesBusinessActivePaneWhenFloatingOwnsVisualFocus(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 5}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellFloatingCommandMsg{Command: state.FloatingCommand{
		Action:   state.FloatingCommandCreate,
		TargetID: "floating-1",
		Pane:     state.PaneState{ID: "floating-pane-1", Title: "float", Kind: state.PaneEmpty},
		Rect:     state.FloatingRect{X: 10, Y: 4, W: 30, H: 8},
		Source:   state.PaneCommandSourceTest,
	}}); err != nil {
		t.Fatalf("post floating: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain floating: %v", err)
	}
	if !runtime.State().Shell.ActiveFloatings()[0].Active {
		t.Fatalf("test expects active floating, got %#v", runtime.State().Shell.ActiveFloatings())
	}
	if err := host.SendResize(100, 40); err != nil {
		t.Fatalf("send resize: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain resize: %v", err)
	}

	if len(terminal.Resizes) == 0 {
		t.Fatalf("floating visual focus must not block business active pane resize")
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 98 || got.Rows != 36 {
		t.Fatalf("resize should still use tiled business active pane content rect, got %#v all=%#v", got, terminal.Resizes)
	}
}

func TestHeaderFooterHideResizesTerminalWithReclaimedContentRows(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 6}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	terminal.Resizes = nil
	if err := runtime.Post(ShellSetHeaderVisibleMsg{Visible: false}); err != nil {
		t.Fatalf("post header hide: %v", err)
	}
	if err := runtime.Post(ShellSetFooterVisibleMsg{Visible: false}); err != nil {
		t.Fatalf("post footer hide: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain chrome resize: %v", err)
	}

	if len(terminal.Resizes) < 2 {
		t.Fatalf("header/footer chrome changes must drive at least the intermediate and final PTY resize, got %#v", terminal.Resizes)
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 78 || got.Rows != 22 {
		t.Fatalf("hidden header/footer must reclaim content rows, got %#v", got)
	}
}

func TestSplitPresentationUsesSplitContentRectForResize(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split presentation: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if len(terminal.Resizes) != 0 {
		t.Fatalf("single split-line pane now shares the same card content rect as card, got %#v", terminal.Resizes)
	}
}

func TestVerticalSplitActivePaneReservesDividerCellForResize(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split presentation: %v", err)
	}
	if err := runtime.Post(ShellSplitActivePaneMsg{
		Pane:      state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive},
		Direction: state.SplitDirectionVertical,
	}); err != nil {
		t.Fatalf("post split active pane: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split: %v", err)
	}

	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 39 || got.Rows != 20 || got.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) {
		t.Fatalf("split should resize existing owner pane rather than follower split pane, got %#v", got)
	}
}

func TestNestedEmptySplitResizesOwnerTerminalViewContentRect(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8}}
	host := NewFakeTerminalHost(16)
	host.SetSize(140, 36)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 140, 36)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split presentation: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	terminal.Resizes = nil

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-right", Title: "pane", Kind: state.PaneEmpty},
	}}); err != nil {
		t.Fatalf("post right split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain right split: %v", err)
	}
	rightSplitResize := terminal.Resizes[len(terminal.Resizes)-1]

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		Target:         state.PaneCommandTarget{PaneID: "pane-right"},
		SplitDirection: state.SplitDirectionHorizontal,
		NewPane:        state.PaneState{ID: "pane-right-bottom", Title: "pane", Kind: state.PaneEmpty},
	}}); err != nil {
		t.Fatalf("post lower right split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain lower right split: %v", err)
	}

	if len(terminal.Resizes) != 1 {
		t.Fatalf("splitting empty right pane must not resize unchanged left owner again, before=%#v all=%#v", rightSplitResize, terminal.Resizes)
	}
	if got := terminal.Resizes[0]; got.TerminalID != "term-1" || got.ViewID != state.TerminalPaneViewID(state.DefaultPaneID) || got.Cols <= 0 || got.Rows <= 0 {
		t.Fatalf("right split should resize original owner terminal view, got %#v", got)
	}
	if err := runtime.Post(LiveSurfaceMsg{Snapshot: state.LiveSurfaceSnapshot{
		TerminalID: "term-1",
		Revision:   2,
		Cols:       terminal.Resizes[0].Cols,
		Rows:       terminal.Resizes[0].Rows,
		Lines:      []string{"owner terminal content"},
	}}); err != nil {
		t.Fatalf("post owner-sized surface: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain owner-sized surface: %v", err)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.DesiredCols != terminal.Resizes[0].Cols || binding.DesiredRows != terminal.Resizes[0].Rows {
		t.Fatalf("owner view desired size should track left content rect, binding=%#v resize=%#v", binding, terminal.Resizes[0])
	}
	if _, ok := runtime.State().TerminalViews.PaneBinding("pane-right"); ok {
		t.Fatalf("empty right pane must not create terminal binding")
	}
	vm := render.NewRenderVMBuilder().Build(runtime.State())
	plan := render.MeasureLayout(vm.Shell, vm.Shell.Layout.Viewport)
	result := render.NewRenderer(render.DefaultTheme()).RenderResult(vm)
	ownerLayer, ok := panelLayerByPaneIDForAppTest(result, plan, state.DefaultPaneID)
	if !ok {
		t.Fatalf("expected owner pane layer")
	}
	if ownerLayer.ContentOverflow != (render.ContentOverflow{}) {
		t.Fatalf("owner pane should not overflow after 211 empty split, got %#v", ownerLayer.ContentOverflow)
	}
	if panelLayerContainsPlainForAppTest(ownerLayer, "·") {
		t.Fatalf("owner pane should not show resize-boundary dots after 211 empty split, got %#v", ownerLayer.Lines)
	}
}

func TestPaneSizeCommandResizesActiveTerminalContentRect(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8}}
	host := NewFakeTerminalHost(16)
	host.SetSize(80, 24)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 80, 24)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post split presentation: %v", err)
	}
	if err := runtime.Post(ShellSplitActivePaneMsg{
		Pane:      state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive},
		Direction: state.SplitDirectionVertical,
	}); err != nil {
		t.Fatalf("post split active pane: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split: %v", err)
	}
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 9, 39, 20, state.TerminalResizeRoleOwner, "test-surface", state.TerminalPaneViewID("pane-2"), true)).TransferPaneResizeOwner("pane-2")
	terminal.Resizes = nil
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:   state.PaneCommandSetSize,
		Target:   state.PaneCommandTarget{PaneID: "pane-2"},
		SizeMode: state.PaneSizeCells,
		Cols:     24,
	}}); err != nil {
		t.Fatalf("post fixed size command: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain size command: %v", err)
	}

	if len(terminal.Resizes) == 0 {
		t.Fatalf("expected explicit pane owner resize request")
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 22 || got.Rows != 20 {
		t.Fatalf("fixed right pane size must drive active content resize, got %#v all=%#v", got, terminal.Resizes)
	}
}

func TestBatchedPaneCommandsResizeTerminalToLatestContentRect(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8}}
	host := NewFakeTerminalHost(16)
	host.SetSize(100, 40)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 100, 40)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive},
	}}); err != nil {
		t.Fatalf("post split command: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split command: %v", err)
	}
	owner, _ := runtime.state.TerminalViews.PaneBinding(state.DefaultPaneID)
	binding := state.NewPaneTerminalView("pane-2", "term-1", 9, 48, 36, state.TerminalResizeRoleOwner, "test-surface", state.TerminalPaneViewID("pane-2"), true)
	binding.Session = owner.AttachmentSession()
	binding.OperationID = owner.OperationID
	runtime.state.TerminalViews = runtime.state.TerminalViews.BindPane(binding).TransferPaneResizeOwner("pane-2")
	terminal.Resizes = nil
	for _, command := range []state.PaneCommand{
		{Action: state.PaneCommandResize, Target: state.PaneCommandTarget{PaneID: "pane-2"}, ResizeDirection: state.PaneResizeLeft, Delta: 6},
		{Action: state.PaneCommandZoom, Target: state.PaneCommandTarget{PaneID: "pane-2"}},
	} {
		if err := runtime.Post(ShellPaneCommandMsg{Command: command}); err != nil {
			t.Fatalf("post pane command: %v", err)
		}
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain pane commands: %v", err)
	}

	if len(terminal.Resizes) < 2 {
		t.Fatalf("expected split and zoom resize requests, got %#v", terminal.Resizes)
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 98 || got.Rows != 36 {
		t.Fatalf("latest zoomed pane content rect must win, got %#v all=%#v", got, terminal.Resizes)
	}
	if runtime.State().Session.Cols != 98 || runtime.State().Session.Rows != 36 {
		t.Fatalf("stale split resize result must not override latest session size, state=%#v", runtime.State().Session)
	}
}

func TestClosePaneTransfersResizeOwnerAndRestoresFullContentRect(t *testing.T) {
	terminal := &testkit.FakeTerminalService{AttachResult: port.TerminalAttachResult{TerminalID: "term-1", Channel: 8}}
	host := NewFakeTerminalHost(16)
	host.SetSize(120, 40)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: ownerLiveAttachConfig("term-1", 120, 40)}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain attach: %v", err)
	}
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action:         state.PaneCommandSplit,
		SplitDirection: state.SplitDirectionVertical,
		NewPane:        state.PaneState{ID: "pane-2", Title: "right", Kind: state.PaneTerminalLive},
	}}); err != nil {
		t.Fatalf("post split: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain split: %v", err)
	}
	terminal.Resizes = nil
	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action: state.PaneCommandClose,
		Target: state.PaneCommandTarget{PaneID: "pane-2"},
	}}); err != nil {
		t.Fatalf("post close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain close: %v", err)
	}

	if len(terminal.Resizes) == 0 {
		t.Fatalf("close pane should trigger owner resize to restored content rect")
	}
	if got := terminal.Resizes[len(terminal.Resizes)-1]; got.Cols != 118 || got.Rows != 36 {
		t.Fatalf("close pane should restore full content rect, got %#v all=%#v", got, terminal.Resizes)
	}
	if binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID); !ok || binding.ResizeRole != state.TerminalResizeRoleOwner || !binding.CanResize {
		t.Fatalf("remaining pane should own resize after sibling close, binding=%#v ok=%v", binding, ok)
	}
}

func TestCloseResizeOwnerPromotesFollowerAndResizesImmediately(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	host.SetSize(122, 28)
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	root := state.Root{
		Shell:    shell.FocusPane(state.PaneCommandTarget{PaneID: state.DefaultPaneID}),
		Viewport: state.ViewportStore{Valid: true, Cols: 122, Rows: 28},
		Session:  state.TerminalSessionStore{TerminalID: "term-1", Channel: 7, Attached: true, Cols: 59, Rows: 24, DesiredCols: 59, DesiredRows: 24, ResizePolicy: state.TerminalResizeRoleOwner, SurfaceID: "surface", ViewID: "view-1"},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-1", 7, 59, 24, state.TerminalResizeRoleOwner, "surface", "view-1", true))
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView("pane-2", "term-1", 8, 59, 24, state.TerminalResizeRoleFollower, "surface", "view-2", false))
	runtime := NewLiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal})

	if err := runtime.Post(ShellPaneCommandMsg{Command: state.PaneCommand{
		Action: state.PaneCommandClose,
		Target: state.PaneCommandTarget{PaneID: state.DefaultPaneID},
	}}); err != nil {
		t.Fatalf("post close owner: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain close owner: %v", err)
	}

	if len(terminal.Resizes) != 1 {
		t.Fatalf("promoted follower should resize immediately once, got %#v", terminal.Resizes)
	}
	if len(terminal.Detaches) != 1 {
		t.Fatalf("closing owner pane should detach the closed attachment, got %#v", terminal.Detaches)
	}
	if got := terminal.Detaches[0]; got.TerminalID != "term-1" || got.Channel != 7 || got.SurfaceID != "surface" || got.ViewID != "view-1" {
		t.Fatalf("detach should use closed owner attachment identity, got %#v", got)
	}
	if got := terminal.Resizes[0]; got.TerminalID != "term-1" || got.Channel != 8 || got.ViewID != "view-2" || got.ResizePolicy != state.TerminalResizeRoleOwner || got.Cols != 120 || got.Rows != 24 {
		t.Fatalf("promoted follower resize should use new owner content rect and channel, got %#v", got)
	}
	binding, ok := runtime.State().TerminalViews.PaneBinding("pane-2")
	if !ok || binding.ResizeRole != state.TerminalResizeRoleOwner || !binding.CanResize || binding.ResizePending || binding.RequestSeq != 1 {
		t.Fatalf("remaining pane should own resize with completed pending check, binding=%#v ok=%v", binding, ok)
	}
}

func TestClosingTabDetachesAllTerminalViewAttachments(t *testing.T) {
	terminal := &testkit.FakeTerminalService{}
	host := NewFakeTerminalHost(16)
	shell := state.DefaultShell().SetPanelPresentation(state.PanelPresentationSplitLine)
	shell, _ = shell.ApplyWorkbenchCommand(state.WorkbenchCommand{Action: state.WorkbenchCommandTabCreate, Name: "logs"})
	shell = shell.SplitActivePane(state.PaneState{ID: "pane-logs-2", Title: "two", Kind: state.PaneTerminalLive, TerminalID: "term-1"}, state.SplitDirectionVertical)
	tabPanes := panesForWorkbenchTarget(shell, "tab-2")
	if len(tabPanes) != 2 {
		t.Fatalf("expected test tab to contain two panes, got %#v", tabPanes)
	}
	root := state.Root{Shell: shell}
	root.TerminalViews = root.TerminalViews.
		BindPane(state.NewPaneTerminalView(tabPanes[0].ID, "term-1", 7, 80, 24, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(tabPanes[0].ID), true)).
		BindPane(state.NewPaneTerminalView(tabPanes[1].ID, "term-1", 8, 40, 12, state.TerminalResizeRoleFollower, "surface", state.TerminalPaneViewID(tabPanes[1].ID), false))
	runtime := NewLiveRuntime(root, host, NewSyncEffectRunner(), LiveDeps{Terminal: terminal})

	if err := runtime.Post(ShellWorkbenchCommandMsg{Command: state.WorkbenchCommand{Action: state.WorkbenchCommandTabClose, TargetID: "tab-2"}}); err != nil {
		t.Fatalf("post tab close: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain tab close: %v", err)
	}

	if len(terminal.Detaches) != 2 {
		t.Fatalf("closing tab should detach every pane attachment, got %#v", terminal.Detaches)
	}
	channels := map[uint16]bool{}
	for _, detach := range terminal.Detaches {
		channels[detach.Channel] = true
		if detach.TerminalID != "term-1" || detach.SurfaceID != "surface" || detach.ViewID == "" {
			t.Fatalf("detach should keep terminal attachment identity, got %#v", detach)
		}
	}
	if !channels[7] || !channels[8] {
		t.Fatalf("closing tab should detach both attachment channels, got %#v", terminal.Detaches)
	}
}

func TestLiveAppShowsTerminalServiceError(t *testing.T) {
	terminal := &testkit.FakeTerminalService{
		AttachErr: errors.New("attach failed"),
	}
	host := NewFakeTerminalHost(4)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: terminal},
	)

	if err := runtime.Post(LiveAttachMsg{Config: LiveConfig{TerminalID: "term-1", Cols: 80, Rows: 24}}); err != nil {
		t.Fatalf("post attach: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if runtime.State().Session.LastError != "attach failed" {
		t.Fatalf("expected attach error in state, got %#v", runtime.State())
	}
	frames := host.Frames()
	last := lastFrame(t, frames)
	if len(last.Lines) < 2 || !frameContains(last, "attach failed") {
		t.Fatalf("expected rendered error status, got %#v", last.Lines)
	}
}

func TestLiveRuntimeIncludesShellReducer(t *testing.T) {
	host := NewFakeTerminalHost(4)
	runtime := NewLiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
	)

	if err := runtime.Post(ShellSetHeaderVisibleMsg{Visible: false}); err != nil {
		t.Fatalf("post shell action: %v", err)
	}
	if err := runtime.Post(ShellSetPanelPresentationMsg{Presentation: state.PanelPresentationSplitLine}); err != nil {
		t.Fatalf("post panel action: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if runtime.State().Shell.HeaderVisible {
		t.Fatalf("expected hidden header, got %#v", runtime.State().Shell)
	}
	if runtime.State().Shell.PanelPresentation != state.PanelPresentationSplitLine {
		t.Fatalf("expected split line presentation, got %#v", runtime.State().Shell)
	}
}

func TestInteractiveRuntimeIncludesShellReducer(t *testing.T) {
	host := NewFakeTerminalHost(4)
	runtime := NewInteractiveRuntime(
		state.Root{},
		host,
		NewSyncEffectRunner(),
		LiveDeps{Terminal: &testkit.FakeTerminalService{}},
		CopyModeDeps{Core: &testkit.FakeCoreClient{}},
	)

	if err := runtime.Post(ShellOpenTerminalPickerMsg{}); err != nil {
		t.Fatalf("post terminal picker action: %v", err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if !runtime.State().Shell.Overlay.Open || runtime.State().Shell.Overlay.Kind != state.OverlayTerminalPicker {
		t.Fatalf("expected terminal picker overlay, got %#v", runtime.State().Shell.Overlay)
	}
}

func compactInputRequests(inputs []port.TerminalInputRequest) []string {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]string, 0, len(inputs))
	var currentTerminal string
	var currentChannel uint16
	var current strings.Builder
	flush := func() {
		if currentTerminal == "" {
			return
		}
		out = append(out, fmt.Sprintf("%s#%d:%s", currentTerminal, currentChannel, current.String()))
		current.Reset()
	}
	for _, input := range inputs {
		if input.TerminalID != currentTerminal || input.Channel != currentChannel {
			flush()
			currentTerminal = input.TerminalID
			currentChannel = input.Channel
		}
		current.Write(input.Bytes)
	}
	flush()
	return out
}

func lastFrame(t *testing.T, frames []render.Frame) render.Frame {
	t.Helper()
	if len(frames) == 0 {
		t.Fatal("expected rendered frames")
	}
	return frames[len(frames)-1]
}

func frameContains(frame render.Frame, value string) bool {
	for _, line := range frame.Lines {
		if strings.Contains(line, value) {
			return true
		}
	}
	return false
}

func ansiFrameContains(frame render.Frame, value string) bool {
	for _, line := range frame.ANSILines {
		if strings.Contains(line, value) {
			return true
		}
	}
	return false
}

func activePanelVMForAppTest(shell render.ShellVM) (render.PanelVM, bool) {
	for _, panel := range shell.Layout.Panels {
		if panel.Active {
			return panel, true
		}
	}
	return render.PanelVM{}, false
}

func activeContentRectForAppTest(layout render.LayoutPlan) (render.Rect, bool) {
	for _, panel := range layout.Panels {
		if panel.Panel.Active {
			return panel.ContentRect, true
		}
	}
	return render.Rect{}, false
}

func firstPanelLayerForAppTest(result render.RenderResult) (render.Layer, bool) {
	for _, layer := range result.Layers {
		if layer.Kind == render.LayerPanel {
			return layer, true
		}
	}
	return render.Layer{}, false
}

func panelLayerByPaneIDForAppTest(result render.RenderResult, plan render.LayoutPlan, paneID string) (render.Layer, bool) {
	for _, panel := range plan.Panels {
		if panel.Panel.ID != paneID {
			continue
		}
		for _, layer := range result.Layers {
			if layer.Kind == render.LayerPanel && layer.Rect == panel.Rect {
				return layer, true
			}
		}
	}
	return render.Layer{}, false
}

func panelLayerContainsPlainForAppTest(layer render.Layer, value string) bool {
	for _, line := range layer.Lines {
		if strings.Contains(line.PlainString(), value) {
			return true
		}
	}
	return false
}

func drainUntilFrameContains(ctx context.Context, runtime *AppRuntime, host *FakeTerminalHost, value string) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := runtime.Drain(deadlineCtx); err != nil {
			return err
		}
		for _, frame := range host.Frames() {
			if frameContains(frame, value) {
				return nil
			}
		}
		select {
		case <-deadlineCtx.Done():
			return deadlineCtx.Err()
		case <-ticker.C:
		}
	}
}

func waitForLiveScreenNextRequest(ctx context.Context, runtime *AppRuntime, terminal *testkit.FakeTerminalService, terminalID string) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := runtime.Drain(deadlineCtx); err != nil {
			return err
		}
		for _, request := range terminal.LiveScreenNextRequestsSnapshot() {
			if request.TerminalID == terminalID {
				return nil
			}
		}
		select {
		case <-deadlineCtx.Done():
			return deadlineCtx.Err()
		case <-ticker.C:
		}
	}
}
