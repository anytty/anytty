//go:build windows

package userdirs

import (
	"os"
	"path/filepath"
	"strings"
)

func platformConfigHome() string {
	if path := strings.TrimSpace(os.Getenv("APPDATA")); path != "" {
		return filepath.Clean(path)
	}
	if path, err := os.UserConfigDir(); err == nil && strings.TrimSpace(path) != "" {
		return filepath.Clean(path)
	}
	return filepath.Join(os.TempDir(), "anytty-config")
}

func platformStateHome() string {
	if path := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); path != "" {
		return filepath.Clean(path)
	}
	if path, err := os.UserCacheDir(); err == nil && strings.TrimSpace(path) != "" {
		return filepath.Clean(path)
	}
	return filepath.Join(os.TempDir(), "anytty-state")
}
