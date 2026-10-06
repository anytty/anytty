package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	localadapter "github.com/anytty/anytty/access/engine/adapter/local"
	protocoladapter "github.com/anytty/anytty/access/engine/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/access/engine/endpoint"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	poolprovider "github.com/anytty/anytty/pool/provider"
)

var (
	v3DialClient                 = dialV3Client
	startV3Pool                  = startCoreV2Pool
	startV3PoolWithConfig        = startCoreV2PoolWithConfig
	startV3Access                = startCoreV2Access
	connectV3EndpointApplication = connectCLIEndpointApplication
	osExecutable                 = os.Executable
)

func dialV3Client(path string) (*localadapter.ProtocolClient, error) {
	return dialV3ClientContext(context.Background(), path)
}

// probeV3Socket 做一次无副作用的 protocol Hello 探活。
// 它集中 v3DialClient 变量替换点，供 lifecycle/status 复用而不扩散 concrete helper。
func probeV3Socket(path string) error {
	client, err := v3DialClient(path)
	if err != nil {
		return err
	}
	return client.Close()
}

// probeProviderSocket 用 provider Hello 探测 pool provider socket。
func probeProviderSocket(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := poolprovider.Dial(ctx, path)
	if err != nil {
		return err
	}
	return client.Close()
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
	return connectLocalApplicationClientWithName(ctx, path, logFile, configPath, cliEndpointClientName, logger)
}

func connectLocalApplicationClientWithName(ctx context.Context, path, logFile, configPath, clientName string, logger *slog.Logger) (*protocoladapter.ApplicationClient, error) {
	registry := clientendpoint.DefaultRegistry()
	target, _ := registry.DefaultEndpoint()
	owner := clientruntime.NewSessionOwner()
	clientName = endpointRuntimeClientName(clientName)
	client, _, err := connectV3EndpointApplication(ctx, owner, target, clientendpoint.DefaultLocalRouteID, clientruntime.ConnectIntentInteractive, localadapter.Options{
		SocketOverride: path, DefaultSocket: resolveV3Socket(""), ClientName: clientName,
		Start: func(_ context.Context, socketPath string) error {
			if logger != nil {
				logger.Warn("core-v2 pool dial failed; starting current pool", "socket", socketPath)
			}
			if err := startV3LocalStackForConfig(socketPath, logFile, configPath); err != nil {
				return fmt.Errorf("start local access stack: %w", err)
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

func startCoreV2Pool(path string, logFile string) (int, error) {
	return startCoreV2PoolWithConfig(path, logFile, "")
}

// startV3LocalStackForConfig 先起 pool（terminal provider），再起 access
// （canonical 客户端入口）。Phase 3 后两者缺一不可，auto-start 必须成对。
// pool 已在运行时只补 access（例如 access 被单独停掉的半栈），并且本次调用如果
// 真的新起了 pool，access 失败必须回滚，不能留下半栈。
func startV3LocalStackForConfig(path string, logFile string, configPath string) error {
	status, _, statusErr := poolStatus(path, logFile, configPath)
	poolStarted := statusErr != nil || status.PID <= 0 ||
		(status.State != "running" && status.State != "starting")
	var startedPID int
	if poolStarted {
		pid, err := startCoreV2PoolForConfig(path, logFile, configPath)
		if err != nil {
			return err
		}
		startedPID = pid
	}
	if err := startV3Access(path, logFile); err != nil {
		if poolStarted {
			rollbackStartedPool(path, logFile, configPath, startedPID)
		}
		return err
	}
	return nil
}

// rollbackStartedPool 只回滚 startedPID 对应的 pool：并发 auto-start 时输掉
// record 竞争的一方不能停掉赢家。记录身份不复验通过才移除。
func rollbackStartedPool(socketPath, logFile, configPath string, startedPID int) {
	if startedPID <= 0 {
		return
	}
	_, record, err := poolStatus(socketPath, logFile, configPath)
	if err != nil || record.PID != startedPID {
		return
	}
	_ = stopPoolProcess(startedPID)
	deadline := time.Now().Add(5 * time.Second)
	for poolRecordProcessMatches(record) && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if !poolRecordProcessMatches(record) {
		removePoolRecord(socketPath)
	}
}

// startCoreV2Access 启动 access 子进程；它总是写自己的日志文件。
// 必须先等 pool provider 就绪，避免 auto-start 后首批终端命令 dial 失败。
// 走 managed 路径把 access PID/身份写回 pool record：否则 auto-start 拉起的
// access 对 pool stop/status/access stop 不可见，停栈时会留下孤儿进程。
func startCoreV2Access(path string, logFile string) error {
	_ = logFile
	if err := waitForProviderSocket(path, 5*time.Second); err != nil {
		return fmt.Errorf("pool provider did not become ready: %w", err)
	}
	return startManagedAccess(path)
}

func startCoreV2PoolWithConfig(path string, logFile string, configPath string) (int, error) {
	cmd, err := buildStartCoreV2PoolCommandWithConfig(path, logFile, configPath)
	if err != nil {
		return 0, err
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	output := devNull
	if strings.TrimSpace(logFile) != "" {
		output, err = openPrivatePoolLog(logFile)
		if err != nil {
			return 0, err
		}
		defer output.Close()
	}
	cmd.Stdout = output
	cmd.Stderr = output
	configureDetachedCommand(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return 0, err
	}
	return pid, nil
}

func startCoreV2PoolForConfig(path string, logFile string, configPath string) (int, error) {
	if strings.TrimSpace(configPath) == "" {
		return startV3Pool(path, logFile)
	}
	if startV3PoolWithConfig == nil {
		return 0, fmt.Errorf("core-v2 pool config starter is nil")
	}
	// 中文说明：显式 --config 入口触发 auto-start 时，必须启动同一
	// config 的 pool；普通 v3 auto-start 仍走可替换的 startV3Pool。
	return startV3PoolWithConfig(path, logFile, configPath)
}

func buildStartCoreV2PoolCommand(path string, logFile string) (*exec.Cmd, error) {
	return buildStartCoreV2PoolCommandWithConfig(path, logFile, "")
}

func buildStartCoreV2PoolCommandWithConfig(path string, logFile string, configPath string) (*exec.Cmd, error) {
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
		cmd := exec.Command(exe, append(args, "pool")...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "ANYTTY_HISTORY_DISABLE=1")
		return cmd, nil
	}
	// 默认 pool 已切换为 core-v2；自动启动不得落回 legacy pool。
	args = append(args, "pool")
	return exec.Command(exe, args...), nil
}
