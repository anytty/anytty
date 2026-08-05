package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/anytty/anytty/shared/securefs"
	"github.com/anytty/anytty/shared/userdirs"
)

const defaultLogMaxBytes int64 = 10 * 1024 * 1024

func ensurePrivateLogDirectory(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return securefs.SecureDirectory(path)
}

func resolveLogFilePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if path := os.Getenv("ANYTTY_LOG_FILE"); path != "" {
		return path
	}
	return filepath.Join(userdirs.StateHome(), "anytty", "anytty.log")
}

func resolveWorkspaceStatePath() string {
	return filepath.Join(userdirs.StateHome(), "anytty", "workspace-state.json")
}

func resolveStateFilePath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "anytty.state"
	}
	return filepath.Join(userdirs.StateHome(), "anytty", name)
}

func resolveGridStatePath() string {
	if path := os.Getenv("ANYTTY_GRID_DIR"); path != "" {
		return path
	}
	return filepath.Join(userdirs.StateHome(), "anytty", "grid")
}

func resolveLogLevel() slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ANYTTY_LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func resolveLogMaxBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("ANYTTY_LOG_MAX_BYTES"))
	if raw == "" {
		return defaultLogMaxBytes
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return defaultLogMaxBytes
	}
	return value
}

type rotatingLogWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	file     *os.File
	size     int64
}

func newRotatingLogWriter(path string, maxBytes int64) (*rotatingLogWriter, error) {
	writer := &rotatingLogWriter{
		path:     path,
		maxBytes: maxBytes,
	}
	if err := writer.openLocked(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.maxBytes > 0 && w.size > 0 && w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

func (w *rotatingLogWriter) openLocked() error {
	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := securefs.SecureFile(w.path); err != nil {
		_ = file.Close()
		return err
	}
	info, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return statErr
	}
	if w.maxBytes > 0 && info.Size() > w.maxBytes {
		_ = file.Close()
		if err := os.Remove(w.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		file, err = os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if err := securefs.SecureFile(w.path); err != nil {
			_ = file.Close()
			return err
		}
		info, statErr = file.Stat()
		if statErr != nil {
			_ = file.Close()
			return statErr
		}
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingLogWriter) rotateLocked() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	rotated := w.path + ".1"
	if err := os.Remove(rotated); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(w.path, rotated); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if err := securefs.SecureFile(w.path); err != nil {
		_ = file.Close()
		return err
	}
	w.file = file
	w.size = 0
	return nil
}

func openLogFileLogger(explicit string) (*slog.Logger, func() error, string, error) {
	path := resolveLogFilePath(explicit)
	parent := filepath.Dir(path)
	if err := ensurePrivateLogDirectory(parent); err != nil {
		return nil, nil, path, err
	}
	writer, err := newRotatingLogWriter(path, resolveLogMaxBytes())
	if err != nil {
		return nil, nil, path, err
	}
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: resolveLogLevel()})
	logger := slog.New(handler).With("pid", os.Getpid())
	closeFn := func() error {
		return writer.Close()
	}
	return logger, closeFn, path, nil
}
