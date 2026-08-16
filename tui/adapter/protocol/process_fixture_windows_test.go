//go:build windows

package protocoladapter

func testIdleTerminalCommand() []string {
	return []string{"cmd.exe", "/d", "/q"}
}
