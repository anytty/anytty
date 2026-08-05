package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/shared/securefs"
)

func TestFileCommandRealDaemonLifecycleAndTransfers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath, _, closeServer := startCLIEndpointServer(t)
	defer closeServer()
	if err := endpointdomain.Save("", endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion, Default: endpointdomain.DefaultEndpointID,
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			endpointdomain.DefaultEndpointID: testLocalEndpoint(endpointdomain.DefaultEndpointID, "Local", socketPath, endpointdomain.ConnectAuto, true),
		},
	}); err != nil {
		t.Fatal(err)
	}
	remoteRoot := t.TempDir()
	sourcePath := filepath.Join(remoteRoot, "source.txt")
	if err := os.WriteFile(sourcePath, []byte("remote-source\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	listing := executeFileCLI(t, "file", "list", "local", remoteRoot, "--all", "--json")
	if !strings.Contains(listing, `"kind":"file_list"`) || !strings.Contains(listing, `"name":"source.txt"`) {
		t.Fatalf("unexpected file list: %s", listing)
	}
	stat := executeFileCLI(t, "file", "stat", "local", sourcePath, "--json")
	if !strings.Contains(stat, `"type":"file"`) || !strings.Contains(stat, `"size":14`) {
		t.Fatalf("unexpected file stat: %s", stat)
	}
	if cat := executeFileCLI(t, "file", "cat", "local", sourcePath); cat != "remote-source\n" {
		t.Fatalf("file cat = %q", cat)
	}

	localDownload := filepath.Join(t.TempDir(), "download.txt")
	downloaded := executeFileCLI(t, "file", "download", "local", sourcePath, localDownload, "--json")
	if !strings.Contains(downloaded, `"kind":"file_download"`) || !strings.Contains(downloaded, `"size":14`) {
		t.Fatalf("unexpected download result: %s", downloaded)
	}
	if payload, err := os.ReadFile(localDownload); err != nil || string(payload) != "remote-source\n" {
		t.Fatalf("downloaded content = %q, %v", payload, err)
	}
	if info, err := os.Stat(localDownload); err != nil || !securefs.IsPrivateFile(localDownload, info) {
		t.Fatalf("download permissions = %v, %v", info, err)
	}

	largeContent := bytes.Repeat([]byte("anytty-upload-window-"), 20000)
	localUpload := filepath.Join(t.TempDir(), "upload.bin")
	if err := os.WriteFile(localUpload, largeContent, 0o600); err != nil {
		t.Fatal(err)
	}
	remoteUpload := filepath.Join(remoteRoot, "upload.bin")
	uploaded := executeFileCLI(t, "file", "upload", "local", localUpload, remoteUpload, "--json")
	if !strings.Contains(uploaded, `"kind":"file_upload"`) || !strings.Contains(uploaded, `"size":420000`) {
		t.Fatalf("unexpected upload result: %s", uploaded)
	}
	if payload, err := os.ReadFile(remoteUpload); err != nil || !bytes.Equal(payload, largeContent) {
		t.Fatalf("uploaded content length = %d, %v", len(payload), err)
	}

	dir := filepath.Join(remoteRoot, "nested", "target")
	executeFileCLI(t, "file", "mkdir", "local", dir, "--parents")
	renamed := filepath.Join(remoteRoot, "renamed.txt")
	executeFileCLI(t, "file", "rename", "local", sourcePath, renamed)
	executeFileCLI(t, "file", "copy", "local", renamed, remoteUpload, dir)
	movedDir := filepath.Join(remoteRoot, "moved")
	executeFileCLI(t, "file", "mkdir", "local", movedDir)
	executeFileCLI(t, "file", "move", "local", filepath.Join(dir, "renamed.txt"), movedDir)
	executeFileCLI(t, "file", "remove", "local", filepath.Join(movedDir, "renamed.txt"), filepath.Join(dir, "upload.bin"))

	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"file", "copy", "local", renamed, filepath.Join(remoteRoot, "missing"), dir, "--json"})
	if err := command.Execute(); cliExitCode(err) != 8 {
		t.Fatalf("partial copy error = %v, exit=%d, output=%s", err, cliExitCode(err), output.String())
	}
	if !strings.Contains(output.String(), `"success":true`) || !strings.Contains(output.String(), `"success":false`) {
		t.Fatalf("partial copy lost per-item results: %s", output.String())
	}
	executeFileCLI(t, "file", "remove", "local", filepath.Join(remoteRoot, "nested"), filepath.Join(remoteRoot, "moved"), "--recursive")
}

func executeFileCLI(t *testing.T, args ...string) string {
	t.Helper()
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetIn(strings.NewReader(""))
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("anytty %s: %v", strings.Join(args, " "), err)
	}
	return output.String()
}
