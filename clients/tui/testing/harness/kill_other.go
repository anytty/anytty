//go:build !linux && !darwin

package harness

// killGroup is a no-op where process groups are not signalable; Kill still
// closes the PTY and terminates the direct child.
func killGroup(int) {}
