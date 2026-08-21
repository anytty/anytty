package binding

import (
	"context"
	"testing"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

func TestEndpointSupervisorControlPlaneUsesDeterministicProtoSnapshots(t *testing.T) {
	host := &supervisorBindingHost{}
	payload, err := proto.Marshal(&bindingpb.EndpointSupervisorDemandSnapshot{
		AttachmentId:   "renderer-a",
		DemandRevision: 7,
		Endpoints: []*bindingpb.EndpointSupervisorDemand{
			{EndpointId: "studio", Mode: bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_TAKEOVER},
			{EndpointId: "laptop", Mode: bindingpb.EndpointSupervisorMode_ENDPOINT_SUPERVISOR_MODE_SHADOW},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplaceEndpointSupervisorDemand(host, payload); err != nil {
		t.Fatal(err)
	}
	if host.demand.AttachmentID != "renderer-a" || host.demand.DemandRevision != 7 || len(host.demand.Endpoints) != 2 {
		t.Fatalf("demand = %#v", host.demand)
	}

	signalPayload, err := proto.Marshal(&bindingpb.EndpointSupervisorHostSignal{
		Revision: 11, Connected: true, Reason: "network_replaced", Foreground: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := SignalEndpointSupervisor(host, signalPayload); err != nil {
		t.Fatal(err)
	}
	if host.signal.Revision != 11 || !host.signal.Connected || !host.signal.Foreground {
		t.Fatalf("signal = %#v", host.signal)
	}

	host.projections = []clientruntime.EndpointSupervisorProjection{
		{EndpointID: "studio", Mode: clientruntime.EndpointSupervisorTakeover, Phase: clientruntime.EndpointSupervisorReady, ControlRevision: 4, AttemptID: 8, ProbeCount: 2},
		{EndpointID: "laptop", Mode: clientruntime.EndpointSupervisorShadow, Phase: clientruntime.EndpointSupervisorShadowDecision, ControlRevision: 3},
	}
	encoded, err := EncodeEndpointSupervisorSnapshot(host)
	if err != nil {
		t.Fatal(err)
	}
	decoded := &bindingpb.EndpointSupervisorSnapshot{}
	if err := proto.Unmarshal(encoded, decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.GetEndpoints()) != 2 || decoded.GetEndpoints()[0].GetEndpointId() != "laptop" || decoded.GetEndpoints()[1].GetProbeCount() != 2 {
		t.Fatalf("snapshot = %#v", decoded)
	}
}

type supervisorBindingHost struct {
	demand      clientruntime.EndpointDemandSnapshot
	signal      clientruntime.EndpointHostSignal
	projections []clientruntime.EndpointSupervisorProjection
}

func (host *supervisorBindingHost) ReplaceEndpointDemand(snapshot clientruntime.EndpointDemandSnapshot) error {
	host.demand = snapshot
	return nil
}

func (host *supervisorBindingHost) SignalEndpointHost(signal clientruntime.EndpointHostSignal) error {
	host.signal = signal
	return nil
}

func (host *supervisorBindingHost) WaitEndpointDemandReady(context.Context) error { return nil }

func (host *supervisorBindingHost) EndpointSupervisorSnapshot() []clientruntime.EndpointSupervisorProjection {
	return host.projections
}
