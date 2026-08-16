//go:build windows

package filelock

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// Lock 是一个由打开文件句柄持有的跨进程 advisory exclusive lock。
// 上层必须在 state owner 或 read-modify-write 事务结束后调用 Close；锁文件本身不承载业务数据。
type Lock struct {
	file       *os.File
	overlapped windows.Overlapped
}

func acquirePlatform(path string) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open process lock: %w", err)
	}
	lock := &Lock{file: file}
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	if err := windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, &lock.overlapped); err != nil {
		_ = file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, ErrHeld
		}
		return nil, fmt.Errorf("acquire process lock: %w", err)
	}
	return lock, nil
}

// Close 释放 advisory lock 并关闭底层文件句柄；重复调用保持幂等。
func (lock *Lock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	file := lock.file
	lock.file = nil
	unlockErr := windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &lock.overlapped)
	closeErr := file.Close()
	return errors.Join(unlockErr, closeErr)
}
