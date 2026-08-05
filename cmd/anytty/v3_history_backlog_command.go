package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
)

type v3HistoryBacklogConfig struct {
	TerminalID string
	OutPath    string
}

func v3HistoryBacklogCommand(socket *string, logFile *string) *cobra.Command {
	cfg := v3HistoryBacklogConfig{}
	cmd := &cobra.Command{
		Use:   "history-backlog <terminal-id>",
		Short: "Dump core-v2 history backlog diagnostics",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.TerminalID = strings.TrimSpace(args[0])
			if cfg.TerminalID == "" {
				return fmt.Errorf("terminal id is required")
			}
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			application, closeApplication, err := openV3HistoryCommandApplication(resolveV3Socket(*socket), logPath, logger)
			if err != nil {
				return err
			}
			defer closeApplication()
			if cfg.OutPath == "" {
				return runV3HistoryBacklog(cmd.Context(), application, cfg, cmd.OutOrStdout())
			}
			file, err := os.Create(cfg.OutPath)
			if err != nil {
				return err
			}
			defer file.Close()
			if err := runV3HistoryBacklog(cmd.Context(), application, cfg, file); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "anytty v3 history backlog ok: terminal=%s out=%s\n", cfg.TerminalID, cfg.OutPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&cfg.OutPath, "out", "o", "", "file path to write the history backlog TSV")
	return cmd
}

func openV3HistoryCommandApplication(socketPath string, logPath string, logger *slog.Logger) (*clientruntime.ApplicationSession, func(), error) {
	client, err := dialOrStartV3Client(socketPath, logPath, logger)
	if err != nil {
		return nil, nil, err
	}
	application, err := newLocalApplicationSession(client)
	if err != nil {
		if client != nil {
			_ = client.Close()
		}
		return nil, nil, err
	}
	return application, func() {
		if client != nil {
			_ = client.Close()
		}
	}, nil
}

func runV3HistoryBacklog(ctx context.Context, client *clientruntime.ApplicationSession, cfg v3HistoryBacklogConfig, writer io.Writer) error {
	if client == nil {
		return fmt.Errorf("nil core-v2 protocol client")
	}
	if strings.TrimSpace(cfg.TerminalID) == "" {
		return fmt.Errorf("terminal id is required")
	}
	status, err := client.HistoryBacklogStatus(ctx, &apipb.HistoryBacklogStatusCommand{Terminal: &apipb.TerminalRef{EndpointId: string(client.Stamp().EndpointID), TerminalId: cfg.TerminalID}})
	if err != nil {
		return err
	}
	writeV3HistoryBacklogTSV(writer, status)
	return nil
}

func writeV3HistoryBacklogTSV(writer io.Writer, status *apipb.HistoryBacklogStatusResult) {
	if status == nil {
		status = &apipb.HistoryBacklogStatusResult{}
	}
	fmt.Fprintln(writer, "terminal_id\thistory_enabled\toutput_buffer_policy\tbuffer_capacity_bytes\tresident_bytes\taggregate_resident_bytes\taggregate_budget_bytes\tdropped_bytes\tgap_count\toutput_buffer_wait_nanos\tunavailable\tunavailable_reason\tclosed")
	fmt.Fprintf(writer, "%s\t%t\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%t\t%s\t%t\n",
		status.GetTerminal().GetTerminalId(), status.GetHistoryEnabled(), status.GetOutputBufferPolicy(), status.GetBufferCapacityBytes(),
		status.GetResidentBytes(), status.GetAggregateResidentBytes(), status.GetAggregateBudgetBytes(), status.GetDroppedBytes(),
		status.GetGapCount(), status.GetOutputBufferWaitNanos(), status.GetUnavailable(), status.GetUnavailableReason(), status.GetClosed(),
	)
}
