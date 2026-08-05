package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	corev2 "github.com/anytty/anytty/core"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

type v3CloudRuntimeControl struct {
	mu      sync.RWMutex
	runtime *clouddaemon.Runtime
}

func (control *v3CloudRuntimeControl) setRuntime(runtime *clouddaemon.Runtime) {
	control.mu.Lock()
	control.runtime = runtime
	control.mu.Unlock()
}

func (control *v3CloudRuntimeControl) current() (*clouddaemon.Runtime, error) {
	control.mu.RLock()
	runtime := control.runtime
	control.mu.RUnlock()
	if runtime == nil {
		return nil, corev2.ErrRemoteServiceUnavailable
	}
	return runtime, nil
}

func (*v3CloudRuntimeControl) Status(context.Context) (corev2.RemoteStatus, error) {
	return corev2.RemoteStatus{}, corev2.ErrRemoteServiceUnavailable
}
func (*v3CloudRuntimeControl) PairStart(context.Context, corev2.RemotePairStartRequest) (corev2.RemotePairStartResult, error) {
	return corev2.RemotePairStartResult{}, corev2.ErrRemoteServiceUnavailable
}
func (*v3CloudRuntimeControl) LocalEnable(context.Context, corev2.RemoteLocalEnableRequest) (corev2.RemoteLocalStatus, error) {
	return corev2.RemoteLocalStatus{}, corev2.ErrRemoteServiceUnavailable
}
func (*v3CloudRuntimeControl) LocalStatus(context.Context) (corev2.RemoteLocalStatus, error) {
	return corev2.RemoteLocalStatus{}, corev2.ErrRemoteServiceUnavailable
}
func (*v3CloudRuntimeControl) LocalDisable(context.Context) (corev2.RemoteLocalStatus, error) {
	return corev2.RemoteLocalStatus{}, corev2.ErrRemoteServiceUnavailable
}
func (control *v3CloudRuntimeControl) CloudEdges(ctx context.Context) (corev2.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current()
	if err != nil {
		return corev2.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.EdgeSelection(ctx)
	return cloudSelectionToCore(selection), err
}
func (control *v3CloudRuntimeControl) CloudPreferEdge(ctx context.Context, edgeID string, expectedRevision uint64) (corev2.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current()
	if err != nil {
		return corev2.RemoteCloudEdgeSelection{}, err
	}
	selection, err := runtime.PreferEdge(ctx, edgeID, expectedRevision)
	return cloudSelectionToCore(selection), err
}
func (control *v3CloudRuntimeControl) CloudReselectEdge(ctx context.Context) (corev2.RemoteCloudEdgeSelection, error) {
	runtime, err := control.current()
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
		value := corev2.RemoteCloudEdgeCandidate{EdgeID: locator.GetEdgeId(), Name: locator.GetName(), Region: locator.GetRegion(), PublicEndpoint: locator.GetPublicEndpoint(), Status: candidate.GetStatus(), Online: candidate.GetOnline(), Eligible: candidate.GetEligible(), Preferred: candidate.GetPreferred(), Current: candidate.GetCurrent(), AgentCount: candidate.GetAgentCount(), Capacity: candidate.GetCapacity(), Score: candidate.GetScore()}
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
	record, err := clouddaemon.LoadRecord(recordPath)
	if errors.Is(err, os.ErrNotExist) {
		record = clouddaemon.EnrollmentRecord{}
		err = nil
	}
	if err != nil {
		return nil, err
	}
	var initial *clouddaemon.Runtime
	if record.DaemonID != "" {
		initial, err = newV3CloudRuntime(record, recordPath, core, clientAccess, logger)
		if err != nil {
			return nil, err
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer control.setRuntime(nil)
		runtime := initial
		for runCtx.Err() == nil {
			if runtime == nil {
				next, loadErr := clouddaemon.LoadRecord(recordPath)
				if errors.Is(loadErr, os.ErrNotExist) {
					if !waitForCloudEnrollment(runCtx) {
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
					logger.Error("AnyTTY Cloud daemon runtime could not start", "error", loadErr)
					return
				}
				record = next
			}
			control.setRuntime(runtime)
			logger.Info("AnyTTY Cloud daemon runtime started", "daemon_id", record.DaemonID)
			runErr := runtime.Run(runCtx)
			control.setRuntime(nil)
			if runErr != nil && runCtx.Err() == nil {
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

func waitForCloudEnrollment(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
