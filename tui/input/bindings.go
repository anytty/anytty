package input

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/shortcut"
	"github.com/anytty/anytty/tui/state"
)

// Binding 是 shortcut catalog 编译后的运行时路由项。
// 它由内置默认 shortcuts 或用户 `tui.shortcuts` 生成；输入路由只查 Binding，
// 不再直接读取旧 keymap 或分散硬编码键位。
type Binding struct {
	ID         string
	ActionID   string
	Label      string
	Mode       InteractionMode
	Key        Key
	Char       string
	Ctrl       bool
	Alt        bool
	Shift      bool
	Invocation actiondomain.Invocation
	// RequiresKeyboardDisambiguation 表示传统 TTY 无法区分该组合；只有规范化后的 CSI-u 事件可命中。
	RequiresKeyboardDisambiguation bool
}

// ShortcutEntry 是 shortcut catalog 面向提示层和 overlay 输入层的只读条目。
// Scene/Key/ActionID 来自内置默认或 `tui.shortcuts`；Label 只来自配置覆盖，不生成执行语义。
type ShortcutEntry struct {
	Scene    string
	Key      string
	KeyLabel string
	ActionID string
	Label    string
	Show     *bool
}

type shortcutCatalog struct {
	bindings []Binding
}

func (catalog shortcutCatalog) bindingsCopy() []Binding {
	out := make([]Binding, len(catalog.bindings))
	copy(out, catalog.bindings)
	return out
}

// BindingCatalog 返回内置默认 shortcuts 编译出的路由项。
// 它只用于测试和后续提示投影读取默认目录；运行时路由会按当前 config 重新生成 catalog。
func BindingCatalog() []Binding {
	return shortcutCatalogForConfig(state.TUIShortcutConfig{}).bindingsCopy()
}

// ShortcutEntriesForConfig 返回当前配置下的完整 shortcut catalog 条目。
// 该函数是 footer/help/overlay 提示层的快捷键真值；route 只消费其中可路由的 scene。
func ShortcutEntriesForConfig(shortcuts state.TUIShortcutConfig) []ShortcutEntry {
	if shortcutConfigUsesUserCatalog(shortcuts) {
		return shortcutEntriesFromConfig(shortcuts)
	}
	entries := make([]ShortcutEntry, 0, len(shortcut.DefaultBindings()))
	for _, item := range shortcut.DefaultBindings() {
		entries = append(entries, shortcutEntryFromParts(item.Scene, item.Key, item.Action, ""))
	}
	return entries
}

// ShortcutEntriesForScene 返回某个 scene 下的快捷键条目，保持 catalog 中的定义顺序。
func ShortcutEntriesForScene(shortcuts state.TUIShortcutConfig, sceneName string) []ShortcutEntry {
	entries := ShortcutEntriesForConfig(shortcuts)
	sceneKey := shortcutSceneCatalogKey(sceneName)
	out := make([]ShortcutEntry, 0, len(entries))
	for _, entry := range entries {
		if shortcutSceneCatalogKey(entry.Scene) == sceneKey {
			out = append(out, entry)
		}
	}
	return out
}

// ShortcutEntriesForHelp returns the actual configurable bindings exposed by Help.
// Keeping this filter beside the canonical input catalog lets rendering and navigation
// use the same ordered item set.
func ShortcutEntriesForHelp(shortcuts state.TUIShortcutConfig, keyboardDisambiguation bool) []ShortcutEntry {
	entries := ShortcutEntriesForConfig(shortcuts)
	out := make([]ShortcutEntry, 0, len(entries))
	for _, entry := range entries {
		policy, _, _, ok := shortcut.PolicyForSource(entry.ActionID)
		if !ok || policy.Help != shortcut.VisibilityVisible {
			continue
		}
		if ShortcutKeyRequiresEnhancedKeyboard(entry.Key) && !keyboardDisambiguation {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// ShortcutEntryForEvent 按 route 同源的 key matcher 查找某个 scene 下的 action。
// overlay reducer 用它把提示 catalog 直接作为键盘动作表，避免 overlay 另写一套快捷键。
func ShortcutEntryForEvent(shortcuts state.TUIShortcutConfig, sceneName string, event InputEvent) (ShortcutEntry, bool) {
	if event.Kind != EventKindKey {
		return ShortcutEntry{}, false
	}
	for _, entry := range ShortcutEntriesForScene(shortcuts, sceneName) {
		key, ok := parseShortcutKeyToken(entry.Key)
		if !ok {
			continue
		}
		binding := Binding{Key: key.Key, Char: key.Char, Ctrl: key.Ctrl, Alt: key.Alt, Shift: key.Shift, RequiresKeyboardDisambiguation: ShortcutKeyRequiresEnhancedKeyboard(entry.Key)}
		if bindingMatches(binding, event) {
			return entry, true
		}
	}
	return ShortcutEntry{}, false
}

func shortcutCatalogForConfig(shortcuts state.TUIShortcutConfig) shortcutCatalog {
	if shortcutConfigUsesUserCatalog(shortcuts) {
		return shortcutCatalogFromConfig(shortcuts)
	}
	catalog := shortcutCatalog{bindings: make([]Binding, 0, len(shortcut.DefaultBindings()))}
	for _, item := range shortcut.DefaultBindings() {
		catalog.add(item.Scene, item.Key, item.Action, "")
	}
	return catalog
}

func shortcutConfigUsesUserCatalog(shortcuts state.TUIShortcutConfig) bool {
	// 只有显式 scene 才声明用户 binding catalog；actions 只覆盖文案，必须继续复用默认 bindings。
	return len(shortcuts.Scenes) > 0
}

func shortcutCatalogFromConfig(shortcuts state.TUIShortcutConfig) shortcutCatalog {
	catalog := shortcutCatalog{}
	for sceneName, scene := range shortcuts.Scenes {
		for key, binding := range scene.Bindings {
			catalog.add(sceneName, key, binding.Action, binding.Label)
		}
	}
	return catalog
}

func shortcutEntriesFromConfig(shortcuts state.TUIShortcutConfig) []ShortcutEntry {
	sceneNames := make([]string, 0, len(shortcuts.Scenes))
	for sceneName := range shortcuts.Scenes {
		sceneNames = append(sceneNames, sceneName)
	}
	sort.Strings(sceneNames)
	entries := []ShortcutEntry{}
	for _, sceneName := range sceneNames {
		scene := shortcuts.Scenes[sceneName]
		keys := make([]string, 0, len(scene.Bindings))
		for key := range scene.Bindings {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			binding := scene.Bindings[key]
			entry := shortcutEntryFromParts(sceneName, key, binding.Action, binding.Label)
			entry.Show = binding.Show
			entries = append(entries, entry)
		}
	}
	return entries
}

func shortcutEntryFromParts(sceneName string, keyToken string, actionID string, label string) ShortcutEntry {
	return ShortcutEntry{
		Scene:    sceneName,
		Key:      keyToken,
		KeyLabel: ShortcutKeyDisplay(keyToken),
		ActionID: actionID,
		Label:    label,
	}
}

func shortcutSceneCatalogKey(sceneName string) string {
	return string(shortcut.NormalizeScene(sceneName))
}

func (catalog *shortcutCatalog) add(sceneName string, keyToken string, actionID string, label string) {
	mode, ok := shortcutSceneMode(sceneName)
	if !ok {
		return
	}
	key, ok := parseShortcutKeyToken(keyToken)
	if !ok {
		return
	}
	invocation, _, err := actiondomain.ParseInvocation(actionID)
	if err != nil {
		return
	}
	definition := Binding{ActionID: actionID, Invocation: invocation}
	definition.ID = sceneName + ":" + keyToken + ":" + actionID
	definition.Mode = mode
	definition.Key = key.Key
	definition.Char = key.Char
	definition.Ctrl = key.Ctrl
	definition.Alt = key.Alt
	definition.Shift = key.Shift
	definition.RequiresKeyboardDisambiguation = ShortcutKeyRequiresEnhancedKeyboard(keyToken)
	definition.Label = label
	catalog.bindings = append(catalog.bindings, definition)
}

type shortcutKey struct {
	Key   Key
	Char  string
	Ctrl  bool
	Alt   bool
	Shift bool
}

func parseShortcutKeyToken(token string) (shortcutKey, bool) {
	rest := strings.TrimSpace(token)
	if rest == "" {
		return shortcutKey{}, false
	}
	key := shortcutKey{}
	for {
		lower := strings.ToLower(rest)
		switch {
		case strings.HasPrefix(lower, "ctrl-"):
			key.Ctrl = true
			rest = rest[len("ctrl-"):]
		case strings.HasPrefix(lower, "alt-"):
			key.Alt = true
			rest = rest[len("alt-"):]
		case strings.HasPrefix(lower, "shift-"):
			key.Shift = true
			rest = rest[len("shift-"):]
		default:
			return parseShortcutKeyTokenBase(rest, key)
		}
		if rest == "" {
			return shortcutKey{}, false
		}
	}
}

func parseShortcutKeyTokenBase(token string, key shortcutKey) (shortcutKey, bool) {
	switch strings.ToLower(token) {
	case "space":
		key.Key = KeyChar
		key.Char = " "
	case "page-up", "pgup":
		key.Key = KeyPageUp
	case "page-down", "pgdn":
		key.Key = KeyPageDn
	case "up":
		key.Key = KeyUp
	case "down":
		key.Key = KeyDown
	case "left":
		key.Key = KeyLeft
	case "right":
		key.Key = KeyRight
	case "home":
		key.Key = KeyHome
	case "end":
		key.Key = KeyEnd
	case "delete":
		key.Key = KeyDelete
	case "insert":
		key.Key = KeyInsert
	case "backspace":
		key.Key = KeyBackspace
	case "tab":
		if key.Shift {
			key.Key = KeyShiftTab
		} else {
			key.Key = KeyTab
		}
	case "esc", "escape":
		key.Key = KeyEsc
	case "enter", "return":
		key.Key = KeyEnter
	case "f1":
		key.Key = KeyF1
	case "f2":
		key.Key = KeyF2
	case "f3":
		key.Key = KeyF3
	case "f4":
		key.Key = KeyF4
	case "f5":
		key.Key = KeyF5
	case "f6":
		key.Key = KeyF6
	case "f7":
		key.Key = KeyF7
	case "f8":
		key.Key = KeyF8
	case "f9":
		key.Key = KeyF9
	case "f10":
		key.Key = KeyF10
	case "f11":
		key.Key = KeyF11
	case "f12":
		key.Key = KeyF12
	default:
		if len([]rune(token)) != 1 {
			return shortcutKey{}, false
		}
		key.Key = KeyChar
		key.Char = token
	}
	return key, true
}

// ShortcutKeyDisplay 将配置里的 key token 转成 footer/help 使用的稳定展示文本。
// 展示文本由同一解析器生成，避免配置能按但提示显示另一套拼写。
func ShortcutKeyDisplay(token string) string {
	key, ok := parseShortcutKeyToken(token)
	if !ok {
		return strings.TrimSpace(token)
	}
	base := shortcutKeyBaseDisplay(key)
	if key.Ctrl && key.Key == KeyChar && base != "" {
		base = "^" + strings.ToUpper(base)
	} else {
		prefixes := []string{}
		if key.Ctrl {
			prefixes = append(prefixes, "Ctrl")
		}
		if key.Alt {
			prefixes = append(prefixes, "Alt")
		}
		if key.Shift {
			prefixes = append(prefixes, "Shift")
		}
		if len(prefixes) > 0 {
			base = strings.Join(append(prefixes, base), "+")
		}
	}
	if base == "" {
		return strings.TrimSpace(token)
	}
	return base
}

// ShortcutKeyRequiresEnhancedKeyboard 判断 binding 是否依赖传统 TTY 无法区分的 modifier 编码。
// catalog 用该结果阻止歧义事件误命中；renderer 同时用它按已确认的 host capability 隐藏不可用提示。
// capability availability 由宿主 TerminalHost 探测；该函数只复用 canonical key parser 判定键位语义。
func ShortcutKeyRequiresEnhancedKeyboard(token string) bool {
	key, ok := parseShortcutKeyToken(token)
	if !ok {
		return false
	}
	if key.Key == KeyChar {
		if key.Shift {
			return true
		}
		if key.Ctrl {
			data, representable := ctrlCharBytes(key.Char)
			if !representable {
				return true
			}
			// 传统 TTY 会先把这些 control bytes 规范化为 named key，无法命中 KeyChar+Ctrl binding。
			switch data[0] {
			case '\t', '\n', '\r', '\x1b', '\x7f':
				return true
			default:
				return false
			}
		}
		return false
	}
	if key.Key == KeyShiftTab {
		return false
	}
	if !key.Ctrl && !key.Shift {
		// Alt+named key 可由传统 ESC prefix 表达；无 modifier 的 named key 走传统 CSI/SS3。
		return false
	}
	switch key.Key {
	case KeyUp, KeyDown, KeyLeft, KeyRight, KeyHome, KeyEnd, KeyDelete, KeyInsert,
		KeyPageUp, KeyPageDn, KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6,
		KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12:
		return false
	default:
		return true
	}
}

func shortcutKeyBaseDisplay(key shortcutKey) string {
	switch key.Key {
	case KeyChar:
		if key.Char == " " {
			return "space"
		}
		return key.Char
	case KeyPageUp:
		return "PgUp"
	case KeyPageDn:
		return "PgDn"
	case KeyUp:
		return "↑"
	case KeyDown:
		return "↓"
	case KeyLeft:
		return "←"
	case KeyRight:
		return "→"
	case KeyHome:
		return "home"
	case KeyEnd:
		return "end"
	case KeyDelete:
		return "delete"
	case KeyInsert:
		return "insert"
	case KeyBackspace:
		return "backspace"
	case KeyTab:
		return "tab"
	case KeyShiftTab:
		return "shift-tab"
	case KeyEsc:
		return "Esc"
	case KeyEnter:
		return "enter"
	default:
		return string(key.Key)
	}
}

// InteractionModeForShortcutScene 是 shortcut scene 到 input sticky mode 的唯一投影。
// shortcut 拥有 scene identity；input 拥有键盘路由 mode。app 只能组合这两个 owner，不能另建 menu action 映射。
func InteractionModeForShortcutScene(sceneID shortcut.SceneID) (InteractionMode, bool) {
	scene, ok := shortcut.SceneByName(string(sceneID))
	if !ok || !scene.Routable {
		return InteractionModeNormal, false
	}
	switch scene.ID {
	case shortcut.SceneGlobal:
		return InteractionModeNormal, true
	case shortcut.SceneSystem:
		return InteractionModeGlobal, true
	case shortcut.ScenePanel:
		return InteractionModePane, true
	case shortcut.SceneResize:
		return InteractionModeResize, true
	case shortcut.SceneFloating:
		return InteractionModeFloating, true
	case shortcut.SceneTab:
		return InteractionModeTab, true
	case shortcut.SceneWorkspace:
		return InteractionModeWorkspace, true
	case shortcut.SceneCopy:
		return InteractionModeCopy, true
	default:
		return InteractionModeNormal, false
	}
}

func shortcutSceneMode(sceneName string) (InteractionMode, bool) {
	return InteractionModeForShortcutScene(shortcut.NormalizeScene(sceneName))
}

// ShortcutBindingSignature 返回 scene/key 在配置层输入模型里的唯一签名。
// routed scene 按 InteractionMode 合并 panel/pane 等别名，overlay scene 按 canonical scene 隔离；
// 它只归一协议无关的 token 别名；实际宿主 capability 与 binding available 状态由 TerminalHost 阶段决定。
func ShortcutBindingSignature(sceneName string, keyToken string) (string, bool) {
	key, ok := parseShortcutKeyToken(keyToken)
	if !ok {
		return "", false
	}
	key.Char = canonicalShortcutChar(key.Key, key.Char, key.Ctrl, key.Shift)
	sceneKey := shortcutSceneCatalogKey(sceneName)
	if mode, routed := shortcutSceneMode(sceneName); routed {
		sceneKey = "route:" + string(mode)
	} else {
		sceneKey = "overlay:" + sceneKey
	}
	parts := []string{
		sceneKey,
		string(key.Key),
		strconv.Quote(key.Char),
		strconv.FormatBool(key.Ctrl),
		strconv.FormatBool(key.Alt),
		strconv.FormatBool(key.Shift),
	}
	return strings.Join(parts, "\x00"), true
}

// ShortcutKeyIsGlobalEscape 判断配置 key 是否占用了 TUI 保留的全局返回键。
// 未修饰的 esc/escape 由 app back-navigation 统一处理，不进入用户 shortcut catalog。
func ShortcutKeyIsGlobalEscape(keyToken string) bool {
	key, ok := parseShortcutKeyToken(keyToken)
	return ok && key.Key == KeyEsc && !key.Ctrl && !key.Alt && !key.Shift
}

// KnownShortcutActionID 判断 action id 是否属于 shortcut domain registry。
// 配置加载期用它拒绝拼写错误；运行时 input 只生成 canonical invocation。
func KnownShortcutActionID(actionID string) bool {
	_, _, _, ok := shortcut.PolicyForSource(actionID)
	return ok
}

// RoutableShortcutActionID 判断 action 是否能由主 input route 直接产出 reducer intent。
// overlay 专用 action 只能出现在 overlay scene，不能绑定到 global/panel 等可路由 scene 后静默吞键。
func RoutableShortcutActionID(actionID string) bool {
	policy, _, _, ok := shortcut.PolicyForSource(actionID)
	return ok && policy.Routable
}

func routeKey(event InputEvent, options RouteOptions) Intent {
	if options.ForceTerminalPassthrough {
		if data := terminalBytes(event); len(data) > 0 {
			return Intent{Kind: IntentTerminalInput, Event: event, Bytes: data}
		}
		return Intent{Kind: IntentNone, Event: event, Reason: "forced passthrough without bytes"}
	}
	catalog := shortcutCatalogForConfig(options.Shortcuts)
	if options.Mode != InteractionModeNormal {
		if binding, ok := lookupBinding(options.Mode, event, catalog); ok {
			return intentFromBinding(event, binding)
		}
		// 中文说明：sticky 场景下 root/global 菜单键仍可用，但这个顺序来自同一个 catalog；
		// 用户删掉 global shortcut 后不会回落到旧硬编码 route。
		if binding, ok := lookupBinding(InteractionModeNormal, event, catalog); ok {
			return intentFromBinding(event, binding)
		}
		return Intent{Kind: IntentNone, Event: event}
	}
	if options.CopyModeActive {
		if binding, ok := lookupBinding(InteractionModeCopy, event, catalog); ok {
			return intentFromBinding(event, binding)
		}
	}
	if binding, ok := lookupBinding(InteractionModeNormal, event, catalog); ok {
		return intentFromBinding(event, binding)
	}
	if options.CopyModeActive {
		return Intent{Kind: IntentNone, Event: event}
	}
	if data := terminalBytes(event); len(data) > 0 {
		return Intent{Kind: IntentTerminalInput, Event: event, Bytes: data}
	}
	return Intent{Kind: IntentNone, Event: event}
}

func routeMouse(event InputEvent, options RouteOptions) Intent {
	// 中文说明：copy/history 尚未接管时，前台 terminal 的 mouse tracking 优先于滚轮进历史；
	// 一旦 copy/history 已经激活，滚轮仍属于 AnyTTY history。
	if !options.CopyModeActive && options.TerminalMousePassthrough && event.RawSeq != "" {
		return Intent{Kind: IntentTerminalInput, Event: event, Bytes: []byte(event.RawSeq), RawMouse: true}
	}
	switch event.Mouse {
	case MouseWheelUp:
		if options.CopyModeActive {
			return Intent{Kind: IntentRequestOlder, Event: event}
		}
		return Intent{Kind: IntentEnterCopyMode, Event: event}
	case MouseWheelDown:
		if options.CopyModeActive {
			return Intent{Kind: IntentRequestNewer, Event: event}
		}
	case MouseLeft, MouseLeftDrag:
		if options.CopyModeActive {
			return Intent{Kind: IntentMouseSelect, Event: event}
		}
	}
	return Intent{Kind: IntentNone, Event: event}
}

func lookupBinding(mode InteractionMode, event InputEvent, catalog shortcutCatalog) (Binding, bool) {
	for _, binding := range catalog.bindings {
		if binding.Mode == mode && bindingMatches(binding, event) {
			return binding, true
		}
	}
	return Binding{}, false
}

func bindingMatches(binding Binding, event InputEvent) bool {
	if binding.RequiresKeyboardDisambiguation && event.KeyboardProtocol != KeyboardProtocolKittyCSIU {
		return false
	}
	if binding.Key != event.Key {
		return false
	}
	if !bindingCharMatches(binding, event) {
		return false
	}
	return binding.Ctrl == event.Ctrl && binding.Alt == event.Alt && binding.Shift == event.Shift
}

func bindingCharMatches(binding Binding, event InputEvent) bool {
	if binding.Char == event.Char {
		return true
	}
	if binding.Key != KeyChar || !binding.Ctrl || !event.Ctrl {
		return false
	}
	if canonicalShortcutChar(binding.Key, binding.Char, binding.Ctrl, binding.Shift) == canonicalShortcutChar(event.Key, event.Char, event.Ctrl, event.Shift) {
		return true
	}
	data, ok := ctrlCharBytes(binding.Char)
	return ok && string(data) == event.Char
}

func canonicalShortcutChar(key Key, char string, ctrl bool, shift bool) string {
	if key != KeyChar || !ctrl || shift {
		return char
	}
	if char == " " || char == "@" || char == "\x00" {
		return "\x00"
	}
	if len(char) == 1 && char[0] >= 'A' && char[0] <= 'Z' {
		// Ctrl-A 与 Ctrl-a 是配置拼写别名；显式 Shift 必须写成 ctrl-shift-a。
		return strings.ToLower(char)
	}
	return char
}

func intentFromBinding(event InputEvent, binding Binding) Intent {
	return Intent{
		Kind:       IntentShortcutAction,
		Event:      event,
		Invocation: binding.Invocation,
	}
}

// LockableRootShortcutIntent 返回会被 shortcut lock 透传的 root 快捷键。
// global/system mode 入口保留为控制面逃生键；开启 lock 后仍可用 Ctrl-G, l 解除锁定。
func LockableRootShortcutIntent(event InputEvent) (Intent, bool) {
	return LockableRootShortcutIntentWithShortcuts(event, state.TUIShortcutConfig{})
}

// LockableRootShortcutIntentWithShortcuts 使用当前 shortcuts 判断会被 shortcut lock 透传的 root 快捷键。
func LockableRootShortcutIntentWithShortcuts(event InputEvent, shortcuts state.TUIShortcutConfig) (Intent, bool) {
	intent, ok := rootShortcutIntent(event, shortcuts)
	if !ok {
		return Intent{}, false
	}
	if intent.Invocation.ID == "menu.system" {
		return Intent{}, false
	}
	return intent, true
}

// StickyModeEntryShortcutIntent 判断当前按键是否是某个 sticky mode 的入口键。
// UI reducer 用它实现双击前缀透传，例如 Ctrl-W Ctrl-W 将第二个 Ctrl-W 发给 terminal。
func StickyModeEntryShortcutIntent(event InputEvent, mode InteractionMode) (Intent, bool) {
	return StickyModeEntryShortcutIntentWithShortcuts(event, mode, state.TUIShortcutConfig{})
}

// StickyModeEntryShortcutIntentWithShortcuts 使用当前 shortcuts 判断 sticky mode 入口键。
func StickyModeEntryShortcutIntentWithShortcuts(event InputEvent, mode InteractionMode, shortcuts state.TUIShortcutConfig) (Intent, bool) {
	if mode == InteractionModeNormal {
		return Intent{}, false
	}
	intent, ok := rootShortcutIntent(event, shortcuts)
	if !ok || shortcutInvocationMode(intent.Invocation) != mode {
		return Intent{}, false
	}
	return intent, true
}

// CopyModeEntryShortcutIntent 判断当前按键是否是 copy/history 的 root 入口键。
// copy mode 本身不是 sticky interaction mode，但第二次按入口键也应显式透传给 terminal。
func CopyModeEntryShortcutIntent(event InputEvent) (Intent, bool) {
	return CopyModeEntryShortcutIntentWithShortcuts(event, state.TUIShortcutConfig{})
}

// CopyModeEntryShortcutIntentWithShortcuts 使用当前 shortcuts 判断 copy/history 入口键。
func CopyModeEntryShortcutIntentWithShortcuts(event InputEvent, shortcuts state.TUIShortcutConfig) (Intent, bool) {
	intent, ok := rootShortcutIntent(event, shortcuts)
	if !ok || (intent.Invocation.ID != "copy.enter" && intent.Invocation.ID != "menu.copy") {
		return Intent{}, false
	}
	if event.Key != KeyChar || !event.Ctrl {
		return Intent{}, false
	}
	return intent, true
}

func rootShortcutIntent(event InputEvent, shortcuts state.TUIShortcutConfig) (Intent, bool) {
	catalog := shortcutCatalogForConfig(shortcuts)
	binding, ok := lookupBinding(InteractionModeNormal, event, catalog)
	if !ok {
		return Intent{}, false
	}
	return intentFromBinding(event, binding), true
}

func shortcutInvocationMode(invocation actiondomain.Invocation) InteractionMode {
	scene, ok := shortcut.SceneByMenuAction(invocation.ID)
	if !ok {
		return InteractionModeNormal
	}
	mode, ok := InteractionModeForShortcutScene(scene.ID)
	if !ok || mode == InteractionModeCopy {
		return InteractionModeNormal
	}
	return mode
}

func terminalBytes(event InputEvent) []byte {
	if event.RawSeq != "" && event.KeyboardProtocol == "" {
		return []byte(event.RawSeq)
	}
	var data []byte
	switch event.Key {
	case KeyEsc:
		data = []byte{'\x1b'}
	case KeyEnter:
		data = []byte{'\r'}
	case KeyBackspace:
		data = []byte{0x7f}
	case KeyTab:
		data = []byte{'\t'}
	case KeyShiftTab:
		if event.Ctrl || event.Alt {
			return modifiedCSI("Z", event)
		}
		return []byte("\x1b[Z")
	case KeyUp:
		return modifiedCSI("A", event)
	case KeyDown:
		return modifiedCSI("B", event)
	case KeyRight:
		return modifiedCSI("C", event)
	case KeyLeft:
		return modifiedCSI("D", event)
	case KeyHome:
		return modifiedCSI("H", event)
	case KeyEnd:
		return modifiedCSI("F", event)
	case KeyInsert:
		return modifiedTildeCSI(2, event)
	case KeyDelete:
		return modifiedTildeCSI(3, event)
	case KeyPageUp:
		return modifiedTildeCSI(5, event)
	case KeyPageDn:
		return modifiedTildeCSI(6, event)
	case KeyF1, KeyF2, KeyF4:
		final := map[Key]string{KeyF1: "P", KeyF2: "Q", KeyF4: "S"}[event.Key]
		if !event.Ctrl && !event.Alt && !event.Shift {
			return []byte("\x1bO" + final)
		}
		return modifiedCSI(final, event)
	case KeyF3:
		return modifiedTildeCSI(13, event)
	case KeyF5, KeyF6, KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12:
		code := map[Key]int{KeyF5: 15, KeyF6: 17, KeyF7: 18, KeyF8: 19, KeyF9: 20, KeyF10: 21, KeyF11: 23, KeyF12: 24}[event.Key]
		return modifiedTildeCSI(code, event)
	case KeyChar:
		if event.Char != "" {
			if event.Ctrl {
				if data, ok := ctrlCharBytes(event.Char); ok {
					if event.Alt {
						return append([]byte{'\x1b'}, data...)
					}
					return data
				}
				return nil
			}
			if event.Alt {
				return append([]byte{'\x1b'}, []byte(event.Char)...)
			}
			return []byte(event.Char)
		}
	}
	if len(data) > 0 {
		if event.Ctrl || event.Shift {
			// Enter/Tab/Backspace/Esc 没有传统 Ctrl/Shift 编码；禁止降级成有副作用的普通 named key。
			return nil
		}
		if event.Alt {
			return append([]byte{'\x1b'}, data...)
		}
		return data
	}
	return nil
}

func modifiedCSI(final string, event InputEvent) []byte {
	modifier := 1
	if event.Shift {
		modifier++
	}
	if event.Alt {
		modifier += 2
	}
	if event.Ctrl {
		modifier += 4
	}
	if modifier == 1 {
		return []byte("\x1b[" + final)
	}
	return []byte(fmt.Sprintf("\x1b[1;%d%s", modifier, final))
}

func modifiedTildeCSI(code int, event InputEvent) []byte {
	modifier := 1
	if event.Shift {
		modifier++
	}
	if event.Alt {
		modifier += 2
	}
	if event.Ctrl {
		modifier += 4
	}
	if modifier == 1 {
		return []byte(fmt.Sprintf("\x1b[%d~", code))
	}
	return []byte(fmt.Sprintf("\x1b[%d;%d~", code, modifier))
}

func ctrlCharBytes(char string) ([]byte, bool) {
	if len(char) == 0 {
		return nil, false
	}
	if len(char) == 1 && char[0] < 0x20 {
		return []byte{char[0]}, true
	}
	if len(char) != 1 {
		return nil, false
	}
	c := char[0]
	if c >= 'a' && c <= 'z' {
		return []byte{c - 'a' + 1}, true
	}
	if c >= 'A' && c <= 'Z' {
		return []byte{c - 'A' + 1}, true
	}
	switch c {
	case ' ', '@':
		return []byte{0x00}, true
	case '[':
		return []byte{0x1b}, true
	case '\\':
		return []byte{0x1c}, true
	case ']':
		return []byte{0x1d}, true
	case '^':
		return []byte{0x1e}, true
	case '_':
		return []byte{0x1f}, true
	case '?':
		return []byte{0x7f}, true
	}
	return nil, false
}
