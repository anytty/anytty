// Package provider 实现 daemon 侧的内部 terminal provider 协议服务器。
//
// 它只服务 access 的 provider 连接：把 provider 请求映射到 core.Server /
// Terminal 的终端服务（进程/PTY/history/live/events），不解析客户端 access wire，
// 也不持有 identity/store/files/auth。协议定义见 proto/provider/v1。
//
// Slice A1 覆盖 terminal lifecycle 与 path 查询；attach 流、history、live、events
// 在后续 slice 按同一 framing 扩展。
package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/daemon/core/history"

	"github.com/anytty/anytty/daemon/core"
	"github.com/anytty/anytty/internal/providerproto"
	providerv1 "github.com/anytty/anytty/proto/provider/v1"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

// Config 是 provider server 的装配输入。
type Config struct {
	// Socket 是 owner-only unix socket 路径。
	Socket string
	// Logger 缺省用 slog.Default。
	Logger *slog.Logger
}

// Server 在 provider socket 上服务 access 的终端请求。
type Server struct {
	core *core.Server
	cfg  Config

	registry         *attachmentRegistry
	nextSessionID    atomic.Uint64
	liveBaselineUsed atomic.Int64

	listenerMu sync.Mutex
	listener   transport.Listener
	closed     bool
}

// New 创建 provider server；coreServer 提供终端真值。
func New(coreServer *core.Server, cfg Config) (*Server, error) {
	if coreServer == nil {
		return nil, errors.New("provider: core server is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Server{core: coreServer, cfg: cfg, registry: newAttachmentRegistry()}, nil
}

// Socket 返回监听路径。
func (server *Server) ListenAndServe(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	listener, err := unixtransport.NewListener(server.cfg.Socket)
	if err != nil {
		return err
	}
	server.listenerMu.Lock()
	if server.closed {
		server.listenerMu.Unlock()
		_ = listener.Close()
		return errors.New("provider: server is closed")
	}
	server.listener = listener
	server.listenerMu.Unlock()
	defer server.clearListener()
	server.cfg.Logger.Info("terminal provider listening", "socket_path", listener.Addr())
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
			if serveErr := server.ServeTransport(ctx, connection); serveErr != nil && ctx.Err() == nil {
				server.cfg.Logger.Debug("terminal provider session stopped", "error", serveErr)
			}
		}()
	}
}

// ServeTransport 在一条已有连接上服务 provider 协议。
func (server *Server) ServeTransport(ctx context.Context, connection transport.Transport) error {
	if connection == nil {
		return errors.New("provider: transport is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	session := newSession(server, connection)
	return session.run(ctx)
}

// Shutdown 关闭监听并停止接受新连接。
func (server *Server) Shutdown(context.Context) error {
	server.listenerMu.Lock()
	server.closed = true
	listener := server.listener
	server.listener = nil
	server.listenerMu.Unlock()
	if listener != nil {
		return listener.Close()
	}
	return nil
}

func (server *Server) reserveLiveBaselineBytes(bytes int64) bool {
	if server == nil || bytes <= 0 {
		return false
	}
	for {
		current := server.liveBaselineUsed.Load()
		if current+bytes > maxLiveScreenBaselineServerBytes {
			return false
		}
		if server.liveBaselineUsed.CompareAndSwap(current, current+bytes) {
			return true
		}
	}
}

func (server *Server) releaseLiveBaselineBytes(bytes int64) {
	if server == nil || bytes <= 0 {
		return
	}
	server.liveBaselineUsed.Add(-bytes)
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

const maxInFlightRequests = 64

type session struct {
	server    *Server
	conn      transport.Transport
	sessionID uint64

	sendMu sync.Mutex
	mu     sync.RWMutex
	hello  bool

	streamMu    sync.Mutex
	attachments map[uint16]*providerAttachment
	tokens      map[string]*providerAttachment
	streams     map[uint16]*providerRawStream
	nextChannel uint32

	resourceMu               sync.Mutex
	historyTokenReservations int
	historyTokens            map[history.HistoryToken]string

	liveBaselineMu     sync.Mutex
	liveBaselines      map[string]*sessionLiveScreenBaselines
	liveBaselineBytes  int64
	liveBaselineTimer  *time.Timer
	liveBaselineClosed bool

	eventMu                sync.Mutex
	nextEventSub           uint64
	eventSubscriptions     map[uint64]*providerEventSubscription
	eventSubscriptionCount int

	slots    chan struct{}
	requests sync.WaitGroup
}

func newSession(server *Server, connection transport.Transport) *session {
	return &session{
		server:             server,
		conn:               connection,
		sessionID:          server.nextSessionID.Add(1),
		attachments:        make(map[uint16]*providerAttachment),
		tokens:             make(map[string]*providerAttachment),
		streams:            make(map[uint16]*providerRawStream),
		nextChannel:        6,
		historyTokens:      make(map[history.HistoryToken]string),
		liveBaselines:      make(map[string]*sessionLiveScreenBaselines),
		eventSubscriptions: make(map[uint64]*providerEventSubscription),
		slots:              make(chan struct{}, maxInFlightRequests),
	}
}

func (session *session) run(ctx context.Context) error {
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer session.requests.Wait()
	defer session.releaseAttachments()
	defer session.releaseSessionState()
	for {
		frame, err := session.conn.Recv()
		if err != nil {
			return err
		}
		channel, typ, payload, err := providerproto.DecodeFrame(frame)
		if err != nil {
			return err
		}
		if channel != 0 {
			if err := session.handleStreamFrame(sessionCtx, channel, typ, payload); err != nil {
				if sendErr := session.sendError(0, providerproto.ErrorBadRequest, err.Error()); sendErr != nil {
					return sendErr
				}
			}
			continue
		}
		switch typ {
		case providerproto.TypeHello:
			if err := session.handleHello(payload); err != nil {
				return err
			}
		case providerproto.TypeRequest:
			if !session.isHello() {
				if err := session.sendError(0, providerproto.ErrorBadRequest, "provider Hello is required before requests"); err != nil {
					return err
				}
				continue
			}
			request, err := providerproto.DecodeRequestPayload(payload)
			if err != nil {
				return err
			}
			if request.GetId() == 0 || request.GetCommand() == nil {
				if err := session.sendError(request.GetId(), providerproto.ErrorBadRequest, "provider request id and command are required"); err != nil {
					return err
				}
				continue
			}
			select {
			case session.slots <- struct{}{}:
			default:
				if err := session.sendError(request.GetId(), providerproto.ErrorExhausted, "provider in-flight request capacity is exhausted"); err != nil {
					return err
				}
				continue
			}
			session.requests.Add(1)
			go func() {
				defer session.requests.Done()
				defer func() { <-session.slots }()
				if err := session.handleRequest(sessionCtx, request); err != nil {
					_ = session.conn.Close()
				}
			}()
		default:
			if err := session.sendError(0, providerproto.ErrorBadRequest, fmt.Sprintf("unsupported provider control frame type %d", typ)); err != nil {
				return err
			}
		}
	}
}

func (session *session) handleHello(payload []byte) error {
	if session.isHello() {
		return session.sendError(0, providerproto.ErrorBadRequest, "provider Hello was already accepted")
	}
	hello, err := providerproto.DecodeHelloPayload(payload)
	if err != nil {
		return session.sendError(0, providerproto.ErrorBadRequest, err.Error())
	}
	if hello.GetVersion() != 0 && hello.GetVersion() != providerproto.Version {
		return session.sendError(0, providerproto.ErrorBadRequest, fmt.Sprintf("unsupported provider version %d", hello.GetVersion()))
	}
	response, err := providerproto.EncodeHelloPayload(&providerv1.Hello{Version: providerproto.Version, Server: "anyttyd-provider"})
	if err != nil {
		return err
	}
	if err := session.sendFrame(0, providerproto.TypeHello, response); err != nil {
		return err
	}
	session.mu.Lock()
	session.hello = true
	session.mu.Unlock()
	return nil
}

func (session *session) handleRequest(ctx context.Context, request *providerv1.Request) error {
	response, err := session.dispatch(ctx, request)
	if err != nil {
		var providerErr *ProviderError
		if errors.As(err, &providerErr) {
			return session.sendError(request.GetId(), providerErr.Code, providerErr.Message)
		}
		return session.sendError(request.GetId(), providerproto.ErrorInternal, err.Error())
	}
	response.Id = request.GetId()
	payload, err := providerproto.EncodeResponsePayload(response)
	if err != nil {
		return err
	}
	return session.sendFrame(0, providerproto.TypeResponse, payload)
}

func (session *session) dispatch(ctx context.Context, request *providerv1.Request) (*providerv1.Response, error) {
	coreServer := session.server.core
	switch command := request.GetCommand().(type) {
	case *providerv1.Request_TerminalCreate:
		record, err := terminalRecordFromSpec(command.TerminalCreate)
		if err != nil {
			return nil, err
		}
		info, err := coreServer.RegisterTerminal(record)
		if err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalCreate{TerminalCreate: terminalInfoToProtoWithAttachments(info, session.server.registry.viewCount(info.ID))}}, nil
	case *providerv1.Request_TerminalList:
		items := coreServer.ListTerminals()
		result := &providerv1.TerminalListResult{Terminals: make([]*providerv1.TerminalInfo, 0, len(items))}
		for _, item := range items {
			result.Terminals = append(result.Terminals, terminalInfoToProtoWithAttachments(item, session.server.registry.viewCount(item.ID)))
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalList{TerminalList: result}}, nil
	case *providerv1.Request_TerminalGet:
		info, err := coreServer.GetTerminal(command.TerminalGet.GetTerminalId())
		if err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalGet{TerminalGet: &providerv1.TerminalGetResult{Terminal: terminalInfoToProtoWithAttachments(info, session.server.registry.viewCount(info.ID))}}}, nil
	case *providerv1.Request_TerminalRestart:
		if err := coreServer.RestartTerminal(ctx, command.TerminalRestart.GetTerminalId()); err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalRestart{TerminalRestart: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalKill:
		if err := coreServer.KillTerminal(ctx, command.TerminalKill.GetTerminalId()); err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalKill{TerminalKill: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalRemove:
		if err := coreServer.RemoveTerminal(command.TerminalRemove.GetTerminalId()); err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalRemove{TerminalRemove: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalSetMetadata:
		name := command.TerminalSetMetadata.GetName()
		if _, err := coreServer.SetMetadata(ctx, command.TerminalSetMetadata.GetTerminal().GetTerminalId(), name, command.TerminalSetMetadata.GetTags()); err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalSetMetadata{TerminalSetMetadata: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalSetTags:
		terminalID := command.TerminalSetTags.GetTerminal().GetTerminalId()
		info, err := coreServer.GetTerminal(terminalID)
		if err != nil {
			return nil, mapCoreError(err)
		}
		if _, err := coreServer.SetMetadata(ctx, terminalID, info.Name, command.TerminalSetTags.GetTags()); err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalSetTags{TerminalSetTags: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalDefaults:
		defaults := coreServer.TerminalDefaultsSnapshot()
		return &providerv1.Response{Result: &providerv1.Response_TerminalDefaults{TerminalDefaults: &providerv1.TerminalDefaults{
			Command: append([]string(nil), defaults.DefaultCommand...), Cwd: defaults.DefaultCWD, Platform: defaults.Platform,
		}}}, nil
	case *providerv1.Request_PathListDirectories:
		directories, err := coreServer.ListPathDirectories(command.PathListDirectories.GetPrefix(), int(command.PathListDirectories.GetLimit()))
		if err != nil {
			return nil, mapCoreError(err)
		}
		result := &providerv1.PathListDirectoriesResult{BasePath: directories.BasePath, Missing: directories.Missing, Truncated: directories.Truncated}
		for _, entry := range directories.Entries {
			result.Entries = append(result.Entries, &providerv1.PathDirectory{Name: entry.Name, Path: entry.Path})
		}
		return &providerv1.Response{Result: &providerv1.Response_PathListDirectories{PathListDirectories: result}}, nil
	case *providerv1.Request_TerminalAttach:
		return session.dispatchAttach(command.TerminalAttach)
	case *providerv1.Request_TerminalDetach:
		attachment, err := session.attachmentForToken(command.TerminalDetach.GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		session.detachAttachment(attachment)
		return &providerv1.Response{Result: &providerv1.Response_TerminalDetach{TerminalDetach: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalInput:
		attachment, err := session.attachmentForToken(command.TerminalInput.GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		if attachment.mode == attachmentModeObserver {
			return nil, &ProviderError{Code: providerproto.ErrorForbidden, Message: "provider: observer attachment cannot send terminal input"}
		}
		if err := coreServer.WriteInput(ctx, attachment.terminalID, command.TerminalInput.GetData()); err != nil {
			return nil, mapCoreError(err)
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalInput{TerminalInput: &providerv1.Acknowledge{}}}, nil
	case *providerv1.Request_TerminalResize:
		return session.dispatchResize(ctx, command.TerminalResize)
	case *providerv1.Request_TerminalResizeLock:
		attachment, err := session.attachmentForToken(command.TerminalResizeLock.GetOpaqueToken())
		if err != nil {
			return nil, err
		}
		info, err := coreServer.GetTerminal(attachment.terminalID)
		if err != nil {
			return nil, mapCoreError(err)
		}
		control, err := session.server.registry.setSizeLock(attachment, info.Size, command.TerminalResizeLock.GetLocked())
		if err != nil {
			return nil, err
		}
		return &providerv1.Response{Result: &providerv1.Response_TerminalResizeLock{TerminalResizeLock: &providerv1.TerminalResizeResult{
			Size: sizeToProto(info.Size), ResizeControl: control,
		}}}, nil
	case *providerv1.Request_HistoryWindow:
		return session.dispatchHistoryWindow(ctx, command.HistoryWindow)
	case *providerv1.Request_HistoryCopy:
		return session.dispatchHistoryCopy(ctx, command.HistoryCopy)
	case *providerv1.Request_HistorySearch:
		return session.dispatchHistorySearch(ctx, command.HistorySearch)
	case *providerv1.Request_HistoryRelease:
		return session.dispatchHistoryRelease(ctx, command.HistoryRelease)
	case *providerv1.Request_HistoryBacklogStatus:
		return session.dispatchHistoryBacklog(command.HistoryBacklogStatus)
	case *providerv1.Request_LiveScreenNext:
		return session.dispatchLiveScreenNext(ctx, command.LiveScreenNext)
	case *providerv1.Request_EventSubscribe:
		return session.dispatchEventSubscribe(ctx, command.EventSubscribe)
	case *providerv1.Request_EventRelease:
		return session.dispatchEventRelease(command.EventRelease)
	default:
		return nil, &ProviderError{Code: providerproto.ErrorBadRequest, Message: "unsupported provider command"}
	}
}

func (session *session) isHello() bool {
	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.hello
}

func (session *session) sendFrame(channel uint16, typ uint8, payload []byte) error {
	frame, err := providerproto.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	session.sendMu.Lock()
	defer session.sendMu.Unlock()
	return session.conn.Send(frame)
}

func (session *session) sendError(id uint64, code uint32, message string) error {
	payload, err := providerproto.EncodeErrorPayload(id, code, message)
	if err != nil {
		return err
	}
	return session.sendFrame(0, providerproto.TypeResponse, payload)
}
