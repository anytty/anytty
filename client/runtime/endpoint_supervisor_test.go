package runtime

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
)

func TestEndpointSupervisorProbesRetainedWinnerWithoutRedial(t *testing.T) {
	controller := newSupervisorController("studio", 7)
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)

	for index := 0; index < 500; index++ {
		if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Foreground: true}); err != nil {
			t.Fatal(err)
		}
	}
	waitForSupervisorCount(t, supervisor, func(value EndpointSupervisorProjection) bool { return value.ProbeCount == 2 })
	time.Sleep(20 * time.Millisecond)
	projection := onlySupervisorProjection(t, supervisor)
	if projection.ProbeCount != 2 || projection.DialCount != 0 || controller.connectCount() != 0 {
		t.Fatalf("projection = %#v, connect_count = %d", projection, controller.connectCount())
	}
}

func TestEndpointSupervisorSnapshotsRevisionAndDrainsWakeAtomically(t *testing.T) {
	controller := &blockingProbeSupervisorController{
		supervisorController: newSupervisorController("studio", 7),
		probeStarted:         make(chan struct{}),
		releaseProbe:         make(chan struct{}),
	}
	supervisor, err := NewEndpointSupervisor(controller, EndpointSupervisorOptions{
		ProbeTimeout: time.Second,
		DialTimeout:  time.Second,
		Backoff:      []time.Duration{time.Millisecond},
		Logf:         func(string, ...any) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()

	control := &endpointControl{
		endpointID:      "studio",
		wake:            make(chan struct{}, 1),
		done:            make(chan struct{}),
		mode:            EndpointSupervisorTakeover,
		demanded:        true,
		controlRevision: 1,
		changed:         make(chan struct{}),
	}
	supervisor.mu.Lock()
	supervisor.beginControlRevisionLocked(control)
	supervisor.endpoints[control.endpointID] = control
	supervisor.wakeLocked(control)
	go supervisor.runEndpoint(control)

	// With the old drain-then-snapshot transition, the worker can consume this
	// wake without the mutex and then block taking its snapshot. Yield until it
	// has had ample opportunity to reach that gap before advancing the control.
	for attempts := 0; attempts < 1_000 && len(control.wake) != 0; attempts++ {
		runtime.Gosched()
	}
	supervisor.advanceControlLocked(control)
	supervisor.mu.Unlock()

	select {
	case <-controller.probeStarted:
	case <-time.After(time.Second):
		t.Fatal("probe did not start")
	}
	if queued := len(control.wake); queued != 0 {
		t.Fatalf("wake queue contains %d stale control wake while revision 2 is being probed", queued)
	}
	close(controller.releaseProbe)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)
	if projection := onlySupervisorProjection(t, supervisor); projection.ControlRevision != 2 || projection.ProbeCount != 1 {
		t.Fatalf("projection = %#v, want one probe for revision 2", projection)
	}
}

func TestEndpointSupervisorRepairAlwaysReprobesOneReadyEndpoint(t *testing.T) {
	controller := newSupervisorController("studio", 7)
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)

	before := onlySupervisorProjection(t, supervisor)
	if err := supervisor.Repair("studio"); err != nil {
		t.Fatal(err)
	}
	waitForSupervisorCount(t, supervisor, func(value EndpointSupervisorProjection) bool {
		return value.Phase == EndpointSupervisorReady &&
			value.ControlRevision == before.ControlRevision+1 &&
			value.ProbeCount == before.ProbeCount+1
	})
	projection := onlySupervisorProjection(t, supervisor)
	if projection.DialCount != before.DialCount || controller.connectCount() != 0 {
		t.Fatalf("projection = %#v, connect_count = %d", projection, controller.connectCount())
	}
}

func TestEndpointSupervisorRepairFencesReadySynchronously(t *testing.T) {
	supervisor, cancel := newDormantEndpointSupervisor(
		newSupervisorController("studio", 7),
		EndpointSupervisorReady,
		"",
		"",
	)
	defer cancel()
	if err := supervisor.Repair("studio"); err != nil {
		t.Fatal(err)
	}
	projection := onlySupervisorProjection(t, supervisor)
	if projection.Phase != EndpointSupervisorReconciling || projection.ControlRevision != 2 {
		t.Fatalf("projection after repair = %#v", projection)
	}
}

func TestEndpointSupervisorRepairRejectsEndpointWithoutTakeoverDemand(t *testing.T) {
	supervisor := newTestEndpointSupervisor(t, newSupervisorController("studio", 0), nil)
	defer supervisor.Close()
	if err := supervisor.Repair("studio"); !errors.Is(err, ErrEndpointNotManaged) {
		t.Fatalf("Repair error = %v, want ErrEndpointNotManaged", err)
	}
}

func TestEndpointSupervisorSignalFencesReadyBeforeWaitReady(t *testing.T) {
	supervisor, cancel := newDormantEndpointSupervisor(
		newSupervisorController("studio", 7),
		EndpointSupervisorReady,
		"",
		"",
	)
	defer cancel()
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Foreground: true}); err != nil {
		t.Fatal(err)
	}
	projection := onlySupervisorProjection(t, supervisor)
	if projection.Phase != EndpointSupervisorReconciling || projection.ErrorCode != "" {
		t.Fatalf("projection after signal = %#v", projection)
	}

	ctx, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	if err := supervisor.WaitReady(ctx); CodeOf(err) != ErrorCanceled || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitReady error = %v, want current-revision timeout", err)
	}
}

func TestEndpointSupervisorSignalFencesReadyBeforeAcquire(t *testing.T) {
	supervisor, cancel := newDormantEndpointSupervisor(
		newSupervisorController("studio", 7),
		EndpointSupervisorReady,
		"",
		"",
	)
	defer cancel()
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Foreground: true}); err != nil {
		t.Fatal(err)
	}

	ctx, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	if _, err := supervisor.Acquire(ctx, "studio"); CodeOf(err) != ErrorCanceled || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire error = %v, want current-revision timeout", err)
	}
}

func TestEndpointSupervisorAcquireRejectsReadyLeaseWhenSignalWinsDuringAcquire(t *testing.T) {
	controller := &blockingAcquireSupervisorController{
		physical: newSupervisorPhysicalSession("studio", 7),
		started:  make(chan struct{}),
		release:  make(chan struct{}),
	}
	supervisor, cancel := newDormantEndpointSupervisor(
		controller,
		EndpointSupervisorReady,
		"",
		"",
	)
	defer cancel()
	control := supervisor.endpoints["studio"]
	control.stamp = controller.physical.stamp

	result := make(chan error, 1)
	go func() {
		ctx, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer stop()
		lease, err := supervisor.Acquire(ctx, "studio")
		if lease != nil {
			_ = lease.Close()
		}
		result <- err
	}()
	select {
	case <-controller.started:
	case <-time.After(time.Second):
		t.Fatal("AcquireCurrent did not start")
	}
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Foreground: true}); err != nil {
		t.Fatal(err)
	}
	close(controller.release)
	select {
	case err := <-result:
		if CodeOf(err) != ErrorCanceled || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Acquire error = %v, want current-revision timeout", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Acquire did not finish")
	}
}

func TestEndpointSupervisorSignalFencesBlockedFailure(t *testing.T) {
	supervisor, cancel := newDormantEndpointSupervisor(
		newSupervisorController("studio", 0),
		EndpointSupervisorBlocked,
		ErrorAuthorization,
		"authorization expired",
	)
	defer cancel()
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Foreground: true}); err != nil {
		t.Fatal(err)
	}
	projection := onlySupervisorProjection(t, supervisor)
	if projection.Phase != EndpointSupervisorReconciling || projection.ErrorCode != "" || projection.Message != "" {
		t.Fatalf("projection after signal = %#v", projection)
	}

	ctx, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	if err := supervisor.WaitReady(ctx); CodeOf(err) != ErrorCanceled || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitReady error = %v, want current-revision timeout", err)
	}
}

func TestEndpointSupervisorDoesNotAutomaticallyRecoverExplicitStop(t *testing.T) {
	if recoverableSupervisorFailure(&Error{Code: ErrorUserStopped, Message: "disconnected by administrator"}) {
		t.Fatal("explicit stop was classified as an automatic recovery failure")
	}
	if !recoverableSupervisorFailure(&Error{Code: ErrorUnavailable, Message: "network changed"}) {
		t.Fatal("transient unavailable failure stopped automatic recovery")
	}
}

func TestEndpointSupervisorStaleNoDemandPublishCannotOverwriteNewRevision(t *testing.T) {
	supervisor, cancel := newDormantEndpointSupervisor(
		newSupervisorController("studio", 7),
		EndpointSupervisorReady,
		"",
		"",
	)
	defer cancel()
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Foreground: true}); err != nil {
		t.Fatal(err)
	}
	control := supervisor.endpoints["studio"]
	supervisor.publish(control, 1, EndpointSupervisorNoDemand, EndpointSessionStamp{}, nil)
	projection := onlySupervisorProjection(t, supervisor)
	if projection.ControlRevision != 2 || projection.Phase != EndpointSupervisorReconciling {
		t.Fatalf("projection after stale no-demand publish = %#v", projection)
	}
}

func TestEndpointSupervisorNoDemandInvalidatesRetainedPhysicalWinner(t *testing.T) {
	controller := newSupervisorController("studio", 7)
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)

	controller.mu.Lock()
	retired := controller.current
	controller.mu.Unlock()
	if err := supervisor.ReplaceDemand(EndpointDemandSnapshot{
		AttachmentID:   "renderer",
		DemandRevision: 2,
	}); err != nil {
		t.Fatal(err)
	}
	waitForInvalidationCount(t, controller, 1)

	if invalidated := controller.invalidatedStamps(); len(invalidated) != 1 || invalidated[0] != retired.stamp {
		t.Fatalf("invalidated = %#v, want %#v", invalidated, retired.stamp)
	}
	select {
	case <-retired.done:
	default:
		t.Fatal("no-demand transition retained the physical winner")
	}
	controller.mu.Lock()
	current := controller.current
	controller.mu.Unlock()
	if current != nil {
		t.Fatalf("current winner = %#v, want nil", current.stamp)
	}
}

func TestEndpointSupervisorNoDemandCannotInvalidateConcurrentNewWinner(t *testing.T) {
	controller := &blockingInvalidateSupervisorController{
		supervisorController: newSupervisorController("studio", 7),
		started:              make(chan EndpointSessionStamp, 1),
		release:              make(chan struct{}),
		done:                 make(chan error, 1),
	}
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)

	if err := supervisor.ReplaceDemand(EndpointDemandSnapshot{
		AttachmentID:   "renderer",
		DemandRevision: 2,
	}); err != nil {
		t.Fatal(err)
	}
	var retired EndpointSessionStamp
	select {
	case retired = <-controller.started:
	case <-time.After(time.Second):
		t.Fatal("no-demand invalidation did not start")
	}

	replaceTestDemand(t, supervisor, 3, EndpointSupervisorTakeover)
	newWinner := newSupervisorPhysicalSession("studio", retired.Generation+1)
	controller.mu.Lock()
	controller.generation = newWinner.stamp.Generation
	controller.current = newWinner
	controller.mu.Unlock()
	close(controller.release)
	select {
	case err := <-controller.done:
		if CodeOf(err) != ErrorStaleSession {
			t.Fatalf("old-generation invalidation error = %v, want stale_session", err)
		}
	case <-time.After(time.Second):
		t.Fatal("no-demand invalidation did not finish")
	}

	if invalidated := controller.invalidatedStamps(); len(invalidated) != 0 {
		t.Fatalf("invalidated = %#v, want no successful invalidation", invalidated)
	}
	select {
	case <-newWinner.done:
		t.Fatal("old no-demand transition closed the concurrent new winner")
	default:
	}
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)
	if projection := onlySupervisorProjection(t, supervisor); projection.SessionStamp != newWinner.stamp {
		t.Fatalf("projection = %#v, want concurrent new winner %#v", projection, newWinner.stamp)
	}
}

func TestEndpointSupervisorNoDemandInvalidatesLateLeaseFromRetiredRevision(t *testing.T) {
	controller := &delayedAcquireSupervisorController{
		supervisorController: newSupervisorController("studio", 7),
		started:              make(chan struct{}),
		release:              make(chan struct{}),
	}
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	select {
	case <-controller.started:
	case <-time.After(time.Second):
		t.Fatal("AcquireCurrent did not start")
	}

	if err := supervisor.ReplaceDemand(EndpointDemandSnapshot{
		AttachmentID:   "renderer",
		DemandRevision: 2,
	}); err != nil {
		t.Fatal(err)
	}
	close(controller.release)
	waitForInvalidationCount(t, controller.supervisorController, 1)

	if invalidated := controller.invalidatedStamps(); len(invalidated) != 1 || invalidated[0].Generation != 7 {
		t.Fatalf("invalidated = %#v, want retired generation 7", invalidated)
	}
}

func TestEndpointSupervisorInvalidatesExactWinnerAndAutomaticallyRecovers(t *testing.T) {
	controller := newSupervisorController("studio", 4)
	controller.probeFailures = []error{io.EOF, nil}
	faults := &supervisorFaults{failDialCount: 1}
	supervisor := newTestEndpointSupervisor(t, controller, faults)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lease, err := supervisor.Acquire(ctx, "studio")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if lease.Stamp().Generation != 5 {
		t.Fatalf("generation = %d, want 5", lease.Stamp().Generation)
	}
	projection := onlySupervisorProjection(t, supervisor)
	if projection.Phase != EndpointSupervisorReady || projection.DialCount != 2 || projection.BackoffCount != 2 {
		t.Fatalf("projection = %#v", projection)
	}
	if invalidated := controller.invalidatedStamps(); len(invalidated) != 1 || invalidated[0].Generation != 4 {
		t.Fatalf("invalidated = %#v", invalidated)
	}
}

func TestEndpointSupervisorWaitsOfflineAndDialsOnceWhenPathReturns(t *testing.T) {
	controller := newSupervisorController("studio", 0)
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: false, Reason: "offline"}); err != nil {
		t.Fatal(err)
	}
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorWaitingNetwork)
	if controller.connectCount() != 0 {
		t.Fatalf("offline connect count = %d", controller.connectCount())
	}
	waitCtx, stopWaiting := context.WithTimeout(context.Background(), 20*time.Millisecond)
	if err := supervisor.WaitReady(waitCtx); CodeOf(err) != ErrorCanceled || !errors.Is(err, context.DeadlineExceeded) {
		stopWaiting()
		t.Fatalf("offline WaitReady error = %v, want timeout", err)
	}
	stopWaiting()
	if err := supervisor.Signal(EndpointHostSignal{Revision: 2, Connected: true, Reason: "available"}); err != nil {
		t.Fatal(err)
	}
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)
	if controller.connectCount() != 1 {
		t.Fatalf("connect count = %d, want 1", controller.connectCount())
	}
}

func newDormantEndpointSupervisor(
	controller EndpointSupervisorController,
	phase EndpointSupervisorPhase,
	errorCode ErrorCode,
	message string,
) (*EndpointSupervisor, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	control := &endpointControl{
		endpointID:      "studio",
		wake:            make(chan struct{}, 1),
		done:            make(chan struct{}),
		mode:            EndpointSupervisorTakeover,
		demanded:        true,
		controlRevision: 1,
		phaseRevision:   1,
		phase:           phase,
		errorCode:       errorCode,
		message:         message,
		changed:         make(chan struct{}),
	}
	return &EndpointSupervisor{
		controller: controller,
		ctx:        ctx,
		cancel:     cancel,
		options: EndpointSupervisorOptions{
			Logf: func(string, ...any) {},
		},
		connected: true,
		endpoints: map[endpoint.EndpointID]*endpointControl{"studio": control},
		changed:   make(chan struct{}),
	}, cancel
}

func TestEndpointSupervisorShadowModeOnlyRecordsDecision(t *testing.T) {
	controller := newSupervisorController("studio", 0)
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorShadow)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorShadowDecision)
	projection := onlySupervisorProjection(t, supervisor)
	if projection.ProbeCount != 0 || projection.DialCount != 0 || controller.connectCount() != 0 {
		t.Fatalf("shadow projection = %#v", projection)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := supervisor.Acquire(ctx, "studio"); !errors.Is(err, ErrEndpointNotManaged) {
		t.Fatalf("Acquire error = %v", err)
	}
}

func TestEndpointSupervisorPermanentFailureBlocksWithoutRetry(t *testing.T) {
	controller := newSupervisorController("studio", 0)
	controller.connectFailures = []error{runtimeError(ErrorAuthorization, "authorization expired", nil)}
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorBlocked)
	time.Sleep(20 * time.Millisecond)
	projection := onlySupervisorProjection(t, supervisor)
	if projection.DialCount != 1 || projection.ErrorCode != ErrorAuthorization {
		t.Fatalf("projection = %#v", projection)
	}
}

func TestEndpointSupervisorInvalidatesDialThatCompletesAfterControlChange(t *testing.T) {
	controller := newSupervisorController("studio", 0)
	controller.blockFirstConnect = make(chan struct{})
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	deadline := time.Now().Add(time.Second)
	for controller.connectCount() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if controller.connectCount() != 1 {
		t.Fatal("first dial did not start")
	}
	if err := supervisor.Signal(EndpointHostSignal{Revision: 1, Connected: true, Reason: "network_replaced"}); err != nil {
		t.Fatal(err)
	}
	close(controller.blockFirstConnect)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)

	invalidated := controller.invalidatedStamps()
	if len(invalidated) != 1 || invalidated[0].Generation != 1 {
		t.Fatalf("invalidated = %#v, want late generation 1", invalidated)
	}
	if projection := onlySupervisorProjection(t, supervisor); projection.SessionStamp.Generation != 2 {
		t.Fatalf("projection = %#v", projection)
	}
}

func TestEndpointSupervisorBacksOffWhenNewWinnerFailsApplicationProbe(t *testing.T) {
	controller := newSupervisorController("studio", 0)
	controller.probeFailures = []error{io.EOF, nil}
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorTakeover)
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)

	projection := onlySupervisorProjection(t, supervisor)
	if projection.DialCount != 2 || projection.BackoffCount != 1 || projection.ControlRevision < 2 {
		t.Fatalf("projection = %#v", projection)
	}
	if invalidated := controller.invalidatedStamps(); len(invalidated) != 1 || invalidated[0].Generation != 1 {
		t.Fatalf("invalidated = %#v", invalidated)
	}
}

func TestEndpointSupervisorWaitReadyObservesBlockedEndpointWhileAnotherConnects(t *testing.T) {
	controller := &blockingSupervisorController{
		blockedStarted: make(chan struct{}),
		slowStarted:    make(chan struct{}),
		releaseBlocked: make(chan struct{}),
	}
	supervisor := newTestEndpointSupervisor(t, controller, nil)
	defer supervisor.Close()
	if err := supervisor.ReplaceDemand(EndpointDemandSnapshot{
		AttachmentID:   "test",
		DemandRevision: 1,
		Endpoints: []EndpointDemand{
			{EndpointID: "blocked", Mode: EndpointSupervisorTakeover},
			{EndpointID: "slow", Mode: EndpointSupervisorTakeover},
		},
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-controller.blockedStarted:
	case <-time.After(time.Second):
		t.Fatal("blocked endpoint did not start connecting")
	}
	select {
	case <-controller.slowStarted:
	case <-time.After(time.Second):
		t.Fatal("slow endpoint did not start connecting")
	}

	result := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		result <- supervisor.WaitReady(ctx)
	}()
	close(controller.releaseBlocked)
	select {
	case err := <-result:
		if CodeOf(err) != ErrorAuthorization {
			t.Fatalf("WaitReady error = %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("WaitReady did not observe the blocked endpoint")
	}
}

func TestEndpointSupervisorAcquireAfterCloseReturnsUnavailable(t *testing.T) {
	supervisor := newTestEndpointSupervisor(t, newSupervisorController("studio", 0), nil)
	if err := supervisor.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := supervisor.Acquire(ctx, "studio"); CodeOf(err) != ErrorUnavailable {
		t.Fatalf("Acquire error = %v", err)
	}
}

func TestEndpointSupervisorRejectsConflictingDemandAtSameRevision(t *testing.T) {
	supervisor := newTestEndpointSupervisor(t, newSupervisorController("studio", 0), nil)
	defer supervisor.Close()
	replaceTestDemand(t, supervisor, 1, EndpointSupervisorShadow)
	err := supervisor.ReplaceDemand(EndpointDemandSnapshot{
		AttachmentID:   "replacement",
		DemandRevision: 1,
		Endpoints:      []EndpointDemand{{EndpointID: "studio", Mode: EndpointSupervisorTakeover}},
	})
	if CodeOf(err) != ErrorStaleResource {
		t.Fatalf("ReplaceDemand error = %v", err)
	}
}

type blockingSupervisorController struct {
	blockedStarted chan struct{}
	slowStarted    chan struct{}
	releaseBlocked chan struct{}
}

type blockingAcquireSupervisorController struct {
	physical *supervisorPhysicalSession
	started  chan struct{}
	release  chan struct{}
}

type blockingProbeSupervisorController struct {
	*supervisorController
	probeStarted     chan struct{}
	probeStartedOnce sync.Once
	releaseProbe     chan struct{}
}

type blockingInvalidateSupervisorController struct {
	*supervisorController
	started chan EndpointSessionStamp
	release chan struct{}
	done    chan error
}

type delayedAcquireSupervisorController struct {
	*supervisorController
	started chan struct{}
	release chan struct{}
}

func (controller *delayedAcquireSupervisorController) AcquireCurrent(endpointID endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	close(controller.started)
	<-controller.release
	return controller.supervisorController.AcquireCurrent(endpointID)
}

func (controller *blockingInvalidateSupervisorController) Invalidate(stamp EndpointSessionStamp, failure error) error {
	controller.started <- stamp
	<-controller.release
	err := controller.supervisorController.Invalidate(stamp, failure)
	controller.done <- err
	return err
}

func (controller *blockingProbeSupervisorController) Probe(ctx context.Context, _ ApplicationReadyPeerSession) error {
	controller.probeStartedOnce.Do(func() { close(controller.probeStarted) })
	select {
	case <-controller.releaseProbe:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (controller *blockingAcquireSupervisorController) AcquireCurrent(endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	close(controller.started)
	<-controller.release
	return &supervisorLease{physical: controller.physical}, nil
}

func (controller *blockingAcquireSupervisorController) Connect(context.Context, endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	return nil, runtimeError(ErrorUnavailable, "unexpected connect", nil)
}

func (controller *blockingAcquireSupervisorController) Probe(context.Context, ApplicationReadyPeerSession) error {
	return nil
}

func (controller *blockingAcquireSupervisorController) Invalidate(EndpointSessionStamp, error) error {
	return nil
}

func (controller *blockingSupervisorController) AcquireCurrent(endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	return nil, runtimeError(ErrorNotFound, "no current session", nil)
}

func (controller *blockingSupervisorController) Connect(ctx context.Context, endpointID endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	switch endpointID {
	case "blocked":
		close(controller.blockedStarted)
		select {
		case <-controller.releaseBlocked:
			return nil, runtimeError(ErrorAuthorization, "authorization expired", nil)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case "slow":
		close(controller.slowStarted)
		<-ctx.Done()
		return nil, ctx.Err()
	default:
		return nil, runtimeError(ErrorNotFound, "unknown endpoint", nil)
	}
}

func (controller *blockingSupervisorController) Probe(context.Context, ApplicationReadyPeerSession) error {
	return nil
}

func (controller *blockingSupervisorController) Invalidate(EndpointSessionStamp, error) error {
	return nil
}

type supervisorController struct {
	mu                sync.Mutex
	endpointID        endpoint.EndpointID
	generation        SessionGeneration
	current           *supervisorPhysicalSession
	probeFailures     []error
	connectFailures   []error
	connects          int
	invalidated       []EndpointSessionStamp
	blockFirstConnect chan struct{}
}

func newSupervisorController(endpointID endpoint.EndpointID, generation SessionGeneration) *supervisorController {
	controller := &supervisorController{endpointID: endpointID, generation: generation}
	if generation != 0 {
		controller.current = newSupervisorPhysicalSession(endpointID, generation)
	}
	return controller
}

func (controller *supervisorController) AcquireCurrent(endpointID endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if endpointID != controller.endpointID || controller.current == nil {
		return nil, runtimeError(ErrorNotFound, "no current session", nil)
	}
	select {
	case <-controller.current.done:
		return nil, runtimeError(ErrorNotFound, "current session ended", nil)
	default:
	}
	return &supervisorLease{physical: controller.current}, nil
}

func (controller *supervisorController) Connect(ctx context.Context, endpointID endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	controller.mu.Lock()
	controller.connects++
	connectIndex := controller.connects
	if len(controller.connectFailures) > 0 {
		failure := controller.connectFailures[0]
		controller.connectFailures = controller.connectFailures[1:]
		controller.mu.Unlock()
		return nil, failure
	}
	controller.generation++
	controller.current = newSupervisorPhysicalSession(endpointID, controller.generation)
	current := controller.current
	block := controller.blockFirstConnect
	controller.mu.Unlock()
	if connectIndex == 1 && block != nil {
		<-block
		return &supervisorLease{physical: current}, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return &supervisorLease{physical: current}, nil
	}
}

func (controller *supervisorController) Probe(ctx context.Context, _ ApplicationReadyPeerSession) error {
	controller.mu.Lock()
	var failure error
	if len(controller.probeFailures) > 0 {
		failure = controller.probeFailures[0]
		controller.probeFailures = controller.probeFailures[1:]
	}
	controller.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return failure
	}
}

func (controller *supervisorController) Invalidate(stamp EndpointSessionStamp, _ error) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.current == nil || controller.current.stamp != stamp {
		return runtimeError(ErrorStaleSession, "stale invalidate", nil)
	}
	controller.invalidated = append(controller.invalidated, stamp)
	controller.current.close()
	controller.current = nil
	return nil
}

func (controller *supervisorController) connectCount() int {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.connects
}

func (controller *supervisorController) invalidatedStamps() []EndpointSessionStamp {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return append([]EndpointSessionStamp(nil), controller.invalidated...)
}

func waitForInvalidationCount(t *testing.T, controller *supervisorController, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for len(controller.invalidatedStamps()) < count && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if actual := len(controller.invalidatedStamps()); actual != count {
		t.Fatalf("invalidation count = %d, want %d", actual, count)
	}
}

type supervisorPhysicalSession struct {
	stamp EndpointSessionStamp
	done  chan struct{}
	once  sync.Once
}

func newSupervisorPhysicalSession(endpointID endpoint.EndpointID, generation SessionGeneration) *supervisorPhysicalSession {
	return &supervisorPhysicalSession{
		stamp: EndpointSessionStamp{EndpointID: endpointID, RouteID: "route", Generation: generation},
		done:  make(chan struct{}),
	}
}

func (session *supervisorPhysicalSession) close() { session.once.Do(func() { close(session.done) }) }

type supervisorLease struct{ physical *supervisorPhysicalSession }

func (lease *supervisorLease) Stamp() EndpointSessionStamp { return lease.physical.stamp }
func (lease *supervisorLease) ObservedPath() string        { return "direct" }
func (lease *supervisorLease) Readiness() ReadyPeerSessionEvidence {
	return ReadyPeerSessionEvidence{IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: 1}
}
func (lease *supervisorLease) Done() <-chan struct{} { return lease.physical.done }
func (lease *supervisorLease) Err() error            { return nil }
func (lease *supervisorLease) Close() error          { return nil }
func (lease *supervisorLease) ExecuteApplication(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{}}}, nil
}
func (lease *supervisorLease) ApplicationEvents(context.Context) (<-chan *apipb.EventEnvelope, error) {
	return make(chan *apipb.EventEnvelope), nil
}

type supervisorFaults struct {
	mu            sync.Mutex
	failDialCount int
}

func (faults *supervisorFaults) Before(action EndpointSupervisorAction, _ endpoint.EndpointID, _ uint64) error {
	faults.mu.Lock()
	defer faults.mu.Unlock()
	if action == EndpointSupervisorActionDial && faults.failDialCount > 0 {
		faults.failDialCount--
		return runtimeError(ErrorUnavailable, "injected transient dial failure", nil)
	}
	return nil
}

func newTestEndpointSupervisor(t *testing.T, controller EndpointSupervisorController, faults EndpointSupervisorFaultInjector) *EndpointSupervisor {
	t.Helper()
	supervisor, err := NewEndpointSupervisor(controller, EndpointSupervisorOptions{
		ProbeTimeout: time.Second,
		DialTimeout:  time.Second,
		Backoff:      []time.Duration{0, time.Millisecond},
		Logf:         func(string, ...any) {},
		Faults:       faults,
	})
	if err != nil {
		t.Fatal(err)
	}
	return supervisor
}

func replaceTestDemand(t *testing.T, supervisor *EndpointSupervisor, revision uint64, mode EndpointSupervisorMode) {
	t.Helper()
	if err := supervisor.ReplaceDemand(EndpointDemandSnapshot{
		AttachmentID:   "renderer",
		DemandRevision: revision,
		Endpoints:      []EndpointDemand{{EndpointID: "studio", Mode: mode}},
	}); err != nil {
		t.Fatal(err)
	}
}

func waitSupervisorPhase(t *testing.T, supervisor *EndpointSupervisor, phase EndpointSupervisorPhase) {
	t.Helper()
	waitForSupervisorCount(t, supervisor, func(value EndpointSupervisorProjection) bool { return value.Phase == phase })
}

func waitForSupervisorCount(t *testing.T, supervisor *EndpointSupervisor, ready func(EndpointSupervisorProjection) bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		projection := supervisor.Snapshot()
		if len(projection) == 1 && ready(projection[0]) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("supervisor did not converge: %#v", supervisor.Snapshot())
}

func onlySupervisorProjection(t *testing.T, supervisor *EndpointSupervisor) EndpointSupervisorProjection {
	t.Helper()
	projection := supervisor.Snapshot()
	if len(projection) != 1 {
		t.Fatalf("projection count = %d", len(projection))
	}
	return projection[0]
}
