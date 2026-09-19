// Command regen regenerates tui2/protobuf/tui2.pb.go without a protoc binary:
// it reconstructs the FileDescriptorProto from the compiled-in descriptor,
// appends the endpoint metadata fields of MethodParams (tui2/proto/tui2.proto
// fields 17..20) and feeds a CodeGeneratorRequest to the protoc-gen-go plugin.
//
// Usage: go run ./tui2/protobuf/regen > tui2/protobuf/tui2.pb.go
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/anytty/anytty/proto/ui/protobuf"
)

func main() {
	file := protodesc.ToFileDescriptorProto(protobuf.File_tui2_proto_tui2_proto)
	var params *descriptorpb.DescriptorProto
	for _, message := range file.MessageType {
		if message.GetName() == "MethodParams" {
			params = message
		}
	}
	if params == nil {
		fmt.Fprintln(os.Stderr, "regen: MethodParams not found in descriptor")
		os.Exit(1)
	}
	present := map[int32]bool{}
	for _, field := range params.Field {
		present[field.GetNumber()] = true
	}
	appendField := func(number int32, name, jsonName string, repeated bool) {
		if present[number] {
			return
		}
		label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
		if repeated {
			label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
		}
		params.Field = append(params.Field, &descriptorpb.FieldDescriptorProto{
			Name:     proto.String(name),
			JsonName: proto.String(jsonName),
			Number:   proto.Int32(number),
			Label:    label.Enum(),
			Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		})
	}
	appendField(17, "kind", "kind", false)
	appendField(18, "socket", "socket", false)
	appendField(19, "connect_mode", "connectMode", false)
	appendField(20, "address", "address", false)
	// Direct/Cloud connection metadata (REMOTE.zh-CN.md §P2, append-only).
	appendField(21, "signaling_addresses", "signalingAddresses", true)
	appendField(22, "ice_tcp_addresses", "iceTcpAddresses", true)
	appendField(23, "daemon_device_id", "daemonDeviceId", false)
	appendField(24, "daemon_fingerprint", "daemonFingerprint", false)
	appendField(25, "credential_dir", "credentialDir", false)
	appendField(26, "credential_ref", "credentialRef", false)
	appendField(27, "cloud_gateway_address", "cloudGatewayAddress", false)

	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"tui2/proto/tui2.proto"},
		Parameter:      proto.String("paths=source_relative"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{file},
	}
	input, err := proto.Marshal(request)
	if err != nil {
		fmt.Fprintln(os.Stderr, "regen:", err)
		os.Exit(1)
	}
	cmd := exec.Command("go", "run", "google.golang.org/protobuf/cmd/protoc-gen-go")
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "regen: protoc-gen-go:", err)
		os.Exit(1)
	}
	response := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(output, response); err != nil {
		fmt.Fprintln(os.Stderr, "regen: decode response:", err)
		os.Exit(1)
	}
	if response.GetError() != "" {
		fmt.Fprintln(os.Stderr, "regen: protoc-gen-go:", response.GetError())
		os.Exit(1)
	}
	if len(response.File) != 1 {
		fmt.Fprintf(os.Stderr, "regen: want exactly one generated file, got %d\n", len(response.File))
		os.Exit(1)
	}
	if _, err := os.Stdout.WriteString(response.File[0].GetContent()); err != nil {
		fmt.Fprintln(os.Stderr, "regen:", err)
		os.Exit(1)
	}
}
