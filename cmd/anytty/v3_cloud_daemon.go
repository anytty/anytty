package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/localweb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

const cloudRuntimeCurrentWait = 3 * time.Second

type v3CloudRuntimeControl struct {
	mu                sync.RWMutex
	runtime           *clouddaemon.Runtime
	runtimeCancel     context.CancelFunc
	runtimeEnrollment cloudRuntimeEnrollmentIdentity
	wake              chan struct{}
	recordPath        string
	disabledPath      string
	lastRuntimeError  string
	updatedAt         time.Time
	localWebCore      localweb.Core
	localWeb          *localweb.Server
	localWebUpdated   time.Time
}

type cloudRuntimeEnrollmentIdentity struct {
	daemonID   string
	accountID  string
	enrolledAt time.Time
}

func cloudRuntimeEnrollmentIdentityFromRecord(record clouddaemon.EnrollmentRecord) cloudRuntimeEnrollmentIdentity {
	return cloudRuntimeEnrollmentIdentity{daemonID: record.DaemonID, accountID: record.AccountID, enrolledAt: record.EnrolledAt}
}

func (control *v3CloudRuntimeControl) configure(recordPath, disabledPath string) {
	control.mu.Lock()
	if control.wake == nil {
		control.wake = make(chan struct{}, 1)
	}
	control.recordPath = recordPath
	control.disabledPath = disabledPath
	control.mu.Unlock()
}

func (control *v3CloudRuntimeControl) configureLocalWeb(core localweb.Core) {
	control.mu.Lock()
	control.localWebCore = core
	control.mu.Unlock()
}

func (control *v3CloudRuntimeControl) setRuntime(runtime *clouddaemon.Runtime, cancel context.CancelFunc, record clouddaemon.EnrollmentRecord) {
	control.mu.Lock()
	control.runtime = runtime
	control.runtimeCancel = cancel
	control.runtimeEnrollment = cloudRuntimeEnrollmentIdentityFromRecord(record)
	control.updatedAt = time.Now().UTC()
	if runtime != nil {
		control.lastRuntimeError = ""
	}
	control.mu.Unlock()
}

func (control *v3CloudRuntimeControl) restartRuntimeForEnrollment(record clouddaemon.EnrollmentRecord) bool {
	desired := cloudRuntimeEnrollmentIdentityFromRecord(record)
	control.mu.RLock()
	running := control.runtime != nil
	current := control.runtimeEnrollment
	cancel := control.runtimeCancel
	control.mu.RUnlock()
	if !running || current == desired {
		return false
	}
	if cancel != nil {
		cancel()
	}
	return true
}

func (control *v3CloudRuntimeControl) runtimeUsesEnrollment(record clouddaemon.EnrollmentRecord) bool {
	desired := cloudRuntimeEnrollmentIdentityFromRecord(record)
	control.mu.RLock()
	defer control.mu.RUnlock()
	return control.runtime != nil && control.runtimeEnrollment == desired
}

func (control *v3CloudRuntimeControl) waitRuntimeEnrollment(ctx context.Context, record clouddaemon.EnrollmentRecord, timeout time.Duration) {
	if timeout <= 0 || control.runtimeUsesEnrollment(record) {
		return
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-waitCtx.Done():
			return
		case <-ticker.C:
			if control.runtimeUsesEnrollment(record) {
				return
			}
		}
	}
}

func (control *v3CloudRuntimeControl) currentRuntime() (*clouddaemon.Runtime, bool, error) {
	control.mu.RLock()
	runtime := control.runtime
	disabledPath := control.disabledPath
	control.mu.RUnlock()
	if disabledPath != "" {
		if _, disabled, err := readV3CloudDisabled(disabledPath); err != nil {
			return nil, false, err
		} else if disabled {
			return nil, true, nil
		}
	}
	return runtime, false, nil
}

func (control *v3CloudRuntimeControl) current(ctx context.Context) (*clouddaemon.Runtime, error) {
	waitCtx, cancel := context.WithTimeout(ctx, cloudRuntimeCurrentWait)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		runtime, disabled, err := control.currentRuntime()
		if err != nil {
			return nil, err
		}
		if disabled {
			return nil, errors.New("Cloud runtime is disabled; run `anytty cloud enable` to resume it")
		}
		if runtime != nil {
			return runtime, nil
		}
		select {
		case <-waitCtx.Done():
			status, statusErr := control.cloudStatus(true)
			if statusErr != nil {
				return nil, statusErr
			}
			return nil, cloudRuntimeUnavailableError(status)
		case <-ticker.C:
		}
	}
}

func cloudRuntimeUnavailableError(status corev2.RemoteCloudStatus) error {
	switch {
	case !status.Enrolled:
		return errors.New("Cloud is not enrolled; run `anytty cloud enroll CODE` first")
	case !status.Enabled:
		return errors.New("Cloud runtime is disabled; run `anytty cloud enable` to resume it")
	case status.Detail != "":
		return fmt.Errorf("Cloud runtime is not ready: %s", status.Detail)
	case status.State != "":
		return fmt.Errorf("Cloud runtime is not ready: state=%s", status.State)
	default:
		return errors.New("Cloud runtime is not ready; run `anytty cloud status` for details")
	}
}

func (control *v3CloudRuntimeControl) wakeLoop() {
	control.mu.RLock()
	wake := control.wake
	control.mu.RUnlock()
	if wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}

func (control *v3CloudRuntimeControl) wakeChannel() <-chan struct{} {
	control.mu.RLock()
	wake := control.wake
	control.mu.RUnlock()
	return wake
}

func (control *v3CloudRuntimeControl) cancelRuntime() {
	control.mu.RLock()
	cancel := control.runtimeCancel
	control.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
}

func (control *v3CloudRuntimeControl) setRuntimeError(err error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	control.updatedAt = time.Now().UTC()
	if err == nil || errors.Is(err, context.Canceled) {
		control.lastRuntimeError = ""
		return
	}
	control.lastRuntimeError = err.Error()
}

func (control *v3CloudRuntimeControl) runtimeRunning() bool {
	control.mu.RLock()
	defer control.mu.RUnlock()
	return control.runtime != nil
}

func (control *v3CloudRuntimeControl) waitRuntimeRunning(ctx context.Context, want bool, timeout time.Duration) {
	if timeout <= 0 || control.runtimeRunning() == want {
		return
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-waitCtx.Done():
			return
		case <-ticker.C:
			if control.runtimeRunning() == want {
				return
			}
		}
	}
}

func (control *v3CloudRuntimeControl) cloudStatus(daemonRunning bool) (corev2.RemoteCloudStatus, error) {
	control.mu.RLock()
	runtime := control.runtime
	recordPath := control.recordPath
	disabledPath := control.disabledPath
	lastRuntimeError := control.lastRuntimeError
	control.mu.RUnlock()
	if recordPath == "" {
		recordPath = v3CloudEnrollmentRecordPath()
	}
	if disabledPath == "" {
		disabledPath = v3CloudDisabledPath()
	}
	return loadV3CloudStatus(recordPath, disabledPath, runtime, daemonRunning, lastRuntimeError)
}

func (*v3CloudRuntimeControl) Status(context.Context) (corev2.RemoteStatus, error) {
	return corev2.RemoteStatus{}, corev2.ErrRemoteServiceUnavailable
}
func (*v3CloudRuntimeControl) PairStart(context.Context, corev2.RemotePairStartRequest) (corev2.RemotePairStartResult, error) {
	return corev2.RemotePairStartResult{}, corev2.ErrRemoteServiceUnavailable
}
func (control *v3CloudRuntimeControl) LocalEnable(_ context.Context, request corev2.RemoteLocalEnableRequest) (corev2.RemoteLocalStatus, error) {
	defer clear(request.LocalWebPassword)
	control.mu.Lock()
	defer control.mu.Unlock()
	if control.localWebCore == nil {
		return corev2.RemoteLocalStatus{}, corev2.ErrRemoteServiceUnavailable
	}
	address := strings.TrimSpace(request.LocalWebAddress)
	if address == "" {
		address = localweb.DefaultAddress
	}
	if control.localWeb != nil {
		if address == localweb.DefaultAddress || address == control.localWeb.Address() {
			if len(request.LocalWebPassword) > 0 {
				return corev2.RemoteLocalStatus{}, fmt.Errorf("local web is already running; stop it before changing password protection")
			}
			return localWebStatus(control.localWeb, control.localWebUpdated), nil
		}
		return corev2.RemoteLocalStatus{}, fmt.Errorf("local web is already running at %s", control.localWeb.Address())
	}
	server, err := localweb.Start(localweb.Options{Core: control.localWebCore, Address: address, Password: request.LocalWebPassword})
	if err != nil {
		return corev2.RemoteLocalStatus{}, err
	}
	control.localWeb = server
	control.localWebUpdated = time.Now().UTC()
	return localWebStatus(server, control.localWebUpdated), nil
}
func (control *v3CloudRuntimeControl) LocalStatus(context.Context) (corev2.RemoteLocalStatus, error) {
	control.mu.RLock()
	defer control.mu.RUnlock()
	return localWebStatus(control.localWeb, control.localWebUpdated), nil
}
func (control *v3CloudRuntimeControl) LocalDisable(ctx context.Context) (corev2.RemoteLocalStatus, error) {
	control.mu.Lock()
	server := control.localWeb
	control.localWeb = nil
	control.localWebUpdated = time.Now().UTC()
	updated := control.localWebUpdated
	control.mu.Unlock()
	if server != nil {
		if err := server.Stop(ctx); err != nil {
			return corev2.RemoteLocalStatus{}, err
		}
	}
	return localWebStatus(nil, updated), nil
}

func (control *v3CloudRuntimeControl) closeLocalWeb() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = control.LocalDisable(ctx)
}

func localWebStatus(server *localweb.Server, updated time.Time) corev2.RemoteLocalStatus {
	status := corev2.RemoteLocalStatus{UpdatedAt: updated}
	if server != nil {
		status.Enabled = true
		status.HTTPURL = server.URL()
		status.LocalWebAddress = server.Address()
		status.PasswordProtected = server.PasswordProtected()
	}
	return status
}
func (control *v3CloudRuntimeControl) CloudStatus(context.Context) (corev2.RemoteCloudStatus, error) {
	return control.cloudStatus(true)
}
func (control *v3CloudRuntimeControl) CloudEnable(ctx context.Context) (corev2.RemoteCloudStatus, error) {
	if err := removeV3CloudDisabled(); err != nil {
		return corev2.RemoteCloudStatus{}, err
	}
	if record, err := clouddaemon.LoadRecord(v3CloudEnrollmentRecordPath()); err == nil {
		control.restartRuntimeForEnrollment(record)
		control.wakeLoop()
		control.waitRuntimeEnrollment(ctx, record, 2*time.Second)
	} else if !errors.Is(err, os.ErrNotExist) {
		return corev2.RemoteCloudStatus{}, err
	} else {
		control.wakeLoop()
	}
	return control.cloudStatus(true)
}
func (control *v3CloudRuntimeControl) CloudDisable(ctx context.Context) (corev2.RemoteCloudStatus, error) {
	if err := writeV3CloudDisabled("disabled by local command"); err != nil {
		return corev2.RemoteCloudStatus{}, err
	}
	control.cancelRuntime()
	control.wakeLoop()
	control.waitRuntimeRunning(ctx, false, 2*time.Second)
	return control.cloudStatus(true)
}
func (control *v3CloudRuntimeControl) CloudEdges(ctx context.Context) (corev2.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current(ctx)
	if err != nil {
		return corev2.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.EdgeSelection(ctx)
	return cloudSelectionToCore(selection), err
}
func (control *v3CloudRuntimeControl) CloudPreferEdge(ctx context.Context, edgeID string, expectedRevision uint64) (corev2.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current(ctx)
	if err != nil {
		return corev2.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.PreferEdge(ctx, edgeID, expectedRevision)
	return cloudSelectionToCore(selection), err
}
func (control *v3CloudRuntimeControl) CloudReselectEdge(ctx context.Context) (corev2.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current(ctx)
	if err != nil {
		return corev2.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.ReselectEdge(ctx)
	return cloudSelectionToCore(selection), err
}

func cloudSelectionToCore(selection *cloudv1.DaemonEdgeSelection) corev2.RemoteCloudEdgeSelection {
	if selection == nil {
		return corev2.RemoteCloudEdgeSelection{}
	}
	result := corev2.RemoteCloudEdgeSelection{DaemonID: selection.GetDaemonId(), PreferredEdgeID: selection.GetPreferredEdgeId(), PreferenceRevision: selection.GetPreferenceRevision(), CurrentEdgeID: selection.GetCurrentEdgeId(), SelectedEdgeID: selection.GetSelectedEdgeId(), Candidates: make([]corev2.RemoteCloudEdgeCandidate, 0, len(selection.GetCandidates()))}
	if selection.GetEvaluatedAt() != nil {
		result.EvaluatedAt = selection.GetEvaluatedAt().AsTime()
	}
	for _, candidate := range selection.GetCandidates() {
		locator := candidate.GetLocator()
		value := corev2.RemoteCloudEdgeCandidate{EdgeID: locator.GetEdgeId(), Name: locator.GetName(), Region: locator.GetRegion(), PublicEndpoint: locator.GetPublicEndpoint(), Status: candidate.GetStatus(), Online: candidate.GetOnline(), Eligible: candidate.GetEligible(), Preferred: candidate.GetPreferred(), Current: candidate.GetCurrent(), Score: candidate.GetScore()}
		if measured := candidate.GetMeasurement(); measured != nil {
			measurement := &corev2.RemoteCloudEdgeMeasurement{Reachable: measured.GetReachable(), ConnectLatencyMS: measured.GetConnectLatencyMs(), ConnectionFailureRate: measured.GetConnectionFailureRate(), SampleCount: measured.GetSampleCount()}
			if measured.GetMeasuredAt() != nil {
				measurement.MeasuredAt = measured.GetMeasuredAt().AsTime()
			}
			value.Measurement = measurement
		}
		result.Candidates = append(result.Candidates, value)
	}
	return result
}

// startV3CloudDaemon 复用同一 DeviceIdentity/AccessStore/Core，并让 Cloud runtime 只拥有发现和信令。
func startV3CloudDaemon(ctx context.Context, core v3RemoteDaemonCore, clientAccess v3ClientAccessRuntime, logger *slog.Logger, control *v3CloudRuntimeControl) (func(), error) {
	if control == nil {
		return nil, errors.New("Cloud runtime control is required")
	}
	recordPath := v3CloudEnrollmentRecordPath()
	disabledPath := v3CloudDisabledPath()
	control.configure(recordPath, disabledPath)
	var initial *clouddaemon.Runtime
	record := clouddaemon.EnrollmentRecord{}
	if _, disabled, err := readV3CloudDisabled(disabledPath); err != nil {
		return nil, err
	} else if !disabled {
		record, err = clouddaemon.LoadRecord(recordPath)
		if errors.Is(err, os.ErrNotExist) {
			record = clouddaemon.EnrollmentRecord{}
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}
	if record.DaemonID != "" {
		var err error
		initial, err = newV3CloudRuntime(record, recordPath, core, clientAccess, logger)
		if err != nil {
			return nil, err
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer control.setRuntime(nil, nil, clouddaemon.EnrollmentRecord{})
		runtime := initial
		for runCtx.Err() == nil {
			if runtime == nil {
				if _, disabled, loadErr := readV3CloudDisabled(disabledPath); loadErr != nil {
					logger.Error("AnyTTY Cloud disabled marker could not be loaded", "error", loadErr)
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				} else if disabled {
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				}
				next, loadErr := clouddaemon.LoadRecord(recordPath)
				if errors.Is(loadErr, os.ErrNotExist) {
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				}
				if loadErr != nil {
					logger.Error("AnyTTY Cloud enrollment could not be loaded", "error", loadErr)
					return
				}
				runtime, loadErr = newV3CloudRuntime(next, recordPath, core, clientAccess, logger)
				if loadErr != nil {
					control.setRuntimeError(loadErr)
					logger.Error("AnyTTY Cloud daemon runtime could not start", "error", loadErr)
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				}
				record = next
			}
			runtimeCtx, runtimeCancel := context.WithCancel(runCtx)
			control.setRuntime(runtime, runtimeCancel, record)
			logger.Info("AnyTTY Cloud daemon runtime started", "daemon_id", record.DaemonID)
			runErr := runtime.Run(runtimeCtx)
			runtimeCancel()
			control.setRuntime(nil, nil, clouddaemon.EnrollmentRecord{})
			if releaseErr := clientAccess.Store.DisableManagedCloudRoute(); releaseErr != nil {
				control.setRuntimeError(releaseErr)
				logger.Error("AnyTTY Cloud route issuers could not be released", "error", releaseErr)
				return
			}
			control.setRuntimeError(runErr)
			if runErr != nil && runCtx.Err() == nil && !errors.Is(runErr, context.Canceled) {
				logger.Error("AnyTTY Cloud daemon runtime stopped", "error", runErr)
			}
			runtime = nil
		}
	}()
	return func() { cancel(); <-done }, nil
}

func newV3CloudRuntime(record clouddaemon.EnrollmentRecord, recordPath string, core v3RemoteDaemonCore, clientAccess v3ClientAccessRuntime, logger *slog.Logger) (*clouddaemon.Runtime, error) {
	controller, err := cliCloudControllerEndpointFromEnvironment()
	if err != nil {
		return nil, err
	}
	return clouddaemon.NewAuthorizedRuntime(
		record, clientAccess.Identity, clientAccess.Store, core, "development",
		func() {
			logger.Info("AnyTTY Cloud DataChannel 已进入端到端授权", "daemon_id", record.DaemonID)
		},
		func(sessionErr error) {
			if errors.Is(sessionErr, io.EOF) || errors.Is(sessionErr, context.Canceled) {
				return
			}
			logger.Warn("AnyTTY Cloud DataChannel 会话异常结束", "daemon_id", record.DaemonID, "error", sessionErr)
		},
		clouddaemon.WithPionLogger(logger),
		clouddaemon.WithEnrollmentRecordPath(recordPath),
		clouddaemon.WithControllerEndpoint(controller.address, controller.serverName, controller.caPEM),
	)
}

func waitForCloudEnrollment(ctx context.Context, wake <-chan struct{}) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-wake:
		return true
	case <-timer.C:
		return true
	}
}
