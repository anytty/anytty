package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

type recordingProtoExecutor struct {
	command  *apipb.CommandEnvelope
	result   *apipb.ResultEnvelope
	terminal bool
}

type nestedApplicationExecutor struct {
	inner *ApplicationSession
}

type staleApplicationExecutor struct {
	called bool
}

type echoApplicationExecutor struct{}

type executeOnlyFileOpenExecutor struct {
	commands []*apipb.CommandEnvelope
}

type eventSubscriptionExecutor struct {
	mu               sync.Mutex
	commands         []*apipb.CommandEnvelope
	events           chan *apipb.EventEnvelope
	released         chan struct{}
	subscribeStarted chan struct{}
	subscribeRelease chan struct{}
	once             sync.Once
	startedOnce      sync.Once
}

type historyWindowCancellationExecutor struct {
	started  chan struct{}
	complete chan struct{}
	released chan *apipb.HistoryReleaseCommand
}

func (executor *historyWindowCancellationExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	result := &apipb.ResultEnvelope{RequestId: command.GetContext().GetRequestId(), OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp)}
	if release := command.GetHistoryRelease(); release != nil {
		executor.released <- proto.Clone(release).(*apipb.HistoryReleaseCommand)
		result.Result = &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}
	}
	return result, nil
}

func (executor *historyWindowCancellationExecutor) ExecuteApplicationTerminal(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	close(executor.started)
	<-executor.complete
	return &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result: &apipb.ResultEnvelope_HistoryWindow{HistoryWindow: &apipb.HistoryWindowResult{
			Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}, Token: "late-token", HistoryGeneration: 7,
		}},
	}, nil
}

func (executor *eventSubscriptionExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	executor.mu.Lock()
	executor.commands = append(executor.commands, proto.Clone(command).(*apipb.CommandEnvelope))
	executor.mu.Unlock()
	result := &apipb.ResultEnvelope{RequestId: command.GetContext().GetRequestId(), OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp)}
	if command.GetEventSubscribe() != nil {
		if executor.subscribeStarted != nil {
			executor.startedOnce.Do(func() { close(executor.subscribeStarted) })
		}
		if executor.subscribeRelease != nil {
			<-executor.subscribeRelease
		}
		result.Result = &apipb.ResultEnvelope_EventSubscription{EventSubscription: &apipb.EventSubscriptionResult{Subscription: &apipb.ResourceHandle{
			Kind: apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION, OpaqueToken: []byte("event-token"),
		}}}
	} else {
		result.Result = &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}
		if command.GetReleaseResource() != nil {
			executor.once.Do(func() { close(executor.released) })
		}
	}
	return result, nil
}

func (executor *eventSubscriptionExecutor) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return executor.ExecuteApplication(ctx, command)
}

func (executor *eventSubscriptionExecutor) ApplicationEvents(context.Context) (<-chan *apipb.EventEnvelope, error) {
	return executor.events, nil
}

func (executor *executeOnlyFileOpenExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	executor.commands = append(executor.commands, proto.Clone(command).(*apipb.CommandEnvelope))
	return &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result:        &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: &apipb.FileTransferOpenResult{}},
	}, nil
}

func (executor *recordingProtoExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	executor.command = command
	if executor.result != nil {
		result := proto.Clone(executor.result).(*apipb.ResultEnvelope)
		if result.RequestId == "" {
			result.RequestId = command.GetContext().GetRequestId()
		}
		if result.OriginSession == nil {
			result.OriginSession = proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp)
		}
		return result, nil
	}
	return &apipb.ResultEnvelope{RequestId: command.GetContext().GetRequestId(), OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp), Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}, nil
}

func (executor *recordingProtoExecutor) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	executor.terminal = true
	return executor.ExecuteApplication(ctx, command)
}

func (executor *nestedApplicationExecutor) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return executor.inner.Execute(ctx, command)
}

func (executor *nestedApplicationExecutor) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return executor.inner.ExecuteTerminal(ctx, command)
}

func (executor *staleApplicationExecutor) ValidateApplicationSession(EndpointSessionStamp) error {
	return runtimeError(ErrorStaleSession, "session was replaced", nil)
}

func (executor *staleApplicationExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	executor.called = true
	return &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result:        &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}},
	}, nil
}

func (echoApplicationExecutor) ExecuteApplication(_ context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return &apipb.ResultEnvelope{
		RequestId:     command.GetContext().GetRequestId(),
		OriginSession: proto.Clone(command.GetContext().GetSession()).(*apipb.EndpointSessionStamp),
		Result:        &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}},
	}, nil
}

func (executor echoApplicationExecutor) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	return executor.ExecuteApplication(ctx, command)
}

func TestApplicationSessionOwnsContextAndOperationStamp(t *testing.T) {
	executor := &recordingProtoExecutor{}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: endpoint.EndpointID("studio"), RouteID: endpoint.RouteID("ssh"), Generation: 7}, executor)
	if err != nil {
		t.Fatal(err)
	}
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachCommand{Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}}}}
	if _, err := session.Execute(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	contextMessage := executor.command.GetContext()
	operation := executor.command.GetTerminalAttach().GetOperation()
	if contextMessage.GetSession().GetRouteId() != "ssh" || contextMessage.GetSession().GetGeneration() != 7 || contextMessage.GetRequestId() == "" {
		t.Fatalf("unexpected request context %#v", contextMessage)
	}
	if operation.GetOperationId() != contextMessage.GetRequestId() || operation.GetSession().GetGeneration() != 7 {
		t.Fatalf("unexpected operation stamp %#v", operation)
	}
	if command.GetContext() != nil || command.GetTerminalAttach().GetOperation() != nil {
		t.Fatal("application session mutated caller command")
	}
}

func TestNestedApplicationSessionsPreserveOwningRequestIdentity(t *testing.T) {
	stamp := EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}
	executor := &recordingProtoExecutor{}
	inner, err := NewApplicationSession(stamp, executor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inner.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}}); err != nil {
		t.Fatal(err)
	}
	consumedID := executor.command.GetContext().GetRequestId()
	outer, err := NewApplicationSession(stamp, &nestedApplicationExecutor{inner: inner})
	if err != nil {
		t.Fatal(err)
	}

	result, err := outer.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalInput{TerminalInput: &apipb.TerminalInputCommand{}}})
	if err != nil {
		t.Fatalf("nested ordinary request: %v", err)
	}
	requestID := executor.command.GetContext().GetRequestId()
	if requestID == consumedID || result.GetRequestId() != requestID || executor.command.GetTerminalInput().GetOperation().GetOperationId() != requestID {
		t.Fatalf("nested ordinary correlation consumed=%q request=%q result=%q operation=%q", consumedID, requestID, result.GetRequestId(), executor.command.GetTerminalInput().GetOperation().GetOperationId())
	}

	result, err = outer.ExecuteTerminal(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileUploadOpen{FileUploadOpen: &apipb.FileUploadOpenCommand{Path: "/tmp/demo"}}})
	if err != nil {
		t.Fatalf("nested terminal request: %v", err)
	}
	requestID = executor.command.GetContext().GetRequestId()
	if result.GetRequestId() != requestID || executor.command.GetFileUploadOpen().GetOperation().GetOperationId() != requestID {
		t.Fatalf("nested terminal correlation request=%q result=%q operation=%q", requestID, result.GetRequestId(), executor.command.GetFileUploadOpen().GetOperation().GetOperationId())
	}
}

func TestNestedApplicationSessionsRejectCrossGenerationForwarding(t *testing.T) {
	executor := &recordingProtoExecutor{}
	inner, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 8}, executor)
	if err != nil {
		t.Fatal(err)
	}
	outer, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}, &nestedApplicationExecutor{inner: inner})
	if err != nil {
		t.Fatal(err)
	}
	_, err = outer.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if CodeOf(err) != ErrorStaleSession || WasAttempted(err) || executor.command != nil {
		t.Fatalf("cross-generation forwarding error=%#v executed=%t", err, executor.command != nil)
	}
}

func TestNestedApplicationSessionsKeepConcurrentCorrelationsDistinct(t *testing.T) {
	stamp := EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}
	inner, err := NewApplicationSession(stamp, echoApplicationExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := NewApplicationSession(stamp, &nestedApplicationExecutor{inner: inner})
	if err != nil {
		t.Fatal(err)
	}

	const requestCount = 128
	results := make(chan string, requestCount)
	errors := make(chan error, requestCount)
	var wait sync.WaitGroup
	for range requestCount {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := outer.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
			if err != nil {
				errors <- err
				return
			}
			results <- result.GetRequestId()
		}()
	}
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Errorf("concurrent nested request: %v", err)
	}
	seen := make(map[string]struct{}, requestCount)
	for requestID := range results {
		if requestID == "" {
			t.Error("concurrent nested request returned an empty correlation")
			continue
		}
		if _, duplicate := seen[requestID]; duplicate {
			t.Errorf("concurrent nested request reused correlation %q", requestID)
		}
		seen[requestID] = struct{}{}
	}
	if len(seen) != requestCount {
		t.Fatalf("concurrent request correlations = %d, want %d", len(seen), requestCount)
	}
}

func TestApplicationSessionsUseDistinctRequestIdentities(t *testing.T) {
	stamp := EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}
	firstExecutor := &recordingProtoExecutor{}
	first, err := NewApplicationSession(stamp, firstExecutor)
	if err != nil {
		t.Fatal(err)
	}
	secondExecutor := &recordingProtoExecutor{}
	second, err := NewApplicationSession(stamp, secondExecutor)
	if err != nil {
		t.Fatal(err)
	}
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}}
	if _, err := first.Execute(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Execute(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	firstID := firstExecutor.command.GetContext().GetRequestId()
	secondID := secondExecutor.command.GetContext().GetRequestId()
	if firstID == secondID {
		t.Fatalf("separate application sessions reused request identity %q", firstID)
	}
}

func TestApplicationSessionValidatesGenerationBeforeExecution(t *testing.T) {
	executor := &staleApplicationExecutor{}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}, executor)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if CodeOf(err) != ErrorStaleSession || executor.called {
		t.Fatalf("stale generation error=%#v executed=%t", err, executor.called)
	}
}

func TestApplicationSessionPreservesCallerOperationIdentityAndOwnsSessionStamp(t *testing.T) {
	executor := &recordingProtoExecutor{}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "ssh", Generation: 7}, executor)
	if err != nil {
		t.Fatal(err)
	}
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalInput{TerminalInput: &apipb.TerminalInputCommand{
		Operation: &apipb.OperationStamp{OperationId: "ui-input-19", Session: &apipb.EndpointSessionStamp{EndpointId: "forged", RouteId: "old", Generation: 1}},
	}}}
	if _, err := session.Execute(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	operation := executor.command.GetTerminalInput().GetOperation()
	if operation.GetOperationId() != "ui-input-19" || operation.GetSession().GetEndpointId() != "studio" || operation.GetSession().GetRouteId() != "ssh" || operation.GetSession().GetGeneration() != 7 {
		t.Fatalf("unexpected operation stamp %#v", operation)
	}
}

func TestApplicationSessionTerminalExecutionOwnsContextAndOperationStamp(t *testing.T) {
	executor := &recordingProtoExecutor{}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 9}, executor)
	if err != nil {
		t.Fatal(err)
	}
	command := &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileUploadOpen{FileUploadOpen: &apipb.FileUploadOpenCommand{Path: "/tmp/demo", Size: 8}}}
	if _, err := session.ExecuteTerminal(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if !executor.terminal {
		t.Fatal("terminal executor was not selected")
	}
	if executor.command.GetContext().GetSession().GetGeneration() != 9 || executor.command.GetFileUploadOpen().GetOperation().GetOperationId() == "" {
		t.Fatalf("terminal command was not stamped: %#v", executor.command)
	}
	if command.GetContext() != nil || command.GetFileUploadOpen().GetOperation() != nil {
		t.Fatal("terminal execution mutated caller command")
	}
}

func TestApplicationSessionEventSubscribeReleasesResourceWhenContextEnds(t *testing.T) {
	executor := &eventSubscriptionExecutor{events: make(chan *apipb.EventEnvelope), released: make(chan struct{})}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "ssh", Generation: 7}, executor)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, events, err := session.EventSubscribe(ctx, &apipb.EventSubscribeCommand{})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("filtered event stream did not close after cancellation")
	}
	select {
	case <-executor.released:
	case <-time.After(time.Second):
		t.Fatal("event subscription resource was not released")
	}
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if len(executor.commands) != 2 {
		t.Fatalf("event subscription commands = %d, want subscribe and release", len(executor.commands))
	}
	resource := executor.commands[1].GetReleaseResource().GetResource()
	if resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION || string(resource.GetOpaqueToken()) != "event-token" {
		t.Fatalf("released event resource = %#v", resource)
	}
}

func TestApplicationSessionEventSubscribeReleasesLateCreationAfterCancellation(t *testing.T) {
	executor := &eventSubscriptionExecutor{
		events:           make(chan *apipb.EventEnvelope),
		released:         make(chan struct{}),
		subscribeStarted: make(chan struct{}),
		subscribeRelease: make(chan struct{}),
	}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "ssh", Generation: 7}, executor)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, _, err := session.EventSubscribe(ctx, &apipb.EventSubscribeCommand{})
		result <- err
	}()
	<-executor.subscribeStarted
	cancel()
	close(executor.subscribeRelease)
	if err := <-result; err != context.Canceled {
		t.Fatalf("event subscribe error=%v, want context canceled", err)
	}
	select {
	case <-executor.released:
	case <-time.After(time.Second):
		t.Fatal("late event subscription resource was not released")
	}
}

func TestApplicationSessionHistoryWindowReleasesLateSnapshotAfterCancellation(t *testing.T) {
	executor := &historyWindowCancellationExecutor{
		started: make(chan struct{}), complete: make(chan struct{}), released: make(chan *apipb.HistoryReleaseCommand, 1),
	}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 7}, executor)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := session.HistoryWindow(ctx, &apipb.HistoryWindowCommand{Mode: apipb.HistoryWindowMode_HISTORY_WINDOW_MODE_LATEST})
		result <- err
	}()
	<-executor.started
	cancel()
	close(executor.complete)
	if err := <-result; err != context.Canceled {
		t.Fatalf("history window error=%v, want context canceled", err)
	}
	select {
	case release := <-executor.released:
		if release.GetToken() != "late-token" || release.GetHistoryGeneration() != 7 || release.GetTerminal().GetTerminalId() != "term-1" {
			t.Fatalf("history release=%#v", release)
		}
	case <-time.After(time.Second):
		t.Fatal("late history snapshot was not released")
	}
}

func TestGeneratedFileOpenWrappersDoNotRequireTerminalResponseExecutor(t *testing.T) {
	executor := &executeOnlyFileOpenExecutor{}
	if _, ok := any(executor).(TerminalResponseApplicationExecutor); ok {
		t.Fatal("test executor unexpectedly supports terminal responses")
	}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "studio", RouteID: "cloud", Generation: 9}, executor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.FileDownloadOpen(context.Background(), &apipb.FileDownloadOpenCommand{Path: "/tmp/download"}); err != nil {
		t.Fatalf("FileDownloadOpen with execute-only executor: %v", err)
	}
	if _, err := session.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{Path: "/tmp/upload", Size: 8}); err != nil {
		t.Fatalf("FileUploadOpen with execute-only executor: %v", err)
	}
	if len(executor.commands) != 2 || executor.commands[0].GetFileDownloadOpen() == nil || executor.commands[1].GetFileUploadOpen() == nil {
		t.Fatalf("file-open commands = %#v", executor.commands)
	}
}

func TestApplicationSessionConvertsTypedAPIError(t *testing.T) {
	executor := &recordingProtoExecutor{result: &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_STALE_SESSION, Message: "stale", Attempted: false}}}}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "local", RouteID: "unix", Generation: 2}, executor)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
	if CodeOf(err) != ErrorStaleSession || WasAttempted(err) {
		t.Fatalf("unexpected runtime error %#v", err)
	}
}

func TestRuntimeErrorFromProtoPreservesDaemonLifecycle(t *testing.T) {
	for _, test := range []struct {
		name      string
		protoCode apipb.ApiErrorCode
		wantCode  ErrorCode
		retryable bool
	}{
		{name: "blocked", protoCode: apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_BLOCKED, wantCode: ErrorDaemonBlocked, retryable: true},
		{name: "deleted", protoCode: apipb.ApiErrorCode_API_ERROR_CODE_DAEMON_DELETED, wantCode: ErrorDaemonDeleted},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := runtimeErrorFromProto(&apipb.ApiError{Code: test.protoCode, Message: test.name, Attempted: true, Retryable: test.retryable})
			var runtimeErr *Error
			if !errors.As(err, &runtimeErr) {
				t.Fatalf("error = %#v", err)
			}
			if runtimeErr.Code != test.wantCode || !runtimeErr.Attempted || runtimeErr.Retryable != test.retryable {
				t.Fatalf("runtime error = %#v", runtimeErr)
			}
		})
	}
}

func TestRuntimeErrorFromProtoPreservesCloudEntitlementCode(t *testing.T) {
	for protoCode, want := range map[apipb.ApiErrorCode]ErrorCode{
		apipb.ApiErrorCode_API_ERROR_CODE_RELAY_NOT_IN_PLAN:           ErrorRelayNotInPlan,
		apipb.ApiErrorCode_API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED:       ErrorRelayQuotaExhausted,
		apipb.ApiErrorCode_API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED: ErrorRelayConcurrencyExhausted,
		apipb.ApiErrorCode_API_ERROR_CODE_SUBSCRIPTION_INACTIVE:       ErrorSubscriptionInactive,
		apipb.ApiErrorCode_API_ERROR_CODE_RELAY_REGION_UNAVAILABLE:    ErrorRelayRegionUnavailable,
	} {
		if got := CodeOf(runtimeErrorFromProto(&apipb.ApiError{Code: protoCode, Message: "opaque"})); got != want {
			t.Fatalf("proto code %s mapped to %s, want %s", protoCode, got, want)
		}
	}
}

func TestApplicationSessionConvertsNotFoundAPIError(t *testing.T) {
	executor := &recordingProtoExecutor{result: &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Error{Error: &apipb.ApiError{
		Code: apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, Message: "missing", Attempted: true,
	}}}}
	session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "local", RouteID: "unix", Generation: 2}, executor)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalGet{
		TerminalGet: &apipb.TerminalGetCommand{Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "missing"}},
	}})
	if CodeOf(err) != ErrorNotFound || !WasAttempted(err) {
		t.Fatalf("unexpected runtime error %#v", err)
	}
}

func TestApplicationSessionRejectsMismatchedResultCorrelation(t *testing.T) {
	tests := []struct {
		name   string
		result *apipb.ResultEnvelope
		code   ErrorCode
	}{
		{name: "request", result: &apipb.ResultEnvelope{RequestId: "wrong", Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}, code: ErrorUnavailable},
		{name: "session", result: &apipb.ResultEnvelope{OriginSession: &apipb.EndpointSessionStamp{EndpointId: "local", RouteId: "unix", Generation: 8}, Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}, code: ErrorStaleSession},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &recordingProtoExecutor{result: test.result}
			session, err := NewApplicationSession(EndpointSessionStamp{EndpointID: "local", RouteID: "unix", Generation: 7}, executor)
			if err != nil {
				t.Fatal(err)
			}
			_, err = session.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_TerminalList{TerminalList: &apipb.TerminalListCommand{}}})
			if CodeOf(err) != test.code {
				t.Fatalf("error=%v code=%s, want %s", err, CodeOf(err), test.code)
			}
		})
	}
}
