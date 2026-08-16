package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	apilayer "github.com/anytty/anytty/api_layer"
	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	core "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	remotev2daemon "github.com/anytty/anytty/remote/daemon"
	remotev2webrtc "github.com/anytty/anytty/remote/webrtc"
	"github.com/anytty/anytty/shared/remoteauth"
	pionwebrtc "github.com/pion/webrtc/v4"
	golangssh "golang.org/x/crypto/ssh"
)

func TestSSHDirectTCPIPCompletesWebRTCAuthHelloAndProtoAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("real sshd WebRTC harness is disabled in short mode")
	}
	fixture := newSSHWebRTCFixture(t)
	host := startIsolatedSSHD(t)
	defer host.Close()

	var clientPeer *pionwebrtc.PeerConnection
	clientAPI := sshClientAPI()
	dialer := NewDialer(Options{
		Peers: pionadapter.Factory{PeerConnections: func(configuration pionwebrtc.Configuration) (*pionwebrtc.PeerConnection, error) {
			peer, err := clientAPI.NewPeerConnection(configuration)
			if err == nil {
				clientPeer = peer
			}
			return peer, err
		}},
		Authorization: peeradapter.CapabilityAuthorizer{
			Credentials: sshCapabilitySource{credential: fixture.credential},
			Now:         func() time.Time { return fixture.now },
		},
		Credentials: sshCredentialSource{credential: port.SSHCredential{AuthMethods: []golangssh.AuthMethod{golangssh.PublicKeys(host.clientSigner)}}},
		ClientName:  "ssh-real-e2e",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ready, err := dialer.Connect(ctx, fixture.attempt(t, host))
	if err != nil {
		t.Fatal(err)
	}
	session, ok := ready.(*Session)
	if !ok {
		t.Fatalf("SSH ready session type = %T", ready)
	}
	forwardAddress := session.forwarder.listener.Addr().String()
	result, err := ready.(clientruntime.ApplicationReadyPeerSession).ExecuteApplication(ctx, &apipb.CommandEnvelope{
		Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}},
	})
	if err != nil || result.GetTerminalList() == nil {
		t.Fatalf("SSH Proto API result=%#v err=%v", result, err)
	}
	if ready.Readiness().Identity.DeviceFingerprint != fixture.identity.Fingerprint || !ready.Readiness().AuthorizationVerified {
		t.Fatalf("SSH readiness=%+v", ready.Readiness())
	}
	pair := selectedSSHTestPair(t, clientPeer)
	if pair.Local.Protocol != pionwebrtc.ICEProtocolTCP || pair.Remote.Protocol != pionwebrtc.ICEProtocolTCP {
		t.Fatalf("SSH selected candidate pair is not TCP: %s", pair)
	}
	if session.forwarder.accepted.Load() == 0 {
		t.Fatal("SSH ICE forwarder did not carry the selected TCP pair")
	}
	if err := ready.Close(); err != nil {
		t.Fatal(err)
	}
	if connection, err := net.DialTimeout("tcp", forwardAddress, 100*time.Millisecond); err == nil {
		_ = connection.Close()
		t.Fatalf("SSH ICE forward listener %s remained reachable after session close", forwardAddress)
	}
}

func TestDialSSHClientSupportsPasswordCredential(t *testing.T) {
	server := startPasswordSSHServer(t, "correct-password")
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	route := endpoint.AccessRoute{
		Host: "127.0.0.1", Port: uint16(server.port), User: "anytty",
		HostKeyFingerprints: []string{server.hostFingerprint},
	}
	client, err := dialSSHClient(ctx, route, []golangssh.AuthMethod{golangssh.Password("correct-password")}, time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
	if _, err := dialSSHClient(ctx, route, []golangssh.AuthMethod{golangssh.Password("wrong-password")}, time.Second, nil); err == nil {
		t.Fatal("wrong SSH password unexpectedly authenticated")
	}
}

func TestDialSSHClientCancellationClosesBlockedHandshake(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- connection
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	address := listener.Addr().(*net.TCPAddr)
	go func() {
		_, dialErr := dialSSHClient(ctx, endpoint.AccessRoute{
			Host: "127.0.0.1", Port: uint16(address.Port), User: "anytty", HostKeyFingerprints: []string{"SHA256:unused"},
		}, []golangssh.AuthMethod{golangssh.Password("unused")}, time.Second, nil)
		done <- dialErr
	}()
	connection := <-accepted
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled SSH handshake error=%v", err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 256)
	for {
		if _, err := connection.Read(buffer); err != nil {
			break
		}
	}
	_ = connection.Close()
}

type sshWebRTCFixture struct {
	identity         remoteauth.Identity
	credential       remoteauth.ClientAccessCredential
	now              time.Time
	signalingAddress string
	iceAddress       string
	server           *remotev2webrtc.DirectServer
	cancel           context.CancelFunc
	done             chan error
	stopOnce         sync.Once
}

func newSSHWebRTCFixture(t *testing.T) *sshWebRTCFixture {
	t.Helper()
	_, daemonKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := remoteauth.NewIdentity("device-ssh", daemonKey)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	clientIdentity, err := remoteauth.GenerateClientAccessIdentity("ssh-e2e", rand.Reader)
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
	exchanged, err := store.RedeemPairingBundle(payload, clientIdentity.PublicKey, "ssh-e2e", now)
	if err != nil {
		t.Fatal(err)
	}
	signaling, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ice, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		_ = signaling.Close()
		t.Fatal(err)
	}
	acceptor := remotev2daemon.SessionAcceptor{
		Core:     core.NewServer(core.WithApplicationExecutorFactory(apilayer.CoreApplicationExecutorFactory), core.WithClientAccessService(sshCoreAccessService{store: store})),
		Identity: identity, AccessStore: store, Now: func() time.Time { return now },
	}
	server, err := remotev2webrtc.NewDirectServer(identity, acceptor, signaling, ice, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	fixture := &sshWebRTCFixture{
		identity: identity,
		credential: remoteauth.ClientAccessCredential{
			Version: 3, EndpointID: "ssh-e2e", Identity: clientIdentity, CapabilityGrant: exchanged.Grant, UpdatedAt: now,
		},
		now: now, signalingAddress: signaling.Addr().String(), iceAddress: ice.Addr().String(),
		server: server, cancel: cancel, done: done,
	}
	t.Cleanup(func() { fixture.stop(t) })
	return fixture
}

type sshCoreAccessService struct {
	store *remoteauth.AccessStore
}

func (sshCoreAccessService) Identity(context.Context, []byte) (core.ClientAccessIdentity, error) {
	return core.ClientAccessIdentity{}, nil
}

func (sshCoreAccessService) CreateTicket(context.Context, core.ClientAccessTicketRequest) (core.ClientAccessTicket, error) {
	return core.ClientAccessTicket{}, nil
}

func (sshCoreAccessService) List(context.Context) ([]core.ClientAccessRecord, error) { return nil, nil }

func (service sshCoreAccessService) GrantActive(_ context.Context, grantID string, expiresAt, now time.Time) bool {
	return service.store.GrantActive(grantID, expiresAt, now)
}

func (sshCoreAccessService) Revoke(context.Context, string) (core.ClientAccessRecord, error) {
	return core.ClientAccessRecord{}, errors.New("unused")
}

func (fixture *sshWebRTCFixture) attempt(t *testing.T, host *isolatedSSHD) clientruntime.AttemptRequest {
	t.Helper()
	target := endpoint.Endpoint{
		ID: "ssh-e2e", DaemonIdentity: endpoint.DaemonIdentity{DeviceID: fixture.identity.DeviceID, DeviceFingerprint: fixture.identity.Fingerprint},
		Routes: map[endpoint.RouteID]endpoint.AccessRoute{"ssh": {
			ID: "ssh", Kind: endpoint.RouteSSHWebRTCTCP, Enabled: true, Source: endpoint.SourceManual, PolicySource: endpoint.SourceUser,
			Host: "127.0.0.1", Port: uint16(host.port), User: host.user,
			CredentialRef: "credential:ssh-e2e", SSHCredentialRef: "ssh:key",
			HostKeyFingerprints:    []string{host.hostFingerprint},
			RemoteSignalingAddress: fixture.signalingAddress, RemoteICETCPAddress: fixture.iceAddress,
		}},
	}
	attempt, err := clientruntime.NewAttemptRequest(target, "ssh", 1, clientruntime.ConnectIntentInteractive)
	if err != nil {
		t.Fatal(err)
	}
	return attempt
}

func (fixture *sshWebRTCFixture) stop(t *testing.T) {
	t.Helper()
	fixture.stopOnce.Do(func() {
		fixture.cancel()
		_ = fixture.server.Close()
		select {
		case err := <-fixture.done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("SSH Direct server shutdown: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("SSH Direct server did not stop")
		}
	})
}

type sshCapabilitySource struct {
	credential remoteauth.ClientAccessCredential
}

func (source sshCapabilitySource) ResolveClientCredential(context.Context, string, string) (remoteauth.ClientAccessCredential, error) {
	return source.credential, nil
}

type sshCredentialSource struct{ credential port.SSHCredential }

func (source sshCredentialSource) ResolveSSHCredential(context.Context, string, *endpoint.CredentialDescriptor) (port.SSHCredential, error) {
	return source.credential, nil
}

type isolatedSSHD struct {
	cmd             *exec.Cmd
	port            int
	user            string
	hostFingerprint string
	clientSigner    golangssh.Signer
}

func startIsolatedSSHD(t *testing.T) *isolatedSSHD {
	t.Helper()
	sshdPath := "/usr/sbin/sshd"
	if _, err := os.Stat(sshdPath); err != nil {
		t.Skipf("OpenSSH server unavailable: %v", err)
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skipf("ssh-keygen unavailable: %v", err)
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	portValue := freeTCPPort(t)
	hostKey := filepath.Join(dir, "host_key")
	clientKey := filepath.Join(dir, "client_key")
	runSSHCommand(t, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", hostKey)
	runSSHCommand(t, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", clientKey)
	clientPublic, err := os.ReadFile(clientKey + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	authorizedKeys := filepath.Join(dir, "authorized_keys")
	entry := "no-agent-forwarding,no-pty,no-user-rc,no-X11-forwarding " + strings.TrimSpace(string(clientPublic)) + "\n"
	if err := os.WriteFile(authorizedKeys, []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "sshd_config")
	configText := fmt.Sprintf("Port %d\nListenAddress 127.0.0.1\nHostKey %s\nPidFile %s\nAuthorizedKeysFile %s\nStrictModes no\nPasswordAuthentication no\nKbdInteractiveAuthentication no\nChallengeResponseAuthentication no\nUsePAM no\nPermitRootLogin no\nAllowUsers %s\nAllowTcpForwarding yes\nDisableForwarding no\nLogLevel ERROR\n", portValue, hostKey, filepath.Join(dir, "sshd.pid"), authorizedKeys, current.Username)
	if err := os.WriteFile(config, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(sshdPath, "-D", "-e", "-f", config)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waitForTCPListener(t, cmd, net.JoinHostPort("127.0.0.1", strconv.Itoa(portValue)), &stderr)
	hostPublic, err := os.ReadFile(hostKey + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	hostKeyValue, _, _, _, err := golangssh.ParseAuthorizedKey(hostPublic)
	if err != nil {
		t.Fatal(err)
	}
	clientPrivate, err := os.ReadFile(clientKey)
	if err != nil {
		t.Fatal(err)
	}
	clientSigner, err := golangssh.ParsePrivateKey(clientPrivate)
	if err != nil {
		t.Fatal(err)
	}
	return &isolatedSSHD{cmd: cmd, port: portValue, user: current.Username, hostFingerprint: golangssh.FingerprintSHA256(hostKeyValue), clientSigner: clientSigner}
}

func (host *isolatedSSHD) Close() {
	if host != nil && host.cmd != nil && host.cmd.Process != nil {
		_ = host.cmd.Process.Kill()
		_ = host.cmd.Wait()
	}
}

type passwordSSHServer struct {
	listener        net.Listener
	port            int
	hostFingerprint string
	done            chan struct{}
}

func startPasswordSSHServer(t *testing.T, password string) *passwordSSHServer {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := golangssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &passwordSSHServer{
		listener: listener, port: listener.Addr().(*net.TCPAddr).Port,
		hostFingerprint: golangssh.FingerprintSHA256(signer.PublicKey()), done: make(chan struct{}),
	}
	config := &golangssh.ServerConfig{PasswordCallback: func(metadata golangssh.ConnMetadata, candidate []byte) (*golangssh.Permissions, error) {
		if metadata.User() != "anytty" || string(candidate) != password {
			return nil, fmt.Errorf("invalid password")
		}
		return nil, nil
	}}
	config.AddHostKey(signer)
	go func() {
		defer close(server.done)
		for {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go servePasswordSSHConnection(connection, config)
		}
	}()
	return server
}

func servePasswordSSHConnection(connection net.Conn, config *golangssh.ServerConfig) {
	serverConnection, channels, requests, err := golangssh.NewServerConn(connection, config)
	if err != nil {
		_ = connection.Close()
		return
	}
	go golangssh.DiscardRequests(requests)
	go func() {
		for channel := range channels {
			_ = channel.Reject(golangssh.UnknownChannelType, "test server has no channels")
		}
	}()
	_ = serverConnection.Wait()
}

func (server *passwordSSHServer) Close() {
	_ = server.listener.Close()
	<-server.done
}

func sshClientAPI() *pionwebrtc.API {
	settings := pionwebrtc.SettingEngine{}
	settings.SetNetworkTypes([]pionwebrtc.NetworkType{pionwebrtc.NetworkTypeTCP4})
	settings.SetIncludeLoopbackCandidate(true)
	settings.SetIPFilter(func(address net.IP) bool { return address.IsLoopback() })
	return pionwebrtc.NewAPI(pionwebrtc.WithSettingEngine(settings))
}

func selectedSSHTestPair(t *testing.T, peer *pionwebrtc.PeerConnection) *pionwebrtc.ICECandidatePair {
	t.Helper()
	if peer == nil || peer.SCTP() == nil || peer.SCTP().Transport() == nil || peer.SCTP().Transport().ICETransport() == nil {
		t.Fatal("SSH client peer transport is unavailable")
	}
	pair, err := peer.SCTP().Transport().ICETransport().GetSelectedCandidatePair()
	if err != nil || pair == nil || pair.Local == nil || pair.Remote == nil {
		t.Fatalf("SSH selected pair=%v err=%v", pair, err)
	}
	return pair
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func runSSHCommand(t *testing.T, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
}

func waitForTCPListener(t *testing.T, command *exec.Cmd, address string, stderr *strings.Builder) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		if command.ProcessState != nil || time.Now().After(deadline) {
			_ = command.Process.Kill()
			t.Fatalf("sshd did not start: %v %s", err, stderr.String())
		}
		time.Sleep(25 * time.Millisecond)
	}
}

var _ port.SSHCredentialSource = sshCredentialSource{}
var _ peeradapter.CredentialSource = sshCapabilitySource{}
