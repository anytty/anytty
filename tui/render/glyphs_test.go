package render

import (
	"strings"
	"testing"
)

func TestPaneChromeGlyphDefaultsAreVisiblePortableSingleCellSymbols(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	glyphs := DefaultPaneChromeGlyphs()
	for name, item := range map[string]struct {
		got  string
		want string
	}{
		"zoom":              {glyphs.Zoom, "↗"},
		"unzoom":            {glyphs.Unzoom, "↙"},
		"split vertical":    {glyphs.SplitVertical, "│"},
		"split horizontal":  {glyphs.SplitHorizontal, "─"},
		"close":             {glyphs.Close, "×"},
		"size lock":         {glyphs.SizeLock, "■"},
		"size unlock":       {glyphs.SizeUnlock, "□"},
		"center floating":   {glyphs.CenterFloating, "◎"},
		"collapse floating": {glyphs.CollapseFloating, "▾"},
		"running":           {glyphs.Running, "●"},
		"waiting":           {glyphs.Waiting, "○"},
		"exited":            {glyphs.Exited, "×"},
		"killed":            {glyphs.Killed, "×"},
	} {
		if item.got != item.want || DisplayWidth(item.got) != 1 {
			t.Fatalf("%s default must be visible and one terminal cell, got=%q want=%q width=%d", name, item.got, item.want, DisplayWidth(item.got))
		}
	}
}

func TestPaneChromeActionTextShowsWiredSplitAndCloseActions(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	glyphs := DefaultPaneChromeGlyphs()
	got := paneChromeActionText(40)
	want := "[" + glyphs.Zoom + "]─[" + glyphs.SplitVertical + "]─[" + glyphs.SplitHorizontal + "]─[" + glyphs.Close + "]"
	if got != want {
		t.Fatalf("wired action text got=%q want=%q", got, want)
	}
	if got := paneChromeActionText(8); got != "["+glyphs.Close+"]" {
		t.Fatalf("narrow pane should degrade to close-only action, got=%q", got)
	}
	if got := paneChromeActionText(7); got != "" {
		t.Fatalf("too-narrow pane should hide action text, got=%q", got)
	}
}

func TestFloatingChromeActionTextKeepsBracketedPaneChromeWithoutSplit(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	glyphs := DefaultPaneChromeGlyphs()
	got := paneChromeActionTextFromItems(floatingChromeActionItems(30))
	want := "[" + glyphs.CenterFloating + "]─[" + glyphs.CollapseFloating + "]─[" + glyphs.Zoom + "]─[" + glyphs.Close + "]"
	if got != want {
		t.Fatalf("floating actions got=%q want=%q", got, want)
	}
	if got := paneChromeActionTextFromItems(floatingChromeActionItems(8)); got != "["+paneChromeCloseGlyph()+"]" {
		t.Fatalf("narrow floating should degrade to close-only action, got=%q", got)
	}
	if got := paneChromeActionTextFromItems(floatingChromeActionItems(7)); got != "" {
		t.Fatalf("too-narrow floating should hide action text, got=%q", got)
	}
}

func TestSetPaneChromeGlyphsAllowsUTF8Overrides(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	SetPaneChromeGlyphs(PaneChromeGlyphs{
		Close:         "❌",
		Zoom:          "🔎",
		Unzoom:        "↙",
		SplitVertical: "↕",
	})
	if paneChromeCloseGlyph() != "❌" || paneChromeZoomGlyph() != "🔎" || paneChromeUnzoomGlyph() != "↙" || paneChromeSplitVerticalGlyph() != "↕" {
		t.Fatalf("glyph override did not apply, got close=%q zoom=%q unzoom=%q splitVertical=%q", paneChromeCloseGlyph(), paneChromeZoomGlyph(), paneChromeUnzoomGlyph(), paneChromeSplitVerticalGlyph())
	}
	if DisplayWidth(paneChromeCloseGlyph()) != 2 {
		t.Fatalf("emoji override width should be measured with display cells, got %d", DisplayWidth(paneChromeCloseGlyph()))
	}
}

func TestPaneChromeActionTemplatesCanUseZoomMode(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	SetPaneChromeGlyphs(PaneChromeGlyphs{
		ActionGroupLeft:    "{{if is_zoom_mode}}Z{{else}}N{{end}}",
		ActionGroupLeftSet: true,
	})
	normal := paneChromeActionRenderedFromItemsForState(paneChromeActionItemsFromVM(defaultPaneChromeActionVMs(StyleAccent)), StyleAccent, true)
	zoomed := paneChromeActionRenderedFromItemsForState(paneChromeActionItemsFromVM(defaultPaneChromeActionVMsForZoom(StyleAccent, true)), StyleAccent, true)
	if !strings.HasPrefix(normal.Text, "N") || !strings.HasPrefix(zoomed.Text, "Z") {
		t.Fatalf("zoom mode template branch not applied, normal=%q zoomed=%q", normal.Text, zoomed.Text)
	}
}

func TestPaneChromeActionTemplatesAllowGroupCapsuleAndEmptySides(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	SetPaneChromeGlyphs(PaneChromeGlyphs{
		ActionLeft:          " ",
		ActionLeftSet:       true,
		ActionRight:         " ",
		ActionRightSet:      true,
		ActionSeparator:     "",
		ActionSeparatorSet:  true,
		ActionGroupLeft:     "[fg:#8ffcff][fg:#071112;bg:#8ffcff;font:bold]",
		ActionGroupLeftSet:  true,
		ActionGroupRight:    "[reset][fg:#8ffcff][reset]",
		ActionGroupRightSet: true,
		Zoom:                "—",
		SplitVertical:       "□",
		SplitHorizontal:     "▭",
		Close:               "⤫",
	})

	got := paneChromeActionText(40)
	want := " —  □  ▭  ⤫ "
	if got != want {
		t.Fatalf("capsule action text got=%q want=%q", got, want)
	}
	if strings.Contains(got, "[fg:") || strings.Contains(got, "[reset]") {
		t.Fatalf("span tags must not render literally, got=%q", got)
	}

	rendered := paneChromeActionRenderedFromItems(paneChromeActionItems(40), StyleAccent)
	foundStyledGlyph := false
	for _, segment := range rendered.Segments {
		if strings.Contains(segment.text, "—") && segment.ansi.FG == "#071112" && segment.ansi.BG == "#8ffcff" && segment.ansi.Bold {
			foundStyledGlyph = true
			break
		}
	}
	if !foundStyledGlyph {
		t.Fatalf("expected action glyph to inherit configured ANSI style, segments=%#v", rendered.Segments)
	}
}

func TestPaneChromeActionTemplatesCanHideOneSide(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	SetPaneChromeGlyphs(PaneChromeGlyphs{
		ActionLeft:     "",
		ActionLeftSet:  true,
		ActionRight:    ")",
		ActionRightSet: true,
		Close:          "x",
	})
	if got, want := paneChromeActionText(8), "x)"; got != want {
		t.Fatalf("right-only action template got=%q want=%q", got, want)
	}
}

func TestPaneChromeActionGroupTemplateCanUseActiveState(t *testing.T) {
	ResetPaneChromeGlyphs()
	defer ResetPaneChromeGlyphs()

	SetPaneChromeGlyphs(PaneChromeGlyphs{
		ActionLeft:          " ",
		ActionLeftSet:       true,
		ActionRight:         " ",
		ActionRightSet:      true,
		ActionSeparator:     "",
		ActionSeparatorSet:  true,
		ActionGroupLeft:     "[fg:{{if active}}#8ffcff{{else}}#ff6bff{{end}}][fg:#071112;bg:{{if active}}#8ffcff{{else}}#ff6bff{{end}};font:bold]",
		ActionGroupLeftSet:  true,
		ActionGroupRight:    "[reset][fg:{{if active}}#8ffcff{{else}}#ff6bff{{end}}][reset]",
		ActionGroupRightSet: true,
		Zoom:                "󰁌",
		SplitVertical:       "",
		SplitHorizontal:     "",
		Close:               "󰅙",
	})

	items := paneChromeActionItems(40)
	active := paneChromeActionRenderedFromItemsForState(items, StyleAccent, true)
	inactive := paneChromeActionRenderedFromItemsForState(items, StyleMuted, false)
	if strings.Contains(active.Text, "│") || strings.Contains(inactive.Text, "│") {
		t.Fatalf("continuous capsule should not render separator, active=%q inactive=%q", active.Text, inactive.Text)
	}
	if got, want := active.Text, inactive.Text; got != want {
		t.Fatalf("active/inactive color branches should keep stable text width got=%q want=%q", got, want)
	}
	if !segmentsContainANSIText(active.Segments, "󰁌", ANSICellStyle{FG: "#071112", BG: "#8ffcff", Bold: true}) {
		t.Fatalf("active capsule should use primary background, segments=%#v", active.Segments)
	}
	if !segmentsContainANSIText(inactive.Segments, "󰁌", ANSICellStyle{FG: "#071112", BG: "#ff6bff", Bold: true}) {
		t.Fatalf("inactive capsule should use secondary background, segments=%#v", inactive.Segments)
	}
}

func segmentsContainANSIText(segments []barSegment, text string, style ANSICellStyle) bool {
	for _, segment := range segments {
		if strings.Contains(segment.text, text) && segment.ansi == style {
			return true
		}
	}
	return false
}
