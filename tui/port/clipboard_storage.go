package port

import (
	"context"
	"errors"

	"github.com/anytty/anytty/tui/state"
)

// ClipboardStorageService 是 reducer-owned clipboard history 的持久化 application port。
// 它不等同于宿主系统剪贴板，也不保存 client runtime 状态。
type ClipboardStorageService interface {
	LoadClipboard(context.Context, state.ClipboardStorageRef) (ClipboardStorageLoadResult, error)
	SaveClipboard(context.Context, ClipboardStorageSaveRequest) (ClipboardStorageSaveResult, error)
	WatchClipboard(context.Context, state.ClipboardStorageRef) (<-chan ClipboardStorageEvent, error)
}

// ClipboardStorageLoadResult 返回 clipboard snapshot 及其乐观并发版本。
type ClipboardStorageLoadResult struct {
	Snapshot state.ClipboardStorageSnapshot
	Version  uint64
	Found    bool
}

// ClipboardStorageSaveRequest 携带 reducer 生成的 clipboard snapshot 和版本前置条件。
type ClipboardStorageSaveRequest struct {
	Ref             state.ClipboardStorageRef
	Snapshot        state.ClipboardStorageSnapshot
	CheckVersion    bool
	ExpectedVersion uint64
}

// ClipboardStorageSaveResult 返回成功写入后的 storage version。
type ClipboardStorageSaveResult struct {
	Ref     state.ClipboardStorageRef
	Version uint64
}

// ClipboardStorageEvent 通知 reducer 某个 clipboard storage ref 已发生外部版本变化。
type ClipboardStorageEvent struct {
	Ref     state.ClipboardStorageRef
	Version uint64
	Op      string
}

var ErrClipboardStorageConflict = errors.New("clipboard storage version conflict")
