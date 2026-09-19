package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/access/files"
	poolprovider "github.com/anytty/anytty/access/provider/pool"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	accessserver "github.com/anytty/anytty/access/server"
	corev2 "github.com/anytty/anytty/pool/core"
	"github.com/anytty/anytty/shared/securefs"
)

func TestFileCommandRealPoolLifecycleAndTransfers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath, closeServer := startCLIAccessStack(t)
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

func TestFileDownloadAcrossCreditWindows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath, closeServer := startCLIAccessStack(t)
	defer closeServer()
	if err := endpointdomain.Save("", endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion, Default: endpointdomain.DefaultEndpointID,
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			endpointdomain.DefaultEndpointID: testLocalEndpoint(endpointdomain.DefaultEndpointID, "Local", socketPath, endpointdomain.ConnectAuto, true),
		},
	}); err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{(1 << 20) - 1, 1 << 20, (1 << 20) + 1, (3 << 20) + 17} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			content := make([]byte, size)
			for index := range content {
				content[index] = byte(index % 251)
			}
			source := filepath.Join(t.TempDir(), "source.bin")
			destination := filepath.Join(t.TempDir(), "download.bin")
			if err := os.WriteFile(source, content, 0o600); err != nil {
				t.Fatal(err)
			}
			executeFileCLI(t, "--timeout", "5s", "file", "download", "local", source, destination, "--json")
			actual, err := os.ReadFile(destination)
			if err != nil || !bytes.Equal(actual, content) {
				t.Fatalf("download length=%d want=%d error=%v", len(actual), size, err)
			}
		})
	}
}

// startCLIAccessStack 启动 Phase 3 拓扑的本地栈：pool 在 <sock>.provider，
// access 在 canonical <sock> 上终结 file/storage/proxy 并路由终端。
func startCLIAccessStack(t *testing.T) (string, func()) {
	t.Helper()
	socketPath := filepath.Join(t.TempDir(), "anytty.sock")
	providerSocket := socketPath + ".provider"
	server := newCoreV2TestServer(corev2.WithSocketPath(providerSocket))
	stopProvider := startCoreV2ProviderServer(t, server, providerSocket)
	ctx, cancel := context.WithCancel(context.Background())
	_ = stopProvider
	access, err := accessserver.New(accessserver.Config{
		Socket: socketPath,
		Files:  files.Config{TransferDir: filepath.Join(t.TempDir(), "transfers")},
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return poolprovider.DialTerminal(dialCtx, providerSocket)
		},
	})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	go func() { _ = access.Serve(ctx) }()
	if err := waitForSocket(socketPath, 2*time.Second, func() error {
		return probeV3Socket(socketPath)
	}); err != nil {
		_ = access.Close()
		cancel()
		t.Fatalf("access did not become ready: %v", err)
	}
	return socketPath, func() {
		_ = access.Close()
		cancel()
		stopProvider()
		_ = server.Shutdown(context.Background())
	}
}
