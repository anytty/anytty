package core

import (
	"fmt"
	"io"

	"github.com/anytty/anytty/proto/wire"
)

func (session *protocolSession) handleBrowserProxyFrame(proxy *sessionBrowserProxy, typ uint8, payload []byte) error {
	switch typ {
	case wire.TypeBrowserData:
		proxy.writeMu.Lock()
		defer proxy.writeMu.Unlock()
		for len(payload) > 0 {
			written, err := proxy.conn.Write(payload)
			if err != nil {
				return fmt.Errorf("write browser proxy data: %w", err)
			}
			if written <= 0 {
				return io.ErrShortWrite
			}
			payload = payload[written:]
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

func (session *protocolSession) forwardBrowserProxy(proxy *sessionBrowserProxy) {
	buffer := make([]byte, 32<<10)
	for {
		count, err := proxy.conn.Read(buffer)
		if count > 0 {
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
	session.browserMu.Lock()
	defer session.browserMu.Unlock()
	return session.browserChannels[channel]
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
