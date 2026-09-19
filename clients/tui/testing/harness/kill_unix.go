//go:build linux || darwin

package harness

import "syscall"

// killGroup terminates the whole process group led by pid (the PTY child is
// started in its own session, so its pid is the group id).
func killGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
