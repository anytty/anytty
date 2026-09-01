package cloud

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	apilayer "github.com/anytty/anytty/api_layer"
	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	clientruntime "github.com/anytty/anytty/client/runtime"
	cloudclient "github.com/anytty/anytty/cloud/client"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport"
	"github.com/anytty/anytty/shared/transport/memory"
	"google.golang.org/protobuf/proto"
)

type cloudApplicationProbeFunc func(context.Context, *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error)

func (probe cloudApplicationProbeFunc) TerminalDefaults(ctx context.Context, command *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
	return probe(ctx, command)
}

type confirmedCloudPeerReleaseFunc func(context.Context) error

func (release confirmedCloudPeerReleaseFunc) ReleaseAndWait(ctx context.Context) error {
	return release(ctx)
}

func TestProbeCloudApplicationRequiresTerminalDefaults(t *testing.T) {
	var calls int
	probe := cloudApplicationProbeFunc(func(_ context.Context, command *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
		calls++
		if command == nil {
			t.Fatal("probe did not use the side-effect-free terminal defaults command")
		}
		return &apipb.TerminalDefaultsResult{}, nil
	})
	if err := probeCloudApplication(context.Background(), probe); err != nil {
		t.Fatalf("probeCloudApplication: %v", err)
	}
	if calls != cloudApplicationProbeRounds {
		t.Fatalf("TerminalDefaults calls = %d, want %d", calls, cloudApplicationProbeRounds)
	}

	emptyResult := cloudApplicationProbeFunc(func(context.Context, *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
		return nil, nil
	})
	if err := probeCloudApplication(context.Background(), emptyResult); err == nil {
		t.Fatal("application probe accepted an empty TerminalDefaults result")
	}
	if err := probeCloudApplication(context.Background(), nil); err == nil {
		t.Fatal("application probe accepted a nil session")
	}
}

func TestProbeCloudApplicationRejectsSecondRequestFailure(t *testing.T) {
	want := errors.New("second request stalled")
	var calls int
	probe := cloudApplicationProbeFunc(func(context.Context, *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
		calls++
		if calls == cloudApplicationProbeRounds {
			return nil, want
		}
		return &apipb.TerminalDefaultsResult{}, nil
	})
	if err := probeCloudApplication(context.Background(), probe); !errors.Is(err, want) {
		t.Fatalf("application probe error = %v, want %v", err, want)
	}
	if calls != cloudApplicationProbeRounds {
		t.Fatalf("TerminalDefaults calls = %d, want %d", calls, cloudApplicationProbeRounds)
	}
}

func TestProbeCloudApplicationPreservesTransportTimeout(t *testing.T) {
	probe := cloudApplicationProbeFunc(func(ctx context.Context, _ *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := probeCloudApplication(ctx, probe); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("application probe error = %v, want deadline exceeded", err)
	}
}

func TestCloudRelayConcurrencyExhaustionIsRetryable(t *testing.T) {
	err := cloudConnectionError(&cloudclient.EntitlementError{Failure: &cloudv1.CloudEntitlementFailure{
		Code: cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED,
	}})
	var runtimeErr *clientruntime.Error
	if !errors.As(err, &runtimeErr) || runtimeErr.Code != clientruntime.ErrorRelayConcurrencyExhausted || !runtimeErr.Retryable {
		t.Fatalf("relay concurrency classification = %#v", err)
	}
}

func TestReleaseConfirmedCloudPeerUsesBoundedCleanupContext(t *testing.T) {
	want := errors.New("release failed")
	err := releaseConfirmedCloudPeer(confirmedCloudPeerReleaseFunc(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > cloudSessionReleaseTimeout {
			t.Fatalf("release deadline = %v, ok=%t", deadline, ok)
		}
		return want
	}))
	if !errors.Is(err, want) {
		t.Fatalf("release error = %v, want %v", err, want)
	}
}

func TestProbeCloudApplicationTraversesStampedProtocolAPI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := memory.NewPair()
	defer clientTransport.Close()
	defer serverTransport.Close()
	stamp := clientruntime.EndpointSessionStamp{EndpointID: "cloud-device", RouteID: "cloud", Generation: 17}
	observed := make(chan *apipb.EndpointSessionStamp, cloudApplicationProbeRounds)
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- serveCloudApplicationProbe(serverTransport, observed)
	}()

	protocolClient := internalprotocol.NewClient(clientTransport)
	defer protocolClient.Close()
	if err := protocolClient.Hello(ctx, internalprotocol.Hello{Version: wire.Version, Client: "cloud-probe-test"}); err != nil {
		t.Fatal(err)
	}
	application, err := protocoladapter.NewApplicationClientWithObservedPath(protocolClient, stamp, "direct")
	if err != nil {
		t.Fatal(err)
	}
	if err := probeCloudApplication(ctx, application.ApplicationSession); err != nil {
		t.Fatalf("probe through stamped application session: %v", err)
	}
	for round := 1; round <= cloudApplicationProbeRounds; round++ {
		if got := <-observed; got.GetEndpointId() != string(stamp.EndpointID) || got.GetRouteId() != string(stamp.RouteID) || got.GetGeneration() != uint64(stamp.Generation) {
			t.Fatalf("daemon API observed round %d stamp=%#v, want %+v", round, got, stamp)
		}
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

type cloudProbeAdmission struct{}

type cloudProbeAdmissionLease struct{}

func (cloudProbeAdmissionLease) Release() {}

func (cloudProbeAdmission) Acquire(_ context.Context, command *apipb.CommandEnvelope, capability apipb.ApiCapability) (apilayer.AdmissionLease, error) {
	if command.GetContext() == nil || command.GetContext().GetRequestId() == "" || command.GetContext().GetApiVersion().GetMajor() != 1 || command.GetContext().GetSession() == nil {
		return nil, errors.New("application probe reached API admission without request context")
	}
	if capability != apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE {
		return nil, fmt.Errorf("application probe requested capability %s", capability)
	}
	return cloudProbeAdmissionLease{}, nil
}

type cloudProbeTerminalController struct {
	apilayer.TerminalController
	observed chan<- *apipb.EndpointSessionStamp
}

func (controller cloudProbeTerminalController) TerminalDefaults(_ context.Context, stamp *apipb.EndpointSessionStamp, _ *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
	controller.observed <- proto.Clone(stamp).(*apipb.EndpointSessionStamp)
	return &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{DefaultCommand: []string{"sh"}}}, nil
}

func serveCloudApplicationProbe(connection transport.Transport, observed chan<- *apipb.EndpointSessionStamp) error {
	if err := receiveCloudProbeHello(connection); err != nil {
		return err
	}
	service := apilayer.NewService(cloudProbeAdmission{}, nil, nil, cloudProbeTerminalController{observed: observed})
	for round := 1; round <= cloudApplicationProbeRounds; round++ {
		channel, frameType, payload, err := receiveCloudProbeFrame(connection)
		if err != nil || channel != 0 || frameType != wire.TypeRequest {
			return fmt.Errorf("probe request round %d frame channel=%d type=%d: %w", round, channel, frameType, err)
		}
		request, err := internalprotocol.DecodeRequestPayload(payload)
		if err != nil || request.Method != "api.execute" {
			return fmt.Errorf("decode probe request round %d method=%q: %w", round, request.Method, err)
		}
		var command apipb.CommandEnvelope
		if err := proto.Unmarshal(request.Params, &command); err != nil {
			return err
		}
		resultPayload, err := proto.Marshal(service.Execute(context.Background(), &command))
		if err != nil {
			return err
		}
		response, err := internalprotocol.EncodeResponsePayload(internalprotocol.Response{ID: request.ID, Result: resultPayload})
		if err != nil {
			return err
		}
		if err := sendCloudProbeFrame(connection, 0, wire.TypeResponse, response); err != nil {
			return err
		}
	}
	return nil
}

func receiveCloudProbeHello(connection transport.Transport) error {
	channel, frameType, payload, err := receiveCloudProbeFrame(connection)
	if err != nil || channel != 0 || frameType != wire.TypeHello {
		return fmt.Errorf("probe Hello frame channel=%d type=%d: %w", channel, frameType, err)
	}
	if _, err := internalprotocol.DecodeHelloPayload(payload); err != nil {
		return err
	}
	response, err := internalprotocol.EncodeHelloPayload(internalprotocol.Hello{Version: wire.Version, Server: "probe-daemon"})
	if err != nil {
		return err
	}
	return sendCloudProbeFrame(connection, 0, wire.TypeHello, response)
}

func receiveCloudProbeFrame(connection transport.Transport) (uint16, uint8, []byte, error) {
	frame, err := connection.Recv()
	if err != nil {
		return 0, 0, nil, err
	}
	return wire.DecodeFrame(frame)
}

func sendCloudProbeFrame(connection transport.Transport, channel uint16, frameType uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, frameType, payload)
	if err != nil {
		return err
	}
	return connection.Send(frame)
}
