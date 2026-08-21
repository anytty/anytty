package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	historySearchPageLines = 512
	historySearchCols      = 4096
)

type historySearchCommandConfig struct {
	terminalID   string
	pattern      string
	mode         string
	fixed        bool
	glob         bool
	regexp       bool
	after        int
	before       int
	context      int
	maxCount     int
	color        string
	resolvedMode history.HistorySearchMode
}

type historySearchProtocol interface {
	HistoryWindow(context.Context, *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error)
	HistorySearch(context.Context, *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error)
	HistoryRelease(context.Context, *apipb.HistoryReleaseCommand) error
}

func newHistorySearchCommand(socket, logFile *string) *cobra.Command {
	cfg := historySearchCommandConfig{mode: "text", color: "auto"}
	command := &cobra.Command{
		Use:   "search <terminal-id> <pattern>",
		Short: "Search authoritative terminal history",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.terminalID = strings.TrimSpace(args[0])
			cfg.pattern = args[1]
			if err := cfg.validate(cmd); err != nil {
				return err
			}
			logger, closeLogger, logPath, err := openLogFileLogger(*logFile)
			if err != nil {
				return err
			}
			defer closeLogger()
			application, closeApplication, err := openV3HistoryCommandApplication(resolveV3Socket(*socket), logPath, logger)
			if err != nil {
				return classifyCLIError(err)
			}
			defer closeApplication()
			terminalRef := &apipb.TerminalRef{
				EndpointId: string(application.Stamp().EndpointID),
				TerminalId: cfg.terminalID,
			}
			matches, err := runHistorySearch(cmd.Context(), application, terminalRef, cfg, cmd.OutOrStdout())
			if err != nil {
				return classifyCLIError(err)
			}
			if matches == 0 {
				return &cliError{code: 1, message: "no matches"}
			}
			return nil
		},
	}
	flags := command.Flags()
	flags.StringVar(&cfg.mode, "mode", "text", "search mode: text, glob, or regex")
	flags.BoolVarP(&cfg.fixed, "fixed-strings", "F", false, "interpret the pattern as literal text")
	flags.BoolVarP(&cfg.regexp, "extended-regexp", "E", false, "interpret the pattern as a Go RE2 regular expression")
	flags.BoolVarP(&cfg.glob, "glob", "G", false, "interpret the pattern as a shell-style wildcard")
	flags.IntVarP(&cfg.after, "after-context", "A", 0, "print NUM lines after each match")
	flags.IntVarP(&cfg.before, "before-context", "B", 0, "print NUM lines before each match")
	flags.IntVarP(&cfg.context, "context", "C", 0, "print NUM lines before and after each match")
	flags.IntVarP(&cfg.maxCount, "max-count", "m", 0, "stop after NUM matching lines (0 is unlimited)")
	flags.StringVar(&cfg.color, "color", "auto", "highlight matches: auto, always, or never")
	return command
}

func (cfg *historySearchCommandConfig) validate(command *cobra.Command) error {
	if cfg.terminalID == "" {
		return usageCLIError("terminal id is required")
	}
	if cfg.pattern == "" {
		return usageCLIError("search pattern must not be empty")
	}
	shortcutCount := 0
	for _, enabled := range []bool{cfg.fixed, cfg.glob, cfg.regexp} {
		if enabled {
			shortcutCount++
		}
	}
	if shortcutCount > 1 || shortcutCount > 0 && command.Flags().Changed("mode") {
		return usageCLIError("--mode, -F, -G, and -E are mutually exclusive")
	}
	mode := history.HistorySearchMode(strings.ToLower(strings.TrimSpace(cfg.mode)))
	if cfg.fixed {
		mode = history.HistorySearchModeText
	} else if cfg.glob {
		mode = history.HistorySearchModeGlob
	} else if cfg.regexp {
		mode = history.HistorySearchModeRegex
	}
	if cfg.context < 0 || cfg.before < 0 || cfg.after < 0 {
		return usageCLIError("history search context must not be negative")
	}
	if command.Flags().Changed("context") {
		if command.Flags().Changed("before-context") || command.Flags().Changed("after-context") {
			return usageCLIError("--context cannot be combined with --before-context or --after-context")
		}
		cfg.before, cfg.after = cfg.context, cfg.context
	}
	if cfg.before+cfg.after >= historySearchPageLines {
		return usageCLIError("combined before and after context must be smaller than 512")
	}
	if cfg.maxCount < 0 {
		return usageCLIError("--max-count must not be negative")
	}
	switch cfg.color {
	case "auto", "always", "never":
	default:
		return usageCLIError("--color must be auto, always, or never")
	}
	if _, err := history.CompileHistorySearchPattern(mode, cfg.pattern); err != nil {
		return usageCLIError(err.Error())
	}
	cfg.resolvedMode = mode
	return nil
}

func runHistorySearch(ctx context.Context, client historySearchProtocol, terminalRef *apipb.TerminalRef, cfg historySearchCommandConfig, writer io.Writer) (int, error) {
	if client == nil || terminalRef == nil || terminalRef.GetTerminalId() == "" {
		return 0, fmt.Errorf("history search requires a terminal")
	}
	pattern, err := history.CompileHistorySearchPattern(cfg.resolvedMode, cfg.pattern)
	if err != nil {
		return 0, err
	}
	latest, err := client.HistoryWindow(ctx, &apipb.HistoryWindowCommand{
		Terminal: terminalRef,
		Mode:     apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
		Cols:     historySearchCols,
		Limit:    1,
	})
	if err != nil {
		return 0, err
	}
	if latest.GetToken() == "" {
		return 0, fmt.Errorf("history search did not receive a frozen token")
	}
	defer func() {
		releaseCtx := context.WithoutCancel(ctx)
		_ = client.HistoryRelease(releaseCtx, &apipb.HistoryReleaseCommand{
			Terminal: terminalRef, Token: latest.GetToken(), HistoryGeneration: latest.GetHistoryGeneration(),
		})
	}()

	printer := newHistoryContextPrinter(writer, pattern, cfg.before, cfg.after, cfg.maxCount, historySearchColorEnabled(writer, cfg.color))
	startLineID := latest.GetFirstLineId()
	if startLineID == 0 {
		startLineID = 1
	}
	var page *apipb.HistoryWindowResult
	var lastProcessed uint64
	for !printer.done {
		if err := ctx.Err(); err != nil {
			return printer.matches, err
		}
		if page == nil {
			result, searchErr := client.HistorySearch(ctx, &apipb.HistorySearchCommand{
				Terminal: terminalRef, Token: latest.GetToken(), HistoryGeneration: latest.GetHistoryGeneration(),
				Query: cfg.pattern, Mode: historySearchModeProto(cfg.resolvedMode), Direction: apipb.HistorySearchDirection_HISTORY_SEARCH_DIRECTION_FORWARD,
				Cols: historySearchCols, Limit: historySearchPageLines, ContextBefore: int32(cfg.before),
				Start: &apipb.HistoryTextPosition{LineId: startLineID},
			})
			if searchErr != nil {
				return printer.matches, searchErr
			}
			if !result.GetFound() || result.GetWrapped() || result.GetWindow() == nil {
				break
			}
			page = result.GetWindow()
		}

		lines := historySearchLines(page.GetRows())
		var lastMatchID uint64
		for _, line := range lines {
			if line.id <= lastProcessed {
				continue
			}
			ranges := pattern.FindAllStringIndex(line.text)
			if len(ranges) > 0 {
				lastMatchID = line.id
			}
			if err := printer.write(line, ranges); err != nil {
				return printer.matches, err
			}
			lastProcessed = line.id
			if printer.done {
				break
			}
		}
		if printer.done || len(lines) == 0 || lastProcessed >= latest.GetLastLineId() {
			break
		}
		lastRow := page.GetRows()[len(page.GetRows())-1]
		if cfg.after > 0 && lastMatchID > 0 && lastMatchID+uint64(cfg.after) >= lastProcessed {
			page, err = client.HistoryWindow(ctx, &apipb.HistoryWindowCommand{
				Terminal: terminalRef, Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_NEWER,
				Cols: historySearchCols, Limit: historySearchPageLines, Token: latest.GetToken(), HistoryGeneration: latest.GetHistoryGeneration(),
				BoundaryFirstLineId: latest.GetFirstLineId(), BoundaryLastLineId: latest.GetLastLineId(),
				AfterCursor: &apipb.HistoryCursor{LineId: lastRow.GetLogicalLineId(), RowInLine: lastRow.GetRowInLine(), Segment: lastRow.GetSegment()},
			})
			if err != nil {
				return printer.matches, err
			}
			continue
		}
		startLineID = lastProcessed + 1
		page = nil
	}
	return printer.matches, nil
}

type historySearchLine struct {
	id   uint64
	text string
}

func historySearchLines(rows []*apipb.HistoryRow) []historySearchLine {
	lines := make([]historySearchLine, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.GetLogicalLineId() == 0 {
			continue
		}
		text := historySearchRowText(row)
		if len(lines) > 0 && lines[len(lines)-1].id == row.GetLogicalLineId() {
			lines[len(lines)-1].text += text
			continue
		}
		lines = append(lines, historySearchLine{id: row.GetLogicalLineId(), text: text})
	}
	return lines
}

func historySearchRowText(row *apipb.HistoryRow) string {
	if row == nil || row.GetRow() == nil {
		return ""
	}
	var text strings.Builder
	for _, cell := range row.GetRow().GetCells() {
		content := cell.GetContent()
		text.WriteString(content)
		if padding := int(cell.GetWidth()) - xansi.StringWidth(content); padding > 0 {
			text.WriteString(strings.Repeat(" ", padding))
		}
	}
	return text.String()
}

type historyContextPrinter struct {
	writer       io.Writer
	pattern      *history.HistorySearchPattern
	before       int
	after        int
	maxCount     int
	color        bool
	beforeLines  []historySearchLine
	afterRemain  int
	matches      int
	lastOutputID uint64
	wroteOutput  bool
	stopping     bool
	done         bool
}

func newHistoryContextPrinter(writer io.Writer, pattern *history.HistorySearchPattern, before, after, maxCount int, color bool) *historyContextPrinter {
	return &historyContextPrinter{writer: writer, pattern: pattern, before: before, after: after, maxCount: maxCount, color: color}
}

func (printer *historyContextPrinter) write(line historySearchLine, ranges []history.HistorySearchByteRange) error {
	matched := len(ranges) > 0 && !printer.stopping
	if matched {
		for _, contextLine := range printer.beforeLines {
			if err := printer.writeLine(contextLine, false, nil); err != nil {
				return err
			}
		}
		printer.beforeLines = printer.beforeLines[:0]
		if err := printer.writeLine(line, true, ranges); err != nil {
			return err
		}
		printer.matches++
		printer.afterRemain = printer.after
		if printer.maxCount > 0 && printer.matches >= printer.maxCount {
			printer.stopping = true
			printer.done = printer.afterRemain == 0
		}
		return nil
	}
	if printer.afterRemain > 0 {
		if err := printer.writeLine(line, false, nil); err != nil {
			return err
		}
		printer.afterRemain--
		if printer.stopping && printer.afterRemain == 0 {
			printer.done = true
		}
		return nil
	}
	if printer.stopping {
		printer.done = true
		return nil
	}
	printer.beforeLines = append(printer.beforeLines, line)
	if len(printer.beforeLines) > printer.before {
		printer.beforeLines = printer.beforeLines[len(printer.beforeLines)-printer.before:]
	}
	return nil
}

func (printer *historyContextPrinter) writeLine(line historySearchLine, matched bool, ranges []history.HistorySearchByteRange) error {
	if printer.wroteOutput && line.id > printer.lastOutputID+1 {
		if _, err := fmt.Fprintln(printer.writer, "--"); err != nil {
			return err
		}
	}
	separator := "-"
	text := line.text
	if matched {
		separator = ":"
		if printer.color {
			text = highlightHistorySearchMatches(text, ranges)
		}
	}
	if _, err := fmt.Fprintf(printer.writer, "%d%s%s\n", line.id, separator, text); err != nil {
		return err
	}
	printer.lastOutputID = line.id
	printer.wroteOutput = true
	return nil
}

func highlightHistorySearchMatches(text string, ranges []history.HistorySearchByteRange) string {
	if len(ranges) == 0 {
		return text
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Start < ranges[j].Start })
	merged := ranges[:0]
	for _, current := range ranges {
		current.Start = max(0, min(len(text), current.Start))
		current.End = max(current.Start, min(len(text), current.End))
		if current.End == current.Start {
			continue
		}
		if len(merged) > 0 && current.Start <= merged[len(merged)-1].End {
			merged[len(merged)-1].End = max(merged[len(merged)-1].End, current.End)
			continue
		}
		merged = append(merged, current)
	}
	var output strings.Builder
	last := 0
	for _, current := range merged {
		output.WriteString(text[last:current.Start])
		output.WriteString("\x1b[1;31m")
		output.WriteString(text[current.Start:current.End])
		output.WriteString("\x1b[0m")
		last = current.End
	}
	output.WriteString(text[last:])
	return output.String()
}

func historySearchColorEnabled(writer io.Writer, mode string) bool {
	if mode == "always" {
		return true
	}
	if mode == "never" {
		return false
	}
	file, ok := writer.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func historySearchModeProto(mode history.HistorySearchMode) apipb.HistorySearchMode {
	switch mode {
	case history.HistorySearchModeGlob:
		return apipb.HistorySearchMode_HISTORY_SEARCH_MODE_GLOB
	case history.HistorySearchModeRegex:
		return apipb.HistorySearchMode_HISTORY_SEARCH_MODE_REGEX
	default:
		return apipb.HistorySearchMode_HISTORY_SEARCH_MODE_TEXT
	}
}
