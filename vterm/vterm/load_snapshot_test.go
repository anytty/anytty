package vterm

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	charmvt "github.com/anytty/anytty/vterm/internal/vt"
	uv "github.com/charmbracelet/ultraviolet"
)

func TestLoadSnapshotRestoresScreenAndCursor(t *testing.T) {
	vt := New(10, 4, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{
				{Content: "h", Width: 1},
				{Content: "i", Width: 1},
			},
		},
	}, CursorState{Row: 0, Col: 2, Visible: true}, TerminalModes{AutoWrap: true})

	screen := vt.ScreenContent()
	if got := screen.Cells[0][0].Content + screen.Cells[0][1].Content; got != "hi" {
		t.Fatalf("expected restored content %q, got %q", "hi", got)
	}
	cursor := vt.CursorState()
	if cursor.Col != 2 || cursor.Row != 0 {
		t.Fatalf("expected restored cursor at (2,0), got (%d,%d)", cursor.Col, cursor.Row)
	}

	if _, err := vt.Write([]byte("!")); err != nil {
		t.Fatalf("write after snapshot failed: %v", err)
	}
	screen = vt.ScreenContent()
	if got := screen.Cells[0][0].Content + screen.Cells[0][1].Content + screen.Cells[0][2].Content; got != "hi!" {
		t.Fatalf("expected continued output %q, got %q", "hi!", got)
	}
}

func TestLoadSnapshotWithScrollbackRestoresHistory(t *testing.T) {
	vt := New(6, 3, 100, nil)
	vt.LoadSnapshotWithScrollback([][]Cell{
		{{Content: "o", Width: 1}, {Content: "l", Width: 1}, {Content: "d", Width: 1}},
	}, ScreenData{
		Cells: [][]Cell{
			{
				{Content: "n", Width: 1},
				{Content: "e", Width: 1},
				{Content: "w", Width: 1},
			},
		},
	}, CursorState{Row: 0, Col: 3, Visible: true}, TerminalModes{AutoWrap: true})

	scrollback := vt.ScrollbackContent()
	if len(scrollback) != 1 {
		t.Fatalf("expected 1 restored scrollback row, got %d", len(scrollback))
	}
	if got := scrollback[0][0].Content + scrollback[0][1].Content + scrollback[0][2].Content; got != "old" {
		t.Fatalf("expected restored scrollback %q, got %q", "old", got)
	}
}

func TestLoadSnapshotRestoresStyledBlankRows(t *testing.T) {
	const bg = "#222222"
	vt := New(12, 3, 100, nil)

	styledBlanks := make([]Cell, 12)
	for i := range styledBlanks {
		styledBlanks[i] = Cell{Content: " ", Width: 1, Style: CellStyle{BG: bg}}
	}
	promptRow := append([]Cell(nil), styledBlanks...)
	for i, r := range "> prompt" {
		promptRow[i] = Cell{Content: string(r), Width: 1, Style: CellStyle{BG: bg}}
	}

	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			styledBlanks,
			promptRow,
		},
		IsAlternateScreen: true,
	}, CursorState{Row: 1, Col: 8, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	screen := vt.ScreenContent()
	if got := screen.Cells[0][0].Style.BG; got != bg {
		t.Fatalf("expected pure styled blank row to restore bg %q, got %#v", bg, screen.Cells[0][0])
	}
	if got := screen.Cells[0][11].Style.BG; got != bg {
		t.Fatalf("expected pure styled blank row tail to restore bg %q, got %#v", bg, screen.Cells[0][11])
	}
	if got := screen.Cells[1][0].Style.BG; got != bg {
		t.Fatalf("expected prompt row bg %q, got %#v", bg, screen.Cells[1][0])
	}
	if got := rowText(screen.Cells[1], 8); got != "> prompt" {
		t.Fatalf("expected prompt content restored, got %q", got)
	}
}

func TestVTermResizeReflowsSoftWrappedRows(t *testing.T) {
	vt := New(10, 4, 100, nil)
	if _, err := vt.Write([]byte("abcdefghijk")); err != nil {
		t.Fatalf("write wrapped content: %v", err)
	}
	if !vt.ScreenRowWrappedAt(0) {
		t.Fatalf("expected first row to be marked as soft-wrapped")
	}

	vt.Resize(20, 4)
	if got := rowText(vt.ScreenRowView(0), 11); got != "abcdefghijk" {
		t.Fatalf("expected widened resize to join soft-wrapped row, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 20)); got != "" {
		t.Fatalf("expected second row blank after widening, got %q", got)
	}
	if vt.ScreenRowWrappedAt(0) {
		t.Fatalf("expected joined row to no longer be marked as wrapped")
	}

	vt.Resize(5, 4)
	if got := rowText(vt.ScreenRowView(0), 5); got != "abcde" {
		t.Fatalf("expected row 0 after narrowing, got %q", got)
	}
	if got := rowText(vt.ScreenRowView(1), 5); got != "fghij" {
		t.Fatalf("expected row 1 after narrowing, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(2), 5)); got != "k" {
		t.Fatalf("expected row 2 after narrowing, got %q", got)
	}
	if !vt.ScreenRowWrappedAt(0) || !vt.ScreenRowWrappedAt(1) || vt.ScreenRowWrappedAt(2) {
		t.Fatalf("unexpected wrapped markers after narrowing: %#v", vt.ScreenWrapped())
	}
}

func TestVTermResizeDoesNotJoinHardNewlineRows(t *testing.T) {
	vt := New(10, 4, 100, nil)
	if _, err := vt.Write([]byte("abc\r\nDEF")); err != nil {
		t.Fatalf("write hard newline content: %v", err)
	}

	vt.Resize(20, 4)
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(0), 20)); got != "abc" {
		t.Fatalf("expected hard-newline row 0 preserved, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 20)); got != "DEF" {
		t.Fatalf("expected hard-newline row 1 preserved, got %q", got)
	}
	if vt.ScreenRowWrappedAt(0) {
		t.Fatalf("hard newline row must not be marked as wrapped")
	}
}

func TestVTermResizeDoesNotJoinExactWidthHardNewlineRows(t *testing.T) {
	vt := New(4, 4, 100, nil)
	if _, err := vt.Write([]byte("ABCD\r\nWXYZ")); err != nil {
		t.Fatalf("write exact-width hard newline content: %v", err)
	}
	if vt.ScreenRowWrappedAt(0) {
		t.Fatalf("exact-width newline row must not be marked as wrapped before resize")
	}

	vt.Resize(8, 4)
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(0), 8)); got != "ABCD" {
		t.Fatalf("expected exact-width hard-newline row 0 preserved after resize, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 8)); got != "WXYZ" {
		t.Fatalf("expected exact-width hard-newline row 1 preserved after resize, got %q", got)
	}
	if vt.ScreenRowWrappedAt(0) {
		t.Fatalf("exact-width hard newline row must not be marked as wrapped after resize")
	}
}

func TestVTermResizeSplitsAndRejoinsHardNewlineArtRows(t *testing.T) {
	vt := New(8, 6, 100, nil)
	if _, err := vt.Write([]byte("AA  BB  \r\nCCDDCCDD\r\nuri: ok")); err != nil {
		t.Fatalf("write art content: %v", err)
	}

	vt.Resize(4, 6)
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(0), 4)); got != "AA" {
		t.Fatalf("expected hard-newline art row 0 split after shrink, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 4)); got != "BB" {
		t.Fatalf("expected hard-newline art row 1 split after shrink, got %q", got)
	}
	if got := rowText(vt.ScreenRowView(2), 4); got != "CCDD" {
		t.Fatalf("expected hard-newline art row 2 split after shrink, got %q", got)
	}
	if got := rowText(vt.ScreenRowView(3), 4); got != "CCDD" {
		t.Fatalf("expected hard-newline art row 3 split after shrink, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(4), 4)); got != "uri:" {
		t.Fatalf("expected following text row to stay after split rows, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(5), 4)); got != "ok" {
		t.Fatalf("expected final text row to stay after split rows, got %q", got)
	}
	if wrapped := vt.ScreenWrapped(); !wrapped[0] || wrapped[1] || !wrapped[2] || wrapped[3] || !wrapped[4] || wrapped[5] {
		t.Fatalf("unexpected wrapped markers after shrink: %#v", wrapped)
	}

	vt.Resize(8, 6)
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(0), 8)); got != "AA  BB" {
		t.Fatalf("expected hard-newline art row 0 rejoined after grow, got %q", got)
	}
	if got := rowText(vt.ScreenRowView(1), 8); got != "CCDDCCDD" {
		t.Fatalf("expected hard-newline art row 1 rejoined after grow, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(2), 8)); got != "uri: ok" {
		t.Fatalf("expected following text row rejoined after grow, got %q", got)
	}
	if wrapped := vt.ScreenWrapped(); wrapped[0] || wrapped[1] || wrapped[2] {
		t.Fatalf("expected rejoined hard-newline rows to clear visible wrap markers, got %#v", wrapped)
	}
}

func TestVTermResizePreservesHardNewlineTrailingSpaceColumns(t *testing.T) {
	vt := New(4, 6, 100, nil)
	if _, err := vt.Write([]byte("AA  \r\nBB")); err != nil {
		t.Fatalf("write hard-newline trailing spaces: %v", err)
	}

	vt.Resize(2, 6)

	if got := rowText(vt.ScreenRowView(0), 2); got != "AA" {
		t.Fatalf("expected first split row to contain AA, got %q", got)
	}
	if !vt.ScreenRowWrappedAt(0) {
		t.Fatalf("expected trailing hard-newline spaces to keep row 0 wrapped after shrink")
	}
	if got := rowText(vt.ScreenRowView(1), 2); got != "  " {
		t.Fatalf("expected second split row to preserve trailing spaces, got %q", got)
	}
	if vt.ScreenRowWrappedAt(1) {
		t.Fatalf("expected trailing-space split row to terminate the hard-newline row")
	}
	if got := rowText(vt.ScreenRowView(2), 2); got != "BB" {
		t.Fatalf("expected following hard-newline row to stay below trailing spaces, got %q", got)
	}
}

func TestVTermScrollbackUsesStoredLineLengthAfterResize(t *testing.T) {
	vt := New(4, 2, 100, nil)
	if _, err := vt.Write([]byte("AA  \r\nBB\r\nCC")); err != nil {
		t.Fatalf("write scrollback trailing spaces: %v", err)
	}

	scrollback := vt.ScrollbackContent()
	if len(scrollback) == 0 {
		t.Fatal("expected scrollback row before resize")
	}
	if got := rowText(scrollback[0], 4); got != "AA  " {
		t.Fatalf("expected scrollback to preserve trailing spaces before resize, got %q row=%#v", got, scrollback[0])
	}

	vt.Resize(2, 2)
	scrollback = vt.ScrollbackContent()
	if len(scrollback) < 2 {
		t.Fatalf("expected resize to reflow scrollback rows, got %d", len(scrollback))
	}
	if got := rowText(scrollback[0], 2); got != "AA" {
		t.Fatalf("expected reflowed scrollback row 0, got %q row=%#v", got, scrollback[0])
	}
	if got := rowText(scrollback[1], 2); got != "  " {
		t.Fatalf("expected trailing-space scrollback segment after resize, got %q row=%#v", got, scrollback[1])
	}
}

func TestVTermResizeWithDamageAppendsRowsDisplacedByShrink(t *testing.T) {
	vt := New(12, 4, 0, nil)
	vt.DisableEmulatorScrollback()
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			cellsFromString("row 000098"),
			cellsFromString("row 000099"),
			cellsFromString("row 000100"),
			cellsFromString("prompt-tail"),
		},
	}, CursorState{Row: 3, Col: 10, Visible: true}, TerminalModes{AutoWrap: true})

	damage := vt.ResizeWithDamage(6, 4)
	if !damage.RequiresFullReplace || damage.FullReplaceReason != "resize" {
		t.Fatalf("expected resize full-replace damage, got %#v", damage)
	}
	gotRows := append(damageRowsText(damage.ScrollbackAppend), screenRowsText(vt.ScreenContent().Cells)...)
	joined := strings.Join(gotRows, "")
	for _, want := range []string{"row 000098", "row 000099", "row 000100", "prompt-tail"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected resize damage plus screen rows to contain %q, got %#v", want, gotRows)
		}
	}
}

func TestVTermResizeWithDamagePreservesWrappedBoundary(t *testing.T) {
	vt := New(5, 2, 0, nil)
	vt.DisableEmulatorScrollback()
	if _, err := vt.Write([]byte("abcdefghij")); err != nil {
		t.Fatalf("write wrapped line: %v", err)
	}
	if !vt.ScreenRowWrappedAt(0) {
		t.Fatal("expected first row wrapped before resize")
	}

	damage := vt.ResizeWithDamage(4, 2)
	gotRows := append(damageRowsText(damage.ScrollbackAppend), screenRowsText(vt.ScreenContent().Cells)...)
	if strings.Contains(strings.Join(gotRows, "|"), "ij|ij") {
		t.Fatalf("visible suffix duplicated in resize damage plus screen, got %#v", gotRows)
	}
	if len(damage.ScrollbackAppend) < 1 || !damage.ScrollbackAppend[0].Wrapped {
		t.Fatalf("expected displaced wrapped prefix in damage, got %#v", damage.ScrollbackAppend)
	}
}

func TestVTermResizeWithDamageDoesNotAppendVisibleSuffix(t *testing.T) {
	vt := New(5, 2, 0, nil)
	vt.DisableEmulatorScrollback()
	if _, err := vt.Write([]byte("abcdefghij")); err != nil {
		t.Fatalf("write wrapped line: %v", err)
	}

	damage := vt.ResizeWithDamage(4, 2)
	gotDamageRows := damageRowsText(damage.ScrollbackAppend)
	if !reflect.DeepEqual(gotDamageRows, []string{"abcd"}) {
		t.Fatalf("expected only displaced rows in damage, got %#v screen=%#v wrapped=%#v", gotDamageRows, screenRowsText(vt.ScreenContent().Cells), vt.ScreenWrapped())
	}
	if len(damage.ScrollbackAppend) < 1 || !damage.ScrollbackAppend[0].Wrapped {
		t.Fatalf("expected damage prefix to remain wrapped into visible screen, got %#v", damage.ScrollbackAppend)
	}
}

func TestVTermResizeWithDamageTailPlanDoesNotMatchEarlierDuplicateRows(t *testing.T) {
	vt := New(5, 3, 0, nil)
	vt.DisableEmulatorScrollback()
	base := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	screenTimes := []time.Time{
		base.Add(1 * time.Second),
		base.Add(2 * time.Second),
		base.Add(3 * time.Second),
	}
	vt.LoadSnapshotWithMetadata(nil, nil, nil, ScreenData{
		Cells: [][]Cell{
			cellsFromString("aaaaa"),
			cellsFromString("aaaaa"),
			cellsFromString("aaaaa"),
		},
	}, screenTimes, []string{"first", "second", "tail"}, CursorState{Row: 2, Col: 4, Visible: true}, TerminalModes{AutoWrap: true})

	damage := vt.ResizeWithDamage(3, 2)
	gotDamageRows := damageRowsText(damage.ScrollbackAppend)
	if !reflect.DeepEqual(gotDamageRows, []string{"aaa", "aa", "aaa", "aa"}) {
		t.Fatalf("expected resize plan to append positional prefix, got %#v screen=%#v", gotDamageRows, screenRowsText(vt.ScreenContent().Cells))
	}
	gotKinds := make([]string, 0, len(damage.ScrollbackAppend))
	for _, row := range damage.ScrollbackAppend {
		gotKinds = append(gotKinds, row.RowKind)
	}
	if !reflect.DeepEqual(gotKinds, []string{"first", "first", "second", "second"}) {
		t.Fatalf("expected explicit resize plan to append the older duplicate line first, got row kinds %#v", gotKinds)
	}
	if gotScreenRows := trimmedScreenRowsText(vt.ScreenContent().Cells); !reflect.DeepEqual(gotScreenRows, []string{"aaa", "aa"}) {
		t.Fatalf("expected resize plan to keep positional tail on screen, got %#v", gotScreenRows)
	}
}

func TestVTermResizeWithDamageTailPlanWinsOverVisibleContentMatches(t *testing.T) {
	vt := New(6, 3, 0, nil)
	vt.DisableEmulatorScrollback()
	base := time.Date(2026, 5, 20, 9, 0, 0, 0, time.UTC)
	vt.LoadSnapshotWithMetadata(nil, nil, nil, ScreenData{
		Cells: [][]Cell{
			cellsFromString("abcabc"),
			cellsFromString("abcabc"),
			cellsFromString("xyzxyz"),
		},
	}, []time.Time{base.Add(1 * time.Second), base.Add(2 * time.Second), base.Add(3 * time.Second)}, []string{"older", "newer", "tail"}, CursorState{Row: 2, Col: 5, Visible: true}, TerminalModes{AutoWrap: true})

	damage := vt.ResizeWithDamage(3, 2)
	gotKinds := make([]string, 0, len(damage.ScrollbackAppend))
	for _, row := range damage.ScrollbackAppend {
		gotKinds = append(gotKinds, row.RowKind)
	}
	if len(gotKinds) == 0 {
		t.Fatal("expected resize plan scrollback append")
	}
	if gotKinds[0] != "older" {
		t.Fatalf("expected explicit tail plan to keep positional older rows first, got %#v", gotKinds)
	}
}

func TestVTermResizeWithDamagePreservesWideRows(t *testing.T) {
	vt := New(8, 3, 0, nil)
	vt.DisableEmulatorScrollback()
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{
				{Content: "你", Width: 2},
				{Content: "", Width: 0},
				{Content: "好", Width: 2},
				{Content: "", Width: 0},
				{Content: "A", Width: 1},
			},
			cellsFromString("qr-####"),
			cellsFromString("tail"),
		},
	}, CursorState{Row: 2, Col: 4, Visible: true}, TerminalModes{AutoWrap: true})

	damage := vt.ResizeWithDamage(4, 3)
	gotRows := append(damageRowsContentText(damage.ScrollbackAppend), screenRowsContentText(vt.ScreenContent().Cells)...)
	if !strings.Contains(strings.Join(gotRows, ""), "tail") {
		t.Fatalf("expected tail to remain visible after resize, got %#v", gotRows)
	}
	if len(damage.ScrollbackAppend) == 0 || len(damage.ScrollbackAppend[0].Cells) < 4 {
		t.Fatalf("expected wide row cells in damage, got %#v", damage.ScrollbackAppend)
	}
	if got := damage.ScrollbackAppend[0].Cells[0]; got.Content != "你" || got.Width != 2 {
		t.Fatalf("expected wide anchor preserved, got %#v", got)
	}
	if got := damage.ScrollbackAppend[0].Cells[1]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected wide continuation preserved, got %#v", got)
	}
}

func TestVTermResizeWithDamageDoesNotFallbackMatchOnGrowWithoutPlan(t *testing.T) {
	vt := New(4, 2, 0, nil)
	vt.DisableEmulatorScrollback()
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			cellsFromString("ab"),
			cellsFromString("cd"),
		},
	}, CursorState{Row: 1, Col: 2, Visible: true}, TerminalModes{AutoWrap: true})

	damage := vt.ResizeWithDamage(6, 2)
	if len(damage.ScrollbackAppend) != 0 {
		t.Fatalf("expected grow resize without explicit plan not to run fallback matching, got %#v", damage.ScrollbackAppend)
	}
}

func TestLoadSnapshotDefaultBlankRowsDoNotBecomeUsedRows(t *testing.T) {
	vt := New(4, 3, 100, nil)
	blank := []Cell{
		{Content: " ", Width: 1},
		{Content: " ", Width: 1},
		{Content: " ", Width: 1},
		{Content: " ", Width: 1},
	}

	vt.LoadSizedSnapshotWithExtendedMetadata(
		4,
		3,
		nil,
		nil,
		nil,
		nil,
		ScreenData{Cells: [][]Cell{blank}},
		nil,
		nil,
		nil,
		CursorState{Row: 0, Col: 0, Visible: true},
		TerminalModes{AutoWrap: true},
	)

	if got := len(vt.ScreenRowView(0)); got != 4 {
		t.Fatalf("expected public screen row to remain full width, got %d", got)
	}
	if got := len(vt.UsedScreenRow(0)); got != 4 {
		t.Fatalf("expected used screen row to keep blank row indexable for snapshots, got %d", got)
	}

	vt.Resize(2, 3)
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(0), 2)); got != "" {
		t.Fatalf("expected blank row to stay blank after resize, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 2)); got != "" {
		t.Fatalf("expected blank row not to reflow into extra used row, got %q", got)
	}
}

func TestTrimmedScreenContentDropsOnlyDefaultBlankTail(t *testing.T) {
	vt := New(8, 3, 100, nil)
	if _, err, _ := vt.WriteForLatestFrame([]byte("abc\r\n\x1b[44mxy  ")); err != nil {
		t.Fatalf("write: %v", err)
	}

	screen := vt.TrimmedScreenContent()
	if len(screen.Cells) != 3 {
		t.Fatalf("trimmed live snapshot must preserve screen row count, got %#v", screen.Cells)
	}
	if got := rowText(screen.Cells[0], 8); got != "abc" || len(screen.Cells[0]) != 3 {
		t.Fatalf("plain row should drop only default blank suffix, got text=%q row=%#v", got, screen.Cells[0])
	}
	if got := len(screen.Cells[1]); got < 4 {
		t.Fatalf("styled trailing blanks must survive trim, got %#v", screen.Cells[1])
	}
	for index, cell := range screen.Cells[1][:4] {
		if cell.Style.BG != "ansi:4" {
			t.Fatalf("styled blank %d should keep background, got %#v", index, screen.Cells[1])
		}
	}
	if got := len(screen.Cells[2]); got != 0 {
		t.Fatalf("pure default blank row should be represented without cloned cells, got %#v", screen.Cells[2])
	}
	if got := len(vt.ScreenRowView(2)); got != 8 {
		t.Fatalf("full public screen row must remain indexable, got %d", got)
	}
}

func TestVisitTrimmedScreenRowsMatchesTrimmedScreenContent(t *testing.T) {
	vt := New(8, 3, 100, nil)
	if _, err, _ := vt.WriteForLatestFrame([]byte("abc\r\n\x1b[44mxy  ")); err != nil {
		t.Fatalf("write: %v", err)
	}
	screen := vt.TrimmedScreenContent()
	visited := make([][]Cell, 0, 3)
	info := vt.VisitTrimmedScreenRows(func(_ int, cellCount int, cellAt func(int) Cell) {
		var row []Cell
		if cellCount > 0 {
			row = make([]Cell, 0, cellCount)
		}
		for index := 0; index < cellCount; index++ {
			row = append(row, cellAt(index))
		}
		visited = append(visited, row)
	})
	if info.Cols != 8 || info.Rows != 3 || info.IsAlternateScreen != screen.IsAlternateScreen {
		t.Fatalf("unexpected visitor metadata %#v screen=%#v", info, screen)
	}
	if !reflect.DeepEqual(visited, screen.Cells) {
		t.Fatalf("visited rows changed trimmed screen\nwant %#v\ngot  %#v", screen.Cells, visited)
	}
}

func TestLoadSnapshotWithExtendedMetadataRestoresWrappedRows(t *testing.T) {
	vt := New(5, 3, 100, nil)
	screen := ScreenData{Cells: [][]Cell{
		cellsFromString("abcde"),
		cellsFromString("fgh"),
	}}
	vt.LoadSnapshotWithExtendedMetadata(nil, nil, nil, nil, screen, nil, nil, []bool{true, false}, CursorState{Row: 1, Col: 3, Visible: true}, TerminalModes{AutoWrap: true})

	vt.Resize(20, 3)
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(0), 20)); got != "abcdefgh" {
		t.Fatalf("expected resize to join snapshot soft-wrap rows, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 20)); got != "" {
		t.Fatalf("expected second row blank after join, got %q", got)
	}
}

func TestLoadSizedSnapshotWithExtendedMetadataPreservesSparseScreenGeometry(t *testing.T) {
	vt := New(10, 4, 100, nil)
	screen := ScreenData{Cells: [][]Cell{
		cellsFromString("100000"),
		cellsFromString("python total"),
		cellsFromString("# prompt"),
	}}

	vt.LoadSizedSnapshotWithExtendedMetadata(20, 5, nil, nil, nil, nil, screen, nil, nil, nil, CursorState{Row: 2, Col: 8, Visible: true}, TerminalModes{AutoWrap: true})

	cols, rows := vt.Size()
	if cols != 20 || rows != 5 {
		t.Fatalf("expected sized snapshot geometry 20x5, got %dx%d", cols, rows)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(1), 20)); got != "python total" {
		t.Fatalf("expected sparse row to stay at row 1 without replay reflow, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(2), 20)); got != "# prompt" {
		t.Fatalf("expected prompt row to stay at row 2, got %q", got)
	}
	if got := strings.TrimSpace(rowText(vt.ScreenRowView(3), 20)); got != "" {
		t.Fatalf("expected padded row 3 blank, got %q", got)
	}
}

func cellsFromString(value string) []Cell {
	cells := make([]Cell, 0, len(value))
	for _, r := range value {
		cells = append(cells, Cell{Content: string(r), Width: 1})
	}
	return cells
}

func rowText(row []Cell, limit int) string {
	var b strings.Builder
	for i, cell := range row {
		if i >= limit {
			break
		}
		if cell.Content == "" {
			b.WriteByte(' ')
			continue
		}
		b.WriteString(cell.Content)
	}
	return b.String()
}

func damageOpText(op DamageOp) string {
	if len(op.Runs) > 0 {
		var b strings.Builder
		for _, run := range op.Runs {
			b.WriteString(run.Text)
		}
		return b.String()
	}
	return semanticCellsContent(op.Cells)
}

func damageOpCells(op DamageOp) []Cell {
	if len(op.Cells) > 0 {
		return op.Cells
	}
	if len(op.Runs) == 0 {
		return nil
	}
	var out []Cell
	for _, run := range op.Runs {
		for _, r := range run.Text {
			out = append(out, Cell{Content: string(r), Width: 1, Style: run.Style})
		}
	}
	return out
}

func damageRowsText(rows []DamageOp) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowText(row.Cells, len(row.Cells)))
	}
	return out
}

func screenRowsText(rows [][]Cell) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowText(row, len(row)))
	}
	return out
}

func trimmedScreenRowsText(rows [][]Cell) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, strings.TrimRight(rowText(row, len(row)), " "))
	}
	return out
}

func damageRowsContentText(rows []DamageOp) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowContentText(row.Cells))
	}
	return out
}

func screenRowsContentText(rows [][]Cell) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowContentText(row))
	}
	return out
}

func rowContentText(row []Cell) string {
	var b strings.Builder
	for _, cell := range row {
		b.WriteString(cell.Content)
	}
	return strings.TrimRight(b.String(), " ")
}

func TestLoadSnapshotPreservesWideCellContinuationsAcrossSubsequentWrites(t *testing.T) {
	vt := New(8, 2, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{
				{Content: "你", Width: 2},
				{Content: "", Width: 0},
				{Content: "好", Width: 2},
				{Content: "", Width: 0},
				{Content: "A", Width: 1},
			},
		},
	}, CursorState{Row: 0, Col: 5, Visible: true}, TerminalModes{AutoWrap: true})

	screen := vt.ScreenContent()
	if got := screen.Cells[0][0]; got.Content != "你" || got.Width != 2 {
		t.Fatalf("expected first wide cell restored, got %#v", got)
	}
	if got := screen.Cells[0][1]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected wide-cell continuation placeholder at x=1, got %#v", got)
	}
	if got := screen.Cells[0][2]; got.Content != "好" || got.Width != 2 {
		t.Fatalf("expected second wide cell restored, got %#v", got)
	}
	if got := screen.Cells[0][3]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected wide-cell continuation placeholder at x=3, got %#v", got)
	}

	if _, err := vt.Write([]byte("!")); err != nil {
		t.Fatalf("write after wide-cell snapshot failed: %v", err)
	}

	screen = vt.ScreenContent()
	if got := screen.Cells[0][0]; got.Content != "你" || got.Width != 2 {
		t.Fatalf("expected first wide cell preserved after write, got %#v", got)
	}
	if got := screen.Cells[0][1]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected continuation placeholder preserved at x=1 after write, got %#v", got)
	}
	if got := screen.Cells[0][2]; got.Content != "好" || got.Width != 2 {
		t.Fatalf("expected second wide cell preserved after write, got %#v", got)
	}
	if got := screen.Cells[0][3]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected continuation placeholder preserved at x=3 after write, got %#v", got)
	}
	if got := screen.Cells[0][5]; got.Content != "!" || got.Width != 1 {
		t.Fatalf("expected trailing ASCII write after restored wide cells, got %#v", got)
	}
}

func TestVTermResizePreservesCurrentBackgroundForSubsequentErase(t *testing.T) {
	const bg = "#000000"
	vt := New(120, 4, 100, nil)

	seed := "\x1b[?1049h" +
		"\x1b[48;2;0;0;0m" +
		"\x1b[2;1Hline 001 some content" +
		"\x1b[K"
	if _, err := vt.Write([]byte(seed)); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	if got := vt.ScreenRowView(1)[60].Style.BG; got != bg {
		t.Fatalf("expected seeded tail bg %q before resize, got %#v", bg, vt.ScreenRowView(1)[60])
	}

	vt.Resize(96, 4)

	if _, err := vt.Write([]byte("\x1b[2;1Hline 001 some content\x1b[K")); err != nil {
		t.Fatalf("post-resize erase write: %v", err)
	}

	screen := vt.ScreenContent()
	if len(screen.Cells) <= 1 || len(screen.Cells[1]) < 61 {
		t.Fatalf("unexpected screen dimensions after resize+erase: %#v", screen.Cells)
	}
	if got := screen.Cells[1][60].Style.BG; got != bg {
		t.Fatalf("expected erase after resize to keep bg %q, got %#v", bg, screen.Cells[1][60])
	}
}

func TestVTermResizeWithDamagePreservesLiveTailBackground(t *testing.T) {
	const bg = "#222222"
	vt := New(12, 4, 0, nil)
	vt.DisableEmulatorScrollback()

	if _, err := vt.Write([]byte("seed\r\n\x1b[48;2;34;34;34mij\x1b[K\x1b[0m")); err != nil {
		t.Fatalf("write styled tail row: %v", err)
	}
	if got := vt.ScreenRowView(1)[11].Style.BG; got != bg {
		t.Fatalf("expected live tail bg before resize, got %#v", vt.ScreenRowView(1)[11])
	}

	vt.ResizeWithDamage(6, 4)

	screen := vt.ScreenContent()
	if got := strings.TrimRight(rowText(screen.Cells[1], len(screen.Cells[1])), " "); got != "ij" {
		t.Fatalf("expected resize to keep content row, got %q row=%#v", got, screen.Cells[1])
	}
	if got := screen.Cells[1][5].Style.BG; got != bg {
		t.Fatalf("expected resize to preserve live tail bg %q, got %#v", bg, screen.Cells[1][5])
	}
}

func TestVTermResizeWithDamagePreservesTailFillBeyondUsedWidth(t *testing.T) {
	const bg = "#222222"
	vt := New(12, 4, 0, nil)
	vt.DisableEmulatorScrollback()
	row := make([]Cell, 12)
	for i := range row {
		row[i] = Cell{Content: " ", Width: 1, Style: CellStyle{BG: bg}}
	}
	row[0] = Cell{Content: "i", Width: 1, Style: CellStyle{BG: bg}}
	row[1] = Cell{Content: "j", Width: 1, Style: CellStyle{BG: bg}}
	vt.LoadSnapshot(ScreenData{Cells: [][]Cell{cellsFromString("seed"), row}}, CursorState{Row: 1, Col: 2, Visible: true}, TerminalModes{AutoWrap: true})
	vt.emu.Emulator.SetScreenLineUsed(1, 2)

	vt.ResizeWithDamage(6, 4)

	screen := vt.ScreenContent()
	if got := rowText(vt.UsedScreenRow(1), len(vt.UsedScreenRow(1))); got != "ij" {
		t.Fatalf("expected tail fill not to grow used width after resize, got %q row=%#v", got, vt.UsedScreenRow(1))
	}
	if got := screen.Cells[1][5].Style.BG; got != bg {
		t.Fatalf("expected tail fill beyond used width to survive resize, got %#v", screen.Cells[1][5])
	}
}

func TestLoadSnapshotWithTimestampsRestoresRowTimes(t *testing.T) {
	vt := New(6, 3, 100, nil)
	scrollbackTS := []time.Time{time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)}
	screenTS := []time.Time{time.Date(2026, 4, 7, 10, 0, 1, 0, time.UTC)}

	vt.LoadSnapshotWithTimestamps([][]Cell{
		{{Content: "o", Width: 1}, {Content: "l", Width: 1}, {Content: "d", Width: 1}},
	}, scrollbackTS, ScreenData{
		Cells: [][]Cell{
			{
				{Content: "n", Width: 1},
				{Content: "e", Width: 1},
				{Content: "w", Width: 1},
			},
		},
	}, screenTS, CursorState{Row: 0, Col: 3, Visible: true}, TerminalModes{AutoWrap: true})

	if got := vt.ScrollbackTimestamps(); len(got) != 1 || !got[0].Equal(scrollbackTS[0]) {
		t.Fatalf("unexpected restored scrollback timestamps: %#v", got)
	}
	if got := vt.ScreenTimestamps(); len(got) == 0 || !got[0].Equal(screenTS[0]) {
		t.Fatalf("unexpected restored screen timestamps: %#v", got)
	}
}

func TestLoadSnapshotWithMetadataRestoresRowKinds(t *testing.T) {
	vt := New(6, 3, 100, nil)

	vt.LoadSnapshotWithMetadata([][]Cell{{}}, []time.Time{time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)}, []string{"restart"}, ScreenData{
		Cells: [][]Cell{
			{
				{Content: "n", Width: 1},
				{Content: "e", Width: 1},
				{Content: "w", Width: 1},
			},
		},
	}, []time.Time{time.Date(2026, 4, 7, 10, 0, 1, 0, time.UTC)}, []string{""}, CursorState{Row: 0, Col: 3, Visible: true}, TerminalModes{AutoWrap: true})

	if got := vt.ScrollbackRowKinds(); len(got) != 1 || got[0] != "restart" {
		t.Fatalf("unexpected restored scrollback row kinds: %#v", got)
	}
}

func TestVTermWriteAssignsRowTimestamps(t *testing.T) {
	vt := New(6, 2, 100, nil)

	if _, err := vt.Write([]byte("one\ntwo\nthree\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	scrollbackTS := vt.ScrollbackTimestamps()
	if len(scrollbackTS) == 0 || scrollbackTS[0].IsZero() {
		t.Fatalf("expected scrollback timestamp after scroll, got %#v", scrollbackTS)
	}
	screenTS := vt.ScreenTimestamps()
	if len(screenTS) == 0 || screenTS[0].IsZero() {
		t.Fatalf("expected screen timestamps for visible rows, got %#v", screenTS)
	}
}

func TestVTermWriteWithDamagePreservesWideRuneContinuations(t *testing.T) {
	vt := New(8, 2, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("你a"))
	if err != nil {
		t.Fatalf("write wide rune: %v", err)
	}
	op := firstWriteSpanOp(t, damage)
	if op.Row != 0 || op.Col != 0 {
		t.Fatalf("expected first-row write op, got %#v", op)
	}
	if got := op.Cells[0]; got.Content != "你" || got.Width != 2 {
		t.Fatalf("expected wide rune anchor cell, got %#v", got)
	}
	if got := op.Cells[1]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected wide rune continuation placeholder, got %#v", got)
	}
	if got := op.Cells[2]; got.Content != "a" || got.Width != 1 {
		t.Fatalf("expected trailing ascii cell, got %#v", got)
	}
}

func TestVTermWriteWithDamageNormalizesCombiningRuneClusters(t *testing.T) {
	vt := New(8, 2, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("e\u0301"))
	if err != nil {
		t.Fatalf("write combining rune: %v", err)
	}
	op := firstWriteSpanOp(t, damage)
	if got := op.Cells[0]; got.Content != "é" || got.Width != 1 {
		t.Fatalf("expected normalized grapheme cell, got %#v", got)
	}
	screen := vt.ScreenContent()
	if got := screen.Cells[0][0]; got.Content != "é" || got.Width != 1 {
		t.Fatalf("expected screen to keep normalized grapheme cell, got %#v", got)
	}
}

func TestVTermWriteWithDamageProducesSingleHighColumnSpan(t *testing.T) {
	vt := New(160, 1, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[1;138HZ"))
	if err != nil {
		t.Fatalf("write high-column edit: %v", err)
	}
	op := firstWriteSpanOp(t, damage)
	if op.Row != 0 || op.Col != 137 {
		t.Fatalf("unexpected high-column op metadata: %#v", op)
	}
	if len(op.Cells) != 1 || op.Cells[0].Content != "Z" {
		t.Fatalf("expected single-cell sparse op, got %#v", op.Cells)
	}
}

func TestVTermWriteWithDamageUsesClearToEOLForTailClear(t *testing.T) {
	vt := New(24, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "p", Width: 1},
			{Content: "r", Width: 1},
			{Content: "e", Width: 1},
			{Content: "f", Width: 1},
			{Content: "i", Width: 1},
			{Content: "x", Width: 1},
			{Content: "X", Width: 1},
			{Content: "Y", Width: 1},
			{Content: "Z", Width: 1},
		}},
	}, CursorState{Row: 0, Col: 9, Visible: true}, TerminalModes{AutoWrap: true})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[1;7H\x1b[K"))
	if err != nil {
		t.Fatalf("clear to eol: %v", err)
	}
	op := firstOpWithCode(t, damage, ScreenOpClearToEOL)
	if op.Row != 0 || op.Col != 6 {
		t.Fatalf("unexpected clear-to-eol op: %#v", op)
	}
}

func TestVTermWriteWithDamagePreservesStyledEraseAsSpan(t *testing.T) {
	const bg = "#222222"
	vt := New(12, 2, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[48;2;34;34;34m\x1b[1;1H\x1b[K"))
	if err != nil {
		t.Fatalf("styled erase: %v", err)
	}
	op := firstWriteSpanOp(t, damage)
	if op.Row != 0 || op.Col != 0 || len(op.Cells) != 12 {
		t.Fatalf("expected full-row styled erase span, got %#v", op)
	}
	for i, cell := range op.Cells {
		if cell.Content != " " || cell.Width != 1 || cell.Style.BG != bg {
			t.Fatalf("expected styled blank cell %d with bg %q, got %#v", i, bg, cell)
		}
	}
	if got := vt.ScreenRowView(0)[11].Style.BG; got != bg {
		t.Fatalf("expected screen tail bg %q, got %#v", bg, vt.ScreenRowView(0)[11])
	}
}

func TestVTermClearWithScrollbackPreservesStyledBlankRows(t *testing.T) {
	const bg = "#222222"
	vt := New(12, 2, 100, nil)

	if _, err, _ := vt.WriteWithDamage([]byte("\x1b[48;2;34;34;34m\x1b[1;1H\x1b[K")); err != nil {
		t.Fatalf("styled erase setup: %v", err)
	}
	_, err, damage := vt.WriteWithDamage([]byte("\x1b[2J"))
	if err != nil {
		t.Fatalf("clear screen: %v", err)
	}
	if len(damage.ScrollbackAppend) != 1 {
		t.Fatalf("expected one styled blank row in scrollback damage, got %#v", damage.ScrollbackAppend)
	}
	op := damage.ScrollbackAppend[0]
	switch {
	case len(op.Runs) == 1:
		if op.Runs[0].Text != strings.Repeat(" ", 12) || op.Runs[0].Style.BG != bg {
			t.Fatalf("expected compact styled blank run with bg %q, got %#v", bg, op.Runs)
		}
	case len(op.Cells) == 12:
		for i, cell := range op.Cells {
			if cell.Content != " " || cell.Width != 1 || cell.Style.BG != bg {
				t.Fatalf("expected styled blank damage cell %d with bg %q, got %#v", i, bg, cell)
			}
		}
	default:
		t.Fatalf("expected full styled blank row in damage, got %#v", damage.ScrollbackAppend[0])
	}

	scrollback := vt.ScrollbackContent()
	if len(scrollback) != 1 || len(scrollback[0]) != 12 {
		t.Fatalf("expected one full styled blank row in scrollback, got %#v", scrollback)
	}
	if got := scrollback[0][11].Style.BG; got != bg {
		t.Fatalf("expected scrollback tail bg %q, got %#v", bg, scrollback[0][11])
	}
}

func TestVTermWriteWithDamageCapturesMidRowStyleOnlySpan(t *testing.T) {
	vt := New(24, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "p", Width: 1},
			{Content: "l", Width: 1},
			{Content: "a", Width: 1},
			{Content: "i", Width: 1},
			{Content: "n", Width: 1},
			{Content: "x", Width: 1},
			{Content: "t", Width: 1},
		}},
	}, CursorState{Row: 0, Col: 7, Visible: true}, TerminalModes{AutoWrap: true})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[1;6H\x1b[1mx\x1b[0m"))
	if err != nil {
		t.Fatalf("style-only write: %v", err)
	}
	op := firstWriteSpanOp(t, damage)
	if op.Col != 5 || len(op.Cells) != 1 {
		t.Fatalf("unexpected style-only op window: %#v", op)
	}
	if got := op.Cells[0]; got.Content != "x" || !got.Style.Bold {
		t.Fatalf("expected bold style-only cell, got %#v", got)
	}
}

func TestVTermWriteWithDamageKeepsWideCharSpanBoundaryStable(t *testing.T) {
	vt := New(8, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "你", Width: 2},
			{Content: "", Width: 0},
			{Content: "a", Width: 1},
		}},
	}, CursorState{Row: 0, Col: 3, Visible: true}, TerminalModes{AutoWrap: true})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[1;1H界"))
	if err != nil {
		t.Fatalf("wide-boundary write: %v", err)
	}
	op := firstWriteSpanOp(t, damage)
	if op.Col != 0 || len(op.Cells) != 2 {
		t.Fatalf("expected op expanded to include wide continuation, got %#v", op)
	}
	if got := op.Cells[0]; got.Content != "界" || got.Width != 2 {
		t.Fatalf("expected wide anchor cell, got %#v", got)
	}
	if got := op.Cells[1]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected continuation placeholder preserved, got %#v", got)
	}
}

func TestVTermWriteSelectivelyInvalidatesOnlyChangedScreenRows(t *testing.T) {
	vt := New(6, 2, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{
				{Content: "t", Width: 1},
				{Content: "o", Width: 1},
				{Content: "p", Width: 1},
			},
			{
				{Content: "b", Width: 1},
				{Content: "o", Width: 1},
				{Content: "t", Width: 1},
			},
		},
	}, CursorState{Row: 1, Col: 0, Visible: true}, TerminalModes{AutoWrap: true})

	topBefore := vt.ScreenRowView(0)
	bottomBefore := vt.ScreenRowView(1)
	if len(topBefore) == 0 || len(bottomBefore) == 0 {
		t.Fatalf("expected cached rows, got top=%#v bottom=%#v", topBefore, bottomBefore)
	}

	if _, err := vt.Write([]byte("\x1b[2;1Hnew")); err != nil {
		t.Fatalf("write updated row: %v", err)
	}

	topAfter := vt.ScreenRowView(0)
	bottomAfter := vt.ScreenRowView(1)
	if strings.TrimSpace(rowToString(topAfter)) != "top" {
		t.Fatalf("expected unchanged top row preserved, got %q", rowToString(topAfter))
	}
	if strings.TrimSpace(rowToString(bottomAfter)) != "new" {
		t.Fatalf("expected updated bottom row, got %q", rowToString(bottomAfter))
	}
	if &topAfter[0] != &topBefore[0] {
		t.Fatal("expected unchanged screen row cache to be reused")
	}
	if &bottomAfter[0] == &bottomBefore[0] {
		t.Fatal("expected changed screen row cache to be invalidated")
	}
}

func TestVTermWriteReusesScrolledCachesAcrossScreenAndScrollback(t *testing.T) {
	vt := New(6, 2, 100, nil)
	vt.LoadSnapshotWithScrollback([][]Cell{
		{
			{Content: "h", Width: 1},
			{Content: "i", Width: 1},
			{Content: "s", Width: 1},
			{Content: "t", Width: 1},
		},
	}, ScreenData{
		Cells: [][]Cell{
			{
				{Content: "t", Width: 1},
				{Content: "o", Width: 1},
				{Content: "p", Width: 1},
			},
			{
				{Content: "b", Width: 1},
				{Content: "o", Width: 1},
				{Content: "t", Width: 1},
			},
		},
	}, CursorState{Row: 1, Col: 0, Visible: true}, TerminalModes{AutoWrap: true})

	scrollbackBefore := vt.ScrollbackRowView(0)
	topBefore := vt.ScreenRowView(0)
	bottomBefore := vt.ScreenRowView(1)
	if len(scrollbackBefore) == 0 || len(topBefore) == 0 || len(bottomBefore) == 0 {
		t.Fatalf("expected primed caches, got scrollback=%#v top=%#v bottom=%#v", scrollbackBefore, topBefore, bottomBefore)
	}

	if _, err := vt.Write([]byte("\n")); err != nil {
		t.Fatalf("scroll write: %v", err)
	}

	scrollbackAfter0 := vt.ScrollbackRowView(0)
	scrollbackAfter1 := vt.ScrollbackRowView(1)
	screenAfter0 := vt.ScreenRowView(0)
	screenAfter1 := vt.ScreenRowView(1)
	if strings.TrimSpace(rowToString(scrollbackAfter0)) == "" {
		t.Fatalf("expected existing scrollback row preserved, got %q", rowToString(scrollbackAfter0))
	}
	if strings.TrimSpace(rowToString(scrollbackAfter1)) != "top" {
		t.Fatalf("expected scrolled-off top row in scrollback, got %q", rowToString(scrollbackAfter1))
	}
	if strings.TrimSpace(rowToString(screenAfter0)) != "bot" {
		t.Fatalf("expected bottom row to move into first screen row, got %q", rowToString(screenAfter0))
	}
	if &scrollbackAfter0[0] != &scrollbackBefore[0] {
		t.Fatal("expected retained scrollback cache to be reused")
	}
	if &scrollbackAfter1[0] != &topBefore[0] {
		t.Fatal("expected scrolled-off screen row cache to move into scrollback cache")
	}
	if &screenAfter0[0] != &bottomBefore[0] {
		t.Fatal("expected shifted screen row cache to be reused")
	}
	if len(screenAfter1) == 0 || &screenAfter1[0] == &topBefore[0] {
		t.Fatal("expected newly blank screen row to allocate a fresh cache")
	}
}

func TestVTermWriteAssignsTimestampsToBlankRows(t *testing.T) {
	vt := New(6, 3, 100, nil)

	if _, err := vt.Write([]byte("\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	screenTS := vt.ScreenTimestamps()
	if len(screenTS) < 2 || screenTS[0].IsZero() || screenTS[1].IsZero() {
		t.Fatalf("expected blank rows created by newline to receive timestamps, got %#v", screenTS)
	}
}

func TestVTermWriteAltScreenScrollDoesNotInvalidateWholeScreen(t *testing.T) {
	vt := New(5, 4, 100, nil)
	vt.LoadSnapshot(ScreenData{
		IsAlternateScreen: true,
		Cells: [][]Cell{
			{{Content: "1", Width: 1}},
			{{Content: "2", Width: 1}},
			{{Content: "3", Width: 1}},
			{{Content: "4", Width: 1}},
		},
	}, CursorState{Row: 3, Col: 0, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	_, err, damage := vt.WriteWithDamage([]byte("\n"))
	if err != nil {
		t.Fatalf("alt-screen scroll write: %v", err)
	}
	if damage.RequiresFullReplace {
		t.Fatalf("expected direct scroll ops instead of full replace, got %#v", damage)
	}
	cols, rows := vt.Size()
	foundScroll := false
	for _, op := range damage.Ops {
		if op.Code != ScreenOpScrollRect {
			continue
		}
		foundScroll = true
		if op.Rect.Width != cols || op.Rect.Height != rows || op.Dy != -1 {
			t.Fatalf("unexpected direct scroll op: %#v", op)
		}
	}
	if !foundScroll {
		t.Fatalf("expected direct scroll op from local x/vt damage stream, got %#v", damage.Ops)
	}
}

func TestVTermWriteWithDamageUsesDirectSpanOps(t *testing.T) {
	vt := New(8, 2, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abc"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if len(damage.Ops) == 0 {
		t.Fatalf("expected direct span ops, got %#v", damage)
	}
	span := damage.Ops[0]
	if span.Code != ScreenOpWriteSpan || span.Row != 0 || span.Col != 0 {
		t.Fatalf("unexpected first direct op: %#v", span)
	}
	if got := damageOpText(span); got != "abc" {
		t.Fatalf("unexpected direct span contents: %#v", span)
	}
	semanticSpan := firstSemanticOpWithCode(t, damage, ScreenOpWriteSpan)
	if got := damageOpText(semanticSpan); got != "abc" {
		t.Fatalf("unexpected semantic span contents: %#v", semanticSpan)
	}
}

func TestVTermWriteWithDamageEmitsCursorControlOps(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abcdef\x1b[3DXYZ\x1b[2;5H!"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	cub := firstControlOp(t, damage, "cub")
	if cub.Mode != 3 {
		t.Fatalf("expected CUB count 3, got %#v", cub)
	}
	cup := firstControlOp(t, damage, "cup")
	if cup.Row != 1 || cup.Col != 4 {
		t.Fatalf("expected CUP cursor position row=1 col=4, got %#v", cup)
	}
	semanticCUB := firstSemanticControlOp(t, damage, "cub")
	if semanticCUB.Mode != 3 {
		t.Fatalf("expected semantic CUB count 3, got %#v", semanticCUB)
	}
}

func TestVTermWriteWithDamageC1CSIControlsKeepSemanticOrder(t *testing.T) {
	vt := New(12, 3, 100, nil)

	raw := "abcdef" + string([]byte{0x9b}) + "3DXYZ" +
		string([]byte{0x9b}) + "2;5H!" +
		string([]byte{0x9b}) + "Ktail"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			switch op.Control {
			case "cub":
				if op.Mode != 3 {
					t.Fatalf("expected C1 CUB count 3, got %#v", op)
				}
			case "cup":
				if op.Row != 1 || op.Col != 4 {
					t.Fatalf("expected C1 CUP cursor position row=1 col=4, got %#v", op)
				}
			case "el":
				if op.Mode != 0 {
					t.Fatalf("expected C1 EL mode 0, got %#v", op)
				}
			case "ech":
				if op.Mode != 7 {
					t.Fatalf("expected C1 EL to expose erased cell count through ECH mode 7, got %#v", op)
				}
			}
		}
	}
	want := []string{"write:abcdef", "control:cub", "write:XYZ", "control:cup", "write:!", "control:ech", "control:el", "write:tail"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageC1CSIEraseLineModesKeepSemanticOrder(t *testing.T) {
	c1 := string([]byte{0x9b})

	for _, tc := range []struct {
		name      string
		mode      int
		raw       string
		wantOrder []string
		wantRow   string
	}{
		{
			name:      "erase to start",
			mode:      1,
			raw:       "abcdef" + c1 + "4G" + c1 + "1KZ",
			wantOrder: []string{"write:abcdef", "control:cha", "control:el", "write:Z"},
			wantRow:   "   Zef",
		},
		{
			name:      "erase whole line",
			mode:      2,
			raw:       "abcdef" + c1 + "4G" + c1 + "2KZ",
			wantOrder: []string{"write:abcdef", "control:cha", "control:el", "write:Z"},
			wantRow:   "   Z",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vt := New(12, 3, 100, nil)
			_, err, damage := vt.WriteWithDamage([]byte(tc.raw))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			el := firstSemanticControlOp(t, damage, "el")
			if el.Mode != tc.mode || el.Col != 3 {
				t.Fatalf("expected C1 EL mode=%d at col=3, got %#v damage=%#v", tc.mode, el, damage)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				switch op.Code {
				case ScreenOpWriteSpan:
					got = append(got, "write:"+damageOpText(op))
				case ScreenOpControl:
					got = append(got, "control:"+op.Control)
				}
			}
			if strings.Join(got, "|") != strings.Join(tc.wantOrder, "|") {
				t.Fatalf("C1 CSI EL mode %d semantic ops must preserve raw order, got %v want %v damage=%#v", tc.mode, got, tc.wantOrder, damage)
			}
			if row := strings.TrimRight(rowText(vt.ScreenRowView(0), 12), " "); row != tc.wantRow {
				t.Fatalf("C1 CSI EL mode %d should mutate line via vterm erase semantics, got %q want %q damage=%#v", tc.mode, row, tc.wantRow, damage)
			}
		})
	}
}

func TestVTermWriteWithDamageC1CSIEraseDisplayModesKeepSemanticOrder(t *testing.T) {
	c1 := string([]byte{0x9b})

	t.Run("erase below cursor", func(t *testing.T) {
		vt := New(12, 3, 100, nil)
		_, err, _ := vt.WriteWithDamage([]byte("top\r\nmiddle\r\nbottom"))
		if err != nil {
			t.Fatalf("seed write with damage: %v", err)
		}
		_, err, damage := vt.WriteWithDamage([]byte(c1 + "2;4H" + c1 + "Jtail"))
		if err != nil {
			t.Fatalf("write with damage: %v", err)
		}
		ed := firstSemanticControlOp(t, damage, "ed")
		if ed.Mode != 0 || ed.Row != 1 || ed.Col != 3 {
			t.Fatalf("expected C1 CSI ED0 at row=1 col=3, got %#v damage=%#v", ed, damage)
		}
		var got []string
		for _, op := range damage.SemanticOps {
			switch op.Code {
			case ScreenOpControl:
				got = append(got, "control:"+op.Control)
			case ScreenOpWriteSpan:
				got = append(got, "write:"+damageOpText(op))
			}
		}
		want := []string{"control:cup", "control:ed", "write:tail"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("C1 CSI ED0 semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
		}
	})

	t.Run("erase above cursor", func(t *testing.T) {
		vt := New(12, 3, 100, nil)
		_, err, _ := vt.WriteWithDamage([]byte("top\r\nmiddle\r\nbottom"))
		if err != nil {
			t.Fatalf("seed write with damage: %v", err)
		}
		_, err, damage := vt.WriteWithDamage([]byte(c1 + "2;4H" + c1 + "1Jtail"))
		if err != nil {
			t.Fatalf("write with damage: %v", err)
		}
		ed := firstSemanticControlOp(t, damage, "ed")
		if ed.Mode != 1 || ed.Row != 1 || ed.Col != 3 {
			t.Fatalf("expected C1 CSI ED1 at row=1 col=3, got %#v damage=%#v", ed, damage)
		}
		var got []string
		for _, op := range damage.SemanticOps {
			switch op.Code {
			case ScreenOpControl:
				got = append(got, "control:"+op.Control)
			case ScreenOpWriteSpan:
				got = append(got, "write:"+damageOpText(op))
			}
		}
		want := []string{"control:cup", "control:ed", "write:tail"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("C1 CSI ED1 semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
		}
	})

	t.Run("page break", func(t *testing.T) {
		vt := New(12, 3, 100, nil)
		_, err, _ := vt.WriteWithDamage([]byte("shell\r\npage"))
		if err != nil {
			t.Fatalf("seed write with damage: %v", err)
		}
		_, err, damage := vt.WriteWithDamage([]byte(c1 + "2Jframe"))
		if err != nil {
			t.Fatalf("write with damage: %v", err)
		}
		ed := firstSemanticControlOp(t, damage, "ed")
		if ed.Mode != 2 {
			t.Fatalf("expected C1 CSI ED2 control mode 2, got %#v damage=%#v", ed, damage)
		}
		var got []string
		for _, op := range damage.SemanticOps {
			switch op.Code {
			case ScreenOpControl:
				got = append(got, "control:"+op.Control)
			case ScreenOpWriteSpan:
				got = append(got, "write:"+damageOpText(op))
			}
		}
		want := []string{"control:ed", "write:frame"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("C1 CSI ED2 semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
		}
		if len(damage.ScrollbackAppend) == 0 {
			t.Fatalf("C1 CSI ED2 should keep live scrollback append as boundary signal, damage=%#v", damage)
		}
	})

	t.Run("clear scrollback", func(t *testing.T) {
		vt := New(12, 3, 100, nil)
		_, err, _ := vt.WriteWithDamage([]byte("old\r\nline"))
		if err != nil {
			t.Fatalf("seed write with damage: %v", err)
		}
		_, err, damage := vt.WriteWithDamage([]byte(c1 + "3Jafter"))
		if err != nil {
			t.Fatalf("write with damage: %v", err)
		}
		ed := firstSemanticControlOp(t, damage, "ed")
		if ed.Mode != 3 {
			t.Fatalf("expected C1 CSI ED3 control mode 3, got %#v damage=%#v", ed, damage)
		}
		var got []string
		for _, op := range damage.SemanticOps {
			switch op.Code {
			case ScreenOpControl:
				got = append(got, "control:"+op.Control)
			case ScreenOpWriteSpan:
				got = append(got, "write:"+damageOpText(op))
			}
		}
		want := []string{"control:ed", "write:after"}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("C1 CSI ED3 semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
		}
	})
}

func TestVTermWriteWithDamageC1CSIMovementAliasesKeepSemanticOrder(t *testing.T) {
	vt := New(16, 5, 100, nil)
	c1 := string([]byte{0x9b})

	raw := "row0\r\nrow1\r\nrow2\r\nrow3" +
		c1 + "2FUP" +
		c1 + "2ELOW" +
		c1 + "3aR" +
		c1 + "2eD" +
		c1 + "4`H" +
		c1 + "1dV" +
		c1 + "5;2fP"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	var cuu, cuf, vpa, cup bool
	cudCount := 0
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			switch op.Control {
			case "cuu":
				cuu = true
				if op.Mode != 2 || op.Row != 1 || op.Col != 4 {
					t.Fatalf("expected C1 CPL to expose CUU count=2 at row=1 col=4, got %#v", op)
				}
			case "cud":
				cudCount++
				if cudCount == 1 && (op.Mode != 2 || op.Row != 3 || op.Col != 2) {
					t.Fatalf("expected C1 CNL to expose CUD count=2 at row=3 col=2, got %#v", op)
				}
				if cudCount == 2 && (op.Mode != 2 || op.Row != 4 || op.Col != 7) {
					t.Fatalf("expected C1 VPR to expose CUD count=2 clamped to bottom, got %#v", op)
				}
			case "cuf":
				cuf = true
				if op.Mode != 3 || op.Col != 6 || op.Row != 3 {
					t.Fatalf("expected C1 HPR to expose CUF count=3 at row=3 col=6, got %#v", op)
				}
			case "vpa":
				vpa = true
				if op.Row != 0 || op.Col != 4 || op.Mode != 1 {
					t.Fatalf("expected C1 VPA to expose row=0 col=4 mode=1, got %#v", op)
				}
			case "cup":
				cup = true
				if op.Row != 4 || op.Col != 1 {
					t.Fatalf("expected C1 HVP to expose row=4 col=1, got %#v", op)
				}
			}
		}
	}
	want := []string{
		"write:row0", "control:cr", "control:lf",
		"write:row1", "control:cr", "control:lf",
		"write:row2", "control:cr", "control:lf",
		"write:row3", "control:cuu", "control:cr", "write:UP",
		"control:cud", "control:cr", "write:LOW",
		"control:cuf", "write:R",
		"control:cud", "write:D",
		"control:cha", "write:H",
		"control:vpa", "write:V",
		"control:cup", "write:P",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI movement aliases must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if !cuu || cudCount != 2 || !cuf || !vpa || !cup {
		t.Fatalf("expected C1 movement aliases to expose all normalized cursor controls, got cuu=%v cud=%d cuf=%v vpa=%v cup=%v damage=%#v", cuu, cudCount, cuf, vpa, cup, damage)
	}
}

func TestVTermWriteWithDamageSemanticOpsPreserveRawOrder(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ab\bX\tZ"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:ab", "control:bs", "write:X", "control:ht", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic ops must preserve raw write/control order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageDistinguishesSoftWrapFromIndex(t *testing.T) {
	vt := New(4, 2, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abcde"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if firstSemanticControlOp(t, damage, "soft-wrap").Control != "soft-wrap" {
		t.Fatalf("expected automatic wrap semantic control, got %#v", damage.SemanticOps)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl && op.Control == "ind" {
			t.Fatalf("automatic wrap must not be exposed as explicit index, got %#v", damage.SemanticOps)
		}
	}

	_, err, damage = vt.WriteWithDamage([]byte("\x1bD"))
	if err != nil {
		t.Fatalf("write explicit index: %v", err)
	}
	if firstSemanticControlOp(t, damage, "ind").Control != "ind" {
		t.Fatalf("expected explicit index semantic control, got %#v", damage.SemanticOps)
	}
}

func TestVTermWriteWithDamageSemanticTextComesFromPrintPath(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[31mred\x1b[0m"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	span := firstSemanticOpWithCode(t, damage, ScreenOpWriteSpan)
	if got := damageOpText(span); got != "red" {
		t.Fatalf("expected semantic print text, got %q op=%#v damage=%#v", got, span, damage)
	}
	for _, cell := range damageOpCells(span) {
		if cell.Style.FG != "ansi:1" {
			t.Fatalf("expected print semantic cells to keep style, got %#v op=%#v", cell, span)
		}
	}
}

func TestVTermWriteWithDamageC1CSISGRKeepsSemanticStyledText(t *testing.T) {
	vt := New(16, 3, 100, nil)

	raw := string([]byte{0x9b}) + "1;31mERR" + string([]byte{0x9b}) + "0m plain"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var spans []DamageOp
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpWriteSpan {
			spans = append(spans, op)
		}
	}
	if len(spans) != 2 {
		t.Fatalf("expected C1 CSI SGR text to produce styled/plain semantic spans, got %#v damage=%#v", spans, damage)
	}
	if damageOpText(spans[0]) != "ERR" {
		t.Fatalf("expected first C1 CSI SGR span text ERR, got %#v damage=%#v", spans[0], damage)
	}
	for _, cell := range damageOpCells(spans[0]) {
		if cell.Style.FG != "ansi:1" || !cell.Style.Bold {
			t.Fatalf("expected C1 CSI SGR styled cell from vterm print path, got %#v damage=%#v", cell, damage)
		}
	}
	if damageOpText(spans[1]) != " plain" {
		t.Fatalf("expected reset plain span, got %#v damage=%#v", spans[1], damage)
	}
	for _, cell := range damageOpCells(spans[1]) {
		if cell.Style != (CellStyle{}) {
			t.Fatalf("expected C1 CSI reset to clear style for following text, got %#v damage=%#v", cell, damage)
		}
	}
}

func TestVTermWriteWithDamageSemanticModesComeFromParserTransaction(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?2026htext\x1b[?2026l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	var enabled []bool
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, "mode")
			if op.Mode != 2026 || !op.Private {
				t.Fatalf("unexpected semantic mode op: %#v damage=%#v", op, damage)
			}
			enabled = append(enabled, op.Enabled)
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode", "write:text", "mode"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic mode ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if len(enabled) != 2 || !enabled[0] || enabled[1] {
		t.Fatalf("expected mode enable then disable, got enabled=%v ops=%#v", enabled, damage.SemanticOps)
	}
}

func TestVTermWriteWithDamageSemanticLegacyMouseModes(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?9h\x1b[?1001htext\x1b[?1001l\x1b[?9l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:9:on", "mode:1001:on", "write:text", "mode:1001:off", "mode:9:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic legacy mouse modes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticFocusAndSGRMouseModes(t *testing.T) {
	vt := New(24, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?1000h\x1b[?1002h\x1b[?1003h\x1b[?1004h\x1b[?1006hframe\x1b[?1006l\x1b[?1004l\x1b[?1003l\x1b[?1002l\x1b[?1000l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{
		"mode:1000:on", "mode:1002:on", "mode:1003:on", "mode:1004:on", "mode:1006:on",
		"write:frame",
		"mode:1006:off", "mode:1004:off", "mode:1003:off", "mode:1002:off", "mode:1000:off",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic focus/SGR mouse modes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageC1CSIPrivateModesKeepSemanticOrder(t *testing.T) {
	vt := New(24, 3, 100, nil)

	raw := string([]byte{0x9b}) + "?2026h" +
		string([]byte{0x9b}) + "?1004h" +
		"frame" +
		string([]byte{0x9b}) + "?1004l" +
		string([]byte{0x9b}) + "?2026l"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:2026:on", "mode:1004:on", "write:frame", "mode:1004:off", "mode:2026:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI private modes must preserve semantic raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageC1CSILinefeedNewlineModeKeepSemanticOrder(t *testing.T) {
	vt := New(12, 4, 100, nil)
	c1 := string([]byte{0x9b})

	raw := c1 + "20h" + "abc\nZ" + c1 + "20l"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if mode := firstSemanticModeOp(t, damage, 20); mode.Code != ScreenOpModes || mode.Private {
		t.Fatalf("expected C1 CSI LNM to expose ANSI mode 20 semantic op, got %#v damage=%#v", mode, damage)
	}
	cr := firstSemanticControlOp(t, damage, "cr")
	if cr.Col != 3 || cr.Row != 1 {
		t.Fatalf("expected LNM CR after LF to be recorded from next row col=3, got %#v damage=%#v", cr, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"mode:20:on", "write:abc", "control:lf", "control:cr", "write:Z", "mode:20:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI LNM semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if row0 := strings.TrimRight(rowText(vt.ScreenRowView(0), 12), " "); row0 != "abc" {
		t.Fatalf("expected C1 CSI LNM to keep first row text, got %q damage=%#v", row0, damage)
	}
	if row1 := strings.TrimRight(rowText(vt.ScreenRowView(1), 12), " "); row1 != "Z" {
		t.Fatalf("expected C1 CSI LNM to return following text to line start, got %q damage=%#v", row1, damage)
	}
}

func TestVTermWriteWithDamageSemanticCursorVisibilityMode(t *testing.T) {
	vt := New(20, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?25lframe\x1b[?25h"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:25:off", "write:frame", "mode:25:on"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic cursor visibility mode must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticCursorStyle(t *testing.T) {
	vt := New(20, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[5 qframe\x1b[2 q"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			if op.Control == "decscusr" {
				got = append(got, "control:"+op.Control+":"+strconv.Itoa(op.Mode))
			}
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:decscusr:5", "write:frame", "control:decscusr:2"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic cursor style must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if row := rowText(vt.ScreenRowView(0), len("frame")); row != "frame" {
		t.Fatalf("cursor style state must not render as text, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageSemanticReportRequests(t *testing.T) {
	responses := make(chan string, 4)
	vt := New(20, 3, 100, func(data []byte) {
		responses <- string(data)
	})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[6n\x1b[?6n\x1b[?25$pframe"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			switch op.Control {
			case "dsr", "decxcpr", "decrqm":
				got = append(got, "control:"+op.Control+":"+strconv.Itoa(op.Mode))
			}
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:dsr:6", "control:decxcpr:6", "control:decrqm:0", "write:frame"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic report requests must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if gotResponses := collectVTermResponses(responses, 3); len(gotResponses) < 3 {
		t.Fatalf("expected DSR/DECRQM responses from vterm, got %#v damage=%#v", gotResponses, damage)
	}
	if row := rowText(vt.ScreenRowView(0), len("frame")); row != "frame" {
		t.Fatalf("terminal report requests must not render as text, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageC1CSIReportRequestsKeepSemanticText(t *testing.T) {
	responses := make(chan string, 4)
	vt := New(20, 3, 100, func(data []byte) {
		responses <- string(data)
	})

	raw := string([]byte{0x9b}) + "6n" +
		string([]byte{0x9b}) + "?6n" +
		string([]byte{0x9b}) + "?25$pframe"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			switch op.Control {
			case "dsr", "decxcpr", "decrqm":
				got = append(got, "control:"+op.Control+":"+strconv.Itoa(op.Mode))
			}
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:dsr:6", "control:decxcpr:6", "control:decrqm:0", "write:frame"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI report requests must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if gotResponses := collectVTermResponses(responses, 3); len(gotResponses) < 3 {
		t.Fatalf("expected C1 DSR/DECRQM responses from vterm, got %#v damage=%#v", gotResponses, damage)
	}
	if row := rowText(vt.ScreenRowView(0), len("frame")); row != "frame" {
		t.Fatalf("C1 terminal report requests must not render as text, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageSemanticDeviceAttributes(t *testing.T) {
	responses := make(chan string, 4)
	vt := New(20, 3, 100, func(data []byte) {
		responses <- string(data)
	})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[c\x1b[>cframe"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			switch op.Control {
			case "da", "da2":
				got = append(got, "control:"+op.Control)
			}
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:da", "control:da2", "write:frame"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic device attributes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if gotResponses := collectVTermResponses(responses, 2); len(gotResponses) < 2 {
		t.Fatalf("expected DA1/DA2 responses from vterm, got %#v damage=%#v", gotResponses, damage)
	}
	if row := rowText(vt.ScreenRowView(0), len("frame")); row != "frame" {
		t.Fatalf("device attribute requests must not render as text, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageC1CSIDeviceAttributesKeepSemanticText(t *testing.T) {
	responses := make(chan string, 4)
	vt := New(20, 3, 100, func(data []byte) {
		responses <- string(data)
	})

	raw := string([]byte{0x9b}) + "c" + string([]byte{0x9b}) + ">cframe"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			switch op.Control {
			case "da", "da2":
				got = append(got, "control:"+op.Control)
			}
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:da", "control:da2", "write:frame"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI device attributes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if gotResponses := collectVTermResponses(responses, 2); len(gotResponses) < 2 {
		t.Fatalf("expected C1 DA1/DA2 responses from vterm, got %#v damage=%#v", gotResponses, damage)
	}
	if row := rowText(vt.ScreenRowView(0), len("frame")); row != "frame" {
		t.Fatalf("C1 device attribute requests must not render as text, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageOSCDefaultColorsKeepSemanticText(t *testing.T) {
	responses := make(chan string, 4)
	vt := New(20, 3, 100, func(data []byte) {
		responses <- string(data)
	})

	raw := "\x1b]10;#112233\x07\x1b]11;#445566\x07\x1b]12;#778899\x07" +
		"\x1b]10;?\x07\x1b]11;?\x07\x1b]12;?\x07" +
		"palette" +
		"\x1b]110\x07\x1b]111\x07\x1b]112\x07"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !semanticOpsContainText(damage.SemanticOps, "palette") {
		t.Fatalf("OSC default color batch should keep following text as semantic text, ops=%#v damage=%#v", damage.SemanticOps, damage)
	}
	if gotResponses := collectVTermResponses(responses, 3); len(gotResponses) < 3 {
		t.Fatalf("expected OSC color query responses from vterm, got %#v damage=%#v", gotResponses, damage)
	} else {
		joined := strings.Join(gotResponses, "")
		for _, want := range []string{"\x1b]10;rgb:", "\x1b]11;rgb:", "\x1b]12;rgb:"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("expected color query response %q, got %#v damage=%#v", want, gotResponses, damage)
			}
		}
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl || op.Code == ScreenOpTitle {
			t.Fatalf("OSC default colors are vterm-owned state, not history semantic ops, got %#v in %#v", op, damage.SemanticOps)
		}
	}
	if row := rowText(vt.ScreenRowView(0), len("palette")); row != "palette" {
		t.Fatalf("OSC default colors must not render as text, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageStringControlsKeepSemanticText(t *testing.T) {
	vt := New(30, 3, 100, nil)

	raw := "\x1bP1;2+qignored-dcs\x1b\\" +
		"\x1b_ignored-apc\x1b\\" +
		"\x1bXignored-sos\x1b\\" +
		"\x1b^ignored-pm\x1b\\" +
		"after"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !semanticOpsContainText(damage.SemanticOps, "after") {
		t.Fatalf("string controls should keep following text as semantic text, ops=%#v damage=%#v", damage.SemanticOps, damage)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl || op.Code == ScreenOpTitle {
			t.Fatalf("string controls are vterm parser-owned state, not history semantic ops, got %#v in %#v", op, damage.SemanticOps)
		}
	}
	text := rowText(vt.ScreenRowView(0), 30)
	for _, forbidden := range []string{"ignored-dcs", "ignored-apc", "ignored-sos", "ignored-pm"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("string control payload must not render as text %q, got %q damage=%#v", forbidden, text, damage)
		}
	}
	if !strings.Contains(text, "after") {
		t.Fatalf("text after string controls must remain visible, got %q damage=%#v", text, damage)
	}
}

func TestVTermWriteWithDamageOSCClipboardKeepSemanticText(t *testing.T) {
	vt := New(30, 3, 100, nil)

	raw := "\x1b]52;c;SGVsbG8=\x07" +
		"\x1b]52;p;?\x07" +
		"\x1b]52;c;\x07" +
		"clip-ok"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !semanticOpsContainText(damage.SemanticOps, "clip-ok") {
		t.Fatalf("OSC clipboard should keep following text as semantic text, ops=%#v damage=%#v", damage.SemanticOps, damage)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl || op.Code == ScreenOpTitle {
			t.Fatalf("OSC clipboard is vterm parser-owned state, not history semantic ops, got %#v in %#v", op, damage.SemanticOps)
		}
	}
	text := rowText(vt.ScreenRowView(0), 30)
	for _, forbidden := range []string{"52;", "SGVsbG8", "?"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("OSC clipboard payload must not render as text %q, got %q damage=%#v", forbidden, text, damage)
		}
	}
	if !strings.Contains(text, "clip-ok") {
		t.Fatalf("text after OSC clipboard must remain visible, got %q damage=%#v", text, damage)
	}
}

func TestVTermWriteWithDamageC1StringControlsKeepSemanticText(t *testing.T) {
	vt := New(40, 3, 100, nil)

	raw := string([]byte{0x9d}) + "52;c;SGVsbG8=" + string([]byte{0x9c}) +
		string([]byte{0x90}) + "1;2+qignored-c1-dcs" + string([]byte{0x9c}) +
		string([]byte{0x9f}) + "ignored-c1-apc" + string([]byte{0x9c}) +
		string([]byte{0x98}) + "ignored-c1-sos" + string([]byte{0x9c}) +
		string([]byte{0x9e}) + "ignored-c1-pm" + string([]byte{0x9c}) +
		"c1-ok"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !semanticOpsContainText(damage.SemanticOps, "c1-ok") {
		t.Fatalf("C1 string controls should keep following text as semantic text, ops=%#v damage=%#v", damage.SemanticOps, damage)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl || op.Code == ScreenOpTitle {
			t.Fatalf("C1 string controls are vterm parser-owned state, not history semantic ops, got %#v in %#v", op, damage.SemanticOps)
		}
	}
	text := rowText(vt.ScreenRowView(0), 40)
	for _, forbidden := range []string{"ignored-c1-dcs", "ignored-c1-apc", "ignored-c1-sos", "ignored-c1-pm", "SGVsbG8"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("C1 string control payload must not render as text %q, got %q damage=%#v", forbidden, text, damage)
		}
	}
	if !strings.Contains(text, "c1-ok") {
		t.Fatalf("text after C1 string controls must remain visible, got %q damage=%#v", text, damage)
	}
}

func collectVTermResponses(ch <-chan string, want int) []string {
	responses := make([]string, 0, want)
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	for len(responses) < want {
		select {
		case response := <-ch:
			responses = append(responses, response)
		case <-timer.C:
			return responses
		}
	}
	return responses
}

func TestVTermWriteWithDamageSemanticApplicationKeyModes(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?1h\x1b[?66hkeys\x1b[?66l\x1b[?1l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:1:on", "mode:66:on", "write:keys", "mode:66:off", "mode:1:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic application key modes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticKeypadEscModes(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b=esc-keys\x1b>"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:66:on", "write:esc-keys", "mode:66:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic keypad ESC modes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticAlternateScrollMode(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?1007hscroll\x1b[?1007l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:1007:on", "write:scroll", "mode:1007:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic alternate scroll mode must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticUTF8MouseMode(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?1005hutf8-mouse\x1b[?1005l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:1005:on", "write:utf8-mouse", "mode:1005:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic UTF-8 mouse mode must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticExtendedMouseEncodingModes(t *testing.T) {
	vt := New(20, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?1015h\x1b[?1016hext-mouse\x1b[?1016l\x1b[?1015l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:1015:on", "mode:1016:on", "write:ext-mouse", "mode:1016:off", "mode:1015:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic extended mouse encoding modes must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticUnicodeCoreMode(t *testing.T) {
	vt := New(20, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?2027he\u0301好\x1b[?2027l"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:2027:on", "write:e", "write:́", "write:好", "mode:2027:off"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic Unicode Core mode must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func semanticCellsContent(cells []Cell) string {
	var b strings.Builder
	for _, cell := range cells {
		if cell.Content == "" {
			continue
		}
		b.WriteString(cell.Content)
	}
	return b.String()
}

func TestVTermWriteWithDamageSemanticBackwardTab(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("123456789\x1b[ZXY"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			if op.Control == "cbt" && op.Col != 8 {
				t.Fatalf("expected CBT to land on previous tab stop col 8, got %#v", op)
			}
		}
	}
	want := []string{"write:123456789", "control:cbt", "write:XY"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic CBT must preserve raw cursor order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticForwardTab(t *testing.T) {
	vt := New(24, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ab\x1b[2IZ"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			if op.Control == "ht" && op.Col != 16 {
				t.Fatalf("expected CHT to land on second forward tab stop col 16, got %#v", op)
			}
		}
	}
	want := []string{"write:ab", "control:ht", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic CHT must preserve raw cursor order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticCustomTabStop(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ab\x1bH\r\tZ"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	hts := firstSemanticControlOp(t, damage, "hts")
	if hts.Col != 2 {
		t.Fatalf("expected HTS to record custom tab stop col 2, got %#v damage=%#v", hts, damage)
	}
	ht := firstSemanticControlOp(t, damage, "ht")
	if ht.Col != 2 {
		t.Fatalf("expected HT to land on custom tab stop col 2, got %#v damage=%#v", ht, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:ab", "control:hts", "control:cr", "control:ht", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic custom tab stop ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageC1ControlsKeepSemanticOrder(t *testing.T) {
	vt := New(16, 4, 100, nil)

	raw := "ab" + string([]byte{0x88}) +
		string([]byte{0x84}) + "down" +
		string([]byte{0x8d}) + "\r\tZ"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	hts := firstSemanticControlOp(t, damage, "hts")
	if hts.Col != 2 {
		t.Fatalf("expected C1 HTS to record custom tab stop col 2, got %#v damage=%#v", hts, damage)
	}
	ind := firstSemanticControlOp(t, damage, "ind")
	if ind.Row != 0 || ind.Col != 2 {
		t.Fatalf("expected C1 IND at row 0 col 2, got %#v damage=%#v", ind, damage)
	}
	ri := firstSemanticControlOp(t, damage, "ri")
	if ri.Row != 1 || ri.Col != 6 {
		t.Fatalf("expected C1 RI at row 1 col 6, got %#v damage=%#v", ri, damage)
	}
	ht := firstSemanticControlOp(t, damage, "ht")
	if ht.Col != 2 {
		t.Fatalf("expected HT after C1 HTS to land on custom tab stop col 2, got %#v damage=%#v", ht, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:ab", "control:hts", "control:ind", "write:down", "control:ri", "control:cr", "control:ht", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 controls semantic ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageC1NELKeepsSemanticOrder(t *testing.T) {
	vt := New(16, 3, 100, nil)

	raw := "abc" + string([]byte{0x85}) + "Z"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	cr := firstSemanticControlOp(t, damage, "cr")
	if cr.Col != 3 || cr.Row != 0 {
		t.Fatalf("expected C1 NEL to record CR from row=0 col=3, got %#v damage=%#v", cr, damage)
	}
	lf := firstSemanticControlOp(t, damage, "lf")
	if lf.Row != 0 {
		t.Fatalf("expected C1 NEL to record LF from row=0 after CR, got %#v damage=%#v", lf, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:abc", "control:cr", "control:lf", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 NEL semantic ops must preserve CR/LF raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if row0 := strings.TrimRight(rowText(vt.ScreenRowView(0), 16), " "); row0 != "abc" {
		t.Fatalf("expected C1 NEL to keep first row text, got %q damage=%#v", row0, damage)
	}
	if row1 := strings.TrimRight(rowText(vt.ScreenRowView(1), 16), " "); row1 != "Z" {
		t.Fatalf("expected C1 NEL to move next text to row start, got %q damage=%#v", row1, damage)
	}
}

func TestVTermWriteWithDamageESCNELKeepsSemanticOrder(t *testing.T) {
	vt := New(16, 3, 100, nil)

	raw := "abc\x1bEZ"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	cr := firstSemanticControlOp(t, damage, "cr")
	if cr.Col != 3 || cr.Row != 0 {
		t.Fatalf("expected ESC NEL to record CR from row=0 col=3, got %#v damage=%#v", cr, damage)
	}
	lf := firstSemanticControlOp(t, damage, "lf")
	if lf.Row != 0 {
		t.Fatalf("expected ESC NEL to record LF from row=0 after CR, got %#v damage=%#v", lf, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:abc", "control:cr", "control:lf", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ESC NEL semantic ops must preserve CR/LF raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if row0 := strings.TrimRight(rowText(vt.ScreenRowView(0), 16), " "); row0 != "abc" {
		t.Fatalf("expected ESC NEL to keep first row text, got %q damage=%#v", row0, damage)
	}
	if row1 := strings.TrimRight(rowText(vt.ScreenRowView(1), 16), " "); row1 != "Z" {
		t.Fatalf("expected ESC NEL to move next text to row start, got %q damage=%#v", row1, damage)
	}
}

func TestVTermWriteWithDamageSemanticTabClear(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ab\r\x1b[3g\tZ"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	tbc := firstSemanticControlOp(t, damage, "tbc")
	if tbc.Mode != 3 {
		t.Fatalf("expected TBC mode 3 to be recorded, got %#v damage=%#v", tbc, damage)
	}
	ht := firstSemanticControlOp(t, damage, "ht")
	if ht.Col != 15 {
		t.Fatalf("expected HT to land on final column after clearing tabs, got %#v damage=%#v", ht, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:ab", "control:cr", "control:tbc", "control:ht", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic tab clear ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticTabReset(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ab\x1bH\x1b[?5W\r\tZ"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	decst8c := firstSemanticControlOp(t, damage, "decst8c")
	if decst8c.Mode != 5 {
		t.Fatalf("expected DECST8C mode 5 to be recorded, got %#v damage=%#v", decst8c, damage)
	}
	ht := firstSemanticControlOp(t, damage, "ht")
	if ht.Col != 8 {
		t.Fatalf("expected HT to land on default tab stop col 8 after DECST8C, got %#v damage=%#v", ht, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:ab", "control:hts", "control:decst8c", "control:cr", "control:ht", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic tab reset ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageC1CSITabControlsKeepSemanticOrder(t *testing.T) {
	c1CSI := string([]byte{0x9b})
	tests := []struct {
		name   string
		cols   int
		raw    string
		want   []string
		checks map[string]int
		modes  map[string]int
	}{
		{
			name:   "backward-tab",
			cols:   16,
			raw:    "123456789" + c1CSI + "ZXY",
			want:   []string{"write:123456789", "control:cbt", "write:XY"},
			checks: map[string]int{"cbt": 8},
		},
		{
			name:   "forward-tab",
			cols:   24,
			raw:    "ab" + c1CSI + "2IZ",
			want:   []string{"write:ab", "control:ht", "write:Z"},
			checks: map[string]int{"ht": 16},
		},
		{
			name:   "tab-clear",
			cols:   16,
			raw:    "ab\r" + c1CSI + "3g\tZ",
			want:   []string{"write:ab", "control:cr", "control:tbc", "control:ht", "write:Z"},
			checks: map[string]int{"ht": 15},
			modes:  map[string]int{"tbc": 3},
		},
		{
			name:   "tab-reset",
			cols:   16,
			raw:    "ab\x1bH" + c1CSI + "?5W\r\tZ",
			want:   []string{"write:ab", "control:hts", "control:decst8c", "control:cr", "control:ht", "write:Z"},
			checks: map[string]int{"ht": 8},
			modes:  map[string]int{"decst8c": 5},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vt := New(tc.cols, 3, 100, nil)
			_, err, damage := vt.WriteWithDamage([]byte(tc.raw))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				switch op.Code {
				case ScreenOpWriteSpan:
					got = append(got, "write:"+damageOpText(op))
				case ScreenOpControl:
					got = append(got, "control:"+op.Control)
				}
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("C1 CSI tab semantic ops must preserve raw order, got %v want %v damage=%#v", got, tc.want, damage)
			}
			for control, wantCol := range tc.checks {
				op := firstSemanticControlOp(t, damage, control)
				if op.Col != wantCol {
					t.Fatalf("%s should expose vterm-resolved column %d, got %#v damage=%#v", control, wantCol, op, damage)
				}
			}
			for control, wantMode := range tc.modes {
				op := firstSemanticControlOp(t, damage, control)
				if op.Mode != wantMode {
					t.Fatalf("%s should expose mode %d, got %#v damage=%#v", control, wantMode, op, damage)
				}
			}
		})
	}
}

func TestVTermWriteWithDamageSemanticSpecialDrawingCharset(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b(0lqk\x1b(Bok"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"control:scs", "┌", "─", "┐", "control:scs", "ok"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic SCS ops must preserve charset state order and mapped text, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticLockingShiftCharset(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b)0\x0eq\x0fq"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpWriteSpan {
			got = append(got, damageOpText(op))
		}
	}
	want := []string{"─", "q"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic locking shift text must come from vterm GL charset state, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticG2G3LockingShiftCharset(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "ls2", raw: "\x1b*0\x1bnq\x0fq"},
		{name: "ls3", raw: "\x1b+0\x1boq\x0fq"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := New(16, 3, 100, nil)

			_, err, damage := vt.WriteWithDamage([]byte(tt.raw))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				if op.Code == ScreenOpWriteSpan {
					got = append(got, damageOpText(op))
				}
			}
			want := []string{"─", "q"}
			if strings.Join(got, "|") != strings.Join(want, "|") {
				t.Fatalf("semantic G2/G3 locking shift text must come from vterm GL charset state, got %v want %v damage=%#v", got, want, damage)
			}
		})
	}
}

func TestVTermWriteWithDamageSemanticSingleShiftCharset(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b*0\x1bNqq"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpWriteSpan {
			got = append(got, damageOpText(op))
		}
	}
	want := []string{"─", "q"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic single shift text must map only the next byte through vterm G2 charset state, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticResetInitialState(t *testing.T) {
	vt := New(16, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abc\x1bcZ"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !damage.RequiresFullReplace {
		t.Fatalf("RIS resets the terminal screen and should require live full replace, damage=%#v", damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		}
	}
	want := []string{"write:abc", "control:ris", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic RIS ops must preserve raw reset order, got %v want %v damage=%#v", got, want, damage)
	}
	screen := vt.ScreenContent()
	if got := strings.TrimRight(rowText(screen.Cells[0], 16), " "); got != "Z" {
		t.Fatalf("expected RIS to clear previous screen content before Z, got %q screen=%#v", got, screen.Cells[0])
	}
}

func TestVTermWriteWithDamageSemanticClearComesFromControl(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abcdef\x1b[4D\x1b[K"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if firstOpWithCode(t, damage, ScreenOpClearToEOL).Code != ScreenOpClearToEOL {
		t.Fatalf("expected screen diff clear op, got %#v", damage.Ops)
	}
	semanticEL := firstSemanticControlOp(t, damage, "el")
	if semanticEL.Mode != 0 {
		t.Fatalf("expected semantic EL mode 0, got %#v", semanticEL)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpClearToEOL || op.Code == ScreenOpClearRect {
			t.Fatalf("screen diff clear must not be semantic op, got %#v in %#v", op, damage.SemanticOps)
		}
	}
}

func TestVTermWriteWithDamageSemanticEraseCharacter(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ABCDE\x1b[2G\x1b[2X"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			if op.Control == "ech" && (op.Col != 1 || op.Mode != 2) {
				t.Fatalf("expected ECH at col 1 count 2, got %#v", op)
			}
		}
	}
	want := []string{"write:ABCDE", "control:cha", "control:ech"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic ECH must preserve raw clear order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticDeleteCharacter(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ABCDE\x1b[2G\x1b[2P"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			if op.Control == "dch" && (op.Col != 1 || op.Mode != 2) {
				t.Fatalf("expected DCH at col 1 count 2, got %#v", op)
			}
		}
	}
	want := []string{"write:ABCDE", "control:cha", "control:dch"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic DCH must preserve raw delete order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticInsertCharacter(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("ABCDE\x1b[2G\x1b[2@"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
			if op.Control == "ich" && (op.Col != 1 || op.Mode != 2) {
				t.Fatalf("expected ICH at col 1 count 2, got %#v", op)
			}
		}
	}
	want := []string{"write:ABCDE", "control:cha", "control:ich"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic ICH must preserve raw insert order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticRepeatCharacter(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("AB\x1b[3bC"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		if op.Code != ScreenOpWriteSpan {
			continue
		}
		got = append(got, damageOpText(op))
	}
	if strings.Join(got, "|") != "AB|B|B|B|C" {
		t.Fatalf("REP should emit repeated text semantic ops from print path, got %v damage=%#v", got, damage)
	}
}

func TestVTermWriteWithDamageC1CSIInlineEditKeepSemanticOrder(t *testing.T) {
	c1 := string([]byte{0x9b})
	for _, tc := range []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "erase-character",
			raw:  "ABCDE" + c1 + "2G" + c1 + "2X",
			want: []string{"write:ABCDE", "control:cha", "control:ech"},
		},
		{
			name: "delete-character",
			raw:  "ABCDE" + c1 + "2G" + c1 + "2P",
			want: []string{"write:ABCDE", "control:cha", "control:dch"},
		},
		{
			name: "insert-character",
			raw:  "ABCDE" + c1 + "2G" + c1 + "2@",
			want: []string{"write:ABCDE", "control:cha", "control:ich"},
		},
		{
			name: "repeat-character",
			raw:  "AB" + c1 + "3bC",
			want: []string{"write:AB", "write:B", "write:B", "write:B", "write:C"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vt := New(12, 3, 100, nil)

			_, err, damage := vt.WriteWithDamage([]byte(tc.raw))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				switch op.Code {
				case ScreenOpWriteSpan:
					got = append(got, "write:"+damageOpText(op))
				case ScreenOpControl:
					got = append(got, "control:"+op.Control)
					if (op.Control == "ech" || op.Control == "dch" || op.Control == "ich") && (op.Col != 1 || op.Mode != 2) {
						t.Fatalf("expected C1 %s at col 1 count 2, got %#v", op.Control, op)
					}
				}
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("C1 CSI inline edit semantic order mismatch, got %v want %v damage=%#v", got, tc.want, damage)
			}
		})
	}
}

func TestVTermWriteWithDamageSemanticSaveRestoreCursor(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abc\x1b7\x1b[2;5HZZ\x1b8X"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		if op.Code != ScreenOpWriteSpan {
			continue
		}
		got = append(got, damageOpText(op))
		if damageOpText(op) == "X" && (op.Row != 0 || op.Col != 3) {
			t.Fatalf("restore cursor should write X at saved row=0 col=3, got %#v damage=%#v", op, damage)
		}
	}
	if strings.Join(got, "|") != "abc|ZZ|X" {
		t.Fatalf("save/restore cursor should preserve ordered write semantics, got %v damage=%#v", got, damage)
	}
}

func TestVTermWriteWithDamageSemanticLineOperations(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     string
		control string
		row     int
		col     int
		count   int
		dy      int
		want    []string
	}{
		{
			name:    "insert-line",
			raw:     "top\r\nmiddle\r\nbottom\x1b[2;1H\x1b[1Lafter",
			control: "il",
			row:     1,
			col:     0,
			count:   1,
			dy:      1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:cup", "control:il", "write:after"},
		},
		{
			name:    "delete-line",
			raw:     "top\r\nmiddle\r\nbottom\x1b[2;1H\x1b[1Mafter",
			control: "dl",
			row:     1,
			col:     0,
			count:   1,
			dy:      -1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:cup", "control:dl", "write:after"},
		},
		{
			name:    "scroll-up",
			raw:     "top\r\nmiddle\r\nbottom\x1b[1Safter",
			control: "su",
			row:     0,
			col:     6,
			count:   1,
			dy:      -1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:su", "write:after"},
		},
		{
			name:    "scroll-down",
			raw:     "top\r\nmiddle\r\nbottom\x1b[1Tafter",
			control: "sd",
			row:     0,
			col:     6,
			count:   1,
			dy:      1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:sd", "write:after"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vt := New(12, 3, 100, nil)

			_, err, damage := vt.WriteWithDamage([]byte(tc.raw))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			op := firstSemanticControlOp(t, damage, tc.control)
			if op.Row != tc.row || op.Col != tc.col || op.Mode != tc.count {
				t.Fatalf("unexpected %s control op: %#v damage=%#v", tc.control, op, damage)
			}
			if op.Bottom != 3 {
				t.Fatalf("%s should carry scroll-region bottom, got %#v", tc.control, op)
			}
			if firstOpWithCode(t, damage, ScreenOpScrollRect).Dy != tc.dy {
				t.Fatalf("expected accompanying vertical scroll diff dy=%d, got ops=%#v", tc.dy, damage.Ops)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				switch op.Code {
				case ScreenOpWriteSpan:
					got = append(got, "write:"+damageOpText(op))
				case ScreenOpControl:
					got = append(got, "control:"+op.Control)
				}
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("semantic line operation order mismatch, got %v want %v damage=%#v", got, tc.want, damage)
			}
		})
	}
}

func TestVTermWriteWithDamageC1CSILineOperationsKeepSemanticOrder(t *testing.T) {
	c1 := string([]byte{0x9b})
	for _, tc := range []struct {
		name    string
		raw     string
		control string
		row     int
		col     int
		count   int
		dy      int
		want    []string
	}{
		{
			name:    "insert-line",
			raw:     "top\r\nmiddle\r\nbottom" + c1 + "2;1H" + c1 + "1Lafter",
			control: "il",
			row:     1,
			col:     0,
			count:   1,
			dy:      1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:cup", "control:il", "write:after"},
		},
		{
			name:    "delete-line",
			raw:     "top\r\nmiddle\r\nbottom" + c1 + "2;1H" + c1 + "1Mafter",
			control: "dl",
			row:     1,
			col:     0,
			count:   1,
			dy:      -1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:cup", "control:dl", "write:after"},
		},
		{
			name:    "scroll-up",
			raw:     "top\r\nmiddle\r\nbottom" + c1 + "1Safter",
			control: "su",
			row:     0,
			col:     6,
			count:   1,
			dy:      -1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:su", "write:after"},
		},
		{
			name:    "scroll-down",
			raw:     "top\r\nmiddle\r\nbottom" + c1 + "1Tafter",
			control: "sd",
			row:     0,
			col:     6,
			count:   1,
			dy:      1,
			want:    []string{"write:top", "control:cr", "control:lf", "write:middle", "control:cr", "control:lf", "write:bottom", "control:sd", "write:after"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vt := New(12, 3, 100, nil)

			_, err, damage := vt.WriteWithDamage([]byte(tc.raw))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			op := firstSemanticControlOp(t, damage, tc.control)
			if op.Row != tc.row || op.Col != tc.col || op.Mode != tc.count {
				t.Fatalf("unexpected C1 %s control op: %#v damage=%#v", tc.control, op, damage)
			}
			if op.Bottom != 3 {
				t.Fatalf("C1 %s should carry scroll-region bottom, got %#v", tc.control, op)
			}
			if firstOpWithCode(t, damage, ScreenOpScrollRect).Dy != tc.dy {
				t.Fatalf("expected C1 accompanying vertical scroll diff dy=%d, got ops=%#v", tc.dy, damage.Ops)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				switch op.Code {
				case ScreenOpWriteSpan:
					got = append(got, "write:"+damageOpText(op))
				case ScreenOpControl:
					got = append(got, "control:"+op.Control)
				}
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("C1 CSI semantic line operation order mismatch, got %v want %v damage=%#v", got, tc.want, damage)
			}
		})
	}
}

func TestVTermWriteWithDamageSemanticEraseDisplayComesFromControl(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abcdef\x1b[H\x1b[Jframe"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	semanticED := firstSemanticControlOp(t, damage, "ed")
	if semanticED.Mode != 0 {
		t.Fatalf("expected semantic ED mode 0, got %#v", semanticED)
	}
	if !semanticOpsContainText(damage.SemanticOps, "frame") {
		t.Fatalf("expected frame text in semantic ops, got %#v", damage.SemanticOps)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpClearToEOL || op.Code == ScreenOpClearRect {
			t.Fatalf("screen diff clear must not be semantic op, got %#v in %#v", op, damage.SemanticOps)
		}
	}
}

func TestVTermWriteWithDamageSemanticScrollRegionAndRI(t *testing.T) {
	vt := New(16, 6, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[2;5r\x1b[2;1H\x1bMregion"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	region := firstSemanticControlOp(t, damage, "decstbm")
	if region.Mode != 2 || region.Bottom != 5 {
		t.Fatalf("expected scroll region top=2 bottom=5, got %#v", region)
	}
	ri := firstSemanticControlOp(t, damage, "ri")
	if ri.Row != 1 || ri.Col != 0 {
		t.Fatalf("expected RI at scroll-region top, got %#v", ri)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:decstbm", "control:cup", "control:ri", "write:region"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic ops must preserve scroll-region/RI order, got %v want %v damage=%#v", got, want, damage)
	}
	if firstOpWithCode(t, damage, ScreenOpScrollRect).Dy <= 0 {
		t.Fatalf("expected RI to produce down-scroll screen op, got %#v", damage.Ops)
	}
}

func TestVTermWriteWithDamageC1CSIScrollRegionAndRIKeepSemanticOrder(t *testing.T) {
	vt := New(16, 6, 100, nil)
	c1 := string([]byte{0x9b})

	raw := c1 + "2;5r" + c1 + "2;1H\x8dregion"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	region := firstSemanticControlOp(t, damage, "decstbm")
	if region.Mode != 2 || region.Bottom != 5 {
		t.Fatalf("expected C1 scroll region top=2 bottom=5, got %#v", region)
	}
	ri := firstSemanticControlOp(t, damage, "ri")
	if ri.Row != 1 || ri.Col != 0 {
		t.Fatalf("expected C1 RI at scroll-region top, got %#v", ri)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:decstbm", "control:cup", "control:ri", "write:region"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI scroll-region/RI ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
	if firstOpWithCode(t, damage, ScreenOpScrollRect).Dy <= 0 {
		t.Fatalf("expected C1 RI to produce down-scroll screen op, got %#v", damage.Ops)
	}
}

func TestVTermWriteWithDamageSemanticOriginModeCursorPosition(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cursor string
	}{
		{name: "cup", cursor: "\x1b[1;1H"},
		{name: "hvp", cursor: "\x1b[1;1f"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vt := New(16, 6, 100, nil)

			_, err, damage := vt.WriteWithDamage([]byte("\x1b[2;4r\x1b[?6h" + tc.cursor + "X"))
			if err != nil {
				t.Fatalf("write with damage: %v", err)
			}
			if firstSemanticModeOp(t, damage, 6).Code != ScreenOpModes {
				t.Fatalf("expected origin mode semantic op, got %#v", damage.SemanticOps)
			}
			cup := firstSemanticControlOp(t, damage, "cup")
			if cup.Row != 1 || cup.Col != 0 {
				t.Fatalf("expected CUP to be resolved relative to scroll-region origin, got %#v damage=%#v", cup, damage)
			}
			var got []string
			for _, op := range damage.SemanticOps {
				switch op.Code {
				case ScreenOpControl:
					got = append(got, "control:"+op.Control)
				case ScreenOpModes:
					got = append(got, modeOpLabel(op))
				case ScreenOpWriteSpan:
					got = append(got, "write:"+damageOpText(op))
				}
			}
			want := []string{"control:decstbm", "mode:6:on", "control:cup", "write:X"}
			if strings.Join(got, "|") != strings.Join(want, "|") {
				t.Fatalf("semantic origin-mode ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
			}
		})
	}
}

func TestVTermWriteWithDamageC1CSIOriginModeCursorPosition(t *testing.T) {
	vt := New(16, 6, 100, nil)
	c1 := string([]byte{0x9b})

	raw := c1 + "2;4r" + c1 + "?6h" + c1 + "1;1HX"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if firstSemanticModeOp(t, damage, 6).Code != ScreenOpModes {
		t.Fatalf("expected C1 origin mode semantic op, got %#v", damage.SemanticOps)
	}
	cup := firstSemanticControlOp(t, damage, "cup")
	if cup.Row != 1 || cup.Col != 0 {
		t.Fatalf("expected C1 CUP to be resolved relative to scroll-region origin, got %#v damage=%#v", cup, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:decstbm", "mode:6:on", "control:cup", "write:X"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI origin-mode ops must preserve raw order, got %v want %v damage=%#v", got, want, damage)
	}
}

func TestVTermWriteWithDamageSemanticAutowrapMode(t *testing.T) {
	vt := New(6, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?7l123456789Z"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	mode := firstSemanticModeOp(t, damage, 7)
	if mode.Code != ScreenOpModes || !mode.Private || mode.Enabled {
		t.Fatalf("expected autowrap disable semantic op, got %#v in %#v", mode, damage.SemanticOps)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl && op.Control == "soft-wrap" {
			t.Fatalf("autowrap disabled batch must not emit soft-wrap semantic control, damage=%#v", damage)
		}
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:7:off", "write:123456", "write:7", "write:8", "write:9", "write:Z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic autowrap ops must use vterm-resolved print path, got %v want %v damage=%#v", got, want, damage)
	}
	if text := rowText(vt.ScreenRowView(0), 6); text != "12345Z" {
		t.Fatalf("autowrap disabled should overwrite final column without wrapping, got %q damage=%#v", text, damage)
	}
}

func TestVTermWriteWithDamageSemanticPrivateSaveRestoreCursor(t *testing.T) {
	vt := New(10, 4, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("abc\x1b[?1048h\x1b[2;1HZZ\x1b[?1048lX"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if firstSemanticModeOp(t, damage, 1048).Code != ScreenOpModes {
		t.Fatalf("expected private save cursor mode semantic op, got %#v", damage.SemanticOps)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			got = append(got, modeOpLabel(op))
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"write:abc", "mode:1048:on", "control:cup", "write:ZZ", "mode:1048:off", "write:X"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic private save/restore ops must preserve vterm cursor state order, got %v want %v damage=%#v", got, want, damage)
	}
	if row := strings.TrimSpace(rowText(vt.ScreenRowView(0), 10)); row != "abcX" {
		t.Fatalf("restore should place X after saved cursor on first row, got %q damage=%#v", row, damage)
	}
	if row := strings.TrimSpace(rowText(vt.ScreenRowView(1), 10)); row != "ZZ" {
		t.Fatalf("intermediate cursor move/write should stay on second row, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageSemanticLeftRightMargins(t *testing.T) {
	vt := New(10, 1, 100, nil)

	raw := "\x1b[?W\x1b[?6h\x1b[?69h\x1b[3;6s\x1b[1;2HX\x1b[ZA"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if firstSemanticModeOp(t, damage, 69).Code != ScreenOpModes {
		t.Fatalf("expected left/right margin mode semantic op, got %#v", damage.SemanticOps)
	}
	region := firstSemanticControlOp(t, damage, "decslrm")
	if region.Mode != 3 || region.Bottom != 6 {
		t.Fatalf("expected DECSLRM left=3 right=6, got %#v damage=%#v", region, damage)
	}
	cbt := firstSemanticControlOp(t, damage, "cbt")
	if cbt.Col != 2 {
		t.Fatalf("CBT should clamp to left margin final col 2, got %#v damage=%#v", cbt, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			if op.Mode == 69 {
				got = append(got, modeOpLabel(op))
			}
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"mode:69:on", "control:decslrm", "control:cup", "write:X", "control:cbt", "write:A"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("semantic left/right margin ops must preserve vterm-owned margin state order, got %v want %v damage=%#v", got, want, damage)
	}
	if row := rowText(vt.ScreenRowView(0), 10); row != "  AX      " {
		t.Fatalf("left/right margin CBT should place A at left margin before X, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageC1CSILeftRightMarginsKeepSemanticOrder(t *testing.T) {
	vt := New(10, 1, 100, nil)
	c1 := string([]byte{0x9b})

	raw := c1 + "?5W" + c1 + "?6h" + c1 + "?69h" + c1 + "3;6s" + c1 + "1;2HX" + c1 + "ZA"
	_, err, damage := vt.WriteWithDamage([]byte(raw))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if firstSemanticModeOp(t, damage, 69).Code != ScreenOpModes {
		t.Fatalf("expected C1 left/right margin mode semantic op, got %#v", damage.SemanticOps)
	}
	region := firstSemanticControlOp(t, damage, "decslrm")
	if region.Mode != 3 || region.Bottom != 6 {
		t.Fatalf("expected C1 DECSLRM left=3 right=6, got %#v damage=%#v", region, damage)
	}
	cbt := firstSemanticControlOp(t, damage, "cbt")
	if cbt.Col != 2 {
		t.Fatalf("C1 CBT should clamp to left margin final col 2, got %#v damage=%#v", cbt, damage)
	}
	var got []string
	for _, op := range damage.SemanticOps {
		switch op.Code {
		case ScreenOpModes:
			if op.Mode == 6 || op.Mode == 69 {
				got = append(got, modeOpLabel(op))
			}
		case ScreenOpControl:
			got = append(got, "control:"+op.Control)
		case ScreenOpWriteSpan:
			got = append(got, "write:"+damageOpText(op))
		}
	}
	want := []string{"control:decst8c", "mode:6:on", "mode:69:on", "control:decslrm", "control:cup", "write:X", "control:cbt", "write:A"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("C1 CSI left/right margin ops must preserve vterm-owned margin state order, got %v want %v damage=%#v", got, want, damage)
	}
	if row := rowText(vt.ScreenRowView(0), 10); row != "  AX      " {
		t.Fatalf("C1 left/right margin CBT should place A at left margin before X, got %q damage=%#v", row, damage)
	}
}

func TestVTermWriteWithDamageSemanticStyledClearCarriesEraseBlankStyle(t *testing.T) {
	vt := New(12, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[48;5;24mBG\x1b[K\x1b[0m"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	semanticEL := firstSemanticControlOp(t, damage, "el")
	if semanticEL.Mode != 0 {
		t.Fatalf("expected semantic EL mode 0, got %#v", semanticEL)
	}
	if len(semanticEL.Cells) != 1 || semanticEL.Cells[0].Style.BG != "idx:24" {
		t.Fatalf("semantic EL should carry erase blank background, got %#v damage=%#v", semanticEL, damage)
	}
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpClearToEOL || op.Code == ScreenOpClearRect {
			t.Fatalf("screen diff clear must not be semantic op, got %#v in %#v", op, damage.SemanticOps)
		}
	}
}

func TestVTermWriteWithDamageSemanticLinefeedCarriesTailFill(t *testing.T) {
	vt := New(8, 3, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("seed1\nseed2\n\x1b[48;5;24mabcdefghij\x1b[0m\n"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	semanticLF := lastSemanticControlOp(t, damage, "lf")
	if semanticLF.TailFill == nil || semanticLF.TailFill.BG != "idx:24" {
		t.Fatalf("semantic LF should carry row tail fill background, got %#v damage=%#v", semanticLF, damage.SemanticOps)
	}
}

func TestVTermWriteWithDamageBroadDirectSpanUsesFullReplace(t *testing.T) {
	vt := New(80, 24, 100, nil)
	prevWithDamage := safeEmulatorWriteWithDamage
	safeEmulatorWriteWithDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
		n, err := emu.Write(data)
		cells := make([]uv.Cell, emu.Width())
		for col := range cells {
			cells[col] = uv.Cell{Content: "x", Width: 1}
		}
		damages := make([]charmvt.Damage, emu.Height())
		for row := 0; row < emu.Height(); row++ {
			damages[row] = charmvt.SpanDamage{X: 0, Y: row, Cells: cells}
		}
		return n, err, damages, true
	}
	t.Cleanup(func() {
		safeEmulatorWriteWithDamage = prevWithDamage
	})

	_, err, damage := vt.WriteWithDamage([]byte("x"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !damage.RequiresFullReplace {
		t.Fatalf("expected broad direct span damage to use full replace, got %#v", damage)
	}
	if damage.FullReplaceReason != "broad_direct_cell_damage" {
		t.Fatalf("unexpected full replace reason %q", damage.FullReplaceReason)
	}
	if len(damage.Ops) != 0 {
		t.Fatalf("expected no expanded span ops for broad damage, got %#v", damage.Ops)
	}
	if len(damage.SemanticOps) != 0 {
		t.Fatalf("expected no semantic text ops for broad screen diff, got %#v", damage.SemanticOps)
	}
	if damage.DirectDamageItems != 24 || damage.DirectDamageRows != 24 || damage.DirectDamageCells != 1920 {
		t.Fatalf("unexpected direct damage stats: items=%d rows=%d cells=%d", damage.DirectDamageItems, damage.DirectDamageRows, damage.DirectDamageCells)
	}
	if len(damage.DirectDamageTouchedRows) != 24 || damage.DirectDamageTouchedRows[0] != 0 || damage.DirectDamageTouchedRows[23] != 23 {
		t.Fatalf("direct damage must expose sorted touched row proof, got %#v", damage.DirectDamageTouchedRows)
	}
}

func TestVTermWriteWithDamageBroadCompactTextCountsDirectDamage(t *testing.T) {
	vt := New(80, 24, 100, nil)
	var raw strings.Builder
	for row := 0; row < 20; row++ {
		if row > 0 {
			raw.WriteString("\r\n")
		}
		raw.WriteString(strings.Repeat("x", 80))
	}

	_, err, damage := vt.WriteWithDamage([]byte(raw.String()))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !damage.RequiresFullReplace || damage.FullReplaceReason != "broad_direct_cell_damage" {
		t.Fatalf("expected compact TextDamage to count toward broad full replace, got %#v", damage)
	}
	if damage.DirectDamageRows < 20 || damage.DirectDamageCells < 1600 {
		t.Fatalf("compact TextDamage must contribute direct stats, rows=%d cells=%d damage=%#v", damage.DirectDamageRows, damage.DirectDamageCells, damage)
	}
	if len(damage.DirectDamageTouchedRows) < 20 || damage.DirectDamageTouchedRows[0] != 0 {
		t.Fatalf("compact TextDamage must expose touched rows, got %#v", damage.DirectDamageTouchedRows)
	}
}

func TestVTermWriteWithDamageStyledCompactRunsCountDirectDamage(t *testing.T) {
	vt := New(80, 4, 100, nil)

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[31m" + strings.Repeat("x", 80) + "\x1b[0m"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if damage.DirectDamageRows != 1 || damage.DirectDamageCells != 80 {
		t.Fatalf("styled compact TextDamage runs must contribute direct stats, rows=%d cells=%d damage=%#v", damage.DirectDamageRows, damage.DirectDamageCells, damage)
	}
	if len(damage.SemanticOps) == 0 || len(damage.SemanticOps[0].Runs) == 0 || damage.SemanticOps[0].Runs[0].Text != strings.Repeat("x", 80) {
		t.Fatalf("expected styled compact run semantic op, got %#v", damage.SemanticOps)
	}
}

func TestVTermWriteWithSemanticDamageCompactTextCountsDirectDamage(t *testing.T) {
	vt := New(80, 24, 100, nil)
	var raw strings.Builder
	for row := 0; row < 20; row++ {
		if row > 0 {
			raw.WriteString("\r\n")
		}
		raw.WriteString("\x1b[31m")
		raw.WriteString(strings.Repeat("x", 80))
		raw.WriteString("\x1b[0m")
	}

	_, err, damage := vt.WriteWithSemanticDamage([]byte(raw.String()))
	if err != nil {
		t.Fatalf("write semantic damage: %v", err)
	}
	if !damage.RequiresFullReplace || damage.FullReplaceReason != "broad_direct_cell_damage" {
		t.Fatalf("expected semantic compact TextDamage to count toward broad full replace, got %#v", damage)
	}
	if damage.DirectDamageRows < 20 || damage.DirectDamageCells < 1600 {
		t.Fatalf("semantic compact TextDamage must contribute direct stats, rows=%d cells=%d damage=%#v", damage.DirectDamageRows, damage.DirectDamageCells, damage)
	}
	if len(damage.DirectDamageTouchedRows) < 20 || damage.DirectDamageTouchedRows[0] != 0 {
		t.Fatalf("semantic compact TextDamage must expose touched rows, got %#v", damage.DirectDamageTouchedRows)
	}
	if len(damage.SemanticOps) == 0 || len(damage.SemanticOps[0].Runs) == 0 {
		t.Fatalf("expected compact run semantic ops, got %#v", damage.SemanticOps)
	}
}

func TestVTermWriteWithDamageRepeatedDirectSpanUsesFullReplace(t *testing.T) {
	vt := New(146, 73, 100, nil)
	prevWithDamage := safeEmulatorWriteWithDamage
	safeEmulatorWriteWithDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
		n, err := emu.Write(data)
		cells := []uv.Cell{{Content: "x", Width: 1}}
		damages := make([]charmvt.Damage, 0, 6000)
		for col := 0; col < 6000; col++ {
			damages = append(damages, charmvt.SpanDamage{X: col % emu.Width(), Y: 2, Cells: cells})
		}
		return n, err, damages, true
	}
	t.Cleanup(func() {
		safeEmulatorWriteWithDamage = prevWithDamage
	})

	_, err, damage := vt.WriteWithDamage([]byte("x"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if !damage.RequiresFullReplace {
		t.Fatalf("expected repeated direct span damage to use full replace, got %#v", damage)
	}
	if damage.FullReplaceReason != "repeated_direct_damage" {
		t.Fatalf("unexpected full replace reason %q", damage.FullReplaceReason)
	}
	if len(damage.Ops) != 0 {
		t.Fatalf("expected no expanded span ops for repeated damage, got %#v", damage.Ops)
	}
	if len(damage.SemanticOps) != 0 {
		t.Fatalf("expected no semantic text ops for repeated screen diff, got %#v", damage.SemanticOps)
	}
	if damage.DirectDamageItems != 6000 || damage.DirectDamageRows != 1 || damage.DirectDamageCells != 6000 {
		t.Fatalf("unexpected direct damage stats: items=%d rows=%d cells=%d", damage.DirectDamageItems, damage.DirectDamageRows, damage.DirectDamageCells)
	}
	if len(damage.DirectDamageTouchedRows) != 1 || damage.DirectDamageTouchedRows[0] != 2 {
		t.Fatalf("repeated direct damage must expose touched row proof, got %#v", damage.DirectDamageTouchedRows)
	}
}

func TestVTermWriteWithDamageKeepsScrollDirectDamageIncremental(t *testing.T) {
	vt := New(10, 4, 100, nil)
	prevWithDamage := safeEmulatorWriteWithDamage
	safeEmulatorWriteWithDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
		n, err := emu.Write(data)
		cells := make([]uv.Cell, emu.Width())
		for col := range cells {
			cells[col] = uv.Cell{Content: "x", Width: 1}
		}
		damages := []charmvt.Damage{
			charmvt.ScrollDamage{Rectangle: uv.Rect(0, 0, emu.Width(), emu.Height()), Dy: -1},
			charmvt.SpanDamage{X: 0, Y: emu.Height() - 1, Cells: cells},
		}
		return n, err, damages, true
	}
	t.Cleanup(func() {
		safeEmulatorWriteWithDamage = prevWithDamage
	})

	_, err, damage := vt.WriteWithDamage([]byte("\n"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if damage.RequiresFullReplace {
		t.Fatalf("expected scroll damage to remain incremental, got %#v", damage)
	}
	if damage.DirectDamageItems != 2 || damage.DirectDamageRows != 1 || damage.DirectDamageCells != 10 {
		t.Fatalf("unexpected direct damage stats: items=%d rows=%d cells=%d", damage.DirectDamageItems, damage.DirectDamageRows, damage.DirectDamageCells)
	}
	foundScroll := false
	for _, op := range damage.Ops {
		if op.Code == ScreenOpScrollRect {
			foundScroll = true
		}
	}
	if !foundScroll {
		t.Fatalf("expected scroll op to be preserved, got %#v", damage.Ops)
	}
}

func TestVTermWriteWithDamagePreservesNonASCIITextScrollbackDamage(t *testing.T) {
	vt := New(12, 2, 100, nil)
	prevWithDamage := safeEmulatorWriteWithDamage
	safeEmulatorWriteWithDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
		n, err := emu.Write(data)
		damages := []charmvt.Damage{
			charmvt.ScrollbackDamage{
				Y:       0,
				Text:    "█▀█",
				ASCII:   false,
				Wrapped: false,
			},
		}
		return n, err, damages, true
	}
	t.Cleanup(func() {
		safeEmulatorWriteWithDamage = prevWithDamage
	})

	_, err, damage := vt.WriteWithDamage([]byte("x"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if len(damage.ScrollbackAppend) != 1 {
		t.Fatalf("expected one scrollback append row, got %#v", damage.ScrollbackAppend)
	}
	row := damage.ScrollbackAppend[0]
	got := ""
	for _, run := range row.Runs {
		got += run.Text
	}
	if got != "█▀█" {
		t.Fatalf("expected non-ASCII text scrollback row preserved, got %q row=%#v", got, row)
	}
}

func TestVTermWriteWithDamageDoesNotPadHardNewlineScrollbackCells(t *testing.T) {
	vt := New(12, 2, 100, nil)
	_, err, damage := vt.WriteWithDamage([]byte("████\r\nnext\r\nlast"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, op := range damage.ScrollbackAppend {
		if got := damageOpText(op); got != "████" {
			continue
		}
		if len(op.Cells) != 4 {
			t.Fatalf("expected hard-newline scrollback cells to keep stored line length, got len=%d row=%#v", len(op.Cells), op.Cells)
		}
		return
	}
	t.Fatalf("expected non-padded non-ASCII scrollback cells, got %#v", damage.ScrollbackAppend)
}

func TestVTermWriteAltScreenSwitchKeepsDamageCorrect(t *testing.T) {
	vt := New(5, 3, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{{Content: "a", Width: 1}},
			{{Content: "b", Width: 1}},
			{{Content: "c", Width: 1}},
		},
	}, CursorState{Row: 2, Col: 1, Visible: true}, TerminalModes{AutoWrap: true})

	_, err, damage := vt.WriteWithDamage([]byte("\x1b[?1049h"))
	if err != nil {
		t.Fatalf("enter alt-screen: %v", err)
	}
	if !vt.IsAltScreen() {
		t.Fatal("expected alt-screen to be enabled")
	}
	if !damage.RequiresFullReplace {
		t.Fatalf("expected full replace damage on alt-screen switch, got %#v", damage)
	}
	mode := firstSemanticModeOp(t, damage, 1049)
	if !mode.Private || !mode.Enabled {
		t.Fatalf("expected alt-screen enter semantic mode to survive full replace, got %#v ops=%#v", mode, damage.SemanticOps)
	}
	screen := vt.ScreenContent()
	for row := range screen.Cells {
		if strings.TrimSpace(rowToString(screen.Cells[row])) != "" {
			t.Fatalf("expected blank alt-screen row %d, got %q", row, rowToString(screen.Cells[row]))
		}
	}

	_, err, damage = vt.WriteWithDamage([]byte("\x1b[?1049l"))
	if err != nil {
		t.Fatalf("leave alt-screen: %v", err)
	}
	if vt.IsAltScreen() {
		t.Fatal("expected alt-screen to be disabled")
	}
	if !damage.RequiresFullReplace {
		t.Fatalf("expected full replace damage when restoring main screen, got %#v", damage)
	}
	mode = firstSemanticModeOp(t, damage, 1049)
	if !mode.Private || mode.Enabled {
		t.Fatalf("expected alt-screen exit semantic mode to survive full replace, got %#v ops=%#v", mode, damage.SemanticOps)
	}
	screen = vt.ScreenContent()
	if got := strings.TrimSpace(rowToString(screen.Cells[0])) + strings.TrimSpace(rowToString(screen.Cells[1])) + strings.TrimSpace(rowToString(screen.Cells[2])); got != "abc" {
		t.Fatalf("expected main-screen content restored, got %q", got)
	}
}

func TestVTermWriteWithDamageSemanticAltScreenModeVariants(t *testing.T) {
	for _, mode := range []int{47, 1047, 1049} {
		t.Run("mode_"+strconv.Itoa(mode), func(t *testing.T) {
			vt := New(8, 3, 100, nil)

			_, err, damage := vt.WriteWithDamage([]byte("\x1b[?" + strconv.Itoa(mode) + "h"))
			if err != nil {
				t.Fatalf("enter alt-screen mode %d: %v", mode, err)
			}
			enter := firstSemanticModeOp(t, damage, mode)
			if enter.Code != ScreenOpModes || !enter.Private || !enter.Enabled {
				t.Fatalf("expected alt-screen mode %d enter semantic op, got %#v ops=%#v", mode, enter, damage.SemanticOps)
			}
			if !damage.RequiresFullReplace {
				t.Fatalf("alt-screen mode %d enter should keep full-replace live boundary, damage=%#v", mode, damage)
			}

			_, err, damage = vt.WriteWithDamage([]byte("\x1b[?" + strconv.Itoa(mode) + "l"))
			if err != nil {
				t.Fatalf("exit alt-screen mode %d: %v", mode, err)
			}
			exit := firstSemanticModeOp(t, damage, mode)
			if exit.Code != ScreenOpModes || !exit.Private || exit.Enabled {
				t.Fatalf("expected alt-screen mode %d exit semantic op, got %#v ops=%#v", mode, exit, damage.SemanticOps)
			}
			if !damage.RequiresFullReplace {
				t.Fatalf("alt-screen mode %d exit should keep full-replace live boundary, damage=%#v", mode, damage)
			}
		})
	}
}

func TestVTermTracksMouseModesFromEscapeSequences(t *testing.T) {
	vt := New(20, 5, 100, nil)

	if vt.Modes().MouseTracking {
		t.Fatal("expected mouse tracking disabled by default")
	}
	if _, err := vt.Write([]byte("\x1b[?1002h\x1b[?1006h")); err != nil {
		t.Fatalf("enable mouse tracking failed: %v", err)
	}
	if !vt.Modes().MouseTracking {
		t.Fatal("expected mouse tracking after enabling button-event mode")
	}
	if !vt.Modes().MouseButtonEvent || !vt.Modes().MouseSGR {
		t.Fatalf("expected button-event+SGR mouse mode flags, got %#v", vt.Modes())
	}
	if _, err := vt.Write([]byte("\x1b[?1006l")); err != nil {
		t.Fatalf("disable sgr mode failed: %v", err)
	}
	if !vt.Modes().MouseTracking {
		t.Fatal("expected mouse tracking to remain enabled after disabling sgr encoding only")
	}
	if vt.Modes().MouseSGR {
		t.Fatalf("expected SGR flag cleared after disabling 1006, got %#v", vt.Modes())
	}
	if _, err := vt.Write([]byte("\x1b[?1002l")); err != nil {
		t.Fatalf("disable mouse tracking failed: %v", err)
	}
	if vt.Modes().MouseTracking {
		t.Fatal("expected mouse tracking disabled after reset")
	}
}

func TestLoadSnapshotRestoresMouseTrackingMode(t *testing.T) {
	vt := New(10, 4, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{{Content: "x", Width: 1}}},
	}, CursorState{Row: 0, Col: 1, Visible: true}, TerminalModes{AutoWrap: true, MouseTracking: true, MouseNormal: true})

	if !vt.Modes().MouseTracking {
		t.Fatal("expected snapshot restore to preserve mouse tracking")
	}
	if !vt.Modes().MouseNormal || vt.Modes().MouseSGR {
		t.Fatalf("expected snapshot restore to preserve legacy mouse encoding, got %#v", vt.Modes())
	}
}

func TestLoadSnapshotRestoresSGRMouseEncodingMode(t *testing.T) {
	vt := New(10, 4, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{{Content: "x", Width: 1}}},
	}, CursorState{Row: 0, Col: 1, Visible: true}, TerminalModes{AutoWrap: true, MouseTracking: true, MouseButtonEvent: true, MouseSGR: true})

	if !vt.Modes().MouseTracking || !vt.Modes().MouseButtonEvent || !vt.Modes().MouseSGR {
		t.Fatalf("expected snapshot restore to preserve SGR mouse encoding, got %#v", vt.Modes())
	}
}

func TestVTermTracksAlternateScrollModeFromEscapeSequences(t *testing.T) {
	vt := New(20, 5, 100, nil)

	if vt.Modes().AlternateScroll {
		t.Fatal("expected alternate scroll disabled by default")
	}
	if _, err := vt.Write([]byte("\x1b[?1007h")); err != nil {
		t.Fatalf("enable alternate scroll failed: %v", err)
	}
	if !vt.Modes().AlternateScroll {
		t.Fatal("expected alternate scroll enabled after escape sequence")
	}
	if _, err := vt.Write([]byte("\x1b[?1007l")); err != nil {
		t.Fatalf("disable alternate scroll failed: %v", err)
	}
	if vt.Modes().AlternateScroll {
		t.Fatal("expected alternate scroll disabled after reset")
	}
}

func TestLoadSnapshotRestoresAlternateScrollMode(t *testing.T) {
	vt := New(10, 4, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{{Content: "x", Width: 1}}},
	}, CursorState{Row: 0, Col: 1, Visible: true}, TerminalModes{AutoWrap: true, AlternateScroll: true})

	if !vt.Modes().AlternateScroll {
		t.Fatal("expected snapshot restore to preserve alternate scroll")
	}
}

func TestApplyScreenUpdateAppliesWriteSpanInPlace(t *testing.T) {
	vt := New(6, 3, 100, nil)
	now := time.Date(2026, 4, 18, 8, 0, 0, 0, time.UTC)
	vt.LoadSnapshotWithMetadata(nil, nil, nil, ScreenData{
		Cells: [][]Cell{
			{
				{Content: "o", Width: 1},
				{Content: "l", Width: 1},
				{Content: "d", Width: 1},
				{Content: " ", Width: 1},
			},
			{
				{Content: "r", Width: 1},
				{Content: "o", Width: 1},
				{Content: "w", Width: 1},
				{Content: " ", Width: 1},
			},
		},
		IsAlternateScreen: true,
	}, []time.Time{now, now}, []string{"old-0", "old-1"}, CursorState{Row: 1, Col: 3, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	oldEmu := vt.emu
	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 4, Rows: 2},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  1,
			Col:  0,
			Cells: []Cell{
				{Content: "n", Width: 1},
				{Content: "e", Width: 1},
				{Content: "w", Width: 1},
				{Content: "!", Width: 1},
			},
			Timestamp: now.Add(time.Second),
			RowKind:   "new-1",
		}},
		Cursor: CursorState{Row: 1, Col: 4, Visible: true, Shape: "bar"},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected incremental screen update to apply")
	}

	if vt.emu != oldEmu {
		t.Fatal("expected incremental apply to keep the existing emulator instance")
	}
	screen := vt.ScreenContent()
	if got := screen.Cells[1][0].Content + screen.Cells[1][1].Content + screen.Cells[1][2].Content + screen.Cells[1][3].Content; got != "new!" {
		t.Fatalf("expected updated row content, got %q", got)
	}
	if got := vt.ScreenRowTimestampAt(1); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("expected updated row timestamp, got %v", got)
	}
	if got := vt.ScreenRowKindAt(1); got != "new-1" {
		t.Fatalf("expected updated row kind, got %q", got)
	}
	if cursor := vt.CursorState(); cursor.Row != 1 || cursor.Col != 4 || cursor.Shape != CursorBar {
		t.Fatalf("expected updated cursor, got %#v", cursor)
	}
}

func TestApplyScreenUpdateAppliesClearToEOLOp(t *testing.T) {
	vt := New(8, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "p", Width: 1},
			{Content: "r", Width: 1},
			{Content: "e", Width: 1},
			{Content: "f", Width: 1},
			{Content: "i", Width: 1},
			{Content: "x", Width: 1},
			{Content: "X", Width: 1},
			{Content: "Y", Width: 1},
		}},
		IsAlternateScreen: true,
	}, CursorState{Row: 0, Col: 8, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 8, Rows: 1},
		Ops: []DamageOp{{
			Code: ScreenOpClearToEOL,
			Row:  0,
			Col:  6,
		}},
		Cursor: CursorState{Row: 0, Col: 6, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected clear-to-eol span to apply incrementally")
	}

	row := vt.ScreenRowView(0)
	if got := strings.TrimRight(rowToString(row), " "); got != "prefix" {
		t.Fatalf("expected row tail cleared, got %q", rowToString(row))
	}
}

func TestApplyScreenUpdateAppliesStyleOnlyWriteOp(t *testing.T) {
	vt := New(8, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "p", Width: 1},
			{Content: "l", Width: 1},
			{Content: "a", Width: 1},
			{Content: "i", Width: 1},
			{Content: "n", Width: 1},
			{Content: "x", Width: 1},
		}},
		IsAlternateScreen: true,
	}, CursorState{Row: 0, Col: 6, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 8, Rows: 1},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  0,
			Col:  5,
			Cells: []Cell{{
				Content: "x",
				Width:   1,
				Style:   CellStyle{Bold: true},
			}},
			Timestamp: time.Date(2026, 4, 18, 8, 0, 2, 0, time.UTC),
			RowKind:   "style-only",
		}},
		Cursor: CursorState{Row: 0, Col: 6, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected style-only span to apply incrementally")
	}

	cell := vt.ScreenRowView(0)[5]
	if cell.Content != "x" || !cell.Style.Bold {
		t.Fatalf("expected bold cell after style-only span, got %#v", cell)
	}
	if got := vt.ScreenRowKindAt(0); got != "style-only" {
		t.Fatalf("expected row kind updated by style-only span, got %q", got)
	}
}

func TestVTermPreservesHyperlinkAcrossSnapshotAndScreenUpdate(t *testing.T) {
	vt := New(8, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "l", Width: 1, LinkURL: "https://example.test/snapshot", LinkParams: "id=snapshot"},
			{Content: "i", Width: 1, LinkURL: "https://example.test/snapshot", LinkParams: "id=snapshot"},
			{Content: "n", Width: 1},
		}},
	}, CursorState{Row: 0, Col: 3, Visible: true}, TerminalModes{AutoWrap: true})

	if got := vt.ScreenRowView(0)[0]; got.LinkURL != "https://example.test/snapshot" || got.LinkParams != "id=snapshot" {
		t.Fatalf("expected snapshot hyperlink restored, got %#v", got)
	}

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 8, Rows: 1},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  0,
			Col:  2,
			Cells: []Cell{{
				Content:    "k",
				Width:      1,
				LinkURL:    "https://example.test/update",
				LinkParams: "id=update",
			}},
		}},
		Cursor: CursorState{Row: 0, Col: 3, Visible: true},
		Modes:  TerminalModes{AutoWrap: true},
	}) {
		t.Fatal("expected hyperlink span to apply incrementally")
	}

	cell := vt.ScreenRowView(0)[2]
	if cell.Content != "k" || cell.LinkURL != "https://example.test/update" || cell.LinkParams != "id=update" {
		t.Fatalf("expected updated hyperlink cell, got %#v", cell)
	}
}

func TestVTermReplayResetsHyperlinkBeforePlainText(t *testing.T) {
	var b strings.Builder
	writeSequentialRows(&b, [][]Cell{{
		{Content: "l", Width: 1, LinkURL: "https://example.test"},
		{Content: "p", Width: 1},
	}})

	got := b.String()
	if !strings.Contains(got, "\x1b]8;;https://example.test\x07") {
		t.Fatalf("expected replay to set hyperlink, got %q", got)
	}
	plainIndex := strings.Index(got, "p")
	resetIndex := strings.LastIndex(got[:plainIndex], "\x1b]8;;\x07")
	if resetIndex < 0 {
		t.Fatalf("expected hyperlink reset before plain cell, got %q", got)
	}

	b.Reset()
	writeSequentialRows(&b, [][]Cell{{
		{Content: "l", Width: 1, LinkURL: "https://example.test"},
	}})
	got = b.String()
	if !strings.HasSuffix(got, "\x1b]8;;\x07\x1b[0m") {
		t.Fatalf("expected final replay reset to close hyperlink state, got %q", got)
	}
}

func TestVTermReplayPreservesTrailingStyledAndLinkedBlanks(t *testing.T) {
	var b strings.Builder
	writeSequentialRows(&b, [][]Cell{{
		{Content: " ", Width: 1, Style: CellStyle{BG: "#222222"}},
		{Content: " ", Width: 1, LinkURL: "https://example.test/tail"},
	}})

	got := b.String()
	if !strings.Contains(got, "48;2;34;34;34") {
		t.Fatalf("expected trailing styled blank to be replayed, got %q", got)
	}
	if !strings.Contains(got, "\x1b]8;;https://example.test/tail\x07") {
		t.Fatalf("expected trailing linked blank to be replayed, got %q", got)
	}
}

func TestApplyScreenUpdateAppliesWideCharBoundaryWriteOp(t *testing.T) {
	vt := New(8, 1, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{{
			{Content: "你", Width: 2},
			{Content: "", Width: 0},
			{Content: "a", Width: 1},
		}},
		IsAlternateScreen: true,
	}, CursorState{Row: 0, Col: 3, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 8, Rows: 1},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  0,
			Col:  0,
			Cells: []Cell{
				{Content: "界", Width: 2},
				{Content: "", Width: 0},
			},
		}},
		Cursor: CursorState{Row: 0, Col: 3, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected wide-boundary span to apply incrementally")
	}

	row := vt.ScreenRowView(0)
	if got := row[0]; got.Content != "界" || got.Width != 2 {
		t.Fatalf("expected wide anchor updated, got %#v", got)
	}
	if got := row[1]; got.Content != "" || got.Width != 0 {
		t.Fatalf("expected wide continuation preserved, got %#v", got)
	}
}

func TestApplyScreenUpdateAppliesOpcodeScrollRect(t *testing.T) {
	vt := New(4, 4, 100, nil)
	now := time.Date(2026, 4, 18, 8, 0, 4, 0, time.UTC)
	vt.LoadSnapshotWithMetadata(nil, nil, nil, ScreenData{
		Cells: [][]Cell{
			{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "1", Width: 1}},
			{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "2", Width: 1}},
			{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "3", Width: 1}},
			{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "4", Width: 1}},
		},
		IsAlternateScreen: true,
	}, []time.Time{now, now.Add(time.Second), now.Add(2 * time.Second), now.Add(3 * time.Second)}, []string{"a", "b", "c", "d"}, CursorState{Row: 3, Col: 0, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	oldEmu := vt.emu
	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size:         Size{Cols: 4, Rows: 4},
		ScreenScroll: 1,
		Ops: []DamageOp{
			{Code: ScreenOpScrollRect, Rect: DamageRect{X: 0, Y: 0, Width: 4, Height: 4}, Dy: -1},
			{Code: ScreenOpWriteSpan, Row: 3, Col: 0, Cells: []Cell{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "5", Width: 1}}, Timestamp: now.Add(4 * time.Second), RowKind: "e"},
		},
		Cursor: CursorState{Row: 3, Col: 0, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected opcode scrollrect update to apply incrementally")
	}
	if vt.emu != oldEmu {
		t.Fatal("expected opcode incremental apply to keep emulator instance")
	}
	screen := vt.ScreenContent()
	got := []string{
		rowToString(screen.Cells[0]),
		rowToString(screen.Cells[1]),
		rowToString(screen.Cells[2]),
		rowToString(screen.Cells[3]),
	}
	if !reflect.DeepEqual(got, []string{"row2", "row3", "row4", "row5"}) {
		t.Fatalf("unexpected opcode scrollrect rows: %#v", got)
	}
	if rowKind := vt.ScreenRowKindAt(0); rowKind != "b" {
		t.Fatalf("expected shifted row kind on row 0, got %q", rowKind)
	}
	if rowKind := vt.ScreenRowKindAt(3); rowKind != "e" {
		t.Fatalf("expected tail write row kind on row 3, got %q", rowKind)
	}
}

func TestApplyScreenUpdateAppliesOpcodeCopyRect(t *testing.T) {
	vt := New(4, 3, 100, nil)
	now := time.Date(2026, 4, 18, 8, 0, 5, 0, time.UTC)
	vt.LoadSnapshotWithMetadata(nil, nil, nil, ScreenData{
		Cells: [][]Cell{
			{{Content: "A", Width: 1}, {Content: "B", Width: 1}, {Content: "C", Width: 1}, {Content: "D", Width: 1}},
			{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "2", Width: 1}},
			{{Content: "r", Width: 1}, {Content: "o", Width: 1}, {Content: "w", Width: 1}, {Content: "3", Width: 1}},
		},
		IsAlternateScreen: true,
	}, []time.Time{now, now.Add(time.Second), now.Add(2 * time.Second)}, []string{"a", "b", "c"}, CursorState{Row: 2, Col: 0, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 4, Rows: 3},
		Ops: []DamageOp{
			{Code: ScreenOpCopyRect, Src: DamageRect{X: 0, Y: 0, Width: 4, Height: 1}, DstX: 0, DstY: 2},
		},
		Cursor: CursorState{Row: 2, Col: 0, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected opcode copyrect update to apply incrementally")
	}

	screen := vt.ScreenContent()
	got := []string{
		rowToString(screen.Cells[0]),
		rowToString(screen.Cells[1]),
		rowToString(screen.Cells[2]),
	}
	if !reflect.DeepEqual(got, []string{"ABCD", "row2", "ABCD"}) {
		t.Fatalf("unexpected opcode copyrect rows: %#v", got)
	}
	if rowKind := vt.ScreenRowKindAt(2); rowKind != "a" {
		t.Fatalf("expected copied row kind on destination row, got %q", rowKind)
	}
}

func TestApplyScreenUpdateAppliesResizeThenSparseWriteOp(t *testing.T) {
	vt := New(4, 2, 100, nil)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{
				{Content: "a", Width: 1},
				{Content: "b", Width: 1},
				{Content: "c", Width: 1},
				{Content: "d", Width: 1},
			},
		},
		IsAlternateScreen: true,
	}, CursorState{Row: 0, Col: 4, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 8, Rows: 3},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  2,
			Col:  5,
			Cells: []Cell{
				{Content: "o", Width: 1},
				{Content: "k", Width: 1},
			},
		}},
		Cursor: CursorState{Row: 2, Col: 7, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected resize + sparse span update to apply incrementally")
	}

	if cols, rows := vt.Size(); cols != 8 || rows != 3 {
		t.Fatalf("expected resized terminal 8x3, got %dx%d", cols, rows)
	}
	row := vt.ScreenRowView(2)
	if got := row[5].Content + row[6].Content; got != "ok" {
		t.Fatalf("expected sparse span applied after resize, got %#v", row)
	}
}

func TestApplyScreenUpdateRejectsUnsupportedResetScrollback(t *testing.T) {
	vt := New(6, 3, 100, nil)
	vt.LoadSnapshotWithScrollback([][]Cell{{{Content: "o", Width: 1}}}, ScreenData{
		Cells: [][]Cell{{
			{Content: "n", Width: 1},
		}},
	}, CursorState{Row: 0, Col: 1, Visible: true}, TerminalModes{AutoWrap: true})

	oldEmu := vt.emu
	if vt.ApplyScreenUpdate(ScreenUpdate{
		Size:            Size{Cols: 1, Rows: 1},
		ResetScrollback: true,
		Cursor:          CursorState{Row: 0, Col: 1, Visible: true},
		Modes:           TerminalModes{AutoWrap: true},
	}) {
		t.Fatal("expected reset scrollback to fall back instead of partial apply")
	}
	if vt.emu != oldEmu {
		t.Fatal("expected rejected partial apply to leave emulator untouched")
	}
}

func TestApplyScreenUpdateAllowsSafeResizeWithoutRecreatingEmulator(t *testing.T) {
	vt := New(4, 2, 100, nil)
	now := time.Date(2026, 4, 18, 8, 30, 0, 0, time.UTC)
	vt.LoadSnapshot(ScreenData{
		Cells: [][]Cell{
			{
				{Content: "a", Width: 1},
				{Content: "b", Width: 1},
				{Content: "c", Width: 1},
				{Content: "d", Width: 1},
			},
			{
				{Content: "1", Width: 1},
				{Content: "2", Width: 1},
				{Content: "3", Width: 1},
				{Content: "4", Width: 1},
			},
		},
		IsAlternateScreen: true,
	}, CursorState{Row: 1, Col: 3, Visible: true}, TerminalModes{AlternateScreen: true, AutoWrap: true})

	oldEmu := vt.emu
	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 6, Rows: 3},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  2,
			Col:  0,
			Cells: []Cell{
				{Content: "x", Width: 1},
				{Content: "y", Width: 1},
			},
			Timestamp: now,
		}},
		Cursor: CursorState{Row: 2, Col: 2, Visible: true},
		Modes:  TerminalModes{AlternateScreen: true, AutoWrap: true},
	}) {
		t.Fatal("expected resize + changed rows update to apply incrementally")
	}
	if vt.emu != oldEmu {
		t.Fatal("expected resize path to keep the existing emulator instance")
	}
	if cols, rows := vt.Size(); cols != 6 || rows != 3 {
		t.Fatalf("expected resized terminal 6x3, got %dx%d", cols, rows)
	}
	screen := vt.ScreenContent()
	if got := screen.Cells[2][0].Content + screen.Cells[2][1].Content; got != "xy" {
		t.Fatalf("expected appended resized row content, got %q", got)
	}
	if cursor := vt.CursorState(); cursor.Row != 2 || cursor.Col != 2 {
		t.Fatalf("expected cursor moved after resize update, got %#v", cursor)
	}
}

func TestApplyScreenUpdateUpdatesScrollbackOwnershipWithoutRecreatingEmulator(t *testing.T) {
	vt := New(4, 2, 100, nil)
	now := time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC)
	vt.LoadSnapshotWithOwnership([][]Cell{
		{{Content: "a", Width: 1}},
		{{Content: "b", Width: 1}},
		{{Content: "c", Width: 1}},
	}, []time.Time{now, now.Add(time.Second), now.Add(2 * time.Second)}, []string{"a", "b", "c"}, nil,
		[]string{"persisted", "persisted", "live-tail-live"}, ScreenData{
			Cells: [][]Cell{
				{
					{Content: "x", Width: 1},
					{Content: "y", Width: 1},
				},
			},
		}, []time.Time{now}, []string{"screen"}, nil, nil, CursorState{Row: 0, Col: 2, Visible: true}, TerminalModes{AutoWrap: true})

	oldEmu := vt.emu
	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size:           Size{Cols: 2, Rows: 1},
		ScrollbackTrim: 1,
		ScrollbackAppend: []ScrollbackRowAppend{{
			Cells:     []Cell{{Content: "d", Width: 1}},
			Timestamp: now.Add(3 * time.Second),
			RowKind:   "d",
			Ownership: "live-tail-live",
		}},
		Cursor: CursorState{Row: 0, Col: 2, Visible: true},
		Modes:  TerminalModes{AutoWrap: true},
	}) {
		t.Fatal("expected scrollback update to apply incrementally")
	}
	if vt.emu != oldEmu {
		t.Fatal("expected scrollback update to keep the existing emulator instance")
	}
	scrollback := vt.ScrollbackContent()
	if len(scrollback) != 3 {
		t.Fatalf("expected trimmed+appended scrollback length 3, got %d", len(scrollback))
	}
	if got := scrollback[0][0].Content + scrollback[1][0].Content + scrollback[2][0].Content; got != "bcd" {
		t.Fatalf("expected scrollback to become bcd, got %q", got)
	}
	if got := vt.ScrollbackRowTimestampAt(0); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("expected retained timestamp after trim, got %v", got)
	}
	if got := vt.ScrollbackRowKindAt(2); got != "d" {
		t.Fatalf("expected appended row kind, got %q", got)
	}
	if got := vt.ScrollbackOwnership(); !reflect.DeepEqual(got, []string{"persisted", "live-tail-live", "live-tail-live"}) {
		t.Fatalf("expected ownership tail to follow trim+append, got %#v", got)
	}
}

func TestApplyScreenUpdateScrollbackAppendKeepsMetadataTailWhenCapped(t *testing.T) {
	vt := New(4, 2, 3, nil)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	vt.LoadSnapshotWithOwnership([][]Cell{
		{{Content: "a", Width: 1}},
		{{Content: "b", Width: 1}},
		{{Content: "c", Width: 1}},
	}, []time.Time{now, now.Add(time.Second), now.Add(2 * time.Second)}, []string{"a", "b", "c"}, nil,
		[]string{"persisted", "persisted", "live-tail-live"}, ScreenData{
			Cells: [][]Cell{{{Content: "x", Width: 1}}},
		}, []time.Time{now}, []string{"screen"}, nil, nil, CursorState{Row: 0, Col: 1, Visible: true}, TerminalModes{AutoWrap: true})

	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 1, Rows: 1},
		ScrollbackAppend: []ScrollbackRowAppend{{
			Cells:     []Cell{{Content: "d", Width: 1}},
			Timestamp: now.Add(3 * time.Second),
			RowKind:   "d",
			Ownership: "live-tail-live",
		}},
		Cursor: CursorState{Row: 0, Col: 1, Visible: true},
		Modes:  TerminalModes{AutoWrap: true},
	}) {
		t.Fatal("expected capped scrollback append to apply incrementally")
	}

	scrollback := vt.ScrollbackContent()
	if len(scrollback) != 3 {
		t.Fatalf("expected capped scrollback length 3, got %d", len(scrollback))
	}
	if got := scrollback[0][0].Content + scrollback[1][0].Content + scrollback[2][0].Content; got != "bcd" {
		t.Fatalf("expected capped scrollback to keep bcd, got %q", got)
	}
	if got := vt.ScrollbackRowTimestampAt(0); !got.Equal(now.Add(time.Second)) {
		t.Fatalf("expected metadata to drop capped prefix, got %v", got)
	}
	if got := vt.ScrollbackRowKindAt(0); got != "b" {
		t.Fatalf("expected metadata row 0 kind b after cap, got %q", got)
	}
	if got := vt.ScrollbackRowKindAt(2); got != "d" {
		t.Fatalf("expected appended metadata kind d, got %q", got)
	}
	if got := vt.ScrollbackOwnership(); !reflect.DeepEqual(got, []string{"persisted", "live-tail-live", "live-tail-live"}) {
		t.Fatalf("expected capped ownership to keep committed/live-tail suffix, got %#v", got)
	}
}

func TestApplyScreenUpdateWriteSpanPreservesTrailingBlankUsedWidthAfterResize(t *testing.T) {
	vt := New(4, 3, 100, nil)
	if !vt.ApplyScreenUpdate(ScreenUpdate{
		Size: Size{Cols: 4, Rows: 3},
		Ops: []DamageOp{{
			Code: ScreenOpWriteSpan,
			Row:  0,
			Col:  0,
			Cells: []Cell{
				{Content: "A", Width: 1},
				{Content: "A", Width: 1},
				{Content: " ", Width: 1},
				{Content: " ", Width: 1},
			},
		}},
		Cursor: CursorState{Row: 0, Col: 4, Visible: true},
		Modes:  TerminalModes{AutoWrap: true},
	}) {
		t.Fatal("expected trailing-space write span to apply incrementally")
	}

	vt.Resize(2, 3)
	if got := rowText(vt.ScreenRowView(0), 2); got != "AA" {
		t.Fatalf("expected first resized row from write span, got %q", got)
	}
	if got := rowText(vt.ScreenRowView(1), 2); got != "  " {
		t.Fatalf("expected trailing blank cells from write span to survive resize, got %q row=%#v", got, vt.ScreenRowView(1))
	}
}

func TestWriteWithDamageDoesNotPadHardNewlineScrollbackRowsToScreenWidth(t *testing.T) {
	vt := New(12, 2, 100, nil)
	_, err, damage := vt.WriteWithDamage([]byte("████\r\nnext\r\nlast"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, op := range damage.ScrollbackAppend {
		if damageOpText(op) != "████" {
			continue
		}
		if op.Wrapped {
			t.Fatalf("expected hard-newline row to remain unwrapped, got %#v", op)
		}
		if got := len(op.Cells); got != 4 {
			t.Fatalf("expected hard-newline scrollback cells to keep stored line length, got len=%d row=%#v", got, op.Cells)
		}
		return
	}
	t.Fatalf("expected non-padded scrollback row, got %#v", damage.ScrollbackAppend)
}

func TestWriteForLatestFrameSkipsScrollbackDamagePayload(t *testing.T) {
	vt := New(12, 2, 100, nil)
	_, err, damage := vt.WriteForLatestFrame([]byte("first\r\nsecond\r\nthird"))
	if err != nil {
		t.Fatalf("write latest frame: %v", err)
	}
	if !damage.RequiresFullReplace {
		t.Fatalf("expected latest-frame write to advertise full replace, got %#v", damage)
	}
	if len(damage.ScrollbackAppend) != 0 || len(damage.AlternateAppend) != 0 {
		t.Fatalf("latest-frame path must not allocate scrollback damage payload, got %#v", damage)
	}
	if got := strings.Join(trimmedScreenRowsText(vt.ScreenContent().Cells), "\n"); !strings.Contains(got, "third") {
		t.Fatalf("expected latest screen to keep newest output, got %q", got)
	}

	vtWithDamage := New(12, 2, 100, nil)
	_, err, damage = vtWithDamage.WriteWithDamage([]byte("first\r\nsecond\r\nthird"))
	if err != nil {
		t.Fatalf("write with damage: %v", err)
	}
	if len(damage.ScrollbackAppend) == 0 {
		t.Fatalf("expected incremental damage path to keep scrollback append payload, got %#v", damage)
	}
}

func TestWriteForLatestFrameExportsOnlyFingerprintProvenRowCopies(t *testing.T) {
	vt := New(12, 3, 100, nil)
	if _, err, _ := vt.WriteForLatestFrame([]byte("\x1b[1;1Hone\x1b[2;1Htwo\x1b[3;1Hthree")); err != nil {
		t.Fatalf("seed latest frame: %v", err)
	}
	_, err, damage := vt.WriteForLatestFrame([]byte("\r\nfour"))
	if err != nil {
		t.Fatalf("scroll latest frame: %v", err)
	}
	if !damage.IncrementalRowsReliable {
		t.Fatalf("scroll should retain exact row provenance: %#v", damage)
	}
	if len(damage.RowCopies) != 1 || damage.RowCopies[0] != (RowCopy{SourceRow: 1, DestinationRow: 0, Count: 2}) {
		t.Fatalf("expected two proven moved rows and one replacement, got %#v", damage.RowCopies)
	}
}

func firstWriteSpanOp(t *testing.T, damage WriteDamage) DamageOp {
	t.Helper()
	return firstOpWithCode(t, damage, ScreenOpWriteSpan)
}

func firstOpWithCode(t *testing.T, damage WriteDamage, code ScreenOpCode) DamageOp {
	t.Helper()
	for _, op := range damage.Ops {
		if op.Code == code {
			return op
		}
	}
	t.Fatalf("expected op %v in damage, got %#v", code, damage)
	return DamageOp{}
}

func firstSemanticOpWithCode(t *testing.T, damage WriteDamage, code ScreenOpCode) DamageOp {
	t.Helper()
	for _, op := range damage.SemanticOps {
		if op.Code == code {
			return op
		}
	}
	t.Fatalf("expected semantic op %v in damage, got %#v", code, damage)
	return DamageOp{}
}

func firstControlOp(t *testing.T, damage WriteDamage, control string) DamageOp {
	t.Helper()
	for _, op := range damage.Ops {
		if op.Code == ScreenOpControl && op.Control == control {
			return op
		}
	}
	t.Fatalf("expected control op %q in damage, got %#v", control, damage)
	return DamageOp{}
}

func firstSemanticControlOp(t *testing.T, damage WriteDamage, control string) DamageOp {
	t.Helper()
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpControl && op.Control == control {
			return op
		}
	}
	t.Fatalf("expected semantic control op %q in damage, got %#v", control, damage)
	return DamageOp{}
}

func lastSemanticControlOp(t *testing.T, damage WriteDamage, control string) DamageOp {
	t.Helper()
	for i := len(damage.SemanticOps) - 1; i >= 0; i-- {
		op := damage.SemanticOps[i]
		if op.Code == ScreenOpControl && op.Control == control {
			return op
		}
	}
	t.Fatalf("expected semantic control op %q in damage, got %#v", control, damage)
	return DamageOp{}
}

func firstSemanticModeOp(t *testing.T, damage WriteDamage, mode int) DamageOp {
	t.Helper()
	for _, op := range damage.SemanticOps {
		if op.Code == ScreenOpModes && op.Mode == mode {
			return op
		}
	}
	t.Fatalf("expected semantic mode op %d in damage, got %#v", mode, damage)
	return DamageOp{}
}

func modeOpLabel(op DamageOp) string {
	state := "off"
	if op.Enabled {
		state = "on"
	}
	return "mode:" + strconv.Itoa(op.Mode) + ":" + state
}

func semanticOpsContainText(ops []DamageOp, text string) bool {
	for _, op := range ops {
		if op.Code == ScreenOpWriteSpan && strings.Contains(damageOpText(op), text) {
			return true
		}
	}
	return false
}
