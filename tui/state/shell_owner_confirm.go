package state

// OwnerConfirmState 只描述鼠标首击后的 UI 待确认态，不改变 terminal owner truth。
type OwnerConfirmState struct {
	ViewID string
	Seq    uint64
}

func (store ShellStore) ArmOwnerConfirm(viewID string) ShellStore {
	store = store.EnsureDefaults()
	store.OwnerConfirm.Seq++
	store.OwnerConfirm.ViewID = viewID
	return store
}

func (store ShellStore) ClearOwnerConfirm(seq uint64) ShellStore {
	store = store.EnsureDefaults()
	if seq != 0 && seq != store.OwnerConfirm.Seq {
		return store
	}
	store.OwnerConfirm.ViewID = ""
	return store
}
