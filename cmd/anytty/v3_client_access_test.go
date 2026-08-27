package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	corev2 "github.com/anytty/anytty/core"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
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
	if label := v3DefaultPairingLabel(); label != record.DisplayName {
		t.Fatalf("default pairing label = %q", label)
	}
}

func TestClientAccessTicketUsesDaemonDefaultLabel(t *testing.T) {
	identity, err := remoteauth.NewIdentity("device-default-label", ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x31}, ed25519.SeedSize)))
	if err != nil {
		t.Fatal(err)
	}
	store, err := remoteauth.LoadAccessStore(t.TempDir(), identity, remoteauth.AccessStoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := v3ClientAccessService{identity: identity, store: store, defaultLabel: func() string { return "Beijing Office Mac" }}
	ticket, err := service.CreateTicket(context.Background(), corev2.ClientAccessTicketRequest{
		Scope:     corev2.ClientAccessScope{AllowDaemon: true},
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
