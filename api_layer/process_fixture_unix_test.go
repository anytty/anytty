//go:build !windows

package apilayer

func testIdleTerminalCommand() []string {
	return []string{"/bin/cat"}
}
