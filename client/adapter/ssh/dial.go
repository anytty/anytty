package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/client/adapter/direct"
	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/remoteauthpb"
	golangssh "golang.org/x/crypto/ssh"
)

const (
	defaultClientName     = "anytty-go-ssh"
	defaultConnectTimeout = 10 * time.Second
	defaultSSHPort        = 22
)

// Options 是单次 ssh-webrtc-tcp connector 的依赖。
// Peers、Authorization 和 Credentials 都由 composition root 注入；adapter 不读取 registry、环境变量或平台 UI state。
type Options struct {
	Peers          direct.PeerFactory
	Authorization  peeradapter.Authorizer
	Credentials    port.SSHCredentialSource
	ClientName     string
	ConnectTimeout time.Duration
	NetworkDialer  direct.ContextDialer
}

// Dialer 通过 Go SSH client/direct-tcpip 到达 daemon loopback signaling 与 ICE-TCP。
// 成功结果复用 Direct connector 的 DTLS-bound auth、Hello、Proto API 和 DataChannel lifecycle。
type Dialer struct{ options Options }

// NewDialer 创建不拥有 route selection、fallback 或 session cache 的 SSH connector。
func NewDialer(options Options) *Dialer {
	options.ClientName = strings.TrimSpace(options.ClientName)
	if options.ClientName == "" {
		options.ClientName = defaultClientName
	}
	if options.ConnectTimeout <= 0 {
		options.ConnectTimeout = defaultConnectTimeout
	}
	return &Dialer{options: options}
}

// Connect 只执行 request 指定的 SSH Route；任一失败都会关闭 SSH client、forward listener、Pion peer 和 DataChannel。
func (dialer *Dialer) Connect(ctx context.Context, request clientruntime.AttemptRequest) (clientruntime.ReadyPeerSession, error) {
	if dialer == nil {
		return nil, fmt.Errorf("SSH dialer is required")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	route := request.Route()
	if route.Kind != endpoint.RouteSSHWebRTCTCP {
		return nil, fmt.Errorf("SSH adapter cannot dial route kind %s", route.Kind)
	}
	if dialer.options.Peers == nil || dialer.options.Authorization == nil || dialer.options.Credentials == nil {
		return nil, fmt.Errorf("SSH WebRTC connector dependencies are incomplete")
	}
	if request.DaemonIdentity().Empty() {
		return nil, fmt.Errorf("SSH WebRTC route requires a paired daemon identity")
	}
	if len(route.HostKeyFingerprints) == 0 {
		return nil, fmt.Errorf("SSH route %q requires at least one pinned host key", route.ID)
	}
	if strings.TrimSpace(route.ProxyJump) != "" {
		return nil, fmt.Errorf("SSH ProxyJump is unavailable until jump host identity pins are modeled")
	}
	clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseResolving, clientruntime.EndpointStageCredentialResolving)
	credential, err := dialer.options.Credentials.ResolveSSHCredential(ctx, route.SSHCredentialRef, route.CredentialDescriptor)
	if err != nil {
		return nil, fmt.Errorf("resolve SSH credential: %w", err)
	}
	clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageSSHConnecting)
	sshClient, err := dialSSHClient(ctx, route, credential.AuthMethods, dialer.options.ConnectTimeout, dialer.options.NetworkDialer)
	releaseErr := credential.Release()
	if err != nil {
		return nil, errors.Join(err, releaseErr)
	}
	if releaseErr != nil {
		_ = sshClient.Close()
		return nil, releaseErr
	}
	forwarder, err := newICEForwarder(sshClient, route.RemoteICETCPAddress)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	clientruntime.ReportEndpointProgress(ctx, clientruntime.EndpointPhaseConnecting, clientruntime.EndpointStageSSHTunnelReady)
	base, err := (&direct.Dialer{
		Peers:           dialer.options.Peers,
		Signaling:       direct.TCPSignalingClient{Dialer: sshClientDialer{client: sshClient}},
		Authorization:   dialer.options.Authorization,
		RouteKind:       endpoint.RouteSSHWebRTCTCP,
		Locators:        []string{route.RemoteSignalingAddress},
		TransformAnswer: forwarder.projectAnswer,
		ClientName:      dialer.options.ClientName,
	}).Connect(ctx, request)
	if err != nil {
		_ = forwarder.Close()
		_ = sshClient.Close()
		return nil, err
	}
	directSession, ok := base.(*direct.Session)
	if !ok {
		_ = base.Close()
		_ = forwarder.Close()
		_ = sshClient.Close()
		return nil, fmt.Errorf("SSH WebRTC connector returned an unexpected session type")
	}
	session := &Session{Session: directSession, forwarder: forwarder, sshClient: sshClient}
	go func() {
		<-directSession.Done()
		_ = session.closeTunnel()
	}()
	return session, nil
}

// Session 在共享 Direct Proto session 外增加 SSH client 与 direct-tcpip forward 生命周期。
type Session struct {
	*direct.Session
	forwarder  *iceForwarder
	sshClient  *golangssh.Client
	closeOnce  sync.Once
	closeErr   error
	tunnelOnce sync.Once
	tunnelErr  error
}

// Close 精确关闭 DataChannel session、forward listener 和 SSH connection；方法幂等。
func (session *Session) Close() error {
	if session == nil {
		return nil
	}
	session.closeOnce.Do(func() {
		if session.Session != nil {
			session.closeErr = session.Session.Close()
		}
		if err := session.closeTunnel(); session.closeErr == nil {
			session.closeErr = err
		}
	})
	return session.closeErr
}

// ConnectionSnapshot 复用同一 WebRTC selected pair，并把获胜 Route kind 修正为 SSH tunnel。
func (session *Session) ConnectionSnapshot(at time.Time) (clientruntime.ConnectionSnapshot, bool) {
	if session == nil || session.Session == nil {
		return clientruntime.ConnectionSnapshot{}, false
	}
	snapshot, ok := session.Session.ConnectionSnapshot(at)
	if ok {
		snapshot.RouteKind = endpoint.RouteSSHWebRTCTCP
	}
	return snapshot, ok
}

func (session *Session) closeTunnel() error {
	session.tunnelOnce.Do(func() {
		var errs []error
		if session.forwarder != nil {
			errs = append(errs, session.forwarder.Close())
		}
		if session.sshClient != nil {
			errs = append(errs, session.sshClient.Close())
		}
		session.tunnelErr = errors.Join(errs...)
	})
	return session.tunnelErr
}

type sshClientDialer struct{ client *golangssh.Client }

func (dialer sshClientDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if dialer.client == nil {
		return nil, fmt.Errorf("SSH client is unavailable")
	}
	return dialer.client.DialContext(ctx, network, address)
}

func dialSSHClient(ctx context.Context, route endpoint.AccessRoute, auth []golangssh.AuthMethod, timeout time.Duration, networkDialer direct.ContextDialer) (*golangssh.Client, error) {
	if len(auth) == 0 {
		return nil, fmt.Errorf("SSH credential returned no authentication methods")
	}
	user, host := sshUserAndHost(route)
	if user == "" || host == "" {
		return nil, fmt.Errorf("SSH route requires explicit user and host")
	}
	portValue := route.Port
	if portValue == 0 {
		portValue = defaultSSHPort
	}
	address := net.JoinHostPort(host, strconv.Itoa(int(portValue)))
	callback := pinnedHostKeyCallback(route.HostKeyFingerprints)
	config := &golangssh.ClientConfig{
		User: user, Auth: append([]golangssh.AuthMethod(nil), auth...), HostKeyCallback: callback,
		ClientVersion: "SSH-2.0-AnyTTY", Timeout: timeout,
	}
	dialer := networkDialer
	if dialer == nil {
		dialer = &net.Dialer{Timeout: timeout}
	}
	raw, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect SSH host: %w", err)
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = raw.Close()
		case <-stop:
		}
	}()
	connection, channels, requests, err := golangssh.NewClientConn(raw, address, config)
	close(stop)
	if err != nil {
		_ = raw.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("SSH handshake: %w", err)
	}
	return golangssh.NewClient(connection, channels, requests), nil
}

func sshUserAndHost(route endpoint.AccessRoute) (string, string) {
	user := strings.TrimSpace(route.User)
	host := strings.TrimSpace(route.Host)
	if index := strings.LastIndex(host, "@"); index >= 0 {
		if user == "" {
			user = strings.TrimSpace(host[:index])
		}
		host = strings.TrimSpace(host[index+1:])
	}
	return user, host
}

func pinnedHostKeyCallback(values []string) golangssh.HostKeyCallback {
	pins := make(map[string]struct{}, len(values))
	for _, value := range values {
		pins[strings.TrimSpace(value)] = struct{}{}
	}
	return func(_ string, _ net.Addr, key golangssh.PublicKey) error {
		fingerprint := golangssh.FingerprintSHA256(key)
		if _, ok := pins[fingerprint]; !ok {
			return fmt.Errorf("SSH host key fingerprint %q is not pinned", fingerprint)
		}
		return nil
	}
}

type iceForwarder struct {
	client        *golangssh.Client
	remoteAddress string
	listener      net.Listener
	mu            sync.Mutex
	connections   map[net.Conn]struct{}
	accepted      atomic.Uint64
	closeOnce     sync.Once
	closeErr      error
	wg            sync.WaitGroup
}

func newICEForwarder(client *golangssh.Client, remoteAddress string) (*iceForwarder, error) {
	if client == nil || strings.TrimSpace(remoteAddress) == "" {
		return nil, fmt.Errorf("SSH ICE forwarder requires client and remote address")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen SSH ICE forwarder: %w", err)
	}
	forwarder := &iceForwarder{client: client, remoteAddress: remoteAddress, listener: listener, connections: make(map[net.Conn]struct{})}
	forwarder.wg.Add(1)
	go forwarder.serve()
	return forwarder, nil
}

func (forwarder *iceForwarder) serve() {
	defer forwarder.wg.Done()
	for {
		local, err := forwarder.listener.Accept()
		if err != nil {
			return
		}
		forwarder.accepted.Add(1)
		forwarder.track(local, true)
		forwarder.wg.Add(1)
		go func() {
			defer forwarder.wg.Done()
			defer forwarder.track(local, false)
			defer local.Close()
			remote, err := forwarder.client.Dial("tcp", forwarder.remoteAddress)
			if err != nil {
				return
			}
			forwarder.track(remote, true)
			defer forwarder.track(remote, false)
			defer remote.Close()
			copyBidirectional(local, remote)
		}()
	}
}

func (forwarder *iceForwarder) projectAnswer(answer *remoteauthpb.DirectSignalingAnswerV2) (*remoteauthpb.DirectSignalingAnswerV2, error) {
	if answer == nil || forwarder == nil || forwarder.listener == nil {
		return nil, fmt.Errorf("SSH ICE answer projection is unavailable")
	}
	address, ok := forwarder.listener.Addr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("SSH ICE forwarder has non-TCP address")
	}
	return direct.ProjectVerifiedTCPAnswer(answer, []string{net.JoinHostPort(address.IP.String(), strconv.Itoa(address.Port))})
}

func copyBidirectional(left, right net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(left, right); done <- struct{}{} }()
	go func() { _, _ = io.Copy(right, left); done <- struct{}{} }()
	<-done
}

func (forwarder *iceForwarder) track(connection net.Conn, add bool) {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	if add {
		forwarder.connections[connection] = struct{}{}
	} else {
		delete(forwarder.connections, connection)
	}
}

func (forwarder *iceForwarder) Close() error {
	if forwarder == nil {
		return nil
	}
	forwarder.closeOnce.Do(func() {
		forwarder.closeErr = forwarder.listener.Close()
		forwarder.mu.Lock()
		connections := make([]net.Conn, 0, len(forwarder.connections))
		for connection := range forwarder.connections {
			connections = append(connections, connection)
		}
		forwarder.mu.Unlock()
		for _, connection := range connections {
			_ = connection.Close()
		}
		forwarder.wg.Wait()
	})
	return forwarder.closeErr
}

var _ clientruntime.PeerConnector = (*Dialer)(nil)
