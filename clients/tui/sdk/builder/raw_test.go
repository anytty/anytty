package builder

import (
	"testing"

	pb "github.com/anytty/anytty/proto/ui/protobuf"
)

func TestRawKeepsPointerIdentity(t *testing.T) {
	box := &pb.Box{Id: "memo"}
	if got := Raw(box).Build(); got != box {
		t.Fatalf("Raw().Build() = %p, want the original %p", got, box)
	}
	parent := Col(Text("a"), Raw(box)).Build()
	if len(parent.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(parent.Children))
	}
	if parent.Children[1] != box {
		t.Fatal("composed child lost pointer identity")
	}
}
