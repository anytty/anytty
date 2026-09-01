package binding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/bindingpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/proto/wirepb"
	"github.com/anytty/anytty/shared/transport/memory"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestEngineUsesProtoBytesOpaqueHandlesAndOrderedEvents(t *testing.T) {
	session := newBindingSession()
	host := &bindingHost{session: session}
	engine, err := NewEngineWithEventCapacity(host, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	openRequest := &bindingpb.OpenSessionRequest{
		RequestId: "open-1", EndpointId: "studio", RouteOverride: "cloud", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	}
	openRequest.ProtoReflect().SetUnknown(protowire.AppendVarint(protowire.AppendTag(nil, 99, protowire.VarintType), 7))
	openPayload, err := proto.Marshal(openRequest)
	if err != nil {
		t.Fatal(err)
	}
	openOperation, err := engine.OpenSession(openPayload)
	if err != nil {
		t.Fatal(err)
	}
	openEvent := nextBindingEvent(t, engine)
	if openEvent.GetSequence() != 1 || openEvent.GetAbiVersion() != ABIVersion || openEvent.GetOpenSession().GetOperationHandle() != openOperation {
		t.Fatalf("open event = %#v", openEvent)
	}
	if len(host.request.ProtoReflect().GetUnknown()) == 0 {
		t.Fatal("open request unknown fields were discarded")
	}
	sessionHandle := openEvent.GetOpenSession().GetSessionHandle()
	if sessionHandle == 0 || openEvent.GetOpenSession().GetSession().GetGeneration() != 3 {
		t.Fatalf("open result = %#v", openEvent.GetOpenSession())
	}
	if err := engine.Release(openOperation); err != nil {
		t.Fatal(err)
	}

	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}}
	command.ProtoReflect().SetUnknown(protowire.AppendVarint(protowire.AppendTag(nil, 999, protowire.VarintType), 42))
	commandPayload, err := proto.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	executeOperation, err := engine.Execute(sessionHandle, commandPayload)
	if err != nil {
		t.Fatal(err)
	}
	executeEvent := nextBindingEvent(t, engine)
	if executeEvent.GetSequence() != 2 || executeEvent.GetExecute().GetOperationHandle() != executeOperation || executeEvent.GetExecute().GetResult().GetTerminalList() == nil {
		t.Fatalf("execute event = %#v", executeEvent)
	}
	if len(session.command.ProtoReflect().GetUnknown()) == 0 {
		t.Fatal("CommandEnvelope unknown fields were discarded")
	}
	if err := engine.Release(executeOperation); err != nil {
		t.Fatal(err)
	}

	session.events <- &apipb.EventEnvelope{EventId: "event-1", Event: &apipb.EventEnvelope_StorageChanged{StorageChanged: &apipb.StorageChangedEvent{}}}
	applicationEvent := nextBindingEvent(t, engine)
	if applicationEvent.GetSequence() != 3 || applicationEvent.GetApplication().GetSessionHandle() != sessionHandle || applicationEvent.GetApplication().GetEvent().GetEventId() != "event-1" {
		t.Fatalf("application event = %#v", applicationEvent)
	}
	if err := engine.CloseSession(sessionHandle); err != nil {
		t.Fatal(err)
	}
	closedEvent := nextBindingEvent(t, engine)
	if closedEvent.GetSequence() != 4 || closedEvent.GetSessionClosed().GetSessionHandle() != sessionHandle {
		t.Fatalf("closed event = %#v", closedEvent)
	}
	if err := engine.Release(sessionHandle); err != nil {
		t.Fatal(err)
	}
}

func TestEngineOpenSessionReturnsReadyConnectionSnapshot(t *testing.T) {
	session := newBindingSession()
	session.connection = &clientruntime.ConnectionSnapshot{
		RouteID: "direct", RouteKind: endpoint.RouteDirectWebRTCTCP, ObservedPath: string(endpoint.PathDirect),
		SampledAt: time.Unix(1_800_000_000, 0).UTC(), RoundTrip: 42 * time.Millisecond,
		LocalCandidateType: "relay", RemoteCandidateType: "relay", LocalAddress: "192.0.2.10", RemoteAddress: "2001:db8::20", LocalPort: 41000, RemotePort: 41121,
		PairID: "pair-selected", LocalRelatedAddress: "10.0.0.2", LocalRelatedPort: 40000,
		LocalProtocol: "udp", RemoteProtocol: "udp", RelayTransport: "tcp", Connected: true,
	}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	payload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{RequestId: "snapshot", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE})
	if _, err := engine.OpenSession(payload); err != nil {
		t.Fatal(err)
	}
	event := nextBindingEvent(t, engine)
	snapshot := event.GetOpenSession().GetConnection()
	if snapshot.GetRouteKind() != bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_DIRECT || snapshot.GetObservedPath() != bindingpb.ConnectionObservedPath_CONNECTION_OBSERVED_PATH_DIRECT ||
		snapshot.GetRelayTransport() != bindingpb.ConnectionTransport_CONNECTION_TRANSPORT_TCP || snapshot.GetRoundTripNanos() != int64(42*time.Millisecond) ||
		snapshot.GetLocalIp() != "192.0.2.10" || snapshot.GetRemoteIp() != "2001:db8::20" || snapshot.GetLocalPort() != 41000 || snapshot.GetRemotePort() != 41121 ||
		snapshot.GetCandidatePairId() != "pair-selected" || snapshot.GetLocalRelatedIp() != "10.0.0.2" || snapshot.GetLocalRelatedPort() != 40000 {
		t.Fatalf("connection snapshot = %#v", snapshot)
	}
}

func TestEngineForwardsGoOwnedEndpointLifecycleBeforeOpenResult(t *testing.T) {
	session := newBindingSession()
	started := make(chan struct{})
	release := make(chan struct{})
	host := &lifecycleBindingHost{
		bindingHost: &bindingHost{session: session, open: func(context.Context, *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
			close(started)
			<-release
			return session, nil
		}},
		events: make(chan clientruntime.EndpointEvent, 4),
	}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	payload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{RequestId: "lifecycle", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE})
	operation, err := engine.OpenSession(payload)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	host.events <- clientruntime.EndpointEvent{EndpointID: "studio", Stamp: clientruntime.EndpointSessionStamp{EndpointID: "studio", Generation: 3}, Phase: clientruntime.EndpointPhasePlanning}
	host.events <- clientruntime.EndpointEvent{
		EndpointID:         "studio",
		Stamp:              session.Stamp(),
		Phase:              clientruntime.EndpointPhaseConnecting,
		AttemptedRouteKind: endpoint.RouteManagedWebRTC,
		ConnectionStage:    clientruntime.EndpointStageCloudP2PAttempt,
	}
	host.events <- clientruntime.EndpointEvent{EndpointID: "studio", Stamp: session.Stamp(), Phase: clientruntime.EndpointPhaseReady, ObservedPath: string(endpoint.PathSingleRelay), RouteSelectionReason: "only_viable"}

	for _, want := range []bindingpb.EndpointConnectionPhase{
		bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_PLANNING,
		bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_CONNECTING,
		bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_READY,
	} {
		event := nextBindingEvent(t, engine).GetEndpointConnection()
		if event.GetOperationHandle() != operation || event.GetRequestId() != "lifecycle" || event.GetEndpointId() != "studio" || event.GetPhase() != want {
			t.Fatalf("endpoint lifecycle event = %#v, want phase %v", event, want)
		}
		if want == bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_CONNECTING &&
			(event.GetAttemptedRouteKind() != bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_CLOUD || event.GetConnectionStage() != string(clientruntime.EndpointStageCloudP2PAttempt)) {
			t.Fatalf("connecting lifecycle event = %#v", event)
		}
		if want == bindingpb.EndpointConnectionPhase_ENDPOINT_CONNECTION_PHASE_READY &&
			(event.GetSession().GetGeneration() != 3 || event.GetObservedPath() != bindingpb.ConnectionObservedPath_CONNECTION_OBSERVED_PATH_SINGLE_RELAY || event.GetRouteSelectionReason() != "only_viable") {
			t.Fatalf("ready lifecycle event = %#v", event)
		}
	}
	close(release)
	if opened := nextBindingEvent(t, engine).GetOpenSession(); opened.GetOperationHandle() != operation || opened.GetSessionHandle() == 0 {
		t.Fatalf("open result = %#v", opened)
	}
}

func TestEngineOpenDoesNotWaitForLifecycleWatcherToClose(t *testing.T) {
	host := &stubbornLifecycleBindingHost{
		bindingHost: &bindingHost{session: newBindingSession()},
		events:      make(chan clientruntime.EndpointEvent),
	}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	payload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "stubborn", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if _, err := engine.OpenSession(payload); err != nil {
		t.Fatal(err)
	}
	if opened := nextBindingEvent(t, engine).GetOpenSession(); opened.GetSessionHandle() == 0 {
		t.Fatalf("open result = %#v", opened)
	}
}

func TestEngineCancelledOpenOnlyReleasesItsLateConsumerLease(t *testing.T) {
	session := newBindingSession()
	started := make(chan struct{})
	release := make(chan struct{})
	host := &bindingHost{
		session: session,
		open: func(context.Context, *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
			close(started)
			<-release
			return session, nil
		},
	}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	payload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "cancel-late", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	operation, err := engine.OpenSession(payload)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := engine.Cancel(operation); err != nil {
		t.Fatal(err)
	}
	close(release)
	result := nextBindingEvent(t, engine).GetOpenSession()
	if result.GetSessionHandle() != 0 || result.GetError() == nil {
		t.Fatalf("cancelled late open result = %#v", result)
	}
	if host.invalidated != (clientruntime.EndpointSessionStamp{}) {
		t.Fatalf("ordinary open cancellation invalidated shared generation %#v", host.invalidated)
	}
}

func TestBindingRouteKindIncludesManagedCloud(t *testing.T) {
	for kind, want := range map[endpoint.RouteKind]bindingpb.ConnectionRouteKind{
		endpoint.RouteLocalUnix:       bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_LOCAL,
		endpoint.RouteDirectWebRTCTCP: bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_DIRECT,
		endpoint.RouteSSHWebRTCTCP:    bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_SSH,
		endpoint.RouteManagedWebRTC:   bindingpb.ConnectionRouteKind_CONNECTION_ROUTE_KIND_CLOUD,
	} {
		if got := bindingRouteKind(kind); got != want {
			t.Fatalf("bindingRouteKind(%q) = %v, want %v", kind, got, want)
		}
	}
}

func TestEngineConnectionSnapshotCommandResamplesCurrentSession(t *testing.T) {
	session := newBindingSession()
	session.connection = &clientruntime.ConnectionSnapshot{RouteID: "direct", RouteKind: endpoint.RouteDirectWebRTCTCP, SampledAt: time.Unix(1_800_000_000, 0).UTC(), BytesReceived: 10, Connected: true}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	payload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{RequestId: "open-resample", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE})
	openOperation, err := engine.OpenSession(payload)
	if err != nil {
		t.Fatal(err)
	}
	opened := nextBindingEvent(t, engine).GetOpenSession()
	if err := engine.Release(openOperation); err != nil {
		t.Fatal(err)
	}
	session.connection = &clientruntime.ConnectionSnapshot{RouteID: "direct", RouteKind: endpoint.RouteDirectWebRTCTCP, SampledAt: time.Unix(1_800_000_005, 0).UTC(), BytesReceived: 99, Connected: true}
	command, _ := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_ConnectionSnapshotGet{ConnectionSnapshotGet: &bindingpb.ConnectionSnapshotGetRequest{
		RequestId: "resample", SessionHandle: opened.GetSessionHandle(),
	}}})
	operation, err := engine.EngineCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	result := nextBindingEvent(t, engine).GetConnectionSnapshotGet()
	if result.GetOperationHandle() != operation || result.GetConnection().GetBytesReceived() != 99 || result.GetConnection().GetSampledAtUnixNano() != time.Unix(1_800_000_005, 0).UnixNano() {
		t.Fatalf("resampled connection = %#v", result)
	}
}

func TestEngineSessionInvalidateUsesExactSessionStamp(t *testing.T) {
	session := newBindingSession()
	host := &bindingHost{session: session}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	command, err := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_SessionInvalidate{SessionInvalidate: &bindingpb.SessionInvalidateRequest{
		RequestId: "network-change", SessionHandle: sessionHandle,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := engine.EngineCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	result := nextBindingEvent(t, engine).GetSessionInvalidate()
	if result.GetOperationHandle() != operation || result.GetSessionHandle() != sessionHandle || result.GetError() != nil {
		t.Fatalf("session invalidate result = %#v", result)
	}
	if host.invalidated != session.Stamp() {
		t.Fatalf("invalidated stamp = %#v, want %#v", host.invalidated, session.Stamp())
	}
}

func TestEngineEndpointDisconnectDoesNotRequireSessionHandle(t *testing.T) {
	host := &bindingHost{session: newBindingSession()}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	command, err := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_EndpointDisconnect{EndpointDisconnect: &bindingpb.EndpointDisconnectRequest{
		RequestId: "user-disconnect", EndpointId: "studio",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := engine.EngineCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	result := nextBindingEvent(t, engine).GetEndpointDisconnect()
	if result.GetOperationHandle() != operation || result.GetEndpointId() != "studio" || result.GetError() != nil {
		t.Fatalf("endpoint disconnect result = %#v", result)
	}
	if host.disconnected != "studio" {
		t.Fatalf("disconnected endpoint = %q", host.disconnected)
	}
}

func TestEngineEndpointCloudPresenceDoesNotRequireSessionHandle(t *testing.T) {
	host := &bindingHost{cloudPresence: true}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	command, err := proto.Marshal(&bindingpb.EngineCommand{Command: &bindingpb.EngineCommand_EndpointCloudPresenceGet{EndpointCloudPresenceGet: &bindingpb.EndpointCloudPresenceGetRequest{
		RequestId: "presence", EndpointId: "studio",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := engine.EngineCommand(command)
	if err != nil {
		t.Fatal(err)
	}
	result := nextBindingEvent(t, engine).GetEndpointCloudPresenceGet()
	if result.GetOperationHandle() != operation || result.GetEndpointId() != "studio" || !result.GetOnline() || result.GetError() != nil {
		t.Fatalf("endpoint Cloud presence result = %#v", result)
	}
	if host.cloudPresenceEndpoint != "studio" {
		t.Fatalf("Cloud presence endpoint = %q", host.cloudPresenceEndpoint)
	}
}

func TestEngineCancellationProducesTypedResult(t *testing.T) {
	session := newBindingSession()
	started := make(chan struct{})
	release := make(chan struct{})
	session.execute = func(_ context.Context, _ *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
		close(started)
		<-release
		return terminalListBindingResult(), nil
	}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	payload, err := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := engine.Execute(sessionHandle, payload)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := engine.Release(operation); !errors.Is(err, ErrHandleActive) {
		t.Fatalf("active operation release error = %v", err)
	}
	if err := engine.Cancel(operation); err != nil {
		t.Fatal(err)
	}
	close(release)
	event := nextBindingEvent(t, engine)
	if event.GetExecute().GetOperationHandle() != operation || event.GetExecute().GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED {
		t.Fatalf("cancel event = %#v", event)
	}
	if err := engine.Release(operation); err != nil {
		t.Fatal(err)
	}
}

func TestEngineCancellationDestroysLateFileOpenResource(t *testing.T) {
	session := newBindingSession()
	var closeCount atomic.Int32
	session.closeFunc = func() error {
		session.closeOnce.Do(func() {
			closeCount.Add(1)
			close(session.done)
		})
		return nil
	}
	started := make(chan struct{})
	release := make(chan struct{})
	cleanup := make(chan *apipb.FileTransferCancelCommand, 1)
	session.execute = func(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
		if command.GetFileTransferCancel() != nil {
			cleanup <- proto.Clone(command.GetFileTransferCancel()).(*apipb.FileTransferCancelCommand)
			return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_FileTransferCancel{FileTransferCancel: &apipb.FileTransferCancelResult{Cancelled: true}}}, nil
		}
		close(started)
		<-release
		return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: &apipb.FileTransferOpenResult{Transfer: &apipb.FileTransferHandle{
			Resource: &apipb.ResourceHandle{Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, OpaqueToken: []byte("resource")},
			Resume:   &apipb.FileUploadResumeHandle{OpaqueToken: []byte("resume")},
		}}}}, nil
	}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	payload, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileUploadOpen{FileUploadOpen: &apipb.FileUploadOpenCommand{Path: "/tmp/demo", Size: 8, Overwrite: true}}})
	operation, err := engine.Execute(sessionHandle, payload)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := engine.Cancel(operation); err != nil {
		t.Fatal(err)
	}
	if err := engine.CloseSession(sessionHandle); err != nil {
		t.Fatal(err)
	}
	if closeCount.Load() != 0 {
		t.Fatal("session closed before the cancelled file-open operation reached cleanup")
	}
	close(release)
	event := nextBindingEvent(t, engine)
	if event.GetExecute().GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED {
		t.Fatalf("cancel event = %#v", event)
	}
	select {
	case command := <-cleanup:
		if string(command.GetUploadResume().GetOpaqueToken()) != "resume" || command.GetTransfer() != nil {
			t.Fatalf("cleanup command = %#v", command)
		}
	case <-time.After(time.Second):
		t.Fatal("late file open resource was not destroyed")
	}
	deadline := time.Now().Add(time.Second)
	for closeCount.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if closeCount.Load() != 1 {
		t.Fatalf("session close count=%d want=1", closeCount.Load())
	}
}

func TestCancelledApplicationCleanupCoversHistoryAndEventResources(t *testing.T) {
	tests := []struct {
		name     string
		original *apipb.CommandEnvelope
		result   *apipb.ResultEnvelope
		check    func(*testing.T, *apipb.CommandEnvelope)
	}{
		{
			name: "history",
			original: &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowCommand{
				Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST,
			}}},
			result: &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowResult{
				Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}, Token: "history-token", HistoryGeneration: 7,
			}}},
			check: func(t *testing.T, command *apipb.CommandEnvelope) {
				release := command.GetHistoryRelease()
				if release.GetTerminal().GetTerminalId() != "term-1" || release.GetToken() != "history-token" || release.GetHistoryGeneration() != 7 {
					t.Fatalf("history cleanup command = %#v", command)
				}
			},
		},
		{
			name:     "event",
			original: &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_EventSubscribe{EventSubscribe: &apipb.EventSubscribeCommand{}}},
			result: &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_EventSubscription{EventSubscription: &apipb.EventSubscriptionResult{Subscription: &apipb.ResourceHandle{
				Kind: apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION, OpaqueToken: []byte("event-token"),
			}}}},
			check: func(t *testing.T, command *apipb.CommandEnvelope) {
				resource := command.GetReleaseResource().GetResource()
				if resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION || string(resource.GetOpaqueToken()) != "event-token" {
					t.Fatalf("event cleanup command = %#v", command)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, confirmUpload := cancelledApplicationCleanupCommand(test.original, test.result)
			if command == nil || confirmUpload {
				t.Fatalf("cleanup command=%#v confirmUpload=%v", command, confirmUpload)
			}
			test.check(t, command)
		})
	}
	older := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowCommand{
		Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_OLDER,
	}}}
	historyResult := tests[0].result
	if command, _ := cancelledApplicationCleanupCommand(older, historyResult); command != nil {
		t.Fatalf("older history page reused an existing token but produced cleanup %#v", command)
	}
	if requiresTerminalResponse(older) {
		t.Fatal("older history page must not require a terminal response")
	}
	for _, command := range []*apipb.CommandEnvelope{
		{Command: &apipb.CommandEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowCommand{Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST}}},
		{Command: &apipb.CommandEnvelope_EventSubscribe{EventSubscribe: &apipb.EventSubscribeCommand{}}},
	} {
		if !requiresTerminalResponse(command) {
			t.Fatalf("resource-producing command does not require terminal response: %#v", command)
		}
	}
}

func TestEngineCancellationReportsLateFileOpenCleanupFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		cleanup func(context.Context) error
	}{
		{name: "error", cleanup: func(context.Context) error { return errors.New("cleanup failed") }},
		{name: "timeout", cleanup: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := newBindingSession()
			var closeCount atomic.Int32
			session.closeFunc = func() error {
				session.closeOnce.Do(func() {
					closeCount.Add(1)
					close(session.done)
				})
				return nil
			}
			started := make(chan struct{})
			release := make(chan struct{})
			session.execute = func(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
				if command.GetFileTransferCancel() != nil {
					return nil, test.cleanup(ctx)
				}
				close(started)
				<-release
				return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: &apipb.FileTransferOpenResult{Transfer: &apipb.FileTransferHandle{
					Resource: &apipb.ResourceHandle{Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, OpaqueToken: []byte("resource")},
					Resume:   &apipb.FileUploadResumeHandle{OpaqueToken: []byte("resume")},
				}}}}, nil
			}
			engine, err := NewEngine(&bindingHost{session: session})
			if err != nil {
				t.Fatal(err)
			}
			engine.cleanupTTL = 10 * time.Millisecond
			defer engine.Close()
			sessionHandle := openBindingSession(t, engine)
			payload, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileUploadOpen{FileUploadOpen: &apipb.FileUploadOpenCommand{Path: "/tmp/demo", Size: 8, Overwrite: true}}})
			operation, err := engine.Execute(sessionHandle, payload)
			if err != nil {
				t.Fatal(err)
			}
			<-started
			if err := engine.Cancel(operation); err != nil {
				t.Fatal(err)
			}
			close(release)
			var executeEvent, closedEvent *bindingpb.EventEnvelope
			for range 2 {
				event := nextBindingEvent(t, engine)
				switch event.GetEvent().(type) {
				case *bindingpb.EventEnvelope_Execute:
					executeEvent = event
				case *bindingpb.EventEnvelope_SessionClosed:
					closedEvent = event
				default:
					t.Fatalf("unexpected cleanup failure event: %#v", event)
				}
			}
			if executeEvent == nil || executeEvent.GetExecute().GetError() == nil || executeEvent.GetExecute().GetError().GetCode() == apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED {
				t.Fatalf("cleanup failure was not observable: %#v", executeEvent)
			}
			if closedEvent == nil || closedEvent.GetSessionClosed().GetSessionHandle() != sessionHandle {
				t.Fatalf("invalidated session close event = %#v", closedEvent)
			}
			if closeCount.Load() != 1 {
				t.Fatalf("cleanup failure session close count=%d want=1", closeCount.Load())
			}
		})
	}
}

func TestEngineCleansDelayedRealProtocolFileOpenBeforeClosingSession(t *testing.T) {
	clientTransport, serverTransport := memory.NewPair()
	defer serverTransport.Close()
	protocolClient := internalprotocol.NewClient(clientTransport)
	helloDone := make(chan error, 1)
	go func() {
		helloDone <- protocolClient.Hello(context.Background(), internalprotocol.Hello{Version: wire.Version, Client: "binding-test"})
	}()
	if err := bindingExpectHello(serverTransport); err != nil {
		t.Fatal(err)
	}
	if err := <-helloDone; err != nil {
		t.Fatal(err)
	}
	session := &protocolBindingSession{client: protocolClient, stamp: clientruntime.EndpointSessionStamp{EndpointID: endpoint.EndpointID("studio"), RouteID: endpoint.RouteID("memory"), Generation: 1}}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)

	openSeen := make(chan struct{})
	releaseOpen := make(chan struct{})
	cleanupSeen := make(chan struct{})
	serverDone := make(chan error, 1)
	go func() {
		openRequest, openCommand, err := bindingReceiveApplicationRequest(serverTransport)
		if err != nil {
			serverDone <- err
			return
		}
		close(openSeen)
		<-releaseOpen
		openResult := &apipb.ResultEnvelope{RequestId: openCommand.GetContext().GetRequestId(), OriginSession: openCommand.GetContext().GetSession(), Result: &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: &apipb.FileTransferOpenResult{Transfer: &apipb.FileTransferHandle{
			Resource: &apipb.ResourceHandle{Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, OpaqueToken: []byte("resource"), Session: openCommand.GetContext().GetSession(), Generation: 1},
			Resume:   &apipb.FileUploadResumeHandle{OpaqueToken: []byte("resume")},
		}}}}
		if err := bindingSendApplicationResult(serverTransport, openRequest.ID, openResult); err != nil {
			serverDone <- err
			return
		}
		cleanupRequest, cleanupCommand, err := bindingReceiveApplicationRequest(serverTransport)
		if err != nil {
			serverDone <- err
			return
		}
		if string(cleanupCommand.GetFileTransferCancel().GetUploadResume().GetOpaqueToken()) != "resume" {
			serverDone <- fmt.Errorf("cleanup command = %#v", cleanupCommand)
			return
		}
		close(cleanupSeen)
		serverDone <- bindingSendApplicationResult(serverTransport, cleanupRequest.ID, &apipb.ResultEnvelope{RequestId: cleanupCommand.GetContext().GetRequestId(), OriginSession: cleanupCommand.GetContext().GetSession(), Result: &apipb.ResultEnvelope_FileTransferCancel{FileTransferCancel: &apipb.FileTransferCancelResult{Cancelled: true}}})
	}()

	payload, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileUploadOpen{FileUploadOpen: &apipb.FileUploadOpenCommand{Path: "/tmp/demo", Size: 8, Overwrite: true}}})
	operation, err := engine.Execute(sessionHandle, payload)
	if err != nil {
		t.Fatal(err)
	}
	<-openSeen
	if err := engine.Cancel(operation); err != nil {
		t.Fatal(err)
	}
	if err := engine.CloseSession(sessionHandle); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.Done():
		t.Fatal("protocol session closed before delayed file resource cleanup")
	default:
	}
	close(releaseOpen)
	select {
	case <-cleanupSeen:
	case <-time.After(time.Second):
		t.Fatal("delayed real protocol file resource was not destroyed")
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestEngineCloseSessionIsConcurrentAndIdempotent(t *testing.T) {
	engine, err := NewEngine(&bindingHost{session: newBindingSession()})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	handle := openBindingSession(t, engine)
	errorsCh := make(chan error, 8)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errorsCh <- engine.CloseSession(handle)
		}()
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent close error = %v", err)
		}
	}
	if event := nextBindingEvent(t, engine); event.GetSessionClosed().GetSessionHandle() != handle {
		t.Fatalf("session closed event = %#v", event)
	}
}

func TestAwaitSessionTerminalErrorDoesNotCloseBeforeReasonIsPublished(t *testing.T) {
	session := newBindingSession()
	waitStarted := make(chan struct{})
	session.doneFunc = func() <-chan struct{} {
		close(waitStarted)
		return session.done
	}
	var closeCalls atomic.Int32
	session.closeFunc = func() error {
		closeCalls.Add(1)
		return nil
	}
	engineDone := make(chan struct{})
	terminalErr := &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: "transport interrupted", Attempted: true, Retryable: true}
	result := make(chan error, 1)
	go func() {
		err, observed := awaitSessionTerminalError(engineDone, session)
		if !observed {
			result <- errors.New("session terminal reason was not observed")
			return
		}
		result <- err
	}()
	<-waitStarted
	if closeCalls.Load() != 0 {
		t.Fatal("session was closed before its terminal reason was published")
	}
	session.err = terminalErr
	session.closeOnce.Do(func() { close(session.done) })
	if got := <-result; got != terminalErr {
		t.Fatalf("terminal error = %#v, want %#v", got, terminalErr)
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("session close calls = %d, want 1", closeCalls.Load())
	}
}

func TestEnginePublishesDeferredSessionCloseFailure(t *testing.T) {
	session := newBindingSession()
	started := make(chan struct{})
	release := make(chan struct{})
	session.execute = func(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
		close(started)
		<-release
		return terminalListBindingResult(), nil
	}
	session.closeFunc = func() error { return errors.New("close failed") }
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	payload, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	operation, err := engine.Execute(sessionHandle, payload)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := engine.CloseSession(sessionHandle); err != nil {
		t.Fatal(err)
	}
	close(release)
	var closed *bindingpb.SessionClosedEvent
	for range 2 {
		event := nextBindingEvent(t, engine)
		if event.GetSessionClosed() != nil {
			closed = event.GetSessionClosed()
		}
	}
	if closed == nil || closed.GetError() == nil {
		t.Fatalf("deferred close failure was not published: %#v", closed)
	}
	if err := engine.Release(operation); err != nil {
		t.Fatal(err)
	}
}

func TestEngineCloseClaimsSessionOnceWhileOperationCompletes(t *testing.T) {
	session := newBindingSession()
	started := make(chan struct{})
	releaseOperation := make(chan struct{})
	releaseClose := make(chan struct{})
	var closeCount atomic.Int32
	session.execute = func(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
		close(started)
		<-releaseOperation
		return terminalListBindingResult(), nil
	}
	session.closeFunc = func() error {
		closeCount.Add(1)
		<-releaseClose
		return nil
	}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	sessionHandle := openBindingSession(t, engine)
	payload, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if _, err := engine.Execute(sessionHandle, payload); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := engine.CloseSession(sessionHandle); err != nil {
		t.Fatal(err)
	}
	closed := make(chan error, 1)
	go func() { closed <- engine.Close() }()
	deadline := time.Now().Add(time.Second)
	for closeCount.Load() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(releaseOperation)
	close(releaseClose)
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if closeCount.Load() != 1 {
		t.Fatalf("session close count=%d want=1", closeCount.Load())
	}
}

func TestEngineClosingSessionRejectsExecuteAndResourceOpen(t *testing.T) {
	session := newBindingSession()
	closeStarted := make(chan struct{})
	releaseClose := make(chan struct{})
	var closeCallbackOnce sync.Once
	session.closeFunc = func() error {
		closeCallbackOnce.Do(func() {
			close(closeStarted)
			<-releaseClose
			session.closeOnce.Do(func() { close(session.done) })
		})
		return nil
	}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	handle := openBindingSession(t, engine)
	closeResult := make(chan error, 1)
	go func() { closeResult <- engine.CloseSession(handle) }()
	<-closeStarted
	command, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if _, err := engine.Execute(handle, command); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("execute while closing error = %v", err)
	}
	engine.mu.Lock()
	operationCount := len(engine.operations)
	engine.mu.Unlock()
	if operationCount != 0 {
		t.Fatalf("closing session admitted %d execute operations", operationCount)
	}
	resource, _ := proto.Marshal(&bindingpb.OpenResourceStreamRequest{Resource: &apipb.ResourceHandle{OpaqueToken: []byte("resource")}})
	if _, err := engine.OpenResourceStream(handle, resource); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("resource open while closing error = %v", err)
	}
	close(releaseClose)
	if err := <-closeResult; err != nil {
		t.Fatal(err)
	}
}

func TestEngineSessionCloseRevokesBlockedStreamSendBeforeTransportWrite(t *testing.T) {
	session := newBindingSession()
	stream := newBindingResourceStream()
	session.resourceStream = stream
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	requestPayload, _ := proto.Marshal(&bindingpb.OpenResourceStreamRequest{
		Resource: &apipb.ResourceHandle{OpaqueToken: []byte("resource"), Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER},
	})
	streamHandle, err := engine.OpenResourceStream(sessionHandle, requestPayload)
	if err != nil {
		t.Fatal(err)
	}
	dataPayload, _ := proto.Marshal(&wirepb.FileTransferData{Offset: 0, Data: []byte("blocked")})
	framePayload, _ := proto.Marshal(&bindingpb.ResourceStreamFrame{
		StreamHandle: streamHandle,
		Type:         bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
		Payload:      dataPayload,
	})
	engine.mu.Lock()
	record := engine.streams[streamHandle]
	engine.mu.Unlock()
	record.sendMu.Lock()
	sendResult := make(chan error, 1)
	go func() { sendResult <- engine.SendResourceStreamFrame(streamHandle, framePayload) }()
	closeResult := make(chan error, 1)
	go func() { closeResult <- engine.CloseSession(sessionHandle) }()
	deadline := time.Now().Add(time.Second)
	for {
		engine.mu.Lock()
		closing := engine.sessions[sessionHandle].closing && record.closed
		engine.mu.Unlock()
		if closing {
			break
		}
		if time.Now().After(deadline) {
			record.sendMu.Unlock()
			t.Fatal("session close did not revoke stream send")
		}
		time.Sleep(time.Millisecond)
	}
	record.sendMu.Unlock()
	if err := <-sendResult; !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("revoked stream send error = %v", err)
	}
	if err := <-closeResult; err != nil {
		t.Fatal(err)
	}
	if stream.sentPayload != nil {
		t.Fatalf("revoked stream wrote transport payload %x", stream.sentPayload)
	}
}

func TestEngineResourceOpenRechecksSessionAfterProviderReturns(t *testing.T) {
	session := newBindingSession()
	stream := newBindingResourceStream()
	openStarted := make(chan struct{})
	releaseOpen := make(chan struct{})
	session.resourceOpen = func(*apipb.ResourceHandle) (clientruntime.ResourceStream, error) {
		close(openStarted)
		<-releaseOpen
		return stream, nil
	}
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	handle := openBindingSession(t, engine)
	resource, _ := proto.Marshal(&bindingpb.OpenResourceStreamRequest{Resource: &apipb.ResourceHandle{OpaqueToken: []byte("resource")}})
	openResult := make(chan error, 1)
	go func() { _, err := engine.OpenResourceStream(handle, resource); openResult <- err }()
	<-openStarted
	if err := engine.CloseSession(handle); err != nil {
		t.Fatal(err)
	}
	close(releaseOpen)
	if err := <-openResult; !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("late resource open error = %v", err)
	}
	select {
	case <-stream.closed:
	default:
		t.Fatal("late resource stream was not closed")
	}
}

func TestEngineHandleCapacityIncludesResourceStreams(t *testing.T) {
	engine, err := NewEngine(&bindingHost{session: newBindingSession()})
	if err != nil {
		t.Fatal(err)
	}
	engine.mu.Lock()
	for handle := uint64(1); handle <= maxLiveHandles; handle++ {
		engine.streams[handle] = nil
	}
	engine.mu.Unlock()
	if _, _, err := engine.startOperation(); err == nil {
		t.Fatal("operation exceeded mixed binding handle capacity")
	}
	engine.mu.Lock()
	clear(engine.streams)
	engine.mu.Unlock()
	_ = engine.Close()
}

func TestEngineResourceStreamUsesProtoFramesAndOpaqueHandle(t *testing.T) {
	session := newBindingSession()
	stream := newBindingResourceStream()
	session.resourceStream = stream
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	resource := &apipb.ResourceHandle{OpaqueToken: []byte{0, 7, 1}, Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER}
	requestPayload, err := proto.Marshal(&bindingpb.OpenResourceStreamRequest{Resource: resource})
	if err != nil {
		t.Fatal(err)
	}
	streamHandle, err := engine.OpenResourceStream(sessionHandle, requestPayload)
	if err != nil {
		t.Fatal(err)
	}
	if streamHandle == 0 || !proto.Equal(session.resource, resource) {
		t.Fatalf("resource stream handle=%d resource=%v", streamHandle, session.resource)
	}
	dataPayload, err := proto.Marshal(&wirepb.FileTransferData{Offset: 0, Data: []byte("upload")})
	if err != nil {
		t.Fatal(err)
	}
	sendPayload, err := proto.Marshal(&bindingpb.ResourceStreamFrame{
		StreamHandle: streamHandle,
		Type:         bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
		Payload:      dataPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.SendResourceStreamFrame(streamHandle, sendPayload); err != nil {
		t.Fatal(err)
	}
	var sentData wirepb.FileTransferData
	if err := proto.Unmarshal(stream.sentPayload, &sentData); err != nil {
		t.Fatal(err)
	}
	if stream.sentType != 0x21 || string(sentData.GetData()) != "upload" {
		t.Fatalf("sent frame type=%x payload=%q", stream.sentType, stream.sentPayload)
	}
	finishPayload, _ := proto.Marshal(&bindingpb.ResourceStreamFrame{StreamHandle: streamHandle, Type: bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO})
	if err := engine.SendResourceStreamFrame(streamHandle, finishPayload); err != nil {
		t.Fatal(err)
	}
	var finish wirepb.FileTransferFinish
	if err := proto.Unmarshal(stream.sentPayload, &finish); err != nil || finish.GetSize() != 6 || len(finish.GetSha256()) != sha256.Size {
		t.Fatalf("automatic finish = %#v err=%v", &finish, err)
	}
	stream.frames <- bindingResourceFrame{typ: 0x24, payload: []byte("done")}
	event := nextBindingEvent(t, engine)
	if event.GetResourceStreamFrame().GetStreamHandle() != streamHandle ||
		event.GetResourceStreamFrame().GetType() != bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT ||
		string(event.GetResourceStreamFrame().GetPayload()) != "done" {
		t.Fatalf("resource stream event = %#v", event)
	}
	if err := engine.CloseResourceStream(streamHandle); err != nil {
		t.Fatal(err)
	}
	closed := nextBindingEvent(t, engine)
	if closed.GetResourceStreamClosed().GetStreamHandle() != streamHandle {
		t.Fatalf("resource stream closed event = %#v", closed)
	}
	if err := engine.Release(streamHandle); err != nil {
		t.Fatal(err)
	}
}

func TestEngineTerminalAttachmentStreamExposesRawPTYFrames(t *testing.T) {
	session := newBindingSession()
	stream := newBindingResourceStream()
	session.resourceStream = stream
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	requestPayload, err := proto.Marshal(&bindingpb.OpenResourceStreamRequest{Resource: &apipb.ResourceHandle{
		OpaqueToken: []byte{0, 7, 1}, Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT,
	}})
	if err != nil {
		t.Fatal(err)
	}
	streamHandle, err := engine.OpenResourceStream(sessionHandle, requestPayload)
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte{0x00, 0xff, 0x1b, '[', 'm'}
	stream.frames <- bindingResourceFrame{typ: wire.TypePTYOutput, payload: raw}
	rawEvent := nextBindingEvent(t, engine).GetResourceStreamFrame()
	if rawEvent.GetStreamHandle() != streamHandle || rawEvent.GetType() != bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT || !bytes.Equal(rawEvent.GetPayload(), raw) {
		t.Fatalf("raw PTY binding event = %#v", rawEvent)
	}

	stream.frames <- bindingResourceFrame{typ: wire.TypeSyncLost, payload: wire.EncodeSyncLostPayload(4096)}
	syncEvent := nextBindingEvent(t, engine).GetResourceStreamFrame()
	var syncLost bindingpb.PTYStreamSyncLost
	if syncEvent.GetType() != bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST || proto.Unmarshal(syncEvent.GetPayload(), &syncLost) != nil || syncLost.GetDroppedBytes() != 4096 {
		t.Fatalf("raw PTY sync-lost binding event = %#v payload=%#v", syncEvent, &syncLost)
	}

	stream.frames <- bindingResourceFrame{typ: wire.TypeClosed, payload: wire.EncodeClosedPayload(23)}
	closedEvent := nextBindingEvent(t, engine).GetResourceStreamFrame()
	var closed bindingpb.PTYStreamClosed
	if closedEvent.GetType() != bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED || proto.Unmarshal(closedEvent.GetPayload(), &closed) != nil || closed.GetExitCode() != 23 {
		t.Fatalf("raw PTY closed binding event = %#v payload=%#v", closedEvent, &closed)
	}

	outbound, _ := proto.Marshal(&bindingpb.ResourceStreamFrame{
		StreamHandle: streamHandle, Type: bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT, Payload: raw,
	})
	if err := engine.SendResourceStreamFrame(streamHandle, outbound); err == nil {
		t.Fatal("binding accepted outbound PTY output; input must use TerminalInputCommand")
	}
}

func TestEngineResourceStreamAcceptsResumedUploadOffsetAndRequiresExplicitDigest(t *testing.T) {
	session := newBindingSession()
	stream := newBindingResourceStream()
	session.resourceStream = stream
	engine, err := NewEngine(&bindingHost{session: session})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	sessionHandle := openBindingSession(t, engine)
	requestPayload, err := proto.Marshal(&bindingpb.OpenResourceStreamRequest{
		Resource:            &apipb.ResourceHandle{OpaqueToken: []byte{0, 7, 1}, Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER},
		InitialUploadOffset: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	streamHandle, err := engine.OpenResourceStream(sessionHandle, requestPayload)
	if err != nil {
		t.Fatal(err)
	}
	dataPayload, _ := proto.Marshal(&wirepb.FileTransferData{Offset: 4, Data: []byte("tail")})
	framePayload, _ := proto.Marshal(&bindingpb.ResourceStreamFrame{
		StreamHandle: streamHandle, Type: bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_DATA, Payload: dataPayload,
	})
	if err := engine.SendResourceStreamFrame(streamHandle, framePayload); err != nil {
		t.Fatal(err)
	}
	autoPayload, _ := proto.Marshal(&bindingpb.ResourceStreamFrame{StreamHandle: streamHandle, Type: bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO})
	if err := engine.SendResourceStreamFrame(streamHandle, autoPayload); err == nil {
		t.Fatal("resumed upload accepted suffix-only automatic digest")
	}
	digest := sha256.Sum256([]byte("headtail"))
	finishPayload, _ := proto.Marshal(&wirepb.FileTransferFinish{Size: 8, Sha256: digest[:]})
	explicitPayload, _ := proto.Marshal(&bindingpb.ResourceStreamFrame{
		StreamHandle: streamHandle, Type: bindingpb.ResourceStreamFrameType_RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH, Payload: finishPayload,
	})
	if err := engine.SendResourceStreamFrame(streamHandle, explicitPayload); err != nil {
		t.Fatal(err)
	}
}

func TestEngineCloseUnblocksBackpressuredProducer(t *testing.T) {
	session := newBindingSession()
	engine, err := NewEngineWithEventCapacity(&bindingHost{session: session}, 1)
	if err != nil {
		t.Fatal(err)
	}
	openPayload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "close-backpressure", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if _, err := engine.OpenSession(openPayload); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for activeRendererEventCount(engine) != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	engine.mu.Lock()
	var sessionHandle uint64
	for handle := range engine.sessions {
		sessionHandle = handle
	}
	engine.mu.Unlock()
	session.events <- &apipb.EventEnvelope{EventId: "blocked-event", Event: &apipb.EventEnvelope_StorageChanged{StorageChanged: &apipb.StorageChangedEvent{}}}
	closed := make(chan struct{})
	go func() {
		_ = engine.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("engine close blocked behind full event queue")
	}
	if sessionHandle == 0 {
		t.Fatal("session was not established before close")
	}
}

func TestEngineAppliesBoundedBackpressureWithoutDroppingResults(t *testing.T) {
	session := newBindingSession()
	executed := make(chan struct{})
	session.execute = func(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
		close(executed)
		return terminalListBindingResult(), nil
	}
	engine, err := NewEngineWithEventCapacity(&bindingHost{session: session}, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	openPayload, _ := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "open-backpressure", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if _, err := engine.OpenSession(openPayload); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for activeRendererEventCount(engine) != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	engine.mu.Lock()
	var sessionHandle uint64
	for handle := range engine.sessions {
		sessionHandle = handle
	}
	engine.mu.Unlock()
	if sessionHandle == 0 {
		t.Fatal("open session was not published before event backpressure")
	}
	commandPayload, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if _, err := engine.Execute(sessionHandle, commandPayload); err != nil {
		t.Fatal(err)
	}
	<-executed
	openEvent := nextBindingEvent(t, engine)
	if openEvent.GetOpenSession().GetSessionHandle() != sessionHandle {
		t.Fatalf("open event = %#v", openEvent)
	}
	result := nextBindingEvent(t, engine)
	if result.GetExecute().GetResult().GetTerminalList() == nil {
		t.Fatalf("backpressured result was lost: %#v", result)
	}
}

func activeRendererEventCount(engine *Engine) int {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	renderer := engine.renderers[engine.activeRenderer]
	if renderer == nil {
		return 0
	}
	return len(renderer.events)
}

func TestRendererReplacementRevokesHandlesAndIsolatesQueuedEvents(t *testing.T) {
	var sessions []*bindingSession
	host := &bindingHost{open: func(_ context.Context, request *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
		session := newBindingSession()
		session.stamp.EndpointID = endpoint.EndpointID(request.GetEndpointId())
		sessions = append(sessions, session)
		return session, nil
	}}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	rendererA, err := engine.AttachRenderer()
	if err != nil {
		t.Fatal(err)
	}
	operationA, sessionA := openBindingSessionForRenderer(t, engine, rendererA, "renderer-a")
	engine.emitForHandle(sessionA, &bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_Application{Application: &bindingpb.ApplicationEvent{
		SessionHandle: sessionA,
		Event:         &apipb.EventEnvelope{EventId: "stale-renderer-a"},
	}}})

	rendererB, err := engine.AttachRenderer()
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-sessions[0].Done():
	case <-time.After(time.Second):
		t.Fatal("renderer replacement did not close its published session")
	}
	if _, err := engine.NextEventForRenderer(context.Background(), rendererA); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("old renderer event read error = %v, want ErrInvalidHandle", err)
	}
	command, _ := proto.Marshal(&apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if _, err := engine.Execute(sessionA, command); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("old session execute error = %v, want ErrInvalidHandle", err)
	}
	if err := engine.Release(operationA); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("old operation release error = %v, want ErrInvalidHandle", err)
	}

	operationB, sessionB := openBindingSessionForRenderer(t, engine, rendererB, "renderer-b")
	if operationB == operationA || sessionB == sessionA {
		t.Fatalf("renderer handles were reused: operation %d/%d session %d/%d", operationA, operationB, sessionA, sessionB)
	}
	engine.mu.Lock()
	for handle, operation := range engine.operations {
		if operation.rendererID != rendererB {
			engine.mu.Unlock()
			t.Fatalf("operation %d retained renderer %d", handle, operation.rendererID)
		}
	}
	for handle, session := range engine.sessions {
		if session.rendererID != rendererB {
			engine.mu.Unlock()
			t.Fatalf("session %d retained renderer %d", handle, session.rendererID)
		}
	}
	engine.mu.Unlock()
}

func TestRendererReplacementReturnsHandleRegistriesToBaselineAcross500Cycles(t *testing.T) {
	var sessions []*bindingSession
	host := &bindingHost{open: func(_ context.Context, _ *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
		session := newBindingSession()
		sessions = append(sessions, session)
		return session, nil
	}}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	engine.mu.Lock()
	rendererID := engine.activeRenderer
	engine.mu.Unlock()

	for cycle := 0; cycle < 500; cycle++ {
		openBindingSessionForRenderer(t, engine, rendererID, fmt.Sprintf("renderer-cycle-%d", cycle))
		rendererID, err = engine.AttachRenderer()
		if err != nil {
			t.Fatal(err)
		}
		engine.mu.Lock()
		operations, publishedSessions, streams, renderers := len(engine.operations), len(engine.sessions), len(engine.streams), len(engine.renderers)
		engine.mu.Unlock()
		if operations != 0 || publishedSessions != 0 || streams != 0 || renderers != 1 {
			t.Fatalf("cycle %d registry counts = operations %d sessions %d streams %d renderers %d", cycle, operations, publishedSessions, streams, renderers)
		}
	}
	for index, session := range sessions {
		select {
		case <-session.Done():
		default:
			t.Fatalf("cycle %d session remained open", index)
		}
	}
}

func openBindingSessionForRenderer(t *testing.T, engine *Engine, rendererID uint64, requestID string) (uint64, uint64) {
	t.Helper()
	payload, err := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: requestID, EndpointId: requestID, Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := engine.OpenSession(payload)
	if err != nil {
		t.Fatal(err)
	}
	event := nextBindingEventForRenderer(t, engine, rendererID)
	if event.GetSequence() != 1 || event.GetOpenSession().GetRequestId() != requestID || event.GetOpenSession().GetSessionHandle() == 0 {
		t.Fatalf("renderer %d open event = %#v", rendererID, event)
	}
	return operation, event.GetOpenSession().GetSessionHandle()
}

func openBindingSession(t *testing.T, engine *Engine) uint64 {
	t.Helper()
	payload, err := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "open", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := engine.OpenSession(payload)
	if err != nil {
		t.Fatal(err)
	}
	event := nextBindingEvent(t, engine)
	if event.GetOpenSession().GetError() != nil {
		t.Fatalf("open error = %#v", event.GetOpenSession().GetError())
	}
	if err := engine.Release(operation); err != nil {
		t.Fatal(err)
	}
	return event.GetOpenSession().GetSessionHandle()
}

func TestEngineOpenSessionDeadlineEndsPlatformWait(t *testing.T) {
	host := &bindingHost{open: func(ctx context.Context, _ *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	engine, err := NewEngine(host)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	engine.openTimeout = 5 * time.Millisecond
	payload, err := proto.Marshal(&bindingpb.OpenSessionRequest{
		RequestId: "deadline", EndpointId: "studio", Intent: bindingpb.ConnectIntent_CONNECT_INTENT_INTERACTIVE,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.OpenSession(payload); err != nil {
		t.Fatal(err)
	}
	event := nextBindingEvent(t, engine)
	result := event.GetOpenSession()
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE || !result.GetError().GetRetryable() {
		t.Fatalf("deadline error = %#v", result.GetError())
	}
}

func TestDefaultOpenTimeoutCoversCloudControlAndManagedPeerWindows(t *testing.T) {
	const (
		cloudControlWindow = 15 * time.Second
		managedPeerWindow  = 15 * time.Second
		readyAuthMargin    = 5 * time.Second
	)
	minimum := cloudControlWindow + managedPeerWindow + readyAuthMargin
	if defaultOpenTimeout < minimum {
		t.Fatalf("default open timeout = %s, want at least %s", defaultOpenTimeout, minimum)
	}
}

func nextBindingEvent(t *testing.T, engine *Engine) *bindingpb.EventEnvelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	payload, err := engine.NextEvent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	event := &bindingpb.EventEnvelope{}
	if err := proto.Unmarshal(payload, event); err != nil {
		t.Fatal(err)
	}
	return event
}

func nextBindingEventForRenderer(t *testing.T, engine *Engine, rendererID uint64) *bindingpb.EventEnvelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	payload, err := engine.NextEventForRenderer(ctx, rendererID)
	if err != nil {
		t.Fatal(err)
	}
	event := &bindingpb.EventEnvelope{}
	if err := proto.Unmarshal(payload, event); err != nil {
		t.Fatal(err)
	}
	return event
}

func TestAPIErrorPreservesDaemonLifecycle(t *testing.T) {
	for _, test := range []struct {
		name      string
		runtime   *clientruntime.Error
		protoCode apipb.ApiErrorCode
		retryable bool
	}{
		{name: "blocked", runtime: &clientruntime.Error{Code: clientruntime.ErrorDaemonBlocked, Message: "blocked", Attempted: true, Retryable: true}, protoCode: apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_BLOCKED, retryable: true},
		{name: "deleted", runtime: &clientruntime.Error{Code: clientruntime.ErrorDaemonDeleted, Message: "deleted", Attempted: true}, protoCode: apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_DELETED},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := apiError(test.runtime)
			if got.GetCode() != test.protoCode || got.GetRetryable() != test.retryable || !got.GetAttempted() {
				t.Fatalf("api error = %#v", got)
			}
		})
	}
}

func TestAPIErrorPreservesCloudEntitlementCode(t *testing.T) {
	for runtimeCode, want := range map[clientruntime.ErrorCode]apipb.ApiErrorCode{
		clientruntime.ErrorRelayNotInPlan:            apipb.ApiErrorCode_API_ERROR_CODE_RELAY_NOT_IN_PLAN,
		clientruntime.ErrorRelayQuotaExhausted:       apipb.ApiErrorCode_API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED,
		clientruntime.ErrorRelayConcurrencyExhausted: apipb.ApiErrorCode_API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED,
		clientruntime.ErrorSubscriptionInactive:      apipb.ApiErrorCode_API_ERROR_CODE_SUBSCRIPTION_INACTIVE,
		clientruntime.ErrorRelayRegionUnavailable:    apipb.ApiErrorCode_API_ERROR_CODE_RELAY_REGION_UNAVAILABLE,
	} {
		got := apiError(&clientruntime.Error{Code: runtimeCode, Message: "opaque", Attempted: true})
		if got.GetCode() != want || got.GetMessage() != "opaque" {
			t.Fatalf("runtime code %s produced %#v", runtimeCode, got)
		}
	}
}

type bindingHost struct {
	session               clientruntime.ApplicationReadyPeerSession
	request               *bindingpb.OpenSessionRequest
	open                  func(context.Context, *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error)
	invalidated           clientruntime.EndpointSessionStamp
	disconnected          endpoint.EndpointID
	cloudPresence         bool
	cloudPresenceEndpoint string
}

type lifecycleBindingHost struct {
	*bindingHost
	events chan clientruntime.EndpointEvent
}

type stubbornLifecycleBindingHost struct {
	*bindingHost
	events chan clientruntime.EndpointEvent
}

func (host *stubbornLifecycleBindingHost) WatchEndpoint(context.Context, endpoint.EndpointID) (<-chan clientruntime.EndpointEvent, error) {
	return host.events, nil
}

func (host *lifecycleBindingHost) WatchEndpoint(ctx context.Context, _ endpoint.EndpointID) (<-chan clientruntime.EndpointEvent, error) {
	out := make(chan clientruntime.EndpointEvent, cap(host.events))
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-host.events:
				select {
				case <-ctx.Done():
					return
				case out <- event:
				}
			}
		}
	}()
	return out, nil
}

type protocolBindingSession struct {
	client *internalprotocol.Client
	stamp  clientruntime.EndpointSessionStamp
	nextID atomic.Uint64
}

func (session *protocolBindingSession) Stamp() clientruntime.EndpointSessionStamp {
	return session.stamp
}
func (session *protocolBindingSession) ObservedPath() string { return "memory" }
func (session *protocolBindingSession) Readiness() clientruntime.ReadyPeerSessionEvidence {
	return clientruntime.ReadyPeerSessionEvidence{Identity: endpoint.DaemonIdentity{DeviceID: "device-binding-protocol", DeviceFingerprint: "SHA256:device-binding-protocol"}, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version}
}
func (session *protocolBindingSession) Done() <-chan struct{} { return session.client.Done() }
func (session *protocolBindingSession) Err() error            { return session.client.Err() }
func (session *protocolBindingSession) Close() error          { return session.client.Close() }
func (session *protocolBindingSession) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	return session.client.ApplicationEvents(ctx)
}
func (session *protocolBindingSession) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.executeApplication(ctx, command, false)
}
func (session *protocolBindingSession) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return session.executeApplication(ctx, command, true)
}
func (session *protocolBindingSession) executeApplication(ctx context.Context, command *apipb.CommandEnvelope, terminal bool) (*apipb.ResultEnvelope, error) {
	requestID := fmt.Sprintf("binding-%d", session.nextID.Add(1))
	stamp := &apipb.EndpointSessionStamp{EndpointId: string(session.stamp.EndpointID), RouteId: string(session.stamp.RouteID), Generation: uint64(session.stamp.Generation)}
	snapshot := proto.Clone(command).(*apipb.CommandEnvelope)
	snapshot.Context = &apipb.RequestContext{RequestId: requestID, ApiVersion: &apipb.ApiVersion{Major: 1}, Session: stamp}
	operation := &apipb.OperationStamp{Session: proto.Clone(stamp).(*apipb.EndpointSessionStamp), OperationId: requestID}
	if snapshot.GetFileUploadOpen() != nil {
		snapshot.GetFileUploadOpen().Operation = operation
	}
	if snapshot.GetFileTransferCancel() != nil {
		snapshot.GetFileTransferCancel().Operation = operation
	}
	if terminal {
		return session.client.ExecuteApplicationTerminal(ctx, snapshot)
	}
	return session.client.ExecuteApplication(ctx, snapshot)
}

func bindingExpectHello(transport *memory.Transport) error {
	channel, typ, payload, err := bindingReceiveFrame(transport)
	if err != nil || channel != 0 || typ != wire.TypeHello {
		return fmt.Errorf("hello frame channel=%d type=%d err=%v", channel, typ, err)
	}
	if _, err := internalprotocol.DecodeHelloPayload(payload); err != nil {
		return err
	}
	response, err := internalprotocol.EncodeHelloPayload(internalprotocol.Hello{Version: wire.Version, Server: "binding-test"})
	if err != nil {
		return err
	}
	return bindingSendFrame(transport, 0, wire.TypeHello, response)
}

func bindingReceiveApplicationRequest(transport *memory.Transport) (internalprotocol.Request, *apipb.CommandEnvelope, error) {
	channel, typ, payload, err := bindingReceiveFrame(transport)
	if err != nil || channel != 0 || typ != wire.TypeRequest {
		return internalprotocol.Request{}, nil, fmt.Errorf("application frame channel=%d type=%d err=%v", channel, typ, err)
	}
	request, err := internalprotocol.DecodeRequestPayload(payload)
	if err != nil {
		return internalprotocol.Request{}, nil, err
	}
	command := &apipb.CommandEnvelope{}
	if err := proto.Unmarshal(request.Params, command); err != nil {
		return internalprotocol.Request{}, nil, err
	}
	return request, command, nil
}

func bindingSendApplicationResult(transport *memory.Transport, id uint64, result *apipb.ResultEnvelope) error {
	payload, err := proto.Marshal(result)
	if err != nil {
		return err
	}
	response, err := internalprotocol.EncodeResponsePayload(internalprotocol.Response{ID: id, Result: payload})
	if err != nil {
		return err
	}
	return bindingSendFrame(transport, 0, wire.TypeResponse, response)
}

func bindingReceiveFrame(transport *memory.Transport) (uint16, uint8, []byte, error) {
	frame, err := transport.Recv()
	if err != nil {
		return 0, 0, nil, err
	}
	return wire.DecodeFrame(frame)
}

func bindingSendFrame(transport *memory.Transport, channel uint16, typ uint8, payload []byte) error {
	frame, err := wire.EncodeFrame(channel, typ, payload)
	if err != nil {
		return err
	}
	return transport.Send(frame)
}

func (host *bindingHost) OpenSession(ctx context.Context, request *bindingpb.OpenSessionRequest) (clientruntime.ApplicationReadyPeerSession, error) {
	host.request = proto.Clone(request).(*bindingpb.OpenSessionRequest)
	if host.open != nil {
		return host.open(ctx, request)
	}
	return host.session, nil
}

func (host *bindingHost) InvalidateSession(_ context.Context, stamp clientruntime.EndpointSessionStamp) error {
	host.invalidated = stamp
	return nil
}

func (host *bindingHost) DisconnectEndpoint(_ context.Context, endpointID endpoint.EndpointID) error {
	host.disconnected = endpointID
	return nil
}

func (host *bindingHost) GetEndpointCloudPresence(_ context.Context, request *bindingpb.EndpointCloudPresenceGetRequest) (*bindingpb.EndpointCloudPresenceGetResult, error) {
	host.cloudPresenceEndpoint = request.GetEndpointId()
	return &bindingpb.EndpointCloudPresenceGetResult{Online: host.cloudPresence}, nil
}

type bindingSession struct {
	stamp             clientruntime.EndpointSessionStamp
	done              chan struct{}
	doneFunc          func() <-chan struct{}
	events            chan *apipb.EventEnvelope
	closeOnce         sync.Once
	err               error
	command           *apipb.CommandEnvelope
	execute           func(context.Context, *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error)
	resource          *apipb.ResourceHandle
	resourceStream    clientruntime.ResourceStream
	resourceOpen      func(*apipb.ResourceHandle) (clientruntime.ResourceStream, error)
	applicationEvents func(context.Context) (<-chan *apipb.EventEnvelope, error)
	closeFunc         func() error
	connection        *clientruntime.ConnectionSnapshot
}

func (session *bindingSession) ConnectionSnapshot(_ time.Time) (clientruntime.ConnectionSnapshot, bool) {
	if session.connection == nil {
		return clientruntime.ConnectionSnapshot{}, false
	}
	return *session.connection, true
}

func (session *bindingSession) OpenResourceStream(resource *apipb.ResourceHandle) (clientruntime.ResourceStream, error) {
	if session.resourceOpen != nil {
		return session.resourceOpen(resource)
	}
	session.resource = proto.Clone(resource).(*apipb.ResourceHandle)
	if session.resourceStream == nil {
		return nil, errors.New("resource stream unavailable")
	}
	return session.resourceStream, nil
}

type bindingResourceFrame struct {
	typ     uint8
	payload []byte
}

type bindingResourceStream struct {
	frames      chan bindingResourceFrame
	closed      chan struct{}
	closeOnce   sync.Once
	sentType    uint8
	sentPayload []byte
}

func newBindingResourceStream() *bindingResourceStream {
	return &bindingResourceStream{frames: make(chan bindingResourceFrame, 4), closed: make(chan struct{})}
}

func (stream *bindingResourceStream) Receive(ctx context.Context) (uint8, []byte, error) {
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case <-stream.closed:
		return 0, nil, io.EOF
	case frame := <-stream.frames:
		return frame.typ, append([]byte(nil), frame.payload...), nil
	}
}

func (stream *bindingResourceStream) Send(_ context.Context, typ uint8, payload []byte) error {
	stream.sentType = typ
	stream.sentPayload = append([]byte(nil), payload...)
	return nil
}

func (stream *bindingResourceStream) Close() error {
	stream.closeOnce.Do(func() { close(stream.closed) })
	return nil
}

func newBindingSession() *bindingSession {
	return &bindingSession{
		stamp: clientruntime.EndpointSessionStamp{EndpointID: endpoint.EndpointID("studio"), RouteID: endpoint.RouteID("cloud"), Generation: 3},
		done:  make(chan struct{}), events: make(chan *apipb.EventEnvelope, 8),
	}
}

func (session *bindingSession) Stamp() clientruntime.EndpointSessionStamp { return session.stamp }
func (session *bindingSession) ObservedPath() string                      { return "direct" }
func (session *bindingSession) Readiness() clientruntime.ReadyPeerSessionEvidence {
	return clientruntime.ReadyPeerSessionEvidence{Identity: endpoint.DaemonIdentity{DeviceID: "device-binding", DeviceFingerprint: "SHA256:device-binding"}, IdentityVerified: true, AuthorizationVerified: true, ProtocolVersion: wire.Version}
}
func (session *bindingSession) Done() <-chan struct{} {
	if session.doneFunc != nil {
		return session.doneFunc()
	}
	return session.done
}
func (session *bindingSession) Err() error { return session.err }
func (session *bindingSession) Close() error {
	if session.closeFunc != nil {
		return session.closeFunc()
	}
	session.closeOnce.Do(func() { close(session.done) })
	return nil
}
func (session *bindingSession) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	session.command = proto.Clone(command).(*apipb.CommandEnvelope)
	if session.execute != nil {
		return session.execute(ctx, command)
	}
	return terminalListBindingResult(), nil
}
func (session *bindingSession) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	if session.applicationEvents != nil {
		return session.applicationEvents(ctx)
	}
	out := make(chan *apipb.EventEnvelope, 8)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-session.done:
				return
			case event := <-session.events:
				select {
				case out <- event:
				case <-ctx.Done():
					return
				case <-session.done:
					return
				}
			}
		}
	}()
	return out, nil
}

func terminalListBindingResult() *apipb.ResultEnvelope {
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_TerminalList{TerminalList: &apipb.TerminalListResult{}}}
}

var _ clientruntime.ApplicationReadyPeerSession = (*bindingSession)(nil)
