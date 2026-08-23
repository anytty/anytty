package main

import (
	"testing"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

func TestFormatClientAccessScope(t *testing.T) {
	for name, test := range map[string]struct {
		scope *remoteauthpb.ClientAccessScope
		want  string
	}{
		"daemon":         {scope: &remoteauthpb.ClientAccessScope{AllowDaemon: true, FileReadContent: true, FileMutate: true}, want: "daemon,file-read,file-mutate"},
		"terminal":       {scope: &remoteauthpb.ClientAccessScope{TerminalId: "term-1"}, want: "terminal:term-1"},
		"machine events": {scope: &remoteauthpb.ClientAccessScope{MachineEventsOnly: true}, want: "machine-events"},
		"missing":        {want: "unknown"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := formatClientAccessScope(test.scope); got != test.want {
				t.Fatalf("scope = %q, want %q", got, test.want)
			}
		})
	}
}
