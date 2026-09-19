// Package server 是 access 的协议服务器：终结客户端 access wire，并按能力路由。
//
// 本包拥有单连接事件循环、Hello、请求预算与 stream channel registry；
// 终端面转发给配置的 terminal provider（当前 = daemon），文件/转发/身份域在
// 后续阶段陆续在 access 本地终结。它不做鉴权：本地 unix 连接信任 0600 socket，
// 远程连接必须先经过 remoteauth。
package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/access/files"
	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/access/proxy"
	"github.com/anytty/anytty/access/sessions"
	"github.com/anytty/anytty/access/storage"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// ModuleName 是 access protocol server 的稳定标识。
const ModuleName = "access"

// Config 是 access protocol server 的装配输入。
type Config struct {
	// Socket 是 canonical 客户端监听路径；Phase 3 后 access 就是唯一本地入口。
	Socket string
	// Provider 为每个客户端会话建立一个 terminal provider 会话；必填。
	Provider terminalprovider.DialFunc
	// Logger 缺省用 slog.Default。
	Logger *slog.Logger
	// Tracker 可选：remoteauth 会话登记表（access/sessions.Registry），
	// 用于 revoke/过期时关闭客户端 transport，从而释放 provider 附件与桥接。
	Tracker SessionTracker
	// Files 配置 access-local 文件服务（路径 roots 与断点续传记录目录）。
	// 零值时路径不设 roots（legacy 语义），续传记录落在用户 state 目录。
	Files files.Config
	// Auth 是 client_access.* / cloud.* 的直答服务（access/runtime）。
	// nil 时 FamilyAuth 回退为 provider（过渡兼容）。
	Auth *AuthServices
}

// SessionTracker 登记已认证的远程会话，供撤销/过期踢线。
// 由 access/sessions.Registry 实现。
type SessionTracker interface {
	Track(grantID string, connection transport.Transport, expiresAt time.Time) func()
}

// Server 监听客户端连接并为每条连接运行一个 access session。
// storage 是 server 级 access-local 真值，跨 session 共享。
type Server struct {
	cfg     Config
	storage *storage.Store
	files   *files.Service
	proxy   *proxy.Service

	listenerMu sync.Mutex
	listener   transport.Listener
	closed     bool
}

// New 创建 access protocol server。
func New(cfg Config) (*Server, error) {
	if strings.TrimSpace(cfg.Socket) == "" {
		return nil, errors.New("access/server: socket path is required")
	}
	if cfg.Provider == nil {
		return nil, errors.New("access/server: terminal provider dial function is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	cfg.Files.Logger = cfg.Logger
	filesService, err := files.NewService(cfg.Files)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, storage: storage.New(), files: filesService, proxy: proxy.New(cfg.Logger)}, nil
}

// Socket 返回配置的客户端监听路径。
func (server *Server) Socket() string {
	if server == nil {
		return ""
	}
	return server.cfg.Socket
}

// Serve 绑定本地 owner-only socket 并服务客户端连接，直到 ctx 取消或 Close。
func (server *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	listener, err := unixtransport.NewListener(server.cfg.Socket)
	if err != nil {
		return err
	}
	if err := server.installListener(listener); err != nil {
		_ = listener.Close()
		return err
	}
	defer server.clearListener()
	server.cfg.Logger.Info("access protocol server listening", "socket_path", listener.Addr())
	for {
		connection, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, transport.ErrListenerClosed) || server.isClosed() {
				return nil
			}
			return err
		}
		go func() {
			if serveErr := server.ServeTransport(ctx, connection); serveErr != nil && !isExpectedSessionEnd(serveErr) {
				server.cfg.Logger.Debug("access protocol session stopped", "error", serveErr)
			}
		}()
	}
}

// ServeTransport 在一条已经可信的 transport 上服务 access wire。
// 远程调用方必须已完成 remoteauth；本函数不再做 identity challenge。
func (server *Server) ServeTransport(ctx context.Context, connection transport.Transport) error {
	if connection == nil {
		return errors.New("access/server: transport is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if server.cfg.Tracker != nil {
		if info, ok := sessions.RemoteSessionInfoFromContext(ctx); ok && info.GrantID != "" {
			release := server.cfg.Tracker.Track(info.GrantID, connection, info.ExpiresAt)
			defer release()
		}
	}
	session, err := newSession(server, connection)
	if err != nil {
		_ = connection.Close()
		return err
	}
	return session.run(ctx)
}

func (server *Server) installListener(listener transport.Listener) error {
	server.listenerMu.Lock()
	defer server.listenerMu.Unlock()
	if server.closed {
		return errors.New("access/server: server is closed")
	}
	server.listener = listener
	return nil
}

func (server *Server) clearListener() {
	server.listenerMu.Lock()
	server.listener = nil
	server.listenerMu.Unlock()
}

func (server *Server) isClosed() bool {
	server.listenerMu.Lock()
	defer server.listenerMu.Unlock()
	return server.closed
}

// Close 停止监听并释放监听 socket；已建立的 session 由其 transport 生命周期结束。
func (server *Server) Close() error {
	if server == nil {
		return nil
	}
	server.listenerMu.Lock()
	server.closed = true
	listener := server.listener
	server.listener = nil
	server.listenerMu.Unlock()
	if server.files != nil {
		_ = server.files.Close()
	}
	if listener == nil {
		return nil
	}
	return listener.Close()
}

// isExpectedSessionEnd 报告客户端正常断开或主动取消，不产生 error 级日志。
func isExpectedSessionEnd(err error) bool {
	if err == nil {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, io.ErrClosedPipe)
}
