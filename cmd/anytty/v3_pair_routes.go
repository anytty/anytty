package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/remoteauthpb"
)

type v3PairRouteFlags struct {
	Routes              []string
	DefaultDirectListen string
	DirectID            string
	DirectName          string
	DirectAddresses     []string
	SignalingAddresses  []string
	ICETCPAddresses     []string
	ServerName          string
	SSHID               string
	SSHName             string
	SSHHost             string
	SSHPort             uint16
	SSHUser             string
	SSHHostKeys         []string
}

// v3PairingRoutes 把普通 flags 与严格 URI 收敛成唯一 generated Route contract。
// 未指定 Route 时只签入 Direct；显式 Cloud Route 可以作为首次配对 bootstrap。
func v3PairingRoutes(flags v3PairRouteFlags) ([]*remoteauthpb.EndpointRouteConfigV1, error) {
	specs := append([]string(nil), flags.Routes...)
	if len(specs) == 0 {
		if hasParameterizedDirectRoute(flags) || hasParameterizedSSHRoute(flags) {
			return nil, fmt.Errorf("Route fields require an explicit --route direct or --route ssh")
		}
		specs = []string{"direct"}
	} else {
		plainDirect, plainSSH := false, false
		for _, spec := range specs {
			plainDirect = plainDirect || strings.TrimSpace(spec) == "direct"
			plainSSH = plainSSH || strings.TrimSpace(spec) == "ssh"
		}
		if hasParameterizedDirectRoute(flags) && !plainDirect {
			return nil, fmt.Errorf("Direct Route fields require --route direct")
		}
		if hasParameterizedSSHRoute(flags) && !plainSSH {
			return nil, fmt.Errorf("SSH Route fields require --route ssh")
		}
	}
	routes := make([]*remoteauthpb.EndpointRouteConfigV1, 0, len(specs))
	plainDirect, plainSSH, hasCloud := false, false, false
	for _, raw := range specs {
		raw = strings.TrimSpace(raw)
		var route *remoteauthpb.EndpointRouteConfigV1
		var err error
		switch raw {
		case "direct":
			if plainDirect {
				return nil, fmt.Errorf("parameterized --route direct may only appear once; use direct://ID for multiple Direct Routes")
			}
			plainDirect = true
			route, err = v3ParameterizedDirectRoute(flags)
		case "cloud":
			if hasCloud {
				return nil, fmt.Errorf("--route cloud may only appear once")
			}
			hasCloud = true
			route = &remoteauthpb.EndpointRouteConfigV1{
				SchemaVersion: endpoint.RouteConfigVersion, RouteId: "cloud", Enabled: true,
				Route: &remoteauthpb.EndpointRouteConfigV1_ManagedWebrtc{ManagedWebrtc: &remoteauthpb.ManagedWebRTCRouteConfig{
					AccountProfileRef: "default", RelayMode: remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_AUTO,
				}},
			}
		case "ssh":
			if plainSSH {
				return nil, fmt.Errorf("parameterized --route ssh may only appear once; use ssh:// URI for multiple SSH Routes")
			}
			plainSSH = true
			route, err = v3ParameterizedSSHRoute(flags)
		default:
			route, err = v3PairingRouteURI(raw, flags.DefaultDirectListen)
		}
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	if len(routes) > 4 {
		return nil, fmt.Errorf("pair create accepts at most four Routes")
	}
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		if _, exists := seen[route.GetRouteId()]; exists {
			return nil, fmt.Errorf("duplicate Route ID %q", route.GetRouteId())
		}
		seen[route.GetRouteId()] = struct{}{}
	}
	return routes, nil
}

func hasParameterizedDirectRoute(flags v3PairRouteFlags) bool {
	return strings.TrimSpace(flags.DirectID) != "" || strings.TrimSpace(flags.DirectName) != "" || len(flags.DirectAddresses) != 0 || len(flags.SignalingAddresses) != 0 || len(flags.ICETCPAddresses) != 0 || strings.TrimSpace(flags.ServerName) != ""
}

func hasParameterizedSSHRoute(flags v3PairRouteFlags) bool {
	return strings.TrimSpace(flags.SSHID) != "" || strings.TrimSpace(flags.SSHName) != "" || strings.TrimSpace(flags.SSHHost) != "" || flags.SSHPort != 0 || strings.TrimSpace(flags.SSHUser) != "" || len(flags.SSHHostKeys) != 0
}

func v3ParameterizedDirectRoute(flags v3PairRouteFlags) (*remoteauthpb.EndpointRouteConfigV1, error) {
	route, err := v3DirectPairingRoute(v3DirectPairingRouteOptions{DefaultListen: flags.DefaultDirectListen, DirectAddresses: flags.DirectAddresses, SignalingAddresses: flags.SignalingAddresses, ICETCPAddresses: flags.ICETCPAddresses, ServerName: flags.ServerName})
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(flags.DirectID)
	if id != "" && id != "direct" {
		route.RouteId = "direct-" + id
	}
	route.DisplayName = strings.TrimSpace(flags.DirectName)
	return route, nil
}

func v3ParameterizedSSHRoute(flags v3PairRouteFlags) (*remoteauthpb.EndpointRouteConfigV1, error) {
	id := strings.TrimSpace(flags.SSHID)
	if id == "" {
		id = "ssh"
	} else if id != "ssh" {
		id = "ssh-" + id
	}
	return v3SSHRoute(id, flags.SSHName, flags.SSHHost, flags.SSHPort, flags.SSHUser, flags.SSHHostKeys)
}

func v3PairingRouteURI(raw, defaultDirectListen string) (*remoteauthpb.EndpointRouteConfigV1, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid --route %q; use direct, ssh, or a supported Route URI", raw)
	}
	switch parsed.Scheme {
	case "direct":
		if parsed.User != nil || parsed.Port() != "" || parsed.Path != "" {
			return nil, fmt.Errorf("Direct Route URI must use direct://ID")
		}
		if err := rejectUnknownPairQuery(parsed.Query(), "name", "address", "signaling", "ice", "server-name"); err != nil {
			return nil, err
		}
		id := strings.TrimSpace(parsed.Hostname())
		if id == "" {
			return nil, fmt.Errorf("Direct Route URI requires an ID")
		}
		route, err := v3DirectPairingRoute(v3DirectPairingRouteOptions{
			DefaultListen: defaultDirectListen, DirectAddresses: parsed.Query()["address"], SignalingAddresses: parsed.Query()["signaling"], ICETCPAddresses: parsed.Query()["ice"], ServerName: parsed.Query().Get("server-name"),
		})
		if err != nil {
			return nil, err
		}
		route.RouteId = "direct-" + id
		if id == "lan" {
			route.RouteId = "direct"
		}
		route.DisplayName = strings.TrimSpace(parsed.Query().Get("name"))
		return route, nil
	case "ssh":
		if parsed.Path != "" {
			return nil, fmt.Errorf("SSH Route URI must not contain a path")
		}
		if err := rejectUnknownPairQuery(parsed.Query(), "name", "fingerprint"); err != nil {
			return nil, err
		}
		if parsed.User == nil || parsed.User.Username() == "" || parsed.User.String() != parsed.User.Username() {
			return nil, fmt.Errorf("SSH Route URI requires a user and must not contain a password")
		}
		port := uint16(22)
		if value := parsed.Port(); value != "" {
			number, err := strconv.ParseUint(value, 10, 16)
			if err != nil || number == 0 {
				return nil, fmt.Errorf("SSH Route URI has an invalid port")
			}
			port = uint16(number)
		}
		host := parsed.Hostname()
		id := strings.ReplaceAll(net.JoinHostPort(host, strconv.Itoa(int(port))), ":", "-")
		return v3SSHRoute("ssh-"+id, parsed.Query().Get("name"), host, port, parsed.User.Username(), parsed.Query()["fingerprint"])
	default:
		return nil, fmt.Errorf("unsupported Route URI scheme %q", parsed.Scheme)
	}
}

func rejectUnknownPairQuery(values url.Values, allowed ...string) error {
	known := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		known[key] = struct{}{}
	}
	for key := range values {
		if _, ok := known[key]; !ok {
			return fmt.Errorf("unknown Route URI query %q", key)
		}
	}
	return nil
}

func v3SSHRoute(id, name, host string, port uint16, user string, hostKeys []string) (*remoteauthpb.EndpointRouteConfigV1, error) {
	host, user, name = strings.TrimSpace(host), strings.TrimSpace(user), strings.TrimSpace(name)
	if host == "" || user == "" || len(hostKeys) == 0 {
		return nil, fmt.Errorf("SSH Route requires --ssh-host, --ssh-user, and at least one --ssh-host-key")
	}
	if port == 0 {
		port = 22
	}
	pins := append([]string(nil), hostKeys...)
	for index := range pins {
		pins[index] = strings.TrimSpace(pins[index])
		if !strings.HasPrefix(pins[index], "SHA256:") {
			return nil, fmt.Errorf("SSH host key must be an SHA256 fingerprint")
		}
	}
	return &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: endpoint.RouteConfigVersion, RouteId: id, DisplayName: name, Enabled: true,
		Route: &remoteauthpb.EndpointRouteConfigV1_SshWebrtcTcp{SshWebrtcTcp: &remoteauthpb.SSHWebRTCTCPRouteConfig{
			Host: host, Port: uint32(port), User: user, HostKeyFingerprints: pins,
			CredentialDescriptor:   &remoteauthpb.EndpointCredentialDescriptor{DescriptorId: id + "-key", Kind: remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY},
			RemoteSignalingAddress: "127.0.0.1:41120", RemoteIceTcpAddress: "127.0.0.1:41121",
		}},
	}, nil
}
