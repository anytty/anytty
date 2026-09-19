package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anytty/anytty/access/accessrun"
	corev2 "github.com/anytty/anytty/daemon/core"
	daemonprovider "github.com/anytty/anytty/daemon/provider"
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
	cmd.AddCommand(v3DaemonCommand(socket, logFile, configPath))
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

func v3DaemonCommand(socket *string, logFile *string, configPath *string) *cobra.Command {
	var runDaemon func(*cobra.Command, []string) error
	var directListen string
	command := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the current-user core-v2 daemon",
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
	runDaemon = func(cmd *cobra.Command, args []string) error {
		logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
		if err != nil {
			return err
		}
		defer closeLogger()

		if address := strings.TrimSpace(os.Getenv("ANYTTY_DIRECT_LISTEN")); address != "" {
			logger.Warn("Direct listener is served by anytty-access; daemon only records this address for pairing route defaults", "route", address)
		}
		socketPath := resolveV3Socket(*socket)
		applyDaemonRuntimeTuning(logger)
		runtimeConfig, err := loadDaemonRuntimeConfig(*configPath)
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
		releaseRecord, err := acquireDaemonRuntimeRecord(socketPath, logPath, *configPath)
		if err != nil {
			return err
		}
		defer releaseRecord()
		historyEnabled := !envBool("ANYTTY_HISTORY_DISABLE")
		if historyEnabled {
			removed, err := corev2.DeleteObsoleteCompactHistory(resolveV3ObsoleteCompactHistoryDir())
			if err != nil {
				return fmt.Errorf("discard obsolete compact history: %w", err)
			}
			if removed > 0 {
				logger.Info("discarded obsolete compact history", "files", removed)
			}
			if err := corev2.PrepareHistoryStorage(historyDir, historyStorage); err != nil {
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
		stopPerfTrace, perfTracePath, perfTraceEnabled := perftrace.EnableFromEnvWithProcess(ctx, "core-v2-daemon")
		defer stopPerfTrace()
		if perfTraceEnabled {
			logger.Info("core-v2 daemon perftrace enabled", "path", perfTracePath)
		}
		writeHeapProfile := startDaemonHeapProfiler(ctx, logger)
		defer func() {
			_ = srv.Shutdown(context.Background())
		}()
		logger.Info("starting core-v2 daemon", "socket", providerSocket, "client_socket", socketPath, "log_file", logPath, "history_dir", historyDir, "history_enabled", historyEnabled, "history_max_bytes_per_terminal", historyStorage.MaxBytesPerTerminal, "history_max_age", historyStorage.MaxAge, "history_compression", historyStorage.Compression, "history_compression_level", historyStorage.CompressionLevel)
		if err := srv.Start(ctx); err != nil {
			logger.Error("core-v2 terminal server failed to start", "error", err)
			return err
		}
		coreServer, isCoreServer := srv.(*corev2.Server)
		if !isCoreServer {
			// 测试注入的 server 自行管理生命周期。
			writeHeapProfile("exit")
			logger.Info("core-v2 daemon exited")
			return nil
		}
		providerServer, err := daemonprovider.New(coreServer, daemonprovider.Config{Socket: providerSocket, Logger: logger})
		if err != nil {
			logger.Error("terminal provider server failed", "error", err)
			return err
		}
		defer func() { _ = providerServer.Shutdown(context.Background()) }()
		err = providerServer.ListenAndServe(ctx)
		writeHeapProfile("exit")
		if err != nil {
			logger.Error("core-v2 daemon exited with error", "error", err)
		} else {
			logger.Info("core-v2 daemon exited")
		}
		return err
	}
	command.RunE = runDaemon
	command.Args = cobra.NoArgs
	addDaemonLifecycleCommands(command, socket, logFile, configPath, runDaemon)
	return command
}

func v3PingCommand(socket *string, logFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Connect to the experimental core-v2 daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			socketPath := resolveV3Socket(*socket)
			client, err := dialOrStartV3Client(socketPath, logPath, logger)
			if err != nil {
				return err
			}
			if client != nil {
				defer client.Close()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "anytty v3 daemon ok: socket=%s\n", socketPath)
			return nil
		},
	}
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
				"anytty v3 tmux terminal smoke ok: terminal=%s session=%s input=%s artifact_dir=%s ansi=%s plain=%s daemon_log=%s socket=%s timeline=%s\n",
				result.TerminalID, result.Session, result.SentInput, result.ArtifactDir,
				result.ANSIPath, result.PlainPath, result.DaemonLog, result.SocketPath, result.TimelinePath,
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
				"anytty v3 tmux resize smoke ok: terminal=%s session=%s before=%s after=%s artifact_dir=%s ansi=%s plain=%s daemon_log=%s socket=%s timeline=%s\n",
				result.TerminalID, result.Session, result.BeforeSize, result.AfterSize, result.ArtifactDir,
				result.ANSIPath, result.PlainPath, result.DaemonLog, result.SocketPath, result.TimelinePath,
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
				"anytty v3 tmux ansi smoke ok: terminal=%s session=%s artifact_dir=%s ansi=%s plain=%s daemon_log=%s socket=%s timeline=%s\n",
				result.TerminalID, result.Session, result.ArtifactDir,
				result.ANSIPath, result.PlainPath, result.DaemonLog, result.SocketPath, result.TimelinePath,
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
