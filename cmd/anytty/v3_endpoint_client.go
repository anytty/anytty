package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	return openEndpointProtocolClientWithName(ctx, endpoint, socketOverride, logFile, cliEndpointClientName, logger)
}

func openTUIEndpointProtocolClientWithLogger(ctx context.Context, endpoint endpointdomain.Endpoint, socketOverride, logFile string, logger *slog.Logger) (*protocoladapter.ApplicationClient, func(), error) {
	return openEndpointProtocolClientWithName(ctx, endpoint, socketOverride, logFile, tuiEndpointClientName, logger)
}

func openEndpointProtocolClientWithName(ctx context.Context, endpoint endpointdomain.Endpoint, socketOverride, logFile, clientName string, logger *slog.Logger) (*protocoladapter.ApplicationClient, func(), error) {
	client, _, err := connectEndpoint(ctx, endpoint, "", socketOverride, logFile, clientruntime.ConnectIntentInteractive, clientName, logger)
	if err != nil {
		return nil, func() {}, err
	}
	return client, func() { _ = client.Close() }, nil
}

func probeEndpointProtocolClient(ctx context.Context, endpoint endpointdomain.Endpoint, requestedRoute endpointdomain.RouteID, registryPath, socketOverride, logFile string) (endpointdomain.RouteID, string, string, clientruntime.ConnectionSnapshot, bool, func(), error) {
	client, route, err := connectEndpointWithRegistry(ctx, endpoint, requestedRoute, registryPath, socketOverride, logFile, clientruntime.ConnectIntentProbe, cliEndpointClientName, nil)
	if err != nil {
		return "", "", "", clientruntime.ConnectionSnapshot{}, false, func() {}, err
	}
	snapshot, valid := client.ConnectionSnapshot(time.Now().UTC())
	return route.ID, client.ObservedPath(), "only_viable", snapshot, valid, func() { _ = client.Close() }, nil
}

func connectCLIEndpoint(ctx context.Context, target endpointdomain.Endpoint, requested endpointdomain.RouteID, socketOverride, logFile string, intent clientruntime.ConnectIntent, logger *slog.Logger) (*protocoladapter.ApplicationClient, endpointdomain.AccessRoute, error) {
	return connectEndpoint(ctx, target, requested, socketOverride, logFile, intent, cliEndpointClientName, logger)
}

func connectEndpoint(ctx context.Context, target endpointdomain.Endpoint, requested endpointdomain.RouteID, socketOverride, logFile string, intent clientruntime.ConnectIntent, clientName string, logger *slog.Logger) (*protocoladapter.ApplicationClient, endpointdomain.AccessRoute, error) {
	return connectEndpointWithRegistry(ctx, target, requested, "", socketOverride, logFile, intent, clientName, logger)
}

func connectEndpointWithRegistry(ctx context.Context, target endpointdomain.Endpoint, requested endpointdomain.RouteID, registryPath, socketOverride, logFile string, intent clientruntime.ConnectIntent, clientName string, logger *slog.Logger) (*protocoladapter.ApplicationClient, endpointdomain.AccessRoute, error) {
	owner := clientruntime.NewSessionOwner()
	clientName = endpointRuntimeClientName(clientName)
	localOptions := localadapter.Options{
		SocketOverride: socketOverride, DefaultSocket: resolveV3Socket(""), ClientName: clientName,
		Start: func(_ context.Context, path string) error {
			if err := startCoreV2DaemonForConfig(path, resolveV3LogFilePath(logFile), ""); err != nil {
				return fmt.Errorf("start core-v2 daemon: %w", err)
			}
			return nil
		},
	}
	var client *protocoladapter.ApplicationClient
	var route endpointdomain.AccessRoute
	var err error
	if strings.TrimSpace(registryPath) == "" {
		client, route, err = connectV3EndpointApplication(ctx, owner, target, requested, intent, localOptions, logger)
	} else {
		client, route, err = connectCLIEndpointApplicationWithRegistry(ctx, owner, target, requested, intent, registryPath, localOptions, logger)
	}
	if err != nil {
		_ = owner.Close()
	}
	return client, route, err
}
