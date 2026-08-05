package cloud

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
)

func filterManagedICEURLs(values []string, preference endpoint.RelayTransport) ([]string, error) {
	switch preference {
	case "", endpoint.RelayTransportAuto:
		return append([]string(nil), values...), nil
	case endpoint.RelayTransportUDP, endpoint.RelayTransportTCP:
	default:
		return nil, fmt.Errorf("unsupported Cloud relay transport %q", preference)
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		raw := strings.TrimSpace(value)
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parse Cloud ICE URL: %w", err)
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
				return nil, fmt.Errorf("Cloud TURN URL has unsupported transport %q", transport)
			}
			if string(preference) == transport {
				filtered = append(filtered, raw)
			}
		case "turns":
			if preference == endpoint.RelayTransportTCP {
				filtered = append(filtered, raw)
			}
		default:
			return nil, fmt.Errorf("Cloud ICE URL has unsupported scheme %q", parsed.Scheme)
		}
	}
	return filtered, nil
}

func hasManagedTURNServer(servers []port.ICEServer) bool {
	for _, server := range servers {
		for _, raw := range server.URLs {
			value := strings.ToLower(strings.TrimSpace(raw))
			if strings.HasPrefix(value, "turn:") || strings.HasPrefix(value, "turns:") {
				return true
			}
		}
	}
	return false
}
