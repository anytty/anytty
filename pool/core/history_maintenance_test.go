package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/anytty/anytty/pool/core/history/linehist"
)

func seedMaintenanceHistory(t *testing.T, dir, terminalID string) {
	t.Helper()
	file, err := linehist.OpenCompressedLineFile(dir, terminalID, linehist.CompressedLineFileOptions{Compression: HistoryCompressionZstd})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.AppendLines([]linehist.Line{{Runs: []linehist.Run{{Text: terminalID}}, HardEnd: true}}); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryMaintenanceDeleteTerminalAndAll(t *testing.T) {
	dir := t.TempDir()
	for _, terminalID := range []string{"alpha", "beta"} {
		seedMaintenanceHistory(t, dir, terminalID)
	}
	maintenance := NewHistoryMaintenance(dir)

	removed, err := maintenance.DeleteTerminal("alpha")
	if err != nil || removed != 2 {
		t.Fatalf("delete terminal removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "alpha.logical-lines.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("alpha history remains: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "beta.logical-lines.bin")); err != nil {
		t.Fatalf("beta history was deleted: %v", err)
	}

	removed, err = maintenance.DeleteAll()
	if err != nil || removed != 2 {
		t.Fatalf("delete all removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "beta.logical-lines.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("beta history remains: %v", err)
	}
}

func TestHistoryMaintenanceDeleteObsolete(t *testing.T) {
	obsoleteDir := filepath.Join(t.TempDir(), "core-v2-history")
	if err := os.MkdirAll(obsoleteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.compact", "two.compact", "keep.txt"} {
		if err := os.WriteFile(filepath.Join(obsoleteDir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	maintenance := NewHistoryMaintenance(t.TempDir())

	removed, err := maintenance.DeleteObsolete(obsoleteDir)
	if err != nil || removed != 2 {
		t.Fatalf("delete obsolete removed=%d err=%v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(obsoleteDir, "keep.txt")); err != nil {
		t.Fatalf("unknown file was not preserved: %v", err)
	}

	if err := os.Remove(filepath.Join(obsoleteDir, "keep.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := maintenance.DeleteObsolete(obsoleteDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obsoleteDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty obsolete dir remains: %v", err)
	}

	removed, err = maintenance.DeleteObsolete(filepath.Join(t.TempDir(), "missing"))
	if err != nil || removed != 0 {
		t.Fatalf("missing obsolete dir removed=%d err=%v", removed, err)
	}
}

func TestHistoryMaintenancePrepareAppliesStorageConfig(t *testing.T) {
	dir := t.TempDir()
	seedMaintenanceHistory(t, dir, "alpha")
	obsolete := filepath.Join(dir, "alpha.history-lines.bin")
	if err := os.WriteFile(obsolete, []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := HistoryStorageConfig{
		MaxBytesPerTerminal: 1 << 20,
		Compression:         HistoryCompressionZstd,
		CompressionLevel:    HistoryCompressionLevelFast,
	}
	if err := NewHistoryMaintenance(dir).Prepare(config); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(obsolete); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("obsolete history file remains: %v", err)
	}
	file, err := linehist.OpenCompressedLineFile(dir, "alpha", linehist.CompressedLineFileOptions{
		MaxBytes:         config.MaxBytesPerTerminal,
		Compression:      config.Compression,
		CompressionLevel: config.CompressionLevel,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
