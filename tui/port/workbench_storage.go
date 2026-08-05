package port

import (
	"context"
	"errors"

	"github.com/anytty/anytty/tui/state"
)

// WorkbenchStorageService 是 reducer-owned workbench snapshot 的持久化 application port。
// storage 只保存 TUI layout/binding intent，不保存 terminal lifecycle、history truth 或 client session generation。
type WorkbenchStorageService interface {
	LoadWorkbench(context.Context, state.WorkbenchStorageRef) (WorkbenchStorageLoadResult, error)
	SaveWorkbench(context.Context, WorkbenchStorageSaveRequest) (WorkbenchStorageSaveResult, error)
	WatchWorkbench(context.Context, state.WorkbenchStorageRef) (<-chan WorkbenchStorageEvent, error)
}

// WorkbenchStorageLoadResult 返回持久化 workbench snapshot 及其乐观并发版本。
type WorkbenchStorageLoadResult struct {
	Snapshot state.WorkbenchStorageSnapshot
	Version  uint64
	Found    bool
}

// WorkbenchStorageSaveRequest 携带 reducer 生成的 snapshot 和可选版本前置条件。
type WorkbenchStorageSaveRequest struct {
	Ref             state.WorkbenchStorageRef
	Snapshot        state.WorkbenchStorageSnapshot
	CheckVersion    bool
	ExpectedVersion uint64
}

// WorkbenchStorageSaveResult 返回成功写入后的 storage version。
type WorkbenchStorageSaveResult struct {
	Ref     state.WorkbenchStorageRef
	Version uint64
}

// WorkbenchStorageEvent 通知 reducer 某个 storage ref 已发生外部版本变化。
type WorkbenchStorageEvent struct {
	Ref     state.WorkbenchStorageRef
	Version uint64
	Op      string
}

var ErrWorkbenchStorageConflict = errors.New("workbench storage version conflict")
