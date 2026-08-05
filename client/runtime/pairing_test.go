package runtime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
)

func TestPairingServiceRejectsIdentityMismatchAndClosesPeerExactlyOnce(t *testing.T) {
	request := pairingAttempt(t)
	peer := &fakePairingPeer{fingerprint: "SHA256:dtls"}
	_, err := (PairingService{}).Redeem(context.Background(), request, peer, remoteauth.ClientPairingRequest{
		ExpectedDeviceID: "different-device", ExpectedDeviceFingerprint: request.DaemonIdentity().DeviceFingerprint,
		PairingClaimOffer: []byte("claim"), Signer: fakePairingSigner{},
	})
	if CodeOf(err) != ErrorIdentity || peer.closeCalls.Load() != 1 {
		t.Fatalf("pairing error=%v close_calls=%d", err, peer.closeCalls.Load())
	}
}

func TestPairingServiceClosesPeerWhenFingerprintReadFails(t *testing.T) {
	request := pairingAttempt(t)
	peer := &fakePairingPeer{connection: &fakePairingTransport{}, fingerprintErr: errors.New("fingerprint unavailable")}
	_, err := (PairingService{}).Redeem(context.Background(), request, peer, remoteauth.ClientPairingRequest{
		ExpectedDeviceID: request.DaemonIdentity().DeviceID, ExpectedDeviceFingerprint: request.DaemonIdentity().DeviceFingerprint,
		PairingClaimOffer: []byte("claim"), Signer: fakePairingSigner{},
	})
	if CodeOf(err) != ErrorIdentity || peer.closeCalls.Load() != 1 {
		t.Fatalf("pairing error=%v close_calls=%d", err, peer.closeCalls.Load())
	}
}

func pairingAttempt(t *testing.T) AttemptRequest {
	t.Helper()
	identity := endpoint.DaemonIdentity{DeviceID: "device-pair", DeviceFingerprint: "SHA256:pair"}
	target := endpoint.Endpoint{
		ID: "pair", Label: "Pair", LabelSource: endpoint.SourceUser, DaemonIdentity: identity,
		ConnectMode: endpoint.ConnectOnDemand, Enabled: true,
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"direct": {
			ID: "direct", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceBootstrap, PolicySource: endpoint.SourceUser,
			SignalingAddresses: []string{"pair.local:41120"}, ICETCPAddresses: []string{"pair.local:41121"},
		}},
	}
	request, err := NewAttemptRequest(target, "direct", 1, ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

type fakePairingSigner struct{}

func (fakePairingSigner) Sign(context.Context, []byte) ([]byte, error) { return nil, nil }

type fakePairingPeer struct {
	connection     transport.Transport
	fingerprint    string
	fingerprintErr error
	closeErr       error
	closeCalls     atomic.Int32
}

func (peer *fakePairingPeer) Transport() transport.Transport { return peer.connection }

func (peer *fakePairingPeer) RemoteCertificateFingerprint() (string, error) {
	return peer.fingerprint, peer.fingerprintErr
}

func (peer *fakePairingPeer) Close() error {
	peer.closeCalls.Add(1)
	return peer.closeErr
}

var _ PairingPeerSession = (*fakePairingPeer)(nil)

type fakePairingTransport struct{}

func (*fakePairingTransport) Send([]byte) error     { return nil }
func (*fakePairingTransport) Recv() ([]byte, error) { return nil, errors.New("closed") }
func (*fakePairingTransport) Close() error          { return nil }
func (*fakePairingTransport) Done() <-chan struct{} { return make(chan struct{}) }

var _ transport.Transport = (*fakePairingTransport)(nil)
