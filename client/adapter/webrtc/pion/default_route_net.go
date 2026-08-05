package pion

import (
	"fmt"
	"net"
	"sort"
	"strings"

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
		return nil, fmt.Errorf("default route has no usable IP address")
	}

	keys := make([]string, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	interfaces := make([]*transport.Interface, 0, len(keys))
	for index, key := range keys {
		ip := addresses[key]
		name := "default-route-v6"
		bits := net.IPv6len * 8
		if ip.To4() != nil {
			ip = ip.To4()
			name, bits = "default-route-v4", net.IPv4len*8
		}
		candidate := transport.NewInterface(net.Interface{Index: index + 1, Name: name, MTU: 1500, Flags: net.FlagUp})
		candidate.AddAddress(&net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		interfaces = append(interfaces, candidate)
	}
	return &defaultRouteNet{Net: &stdnet.Net{}, interfaces: interfaces}, nil
}

// defaultRouteNet 复用标准 socket IO，只替换 Android 无权执行的 interface enumeration。
type defaultRouteNet struct {
	*stdnet.Net
	interfaces []*transport.Interface
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
