// Package direct 实现不依赖 AnyTTY Cloud 的 daemon embedded signaling + ICE-TCP connector。
// Endpoint/Route 选择与 generation 属于 client/runtime；本包只执行当前 Direct attempt 的 signaling、DTLS auth、Hello 和资源清理。
package direct

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	peeradapter "github.com/anytty/anytty/client/adapter/peer"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	clientruntime "github.com/anytty/anytty/client/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/internal/protocol/directsignal"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/datachannel"
	"google.golang.org/protobuf/proto"
)

const defaultClientName = "anytty-go-direct"

// PeerFactory 创建只启用 ICE-TCP 的 WebRTC peer primitive。
// factory 不解析 Endpoint、不访问 credential，也不执行 signaling 或 remote auth。
type PeerFactory interface {
	// OpenDirectPeer 为当前 attempt 创建一个可靠有序 protocol DataChannel。
	OpenDirectPeer(context.Context) (port.WebRTCPeer, error)
}

// RouteNetworkPeerFactory is an optional platform extension for ephemeral LAN
// routes. The route carries only short-lived routing metadata; endpoint identity
// and authorization are still verified by the normal Direct protocol.
type RouteNetworkPeerFactory interface {
	OpenDirectPeerForRoute(context.Context, endpoint.AccessRoute) (port.WebRTCPeer, error)
	DialContextForRoute(context.Context, endpoint.AccessRoute, string, string) (net.Conn, error)
}

// SignalingClient 是 Direct connector 对 daemon embedded signaling 的单次 exchange 边界。
// 实现只能传输 generated Proto request/response，不能返回预授权 session 或修改 Endpoint pin。
type SignalingClient interface {
	// Exchange 在给定 locator 中建立一条 signaling TCP connection，并返回 daemon-signed answer。
	Exchange(context.Context, []string, *remoteauthpb.DirectSignalingRequestV2) (*remoteauthpb.DirectSignalingAnswerV2, error)
}

// Dialer 是 direct-webrtc-tcp Route 的 Go-owned connector。
// 成功结果已经完成 daemon-signed signaling、实际 DTLS-bound capability auth、protocol Hello 与 ReadyPeerSession 装配。
type Dialer struct {
	Peers           PeerFactory
	Signaling       SignalingClient
	Authorization   peeradapter.Authorizer
	RouteKind       endpoint.RouteKind
	Locators        []string
	TransformAnswer func(*remoteauthpb.DirectSignalingAnswerV2) (*remoteauthpb.DirectSignalingAnswerV2, error)
	Random          io.Reader
	Now             func() time.Time
	ClientName      string
	Phase           func(clientruntime.EndpointPhase)
}

// Connect 只尝试 request 指定的 Direct Route；任何失败都会关闭 peer、DataChannel 和 protocol client。
// signaling locator 变化不改变 Endpoint identity，answer 必须由 pin 对应的 daemon DeviceIdentity 签名。
func (dialer *Dialer) Connect(ctx context.Context, request clientruntime.AttemptRequest) (clientruntime.ReadyPeerSession, error) {
	if dialer == nil {
		return nil, fmt.Errorf("direct WebRTC connector is required")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	route := request.Route()
	expectedKind := dialer.RouteKind
	if expectedKind == "" {
		expectedKind = endpoint.RouteDirectWebRTCTCP
	}
	if route.Kind != expectedKind {
		return nil, fmt.Errorf("route %q kind %q does not match WebRTC connector kind %q", route.ID, route.Kind, expectedKind)
	}
	if dialer.Peers == nil || dialer.Authorization == nil {
		return nil, fmt.Errorf("direct WebRTC connector dependencies are incomplete")
	}
	startedAt := time.Now()
	lastAt := startedAt
	stage := func(name string) {
		now := time.Now()
		log.Printf("anytty direct connect generation=%d stage=%s stage_ms=%d total_ms=%d", request.Stamp().Generation, name, now.Sub(lastAt).Milliseconds(), now.Sub(startedAt).Milliseconds())
		lastAt = now
	}
	prepared, err := dialer.Authorization.Prepare(ctx, request)
	if err != nil {
		return nil, reportDirectFailure(request.Stamp().Generation, directFailureAuthorization, fmt.Errorf("prepare direct endpoint authorization: %w", err))
	}
	if prepared == nil {
		return nil, fmt.Errorf("direct endpoint authorizer returned no transaction")
	}
	stage("authorization_prepared")
	opened, err := openDirectPeer(ctx, request, directPeerOptions{
		Peers: dialer.Peers, Signaling: dialer.Signaling, Locators: dialer.Locators, TransformAnswer: dialer.TransformAnswer,
		Random: dialer.Random, Now: dialer.Now, Phase: dialer.Phase, Timing: stage,
		GrantID: prepared.GrantID(), GrantExpiresAt: prepared.GrantExpiresAt(),
	})
	if err != nil {
		return nil, reportDirectFailure(request.Stamp().Generation, directFailurePeerSetup, err)
	}
	peer := opened.peer
	connection := opened.connection
	closeAttempt := func() { _ = opened.Close() }
	fingerprint, err := opened.RemoteCertificateFingerprint()
	if err != nil {
		closeAttempt()
		return nil, reportDirectFailure(request.Stamp().Generation, directFailureDataChannelAuth, fmt.Errorf("read direct endpoint DTLS certificate: %w", err))
	}
	stage("dtls_fingerprint")
	dialer.reportPhase(clientruntime.EndpointPhaseAuthorizing)
	if _, err := prepared.Authenticate(ctx, connection, fingerprint); err != nil {
		closeAttempt()
		return nil, reportDirectFailure(request.Stamp().Generation, directFailureDataChannelAuth, fmt.Errorf("authenticate direct endpoint DataChannel: %w", err))
	}
	stage("authorization")
	protocolClient := internalprotocol.NewClient(connection)
	clientName := strings.TrimSpace(dialer.ClientName)
	if clientName == "" {
		clientName = defaultClientName
	}
	if err := protocolClient.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: clientName}); err != nil {
		_ = protocolClient.Close()
		_ = peer.Close()
		return nil, reportDirectFailure(request.Stamp().Generation, directFailureProtocolHello, fmt.Errorf("direct endpoint protocol Hello: %w", err))
	}
	stage("protocol_ready")
	application, err := protocoladapter.NewApplicationClientWithObservedPath(protocolClient, request.Stamp(), string(endpoint.PathDirect))
	if err != nil {
		_ = protocolClient.Close()
		_ = peer.Close()
		return nil, err
	}
	if err := application.MarkReady(clientruntime.ReadyPeerSessionEvidence{
		Identity: request.DaemonIdentity(), IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version,
	}); err != nil {
		_ = application.Close()
		_ = peer.Close()
		return nil, err
	}
	session := newSession(application, peer)
	dialer.reportPhase(clientruntime.EndpointPhaseReady)
	return session, nil
}

func directRequestID(randomSource io.Reader) (string, error) {
	if randomSource == nil {
		randomSource = rand.Reader
	}
	payload := make([]byte, 24)
	if _, err := io.ReadFull(randomSource, payload); err != nil {
		return "", fmt.Errorf("generate direct signaling request id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

type directPeerOptions struct {
	Peers            PeerFactory
	Signaling        SignalingClient
	Locators         []string
	TransformAnswer  func(*remoteauthpb.DirectSignalingAnswerV2) (*remoteauthpb.DirectSignalingAnswerV2, error)
	Random           io.Reader
	Now              func() time.Time
	Phase            func(clientruntime.EndpointPhase)
	Timing           func(string)
	GrantID          string
	GrantExpiresAt   time.Time
	PairingDigest    []byte
	PairingPublicKey []byte
	PairingExpiresAt time.Time
}

type openedDirectPeer struct {
	peer       port.WebRTCPeer
	connection *datachannel.Transport
	closeOnce  sync.Once
	closeErr   error
}

func openDirectPeer(ctx context.Context, request clientruntime.AttemptRequest, options directPeerOptions) (*openedDirectPeer, error) {
	if options.Peers == nil {
		return nil, fmt.Errorf("direct peer factory is required")
	}
	route := request.Route()
	if options.Phase != nil {
		options.Phase(clientruntime.EndpointPhaseConnecting)
	}
	peer, err := openDirectRoutePeer(ctx, options.Peers, route)
	if err != nil {
		return nil, fmt.Errorf("create direct endpoint peer: %w", err)
	}
	if peer == nil || peer.Channel() == nil {
		if peer != nil {
			_ = peer.Close()
		}
		return nil, fmt.Errorf("direct endpoint peer has no protocol DataChannel")
	}
	if options.Timing != nil {
		options.Timing("peer_open")
	}
	opened := &openedDirectPeer{peer: peer, connection: datachannel.New(peer.Channel())}
	offer, err := peer.CreateOffer(ctx)
	if err != nil {
		_ = opened.Close()
		return nil, fmt.Errorf("create direct endpoint offer: %w", err)
	}
	if options.Timing != nil {
		options.Timing("offer_gathered")
	}
	requestID, err := directRequestID(options.Random)
	if err != nil {
		_ = opened.Close()
		return nil, err
	}
	now := time.Now().UTC()
	if options.Now != nil {
		now = options.Now().UTC()
	}
	signalingRequest := &remoteauthpb.DirectSignalingRequestV2{
		SchemaVersion: remoteauth.DirectSignalingSchemaVersion, RequestId: requestID,
		ExpectedDeviceId: request.DaemonIdentity().DeviceID, ExpectedDeviceFingerprint: request.DaemonIdentity().DeviceFingerprint,
		OfferSdp: offer, IssuedAtUnixNano: now.UnixNano(), ExpiresAtUnixNano: now.Add(remoteauth.DirectSignalingMaxTTL).UnixNano(),
	}
	if options.GrantID != "" {
		signalingRequest.GrantId = options.GrantID
		if !options.GrantExpiresAt.IsZero() {
			signalingRequest.GrantExpiresAtUnixNano = options.GrantExpiresAt.UnixNano()
		}
	} else if len(options.PairingDigest) > 0 && len(options.PairingPublicKey) > 0 && !options.PairingExpiresAt.IsZero() {
		signalingRequest.PairingClaimDigest = append([]byte(nil), options.PairingDigest...)
		signalingRequest.PairingClientPublicKey = append([]byte(nil), options.PairingPublicKey...)
		signalingRequest.PairingExpiresAtUnixNano = options.PairingExpiresAt.UnixNano()
	}
	if options.Phase != nil {
		options.Phase(clientruntime.EndpointPhaseSignaling)
	}
	signaling := options.Signaling
	if signaling == nil {
		if routed, ok := options.Peers.(RouteNetworkPeerFactory); ok {
			signaling = TCPSignalingClient{Dialer: routeContextDialer{route: route, factory: routed}}
		} else {
			signaling = TCPSignalingClient{}
		}
	}
	locators := options.Locators
	if len(locators) == 0 {
		locators = route.SignalingAddresses
	}
	answer, err := signaling.Exchange(ctx, locators, signalingRequest)
	if err != nil {
		_ = opened.Close()
		return nil, err
	}
	if options.Timing != nil {
		options.Timing("signaling_answer")
	}
	verifyNow := time.Now().UTC()
	if options.Now != nil {
		verifyNow = options.Now().UTC()
	}
	if err := remoteauth.VerifyDirectSignalingAnswer(answer, requestID, request.DaemonIdentity().DeviceID, request.DaemonIdentity().DeviceFingerprint, verifyNow); err != nil {
		_ = opened.Close()
		return nil, fmt.Errorf("verify direct signaling answer: %w", err)
	}
	if route.Kind == endpoint.RouteDirectWebRTCTCP {
		answer, err = ProjectVerifiedTCPAnswer(answer, route.ICETCPAddresses)
		if err != nil {
			_ = opened.Close()
			return nil, fmt.Errorf("project verified Direct ICE-TCP answer: %w", err)
		}
	}
	if options.TransformAnswer != nil {
		answer, err = options.TransformAnswer(proto.Clone(answer).(*remoteauthpb.DirectSignalingAnswerV2))
		if err != nil {
			_ = opened.Close()
			return nil, fmt.Errorf("project verified WebRTC answer: %w", err)
		}
		if answer == nil {
			_ = opened.Close()
			return nil, fmt.Errorf("project verified WebRTC answer: empty result")
		}
	}
	candidates := make([]port.ICECandidate, 0, len(answer.GetCandidates()))
	for _, candidate := range answer.GetCandidates() {
		if candidate == nil {
			continue
		}
		candidates = append(candidates, port.ICECandidate{
			Candidate: candidate.GetCandidate(), SDPMid: candidate.GetSdpMid(), SDPMLineIndex: candidate.GetSdpMlineIndex(), UsernameFragment: candidate.GetUsernameFragment(),
		})
	}
	if err := peer.ApplyAnswer(ctx, answer.GetAnswerSdp(), candidates); err != nil {
		_ = opened.Close()
		return nil, fmt.Errorf("apply direct endpoint answer: %w", err)
	}
	if options.Timing != nil {
		options.Timing("answer_applied")
	}
	if err := peer.WaitReady(ctx); err != nil {
		_ = opened.Close()
		return nil, fmt.Errorf("wait direct endpoint DataChannel: %w", err)
	}
	if options.Timing != nil {
		options.Timing("datachannel_ready")
	}
	if peer.ObservedPath() != endpoint.PathDirect {
		_ = opened.Close()
		return nil, fmt.Errorf("direct endpoint established unexpected path %q", peer.ObservedPath())
	}
	return opened, nil
}

func openDirectRoutePeer(ctx context.Context, peers PeerFactory, route endpoint.AccessRoute) (port.WebRTCPeer, error) {
	if routed, ok := peers.(RouteNetworkPeerFactory); ok {
		return routed.OpenDirectPeerForRoute(ctx, route)
	}
	return peers.OpenDirectPeer(ctx)
}

type routeContextDialer struct {
	route   endpoint.AccessRoute
	factory RouteNetworkPeerFactory
}

func (dialer routeContextDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return dialer.factory.DialContextForRoute(ctx, dialer.route, network, address)
}

// ProjectVerifiedTCPAnswer 把已经通过 daemon DeviceIdentity 签名验证的 TCP candidates 投影到 Route 声明的可达 locator。
// 该函数不验证签名也不选择 Route；调用方必须先完成 VerifyDirectSignalingAnswer，地址投影失败时必须终止当前 attempt。
func ProjectVerifiedTCPAnswer(answer *remoteauthpb.DirectSignalingAnswerV2, addresses []string) (*remoteauthpb.DirectSignalingAnswerV2, error) {
	if answer == nil {
		return nil, fmt.Errorf("verified Direct answer is required")
	}
	locators, err := tcpCandidateLocators(addresses)
	if err != nil {
		return nil, err
	}
	projected := proto.Clone(answer).(*remoteauthpb.DirectSignalingAnswerV2)
	if answerAlreadyUsesTCPCandidateLocators(answer, locators) {
		return projected, nil
	}
	lines := strings.Split(projected.GetAnswerSdp(), "\n")
	projectedLines := make([]string, 0, len(lines)+len(locators))
	sdpProjected := false
	for _, line := range lines {
		ending := ""
		if strings.HasSuffix(line, "\r") {
			line, ending = strings.TrimSuffix(line, "\r"), "\r"
		}
		if !strings.HasPrefix(line, "a=candidate:") || !isTCPCandidate(strings.TrimPrefix(line, "a=")) {
			projectedLines = append(projectedLines, line+ending)
			continue
		}
		if sdpProjected {
			continue
		}
		for _, locator := range locators {
			projectedLines = append(projectedLines, "a="+projectTCPCandidate(strings.TrimPrefix(line, "a="), locator.host, locator.port)+ending)
		}
		sdpProjected = true
	}
	projected.AnswerSdp = strings.Join(projectedLines, "\n")
	projected.Candidates = nil
	for _, candidate := range answer.GetCandidates() {
		if candidate == nil || !isTCPCandidate(candidate.GetCandidate()) {
			continue
		}
		for _, locator := range locators {
			cloned := proto.Clone(candidate).(*remoteauthpb.DirectIceCandidate)
			cloned.Candidate = projectTCPCandidate(cloned.GetCandidate(), locator.host, locator.port)
			projected.Candidates = append(projected.Candidates, cloned)
		}
		break
	}
	return projected, nil
}

func answerAlreadyUsesTCPCandidateLocators(answer *remoteauthpb.DirectSignalingAnswerV2, locators []tcpCandidateLocator) bool {
	wanted := make(map[string]struct{}, len(locators))
	for _, locator := range locators {
		wanted[net.JoinHostPort(locator.host, strconv.Itoa(locator.port))] = struct{}{}
	}
	found := false
	check := func(candidate string) bool {
		fields := strings.Fields(candidate)
		if len(fields) < 8 || !strings.EqualFold(fields[2], "tcp") {
			return true
		}
		found = true
		_, ok := wanted[net.JoinHostPort(fields[4], fields[5])]
		return ok
	}
	for _, line := range strings.Split(answer.GetAnswerSdp(), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "a=candidate:") && !check(strings.TrimPrefix(line, "a=")) {
			return false
		}
	}
	for _, candidate := range answer.GetCandidates() {
		if candidate != nil && !check(candidate.GetCandidate()) {
			return false
		}
	}
	return found
}

type tcpCandidateLocator struct {
	host string
	port int
}

func tcpCandidateLocators(addresses []string) ([]tcpCandidateLocator, error) {
	canonical := make(map[string]tcpCandidateLocator, len(addresses))
	for _, address := range addresses {
		host, portValue, err := net.SplitHostPort(strings.TrimSpace(address))
		port, portErr := strconv.ParseUint(portValue, 10, 16)
		if err != nil || portErr != nil || port == 0 || strings.TrimSpace(host) == "" || host == "0.0.0.0" || host == "::" {
			return nil, fmt.Errorf("ICE-TCP locator %q must be a reachable HOST:PORT", address)
		}
		key := net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(int(port)))
		canonical[key] = tcpCandidateLocator{host: strings.TrimSpace(host), port: int(port)}
	}
	if len(canonical) == 0 {
		return nil, fmt.Errorf("at least one ICE-TCP locator is required")
	}
	keys := make([]string, 0, len(canonical))
	for key := range canonical {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]tcpCandidateLocator, 0, len(keys))
	for _, key := range keys {
		result = append(result, canonical[key])
	}
	return result, nil
}

func isTCPCandidate(candidate string) bool {
	fields := strings.Fields(candidate)
	return len(fields) >= 8 && strings.EqualFold(fields[2], "tcp")
}

func projectTCPCandidate(candidate, host string, port int) string {
	fields := strings.Fields(candidate)
	if len(fields) < 8 || !strings.EqualFold(fields[2], "tcp") {
		return candidate
	}
	fields[4], fields[5] = host, strconv.Itoa(port)
	return strings.Join(fields, " ")
}

func (peer *openedDirectPeer) Transport() transport.Transport {
	if peer == nil {
		return nil
	}
	return peer.connection
}

func (peer *openedDirectPeer) RemoteCertificateFingerprint() (string, error) {
	if peer == nil || peer.peer == nil {
		return "", fmt.Errorf("direct peer is unavailable")
	}
	return peer.peer.RemoteCertificateFingerprint()
}

func (peer *openedDirectPeer) Close() error {
	if peer == nil {
		return nil
	}
	peer.closeOnce.Do(func() {
		if peer.connection != nil {
			peer.closeErr = peer.connection.Close()
		}
		if peer.peer != nil {
			if err := peer.peer.Close(); peer.closeErr == nil {
				peer.closeErr = err
			}
		}
	})
	return peer.closeErr
}

func (dialer *Dialer) reportPhase(phase clientruntime.EndpointPhase) {
	if dialer != nil && dialer.Phase != nil {
		dialer.Phase(phase)
	}
}

// TCPSignalingClient 使用首个可建立 TCP connection 的 locator 完成一次 Proto exchange。
// 地址选择只发生在写入 request 前；一旦请求可能已被 daemon 消费，就不重放到其他地址。
type TCPSignalingClient struct {
	Dialer ContextDialer
}

type signalingDialResult struct {
	connection net.Conn
	address    string
	err        error
}

// ContextDialer 是 Direct/SSH signaling 对单条 TCP connection 的最小拨号要求。
// SSH 实现通过 direct-tcpip 返回 net.Conn；默认实现仍使用标准 net.Dialer。
type ContextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

// Exchange 建连后写入一个 request 并读取一个 response；context 取消会立即打断当前 socket。
func (client TCPSignalingClient) Exchange(ctx context.Context, addresses []string, request *remoteauthpb.DirectSignalingRequestV2) (*remoteauthpb.DirectSignalingAnswerV2, error) {
	if request == nil || len(addresses) == 0 {
		return nil, fmt.Errorf("direct signaling request and addresses are required")
	}
	dialer := client.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	dialContext, cancelDial := context.WithCancel(ctx)
	defer cancelDial()
	unique := make(map[string]struct{}, len(addresses))
	results := make(chan signalingDialResult, len(addresses))
	dialCount := 0
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}
		if _, exists := unique[address]; exists {
			continue
		}
		unique[address] = struct{}{}
		dialCount++
		go func() {
			connection, err := dialer.DialContext(dialContext, "tcp", address)
			results <- signalingDialResult{connection: connection, address: address, err: err}
		}()
	}
	if dialCount == 0 {
		return nil, fmt.Errorf("direct signaling addresses are empty")
	}
	var connection net.Conn
	var dialErrors []error
	for received := 0; received < dialCount; received++ {
		select {
		case <-ctx.Done():
			cancelDial()
			go closeLateSignalingConnections(results, dialCount-received)
			return nil, ctx.Err()
		case result := <-results:
			if result.err != nil || result.connection == nil {
				if result.err == nil {
					result.err = errors.New("dialer returned no connection")
				}
				dialErrors = append(dialErrors, fmt.Errorf("%s: %w", result.address, result.err))
				continue
			}
			connection = result.connection
			cancelDial()
			go closeLateSignalingConnections(results, dialCount-received-1)
			received = dialCount
		}
	}
	if connection == nil {
		return nil, &clientruntime.Error{
			Code: clientruntime.ErrorUnavailable, Message: "direct signaling is unavailable",
			Cause: errors.Join(dialErrors...), Attempted: true, Retryable: true,
		}
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.SetDeadline(time.Now())
		case <-stop:
		}
	}()
	defer close(stop)
	if err := directsignal.WriteMessage(connection, request); err != nil {
		return nil, err
	}
	response := &remoteauthpb.DirectSignalingResponseV2{}
	if err := directsignal.ReadMessage(connection, response); err != nil {
		return nil, err
	}
	switch payload := response.GetPayload().(type) {
	case *remoteauthpb.DirectSignalingResponseV2_Answer:
		if payload.Answer == nil {
			return nil, fmt.Errorf("direct signaling returned an empty answer")
		}
		return payload.Answer, nil
	case *remoteauthpb.DirectSignalingResponseV2_Error:
		if payload.Error == nil {
			return nil, fmt.Errorf("direct signaling returned an empty error")
		}
		return nil, &SignalingError{Code: payload.Error.GetCode(), Message: payload.Error.GetMessage()}
	default:
		return nil, fmt.Errorf("direct signaling returned an unknown response")
	}
}

func closeLateSignalingConnections(results <-chan signalingDialResult, remaining int) {
	for ; remaining > 0; remaining-- {
		result := <-results
		if result.connection != nil {
			_ = result.connection.Close()
		}
	}
}

// SignalingError 是 daemon 返回的稳定 Direct signaling admission 失败。
type SignalingError struct {
	Code    remoteauthpb.DirectSignalingErrorCode
	Message string
}

// Error 返回脱敏错误文本；调用方应使用 Code 分类，不解析字符串。
func (failure *SignalingError) Error() string {
	if failure == nil {
		return ""
	}
	return failure.Message
}

// Session 把 authenticated protocol client 与当前 Pion peer 绑定为 exact-close ReadyPeerSession。
// ApplicationClient 拥有 Proto command/event/resource；本类型只补齐 peer 生命周期，不能创建新 generation。
type Session struct {
	*protocoladapter.ApplicationClient
	peer      port.WebRTCPeer
	closeOnce sync.Once
	closeErr  error
}

func newSession(application *protocoladapter.ApplicationClient, peer port.WebRTCPeer) *Session {
	session := &Session{ApplicationClient: application, peer: peer}
	go func() {
		<-application.Done()
		_ = session.Close()
	}()
	return session
}

// ExecuteApplication 通过当前 generation 的 ApplicationSession 写入 correlation stamp 后执行 generated Proto command。
func (session *Session) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.ApplicationSession.Execute(ctx, command)
}

// ExecuteApplicationTerminal 为 resource-producing command 保留有界 terminal response，并使用同一 generation fence。
func (session *Session) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.ApplicationSession.ExecuteTerminal(ctx, command)
}

// ConnectionSnapshot 投影 Direct ReadySession 的实际 selected ICE-TCP pair。
func (session *Session) ConnectionSnapshot(at time.Time) (clientruntime.ConnectionSnapshot, bool) {
	if session == nil || session.ApplicationClient == nil {
		return clientruntime.ConnectionSnapshot{}, false
	}
	result := clientruntime.ConnectionSnapshot{
		RouteID: session.Stamp().RouteID, RouteKind: endpoint.RouteDirectWebRTCTCP,
		ObservedPath: session.ObservedPath(), SampledAt: at.UTC(), Connected: true,
	}
	if snapshot, ok := session.peer.Snapshot(at); ok {
		result.ObservedPath = string(snapshot.Path)
		result.PairID = snapshot.PairID
		result.SampledAt = snapshot.At
		result.RoundTrip = snapshot.RoundTrip
		result.LocalCandidateType = snapshot.LocalCandidateType
		result.RemoteCandidateType = snapshot.RemoteCandidateType
		result.LocalAddress = snapshot.LocalAddress
		result.RemoteAddress = snapshot.RemoteAddress
		result.LocalPort = snapshot.LocalPort
		result.RemotePort = snapshot.RemotePort
		result.LocalRelatedAddress = snapshot.LocalRelatedAddress
		result.RemoteRelatedAddress = snapshot.RemoteRelatedAddress
		result.LocalRelatedPort = snapshot.LocalRelatedPort
		result.RemoteRelatedPort = snapshot.RemoteRelatedPort
		result.LocalProtocol = snapshot.LocalProtocol
		result.RemoteProtocol = snapshot.RemoteProtocol
		result.RelayTransport = snapshot.RelayProtocol
		result.NetworkClass = snapshot.NetworkClass
		result.BytesSent = snapshot.BytesSent
		result.BytesReceived = snapshot.BytesRecv
		result.PacketsSent = snapshot.PacketsSent
		result.LossEvents = snapshot.LossEvents
		result.Connected = snapshot.Connected
	}
	return result, true
}

// Close 幂等关闭 protocol/DataChannel 与 Pion peer，并等待两侧资源释放请求完成。
func (session *Session) Close() error {
	if session == nil {
		return nil
	}
	session.closeOnce.Do(func() {
		if session.ApplicationClient != nil {
			session.closeErr = session.ApplicationClient.Close()
		}
		if session.peer != nil {
			if err := session.peer.Close(); session.closeErr == nil {
				session.closeErr = err
			}
		}
	})
	return session.closeErr
}

var _ clientruntime.PeerConnector = (*Dialer)(nil)
var _ clientruntime.ApplicationReadyPeerSession = (*Session)(nil)
