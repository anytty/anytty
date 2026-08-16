package main

import (
	"os"

	tuiconfig "github.com/anytty/anytty/tui/config"
	"github.com/anytty/anytty/tui/state"
)

func loadV3TUIConfig(path string) (state.TUIConfigStore, error) {
	return tuiconfig.Load(path, os.Getenv)
}
