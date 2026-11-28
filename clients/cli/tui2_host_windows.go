//go:build windows

package cli

// runTUI2Host runs the TUI host as a child on platforms without exec.
func runTUI2Host(hostPath string, args []string, routes string) error {
	return runTUI2Process(hostPath, args, routes)
}
