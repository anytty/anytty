package state

type PaneCommandAction string

const (
	PaneCommandSplit              PaneCommandAction = "pane.split"
	PaneCommandClose              PaneCommandAction = "pane.close"
	PaneCommandKill               PaneCommandAction = "pane.kill"
	PaneCommandCloseAndKill       PaneCommandAction = "pane.close-and-kill"
	PaneCommandFocus              PaneCommandAction = "pane.focus"
	PaneCommandFocusNext          PaneCommandAction = "pane.focus-next"
	PaneCommandFocusPrevious      PaneCommandAction = "pane.focus-previous"
	PaneCommandZoom               PaneCommandAction = "pane.zoom"
	PaneCommandUnzoom             PaneCommandAction = "pane.unzoom"
	PaneCommandToggleZoom         PaneCommandAction = "pane.toggle-zoom"
	PaneCommandResize             PaneCommandAction = "pane.resize"
	PaneCommandSetSize            PaneCommandAction = "pane.set-size"
	PaneCommandBalance            PaneCommandAction = "pane.balance"
	PaneCommandSetPresentation    PaneCommandAction = "pane.set-presentation"
	PaneCommandTogglePresentation PaneCommandAction = "pane.toggle-presentation"
)

type PaneCommandTarget struct {
	WorkspaceID string
	TabID       string
	PaneID      string
}

type PaneResizeDirection string

const (
	PaneResizeLeft  PaneResizeDirection = "left"
	PaneResizeRight PaneResizeDirection = "right"
	PaneResizeUp    PaneResizeDirection = "up"
	PaneResizeDown  PaneResizeDirection = "down"
)

const PaneResizeRootSplitPath = "root"

type PaneSizeMode string

const (
	PaneSizeRatio PaneSizeMode = "ratio"
	PaneSizeCells PaneSizeMode = "cells"
)

type PaneResizeGroupItem struct {
	PaneID    string
	Cells     int
	DeltaSign int
}

type PaneCommandSource string

const (
	PaneCommandSourceKeyboard PaneCommandSource = "keyboard"
	PaneCommandSourceMouse    PaneCommandSource = "mouse"
	PaneCommandSourceTest     PaneCommandSource = "test"
	PaneCommandSourceCLIMini  PaneCommandSource = "cli-mini"
	PaneCommandSourcePalette  PaneCommandSource = "palette"
)

type PaneConfirmPolicy string

const (
	PaneConfirmDefault  PaneConfirmPolicy = ""
	PaneConfirmRequired PaneConfirmPolicy = "required"
	PaneConfirmAccepted PaneConfirmPolicy = "accepted"
)

// PaneCommand 是 pane 结构操作的唯一业务契约。
// 快捷键、鼠标、测试入口和后续 CLI mini command 都只能适配到这个结构。
type PaneCommand struct {
	Action          PaneCommandAction
	Target          PaneCommandTarget
	SplitDirection  SplitDirection
	ResizeDirection PaneResizeDirection
	// Delta 对键盘/CLI 是正向步长；鼠标拖拽固定某条边时可为负，表示该边向反方向移动。
	Delta int
	// ResizeSplitPath 只用于鼠标拖拽真实 divider，避免按 pane id 向上误改外层 split。
	ResizeSplitPath string
	// ResizeGroupCells 是鼠标 divider 在同轴 pane 链中的叶子目标尺寸。
	// DeltaSign 表达该 pane 跟随 divider 哪一侧变化，用于 stacked pane 共享宽/高场景。
	ResizeGroupCells []PaneResizeGroupItem
	SizeMode         PaneSizeMode
	Ratio            float64
	Cols             int
	Rows             int
	Presentation     PanelPresentation
	NewPane          PaneState
	Source           PaneCommandSource
	Confirm          PaneConfirmPolicy
}

type PaneCommandStatus string

const (
	PaneCommandOK                PaneCommandStatus = "ok"
	PaneCommandNeedsConfirmation PaneCommandStatus = "needs-confirmation"
	PaneCommandInvalid           PaneCommandStatus = "invalid"
)

type PaneCommandResult struct {
	Status PaneCommandStatus
	Action PaneCommandAction
	Reason string
}

// WithDefaults 只填充默认目标，不改变命令动作语义。
func (command PaneCommand) WithDefaults(shell ShellStore) PaneCommand {
	shell = shell.EnsureDefaults()
	if command.Target.WorkspaceID == "" {
		command.Target.WorkspaceID = shell.Workspace.ID
	}
	activeTab := shell.activeTab()
	if command.Target.TabID == "" {
		command.Target.TabID = activeTab.ID
	}
	if command.Target.PaneID == "" {
		command.Target.PaneID = shell.ActivePaneID
	}
	return command
}

// Validate 只验证命令契约是否成立，不直接修改 shell state。
func (command PaneCommand) Validate(shell ShellStore) PaneCommandResult {
	command = command.WithDefaults(shell)
	if command.Action == "" {
		return paneCommandInvalid(command.Action, "missing action")
	}
	if !shell.HasPane(command.Target) {
		return paneCommandInvalid(command.Action, "target pane not found")
	}
	switch command.Action {
	case PaneCommandSplit:
		if command.SplitDirection != SplitDirectionHorizontal && command.SplitDirection != SplitDirectionVertical {
			return paneCommandInvalid(command.Action, "invalid split direction")
		}
		if command.NewPane.ID == "" {
			return paneCommandInvalid(command.Action, "missing new pane id")
		}
		if shell.HasPane(PaneCommandTarget{WorkspaceID: command.Target.WorkspaceID, TabID: command.Target.TabID, PaneID: command.NewPane.ID}) {
			return paneCommandInvalid(command.Action, "new pane already exists")
		}
	case PaneCommandClose:
		if shell.paneCountForTarget(command.Target) <= 1 {
			return paneCommandInvalid(command.Action, "cannot close last pane")
		}
	case PaneCommandKill:
		if command.Confirm != PaneConfirmAccepted {
			return PaneCommandResult{Status: PaneCommandNeedsConfirmation, Action: command.Action, Reason: "confirmation required"}
		}
	case PaneCommandCloseAndKill:
		if command.Confirm != PaneConfirmAccepted {
			return PaneCommandResult{Status: PaneCommandNeedsConfirmation, Action: command.Action, Reason: "confirmation required"}
		}
	case PaneCommandResize:
		if command.ResizeDirection != PaneResizeLeft && command.ResizeDirection != PaneResizeRight && command.ResizeDirection != PaneResizeUp && command.ResizeDirection != PaneResizeDown {
			return paneCommandInvalid(command.Action, "invalid resize direction")
		}
		if command.Delta == 0 {
			return paneCommandInvalid(command.Action, "missing resize delta")
		}
	case PaneCommandSetSize:
		switch command.SizeMode {
		case PaneSizeRatio:
			if command.Ratio <= 0 || command.Ratio > 1 {
				return paneCommandInvalid(command.Action, "invalid size ratio")
			}
		case PaneSizeCells:
			if command.Cols <= 0 && command.Rows <= 0 {
				return paneCommandInvalid(command.Action, "missing fixed cell size")
			}
		default:
			return paneCommandInvalid(command.Action, "invalid size mode")
		}
	case PaneCommandSetPresentation:
		if command.Presentation != PanelPresentationCard && command.Presentation != PanelPresentationSplitLine {
			return paneCommandInvalid(command.Action, "invalid panel presentation")
		}
	case PaneCommandFocus, PaneCommandFocusNext, PaneCommandFocusPrevious, PaneCommandZoom, PaneCommandUnzoom, PaneCommandToggleZoom, PaneCommandBalance, PaneCommandTogglePresentation:
	default:
		return paneCommandInvalid(command.Action, "unknown action")
	}
	return PaneCommandResult{Status: PaneCommandOK, Action: command.Action}
}

func (store ShellStore) ApplyPaneCommand(command PaneCommand) (ShellStore, PaneCommandResult) {
	store = store.EnsureDefaults()
	if command.Target.WorkspaceID != "" && command.Target.WorkspaceID != store.Workspace.ID {
		// Pane command 也可能来自 Workbench Navigator，先切到目标 workspace 再补默认 tab/pane。
		next, result := store.switchWorkspace(command.Target.WorkspaceID, WorkbenchCommandAction(command.Action))
		if result.Status != WorkbenchCommandOK {
			return store, paneCommandInvalid(command.Action, "workspace not found")
		}
		store = next
	}
	command = command.WithDefaults(store)
	result := command.Validate(store)
	if result.Status != PaneCommandOK {
		return store.EnsureDefaults(), result
	}
	switch command.Action {
	case PaneCommandSplit:
		// 鼠标命中区会携带 target pane；先聚焦 target，保证 split 的结构落在被点击的 pane 上。
		return store.FocusPane(command.Target).SplitActivePane(command.NewPane, command.SplitDirection), result
	case PaneCommandClose:
		return store.ClosePane(command.Target), result
	case PaneCommandKill, PaneCommandCloseAndKill:
		return store, result
	case PaneCommandFocus:
		return store.FocusPane(command.Target), result
	case PaneCommandFocusNext:
		return store.FocusRelativePane(1), result
	case PaneCommandFocusPrevious:
		return store.FocusRelativePane(-1), result
	case PaneCommandZoom:
		return store.ZoomPane(command.Target), result
	case PaneCommandUnzoom:
		return store.UnzoomPane(), result
	case PaneCommandToggleZoom:
		return store.ToggleZoomPane(command.Target), result
	case PaneCommandResize:
		if len(command.ResizeGroupCells) > 0 {
			return store.ResizePaneGroup(command.Target, command.ResizeDirection, command.ResizeGroupCells), result
		}
		if command.ResizeSplitPath != "" {
			return store.ResizeSplitPath(command.Target, command.ResizeSplitPath, command.ResizeDirection, command.Delta), result
		}
		return store.ResizePane(command.Target, command.ResizeDirection, command.Delta), result
	case PaneCommandSetSize:
		return store.SetPaneSize(command), result
	case PaneCommandBalance:
		return store.BalancePanes(command.Target), result
	case PaneCommandSetPresentation:
		return store.SetPanelPresentation(command.Presentation), result
	case PaneCommandTogglePresentation:
		return store.TogglePanelPresentation(), result
	default:
		return store.EnsureDefaults(), result
	}
}

func (store ShellStore) HasPane(target PaneCommandTarget) bool {
	_, ok := store.Pane(target)
	return ok
}

func (store ShellStore) Pane(target PaneCommandTarget) (PaneState, bool) {
	store = store.ReadonlyDefaults()
	if target.WorkspaceID != "" && target.WorkspaceID != store.Workspace.ID {
		return PaneState{}, false
	}
	for _, tab := range store.Workspace.Tabs {
		if target.TabID != "" && target.TabID != tab.ID {
			continue
		}
		for _, pane := range tab.Panes {
			if pane.ID == target.PaneID {
				return pane, true
			}
		}
	}
	return PaneState{}, false
}

func paneCommandInvalid(action PaneCommandAction, reason string) PaneCommandResult {
	return PaneCommandResult{Status: PaneCommandInvalid, Action: action, Reason: reason}
}
