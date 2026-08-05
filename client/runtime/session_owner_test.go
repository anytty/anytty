package runtime

import (
	"context"
	"errors"
	"io"
	goruntime "runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
)

func TestSessionOwnerReplacesGenerationAndRejectsStaleOperations(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	firstDialer := &ownerDialer{}
	firstLease, err := owner.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, firstDialer)
	if err != nil {
		t.Fatal(err)
	}
	secondDialer := &ownerDialer{}
	secondLease, err := owner.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, secondDialer)
	if err != nil {
		t.Fatal(err)
	}
	if secondLease.Stamp.Generation != firstLease.Stamp.Generation+1 || !firstDialer.session.closed {
		t.Fatalf("generation first=%#v second=%#v firstClosed=%v", firstLease, secondLease, firstDialer.session.closed)
	}
	if _, err := owner.ExecuteApplication(context.Background(), firstLease.Stamp, &apipb.CommandEnvelope{}); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("stale execute error = %v", err)
	}
	result, err := owner.ExecuteApplication(context.Background(), secondLease.Stamp, &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	})
	if err != nil || result.GetTerminalList() == nil || secondDialer.session.calls != 1 {
		t.Fatalf("current execute result=%#v err=%v calls=%d", result, err, secondDialer.session.calls)
	}
	if err := owner.Disconnect(context.Background(), DisconnectRequest{Stamp: firstLease.Stamp}); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("stale disconnect error = %v", err)
	}
	if secondDialer.session.closed {
		t.Fatal("stale disconnect closed current session")
	}
}

func TestSessionOwnerClosesLateAttemptAfterNewGeneration(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	started := make(chan struct{})
	release := make(chan struct{})
	lateDialer := &ownerDialer{started: started, release: release}
	errCh := make(chan error, 1)
	go func() {
		_, err := owner.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, lateDialer)
		errCh <- err
	}()
	<-started
	winner := &ownerDialer{}
	if _, err := owner.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, winner); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-errCh; CodeOf(err) != ErrorStaleSession {
		t.Fatalf("late attempt error = %v", err)
	}
	if lateDialer.session == nil || !lateDialer.session.closed || winner.session.closed {
		t.Fatalf("lateClosed=%v winnerClosed=%v", lateDialer.session != nil && lateDialer.session.closed, winner.session.closed)
	}
}

func TestSessionOwnerBeginRouteAttemptInvalidatesCurrentSession(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	dialer := &ownerDialer{}
	lease, err := owner.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, dialer)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := owner.BeginRouteAttempt(target, "cloud", ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Stamp().Generation != lease.Stamp.Generation+1 || !dialer.session.closed {
		t.Fatalf("lease=%#v attempt=%#v closed=%v", lease, attempt, dialer.session.closed)
	}
	if _, err := owner.ApplicationSession(lease); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("stale application session error = %v", err)
	}
}

func TestSessionOwnerBeginRouteAttemptsShareOneGeneration(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	target.Routes["direct"] = endpoint.AccessRoute{ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceUser, SignalingAddresses: []string{"127.0.0.1:41120"}, ICETCPAddresses: []string{"127.0.0.1:41121"}}
	attempts, err := owner.BeginRouteAttempts(target, []endpoint.RouteID{"direct", "cloud"}, ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 || attempts[0].Stamp().Generation != attempts[1].Stamp().Generation || attempts[0].Route().ID != "direct" || attempts[1].Route().ID != "cloud" {
		t.Fatalf("attempts = %#v", attempts)
	}
	if _, err := owner.BeginRouteAttempts(target, []endpoint.RouteID{"cloud", "cloud"}, ConnectIntentInteractive); CodeOf(err) != ErrorInvalidRequest {
		t.Fatalf("duplicate Route error = %v", err)
	}
}

func TestSessionOwnerInvalidAttemptDoesNotInvalidateCurrentSession(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	dialer := &ownerDialer{}
	lease, err := owner.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, dialer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.BeginRouteAttempt(target, "missing", ConnectIntentInteractive); err == nil {
		t.Fatal("invalid route attempt succeeded")
	}
	if dialer.session.closed {
		t.Fatal("invalid route attempt closed current session")
	}
	if _, err := owner.ApplicationSession(lease); err != nil {
		t.Fatalf("current lease became stale: %v", err)
	}
}

func TestSessionGenerationAuthoritySurvivesOwnerReplacement(t *testing.T) {
	authority := NewSessionGenerationAuthority()
	target := ownerEndpoint()
	first := NewSessionOwnerWithAuthority(authority)
	firstLease, err := first.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, &ownerDialer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second := NewSessionOwnerWithAuthority(authority)
	defer second.Close()
	secondLease, err := second.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, &ownerDialer{})
	if err != nil {
		t.Fatal(err)
	}
	if secondLease.Stamp.Generation != firstLease.Stamp.Generation+1 {
		t.Fatalf("first=%#v second=%#v", firstLease, secondLease)
	}
}

func TestSessionGenerationAuthorityRevokesSessionOwnedByAnotherHost(t *testing.T) {
	authority := NewSessionGenerationAuthority()
	target := ownerEndpoint()
	first := NewSessionOwnerWithAuthority(authority)
	defer first.Close()
	firstDialer := &ownerDialer{}
	firstLease, err := first.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, firstDialer)
	if err != nil {
		t.Fatal(err)
	}
	second := NewSessionOwnerWithAuthority(authority)
	defer second.Close()
	secondDialer := &ownerDialer{}
	if _, err := second.ConnectRoute(context.Background(), target, "cloud", ConnectIntentInteractive, secondDialer); err != nil {
		t.Fatal(err)
	}
	if !firstDialer.session.closed {
		t.Fatal("new host generation did not close previous host session")
	}
	if _, err := first.ExecuteApplication(context.Background(), firstLease.Stamp, &apipb.CommandEnvelope{}); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("previous host execute error = %v", err)
	}
}

func TestSessionOwnerAcquireRouteSharesConsumerLeases(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	firstDialer := &ownerDialer{}
	first, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", firstDialer)
	if err != nil {
		t.Fatal(err)
	}
	reuseDialer := &ownerDialer{}
	second, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", reuseDialer)
	if err != nil {
		t.Fatal(err)
	}
	third, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", reuseDialer)
	if err != nil {
		t.Fatal(err)
	}
	if first.Stamp() != second.Stamp() || second.Stamp() != third.Stamp() || firstDialer.calls != 1 || reuseDialer.calls != 0 {
		t.Fatalf("first=%#v second=%#v third=%#v dialCalls=%d/%d", first.Stamp(), second.Stamp(), third.Stamp(), firstDialer.calls, reuseDialer.calls)
	}
	firstDialer.session.eventCancelled = make(chan struct{}, 2)
	if _, err := first.ApplicationEvents(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := second.ApplicationEvents(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstDialer.session.eventCancelled:
	case <-time.After(time.Second):
		t.Fatal("released lease did not cancel its event subscription")
	}
	select {
	case <-firstDialer.session.eventCancelled:
		t.Fatal("releasing one lease cancelled another event subscription")
	case <-time.After(20 * time.Millisecond):
	}
	if firstDialer.session.closed {
		t.Fatal("consumer lease closed shared ready session")
	}
	if _, err := second.ExecuteApplication(context.Background(), &apipb.CommandEnvelope{}); err != nil {
		t.Fatal(err)
	}
	provider := third.(ResourceStreamSession)
	stream, err := provider.OpenResourceStream(&apipb.ResourceHandle{})
	if err != nil {
		t.Fatal(err)
	}
	_ = stream.Close()
	_ = second.Close()
	select {
	case <-firstDialer.session.eventCancelled:
	case <-time.After(time.Second):
		t.Fatal("second lease did not cancel its event subscription")
	}
	_ = third.Close()
	if firstDialer.session.closed {
		t.Fatal("final consumer release closed cached ready session")
	}
}

func TestSessionOwnerInvalidationClosesSharedGeneration(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	dialer := &ownerDialer{}
	first, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", dialer)
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{})
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("cancelled file open cleanup failed")
	if err := first.(ApplicationSessionInvalidator).InvalidateApplicationSession(cause); err != nil {
		t.Fatal(err)
	}
	if !dialer.session.closed {
		t.Fatal("session invalidation did not close the shared ready session")
	}
	for index, lease := range []ApplicationReadyPeerSession{first, second} {
		select {
		case <-lease.Done():
			if !errors.Is(lease.Err(), cause) {
				t.Fatalf("lease %d error=%v want=%v", index, lease.Err(), cause)
			}
		case <-time.After(time.Second):
			t.Fatalf("lease %d remained alive after generation invalidation", index)
		}
	}
	if _, err := second.ExecuteApplication(context.Background(), &apipb.CommandEnvelope{}); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("second lease execute error=%v", err)
	}
}

func TestSessionOwnerAcquireRouteDoesNotReuseEndedOrFailedReplacement(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	firstDialer := &ownerDialer{}
	first, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", firstDialer)
	if err != nil {
		t.Fatal(err)
	}
	_ = firstDialer.session.Close()
	select {
	case <-first.Done():
	case <-time.After(time.Second):
		t.Fatal("ended ready session did not close consumer lease")
	}
	secondDialer := &ownerDialer{}
	second, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", secondDialer)
	if err != nil {
		t.Fatal(err)
	}
	if second.Stamp().Generation <= first.Stamp().Generation {
		t.Fatalf("ended session generation was reused: first=%#v second=%#v", first.Stamp(), second.Stamp())
	}
	if _, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-b", &ownerDialer{err: io.ErrUnexpectedEOF}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("replacement error = %v", err)
	}
	thirdDialer := &ownerDialer{}
	third, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", thirdDialer)
	if err != nil {
		t.Fatal(err)
	}
	if thirdDialer.calls != 1 || third.Stamp().Generation <= second.Stamp().Generation {
		t.Fatalf("failed replacement reused stale session: second=%#v third=%#v calls=%d", second.Stamp(), third.Stamp(), thirdDialer.calls)
	}
	_ = second.Close()
	_ = third.Close()
}

func TestSessionOwnerDelayedOldDoneDoesNotCloseNewGenerationLease(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	firstDialer := &ownerDialer{delayDone: true}
	first, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", firstDialer)
	if err != nil {
		t.Fatal(err)
	}
	secondDialer := &ownerDialer{}
	second, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-b", secondDialer)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.(*sharedApplicationLease).active(); err == nil {
		t.Fatal("replaced old consumer lease remained active")
	}
	firstDialer.session.finishDone()
	select {
	case <-firstDialer.session.removeObserved:
	case <-time.After(time.Second):
		t.Fatal("old session Done cleanup was not observed")
	}
	if _, err := second.ExecuteApplication(context.Background(), &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	}); err != nil {
		t.Fatalf("old Done closed new generation lease: %v", err)
	}
	select {
	case <-second.Done():
		t.Fatalf("new generation lease finished after old Done: %v", second.Err())
	default:
	}
	_ = second.Close()
}

func TestSessionOwnerEndpointAcquireLockSurvivesCloseWithWaiter(t *testing.T) {
	owner := NewSessionOwner()
	target := ownerEndpoint()
	holderStarted := make(chan struct{})
	holderRelease := make(chan struct{})
	holderDialer := &ownerDialer{started: holderStarted, release: holderRelease}
	holderResult := make(chan error, 1)
	go func() {
		_, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", holderDialer)
		holderResult <- err
	}()
	<-holderStarted

	waiterDialer := &ownerDialer{}
	waiterCtx := newAcquireTestContext(false)
	waiterResult := make(chan error, 1)
	go func() {
		_, err := owner.AcquireRoute(waiterCtx, target, "cloud", ConnectIntentInteractive, "config-b", waiterDialer)
		waiterResult <- err
	}()
	<-waiterCtx.observed
	entry := endpointAcquireEntryForTest(t, owner, target.ID, 2)

	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-waiterResult:
		if CodeOf(err) != ErrorUnavailable || WasAttempted(err) {
			t.Fatalf("waiter error = %v code=%q attempted=%v", err, CodeOf(err), WasAttempted(err))
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not wake endpoint acquire waiter")
	}
	owner.mu.Lock()
	current := owner.acquireLocks[target.ID]
	refs := entry.refs
	closed := owner.closed
	owner.mu.Unlock()
	if !closed || current != entry || refs != 1 {
		t.Fatalf("after Close: closed=%v entry=%p refs=%d, want closed and entry=%p refs=1", closed, current, refs, entry)
	}
	if waiterDialer.calls != 0 {
		t.Fatalf("waiter dial calls = %d, want 0 for closed owner", waiterDialer.calls)
	}

	close(holderRelease)
	select {
	case err := <-holderResult:
		if CodeOf(err) != ErrorStaleSession {
			t.Fatalf("holder error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("holder deadlocked after Close")
	}
	if holderDialer.session == nil || holderDialer.session.closeCalls.Load() != 1 {
		t.Fatalf("holder session=%p close calls=%d, want one closed session", holderDialer.session, closeCalls(holderDialer.session))
	}
	owner.mu.Lock()
	finalRefs := entry.refs
	owner.mu.Unlock()
	if finalRefs != 0 {
		t.Fatalf("released entry refs = %d, want 0", finalRefs)
	}
	assertEndpointAcquireLocksEmpty(t, owner)
}

func TestSessionOwnerEndpointAcquireDeadlineReleasesWaiterRef(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	target := ownerEndpoint()
	holderStarted := make(chan struct{})
	holderRelease := make(chan struct{})
	holderResult := make(chan error, 1)
	go func() {
		_, err := owner.AcquireRoute(context.Background(), target, "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{started: holderStarted, release: holderRelease})
		holderResult <- err
	}()
	<-holderStarted

	waiterCtx := newAcquireTestContext(true)
	waiterDialer := &ownerDialer{}
	waiterResult := make(chan error, 1)
	go func() {
		_, err := owner.AcquireRoute(waiterCtx, target, "cloud", ConnectIntentInteractive, "config-b", waiterDialer)
		waiterResult <- err
	}()
	<-waiterCtx.observed
	entry := endpointAcquireEntryForTest(t, owner, target.ID, 2)
	waiterCtx.finish(context.DeadlineExceeded)

	select {
	case err := <-waiterResult:
		var runtimeErr *Error
		if !errors.As(err, &runtimeErr) || runtimeErr.Cause != context.DeadlineExceeded {
			t.Fatalf("deadline error = %#v, want preserved context cause", err)
		}
		if CodeOf(err) != ErrorCanceled || !errors.Is(err, context.DeadlineExceeded) || WasAttempted(err) {
			t.Fatalf("deadline error = %v code=%q attempted=%v", err, CodeOf(err), WasAttempted(err))
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not wake endpoint acquire waiter")
	}
	owner.mu.Lock()
	current := owner.acquireLocks[target.ID]
	refs := entry.refs
	owner.mu.Unlock()
	if current != entry || refs != 1 {
		t.Fatalf("after deadline: entry=%p refs=%d, want entry=%p refs=1", current, refs, entry)
	}
	if waiterDialer.calls != 0 {
		t.Fatalf("deadline waiter dial calls = %d, want 0", waiterDialer.calls)
	}

	close(holderRelease)
	select {
	case err := <-holderResult:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("holder did not finish after release")
	}
	assertEndpointAcquireLocksEmpty(t, owner)
}

func TestSessionOwnerEndpointAcquireChecksCanceledContextAfterToken(t *testing.T) {
	owner := NewSessionOwner()
	defer owner.Close()
	ctx := newAcquireTestContext(false)
	ctx.state.Store(1)
	dialer := &ownerDialer{}

	_, err := owner.AcquireRoute(ctx, ownerEndpoint(), "cloud", ConnectIntentInteractive, "config-a", dialer)
	if CodeOf(err) != ErrorCanceled || !errors.Is(err, context.Canceled) || WasAttempted(err) {
		t.Fatalf("post-token cancellation error = %v code=%q attempted=%v", err, CodeOf(err), WasAttempted(err))
	}
	if dialer.calls != 0 {
		t.Fatalf("post-token cancellation dial calls = %d, want 0", dialer.calls)
	}
	assertEndpointAcquireLocksEmpty(t, owner)
}

func TestSessionOwnerEndpointAcquireLocksReclaimedAcrossOutcomes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		owner := NewSessionOwner()
		defer owner.Close()
		lease, err := owner.AcquireRoute(context.Background(), ownerEndpoint(), "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{})
		if err != nil {
			t.Fatal(err)
		}
		defer lease.Close()
		assertEndpointAcquireLocksEmpty(t, owner)
	})

	t.Run("dial failure", func(t *testing.T) {
		owner := NewSessionOwner()
		defer owner.Close()
		_, err := owner.AcquireRoute(context.Background(), ownerEndpoint(), "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{err: io.ErrUnexpectedEOF})
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("dial error = %v", err)
		}
		assertEndpointAcquireLocksEmpty(t, owner)
	})

	t.Run("context cancel", func(t *testing.T) {
		owner := NewSessionOwner()
		defer owner.Close()
		ctx, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			_, err := owner.AcquireRoute(ctx, ownerEndpoint(), "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{started: started, waitForContext: true})
			result <- err
		}()
		<-started
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel error = %v", err)
		}
		assertEndpointAcquireLocksEmpty(t, owner)
	})

	t.Run("repeated endpoint", func(t *testing.T) {
		owner := NewSessionOwner()
		defer owner.Close()
		dialer := &ownerDialer{}
		for index := 0; index < 5; index++ {
			lease, err := owner.AcquireRoute(context.Background(), ownerEndpoint(), "cloud", ConnectIntentInteractive, "config-a", dialer)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			assertEndpointAcquireLocksEmpty(t, owner)
		}
		if dialer.calls != 1 {
			t.Fatalf("dial calls = %d, want 1", dialer.calls)
		}
	})

	t.Run("multiple endpoints", func(t *testing.T) {
		owner := NewSessionOwner()
		defer owner.Close()
		firstTarget := ownerEndpoint()
		firstTarget.ID = "first"
		secondTarget := ownerEndpoint()
		secondTarget.ID = "second"
		firstStarted, firstRelease := make(chan struct{}), make(chan struct{})
		secondStarted, secondRelease := make(chan struct{}), make(chan struct{})
		results := make(chan error, 2)
		go func() {
			_, err := owner.AcquireRoute(context.Background(), firstTarget, "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{started: firstStarted, release: firstRelease})
			results <- err
		}()
		go func() {
			_, err := owner.AcquireRoute(context.Background(), secondTarget, "cloud", ConnectIntentInteractive, "config-a", &ownerDialer{started: secondStarted, release: secondRelease})
			results <- err
		}()
		<-firstStarted
		<-secondStarted
		owner.mu.Lock()
		lockCount := len(owner.acquireLocks)
		firstEntry := owner.acquireLocks[firstTarget.ID]
		secondEntry := owner.acquireLocks[secondTarget.ID]
		firstRefs, secondRefs := 0, 0
		if firstEntry != nil {
			firstRefs = firstEntry.refs
		}
		if secondEntry != nil {
			secondRefs = secondEntry.refs
		}
		owner.mu.Unlock()
		if lockCount != 2 || firstRefs != 1 || secondRefs != 1 {
			t.Fatalf("active locks=%d refs=%d/%d, want 2 and 1/1", lockCount, firstRefs, secondRefs)
		}
		close(firstRelease)
		close(secondRelease)
		for range 2 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		assertEndpointAcquireLocksEmpty(t, owner)
	})
}

func waitForEndpointAcquireRefs(t *testing.T, owner *SessionOwner, endpointID endpoint.EndpointID, want int) *endpointAcquireEntry {
	t.Helper()
	timeout := time.After(time.Second)
	for {
		owner.mu.Lock()
		entry := owner.acquireLocks[endpointID]
		refs := 0
		if entry != nil {
			refs = entry.refs
		}
		owner.mu.Unlock()
		if refs == want {
			return entry
		}
		select {
		case <-timeout:
			t.Fatalf("endpoint %q acquire refs = %d, want %d", endpointID, refs, want)
		default:
			goruntime.Gosched()
		}
	}
}

func endpointAcquireEntryForTest(t *testing.T, owner *SessionOwner, endpointID endpoint.EndpointID, wantRefs int) *endpointAcquireEntry {
	t.Helper()
	owner.mu.Lock()
	defer owner.mu.Unlock()
	entry := owner.acquireLocks[endpointID]
	if entry == nil || entry.refs != wantRefs {
		t.Fatalf("endpoint %q acquire entry=%p refs=%d, want non-nil refs=%d", endpointID, entry, acquireRefs(entry), wantRefs)
	}
	return entry
}

func acquireRefs(entry *endpointAcquireEntry) int {
	if entry == nil {
		return 0
	}
	return entry.refs
}

type acquireTestContext struct {
	context.Context
	done        chan struct{}
	observed    chan struct{}
	observeOnce sync.Once
	state       atomic.Int32
	hasDeadline bool
}

func newAcquireTestContext(hasDeadline bool) *acquireTestContext {
	return &acquireTestContext{
		Context: context.Background(),
		done:    make(chan struct{}), observed: make(chan struct{}), hasDeadline: hasDeadline,
	}
}

func (ctx *acquireTestContext) Deadline() (time.Time, bool) {
	return time.Time{}, ctx.hasDeadline
}

func (ctx *acquireTestContext) Done() <-chan struct{} {
	ctx.observeOnce.Do(func() { close(ctx.observed) })
	return ctx.done
}

func (ctx *acquireTestContext) Err() error {
	switch ctx.state.Load() {
	case 1:
		return context.Canceled
	case 2:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func (ctx *acquireTestContext) finish(err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		ctx.state.Store(2)
	} else {
		ctx.state.Store(1)
	}
	close(ctx.done)
}

func assertEndpointAcquireLocksEmpty(t *testing.T, owner *SessionOwner) {
	t.Helper()
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if len(owner.acquireLocks) != 0 {
		t.Fatalf("endpoint acquire locks = %#v, want empty", owner.acquireLocks)
	}
}

func ownerEndpoint() endpoint.Endpoint {
	identity := endpoint.DaemonIdentity{DeviceID: "device-1", DeviceFingerprint: "fingerprint-1"}
	return endpoint.Endpoint{
		ID: "studio", DaemonIdentity: identity,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"cloud": {
				ID: "cloud", Kind: endpoint.RouteManagedWebRTC, Enabled: true, Source: endpoint.SourceCloud, PolicySource: endpoint.SourceUser,
				TargetDeviceID: identity.DeviceID, AccountProfileRef: "default", RelayMode: endpoint.RelayDirect,
			},
		},
	}
}

type ownerDialer struct {
	started        chan struct{}
	release        chan struct{}
	session        *ownerSession
	calls          int
	err            error
	delayDone      bool
	waitForContext bool
}

func (dialer *ownerDialer) Connect(ctx context.Context, request AttemptRequest) (ReadyPeerSession, error) {
	dialer.calls++
	if dialer.err != nil {
		return nil, dialer.err
	}
	if dialer.started != nil {
		close(dialer.started)
	}
	if dialer.waitForContext {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if dialer.release != nil {
		<-dialer.release
	}
	dialer.session = newOwnerSession(request.Stamp())
	identity := request.DaemonIdentity()
	if identity.Empty() {
		identity = endpoint.DaemonIdentity{DeviceID: "device-owner-test", DeviceFingerprint: "SHA256:device-owner-test"}
	}
	dialer.session.evidence = ReadyPeerSessionEvidence{
		Identity: identity, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: 1,
	}
	dialer.session.delayDone = dialer.delayDone
	return dialer.session, nil
}

type ownerSession struct {
	stamp          EndpointSessionStamp
	evidence       ReadyPeerSessionEvidence
	done           chan struct{}
	closeOnce      sync.Once
	closed         bool
	calls          int
	eventCancelled chan struct{}
	delayDone      bool
	removeObserved chan struct{}
	removeOnce     sync.Once
	closeCalls     atomic.Int32
}

func newOwnerSession(stamp EndpointSessionStamp) *ownerSession {
	return &ownerSession{stamp: stamp, done: make(chan struct{}), removeObserved: make(chan struct{})}
}

func (session *ownerSession) Stamp() EndpointSessionStamp         { return session.stamp }
func (session *ownerSession) ObservedPath() string                { return "direct" }
func (session *ownerSession) Readiness() ReadyPeerSessionEvidence { return session.evidence }
func (session *ownerSession) Done() <-chan struct{}               { return session.done }
func (session *ownerSession) ConnectionSnapshot(at time.Time) (ConnectionSnapshot, bool) {
	return ConnectionSnapshot{RouteID: session.stamp.RouteID, SampledAt: at, Connected: true}, true
}
func (session *ownerSession) Err() error {
	if session.closed {
		return io.EOF
	}
	return nil
}
func (session *ownerSession) Close() error {
	if session.closeCalls.Add(1) > 1 {
		session.removeOnce.Do(func() { close(session.removeObserved) })
	}
	session.closeOnce.Do(func() {
		session.closed = true
		if !session.delayDone {
			close(session.done)
		}
	})
	return nil
}
func (session *ownerSession) finishDone() {
	session.delayDone = false
	select {
	case <-session.done:
	default:
		close(session.done)
	}
}
func (session *ownerSession) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	session.calls++
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_TerminalList{TerminalList: &apipb.TerminalListResult{}}}, nil
}
func (session *ownerSession) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	out := make(chan *apipb.EventEnvelope)
	if session.eventCancelled != nil {
		go func() {
			<-ctx.Done()
			session.eventCancelled <- struct{}{}
			close(out)
		}()
	}
	return out, nil
}
func (session *ownerSession) OpenResourceStream(*apipb.ResourceHandle) (ResourceStream, error) {
	return ownerResourceStream{}, nil
}

type ownerResourceStream struct{}

func (ownerResourceStream) Receive(context.Context) (uint8, []byte, error) { return 0, nil, io.EOF }
func (ownerResourceStream) Send(context.Context, uint8, []byte) error      { return nil }
func (ownerResourceStream) Close() error                                   { return nil }

var _ ApplicationReadyPeerSession = (*ownerSession)(nil)
var _ ResourceStreamSession = (*ownerSession)(nil)
