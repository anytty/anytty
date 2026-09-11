// A test-only stdio host with independently owned in-memory daemon transports.
// Never connects to, stops, or replaces the user's running daemon.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	apilayer "github.com/anytty/anytty/api_layer"
	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/core"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	root, err := os.MkdirTemp("", "anytty-ts-proto-test-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	apps := map[string]*clientruntime.ApplicationSession{}
	var clients []*protocol.Client
	var servers []*core.Server
	var workers sync.WaitGroup
	defer func() {
		cancel()
		for _, client := range clients {
			_ = client.Close()
		}
		for _, server := range servers {
			_ = server.Shutdown(context.Background())
		}
		workers.Wait()
	}()
	lifetimeCtx := ctx
	return sdk.ServeStdioRouted(ctx, os.Stdin, os.Stdout, func(ctx context.Context, id string, command *apipb.PluginCommand) (*apipb.PluginResult, error) {
		mu.Lock()
		app := apps[id]
		if app == nil {
			name := id
			if name == "" {
				name = "default"
			}
			server := core.NewServer(core.WithSocketPath(filepath.Join(root, name+".sock")), core.WithPluginRuntime("daemon-"+name, filepath.Join(root, name)), core.WithApplicationExecutorFactory(apilayer.CoreApplicationExecutorFactory))
			clientTransport, serverTransport := memory.NewPair()
			workers.Add(1)
			go func() { defer workers.Done(); _ = server.ServeTransport(lifetimeCtx, serverTransport) }()
			client := protocol.NewClient(clientTransport)
			if err := client.Hello(ctx, protocol.Hello{Version: wire.Version, Client: "sdk-ts-interop"}); err != nil {
				mu.Unlock()
				return nil, err
			}
			app, err = clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{EndpointID: endpoint.EndpointID(name), RouteID: "memory", Generation: 1}, client)
			if err != nil {
				mu.Unlock()
				return nil, err
			}
			apps[id] = app
			clients = append(clients, client)
			servers = append(servers, server)
		}
		mu.Unlock()
		result, err := app.Execute(ctx, &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_Plugin{Plugin: command}})
		if err != nil {
			return nil, err
		}
		return result.GetPlugin(), nil
	})
}
