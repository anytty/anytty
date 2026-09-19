package cloud

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	accesscontract "github.com/anytty/anytty/access/contract"
	protocoladapter "github.com/anytty/anytty/access/engine/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	poolprovider "github.com/anytty/anytty/access/provider/pool"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	remote "github.com/anytty/anytty/access/remote"
	accessserver "github.com/anytty/anytty/access/server"
	"github.com/anytty/anytty/access/transport/webrtc"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	core "github.com/anytty/anytty/pool/core"
	providercore "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/proto/access/apipb"
	"github.com/anytty/anytty/proto/access/wire"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport/datachannel"
	pion "github.com/pion/webrtc/v4"
)

const cloudE2ETimeout = 10 * time.Second

// TestCloudManagedSessionTerminalOverRealPionDataChannel 覆盖 Cloud managed
// WebRTC 的完整真实路径：mock AgentGateway 传递真实 Pion client peer offer，
// runtime 用 remote.SessionAcceptor + 真实 access server(answerer) 应答，
// client 侧完成 DTLS binding 的 ClientHandshake、protocol Hello v7 与
// terminal create/list/attach/input/output。文件传输未覆盖（Direct/SSH 已覆盖），
// 这里保持有界并只验证 Cloud 路径的 session 语义。
func TestCloudManagedSessionTerminalOverRealPionDataChannel(t *testing.T) {
	api := daemonLoopbackWebRTCAPI()
	runtime, _ := daemonRuntimeFixture(t, webrtc.Answerer{PeerConnections: api.NewPeerConnection})
	store := runtime.config.AccessStore
	identity := runtime.config.Identity
	accessCore, stopAccessCore := startCloudAccessCoreForRemoteTest(t, store)
	t.Cleanup(stopAccessCore)
	runtime.config.Answerer = webrtc.Answerer{
		Handler:         remote.SessionAcceptor{Core: accessCore, Identity: identity, AccessStore: store},
		PeerConnections: api.NewPeerConnection,
	}

	// Client identity + capability credential redeemed from the fixture store,
	// mirroring the pairing-backed public client path.
	now := time.Now().UTC()
	client, err := remoteauth.GenerateClientAccessIdentity("cloud-e2e", rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := remoteauth.NewPrivateClientAccessSigner(client)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := store.IssuePairingBundle(remoteauth.PairingIssueOptions{
		Label: "cloud-e2e", Scope: remoteauth.FullDaemonScope(), TicketTTL: time.Hour, GrantLifetime: time.Hour, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := remoteauth.EncodePairingBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	exchanged, err := store.RedeemPairingBundle(payload, client.PublicKey, "cloud-e2e", now)
	if err != nil {
		t.Fatal(err)
	}
	credential := remoteauth.ClientAccessCredential{
		Version: 3, EndpointID: "cloud-e2e", Identity: client, CapabilityGrant: exchanged.Grant, UpdatedAt: now,
	}

	clientPeer, channel := cloudOfferPeerWithDataChannel(t, api)
	t.Cleanup(func() { _ = clientPeer.Close() })
	channelOpened := make(chan struct{})
	channel.OnOpen(func() { close(channelOpened) })
	transport := datachannel.New(webrtc.NewChannel(channel))
	t.Cleanup(func() { _ = transport.Close() })

	gateway := &daemonTestAgentGateway{
		offer:  daemonAgentOffer(t, clientPeer, client.PublicKey),
		answer: make(chan *cloudv1.AgentAnswer, 1),
	}
	locator := startDaemonTestAgentGateway(t, gateway)

	ctx, cancel := context.WithTimeout(context.Background(), cloudE2ETimeout)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- runtime.connectEdge(ctx, runtime.currentRecord().DaemonID, &cloudv1.SignedEnvelope{KeyId: "test-binding"}, locator, 0)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("cloud connectEdge shutdown: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("cloud connectEdge did not stop")
		}
	})

	applyDaemonAnswer(t, clientPeer, waitDaemonAnswer(t, gateway.answer))
	select {
	case <-channelOpened:
	case <-ctx.Done():
		t.Fatal("Cloud protocol DataChannel did not open")
	case <-time.After(cloudE2ETimeout):
		t.Fatal("Cloud protocol DataChannel did not open")
	}

	// The client pins the daemon's actual DTLS certificate, not the SDP.
	fingerprint, err := webrtc.RemoteCertificateFingerprint(clientPeer)
	if err != nil {
		t.Fatalf("read Cloud daemon DTLS certificate: %v", err)
	}
	binding, err := remoteauth.DTLSChannelBinding(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := (remoteauth.ClientHandshake{}).Authenticate(ctx, transport, remoteauth.ClientHandshakeRequest{
		ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.Fingerprint,
		Credential: credential, Signer: signer, ChannelBinding: binding,
	})
	if err != nil {
		t.Fatalf("Cloud DataChannel capability handshake: %v", err)
	}
	if claims.IssuerDeviceID != identity.DeviceID || claims.SubjectKeyFingerprint != client.Fingerprint {
		t.Fatalf("Cloud claims = %+v", claims)
	}

	protocolClient := internalprotocol.NewClient(transport)
	if err := protocolClient.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "cloud-e2e"}); err != nil {
		t.Fatalf("Cloud protocol Hello: %v", err)
	}
	application, err := protocoladapter.NewApplicationClientWithObservedPath(protocolClient, clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("cloud"), RouteID: clientendpoint.RouteID("cloud-e2e"), Generation: 1,
	}, "cloud-e2e")
	if err != nil {
		t.Fatal(err)
	}
	if err := application.MarkReady(clientruntime.ReadyPeerSessionEvidence{
		Identity:         clientendpoint.DaemonIdentity{DeviceID: identity.DeviceID, DeviceFingerprint: identity.Fingerprint},
		IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version,
	}); err != nil {
		t.Fatal(err)
	}

	created, err := application.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: "cloud-e2e", Name: "cloud-e2e", Command: []string{"/bin/sh"}, Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}})
	if err != nil {
		t.Fatalf("Cloud terminal create: %v", err)
	}
	ref := created.GetTerminal().GetRef()
	if ref.GetTerminalId() != "cloud-e2e" {
		t.Fatalf("Cloud terminal ref = %#v", ref)
	}
	list, err := application.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatalf("Cloud terminal list: %v", err)
	}
	found := false
	for _, terminal := range list.GetTerminals() {
		if terminal.GetRef().GetTerminalId() == "cloud-e2e" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Cloud terminal list = %#v", list.GetTerminals())
	}

	attached, err := application.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal: ref, Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER, SurfaceId: "cloud-e2e", ViewId: "e2e",
	})
	if err != nil {
		t.Fatalf("Cloud terminal attach: %v", err)
	}
	attachment := attached.GetAttachment()
	stream, err := application.OpenResourceStream(attachment.GetResource())
	if err != nil {
		t.Fatalf("Cloud open attachment stream: %v", err)
	}
	defer func() { _ = stream.Close() }()
	if err := application.TerminalInput(ctx, &apipb.TerminalInputCommand{
		Attachment: attachment.GetResource(), Data: []byte("echo CLOUD-E2E-$(echo OK)\r"),
	}); err != nil {
		t.Fatalf("Cloud terminal input: %v", err)
	}
	if output := waitForCloudPTYOutput(t, ctx, stream, "CLOUD-E2E-OK"); output == "" {
		t.Fatal("Cloud PTY output did not reach the client")
	}

	if err := application.TerminalDetach(ctx, &apipb.TerminalDetachCommand{Attachment: attachment.GetResource()}); err != nil {
		t.Fatalf("Cloud terminal detach: %v", err)
	}
	if err := application.TerminalKill(ctx, &apipb.TerminalKillCommand{Terminal: ref}); err != nil {
		t.Fatalf("Cloud terminal kill: %v", err)
	}
}

func cloudOfferPeerWithDataChannel(t *testing.T, api *pion.API) (*pion.PeerConnection, *pion.DataChannel) {
	t.Helper()
	peer, err := api.NewPeerConnection(pion.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := peer.CreateDataChannel("protocol", nil)
	if err != nil {
		_ = peer.Close()
		t.Fatal(err)
	}
	return peer, channel
}

func waitForCloudPTYOutput(t *testing.T, ctx context.Context, stream clientruntime.ResourceStream, needle string) string {
	t.Helper()
	output := &bytes.Buffer{}
	for {
		typ, payload, err := stream.Receive(ctx)
		if err != nil {
			t.Fatalf("receive Cloud attachment frame: %v; output=%q", err, output.String())
		}
		switch typ {
		case wire.TypePTYOutput:
			output.Write(payload)
			if bytes.Contains(output.Bytes(), []byte(needle)) {
				return output.String()
			}
		case wire.TypeError:
			t.Fatalf("Cloud attachment stream error: %s", string(payload))
		case wire.TypeClosed, wire.TypeSyncLost:
			t.Fatalf("Cloud attachment stream closed before %q; output=%q", needle, output.String())
		}
	}
}

// startCloudAccessCoreForRemoteTest 在测试进程内启动 pool provider（unix）+
// access server 的 Phase 4 拓扑，作为 Cloud SessionAcceptor.Core。这是 direct
// 测试同款 helper 的本地最小副本（package cloud 无法导入 direct_test）。
func startCloudAccessCoreForRemoteTest(t *testing.T, store *remoteauth.AccessStore) (*accessserver.Server, func()) {
	t.Helper()
	providerSocket := filepath.Join(t.TempDir(), "core-provider.sock")
	coreServer := core.NewServer(core.WithSocketPath(providerSocket))
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
		Auth:   &accessserver.AuthServices{Access: cloudCoreAccessService{store: store}},
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

// cloudCoreAccessService 是 access server Auth 直答面，仅把 grant 真值委托给
// fixture store；Cloud session 授权本身由 SessionAcceptor 的 AccessStore 完成。
type cloudCoreAccessService struct {
	store *remoteauth.AccessStore
}

func (cloudCoreAccessService) Identity(context.Context, []byte) (accesscontract.ClientAccessIdentity, error) {
	return accesscontract.ClientAccessIdentity{}, nil
}

func (cloudCoreAccessService) CreateTicket(context.Context, accesscontract.ClientAccessTicketRequest) (accesscontract.ClientAccessTicket, error) {
	return accesscontract.ClientAccessTicket{}, nil
}

func (cloudCoreAccessService) List(context.Context) ([]accesscontract.ClientAccessRecord, error) {
	return nil, nil
}

func (service cloudCoreAccessService) GrantActive(_ context.Context, grantID string, expiresAt, now time.Time) bool {
	return service.store.GrantActive(grantID, expiresAt, now)
}

func (cloudCoreAccessService) Revoke(context.Context, string) (accesscontract.ClientAccessRecord, error) {
	return accesscontract.ClientAccessRecord{}, errors.New("unused")
}

var _ accesscontract.ClientAccessService = cloudCoreAccessService{}
