package provider_test

import (
	"context"
	"testing"

	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/pool/core/history"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// TestProviderHistoryTokenIsolationAndRollback 迁移自旧 daemon protocol 的
// cross-owner stale、invalid copy 保留 owner token、以及 freeze 失败回滚配额。
func TestProviderHistoryTokenIsolationAndRollback(t *testing.T) {
	t.Parallel()
	_, _, socketPath := startProvider(t)
	ctx := context.Background()
	owner := dialOwnershipClient(t, socketPath)
	other := dialOwnershipClient(t, socketPath)

	if _, err := owner.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-history-token", Command: []string{"/bin/sh", "-c", "printf 'token-alpha\\ntoken-beta\\n'; sleep 30"},
		Size: &providerv1.Size{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatal(err)
	}
	window, err := owner.HistoryWindow(ctx, &providerv1.HistoryWindowCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Mode:     providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols:     80, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	token := window.GetToken()
	if token == "" {
		t.Fatalf("history window token is empty")
	}

	if _, err := other.HistoryWindow(ctx, &providerv1.HistoryWindowCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Mode:     providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST,
		Token:    token, Cols: 80, Limit: 5,
	}); providerCode(err) != providerproto.ErrorStaleResource {
		t.Fatalf("cross-owner window err = %v, want stale", err)
	}
	if _, err := other.HistoryCopy(ctx, &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Window:   &providerv1.HistoryWindowCommand{Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"}, Token: token, Cols: 80},
	}); providerCode(err) != providerproto.ErrorStaleResource {
		t.Fatalf("cross-owner copy err = %v, want stale", err)
	}
	if err := other.HistoryRelease(ctx, "term-history-token", token); providerCode(err) != providerproto.ErrorStaleResource {
		t.Fatalf("cross-owner release err = %v, want stale", err)
	}

	// 非 owner 的失败尝试不得破坏 owner token。
	if _, err := owner.HistoryCopy(ctx, &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Window:   &providerv1.HistoryWindowCommand{Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"}, Token: token, Cols: 80},
	}); err != nil {
		t.Fatalf("owner copy after cross-owner attempts: %v", err)
	}

	// freeze 失败（超出 window limit）必须回滚额度：随后仍能创建新 token。
	if _, err := owner.HistoryWindow(ctx, &providerv1.HistoryWindowCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Mode:     providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols:     80, Limit: history.MaxHistoryWindowLines + 1,
	}); err == nil {
		t.Fatalf("oversized latest window unexpectedly succeeded")
	}
	next, err := owner.HistoryWindow(ctx, &providerv1.HistoryWindowCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Mode:     providerv1.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols:     80, Limit: 20,
	})
	if err != nil || next.GetToken() == "" {
		t.Fatalf("window after failed freeze = %#v err=%v", next, err)
	}
	if err := owner.HistoryRelease(ctx, "term-history-token", next.GetToken()); err != nil {
		t.Fatal(err)
	}

	// release 后 token 变为 stale。
	if _, err := owner.HistoryCopy(ctx, &providerv1.HistoryCopyCommand{
		Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"},
		Window:   &providerv1.HistoryWindowCommand{Terminal: &providerv1.TerminalRef{TerminalId: "term-history-token"}, Token: next.GetToken(), Cols: 80},
	}); providerCode(err) != providerproto.ErrorStaleResource {
		t.Fatalf("released token copy err = %v, want stale", err)
	}
}

// TestProviderLiveScreenBaselineDeltaAndIsolation 迁移自旧 protocol 的
// confirmed baseline delta 与 session 隔离语义。
func TestProviderLiveScreenBaselineDeltaAndIsolation(t *testing.T) {
	t.Parallel()
	coreServer, _, socketPath := startProvider(t)
	ctx := context.Background()
	owner := dialOwnershipClient(t, socketPath)

	if _, err := owner.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-live-baseline", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 40, Rows: 10},
	}); err != nil {
		t.Fatal(err)
	}
	bootstrap, err := owner.LiveScreenNext(ctx, "term-live-baseline", 0)
	if err != nil || !bootstrap.GetFullReplace() || bootstrap.GetLiveRevision() == 0 {
		t.Fatalf("bootstrap screen = %#v err=%v", bootstrap, err)
	}
	if err := coreServer.IngestOutput(ctx, "term-live-baseline", "baseline-delta\r\n"); err != nil {
		t.Fatal(err)
	}
	delta, err := owner.LiveScreenNext(ctx, "term-live-baseline", bootstrap.GetLiveRevision())
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if delta.GetFullReplace() || delta.GetBaseRevision() != bootstrap.GetLiveRevision() {
		t.Fatalf("confirmed session baseline did not bridge delta: %#v", delta)
	}

	other := dialOwnershipClient(t, socketPath)
	full, err := other.LiveScreenNext(ctx, "term-live-baseline", bootstrap.GetLiveRevision())
	if err != nil {
		t.Fatalf("isolated full response: %v", err)
	}
	if !full.GetFullReplace() {
		t.Fatalf("session without confirmed baseline must receive full response: %#v", full)
	}
}

// TestProviderSessionBaselineBridgesMoreThanJournalWindow 迁移自旧 protocol 的
// "baseline bridges arbitrary intermediate revisions" 断言。
func TestProviderSessionBaselineBridgesMoreThanJournalWindow(t *testing.T) {
	t.Parallel()
	coreServer, _, socketPath := startProvider(t)
	ctx := context.Background()
	client := dialOwnershipClient(t, socketPath)
	if _, err := client.Create(ctx, &providerv1.TerminalCreateSpec{
		TerminalId: "term-live-bridge", Command: []string{"/bin/cat"}, Size: &providerv1.Size{Cols: 20, Rows: 3},
	}); err != nil {
		t.Fatal(err)
	}
	bootstrap, err := client.LiveScreenNext(ctx, "term-live-bridge", 0)
	if err != nil || !bootstrap.GetFullReplace() {
		t.Fatalf("bootstrap = %#v err=%v", bootstrap, err)
	}
	for revision := 0; revision < 100; revision++ {
		if err := coreServer.IngestOutput(ctx, "term-live-bridge", "\rnext"); err != nil {
			t.Fatal(err)
		}
	}
	delta, err := client.LiveScreenNext(ctx, "term-live-bridge", bootstrap.GetLiveRevision())
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if delta.GetFullReplace() || delta.GetBaseRevision() != bootstrap.GetLiveRevision() {
		t.Fatalf("baseline must bridge arbitrary intermediate revisions: %#v", delta)
	}
}
