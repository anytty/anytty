package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAgentHookProviderFailureExitContract(t *testing.T) {
	t.Setenv("ANYTTY_TERMINAL_ID", "fixture")
	t.Setenv("ANYTTY_DAEMON_SOCKET", "/fixture-not-connected.sock")
	for _, provider := range []string{"codex", "opencode"} {
		t.Run(provider, func(t *testing.T) {
			command := newPluginCommand(terminalCommandRuntime{})
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.SetIn(strings.NewReader("not valid JSON"))
			command.SetArgs([]string{"agents", "hook", provider})
			command.SilenceUsage = true
			command.SilenceErrors = true
			err := command.Execute()
			if (err != nil) != (provider == "opencode") {
				t.Fatalf("%s error=%v", provider, err)
			}
			if stdout.String() != "{}\n" {
				t.Fatalf("hook changed Agent output contract: %q", stdout.String())
			}
		})
	}
}
