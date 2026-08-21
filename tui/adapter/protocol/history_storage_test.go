package protocoladapter

import (
	"context"
	"errors"
	"testing"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

func TestProtocolCoreClientAdapterUsesGeneratedHistoryWindow(t *testing.T) {
	executor := &recordingProtoExecutor{}
	executor.handle = func(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
		return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowResult{
			Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-1"}, Token: "token-1",
			Operation: apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_REPLACE,
			Size:      &apipb.TerminalSize{Cols: 80, Rows: 24}, HistoryGeneration: 7, FirstLineId: 41, LastLineId: 41,
			ViewportAnchor: &apipb.HistoryViewportAnchor{TopLineId: 41, TopCellOffset: 9, ScreenCols: 80, ScreenRows: 24},
			Rows:           []*apipb.HistoryRow{{LogicalLineId: 41, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "hello", Width: 5}}}}},
		}}})
	}
	adapter := ProtocolCoreClientAdapter{Application: newProtoTestApplication(t, executor)}

	result, err := adapter.HistoryLatest(context.Background(), port.HistoryLatestRequest{RequestID: 9, EndpointID: "local", TerminalID: "term-1", Rows: 20, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestID != 9 || result.Window.Token != "token-1" || len(result.Window.Rows) != 1 || result.Window.Rows[0].Text != "hello" {
		t.Fatalf("unexpected history result %#v", result)
	}
	if anchor := result.Window.ViewportAnchor; !anchor.Valid || anchor.TopLineID != 41 || anchor.TopCellOffset != 9 || anchor.ScreenRows != 24 {
		t.Fatalf("unexpected viewport anchor %#v", anchor)
	}
	if got := executor.commands[0].GetHistoryWindow(); got.GetTerminal().GetTerminalId() != "term-1" || got.GetLimit() != 20 {
		t.Fatalf("unexpected history command %#v", got)
	}
}

func TestProtocolCoreClientAdapterReleasesLateCancelledLatestToken(t *testing.T) {
	executor := &recordingProtoExecutor{}
	executor.handle = func(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
		if command.GetHistoryRelease() != nil {
			return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}})
		}
		return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowResult{
			Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-1"}, Token: "late-token",
			Operation: apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_REPLACE,
		}}})
	}
	adapter := ProtocolCoreClientAdapter{Application: newProtoTestApplication(t, executor)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := adapter.HistoryLatest(ctx, port.HistoryLatestRequest{RequestID: 9, EndpointID: "local", TerminalID: "term-1", Rows: 20, Cols: 80})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled latest error = %v, want context canceled", err)
	}
	if len(executor.commands) != 2 || executor.commands[0].GetHistoryWindow() == nil {
		t.Fatalf("late latest commands = %#v", executor.commands)
	}
	release := executor.commands[1].GetHistoryRelease()
	if release.GetToken() != "late-token" || release.GetTerminal().GetTerminalId() != "term-1" {
		t.Fatalf("late latest token was not released: %#v", release)
	}
}

func TestHistoryCursorUsesLogicalCoordinates(t *testing.T) {
	want := state.HistoryCursor{
		Valid:           true,
		BeforeLineID:    42,
		BeforeRowInLine: 3,
		Segment:         state.HistoryCursorSegmentArchivedPrimaryFrame,
	}
	wire := historyCursorToProto(want)
	window := historyWindowFromProto(&apipb.HistoryWindowResult{
		Terminal: &apipb.TerminalRef{TerminalId: "term-1"},
		Size:     &apipb.TerminalSize{Cols: 80},
		Cursor:   wire,
	}, 80)
	if window.Cursor != want {
		t.Fatalf("history cursor round trip = %#v, want %#v", window.Cursor, want)
	}
}

func TestProtocolCoreClientAdapterUsesGeneratedCopyAndRelease(t *testing.T) {
	executor := &recordingProtoExecutor{}
	executor.handle = func(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
		switch command.Command.(type) {
		case *apipb.CommandEnvelope_HistoryCopy:
			return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryCopy{HistoryCopy: &apipb.HistoryCopyResult{Text: "copied", Done: true}}})
		default:
			return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}})
		}
	}
	adapter := ProtocolCoreClientAdapter{Application: newProtoTestApplication(t, executor)}
	copyResult, err := adapter.HistoryCopyRange(context.Background(), port.HistoryCopyRangeRequest{
		EndpointID: "local", TerminalID: "term-1", Token: "token-1", Generation: 7, Cols: 80,
		Start: state.CopyLogicalPosition{Valid: true, LineID: 41, Col: 1}, End: state.CopyLogicalPosition{Valid: true, LineID: 42, Col: 2},
	})
	if err != nil || copyResult.Text != "copied" {
		t.Fatalf("copy=%#v err=%v", copyResult, err)
	}
	if err := adapter.ReleaseHistory(context.Background(), port.HistoryReleaseRequest{EndpointID: "local", TerminalID: "term-1", Token: "token-1"}); err != nil {
		t.Fatal(err)
	}
	if len(executor.commands) != 2 || executor.commands[0].GetHistoryCopy() == nil || executor.commands[1].GetHistoryRelease() == nil {
		t.Fatalf("unexpected commands %#v", executor.commands)
	}
}

func TestProtocolCoreClientAdapterJoinsCopyChunksAtLineBoundary(t *testing.T) {
	executor := &recordingProtoExecutor{}
	executor.handle = func(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
		copyCommand := command.GetHistoryCopy()
		if copyCommand.GetWindow().GetRange().GetStartLineId() == 1 {
			return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryCopy{HistoryCopy: &apipb.HistoryCopyResult{
				Text: "one\ntwo", Next: &apipb.HistoryTextPosition{LineId: 3},
			}}})
		}
		return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryCopy{HistoryCopy: &apipb.HistoryCopyResult{Text: "three", Done: true}}})
	}
	adapter := ProtocolCoreClientAdapter{Application: newProtoTestApplication(t, executor)}
	result, err := adapter.HistoryCopyRange(context.Background(), port.HistoryCopyRangeRequest{
		EndpointID: "local", TerminalID: "term-1", Token: "token-1", Cols: 80,
		Start: state.CopyLogicalPosition{Valid: true, LineID: 1},
		End:   state.CopyLogicalPosition{Valid: true, LineID: 3, Col: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "one\ntwo\nthree" || len(executor.commands) != 2 {
		t.Fatalf("chunked copy=%q commands=%d", result.Text, len(executor.commands))
	}
}

func TestProtocolCoreClientAdapterMapsHistorySearch(t *testing.T) {
	executor := &recordingProtoExecutor{}
	executor.handle = func(command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
		return protoTestResult(command, &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistorySearch{HistorySearch: &apipb.HistorySearchResult{
			Found: true,
			Match: &apipb.HistoryRange{StartLineId: 41, StartCol: 2, EndLineId: 41, EndCol: 8},
			Window: &apipb.HistoryWindowResult{
				Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-1"}, Token: "token-1",
				Operation: apipb.HistoryWindowOperation_HISTORY_WINDOW_OPERATION_REPLACE,
				Size:      &apipb.TerminalSize{Cols: 80}, HistoryGeneration: 7,
				Rows: []*apipb.HistoryRow{{LogicalLineId: 41, Row: &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: "a needle", Width: 8}}}}},
			},
		}}})
	}
	adapter := ProtocolCoreClientAdapter{Application: newProtoTestApplication(t, executor)}
	result, err := adapter.HistorySearch(context.Background(), port.HistorySearchRequest{
		RequestID: 9, EndpointID: "local", TerminalID: "term-1", Token: "token-1", Cols: 80, Rows: 20,
		Query: "needle.*", Mode: state.HistorySearchModeRegex, Direction: port.HistorySearchForward,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Found || result.Start.LineID != 41 || result.Start.Col != 2 || result.Window.Token != "token-1" {
		t.Fatalf("search result=%#v", result)
	}
	command := executor.commands[len(executor.commands)-1].GetHistorySearch()
	if command.GetMode() != apipb.HistorySearchMode_HISTORY_SEARCH_MODE_REGEX || command.GetQuery() != "needle.*" {
		t.Fatalf("search command mode=%v query=%q", command.GetMode(), command.GetQuery())
	}
}

func TestProtocolCoreClientAdapterNormalizesStaleHistoryError(t *testing.T) {
	err := normalizeProtocolHistoryWindowError(&clientruntime.Error{Code: clientruntime.ErrorStaleResource, Message: "generation changed"})
	if !errors.Is(err, port.ErrStaleHistoryWindow) {
		t.Fatalf("stale error = %v", err)
	}
	messageOnly := errors.New("stale history window: generation changed")
	if normalized := normalizeProtocolHistoryWindowError(messageOnly); !errors.Is(normalized, messageOnly) || errors.Is(normalized, port.ErrStaleHistoryWindow) {
		t.Fatalf("display text must not classify stale history: %v", normalized)
	}
}

func TestProtocolCoreClientAdapterClassifiesResourceExhaustionByOperation(t *testing.T) {
	nonRetryable := &clientruntime.Error{Code: clientruntime.ErrorResourceExhausted, Message: "bounded response exceeded"}
	if err := normalizeProtocolHistoryWindowError(nonRetryable); !errors.Is(err, port.ErrHistoryWindowTooLarge) || errors.Is(err, port.ErrHistoryCopyTooLarge) {
		t.Fatalf("window resource exhaustion = %v", err)
	}
	if err := normalizeProtocolHistoryCopyError(nonRetryable); !errors.Is(err, port.ErrHistoryCopyTooLarge) || errors.Is(err, port.ErrHistoryWindowTooLarge) {
		t.Fatalf("copy resource exhaustion = %v", err)
	}

	retryable := &clientruntime.Error{Code: clientruntime.ErrorResourceExhausted, Message: "session quota", Retryable: true}
	for _, err := range []error{normalizeProtocolHistoryWindowError(retryable), normalizeProtocolHistoryCopyError(retryable)} {
		if !errors.Is(err, port.ErrHistoryResourceExhausted) || errors.Is(err, port.ErrHistoryWindowTooLarge) || errors.Is(err, port.ErrHistoryCopyTooLarge) {
			t.Fatalf("retryable resource exhaustion = %v", err)
		}
	}
}

type recordingProtoExecutor struct {
	commands []*apipb.CommandEnvelope
	handle   func(*apipb.CommandEnvelope) *apipb.ResultEnvelope
}

func (executor *recordingProtoExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	executor.commands = append(executor.commands, command)
	return executor.handle(command), nil
}

func (executor *recordingProtoExecutor) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return executor.ExecuteApplication(ctx, command)
}

func newProtoTestApplication(t *testing.T, executor *recordingProtoExecutor) *clientruntime.ApplicationSession {
	t.Helper()
	session, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{EndpointID: "local", RouteID: "test", Generation: 1}, executor)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func protoTestResult(command *apipb.CommandEnvelope, result *apipb.ResultEnvelope) *apipb.ResultEnvelope {
	result.RequestId = command.GetContext().GetRequestId()
	result.OriginSession = command.GetContext().GetSession()
	return result
}
