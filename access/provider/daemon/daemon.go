// Package daemon hosts two daemon-facing adapters: the provider protocol
// TerminalProvider (apipb ↔ providerv1, see terminal.go/events.go) and a
// byte-transparent SessionProvider used by the access relay. In Phase 4+ the
// relay target is the canonical access socket because remote peers speak the
// access wire; route selection stays in the composition root.
package daemon

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/anytty/anytty/access/provider"
)

// DefaultDialTimeout bounds one daemon socket dial.
const DefaultDialTimeout = 5 * time.Second

// Provider is a byte-transparent SessionProvider: it dials Socket and hands
// the raw stream to the relay without protocol participation.
type Provider struct {
	Socket      string
	DialTimeout time.Duration
}

var _ provider.SessionProvider = Provider{}

// New returns a byte-relay provider for the given socket path.
func New(socket string) Provider {
	return Provider{Socket: strings.TrimSpace(socket)}
}

// Dial implements provider.SessionProvider. It returns the raw byte stream to
// the target socket without a Hello or any protocol participation, so the
// gateway stays transparent to the access wire. Long path aliases are followed
// by the kernel when the listener publishes them as symlinks; on Windows, where
// the alias is not created, pass the resolvable short path.
func (p Provider) Dial(ctx context.Context) (net.Conn, error) {
	path := strings.TrimSpace(p.Socket)
	if path == "" {
		return nil, errors.New("provider/daemon: socket path is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := p.DialTimeout
	if timeout <= 0 {
		timeout = DefaultDialTimeout
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return (&net.Dialer{}).DialContext(dialCtx, "unix", path)
}
