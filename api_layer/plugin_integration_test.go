package apilayer

import (
	"context"
	"testing"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

func pluginTestCall(t *testing.T, app *clientruntime.ApplicationSession, command *apipb.PluginCommand) *apipb.PluginResult {
	t.Helper()
	value, err := app.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_Plugin{Plugin: command}})
	if err != nil {
		t.Fatal(err)
	}
	if value.GetPlugin() == nil {
		t.Fatalf("missing plugin result: %v", value)
	}
	return value.GetPlugin()
}
func pluginTestRegister(t *testing.T, app *clientruntime.ApplicationSession, tui, instance string, service bool) *apipb.PluginRegistration {
	t.Helper()
	result := pluginTestCall(t, app, &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: "org.anytty.agents", TuiInstanceId: tui, PluginInstanceId: instance}, DaemonService: service, Topics: []string{"plugin.org.anytty.agents.changed"}}}})
	if result.GetRegistration() == nil {
		t.Fatal(result)
	}
	return result.GetRegistration()
}
func pluginTestSend(t *testing.T, app *clientruntime.ApplicationSession, reg *apipb.PluginRegistration, m *apipb.PluginMessage) *apipb.PluginResult {
	t.Helper()
	return pluginTestCall(t, app, &apipb.PluginCommand{Command: &apipb.PluginCommand_Send{Send: &apipb.PluginSendRequest{SourceLease: reg.SourceLease, Message: m}}})
}
func pluginTestReceive(t *testing.T, app *clientruntime.ApplicationSession, reg *apipb.PluginRegistration) *apipb.PluginBatch {
	t.Helper()
	result := pluginTestCall(t, app, &apipb.PluginCommand{Command: &apipb.PluginCommand_Receive{Receive: &apipb.PluginReceiveRequest{SourceLease: reg.SourceLease, MaxMessages: 32}}})
	if result.GetBatch() == nil {
		t.Fatal(result)
	}
	return result.GetBatch()
}
func pluginTestPayload(destination *apipb.PluginAddress) *apipb.PluginMessage {
	return &apipb.PluginMessage{Destination: destination, Body: &apipb.PluginMessage_Payload{Payload: &apipb.PluginPayload{Schema: "test", Version: 1, Data: []byte("hello")}}}
}
func pluginExpectCode(t *testing.T, result *apipb.PluginResult, want string) {
	t.Helper()
	if result.GetError().GetCode() != want {
		t.Fatalf("got %v, want %s", result, want)
	}
}

func TestPluginProtobufRoutingSelfBroadcastIsolationAndLease(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithPluginRuntime("daemon-one", t.TempDir()))
	defer server.Shutdown(context.Background())
	a, _, closeA := newProtoTransportClient(t, server, nil, 1)
	defer closeA()
	b, _, closeB := newProtoTransportClient(t, server, nil, 2)
	defer closeB()
	regA := pluginTestRegister(t, a, "tui-a", "host", false)
	regB := pluginTestRegister(t, b, "tui-b", "host", false)
	self := pluginTestPayload(regA.Address)
	self.RequestId = "self-1"
	self.IdempotencyKey = "self-1"
	if got := pluginTestSend(t, a, regA, self); got.GetAck().GetDelivered() != 1 {
		t.Fatal(got)
	}
	// A repeated idempotent request never creates a second delivery.
	if got := pluginTestSend(t, a, regA, self); got.GetAck().GetDelivered() != 1 {
		t.Fatal(got)
	}
	batch := pluginTestReceive(t, a, regA)
	if len(batch.Messages) != 1 || batch.Messages[0].GetSource().GetRegistrationEpoch() != regA.Address.RegistrationEpoch || server.PluginRouteCount() != 1 {
		t.Fatalf("self route bypassed daemon: %v", batch)
	}
	if len(pluginTestReceive(t, b, regB).Messages) != 0 {
		t.Fatal("self message leaked to TUI B")
	}
	changed := proto.Clone(self).(*apipb.PluginMessage)
	changed.GetPayload().Data = []byte("changed")
	pluginExpectCode(t, pluginTestSend(t, a, regA, changed), "CONFLICT")
	pluginExpectCode(t, pluginTestSend(t, a, regA, pluginTestPayload(regB.Address)), "PERMISSION_DENIED")
	forged := pluginTestPayload(regA.Address)
	forged.Source = regB.Address
	pluginExpectCode(t, pluginTestSend(t, a, regA, forged), "PERMISSION_DENIED")
	pluginExpectCode(t, pluginTestSend(t, b, regA, pluginTestPayload(regA.Address)), "STALE_CONTEXT")
	stale := clonePluginAddress(regA.Address)
	stale.RegistrationEpoch++
	pluginExpectCode(t, pluginTestSend(t, a, regA, pluginTestPayload(stale)), "TARGET_OFFLINE")
	broadcast := pluginTestPayload(nil)
	broadcast.Topic = "plugin.org.anytty.agents.changed"
	if got := pluginTestSend(t, a, regA, broadcast); got.GetAck().GetDelivered() != 2 {
		t.Fatal(got)
	}
	if len(pluginTestReceive(t, a, regA).Messages) != 1 || len(pluginTestReceive(t, b, regB).Messages) != 1 {
		t.Fatal("broadcast missing")
	}
	broadcast.Body = &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Operation: &apipb.PluginUiOperation_Notification{Notification: &apipb.PluginUiNotification{Title: "bad"}}}}
	pluginExpectCode(t, pluginTestSend(t, a, regA, broadcast), "PERMISSION_DENIED")
}
func clonePluginAddress(a *apipb.PluginAddress) *apipb.PluginAddress {
	return proto.Clone(a).(*apipb.PluginAddress)
}

func TestPluginInteractionContextAndReplyCannotBeForged(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithPluginRuntime("daemon-one", t.TempDir()))
	defer server.Shutdown(context.Background())
	app, _, closeApp := newProtoTransportClient(t, server, nil, 1)
	defer closeApp()
	host := pluginTestRegister(t, app, "tui-a", "host", false)
	ui := pluginTestRegister(t, app, "tui-a", "ui", false)
	if len(ui.Peers) != 1 || !proto.Equal(ui.Peers[0], host.Address) {
		t.Fatalf("peer discovery leaked or omitted: %v", ui.Peers)
	}
	interaction := &apipb.PluginMessage{RequestId: "click", Destination: ui.Address, Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{ActionId: "agents.open", Context: &apipb.PluginTargetContext{ContextId: "9a93cf223f774c9ca69b154a6f1ac6bc", TuiInstanceId: "tui-a", PaneId: "pane-a", BindingRevision: 9}}}}
	if got := pluginTestSend(t, app, host, interaction); got.GetError() != nil {
		t.Fatal(got)
	}
	batch := pluginTestReceive(t, app, ui)
	context := batch.Messages[0].GetInteraction().GetContext()
	if !proto.Equal(context, interaction.GetInteraction().GetContext()) {
		t.Fatal("daemon rewrote host-indexed interaction context; returning bind cannot find its target")
	}
	duplicate := proto.Clone(interaction).(*apipb.PluginMessage)
	duplicate.RequestId = "click-duplicate"
	duplicate.GetInteraction().Context.PaneId = "evil-other-pane"
	pluginExpectCode(t, pluginTestSend(t, app, host, duplicate), "CONFLICT")
	missing := proto.Clone(interaction).(*apipb.PluginMessage)
	missing.RequestId = "click-missing"
	missing.GetInteraction().Context.ContextId = ""
	pluginExpectCode(t, pluginTestSend(t, app, host, missing), "INVALID_REQUEST")
	impostor := pluginTestRegister(t, app, "tui-a", "impostor", false)
	bind := &apipb.PluginMessage{RequestId: "bind", Destination: host.Address, Body: &apipb.PluginMessage_Operation{Operation: &apipb.PluginUiOperation{Context: context, Operation: &apipb.PluginUiOperation_Bind{Bind: &apipb.PluginPaneBind{Terminal: &apipb.PluginTerminalRef{DaemonId: "daemon-one", TerminalId: "term"}}}}}}
	pluginExpectCode(t, pluginTestSend(t, app, impostor, bind), "STALE_CONTEXT")
	forged := proto.Clone(bind).(*apipb.PluginMessage)
	forged.GetOperation().Context.PaneId = "other"
	pluginExpectCode(t, pluginTestSend(t, app, ui, forged), "STALE_CONTEXT")
	if got := pluginTestSend(t, app, ui, bind); got.GetError() != nil {
		t.Fatal(got)
	}
	returned := pluginTestReceive(t, app, host)
	if len(returned.Messages) != 1 || !proto.Equal(returned.Messages[0].GetOperation().GetContext(), interaction.GetInteraction().GetContext()) {
		t.Fatal("bind did not return with the host's originally saved context")
	}
	reply := &apipb.PluginMessage{Destination: ui.Address, Body: &apipb.PluginMessage_Reply{Reply: &apipb.PluginReply{RequestId: "bind"}}}
	if got := pluginTestSend(t, app, host, reply); got.GetError() != nil {
		t.Fatal(got)
	}
	pluginExpectCode(t, pluginTestSend(t, app, host, reply), "PERMISSION_DENIED")
}

func TestPluginScopedSessionsDeniedAndDisconnectFencesRegistration(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithPluginRuntime("daemon-one", t.TempDir()), corev2.WithClientAccessService(&protoGrantAccessService{grants: map[string]time.Time{"share": expires, "events": expires}}))
	defer server.Shutdown(context.Background())
	for _, scope := range []corev2.TransportScope{{GrantID: "share", GrantExpiresAt: expires, PrincipalID: "p", TerminalID: "term"}, {GrantID: "events", GrantExpiresAt: expires, PrincipalID: "p", MachineEventsOnly: true}} {
		app, _, closeApp := newProtoTransportClient(t, server, &scope, 1)
		_, err := app.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_Plugin{Plugin: &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: &apipb.PluginRegisterRequest{Address: &apipb.PluginAddress{PluginId: "org.anytty.agents", PluginInstanceId: "bad"}}}}}})
		if clientruntime.CodeOf(err) != clientruntime.ErrorAuthorization {
			t.Fatalf("scope admitted: %v", err)
		}
		closeApp()
	}
	a, _, closeA := newProtoTransportClient(t, server, nil, 2)
	old := pluginTestRegister(t, a, "tui-old", "host", false)
	closeA()
	b, _, closeB := newProtoTransportClient(t, server, nil, 3)
	defer closeB()
	fresh := pluginTestRegister(t, b, "tui-new", "host", false)
	pluginExpectCode(t, pluginTestSend(t, b, old, pluginTestPayload(fresh.Address)), "STALE_CONTEXT")
	pluginExpectCode(t, pluginTestSend(t, b, fresh, pluginTestPayload(old.Address)), "TARGET_OFFLINE")
}

func TestPluginTwoDaemonDomainsAndAuthenticatedHandoff(t *testing.T) {
	first := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithPluginRuntime("first", t.TempDir()))
	defer first.Shutdown(context.Background())
	second := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithPluginRuntime("second", t.TempDir()))
	defer second.Shutdown(context.Background())
	a, _, closeA := newProtoTransportClient(t, first, nil, 1)
	defer closeA()
	b, _, closeB := newProtoTransportClient(t, second, nil, 1)
	defer closeB()
	ra := pluginTestRegister(t, a, "same-tui", "host", false)
	rb := pluginTestRegister(t, b, "same-tui", "host", false)
	pluginExpectCode(t, pluginTestSend(t, a, ra, pluginTestPayload(rb.Address)), "TARGET_OFFLINE")
	alias, _, closeAlias := newProtoTransportClient(t, first, nil, 2)
	defer closeAlias()
	request := &apipb.PluginRegisterRequest{Address: clonePluginAddress(ra.Address)}
	call := func() *apipb.PluginResult {
		return pluginTestCall(t, alias, &apipb.PluginCommand{Command: &apipb.PluginCommand_Register{Register: request}})
	}
	pluginExpectCode(t, call(), "CONFLICT")
	request.PreviousLease = ra.SourceLease
	newReg := call().GetRegistration()
	if newReg == nil || newReg.Address.RegistrationEpoch == ra.Address.RegistrationEpoch {
		t.Fatalf("handoff failed: %v", newReg)
	}
	pluginExpectCode(t, pluginTestSend(t, a, ra, pluginTestPayload(ra.Address)), "STALE_CONTEXT")
	if got := pluginTestSend(t, alias, newReg, pluginTestPayload(newReg.Address)); got.GetAck().GetDelivered() != 1 {
		t.Fatal(got)
	}
}

func TestPluginRejectedDeliveryDoesNotConsumeHostContextIdentity(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithPluginRuntime("daemon", t.TempDir()))
	defer server.Shutdown(context.Background())
	app, _, closeApp := newProtoTransportClient(t, server, nil, 1)
	defer closeApp()
	host := pluginTestRegister(t, app, "tui", "host", false)
	ui := pluginTestRegister(t, app, "tui", "ui", false)
	// Fill the bounded mailbox until its public delivery contract rejects a send.
	for i := 0; i < 128; i++ {
		result := pluginTestSend(t, app, host, pluginTestPayload(ui.Address))
		if result.GetError() != nil {
			pluginExpectCode(t, result, "RESOURCE_EXHAUSTED")
			break
		}
		if i == 127 {
			t.Fatal("mailbox was not bounded")
		}
	}
	click := &apipb.PluginMessage{RequestId: "retry-click", Destination: ui.Address, Body: &apipb.PluginMessage_Interaction{Interaction: &apipb.PluginUiInteraction{Context: &apipb.PluginTargetContext{ContextId: "e9dce127da9c4d35a9913f6316e81fb9", TuiInstanceId: "tui", PaneId: "pane"}}}}
	pluginExpectCode(t, pluginTestSend(t, app, host, click), "RESOURCE_EXHAUSTED")
	_ = pluginTestReceive(t, app, ui)
	if result := pluginTestSend(t, app, host, click); result.GetError() != nil {
		t.Fatalf("rejected delivery consumed host context identity: %v", result)
	}
}
