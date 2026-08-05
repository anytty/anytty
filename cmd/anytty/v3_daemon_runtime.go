package main

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

const daemonMemoryLimitMBEnv = "ANYTTY_DAEMON_MEMORY_LIMIT_MB"

func applyDaemonRuntimeTuning(logger *slog.Logger) {
	limitMBText := strings.TrimSpace(os.Getenv(daemonMemoryLimitMBEnv))
	if limitMBText == "" {
		return
	}
	limitMB, err := strconv.Atoi(limitMBText)
	if err != nil || limitMB <= 0 {
		logger.Warn("invalid daemon memory limit", "env", daemonMemoryLimitMBEnv, "value", limitMBText)
		return
	}
	limitBytes := int64(limitMB) << 20
	previous := debug.SetMemoryLimit(limitBytes)
	// 中文说明：这是显式 GC pacing 上限，用于 RSS smoke 验证 allocator 高水位；
	// 不清理 history/live truth，也不在运行中靠定时 scrub 掩盖内存问题。
	logger.Info("daemon memory limit configured", "limit_mb", limitMB, "previous_bytes", previous)
}
