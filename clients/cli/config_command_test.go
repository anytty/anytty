package cli

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

// TestConfigCommandsAtomicallyUseRuntimeParser 覆盖 tui2 JSON 配置的
// get/set/unset/show/validate 与失败不改写源文件。
func TestConfigCommandsAtomicallyUseRuntimeParser(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path := filepath.Join(configHome, "anytty", "tui2.json")

	runConfigCommand(t, nil, "config", "set", "theme", "light")
	runConfigCommand(t, nil, "config", "set", "gap", "1")
	runConfigCommand(t, nil, "config", "set", "startup.cwd", "/tmp/anytty-demo")
	runConfigCommand(t, nil, "config", "set", "clock.format", "12h")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !securefs.IsPrivateFile(path, info) {
		t.Fatalf("config permissions are not private: %v", info.Mode())
	}

	if got := strings.TrimSpace(runConfigCommand(t, nil, "config", "get", "theme")); got != `"light"` {
		t.Fatalf("config get theme = %q", got)
	}
	if got := strings.TrimSpace(runConfigCommand(t, nil, "config", "get", "gap")); got != "1" {
		t.Fatalf("config get gap = %q", got)
	}
	if got := strings.TrimSpace(runConfigCommand(t, nil, "config", "get", "startup.cwd")); got != `"/tmp/anytty-demo"` {
		t.Fatalf("config get startup cwd = %q", got)
	}
	if got := strings.TrimSpace(runConfigCommand(t, nil, "config", "get", "clock.format")); got != `"12h"` {
		t.Fatalf("config get clock = %q", got)
	}
	effective := runConfigCommand(t, nil, "config", "show", "--effective")
	if !strings.Contains(effective, `"theme": "light"`) || !strings.Contains(effective, `"gap": 1`) || !strings.Contains(effective, `"clock"`) {
		t.Fatalf("effective config did not use updated values: %s", effective)
	}
	runConfigCommand(t, nil, "config", "validate")

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	invalid := newRootCmd()
	invalid.SetOut(io.Discard)
	invalid.SetErr(io.Discard)
	invalid.SetArgs([]string{"config", "set", "theme", "invalid-mode"})
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

	runConfigCommand(t, nil, "config", "unset", "clock.format")
	missing := newRootCmd()
	missing.SetOut(io.Discard)
	missing.SetErr(io.Discard)
	missing.SetArgs([]string{"config", "get", "clock.format"})
	if err := missing.Execute(); cliExitCode(err) != 3 {
		t.Fatalf("unset config get error = %v, exit=%d", err, cliExitCode(err))
	}
}

func TestConfigPathsUseActualTUIConfigPath(t *testing.T) {
	configHome := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	output := runConfigCommand(t, nil, "config", "paths", "--json")
	var paths struct {
		Config    string `json:"config"`
		Clipboard string `json:"clipboard"`
	}
	if err := json.Unmarshal([]byte(output), &paths); err != nil {
		t.Fatal(err)
	}
	if paths.Config != filepath.Join(configHome, "anytty", "tui2.json") {
		t.Fatalf("config paths = %s", output)
	}
	wantClipboard := filepath.Join(stateHome, "anytty", "clipboard-history.json")
	if paths.Clipboard != wantClipboard {
		t.Fatalf("clipboard path = %q, want %q", paths.Clipboard, wantClipboard)
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
