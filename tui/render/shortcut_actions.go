package render

import (
	"strconv"
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

// canonicalActionForProjection 把仍在生产的视觉投影关联到唯一 canonical action。
// ProjectionID 只定位 glyph 与几何；执行身份始终由 tui/action 拥有。
func canonicalActionForProjection(id ProjectionID) actiondomain.ID {
	explicit := map[ProjectionID]actiondomain.ID{
		ActionPaneFocus: actiondomain.ActionPanelFocus, ActionPaneResize: actiondomain.ActionPanelResizeDrag,
		ActionPaneSplitDown: "panel.split_down", ActionPaneSplitRight: "panel.split_right",
		ActionPaneZoom: "panel.toggle_zoom", ActionPaneClose: "panel.close",
		ActionTerminalTakeResizeOwner: "panel.take_owner", ActionResizeLayoutLock: "panel.size_lock",
		ActionTabCreate: "tab.create", ActionTabSwitch: actiondomain.ActionTabSelect, ActionTabClose: "tab.close",
		ActionFloatingRaise: actiondomain.ActionFloatingRaise, ActionFloatingSummon: "floating_overview.open",
		ActionFloatingMoveDrag:   actiondomain.ActionFloatingMoveDrag,
		ActionFloatingResizeDrag: actiondomain.ActionFloatingResizeDrag, ActionFloatingClose: "floating.close",
		ActionFloatingCenter: "floating.center", ActionFloatingCollapse: "floating.collapse",
		ActionEmptyAttach: actiondomain.ActionEmptyAttach, ActionEmptyCreate: actiondomain.ActionEmptyCreate,
		ActionEmptyManager: actiondomain.ActionEmptyManager, ActionEmptyClose: actiondomain.ActionEmptyClose,
		ActionExitedRestart: actiondomain.ActionExitedRestart, ActionExitedReconnect: actiondomain.ActionExitedReconnect,
		ActionExitedClose: actiondomain.ActionExitedClose, ActionDisconnectedReconnect: actiondomain.ActionDisconnectedReconnect,
		ActionDisconnectedDisconnect: actiondomain.ActionDisconnectedDisconnect,
		ActionPickerAttach:           "terminal_picker.attach", ActionPickerNew: actiondomain.ActionTerminalPickerNew,
		ActionPoolSelect: actiondomain.ActionTerminalPoolSelect, ActionWorkbenchOpen: "workbench_tree.open",
		ActionClipboardHistorySelect:      actiondomain.ActionClipboardHistorySelect,
		ActionClipboardHistoryDividerDrag: actiondomain.ActionClipboardHistoryDividerDrag,
		ActionHelpClose:                   "help.close",
	}
	return explicit[id]
}

// projectionActionLabel 只读取 action domain 的默认语义文案。
// render projection 只拥有 glyph 和布局，不能重新保存快捷键或帮助文案。
func projectionActionLabel(spec ProjectionSpec) string {
	actionSpec, ok := actiondomain.SpecByID(spec.CanonicalActionID)
	if !ok {
		return ""
	}
	return actionSpec.DefaultLabel
}

// invocationForProjection 把 render-local 投影转换为唯一 canonical invocation。
func invocationForProjection(id ProjectionID) actiondomain.Invocation {
	spec, ok := ProjectionByID(id)
	if !ok || spec.CanonicalActionID == "" {
		return actiondomain.Invocation{}
	}
	return actiondomain.Invocation{ID: spec.CanonicalActionID, SourceActionID: spec.CanonicalActionID.String()}
}

func invocationForHeaderActionID(id string) actiondomain.Invocation {
	canonicalID := actiondomain.ID(id)
	if _, ok := actiondomain.SpecByID(canonicalID); ok {
		return actiondomain.Invocation{ID: canonicalID, SourceActionID: id}
	}
	return invocationForProjection(ProjectionID(id))
}

func shortcutSceneForFooterMode(mode string) string {
	switch mode {
	case "live", "normal", "":
		return string(shortcut.SceneGlobal)
	case "global":
		return string(shortcut.SceneSystem)
	case "pane":
		return string(shortcut.ScenePanel)
	default:
		if scene, ok := shortcut.SceneByName(mode); ok {
			return string(scene.ID)
		}
		return string(shortcut.SceneGlobal)
	}
}

// bindOverlayShortcutInvocations 把 overlay content 的可配置点击动作绑定到当前 catalog。
// invocation 是“执行什么”的真值，HitRegion 上的 row/pane 仍只是点击目标上下文；catalog 未配置的
// shortcut action 不保留旧 ActionID 点击入口，行选择和拖拽等非 shortcut 交互不受影响。
func bindOverlayShortcutInvocations(kind OverlayKind, regions []HitRegion, shortcuts state.TUIShortcutConfig) []HitRegion {
	scene := shortcutSceneForFooterMode(string(kind))
	entries := input.ShortcutEntriesForScene(shortcuts, scene)
	invocations := map[actiondomain.ID][]actiondomain.Invocation{}
	for _, entry := range entries {
		invocation, _, err := actiondomain.ParseInvocation(entry.ActionID)
		if err != nil {
			continue
		}
		invocations[invocation.ID] = appendUniqueInvocation(invocations[invocation.ID], invocation)
	}
	out := make([]HitRegion, 0, len(regions))
	for _, region := range regions {
		if region.Invocation.ID == "" {
			if region.ActionID != "" {
				continue
			}
			out = append(out, region)
			continue
		}
		if _, shortcutAction := shortcut.Policies()[region.Invocation.ID]; !shortcutAction {
			out = append(out, region)
			continue
		}
		candidates := invocations[region.Invocation.ID]
		if len(candidates) != 1 {
			continue
		}
		region.Invocation = candidates[0]
		out = append(out, region)
	}
	return out
}

func appendUniqueInvocation(items []actiondomain.Invocation, invocation actiondomain.Invocation) []actiondomain.Invocation {
	signature := invocation.Signature()
	for _, item := range items {
		if item.Signature() == signature {
			return items
		}
	}
	return append(items, invocation)
}

func footerActionCatalogFromShortcuts(mode string, root state.Root) []FooterActionVM {
	return shortcutActionCatalogFromShortcuts(mode, root, false)
}

func helpActionCatalogFromShortcuts(mode string, root state.Root) []FooterActionVM {
	return shortcutActionCatalogFromShortcuts(mode, root, true)
}

func shortcutActionCatalogFromShortcuts(mode string, root state.Root, includeHidden bool) []FooterActionVM {
	scene := shortcutSceneForFooterMode(mode)
	entries := input.ShortcutEntriesForScene(root.Config.Shortcuts, scene)
	out := make([]FooterActionVM, 0, len(entries))
	for _, entry := range entries {
		if input.ShortcutKeyRequiresEnhancedKeyboard(entry.Key) && !root.HostCapabilities.KeyboardDisambiguation {
			continue
		}
		action, ok := shortcutActionFromShortcutEntry(entry, root.Config, includeHidden)
		if ok {
			out = append(out, action)
		}
	}
	out = compactShortcutFooterActions(scene, out)
	return out
}

func shortcutActionFromShortcutEntry(entry input.ShortcutEntry, cfg state.TUIConfigStore, forHelp bool) (FooterActionVM, bool) {
	policy, invocation, _, ok := shortcut.PolicyForSource(entry.ActionID)
	if !ok {
		return FooterActionVM{}, false
	}
	visible := policy.Footer == shortcut.VisibilityVisible
	if entry.Show != nil {
		visible = *entry.Show
	}
	if forHelp {
		visible = policy.Help == shortcut.VisibilityVisible
	}
	if !visible {
		return FooterActionVM{}, false
	}
	action := FooterActionVM{
		ActionID:   invocation.ID.String(),
		Style:      shortcutActionStyle(shortcutCatalogScene(entry.Scene), invocation.ID),
		Invocation: invocation,
		Click:      ClickClickable,
	}
	action.Key = entry.KeyLabel
	if action.Key == "" {
		action.Key = input.ShortcutKeyDisplay(entry.Key)
	}
	if label := shortcutActionLabel(entry, cfg, action); label != "" {
		action.Label = label
	}
	if style := configuredShortcutActionStyle(cfg.Shortcuts.Actions, entry.ActionID); style != "" {
		action.Style = footerStyleTokenFromConfig(style)
	}
	return action, true
}

func shortcutActionStyle(scene string, id actiondomain.ID) StyleToken {
	if scene == string(shortcut.SceneGlobal) {
		switch id {
		case "menu.panel":
			return StyleFooterKeyPane
		case "menu.resize":
			return StyleFooterKeyResize
		case "menu.tab":
			return StyleFooterKeyTab
		case "menu.workspace":
			return StyleFooterKeyWorkspace
		case "menu.floating":
			return StyleFooterKeyFloat
		case "menu.copy":
			return StyleFooterKeyCopy
		case "menu.terminal_picker":
			return StyleFooterKeyPicker
		case "menu.system":
			return StyleFooterKeyGlobal
		}
	}
	value := id.String()
	if strings.Contains(value, "kill") || strings.Contains(value, "delete") || strings.Contains(value, "remove") ||
		strings.HasSuffix(value, ".close") || value == "system.quit" {
		return StyleStatusWarning
	}
	return StyleStatusAccent
}

func shortcutCatalogScene(scene string) string {
	return string(shortcut.NormalizeScene(scene))
}

func shortcutActionLabel(entry input.ShortcutEntry, cfg state.TUIConfigStore, action FooterActionVM) string {
	if label := strings.TrimSpace(entry.Label); label != "" {
		return label
	}
	if label := configuredShortcutActionLabel(cfg.Shortcuts.Actions, entry.ActionID); label != "" {
		return label
	}
	_, spec, err := actiondomain.ParseInvocation(entry.ActionID)
	if err == nil {
		return strings.TrimSpace(spec.DefaultLabel)
	}
	return ""
}

func configuredShortcutActionLabel(actions map[string]state.TUIShortcutActionConfig, actionID string) string {
	_, target, err := actiondomain.ParseInvocation(actionID)
	if err != nil {
		return ""
	}
	for configuredID, action := range actions {
		_, configured, err := actiondomain.ParseInvocation(configuredID)
		if err == nil && configured.ID == target.ID {
			return strings.TrimSpace(action.Label)
		}
	}
	return ""
}

func configuredShortcutActionStyle(actions map[string]state.TUIShortcutActionConfig, actionID string) string {
	_, target, err := actiondomain.ParseInvocation(actionID)
	if err != nil {
		return ""
	}
	for configuredID, action := range actions {
		_, configured, err := actiondomain.ParseInvocation(configuredID)
		if err == nil && configured.ID == target.ID {
			return strings.TrimSpace(action.Style)
		}
	}
	return ""
}

func compactShortcutFooterActions(scene string, actions []FooterActionVM) []FooterActionVM {
	scene = shortcutCatalogScene(scene)
	if scene == "global" || scene == "system" {
		actions = compactFooterActionGroup(actions, "menu.terminal_pool")
		actions = orderShortcutFooterActions(actions, []actiondomain.ID{
			"menu.connections", "system.toggle_header", "system.toggle_footer", "menu.terminal_pool", "menu.help",
			"menu.prompt", "menu.workbench_tree", "system.toggle_shortcut_lock", "system.clear_toasts", "system.close_toast", "system.quit",
		})
	}
	if scene == "panel" {
		actions = compactFooterActionGroup(actions, "panel.close")
		actions = compactFooterActionGroup(actions, "panel.split_right")
		actions = compactFooterActionGroup(actions, "panel.split_down")
		actions = compactFooterActionGroup(actions, "panel.focus_next", "panel.focus_prev")
	}
	if scene == "resize" {
		actions = compactFooterActionGroup(actions, "resize.left", "resize.left_large")
		actions = compactFooterActionGroup(actions, "resize.right", "resize.right_large")
		actions = compactFooterActionGroup(actions, "resize.up", "resize.up_large")
		actions = compactFooterActionGroup(actions, "resize.down", "resize.down_large")
		actions = compactFooterActionGroup(actions, "panel.balance")
		actions = compactFooterActionGroup(actions, "resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down")
		actions = compactFooterActionGroup(actions, "resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom")
		actions = compactFooterActionGroup(actions, "resize.center", "resize.center_x", "resize.center_y")
	}
	if scene == "tab" {
		actions = compactFooterActionGroup(actions, "tab.jump")
		actions = compactFooterActionGroup(actions, "tab.next")
		actions = compactFooterActionGroup(actions, "tab.previous")
	}
	if scene == "workspace" {
		actions = compactFooterActionGroup(actions, "workspace.next")
		actions = compactFooterActionGroup(actions, "workspace.previous")
		actions = compactFooterActionGroup(actions, "menu.workbench_tree")
	}
	if scene == "floating" || scene == "floating_overview" {
		actions = compactFloatingSummonActions(actions)
		actions = compactFooterActionGroup(actions, "floating.collapse")
		actions = compactFooterActionGroup(actions, "floating.move_left")
		actions = compactFooterActionGroup(actions, "floating.move_right")
		actions = compactFooterActionGroup(actions, "floating.move_up")
		actions = compactFooterActionGroup(actions, "floating.move_down")
	}
	return actions
}

func compactFloatingSummonActions(actions []FooterActionVM) []FooterActionVM {
	actionID := actiondomain.ID("floating.summon").String()
	indexes := []int{}
	keys := []string{}
	var base FooterActionVM
	for index, action := range actions {
		if action.ActionID != actionID {
			continue
		}
		if _, err := strconv.Atoi(action.Key); err != nil {
			continue
		}
		if len(indexes) == 0 {
			base = action
		}
		indexes = append(indexes, index)
		keys = append(keys, action.Key)
	}
	if len(indexes) <= 1 {
		return actions
	}
	if collapsed := collapsedKeyRange(keys); collapsed != "" {
		base.Key = collapsed
	} else {
		base.Key = strings.Join(keys, "/")
	}
	base.Click = ClickHintOnly
	base.Invocation = actiondomain.Invocation{}
	return replaceFooterActionIndexes(actions, indexes, base)
}

func orderShortcutFooterActions(actions []FooterActionVM, order []actiondomain.ID) []FooterActionVM {
	if len(actions) <= 1 {
		return actions
	}
	remaining := append([]FooterActionVM(nil), actions...)
	out := make([]FooterActionVM, 0, len(actions))
	for _, id := range order {
		actionID := id.String()
		for index := 0; index < len(remaining); index++ {
			if remaining[index].ActionID != actionID {
				continue
			}
			out = append(out, remaining[index])
			remaining = append(remaining[:index], remaining[index+1:]...)
			index--
		}
	}
	out = append(out, remaining...)
	return out
}

func compactFooterActionGroup(actions []FooterActionVM, ids ...actiondomain.ID) []FooterActionVM {
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id.String()] = struct{}{}
	}
	indexes := []int{}
	keys := []string{}
	var base FooterActionVM
	for index, action := range actions {
		if _, ok := wanted[action.ActionID]; !ok {
			continue
		}
		if len(indexes) == 0 {
			base = action
		}
		indexes = append(indexes, index)
		keys = append(keys, action.Key)
	}
	if len(indexes) <= 1 {
		return actions
	}
	if collapsed := collapsedKeyRange(keys); collapsed != "" {
		base.Key = collapsed
	} else {
		base.Key = strings.Join(keys, "/")
	}
	for _, index := range indexes[1:] {
		if actions[index].Invocation.Signature() != base.Invocation.Signature() {
			base.Click = ClickHintOnly
			base.Invocation = actiondomain.Invocation{}
			break
		}
	}
	return replaceFooterActionIndexes(actions, indexes, base)
}

func replaceFooterActionIndexes(actions []FooterActionVM, indexes []int, base FooterActionVM) []FooterActionVM {
	out := make([]FooterActionVM, 0, len(actions)-len(indexes)+1)
	inserted := false
	skip := map[int]struct{}{}
	for _, index := range indexes {
		skip[index] = struct{}{}
	}
	for index, action := range actions {
		if _, ok := skip[index]; ok {
			if !inserted {
				out = append(out, base)
				inserted = true
			}
			continue
		}
		out = append(out, action)
	}
	return out
}

func collapsedKeyRange(keys []string) string {
	if len(keys) < 3 {
		return ""
	}
	for index, key := range keys {
		if key != strconv.Itoa(index+1) {
			return ""
		}
	}
	return keys[0] + "-" + keys[len(keys)-1]
}
