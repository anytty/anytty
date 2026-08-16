//go:build android && cgo

package main

/*
#cgo LDFLAGS: -landroid
#include <android/multinetwork.h>
#include <errno.h>

static int anytty_bind_socket_network(uint64_t network, int fd) {
	if (android_setsocknetwork((net_handle_t)network, fd) == 0) return 0;
	return errno;
}
*/
import "C"

import (
	"context"
	"fmt"
	"net"
	"syscall"

	pionadapter "github.com/anytty/anytty/client/adapter/webrtc/pion"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/pion/transport/v4"
)

func newAndroidRouteNetwork(handle uint64) (transport.Net, error) {
	if handle == 0 {
		return nil, fmt.Errorf("Android route network handle is required")
	}
	return pionadapter.NewBoundRouteNet(func(fd uintptr) error {
		if errno := int(C.anytty_bind_socket_network(C.uint64_t(handle), C.int(fd))); errno != 0 {
			return syscall.Errno(errno)
		}
		return nil
	})
}

func dialLocalDiscoveryCandidate(ctx context.Context, candidate *bindingpb.LocalDiscoveryCandidate, address string) (net.Conn, error) {
	dialer := &net.Dialer{}
	if handle := candidate.GetNetworkHandle(); handle != 0 {
		dialer.Control = func(_, _ string, raw syscall.RawConn) error {
			var bindErr error
			if err := raw.Control(func(fd uintptr) {
				if errno := int(C.anytty_bind_socket_network(C.uint64_t(handle), C.int(fd))); errno != 0 {
					bindErr = syscall.Errno(errno)
				}
			}); err != nil {
				return err
			}
			return bindErr
		}
	}
	return dialer.DialContext(ctx, "tcp", address)
}
