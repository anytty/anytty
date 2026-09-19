package client

import (
	"context"
	"errors"
	"testing"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestOfferFutureStartsImmediatelyAndPreservesReadyIdentity(t *testing.T) {
	started := make(chan *cloudv1.ClientReady, 1)
	release := make(chan struct{})
	future := newOfferFuture(context.Background(), func(_ context.Context, ready *cloudv1.ClientReady) (string, error) {
		started <- ready
		<-release
		return "direct-offer", nil
	}, &cloudv1.ClientReady{SessionId: "session-1", Generation: 7})

	ready := <-started
	if ready.GetSessionId() != "session-1" || ready.GetGeneration() != 7 {
		t.Fatalf("prefetched ready = %#v", ready)
	}
	close(release)
	if offer, err := future.Await(); err != nil || offer != "direct-offer" {
		t.Fatalf("offer=%q err=%v", offer, err)
	}
	future.Close()
}

func TestOfferFutureCloseCancelsAndWaitsForProducer(t *testing.T) {
	finished := make(chan struct{})
	future := newOfferFuture(context.Background(), func(ctx context.Context, _ *cloudv1.ClientReady) (string, error) {
		<-ctx.Done()
		close(finished)
		return "", ctx.Err()
	}, &cloudv1.ClientReady{})
	future.Close()
	<-finished
	if _, err := future.Await(); !errors.Is(err, context.Canceled) {
		t.Fatalf("offer future error=%v", err)
	}
}
