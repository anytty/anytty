//go:build !windows

package main

func testShellSleepCommand() []string {
	return []string{"/bin/sh", "-c", "while true; do sleep 1; done"}
}

func testAutomationCommand() []string {
	return []string{"/bin/sh", "-c", `printf 'READY\n'; IFS= read -r line; printf 'GOT:%s\n' "$line"`}
}

func testExitCommand() []string {
	return []string{"/bin/sh", "-c", "exit 0"}
}

func testInitialOutputCommand() []string {
	return []string{"/bin/sh", "-c", "printf 'anytty-initial-output\\n'; sleep 5"}
}
