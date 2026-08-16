package state

// TerminalInputPassthroughArmed 返回当前 InputMsg 是否已被上游 reducer 判定为必须透传。
// copy/history 等中间 reducer 只能用它让路，不能消费或延长这个同消息令牌。
func (store ShellStore) TerminalInputPassthroughArmed() bool {
	return store.EnsureDefaults().forceTerminalInput
}

// ArmShortcutPassthroughWindow 为某个入口键开启一次双击透传窗口。
// kind 来自 app/input 路由语义；seq 是 reducer-owned timer 防陈旧令牌。
func (store ShellStore) ArmShortcutPassthroughWindow(kind string) ShellStore {
	store = store.EnsureDefaults()
	if kind == "" {
		return store
	}
	store.shortcutPassthroughSeq++
	store.shortcutPassthroughKind = kind
	return store
}

// ShortcutPassthroughWindow 返回当前入口键透传窗口的 seq。
// 返回 false 表示 kind 不匹配，调用方不应把重复按键解释成 PTY 透传。
func (store ShellStore) ShortcutPassthroughWindow(kind string) (uint64, bool) {
	store = store.EnsureDefaults()
	if kind == "" || store.shortcutPassthroughKind != kind {
		return 0, false
	}
	return store.shortcutPassthroughSeq, true
}

// ShortcutPassthroughWindowMatches 判断当前入口键透传窗口是否仍然有效。
// 它只表达同一 TUI 进程内的输入路由状态，不改变 copy mode、overlay 或 terminal 状态。
func (store ShellStore) ShortcutPassthroughWindowMatches(kind string) bool {
	_, ok := store.ShortcutPassthroughWindow(kind)
	return ok
}

// ClearShortcutPassthroughWindow 主动清除某个入口键透传窗口。
// 显式透传成功或进入状态被取消时调用；seq 保持单调，避免旧 timeout 复用。
func (store ShellStore) ClearShortcutPassthroughWindow(kind string) ShellStore {
	store = store.EnsureDefaults()
	if kind == "" || store.shortcutPassthroughKind != kind {
		return store
	}
	store.shortcutPassthroughKind = ""
	return store
}

// ClearShortcutPassthroughWindowTimeout 按 kind/seq 清除入口键透传窗口。
// 返回 false 表示 timeout 已经过期，不能清除用户刚重新打开的新窗口。
func (store ShellStore) ClearShortcutPassthroughWindowTimeout(kind string, seq uint64) (ShellStore, bool) {
	store = store.EnsureDefaults()
	if kind == "" || seq == 0 || store.shortcutPassthroughKind != kind || store.shortcutPassthroughSeq != seq {
		return store, false
	}
	store.shortcutPassthroughKind = ""
	return store, true
}
