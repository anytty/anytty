// Package directprobe performs a lightweight TCP reachability check for a
// configured Direct route. It does not authenticate the daemon or create a
// WebRTC session.
package directprobe

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

const AddressTimeout = time.Second

type DialAddress func(context.Context, string) (net.Conn, error)

func DefaultDialAddress(ctx context.Context, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "tcp", address)
}

// Reachable probes the signaling addresses because a Direct session cannot
// start when its signaling TCP entry point is unavailable. Multiple configured
// addresses race so one slow address does not delay a reachable route.
func Reachable(ctx context.Context, route *remoteauthpb.DirectWebRTCTCPRouteConfig, dial DialAddress) bool {
	if route == nil {
		return false
	}
	if dial == nil {
		dial = DefaultDialAddress
	}
	addresses := normalizedAddresses(route.GetSignalingAddresses())
	if len(addresses) == 0 {
		return false
	}

	probeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan bool, len(addresses))
	for _, address := range addresses {
		go func() {
			addressCtx, addressCancel := context.WithTimeout(probeCtx, AddressTimeout)
			connection, err := dial(addressCtx, address)
			addressCancel()
			reachable := err == nil && connection != nil
			if connection != nil {
				_ = connection.Close()
			}
			results <- reachable
		}()
	}
	for range addresses {
		select {
		case reachable := <-results:
			if reachable {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
	return false
}

func normalizedAddresses(values []string) []string {
	addresses := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		host, portText, err := net.SplitHostPort(strings.TrimSpace(value))
		if err != nil || host == "" || strings.ContainsAny(host, " \t\r\n/") {
			continue
		}
		port, err := strconv.ParseUint(portText, 10, 16)
		if err != nil || port == 0 {
			continue
		}
		if ip := net.ParseIP(host); ip != nil {
			host = ip.String()
		}
		address := net.JoinHostPort(host, strconv.FormatUint(port, 10))
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		addresses = append(addresses, address)
	}
	return addresses
}
