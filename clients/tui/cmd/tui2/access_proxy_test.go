package main

import (
	"strings"
	"testing"

	"github.com/anytty/anytty/clients/tui/runtime"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// access.call is a transparent forwarder: the only host-side failure before
// reaching the endpoint connection is an undecodable opaque command.
func TestAccessCallRejectsUndecodableCommand(t *testing.T) {
	gate := &gateHandler{host: &Host{}}
	outcome, pending := gate.accessCall(runtime.Request{Params: &pb.MethodParams{
		Endpoint:      "local",
		AccessCommand: []byte{0xff, 0xff, 0xff},
	}})
	if pending {
		t.Fatal("undecodable command must not ask for confirmation")
	}
	if outcome.OK || !strings.Contains(outcome.Error, "decode command") {
		t.Fatalf("outcome = %+v", outcome)
	}
}
