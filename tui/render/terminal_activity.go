package render

import (
	"fmt"
	"time"
)

// TerminalOutputActivityLabel 把 terminal 最近一次非空 PTY 输出的时间转成简洁的活跃度标签。
// 空时间（从未产生输出）返回空字符串；刚有输出显示 "now"，一分钟内显示秒数，
// 一小时内显示分钟数，更久显示小时数。
func TerminalOutputActivityLabel(lastOutputAt time.Time, now time.Time) string {
	if lastOutputAt.IsZero() {
		return ""
	}
	quiet := now.Sub(lastOutputAt)
	if quiet < 0 {
		quiet = 0
	}
	switch {
	case quiet < 5*time.Second:
		return "now"
	case quiet < time.Minute:
		return fmt.Sprintf("%ds", int(quiet.Seconds()))
	case quiet < time.Hour:
		return fmt.Sprintf("%dm", int(quiet.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(quiet.Hours()))
	}
}

// TerminalOutputActivityStyle 给活跃度标签一个非红黄绿的样式：
// 刚有输出用 Info（蓝）表达活跃，静默中/更久用 Muted，避免把"安静"误读成错误。
func TerminalOutputActivityStyle(label string) StyleToken {
	if label == "now" {
		return StyleInfo
	}
	return StyleMuted
}
