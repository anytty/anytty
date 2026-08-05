package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/app"
	"github.com/anytty/anytty/tui/state"
	"github.com/spf13/cobra"
)

const v3RootTerminalID = "anytty-main"

type v3RootConfig struct {
	SocketPath         string
	LogFile            string
	TUIConfig          state.TUIConfigStore
	ConnectionRegistry endpointdomain.Registry
}

type v3RootEmptyConfig struct {
	SocketPath         string
	LogFile            string
	TUIConfig          state.TUIConfigStore
	ConnectionRegistry endpointdomain.Registry
}

var runV3Root = runV3RootRuntime
var runV3RootEmpty = runV3RootEmptyRuntime

func runV3RootCommand(cmd *cobra.Command, socket string, logFile string, configPath string) error {
	if !isInteractiveTerminal() {
		return fmt.Errorf("anytty TUI requires an interactive terminal; use `anytty --help` or subcommands like `new`, `ls`, `attach`, `kill`, `rm`, `daemon`")
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
	connectionRegistry, err := loadV3ConnectionRegistry()
	if err != nil {
		return err
	}
	socketPath, err := resolveV3SocketForConnectionRegistry(socket, connectionRegistry)
	if err != nil {
		return err
	}
	return runV3Root(ctx, v3RootConfig{
		SocketPath:         socketPath,
		LogFile:            resolveV3LogFilePath(logFile),
		TUIConfig:          tuiConfig,
		ConnectionRegistry: connectionRegistry,
	})
}

func runV3RootRuntime(ctx context.Context, cfg v3RootConfig) error {
	logger, closeLogger, logPath, err := openLogFileLogger(cfg.LogFile)
	if err != nil {
		return err
	}
	defer closeLogger()

	client, err := dialOrStartV3Client(cfg.SocketPath, logPath, logger)
	if err != nil {
		return err
	}
	terminalID, ok, err := selectV3RootAttachTerminal(ctx, client)
	closeErr := client.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if !ok {
		logger.Info("starting tui-v3 root empty command", "socket", cfg.SocketPath, "log_file", logPath)
		return runV3RootEmpty(ctx, v3RootEmptyConfig{
			SocketPath:         cfg.SocketPath,
			LogFile:            logPath,
			TUIConfig:          cfg.TUIConfig,
			ConnectionRegistry: cfg.ConnectionRegistry,
		})
	}
	logger.Info("starting tui-v3 root command", "terminal_id", terminalID, "socket", cfg.SocketPath, "log_file", logPath)
	return runV3Attach(ctx, v3AttachConfig{
		TerminalID:         terminalID,
		SocketPath:         cfg.SocketPath,
		LogFile:            logPath,
		TUIConfig:          cfg.TUIConfig,
		ConnectionRegistry: cfg.ConnectionRegistry,
	})
}

func runV3RootEmptyRuntime(ctx context.Context, cfg v3RootEmptyConfig) error {
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
	client, err := dialOrStartV3Client(cfg.SocketPath, logPath, logger)
	if err != nil {
		return err
	}
	defer client.Close()
	endpointApplications, err := newV3EndpointApplicationRouter(state.DefaultEndpointID, client)
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
	runtime := newV3InteractiveRuntimeWithOptions("", cols, rows, client, host, logger, v3InteractiveRuntimeOptions{
		SkipWorkbenchInitialLoad: true,
		InitialEndpointID:        state.DefaultEndpointID,
		TUIConfig:                cfg.TUIConfig,
		ConnectionRegistry:       cfg.ConnectionRegistry,
		EndpointContext:          ctx,
		EndpointApplications:     endpointApplications,
	})
	// root 空启动不创建 terminal；先让用户在 picker 中显式选择创建或连接。
	if err := runtime.Post(app.ShellOpenTerminalPickerMsg{}); err != nil {
		return err
	}
	if err := runtime.Run(ctx); err != nil {
		return err
	}
	return ctx.Err()
}

func selectV3RootAttachTerminal(ctx context.Context, client *protocoladapter.ApplicationClient) (string, bool, error) {
	application, err := newLocalApplicationSession(client)
	if err != nil {
		return "", false, err
	}
	list, err := application.TerminalList(ctx, &apipb.TerminalListCommand{})
	if err != nil {
		return "", false, fmt.Errorf("list core-v2 terminals for root: %w", err)
	}
	if id := selectV3RootTerminal(list.Terminals); id != "" {
		return id, true, nil
	}
	// 固定 root terminal 退出后仍会留在 core 里；root 入口只选择连接对象。
	// restart 必须是用户显式动作，不能在重进 TUI 时自动 HUP 旧 PTY。
	if item, ok := findV3RootTerminal(list.Terminals); ok {
		return item.GetRef().GetTerminalId(), true, nil
	}
	return "", false, nil
}

func selectV3RootTerminal(items []*apipb.TerminalInfo) string {
	for _, item := range items {
		if item.GetState() == apipb.TerminalState_TERMINAL_STATE_RUNNING {
			return item.GetRef().GetTerminalId()
		}
	}
	return ""
}

func findV3RootTerminal(items []*apipb.TerminalInfo) (*apipb.TerminalInfo, bool) {
	for _, item := range items {
		if item.GetRef().GetTerminalId() == v3RootTerminalID {
			return item, true
		}
	}
	return nil, false
}
