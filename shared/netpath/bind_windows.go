package netpath

import (
	"golang.org/x/sys/windows"
	"math/bits"
	"net"
	"strings"
)

func interfaceBinder(iface net.Interface) func(uintptr, string) error {
	return func(fd uintptr, network string) error {
		if strings.HasSuffix(network, "6") {
			return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, 31, iface.Index)
		}
		return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, 31, int(bits.ReverseBytes32(uint32(iface.Index))))
	}
}
