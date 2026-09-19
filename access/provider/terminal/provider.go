// Package terminal 定义 access 路由层使用的 terminal provider 契约。
//
// access 终结客户端 access wire 后，把终端能力路由给一个 Provider。当前实现是
// pool provider（owner-only 本地 socket）；tmux/zellij 等翻译型 provider 以后
// 实现同一契约。Provider 不持有 DeviceIdentity/AccessStore，也不做鉴权：调用方
// （access/server）已经完成本地信任边界或 remoteauth。
//
// 领域 DTO 说明：provider 方法直接运输 generated Proto command/result，
// Proto 与 core-native 类型的转换仍由 api_mapping 承担。这样 access 路由层不会
// 建立第二套 DTO 真值，也不会在桥接时丢字段；Provider 接口只约束能力边界。
package terminal

import (
	"context"
	"errors"

	"github.com/anytty/anytty/proto/access/apipb"
)

// ErrUnsupported 表示 provider 不支持该能力（例如未实现的 tmux provider）。
// access/server 必须把它映射为 typed unavailable，而不是内部错误。
var ErrUnsupported = errors.New("terminal provider capability is unsupported")

// DialFunc 为每个客户端会话建立一个 Provider。conn 生命周期与返回的
// Provider 一致：会话结束时由 access/server 调用 Close。
type DialFunc func(ctx context.Context) (Provider, error)

// Stream 是一个 provider session 上已经打开的 stream resource（attachment、
// file transfer 或 browser proxy）。Receive 必须响应 context 取消；Send 只允许
// 该 resource kind 对应的 frame type；Close 必须幂等。
type Stream interface {
	Receive(ctx context.Context) (typ uint8, payload []byte, err error)
	Send(ctx context.Context, typ uint8, payload []byte) error
	Close() error
}

// Provider 是一条 provider 会话。实现由 DialFactory 为每个客户端会话建立，
// 会话生命周期与客户端连接一致；一个 Provider 只服务一个 access client session。
type Provider interface {
	// Execute 运输一个完整 generated Proto command，并返回同 correlation 的 result。
	// 它不解释 command oneof 的领域语义，也不重写 request/session stamp。
	Execute(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error)
	// OpenStream 在 provider session 上打开一个由 Execute 结果发布的 stream resource。
	// attachment 会完成 bootstrap/ready 握手；file/browser 直接绑定 channel。
	OpenStream(ctx context.Context, resource *apipb.ResourceHandle) (Stream, error)
	// Events 返回该 provider session 的 application event 流。
	Events(ctx context.Context) (<-chan *apipb.EventEnvelope, error)
	// Done 在 provider 连接终止时关闭；Err 返回终止原因。
	Done() <-chan struct{}
	Err() error
	// Close 释放 provider 连接；必须幂等。
	Close() error
}
