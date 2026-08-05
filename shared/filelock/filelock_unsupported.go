//go:build aix || (!darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows)

package filelock

import "fmt"

// Lock 是受支持平台上跨进程 advisory lock 的占位类型。
// 当前平台不提供可靠实现，因此 Acquire 总是失败，调用方必须 fail closed。
type Lock struct{}

func acquirePlatform(string) (*Lock, error) {
	return nil, fmt.Errorf("process file locks are unsupported on this platform")
}

// Close 保持接口幂等；不受支持的平台不会产生有效 Lock。
func (*Lock) Close() error { return nil }
