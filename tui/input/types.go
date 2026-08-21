package input

import (
	actiondomain "github.com/anytty/anytty/tui/action"
	"github.com/anytty/anytty/tui/state"
)

// EventKind 分类宿主输入，不依赖 Bubble Tea key/mouse 类型。
type EventKind string

const (
	EventKindKey            EventKind = "key"
	EventKindPaste          EventKind = "paste"
	EventKindMouse          EventKind = "mouse"
	EventKindResize         EventKind = "resize"
	EventKindHostTheme      EventKind = "host-theme"
	EventKindHostCapability EventKind = "host-capability"
	EventKindHostControl    EventKind = "host-control"
)

// KeyboardProtocol 标记 InputEvent 来自哪一种宿主键盘协议。
// input router 只用它阻止宿主控制序列泄漏到 PTY；动作匹配仍只依赖标准化后的 key/char/modifier。
type KeyboardProtocol string

const KeyboardProtocolKittyCSIU KeyboardProtocol = "kitty-csi-u"

type Key string

const (
	KeyPageUp    Key = "page-up"
	KeyPageDn    Key = "page-down"
	KeyUp        Key = "up"
	KeyDown      Key = "down"
	KeyLeft      Key = "left"
	KeyRight     Key = "right"
	KeyHome      Key = "home"
	KeyEnd       Key = "end"
	KeyDelete    Key = "delete"
	KeyInsert    Key = "insert"
	KeyBackspace Key = "backspace"
	KeyTab       Key = "tab"
	KeyShiftTab  Key = "shift-tab"
	KeyEsc       Key = "esc"
	KeyEnter     Key = "enter"
	KeyF1        Key = "f1"
	KeyF2        Key = "f2"
	KeyF3        Key = "f3"
	KeyF4        Key = "f4"
	KeyF5        Key = "f5"
	KeyF6        Key = "f6"
	KeyF7        Key = "f7"
	KeyF8        Key = "f8"
	KeyF9        Key = "f9"
	KeyF10       Key = "f10"
	KeyF11       Key = "f11"
	KeyF12       Key = "f12"
	KeyChar      Key = "char"
	KeyUnknown   Key = "unknown"
)

type MouseButton string

const (
	MouseWheelUp    MouseButton = "wheel-up"
	MouseWheelDown  MouseButton = "wheel-down"
	MouseLeft       MouseButton = "left"
	MouseLeftDrag   MouseButton = "left-drag"
	MouseLeftUp     MouseButton = "left-up"
	MouseMiddle     MouseButton = "middle"
	MouseMiddleDrag MouseButton = "middle-drag"
	MouseMiddleUp   MouseButton = "middle-up"
	MouseRight      MouseButton = "right"
	MouseRightDrag  MouseButton = "right-drag"
	MouseRightUp    MouseButton = "right-up"
	MouseMove       MouseButton = "move"
	MouseRelease    MouseButton = "release"
)

// InputEvent 是 TerminalHost 拥有的宿主输入边界。
type InputEvent struct {
	Kind             EventKind
	Key              Key
	Char             string
	Paste            string
	Mouse            MouseButton
	Row              int
	Col              int
	Cols             int
	Rows             int
	Theme            HostThemeEvent
	Capability       HostCapabilityEvent
	KeyboardProtocol KeyboardProtocol
	Alt              bool
	Ctrl             bool
	Shift            bool
	RawSeq           string
}

// HostCapabilityEvent 是 TerminalHost capability query 的结构化结果。
// 它只描述当前宿主 terminal emulator 的会话能力，不代表远端 terminal 或 daemon 能力。
type HostCapabilityEvent struct {
	KeyboardDisambiguation bool
}

type HostThemeEvent struct {
	DefaultFG    string
	DefaultBG    string
	PaletteIndex int
	PaletteColor string
}

type IntentKind string

const (
	IntentNone                 IntentKind = "none"
	IntentOpenTerminalPicker   IntentKind = "open-terminal-picker"
	IntentEnterCopyMode        IntentKind = "enter-copy-mode"
	IntentRequestOlder         IntentKind = "request-older"
	IntentRequestNewer         IntentKind = "request-newer"
	IntentOpenClipboardHistory IntentKind = "open-clipboard-history"
	IntentPasteLastCopy        IntentKind = "paste-last-copy"
	IntentPasteClipboard       IntentKind = "paste-clipboard"
	IntentTerminalInput        IntentKind = "terminal-input"
	IntentMouseSelect          IntentKind = "mouse-select"
	IntentSetInteractionMode   IntentKind = "set-interaction-mode"
	IntentShellAction          IntentKind = "shell-action"
	IntentPaneCommand          IntentKind = "pane-command"
	IntentWorkbenchCommand     IntentKind = "workbench-command"
	IntentCopyCommand          IntentKind = "copy-command"
	IntentShortcutAction       IntentKind = "shortcut-action"
	IntentAppAction            IntentKind = "app-action"
)

type InteractionMode string

const (
	InteractionModeNormal    InteractionMode = ""
	InteractionModePane      InteractionMode = "pane"
	InteractionModeResize    InteractionMode = "resize"
	InteractionModeGlobal    InteractionMode = "global"
	InteractionModeFloating  InteractionMode = "floating"
	InteractionModeTab       InteractionMode = "tab"
	InteractionModeWorkspace InteractionMode = "workspace"
	InteractionModeCopy      InteractionMode = "copy"
)

type ShellAction string

const (
	ShellActionToggleHeader         ShellAction = "shell.toggle-header"
	ShellActionToggleFooter         ShellAction = "shell.toggle-footer"
	ShellActionClearToasts          ShellAction = "shell.clear-toasts"
	ShellActionCloseToast           ShellAction = "shell.close-toast"
	ShellActionFloatingCtrl         ShellAction = "shell.floating-control"
	ShellActionFloatingNew          ShellAction = "shell.floating-new"
	ShellActionFloatingOverview     ShellAction = "shell.floating-overview"
	ShellActionFloatingSummon       ShellAction = "shell.floating-summon"
	ShellActionFloatingMove         ShellAction = "shell.floating-move"
	ShellActionFloatingSize         ShellAction = "shell.floating-size"
	ShellActionFloatingGroup        ShellAction = "shell.floating-group"
	ShellActionOpenPool             ShellAction = "shell.open-terminal-pool"
	ShellActionOpenConnections      ShellAction = "shell.open-connections"
	ShellActionOpenTree             ShellAction = "shell.open-workbench-tree"
	ShellActionOpenPicker           ShellAction = "shell.open-terminal-picker"
	ShellActionOpenPrompt           ShellAction = "shell.open-prompt"
	ShellActionOpenHelp             ShellAction = "shell.open-help"
	ShellActionOpenClipboardHistory ShellAction = "shell.open-clipboard-history"
	ShellActionToggleShortcutLock   ShellAction = "shell.toggle-shortcut-lock"
	ShellActionQuit                 ShellAction = "shell.quit"
)

// Intent 是 input router 输出的 semantic intent，不直接修改 state。
type Intent struct {
	Kind     IntentKind
	Event    InputEvent
	Bytes    []byte
	Reason   string
	Command  string
	Mode     InteractionMode
	Action   ShellAction
	RawMouse bool
	// Invocation 只由 shortcut catalog 命中产生；app dispatcher 是其执行语义 owner。
	Invocation actiondomain.Invocation
}

// RouteOptions 是 input router 的只读上下文。
// Shortcuts 来自 reducer-owned config 快照；router 只消费已经解析过的配置，不读取文件也不修改 state。
type RouteOptions struct {
	Mode                     InteractionMode
	CopyModeActive           bool
	TextInputActive          bool
	TerminalMousePassthrough bool
	ForceTerminalPassthrough bool
	Shortcuts                state.TUIShortcutConfig
}

func Route(event InputEvent, copyModeActive bool) Intent {
	return RouteWithOptions(event, RouteOptions{CopyModeActive: copyModeActive})
}

func RouteWithMode(event InputEvent, copyModeActive bool, mode InteractionMode) Intent {
	return RouteWithOptions(event, RouteOptions{Mode: mode, CopyModeActive: copyModeActive})
}

func RouteWithOptions(event InputEvent, options RouteOptions) Intent {
	switch event.Kind {
	case EventKindKey:
		return routeKey(event, options)
	case EventKindPaste:
		if options.TextInputActive {
			return Intent{Kind: IntentNone, Event: event, Reason: "text input paste"}
		}
		return Intent{Kind: IntentTerminalInput, Event: event, Bytes: []byte(event.Paste), Reason: "structured paste body"}
	case EventKindMouse:
		return routeMouse(event, options)
	case EventKindHostTheme, EventKindHostCapability, EventKindHostControl:
		return Intent{Kind: IntentNone, Event: event, Reason: "host control event"}
	default:
		return Intent{Kind: IntentNone, Event: event, Reason: "unknown input kind"}
	}
}
