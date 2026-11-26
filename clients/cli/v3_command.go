package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/anytty/anytty/access/accessrun"
	clientruntime "github.com/anytty/anytty/access/engine/runtime"
	endpointdomain "github.com/anytty/anytty/access/engine/endpoint"
	"github.com/anytty/anytty/proto/access/apipb"
	corev2 "github.com/anytty/anytty/pool/core"
	poolprovider "github.com/anytty/anytty/pool/provider"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/spf13/cobra"
)

type coreV2Server interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

var (
	newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
		return corev2.NewServer(opts...)
	}
)

func v3Command(socket *string, logFile *string, configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "v3",
		Short: "Run experimental core and tui commands",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(v3PoolCommand(socket, logFile, configPath))
	cmd.AddCommand(v3PingCommand(socket, logFile))
	cmd.AddCommand(v3TmuxSmokeCommand())
	cmd.AddCommand(v3TmuxTerminalSmokeCommand())
	cmd.AddCommand(v3TmuxResizeSmokeCommand())
	cmd.AddCommand(v3TmuxANSISmokeCommand())
	cmd.AddCommand(v3TmuxStabilitySmokeCommand())
	cmd.AddCommand(v3NewCommand(socket, logFile))
	cmd.AddCommand(v3LsCommand(socket, logFile))
	cmd.AddCommand(v3KillCommand(socket, logFile))
	cmd.AddCommand(v3RemoveCommand(socket, logFile))
	cmd.AddCommand(v3AttachCommand(socket, logFile))
	cmd.AddCommand(v3HistoryDumpCommand(socket, logFile))
	cmd.AddCommand(v3HistoryBacklogCommand(socket, logFile))
	return cmd
}

// v3PoolCommand is the `anytty pool` command group.
func v3PoolCommand(socket *string, logFile *string, configPath *string) *cobra.Command {
	return newPoolCommandGroup("pool", socket, logFile, configPath)
}

// v3DeprecatedDaemonCommand keeps `anytty daemon ...` working as a hidden alias
// of `anytty pool ...` while printing a one-line deprecation notice.
func v3DeprecatedDaemonCommand(socket *string, logFile *string, configPath *string) *cobra.Command {
	command := newPoolCommandGroup("daemon", socket, logFile, configPath)
	command.Hidden = true
	command.Short = "Deprecated alias for `anytty pool`"
	command.Long = "Deprecated alias for `anytty pool`. Use `anytty pool ...` instead."
	wrapPoolCommandDeprecation(command)
	return command
}

func wrapPoolCommandDeprecation(command *cobra.Command) {
	if command.RunE != nil {
		run := command.RunE
		command.RunE = func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "anytty daemon is deprecated; use `anytty pool` instead")
			return run(cmd, args)
		}
	}
	for _, child := range command.Commands() {
		wrapPoolCommandDeprecation(child)
	}
}

func newPoolCommandGroup(use string, socket *string, logFile *string, configPath *string) *cobra.Command {
	var runPool func(*cobra.Command, []string) error
	var directListen string
	command := &cobra.Command{
		Use:   use,
		Short: "Manage the current-user terminal pool",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			address := strings.TrimSpace(directListen)
			if address == "" {
				return nil
			}
			if err := validateDirectListenAddress(address); err != nil {
				return usageCLIError(err.Error())
			}
			return os.Setenv("ANYTTY_DIRECT_LISTEN", address)
		},
	}
	command.PersistentFlags().StringVar(&directListen, "route", "", "Direct listen HOST:PORT recorded for pairing route defaults; serve it with anytty-access --route")
	runPool = func(cmd *cobra.Command, args []string) error {
		logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
		if err != nil {
			return err
		}
		defer closeLogger()

		if address := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN")); address != "" {
			logger.Warn("Direct listener is served by anytty-access; terminal pool only records this address for pairing route defaults", "route", address)
		}
		socketPath := resolveV3Socket(*socket)
		applyPoolRuntimeTuning(logger)
		runtimeConfig, err := loadPoolRuntimeConfig(*configPath)
		if err != nil {
			return err
		}
		historyStorage := corev2.HistoryStorageConfig{
			MaxBytesPerTerminal: int64(runtimeConfig.History.MaxSizeMB) << 20,
			MaxAge:              time.Duration(runtimeConfig.History.MaxAgeDays) * 24 * time.Hour,
			Compression:         runtimeConfig.History.Compression,
			CompressionLevel:    runtimeConfig.History.CompressionLevel,
		}
		outputBuffer := corev2.TerminalOutputBufferConfig{
			CapacityBytes: runtimeConfig.OutputBuffer.CapacityBytes,
			Overflow:      corev2.TerminalOutputOverflowPolicy(runtimeConfig.OutputBuffer.Overflow),
		}
		resourceSampling := corev2.TerminalResourceSamplingConfig{
			Interval:   time.Duration(runtimeConfig.ResourceSampling.IntervalMS) * time.Millisecond,
			MaxSamples: runtimeConfig.ResourceSampling.MaxSamples,
		}
		historyDir := resolveV3HistoryStorageDir()
		historyMaintenance := corev2.NewHistoryMaintenance(historyDir)
		releaseRecord, err := acquirePoolRuntimeRecord(socketPath, logPath, *configPath)
		if err != nil {
			return err
		}
		defer releaseRecord()
		historyEnabled := !envBool("ANYTTY_HISTORY_DISABLE")
		if historyEnabled {
			removed, err := historyMaintenance.DeleteObsolete(resolveV3ObsoleteCompactHistoryDir())
			if err != nil {
				return fmt.Errorf("discard obsolete compact history: %w", err)
			}
			if removed > 0 {
				logger.Info("discarded obsolete compact history", "files", removed)
			}
			if err := historyMaintenance.Prepare(historyStorage); err != nil {
				return fmt.Errorf("prepare history storage: %w", err)
			}
		}
		providerSocket := accessrun.ProviderSocketPath(socketPath)
		opts := []corev2.ServerOption{corev2.WithLogger(logger), corev2.WithSocketPath(providerSocket), corev2.WithHistoryStorageDir(historyDir), corev2.WithHistoryStorageConfig(historyStorage), corev2.WithTerminalOutputBufferConfig(outputBuffer), corev2.WithTerminalResourceSamplingConfig(resourceSampling), corev2.WithTerminalOutputResidentBudget(runtimeConfig.OutputBuffer.ResidentBudgetBytes)}
		if !historyEnabled {
			historyDir = ""
			opts = []corev2.ServerOption{corev2.WithLogger(logger), corev2.WithSocketPath(providerSocket), corev2.WithHistoryDisabled(), corev2.WithTerminalOutputBufferConfig(outputBuffer), corev2.WithTerminalResourceSamplingConfig(resourceSampling), corev2.WithTerminalOutputResidentBudget(runtimeConfig.OutputBuffer.ResidentBudgetBytes)}
		}
		srv := newCoreV2Server(opts...)
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		stopPerfTrace, perfTracePath, perfTraceEnabled := perftrace.EnableFromEnvWithProcess(ctx, "core-v2-pool")
		defer stopPerfTrace()
		if perfTraceEnabled {
			logger.Info("terminal pool perftrace enabled", "path", perfTracePath)
		}
		writeHeapProfile := startPoolHeapProfiler(ctx, logger)
		defer func() {
			_ = srv.Shutdown(context.Background())
		}()
		logger.Info("starting terminal pool", "socket", providerSocket, "client_socket", socketPath, "log_file", logPath, "history_dir", historyDir, "history_enabled", historyEnabled, "history_max_bytes_per_terminal", historyStorage.MaxBytesPerTerminal, "history_max_age", historyStorage.MaxAge, "history_compression", historyStorage.Compression, "history_compression_level", historyStorage.CompressionLevel)
		if err := srv.Start(ctx); err != nil {
			logger.Error("core-v2 terminal server failed to start", "error", err)
			return err
		}
		coreServer, isCoreServer := srv.(*corev2.Server)
		if !isCoreServer {
			// 测试注入的 server 自行管理生命周期。
			writeHeapProfile("exit")
			logger.Info("terminal pool exited")
			return nil
		}
		providerServer, err := poolprovider.New(coreServer, poolprovider.Config{Socket: providerSocket, Logger: logger})
		if err != nil {
			logger.Error("terminal provider server failed", "error", err)
			return err
		}
		defer func() { _ = providerServer.Shutdown(context.Background()) }()
		err = providerServer.ListenAndServe(ctx)
		writeHeapProfile("exit")
		if err != nil {
			logger.Error("terminal pool exited with error", "error", err)
		} else {
			logger.Info("terminal pool exited")
		}
		return err
	}
	command.RunE = runPool
	command.Args = cobra.NoArgs
	addPoolLifecycleCommands(command, socket, logFile, configPath, runPool)
	return command
}

func v3PingCommand(socket *string, logFile *string) *cobra.Command {
	var endpointID, routeID string
	var count int
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check pool connectivity; with --endpoint measure routed round trips",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			if strings.TrimSpace(endpointID) == "" {
				socketPath := resolveV3Socket(*socket)
				client, dialErr := dialOrStartV3Client(socketPath, logPath, logger)
				if dialErr != nil {
					return dialErr
				}
				if client != nil {
					defer client.Close()
				}
				fmt.Fprintf(cmd.OutOrStdout(), "anytty v3 pool ok: socket=%s\n", socketPath)
				return nil
			}
			return pingRemoteEndpoint(cmd, endpointID, routeID, count, jsonOutput, *socket, *logFile, logger)
		},
	}
	cmd.Flags().StringVar(&endpointID, "endpoint", "", "endpoint ID or label to ping over its configured routes")
	cmd.Flags().StringVar(&routeID, "route", "", "force one route ID (cloud, direct, ...)")
	cmd.Flags().IntVar(&count, "count", 10, "round trips to measure")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "machine-readable output")
	return cmd
}

// pingRemoteEndpoint 通过真实路由测一次「请求→应答」的往返，用来把网络 RTT
// 和栈内开销分开。它只调用只读的 terminal.list，不改任何状态。
func pingRemoteEndpoint(cmd *cobra.Command, endpointID, routeID string, count int, jsonOutput bool, socketOverride, logFile string, logger *slog.Logger) error {
	if count <= 0 {
		count = 10
	}
	registry, err := loadNormalizedConnectionRegistry()
	if err != nil {
		return err
	}
	var target endpointdomain.Endpoint
	found := false
	for _, endpoint := range registry.Endpoints {
		if string(endpoint.ID) == endpointID || strings.EqualFold(endpoint.Label, endpointID) {
			target, found = endpoint, true
			break
		}
	}
	if !found {
		return usageCLIError(fmt.Sprintf("endpoint %q was not found", endpointID))
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(count)*5*time.Second+30*time.Second)
	defer cancel()
	client, route, err := connectEndpoint(ctx, target, endpointdomain.RouteID(strings.TrimSpace(routeID)), socketOverride, logFile, clientruntime.ConnectIntentProbe, "anytty-ping", logger)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	samples := make([]time.Duration, 0, count)
	for i := 0; i < count; i++ {
		started := time.Now()
		if _, err := client.TerminalList(ctx, &apipb.TerminalListCommand{}); err != nil {
			return err
		}
		samples = append(samples, time.Since(started))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	pick := func(q float64) time.Duration { return samples[int(float64(len(samples)-1)*q)] }
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			Kind    string `json:"kind"`
			Route   string `json:"route"`
			Path    string `json:"observed_path"`
			Count   int    `json:"count"`
			MinMS   int64  `json:"min_ms"`
			P50MS   int64  `json:"p50_ms"`
			P95MS   int64  `json:"p95_ms"`
			MaxMS   int64  `json:"max_ms"`
		}{
			Kind: "ping", Route: string(route.ID), Path: client.ObservedPath(), Count: len(samples),
			MinMS: samples[0].Milliseconds(), P50MS: pick(0.5).Milliseconds(), P95MS: pick(0.95).Milliseconds(), MaxMS: samples[len(samples)-1].Milliseconds(),
		})
	}
	fmt.Fprintf(cmd.OutOrStdout(), "route=%s path=%s n=%d min=%s p50=%s p95=%s max=%s\n",
		route.ID, client.ObservedPath(), len(samples), samples[0].Round(time.Millisecond), pick(0.5).Round(time.Millisecond), pick(0.95).Round(time.Millisecond), samples[len(samples)-1].Round(time.Millisecond))
	return nil
}

func v3TmuxSmokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tmux-smoke",
		Short: "Run a tmux black-box harness smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			anyttyBin, err := osExecutable()
			if err != nil {
				return err
			}
			result, err := runV3TmuxSmoke(cmd.Context(), anyttyBin)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux smoke ok: session=%s input=%s artifact_dir=%s ansi=%s plain=%s\n",
				result.Session, result.SentInput, result.ArtifactDir, result.ANSIPath, result.PlainPath,
			)
			return nil
		},
	}
}

func v3TmuxTerminalSmokeCommand() *cobra.Command {
	var anyttyBin string
	cmd := &cobra.Command{
		Use:   "tmux-terminal-smoke",
		Short: "Run a tmux black-box attach/input smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			if anyttyBin == "" {
				exe, err := osExecutable()
				if err != nil {
					return err
				}
				anyttyBin = exe
			}
			result, err := runV3TmuxTerminalSmoke(cmd.Context(), anyttyBin)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux terminal smoke ok: terminal=%s session=%s input=%s artifact_dir=%s ansi=%s plain=%s pool_log=%s socket=%s timeline=%s\n",
				result.TerminalID, result.Session, result.SentInput, result.ArtifactDir,
				result.ANSIPath, result.PlainPath, result.PoolLog, result.SocketPath, result.TimelinePath,
			)
			return nil
		},
	}
	cmd.Flags().StringVar(&anyttyBin, "anytty-bin", "", "anytty binary path to run inside tmux")
	return cmd
}

func v3TmuxResizeSmokeCommand() *cobra.Command {
	var anyttyBin string
	cmd := &cobra.Command{
		Use:   "tmux-resize-smoke",
		Short: "Run a tmux black-box resize propagation smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			if anyttyBin == "" {
				exe, err := osExecutable()
				if err != nil {
					return err
				}
				anyttyBin = exe
			}
			result, err := runV3TmuxResizeSmoke(cmd.Context(), anyttyBin)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux resize smoke ok: terminal=%s session=%s before=%s after=%s artifact_dir=%s ansi=%s plain=%s pool_log=%s socket=%s timeline=%s\n",
				result.TerminalID, result.Session, result.BeforeSize, result.AfterSize, result.ArtifactDir,
				result.ANSIPath, result.PlainPath, result.PoolLog, result.SocketPath, result.TimelinePath,
			)
			return nil
		},
	}
	cmd.Flags().StringVar(&anyttyBin, "anytty-bin", "", "anytty binary path to run inside tmux")
	return cmd
}

func v3TmuxANSISmokeCommand() *cobra.Command {
	var anyttyBin string
	cmd := &cobra.Command{
		Use:   "tmux-ansi-smoke",
		Short: "Run a tmux black-box ANSI/Unicode smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			if anyttyBin == "" {
				exe, err := osExecutable()
				if err != nil {
					return err
				}
				anyttyBin = exe
			}
			result, err := runV3TmuxANSISmoke(cmd.Context(), anyttyBin)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux ansi smoke ok: terminal=%s session=%s artifact_dir=%s ansi=%s plain=%s pool_log=%s socket=%s timeline=%s\n",
				result.TerminalID, result.Session, result.ArtifactDir,
				result.ANSIPath, result.PlainPath, result.PoolLog, result.SocketPath, result.TimelinePath,
			)
			return nil
		},
	}
	cmd.Flags().StringVar(&anyttyBin, "anytty-bin", "", "anytty binary path to run inside tmux")
	return cmd
}

func v3TmuxStabilitySmokeCommand() *cobra.Command {
	var anyttyBin string
	var rounds int
	cmd := &cobra.Command{
		Use:   "tmux-stability-smoke",
		Short: "Run a short tmux black-box stability smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			if anyttyBin == "" {
				exe, err := osExecutable()
				if err != nil {
					return err
				}
				anyttyBin = exe
			}
			result, err := runV3TmuxStabilitySmoke(cmd.Context(), anyttyBin, rounds)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux stability smoke ok: rounds=%d artifacts=%d artifact_dir=%s timeline=%s\n",
				result.Rounds, len(result.Artifacts), result.ArtifactDir, result.TimelinePath,
			)
			for _, artifact := range result.Artifacts {
				fmt.Fprintf(cmd.OutOrStdout(), "artifact: %s\n", artifact)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&anyttyBin, "anytty-bin", "", "anytty binary path to run inside tmux")
	cmd.Flags().IntVar(&rounds, "rounds", 1, "number of stability rounds")
	return cmd
}
