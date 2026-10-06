// Command tui2-v3shell is the standalone Go replica of the legacy v3 TUI.
//
// It speaks the tui2 layout-program protocol on stdin/stdout (the host owns
// the terminals) and draws the old surface framework's chrome with the
// default recommended configuration (coralline-candy):
//
//	go build -o /tmp/tui2-v3shell ./clients/tui/examples/v3shell
//	TUI2_SHELL=/tmp/tui2-v3shell anytty
//	# or directly:
//	tui2 -shell /tmp/tui2-v3shell
//
// Offline modes (no host, for tests and pixel comparison):
//
//	tui2-v3shell -selftest                 # rasterize the 1.txt demo state
//	tui2-v3shell -footer-lines             # print every scene's footer row
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/anytty/anytty/clients/tui/sdk"
	"github.com/anytty/anytty/clients/tui/sdk/app"
)

const version = "0.1.0"

func main() {
	demo := flag.Bool("demo", false, "run with the 1.txt demo state preloaded")
	selftest := flag.Bool("selftest", false, "print the demo screen and exit (offline)")
	footerLines := flag.Bool("footer-lines", false, "print <SCENE>|<footer line>| rows and exit (offline)")
	cols := flag.Int("cols", 120, "offline viewport columns")
	rows := flag.Int("rows", 32, "offline viewport rows")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("tui2-v3shell", version)
		return
	}

	if *selftest || *footerLines {
		m := newModel(nil, true)
		m.viewID = "selftest"
		m.cols, m.rows = *cols, *rows
		if *footerLines {
			printFooterLines(m)
			return
		}
		lines, _ := m.rasterize(m.View())
		fmt.Println(strings.Join(lines, "\n"))
		return
	}

	client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{})
	m := newModel(client, *demo)
	claim := m.claim()
	program := &app.Program{
		Client: client,
		Model:  m,
		Keys:   &claim,
	}
	if err := program.Run(); err != nil && !errors.Is(err, app.ErrQuit) {
		fmt.Fprintln(os.Stderr, "tui2-v3shell:", err)
		os.Exit(1)
	}
}

// printFooterLines emits every scene's footer row in the reference
// <SCENE>|<line>| format (the golden suite compares the 8-scene subset).
func printFooterLines(m *model) {
	lines := map[string]string{}
	lines["LIVE"] = m.footerLine()
	m.mode = modePane
	lines["PANE"] = m.footerLine()
	m.mode = modeResize
	lines["RESIZE"] = m.footerLine()
	m.mode = modeTab
	lines["TAB"] = m.footerLine()
	m.mode = modeWorkspace
	lines["WORKSPACE"] = m.footerLine()
	m.mode = modeSystem
	lines["SYSTEM"] = m.footerLine()
	m.mode = modeFloating
	lines["FLOATING"] = m.footerLine()
	m.mode = modeLive
	m.overlay = overlayPicker
	lines["PICKER"] = m.footerLine()
	m.overlay = overlayPrompt
	lines["PROMPT"] = m.footerLine()
	m.overlay = overlayHelp
	lines["HELP"] = m.footerLine()
	m.overlay = ""
	names := make([]string, 0, len(lines))
	for name := range lines {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s|%s|\n", name, lines[name])
	}
}
