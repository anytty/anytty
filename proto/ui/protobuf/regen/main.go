// Command regen regenerates tui2/protobuf/tui2.pb.go without a protoc binary:
// it reconstructs the FileDescriptorProto from the compiled-in descriptor,
// appends the endpoint metadata fields of MethodParams (tui2/proto/tui2.proto
// fields 17..20) and feeds a CodeGeneratorRequest to the protoc-gen-go plugin.
//
// Usage: go run ./tui2/protobuf/regen > tui2/protobuf/tui2.pb.go
package main

import (
	"bytes"
	"flag"
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
	outputPath := flag.String("out", "", "generated Go output path (default stdout)")
	flag.Parse()
	file := protodesc.ToFileDescriptorProto(protobuf.File_tui2_proto_tui2_proto)
	findMessage := func(name string) *descriptorpb.DescriptorProto {
		for _, message := range file.MessageType {
			if message.GetName() == name {
				return message
			}
		}
		return nil
	}
	params := findMessage("MethodParams")
	data := findMessage("MethodData")
	if params == nil || data == nil {
		fmt.Fprintln(os.Stderr, "regen: MethodParams/MethodData not found in descriptor")
		os.Exit(1)
	}
	appendField := func(message *descriptorpb.DescriptorProto, fieldType descriptorpb.FieldDescriptorProto_Type, number int32, name, jsonName string, repeated bool) {
		for _, field := range message.Field {
			if field.GetNumber() == number {
				return
			}
		}
		label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
		if repeated {
			label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
		}
		message.Field = append(message.Field, &descriptorpb.FieldDescriptorProto{
			Name:     proto.String(name),
			JsonName: proto.String(jsonName),
			Number:   proto.Int32(number),
			Label:    label.Enum(),
			Type:     fieldType.Enum(),
		})
	}
	stringField := func(message *descriptorpb.DescriptorProto, number int32, name, jsonName string, repeated bool) {
		appendField(message, descriptorpb.FieldDescriptorProto_TYPE_STRING, number, name, jsonName, repeated)
	}
	bytesField := func(message *descriptorpb.DescriptorProto, number int32, name, jsonName string, repeated bool) {
		appendField(message, descriptorpb.FieldDescriptorProto_TYPE_BYTES, number, name, jsonName, repeated)
	}
	// appendMapField adds a proto3 map<key,string> field: a repeated message
	// field whose nested entry has key=1/value=2 fields.
	appendMapField := func(message *descriptorpb.DescriptorProto, number int32, name, jsonName, entryName string, valueType descriptorpb.FieldDescriptorProto_Type) {
		for _, field := range message.Field {
			if field.GetNumber() == number {
				return
			}
		}
		message.NestedType = append(message.NestedType, &descriptorpb.DescriptorProto{
			Name: proto.String(entryName),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name: proto.String("key"), JsonName: proto.String("key"), Number: proto.Int32(1),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				},
				{
					Name: proto.String("value"), JsonName: proto.String("value"), Number: proto.Int32(2),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  valueType.Enum(),
				},
			},
			Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
		})
		message.Field = append(message.Field, &descriptorpb.FieldDescriptorProto{
			Name: proto.String(name), JsonName: proto.String(jsonName), Number: proto.Int32(number),
			Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
			Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
			TypeName: proto.String(".anytty.tui2.v1." + message.GetName() + "." + entryName),
		})
	}
	stringField(params, 17, "kind", "kind", false)
	stringField(params, 18, "socket", "socket", false)
	stringField(params, 19, "connect_mode", "connectMode", false)
	stringField(params, 20, "address", "address", false)
	// Direct/Cloud connection metadata (REMOTE.zh-CN.md §P2, append-only).
	stringField(params, 21, "signaling_addresses", "signalingAddresses", true)
	stringField(params, 22, "ice_tcp_addresses", "iceTcpAddresses", true)
	stringField(params, 23, "daemon_device_id", "daemonDeviceId", false)
	stringField(params, 24, "daemon_fingerprint", "daemonFingerprint", false)
	stringField(params, 25, "credential_dir", "credentialDir", false)
	stringField(params, 26, "credential_ref", "credentialRef", false)
	stringField(params, 27, "cloud_gateway_address", "cloudGatewayAddress", false)
	// Generic access forwarding (PROTOCOL §4 access.call, append-only).
	bytesField(params, 28, "access_command", "accessCommand", false)
	bytesField(data, 5, "access_result", "accessResult", false)
	// Clamped copy-view offset reported by terminal.scroll/history.window
	// (PROTOCOL §4, append-only).
	appendField(data, descriptorpb.FieldDescriptorProto_TYPE_INT32, 6, "offset", "offset", false)
	// access.stream.open (PROTOCOL §4, append-only): program-allocated stream
	// id plus the serialized access ResourceHandle.
	appendField(params, descriptorpb.FieldDescriptorProto_TYPE_UINT64, 29, "stream_id", "streamId", false)
	bytesField(params, 30, "access_resource", "accessResource", false)
	stringField(params, 31, "query", "query", false)
	stringField(params, 32, "search_mode", "searchMode", false)
	appendField(params, descriptorpb.FieldDescriptorProto_TYPE_BOOL, 33, "backward", "backward", false)
	stringField(params, 35, "clipboard_id", "clipboardId", false)
	appendField(data, descriptorpb.FieldDescriptorProto_TYPE_BOOL, 7, "found", "found", false)
	appendField(data, descriptorpb.FieldDescriptorProto_TYPE_BOOL, 8, "wrapped", "wrapped", false)
	appendField(data, descriptorpb.FieldDescriptorProto_TYPE_INT32, 9, "match_start", "matchStart", false)
	appendField(data, descriptorpb.FieldDescriptorProto_TYPE_INT32, 10, "match_end", "matchEnd", false)
	// Source carries the daemon-reported terminal grid so the picker can show
	// the size column without a separate method (append-only).
	if source := findMessage("Source"); source != nil {
		appendField(source, descriptorpb.FieldDescriptorProto_TYPE_INT32, 13, "cols", "cols", false)
		appendField(source, descriptorpb.FieldDescriptorProto_TYPE_INT32, 14, "rows", "rows", false)
		appendMapField(source, 15, "tags", "tags", "TagsEntry", descriptorpb.FieldDescriptorProto_TYPE_STRING)
		stringField(source, 16, "endpoint_label", "endpointLabel", false)
		appendField(source, descriptorpb.FieldDescriptorProto_TYPE_INT64, 17, "last_output_ms", "lastOutputMs", false)
		// attachment_count is the daemon-reported observer count for a terminal
		// source (append-only; the pane chrome derives the visible count from
		// it plus the local pane count).
		appendField(source, descriptorpb.FieldDescriptorProto_TYPE_INT32, 18, "attachment_count", "attachmentCount", false)
	}
	appendMapField(params, 34, "tags", "tags", "TagsEntry", descriptorpb.FieldDescriptorProto_TYPE_STRING)
	// view identifies the pane's independent terminal viewport (scroll/copy).
	// Append-only: field 34 is params.tags and 35 is clipboard_id, so the next
	// free field is 36.
	stringField(params, 36, "view", "view", false)
	// StreamFrame wire_type is append-only too: add it when an older compiled
	// descriptor already carries the message without the field.
	if stream := findMessage("StreamFrame"); stream != nil {
		appendField(stream, descriptorpb.FieldDescriptorProto_TYPE_UINT32, 6, "wire_type", "wireType", false)
	}
	// StreamFrame is not present in the compiled descriptor until this tool
	// regenerates it; append the whole message when missing.
	if findMessage("StreamFrame") == nil {
		streamKind := &descriptorpb.FieldDescriptorProto{
			Name: proto.String("kind"), JsonName: proto.String("kind"), Number: proto.Int32(2),
			Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
		}
		file.MessageType = append(file.MessageType, &descriptorpb.DescriptorProto{
			Name: proto.String("StreamFrame"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name: proto.String("stream_id"), JsonName: proto.String("streamId"), Number: proto.Int32(1),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
				},
				streamKind,
				{
					Name: proto.String("payload"), JsonName: proto.String("payload"), Number: proto.Int32(3),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
				},
				{
					Name: proto.String("offset"), JsonName: proto.String("offset"), Number: proto.Int32(4),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
				},
				{
					Name: proto.String("error"), JsonName: proto.String("error"), Number: proto.Int32(5),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				},
				{
					Name: proto.String("wire_type"), JsonName: proto.String("wireType"), Number: proto.Int32(6),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
				},
			},
		})
	}

	if findMessage("ViewDelta") == nil {
		file.MessageType = append(file.MessageType, &descriptorpb.DescriptorProto{
			Name: proto.String("ViewDelta"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name: proto.String("epoch"), JsonName: proto.String("epoch"), Number: proto.Int32(1),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
				},
				{
					Name: proto.String("rev"), JsonName: proto.String("rev"), Number: proto.Int32(2),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
				},
				{
					Name: proto.String("rev_base"), JsonName: proto.String("revBase"), Number: proto.Int32(3),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
				},
				{
					Name: proto.String("keys"), JsonName: proto.String("keys"), Number: proto.Int32(4),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".anytty.tui2.v1.Keys"),
				},
				{
					Name: proto.String("patches"), JsonName: proto.String("patches"), Number: proto.Int32(5),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".anytty.tui2.v1.Patch"),
				},
			},
		})
	}
	if findMessage("Patch") == nil {
		file.MessageType = append(file.MessageType, &descriptorpb.DescriptorProto{
			Name: proto.String("Patch"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name: proto.String("op"), JsonName: proto.String("op"), Number: proto.Int32(1),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				},
				{
					Name: proto.String("path"), JsonName: proto.String("path"), Number: proto.Int32(2),
					Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
				},
				{
					Name: proto.String("index"), JsonName: proto.String("index"), Number: proto.Int32(3),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
				},
				{
					Name: proto.String("from"), JsonName: proto.String("from"), Number: proto.Int32(4),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
				},
				{
					Name: proto.String("to"), JsonName: proto.String("to"), Number: proto.Int32(5),
					Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:  descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
				},
				{
					Name: proto.String("box"), JsonName: proto.String("box"), Number: proto.Int32(6),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".anytty.tui2.v1.Box"),
				},
			},
		})
	}

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
	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, []byte(response.File[0].GetContent()), 0644); err != nil {
			fmt.Fprintln(os.Stderr, "regen:", err)
			os.Exit(1)
		}
		return
	}
	if _, err := os.Stdout.WriteString(response.File[0].GetContent()); err != nil {
		fmt.Fprintln(os.Stderr, "regen:", err)
		os.Exit(1)
	}
}
