package config

import (
	"fmt"
	"strings"
)

// Endpoint is one named launch target of the layout program.
//
// The v1 "command" model starts a local PTY whose argv is the endpoint
// command, so the same protocol path serves a plain shell, a container
// (`docker exec -it ...`) or a remote attach (`ssh host anytty attach ...`).
// The v2 "daemon" model connects the host to an already running anytty daemon
// over a transport (local-unix first): the shell sends kind/socket/
// connect_mode in terminal.create/attach and endpoint.sync, while the host
// speaks the daemon protocol directly (ENDPOINTS.zh-CN.md §2, §3).
type Endpoint struct {
	// Name is the protocol endpoint id (terminal:<endpoint>:<id>); it must
	// not contain ":" and should be stable across sessions.
	Name string `json:"name"`
	// Kind is "command" (default) or "daemon".
	Kind string `json:"kind,omitempty"`
	// Label is the display name in the picker; Name is used when empty.
	Label string `json:"label,omitempty"`
	// Argv is the endpoint command (command model), or the optional command
	// for a new daemon terminal. Empty daemon argv means "daemon default".
	Argv []string `json:"argv,omitempty"`
	// Cwd and Env override the host defaults for the spawned command.
	Cwd string            `json:"cwd,omitempty"`
	Env map[string]string `json:"env,omitempty"`
	// Socket is the daemon unix socket path for connect_mode local-unix.
	Socket string `json:"socket,omitempty"`
	// Address is the HOST:PORT peer for connect_mode tcp. The peer must
	// terminate at the daemon framed transport; typical values are an
	// `ssh -L 127.0.0.1:PORT:/remote/daemon.sock` forward or a loopback
	// bridge in front of the remote socket.
	Address string `json:"address,omitempty"`
	// ConnectMode is "local-unix" (default for daemon), "tcp",
	// "direct-webrtc-tcp" or "managed-webrtc". Recognized modes stay
	// configurable and fail at connect time with a readable error when
	// their transport parameters are missing.
	ConnectMode string `json:"connect_mode,omitempty"`
	// SignalingAddresses are the daemon embedded-signaling HOST:PORT
	// locators for connect_mode direct-webrtc-tcp.
	SignalingAddresses []string `json:"signaling_addresses,omitempty"`
	// ICETCPAddresses are the reachable ICE-TCP locators; empty means the
	// signaling locators.
	ICETCPAddresses []string `json:"ice_tcp_addresses,omitempty"`
	// DaemonDeviceID and DaemonFingerprint pin the remote daemon identity
	// (`anytty pair inspect --json`).
	DaemonDeviceID    string `json:"daemon_device_id,omitempty"`
	DaemonFingerprint string `json:"daemon_fingerprint,omitempty"`
	// CredentialDir and CredentialRef select the paired capability
	// credential (`anytty pair import`).
	CredentialDir string `json:"credential_dir,omitempty"`
	CredentialRef string `json:"credential_ref,omitempty"`
	// CloudGatewayAddress is the Cloud controller/client_gateway address
	// for connect_mode managed-webrtc.
	CloudGatewayAddress string `json:"cloud_gateway_address,omitempty"`
}

// DisplayName resolves the picker label.
func (e Endpoint) DisplayName() string {
	if label := strings.TrimSpace(e.Label); label != "" {
		return label
	}
	return e.Name
}

// KindName resolves the effective kind.
func (e Endpoint) KindName() string {
	if kind := strings.TrimSpace(e.Kind); kind != "" {
		return kind
	}
	return "command"
}

// ConnectModeName resolves the effective connect mode of a daemon endpoint.
func (e Endpoint) ConnectModeName() string {
	if e.KindName() != "daemon" {
		return ""
	}
	if mode := strings.TrimSpace(e.ConnectMode); mode != "" {
		return mode
	}
	return "local-unix"
}

// validateEndpoints checks the endpoint list: names are unique protocol ids,
// the kind is known and a command endpoint carries a non-empty argv.
func validateEndpoints(endpoints []Endpoint) error {
	seen := make(map[string]string, len(endpoints))
	for index, endpoint := range endpoints {
		path := fmt.Sprintf("endpoints[%d]", index)
		name := strings.TrimSpace(endpoint.Name)
		if name == "" {
			return fmt.Errorf("%s.name must not be empty", path)
		}
		if !validEndpointName(name) {
			return fmt.Errorf("%s.name %q: want letters, digits, dot, dash or underscore (it becomes terminal:<endpoint>:<id>)", path, name)
		}
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("%s.name %q duplicates %s", path, name, previous)
		}
		seen[name] = path
		switch endpoint.KindName() {
		case "command":
			if len(endpoint.Argv) == 0 {
				return fmt.Errorf("%s.argv must not be empty", path)
			}
		case "daemon":
			switch endpoint.ConnectModeName() {
			case "local-unix":
				if strings.TrimSpace(endpoint.Socket) == "" {
					return fmt.Errorf("%s.socket is required for connect_mode local-unix", path)
				}
			case "tcp":
				if strings.TrimSpace(endpoint.Address) == "" {
					return fmt.Errorf("%s.address is required for connect_mode tcp (HOST:PORT of the tunnel/bridge)", path)
				}
			case "direct-webrtc-tcp", "managed-webrtc":
				// Implemented transports whose parameters are validated at
				// connect time so the picker can still list the endpoint
				// and report a readable error.
			default:
				return fmt.Errorf("%s.connect_mode %q: want local-unix, tcp, direct-webrtc-tcp or managed-webrtc", path, endpoint.ConnectModeName())
			}
		default:
			return fmt.Errorf("%s.kind %q: want command or daemon", path, endpoint.KindName())
		}
		for argIndex, arg := range endpoint.Argv {
			if strings.ContainsAny(arg, "\r\n") {
				return fmt.Errorf("%s.argv[%d] must be a single-line argument", path, argIndex)
			}
		}
		if strings.ContainsAny(endpoint.Cwd, "\r\n") {
			return fmt.Errorf("%s.cwd must be a single-line path", path)
		}
		for key, value := range endpoint.Env {
			if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "=\r\n") {
				return fmt.Errorf("%s.env has invalid key %q", path, key)
			}
			if strings.ContainsAny(value, "\r\n") {
				return fmt.Errorf("%s.env[%s] must be a single-line value", path, key)
			}
		}
	}
	return nil
}

func validEndpointName(value string) bool {
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
