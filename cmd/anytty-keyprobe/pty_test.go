//go:build darwin || linux

package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/anytty/anytty/tui/config"
	"github.com/anytty/anytty/tui/terminalhost"
	"github.com/creack/pty"
)

func TestRealHostRawPTYReceivesCtrlBackslash(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	host := terminalhost.New(terminalhost.WithInput(slave, slave.Fd()), terminalhost.WithOutput(io.Discard), terminalhost.WithThemeProbe(false))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := host.Enter(ctx); err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	// The actual termios raw mode must allow control bytes through the kernel.
	for _, seq := range []string{"\x1c", "\x1b[92;5u", "\x1b[92;5:2u", "\x03", "\x13", "\x11"} {
		if _, err := io.WriteString(master, seq); err != nil {
			t.Fatal(err)
		}
		select {
		case event := <-host.InputEvents():
			got := describe(event, config.Default().Shortcuts)
			if got.ForcedOutputHex == "" {
				t.Fatalf("%q lost: %+v", seq, got)
			}
			if seq == "\x1c" || seq == "\x1b[92;5u" || seq == "\x1b[92;5:2u" {
				if got.OutputHex != "1c" {
					t.Fatalf("ctrl-backslash: %+v", got)
				}
			}
		case <-ctx.Done():
			t.Fatalf("host did not receive %q", seq)
		}
	}
}
