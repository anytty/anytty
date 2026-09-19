package gateway

import (
	"context"
	"io"
	"net"
)

// halfCloser is the optional TCP/Unix half-close capability. It lets one
// direction finish (FIN) while the other keeps streaming, matching raw TCP
// proxy semantics.
type halfCloser interface {
	CloseWrite() error
}

// proxy copies bytes in both directions between client and upstream and
// propagates half-close. It performs no framing, buffering or rewriting:
// whatever wire the client speaks reaches the pool unchanged and vice versa.
//
// clientReader may carry buffered bytes from the pair-token pre-handshake so
// no client byte is lost or duplicated. The proxy returns when both directions
// end or ctx is canceled.
func proxy(ctx context.Context, client net.Conn, upstream net.Conn, clientReader io.Reader) {
	if clientReader == nil {
		clientReader = client
	}
	done := make(chan struct{}, 2)
	copyDirection := func(dst net.Conn, src io.Reader) {
		_, _ = io.Copy(dst, src)
		closeWrite(dst)
		done <- struct{}{}
	}
	go copyDirection(upstream, clientReader)
	go copyDirection(client, upstream)

	finished := 0
	for finished < 2 {
		select {
		case <-done:
			finished++
		case <-ctx.Done():
			// The gateway is closing: drop both ends so no copier stays
			// blocked on a peer that will never finish, then drain the
			// remaining direction signals (each direction sends exactly one).
			_ = client.Close()
			_ = upstream.Close()
			for finished < 2 {
				<-done
				finished++
			}
			return
		}
	}
}

// closeWrite half-closes the write side when the connection supports it, so
// the peer sees EOF without losing the reverse direction. Connections without
// half-close support stay open until Close.
func closeWrite(conn net.Conn) {
	if closer, ok := conn.(halfCloser); ok {
		_ = closer.CloseWrite()
	}
}
