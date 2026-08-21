package app

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

type CopyModeDeps struct {
	Core      port.CoreClient
	Clipboard port.ClipboardService
	// Logger 是 copy/history 诊断链路的日志出口；truth source 仍是 core 返回的 HistoryWindow，
	// reducer 不能从日志反推分页、合并或渲染状态。
	Logger *slog.Logger
	// Rows 只作为没有 panel content rect 时的 fallback；真实 copy history 请求量必须跟随当前 panel 尺寸。
	Rows int
}

const (
	copyModeHistoryRequestScreens  = 6
	copyModeHistoryPrefetchScreens = 2
	copyModeHistoryMinRequestRows  = 64
	copyModeHistoryMaxRequestRows  = 512
	copyModeHistoryWindowScreens   = 8
	copyModeSearchScanRequestSize  = 256
	copyModeSearchScanPostSize     = 4096
	copyModeSearchScanPostInterval = 50 * time.Millisecond
)

type CopyModeHistoryResultMsg struct {
	Result     port.HistoryResult
	Err        error
	RequestID  state.RequestID
	PaneID     string
	ViewID     string
	EndpointID state.EndpointID
	TerminalID string
}

func (CopyModeHistoryResultMsg) isMsg() {}

type CopyModeEnterViewMsg struct {
	Binding            state.TerminalViewBinding
	Cols               int
	Rows               int
	InitialScrollDelta int
}

func (CopyModeEnterViewMsg) isMsg() {}

type CopyModeMoveCursorMsg struct {
	Position state.CopyPosition
}

func (CopyModeMoveCursorMsg) isMsg() {}

type CopyModeSetMarkMsg struct {
	Position state.CopyPosition
}

func (CopyModeSetMarkMsg) isMsg() {}

type CopyModeCopySelectionMsg struct{}

func (CopyModeCopySelectionMsg) isMsg() {}

type CopyModeCopyResultMsg struct {
	Text         string
	Err          error
	ClipboardErr error
	Commit       bool
	RequestID    state.RequestID
	ViewID       string
	EndpointID   state.EndpointID
	TerminalID   string
	Token        string
}

func (CopyModeCopyResultMsg) isMsg() {}

type CopyModeSearchResultMsg struct {
	Result     port.HistorySearchResult
	Err        error
	RequestID  state.RequestID
	Query      string
	PaneID     string
	ViewID     string
	EndpointID state.EndpointID
	TerminalID string
	Token      string
	Mode       state.HistorySearchMode
}

func (CopyModeSearchResultMsg) isMsg() {}

type CopyModeSearchScanBatchMsg struct {
	Err        error
	Query      string
	Mode       state.HistorySearchMode
	ViewID     string
	EndpointID state.EndpointID
	TerminalID string
	Token      string
	Matches    []port.HistorySearchMatch
	Done       bool
}

func (CopyModeSearchScanBatchMsg) isMsg() {}

type CopyModeReleaseHistoryMsg struct {
	EndpointID state.EndpointID
	TerminalID string
	Token      string
}

func (CopyModeReleaseHistoryMsg) isMsg() {}

type CopyModeScrollMsg struct {
	Delta  int
	ViewID string
}

func (CopyModeScrollMsg) isMsg() {}

type CopyModeMouseSelectMsg struct {
	Position state.CopyPosition
	PaneID   string
	ViewID   string
}

func (CopyModeMouseSelectMsg) isMsg() {}

type CopyModeWheelMsg struct {
	Event  input.InputEvent
	ViewID string
}

func (CopyModeWheelMsg) isMsg() {}

func staleCopyModeHistoryResult(root state.Root, msg CopyModeHistoryResultMsg) bool {
	return root.History.Pending == nil || copyModeHistoryResultRequestID(msg) != root.History.Pending.ID
}

func copyModeHistoryResultRequestID(msg CopyModeHistoryResultMsg) state.RequestID {
	if msg.RequestID != 0 {
		return msg.RequestID
	}
	return state.RequestID(msg.Result.RequestID)
}

func rootWithActiveCopyHistorySession(root state.Root) (state.Root, string) {
	viewID := ""
	if binding, ok := activeTerminalViewBinding(root); ok {
		viewID = binding.ViewID
	}
	if viewID == "" {
		viewID = copyHistoryWorkingViewID(root)
	}
	if workingViewID := copyHistoryWorkingViewID(root); workingViewID != "" {
		if viewID == "" || workingViewID == viewID || root.CopyModeByView[viewID].ViewID == "" && root.HistoryByView[viewID].ViewID == "" {
			return root, workingViewID
		}
	}
	if copyModeInputContext(root.CopyMode) && root.CopyMode.ViewID == "" && root.CopyMode.PaneID == "" {
		return root, viewID
	}
	return rootWithCopyHistorySessionForView(root, viewID), viewID
}

func rootWithCopyHistorySessionForResult(root state.Root, msg CopyModeHistoryResultMsg) (state.Root, string) {
	viewID := copyHistoryResultViewID(root, msg)
	if viewID == "" {
		return root, copyHistoryWorkingViewID(root)
	}
	return rootWithCopyHistorySessionForView(root, viewID), viewID
}

func rootWithCopyHistorySessionForView(root state.Root, viewID string) state.Root {
	if viewID == "" {
		return root
	}
	history, copyMode := root.CopyHistorySessionForView(viewID)
	root.History = history
	root.CopyMode = copyMode
	return root
}

func saveCopyHistorySessionForView(root state.Root, viewID string) state.Root {
	if viewID == "" {
		viewID = copyHistoryWorkingViewID(root)
	}
	if viewID == "" {
		return root
	}
	// 中文说明：reducer 仍复用单会话算法，提交时按 TerminalView 写回，
	// 避免 floating/pane 的 history 交互态互相覆盖。
	return root.WithCopyHistorySession(viewID, root.History, root.CopyMode)
}

func copyHistoryWorkingViewID(root state.Root) string {
	if root.CopyMode.ViewID != "" {
		return root.CopyMode.ViewID
	}
	if root.History.ViewID != "" {
		return root.History.ViewID
	}
	if root.CopyMode.PaneID != "" {
		return state.TerminalPaneViewID(root.CopyMode.PaneID)
	}
	if root.History.PaneID != "" {
		return state.TerminalPaneViewID(root.History.PaneID)
	}
	return ""
}

func copyHistoryResultViewID(root state.Root, msg CopyModeHistoryResultMsg) string {
	requestID := copyModeHistoryResultRequestID(msg)
	if requestID != 0 {
		if viewID := copyHistoryPendingViewID(root.History, root.CopyMode, requestID); viewID != "" {
			return viewID
		}
		for viewID, history := range root.HistoryByView {
			copyMode := root.CopyModeByView[viewID]
			if pendingViewID := copyHistoryPendingViewID(history, copyMode, requestID); pendingViewID != "" {
				return pendingViewID
			}
		}
		return ""
	}
	if msg.ViewID != "" {
		return msg.ViewID
	}
	if msg.Result.Window.ViewID != "" {
		return msg.Result.Window.ViewID
	}
	if msg.PaneID != "" {
		return state.TerminalPaneViewID(msg.PaneID)
	}
	return ""
}

func copyHistoryPendingViewID(history state.HistoryStore, copyMode state.CopyModeStore, requestID state.RequestID) string {
	if history.Pending == nil || history.Pending.ID != requestID {
		return ""
	}
	if history.Pending.ViewID != "" {
		return history.Pending.ViewID
	}
	if copyMode.ViewID != "" {
		return copyMode.ViewID
	}
	if history.ViewID != "" {
		return history.ViewID
	}
	if history.Pending.PaneID != "" {
		return state.TerminalPaneViewID(history.Pending.PaneID)
	}
	return ""
}

func NewCopyModeReducer(deps CopyModeDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		switch msg := msg.(type) {
		case InputMsg:
			if msg.TerminalMousePassthrough {
				return root, nil
			}
			if root.Shell.ReadonlyDefaults().TerminalInputPassthroughArmed() {
				return root, nil
			}
			root, activeViewID := rootWithActiveCopyHistorySession(root)
			copyOwnsInput := copyModeOwnsActiveInput(root)
			textInputActive := copyOwnsInput && root.CopyMode.SearchEditing
			if copyOwnsInput && !textInputActive {
				if _, ok := input.CopyModeEntryShortcutIntentWithShortcuts(msg.Event, root.Config.Shortcuts); ok {
					if root.Shell.ReadonlyDefaults().ShortcutPassthroughWindowMatches(shortcutPassthroughKindCopy) {
						// 中文说明：copy/history 的入口键本身不是 sticky mode；
						// 但在窗口内第二次按入口键表示显式 PTY 透传，必须让同一 InputMsg 继续到 terminal router。
						next, effects := exitCopyModeWithRelease(root, deps)
						next.Shell = next.Shell.ClearShortcutPassthroughWindow(shortcutPassthroughKindCopy).ArmTerminalInputPassthroughOnce()
						return saveCopyHistorySessionForView(next.Advance(), activeViewID), effects
					}
					return root, []Effect{handledEffect{}}
				}
			}
			intent := input.RouteWithOptions(msg.Event, input.RouteOptions{
				CopyModeActive:  copyOwnsInput,
				TextInputActive: textInputActive,
				Shortcuts:       root.Config.Shortcuts,
			})
			if intent.Kind == input.IntentShortcutAction {
				var ok bool
				intent, ok = shortcutIntentForInvocation(intent.Invocation, intent.Event)
				if !ok {
					return root, []Effect{handledEffect{}}
				}
			}
			if !copyOwnsInput && !copyModeEnterIntent(intent) {
				return root, nil
			}
			next, effects := reduceCopyModeIntent(root, intent, deps)
			if copyModeEnterIntent(intent) {
				activeViewID = copyHistoryWorkingViewID(next)
			}
			return saveCopyHistorySessionForView(next, activeViewID), effects
		case ShellShortcutActionMsg:
			intent, ok := shortcutIntentForInvocation(msg.Invocation, input.InputEvent{})
			if !ok || !shortcutIntentOwnedByCopy(intent) {
				return root, nil
			}
			root, activeViewID := rootWithActiveCopyHistorySession(root)
			next, effects := reduceCopyModeIntent(root, intent, deps)
			if copyModeEnterIntent(intent) {
				activeViewID = copyHistoryWorkingViewID(next)
			}
			return saveCopyHistorySessionForView(next, activeViewID), effects
		case CopyModeEnterViewMsg:
			next, effects := beginCopyModeLatestForView(root, deps, msg.Binding, msg.Cols, msg.Rows)
			next = applyCopyModeEnteringScrollDelta(next, msg.InitialScrollDelta)
			next = saveCopyHistorySessionForView(next, msg.Binding.ViewID)
			return next, append([]Effect{handledEffect{}}, effects...)
		case CopyModeHistoryResultMsg:
			root, viewID := rootWithCopyHistorySessionForResult(root, msg)
			next, effects := reduceCopyModeHistoryResult(root, msg, deps)
			return saveCopyHistorySessionForView(next, viewID), effects
		case CopyModeMoveCursorMsg:
			root, activeViewID := rootWithActiveCopyHistorySession(root)
			if !root.CopyMode.CanSelect() {
				return saveCopyHistorySessionForView(root, activeViewID), nil
			}
			root.CopyMode = root.CopyMode.MoveCursor(msg.Position).RefreshLogicalSelectionFocus(root.History)
			root = root.Advance()
			return saveCopyHistorySessionForView(root, activeViewID), nil
		case CopyModeSetMarkMsg:
			root, activeViewID := rootWithActiveCopyHistorySession(root)
			if !root.CopyMode.CanSelect() {
				return saveCopyHistorySessionForView(root, activeViewID), nil
			}
			root.CopyMode = root.CopyMode.SetMark(msg.Position).RefreshLogicalSelection(root.History)
			root = root.Advance()
			return saveCopyHistorySessionForView(root, activeViewID), nil
		case CopyModeCopySelectionMsg:
			root, activeViewID := rootWithActiveCopyHistorySession(root)
			if !root.CopyMode.CopyPending {
				root.CopyMode.CopyExitAfterSuccess = false
			}
			next, effects := reduceCopyModeCopySelection(root, deps)
			return saveCopyHistorySessionForView(next, activeViewID), effects
		case CopyModeCopyResultMsg:
			viewID := msg.ViewID
			if viewID == "" {
				root, viewID = rootWithActiveCopyHistorySession(root)
			} else {
				root = rootWithCopyHistorySessionForView(root, viewID)
			}
			if !root.CopyMode.Active || !root.CopyMode.CopyPending ||
				root.CopyMode.CopyRequestID != msg.RequestID ||
				(msg.EndpointID != "" && root.CopyMode.EndpointID != msg.EndpointID) ||
				(msg.TerminalID != "" && root.CopyMode.TerminalID != msg.TerminalID) ||
				(msg.Token != "" && root.CopyMode.BoundToken != msg.Token) {
				return saveCopyHistorySessionForView(root, viewID), nil
			}
			exitAfterSuccess := root.CopyMode.CopyExitAfterSuccess
			root.CopyMode.CopyPending = false
			root.CopyMode.CopyExitAfterSuccess = false
			if msg.Err != nil {
				title := "Copy failed"
				body := msg.Err.Error()
				if errors.Is(msg.Err, port.ErrHistoryCopyTooLarge) {
					title = "Selection is too large"
					body = "Select a smaller range and copy again"
				} else if errors.Is(msg.Err, port.ErrStaleHistoryWindow) {
					title = "History window expired"
					body = "Exit and reopen copy mode before retrying"
				} else if errors.Is(msg.Err, port.ErrHistoryResourceExhausted) {
					title = "History is temporarily unavailable"
					body = "Retry the copy in a moment"
				}
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: title, Body: body, DismissAfterTicks: 5})
				return saveCopyHistorySessionForView(root.Advance(), viewID), nil
			}
			if !msg.Commit {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Nothing to copy", DismissAfterTicks: 3})
				return saveCopyHistorySessionForView(root.Advance(), viewID), nil
			}
			var effects []Effect
			if msg.Text != "" {
				var stored bool
				root.Clipboard, stored = root.Clipboard.WithCopiedTextLimit(msg.Text, clipboardHistoryMaxItems(root))
				if stored {
					effects = append(effects, FuncEffect{Run: func(context.Context) Msg { return ClipboardStoragePersistRequestMsg{Reason: "copy"} }})
				}
			}
			if msg.ClipboardErr != nil {
				root.Shell = root.Shell.AddToast(state.ToastSpec{
					Severity:          state.ToastWarning,
					Title:             "Copied to buffer",
					Body:              "System clipboard unavailable: " + msg.ClipboardErr.Error(),
					DismissAfterTicks: 5,
				})
			} else {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastSuccess, Title: "Copied to clipboard", DismissAfterTicks: 3})
			}
			root = root.Advance()
			if exitAfterSuccess {
				root, releaseEffects := exitCopyModeWithRelease(root, deps)
				effects = append(effects, releaseEffects...)
				return root, effects
			}
			return saveCopyHistorySessionForView(root, viewID), effects
		case CopyModeSearchResultMsg:
			activeViewID := msg.ViewID
			if activeViewID == "" {
				root, activeViewID = rootWithActiveCopyHistorySession(root)
			} else {
				root = rootWithCopyHistorySessionForView(root, activeViewID)
			}
			next, effects := reduceCopyModeSearchResult(root, msg, deps)
			return saveCopyHistorySessionForView(next, activeViewID), effects
		case CopyModeSearchScanBatchMsg:
			activeViewID := msg.ViewID
			if activeViewID == "" {
				root, activeViewID = rootWithActiveCopyHistorySession(root)
			} else {
				root = rootWithCopyHistorySessionForView(root, activeViewID)
			}
			next := reduceCopyModeSearchScanBatch(root, msg)
			return saveCopyHistorySessionForView(next, activeViewID), nil
		case CopyModeReleaseHistoryMsg:
			return root, releaseHistoryTokenEffects(deps, msg.EndpointID, msg.TerminalID, msg.Token)
		case CopyModeScrollMsg:
			activeViewID := msg.ViewID
			if activeViewID != "" {
				root = rootWithCopyHistorySessionForView(root, activeViewID)
			} else {
				root, activeViewID = rootWithActiveCopyHistorySession(root)
			}
			if !root.CopyMode.CanSelect() {
				return saveCopyHistorySessionForView(root, activeViewID), nil
			}
			if msg.Delta == 0 {
				return saveCopyHistorySessionForView(root, activeViewID), nil
			}
			var effects []Effect
			if msg.Delta < 0 {
				root, effects = reduceCopyModeScrollOlderRows(root, deps, -msg.Delta)
			} else {
				root, effects = reduceCopyModeScrollNewer(root, deps, msg.Delta)
			}
			return saveCopyHistorySessionForView(root, activeViewID), effects
		case CopyModeMouseSelectMsg:
			root = rootWithCopyHistorySessionForView(root, msg.ViewID)
			if !copyModeMouseSelectTargetMatches(root, msg.PaneID) {
				return root, nil
			}
			if !root.CopyMode.CanSelect() {
				return saveCopyHistorySessionForView(root, msg.ViewID), nil
			}
			root.CopyMode = root.CopyMode.MoveCursor(msg.Position)
			root.CopyMode = clampCopyCursor(root.CopyMode, root.History)
			root.CopyMode = root.CopyMode.SetMark(root.CopyMode.Cursor)
			root.CopyMode = ensureCopyCursorVisible(root.CopyMode, len(root.History.Rows))
			root.CopyMode = root.CopyMode.RefreshLogicalSelection(root.History)
			root = root.Advance()
			return saveCopyHistorySessionForView(root, msg.ViewID), nil
		case CopyModeWheelMsg:
			root = rootWithCopyHistorySessionForView(root, msg.ViewID)
			if !copyModeInputContext(root.CopyMode) {
				return root, nil
			}
			intent := input.RouteWithOptions(msg.Event, input.RouteOptions{CopyModeActive: true, Shortcuts: root.Config.Shortcuts})
			next, effects := reduceCopyModeIntent(root, intent, deps)
			return saveCopyHistorySessionForView(next, msg.ViewID), effects
		default:
			return root, nil
		}
	}
}

func copyModeMouseSelectTargetMatches(root state.Root, paneID string) bool {
	if paneID == "" {
		return true
	}
	copyMode := root.CopyMode
	if copyMode.PaneID == paneID || copyMode.ViewID == root.TerminalViews.FloatingViewID(paneID) {
		return true
	}
	for _, binding := range root.TerminalViews.Bindings() {
		if binding.PaneID == paneID && binding.ViewID == copyMode.ViewID {
			return true
		}
	}
	shell := root.Shell.ReadonlyDefaults()
	return copyMode.PaneID == "" && copyMode.ViewID == "" && paneID == shell.ActivePaneID && shell.ActiveFloatingID() == ""
}

func refreshCopyModeLogicalSelectionFocus(root state.Root) state.Root {
	root.CopyMode = root.CopyMode.RefreshLogicalSelectionFocus(root.History)
	return root
}

func copyModeInputContext(copyMode state.CopyModeStore) bool {
	return copyMode.InputActive()
}

func copyModeOwnsActiveInput(root state.Root) bool {
	return root.ActiveViewOwnsCopyInput()
}

func copyModeTextInputOwnsActiveInput(root state.Root) bool {
	root, _ = rootWithActiveCopyHistorySession(root)
	return copyModeOwnsActiveInput(root) && root.CopyMode.SearchEditing
}

func reduceCopyModeIntent(root state.Root, intent input.Intent, deps CopyModeDeps) (state.Root, []Effect) {
	if intent.Kind == input.IntentShortcutAction {
		var ok bool
		intent, ok = shortcutIntentForInvocation(intent.Invocation, intent.Event)
		if !ok {
			return root, []Effect{handledEffect{}}
		}
	}
	if next, effects, handled := reduceCopyModeEnteringIntent(root, intent); handled {
		return next, effects
	}
	switch intent.Kind {
	case input.IntentEnterCopyMode:
		next, effects := beginCopyModeLatest(root, deps)
		if delta, ok := copyModeEnteringScrollDelta(next.CopyMode, intent); ok {
			next = applyCopyModeEnteringScrollDelta(next, delta)
		}
		if next.CopyMode.InputActive() {
			next, effects = armShortcutPassthroughWindow(next, shortcutPassthroughKindCopy, effects)
		}
		return next, append([]Effect{handledEffect{}}, effects...)
	case input.IntentRequestOlder:
		if !root.CopyMode.CanSelect() {
			return root, []Effect{handledEffect{}}
		}
		next, effects := reduceCopyModeScrollOlder(root, deps, intent.Event)
		return next, append([]Effect{handledEffect{}}, effects...)
	case input.IntentRequestNewer:
		if !root.CopyMode.CanSelect() {
			return root, []Effect{handledEffect{}}
		}
		rows := copyModeNewerScrollRows(root.CopyMode, intent.Event)
		next, effects := reduceCopyModeScrollNewer(root, deps, rows)
		return next, append([]Effect{handledEffect{}}, effects...)
	case input.IntentCopyCommand:
		next, effects := reduceCopyModeCommand(root, intent.Command, deps)
		return next, append([]Effect{handledEffect{}}, effects...)
	case input.IntentMouseSelect:
		if !root.CopyMode.CanSelect() {
			return root, []Effect{handledEffect{}}
		}
		root.CopyMode = root.CopyMode.SetMark(root.CopyMode.Cursor)
		root.CopyMode = root.CopyMode.RefreshLogicalSelection(root.History)
		return root.Advance(), []Effect{handledEffect{}}
	default:
		if root.CopyMode.Active {
			next, mouseEffects, handled := reduceCopyModeMouseInput(root, intent.Event, deps)
			if handled {
				return next, append([]Effect{handledEffect{}}, mouseEffects...)
			}
			next, keyEffects, handled := reduceCopyModeTextInput(root, intent.Event, deps)
			if handled {
				return next, append([]Effect{handledEffect{}}, keyEffects...)
			}
		}
		if root.CopyMode.Entering {
			return root, []Effect{handledEffect{}}
		}
		return root, nil
	}
}

func copyModeEnterIntent(intent input.Intent) bool {
	return intent.Kind == input.IntentEnterCopyMode
}

func reduceCopyModeEnteringIntent(root state.Root, intent input.Intent) (state.Root, []Effect, bool) {
	if !root.CopyMode.Entering {
		return root, nil, false
	}
	if intent.Kind == input.IntentOpenClipboardHistory {
		root.Shell = root.Shell.OpenClipboardHistory()
		return root.Advance(), []Effect{
			handledEffect{},
			FuncEffect{Run: func(context.Context) Msg { return ClipboardStorageLoadRequestMsg{Reason: "open"} }},
		}, true
	}
	if delta, ok := copyModeEnteringScrollDelta(root.CopyMode, intent); ok {
		root.CopyMode.EnteringScrollDelta += delta
		return root.Advance(), []Effect{handledEffect{}}, true
	}
	// 中文说明：latest 还没回来时 copy/history 尚未真正激活；这段时间只拦截输入，
	// 防止 p/P 或普通按键落到 terminal，也不提前打开依赖 frozen history 的功能。
	return root, []Effect{handledEffect{}}, true
}

func copyModeEnteringScrollDelta(copyMode state.CopyModeStore, intent input.Intent) (int, bool) {
	switch intent.Kind {
	case input.IntentEnterCopyMode:
		if intent.Event.Kind == input.EventKindMouse && intent.Event.Mouse == input.MouseWheelUp {
			return -copyModeLineScrollRows(), true
		}
		return 0, false
	case input.IntentRequestOlder:
		rows := copyModeOlderScrollRows(copyMode, intent.Event)
		if rows <= 0 {
			rows = 1
		}
		return -rows, true
	case input.IntentRequestNewer:
		rows := copyModeNewerScrollRows(copyMode, intent.Event)
		if rows <= 0 {
			rows = 1
		}
		return rows, true
	default:
		if intent.Event.Kind == input.EventKindMouse && intent.Event.Mouse == input.MouseWheelDown {
			return copyModeLineScrollRows(), true
		}
		if intent.Event.Kind == input.EventKindKey && intent.Event.Key == input.KeyPageDn {
			return copyModePageRows(copyMode), true
		}
		return 0, false
	}
}

func applyCopyModeEnteringScrollDelta(root state.Root, delta int) state.Root {
	if delta == 0 || !root.CopyMode.Entering {
		return root
	}
	root.CopyMode.EnteringScrollDelta += delta
	return root.Advance()
}

func reduceCopyModeMouseInput(root state.Root, event input.InputEvent, deps CopyModeDeps) (state.Root, []Effect, bool) {
	if event.Kind != input.EventKindMouse {
		return root, nil, false
	}
	switch event.Mouse {
	case input.MouseWheelUp:
		next, effects := reduceCopyModeScrollOlderRows(root, deps, copyModeLineScrollRows())
		return next, effects, true
	case input.MouseWheelDown:
		next, effects := reduceCopyModeScrollNewer(root, deps, copyModeLineScrollRows())
		return next, effects, true
	default:
		return root, nil, false
	}
}

func reduceCopyModeScrollNewer(root state.Root, deps CopyModeDeps, rows int) (state.Root, []Effect) {
	if rows <= 0 {
		rows = 1
	}
	if !root.CopyMode.Active {
		return root, nil
	}
	previousCopyMode := root.CopyMode
	var consumedRows int
	root.CopyMode, consumedRows = root.CopyMode.ScrollNewer(rows, len(root.History.Rows))
	unconsumedRows := rows - consumedRows
	if unconsumedRows < 0 {
		unconsumedRows = 0
	}
	if root.CopyMode.AtFrozenBottom(root.History) && root.CopyMode.Mark == nil && root.CopyMode.Selection == nil {
		next, effects := exitCopyModeWithRelease(root, deps)
		return next.Advance(), effects
	}
	if unconsumedRows > 0 && root.History.Pending != nil && root.History.Pending.Kind == state.HistoryRequestNewer {
		root.History.Pending.DeferredScrollRows += unconsumedRows
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	}
	if unconsumedRows == 0 || !root.CopyMode.CanPageHistory() || root.History.NewerRequestState() != state.NewerRequestReady {
		if root.CopyMode.Cursor != previousCopyMode.Cursor || root.CopyMode.ViewportTop != previousCopyMode.ViewportTop {
			return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
		}
		return root, nil
	}
	if root.CopyMode.Cursor != previousCopyMode.Cursor || root.CopyMode.ViewportTop != previousCopyMode.ViewportTop {
		root = refreshCopyModeLogicalSelectionFocus(root)
	}
	next, effects := beginCopyModeNewer(root, deps, unconsumedRows)
	if len(effects) > 0 {
		return next, effects
	}
	if root.CopyMode.Cursor != previousCopyMode.Cursor || root.CopyMode.ViewportTop != previousCopyMode.ViewportTop {
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	}
	return root, nil
}

func reduceCopyModeScrollOlder(root state.Root, deps CopyModeDeps, event input.InputEvent) (state.Root, []Effect) {
	rows := copyModeOlderScrollRows(root.CopyMode, event)
	if rows <= 0 {
		rows = 1
	}
	return reduceCopyModeScrollOlderRows(root, deps, rows)
}

func reduceCopyModeScrollOlderRows(root state.Root, deps CopyModeDeps, rows int) (state.Root, []Effect) {
	if rows <= 0 {
		rows = 1
	}
	if root.CopyMode.Active {
		previousCopyMode := root.CopyMode
		root.CopyMode = root.CopyMode.ScrollCursor(-rows, len(root.History.Rows))
		consumedRows := previousCopyMode.Cursor.Row - root.CopyMode.Cursor.Row
		unconsumedRows := rows - consumedRows
		if unconsumedRows < 0 {
			unconsumedRows = 0
		}
		if root.CopyMode.Cursor != previousCopyMode.Cursor || root.CopyMode.ViewportTop != previousCopyMode.ViewportTop {
			root = refreshCopyModeLogicalSelectionFocus(root)
		}
		next, effects := maybePrefetchCopyModeOlder(root, deps, unconsumedRows)
		if len(effects) > 0 {
			return next, effects
		}
		if root.History.Pending != nil && root.History.Pending.Kind == state.HistoryRequestOlder {
			// 本地已加载区域先消费 cursor 移动；跨过顶部但 older 仍在飞时，只把没消费的行数挂到 pending。
			root.History.Pending.DeferredScrollRows += unconsumedRows
			return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
		}
		if root.CopyMode.Cursor != previousCopyMode.Cursor || root.CopyMode.ViewportTop != previousCopyMode.ViewportTop {
			return root.Advance(), nil
		}
		rows = unconsumedRows
	}
	if rows <= 0 {
		return root, nil
	}
	if !root.CopyMode.CanPageHistory() {
		return root, nil
	}
	return beginCopyModeOlder(root, deps, rows)
}

func reduceCopyModeCommand(root state.Root, command string, deps CopyModeDeps) (state.Root, []Effect) {
	switch command {
	case "copy.line_start":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		root.CopyMode = root.CopyMode.MoveCursor(state.CopyPosition{Row: root.CopyMode.Cursor.Row, Col: 0})
		root.CopyMode = clampCopyCursor(root.CopyMode, root.History)
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	case "copy.line_end":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		root.CopyMode = root.CopyMode.MoveCursor(copyModeLineEndPosition(root.History, root.CopyMode.Cursor.Row))
		root.CopyMode = clampCopyCursor(root.CopyMode, root.History)
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	case "copy.cursor_left":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		root.CopyMode = root.CopyMode.MoveCursor(state.CopyPosition{Row: root.CopyMode.Cursor.Row, Col: root.CopyMode.Cursor.Col - 1})
		root.CopyMode = clampCopyCursor(root.CopyMode, root.History)
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	case "copy.cursor_right":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		root.CopyMode = root.CopyMode.MoveCursor(state.CopyPosition{Row: root.CopyMode.Cursor.Row, Col: root.CopyMode.Cursor.Col + 1})
		root.CopyMode = clampCopyCursor(root.CopyMode, root.History)
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	case "copy.cursor_down":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		return reduceCopyModeScrollNewer(root, deps, 1)
	case "copy.cursor_up":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		return reduceCopyModeScrollOlderRows(root, deps, 1)
	case "copy.accept":
		if root.CopyMode.SearchEditing {
			root.CopyMode.SearchEditing = false
			if root.CopyMode.Query == "" {
				return root.Advance(), nil
			}
			return beginCopyModeSearch(root.Advance(), deps, port.HistorySearchForward)
		}
		if root.CopyMode.Selection != nil {
			if root.CopyMode.CopyPending {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "Copy in progress", DismissAfterTicks: 2})
				return root.Advance(), nil
			}
			root.CopyMode.CopyExitAfterSuccess = true
			return reduceCopyModeCopySelection(root, deps)
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Nothing to copy", Body: "Select text before pressing Enter", DismissAfterTicks: 3})
		return root.Advance(), nil
	case "copy.oldest":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		root.CopyMode = root.CopyMode.MoveCursor(state.CopyPosition{Row: 0, Col: root.CopyMode.Cursor.Col})
		root.CopyMode = clampCopyCursor(root.CopyMode, root.History)
		root.CopyMode = ensureCopyCursorVisible(root.CopyMode, len(root.History.Rows))
		// `g` 在 copy mode 里表达“去最老处”。这里直接请求 frozen snapshot
		// 的 oldest page，不能靠重复 older 把中间所有页都拉进 TUI。
		if root.CopyMode.CanPageHistory() && root.CopyMode.ViewportTop == 0 && root.History.OlderRequestState() == state.OlderRequestReady {
			next, effects := beginCopyModeOldest(root, deps)
			return next, effects
		}
		return refreshCopyModeLogicalSelectionFocus(root).Advance(), nil
	case "copy.newest":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		next, effects := exitCopyModeWithRelease(root, deps)
		return next.Advance(), effects
	case "copy.half_page_older":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		return reduceCopyModeScrollOlderRows(root, deps, copyModePageRows(root.CopyMode)/2)
	case "copy.half_page_newer":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		return reduceCopyModeScrollNewer(root, deps, copyModePageRows(root.CopyMode)/2)
	case "copy.mark":
		if !root.CopyMode.CanSelect() {
			return root, nil
		}
		if root.CopyMode.Mark != nil || root.CopyMode.Selection != nil {
			root.CopyMode = root.CopyMode.ClearSelection()
			return root.Advance(), nil
		}
		root.CopyMode = root.CopyMode.SetMark(root.CopyMode.Cursor)
		root.CopyMode = root.CopyMode.RefreshLogicalSelection(root.History)
		return root.Advance(), nil
	case "copy.copy_selection":
		if root.CopyMode.Selection != nil {
			if root.CopyMode.CopyPending {
				root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "Copy in progress", DismissAfterTicks: 2})
				return root.Advance(), nil
			}
			root.CopyMode.CopyExitAfterSuccess = false
			return reduceCopyModeCopySelection(root, deps)
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Nothing to copy", Body: "Select text before copying", DismissAfterTicks: 3})
		return root.Advance(), nil
	case "copy.search_start":
		if !root.CopyMode.CanSearch() {
			return root, nil
		}
		wasPending := root.CopyMode.SearchPending
		wasScanning := root.CopyMode.SearchScanPending
		if root.CopyMode.Query == "" {
			root.CopyMode = root.CopyMode.SetQuery("", nil)
			root.CopyMode.SearchWrapped = false
		}
		root.CopyMode = root.CopyMode.SetSearchCursor(len([]rune(root.CopyMode.Query)))
		root.CopyMode.SearchEditing = true
		root.CopyMode.SearchPending = false
		root.CopyMode.SearchScanPending = false
		return root.Advance(), copyModeSearchCancelEffects(root.CopyMode.ViewID, wasPending, wasScanning)
	case "copy.search_mode":
		if !root.CopyMode.CanSearch() || !root.CopyMode.SearchBarVisible() {
			return root, nil
		}
		wasPending := root.CopyMode.SearchPending
		wasScanning := root.CopyMode.SearchScanPending
		wasEditing := root.CopyMode.SearchEditing
		root.CopyMode = root.CopyMode.CycleSearchMode()
		if !wasEditing {
			root.CopyMode = root.CopyMode.SetSearchCursor(len([]rune(root.CopyMode.Query)))
		}
		root.CopyMode.SearchEditing = true
		root.CopyMode.SearchPending = false
		return root.Advance(), copyModeSearchCancelEffects(root.CopyMode.ViewID, wasPending, wasScanning)
	case "copy.search_next":
		return beginCopyModeSearch(root, deps, port.HistorySearchForward)
	case "copy.search_previous":
		return beginCopyModeSearch(root, deps, port.HistorySearchBackward)
	default:
		return root, nil
	}
}

func reduceCopyModeTextInput(root state.Root, event input.InputEvent, deps CopyModeDeps) (state.Root, []Effect, bool) {
	if !root.CopyMode.SearchEditing {
		return root, nil, false
	}
	if event.Kind == input.EventKindPaste {
		text := normalizeCopyModeSearchPaste(event.Paste)
		if text == "" {
			return root, nil, true
		}
		return mutateCopyModeSearchQuery(root, func(store state.CopyModeStore) state.CopyModeStore {
			return store.InsertSearchText(text)
		})
	}
	if event.Kind != input.EventKindKey {
		return root, nil, false
	}
	if isBackspaceEvent(event) {
		return mutateCopyModeSearchQuery(root, func(store state.CopyModeStore) state.CopyModeStore {
			return store.DeleteSearchBackward()
		})
	}
	switch event.Key {
	case input.KeyDelete:
		return mutateCopyModeSearchQuery(root, func(store state.CopyModeStore) state.CopyModeStore {
			return store.DeleteSearchForward()
		})
	case input.KeyLeft:
		root.CopyMode = root.CopyMode.MoveSearchCursor(-1)
		return root.Advance(), nil, true
	case input.KeyRight:
		root.CopyMode = root.CopyMode.MoveSearchCursor(1)
		return root.Advance(), nil, true
	case input.KeyHome:
		root.CopyMode = root.CopyMode.SetSearchCursor(0)
		return root.Advance(), nil, true
	case input.KeyEnd:
		root.CopyMode = root.CopyMode.SetSearchCursor(len([]rune(root.CopyMode.Query)))
		return root.Advance(), nil, true
	case input.KeyChar:
		if event.Ctrl || event.Alt || event.Char == "" {
			return root, nil, false
		}
		return mutateCopyModeSearchQuery(root, func(store state.CopyModeStore) state.CopyModeStore {
			return store.InsertSearchText(event.Char)
		})
	default:
		return root, nil, false
	}
}

func mutateCopyModeSearchQuery(root state.Root, update func(state.CopyModeStore) state.CopyModeStore) (state.Root, []Effect, bool) {
	before := root.CopyMode.Query
	wasPending := root.CopyMode.SearchPending
	wasScanning := root.CopyMode.SearchScanPending
	root.CopyMode = update(root.CopyMode)
	root.CopyMode.SearchEditing = true
	if root.CopyMode.Query == before {
		return root, nil, true
	}
	return root.Advance(), copyModeSearchCancelEffects(root.CopyMode.ViewID, wasPending, wasScanning), true
}

func normalizeCopyModeSearchPaste(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Map(func(value rune) rune {
		switch value {
		case '\r', '\n', '\t':
			return ' '
		case '\x7f':
			return -1
		default:
			if value < ' ' {
				return -1
			}
			return value
		}
	}, text)
}

func beginCopyModeSearch(root state.Root, deps CopyModeDeps, direction port.HistorySearchDirection) (state.Root, []Effect) {
	if deps.Core == nil || !root.CopyMode.CanSearch() || root.CopyMode.Query == "" {
		return root, nil
	}
	start := state.CopyLogicalPositionForPosition(root.History, root.CopyMode.Cursor)
	if root.CopyMode.SearchMatchStart.Valid {
		start = root.CopyMode.SearchMatchStart
		if direction == port.HistorySearchForward {
			start.Col++
		}
	}
	requestID := nextHistoryRequestID(root)
	req := port.HistorySearchRequest{
		EndpointID: root.CopyMode.EndpointID,
		RequestID:  port.RequestID(requestID),
		TerminalID: root.CopyMode.TerminalID,
		Cols:       root.History.Cols,
		Rows:       copyModeHistoryRequestRows(root, deps),
		Token:      root.CopyMode.BoundToken,
		Generation: root.History.Generation,
		Query:      root.CopyMode.Query,
		Mode:       root.CopyMode.NormalizedSearchMode(),
		Direction:  direction,
		Start:      start,
	}
	root.CopyMode.SearchPending = true
	root.CopyMode.SearchError = ""
	root.CopyMode.SearchRequestID = state.RequestID(requestID)
	root.CopyMode.SearchEditing = false
	viewID := root.CopyMode.ViewID
	paneID := root.CopyMode.PaneID
	query := root.CopyMode.Query
	var scanEffects []Effect
	root, scanEffects = beginCopyModeSearchScan(root, deps)
	root = root.Advance()
	token := copyModeSearchRequestToken(viewID)
	effects := []Effect{
		FuncEffect{
			Token:            token,
			Async:            true,
			ForceSyncInTests: true,
			Run: func(ctx context.Context) Msg {
				result, err := deps.Core.HistorySearch(ctx, req)
				result.Window.PaneID = paneID
				result.Window.ViewID = viewID
				result.Window.EndpointID = req.EndpointID
				return CopyModeSearchResultMsg{
					Result: result, Err: err, RequestID: state.RequestID(requestID), Query: query, PaneID: paneID, ViewID: viewID,
					EndpointID: req.EndpointID, TerminalID: req.TerminalID, Token: req.Token, Mode: req.Mode,
				}
			},
		},
	}
	return root, append(effects, scanEffects...)
}

func reduceCopyModeSearchResult(root state.Root, msg CopyModeSearchResultMsg, deps CopyModeDeps) (state.Root, []Effect) {
	if !root.CopyMode.Active || !root.CopyMode.SearchPending ||
		root.CopyMode.SearchRequestID != msg.RequestID ||
		root.CopyMode.Query != msg.Query || msg.Mode != "" && root.CopyMode.NormalizedSearchMode() != msg.Mode || root.CopyMode.BoundToken != root.History.Token ||
		(msg.EndpointID != "" && root.CopyMode.EndpointID != msg.EndpointID) ||
		(msg.TerminalID != "" && root.CopyMode.TerminalID != msg.TerminalID) ||
		(msg.Token != "" && root.CopyMode.BoundToken != msg.Token) {
		return root, nil
	}
	root.CopyMode.SearchPending = false
	if msg.Err != nil {
		root.CopyMode.SearchError = msg.Err.Error()
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Search failed", Body: msg.Err.Error(), DismissAfterTicks: 5})
		return root.Advance(), nil
	}
	if !msg.Result.Found {
		root.CopyMode.Matches = nil
		root.CopyMode.SearchMatchStart = state.CopyLogicalPosition{}
		root.CopyMode.SearchMatchEnd = state.CopyLogicalPosition{}
		root.CopyMode.SearchWrapped = false
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "No search match", DismissAfterTicks: 3})
		return root.Advance(), nil
	}
	nextHistory, err := root.History.ReplaceSearchWindow(msg.Result.Window)
	if err != nil {
		if errors.Is(err, state.ErrStaleHistoryResponse) {
			return root, nil
		}
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Search failed", Body: err.Error(), DismissAfterTicks: 5})
		return root.Advance(), nil
	}
	root.History = nextHistory
	start := state.CopyPositionForLogicalPosition(root.History, msg.Result.Start)
	end := state.CopyPositionForLogicalPosition(root.History, msg.Result.End)
	root.CopyMode.Cursor = start
	root.CopyMode.Mark = nil
	root.CopyMode.Selection = nil
	root.CopyMode.Matches = []state.CopyMatch{{StartRow: start.Row, StartCol: start.Col, EndRow: end.Row, EndCol: end.Col}}
	root.CopyMode.ActiveMatch = 0
	root.CopyMode.SearchMatchStart = msg.Result.Start
	root.CopyMode.SearchMatchEnd = msg.Result.End
	root.CopyMode.SearchWrapped = msg.Result.Wrapped
	root.CopyMode = root.CopyMode.RefreshSearchMatches(root.History)
	root.CopyMode = ensureCopyCursorVisible(root.CopyMode, len(root.History.Rows))
	return root.Advance(), nil
}

func beginCopyModeSearchScan(root state.Root, deps CopyModeDeps) (state.Root, []Effect) {
	if deps.Core == nil || root.CopyMode.Query == "" || root.CopyMode.SearchScanPending || root.CopyMode.SearchScanComplete {
		return root, nil
	}
	query := root.CopyMode.Query
	mode := root.CopyMode.NormalizedSearchMode()
	viewID := root.CopyMode.ViewID
	req := port.HistorySearchRequest{
		EndpointID: root.CopyMode.EndpointID, TerminalID: root.CopyMode.TerminalID,
		Cols: root.History.Cols, Rows: 1, Token: root.CopyMode.BoundToken, Generation: root.History.Generation,
		Query: query, Mode: mode, Direction: port.HistorySearchForward, Scan: true, MaxMatches: copyModeSearchScanRequestSize,
	}
	root.CopyMode.SearchScanPending = true
	root.CopyMode.SearchScanComplete = false
	return root, []Effect{StreamEffect{Token: copyModeSearchScanToken(viewID), Run: func(ctx context.Context, post func(Msg)) {
		pending := make([]port.HistorySearchMatch, 0, copyModeSearchScanPostSize)
		lastPost := time.Now()
		postPending := func(err error, done bool) {
			matches := pending
			pending = make([]port.HistorySearchMatch, 0, copyModeSearchScanPostSize)
			post(CopyModeSearchScanBatchMsg{
				Err: err, Query: query, Mode: mode, ViewID: viewID, EndpointID: req.EndpointID,
				TerminalID: req.TerminalID, Token: req.Token, Matches: matches, Done: done,
			})
			lastPost = time.Now()
		}
		for {
			result, err := deps.Core.HistorySearch(ctx, req)
			if ctx.Err() != nil {
				return
			}
			pending = append(pending, result.Matches...)
			if err != nil || result.Done || len(pending) >= copyModeSearchScanPostSize || time.Since(lastPost) >= copyModeSearchScanPostInterval {
				postPending(err, result.Done || err != nil)
			}
			if err != nil || result.Done {
				return
			}
			if !result.Next.Valid || result.Next == req.Start {
				postPending(nil, true)
				return
			}
			req.Start = result.Next
			runtime.Gosched()
		}
	}}}
}

func reduceCopyModeSearchScanBatch(root state.Root, msg CopyModeSearchScanBatchMsg) state.Root {
	if !root.CopyMode.Active || !root.CopyMode.SearchScanPending || root.CopyMode.Query != msg.Query ||
		root.CopyMode.NormalizedSearchMode() != msg.Mode ||
		(msg.EndpointID != "" && root.CopyMode.EndpointID != msg.EndpointID) ||
		(msg.TerminalID != "" && root.CopyMode.TerminalID != msg.TerminalID) ||
		(msg.Token != "" && root.CopyMode.BoundToken != msg.Token) {
		return root
	}
	if msg.Err != nil {
		root.CopyMode.SearchScanPending = false
		root.CopyMode.SearchError = msg.Err.Error()
		return root.Advance()
	}
	results := make([]state.CopyLogicalMatch, len(msg.Matches))
	for index, match := range msg.Matches {
		results[index] = state.CopyLogicalMatch{Start: match.Start, End: match.End}
	}
	var err error
	root.CopyMode, err = root.CopyMode.AppendSearchResults(root.History, results)
	if err != nil {
		root.CopyMode.SearchScanPending = false
		root.CopyMode.SearchError = err.Error()
		return root.Advance()
	}
	if msg.Done {
		root.CopyMode.SearchScanPending = false
		root.CopyMode.SearchScanComplete = true
	}
	return root.Advance()
}

func copyModeSearchCancelEffects(viewID string, pending bool, scanning bool) []Effect {
	var effects []Effect
	if pending {
		effects = append(effects, CancelEffect{Token: copyModeSearchRequestToken(viewID)})
	}
	if scanning {
		effects = append(effects, CancelEffect{Token: copyModeSearchScanToken(viewID)})
	}
	return effects
}

func copyModeLineEndPosition(history state.HistoryStore, row int) state.CopyPosition {
	if len(history.Rows) == 0 {
		return state.CopyPosition{}
	}
	row = clampColumn(row, 0, len(history.Rows)-1)
	return state.CopyPosition{Row: row, Col: state.HistoryRowDisplayWidth(history.Rows[row])}
}

func beginCopyModeLatest(root state.Root, deps CopyModeDeps) (state.Root, []Effect) {
	if deps.Core == nil {
		return setCopyModeError(root, "core client missing"), nil
	}
	binding, hasBinding := activeTerminalViewBinding(root)
	rect, ok := copyModeContentRect(root)
	if !hasBinding || binding.TerminalID == "" || !ok || rect.W <= 0 {
		return setCopyModeError(root, "copy mode requires attached terminal and cols"), nil
	}
	return beginCopyModeLatestForView(root, deps, binding, rect.W, rect.H)
}

func beginCopyModeLatestForView(root state.Root, deps CopyModeDeps, binding state.TerminalViewBinding, visibleCols int, rowsHint int) (state.Root, []Effect) {
	root = rootWithCopyHistorySessionForView(root, binding.ViewID)
	if deps.Core == nil {
		return setCopyModeError(root, "core client missing"), nil
	}
	if binding.TerminalID == "" || visibleCols <= 0 {
		return setCopyModeError(root, "copy mode requires attached terminal and cols"), nil
	}
	cols, terminalRows := copyModeTerminalViewportSize(root, binding, visibleCols, rowsHint)
	viewRows := copyModeVisibleRows(terminalRows, rowsHint)
	requestID := nextHistoryRequestID(root)
	nextHistory, err := root.History.BeginLatest(state.HistoryPendingRequest{
		ID:         requestID,
		PaneID:     binding.PaneID,
		ViewID:     binding.ViewID,
		EndpointID: binding.EndpointID,
		TerminalID: binding.TerminalID,
		Cols:       cols,
	})
	if err != nil {
		// 连续 latest/rebind 期间如果上一个请求还在飞，只保留 pending，不把内部背压抬成用户可见错误。
		if errors.Is(err, state.ErrHistoryRequestPending) {
			return root, nil
		}
		return setCopyModeError(root, err.Error()), nil
	}
	root.History = nextHistory
	root.CopyMode = root.CopyMode.BindLatestRef(binding.PaneID, binding.ViewID, binding.TerminalRef(), requestID, cols, viewRows)
	rows := requestRows(viewRows, deps.Rows)
	root = root.Advance()
	root = saveCopyHistorySessionForView(root, binding.ViewID)
	latestEffect := FuncEffect{
		Token: copyModeHistoryRequestToken(binding.ViewID),
		// history.window 真实走 protocol/client 时可能明显慢于一帧；
		// 这里必须异步请求，不能把 copy mode 入口卡在 runtime 主循环里。
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.copy.history_latest.effect")
			result, err := deps.Core.HistoryLatest(ctx, port.HistoryLatestRequest{
				EndpointID: binding.EndpointID,
				RequestID:  port.RequestID(requestID),
				PaneID:     binding.PaneID,
				ViewID:     binding.ViewID,
				TerminalID: binding.TerminalID,
				Cols:       cols,
				Rows:       rows,
				// 中文说明：history.window 的 frozen logical-line 边界由 core 请求时建立；
				// TUI copy/history 入口不能用 live surface revision 截断或校验。
			})
			finish(len(result.Window.Rows))
			perftrace.Count("tui.copy.history_latest.rows", len(result.Window.Rows))
			result.Window.PaneID = binding.PaneID
			result.Window.ViewID = binding.ViewID
			result.Window.EndpointID = binding.EndpointID
			return CopyModeHistoryResultMsg{Result: result, Err: err, RequestID: state.RequestID(requestID), PaneID: binding.PaneID, ViewID: binding.ViewID, EndpointID: binding.EndpointID, TerminalID: binding.TerminalID}
		},
	}
	return root, []Effect{latestEffect}
}

func copyModeTerminalViewportSize(root state.Root, binding state.TerminalViewBinding, fallbackCols int, fallbackRows int) (int, int) {
	cols, rows := binding.DesiredCols, binding.DesiredRows
	ref := binding.TerminalRef()
	if surface := root.Surface.SurfaceForTerminalRef(ref); surface.TerminalID != "" {
		if cols <= 0 {
			cols = surface.Cols
		}
		if rows <= 0 {
			rows = surface.Rows
		}
	}
	if root.Session.TerminalRef().Equal(ref) {
		if cols <= 0 {
			cols = root.Session.Cols
		}
		if rows <= 0 {
			rows = root.Session.Rows
		}
	}
	if cols <= 0 {
		cols = fallbackCols
	}
	if rows <= 0 {
		rows = fallbackRows
	}
	return cols, rows
}

func copyModeVisibleRows(terminalRows int, paneRows int) int {
	if terminalRows <= 0 {
		return paneRows
	}
	if paneRows <= 0 || terminalRows < paneRows {
		return terminalRows
	}
	return paneRows
}

func beginCopyModeOlder(root state.Root, deps CopyModeDeps, scrollDeltaAfterPrepend int) (state.Root, []Effect) {
	if deps.Core == nil {
		return setCopyModeError(root, "core client missing"), nil
	}
	switch root.History.OlderRequestState() {
	case state.OlderRequestPending, state.OlderRequestExhausted, state.OlderRequestMissing:
		return root, nil
	}
	requestID := nextHistoryRequestID(root)
	req := state.HistoryPendingRequest{
		ID:                 requestID,
		PaneID:             root.History.PaneID,
		ViewID:             root.History.ViewID,
		EndpointID:         root.History.EndpointID,
		TerminalID:         root.History.TerminalID,
		Cols:               root.History.Cols,
		Token:              root.History.Token,
		Generation:         root.History.Generation,
		Cursor:             root.History.Cursor,
		Boundary:           root.History.Boundary,
		DeferredScrollRows: scrollDeltaAfterPrepend,
	}
	nextHistory, err := root.History.BeginOlder(req)
	if err != nil {
		// older 重复触发时直接忽略，保持当前 pending/history 视图，不弹假错误。
		if errors.Is(err, state.ErrHistoryRequestPending) {
			return root, nil
		}
		return setCopyModeError(root, err.Error()), nil
	}
	root.History = nextHistory
	rows := copyModeHistoryRequestRows(root, deps)
	return root.Advance(), []Effect{FuncEffect{
		Token: copyModeHistoryRequestToken(req.ViewID),
		// older 分页也必须异步，否则连续 PageUp / wheel up 会把整个 UI 主循环卡住。
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.copy.history_older.effect")
			result, err := deps.Core.HistoryOlder(ctx, port.HistoryOlderRequest{
				EndpointID: req.EndpointID,
				RequestID:  port.RequestID(requestID),
				PaneID:     req.PaneID,
				ViewID:     req.ViewID,
				TerminalID: req.TerminalID,
				Cols:       req.Cols,
				Rows:       rows,
				Token:      req.Token,
				Generation: req.Generation,
				Cursor:     req.Cursor,
				Boundary:   req.Boundary,
			})
			finish(len(result.Window.Rows))
			perftrace.Count("tui.copy.history_older.rows", len(result.Window.Rows))
			result.Window.PaneID = req.PaneID
			result.Window.ViewID = req.ViewID
			result.Window.EndpointID = req.EndpointID
			return CopyModeHistoryResultMsg{Result: result, Err: err, RequestID: state.RequestID(requestID), PaneID: req.PaneID, ViewID: req.ViewID, EndpointID: req.EndpointID, TerminalID: req.TerminalID}
		},
	}}
}

func beginCopyModeNewer(root state.Root, deps CopyModeDeps, scrollDeltaAfterAppend int) (state.Root, []Effect) {
	if deps.Core == nil {
		return setCopyModeError(root, "core client missing"), nil
	}
	if root.History.NewerRequestState() != state.NewerRequestReady {
		return root, nil
	}
	tail := root.History.Rows[len(root.History.Rows)-1]
	requestID := nextHistoryRequestID(root)
	req := state.HistoryPendingRequest{
		ID:         requestID,
		PaneID:     root.History.PaneID,
		ViewID:     root.History.ViewID,
		EndpointID: root.History.EndpointID,
		TerminalID: root.History.TerminalID,
		Cols:       root.History.Cols,
		Token:      root.History.Token,
		Generation: root.History.Generation,
		Cursor: state.HistoryCursor{
			Valid:           tail.LineID != 0,
			BeforeLineID:    tail.LineID,
			BeforeRowInLine: tail.RowInLine,
			Segment:         tail.Segment,
		},
		Boundary:           root.History.Boundary,
		DeferredScrollRows: scrollDeltaAfterAppend,
	}
	nextHistory, err := root.History.BeginNewer(req)
	if err != nil {
		if errors.Is(err, state.ErrHistoryRequestPending) {
			return root, nil
		}
		return setCopyModeError(root, err.Error()), nil
	}
	root.History = nextHistory
	rows := copyModeHistoryRequestRows(root, deps)
	return root.Advance(), []Effect{FuncEffect{
		Token: copyModeHistoryRequestToken(req.ViewID),
		// newer 只在本地窗口已回收较新尾部时触发，仍然按事件循环异步拉取。
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.copy.history_newer.effect")
			result, err := deps.Core.HistoryNewer(ctx, port.HistoryNewerRequest{
				EndpointID: req.EndpointID,
				RequestID:  port.RequestID(requestID),
				PaneID:     req.PaneID,
				ViewID:     req.ViewID,
				TerminalID: req.TerminalID,
				Cols:       req.Cols,
				Rows:       rows,
				Token:      req.Token,
				Generation: req.Generation,
				Cursor:     req.Cursor,
				Boundary:   req.Boundary,
			})
			finish(len(result.Window.Rows))
			perftrace.Count("tui.copy.history_newer.rows", len(result.Window.Rows))
			result.Window.PaneID = req.PaneID
			result.Window.ViewID = req.ViewID
			result.Window.EndpointID = req.EndpointID
			return CopyModeHistoryResultMsg{Result: result, Err: err, RequestID: state.RequestID(requestID), PaneID: req.PaneID, ViewID: req.ViewID, EndpointID: req.EndpointID, TerminalID: req.TerminalID}
		},
	}}
}

func beginCopyModeOldest(root state.Root, deps CopyModeDeps) (state.Root, []Effect) {
	if deps.Core == nil {
		return setCopyModeError(root, "core client missing"), nil
	}
	switch root.History.OlderRequestState() {
	case state.OlderRequestPending, state.OlderRequestExhausted, state.OlderRequestMissing:
		return root, nil
	}
	requestID := nextHistoryRequestID(root)
	req := state.HistoryPendingRequest{
		ID:         requestID,
		Kind:       state.HistoryRequestOldest,
		PaneID:     root.History.PaneID,
		ViewID:     root.History.ViewID,
		EndpointID: root.History.EndpointID,
		TerminalID: root.History.TerminalID,
		Cols:       root.History.Cols,
		Token:      root.History.Token,
		Generation: root.History.Generation,
		Boundary:   root.History.Boundary,
	}
	nextHistory, err := root.History.BeginOldest(req)
	if err != nil {
		if errors.Is(err, state.ErrHistoryRequestPending) {
			return root, nil
		}
		return setCopyModeError(root, err.Error()), nil
	}
	root.History = nextHistory
	rows := copyModeHistoryRequestRows(root, deps)
	return root.Advance(), []Effect{FuncEffect{
		Token:            copyModeHistoryRequestToken(req.ViewID),
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			finish := perftrace.Measure("tui.copy.history_oldest.effect")
			result, err := deps.Core.HistoryOldest(ctx, port.HistoryOldestRequest{
				EndpointID: req.EndpointID,
				RequestID:  port.RequestID(requestID),
				PaneID:     req.PaneID,
				ViewID:     req.ViewID,
				TerminalID: req.TerminalID,
				Cols:       req.Cols,
				Rows:       rows,
				Token:      req.Token,
				Generation: req.Generation,
				Boundary:   req.Boundary,
			})
			finish(len(result.Window.Rows))
			perftrace.Count("tui.copy.history_oldest.rows", len(result.Window.Rows))
			result.Window.PaneID = req.PaneID
			result.Window.ViewID = req.ViewID
			result.Window.EndpointID = req.EndpointID
			return CopyModeHistoryResultMsg{Result: result, Err: err, RequestID: state.RequestID(requestID), PaneID: req.PaneID, ViewID: req.ViewID, EndpointID: req.EndpointID, TerminalID: req.TerminalID}
		},
	}}
}

func reduceCopyModeHistoryResult(root state.Root, msg CopyModeHistoryResultMsg, deps CopyModeDeps) (state.Root, []Effect) {
	if msg.Err != nil {
		if errors.Is(msg.Err, port.ErrStaleHistoryWindow) {
			return rejectStaleHistoryWindowError(root, msg, deps)
		}
		if errors.Is(msg.Err, port.ErrHistoryWindowTooLarge) {
			if staleCopyModeHistoryResult(root, msg) {
				return root, nil
			}
			return invalidateCopyModeHistory(root, deps, "History line is too large", "This line cannot be shown; loaded rows remain available")
		}
		if errors.Is(msg.Err, port.ErrHistoryResourceExhausted) {
			if staleCopyModeHistoryResult(root, msg) {
				return root, nil
			}
			return invalidateCopyModeHistory(root, deps, "History is temporarily unavailable", "Exit and reopen copy mode to retry")
		}
		if staleCopyModeHistoryResult(root, msg) {
			return root, nil
		}
		if root.CopyMode.Entering {
			return setCopyModeEnterError(root, msg.Err.Error()), nil
		}
		root.History.Pending = nil
		return setCopyModeError(root, msg.Err.Error()), nil
	}
	pending := root.History.Pending
	enteringScrollDelta := 0
	if pending != nil && pending.Kind == state.HistoryRequestLatest {
		enteringScrollDelta = root.CopyMode.EnteringScrollDelta
	}
	finishApply := perftrace.Measure("tui.copy.history_apply." + copyModePerfRequestKind(pending))
	beforeHistory := root.History
	window := msg.Result.Window
	if window.EndpointID == "" {
		window.EndpointID = msg.EndpointID
	}
	if window.EndpointID == "" && pending != nil {
		window.EndpointID = pending.EndpointID
	}
	nextHistory, inserted, err := root.History.ApplyWindow(copyModeHistoryResultRequestID(msg), window)
	finishApply(len(window.Rows))
	if err != nil {
		if errors.Is(err, state.ErrStaleHistoryResponse) {
			return rejectMatchingHistoryResponse(root, msg, err), nil
		}
		if root.CopyMode.Entering {
			return setCopyModeEnterError(root, err.Error()), nil
		}
		return setCopyModeError(root, err.Error()), nil
	}
	root.History = nextHistory
	remainingEnteringOlderRows := 0
	remainingNewerRows := 0
	clampViewport := true
	if pending != nil && pending.Kind == state.HistoryRequestLatest {
		root.CopyMode = root.CopyMode.AcceptLatest(historyWindowForCopyModeAnchor(window, nextHistory), nextHistory.Cols, len(nextHistory.Rows))
		remainingEnteringOlderRows = copyModeEnteringOlderRemainder(enteringScrollDelta, len(nextHistory.Rows))
		clampViewport = false
	} else if pending != nil && pending.Kind == state.HistoryRequestOldest {
		root.CopyMode = root.CopyMode.AcceptOldest(window, nextHistory.Cols, len(nextHistory.Rows))
	} else if pending != nil && pending.Kind == state.HistoryRequestNewer {
		root.CopyMode.BoundToken = window.Token
		root.CopyMode.BoundCols = nextHistory.Cols
		root.CopyMode = root.CopyMode.RestoreViewportTail(nextHistory)
		root.CopyMode = root.CopyMode.FollowCursor(len(nextHistory.Rows))
		var consumedRows int
		root.CopyMode, consumedRows = root.CopyMode.ScrollNewer(pending.DeferredScrollRows, len(nextHistory.Rows))
		remainingNewerRows = max(0, pending.DeferredScrollRows-consumedRows)
	} else {
		root.CopyMode = root.CopyMode.AcceptOlder(inserted, beforeHistory, nextHistory, window, nextHistory.Cols)
		deferredRows := 0
		if pending != nil {
			deferredRows = pending.DeferredScrollRows
		}
		root.CopyMode = root.CopyMode.ApplyDeferredOlderScroll(deferredRows, len(nextHistory.Rows))
	}
	if root.CopyMode.Query != "" {
		root.CopyMode = root.CopyMode.RefreshSearchMatch(root.History)
	}
	if clampViewport {
		root.CopyMode = root.CopyMode.Scroll(0, len(root.History.Rows))
	}
	root = refreshCopyModeLogicalSelectionFocus(root)
	if pending != nil && (pending.Kind == state.HistoryRequestOlder || pending.Kind == state.HistoryRequestNewer) {
		root = trimCopyModeHistoryWindow(root, deps)
	}
	if pending != nil && pending.Kind == state.HistoryRequestNewer {
		root.CopyMode = root.CopyMode.RestoreViewportTail(root.History)
		if root.CopyMode.AtFrozenBottom(root.History) && root.CopyMode.Mark == nil && root.CopyMode.Selection == nil {
			next, effects := exitCopyModeWithRelease(root, deps)
			return next.Advance(), effects
		}
		if remainingNewerRows > 0 && root.History.NewerRequestState() == state.NewerRequestReady {
			return beginCopyModeNewer(root, deps, remainingNewerRows)
		}
	}
	if remainingEnteringOlderRows > 0 && root.History.OlderRequestState() == state.OlderRequestReady {
		return beginCopyModeOlder(root, deps, remainingEnteringOlderRows)
	}
	return root.Advance(), nil
}

func copyModePerfRequestKind(pending *state.HistoryPendingRequest) string {
	if pending == nil || pending.Kind == "" {
		return "unknown"
	}
	return string(pending.Kind)
}

func trimCopyModeHistoryWindow(root state.Root, deps CopyModeDeps) state.Root {
	if !root.CopyMode.Active || len(root.History.Rows) == 0 {
		return root
	}
	visible := copyModePanelRows(root, deps)
	if visible <= 0 {
		visible = copyModePageRows(root.CopyMode)
	}
	keepRows := clampColumn(visible*copyModeHistoryWindowScreens, copyModeHistoryMinRequestRows, copyModeHistoryMaxRequestRows)
	if len(root.History.Rows) <= keepRows {
		return root
	}
	root.CopyMode = root.CopyMode.EnsureLogicalSelection(root.History)
	start, end := copyModeHistoryTrimRange(root.CopyMode, len(root.History.Rows), keepRows)
	nextHistory, trim := root.History.TrimRows(start, end)
	if trim.DroppedRowsBefore == 0 && trim.DroppedRowsAfter == 0 {
		return root
	}
	root.History = nextHistory
	root.CopyMode = root.CopyMode.ApplyHistoryTrim(trim, len(root.History.Rows))
	if root.CopyMode.Query != "" {
		root.CopyMode = root.CopyMode.RefreshSearchMatch(root.History)
	}
	return root
}

func copyModeHistoryTrimRange(copyMode state.CopyModeStore, totalRows int, keepRows int) (int, int) {
	if totalRows <= 0 {
		return 0, -1
	}
	if keepRows <= 0 || keepRows >= totalRows {
		return 0, totalRows - 1
	}
	visible := copyMode.ViewRows
	if visible <= 0 {
		visible = 8
	}
	// 中文说明：history.window 已有 older/newer cursor，TUI 只保留围绕当前
	// viewport 的本地投影；窗口外内容后续按 frozen token 从 core/backend 拉回。
	center := copyMode.ViewportTop + visible/2
	start := center - keepRows/2
	if start < 0 {
		start = 0
	}
	end := start + keepRows - 1
	if end >= totalRows {
		end = totalRows - 1
		start = end - keepRows + 1
		if start < 0 {
			start = 0
		}
	}
	viewportEnd := copyMode.ViewportTop + visible - 1
	marginStart := copyMode.ViewportTop - visible*copyModeHistoryPrefetchScreens
	marginEnd := viewportEnd + visible*copyModeHistoryPrefetchScreens
	if marginStart < 0 {
		marginStart = 0
	}
	if marginEnd >= totalRows {
		marginEnd = totalRows - 1
	}
	if marginStart < start {
		start = marginStart
	}
	if marginEnd > end {
		end = marginEnd
	}
	start, end = expandTrimRangeForSelection(copyMode, start, end, totalRows)
	start = clampColumn(start, 0, totalRows-1)
	end = clampColumn(end, start, totalRows-1)
	return start, end
}

func expandTrimRangeForSelection(copyMode state.CopyModeStore, start int, end int, totalRows int) (int, int) {
	positions := []state.CopyPosition{copyMode.Cursor}
	selectionHasLogicalRange := copyMode.Selection != nil &&
		copyMode.Selection.LogicalAnchor.Valid &&
		copyMode.Selection.LogicalFocus.Valid
	if !selectionHasLogicalRange {
		if copyMode.Mark != nil {
			positions = append(positions, *copyMode.Mark)
		}
		if copyMode.Selection != nil {
			positions = append(positions, copyMode.Selection.Anchor, copyMode.Selection.Focus)
		}
	}
	for _, pos := range positions {
		if pos.Row < start {
			start = pos.Row
		}
		if pos.Row > end {
			end = pos.Row
		}
	}
	start = clampColumn(start, 0, totalRows-1)
	end = clampColumn(end, start, totalRows-1)
	return start, end
}

func copyModeEnteringOlderRemainder(scrollDelta int, loadedRows int) int {
	if scrollDelta >= 0 {
		return 0
	}
	localRowsAboveTail := loadedRows - 1
	if localRowsAboveTail < 0 {
		localRowsAboveTail = 0
	}
	olderRows := -scrollDelta - localRowsAboveTail
	if olderRows < 0 {
		return 0
	}
	return olderRows
}

func rejectMatchingHistoryResponse(root state.Root, msg CopyModeHistoryResultMsg, err error) state.Root {
	if root.History.Pending == nil || copyModeHistoryResultRequestID(msg) != root.History.Pending.ID || !copyModeOwnsPendingHistory(root) {
		return root
	}
	// 中文说明：同一个 request 已经返回但被 stale guard 拒绝时，不能继续保留 pending；
	// 否则 copy history footer 会永久显示 `↑ loading`，并阻止下一次 older 请求重试。
	root.History.Pending = nil
	if root.CopyMode.Entering {
		return setCopyModeEnterError(root, err.Error())
	}
	return setCopyModeError(root, err.Error())
}

func rejectStaleHistoryWindowError(root state.Root, msg CopyModeHistoryResultMsg, deps CopyModeDeps) (state.Root, []Effect) {
	if root.History.Pending == nil || copyModeHistoryResultRequestID(msg) != root.History.Pending.ID || !copyModeOwnsPendingHistory(root) {
		return root, nil
	}
	pending := *root.History.Pending
	// 中文说明：protocol stale history window 表达 frozen token/cursor 已过期；
	// 这是请求生命周期控制信号，不能把 protocol 400 暴露成用户可见错误。
	root.History.Pending = nil
	if pending.Kind == state.HistoryRequestLatest && root.CopyMode.Entering {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "History window expired", Body: "Reopen copy mode to reload history", DismissAfterTicks: 5})
		return exitCopyMode(root).Advance(), nil
	}
	return invalidateCopyModeHistory(root, deps, "History window expired", "Exit and reopen copy mode to reload history")
}

func copyModeOwnsPendingHistory(root state.Root) bool {
	pending := root.History.Pending
	if pending == nil || !copyModeInputContext(root.CopyMode) {
		return false
	}
	if pending.TerminalID != "" && root.CopyMode.TerminalID != pending.TerminalID {
		return false
	}
	if pending.PaneID != "" && root.CopyMode.PaneID != pending.PaneID {
		return false
	}
	if pending.ViewID != "" && root.CopyMode.ViewID != pending.ViewID {
		return false
	}
	if (pending.Kind == state.HistoryRequestOlder || pending.Kind == state.HistoryRequestOldest) && pending.Token != "" && root.CopyMode.BoundToken != pending.Token {
		return false
	}
	return true
}

func reduceCopyModeCopySelection(root state.Root, deps CopyModeDeps) (state.Root, []Effect) {
	if !root.CopyMode.CanCopy() {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Copy unavailable", Body: "History selection is not ready", DismissAfterTicks: 3})
		return root.Advance(), nil
	}
	if root.CopyMode.CopyPending {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "Copy in progress", DismissAfterTicks: 2})
		return root.Advance(), nil
	}
	req := port.HistoryCopyRangeRequest{
		EndpointID: root.History.EndpointID,
		TerminalID: root.History.TerminalID,
		Cols:       root.History.Cols,
		Token:      root.History.Token,
		Generation: root.History.Generation,
		Boundary:   root.History.Boundary,
	}
	if !root.CopyMode.SelectionNeedsBackend(root.History) {
		copied := SelectedText(root.History, root.CopyMode)
		if copied == "" {
			root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Nothing to copy", Body: "The selected range is empty", DismissAfterTicks: 3})
			return root.Advance(), nil
		}
		return beginCopyModeLocalSelection(root, deps, req, copied)
	}
	start, end, hasRange := root.CopyMode.SelectionLogicalRange(root.History)
	if !hasRange {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Copy unavailable", Body: "The selected history range is no longer available", DismissAfterTicks: 4})
		return root.Advance(), nil
	}
	req.Start = start
	req.End = end
	if deps.Core == nil || req.TerminalID == "" || req.Token == "" {
		root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "Copy failed", Body: "history copy backend missing", DismissAfterTicks: 5})
		return root.Advance(), nil
	}
	requestID := state.RequestID(nextHistoryRequestID(root))
	viewID := root.CopyMode.ViewID
	root.CopyMode.CopyPending = true
	root.CopyMode.CopyRequestID = requestID
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "Loading selected history", DismissAfterTicks: 3})
	root = root.Advance()
	return root, []Effect{
		FuncEffect{
			Token:            copyModeCopyRequestToken(viewID),
			Async:            true,
			ForceSyncInTests: true,
			Run: func(ctx context.Context) Msg {
				result, err := deps.Core.HistoryCopyRange(ctx, req)
				if err != nil {
					return copyModeCopyResultMessage(requestID, viewID, req, "", false, err, nil)
				}
				copied := result.Text
				if copied == "" {
					return copyModeCopyResultMessage(requestID, viewID, req, "", false, nil, nil)
				}
				var clipboardErr error
				if deps.Clipboard == nil {
					clipboardErr = errors.New("clipboard service missing")
				} else {
					clipboardErr = deps.Clipboard.Write(ctx, port.ClipboardWriteRequest{Text: copied})
				}
				if len(copied) > state.MaxClipboardHistoryEntryBytes {
					if clipboardErr != nil {
						return copyModeCopyResultMessage(requestID, viewID, req, "", false, clipboardErr, nil)
					}
					return copyModeCopyResultMessage(requestID, viewID, req, "", true, nil, nil)
				}
				return copyModeCopyResultMessage(requestID, viewID, req, copied, true, nil, clipboardErr)
			},
		},
	}
}

func beginCopyModeLocalSelection(root state.Root, deps CopyModeDeps, req port.HistoryCopyRangeRequest, copied string) (state.Root, []Effect) {
	requestID := state.RequestID(nextHistoryRequestID(root))
	viewID := root.CopyMode.ViewID
	root.CopyMode.CopyPending = true
	root.CopyMode.CopyRequestID = requestID

	var stored bool
	root.Clipboard, stored = root.Clipboard.WithCopiedTextLimit(copied, clipboardHistoryMaxItems(root))
	body := "Syncing the system clipboard"
	if stored {
		body = "Saved to the local buffer; syncing the system clipboard"
	}
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastInfo, Title: "Copying selection", Body: body, DismissAfterTicks: 3})
	root = root.Advance()

	effects := []Effect{FuncEffect{
		Token:            copyModeCopyRequestToken(viewID),
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			var clipboardErr error
			if deps.Clipboard == nil {
				clipboardErr = errors.New("clipboard service missing")
			} else {
				clipboardErr = deps.Clipboard.Write(ctx, port.ClipboardWriteRequest{Text: copied})
			}
			if clipboardErr != nil && !stored {
				return copyModeCopyResultMessage(requestID, viewID, req, "", false, clipboardErr, nil)
			}
			return copyModeCopyResultMessage(requestID, viewID, req, "", true, nil, clipboardErr)
		},
	}}
	if stored {
		effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
			return ClipboardStoragePersistRequestMsg{Reason: "copy"}
		}})
	}
	return root, effects
}

func copyModeCopyResultMessage(requestID state.RequestID, viewID string, req port.HistoryCopyRangeRequest, text string, commit bool, err error, clipboardErr error) CopyModeCopyResultMsg {
	return CopyModeCopyResultMsg{
		Text: text, Err: err, ClipboardErr: clipboardErr, Commit: commit, RequestID: requestID, ViewID: viewID,
		EndpointID: req.EndpointID, TerminalID: req.TerminalID, Token: req.Token,
	}
}

func historyWindowForCopyModeAnchor(window state.HistoryWindow, history state.HistoryStore) state.HistoryWindow {
	window.Rows = history.Rows
	window.Cols = history.Cols
	return window
}

func SelectedText(history state.HistoryStore, copyMode state.CopyModeStore) string {
	if copyMode.Selection == nil || !copyMode.Active {
		return ""
	}
	start := copyMode.Selection.Anchor
	end := copyMode.Selection.Focus
	if positionAfter(start, end) {
		start, end = end, start
	}
	if start.Row < 0 {
		start.Row = 0
	}
	if end.Row >= len(history.Rows) {
		end.Row = len(history.Rows) - 1
	}
	if start.Row > end.Row || start.Row >= len(history.Rows) || end.Row < 0 {
		return ""
	}
	lines := make([]string, 0, end.Row-start.Row+1)
	var currentLineID uint64
	var currentLine strings.Builder
	flushCurrentLine := func() {
		if currentLine.Len() == 0 && currentLineID == 0 {
			return
		}
		lines = append(lines, currentLine.String())
		currentLine.Reset()
		currentLineID = 0
	}
	for row := start.Row; row <= end.Row; row++ {
		from := 0
		to := state.HistoryRowDisplayWidth(history.Rows[row])
		if row == start.Row {
			from = clampColumn(start.Col, 0, to)
		}
		if row == end.Row {
			to = clampColumn(end.Col, 0, to)
		}
		if from > to {
			from, to = to, from
		}
		segment := state.HistoryRowSliceDisplay(history.Rows[row], from, to)
		lineID := history.Rows[row].LineID
		if currentLineID != 0 && lineID != 0 && currentLineID != lineID {
			flushCurrentLine()
		}
		if currentLineID == 0 {
			currentLineID = lineID
		}
		currentLine.WriteString(segment)
	}
	flushCurrentLine()
	return strings.Join(lines, "\n")
}

func positionAfter(left state.CopyPosition, right state.CopyPosition) bool {
	if left.Row != right.Row {
		return left.Row > right.Row
	}
	return left.Col > right.Col
}

func clampColumn(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func nextHistoryRequestID(root state.Root) state.RequestID {
	next := state.RequestID(root.Generation + 1)
	if root.History.Pending != nil && root.History.Pending.ID >= next {
		next = root.History.Pending.ID + 1
	}
	return next
}

func clampCopyCursor(copyMode state.CopyModeStore, history state.HistoryStore) state.CopyModeStore {
	if len(history.Rows) == 0 {
		copyMode.Cursor = state.CopyPosition{}
		return copyMode
	}
	row := clampColumn(copyMode.Cursor.Row, 0, len(history.Rows)-1)
	col := clampColumn(copyMode.Cursor.Col, 0, state.HistoryRowDisplayWidth(history.Rows[row]))
	copyMode.Cursor = state.CopyPosition{Row: row, Col: col}
	return copyMode
}

func ensureCopyCursorVisible(copyMode state.CopyModeStore, totalRows int) state.CopyModeStore {
	return copyMode.FollowCursor(totalRows)
}

func copyModePageRows(copyMode state.CopyModeStore) int {
	if copyMode.ViewRows > 2 {
		return copyMode.ViewRows - 2
	}
	return 8
}

func copyModeLineScrollRows() int {
	return 1
}

func copyModeOlderScrollRows(copyMode state.CopyModeStore, event input.InputEvent) int {
	if event.Kind == input.EventKindKey && event.Key == input.KeyPageUp {
		return copyModePageRows(copyMode)
	}
	return copyModeLineScrollRows()
}

func copyModeNewerScrollRows(copyMode state.CopyModeStore, event input.InputEvent) int {
	if event.Kind == input.EventKindKey && event.Key == input.KeyPageDn {
		return copyModePageRows(copyMode)
	}
	return copyModeLineScrollRows()
}

func maybePrefetchCopyModeOlder(root state.Root, deps CopyModeDeps, scrollDeltaAfterPrepend int) (state.Root, []Effect) {
	if root.CopyMode.ViewportTop > copyModeOlderPrefetchRows(root, deps) {
		return root, nil
	}
	if !root.CopyMode.CanPageHistory() {
		return root, nil
	}
	if root.History.OlderRequestState() != state.OlderRequestReady {
		return root, nil
	}
	return beginCopyModeOlder(root, deps, scrollDeltaAfterPrepend)
}

func copyModeHistoryRequestRows(root state.Root, deps CopyModeDeps) int {
	panelRows := copyModePanelRows(root, deps)
	return clampColumn(panelRows*copyModeHistoryRequestScreens, copyModeHistoryMinRequestRows, copyModeHistoryMaxRequestRows)
}

func copyModeOlderPrefetchRows(root state.Root, deps CopyModeDeps) int {
	panelRows := copyModePanelRows(root, deps)
	return clampColumn(panelRows*copyModeHistoryPrefetchScreens, panelRows, copyModeHistoryMaxRequestRows/2)
}

func copyModePanelRows(root state.Root, deps CopyModeDeps) int {
	if root.CopyMode.ViewRows > 0 {
		return root.CopyMode.ViewRows
	}
	if rect, ok := copyModeContentRect(root); ok && rect.H > 0 {
		return rect.H
	}
	if deps.Rows > 0 {
		return deps.Rows
	}
	return 24
}

func requestRows(panelRows int, fallbackRows int) int {
	if panelRows <= 0 {
		panelRows = fallbackRows
	}
	if panelRows <= 0 {
		panelRows = 24
	}
	return clampColumn(panelRows*copyModeHistoryRequestScreens, copyModeHistoryMinRequestRows, copyModeHistoryMaxRequestRows)
}

func setCopyModeError(root state.Root, message string) state.Root {
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: "History unavailable", Body: message, DismissAfterTicks: 5})
	return root.Advance()
}

func setCopyModeEnterError(root state.Root, message string) state.Root {
	root = exitCopyMode(root)
	return setCopyModeError(root, message)
}

func exitCopyModeWithRelease(root state.Root, deps CopyModeDeps) (state.Root, []Effect) {
	cancelEffects := cancelPendingCopyModeHistoryEffects(root)
	token := root.CopyMode.BoundToken
	endpointID := root.CopyMode.EndpointID
	terminalID := root.CopyMode.TerminalID
	viewID := copyHistoryWorkingViewID(root)
	root = exitCopyMode(root)
	root = root.WithoutCopyHistorySession(viewID)
	return root, append(cancelEffects, releaseHistoryTokenEffects(deps, endpointID, terminalID, token)...)
}

func copyModeHistoryRequestToken(viewID string) CancelToken {
	return CancelToken("copy.history.request:" + viewID)
}

func copyModeSearchRequestToken(viewID string) CancelToken {
	return CancelToken("copy.history.search:" + viewID)
}

func copyModeSearchScanToken(viewID string) CancelToken {
	return CancelToken("copy.history.search-scan:" + viewID)
}

func copyModeCopyRequestToken(viewID string) CancelToken {
	return CancelToken("copy.history.copy:" + viewID)
}

func cancelPendingCopyModeHistoryEffects(root state.Root) []Effect {
	viewID := copyHistoryWorkingViewID(root)
	var effects []Effect
	if root.History.Pending != nil {
		if root.History.Pending.ViewID != "" {
			viewID = root.History.Pending.ViewID
		}
		if viewID != "" {
			effects = append(effects, CancelEffect{Token: copyModeHistoryRequestToken(viewID)})
		}
	}
	if root.CopyMode.SearchPending && viewID != "" {
		effects = append(effects, CancelEffect{Token: copyModeSearchRequestToken(viewID)})
	}
	if root.CopyMode.SearchScanPending && viewID != "" {
		effects = append(effects, CancelEffect{Token: copyModeSearchScanToken(viewID)})
	}
	if root.CopyMode.CopyPending && viewID != "" {
		effects = append(effects, CancelEffect{Token: copyModeCopyRequestToken(viewID)})
	}
	return effects
}

func copyHistoryCleanupEffectsForView(root state.Root, viewID string) []Effect {
	if viewID == "" {
		return nil
	}
	history, copyMode := root.CopyHistorySessionForView(viewID)
	var effects []Effect
	if history.Pending != nil {
		effects = append(effects, CancelEffect{Token: copyModeHistoryRequestToken(viewID)})
	}
	if copyMode.SearchPending {
		effects = append(effects, CancelEffect{Token: copyModeSearchRequestToken(viewID)})
	}
	if copyMode.SearchScanPending {
		effects = append(effects, CancelEffect{Token: copyModeSearchScanToken(viewID)})
	}
	if copyMode.CopyPending {
		effects = append(effects, CancelEffect{Token: copyModeCopyRequestToken(viewID)})
	}
	token := copyMode.BoundToken
	if token == "" {
		token = history.Token
	}
	if token == "" {
		return effects
	}
	endpointID := copyMode.EndpointID
	if endpointID == "" {
		endpointID = history.EndpointID
	}
	terminalID := copyMode.TerminalID
	if terminalID == "" {
		terminalID = history.TerminalID
	}
	effects = append(effects, FuncEffect{Run: func(context.Context) Msg {
		return CopyModeReleaseHistoryMsg{EndpointID: endpointID, TerminalID: terminalID, Token: token}
	}})
	return effects
}

func releaseHistoryTokenEffects(deps CopyModeDeps, endpointID state.EndpointID, terminalID string, token string) []Effect {
	if deps.Core == nil || token == "" {
		return nil
	}
	return []Effect{FuncEffect{
		Async:            true,
		ForceSyncInTests: true,
		Run: func(ctx context.Context) Msg {
			_ = deps.Core.ReleaseHistory(ctx, port.HistoryReleaseRequest{
				EndpointID: endpointID,
				TerminalID: terminalID,
				Token:      token,
			})
			return NoopMsg{}
		},
	}}
}

func invalidateCopyModeHistory(root state.Root, deps CopyModeDeps, title string, body string) (state.Root, []Effect) {
	token := root.CopyMode.BoundToken
	if token == "" {
		token = root.History.Token
	}
	endpointID := root.CopyMode.EndpointID
	terminalID := root.CopyMode.TerminalID
	root.History.Pending = nil
	root.History.Token = ""
	root.History.HasMore = false
	root.History.Cursor = state.HistoryCursor{}
	root.History.Exhausted = state.ExhaustedMarker{}
	root.CopyMode.BoundToken = ""
	root.CopyMode.Selection = nil
	root.CopyMode.Mark = nil
	root.Shell = root.Shell.AddToast(state.ToastSpec{Severity: state.ToastWarning, Title: title, Body: body, DismissAfterTicks: 5})
	return root.Advance(), releaseHistoryTokenEffects(deps, endpointID, terminalID, token)
}

func exitCopyMode(root state.Root) state.Root {
	// 中文说明：copy/history window 只是当前交互投影；退出或取消后必须释放，
	// 否则 TUI 会继续持有 rows/source lines 的 backing array。
	viewID := copyHistoryWorkingViewID(root)
	root.History = state.HistoryStore{}
	root.CopyMode = state.CopyModeStore{}
	if viewID != "" {
		root = root.WithoutCopyHistorySession(viewID)
	}
	return root
}
