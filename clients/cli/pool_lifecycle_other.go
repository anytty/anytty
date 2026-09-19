//go:build !darwin && !linux && !windows

package cli

import (
	"fmt"
)

func poolLifecycleSupported() bool { return false }
func poolProcessIdentity(int) (string, error) {
	return "", fmt.Errorf("terminal pool lifecycle is unsupported")
}
func stopPoolProcess(int) error { return fmt.Errorf("terminal pool lifecycle is unsupported") }
func startDetachedPool(string, string, string) error {
	return fmt.Errorf("terminal pool lifecycle is unsupported")
}
func startDetachedAccess(string, string) (int, error) {
	return 0, fmt.Errorf("terminal pool lifecycle is unsupported")
}
