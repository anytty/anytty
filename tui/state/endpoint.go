package state

import "strings"

const (
	// DefaultEndpointID 是当前单 daemon 本地路径的 endpoint 真值。
	// 旧 workbench snapshot、旧测试和 protocol adapter 只携带 TerminalID 时，都必须显式归入该 endpoint。
	DefaultEndpointID EndpointID = "local"
)

// EndpointID 是当前 TUI/client 侧识别 daemon endpoint 的稳定主键。
// 它属于客户端本地配置和 workbench 连接意图，不是展示名称，也不能替代 SSH host key、hub device id 或其他安全身份。
type EndpointID string

// EndpointConnectionPhase 是 TUI-owned 的 endpoint 连接阶段投影。
// 它只服务 reducer/view-model，不拥有 runtime 状态，也不绑定 Cloud、WebRTC 或 protocol concrete type。
type EndpointConnectionPhase string

const (
	EndpointConnectionIdle        EndpointConnectionPhase = "idle"
	EndpointConnectionResolving   EndpointConnectionPhase = "resolving"
	EndpointConnectionSignaling   EndpointConnectionPhase = "signaling"
	EndpointConnectionConnecting  EndpointConnectionPhase = "connecting"
	EndpointConnectionAuthorizing EndpointConnectionPhase = "authorizing"
	EndpointConnectionConnected   EndpointConnectionPhase = "connected"
	EndpointConnectionFailed      EndpointConnectionPhase = "failed"
)

// NormalizeEndpointID 把空 endpoint 映射到默认 local endpoint。
// 这是旧单 daemon 数据进入新多 endpoint 模型的唯一默认化入口，避免裸 TerminalID 在 reducer state 中保持无归属状态。
func NormalizeEndpointID(endpointID EndpointID) EndpointID {
	if strings.TrimSpace(string(endpointID)) == "" {
		return DefaultEndpointID
	}
	return EndpointID(strings.TrimSpace(string(endpointID)))
}

// TerminalRef 是 TUI/client 侧跨 endpoint 引用 terminal 的最小身份。
// TerminalID 仍是 owning daemon 内部的局部 ID；只有和 EndpointID 组合后，才能作为 reducer map key、owner 判定、picker selection 或 workbench binding 的真值。
type TerminalRef struct {
	EndpointID EndpointID `json:"endpointId,omitempty"`
	TerminalID string     `json:"terminalId"`
}

// NewTerminalRef 构造一个规范化的跨 endpoint terminal 引用。
// 调用边界允许 endpoint 为空以兼容旧本地路径，但返回值一定带有明确 endpoint。
func NewTerminalRef(endpointID EndpointID, terminalID string) TerminalRef {
	return TerminalRef{EndpointID: NormalizeEndpointID(endpointID), TerminalID: terminalID}
}

// LocalTerminalRef 把旧单 daemon 调用中的裸 TerminalID 显式归入默认 local endpoint。
// 该函数只用于兼容旧 API；新增跨 endpoint 路径应直接传递完整 TerminalRef。
func LocalTerminalRef(terminalID string) TerminalRef {
	return NewTerminalRef(DefaultEndpointID, terminalID)
}

// Normalize 返回带有默认 endpoint 的 TerminalRef 副本。
// 它不改写 TerminalID，因为 TerminalID 的合法字符和空值语义由 owning daemon/protocol 边界决定。
func (ref TerminalRef) Normalize() TerminalRef {
	return NewTerminalRef(ref.EndpointID, ref.TerminalID)
}

// Empty 表示该引用没有 daemon-local terminal id，不能用于路由或状态匹配。
func (ref TerminalRef) Empty() bool {
	return ref.TerminalID == ""
}

// Equal 按规范化后的 endpoint 和 terminal 双字段比较。
// 任何跨 endpoint reducer 判定都必须用这个语义，不能只比较 TerminalID。
func (ref TerminalRef) Equal(other TerminalRef) bool {
	ref = ref.Normalize()
	other = other.Normalize()
	return ref.EndpointID == other.EndpointID && ref.TerminalID == other.TerminalID
}

// Key 返回适合作为 reducer 内部 map key 的稳定字符串。
// 该 key 只表达客户端本地 routing identity，不应暴露为安全身份或用户可编辑名称。
func (ref TerminalRef) Key() string {
	ref = ref.Normalize()
	if ref.TerminalID == "" {
		return ""
	}
	return string(ref.EndpointID) + "/" + ref.TerminalID
}
