package terminalmeta

import "strings"

// SizeLockTag 是 terminal metadata 中表达尺寸锁定策略的共享 tag。
// daemon、remote inventory 和 TUI 都只能把它当作语义字段；具体图标和按钮文案归展示层所有。
const SizeLockTag = "anytty.size_lock"

const (
	SizeLockOff  = "off"
	SizeLockWarn = "warn"
	SizeLockLock = "lock"
)

// SizeLockMode 从 terminal tags 中解析尺寸锁定模式。
// tags 是跨进程 metadata truth；未知值按 off 处理，避免展示层把无效 tag 误判成硬锁。
func SizeLockMode(tags map[string]string) string {
	if len(tags) == 0 {
		return SizeLockOff
	}
	switch strings.ToLower(strings.TrimSpace(tags[SizeLockTag])) {
	case SizeLockLock, "locked", "hard", "true", "on", "1":
		return SizeLockLock
	case SizeLockWarn:
		return SizeLockWarn
	default:
		return SizeLockOff
	}
}

// SizeLocked 判断 terminal 是否处于硬锁尺寸模式。
// 该 helper 只返回共享语义结果，不负责 UI 图标、按钮 label 或颜色。
func SizeLocked(tags map[string]string) bool {
	return SizeLockMode(tags) == SizeLockLock
}
