package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/shared/transport"
)

func TestGrantRevokePersistenceFailureLeavesEpochTransportsAndAdmissionUnchanged(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	injected := errors.New("persist failed")
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-a": expiresAt})
	service.setRevokeError("grant-a", injected)
	revokeGate := service.gateRevoke("grant-a")
	server := newGrantTransportTestServer(service, clock, nil)

	firstRaw, first := admitGrantTestTransport(t, server, "grant-a", expiresAt)
	operation := first.grantOperation.Load()
	epoch := grantOperationEpoch(operation)
	revokeDone := make(chan error, 1)
	go func() {
		_, err := server.revokeClientAccess(context.Background(), "grant-a")
		revokeDone <- err
	}()
	waitGrantStage(t, revokeGate.entered)
	if got := grantOperationEpoch(operation); got != epoch {
		t.Fatalf("epoch advanced before persistence completed: got %d, want %d", got, epoch)
	}
	if firstRaw.closeCount.Load() != 0 {
		t.Fatalf("transport closed before persistence completed: %d", firstRaw.closeCount.Load())
	}
	revokeGate.open()
	if err := waitGrantResult(t, revokeDone); !errors.Is(err, injected) {
		t.Fatalf("revoke error = %v", err)
	}
	if got := grantOperationEpoch(operation); got != epoch {
		t.Fatalf("persist failure advanced epoch: got %d, want %d", got, epoch)
	}
	if firstRaw.closeCount.Load() != 0 {
		t.Fatalf("persist failure closed active transport %d times", firstRaw.closeCount.Load())
	}
	secondRaw, second := admitGrantTestTransport(t, server, "grant-a", expiresAt)

	closeGrantTestTransport(server, first)
	closeGrantTestTransport(server, second)
	assertGrantCloseCount(t, firstRaw, 1)
	assertGrantCloseCount(t, secondRaw, 1)
	assertGrantServerEmpty(t, server)
}

func TestGrantRevokeClosesAllMatchingTransportsExactlyOnceAndIsolatesOtherGrant(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-a": expiresAt, "grant-b": expiresAt})
	server := newGrantTransportTestServer(service, clock, nil)

	raws := make([]*grantCountingTransport, 0, 3)
	tracked := make([]*trackedTransport, 0, 3)
	for range 3 {
		raw, connection := admitGrantTestTransport(t, server, "grant-a", expiresAt)
		raws, tracked = append(raws, raw), append(tracked, connection)
	}
	isolatedRaw, isolated := admitGrantTestTransport(t, server, "grant-b", expiresAt)

	if record, err := server.revokeClientAccess(context.Background(), "grant-a"); err != nil || record.GrantID != "grant-a" {
		t.Fatalf("revoke record = %#v, error = %v", record, err)
	}
	for index, raw := range raws {
		assertGrantCloseCount(t, raw, 1)
		assertGrantReverseIndexCleared(t, tracked[index])
	}
	if isolatedRaw.closeCount.Load() != 0 {
		t.Fatalf("other grant transport closed %d times", isolatedRaw.closeCount.Load())
	}

	rejected := newGrantCountingTransport()
	if _, err := beginAndAdmitGrantTestTransport(server, rejected, grantTestScope("grant-a", expiresAt)); !errors.Is(err, ErrGrantTransportInactive) {
		t.Fatalf("post-revoke admission error = %v", err)
	}
	assertGrantCloseCount(t, rejected, 1)
	for index, connection := range tracked {
		closeGrantTestTransport(server, connection)
		assertGrantCloseCount(t, raws[index], 1)
	}
	closeGrantTestTransport(server, isolated)
	assertGrantCloseCount(t, isolatedRaw, 1)
	assertGrantServerEmpty(t, server)
}

func TestSlowGrantRevokeDoesNotDelayAnotherGrantExpiry(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	clock := newGrantTestClock(now)
	aExpiresAt := now.Add(2 * time.Hour)
	bExpiresAt := now.Add(time.Hour)
	service := newGrantAccessTestService(map[string]time.Time{
		"grant-a": aExpiresAt,
		"grant-b": bExpiresAt,
	})
	revokeGate := service.gateRevoke("grant-a")
	timers := &grantManualTimerFactory{}
	server := newGrantTransportTestServer(service, clock, timers.afterFunc)
	aRaw, aTracked := admitGrantTestTransport(t, server, "grant-a", aExpiresAt)
	bRaw, bTracked := admitGrantTestTransport(t, server, "grant-b", bExpiresAt)
	bTimer := grantOperationTimer(t, bTracked.grantOperation.Load())

	revokeDone := make(chan error, 1)
	go func() {
		_, err := server.revokeClientAccess(context.Background(), "grant-a")
		revokeDone <- err
	}()
	waitGrantStage(t, revokeGate.entered)

	clock.Set(bExpiresAt)
	bTimer.fire(t)
	assertGrantCloseCount(t, bRaw, 1)
	assertGrantReverseIndexCleared(t, bTracked)
	if aRaw.closeCount.Load() != 0 {
		t.Fatalf("blocked grant revoke affected its transport before persistence: %d", aRaw.closeCount.Load())
	}
	assertGrantNoResult(t, revokeDone)

	revokeGate.open()
	if err := waitGrantResult(t, revokeDone); err != nil {
		t.Fatalf("revoke grant-a: %v", err)
	}
	assertGrantCloseCount(t, aRaw, 1)
	assertGrantReverseIndexCleared(t, aTracked)
	closeGrantTestTransport(server, aTracked)
	closeGrantTestTransport(server, bTracked)
	assertGrantServerEmpty(t, server)
}

func TestSlowGrantActiveDoesNotDelayAnotherGrantExpiry(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	clock := newGrantTestClock(now)
	aExpiresAt := now.Add(2 * time.Hour)
	bExpiresAt := now.Add(time.Hour)
	service := newGrantAccessTestService(map[string]time.Time{
		"grant-a": aExpiresAt,
		"grant-b": bExpiresAt,
	})
	timers := &grantManualTimerFactory{}
	server := newGrantTransportTestServer(service, clock, timers.afterFunc)
	bRaw, bTracked := admitGrantTestTransport(t, server, "grant-b", bExpiresAt)
	bTimer := grantOperationTimer(t, bTracked.grantOperation.Load())
	activeGate := service.gateActive("grant-a")
	aRaw := newGrantCountingTransport()
	admitDone := make(chan grantAdmissionResult, 1)
	go func() {
		tracked, err := beginAndAdmitGrantTestTransport(server, aRaw, grantTestScope("grant-a", aExpiresAt))
		admitDone <- grantAdmissionResult{tracked: tracked, err: err}
	}()
	waitGrantStage(t, activeGate.entered)

	clock.Set(bExpiresAt)
	bTimer.fire(t)
	assertGrantCloseCount(t, bRaw, 1)
	assertGrantReverseIndexCleared(t, bTracked)
	assertGrantNoAdmission(t, admitDone)

	activeGate.open()
	admission := waitGrantAdmission(t, admitDone)
	if admission.err != nil {
		t.Fatalf("admit grant-a after query release: %v", admission.err)
	}
	if aRaw.closeCount.Load() != 0 {
		t.Fatalf("grant-a transport closed %d times", aRaw.closeCount.Load())
	}
	closeGrantTestTransport(server, admission.tracked)
	closeGrantTestTransport(server, bTracked)
	assertGrantCloseCount(t, aRaw, 1)
	assertGrantServerEmpty(t, server)
}

func TestGrantAdmissionRechecksAbsoluteExpiryAfterStoreQuery(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-boundary": expiresAt})
	activeGate := service.gateActive("grant-boundary")
	server := newGrantTransportTestServer(service, clock, nil)
	raw := newGrantCountingTransport()
	admitDone := make(chan grantAdmissionResult, 1)
	go func() {
		tracked, err := beginAndAdmitGrantTestTransport(server, raw, grantTestScope("grant-boundary", expiresAt))
		admitDone <- grantAdmissionResult{tracked: tracked, err: err}
	}()
	waitGrantStage(t, activeGate.entered)

	clock.Set(expiresAt)
	activeGate.open()
	admission := waitGrantAdmission(t, admitDone)
	if !errors.Is(admission.err, ErrGrantTransportInactive) {
		t.Fatalf("expiry-boundary admission error = %v", admission.err)
	}
	if admission.tracked != nil {
		t.Fatal("expired admission returned tracked transport")
	}
	assertGrantCloseCount(t, raw, 1)
	assertGrantServerEmpty(t, server)
}

func TestGrantAdmissionBeforeDuringAndAfterSuccessfulRevoke(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-race": expiresAt})
	server := newGrantTransportTestServer(service, clock, nil)
	beforeRaw, before := admitGrantTestTransport(t, server, "grant-race", expiresAt)
	revokeGate := service.gateRevoke("grant-race")
	activeGate := service.gateActive("grant-race")

	revokeDone := make(chan error, 1)
	go func() {
		_, err := server.revokeClientAccess(context.Background(), "grant-race")
		revokeDone <- err
	}()
	waitGrantStage(t, revokeGate.entered)
	duringRaw := newGrantCountingTransport()
	admitDone := make(chan grantAdmissionResult, 1)
	go func() {
		tracked, err := beginAndAdmitGrantTestTransport(server, duringRaw, grantTestScope("grant-race", expiresAt))
		admitDone <- grantAdmissionResult{tracked: tracked, err: err}
	}()
	waitGrantStage(t, activeGate.entered)
	activeGate.open()
	during := waitGrantAdmission(t, admitDone)
	if during.err != nil {
		t.Fatalf("admission while persistence blocked: %v", during.err)
	}
	if beforeRaw.closeCount.Load() != 0 || duringRaw.closeCount.Load() != 0 {
		t.Fatal("successful persistence has not returned but a transport was closed")
	}

	revokeGate.open()
	if err := waitGrantResult(t, revokeDone); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	assertGrantCloseCount(t, beforeRaw, 1)
	assertGrantCloseCount(t, duringRaw, 1)
	assertGrantReverseIndexCleared(t, before)
	assertGrantReverseIndexCleared(t, during.tracked)
	afterRaw := newGrantCountingTransport()
	if _, err := beginAndAdmitGrantTestTransport(server, afterRaw, grantTestScope("grant-race", expiresAt)); !errors.Is(err, ErrGrantTransportInactive) {
		t.Fatalf("post-revoke admission error = %v", err)
	}
	assertGrantCloseCount(t, afterRaw, 1)

	closeGrantTestTransport(server, before)
	closeGrantTestTransport(server, during.tracked)
	assertGrantCloseCount(t, beforeRaw, 1)
	assertGrantCloseCount(t, duringRaw, 1)
	assertGrantServerEmpty(t, server)
}

func TestGrantAdmissionRejectsStaleActiveResultAfterRevokeEpoch(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-stale": expiresAt})
	server := newGrantTransportTestServer(service, clock, nil)
	beforeRaw, before := admitGrantTestTransport(t, server, "grant-stale", expiresAt)
	operation := before.grantOperation.Load()
	initialEpoch := grantOperationEpoch(operation)
	activeReturnGate := service.gateActiveReturn("grant-stale")

	duringRaw := newGrantCountingTransport()
	admitDone := make(chan grantAdmissionResult, 1)
	go func() {
		tracked, err := beginAndAdmitGrantTestTransport(server, duringRaw, grantTestScope("grant-stale", expiresAt))
		admitDone <- grantAdmissionResult{tracked: tracked, err: err}
	}()
	waitGrantStage(t, activeReturnGate.entered)
	if _, err := server.revokeClientAccess(context.Background(), "grant-stale"); err != nil {
		t.Fatalf("revoke after active store snapshot: %v", err)
	}
	if got := grantOperationEpoch(operation); got != initialEpoch+1 {
		t.Fatalf("revoke epoch = %d, want %d", got, initialEpoch+1)
	}
	assertGrantCloseCount(t, beforeRaw, 1)
	assertGrantReverseIndexCleared(t, before)
	assertGrantNoAdmission(t, admitDone)

	activeReturnGate.open()
	admission := waitGrantAdmission(t, admitDone)
	if !errors.Is(admission.err, ErrGrantTransportInactive) {
		t.Fatalf("stale active admission error = %v", admission.err)
	}
	if admission.tracked != nil {
		t.Fatal("stale active admission returned tracked transport")
	}
	assertGrantCloseCount(t, duringRaw, 1)
	closeGrantTestTransport(server, before)
	assertGrantCloseCount(t, beforeRaw, 1)
	assertGrantServerEmpty(t, server)
}

func TestGrantAbsoluteExpirySharesOneCancelableTimerAndClosesAtBoundary(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-expiry": expiresAt})
	timers := &grantManualTimerFactory{}
	server := newGrantTransportTestServer(service, clock, timers.afterFunc)

	firstRaw, first := admitGrantTestTransport(t, server, "grant-expiry", expiresAt)
	secondRaw, second := admitGrantTestTransport(t, server, "grant-expiry", expiresAt)
	if timers.count() != 1 {
		t.Fatalf("timers = %d, want one per active grant", timers.count())
	}
	clock.Set(expiresAt)
	grantOperationTimer(t, first.grantOperation.Load()).fire(t)
	assertGrantCloseCount(t, firstRaw, 1)
	assertGrantCloseCount(t, secondRaw, 1)
	assertGrantReverseIndexCleared(t, first)
	assertGrantReverseIndexCleared(t, second)

	rejected := newGrantCountingTransport()
	if _, err := beginAndAdmitGrantTestTransport(server, rejected, grantTestScope("grant-expiry", expiresAt)); !errors.Is(err, ErrGrantTransportInactive) {
		t.Fatalf("expiry-boundary admission error = %v", err)
	}
	assertGrantCloseCount(t, rejected, 1)
	if timers.count() != 1 {
		t.Fatalf("expiry created replacement timer, count = %d", timers.count())
	}
	closeGrantTestTransport(server, first)
	closeGrantTestTransport(server, second)
	assertGrantServerEmpty(t, server)
}

func TestGrantDisconnectCancelsLastTimerAndPrunesOperation(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(map[string]time.Time{"grant-disconnect": expiresAt})
	timers := &grantManualTimerFactory{}
	server := newGrantTransportTestServer(service, clock, timers.afterFunc)
	raw, tracked := admitGrantTestTransport(t, server, "grant-disconnect", expiresAt)
	timer := grantOperationTimer(t, tracked.grantOperation.Load())

	closeGrantTestTransport(server, tracked)
	assertGrantCloseCount(t, raw, 1)
	if got := timer.stopCount.Load(); got != 1 {
		t.Fatalf("timer stop count = %d, want 1", got)
	}
	assertGrantServerEmpty(t, server)
}

func TestGrantRevokeExpiryDisconnectShutdownRaceExactOnceCount40(t *testing.T) {
	for iteration := range 40 {
		now := time.Date(2026, 7, 30, 12, 0, 0, iteration, time.UTC)
		expiresAt := now.Add(time.Hour)
		grantID := fmt.Sprintf("grant-race-%d", iteration)
		clock := newGrantTestClock(now)
		service := newGrantAccessTestService(map[string]time.Time{grantID: expiresAt})
		timers := &grantManualTimerFactory{}
		server := newGrantTransportTestServer(service, clock, timers.afterFunc)
		raw, tracked := admitGrantTestTransport(t, server, grantID, expiresAt)
		operation := tracked.grantOperation.Load()
		entry := grantOperationTimerEntry(t, operation)
		clock.Set(expiresAt)

		start := make(chan struct{})
		var wait sync.WaitGroup
		wait.Add(4)
		go func() {
			defer wait.Done()
			<-start
			closeGrantTestTransport(server, tracked)
		}()
		go func() {
			defer wait.Done()
			<-start
			_, _ = server.revokeClientAccess(context.Background(), grantID)
		}()
		go func() {
			defer wait.Done()
			<-start
			server.expireGrantOperation(operation, entry)
		}()
		go func() {
			defer wait.Done()
			<-start
			_ = server.Shutdown(context.Background())
		}()
		close(start)
		wait.Wait()
		closeGrantTestTransport(server, tracked)
		assertGrantCloseCount(t, raw, 1)
		assertGrantServerEmpty(t, server)
	}
}

func TestGrantOperationTimerAndTransportStateReturnsToZero(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	clock := newGrantTestClock(now)
	service := newGrantAccessTestService(nil)
	timers := &grantManualTimerFactory{}
	server := newGrantTransportTestServer(service, clock, timers.afterFunc)

	const grants = 96
	for index := range grants {
		grantID := fmt.Sprintf("grant-sequential-%d", index)
		expiresAt := clock.Now().Add(time.Hour)
		service.addGrant(grantID, expiresAt)
		raw, tracked := admitGrantTestTransport(t, server, grantID, expiresAt)
		timer := grantOperationTimer(t, tracked.grantOperation.Load())
		switch index % 3 {
		case 0:
			closeGrantTestTransport(server, tracked)
		case 1:
			if _, err := server.revokeClientAccess(context.Background(), grantID); err != nil {
				t.Fatalf("revoke %s: %v", grantID, err)
			}
			assertGrantReverseIndexCleared(t, tracked)
			closeGrantTestTransport(server, tracked)
		case 2:
			clock.Set(expiresAt)
			timer.fire(t)
			assertGrantReverseIndexCleared(t, tracked)
			closeGrantTestTransport(server, tracked)
		}
		assertGrantCloseCount(t, raw, 1)
		if got := timer.stopCount.Load(); got != 1 {
			t.Fatalf("grant %d timer stop count = %d, want 1", index, got)
		}
		assertGrantServerEmpty(t, server)
	}

	if timers.count() != grants {
		t.Fatalf("created timers = %d, want %d", timers.count(), grants)
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	assertGrantServerEmpty(t, server)
}

type grantStageGate struct {
	entered     chan struct{}
	release     chan struct{}
	enteredOnce sync.Once
	releaseOnce sync.Once
}

func newGrantStageGate() *grantStageGate {
	return &grantStageGate{entered: make(chan struct{}), release: make(chan struct{})}
}

func (gate *grantStageGate) wait() {
	if gate == nil {
		return
	}
	gate.enteredOnce.Do(func() { close(gate.entered) })
	<-gate.release
}

func (gate *grantStageGate) open() {
	gate.releaseOnce.Do(func() { close(gate.release) })
}

type grantAccessTestService struct {
	mu            sync.Mutex
	grants        map[string]time.Time
	revoked       map[string]bool
	revokeErrors  map[string]error
	activeGates   map[string]*grantStageGate
	activeReturns map[string]*grantStageGate
	revokeGates   map[string]*grantStageGate
}

func newGrantAccessTestService(grants map[string]time.Time) *grantAccessTestService {
	service := &grantAccessTestService{
		grants:        make(map[string]time.Time),
		revoked:       make(map[string]bool),
		revokeErrors:  make(map[string]error),
		activeGates:   make(map[string]*grantStageGate),
		activeReturns: make(map[string]*grantStageGate),
		revokeGates:   make(map[string]*grantStageGate),
	}
	for grantID, expiresAt := range grants {
		service.grants[grantID] = expiresAt
	}
	return service
}

func (service *grantAccessTestService) Identity(context.Context, []byte) (ClientAccessIdentity, error) {
	return ClientAccessIdentity{}, nil
}

func (service *grantAccessTestService) CreateTicket(context.Context, ClientAccessTicketRequest) (ClientAccessTicket, error) {
	return ClientAccessTicket{}, nil
}

func (service *grantAccessTestService) List(context.Context) ([]ClientAccessRecord, error) {
	return nil, nil
}

func (service *grantAccessTestService) GrantActive(_ context.Context, grantID string, expiresAt, now time.Time) bool {
	service.mu.Lock()
	gate := service.activeGates[grantID]
	service.mu.Unlock()
	gate.wait()

	service.mu.Lock()
	stored, ok := service.grants[grantID]
	active := ok && !service.revoked[grantID] && stored.Equal(expiresAt) && now.Before(stored)
	returnGate := service.activeReturns[grantID]
	service.mu.Unlock()
	returnGate.wait()
	return active
}

func (service *grantAccessTestService) Revoke(_ context.Context, grantID string) (ClientAccessRecord, error) {
	service.mu.Lock()
	gate := service.revokeGates[grantID]
	service.mu.Unlock()
	gate.wait()

	service.mu.Lock()
	defer service.mu.Unlock()
	if err := service.revokeErrors[grantID]; err != nil {
		return ClientAccessRecord{}, err
	}
	expiresAt, ok := service.grants[grantID]
	if !ok {
		return ClientAccessRecord{}, errors.New("client access grant not found")
	}
	service.revoked[grantID] = true
	return ClientAccessRecord{GrantID: grantID, ExpiresAt: expiresAt, RevokedAt: expiresAt.Add(-time.Minute)}, nil
}

func (service *grantAccessTestService) addGrant(grantID string, expiresAt time.Time) {
	service.mu.Lock()
	service.grants[grantID] = expiresAt
	service.mu.Unlock()
}

func (service *grantAccessTestService) setRevokeError(grantID string, err error) {
	service.mu.Lock()
	service.revokeErrors[grantID] = err
	service.mu.Unlock()
}

func (service *grantAccessTestService) gateActive(grantID string) *grantStageGate {
	gate := newGrantStageGate()
	service.mu.Lock()
	service.activeGates[grantID] = gate
	service.mu.Unlock()
	return gate
}

func (service *grantAccessTestService) gateActiveReturn(grantID string) *grantStageGate {
	gate := newGrantStageGate()
	service.mu.Lock()
	service.activeReturns[grantID] = gate
	service.mu.Unlock()
	return gate
}

func (service *grantAccessTestService) gateRevoke(grantID string) *grantStageGate {
	gate := newGrantStageGate()
	service.mu.Lock()
	service.revokeGates[grantID] = gate
	service.mu.Unlock()
	return gate
}

type grantTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func newGrantTestClock(now time.Time) *grantTestClock {
	return &grantTestClock{now: now}
}

func (clock *grantTestClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *grantTestClock) Set(now time.Time) {
	clock.mu.Lock()
	clock.now = now
	clock.mu.Unlock()
}

type grantCountingTransport struct {
	done       chan struct{}
	doneOnce   sync.Once
	closeCount atomic.Int32
}

func newGrantCountingTransport() *grantCountingTransport {
	return &grantCountingTransport{done: make(chan struct{})}
}

func (*grantCountingTransport) Send([]byte) error     { return io.EOF }
func (*grantCountingTransport) Recv() ([]byte, error) { return nil, io.EOF }

func (transport *grantCountingTransport) Close() error {
	transport.closeCount.Add(1)
	transport.doneOnce.Do(func() { close(transport.done) })
	return nil
}

func (transport *grantCountingTransport) Done() <-chan struct{} { return transport.done }

type grantManualTimer struct {
	callback  func()
	stopCount atomic.Int32
}

func (timer *grantManualTimer) Stop() bool {
	return timer.stopCount.Add(1) == 1
}

func (timer *grantManualTimer) fire(t *testing.T) {
	t.Helper()
	if timer.callback == nil {
		t.Fatal("manual grant timer has no callback")
	}
	timer.callback()
}

type grantManualTimerFactory struct {
	mu     sync.Mutex
	timers []*grantManualTimer
}

func (factory *grantManualTimerFactory) afterFunc(_ time.Duration, callback func()) grantTimer {
	timer := &grantManualTimer{callback: callback}
	factory.mu.Lock()
	factory.timers = append(factory.timers, timer)
	factory.mu.Unlock()
	return timer
}

func (factory *grantManualTimerFactory) count() int {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	return len(factory.timers)
}

type grantAdmissionResult struct {
	tracked *trackedTransport
	err     error
}

func newGrantTransportTestServer(service ClientAccessService, clock *grantTestClock, afterFunc func(time.Duration, func()) grantTimer) *Server {
	server := NewServer(WithClientAccessService(service))
	server.grantNow = clock.Now
	if afterFunc != nil {
		server.grantAfterFunc = afterFunc
	}
	return server
}

func grantTestScope(grantID string, expiresAt time.Time) TransportScope {
	return TransportScope{GrantID: grantID, GrantExpiresAt: expiresAt, PrincipalID: "subject", AllowDaemon: true}
}

func admitGrantTestTransport(t *testing.T, server *Server, grantID string, expiresAt time.Time) (*grantCountingTransport, *trackedTransport) {
	t.Helper()
	raw := newGrantCountingTransport()
	tracked, err := beginAndAdmitGrantTestTransport(server, raw, grantTestScope(grantID, expiresAt))
	if err != nil {
		t.Fatalf("admit %s: %v", grantID, err)
	}
	return raw, tracked
}

func closeGrantTestTransport(server *Server, tracked *trackedTransport) {
	if tracked == nil {
		return
	}
	server.finishTrackedTransport(tracked)
}

func beginAndAdmitGrantTestTransport(server *Server, raw transport.Transport, scope TransportScope) (*trackedTransport, error) {
	tracked, err := server.beginTrackedTransport(raw)
	if err != nil {
		return nil, err
	}
	if err := server.admitTransport(context.Background(), tracked, scope); err != nil {
		server.finishTrackedTransport(tracked)
		return nil, err
	}
	return tracked, nil
}

func grantOperationEpoch(operation *grantOperation) uint64 {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	return operation.epoch
}

func grantOperationTimerEntry(t *testing.T, operation *grantOperation) *grantTimerEntry {
	t.Helper()
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.timer == nil {
		t.Fatal("grant operation has no timer")
	}
	return operation.timer
}

func grantOperationTimer(t *testing.T, operation *grantOperation) *grantManualTimer {
	t.Helper()
	entry := grantOperationTimerEntry(t, operation)
	timer, ok := entry.timer.(*grantManualTimer)
	if !ok {
		t.Fatalf("timer type = %T, want *grantManualTimer", entry.timer)
	}
	return timer
}

func assertGrantCloseCount(t *testing.T, transport *grantCountingTransport, want int32) {
	t.Helper()
	if got := transport.closeCount.Load(); got != want {
		t.Fatalf("transport close count = %d, want %d", got, want)
	}
}

func assertGrantReverseIndexCleared(t *testing.T, tracked *trackedTransport) {
	t.Helper()
	if operation := tracked.grantOperation.Load(); operation != nil {
		t.Fatalf("transport retained reverse grant index for %q", operation.grantID)
	}
}

func assertGrantServerEmpty(t *testing.T, server *Server) {
	t.Helper()
	server.grantOperationsMu.Lock()
	operationCount := len(server.grantOperations)
	var refs, transports, timers int
	for _, operation := range server.grantOperations {
		refs += operation.refs
		operation.mu.Lock()
		transports += len(operation.transports)
		if operation.timer != nil {
			timers++
		}
		operation.mu.Unlock()
	}
	server.grantOperationsMu.Unlock()
	server.mu.Lock()
	globalTransports := len(server.transports)
	server.mu.Unlock()
	if operationCount != 0 || refs != 0 || transports != 0 || timers != 0 || globalTransports != 0 {
		t.Fatalf("grant state retained: operations=%d refs=%d indexed=%d timers=%d transports=%d", operationCount, refs, transports, timers, globalTransports)
	}
}

func waitGrantStage(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for grant service stage")
	}
}

func waitGrantResult[T any](t *testing.T, result <-chan T) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for grant operation")
		var zero T
		return zero
	}
}

func waitGrantAdmission(t *testing.T, result <-chan grantAdmissionResult) grantAdmissionResult {
	t.Helper()
	return waitGrantResult(t, result)
}

func assertGrantNoResult[T any](t *testing.T, result <-chan T) {
	t.Helper()
	select {
	case value := <-result:
		t.Fatalf("grant operation completed while its store stage was blocked: %#v", value)
	default:
	}
}

func assertGrantNoAdmission(t *testing.T, result <-chan grantAdmissionResult) {
	t.Helper()
	assertGrantNoResult(t, result)
}
