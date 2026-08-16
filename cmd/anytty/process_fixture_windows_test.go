//go:build windows

package main

func testShellSleepCommand() []string {
	return []string{"powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 30"}
}

func testAutomationCommand() []string {
	return []string{
		"powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
		`[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false); Write-Output 'READY'; $line=[Console]::In.ReadLine(); Write-Output ('GOT:'+$line)`,
	}
}

func testExitCommand() []string {
	return []string{"cmd.exe", "/d", "/q", "/c", "exit 0"}
}

func testInitialOutputCommand() []string {
	return []string{"powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "Write-Output 'anytty-initial-output'; Start-Sleep -Seconds 5"}
}
