// Package tmux 是 terminal provider 接口的 tmux 占位实现。
//
// 本期只落接口与错误占位：tmux/zellij 需要把 provider 契约翻译成 control mode
// 与 pane 流，尚未实现。所有能力都返回 terminal.ErrUnsupported，access/server
// 会把它映射为 typed unavailable，不影响 auth/文件/转发路由。
package tmux

import (
	"context"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
)

// Provider 是未经实现的 tmux terminal provider。
type Provider struct{}

var _ terminalprovider.Provider = Provider{}

// Unsupported 返回占位 provider；保留构造函数便于以后替换为真实实现。
func Unsupported() Provider {
	return Provider{}
}

// Info 返回 tmux provider 身份；Capabilities 全为 false。
func (Provider) Info(context.Context) (terminalprovider.Info, error) {
	return terminalprovider.Info{Kind: "tmux"}, nil
}

// Capabilities 声明 tmux 占位实现不支持任何能力。
func (Provider) Capabilities() terminalprovider.Capabilities {
	return terminalprovider.Capabilities{}
}

// Create 永远返回 terminal.ErrUnsupported。
func (Provider) Create(context.Context, *providerv1.TerminalCreateSpec) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

// List 永远返回 terminal.ErrUnsupported。
func (Provider) List(context.Context) ([]*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Get 永远返回 terminal.ErrUnsupported。
func (Provider) Get(context.Context, string) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Restart 永远返回 terminal.ErrUnsupported。
func (Provider) Restart(context.Context, string) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Kill 永远返回 terminal.ErrUnsupported。
func (Provider) Kill(context.Context, string) error {
	return terminalprovider.ErrUnsupported
}

// Remove 永远返回 terminal.ErrUnsupported。
func (Provider) Remove(context.Context, string) error {
	return terminalprovider.ErrUnsupported
}

// SetMetadata 永远返回 terminal.ErrUnsupported。
func (Provider) SetMetadata(context.Context, string, terminalprovider.MetadataPatch) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

// SetTags 永远返回 terminal.ErrUnsupported。
func (Provider) SetTags(context.Context, string, terminalprovider.TagsPatch) (*providerv1.TerminalInfo, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Attach 永远返回 terminal.ErrUnsupported。
func (Provider) Attach(context.Context, terminalprovider.AttachRequest) (terminalprovider.Attachment, error) {
	return nil, terminalprovider.ErrUnsupported
}

// HistoryWindow 永远返回 terminal.ErrUnsupported。
func (Provider) HistoryWindow(context.Context, terminalprovider.HistoryRequest) (*providerv1.HistoryWindowResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

// HistoryCopy 永远返回 terminal.ErrUnsupported。
func (Provider) HistoryCopy(context.Context, terminalprovider.HistoryCopyRequest) (*providerv1.HistoryCopyResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

// HistorySearch 永远返回 terminal.ErrUnsupported。
func (Provider) HistorySearch(context.Context, terminalprovider.HistorySearchRequest) (*providerv1.HistorySearchResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

// HistoryRelease 永远返回 terminal.ErrUnsupported。
func (Provider) HistoryRelease(context.Context, string, string) error {
	return terminalprovider.ErrUnsupported
}

// HistoryBacklogStatus 永远返回 terminal.ErrUnsupported。
func (Provider) HistoryBacklogStatus(context.Context, string) (*providerv1.HistoryBacklogStatusResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

// LiveScreen 永远返回 terminal.ErrUnsupported。
func (Provider) LiveScreen(context.Context, string, uint64) (*providerv1.NativeScreenResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Subscribe 永远返回 terminal.ErrUnsupported。
func (Provider) Subscribe(context.Context, *providerv1.EventSubscribeCommand) (*providerv1.EventSubscriptionResult, error) {
	return nil, terminalprovider.ErrUnsupported
}

// Events 永远返回 terminal.ErrUnsupported。
func (Provider) Events(context.Context) (<-chan *providerv1.TerminalEvent, error) {
	return nil, terminalprovider.ErrUnsupported
}

// EventRelease 永远返回 terminal.ErrUnsupported。
func (Provider) EventRelease(context.Context, []byte) error {
	return terminalprovider.ErrUnsupported
}

// TerminalDefaults 永远返回 terminal.ErrUnsupported。
func (Provider) TerminalDefaults(context.Context) (*providerv1.TerminalDefaults, error) {
	return nil, terminalprovider.ErrUnsupported
}

// ListDirectories 永远返回 terminal.ErrUnsupported。
func (Provider) ListDirectories(context.Context, string, int32) (*providerv1.PathListDirectoriesResult, error) {
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
