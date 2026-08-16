package webrtc

import (
	"context"
	"testing"
	"time"

	pion "github.com/pion/webrtc/v4"
)

func TestICEGatheringWaiterStopsAfterPreferredCandidate(t *testing.T) {
	waiter := NewICEGatheringWaiter(true, false, ICEGatheringCloudGrace)
	waiter.Observe(&pion.ICECandidate{Typ: pion.ICECandidateTypeHost})
	done := make(chan error, 1)
	go func() { done <- waiter.Wait(context.Background(), make(chan struct{}), 2*time.Second) }()

	select {
	case err := <-done:
		t.Fatalf("host candidate prematurely ended relay gathering: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	waiter.Observe(&pion.ICECandidate{Typ: pion.ICECandidateTypeRelay})
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(ICEGatheringCloudGrace + time.Second):
		t.Fatal("relay candidate did not end bounded gathering")
	}
}

func TestICEGatheringWaiterReturnsImmediatelyForDirectCandidate(t *testing.T) {
	waiter := NewICEGatheringWaiter(false, false, ICEGatheringDirectGrace)
	waiter.Observe(&pion.ICECandidate{Typ: pion.ICECandidateTypeHost})
	started := time.Now()
	if err := waiter.Wait(context.Background(), make(chan struct{}), time.Second); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("direct candidate waited %v before signaling", elapsed)
	}
}

func TestICEGatheringWaiterHardTimeoutKeepsDirectCandidate(t *testing.T) {
	waiter := NewICEGatheringWaiter(true, false, ICEGatheringCloudGrace)
	waiter.Observe(&pion.ICECandidate{Typ: pion.ICECandidateTypeHost})
	if err := waiter.Wait(context.Background(), make(chan struct{}), time.Millisecond); err != nil {
		t.Fatalf("hard timeout discarded usable direct candidate: %v", err)
	}
}

func TestICEGatheringWaiterRejectsEmptyGathering(t *testing.T) {
	waiter := NewICEGatheringWaiter(false, false, ICEGatheringDirectGrace)
	complete := make(chan struct{})
	close(complete)
	if err := waiter.Wait(context.Background(), complete, time.Second); err == nil {
		t.Fatal("empty ICE gathering was accepted")
	}
}

func TestICEGatheringWaiterAllowsEmptyDirectTCPGathering(t *testing.T) {
	waiter := NewICEGatheringWaiter(false, true, ICEGatheringDirectGrace)
	complete := make(chan struct{})
	close(complete)
	if err := waiter.Wait(context.Background(), complete, time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestICEGatheringPreferredGraceOnlyWaitsForManagedICECandidates(t *testing.T) {
	if got := ICEGatheringPreferredGrace(false); got != ICEGatheringDirectGrace {
		t.Fatalf("direct-only grace = %s, want %s", got, ICEGatheringDirectGrace)
	}
	if got := ICEGatheringPreferredGrace(true); got != ICEGatheringCloudGrace {
		t.Fatalf("managed ICE grace = %s, want %s", got, ICEGatheringCloudGrace)
	}
}
