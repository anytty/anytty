package endpoint

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRemotePTYDetachedStreamDoesNotMarkTerminalExited(t *testing.T) {
	p := &RemotePTY{
		mgr:     &Manager{baseCtx: context.Background()},
		notify:  make(chan struct{}, 1),
		closeCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	p.cond = sync.NewCond(&p.mu)
	att := &attachment{stream: newStream()}
	p.pending = att
	go p.run()

	p.mu.Lock()
	p.detached = true
	p.mu.Unlock()
	att.stream.close(errAttachmentDetached)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !p.Exited() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if p.Exited() {
		t.Fatal("detaching an attachment must not mark the daemon terminal exited")
	}
	close(p.closeCh)
	select {
	case <-p.doneCh:
	case <-time.After(time.Second):
		t.Fatal("remote pty pump did not stop")
	}
}
