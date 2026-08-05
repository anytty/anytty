// Package shortcut owns bindable scenes and scene+key presentation policy.
// Canonical action identity, parameters, invocation, and semantic labels belong to tui/action.
package shortcut

import (
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
)

// Visibility 描述一个 shortcut binding 是否进入指定快捷键提示目录。
// 它只控制 footer/help 的快捷键投影，不代表 action 全局可见性或 clickability。
type Visibility string

const (
	VisibilityHidden  Visibility = "hidden"
	VisibilityVisible Visibility = "visible"
)

// BindingPolicy 是 canonical action 在 shortcut catalog 中的可绑定约束。
// scene、默认 footer/help 展示与是否进入 root/sticky router 都属于 shortcut；业务可用性和点击属性不在此声明。
type BindingPolicy struct {
	AllowedScenes []SceneID
	Footer        Visibility
	Help          Visibility
	Routable      bool
}

// SceneID 是 shortcut domain 拥有的内置场景身份。
// 配置、input 和 render 只能通过本类型及 NormalizeScene 引用场景，不能各自维护有效场景集合。
type SceneID string

const (
	SceneGlobal           SceneID = "global"
	SceneSystem           SceneID = "system"
	ScenePanel            SceneID = "panel"
	SceneFloating         SceneID = "floating"
	SceneTab              SceneID = "tab"
	SceneWorkspace        SceneID = "workspace"
	SceneResize           SceneID = "resize"
	SceneCopy             SceneID = "copy"
	SceneTerminalPicker   SceneID = "terminal_picker"
	SceneTerminalPool     SceneID = "terminal_pool"
	SceneConnections      SceneID = "connections"
	SceneWorkbenchTree    SceneID = "workbench_tree"
	SceneClipboardHistory SceneID = "clipboard_history"
	SceneFloatingOverview SceneID = "floating_overview"
	ScenePrompt           SceneID = "prompt"
	SceneHelp             SceneID = "help"
)

// SceneSpec 描述一个内置 shortcut scene 及进入它的 canonical menu action。
// MenuAction 为空表示 root/global 场景不能由 menu action 进入；Routable 表示 input 主路由可直接消费。
type SceneSpec struct {
	ID         SceneID
	MenuAction actiondomain.ID
	Routable   bool
}

var sceneSpecs = []SceneSpec{
	{ID: SceneGlobal, Routable: true},
	{ID: SceneSystem, MenuAction: "menu.system", Routable: true},
	{ID: ScenePanel, MenuAction: "menu.panel", Routable: true},
	{ID: SceneFloating, MenuAction: "menu.floating", Routable: true},
	{ID: SceneTab, MenuAction: "menu.tab", Routable: true},
	{ID: SceneWorkspace, MenuAction: "menu.workspace", Routable: true},
	{ID: SceneResize, MenuAction: "menu.resize", Routable: true},
	{ID: SceneCopy, MenuAction: "menu.copy", Routable: true},
	{ID: SceneTerminalPicker, MenuAction: "menu.terminal_picker"},
	{ID: SceneTerminalPool, MenuAction: "menu.terminal_pool"},
	{ID: SceneConnections, MenuAction: "menu.connections"},
	{ID: SceneWorkbenchTree, MenuAction: "menu.workbench_tree"},
	{ID: SceneClipboardHistory, MenuAction: "menu.clipboard_history"},
	{ID: SceneFloatingOverview, MenuAction: "menu.floating_overview"},
	{ID: ScenePrompt, MenuAction: "menu.prompt"},
	{ID: SceneHelp, MenuAction: "menu.help"},
}

// NormalizeScene 规范化配置和历史 input mode 中的场景名；pane 只作为 panel 的输入别名存在。
func NormalizeScene(source string) SceneID {
	source = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(source)), "-", "_")
	if source == "pane" {
		return ScenePanel
	}
	return SceneID(source)
}

// SceneByName 返回内置 scene 声明；未知或未来插件 scene 不会被默认接受。
func SceneByName(source string) (SceneSpec, bool) {
	id := NormalizeScene(source)
	for _, scene := range sceneSpecs {
		if scene.ID == id {
			return scene, true
		}
	}
	return SceneSpec{}, false
}

// SceneByMenuAction 返回 canonical menu action 要进入的内置 scene。
// input 的双击前缀和 app handler 通过该反向查询复用同一声明，不能再维护 menu action -> scene 字符串表。
func SceneByMenuAction(id actiondomain.ID) (SceneSpec, bool) {
	for _, scene := range sceneSpecs {
		if scene.MenuAction == id && id != "" {
			return scene, true
		}
	}
	return SceneSpec{}, false
}

// Scenes 返回内置 scene 的只读副本，供 catalog、配置和 handler 完备性门禁消费。
func Scenes() []SceneSpec { return append([]SceneSpec(nil), sceneSpecs...) }

var policies = buildPolicies()

func buildPolicies() map[actiondomain.ID]BindingPolicy {
	out := map[actiondomain.ID]BindingPolicy{}
	visible := func(scenes []SceneID, routable bool, ids ...string) {
		for _, id := range ids {
			out[actiondomain.ID(id)] = BindingPolicy{AllowedScenes: scenes, Footer: VisibilityVisible, Help: VisibilityVisible, Routable: routable}
		}
	}
	helpOnly := func(scene SceneID, ids ...string) {
		for _, id := range ids {
			out[actiondomain.ID(id)] = BindingPolicy{AllowedScenes: []SceneID{scene}, Footer: VisibilityHidden, Help: VisibilityVisible, Routable: true}
		}
	}
	routed := []SceneID{SceneGlobal, ScenePanel, SceneResize, SceneSystem, SceneFloating, SceneTab, SceneWorkspace, SceneCopy}
	for _, scene := range sceneSpecs {
		if scene.MenuAction != "" {
			visible(routed, true, scene.MenuAction.String())
		}
	}
	visible([]SceneID{SceneCopy}, true, "copy.request_older", "copy.request_newer")
	visible([]SceneID{SceneGlobal, ScenePanel, SceneCopy}, true, "clipboard.paste_latest", "clipboard.paste_system")
	helpOnly(SceneCopy, "copy.line_start", "copy.line_end", "copy.cursor_left", "copy.cursor_right", "copy.cursor_down", "copy.cursor_up", "copy.accept", "copy.oldest", "copy.newest", "copy.half_page_older", "copy.half_page_newer", "copy.mark", "copy.copy_selection", "copy.search_start", "copy.search_next", "copy.search_previous")
	visible([]SceneID{ScenePanel}, true, "panel.close", "panel.detach", "panel.reconnect", "panel.restart", "panel.split_right", "panel.split_down", "panel.kill", "panel.kill_and_close", "panel.toggle_zoom", "panel.presentation_card", "panel.presentation_split_line", "panel.focus_next", "panel.focus_prev")
	visible([]SceneID{ScenePanel, SceneResize}, true, "panel.take_owner", "panel.size_lock", "panel.balance")
	visible([]SceneID{SceneResize}, true, "resize.left", "resize.right", "resize.up", "resize.down", "resize.left_large", "resize.right_large", "resize.up_large", "resize.down_large", "resize.layout_toggle", "resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down", "resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom", "resize.center", "resize.center_x", "resize.center_y", "resize.layout_reset")
	visible([]SceneID{SceneSystem}, true, "system.toggle_header", "system.toggle_footer", "system.clear_toasts", "system.close_toast", "system.toggle_shortcut_lock", "system.quit")
	visible([]SceneID{SceneFloating}, true, "floating.new", "floating.take_owner", "floating.collapse", "floating.center", "floating.toggle_all", "floating.fit", "floating.auto_fit", "floating.move_left", "floating.move_right", "floating.move_up", "floating.move_down", "floating.narrow", "floating.wide", "floating.short", "floating.tall")
	visible([]SceneID{SceneFloating, SceneFloatingOverview}, true, "floating.close", "floating.summon")
	visible([]SceneID{SceneTab}, true, "tab.create", "tab.next", "tab.previous", "tab.rename", "tab.close", "tab.kill")
	visible([]SceneID{SceneGlobal, SceneTab}, true, "tab.jump")
	visible([]SceneID{SceneWorkspace}, true, "workspace.create", "workspace.next", "workspace.previous", "workspace.rename", "workspace.delete")
	visible([]SceneID{SceneTerminalPicker}, false, "terminal_picker.attach", "terminal_picker.split", "terminal_picker.edit", "terminal_picker.kill", "terminal_picker.delete", "terminal_picker.close")
	visible([]SceneID{SceneTerminalPool}, false, "terminal_pool.attach", "terminal_pool.attach_tab", "terminal_pool.attach_float", "terminal_pool.restart", "terminal_pool.edit", "terminal_pool.kill", "terminal_pool.delete", "terminal_pool.close")
	visible([]SceneID{SceneConnections}, false, "connections.edit", "connections.toggle", "connections.refresh", "connections.close")
	visible([]SceneID{SceneWorkbenchTree}, false, "workbench_tree.open", "workbench_tree.new", "workbench_tree.rename", "workbench_tree.delete", "workbench_tree.detach", "workbench_tree.zoom", "workbench_tree.close")
	visible([]SceneID{SceneClipboardHistory}, false, "clipboard_history.paste", "clipboard_history.new", "clipboard_history.edit", "clipboard_history.delete", "clipboard_history.close")
	visible([]SceneID{SceneFloatingOverview}, false, "floating_overview.open", "floating_overview.show_all", "floating_overview.collapse_all", "floating_overview.close")
	visible([]SceneID{ScenePrompt}, false, "prompt.submit", "prompt.cancel")
	visible([]SceneID{SceneHelp}, false, "help.previous", "help.next", "help.page_up", "help.page_down")
	helpOnly(SceneHelp, "help.first", "help.last", "help.close")
	return out
}

// PolicyForSource 解析配置 action 引用并返回其 shortcut binding policy。
// action alias/参数先由 tui/action canonicalize；不存在 policy 表示该 action 不是可配置快捷键入口。
func PolicyForSource(source string) (BindingPolicy, actiondomain.Invocation, actiondomain.Spec, bool) {
	invocation, spec, err := actiondomain.ParseInvocation(source)
	if err != nil {
		return BindingPolicy{}, actiondomain.Invocation{}, actiondomain.Spec{}, false
	}
	policy, ok := policies[invocation.ID]
	return policy, invocation, spec, ok
}

// AllowsScene 判断 canonical action 是否允许绑定到 scene。
// pane 仅作为输入 mode 的历史名称归一为 panel，不建立新的 scene identity。
func AllowsScene(id actiondomain.ID, scene string) bool {
	sceneID := NormalizeScene(scene)
	policy, ok := policies[id]
	if !ok {
		return false
	}
	for _, allowed := range policy.AllowedScenes {
		if allowed == sceneID {
			return true
		}
	}
	return false
}

// Policies 返回 shortcut 可绑定 action 的只读副本，用于完备性测试。
func Policies() map[actiondomain.ID]BindingPolicy {
	out := make(map[actiondomain.ID]BindingPolicy, len(policies))
	for id, policy := range policies {
		policy.AllowedScenes = append([]SceneID(nil), policy.AllowedScenes...)
		out[id] = policy
	}
	return out
}
