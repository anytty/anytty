package provider

import (
	"context"
	"errors"
	"net"
)

// SessionProvider opens one byte stream for one client connection. Dial must
// return a connection owned by the caller, which is responsible for closing
// it; ctx only bounds stream establishment.
type SessionProvider interface {
	Dial(ctx context.Context) (net.Conn, error)
}

// DialFunc adapts a function to SessionProvider.
type DialFunc func(ctx context.Context) (net.Conn, error)

// Dial implements SessionProvider.
func (f DialFunc) Dial(ctx context.Context) (net.Conn, error) {
	if f == nil {
		return nil, errors.New("provider: nil dial function")
	}
	return f(ctx)
}
