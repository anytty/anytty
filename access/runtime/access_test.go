package accessruntime_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	accesscontract "github.com/anytty/anytty/access/contract"
	accessruntime "github.com/anytty/anytty/access/runtime"
	"github.com/anytty/anytty/access/sessions"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestClientAccessTicketUsesOwnerDefaultLabel(t *testing.T) {
	identity, err := remoteauth.NewIdentity("device-default-label", ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x31}, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	store, err := remoteauth.LoadAccessStore(t.TempDir(), identity, remoteauth.AccessStoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := accessruntime.Service{DeviceIdentity: identity, Store: store, DefaultLabel: func() string { return "Beijing Office Mac" }}
	ticket, err := service.CreateTicket(context.Background(), accesscontract.ClientAccessTicketRequest{
		Scope:     accesscontract.ClientAccessScope{AllowDaemon: true},
		TicketTTL: time.Minute,
		Routes: []*remoteauthpb.EndpointRouteConfigV1{{
			SchemaVersion: 1, RouteId: "direct", Enabled: true,
			Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{
				SignalingAddresses: []string{"127.0.0.1:41120"}, IceTcpAddresses: []string{"127.0.0.1:41121"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := remoteauth.GenerateClientAccessIdentity("test-client", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, bundlePayload, err := store.RedeemPairingClaim(ticket.ClaimOffer, client.PublicKey, "test-client", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := remoteauth.ParsePairingBundleForExchange(bundlePayload)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.GetSuggestedLabel() != "Beijing Office Mac" {
		t.Fatalf("pairing bundle label = %q", bundle.GetSuggestedLabel())
	}
}

func TestServiceRevokeClosesTrackedSessions(t *testing.T) {
	identity, err := remoteauth.NewIdentity("device-revoke", ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x32}, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	store, err := remoteauth.LoadAccessStore(t.TempDir(), identity, remoteauth.AccessStoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("test-client", nil)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := store.IssuePairingBundle(remoteauth.PairingIssueOptions{Scope: remoteauth.Scope{AllowDaemon: true}, TicketTTL: time.Minute, GrantLifetime: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := remoteauth.EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := store.RedeemPairingBundle(payload, clientIdentity.PublicKey, "test-client", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	registry := sessions.NewRegistry()
	client, server := memory.NewPair()
	defer client.Close()
	release := registry.Track(exchanged.GrantID, server, time.Time{})
	defer release()
	service := accessruntime.Service{DeviceIdentity: identity, Store: store, Sessions: registry}
	if _, err := service.Revoke(context.Background(), exchanged.GrantID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Recv(); err == nil {
		t.Fatal("tracked session survived revoke")
	}
	if registry.ActiveGrant(exchanged.GrantID) != 0 {
		t.Fatalf("active sessions after revoke = %d", registry.ActiveGrant(exchanged.GrantID))
	}
}
