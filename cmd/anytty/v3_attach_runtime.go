package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	clientruntimeadapter "github.com/anytty/anytty/tui/adapter/clientruntime"
	localstorage "github.com/anytty/anytty/tui/adapter/localstorage"
	tuiprotocol "github.com/anytty/anytty/tui/adapter/protocol"
	systemadapter "github.com/anytty/anytty/tui/adapter/system"
	"github.com/anytty/anytty/tui/app"
	tuiport "github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"github.com/spf13/cobra"
)

type v3EndpointApplicationServices interface {
	tuiport.TerminalService
	tuiport.PathService
	tuiport.CoreClient
	tuiport.NativeScreenSource
	tuiport.LiveScreenSource
	Close() error
}

func runAttachCommandWithClientRuntime(cmd *cobra.Command, endpointID, terminalID, socket, logFile, configPath string) error {
	if !isInteractiveTerminal() {
		return usageCLIError("anytty terminal attach requires an interactive terminal")
	}
	if err := rejectNestedTUI(); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	tuiConfig, err := loadV3TUIConfig(configPath)
	if err != nil {
		return err
	}
	registry, err := loadV3ConnectionRegistry()
	if err != nil {
		return err
	}
	resolvedID := state.NormalizeEndpointID(state.EndpointID(endpointID))
	if strings.TrimSpace(endpointID) == "" {
		resolvedID = state.EndpointID(registry.Default)
	}
	endpoint, ok := registry.Endpoints[endpointIDFromState(resolvedID)]
	if !ok {
		return &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", resolvedID)}
	}
	if !endpoint.Enabled {
		return &cliError{code: 4, message: fmt.Sprintf("endpoint %s is disabled", resolvedID)}
	}
	socketPath := strings.TrimSpace(socket)
	if socketPath != "" {
		socketPath = resolveV3Socket(socketPath)
	}
	return runV3Attach(ctx, v3AttachConfig{
		EndpointID: resolvedID, TerminalID: terminalID, SocketPath: socketPath, LogFile: resolveV3LogFilePath(logFile),
		TUIConfig: tuiConfig, ConnectionRegistry: registry,
	})
}

func newV3InteractiveRuntimeFromClientRuntime(terminalID string, cols, rows int, client *clientprotocol.ApplicationClient, host app.TerminalHost, logger *slog.Logger, opts v3InteractiveRuntimeOptions) *app.AppRuntime {
	endpointID := state.NormalizeEndpointID(opts.InitialEndpointID)
	if endpointID == "" {
		endpointID = state.DefaultEndpointID
	}
	var application *clientruntime.ApplicationSession
	var terminalClient tuiprotocol.ProtocolTerminalClient
	if client != nil {
		application = client.ApplicationSession
		terminalClient = client
	}
	terminalAdapter := tuiprotocol.ProtocolTerminalServiceAdapter{Client: terminalClient, Application: application}
	coreAdapter := tuiprotocol.ProtocolCoreClientAdapter{Application: application}
	pathAdapter, _ := tuiprotocol.NewProtocolPathServiceAdapter(application)
	var terminalService tuiport.TerminalService = terminalAdapter
	var coreService tuiport.CoreClient = coreAdapter
	var pathService tuiport.PathService = pathAdapter
	if opts.EndpointApplications != nil {
		terminalService = opts.EndpointApplications
		coreService = opts.EndpointApplications
		pathService = opts.EndpointApplications
	}
	var endpointEvents tuiport.EndpointEventSource
	var endpointConnections tuiport.EndpointConnectionService
	if client != nil && client.ConnectionRuntime() != nil {
		endpointEvents = clientruntimeadapter.EndpointEventSource{Runtime: client.ConnectionRuntime(), EndpointID: endpointID, EndpointIDs: registryEndpointIDs(opts.ConnectionRegistry)}
		if planRuntime, ok := client.ConnectionRuntime().(clientruntimeadapter.EndpointPlanSnapshotSource); ok {
			control := &clientruntimeadapter.EndpointConnectionControl{Runtime: planRuntime, InitialRegistry: opts.ConnectionRegistry}
			if diagnostics, ok := opts.EndpointApplications.(clientruntimeadapter.EndpointConnectionSnapshotSource); ok {
				control.Diagnostics = diagnostics
			}
			if lifecycle, ok := opts.EndpointApplications.(clientruntimeadapter.EndpointEnableLifecycle); ok {
				control.Lifecycle = lifecycle
			}
			endpointConnections = control
		}
	}

	initial := state.Root{
		RuntimeSurfaceID: opts.RuntimeSurfaceID,
		Session:          state.TerminalSessionStore{EndpointID: endpointID, TerminalID: terminalID, Cols: cols, Rows: rows},
		Surface:          state.TerminalSurfaceStore{EndpointID: endpointID, TerminalID: terminalID, Cols: cols, Rows: rows},
		Config:           opts.TUIConfig,
		Endpoints:        state.EndpointStore{}.ApplyConnectionRegistry(opts.ConnectionRegistry),
	}
	if initial.RuntimeSurfaceID == "" {
		initial.RuntimeSurfaceID = app.DefaultRuntimeSurfaceID
	}
	if terminalID == "" {
		initial.Shell = v3EmptyRootShell()
	}

	var workbench app.WorkbenchDeps
	if client != nil {
		workbench.Storage = tuiprotocol.ProtocolWorkbenchStorageAdapter{Application: client.ApplicationSession}
		workbench.Ref = state.DefaultWorkbenchStorageRef(state.DefaultWorkspaceID)
		workbench.Logger = logger
		workbench.SkipInitialLoad = opts.SkipWorkbenchInitialLoad
	}
	clipboard := app.ClipboardDeps{
		Storage: opts.ClipboardStorage,
		Ref:     state.DefaultClipboardStorageRef(state.DefaultWorkspaceID),
		Logger:  logger,
	}
	return app.NewInteractiveRuntimeWithStorage(
		initial, host, app.NewAsyncEffectRunner(),
		app.LiveDeps{Terminal: terminalService, Path: pathService, EndpointEvents: endpointEvents, EndpointConnections: endpointConnections, Logger: logger},
		app.CopyModeDeps{Core: coreService, Clipboard: &systemadapter.ClipboardService{}, Logger: logger, Rows: rows},
		workbench, clipboard,
	)
}

func newV3ClipboardStorage() tuiport.ClipboardStorageService {
	return localstorage.ClipboardStorage{Path: resolveV3ClipboardStoragePath()}
}

var newV3EndpointApplicationRouter = func(endpointID state.EndpointID, client *clientprotocol.ApplicationClient) (v3EndpointApplicationServices, error) {
	return clientruntimeadapter.NewEndpointApplicationRouter(endpointID, client)
}

func registryEndpointIDs(registry endpointdomain.Registry) []state.EndpointID {
	result := make([]state.EndpointID, 0, len(registry.Endpoints))
	for endpointID := range registry.Endpoints {
		result = append(result, state.EndpointID(endpointID))
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func openV3AttachProtocolClientsWithClientRuntime(ctx context.Context, cfg v3AttachConfig, logPath string, logger *slog.Logger) (*clientprotocol.ApplicationClient, func(), error) {
	endpointID := endpointIDFromState(state.NormalizeEndpointID(cfg.EndpointID))
	endpoint, ok := cfg.ConnectionRegistry.Endpoints[endpointID]
	if !ok {
		return nil, func() {}, fmt.Errorf("attach endpoint %q is not registered", endpointID)
	}
	application, closeClient, err := openTUIEndpointProtocolClientWithLogger(ctx, endpoint, cfg.SocketPath, logPath, logger)
	if err != nil {
		return nil, func() {}, err
	}
	// terminal/workbench/clipboard 是同一 endpoint session 上的独立 Proto consumer；
	// subscription resource 已经提供隔离，不得再为它们伪造三条 generation。
	return application, closeClient, nil
}

func endpointIDFromState(id state.EndpointID) endpointdomain.EndpointID {
	return endpointdomain.EndpointID(id)
}
