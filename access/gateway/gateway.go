// Package gateway is the AnyTTY access gateway: an independent process that
// makes the current user's daemon reachable over operator-chosen listeners
// (tcp, unix) while staying byte-transparent to the access wire.
//
// One client connection is paired with one daemon socket connection and the
// bytes are copied in both directions (a controlled socat). The gateway does
// not parse frames, does not negotiate the Hello/version handshake and does
// not participate in pairing or capability grants: clients keep speaking the
// exact wire their transport already speaks, so shipped Flutter clients and
// the existing TUI/engine stay compatible.
//
// Authorization is deliberately network-layer only in this milestone:
// optional CIDR allow lists and an optional pre-shared pair-token
// pre-handshake. The access wire has no client-token field at connection
// level (Hello v7 carries only version/client labels), so no token bytes are
// injected into the transparent path by default. See
// access/docs/GATEWAY.zh-CN.md for the gap list and the tmux provider plan.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/anytty/anytty/access/provider"

	"log/slog"
)

// DefaultHandshakeTimeout bounds the optional pair-token pre-handshake.
const DefaultHandshakeTimeout = 5 * time.Second

// DefaultDialTimeout bounds one daemon socket dial when the provider does not
// carry its own timeout.
const DefaultDialTimeout = 5 * time.Second

// Config describes one gateway. Provider and at least one Listener are
// required. Allow, PairToken, Logger and timeouts are optional.
type Config struct {
	// Provider opens the daemon-side byte stream for every accepted client.
	Provider provider.SessionProvider
	// Listeners are the parsed listener specs to bind, e.g.
	// "tcp:127.0.0.1:7331" or "unix:/tmp/anytty-access.sock".
	Listeners []ListenerSpec
	// Allow restricts TCP peers to the listed networks. Empty allows all
	// peers (unix listeners are always allowed: the socket is owner-only).
	Allow []*net.IPNet
	// PairToken enables the optional pre-handshake when non-empty. Nil/empty
	// keeps the gateway fully transparent.
	PairToken []byte
	// Logger receives lifecycle logs; nil disables logging.
	Logger *slog.Logger
	// HandshakeTimeout bounds the pair-token pre-handshake.
	HandshakeTimeout time.Duration
	// DialTimeout bounds one provider dial.
	DialTimeout time.Duration
}

// Gateway owns the listeners, accepted client connections and the proxy
// lifecycle. Serve must be called before Addrs is meaningful.
type Gateway struct {
	cfg Config

	mu        sync.Mutex
	listeners []net.Listener
	addrs     []string
	conns     map[net.Conn]struct{}
	closed    bool

	ready     chan struct{}
	readyOnce sync.Once
	serveErr  chan error

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New validates cfg and returns a gateway that binds nothing yet.
func New(cfg Config) (*Gateway, error) {
	if cfg.Provider == nil {
		return nil, errors.New("gateway: provider is required")
	}
	if len(cfg.Listeners) == 0 {
		return nil, errors.New("gateway: at least one listener is required")
	}
	for _, spec := range cfg.Listeners {
		if err := spec.Validate(); err != nil {
			return nil, err
		}
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = DefaultHandshakeTimeout
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = DefaultDialTimeout
	}
	cfg.Logger = normalizeLogger(cfg.Logger)
	return &Gateway{
		cfg:      cfg,
		conns:    map[net.Conn]struct{}{},
		ready:    make(chan struct{}),
		serveErr: make(chan error, 1),
	}, nil
}

// Serve binds every configured listener and proxies connections until ctx is
// canceled or the gateway is closed. It returns nil on an orderly shutdown.
func (g *Gateway) Serve(ctx context.Context) error {
	if g == nil {
		return errors.New("gateway: nil gateway")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel
	defer cancel()

	bound := make([]net.Listener, 0, len(g.cfg.Listeners))
	addrs := make([]string, 0, len(g.cfg.Listeners))
	for _, spec := range g.cfg.Listeners {
		listener, err := spec.Listen()
		if err != nil {
			for _, existing := range bound {
				_ = existing.Close()
			}
			g.failReady(err)
			return err
		}
		bound = append(bound, listener)
		addrs = append(addrs, listener.Addr().String())
		g.cfg.Logger.Info("anytty-access listening", "listener", listener.Addr().String(), "network", listener.Addr().Network())
	}
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		for _, listener := range bound {
			_ = listener.Close()
		}
		err := errors.New("gateway: closed before serve")
		g.failReady(err)
		return err
	}
	g.listeners = bound
	g.addrs = append([]string(nil), addrs...)
	g.mu.Unlock()
	g.markReady()

	for _, listener := range bound {
		g.wg.Add(1)
		go g.acceptLoop(runCtx, listener)
	}
	<-runCtx.Done()
	g.closeListeners()
	g.wg.Wait()
	g.closeConns()
	g.cfg.Logger.Info("anytty-access stopped")
	return nil
}

// WaitReady blocks until every listener is bound or Serve failed.
func (g *Gateway) WaitReady(ctx context.Context) error {
	if g == nil {
		return errors.New("gateway: nil gateway")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-g.ready:
		// ready closes on success and on bind failure; a buffered serveErr
		// distinguishes the two.
		select {
		case err := <-g.serveErr:
			return err
		default:
			return nil
		}
	case err := <-g.serveErr:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addrs returns the bound listener addresses in configuration order.
func (g *Gateway) Addrs() []string {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.addrs...)
}

// Close stops the gateway: it closes listeners, cancels in-flight proxies and
// unblocks every tracked client connection. It is safe to call repeatedly and
// after Serve returned.
func (g *Gateway) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	alreadyClosed := g.closed
	g.closed = true
	listeners := append([]net.Listener(nil), g.listeners...)
	g.mu.Unlock()
	if alreadyClosed {
		return nil
	}
	if g.cancel != nil {
		g.cancel()
	}
	for _, listener := range listeners {
		_ = listener.Close()
	}
	g.closeConns()
	return nil
}

// closeConns closes every tracked client connection exactly once.
func (g *Gateway) closeConns() {
	g.mu.Lock()
	conns := make([]net.Conn, 0, len(g.conns))
	for conn := range g.conns {
		conns = append(conns, conn)
	}
	g.conns = map[net.Conn]struct{}{}
	g.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
}

func (g *Gateway) closeListeners() {
	for _, listener := range g.listeners {
		_ = listener.Close()
	}
}

func (g *Gateway) acceptLoop(ctx context.Context, listener net.Listener) {
	defer g.wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			g.cfg.Logger.Warn("anytty-access accept failed", "error", err)
			continue
		}
		g.wg.Add(1)
		go func() {
			defer g.wg.Done()
			g.serveConn(ctx, conn)
		}()
	}
}

// serveConn authorizes one client and proxies it to the provider stream. The
// access wire is copied verbatim; only the optional pair-token pre-handshake
// reads application bytes before the proxy starts.
func (g *Gateway) serveConn(ctx context.Context, client net.Conn) {
	if !g.trackConn(client) {
		_ = client.Close()
		return
	}
	defer func() {
		_ = client.Close()
		g.untrackConn(client)
	}()
	if !peerAllowed(g.cfg.Allow, client.RemoteAddr()) {
		g.cfg.Logger.Warn("anytty-access rejected peer by allow list", "peer", client.RemoteAddr().String())
		return
	}
	source, err := g.authorize(client)
	if err != nil {
		g.cfg.Logger.Warn("anytty-access rejected client", "peer", client.RemoteAddr().String(), "error", err)
		return
	}
	dialCtx, cancel := context.WithTimeout(ctx, g.cfg.DialTimeout)
	upstream, err := g.cfg.Provider.Dial(dialCtx)
	cancel()
	if err != nil {
		g.cfg.Logger.Warn("anytty-access daemon dial failed", "peer", client.RemoteAddr().String(), "error", err)
		return
	}
	g.cfg.Logger.Info("anytty-access connection established", "peer", client.RemoteAddr().String())
	defer func() {
		_ = upstream.Close()
		g.cfg.Logger.Info("anytty-access connection closed", "peer", client.RemoteAddr().String())
	}()
	proxy(ctx, client, upstream, source)
}

func (g *Gateway) trackConn(conn net.Conn) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return false
	}
	g.conns[conn] = struct{}{}
	return true
}

func (g *Gateway) untrackConn(conn net.Conn) {
	g.mu.Lock()
	delete(g.conns, conn)
	g.mu.Unlock()
}

func (g *Gateway) markReady() {
	if g == nil {
		return
	}
	g.readyOnce.Do(func() { close(g.ready) })
}

func (g *Gateway) failReady(err error) {
	if g == nil {
		return
	}
	g.readyOnce.Do(func() { close(g.ready) })
	select {
	case g.serveErr <- err:
	default:
	}
}

func normalizeLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.New(slog.DiscardHandler)
}

// parsePeer extracts the IP part of a TCP peer address. Unix peers report
// "@"-style or filesystem addresses and are not subject to the IP allow list.
func parsePeer(addr net.Addr) net.IP {
	if addr == nil {
		return nil
	}
	if tcp, ok := addr.(*net.TCPAddr); ok {
		return tcp.IP
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil
	}
	return net.ParseIP(strings.Trim(host, "[]"))
}

// peerAllowed reports whether addr passes the configured allow list. An empty
// list allows everything; non-TCP peers bypass the list.
func peerAllowed(allow []*net.IPNet, addr net.Addr) bool {
	if len(allow) == 0 {
		return true
	}
	if _, ok := addr.(*net.TCPAddr); !ok {
		return true
	}
	ip := parsePeer(addr)
	if ip == nil {
		return false
	}
	for _, network := range allow {
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

// ParseNetworks parses allow-list entries: CIDR ("10.0.0.0/8") or a bare IP
// ("127.0.0.1", treated as a single host). Empty entries are skipped.
func ParseNetworks(values []string) ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(value); err == nil {
			networks = append(networks, network)
			continue
		}
		ip := net.ParseIP(strings.Trim(value, "[]"))
		if ip == nil {
			return nil, fmt.Errorf("gateway: allow entry %q is not an IP or CIDR", value)
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		networks = append(networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return networks, nil
}
