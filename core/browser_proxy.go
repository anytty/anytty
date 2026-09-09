package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/wire"
)

const browserProxyIdleTimeout = 2 * time.Minute
const browserProxyWriteTimeout = 30 * time.Second

func (session *protocolSession) handleBrowserProxyFrame(proxy *sessionBrowserProxy, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeFileAck:
		if proxy.receiveWindow == nil {
			return fmt.Errorf("browser receive window was not negotiated")
		}
		ack, err := protocol.DecodeFileTransferAck(payload)
		if err != nil {
			return err
		}
		return proxy.receiveWindow.acknowledge(ack.Offset, ack.WindowBytes)
	case wire.TypeBrowserData:
		if proxy.uploadQueue != nil {
			if err := proxy.uploadQueue.enqueue(payload); err != nil {
				session.removeBrowserProxy(proxy)
				return err
			}
			proxy.uploadOnce.Do(func() { go session.writeBrowserProxyUploads(proxy) })
			proxy.forwardOnce.Do(func() { go session.forwardBrowserProxy(proxy) })
			return nil
		}
		proxy.clientDataOnce.Do(func() {
			session.server.cfg.logger.Info(
				"browser proxy received client data",
				"session_id", session.sessionID,
				"channel", proxy.channel,
				"bytes", len(payload),
			)
		})
		proxy.writeMu.Lock()
		defer proxy.writeMu.Unlock()
		_ = proxy.conn.SetReadDeadline(time.Now().Add(browserProxyIdleTimeout))
		started := time.Now()
		written, err := writeBrowserProxyData(proxy.conn, payload)
		if err != nil {
			session.server.cfg.logger.Warn("browser proxy target write failed", "session_id", session.sessionID, "channel", proxy.channel, "written_bytes", written, "elapsed_ms", time.Since(started).Milliseconds(), "error", err)
			if session.removeBrowserProxy(proxy) {
				return session.sendFrame(proxy.channel, wire.TypeBrowserClosed, nil)
			}
			return nil
		}
		proxy.forwardOnce.Do(func() { go session.forwardBrowserProxy(proxy) })
		return nil
	case wire.TypeClosed:
		if len(payload) != 0 {
			return fmt.Errorf("browser proxy close payload must be empty")
		}
		session.removeBrowserProxy(proxy)
		return nil
	default:
		return fmt.Errorf("unsupported browser proxy frame type %d", typ)
	}
}

func (session *protocolSession) writeBrowserProxyUploads(proxy *sessionBrowserProxy) {
	buffer := make([]byte, 32<<10)
	var offset int64
	var writeWait, maximumWrite time.Duration
	defer func() {
		session.server.cfg.logger.Info("browser proxy upload worker stopped", "session_id", session.sessionID, "channel", proxy.channel, "stage", "target_write", "upload_bytes", offset, "write_wait_ms", writeWait.Milliseconds(), "max_write_ms", maximumWrite.Milliseconds())
	}()
	for {
		count, err := proxy.uploadQueue.peek(buffer)
		if err != nil {
			return
		}
		started := time.Now()
		_ = proxy.conn.SetReadDeadline(time.Now().Add(browserProxyIdleTimeout))
		written, err := writeBrowserProxyData(proxy.conn, buffer[:count])
		duration := time.Since(started)
		writeWait += duration
		maximumWrite = max(maximumWrite, duration)
		if err != nil {
			session.server.cfg.logger.Warn("browser proxy target write failed", "session_id", session.sessionID, "channel", proxy.channel, "stage", "target_write", "written_bytes", written, "elapsed_ms", time.Since(started).Milliseconds(), "error", err)
			if session.removeBrowserProxy(proxy) {
				_ = session.sendFrame(proxy.channel, wire.TypeBrowserClosed, nil)
			}
			return
		}
		proxy.uploadQueue.consume(count)
		_ = proxy.conn.SetReadDeadline(time.Now().Add(browserProxyIdleTimeout))
		offset += int64(count)
		ack, err := protocol.EncodeFileTransferAck(protocol.FileTransferAck{Offset: offset, WindowBytes: int64(count)})
		if err == nil {
			err = session.sendFrame(proxy.channel, wire.TypeFileAck, ack)
		}
		if err != nil {
			session.removeBrowserProxy(proxy)
			return
		}
	}
}

func writeBrowserProxyData(conn net.Conn, payload []byte) (int, error) {
	if err := conn.SetWriteDeadline(time.Now().Add(browserProxyWriteTimeout)); err != nil {
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

func (session *protocolSession) browserChannelState(channel uint16) (*sessionBrowserProxy, bool) {
	session.browserMu.Lock()
	defer session.browserMu.Unlock()
	return session.browserChannels[channel], session.closedBrowserChannels[channel/64]&(uint64(1)<<(channel%64)) != 0
}

func (session *protocolSession) forwardBrowserProxy(proxy *sessionBrowserProxy) {
	buffer := make([]byte, 32<<10)
	for {
		limit := len(buffer)
		if proxy.receiveWindow != nil {
			var err error
			limit, err = proxy.receiveWindow.available(limit, browserProxyWriteTimeout)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					session.server.cfg.logger.Warn("browser proxy receive credit timed out", "session_id", session.sessionID, "channel", proxy.channel, "stage", "receive_credit", "timeout_ms", browserProxyWriteTimeout.Milliseconds())
				}
				if session.removeBrowserProxy(proxy) {
					_ = session.sendFrame(proxy.channel, wire.TypeBrowserClosed, nil)
				}
				return
			}
		}
		_ = proxy.conn.SetReadDeadline(time.Now().Add(browserProxyIdleTimeout))
		count, err := proxy.conn.Read(buffer[:limit])
		if count > 0 {
			if proxy.receiveWindow != nil {
				if windowErr := proxy.receiveWindow.recordSent(count); windowErr != nil {
					if session.removeBrowserProxy(proxy) {
						_ = session.sendFrame(proxy.channel, wire.TypeBrowserClosed, nil)
					}
					return
				}
			}
			_ = proxy.conn.SetReadDeadline(time.Now().Add(browserProxyIdleTimeout))
			proxy.serverDataOnce.Do(func() {
				session.server.cfg.logger.Info(
					"browser proxy sent server data",
					"session_id", session.sessionID,
					"channel", proxy.channel,
					"bytes", count,
				)
			})
			if sendErr := session.sendFrame(proxy.channel, wire.TypeBrowserData, buffer[:count]); sendErr != nil {
				session.removeBrowserProxy(proxy)
				return
			}
		}
		if err != nil {
			if session.removeBrowserProxy(proxy) {
				_ = session.sendFrame(proxy.channel, wire.TypeBrowserClosed, nil)
			}
			return
		}
	}
}

func (session *protocolSession) browserProxyForChannel(channel uint16) *sessionBrowserProxy {
	proxy, _ := session.browserChannelState(channel)
	return proxy
}

func (session *protocolSession) browserProxyForToken(token []byte) *sessionBrowserProxy {
	session.browserMu.Lock()
	defer session.browserMu.Unlock()
	channel, ok := session.browserTokens[string(token)]
	if !ok {
		return nil
	}
	return session.browserChannels[channel]
}

func (session *protocolSession) removeBrowserProxy(proxy *sessionBrowserProxy) bool {
	if proxy == nil {
		return false
	}
	session.browserMu.Lock()
	current := session.browserChannels[proxy.channel]
	if current != proxy {
		session.browserMu.Unlock()
		return false
	}
	delete(session.browserChannels, proxy.channel)
	delete(session.browserTokens, string(proxy.token))
	// Channel IDs are never reused within a session. A fixed 8 KiB bitmap
	// recognizes in-flight frames without retaining closed sockets or tokens.
	session.closedBrowserChannels[proxy.channel/64] |= uint64(1) << (proxy.channel % 64)
	session.browserMu.Unlock()
	session.releaseChannel(proxy.channel, protocolChannelBrowserProxy)
	proxy.close()
	return true
}

func (session *protocolSession) releaseAllBrowserProxies() {
	session.browserMu.Lock()
	proxies := make([]*sessionBrowserProxy, 0, len(session.browserChannels))
	for _, proxy := range session.browserChannels {
		proxies = append(proxies, proxy)
	}
	session.browserMu.Unlock()
	for _, proxy := range proxies {
		session.removeBrowserProxy(proxy)
	}
}
