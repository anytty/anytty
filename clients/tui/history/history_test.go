package history

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/access/apipb"
)

type fixtureBackend struct {
	rows     []*apipb.HistoryRow
	released bool
	calls    int
}

func fixture(count int) *fixtureBackend {
	b := &fixtureBackend{}
	for i := 1; i <= count; i++ {
		b.rows = append(b.rows, &apipb.HistoryRow{LogicalLineId: uint64(i), Row: &apipb.ScreenRow{
			Cells: []*apipb.ScreenCell{{Content: fmt.Sprintf("line-%05d", i), Width: 10}},
		}})
	}
	return b
}

func (b *fixtureBackend) Window(ctx context.Context, req *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b.calls++
	end := len(b.rows)
	if req.GetMode() == apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER {
		if req.GetToken() != "frozen" || req.GetHistoryGeneration() != 7 {
			return nil, fmt.Errorf("wrong snapshot")
		}
		end = int(req.GetBeforeCursor().GetLineId()) - 1
	}
	start := max(0, end-int(req.GetLimit()))
	return &apipb.HistoryWindowResult{Token: "frozen", HistoryGeneration: 7, Rows: b.rows[start:end], HasMore: start > 0}, nil
}

func (b *fixtureBackend) Search(ctx context.Context, req *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	if req.GetToken() != "frozen" || req.GetQuery() != "line-00001" {
		return nil, fmt.Errorf("wrong search")
	}
	return &apipb.HistorySearchResult{Found: true, Match: &apipb.HistoryRange{StartLineId: 1, EndLineId: 1, EndCol: 10}}, nil
}

func (b *fixtureBackend) Copy(ctx context.Context, req *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	return &apipb.HistoryCopyResult{Done: true, Text: "copied"}, nil
}

func (b *fixtureBackend) Release(ctx context.Context, req *apipb.HistoryReleaseCommand) error {
	if req.GetToken() != "frozen" || req.GetHistoryGeneration() != 7 {
		return fmt.Errorf("wrong release")
	}
	b.released = true
	return nil
}

func TestSnapshotPagesBeyondParserCacheWithBoundedPages(t *testing.T) {
	b := fixture(12000)
	s, err := Open(context.Background(), b, &apipb.TerminalRef{TerminalId: "old"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{0, 510, 7000, 7100, 20, 11980, 1000000, 0} {
		rows, applied, err := s.Window(context.Background(), offset, 20)
		if err != nil {
			t.Fatalf("offset %d: %v", offset, err)
		}
		want := min(offset, 11980)
		if applied != want || len(rows) != 20 {
			t.Fatalf("offset=%d applied=%d rows=%d", offset, applied, len(rows))
		}
		if got := rows[0].GetLogicalLineId(); got != uint64(12000-want-19) {
			t.Fatalf("offset=%d first row=%d", offset, got)
		}
		if len(s.page) > PageLines || len(s.latest) > PageLines {
			t.Fatal("paging retained the entire history")
		}
	}
	result, err := s.Search(context.Background(), "line-00001", apipb.HistorySearchMode_HISTORY_SEARCH_MODE_TEXT, false, nil)
	if err != nil || !result.GetFound() {
		t.Fatalf("search=%v err=%v", result, err)
	}
	if err := s.Close(context.Background()); err != nil || !b.released {
		t.Fatalf("release: %v", err)
	}
}

func TestSnapshotCancelledPaging(t *testing.T) {
	b := fixture(10000)
	s, err := Open(context.Background(), b, &apipb.TerminalRef{TerminalId: "old"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.Window(ctx, 9000, 20); err != context.Canceled {
		t.Fatalf("error=%v", err)
	}
	if b.calls != 1 {
		t.Fatal("cancelled request performed I/O")
	}
}

func TestSnapshotSmallHistoryClampsOnce(t *testing.T) {
	s, err := Open(context.Background(), fixture(3), &apipb.TerminalRef{TerminalId: "small"}, 80)
	if err != nil {
		t.Fatal(err)
	}
	rows, offset, err := s.Window(context.Background(), 1000, 24)
	if err != nil || offset != 0 || len(rows) != 3 {
		t.Fatalf("rows=%v offset=%d err=%v", Text(rows), offset, err)
	}
	if !strings.Contains(Text(rows)[0], "00001") {
		t.Fatal("oldest row missing")
	}
}
