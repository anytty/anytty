package app

import (
	"context"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
)

// Exercise the public runtime constructor: terminal surface IDs are not plugin
// routing IDs and may contain characters unsuitable for registration/log paths.
func TestInteractiveRuntimeUsesIndependentPluginIdentity(t *testing.T) {
	root, _, service := pluginFixture(t)
	root.RuntimeSurfaceID = "cmd/anytty-v3:123:456"
	runtime := NewInteractiveRuntime(root, NewFakeTerminalHost(16), pluginWiringRunner{}, LiveDeps{Plugins: service, PluginTUIInstanceID: "tui-routing-id"}, CopyModeDeps{})
	delivery := port.PluginDelivery{EndpointID: "local", Message: &apipb.PluginMessage{
		Source:      &apipb.PluginAddress{DaemonId: "daemon-a", PluginId: "agents"},
		Destination: &apipb.PluginAddress{TuiInstanceId: "tui-routing-id"},
		Body:        &apipb.PluginMessage_Init{Init: &apipb.PluginUiInit{}},
	}}
	if err := runtime.Post(PluginDeliveryMsg{Delivery: delivery}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(service.messages) != 1 || service.messages[0].GetInit().GetContext().GetTuiInstanceId() != "tui-routing-id" {
		t.Fatalf("plugin init did not use routing identity: %v", service.messages)
	}
	if runtime.State().RuntimeSurfaceID != root.RuntimeSurfaceID {
		t.Fatal("terminal surface identity changed")
	}
}

// Drive request/reply effects synchronously while deliveries are supplied by the test.
type pluginWiringRunner struct{}

func (pluginWiringRunner) Run(ctx context.Context, effect Effect, post func(Msg)) {
	if f, ok := effect.(FuncEffect); ok && f.Run != nil {
		if msg := f.Run(ctx); msg != nil {
			post(msg)
		}
	}
}
func (pluginWiringRunner) Cancel(CancelToken) {}
