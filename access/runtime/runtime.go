package accessruntime

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/anytty/anytty/access/localstate"
	remote "github.com/anytty/anytty/access/remote"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// Runtime 是 access 进程唯一装配的 DeviceIdentity、AccessStore 与 local pairing socket。
// Local protocol、Direct/SSH、Cloud 都必须引用同一 Identity/Store，不能各自加载第二份授权真值。
type Runtime struct {
	Identity      remoteauth.Identity
	Store         *remoteauth.AccessStore
	PairingSocket string
}

// Load 加载或创建 access runtime，并取得 AccessStore 的唯一进程 owner lock。
// 第二个 owner、损坏 state、签名错误或 identity 不一致必须 fail closed。
func Load(socketPath string) (Runtime, error) {
	identity, err := remoteauth.LoadOrCreateLocalIdentity(localstate.RemoteIdentityDir())
	if err != nil {
		return Runtime{}, err
	}
	store, err := remoteauth.LoadAccessStore(localstate.RemoteAccessDir(), identity, remoteauth.AccessStoreOptions{})
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{Identity: identity, Store: store, PairingSocket: localstate.PairingSocketPath(socketPath)}, nil
}

// Close 释放 AccessStore 的唯一进程 owner lock。
// 调用方必须先停止 pairing listener、Direct/Cloud runtime 和 core session，再结束该 runtime。
func (rt Runtime) Close() error {
	if rt.Store == nil {
		return nil
	}
	return rt.Store.Close()
}

// StartPairingListener 启动 owner-only Unix PairingExchange listener。
// listener 只运行 remoteauth v2，不接受 anytty Hello 或 capability session；context 取消会关闭 socket。
func (rt Runtime) StartPairingListener(ctx context.Context, logger *slog.Logger) (func(), error) {
	if rt.Store == nil {
		return nil, fmt.Errorf("pairing listener requires the access store")
	}
	socket := rt.PairingSocket
	if socket == "" {
		return nil, fmt.Errorf("pairing socket path is required")
	}
	binding, err := remoteauth.LocalUnixChannelBinding(socket)
	if err != nil {
		return nil, err
	}
	listener, err := unixtransport.NewListener(socket)
	if err != nil {
		return nil, fmt.Errorf("listen for local PairingExchange: %w", err)
	}
	acceptor := remote.PairingAcceptor{Identity: rt.Identity, AccessStore: rt.Store}
	serveCtx, cancel := context.WithCancel(ctx)
	acceptDone := make(chan struct{})
	shutdownDone := make(chan struct{})
	var handlers sync.WaitGroup
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			cancel()
			_ = listener.Close()
			<-acceptDone
			handlers.Wait()
			close(shutdownDone)
		})
	}
	go func() {
		defer close(acceptDone)
		for {
			connection, acceptErr := listener.Accept(serveCtx)
			if acceptErr != nil {
				if serveCtx.Err() == nil && logger != nil {
					logger.Warn("local PairingExchange listener stopped", "error", acceptErr)
				}
				return
			}
			handlers.Add(1)
			go func(connection transport.Transport) {
				defer handlers.Done()
				if serveErr := acceptor.ServeBoundTransport(serveCtx, connection, binding); serveErr != nil && serveCtx.Err() == nil && logger != nil {
					logger.Debug("local PairingExchange rejected", "error", serveErr)
				}
			}(connection)
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
			shutdown()
		case <-shutdownDone:
		}
	}()
	return shutdown, nil
}
