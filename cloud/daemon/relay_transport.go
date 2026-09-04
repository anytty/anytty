package daemon

import (
	"fmt"
	"net/url"
	"strings"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

// filterDaemonRelayICEURLs keeps both peers on the transport selected by the
// client attempt. STUN remains available for ICE bookkeeping, while TURN URLs
// are restricted to the requested allocation transport.
func filterDaemonRelayICEURLs(values []string, preference cloudv1.RelayTransport) ([]string, error) {
	switch preference {
	case cloudv1.RelayTransport_RELAY_TRANSPORT_UNSPECIFIED,
		cloudv1.RelayTransport_RELAY_TRANSPORT_UDP,
		cloudv1.RelayTransport_RELAY_TRANSPORT_TCP,
		cloudv1.RelayTransport_RELAY_TRANSPORT_TLS:
	default:
		return nil, fmt.Errorf("unsupported Relay transport %d", preference)
	}
	if preference == cloudv1.RelayTransport_RELAY_TRANSPORT_UNSPECIFIED {
		return append([]string(nil), values...), nil
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		raw := strings.TrimSpace(value)
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parse Relay ICE URL: %w", err)
		}
		switch strings.ToLower(parsed.Scheme) {
		case "stun", "stuns":
			filtered = append(filtered, raw)
		case "turn":
			transport := strings.ToLower(strings.TrimSpace(parsed.Query().Get("transport")))
			if transport == "" {
				transport = "udp"
			}
			if transport != "udp" && transport != "tcp" {
				return nil, fmt.Errorf("Relay TURN URL has unsupported transport %q", transport)
			}
			if (preference == cloudv1.RelayTransport_RELAY_TRANSPORT_UDP && transport == "udp") ||
				(preference == cloudv1.RelayTransport_RELAY_TRANSPORT_TCP && transport == "tcp") {
				filtered = append(filtered, raw)
			}
		case "turns":
			if preference == cloudv1.RelayTransport_RELAY_TRANSPORT_TLS {
				filtered = append(filtered, raw)
			}
		default:
			return nil, fmt.Errorf("Relay ICE URL has unsupported scheme %q", parsed.Scheme)
		}
	}
	return filtered, nil
}

func hasDaemonTURNICEURL(values []string) bool {
	for _, value := range values {
		scheme := strings.ToLower(strings.TrimSpace(value))
		if strings.HasPrefix(scheme, "turn:") || strings.HasPrefix(scheme, "turns:") {
			return true
		}
	}
	return false
}
