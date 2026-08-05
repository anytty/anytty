package webrtc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/datachannel"
	pion "github.com/pion/webrtc/v4"
)

const protocolChannelLabel = "protocol"

const (
	directGatherTimeout = 3 * time.Second
	cloudGatherTimeout  = 5 * time.Second
)

// ICECandidate 是 remote WebRTC primitive 的中性信令字段。
// 它不包含 Cloud 票据、Presence、Route 策略或 terminal 授权。
type ICECandidate struct {
	Candidate        string
	SDPMid           string
	SDPMLineIndex    uint32
	UsernameFragment string
}

// ICEServer 是创建 PeerConnection 所需的中性 ICE server 配置。
// credential 的来源与租约校验属于调用方，本包只把已经验证的值交给 Pion。
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

// SignalingOffer 是 Answerer 接受的单次 WebRTC offer。
// SessionID 只用于请求与响应关联，不代表 Cloud 或 daemon session 真值。
type SignalingOffer struct {
	SessionID  string
	SDP        string
	Candidates []ICECandidate
}

// SignalingAnswer 是 Answerer 完成 gathering 后返回的 SDP 与 ICE candidate。
type SignalingAnswer struct {
	SessionID  string
	SDP        string
	Candidates []ICECandidate
	lifecycle  *peerLifecycle
}

// peerLifecycle owns one Pion peer and its single authorized protocol handler.
// Finalization always runs outside Pion callbacks and the handler goroutine.
type peerLifecycle struct {
	peer             *pion.PeerConnection
	cancel           context.CancelFunc
	onPeerClosed     func()
	closePeerForTest func(*pion.PeerConnection) error

	mu              sync.Mutex
	handlerClaimed  bool
	handlerSealed   bool
	handlerFinished bool
	handlerDone     chan struct{}
	closing         chan struct{}
	watcherDone     chan struct{}
	done            chan struct{}
	closeOnce       sync.Once
}

func newPeerLifecycle(ctx context.Context, peer *pion.PeerConnection, cancel context.CancelFunc, onPeerClosed func(), closePeerForTest func(*pion.PeerConnection) error) *peerLifecycle {
	lifecycle := &peerLifecycle{
		peer: peer, cancel: cancel, onPeerClosed: onPeerClosed, closePeerForTest: closePeerForTest,
		handlerDone: make(chan struct{}), closing: make(chan struct{}), watcherDone: make(chan struct{}), done: make(chan struct{}),
	}
	go func() {
		defer close(lifecycle.watcherDone)
		select {
		case <-ctx.Done():
			lifecycle.requestClose()
		case <-lifecycle.closing:
		}
	}()
	return lifecycle
}

func (lifecycle *peerLifecycle) claimHandler() bool {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	if lifecycle.handlerSealed || lifecycle.handlerClaimed {
		return false
	}
	lifecycle.handlerClaimed = true
	return true
}

func (lifecycle *peerLifecycle) finishHandler() {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	if !lifecycle.handlerClaimed || lifecycle.handlerFinished {
		return
	}
	lifecycle.handlerFinished = true
	close(lifecycle.handlerDone)
}

func (lifecycle *peerLifecycle) requestClose() {
	if lifecycle == nil {
		return
	}
	lifecycle.closeOnce.Do(func() {
		lifecycle.mu.Lock()
		lifecycle.handlerSealed = true
		if !lifecycle.handlerClaimed {
			lifecycle.handlerFinished = true
			close(lifecycle.handlerDone)
		}
		lifecycle.mu.Unlock()
		close(lifecycle.closing)
		lifecycle.cancel()
		go lifecycle.finalize()
	})
}

func (lifecycle *peerLifecycle) closeAndWait() {
	if lifecycle == nil {
		return
	}
	lifecycle.requestClose()
	<-lifecycle.done
}

func (lifecycle *peerLifecycle) finalize() {
	func() {
		defer func() { _ = recover() }()
		if lifecycle.closePeerForTest != nil {
			_ = lifecycle.closePeerForTest(lifecycle.peer)
		} else if lifecycle.peer != nil {
			_ = lifecycle.peer.GracefulClose()
		}
	}()
	<-lifecycle.handlerDone
	<-lifecycle.watcherDone
	func() {
		defer func() { _ = recover() }()
		if lifecycle.onPeerClosed != nil {
			lifecycle.onPeerClosed()
		}
	}()
	close(lifecycle.done)
}

// DataChannelSessionHandler 是 daemon 侧 DTLS DataChannel 的端到端授权 owner。
// 实现必须先在 DataChannel 内完成 DeviceIdentity proof 与 CapabilityGrant 校验，再把受限 scope 交给 core-v2。
type DataChannelSessionHandler interface {
	// ServeDataChannel 接收尚未授权的可靠有序 DataChannel；实现完成握手前不得调用 core-v2。
	// daemonDTLSFingerprint 必须由 WebRTC adapter 从当前本端 DTLSTransport 读取，不能来自 SDP。
	ServeDataChannel(context.Context, transport.Transport, string) error
}

// Answerer 把一个公开 WebRTC offer 协商成 daemon answer。
// PeerConnection 只负责 ICE/DTLS/SCTP，不接收 grant、terminal payload 或 Cloud runtime 类型。
type Answerer struct {
	Handler DataChannelSessionHandler
	// PeerConnections 只允许注入 Pion primitive 创建策略；nil 保持当前生产默认配置。
	PeerConnections PeerConnectionFactory
	// PionLogger owns embedded Pion diagnostics when PeerConnections is nil.
	PionLogger *slog.Logger
	// CloseOnDisconnected 用于 Direct/SSH ICE-TCP 的短连接 owner：对端关闭后立即释放共享 TCPMux ufrag。
	CloseOnDisconnected bool
	// OnPeerClosed 在 Pion peer 真正关闭后调用一次，供 listener 释放有界 admission token。
	OnPeerClosed func()
	// OnSessionStart 在可靠有序 DataChannel 打开并即将进入 daemon 端到端授权入口时调用。
	// 它只用于连接级可观测性，不能修改授权、session generation 或 PeerConnection lifecycle。
	OnSessionStart func()
	// OnSessionError 接收 DataChannel 端到端认证或 application handler 的失败。
	OnSessionError   func(error)
	closePeerForTest func(*pion.PeerConnection) error
}

// Answer 创建 WebRTC answer，并把唯一可靠有序的 anytty DataChannel 交给端到端授权 handler。
func (answerer Answerer) Answer(ctx context.Context, offer *SignalingOffer, iceServers []ICEServer) (*SignalingAnswer, error) {
	if answerer.Handler == nil {
		return nil, fmt.Errorf("remote daemon authorized data channel handler is not configured")
	}
	if offer == nil || strings.TrimSpace(offer.SDP) == "" {
		return nil, fmt.Errorf("remote daemon signaling offer is empty")
	}
	configuration := pion.Configuration{ICEServers: make([]pion.ICEServer, 0, len(iceServers))}
	for _, server := range iceServers {
		if len(server.URLs) == 0 {
			continue
		}
		configuration.ICEServers = append(configuration.ICEServers, pion.ICEServer{
			URLs: append([]string(nil), server.URLs...), Username: server.Username, Credential: server.Credential,
		})
	}
	peerFactory := answerer.PeerConnections
	if peerFactory == nil {
		peerFactory = func(configuration pion.Configuration) (*pion.PeerConnection, error) {
			return NewPeerConnectionWithLogger(configuration, answerer.PionLogger)
		}
	}
	peer, err := peerFactory(configuration)
	if err != nil {
		return nil, fmt.Errorf("create remote daemon peer connection: %w", err)
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	lifecycle := newPeerLifecycle(ctx, peer, cancel, answerer.OnPeerClosed, answerer.closePeerForTest)
	var candidateMu sync.Mutex
	candidates := make([]ICECandidate, 0, 4)
	gathering := NewICEGatheringWaiter(false, len(iceServers) == 0, ICEGatheringPreferredGrace(len(iceServers) > 0))
	peer.OnICECandidate(func(candidate *pion.ICECandidate) {
		if candidate == nil {
			return
		}
		gathering.Observe(candidate)
		json := candidate.ToJSON()
		wire := ICECandidate{Candidate: json.Candidate}
		if json.SDPMid != nil {
			wire.SDPMid = *json.SDPMid
		}
		if json.SDPMLineIndex != nil {
			wire.SDPMLineIndex = uint32(*json.SDPMLineIndex)
		}
		if json.UsernameFragment != nil {
			wire.UsernameFragment = *json.UsernameFragment
		}
		candidateMu.Lock()
		candidates = append(candidates, wire)
		candidateMu.Unlock()
	})
	peer.OnConnectionStateChange(func(state pion.PeerConnectionState) {
		if state == pion.PeerConnectionStateClosed {
			lifecycle.requestClose()
			return
		}
		if state == pion.PeerConnectionStateFailed || answerer.CloseOnDisconnected && state == pion.PeerConnectionStateDisconnected {
			lifecycle.requestClose()
		}
	})
	peer.OnDataChannel(func(channel *pion.DataChannel) {
		if channel.Label() != protocolChannelLabel || !channel.Ordered() || channel.MaxPacketLifeTime() != nil || channel.MaxRetransmits() != nil {
			channel.OnOpen(func() { _ = channel.Close() })
			return
		}
		protocolTransport := datachannel.New(NewChannel(channel))
		channel.OnOpen(func() {
			if !lifecycle.claimHandler() {
				_ = protocolTransport.Close()
				return
			}
			go func() {
				defer lifecycle.requestClose()
				defer lifecycle.finishHandler()
				defer func() {
					_ = recover()
				}()
				if answerer.OnSessionStart != nil {
					answerer.OnSessionStart()
				}
				dtlsFingerprint, fingerprintErr := LocalCertificateFingerprint(peer)
				if fingerprintErr == nil {
					fingerprintErr = answerer.Handler.ServeDataChannel(sessionCtx, protocolTransport, dtlsFingerprint)
				} else {
					_ = protocolTransport.Close()
				}
				if fingerprintErr != nil && sessionCtx.Err() == nil && answerer.OnSessionError != nil {
					answerer.OnSessionError(fingerprintErr)
				}
			}()
		})
	})
	if err := peer.SetRemoteDescription(pion.SessionDescription{Type: pion.SDPTypeOffer, SDP: offer.SDP}); err != nil {
		lifecycle.closeAndWait()
		return nil, fmt.Errorf("set remote daemon offer: %w", err)
	}
	for _, candidate := range offer.Candidates {
		if err := peer.AddICECandidate(toPionCandidate(candidate)); err != nil {
			lifecycle.closeAndWait()
			return nil, fmt.Errorf("add remote daemon ICE candidate: %w", err)
		}
	}
	localAnswer, err := peer.CreateAnswer(nil)
	if err != nil {
		lifecycle.closeAndWait()
		return nil, fmt.Errorf("create remote daemon answer: %w", err)
	}
	gatherComplete := pion.GatheringCompletePromise(peer)
	if err := peer.SetLocalDescription(localAnswer); err != nil {
		lifecycle.closeAndWait()
		return nil, fmt.Errorf("set remote daemon answer: %w", err)
	}
	timeout := directGatherTimeout
	if len(iceServers) > 0 {
		timeout = cloudGatherTimeout
	}
	if err := gathering.Wait(ctx, gatherComplete, timeout); err != nil {
		lifecycle.closeAndWait()
		return nil, err
	}
	description := peer.LocalDescription()
	if description == nil || strings.TrimSpace(description.SDP) == "" {
		lifecycle.closeAndWait()
		return nil, fmt.Errorf("remote daemon answer has no local description")
	}
	candidateMu.Lock()
	wireCandidates := append([]ICECandidate(nil), candidates...)
	candidateMu.Unlock()
	return &SignalingAnswer{SessionID: offer.SessionID, SDP: description.SDP, Candidates: wireCandidates, lifecycle: lifecycle}, nil
}

func toPionCandidate(candidate ICECandidate) pion.ICECandidateInit {
	lineIndex := uint16(candidate.SDPMLineIndex)
	result := pion.ICECandidateInit{Candidate: candidate.Candidate, SDPMLineIndex: &lineIndex}
	if candidate.SDPMid != "" {
		mid := candidate.SDPMid
		result.SDPMid = &mid
	}
	if candidate.UsernameFragment != "" {
		fragment := candidate.UsernameFragment
		result.UsernameFragment = &fragment
	}
	return result
}
