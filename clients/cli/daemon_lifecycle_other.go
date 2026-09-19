//go:build !darwin && !linux && !windows

package cli

import (
	"fmt"
)

func daemonLifecycleSupported() bool { return false }
func daemonProcessIdentity(int) (string, error) {
	return "", fmt.Errorf("daemon lifecycle is unsupported")
}
func stopDaemonProcess(int) error { return fmt.Errorf("daemon lifecycle is unsupported") }
func startDetachedDaemon(string, string, string) error {
	return fmt.Errorf("daemon lifecycle is unsupported")
}
func startDetachedAccess(string, string) (int, error) {
	return 0, fmt.Errorf("daemon lifecycle is unsupported")
}
