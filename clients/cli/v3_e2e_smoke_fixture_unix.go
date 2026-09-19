//go:build !windows

package cli

func v3E2ESmokeTerminalCommand() []string {
	return []string{"/bin/sh", "-c", "printf 'alpha\\nbeta\\n'; while IFS= read -r line; do printf 'echo:%s\\n' \"$line\"; done"}
}
