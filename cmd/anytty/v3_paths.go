package main

import (
	"fmt"
	"strings"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/runtimepath"
	tuiconfig "github.com/anytty/anytty/tui/config"
)

func resolveV3Socket(path string) string {
	if path != "" {
		return path
	}
	return runtimepath.SocketPath(fmt.Sprintf("anytty-v2-wire%d.sock", wire.Version))
}

func loadV3ConnectionRegistry() (endpointdomain.Registry, error) {
	return endpointdomain.Load("")
}

func resolveV3SocketForConnectionRegistry(path string, registry endpointdomain.Registry) (string, error) {
	return resolveV3SocketWithClientRuntime(path, registry)
}

func resolveV3SocketWithClientRuntime(path string, registry endpointdomain.Registry) (string, error) {
	if strings.TrimSpace(path) != "" {
		return resolveV3Socket(path), nil
	}
	endpoint, ok := registry.Endpoints[registry.Default]
	if !ok || !endpoint.Enabled {
		return "", fmt.Errorf("default endpoint %q is not available", registry.Default)
	}
	route, ok := endpoint.Route(endpointdomain.DefaultLocalRouteID)
	if !ok || !route.Enabled || route.Kind != endpointdomain.RouteLocalUnix {
		return "", fmt.Errorf("default endpoint %q is not a local unix endpoint", registry.Default)
	}
	if strings.TrimSpace(route.Socket) == "" || route.Socket == "auto" {
		return resolveV3Socket(""), nil
	}
	return route.Socket, nil
}

func resolveV3LogFilePath(path string) string {
	return resolveLogFilePath(path)
}

func v3ConfigPathPolicy() string {
	return tuiconfig.DefaultPath()
}

func v3StatePathPolicy() string {
	return "unused"
}

func resolveV3HistoryStorageDir() string {
	return resolveStateFilePath("history-v2")
}

func resolveV3ClipboardStoragePath() string {
	return resolveStateFilePath("clipboard-history.json")
}

func resolveV3ObsoleteCompactHistoryDir() string {
	return resolveStateFilePath("core-v2-history")
}
