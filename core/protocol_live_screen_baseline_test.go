package core

import (
	"context"
	"testing"
	"time"
)

func TestLiveScreenBaselinePromotesOnlyObservedOffer(t *testing.T) {
	server := NewServer()
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer session.clearLiveScreenBaselines()

	first := &nativeScreenBaseline{revision: 7, size: NativeScreenSize{Cols: 80, Rows: 2}, rowHashes: []uint64{1, 2}}
	session.offerLiveScreenBaseline("term-1", first)
	confirmed, release := session.acquireLiveScreenBaseline("term-1", 7)
	if confirmed != first {
		t.Fatalf("observed offer was not promoted: got %#v want %#v", confirmed, first)
	}
	release()

	second := &nativeScreenBaseline{revision: 8, size: first.size, rowHashes: []uint64{2, 3}}
	session.offerLiveScreenBaseline("term-1", second)
	retry, releaseRetry := session.acquireLiveScreenBaseline("term-1", 7)
	if retry != first {
		t.Fatalf("unconfirmed offer replaced client-confirmed baseline: got %#v want %#v", retry, first)
	}
	releaseRetry()
}

func TestLiveScreenBaselineExpiryAndSessionIsolation(t *testing.T) {
	server := NewServer()
	owner := newProtocolSession(server, nil, fullDaemonTransportScope())
	other := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer owner.clearLiveScreenBaselines()
	defer other.clearLiveScreenBaselines()

	baseline := &nativeScreenBaseline{revision: 4, rowHashes: []uint64{1, 2, 3}}
	owner.offerLiveScreenBaseline("term-1", baseline)
	if got, release := other.acquireLiveScreenBaseline("term-1", 4); got != nil {
		release()
		t.Fatalf("baseline leaked across protocol sessions: %#v", got)
	}

	owner.liveBaselineMu.Lock()
	owner.liveBaselines["term-1"].offered.expiresAt = time.Now().Add(-time.Second)
	owner.pruneLiveScreenBaselinesLocked(time.Now())
	remaining := len(owner.liveBaselines)
	bytes := owner.liveBaselineBytes
	owner.liveBaselineMu.Unlock()
	if remaining != 0 || bytes != 0 || server.liveBaselineBytes.Load() != 0 {
		t.Fatalf("expired baseline retained: entries=%d session_bytes=%d server_bytes=%d", remaining, bytes, server.liveBaselineBytes.Load())
	}
}

func TestLiveScreenBaselinePinSurvivesLongPollTTL(t *testing.T) {
	server := NewServer()
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer session.clearLiveScreenBaselines()
	baseline := &nativeScreenBaseline{revision: 9, rowHashes: []uint64{1}}
	session.offerLiveScreenBaseline("term-1", baseline)
	got, release := session.acquireLiveScreenBaseline("term-1", 9)
	if got != baseline {
		t.Fatalf("acquire baseline = %#v, want %#v", got, baseline)
	}
	session.liveBaselineMu.Lock()
	entry := session.liveBaselines["term-1"]
	entry.confirmed.expiresAt = time.Now().Add(-time.Second)
	session.pruneLiveScreenBaselinesLocked(time.Now())
	stillPinned := entry.confirmed != nil
	session.liveBaselineMu.Unlock()
	if !stillPinned {
		t.Fatal("active long poll lost its pinned baseline")
	}
	release()
}

func TestProtocolSessionBaselineBridgesMoreThanOldJournalWindow(t *testing.T) {
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-baseline", Command: []string{"shell"}, Size: Size{Cols: 20, Rows: 3}}); err != nil {
		t.Fatalf("register terminal: %v", err)
	}
	terminal, err := server.Terminal("term-baseline")
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if _, err := terminal.applyLiveOutput("base"); err != nil {
		t.Fatalf("base output: %v", err)
	}
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer session.clearLiveScreenBaselines()
	bootstrap, err := session.ApplicationLiveScreenNext(context.Background(), "term-baseline", 0)
	if err != nil || !bootstrap.FullReplace {
		t.Fatalf("bootstrap=%#v err=%v", bootstrap, err)
	}
	for revision := 0; revision < 100; revision++ {
		if _, err := terminal.applyLiveOutput("\rnext"); err != nil {
			t.Fatalf("output %d: %v", revision, err)
		}
	}
	delta, err := session.ApplicationLiveScreenNext(context.Background(), "term-baseline", bootstrap.Revision)
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if delta.FullReplace || delta.BaseRevision != bootstrap.Revision {
		t.Fatalf("client baseline should bridge arbitrary intermediate revisions: %#v", delta)
	}

	other := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer other.clearLiveScreenBaselines()
	full, err := other.ApplicationLiveScreenNext(context.Background(), "term-baseline", bootstrap.Revision)
	if err != nil {
		t.Fatalf("isolated full response: %v", err)
	}
	if !full.FullReplace {
		t.Fatalf("session without confirmed baseline must receive full response: %#v", full)
	}
}

func TestLiveScreenBaselineEntryCountIsBounded(t *testing.T) {
	server := NewServer()
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	defer session.clearLiveScreenBaselines()
	for index := 0; index < maxLiveScreenBaselineEntries+10; index++ {
		session.offerLiveScreenBaseline(string(rune(index+1)), &nativeScreenBaseline{revision: 1, rowHashes: []uint64{uint64(index)}})
	}
	session.liveBaselineMu.Lock()
	entries := len(session.liveBaselines)
	session.liveBaselineMu.Unlock()
	if entries != maxLiveScreenBaselineEntries {
		t.Fatalf("baseline entries=%d, want cap %d", entries, maxLiveScreenBaselineEntries)
	}
}
