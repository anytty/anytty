package cli

import (
	"github.com/anytty/anytty/access/direct"
	"github.com/anytty/anytty/proto/access/remoteauthpb"
)

type v3DirectPairingRouteOptions struct {
	DefaultListen      string
	DirectAddresses    []string
	SignalingAddresses []string
	ICETCPAddresses    []string
	ServerName         string
}

// v3DirectPairingRoute 返回 pair create 签名进 bootstrap 的唯一 Direct Route Proto。
// 实现由 access/direct 持有，pairing route 投影与 listener 组合保持同一套地址规则。
func v3DirectPairingRoute(options v3DirectPairingRouteOptions) (*remoteauthpb.EndpointRouteConfigV1, error) {
	return direct.PairingRoute(direct.RouteOptions{
		DefaultListen:      options.DefaultListen,
		DirectAddresses:    options.DirectAddresses,
		SignalingAddresses: options.SignalingAddresses,
		ICETCPAddresses:    options.ICETCPAddresses,
		ServerName:         options.ServerName,
	})
}

func directListenerSeeds(signalingAddress, iceAddress string) ([]string, []string, error) {
	return direct.ListenerSeeds(signalingAddress, iceAddress)
}

func normalizeDirectPairingAddresses(values []string) ([]string, error) {
	return direct.NormalizeAddresses(values)
}

func isWildcardHost(host string) bool {
	return direct.IsWildcardHost(host)
}

func v3DirectAddresses() (string, string) {
	return direct.ConfiguredAddresses()
}

func validateDirectListenAddress(address string) error {
	return direct.ValidateListenAddress(address)
}
