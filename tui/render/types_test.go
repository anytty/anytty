package render

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

type recordingSink struct {
	frames []Frame
}

func (s *recordingSink) WriteFrame(frame Frame) error {
	s.frames = append(s.frames, frame)
	return nil
}

func TestFrameSinkContract(t *testing.T) {
	sink := &recordingSink{}
	if err := sink.WriteFrame(Frame{Lines: []string{"ok"}}); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	if len(sink.frames) != 1 || sink.frames[0].Lines[0] != "ok" {
		t.Fatalf("unexpected frames %#v", sink.frames)
	}
}

func TestFrameFromRenderResultUsesSingleResultPath(t *testing.T) {
	result := RenderResult{
		Content:    []Line{NewLine("hello"), NewLine("世界")},
		HitRegions: []HitRegion{{Kind: HitRegionStatus, Rect: Rect{W: 5, H: 1}}},
		Metadata:   RenderMetadata{Width: 5, Height: 2},
	}

	frame := FrameFromRenderResult(result)
	if len(frame.Lines) != 2 || frame.Lines[0] != "hello" || frame.Lines[1] != "世界" {
		t.Fatalf("unexpected frame %#v", frame)
	}
	if len(frame.HitRegions) != 1 || frame.HitRegions[0].Kind != HitRegionStatus {
		t.Fatalf("frame must preserve hit regions for runtime dispatch, got %#v", frame.HitRegions)
	}
}

func TestFrameFromRenderResultPreservesStyledANSIAndMetadata(t *testing.T) {
	result := RenderResult{
		Content: []Line{{
			Cells: []Cell{
				{Text: "hot", Width: 3, Style: StyleAccent, Safe: true},
				{Text: "\x1b[31mraw\x1b[0m", Width: 3, Safe: true},
			},
		}},
		Cursor:     Cursor{Visible: true, Row: 2, Col: 3, Shape: CursorShapeBar},
		CursorRect: Rect{X: 10, Y: 4, W: 1, H: 1},
		Blink:      true,
		Metadata:   RenderMetadata{Width: 6, Height: 1},
	}

	frame := FrameFromRenderResult(result)
	if len(frame.Lines) != 1 || frame.Lines[0] != "hotraw" {
		t.Fatalf("plain snapshot must strip ANSI while preserving text, got %#v", frame.Lines)
	}
	if len(frame.StyledLines) != 1 || frame.StyledLines[0].Cells[0].Style != StyleAccent {
		t.Fatalf("styled lines not preserved: %#v", frame.StyledLines)
	}
	if len(frame.ANSILines) != 1 || !strings.Contains(frame.ANSILines[0], "\x1b[") || !strings.Contains(frame.ANSILines[0], "hot") || !strings.HasSuffix(frame.ANSILines[0], ANSIReset) {
		t.Fatalf("ANSI line must retain SGR and reset, got %#v", frame.ANSILines)
	}
	if !frame.Cursor.Visible || frame.Cursor.Row != 2 || frame.Cursor.Col != 3 || frame.Cursor.Shape != CursorShapeBar {
		t.Fatalf("cursor metadata lost: %#v", frame.Cursor)
	}
	if frame.CursorRect != (Rect{X: 10, Y: 4, W: 1, H: 1}) {
		t.Fatalf("cursor rect metadata lost: %#v", frame.CursorRect)
	}
	if !frame.Blink || frame.Metadata.Width != 6 || frame.Metadata.Height != 1 {
		t.Fatalf("frame metadata lost: blink=%v metadata=%#v", frame.Blink, frame.Metadata)
	}
}

func TestThemeTokenPaletteDrivesANSILines(t *testing.T) {
	result := RenderResult{
		Content: []Line{{Cells: []Cell{
			{Text: "accent", Width: 6, Style: StyleAccent, Safe: true},
			{Text: " warn", Width: 5, Style: StyleWarning, Safe: true},
			{Text: " fg", Width: 3, Style: StyleForeground, Safe: true},
			{Text: " strong", Width: 7, Style: StyleStrongForeground, Safe: true},
			{Text: " match", Width: 6, Style: StylePickerMatch, Safe: true},
		}}},
		Theme: Theme{
			ChromeFG: "#ddeeff",
			HostBG:   "#000000",
			Accent:   "#010203",
			Warning:  "#a0b0c0",
		},
	}

	frame := FrameFromRenderResult(result)
	if len(frame.ANSILines) != 1 || !strings.Contains(frame.ANSILines[0], "\x1b[1;38;2;1;2;3m") {
		t.Fatalf("accent token should use custom theme color, got %#v", frame.ANSILines)
	}
	if !strings.Contains(frame.ANSILines[0], "\x1b[38;2;160;176;192m") {
		t.Fatalf("warning token should use semantic theme color, got %#v", frame.ANSILines)
	}
	if !strings.Contains(frame.ANSILines[0], "\x1b[38;2;221;238;255m fg") || strings.Contains(frame.ANSILines[0], "48;2;221;238;255") {
		t.Fatalf("foreground token should use chrome foreground without background, got %#v", frame.ANSILines)
	}
	if !strings.Contains(frame.ANSILines[0], "\x1b[1;38;2;221;238;255m strong") {
		t.Fatalf("strong foreground token should use bold chrome foreground without background, got %#v", frame.ANSILines)
	}
	if !strings.Contains(frame.ANSILines[0], "\x1b[1;38;2;160;176;192m match") || strings.Contains(frame.ANSILines[0], "\x1b[48;2;") {
		t.Fatalf("picker match should use warning foreground without modal background, got %#v", frame.ANSILines)
	}
	if frame.Theme.HostFG == "" || frame.Theme.StatusBG == "" {
		t.Fatalf("frame theme should be filled with fallback values, got %#v", frame.Theme)
	}
}

func TestANSICellStyleUsesHostPaletteCodes(t *testing.T) {
	result := RenderResult{
		Content: []Line{{Cells: []Cell{
			{Text: "dir", Width: 3, ANSIStyle: ANSICellStyle{FG: "ansi:4", Bold: true}, Safe: true},
			{Text: " err", Width: 4, ANSIStyle: ANSICellStyle{FG: "ansi:9", BG: "ansi:0"}, Safe: true},
			{Text: " rgb", Width: 4, ANSIStyle: ANSICellStyle{FG: "#010203", BG: "#0a0b0c"}, Safe: true},
			{Text: " idx", Width: 4, ANSIStyle: ANSICellStyle{FG: "idx:17", BG: "idx:236"}, Safe: true},
		}}},
		Theme: Theme{Info: "#7ab8ff"},
	}

	frame := FrameFromRenderResult(result)
	if len(frame.ANSILines) != 1 {
		t.Fatalf("expected one ANSI line, got %#v", frame.ANSILines)
	}
	got := frame.ANSILines[0]
	if !strings.Contains(got, "\x1b[1;34mdir") {
		t.Fatalf("terminal ansi:4 should pass through as host palette blue, got %#v", got)
	}
	if !strings.Contains(got, "\x1b[91;40m err") {
		t.Fatalf("bright foreground/background palette codes should pass through, got %#v", got)
	}
	if !strings.Contains(got, "\x1b[38;2;1;2;3;48;2;10;11;12m rgb") {
		t.Fatalf("terminal truecolor style should serialize as cell SGR, got %#v", got)
	}
	if !strings.Contains(got, "\x1b[38;5;17;48;5;236m idx") {
		t.Fatalf("terminal indexed color style should serialize as 256-color SGR, got %#v", got)
	}
	if strings.Contains(got, "38;2;122;184;255") {
		t.Fatalf("terminal ansi:4 must not be remapped to theme info truecolor, got %#v", got)
	}
}

func TestANSILinesPreserveCellHyperlinks(t *testing.T) {
	result := RenderResult{
		Content: []Line{{Cells: []Cell{
			{Text: "log", Width: 3, ANSIStyle: ANSICellStyle{FG: "ansi:4", Underline: true}, LinkURL: "file://build.log", LinkParams: "id=build", Safe: true},
		}}},
	}

	frame := FrameFromRenderResult(result)
	if len(frame.StyledLines) != 1 || frame.StyledLines[0].Cells[0].LinkURL != "file://build.log" || frame.StyledLines[0].Cells[0].LinkParams != "id=build" {
		t.Fatalf("styled frame should retain hyperlink metadata, got %#v", frame.StyledLines)
	}
	if len(frame.ANSILines) != 1 || !strings.Contains(frame.ANSILines[0], "\x1b]8;id=build;file://build.log\x1b\\") || !strings.Contains(frame.ANSILines[0], "\x1b]8;;\x1b\\") {
		t.Fatalf("ANSI frame should emit OSC 8 hyperlink metadata, got %#v", frame.ANSILines)
	}
}

func TestANSILinesEraseFE0FContinuationOnlyForTerminalCells(t *testing.T) {
	uiLine := Line{Cells: []Cell{
		{Text: "♻️", Width: 2, Safe: true},
		{Text: "·", Width: 1, Safe: true},
	}}
	if got := uiLine.ANSIString(DefaultTheme()); strings.Contains(got, "\x1b[1X") {
		t.Fatalf("non-terminal FE0F text should not use TTY continuation erase, got %q", got)
	}

	terminalLine := Line{Cells: []Cell{
		{Text: "♻️", Width: 2, TerminalContent: true, Safe: true},
		{Text: "·", Width: 1, Safe: true},
	}}
	if got := terminalLine.ANSIString(DefaultTheme()); !strings.Contains(got, "♻️\x1b[1X\x1b[3G·") {
		t.Fatalf("terminal FE0F cell should erase continuation before model-column anchor, got %q", got)
	}
}

func TestANSILinesReanchorsAfterFE0FInsideTerminalCell(t *testing.T) {
	line := Line{Cells: []Cell{
		{Text: "a♻️b", Width: 4, TerminalContent: true, Safe: true},
		{Text: "│", Width: 1, Safe: true},
	}}
	got := line.ANSIString(DefaultTheme())
	if !strings.Contains(got, "a♻️\x1b[1X\x1b[4Gb") || !strings.Contains(got, "b\x1b[5G│") {
		t.Fatalf("terminal FE0F inside a cell should clear continuation and re-anchor following text/border, got %q", got)
	}
}

func TestANSILinesMaterializeAuthoritativeCellPadding(t *testing.T) {
	line := Line{Cells: []Cell{
		{Text: "AGENTS.md", Width: 12, ANSIStyle: ANSICellStyle{FG: "ansi:4"}, TerminalContent: true, Safe: true},
		{Text: "go.work", Width: 9, ANSIStyle: ANSICellStyle{FG: "ansi:2"}, TerminalContent: true, Safe: true},
		{Text: "README.md", Width: 9, TerminalContent: true, Safe: true},
	}}
	if got := line.PlainString(); got != "AGENTS.md   go.work  README.md" {
		t.Fatalf("plain string should preserve terminal cell padding, got %q", got)
	}
	ansi := line.ANSIString(DefaultTheme())
	if !strings.Contains(ansi, "AGENTS.md   ") || !strings.Contains(ansi, "go.work  ") {
		t.Fatalf("ANSI string should write padded cell footprints, got %q", ansi)
	}
}

func TestThemeFallbackAndPaneTokensAreDistinct(t *testing.T) {
	theme := Theme{ActivePaneBorder: "#010203"}.WithFallback()
	if theme.ActivePaneBorder != "#010203" {
		t.Fatalf("explicit active pane border should be preserved, got %#v", theme)
	}
	if theme.InactivePane == "" || theme.MutedBorder == "" || theme.Accent == "" {
		t.Fatalf("fallback should populate missing pane tokens, got %#v", theme)
	}
	if theme.ActivePaneBorder == theme.InactivePane {
		t.Fatalf("active/inactive pane tokens must be distinct, got %#v", theme)
	}
}

func TestThemeFromEmptyHostThemeKeepsDefaultTheme(t *testing.T) {
	if got, want := ThemeFromHostTheme(state.HostThemeStore{}), DefaultTheme().WithFallback(); got != want {
		t.Fatalf("empty host theme should keep default theme\n got=%#v\nwant=%#v", got, want)
	}
}

func TestThemeFromHostThemeUsesHostPaletteForChromeOnly(t *testing.T) {
	host := state.HostThemeStore{}
	host = host.ApplyUpdate(state.HostThemeUpdate{DefaultFG: "#eeeeee"})
	host = host.ApplyUpdate(state.HostThemeUpdate{DefaultBG: "#101010"})
	host = host.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 5, PaletteColor: "#bb66ff"})
	host = host.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 4, PaletteColor: "#3366dd"})
	theme := ThemeFromHostTheme(host)
	if theme.HostFG != "#eeeeee" || theme.HostBG != "#101010" {
		t.Fatalf("host fg/bg should be preserved, got %#v", theme)
	}
	if theme.Accent != "#bb66ff" || theme.ActivePaneBorder != "#bb66ff" {
		t.Fatalf("palette index 5 should drive chrome accent, got %#v", theme)
	}
	if theme.Info != "#3366dd" {
		t.Fatalf("palette index 4 should drive info token, got %#v", theme)
	}

	result := RenderResult{
		Content: []Line{{
			Cells: []Cell{
				{Text: "chrome", Width: 6, Style: StyleAccent, Safe: true},
				{Text: "term", Width: 4, ANSIStyle: ANSICellStyle{FG: "ansi:4"}, Safe: true},
			},
		}},
		Theme: theme,
	}
	frame := result.Frame()
	if !strings.Contains(frame.ANSILines[0], "38;2;187;102;255") {
		t.Fatalf("chrome accent should use host-derived truecolor, got %#v", frame.ANSILines)
	}
	if !strings.Contains(frame.ANSILines[0], "\x1b[34mterm") {
		t.Fatalf("terminal content ansi:4 must still pass through host palette code, got %#v", frame.ANSILines)
	}
}

func TestThemeFromHostThemeConfigUserTokensOverrideHostPalette(t *testing.T) {
	host := state.HostThemeStore{}
	host = host.ApplyUpdate(state.HostThemeUpdate{DefaultFG: "#eeeeee"})
	host = host.ApplyUpdate(state.HostThemeUpdate{DefaultBG: "#101010"})
	host = host.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 5, PaletteColor: "#bb66ff"})
	host = host.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 4, PaletteColor: "#3366dd"})
	cfg := state.TUIConfigStore{
		Theme: state.TUIThemeConfig{
			Palette:   "host",
			Primary:   "#d65cff",
			Secondary: "#66e3ff",
			Border:    state.TUIThemeBorderConfig{Active: "#ff00aa"},
			Surface:   state.TUIThemeSurfaceConfig{StatusBG: "#090909"},
		},
	}
	theme := ThemeFromHostThemeConfig(host, cfg)
	if theme.HostFG != "#eeeeee" || theme.HostBG != "#101010" {
		t.Fatalf("host fg/bg should stay host-aware, got %#v", theme)
	}
	if theme.Accent != "#d65cff" || theme.Info != "#66e3ff" {
		t.Fatalf("user primary/secondary should override host palette, got %#v", theme)
	}
	if theme.ActivePaneBorder != "#ff00aa" || theme.StatusBG != "#090909" {
		t.Fatalf("user border/surface overrides should win, got %#v", theme)
	}
}

func TestThemeFromHostThemeConfigBuiltinPaletteIgnoresHostPalette(t *testing.T) {
	host := state.HostThemeStore{}
	host = host.ApplyUpdate(state.HostThemeUpdate{DefaultFG: "#eeeeee"})
	host = host.ApplyUpdate(state.HostThemeUpdate{DefaultBG: "#101010"})
	host = host.ApplyUpdate(state.HostThemeUpdate{PaletteIndex: 5, PaletteColor: "#bb66ff"})
	theme := ThemeFromHostThemeConfig(host, state.TUIConfigStore{Theme: state.TUIThemeConfig{Palette: "builtin"}})
	if theme.HostFG == "#eeeeee" || theme.HostBG == "#101010" || theme.Accent == "#bb66ff" {
		t.Fatalf("builtin palette should ignore host theme, got %#v", theme)
	}
	if theme != DefaultTheme().WithFallback() {
		t.Fatalf("builtin palette without user overrides should equal default theme, got %#v", theme)
	}
}

func TestFrameCloneDetachesLines(t *testing.T) {
	frame := Frame{
		Lines:       []string{"one"},
		ANSILines:   []string{"\x1b[31mone\x1b[0m"},
		StyledLines: []Line{{Cells: []Cell{{Text: "one", Width: 3, Style: StyleAccent, Safe: true}}}},
		Cursor:      Cursor{Visible: true, Row: 1, Col: 2},
		CursorRect:  Rect{X: 2, Y: 1, W: 1, H: 1},
		Metadata:    RenderMetadata{Width: 3, Height: 1},
	}
	cloned := frame.Clone()
	frame.Lines[0] = "mutated"
	frame.ANSILines[0] = "mutated"
	frame.StyledLines[0].Cells[0].Text = "mutated"
	if cloned.Lines[0] != "one" {
		t.Fatalf("expected detached plain clone, got %#v", cloned)
	}
	if cloned.ANSILines[0] != "\x1b[31mone\x1b[0m" {
		t.Fatalf("expected detached ANSI clone, got %#v", cloned)
	}
	if cloned.StyledLines[0].Cells[0].Text != "one" || cloned.Cursor.Row != 1 || cloned.CursorRect.X != 2 || cloned.Metadata.Width != 3 {
		t.Fatalf("expected detached styled clone with metadata, got %#v", cloned)
	}
}
