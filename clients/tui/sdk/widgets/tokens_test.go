package widgets

import "testing"

func TestTokenConstantsMatchHostNames(t *testing.T) {
	cases := map[string]string{
		"StyleDefault":          "default",
		"StyleBackground":       "bg",
		"StyleFG":               "fg",
		"StyleForeground":       "foreground",
		"StyleStrongForeground": "strong-foreground",
		"StyleMuted":            "muted",
		"StyleAccent":           "accent",
		"StyleAccentDim":        "accent_dim",
		"StyleSuccess":          "success",
		"StyleOK":               "ok",
		"StyleWarning":          "warning",
		"StyleDanger":           "danger",
		"StyleInfo":             "info",
		"StyleChrome":           "chrome",
		"StyleChromeFocus":      "chrome_focus",
		"StyleHeader":           "header",
		"StyleTabActive":        "tab_active",
		"StyleTabInactive":      "tab_inactive",
		"StyleFooter":           "footer",
		"StyleFooterAccent":     "footer-accent",
		"StyleStatus":           "status",
		"StyleOverlay":          "overlay",
		"StyleBorder":           "border",
		"StyleBorderFocus":      "border_focus",
		"StyleBorderDead":       "border_dead",
		"StyleActiveBorder":     "active-border",
		"StyleInactiveBorder":   "inactive-border",
		"StyleSelection":        "selection",
	}
	got := map[string]string{
		"StyleDefault": StyleDefault, "StyleBackground": StyleBackground,
		"StyleFG": StyleFG, "StyleForeground": StyleForeground,
		"StyleStrongForeground": StyleStrongForeground, "StyleMuted": StyleMuted,
		"StyleAccent": StyleAccent, "StyleAccentDim": StyleAccentDim,
		"StyleSuccess": StyleSuccess, "StyleOK": StyleOK,
		"StyleWarning": StyleWarning, "StyleDanger": StyleDanger,
		"StyleInfo": StyleInfo, "StyleChrome": StyleChrome,
		"StyleChromeFocus": StyleChromeFocus, "StyleHeader": StyleHeader,
		"StyleTabActive": StyleTabActive, "StyleTabInactive": StyleTabInactive,
		"StyleFooter": StyleFooter, "StyleFooterAccent": StyleFooterAccent,
		"StyleStatus": StyleStatus, "StyleOverlay": StyleOverlay,
		"StyleBorder": StyleBorder, "StyleBorderFocus": StyleBorderFocus,
		"StyleBorderDead": StyleBorderDead, "StyleActiveBorder": StyleActiveBorder,
		"StyleInactiveBorder": StyleInactiveBorder, "StyleSelection": StyleSelection,
	}
	for name, want := range cases {
		if got[name] != want {
			t.Errorf("%s = %q, want %q", name, got[name], want)
		}
	}
}

func TestStyleAttributeHelpersAreIdempotent(t *testing.T) {
	if got := WithBold(""); got != "bold" {
		t.Fatalf("WithBold(empty) = %q", got)
	}
	if got := WithBold(WithBold("accent")); got != "accent;bold" {
		t.Fatalf("WithBold twice = %q", got)
	}
	if got := WithUnderline(""); got != "underline" {
		t.Fatalf("WithUnderline(empty) = %q", got)
	}
	if got := WithReverse(WithReverse("fg:#101010")); got != "fg:#101010;reverse" {
		t.Fatalf("WithReverse twice = %q", got)
	}
	if got := WithItalic(WithItalic("")); got != "italic" {
		t.Fatalf("WithItalic twice = %q", got)
	}
	if got := WithBold("fg:#111111;bold"); got != "fg:#111111;bold" {
		t.Fatalf("WithBold on an existing bold = %q", got)
	}
}

func TestStyleColorHelpers(t *testing.T) {
	if got := WithFg("", "#aabbcc"); got != "fg:#aabbcc" {
		t.Fatalf("WithFg(empty) = %q", got)
	}
	if got := WithFg("bold", "#aabbcc"); got != "fg:#aabbcc;bold" {
		t.Fatalf("WithFg prepend = %q", got)
	}
	if got := WithFg("fg:#111111;bold", "#aabbcc"); got != "fg:#aabbcc;bold" {
		t.Fatalf("WithFg replace = %q", got)
	}
	if got := WithBg("fg:#111111;bg:#000000", "#ffffff"); got != "fg:#111111;bg:#ffffff" {
		t.Fatalf("WithBg replace = %q", got)
	}
	if got := WithFg("bold", ""); got != "bold" {
		t.Fatalf("WithFg with empty hex = %q", got)
	}
	once := WithFg("bold", "#aabbcc")
	if twice := WithFg(once, "#aabbcc"); twice != once {
		t.Fatalf("WithFg is not idempotent: %q vs %q", twice, once)
	}
}
