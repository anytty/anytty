package webrtc

import (
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/netpath"
	pion "github.com/pion/webrtc/v4"
)

// PeerConnectionFactory 创建单个 Pion PeerConnection。
// 默认生产路径使用 NewPeerConnection；测试或后续 direct transport 可以注入限定网络类型和 mux 的独立 Pion API，但 factory 不拥有 signaling、auth 或 session lifecycle。
type PeerConnectionFactory func(pion.Configuration) (*pion.PeerConnection, error)

// NewPeerConnection 按受信 ICE 配置创建公开 WebRTC primitive。
// 只有显式 loopback TURN URL 会启用 loopback candidate，用于本地开发云或自托管 harness；普通公网配置保持 Pion 默认网络边界。
func NewPeerConnection(configuration pion.Configuration) (*pion.PeerConnection, error) {
	return NewPeerConnectionWithLogger(configuration, nil)
}

// NewPeerConnectionWithLogger creates a public WebRTC primitive whose Pion
// diagnostics are owned by the supplied process logger instead of stderr.
func NewPeerConnectionWithLogger(configuration pion.Configuration, logger *slog.Logger) (*pion.PeerConnection, error) {
	settingEngine := pion.SettingEngine{}
	EnsureLoggerFactory(&settingEngine, logger)
	if containsLoopbackTURN(configuration.ICEServers) {
		settingEngine.SetIncludeLoopbackCandidate(true)
	} else if network, err := netpath.NewICENetwork(); err == nil {
		settingEngine.SetNet(network)
	}
	return newPeerConnectionAPI(settingEngine).NewPeerConnection(configuration)
}

func newPeerConnectionAPI(settingEngine pion.SettingEngine) *pion.API {
	EnsureLoggerFactory(&settingEngine, nil)
	settingEngine.SetSCTPMaxMessageSize(wire.MaxEncodedFrameSize)
	// SCTP 默认 cwnd 从 0 起步、RTO 退避到 60s：丢包后恢复要按 RTT 翻倍很多轮。
	// 交互式终端更需要「尽快恢复」而不是「最大化吞吐」，这里收窄两个窗口。
	settingEngine.SetSCTPMinCwnd(19200)
	settingEngine.SetSCTPRTOMax(2 * time.Second)
	return pion.NewAPI(pion.WithSettingEngine(settingEngine))
}

// NewAnswererPeerConnection 是 daemon 侧（answering）的默认工厂。
//
// 相比客户端它额外放宽接收缓冲（多条 request/文件/屏幕共享同一 association），
// 并把 ICE 恢复窗口从默认 5s/25s 收紧到 4s/8s：对端已经消失时不要让用户盯着
// 一个半死连接。
func NewAnswererPeerConnection(configuration pion.Configuration, logger *slog.Logger) (*pion.PeerConnection, error) {
	settingEngine := pion.SettingEngine{}
	EnsureLoggerFactory(&settingEngine, logger)
	if containsLoopbackTURN(configuration.ICEServers) {
		settingEngine.SetIncludeLoopbackCandidate(true)
	} else if network, err := netpath.NewICENetwork(); err == nil {
		settingEngine.SetNet(network)
	}
	settingEngine.SetSCTPMaxReceiveBufferSize(3 << 20)
	settingEngine.SetICETimeouts(4*time.Second, 8*time.Second, time.Second)
	return newPeerConnectionAPI(settingEngine).NewPeerConnection(configuration)
}

func containsLoopbackTURN(servers []pion.ICEServer) bool {
	for _, server := range servers {
		for _, rawURL := range server.URLs {
			parsed, err := url.Parse(strings.TrimSpace(rawURL))
			if err != nil || !strings.EqualFold(parsed.Scheme, "turn") && !strings.EqualFold(parsed.Scheme, "turns") {
				continue
			}
			authority := parsed.Host
			if authority == "" {
				authority = parsed.Opaque
			}
			authority, _, _ = strings.Cut(authority, "?")
			host, _, err := net.SplitHostPort(authority)
			if err == nil {
				if strings.EqualFold(host, "localhost") {
					return true
				}
				if address := net.ParseIP(host); address != nil && address.IsLoopback() {
					return true
				}
			}
		}
	}
	return false
}
