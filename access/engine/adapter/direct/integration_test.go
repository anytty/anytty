package direct_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	accesscontract "github.com/anytty/anytty/access/contract"
	"github.com/anytty/anytty/access/engine/adapter/direct"
	"github.com/anytty/anytty/access/engine/adapter/internal/e2etest"
	peeradapter "github.com/anytty/anytty/access/engine/adapter/peer"
	pionadapter "github.com/anytty/anytty/access/engine/adapter/webrtc/pion"
	"github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	"github.com/anytty/anytty/access/files"
	poolprovider "github.com/anytty/anytty/access/provider/pool"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	remote "github.com/anytty/anytty/access/remote"
	accessserver "github.com/anytty/anytty/access/server"
	remotev2webrtc "github.com/anytty/anytty/access/transport/webrtc"
	core "github.com/anytty/anytty/pool/core"
	providercore "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestDirectICETCPCompletesSignedSignalingAuthHelloAndProtoAPI(t *testing.T) {
	fixture := newDirectFixture(t)
	var clientPeer *pionwebrtc.PeerConnection
	clientAPI := directClientAPI()
	dialer := fixture.dialer(pionadapter.Factory{PeerConnections: func(configuration pionwebrtc.Configuration) (*pionwebrtc.PeerConnection, error) {
		peer, err := clientAPI.NewPeerConnection(configuration)
		if err == nil {
			clientPeer = peer
		}
		return peer, err
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ready, err := dialer.Connect(ctx, fixture.attempt(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	defer ready.Close()
	result, err := ready.(clientruntime.ApplicationReadyPeerSession).ExecuteApplication(ctx, &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	})
	if err != nil || result.GetTerminalList() == nil {
		t.Fatalf("Direct Proto API result=%#v err=%v", result, err)
	}
	if ready.Readiness().Identity.DeviceFingerprint != fixture.identity.Fingerprint || !ready.Readiness().AuthorizationVerified {
		t.Fatalf("Direct readiness=%+v", ready.Readiness())
	}
	pair := selectedPair(t, clientPeer)
	if pair.Local.Protocol != pionwebrtc.ICEProtocolTCP || pair.Remote.Protocol != pionwebrtc.ICEProtocolTCP {
		t.Fatalf("Direct selected candidate pair is not TCP: %s", pair)
	}
}

func TestDirectICETCPCompletesOverOneSharedTCPPort(t *testing.T) {
	fixture := newDirectFixtureWithSharedPort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ready, err := fixture.dialer(pionadapter.Factory{PeerConnections: directClientAPI().NewPeerConnection}).Connect(ctx, fixture.attempt(t, 101))
	if err != nil {
		t.Fatal(err)
	}
	defer ready.Close()
	result, err := ready.(clientruntime.ApplicationReadyPeerSession).ExecuteApplication(ctx, &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	})
	if err != nil || result.GetTerminalList() == nil {
		t.Fatalf("shared-port Direct result=%#v err=%v", result, err)
	}
	if fixture.signalingAddress != fixture.iceAddress {
		t.Fatalf("shared-port fixture used signaling=%q ice=%q", fixture.signalingAddress, fixture.iceAddress)
	}
}

// TestDirectICETCPFullSessionTerminalAndFileRoundTrip 覆盖 Direct 完整 session：
// terminal create/list/attach/input/PTY output 之后，在同一 ready session 上做
// upload→download 文件往返，校验字节一致与 ack/window 语义。
func TestDirectICETCPFullSessionTerminalAndFileRoundTrip(t *testing.T) {
	fixture := newDirectFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ready, err := fixture.dialer(pionadapter.Factory{PeerConnections: directClientAPI().NewPeerConnection}).Connect(ctx, fixture.attempt(t, 7))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ready.Close() }()
	session, ok := ready.(*direct.Session)
	if !ok {
		t.Fatalf("Direct ready session type = %T", ready)
	}

	created, err := session.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: "direct-e2e", Name: "direct-e2e", Command: []string{"/bin/sh"}, Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}})
	if err != nil {
		t.Fatalf("Direct terminal create: %v", err)
	}
	ref := created.GetTerminal().GetRef()
	if ref.GetTerminalId() != "direct-e2e" {
		t.Fatalf("Direct terminal create ref = %#v", ref)
	}
	list, err := session.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatalf("Direct terminal list: %v", err)
	}
	found := false
	for _, terminal := range list.GetTerminals() {
		if terminal.GetRef().GetTerminalId() == "direct-e2e" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Direct terminal list = %#v", list.GetTerminals())
	}

	attached, err := session.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal: ref, Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER, SurfaceId: "direct-e2e", ViewId: "e2e",
	})
	if err != nil {
		t.Fatalf("Direct terminal attach: %v", err)
	}
	attachment := attached.GetAttachment()
	if attachment.GetResource().GetKind() != apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		t.Fatalf("Direct attachment resource = %#v", attachment.GetResource())
	}
	stream, err := session.OpenResourceStream(attachment.GetResource())
	if err != nil {
		t.Fatalf("Direct open attachment stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	// The echoed command line contains "DIRECT-E2E-$(echo FULL)"; only execution
	// output can contain the joined marker.
	if err := session.TerminalInput(ctx, &apipb.TerminalInputCommand{
		Attachment: attachment.GetResource(), Data: []byte("echo DIRECT-E2E-$(echo FULL)\r"),
	}); err != nil {
		t.Fatalf("Direct terminal input: %v", err)
	}
	if output := waitForDirectPTYOutput(t, ctx, stream, "DIRECT-E2E-FULL"); output == "" {
		t.Fatal("Direct PTY output did not reach the client")
	}

	// Slightly larger than the 1 MiB download window so the transfer must
	// exercise the window/ack flow, with a tail that is not chunk-aligned.
	content := make([]byte, 1<<20+12345)
	for index := range content {
		content[index] = byte(index * 31)
	}
	remotePath := filepath.Join(t.TempDir(), "direct-session.bin")
	e2etest.UploadFile(t, ctx, session, remotePath, content)
	if stored, err := os.ReadFile(remotePath); err != nil || !bytes.Equal(stored, content) {
		t.Fatalf("Direct uploaded file mismatch: bytes=%d err=%v", len(stored), err)
	}
	e2etest.DownloadFile(t, ctx, session, remotePath, content)

	if err := session.TerminalDetach(ctx, &apipb.TerminalDetachCommand{Attachment: attachment.GetResource()}); err != nil {
		t.Fatalf("Direct terminal detach: %v", err)
	}
	if err := session.TerminalKill(ctx, &apipb.TerminalKillCommand{Terminal: ref}); err != nil {
		t.Fatalf("Direct terminal kill: %v", err)
	}
	if err := session.TerminalRemove(ctx, &apipb.TerminalRemoveCommand{Terminal: ref}); err != nil {
		t.Fatalf("Direct terminal remove: %v", err)
	}
}

func TestDirectICETCPRedeemsShortClaimAndReturnsSignedBundle(t *testing.T) {
	fixture := newDirectFixture(t)
	issued, err := fixture.store.IssuePairingClaim(remoteauth.PairingIssueOptions{
		Scope: remoteauth.Scope{AllowDaemon: true}, TicketTTL: 10 * time.Minute, GrantLifetime: time.Hour, Now: fixture.now,
		Routes: []*remoteauthpb.EndpointRouteConfigV1{{SchemaVersion: 1, RouteId: "direct", Enabled: true, Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{SignalingAddresses: []string{fixture.signalingAddress}, IceTcpAddresses: []string{fixture.iceAddress}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("direct", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := remoteauth.NewPrivateClientAccessSigner(clientIdentity)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := (&direct.PairingConnector{Peers: pionadapter.Factory{PeerConnections: directClientAPI().NewPeerConnection}, Now: func() time.Time { return fixture.now }}).Redeem(ctx, fixture.attempt(t, 201), remoteauth.ClientPairingRequest{
		ExpectedDeviceID: fixture.identity.DeviceID, ExpectedDeviceFingerprint: fixture.identity.Fingerprint,
		PairingClaimOffer: issued.OfferPayload, Identity: clientIdentity, Signer: signer, ClientLabel: "direct-claim-e2e",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TicketID != issued.Claims.TicketID || !bytes.Equal(result.Bundle, issued.BundlePayload) {
		t.Fatalf("Direct claim result=%#v", result)
	}
}

func TestDirectTCPMappingChangesLocatorsWithoutChangingIdentity(t *testing.T) {
	fixture := newDirectFixture(t)
	signalingProxy := newTCPMappingProxy(t, fixture.signalingAddress)
	iceProxy := newTCPMappingProxy(t, fixture.iceAddress)
	var clientPeer *pionwebrtc.PeerConnection
	clientAPI := directClientAPI()
	dialer := fixture.dialer(pionadapter.Factory{PeerConnections: func(configuration pionwebrtc.Configuration) (*pionwebrtc.PeerConnection, error) {
		peer, err := clientAPI.NewPeerConnection(configuration)
		if err == nil {
			clientPeer = peer
		}
		return peer, err
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ready, err := dialer.Connect(ctx, fixture.attemptWithLocators(t, 2, signalingProxy.address(), iceProxy.address()))
	if err != nil {
		t.Fatal(err)
	}
	defer ready.Close()
	if ready.Readiness().Identity.DeviceFingerprint != fixture.identity.Fingerprint || !ready.Readiness().IdentityVerified {
		t.Fatalf("mapped Direct readiness=%+v", ready.Readiness())
	}
	pair := selectedPair(t, clientPeer)
	_, mappedPort, err := net.SplitHostPort(iceProxy.address())
	if err != nil {
		t.Fatal(err)
	}
	if pair.Remote.Protocol != pionwebrtc.ICEProtocolTCP || strconv.Itoa(int(pair.Remote.Port)) != mappedPort {
		t.Fatalf("mapped selected pair=%s, want remote TCP port %s", pair, mappedPort)
	}
}

func TestDirectTCPMappingFailsClosedForUnreachableSignalingOrICE(t *testing.T) {
	fixture := newDirectFixture(t)
	unreachableSignaling := closedTCPAddress(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := fixture.dialer(pionadapter.Factory{}).Connect(ctx, fixture.attemptWithLocators(t, 3, unreachableSignaling, fixture.iceAddress)); err == nil {
		t.Fatal("unreachable mapped signaling unexpectedly connected")
	}
	signalingProxy := newTCPMappingProxy(t, fixture.signalingAddress)
	unreachableICE := closedTCPAddress(t)
	ctxICE, cancelICE := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelICE()
	if _, err := fixture.dialer(pionadapter.Factory{}).Connect(ctxICE, fixture.attemptWithLocators(t, 4, signalingProxy.address(), unreachableICE)); err == nil {
		t.Fatal("unreachable mapped ICE-TCP address unexpectedly connected")
	}
}

func TestDirectSignalingRejectsExpiryReplayAndPinMismatch(t *testing.T) {
	fixture := newDirectFixture(t)
	clientAPI := directClientAPI()
	peer, err := (pionadapter.Factory{PeerConnections: clientAPI.NewPeerConnection}).OpenDirectPeer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	offer, err := peer.CreateOffer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	client := direct.TCPSignalingClient{}
	claims, err := remoteauth.Verify(fixture.credential.CapabilityGrant, fixture.identity.Fingerprint, fixture.now, nil)
	if err != nil {
		t.Fatal(err)
	}
	base := &remoteauthpb.DirectSignalingRequestV2{
		SchemaVersion: remoteauth.DirectSignalingSchemaVersion, RequestId: "request-valid-0000000000000001",
		ExpectedDeviceId: fixture.identity.DeviceID, ExpectedDeviceFingerprint: fixture.identity.Fingerprint,
		OfferSdp: offer, IssuedAtUnixNano: fixture.now.UnixNano(), ExpiresAtUnixNano: fixture.now.Add(remoteauth.DirectSignalingMaxTTL).UnixNano(),
		GrantId: claims.GrantID, GrantExpiresAtUnixNano: claims.ExpiresAt.UnixNano(),
	}
	answer, err := client.Exchange(context.Background(), []string{fixture.signalingAddress}, base)
	if err != nil {
		t.Fatal(err)
	}
	if err := remoteauth.VerifyDirectSignalingAnswer(answer, base.GetRequestId(), fixture.identity.DeviceID, fixture.identity.Fingerprint, fixture.now); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Exchange(context.Background(), []string{fixture.signalingAddress}, base); signalingCode(err) != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_REPLAYED {
		t.Fatalf("replay error=%v code=%v", err, signalingCode(err))
	}
	expired := cloneRequest(base)
	expired.RequestId = "request-expired-00000000000001"
	expired.IssuedAtUnixNano = fixture.now.Add(-time.Minute).UnixNano()
	expired.ExpiresAtUnixNano = fixture.now.Add(-time.Second).UnixNano()
	if _, err := client.Exchange(context.Background(), []string{fixture.signalingAddress}, expired); signalingCode(err) != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_EXPIRED {
		t.Fatalf("expiry error=%v code=%v", err, signalingCode(err))
	}
	mismatch := cloneRequest(base)
	mismatch.RequestId = "request-mismatch-0000000000001"
	mismatch.ExpectedDeviceFingerprint = "sha256:wrong-pin"
	if _, err := client.Exchange(context.Background(), []string{fixture.signalingAddress}, mismatch); signalingCode(err) != remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_IDENTITY_MISMATCH {
		t.Fatalf("pin mismatch error=%v code=%v", err, signalingCode(err))
	}
}

func TestDirectICETCPHundredConnectionsAndListenerCleanup(t *testing.T) {
	fixture := newDirectFixture(t)
	clientAPI := directClientAPI()
	dialer := fixture.dialer(pionadapter.Factory{PeerConnections: clientAPI.NewPeerConnection})
	for index := 0; index < 100; index++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ready, err := dialer.Connect(ctx, fixture.attempt(t, clientruntime.SessionGeneration(index+1)))
		if err != nil {
			cancel()
			t.Fatalf("Direct connection %d: %v", index, err)
		}
		if err := ready.Close(); err != nil {
			cancel()
			t.Fatalf("close Direct connection %d: %v", index, err)
		}
		cancel()
		deadline := time.Now().Add(time.Second)
		for fixture.closedSessions.Load() < int32(index+1) && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if fixture.closedSessions.Load() < int32(index+1) {
			t.Fatalf("daemon session %d did not observe protocol close", index)
		}
	}
	fixture.stop(t)
	for _, address := range []string{fixture.signalingAddress, fixture.iceAddress} {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			t.Fatalf("listener %s remained reachable after Direct server shutdown", address)
		}
	}
}

func TestDirectCancelDuringSignalingClosesPeer(t *testing.T) {
	fixture := newDirectFixture(t)
	clientAPI := directClientAPI()
	var mu sync.Mutex
	var clientPeer *pionwebrtc.PeerConnection
	started := make(chan struct{})
	dialer := fixture.dialer(pionadapter.Factory{PeerConnections: func(configuration pionwebrtc.Configuration) (*pionwebrtc.PeerConnection, error) {
		peer, err := clientAPI.NewPeerConnection(configuration)
		mu.Lock()
		clientPeer = peer
		mu.Unlock()
		return peer, err
	}})
	dialer.Signaling = blockingSignaling{started: started}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := dialer.Connect(ctx, fixture.attempt(t, 101))
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel Direct signaling error=%v", err)
	}
	mu.Lock()
	peer := clientPeer
	mu.Unlock()
	deadline := time.Now().Add(time.Second)
	for peer != nil && peer.ConnectionState() != pionwebrtc.PeerConnectionStateClosed && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if peer == nil || peer.ConnectionState() != pionwebrtc.PeerConnectionStateClosed {
		t.Fatalf("cancelled Direct peer=%v state=%v", peer, peer.ConnectionState())
	}
}

func TestNilDirectDialerFailsWithoutPanic(t *testing.T) {
	fixture := newDirectFixture(t)
	var dialer *direct.Dialer
	if _, err := dialer.Connect(context.Background(), fixture.attempt(t, 102)); err == nil {
		t.Fatal("nil Direct dialer unexpectedly connected")
	}
}

func TestVerifiedTCPAnswerProjectionPublishesOneCandidatePerLocator(t *testing.T) {
	answer, err := direct.ProjectVerifiedTCPAnswer(&remoteauthpb.DirectSignalingAnswerV2{
		AnswerSdp: "v=0\r\na=candidate:one 1 tcp 10 192.168.1.10 41121 typ host tcptype passive\r\na=candidate:two 1 tcp 9 10.0.0.10 41121 typ host tcptype passive\r\na=end-of-candidates\r\n",
		Candidates: []*remoteauthpb.DirectIceCandidate{
			{Candidate: "candidate:one 1 tcp 10 192.168.1.10 41121 typ host tcptype passive"},
			{Candidate: "candidate:two 1 tcp 9 10.0.0.10 41121 typ host tcptype passive"},
		},
	}, []string{"127.0.0.1:52121"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(answer.GetAnswerSdp(), "a=candidate:") != 1 || len(answer.GetCandidates()) != 1 {
		t.Fatalf("projected answer retained duplicate mux candidates: %#v", answer)
	}
	if !strings.Contains(answer.GetAnswerSdp(), "127.0.0.1 52121") || !strings.Contains(answer.GetCandidates()[0].GetCandidate(), "127.0.0.1 52121") {
		t.Fatalf("projected answer missed mapped locator: %#v", answer)
	}
}

type directFixture struct {
	identity         remoteauth.Identity
	credential       remoteauth.ClientAccessCredential
	store            *remoteauth.AccessStore
	now              time.Time
	signalingAddress string
	iceAddress       string
	server           *remotev2webrtc.DirectServer
	cancel           context.CancelFunc
	done             chan error
	stopOnce         sync.Once
	closedSessions   *atomic.Int32
}

func newDirectFixture(t *testing.T) *directFixture {
	return newDirectFixtureWithPortMode(t, false)
}

func newDirectFixtureWithSharedPort(t *testing.T) *directFixture {
	return newDirectFixtureWithPortMode(t, true)
}

func newDirectFixtureWithPortMode(t *testing.T, sharedPort bool) *directFixture {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.NewIdentity("device-direct", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("direct", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store, err := remoteauth.LoadAccessStore(t.TempDir(), identity, remoteauth.AccessStoreOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	bundle, _, err := store.IssuePairingBundle(remoteauth.PairingIssueOptions{
		Scope: remoteauth.Scope{AllowDaemon: true}, TicketTTL: time.Hour, GrantLifetime: time.Hour, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := remoteauth.EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := store.RedeemPairingBundle(payload, clientIdentity.PublicKey, "direct-e2e", now)
	if err != nil {
		t.Fatal(err)
	}
	rootListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	signalingListener := rootListener
	var iceListener net.Listener
	if sharedPort {
		signalingListener, iceListener, err = remotev2webrtc.SplitDirectListener(rootListener)
		if err != nil {
			_ = rootListener.Close()
			t.Fatal(err)
		}
	} else {
		iceListener, err = net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			_ = signalingListener.Close()
			t.Fatal(err)
		}
	}
	accessCore, stopAccessCore := startAccessCoreForRemoteTest(t, directCoreAccessService{store: store})
	t.Cleanup(stopAccessCore)
	acceptor := remote.SessionAcceptor{
		Core: accessCore, Identity: identity, AccessStore: store, Now: func() time.Time { return now },
	}
	closedSessions := &atomic.Int32{}
	directServer, err := remotev2webrtc.NewDirectServer(identity, countingHandler{inner: acceptor, closed: closedSessions}, signalingListener, iceListener, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- directServer.Serve(ctx) }()
	fixture := &directFixture{
		identity: identity, credential: remoteauth.ClientAccessCredential{
			Version: 3, EndpointID: "direct", Identity: clientIdentity, CapabilityGrant: exchanged.Grant, UpdatedAt: now,
		},
		store: store, now: now, signalingAddress: signalingListener.Addr().String(), iceAddress: iceListener.Addr().String(), server: directServer, cancel: cancel, done: done,
	}
	fixture.closedSessions = closedSessions
	t.Cleanup(func() { fixture.stop(t) })
	return fixture
}

func (fixture *directFixture) dialer(peers direct.PeerFactory) *direct.Dialer {
	return &direct.Dialer{
		Peers: peers, Authorization: peeradapter.CapabilityAuthorizer{
			Credentials: directCredentialSource{credential: fixture.credential}, Now: func() time.Time { return fixture.now },
		},
		Now: func() time.Time { return fixture.now },
	}
}

func (fixture *directFixture) attempt(t *testing.T, generation clientruntime.SessionGeneration) clientruntime.AttemptRequest {
	return fixture.attemptWithLocators(t, generation, fixture.signalingAddress, fixture.iceAddress)
}

func (fixture *directFixture) attemptWithLocators(t *testing.T, generation clientruntime.SessionGeneration, signalingAddress, iceAddress string) clientruntime.AttemptRequest {
	t.Helper()
	target := endpoint.Endpoint{
		ID: "direct", DaemonIdentity: endpoint.DaemonIdentity{DeviceID: fixture.identity.DeviceID, DeviceFingerprint: fixture.identity.Fingerprint},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"lan": {
			ID: "lan", Kind: endpoint.RouteDirectWebRTCTCP, Enabled: true, Source: endpoint.SourceManual, PolicySource: endpoint.SourceUser,
			CredentialRef: "credential:direct", SignalingAddresses: []string{signalingAddress}, ICETCPAddresses: []string{iceAddress},
		}},
	}
	attempt, err := clientruntime.NewAttemptRequest(target, "lan", generation, clientruntime.ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	return attempt
}

type tcpMappingProxy struct {
	listener net.Listener
	target   string
}

func newTCPMappingProxy(t *testing.T, target string) *tcpMappingProxy {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	proxy := &tcpMappingProxy{listener: listener, target: target}
	go proxy.serve()
	t.Cleanup(func() { _ = listener.Close() })
	return proxy
}

func (proxy *tcpMappingProxy) address() string { return proxy.listener.Addr().String() }

func (proxy *tcpMappingProxy) serve() {
	for {
		incoming, err := proxy.listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer incoming.Close()
			outgoing, err := net.Dial("tcp", proxy.target)
			if err != nil {
				return
			}
			defer outgoing.Close()
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(outgoing, incoming); done <- struct{}{} }()
			go func() { _, _ = io.Copy(incoming, outgoing); done <- struct{}{} }()
			<-done
		}()
	}
}

func closedTCPAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func (fixture *directFixture) stop(t *testing.T) {
	t.Helper()
	fixture.stopOnce.Do(func() {
		fixture.cancel()
		_ = fixture.server.Close()
		select {
		case err := <-fixture.done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("Direct server shutdown: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("Direct server did not stop")
		}
	})
}

type directCredentialSource struct {
	credential remoteauth.ClientAccessCredential
}

type directCoreAccessService struct {
	store *remoteauth.AccessStore
}

func (directCoreAccessService) Identity(context.Context, []byte) (accesscontract.ClientAccessIdentity, error) {
	return accesscontract.ClientAccessIdentity{}, nil
}

func (directCoreAccessService) CreateTicket(context.Context, accesscontract.ClientAccessTicketRequest) (accesscontract.ClientAccessTicket, error) {
	return accesscontract.ClientAccessTicket{}, nil
}

func (directCoreAccessService) List(context.Context) ([]accesscontract.ClientAccessRecord, error) {
	return nil, nil
}

func (service directCoreAccessService) GrantActive(_ context.Context, grantID string, expiresAt, now time.Time) bool {
	return service.store.GrantActive(grantID, expiresAt, now)
}

func (directCoreAccessService) Revoke(context.Context, string) (accesscontract.ClientAccessRecord, error) {
	return accesscontract.ClientAccessRecord{}, errors.New("unused")
}

type countingHandler struct {
	inner  remote.SessionAcceptor
	closed *atomic.Int32
}

func (handler countingHandler) GrantActive(grantID string, expiresAt time.Time) bool {
	return handler.inner.GrantActive(grantID, expiresAt)
}

func (handler countingHandler) PairingClaimActive(digest, clientPublicKey []byte, expiresAt time.Time) bool {
	return handler.inner.PairingClaimActive(digest, clientPublicKey, expiresAt)
}

func (handler countingHandler) ServeDataChannel(ctx context.Context, connection transport.Transport, fingerprint string) error {
	err := handler.inner.ServeDataChannel(ctx, connection, fingerprint)
	handler.closed.Add(1)
	return err
}

func (source directCredentialSource) ResolveClientCredential(context.Context, string, string) (remoteauth.ClientAccessCredential, error) {
	return source.credential, nil
}

type blockingSignaling struct {
	started chan struct{}
}

func (signaling blockingSignaling) Exchange(ctx context.Context, _ []string, _ *remoteauthpb.DirectSignalingRequestV2) (*remoteauthpb.DirectSignalingAnswerV2, error) {
	close(signaling.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func directClientAPI() *pionwebrtc.API {
	settings := pionwebrtc.SettingEngine{}
	settings.SetNetworkTypes([]pionwebrtc.NetworkType{pionwebrtc.NetworkTypeTCP4})
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetIPFilter(func(address net.IP) bool { return address.IsLoopback() })
	return pionwebrtc.NewAPI(pionwebrtc.WithSettingEngine(settings))
}

func waitForDirectPTYOutput(t *testing.T, ctx context.Context, stream clientruntime.ResourceStream, needle string) string {
	t.Helper()
	output := &strings.Builder{}
	for {
		typ, payload, err := stream.Receive(ctx)
		if err != nil {
			t.Fatalf("receive Direct attachment frame: %v; output=%q", err, output.String())
		}
		switch typ {
		case wire.TypePTYOutput:
			output.Write(payload)
			if strings.Contains(output.String(), needle) {
				return output.String()
			}
		case wire.TypeError:
			t.Fatalf("Direct attachment stream error: %s", string(payload))
		case wire.TypeClosed, wire.TypeSyncLost:
			t.Fatalf("Direct attachment stream closed before %q; output=%q", needle, output.String())
		}
	}
}

func selectedPair(t *testing.T, peer *pionwebrtc.PeerConnection) *pionwebrtc.ICECandidatePair {
	t.Helper()
	if peer == nil || peer.SCTP() == nil || peer.SCTP().Transport() == nil || peer.SCTP().Transport().ICETransport() == nil {
		t.Fatal("Direct client peer transport is unavailable")
	}
	pair, err := peer.SCTP().Transport().ICETransport().GetSelectedCandidatePair()
	if err != nil || pair == nil || pair.Local == nil || pair.Remote == nil {
		t.Fatalf("Direct selected pair=%v err=%v", pair, err)
	}
	return pair
}

func signalingCode(err error) remoteauthpb.DirectSignalingErrorCode {
	var failure *direct.SignalingError
	if errors.As(err, &failure) {
		return failure.Code
	}
	return remoteauthpb.DirectSignalingErrorCode_DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED
}

func cloneRequest(request *remoteauthpb.DirectSignalingRequestV2) *remoteauthpb.DirectSignalingRequestV2 {
	return &remoteauthpb.DirectSignalingRequestV2{
		SchemaVersion: request.GetSchemaVersion(), RequestId: request.GetRequestId(), ExpectedDeviceId: request.GetExpectedDeviceId(),
		ExpectedDeviceFingerprint: request.GetExpectedDeviceFingerprint(), OfferSdp: request.GetOfferSdp(),
		IssuedAtUnixNano: request.GetIssuedAtUnixNano(), ExpiresAtUnixNano: request.GetExpiresAtUnixNano(),
		GrantId: request.GetGrantId(), GrantExpiresAtUnixNano: request.GetGrantExpiresAtUnixNano(),
		PairingClaimDigest: append([]byte(nil), request.GetPairingClaimDigest()...), PairingClientPublicKey: append([]byte(nil), request.GetPairingClientPublicKey()...),
		PairingExpiresAtUnixNano: request.GetPairingExpiresAtUnixNano(),
	}
}

// startAccessCoreForRemoteTest 启动 Phase 4 拓扑的远端 Core：
// daemon provider（unix）+ access server（Auth 直答 + 终端路由）。
// 返回的 access server 可直接作为 SessionAcceptor.Core。
func startAccessCoreForRemoteTest(t *testing.T, accessService accesscontract.ClientAccessService) (*accessserver.Server, func()) {
	t.Helper()
	providerSocket := filepath.Join(t.TempDir(), "core-provider.sock")
	coreServer := core.NewServer(
		core.WithSocketPath(providerSocket),
	)
	ctx, cancel := context.WithCancel(context.Background())
	providerServer, err := providercore.New(coreServer, providercore.Config{Socket: providerSocket})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- providerServer.ListenAndServe(ctx) }()
	t.Cleanup(func() { _ = providerServer.Shutdown(context.Background()) })
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(providerSocket); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	accessCore, err := accessserver.New(accessserver.Config{
		Socket: filepath.Join(t.TempDir(), "access.sock"),
		Auth:   &accessserver.AuthServices{Access: accessService},
		Files:  files.Config{TransferDir: filepath.Join(t.TempDir(), "transfers")},
		Provider: func(dialCtx context.Context) (terminalprovider.Provider, error) {
			return poolprovider.DialTerminal(dialCtx, providerSocket)
		},
	})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	return accessCore, func() {
		_ = accessCore.Close()
		cancel()
		_ = coreServer.Shutdown(context.Background())
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
}
