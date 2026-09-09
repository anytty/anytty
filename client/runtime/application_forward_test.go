package runtime

import (
	"context"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type forwardingExecutor struct {
	inner   *ApplicationSession
	restamp bool
}

func (e forwardingExecutor) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if e.restamp {
		return e.inner.Execute(ctx, command)
	}
	return e.inner.Forward(ctx, command, false)
}
func (e forwardingExecutor) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if e.restamp {
		return e.inner.ExecuteTerminal(ctx, command)
	}
	return e.inner.Forward(ctx, command, true)
}

func TestForwardPreparedCommandPreservesOuterCorrelation(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		raw := &recordingProtoExecutor{}
		stamp := EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}
		inner, _ := NewApplicationSession(stamp, raw)
		command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}}
		_, _ = inner.Execute(context.Background(), command)
		nested, _ := NewApplicationSession(stamp, forwardingExecutor{inner: inner, restamp: true})
		if _, err := nested.execute(context.Background(), command, terminal); err != nil {
			t.Fatalf("nested Execute must preserve the prepared correlation: %v", err)
		}
		outer, _ := NewApplicationSession(stamp, forwardingExecutor{inner: inner})
		if _, err := outer.execute(context.Background(), command, terminal); err != nil {
			t.Fatal(err)
		}
		if raw.terminal != terminal {
			t.Fatal("terminal-response capability changed")
		}
		if command.Context != nil {
			t.Fatal("mutated caller")
		}
	}
}

func TestForwardRejectsForeignGenerationBeforeExecution(t *testing.T) {
	raw := &recordingProtoExecutor{}
	session, _ := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}, raw)
	command := &apipb.CommandEnvelope{Context: &apipb.RequestContext{RequestId: "prepared", Session: session.ProtoStamp()}, Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}}
	command.Context.Session.Generation++
	before := proto.Clone(command)
	if _, err := session.Forward(context.Background(), command, false); CodeOf(err) != ErrorStaleSession {
		t.Fatalf("foreign request: %v", err)
	}
	if raw.command != nil || !proto.Equal(command, before) {
		t.Fatal("foreign request executed or mutated")
	}
}
