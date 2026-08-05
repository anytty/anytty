package webrtc

import (
	"context"
	"fmt"
	"sync"
	"time"

	pion "github.com/pion/webrtc/v4"
)

const (
	ICEGatheringDirectGrace = 25 * time.Millisecond
	ICEGatheringCloudGrace  = 500 * time.Millisecond
)

// ICEGatheringPreferredGrace keeps the longer Cloud window only when the attempt has managed
// ICE servers and may still gather a TURN candidate. Direct-only attempts have no later class
// of candidate to wait for after host/srflx becomes available.
func ICEGatheringPreferredGrace(hasManagedICEServer bool) time.Duration {
	if hasManagedICEServer {
		return ICEGatheringCloudGrace
	}
	return ICEGatheringDirectGrace
}

// ICEGatheringWaiter bounds non-trickle gathering. Callers choose whether the
// preferred candidate is host/srflx or relay according to the route policy.
type ICEGatheringWaiter struct {
	requireRelay   bool
	allowEmpty     bool
	preferredGrace time.Duration
	anyReady       chan struct{}
	preferred      chan struct{}
	anyOnce        sync.Once
	preferredOnce  sync.Once
}

func NewICEGatheringWaiter(requireRelay, allowEmpty bool, preferredGrace time.Duration) *ICEGatheringWaiter {
	return &ICEGatheringWaiter{
		requireRelay:   requireRelay,
		allowEmpty:     allowEmpty,
		preferredGrace: preferredGrace,
		anyReady:       make(chan struct{}),
		preferred:      make(chan struct{}),
	}
}

func (waiter *ICEGatheringWaiter) Observe(candidate *pion.ICECandidate) {
	if waiter == nil || candidate == nil {
		return
	}
	waiter.anyOnce.Do(func() { close(waiter.anyReady) })
	preferred := candidate.Typ == pion.ICECandidateTypeRelay
	if !waiter.requireRelay {
		preferred = candidate.Typ == pion.ICECandidateTypeHost || candidate.Typ == pion.ICECandidateTypeSrflx
	}
	if preferred {
		waiter.preferredOnce.Do(func() { close(waiter.preferred) })
	}
}

func (waiter *ICEGatheringWaiter) Wait(ctx context.Context, complete <-chan struct{}, hardTimeout time.Duration) error {
	if waiter == nil {
		return fmt.Errorf("ICE gathering waiter is not configured")
	}
	if hardTimeout <= 0 {
		return fmt.Errorf("ICE gathering timeout must be positive")
	}
	if waiter.preferredGrace <= 0 {
		return fmt.Errorf("ICE gathering grace must be positive")
	}
	hard := time.NewTimer(hardTimeout)
	defer hard.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-complete:
		return waiter.requireCandidate()
	case <-waiter.preferred:
		grace := time.NewTimer(waiter.preferredGrace)
		defer grace.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-complete:
			return waiter.requireCandidate()
		case <-grace.C:
			return nil
		case <-hard.C:
			return waiter.requireCandidate()
		}
	case <-hard.C:
		return waiter.requireCandidate()
	}
}

func (waiter *ICEGatheringWaiter) requireCandidate() error {
	if waiter.allowEmpty {
		return nil
	}
	select {
	case <-waiter.anyReady:
		return nil
	default:
		return fmt.Errorf("ICE gathering produced no local candidates")
	}
}
