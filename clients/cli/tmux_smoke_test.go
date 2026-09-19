package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// v3 TUI 黑盒测试：通过构建的 anytty+tui2+tui2-shell 在 tmux 中运行默认入口。
// 依赖 tmux；未安装时跳过。

func tmuxSmokeContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func requireTmux(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux is not installed: %v", err)
	}
}

func TestV3TmuxSmokeCreatesTerminalAndExits(t *testing.T) {
	requireTmux(t)
	anyttyBin := buildAnyTTYBinaryForTest(t)
	result, err := runV3TmuxSmoke(tmuxSmokeContext(t), anyttyBin)
	if err != nil {
		t.Fatalf("tmux smoke: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(result.ArtifactDir) })
	if result.TerminalID != "term-1" || result.Session == "" {
		t.Fatalf("smoke result = %#v", result)
	}
	for _, path := range []string{result.ANSIPath, result.PlainPath} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing/empty: %v", path, err)
		}
	}
	plain, err := os.ReadFile(result.PlainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plain), "tui2-smoke-ok") {
		t.Fatalf("plain capture lost terminal output:\n%s", plain)
	}
}

func TestV3TmuxResizeSmokePropagatesPTYSize(t *testing.T) {
	requireTmux(t)
	anyttyBin := buildAnyTTYBinaryForTest(t)
	result, err := runV3TmuxResizeSmoke(tmuxSmokeContext(t), anyttyBin)
	if err != nil {
		t.Fatalf("tmux resize smoke: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(result.ArtifactDir) })
	beforeRows, beforeCols := v3TmuxSizeValues(result.BeforeSize)
	afterRows, afterCols := v3TmuxSizeValues(result.AfterSize)
	if beforeRows <= 0 || beforeCols <= 0 || afterRows <= beforeRows || afterCols <= beforeCols {
		t.Fatalf("size did not grow: before=%q after=%q", result.BeforeSize, result.AfterSize)
	}
}

func TestV3TmuxANSISmokeRoundTripsSGRAndUnicode(t *testing.T) {
	requireTmux(t)
	anyttyBin := buildAnyTTYBinaryForTest(t)
	result, err := runV3TmuxANSISmoke(tmuxSmokeContext(t), anyttyBin)
	if err != nil {
		t.Fatalf("tmux ansi smoke: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(result.ArtifactDir) })
	plain, err := os.ReadFile(result.PlainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plain), "ANSI-RED") || !strings.Contains(string(plain), "UNICODE-你好-😀") {
		t.Fatalf("plain capture lost ANSI/Unicode output:\n%s", plain)
	}
}

func TestV3TmuxStabilitySmokeRunsRounds(t *testing.T) {
	requireTmux(t)
	anyttyBin := buildAnyTTYBinaryForTest(t)
	result, err := runV3TmuxStabilitySmoke(tmuxSmokeContext(t), anyttyBin, 2)
	if err != nil {
		t.Fatalf("tmux stability smoke: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(result.ArtifactDir) })
	if result.Rounds != 2 || len(result.Artifacts) == 0 {
		t.Fatalf("stability result = %#v", result)
	}
}

func tmuxSmokeFieldValue(output string, key string) string {
	prefix := key + "="
	for _, field := range strings.Fields(output) {
		if strings.HasPrefix(field, prefix) {
			return strings.TrimPrefix(field, prefix)
		}
	}
	return ""
}

func TestV3TmuxTerminalSmokeCommandReportsArtifacts(t *testing.T) {
	requireTmux(t)
	anyttyBin := buildAnyTTYBinaryForTest(t)
	var out bytes.Buffer
	cmd := newDevelopmentRootCmd()
	cmd.SetArgs([]string{"v3", "tmux-terminal-smoke", "--anytty-bin", anyttyBin})
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("v3 tmux-terminal-smoke returned error: %v", err)
	}
	text := out.String()
	if dir := tmuxSmokeFieldValue(text, "artifact_dir"); dir != "" {
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
	}
	if !strings.Contains(text, "anytty v3 tmux terminal smoke ok") ||
		!strings.Contains(text, "terminal=term-1") ||
		!strings.Contains(text, "artifact_dir=") {
		t.Fatalf("unexpected tmux terminal smoke output:\n%s", text)
	}
}
