// Package action owns every TUI interaction's canonical identity, invocation,
// parameter schema, and default semantic label.
package action

import (
	"fmt"
	"strconv"
	"strings"
)

// ID 是全部 TUI 交互入口共享的 canonical action identity。
// action domain 是它的唯一 owner；shortcut、render 和 app 只能引用，不能重新声明别名或第二套身份。
type ID string

// String 返回 canonical action ID 的配置/消息字符串表示。
func (id ID) String() string { return string(id) }

// ParamSpec 描述一个 canonical action 的整数参数约束。
// 参数由 action domain 在配置加载或 surface invocation 构造时验证，执行层不得自行猜测范围。
type ParamSpec struct {
	Name string
	Min  int
	Max  int
}

// Spec 是一个 action 的中立领域声明。
// 它只拥有 identity、配置别名、参数 schema 与默认语义 label；scene/binding 属于 shortcut，
// projection/clickability 属于 render，handler 与失败语义属于 app。
type Spec struct {
	ID           ID
	Aliases      []string
	DefaultLabel string
	Param        *ParamSpec
}

// Invocation 是 keyboard、mouse、drag 与内容 CTA 共享的执行请求。
// SourceActionID 只用于配置错误定位，执行只能读取 canonical ID 和已验证参数。
type Invocation struct {
	ID             ID
	Params         map[string]int
	SourceActionID string
}

// Signature 返回可稳定比较的 invocation 身份。
// shortcut domain 用它判断多个展示项是否仍指向同一个 action 与参数；当前 canonical 参数只有 index，
// 新增参数时必须同步扩展这里，避免 renderer 把语义不同的动作合并成可点击提示。
func (invocation Invocation) Signature() string {
	parts := []string{string(invocation.ID)}
	if invocation.Params != nil {
		if value, ok := invocation.Params["index"]; ok {
			parts = append(parts, "index="+strconv.Itoa(value))
		}
	}
	return strings.Join(parts, "\x00")
}

// Param 返回 invocation 中已由 action schema 验证的整数参数。
func (invocation Invocation) Param(name string) (int, bool) {
	value, ok := invocation.Params[name]
	return value, ok
}

var defaultLabels = map[string]string{
	"menu.panel":                     "PANE",
	"menu.resize":                    "RESIZE",
	"menu.system":                    "GLOBAL",
	"menu.floating":                  "FLOAT",
	"menu.tab":                       "TAB",
	"menu.workspace":                 "WORKSPACE",
	"menu.copy":                      "COPY",
	"menu.terminal_picker":           "PICKER",
	"menu.terminal_pool":             "TERMINALS",
	"menu.connections":               "CONNECTIONS",
	"menu.workbench_tree":            "TREE",
	"menu.clipboard_history":         "CLIPBOARD",
	"menu.floating_overview":         "OVERVIEW",
	"menu.prompt":                    "PROMPT",
	"menu.help":                      "HELP",
	"panel.close":                    "CLOSE",
	"panel.detach":                   "DETACH",
	"panel.take_owner":               "OWNER",
	"panel.size_lock":                "LOCK",
	"panel.split_right":              "VSPLIT",
	"panel.split_down":               "HSPLIT",
	"panel.toggle_zoom":              "ZOOM",
	"panel.balance":                  "BALANCE",
	"panel.presentation_card":        "CARD",
	"panel.presentation_split_line":  "LINE",
	"panel.focus_next":               "FOCUS",
	"panel.focus_prev":               "FOCUS",
	"resize.left":                    "resize left",
	"resize.right":                   "resize right",
	"resize.up":                      "resize up",
	"resize.down":                    "resize down",
	"resize.layout_toggle":           "LAYOUT",
	"resize.pan_left":                "PAN",
	"resize.pan_down":                "PAN",
	"resize.pan_up":                  "PAN",
	"resize.pan_right":               "PAN",
	"resize.align_left":              "ALIGN",
	"resize.align_right":             "ALIGN",
	"resize.align_top":               "ALIGN",
	"resize.align_bottom":            "ALIGN",
	"resize.center_x":                "CENTER",
	"resize.center_y":                "CENTER",
	"resize.center":                  "CENTER",
	"resize.layout_reset":            "RESET",
	"system.toggle_header":           "HEADER",
	"system.toggle_footer":           "FOOTER",
	"system.clear_toasts":            "CLEAR",
	"system.close_toast":             "TOAST",
	"system.toggle_shortcut_lock":    "KEYLOCK",
	"system.quit":                    "QUIT",
	"floating.new":                   "NEW FLOAT",
	"floating.take_owner":            "OWNER",
	"floating.collapse":              "HIDE",
	"floating.toggle_all":            "ALL",
	"floating.auto_fit":              "AUTO-FIT",
	"floating.summon":                "SUMMON",
	"floating.close":                 "CLOSE",
	"floating.center":                "CENTER",
	"floating.fit":                   "FIT",
	"tab.create":                     "NEW",
	"tab.next":                       "NEXT",
	"tab.previous":                   "PREV",
	"tab.rename":                     "RENAME",
	"tab.close":                      "CLOSE",
	"tab.kill":                       "KILL",
	"workspace.create":               "NEW",
	"workspace.next":                 "NEXT",
	"workspace.previous":             "PREV",
	"workspace.rename":               "RENAME",
	"workspace.delete":               "DELETE",
	"copy.request_older":             "SCROLL",
	"clipboard.paste_latest":         "PASTE",
	"clipboard.paste_system":         "PASTE SYSTEM",
	"terminal_pool.attach_tab":       "TAB",
	"terminal_pool.attach_float":     "FLOAT",
	"terminal_pool.attach":           "ATTACH",
	"terminal_pool.restart":          "RESTART",
	"terminal_pool.edit":             "RENAME",
	"terminal_pool.kill":             "KILL",
	"terminal_pool.delete":           "REMOVE",
	"connections.edit":               "SETTINGS",
	"connections.toggle":             "TOGGLE",
	"connections.refresh":            "REFRESH",
	"floating_overview.open":         "OPEN",
	"floating_overview.show_all":     "SHOW ALL",
	"floating_overview.collapse_all": "COLLAPSE ALL",
	"help.previous":                  "PREV",
	"help.next":                      "NEXT",
	"help.page_up":                   "PAGE",
	"help.page_down":                 "PAGE",
	"help.first":                     "FIRST",
	"help.last":                      "LAST",
	"clipboard_history.paste":        "PASTE",
	"clipboard_history.new":          "NEW",
	"clipboard_history.edit":         "EDIT",
	"clipboard_history.delete":       "DELETE",
}

var specs = buildSpecs()

func buildSpecs() map[string]Spec {
	out := map[string]Spec{}
	add := func(spec Spec) {
		if spec.DefaultLabel == "" {
			if label := defaultLabels[string(spec.ID)]; label != "" {
				spec.DefaultLabel = label
			} else {
				labelID := string(spec.ID)
				if _, suffix, ok := strings.Cut(labelID, "."); ok {
					labelID = suffix
				}
				spec.DefaultLabel = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(labelID)
			}
		}
		if _, exists := out[string(spec.ID)]; exists {
			panic("duplicate canonical action or alias " + string(spec.ID))
		}
		out[string(spec.ID)] = spec
		for _, alias := range spec.Aliases {
			if _, exists := out[alias]; exists {
				panic("duplicate canonical action or alias " + alias)
			}
			out[alias] = spec
		}
	}
	for _, spec := range []Spec{
		{ID: "menu.panel", Aliases: []string{"menu.pane"}},
		{ID: "menu.resize"}, {ID: "menu.system"}, {ID: "menu.floating"}, {ID: "menu.tab"}, {ID: "menu.workspace"},
		{ID: "menu.copy", Aliases: []string{"copy.enter"}},
		{ID: "menu.terminal_picker", Aliases: []string{"terminal_picker.open", "picker.open", "system.open_terminal_picker"}},
		{ID: "menu.terminal_pool", Aliases: []string{"system.open_terminal_pool"}},
		{ID: "menu.connections", Aliases: []string{"system.open_connections"}},
		{ID: "menu.workbench_tree", Aliases: []string{"system.open_workbench_tree"}},
		{ID: "menu.clipboard_history"},
		{ID: "menu.floating_overview", Aliases: []string{"floating.overview"}},
		{ID: "menu.prompt", Aliases: []string{"system.open_prompt"}},
		{ID: "menu.help", Aliases: []string{"system.open_help"}},
	} {
		add(spec)
	}
	addFixed := func(ids ...string) {
		for _, id := range ids {
			add(Spec{ID: ID(id)})
		}
	}
	addFixed("copy.request_older", "copy.request_newer", "clipboard.paste_latest", "clipboard.paste_system", "copy.line_start", "copy.line_end", "copy.cursor_left", "copy.cursor_right", "copy.cursor_down", "copy.cursor_up", "copy.accept", "copy.oldest", "copy.newest", "copy.half_page_older", "copy.half_page_newer", "copy.mark", "copy.copy_selection", "copy.search_start", "copy.search_mode", "copy.search_next", "copy.search_previous")
	panel := []string{"panel.close", "panel.detach", "panel.reconnect", "panel.restart", "panel.take_owner", "panel.size_lock", "panel.split_right", "panel.split_down", "panel.kill", "panel.kill_and_close", "panel.toggle_zoom", "panel.balance", "panel.presentation_card", "panel.presentation_split_line", "panel.focus_next", "panel.focus_prev"}
	for _, id := range panel {
		alias := "pane." + strings.TrimPrefix(id, "panel.")
		add(Spec{ID: ID(id), Aliases: []string{alias}})
	}
	addFixed("resize.left", "resize.right", "resize.up", "resize.down", "resize.left_large", "resize.right_large", "resize.up_large", "resize.down_large", "resize.layout_toggle", "resize.pan_left", "resize.pan_right", "resize.pan_up", "resize.pan_down", "resize.align_left", "resize.align_right", "resize.align_top", "resize.align_bottom", "resize.center", "resize.center_x", "resize.center_y", "resize.layout_reset")
	addFixed("system.toggle_header", "system.toggle_footer", "system.clear_toasts", "system.close_toast", "system.toggle_shortcut_lock", "system.quit")
	addFixed("floating.new", "floating.take_owner", "floating.collapse", "floating.center", "floating.toggle_all", "floating.fit", "floating.auto_fit", "floating.move_left", "floating.move_right", "floating.move_up", "floating.move_down", "floating.narrow", "floating.wide", "floating.short", "floating.tall", "floating.close")
	addFixed("tab.create", "tab.next", "tab.previous", "tab.rename", "tab.close", "tab.kill")
	add(Spec{ID: "tab.jump", Param: &ParamSpec{Name: "index", Min: 1, Max: 9}})
	add(Spec{ID: "floating.summon", Param: &ParamSpec{Name: "index", Min: 1, Max: 9}})
	addFixed("workspace.create", "workspace.next", "workspace.previous", "workspace.rename", "workspace.delete")
	addFixed("terminal_picker.attach", "terminal_picker.split", "terminal_picker.edit", "terminal_picker.kill", "terminal_picker.delete", "terminal_picker.close")
	addFixed("terminal_pool.attach", "terminal_pool.attach_tab", "terminal_pool.attach_float", "terminal_pool.restart", "terminal_pool.edit", "terminal_pool.kill", "terminal_pool.delete", "terminal_pool.close")
	addFixed("connections.edit", "connections.toggle", "connections.refresh", "connections.close")
	addFixed("workbench_tree.open", "workbench_tree.new", "workbench_tree.rename", "workbench_tree.delete", "workbench_tree.detach", "workbench_tree.zoom", "workbench_tree.close")
	addFixed("clipboard_history.paste", "clipboard_history.new", "clipboard_history.edit", "clipboard_history.delete", "clipboard_history.close")
	addFixed("floating_overview.open", "floating_overview.show_all", "floating_overview.collapse_all", "floating_overview.close")
	addFixed("prompt.submit", "prompt.cancel", "help.previous", "help.next", "help.page_up", "help.page_down", "help.first", "help.last", "help.close")
	for _, id := range surfaceOnlyActionIDs {
		add(Spec{ID: id})
	}
	return out
}

// ParseInvocation 把配置引用 canonicalize 为唯一 Invocation，并验证参数 schema。
// 未知 identity、缺少参数或参数越界都会失败，调用方不得 fallback 到字符串执行路径。
func ParseInvocation(source string) (Invocation, Spec, error) {
	source = strings.TrimSpace(source)
	base, paramText, parameterized := splitParameterizedAction(source)
	spec, ok := specs[base]
	if !ok {
		return Invocation{}, Spec{}, fmt.Errorf("unknown action %q", source)
	}
	invocation := Invocation{ID: spec.ID, SourceActionID: source}
	if spec.Param == nil {
		if parameterized {
			return Invocation{}, Spec{}, fmt.Errorf("action %q does not accept a parameter", source)
		}
		return invocation, spec, nil
	}
	if !parameterized {
		return Invocation{}, Spec{}, fmt.Errorf("action %q requires %s", source, spec.Param.Name)
	}
	value, err := strconv.Atoi(paramText)
	if err != nil || value < spec.Param.Min || value > spec.Param.Max {
		return Invocation{}, Spec{}, fmt.Errorf("action %q has invalid %s", source, spec.Param.Name)
	}
	invocation.Params = map[string]int{spec.Param.Name: value}
	return invocation, spec, nil
}

func splitParameterizedAction(source string) (string, string, bool) {
	for _, base := range []string{"tab.jump", "floating.summon"} {
		prefix := base + "."
		if strings.HasPrefix(source, prefix) {
			return base, strings.TrimPrefix(source, prefix), true
		}
	}
	return source, "", false
}

// SpecForSource 解析配置引用并返回 canonical spec；参数错误同样视为解析失败。
func SpecForSource(source string) (Spec, bool) {
	_, spec, err := ParseInvocation(source)
	return spec, err == nil
}

// SpecByID 返回 canonical ID 对应的领域 spec，不执行参数解析或 alias 解析。
// renderer/app 用它验证引用存在性；配置字符串仍必须走 ParseInvocation。
func SpecByID(id ID) (Spec, bool) {
	spec, ok := specs[string(id)]
	return spec, ok && spec.ID == id
}

// Specs 返回每个 canonical action spec 一次，不包含 alias 条目。
// 该只读 inventory 用于 shortcut、render 与 app 的跨层完备性门禁。
func Specs() []Spec {
	seen := map[ID]bool{}
	out := []Spec{}
	for _, spec := range specs {
		if seen[spec.ID] {
			continue
		}
		seen[spec.ID] = true
		out = append(out, spec)
	}
	return out
}
