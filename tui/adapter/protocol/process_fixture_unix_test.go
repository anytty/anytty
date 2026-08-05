//go:build !windows

package protocoladapter

func testIdleTerminalCommand() []string {
	return []string{"/bin/cat"}
}
