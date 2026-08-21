package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
)

var ErrEndpointNotManaged = errors.New("endpoint is not managed by the Go supervisor")

type EndpointSupervisorMode string

const (
	EndpointSupervisorShadow   EndpointSupervisorMode = "shadow"
	EndpointSupervisorTakeover EndpointSupervisorMode = "takeover"
)

type EndpointSupervisorPhase string

const (
	EndpointSupervisorNoDemand       EndpointSupervisorPhase = "no_demand"
	EndpointSupervisorShadowDecision EndpointSupervisorPhase = "shadow_decision"
	EndpointSupervisorWaitingNetwork EndpointSupervisorPhase = "waiting_network"
	EndpointSupervisorVerifying      EndpointSupervisorPhase = "verifying"
	EndpointSupervisorConnecting     EndpointSupervisorPhase = "connecting"
	EndpointSupervisorBackoff        EndpointSupervisorPhase = "backoff"
	EndpointSupervisorBlocked        EndpointSupervisorPhase = "blocked"
	EndpointSupervisorReady          EndpointSupervisorPhase = "ready"
)

type EndpointSupervisorAction string

const (
	EndpointSupervisorActionProbe      EndpointSupervisorAction = "probe"
	EndpointSupervisorActionInvalidate EndpointSupervisorAction = "invalidate"
	EndpointSupervisorActionDial       EndpointSupervisorAction = "dial"
)

type EndpointDemand struct {
	EndpointID endpoint.EndpointID
	Mode       EndpointSupervisorMode
}

type EndpointDemandSnapshot struct {
	AttachmentID   string
	DemandRevision uint64
	Endpoints      []EndpointDemand
}

type EndpointHostSignal struct {
	Revision   uint64
	Connected  bool
	Reason     string
	Foreground bool
}

type EndpointSupervisorProjection struct {
	EndpointID      endpoint.EndpointID
	Mode            EndpointSupervisorMode
	Phase           EndpointSupervisorPhase
	ControlRevision uint64
	AttemptID       uint64
	SessionStamp    EndpointSessionStamp
	ErrorCode       ErrorCode
	Message         string
	ProbeCount      uint64
	DialCount       uint64
	BackoffCount    uint64
}

type EndpointSupervisorController interface {
	AcquireCurrent(endpoint.EndpointID) (ApplicationReadyPeerSession, error)
	Connect(context.Context, endpoint.EndpointID) (ApplicationReadyPeerSession, error)
	Probe(context.Context, ApplicationReadyPeerSession) error
	Invalidate(EndpointSessionStamp, error) error
}

type EndpointSupervisorFaultInjector interface {
	Before(EndpointSupervisorAction, endpoint.EndpointID, uint64) error
}

type EndpointSupervisorOptions struct {
	ProbeTimeout time.Duration
	DialTimeout  time.Duration
	Backoff      []time.Duration
	Random       *rand.Rand
	Logf         func(string, ...any)
	Faults       EndpointSupervisorFaultInjector
}

type EndpointSupervisor struct {
	controller EndpointSupervisorController
	ctx        context.Context
	cancel     context.CancelFunc
	options    EndpointSupervisorOptions

	mu             sync.Mutex
	closed         bool
	demandRevision uint64
	hostRevision   uint64
	connected      bool
	endpoints      map[endpoint.EndpointID]*endpointControl
	changed        chan struct{}
}

type endpointControl struct {
	endpointID endpoint.EndpointID
	wake       chan struct{}
	done       chan struct{}

	mode            EndpointSupervisorMode
	demanded        bool
	controlRevision uint64
	attemptID       uint64
	phase           EndpointSupervisorPhase
	stamp           EndpointSessionStamp
	errorCode       ErrorCode
	message         string
	probeCount      uint64
	dialCount       uint64
	backoffCount    uint64
	backoffIndex    int
	maintenance     ApplicationReadyPeerSession
	attemptCancel   context.CancelFunc
	changed         chan struct{}
}

type endpointControlSnapshot struct {
	mode            EndpointSupervisorMode
	demanded        bool
	connected       bool
	controlRevision uint64
}

func NewEndpointSupervisor(controller EndpointSupervisorController, options EndpointSupervisorOptions) (*EndpointSupervisor, error) {
	if controller == nil {
		return nil, runtimeError(ErrorInvalidRequest, "endpoint supervisor controller is required", nil)
	}
	if options.ProbeTimeout <= 0 {
		options.ProbeTimeout = 4 * time.Second
	}
	if options.DialTimeout <= 0 {
		options.DialTimeout = 45 * time.Second
	}
	if len(options.Backoff) == 0 {
		options.Backoff = []time.Duration{0, 500 * time.Millisecond, 2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second}
	} else {
		options.Backoff = append([]time.Duration(nil), options.Backoff...)
	}
	if options.Logf == nil {
		options.Logf = log.Printf
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &EndpointSupervisor{
		controller: controller,
		ctx:        ctx,
		cancel:     cancel,
		options:    options,
		connected:  true,
		endpoints:  make(map[endpoint.EndpointID]*endpointControl),
		changed:    make(chan struct{}),
	}, nil
}

func (supervisor *EndpointSupervisor) ReplaceDemand(snapshot EndpointDemandSnapshot) error {
	if supervisor == nil {
		return runtimeError(ErrorUnavailable, "endpoint supervisor is unavailable", nil)
	}
	seen := make(map[endpoint.EndpointID]EndpointSupervisorMode, len(snapshot.Endpoints))
	for _, demand := range snapshot.Endpoints {
		id := endpoint.EndpointID(strings.TrimSpace(string(demand.EndpointID)))
		if id == "" {
			return runtimeError(ErrorInvalidRequest, "endpoint supervisor demand contains an empty endpoint_id", nil)
		}
		if demand.Mode != EndpointSupervisorShadow && demand.Mode != EndpointSupervisorTakeover {
			return runtimeError(ErrorInvalidRequest, fmt.Sprintf("endpoint %q has invalid supervisor mode %q", id, demand.Mode), nil)
		}
		if _, duplicate := seen[id]; duplicate {
			return runtimeError(ErrorInvalidRequest, fmt.Sprintf("endpoint supervisor demand contains duplicate endpoint %q", id), nil)
		}
		seen[id] = demand.Mode
	}

	supervisor.mu.Lock()
	if supervisor.closed {
		supervisor.mu.Unlock()
		return runtimeError(ErrorUnavailable, "endpoint supervisor is closed", nil)
	}
	if snapshot.DemandRevision < supervisor.demandRevision {
		supervisor.mu.Unlock()
		return runtimeError(ErrorStaleResource, "endpoint supervisor demand revision is stale", nil)
	}
	if snapshot.DemandRevision == supervisor.demandRevision && supervisor.demandRevision != 0 {
		if !supervisor.demandMatchesLocked(seen) {
			supervisor.mu.Unlock()
			return runtimeError(ErrorStaleResource, "endpoint supervisor demand revision conflicts with the current snapshot", nil)
		}
		supervisor.mu.Unlock()
		return nil
	}
	supervisor.demandRevision = snapshot.DemandRevision
	for id, control := range supervisor.endpoints {
		mode, demanded := seen[id]
		if control.demanded == demanded && (!demanded || control.mode == mode) {
			continue
		}
		control.demanded = demanded
		if demanded {
			control.mode = mode
		}
		supervisor.advanceControlLocked(control)
	}
	for id, mode := range seen {
		if supervisor.endpoints[id] != nil {
			continue
		}
		control := &endpointControl{
			endpointID:      id,
			wake:            make(chan struct{}, 1),
			done:            make(chan struct{}),
			mode:            mode,
			demanded:        true,
			controlRevision: 1,
			phase:           EndpointSupervisorNoDemand,
			changed:         make(chan struct{}),
		}
		supervisor.endpoints[id] = control
		go supervisor.runEndpoint(control)
		supervisor.wakeLocked(control)
	}
	supervisor.mu.Unlock()
	return nil
}

func (supervisor *EndpointSupervisor) Signal(signal EndpointHostSignal) error {
	if supervisor == nil || signal.Revision == 0 {
		return runtimeError(ErrorInvalidRequest, "endpoint supervisor host revision is required", nil)
	}
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.closed {
		return runtimeError(ErrorUnavailable, "endpoint supervisor is closed", nil)
	}
	if signal.Revision <= supervisor.hostRevision {
		return nil
	}
	supervisor.hostRevision = signal.Revision
	supervisor.connected = signal.Connected
	for _, control := range supervisor.endpoints {
		if !control.demanded {
			continue
		}
		supervisor.advanceControlLocked(control)
	}
	return nil
}

func (supervisor *EndpointSupervisor) Mode(endpointID endpoint.EndpointID) EndpointSupervisorMode {
	if supervisor == nil {
		return ""
	}
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	control := supervisor.endpoints[endpointID]
	if control == nil || !control.demanded {
		return ""
	}
	return control.mode
}

func (supervisor *EndpointSupervisor) Acquire(ctx context.Context, endpointID endpoint.EndpointID) (ApplicationReadyPeerSession, error) {
	if supervisor == nil || ctx == nil {
		return nil, runtimeError(ErrorInvalidRequest, "endpoint supervisor and context are required", nil)
	}
	for {
		supervisor.mu.Lock()
		control := supervisor.endpoints[endpointID]
		if supervisor.closed {
			supervisor.mu.Unlock()
			return nil, runtimeError(ErrorUnavailable, "endpoint supervisor is closed", nil)
		}
		if control == nil || !control.demanded || control.mode != EndpointSupervisorTakeover {
			supervisor.mu.Unlock()
			return nil, ErrEndpointNotManaged
		}
		phase := control.phase
		failure := controlFailure(control)
		changed := control.changed
		supervisor.mu.Unlock()
		if phase == EndpointSupervisorReady {
			lease, err := supervisor.controller.AcquireCurrent(endpointID)
			if err == nil {
				return lease, nil
			}
			supervisor.wake(endpointID)
		}
		if phase == EndpointSupervisorBlocked {
			return nil, failure
		}
		select {
		case <-ctx.Done():
			return nil, runtimeError(ErrorCanceled, "endpoint supervisor acquire was canceled", ctx.Err())
		case <-supervisor.ctx.Done():
			return nil, runtimeError(ErrorUnavailable, "endpoint supervisor is closed", nil)
		case <-changed:
		}
	}
}

func (supervisor *EndpointSupervisor) WaitReady(ctx context.Context) error {
	if supervisor == nil || ctx == nil {
		return runtimeError(ErrorInvalidRequest, "endpoint supervisor and context are required", nil)
	}
	for {
		supervisor.mu.Lock()
		if supervisor.closed {
			supervisor.mu.Unlock()
			return runtimeError(ErrorUnavailable, "endpoint supervisor is closed", nil)
		}
		changed := supervisor.changed
		var failure error
		pending := false
		for _, control := range supervisor.endpoints {
			if !control.demanded || control.mode != EndpointSupervisorTakeover {
				continue
			}
			switch control.phase {
			case EndpointSupervisorReady, EndpointSupervisorWaitingNetwork:
				continue
			case EndpointSupervisorBlocked:
				failure = controlFailure(control)
			default:
				pending = true
			}
			if failure != nil {
				break
			}
		}
		supervisor.mu.Unlock()
		if failure != nil {
			return failure
		}
		if !pending {
			return nil
		}
		select {
		case <-ctx.Done():
			return runtimeError(ErrorCanceled, "endpoint supervisor readiness timed out", ctx.Err())
		case <-supervisor.ctx.Done():
			return runtimeError(ErrorUnavailable, "endpoint supervisor is closed", nil)
		case <-changed:
		}
	}
}

func (supervisor *EndpointSupervisor) Snapshot() []EndpointSupervisorProjection {
	if supervisor == nil {
		return nil
	}
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	result := make([]EndpointSupervisorProjection, 0, len(supervisor.endpoints))
	for _, control := range supervisor.endpoints {
		if !control.demanded {
			continue
		}
		result = append(result, projectionOf(control))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].EndpointID < result[right].EndpointID })
	return result
}

func (supervisor *EndpointSupervisor) demandMatchesLocked(expected map[endpoint.EndpointID]EndpointSupervisorMode) bool {
	matched := 0
	for id, control := range supervisor.endpoints {
		if !control.demanded {
			continue
		}
		mode, exists := expected[id]
		if !exists || mode != control.mode {
			return false
		}
		matched++
	}
	return matched == len(expected)
}

func (supervisor *EndpointSupervisor) Close() error {
	if supervisor == nil {
		return nil
	}
	supervisor.mu.Lock()
	if supervisor.closed {
		supervisor.mu.Unlock()
		return nil
	}
	supervisor.closed = true
	supervisor.cancel()
	controls := make([]*endpointControl, 0, len(supervisor.endpoints))
	for _, control := range supervisor.endpoints {
		if control.attemptCancel != nil {
			control.attemptCancel()
		}
		supervisor.notifyLocked(control)
		controls = append(controls, control)
	}
	supervisor.mu.Unlock()
	for _, control := range controls {
		<-control.done
	}
	return nil
}

func (supervisor *EndpointSupervisor) runEndpoint(control *endpointControl) {
	defer close(control.done)
	for {
		select {
		case <-supervisor.ctx.Done():
			supervisor.releaseMaintenance(control)
			return
		default:
		}
		supervisor.drainWake(control)
		snapshot := supervisor.controlSnapshot(control)
		if !snapshot.demanded {
			supervisor.releaseMaintenance(control)
			supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorNoDemand, EndpointSessionStamp{}, nil)
			if !supervisor.waitWake(control, nil, -1) {
				return
			}
			continue
		}
		if snapshot.mode == EndpointSupervisorShadow {
			supervisor.releaseMaintenance(control)
			decision := "would_probe"
			lease, err := supervisor.controller.AcquireCurrent(control.endpointID)
			if err != nil {
				decision = "would_dial"
			} else {
				_ = lease.Close()
			}
			supervisor.options.Logf("anytty endpoint_supervisor endpoint=%s mode=shadow control_revision=%d decision=%s", control.endpointID, snapshot.controlRevision, decision)
			supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorShadowDecision, EndpointSessionStamp{}, nil)
			if !supervisor.waitWake(control, nil, -1) {
				return
			}
			continue
		}
		if !snapshot.connected {
			supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorWaitingNetwork, supervisor.maintenanceStamp(control), nil)
			if !supervisor.waitWake(control, supervisor.maintenanceDone(control), -1) {
				return
			}
			continue
		}

		lease := supervisor.takeOrAcquireCurrent(control)
		if lease != nil {
			verified, failure := supervisor.verify(control, snapshot, lease)
			if !verified {
				if failure != nil && supervisor.isCurrent(control, snapshot.controlRevision) {
					if !supervisor.pauseAfterFailure(control, snapshot, failure) {
						return
					}
				}
				continue
			}
			supervisor.setMaintenance(control, snapshot.controlRevision, lease)
			supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorReady, lease.Stamp(), nil)
			if !supervisor.waitWake(control, lease.Done(), -1) {
				return
			}
			continue
		}

		connected, failure := supervisor.connect(control, snapshot)
		if connected != nil {
			verified, probeFailure := supervisor.verify(control, snapshot, connected)
			if !verified {
				if probeFailure != nil && supervisor.isCurrent(control, snapshot.controlRevision) {
					if !supervisor.pauseAfterFailure(control, snapshot, probeFailure) {
						return
					}
				}
				continue
			}
			supervisor.setMaintenance(control, snapshot.controlRevision, connected)
			supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorReady, connected.Stamp(), nil)
			if !supervisor.waitWake(control, connected.Done(), -1) {
				return
			}
			continue
		}
		if !supervisor.pauseAfterFailure(control, snapshot, failure) {
			return
		}
	}
}

func (supervisor *EndpointSupervisor) verify(control *endpointControl, snapshot endpointControlSnapshot, lease ApplicationReadyPeerSession) (bool, error) {
	attemptID, ctx, cancel, ok := supervisor.beginAttempt(control, snapshot.controlRevision, EndpointSupervisorActionProbe)
	if !ok {
		_ = lease.Close()
		return false, runtimeError(ErrorCanceled, "endpoint supervisor probe was superseded", nil)
	}
	supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorVerifying, lease.Stamp(), nil)
	failure := supervisor.inject(EndpointSupervisorActionProbe, control.endpointID, attemptID)
	if failure == nil {
		failure = supervisor.controller.Probe(ctx, lease)
	}
	cancel()
	supervisor.endAttempt(control, attemptID)
	if failure == nil && supervisor.isCurrent(control, snapshot.controlRevision) {
		supervisor.resetBackoff(control)
		return true, nil
	}
	if !supervisor.isCurrent(control, snapshot.controlRevision) {
		_ = lease.Close()
		return false, runtimeError(ErrorCanceled, "endpoint supervisor probe completed after control changed", failure)
	}
	stamp := lease.Stamp()
	invalidateFailure := supervisor.inject(EndpointSupervisorActionInvalidate, control.endpointID, attemptID)
	if invalidateFailure == nil {
		invalidateFailure = supervisor.controller.Invalidate(stamp, failure)
	}
	if invalidateFailure != nil && CodeOf(invalidateFailure) != ErrorStaleSession {
		supervisor.options.Logf("anytty endpoint_supervisor endpoint=%s control_revision=%d attempt_id=%d invalidate_error=%v", control.endpointID, snapshot.controlRevision, attemptID, invalidateFailure)
	}
	_ = lease.Close()
	supervisor.clearMaintenance(control, lease)
	return false, failure
}

func (supervisor *EndpointSupervisor) pauseAfterFailure(control *endpointControl, snapshot endpointControlSnapshot, failure error) bool {
	if !supervisor.isCurrent(control, snapshot.controlRevision) {
		return true
	}
	if !recoverableSupervisorFailure(failure) {
		supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorBlocked, EndpointSessionStamp{}, failure)
		return supervisor.waitWake(control, nil, -1)
	}
	delay := supervisor.nextBackoff(control, snapshot.controlRevision)
	supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorBackoff, EndpointSessionStamp{}, failure)
	if !supervisor.waitWake(control, nil, delay) {
		return false
	}
	supervisor.advanceRetryRevision(control, snapshot.controlRevision)
	return true
}

func (supervisor *EndpointSupervisor) connect(control *endpointControl, snapshot endpointControlSnapshot) (ApplicationReadyPeerSession, error) {
	attemptID, ctx, cancel, ok := supervisor.beginAttempt(control, snapshot.controlRevision, EndpointSupervisorActionDial)
	if !ok {
		return nil, runtimeError(ErrorCanceled, "endpoint supervisor dial was superseded", nil)
	}
	supervisor.publish(control, snapshot.controlRevision, EndpointSupervisorConnecting, EndpointSessionStamp{}, nil)
	failure := supervisor.inject(EndpointSupervisorActionDial, control.endpointID, attemptID)
	var lease ApplicationReadyPeerSession
	if failure == nil {
		lease, failure = supervisor.controller.Connect(ctx, control.endpointID)
	}
	cancel()
	supervisor.endAttempt(control, attemptID)
	if failure != nil {
		return nil, failure
	}
	if !supervisor.isCurrent(control, snapshot.controlRevision) {
		_ = supervisor.controller.Invalidate(lease.Stamp(), runtimeError(ErrorCanceled, "endpoint supervisor dial completed after control changed", nil))
		_ = lease.Close()
		return nil, runtimeError(ErrorCanceled, "endpoint supervisor dial completed after control changed", nil)
	}
	return lease, nil
}

func (supervisor *EndpointSupervisor) beginAttempt(control *endpointControl, revision uint64, action EndpointSupervisorAction) (uint64, context.Context, context.CancelFunc, bool) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.closed || !control.demanded || control.mode != EndpointSupervisorTakeover || control.controlRevision != revision {
		ctx, cancel := context.WithCancel(supervisor.ctx)
		cancel()
		return 0, ctx, func() {}, false
	}
	control.attemptID++
	attemptID := control.attemptID
	base := supervisor.options.ProbeTimeout
	if action == EndpointSupervisorActionDial {
		base = supervisor.options.DialTimeout
		control.dialCount++
	} else {
		control.probeCount++
	}
	ctx, cancel := context.WithTimeout(supervisor.ctx, base)
	control.attemptCancel = cancel
	return attemptID, ctx, cancel, true
}

func (supervisor *EndpointSupervisor) endAttempt(control *endpointControl, attemptID uint64) {
	supervisor.mu.Lock()
	if control.attemptID == attemptID {
		control.attemptCancel = nil
	}
	supervisor.mu.Unlock()
}

func (supervisor *EndpointSupervisor) controlSnapshot(control *endpointControl) endpointControlSnapshot {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return endpointControlSnapshot{
		mode:            control.mode,
		demanded:        control.demanded,
		connected:       supervisor.connected,
		controlRevision: control.controlRevision,
	}
}

func (supervisor *EndpointSupervisor) isCurrent(control *endpointControl, revision uint64) bool {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return !supervisor.closed && control.demanded && control.mode == EndpointSupervisorTakeover && control.controlRevision == revision
}

func (supervisor *EndpointSupervisor) takeOrAcquireCurrent(control *endpointControl) ApplicationReadyPeerSession {
	supervisor.mu.Lock()
	maintenance := control.maintenance
	control.maintenance = nil
	supervisor.mu.Unlock()
	if maintenance != nil {
		select {
		case <-maintenance.Done():
			_ = maintenance.Close()
		default:
			return maintenance
		}
	}
	lease, err := supervisor.controller.AcquireCurrent(control.endpointID)
	if err != nil {
		return nil
	}
	return lease
}

func (supervisor *EndpointSupervisor) setMaintenance(control *endpointControl, revision uint64, lease ApplicationReadyPeerSession) {
	supervisor.mu.Lock()
	if !supervisor.closed && control.demanded && control.controlRevision == revision {
		control.maintenance = lease
		control.stamp = lease.Stamp()
		supervisor.mu.Unlock()
		return
	}
	supervisor.mu.Unlock()
	_ = lease.Close()
}

func (supervisor *EndpointSupervisor) clearMaintenance(control *endpointControl, lease ApplicationReadyPeerSession) {
	supervisor.mu.Lock()
	if control.maintenance == lease {
		control.maintenance = nil
		control.stamp = EndpointSessionStamp{}
	}
	supervisor.mu.Unlock()
}

func (supervisor *EndpointSupervisor) releaseMaintenance(control *endpointControl) {
	supervisor.mu.Lock()
	lease := control.maintenance
	control.maintenance = nil
	control.stamp = EndpointSessionStamp{}
	supervisor.mu.Unlock()
	if lease != nil {
		_ = lease.Close()
	}
}

func (supervisor *EndpointSupervisor) maintenanceDone(control *endpointControl) <-chan struct{} {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if control.maintenance == nil {
		return nil
	}
	return control.maintenance.Done()
}

func (supervisor *EndpointSupervisor) maintenanceStamp(control *endpointControl) EndpointSessionStamp {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return control.stamp
}

func (supervisor *EndpointSupervisor) nextBackoff(control *endpointControl, revision uint64) time.Duration {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if control.controlRevision != revision {
		return 0
	}
	index := control.backoffIndex
	if index >= len(supervisor.options.Backoff) {
		index = len(supervisor.options.Backoff) - 1
	}
	delay := supervisor.options.Backoff[index]
	control.backoffIndex++
	control.backoffCount++
	if delay > 0 && supervisor.options.Random != nil {
		delay = delay/2 + time.Duration(supervisor.options.Random.Int63n(int64(delay/2)+1))
	}
	return delay
}

func (supervisor *EndpointSupervisor) resetBackoff(control *endpointControl) {
	supervisor.mu.Lock()
	control.backoffIndex = 0
	supervisor.mu.Unlock()
}

func (supervisor *EndpointSupervisor) waitWake(control *endpointControl, sessionDone <-chan struct{}, delay time.Duration) bool {
	var timer <-chan time.Time
	var value *time.Timer
	if delay > 0 {
		value = time.NewTimer(delay)
		timer = value.C
		defer value.Stop()
	} else if delay == 0 {
		timer = time.After(0)
	}
	select {
	case <-supervisor.ctx.Done():
		supervisor.releaseMaintenance(control)
		return false
	case <-control.wake:
		return true
	case <-sessionDone:
		supervisor.releaseMaintenance(control)
		return true
	case <-timer:
		return true
	}
}

func (supervisor *EndpointSupervisor) publish(control *endpointControl, revision uint64, phase EndpointSupervisorPhase, stamp EndpointSessionStamp, failure error) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if control.controlRevision != revision && phase != EndpointSupervisorNoDemand {
		return
	}
	control.phase = phase
	control.stamp = stamp
	control.errorCode = CodeOf(failure)
	control.message = errorMessage(failure)
	supervisor.notifyLocked(control)
	supervisor.options.Logf("anytty endpoint_supervisor endpoint=%s mode=%s phase=%s control_revision=%d attempt_id=%d stamp_generation=%d error_code=%s", control.endpointID, control.mode, phase, control.controlRevision, control.attemptID, stamp.Generation, control.errorCode)
}

func (supervisor *EndpointSupervisor) advanceControlLocked(control *endpointControl) {
	control.controlRevision++
	control.backoffIndex = 0
	if control.attemptCancel != nil {
		control.attemptCancel()
	}
	supervisor.notifyLocked(control)
	supervisor.wakeLocked(control)
}

func (supervisor *EndpointSupervisor) advanceRetryRevision(control *endpointControl, revision uint64) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.closed || !control.demanded || control.mode != EndpointSupervisorTakeover || control.controlRevision != revision {
		return
	}
	control.controlRevision++
	supervisor.notifyLocked(control)
}

func (supervisor *EndpointSupervisor) notifyLocked(control *endpointControl) {
	close(control.changed)
	control.changed = make(chan struct{})
	close(supervisor.changed)
	supervisor.changed = make(chan struct{})
}

func (supervisor *EndpointSupervisor) wakeLocked(control *endpointControl) {
	select {
	case control.wake <- struct{}{}:
	default:
	}
}

func (supervisor *EndpointSupervisor) wake(endpointID endpoint.EndpointID) {
	supervisor.mu.Lock()
	if control := supervisor.endpoints[endpointID]; control != nil {
		supervisor.wakeLocked(control)
	}
	supervisor.mu.Unlock()
}

func (supervisor *EndpointSupervisor) drainWake(control *endpointControl) {
	for {
		select {
		case <-control.wake:
		default:
			return
		}
	}
}

func (supervisor *EndpointSupervisor) inject(action EndpointSupervisorAction, endpointID endpoint.EndpointID, attemptID uint64) error {
	if supervisor.options.Faults == nil {
		return nil
	}
	return supervisor.options.Faults.Before(action, endpointID, attemptID)
}

func projectionOf(control *endpointControl) EndpointSupervisorProjection {
	return EndpointSupervisorProjection{
		EndpointID:      control.endpointID,
		Mode:            control.mode,
		Phase:           control.phase,
		ControlRevision: control.controlRevision,
		AttemptID:       control.attemptID,
		SessionStamp:    control.stamp,
		ErrorCode:       control.errorCode,
		Message:         control.message,
		ProbeCount:      control.probeCount,
		DialCount:       control.dialCount,
		BackoffCount:    control.backoffCount,
	}
}

func controlFailure(control *endpointControl) error {
	code := control.errorCode
	if code == "" {
		code = ErrorUnavailable
	}
	message := control.message
	if message == "" {
		message = fmt.Sprintf("endpoint %q recovery is blocked", control.endpointID)
	}
	return runtimeError(code, message, nil)
}

func recoverableSupervisorFailure(err error) bool {
	if err == nil {
		return true
	}
	var runtimeErr *Error
	if errors.As(err, &runtimeErr) && runtimeErr.Retryable {
		return true
	}
	switch CodeOf(err) {
	case ErrorInvalidRequest, ErrorUnsupportedRoute, ErrorIdentity, ErrorAuthorization, ErrorNotFound,
		ErrorResourceExhausted, ErrorEntitlement,
		ErrorDaemonBlocked, ErrorDaemonDeleted, ErrorRelayNotInPlan,
		ErrorRelayQuotaExhausted, ErrorRelayConcurrencyExhausted,
		ErrorSubscriptionInactive, ErrorRelayRegionUnavailable:
		return false
	default:
		return true
	}
}
