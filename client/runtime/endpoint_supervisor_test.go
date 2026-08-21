package runtime

import (
	"context"
	"errors"
	"io"
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
	if err := supervisor.Signal(EndpointHostSignal{Revision: 2, Connected: true, Reason: "available"}); err != nil {
		t.Fatal(err)
	}
	waitSupervisorPhase(t, supervisor, EndpointSupervisorReady)
	if controller.connectCount() != 1 {
		t.Fatalf("connect count = %d, want 1", controller.connectCount())
	}
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
