package endpoint

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCONN002BootstrapFixtureUsesCanonicalGoParser(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "clients", "mobile", "android", "app", "src", "test", "resources", "pairing_bootstrap_v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Now              string `json:"now"`
		PayloadBase64URL string `json:"payload_base64url"`
		DeviceID         string `json:"device_id"`
		TicketID         string `json:"ticket_id"`
	}
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatal(err)
	}
	now, err := time.Parse(time.RFC3339Nano, fixture.Now)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := base64.RawURLEncoding.DecodeString(fixture.PayloadBase64URL)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseEndpointBootstrapBundleAt(wire, now)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.GetIdentity().GetDeviceId() != fixture.DeviceID || bundle.GetAuthorization().GetPairingTicket().GetTicketId() != fixture.TicketID {
		t.Fatalf("fixture identity mismatch: bundle=%#v fixture=%#v", bundle, fixture)
	}
}
