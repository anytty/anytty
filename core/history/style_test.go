package history

import "testing"

func TestR308ResolveCellStyleUsesThemeOnlyForSemanticDefault(t *testing.T) {
	resolved := ResolveCellStyleForTheme(CellStyle{Bold: true}, HistoryTheme{
		DefaultFG: "#eeeeee",
		DefaultBG: "#111111",
	})
	if resolved.FG != "#eeeeee" || resolved.BG != "#111111" || !resolved.Bold {
		t.Fatalf("default fg/bg should resolve at view time, got %#v", resolved)
	}
}

func TestR308ResolveCellStylePreservesExplicitColorTokens(t *testing.T) {
	for _, style := range []CellStyle{
		{FG: "ansi:1", BG: "ansi:4"},
		{FG: "idx:123", BG: "idx:45"},
		{FG: "#010203", BG: "#a0b0c0"},
	} {
		resolved := ResolveCellStyleForTheme(style, HistoryTheme{
			DefaultFG: "#eeeeee",
			DefaultBG: "#111111",
		})
		if resolved.FG != style.FG || resolved.BG != style.BG {
			t.Fatalf("explicit color token must survive theme resolution: style=%#v resolved=%#v", style, resolved)
		}
	}
}
