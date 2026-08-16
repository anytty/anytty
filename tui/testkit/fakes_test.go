package testkit

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/tui/input"
	. "github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestRequestIDValid(t *testing.T) {
	if !RequestID(1).Valid() {
		t.Fatal("expected non-zero request id to be valid")
	}
	if RequestID(0).Valid() {
		t.Fatal("expected zero request id to be invalid")
	}
}

func TestFakeCoreClientRecordsHistoryRequests(t *testing.T) {
	client := &FakeCoreClient{
		LatestResponses: []HistoryResult{{Window: state.HistoryWindow{Token: "latest"}}},
		OlderResponses:  []HistoryResult{{Window: state.HistoryWindow{Token: "older"}}},
		NewerResponses:  []HistoryResult{{Window: state.HistoryWindow{Token: "newer"}}},
		OldestResponses: []HistoryResult{{Window: state.HistoryWindow{Token: "oldest"}}},
		CopyResponses:   []HistoryCopyRangeResult{{Text: "copied"}},
	}

	latest, err := client.HistoryLatest(context.Background(), HistoryLatestRequest{RequestID: 1, TerminalID: "term-1", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	older, err := client.HistoryOlder(context.Background(), HistoryOlderRequest{
		RequestID:  2,
		TerminalID: "term-1",
		Cols:       80,
		Rows:       10,
		Token:      "latest",
		Generation: 7,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 10},
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 20},
	})
	if err != nil {
		t.Fatalf("older: %v", err)
	}
	newer, err := client.HistoryNewer(context.Background(), HistoryNewerRequest{
		RequestID:  3,
		TerminalID: "term-1",
		Cols:       80,
		Rows:       10,
		Token:      "latest",
		Generation: 7,
		Cursor:     state.HistoryCursor{Valid: true, BeforeLineID: 20},
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 30},
	})
	if err != nil {
		t.Fatalf("newer: %v", err)
	}
	oldest, err := client.HistoryOldest(context.Background(), HistoryOldestRequest{
		RequestID:  4,
		TerminalID: "term-1",
		Cols:       80,
		Rows:       10,
		Token:      "latest",
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 20},
	})
	if err != nil {
		t.Fatalf("oldest: %v", err)
	}
	if err := client.ReleaseHistory(context.Background(), HistoryReleaseRequest{TerminalID: "term-1", Token: "latest"}); err != nil {
		t.Fatalf("release history: %v", err)
	}
	copied, err := client.HistoryCopyRange(context.Background(), HistoryCopyRangeRequest{
		TerminalID: "term-1",
		Cols:       80,
		Token:      "latest",
		Generation: 7,
		Boundary:   state.HistoryBoundary{FirstLineID: 10, LastLineID: 20},
		Start:      state.CopyLogicalPosition{Valid: true, LineID: 10, Col: 2},
		End:        state.CopyLogicalPosition{Valid: true, LineID: 20, Col: 4},
	})
	if err != nil {
		t.Fatalf("copy range: %v", err)
	}

	if latest.RequestID != 1 || latest.Window.Token != "latest" {
		t.Fatalf("unexpected latest result %#v", latest)
	}
	if older.RequestID != 2 || older.Window.Token != "older" {
		t.Fatalf("unexpected older result %#v", older)
	}
	if newer.RequestID != 3 || newer.Window.Token != "newer" {
		t.Fatalf("unexpected newer result %#v", newer)
	}
	if oldest.RequestID != 4 || oldest.Window.Token != "oldest" {
		t.Fatalf("unexpected oldest result %#v", oldest)
	}
	if len(client.LatestRequests) != 1 || client.LatestRequests[0].TerminalID != "term-1" {
		t.Fatalf("unexpected latest requests %#v", client.LatestRequests)
	}
	if len(client.OlderRequests) != 1 || client.OlderRequests[0].Token != "latest" {
		t.Fatalf("unexpected older requests %#v", client.OlderRequests)
	}
	if len(client.NewerRequests) != 1 || client.NewerRequests[0].Boundary.LastLineID != 30 {
		t.Fatalf("unexpected newer requests %#v", client.NewerRequests)
	}
	if len(client.OldestRequests) != 1 || client.OldestRequests[0].Boundary.LastLineID != 20 {
		t.Fatalf("unexpected oldest requests %#v", client.OldestRequests)
	}
	if len(client.ReleaseRequests) != 1 || client.ReleaseRequests[0].Token != "latest" {
		t.Fatalf("unexpected release requests %#v", client.ReleaseRequests)
	}
	if copied.Text != "copied" || len(client.CopyRequests) != 1 || client.CopyRequests[0].Start.LineID != 10 || client.CopyRequests[0].End.Col != 4 {
		t.Fatalf("unexpected copy range result=%#v requests=%#v", copied, client.CopyRequests)
	}
}

func TestFakeCoreClientMissingResponse(t *testing.T) {
	client := &FakeCoreClient{}
	if _, err := client.HistoryLatest(context.Background(), HistoryLatestRequest{RequestID: 1}); !errors.Is(err, ErrMissingHistoryResponse) {
		t.Fatalf("expected ErrMissingHistoryResponse, got %v", err)
	}
}

func TestFakeTerminalClipboardServices(t *testing.T) {
	terminal := &FakeTerminalService{AttachResult: TerminalAttachResult{Channel: 3}}
	attached, err := terminal.Attach(context.Background(), TerminalAttachRequest{TerminalID: "term-1", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if attached.TerminalID != "term-1" || attached.Channel != 3 || attached.Cols != 80 {
		t.Fatalf("unexpected attach result %#v", attached)
	}
	if err := terminal.SendInput(context.Background(), TerminalInputRequest{
		TerminalID: "term-1",
		Channel:    3,
		Event:      input.InputEvent{Kind: input.EventKindKey, Key: input.KeyChar, Char: "x"},
		Bytes:      []byte("x"),
	}); err != nil {
		t.Fatalf("send input: %v", err)
	}
	if _, err := terminal.Resize(context.Background(), TerminalResizeRequest{TerminalID: "term-1", Channel: 3, Cols: 100, Rows: 40}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	if len(terminal.Inputs) != 1 || string(terminal.Inputs[0].Bytes) != "x" {
		t.Fatalf("unexpected terminal inputs %#v", terminal.Inputs)
	}
	if len(terminal.Resizes) != 1 || terminal.Resizes[0].Cols != 100 {
		t.Fatalf("unexpected terminal resizes %#v", terminal.Resizes)
	}

	clipboard := &FakeClipboardService{}
	if err := clipboard.Write(context.Background(), ClipboardWriteRequest{Text: "copy"}); err != nil {
		t.Fatalf("write clipboard: %v", err)
	}
	if len(clipboard.Writes) != 1 || clipboard.Writes[0].Text != "copy" {
		t.Fatalf("unexpected clipboard writes %#v", clipboard.Writes)
	}
	if last := clipboard.LastCopy(); last != "copy" {
		t.Fatalf("unexpected last copy %q", last)
	}
	clipboard.ReadResult = ClipboardReadResult{Text: "clip"}
	read, err := clipboard.Read(context.Background())
	if err != nil {
		t.Fatalf("read clipboard: %v", err)
	}
	if read.Text != "clip" {
		t.Fatalf("unexpected clipboard read %#v", read)
	}
}
