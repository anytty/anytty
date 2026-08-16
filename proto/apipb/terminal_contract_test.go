package apipb

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestTerminalInputRoundTripKeepsAttachmentAndOperationFence(t *testing.T) {
	session := &EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 9}
	command := &CommandEnvelope{
		Context: &RequestContext{
			RequestId:  "input-1",
			ApiVersion: &ApiVersion{Major: 1},
			Session:    session,
		},
		Command: &CommandEnvelope_TerminalInput{TerminalInput: &TerminalInputCommand{
			Attachment: &ResourceHandle{OpaqueToken: []byte("attachment-1"), Kind: ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: session, Generation: 3},
			Operation: &OperationStamp{
				Session: session, OperationId: "input-op-1",
			},
			Data: []byte("echo test\r"),
		}}}
	payload, err := proto.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	var decoded CommandEnvelope
	if err := proto.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	input := decoded.GetTerminalInput()
	if string(input.GetAttachment().GetOpaqueToken()) != "attachment-1" || input.GetAttachment().GetSession().GetGeneration() != 9 || input.GetOperation().GetOperationId() != "input-op-1" || string(input.GetData()) != "echo test\r" {
		t.Fatalf("terminal input did not preserve fences: %#v", input)
	}
}

func TestTerminalApplicationContractHidesProtocolChannel(t *testing.T) {
	descriptor := (&AttachmentHandle{}).ProtoReflect().Descriptor()
	if descriptor.Fields().ByName("channel") != nil {
		t.Fatal("public attachment handle must not expose protocol channel")
	}
	assertFieldNumber(t, descriptor, "resource", 1)
	assertFieldNumber(t, descriptor, "terminal", 2)
	assertFieldNumber(t, descriptor, "operation", 3)
	assertFieldNumber(t, (&CommandEnvelope{}).ProtoReflect().Descriptor(), "terminal_attach", 29)
	assertFieldNumber(t, (&CommandEnvelope{}).ProtoReflect().Descriptor(), "terminal_input", 31)
	assertFieldNumber(t, (&CommandEnvelope{}).ProtoReflect().Descriptor(), "path_list_directories", 34)
}
