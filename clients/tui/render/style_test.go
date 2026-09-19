package render

import "testing"

// Golden SGR sequences for the ThemeDark palette. They are written out
// literally so a palette edit fails loudly instead of silently churning the
// whole host output.
const (
	defaultSGR        = "\x1b[38;2;230;226;236m"
	chromeFGSGR       = "\x1b[38;2;217;212;228m"
	defaultBoldSGR    = "\x1b[1;38;2;217;212;228m"
	mutedSGR          = "\x1b[38;2;154;148;168m\x1b[2m"
	accentSGR         = "\x1b[1;38;2;167;139;250m"
	successSGR        = "\x1b[38;2;95;208;138m"
	warningSGR        = "\x1b[38;2;240;192;90m"
	dangerSGR         = "\x1b[38;2;239;111;111m"
	infoSGR           = "\x1b[38;2;127;178;240m"
	headerSGR         = "\x1b[1;38;2;217;212;228m\x1b[48;2;22;24;35m"
	statusSGR         = "\x1b[38;2;213;208;224m\x1b[48;2;18;20;28m"
	overlaySGR        = "\x1b[38;2;217;212;228m\x1b[48;2;25;27;38m"
	borderSGR         = "\x1b[38;2;70;74;90m"
	activeBorderSGR   = "\x1b[1;38;2;167;139;250m"
	inactiveBorderSGR = "\x1b[38;2;154;148;168m"
	backgroundSGR     = "\x1b[48;2;15;17;23m"
	accentDimSGR      = "\x1b[38;2;107;95;168m"
	chromeFocusSGR    = "\x1b[1;38;2;242;239;250m\x1b[48;2;35;38;58m"
	tabActiveSGR      = "\x1b[1;38;2;20;18;28m\x1b[48;2;167;139;250m"
	tabInactiveSGR    = "\x1b[38;2;154;148;168m\x1b[2m"
	borderDeadSGR     = "\x1b[38;2;163;93;93m"
	selectionSGR      = "\x1b[38;2;244;241;250m\x1b[48;2;58;51;87m"
)

func TestExplicitStyleSGRGolden(t *testing.T) {
	theme := DefaultTheme()
	tests := []struct {
		name  string
		token Token
		want  string
	}{
		{"foreground only", "fg:#ff0000", "\x1b[38;2;255;0;0m"},
		{"background only", "bg:#00ff00", "\x1b[48;2;0;255;0m"},
		{"foreground and background", "fg:#a78bfa;bg:#161823", "\x1b[38;2;167;139;250;48;2;22;24;35m"},
		{"attributes only", "bold;dim;reverse", "\x1b[1;2;7m"},
		{"full form", "fg:#e6e2ec;bg:#0f1117;bold;dim;italic;underline;reverse",
			"\x1b[1;2;3;4;7;38;2;230;226;236;48;2;15;17;23m"},
		{"case insensitive hex", "fg:#AABBCC", "\x1b[38;2;170;187;204m"},
		{"no hash accepted", "fg:ff0000", "\x1b[38;2;255;0;0m"},
		{"unknown segment degrades", "fg:#112233;wat;bold", "\x1b[1;38;2;17;34;51m"},
		{"malformed color dropped", "fg:nothex;bold", "\x1b[1m"},
		{"explicit syntax garbage is empty", "wat:ever;nope", ""},
		{"explicit style bypasses theme", "fg:#000000", "\x1b[38;2;0;0;0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.SGR(theme); got != tt.want {
				t.Fatalf("Token(%q).SGR = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestParseStyleRoundTrip(t *testing.T) {
	tests := []struct {
		in       string
		want     string
		explicit bool
	}{
		{"", "", false},
		{"accent", "", false},
		{"bold", "bold", true},
		{"fg:#AABBCC", "fg:#aabbcc", true},
		{"fg:#aabbcc;bg:#010203;bold;reverse", "fg:#aabbcc;bg:#010203;bold;reverse", true},
		{"unknown:whatever", "", true},
	}
	for _, tt := range tests {
		style, ok := ParseStyle(tt.in)
		if ok != tt.explicit {
			t.Fatalf("ParseStyle(%q) explicit = %v, want %v", tt.in, ok, tt.explicit)
		}
		if got := style.String(); got != tt.want {
			t.Fatalf("ParseStyle(%q).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTokenSGR(t *testing.T) {
	theme := DefaultTheme()
	tests := []struct {
		name  string
		token Token
		want  string
	}{
		{"empty token is default", "", defaultSGR},
		{"default", TokenDefault, defaultSGR},
		{"fg", TokenFG, defaultSGR},
		{"background", TokenBackground, backgroundSGR},
		{"foreground alias", TokenForeground, chromeFGSGR},
		{"strong foreground", TokenStrongForeground, defaultBoldSGR},
		{"muted", TokenMuted, mutedSGR},
		{"accent", TokenAccent, accentSGR},
		{"accent dim", TokenAccentDim, accentDimSGR},
		{"success alias", TokenSuccess, successSGR},
		{"ok", TokenOK, successSGR},
		{"warning", TokenWarning, warningSGR},
		{"danger", TokenDanger, dangerSGR},
		{"info", TokenInfo, infoSGR},
		{"chrome", TokenChrome, headerSGR},
		{"chrome focus", TokenChromeFocus, chromeFocusSGR},
		{"header alias", TokenHeader, headerSGR},
		{"tab active", TokenTabActive, tabActiveSGR},
		{"tab inactive", TokenTabInactive, tabInactiveSGR},
		{"footer", TokenFooter, "\x1b[38;2;154;148;168m"},
		{"footer accent", TokenFooterAccent, accentSGR},
		{"status", TokenStatus, statusSGR},
		{"overlay", TokenOverlay, overlaySGR},
		{"border", TokenBorder, borderSGR},
		{"border focus", TokenBorderFocus, activeBorderSGR},
		{"border dead", TokenBorderDead, borderDeadSGR},
		{"active border alias", TokenActiveBorder, activeBorderSGR},
		{"inactive border", TokenInactiveBorder, inactiveBorderSGR},
		{"selection", TokenSelection, selectionSGR},
		{"unknown", Token("nope"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.SGR(theme); got != tt.want {
				t.Fatalf("Token(%q).SGR = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestANSITokenBypassesTheme(t *testing.T) {
	token := ANSIToken("38;5;196")
	if got := token.SGR(ThemeLight()); got != "\x1b[38;5;196m" {
		t.Fatalf("raw ANSI token SGR = %q, want verbatim", got)
	}
	if params, ok := token.RawSGR(); !ok || params != "38;5;196" {
		t.Fatalf("RawSGR = %q/%v", params, ok)
	}
	if _, ok := Token("accent").RawSGR(); ok {
		t.Fatal("named tokens must not report raw SGR")
	}
	if got := ANSIToken("").SGR(DefaultTheme()); got != defaultSGR {
		t.Fatalf("empty ANSI token = %q, want default", got)
	}
}

func TestThemeByName(t *testing.T) {
	for name, want := range map[string]string{"": "dark", "dark": "dark", "DARK": "dark", "light": "light"} {
		theme, ok := ThemeByName(name)
		if !ok {
			t.Fatalf("ThemeByName(%q) not found", name)
		}
		if theme.Name != want {
			t.Fatalf("ThemeByName(%q).Name = %q, want %q", name, theme.Name, want)
		}
	}
	if _, ok := ThemeByName("solarized"); ok {
		t.Fatal("unknown theme name must not resolve")
	}
	if ThemeDark().Background == ThemeLight().Background {
		t.Fatal("dark and light backgrounds must differ")
	}
	if ThemeDark().SelectionBG == "" || ThemeLight().SelectionBG == "" {
		t.Fatal("both themes need every semantic slot")
	}
}

func TestThemeFallback(t *testing.T) {
	theme := Theme{Accent: "#010203"}.WithFallback()
	if theme.Accent != "#010203" {
		t.Fatalf("Accent = %q, want custom value kept", theme.Accent)
	}
	if theme.Muted != ThemeDark().Muted {
		t.Fatalf("Muted = %q, want dark fallback", theme.Muted)
	}
	if got := TokenSelection.SGR(theme); got != selectionSGR {
		t.Fatalf("selection SGR = %q", got)
	}
	if TokenAccent.SGR(theme) != "\x1b[1;38;2;1;2;3m" {
		t.Fatalf("custom accent SGR = %q", TokenAccent.SGR(theme))
	}
}

func TestVisibleText(t *testing.T) {
	if got := VisibleText("a\nb\tc\r"); got != "a b c " {
		t.Fatalf("VisibleText = %q, want %q", got, "a b c ")
	}
	if got := VisibleText("你好😀"); got != "你好😀" {
		t.Fatalf("VisibleText mangled wide text: %q", got)
	}
	if got := VisibleText("a\x1b[0mb"); got != "a [0mb" {
		t.Fatalf("VisibleText = %q, want control bytes replaced", got)
	}
}
