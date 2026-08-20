package state

import "time"

const (
	DefaultWorkspaceID = "workspace-main"
	DefaultTabID       = "tab-main"
	DefaultPaneID      = "pane-main"
)

type PanelPresentation string

const (
	PanelPresentationCard      PanelPresentation = "card"
	PanelPresentationSplitLine PanelPresentation = "split-line"
)

type PaneKind string

const (
	// 中文说明：pane 只表达连接槽位；copy/exited 由 CopyModeStore 和 terminal lifecycle 投影。
	PaneEmpty        PaneKind = "empty"
	PaneTerminalLive PaneKind = "terminal-live"
)

type SplitDirection string

const (
	SplitDirectionHorizontal SplitDirection = "horizontal"
	SplitDirectionVertical   SplitDirection = "vertical"
)

type OverlayKind string

const (
	OverlayNone             OverlayKind = ""
	OverlayTerminalPicker   OverlayKind = "terminal-picker"
	OverlayTerminalPool     OverlayKind = "terminal-pool"
	OverlayConnections      OverlayKind = "connections"
	OverlayWorkbenchTree    OverlayKind = "workbench-tree"
	OverlayClipboardHistory OverlayKind = "clipboard-history"
	OverlayFloatingOverview OverlayKind = "floating-overview"
	OverlayPrompt           OverlayKind = "prompt"
	OverlayHelp             OverlayKind = "help"
)

type ToastSeverity string

const (
	ToastInfo    ToastSeverity = "info"
	ToastSuccess ToastSeverity = "success"
	ToastWarning ToastSeverity = "warning"
	ToastError   ToastSeverity = "error"
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
)

type WorkbenchCommandAction string

const (
	WorkbenchCommandTabCreate         WorkbenchCommandAction = "tab.create"
	WorkbenchCommandTabSwitch         WorkbenchCommandAction = "tab.switch"
	WorkbenchCommandTabNext           WorkbenchCommandAction = "tab.next"
	WorkbenchCommandTabPrevious       WorkbenchCommandAction = "tab.previous"
	WorkbenchCommandTabRename         WorkbenchCommandAction = "tab.rename"
	WorkbenchCommandTabClose          WorkbenchCommandAction = "tab.close"
	WorkbenchCommandTabKill           WorkbenchCommandAction = "tab.kill"
	WorkbenchCommandWorkspaceCreate   WorkbenchCommandAction = "workspace.create"
	WorkbenchCommandWorkspaceSwitch   WorkbenchCommandAction = "workspace.switch"
	WorkbenchCommandWorkspaceNext     WorkbenchCommandAction = "workspace.next"
	WorkbenchCommandWorkspacePrevious WorkbenchCommandAction = "workspace.previous"
	WorkbenchCommandWorkspaceRename   WorkbenchCommandAction = "workspace.rename"
	WorkbenchCommandWorkspaceDelete   WorkbenchCommandAction = "workspace.delete"
	WorkbenchCommandPaneSplit         WorkbenchCommandAction = "pane.split"
	WorkbenchCommandPaneRename        WorkbenchCommandAction = "pane.rename"
	WorkbenchCommandPaneDetach        WorkbenchCommandAction = "pane.detach"
	WorkbenchCommandPaneClose         WorkbenchCommandAction = "pane.close"
	WorkbenchCommandPaneKill          WorkbenchCommandAction = "pane.kill"
)

type WorkbenchCommand struct {
	Action   WorkbenchCommandAction
	TargetID string
	Target   PaneCommandTarget
	Name     string
	Pane     PaneCommand
	Source   PaneCommandSource
	Confirm  PaneConfirmPolicy
}

type WorkbenchCommandStatus string

const (
	WorkbenchCommandOK                WorkbenchCommandStatus = "ok"
	WorkbenchCommandNeedsConfirmation WorkbenchCommandStatus = "needs-confirmation"
	WorkbenchCommandInvalid           WorkbenchCommandStatus = "invalid"
)

type WorkbenchCommandResult struct {
	Status WorkbenchCommandStatus
	Action WorkbenchCommandAction
	Reason string
	ID     string
	Killed []string
}

// ShellStore 保存 Workbench 外壳相关的 reducer-owned 产品状态。
// 它只描述用户可操作的结构，不计算最终屏幕矩形，也不画 panel chrome。
type ShellStore struct {
	Workspace          WorkspaceState
	Workspaces         []WorkspaceState
	PanelPresentation  PanelPresentation
	ActivePaneID       string
	ZoomedPaneID       string
	InteractionMode    InteractionMode
	InteractionModeSeq uint64
	// ShortcutPassthroughLocked 表示 root shortcut 是否临时让路给前台 terminal。
	// 它属于本 TUI 输入路由状态，不写入 workbench storage；global mode 仍作为解锁控制面。
	ShortcutPassthroughLocked bool
	OwnerConfirm              OwnerConfirmState
	HeaderVisible             bool
	FooterVisible             bool
	Overlay                   OverlayState
	EmptyPaneCTA              EmptyPaneCTAState
	ExitedPaneCTA             ExitedPaneCTAState
	TerminalCreateDraft       TerminalCreateDraft
	Toasts                    []ToastState
	nextToastSeq              uint64
	nextFloatingSeq           uint64
	initialized               bool
	forceTerminalInput        bool
	shortcutPassthroughKind   string
	shortcutPassthroughSeq    uint64
}

type WorkspaceState struct {
	ID          string
	Name        string
	Tabs        []TabState
	ActiveTabID string
}

type TabState struct {
	ID               string
	Title            string
	Panes            []PaneState
	ActivePaneID     string
	RootSplit        SplitNode
	Floatings        []FloatingPaneState
	ActiveFloatingID string
}

type PaneState struct {
	ID         string
	Title      string
	Kind       PaneKind
	TerminalID string
	Active     bool
}

type FloatingPaneState struct {
	ID        string
	Title     string
	Pane      PaneState
	Rect      FloatingRect
	Z         int
	Active    bool
	Collapsed bool
	FitMode   FloatingFitMode
	AutoFit   FloatingAutoFitState
}

type FloatingRect struct {
	X int
	Y int
	W int
	H int
}

type FloatingFitMode string

const (
	FloatingFitManual FloatingFitMode = "manual"
	FloatingFitAuto   FloatingFitMode = "auto"
)

type FloatingAutoFitState struct {
	Cols int
	Rows int
}

type EmptyPaneCTAState struct {
	SelectedIndex int
}

// TerminalCreateDraft 保存上一次成功创建 terminal 时的表单默认值。
// 该状态属于 TUI reducer 交互记忆；EndpointID/Command/Workdir 只用于下一次打开 create prompt，
// 不代表 terminal lifecycle truth，也不替代 owning daemon 返回的 terminal list。
type TerminalCreateDraft struct {
	EndpointID EndpointID
	Command    string
	Workdir    string
}

type SplitNode struct {
	PaneID    string
	Direction SplitDirection
	Children  []SplitNode
	// 几何 hint 属于 reducer-owned pane tree；renderer 只消费投影后的 SplitVM，不反写 state。
	Ratio       float64
	BiasCells   int
	FixedPaneID string
	FixedCols   int
	FixedRows   int
}

type OverlayState struct {
	Kind          OverlayKind
	Open          bool
	TargetID      string
	Query         string
	SelectedIndex int
	// WorkbenchCollapsed 保存 Workbench Navigator 当前 overlay 的折叠节点。
	// 它只影响树投影可见性，不改变 workspace/tab/pane 的真实布局状态。
	WorkbenchCollapsed map[string]bool
	Prompt             PromptState
	HelpSection        string
	ClipboardNameWidth int
}

type PromptState struct {
	Title              string
	Context            string
	TargetEndpointID   EndpointID
	TargetID           string
	TargetTabID        string
	TargetWorkspaceID  string
	Purpose            string
	Value              string
	Placeholder        string
	Destructive        bool
	ConfirmText        string
	Submitted          bool
	Canceled           bool
	LastResult         string
	Fields             []PromptFieldState
	ActiveField        int
	SuggestionFocused  bool
	SuggestionSelected int
	SuggestionOffset   int
	Command            []string
	Workdir            string
	Tags               map[string]string
	DefaultName        string
}

type PromptFieldState struct {
	Key             string
	Label           string
	Value           string
	Cursor          int
	Placeholder     string
	Required        bool
	SuggestionTitle string
	SuggestionItems []string
	SuggestionEmpty string
}

type WorkbenchTreeItem struct {
	Kind              string
	WorkspaceID       string
	WorkspaceName     string
	TabID             string
	TabTitle          string
	FloatingID        string
	FloatingTitle     string
	PaneID            string
	PaneTitle         string
	DisplayTitle      string
	PaneKind          PaneKind
	EndpointID        EndpointID
	EndpointLabel     string
	EndpointStatus    EndpointStatusKind
	EndpointLastError string
	EndpointErrorKind EndpointErrorKind
	TerminalID        string
	LastOutputAt      time.Time
	Depth             int
	Expandable        bool
	Collapsed         bool
	Active            bool
	Selected          bool
	Summary           string
}

const (
	WorkbenchTreeKindWorkspace = "workspace"
	WorkbenchTreeKindTab       = "tab"
	WorkbenchTreeKindPane      = "pane"
	WorkbenchTreeKindFloating  = "floating"
)

type ToastState struct {
	ID                string
	Severity          ToastSeverity
	Title             string
	Body              string
	Pending           bool
	AgeTicks          uint64
	DismissAfterTicks uint64
}

type ToastSpec struct {
	ID                string
	Severity          ToastSeverity
	Title             string
	Body              string
	Pending           bool
	DismissAfterTicks uint64
}

const (
	defaultToastDismissTicks   uint64 = 3
	attentionToastDismissTicks uint64 = 6
	pendingToastDismissTicks   uint64 = 8
)

func DefaultShell() ShellStore {
	return (ShellStore{}).EnsureDefaults()
}

func (store ShellStore) EnsureDefaults() ShellStore {
	store.Workspace = cloneWorkspace(store.Workspace)
	store.Workspaces = cloneWorkspaces(store.Workspaces)
	store.Toasts = cloneToasts(store.Toasts)
	seedDefaultWorkbench := !store.initialized && len(store.Workspace.Tabs) == 0
	if !store.initialized {
		store.HeaderVisible = true
		store.FooterVisible = true
		store.initialized = true
	}
	if store.Workspace.ID == "" {
		store.Workspace.ID = DefaultWorkspaceID
	}
	if store.Workspace.Name == "" {
		store.Workspace.Name = "main"
	}
	if seedDefaultWorkbench {
		store.Workspace.ActiveTabID = DefaultTabID
		store.Workspace.Tabs = []TabState{defaultTabState()}
	}
	store.Workspace = store.Workspace.ensureDefaults()
	store.Workspaces = upsertWorkspace(ensureWorkspaceList(store.Workspaces, store.Workspace), store.Workspace)
	store.Workspace = store.Workspace.ensureTabDefaults()
	store.Workspace = store.Workspace.ensureActiveTab()
	if len(store.Workspace.Tabs) == 0 {
		store.Workspace.ActiveTabID = ""
		store.ActivePaneID = ""
		store.ZoomedPaneID = ""
	} else if store.ActivePaneID == "" {
		store.ActivePaneID = store.activeTab().ActivePaneID
	}
	if store.PanelPresentation == "" {
		store.PanelPresentation = PanelPresentationCard
	}
	if store.ZoomedPaneID != "" && !store.hasPaneInActiveTab(store.ZoomedPaneID) {
		store.ZoomedPaneID = ""
	}
	store = store.ensureFloatingDefaults()
	store.Workspace = store.Workspace.ensureActive(store.ActivePaneID)
	store.Workspaces = upsertWorkspace(store.Workspaces, store.Workspace)
	return store
}

func (store ShellStore) SetPanelPresentation(presentation PanelPresentation) ShellStore {
	if presentation != PanelPresentationCard && presentation != PanelPresentationSplitLine {
		return store
	}
	store.PanelPresentation = presentation
	return store.EnsureDefaults()
}

func (store ShellStore) TogglePanelPresentation() ShellStore {
	store = store.EnsureDefaults()
	if store.PanelPresentation == PanelPresentationSplitLine {
		store.PanelPresentation = PanelPresentationCard
	} else {
		store.PanelPresentation = PanelPresentationSplitLine
	}
	return store
}

func (store ShellStore) SetHeaderVisible(visible bool) ShellStore {
	store = store.EnsureDefaults()
	store.HeaderVisible = visible
	return store
}

func (store ShellStore) ToggleHeaderVisible() ShellStore {
	store = store.EnsureDefaults()
	store.HeaderVisible = !store.HeaderVisible
	return store
}

func (store ShellStore) SetFooterVisible(visible bool) ShellStore {
	store = store.EnsureDefaults()
	store.FooterVisible = visible
	return store
}

func (store ShellStore) ToggleFooterVisible() ShellStore {
	store = store.EnsureDefaults()
	store.FooterVisible = !store.FooterVisible
	return store
}

func (store ShellStore) MoveEmptyPaneCTASelection(delta int, count int) ShellStore {
	store = store.EnsureDefaults()
	if count <= 0 {
		store.EmptyPaneCTA.SelectedIndex = 0
		return store
	}
	selected := store.EmptyPaneCTA.SelectedIndex + delta
	for selected < 0 {
		selected += count
	}
	store.EmptyPaneCTA.SelectedIndex = selected % count
	return store
}

func (store ShellStore) SetEmptyPaneCTASelection(index int, count int) ShellStore {
	store = store.EnsureDefaults()
	if count <= 0 || index < 0 {
		store.EmptyPaneCTA.SelectedIndex = 0
		return store
	}
	if index >= count {
		index = count - 1
	}
	store.EmptyPaneCTA.SelectedIndex = index
	return store
}

func (store ShellStore) SetInteractionMode(mode InteractionMode) ShellStore {
	store = store.EnsureDefaults()
	switch mode {
	case InteractionModeNormal, InteractionModePane, InteractionModeResize, InteractionModeGlobal, InteractionModeFloating, InteractionModeTab, InteractionModeWorkspace:
		if store.InteractionMode != mode || stickyInteractionMode(mode) {
			store.InteractionModeSeq++
		}
		store.InteractionMode = mode
	}
	return store
}

func (store ShellStore) RearmInteractionMode() ShellStore {
	store = store.EnsureDefaults()
	if stickyInteractionMode(store.InteractionMode) {
		store.InteractionModeSeq++
	}
	return store
}

func (store ShellStore) StickyInteractionMode() bool {
	return stickyInteractionMode(store.EnsureDefaults().InteractionMode)
}

// ToggleShortcutPassthroughLock 切换 root shortcut 透传锁。
// domain owner 是 ShellStore；调用方只能改变 TUI 输入路由，不改变 terminal lifecycle 或 workbench storage。
func (store ShellStore) ToggleShortcutPassthroughLock() ShellStore {
	store = store.EnsureDefaults()
	store.ShortcutPassthroughLocked = !store.ShortcutPassthroughLocked
	return store
}

// ArmTerminalInputPassthroughOnce 标记当前 InputMsg 必须由 terminal reducer 强制透传一次。
// 这个标记只解决同一消息链路内的 UI shortcut 让路，消费方必须立刻清掉，避免后续按键串台。
func (store ShellStore) ArmTerminalInputPassthroughOnce() ShellStore {
	store = store.EnsureDefaults()
	store.forceTerminalInput = true
	return store
}

// ConsumeTerminalInputPassthroughOnce 消费一次强制 terminal 透传标记。
// 返回 false 表示当前消息仍应按普通 UI/terminal 路由判断。
func (store ShellStore) ConsumeTerminalInputPassthroughOnce() (ShellStore, bool) {
	store = store.EnsureDefaults()
	forced := store.forceTerminalInput
	store.forceTerminalInput = false
	return store, forced
}

func stickyInteractionMode(mode InteractionMode) bool {
	switch mode {
	case InteractionModePane, InteractionModeResize, InteractionModeGlobal, InteractionModeFloating, InteractionModeTab, InteractionModeWorkspace:
		return true
	default:
		return false
	}
}
