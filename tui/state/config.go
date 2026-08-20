package state

// TUIConfigStore 是当前配置文件的已验证快照。Daemon 保存服务端物理策略；
// 其余字段仍只保存当前客户端视觉、交互和 shortcuts 偏好。
type TUIConfigStore struct {
	Version     int
	Daemon      DaemonConfig
	Profile     string
	Theme       TUIThemeConfig
	Chrome      TUIChromeConfig
	Footer      TUIFooterConfig
	Interaction TUIInteractionConfig
	Terminal    TUITerminalConfig
	Shortcuts   TUIShortcutConfig
}

type DaemonConfig struct {
	History      DaemonHistoryConfig
	OutputBuffer DaemonOutputBufferConfig
}

type DaemonOutputBufferConfig struct {
	CapacityBytes       int64
	Overflow            string
	ResidentBudgetBytes int64
}

// DaemonHistoryConfig 只控制 history 的物理存储，不改变 terminal/history
// semantic truth。MaxSizeMB=0、MaxAgeDays=0 分别表示关闭对应限制。
type DaemonHistoryConfig struct {
	MaxSizeMB        int
	MaxAgeDays       int
	Compression      string
	CompressionLevel string
}

type TUIThemeConfig struct {
	Mode       string
	Palette    string
	Primary    string
	Secondary  string
	Foreground string
	Background string
	Muted      string
	Success    string
	Warning    string
	Danger     string
	Info       string
	Border     TUIThemeBorderConfig
	Surface    TUIThemeSurfaceConfig
}

type TUIThemeBorderConfig struct {
	Panel    string
	Active   string
	Inactive string
	Muted    string
}

type TUIThemeSurfaceConfig struct {
	ChromeBG  string
	StatusBG  string
	OverlayBG string
	ToastBG   string
}

// TUIChromeConfig 描述 TUI 外层 chrome 的本地展示偏好。
// 它只控制 header/footer、pane 呈现和可见 action 字形；pane/tab/workspace 的真实状态仍由 ShellStore 持有。
type TUIChromeConfig struct {
	Header            bool
	Footer            bool
	PanelPresentation string
	TabCreateIcon     string
	// TabCreateTemplate 是顶部创建 tab 按钮的展示模板。
	// 它只改变 tab.create action 的可见片段和 hit region 样式，真实新建 tab 消息仍由 app reducer 处理。
	TabCreateTemplate string
	// WorkspaceTemplate 是顶部 workspace 槽位的展示模板。
	// 它只消费当前 ShellStore workspace 名称并绑定打开 Workbench Navigator 的既有 action，
	// 不保存 workspace truth，也不能改写 workspace/tab/pane 的 reducer-owned 状态。
	WorkspaceTemplate string
	TabTemplate       string
	// PaneTitleTemplate 是 pane/floating terminal chrome 标题模板。
	// 它只消费 reducer-owned terminal pool 与 endpoint 展示投影，不改变 pane title、terminal identity 或输入路由。
	PaneTitleTemplate string
	PaneGlyphs        TUIPaneChromeGlyphsConfig
}

// TUIPaneChromeGlyphsConfig 描述 pane/floating chrome 可点击动作、状态标记和左右装饰模板。
// 这些字段只改变 renderer 展示文字；真实 action id、鼠标 hit region、resize owner
// 消息链路仍由 reducer/app 持有，配置不能改写这些交互语义。
type TUIPaneChromeGlyphsConfig struct {
	ActionLeft          string
	ActionLeftSet       bool
	ActionRight         string
	ActionRightSet      bool
	ActionSeparator     string
	ActionSeparatorSet  bool
	ActionGroupLeft     string
	ActionGroupLeftSet  bool
	ActionGroupRight    string
	ActionGroupRightSet bool
	OwnerLeft           string
	OwnerLeftSet        bool
	OwnerRight          string
	OwnerRightSet       bool
	Owner               string
	OwnerSet            bool
	OwnerPending        string
	OwnerPendingSet     bool
	TakeOwner           string
	TakeOwnerSet        bool
	Zoom                string
	// Unzoom 是 zoom 状态下 pane.zoom toggle action 的展示 glyph；配置只影响 renderer。
	Unzoom           string
	SplitVertical    string
	SplitHorizontal  string
	Close            string
	SizeLock         string
	SizeUnlock       string
	CenterFloating   string
	CollapseFloating string
	Running          string
	Waiting          string
	Exited           string
	Killed           string
	// Overflow* 只配置内容裁切提示 glyph；是否裁切仍由 renderer ContentOverflow 计算。
	OverflowLeft      string
	OverflowLeftSet   bool
	OverflowRight     string
	OverflowRightSet  bool
	OverflowTop       string
	OverflowTopSet    bool
	OverflowBottom    string
	OverflowBottomSet bool
	// OverflowStyle 可写内置 style token 或 #RRGGBB，作用域只限 pane/floating chrome marker。
	OverflowStyle    string
	OverflowStyleSet bool
	// ExtentPlaceholder* 配置 live surface 尺寸差异区域的占位点展示，不改变 terminal 内容。
	ExtentPlaceholder         string
	ExtentPlaceholderSet      bool
	ExtentPlaceholderStyle    string
	ExtentPlaceholderStyleSet bool
}

// TUIFooterConfig 描述 footer 的用户可配置展示层。
// 它只影响 mode badge、模板和 summary；快捷键 action token 统一由 TUIShortcutConfig 派生。
type TUIFooterConfig struct {
	Templates TUIFooterTemplatesConfig
	Modes     map[string]TUIFooterModeConfig
}

// TUIFooterTemplatesConfig 描述 footer token 的轻量模板。
// 当前只支持文本占位符替换，不执行脚本，也不改变 hit region/action owner。
type TUIFooterTemplatesConfig struct {
	ModeBadge                string
	Key                      string
	Action                   string
	Separator                string
	WorkspaceSummary         string
	FloatingSummary          string
	FloatingCollapsedSummary string
	TerminalsSummary         string
	TabsSummary              string
	PanesSummary             string
	KeylockOn                string
	KeylockOff               string
}

// TUIFooterModeConfig 描述某个 footer mode 的徽标。
// action token 不在 footer 配置中保存，避免和 shortcut catalog 形成第二份快捷键真值。
type TUIFooterModeConfig struct {
	Icon  string
	Label string
	Style string
}

type TUIInteractionConfig struct {
	Mouse                         bool
	StickyPrefixTimeoutMS         int
	ShortcutPassthroughIntervalMS int
	ConfirmDestructive            bool
	ClipboardHistory              TUIClipboardHistoryConfig
	Picker                        TUIPickerConfig
}

type TUIClipboardHistoryConfig struct {
	MaxItems          int
	NameWidth         int
	PreviewWidthRatio float64
}

type TUIPickerConfig struct {
	FuzzyMatch       string
	HighlightMatches bool
}

// TUITerminalConfig 描述 TUI 对 terminal resize ownership 的本地策略。
// AutoTakeOwner 只影响 TUI 是否在 attach/布局变化时主动请求 owner；daemon 仍是最终仲裁者。
type TUITerminalConfig struct {
	AutoTakeOwner bool
}

// TUIShortcutConfig 是当前 TUI 客户端快捷键配置的唯一入口。
// Actions 只声明 action 默认展示文案；Scenes 按场景保存 key -> action 绑定。
// 真实执行仍由后续 shortcut catalog / action registry 转成 reducer 消息，配置层不持有运行时状态。
// Configured 只记录 shortcuts 配置参与过解析，供配置诊断使用；它不决定 binding catalog。
// Actions 只覆盖默认文案；Scenes 只要包含任意显式 scene，就声明完整用户 binding catalog，空 scene 也不回退默认。
type TUIShortcutConfig struct {
	Configured bool
	Actions    map[string]TUIShortcutActionConfig
	Scenes     map[string]TUIShortcutSceneConfig
}

// TUIShortcutActionConfig 描述一个 action 的默认展示文案与 footer style。
// action id 是后续输入路由、footer/help 展示和 action registry 的共享语义键；展示配置不能改写 action 的执行链路。
type TUIShortcutActionConfig struct {
	Label string
	Style string
}

// TUIShortcutSceneConfig 描述某个输入场景内的按键绑定集合。
// 同一场景内 key 必须唯一；是否进入该场景由 global 里的 menu.<scene> action 或运行时 overlay 状态决定。
// 未修饰 Esc 是运行时保留的全局返回键，不属于任何 scene，也不能在这里覆盖。
type TUIShortcutSceneConfig struct {
	Bindings map[string]TUIShortcutBindingConfig
	KeyOrder []string
}

// TUIShortcutBindingConfig 描述一个按键到 action 的绑定。
// Action 是必填语义目标；Label 只覆盖该场景下的展示文案，不能产生新的执行语义。
// Show 是 footer 展示意图：nil 沿用 action domain 默认值，false 只隐藏 footer 提示但不移除键盘路由或 Help 条目。
type TUIShortcutBindingConfig struct {
	Action string
	Label  string
	Show   *bool
}
