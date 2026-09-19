package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseOptionsRequiresSocketAndListener(t *testing.T) {
	if _, err := parseOptions(nil); err == nil || !strings.Contains(err.Error(), "--socket") {
		t.Fatalf("missing socket error = %v", err)
	}
	if _, err := parseOptions([]string{"--socket", "/tmp/daemon.sock"}); err == nil || !strings.Contains(err.Error(), "--listen") {
		t.Fatalf("missing listener error = %v", err)
	}
	opts, err := parseOptions([]string{"--socket", "/tmp/daemon.sock", "--listen", "tcp:127.0.0.1:0", "--listen", "unix:/tmp/access.sock"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Listeners) != 2 || opts.Listeners[0].Network != "tcp" || opts.Listeners[1].Network != "unix" {
		t.Fatalf("listeners = %+v", opts.Listeners)
	}
}

func TestParseOptionsConfigFileAndFlagPrecedence(t *testing.T) {
	dir := t.TempDir()
	tokenPath := writeFile(t, dir, "token", "config-token\n")
	configPath := writeFile(t, dir, "access.json", `{
  "socket": "/config/daemon.sock",
  "listen": ["tcp:127.0.0.1:7331"],
  "allow": ["127.0.0.1/32"],
  "pair_token_file": "`+tokenPath+`"
}`)
	opts, err := parseOptions([]string{"--config", configPath})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Socket != "/config/daemon.sock" || len(opts.Listeners) != 1 || len(opts.Allow) != 1 {
		t.Fatalf("config merge = %+v", opts)
	}
	if string(opts.PairToken) != "config-token" || opts.PairTokenFrom != tokenPath {
		t.Fatalf("token = %q from %q", opts.PairToken, opts.PairTokenFrom)
	}

	overrideToken := writeFile(t, dir, "override-token", "flag-token")
	overridden, err := parseOptions([]string{
		"--config", configPath,
		"--socket", "/flag/daemon.sock",
		"--listen", "unix:/flag/access.sock",
		"--allow", "10.0.0.0/8",
		"--pair-token-file", overrideToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if overridden.Socket != "/flag/daemon.sock" || overridden.Listeners[0].Network != "unix" {
		t.Fatalf("flag override = %+v", overridden)
	}
	if len(overridden.Allow) != 1 || overridden.Allow[0] != "10.0.0.0/8" {
		t.Fatalf("allow override = %v", overridden.Allow)
	}
	if string(overridden.PairToken) != "flag-token" {
		t.Fatalf("token override = %q", overridden.PairToken)
	}
}

func TestParseOptionsRejectsBadConfigAndToken(t *testing.T) {
	dir := t.TempDir()
	badConfig := writeFile(t, dir, "bad.json", `{"socket": "/s", "listen": ["tcp:1"], "unknown": true}`)
	if _, err := parseOptions([]string{"--config", badConfig}); err == nil {
		t.Fatal("unknown config field unexpectedly accepted")
	}
	emptyToken := writeFile(t, dir, "empty-token", "\n")
	if _, err := parseOptions([]string{"--socket", "/s", "--listen", "tcp:127.0.0.1:0", "--pair-token-file", emptyToken}); err == nil {
		t.Fatal("empty token file unexpectedly accepted")
	}
	if _, err := parseOptions([]string{"--socket", "/s", "--listen", "tcp:127.0.0.1:0", "--allow", "not-a-network"}); err == nil {
		t.Fatal("bad allow entry unexpectedly accepted")
	}
}

func TestParseOptionsAcceptsDirectRouteWithoutRelayListener(t *testing.T) {
	for _, key := range []string{"ANYTTY_DIRECT_LISTEN", "ANYTTY_DIRECT_SIGNALING_LISTEN", "ANYTTY_DIRECT_ICE_TCP_LISTEN"} {
		t.Setenv(key, "")
	}
	opts, err := parseOptions([]string{"--socket", "/tmp/daemon.sock", "--route", "127.0.0.1:7331"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Listeners) != 0 || opts.Route != "127.0.0.1:7331" || !opts.directConfigured() {
		t.Fatalf("route-only options = %+v", opts)
	}
	if _, err := parseOptions([]string{"--socket", "/tmp/daemon.sock", "--route", "not-a-route"}); err == nil {
		t.Fatal("invalid route unexpectedly accepted")
	}
}

func TestParseOptionsConfigFileRoute(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "access.json", `{
  "socket": "/config/daemon.sock",
  "route": "0.0.0.0:7331"
}`)
	opts, err := parseOptions([]string{"--config", configPath})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Route != "0.0.0.0:7331" {
		t.Fatalf("config route merge = %+v", opts)
	}
	overridden, err := parseOptions([]string{"--config", configPath, "--route", "127.0.0.1:7442"})
	if err != nil {
		t.Fatal(err)
	}
	if overridden.Route != "127.0.0.1:7442" {
		t.Fatalf("route override = %+v", overridden)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 || !strings.HasPrefix(stdout.String(), "anytty-access ") {
		t.Fatalf("run --version code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunRejectsMissingConfigurationOnStderr(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run(nil, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "anytty-access:") {
		t.Fatalf("run(nil) code=%d stderr=%q", code, stderr.String())
	}
}
