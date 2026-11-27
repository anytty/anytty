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
	entry.frames[0].expiresAt = time.Now().Add(-time.Second)
	session.pruneLiveScreenBaselinesLocked(time.Now())
	stillPinned := len(entry.frames) > 0
	session.liveBaselineMu.Unlock()
	if !stillPinned {
		t.Fatal("active long poll lost its pinned baseline")
	}
	release()

	session.liveBaselineMu.Lock()
	entry = session.liveBaselines["term-baseline-expiry"]
	entry.frames[0].expiresAt = time.Now().Add(-time.Second)
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

// TestSessionLiveScreenBaselineSurvivesMissedFrame 覆盖“客户端漏收一帧”场景：
// 客户端停在 R0，服务端已经 offer 到 R2，acquire(R0) 仍必须命中，避免整屏重发。
func TestSessionLiveScreenBaselineSurvivesMissedFrame(t *testing.T) {
	session, coreServer := func() (*session, *core.Server) {
		_, session, coreServer := newBaselineTestSession(t)
		return session, coreServer
	}()
	ctx := context.Background()
	first, second := baselinePairForTest(t, coreServer, "term-baseline-missed")
	if err := coreServer.IngestOutput(ctx, "term-baseline-missed", "base-c\r\n"); err != nil {
		t.Fatal(err)
	}
	_, third, err := coreServer.NextLiveScreenWithBaseline(ctx, "term-baseline-missed", second.Revision(), second)
	if err != nil || third == nil {
		t.Fatalf("third baseline = %#v err=%v", third, err)
	}

	session.offerLiveScreenBaseline("term-baseline-missed", first)
	session.offerLiveScreenBaseline("term-baseline-missed", second)
	session.offerLiveScreenBaseline("term-baseline-missed", third)

	// 客户端只确认到 first（中间两帧的应答丢了）。
	acquired, release := session.acquireLiveScreenBaseline("term-baseline-missed", first.Revision())
	if acquired != first {
		t.Fatalf("missed-frame acquire = %p want %p", acquired, first)
	}
	release()

	// 每一帧都能命中，直到超出环容量。
	for _, want := range []*core.NativeScreenBaseline{second, third} {
		got, releaseFrame := session.acquireLiveScreenBaseline("term-baseline-missed", want.Revision())
		if got != want {
			t.Fatalf("acquire rev %d = %p want %p", want.Revision(), got, want)
		}
		releaseFrame()
	}

	// 未知 revision 仍然退回整屏快照（nil baseline）。
	if leaked, releaseLeaked := session.acquireLiveScreenBaseline("term-baseline-missed", 9999); leaked != nil {
		releaseLeaked()
		t.Fatalf("unknown revision returned baseline %p", leaked)
	}
}
