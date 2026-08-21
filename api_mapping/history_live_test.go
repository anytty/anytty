package apimapping

import (
	"fmt"
	"reflect"
	"testing"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
	vterm "github.com/anytty/anytty/vterm/vterm"
	"google.golang.org/protobuf/proto"
)

func TestValidateHistoryWindowRequiresFrozenPaginationToken(t *testing.T) {
	terminal := &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}
	for _, mode := range []apipb.HistoryWindowMode{
		apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER,
		apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER,
		apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST,
	} {
		t.Run(mode.String(), func(t *testing.T) {
			command := &apipb.CommandEnvelope{
				Context: terminalRequestContext("history-pagination"),
				Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowCommand{
					Terminal: terminal,
					Mode:     mode,
					Limit:    1,
				}},
			}
			if err := ValidateHistoryLiveCommand(command); err == nil {
				t.Fatal("tokenless pagination must fail validation")
			}
		})
	}
	for _, mode := range []apipb.HistoryWindowMode{
		apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_UNSPECIFIED,
		apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
	} {
		command := &apipb.CommandEnvelope{
			Context: terminalRequestContext("history-latest"),
			Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowCommand{
				Terminal: terminal,
				Mode:     mode,
				Limit:    1,
			}},
		}
		if err := ValidateHistoryLiveCommand(command); err != nil {
			t.Fatalf("latest mode %s failed validation: %v", mode, err)
		}
	}
	unknown := &apipb.CommandEnvelope{
		Context: terminalRequestContext("history-unknown"),
		Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowCommand{
			Terminal: terminal,
			Mode:     apipb.HistoryWindowMode(99),
			Limit:    1,
		}},
	}
	if err := ValidateHistoryLiveCommand(unknown); err == nil {
		t.Fatal("unknown history window mode must fail validation")
	}
}

func TestValidateHistoryCopyRequiresPositiveRangeLineIDs(t *testing.T) {
	terminal := &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}
	command := func(startLineID, endLineID uint64) *apipb.CommandEnvelope {
		return &apipb.CommandEnvelope{
			Context: terminalRequestContext("history-copy-range"),
			Command: &apipb.CommandEnvelope_HistoryCopy{HistoryCopy: &apipb.HistoryCopyCommand{
				Terminal: terminal,
				Window: &apipb.HistoryWindowCommand{
					Terminal: terminal,
					Token:    "frozen-token",
					Range:    &apipb.HistoryRange{StartLineId: startLineID, EndLineId: endLineID},
				},
			}},
		}
	}
	for _, ids := range [][2]uint64{{0, 1}, {1, 0}, {0, 0}} {
		if err := ValidateHistoryLiveCommand(command(ids[0], ids[1])); err == nil {
			t.Fatalf("copy range line IDs %v must fail validation", ids)
		}
	}
	if err := ValidateHistoryLiveCommand(command(1, 2)); err != nil {
		t.Fatalf("positive copy range line IDs failed validation: %v", err)
	}
}

func TestHistoryCursorRoundTripsLogicalCoordinates(t *testing.T) {
	want := history.HistoryCursor{
		LineID:     42,
		RowInLine:  3,
		Segment:    history.HistorySegmentArchivedPrimaryFrame,
		Generation: 7,
		Token:      "frozen-token",
		Valid:      true,
	}
	wire := historyCursorToProto(want)
	got := historyCursorFromProto(wire, uint64(want.Generation), string(want.Token))
	if got != want {
		t.Fatalf("history cursor round trip = %#v, want %#v", got, want)
	}
}

func TestHistoryWindowToProtoCoalescesAdjacentStyleRuns(t *testing.T) {
	red := history.CellStyle{FG: "ansi:1", Bold: true}
	window := HistoryWindowToProto("machine", history.HistoryWindow{
		TerminalID: "terminal",
		ViewportAnchor: history.HistoryViewportAnchor{
			TopLineID: 42, TopCellOffset: 17, ScreenCols: 80, ScreenRows: 24, Valid: true,
		},
		Rows: []history.HistoryRow{{Cells: []history.Cell{
			{Text: "a", Width: 1, Style: red},
			{Text: "好", Width: 2, Style: red},
			{Text: "b", Width: 1, Style: red, LinkURL: "https://example.test"},
			{Text: "c", Width: 1, Style: red, LinkURL: "https://example.test"},
			{Text: "", Width: 1},
			{Text: "", Width: 2},
		}}},
	})
	cells := window.GetRows()[0].GetRow().GetCells()
	if len(cells) != 5 {
		t.Fatalf("coalesced history cells = %#v", cells)
	}
	if cells[0].GetContent() != "a" || cells[0].GetWidth() != 1 || cells[1].GetContent() != "好" || cells[1].GetWidth() != 2 || cells[2].GetContent() != "bc" || cells[2].GetWidth() != 2 || cells[3].GetContent() != "" || cells[3].GetWidth() != 1 || cells[4].GetContent() != "" || cells[4].GetWidth() != 2 {
		t.Fatalf("coalesced history runs lost content or width: %#v", cells)
	}
	if anchor := window.GetViewportAnchor(); anchor.GetTopLineId() != 42 || anchor.GetTopCellOffset() != 17 || anchor.GetScreenCols() != 80 || anchor.GetScreenRows() != 24 {
		t.Fatalf("viewport anchor lost in protocol mapping: %#v", anchor)
	}
}

func TestNativeScreenToProtoCoalescesAdjacentStyleRuns(t *testing.T) {
	green := vterm.CellStyle{FG: "ansi:2"}
	screen := NativeScreenToProto("machine", corev2.NativeScreenSnapshot{
		TerminalID: "terminal",
		Rows: []corev2.NativeScreenRow{{Cells: []vterm.Cell{
			{Content: "x", Width: 1, Style: green},
			{Content: "y", Width: 1, Style: green},
			{Content: "z", Width: 1},
		}}},
	})
	cells := screen.GetRowReplacements()[0].GetRow().GetCells()
	if len(cells) != 2 || cells[0].GetContent() != "xy" || cells[0].GetWidth() != 2 || cells[1].GetContent() != "z" {
		t.Fatalf("coalesced live runs = %#v", cells)
	}
}

func TestNativeScreenToProtoOmitsWideCharacterContinuationCells(t *testing.T) {
	inputBackground := vterm.CellStyle{BG: "#222222"}
	screen := NativeScreenToProto("machine", corev2.NativeScreenSnapshot{
		TerminalID: "terminal",
		Rows: []corev2.NativeScreenRow{{Cells: []vterm.Cell{
			{Content: "现", Width: 2, Style: inputBackground},
			{Content: "", Width: 0},
			{Content: "在", Width: 2, Style: inputBackground},
			{Content: "", Width: 0},
		}}},
	})

	cells := screen.GetRowReplacements()[0].GetRow().GetCells()
	if len(cells) != 1 || cells[0].GetContent() != "现在" || cells[0].GetWidth() != 4 || cells[0].GetStyle().GetBackground() != "#222222" {
		t.Fatalf("wide live row introduced visible continuation cells: %#v", cells)
	}
}

func TestNativeScreenToProtoPreservesSparseRevisionAndRowIndexes(t *testing.T) {
	screen := NativeScreenToProto("machine", corev2.NativeScreenSnapshot{
		TerminalID:   "terminal",
		BaseRevision: 7,
		Revision:     10,
		Size:         corev2.NativeScreenSize{Cols: 80, Rows: 24},
		RowCopies:    []corev2.NativeScreenRowCopy{{SourceRow: 4, DestinationRow: 3, Count: 2}},
		Rows: []corev2.NativeScreenRow{
			{Index: 3, Cells: []vterm.Cell{{Content: "three", Width: 5}}},
			{Index: 9, Cells: []vterm.Cell{{Content: "nine", Width: 4}}},
		},
	})
	if screen.GetBaseRevision() != 7 || screen.GetLiveRevision() != 10 || screen.GetFullReplace() {
		t.Fatalf("sparse revision metadata lost: %#v", screen)
	}
	got := []int32{screen.GetRowReplacements()[0].GetRowIndex(), screen.GetRowReplacements()[1].GetRowIndex()}
	if !reflect.DeepEqual(got, []int32{3, 9}) {
		t.Fatalf("sparse row indexes lost: %#v", got)
	}
	if copies := screen.GetRowCopies(); len(copies) != 1 || copies[0].GetSourceRow() != 4 || copies[0].GetDestinationRow() != 3 || copies[0].GetCount() != 2 {
		t.Fatalf("row copy lost: %#v", copies)
	}
}

func TestHistoryWindowToProtoFitsRemotePageBudget(t *testing.T) {
	styles := []history.CellStyle{
		{FG: "ansi:1", Bold: true},
		{FG: "ansi:2"},
		{FG: "ansi:4", Underline: true},
	}
	rows := make([]history.HistoryRow, 100)
	for rowIndex := range rows {
		cells := make([]history.Cell, 49)
		for column := range cells {
			cells[column] = history.Cell{
				Text:  fmt.Sprintf("%x", (rowIndex+column)%16),
				Width: 1,
				Style: styles[(column/8+rowIndex)%len(styles)],
			}
		}
		rows[rowIndex] = history.HistoryRow{Cells: cells}
	}

	window := HistoryWindowToProto("machine", history.HistoryWindow{
		TerminalID: "terminal",
		Rows:       rows,
	})
	if size := proto.Size(window); size >= 60*1024 {
		t.Fatalf("100-row history response = %d bytes, want < 60 KiB", size)
	}
}

func TestHistorySearchScanMappingPreservesBatchContinuation(t *testing.T) {
	command := &apipb.HistorySearchCommand{
		Terminal: &apipb.TerminalRef{TerminalId: "term-search"}, Token: "frozen", Query: "needle",
		Direction: apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_FORWARD,
		Scan:      true, MaxMatches: 32,
		Start: &apipb.HistoryTextPosition{LineId: 9, Col: 4},
	}
	request := HistorySearchRequestFromProto(command)
	if !request.Scan || request.MaxMatches != 32 || request.Start.LineID != 9 || request.Start.Col != 4 {
		t.Fatalf("scan request = %#v", request)
	}

	result := HistorySearchToProto("machine", history.HistorySearchResult{
		Found: true,
		Matches: []history.HistoryCopyRange{{
			Start: history.HistoryCopyPosition{LineID: 9, Col: 7},
			End:   history.HistoryCopyPosition{LineID: 9, Col: 13},
		}},
		Next: history.HistoryCopyPosition{LineID: 9, Col: 13},
	})
	if len(result.GetScanMatches()) != 1 || result.GetScanMatches()[0].GetStartLineId() != 9 ||
		result.GetScanNext().GetLineId() != 9 || result.GetScanNext().GetCol() != 13 || result.GetScanDone() {
		t.Fatalf("scan result = %#v", result)
	}
}
