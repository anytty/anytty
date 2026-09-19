package cli

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	accessruntime "github.com/anytty/anytty/access/runtime"
	clouddaemon "github.com/anytty/anytty/daemon/cloud"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
)

func TestDefaultPairingLabelUsesCloudEnrollmentName(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	identity, err := remoteauth.NewIdentity("device-cloud-label", ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x30}, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	record := cloudEdgeListE2ERecord(t, "daemon-cloud-label", "account-cloud-label", identity, &cloudv1.EdgeLocator{
		EdgeId: "edge-cloud-label", PublicEndpoint: "edge.example:41102", ServerName: "edge.example", CaCertificatePem: []byte("test-ca"),
	})
	record.DisplayName = "Shanghai Development Mac"
	if err := clouddaemon.SaveRecord(v3CloudEnrollmentRecordPath(), record); err != nil {
		t.Fatal(err)
	}
	if label := accessruntime.DefaultPairingLabelFromEnrollment(v3CloudEnrollmentRecordPath()); label != record.DisplayName {
		t.Fatalf("default pairing label = %q", label)
	}
}
