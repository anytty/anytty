// Package pion 提供 native Go 平台的 WebRTC primitive adapter。
// signaling、remote auth、protocol Hello 和 session generation 均由上层 Route/runtime 持有，本包只操作 Pion peer 与 DataChannel。
package pion

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	"github.com/anytty/anytty/proto/wire"
	remotewebrtc "github.com/anytty/anytty/remote/webrtc"
	pionice "github.com/pion/ice/v4"
	"github.com/pion/transport/v4"
	pionwebrtc "github.com/pion/webrtc/v4"
)

const (
	protocolChannelLabel = "protocol"
	peerReadyTimeout     = 15 * time.Second
	directGatherTimeout  = 3 * time.Second
	cloudGatherTimeout   = 8 * time.Second
)

// Factory 创建当前 native 进程使用的 Pion PeerConnection。
// 它不持有 endpoint、credential、Cloud client 或 route winner，因此可被桌面与 Android Go library 共同复用。
type Factory struct {
	// PeerConnections 只覆盖底层 Pion primitive 创建策略；nil 使用当前生产默认配置。
	PeerConnections remotewebrtc.PeerConnectionFactory
	// Network 是固定的网络接口快照；nil 使用 Pion 默认网络枚举。
	// 它保留给无需按 peer 刷新网络的 native 平台；不能与 NetworkFactory 同时设置。
	Network transport.Net
	// NetworkFactory 在每次创建 PeerConnection 前取得当前网络接口快照。
	// Android 使用它绕过受限 netlink，并确保网络切换后的新 peer 不复用旧地址。
	NetworkFactory func() (transport.Net, error)
	// Logger owns Pion diagnostics. nil is silent and never falls back to stderr.
	Logger *slog.Logger
}

// OpenDirectPeer 创建只启用 ICE-TCP 的 native Pion peer，供 Direct 与后续 SSH tunnel connector 使用。
// 默认 factory 不发布 UDP candidate；测试可以通过 PeerConnections 注入受控 API，但不能改变上层 Route 或授权语义。
func (factory Factory) OpenDirectPeer(_ context.Context) (port.WebRTCPeer, error) {
	peerFactory := factory.PeerConnections
	if peerFactory == nil {
		network, err := factory.network()
		if err != nil {
			return nil, err
		}
		settings := pionwebrtc.SettingEngine{}
		settings.LoggerFactory = remotewebrtc.NewLoggerFactory(factory.Logger)
		settings.SetNetworkTypes([]pionwebrtc.NetworkType{pionwebrtc.NetworkTypeTCP4, pionwebrtc.NetworkTypeTCP6})
		settings.SetIncludeLoopbackCandidate(true)
		if network != nil {
			settings.SetNet(network)
			// Android sandbox 禁止 mDNS 内部绕过 transport.Net 再读取系统网卡。
			// Direct 使用 daemon-signed locator，Cloud 使用显式 ICE server，二者均不依赖 mDNS candidate。
			settings.SetICEMulticastDNSMode(pionice.MulticastDNSModeDisabled)
		}
		peerFactory = newPeerConnectionAPI(settings).NewPeerConnection
	}
	return openPeer(peerFactory, pionwebrtc.Configuration{}, false, true, directGatherTimeout, remotewebrtc.ICEGatheringDirectGrace)
}

// OpenCloudPeer 创建允许 ICE-UDP host/srflx candidate 的 native Pion peer。
// TURN server 与 ICE transport policy 由 Cloud adapter 按当前 attempt 显式传入。
func (factory Factory) OpenCloudPeer(_ context.Context, config port.WebRTCConfig) (port.WebRTCPeer, error) {
	peerFactory := factory.PeerConnections
	if peerFactory == nil {
		network, err := factory.network()
		if err != nil {
			return nil, err
		}
		if network == nil {
			peerFactory = func(configuration pionwebrtc.Configuration) (*pionwebrtc.PeerConnection, error) {
				return remotewebrtc.NewPeerConnectionWithLogger(configuration, factory.Logger)
			}
		} else {
			settings := pionwebrtc.SettingEngine{}
			settings.LoggerFactory = remotewebrtc.NewLoggerFactory(factory.Logger)
			settings.SetNet(network)
			settings.SetICEMulticastDNSMode(pionice.MulticastDNSModeDisabled)
			peerFactory = newPeerConnectionAPI(settings).NewPeerConnection
		}
	}
	configuration := pionwebrtc.Configuration{ICEServers: make([]pionwebrtc.ICEServer, 0, len(config.Servers))}
	for _, server := range config.Servers {
		configuration.ICEServers = append(configuration.ICEServers, pionwebrtc.ICEServer{URLs: append([]string(nil), server.URLs...), Username: server.Username, Credential: server.Credential})
	}
	if config.Policy == port.ICETransportRelayOnly {
		configuration.ICETransportPolicy = pionwebrtc.ICETransportPolicyRelay
	}
	return openPeer(
		peerFactory,
		configuration,
		config.Policy == port.ICETransportRelayOnly,
		false,
		cloudGatherTimeout,
		remotewebrtc.ICEGatheringPreferredGrace(len(config.Servers) > 0),
	)
}

func (factory Factory) network() (transport.Net, error) {
	if factory.Network != nil && factory.NetworkFactory != nil {
		return nil, fmt.Errorf("Pion network and network factory are mutually exclusive")
	}
	if factory.NetworkFactory == nil {
		return factory.Network, nil
	}
	network, err := factory.NetworkFactory()
	if err != nil {
		return nil, fmt.Errorf("create Pion network snapshot: %w", err)
	}
	if network == nil {
		return nil, fmt.Errorf("create Pion network snapshot: factory returned nil")
	}
	return network, nil
}

func newPeerConnectionAPI(settings pionwebrtc.SettingEngine) *pionwebrtc.API {
	settings.SetSCTPMaxMessageSize(wire.MaxEncodedFrameSize)
	return pionwebrtc.NewAPI(pionwebrtc.WithSettingEngine(settings))
}

func openPeer(
	peerFactory remotewebrtc.PeerConnectionFactory,
	configuration pionwebrtc.Configuration,
	waitForRelayCandidate, allowEmptyCandidates bool,
	gatherTimeout, gatheringGrace time.Duration,
) (port.WebRTCPeer, error) {
	peer, err := peerFactory(configuration)
	if err != nil {
		return nil, err
	}
	channel, err := peer.CreateDataChannel(protocolChannelLabel, nil)
	if err != nil {
		_ = peer.Close()
		return nil, err
	}
	ready := make(chan struct{})
	closed := make(chan struct{})
	connectionFailed := make(chan error, 1)
	channelAdapter := remotewebrtc.NewChannel(channel)
	value := &webRTCPeer{
		peer: peer, channel: channelAdapter, ready: ready, channelClosed: closed,
		connectionFailed: connectionFailed, readyTimeout: peerReadyTimeout,
		waitForRelayCandidate: waitForRelayCandidate, allowEmptyCandidates: allowEmptyCandidates,
		gatherTimeout: gatherTimeout, gatheringGrace: gatheringGrace,
	}
	channel.OnOpen(func() { value.readyOnce.Do(func() { close(ready) }) })
	channelAdapter.SetCloseHandler(func() { value.channelClosedOnce.Do(func() { close(closed) }) })
	peer.OnConnectionStateChange(func(state pionwebrtc.PeerConnectionState) {
		value.handleConnectionState(state)
	})
	return value, nil
}

type webRTCPeer struct {
	peer                  *pionwebrtc.PeerConnection
	channel               port.WebRTCMessageChannel
	ready                 chan struct{}
	readyOnce             sync.Once
	channelClosed         chan struct{}
	channelClosedOnce     sync.Once
	channelCloseOnce      sync.Once
	channelCloseErr       error
	connectionFailed      chan error
	connectionFailureOnce sync.Once
	readyTimeout          time.Duration
	waitForRelayCandidate bool
	allowEmptyCandidates  bool
	gatherTimeout         time.Duration
	gatheringGrace        time.Duration
	closeOnce             sync.Once
	closeErr              error
}

// handleConnectionState 把 Pion 的最终失败接回 protocol/DataChannel lifecycle。
// Ready 之前 connectionFailed 结束建连等待；Ready 之后关闭同一 channel，使 protocol Done 和 SessionOwner generation 失效，禁止留下半开 application session。
func (peer *webRTCPeer) handleConnectionState(state pionwebrtc.PeerConnectionState) {
	var failure error
	switch state {
	case pionwebrtc.PeerConnectionStateFailed:
		failure = fmt.Errorf("WebRTC peer failed")
	case pionwebrtc.PeerConnectionStateClosed:
		failure = fmt.Errorf("WebRTC peer closed")
	default:
		return
	}
	peer.connectionFailureOnce.Do(func() { peer.connectionFailed <- failure })
	// Pion 的状态回调不能同步等待 PeerConnection teardown；channel close 会让现有 protocol read loop 自行完成。
	go peer.closeProtocolChannel()
}

func (peer *webRTCPeer) closeProtocolChannel() {
	peer.channelCloseOnce.Do(func() {
		if peer.channel != nil {
			peer.channelCloseErr = peer.channel.Close()
		}
	})
}

func (peer *webRTCPeer) Channel() port.WebRTCMessageChannel { return peer.channel }

func (peer *webRTCPeer) CreateOffer(ctx context.Context) (string, error) {
	offer, err := peer.peer.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	gatherComplete := pionwebrtc.GatheringCompletePromise(peer.peer)
	gathering := remotewebrtc.NewICEGatheringWaiter(peer.waitForRelayCandidate, peer.allowEmptyCandidates, peer.gatheringGrace)
	peer.peer.OnICECandidate(gathering.Observe)
	if err := peer.peer.SetLocalDescription(offer); err != nil {
		return "", err
	}
	if err := gathering.Wait(ctx, gatherComplete, peer.gatherTimeout); err != nil {
		return "", err
	}
	description := peer.peer.LocalDescription()
	if description == nil || strings.TrimSpace(description.SDP) == "" {
		return "", fmt.Errorf("WebRTC offer has no local description")
	}
	return description.SDP, nil
}

func (peer *webRTCPeer) ApplyAnswer(ctx context.Context, answer string, candidates []port.ICECandidate) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if strings.TrimSpace(answer) == "" {
		return fmt.Errorf("WebRTC answer is empty")
	}
	if err := peer.peer.SetRemoteDescription(pionwebrtc.SessionDescription{Type: pionwebrtc.SDPTypeAnswer, SDP: answer}); err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := peer.peer.AddICECandidate(toPionCandidate(candidate)); err != nil {
			return err
		}
	}
	return nil
}

func (peer *webRTCPeer) WaitReady(ctx context.Context) error {
	select {
	case <-peer.ready:
		return nil
	default:
	}
	timeout := peer.readyTimeout
	if timeout <= 0 {
		timeout = peerReadyTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-peer.ready:
		return nil
	case <-peer.channelClosed:
		return fmt.Errorf("WebRTC protocol DataChannel closed before becoming ready")
	case err := <-peer.connectionFailed:
		return err
	case <-timer.C:
		return fmt.Errorf("WebRTC protocol DataChannel was not ready within %s", timeout)
	}
}

func (peer *webRTCPeer) RemoteCertificateFingerprint() (string, error) {
	return remotewebrtc.RemoteCertificateFingerprint(peer.peer)
}

func (peer *webRTCPeer) ObservedPath() endpoint.Path {
	_, local, remote, _, ok := peer.selectedCandidatePair()
	if !ok {
		return ""
	}
	return candidatePath(local, remote)
}

func (peer *webRTCPeer) Snapshot(at time.Time) (port.WebRTCPeerSnapshot, bool) {
	pair, local, remote, report, ok := peer.selectedCandidatePair()
	if !ok {
		return port.WebRTCPeerSnapshot{}, false
	}
	rtt := secondsDuration(pair.CurrentRoundTripTime)
	if rtt == 0 {
		for _, stat := range report {
			if sctp, ok := stat.(pionwebrtc.SCTPTransportStats); ok {
				rtt = secondsDuration(sctp.SmoothedRoundTripTime)
				break
			}
		}
	}
	localRelatedAddress, localRelatedPort, remoteRelatedAddress, remoteRelatedPort := peer.selectedRelatedAddresses(local, remote)
	return port.WebRTCPeerSnapshot{
		PairID: pair.ID, Path: candidatePath(local, remote), NetworkClass: strings.ToLower(strings.TrimSpace(local.NetworkType)), At: at.UTC(),
		RoundTrip: rtt, BytesSent: pair.BytesSent, BytesRecv: pair.BytesReceived, PacketsSent: uint64(pair.PacketsSent),
		LossEvents:         saturatingAdd(pair.RetransmissionsSent, uint64(pair.PacketsDiscardedOnSend)),
		Connected:          peer.peer.ConnectionState() == pionwebrtc.PeerConnectionStateConnected,
		LocalCandidateType: strings.ToLower(local.CandidateType.String()), RemoteCandidateType: strings.ToLower(remote.CandidateType.String()),
		LocalAddress: strings.TrimSpace(local.IP), RemoteAddress: strings.TrimSpace(remote.IP),
		LocalPort: candidatePort(local.Port), RemotePort: candidatePort(remote.Port),
		LocalRelatedAddress: localRelatedAddress, RemoteRelatedAddress: remoteRelatedAddress,
		LocalRelatedPort: localRelatedPort, RemoteRelatedPort: remoteRelatedPort,
		LocalProtocol: strings.ToLower(strings.TrimSpace(local.Protocol)), RemoteProtocol: strings.ToLower(strings.TrimSpace(remote.Protocol)),
		RelayProtocol: strings.ToLower(strings.TrimSpace(local.RelayProtocol)),
	}, true
}

func (peer *webRTCPeer) selectedRelatedAddresses(local, remote pionwebrtc.ICECandidateStats) (string, uint16, string, uint16) {
	if peer == nil || peer.peer == nil {
		return "", 0, "", 0
	}
	sctp := peer.peer.SCTP()
	if sctp == nil || sctp.Transport() == nil || sctp.Transport().ICETransport() == nil {
		return "", 0, "", 0
	}
	selected, err := sctp.Transport().ICETransport().GetSelectedCandidatePair()
	if err != nil || selected == nil {
		return "", 0, "", 0
	}
	return relatedCandidateAddresses(selected, local, remote)
}

func relatedCandidateAddresses(pair *pionwebrtc.ICECandidatePair, local, remote pionwebrtc.ICECandidateStats) (string, uint16, string, uint16) {
	if pair == nil || !candidateMatchesStats(pair.Local, local) || !candidateMatchesStats(pair.Remote, remote) {
		// The selected pair changed between the stats and candidate reads. Never mix two pairs.
		return "", 0, "", 0
	}
	return strings.TrimSpace(pair.Local.RelatedAddress), pair.Local.RelatedPort,
		strings.TrimSpace(pair.Remote.RelatedAddress), pair.Remote.RelatedPort
}

func candidateMatchesStats(candidate *pionwebrtc.ICECandidate, stats pionwebrtc.ICECandidateStats) bool {
	return candidate != nil && strings.TrimSpace(candidate.Address) == strings.TrimSpace(stats.IP) &&
		candidate.Port == candidatePort(stats.Port) && candidate.Typ == stats.CandidateType
}

type selectedCandidatePairStatsReader interface {
	GetSelectedCandidatePairStats() (pionwebrtc.ICECandidatePairStats, bool)
}

func (peer *webRTCPeer) selectedCandidatePair() (
	pionwebrtc.ICECandidatePairStats,
	pionwebrtc.ICECandidateStats,
	pionwebrtc.ICECandidateStats,
	pionwebrtc.StatsReport,
	bool,
) {
	if peer == nil || peer.peer == nil {
		return pionwebrtc.ICECandidatePairStats{}, pionwebrtc.ICECandidateStats{}, pionwebrtc.ICECandidateStats{}, nil, false
	}
	sctp := peer.peer.SCTP()
	if sctp == nil || sctp.Transport() == nil || sctp.Transport().ICETransport() == nil {
		return pionwebrtc.ICECandidatePairStats{}, pionwebrtc.ICECandidateStats{}, pionwebrtc.ICECandidateStats{}, nil, false
	}
	report := peer.peer.GetStats()
	pair, local, remote, ok := selectedCandidatePair(sctp.Transport().ICETransport(), report)
	return pair, local, remote, report, ok
}

func selectedCandidatePair(
	reader selectedCandidatePairStatsReader,
	report pionwebrtc.StatsReport,
) (pionwebrtc.ICECandidatePairStats, pionwebrtc.ICECandidateStats, pionwebrtc.ICECandidateStats, bool) {
	if reader == nil {
		return pionwebrtc.ICECandidatePairStats{}, pionwebrtc.ICECandidateStats{}, pionwebrtc.ICECandidateStats{}, false
	}
	pair, ok := reader.GetSelectedCandidatePairStats()
	if !ok {
		return pionwebrtc.ICECandidatePairStats{}, pionwebrtc.ICECandidateStats{}, pionwebrtc.ICECandidateStats{}, false
	}
	local, localOK := report[pair.LocalCandidateID].(pionwebrtc.ICECandidateStats)
	remote, remoteOK := report[pair.RemoteCandidateID].(pionwebrtc.ICECandidateStats)
	if !localOK || !remoteOK {
		return pionwebrtc.ICECandidatePairStats{}, pionwebrtc.ICECandidateStats{}, pionwebrtc.ICECandidateStats{}, false
	}
	return pair, local, remote, true
}

func candidatePort(port int32) uint16 {
	if port <= 0 || port > 65535 {
		return 0
	}
	return uint16(port)
}

func (peer *webRTCPeer) Close() error {
	if peer == nil {
		return nil
	}
	peer.closeOnce.Do(func() {
		peer.closeProtocolChannel()
		peer.closeErr = peer.channelCloseErr
		if peer.channelClosed != nil {
			timer := time.NewTimer(250 * time.Millisecond)
			select {
			case <-peer.channelClosed:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
			}
		}
		if err := peer.peer.Close(); peer.closeErr == nil {
			peer.closeErr = err
		}
	})
	return peer.closeErr
}

func toPionCandidate(candidate port.ICECandidate) pionwebrtc.ICECandidateInit {
	mid := candidate.SDPMid
	lineIndex := uint16(candidate.SDPMLineIndex)
	usernameFragment := candidate.UsernameFragment
	result := pionwebrtc.ICECandidateInit{Candidate: candidate.Candidate, SDPMLineIndex: &lineIndex}
	if mid != "" {
		result.SDPMid = &mid
	}
	if usernameFragment != "" {
		result.UsernameFragment = &usernameFragment
	}
	return result
}

func candidatePath(local, remote pionwebrtc.ICECandidateStats) endpoint.Path {
	if local.CandidateType == pionwebrtc.ICECandidateTypeRelay || remote.CandidateType == pionwebrtc.ICECandidateTypeRelay {
		return endpoint.PathSingleRelay
	}
	return endpoint.PathDirect
}

func secondsDuration(seconds float64) time.Duration {
	if seconds <= 0 {
		return 0
	}
	if seconds >= float64(math.MaxInt64)/float64(time.Second) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(seconds * float64(time.Second))
}

func saturatingAdd(left, right uint64) uint64 {
	if math.MaxUint64-left < right {
		return math.MaxUint64
	}
	return left + right
}

var _ port.WebRTCPeer = (*webRTCPeer)(nil)
