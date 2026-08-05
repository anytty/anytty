package core

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestProtocolSessionRejectsExcessRequestWithoutBlockingReadLoop(t *testing.T) {
	executor := &blockingProtocolExecutor{started: make(chan struct{}), release: make(chan struct{})}
	server := NewServer(
		WithProtocolSessionLimits(ProtocolSessionLimits{MaxInFlightRequests: 1}),
		WithApplicationExecutorFactory(func(ApplicationSessionPort) ApplicationExecutor { return executor }),
	)
	client, daemon := memory.NewPair()
	done := make(chan error, 1)
	go func() {
		done <- newProtocolSession(server, daemon, fullDaemonTransportScope()).run(context.Background())
	}()

	helloPayload, err := protocol.EncodeHelloPayload(protocol.Hello{Version: wire.Version, Client: "limit-test"})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolFrame(t, client, wire.TypeHello, helloPayload)
	if _, typ, _ := receiveProtocolFrame(t, client); typ != wire.TypeHello {
		t.Fatalf("hello response type = %d", typ)
	}
	commandPayload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolRequest(t, client, protocol.Request{ID: 1, Method: "api.execute", Params: commandPayload})
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("first request did not enter executor")
	}
	sendProtocolRequest(t, client, protocol.Request{ID: 2, Method: "api.execute", Params: commandPayload})
	_, typ, payload := receiveProtocolFrame(t, client)
	if typ != wire.TypeError {
		t.Fatalf("excess request response type = %d", typ)
	}
	message, err := protocol.DecodeErrorPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if message.ID != 2 || message.Error.Code != protocolErrorExhausted {
		t.Fatalf("excess request error = %#v", message)
	}
	close(executor.release)
	_, typ, payload = receiveProtocolFrame(t, client)
	if typ != wire.TypeResponse {
		t.Fatalf("first request response type = %d", typ)
	}
	response, err := protocol.DecodeResponsePayload(payload)
	if err != nil || response.ID != 1 {
		t.Fatalf("first response = %#v, %v", response, err)
	}
	_ = client.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("session exit = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("session did not stop")
	}
}

func TestProtocolChannelAllocatorNeverReusesReleasedIDs(t *testing.T) {
	server := NewServer(WithProtocolSessionLimits(ProtocolSessionLimits{
		MaxResources: 8, MaxAttachments: 4, MaxFileTransfers: 4,
	}))
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	session.nextChannel = maxProtocolChannelID - 1
	attachmentChannel, err := session.reserveAttachmentChannel()
	if err != nil {
		t.Fatal(err)
	}
	if attachmentChannel != ^uint16(0) {
		t.Fatalf("last channel = %d", attachmentChannel)
	}
	session.releaseChannel(attachmentChannel, protocolChannelAttachment)
	if _, err := session.reserveFileChannel(); !errors.Is(err, ErrProtocolResourceExhausted) {
		t.Fatalf("released channel allowed reuse: %v", err)
	}
}

func TestProtocolChannelAllocatorReturnsExplicitExhaustion(t *testing.T) {
	server := NewServer(WithProtocolSessionLimits(ProtocolSessionLimits{
		MaxResources: int(^uint16(0)), MaxAttachments: int(^uint16(0)),
	}))
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	session.nextChannel = maxProtocolChannelID
	_, err := session.reserveAttachmentChannel()
	if !errors.Is(err, ErrProtocolResourceExhausted) {
		t.Fatalf("channel exhaustion error = %v", err)
	}
}

func TestProtocolConcurrentSameViewReplacementAndDetachLeavesNoGhost(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1})
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	first, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	publishedLocally := make(chan struct{})
	releasePublication := make(chan struct{})
	var hookOnce sync.Once
	session.beforeGlobalAttachmentPublish = func(protocolAttachment) {
		hookOnce.Do(func() {
			close(publishedLocally)
			<-releasePublication
		})
	}
	replacementResult := make(chan TerminalAttachmentTransaction, 1)
	replacementErr := make(chan error, 1)
	go func() {
		transaction, attachErr := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
		replacementResult <- transaction
		replacementErr <- attachErr
	}()
	select {
	case <-publishedLocally:
	case <-time.After(time.Second):
		t.Fatal("replacement did not reach local publication")
	}
	detachDone := make(chan struct{})
	go func() {
		session.detach(attachmentDetachRequest{TerminalID: "term-resource", SurfaceID: "surface", ViewID: "view"})
		close(detachDone)
	}()
	close(releasePublication)
	transaction := <-replacementResult
	if err := <-replacementErr; err != nil {
		t.Fatal(err)
	}
	select {
	case <-detachDone:
	case <-time.After(time.Second):
		t.Fatal("concurrent detach did not complete")
	}
	if transaction != nil {
		if err := transaction.Rollback(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	assertProtocolResourceCounts(t, session, 0, 0, 0)
	session.mu.RLock()
	localAttachments := len(session.attachments)
	session.mu.RUnlock()
	server.protocolAttachmentMu.Lock()
	globalAttachments := len(server.protocolAttachments)
	resizeOwners := len(server.protocolResizeOwners)
	server.protocolAttachmentMu.Unlock()
	if localAttachments != 0 || globalAttachments != 0 || resizeOwners != 0 {
		t.Fatalf("ghost attachment state local=%d global=%d owners=%d", localAttachments, globalAttachments, resizeOwners)
	}
}

func TestProtocolSameViewReplacementIsNetZeroAtAttachmentLimit(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1})
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	first, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	replacement, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
	if err != nil {
		t.Fatalf("net-zero replacement was rejected at limit: %v", err)
	}
	if err := replacement.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertProtocolResourceCounts(t, session, 1, 0, 0)
	if count := session.server.protocolAttachmentCount("term-resource"); count != 1 {
		t.Fatalf("replacement attachment count = %d", count)
	}
	if err := session.ReleaseApplicationResource(context.Background(), replacement.Result().Token); err != nil {
		t.Fatal(err)
	}
	assertProtocolResourceCounts(t, session, 0, 0, 0)
}

func TestProtocolAttachmentRollbackCleansOnlyItsPublishedOwnership(t *testing.T) {
	server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 2, MaxAttachments: 2})
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	first, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
	if err != nil {
		t.Fatal(err)
	}
	second, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := second.Commit(context.Background()); err != nil {
		t.Fatalf("rollback removed replacement ownership: %v", err)
	}
	assertProtocolResourceCounts(t, session, 1, 0, 0)
	if err := session.ReleaseApplicationResource(context.Background(), second.Result().Token); err != nil {
		t.Fatal(err)
	}
	assertProtocolResourceCounts(t, session, 0, 0, 0)
}

func TestProtocolDuplicateInFlightRequestClosesPeerWithoutSecondExecution(t *testing.T) {
	executor := &blockingProtocolExecutor{started: make(chan struct{}), release: make(chan struct{})}
	server := NewServer(WithApplicationExecutorFactory(func(ApplicationSessionPort) ApplicationExecutor { return executor }))
	client, daemon := memory.NewPair()
	session := newProtocolSession(server, daemon, fullDaemonTransportScope())
	done := make(chan error, 1)
	go func() { done <- session.run(context.Background()) }()
	completeServerProtocolHello(t, client)
	commandPayload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}
	request := protocol.Request{ID: 11, Method: "api.execute", Params: commandPayload}
	sendProtocolRequest(t, client, request)
	select {
	case <-executor.started:
	case <-time.After(time.Second):
		t.Fatal("first request did not execute")
	}
	sendProtocolRequest(t, client, request)
	recvDone := make(chan error, 1)
	go func() {
		_, err := client.Recv()
		recvDone <- err
	}()
	select {
	case err := <-recvDone:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("peer close error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("duplicate request did not close peer")
	}
	if calls := executor.calls.Load(); calls != 1 {
		t.Fatalf("duplicate request executions = %d", calls)
	}
	close(executor.release)
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "duplicate in-flight") {
			t.Fatalf("session error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("duplicate request session did not stop")
	}
}

func TestProtocolRequestSlotsReuseAfterSuccessErrorAndCancellation(t *testing.T) {
	executor := &requestLifecycleExecutor{cancelStarted: make(chan struct{}, 1)}
	server := NewServer(
		WithProtocolSessionLimits(ProtocolSessionLimits{MaxInFlightRequests: 1}),
		WithApplicationExecutorFactory(func(ApplicationSessionPort) ApplicationExecutor { return executor }),
	)
	client, daemon := memory.NewPair()
	defer client.Close()
	defer daemon.Close()
	session := newProtocolSession(server, daemon, fullDaemonTransportScope())
	session.helloAccepted = true
	commandPayload, err := protocol.EncodeApplicationCommand(&apipb.CommandEnvelope{})
	if err != nil {
		t.Fatal(err)
	}

	handleProtocolRequestDirect(t, session, context.Background(), protocol.Request{ID: 1, Method: "api.execute", Params: commandPayload})
	if _, typ, _ := receiveProtocolFrame(t, client); typ != wire.TypeResponse {
		t.Fatalf("success response type = %d", typ)
	}
	waitForProtocolRequestIdle(t, session)

	handleProtocolRequestDirect(t, session, context.Background(), protocol.Request{ID: 2, Method: "missing"})
	if _, typ, _ := receiveProtocolFrame(t, client); typ != wire.TypeError {
		t.Fatalf("error response type = %d", typ)
	}
	waitForProtocolRequestIdle(t, session)

	executor.cancelNext.Store(true)
	cancelCtx, cancel := context.WithCancel(context.Background())
	handleProtocolRequestDirect(t, session, cancelCtx, protocol.Request{ID: 3, Method: "api.execute", Params: commandPayload})
	select {
	case <-executor.cancelStarted:
	case <-time.After(time.Second):
		t.Fatal("cancelled request did not enter executor")
	}
	cancel()
	if _, typ, _ := receiveProtocolFrame(t, client); typ != wire.TypeResponse {
		t.Fatalf("cancel response type = %d", typ)
	}
	waitForProtocolRequestIdle(t, session)

	handleProtocolRequestDirect(t, session, context.Background(), protocol.Request{ID: 4, Method: "api.execute", Params: commandPayload})
	if _, typ, _ := receiveProtocolFrame(t, client); typ != wire.TypeResponse {
		t.Fatalf("post-cancel response type = %d", typ)
	}
	waitForProtocolRequestIdle(t, session)
}

func TestProtocolResourceCountsReturnToZeroAcrossRealLifecycles(t *testing.T) {
	t.Run("attach failure", func(t *testing.T) {
		server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxAttachments: 1})
		session := newProtocolSession(server, nil, fullDaemonTransportScope())
		session.beforeGlobalAttachmentPublish = func(protocolAttachment) {
			_, _ = server.registry.remove("term-resource")
		}
		if _, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest()); !errors.Is(err, ErrTerminalNotFound) {
			t.Fatalf("attach failure = %v", err)
		}
		assertProtocolResourceCounts(t, session, 0, 0, 0)
		server.protocolAttachmentMu.Lock()
		globalAttachments := len(server.protocolAttachments)
		resizeOwners := len(server.protocolResizeOwners)
		server.protocolAttachmentMu.Unlock()
		if globalAttachments != 0 || resizeOwners != 0 {
			t.Fatalf("failed attach leaked global state attachments=%d owners=%d", globalAttachments, resizeOwners)
		}
	})

	t.Run("file failure completion and cancellation", func(t *testing.T) {
		server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 1, MaxFileTransfers: 1})
		client, daemon := memory.NewPair()
		defer client.Close()
		defer daemon.Close()
		session := newProtocolSession(server, daemon, fullDaemonTransportScope())
		missingTarget := filepath.Join(t.TempDir(), "missing", "upload.bin")
		if _, err := session.openFileUpload(FileUploadOpenRequest{Path: missingTarget, Size: 1, Overwrite: true}); err == nil {
			t.Fatal("file open unexpectedly succeeded")
		}
		assertProtocolResourceCounts(t, session, 0, 0, 0)

		completedTarget := filepath.Join(t.TempDir(), "completed.bin")
		completed, err := session.openFileUpload(FileUploadOpenRequest{Path: completedTarget, Size: 0, Overwrite: true})
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(nil)
		finish, err := protocol.EncodeFileTransferFinish(protocol.FileTransferFinish{Size: 0, SHA256: digest[:]})
		if err != nil {
			t.Fatal(err)
		}
		transfer := session.fileTransferForChannel(completed.Channel)
		if err := session.handleFileTransferFrame(context.Background(), transfer, wire.TypeFileFinish, finish); err != nil {
			t.Fatal(err)
		}
		assertProtocolResourceCounts(t, session, 0, 0, 0)
		if _, err := os.Stat(completedTarget); err != nil {
			t.Fatal(err)
		}

		cancelTarget := filepath.Join(t.TempDir(), "cancel.bin")
		cancelled, err := session.openFileUpload(FileUploadOpenRequest{Path: cancelTarget, Size: 1, Overwrite: true})
		if err != nil {
			t.Fatal(err)
		}
		if !session.cancelCurrentFileTransfer(cancelled.ID) {
			t.Fatal("file transfer cancellation failed")
		}
		assertProtocolResourceCounts(t, session, 0, 0, 0)
	})

	t.Run("session close", func(t *testing.T) {
		server := newProtocolResourceTestServer(t, ProtocolSessionLimits{MaxResources: 3, MaxAttachments: 1, MaxFileTransfers: 1, MaxEventSubscriptions: 1})
		client, daemon := memory.NewPair()
		session := newProtocolSession(server, daemon, fullDaemonTransportScope())
		attachment, err := session.ApplicationTerminalAttach(context.Background(), protocolResourceAttachmentRequest())
		if err != nil {
			t.Fatal(err)
		}
		if err := attachment.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := session.openFileUpload(FileUploadOpenRequest{Path: filepath.Join(t.TempDir(), "session.bin"), Size: 1, Overwrite: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := session.ApplicationEventSubscribe(context.Background(), EventFilter{}, func(Event, []byte) ([]byte, error) { return nil, nil }); err != nil {
			t.Fatal(err)
		}
		assertProtocolResourceCounts(t, session, 1, 1, 1)
		done := make(chan error, 1)
		go func() { done <- session.run(context.Background()) }()
		_ = client.Close()
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("session close error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("session close did not finish")
		}
		_ = daemon.Close()
		assertProtocolResourceCounts(t, session, 0, 0, 0)
	})
}

func TestProtocolSessionAggregateResourceLimitSpansKinds(t *testing.T) {
	server := NewServer(WithProtocolSessionLimits(ProtocolSessionLimits{
		MaxResources: 2, MaxAttachments: 2, MaxFileTransfers: 2, MaxEventSubscriptions: 2,
	}))
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	attachmentChannel, err := session.reserveAttachmentChannel()
	if err != nil {
		t.Fatal(err)
	}
	fileChannel, err := session.reserveFileChannel()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.reserveEventSubscription(); !errors.Is(err, ErrProtocolResourceExhausted) {
		t.Fatalf("aggregate resource exhaustion error = %v", err)
	}
	session.releaseChannel(attachmentChannel, protocolChannelAttachment)
	if err := session.reserveEventSubscription(); err != nil {
		t.Fatalf("released attachment did not restore aggregate capacity: %v", err)
	}
	session.releaseChannel(fileChannel, protocolChannelFileTransfer)
	session.releaseEventSubscription()
}

func TestProtocolSessionEventSubscriptionLimitReleasesCapacity(t *testing.T) {
	server := NewServer(WithProtocolSessionLimits(ProtocolSessionLimits{
		MaxResources: 1, MaxEventSubscriptions: 1,
	}))
	session := newProtocolSession(server, nil, fullDaemonTransportScope())
	encoder := func(Event, []byte) ([]byte, error) { return nil, nil }
	token, err := session.ApplicationEventSubscribe(context.Background(), EventFilter{}, encoder)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ApplicationEventSubscribe(context.Background(), EventFilter{}, encoder); !errors.Is(err, ErrProtocolResourceExhausted) {
		t.Fatalf("event subscription exhaustion error = %v", err)
	}
	if err := session.ReleaseApplicationResource(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := session.ApplicationEventSubscribe(context.Background(), EventFilter{}, encoder); err != nil {
		t.Fatalf("released subscription did not restore capacity: %v", err)
	}
	session.stopEvents()
}

type blockingProtocolExecutor struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func (executor *blockingProtocolExecutor) Execute(context.Context, *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	executor.calls.Add(1)
	executor.once.Do(func() { close(executor.started) })
	<-executor.release
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
}

type requestLifecycleExecutor struct {
	cancelNext    atomic.Bool
	cancelStarted chan struct{}
}

func (executor *requestLifecycleExecutor) Execute(ctx context.Context, _ *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	if executor.cancelNext.CompareAndSwap(true, false) {
		executor.cancelStarted <- struct{}{}
		<-ctx.Done()
	}
	return &apipb.ResultEnvelope{Result: &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}}
}

func newProtocolResourceTestServer(t *testing.T, limits ProtocolSessionLimits) *Server {
	t.Helper()
	server := NewServer(WithProcessFactory(newRecordingProcessFactory()), WithProtocolSessionLimits(limits))
	if _, err := server.RegisterTerminal(TerminalRecord{ID: "term-resource", Command: []string{"shell"}}); err != nil {
		t.Fatal(err)
	}
	return server
}

func protocolResourceAttachmentRequest() TerminalAttachmentRequest {
	return TerminalAttachmentRequest{
		TerminalID: "term-resource", Mode: TerminalAttachmentModeCollaborator, ResizePolicy: TerminalResizePolicyOwner,
		SurfaceID: "surface", ViewID: "view",
	}
}

func assertProtocolResourceCounts(t *testing.T, session *protocolSession, attachments, files, events int) {
	t.Helper()
	session.resourceMu.Lock()
	defer session.resourceMu.Unlock()
	if session.attachmentCount != attachments || session.fileTransferCount != files || session.eventSubscriptionCount != events {
		t.Fatalf("resource counts = attachments:%d files:%d events:%d, want %d/%d/%d", session.attachmentCount, session.fileTransferCount, session.eventSubscriptionCount, attachments, files, events)
	}
	if total := session.totalResourcesLocked(); total != attachments+files+events {
		t.Fatalf("total resource count = %d", total)
	}
}

func completeServerProtocolHello(t *testing.T, client *memory.Transport) {
	t.Helper()
	payload, err := protocol.EncodeHelloPayload(protocol.Hello{Version: wire.Version, Client: "reliability-test"})
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolFrame(t, client, wire.TypeHello, payload)
	if _, typ, _ := receiveProtocolFrame(t, client); typ != wire.TypeHello {
		t.Fatalf("hello response type = %d", typ)
	}
}

func handleProtocolRequestDirect(t *testing.T, session *protocolSession, ctx context.Context, request protocol.Request) {
	t.Helper()
	payload, err := protocol.EncodeRequestPayload(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.handleControlFrame(ctx, wire.TypeRequest, payload); err != nil {
		t.Fatal(err)
	}
}

func waitForProtocolRequestIdle(t *testing.T, session *protocolSession) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		session.requestMu.Lock()
		active := len(session.activeRequests)
		session.requestMu.Unlock()
		if len(session.requestSlots) == 0 && active == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("protocol request resources were not released")
}

func sendProtocolRequest(t *testing.T, connection *memory.Transport, request protocol.Request) {
	t.Helper()
	payload, err := protocol.EncodeRequestPayload(request)
	if err != nil {
		t.Fatal(err)
	}
	sendProtocolFrame(t, connection, wire.TypeRequest, payload)
}
