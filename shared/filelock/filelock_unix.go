//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Lock 是一个由打开文件描述符持有的跨进程 advisory exclusive lock。
// 上层必须在 state owner 或 read-modify-write 事务结束后调用 Close；锁文件本身不承载业务数据。
type Lock struct {
	file *os.File
}

func acquirePlatform(path string) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open process lock: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("secure process lock: %w", err)
	}
	operation := syscall.LOCK_EX | syscall.LOCK_NB
	if err := syscall.Flock(int(file.Fd()), operation); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrHeld
		}
		return nil, fmt.Errorf("acquire process lock: %w", err)
	}
	return &Lock{file: file}, nil
}

// Close 释放 advisory lock 并关闭底层文件描述符；重复调用保持幂等。
func (lock *Lock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	file := lock.file
	lock.file = nil
	unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	closeErr := file.Close()
	return errors.Join(unlockErr, closeErr)
}
