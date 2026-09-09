package netpath

import (
	"golang.org/x/sys/unix"
	"net"
	"strings"
)

func interfaceBinder(iface net.Interface) func(uintptr, string) error {
	return func(fd uintptr, network string) error {
		if strings.HasSuffix(network, "6") {
			return unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, iface.Index)
		}
		return unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, iface.Index)
	}
}
