package netpath

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pion/transport/v4"
	"github.com/pion/transport/v4/stdnet"
)

// ICENetwork exposes every usable source address, and binds candidate sockets
// to their source network so a VPN default route cannot capture LAN checks.
type ICENetwork struct {
	*stdnet.Net
	paths      []Path
	interfaces []*transport.Interface
	bound      *Path
}

func (value *ICENetwork) ICEGatherNetworks() []transport.Net {
	networks := make([]transport.Net, 0, len(value.paths))
	for _, path := range value.paths {
		copy := *value
		copy.bound = &path
		networks = append(networks, &copy)
	}
	return networks
}

func NewICENetwork() (transport.Net, error) {
	value := &ICENetwork{Net: &stdnet.Net{}, paths: Paths()}
	for _, path := range value.paths {
		if len(path.Addresses) == 0 {
			continue
		}
		iface := transport.NewInterface(net.Interface{Index: len(value.interfaces) + 1, Name: path.Name, MTU: 1500, Flags: net.FlagUp | net.FlagMulticast})
		for _, address := range path.Addresses {
			copy := address
			iface.AddAddress(&copy)
		}
		value.interfaces = append(value.interfaces, iface)
	}
	if len(value.interfaces) == 0 {
		return nil, fmt.Errorf("no usable ICE network addresses")
	}
	return value, nil
}

func (value *ICENetwork) Interfaces() ([]*transport.Interface, error) { return value.interfaces, nil }
func (value *ICENetwork) InterfaceByIndex(index int) (*transport.Interface, error) {
	for _, iface := range value.interfaces {
		if iface.Index == index {
			return iface, nil
		}
	}
	return nil, transport.ErrInterfaceNotFound
}
func (value *ICENetwork) InterfaceByName(name string) (*transport.Interface, error) {
	for _, iface := range value.interfaces {
		if iface.Name == name {
			return iface, nil
		}
	}
	return nil, transport.ErrInterfaceNotFound
}
func (value *ICENetwork) source(ip net.IP) Path {
	if value.bound != nil {
		return *value.bound
	}
	for _, path := range value.paths {
		for _, address := range path.Addresses {
			if address.IP.Equal(ip) {
				return path
			}
		}
	}
	return Path{ID: "default"}
}

func (value *ICENetwork) local(network string, ip net.IP) net.IP {
	if value.bound == nil || (ip != nil && !ip.IsUnspecified()) {
		return ip
	}
	for _, address := range value.bound.Addresses {
		if strings.HasSuffix(network, "6") == (address.IP.To4() == nil) {
			return address.IP
		}
	}
	return ip
}
func (value *ICENetwork) ListenUDP(network string, local *net.UDPAddr) (transport.UDPConn, error) {
	path := value.source(nil)
	address := ":0"
	if local == nil {
		local = &net.UDPAddr{}
	}
	if local != nil {
		copy := *local
		copy.IP = value.local(network, copy.IP)
		local = &copy
		path = value.source(local.IP)
		address = local.String()
	}
	conn, err := (&net.ListenConfig{Control: path.Control}).ListenPacket(context.Background(), network, address)
	if err != nil {
		return nil, err
	}
	return conn.(*net.UDPConn), nil
}
func (value *ICENetwork) DialUDP(network string, local, remote *net.UDPAddr) (transport.UDPConn, error) {
	path := value.source(nil)
	if local != nil {
		path = value.source(local.IP)
	}
	dialer := path.Dialer()
	dialer.LocalAddr = local
	conn, err := dialer.Dial(network, remote.String())
	if err != nil {
		return nil, err
	}
	return conn.(*net.UDPConn), nil
}
func (value *ICENetwork) Dial(network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if value.bound != nil {
		return value.bound.Dialer().DialContext(ctx, network, address)
	}
	return Default.DialContext(ctx, network, address)
}

func (value *ICENetwork) ListenPacket(network, address string) (net.PacketConn, error) {
	local, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}
	conn, err := value.ListenUDP(network, local)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (value *ICENetwork) resolve(network, address string) (net.IP, string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, "", err
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip, port, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	resolver := net.DefaultResolver
	if value.bound != nil && value.bound.Dialer().Resolver != nil {
		resolver = value.bound.Dialer().Resolver
	}
	family := "ip"
	if strings.HasSuffix(network, "4") {
		family = "ip4"
	} else if strings.HasSuffix(network, "6") {
		family = "ip6"
	}
	ips, err := resolver.LookupIP(ctx, family, host)
	if err != nil {
		return nil, "", err
	}
	if len(ips) == 0 {
		return nil, "", fmt.Errorf("no address for %s on source network", host)
	}
	return ips[0], port, nil
}

func (value *ICENetwork) ResolveUDPAddr(network, address string) (*net.UDPAddr, error) {
	if host, _, err := net.SplitHostPort(address); err == nil && strings.Contains(host, "%") {
		return net.ResolveUDPAddr(network, address)
	}
	ip, port, err := value.resolve(network, address)
	if err != nil {
		return nil, err
	}
	return net.ResolveUDPAddr(network, net.JoinHostPort(ip.String(), port))
}
func (value *ICENetwork) ResolveTCPAddr(network, address string) (*net.TCPAddr, error) {
	if host, _, err := net.SplitHostPort(address); err == nil && strings.Contains(host, "%") {
		return net.ResolveTCPAddr(network, address)
	}
	ip, port, err := value.resolve(network, address)
	if err != nil {
		return nil, err
	}
	return net.ResolveTCPAddr(network, net.JoinHostPort(ip.String(), port))
}
func (value *ICENetwork) DialTCP(network string, local, remote *net.TCPAddr) (transport.TCPConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, err := value.DialTCPContext(ctx, network, local, remote)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

// DialTCPContext is consumed by active ICE-TCP checks in the pinned ICE transport.
func (value *ICENetwork) DialTCPContext(ctx context.Context, network string, local, remote *net.TCPAddr) (net.Conn, error) {
	if local == nil || local.IP.IsUnspecified() {
		if value.bound != nil {
			return value.bound.Dialer().DialContext(ctx, network, remote.String())
		}
		return Default.DialContext(ctx, network, remote.String())
	}
	dialer := value.source(local.IP).Dialer()
	dialer.LocalAddr = local
	return dialer.DialContext(ctx, network, remote.String())
}
