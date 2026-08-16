package port

import (
	"context"
	"time"

	"github.com/anytty/anytty/client/endpoint"
)

// ICECandidate 是 WebRTC 信令边界使用的中性 ICE candidate。
// 它只承载 Pion/平台 peer 所需的 SDP 字段，不拥有 Route、Cloud 票据或连接生命周期。
type ICECandidate struct {
	Candidate        string
	SDPMid           string
	SDPMLineIndex    uint32
	UsernameFragment string
}

// ICEServer 是 WebRTC primitive 的中性 STUN/TURN 参数；credential 来源和租约验证属于上层 adapter。
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

// ICETransportPolicy 约束当前 peer 是否允许 direct candidate 或必须只使用 Relay candidate。
type ICETransportPolicy uint8

const (
	ICETransportAll ICETransportPolicy = iota
	ICETransportRelayOnly
)

// WebRTCConfig 是创建单个 peer 所需的 Cloud 无关 ICE 配置，不拥有 Route 或 session lifecycle。
type WebRTCConfig struct {
	Servers []ICEServer
	Policy  ICETransportPolicy
}

// WebRTCMessageChannel 是 Go Client Engine 对可靠有序 WebRTC DataChannel 的最小平台要求。
// native Pion 与浏览器 adapter 都必须复制收到的 bytes，并实现显式背压和关闭；该接口不解释 remote-auth 或 Proto API payload。
type WebRTCMessageChannel interface {
	// SetMessageHandler 注册完整 DataChannel message 接收器；实现必须在回调前复制平台可复用 buffer。
	SetMessageHandler(func([]byte))
	// SetCloseHandler 注册底层 channel 生命周期终止通知；回调不得自行创建新 session。
	SetCloseHandler(func())
	// BufferedAmount 返回平台当前尚未发送的字节数，只用于 transport 背压。
	BufferedAmount() uint64
	// SetBufferedAmountLowThreshold 设置解除发送等待的低水位。
	SetBufferedAmountLowThreshold(uint64)
	// SetBufferedAmountLowHandler 注册低水位通知，允许重复或合并通知但不得丢失 channel close。
	SetBufferedAmountLowHandler(func())
	// Send 发送一个可靠有序 message；失败必须显式返回，不能静默丢帧或 fallback。
	Send([]byte) error
	// Close 幂等关闭当前 channel，不修改 endpoint registry 或 session generation。
	Close() error
}

// WebRTCPeer 是 native Pion 与浏览器 RTCPeerConnection 共同实现的平台 primitive。
// Route adapter 拥有 signaling/auth/protocol 顺序；实现只能执行指定 SDP/ICE 操作并报告实际 DTLS/path 证据。
type WebRTCPeer interface {
	// Channel 返回创建 peer 时同步建立的可靠有序 protocol DataChannel。
	Channel() WebRTCMessageChannel
	// CreateOffer 生成完成本地 candidate gathering 的 SDP offer。
	CreateOffer(context.Context) (string, error)
	// ApplyAnswer 应用受信 signaling 返回的 SDP answer 与 remote candidates。
	ApplyAnswer(context.Context, string, []ICECandidate) error
	// WaitReady 等待 DataChannel 可发送；context 取消必须停止等待并由调用方关闭 peer。
	WaitReady(context.Context) error
	// RemoteCertificateFingerprint 返回实际 DTLS peer certificate 的规范化 SHA-256 fingerprint。
	RemoteCertificateFingerprint() (string, error)
	// ObservedPath 返回当前 selected candidate pair 的 direct/single-relay 投影。
	ObservedPath() endpoint.Path
	// Snapshot 返回当前 selected candidate pair 的地址与质量计数；无稳定 candidate pair 时返回 false。
	Snapshot(time.Time) (WebRTCPeerSnapshot, bool)
	// Close 幂等释放 peer、ICE、DTLS、SCTP 和 DataChannel 平台资源。
	Close() error
}

// WebRTCPeerSnapshot 是质量观测读取的平台网络计数快照。
// 它只包含 selected pair 的 IP/port，不包含 SDP、credential、endpoint label 或 terminal identity；计数回退表示底层 candidate pair 已换代。
type WebRTCPeerSnapshot struct {
	PairID               string
	Path                 endpoint.Path
	NetworkClass         string
	At                   time.Time
	RoundTrip            time.Duration
	BytesSent            uint64
	BytesRecv            uint64
	PacketsSent          uint64
	LossEvents           uint64
	Connected            bool
	LocalCandidateType   string
	RemoteCandidateType  string
	LocalAddress         string
	RemoteAddress        string
	LocalPort            uint16
	RemotePort           uint16
	LocalRelatedAddress  string
	RemoteRelatedAddress string
	LocalRelatedPort     uint16
	RemoteRelatedPort    uint16
	LocalProtocol        string
	RemoteProtocol       string
	RelayProtocol        string
}
