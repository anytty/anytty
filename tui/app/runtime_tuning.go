package app

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

const tuiMemoryLimitMBEnv = "ANYTTY_TUI_MEMORY_LIMIT_MB"

func applyRuntimeTuning(logger *slog.Logger) {
	limitMBText := strings.TrimSpace(os.Getenv(tuiMemoryLimitMBEnv))
	if limitMBText == "" {
		return
	}
	limitMB, err := strconv.Atoi(limitMBText)
	if err != nil || limitMB <= 0 {
		if logger != nil {
			logger.Warn("invalid tui memory limit", "env", tuiMemoryLimitMBEnv, "value", limitMBText)
		}
		return
	}
	limitBytes := int64(limitMB) << 20
	previous := debug.SetMemoryLimit(limitBytes)
	if logger != nil {
		// 中文说明：这是显式 GC pacing 上限，用于真实 TUI stress RSS 验证；
		// 不清理 live/history truth，也不通过定时 scrub 伪造低内存。
		logger.Info("tui memory limit configured", "limit_mb", limitMB, "previous_bytes", previous)
	}
}
