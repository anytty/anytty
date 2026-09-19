package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/anytty/anytty/shared/userdirs"
)

// LogFileEnvVar names the log file the tui2 host redirects the standard
// logger to. The -log-file flag wins over it; both default to
// $XDG_STATE_HOME/anytty/tui2.log.
const LogFileEnvVar = "TUI2_LOG_FILE"

// logFileName is the log file name under StateHome/anytty.
const logFileName = "tui2.log"

// logFile is the process-wide log file opened by setupLogging. The host
// writes every diagnostic there, so libraries using the standard logger
// (client/runtime, client/adapter/*, shared/netpath, shared/connecttrace)
// can never print over the alt screen.
var logFile *os.File

// resolveLogPath picks the log file: explicit flag, then environment, then
// the default state directory.
func resolveLogPath(explicit string) string {
	if path := strings.TrimSpace(explicit); path != "" {
		return path
	}
	if path := strings.TrimSpace(os.Getenv(LogFileEnvVar)); path != "" {
		return path
	}
	return filepath.Join(userdirs.StateHome(), "anytty", logFileName)
}

// setupLogging redirects the Go standard logger to the TUI log file and
// returns the resolved path. It must run before the host takes over the
// terminal: from that point on no library diagnostic may reach stderr.
// The file is created owner-only with its parent directory.
func setupLogging(explicit string) (string, error) {
	path := resolveLogPath(explicit)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("log file %s: %w", path, err)
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("log file %s: %w", path, err)
	}
	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	if logFile != nil {
		_ = logFile.Close()
	}
	logFile = file
	return path, nil
}

// closeLogging flushes and closes the host log file.
func closeLogging() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}
