package provider

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/anytty/anytty/pool/core"
)

// newBaselineTestSession 建立不监听 socket 的 provider session，用于 baseline 缓存单测。
func newBaselineTestSession(t *testing.T) (*Server, *session, *core.Server) {
	t.Helper()
	coreServer := core.NewServer(core.WithHistoryDisabled())
	t.Cleanup(func() { _ = coreServer.Shutdown(context.Background()) })
	server, err := New(coreServer, Config{Socket: filepath.Join(t.TempDir(), "provider.sock")})
	if err != nil {
		t.Fatal(err)
	}
	return server, newSession(server, nil), coreServer
}

// baselinePairForTest 从真实 core terminal 产生两个不同 revision 的 opaque baseline。
func baselinePairForTest(t *testing.T, coreServer *core.Server, terminalID string) (*core.NativeScreenBaseline, *core.NativeScreenBaseline) {
	t.Helper()
	if _, err := coreServer.RegisterTerminal(core.TerminalRecord{ID: terminalID, Command: []string{"/bin/cat"}, Size: core.Size{Cols: 20, Rows: 3}}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := coreServer.IngestOutput(ctx, terminalID, "base-a\r\n"); err != nil {
		t.Fatal(err)
	}
	_, first, err := coreServer.NextLiveScreenWithBaseline(ctx, terminalID, 0, nil)
	if err != nil || first == nil {
		t.Fatalf("first baseline = %#v err=%v", first, err)
	}
	if err := coreServer.IngestOutput(ctx, terminalID, "base-b\r\n"); err != nil {
		t.Fatal(err)
	}
	_, second, err := coreServer.NextLiveScreenWithBaseline(ctx, terminalID, first.Revision(), first)
	if err != nil || second == nil {
		t.Fatalf("second baseline = %#v err=%v", second, err)
	}
	if second.Revision() <= first.Revision() {
		t.Fatalf("baseline revisions not increasing: %d then %d", first.Revision(), second.Revision())
	}
	return first, second
}

// TestSessionLiveScreenBaselinePromotionAndIsolation 迁移自旧 protocol 的
// "promotes only observed offer" 与 session 隔离断言。
func TestSessionLiveScreenBaselinePromotionAndIsolation(t *testing.T) {
	server, session, coreServer := newBaselineTestSession(t)
	first, second := baselinePairForTest(t, coreServer, "term-baseline-cache")

	session.offerLiveScreenBaseline("term-baseline-cache", first)
	confirmed, release := session.acquireLiveScreenBaseline("term-baseline-cache", first.Revision())
	if confirmed != first {
		t.Fatalf("observed offer was not promoted: got %p want %p", confirmed, first)
	}
	release()

	session.offerLiveScreenBaseline("term-baseline-cache", second)
	retry, releaseRetry := session.acquireLiveScreenBaseline("term-baseline-cache", first.Revision())
	if retry != first {
		t.Fatalf("unconfirmed offer replaced confirmed baseline: got %p want %p", retry, first)
	}
	releaseRetry()

	other := newSession(server, nil)
	defer other.clearLiveScreenBaselines()
	if leaked, releaseLeaked := other.acquireLiveScreenBaseline("term-baseline-cache", first.Revision()); leaked != nil {
		releaseLeaked()
		t.Fatalf("baseline leaked across sessions: %p", leaked)
	}
}

// TestSessionLiveScreenBaselinePinAndExpiry 迁移自旧 protocol 的 pin 与过期断言。
func TestSessionLiveScreenBaselinePinAndExpiry(t *testing.T) {
	server, session, coreServer := newBaselineTestSession(t)
	first, _ := baselinePairForTest(t, coreServer, "term-baseline-expiry")

	session.offerLiveScreenBaseline("term-baseline-expiry", first)
	acquired, release := session.acquireLiveScreenBaseline("term-baseline-expiry", first.Revision())
	if acquired != first {
		t.Fatalf("acquire = %p want %p", acquired, first)
	}
	session.liveBaselineMu.Lock()
	entry := session.liveBaselines["term-baseline-expiry"]
	entry.confirmed.expiresAt = time.Now().Add(-time.Second)
	session.pruneLiveScreenBaselinesLocked(time.Now())
	stillPinned := entry.confirmed != nil
	session.liveBaselineMu.Unlock()
	if !stillPinned {
		t.Fatal("active long poll lost its pinned baseline")
	}
	release()

	session.liveBaselineMu.Lock()
	entry = session.liveBaselines["term-baseline-expiry"]
	entry.confirmed.expiresAt = time.Now().Add(-time.Second)
	session.pruneLiveScreenBaselinesLocked(time.Now())
	remaining := len(session.liveBaselines)
	bytes := session.liveBaselineBytes
	session.liveBaselineMu.Unlock()
	if remaining != 0 || bytes != 0 || server.liveBaselineUsed.Load() != 0 {
		t.Fatalf("expired baseline retained: entries=%d session_bytes=%d server_bytes=%d", remaining, bytes, server.liveBaselineUsed.Load())
	}
}

// TestSessionLiveScreenBaselineEntryCountIsBounded 迁移自旧 protocol 的条目上限断言。
func TestSessionLiveScreenBaselineEntryCountIsBounded(t *testing.T) {
	_, session, coreServer := newBaselineTestSession(t)
	first, _ := baselinePairForTest(t, coreServer, "term-baseline-bound")

	for index := 0; index < maxLiveScreenBaselineEntries+10; index++ {
		session.offerLiveScreenBaseline(string(rune(index+1)), first)
	}
	session.liveBaselineMu.Lock()
	entries := len(session.liveBaselines)
	session.liveBaselineMu.Unlock()
	if entries != maxLiveScreenBaselineEntries {
		t.Fatalf("baseline entries=%d, want cap %d", entries, maxLiveScreenBaselineEntries)
	}
}
