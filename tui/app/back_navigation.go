package app

import (
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/state"
)

// NewBackNavigationReducer 是 TUI 全局返回语义的唯一 owner。
//
// Esc 不属于可配置 shortcut catalog。它始终按当前交互层级返回一层：先退出 prompt
// suggestion，再关闭顶层 overlay，然后退出 copy/history，最后退出 sticky interaction mode。
// 当没有任何 TUI 层可返回时不消费事件，由后续 terminal input router 透传给前台 PTY。
func NewBackNavigationReducer(copyMode CopyModeDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		inputMsg, ok := msg.(InputMsg)
		if !ok || !isGlobalBackEvent(inputMsg.Event) || inputMsg.TerminalMousePassthrough {
			return root, nil
		}

		switch root.CurrentBackNavigationLayer() {
		case state.BackNavigationPromptSuggestion:
			root.Shell = root.Shell.SetPromptSuggestionFocused(false)
			return root.Advance(), []Effect{handledEffect{}}
		case state.BackNavigationTerminalTags:
			root.Shell = root.Shell.CloseTerminalPickerTags()
			return root.Advance(), []Effect{handledEffect{}}
		case state.BackNavigationOverlay:
			if root.Shell.EnsureDefaults().Overlay.Kind == state.OverlayPrompt {
				root.Shell = root.Shell.CancelPrompt()
			}
			root.Shell = root.Shell.CloseOverlay()
			return root.Advance(), []Effect{handledEffect{}}
		case state.BackNavigationCopy:
			if root.CopyMode.SearchBarVisible() {
				pending := root.CopyMode.SearchPending
				scanning := root.CopyMode.SearchScanPending
				viewID := root.CopyMode.ViewID
				root.CopyMode = root.CopyMode.CloseSearch()
				effects := []Effect{handledEffect{}}
				if pending {
					effects = append(effects, CancelEffect{Token: copyModeSearchRequestToken(viewID)})
				}
				if scanning {
					effects = append(effects, CancelEffect{Token: copyModeSearchScanToken(viewID)})
				}
				return root.Advance(), effects
			}
			next, effects := exitCopyModeWithRelease(root, copyMode)
			return next.Advance(), append([]Effect{handledEffect{}}, effects...)
		case state.BackNavigationInteraction:
			root.Shell = root.Shell.ExitInteractionMode()
			return root.Advance(), []Effect{handledEffect{}}
		}
		return root, nil
	}
}

func isGlobalBackEvent(event input.InputEvent) bool {
	return event.Kind == input.EventKindKey && event.Key == input.KeyEsc && !event.Ctrl && !event.Alt && !event.Shift
}
