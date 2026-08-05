//go:build !windows

package main

import (
	"os"
	"strings"
)

func defaultTerminalCommand() []string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return []string{shell}
	}
	return []string{"/bin/sh"}
}
