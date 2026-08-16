package pion

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"syscall"

	"github.com/pion/transport/v4"
	"github.com/pion/transport/v4/stdnet"
)

var defaultRouteProbes = []struct {
	network string
	address string
}{
	{network: "udp4", address: "192.0.2.1:9"},
	{network: "udp6", address: "[2001:db8::1]:9"},
}

// NewDefaultRouteNet 创建只暴露当前系统默认路由地址的 Pion 网络 primitive。
// 它用于 Android application sandbox：该环境禁止直接读取 netlink，但允许普通 socket
// 由内核选择当前 active network。返回值是调用时的快照，必须为每个新 peer 重新创建。
func NewDefaultRouteNet() (transport.Net, error) {
	return newDefaultRouteNet(net.Dial)
}

type routeDial func(network, address string) (net.Conn, error)

// SocketBinder binds a newly-created socket before connect/listen. Android uses
// it to keep a discovered LAN route on the Network that delivered the mDNS
// candidate, without changing the process-wide default used by Cloud.
type SocketBinder func(fd uintptr) error

// NewBoundRouteNet creates a Pion network whose sockets are bound independently
// from the process default route.
func NewBoundRouteNet(bind SocketBinder) (transport.Net, error) {
	if bind == nil {
		return nil, fmt.Errorf("socket binder is required")
	}
	dialer := boundDialer(bind)
	network, err := newDefaultRouteNet(func(network, address string) (net.Conn, error) {
		return dialer.DialContext(context.Background(), network, address)
	})
	if err != nil {
		return nil, err
	}
	return &boundRouteNet{defaultRouteNet: network, bind: bind}, nil
}

func newDefaultRouteNet(dial routeDial) (*defaultRouteNet, error) {
	if dial == nil {
		return nil, fmt.Errorf("default route dialer is required")
	}
	addresses := make(map[string]net.IP)
	for _, probe := range defaultRouteProbes {
		connection, err := dial(probe.network, probe.address)
		if err != nil {
			continue
		}
		local, ok := connection.LocalAddr().(*net.UDPAddr)
		_ = connection.Close()
		if !ok || local == nil || local.IP == nil || local.IP.IsUnspecified() {
			continue
		}
		ip := append(net.IP(nil), local.IP...)
		addresses[ip.String()] = ip
	}
	if len(addresses) == 0 {
		log.Printf("anytty webrtc network ipv4=false ipv6=false route_count=0")
		return nil, fmt.Errorf("default route has no usable IP address")
	}

	keys := make([]string, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	interfaces := make([]*transport.Interface, 0, len(keys))
	hasIPv4, hasIPv6 := false, false
	for index, key := range keys {
		ip := addresses[key]
		name := "default-route-v6"
		bits := net.IPv6len * 8
		if ip.To4() != nil {
			ip = ip.To4()
			name, bits = "default-route-v4", net.IPv4len*8
			hasIPv4 = true
		} else {
			hasIPv6 = true
		}
		candidate := transport.NewInterface(net.Interface{Index: index + 1, Name: name, MTU: 1500, Flags: net.FlagUp})
		candidate.AddAddress(&net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		interfaces = append(interfaces, candidate)
	}
	log.Printf("anytty webrtc network ipv4=%t ipv6=%t route_count=%d", hasIPv4, hasIPv6, len(interfaces))
	return &defaultRouteNet{Net: &stdnet.Net{}, interfaces: interfaces}, nil
}

// defaultRouteNet 复用标准 socket IO，只替换 Android 无权执行的 interface enumeration。
type defaultRouteNet struct {
	*stdnet.Net
	interfaces []*transport.Interface
}

type boundRouteNet struct {
	*defaultRouteNet
	bind SocketBinder
}

func boundControl(bind SocketBinder) func(string, string, syscall.RawConn) error {
	return func(_, _ string, raw syscall.RawConn) error {
		var bindErr error
		if err := raw.Control(func(fd uintptr) { bindErr = bind(fd) }); err != nil {
			return err
		}
		return bindErr
	}
}

func boundDialer(bind SocketBinder) *net.Dialer {
	return &net.Dialer{Control: boundControl(bind)}
}

func (network *boundRouteNet) Dial(name, address string) (net.Conn, error) {
	return boundDialer(network.bind).DialContext(context.Background(), name, address)
}

func (network *boundRouteNet) DialTCP(name string, local, remote *net.TCPAddr) (transport.TCPConn, error) {
	dialer := boundDialer(network.bind)
	dialer.LocalAddr = local
	connection, err := dialer.DialContext(context.Background(), name, remote.String())
	if err != nil {
		return nil, err
	}
	tcp, ok := connection.(*net.TCPConn)
	if !ok {
		_ = connection.Close()
		return nil, fmt.Errorf("bound TCP dial returned %T", connection)
	}
	return tcp, nil
}

func (network *boundRouteNet) DialUDP(name string, local, remote *net.UDPAddr) (transport.UDPConn, error) {
	dialer := boundDialer(network.bind)
	dialer.LocalAddr = local
	connection, err := dialer.DialContext(context.Background(), name, remote.String())
	if err != nil {
		return nil, err
	}
	udp, ok := connection.(*net.UDPConn)
	if !ok {
		_ = connection.Close()
		return nil, fmt.Errorf("bound UDP dial returned %T", connection)
	}
	return udp, nil
}

func (network *boundRouteNet) ListenPacket(name, address string) (net.PacketConn, error) {
	return (&net.ListenConfig{Control: boundControl(network.bind)}).ListenPacket(context.Background(), name, address)
}

func (network *boundRouteNet) ListenUDP(name string, local *net.UDPAddr) (transport.UDPConn, error) {
	connection, err := network.ListenPacket(name, local.String())
	if err != nil {
		return nil, err
	}
	udp, ok := connection.(*net.UDPConn)
	if !ok {
		_ = connection.Close()
		return nil, fmt.Errorf("bound UDP listen returned %T", connection)
	}
	return udp, nil
}

func (network *boundRouteNet) ListenTCP(name string, local *net.TCPAddr) (transport.TCPListener, error) {
	listener, err := (&net.ListenConfig{Control: boundControl(network.bind)}).Listen(context.Background(), name, local.String())
	if err != nil {
		return nil, err
	}
	tcp, ok := listener.(*net.TCPListener)
	if !ok {
		_ = listener.Close()
		return nil, fmt.Errorf("bound TCP listen returned %T", listener)
	}
	return boundTCPListener{TCPListener: tcp}, nil
}

func (network *boundRouteNet) CreateDialer(dialer *net.Dialer) transport.Dialer {
	copy := *dialer
	copy.Control = boundControl(network.bind)
	return boundTransportDialer{Dialer: &copy}
}

func (network *boundRouteNet) CreateListenConfig(config *net.ListenConfig) transport.ListenConfig {
	copy := *config
	copy.Control = boundControl(network.bind)
	return boundTransportListenConfig{ListenConfig: &copy}
}

type boundTCPListener struct{ *net.TCPListener }

func (listener boundTCPListener) AcceptTCP() (transport.TCPConn, error) {
	return listener.TCPListener.AcceptTCP()
}

type boundTransportDialer struct{ *net.Dialer }

func (dialer boundTransportDialer) Dial(network, address string) (net.Conn, error) {
	return dialer.Dialer.Dial(network, address)
}

type boundTransportListenConfig struct{ *net.ListenConfig }

func (config boundTransportListenConfig) Listen(ctx context.Context, network, address string) (net.Listener, error) {
	return config.ListenConfig.Listen(ctx, network, address)
}
func (config boundTransportListenConfig) ListenPacket(ctx context.Context, network, address string) (net.PacketConn, error) {
	return config.ListenConfig.ListenPacket(ctx, network, address)
}

func (network *defaultRouteNet) Interfaces() ([]*transport.Interface, error) {
	return append([]*transport.Interface(nil), network.interfaces...), nil
}

func (network *defaultRouteNet) InterfaceByIndex(index int) (*transport.Interface, error) {
	for _, candidate := range network.interfaces {
		if candidate.Index == index {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("%w: index=%d", transport.ErrInterfaceNotFound, index)
}

func (network *defaultRouteNet) InterfaceByName(name string) (*transport.Interface, error) {
	for _, candidate := range network.interfaces {
		if candidate.Name == strings.TrimSpace(name) {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", transport.ErrInterfaceNotFound, name)
}
