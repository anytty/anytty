package daemon

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	core "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestSessionAcceptorAuthenticatesBeforeServingScopedTransport(t *testing.T) {
	identity, credential, store, now := sessionFixture(t, remoteauth.Scope{TerminalID: "term-1"})
	claims, err := remoteauth.Verify(credential.CapabilityGrant, identity.Fingerprint, now, store)
	if err != nil {
		t.Fatal(err)
	}
	core := &recordingCore{}
	clientConn, serverConn := memory.NewPair()
	countedServerConn := &countingSessionTransport{Transport: serverConn}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- (SessionAcceptor{Core: core, Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).
			ServeDataChannel(context.Background(), countedServerConn, sessionDTLSFingerprint())
	}()
	if _, err := (remoteauth.ClientHandshake{Now: fixedSessionNow(now)}).Authenticate(context.Background(), clientConn, remoteauth.ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: sessionDTLSBinding(t),
	}); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("ServeDataChannel: %v", err)
	}
	if core.calls != 1 || core.scope.TerminalID != "term-1" || core.scope.AllowDaemon ||
		core.scope.GrantID != claims.GrantID || !core.scope.GrantExpiresAt.Equal(claims.ExpiresAt) {
		t.Fatalf("unexpected scoped core call: calls=%d scope=%#v", core.calls, core.scope)
	}
	if countedServerConn.closes.Load() != 1 {
		t.Fatalf("handed-off transport closes = %d, want exactly 1", countedServerConn.closes.Load())
	}
}

func TestSessionAcceptorRejectsRevokedGrantBeforeCore(t *testing.T) {
	identity, credential, store, now := sessionFixture(t, remoteauth.Scope{AllowDaemon: true})
	claims, err := remoteauth.Verify(credential.CapabilityGrant, identity.Fingerprint, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RevokeGrant(claims.GrantID); err != nil {
		t.Fatal(err)
	}
	core := &recordingCore{}
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- (SessionAcceptor{Core: core, Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).
			ServeDataChannel(context.Background(), serverConn, sessionDTLSFingerprint())
	}()
	_, clientErr := (remoteauth.ClientHandshake{Now: fixedSessionNow(now)}).Authenticate(context.Background(), clientConn, remoteauth.ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: sessionDTLSBinding(t),
	})
	if clientErr == nil || !strings.Contains(clientErr.Error(), "REVOKED") {
		t.Fatalf("expected revoked capability rejection, got %v", clientErr)
	}
	if serverErr := <-serverDone; serverErr == nil || !strings.Contains(serverErr.Error(), "revoked") {
		t.Fatalf("expected daemon revoke error, got %v", serverErr)
	}
	if core.calls != 0 {
		t.Fatalf("core must not see unauthorized transport, calls=%d", core.calls)
	}
}

func TestSessionAcceptorRejectsSignedGrantAbsentFromAccessStore(t *testing.T) {
	identity, credential, store, now := sessionFixture(t, remoteauth.Scope{AllowDaemon: true})
	unknown, err := remoteauth.Issue(identity.PrivateKey, remoteauth.Claims{
		GrantID: "grant-not-registered", IssuerDeviceID: identity.DeviceID,
		SubjectKeyFingerprint: credential.Identity.Fingerprint, Scope: remoteauth.Scope{AllowDaemon: true},
		IssuedAt: now.Add(-time.Minute), NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
		RevocationID: "grant-not-registered", Nonce: "unregistered-grant-nonce",
	})
	if err != nil {
		t.Fatal(err)
	}
	credential.CapabilityGrant = unknown
	recorded := &recordingCore{}
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- (SessionAcceptor{Core: recorded, Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).
			ServeDataChannel(context.Background(), serverConn, sessionDTLSFingerprint())
	}()
	_, clientErr := (remoteauth.ClientHandshake{Now: fixedSessionNow(now)}).Authenticate(context.Background(), clientConn, remoteauth.ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: sessionDTLSBinding(t),
	})
	if clientErr == nil || !strings.Contains(clientErr.Error(), "REVOKED") {
		t.Fatalf("unregistered signed grant client error = %v", clientErr)
	}
	if serverErr := <-serverDone; serverErr == nil || !strings.Contains(serverErr.Error(), "revoked") {
		t.Fatalf("unregistered signed grant daemon error = %v", serverErr)
	}
	if recorded.calls != 0 {
		t.Fatalf("unregistered signed grant reached core %d times", recorded.calls)
	}
}

func TestSessionAcceptorMapsExplicitFilePermissions(t *testing.T) {
	identity, credential, store, now := sessionFixture(t, remoteauth.FullDaemonScope())
	recorded := &recordingCore{}
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- (SessionAcceptor{Core: recorded, Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).ServeDataChannel(context.Background(), serverConn, sessionDTLSFingerprint())
	}()
	if _, err := (remoteauth.ClientHandshake{Now: fixedSessionNow(now)}).Authenticate(context.Background(), clientConn, remoteauth.ClientHandshakeRequest{ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint, Credential: credential, ChannelBinding: sessionDTLSBinding(t)}); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	if !recorded.scope.FileReadMetadata || !recorded.scope.FileReadContent || !recorded.scope.FileWriteContent || !recorded.scope.FileMutate {
		t.Fatalf("file permissions lost: %#v", recorded.scope)
	}
}

func TestSessionAcceptorPairingModeNeverStartsCoreProtocol(t *testing.T) {
	identity, _, store, now := sessionFixture(t, remoteauth.FullDaemonScope())
	issued, err := store.IssuePairingClaim(remoteauth.PairingIssueOptions{
		Scope: remoteauth.FullDaemonScope(), TicketTTL: time.Hour, GrantLifetime: 24 * time.Hour,
		Routes: []*remoteauthpb.EndpointRouteConfigV1{{SchemaVersion: 1, RouteId: "direct", Enabled: true, Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{SignalingAddresses: []string{"127.0.0.1:4040"}, IceTcpAddresses: []string{"127.0.0.1:4041"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("endpoint-1", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recorded := &recordingCore{}
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- (SessionAcceptor{Core: recorded, Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).
			ServeDataChannel(context.Background(), serverConn, sessionDTLSFingerprint())
	}()
	result, err := (remoteauth.ClientPairingHandshake{Now: fixedSessionNow(now)}).Redeem(context.Background(), clientConn, remoteauth.ClientPairingRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		PairingClaimOffer: issued.OfferPayload, Identity: clientIdentity, ChannelBinding: sessionDTLSBinding(t),
	})
	if err != nil || result.Grant == "" {
		t.Fatalf("pairing result=%#v err=%v", result, err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	if recorded.calls != 0 {
		t.Fatalf("pairing transport reached core %d times", recorded.calls)
	}
}

func TestPairingAcceptorRejectsCapabilityMode(t *testing.T) {
	identity, credential, store, now := sessionFixture(t, remoteauth.FullDaemonScope())
	binding, err := remoteauth.LocalUnixChannelBinding("/tmp/anytty-session-pairing.sock")
	if err != nil {
		t.Fatal(err)
	}
	clientConn, serverConn := memory.NewPair()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- (PairingAcceptor{Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).
			ServeBoundTransport(context.Background(), serverConn, binding)
	}()
	_, clientErr := (remoteauth.ClientHandshake{Now: fixedSessionNow(now)}).Authenticate(context.Background(), clientConn, remoteauth.ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, ChannelBinding: binding,
	})
	if clientErr == nil || !strings.Contains(clientErr.Error(), "PROTOCOL") {
		t.Fatalf("expected pairing-only capability rejection, got %v", clientErr)
	}
	if serverErr := <-serverDone; serverErr == nil || !strings.Contains(serverErr.Error(), "pairing-only") {
		t.Fatalf("expected pairing-only daemon error, got %v", serverErr)
	}
}

func TestSessionAcceptorRejectsLocalUnixBindingBeforeCore(t *testing.T) {
	identity, _, store, now := sessionFixture(t, remoteauth.FullDaemonScope())
	binding, err := remoteauth.LocalUnixChannelBinding("/tmp/anytty-generic-session.sock")
	if err != nil {
		t.Fatal(err)
	}
	clientConn, serverConn := memory.NewPair()
	defer clientConn.Close()
	recorded := &recordingCore{}
	err = (SessionAcceptor{Core: recorded, Identity: identity, AccessStore: store, Now: fixedSessionNow(now)}).
		ServeBoundTransport(context.Background(), serverConn, binding)
	if err == nil || !strings.Contains(err.Error(), "rejects local Unix") {
		t.Fatalf("generic local Unix ingress error = %v", err)
	}
	if recorded.calls != 0 {
		t.Fatalf("local Unix session reached core %d times", recorded.calls)
	}
}

type recordingCore struct {
	calls int
	scope core.TransportScope
}

type countingSessionTransport struct {
	transport.Transport
	closes atomic.Int32
}

func (connection *countingSessionTransport) Close() error {
	connection.closes.Add(1)
	return connection.Transport.Close()
}

func (core *recordingCore) ServeScopedTransport(_ context.Context, conn transport.Transport, scope core.TransportScope) error {
	core.calls++
	core.scope = scope
	return conn.Close()
}

func sessionFixture(t *testing.T, scope remoteauth.Scope) (remoteauth.Identity, remoteauth.ClientAccessCredential, *remoteauth.AccessStore, time.Time) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	identity, err := remoteauth.NewIdentity("device-1", privateKey)
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	if !bytes.Equal(identity.PublicKey, publicKey) {
		t.Fatal("identity public key mismatch")
	}
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("endpoint-1", rand.Reader)
	if err != nil {
		t.Fatalf("GenerateClientAccessIdentity: %v", err)
	}
	store, err := remoteauth.LoadAccessStore(t.TempDir(), identity, remoteauth.AccessStoreOptions{Now: fixedSessionNow(now)})
	if err != nil {
		t.Fatalf("LoadAccessStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, _, err := store.IssuePairingBundle(remoteauth.PairingIssueOptions{Scope: scope, TicketTTL: time.Hour, GrantLifetime: time.Hour, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := remoteauth.EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.RedeemPairingBundle(payload, clientIdentity.PublicKey, "session-fixture", now)
	if err != nil {
		t.Fatal(err)
	}
	return identity, remoteauth.ClientAccessCredential{
		Version: 3, EndpointID: clientIdentity.EndpointID, Identity: clientIdentity,
		CapabilityGrant: result.Grant, UpdatedAt: now,
	}, store, now
}

func sessionDTLSBinding(t *testing.T) remoteauth.ChannelBinding {
	t.Helper()
	binding, err := remoteauth.DTLSChannelBinding(sessionDTLSFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func sessionDTLSFingerprint() string {
	return "sha-256:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11:11"
}

func fixedSessionNow(now time.Time) func() time.Time {
	return func() time.Time { return now }
}
