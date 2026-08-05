package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var addDevelopmentCommands = func(*cobra.Command, *string, *string, *string) {}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cliExitCode(err))
	}
}

func newRootCmd() *cobra.Command {
	var socket string
	var logFile string
	var configPath string
	var globalTimeout time.Duration
	cmd := &cobra.Command{
		Use:           "anytty",
		Short:         "A terminal multiplexer for local and remote daemon endpoints",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runV3RootCommand(cmd, socket, logFile, configPath)
		},
	}
	cmd.PersistentFlags().StringVar(&socket, "socket", "", "socket path")
	cmd.PersistentFlags().StringVar(&logFile, "log-file", "", "log file path (default: $ANYTTY_LOG_FILE or XDG state dir)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "anytty config path (default: XDG config dir tui-v3.yaml)")
	cmd.PersistentFlags().DurationVar(&globalTimeout, "timeout", 0, "maximum duration for the complete command (0 disables)")
	cmd.AddCommand(v3DaemonCommand(&socket, &logFile, &configPath))
	terminalRuntime := terminalCommandRuntime{socket: &socket, logFile: &logFile, configPath: &configPath}
	cmd.AddCommand(newTerminalCommand(terminalRuntime))
	cmd.AddCommand(newTerminalAliasCommands(terminalRuntime)...)
	cmd.AddCommand(v3PairCommand(&socket, &logFile))
	cmd.AddCommand(v3AccessCommand(&socket, &logFile))
	cmd.AddCommand(cloudCommand(&socket, &logFile))
	cmd.AddCommand(newConfigCommand(&configPath, &socket, &logFile))
	cmd.AddCommand(newHistoryCommand(&socket, &logFile, &configPath))
	cmd.AddCommand(newEndpointCommand(&socket, &logFile))
	cmd.AddCommand(newFileCommand(&socket, &logFile))
	cmd.AddCommand(v3LicensesCommand())
	addDevelopmentCommands(cmd, &socket, &logFile, &configPath)
	wrapGlobalTimeout(cmd, &globalTimeout)
	return cmd
}

func wrapGlobalTimeout(command *cobra.Command, timeout *time.Duration) {
	if command.RunE != nil {
		run := command.RunE
		command.RunE = func(cmd *cobra.Command, args []string) error {
			if *timeout < 0 {
				return usageCLIError("--timeout cannot be negative")
			}
			if *timeout == 0 {
				return run(cmd, args)
			}
			// 根 context 是 endpoint dial、Hello 与 RPC 的共同截止时间；子命令只能进一步缩短它。
			ctx, cancel := context.WithTimeout(cmd.Context(), *timeout)
			defer cancel()
			cmd.SetContext(ctx)
			return run(cmd, args)
		}
	}
	for _, child := range command.Commands() {
		wrapGlobalTimeout(child, timeout)
	}
}
