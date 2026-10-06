// Command herdr is a self-drawn terminal workspace manager: workspaces ->
// tabs -> panes over the host terminal pool, with a sidebar (Agents/Spaces),
// a tab strip, program-drawn pane chrome, mode bars, transient toasts and
// overlays. It is the
// P1 self-drawn demo from docs/design/TUI_SURFACE_REFACTOR.zh-CN.md §12 and
// uses only clients/tui/sdk, clients/tui/sdk/app and clients/tui/sdk/widgets:
// the host owns the terminals, herdr owns every box and never touches the
// wire bytes.
//
// The product/interaction contract is SPEC.zh-CN.md; README.zh-CN.md is the
// user-facing summary. The program claims every key except while a live
// terminal pane is focused, where only ctrl+b and the direct ctrl+alt chords
// are claimed so ordinary typing reaches the PTY (SPEC §5).
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
)

func main() {
	client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{})
	memo := &app.Memo{}
	program := &app.Program{
		Client: client,
		Model:  newModel(client, memo),
		Memo:   memo,
		// The first view claims every key; the model returns SetKeys with
		// each mode/focus change so the claim and the view stay atomic.
		Keys: &sdk.Keys{All: true},
	}
	if err := program.Run(); err != nil && !errors.Is(err, app.ErrQuit) {
		fmt.Fprintln(os.Stderr, "herdr:", err)
		os.Exit(1)
	}
}
