package webrtc

import (
	"log/slog"
	"net"
	"net/url"
	"strings"

	"github.com/anytty/anytty/proto/wire"
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
	}
	return newPeerConnectionAPI(settingEngine).NewPeerConnection(configuration)
}

func newPeerConnectionAPI(settingEngine pion.SettingEngine) *pion.API {
	EnsureLoggerFactory(&settingEngine, nil)
	settingEngine.SetSCTPMaxMessageSize(wire.MaxEncodedFrameSize)
	return pion.NewAPI(pion.WithSettingEngine(settingEngine))
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
