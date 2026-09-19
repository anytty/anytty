package direct

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	remotev2webrtc "github.com/anytty/anytty/access/transport/webrtc"
	"github.com/anytty/anytty/shared/remoteauth"
)

// Options 描述一次 Direct listener 启动。
// Identity 与 Handler 必填；地址为空时回退 ConfiguredAddresses()。
type Options struct {
	Identity        remoteauth.Identity
	Handler         remotev2webrtc.DataChannelSessionHandler
	SignalingListen string
	ICETCPListen    string
	// RecordPath 非空时，Start 会发布 owner-only listener 记录供 CLI pairing 默认地址读取。
	RecordPath string
	Logger     *slog.Logger
	Now        func() time.Time
}

// Server 是 access 进程中 Direct signaling 与共享 ICE-TCP mux 的生命周期 owner。
type Server struct {
	server            *remotev2webrtc.DirectServer
	signalingListener net.Listener
	iceListener       net.Listener
	closeDiscovery    func()
	signaling         string
	ice               string
	recordPath        string
	done              chan struct{}
}

// Start 绑定 Direct listener 并启动 serve。调用方必须 Close 释放 listener 与 discovery。
func Start(ctx context.Context, options Options) (*Server, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := options.Identity.Validate(); err != nil {
		return nil, fmt.Errorf("Direct server identity: %w", err)
	}
	if options.Handler == nil {
		return nil, fmt.Errorf("Direct server handler is required")
	}
	signalingAddress := strings.TrimSpace(options.SignalingListen)
	iceAddress := strings.TrimSpace(options.ICETCPListen)
	if signalingAddress == "" || iceAddress == "" {
		configuredSignaling, configuredICE := ConfiguredAddresses()
		if signalingAddress == "" {
			signalingAddress = configuredSignaling
		}
		if iceAddress == "" {
			iceAddress = configuredICE
		}
	}
	rootListener, err := net.Listen("tcp", signalingAddress)
	if err != nil {
		return nil, fmt.Errorf("listen Direct signaling %q: %w", signalingAddress, err)
	}
	signalingListener := rootListener
	var iceListener net.Listener
	if signalingAddress == iceAddress {
		signalingListener, iceListener, err = remotev2webrtc.SplitDirectListener(rootListener)
		if err != nil {
			_ = rootListener.Close()
			return nil, fmt.Errorf("split Direct signaling and ICE-TCP listener: %w", err)
		}
	} else {
		iceListener, err = net.Listen("tcp", iceAddress)
		if err != nil {
			_ = signalingListener.Close()
			return nil, fmt.Errorf("listen Direct ICE-TCP %q: %w", iceAddress, err)
		}
	}
	server, err := remotev2webrtc.NewDirectServer(options.Identity, options.Handler, signalingListener, iceListener, options.Now, remotev2webrtc.WithPionLogger(options.Logger))
	if err != nil {
		_ = signalingListener.Close()
		_ = iceListener.Close()
		return nil, err
	}
	actualSignaling := signalingListener.Addr().String()
	actualICE := iceListener.Addr().String()
	closeDiscovery := func() {}
	if signalingAddress == iceAddress {
		var discoveryErr error
		closeDiscovery, discoveryErr = startDirectDiscovery(ctx, options.Identity.DeviceID, options.Identity.Fingerprint, signalingAddress, actualSignaling)
		if closeDiscovery == nil {
			closeDiscovery = func() {}
		}
		if discoveryErr != nil && options.Logger != nil {
			options.Logger.Warn("Direct LAN discovery is unavailable", "error", discoveryErr)
		}
	}
	setRuntimeAddresses(actualSignaling, actualICE)
	recordPath := strings.TrimSpace(options.RecordPath)
	if recordPath != "" {
		if recordErr := WriteListenerRecord(recordPath, ListenerRecord{
			Listen: signalingAddress, Signaling: actualSignaling, ICETCP: actualICE, UpdatedAt: time.Now().UTC(),
		}); recordErr != nil && options.Logger != nil {
			options.Logger.Warn("Direct listener record could not be published", "error", recordErr)
		}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if serveErr := server.Serve(ctx); serveErr != nil && ctx.Err() == nil && options.Logger != nil {
			options.Logger.Error("Direct WebRTC server stopped", "error", serveErr)
		}
	}()
	if options.Logger != nil {
		options.Logger.Info("Direct WebRTC server listening", "signaling", actualSignaling, "ice_tcp", actualICE)
	}
	return &Server{
		server: server, signalingListener: signalingListener, iceListener: iceListener,
		closeDiscovery: closeDiscovery, signaling: actualSignaling, ice: actualICE, recordPath: recordPath, done: done,
	}, nil
}

// Addresses 返回实际绑定的 signaling/ICE-TCP 地址。
func (server *Server) Addresses() (string, string) {
	if server == nil {
		return "", ""
	}
	return server.signaling, server.ice
}

// Close 关闭 discovery、listener 与所有已建立 peer，并等待 Serve 退出。
func (server *Server) Close() {
	if server == nil {
		return
	}
	server.closeDiscovery()
	_ = server.server.Close()
	<-server.done
	if server.recordPath != "" {
		_ = RemoveListenerRecord(server.recordPath)
	}
	clearRuntimeAddresses(server.signaling, server.ice)
}
