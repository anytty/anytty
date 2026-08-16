package apilayer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	clientendpoint "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/transport/memory"
)

func TestGeneratedEventSubscriptionsCorrelateAndRelease(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory))
	defer func() { _ = server.Shutdown(context.Background()) }()
	application, _, closeClient := newProtoTransportClient(t, server, nil, 1)
	defer closeClient()

	command := &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	}}
	first, firstEvents, err := application.EventSubscribe(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	second, secondEvents, err := application.EventSubscribe(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.GetSubscription().GetKind() != apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION ||
		second.GetSubscription().GetKind() != apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION ||
		bytes.Equal(first.GetSubscription().GetOpaqueToken(), second.GetSubscription().GetOpaqueToken()) {
		t.Fatalf("subscriptions are not independently correlated: first=%#v second=%#v", first, second)
	}

	createProtoTestTerminal(t, application, "term-event-first")
	assertProtoSubscriptionEvent(t, firstEvents, first.GetSubscription(), "term-event-first")
	assertProtoSubscriptionEvent(t, secondEvents, second.GetSubscription(), "term-event-first")
	if _, err := application.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{
		ReleaseResource: &apipb.ReleaseResourceCommand{Resource: first.GetSubscription()},
	}}); err != nil {
		t.Fatal(err)
	}
	createProtoTestTerminal(t, application, "term-event-second")
	assertProtoSubscriptionEvent(t, secondEvents, second.GetSubscription(), "term-event-second")
	select {
	case event := <-firstEvents:
		t.Fatalf("released subscription received event %#v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestGeneratedMachineEventsScopeAllowsOnlyLifecycleSubscription(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithClientAccessService(&protoGrantAccessService{grants: map[string]time.Time{"grant-events": expiresAt}}))
	defer func() { _ = server.Shutdown(context.Background()) }()
	scope := corev2.TransportScope{GrantID: "grant-events", GrantExpiresAt: expiresAt, PrincipalID: "machine-observer", MachineEventsOnly: true}
	observer, _, closeObserver := newProtoTransportClient(t, server, &scope, 1)
	defer closeObserver()
	owner, _, closeOwner := newProtoTransportClient(t, server, nil, 2)
	defer closeOwner()

	subscription, events, err := observer.EventSubscribe(context.Background(), &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	}})
	if err != nil {
		t.Fatal(err)
	}
	createProtoTestTerminal(t, owner, "term-machine-event")
	assertProtoSubscriptionEvent(t, events, subscription.GetSubscription(), "term-machine-event")
	if _, err := observer.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{
		ReleaseResource: &apipb.ReleaseResourceCommand{Resource: subscription.GetSubscription()},
	}}); err != nil {
		t.Fatalf("machine-events-only subscription release failed: %v", err)
	}
	if _, err := observer.TerminalList(context.Background(), &apipb.TerminalListCommand{}); clientruntime.CodeOf(err) != clientruntime.ErrorAuthorization {
		t.Fatalf("machine-events-only terminal list error = %v", err)
	}
	if _, _, err := observer.EventSubscribe(context.Background(), &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_STORAGE_CHANGED,
	}}); clientruntime.CodeOf(err) != clientruntime.ErrorAuthorization {
		t.Fatalf("machine-events-only storage subscription error = %v", err)
	}
}

func TestGeneratedTerminalScopeListsOnlyAuthorizedTerminal(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithClientAccessService(&protoGrantAccessService{grants: map[string]time.Time{"grant-terminal": expiresAt}}))
	defer func() { _ = server.Shutdown(context.Background()) }()
	owner, _, closeOwner := newProtoTransportClient(t, server, nil, 1)
	defer closeOwner()
	createProtoTestTerminal(t, owner, "term-scoped-visible")
	createProtoTestTerminal(t, owner, "term-scoped-hidden")

	scope := corev2.TransportScope{GrantID: "grant-terminal", GrantExpiresAt: expiresAt, PrincipalID: "terminal-client", TerminalID: "term-scoped-visible"}
	application, _, closeApplication := newProtoTransportClient(t, server, &scope, 2)
	defer closeApplication()
	listed, err := application.TerminalList(context.Background(), &apipb.TerminalListCommand{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.GetTerminals()) != 1 || listed.GetTerminals()[0].GetRef().GetTerminalId() != "term-scoped-visible" {
		t.Fatalf("terminal-scoped list = %#v", listed.GetTerminals())
	}
	if _, _, err := application.EventSubscribe(context.Background(), &apipb.EventSubscribeCommand{
		Types: []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE},
	}); clientruntime.CodeOf(err) != clientruntime.ErrorAuthorization {
		t.Fatalf("unfiltered terminal-scoped subscription error = %v", err)
	}
	if _, _, err := application.EventSubscribe(context.Background(), &apipb.EventSubscribeCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-scoped-visible"},
		Types:    []apipb.ApplicationEventType{apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE},
	}); err != nil {
		t.Fatalf("filtered terminal-scoped subscription failed: %v", err)
	}
	if _, err := application.TerminalAttach(context.Background(), &apipb.TerminalAttachCommand{
		Terminal:     &apipb.TerminalRef{EndpointId: "local", TerminalId: "term-scoped-visible"},
		Mode:         apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR,
		ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId:    "terminal-scoped-surface",
		ViewId:       "terminal-scoped-view",
	}); err != nil {
		t.Fatalf("terminal-scoped attach failed: %v", err)
	}
}

func TestClientAccessRevokeClosesEstablishedTerminalAndFileTransports(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	service := &protoGrantAccessService{grants: map[string]time.Time{"grant-business": expiresAt}, revoked: make(map[string]bool)}
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory), corev2.WithClientAccessService(service))
	defer func() { _ = server.Shutdown(context.Background()) }()
	owner, _, closeOwner := newProtoTransportClient(t, server, nil, 1)
	defer closeOwner()
	createProtoTestTerminal(t, owner, "term-business")
	scope := corev2.TransportScope{
		GrantID: "grant-business", GrantExpiresAt: expiresAt, PrincipalID: "subject", AllowDaemon: true,
		FileReadMetadata: true, FileReadContent: true, FileWriteContent: true, FileMutate: true,
	}
	terminalApplication, _, closeTerminal := newProtoTransportClient(t, server, &scope, 2)
	defer closeTerminal()
	fileApplication, _, closeFile := newProtoTransportClient(t, server, &scope, 3)
	defer closeFile()

	if _, err := terminalApplication.TerminalList(context.Background(), &apipb.TerminalListCommand{}); err != nil {
		t.Fatalf("terminal transport before revoke: %v", err)
	}
	path := t.TempDir()
	if _, err := fileApplication.FileStat(context.Background(), &apipb.FileStatCommand{Path: path}); err != nil {
		t.Fatalf("file transport before revoke: %v", err)
	}
	revoked, err := owner.ClientAccessRevoke(context.Background(), &apipb.ClientAccessRevokeCommand{
		Request: &remoteauthpb.ClientAccessRevokeRequest{GrantId: "grant-business"},
	})
	if err != nil || revoked.GetRecord().GetGrantId() != "grant-business" {
		t.Fatalf("revoke result = %#v, error = %v", revoked, err)
	}
	if _, err := terminalApplication.TerminalList(context.Background(), &apipb.TerminalListCommand{}); err == nil {
		t.Fatal("terminal transport survived revoke")
	}
	if _, err := fileApplication.FileStat(context.Background(), &apipb.FileStatCommand{Path: path}); err == nil {
		t.Fatal("file transport survived revoke")
	}
}

func TestGeneratedFileUploadSeparatesActiveAndResumeTokenNamespaces(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory))
	defer func() { _ = server.Shutdown(context.Background()) }()
	application, client, closeClient := newProtoTransportClient(t, server, nil, 1)
	defer closeClient()
	target := filepath.Join(t.TempDir(), "namespace.bin")

	opened, err := application.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{Path: target, Size: 3, Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	transfer := opened.GetTransfer()
	resourceToken := transfer.GetResource().GetOpaqueToken()
	resumeToken := transfer.GetResume().GetOpaqueToken()
	if transfer.GetResource().GetKind() != apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER || len(resourceToken) == 0 || len(resumeToken) == 0 || bytes.Equal(resourceToken, resumeToken) {
		t.Fatalf("file upload token namespaces collapsed: transfer=%#v", transfer)
	}
	if _, ok := client.ApplicationResourceChannel(transfer.GetResource()); !ok {
		t.Fatal("active file resource was not bound to its protocol stream")
	}
	if _, err := application.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{
		Path: target, Size: 3, Overwrite: true, Resume: &apipb.FileUploadResumeHandle{OpaqueToken: resourceToken},
	}); clientruntime.CodeOf(err) != clientruntime.ErrorInvalidRequest {
		t.Fatalf("active resource token was accepted as resume credential: %v", err)
	}
}

func TestGeneratedFileUploadResumesAcrossProtocolSessions(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory))
	defer func() { _ = server.Shutdown(context.Background()) }()
	target := filepath.Join(t.TempDir(), "resumed.bin")
	content := bytes.Repeat([]byte("generated-proto-resume-"), 3000)

	firstApplication, firstClient, closeFirst := newProtoTransportClient(t, server, nil, 1)
	opened, err := firstApplication.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{Path: target, Size: int64(len(content)), Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	firstTransfer := opened.GetTransfer()
	firstChannel, ok := firstClient.ApplicationResourceChannel(firstTransfer.GetResource())
	if !ok {
		t.Fatal("first upload resource has no stream channel")
	}
	firstStream, stopFirstStream := firstClient.Stream(firstChannel)
	firstChunkSize := min(int(firstTransfer.GetChunkBytes()), len(content))
	sendProtoUploadData(t, firstClient, firstChannel, 0, content[:firstChunkSize])
	if ack := waitProtoUploadAck(t, firstStream); ack.Offset != int64(firstChunkSize) {
		t.Fatalf("first upload ack offset=%d want=%d", ack.Offset, firstChunkSize)
	}
	stopFirstStream()
	closeFirst()

	secondApplication, secondClient, closeSecond := newProtoTransportClient(t, server, nil, 2)
	defer closeSecond()
	resumed, err := secondApplication.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{
		Path: target, Size: int64(len(content)), Overwrite: true, Resume: firstTransfer.GetResume(),
	})
	if err != nil {
		t.Fatal(err)
	}
	secondTransfer := resumed.GetTransfer()
	if secondTransfer.GetOffset() != int64(firstChunkSize) {
		t.Fatalf("resume offset=%d want=%d", secondTransfer.GetOffset(), firstChunkSize)
	}
	if !bytes.Equal(secondTransfer.GetResume().GetOpaqueToken(), firstTransfer.GetResume().GetOpaqueToken()) || secondTransfer.GetResource().GetSession().GetGeneration() != 2 {
		t.Fatalf("resumed transfer was not rebound correctly: %#v", secondTransfer)
	}
	secondChannel, ok := secondClient.ApplicationResourceChannel(secondTransfer.GetResource())
	if !ok {
		t.Fatal("resumed upload resource has no stream channel")
	}
	stream, stopStream := secondClient.Stream(secondChannel)
	defer stopStream()
	for offset := firstChunkSize; offset < len(content); {
		end := min(offset+int(secondTransfer.GetChunkBytes()), len(content))
		sendProtoUploadData(t, secondClient, secondChannel, int64(offset), content[offset:end])
		if ack := waitProtoUploadAck(t, stream); ack.Offset != int64(end) {
			t.Fatalf("resumed upload ack offset=%d want=%d", ack.Offset, end)
		}
		offset = end
	}
	digest := sha256.Sum256(content)
	finish, err := protocol.EncodeFileTransferFinish(protocol.FileTransferFinish{Size: int64(len(content)), SHA256: digest[:]})
	if err != nil {
		t.Fatal(err)
	}
	if err := secondClient.SendFileFrame(secondChannel, wire.TypeFileFinish, finish); err != nil {
		t.Fatal(err)
	}
	if frame := waitProtoFileFrame(t, stream); frame.Type != wire.TypeFileResult {
		t.Fatalf("upload completion frame type=%d want=%d", frame.Type, wire.TypeFileResult)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("resumed upload content mismatch: bytes=%d err=%v", len(got), err)
	}
}

func TestGeneratedFileUploadCancelUsesPrincipalBoundResumeAcrossSessions(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory))
	defer func() { _ = server.Shutdown(context.Background()) }()
	target := filepath.Join(t.TempDir(), "cancel-resume.bin")
	firstApplication, _, closeFirst := newProtoTransportClient(t, server, nil, 1)
	defer closeFirst()
	opened, err := firstApplication.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{Path: target, Size: 8, Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	secondApplication, _, closeSecond := newProtoTransportClient(t, server, nil, 2)
	defer closeSecond()
	forgedResource := &apipb.ResourceHandle{
		OpaqueToken: append([]byte(nil), opened.GetTransfer().GetResource().GetOpaqueToken()...),
		Kind:        apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER,
		Session:     &apipb.EndpointSessionStamp{EndpointId: "local", RouteId: "memory", Generation: 2},
		Generation:  opened.GetTransfer().GetResource().GetGeneration(),
	}
	forged, err := secondApplication.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileTransferCancel{FileTransferCancel: &apipb.FileTransferCancelCommand{Transfer: forgedResource}}})
	if err != nil {
		t.Fatal(err)
	}
	if forged.GetFileTransferCancel().GetCancelled() {
		t.Fatal("stale resource token was accepted after its session stamp was forged")
	}
	cancelled, err := secondApplication.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_FileTransferCancel{FileTransferCancel: &apipb.FileTransferCancelCommand{UploadResume: opened.GetTransfer().GetResume()}}})
	if err != nil {
		t.Fatal(err)
	}
	if !cancelled.GetFileTransferCancel().GetCancelled() {
		t.Fatalf("resume cancellation was not confirmed: %#v", cancelled)
	}
	if _, err := secondApplication.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{Path: target, Size: 8, Overwrite: true, Resume: opened.GetTransfer().GetResume()}); err == nil {
		t.Fatal("cancelled upload resume credential remained usable")
	}
}

func TestGeneratedFileUploadResumesAfterResourceReleaseOnSharedSession(t *testing.T) {
	server := corev2.NewServer(corev2.WithApplicationExecutorFactory(CoreApplicationExecutorFactory))
	defer func() { _ = server.Shutdown(context.Background()) }()
	target := filepath.Join(t.TempDir(), "shared-session-resume.bin")
	content := bytes.Repeat([]byte("shared-session-resume-"), 128)
	application, client, closeClient := newProtoTransportClient(t, server, nil, 1)
	defer closeClient()
	opened, err := application.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{Path: target, Size: int64(len(content)), Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	first := opened.GetTransfer()
	channel, ok := client.ApplicationResourceChannel(first.GetResource())
	if !ok {
		t.Fatal("first upload resource has no stream channel")
	}
	stream, stop := client.Stream(channel)
	chunkSize := min(int(first.GetChunkBytes()), len(content))
	sendProtoUploadData(t, client, channel, 0, content[:chunkSize])
	if ack := waitProtoUploadAck(t, stream); ack.Offset != int64(chunkSize) {
		t.Fatalf("first upload ack offset=%d want=%d", ack.Offset, chunkSize)
	}
	stop()
	if _, err := application.Execute(context.Background(), &apipb.CommandEnvelope{Command: &apipb.CommandEnvelope_ReleaseResource{ReleaseResource: &apipb.ReleaseResourceCommand{Resource: first.GetResource()}}}); err != nil {
		t.Fatal(err)
	}
	resumed, err := application.FileUploadOpen(context.Background(), &apipb.FileUploadOpenCommand{
		Path: target, Size: int64(len(content)), Overwrite: true, Resume: first.GetResume(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.GetTransfer().GetOffset() != int64(chunkSize) || resumed.GetTransfer().GetResource().GetSession().GetGeneration() != 1 {
		t.Fatalf("shared-session resume = %#v", resumed.GetTransfer())
	}
}

func newProtoTransportClient(t *testing.T, server *corev2.Server, scope *corev2.TransportScope, generation uint64) (*clientruntime.ApplicationSession, *protocol.Client, func()) {
	t.Helper()
	clientTransport, serverTransport := memory.NewPair()
	errCh := make(chan error, 1)
	go func() {
		if scope == nil {
			errCh <- server.ServeTransport(context.Background(), serverTransport)
			return
		}
		errCh <- server.ServeScopedTransport(context.Background(), serverTransport, *scope)
	}()
	client := protocol.NewClient(clientTransport)
	if err := client.Hello(context.Background(), protocol.Hello{Version: wire.Version, Client: "proto-transport-test"}); err != nil {
		t.Fatal(err)
	}
	application, err := clientruntime.NewApplicationSession(clientruntime.EndpointSessionStamp{
		EndpointID: clientendpoint.EndpointID("local"), RouteID: clientendpoint.RouteID("memory"), Generation: clientruntime.SessionGeneration(generation),
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	return application, client, func() {
		_ = client.Close()
		select {
		case err := <-errCh:
			if err != nil && !strings.Contains(err.Error(), "EOF") {
				t.Fatalf("server transport returned error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("server transport did not stop")
		}
	}
}

func createProtoTestTerminal(t *testing.T, application *clientruntime.ApplicationSession, terminalID string) {
	t.Helper()
	if _, err := application.TerminalCreate(context.Background(), &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: terminalID, Command: testIdleTerminalCommand(), Size: &apipb.TerminalSize{Cols: 12, Rows: 4},
	}}); err != nil {
		t.Fatal(err)
	}
}

func assertProtoSubscriptionEvent(t *testing.T, events <-chan *apipb.EventEnvelope, subscription *apipb.ResourceHandle, terminalID string) {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("application event stream closed")
		}
		if !bytes.Equal(event.GetSubscription().GetOpaqueToken(), subscription.GetOpaqueToken()) || event.GetTerminalLifecycle().GetTerminal().GetRef().GetTerminalId() != terminalID {
			t.Fatalf("event correlation mismatch: event=%#v subscription=%#v", event, subscription)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for terminal lifecycle event %q", terminalID)
	}
}

func sendProtoUploadData(t *testing.T, client *protocol.Client, channel uint16, offset int64, data []byte) {
	t.Helper()
	payload, err := protocol.EncodeFileTransferData(protocol.FileTransferData{Offset: offset, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SendFileFrame(channel, wire.TypeFileData, payload); err != nil {
		t.Fatal(err)
	}
}

func waitProtoUploadAck(t *testing.T, stream <-chan protocol.StreamFrame) protocol.FileTransferAck {
	t.Helper()
	frame := waitProtoFileFrame(t, stream)
	if frame.Type != wire.TypeFileAck {
		t.Fatalf("file stream frame type=%d want=%d", frame.Type, wire.TypeFileAck)
	}
	ack, err := protocol.DecodeFileTransferAck(frame.Payload)
	if err != nil {
		t.Fatal(err)
	}
	return ack
}

func waitProtoFileFrame(t *testing.T, stream <-chan protocol.StreamFrame) protocol.StreamFrame {
	t.Helper()
	select {
	case frame, ok := <-stream:
		if !ok {
			t.Fatal("file stream closed")
		}
		return frame
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for file stream")
		return protocol.StreamFrame{}
	}
}

type protoGrantAccessService struct {
	mu      sync.Mutex
	grants  map[string]time.Time
	revoked map[string]bool
}

func (*protoGrantAccessService) Identity(context.Context, []byte) (corev2.ClientAccessIdentity, error) {
	return corev2.ClientAccessIdentity{}, nil
}

func (*protoGrantAccessService) CreateTicket(context.Context, corev2.ClientAccessTicketRequest) (corev2.ClientAccessTicket, error) {
	return corev2.ClientAccessTicket{}, nil
}

func (*protoGrantAccessService) List(context.Context) ([]corev2.ClientAccessRecord, error) {
	return nil, nil
}

func (service *protoGrantAccessService) GrantActive(_ context.Context, grantID string, expiresAt, now time.Time) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	stored, ok := service.grants[grantID]
	return ok && !service.revoked[grantID] && stored.Equal(expiresAt) && now.Before(stored)
}

func (service *protoGrantAccessService) Revoke(_ context.Context, grantID string) (corev2.ClientAccessRecord, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	expiresAt, ok := service.grants[grantID]
	if !ok {
		return corev2.ClientAccessRecord{}, errors.New("client access grant not found")
	}
	if service.revoked == nil {
		service.revoked = make(map[string]bool)
	}
	service.revoked[grantID] = true
	return corev2.ClientAccessRecord{GrantID: grantID, ExpiresAt: expiresAt, RevokedAt: time.Now().UTC()}, nil
}
