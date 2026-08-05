package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
)

func TestTerminalAutomationLocalDataPlane(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath, client, closeServer := startCLIEndpointServer(t)
	defer closeServer()
	if err := endpointdomain.Save("", endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion, Default: endpointdomain.DefaultEndpointID,
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			endpointdomain.DefaultEndpointID: testLocalEndpoint(endpointdomain.DefaultEndpointID, "Local", socketPath, endpointdomain.ConnectAuto, true),
		},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "automation", Name: "automation", Command: testAutomationCommand(),
		Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatal(err)
	}
	timeoutCommand := newRootCmd()
	timeoutCommand.SetOut(io.Discard)
	timeoutCommand.SetErr(io.Discard)
	timeoutCommand.SetArgs([]string{"terminal", "wait", "local:automation", "--state", "exited", "--timeout", "20ms"})
	if err := timeoutCommand.Execute(); cliExitCode(err) != 7 {
		t.Fatalf("wait timeout = %v, exit=%d", err, cliExitCode(err))
	}

	sent := executeTerminalCLI(t, nil, "terminal", "send", "local:automation", "hello", "--enter", "--json")
	if !strings.Contains(sent, `"kind":"terminal_input_sent"`) || !strings.Contains(sent, `"bytes":6`) {
		t.Fatalf("unexpected send output: %s", sent)
	}
	waited := executeTerminalCLI(t, nil, "terminal", "wait", "local:automation", "--state", "exited", "--timeout", "5s", "--json")
	if !strings.Contains(waited, `"state":"exited"`) {
		t.Fatalf("unexpected wait output: %s", waited)
	}
	captured := executeTerminalCLI(t, nil, "terminal", "capture", "local:automation", "--lines", "100", "--cols", "80")
	if !strings.Contains(captured, "READY") || !strings.Contains(captured, "GOT:hello") {
		t.Fatalf("capture did not read authoritative terminal output: %q", captured)
	}

	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "resize-me", Name: "resize-me", Command: testShellSleepCommand(), Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatal(err)
	}
	resized := executeTerminalCLI(t, nil, "terminal", "resize", "local:resize-me", "100", "35", "--json")
	if !strings.Contains(resized, `"resized":true`) || !strings.Contains(resized, `"cols":100`) || !strings.Contains(resized, `"rows":35`) {
		t.Fatalf("unexpected resize output: %s", resized)
	}
	formatted := executeTerminalCLI(t, nil, "terminal", "list", "--format", `{{.target}}|{{.state}}|{{.cols}}x{{.rows}}`)
	if !strings.Contains(formatted, "local:resize-me|running|100x35") {
		t.Fatalf("unexpected formatted list: %s", formatted)
	}
	live := executeTerminalCLI(t, nil, "terminal", "capture", "local:resize-me", "--live", "--json")
	if !strings.Contains(live, `"source":"live"`) || !strings.Contains(live, `"target":"local:resize-me"`) {
		t.Fatalf("unexpected live capture: %s", live)
	}
}

func TestTerminalStreamRoutesThroughOwningEndpoint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath, client, closeServer := startCLIEndpointServer(t)
	defer closeServer()
	const endpointID = endpointdomain.EndpointID("west")
	if err := endpointdomain.Save("", endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion, Default: endpointID,
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			endpointID: testLocalEndpoint(endpointID, "West", socketPath, endpointdomain.ConnectAuto, true),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "raw-endpoint", Name: "raw-endpoint", Command: testAutomationCommand(),
		Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatal(err)
	}

	output := executeTerminalCLI(t, strings.NewReader("endpoint-stream\n"),
		"--timeout", "5s", "terminal", "stream", "west:raw-endpoint", "--stdin")
	if !strings.Contains(output, "GOT:endpoint-stream") {
		t.Fatalf("endpoint raw PTY stream did not contain process output: %q", output)
	}
}

func TestTerminalEventsWritesStableNDJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	socketPath, client, closeServer := startCLIEndpointServer(t)
	defer closeServer()
	if err := endpointdomain.Save("", endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion, Default: endpointdomain.DefaultEndpointID,
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			endpointdomain.DefaultEndpointID: testLocalEndpoint(endpointdomain.DefaultEndpointID, "Local", socketPath, endpointdomain.ConnectAuto, true),
		},
	}); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct {
		output string
		err    error
	}, 1)
	go func() {
		command := newRootCmd()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(io.Discard)
		command.SetArgs([]string{"terminal", "events", "--type", "created", "--output", "ndjson", "--count", "1", "--timeout", "5s"})
		err := command.Execute()
		done <- struct {
			output string
			err    error
		}{output.String(), err}
	}()
	var result struct {
		output string
		err    error
	}
	for index := 0; index < 50; index++ {
		terminalID := fmt.Sprintf("event-created-%d", index)
		if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
			TerminalId: terminalID, Command: testExitCommand(), Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
		}); err != nil {
			t.Fatal(err)
		}
		select {
		case result = <-done:
			goto received
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatal("events command did not observe a created event")

received:
	if result.err != nil {
		t.Fatal(result.err)
	}
	var event terminalEventEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.output)), &event); err != nil {
		t.Fatalf("invalid NDJSON event %q: %v", result.output, err)
	}
	if event.Kind != "terminal_event" || event.Type != "lifecycle" || !strings.HasPrefix(event.Target, "local:event-created-") || event.SchemaVersion != 1 {
		t.Fatalf("unexpected event %#v", event)
	}
}

func TestTerminalAutomationInputAndFormatValidation(t *testing.T) {
	data, err := terminalSendPayload(strings.NewReader("ignored"), nil, nil, []string{"Ctrl-C", "Enter", "Up"}, nil, false, false)
	if err != nil || !bytes.Equal(data, []byte{3, '\r', 0x1b, '[', 'A'}) {
		t.Fatalf("encoded keys = %v, %v", data, err)
	}
	if _, err := terminalSendPayload(strings.NewReader("stdin"), []string{"text"}, nil, nil, nil, true, false); cliExitCode(err) != 2 {
		t.Fatalf("mixed source error = %v", err)
	}
	var output bytes.Buffer
	if err := writeTerminalFormat(&output, `{{.unknown}}`, []terminalView{{Target: "local:one"}}); cliExitCode(err) != 2 {
		t.Fatalf("unknown format field error = %v", err)
	}
	if _, _, err := parseTerminalSize("80x", "24"); cliExitCode(err) != 2 {
		t.Fatalf("invalid size error = %v", err)
	}
}

func executeTerminalCLI(t *testing.T, input io.Reader, args ...string) string {
	t.Helper()
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	if input == nil {
		input = strings.NewReader("")
	}
	command.SetIn(input)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("anytty %s: %v", strings.Join(args, " "), err)
	}
	return output.String()
}
