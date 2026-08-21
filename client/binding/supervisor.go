package binding

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

// EndpointSupervisorHost is the process-owned native recovery control plane.
// It is intentionally separate from renderer handles and application events.
type EndpointSupervisorHost interface {
	ReplaceEndpointDemand(clientruntime.EndpointDemandSnapshot) error
	SignalEndpointHost(clientruntime.EndpointHostSignal) error
	WaitEndpointDemandReady(context.Context) error
	EndpointSupervisorSnapshot() []clientruntime.EndpointSupervisorProjection
}

func ReplaceEndpointSupervisorDemand(host EndpointSupervisorHost, payload []byte) error {
	if host == nil {
		return ErrInvalidHandle
	}
	if err := validatePayload(payload); err != nil {
		return err
	}
	wire := &bindingpb.EndpointSupervisorDemandSnapshot{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, wire); err != nil {
		return fmt.Errorf("decode endpoint supervisor demand: %w", err)
	}
	snapshot := clientruntime.EndpointDemandSnapshot{
		AttachmentID:   strings.TrimSpace(wire.GetAttachmentId()),
		DemandRevision: wire.GetDemandRevision(),
		Endpoints:      make([]clientruntime.EndpointDemand, 0, len(wire.GetEndpoints())),
	}
	for _, value := range wire.GetEndpoints() {
		mode := clientruntime.EndpointSupervisorMode("")
		switch value.GetMode() {
		case bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_SHADOW:
			mode = clientruntime.EndpointSupervisorShadow
		case bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_TAKEOVER:
			mode = clientruntime.EndpointSupervisorTakeover
		default:
			return fmt.Errorf("endpoint %q supervisor mode is required", value.GetEndpointId())
		}
		snapshot.Endpoints = append(snapshot.Endpoints, clientruntime.EndpointDemand{
			EndpointID: endpoint.EndpointID(strings.TrimSpace(value.GetEndpointId())),
			Mode:       mode,
		})
	}
	return host.ReplaceEndpointDemand(snapshot)
}

func SignalEndpointSupervisor(host EndpointSupervisorHost, payload []byte) error {
	if host == nil {
		return ErrInvalidHandle
	}
	if err := validatePayload(payload); err != nil {
		return err
	}
	wire := &bindingpb.EndpointSupervisorHostSignal{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, wire); err != nil {
		return fmt.Errorf("decode endpoint supervisor host signal: %w", err)
	}
	return host.SignalEndpointHost(clientruntime.EndpointHostSignal{
		Revision: wire.GetRevision(), Connected: wire.GetConnected(),
		Reason: strings.TrimSpace(wire.GetReason()), Foreground: wire.GetForeground(),
	})
}

func EncodeEndpointSupervisorSnapshot(host EndpointSupervisorHost) ([]byte, error) {
	if host == nil {
		return nil, ErrInvalidHandle
	}
	values := host.EndpointSupervisorSnapshot()
	sort.Slice(values, func(i, j int) bool { return values[i].EndpointID < values[j].EndpointID })
	wire := &bindingpb.EndpointSupervisorSnapshot{Endpoints: make([]*bindingpb.EndpointSupervisorProjection, 0, len(values))}
	for _, value := range values {
		mode := bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_UNSPECIFIED
		switch value.Mode {
		case clientruntime.EndpointSupervisorShadow:
			mode = bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_SHADOW
		case clientruntime.EndpointSupervisorTakeover:
			mode = bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_TAKEOVER
		}
		projection := &bindingpb.EndpointSupervisorProjection{
			EndpointId: string(value.EndpointID), Mode: mode, Phase: string(value.Phase),
			ControlRevision: value.ControlRevision, AttemptId: value.AttemptID,
			ErrorCode: string(value.ErrorCode), Message: value.Message,
			ProbeCount: value.ProbeCount, DialCount: value.DialCount, BackoffCount: value.BackoffCount,
		}
		if value.SessionStamp.Validate() == nil {
			projection.Session = &apipb.EndpointSessionStamp{
				EndpointId: string(value.SessionStamp.EndpointID), RouteId: string(value.SessionStamp.RouteID), Generation: uint64(value.SessionStamp.Generation),
			}
		}
		wire.Endpoints = append(wire.Endpoints, projection)
	}
	return (proto.MarshalOptions{Deterministic: true}).Marshal(wire)
}
