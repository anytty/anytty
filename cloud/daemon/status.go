package daemon

import (
	"strings"
	"time"

	cloudclient "github.com/anytty/anytty/cloud/client"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
)

// StatusSnapshot is a read-only projection of daemon-local Cloud runtime state.
type StatusSnapshot struct {
	DaemonID, AccountID                                 string
	EdgeID, EdgeName, EdgeRegion, PublicEndpoint        string
	ServerName                                          string
	LifecycleState                                      string
	LifecycleRevision                                   uint64
	Ready                                               bool
	ActiveSessions                                      int
	EnrolledAt, AgentReadyAt, UpdatedAt                 time.Time
	EnrollmentDeleted, ActiveAttempt, HasLifecycleState bool
	EntitlementFailure                                  *cloudv1.CloudEntitlementFailure
}

func (runtime *Runtime) StatusSnapshot() StatusSnapshot {
	if runtime == nil {
		return StatusSnapshot{}
	}
	record := runtime.currentRecord()
	locator := &cloudv1.EdgeLocator{}
	_ = proto.Unmarshal(record.EdgeLocator, locator)

	runtime.lifecycleMu.Lock()
	daemonState := runtime.daemonState
	ready := runtime.cloudActiveLocked()
	activeSessions := len(runtime.cloudSessions)
	enrollmentDeleted := runtime.enrollmentDeleted
	runtime.lifecycleMu.Unlock()

	runtime.attemptMu.Lock()
	activeAttempt := runtime.activeAttemptID != ""
	runtime.attemptMu.Unlock()
	runtime.connectionFailureMu.RLock()
	connectionFailure := runtime.connectionFailure
	if connectionFailure != nil {
		connectionFailure = proto.Clone(connectionFailure).(*cloudv1.CloudEntitlementFailure)
	}
	runtime.connectionFailureMu.RUnlock()

	snapshot := StatusSnapshot{
		DaemonID: record.DaemonID, AccountID: record.AccountID,
		EdgeID: locator.GetEdgeId(), EdgeName: locator.GetName(), EdgeRegion: locator.GetRegion(),
		PublicEndpoint: locator.GetPublicEndpoint(), ServerName: locator.GetServerName(),
		Ready: ready, ActiveSessions: activeSessions, EnrolledAt: record.EnrolledAt, UpdatedAt: time.Now().UTC(),
		EnrollmentDeleted: enrollmentDeleted, ActiveAttempt: activeAttempt, EntitlementFailure: connectionFailure,
	}
	if readyAt := runtime.agentReadyAt.Load(); readyAt != 0 {
		snapshot.AgentReadyAt = time.Unix(0, readyAt).UTC()
	}
	if daemonState != nil {
		snapshot.HasLifecycleState = true
		snapshot.LifecycleState = normalizeDaemonState(daemonState.GetState())
		snapshot.LifecycleRevision = daemonState.GetStateRevision()
	}
	return snapshot
}

func (runtime *Runtime) recordConnectionFailure(err error) {
	if runtime == nil {
		return
	}
	failure := cloudclient.EntitlementFailure(err)
	runtime.connectionFailureMu.Lock()
	defer runtime.connectionFailureMu.Unlock()
	runtime.connectionFailure = failure
}

func normalizeDaemonState(state cloudv1.DaemonState) string {
	value := strings.TrimPrefix(state.String(), "DAEMON_STATE_")
	if value == "" || value == "UNSPECIFIED" {
		return ""
	}
	return strings.ToLower(value)
}
