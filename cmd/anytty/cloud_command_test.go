package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCloudStatusDisableEnableOffline(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	socketPath := filepath.Join(root, "anytty.sock")
	logPath := filepath.Join(root, "anytty.log")

	status := executeCloudStatusJSON(t, "--socket", socketPath, "--log-file", logPath, "cloud", "status", "--json")
	if status.State != "not_enrolled" || !status.Enabled || status.Enrolled || status.Running {
		t.Fatalf("initial cloud status = %#v", status)
	}
	if _, err := os.Stat(daemonRecordPath(socketPath)); !os.IsNotExist(err) {
		t.Fatalf("cloud status unexpectedly touched daemon runtime record: %v", err)
	}

	disabled := executeCloudStatusJSON(t, "--socket", socketPath, "--log-file", logPath, "cloud", "disable", "--json")
	if disabled.State != "disabled" || disabled.Enabled || disabled.Running {
		t.Fatalf("disabled cloud status = %#v", disabled)
	}
	if _, err := os.Stat(v3CloudDisabledPath()); err != nil {
		t.Fatalf("disabled marker was not written: %v", err)
	}

	enabled := executeCloudStatusJSON(t, "--socket", socketPath, "--log-file", logPath, "cloud", "enable", "--json")
	if enabled.State != "not_enrolled" || !enabled.Enabled || enabled.Enrolled || enabled.Running {
		t.Fatalf("enabled cloud status = %#v", enabled)
	}
	if _, err := os.Stat(v3CloudDisabledPath()); !os.IsNotExist(err) {
		t.Fatalf("disabled marker remains after enable: %v", err)
	}
}

func TestResolveOfficialControllerUsesPublicHTTPSPort(t *testing.T) {
	address, serverName, err := resolveController("https://cloud.anytty.com", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if address != "cloud.anytty.com:443" || serverName != "cloud.anytty.com" {
		t.Fatalf("official controller = %s / %s", address, serverName)
	}
}

func executeCloudStatusJSON(t *testing.T, args ...string) cloudStatusView {
	t.Helper()
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("anytty %v: %v\n%s", args, err, output.String())
	}
	var status cloudStatusView
	if err := json.Unmarshal(output.Bytes(), &status); err != nil {
		t.Fatalf("decode cloud status %q: %v", output.String(), err)
	}
	return status
}
