package main

import (
	"testing"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
)

func TestTerminalInfoViewIncludesRuntimeSummary(t *testing.T) {
	lastOutput := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	view := terminalInfoView(endpointdomain.EndpointID("studio"), &apipb.TerminalInfo{
		Ref:                  &apipb.TerminalRef{EndpointId: "studio", TerminalId: "agent"},
		ForegroundProcess:    "codex",
		LastOutputAtUnixNano: lastOutput.UnixNano(),
	})
	if view.ForegroundProcess != "codex" || view.LastOutputAt != lastOutput.Format(time.RFC3339Nano) {
		t.Fatalf("runtime view = %#v", view)
	}
}
