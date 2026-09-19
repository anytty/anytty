package pty

import "testing"

func TestNormalizeDefaultsAndClamp(t *testing.T) {
	cfg := normalize(Config{})
	if cfg.Cols != DefaultCols || cfg.Rows != DefaultRows {
		t.Fatalf("defaults = %dx%d, want %dx%d", cfg.Cols, cfg.Rows, DefaultCols, DefaultRows)
	}
	neg := normalize(Config{Cols: -3, Rows: -9})
	if neg.Cols != DefaultCols || neg.Rows != DefaultRows {
		t.Fatalf("negative clamp = %dx%d", neg.Cols, neg.Rows)
	}
	huge := normalize(Config{Cols: 1 << 20, Rows: 1 << 20})
	if huge.Cols != 0xffff || huge.Rows != 0xffff {
		t.Fatalf("huge clamp = %dx%d, want 65535x65535", huge.Cols, huge.Rows)
	}
	explicit := normalize(Config{Cols: 120, Rows: 40})
	if explicit.Cols != 120 || explicit.Rows != 40 {
		t.Fatalf("explicit size changed: %dx%d", explicit.Cols, explicit.Rows)
	}
}
