package system

import (
	"fmt"
	"net"
	"sort"
)

// PrivateLANIPv4Addresses 返回当前主机已启用接口上的 RFC1918 IPv4 地址。
// 该函数只提供 host capability 快照，不选择 Endpoint、Route 或 runtime session；调用方必须把结果作为可预览的 locator seed。
func PrivateLANIPv4Addresses() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}
	seen := make(map[string]struct{})
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, addressErr := networkInterface.Addrs()
		if addressErr != nil {
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ipv4 := ip.To4(); ipv4 != nil && ipv4.IsPrivate() {
				seen[ipv4.String()] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for address := range seen {
		result = append(result, address)
	}
	sort.Strings(result)
	return result, nil
}
