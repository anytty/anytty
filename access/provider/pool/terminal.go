package pool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/internal/providerproto"
	poolprovider "github.com/anytty/anytty/pool/provider"
)

// TerminalDialTimeout bounds one provider session establishment (dial + Hello).
const TerminalDialTimeout = 5 * time.Second

// TerminalProvider adapts the access terminal routing to the pool provider
// protocol. access/server keeps owning client channels/tokens and stream
// bridging; this adapter translates the typed provider contract to pool
// provider calls (see typed.go).
type TerminalProvider struct {
	socket string
	client *poolprovider.Client

	cancel context.CancelFunc
}

var _ terminalprovider.Provider = (*TerminalProvider)(nil)

// DialTerminal 建立一条 pool provider 会话：owner-only unix dial + provider Hello。
func DialTerminal(ctx context.Context, socket string) (*TerminalProvider, error) {
	path := strings.TrimSpace(socket)
	if path == "" {
		return nil, errors.New("provider/pool: socket path is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, TerminalDialTimeout)
	defer cancelDial()
	client, err := poolprovider.Dial(dialCtx, path)
	if err != nil {
		return nil, fmt.Errorf("provider/pool: %w", err)
	}
	_, cancel := context.WithCancel(context.Background())
	provider := &TerminalProvider{
		socket: path,
		client: client,
		cancel: cancel,
	}
	return provider, nil
}

// Socket 返回 provider socket 路径（诊断用）。
func (provider *TerminalProvider) Socket() string {
	if provider == nil {
		return ""
	}
	return provider.socket
}

// Done 在 provider 连接终止时关闭。
func (provider *TerminalProvider) Done() <-chan struct{} {
	if provider == nil || provider.client == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return provider.client.Done()
}

// Err 返回 provider 连接终止原因。
func (provider *TerminalProvider) Err() error {
	if provider == nil || provider.client == nil {
		return errors.New("provider/pool: provider is closed")
	}
	return provider.client.Err()
}

// Close 释放 provider 连接。幂等。
func (provider *TerminalProvider) Close() error {
	if provider == nil {
		return nil
	}
	if provider.cancel != nil {
		provider.cancel()
	}
	if provider.client == nil {
		return nil
	}
	return provider.client.Close()
}

const attachmentStreamReadyTimeout = 5 * time.Second

// awaitStreamReady 等待 provider attachment channel 完成 ready 握手；
// typed Duplex（typed.go）用它隐藏 bootstrap/ready 流控帧。
func awaitStreamReady(frames <-chan poolprovider.StreamFrame, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return errors.New("provider/pool: terminal attachment stream ready timed out")
		case frame, ok := <-frames:
			if !ok {
				return errors.New("provider/pool: terminal attachment stream closed before ready")
			}
			if frame.Type == providerproto.StreamReady {
				return nil
			}
			if frame.Type == providerproto.StreamClosed {
				return errors.New("provider/pool: terminal attachment stream closed before ready")
			}
		}
	}
}
