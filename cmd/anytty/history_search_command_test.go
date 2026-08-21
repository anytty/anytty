package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
)

func TestHistorySearchCommandRejectsInvalidRegexBeforeConnecting(t *testing.T) {
	var socket, logFile string
	command := newHistorySearchCommand(&socket, &logFile)
	command.SetArgs([]string{"terminal-1", "[", "--mode", "regex"})

	err := command.Execute()
	if cliExitCode(err) != 2 || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("invalid regex error = %v, exit=%d", err, cliExitCode(err))
	}
}

func TestHistoryContextPrinterHighlightsChineseMatchAndMergesContext(t *testing.T) {
	pattern, err := history.CompileHistorySearchPattern(history.HistorySearchModeText, "错误")
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	printer := newHistoryContextPrinter(&output, pattern, 1, 1, 0, true)
	for id, text := range []string{"before", "中文错误发生", "after", "unused"} {
		ranges := pattern.FindAllStringIndex(text)
		if err := printer.write(historySearchLine{id: uint64(id + 1), text: text}, ranges); err != nil {
			t.Fatal(err)
		}
	}

	want := "1-before\n2:中文\x1b[1;31m错误\x1b[0m发生\n3-after\n"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestRunHistorySearchUsesServerModeAndContext(t *testing.T) {
	client := &fakeHistorySearchProtocol{
		latest: &apipb.HistoryWindowResult{Token: "frozen", HistoryGeneration: 7, FirstLineId: 5, LastLineId: 5},
		searchResults: []*apipb.HistorySearchResult{{
			Found: true,
			Match: &apipb.HistoryRange{StartLineId: 2, EndLineId: 2, EndCol: 2},
			Window: &apipb.HistoryWindowResult{Rows: []*apipb.HistoryRow{
				historySearchTestRow(1, "before"),
				historySearchTestRow(2, "错误"),
				historySearchTestRow(3, "after"),
				historySearchTestRow(4, "other"),
				historySearchTestRow(5, "end"),
			}},
		}},
	}
	cfg := historySearchCommandConfig{
		terminalID: "terminal-1", pattern: "错?", resolvedMode: history.HistorySearchModeGlob,
		before: 1, after: 1, color: "never",
	}
	var output bytes.Buffer

	matches, err := runHistorySearch(context.Background(), client, &apipb.TerminalRef{EndpointId: "local", TerminalId: "terminal-1"}, cfg, &output)
	if err != nil {
		t.Fatal(err)
	}
	if matches != 1 || output.String() != "1-before\n2:错误\n3-after\n" {
		t.Fatalf("matches=%d output=%q", matches, output.String())
	}
	if len(client.searchCommands) != 1 {
		t.Fatalf("search calls = %d", len(client.searchCommands))
	}
	request := client.searchCommands[0]
	if request.GetMode() != apipb.HistorySearchMode_HISTORY_SEARCH_MODE_GLOB || request.GetContextBefore() != 1 {
		t.Fatalf("search mode=%v context=%d", request.GetMode(), request.GetContextBefore())
	}
	if request.GetStart().GetLineId() != 1 {
		t.Fatalf("chronological search started at logical line %d, want 1", request.GetStart().GetLineId())
	}
	if client.releases != 1 {
		t.Fatalf("releases = %d", client.releases)
	}
}

func historySearchTestRow(lineID uint64, text string) *apipb.HistoryRow {
	return &apipb.HistoryRow{
		LogicalLineId: lineID,
		Row:           &apipb.ScreenRow{Cells: []*apipb.ScreenCell{{Content: text, Width: int32(len([]rune(text)))}}},
	}
}

type fakeHistorySearchProtocol struct {
	latest         *apipb.HistoryWindowResult
	searchResults  []*apipb.HistorySearchResult
	searchCommands []*apipb.HistorySearchCommand
	releases       int
}

func (client *fakeHistorySearchProtocol) HistoryWindow(_ context.Context, command *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	if command.GetMode() == apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST {
		return client.latest, nil
	}
	return &apipb.HistoryWindowResult{}, nil
}

func (client *fakeHistorySearchProtocol) HistorySearch(_ context.Context, command *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	client.searchCommands = append(client.searchCommands, command)
	if len(client.searchResults) == 0 {
		return &apipb.HistorySearchResult{}, nil
	}
	result := client.searchResults[0]
	client.searchResults = client.searchResults[1:]
	return result, nil
}

func (client *fakeHistorySearchProtocol) HistoryRelease(context.Context, *apipb.HistoryReleaseCommand) error {
	client.releases++
	return nil
}
