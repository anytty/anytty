package app

import (
	"context"
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"google.golang.org/protobuf/proto"
)

func TestPluginRuntimeContextSurvivesDaemonRoundtrip(t *testing.T) {
	root, deps, service := pluginFixture(t)
	runtime := NewInteractiveRuntime(root, NewFakeTerminalHost(16), NewSyncEffectRunner(), LiveDeps{Plugins: service, PluginTUIInstanceID: "tui-a", Terminal: deps.Live.Terminal}, CopyModeDeps{})
	runtime.queue = nil
	reduce := runtime.reduce
	runtime.reduce = func(root state.Root, msg Msg) (state.Root, []Effect) {
		next, effects := reduce(root, msg)
		for i, e := range effects {
			if f, ok := e.(FuncEffect); ok && strings.HasPrefix(f.SerialKey, "plugin:") {
				f.ForceSyncInTests = true
				effects[i] = f
			}
		}
		return next, effects
	}
	for _, event := range []input.InputEvent{{Kind: input.EventKindKey, Key: input.KeyChar, Char: "g", Ctrl: true}, {Kind: input.EventKindKey, Key: input.KeyChar, Char: "a"}, {Kind: input.EventKindKey, Key: input.KeyEnter}} {
		if err := runtime.Post(InputMsg{Event: event}); err != nil {
			t.Fatal(err)
		}
		if err := runtime.Drain(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	var sent *apipb.PluginMessage
	for _, message := range service.messages {
		if message.GetInteraction() != nil {
			sent = message
		}
	}
	if sent == nil {
		t.Fatalf("no routed interaction: plugins=%+v mode=%v messages=%v", runtime.State().Plugins, runtime.State().Shell.InteractionMode, service.messages)
	}
	wire, err := proto.Marshal(sent.GetInteraction().Context)
	if err != nil {
		t.Fatal(err)
	}
	returned := &apipb.PluginTargetContext{}
	if err := proto.Unmarshal(wire, returned); err != nil {
		t.Fatal(err)
	}
	delivery := port.PluginDelivery{EndpointID: "remote", Message: &apipb.PluginMessage{RequestId: "bind", Source: &apipb.PluginAddress{PluginId: "agents", DaemonId: "daemon-b"}, Destination: &apipb.PluginAddress{TuiInstanceId: "tui-a"}, Body: &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Context: returned, Operation: &apipb.PluginUiOperation_Bind{Bind: &apipb.PluginPaneBind{Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-b", TerminalId: "new"}}}}}}}
	if err := runtime.Post(PluginDeliveryMsg{delivery}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.State().Plugins.Error != "" {
		t.Fatalf("routed bind rejected: %s, contexts=%v returned=%v", runtime.State().Plugins.Error, runtime.State().Plugins.Contexts, returned)
	}
	binding, ok := runtime.State().TerminalViews.PaneBinding(state.DefaultPaneID)
	if !ok || !binding.Attached || binding.TerminalID != "new" || binding.EndpointID != "remote" {
		t.Fatalf("routed operation did not commit the pinned attachment: %+v", binding)
	}

}
