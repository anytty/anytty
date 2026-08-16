package action

var (
	emptyPaneCTAActions = []ID{
		ActionEmptyAttach,
		ActionEmptyCreate,
		ActionEmptyManager,
		ActionEmptyClose,
	}
	exitedPaneCTAActions = []ID{
		ActionExitedRestart,
		ActionExitedReconnect,
	}
	disconnectedPaneCTAActions = []ID{
		ActionDisconnectedReconnect,
		ActionDisconnectedDisconnect,
	}
)

// EmptyPaneCTAActions 返回空 pane 选择器的 canonical action 顺序。
// action domain 拥有顺序与执行身份；app 用它处理键盘选择，render 只能据此投影文案和几何。
func EmptyPaneCTAActions() []ID {
	return append([]ID(nil), emptyPaneCTAActions...)
}

// ExitedPaneCTAActions 返回已退出 terminal pane 的 canonical action 顺序。
// 列表不携带键位或样式；terminal lifecycle 仍由 core 拥有，app 只把选择转换为 invocation。
func ExitedPaneCTAActions() []ID {
	return append([]ID(nil), exitedPaneCTAActions...)
}

// DisconnectedPaneCTAActions 返回连接中断 pane 的 canonical action 顺序。
// reconnect 与 disconnect 的失败语义由 app handler 拥有，render 不得从 ProjectionID 反向定义执行命令。
func DisconnectedPaneCTAActions() []ID {
	return append([]ID(nil), disconnectedPaneCTAActions...)
}
