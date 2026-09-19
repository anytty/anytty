package accessruntime

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

	cloud "github.com/anytty/anytty/access/cloud"
	accesscontract "github.com/anytty/anytty/access/contract"
	"github.com/anytty/anytty/access/localweb"
	remote "github.com/anytty/anytty/access/remote"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

const cloudRuntimeCurrentWait = 3 * time.Second

// CloudControl 是 access runtime 内 Cloud runtime 与 local web 的控制面。
// 它实现 core.RemoteService，并拥有 enrollment 切换、撤销与 edge 选择。
type CloudControl struct {
	mu                sync.RWMutex
	runtime           *cloud.Runtime
	runtimeCancel     context.CancelFunc
	runtimeEnrollment cloudEnrollmentIdentity
	wake              chan struct{}
	recordPath        string
	disabledPath      string
	lastRuntimeError  string
	updatedAt         time.Time
	localWebCore      localweb.Core
	localWeb          *localweb.Server
	localWebUpdated   time.Time
}

type cloudEnrollmentIdentity struct {
	daemonID   string
	accountID  string
	enrolledAt time.Time
}

func cloudEnrollmentIdentityFromRecord(record cloud.EnrollmentRecord) cloudEnrollmentIdentity {
	return cloudEnrollmentIdentity{daemonID: record.DaemonID, accountID: record.AccountID, enrolledAt: record.EnrolledAt}
}

// Configure 绑定 enrollment/disabled 路径并创建 wake channel。
func (control *CloudControl) Configure(recordPath, disabledPath string) {
	control.mu.Lock()
	if control.wake == nil {
		control.wake = make(chan struct{}, 1)
	}
	control.recordPath = recordPath
	control.disabledPath = disabledPath
	control.mu.Unlock()
}

// ConfigureLocalWeb 注入 local web 的 core 边界。
func (control *CloudControl) ConfigureLocalWeb(core localweb.Core) {
	control.mu.Lock()
	control.localWebCore = core
	control.mu.Unlock()
}

func (control *CloudControl) setRuntime(runtime *cloud.Runtime, cancel context.CancelFunc, record cloud.EnrollmentRecord) {
	control.mu.Lock()
	control.runtime = runtime
	control.runtimeCancel = cancel
	control.runtimeEnrollment = cloudEnrollmentIdentityFromRecord(record)
	control.updatedAt = time.Now().UTC()
	if runtime != nil {
		control.lastRuntimeError = ""
	}
	control.mu.Unlock()
}

func (control *CloudControl) restartRuntimeForEnrollment(record cloud.EnrollmentRecord) bool {
	desired := cloudEnrollmentIdentityFromRecord(record)
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

func (control *CloudControl) runtimeUsesEnrollment(record cloud.EnrollmentRecord) bool {
	desired := cloudEnrollmentIdentityFromRecord(record)
	control.mu.RLock()
	defer control.mu.RUnlock()
	return control.runtime != nil && control.runtimeEnrollment == desired
}

func (control *CloudControl) waitRuntimeEnrollment(ctx context.Context, record cloud.EnrollmentRecord, timeout time.Duration) {
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

func (control *CloudControl) currentRuntime() (*cloud.Runtime, bool, error) {
	control.mu.RLock()
	runtime := control.runtime
	disabledPath := control.disabledPath
	control.mu.RUnlock()
	if disabledPath != "" {
		if disabled, err := CloudDisabled(disabledPath); err != nil {
			return nil, false, err
		} else if disabled {
			return nil, true, nil
		}
	}
	return runtime, false, nil
}

func (control *CloudControl) current(ctx context.Context) (*cloud.Runtime, error) {
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
			status, statusErr := control.CloudStatus(context.Background())
			if statusErr != nil {
				return nil, statusErr
			}
			return nil, cloudRuntimeUnavailableError(status)
		case <-ticker.C:
		}
	}
}

func cloudRuntimeUnavailableError(status accesscontract.RemoteCloudStatus) error {
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

func (control *CloudControl) wakeLoop() {
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

func (control *CloudControl) wakeChannel() <-chan struct{} {
	control.mu.RLock()
	wake := control.wake
	control.mu.RUnlock()
	return wake
}

func (control *CloudControl) cancelRuntime() {
	control.mu.RLock()
	cancel := control.runtimeCancel
	control.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
}

func (control *CloudControl) setRuntimeError(err error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	control.updatedAt = time.Now().UTC()
	if err == nil || errors.Is(err, context.Canceled) {
		control.lastRuntimeError = ""
		return
	}
	control.lastRuntimeError = err.Error()
}

func (control *CloudControl) runtimeRunning() bool {
	control.mu.RLock()
	defer control.mu.RUnlock()
	return control.runtime != nil
}

func (control *CloudControl) waitRuntimeRunning(ctx context.Context, want bool, timeout time.Duration) {
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

func (control *CloudControl) cloudStatus(poolRunning bool) (accesscontract.RemoteCloudStatus, error) {
	control.mu.RLock()
	runtime := control.runtime
	recordPath := control.recordPath
	disabledPath := control.disabledPath
	lastRuntimeError := control.lastRuntimeError
	control.mu.RUnlock()
	if recordPath == "" {
		recordPath = EnrollmentRecordPath()
	}
	if disabledPath == "" {
		disabledPath = DisabledPath()
	}
	return LoadCloudStatus(recordPath, disabledPath, runtime, poolRunning, lastRuntimeError)
}

// Status 实现 core.RemoteService；当前 access runtime 不提供旧 remote 控制面。
func (*CloudControl) Status(context.Context) (accesscontract.RemoteStatus, error) {
	return accesscontract.RemoteStatus{}, accesscontract.ErrRemoteServiceUnavailable
}

// PairStart 实现 core.RemoteService；旧 remote 配对已由本机 pairing socket 取代。
func (*CloudControl) PairStart(context.Context, accesscontract.RemotePairStartRequest) (accesscontract.RemotePairStartResult, error) {
	return accesscontract.RemotePairStartResult{}, accesscontract.ErrRemoteServiceUnavailable
}

func (control *CloudControl) LocalEnable(_ context.Context, request accesscontract.RemoteLocalEnableRequest) (accesscontract.RemoteLocalStatus, error) {
	defer clear(request.LocalWebPassword)
	control.mu.Lock()
	defer control.mu.Unlock()
	if control.localWebCore == nil {
		return accesscontract.RemoteLocalStatus{}, accesscontract.ErrRemoteServiceUnavailable
	}
	address := strings.TrimSpace(request.LocalWebAddress)
	if address == "" {
		address = localweb.DefaultAddress
	}
	if control.localWeb != nil {
		if address == localweb.DefaultAddress || address == control.localWeb.Address() {
			if len(request.LocalWebPassword) > 0 {
				return accesscontract.RemoteLocalStatus{}, fmt.Errorf("local web is already running; stop it before changing password protection")
			}
			return localWebStatus(control.localWeb, control.localWebUpdated), nil
		}
		return accesscontract.RemoteLocalStatus{}, fmt.Errorf("local web is already running at %s", control.localWeb.Address())
	}
	server, err := localweb.Start(localweb.Options{Core: control.localWebCore, Address: address, Password: request.LocalWebPassword})
	if err != nil {
		return accesscontract.RemoteLocalStatus{}, err
	}
	control.localWeb = server
	control.localWebUpdated = time.Now().UTC()
	return localWebStatus(server, control.localWebUpdated), nil
}

func (control *CloudControl) LocalStatus(context.Context) (accesscontract.RemoteLocalStatus, error) {
	control.mu.RLock()
	defer control.mu.RUnlock()
	return localWebStatus(control.localWeb, control.localWebUpdated), nil
}

func (control *CloudControl) LocalDisable(ctx context.Context) (accesscontract.RemoteLocalStatus, error) {
	control.mu.Lock()
	server := control.localWeb
	control.localWeb = nil
	control.localWebUpdated = time.Now().UTC()
	updated := control.localWebUpdated
	control.mu.Unlock()
	if server != nil {
		if err := server.Stop(ctx); err != nil {
			return accesscontract.RemoteLocalStatus{}, err
		}
	}
	return localWebStatus(nil, updated), nil
}

// CloseLocalWeb 关闭本进程内 local web。
func (control *CloudControl) CloseLocalWeb() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = control.LocalDisable(ctx)
}

func localWebStatus(server *localweb.Server, updated time.Time) accesscontract.RemoteLocalStatus {
	status := accesscontract.RemoteLocalStatus{UpdatedAt: updated}
	if server != nil {
		status.Enabled = true
		status.HTTPURL = server.URL()
		status.LocalWebAddress = server.Address()
		status.PasswordProtected = server.PasswordProtected()
	}
	return status
}

// CloudStatus 实现 core.RemoteService。
func (control *CloudControl) CloudStatus(context.Context) (accesscontract.RemoteCloudStatus, error) {
	return control.cloudStatus(true)
}

// CloudEnable 实现 core.RemoteService。
func (control *CloudControl) CloudEnable(ctx context.Context) (accesscontract.RemoteCloudStatus, error) {
	if err := removeCloudDisabled(); err != nil {
		return accesscontract.RemoteCloudStatus{}, err
	}
	if record, err := cloud.LoadRecord(EnrollmentRecordPath()); err == nil {
		control.restartRuntimeForEnrollment(record)
		control.wakeLoop()
		control.waitRuntimeEnrollment(ctx, record, 2*time.Second)
	} else if !errors.Is(err, os.ErrNotExist) {
		return accesscontract.RemoteCloudStatus{}, err
	} else {
		control.wakeLoop()
	}
	return control.cloudStatus(true)
}

// CloudDisable 实现 core.RemoteService。
func (control *CloudControl) CloudDisable(ctx context.Context) (accesscontract.RemoteCloudStatus, error) {
	if err := writeCloudDisabled("disabled by local command"); err != nil {
		return accesscontract.RemoteCloudStatus{}, err
	}
	control.cancelRuntime()
	control.wakeLoop()
	control.waitRuntimeRunning(ctx, false, 2*time.Second)
	return control.cloudStatus(true)
}

// CloudEdges 实现 core.RemoteService。
func (control *CloudControl) CloudEdges(ctx context.Context) (accesscontract.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current(ctx)
	if err != nil {
		return accesscontract.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.EdgeSelection(ctx)
	return cloudSelectionToCore(selection), err
}

// CloudPreferEdge 实现 core.RemoteService。
func (control *CloudControl) CloudPreferEdge(ctx context.Context, edgeID string, expectedRevision uint64) (accesscontract.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current(ctx)
	if err != nil {
		return accesscontract.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.PreferEdge(ctx, edgeID, expectedRevision)
	return cloudSelectionToCore(selection), err
}

// CloudReselectEdge 实现 core.RemoteService。
func (control *CloudControl) CloudReselectEdge(ctx context.Context) (accesscontract.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current(ctx)
	if err != nil {
		return accesscontract.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.ReselectEdge(ctx)
	return cloudSelectionToCore(selection), err
}

func cloudSelectionToCore(selection *cloudv1.DaemonEdgeSelection) accesscontract.RemoteCloudEdgeSelection {
	if selection == nil {
		return accesscontract.RemoteCloudEdgeSelection{}
	}
	result := accesscontract.RemoteCloudEdgeSelection{DaemonID: selection.GetDaemonId(), PreferredEdgeID: selection.GetPreferredEdgeId(), PreferenceRevision: selection.GetPreferenceRevision(), CurrentEdgeID: selection.GetCurrentEdgeId(), SelectedEdgeID: selection.GetSelectedEdgeId(), Candidates: make([]accesscontract.RemoteCloudEdgeCandidate, 0, len(selection.GetCandidates()))}
	if selection.GetEvaluatedAt() != nil {
		result.EvaluatedAt = selection.GetEvaluatedAt().AsTime()
	}
	for _, candidate := range selection.GetCandidates() {
		locator := candidate.GetLocator()
		value := accesscontract.RemoteCloudEdgeCandidate{EdgeID: locator.GetEdgeId(), Name: locator.GetName(), Region: locator.GetRegion(), PublicEndpoint: locator.GetPublicEndpoint(), Status: candidate.GetStatus(), Online: candidate.GetOnline(), Eligible: candidate.GetEligible(), Preferred: candidate.GetPreferred(), Current: candidate.GetCurrent(), Score: candidate.GetScore()}
		if measured := candidate.GetMeasurement(); measured != nil {
			measurement := &accesscontract.RemoteCloudEdgeMeasurement{Reachable: measured.GetReachable(), ConnectLatencyMS: measured.GetConnectLatencyMs(), ConnectionFailureRate: measured.GetConnectionFailureRate(), SampleCount: measured.GetSampleCount()}
			if measured.GetMeasuredAt() != nil {
				measurement.MeasuredAt = measured.GetMeasuredAt().AsTime()
			}
			value.Measurement = measurement
		}
		result.Candidates = append(result.Candidates, value)
	}
	return result
}

// StartCloud 复用同一 Runtime 的 DeviceIdentity/AccessStore/Core，并让 Cloud runtime 只拥有发现和信令。
func StartCloud(ctx context.Context, core remote.TransportServer, access Runtime, logger *slog.Logger, control *CloudControl) (func(), error) {
	if control == nil {
		return nil, errors.New("Cloud runtime control is required")
	}
	recordPath := EnrollmentRecordPath()
	disabledPath := DisabledPath()
	control.Configure(recordPath, disabledPath)
	var initial *cloud.Runtime
	record := cloud.EnrollmentRecord{}
	if disabled, err := CloudDisabled(disabledPath); err != nil {
		return nil, err
	} else if !disabled {
		var err error
		record, err = cloud.LoadRecord(recordPath)
		if errors.Is(err, os.ErrNotExist) {
			record = cloud.EnrollmentRecord{}
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}
	if record.DaemonID != "" {
		var err error
		initial, err = newCloudRuntime(record, recordPath, core, access, logger)
		if err != nil {
			return nil, err
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer control.setRuntime(nil, nil, cloud.EnrollmentRecord{})
		runtime := initial
		for runCtx.Err() == nil {
			if runtime == nil {
				disabled, loadErr := CloudDisabled(disabledPath)
				if loadErr != nil {
					logger.Error("AnyTTY Cloud disabled marker could not be loaded", "error", loadErr)
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				}
				if disabled {
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				}
				next, loadErr := cloud.LoadRecord(recordPath)
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
				runtime, loadErr = newCloudRuntime(next, recordPath, core, access, logger)
				if loadErr != nil {
					control.setRuntimeError(loadErr)
					logger.Error("AnyTTY Cloud pool runtime could not start", "error", loadErr)
					if !waitForCloudEnrollment(runCtx, control.wakeChannel()) {
						return
					}
					continue
				}
				record = next
			}
			runtimeCtx, runtimeCancel := context.WithCancel(runCtx)
			control.setRuntime(runtime, runtimeCancel, record)
			logger.Info("AnyTTY Cloud pool runtime started", "daemon_id", record.DaemonID)
			runErr := runtime.Run(runtimeCtx)
			runtimeCancel()
			control.setRuntime(nil, nil, cloud.EnrollmentRecord{})
			if releaseErr := access.Store.DisableManagedCloudRoute(); releaseErr != nil {
				control.setRuntimeError(releaseErr)
				logger.Error("AnyTTY Cloud route issuers could not be released", "error", releaseErr)
				return
			}
			control.setRuntimeError(runErr)
			if runErr != nil && runCtx.Err() == nil && !errors.Is(runErr, context.Canceled) {
				logger.Error("AnyTTY Cloud pool runtime stopped", "error", runErr)
			}
			runtime = nil
		}
	}()
	return func() { cancel(); <-done }, nil
}

func newCloudRuntime(record cloud.EnrollmentRecord, recordPath string, core remote.TransportServer, access Runtime, logger *slog.Logger) (*cloud.Runtime, error) {
	controller, err := CloudControllerEndpointFromEnvironment()
	if err != nil {
		return nil, err
	}
	return cloud.NewAuthorizedRuntime(
		record, access.Identity, access.Store, core, "development",
		func() {
			logger.Info("AnyTTY Cloud DataChannel 已进入端到端授权", "daemon_id", record.DaemonID)
		},
		func(sessionErr error) {
			if errors.Is(sessionErr, io.EOF) || errors.Is(sessionErr, context.Canceled) {
				return
			}
			logger.Warn("AnyTTY Cloud DataChannel 会话异常结束", "daemon_id", record.DaemonID, "error", sessionErr)
		},
		cloud.WithPionLogger(logger),
		cloud.WithEnrollmentRecordPath(recordPath),
		cloud.WithControllerEndpoint(controller.Address, controller.ServerName, controller.CAPEM),
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
