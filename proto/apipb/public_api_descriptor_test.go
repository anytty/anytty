package apipb

import (
	"os"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TestPublicAPIDescriptorBaseline 锁定完整公共 API descriptor。
// 任何字段、enum、oneof、reserved 或 message 变动都必须显式更新 baseline 并接受 schema review。
func TestPublicAPIDescriptorBaseline(t *testing.T) {
	payload, err := os.ReadFile("testdata/public-api-v1.pb")
	if err != nil {
		t.Fatal(err)
	}
	var baseline descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(payload, &baseline); err != nil {
		t.Fatalf("decode descriptor baseline: %v", err)
	}
	want := []string{
		"apipb/common.proto", "remoteauthpb/remote_auth.proto", "apipb/access_remote.proto",
		"apipb/storage.proto", "apipb/terminal.proto", "apipb/events.proto", "apipb/file.proto",
		"apipb/history.proto", "apipb/runtime.proto", "apipb/workbench.proto", "apipb/application.proto",
	}
	if len(baseline.GetFile()) != len(want) {
		t.Fatalf("descriptor baseline contains %d files, want %d", len(baseline.GetFile()), len(want))
	}
	for index, name := range want {
		if baseline.GetFile()[index].GetName() != name {
			t.Fatalf("descriptor file[%d]=%q want %q", index, baseline.GetFile()[index].GetName(), name)
		}
	}
	current := &descriptorpb.FileDescriptorSet{File: make([]*descriptorpb.FileDescriptorProto, 0, len(want))}
	for _, name := range want {
		descriptor, err := protoregistry.GlobalFiles.FindFileByPath(name)
		if err != nil {
			t.Fatalf("find generated descriptor %s: %v", name, err)
		}
		current.File = append(current.File, protodesc.ToFileDescriptorProto(descriptor))
	}
	if !proto.Equal(&baseline, current) {
		t.Fatal("public API descriptor differs from testdata/public-api-v1.pb; update the baseline only after schema compatibility review")
	}
}
