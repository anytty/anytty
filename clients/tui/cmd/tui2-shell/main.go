// Command tui2-shell is the v2 reference layout program: a workspace / tab /
// slot terminal multiplexer speaking the TUI v2 binary protocol on
// stdin/stdout (PROTOCOL §0). It owns every business concept and every
// appearance decision; the host only sees boxes, sources, style tokens and
// method calls.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anytty/anytty/clients/tui/config"
	"github.com/anytty/anytty/clients/tui/sdk"
)

// version is the layout program version reported by -version.
const version = "0.2.0"

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2-shell:", err)
		os.Exit(2)
	}
	if opts.version {
		fmt.Println("tui2-shell", version)
		return
	}
	if opts.printDefault {
		fmt.Print(config.Example())
		return
	}

	cfg, warning := loadConfig(opts.configPath)
	m := newModel()
	m.attachTarget = strings.TrimSpace(opts.attach)
	if err := m.applyConfig(cfg); err != nil {
		m = newModel()
		warning = "config error: " + err.Error()
	}
	if warning != "" {
		m.status = warning
	}

	client := sdk.New(os.Stdin, os.Stdout, sdk.Handlers{})
	p := newProgram(m, client)
	client.SetHandlers(p.handlers())
	if m.clockEnabled {
		p.startClock(time.Second)
	}
	if err := client.Loop(); err != nil {
		fmt.Fprintln(os.Stderr, "tui2-shell:", err)
		os.Exit(1)
	}
}
