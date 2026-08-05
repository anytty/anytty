package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/internal/protocol"
)

func TestProductCommandTreeExposesTerminalAndRejectsV3(t *testing.T) {
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	help := output.String()
	for _, expected := range []string{"terminal", "daemon", "pair"} {
		if !strings.Contains(help, expected) {
			t.Fatalf("root help missing %q:\n%s", expected, help)
		}
	}
	for _, forbidden := range []string{"\n  v3 ", "smoke", "visual-snapshot", "tmux-smoke"} {
		if strings.Contains(help, forbidden) {
			t.Fatalf("product help exposes %q:\n%s", forbidden, help)
		}
	}

	command = newRootCmd()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"v3", "smoke"})
	if err := command.Execute(); err == nil {
		t.Fatal("product command tree accepted removed v3 namespace")
	}
}

func TestTerminalHelpListsCompleteCLI002Lifecycle(t *testing.T) {
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"terminal", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"create", "list", "show", "attach", "restart", "kill", "remove", "rename", "tag"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("terminal help missing %q:\n%s", expected, output.String())
		}
	}
}

func TestTerminalHelpListsCLI005AutomationCommands(t *testing.T) {
	command := newRootCmd()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"terminal", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"send", "capture", "resize", "wait", "events"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("terminal help missing CLI005 command %q:\n%s", expected, output.String())
		}
	}
}

func TestCLIExitCodeUsesTypedProtocolErrors(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{err: usageCLIError("bad argument"), want: 2},
		{err: classifyCLIError(&protocol.RequestError{Code: 404, Message: "missing"}), want: 3},
		{err: classifyCLIError(&protocol.RequestError{Code: 400, Message: "conflict"}), want: 4},
		{err: classifyCLIError(&protocol.RequestError{Code: 403, Message: "denied"}), want: 5},
		{err: classifyCLIError(&protocol.RequestError{Code: 503, Message: "offline"}), want: 6},
		{err: classifyCLIError(errors.New("dial failed")), want: 6},
	}
	for _, test := range cases {
		if got := cliExitCode(test.err); got != test.want {
			t.Fatalf("cliExitCode(%v) = %d, want %d", test.err, got, test.want)
		}
	}
}

func TestResolveTerminalRefUsesOwningEndpoint(t *testing.T) {
	registry := endpointdomain.Registry{
		Version: endpointdomain.RegistryVersion,
		Default: "west",
		Endpoints: map[endpointdomain.EndpointID]endpointdomain.Endpoint{
			"local": testLocalEndpoint("local", "Local", "auto", endpointdomain.ConnectAuto, true),
			"west":  testSSHEndpoint("west", "West", "west.example", "", endpointdomain.ConnectOnDemand, true),
			"off":   testLocalEndpoint("off", "Off", "auto", endpointdomain.ConnectOnDemand, false),
		},
	}
	if ref, err := resolveTerminalRef("demo", "", registry); err != nil || ref.String() != "west:demo" {
		t.Fatalf("default target = (%q, %v)", ref.String(), err)
	}
	if ref, err := resolveTerminalRef("local:demo", "", registry); err != nil || ref.String() != "local:demo" {
		t.Fatalf("explicit target = (%q, %v)", ref.String(), err)
	}
	if _, err := resolveTerminalRef("local:demo", "west", registry); cliExitCode(err) != 2 {
		t.Fatalf("conflicting endpoint error = %v, exit=%d", err, cliExitCode(err))
	}
	if _, err := resolveTerminalRef("off:demo", "", registry); cliExitCode(err) != 4 {
		t.Fatalf("disabled endpoint error = %v, exit=%d", err, cliExitCode(err))
	}
}
