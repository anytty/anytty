package cli

import (
	"fmt"
	"strings"
	"time"

	corev2 "github.com/anytty/anytty/pool/core"
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
	command.AddCommand(newHistorySearchCommand(socket, logFile))
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
			release, err := acquirePoolRuntimeRecord(resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), *configPath)
			if err != nil {
				return fmt.Errorf("history deletion requires the terminal pool to be stopped: %w", err)
			}
			defer release()
			maintenance := corev2.NewHistoryMaintenance(resolveV3HistoryStorageDir())
			var removed int
			if all {
				removed, err = maintenance.DeleteAll()
				if err == nil {
					var obsoleteRemoved int
					obsoleteRemoved, err = maintenance.DeleteObsolete(resolveV3ObsoleteCompactHistoryDir())
					removed += obsoleteRemoved
				}
			} else {
				removed, err = maintenance.DeleteTerminal(strings.TrimSpace(args[0]))
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
			cfg, err := loadPoolRuntimeConfig(*configPath)
			if err != nil {
				return err
			}
			release, err := acquirePoolRuntimeRecord(resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), *configPath)
			if err != nil {
				return fmt.Errorf("manual history pruning requires the terminal pool to be stopped: %w", err)
			}
			defer release()
			storage := corev2.HistoryStorageConfig{
				MaxBytesPerTerminal: int64(cfg.History.MaxSizeMB) << 20,
				MaxAge:              time.Duration(cfg.History.MaxAgeDays) * 24 * time.Hour,
				Compression:         cfg.History.Compression,
				CompressionLevel:    cfg.History.CompressionLevel,
			}
			maintenance := corev2.NewHistoryMaintenance(resolveV3HistoryStorageDir())
			if _, err := maintenance.DeleteObsolete(resolveV3ObsoleteCompactHistoryDir()); err != nil {
				return err
			}
			if err := maintenance.Prepare(storage); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "history retention applied")
			return nil
		},
	}
}
