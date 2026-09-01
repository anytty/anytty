package core

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileEntryResolvesSymlinkTargetTypeAndMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symbolic links requires additional Windows privileges")
	}

	root := t.TempDir()
	targetFile := filepath.Join(root, "target.txt")
	targetDir := filepath.Join(root, "target-dir")
	if err := os.WriteFile(targetFile, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(targetDir, 0o700); err != nil {
		t.Fatal(err)
	}

	fileLink := filepath.Join(root, "file-link")
	dirLink := filepath.Join(root, "dir-link")
	brokenLink := filepath.Join(root, "broken-link")
	if err := os.Symlink("target.txt", fileLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target-dir", dirLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", brokenLink); err != nil {
		t.Fatal(err)
	}

	linkedFileEntry, err := fileEntry(fileLink)
	if err != nil {
		t.Fatal(err)
	}
	if linkedFileEntry.Type != "file" || linkedFileEntry.LinkTarget != "target.txt" || linkedFileEntry.Size != 5 {
		t.Fatalf("file link entry = %+v", linkedFileEntry)
	}
	dirEntry, err := fileEntry(dirLink)
	if err != nil {
		t.Fatal(err)
	}
	if dirEntry.Type != "dir" || dirEntry.LinkTarget != "target-dir" {
		t.Fatalf("directory link entry = %+v", dirEntry)
	}
	brokenEntry, err := fileEntry(brokenLink)
	if err != nil {
		t.Fatal(err)
	}
	if brokenEntry.Type != "symlink" || brokenEntry.LinkTarget != "missing" {
		t.Fatalf("broken link entry = %+v", brokenEntry)
	}

	preview, err := filePreview(FilePreviewRequest{Path: fileLink, MaxBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	if string(preview.Content) != "hello" {
		t.Fatalf("preview content = %q", preview.Content)
	}
	wantDigest := sha256.Sum256([]byte("hello"))
	if string(preview.SHA256) != string(wantDigest[:]) {
		t.Fatalf("preview digest = %x, want %x", preview.SHA256, wantDigest)
	}
}

func TestFilePreviewResolvesSymlinkChainsAndRejectsLoops(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symbolic links requires additional Windows privileges")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	second := filepath.Join(root, "second")
	first := filepath.Join(root, "first")
	if err := os.WriteFile(target, []byte("chain"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", second); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("second", first); err != nil {
		t.Fatal(err)
	}
	preview, err := filePreview(FilePreviewRequest{Path: first, MaxBytes: 32})
	if err != nil || string(preview.Content) != "chain" {
		t.Fatalf("chain preview = %#v, %v", preview, err)
	}

	loopA := filepath.Join(root, "loop-a")
	loopB := filepath.Join(root, "loop-b")
	if err := os.Symlink("loop-b", loopA); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("loop-a", loopB); err != nil {
		t.Fatal(err)
	}
	if _, err := filePreview(FilePreviewRequest{Path: loopA, MaxBytes: 32}); err == nil {
		t.Fatal("symlink loop unexpectedly opened")
	}
}

func TestFilePreviewOpensHardLinkAsRegularFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "hard-link.txt")
	if err := os.WriteFile(target, []byte("shared inode"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(target, link); err != nil {
		t.Skipf("hard links are unavailable: %v", err)
	}

	entry, err := fileEntry(link)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Type != "file" || entry.LinkTarget != "" {
		t.Fatalf("hard link entry = %+v", entry)
	}
	preview, err := filePreview(FilePreviewRequest{Path: link, MaxBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	if string(preview.Content) != "shared inode" {
		t.Fatalf("preview content = %q", preview.Content)
	}
}
