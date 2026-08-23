//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func replaceUpdateExecutable(candidatePath, targetPath string) error {
	if err := os.Rename(candidatePath, targetPath); err != nil {
		return fmt.Errorf("replace current executable: %w", err)
	}
	directory, err := os.Open(filepath.Dir(targetPath))
	if err != nil {
		return nil
	}
	defer directory.Close()
	_ = directory.Sync()
	return nil
}
