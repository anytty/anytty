package endpoint

import (
	"fmt"
	"net"
	"strings"
)

// Kinds and connect modes of the v2 endpoint model (ENDPOINTS.zh-CN.md §2).
const (
	// KindCommand is the v1 command endpoint: a local PTY runs argv.
	KindCommand = "command"
	// KindDaemon connects to an already running anytty daemon.
	KindDaemon = "daemon"

	// ConnectLocalUnix dials a daemon transport over a unix socket on the
	// local machine. It is also the P0 remote path: the user forwards a
	// remote daemon socket with `ssh -L local.sock:remote.sock` and points
	// this endpoint at the forwarded local socket.
	ConnectLocalUnix = "local-unix"
	// ConnectDirectTCP dials host:port and speaks the same framed daemon
	// transport as local-unix. The TCP peer must terminate at the daemon
	// transport, e.g. an `ssh -L 127.0.0.1:PORT:/remote/daemon.sock` forward
	// or a byte-transparent bridge (socat) in front of the remote socket.
	// It is not the legacy WebRTC direct route.
	ConnectDirectTCP = "tcp"
	// ConnectDirectWebRTC is the shared-layer direct route
	// (direct-webrtc-tcp): signaling + ICE-TCP + DTLS capability auth. tui2
	// no longer implements the transport itself; the shared client layer
	// (client/adapter/direct) dials it from the CLI-owned endpoints.yaml
	// registry. A tui2 daemon endpoint in this mode fails with a readable
	// error until it is configured there (see tui2/docs/CLIENT_SHARING.zh-CN.md).
	ConnectDirectWebRTC = "direct-webrtc-tcp"
	// ConnectManagedWebRTC is the shared-layer managed Cloud route
	// (managed-webrtc): enrollment, route resolution and an Edge relay. It is
	// dialed only by the shared client layer from endpoints.yaml; tui2 keeps
	// the mode recognized so the endpoint stays listed (see
	// tui2/docs/CLIENT_SHARING.zh-CN.md).
	ConnectManagedWebRTC = "managed-webrtc"
	// ConnectSSHWebRTC is the shared-layer SSH route (ssh-webrtc-tcp). tui2
	// has no local representation for it: the full route (host, tunneled
	// signaling/ICE-TCP addresses, credentials) comes from the CLI-owned
	// registry and the shared SSH adapter dials it. The mode stays recognized
	// so the picker can list the endpoint.
	ConnectSSHWebRTC = "ssh-webrtc-tcp"
)

// Health values published on Source.health and observed through Manager.
const (
	HealthUnknown    = "unknown"
	HealthConnecting = "connecting"
	HealthOK         = "ok"
	HealthOffline    = "offline"
)

// Config is one endpoint as registered by the layout program. It mirrors the
// legacy endpoints.yaml entry (label/connect_mode/routes) reduced to the v2
// field set of ENDPOINTS.zh-CN.md §2.
type Config struct {
	// Name is the protocol endpoint id used in terminal:<endpoint>:<id>.
	Name string
	// Label is the display label from the shared registry (empty falls back
	// to Name). tui2.json compatibility endpoints carry no label.
	Label string
	// Kind is "command" (default) or "daemon".
	Kind string
	// Socket is the daemon unix socket path for connect_mode local-unix.
	Socket string
	// Address is the HOST:PORT peer for connect_mode tcp. It must terminate
	// at the daemon framed transport (ssh -L TCP->socket forward or a
	// transparent bridge); local-unix ignores it.
	Address string
	// ConnectMode is "local-unix" (default for daemon), "tcp",
	// "direct-webrtc-tcp" or "managed-webrtc".
	ConnectMode string
	// SignalingAddresses are the daemon embedded-signaling HOST:PORT
	// locators for connect_mode direct-webrtc-tcp. Address is accepted as
	// the shared signaling+ICE shorthand.
	SignalingAddresses []string
	// ICETCPAddresses are the reachable ICE-TCP HOST:PORT locators. Empty
	// means "same as the signaling locators".
	ICETCPAddresses []string
	// DaemonDeviceID and DaemonFingerprint pin the remote daemon identity;
	// both are required to verify signaling answers and capability grants.
	DaemonDeviceID    string
	DaemonFingerprint string
	// CredentialDir is the remoteauth credential store directory written by
	// `anytty pair import` (StateHome/anytty/remote-v2/credentials by
	// default) and CredentialRef names the entry inside it.
	CredentialDir string
	CredentialRef string
	// CloudGatewayAddress is the Cloud controller/client_gateway endpoint
	// for connect_mode managed-webrtc.
	CloudGatewayAddress string
}

// KindName resolves the effective kind.
func (c Config) KindName() string {
	if kind := strings.TrimSpace(c.Kind); kind != "" {
		return kind
	}
	return KindCommand
}

// ConnectModeName resolves the effective connect mode. Command endpoints have
// no transport and report an empty mode.
func (c Config) ConnectModeName() string {
	if c.KindName() != KindDaemon {
		return ""
	}
	if mode := strings.TrimSpace(c.ConnectMode); mode != "" {
		return mode
	}
	return ConnectLocalUnix
}

// DialTarget returns the human-readable dial target for error messages:
// the socket path for local-unix, host:port for tcp, the first signaling or
// gateway locator for the WebRTC modes, and an empty string for command
// endpoints.
func (c Config) DialTarget() string {
	switch c.ConnectModeName() {
	case ConnectDirectTCP:
		return strings.TrimSpace(c.Address)
	case ConnectDirectWebRTC:
		if locators := trimLocators(c.SignalingAddresses); len(locators) > 0 {
			return strings.Join(locators, ",")
		}
		return ConnectDirectWebRTC
	case ConnectManagedWebRTC:
		if address := strings.TrimSpace(c.CloudGatewayAddress); address != "" {
			return address
		}
		return ConnectManagedWebRTC
	case ConnectLocalUnix:
		return strings.TrimSpace(c.Socket)
	default:
		return ""
	}
}

// Validate rejects configurations the manager cannot represent. An
// unsupported connect mode is not a validation error: it is a recognized
// value that fails at dial time with a readable message, so the picker can
// still show the endpoint (webrtc is out of scope, tcp is implemented).
func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("endpoint name must not be empty")
	}
	switch c.KindName() {
	case KindCommand:
		return nil
	case KindDaemon:
		switch c.ConnectModeName() {
		case ConnectLocalUnix:
			if strings.TrimSpace(c.Socket) == "" {
				return fmt.Errorf("endpoint %q: connect_mode local-unix requires a socket path", c.Name)
			}
			return nil
		case ConnectDirectTCP:
			return validateTCPAddress(c)
		case ConnectDirectWebRTC, ConnectManagedWebRTC, ConnectSSHWebRTC:
			// Recognized shared-layer modes: their required fields are checked
			// when the shared endpoint is resolved at dial time so the endpoint
			// stays listed and the notice carries a readable, actionable error.
			return nil
		default:
			return fmt.Errorf("endpoint %q: unknown connect_mode %q", c.Name, c.ConnectModeName())
		}
	default:
		return fmt.Errorf("endpoint %q: unknown kind %q (want command or daemon)", c.Name, c.KindName())
	}
}

// validateTCPAddress requires an explicit HOST:PORT so a misconfigured tcp
// endpoint fails at config time instead of producing an opaque dial error.
func validateTCPAddress(c Config) error {
	address := strings.TrimSpace(c.Address)
	if address == "" {
		return fmt.Errorf("endpoint %q: connect_mode tcp requires an address (HOST:PORT)", c.Name)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil || strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
		return fmt.Errorf("endpoint %q: connect_mode tcp address %q must be HOST:PORT", c.Name, address)
	}
	return nil
}

// UnsupportedModeError returns a readable error for connect modes the shared
// client layer cannot represent at all, or nil when the mode is dialable
// through it (local-unix/tcp/direct-webrtc-tcp/managed-webrtc). Missing route
// parameters for direct/cloud are reported by the shared planner/validator at
// dial time so the endpoint stays listed.
func (c Config) UnsupportedModeError() error {
	switch c.ConnectModeName() {
	case ConnectLocalUnix, ConnectDirectTCP, ConnectDirectWebRTC, ConnectManagedWebRTC, ConnectSSHWebRTC:
		return nil
	default:
		return fmt.Errorf("endpoint %q: unsupported connect_mode %q", c.Name, c.ConnectModeName())
	}
}

// trimLocators drops empty entries and surrounding whitespace without
// changing order.
func trimLocators(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
