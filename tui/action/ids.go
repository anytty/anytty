package action

// 下列 ID 是没有默认键位、但由 mouse/drag/CTA 触发的 canonical action。
// 视觉投影名称不在这里注册；同一业务动作的 footer、chrome 与 overlay 投影必须引用同一个 ID。
const (
	ActionPanelFocus                  ID = "panel.focus"
	ActionPanelResizeDrag             ID = "panel.resize_drag"
	ActionTabSelect                   ID = "tab.select"
	ActionFloatingRaise               ID = "floating.raise"
	ActionFloatingResize              ID = "floating.resize"
	ActionFloatingMoveDrag            ID = "floating.move_drag"
	ActionFloatingResizeDrag          ID = "floating.resize_drag"
	ActionEmptyAttach                 ID = "empty.attach"
	ActionEmptyCreate                 ID = "empty.create"
	ActionEmptyManager                ID = "empty.manager"
	ActionEmptyClose                  ID = "empty.close"
	ActionExitedRestart               ID = "exited.restart"
	ActionExitedReconnect             ID = "exited.reconnect"
	ActionExitedClose                 ID = "exited.close"
	ActionDisconnectedReconnect       ID = "disconnected.reconnect"
	ActionDisconnectedDisconnect      ID = "disconnected.disconnect"
	ActionTerminalPickerNew           ID = "terminal_picker.new"
	ActionTerminalPoolSelect          ID = "terminal_pool.select"
	ActionWorkbenchTreeSelect         ID = "workbench_tree.select"
	ActionClipboardHistorySelect      ID = "clipboard_history.select"
	ActionClipboardHistoryDividerDrag ID = "clipboard_history.divider_drag"
)

var surfaceOnlyActionIDs = []ID{
	ActionPanelFocus, ActionPanelResizeDrag, ActionTabSelect,
	ActionFloatingRaise, ActionFloatingResize, ActionFloatingMoveDrag, ActionFloatingResizeDrag,
	ActionEmptyAttach, ActionEmptyCreate, ActionEmptyManager, ActionEmptyClose,
	ActionExitedRestart, ActionExitedReconnect, ActionExitedClose,
	ActionDisconnectedReconnect, ActionDisconnectedDisconnect,
	ActionTerminalPickerNew, ActionTerminalPoolSelect, ActionWorkbenchTreeSelect,
	ActionClipboardHistorySelect, ActionClipboardHistoryDividerDrag,
}
