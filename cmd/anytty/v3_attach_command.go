package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/app"
	tuiport "github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"github.com/anytty/anytty/tui/terminalhost"
	"github.com/spf13/cobra"
)

type v3TerminalHost interface {
	app.TerminalHost
	Enter(context.Context) error
	Close() error
	Size() (int, int, error)
}

type v3TerminalHostLogger interface {
	SetLogger(*slog.Logger)
}

type v3AttachConfig struct {
	EndpointID         state.EndpointID
	TerminalID         string
	SocketPath         string
	LogFile            string
	TUIConfig          state.TUIConfigStore
	ConnectionRegistry endpointdomain.Registry
}

var (
	newV3TerminalHost = func() v3TerminalHost {
		return terminalhost.New()
	}
	runV3Attach = runV3AttachRuntime
)

func v3AttachCommand(socket *string, logFile *string) *cobra.Command {
	return &cobra.Command{
		Use:  "attach <id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLocalAttachCommand(cmd, args[0], *socket, *logFile, "")
		},
	}
}

func runLocalAttachCommand(cmd *cobra.Command, terminalID, socket, logFile, configPath string) error {
	return runAttachCommand(cmd, string(state.DefaultEndpointID), terminalID, socket, logFile, configPath)
}

func runAttachCommand(cmd *cobra.Command, endpointID, terminalID, socket, logFile, configPath string) error {
	return runAttachCommandWithClientRuntime(cmd, endpointID, terminalID, socket, logFile, configPath)
}

func runV3AttachRuntime(ctx context.Context, cfg v3AttachConfig) error {
	logger, closeLogger, logPath, err := openLogFileLogger(cfg.LogFile)
	if err != nil {
		return err
	}
	defer closeLogger()
	stopPerfTrace, perfTracePath, perfTraceEnabled := perftrace.EnableFromEnvWithProcess(ctx, "tui-v3")
	defer stopPerfTrace()
	if perfTraceEnabled {
		logger.Info("tui-v3 perftrace enabled", "path", perfTracePath)
	}
	client, closeClients, err := openV3AttachProtocolClients(ctx, cfg, logPath, logger)
	if err != nil {
		return err
	}
	defer closeClients()
	endpointApplications, err := newV3EndpointApplicationRouter(cfg.EndpointID, client)
	if err != nil {
		return err
	}
	defer endpointApplications.Close()

	host := newV3TerminalHost()
	if loggerHost, ok := host.(v3TerminalHostLogger); ok {
		loggerHost.SetLogger(logger)
	}
	if err := host.Enter(ctx); err != nil {
		return err
	}
	defer host.Close()

	cols, rows, err := host.Size()
	if err != nil || cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	surfaceID := newV3RuntimeSurfaceID()
	runtime := newV3InteractiveRuntimeFromClientRuntime(cfg.TerminalID, cols, rows, client, host, logger, v3InteractiveRuntimeOptions{
		InitialEndpointID:    cfg.EndpointID,
		RuntimeSurfaceID:     surfaceID,
		TUIConfig:            cfg.TUIConfig,
		ConnectionRegistry:   cfg.ConnectionRegistry,
		EndpointContext:      ctx,
		EndpointApplications: endpointApplications,
		ClipboardStorage:     newV3ClipboardStorage(),
	})
	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		EndpointID:   cfg.EndpointID,
		TerminalID:   cfg.TerminalID,
		Cols:         cols,
		Rows:         rows,
		Mode:         "collaborator",
		ResizePolicy: state.TerminalResizeRoleFollower,
		SurfaceID:    surfaceID,
		ViewID:       state.TerminalPaneViewID(state.DefaultPaneID),
	}}); err != nil {
		return err
	}
	if err := runtime.Run(ctx); err != nil {
		return err
	}
	return ctx.Err()
}

func newV3InteractiveRuntime(terminalID string, cols int, rows int, client *protocoladapter.ApplicationClient, host app.TerminalHost, logger *slog.Logger) *app.AppRuntime {
	return newV3InteractiveRuntimeWithOptions(terminalID, cols, rows, client, host, logger, v3InteractiveRuntimeOptions{
		InitialEndpointID:  state.DefaultEndpointID,
		ConnectionRegistry: endpointdomain.DefaultRegistry(),
	})
}

type v3InteractiveRuntimeOptions struct {
	SkipWorkbenchInitialLoad bool
	InitialEndpointID        state.EndpointID
	RuntimeSurfaceID         string
	TUIConfig                state.TUIConfigStore
	ConnectionRegistry       endpointdomain.Registry
	EndpointContext          context.Context
	EndpointApplications     v3EndpointApplicationServices
	ClipboardStorage         tuiport.ClipboardStorageService
}

func newV3InteractiveRuntimeWithOptions(terminalID string, cols int, rows int, client *protocoladapter.ApplicationClient, host app.TerminalHost, logger *slog.Logger, opts v3InteractiveRuntimeOptions) *app.AppRuntime {
	return newV3InteractiveRuntimeFromClientRuntime(terminalID, cols, rows, client, host, logger, opts)
}

func openV3AttachProtocolClients(ctx context.Context, cfg v3AttachConfig, logPath string, logger *slog.Logger) (*protocoladapter.ApplicationClient, func(), error) {
	return openV3AttachProtocolClientsWithClientRuntime(ctx, cfg, logPath, logger)
}

func newV3RuntimeSurfaceID() string {
	return fmt.Sprintf("cmd/anytty-v3:%d:%d", os.Getpid(), time.Now().UnixNano())
}

func v3EmptyRootShell() state.ShellStore {
	shell := state.DefaultShell()
	shell.Workspace.Tabs[0].Panes[0] = state.PaneState{
		ID:     state.DefaultPaneID,
		Title:  "unconnected",
		Kind:   state.PaneEmpty,
		Active: true,
	}
	return shell
}
