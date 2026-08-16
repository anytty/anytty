// Package filelock 提供跨进程 owner/事务锁 primitive，不拥有上层 state 或业务语义。
package filelock

import (
	"context"
	"errors"
	"time"
)

const retryInterval = 10 * time.Millisecond

var (
	// ErrHeld 表示 non-blocking Acquire 发现锁已由另一个进程持有。
	// 调用方必须按自己的领域决定 fail closed 或稍后重试，不能把它解释为空 state。
	ErrHeld = errors.New("process file lock is already held")
)

// Acquire 在 path 上取得跨进程 exclusive lock。
// nonBlocking 为 true 时冲突立即返回 ErrHeld；为 false 时使用 Background context 等待，适合没有调用 deadline 的 daemon owner 路径。
func Acquire(path string, nonBlocking bool) (*Lock, error) {
	return AcquireContext(context.Background(), path, nonBlocking)
}

// AcquireContext 在 path 上取得跨进程 exclusive lock，并让阻塞等待响应调用方取消或 deadline。
// 平台层每次只做 non-blocking 尝试；context 结束后返回其错误，不保留文件描述符或进程内 fallback。
func AcquireContext(ctx context.Context, path string, nonBlocking bool) (*Lock, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if nonBlocking {
		return acquirePlatform(path)
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lock, err := acquirePlatform(path)
		if err == nil {
			return lock, nil
		}
		if !errors.Is(err, ErrHeld) {
			return nil, err
		}
		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
