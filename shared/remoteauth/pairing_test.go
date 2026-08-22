package remoteauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func TestPairingBundleCarriesOnlySignedOneTimeTicket(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-pairing", privateKey)
	now := time.Date(2026, 7, 11, 15, 0, 0, 0, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, issued, err := store.IssuePairingBundle(PairingIssueOptions{
		Label: "Lab daemon", Platform: "darwin", Scope: Scope{AllowDaemon: true}, TicketTTL: time.Hour, GrantLifetime: 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "capability_grant") || strings.Contains(string(payload), "private_key") {
		t.Fatalf("pairing bundle leaked long-lived credential: %s", payload)
	}
	parsed, claims, err := ParsePairingBundle(payload, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.GetIdentity().GetDeviceId() != identity.DeviceID || parsed.GetIdentity().GetDeviceFingerprint() != identity.Fingerprint || parsed.GetSuggestedLabel() != "Lab daemon" || parsed.GetPlatform() != "darwin" || claims.TicketID != issued.TicketID || claims.MaxRedemptions != 1 || !claims.ScopeCeiling.AllowDaemon {
		t.Fatalf("pairing round trip mismatch: bundle=%#v claims=%#v", parsed, claims)
	}
}

func TestPairingBundleRejectsUnknownFieldExpiryAndIdentityMismatch(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-pairing", privateKey)
	now := time.Date(2026, 7, 11, 15, 0, 0, 0, time.UTC)
	store, _ := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: func() time.Time { return now }})
	t.Cleanup(func() { _ = store.Close() })
	bundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{TerminalID: "term-1"}, TicketTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	unknownBundle := proto.Clone(bundle).(*PairingBundle)
	unknownBundle.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x01})
	unknown, _ := proto.Marshal(unknownBundle)
	if _, _, err := ParsePairingBundle(unknown, now); err == nil {
		t.Fatal("unknown Hub field was accepted")
	}
	bundle.Identity.DeviceId = "device-other"
	payload, _ := proto.Marshal(bundle)
	if _, _, err := ParsePairingBundle(payload, now); err == nil {
		t.Fatal("pairing identity mismatch was accepted")
	}
	validBundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{TerminalID: "term-expiry"}, TicketTTL: time.Minute, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	validPayload, _ := EncodePairingBundle(validBundle)
	if _, err := VerifyPairingTicket(validPayload, identity.Fingerprint, now.Add(time.Minute+ClockSkewTolerance)); !errors.Is(err, ErrPairingTicketExpired) {
		t.Fatalf("ticket expiry error = %v", err)
	}
}

func TestPairingLabelsRejectControlCharactersAndOversizeValues(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	identity, _ := NewIdentity("device-pairing", privateKey)
	now := time.Date(2026, 7, 11, 15, 0, 0, 0, time.UTC)
	store, _ := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: func() time.Time { return now }})
	t.Cleanup(func() { _ = store.Close() })
	if _, _, err := store.IssuePairingBundle(PairingIssueOptions{Label: "bad\nlabel", Scope: Scope{AllowDaemon: true}}); err == nil {
		t.Fatal("control character pairing label was accepted")
	}
	if _, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{AllowDaemon: true}, TicketTTL: maxPairingTicketTTL + time.Second}); err == nil {
		t.Fatal("long-lived pairing ticket was accepted")
	}
	if _, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: Scope{AllowDaemon: true}, GrantLifetime: maxPairingGrantTTL + time.Second}); err == nil {
		t.Fatal("overlong bound grant lifetime was accepted")
	}
	bundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Label: "valid", Scope: Scope{AllowDaemon: true}})
	if err != nil {
		t.Fatal(err)
	}
	bundle.SuggestedLabel = strings.Repeat("x", maxPairingLabelBytes+1)
	if _, err := EncodePairingBundle(bundle); err == nil {
		t.Fatal("oversize pairing label was encoded")
	}
	bundle.SuggestedLabel = "valid"
	client, _ := GenerateClientAccessIdentity("endpoint-label", rand.Reader)
	payload, _ := EncodePairingBundle(bundle)
	if _, err := store.RedeemPairingBundle(payload, client.PublicKey, "bad\tclient", now); err == nil {
		t.Fatal("control character client label was persisted")
	}
}
