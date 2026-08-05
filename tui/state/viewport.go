package state

// ViewportStore 保存外部 terminal emulator 的可绘制区域。
// 它是 TUI canvas 的尺寸来源，不能与 PTY session/surface 尺寸混用。
type ViewportStore struct {
	Cols  int
	Rows  int
	Valid bool
}

// Resize 返回应用外部 viewport 后的状态，以及尺寸是否真实变化。
func (store ViewportStore) Resize(cols int, rows int) (ViewportStore, bool) {
	if cols <= 0 || rows <= 0 {
		return store, false
	}
	if store.Valid && store.Cols == cols && store.Rows == rows {
		return store, false
	}
	store.Cols = cols
	store.Rows = rows
	store.Valid = true
	return store, true
}
