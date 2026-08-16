// Package localstorage implements host-owned persistence for TUI state that
// must remain available when the active endpoint changes or is disconnected.
package localstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/anytty/anytty/shared/filelock"
	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/securefs"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

const (
	clipboardFileSchema        = "anytty.tui.clipboard-file"
	clipboardFileSchemaVersion = 1
	clipboardFileMaxBytes      = 2 << 20
	defaultClipboardPoll       = 500 * time.Millisecond
)

type clipboardFileDocument struct {
	Schema        string                         `json:"schema"`
	SchemaVersion int                            `json:"schemaVersion"`
	Version       uint64                         `json:"version"`
	Snapshot      state.ClipboardStorageSnapshot `json:"snapshot"`
}

// ClipboardStorage keeps the materialized copy buffer on the TUI host. Path
// is explicit so command startup, tests, and embedded clients control which
// machine owns the state.
type ClipboardStorage struct {
	Path         string
	PollInterval time.Duration
}

func (storage ClipboardStorage) LoadClipboard(_ context.Context, ref state.ClipboardStorageRef) (port.ClipboardStorageLoadResult, error) {
	document, found, err := storage.load()
	if err != nil {
		return port.ClipboardStorageLoadResult{}, err
	}
	if !found {
		return port.ClipboardStorageLoadResult{Found: false}, nil
	}
	return port.ClipboardStorageLoadResult{
		Snapshot: document.Snapshot,
		Version:  document.Version,
		Found:    true,
	}, nil
}

func (storage ClipboardStorage) SaveClipboard(ctx context.Context, request port.ClipboardStorageSaveRequest) (port.ClipboardStorageSaveResult, error) {
	if err := request.Snapshot.Validate(); err != nil {
		return port.ClipboardStorageSaveResult{}, err
	}
	path, err := storage.resolvedPath()
	if err != nil {
		return port.ClipboardStorageSaveResult{}, err
	}
	if err := ensurePrivateDirectory(filepath.Dir(path)); err != nil {
		return port.ClipboardStorageSaveResult{}, fmt.Errorf("prepare clipboard storage: %w", err)
	}
	lock, err := filelock.AcquireContext(ctx, path+".lock", false)
	if err != nil {
		return port.ClipboardStorageSaveResult{}, fmt.Errorf("lock clipboard storage: %w", err)
	}
	defer lock.Close()

	current, found, err := storage.load()
	if err != nil {
		return port.ClipboardStorageSaveResult{}, err
	}
	currentVersion := uint64(0)
	if found {
		currentVersion = current.Version
	}
	if request.CheckVersion && request.ExpectedVersion != currentVersion {
		return port.ClipboardStorageSaveResult{}, port.ErrClipboardStorageConflict
	}
	nextVersion := currentVersion + 1
	document := clipboardFileDocument{
		Schema:        clipboardFileSchema,
		SchemaVersion: clipboardFileSchemaVersion,
		Version:       nextVersion,
		Snapshot:      request.Snapshot,
	}
	if err := writeClipboardDocument(path, document); err != nil {
		return port.ClipboardStorageSaveResult{}, err
	}
	return port.ClipboardStorageSaveResult{
		Ref:     request.Ref.WithVersion(nextVersion),
		Version: nextVersion,
	}, nil
}

func (storage ClipboardStorage) WatchClipboard(ctx context.Context, ref state.ClipboardStorageRef) (<-chan port.ClipboardStorageEvent, error) {
	path, err := storage.resolvedPath()
	if err != nil {
		return nil, err
	}
	document, found, err := storage.load()
	if err != nil {
		return nil, err
	}
	version := uint64(0)
	if found {
		version = document.Version
	}
	// Start without a fingerprint so the first tick closes the race between the
	// initial load and a writer atomically replacing the file.
	var lastInfo os.FileInfo
	interval := storage.PollInterval
	if interval <= 0 {
		interval = defaultClipboardPoll
	}
	events := make(chan port.ClipboardStorageEvent, 1)
	go func() {
		defer close(events)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				info, infoErr := os.Stat(path)
				if errors.Is(infoErr, os.ErrNotExist) {
					lastInfo = nil
					continue
				}
				if infoErr != nil {
					continue
				}
				if sameClipboardFileState(lastInfo, info) {
					continue
				}
				next, nextFound, loadErr := storage.load()
				if loadErr != nil || !nextFound || next.Version == version {
					lastInfo = info
					continue
				}
				lastInfo = info
				version = next.Version
				select {
				case events <- port.ClipboardStorageEvent{Ref: ref.WithVersion(version), Version: version}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return events, nil
}

func sameClipboardFileState(previous os.FileInfo, current os.FileInfo) bool {
	if previous == nil || current == nil || !os.SameFile(previous, current) {
		return false
	}
	return previous.Size() == current.Size() && previous.ModTime().Equal(current.ModTime())
}

func (storage ClipboardStorage) resolvedPath() (string, error) {
	path := filepath.Clean(storage.Path)
	if storage.Path == "" || path == "." || filepath.Base(path) == "." {
		return "", fmt.Errorf("clipboard storage path is required")
	}
	return path, nil
}

func (storage ClipboardStorage) load() (clipboardFileDocument, bool, error) {
	path, err := storage.resolvedPath()
	if err != nil {
		return clipboardFileDocument{}, false, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return clipboardFileDocument{}, false, nil
	}
	if err != nil {
		return clipboardFileDocument{}, false, fmt.Errorf("open clipboard storage: %w", err)
	}
	defer file.Close()
	if err := securefs.ValidatePrivateFileHandle(file); err != nil {
		return clipboardFileDocument{}, false, fmt.Errorf("validate clipboard storage: %w", err)
	}
	payload, err := io.ReadAll(io.LimitReader(file, clipboardFileMaxBytes+1))
	if err != nil {
		return clipboardFileDocument{}, false, fmt.Errorf("read clipboard storage: %w", err)
	}
	if len(payload) > clipboardFileMaxBytes {
		return clipboardFileDocument{}, false, fmt.Errorf("clipboard storage exceeds %d bytes", clipboardFileMaxBytes)
	}
	var document clipboardFileDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		return clipboardFileDocument{}, false, fmt.Errorf("decode clipboard storage: %w", err)
	}
	if document.Schema != clipboardFileSchema || document.SchemaVersion != clipboardFileSchemaVersion || document.Version == 0 {
		return clipboardFileDocument{}, false, fmt.Errorf("unsupported clipboard storage schema")
	}
	if err := document.Snapshot.Validate(); err != nil {
		return clipboardFileDocument{}, false, fmt.Errorf("validate clipboard snapshot: %w", err)
	}
	return document, true, nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return securefs.SecureDirectory(path)
}

func writeClipboardDocument(path string, document clipboardFileDocument) error {
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode clipboard storage: %w", err)
	}
	payload = append(payload, '\n')
	if len(payload) > clipboardFileMaxBytes {
		return fmt.Errorf("clipboard storage exceeds %d bytes", clipboardFileMaxBytes)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".clipboard-history-*.tmp")
	if err != nil {
		return fmt.Errorf("create clipboard storage temp file: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := securefs.SecureFileHandle(temporary); err != nil {
		return fmt.Errorf("secure clipboard storage temp file: %w", err)
	}
	if _, err := temporary.Write(payload); err != nil {
		return fmt.Errorf("write clipboard storage: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync clipboard storage: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close clipboard storage: %w", err)
	}
	if err := filepublish.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish clipboard storage: %w", err)
	}
	committed = true
	if err := filepublish.SyncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("sync clipboard storage directory: %w", err)
	}
	return nil
}
