package daemon

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
	"time"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
)

func TestApplyBlockedDrainsCloudSessionsBeforeReturning(t *testing.T) {
	sessionCtx, cancelSession := context.WithCancel(context.Background())
	session := &cloudSession{cancel: cancelSession, done: make(chan struct{})}
	go func() {
		<-sessionCtx.Done()
		close(session.done)
	}()
	runtime := &Runtime{
		config:            Config{Record: EnrollmentRecord{DaemonID: "daemon"}},
		record:            EnrollmentRecord{DaemonID: "daemon"},
		daemonState:       daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_ACTIVE, 1),
		readyConnectionID: "connection",
		lifecycleAck:      1,
		cloudSessions:     map[string]*cloudSession{"session": session},
	}

	if err := runtime.applyDaemonState(context.Background(), daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_BLOCKED, 2)); err != nil {
		t.Fatal(err)
	}
	if runtime.cloudActive() {
		t.Fatal("blocked daemon still accepts Cloud sessions")
	}
	select {
	case <-session.done:
	default:
		t.Fatal("blocked state returned before the Cloud session drained")
	}
}

func TestApplyDeletedRemovesOnlyCloudEnrollmentRecord(t *testing.T) {
	directory := t.TempDir()
	recordPath := filepath.Join(directory, "cloud.json")
	identityPath := filepath.Join(directory, "device.key")
	if err := os.WriteFile(recordPath, []byte("cloud"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identityPath, []byte("identity"), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.LoadOrCreateLocalIdentity(filepath.Join(directory, "identity"))
	if err != nil {
		t.Fatal(err)
	}
	accessStore, err := remoteauth.LoadAccessStore(filepath.Join(directory, "access"), identity, remoteauth.AccessStoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer accessStore.Close()
	if err := accessStore.ConfigureManagedRouteGrantIssuer(func(ed25519.PublicKey, uint32, time.Time, time.Time) ([]byte, []byte, error) {
		return []byte("grant"), []byte("locator"), nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := accessStore.ConfigureManagedPairingBootstrapIssuer(func() (*remoteauthpb.PairingManagedRouteSeed, error) {
		return &remoteauthpb.PairingManagedRouteSeed{DaemonId: "daemon"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{
		config:        Config{Record: EnrollmentRecord{DaemonID: "daemon"}, RecordPath: recordPath, AccessStore: accessStore},
		record:        EnrollmentRecord{DaemonID: "daemon"},
		daemonState:   daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_BLOCKED, 2),
		cloudSessions: make(map[string]*cloudSession),
	}

	if err := runtime.applyDaemonState(context.Background(), daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_DELETED, 3)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("Cloud enrollment record still exists: %v", err)
	}
	if !runtime.daemonDeleted() {
		t.Fatal("daemon did not stop Cloud reconnect after deleting enrollment")
	}
	if err := accessStore.ConfigureManagedRouteGrantIssuer(func(ed25519.PublicKey, uint32, time.Time, time.Time) ([]byte, []byte, error) { return nil, nil, nil }); err != nil {
		t.Fatalf("Cloud route issuer was not released: %v", err)
	}
	if err := accessStore.ConfigureManagedPairingBootstrapIssuer(func() (*remoteauthpb.PairingManagedRouteSeed, error) { return nil, nil }); err != nil {
		t.Fatalf("Cloud pairing issuer was not released: %v", err)
	}
	if payload, err := os.ReadFile(identityPath); err != nil || string(payload) != "identity" {
		t.Fatalf("device identity changed: payload=%q err=%v", payload, err)
	}
}

func TestCloudActiveRequiresCurrentConnectionLifecycleAcknowledgement(t *testing.T) {
	runtime := &Runtime{
		record:        EnrollmentRecord{DaemonID: "daemon"},
		daemonState:   daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_ACTIVE, 4),
		cloudSessions: make(map[string]*cloudSession),
	}
	if runtime.cloudActive() {
		t.Fatal("disconnected daemon reported Cloud active")
	}
	runtime.markAgentReady("connection-new")
	if runtime.cloudActive() {
		t.Fatal("unacknowledged lifecycle reported Cloud active")
	}
	runtime.markLifecycleAcknowledged("connection-old", 4)
	if runtime.cloudActive() {
		t.Fatal("stale connection acknowledgement reported Cloud active")
	}
	runtime.markLifecycleAcknowledged("connection-new", 4)
	if !runtime.cloudActive() {
		t.Fatal("ready connection with acknowledged ACTIVE lifecycle did not report Cloud active")
	}
	if err := runtime.applyDaemonState(context.Background(), daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_BLOCKED, 5)); err != nil {
		t.Fatal(err)
	}
	if runtime.cloudActive() {
		t.Fatal("BLOCKED lifecycle remained Cloud active")
	}
	if err := runtime.applyDaemonState(context.Background(), daemonLifecycleState("daemon", cloudv1.DaemonState_DAEMON_STATE_ACTIVE, 6)); err != nil {
		t.Fatal(err)
	}
	runtime.markLifecycleAcknowledged("connection-new", 5)
	if runtime.cloudActive() {
		t.Fatal("restored ACTIVE lifecycle accepted stale acknowledgement")
	}
	runtime.markLifecycleAcknowledged("connection-new", 6)
	if !runtime.cloudActive() {
		t.Fatal("restored ACTIVE lifecycle did not accept its current acknowledgement")
	}
	runtime.clearAgentReady("connection-old")
	if !runtime.cloudActive() {
		t.Fatal("stale connection cleanup cleared the current connection")
	}
	runtime.clearAgentReady("connection-new")
	if runtime.cloudActive() {
		t.Fatal("disconnected current connection remained Cloud active")
	}
}

func daemonLifecycleState(daemonID string, state cloudv1.DaemonState, revision uint64) *cloudv1.DaemonStateRecord {
	return &cloudv1.DaemonStateRecord{DaemonId: daemonID, State: state, StateRevision: revision}
}
