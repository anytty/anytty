package share_test

import (
	"context"
	"net"
	"testing"
	"time"

	shareadapter "github.com/anytty/anytty/client/adapter/share"
	"github.com/anytty/anytty/client/endpoint"
	"google.golang.org/protobuf/proto"
)

func TestShareSessionCompletesReceiverProofAndConsumesOnce(t *testing.T) {
	server, offer := newShareServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	served := make(chan error, 1)
	go func() { served <- server.Serve(ctx) }()
	bundle, err := shareadapter.Receive(ctx, offer)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.GetTransferId() != offer.GetTransferId() {
		t.Fatalf("received transfer=%q", bundle.GetTransferId())
	}
	if err := <-served; err != nil {
		t.Fatal(err)
	}

	replayCtx, replayCancel := context.WithTimeout(context.Background(), time.Second)
	defer replayCancel()
	go func() { _ = server.Serve(replayCtx) }()
	if _, err := shareadapter.Receive(replayCtx, offer); err == nil {
		t.Fatal("consumed share session unexpectedly replayed")
	}
}

func TestShareSessionRejectsCertificatePinMismatch(t *testing.T) {
	server, offer := newShareServer(t)
	offer = proto.Clone(offer).(*endpoint.ShareSessionOffer)
	offer.EphemeralCertificateSha256 = "sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() { _ = server.Serve(ctx) }()
	if _, err := shareadapter.Receive(ctx, offer); err == nil {
		t.Fatal("share certificate pin mismatch unexpectedly succeeded")
	}
}

func TestShareOfferExpiryFailsBeforeDial(t *testing.T) {
	_, offer := newShareServer(t)
	offer = proto.Clone(offer).(*endpoint.ShareSessionOffer)
	offer.ExpiresAtUnixNano = time.Now().Add(-time.Second).UnixNano()
	if _, err := shareadapter.Receive(context.Background(), offer); err == nil {
		t.Fatal("expired share offer unexpectedly dialed")
	}
}

func newShareServer(t *testing.T) (*shareadapter.Server, *endpoint.ShareSessionOffer) {
	t.Helper()
	target := endpoint.Endpoint{
		ID: "studio", Label: "Studio", LabelSource: endpoint.SourceUser,
		DaemonIdentity: endpoint.DaemonIdentity{DeviceID: "device-studio", DeviceFingerprint: "SHA256:studio"},
		ConnectMode:    endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{
			"direct": {ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceManual, PolicySource: endpoint.SourceManual, SignalingAddresses: []string{"studio:41120"}, ICETCPAddresses: []string{"studio:41121"}},
		},
	}
	bundle, err := endpoint.NewClientEndpointShareBundle(target, "share-test", time.Now(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, err := shareadapter.NewServer(shareadapter.ServerOptions{Listener: listener, AdvertisedAddresses: []string{listener.Addr().String()}, Bundle: bundle, TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server, server.Offer()
}
