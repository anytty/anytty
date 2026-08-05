package core

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestProtocolHistoryInvalidCopyKeepsOwnerQuotaAndTokenUsable(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 1}, "term-1")
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	window := applicationLatestHistory(t, session, "term-1")

	_, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{
		TerminalID: "term-1",
		Token:      window.Token,
		Range: &history.HistoryCopyRange{
			Start: history.HistoryCopyPosition{LineID: 1},
			End:   history.HistoryCopyPosition{LineID: history.LogicalLineID(window.LogicalTotal + 1)},
		},
	})
	if !errors.Is(err, history.ErrHistoryInvalidMutation) {
		t.Fatalf("out-of-range copy error = %v, want invalid mutation", err)
	}
	assertHistoryResources(t, session, 1, 0)
	if !session.ownsHistoryToken("term-1", window.Token) {
		t.Fatal("invalid copy discarded the valid owner token")
	}
	if _, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: "term-1", Mode: history.HistoryWindowModeOldest, Token: window.Token, Limit: 1,
	}); err != nil {
		t.Fatalf("legal pagination after invalid copy: %v", err)
	}
	if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{
		TerminalID: "term-1", Token: window.Token,
		Range: &history.HistoryCopyRange{
			Start: history.HistoryCopyPosition{LineID: 1},
			End:   history.HistoryCopyPosition{LineID: 1, Col: 1},
		},
	}); err != nil {
		t.Fatalf("legal copy after invalid copy: %v", err)
	}
	if _, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: "term-1", Mode: history.HistoryWindowModeLatest, Limit: 1,
	}); !errors.Is(err, ErrProtocolResourceExhausted) {
		t.Fatalf("invalid copy released quota unexpectedly: %v", err)
	}
	if err := session.ApplicationHistoryRelease(context.Background(), "term-1", window.Token); err != nil {
		t.Fatal(err)
	}
	assertHistoryResources(t, session, 0, 0)
}

func TestProtocolHistoryOwnerIsolationAndStaleCleanup(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 2}, "term-1")
	owner := newProtocolSession(server, nil, fullDaemonTransportScope())
	other := newProtocolSession(server, nil, fullDaemonTransportScope())
	window := applicationLatestHistory(t, owner, "term-1")

	if _, err := other.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: "term-1", Mode: history.HistoryWindowModeOldest, Token: window.Token, Limit: 1,
	}); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("cross-owner pagination error = %v", err)
	}
	if _, err := other.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-1", Token: window.Token}); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("cross-owner copy error = %v", err)
	}
	if err := other.ApplicationHistoryRelease(context.Background(), "term-1", window.Token); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("cross-owner release error = %v", err)
	}
	assertHistoryResources(t, other, 0, 0)
	if _, err := owner.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-1", Token: window.Token}); err != nil {
		t.Fatalf("non-owner attempts damaged owner token: %v", err)
	}

	if err := server.TerminalHistoryRelease(context.Background(), "term-1", window.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-1", Token: window.Token}); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("store-stale copy error = %v", err)
	}
	assertHistoryResources(t, owner, 0, 0)

	window = applicationLatestHistory(t, owner, "term-1")
	if err := server.TerminalHistoryRelease(context.Background(), "term-1", window.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: "term-1", Mode: history.HistoryWindowModeOldest, Token: window.Token, Limit: 1,
	}); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("store-stale pagination error = %v", err)
	}
	assertHistoryResources(t, owner, 0, 0)
}

func TestProtocolHistoryFreezeFailureRollsBackReservation(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 1}, "term-1")
	session := newProtocolSession(server, nil, fullDaemonTransportScope())

	if _, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: "term-1", Mode: history.HistoryWindowModeLatest, Limit: history.MaxHistoryWindowLines + 1,
	}); !errors.Is(err, history.ErrHistoryWindowLimit) {
		t.Fatalf("oversized freeze error = %v, want window limit", err)
	}
	assertHistoryResources(t, session, 0, 0)

	window := applicationLatestHistory(t, session, "term-1")
	assertHistoryResources(t, session, 1, 0)
	if err := session.ApplicationHistoryRelease(context.Background(), "term-1", window.Token); err != nil {
		t.Fatal(err)
	}
}

func TestProtocolCancelledLatestReleasesFrozenToken(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 1}, "term-1")
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := session.ApplicationHistoryWindow(ctx, history.HistoryWindowRequest{
		TerminalID: "term-1", Mode: history.HistoryWindowModeLatest, Limit: 1,
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled latest error = %v, want context canceled", err)
	}
	assertHistoryResources(t, session, 0, 0)
	window := applicationLatestHistory(t, session, "term-1")
	if err := session.ApplicationHistoryRelease(context.Background(), "term-1", window.Token); err != nil {
		t.Fatal(err)
	}
}

func TestProtocolHistoryCompositeKeyAndSharedResourceQuota(t *testing.T) {
	t.Run("tokens are unique across terminal stores", func(t *testing.T) {
		server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 2}, "term-a", "term-b")
		session := newProtocolSession(server, nil, fullDaemonTransportScope())
		first := applicationLatestHistory(t, session, "term-a")
		second := applicationLatestHistory(t, session, "term-b")
		if first.Token == second.Token {
			t.Fatalf("history tokens must identify their store instance, got %q for both terminals", first.Token)
		}
		assertHistoryResources(t, session, 2, 0)
		if err := session.ApplicationHistoryRelease(context.Background(), "term-a", first.Token); err != nil {
			t.Fatal(err)
		}
		if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-b", Token: second.Token}); err != nil {
			t.Fatalf("releasing terminal A removed terminal B's token: %v", err)
		}
		assertHistoryResources(t, session, 1, 0)
	})

	t.Run("attachment and history share MaxResources", func(t *testing.T) {
		server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1}, "term-resource")
		session := newProtocolSession(server, nil, fullDaemonTransportScope())
		attachment, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
		if err != nil {
			t.Fatal(err)
		}
		if err := attachment.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
			TerminalID: "term-resource", Mode: history.HistoryWindowModeLatest, Limit: 1,
		}); !errors.Is(err, ErrProtocolResourceExhausted) {
			t.Fatalf("history ignored attachment quota: %v", err)
		}
		if err := session.ReleaseApplicationResource(context.Background(), attachment.Result().Token); err != nil {
			t.Fatal(err)
		}
		window := applicationLatestHistory(t, session, "term-resource")
		assertHistoryResources(t, session, 1, 0)
		if err := session.ApplicationHistoryRelease(context.Background(), "term-resource", window.Token); err != nil {
			t.Fatal(err)
		}
	})
}

func TestProtocolHistoryTokensCannotCrossTerminalRecreation(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 4}, "term-recreated")
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	oldRelease := applicationLatestHistory(t, session, "term-recreated")
	oldPage := applicationLatestHistory(t, session, "term-recreated")

	if err := server.RemoveTerminal("term-recreated"); err != nil {
		t.Fatal(err)
	}
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-recreated", Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 24}}); err != nil {
		t.Fatal(err)
	}
	terminal, err := server.Terminal("term-recreated")
	if err != nil {
		t.Fatal(err)
	}
	if err := terminal.lineHistory.AppendLifecycleLines([]string{"new-store-line"}); err != nil {
		t.Fatal(err)
	}
	current := applicationLatestHistory(t, session, "term-recreated")
	if current.Token == oldRelease.Token || current.Token == oldPage.Token {
		t.Fatalf("recreated terminal reused a frozen token: old=%q/%q current=%q", oldRelease.Token, oldPage.Token, current.Token)
	}

	if err := session.ApplicationHistoryRelease(context.Background(), "term-recreated", oldRelease.Token); err != nil {
		t.Fatalf("release old snapshot: %v", err)
	}
	if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-recreated", Token: current.Token}); err != nil {
		t.Fatalf("old release damaged recreated terminal snapshot: %v", err)
	}
	if _, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: "term-recreated", Mode: history.HistoryWindowModeOldest, Token: oldPage.Token, Limit: 1,
	}); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("old snapshot pagination error = %v, want stale window", err)
	}
	if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-recreated", Token: current.Token}); err != nil {
		t.Fatalf("old pagination damaged recreated terminal snapshot: %v", err)
	}
	assertHistoryResources(t, session, 1, 0)
}

func TestProtocolHistoryReleaseAfterTerminalRemovalFreesQuota(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 1}, "term-removed")
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	window := applicationLatestHistory(t, session, "term-removed")
	if err := server.RemoveTerminal("term-removed"); err != nil {
		t.Fatal(err)
	}
	if err := session.ApplicationHistoryRelease(context.Background(), "term-removed", window.Token); err != nil {
		t.Fatalf("release after terminal removal: %v", err)
	}
	assertHistoryResources(t, session, 0, 0)
}

func TestProtocolHistoryDisconnectWaitsThenReleasesSnapshots(t *testing.T) {
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: 1}, "term-1")
	client, daemon := memory.NewPair()
	session := newProtocolSession(server, daemon, fullDaemonTransportScope())
	window := applicationLatestHistory(t, session, "term-1")

	session.requests.Add(1)
	done := make(chan error, 1)
	go func() { done <- session.run(context.Background()) }()
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		t.Fatalf("session cleanup did not wait for requests: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if _, err := server.TerminalHistoryWindow(context.Background(), "term-1", history.HistoryWindowRequest{Token: window.Token, Limit: 1}); err != nil {
		t.Fatalf("snapshot released before requests.Wait: %v", err)
	}
	session.requests.Done()
	if err := <-done; err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("session run error = %v", err)
	}
	assertHistoryResources(t, session, 0, 0)
	if _, err := server.TerminalHistoryWindow(context.Background(), "term-1", history.HistoryWindowRequest{Token: window.Token, Limit: 1}); !errors.Is(err, history.ErrHistoryStaleWindow) {
		t.Fatalf("disconnect did not release store token: %v", err)
	}
}

func TestProtocolHistoryConcurrentLatestCopyReleaseLeavesNoResources(t *testing.T) {
	const workers = 24
	server := newHistoryOwnerTestServer(t, ProtocolSessionLimits{MaxResources: workers}, "term-1")
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			window, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
				TerminalID: "term-1", Mode: history.HistoryWindowModeLatest, Limit: 1,
			})
			if err != nil {
				errs <- err
				return
			}
			if _, err := session.ApplicationHistoryCopy(context.Background(), history.HistoryCopyRequest{TerminalID: "term-1", Token: window.Token}); err != nil {
				errs <- err
				return
			}
			if err := session.ApplicationHistoryRelease(context.Background(), "term-1", window.Token); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent history lifecycle: %v", err)
	}
	assertHistoryResources(t, session, 0, 0)
}

func newHistoryOwnerTestServer(t *testing.T, limits ProtocolSessionLimits, terminalIDs ...string) *Server {
	t.Helper()
	server := NewServer(
		WithProcessFactory(newRecordingProcessFactory()),
		WithHistoryStoreFactory(LineHistoryStoreFactory(t.TempDir())),
		WithProtocolSessionLimits(limits),
	)
	t.Cleanup(func() { _ = server.Shutdown(context.Background()) })
	for _, terminalID := range terminalIDs {
		if _, err := server.RegisterTerminal(TerminalRecord{ID: terminalID, Command: []string{"shell"}, Size: Size{Cols: 80, Rows: 24}}); err != nil {
			t.Fatal(err)
		}
		terminal, err := server.Terminal(terminalID)
		if err != nil {
			t.Fatal(err)
		}
		if err := terminal.lineHistory.AppendLifecycleLines([]string{"history-owner-line"}); err != nil {
			t.Fatal(err)
		}
	}
	return server
}

func applicationLatestHistory(t *testing.T, session *protocolSession, terminalID string) history.HistoryWindow {
	t.Helper()
	window, err := session.ApplicationHistoryWindow(context.Background(), history.HistoryWindowRequest{
		TerminalID: terminalID, Mode: history.HistoryWindowModeLatest, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if window.Token == "" {
		t.Fatal("latest history did not return a token")
	}
	return window
}

func assertHistoryResources(t *testing.T, session *protocolSession, tokens, reservations int) {
	t.Helper()
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if len(session.historyTokens) != tokens || session.historyTokenReservations != reservations {
		t.Fatalf("history resources = tokens:%d reservations:%d, want %d/%d", len(session.historyTokens), session.historyTokenReservations, tokens, reservations)
	}
}
