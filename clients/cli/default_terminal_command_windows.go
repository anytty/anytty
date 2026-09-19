//go:build windows

package cli

import (
	"os"
	"strings"
)

func defaultTerminalCommand() []string {
	if shell := strings.TrimSpace(os.Getenv("COMSPEC")); shell != "" {
		return []string{shell}
	}
	return []string{"cmd.exe"}
}
