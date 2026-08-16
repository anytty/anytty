package app

import (
	"context"
	"errors"
	"log/slog"
)

func isContextLifecycleError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// 正常退出、重连或 watcher 重订阅会触发 context 取消；只记日志，不能污染 pane/toast。
func logEffectError(logger *slog.Logger, component string, err error, attrs ...any) {
	if logger == nil || err == nil {
		return
	}
	values := append([]any{"component", component, "error", err}, attrs...)
	if isContextLifecycleError(err) {
		logger.Debug("tui-v3 effect stopped", values...)
		return
	}
	logger.Warn("tui-v3 effect failed", values...)
}

func logLifecycleTrace(logger *slog.Logger, event string, attrs ...any) {
	if logger == nil {
		return
	}
	if !terminalInputTraceEnabled() && !diagnosticsEnabledFromEnv(tuiDiagnosticsEnv) {
		return
	}
	values := append([]any{"event", event}, attrs...)
	logger.Info("tui-v3 lifecycle trace", values...)
}
