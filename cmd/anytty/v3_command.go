package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	apilayer "github.com/anytty/anytty/api_layer"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/localweb"
	"github.com/anytty/anytty/shared/perftrace"
	tuiv3 "github.com/anytty/anytty/tui"
	tuiapp "github.com/anytty/anytty/tui/app"
	"github.com/anytty/anytty/tui/render"
	"github.com/anytty/anytty/tui/terminalhost"
	"github.com/spf13/cobra"
)

type coreV2Server interface {
	ListenAndServe(context.Context) error
	Shutdown(context.Context) error
}

var (
	newCoreV2Server = func(opts ...corev2.ServerOption) coreV2Server {
		return corev2.NewServer(append([]corev2.ServerOption{corev2.WithApplicationExecutorFactory(apilayer.CoreApplicationExecutorFactory)}, opts...)...)
	}
	runTUIv3Smoke         = tuiv3.SmokeRun
	runTUIv3SmokeDetailed = tuiv3.SmokeRunDetailed
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
	cmd.AddCommand(v3SmokeCommand())
	cmd.AddCommand(v3VisualSnapshotCommand())
	cmd.AddCommand(v3E2ESmokeCommand())
	cmd.AddCommand(v3TmuxSmokeCommand())
	cmd.AddCommand(v3TmuxTerminalSmokeCommand())
	cmd.AddCommand(v3TmuxResizeSmokeCommand())
	cmd.AddCommand(v3TmuxANSISmokeCommand())
	cmd.AddCommand(v3TmuxEmojiDotsSmokeCommand())
	cmd.AddCommand(v3TmuxVisualCompareCommand())
	cmd.AddCommand(v3TmuxStabilitySmokeCommand())
	cmd.AddCommand(v3NewCommand(socket, logFile))
	cmd.AddCommand(v3LsCommand(socket, logFile))
	cmd.AddCommand(v3KillCommand(socket, logFile))
	cmd.AddCommand(v3RemoveCommand(socket, logFile))
	cmd.AddCommand(v3AttachCommand(socket, logFile))
	cmd.AddCommand(v3HistoryDumpCommand(socket, logFile))
	cmd.AddCommand(v3HistoryBacklogCommand(socket, logFile))
	cmd.AddCommand(v3PaneCommandAdapterCommand())
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
	command.PersistentFlags().StringVar(&directListen, "route", "", "Direct listen HOST:PORT; wildcard hosts enable paired-device LAN discovery")
	runDaemon = func(cmd *cobra.Command, args []string) error {
		logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
		if err != nil {
			return err
		}
		defer closeLogger()

		socketPath := resolveV3Socket(*socket)
		applyDaemonRuntimeTuning(logger)
		runtimeConfig, err := loadV3TUIConfig(*configPath)
		if err != nil {
			return err
		}
		historyStorage := corev2.HistoryStorageConfig{
			MaxBytesPerTerminal: int64(runtimeConfig.Daemon.History.MaxSizeMB) << 20,
			MaxAge:              time.Duration(runtimeConfig.Daemon.History.MaxAgeDays) * 24 * time.Hour,
			Compression:         runtimeConfig.Daemon.History.Compression,
			CompressionLevel:    runtimeConfig.Daemon.History.CompressionLevel,
		}
		outputBuffer := corev2.TerminalOutputBufferConfig{
			CapacityBytes: runtimeConfig.Daemon.OutputBuffer.CapacityBytes,
			Overflow:      corev2.TerminalOutputOverflowPolicy(runtimeConfig.Daemon.OutputBuffer.Overflow),
		}
		historyDir := resolveV3HistoryStorageDir()
		clientAccess, err := loadV3ClientAccessRuntime(socketPath)
		if err != nil {
			return fmt.Errorf("load daemon identity and client access store: %w", err)
		}
		defer clientAccess.Close()
		releaseRecord, err := acquireDaemonRuntimeRecord(socketPath, logPath, *configPath)
		if err != nil {
			return err
		}
		defer releaseRecord()
		accessService := v3ClientAccessService{identity: clientAccess.Identity, store: clientAccess.Store, defaultLabel: v3DefaultPairingLabel}
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
		cloudControl := &v3CloudRuntimeControl{}
		opts := []corev2.ServerOption{corev2.WithLogger(logger), corev2.WithSocketPath(socketPath), corev2.WithHistoryStorageDir(historyDir), corev2.WithHistoryStorageConfig(historyStorage), corev2.WithTerminalOutputBufferConfig(outputBuffer), corev2.WithTerminalOutputResidentBudget(runtimeConfig.Daemon.OutputBuffer.ResidentBudgetBytes), corev2.WithClientAccessService(accessService), corev2.WithRemoteService(cloudControl)}
		if !historyEnabled {
			historyDir = ""
			opts = []corev2.ServerOption{corev2.WithLogger(logger), corev2.WithSocketPath(socketPath), corev2.WithHistoryDisabled(), corev2.WithTerminalOutputBufferConfig(outputBuffer), corev2.WithTerminalOutputResidentBudget(runtimeConfig.Daemon.OutputBuffer.ResidentBudgetBytes), corev2.WithClientAccessService(accessService), corev2.WithRemoteService(cloudControl)}
		}
		srv := newCoreV2Server(opts...)
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		closePairing, err := startV3PairingListener(ctx, clientAccess, logger)
		if err != nil {
			return err
		}
		defer closePairing()
		directCore, ok := srv.(v3RemoteDaemonCore)
		if !ok {
			return fmt.Errorf("Direct WebRTC requires core-v2 scoped transport")
		}
		if localWebCore, supported := srv.(localweb.Core); supported {
			cloudControl.configureLocalWeb(localWebCore)
			defer cloudControl.closeLocalWeb()
		}
		closeDirect, err := startV3DirectDaemon(ctx, directCore, clientAccess, logger)
		if err != nil {
			return err
		}
		defer closeDirect()
		closeCloud, err := startV3CloudDaemon(ctx, directCore, clientAccess, logger, cloudControl)
		if err != nil {
			return err
		}
		defer closeCloud()
		stopPerfTrace, perfTracePath, perfTraceEnabled := perftrace.EnableFromEnvWithProcess(ctx, "core-v2-daemon")
		defer stopPerfTrace()
		if perfTraceEnabled {
			logger.Info("core-v2 daemon perftrace enabled", "path", perfTracePath)
		}
		writeHeapProfile := startDaemonHeapProfiler(ctx, logger)
		defer func() {
			_ = srv.Shutdown(context.Background())
		}()
		logger.Info("starting core-v2 daemon", "socket", socketPath, "log_file", logPath, "history_dir", historyDir, "history_enabled", historyEnabled, "history_max_bytes_per_terminal", historyStorage.MaxBytesPerTerminal, "history_max_age", historyStorage.MaxAge, "history_compression", historyStorage.Compression, "history_compression_level", historyStorage.CompressionLevel)
		err = srv.ListenAndServe(ctx)
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

func validateDirectListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("Direct route %q must be HOST:PORT", address)
	}
	if strings.TrimSpace(host) == "" {
		host = "0.0.0.0"
	}
	value, err := strconv.ParseUint(strings.TrimSpace(port), 10, 16)
	if err != nil || value == 0 {
		return fmt.Errorf("Direct route %q has an invalid port", address)
	}
	return nil
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

func v3SmokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "smoke",
		Short: "Run a non-interactive tui-v3 smoke render",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := runTUIv3SmokeDetailed(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "anytty v3 smoke ok: tui=%s cases=%d\n", tuiv3.ModuleName, len(result.Cases))
			for _, item := range result.Cases {
				fmt.Fprintf(cmd.OutOrStdout(), "case: %s lines=%d\n", item.Name, len(item.Frame.Lines))
				for _, line := range item.Frame.Lines {
					fmt.Fprintln(cmd.OutOrStdout(), line)
				}
			}
			return nil
		},
	}
}

func v3VisualSnapshotCommand() *cobra.Command {
	var ansi bool
	var caseName string
	cmd := &cobra.Command{
		Use:   "visual-snapshot",
		Short: "Render a tui-v3 visual review smoke case",
		RunE: func(cmd *cobra.Command, args []string) error {
			frame, err := v3VisualSmokeCaseFrame(cmd.Context(), caseName)
			if err != nil {
				return err
			}
			if ansi {
				return writeVisualSnapshotANSI(cmd.OutOrStdout(), frame)
			}
			for _, line := range frame.Lines {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&ansi, "ansi", false, "write the frame as an ANSI terminal screen repaint")
	cmd.Flags().StringVar(&caseName, "case", "visual-audit-current", "smoke case name to render")
	return cmd
}

func v3VisualSmokeCaseFrame(ctx context.Context, caseName string) (render.Frame, error) {
	caseName = strings.TrimSpace(caseName)
	if caseName == "" {
		caseName = "visual-audit-current"
	}
	result, err := runTUIv3SmokeDetailed(ctx)
	if err != nil {
		return render.Frame{}, err
	}
	for _, item := range result.Cases {
		if item.Name == caseName {
			return item.Frame, nil
		}
	}
	return render.Frame{}, fmt.Errorf("%s smoke case not found", caseName)
}

func writeVisualSnapshotANSI(writer io.Writer, frame render.Frame) error {
	return terminalhost.NewFrameSink(writer).WriteFrame(frame)
}

func v3E2ESmokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "e2e-smoke",
		Short: "Run a non-interactive local core-v2/tui-v3 end-to-end smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := runV3E2ESmoke(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 e2e smoke ok: terminal=%s frames=%d viewport=%dx%d session=%dx%d copy_cols=%d pane_commands=%d panes=%d active=%s zoom_checked=%v\n",
				result.TerminalID,
				result.Frames,
				result.ViewportCols,
				result.ViewportRows,
				result.SessionCols,
				result.SessionRows,
				result.CopyCols,
				result.PaneCommands,
				result.PaneCount,
				result.ActivePaneID,
				result.ZoomChecked,
			)
			return nil
		},
	}
}

func v3TmuxSmokeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tmux-smoke",
		Short: "Run a tmux black-box harness smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := runV3TmuxSmoke(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux smoke ok: session=%s input=%s artifact_dir=%s ansi=%s plain=%s\n",
				result.Session,
				result.SentInput,
				result.ArtifactDir,
				result.ANSIPath,
				result.PlainPath,
			)
			return nil
		},
	}
}

func v3TmuxTerminalSmokeCommand() *cobra.Command {
	var anyttyBin string
	cmd := &cobra.Command{
		Use:   "tmux-terminal-smoke",
		Short: "Run a tmux black-box terminal create/attach/input smoke",
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
				result.TerminalID,
				result.Session,
				result.SentInput,
				result.ArtifactDir,
				result.ANSIPath,
				result.PlainPath,
				result.DaemonLog,
				result.SocketPath,
				result.TimelinePath,
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
		Short: "Run a tmux black-box resize and layout smoke",
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
				"anytty v3 tmux resize smoke ok: terminal=%s session=%s window=%s before=%s after=%s artifact_dir=%s ansi=%s plain=%s daemon_log=%s socket=%s timeline=%s\n",
				result.TerminalID,
				result.Session,
				result.WindowSize,
				result.BeforeSize,
				result.AfterSize,
				result.ArtifactDir,
				result.ANSIPath,
				result.PlainPath,
				result.DaemonLog,
				result.SocketPath,
				result.TimelinePath,
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
		Short: "Run a tmux black-box ANSI/theme/live surface smoke",
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
				result.TerminalID,
				result.Session,
				result.ArtifactDir,
				result.ANSIPath,
				result.PlainPath,
				result.DaemonLog,
				result.SocketPath,
				result.TimelinePath,
			)
			return nil
		},
	}
	cmd.Flags().StringVar(&anyttyBin, "anytty-bin", "", "anytty binary path to run inside tmux")
	return cmd
}

func v3TmuxEmojiDotsSmokeCommand() *cobra.Command {
	var anyttyBin string
	cmd := &cobra.Command{
		Use:   "tmux-emoji-dots-smoke",
		Short: "Run a tmux black-box owner/follower emoji+dots smoke",
		RunE: func(cmd *cobra.Command, args []string) error {
			if anyttyBin == "" {
				exe, err := osExecutable()
				if err != nil {
					return err
				}
				anyttyBin = exe
			}
			result, err := runV3TmuxEmojiDotsSmoke(cmd.Context(), anyttyBin)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux emoji dots smoke ok: terminal=%s session=%s before=%s after=%s dots=%v artifact_dir=%s ansi=%s plain=%s daemon_log=%s socket=%s timeline=%s\n",
				result.TerminalID,
				result.Session,
				result.BeforeSize,
				result.AfterSize,
				result.DotsVisible,
				result.ArtifactDir,
				result.ANSIPath,
				result.PlainPath,
				result.DaemonLog,
				result.SocketPath,
				result.TimelinePath,
			)
			return nil
		},
	}
	cmd.Flags().StringVar(&anyttyBin, "anytty-bin", "", "anytty binary path to run inside tmux")
	return cmd
}

func v3TmuxVisualCompareCommand() *cobra.Command {
	var anyttyBin string
	cmd := &cobra.Command{
		Use:   "tmux-visual-compare",
		Short: "Capture the fixed visual review frame in tmux and compare it with the target baseline",
		RunE: func(cmd *cobra.Command, args []string) error {
			if anyttyBin == "" {
				exe, err := osExecutable()
				if err != nil {
					return err
				}
				anyttyBin = exe
			}
			result, err := runV3TmuxVisualCompare(cmd.Context(), anyttyBin)
			if err != nil {
				return err
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"anytty v3 tmux visual compare ok: session=%s mismatches=%d style_mismatches=%d stylemap_mismatches=%d artifact_dir=%s current_plain=%s current_ansi=%s target=%s diff=%s style=%s style_diff=%s current_stylemap=%s target_stylemap=%s stylemap_diff=%s\n",
				result.Session,
				result.Mismatches,
				result.StyleMismatches,
				result.StyleMapMismatches,
				result.ArtifactDir,
				result.CurrentPlainPath,
				result.CurrentANSIPath,
				result.TargetPath,
				result.DiffPath,
				result.StylePath,
				result.StyleDiffPath,
				result.CurrentStyleMapPath,
				result.TargetStyleMapPath,
				result.StyleMapDiffPath,
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
				result.Rounds,
				len(result.Artifacts),
				result.ArtifactDir,
				result.TimelinePath,
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

func v3PaneCommandAdapterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pane-command -- COMMAND",
		Short: "Parse a tui-v3 pane mini command without mutating runtime state",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pane-command requires a mini command")
			}
			parsed, err := tuiapp.ParsePaneMiniCommand(strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pane command: action=%s pane=%s source=%s\n", parsed.Action, parsed.Target.PaneID, parsed.Source)
			return nil
		},
	}
}
