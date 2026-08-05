package app

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/tui/state"
)

func TestNextTerminalPoolIDDoesNotReuseLegacyCountID(t *testing.T) {
	id := nextTerminalPoolID(state.Root{})
	if id == "term-pool-1" {
		t.Fatalf("new terminal id must not reuse legacy count id")
	}
	if !strings.HasPrefix(id, "term-pool-") {
		t.Fatalf("unexpected terminal id %q", id)
	}
}

func TestNextTerminalPoolIDSkipsFreshCollision(t *testing.T) {
	first := nextTerminalPoolID(state.Root{})
	root := state.Root{
		TerminalPool: state.TerminalPoolStore{
			Items: []state.TerminalPoolItem{{TerminalID: first}},
		},
	}
	second := nextTerminalPoolID(root)
	if second == first {
		t.Fatalf("next terminal id must avoid currently used id %q", first)
	}
}
