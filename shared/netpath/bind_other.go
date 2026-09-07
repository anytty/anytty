//go:build !darwin && !linux && !windows

package netpath

import "net"

func interfaceBinder(net.Interface) func(uintptr, string) error { return nil }
