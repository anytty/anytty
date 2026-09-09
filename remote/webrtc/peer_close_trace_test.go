package webrtc

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/connecttrace"
	pion "github.com/pion/webrtc/v4"
)

type peerTraceBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *peerTraceBuffer) Write(value []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(value)
}

func (buffer *peerTraceBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}

func TestPeerCloseTraceIdentifiesBlockedPhaseWithoutSecrets(t *testing.T) {
	var output peerTraceBuffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	const traceID = "b8487da6-2a16-40d4-809a-812252970dfd"
	const secret = "token=private-close-error"
	traceText := func() string {
		var lines []string
		for _, line := range strings.Split(output.String(), "\n") {
			if strings.Contains(line, "trace_id="+traceID) {
				lines = append(lines, line)
			}
		}
		return strings.Join(lines, "\n")
	}
	ctx, cancel := context.WithCancel(connecttrace.Attach(context.Background(), traceID))
	defer cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	lifecycle := newPeerLifecycle(ctx, nil, cancel, func() { panic(secret) }, func(*pion.PeerConnection) error {
		close(started)
		<-release
		return errors.New(secret)
	})
	lifecycle.requestClose()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("peer close did not start")
	}
	before := traceText()
	if !strings.Contains(before, "component=daemon_peer_close stage=peer_close_started") || strings.Contains(before, "stage=peer_close_returned") {
		t.Fatalf("blocked close stage was not identified: %s", before)
	}
	unblock()
	select {
	case <-lifecycle.done:
	case <-time.After(time.Second):
		t.Fatal("close did not complete")
	}
	text := traceText()
	if strings.Contains(text, secret) || !strings.Contains(text, "trace_id="+traceID) {
		t.Fatalf("unsafe or uncorrelated close trace: %s", text)
	}
	last := -1
	for _, stage := range []string{"peer_close_started", "peer_close_returned", "handler_wait_started", "handler_finished", "watcher_wait_started", "watcher_finished", "callback_started", "callback_finished", "failed error_type="} {
		index := strings.Index(text, "stage="+stage)
		if index <= last {
			t.Fatalf("missing/out-of-order stage %s: %s", stage, text)
		}
		last = index
	}
}
