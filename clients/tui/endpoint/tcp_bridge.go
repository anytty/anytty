package endpoint

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// tcpBridge is the tcp compatibility adapter: it exposes one unix socket that
// byte-transparently relays every connection to a HOST:PORT peer terminating
// at the daemon framed transport (an `ssh -L TCP->socket` forward or a
// transparent bridge). It carries no protocol logic: the connection itself is
// dialed by the shared local-unix route adapter over the relayed socket, so
// tui2 owns no second daemon dialer (CLIENT_SHARING §3, M1).
type tcpBridge struct {
	ln     net.Listener
	dir    string
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// startTCPBridge listens on a private unix socket and relays to address.
func startTCPBridge(parent context.Context, address, name string) (*tcpBridge, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("endpoint %q: connect_mode tcp requires an address (HOST:PORT)", name)
	}
	dir, err := os.MkdirTemp("", "anytty-tui2-tcp-")
	if err != nil {
		return nil, fmt.Errorf("endpoint %q: tcp bridge temp dir: %w", name, err)
	}
	socket := filepath.Join(dir, "daemon.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("endpoint %q: tcp bridge listen: %w", name, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if parent != nil {
		go func() {
			select {
			case <-parent.Done():
				cancel()
			case <-ctx.Done():
			}
		}()
	}
	bridge := &tcpBridge{ln: listener, dir: dir, cancel: cancel}
	bridge.wg.Add(1)
	go bridge.acceptLoop(ctx, address)
	return bridge, nil
}

// Socket returns the local unix socket the shared local route dials.
func (b *tcpBridge) Socket() string {
	if b == nil {
		return ""
	}
	return filepath.Join(b.dir, "daemon.sock")
}

func (b *tcpBridge) acceptLoop(ctx context.Context, address string) {
	defer b.wg.Done()
	for {
		conn, err := b.ln.Accept()
		if err != nil {
			return
		}
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.relay(ctx, conn, address)
		}()
	}
}

func (b *tcpBridge) relay(ctx context.Context, local net.Conn, address string) {
	defer local.Close()
	remote, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return
	}
	defer remote.Close()
	done := make(chan struct{}, 2)
	copyBoth := func(dst net.Conn, src net.Conn) {
		_, _ = io.Copy(dst, src)
		_ = dst.Close()
		_ = src.Close()
		done <- struct{}{}
	}
	go copyBoth(remote, local)
	go copyBoth(local, remote)
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Close stops the relay and removes the socket directory.
func (b *tcpBridge) Close() error {
	if b == nil {
		return nil
	}
	b.cancel()
	_ = b.ln.Close()
	b.wg.Wait()
	return os.RemoveAll(b.dir)
}
