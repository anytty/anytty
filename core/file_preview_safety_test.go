package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePreviewRejectsUnsupportedAndOversizedSources(t *testing.T) {
	root := t.TempDir()
	binaryPath := filepath.Join(root, "payload.bin")
	if err := os.WriteFile(binaryPath, bytes.Repeat([]byte{0, 1, 2, 3}, 200), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := filePreview(FilePreviewRequest{Path: binaryPath}); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("binary preview error = %v", err)
	}

	oversizedPath := filepath.Join(root, "oversized.txt")
	file, err := os.Create(oversizedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(filePreviewMaxSourceBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := filePreview(FilePreviewRequest{Path: oversizedPath}); err == nil || !strings.Contains(err.Error(), "source exceeds") {
		t.Fatalf("oversized preview error = %v", err)
	}
}

func TestFilePreviewTruncatesTextButRejectsTruncatedImages(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(textPath, []byte("abcdefghij"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := filePreview(FilePreviewRequest{Path: textPath, MaxBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if string(preview.Content) != "abcde" || !preview.Truncated || preview.MIMEType != "text/plain" {
		t.Fatalf("text preview = %#v", preview)
	}

	imagePath := filepath.Join(root, "large.png")
	pngPrefix := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if err := os.WriteFile(imagePath, append(pngPrefix, bytes.Repeat([]byte{1}, 32)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := filePreview(FilePreviewRequest{Path: imagePath, MaxBytes: 8}); err == nil || !strings.Contains(err.Error(), "image preview exceeds") {
		t.Fatalf("truncated image preview error = %v", err)
	}
}
