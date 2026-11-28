//go:build !windows

package cli

import (
	"os"
	"syscall"
)

// runTUI2Host replaces the CLI process with the TUI host.
//
// `anytty` used to spawn tui2 as a child and wait for it. The CLI then still
// held the terminal, and a stray reader inside it raced the TUI for
// keystrokes: the picker showed but ESC and typing went to the parent, so the
// UI looked frozen. exec keeps exactly one process on the terminal, which is
// what every other TUI launcher does.
//
// On success this never returns. Defers in the caller (the temporary
// endpoints registry cleanup) do not run; the temp file stays in /tmp and is
// recreated per launch.
func runTUI2Host(hostPath string, args []string, routes string) error {
	argv := append([]string{hostPath}, args...)
	env := append(os.Environ(), "TUI2_ROUTES="+routes)
	return syscall.Exec(hostPath, argv, env)
}
