// Package tmux 是 terminal provider 接口的 tmux 占位实现。
//
// 本期只落接口与错误占位：tmux/zellij 需要把 provider 契约翻译成 control mode
// 与 pane 流，尚未实现。所有能力都返回 terminal.ErrUnsupported，access/server
// 会把它映射为 typed unavailable，不影响 auth/文件/转发路由。
package tmux

import (
	"context"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/proto/access/apipb"
)

// Provider 是未经实现的 tmux terminal provider。
type Provider struct{}

var _ terminalprovider.Provider = Provider{}

// Unsupported 返回占位 provider；保留构造函数便于以后替换为真实实现。
func Unsupported() Provider {
	return Provider{}
}

// Execute 永远返回 terminal.ErrUnsupported。
func (Provider) Execute(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return nil, terminalprovider.ErrUnsupported
}

// OpenStream 永远返回 terminal.ErrUnsupported。
func (Provider) OpenStream(context.Context, *apipb.ResourceHandle) (terminalprovider.Stream, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Events 永远返回 terminal.ErrUnsupported。
func (Provider) Events(context.Context) (<-chan *apipb.EventEnvelope, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Done 返回已经关闭的信号。
func (Provider) Done() <-chan struct{} {
	closed := make(chan struct{})
	close(closed)
	return closed
}

// Err 返回 terminal.ErrUnsupported。
func (Provider) Err() error {
	return terminalprovider.ErrUnsupported
}

// Close 幂等。
func (Provider) Close() error {
	return nil
}
