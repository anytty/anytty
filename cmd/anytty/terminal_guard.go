package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func nestedTUIBlocked() bool {
	return os.Getenv("ANYTTY") == "1" && os.Getenv("ANYTTY_ALLOW_NESTED") != "1"
}

func rejectNestedTUI() error {
	if !nestedTUIBlocked() {
		return nil
	}
	return fmt.Errorf("refusing to start anytty TUI inside a anytty remote terminal; use a normal shell, or set ANYTTY_ALLOW_NESTED=1 if you really want nesting")
}
