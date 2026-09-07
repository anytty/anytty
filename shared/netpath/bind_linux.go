package netpath

import (
	"golang.org/x/sys/unix"
	"net"
)

func interfaceBinder(iface net.Interface) func(uintptr, string) error {
	return func(fd uintptr, _ string) error { return unix.BindToDevice(int(fd), iface.Name) }
}
