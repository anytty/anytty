package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This checks Codex's own loader, without creating a model thread, trusting a
// hook, modifying the user's config, or bypassing hook trust. Only the installer's
// temporary definitions, passed as process-local overrides, are inspected.
func TestInstalledCodexParsesUntrustedHooks(t *testing.T) {
	if os.Getenv("ANYTTY_TEST_CODEX") != "1" {
		t.Skip("set ANYTTY_TEST_CODEX=1 to test installed Codex loader")
	}
	binary, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	config := filepath.Join(dir, ".codex")
	if err = InstallCodex(config, filepath.Join(dir, "anytty-not-executed")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// The installed definitions are fed to Codex as process-local config overrides.
	// This avoids changing CODEX_HOME, project trust, or any user configuration.
	data, err := os.ReadFile(filepath.Join(config, "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
				Status  string `json:"statusMessage"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	args := []string{"app-server", "--stdio"}
	for event, groups := range fixture.Hooks {
		hook := groups[0].Hooks[0]
		args = append(args, "-c", fmt.Sprintf("hooks.%s=[{hooks=[{type=\"command\",command=%q,statusMessage=%q,timeout=3}]}]", event, hook.Command, hook.Status))
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	encoder := json.NewEncoder(stdin)
	if err = encoder.Encode(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]string{"name": "anytty-hook-loader-test", "version": "1"}, "capabilities": map[string]bool{"experimentalApi": true}}}); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var response struct {
			ID    int             `json:"id"`
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &response) == nil && response.ID == 1 {
			if len(response.Error) > 0 {
				t.Fatal("Codex initialize rejected")
			}
			break
		}
	}
	if err = encoder.Encode(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	if err = encoder.Encode(map[string]any{"id": 2, "method": "hooks/list", "params": map[string]any{"cwds": []string{dir}}}); err != nil {
		t.Fatal(err)
	}
	for scanner.Scan() {
		var response struct {
			ID     int `json:"id"`
			Result struct {
				Data []struct {
					Warnings []string `json:"warnings"`
					Errors   []struct {
						Path    string `json:"path"`
						Message string `json:"message"`
					} `json:"errors"`
					Hooks []struct {
						SourcePath    string `json:"sourcePath"`
						TrustStatus   string `json:"trustStatus"`
						EventName     string `json:"eventName"`
						StatusMessage string `json:"statusMessage"`
					} `json:"hooks"`
				} `json:"data"`
			} `json:"result"`
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &response) != nil || response.ID != 2 {
			continue
		}
		if len(response.Error) > 0 {
			t.Fatal("Codex hooks/list rejected")
		}
		seen := map[string]bool{}
		for _, entry := range response.Result.Data {

			for _, e := range entry.Errors {
				if e.Path == filepath.Join(config, "hooks.json") || e.Path == dir {
					t.Logf("fixture load error: %s", e.Message)
				}
			}
			for _, hook := range entry.Hooks {

				if hook.StatusMessage != managedMarker {
					continue
				}
				if hook.SourcePath != "/<session-flags>/config.toml" {
					continue
				}
				if hook.TrustStatus != "untrusted" {
					t.Fatalf("fixture hook unexpectedly %s", hook.TrustStatus)
				}
				seen[hook.EventName] = true
			}
		}
		if len(seen) != len(codexEvents) {
			t.Fatalf("Codex loader found %d of %d fixture hooks", len(seen), len(codexEvents))
		}
		return
	}
	t.Fatalf("Codex loader ended before response: %v", scanner.Err())
}
