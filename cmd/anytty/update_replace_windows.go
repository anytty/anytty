//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

func replaceUpdateExecutable(candidatePath, targetPath string) error {
	// Windows cannot overwrite a mapped executable, but it permits renaming it.
	// The running CLI and daemon keep their old image while new invocations use targetPath.
	oldPath := filepath.Join(filepath.Dir(targetPath), fmt.Sprintf(".%s.old-%d-%d", filepath.Base(targetPath), os.Getpid(), time.Now().UnixNano()))
	if err := os.Rename(targetPath, oldPath); err != nil {
		return fmt.Errorf("move current executable aside: %w", err)
	}
	if err := os.Rename(candidatePath, targetPath); err != nil {
		rollbackErr := os.Rename(oldPath, targetPath)
		if rollbackErr != nil {
			return fmt.Errorf("install update: %v; rollback failed: %w", err, rollbackErr)
		}
		return fmt.Errorf("install update: %w", err)
	}
	if err := os.Remove(oldPath); err != nil {
		if pointer, pointerErr := windows.UTF16PtrFromString(oldPath); pointerErr == nil {
			_ = windows.SetFileAttributes(pointer, windows.FILE_ATTRIBUTE_HIDDEN)
		}
	}
	return nil
}
