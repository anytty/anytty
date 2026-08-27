package daemon

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
	"time"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"google.golang.org/protobuf/proto"
)

func TestEnrollmentRecordV3RoundTripsAndRejectsV2(t *testing.T) {
	locatorPayload, err := proto.Marshal(&cloudv1.EdgeLocator{EdgeId: "edge-1", PublicEndpoint: "edge.example:443", ServerName: "edge.example", CaCertificatePem: []byte("ca")})
	if err != nil {
		t.Fatal(err)
	}
	locatorDigest := sha256.Sum256(locatorPayload)
	claimsPayload, err := proto.Marshal(&cloudv1.DaemonBindingClaims{DaemonId: "daemon-1", AccountId: "account-1", EdgeId: "edge-1", EdgeLocatorSha256: locatorDigest[:]})
	if err != nil {
		t.Fatal(err)
	}
	bindingPayload, err := proto.Marshal(&cloudv1.SignedEnvelope{KeyId: "binding-key", Payload: claimsPayload, Signature: []byte("signature")})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cloud-enrollment.json")
	record := EnrollmentRecord{DaemonID: "daemon-1", AccountID: "account-1", DisplayName: "Office Mac", DaemonBinding: bindingPayload, EdgeLocator: locatorPayload, EnrolledAt: time.Now().UTC()}
	if err := SaveRecord(path, record); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRecord(path)
	if err != nil || loaded.Version != recordVersion || loaded.DaemonID != record.DaemonID {
		t.Fatalf("loaded record=%+v err=%v", loaded, err)
	}
	tampered := loaded
	tampered.EdgeLocator = append([]byte(nil), loaded.EdgeLocator...)
	tampered.EdgeLocator[len(tampered.EdgeLocator)-1] ^= 0xff
	if err := tampered.Validate(); err == nil {
		t.Fatal("record accepted an Edge locator not covered by the daemon binding")
	}
	if err := os.WriteFile(path, []byte(`{"version":2,"daemon_id":"daemon-1","account_id":"account-1","daemon_binding":"","edge_locator":"","enrolled_at":"2026-07-29T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRecord(path); err == nil {
		t.Fatal("v2 enrollment record was accepted")
	}
}
