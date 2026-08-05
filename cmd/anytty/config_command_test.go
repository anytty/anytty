package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anytty/anytty/shared/securefs"
)

func TestConfigCommandsAtomicallyUseRuntimeParser(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path := filepath.Join(configHome, "anytty", "tui-v3.yaml")

	runConfigCommand(t, nil, "config", "set", "tui.theme.mode", "light")
	runConfigCommand(t, nil, "config", "set", "daemon.history.max_size_mb", "256")
	runConfigCommand(t, nil, "config", "set", "daemon.history.max_age_days", "14")
	runConfigCommand(t, nil, "config", "set", "daemon.history.compression", "s2")
	runConfigCommand(t, nil, "config", "set", "daemon.history.compression_level", "balanced")
	runConfigCommand(t, nil, "config", "set", "daemon.output_buffer.capacity_bytes", "1048576")
	runConfigCommand(t, nil, "config", "set", "daemon.output_buffer.overflow", "block")
	runConfigCommand(t, nil, "config", "set", "daemon.output_buffer.resident_budget_bytes", "268435456")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !securefs.IsPrivateFile(path, info) {
		t.Fatalf("config permissions are not private: %v", info.Mode())
	}

	getOutput := runConfigCommand(t, nil, "config", "get", "tui.theme.mode")
	if strings.TrimSpace(getOutput) != "light" {
		t.Fatalf("config get = %q", getOutput)
	}
	maxOutput := runConfigCommand(t, nil, "config", "get", "daemon.history.max_size_mb")
	if strings.TrimSpace(maxOutput) != "256" {
		t.Fatalf("history max config get = %q", maxOutput)
	}
	ageOutput := runConfigCommand(t, nil, "config", "get", "daemon.history.max_age_days")
	if strings.TrimSpace(ageOutput) != "14" {
		t.Fatalf("history max age config get = %q", ageOutput)
	}
	algorithmOutput := runConfigCommand(t, nil, "config", "get", "daemon.history.compression")
	if strings.TrimSpace(algorithmOutput) != "s2" {
		t.Fatalf("history compression config get = %q", algorithmOutput)
	}
	levelOutput := runConfigCommand(t, nil, "config", "get", "daemon.history.compression_level")
	if strings.TrimSpace(levelOutput) != "balanced" {
		t.Fatalf("history compression level config get = %q", levelOutput)
	}
	if output := runConfigCommand(t, nil, "config", "get", "daemon.output_buffer.overflow"); strings.TrimSpace(output) != "block" {
		t.Fatalf("output buffer overflow config get = %q", output)
	}
	effective := runConfigCommand(t, nil, "config", "show", "--effective")
	if !strings.Contains(effective, `"Mode": "light"`) || !strings.Contains(effective, `"MaxSizeMB": 256`) || !strings.Contains(effective, `"Overflow": "block"`) {
		t.Fatalf("effective config did not use updated runtime value: %s", effective)
	}
	runConfigCommand(t, nil, "config", "validate")

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	invalid := newRootCmd()
	invalid.SetOut(io.Discard)
	invalid.SetErr(io.Discard)
	invalid.SetArgs([]string{"config", "set", "tui.theme.mode", "invalid-mode"})
	if err := invalid.Execute(); cliExitCode(err) != 2 {
		t.Fatalf("invalid config error = %v, exit=%d", err, cliExitCode(err))
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("invalid config mutation changed the source file")
	}

	runConfigCommand(t, nil, "config", "unset", "tui.theme.mode")
	missing := newRootCmd()
	missing.SetOut(io.Discard)
	missing.SetErr(io.Discard)
	missing.SetArgs([]string{"config", "get", "tui.theme.mode"})
	if err := missing.Execute(); cliExitCode(err) != 3 {
		t.Fatalf("unset config get error = %v, exit=%d", err, cliExitCode(err))
	}
}

func TestConfigPathsUseActualTUIConfigPath(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	output := runConfigCommand(t, nil, "config", "paths", "--json")
	var paths struct {
		Config string `json:"config"`
	}
	if err := json.Unmarshal([]byte(output), &paths); err != nil {
		t.Fatal(err)
	}
	if paths.Config != filepath.Join(configHome, "anytty", "tui-v3.yaml") {
		t.Fatalf("config paths = %s", output)
	}
}

func runConfigCommand(t *testing.T, input io.Reader, args ...string) string {
	t.Helper()
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	if input != nil {
		command.SetIn(input)
	}
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("anytty %s: %v", strings.Join(args, " "), err)
	}
	return output.String()
}
