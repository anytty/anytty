package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestCopyHistoryContentANSILineAtOffsetsInternalColumnAnchors(t *testing.T) {
	history := state.HistoryStore{
		Rows: []state.HistoryRow{{
			Text:   "099975",
			LineID: 99,
			Cells: []state.HistoryCell{
				{Text: "0", Width: 1},
				{Text: "99975", Width: 5},
			},
		}},
	}

	got := CopyHistoryContentANSILineAt(history, state.CopyModeStore{}, 0, 10, 1, DefaultTheme())
	if !strings.Contains(got, "0\x1b[3G99975") || strings.Contains(got, "0\x1b[2G99975") {
		t.Fatalf("pane-local patch must offset ANSI column anchors, got %q", got)
	}
}

func TestCopyHistoryContentANSILineUsesTailFillDisplayOnly(t *testing.T) {
	tail := state.HistoryCellStyle{BG: "idx:24"}
	row := state.HistoryRow{
		Text:     "ij",
		LineID:   99,
		TailFill: &tail,
		Cells: []state.HistoryCell{
			{Text: "i", Width: 1, Style: tail},
			{Text: "j", Width: 1, Style: tail},
		},
	}
	if got := state.HistoryRowDisplayWidth(row); got != 2 {
		t.Fatalf("tail fill must not affect logical row width, got %d", got)
	}
	ansi := CopyHistoryContentANSILineAt(state.HistoryStore{Cols: 8, Rows: []state.HistoryRow{row}}, state.CopyModeStore{BoundCols: 8}, 0, 8, 0, DefaultTheme())
	if !strings.Contains(ansi, "\x1b[48;5;24m      ") {
		t.Fatalf("tail fill should render display-only background to EOL, got %q", ansi)
	}
}

func TestCopyHistoryDefaultBlankCellsRenderAsViewportBackground(t *testing.T) {
	row := state.HistoryRow{
		Text:   "    ",
		LineID: 99,
		Kind:   state.HistoryRowKindArchivedScreenFrame,
		Cells: []state.HistoryCell{
			{Text: " ", Width: 1},
			{Text: " ", Width: 1},
			{Text: " ", Width: 1},
			{Text: " ", Width: 1},
		},
	}

	ansi := CopyHistoryContentANSILineAt(state.HistoryStore{Cols: 8, Rows: []state.HistoryRow{row}}, state.CopyModeStore{BoundCols: 8}, 0, 8, 0, DefaultTheme())
	if strings.Contains(ansi, "\x1b[49m") || strings.Contains(ansi, "\x1b[39m") || strings.Contains(ansi, "\x1b[48;") {
		t.Fatalf("default blank frame cells should not force terminal ANSI background, got %q", ansi)
	}
	plain := ansiEscapePattern.ReplaceAllString(ansi, "")
	if got := DisplayWidth(plain); got != 8 {
		t.Fatalf("default blank frame row should still occupy viewport columns, got width=%d ansi=%q plain=%q", got, ansi, plain)
	}
}

func TestCopyHistoryContentANSILineSelectionFillsRowTail(t *testing.T) {
	history := state.HistoryStore{
		Cols: 8,
		Rows: []state.HistoryRow{{
			Text:   "uv",
			LineID: 1,
			Cells: []state.HistoryCell{
				{Text: "u", Width: 1},
				{Text: "v", Width: 1},
			},
		}},
	}
	copyMode := state.CopyModeStore{
		Active:    true,
		BoundCols: 8,
		Selection: &state.CopySelection{
			Anchor: state.CopyPosition{Row: 0, Col: 0},
			Focus:  state.CopyPosition{Row: 0, Col: 8},
		},
	}

	line := copyHistoryLine(history.Rows[0], 0, normalizedCopySelection(copyMode), copySearchRange{}, 8)
	if len(line.Cells) != 3 || line.Cells[2].Width != 6 || line.Cells[2].ANSIStyle != copyHistorySelectionANSIStyle {
		t.Fatalf("selection should paint display-only row tail like tmux, got %#v", line.Cells)
	}
	if got := line.Width(); got != 8 {
		t.Fatalf("selection line should fill to viewport width, got %d cells=%#v", got, line.Cells)
	}
	if got := state.HistoryRowDisplayWidth(history.Rows[0]); got != 2 {
		t.Fatalf("selection fill must not mutate logical row width, got %d", got)
	}
}

func TestCopyHistoryContentANSILineAnchorsAfterStyledWideCells(t *testing.T) {
	style := state.HistoryCellStyle{BG: "idx:236"}
	history := state.HistoryStore{
		Cols: 12,
		Rows: []state.HistoryRow{{
			Text:   "验证 ok",
			LineID: 1,
			Cells: []state.HistoryCell{
				{Text: "验", Width: 2, Style: style},
				{Text: "证", Width: 2, Style: style},
				{Text: " ok", Width: 3, Style: style},
			},
			TailFill: &style,
		}},
	}

	ansi := CopyHistoryContentANSILineAt(history, state.CopyModeStore{BoundCols: 12}, 0, 12, 0, DefaultTheme())
	if !strings.Contains(ansi, "\x1b[3G") || !strings.Contains(ansi, "\x1b[5G") || !strings.Contains(ansi, "\x1b[8G") || strings.Contains(ansi, "\x1b[2G") || strings.Contains(ansi, "\x1b[4G") {
		t.Fatalf("styled wide cells should advance ANSI anchors by display width, got %q", ansi)
	}
	if !strings.Contains(ansi, "\x1b[48;5;236m     ") {
		t.Fatalf("tail fill should start after wide text footprint, got %q", ansi)
	}
}

func TestCopyHistoryCursorClampsToVisibleViewport(t *testing.T) {
	history := state.HistoryStore{
		Cols: 10,
		Rows: []state.HistoryRow{
			{Text: "one", LineID: 1},
			{Text: "two", LineID: 2},
			{Text: "three", LineID: 3},
			{Text: "four", LineID: 4},
			{Text: "five", LineID: 5},
		},
	}
	cursor := copyHistoryCursor(history, state.CopyModeStore{
		Active:      true,
		ViewportTop: 1,
		ViewRows:    2,
		Cursor:      state.CopyPosition{Row: 4, Col: 2},
	})
	if !cursor.Visible || cursor.Row != 1 {
		t.Fatalf("copy cursor must stay inside visible viewport, got %#v", cursor)
	}
}

func TestR411CopyHistoryUsesContinuousRowsWithoutScreenRowPadding(t *testing.T) {
	history := state.HistoryStore{
		Cols: 12,
		Rows: []state.HistoryRow{
			{Text: "shell prompt", LineID: 1, Segment: state.HistoryCursorSegmentCommitted},
			{
				Text:         "OpenAI Codex",
				LineID:       20,
				Kind:         state.HistoryRowKindScreenFrame,
				Segment:      state.HistoryCursorSegmentCurrentPrimaryFrame,
				SessionID:    1,
				FrameID:      10,
				FixedGrid:    true,
				ScreenCols:   12,
				ScreenRow:    4,
				ScreenRowSet: true,
			},
			{
				Text:         "Use /skills",
				LineID:       21,
				Kind:         state.HistoryRowKindScreenFrame,
				Segment:      state.HistoryCursorSegmentCurrentPrimaryFrame,
				SessionID:    1,
				FrameID:      10,
				FixedGrid:    true,
				ScreenCols:   12,
				ScreenRow:    5,
				ScreenRowSet: true,
			},
		},
	}
	copyMode := state.CopyModeStore{
		Active:      true,
		ViewportTop: 1,
		ViewRows:    8,
		BoundCols:   12,
		Cursor:      state.CopyPosition{Row: 2},
	}

	lines := copyHistoryLines(history, copyMode)
	if len(lines) != 2 {
		t.Fatalf("expected only visible authoritative rows, got %#v", lines)
	}
	if got := lines[0].String(); got != "OpenAI Codex" {
		t.Fatalf("first visible row should be viewport top, got %q lines=%#v", got, lines)
	}
	if got := lines[1].String(); got != "Use /skills" {
		t.Fatalf("second visible row should stay contiguous, got %q lines=%#v", got, lines)
	}
	cursor := copyHistoryCursor(history, copyMode)
	if !cursor.Visible || cursor.Row != 1 {
		t.Fatalf("cursor row should use viewport-local y without padding, got %#v", cursor)
	}
	regions := copyHistoryHitRegions(history, copyMode)
	if len(regions) == 0 || regions[0].Rect.Y != 0 {
		t.Fatalf("hit regions should use contiguous viewport-local y, got %#v", regions)
	}
}

func TestCopyHistoryAllowsViewportAnchoredAfterLastHistoryRow(t *testing.T) {
	history := state.HistoryStore{
		Cols: 12,
		Rows: []state.HistoryRow{
			{Text: "old shell prompt", LineID: 1},
			{Text: "old shell marker", LineID: 2},
		},
	}
	copyMode := state.CopyModeStore{Active: true, ViewRows: 8, ViewportTop: len(history.Rows), Cursor: state.CopyPosition{Row: 1}}

	lines := copyHistoryLines(history, copyMode)
	if len(lines) != 0 {
		t.Fatalf("viewport after the final history row should render only terminal background, got %#v", lines)
	}
	if cursor := copyHistoryCursor(history, copyMode); cursor.Visible {
		t.Fatalf("history cursor must be hidden outside the anchored viewport, got %#v", cursor)
	}
}
