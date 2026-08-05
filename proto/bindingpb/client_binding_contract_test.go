package bindingpb

import (
	"bytes"
	"os"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestClientBindingRoundTripPreservesUnknownFields(t *testing.T) {
	event := &EventEnvelope{
		AbiVersion: 1, Sequence: 7,
		Event: &EventEnvelope_Execute{Execute: &ExecuteResult{
			OperationHandle: 11, SessionHandle: 12,
			Result: &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_TerminalList{TerminalList: &apipb.TerminalListResult{}}},
		}},
	}
	payload, err := proto.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	unknown := protowire.AppendVarint(protowire.AppendTag(nil, 99, protowire.VarintType), 42)
	payload = append(payload, unknown...)
	decoded := &EventEnvelope{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.GetExecute().GetResult().GetTerminalList() == nil || decoded.GetExecute().GetOperationHandle() != 11 {
		t.Fatalf("binding event did not round trip: %#v", decoded)
	}
	reencoded, err := proto.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(reencoded, unknown) {
		t.Fatalf("binding event lost unknown field: %x", reencoded)
	}
}

func TestOpenSessionCarriesOnlyRedactedConnectionSnapshot(t *testing.T) {
	open := (&OpenSessionResult{}).ProtoReflect().Descriptor()
	if field := open.Fields().ByName("connection"); field == nil || field.Number() != 6 {
		t.Fatal("OpenSessionResult.connection must remain field 6")
	}
	snapshot := (&ConnectionSnapshot{}).ProtoReflect().Descriptor()
	for _, forbidden := range []string{"ip", "address", "hostname", "sdp", "credential", "terminal"} {
		if snapshot.Fields().ByName(protoreflect.Name(forbidden)) != nil {
			t.Fatalf("ConnectionSnapshot must not expose %s", forbidden)
		}
	}
}

func TestClientBindingDescriptorBaseline(t *testing.T) {
	payload, err := os.ReadFile("testdata/client-binding-v1.pb")
	if err != nil {
		t.Fatal(err)
	}
	baseline := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(payload, baseline); err != nil {
		t.Fatal(err)
	}
	if len(baseline.GetFile()) != 1 || baseline.GetFile()[0].GetName() != "bindingpb/client_binding.proto" {
		t.Fatalf("binding descriptor files = %#v", baseline.GetFile())
	}
	current := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		protodesc.ToFileDescriptorProto(File_bindingpb_client_binding_proto),
	}}
	if !proto.Equal(baseline, current) {
		t.Fatal("client binding descriptor differs from testdata/client-binding-v1.pb")
	}
}
