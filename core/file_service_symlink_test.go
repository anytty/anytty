package core

import (
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
