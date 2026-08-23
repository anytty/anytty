package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCmdIncludesUpdateCommand(t *testing.T) {
	command, _, err := newRootCmd().Find([]string{"update"})
	if err != nil || command == nil || command.Name() != "update" {
		t.Fatalf("root update command = %#v, %v", command, err)
	}
}

func TestSelectUpdateReleaseUsesCurrentChannel(t *testing.T) {
	releases := []updateRelease{
		{TagName: "v1.2.0-beta.1", Prerelease: true},
		{TagName: "v1.1.1"},
		{TagName: "v1.1.0"},
		{TagName: "v9.0.0", Draft: true},
	}
	stable, err := selectUpdateRelease(releases, "v1.1.0", "", false)
	if err != nil || stable.TagName != "v1.1.1" {
		t.Fatalf("stable release = %#v, %v", stable, err)
	}
	beta, err := selectUpdateRelease(releases, "v1.1.0-beta.1", "", false)
	if err != nil || beta.TagName != "v1.2.0-beta.1" {
		t.Fatalf("beta release = %#v, %v", beta, err)
	}
	explicit, err := selectUpdateRelease(releases, "v1.1.0", "1.2.0-beta.1", false)
	if err != nil || explicit.TagName != "v1.2.0-beta.1" {
		t.Fatalf("explicit release = %#v, %v", explicit, err)
	}
}

func TestUpdateCheckDoesNotReplaceExecutable(t *testing.T) {
	fixture := newUpdateFixture(t, "v1.1.0", []byte("new anytty"), "")
	target := filepath.Join(t.TempDir(), "anytty")
	if err := os.WriteFile(target, []byte("old anytty"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtime := fixture.runtime(target, "v1.0.0")
	runtime.replaceExecutable = func(_, _ string) error {
		t.Fatal("--check must not replace the executable")
		return nil
	}
	command := newUpdateCommand(runtime)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--check"})
	if err := command.Execute(); err != nil {
		t.Fatalf("update --check: %v", err)
	}
	if !strings.Contains(output.String(), "update available") {
		t.Fatalf("check output = %q", output.String())
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "old anytty" {
		t.Fatalf("target after check = %q, %v", data, err)
	}
}

func TestUpdateInstallsVerifiedReleaseWithoutRestartingDaemon(t *testing.T) {
	fixture := newUpdateFixture(t, "v1.1.0", []byte("new anytty"), "")
	target := filepath.Join(t.TempDir(), "anytty")
	if err := os.WriteFile(target, []byte("old anytty"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtime := fixture.runtime(target, "v1.0.0")
	command := newUpdateCommand(runtime)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "new anytty" {
		t.Fatalf("installed executable = %q, %v", data, err)
	}
	if !strings.Contains(output.String(), "not restarted; running terminals are unchanged") {
		t.Fatalf("update output = %q", output.String())
	}
}

func TestUpdateRejectsChecksumMismatchWithoutReplacingExecutable(t *testing.T) {
	fixture := newUpdateFixture(t, "v1.1.0", []byte("new anytty"), strings.Repeat("0", 64))
	target := filepath.Join(t.TempDir(), "anytty")
	if err := os.WriteFile(target, []byte("old anytty"), 0o755); err != nil {
		t.Fatal(err)
	}
	runtime := fixture.runtime(target, "v1.0.0")
	command := newUpdateCommand(runtime)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	command.SetArgs(nil)
	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "SHA-256 verification failed") {
		t.Fatalf("checksum mismatch error = %v", err)
	}
	if data, readErr := os.ReadFile(target); readErr != nil || string(data) != "old anytty" {
		t.Fatalf("target after rejected update = %q, %v", data, readErr)
	}
}

func TestUpdateReportsUpToDateAsJSON(t *testing.T) {
	fixture := newUpdateFixture(t, "v1.1.0", []byte("new anytty"), "")
	target := filepath.Join(t.TempDir(), "anytty")
	if err := os.WriteFile(target, []byte("old anytty"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := newUpdateCommand(fixture.runtime(target, "v1.1.0"))
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("update --json: %v", err)
	}
	var view updateView
	if err := json.Unmarshal(output.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Status != "up_to_date" || view.DaemonRestart {
		t.Fatalf("update view = %#v", view)
	}
}

func TestExtractUpdateExecutableZip(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "anytty.zip")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(archiveFile)
	member, err := zipWriter.Create("anytty-v1.1.0-windows-amd64/anytty.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := member.Write([]byte("windows anytty")); err != nil {
		t.Fatal(err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := extractUpdateExecutableZip(archivePath, "anytty-v1.1.0-windows-amd64/anytty.exe", &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "windows anytty" {
		t.Fatalf("extracted executable = %q", output.String())
	}
}

type updateFixture struct {
	server  *httptest.Server
	tag     string
	binary  []byte
	archive []byte
	sums    []byte
}

func newUpdateFixture(t *testing.T, tag string, binary []byte, checksumOverride string) *updateFixture {
	t.Helper()
	archiveBase := "anytty-" + tag + "-darwin-arm64"
	archive := updateTarGZ(t, archiveBase+"/anytty", binary)
	archiveDigest := updateTestDigest(archive)
	checksum := archiveDigest
	if checksumOverride != "" {
		checksum = checksumOverride
	}
	sums := []byte(fmt.Sprintf("%s  %s.tar.gz\n", checksum, archiveBase))
	fixture := &updateFixture{tag: tag, binary: binary, archive: archive, sums: sums}
	fixture.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/anytty/anytty/releases":
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode([]updateRelease{{
				TagName: tag, HTMLURL: fixture.server.URL + "/release/" + tag, Prerelease: strings.Contains(tag, "-"),
				Assets: []updateReleaseAsset{
					{Name: archiveBase + ".tar.gz", BrowserDownloadURL: fixture.server.URL + "/assets/archive", Digest: "sha256:" + archiveDigest},
					{Name: "SHA256SUMS", BrowserDownloadURL: fixture.server.URL + "/assets/sums", Digest: "sha256:" + updateTestDigest(sums)},
				},
			}})
		case "/assets/archive":
			_, _ = writer.Write(archive)
		case "/assets/sums":
			_, _ = writer.Write(sums)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (fixture *updateFixture) runtime(target, current string) updateCommandRuntime {
	return updateCommandRuntime{
		client:         fixture.server.Client(),
		apiBase:        fixture.server.URL,
		repository:     defaultUpdateRepository,
		currentVersion: current,
		goos:           "darwin",
		goarch:         "arm64",
		executable:     func() (string, error) { return target, nil },
		validateExecutable: func(_ context.Context, path, expectedVersion string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if string(data) != string(fixture.binary) || expectedVersion != fixture.tag {
				return fmt.Errorf("unexpected candidate %q for %s", data, expectedVersion)
			}
			return nil
		},
		replaceExecutable: os.Rename,
		allowHTTP:         true,
	}
}

func updateTarGZ(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	compressed := gzip.NewWriter(&output)
	archive := tar.NewWriter(compressed)
	if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func updateTestDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
