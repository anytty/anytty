package app

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// Msg 是 TUI-v3 runtime 的根消息契约，不绑定 Bubble Tea。
type Msg interface {
	isMsg()
}

// Effect 是 TUI-v3 runtime 的根副作用契约，不等同于 Bubble Tea Cmd。
type Effect interface {
	isEffect()
}

// NoopMsg 是 smoke-test 消息。
type NoopMsg struct{}

func (NoopMsg) isMsg() {}

type noRenderMsg interface {
	SkipRender() bool
}

type frameWriteCompletedMsg struct {
	Written bool
	Err     error
}

func (frameWriteCompletedMsg) isMsg() {}

// QuitMsg 请求 runtime 退出。
type QuitMsg struct{}

func (QuitMsg) isMsg() {}

// InputMsg 是 TerminalHost 输入事件进入 message path 的边界消息。
type InputMsg struct {
	Event input.InputEvent
	// 中文说明：runtime 命中测试已经确认 raw mouse 属于命中的 terminal 内容区；
	// UI/copy reducer 必须避让，交给 terminal input router 发送给子进程。
	TerminalMousePassthrough bool
	// 中文说明：鼠标命中的 TerminalView 是 passthrough 目标；它避免点击非 active pane
	// 时把 nvim/htop 的 raw mouse 发送到旧 active terminal。
	TerminalMouseTargetViewID string
}

func (InputMsg) isMsg() {}

// HostResizeMsg 是外部 terminal emulator 尺寸进入 reducer-owned state 的入口。
type HostResizeMsg struct {
	Cols int
	Rows int
}

func (HostResizeMsg) isMsg() {}

// TickMsg 是 timer/interval effect 回投后的普通消息。
type TickMsg struct {
	Token CancelToken
	Ticks uint64
}

func (TickMsg) isMsg() {}

// NoopEffect 是无需执行的副作用。
type NoopEffect struct{}

func (NoopEffect) isEffect() {}

// FuncEffect 是 harness 和 service adapter 使用的最小副作用包装。
type FuncEffect struct {
	Token CancelToken
	Async bool
	// SerialKey 让真实异步 runner 对同一个 domain owner 串行执行 effect。
	// 它用于 terminal input 这类 byte stream：主循环不能同步等待 ack，但同一
	// terminal/view/channel 的输入顺序是 PTY 消息链路 truth，不能被 goroutine 调度打乱。
	SerialKey string
	// ForceSyncInTests 只给 deterministic harness 使用：真实 runtime 仍按 Async 异步执行，
	// 但 SyncEffectRunner 需要同步跑完该 effect，避免测试必须引入额外 goroutine 等待。
	ForceSyncInTests bool
	Run              func(context.Context) Msg
}

func (FuncEffect) isEffect() {}

// StreamEffect 表达会多次回投消息的长生命周期 service 订阅。
type StreamEffect struct {
	Token CancelToken
	Run   func(context.Context, func(Msg))
}

func (StreamEffect) isEffect() {}

// BatchEffect 表达多个 effect 的调度组合。
type BatchEffect struct {
	Effects []Effect
}

func (BatchEffect) isEffect() {}

// CancelEffect 请求取消某个 token 绑定的 pending effect。
type CancelEffect struct {
	Token CancelToken
}

func (CancelEffect) isEffect() {}

// CancelToken 用于取消 pending history request、terminal operation 和 timer。
type CancelToken string

// Reducer 保持同步纯边界：输入 state/message，返回新 state/effects。
type Reducer func(state.Root, Msg) (state.Root, []Effect)

// RenderFunc 把 reducer-owned state 投影成 frame。
type RenderFunc func(state.Root) render.Frame

type handledEffect struct{}

func (handledEffect) isEffect() {}

const (
	defaultToastTickInterval   = time.Second
	toastTickToken             = CancelToken("toast.tick")
	defaultMaxMessagesPerBatch = 128
)

// ComposeReducers 按顺序执行多个 reducer，并合并它们产生的 effects。
func ComposeReducers(reducers ...Reducer) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		var effects []Effect
		for _, reducer := range reducers {
			if reducer == nil {
				continue
			}
			next, nextEffects := reducer(root, msg)
			root = next
			handled := false
			for _, effect := range nextEffects {
				if _, ok := effect.(handledEffect); ok {
					handled = true
					continue
				}
				effects = append(effects, effect)
			}
			if handled {
				break
			}
		}
		return root, effects
	}
}

// TerminalHost 是宿主 TTY 边界。真实实现负责 raw mode、输入和 FrameSink；
// fake 实现只用于 harness。
type TerminalHost interface {
	Size() (cols int, rows int, err error)
	InputEvents() <-chan input.InputEvent
	EventsReady() <-chan struct{}
	FrameSink() render.FrameSink
}

// EffectRunner 执行 effect，并只通过 message path 回投结果。
type EffectRunner interface {
	Run(context.Context, Effect, func(Msg))
	Cancel(CancelToken)
}

var (
	ErrRuntimeStopped = errors.New("app runtime stopped")
	ErrInputQueueFull = errors.New("terminal host input queue full")
)

// AppRuntime 是 TUI-v3 自有单线程消息循环。
type AppRuntime struct {
	mu                   sync.Mutex
	wake                 chan struct{}
	state                state.Root
	reduce               Reducer
	render               RenderFunc
	host                 TerminalHost
	runner               EffectRunner
	queue                []Msg
	lastHitRegions       []render.HitRegion
	mouseDrag            mouseDragState
	lastMouseAction      mouseActionClickState
	lastMouseActionAt    time.Time
	hostSizeInitialized  bool
	now                  func() time.Time
	toastTickInterval    time.Duration
	lastToastTick        time.Time
	running              bool
	quit                 bool
	firstFrameWritten    bool
	startupFrameReady    bool
	maxMessagesPerBatch  int
	frameWriteInFlight   bool
	frameWriteCommit     func()
	frameWriteNeedsRetry bool
	frameWriteErr        error
	renderPending        bool
	forceFullFrame       bool
	copyHistoryPatch     copyHistoryPatchCache
	diagnostics          *runtimeDiagnostics
	stopDiagnostics      func()
}

func NewAppRuntime(
	initial state.Root,
	reducer Reducer,
	renderer RenderFunc,
	host TerminalHost,
	runner EffectRunner,
) *AppRuntime {
	if reducer == nil {
		reducer = func(root state.Root, _ Msg) (state.Root, []Effect) {
			return root, nil
		}
	}
	if renderer == nil {
		renderer = func(state.Root) render.Frame { return render.Frame{} }
	}
	if runner == nil {
		runner = NewSyncEffectRunner()
	}
	return &AppRuntime{
		state:             initial,
		wake:              make(chan struct{}, 1),
		reduce:            reducer,
		render:            renderer,
		host:              host,
		runner:            runner,
		now:               time.Now,
		toastTickInterval: defaultToastTickInterval,
	}
}

func (runtime *AppRuntime) SetLogger(logger *slog.Logger) {
	if runtime.stopDiagnostics != nil {
		runtime.stopDiagnostics()
		runtime.stopDiagnostics = nil
	}
	applyRuntimeTuning(logger)
	runtime.diagnostics = newRuntimeDiagnostics(logger)
	runtime.stopDiagnostics = startRuntimeHeapSignalProfiler(runtime, logger)
}

func (runtime *AppRuntime) State() state.Root {
	return runtime.state
}

func (runtime *AppRuntime) Post(msg Msg) error {
	if runtime.quit {
		return ErrRuntimeStopped
	}
	msg = runtime.dispatchMouseHitRegion(msg)
	runtime.enqueue(msg)
	return nil
}

func (runtime *AppRuntime) Drain(ctx context.Context) error {
	runtime.running = true
	defer func() {
		runtime.running = false
		runtime.stopRuntimeDiagnostics()
	}()
	return runtime.drainBatch(ctx)
}

// Run 进入真正的事件驱动主循环：处理一批消息后等待下一次唤醒，
// 不依赖外层固定 sleep 轮询。
func (runtime *AppRuntime) Run(ctx context.Context) error {
	runtime.running = true
	defer func() {
		runtime.running = false
		runtime.stopRuntimeDiagnostics()
	}()
	for {
		if err := runtime.drainBatch(ctx); err != nil {
			return err
		}
		if runtime.quit {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !runtime.waitForWake(ctx) {
			return ctx.Err()
		}
	}
}

func (runtime *AppRuntime) stopRuntimeDiagnostics() {
	if runtime.stopDiagnostics == nil {
		return
	}
	runtime.stopDiagnostics()
	runtime.stopDiagnostics = nil
}

func (runtime *AppRuntime) RequestHeapProfile(reason string) {
	if runtime == nil || runtime.diagnostics == nil {
		return
	}
	runtime.diagnostics.RequestHeapProfile(runtime.State(), reason)
}

func (runtime *AppRuntime) RequestMemstats(reason string) {
	if runtime == nil || runtime.diagnostics == nil {
		return
	}
	runtime.diagnostics.RequestMemstats(runtime.State(), reason)
}

func (runtime *AppRuntime) drainBatch(ctx context.Context) error {
	processed := 0
	for {
		if err := runtime.takeFrameWriteError(); err != nil {
			return err
		}
		runtime.ingestHostInitialSize()
		runtime.ingestHostInput()
		runtime.enqueueDueToastTick()
		runtime.enqueueDueToastTick()
		msg, ok := runtime.dequeue()
		if !ok {
			runtime.ingestHostCurrentSize()
			msg, ok = runtime.dequeue()
			if !ok {
				if runtime.renderPending && !runtime.frameWriteInFlight {
					runtime.renderFrame()
					runtime.ingestHostInput()
					runtime.enqueueDueToastTick()
					continue
				}
				return nil
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok := msg.(QuitMsg); ok {
			runtime.quit = true
			return nil
		}
		if !runtime.prepareRuntimeMessage(msg) {
			runtime.ingestHostInput()
			if runtime.quit {
				return nil
			}
			continue
		}
		next, effects := runtime.reduce(runtime.state, msg)
		runtime.state = next
		runtime.observeRuntimeMessage(msg, effects)
		processed++
		if !messageSkipsRender(msg) {
			runtime.noteRenderPending(msg)
		} else {
			perftrace.Count("tui.message_skip_render", messageApproxBytes(msg))
		}
		if _, ok := msg.(HostResizeMsg); ok {
			runtime.renderFrame()
		}
		if runtime.shouldWriteFirstFrame() {
			runtime.noteRenderPending(nil)
			runtime.renderFrame()
		}
		for _, effect := range effects {
			runtime.scheduleEffect(ctx, effect)
		}
		runtime.ingestHostInput()
		runtime.enqueueDueToastTick()
		if runtime.quit {
			return nil
		}
		if runtime.renderPending && processed >= runtime.messageBatchLimit() {
			if !runtime.frameWriteInFlight {
				runtime.renderFrame()
			}
			return nil
		}
	}
}

func (runtime *AppRuntime) messageBatchLimit() int {
	if runtime.maxMessagesPerBatch > 0 {
		return runtime.maxMessagesPerBatch
	}
	return defaultMaxMessagesPerBatch
}

func messageSkipsRender(msg Msg) bool {
	noRender, ok := msg.(noRenderMsg)
	return ok && noRender.SkipRender()
}

func (runtime *AppRuntime) noteRenderPending(msg Msg) {
	if _, ok := msg.(LiveScreenNextResultMsg); ok {
		runtime.forceFullFrame = true
	}
	runtime.renderPending = true
}

func (runtime *AppRuntime) scheduleEffect(ctx context.Context, effect Effect) {
	switch effect := effect.(type) {
	case nil:
		return
	case NoopEffect:
		return
	case BatchEffect:
		for _, child := range effect.Effects {
			runtime.scheduleEffect(ctx, child)
		}
	case CancelEffect:
		runtime.runner.Cancel(effect.Token)
	default:
		runtime.runner.Run(ctx, effect, func(msg Msg) {
			if msg != nil && !runtime.quit {
				runtime.enqueue(msg)
			}
		})
	}
}

func (runtime *AppRuntime) prepareRuntimeMessage(msg Msg) bool {
	switch msg := msg.(type) {
	case frameWriteCompletedMsg:
		runtime.finishFrameWrite(msg.Written, msg.Err)
		return false
	case HostResizeMsg:
		next, changed := runtime.state.Viewport.Resize(msg.Cols, msg.Rows)
		if !changed {
			return false
		}
		runtime.state.Viewport = next
		runtime.state = runtime.state.Advance()
		return true
	default:
		return true
	}
}

func (runtime *AppRuntime) ingestHostInitialSize() {
	if runtime.host == nil || runtime.hostSizeInitialized {
		return
	}
	runtime.hostSizeInitialized = true
	if runtime.ingestHostCurrentSize() {
		runtime.startupFrameReady = true
	}
}

func (runtime *AppRuntime) ingestHostCurrentSize() bool {
	if runtime.host == nil {
		return false
	}
	cols, rows, err := runtime.host.Size()
	if err != nil || cols <= 0 || rows <= 0 {
		return false
	}
	if runtime.state.Viewport.Valid && runtime.state.Viewport.Cols == cols && runtime.state.Viewport.Rows == rows {
		return false
	}
	msg := HostResizeMsg{Cols: cols, Rows: rows}
	runtime.prepend(msg)
	return true
}

func (runtime *AppRuntime) ingestHostInput() {
	if runtime.host == nil {
		return
	}
	events := runtime.host.InputEvents()
	var terminalBatch []byte
	var terminalBatchEvent input.InputEvent
	blockTerminalBatch := false
	flushTerminalBatch := func() {
		if len(terminalBatch) == 0 {
			return
		}
		payload := append([]byte(nil), terminalBatch...)
		runtime.enqueue(TerminalInputBytesMsg{Event: terminalBatchEvent, Bytes: payload})
		terminalBatch = nil
		terminalBatchEvent = input.InputEvent{}
	}
	for {
		select {
		case event := <-events:
			switch event.Kind {
			case input.EventKindResize:
				flushTerminalBatch()
				runtime.enqueue(HostResizeMsg{Cols: event.Cols, Rows: event.Rows})
			case input.EventKindHostTheme:
				flushTerminalBatch()
				runtime.enqueue(HostThemeMsg{Update: state.HostThemeUpdate{
					DefaultFG:    event.Theme.DefaultFG,
					DefaultBG:    event.Theme.DefaultBG,
					PaletteIndex: event.Theme.PaletteIndex,
					PaletteColor: event.Theme.PaletteColor,
				}})
			case input.EventKindHostCapability:
				flushTerminalBatch()
				runtime.enqueue(HostCapabilityMsg{Update: state.HostCapabilityUpdate{
					KeyboardDisambiguation: event.Capability.KeyboardDisambiguation,
				}})
			case input.EventKindHostControl:
				// 未识别 host 控制响应在 TerminalHost 已完成分帧；这里只隔断 terminal batch，不生成用户输入消息。
				flushTerminalBatch()
			default:
				if !blockTerminalBatch {
					if bytes, ok := runtime.coalescableTerminalInputBytes(event); ok {
						if len(terminalBatch) == 0 {
							terminalBatchEvent = event
						}
						terminalBatch = append(terminalBatch, bytes...)
						continue
					}
				}
				flushTerminalBatch()
				msg := runtime.dispatchMouseHitRegion(InputMsg{Event: event})
				runtime.logHostInputEvent(event, msg)
				runtime.enqueue(msg)
				// 中文说明：同一次 host input drain 内，前一个普通 InputMsg 可能打开
				// overlay/copy/prefix mode，但 reducer 还没来得及更新 runtime.state；
				// 后续输入保持原消息边界，避免基于旧 state 误合并进 PTY。
				blockTerminalBatch = true
			}
		default:
			flushTerminalBatch()
			return
		}
	}
}

func (runtime *AppRuntime) coalescableTerminalInputBytes(event input.InputEvent) ([]byte, bool) {
	if event.Kind != input.EventKindKey {
		return nil, false
	}
	if event.Alt || event.Ctrl || event.Shift {
		return nil, false
	}
	var bytes []byte
	switch event.Key {
	case input.KeyChar:
		if event.Char == "" {
			return nil, false
		}
		bytes = []byte(event.Char)
	case input.KeyEnter:
		bytes = []byte{'\r'}
	case input.KeyBackspace:
		bytes = []byte{0x7f}
	case input.KeyTab:
		bytes = []byte{'\t'}
	default:
		return nil, false
	}
	root := runtime.state
	if copyModeOwnsActiveInput(root) {
		return nil, false
	}
	shell := root.Shell.ReadonlyDefaults()
	if shell.Overlay.Open || shell.InteractionMode != state.InteractionModeNormal {
		return nil, false
	}
	if _, _, ok := activeExitedPaneCTATarget(root, shell); ok {
		// 退出态 action 由 UI reducer 拥有；runtime 的 PTY byte 合并不能在 reducer 前吞掉 R/Enter。
		return nil, false
	}
	if _, _, ok := activeDisconnectedPaneCTATarget(root, shell); ok {
		// 断线态 action 由 UI reducer 拥有；runtime 不能把 R/Enter 发送到已经失效的 channel。
		return nil, false
	}
	target, ok := liveInputTarget(root)
	if !ok || target.Channel == 0 || target.AttachPending {
		return nil, false
	}
	intent := input.RouteWithOptions(event, input.RouteOptions{
		CopyModeActive:           false,
		TerminalMousePassthrough: false,
		Shortcuts:                root.Config.Shortcuts,
	})
	if intent.Kind != input.IntentTerminalInput || len(intent.Bytes) == 0 || intent.RawMouse {
		return nil, false
	}
	if string(intent.Bytes) != string(bytes) {
		return nil, false
	}
	// 中文说明：这里只合并已经明确属于 terminal passthrough 的普通 key bytes；
	// copy/overlay/prefix/鼠标/resize/theme 保持原消息边界，避免 UI command 被吞进 PTY。
	return bytes, true
}

func (runtime *AppRuntime) enqueueDueToastTick() {
	if len(runtime.state.Shell.Toasts) == 0 {
		runtime.lastToastTick = time.Time{}
		return
	}
	interval := runtime.toastTickInterval
	if interval <= 0 {
		interval = defaultToastTickInterval
	}
	now := runtime.currentTime()
	if runtime.lastToastTick.IsZero() {
		runtime.lastToastTick = now
		return
	}
	elapsed := now.Sub(runtime.lastToastTick)
	if elapsed < interval {
		return
	}
	ticks := uint64(elapsed / interval)
	if ticks == 0 {
		return
	}
	runtime.lastToastTick = runtime.lastToastTick.Add(time.Duration(ticks) * interval)
	runtime.enqueue(TickMsg{Token: toastTickToken, Ticks: ticks})
}

func (runtime *AppRuntime) currentTime() time.Time {
	if runtime.now != nil {
		return runtime.now()
	}
	return time.Now()
}

func (runtime *AppRuntime) waitForWake(ctx context.Context) bool {
	runtime.enqueueDueToastTick()
	if runtime.ingestHostCurrentSize() {
		return true
	}
	runtime.mu.Lock()
	hasQueued := len(runtime.queue) > 0
	wake := runtime.wake
	runtime.mu.Unlock()
	if hasQueued {
		return true
	}
	var hostReady <-chan struct{}
	if runtime.host != nil {
		hostReady = runtime.host.EventsReady()
	}
	var toastC <-chan time.Time
	if delay := runtime.nextToastWakeDelay(); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		toastC = timer.C
	} else if delay == 0 {
		runtime.enqueueDueToastTick()
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-wake:
		return true
	case <-hostReady:
		return true
	case <-toastC:
		runtime.enqueueDueToastTick()
		return true
	}
}

func (runtime *AppRuntime) nextToastWakeDelay() time.Duration {
	if len(runtime.state.Shell.Toasts) == 0 {
		return -1
	}
	interval := runtime.toastTickInterval
	if interval <= 0 {
		interval = defaultToastTickInterval
	}
	now := runtime.currentTime()
	if runtime.lastToastTick.IsZero() {
		return 0
	}
	elapsed := now.Sub(runtime.lastToastTick)
	if elapsed >= interval {
		return 0
	}
	return interval - elapsed
}

func (runtime *AppRuntime) renderFrame() bool {
	if runtime.host == nil {
		runtime.renderPending = false
		return false
	}
	if runtime.frameWriteInFlight {
		return false
	}
	finishTotal := perftrace.Measure("tui.render_frame_total")
	if !runtime.forceFullFrame && runtime.tryRenderCopyHistoryPatch() {
		runtime.renderPending = runtime.frameWriteNeedsRetry
		finishTotal(0)
		return true
	}
	finishRender := perftrace.Measure("tui.render_build_frame")
	frame := runtime.render(runtime.state)
	frameBytes := frameApproxBytes(frame)
	finishRender(frameBytes)
	finishSinkEnqueue := perftrace.Measure("tui.frame_sink_enqueue")
	done := runtime.writeFrame(frame)
	runtime.enqueueLiveScreenFrameSelected(frame, true)
	finishSinkEnqueue(frameBytes)
	perftrace.Count("tui.frame", frameBytes)
	runtime.firstFrameWritten = true
	commit := runtime.fullFrameCommit(frame, runtime.state)
	runtime.observeRuntimeFrame(frame)
	runtime.renderPending = false
	runtime.forceFullFrame = false
	runtime.trackFrameCompletion(done, commit)
	finishTotal(frameBytes)
	return true
}

func (runtime *AppRuntime) writeFrame(frame render.Frame) <-chan render.FrameWriteCompletion {
	if runtime.host == nil {
		done := make(chan render.FrameWriteCompletion, 1)
		done <- render.FrameWriteCompletion{Written: true}
		close(done)
		return done
	}
	sink := runtime.host.FrameSink()
	if completion, ok := sink.(render.FrameSinkCompletion); ok {
		done, err := completion.WriteFrameWithCompletion(frame)
		if err != nil {
			completed := make(chan render.FrameWriteCompletion, 1)
			completed <- render.FrameWriteCompletion{Err: err}
			close(completed)
			return completed
		}
		if done != nil {
			return done
		}
	}
	err := sink.WriteFrame(frame)
	done := make(chan render.FrameWriteCompletion, 1)
	done <- render.FrameWriteCompletion{Written: err == nil, Err: err}
	close(done)
	return done
}

func (runtime *AppRuntime) trackFrameCompletion(done <-chan render.FrameWriteCompletion, commit func()) {
	runtime.frameWriteNeedsRetry = false
	runtime.frameWriteInFlight = true
	runtime.frameWriteCommit = commit
	if done == nil {
		runtime.finishFrameWrite(false, nil)
		return
	}
	select {
	case completion, ok := <-done:
		if !ok {
			runtime.finishFrameWrite(false, nil)
		} else {
			runtime.finishFrameWrite(completion.Written, completion.Err)
		}
	default:
		go runtime.awaitFrameCompletion(done)
	}
}

func (runtime *AppRuntime) finishFrameWrite(written bool, err error) {
	commit := runtime.frameWriteCommit
	runtime.frameWriteCommit = nil
	runtime.frameWriteInFlight = false
	runtime.frameWriteNeedsRetry = !written && err == nil
	if err != nil {
		runtime.frameWriteErr = err
		return
	}
	if written {
		if commit != nil {
			commit()
		}
		return
	}
	// 失败或被 sink 丢弃时，旧视觉基线仍然有效；下一次必须从 canonical state 完整重绘。
	runtime.renderPending = true
	runtime.forceFullFrame = true
	runtime.copyHistoryPatch = copyHistoryPatchCache{}
}

func (runtime *AppRuntime) takeFrameWriteError() error {
	err := runtime.frameWriteErr
	runtime.frameWriteErr = nil
	return err
}

func (runtime *AppRuntime) fullFrameCommit(frame render.Frame, root state.Root) func() {
	hitRegions := cloneRenderHitRegions(frame.HitRegions)
	copyCache, copyOK := copyHistoryPatchCacheForFrame(runtime, root, frame)
	return func() {
		runtime.lastHitRegions = hitRegions
		if copyOK {
			runtime.copyHistoryPatch = copyCache
		} else {
			runtime.copyHistoryPatch = copyHistoryPatchCache{}
		}
	}
}

func (runtime *AppRuntime) awaitFrameCompletion(done <-chan render.FrameWriteCompletion) {
	completion, ok := <-done
	runtime.enqueue(frameWriteCompletedMsg{Written: ok && completion.Written, Err: completion.Err})
}

func (runtime *AppRuntime) enqueueLiveScreenFrameSelected(frame render.Frame, full bool) {
	if len(frame.LiveTargets) == 0 && len(runtime.state.Surface.LiveScreens) == 0 {
		return
	}
	runtime.enqueue(LiveScreenFrameSelectedMsg{
		Full:    full,
		Targets: append([]render.LiveRenderTarget(nil), frame.LiveTargets...),
	})
}

func frameApproxBytes(frame render.Frame) int {
	total := 0
	for _, line := range frame.Lines {
		total += len(line)
	}
	for _, line := range frame.ANSILines {
		total += len(line)
	}
	return total
}

func (runtime *AppRuntime) shouldWriteFirstFrame() bool {
	return runtime.startupFrameReady && !runtime.firstFrameWritten && runtime.state.Viewport.Valid
}

func (runtime *AppRuntime) Running() bool {
	return runtime.running
}

func (runtime *AppRuntime) Quit() bool {
	return runtime.quit
}

// FakeTerminalHost 是 runtime harness 使用的 TerminalHost fake。
type FakeTerminalHost struct {
	events chan input.InputEvent
	ready  chan struct{}
	sink   *FakeFrameSink
	cols   int
	rows   int
}

func NewFakeTerminalHost(buffer int) *FakeTerminalHost {
	if buffer <= 0 {
		buffer = 1
	}
	return &FakeTerminalHost{
		events: make(chan input.InputEvent, buffer),
		ready:  make(chan struct{}, 1),
		sink:   &FakeFrameSink{},
	}
}

func (host *FakeTerminalHost) SendInput(event input.InputEvent) error {
	select {
	case host.events <- event:
		host.signalReady()
		return nil
	default:
		return ErrInputQueueFull
	}
}

func (host *FakeTerminalHost) SetSize(cols int, rows int) {
	host.cols = cols
	host.rows = rows
}

func (host *FakeTerminalHost) Size() (int, int, error) {
	return host.cols, host.rows, nil
}

func (host *FakeTerminalHost) SendResize(cols int, rows int) error {
	host.SetSize(cols, rows)
	select {
	case host.events <- input.InputEvent{Kind: input.EventKindResize, Cols: cols, Rows: rows}:
		host.signalReady()
		return nil
	default:
		return ErrInputQueueFull
	}
}

func (host *FakeTerminalHost) InputEvents() <-chan input.InputEvent {
	return host.events
}

func (host *FakeTerminalHost) EventsReady() <-chan struct{} {
	return host.ready
}

func (host *FakeTerminalHost) FrameSink() render.FrameSink {
	return host.sink
}

func (host *FakeTerminalHost) Frames() []render.Frame {
	return host.sink.Frames()
}

func (host *FakeTerminalHost) signalReady() {
	select {
	case host.ready <- struct{}{}:
	default:
	}
}

// FakeFrameSink 记录 renderer 输出，供 harness 断言。
type FakeFrameSink struct {
	frames  []render.Frame
	onWrite func()
}

func (sink *FakeFrameSink) WriteFrame(frame render.Frame) error {
	if sink.onWrite != nil {
		sink.onWrite()
	}
	cloned := frame.Clone()
	sink.frames = append(sink.frames, cloned)
	return nil
}

func (sink *FakeFrameSink) Frames() []render.Frame {
	frames := make([]render.Frame, len(sink.frames))
	for i, frame := range sink.frames {
		frames[i] = frame.Clone()
	}
	return frames
}
