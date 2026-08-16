package core

import "testing"

func TestParseForegroundProcessUsesPTYForegroundGroupAndNormalizesAgent(t *testing.T) {
	output := []byte("100 1 220 /bin/zsh -l\n220 100 220 /opt/homebrew/bin/node /opt/tools/codex/bin/codex secret-argument\n")
	process, ok := parseForegroundProcess(100, output)
	if !ok || process != "codex" {
		t.Fatalf("foreground process = %q, %v", process, ok)
	}
}

func TestNormalizeForegroundProcessNameReturnsOnlyExecutableIdentity(t *testing.T) {
	process, ok := normalizeForegroundProcessName("/usr/local/bin/python3 --token secret")
	if !ok || process != "python3" {
		t.Fatalf("normalized process = %q, %v", process, ok)
	}
	if _, ok := normalizeForegroundProcessName("bad\nprocess"); ok {
		t.Fatal("control characters must be rejected")
	}
}

func TestNormalizeForegroundProcessNameRecognizesWrappedAgentCLIs(t *testing.T) {
	tests := map[string]string{
		"/usr/bin/node /opt/npm/github-copilot/bin/copilot.js": "copilot",
		"/usr/bin/node /opt/npm/@qwen-code/qwen-code.js":       "qwen",
		"/usr/bin/node /opt/npm/cursor-agent/index.js":         "cursor",
		"/usr/bin/node /opt/npm/opencode/bin/opencode.js":      "opencode",
	}
	for command, want := range tests {
		got, ok := normalizeForegroundProcessName(command)
		if !ok || got != want {
			t.Fatalf("normalizeForegroundProcessName(%q) = %q, %v; want %q, true", command, got, ok, want)
		}
	}
}
