package cli

import (
	"fmt"
	"strings"

	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/proto/access/wire"
	"github.com/anytty/anytty/shared/runtimepath"
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

func resolveTUI2LocalSocket(registry endpointdomain.Registry, endpointID string) (string, bool, error) {
	endpoint, ok := registry.Endpoints[endpointdomain.EndpointID(endpointID)]
	if !ok {
		return "", false, nil
	}
	route, ok := endpoint.Route(endpointdomain.DefaultLocalRouteID)
	if !ok || !route.Enabled || route.Kind != endpointdomain.RouteLocalUnix {
		return "", false, nil
	}
	if strings.TrimSpace(route.Socket) == "" || route.Socket == "auto" {
		return resolveV3Socket(""), true, nil
	}
	return route.Socket, true, nil
}

func resolveV3LogFilePath(path string) string {
	return resolveLogFilePath(path)
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
