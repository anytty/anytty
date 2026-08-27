package cloudv1

import (
	"os"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TestCloudV1DescriptorBaseline 锁定当前 Cloud v1 跨进程契约；直接升级也必须显式更新并审查 baseline。
func TestCloudV1DescriptorBaseline(t *testing.T) {
	payload, err := os.ReadFile("testdata/cloud-v1.pb")
	if err != nil {
		t.Fatal(err)
	}
	baseline := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(payload, baseline); err != nil {
		t.Fatalf("decode Cloud descriptor baseline: %v", err)
	}
	current := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		protodesc.ToFileDescriptorProto(File_cloud_v1_common_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_account_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_commerce_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_certificate_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_edge_config_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_runtime_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_enrollment_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_usage_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_edge_control_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_ticket_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_directory_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_client_gateway_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_agent_gateway_proto),
		protodesc.ToFileDescriptorProto(File_cloud_v1_operator_proto),
	}}
	if !proto.Equal(baseline, current) {
		t.Fatal("Cloud v1 descriptor differs from testdata/cloud-v1.pb")
	}
}

// TestS2GatewayContracts 锁定 challenge-first 双向流，且 envelope 不能退化成无 generation 的 unary API。
func TestS2GatewayContracts(t *testing.T) {
	for _, service := range []protoreflect.ServiceDescriptor{
		File_cloud_v1_agent_gateway_proto.Services().ByName("AgentGateway"),
		File_cloud_v1_client_gateway_proto.Services().ByName("ClientGateway"),
	} {
		if service == nil {
			t.Fatal("R5 gateway service is missing")
		}
		method := service.Methods().ByName("Connect")
		if method == nil || !method.IsStreamingClient() || !method.IsStreamingServer() {
			t.Fatalf("%s.Connect must be bidirectional streaming", service.FullName())
		}
	}
	assertEnvelopeFields(t, (&ClientSignal{}).ProtoReflect().Descriptor(), map[protoreflect.Name]protoreflect.FieldNumber{"hello": 20, "offer": 21, "path_selected": 22})
	assertEnvelopeFields(t, (&EdgeSignal{}).ProtoReflect().Descriptor(), map[protoreflect.Name]protoreflect.FieldNumber{"ready": 20, "answer": 21, "rejected": 22, "challenge": 23, "closed": 24, "presence": 25})
	assertEnvelopeFields(t, (&AgentEvent{}).ProtoReflect().Descriptor(), map[protoreflect.Name]protoreflect.FieldNumber{"hello": 20, "heartbeat": 21, "answer": 22, "rejected": 23, "authorization": 24, "lifecycle_result": 25})
	assertEnvelopeFields(t, (&EdgeCommand{}).ProtoReflect().Descriptor(), map[protoreflect.Name]protoreflect.FieldNumber{"ready": 20, "offer": 21, "authorize": 22, "challenge": 23, "lifecycle": 24, "edge_reselect": 25})
	challenge := (&EdgeChallenge{}).ProtoReflect().Descriptor()
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{"nonce": 1, "edge_id": 2, "edge_boot_id": 3, "stream_id": 4, "issued_at": 5, "expires_at": 6, "target": 7} {
		field := challenge.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("EdgeChallenge.%s field=%v want=%d", name, field, number)
		}
	}
}

func TestS2B3BindingKeyBundleContract(t *testing.T) {
	bundle := (&KeyBundle{}).ProtoReflect().Descriptor()
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{"revision": 1, "issued_at": 2, "expires_at": 3, "keys": 4} {
		field := bundle.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("KeyBundle.%s field=%v want=%d", name, field, number)
		}
	}
	welcome := (&EdgeWelcome{}).ProtoReflect().Descriptor()
	field := welcome.Fields().ByName("binding_key_bundle")
	if field == nil || field.Number() != 3 || field.Message() != bundle {
		t.Fatalf("EdgeWelcome.binding_key_bundle=%v", field)
	}
	if welcome.Fields().ByName("binding_verification_keys") != nil {
		t.Fatal("legacy naked binding_verification_keys remains in EdgeWelcome")
	}
	command := (&ControllerCommand{}).ProtoReflect().Descriptor()
	update := command.Fields().ByName("binding_key_bundle")
	if update == nil || update.Number() != 24 || update.Message() != bundle {
		t.Fatalf("ControllerCommand.binding_key_bundle=%v", update)
	}
}

func TestClientHelloSeparatesCapabilityAndPairingAdmission(t *testing.T) {
	hello := (&ClientHello{}).ProtoReflect().Descriptor()
	authorization := hello.Oneofs().ByName("authorization")
	if authorization == nil || authorization.Fields().Len() != 2 {
		t.Fatalf("ClientHello.authorization = %v, want exactly two variants", authorization)
	}
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{"cloud_route_grant": 10, "pairing_admission": 11} {
		field := authorization.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("ClientHello.authorization.%s field = %v, want number %d", name, field, number)
		}
	}
	if hello.Fields().ByName("route_grant") != nil || hello.Fields().ByName("access_mode") != nil {
		t.Fatal("ClientHello must not restore generic route_grant or caller-selected access_mode")
	}
	if File_cloud_v1_ticket_proto.Messages().ByName("PairingRouteGrantClaims") != nil {
		t.Fatal("PairingRouteGrantClaims must not remain in the Cloud protocol")
	}
}

// TestEdgeControlIsBidirectionalStreaming 保证 Edge 连接数不会回退为轮询或一元 RPC。
func TestEdgeControlIsBidirectionalStreaming(t *testing.T) {
	service := File_cloud_v1_edge_control_proto.Services().ByName("EdgeControl")
	if service == nil {
		t.Fatal("EdgeControl service is missing")
	}
	method := service.Methods().ByName("Connect")
	if method == nil || !method.IsStreamingClient() || !method.IsStreamingServer() {
		t.Fatalf("EdgeControl.Connect must be bidirectional streaming: %#v", method)
	}
	assertEnvelopeFields(t, (&EdgeEvent{}).ProtoReflect().Descriptor(), map[protoreflect.Name]protoreflect.FieldNumber{
		"hello": 20, "snapshot_begin": 21, "snapshot_chunk": 22, "snapshot_end": 23, "runtime_delta": 24, "heartbeat": 25, "config_applied": 26, "relay_reserve": 28, "command_result": 29, "certificate_applied": 30, "relay_renew": 31, "relay_settle": 32, "relay_query": 33, "daemon_state_query": 34, "identity_renew": 35, "identity_applied": 36, "daemon_connection_admission": 37, "relay_authorize": 38, "relay_usage_batch": 39, "daemon_state_sync": 40,
	})
	assertEnvelopeFields(t, (&ControllerCommand{}).ProtoReflect().Descriptor(), map[protoreflect.Name]protoreflect.FieldNumber{
		"welcome": 20, "snapshot_accepted": 21, "resync_required": 22, "desired_config": 23, "binding_key_bundle": 24, "relay_reserve": 25, "close_daemon": 26, "close_session": 27, "certificate_bundle": 28, "relay_renew": 29, "relay_settle": 30, "relay_query": 31, "daemon_state_delta": 32, "daemon_state_query_result": 33, "reselect_daemon_edge": 34, "identity_renew": 35, "daemon_connection_admission": 36, "relay_authorize": 37, "relay_usage_ack": 38, "relay_account_action": 39, "daemon_state_sync_chunk": 40, "daemon_state_sync_end": 41,
	})
}

func TestDaemonLifecycleProtocol(t *testing.T) {
	state := File_cloud_v1_enrollment_proto.Enums().ByName("DaemonState")
	if state == nil || state.Values().Len() != 4 {
		t.Fatalf("DaemonState = %v", state)
	}
	for name, number := range map[protoreflect.Name]protoreflect.EnumNumber{
		"DAEMON_STATE_UNSPECIFIED": 0,
		"DAEMON_STATE_ACTIVE":      1,
		"DAEMON_STATE_BLOCKED":     2,
		"DAEMON_STATE_DELETED":     3,
	} {
		value := state.Values().ByName(name)
		if value == nil || value.Number() != number {
			t.Fatalf("DaemonState.%s = %v, want %d", name, value, number)
		}
	}
	record := (&DaemonRecord{}).ProtoReflect().Descriptor()
	if record.Fields().ByName("revoked") != nil || record.Fields().ByName("revision") != nil {
		t.Fatal("DaemonRecord must not retain revoked or a shared revision")
	}
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{"state": 7, "state_revision": 8} {
		field := record.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("DaemonRecord.%s = %v, want %d", name, field, number)
		}
	}
	management := File_cloud_v1_enrollment_proto.Services().ByName("DaemonManagementService")
	if management == nil || management.Methods().Len() != 6 || management.Methods().ByName("ChangeMyDaemonState") == nil || management.Methods().ByName("ListMyDaemonEdges") == nil || management.Methods().ByName("ChangeMyDaemonEdgePreference") == nil || management.Methods().ByName("ReselectMyDaemonEdge") == nil || management.Methods().ByName("RevokeMyDaemon") != nil {
		t.Fatalf("DaemonManagementService methods = %v", management)
	}
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{"preferred_edge_id": 11, "edge_preference_revision": 12, "edge_preference_updated_at": 13} {
		field := record.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("DaemonRecord.%s = %v, want %d", name, field, number)
		}
	}
	claims := (&DaemonBindingClaims{}).ProtoReflect().Descriptor()
	if claims.Fields().ByName("edge_id") == nil || claims.Fields().ByName("revision") != nil || claims.Fields().ByName("state_revision") != nil {
		t.Fatal("DaemonBindingClaims must retain its target Edge without lifecycle revisions")
	}
	for _, name := range []protoreflect.Name{"binding_id", "edge_id"} {
		if claims.Fields().ByName(name) == nil {
			t.Fatalf("DaemonBindingClaims.%s is missing", name)
		}
	}
	command := (&DaemonLifecycleCommand{}).ProtoReflect().Descriptor()
	if command.Fields().ByName("daemon_state") == nil || command.Fields().ByName("agent_generation") == nil {
		t.Fatal("DaemonLifecycleCommand must target one state and Agent generation")
	}
	snapshot := (&DaemonStateSnapshot{}).ProtoReflect().Descriptor()
	if field := snapshot.Fields().ByName("daemons"); field == nil || field.Number() != 1 {
		t.Fatalf("DaemonStateSnapshot.daemons = %v", field)
	}
	welcome := (&EdgeWelcome{}).ProtoReflect().Descriptor()
	if welcome.Fields().ByName("daemon_states") != nil {
		t.Fatal("EdgeWelcome must not contain a global daemon state snapshot")
	}
	response := (&ChangeMyDaemonStateResponse{}).ProtoReflect().Descriptor()
	if response.Fields().Len() != 1 || response.Fields().ByName("daemon") == nil {
		t.Fatal("state mutation response must return only committed daemon state")
	}
}

// TestR8CertificateContracts 锁定简化后的证书双文件上传、单一 revision 和控制流回执。
func TestR8CertificateContracts(t *testing.T) {
	profile := (&CertificateProfile{}).ProtoReflect().Descriptor()
	if profile.Fields().ByName("private_key_pem") != nil || profile.Fields().ByName("certificate_chain_pem") != nil {
		t.Fatal("operator-visible CertificateProfile must not expose certificate secret bytes")
	}
	bundle := (&EdgeCertificateBundle{}).ProtoReflect().Descriptor()
	if bundle.Fields().ByName("target_edge_id") == nil || bundle.Fields().ByName("private_key_pem") == nil {
		t.Fatal("EdgeCertificateBundle must carry an explicit target and private key over mTLS")
	}
}

// TestDurableRelayReservationContracts locks the request/grant/settlement identity and journal stages.
func TestDurableRelayReservationContracts(t *testing.T) {
	request := (&RelayReserveRequest{}).ProtoReflect().Descriptor()
	if field := request.Fields().ByName("reservation_id"); field == nil || field.Number() != 1 {
		t.Fatalf("RelayReserveRequest.reservation_id field=%v", field)
	}
	for _, descriptor := range []protoreflect.MessageDescriptor{
		(&RelayGrant{}).ProtoReflect().Descriptor(), (&RelayRenewRequest{}).ProtoReflect().Descriptor(),
		(&RelaySettlement{}).ProtoReflect().Descriptor(), (&RelaySettlementAck{}).ProtoReflect().Descriptor(),
	} {
		if descriptor.Fields().ByName("reservation_id").Number() != 1 {
			t.Fatalf("%s does not use reservation_id at field 1", descriptor.FullName())
		}
	}
	journal := (&RelayJournalRecord{}).ProtoReflect().Descriptor()
	if journal.Fields().ByName("stage") == nil || journal.Fields().ByName("reserve_request") == nil || journal.Fields().ByName("settlement") == nil {
		t.Fatal("RelayJournalRecord is missing durable lifecycle fields")
	}
	if File_cloud_v1_runtime_proto.Messages().ByName("RelayAllocationSummary") != nil ||
		(&RuntimeSnapshot{}).ProtoReflect().Descriptor().Fields().ByName("allocations") != nil ||
		(&SnapshotChunk{}).ProtoReflect().Descriptor().Fields().ByName("allocations") != nil {
		t.Fatal("physical Relay allocations must remain Edge-local")
	}
}

func TestEnrollmentAndDirectoryExcludeRemovedHotPathRPCs(t *testing.T) {
	enrollment := File_cloud_v1_enrollment_proto.Services().ByName("EnrollmentService")
	if enrollment == nil || enrollment.Methods().Len() != 4 ||
		enrollment.Methods().ByName("BeginDaemonEnrollment") == nil || enrollment.Methods().ByName("CompleteDaemonEnrollment") == nil ||
		enrollment.Methods().ByName("BeginDaemonBindingRefresh") == nil || enrollment.Methods().ByName("CompleteDaemonBindingRefresh") == nil {
		t.Fatalf("EnrollmentService methods = %v", enrollment)
	}
	begin := enrollment.Methods().ByName("BeginDaemonEnrollment")
	challenge := (&DaemonEnrollmentChallenge{}).ProtoReflect().Descriptor()
	if begin.Output() != challenge || challenge.Fields().ByName("identity_challenge") == nil || challenge.Fields().ByName("edge_candidates") == nil {
		t.Fatalf("BeginDaemonEnrollment output = %v", begin.Output())
	}
	complete := enrollment.Methods().ByName("CompleteDaemonEnrollment")
	if field := complete.Input().Fields().ByName("edge_measurements"); field == nil || field.Number() != 3 || !field.IsList() {
		t.Fatalf("CompleteDaemonEnrollment.edge_measurements = %v", field)
	}
	directory := File_cloud_v1_directory_proto.Services().ByName("DirectoryService")
	if directory == nil || directory.Methods().Len() != 2 || directory.Methods().ByName("BeginClientRoute") == nil || directory.Methods().ByName("ResolveClientRoute") == nil {
		t.Fatalf("DirectoryService methods = %v", directory)
	}
}

func assertEnvelopeFields(t *testing.T, descriptor protoreflect.MessageDescriptor, wantPayload map[protoreflect.Name]protoreflect.FieldNumber) {
	t.Helper()
	want := map[protoreflect.Name]protoreflect.FieldNumber{
		"protocol_version": 1,
		"message_id":       2,
		"sender_id":        3,
		"boot_id":          4,
		"connection_id":    5,
		"stream_seq":       6,
		"sent_at":          7,
	}
	for name, number := range want {
		field := descriptor.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("%s.%s field number=%v want=%d", descriptor.FullName(), name, field, number)
		}
	}
	payload := descriptor.Oneofs().ByName("payload")
	if payload == nil || payload.Fields().Len() != len(wantPayload) {
		t.Fatalf("%s payload field count=%d want=%d", descriptor.FullName(), payload.Fields().Len(), len(wantPayload))
	}
	for name, number := range wantPayload {
		field := payload.Fields().ByName(name)
		if field == nil || field.Number() != number {
			t.Fatalf("%s payload %s field number=%v want=%d", descriptor.FullName(), name, field, number)
		}
	}
}
