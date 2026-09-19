// Package proxy 是 access-local 的 browser/webview 端口转发服务。
//
// 它从 daemon/core 迁移而来（Phase 2）：access 从本机拨号目标 TCP 服务，
// 双向字节流终结在 access session 的 stream channel 上。它不解释 HTTP，
// 只做有界转发与接收窗口/上传队列流控。
package proxy

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/access/wire"
)

// MaximumReceiveWindow 是一条 browser proxy 允许协商的最大接收窗口。
const MaximumReceiveWindow uint32 = 1 << 20

const (
	idleTimeout  = 2 * time.Minute
	writeTimeout = 30 * time.Second
)

// ErrDone 由 stream handler 返回，表示该 stream 已到达终态，调用方必须回收 binding。
var ErrDone = errors.New("browser proxy stream is done")

// Stream 是 access session 提供给 proxy 的帧通道。
type Stream interface {
	Send(typ uint8, payload []byte) error
	OutboundBuffered() uint64
	Done() <-chan struct{}
}

// Handle 是一个已打开 browser proxy 的 session 侧句柄。
type Handle interface {
	HandleFrame(ctx context.Context, typ uint8, payload []byte) (bool, error)
	Done() <-chan struct{}
	Close() error
}

// OpenRequest 是一次 browser proxy open 输入。
type OpenRequest struct {
	Host          string
	Port          uint16
	ReceiveWindow uint32
	SendWindow    uint32
	Channel       uint16
}

// Proxy 是 open 成功的元数据投影；Token 内嵌 channel。
type Proxy struct {
	Token              []byte
	ReceiveWindowBytes uint32
	SendWindowBytes    uint32
}

// Service 是 access 级 proxy registry；每个 Handle 绑定一个 session stream。
type Service struct {
	logger *slog.Logger
}

// New 创建 proxy 服务。
func New(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{logger: logger}
}

// Open 从 access 主机拨号目标并建立双向转发。
func (service *Service) Open(ctx context.Context, request OpenRequest, stream Stream) (*Proxy, Handle, error) {
	if stream == nil {
		return nil, nil, errors.New("browser proxy stream is required")
	}
	if request.ReceiveWindow > MaximumReceiveWindow {
		return nil, nil, errors.New("browser receive window exceeds maximum")
	}
	if request.SendWindow > MaximumReceiveWindow {
		return nil, nil, errors.New("browser send window exceeds maximum")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	address := net.JoinHostPort(request.Host, fmt.Sprintf("%d", request.Port))
	service.logger.Info("browser proxy dialing remote target", "host", request.Host, "port", request.Port, "channel", request.Channel)
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		service.logger.Warn("browser proxy remote dial failed", "host", request.Host, "port", request.Port, "channel", request.Channel, "error", err)
		return nil, nil, err
	}
	token := make([]byte, 34)
	binary.BigEndian.PutUint16(token[:2], request.Channel)
	if _, err := rand.Read(token[2:]); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	// dial 受请求 context 约束；已建立连接的 handle 生命周期独立于请求，
	// 只由 Close/session 结束终止。
	proxyCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	handle := &proxyHandle{
		service:  service,
		ctx:      proxyCtx,
		cancel:   cancel,
		channel:  request.Channel,
		token:    token,
		conn:     conn,
		stream:   stream,
		done:     make(chan struct{}),
		closeOne: sync.Once{},
	}
	if request.ReceiveWindow != 0 {
		handle.receiveWindow = newReceiveWindow(request.ReceiveWindow)
	}
	if request.SendWindow != 0 {
		handle.uploadQueue = newUploadQueue(int(request.SendWindow))
	}
	return &Proxy{Token: append([]byte(nil), token...), ReceiveWindowBytes: request.ReceiveWindow, SendWindowBytes: request.SendWindow}, handle, nil
}

// proxyHandle 实现 session localResource 契约。
type proxyHandle struct {
	service *Service

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	channel uint16
	token   []byte
	conn    net.Conn
	stream  Stream

	receiveWindow *receiveWindow
	uploadQueue   *uploadQueue
	uploadOnce    sync.Once
	forwardOnce   sync.Once
	closeOne      sync.Once
	clientOnce    sync.Once
	serverOnce    sync.Once
	writeMu       sync.Mutex
	sendMu        sync.Mutex
}

func (handle *proxyHandle) startForward() {
	handle.forwardOnce.Do(func() { go handle.forwardFromTarget() })
}

func (handle *proxyHandle) HandleFrame(ctx context.Context, typ uint8, payload []byte) (bool, error) {
	_ = ctx
	switch typ {
	case wire.TypeFileAck:
		if handle.receiveWindow == nil {
			return false, errors.New("browser receive window was not negotiated")
		}
		ack, err := protocol.DecodeFileTransferAck(payload)
		if err != nil {
			return false, err
		}
		return false, handle.receiveWindow.acknowledge(ack.Offset, ack.WindowBytes)
	case wire.TypeBrowserData:
		if handle.uploadQueue != nil {
			if err := handle.uploadQueue.enqueue(payload); err != nil {
				handle.Close()
				return true, err
			}
			handle.uploadOnce.Do(func() { go handle.writeUploads() })
			handle.startForward()
			return false, nil
		}
		handle.clientOnce.Do(func() {
			handle.service.logger.Info("browser proxy received client data", "channel", handle.channel, "bytes", len(payload))
		})
		handle.writeMu.Lock()
		defer handle.writeMu.Unlock()
		_ = handle.conn.SetReadDeadline(time.Now().Add(idleTimeout))
		written, err := writeAll(handle.conn, payload)
		if err != nil {
			handle.service.logger.Warn("browser proxy target write failed", "channel", handle.channel, "written_bytes", written, "error", err)
			handle.Close()
			return true, nil
		}
		handle.startForward()
		return false, nil
	case wire.TypeClosed:
		if len(payload) != 0 {
			return false, errors.New("browser proxy close payload must be empty")
		}
		handle.Close()
		return true, nil
	default:
		return false, fmt.Errorf("unsupported browser proxy frame type %d", typ)
	}
}

func (handle *proxyHandle) Done() <-chan struct{} { return handle.done }

// Close 幂等关闭目标连接、队列与窗口。
func (handle *proxyHandle) Close() error {
	handle.closeOne.Do(func() {
		handle.cancel()
		if handle.uploadQueue != nil {
			handle.uploadQueue.close()
		}
		if handle.receiveWindow != nil {
			handle.receiveWindow.close()
		}
		_ = handle.conn.Close()
		close(handle.done)
	})
	return nil
}

func (handle *proxyHandle) writeUploads() {
	buffer := make([]byte, 32<<10)
	var offset int64
	for {
		count, err := handle.uploadQueue.peek(buffer)
		if err != nil {
			return
		}
		_ = handle.conn.SetReadDeadline(time.Now().Add(idleTimeout))
		written, err := writeAll(handle.conn, buffer[:count])
		if err != nil {
			handle.service.logger.Warn("browser proxy target write failed", "channel", handle.channel, "written_bytes", written, "error", err)
			handle.Close()
			return
		}
		handle.uploadQueue.consume(count)
		offset += int64(count)
		ack, err := protocol.EncodeFileTransferAck(protocol.FileTransferAck{Offset: offset, WindowBytes: int64(count)})
		if err == nil {
			handle.sendMu.Lock()
			err = handle.stream.Send(wire.TypeFileAck, ack)
			handle.sendMu.Unlock()
		}
		if err != nil {
			handle.Close()
			return
		}
	}
}

func (handle *proxyHandle) forwardFromTarget() {
	defer handle.Close()
	buffer := make([]byte, 32<<10)
	for {
		limit := len(buffer)
		if handle.receiveWindow != nil {
			var err error
			limit, err = handle.receiveWindow.available(limit, writeTimeout)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					handle.service.logger.Warn("browser proxy receive credit timed out", "channel", handle.channel)
				}
				return
			}
		}
		_ = handle.conn.SetReadDeadline(time.Now().Add(idleTimeout))
		count, err := handle.conn.Read(buffer[:limit])
		if count > 0 {
			if handle.receiveWindow != nil {
				if windowErr := handle.receiveWindow.recordSent(count); windowErr != nil {
					return
				}
			}
			handle.serverOnce.Do(func() {
				handle.service.logger.Info("browser proxy sent server data", "channel", handle.channel, "bytes", count)
			})
			handle.sendMu.Lock()
			sendErr := handle.stream.Send(wire.TypeBrowserData, buffer[:count])
			handle.sendMu.Unlock()
			if sendErr != nil {
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return
			}
			return
		}
	}
}

func writeAll(conn net.Conn, payload []byte) (int, error) {
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return 0, fmt.Errorf("set browser proxy write deadline: %w", err)
	}
	total := 0
	for len(payload) > 0 {
		written, err := conn.Write(payload)
		total += written
		if err != nil {
			return total, fmt.Errorf("write browser proxy data: %w", err)
		}
		if written <= 0 {
			return total, io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return total, nil
}
