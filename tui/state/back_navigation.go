package state

// BackNavigationLayer 描述 Esc 当前应退出的最上层 TUI 交互。
// 层级只来自 reducer-owned Root，不读取 shortcut 配置，也不执行副作用。
type BackNavigationLayer string

const (
	BackNavigationNone              BackNavigationLayer = ""
	BackNavigationPromptSuggestion  BackNavigationLayer = "prompt-suggestion"
	BackNavigationTerminalEndpoints BackNavigationLayer = "terminal-picker-endpoints"
	BackNavigationTerminalTags      BackNavigationLayer = "terminal-picker-tags"
	BackNavigationOverlay           BackNavigationLayer = "overlay"
	BackNavigationCopy              BackNavigationLayer = "copy"
	BackNavigationInteraction       BackNavigationLayer = "interaction"
)

// CurrentBackNavigationLayer 返回全局 Esc 当前应退出的唯一层级。
// 顺序固定为 prompt suggestion、overlay、当前 view 的 copy/history、sticky interaction；
// 返回 None 表示 Esc 应继续交给前台 terminal，而不是被 TUI 吞掉。
func (r Root) CurrentBackNavigationLayer() BackNavigationLayer {
	shell := r.Shell.EnsureDefaults()
	if shell.Overlay.Open {
		if shell.Overlay.Kind == OverlayPrompt && shell.Overlay.Prompt.SuggestionFocused {
			return BackNavigationPromptSuggestion
		}
		if shell.Overlay.Kind == OverlayTerminalPicker && shell.Overlay.TerminalPickerView == TerminalPickerViewTags {
			return BackNavigationTerminalTags
		}
		if shell.Overlay.Kind == OverlayTerminalPicker && shell.Overlay.TerminalPickerView == TerminalPickerViewEndpoints {
			return BackNavigationTerminalEndpoints
		}
		return BackNavigationOverlay
	}
	if r.ActiveViewOwnsCopyInput() {
		return BackNavigationCopy
	}
	if shell.InteractionMode != InteractionModeNormal {
		return BackNavigationInteraction
	}
	return BackNavigationNone
}

// ActiveViewOwnsCopyInput 判断当前 active terminal view 是否拥有正在进行的 copy/history 输入态。
// 该查询独立于 overlay 和 sticky interaction 层级；上层 overlay 可以暂时抢占 Esc，但不能改变
// 底层 copy session 的 view ownership 或生命周期。
func (r Root) ActiveViewOwnsCopyInput() bool {
	if !r.CopyMode.InputActive() {
		return false
	}
	if r.CopyMode.ViewID == "" && r.CopyMode.PaneID == "" {
		return true
	}
	shell := r.Shell.ReadonlyDefaults()
	var binding TerminalViewBinding
	var ok bool
	if floatingID := shell.ActiveFloatingID(); floatingID != "" {
		binding, ok = r.TerminalViews.FloatingBinding(floatingID)
	} else {
		binding, ok = r.TerminalViews.PaneBinding(shell.ActivePaneID)
	}
	if !ok {
		return false
	}
	if r.CopyMode.ViewID != "" {
		return binding.ViewID == r.CopyMode.ViewID
	}
	return r.CopyMode.PaneID == binding.PaneID
}
