package local

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// Starter 是 local Unix route 首次拨号失败后的 daemon 启动 primitive。
// 具体可执行文件、日志和 config 参数仍由 composition root 决定；adapter 只控制一次 route attempt 的重试生命周期。
type Starter func(context.Context, string) error

// ProtocolClient 是 local adapter 返回的 framing client 类型别名。
// 别名只用于迁移 composition 调用签名，业务调用仍应优先使用 owner-fenced ApplicationClient。
type ProtocolClient = internalprotocol.Client

// Transport 是 local Unix transport 类型别名，仅供 daemon composition/harness 使用。
type Transport = unixtransport.Transport

// Options 定义 local Unix route adapter 的平台无关拨号参数。
type Options struct {
	SocketOverride string
	DefaultSocket  string
	ClientName     string
	Start          Starter
	ReadyTimeout   time.Duration
	RetryInterval  time.Duration
}

// Dialer 只执行 AttemptRequest 已选定的 local Unix route，不选择其它 route 或生成 generation。
type Dialer struct {
	options Options
}

// NewDialer 创建 local Unix route dialer。
func NewDialer(options Options) *Dialer {
	if strings.TrimSpace(options.ClientName) == "" {
		options.ClientName = "anytty-client"
	}
	if options.ReadyTimeout <= 0 {
		options.ReadyTimeout = 5 * time.Second
	}
	if options.RetryInterval <= 0 {
		options.RetryInterval = 25 * time.Millisecond
	}
	return &Dialer{options: options}
}

// Dial 建立 Unix transport、完成 protocol Hello，并返回与 attempt stamp 严格匹配的 ready session。
func (dialer *Dialer) Connect(ctx context.Context, request clientruntime.AttemptRequest) (clientruntime.ReadyPeerSession, error) {
	route := request.Route()
	if route.Kind != endpoint.RouteLocalUnix {
		return nil, fmt.Errorf("local adapter cannot dial route kind %s", route.Kind)
	}
	path := strings.TrimSpace(dialer.options.SocketOverride)
	if path == "" {
		path = strings.TrimSpace(route.Socket)
	}
	if path == "" || path == "auto" {
		path = strings.TrimSpace(dialer.options.DefaultSocket)
	}
	if path == "" {
		return nil, fmt.Errorf("local Unix socket path is required")
	}
	client, err := dialProtocol(ctx, path, dialer.options.ClientName)
	var handshakeErr *protocolHandshakeError
	if err != nil && dialer.options.Start != nil && !errors.As(err, &handshakeErr) {
		if startErr := dialer.options.Start(ctx, path); startErr != nil {
			return nil, fmt.Errorf("start local daemon: %w", startErr)
		}
		waitCtx, cancel := context.WithTimeout(ctx, dialer.options.ReadyTimeout)
		defer cancel()
		for err != nil && waitCtx.Err() == nil {
			timer := time.NewTimer(dialer.options.RetryInterval)
			select {
			case <-waitCtx.Done():
				timer.Stop()
			case <-timer.C:
			}
			if waitCtx.Err() == nil {
				client, err = dialProtocol(waitCtx, path, dialer.options.ClientName)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	ready, err := protocoladapter.NewApplicationClientWithObservedPath(client, request.Stamp(), "unix:"+path)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	identity, err := protocoladapter.VerifyDaemonIdentity(ctx, ready.ApplicationSession, request.DaemonIdentity())
	if err != nil {
		_ = ready.Close()
		return nil, err
	}
	if err := ready.MarkReady(clientruntime.ReadyPeerSessionEvidence{
		Identity: identity, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version,
	}); err != nil {
		_ = ready.Close()
		return nil, err
	}
	return ready, nil
}

type protocolHandshakeError struct{ cause error }

func (err *protocolHandshakeError) Error() string { return err.cause.Error() }
func (err *protocolHandshakeError) Unwrap() error { return err.cause }

func dialProtocol(ctx context.Context, path, clientName string) (*internalprotocol.Client, error) {
	transport, err := unixtransport.DialContext(ctx, path)
	if err != nil {
		return nil, err
	}
	client := internalprotocol.NewClient(transport)
	if err := client.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: clientName}); err != nil {
		_ = client.Close()
		return nil, &protocolHandshakeError{cause: err}
	}
	return client, nil
}

// DialProtocolClientForComposition 建立单条 local Unix framing connection 并完成 Hello。
// 该迁移入口只供 daemon/CLI composition 随后交给 SessionOwner adopt；普通 application consumer 必须直接使用 Connect。
func DialProtocolClientForComposition(ctx context.Context, path, clientName string) (*ProtocolClient, error) {
	return dialProtocol(ctx, path, clientName)
}

// DialTransport 建立不执行 Hello 的 local Unix transport，供 daemon lifecycle harness 使用。
func DialTransport(ctx context.Context, path string) (*Transport, error) {
	return unixtransport.DialContext(ctx, path)
}

var _ clientruntime.PeerConnector = (*Dialer)(nil)
