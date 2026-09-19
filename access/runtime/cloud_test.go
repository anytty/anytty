package accessruntime

import (
	"context"
	"testing"
	"time"

	clouddaemon "github.com/anytty/anytty/daemon/cloud"
)

func TestCloudRuntimeControlRestartsChangedEnrollment(t *testing.T) {
	oldRecord := clouddaemon.EnrollmentRecord{DaemonID: "daemon-old", AccountID: "account", EnrolledAt: time.Unix(1, 0).UTC()}
	newRecord := clouddaemon.EnrollmentRecord{DaemonID: "daemon-new", AccountID: "account", EnrolledAt: time.Unix(2, 0).UTC()}
	canceled := false
	control := &CloudControl{
		runtime:           new(clouddaemon.Runtime),
		runtimeCancel:     func() { canceled = true },
		runtimeEnrollment: cloudEnrollmentIdentityFromRecord(oldRecord),
	}
	if !control.restartRuntimeForEnrollment(newRecord) || !canceled {
		t.Fatal("changed enrollment did not cancel the stale Cloud runtime")
	}
}

func TestCloudRuntimeControlKeepsMatchingEnrollment(t *testing.T) {
	record := clouddaemon.EnrollmentRecord{DaemonID: "daemon", AccountID: "account", EnrolledAt: time.Unix(1, 0).UTC()}
	canceled := false
	control := &CloudControl{
		runtime:           new(clouddaemon.Runtime),
		runtimeCancel:     func() { canceled = true },
		runtimeEnrollment: cloudEnrollmentIdentityFromRecord(record),
	}
	if control.restartRuntimeForEnrollment(record) || canceled {
		t.Fatal("matching enrollment restarted the Cloud runtime")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	control.waitRuntimeEnrollment(ctx, record, time.Second)
}

func TestCloudRuntimeStatusRejectsStaleEnrollment(t *testing.T) {
	record := clouddaemon.EnrollmentRecord{DaemonID: "daemon-new", AccountID: "account", EnrolledAt: time.Unix(2, 0).UTC()}
	stale := clouddaemon.StatusSnapshot{DaemonID: "daemon-old", AccountID: "account", EnrolledAt: time.Unix(1, 0).UTC(), Ready: true}
	if cloudRuntimeMatchesEnrollment(record, stale) {
		t.Fatal("stale ready runtime matched the replacement enrollment")
	}
	current := clouddaemon.StatusSnapshot{DaemonID: record.DaemonID, AccountID: record.AccountID, EnrolledAt: record.EnrolledAt, Ready: true}
	if !cloudRuntimeMatchesEnrollment(record, current) {
		t.Fatal("current runtime did not match its enrollment")
	}
}
