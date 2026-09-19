package accessruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	accesscontract "github.com/anytty/anytty/access/contract"
	clouddaemon "github.com/anytty/anytty/daemon/cloud"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
)

const cloudDisabledSchemaVersion = 1

type cloudDisabledRecord struct {
	SchemaVersion int       `json:"schema_version"`
	Disabled      bool      `json:"disabled"`
	Reason        string    `json:"reason,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func writeCloudDisabled(reason string) error {
	path := filepath.Clean(DisabledPath())
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	record := cloudDisabledRecord{SchemaVersion: cloudDisabledSchemaVersion, Disabled: true, Reason: strings.TrimSpace(reason), UpdatedAt: time.Now().UTC()}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	temp, err := os.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func removeCloudDisabled() error {
	err := os.Remove(filepath.Clean(DisabledPath()))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func readCloudDisabled(path string) (cloudDisabledRecord, bool, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, os.ErrNotExist) {
		return cloudDisabledRecord{}, false, nil
	}
	if err != nil {
		return cloudDisabledRecord{}, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var record cloudDisabledRecord
	if err := decoder.Decode(&record); err != nil {
		return cloudDisabledRecord{}, false, fmt.Errorf("decode Cloud disabled marker: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return cloudDisabledRecord{}, false, errors.New("Cloud disabled marker has trailing data")
	}
	if record.SchemaVersion != cloudDisabledSchemaVersion || !record.Disabled || record.UpdatedAt.IsZero() {
		return cloudDisabledRecord{}, false, errors.New("Cloud disabled marker metadata is invalid")
	}
	return record, true, nil
}

// WriteCloudDisabled 原子写入 Cloud runtime disabled marker。
func WriteCloudDisabled(reason string) error {
	return writeCloudDisabled(reason)
}

// RemoveCloudDisabled 删除 Cloud runtime disabled marker；不存在视为成功。
func RemoveCloudDisabled() error {
	return removeCloudDisabled()
}

// CloudDisabled 报告持久 disabled marker 是否存在；读取错误必须 fail closed。
func CloudDisabled(path string) (bool, error) {
	_, disabled, err := readCloudDisabled(path)
	return disabled, err
}

// LoadCloudStatus 把 enrollment record 与 runtime 快照投影为 core-native Cloud 状态。
func LoadCloudStatus(recordPath, disabledPath string, runtime *clouddaemon.Runtime, daemonRunning bool, lastRuntimeError string) (accesscontract.RemoteCloudStatus, error) {
	now := time.Now().UTC()
	disabledRecord, disabled, err := readCloudDisabled(disabledPath)
	if err != nil {
		return accesscontract.RemoteCloudStatus{}, err
	}
	status := accesscontract.RemoteCloudStatus{
		Enabled: !disabled, Running: runtime != nil, RecordPath: filepath.Clean(recordPath), DisabledPath: filepath.Clean(disabledPath),
		UpdatedAt: now,
	}
	if disabled {
		status.UpdatedAt = disabledRecord.UpdatedAt
		status.State = "disabled"
		status.Detail = "Cloud runtime is temporarily disabled on this daemon"
	}

	record, err := clouddaemon.LoadRecord(recordPath)
	if errors.Is(err, os.ErrNotExist) {
		status.Enrolled = false
		if disabled {
			status.Detail = "Cloud runtime is disabled, and no enrollment record exists"
			return status, nil
		}
		status.State = "not_enrolled"
		status.Detail = "Cloud enrollment record was not found"
		return status, nil
	}
	if err != nil {
		return accesscontract.RemoteCloudStatus{}, err
	}
	status.Enrolled = true
	status.DaemonID = record.DaemonID
	status.AccountID = record.AccountID
	status.EnrolledAt = record.EnrolledAt
	applyCloudRecordLocator(&status, record)

	if disabled {
		return status, nil
	}
	if runtime == nil {
		if daemonRunning {
			status.State = "starting"
			status.Detail = "Cloud runtime is waiting to start"
			if strings.TrimSpace(lastRuntimeError) != "" {
				status.Detail = strings.TrimSpace(lastRuntimeError)
			}
			return status, nil
		}
		status.State = "daemon_stopped"
		status.Detail = "local daemon is not running"
		return status, nil
	}

	snapshot := runtime.StatusSnapshot()
	status.Running = true
	if !cloudRuntimeMatchesEnrollment(record, snapshot) {
		status.State = "enrollment_changed"
		status.Detail = "Cloud enrollment changed; the daemon runtime is restarting"
		return status, nil
	}
	status.Ready = snapshot.Ready
	status.DaemonID = firstNonEmpty(snapshot.DaemonID, status.DaemonID)
	status.AccountID = firstNonEmpty(snapshot.AccountID, status.AccountID)
	status.EdgeID = firstNonEmpty(snapshot.EdgeID, status.EdgeID)
	status.EdgeName = firstNonEmpty(snapshot.EdgeName, status.EdgeName)
	status.EdgeRegion = firstNonEmpty(snapshot.EdgeRegion, status.EdgeRegion)
	status.PublicEndpoint = firstNonEmpty(snapshot.PublicEndpoint, status.PublicEndpoint)
	status.ServerName = firstNonEmpty(snapshot.ServerName, status.ServerName)
	status.LifecycleState = snapshot.LifecycleState
	status.LifecycleRevision = snapshot.LifecycleRevision
	status.ActiveSessions = snapshot.ActiveSessions
	if !snapshot.UpdatedAt.IsZero() {
		status.UpdatedAt = snapshot.UpdatedAt
	}
	switch {
	case snapshot.EnrollmentDeleted:
		status.State = "deleted"
		status.Detail = "Controller marked this daemon as deleted"
	case snapshot.Ready:
		status.State = "online"
		status.Detail = "Cloud AgentGateway is connected"
	case snapshot.LifecycleState == "blocked":
		status.State = "blocked"
		status.Detail = "Controller blocked this daemon"
	case snapshot.LifecycleState == "deleted":
		status.State = "deleted"
		status.Detail = "Controller marked this daemon as deleted"
	case snapshot.ActiveAttempt:
		status.State = "connecting"
		status.Detail = "Cloud runtime is connecting to the selected Edge"
	default:
		status.State = "starting"
		status.Detail = "Cloud runtime is waiting for Edge readiness"
	}
	if strings.TrimSpace(lastRuntimeError) != "" && status.State != "online" {
		status.Detail = strings.TrimSpace(lastRuntimeError)
	}
	if snapshot.EntitlementFailure != nil && status.State != "online" {
		status.State, status.Detail = cloudEntitlementRuntimeStatus(snapshot.EntitlementFailure)
	}
	return status, nil
}

func cloudRuntimeMatchesEnrollment(record clouddaemon.EnrollmentRecord, snapshot clouddaemon.StatusSnapshot) bool {
	return snapshot.DaemonID == record.DaemonID && snapshot.AccountID == record.AccountID && snapshot.EnrolledAt.Equal(record.EnrolledAt)
}

func cloudEntitlementRuntimeStatus(failure *cloudv1.CloudEntitlementFailure) (string, string) {
	if failure == nil {
		return "entitlement_denied", "AnyTTY Cloud access is not allowed by the current plan"
	}
	switch failure.GetCode() {
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED:
		limit := ""
		if failure.GetLimit() > 0 {
			limit = fmt.Sprintf(" (limit %d)", failure.GetLimit())
		}
		return "quota_limited", "AnyTTY Cloud daemon connection limit is reached" + limit + "; stop another Cloud daemon or upgrade at https://cloud.anytty.com/app/subscription. Direct and SSH remain available"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE:
		return "subscription_inactive", "AnyTTY Cloud subscription is inactive; renew it at https://cloud.anytty.com/app/subscription. Direct and SSH remain available"
	case cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SERVICE_UNAVAILABLE:
		return "degraded", "AnyTTY Cloud entitlement is temporarily unavailable; retry later. Direct and SSH remain available"
	default:
		return "entitlement_denied", "AnyTTY Cloud access is not allowed by the current plan; review https://cloud.anytty.com/app/subscription"
	}
}

func applyCloudRecordLocator(status *accesscontract.RemoteCloudStatus, record clouddaemon.EnrollmentRecord) {
	locator := &cloudv1.EdgeLocator{}
	if proto.Unmarshal(record.EdgeLocator, locator) != nil {
		return
	}
	status.EdgeID = locator.GetEdgeId()
	status.EdgeName = locator.GetName()
	status.EdgeRegion = locator.GetRegion()
	status.PublicEndpoint = locator.GetPublicEndpoint()
	status.ServerName = locator.GetServerName()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
