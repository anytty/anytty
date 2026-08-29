package render

import (
	"fmt"
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

type RenderVM struct {
	Shell ShellVM
	Theme Theme
}

type HitRegionKind string

// HitTargetMode 声明可执行命中区如何解析业务目标。
// Active 复用 keyboard 的当前目标语义；Explicit 要求 producer 提供 pane/row 等 surface context，
// 目标缺失时 app 必须 fail closed，不能回退到 active target。
type HitTargetMode string

const (
	HitTargetActive   HitTargetMode = "active"
	HitTargetExplicit HitTargetMode = "explicit"
)

const (
	HitRegionHistoryRow    HitRegionKind = "history-row"
	HitRegionStatus        HitRegionKind = "status"
	HitRegionOverlay       HitRegionKind = "overlay"
	HitRegionToast         HitRegionKind = "toast"
	HitRegionToastClose    HitRegionKind = "toast-close"
	HitRegionPaneChrome    HitRegionKind = "pane-chrome"
	HitRegionPaneAction    HitRegionKind = "pane-action"
	HitRegionPaneResize    HitRegionKind = "pane-resize"
	HitRegionPaneContent   HitRegionKind = "pane-content"
	HitRegionContentAction HitRegionKind = "content-action"
)

type Rect struct {
	X int
	Y int
	W int
	H int
}

type ResizeGroupItem struct {
	PaneID    string
	Cells     int
	DeltaSign int
}

type HitRegion struct {
	Kind               HitRegionKind
	Rect               Rect
	LineID             uint64
	Row                int
	HasRow             bool
	PaneID             string
	Floating           bool
	ActionID           string
	Invocation         actiondomain.Invocation
	TargetMode         HitTargetMode
	Direction          string
	SplitPath          string
	ResizeBeforePaneID string
	ResizeAfterPaneID  string
	ResizeBeforeCells  int
	ResizeAfterCells   int
	ResizeGroup        []ResizeGroupItem
}

type RenderVMBuilder struct{}

func NewRenderVMBuilder() RenderVMBuilder {
	return RenderVMBuilder{}
}

func buildHeaderVM(shell state.ShellStore, root state.Root) HeaderVM {
	notice := root.Surface.Err
	if root.Session.LastError != "" {
		notice = root.Session.LastError
	}
	return HeaderVM{
		Visible:           shell.HeaderVisible,
		Workspace:         shell.Workspace.Name,
		Tab:               tabStripSummary(shell),
		Tabs:              buildHeaderTabVMs(shell),
		WorkspaceTemplate: root.Config.Chrome.WorkspaceTemplate,
		TabTemplate:       root.Config.Chrome.TabTemplate,
		TabCreateIcon:     root.Config.Chrome.TabCreateIcon,
		TabCreateTemplate: root.Config.Chrome.TabCreateTemplate,
		ActivePane:        shell.ActivePaneID,
		TerminalSummary:   terminalSummary(root),
		FloatingSummary:   floatingSummary(shell),
		Notice:            notice,
		Title:             shell.Workspace.Name,
	}
}

func buildFooterVM(root state.Root, shell state.ShellStore, content ContentVM) FooterVM {
	shell = shell.ReadonlyDefaults()
	mode := footerMode(root, shell)
	summary := globalSummary(root, shell)
	if shell.Overlay.Open && shell.Overlay.Kind == state.OverlayHelp {
		summary = ""
	}
	hint := ""
	if target, ok := shell.ActiveSurfaceTarget(); ok && !target.Floating {
		hint = content.Status
	}
	if hint == "" {
		hint = activeViewLiveStatus(root, shell)
	}
	footerConfig := root.Config.Footer
	modeConfig := footerConfig.Modes[mode]
	return FooterVM{
		Visible:                          shell.FooterVisible || content.Search.Visible,
		Search:                           content.Search,
		Mode:                             mode,
		ModeIcon:                         modeConfig.Icon,
		ModeLabel:                        modeConfig.Label,
		ModeStyle:                        footerStyleTokenFromConfig(modeConfig.Style),
		Hint:                             hint,
		ActionTokens:                     footerActionCatalogForRoot(mode, root, shell),
		KeyTemplate:                      footerKeyTemplateForConfig(root.Config),
		KeyTemplateSet:                   true,
		ActionTemplate:                   footerTemplateOrDefault(footerConfig.Templates.Action, defaultFooterActionTemplate),
		ModeBadgeTemplate:                footerTemplateOrDefault(footerConfig.Templates.ModeBadge, defaultFooterModeBadgeTemplate),
		ActionSeparator:                  footerConfig.Templates.Separator,
		WorkspaceSummaryTemplate:         footerTemplateOrDefault(footerConfig.Templates.WorkspaceSummary, defaultFooterWorkspaceSummaryTemplate),
		FloatingSummaryTemplate:          footerTemplateOrDefault(footerConfig.Templates.FloatingSummary, defaultFooterFloatingSummaryTemplate),
		FloatingCollapsedSummaryTemplate: footerTemplateOrDefault(footerConfig.Templates.FloatingCollapsedSummary, defaultFooterFloatingCollapsedSummaryTemplate),
		TerminalsSummaryTemplate:         footerTemplateOrDefault(footerConfig.Templates.TerminalsSummary, defaultFooterTerminalsSummaryTemplate),
		TabsSummaryTemplate:              footerTemplateOrDefault(footerConfig.Templates.TabsSummary, defaultFooterTabsSummaryTemplate),
		PanesSummaryTemplate:             footerTemplateOrDefault(footerConfig.Templates.PanesSummary, defaultFooterPanesSummaryTemplate),
		KeylockOnTemplate:                footerConfig.Templates.KeylockOn,
		KeylockOffTemplate:               footerConfig.Templates.KeylockOff,
		ActiveTarget:                     activeTargetSummary(shell, root),
		GlobalSummary:                    summary,
		FloatingSummaryOpen:              len(shell.ActiveFloatings()) > 0,
	}
}

const (
	defaultFooterActionTemplate                   = "{{key}} {{icon}} {{label}}"
	defaultFooterKeyTemplate                      = "{{key}}"
	defaultFooterModeBadgeTemplate                = "{{mode_icon}} {{mode_label}}"
	defaultFooterWorkspaceSummaryTemplate         = "ws:{{workspace}}"
	defaultFooterFloatingSummaryTemplate          = "float:{{count}}"
	defaultFooterFloatingCollapsedSummaryTemplate = "collapsed:{{count}}"
	defaultFooterTerminalsSummaryTemplate         = "terminals:{{count}}"
	defaultFooterTabsSummaryTemplate              = "tabs:{{count}}"
	defaultFooterPanesSummaryTemplate             = "panes:{{count}}"
)

func footerTemplateOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func footerStyleTokenFromConfig(value string) StyleToken {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return StyleToken(value)
}

func footerKeyTemplateForConfig(cfg state.TUIConfigStore) string {
	if cfg.Version == 0 && strings.TrimSpace(cfg.Footer.Templates.Key) == "" {
		return defaultFooterKeyTemplate
	}
	return cfg.Footer.Templates.Key
}

func activeViewLiveStatus(root state.Root, shell state.ShellStore) string {
	target, ok := shell.ActiveSurfaceTarget()
	if !ok {
		return ""
	}
	var binding state.TerminalViewBinding
	if target.Floating {
		binding, ok = root.TerminalViews.FloatingBinding(target.FloatingID)
	} else {
		binding, ok = root.TerminalViews.PaneBinding(target.PaneID)
	}
	if !ok || binding.TerminalID == "" {
		return ""
	}
	return liveStatus(surfaceForBinding(root, binding), sessionForBinding(root, binding))
}

func terminalSummary(root state.Root) string {
	count := terminalCount(root)
	if count == 0 {
		return "term:0"
	}
	return fmt.Sprintf("term:%d", count)
}

func terminalCount(root state.Root) int {
	ids := map[string]struct{}{}
	for terminalID := range root.Surface.Surfaces {
		if terminalID != "" {
			ids[terminalID] = struct{}{}
		}
	}
	for _, binding := range root.TerminalViews.Views {
		if binding.TerminalID != "" {
			ids[binding.TerminalID] = struct{}{}
		}
	}
	if root.Surface.TerminalID != "" {
		ids[root.Surface.TerminalID] = struct{}{}
	}
	if root.Session.TerminalID != "" {
		ids[root.Session.TerminalID] = struct{}{}
	}
	for _, terminalID := range root.CopyHistoryTerminalIDs() {
		ids[terminalID] = struct{}{}
	}
	return len(ids)
}

func floatingSummary(shell state.ShellStore) string {
	shell = shell.ReadonlyDefaults()
	floatings := shell.ActiveFloatings()
	collapsed := 0
	for _, floating := range floatings {
		if floating.Collapsed {
			collapsed++
		}
	}
	if collapsed > 0 {
		return fmt.Sprintf("float:%d collapsed:%d", len(floatings), collapsed)
	}
	return fmt.Sprintf("float:%d", len(floatings))
}

func footerMode(root state.Root, shell state.ShellStore) string {
	if shell.Overlay.Open {
		if shell.Overlay.Kind == state.OverlayTerminalPicker && shell.Overlay.TerminalPickerView == state.TerminalPickerViewEndpoints {
			return "terminal-picker-endpoints"
		}
		if shell.Overlay.Kind == state.OverlayTerminalPicker && shell.Overlay.TerminalPickerView == state.TerminalPickerViewTags {
			return "terminal-picker-tags"
		}
		return string(shell.Overlay.Kind)
	}
	if root.ActiveViewOwnsCopyInput() {
		return "copy"
	}
	if shell.InteractionMode != state.InteractionModeNormal {
		return string(shell.InteractionMode)
	}
	return "live"
}

func footerActionCatalogForRoot(mode string, root state.Root, shell state.ShellStore) []FooterActionVM {
	tokens := footerActionCatalogFromShortcuts(mode, root)
	if root.CurrentBackNavigationLayer() != state.BackNavigationNone {
		back := FooterActionVM{Key: "Esc", Label: "BACK", ActionID: "shortcut.exit", Style: StyleStatusWarning, Click: ClickHintOnly}
		tokens = append(tokens, back)
	}
	out := make([]FooterActionVM, 0, len(tokens))
	for _, token := range tokens {
		if footerActionAvailable(token, mode, root, shell) {
			out = append(out, token)
		}
	}
	if target, ok := shell.ActiveSurfaceTarget(); ok && target.Floating && mode == string(state.InteractionModeResize) {
		return floatingResizeFooterActions(out)
	}
	return out
}

func floatingResizeFooterActions(actions []FooterActionVM) []FooterActionVM {
	out := make([]FooterActionVM, 0, 7)
	moveKeys := ""
	var move FooterActionVM
	for _, id := range []actiondomain.ID{"resize.left", "resize.up", "resize.right", "resize.down"} {
		if action, ok := footerActionForID(actions, id); ok {
			if moveKeys == "" {
				move = action
			}
			moveKeys += primaryFooterShortcutKey(action.Key)
		}
	}
	if moveKeys != "" {
		move.Key = moveKeys
		move.Label = "MOVE"
		move.ActionID = "floating.position"
		move.Invocation = actiondomain.Invocation{}
		move.Click = ClickHintOnly
		out = append(out, move)
	} else if action, ok := footerActionForID(actions, "resize.pan_left"); ok {
		action.Key = compactFooterShortcutKeys(action.Key)
		action.Label = "MOVE"
		action.ActionID = "floating.position"
		action.Invocation = actiondomain.Invocation{}
		action.Click = ClickHintOnly
		out = append(out, action)
	}
	for _, spec := range []struct {
		id      actiondomain.ID
		compact bool
	}{
		{id: "resize.align_left", compact: true},
		{id: "resize.center", compact: true},
		{id: "panel.take_owner"},
		{id: "panel.size_lock"},
		{id: "resize.layout_reset"},
		{id: "shortcut.exit"},
	} {
		action, ok := footerActionForID(actions, spec.id)
		if !ok {
			continue
		}
		if spec.compact {
			action.Key = compactFooterShortcutKeys(action.Key)
		}
		out = append(out, action)
	}
	return out
}

func footerActionForID(actions []FooterActionVM, id actiondomain.ID) (FooterActionVM, bool) {
	for _, action := range actions {
		if action.Invocation.ID == id || action.ActionID == id.String() {
			return action, true
		}
	}
	return FooterActionVM{}, false
}

func primaryFooterShortcutKey(key string) string {
	key = strings.TrimSpace(key)
	if before, _, ok := strings.Cut(key, "/"); ok {
		return before
	}
	return key
}

func compactFooterShortcutKeys(key string) string {
	return strings.ReplaceAll(strings.TrimSpace(key), "/", "")
}

// 中文说明：这里仅过滤可见 footer token；reducer 仍负责最终语义校验和错误反馈。
func footerActionAvailable(action FooterActionVM, mode string, root state.Root, shell state.ShellStore) bool {
	actionID := action.Invocation.ID
	if actionID == "" {
		actionID = actiondomain.ID(action.ActionID)
	}
	if actionID == "" {
		return true
	}
	activeTarget, hasActiveTarget := shell.ActiveSurfaceTarget()
	switch actionID {
	case "panel.close":
		if hasActiveTarget && activeTarget.Floating {
			return true
		}
		return activeTabPaneCount(shell) > 1
	case "panel.detach":
		if !hasActiveTarget {
			return false
		}
		var binding state.TerminalViewBinding
		var ok bool
		if activeTarget.Floating {
			binding, ok = root.TerminalViews.FloatingBinding(activeTarget.FloatingID)
		} else {
			binding, ok = root.TerminalViews.PaneBinding(activeTarget.PaneID)
		}
		return ok && binding.TerminalID != ""
	case "panel.focus_next", "panel.focus_prev":
		return hasActiveTarget && !activeTarget.Floating && activeTabPaneCount(shell) > 1
	case "panel.split_right", "panel.split_down", "panel.toggle_zoom", "panel.presentation_card", "panel.presentation_split_line":
		return hasActiveTarget && !activeTarget.Floating
	case "panel.balance":
		return hasActiveTarget && !activeTarget.Floating && activeTabPaneCount(shell) > 1
	case "resize.left", "resize.left_large", "resize.right", "resize.right_large", "resize.up", "resize.up_large", "resize.down", "resize.down_large":
		return hasActiveTarget && (activeTarget.Floating || activeTabPaneCount(shell) > 1)
	case "resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down",
		"resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom",
		"resize.center", "resize.center_x", "resize.center_y", "resize.layout_reset":
		return hasActiveTarget
	case "resize.layout_toggle":
		return hasActiveTarget && !activeTarget.Floating
	case "tab.previous", "tab.next", "tab.close", "tab.kill":
		return activeWorkspaceTabCount(shell) > 1
	case "workspace.previous", "workspace.next", "workspace.delete":
		return len(shell.ReadonlyDefaults().Workspaces) > 1
	case "system.close_toast", "system.clear_toasts":
		return len(shell.ReadonlyDefaults().Toasts) > 0
	case "floating.summon", "floating.toggle_all", "floating_overview.show_all", "floating_overview.collapse_all":
		return len(shell.ActiveFloatings()) > 0
	case "floating.take_owner", "floating.fit", "floating.auto_fit", "floating.center", "floating.collapse", "floating.close":
		return shell.ActiveFloatingID() != "" || mode == string(state.OverlayFloatingOverview) && len(shell.ActiveFloatings()) > 0
	case "terminal_pool.attach", "terminal_pool.attach_tab", "terminal_pool.attach_float", "terminal_pool.restart", "terminal_pool.edit", "terminal_pool.kill", "terminal_pool.delete":
		if mode == string(state.OverlayTerminalPool) {
			return len(state.TerminalPoolPageItems(root)) > 0
		}
		return len(root.TerminalPool.Items) > 0
	case "terminal_picker.select_previous", "terminal_picker.select_next":
		if shell.Overlay.TerminalPickerView == state.TerminalPickerViewEndpoints {
			return len(state.TerminalPickerVisibleEndpointOptions(root)) > 1
		}
		if shell.Overlay.TerminalPickerView == state.TerminalPickerViewTags {
			return len(state.TerminalPickerVisibleTagOptions(root)) > 1
		}
		return len(state.TerminalPickerItems(root)) > 1
	case "terminal_picker.tag_toggle":
		return shell.Overlay.TerminalPickerView == state.TerminalPickerViewTags && len(state.TerminalPickerVisibleTagOptions(root)) > 0
	case "terminal_picker.endpoint_choose":
		return shell.Overlay.TerminalPickerView == state.TerminalPickerViewEndpoints && len(state.TerminalPickerVisibleEndpointOptions(root)) > 0
	case "terminal_picker.attach":
		return shell.Overlay.TerminalPickerView == state.TerminalPickerViewList && len(state.TerminalPickerItems(root)) > 0
	case "terminal_picker.split", "terminal_picker.edit", "terminal_picker.kill", "terminal_picker.delete":
		if shell.Overlay.TerminalPickerView != state.TerminalPickerViewList {
			return false
		}
		items := state.TerminalPickerItems(root)
		for _, item := range items {
			if item.Selected {
				return item.TerminalID != ""
			}
		}
		return false
	case "clipboard_history.paste", "clipboard_history.edit", "clipboard_history.delete":
		return len(state.ClipboardHistoryItems(root)) > 0
	case "help.previous", "help.page_up", "help.first":
		return shell.Overlay.Kind == state.OverlayHelp && shell.Overlay.SelectedIndex > 0
	case "help.next", "help.page_down", "help.last":
		itemCount := len(input.ShortcutEntriesForHelp(root.Config.Shortcuts, root.HostCapabilities.KeyboardDisambiguation))
		return shell.Overlay.Kind == state.OverlayHelp && shell.Overlay.SelectedIndex < itemCount-1
	default:
		return true
	}
}

func activeWorkspaceTabCount(shell state.ShellStore) int {
	shell = shell.ReadonlyDefaults()
	return len(shell.Workspace.Tabs)
}

func activeTabPaneCount(shell state.ShellStore) int {
	shell = shell.ReadonlyDefaults()
	activeTabID := shell.Workspace.ActiveTabID
	for _, tab := range shell.Workspace.Tabs {
		if activeTabID == "" || tab.ID == activeTabID {
			return len(tab.Panes)
		}
	}
	return 0
}

func activeTargetSummary(shell state.ShellStore, root state.Root) string {
	target, ok := shell.ActiveSurfaceTarget()
	if !ok {
		return ""
	}
	prefix := "pane:"
	pane := state.PaneState{ID: target.PaneID}
	terminalState := ""
	if target.Floating {
		prefix = "float:"
		if floating, found := shell.FloatingByID(target.FloatingID); found {
			pane = floating.Pane
			if floating.Title != "" {
				pane.Title = floating.Title
			}
		}
		terminalState = floatingTerminalStateSummary(root, target.FloatingID)
	} else {
		if activePane, found := shell.Pane(state.PaneCommandTarget{PaneID: target.PaneID}); found {
			pane = activePane
		}
		terminalState = terminalStateSummary(root, pane)
	}
	title := pane.ID
	if pane.Title != "" {
		title = pane.Title
	}
	if terminalState == "" {
		return prefix + title
	}
	return prefix + title + " " + terminalState
}

func globalSummary(root state.Root, shell state.ShellStore) string {
	summary := fmt.Sprintf("ws:%s %s terminals:%d", shell.Workspace.Name, floatingSummary(shell), terminalCount(root))
	if shell.ShortcutPassthroughLocked {
		summary += " keylock:on"
	}
	return summary
}

func tabStripSummary(shell state.ShellStore) string {
	shell = shell.ReadonlyDefaults()
	if len(shell.Workspace.Tabs) == 0 {
		return ""
	}
	out := ""
	for index, tab := range shell.Workspace.Tabs {
		if index > 0 {
			out += " "
		}
		title := tab.Title
		if title == "" {
			title = tab.ID
		}
		if tab.ID == shell.Workspace.ActiveTabID {
			out += "[" + title + "]"
		} else {
			out += title
		}
	}
	return out
}

func buildHeaderTabVMs(shell state.ShellStore) []HeaderTabVM {
	shell = shell.ReadonlyDefaults()
	tabs := shell.Workspace.Tabs
	if len(tabs) == 0 {
		return nil
	}
	out := make([]HeaderTabVM, 0, len(tabs))
	for index, tab := range tabs {
		title := tab.Title
		if title == "" {
			title = tab.ID
		}
		if title == "" {
			title = fmt.Sprintf("tab-%d", index+1)
		}
		out = append(out, HeaderTabVM{
			ID:            tab.ID,
			Title:         title,
			Index:         index + 1,
			Active:        tab.ID == shell.Workspace.ActiveTabID || (shell.Workspace.ActiveTabID == "" && index == 0),
			CloseActionID: ActionTabClose.String(),
			CloseTargetID: tab.ID,
		})
	}
	return out
}

func terminalStateSummary(root state.Root, pane state.PaneState) string {
	binding, hasBinding := root.TerminalViews.PaneBinding(pane.ID)
	return terminalBindingStateSummary(root, binding, hasBinding, copyModeBelongsToPane(root, pane.ID))
}

func floatingTerminalStateSummary(root state.Root, floatingID string) string {
	binding, hasBinding := root.TerminalViews.FloatingBinding(floatingID)
	return terminalBindingStateSummary(root, binding, hasBinding, copyModeBelongsToFloating(root, floatingID))
}

func terminalBindingStateSummary(root state.Root, binding state.TerminalViewBinding, hasBinding bool, copyMode bool) string {
	surface := state.TerminalSurfaceStore{}
	session := state.TerminalSessionStore{}
	if hasBinding && binding.TerminalID != "" {
		surface = surfaceForBinding(root, binding)
		session = sessionForBinding(root, binding)
	}
	switch {
	case session.LastError != "" || surface.Err != "":
		return "error"
	case copyMode:
		return "copy"
	case session.State == state.TerminalLiveExited || surface.State == state.TerminalLiveExited:
		return "exited"
	case hasBinding && binding.Attached:
		return "attached"
	case surface.TerminalID != "":
		return "live"
	case hasBinding && binding.TerminalID != "":
		return "bound"
	default:
		return ""
	}
}

func (projector ShellProjector) buildLayoutVM(shell state.ShellStore, activeContent ContentVM, root state.Root) LayoutVM {
	if shell.ZoomedPaneID != "" {
		return LayoutVM{
			Viewport:    viewportRect(root.Viewport),
			Panels:      projector.buildZoomedPanelVMs(shell, activeContent, root),
			BodyContent: activeContent,
			Floating:    projector.buildFloatingVMs(shell, root),
			Split:       SplitVM{PaneID: shell.ZoomedPaneID},
		}
	}
	return LayoutVM{
		Viewport:    viewportRect(root.Viewport),
		Panels:      projector.buildPanelVMs(shell, activeContent, root),
		BodyContent: activeContent,
		Floating:    projector.buildFloatingVMs(shell, root),
		Split:       buildSplitVM(activeTab(shell).RootSplit),
	}
}

func viewportRect(viewport state.ViewportStore) Rect {
	if !viewport.Valid {
		return Rect{}
	}
	return Rect{W: viewport.Cols, H: viewport.Rows}
}

func (projector ShellProjector) buildPanelVMs(shell state.ShellStore, activeContent ContentVM, root state.Root) []PanelVM {
	shell = shell.ReadonlyDefaults()
	tab := activeTab(shell)
	floatingOwnsFocus := shell.ActiveFloatingID() != ""
	if len(tab.Panes) == 0 {
		return nil
	}
	panels := make([]PanelVM, len(tab.Panes))
	for i, pane := range tab.Panes {
		active := pane.ID == shell.ActivePaneID
		focused := active && !floatingOwnsFocus
		content := projector.contentForPane(root, pane, activeContent, focused)
		if focused && !copyModeBelongsToPane(root, pane.ID) {
			content = activeContent
		}
		content = contentWithPaneLayout(root, pane, content)
		panels[i] = PanelVM{
			ID:           pane.ID,
			Title:        activePaneTitle(pane),
			Presentation: renderPanelPresentation(shell.PanelPresentation),
			Active:       focused,
			Content:      content,
			Chrome:       buildPanelChromeVM(root, pane, focused, content),
		}
	}
	return panels
}

func (projector ShellProjector) buildZoomedPanelVMs(shell state.ShellStore, activeContent ContentVM, root state.Root) []PanelVM {
	shell = shell.ReadonlyDefaults()
	floatingOwnsFocus := shell.ActiveFloatingID() != ""
	for _, pane := range activeTab(shell).Panes {
		if pane.ID == shell.ZoomedPaneID {
			focused := pane.ID == shell.ActivePaneID && !floatingOwnsFocus
			content := projector.contentForPane(root, pane, activeContent, focused)
			return []PanelVM{{
				ID:           pane.ID,
				Title:        activePaneTitle(pane),
				Presentation: renderPanelPresentation(shell.PanelPresentation),
				Active:       focused,
				IsZoomMode:   true,
				Content:      content,
				Chrome:       buildPanelChromeVMWithZoom(root, pane, focused, content, true),
			}}
		}
	}
	return projector.buildPanelVMs(shell, activeContent, root)
}

func (projector ShellProjector) buildFloatingVMs(shell state.ShellStore, root state.Root) []FloatingVM {
	shell = shell.ReadonlyDefaults()
	floatings := shell.ActiveFloatings()
	if len(floatings) == 0 {
		return nil
	}
	out := make([]FloatingVM, 0, len(floatings))
	for _, floating := range floatings {
		content := projector.contentForFloating(root, shell, floating)
		content = contentWithFloatingLayout(root, floating, content)
		out = append(out, FloatingVM{
			ID:        floating.ID,
			PaneID:    floating.Pane.ID,
			Title:     floating.Title,
			Rect:      Rect{X: floating.Rect.X, Y: floating.Rect.Y, W: floating.Rect.W, H: floating.Rect.H},
			Z:         floating.Z,
			Active:    floating.Active,
			Collapsed: floating.Collapsed,
			Chrome:    floatingChromeVM(root, floating, content),
			Content:   content,
		})
	}
	return out
}

func contentWithPaneLayout(root state.Root, pane state.PaneState, content ContentVM) ContentVM {
	if content.Kind != ContentTerminalLive && content.Kind != ContentCopyHistory {
		return content
	}
	binding, ok := root.TerminalViews.PaneBinding(pane.ID)
	if !ok {
		return content
	}
	content.Layout = contentLayoutVMFromState(binding.Layout)
	return content
}

func contentWithFloatingLayout(root state.Root, floating state.FloatingPaneState, content ContentVM) ContentVM {
	if content.Kind != ContentTerminalLive && content.Kind != ContentCopyHistory {
		return content
	}
	binding, ok := root.TerminalViews.FloatingBinding(floating.ID)
	if !ok {
		return content
	}
	content.Layout = contentLayoutVMFromState(binding.Layout)
	return content
}

func contentLayoutVMFromState(layout state.TerminalViewLayout) ContentLayoutVM {
	layout = layout.Normalize()
	return ContentLayoutVM{Known: true, Mode: layout.Mode, PanX: layout.PanX, PanY: layout.PanY, AlignX: layout.AlignX, AlignY: layout.AlignY}
}

func floatingChromeVM(root state.Root, floating state.FloatingPaneState, content ContentVM) FloatingChromeVM {
	style := StyleMuted
	if floating.Active {
		style = StyleAccent
	}
	return FloatingChromeVM{
		FillOverlay:      true,
		ShowResizeHandle: false,
		Terminal:         terminalChromeVMForFloating(root, floating, content, style),
		Actions:          defaultFloatingChromeActionVMs(style),
	}
}

func buildPanelChromeVM(root state.Root, pane state.PaneState, active bool, content ContentVM) PanelChromeVM {
	return buildPanelChromeVMWithZoom(root, pane, active, content, false)
}

func buildPanelChromeVMWithZoom(root state.Root, pane state.PaneState, active bool, content ContentVM, zoomMode bool) PanelChromeVM {
	style := StyleMuted
	if active {
		style = StyleAccent
	}
	actions := defaultPaneChromeActionVMsForZoom(style, zoomMode)
	terminal := terminalChromeVM(root, pane, active, content, style)
	var meta []ChromeSlotVM
	if active && content.Kind == ContentCopyHistory && content.Status != "" && !content.Search.Visible {
		meta = append(meta, ChromeSlotVM{Text: compactCopyHistoryChromeStatus(content.Status), Style: StyleMuted})
	}
	return PanelChromeVM{
		Title:    ChromeSlotVM{Text: activePaneTitle(pane), Style: style},
		State:    paneChromeStateSlot(active, content),
		Meta:     meta,
		Terminal: terminal,
		Actions:  actions,
	}
}

func compactCopyHistoryChromeStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return ""
	}
	if DisplayWidth(status) <= 42 {
		return status
	}
	return TruncateCells(status, 42)
}

func terminalChromeVM(root state.Root, pane state.PaneState, active bool, content ContentVM, style StyleToken) TerminalChromeVM {
	binding, ok := root.TerminalViews.PaneBinding(pane.ID)
	return terminalChromeVMFromBinding(root, pane, binding, ok, active, content, style)
}

func terminalChromeVMForFloating(root state.Root, floating state.FloatingPaneState, content ContentVM, style StyleToken) TerminalChromeVM {
	binding, ok := root.TerminalViews.FloatingBinding(floating.ID)
	pane := floating.Pane
	if strings.TrimSpace(pane.Title) == "" {
		pane.Title = floating.Title
	}
	return terminalChromeVMFromBinding(root, pane, binding, ok, floating.Active, content, style)
}

func terminalChromeVMFromBinding(root state.Root, pane state.PaneState, binding state.TerminalViewBinding, ok bool, active bool, content ContentVM, style StyleToken) TerminalChromeVM {
	if !ok || binding.TerminalID == "" {
		return TerminalChromeVM{}
	}
	role := binding.ResizeRole
	if role == "" {
		role = state.TerminalResizeRoleFollower
	}
	ownerText := paneChromeTakeOwnerText()
	ownerStyle := StyleMuted
	projectedOwner := binding.HasProjectedResizeOwner()
	if root.Shell.ReadonlyDefaults().OwnerConfirm.ViewID == binding.ViewID || binding.OwnerAcquirePending {
		ownerText = paneChromeOwnerPendingText()
		ownerStyle = StyleWarning
	} else if projectedOwner {
		ownerText = paneChromeOwnerText()
		ownerStyle = StyleSuccess
	}
	title := terminalChromeTitle(root, pane, binding)
	layout := binding.Layout.Normalize()
	return TerminalChromeVM{
		Locked:       binding.SizeLocked,
		LayoutMode:   layout.Mode,
		PanX:         layout.PanX,
		PanY:         layout.PanY,
		AlignX:       layout.AlignX,
		AlignY:       layout.AlignY,
		Title:        ChromeSlotVM{Text: title, Style: style},
		State:        terminalChromeStateSlot(root, binding, active, content),
		AttachCount:  terminalChromeAttachCount(root, binding),
		Owner:        ChromeSlotVM{Text: ownerText, Style: ownerStyle},
		TakeOwner:    !projectedOwner && !binding.OwnerAcquirePending,
		CanLockSize:  projectedOwner,
		ResizeRole:   role,
		CanResize:    binding.CanResize,
		TerminalID:   binding.TerminalID,
		TerminalView: binding.ViewID,
	}
}

func terminalChromeAttachCount(root state.Root, binding state.TerminalViewBinding) int {
	ref := binding.TerminalRef()
	count := 0
	for _, item := range root.TerminalPool.Items {
		if item.TerminalRef().Equal(ref) {
			count = item.AttachmentCount
			break
		}
	}
	localAttached := 0
	for _, candidate := range root.TerminalViews.BindingsForTerminalRef(ref) {
		if candidate.Attached && candidate.Channel != 0 {
			localAttached++
		}
	}
	if localAttached > count {
		count = localAttached
	}
	return count
}

func defaultFloatingChromeActionVMs(style StyleToken) []ChromeActionVM {
	return []ChromeActionVM{
		paneChromeActionVM(ActionFloatingCenter, style),
		paneChromeActionVM(ActionFloatingCollapse, style),
		paneChromeActionVM(ActionPaneZoom, style),
		paneChromeActionVM(ActionFloatingClose, style),
	}
}

func terminalChromeTitle(root state.Root, pane state.PaneState, binding state.TerminalViewBinding) string {
	ref := binding.TerminalRef()
	title := terminalChromeDefaultTitle(root, pane, binding)
	if rendered, ok := renderTerminalChromeTitleTemplate(root.Config.Chrome.PaneTitleTemplate, terminalChromeTitleTemplateContext{
		Terminal:      title,
		TerminalTitle: title,
		TerminalID:    binding.TerminalID,
		Endpoint:      terminalChromeEndpointLabel(root, ref),
		EndpointLabel: terminalChromeEndpointLabel(root, ref),
		EndpointID:    string(ref.EndpointID),
		Pane:          strings.TrimSpace(pane.Title),
		PaneTitle:     strings.TrimSpace(pane.Title),
	}); ok && strings.TrimSpace(rendered) != "" {
		return strings.TrimSpace(rendered)
	}
	return title
}

func terminalChromeDefaultTitle(root state.Root, pane state.PaneState, binding state.TerminalViewBinding) string {
	ref := binding.TerminalRef()
	for _, item := range root.TerminalPool.Items {
		if !item.TerminalRef().Equal(ref) {
			continue
		}
		if title := strings.TrimSpace(item.Title); title != "" {
			return title
		}
		break
	}
	if title := strings.TrimSpace(pane.Title); title != "" {
		return title
	}
	return binding.TerminalID
}

func terminalChromeEndpointLabel(root state.Root, ref state.TerminalRef) string {
	if endpoint, ok := root.Endpoints.DisplayEndpoint(ref.EndpointID); ok {
		return endpoint.DisplayLabel()
	}
	ref = ref.Normalize()
	return string(ref.EndpointID)
}

func terminalChromeStateSlot(root state.Root, binding state.TerminalViewBinding, active bool, content ContentVM) ChromeSlotVM {
	style := StyleMuted
	if active {
		style = StyleSuccess
	}
	if content.Pending {
		style = StyleWarning
	}
	if content.Error != "" || content.Empty {
		style = StyleDanger
	}
	surface := surfaceForBinding(root, binding)
	if surface.State == state.TerminalLiveExited || surface.State == state.TerminalLiveError {
		style = StyleDanger
	}
	return ChromeSlotVM{Text: paneChromeRunningGlyph(), Style: style}
}

func paneChromeStateSlot(active bool, content ContentVM) ChromeSlotVM {
	switch {
	case content.Error != "":
		return ChromeSlotVM{Text: "error", Style: StyleDanger}
	case content.Pending:
		return ChromeSlotVM{Text: "pending", Style: StyleWarning}
	case content.Empty:
		return ChromeSlotVM{Text: "empty", Style: StyleMuted}
	case active:
		return ChromeSlotVM{Text: "active", Style: StyleSuccess}
	default:
		return ChromeSlotVM{Text: "idle", Style: StyleMuted}
	}
}

func defaultPaneChromeActionVMs(style StyleToken) []ChromeActionVM {
	return defaultPaneChromeActionVMsForZoom(style, false)
}

func defaultPaneChromeActionVMsForZoom(style StyleToken, zoomMode bool) []ChromeActionVM {
	if zoomMode {
		return []ChromeActionVM{
			paneChromeZoomActionVM(style, true),
			paneChromeActionVMWithZoomMode(ActionPaneClose, style, true),
		}
	}
	return []ChromeActionVM{
		paneChromeZoomActionVM(style, false),
		paneChromeActionVM(ActionPaneSplitRight, style),
		paneChromeActionVM(ActionPaneSplitDown, style),
		paneChromeActionVM(ActionPaneClose, style),
	}
}

func paneChromeActionVM(id ProjectionID, style StyleToken) ChromeActionVM {
	return paneChromeActionVMWithZoomMode(id, style, false)
}

func paneChromeActionVMWithZoomMode(id ProjectionID, style StyleToken, zoomMode bool) ChromeActionVM {
	spec, ok := ProjectionByID(id)
	if !ok {
		return ChromeActionVM{ActionID: id.String(), Style: style, IsZoomMode: zoomMode}
	}
	return ChromeActionVM{Text: spec.ChromeGlyph, ActionID: spec.ID.String(), Label: projectionActionLabel(spec), Style: style, IsZoomMode: zoomMode}
}

func paneChromeZoomActionVM(style StyleToken, zoomMode bool) ChromeActionVM {
	action := paneChromeActionVMWithZoomMode(ActionPaneZoom, style, zoomMode)
	if zoomMode {
		action.Text = paneChromeUnzoomGlyph()
		action.Label = "unzoom"
	}
	return action
}

func (projector ShellProjector) buildActiveContentVM(root state.Root, shell state.ShellStore) ContentVM {
	shell = shell.ReadonlyDefaults()
	if floatingID := shell.ActiveFloatingID(); floatingID != "" && copyModeBelongsToFloating(root, floatingID) {
		return projector.copyHistoryContentForView(root, shell, copyHistoryViewIDForFloating(root, floatingID), state.PaneState{}, true)
	}
	if pane, ok := shell.Pane(state.PaneCommandTarget{PaneID: shell.ActivePaneID}); ok {
		if copyModeBelongsToPane(root, pane.ID) {
			return projector.copyHistoryContentForPane(root, shell, pane, true)
		}
		if pane.Kind == state.PaneEmpty && !paneHasTerminalBinding(root, pane.ID) {
			return buildEmptyPaneContentWithSelection(pane, shell.EmptyPaneCTA.SelectedIndex)
		}
		if root.Session.LastError != "" && root.Session.TerminalID == "" {
			return projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Pane: pane, Kind: contentKindForPaneRoot(root, pane), Session: root.Session, Active: true})
		}
		surface, session := terminalContentStoresForPane(root, pane)
		binding, _ := root.TerminalViews.PaneBinding(pane.ID)
		return projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Pane: pane, Binding: binding, Kind: contentKindForPaneRoot(root, pane), Surface: surface, Session: session, Active: true})
	}
	if len(shell.Workspace.Tabs) == 0 {
		return buildEmptyWorkspaceContent(shell.Workspace)
	}
	if tab := activeTab(shell); len(tab.Panes) == 0 {
		return buildEmptyTabContent(tab)
	}
	return projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentPlaceholder, Active: true})
}

func (projector ShellProjector) contentForPane(root state.Root, pane state.PaneState, activeContent ContentVM, active bool) ContentVM {
	if copyModeBelongsToPane(root, pane.ID) {
		return projector.copyHistoryContentForPane(root, root.Shell.ReadonlyDefaults(), pane, active)
	}
	if active {
		return activeContent
	}
	surface, session := terminalContentStoresForPane(root, pane)
	binding, _ := root.TerminalViews.PaneBinding(pane.ID)
	return projector.Content.Project(ContentProjectorContext{Root: root, Shell: root.Shell.ReadonlyDefaults(), Pane: pane, Binding: binding, Kind: contentKindForPaneRoot(root, pane), Surface: surface, Session: session, Active: false})
}

func (projector ShellProjector) copyHistoryContentForPane(root state.Root, shell state.ShellStore, pane state.PaneState, active bool) ContentVM {
	return projector.copyHistoryContentForView(root, shell, copyHistoryViewIDForPane(root, pane.ID), pane, active)
}

func (projector ShellProjector) copyHistoryContentForView(root state.Root, shell state.ShellStore, viewID string, pane state.PaneState, active bool) ContentVM {
	history, copyMode := root.CopyHistorySessionForView(viewID)
	if historyStoreBelongsToView(root.History, viewID) || copyModeStoreBelongsToView(root.CopyMode, viewID) {
		history = root.History
		copyMode = root.CopyMode
	} else if root.CopyMode.Active && root.CopyMode.ViewID == "" && root.CopyMode.PaneID == "" {
		history = root.History
		copyMode = root.CopyMode
	}
	return projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Pane: pane, Kind: ContentCopyHistory, Active: active, History: history, CopyMode: copyMode})
}

func (projector ShellProjector) contentForFloating(root state.Root, shell state.ShellStore, floating state.FloatingPaneState) ContentVM {
	if copyModeBelongsToFloating(root, floating.ID) {
		return projector.copyHistoryContentForView(root, shell, copyHistoryViewIDForFloating(root, floating.ID), floating.Pane, floating.Active)
	}
	if floating.Active && floating.Pane.Kind == state.PaneEmpty {
		return buildEmptyPaneContentWithSelection(floating.Pane, shell.EmptyPaneCTA.SelectedIndex)
	}
	return projector.Content.Project(ContentProjectorContext{
		Root:    root,
		Shell:   shell,
		Pane:    floating.Pane,
		Binding: terminalBindingForFloating(root, floating.ID),
		Kind:    contentKindForFloating(root, floating),
		Active:  floating.Active,
		Surface: surfaceForFloating(root, floating.ID),
		Session: sessionForFloating(root, floating.ID),
	})
}

func terminalBindingForFloating(root state.Root, floatingID string) state.TerminalViewBinding {
	binding, _ := root.TerminalViews.FloatingBinding(floatingID)
	return binding
}

func copyModeBelongsToPane(root state.Root, paneID string) bool {
	viewID := copyHistoryViewIDForPane(root, paneID)
	copyMode := copyModeForView(root, viewID)
	if !copyMode.HistoryRenderable() || paneID == "" {
		return false
	}
	if copyMode.PaneID == "" {
		return paneID == root.Shell.ReadonlyDefaults().ActivePaneID
	}
	return copyMode.PaneID == paneID
}

func copyModeBelongsToFloating(root state.Root, floatingID string) bool {
	viewID := copyHistoryViewIDForFloating(root, floatingID)
	copyMode := copyModeForView(root, viewID)
	return copyMode.HistoryRenderable() && floatingID != "" && copyMode.ViewID == viewID
}

func copyModeForView(root state.Root, viewID string) state.CopyModeStore {
	if copyModeStoreBelongsToView(root.CopyMode, viewID) {
		return root.CopyMode
	}
	if root.CopyMode.HistoryRenderable() && root.CopyMode.ViewID == "" && root.CopyMode.PaneID == "" {
		return root.CopyMode
	}
	_, copyMode := root.CopyHistorySessionForView(viewID)
	return copyMode
}

func copyModeStoreBelongsToView(copyMode state.CopyModeStore, viewID string) bool {
	if viewID == "" {
		return false
	}
	if copyMode.ViewID == viewID {
		return true
	}
	return copyMode.ViewID == "" && copyMode.PaneID != "" && state.TerminalPaneViewID(copyMode.PaneID) == viewID
}

func historyStoreBelongsToView(history state.HistoryStore, viewID string) bool {
	if viewID == "" {
		return false
	}
	if history.ViewID == viewID {
		return true
	}
	return history.ViewID == "" && history.PaneID != "" && state.TerminalPaneViewID(history.PaneID) == viewID
}

func copyHistoryViewIDForPane(root state.Root, paneID string) string {
	// 中文说明：真实 CLI / restore 里的 TerminalView ID 不一定等于 pane:<paneID>。
	// copy/history 会话必须按绑定的 ViewID 查找，否则最新/oldest 已写回但 pane 仍渲染旧会话。
	return root.TerminalViews.PaneViewID(paneID)
}

func copyHistoryViewIDForFloating(root state.Root, floatingID string) string {
	// 中文说明：floating 也可能拥有非派生 ViewID；必须和 pane 一样以 TerminalViewStore 为准。
	return root.TerminalViews.FloatingViewID(floatingID)
}

func contentKindForPane(pane state.PaneState) ContentKind {
	switch pane.Kind {
	case state.PaneEmpty:
		return ContentEmptyPane
	case state.PaneTerminalLive:
		return ContentTerminalLive
	default:
		return ContentPlaceholder
	}
}

func contentKindForPaneRoot(root state.Root, pane state.PaneState) ContentKind {
	if pane.Kind == state.PaneEmpty && paneHasTerminalBinding(root, pane.ID) {
		return ContentTerminalLive
	}
	return contentKindForPane(pane)
}

func contentKindForFloating(root state.Root, floating state.FloatingPaneState) ContentKind {
	if floating.Pane.Kind == state.PaneEmpty {
		if binding, ok := root.TerminalViews.FloatingBinding(floating.ID); ok && binding.TerminalID != "" {
			return ContentTerminalLive
		}
	}
	return contentKindForPane(floating.Pane)
}

func paneHasTerminalBinding(root state.Root, paneID string) bool {
	binding, ok := root.TerminalViews.PaneBinding(paneID)
	return ok && binding.TerminalID != ""
}

func terminalContentStoresForPane(root state.Root, pane state.PaneState) (state.TerminalSurfaceStore, state.TerminalSessionStore) {
	binding, ok := root.TerminalViews.PaneBinding(pane.ID)
	if !ok || binding.TerminalID == "" {
		return state.TerminalSurfaceStore{}, state.TerminalSessionStore{}
	}
	return surfaceForBinding(root, binding), sessionForBinding(root, binding)
}

func surfaceForFloating(root state.Root, floatingID string) state.TerminalSurfaceStore {
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	if !ok || binding.TerminalID == "" {
		return state.TerminalSurfaceStore{}
	}
	return surfaceForBinding(root, binding)
}

func sessionForFloating(root state.Root, floatingID string) state.TerminalSessionStore {
	binding, ok := root.TerminalViews.FloatingBinding(floatingID)
	if !ok || binding.TerminalID == "" {
		return state.TerminalSessionStore{}
	}
	return sessionForBinding(root, binding)
}

func surfaceForBinding(root state.Root, binding state.TerminalViewBinding) state.TerminalSurfaceStore {
	if binding.TerminalID == "" {
		return state.TerminalSurfaceStore{}
	}
	return root.Surface.SurfaceForTerminalRef(binding.TerminalRef())
}

func sessionForBinding(root state.Root, binding state.TerminalViewBinding) state.TerminalSessionStore {
	if binding.TerminalID == "" {
		return state.TerminalSessionStore{}
	}
	ref := binding.TerminalRef()
	surface := root.Surface.SurfaceForTerminalRef(ref)
	cols, rows := binding.DesiredCols, binding.DesiredRows
	if cols <= 0 {
		cols = surface.Cols
	}
	if rows <= 0 {
		rows = surface.Rows
	}
	session := state.TerminalSessionStore{
		EndpointID:   ref.EndpointID,
		TerminalID:   binding.TerminalID,
		Channel:      binding.Channel,
		Attached:     binding.Attached,
		Cols:         cols,
		Rows:         rows,
		ResizePolicy: binding.ResizeRole,
		SurfaceID:    binding.SurfaceID,
		ViewID:       binding.ViewID,
		DesiredCols:  binding.DesiredCols,
		DesiredRows:  binding.DesiredRows,
		LastError:    binding.LastError,
		State:        surface.State,
		ExitCode:     surface.ExitCode,
		ExitReason:   surface.ExitReason,
		ExitedAt:     surface.ExitedAt,
		Command:      append([]string(nil), surface.Command...),
	}
	session = mergeSessionExitedLifecycle(session, root.Session)
	return session
}

func mergeSessionExitedLifecycle(session state.TerminalSessionStore, lifecycle state.TerminalSessionStore) state.TerminalSessionStore {
	if session.TerminalID == "" || lifecycle.TerminalID != session.TerminalID || lifecycle.State != state.TerminalLiveExited {
		return session
	}
	// TerminalSessionStore 是 input/live reducer 回投的 lifecycle truth；同一 terminal 的退出态必须
	// 覆盖仍 attached 的旧 surface 投影，否则渲染、hit region 和键盘 CTA 会看到不同状态。
	session.Attached = false
	session.State = state.TerminalLiveExited
	session.ExitCode = lifecycle.ExitCode
	session.ExitReason = lifecycle.ExitReason
	session.ExitedAt = lifecycle.ExitedAt
	session.Command = append([]string(nil), lifecycle.Command...)
	session.LastError = ""
	return session
}

func canRenderCopyHistory(root state.Root, history state.HistoryStore, copyMode state.CopyModeStore) bool {
	phase := copyMode.PhaseKind()
	tokenMatches := false
	switch phase {
	case state.CopyModeFrozenHistory:
		tokenMatches = copyMode.BoundToken != "" && copyMode.BoundToken == history.Token
	default:
		return false
	}
	return copyMode.HistoryRenderable() &&
		copyMode.TerminalID != "" &&
		history.TerminalID != "" &&
		copyMode.BoundCols != 0 &&
		history.Cols != 0 &&
		copyModeBindingStillValid(root, copyMode) &&
		tokenMatches &&
		copyMode.BoundCols == history.Cols &&
		copyMode.TerminalID == history.TerminalID &&
		len(history.Rows) > 0
}

func copyHistoryPendingReason(root state.Root, history state.HistoryStore, copyMode state.CopyModeStore) string {
	switch {
	case !copyMode.HistoryRenderable():
		return ""
	case copyMode.TerminalID == "":
		return "copy history pending: terminal binding missing"
	case !copyModeBindingStillValid(root, copyMode):
		return "copy history pending: copy binding missing"
	case copyMode.PhaseKind() == state.CopyModeFrozenHistory && copyMode.BoundToken == "":
		return "copy history pending: window pending"
	case copyMode.BoundCols == 0:
		return "copy history pending: bound cols missing"
	case history.TerminalID == "":
		return "copy history pending: window pending"
	case copyMode.TerminalID != history.TerminalID:
		return "copy history error: terminal mismatch"
	case copyMode.PhaseKind() == state.CopyModeFrozenHistory && copyMode.BoundToken != history.Token:
		return "copy history pending: stale history token"
	case history.Cols == 0:
		return "copy history pending: window pending"
	case copyMode.BoundCols != history.Cols:
		return "copy history pending: history cols changed"
	case len(history.Rows) == 0:
		return "copy history empty"
	default:
		return ""
	}
}

func buildCopyHistoryContentVM(root state.Root, history state.HistoryStore, copyMode state.CopyModeStore) ContentVM {
	cursor := copyMode.Cursor
	copyMode = copyMode.RefreshSearchMatches(history)
	copyMode.Cursor = cursor
	extent := copyHistoryContentExtent(root, history, copyMode)
	search := copyHistorySearchFooter(copyMode)
	if !canRenderCopyHistory(root, history, copyMode) {
		reason := copyHistoryPendingReason(root, history, copyMode)
		content := ContentVM{
			Kind:    ContentCopyHistory,
			Lines:   []Line{NewLine(reason)},
			Search:  search,
			Status:  copyHistoryStatus(history, copyMode),
			Pending: true,
			Extent:  extent,
		}
		if reason == "copy history empty" {
			content.Pending = false
			content.Empty = true
		}
		if reason == "copy history error: terminal mismatch" {
			content.Pending = false
			content.Error = reason
		}
		return content
	}
	content := ContentVM{
		Kind:   ContentCopyHistory,
		Lines:  copyHistoryLines(history, copyMode),
		Search: search,
		Status: copyHistoryStatus(history, copyMode),
		Extent: extent,
	}
	if copyMode.CanSelect() {
		content.Cursor = copyHistoryCursor(history, copyMode)
		content.HitRegions = copyHistoryHitRegions(history, copyMode)
	}
	return content
}

func copyHistoryContentExtent(root state.Root, history state.HistoryStore, copyMode state.CopyModeStore) ContentExtent {
	cols := copyMode.BoundCols
	if cols <= 0 {
		cols = history.Cols
	}
	rows := copyMode.ViewRows
	if binding, ok := copyHistoryTerminalBinding(root, copyMode); ok {
		surface := surfaceForBinding(root, binding)
		session := sessionForBinding(root, binding)
		if terminalCols, terminalRows := liveStatusSize(surface, session); terminalCols > 0 && terminalRows > 0 {
			cols = terminalCols
			rows = terminalRows
		}
	}
	if cols <= 0 || rows <= 0 {
		return ContentExtent{}
	}
	return ContentExtent{Known: true, Cols: cols, Rows: rows}
}

func copyHistoryTerminalBinding(root state.Root, copyMode state.CopyModeStore) (state.TerminalViewBinding, bool) {
	if copyMode.ViewID != "" {
		binding, ok := root.TerminalViews.Views[copyMode.ViewID]
		return binding, ok && binding.TerminalID == copyMode.TerminalID
	}
	if copyMode.PaneID != "" {
		binding, ok := root.TerminalViews.PaneBinding(copyMode.PaneID)
		return binding, ok && binding.TerminalID == copyMode.TerminalID
	}
	return state.TerminalViewBinding{}, false
}

func copyModeBindingStillValid(root state.Root, copyMode state.CopyModeStore) bool {
	if !copyMode.HistoryRenderable() {
		return false
	}
	if copyMode.ViewID != "" {
		if binding, ok := root.TerminalViews.Views[copyMode.ViewID]; ok {
			return binding.TerminalID == copyMode.TerminalID
		}
		return false
	}
	if copyMode.PaneID != "" {
		if _, ok := root.Shell.ReadonlyDefaults().Pane(state.PaneCommandTarget{PaneID: copyMode.PaneID}); ok {
			return true
		}
		return false
	}
	return true
}

func (projector ShellProjector) buildOverlayVM(root state.Root, shell state.ShellStore) OverlayVM {
	if !shell.Overlay.Open {
		return OverlayVM{}
	}
	var overlay OverlayVM
	switch shell.Overlay.Kind {
	case state.OverlayTerminalPicker:
		title := "Terminal Picker"
		if shell.Overlay.TerminalPickerView == state.TerminalPickerViewEndpoints {
			title = "Terminal Picker / Endpoints"
		} else if shell.Overlay.TerminalPickerView == state.TerminalPickerViewTags {
			title = "Terminal Picker / Tags"
		}
		overlay = OverlayVM{
			Kind:   OverlayTerminalPicker,
			Opaque: false,
			Title:  title,
			Picker: PickerChromeVM{
				Presentation: root.Config.Chrome.Picker.Presentation,
				Width:        root.Config.Chrome.Picker.Width,
				Density:      root.Config.Chrome.Picker.Density,
				EndpointTabs: root.Config.Chrome.Picker.EndpointTabs,
			},
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentTerminalPicker}),
		}
	case state.OverlayTerminalPool:
		overlay = OverlayVM{
			Kind:    OverlayTerminalPool,
			Opaque:  true,
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentTerminalPool}),
		}
	case state.OverlayConnections:
		overlay = OverlayVM{
			Kind:    OverlayConnections,
			Opaque:  true,
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentConnections}),
		}
	case state.OverlayWorkbenchTree:
		overlay = OverlayVM{
			Kind:    OverlayWorkbenchTree,
			Opaque:  true,
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentWorkbenchTree}),
		}
	case state.OverlayClipboardHistory:
		overlay = OverlayVM{
			Kind:    OverlayClipboardHistory,
			Opaque:  true,
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentClipboardHistory}),
		}
	case state.OverlayFloatingOverview:
		overlay = OverlayVM{
			Kind:    OverlayFloatingOverview,
			Opaque:  true,
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentFloatingOverview}),
		}
	case state.OverlayPrompt:
		overlay = OverlayVM{
			Kind:    OverlayPrompt,
			Opaque:  true,
			Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentPrompt}),
			Popup:   buildPromptSuggestionPopupVM(shell.Overlay.Prompt),
		}
	case state.OverlayHelp:
		overlay = OverlayVM{Kind: OverlayHelp, Opaque: true, Content: projector.Content.Project(ContentProjectorContext{Root: root, Shell: shell, Kind: ContentHelp})}
	default:
		return OverlayVM{}
	}
	if shell.Overlay.Kind == state.OverlayTerminalPicker && shell.Overlay.TerminalPickerView == state.TerminalPickerViewTags {
		overlay.Content.HitRegions = bindOverlayShortcutInvocationsForScene(string(shortcut.SceneTerminalPickerTags), overlay.Content.HitRegions, root.Config.Shortcuts)
	} else {
		overlay.Content.HitRegions = bindOverlayShortcutInvocations(overlay.Kind, overlay.Content.HitRegions, root.Config.Shortcuts)
	}
	return overlay
}

func buildToastVMs(shell state.ShellStore) []ToastVM {
	if len(shell.Toasts) == 0 {
		return nil
	}
	toasts := make([]ToastVM, len(shell.Toasts))
	for i, toast := range shell.Toasts {
		toasts[i] = ToastVM{
			ID:       toast.ID,
			Severity: renderToastSeverity(toast.Severity),
			Title:    toast.Title,
			Body:     toast.Body,
			Pending:  toast.Pending,
		}
	}
	return toasts
}

func activeContent(shell ShellVM) ContentVM {
	for _, panel := range shell.Layout.Panels {
		if panel.Active {
			return panel.Content
		}
	}
	if len(shell.Layout.Panels) > 0 {
		return shell.Layout.Panels[0].Content
	}
	return ContentVM{}
}

func activeTab(shell state.ShellStore) state.TabState {
	shell = shell.ReadonlyDefaults()
	for _, tab := range shell.Workspace.Tabs {
		if tab.ID == shell.Workspace.ActiveTabID {
			return tab
		}
	}
	if len(shell.Workspace.Tabs) > 0 {
		return shell.Workspace.Tabs[0]
	}
	return state.TabState{}
}

func buildSplitVM(node state.SplitNode) SplitVM {
	out := SplitVM{
		PaneID:      node.PaneID,
		Direction:   renderSplitDirection(node.Direction),
		Ratio:       node.Ratio,
		BiasCells:   node.BiasCells,
		FixedPaneID: node.FixedPaneID,
		FixedCols:   node.FixedCols,
		FixedRows:   node.FixedRows,
	}
	if len(node.Children) > 0 {
		out.Children = make([]SplitVM, len(node.Children))
		for i, child := range node.Children {
			out.Children[i] = buildSplitVM(child)
		}
	}
	return out
}

func renderSplitDirection(direction state.SplitDirection) SplitDirection {
	switch direction {
	case state.SplitDirectionVertical:
		return SplitVertical
	case state.SplitDirectionHorizontal:
		return SplitHorizontal
	default:
		return ""
	}
}

func placeholderContentForPane(pane state.PaneState) ContentVM {
	title := activePaneTitle(pane)
	switch pane.Kind {
	case state.PaneEmpty:
		return buildEmptyPaneContent(pane)
	default:
		return ContentVM{Kind: ContentPlaceholder, Lines: []Line{NewLine(title + " inactive")}, Pending: true}
	}
}

func activePaneTitle(pane state.PaneState) string {
	if pane.Title != "" {
		return pane.Title
	}
	if pane.TerminalID != "" {
		return pane.TerminalID
	}
	if pane.ID != "" {
		return pane.ID
	}
	return "pane"
}

func renderPanelPresentation(presentation state.PanelPresentation) PanelPresentation {
	switch presentation {
	case state.PanelPresentationSplitLine:
		return PanelPresentationSplitLine
	default:
		return PanelPresentationCard
	}
}

func renderToastSeverity(severity state.ToastSeverity) ToastSeverity {
	switch severity {
	case state.ToastSuccess:
		return ToastSuccess
	case state.ToastWarning:
		return ToastWarning
	case state.ToastError:
		return ToastError
	default:
		return ToastInfo
	}
}

func cloneHitRegions(regions []HitRegion) []HitRegion {
	if len(regions) == 0 {
		return nil
	}
	cloned := make([]HitRegion, len(regions))
	copy(cloned, regions)
	return cloned
}
