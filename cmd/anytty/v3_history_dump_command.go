package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/spf13/cobra"
)

type v3HistoryDumpConfig struct {
	TerminalID string
	OutPath    string
	Cols       int
	Limit      int
}

func v3HistoryDumpCommand(socket *string, logFile *string) *cobra.Command {
	cfg := v3HistoryDumpConfig{Cols: 80, Limit: 512}
	cmd := &cobra.Command{
		Use:   "history-dump <terminal-id>",
		Short: "Dump authoritative terminal history rows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.TerminalID = strings.TrimSpace(args[0])
			if cfg.TerminalID == "" {
				return fmt.Errorf("terminal id is required")
			}
			if cfg.Cols <= 0 || cfg.Limit <= 0 {
				return fmt.Errorf("history dump cols and limit must be positive")
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
			writer := cmd.OutOrStdout()
			var file *os.File
			if cfg.OutPath != "" {
				file, err = os.Create(cfg.OutPath)
				if err != nil {
					return err
				}
				defer file.Close()
				writer = file
			}
			rows, err := runV3HistoryDump(cmd.Context(), application, cfg, writer)
			if err != nil {
				return err
			}
			if cfg.OutPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "anytty v3 history dump ok: terminal=%s rows=%d out=%s\n", cfg.TerminalID, rows, cfg.OutPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&cfg.OutPath, "out", "o", "", "file path to write authoritative history rows")
	cmd.Flags().IntVar(&cfg.Cols, "cols", 80, "history projection width")
	cmd.Flags().IntVar(&cfg.Limit, "limit", 512, "logical lines per page")
	return cmd
}

func runV3HistoryDump(ctx context.Context, application *clientruntime.ApplicationSession, cfg v3HistoryDumpConfig, writer io.Writer) (int, error) {
	if application == nil {
		return 0, fmt.Errorf("nil core-v2 protocol client")
	}
	terminal := &apipb.TerminalRef{EndpointId: string(application.Stamp().EndpointID), TerminalId: cfg.TerminalID}
	latest, err := application.HistoryWindow(ctx, &apipb.HistoryWindowCommand{
		Terminal: terminal, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols: int32(cfg.Cols), Limit: int32(cfg.Limit),
	})
	if err != nil {
		return 0, err
	}
	if latest.GetToken() == "" {
		return 0, fmt.Errorf("history dump did not receive a frozen token")
	}
	release := func() {
		_ = application.HistoryRelease(ctx, &apipb.HistoryReleaseCommand{Terminal: terminal, Token: latest.GetToken(), HistoryGeneration: latest.GetHistoryGeneration()})
	}
	defer release()

	request := &apipb.HistoryWindowCommand{
		Terminal: terminal, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDEST,
		Cols: int32(cfg.Cols), Limit: int32(cfg.Limit), Token: latest.GetToken(),
		HistoryGeneration: latest.GetHistoryGeneration(), BoundaryFirstLineId: latest.GetFirstLineId(), BoundaryLastLineId: latest.GetLastLineId(),
	}
	rowNumber := 0
	var previousLineID uint64
	for {
		page, pageErr := application.HistoryWindow(ctx, request)
		if pageErr != nil {
			return rowNumber, pageErr
		}
		for _, row := range page.GetRows() {
			fmt.Fprintf(writer, "%06d page_row=%d logical_line=%d row_in_line=%d text=%q\n",
				rowNumber, rowNumber, row.GetLogicalLineId(), row.GetRowInLine(), historyDumpRowText(row))
			rowNumber++
		}
		if !page.GetHasMore() {
			return rowNumber, nil
		}
		rows := page.GetRows()
		if len(rows) == 0 {
			return rowNumber, fmt.Errorf("history dump pagination reported more rows without a cursor")
		}
		last := rows[len(rows)-1]
		if last.GetLogicalLineId() == 0 || last.GetLogicalLineId() == previousLineID {
			return rowNumber, fmt.Errorf("history dump pagination did not advance")
		}
		previousLineID = last.GetLogicalLineId()
		request = &apipb.HistoryWindowCommand{
			Terminal: terminal, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER,
			Cols: int32(cfg.Cols), Limit: int32(cfg.Limit), Token: latest.GetToken(), HistoryGeneration: latest.GetHistoryGeneration(),
			BoundaryFirstLineId: latest.GetFirstLineId(), BoundaryLastLineId: latest.GetLastLineId(),
			AfterCursor: &apipb.HistoryCursor{
				LineId: last.GetLogicalLineId(), RowInLine: last.GetRowInLine(), Segment: last.GetSegment(),
			},
		}
	}
}

func historyDumpRowText(row *apipb.HistoryRow) string {
	if row == nil || row.GetRow() == nil {
		return ""
	}
	var text strings.Builder
	for _, cell := range row.GetRow().GetCells() {
		text.WriteString(cell.GetContent())
	}
	return text.String()
}
