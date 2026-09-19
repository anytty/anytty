//go:build windows

package core

import (
	"os"
	"strings"
)

func currentAccountShell() string {
	if shell := strings.TrimSpace(os.Getenv("COMSPEC")); shell != "" {
		return shell
	}
	return "cmd.exe"
}
