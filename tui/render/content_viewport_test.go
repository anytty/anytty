package render

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func TestContentViewportKeepsTerminalBlankCellsAsSpaces(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 20, H: 5},
		Content: ContentVM{
			Kind: ContentTerminalLive,
			Lines: []Line{
				NewLine("123456789012345678901234"),
				NewLine("ok"),
			},
			Extent: ContentExtent{Known: true, Cols: 20, Rows: 5},
		},
	})

	if got, want := plainContentViewportLines(result.Lines), []string{
		"12345678901234567890",
		"ok                  ",
		"                    ",
		"                    ",
		"                    ",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ordinary terminal blanks must stay spaces\n got=%#v\nwant=%#v", got, want)
	}
	if result.Overflow != (ContentOverflow{Right: true}) {
		t.Fatalf("terminal live viewport should expose clipped columns to chrome, got %#v", result.Overflow)
	}
	assertContentViewportLineWidths(t, result.Lines, 20)
}

func TestContentViewportKeepsAreaOutsideTerminalExtentBlank(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 10, H: 6},
		Content: ContentVM{
			Kind: ContentTerminalLive,
			Lines: []Line{
				NewLine("abcde"),
				NewLine("fghij"),
				NewLine("klmno"),
			},
			Extent: ContentExtent{Known: true, Cols: 5, Rows: 3},
		},
	})

	if got, want := plainContentViewportLines(result.Lines), []string{
		"abcde·····",
		"fghij·····",
		"klmno·····",
		"··········",
		"··········",
		"··········",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("terminal extent outside cells should show boundary dots\n got=%#v\nwant=%#v", got, want)
	}
	if result.Overflow != (ContentOverflow{}) {
		t.Fatalf("small extent should not report overflow, got %#v", result.Overflow)
	}
	assertContentViewportLineWidths(t, result.Lines, 10)
}

func TestCopyHistoryKeepsAreaOutsideTerminalExtentMarked(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 10, H: 5},
		Content: ContentVM{
			Kind: ContentCopyHistory,
			Lines: []Line{
				NewLine("old-1"),
				NewLine("old-2"),
			},
			Extent: ContentExtent{Known: true, Cols: 6, Rows: 3},
		},
	})

	if got, want := plainContentViewportLines(result.Lines), []string{
		"old-1 ····",
		"old-2 ····",
		"      ····",
		"··········",
		"··········",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("copy history must preserve the terminal extent boundary\n got=%#v\nwant=%#v", got, want)
	}
}

func TestCopyHistoryLayoutMovesAndClipsRowHitRegionsWithTerminalExtent(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 10, H: 5},
		Content: ContentVM{
			Kind:       ContentCopyHistory,
			Lines:      []Line{NewLine("history")},
			Extent:     ContentExtent{Known: true, Cols: 6, Rows: 3},
			Layout:     ContentLayoutVM{Known: true, Mode: "center"},
			HitRegions: []HitRegion{{Kind: HitRegionHistoryRow, Rect: Rect{W: 8, H: 1}, Row: 7}},
		},
	})

	if len(result.HitRegions) != 1 || result.HitRegions[0].Rect != (Rect{X: 2, Y: 1, W: 6, H: 1}) {
		t.Fatalf("copy history row hit region must follow centered terminal extent, got %#v", result.HitRegions)
	}
}

func TestContentViewportSupportsOffsetTerminalExtent(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 4, H: 3},
		Content: ContentVM{
			Kind:   ContentTerminalLive,
			Lines:  []Line{NewLine("hi")},
			Extent: ContentExtent{Known: true, X: 1, Y: 1, Cols: 2, Rows: 1},
		},
	})

	if got, want := plainContentViewportLines(result.Lines), []string{
		"····",
		"·hi·",
		"····",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("offset extent should keep terminal area aligned with visible boundary padding\n got=%#v\nwant=%#v", got, want)
	}
	if result.Overflow != (ContentOverflow{}) {
		t.Fatalf("offset extent inside rect should not overflow, got %#v", result.Overflow)
	}
}

func TestContentViewportAppliesTerminalViewPan(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 4, H: 2},
		Content: ContentVM{
			Kind:   ContentTerminalLive,
			Lines:  []Line{NewLine("abcdef"), NewLine("ghijkl")},
			Extent: ContentExtent{Known: true, Cols: 6, Rows: 2},
			Layout: ContentLayoutVM{Known: true, PanX: 2, PanY: 1},
			Cursor: Cursor{Visible: true, Row: 1, Col: 3, Shape: CursorShapeBar},
		},
	})

	if got, want := plainContentViewportLines(result.Lines), []string{
		"ijkl",
		"····",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("pan should move the source window inside the terminal extent\n got=%#v\nwant=%#v", got, want)
	}
	if result.Overflow != (ContentOverflow{Left: true, Top: true}) {
		t.Fatalf("panned extent should expose viewport overflow, got %#v", result.Overflow)
	}
	if result.Cursor != (Cursor{Visible: true, Row: 0, Col: 1, Shape: CursorShapeBar}) {
		t.Fatalf("panned content cursor should move with viewport, got %#v", result.Cursor)
	}
}

func TestContentViewportAppliesTerminalViewCenterAndFit(t *testing.T) {
	centered := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 6, H: 3},
		Content: ContentVM{
			Kind:   ContentTerminalLive,
			Lines:  []Line{NewLine("ab"), NewLine("cd")},
			Extent: ContentExtent{Known: true, Cols: 2, Rows: 2},
			Layout: ContentLayoutVM{Known: true, Mode: "center"},
			Cursor: Cursor{Visible: true, Row: 1, Col: 1, Shape: CursorShapeBar},
		},
	})
	if got, want := plainContentViewportLines(centered.Lines), []string{
		"······",
		"··ab··",
		"··cd··",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("center layout should center terminal extent in content rect\n got=%#v\nwant=%#v", got, want)
	}
	if centered.Cursor != (Cursor{Visible: true, Row: 2, Col: 3, Shape: CursorShapeBar}) {
		t.Fatalf("centered content cursor should move with viewport, got %#v", centered.Cursor)
	}

	fit := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 6, H: 3},
		Content: ContentVM{
			Kind:   ContentTerminalLive,
			Lines:  []Line{NewLine("ab"), NewLine("cd")},
			Extent: ContentExtent{Known: true, Cols: 2, Rows: 2},
			Layout: ContentLayoutVM{Known: true, Mode: "fit"},
		},
	})
	if got, want := plainContentViewportLines(fit.Lines), []string{
		"ab    ",
		"cd    ",
		"      ",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("fit layout should use pane content rect as terminal extent\n got=%#v\nwant=%#v", got, want)
	}
}

func TestContentViewportReturnsOverflowHintsWithoutWritingMarkers(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 10, H: 5},
		Content: ContentVM{
			Kind: ContentTerminalLive,
			Lines: []Line{
				NewLine("abcdefghijk"),
				NewLine("row 2"),
				NewLine("row 3"),
				NewLine("row 4"),
				NewLine("row 5"),
				NewLine("row 6"),
			},
			Extent: ContentExtent{Known: true, Cols: 12, Rows: 8},
		},
	})

	if result.Overflow != (ContentOverflow{Right: true, Bottom: true}) {
		t.Fatalf("terminal live viewport should expose overflow hints to chrome, got %#v", result.Overflow)
	}
	if rendered := strings.Join(plainContentViewportLines(result.Lines), "\n"); strings.Contains(rendered, ">") || strings.Contains(rendered, "v") {
		t.Fatalf("overflow markers belong to chrome, not content cells: %q", rendered)
	}
	if got := len(result.Lines); got != 5 {
		t.Fatalf("content viewport must return rect height lines, got %d", got)
	}
	assertContentViewportLineWidths(t, result.Lines, 10)
}

func TestContentViewportPreservesANSICellStyleAndWideTruncation(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 5, H: 2},
		Content: ContentVM{
			Kind: ContentTerminalLive,
			Lines: []Line{{Cells: []Cell{
				{Text: "你", Width: 2, ANSIStyle: ANSICellStyle{FG: "ansi:2", Bold: true}, Safe: true},
				{Text: "好", Width: 2, ANSIStyle: ANSICellStyle{FG: "ansi:3"}, Safe: true},
				{Text: "w", Width: 1, ANSIStyle: ANSICellStyle{FG: "ansi:4"}, Safe: true},
				{Text: "orld", Width: 4, ANSIStyle: ANSICellStyle{FG: "ansi:5"}, Safe: true},
			}}},
			Extent: ContentExtent{Known: true, Cols: 5, Rows: 2},
		},
	})

	if got := result.Lines[0].PlainString(); got != "你好w" {
		t.Fatalf("wide cells should truncate at cell boundary, got %q", got)
	}
	if got := result.Lines[1].PlainString(); got != "     " {
		t.Fatalf("terminal-internal empty row should be spaces, got %q", got)
	}
	if !contentViewportLineHasANSI(result.Lines[0], "你", ANSICellStyle{FG: "ansi:2", Bold: true}) ||
		!contentViewportLineHasANSI(result.Lines[0], "w", ANSICellStyle{FG: "ansi:4"}) {
		t.Fatalf("ANSI cell style should survive viewport clipping, got %#v", result.Lines[0])
	}
	if strings.Contains(result.Lines[0].ANSIString(DefaultTheme()), "\x1b[38;2") {
		t.Fatalf("terminal ANSI cells must not be remapped to theme tokens, got %q", result.Lines[0].ANSIString(DefaultTheme()))
	}
}

func TestContentViewportPreservesANSICellStyleOnPaddedTail(t *testing.T) {
	style := ANSICellStyle{BG: "ansi:4"}
	line := Line{Cells: []Cell{{
		Text:            "BG",
		Width:           6,
		ANSIStyle:       style,
		TerminalContent: true,
		Safe:            true,
	}}}

	visible := contentViewportLineWindow(line, 2, 4)

	if got := visible.PlainString(); got != "    " {
		t.Fatalf("padded terminal tail should render as visible spaces, got %q", got)
	}
	if len(visible.Cells) != 1 || visible.Cells[0].ANSIStyle != style || visible.Cells[0].Width != 4 {
		t.Fatalf("padded terminal tail should keep source ANSI background, got %#v", visible.Cells)
	}
}

func TestContentViewportCentersEmptyPaneActionsAndStyles(t *testing.T) {
	lines, regions, cursor := emptyPaneContentLayout("pane-1", 0)
	result := RenderContentViewport(ContentRenderRequest{
		Rect:    Rect{W: 40, H: 8},
		Content: ContentVM{Kind: ContentEmptyPane, Lines: lines, HitRegions: regions, Cursor: cursor},
	})

	if got, want := SliceCells(result.Lines[0].PlainString(), 8, 31), "○ No terminal connected"; got != want {
		t.Fatalf("empty pane headline should be centered got=%q want=%q lines=%#v", got, want, plainContentViewportLines(result.Lines))
	}
	if got, want := SliceCells(result.Lines[4].PlainString(), 6, 34), "► Attach existing terminal ◄"; got != want {
		t.Fatalf("selected empty action should use tuiv2 arrows got=%q want=%q lines=%#v", got, want, plainContentViewportLines(result.Lines))
	}
	if got, want := SliceCells(result.Lines[5].PlainString(), 8, 31), "[ Create new terminal ]"; got != want {
		t.Fatalf("unselected empty action should use brackets got=%q want=%q lines=%#v", got, want, plainContentViewportLines(result.Lines))
	}
	if !styledLinesContainText(result.Lines, "► Attach existing terminal ◄", StyleAccent) ||
		!styledLinesContainText(result.Lines, "[ Create new terminal ]", StyleSuccess) ||
		!styledLinesContainText(result.Lines, "[ Open terminal manager ]", StyleForeground) ||
		!styledLinesContainText(result.Lines, "[ Close pane ]", StyleDangerStrong) {
		t.Fatalf("empty actions should keep action-specific styles, got %#v", result.Lines)
	}
	attach := hitRegionByAction(t, result.HitRegions, ActionEmptyAttach.String())
	if attach.Rect != (Rect{X: 6, Y: 4, W: 28, H: 1}) {
		t.Fatalf("selected empty action hit region should follow centered text, got %#v", attach)
	}
	if result.Cursor.Visible || result.Cursor.Anchor {
		t.Fatalf("empty pane must not expose cursor or IME anchor, got %#v", result.Cursor)
	}
}

func TestContentViewportKeepsExitedTextLeftAndCentersActions(t *testing.T) {
	lines, regions := liveExitedContentLines(
		state.TerminalSurfaceStore{TerminalID: "term-1", State: state.TerminalLiveExited, ExitCode: 23, Command: []string{"bash", "-lc", "exit 23"}},
		state.TerminalSessionStore{},
		0,
	)
	result := RenderContentViewport(ContentRenderRequest{
		Rect:    Rect{W: 80, H: 8},
		Content: ContentVM{Kind: ContentExitedPane, Lines: lines, HitRegions: regions},
	})

	if got, want := result.Lines[1].PlainString(), "terminal exited: term-1 code:23"; !strings.HasPrefix(got, want) {
		t.Fatalf("exited pane headline should remain left-aligned got=%q want prefix=%q lines=%#v", got, want, plainContentViewportLines(result.Lines))
	}
	if got, want := strings.TrimSpace(result.Lines[3].PlainString()), "► restart ◄"; got != want {
		t.Fatalf("selected exited restart should be centered got=%q want=%q lines=%#v", got, want, plainContentViewportLines(result.Lines))
	}
	if !styledLinesContainText(result.Lines, "► restart ◄", StyleWarning) ||
		!styledLinesContainText(result.Lines, "[ reconnect ]", StyleMuted) {
		t.Fatalf("exited actions should keep action-specific styles, got %#v", result.Lines)
	}
	restart := hitRegionByAction(t, result.HitRegions, ActionExitedRestart.String())
	restartWidth := DisplayWidth("► restart ◄")
	if restart.Rect != (Rect{X: (80 - restartWidth) / 2, Y: 3, W: restartWidth, H: 1}) {
		t.Fatalf("exited restart hit region should follow centered text, got %#v", restart)
	}
	picker := hitRegionByAction(t, result.HitRegions, ActionExitedReconnect.String())
	pickerWidth := DisplayWidth("[ reconnect ]")
	if picker.Rect != (Rect{X: (80 - pickerWidth) / 2, Y: 4, W: pickerWidth, H: 1}) {
		t.Fatalf("exited picker hit region should follow centered text, got %#v", picker)
	}
}

func TestContentViewportBottomAlignsExitedPaneTailAndActions(t *testing.T) {
	lines := []Line{
		NewLine("history A"),
		NewLine("history B"),
		NewLine("history C"),
		NewLine("history D"),
		NewLine(""),
		NewLine("terminal exited: term-1 code:23"),
		NewLine("exited at: 2026-06-17T12:30:00Z"),
		NewLine("command: bash -lc exit 23"),
		centeredStyledLine("► restart ◄", StyleWarning),
		centeredStyledLine("[ reconnect ]", StyleMuted),
	}
	regions := []HitRegion{
		{Kind: HitRegionContentAction, Rect: Rect{Y: 8, W: DisplayWidth("► restart ◄"), H: 1}, ActionID: ActionExitedRestart.String()},
		{Kind: HitRegionContentAction, Rect: Rect{Y: 9, W: DisplayWidth("[ reconnect ]"), H: 1}, ActionID: ActionExitedReconnect.String()},
	}

	result := RenderContentViewport(ContentRenderRequest{
		Rect:    Rect{W: 80, H: 8},
		Content: ContentVM{Kind: ContentExitedPane, Lines: lines, HitRegions: regions},
	})

	got := plainContentViewportLines(result.Lines)
	want := []string{
		"history C",
		"history D",
		"",
		"terminal exited: term-1 code:23",
		"exited at: 2026-06-17T12:30:00Z",
		"command: bash -lc exit 23",
		"► restart ◄",
		"[ reconnect ]",
	}
	for index, wantLine := range want {
		if strings.TrimSpace(got[index]) != wantLine {
			t.Fatalf("exited content should render the tail after history row=%d got=%q want=%q all=%#v", index, got[index], wantLine, got)
		}
	}
	if !strings.HasPrefix(got[0], "history C") || !strings.HasPrefix(got[3], "terminal exited") {
		t.Fatalf("exited terminal text should stay left-aligned while actions are centered, got %#v", got)
	}
	restart := hitRegionByAction(t, result.HitRegions, ActionExitedRestart.String())
	restartWidth := DisplayWidth("► restart ◄")
	if restart.Rect != (Rect{X: (80 - restartWidth) / 2, Y: 6, W: restartWidth, H: 1}) {
		t.Fatalf("restart hit region should follow tail-aligned action, got %#v", restart)
	}
	picker := hitRegionByAction(t, result.HitRegions, ActionExitedReconnect.String())
	pickerWidth := DisplayWidth("[ reconnect ]")
	if picker.Rect != (Rect{X: (80 - pickerWidth) / 2, Y: 7, W: pickerWidth, H: 1}) {
		t.Fatalf("picker hit region should follow tail-aligned action, got %#v", picker)
	}
}

func TestFrameworkRendersLiveExtentBoundaryDotsWithChromeOverflowMarker(t *testing.T) {
	result := NewRenderer(DefaultTheme()).RenderResult(RenderVM{Shell: ShellVM{
		Layout: LayoutVM{Viewport: Rect{W: 16, H: 8}, Panels: []PanelVM{{
			ID:           "pane-1",
			Title:        "pane",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:   ContentTerminalLive,
				Lines:  []Line{NewLine("abcdef")},
				Extent: ContentExtent{Known: true, Cols: 3, Rows: 1},
			},
		}}},
	}})

	layer := firstLayer(t, result, LayerPanel)
	if got, want := plainContentViewportLines(layer.Lines), []string{
		"abc···········",
		"··············",
		"··············",
		"··············",
		"··············",
		"··············",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("panel layer should keep content viewport projection\n got=%#v\nwant=%#v", got, want)
	}
	if layer.ContentOverflow != (ContentOverflow{Right: true}) {
		t.Fatalf("terminal live resize extent mismatch should expose chrome overflow, got %#v", layer.ContentOverflow)
	}
	lines := result.Lines()
	if got := SliceCells(lines[6], 15, 16); got != ">" {
		t.Fatalf("right overflow marker should be drawn on pane right edge, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[7], 15, 16); got != "┘" {
		t.Fatalf("right overflow marker should keep pane corner, got %q frame=%#v", got, lines)
	}
	if strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), ">") {
		t.Fatalf("right overflow marker must not be written into content layer, got %#v", layer.Lines)
	}
}

func TestFrameworkRendersTerminalLiveBottomOverflowOnPaneChrome(t *testing.T) {
	result := NewRenderer(DefaultTheme()).RenderResult(RenderVM{Shell: ShellVM{
		Layout: LayoutVM{Viewport: Rect{W: 16, H: 8}, Panels: []PanelVM{{
			ID:           "pane-1",
			Title:        "pane",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind: ContentTerminalLive,
				Lines: []Line{
					NewLine("row 1"),
					NewLine("row 2"),
					NewLine("row 3"),
				},
				Extent: ContentExtent{Known: true, Cols: 14, Rows: 8},
			},
		}}},
	}})

	layer := firstLayer(t, result, LayerPanel)
	if layer.ContentOverflow != (ContentOverflow{Bottom: true}) {
		t.Fatalf("terminal live bottom overflow should reach chrome, got %#v", layer.ContentOverflow)
	}
	lines := result.Lines()
	if got := SliceCells(lines[7], 14, 15); got != "v" {
		t.Fatalf("bottom overflow marker should be drawn on pane bottom-right corner, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[7], 15, 16); got != "┘" {
		t.Fatalf("bottom overflow marker should keep pane corner, got %q frame=%#v", got, lines)
	}
	if strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), "v") {
		t.Fatalf("bottom overflow marker must not be written into content layer, got %#v", layer.Lines)
	}
}

func TestFrameworkRendersTerminalLiveLeftTopOverflowOnPaneChrome(t *testing.T) {
	result := NewRenderer(DefaultTheme()).RenderResult(RenderVM{Shell: ShellVM{
		Layout: LayoutVM{Viewport: Rect{W: 16, H: 8}, Panels: []PanelVM{{
			ID:           "pane-1",
			Title:        "pane",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:   ContentTerminalLive,
				Lines:  []Line{NewLine("abcdefghijklmn"), NewLine("opqrstuvwxyz")},
				Extent: ContentExtent{Known: true, Cols: 14, Rows: 6},
				Layout: ContentLayoutVM{Known: true, PanX: 2, PanY: 1},
			},
		}}},
	}})

	layer := firstLayer(t, result, LayerPanel)
	if layer.ContentOverflow != (ContentOverflow{Left: true, Top: true}) {
		t.Fatalf("terminal live pan should expose left/top chrome overflow, got %#v", layer.ContentOverflow)
	}
	lines := result.Lines()
	if got := SliceCells(lines[0], 0, 1); got != "┌" {
		t.Fatalf("left/top overflow marker should keep pane corner, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[0], 1, 2); got != "^" {
		t.Fatalf("top overflow marker should be drawn before pane title/lock area, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[1], 0, 1); got != "<" {
		t.Fatalf("left overflow marker should be drawn on pane left edge, got %q frame=%#v", got, lines)
	}
	if strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), "<") ||
		strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), "^") {
		t.Fatalf("left/top overflow markers must stay out of content layer, got %#v", layer.Lines)
	}
}

func TestFrameworkRendersConfiguredOverflowMarkersAndExtentDots(t *testing.T) {
	ResetPaneChromeGlyphs()
	t.Cleanup(ResetPaneChromeGlyphs)
	SetPaneChromeGlyphs(PaneChromeGlyphs{
		OverflowLeft:              "‹",
		OverflowLeftSet:           true,
		OverflowRight:             "›",
		OverflowRightSet:          true,
		OverflowTop:               "˄",
		OverflowTopSet:            true,
		OverflowBottom:            "˅",
		OverflowBottomSet:         true,
		OverflowStyle:             "#c7c7c7",
		OverflowStyleSet:          true,
		ExtentPlaceholder:         "•",
		ExtentPlaceholderSet:      true,
		ExtentPlaceholderStyle:    "#a8a8a8",
		ExtentPlaceholderStyleSet: true,
	})
	result := NewRenderer(DefaultTheme()).RenderResult(RenderVM{Shell: ShellVM{
		Layout: LayoutVM{Viewport: Rect{W: 16, H: 8}, Panels: []PanelVM{{
			ID:           "pane-1",
			Title:        "pane",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:   ContentTerminalLive,
				Lines:  []Line{NewLine("abcdefghijklmn"), NewLine("opqrstuvwxyz")},
				Extent: ContentExtent{Known: true, Cols: 8, Rows: 8},
				Layout: ContentLayoutVM{Known: true, PanX: 1, PanY: 1},
			},
		}}},
	}})

	layer := firstLayer(t, result, LayerPanel)
	if layer.ContentOverflow != (ContentOverflow{Left: true, Right: true, Top: true, Bottom: true}) {
		t.Fatalf("configured marker case should expose all overflow directions, got %#v", layer.ContentOverflow)
	}
	lines := result.Lines()
	if got := SliceCells(lines[0], 1, 2); got != "˄" {
		t.Fatalf("configured top marker not rendered at top edge, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[1], 0, 1); got != "‹" {
		t.Fatalf("configured left marker not rendered at left edge, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[6], 15, 16); got != "›" {
		t.Fatalf("configured right marker not rendered at right edge, got %q frame=%#v", got, lines)
	}
	if got := SliceCells(lines[7], 14, 15); got != "˅" {
		t.Fatalf("configured bottom marker not rendered at bottom edge, got %q frame=%#v", got, lines)
	}
	if !strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), "•") {
		t.Fatalf("configured extent placeholder should be used in content projection, got %#v", layer.Lines)
	}
	ansi := strings.Join(result.ANSILines(), "\n")
	if !strings.Contains(ansi, "\x1b[38;2;199;199;199m") {
		t.Fatalf("configured overflow marker color should be rendered as ANSI, got %#v", result.ANSILines())
	}
	if !strings.Contains(ansi, "\x1b[38;2;168;168;168m") {
		t.Fatalf("configured extent placeholder color should be rendered as ANSI, got %#v", result.ANSILines())
	}
}

func TestFrameworkRendersTerminalLiveExtentFromBuilder(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 18, Rows: 8},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Cols:       6,
			Rows:       2,
			Ready:      true,
			Screen: [][]state.LiveCell{
				{{Text: "abc", Width: 3}},
				{{Text: "你好", Width: 4, FG: "ansi:2"}},
			},
			Cursor: state.LiveCursor{Visible: true, Row: 1, Col: 4, Shape: "bar"},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 6, Rows: 2},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-live", 7, 6, 2, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))

	result := NewRenderer(DefaultTheme()).RenderResult(NewRenderVMBuilder().Build(root))
	layer := firstLayer(t, result, LayerPanel)
	if got, wantPrefix := plainContentViewportLines(layer.Lines), []string{
		"abc   ··········",
		"你好  ··········",
		"················",
	}; !strings.HasPrefix(strings.Join(got, "\n"), strings.Join(wantPrefix, "\n")) {
		t.Fatalf("live content should mark area outside known extent\n got=%#v\nwant prefix=%#v", got, wantPrefix)
	}
	if result.Cursor.Row != 1 || result.Cursor.Col != 4 || result.Cursor.Shape != CursorShapeBar ||
		result.CursorRect != (Rect{X: 5, Y: 3, W: 1, H: 1}) {
		t.Fatalf("live cursor should stay content-local then translate through layout, got cursor=%#v rect=%#v", result.Cursor, result.CursorRect)
	}
	if !styledLinesContainANSI(result.StyledLines(), "你", ANSICellStyle{FG: "ansi:2"}) ||
		!styledLinesContainANSI(result.StyledLines(), "好", ANSICellStyle{FG: "ansi:2"}) {
		t.Fatalf("live ANSI cells should survive ContentViewport/render framework, got %#v", result.StyledLines())
	}
}

func TestFrameworkProjectsTerminalLiveCursorThroughViewLayout(t *testing.T) {
	result := NewRenderer(DefaultTheme()).RenderResult(RenderVM{Shell: ShellVM{
		Layout: LayoutVM{Viewport: Rect{W: 12, H: 6}, Panels: []PanelVM{{
			ID:           "pane-1",
			Title:        "pane",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:   ContentTerminalLive,
				Lines:  []Line{NewLine("abcd"), NewLine("efgh")},
				Extent: ContentExtent{Known: true, Cols: 4, Rows: 2},
				Layout: ContentLayoutVM{Known: true, Mode: "center"},
				Cursor: Cursor{Visible: true, Row: 1, Col: 2, Shape: CursorShapeBar},
			},
		}}},
	}})

	if result.Cursor != (Cursor{Visible: true, Row: 2, Col: 5, Shape: CursorShapeBar}) {
		t.Fatalf("centered live cursor should move with content viewport, got %#v", result.Cursor)
	}
	if result.CursorRect != (Rect{X: 6, Y: 3, W: 1, H: 1}) {
		t.Fatalf("centered live cursor rect should include panel content origin, got %#v", result.CursorRect)
	}
}

func TestFrameworkHidesTerminalLiveCursorWhenViewLayoutClipsIt(t *testing.T) {
	result := NewRenderer(DefaultTheme()).RenderResult(RenderVM{Shell: ShellVM{
		Layout: LayoutVM{Viewport: Rect{W: 12, H: 6}, Panels: []PanelVM{{
			ID:           "pane-1",
			Title:        "pane",
			Presentation: PanelPresentationCard,
			Active:       true,
			Content: ContentVM{
				Kind:   ContentTerminalLive,
				Lines:  []Line{NewLine("abcd"), NewLine("efgh")},
				Extent: ContentExtent{Known: true, Cols: 4, Rows: 2},
				Layout: ContentLayoutVM{Known: true, PanX: 8},
				Cursor: Cursor{Visible: true, Row: 0, Col: 1, Shape: CursorShapeBar},
			},
		}}},
	}})

	if result.Cursor.Visible || result.Cursor.Anchor || result.CursorRect.W != 0 {
		t.Fatalf("clipped live cursor should be hidden instead of anchored at old origin, cursor=%#v rect=%#v", result.Cursor, result.CursorRect)
	}
}

func TestContentViewportKeepsEmojiBeforeExtentDots(t *testing.T) {
	result := RenderContentViewport(ContentRenderRequest{
		Rect: Rect{W: 6, H: 1},
		Content: ContentVM{
			Kind:   ContentTerminalLive,
			Lines:  []Line{NewLine("x🚀")},
			Extent: ContentExtent{Known: true, Cols: 3, Rows: 1},
		},
	})

	if got := result.Lines[0].PlainString(); got != "x🚀···" {
		t.Fatalf("live viewport should keep extent dots after emoji, got %q", got)
	}
	ansi := result.Lines[0].ANSIString(DefaultTheme())
	if !strings.Contains(ansi, "x🚀\x1b[4G") || !strings.Contains(ansi, "·") {
		t.Fatalf("live viewport should place dots at the model cell boundary, got %q", ansi)
	}
}

func TestFrameworkRendersTerminalLiveOverflowFromBuilder(t *testing.T) {
	root := state.Root{
		Viewport: state.ViewportStore{Valid: true, Cols: 18, Rows: 8},
		Surface: state.TerminalSurfaceStore{
			TerminalID: "term-live",
			Cols:       20,
			Rows:       8,
			Ready:      true,
			Lines:      []string{"lozzow@RedmiBook"},
		},
		Session: state.TerminalSessionStore{TerminalID: "term-live", Attached: true, Cols: 20, Rows: 8},
	}
	root.TerminalViews = root.TerminalViews.BindPane(state.NewPaneTerminalView(state.DefaultPaneID, "term-live", 7, 20, 8, state.TerminalResizeRoleOwner, "surface", state.TerminalPaneViewID(state.DefaultPaneID), true))

	result := NewRenderer(DefaultTheme()).RenderResult(NewRenderVMBuilder().Build(root))
	layer := firstLayer(t, result, LayerPanel)
	if layer.ContentOverflow != (ContentOverflow{Right: true, Bottom: true}) {
		t.Fatalf("terminal live extent mismatch should expose chrome overflow, got %#v", layer.ContentOverflow)
	}
	lines := result.Lines()
	rightRow := layer.Rect.Y + layer.Rect.H - 2
	rightCol := layer.Rect.X + layer.Rect.W - 1
	if got := SliceCells(lines[rightRow], rightCol, rightCol+1); got != ">" {
		t.Fatalf("right overflow marker should be drawn on pane right edge, got %q frame=%#v", got, lines)
	}
	bottomRow := layer.Rect.Y + layer.Rect.H - 1
	bottomCol := layer.Rect.X + layer.Rect.W - 2
	if got := SliceCells(lines[bottomRow], bottomCol, bottomCol+1); got != "v" {
		t.Fatalf("bottom overflow marker should be drawn on pane bottom edge, got %q frame=%#v", got, lines)
	}
	cornerCol := layer.Rect.X + layer.Rect.W - 1
	if got := SliceCells(lines[bottomRow], cornerCol, cornerCol+1); got != "┘" {
		t.Fatalf("overflow marker should keep pane bottom-right corner, got %q frame=%#v", got, lines)
	}
	if strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), ">") ||
		strings.Contains(strings.Join(plainContentViewportLines(layer.Lines), "\n"), "v") {
		t.Fatalf("overflow markers must stay out of live content layer, got %#v", layer.Lines)
	}
}

func TestRenderContentWritesRequestedOwnerLayer(t *testing.T) {
	c := newCanvas(12, 3)
	result := renderContent(c, ContentVM{
		Kind:   ContentTerminalLive,
		Lines:  []Line{NewLine("float")},
		Extent: ContentExtent{Known: true, Cols: 8, Rows: 2},
	}, Rect{X: 1, Y: 1, W: 8, H: 2}, "floating:float-1:content", LayerFloating)

	if got := result.Lines[0].PlainString(); got != "float   " {
		t.Fatalf("expected content viewport output, got %q", got)
	}
	cell := c.rows[1][1]
	if cell.text != "float" || cell.width != 5 || cell.owner != "floating:float-1:content" || cell.layer != LayerFloating {
		t.Fatalf("content writer must preserve requested owner/layer, got %#v", cell)
	}
	if continuation := c.rows[1][2]; !continuation.continuation || continuation.owner != "floating:float-1:content" || continuation.layer != LayerFloating {
		t.Fatalf("content continuation must preserve requested owner/layer, got %#v", continuation)
	}
}

func plainContentViewportLines(lines []Line) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = line.PlainString()
	}
	return out
}

func assertContentViewportLineWidths(t *testing.T, lines []Line, width int) {
	t.Helper()
	for i, line := range lines {
		if got := line.Width(); got != width {
			t.Fatalf("line %d width=%d want=%d line=%#v plain=%q", i, got, width, line, line.PlainString())
		}
	}
}

func contentViewportLineHasANSI(line Line, text string, style ANSICellStyle) bool {
	for _, cell := range line.Cells {
		if cell.Text == text && cell.ANSIStyle == style {
			return true
		}
	}
	return false
}
