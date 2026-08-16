package remoteauth

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/memory"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestCapabilityHandshakeAuthenticatesClientKeyAndSwitchesProtocol(t *testing.T) {
	identity, credential, store, now := handshakeFixture(t, Scope{TerminalID: "term-1", ManageClientAccess: true})
	binding := fixtureDTLSBinding(t, 0x11)
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	defer serverConn.Close()
	type serverResult struct {
		result ServerHandshakeResult
		frame  []byte
		err    error
	}
	serverDone := make(chan serverResult, 1)
	go func() {
		result, err := (ServerHandshake{Identity: identity, AccessStore: store, Now: fixedNow(now)}).Accept(context.Background(), serverConn, binding)
		if err != nil {
			serverDone <- serverResult{err: err}
			return
		}
		frame, err := serverConn.Recv()
		serverDone <- serverResult{result: result, frame: frame, err: err}
	}()
	clientClaims, err := (ClientHandshake{Now: fixedNow(now)}).Authenticate(context.Background(), clientConn, ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: binding,
	})
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if clientClaims.Scope.TerminalID != "term-1" || !clientClaims.Scope.ManageClientAccess {
		t.Fatalf("client claims = %+v", clientClaims)
	}
	wantFrame := []byte{0, 0, 0, 0, 0, 0, 0}
	if err := clientConn.Send(wantFrame); err != nil {
		t.Fatal(err)
	}
	result := <-serverDone
	if result.err != nil || result.result.Mode != ServerHandshakeModeCapability || result.result.Claims.SubjectKeyFingerprint != credential.Identity.Fingerprint || !bytes.Equal(result.frame, wantFrame) {
		t.Fatalf("server result=%+v frame=%x err=%v", result.result, result.frame, result.err)
	}
}

func TestCopiedGrantWithoutBoundPrivateKeyFails(t *testing.T) {
	identity, credential, store, now := handshakeFixture(t, Scope{AllowDaemon: true})
	attacker, _ := GenerateClientAccessIdentity("endpoint-1", bytes.NewReader(bytes.Repeat([]byte{0x77}, 64)))
	credential.Identity = attacker
	binding := fixtureDTLSBinding(t, 0x11)
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		_, err := (ServerHandshake{Identity: identity, AccessStore: store, Now: fixedNow(now)}).Accept(context.Background(), serverConn, binding)
		serverDone <- err
	}()
	_, err := (ClientHandshake{Now: fixedNow(now)}).Authenticate(context.Background(), clientConn, ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: binding,
	})
	if HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH {
		t.Fatalf("copied grant error = %v", err)
	}
	_ = clientConn.Close()
	_ = serverConn.Close()
	<-serverDone
}

func TestClientRejectsDeviceHelloBoundToDifferentTransport(t *testing.T) {
	identity, credential, store, now := handshakeFixture(t, Scope{AllowDaemon: true})
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		_, err := (ServerHandshake{Identity: identity, AccessStore: store, Now: fixedNow(now)}).Accept(context.Background(), serverConn, fixtureDTLSBinding(t, 0x11))
		serverDone <- err
	}()
	_, err := (ClientHandshake{Now: fixedNow(now)}).Authenticate(context.Background(), clientConn, ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: fixtureDTLSBinding(t, 0x22),
	})
	if HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH {
		t.Fatalf("binding mismatch error = %v", err)
	}
	_ = clientConn.Close()
	_ = serverConn.Close()
	<-serverDone
}

func TestPairingHandshakeReturnsBoundGrantAndIdempotentReceipt(t *testing.T) {
	daemonSeed := bytes.Repeat([]byte{0x23}, ed25519.SeedSize)
	identity, _ := NewIdentity("device-1", ed25519.NewKeyFromSeed(daemonSeed))
	now := time.Date(2026, 7, 11, 12, 34, 56, 789000000, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: fixedNow(now)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	issued := issueHandshakePairingClaim(t, store, FullDaemonScope(), time.Hour, 24*time.Hour, now)
	client, _ := GenerateClientAccessIdentity("endpoint-1", bytes.NewReader(bytes.Repeat([]byte{0x51}, 64)))
	binding := fixtureDTLSBinding(t, 0x11)
	redeem := func() PairingExchangeResult {
		clientConn, serverConn := memory.NewPair()
		serverDone := make(chan ServerHandshakeResult, 1)
		serverErr := make(chan error, 1)
		go func() {
			result, acceptErr := (ServerHandshake{Identity: identity, AccessStore: store, Now: fixedNow(now)}).Accept(context.Background(), serverConn, binding)
			serverDone <- result
			serverErr <- acceptErr
		}()
		result, redeemErr := (ClientPairingHandshake{Now: fixedNow(now)}).Redeem(context.Background(), clientConn, ClientPairingRequest{
			ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
			PairingClaimOffer: issued.OfferPayload, Identity: client, ClientLabel: "Phone", ChannelBinding: binding,
		})
		if redeemErr != nil {
			t.Fatalf("Redeem: %v", redeemErr)
		}
		serverResult := <-serverDone
		if acceptErr := <-serverErr; acceptErr != nil || serverResult.Mode != ServerHandshakeModePairing {
			t.Fatalf("server pairing result=%+v err=%v", serverResult, acceptErr)
		}
		_ = clientConn.Close()
		_ = serverConn.Close()
		return result
	}
	first := redeem()
	second := redeem()
	if first.Grant == "" || first.Grant != second.Grant || first.DeliveryReceipt != second.DeliveryReceipt || first.SubjectKeyFingerprint != client.Fingerprint {
		t.Fatalf("pairing retry mismatch: first=%+v second=%+v", first, second)
	}
}

func TestClientPairingHandshakeCancelsBlockedPairingOpenSend(t *testing.T) {
	daemonSeed := bytes.Repeat([]byte{0x23}, ed25519.SeedSize)
	identity, err := NewIdentity("device-1", ed25519.NewKeyFromSeed(daemonSeed))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 11, 12, 34, 56, 789000000, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: fixedNow(now)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	issued := issueHandshakePairingClaim(t, store, FullDaemonScope(), time.Hour, 24*time.Hour, now)
	client, err := GenerateClientAccessIdentity("endpoint-1", bytes.NewReader(bytes.Repeat([]byte{0x51}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	binding := fixtureDTLSBinding(t, 0x11)
	connection := newBlockedAuthSendTransport(signedDeviceHelloFrame(t, identity, binding, now))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = (ClientPairingHandshake{Now: fixedNow(now)}).Redeem(ctx, connection, ClientPairingRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		PairingClaimOffer: issued.OfferPayload, Identity: client, ClientLabel: "Phone", ChannelBinding: binding,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked PairingOpen send error = %v", err)
	}
	select {
	case <-connection.sendStarted:
	default:
		t.Fatal("PairingOpen send did not reach transport backpressure")
	}
	select {
	case <-connection.Done():
	default:
		t.Fatal("pairing handshake did not close transport on context deadline")
	}
	select {
	case <-connection.sendFinished:
	case <-time.After(time.Second):
		t.Fatal("blocked PairingOpen send goroutine did not exit after transport close")
	}
}

func TestCapturedCapabilityOpenFailsAgainstNewChallenge(t *testing.T) {
	identity, credential, store, now := handshakeFixture(t, Scope{AllowDaemon: true})
	binding := fixtureDTLSBinding(t, 0x11)
	captured := completeManualCapabilityHandshake(t, ServerHandshake{
		Identity: identity, AccessStore: store, Now: fixedNow(now),
		Random: bytes.NewReader(bytes.Repeat([]byte{0x31}, authSessionBytes+authNonceBytes)),
	}, credential, binding)
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	defer serverConn.Close()
	serverDone := make(chan error, 1)
	go func() {
		_, err := (ServerHandshake{
			Identity: identity, AccessStore: store, Now: fixedNow(now),
			Random: bytes.NewReader(bytes.Repeat([]byte{0x42}, authSessionBytes+authNonceBytes)),
		}).Accept(context.Background(), serverConn, binding)
		serverDone <- err
	}()
	if _, err := receiveAuthEnvelope(context.Background(), clientConn); err != nil {
		t.Fatal(err)
	}
	if err := clientConn.Send(captured); err != nil {
		t.Fatal(err)
	}
	rejected, err := receiveAuthEnvelope(context.Background(), clientConn)
	if err != nil {
		t.Fatal(err)
	}
	if rejected.GetCapabilityRejected().GetCode() != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_REPLAYED {
		t.Fatalf("replay result = %+v", rejected)
	}
	if serverErr := <-serverDone; HandshakeCodeOf(serverErr) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_REPLAYED {
		t.Fatalf("server replay error = %v", serverErr)
	}
}

func TestAuthEnvelopeRejectsV1AnyTTYFrameAndUnknownFields(t *testing.T) {
	if _, err := UnmarshalAuthEnvelope([]byte{0, 0, 0, 0, 0, 0, 0}); HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL {
		t.Fatalf("anytty frame error = %v", err)
	}
	envelope := &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: 1, AuthSessionId: "fixture-auth-session-01",
		Payload: &remoteauthpb.AuthEnvelope_CapabilityRejected{CapabilityRejected: &remoteauthpb.CapabilityRejected{Code: remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL}},
	}
	if _, err := MarshalAuthEnvelope(envelope); HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL {
		t.Fatalf("v1 envelope error = %v", err)
	}
	envelope.Version = AuthVersion
	frame, err := MarshalAuthEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	frame = append(frame, 0x80, 0x06, 0x01)
	if _, err := UnmarshalAuthEnvelope(frame); HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL {
		t.Fatalf("unknown field error = %v", err)
	}
	valid, err := MarshalAuthEnvelope(&remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: "fixture-auth-session-01",
		Payload: &remoteauthpb.AuthEnvelope_CapabilityRejected{CapabilityRejected: &remoteauthpb.CapabilityRejected{Code: remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL}},
	})
	if err != nil {
		t.Fatal(err)
	}
	extra, err := proto.Marshal(&remoteauthpb.PairingAccepted{})
	if err != nil {
		t.Fatal(err)
	}
	duplicateOneof := append([]byte(nil), valid...)
	duplicateOneof = protowire.AppendTag(duplicateOneof, 9, protowire.BytesType)
	duplicateOneof = protowire.AppendBytes(duplicateOneof, extra)
	if _, err := UnmarshalAuthEnvelope(duplicateOneof); HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PROTOCOL {
		t.Fatalf("duplicate oneof payload error = %v", err)
	}
}

func TestServerHandshakeRequiresAccessStoreBeforeDeviceHello(t *testing.T) {
	seed := bytes.Repeat([]byte{0x55}, ed25519.SeedSize)
	identity, err := NewIdentity("device-no-store", ed25519.NewKeyFromSeed(seed))
	if err != nil {
		t.Fatal(err)
	}
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	defer serverConn.Close()
	_, err = (ServerHandshake{Identity: identity}).Accept(context.Background(), serverConn, fixtureDTLSBinding(t, 0x11))
	if HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL {
		t.Fatalf("missing AccessStore error = %v", err)
	}
}

func TestServerHandshakeValidatesCredentialAtOpenReceiptTime(t *testing.T) {
	t.Run("capability exact expiry", func(t *testing.T) {
		identity, credential, store, now := handshakeFixtureWithLifetime(t, Scope{AllowDaemon: true}, time.Hour, time.Second)
		binding := fixtureDTLSBinding(t, 0x11)
		clientConn, serverConn := memory.NewPair()
		defer clientConn.Close()
		defer serverConn.Close()
		calls := 0
		serverDone := make(chan error, 1)
		go func() {
			_, err := (ServerHandshake{Identity: identity, AccessStore: store, Now: func() time.Time {
				calls++
				if calls == 1 {
					return now
				}
				return now.Add(time.Second)
			}}).Accept(context.Background(), serverConn, binding)
			serverDone <- err
		}()
		_, err := (ClientHandshake{Now: fixedNow(now)}).Authenticate(context.Background(), clientConn, ClientHandshakeRequest{
			ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
			Credential: credential, ChannelBinding: binding,
		})
		if HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_EXPIRED {
			t.Fatalf("grant accepted after exact expiry: %v", err)
		}
		if serverErr := <-serverDone; HandshakeCodeOf(serverErr) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_EXPIRED {
			t.Fatalf("server expiry error = %v", serverErr)
		}
	})

	t.Run("pairing exact expiry", func(t *testing.T) {
		daemonSeed := bytes.Repeat([]byte{0x23}, ed25519.SeedSize)
		identity, _ := NewIdentity("device-1", ed25519.NewKeyFromSeed(daemonSeed))
		now := time.Date(2026, 7, 11, 12, 34, 56, 789000000, time.UTC)
		store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: fixedNow(now)})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = store.Close() })
		issued := issueHandshakePairingClaim(t, store, Scope{AllowDaemon: true}, time.Second, time.Hour, now)
		client, _ := GenerateClientAccessIdentity("endpoint-1", bytes.NewReader(bytes.Repeat([]byte{0x51}, 64)))
		binding := fixtureDTLSBinding(t, 0x11)
		clientConn, serverConn := memory.NewPair()
		defer clientConn.Close()
		defer serverConn.Close()
		calls := 0
		serverDone := make(chan error, 1)
		go func() {
			_, acceptErr := (ServerHandshake{Identity: identity, AccessStore: store, Now: func() time.Time {
				calls++
				if calls == 1 {
					return now
				}
				return now.Add(time.Second)
			}}).Accept(context.Background(), serverConn, binding)
			serverDone <- acceptErr
		}()
		_, err = (ClientPairingHandshake{Now: fixedNow(now)}).Redeem(context.Background(), clientConn, ClientPairingRequest{
			ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
			PairingClaimOffer: issued.OfferPayload, Identity: client, ClientLabel: "Phone", ChannelBinding: binding,
		})
		if HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED {
			t.Fatalf("ticket accepted after exact expiry: %v", err)
		}
		if serverErr := <-serverDone; HandshakeCodeOf(serverErr) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED {
			t.Fatalf("server ticket expiry error = %v", serverErr)
		}
	})
}

func TestClientPairingRejectsGrantExpiredBeforeResponseValidation(t *testing.T) {
	daemonSeed := bytes.Repeat([]byte{0x23}, ed25519.SeedSize)
	identity, _ := NewIdentity("device-1", ed25519.NewKeyFromSeed(daemonSeed))
	now := time.Date(2026, 7, 11, 12, 34, 56, 789000000, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: fixedNow(now)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	issued := issueHandshakePairingClaim(t, store, Scope{AllowDaemon: true}, time.Hour, time.Second, now)
	client, _ := GenerateClientAccessIdentity("endpoint-1", bytes.NewReader(bytes.Repeat([]byte{0x51}, 64)))
	binding := fixtureDTLSBinding(t, 0x11)
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	defer serverConn.Close()
	serverDone := make(chan error, 1)
	go func() {
		_, acceptErr := (ServerHandshake{Identity: identity, AccessStore: store, Now: fixedNow(now)}).Accept(context.Background(), serverConn, binding)
		serverDone <- acceptErr
	}()
	clockCalls := 0
	_, err = (ClientPairingHandshake{Now: func() time.Time {
		clockCalls++
		if clockCalls == 1 {
			return now
		}
		return now.Add(time.Second)
	}}).Redeem(context.Background(), clientConn, ClientPairingRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		PairingClaimOffer: issued.OfferPayload, Identity: client, ClientLabel: "Phone", ChannelBinding: binding,
	})
	if HandshakeCodeOf(err) != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_CAPABILITY_EXPIRED {
		t.Fatalf("expired PairingAccepted grant error = %v", err)
	}
	if serverErr := <-serverDone; serverErr != nil {
		t.Fatalf("daemon failed before returning short grant: %v", serverErr)
	}
}

func TestPairingHandshakeRedeemsClaimAndReturnsSignedBundle(t *testing.T) {
	identity, _, store, now := handshakeFixture(t, Scope{AllowDaemon: true})
	issued, err := store.IssuePairingClaim(PairingIssueOptions{
		Scope: Scope{TerminalID: "terminal-claim"}, TicketTTL: 10 * time.Minute, GrantLifetime: time.Hour, Now: now,
		Routes: []*remoteauthpb.EndpointRouteConfigV1{{SchemaVersion: 1, RouteId: "direct", Enabled: true, Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{SignalingAddresses: []string{"127.0.0.1:4040"}, IceTcpAddresses: []string{"127.0.0.1:4041"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := GenerateClientAccessIdentity("endpoint-claim", bytes.NewReader(bytes.Repeat([]byte{0x71}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	binding := fixtureDTLSBinding(t, 0x29)
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	defer serverConn.Close()
	serverDone := make(chan error, 1)
	go func() {
		_, acceptErr := (ServerHandshake{Identity: identity, AccessStore: store, Now: fixedNow(now)}).Accept(context.Background(), serverConn, binding)
		serverDone <- acceptErr
	}()
	result, err := (ClientPairingHandshake{Now: fixedNow(now)}).Redeem(context.Background(), clientConn, ClientPairingRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		PairingClaimOffer: issued.OfferPayload, Identity: client, ClientLabel: "Phone", ChannelBinding: binding,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	if result.TicketID != issued.Claims.TicketID || result.Scope.TerminalID != "terminal-claim" || !bytes.Equal(result.Bundle, issued.BundlePayload) {
		t.Fatalf("claim exchange result = %#v", result)
	}
}

func TestGrantFingerprintMismatchMapsToDeviceIdentityError(t *testing.T) {
	if code := HandshakeCodeOf(mapGrantError(ErrGrantFingerprintMismatch)); code != remoteauthpb.AuthErrorCode_AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH {
		t.Fatalf("fingerprint mismatch code = %s", code)
	}
}

func completeManualCapabilityHandshake(t *testing.T, server ServerHandshake, credential ClientAccessCredential, binding ChannelBinding) []byte {
	t.Helper()
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	defer serverConn.Close()
	serverDone := make(chan error, 1)
	go func() {
		_, err := server.Accept(context.Background(), serverConn, binding)
		serverDone <- err
	}()
	helloEnvelope, err := receiveAuthEnvelope(context.Background(), clientConn)
	if err != nil {
		t.Fatal(err)
	}
	clientNonce := bytes.Repeat([]byte{0x77}, authNonceBytes)
	proof, err := SignClientProof(credential.Identity, remoteauthpb.AuthOpenKind_AUTH_OPEN_KIND_CAPABILITY,
		[]byte(credential.CapabilityGrant), helloEnvelope.GetAuthSessionId(), helloEnvelope.GetDeviceHello().GetServerNonce(), clientNonce, binding)
	if err != nil {
		t.Fatal(err)
	}
	open := &remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: helloEnvelope.GetAuthSessionId(),
		Payload: &remoteauthpb.AuthEnvelope_CapabilityOpen{CapabilityOpen: &remoteauthpb.CapabilityOpen{
			Grant: credential.CapabilityGrant, ClientPublicKey: credential.Identity.PublicKey, ClientNonce: clientNonce, Proof: proof,
		}},
	}
	frame, err := MarshalAuthEnvelope(open)
	if err != nil {
		t.Fatal(err)
	}
	if err := clientConn.Send(frame); err != nil {
		t.Fatal(err)
	}
	accepted, err := receiveAuthEnvelope(context.Background(), clientConn)
	if err != nil || accepted.GetCapabilityAccepted() == nil {
		t.Fatalf("accepted=%+v err=%v", accepted, err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), frame...)
}

func handshakeFixture(t *testing.T, scope Scope) (Identity, ClientAccessCredential, *AccessStore, time.Time) {
	return handshakeFixtureWithLifetime(t, scope, time.Hour, time.Hour)
}

func handshakeFixtureWithLifetime(t *testing.T, scope Scope, ticketTTL time.Duration, grantLifetime time.Duration) (Identity, ClientAccessCredential, *AccessStore, time.Time) {
	t.Helper()
	daemonSeed := bytes.Repeat([]byte{0x23}, ed25519.SeedSize)
	identity, err := NewIdentity("device-1", ed25519.NewKeyFromSeed(daemonSeed))
	if err != nil {
		t.Fatal(err)
	}
	client, err := GenerateClientAccessIdentity("endpoint-1", bytes.NewReader(bytes.Repeat([]byte{0x34}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 11, 12, 34, 56, 789000000, time.UTC)
	store, err := LoadAccessStore(t.TempDir(), identity, AccessStoreOptions{Now: fixedNow(now)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, _, err := store.IssuePairingBundle(PairingIssueOptions{Scope: scope, TicketTTL: ticketTTL, GrantLifetime: grantLifetime, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.RedeemPairingBundle(payload, client.PublicKey, "fixture-client", now)
	if err != nil {
		t.Fatal(err)
	}
	return identity, ClientAccessCredential{Version: 3, EndpointID: client.EndpointID, Identity: client, CapabilityGrant: result.Grant, UpdatedAt: now}, store, now
}

func fixtureDTLSBinding(t *testing.T, octet byte) ChannelBinding {
	t.Helper()
	binding, err := DTLSChannelBinding(fixtureDTLSFingerprint(octet))
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func fixtureDTLSFingerprint(octet byte) string {
	parts := make([]byte, 0, len("sha-256:")+32*3)
	parts = append(parts, "sha-256:"...)
	for index := 0; index < 32; index++ {
		if index > 0 {
			parts = append(parts, ':')
		}
		parts = append(parts, hex.EncodeToString([]byte{octet})...)
	}
	return string(parts)
}

func fixedNow(now time.Time) func() time.Time { return func() time.Time { return now } }

func issueHandshakePairingClaim(t *testing.T, store *AccessStore, scope Scope, ticketTTL, grantLifetime time.Duration, now time.Time) PairingClaimIssueResult {
	t.Helper()
	issued, err := store.IssuePairingClaim(PairingIssueOptions{
		Scope: scope, TicketTTL: ticketTTL, GrantLifetime: grantLifetime, Now: now,
		Routes: []*remoteauthpb.EndpointRouteConfigV1{{
			SchemaVersion: 1, RouteId: "direct", Enabled: true,
			Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{
				SignalingAddresses: []string{"127.0.0.1:4040"}, IceTcpAddresses: []string{"127.0.0.1:4041"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return issued
}

func signedDeviceHelloFrame(t *testing.T, identity Identity, binding ChannelBinding, now time.Time) []byte {
	t.Helper()
	authSessionID := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x61}, authSessionBytes))
	hello := &remoteauthpb.DeviceHello{
		DeviceId: identity.DeviceID, DevicePublicKey: append([]byte(nil), identity.PublicKey...),
		DeviceFingerprint: identity.Fingerprint, ServerNonce: bytes.Repeat([]byte{0x62}, authNonceBytes),
		ChannelBinding: channelBindingToProto(binding), IssuedAtUnixNano: now.UnixNano(),
	}
	signingBytes, err := DeviceHelloSigningBytes(authSessionID, hello)
	if err != nil {
		t.Fatal(err)
	}
	hello.Signature = ed25519.Sign(identity.PrivateKey, signingBytes)
	frame, err := MarshalAuthEnvelope(&remoteauthpb.AuthEnvelope{
		Protocol: AuthProtocol, Version: AuthVersion, AuthSessionId: authSessionID,
		Payload: &remoteauthpb.AuthEnvelope_DeviceHello{DeviceHello: hello},
	})
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

type blockedAuthSendTransport struct {
	recvFrame    []byte
	recvOnce     sync.Once
	sendOnce     sync.Once
	closeOnce    sync.Once
	sendStarted  chan struct{}
	sendFinished chan struct{}
	done         chan struct{}
}

func newBlockedAuthSendTransport(recvFrame []byte) *blockedAuthSendTransport {
	return &blockedAuthSendTransport{
		recvFrame: append([]byte(nil), recvFrame...), sendStarted: make(chan struct{}),
		sendFinished: make(chan struct{}), done: make(chan struct{}),
	}
}

func (transport *blockedAuthSendTransport) Send([]byte) error {
	transport.sendOnce.Do(func() { close(transport.sendStarted) })
	<-transport.done
	close(transport.sendFinished)
	return io.EOF
}

func (transport *blockedAuthSendTransport) Recv() ([]byte, error) {
	var frame []byte
	transport.recvOnce.Do(func() { frame = append([]byte(nil), transport.recvFrame...) })
	if frame != nil {
		return frame, nil
	}
	<-transport.done
	return nil, io.EOF
}

func (transport *blockedAuthSendTransport) Close() error {
	transport.closeOnce.Do(func() { close(transport.done) })
	return nil
}

func (transport *blockedAuthSendTransport) Done() <-chan struct{} { return transport.done }

var _ transport.Transport = (*memory.Transport)(nil)
var _ transport.Transport = (*blockedAuthSendTransport)(nil)
var _ = errors.Is
