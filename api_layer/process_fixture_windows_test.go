//go:build windows

package apilayer

func testIdleTerminalCommand() []string {
	return []string{"cmd.exe", "/d", "/q"}
}
