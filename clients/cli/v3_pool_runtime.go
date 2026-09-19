package cli

import (
	"log/slog"
	"runtime/debug"
	"strconv"
)

const poolMemoryLimitMBEnv = "ANYTTY_POOL_MEMORY_LIMIT_MB"
const legacyPoolMemoryLimitMBEnv = "ANYTTY_DAEMON_MEMORY_LIMIT_MB"

func applyPoolRuntimeTuning(logger *slog.Logger) {
	limitMBText := poolEnv(poolMemoryLimitMBEnv, legacyPoolMemoryLimitMBEnv)
	if limitMBText == "" {
		return
	}
	limitMB, err := strconv.Atoi(limitMBText)
	if err != nil || limitMB <= 0 {
		logger.Warn("invalid terminal pool memory limit", "env", poolMemoryLimitMBEnv, "value", limitMBText)
		return
	}
	limitBytes := int64(limitMB) << 20
	previous := debug.SetMemoryLimit(limitBytes)
	// 中文说明：这是显式 GC pacing 上限，用于 RSS smoke 验证 allocator 高水位；
	// 不清理 history/live truth，也不在运行中靠定时 scrub 掩盖内存问题。
	logger.Info("terminal pool memory limit configured", "limit_mb", limitMB, "previous_bytes", previous)
}
