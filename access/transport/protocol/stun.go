package protocol

import (
	"net"
	"strings"
)

// EdgeSTUNURL probes the Edge's shared UDP service port without allocating TURN.
// A deployment without STUN on this port simply contributes no srflx candidate;
// host candidates and explicitly advertised TURN endpoints remain usable.
func EdgeSTUNURL(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if _, _, err := net.SplitHostPort(endpoint); err != nil {
		return ""
	}
	return "stun:" + endpoint
}
