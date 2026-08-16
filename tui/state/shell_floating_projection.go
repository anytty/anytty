package state

func FloatingOverviewItems(root Root) []FloatingOverviewItem {
	shell := root.Shell.ReadonlyDefaults()
	floatings := shell.ActiveFloatings()
	activeFloatingID := shell.ActiveFloatingID()
	items := make([]FloatingOverviewItem, 0, len(floatings))
	for _, floating := range floatings {
		terminalID := floatingOverviewTerminalID(root, floating)
		title, stateText, cols, rows := floatingOverviewTerminalProjection(root, floating, terminalID)
		item := FloatingOverviewItem{
			FloatingID: floating.ID,
			Title:      title,
			PaneID:     floating.Pane.ID,
			PaneKind:   floating.Pane.Kind,
			TerminalID: terminalID,
			State:      stateText,
			Cols:       cols,
			Rows:       rows,
			Rect:       floating.Rect,
			Z:          floating.Z,
			Active:     floating.ID == activeFloatingID,
			Collapsed:  floating.Collapsed,
			FitMode:    floating.FitMode,
		}
		if item.Title == "" {
			item.Title = floating.ID
		}
		items = append(items, item)
	}
	if len(items) > 0 {
		selected := shell.Overlay.SelectedIndex
		if selected < 0 {
			selected = 0
		}
		if selected >= len(items) {
			selected = len(items) - 1
		}
		items[selected].Selected = true
	}
	return items
}

func floatingOverviewTerminalID(root Root, floating FloatingPaneState) string {
	if binding, ok := root.TerminalViews.FloatingBinding(floating.ID); ok && binding.TerminalID != "" {
		return binding.TerminalID
	}
	return pickerTerminalID(root, floating.Pane)
}

func floatingOverviewTerminalProjection(root Root, floating FloatingPaneState, terminalID string) (string, string, int, int) {
	title := ""
	stateText := string(floating.Pane.Kind)
	cols, rows := floating.Rect.W-2, floating.Rect.H-2
	if terminalID != "" {
		ref := LocalTerminalRef(terminalID)
		if binding, ok := root.TerminalViews.FloatingBinding(floating.ID); ok && binding.TerminalID != "" {
			ref = binding.TerminalRef()
		}
		// 中文说明：overview 只做展示投影，terminal lifecycle/size 仍以 core/pool/live/binding 为来源。
		if poolItem, ok := terminalPoolItemByRef(root.TerminalPool, ref); ok {
			title = terminalPoolTitle(poolItem)
			stateText = poolItem.State
			cols, rows = poolItem.Cols, poolItem.Rows
		} else {
			surface := root.Surface.SurfaceForTerminalRef(ref)
			if surface.Title != "" {
				title = surface.Title
			}
			if surface.State != "" && surface.State != TerminalLivePending {
				stateText = string(surface.State)
			}
			if surface.Cols > 0 {
				cols = surface.Cols
			}
			if surface.Rows > 0 {
				rows = surface.Rows
			}
		}
		if binding, ok := root.TerminalViews.FloatingBinding(floating.ID); ok {
			if title == "" {
				title = binding.TerminalID
			}
			if cols <= 0 {
				cols = binding.DesiredCols
			}
			if rows <= 0 {
				rows = binding.DesiredRows
			}
		}
	}
	if terminalID == "" {
		title = "unconnected"
		if floating.Pane.Kind == PaneEmpty || stateText == "" {
			stateText = "empty"
		}
	}
	if title == "" && terminalID == "" {
		title = floating.Title
	}
	if title == "" && terminalID == "" {
		title = floating.Pane.Title
	}
	if title == "" {
		title = terminalID
	}
	if title == "" {
		title = floating.ID
	}
	if stateText == "" {
		stateText = "floating"
	}
	return title, stateText, cols, rows
}

func terminalPoolItemByRef(pool TerminalPoolStore, ref TerminalRef) (TerminalPoolItem, bool) {
	ref = ref.Normalize()
	if ref.Empty() {
		return TerminalPoolItem{}, false
	}
	for _, item := range pool.Items {
		if item.TerminalRef().Equal(ref) {
			return item, true
		}
	}
	return TerminalPoolItem{}, false
}
