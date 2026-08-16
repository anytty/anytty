package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	corev2 "github.com/anytty/anytty/core"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
)

const v3CloudDisabledSchemaVersion = 1

type v3CloudDisabledRecord struct {
	SchemaVersion int       `json:"schema_version"`
	Disabled      bool      `json:"disabled"`
	Reason        string    `json:"reason,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func v3CloudDisabledPath() string {
	return filepath.Join(v3RemoteIdentityDir(), "cloud_disabled.json")
}

func writeV3CloudDisabled(reason string) error {
	path := filepath.Clean(v3CloudDisabledPath())
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return err
	}
	record := v3CloudDisabledRecord{SchemaVersion: v3CloudDisabledSchemaVersion, Disabled: true, Reason: strings.TrimSpace(reason), UpdatedAt: time.Now().UTC()}
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

func removeV3CloudDisabled() error {
	err := os.Remove(filepath.Clean(v3CloudDisabledPath()))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func readV3CloudDisabled(path string) (v3CloudDisabledRecord, bool, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, os.ErrNotExist) {
		return v3CloudDisabledRecord{}, false, nil
	}
	if err != nil {
		return v3CloudDisabledRecord{}, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var record v3CloudDisabledRecord
	if err := decoder.Decode(&record); err != nil {
		return v3CloudDisabledRecord{}, false, fmt.Errorf("decode Cloud disabled marker: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return v3CloudDisabledRecord{}, false, errors.New("Cloud disabled marker has trailing data")
	}
	if record.SchemaVersion != v3CloudDisabledSchemaVersion || !record.Disabled || record.UpdatedAt.IsZero() {
		return v3CloudDisabledRecord{}, false, errors.New("Cloud disabled marker metadata is invalid")
	}
	return record, true, nil
}

func loadV3CloudStatus(recordPath, disabledPath string, runtime *clouddaemon.Runtime, daemonRunning bool, lastRuntimeError string) (corev2.RemoteCloudStatus, error) {
	now := time.Now().UTC()
	disabledRecord, disabled, err := readV3CloudDisabled(disabledPath)
	if err != nil {
		return corev2.RemoteCloudStatus{}, err
	}
	status := corev2.RemoteCloudStatus{
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
		return corev2.RemoteCloudStatus{}, err
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
	return status, nil
}

func applyCloudRecordLocator(status *corev2.RemoteCloudStatus, record clouddaemon.EnrollmentRecord) {
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
