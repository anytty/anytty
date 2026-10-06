package endpoint

import (
	"fmt"
	"os"
	"strings"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
)

// RoutesEnvVar names the comma-separated shared route kinds the TUI host is
// allowed to dial. The -routes host flag wins over it. Leaving it unset lets
// every route kind race; setting it narrows the policy (see
// tui2/docs/REMOTE.zh-CN.md §3.1).
const RoutesEnvVar = "TUI2_ROUTES"

// routeKindAliases maps operator-facing names to shared route kinds. "tcp"
// stays an alias of local-unix because the tui2 tcp compatibility mode relays
// bytes into a local unix socket; it is never a distinct transport.
var routeKindAliases = map[string]clientendpoint.RouteKind{
	"local":             clientendpoint.RouteLocalUnix,
	"local-unix":        clientendpoint.RouteLocalUnix,
	"unix":              clientendpoint.RouteLocalUnix,
	"tcp":               clientendpoint.RouteLocalUnix,
	"ssh":               clientendpoint.RouteSSHWebRTCTCP,
	"ssh-webrtc-tcp":    clientendpoint.RouteSSHWebRTCTCP,
	"direct":            clientendpoint.RouteDirectWebRTCTCP,
	"direct-webrtc-tcp": clientendpoint.RouteDirectWebRTCTCP,
	"webrtc":            clientendpoint.RouteDirectWebRTCTCP,
	"cloud":             clientendpoint.RouteManagedWebRTC,
	"managed":           clientendpoint.RouteManagedWebRTC,
	"managed-webrtc":    clientendpoint.RouteManagedWebRTC,
}

// DefaultRouteKinds is the out-of-the-box dial policy: every transport kind a
// route can be configured with races for the connection, and the first one that
// becomes ready wins (the planner staggers attempts by kind order: local, ssh,
// direct, cloud). A route still only participates when the registry enables it,
// its credentials exist and the kind is available on this platform; narrowing
// the race is an explicit opt-out via -routes / TUI2_ROUTES.
func DefaultRouteKinds() []clientendpoint.RouteKind {
	return []clientendpoint.RouteKind{
		clientendpoint.RouteLocalUnix,
		clientendpoint.RouteSSHWebRTCTCP,
		clientendpoint.RouteDirectWebRTCTCP,
		clientendpoint.RouteManagedWebRTC,
	}
}

// ResolveRouteKinds returns the effective enabled route kinds: the explicit
// flag value, else TUI2_ROUTES, else DefaultRouteKinds.
func ResolveRouteKinds(explicit string) ([]clientendpoint.RouteKind, error) {
	if strings.TrimSpace(explicit) == "" {
		explicit = os.Getenv(RoutesEnvVar)
	}
	if strings.TrimSpace(explicit) == "" {
		return DefaultRouteKinds(), nil
	}
	return ParseRouteKinds(explicit)
}

// ParseRouteKinds parses a comma-separated route name list ("local-unix,ssh",
// "direct", "all"). Unknown names fail with a readable error naming the
// accepted values; duplicates collapse while keeping the first position.
func ParseRouteKinds(value string) ([]clientendpoint.RouteKind, error) {
	var kinds []clientendpoint.RouteKind
	seen := map[clientendpoint.RouteKind]bool{}
	add := func(kind clientendpoint.RouteKind) {
		if !seen[kind] {
			seen[kind] = true
			kinds = append(kinds, kind)
		}
	}
	for _, raw := range strings.Split(value, ",") {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if name == "all" || name == "default" {
			for _, kind := range DefaultRouteKinds() {
				add(kind)
			}
			continue
		}
		kind, ok := routeKindAliases[name]
		if !ok {
			return nil, fmt.Errorf("unknown route %q (want local-unix, ssh-webrtc-tcp, direct-webrtc-tcp, managed-webrtc or all)", raw)
		}
		add(kind)
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("route list %q enables no transport", value)
	}
	return kinds, nil
}

// normalizeRouteKinds returns the effective kinds, substituting the default
// policy for an empty list.
func normalizeRouteKinds(kinds []clientendpoint.RouteKind) []clientendpoint.RouteKind {
	if len(kinds) == 0 {
		return DefaultRouteKinds()
	}
	return append([]clientendpoint.RouteKind(nil), kinds...)
}

// routeKindEnabled reports whether kind is part of the policy.
func routeKindEnabled(kinds []clientendpoint.RouteKind, kind clientendpoint.RouteKind) bool {
	for _, enabled := range kinds {
		if enabled == kind {
			return true
		}
	}
	return false
}

// RouteKindsLabel renders the effective policy for logs ("local-unix,ssh").
func RouteKindsLabel(kinds []clientendpoint.RouteKind) string {
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, string(kind))
	}
	return strings.Join(names, ",")
}
