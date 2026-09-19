package core

import (
	"context"
	"io"
)

// Provider-facing terminal service hooks.
//
// daemon/provider 只通过这里暴露的窄接口访问 terminal core；它不读取 core
// 未导出状态，也不复制 terminal truth。Provider 协议见 proto/provider/v1。

// TerminalDefaultsSnapshot 是 owning daemon 机器的 shell/cwd 默认值快照。
func (server *Server) TerminalDefaultsSnapshot() TerminalDefaults {
	return pathDefaults()
}

// ListPathDirectories 返回 daemon 主机上的 path completion 窗口。
func (server *Server) ListPathDirectories(prefix string, limit int) (PathDirectories, error) {
	return listPathDirectories(prefix, limit)
}

// RawPTYSubscription 是 provider attachment 流的有界、有序 PTY 输出订阅。
// Receive 必须在 ctx 取消时返回；Termination 报告 dropped bytes 与 exit code。
type RawPTYSubscription struct {
	inner *rawPTYSubscription
}

// SubscribeRawPTY 订阅当前 terminal 进程代的原始 PTY 输出。
func (terminal *Terminal) SubscribeRawPTY(ctx context.Context) *RawPTYSubscription {
	return &RawPTYSubscription{inner: terminal.subscribeRawPTY(ctx)}
}

// Receive 返回下一个 PTY output chunk。
func (subscription *RawPTYSubscription) Receive(ctx context.Context) ([]byte, error) {
	if subscription == nil {
		return nil, io.EOF
	}
	return subscription.inner.receive(ctx)
}

// Termination 返回 dropped bytes 与（可选的）进程退出码。
func (subscription *RawPTYSubscription) Termination() (uint64, *int) {
	if subscription == nil {
		return 0, nil
	}
	return subscription.inner.termination()
}

// NativeScreenBaseline 是 provider live-screen delta 的 opaque baseline。
// 它只由 Server.NextLiveScreenWithBaseline 创建与消费，不暴露 terminal internals。
type NativeScreenBaseline struct {
	inner *nativeScreenBaseline
}

// Revision 返回 baseline 对应的 native screen revision。
func (baseline *NativeScreenBaseline) Revision() LiveRevision {
	if baseline == nil || baseline.inner == nil {
		return 0
	}
	return baseline.inner.revision
}

// Bytes 返回 baseline 的近似内存占用，供 provider session 预算裁剪。
func (baseline *NativeScreenBaseline) Bytes() int64 {
	if baseline == nil || baseline.inner == nil {
		return 0
	}
	return int64(len(baseline.inner.rowHashes)) * 8
}

// NextLiveScreenWithBaseline 返回 observed revision 之后的 native screen delta，
// 并返回可继续传递的 opaque baseline。
func (server *Server) NextLiveScreenWithBaseline(ctx context.Context, id string, observed LiveRevision, base *NativeScreenBaseline) (NativeScreenSnapshot, *NativeScreenBaseline, error) {
	terminal, err := server.Terminal(id)
	if err != nil {
		return NativeScreenSnapshot{}, nil, err
	}
	terminal.queueMu.Lock()
	liveErr := terminal.liveOutputError
	if liveErr == nil && terminal.outputBuffer != nil {
		liveErr = terminal.outputBuffer.ConsumerError(terminalOutputConsumerLive)
	}
	terminal.queueMu.Unlock()
	if liveErr != nil {
		return NativeScreenSnapshot{}, nil, liveErr
	}
	var inner *nativeScreenBaseline
	if base != nil {
		inner = base.inner
	}
	snapshot, next, err := server.nextLiveScreenWithBaseline(ctx, id, observed, inner)
	if err != nil {
		return NativeScreenSnapshot{}, nil, err
	}
	if next == nil {
		return snapshot, nil, nil
	}
	return snapshot, &NativeScreenBaseline{inner: next}, nil
}

// Start 启动 terminal server 的后台服务（history retention）并发布 listening
// 事件，但不绑定客户端协议 listener：.provider 由 provider server 绑定。
func (server *Server) Start(ctx context.Context) error {
	_ = ctx
	if server == nil || server.closed.Load() {
		return ErrServerClosed
	}
	server.events.publish(Event{Type: EventServerListening, SocketPath: server.cfg.socketPath})
	server.cfg.logger.Info("core-v2 server started", "socket_path", server.cfg.socketPath)
	server.startHistoryRetention()
	return nil
}
