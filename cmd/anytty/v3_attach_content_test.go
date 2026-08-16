package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/app"
	tuistate "github.com/anytty/anytty/tui/state"
)

func TestV3InteractiveRuntimeRendersInitialTerminalOutput(t *testing.T) {
	_, client, closeClient := newCoreV2ProtocolClientForCLITest(t)
	defer closeClient()
	if _, err := createCLIProtoTerminal(context.Background(), client, &apipb.TerminalCreateSpec{
		TerminalId: "initial-output", Name: "initial-output", Command: testInitialOutputCommand(),
		Size: &apipb.TerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatal(err)
	}
	host := app.NewFakeTerminalHost(32)
	runtime := newV3InteractiveRuntime("initial-output", 100, 30, wrapCLIProtocolClientForTest(t, client), host, nil)
	if err := runtime.Post(app.LiveAttachMsg{Config: app.LiveConfig{
		TerminalID: "initial-output", Cols: 100, Rows: 30, Mode: "collaborator", ResizePolicy: tuistate.TerminalResizeRoleFollower,
		SurfaceID: "initial-output-surface", ViewID: tuistate.TerminalPaneViewID(tuistate.DefaultPaneID),
	}}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_ = runtime.Drain(context.Background())
		for _, frame := range host.Frames() {
			if strings.Contains(strings.Join(frame.Lines, "\n"), "anytty-initial-output") {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("initial output was not rendered; state=%#v frames=%#v", runtime.State().Session, host.Frames())
}
