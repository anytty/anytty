package state

// HostCapabilityStore 保存宿主 terminal emulator 已确认的运行时能力。
// 它只描述当前 TUI 会话的宿主输入能力，不属于 terminal daemon、PTY 或持久化配置真值。
type HostCapabilityStore struct {
	KeyboardProbed         bool
	KeyboardDisambiguation bool
}

// HostCapabilityUpdate 是 TerminalHost capability query 回投 reducer 的消息载荷。
type HostCapabilityUpdate struct {
	KeyboardDisambiguation bool
}

// ApplyUpdate 应用一次已解析的宿主 capability 响应。
func (store HostCapabilityStore) ApplyUpdate(update HostCapabilityUpdate) HostCapabilityStore {
	store.KeyboardProbed = true
	store.KeyboardDisambiguation = update.KeyboardDisambiguation
	return store
}
