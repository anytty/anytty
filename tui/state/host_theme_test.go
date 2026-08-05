package state

import "testing"

func TestHostThemeStoreAppliesDefaultColorsAndPaletteImmutably(t *testing.T) {
	store := HostThemeStore{}
	store = store.ApplyUpdate(HostThemeUpdate{DefaultFG: "#aabbcc"})
	store = store.ApplyUpdate(HostThemeUpdate{DefaultBG: "#010203"})
	store = store.ApplyUpdate(HostThemeUpdate{PaletteIndex: 5, PaletteColor: "#445566"})
	if !store.Probed || store.DefaultFG != "#aabbcc" || store.DefaultBG != "#010203" {
		t.Fatalf("unexpected host theme defaults %#v", store)
	}
	if color, ok := store.PaletteColor(5); !ok || color != "#445566" {
		t.Fatalf("expected palette color, got %q ok=%v", color, ok)
	}
	next := store.ApplyUpdate(HostThemeUpdate{PaletteIndex: 6, PaletteColor: "#778899"})
	if _, ok := store.PaletteColor(6); ok {
		t.Fatalf("palette update should clone prior map, got original %#v", store)
	}
	if color, ok := next.PaletteColor(6); !ok || color != "#778899" {
		t.Fatalf("expected next palette color, got %q ok=%v", color, ok)
	}
}
