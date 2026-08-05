//go:build windows

package main

func v3E2ESmokeTerminalCommand() []string {
	return []string{
		"powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
		`[Console]::OutputEncoding=[Text.UTF8Encoding]::new($false); Write-Output 'alpha'; Write-Output 'beta'; while (($line=[Console]::In.ReadLine()) -ne $null) { Write-Output ('echo:'+$line) }`,
	}
}
