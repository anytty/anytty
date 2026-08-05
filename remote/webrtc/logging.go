package webrtc

import (
	"fmt"
	"log/slog"
	"strings"

	pionlogging "github.com/pion/logging"
)

// NewLoggerFactory routes Pion's error diagnostics through the owning process
// logger. A nil logger is intentionally silent so library defaults can never
// write directly to an interactive terminal's stderr.
func NewLoggerFactory(logger *slog.Logger) pionlogging.LoggerFactory {
	return slogLoggerFactory{logger: logger}
}

type slogLoggerFactory struct {
	logger *slog.Logger
}

func (factory slogLoggerFactory) NewLogger(scope string) pionlogging.LeveledLogger {
	if factory.logger == nil {
		return slogLeveledLogger{}
	}
	return slogLeveledLogger{logger: factory.logger.With(
		"component", "pion",
		"scope", strings.TrimSpace(scope),
	)}
}

// Pion's default factory emits errors only. Preserve that boundary here so
// enabling AnyTTY's normal info logging does not turn on verbose transport logs.
type slogLeveledLogger struct {
	logger *slog.Logger
}

func (slogLeveledLogger) Trace(string)                {}
func (slogLeveledLogger) Tracef(string, ...any)       {}
func (slogLeveledLogger) Debug(string)                {}
func (slogLeveledLogger) Debugf(string, ...any)       {}
func (slogLeveledLogger) Info(string)                 {}
func (slogLeveledLogger) Infof(string, ...any)        {}
func (slogLeveledLogger) Warn(string)                 {}
func (slogLeveledLogger) Warnf(string, ...any)        {}
func (logger slogLeveledLogger) Error(message string) { logger.logError(message) }
func (logger slogLeveledLogger) Errorf(format string, args ...any) {
	logger.logError(fmt.Sprintf(format, args...))
}

func (logger slogLeveledLogger) logError(message string) {
	if logger.logger != nil {
		logger.logger.Error(strings.TrimSpace(message))
	}
}
