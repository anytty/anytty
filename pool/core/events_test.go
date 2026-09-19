package core

import (
	"context"
	"testing"
	"time"
)

func TestEventBrokerCoalescesBlockedAttachmentProjectionToLatestEpoch(t *testing.T) {
	broker := newEventBroker(1)
	ctx, cancel := context.WithCancel(context.Background())
	events := broker.subscribe(ctx, EventFilter{TerminalID: "term-1"})
	defer func() {
		cancel()
		broker.close()
	}()

	for epoch := uint64(1); epoch <= 100; epoch++ {
		broker.publish(Event{
			Type:       EventTerminalMetadataChanged,
			TerminalID: "term-1",
			Attachment: &TerminalAttachmentProjection{ResizeEpoch: epoch},
		})
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Attachment != nil && event.Attachment.ResizeEpoch == 100 {
				return
			}
		case <-deadline:
			t.Fatal("latest attachment projection was permanently dropped behind a blocked subscriber")
		}
	}
}

func TestEventBrokerCancellationClosesSubscription(t *testing.T) {
	broker := newEventBroker(1)
	ctx, cancel := context.WithCancel(context.Background())
	events := broker.subscribe(ctx, EventFilter{})
	cancel()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("subscription emitted an event after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscription channel did not close after cancellation")
	}
	broker.close()
}
