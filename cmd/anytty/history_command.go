package main

import (
	"fmt"
	"strings"
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/spf13/cobra"
)

func newHistoryCommand(socket, logFile, configPath *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "history",
		Short: "Manage local terminal history files",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newHistoryDeleteCommand(socket, logFile, configPath))
	command.AddCommand(newHistoryPruneCommand(socket, logFile, configPath))
	return command
}

func newHistoryDeleteCommand(socket, logFile, configPath *string) *cobra.Command {
	var all bool
	command := &cobra.Command{
		Use:   "delete [terminal-id]",
		Short: "Delete local terminal history",
		Args: func(_ *cobra.Command, args []string) error {
			if all && len(args) != 0 {
				return usageCLIError("terminal-id and --all cannot be used together")
			}
			if !all && len(args) != 1 {
				return usageCLIError("provide one terminal-id or use --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			release, err := acquireDaemonRuntimeRecord(resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), *configPath)
			if err != nil {
				return fmt.Errorf("history deletion requires the daemon to be stopped: %w", err)
			}
			defer release()
			var removed int
			if all {
				removed, err = corev2.DeleteAllHistory(resolveV3HistoryStorageDir())
				if err == nil {
					var obsoleteRemoved int
					obsoleteRemoved, err = corev2.DeleteObsoleteCompactHistory(resolveV3ObsoleteCompactHistoryDir())
					removed += obsoleteRemoved
				}
			} else {
				removed, err = corev2.DeleteTerminalHistory(resolveV3HistoryStorageDir(), strings.TrimSpace(args[0]))
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deleted %d history file(s)\n", removed)
			return nil
		},
	}
	command.Flags().BoolVar(&all, "all", false, "delete history for every terminal")
	return command
}

func newHistoryPruneCommand(socket, logFile, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Apply configured history retention immediately",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadV3TUIConfig(*configPath)
			if err != nil {
				return err
			}
			release, err := acquireDaemonRuntimeRecord(resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), *configPath)
			if err != nil {
				return fmt.Errorf("manual history pruning requires the daemon to be stopped: %w", err)
			}
			defer release()
			storage := corev2.HistoryStorageConfig{
				MaxBytesPerTerminal: int64(cfg.Daemon.History.MaxSizeMB) << 20,
				MaxAge:              time.Duration(cfg.Daemon.History.MaxAgeDays) * 24 * time.Hour,
				Compression:         cfg.Daemon.History.Compression,
				CompressionLevel:    cfg.Daemon.History.CompressionLevel,
			}
			if _, err := corev2.DeleteObsoleteCompactHistory(resolveV3ObsoleteCompactHistoryDir()); err != nil {
				return err
			}
			if err := corev2.PrepareHistoryStorage(resolveV3HistoryStorageDir(), storage); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "history retention applied")
			return nil
		},
	}
}
