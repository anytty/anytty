package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	localadapter "github.com/anytty/anytty/client/adapter/local"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
)

func openEndpointProtocolClient(ctx context.Context, endpoint endpointdomain.Endpoint, socketOverride, logFile string) (*protocoladapter.ApplicationClient, func(), error) {
	return openEndpointProtocolClientWithLogger(ctx, endpoint, socketOverride, logFile, nil)
}

func openEndpointProtocolClientWithLogger(ctx context.Context, endpoint endpointdomain.Endpoint, socketOverride, logFile string, logger *slog.Logger) (*protocoladapter.ApplicationClient, func(), error) {
	client, _, err := connectCLIEndpoint(ctx, endpoint, "", socketOverride, logFile, clientruntime.ConnectIntentInteractive, logger)
	if err != nil {
		return nil, func() {}, err
	}
	return client, func() { _ = client.Close() }, nil
}

func probeEndpointProtocolClient(ctx context.Context, endpoint endpointdomain.Endpoint, requestedRoute endpointdomain.RouteID, socketOverride, logFile string) (endpointdomain.RouteID, string, string, clientruntime.ConnectionSnapshot, bool, func(), error) {
	client, route, err := connectCLIEndpoint(ctx, endpoint, requestedRoute, socketOverride, logFile, clientruntime.ConnectIntentProbe, nil)
	if err != nil {
		return "", "", "", clientruntime.ConnectionSnapshot{}, false, func() {}, err
	}
	snapshot, valid := client.ConnectionSnapshot(time.Now().UTC())
	return route.ID, client.ObservedPath(), "only_viable", snapshot, valid, func() { _ = client.Close() }, nil
}

func connectCLIEndpoint(ctx context.Context, target endpointdomain.Endpoint, requested endpointdomain.RouteID, socketOverride, logFile string, intent clientruntime.ConnectIntent, logger *slog.Logger) (*protocoladapter.ApplicationClient, endpointdomain.AccessRoute, error) {
	owner := clientruntime.NewSessionOwner()
	client, route, err := connectV3EndpointApplication(ctx, owner, target, requested, intent, localadapter.Options{
		SocketOverride: socketOverride, DefaultSocket: resolveV3Socket(""), ClientName: "anytty-cli",
		Start: func(_ context.Context, path string) error {
			if err := startCoreV2DaemonForConfig(path, resolveV3LogFilePath(logFile), ""); err != nil {
				return fmt.Errorf("start core-v2 daemon: %w", err)
			}
			return nil
		},
	}, logger)
	if err != nil {
		_ = owner.Close()
	}
	return client, route, err
}
