// Package history is the terminal-owned persistent-history boundary. A
// transport supplies a connection-bound Backend; Snapshot owns its frozen
// token, paging cursors and release. No shell program needs provider tokens.
package history

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anytty/anytty/proto/access/apipb"
)

// Backend is bound to one authenticated connection generation. Implementations
// must fail after that connection closes, never replay tokens on a new one.
type Backend interface {
	Window(context.Context, *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error)
	Search(context.Context, *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error)
	Copy(context.Context, *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error)
	Release(context.Context, *apipb.HistoryReleaseCommand) error
}

// Source is an optional capability of an attached PTY transport.
type Source interface {
	HistoryBackend(context.Context) (Backend, *apipb.TerminalRef, error)
}

const PageLines = 256
const MaxWindowRows = 4096

var ErrNoProgress = errors.New("terminal history: paging cursor did not advance")

// Snapshot holds only the initial tail page and the last traversed page.
// Scrolling thousands of lines does not retain thousands of pages in the UI.
// Callers serialize operations on a Snapshot, but need not lock rendering.
type Snapshot struct {
	backend                             Backend
	ref                                 *apipb.TerminalRef
	token                               string
	generation                          uint64
	Cols                                int
	latest, page                        []*apipb.HistoryRow
	bottom                              int
	oldest                              bool
	latestBases, pageBases, windowBases map[*apipb.HistoryRow]int
}

func Open(ctx context.Context, backend Backend, ref *apipb.TerminalRef, cols int) (*Snapshot, error) {
	if cols < 1 {
		return nil, errors.New("terminal history: columns must be positive")
	}
	result, err := backend.Window(ctx, &apipb.HistoryWindowCommand{
		Terminal: ref, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols: int32(cols), Limit: PageLines,
	})
	if err != nil {
		return nil, err
	}
	if result.GetToken() == "" {
		return nil, errors.New("terminal history: missing frozen token")
	}
	rows := project(result.GetRows(), cols)
	s := &Snapshot{backend: backend, ref: ref, token: result.GetToken(),
		generation: result.GetHistoryGeneration(), Cols: cols,
		latest: rows, page: rows, oldest: !result.GetHasMore()}
	s.latestBases = rowBases(s.latest, cols)
	s.pageBases = s.latestBases
	return s, nil
}

// A wide glyph can wrap before the last column. RowInLine*Cols would then
// shift every later position; count the actual cells of each logical line.
func rowBases(rows []*apipb.HistoryRow, cols int) map[*apipb.HistoryRow]int {
	bases := make(map[*apipb.HistoryRow]int, len(rows))
	base := 0
	for i, row := range rows {
		if i == 0 || rows[i-1].GetLogicalLineId() != row.GetLogicalLineId() || rows[i-1].GetSegment() != row.GetSegment() {
			base = int(row.GetRowInLine()) * cols
		}
		bases[row] = base
		for _, cell := range row.GetRow().GetCells() {
			base += max(0, int(cell.GetWidth()))
		}
	}
	return bases
}

func (s *Snapshot) ColumnBase(row *apipb.HistoryRow) int {
	for _, bases := range []map[*apipb.HistoryRow]int{s.windowBases, s.pageBases, s.latestBases} {
		if base, ok := bases[row]; ok {
			return base
		}
	}
	return int(row.GetRowInLine()) * s.Cols
}

func cursor(row *apipb.HistoryRow) *apipb.HistoryCursor {
	return &apipb.HistoryCursor{LineId: row.GetLogicalLineId(), RowInLine: row.GetRowInLine(), Segment: row.GetSegment()}
}

func sameRow(a, b *apipb.HistoryRow) bool {
	return a.GetLogicalLineId() == b.GetLogicalLineId() && a.GetRowInLine() == b.GetRowInLine() && a.GetSegment() == b.GetSegment()
}

func (s *Snapshot) request(mode apipb.HistoryWindowMode) *apipb.HistoryWindowCommand {
	return &apipb.HistoryWindowCommand{Terminal: s.ref, Mode: mode, Cols: int32(s.Cols),
		Limit: PageLines, Token: s.token, HistoryGeneration: s.generation}
}

func (s *Snapshot) older(ctx context.Context) error {
	if s.oldest {
		return nil
	}
	if len(s.page) == 0 {
		return ErrNoProgress
	}
	first := s.page[0]
	req := s.request(apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER)
	req.BeforeCursor = cursor(first)
	result, err := s.backend.Window(ctx, req)
	if err != nil {
		return err
	}
	rows := project(result.GetRows(), s.Cols)
	// Some providers return the anchor's whole logical line. Remove the
	// overlap at the boundary rather than duplicating its visual rows.
	for i, row := range rows {
		if sameRow(row, first) {
			rows = rows[:i]
			break
		}
	}
	if len(rows) == 0 {
		if result.GetHasMore() {
			return ErrNoProgress
		}
		s.oldest = true
		return nil
	}
	if sameRow(rows[0], first) {
		return ErrNoProgress
	}
	s.bottom += len(s.page)
	s.page = rows
	s.pageBases = rowBases(rows, s.Cols)
	s.oldest = !result.GetHasMore()
	return nil
}

// Window reads rows at a visual-row distance from the frozen tail. It pages
// through provider history with cancellation and bounded memory. At the
// oldest boundary it clamps to the first full visible window.
func (s *Snapshot) Window(ctx context.Context, offset, height int) ([]*apipb.HistoryRow, int, error) {
	if offset < 0 {
		offset = 0
	}
	if height < 1 || height > MaxWindowRows {
		return nil, 0, fmt.Errorf("terminal history: rows must be 1..%d", MaxWindowRows)
	}
	if offset < s.bottom {
		s.page, s.bottom, s.oldest = s.latest, 0, false
		s.pageBases = s.latestBases
	}
	var out []*apipb.HistoryRow
	bases := make(map[*apipb.HistoryRow]int)
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		for i := len(s.page) - 1; i >= 0; i-- {
			distance := s.bottom + len(s.page) - 1 - i
			if distance >= offset && distance < offset+height {
				out = append(out, s.page[i])
				bases[s.page[i]] = s.pageBases[s.page[i]]
			}
		}
		if s.bottom+len(s.page) >= offset+height {
			break
		}
		if s.oldest {
			maxOffset := max(0, s.bottom+len(s.page)-height)
			if offset > maxOffset {
				return s.Window(ctx, maxOffset, height)
			}
			break
		}
		before := s.bottom
		if err := s.older(ctx); err != nil {
			return nil, 0, err
		}
		if s.bottom == before && s.oldest {
			maxOffset := max(0, s.bottom+len(s.page)-height)
			if offset > maxOffset {
				return s.Window(ctx, maxOffset, height)
			}
			break
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	s.windowBases = bases
	return out, offset, nil
}

// Search executes the provider's text/glob/regexp search on the frozen token.
// The returned match uses logical line IDs, so searching never requires
// downloading the entire history into a layout program.
func (s *Snapshot) Search(ctx context.Context, query string, mode apipb.HistorySearchMode, backward bool, start *apipb.HistoryTextPosition) (*apipb.HistorySearchResult, error) {
	direction := apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_FORWARD
	if backward {
		direction = apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_BACKWARD
	}
	return s.backend.Search(ctx, &apipb.HistorySearchCommand{Terminal: s.ref, Token: s.token,
		HistoryGeneration: s.generation, Query: query, Mode: mode, Direction: direction,
		Cols: int32(s.Cols), Limit: PageLines, Start: start})
}

// Locate maps a provider logical match into a viewport. Traversal keeps only
// a page at a time, even when the result predates the client's attachment.
func (s *Snapshot) Locate(ctx context.Context, match *apipb.HistoryRange, height int) ([]*apipb.HistoryRow, int, int, int, error) {
	s.page, s.bottom, s.oldest = s.latest, 0, false
	s.pageBases = s.latestBases
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, 0, 0, err
		}
		for i, row := range s.page {
			col := int(match.GetStartCol()) - s.pageBases[row]
			width := 0
			for _, cell := range row.GetRow().GetCells() {
				width += max(0, int(cell.GetWidth()))
			}
			if row.GetLogicalLineId() != match.GetStartLineId() || col < 0 || col >= width {
				continue
			}
			distance := s.bottom + len(s.page) - 1 - i
			rows, offset, err := s.Window(ctx, max(0, distance-height/2), height)
			if err != nil {
				return nil, 0, 0, 0, err
			}
			start, end := -1, -1
			for j, r := range rows {
				base := s.ColumnBase(r)
				if sameRow(r, row) {
					start = j*s.Cols + col
				}
				if r.GetLogicalLineId() == match.GetEndLineId() && int(match.GetEndCol()) > base && int(match.GetEndCol()) <= base+s.Cols {
					end = j*s.Cols + int(match.GetEndCol()) - base
				}
			}
			if start < 0 {
				return nil, 0, 0, 0, ErrNoProgress
			}
			if end < start {
				end = len(rows) * s.Cols
			}
			return rows, offset, start, end, nil
		}
		if s.oldest {
			return nil, 0, 0, 0, errors.New("terminal history: match is outside the frozen snapshot")
		}
		if err := s.older(ctx); err != nil {
			return nil, 0, 0, 0, err
		}
	}
}

// Copy streams logical-line text from the provider, preserving its soft-wrap
// semantics. Returned continuations are followed with the same frozen token.
func (s *Snapshot) Copy(ctx context.Context, selected *apipb.HistoryRange) (string, error) {
	req := &apipb.HistoryCopyCommand{Terminal: s.ref, Window: s.request(apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST), MaxLines: PageLines, MaxBytes: 512 << 10}
	req.Window.Range = selected
	var text strings.Builder
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		result, err := s.backend.Copy(ctx, req)
		if err != nil {
			return "", err
		}
		text.WriteString(result.GetText())
		if text.Len() > 16<<20 {
			return "", errors.New("terminal history: clipboard selection exceeds 16 MiB")
		}
		if result.GetDone() {
			return text.String(), nil
		}
		next := result.GetNext()
		if next == nil || next.GetLineId() < req.Window.Range.GetStartLineId() ||
			(next.GetLineId() == req.Window.Range.GetStartLineId() && next.GetCol() <= req.Window.Range.GetStartCol()) {
			return "", ErrNoProgress
		}
		req.Window.Range = &apipb.HistoryRange{StartLineId: next.GetLineId(), StartCol: next.GetCol(), EndLineId: selected.GetEndLineId(), EndCol: selected.GetEndCol()}
	}
}

func (s *Snapshot) Close(ctx context.Context) error {
	if s.token == "" {
		return nil
	}
	token := s.token
	s.token = ""
	return s.backend.Release(ctx, &apipb.HistoryReleaseCommand{Terminal: s.ref, Token: token, HistoryGeneration: s.generation})
}

func Text(rows []*apipb.HistoryRow) []string {
	text := make([]string, len(rows))
	for i, row := range rows {
		var b strings.Builder
		for _, cell := range row.GetRow().GetCells() {
			b.WriteString(cell.GetContent())
		}
		text[i] = strings.TrimRight(b.String(), " ")
	}
	return text
}
