//go:build windows

package core

import (
	"os"
	"strings"
)

func ptyInteractiveFixture() ([]string, []byte) {
	return []string{windowsCommandInterpreter(), "/d", "/v:on", "/q", "/c", "echo alpha & set /p line= & echo echo:!line!"}, []byte("beta\r\n")
}

func ptyEnvironmentFixture() []string {
	return []string{windowsCommandInterpreter(), "/d", "/q", "/c", "echo cwd:%CD% env:%ANYTTY_REMOTE_TEST%"}
}

func ptyLongRunningFixture() []string {
	return []string{windowsCommandInterpreter(), "/d", "/q"}
}

func windowsCommandInterpreter() string {
	if command := strings.TrimSpace(os.Getenv("COMSPEC")); command != "" {
		return command
	}
	return "cmd.exe"
}
