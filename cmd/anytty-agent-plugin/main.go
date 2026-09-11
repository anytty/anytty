package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/anytty/anytty/plugin/sdk"
	"github.com/anytty/anytty/plugins/agents"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	client := sdk.NewStdioClient(os.Stdin, os.Stdout)
	defer client.Close()
	var id [16]byte
	_, err := rand.Read(id[:])
	if err == nil {
		instance := hex.EncodeToString(id[:])
		switch os.Getenv("ANYTTY_PLUGIN_MODE") {
		case "daemon":
			err = agents.RunDaemon(ctx, client, instance)
		case "tui":
			var endpoints []string
			if raw := os.Getenv("ANYTTY_PLUGIN_ENDPOINTS"); raw != "" {
				err = json.Unmarshal([]byte(raw), &endpoints)
			}
			if len(endpoints) == 0 {
				endpoints = []string{os.Getenv("ANYTTY_PLUGIN_ENDPOINT")}
			}
			if err == nil {
				err = agents.RunTUI(ctx, client, os.Getenv("ANYTTY_TUI_INSTANCE_ID"), instance, endpoints)
			}
		default:
			err = errors.New("ANYTTY_PLUGIN_MODE must be daemon or tui")
		}
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
