package main

import (
	"context"
	"fmt"

	"github.com/anytty/anytty/clients/tui/runtime"
	"github.com/anytty/anytty/proto/access/apipb"

	gproto "google.golang.org/protobuf/proto"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

// access.call is a transparent forwarder: the layout program serializes any
// access CommandEnvelope with generated protobuf types and the host only
// selects the endpoint's ready connection and relays the opaque result. The
// host applies no family policy and no confirmation; single-writer discipline
// (attach/resize/input) is the program's responsibility, and the typed
// terminal methods remain the recommended path.
// accessCall validates the command synchronously (fast, no I/O) and then runs
// the endpoint call on a background goroutine. The session loop only registers
// the request as in flight; a slow or offline endpoint can therefore never
// block the frame loop while the call times out.
func (g *gateHandler) accessCall(req runtime.Request) (runtime.Outcome, bool) {
	command := &apipb.CommandEnvelope{}
	if err := gproto.Unmarshal(req.Params.GetAccessCommand(), command); err != nil {
		return runtime.Outcome{Error: fmt.Sprintf("access.call: decode command: %v", err)}, false
	}
	session := g.host.currentSession()
	if session == nil {
		return runtime.Outcome{Error: "access.call: no active session"}, false
	}
	if g.host.endpoints == nil {
		return runtime.Outcome{Error: "access.call: endpoint manager unavailable"}, false
	}
	go func() {
		g.completeAsync(session, req.RequestID, g.host.executeAccessCall(req.Params.GetEndpoint(), command))
	}()
	return runtime.Outcome{}, true
}

// completeAsync delivers the outcome of a background request as the RESPONSE
// for requestID. A session that already moved on (program restart) rejects the
// completion; the stale outcome is dropped with it.
func (g *gateHandler) completeAsync(session *runtime.Session, requestID uint64, outcome runtime.Outcome) {
	var data *pb.MethodData
	errMsg := ""
	if outcome.OK {
		data = outcome.Data
	} else {
		errMsg = outcome.Error
		if errMsg == "" {
			errMsg = "rejected"
		}
	}
	_ = session.Complete(requestID, data, errMsg)
}

// completeAsyncEpoch is completeAsync fenced by the request's epoch, so a
// worker that outlives a program restart cannot answer a reused request id.
func (g *gateHandler) completeAsyncEpoch(session *runtime.Session, epoch, requestID uint64, outcome runtime.Outcome) {
	var data *pb.MethodData
	errMsg := ""
	if outcome.OK {
		data = outcome.Data
	} else {
		errMsg = outcome.Error
		if errMsg == "" {
			errMsg = "rejected"
		}
	}
	_ = session.CompleteForEpoch(epoch, requestID, data, errMsg)
}

// accessCallContext bounds one background access call with the host call
// timeout, matching the endpoint manager's own budget.
func (h *Host) accessCallContext() (context.Context, context.CancelFunc) {
	if t := h.opts.CallTimeout; t > 0 {
		return context.WithTimeout(context.Background(), t)
	}
	return context.WithCancel(context.Background())
}

// executeAccessCall runs one access.call request and encodes the opaque access
// ResultEnvelope into MethodData.access_result.
func (h *Host) executeAccessCall(endpointName string, command *apipb.CommandEnvelope) runtime.Outcome {
	ctx, cancel := h.accessCallContext()
	defer cancel()
	result, err := h.endpoints.Execute(ctx, endpointName, command)
	if err != nil {
		return runtime.Outcome{Error: "access.call: " + err.Error()}
	}
	payload, err := gproto.Marshal(result)
	if err != nil {
		return runtime.Outcome{Error: fmt.Sprintf("access.call: encode result: %v", err)}
	}
	return runtime.Outcome{OK: true, Data: &pb.MethodData{AccessResult: payload}}
}
