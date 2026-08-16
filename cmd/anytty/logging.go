package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/anytty/anytty/shared/securefs"
	"github.com/anytty/anytty/shared/userdirs"
	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/grpclog"
)

const defaultLogMaxBytes int64 = 10 * 1024 * 1024

var processLogRedirects = struct {
	sync.Mutex
	nextID uint64
	base   io.Writer
	stack  []processLogRedirect
}{}

var processStderrRedirects = struct {
	sync.Mutex
	nextID uint64
	base   *os.File
	stack  []processStderrRedirect
}{}

type processLogRedirect struct {
	id     uint64
	writer io.Writer
}

type processStderrRedirect struct {
	id     uint64
	writer *os.File
}

type processLogWriter struct{}

func init() {
	output := io.Writer(processLogWriter{})
	otel.SetLogger(stdr.New(log.New(output, "", log.LstdFlags|log.Lshortfile)))
	infoOutput := io.Discard
	warningOutput := io.Discard
	errorOutput := io.Discard
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GRPC_GO_LOG_SEVERITY_LEVEL"))) {
	case "info":
		infoOutput = output
	case "warning":
		warningOutput = output
	default:
		errorOutput = output
	}
	verbosity, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("GRPC_GO_LOG_VERBOSITY_LEVEL")))
	grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(infoOutput, warningOutput, errorOutput, verbosity))
}

func (processLogWriter) Write(payload []byte) (int, error) {
	processLogRedirects.Lock()
	writer := io.Writer(os.Stderr)
	if size := len(processLogRedirects.stack); size > 0 {
		writer = processLogRedirects.stack[size-1].writer
	}
	processLogRedirects.Unlock()
	return writer.Write(payload)
}

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
	restoreProcessLog, err := redirectProcessLog(writer)
	if err != nil {
		_ = writer.Close()
		return nil, nil, path, fmt.Errorf("redirect process diagnostics: %w", err)
	}
	var closeOnce sync.Once
	var closeErr error
	closeFn := func() error {
		closeOnce.Do(func() {
			restoreProcessLog()
			closeErr = writer.Close()
		})
		return closeErr
	}
	return logger, closeFn, path, nil
}

// Some connection diagnostics still use the standard library logger. Route it
// through the command's rotating file so it cannot corrupt an active TUI screen.
func redirectProcessLog(writer io.Writer) (func(), error) {
	processLogRedirects.Lock()
	if len(processLogRedirects.stack) == 0 {
		processLogRedirects.base = log.Writer()
	}
	processLogRedirects.nextID++
	id := processLogRedirects.nextID
	processLogRedirects.stack = append(processLogRedirects.stack, processLogRedirect{id: id, writer: writer})
	log.SetOutput(writer)
	processLogRedirects.Unlock()
	restoreStderr, err := redirectProcessStderr(writer)
	if err != nil {
		restoreProcessLogRedirect(id)
		return nil, err
	}

	var restoreOnce sync.Once
	return func() {
		restoreOnce.Do(func() {
			restoreStderr()
			restoreProcessLogRedirect(id)
		})
	}, nil
}

func restoreProcessLogRedirect(id uint64) {
	processLogRedirects.Lock()
	defer processLogRedirects.Unlock()
	for index, redirect := range processLogRedirects.stack {
		if redirect.id != id {
			continue
		}
		processLogRedirects.stack = append(processLogRedirects.stack[:index], processLogRedirects.stack[index+1:]...)
		break
	}
	if size := len(processLogRedirects.stack); size > 0 {
		log.SetOutput(processLogRedirects.stack[size-1].writer)
	} else if processLogRedirects.base != nil {
		log.SetOutput(processLogRedirects.base)
		processLogRedirects.base = nil
	}
}

// redirectProcessStderr catches libraries that keep a private logger around
// os.Stderr instead of using slog or the standard library logger.
func redirectProcessStderr(destination io.Writer) (func(), error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	processStderrRedirects.Lock()
	if len(processStderrRedirects.stack) == 0 {
		processStderrRedirects.base = os.Stderr
	}
	processStderrRedirects.nextID++
	id := processStderrRedirects.nextID
	redirect := processStderrRedirect{id: id, writer: writer}
	processStderrRedirects.stack = append(processStderrRedirects.stack, redirect)
	os.Stderr = writer
	processStderrRedirects.Unlock()
	go func() {
		_, _ = io.Copy(destination, reader)
		close(done)
	}()

	var restoreOnce sync.Once
	return func() {
		restoreOnce.Do(func() {
			processStderrRedirects.Lock()
			for index, candidate := range processStderrRedirects.stack {
				if candidate.id != id {
					continue
				}
				processStderrRedirects.stack = append(processStderrRedirects.stack[:index], processStderrRedirects.stack[index+1:]...)
				break
			}
			if size := len(processStderrRedirects.stack); size > 0 {
				os.Stderr = processStderrRedirects.stack[size-1].writer
			} else if processStderrRedirects.base != nil {
				os.Stderr = processStderrRedirects.base
				processStderrRedirects.base = nil
			}
			processStderrRedirects.Unlock()
			_ = writer.Close()
			<-done
			_ = reader.Close()
		})
	}, nil
}
