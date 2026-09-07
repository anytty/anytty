// Package netpath owns network-scoped sockets, not application sessions.
package netpath

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
)

// Path is an ephemeral OS network snapshot. Bind must never change process routing.
type Path struct {
	ID        string
	Name      string
	Addresses []net.IPNet
	DNS       []string
	Bind      func(uintptr, string) error
}

var provider = struct {
	sync.RWMutex
	get func() []Path
}{get: systemPaths}

// SetProvider installs the platform's network snapshot source at startup.
func SetProvider(get func() []Path) {
	provider.Lock()
	defer provider.Unlock()
	provider.get = get
}

func Paths() []Path {
	provider.RLock()
	get := provider.get
	provider.RUnlock()
	if get == nil {
		return nil
	}
	return get()
}

func (path Path) Control(network, _ string, raw syscall.RawConn) error {
	if path.Bind == nil {
		return nil
	}
	var bindErr error
	if err := raw.Control(func(fd uintptr) { bindErr = path.Bind(fd, network) }); err != nil {
		return err
	}
	return bindErr
}

func (path Path) Dialer() *net.Dialer {
	dialer := &net.Dialer{Control: path.Control}
	if path.ID == "default" {
		return dialer
	}
	// Force DNS sockets through the same network as the eventual connection.
	dialer.Resolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		dnsDialer := &net.Dialer{Control: path.Control}
		if len(path.DNS) == 0 {
			return dnsDialer.DialContext(ctx, network, address)
		}
		var last error
		for _, server := range path.DNS {
			conn, err := dnsDialer.DialContext(ctx, network, net.JoinHostPort(server, "53"))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		return nil, last
	}}
	return dialer
}

func (path Path) Key() string {
	var values []string
	for _, address := range path.Addresses {
		values = append(values, address.String())
	}
	return fmt.Sprintf("%s/%s/%s", path.ID, strings.Join(values, ","), strings.Join(path.DNS, ","))
}

func systemPaths() []Path {
	paths := []Path{{ID: "default", Name: "system default"}}
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		path := Path{ID: fmt.Sprintf("interface:%d", iface.Index), Name: iface.Name}
		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			ip, prefix, err := net.ParseCIDR(address.String())
			if err != nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLoopback() {
				continue
			}
			prefix.IP = ip
			path.Addresses = append(path.Addresses, *prefix)
		}
		if len(path.Addresses) == 0 {
			continue
		}
		bind := interfaceBinder(iface)
		if bind == nil {
			continue
		}
		path.Bind = bind
		paths = append(paths, path)
	}
	return paths
}
