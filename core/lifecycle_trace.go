package core

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

func coreLifecycleTrace(logger *slog.Logger, event string, attrs ...any) {
	if logger == nil || !coreLifecycleTraceEnabled() {
		return
	}
	values := append([]any{"event", event}, attrs...)
	logger.Info("core-v2 lifecycle trace", values...)
}

func coreLifecycleTraceEnabled() bool {
	return coreTraceEnvEnabled("ANYTTY_TUI_INPUT_TRACE") || coreTraceEnvEnabled("ANYTTY_TUI_DIAG")
}

func coreTraceEnvEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "on", "yes", "debug":
		return true
	default:
		return false
	}
}

func coreTerminalInfoAttrs(info TerminalInfo) []any {
	return []any{
		"terminal_id", info.ID,
		"state", string(info.State),
		"exit_code", coreTraceExitCode(info.ExitCode),
		"exited_at", coreTraceTime(info.ExitedAt),
		"command", strings.Join(info.Command, " "),
		"cols", info.Size.Cols,
		"rows", info.Size.Rows,
	}
}

func coreTraceExitCode(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func coreTraceTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
