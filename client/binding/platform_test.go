package binding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

func TestPlatformBrokerCorrelatesProtoResponsesAndRejectsLateCompletion(t *testing.T) {
	broker := NewPlatformBroker()
	defer broker.Close()
	result := make(chan *bindingpb.PlatformResponse, 1)
	errCh := make(chan error, 1)
	go func() {
		response, err := broker.Exchange(context.Background(), &bindingpb.PlatformRequest{
			Request: &bindingpb.PlatformRequest_CredentialSign{CredentialSign: &bindingpb.CredentialSignRequest{
				CredentialRef: "credential:studio", Payload: []byte("proof"),
			}},
		})
		result <- response
		errCh <- err
	}()
	payload, err := broker.NextRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	request := &bindingpb.PlatformRequest{}
	if err := proto.Unmarshal(payload, request); err != nil {
		t.Fatal(err)
	}
	if request.GetRequestId() == 0 || string(request.GetCredentialSign().GetPayload()) != "proof" {
		t.Fatalf("platform request = %#v", request)
	}
	responsePayload, _ := proto.Marshal(&bindingpb.PlatformResponse{
		RequestId: request.GetRequestId(),
		Response:  &bindingpb.PlatformResponse_CredentialSign{CredentialSign: &bindingpb.CredentialSignResponse{Signature: []byte("signature")}},
	})
	if err := broker.Complete(responsePayload); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if response := <-result; string(response.GetCredentialSign().GetSignature()) != "signature" {
		t.Fatalf("platform response = %#v", response)
	}
	if err := broker.Complete(responsePayload); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("duplicate completion error = %v", err)
	}
}

func TestPlatformBrokerCancellationAndCloseUnblockBothSides(t *testing.T) {
	broker := NewPlatformBroker()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := broker.Exchange(ctx, &bindingpb.PlatformRequest{
		Request: &bindingpb.PlatformRequest_CredentialDelete{CredentialDelete: &bindingpb.CredentialDeleteRequest{CredentialRef: "credential:studio"}},
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled exchange error = %v", err)
	}
	nextErr := make(chan error, 1)
	go func() {
		_, err := broker.NextRequest(context.Background())
		nextErr <- err
	}()
	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-nextErr:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("next request close error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("broker close did not unblock NextRequest")
	}
}

func TestEnginePairingAndCredentialOperationsUseGenericEvents(t *testing.T) {
	host := &extendedBindingHost{bindingHost: bindingHost{session: newBindingSession()}}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	importPayload, _ := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_ImportPairing{ImportPairing: &bindingpb.ImportPairingRequest{RequestId: "pair-1", PortablePayload: "MXP2-proof"}}})
	importOperation, err := engine.EngineCommand(importPayload)
	if err != nil {
		t.Fatal(err)
	}
	importEvent := nextBindingEvent(t, engine)
	if importEvent.GetAbiVersion() != ABIVersion || importEvent.GetImportPairing().GetOperationHandle() != importOperation || importEvent.GetImportPairing().GetEndpoint().GetEndpointId() != "studio" {
		t.Fatalf("pairing event = %#v", importEvent)
	}
	if err := engine.Release(importOperation); err != nil {
		t.Fatal(err)
	}
	deletePayload, _ := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_DeleteCredential{DeleteCredential: &bindingpb.DeleteCredentialRequest{RequestId: "delete-1", CredentialRef: "credential:studio"}}})
	deleteOperation, err := engine.EngineCommand(deletePayload)
	if err != nil {
		t.Fatal(err)
	}
	deleteEvent := nextBindingEvent(t, engine)
	if deleteEvent.GetDeleteCredential().GetOperationHandle() != deleteOperation || deleteEvent.GetDeleteCredential().GetError() != nil || host.deleted != "credential:studio" {
		t.Fatalf("delete event = %#v deleted=%q", deleteEvent, host.deleted)
	}
	registryPayload, _ := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_EndpointRegistryGet{EndpointRegistryGet: &bindingpb.EndpointRegistryGetRequest{RequestId: "registry-1"}}})
	registryOperation, err := engine.EngineCommand(registryPayload)
	if err != nil {
		t.Fatal(err)
	}
	registryEvent := nextBindingEvent(t, engine)
	if registryEvent.GetEndpointRegistryGet().GetOperationHandle() != registryOperation || registryEvent.GetEndpointRegistryGet().GetRegistry().GetDefaultEndpointId() != "studio" {
		t.Fatalf("registry event = %#v", registryEvent)
	}
}

type extendedBindingHost struct {
	bindingHost
	deleted string
}

func (host *extendedBindingHost) ImportPairing(_ context.Context, request *bindingpb.ImportPairingRequest) (*bindingpb.ImportPairingResult, error) {
	return &bindingpb.ImportPairingResult{Endpoint: &remoteauthpb.EndpointConfigV1{EndpointId: "studio"}, AuthorizationRequired: true}, nil
}

func (host *extendedBindingHost) DeleteCredential(_ context.Context, request *bindingpb.DeleteCredentialRequest) error {
	host.deleted = request.GetCredentialRef()
	return nil
}

func (host *extendedBindingHost) GetEndpointRegistry(context.Context, *bindingpb.EndpointRegistryGetRequest) (*bindingpb.EndpointRegistryGetResult, error) {
	return &bindingpb.EndpointRegistryGetResult{Registry: &remoteauthpb.EndpointRegistryV1{DefaultEndpointId: "studio"}}, nil
}

func (host *extendedBindingHost) UpsertEndpoint(_ context.Context, request *bindingpb.EndpointUpsertRequest) (*bindingpb.EndpointUpsertResult, error) {
	return &bindingpb.EndpointUpsertResult{Endpoint: request.GetEndpoint()}, nil
}

func (host *extendedBindingHost) DeleteEndpoint(_ context.Context, request *bindingpb.EndpointDeleteRequest) (*bindingpb.EndpointDeleteResult, error) {
	return &bindingpb.EndpointDeleteResult{EndpointId: request.GetEndpointId()}, nil
}

var _ PairingHost = (*extendedBindingHost)(nil)
var _ CredentialHost = (*extendedBindingHost)(nil)
var _ EndpointRegistryHost = (*extendedBindingHost)(nil)
