package app

import (
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/state"
)

// NewCopyModeResizeRebindReducer 在 copy content rect 宽度变化后失效旧窗口并重新请求 latest。
// 进入 frozen copy mode 后，宽度变化只做本地重排；不再回 core 请求新投影。
func NewCopyModeResizeRebindReducer(deps CopyModeDeps) Reducer {
	return func(root state.Root, msg Msg) (state.Root, []Effect) {
		if !copyModeLayoutMayNeedRebind(msg) || deps.Core == nil {
			return root, nil
		}
		root, activeViewID := rootWithActiveCopyHistorySession(root)
		binding, hasBinding := copyModeTerminalBinding(root)
		if root.CopyMode.Entering {
			rect, ok := copyModeContentRect(root)
			if !ok || rect.W <= 0 {
				return root, nil
			}
			cols, rows := copyModeRebindViewportSize(root, binding, hasBinding, rect)
			if cols == root.CopyMode.BoundCols && rows == root.CopyMode.ViewRows {
				return root, nil
			}
			root.History = root.History.RebindPendingLatest(cols)
			root.CopyMode = root.CopyMode.Resize(cols, rows)
			root = root.Advance()
			return saveCopyHistorySessionForView(root, activeViewID), nil
		}
		if !root.CopyMode.Active {
			return root, nil
		}
		rect, ok := copyModeContentRect(root)
		if !ok {
			return root, nil
		}
		cols, rows := copyModeRebindViewportSize(root, binding, hasBinding, rect)
		if cols == root.CopyMode.BoundCols && root.History.Cols == cols {
			if rows == root.CopyMode.ViewRows {
				return root, nil
			}
			root.CopyMode = root.CopyMode.SetViewRows(rows)
			root.CopyMode = root.CopyMode.Scroll(0, len(root.History.Rows))
			root = root.Advance()
			return saveCopyHistorySessionForView(root, activeViewID), nil
		}
		if len(root.History.SourceLines) == 0 {
			root.History = root.History.EnsureSourceLines()
		}
		if len(root.History.SourceLines) == 0 {
			root.History = root.History.InvalidateWindow()
			root.CopyMode = root.CopyMode.Resize(cols, rows)
			root.CopyMode.BoundToken = ""
			root.CopyMode.ViewportTop = 0
			root.CopyMode.Cursor = state.CopyPosition{}
			root.CopyMode.Mark = nil
			root.CopyMode.Selection = nil
			root.CopyMode.Query = ""
			root.CopyMode.Matches = nil
			root.CopyMode.ActiveMatch = 0
			root.CopyMode.Empty = true
			root = root.Advance()
			return saveCopyHistorySessionForView(root, activeViewID), nil
		}
		root.CopyMode = root.CopyMode.Resize(cols, rows)
		beforeHistory := root.History
		root.History.Cols = cols
		root.History.Rows, root.History.Lines = state.ReflowHistoryLogicalLines(root.History.SourceLines, cols)
		root.CopyMode = root.CopyMode.RebindToReflowedHistory(beforeHistory, root.History)
		root.CopyMode = root.CopyMode.Scroll(0, len(root.History.Rows))
		root = root.Advance()
		return saveCopyHistorySessionForView(root, activeViewID), nil
	}
}

func copyModeTerminalBinding(root state.Root) (state.TerminalViewBinding, bool) {
	if root.CopyMode.ViewID != "" {
		binding, ok := root.TerminalViews.Views[root.CopyMode.ViewID]
		return binding, ok && binding.TerminalID == root.CopyMode.TerminalID
	}
	return activeTerminalViewBinding(root)
}

func copyModeRebindViewportSize(root state.Root, binding state.TerminalViewBinding, hasBinding bool, rect render.Rect) (int, int) {
	fallbackCols, fallbackRows := root.CopyMode.BoundCols, root.CopyMode.ViewRows
	if fallbackCols <= 0 {
		fallbackCols = rect.W
	}
	if fallbackRows <= 0 {
		fallbackRows = rect.H
	}
	if !hasBinding {
		return fallbackCols, fallbackRows
	}
	cols, terminalRows := copyModeTerminalViewportSize(root, binding, fallbackCols, fallbackRows)
	return cols, copyModeVisibleRows(terminalRows, rect.H)
}

func copyModeLayoutMayNeedRebind(msg Msg) bool {
	switch msg.(type) {
	case HostResizeMsg,
		LiveResizeResultMsg,
		ShellSetPanelPresentationMsg,
		ShellTogglePanelPresentationMsg,
		ShellSetHeaderVisibleMsg,
		ShellToggleHeaderVisibleMsg,
		ShellSetFooterVisibleMsg,
		ShellToggleFooterVisibleMsg,
		ShellSplitActivePaneMsg,
		ShellPaneCommandMsg:
		return true
	default:
		return false
	}
}

func copyModeContentRect(root state.Root) (render.Rect, bool) {
	rect, ok := activeTerminalContentRect(root, render.Rect{})
	if ok {
		return rect, true
	}
	return render.Rect{}, false
}
