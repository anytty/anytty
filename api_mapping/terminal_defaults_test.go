package apimapping

import (
	"reflect"
	"testing"

	corev2 "github.com/anytty/anytty/core"
)

func TestTerminalDefaultsToProtoIncludesDaemonPlatformWithoutAliasing(t *testing.T) {
	defaults := corev2.TerminalDefaults{
		DefaultCommand: []string{"/bin/zsh"},
		DefaultCWD:     "/Users/test",
		Platform:       "darwin",
	}
	projection := TerminalDefaultsToProto(defaults).GetDefaults()
	if projection.GetPlatform() != "darwin" || projection.GetDefaultCwd() != "/Users/test" {
		t.Fatalf("projection = %#v", projection)
	}
	projection.DefaultCommand[0] = "mutated"
	if !reflect.DeepEqual(defaults.DefaultCommand, []string{"/bin/zsh"}) {
		t.Fatalf("projection aliases core defaults: %#v", defaults)
	}
}
