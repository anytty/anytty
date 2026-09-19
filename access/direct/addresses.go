// Package direct 拥有 Direct WebRTC 的网络入口、地址投影与 LAN discovery。
// access gateway 组合本包；pool 不再是 Direct listener 的 owner。
package direct

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	systemadapter "github.com/anytty/anytty/access/engine/adapter/system"
	"github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
)

// DefaultSignalingAddress 是非显式配置时的 Direct 共享 listener 默认地址。
const DefaultSignalingAddress = "0.0.0.0:41120"

// PrivateLANAddresses 是 LAN locator seed 的 host capability hook；测试可替换。
var PrivateLANAddresses = systemadapter.PrivateLANIPv4Addresses

// RuntimeAddresses 持有当前进程实际绑定的 Direct listener 地址。
var runtimeAddresses struct {
	sync.RWMutex
	signaling string
	ice       string
}

// ConfiguredAddresses 返回 Direct listener 配置：优先本进程实际绑定地址，其次环境变量。
func ConfiguredAddresses() (string, string) {
	runtimeAddresses.RLock()
	activeSignaling := runtimeAddresses.signaling
	activeICE := runtimeAddresses.ice
	runtimeAddresses.RUnlock()
	if activeSignaling != "" && activeICE != "" {
		return activeSignaling, activeICE
	}
	shared := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN"))
	signaling := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_SIGNALING_LISTEN"))
	if signaling == "" {
		signaling = shared
	}
	if signaling == "" {
		signaling = DefaultSignalingAddress
	}
	ice := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_ICE_TCP_LISTEN"))
	if ice == "" {
		if shared != "" {
			ice = shared
		} else {
			ice = signaling
		}
	}
	return signaling, ice
}

func setRuntimeAddresses(signaling, ice string) {
	runtimeAddresses.Lock()
	runtimeAddresses.signaling = signaling
	runtimeAddresses.ice = ice
	runtimeAddresses.Unlock()
}

func clearRuntimeAddresses(signaling, ice string) {
	runtimeAddresses.Lock()
	if runtimeAddresses.signaling == signaling && runtimeAddresses.ice == ice {
		runtimeAddresses.signaling = ""
		runtimeAddresses.ice = ""
	}
	runtimeAddresses.Unlock()
}

// ValidateListenAddress 校验 Direct listener 的 HOST:PORT 形式。
func ValidateListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("Direct route %q must be HOST:PORT", address)
	}
	if strings.TrimSpace(host) == "" {
		host = "0.0.0.0"
	}
	value, err := strconv.ParseUint(strings.TrimSpace(port), 10, 16)
	if err != nil || value == 0 {
		return fmt.Errorf("Direct route %q has an invalid port", address)
	}
	return nil
}

// RouteOptions 是一次 Direct pairing Route 投影的输入。
type RouteOptions struct {
	DefaultListen      string
	DirectAddresses    []string
	SignalingAddresses []string
	ICETCPAddresses    []string
	ServerName         string
}

// PairingRoute 返回 pair create 签名进 bootstrap 的唯一 Direct Route Proto。
// 显式地址用于 FRP/TCP mapping 并完全替代 LAN seed；默认 wildcard listener 只投影为可预览的 RFC1918 locator。
func PairingRoute(options RouteOptions) (*remoteauthpb.EndpointRouteConfigV1, error) {
	signalingAddress, iceAddress := ConfiguredAddresses()
	if value := strings.TrimSpace(options.DefaultListen); value != "" {
		signalingAddress, iceAddress = value, value
	}
	directOverrides, err := NormalizeAddresses(options.DirectAddresses)
	if err != nil {
		return nil, fmt.Errorf("Direct address override: %w", err)
	}
	signalingOverrides, err := NormalizeAddresses(options.SignalingAddresses)
	if err != nil {
		return nil, fmt.Errorf("Direct signaling address override: %w", err)
	}
	iceOverrides, err := NormalizeAddresses(options.ICETCPAddresses)
	if err != nil {
		return nil, fmt.Errorf("Direct ICE-TCP address override: %w", err)
	}
	if len(signalingOverrides) == 0 != (len(iceOverrides) == 0) {
		return nil, fmt.Errorf("Direct signaling and ICE-TCP overrides must be provided together")
	}
	if len(directOverrides) > 0 && (len(signalingOverrides) > 0 || len(iceOverrides) > 0) {
		return nil, fmt.Errorf("Direct address cannot be combined with separate signaling or ICE-TCP overrides")
	}
	if len(directOverrides) > 0 {
		signalingOverrides = append([]string(nil), directOverrides...)
		iceOverrides = append([]string(nil), directOverrides...)
	} else if len(signalingOverrides) == 0 {
		signalingOverrides, iceOverrides, err = ListenerSeeds(signalingAddress, iceAddress)
		if err != nil {
			return nil, err
		}
	}
	advertised := append(append([]string(nil), signalingOverrides...), iceOverrides...)
	advertised = UniqueSortedStrings(advertised)
	serverName := strings.TrimSpace(options.ServerName)
	if strings.ContainsAny(serverName, "\r\n\t ") {
		return nil, fmt.Errorf("Direct server name must not contain whitespace")
	}
	return &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: endpoint.RouteConfigVersion, RouteId: "direct", Enabled: true,
		Route: &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{
			SignalingAddresses: signalingOverrides, IceTcpAddresses: iceOverrides, AdvertisedAddresses: advertised, ServerName: serverName,
		}},
	}, nil
}

// ListenerSeeds 把 listener 地址投影为可发布的 Direct signaling/ICE-TCP seed。
func ListenerSeeds(signalingAddress, iceAddress string) ([]string, []string, error) {
	signalingHost, signalingPort, err := net.SplitHostPort(signalingAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("parse Direct signaling listener %q: %w", signalingAddress, err)
	}
	iceHost, icePort, err := net.SplitHostPort(iceAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("parse Direct ICE-TCP listener %q: %w", iceAddress, err)
	}
	if !IsWildcardHost(signalingHost) || !IsWildcardHost(iceHost) {
		signaling, err := NormalizeAddresses([]string{signalingAddress})
		if err != nil {
			return nil, nil, err
		}
		ice, err := NormalizeAddresses([]string{iceAddress})
		return signaling, ice, err
	}
	hosts, err := PrivateLANAddresses()
	if err != nil {
		return nil, nil, err
	}
	if len(hosts) == 0 {
		return nil, nil, fmt.Errorf("no private LAN address is available; provide explicit Direct signaling and ICE-TCP addresses")
	}
	hosts = UniqueSortedStrings(hosts)
	signaling := make([]string, 0, len(hosts))
	ice := make([]string, 0, len(hosts))
	for _, host := range hosts {
		signaling = append(signaling, net.JoinHostPort(host, signalingPort))
		ice = append(ice, net.JoinHostPort(host, icePort))
	}
	return signaling, ice, nil
}

// NormalizeAddresses 校验并规范化显式 HOST:PORT 列表，拒绝 wildcard 与零端口。
func NormalizeAddresses(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("address must not be empty")
		}
		host, port, err := net.SplitHostPort(value)
		portNumber, portErr := strconv.ParseUint(strings.TrimSpace(port), 10, 16)
		if err != nil || portErr != nil || portNumber == 0 || strings.TrimSpace(host) == "" || IsWildcardHost(host) {
			return nil, fmt.Errorf("address %q must be a reachable HOST:PORT", value)
		}
		result = append(result, net.JoinHostPort(strings.TrimSpace(host), strings.TrimSpace(port)))
	}
	return UniqueSortedStrings(result), nil
}

// UniqueSortedStrings 去重并按字典序返回。
func UniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// IsWildcardHost 判断 host 是否为空或通配地址。
func IsWildcardHost(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::"
}
