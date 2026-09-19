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
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"syscall"

	pionadapter "github.com/anytty/anytty/access/engine/adapter/webrtc/pion"
	"github.com/anytty/anytty/proto/access/bindingpb"
	"github.com/anytty/anytty/shared/netpath"
	"github.com/pion/transport/v4"
)

var androidPaths = struct {
	sync.RWMutex
	paths []netpath.Path
}{}

func init() {
	netpath.SetProvider(func() []netpath.Path {
		androidPaths.RLock()
		defer androidPaths.RUnlock()
		return append([]netpath.Path{{ID: "default", Name: "system default"}}, androidPaths.paths...)
	})
}

//export anytty_android_network_snapshot
func anytty_android_network_snapshot(data *C.char) {
	var snapshot []struct {
		Handle    uint64
		Name      string
		Addresses []string
		DNS       []string
	}
	if data == nil || json.Unmarshal([]byte(C.GoString(data)), &snapshot) != nil {
		return
	}
	var paths []netpath.Path
	for _, value := range snapshot {
		if value.Handle == 0 {
			continue
		}
		handle := value.Handle
		path := netpath.Path{ID: fmt.Sprintf("android:%d", handle), Name: value.Name, DNS: value.DNS}
		for _, address := range value.Addresses {
			ip, prefix, err := net.ParseCIDR(address)
			if err != nil {
				continue
			}
			prefix.IP = ip
			path.Addresses = append(path.Addresses, *prefix)
		}
		path.Bind = func(fd uintptr, _ string) error {
			if errno := int(C.anytty_bind_socket_network(C.uint64_t(handle), C.int(fd))); errno != 0 {
				return syscall.Errno(errno)
			}
			return nil
		}
		paths = append(paths, path)
	}
	androidPaths.Lock()
	androidPaths.paths = paths
	androidPaths.Unlock()
}

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
