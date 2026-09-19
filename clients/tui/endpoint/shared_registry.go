package endpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
)

// The shared Endpoint registry (client/endpoint) is the single connection
// source of truth for CLI, TUI and every native client. tui2 reads it
// read-only: it never writes, pairs or manages credentials (that stays with
// the CLI). tui2.json's endpoints[] remains a compatibility surface for
// command endpoints and the local-unix/tcp daemon dial it still owns; when
// both name the same daemon endpoint, the shared registry wins.
// See tui2/docs/CLIENT_SHARING.zh-CN.md.
//
// A dev build runs under an isolated XDG tree, so the production registry
// (~/.config/anytty/endpoints.yaml) is not on the default path. The host can
// therefore read extra explicit registries from TUI2_ENDPOINTS (or
// -endpoints): they are loaded before the default path and win by endpoint
// name, which lets a dev TUI see already paired production endpoints without
// re-pairing or copying the file.

// RegistryEnvVar names the explicit CLI-owned endpoints.yaml files the tui2
// host should read. The value is an OS path list (os.PathListSeparator: ":"
// on Unix, ";" on Windows). Explicit files are read before the default
// client/endpoint path; on duplicate endpoint names the earlier file wins.
// tui2 never writes these files.
const RegistryEnvVar = "TUI2_ENDPOINTS"

// SharedRegistryPath returns the CLI/TUI shared endpoints.yaml path.
func SharedRegistryPath() string {
	return clientendpoint.DefaultPath()
}

// RegistryPathsFromEnv resolves TUI2_ENDPOINTS into an ordered read list.
func RegistryPathsFromEnv() []string {
	return RegistryPaths(os.Getenv(RegistryEnvVar))
}

// RegistryPaths returns the ordered registry read list: explicit entries
// first (split on os.PathListSeparator, cleaned and de-duplicated), then the
// default CLI/TUI registry unless it is already listed. The list is never
// empty.
func RegistryPaths(explicit ...string) []string {
	paths := make([]string, 0, len(explicit)+1)
	seen := make(map[string]struct{}, len(explicit)+1)
	add := func(raw string) {
		for _, value := range filepath.SplitList(raw) {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			path := filepath.Clean(value)
			if _, duplicate := seen[path]; duplicate {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	for _, value := range explicit {
		add(value)
	}
	add(SharedRegistryPath())
	return paths
}

// LoadSharedEndpointConfigs loads every registry path in order and returns
// the projected daemon configs merged by endpoint name (earlier paths win).
// paths may be empty: TUI2_ENDPOINTS + the default path are resolved.
//
// The load is read-only and best effort: a missing explicit file, a corrupt
// file or a shadowed duplicate becomes a human-readable warning and the
// remaining registries stay usable, so a bad legacy registry can never crash
// the host. A missing default path stays silent (same as before the override
// existed).
func LoadSharedEndpointConfigs(paths ...string) ([]Config, []string) {
	if len(paths) == 0 {
		paths = RegistryPathsFromEnv()
	}
	var (
		configs  []Config
		warnings []string
	)
	owner := make(map[string]string, len(paths))
	for _, path := range paths {
		if !registryFileExists(path) {
			if !isDefaultRegistryPath(path) {
				warnings = append(warnings, fmt.Sprintf("endpoint registry %s: file does not exist (skipped)", path))
			}
			continue
		}
		registry, err := clientendpoint.Load(path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("endpoint registry %s: %v (skipped)", path, err))
			continue
		}
		for _, shared := range registry.List() {
			name := string(shared.ID)
			if first, exists := owner[name]; exists {
				warnings = append(warnings, fmt.Sprintf("endpoint %s: %s is shadowed by %s (earlier registry wins)", name, path, first))
				continue
			}
			owner[name] = path
			if cfg, ok := configFromSharedEndpoint(shared); ok {
				configs = append(configs, cfg)
			}
		}
	}
	return configs, warnings
}

// SharedConfigForEndpoint projects the shared registry entry with the same
// ID into the tui2 host Config. It resolves TUI2_ENDPOINTS + the default path
// (explicit files first). ok is false when no registry defines the endpoint
// or it has no route tui2 can list; a corrupt file is reported as an error
// only when no lower-priority registry defines the endpoint either, so a
// recoverable explicit file never hides a default entry.
func SharedConfigForEndpoint(name string) (Config, bool, error) {
	return SharedConfigForEndpointIn(RegistryPathsFromEnv(), name)
}

// SharedConfigForEndpointIn is SharedConfigForEndpoint with an explicit
// ordered path list (empty resolves TUI2_ENDPOINTS + default).
func SharedConfigForEndpointIn(paths []string, name string) (Config, bool, error) {
	shared, ok, err := sharedEndpointIn(paths, name)
	if err != nil || !ok {
		return Config{}, false, err
	}
	cfg, ok := configFromSharedEndpoint(shared)
	if !ok {
		return Config{}, false, nil
	}
	return cfg, true, nil
}

// sharedEndpointIn returns the first registry entry with the given name in
// path order (explicit registries first). File-level problems are skipped
// only while a lower-priority path may still define the endpoint; the first
// load error is returned together with ok=false when nothing matched, so the
// caller gets a readable diagnostic without losing the fallback merge.
func sharedEndpointIn(paths []string, name string) (clientendpoint.Endpoint, bool, error) {
	if len(paths) == 0 {
		paths = RegistryPathsFromEnv()
	}
	name = strings.TrimSpace(name)
	var firstErr error
	for _, path := range paths {
		if !registryFileExists(path) {
			continue
		}
		registry, err := clientendpoint.Load(path)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("endpoint registry %s: %w", path, err)
			}
			continue
		}
		shared, ok := registry.Endpoints[clientendpoint.EndpointID(name)]
		if !ok {
			continue
		}
		return shared, true, nil
	}
	return clientendpoint.Endpoint{}, false, firstErr
}

func sharedConfigForEndpointAt(path, name string) (Config, bool, error) {
	return SharedConfigForEndpointIn([]string{path}, name)
}

func isDefaultRegistryPath(path string) bool {
	return filepath.Clean(path) == filepath.Clean(SharedRegistryPath())
}

func registryFileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// configFromSharedEndpoint reduces one shared Endpoint to the single-route
// Config the v2 host understands. Preference is local-unix, then ssh, then
// direct, then managed cloud: an old registry that carries a webrtc/cloud
// route beside a unix/tcp/ssh route projects the dialable one, so the endpoint
// degrades instead of being listed as an unconfigured WebRTC route. A
// direct/managed-only endpoint still projects its route and stays listed (the
// planner reports a readable no-eligible-route state when it is not enabled).
func configFromSharedEndpoint(shared clientendpoint.Endpoint) (Config, bool) {
	cfg := Config{Kind: KindDaemon, Name: string(shared.ID), Label: shared.Label}
	routes := shared.RouteList()
	selectRoute := func(kind clientendpoint.RouteKind) (clientendpoint.AccessRoute, bool) {
		for _, route := range routes {
			if route.Kind == kind && route.Enabled {
				return route, true
			}
		}
		return clientendpoint.AccessRoute{}, false
	}
	applyIdentity := func() {
		cfg.DaemonDeviceID = shared.DaemonIdentity.DeviceID
		cfg.DaemonFingerprint = shared.DaemonIdentity.DeviceFingerprint
	}
	if route, ok := selectRoute(clientendpoint.RouteLocalUnix); ok {
		socket := strings.TrimSpace(route.Socket)
		if socket == "" || socket == "auto" {
			return Config{}, false
		}
		cfg.ConnectMode = ConnectLocalUnix
		cfg.Socket = socket
		applyIdentity()
		return cfg, true
	}
	if route, ok := selectRoute(clientendpoint.RouteSSHWebRTCTCP); ok {
		cfg.ConnectMode = ConnectSSHWebRTC
		cfg.CredentialRef = route.CredentialRef
		applyIdentity()
		return cfg, true
	}
	if route, ok := selectRoute(clientendpoint.RouteDirectWebRTCTCP); ok {
		cfg.ConnectMode = ConnectDirectWebRTC
		cfg.SignalingAddresses = append([]string(nil), route.SignalingAddresses...)
		cfg.ICETCPAddresses = append([]string(nil), route.ICETCPAddresses...)
		applyIdentity()
		return cfg, true
	}
	if route, ok := selectRoute(clientendpoint.RouteManagedWebRTC); ok {
		cfg.ConnectMode = ConnectManagedWebRTC
		cfg.CredentialRef = route.CredentialRef
		applyIdentity()
		return cfg, true
	}
	return Config{}, false
}
