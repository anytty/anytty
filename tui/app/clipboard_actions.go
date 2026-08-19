package app

import (
	"context"
	"errors"
	"strings"

	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

type ClipboardActionDeps struct {
	Core      port.CoreClient
	Clipboard port.ClipboardService
	Terminal  port.TerminalService
}

type ClipboardPasteResultMsg struct {
	Text string
	Err  error
}

func (ClipboardPasteResultMsg) isMsg() {}

type ClipboardPasteTextMsg struct {
	Text string
}

func (ClipboardPasteTextMsg) isMsg() {}

func NewClipboardActionReducer(deps ClipboardActionDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case InputMsg:
			if msg.TerminalMousePassthrough || root.Shell.ReadonlyDefaults().Overlay.Open {
				return root, nil
			}
			intent := input.RouteWithOptions(msg.Event, input.RouteOptions{
				Mode:           inputMode(root.Shell.ReadonlyDefaults().InteractionMode),
				CopyModeActive: copyModeOwnsActiveInput(root),
				Shortcuts:      root.Config.Shortcuts,
			})
			if intent.Kind == input.IntentShortcutAction {
				var ok bool
				intent, ok = shortcutIntentForInvocation(intent.Invocation, intent.Event)
				if !ok {
					return root, nil
				}
			}
			return reduceClipboardIntent(root, intent, deps)
		case ShellShortcutActionMsg:
			intent, ok := shortcutIntentForInvocation(msg.Invocation, input.InputEvent{})
			if !ok {
				return root, nil
			}
			return reduceClipboardIntent(root, intent, deps)
		case ClipboardPasteTextMsg:
			return reduceClipboardPasteText(root, deps, msg.Text)
		case ClipboardPasteResultMsg:
			if msg.Err == nil {
				return root, nil
			}
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Paste failed", Body: msg.Err.Error(), DismissAfterTicks: 5})
			return root.Advance(), nil
		default:
			return root, nil
		}
	}
}

func reduceClipboardIntent(root state.Root, intent input.Intent, deps ClipboardActionDeps) (state.Root, []Effect) {
	switch intent.Kind {
	case input.IntentOpenClipboardHistory:
		root.Shell = root.Shell.OpenClipboardHistory()
		return root.Advance(), []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg { return ClipboardStorageLoadRequestMsg{Reason: "open"} }},
		}
	case input.IntentShellAction:
		if intent.Action != input.ShellActionOpenClipboardHistory {
			return root, nil
		}
		root.Shell = root.Shell.OpenClipboardHistory()
		return root.Advance(), []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg { return ClipboardStorageLoadRequestMsg{Reason: "open"} }},
		}
	case input.IntentPasteLastCopy:
		next, effects := reduceClipboardPaste(root, deps, false)
		return next, append([]Effect{handledEffect{}}, effects...)
	case input.IntentPasteClipboard:
		next, effects := reduceClipboardPaste(root, deps, true)
		return next, append([]Effect{handledEffect{}}, effects...)
	default:
		return root, nil
	}
}

func reduceClipboardPaste(root state.Root, deps ClipboardActionDeps, readSystemClipboard bool) (state.Root, []Effect) {
	if readSystemClipboard {
		if deps.Clipboard == nil {
			return setClipboardActionError(root, "clipboard service missing"), nil
		}
		return root, []Effect{FuncEffect{
			Async:            true,
			ForceSyncInTests: true,
			Run: func(ctx context.Context) Msg {
				result, err := deps.Clipboard.Read(ctx)
				if err != nil {
					return ClipboardPasteResultMsg{Err: err}
				}
				if result.Text == "" {
					return ClipboardPasteResultMsg{Err: errors.New("system clipboard is empty")}
				}
				return ClipboardPasteTextMsg{Text: result.Text}
			},
		}}
	}
	text := ""
	if deps.Clipboard != nil {
		text = deps.Clipboard.LastCopy()
	}
	if text == "" {
		text = latestClipboardHistoryText(root.Clipboard)
	}
	if text == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Paste failed", Body: "copy buffer is empty", DismissAfterTicks: 5})
		return root.Advance(), nil
	}
	return beginClipboardPaste(root, deps, text)
}

func reduceClipboardPasteText(root state.Root, deps ClipboardActionDeps, text string) (state.Root, []Effect) {
	if text == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "clipboard history", Body: "no clipboard entry"})
		return root.Advance(), nil
	}
	return beginClipboardPaste(root, deps, text)
}

func beginClipboardPaste(root state.Root, deps ClipboardActionDeps, text string) (state.Root, []Effect) {
	if deps.Terminal == nil {
		return setClipboardActionError(root, "terminal service missing"), nil
	}
	target, ok := liveInputTarget(root)
	if !ok || target.TerminalID == "" || target.Channel == 0 {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "terminal.input", Body: "no terminal bound"})
		return root.Advance(), nil
	}
	var releaseEffects []Effect
	if copyModeInputContext(root.CopyMode) {
		root, releaseEffects = exitCopyModeWithRelease(root, CopyModeDeps{Core: deps.Core})
	}
	root = root.Advance()
	var operationID string
	root.TerminalViews, operationID = root.TerminalViews.NextTerminalOperation("paste", target.ViewID)
	effects := append([]Effect{}, releaseEffects...)
	effects = append(effects, FuncEffect{
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
				Bytes:       encodeTerminalPaste(text, root.Surface.SurfaceForTerminalRef(state.NewTerminalRef(target.EndpointID, target.TerminalID)).Modes),
				Session:     target.Session,
				OperationID: operationID,
			})
			return ClipboardPasteResultMsg{Text: text, Err: err}
		},
	})
	return root, effects
}

func latestClipboardHistoryText(store state.ClipboardStore) string {
	for _, entry := range store.Entries {
		if strings.TrimSpace(entry.Text) != "" {
			return entry.Text
		}
	}
	return ""
}

func setClipboardActionError(root state.Root, message string) state.Root {
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Paste failed", Body: message, DismissAfterTicks: 5})
	return root.Advance()
}
