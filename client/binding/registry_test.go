package binding

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

func TestRegistryOwnsEngineHandlesWithoutReuse(t *testing.T) {
	registry := NewRegistry()
	host := &bindingHost{session: newBindingSession()}
	first, err := registry.CreateEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	request, _ := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "registry-open", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if _, err := registry.OpenSession(first, request); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.NextEvent(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := registry.CloseEngine(first); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.OpenSession(first, request); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("closed engine handle error = %v", err)
	}
	second, err := registry.CreateEngine(&bindingHost{session: newBindingSession()})
	if err != nil {
		t.Fatal(err)
	}
	if second <= first {
		t.Fatalf("engine handle was reused: first=%d second=%d", first, second)
	}
	if err := registry.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.CreateEngine(&bindingHost{session: newBindingSession()}); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed registry create error = %v", err)
	}
}
