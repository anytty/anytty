package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const (
	defaultUpdateRepository = "anytty/anytty"
	maxUpdateChecksumBytes  = 1 << 20
	maxUpdateArchiveBytes   = 512 << 20
	maxUpdateBinaryBytes    = 256 << 20
)

type updateReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type updateRelease struct {
	TagName    string               `json:"tag_name"`
	HTMLURL    string               `json:"html_url"`
	Draft      bool                 `json:"draft"`
	Prerelease bool                 `json:"prerelease"`
	Assets     []updateReleaseAsset `json:"assets"`
}

type updateCommandRuntime struct {
	client             *http.Client
	apiBase            string
	repository         string
	currentVersion     string
	goos               string
	goarch             string
	executable         func() (string, error)
	validateExecutable func(context.Context, string, string) error
	replaceExecutable  func(string, string) error
	allowHTTP          bool
}

type updateView struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Current       string `json:"current_version"`
	Latest        string `json:"latest_version"`
	Status        string `json:"status"`
	ReleaseURL    string `json:"release_url"`
	InstalledPath string `json:"installed_path,omitempty"`
	DaemonRestart bool   `json:"daemon_restarted"`
}

func defaultUpdateCommandRuntime() updateCommandRuntime {
	return updateCommandRuntime{
		client:             &http.Client{Timeout: 2 * time.Minute},
		apiBase:            "https://api.github.com",
		repository:         defaultUpdateRepository,
		currentVersion:     version,
		goos:               runtime.GOOS,
		goarch:             runtime.GOARCH,
		executable:         currentExecutablePath,
		validateExecutable: validateUpdateExecutable,
		replaceExecutable:  replaceUpdateExecutable,
	}
}

func newUpdateCommand(updateRuntime updateCommandRuntime) *cobra.Command {
	var checkOnly bool
	var includePrerelease bool
	var requestedVersion string
	var force bool
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "update",
		Short: "Check for and install AnyTTY releases from GitHub",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if checkOnly && force {
				return usageCLIError("--check and --force cannot be used together")
			}
			releases, err := updateRuntime.fetchReleases(cmd.Context())
			if err != nil {
				return classifyCLIError(err)
			}
			release, err := selectUpdateRelease(releases, updateRuntime.currentVersion, requestedVersion, includePrerelease)
			if err != nil {
				return usageCLIError(err.Error())
			}
			view := updateView{
				SchemaVersion: 1,
				Kind:          "update",
				Current:       updateRuntime.currentVersion,
				Latest:        release.TagName,
				Status:        "update_available",
				ReleaseURL:    release.HTMLURL,
			}
			if !force && !updateVersionIsNewer(release.TagName, updateRuntime.currentVersion) {
				view.Status = "up_to_date"
				return writeUpdateView(cmd, view, jsonOutput)
			}
			if checkOnly {
				return writeUpdateView(cmd, view, jsonOutput)
			}
			installedPath, err := updateRuntime.install(cmd.Context(), release)
			if err != nil {
				return classifyCLIError(err)
			}
			view.Status = "updated"
			view.InstalledPath = installedPath
			return writeUpdateView(cmd, view, jsonOutput)
		},
	}
	command.Flags().BoolVar(&checkOnly, "check", false, "check without installing")
	command.Flags().BoolVar(&includePrerelease, "prerelease", false, "include prerelease versions")
	command.Flags().StringVar(&requestedVersion, "version", "", "install a specific release tag")
	command.Flags().BoolVar(&force, "force", false, "reinstall or downgrade to the selected version")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func writeUpdateView(cmd *cobra.Command, view updateView, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
	}
	fields := []cliField{
		{Label: "Current", Value: view.Current},
		{Label: "Latest", Value: view.Latest},
		{Label: "Status", Value: strings.ReplaceAll(view.Status, "_", " ")},
		{Label: "Release", Value: view.ReleaseURL},
	}
	if view.InstalledPath != "" {
		fields = append(fields,
			cliField{Label: "Installed", Value: view.InstalledPath},
			cliField{Label: "Daemon", Value: "not restarted; running terminals are unchanged"},
		)
	}
	return writeCLIFields(cmd.OutOrStdout(), fields...)
}

func (updateRuntime updateCommandRuntime) fetchReleases(ctx context.Context) ([]updateRelease, error) {
	parts := strings.Split(updateRuntime.repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid GitHub repository %q", updateRuntime.repository)
	}
	endpoint := strings.TrimRight(updateRuntime.apiBase, "/") + "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/releases?per_page=100"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "anytty/"+updateRuntime.currentVersion)
	response, err := updateRuntime.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query GitHub Releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return nil, fmt.Errorf("query GitHub Releases: HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	var releases []updateRelease
	if err := decoder.Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode GitHub Releases: %w", err)
	}
	return releases, nil
}

func selectUpdateRelease(releases []updateRelease, currentVersion, requestedVersion string, includePrerelease bool) (updateRelease, error) {
	requestedVersion = normalizeUpdateVersion(requestedVersion)
	if requestedVersion != "" && !semver.IsValid(requestedVersion) {
		return updateRelease{}, fmt.Errorf("invalid release version %q", requestedVersion)
	}
	current := normalizeUpdateVersion(currentVersion)
	if current == "" || !semver.IsValid(current) || semver.Prerelease(current) != "" {
		includePrerelease = true
	}
	var selected updateRelease
	for _, release := range releases {
		tag := normalizeUpdateVersion(release.TagName)
		if release.Draft || !semver.IsValid(tag) {
			continue
		}
		if requestedVersion != "" {
			if tag == requestedVersion {
				return release, nil
			}
			continue
		}
		if (release.Prerelease || semver.Prerelease(tag) != "") && !includePrerelease {
			continue
		}
		if selected.TagName == "" || semver.Compare(tag, normalizeUpdateVersion(selected.TagName)) > 0 {
			selected = release
		}
	}
	if requestedVersion != "" {
		return updateRelease{}, fmt.Errorf("release %s was not found", requestedVersion)
	}
	if selected.TagName == "" {
		return updateRelease{}, fmt.Errorf("no eligible GitHub Release was found")
	}
	return selected, nil
}

func normalizeUpdateVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "dev" {
		return ""
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return value
}

func updateVersionIsNewer(candidate, current string) bool {
	candidate = normalizeUpdateVersion(candidate)
	current = normalizeUpdateVersion(current)
	return semver.IsValid(candidate) && (!semver.IsValid(current) || semver.Compare(candidate, current) > 0)
}

func (updateRuntime updateCommandRuntime) install(ctx context.Context, release updateRelease) (string, error) {
	extension := ".tar.gz"
	executableName := "anytty"
	if updateRuntime.goos == "windows" {
		extension = ".zip"
		executableName = "anytty.exe"
	} else if updateRuntime.goos != "darwin" && updateRuntime.goos != "linux" {
		return "", fmt.Errorf("self-update is not supported on %s", updateRuntime.goos)
	}
	if updateRuntime.goarch != "amd64" && updateRuntime.goarch != "arm64" {
		return "", fmt.Errorf("self-update is not supported on %s/%s", updateRuntime.goos, updateRuntime.goarch)
	}
	archiveBase := fmt.Sprintf("anytty-%s-%s-%s", release.TagName, updateRuntime.goos, updateRuntime.goarch)
	archiveName := archiveBase + extension
	archiveAsset, ok := findUpdateAsset(release.Assets, archiveName)
	if !ok {
		return "", fmt.Errorf("release %s does not contain %s", release.TagName, archiveName)
	}
	checksumAsset, ok := findUpdateAsset(release.Assets, "SHA256SUMS")
	if !ok {
		return "", fmt.Errorf("release %s does not contain SHA256SUMS", release.TagName)
	}

	checksumData, checksumDigest, err := updateRuntime.downloadBytes(ctx, checksumAsset, maxUpdateChecksumBytes)
	if err != nil {
		return "", err
	}
	if err := verifyGitHubAssetDigest(checksumAsset, checksumDigest); err != nil {
		return "", err
	}
	expectedDigest, err := checksumForUpdateAsset(checksumData, archiveName)
	if err != nil {
		return "", err
	}

	workDir, err := os.MkdirTemp("", "anytty-update-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workDir)
	archivePath := filepath.Join(workDir, archiveName)
	archiveDigest, err := updateRuntime.downloadFile(ctx, archiveAsset, archivePath, maxUpdateArchiveBytes)
	if err != nil {
		return "", err
	}
	if archiveDigest != expectedDigest {
		return "", fmt.Errorf("SHA-256 verification failed for %s", archiveName)
	}
	if err := verifyGitHubAssetDigest(archiveAsset, archiveDigest); err != nil {
		return "", err
	}

	targetPath, err := updateRuntime.executable()
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}
	targetPath, err = filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve current executable symlinks: %w", err)
	}
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		return "", fmt.Errorf("inspect current executable: %w", err)
	}
	if !targetInfo.Mode().IsRegular() {
		return "", fmt.Errorf("current executable is not a regular file: %s", targetPath)
	}
	candidate, err := os.CreateTemp(filepath.Dir(targetPath), ".anytty-update-*")
	if err != nil {
		return "", fmt.Errorf("create update beside current executable: %w", err)
	}
	candidatePath := candidate.Name()
	keepCandidate := false
	defer func() {
		_ = candidate.Close()
		if !keepCandidate {
			_ = os.Remove(candidatePath)
		}
	}()
	memberName := archiveBase + "/" + executableName
	if err := extractUpdateExecutable(archivePath, extension, memberName, candidate); err != nil {
		return "", err
	}
	if err := candidate.Sync(); err != nil {
		return "", err
	}
	if err := candidate.Close(); err != nil {
		return "", err
	}
	mode := targetInfo.Mode().Perm()
	if mode&0o111 == 0 {
		mode = 0o755
	}
	if err := os.Chmod(candidatePath, mode); err != nil {
		return "", err
	}
	if err := updateRuntime.validateExecutable(ctx, candidatePath, release.TagName); err != nil {
		return "", err
	}
	if err := updateRuntime.replaceExecutable(candidatePath, targetPath); err != nil {
		return "", err
	}
	keepCandidate = true
	return targetPath, nil
}

func findUpdateAsset(assets []updateReleaseAsset, name string) (updateReleaseAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name && asset.BrowserDownloadURL != "" {
			return asset, true
		}
	}
	return updateReleaseAsset{}, false
}

func (updateRuntime updateCommandRuntime) downloadBytes(ctx context.Context, asset updateReleaseAsset, limit int64) ([]byte, string, error) {
	response, err := updateRuntime.download(ctx, asset)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	hash := sha256.New()
	data, err := io.ReadAll(io.LimitReader(io.TeeReader(response.Body, hash), limit+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > limit {
		return nil, "", fmt.Errorf("release asset %s exceeds the size limit", asset.Name)
	}
	return data, hex.EncodeToString(hash.Sum(nil)), nil
}

func (updateRuntime updateCommandRuntime) downloadFile(ctx context.Context, asset updateReleaseAsset, path string, limit int64) (string, error) {
	response, err := updateRuntime.download(ctx, asset)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, limit+1))
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written > limit {
		return "", fmt.Errorf("release asset %s exceeds the size limit", asset.Name)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (updateRuntime updateCommandRuntime) download(ctx context.Context, asset updateReleaseAsset) (*http.Response, error) {
	parsed, err := url.Parse(asset.BrowserDownloadURL)
	if err != nil || parsed.Host == "" || (!updateRuntime.allowHTTP && parsed.Scheme != "https") {
		return nil, fmt.Errorf("release asset %s has an invalid download URL", asset.Name)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "anytty/"+updateRuntime.currentVersion)
	response, err := updateRuntime.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", asset.Name, err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("download %s: HTTP %d", asset.Name, response.StatusCode)
	}
	return response, nil
}

func verifyGitHubAssetDigest(asset updateReleaseAsset, actual string) error {
	if asset.Digest == "" {
		return nil
	}
	expected, found := strings.CutPrefix(strings.ToLower(strings.TrimSpace(asset.Digest)), "sha256:")
	if !found || len(expected) != sha256.Size*2 {
		return fmt.Errorf("release asset %s has an invalid GitHub digest", asset.Name)
	}
	if expected != actual {
		return fmt.Errorf("GitHub digest verification failed for %s", asset.Name)
	}
	return nil
}

func checksumForUpdateAsset(data []byte, name string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		digest := strings.ToLower(fields[0])
		decoded, err := hex.DecodeString(digest)
		if err != nil || len(decoded) != sha256.Size {
			return "", fmt.Errorf("SHA256SUMS contains an invalid digest for %s", name)
		}
		return digest, nil
	}
	return "", fmt.Errorf("SHA256SUMS does not contain %s", name)
}

func extractUpdateExecutable(archivePath, extension, memberName string, destination *os.File) error {
	switch extension {
	case ".tar.gz":
		return extractUpdateExecutableTar(archivePath, memberName, destination)
	case ".zip":
		return extractUpdateExecutableZip(archivePath, memberName, destination)
	default:
		return fmt.Errorf("unsupported release archive %s", extension)
	}
}

func extractUpdateExecutableTar(archivePath, memberName string, destination io.Writer) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open release archive: %w", err)
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read release archive: %w", err)
		}
		if filepath.ToSlash(header.Name) != memberName {
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Size < 1 || header.Size > maxUpdateBinaryBytes {
			return fmt.Errorf("release archive contains an invalid %s", memberName)
		}
		written, err := io.Copy(destination, io.LimitReader(reader, header.Size))
		if err != nil {
			return err
		}
		if written != header.Size {
			return fmt.Errorf("release archive contains a truncated %s", memberName)
		}
		return nil
	}
	return fmt.Errorf("release archive does not contain %s", memberName)
}

func extractUpdateExecutableZip(archivePath, memberName string, destination io.Writer) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open release archive: %w", err)
	}
	defer archive.Close()
	for _, member := range archive.File {
		if filepath.ToSlash(member.Name) != memberName {
			continue
		}
		if member.FileInfo().Mode().IsRegular() == false || member.UncompressedSize64 < 1 || member.UncompressedSize64 > maxUpdateBinaryBytes {
			return fmt.Errorf("release archive contains an invalid %s", memberName)
		}
		source, err := member.Open()
		if err != nil {
			return err
		}
		written, copyErr := io.Copy(destination, io.LimitReader(source, int64(member.UncompressedSize64)))
		closeErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if uint64(written) != member.UncompressedSize64 {
			return fmt.Errorf("release archive contains a truncated %s", memberName)
		}
		return nil
	}
	return fmt.Errorf("release archive does not contain %s", memberName)
}

func currentExecutablePath() (string, error) { return os.Executable() }

func validateUpdateExecutable(ctx context.Context, path, expectedVersion string) error {
	command := exec.CommandContext(ctx, path, "--version")
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("validate downloaded executable: %w", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) == 0 || fields[len(fields)-1] != expectedVersion {
		return fmt.Errorf("downloaded executable reports unexpected version %q", strings.TrimSpace(string(output)))
	}
	return nil
}
