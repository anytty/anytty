package frameaudit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	localvterm "github.com/anytty/anytty/vterm/vterm"
)

var dumpHeaderPattern = regexp.MustCompile(`^--- ([a-z_]+) (.+) len=([0-9]+) ---$`)

type DumpEntry struct {
	Index     int
	Kind      string
	Timestamp string
	Payload   []byte
}

type EntryStats struct {
	Index         int
	Kind          string
	Bytes         int
	ScreenChanged bool
	CursorChanged bool
	Noop          bool
	ChangedRows   int
	ChangedCells  int
}

type Summary struct {
	Entries              int
	Bytes                int
	Noops                int
	NoopBytes            int
	ScreenChangedEntries int
	CursorOnlyEntries    int
	ChangedRows          int
	ChangedCells         int
	ByKind               map[string]KindSummary
}

type KindSummary struct {
	Entries              int
	Bytes                int
	Noops                int
	NoopBytes            int
	ScreenChangedEntries int
	CursorOnlyEntries    int
	ChangedRows          int
	ChangedCells         int
}

type Report struct {
	Width           int
	Height          int
	SuggestedWidth  int
	SuggestedHeight int
	Entries         []EntryStats
	Summary         Summary
}

func ParseDump(data []byte) ([]DumpEntry, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	var entries []DumpEntry
	for {
		header, err := reader.ReadString('\n')
		if err != nil {
			if len(strings.TrimSpace(header)) == 0 {
				return entries, nil
			}
			return nil, fmt.Errorf("read dump header: %w", err)
		}
		header = strings.TrimSuffix(header, "\n")
		header = strings.TrimSuffix(header, "\r")
		if strings.TrimSpace(header) == "" {
			continue
		}
		matches := dumpHeaderPattern.FindStringSubmatch(header)
		if matches == nil {
			return nil, fmt.Errorf("invalid dump header %q", header)
		}
		length, err := strconv.Atoi(matches[3])
		if err != nil || length < 0 {
			return nil, fmt.Errorf("invalid dump payload length %q", matches[3])
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, fmt.Errorf("read dump payload for entry %d: %w", len(entries)+1, err)
		}
		separator, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("read dump payload separator for entry %d: %w", len(entries)+1, err)
		}
		if separator != '\n' {
			return nil, fmt.Errorf("expected newline after payload for entry %d, got %#x", len(entries)+1, separator)
		}
		entries = append(entries, DumpEntry{
			Index:     len(entries) + 1,
			Kind:      matches[1],
			Timestamp: matches[2],
			Payload:   payload,
		})
	}
}

func AuditEntries(entries []DumpEntry, width, height int) (Report, error) {
	if width <= 0 || height <= 0 {
		return Report{}, fmt.Errorf("invalid terminal size %dx%d", width, height)
	}
	suggestedWidth, suggestedHeight := SuggestedReplaySize(entries)
	vt := localvterm.New(width, height, 0, nil)
	report := Report{
		Width:           width,
		Height:          height,
		SuggestedWidth:  suggestedWidth,
		SuggestedHeight: suggestedHeight,
		Summary: Summary{
			ByKind: make(map[string]KindSummary),
		},
	}
	for _, entry := range entries {
		beforeScreen := vt.ScreenContent()
		beforeCursor := vt.CursorState()
		if _, err := vt.Write(entry.Payload); err != nil {
			return Report{}, fmt.Errorf("replay dump entry %d (%s): %w", entry.Index, entry.Kind, err)
		}
		afterScreen := vt.ScreenContent()
		afterCursor := vt.CursorState()
		changedRows, changedCells := diffScreen(beforeScreen, afterScreen)
		screenChanged := changedRows > 0
		cursorChanged := beforeCursor != afterCursor
		stats := EntryStats{
			Index:         entry.Index,
			Kind:          entry.Kind,
			Bytes:         len(entry.Payload),
			ScreenChanged: screenChanged,
			CursorChanged: cursorChanged,
			Noop:          !screenChanged && !cursorChanged,
			ChangedRows:   changedRows,
			ChangedCells:  changedCells,
		}
		report.Entries = append(report.Entries, stats)
		accumulateSummary(&report.Summary, stats)
	}
	return report, nil
}

func SuggestedReplaySize(entries []DumpEntry) (int, int) {
	maxCol := 0
	maxRow := 0
	for _, entry := range entries {
		inspectPayloadBounds(entry.Payload, &maxCol, &maxRow)
	}
	if maxCol <= 0 {
		maxCol = 1
	}
	if maxRow <= 0 {
		maxRow = 1
	}
	return maxCol, maxRow
}

func diffScreen(before, after localvterm.ScreenData) (int, int) {
	maxRows := len(before.Cells)
	if len(after.Cells) > maxRows {
		maxRows = len(after.Cells)
	}
	changedRows := 0
	changedCells := 0
	for row := 0; row < maxRows; row++ {
		beforeRow := screenRow(before, row)
		afterRow := screenRow(after, row)
		maxCols := len(beforeRow)
		if len(afterRow) > maxCols {
			maxCols = len(afterRow)
		}
		rowChanged := false
		for col := 0; col < maxCols; col++ {
			if screenCell(beforeRow, col) != screenCell(afterRow, col) {
				rowChanged = true
				changedCells++
			}
		}
		if rowChanged {
			changedRows++
		}
	}
	return changedRows, changedCells
}

func accumulateSummary(summary *Summary, stats EntryStats) {
	if summary == nil {
		return
	}
	summary.Entries++
	summary.Bytes += stats.Bytes
	summary.ChangedRows += stats.ChangedRows
	summary.ChangedCells += stats.ChangedCells
	if stats.Noop {
		summary.Noops++
		summary.NoopBytes += stats.Bytes
	}
	if stats.ScreenChanged {
		summary.ScreenChangedEntries++
	} else if stats.CursorChanged {
		summary.CursorOnlyEntries++
	}
	kind := summary.ByKind[stats.Kind]
	kind.Entries++
	kind.Bytes += stats.Bytes
	kind.ChangedRows += stats.ChangedRows
	kind.ChangedCells += stats.ChangedCells
	if stats.Noop {
		kind.Noops++
		kind.NoopBytes += stats.Bytes
	}
	if stats.ScreenChanged {
		kind.ScreenChangedEntries++
	} else if stats.CursorChanged {
		kind.CursorOnlyEntries++
	}
	summary.ByKind[stats.Kind] = kind
}

func screenRow(screen localvterm.ScreenData, row int) []localvterm.Cell {
	if row < 0 || row >= len(screen.Cells) {
		return nil
	}
	return screen.Cells[row]
}

func screenCell(row []localvterm.Cell, col int) localvterm.Cell {
	if col < 0 || col >= len(row) {
		return localvterm.Cell{}
	}
	return row[col]
}

func inspectPayloadBounds(payload []byte, maxCol, maxRow *int) {
	for i := 0; i < len(payload); i++ {
		if payload[i] != '\x1b' || i+2 >= len(payload) || payload[i+1] != '[' {
			continue
		}
		j := i + 2
		for j < len(payload) && payload[j] >= '0' && payload[j] <= '9' {
			j++
		}
		if j >= len(payload) {
			return
		}
		if payload[j] == ';' {
			rowStart := i + 2
			colStart := j + 1
			k := colStart
			for k < len(payload) && payload[k] >= '0' && payload[k] <= '9' {
				k++
			}
			if k < len(payload) && payload[k] == 'H' {
				row := parsePositiveInt(payload[rowStart:j])
				col := parsePositiveInt(payload[colStart:k])
				if row > 0 && maxRow != nil && row > *maxRow {
					*maxRow = row
				}
				if col > 0 && maxCol != nil && col > *maxCol {
					*maxCol = col
				}
				i = k
				continue
			}
		}
		if payload[j] == 'G' {
			col := parsePositiveInt(payload[i+2 : j])
			if col > 0 && maxCol != nil && col > *maxCol {
				*maxCol = col
			}
			i = j
		}
	}
}

func parsePositiveInt(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	value := 0
	for _, b := range raw {
		if b < '0' || b > '9' {
			return 0
		}
		value = value*10 + int(b-'0')
	}
	return value
}
