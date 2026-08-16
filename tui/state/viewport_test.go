package state

import "testing"

func TestViewportStoreResizeValidatesAndDeduplicates(t *testing.T) {
	var viewport ViewportStore
	if next, changed := viewport.Resize(0, 24); changed || next.Valid {
		t.Fatalf("invalid cols must not change viewport: %#v changed=%v", next, changed)
	}
	var changed bool
	viewport, changed = viewport.Resize(100, 40)
	if !changed || !viewport.Valid || viewport.Cols != 100 || viewport.Rows != 40 {
		t.Fatalf("expected valid viewport resize, got %#v changed=%v", viewport, changed)
	}
	viewport, changed = viewport.Resize(100, 40)
	if changed {
		t.Fatalf("duplicate viewport resize must be ignored: %#v", viewport)
	}
	viewport, changed = viewport.Resize(120, 40)
	if !changed || viewport.Cols != 120 || viewport.Rows != 40 {
		t.Fatalf("expected changed viewport cols, got %#v changed=%v", viewport, changed)
	}
}
