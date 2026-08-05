package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	localadapter "github.com/anytty/anytty/client/adapter/local"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
)

var (
	v3DialClient                 = dialV3Client
	startV3Daemon                = startCoreV2Daemon
	startV3DaemonWithConfig      = startCoreV2DaemonWithConfig
	connectV3EndpointApplication = connectCLIEndpointApplication
	osExecutable                 = os.Executable
)

func dialV3Client(path string) (*localadapter.ProtocolClient, error) {
	return dialV3ClientContext(context.Background(), path)
}

func dialV3ClientContext(ctx context.Context, path string) (*localadapter.ProtocolClient, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return localadapter.DialProtocolClientForComposition(ctx, path, "anytty-cli")
}

func dialOrStartV3Client(path, logFile string, logger *slog.Logger) (*protocoladapter.ApplicationClient, error) {
	return dialOrStartV3ClientWithConfig(path, logFile, "", logger)
}

func dialOrStartV3ClientWithConfig(path, logFile, configPath string, logger *slog.Logger) (*protocoladapter.ApplicationClient, error) {
	return connectLocalApplicationClient(context.Background(), path, logFile, configPath, logger)
}

func dialOrStartV3ClientContext(ctx context.Context, path, logFile string, logger *slog.Logger) (*protocoladapter.ApplicationClient, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return connectLocalApplicationClient(ctx, path, logFile, "", logger)
}

func dialLocalApplicationSession(ctx context.Context, path, logFile string) (*clientruntime.ApplicationSession, *protocoladapter.ApplicationClient, error) {
	client, err := dialOrStartV3ClientContext(ctx, path, logFile, nil)
	if err != nil {
		return nil, nil, err
	}
	application, err := newLocalApplicationSession(client)
	if err != nil {
		_ = client.Close()
		return nil, nil, err
	}
	return application, client, nil
}

func connectLocalApplicationClient(ctx context.Context, path, logFile, configPath string, logger *slog.Logger) (*protocoladapter.ApplicationClient, error) {
	registry := clientendpoint.DefaultRegistry()
	target, _ := registry.DefaultEndpoint()
	owner := clientruntime.NewSessionOwner()
	client, _, err := connectV3EndpointApplication(ctx, owner, target, clientendpoint.DefaultLocalRouteID, clientruntime.ConnectIntentInteractive, localadapter.Options{
		SocketOverride: path, DefaultSocket: resolveV3Socket(""), ClientName: "anytty-cli",
		Start: func(_ context.Context, socketPath string) error {
			if logger != nil {
				logger.Warn("core-v2 daemon dial failed; starting current daemon", "socket", socketPath)
			}
			if err := startCoreV2DaemonForConfig(socketPath, logFile, configPath); err != nil {
				return fmt.Errorf("start core-v2 daemon: %w", err)
			}
			return nil
		},
	}, logger)
	if err != nil {
		_ = owner.Close()
		return nil, err
	}
	return client, nil
}

func startCoreV2Daemon(path string, logFile string) error {
	return startCoreV2DaemonWithConfig(path, logFile, "")
}

func startCoreV2DaemonWithConfig(path string, logFile string, configPath string) error {
	cmd, err := buildStartCoreV2DaemonCommandWithConfig(path, logFile, configPath)
	if err != nil {
		return err
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	output := devNull
	if strings.TrimSpace(logFile) != "" {
		output, err = openPrivateDaemonLog(logFile)
		if err != nil {
			return err
		}
		defer output.Close()
	}
	cmd.Stdout = output
	cmd.Stderr = output
	configureDetachedCommand(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func startCoreV2DaemonForConfig(path string, logFile string, configPath string) error {
	if strings.TrimSpace(configPath) == "" {
		return startV3Daemon(path, logFile)
	}
	if startV3DaemonWithConfig == nil {
		return fmt.Errorf("core-v2 daemon config starter is nil")
	}
	// 中文说明：显式 --config 入口触发 auto-start 时，必须启动同一
	// config 的 daemon；普通 v3 auto-start 仍走可替换的 startV3Daemon。
	return startV3DaemonWithConfig(path, logFile, configPath)
}

func buildStartCoreV2DaemonCommand(path string, logFile string) (*exec.Cmd, error) {
	return buildStartCoreV2DaemonCommandWithConfig(path, logFile, "")
}

func buildStartCoreV2DaemonCommandWithConfig(path string, logFile string, configPath string) (*exec.Cmd, error) {
	exe, err := osExecutable()
	if err != nil {
		return nil, err
	}
	args := []string{"--socket", path}
	if logFile != "" {
		args = append(args, "--log-file", logFile)
	}
	if configPath = strings.TrimSpace(configPath); configPath != "" {
		args = append(args, "--config", configPath)
	}
	if envBool("ANYTTY_HISTORY_DISABLE") {
		cmd := exec.Command(exe, append(args, "daemon")...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ANYTTY_HISTORY_DISABLE=1")
		return cmd, nil
	}
	// 默认 daemon 已切换为 core-v2；自动启动不得落回 legacy daemon。
	args = append(args, "daemon")
	return exec.Command(exe, args...), nil
}
