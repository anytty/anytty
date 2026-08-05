package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

// NewTerminalInputRouterReducer 是普通 terminal 输入的唯一入口。
// UI、overlay、copy mode 已消费的输入不会到这里；未消费的 key/mouse passthrough
// 必须在这里统一解析 active TerminalView binding，避免 key 和 mouse 走两套目标选择逻辑。
func NewTerminalInputRouterReducer(deps LiveDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case InputMsg:
			return reduceTerminalInputRoute(root, msg, deps)
		case TerminalInputBytesMsg:
			return reduceTerminalInputBytes(root, msg, deps)
		default:
			return root, nil
		}
	}
}

// TerminalInputBytesMsg 是 runtime 在普通 terminal passthrough 状态下合并出的
// 有序 PTY byte stream。它只服务 terminal input，不改变 TUI view state；
// reducer 会在发送前重新检查当前目标，避免 stale batch 穿透 overlay/copy mode。
type TerminalInputBytesMsg struct {
	Event input.InputEvent
	Bytes []byte
}

func (TerminalInputBytesMsg) isMsg() {}

func (TerminalInputBytesMsg) SkipRender() bool {
	return true
}

func reduceTerminalInputRoute(root state.Root, msg InputMsg, deps LiveDeps) (state.Root, []Effect) {
	if deps.Terminal == nil {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Result: "blocked",
			Reason: "terminal service missing",
		})
		return setLiveError(root, "terminal service missing"), nil
	}
	shell := root.Shell.ReadonlyDefaults()
	forceTerminalPassthrough := false
	if nextShell, forced := shell.ConsumeTerminalInputPassthroughOnce(); forced {
		// 中文说明：UI reducer 已确认当前 InputMsg 是显式 passthrough；
		// terminal reducer 只消费同一消息上的一次性令牌，避免异步补发造成 PTY 输入乱序。
		root.Shell = nextShell
		shell = root.Shell.ReadonlyDefaults()
		forceTerminalPassthrough = true
	} else if shell.ShortcutPassthroughLocked {
		if _, ok := input.LockableRootShortcutIntentWithShortcuts(msg.Event, root.Config.Shortcuts); ok {
			forceTerminalPassthrough = true
		}
	}
	if copyModeOwnsActiveInput(root) && !forceTerminalPassthrough {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Result: "consumed",
			Reason: "copy mode active",
		})
		return root, []Effect{handledEffect{}}
	}
	if shell.Overlay.Open {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Result: "consumed",
			Reason: "overlay open",
		})
		return root, []Effect{handledEffect{}}
	}
	if shell.InteractionMode != state.InteractionModeNormal {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Result: "consumed",
			Reason: "interaction mode active",
		})
		return root, []Effect{handledEffect{}}
	}
	var target liveInputTargetInfo
	target, ok := liveInputTarget(root)
	if msg.TerminalMouseTargetViewID != "" {
		var targetOK bool
		target, targetOK = liveInputTargetForView(root, msg.TerminalMouseTargetViewID)
		if !targetOK {
			ok = false
		} else {
			ok = true
			root = focusTerminalInputTarget(root, target)
		}
	}
	if !ok {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Result: "blocked",
			Reason: "no active terminal view binding",
		})
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.input", Body: "no terminal bound"})
		return root.Advance(), []Effect{handledEffect{}}
	}
	intent := input.RouteWithOptions(msg.Event, input.RouteOptions{
		CopyModeActive:           false,
		TerminalMousePassthrough: msg.TerminalMousePassthrough || liveMousePassthroughEnabled(root, msg.Event, target),
		ForceTerminalPassthrough: forceTerminalPassthrough,
		Shortcuts:                root.Config.Shortcuts,
	})
	if intent.Kind != input.IntentTerminalInput || len(intent.Bytes) == 0 {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Target: target,
			Result: "ignored",
			Reason: string(intent.Kind),
		})
		return root, nil
	}
	terminalBytes := intent.Bytes
	if msg.Event.Kind == input.EventKindPaste {
		// paste marker 是否发给 PTY 只由 owning TerminalRef 的 live mode 决定；Host/input 不拥有该真值。
		modes := root.Surface.SurfaceForTerminalRef(state.NewTerminalRef(target.EndpointID, target.TerminalID)).Modes
		terminalBytes = encodeTerminalPaste(msg.Event.Paste, modes)
	}
	logTerminalInputRoute(deps, root, terminalInputRouteLog{
		Event:       msg.Event,
		Target:      target,
		Result:      "terminal",
		Bytes:       len(terminalBytes),
		RawMouse:    intent.RawMouse,
		NeedsAttach: target.Channel == 0,
	})
	if target.AttachPending {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "terminal.input", Body: "attach pending", Pending: true})
		return root.Advance(), []Effect{handledEffect{}}
	}
	if target.Channel == 0 {
		var effect Effect
		root, effect = liveAttachForInputEffect(root, target, msg.Event, terminalBytes, deps)
		return root, []Effect{effect}
	}
	var operationID string
	root.TerminalViews, operationID = root.TerminalViews.NextTerminalOperation(inputOperationKind(msg.Event), target.ViewID)
	return root, []Effect{terminalSendInputEffect(target, msg.Event, terminalBytes, operationID, deps)}
}

func encodeTerminalPaste(text string, modes state.LiveTerminalModes) []byte {
	if text == "" {
		return nil
	}
	if !modes.BracketedPaste {
		return []byte(text)
	}
	return []byte("\x1b[200~" + text + "\x1b[201~")
}

func focusTerminalInputTarget(root state.Root, target liveInputTargetInfo) state.Root {
	shell := root.Shell.EnsureDefaults()
	if target.FloatingID != "" {
		nextShell, result := shell.ApplyFloatingCommand(state.FloatingCommand{
			Action:   state.FloatingCommandFocusRaise,
			TargetID: target.FloatingID,
			Source:   state.PaneCommandSourceMouse,
		})
		if result.Status == state.FloatingCommandOK {
			root.Shell = nextShell.ExitInteractionMode()
		}
		return root
	}
	if target.PaneID != "" {
		// 中文说明：tracked mouse 的同一次点击既是 terminal input，也是用户焦点意图；
		// focus 必须落到命中的 pane，避免 raw mouse 被路由给旧 active terminal。
		root.Shell = shell.FocusPane(state.PaneCommandTarget{PaneID: target.PaneID}).ExitInteractionMode()
	}
	return root
}

func reduceTerminalInputBytes(root state.Root, msg TerminalInputBytesMsg, deps LiveDeps) (state.Root, []Effect) {
	if len(msg.Bytes) == 0 {
		return root, nil
	}
	if deps.Terminal == nil {
		logTerminalInputRoute(deps, root, terminalInputRouteLog{
			Event:  msg.Event,
			Result: "blocked",
			Reason: "terminal service missing",
			Bytes:  len(msg.Bytes),
		})
		return setLiveError(root, "terminal service missing"), nil
	}
	if copyModeOwnsActiveInput(root) || root.Shell.ReadonlyDefaults().Overlay.Open || root.Shell.ReadonlyDefaults().InteractionMode != state.InteractionModeNormal {
		// 中文说明：batch 是 runtime 基于入队时状态做的优化；真正发送前
		// reducer 仍以当前 UI owner 为准，overlay/copy/prefix mode 不能漏字节到 PTY。
		return root, nil
	}
	target, ok := liveInputTarget(root)
	if !ok || target.AttachPending {
		return root, nil
	}
	if target.Channel == 0 {
		var effect Effect
		root, effect = liveAttachForInputEffect(root, target, msg.Event, msg.Bytes, deps)
		return root, []Effect{effect}
	}
	logTerminalInputRoute(deps, root, terminalInputRouteLog{
		Event:       msg.Event,
		Target:      target,
		Result:      "terminal-batch",
		Bytes:       len(msg.Bytes),
		NeedsAttach: false,
	})
	var operationID string
	root.TerminalViews, operationID = root.TerminalViews.NextTerminalOperation(inputOperationKind(msg.Event), target.ViewID)
	return root, []Effect{terminalSendInputEffect(target, msg.Event, msg.Bytes, operationID, deps)}
}

func terminalSendInputEffect(target liveInputTargetInfo, event input.InputEvent, bytes []byte, operationID string, deps LiveDeps) Effect {
	payload := append([]byte(nil), bytes...)
	return FuncEffect{
		Async:            true,
		SerialKey:        terminalInputSerialKey(target),
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			err := deps.Terminal.SendInput(ctx, port.TerminalInputRequest{
				EndpointID:  target.EndpointID,
				TerminalID:  target.TerminalID,
				Channel:     target.Channel,
				SurfaceID:   target.SurfaceID,
				ViewID:      target.ViewID,
				Event:       event,
				Bytes:       payload,
				Session:     target.Session,
				OperationID: operationID,
			})
			if err != nil {
				logTerminalInputSend(deps, target, event, len(payload), err)
			} else {
				logTerminalInputSendOK(deps, target, event, len(payload))
			}
			return LiveInputResultMsg{
				EndpointID:  target.EndpointID,
				TerminalID:  target.TerminalID,
				ViewID:      target.ViewID,
				Channel:     target.Channel,
				Session:     target.Session,
				Event:       event,
				Bytes:       payload,
				OperationID: operationID,
				Err:         err,
			}
		},
	}
}

func inputOperationKind(event input.InputEvent) string {
	if event.Kind == input.EventKindPaste {
		return "paste"
	}
	return "input"
}

func terminalInputSerialKey(target liveInputTargetInfo) string {
	// 中文说明：PTY input 是有序 byte stream；真实 runtime 可以异步发送，
	// 但同一 terminal/view/channel 不能并发乱序，尤其是 tmux paste/长命令输入。
	return "terminal.input:" + state.NewTerminalRef(target.EndpointID, target.TerminalID).Key() + ":" + target.ViewID + ":" + strconv.FormatUint(uint64(target.Channel), 10)
}

type terminalInputRouteLog struct {
	Event       input.InputEvent
	Target      liveInputTargetInfo
	Result      string
	Reason      string
	Bytes       int
	RawMouse    bool
	NeedsAttach bool
}

func logTerminalInputRoute(deps LiveDeps, root state.Root, entry terminalInputRouteLog) {
	if deps.Logger == nil || !terminalInputTraceEnabled() {
		return
	}
	shell := root.Shell.ReadonlyDefaults()
	deps.Logger.Info("tui-v3 input route",
		"event", terminalInputEventSummary(entry.Event),
		"result", entry.Result,
		"reason", entry.Reason,
		"bytes", entry.Bytes,
		"raw_mouse", entry.RawMouse,
		"needs_attach", entry.NeedsAttach,
		"active_pane", shell.ActivePaneID,
		"active_floating", shell.ActiveFloatingID(),
		"interaction_mode", string(shell.InteractionMode),
		"overlay_open", shell.Overlay.Open,
		"overlay_kind", string(shell.Overlay.Kind),
		"copy_active", root.CopyMode.Active,
		"copy_entering", root.CopyMode.Entering,
		"target_view", entry.Target.ViewID,
		"target_pane", entry.Target.PaneID,
		"target_floating", entry.Target.FloatingID,
		"endpoint_id", string(entry.Target.EndpointID),
		"terminal_id", entry.Target.TerminalID,
		"channel", entry.Target.Channel,
		"surface_id", entry.Target.SurfaceID,
	)
}

func logTerminalInputSend(deps LiveDeps, target liveInputTargetInfo, event input.InputEvent, bytes int, err error) {
	if deps.Logger == nil {
		return
	}
	deps.Logger.Warn("tui-v3 terminal input send failed",
		"event", terminalInputEventSummary(event),
		"bytes", bytes,
		"target_view", target.ViewID,
		"target_pane", target.PaneID,
		"target_floating", target.FloatingID,
		"endpoint_id", string(target.EndpointID),
		"terminal_id", target.TerminalID,
		"channel", target.Channel,
		"surface_id", target.SurfaceID,
		"error", err,
	)
}

func logTerminalInputSendOK(deps LiveDeps, target liveInputTargetInfo, event input.InputEvent, bytes int) {
	if deps.Logger == nil || !terminalInputTraceEnabled() {
		return
	}
	deps.Logger.Info("tui-v3 terminal input sent",
		"event", terminalInputEventSummary(event),
		"bytes", bytes,
		"target_view", target.ViewID,
		"target_pane", target.PaneID,
		"target_floating", target.FloatingID,
		"endpoint_id", string(target.EndpointID),
		"terminal_id", target.TerminalID,
		"channel", target.Channel,
		"surface_id", target.SurfaceID,
	)
}

func terminalInputTraceEnabled() bool {
	return diagnosticsEnabledFromEnv(tuiInputTraceEnv) || diagnosticsEnabledFromEnv(tuiDiagnosticsEnv)
}

func (runtime *AppRuntime) logHostInputEvent(event input.InputEvent, routed Msg) {
	if runtime == nil || runtime.diagnostics == nil || runtime.diagnostics.logger == nil || !terminalInputTraceEnabled() {
		return
	}
	runtime.diagnostics.logger.Info("tui-v3 host input",
		"event", terminalInputEventSummary(event),
		"routed_msg", fmt.Sprintf("%T", routed),
	)
}

func terminalInputEventSummary(event input.InputEvent) string {
	switch event.Kind {
	case input.EventKindKey:
		if event.Key == input.KeyChar {
			return fmt.Sprintf("key:%s ctrl=%t alt=%t shift=%t raw=%q", event.Char, event.Ctrl, event.Alt, event.Shift, event.RawSeq)
		}
		return fmt.Sprintf("key:%s ctrl=%t alt=%t shift=%t raw=%q", event.Key, event.Ctrl, event.Alt, event.Shift, event.RawSeq)
	case input.EventKindMouse:
		return fmt.Sprintf("mouse:%s row=%d col=%d raw=%q", event.Mouse, event.Row, event.Col, event.RawSeq)
	default:
		return string(event.Kind)
	}
}
