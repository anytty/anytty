package apilayer

import (
	"context"
	"errors"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
)

type fakeTerminalController struct {
	creates           []*apipb.TerminalCreateCommand
	onCreate          func(*apipb.EndpointSessionStamp, *apipb.TerminalCreateCommand)
	inputs            []*apipb.TerminalInputCommand
	inputErr          error
	createResult      *apipb.TerminalCreateResult
	returnNilCreate   bool
	attachTransaction TerminalAttachTransaction
	attachErr         error
}

func (*fakeTerminalController) TerminalDefaults(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
	return &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{DefaultCommand: []string{"sh"}, DefaultCwd: "/tmp"}}, nil
}

func (controller *fakeTerminalController) TerminalCreate(_ context.Context, session *apipb.EndpointSessionStamp, command *apipb.TerminalCreateCommand) (*apipb.TerminalCreateResult, error) {
	if controller.onCreate != nil {
		controller.onCreate(session, command)
	}
	controller.creates = append(controller.creates, command)
	if controller.returnNilCreate {
		return nil, nil
	}
	if controller.createResult != nil {
		return controller.createResult, nil
	}
	return &apipb.TerminalCreateResult{Terminal: &apipb.TerminalInfo{Ref: &apipb.TerminalRef{EndpointId: "studio", TerminalId: command.GetTerminal().GetTerminalId()}}}, nil
}

func (*fakeTerminalController) TerminalList(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalListCommand) (*apipb.TerminalListResult, error) {
	return &apipb.TerminalListResult{}, nil
}

func (*fakeTerminalController) TerminalGet(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalGetCommand) (*apipb.TerminalGetResult, error) {
	return &apipb.TerminalGetResult{}, nil
}

func (*fakeTerminalController) TerminalRestart(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalRestartCommand) error {
	return nil
}

func (*fakeTerminalController) TerminalKill(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalKillCommand) error {
	return nil
}

func (*fakeTerminalController) TerminalRemove(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalRemoveCommand) error {
	return nil
}

func (*fakeTerminalController) TerminalSetMetadata(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalSetMetadataCommand) error {
	return nil
}

func (*fakeTerminalController) TerminalSetTags(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalSetTagsCommand) error {
	return nil
}

func (controller *fakeTerminalController) TerminalAttach(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalAttachCommand) (TerminalAttachTransaction, error) {
	return controller.attachTransaction, controller.attachErr
}

type fakeAttachTransaction struct {
	result             *apipb.TerminalAttachResult
	commitErr          error
	rollbackErr        error
	onResult           func()
	committed          bool
	rolledBack         bool
	rollbackContextErr error
}

func (transaction *fakeAttachTransaction) Result() *apipb.TerminalAttachResult {
	if transaction.onResult != nil {
		transaction.onResult()
	}
	return transaction.result
}

func (transaction *fakeAttachTransaction) Commit(context.Context) error {
	transaction.committed = true
	return transaction.commitErr
}

func (transaction *fakeAttachTransaction) Rollback(ctx context.Context) error {
	transaction.rolledBack = true
	transaction.rollbackContextErr = ctx.Err()
	return transaction.rollbackErr
}

func (*fakeTerminalController) TerminalDetach(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalDetachCommand) error {
	return nil
}

func (controller *fakeTerminalController) TerminalInput(_ context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalInputCommand) error {
	controller.inputs = append(controller.inputs, command)
	return controller.inputErr
}

func (*fakeTerminalController) TerminalResize(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalResizeCommand) (*apipb.TerminalResizeResult, error) {
	return &apipb.TerminalResizeResult{}, nil
}

func (*fakeTerminalController) TerminalResizeLock(context.Context, *apipb.EndpointSessionStamp, *apipb.TerminalResizeLockCommand) (*apipb.TerminalResizeResult, error) {
	return &apipb.TerminalResizeResult{}, nil
}

func (*fakeTerminalController) PathListDirectories(context.Context, *apipb.EndpointSessionStamp, *apipb.PathListDirectoriesCommand) (*apipb.PathListDirectoriesResult, error) {
	return &apipb.PathListDirectoriesResult{}, nil
}

func TestServiceDispatchesValidatedTerminalCreateFromPrivateSnapshot(t *testing.T) {
	controller := &fakeTerminalController{}
	command := terminalCreateCommand()
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE), nil, nil, controller).Execute(context.Background(), command)
	if result.GetTerminalCreate().GetTerminal().GetRef().GetTerminalId() != "term-1" || len(controller.creates) != 1 {
		t.Fatalf("create result=%#v calls=%#v", result, controller.creates)
	}
	controller.creates[0].Terminal.Name = "controller-mutated"
	if command.GetTerminalCreate().GetTerminal().GetName() != "demo" {
		t.Fatal("terminal controller mutated client command")
	}
}

func TestServiceUsesOnePrivateEnvelopeSnapshotForAdmissionAndDispatch(t *testing.T) {
	command := terminalCreateCommand()
	var snapshotSession *apipb.EndpointSessionStamp
	var snapshotCommand *apipb.TerminalCreateCommand
	controller := &fakeTerminalController{onCreate: func(session *apipb.EndpointSessionStamp, command *apipb.TerminalCreateCommand) {
		if session != snapshotSession || command != snapshotCommand {
			t.Fatalf("dispatch cloned private snapshot fields: session=%p/%p command=%p/%p", session, snapshotSession, command, snapshotCommand)
		}
	}}
	admission := admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE)
	admission.authorize = func(snapshot *apipb.CommandEnvelope) error {
		if snapshot.GetTerminalCreate().GetTerminal().GetTerminalId() != "term-1" {
			t.Fatalf("admission received unexpected snapshot: %#v", snapshot)
		}
		snapshotSession = snapshot.GetContext().GetSession()
		snapshotCommand = snapshot.GetTerminalCreate()
		command.GetTerminalCreate().Terminal.TerminalId = "mutated-after-entry"
		return nil
	}
	result := NewService(admission, nil, nil, controller).Execute(context.Background(), command)
	if result.GetTerminalCreate().GetTerminal().GetRef().GetTerminalId() != "term-1" || controller.creates[0].GetTerminal().GetTerminalId() != "term-1" {
		t.Fatalf("caller mutation changed admitted command: result=%#v calls=%#v", result, controller.creates)
	}
}

func TestServiceRejectsOversizedEnvelopeBeforeCloneAndAdmission(t *testing.T) {
	command := terminalInputCommand()
	command.GetTerminalInput().Data = make([]byte, 3<<20)
	admission := admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_ATTACHMENT)
	result := NewService(admission, nil, nil, &fakeTerminalController{}).Execute(context.Background(), command)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST || admission.acquired != 0 {
		t.Fatalf("oversized envelope result=%#v admission=%d", result, admission.acquired)
	}
}

func TestServiceClonesControllerResultBeforeReturning(t *testing.T) {
	controllerResult := &apipb.TerminalCreateResult{Terminal: &apipb.TerminalInfo{Ref: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"}}}
	controller := &fakeTerminalController{createResult: controllerResult}
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE), nil, nil, controller).Execute(context.Background(), terminalCreateCommand())
	controllerResult.Terminal.Ref.TerminalId = "mutated-after-return"
	if result.GetTerminalCreate().GetTerminal().GetRef().GetTerminalId() != "term-1" {
		t.Fatalf("response aliases controller result: %#v", result)
	}
}

func TestServiceRejectsNilControllerResultWithTypedError(t *testing.T) {
	controller := &fakeTerminalController{returnNilCreate: true}
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE), nil, nil, controller).Execute(context.Background(), terminalCreateCommand())
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL || !result.GetError().GetAttempted() {
		t.Fatalf("nil controller result=%#v", result)
	}
}

func TestServiceRejectsInvalidAttachmentResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transaction := &fakeAttachTransaction{onResult: cancel, result: &apipb.TerminalAttachResult{
		Attachment: &apipb.AttachmentHandle{
			Resource:  &apipb.ResourceHandle{OpaqueToken: []byte("attachment"), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: cloneTestSession(), Generation: 1},
			Terminal:  &apipb.TerminalRef{EndpointId: "studio", TerminalId: "wrong-terminal"},
			Operation: &apipb.OperationStamp{Session: cloneTestSession(), OperationId: "attach-1"},
			SurfaceId: "surface", ViewId: "view",
		},
		Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}}
	controller := &fakeTerminalController{attachTransaction: transaction}
	command := terminalAttachCommand()
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_ATTACHMENT), nil, nil, controller).Execute(ctx, command)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL || !result.GetError().GetAttempted() || !transaction.rolledBack || transaction.committed || transaction.rollbackContextErr != nil {
		t.Fatalf("invalid attachment result=%#v transaction=%#v", result, transaction)
	}
}

func TestServiceCommitsValidatedAttachmentTransaction(t *testing.T) {
	transaction := &fakeAttachTransaction{result: validAttachResult()}
	controller := &fakeTerminalController{attachTransaction: transaction}
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_ATTACHMENT), nil, nil, controller).Execute(context.Background(), terminalAttachCommand())
	if result.GetTerminalAttach() == nil || !transaction.committed || transaction.rolledBack {
		t.Fatalf("attachment commit result=%#v transaction=%#v", result, transaction)
	}
}

func TestServiceRollsBackAttachmentTransactionReturnedWithError(t *testing.T) {
	transaction := &fakeAttachTransaction{result: validAttachResult()}
	controller := &fakeTerminalController{attachTransaction: transaction, attachErr: errors.New("attach failed")}
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_ATTACHMENT), nil, nil, controller).Execute(context.Background(), terminalAttachCommand())
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL || !transaction.rolledBack || transaction.committed {
		t.Fatalf("attachment error rollback result=%#v transaction=%#v", result, transaction)
	}
}

func TestServiceRejectsTerminalRefForDifferentEndpointBeforeController(t *testing.T) {
	controller := &fakeTerminalController{}
	command := &apipb.CommandEnvelope{Context: terminalAPIContext("request-get"), Command: &apipb.CommandEnvelope_TerminalGet{TerminalGet: &apipb.TerminalGetCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "other", TerminalId: "term-1"},
	}}}
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE), nil, nil, controller).Execute(context.Background(), command)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST || result.GetError().GetAttempted() {
		t.Fatalf("endpoint mismatch result=%#v", result)
	}
}

func TestServiceMarksTerminalInputControllerFailureAttempted(t *testing.T) {
	controller := &fakeTerminalController{inputErr: errors.New("input write failed")}
	command := terminalInputCommand()
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_TERMINAL_ATTACHMENT), nil, nil, controller).Execute(context.Background(), command)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL || !result.GetError().GetAttempted() || len(controller.inputs) != 1 {
		t.Fatalf("input result=%#v calls=%d", result, len(controller.inputs))
	}
}

func terminalCreateCommand() *apipb.CommandEnvelope {
	return &apipb.CommandEnvelope{Context: terminalAPIContext("request-create"), Command: &apipb.CommandEnvelope_TerminalCreate{TerminalCreate: &apipb.TerminalCreateCommand{
		Terminal: &apipb.TerminalCreateSpec{TerminalId: "term-1", Name: "demo", Command: []string{"sh"}, Size: &apipb.TerminalSize{Cols: 80, Rows: 24}},
	}}}
}

func terminalInputCommand() *apipb.CommandEnvelope {
	contextMessage := terminalAPIContext("request-input")
	return &apipb.CommandEnvelope{Context: contextMessage, Command: &apipb.CommandEnvelope_TerminalInput{TerminalInput: &apipb.TerminalInputCommand{
		Attachment: &apipb.ResourceHandle{OpaqueToken: []byte("attachment-1"), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: cloneTestSession(), Generation: 2},
		Operation:  &apipb.OperationStamp{Session: contextMessage.GetSession(), OperationId: "input-1"},
		Data:       []byte("x"),
	}}}
}

func terminalAttachCommand() *apipb.CommandEnvelope {
	contextMessage := terminalAPIContext("request-attach")
	return &apipb.CommandEnvelope{Context: contextMessage, Command: &apipb.CommandEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"},
		Mode:     apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		SurfaceId: "surface", ViewId: "view",
		Operation: &apipb.OperationStamp{Session: cloneTestSession(), OperationId: "attach-1"},
	}}}
}

func validAttachResult() *apipb.TerminalAttachResult {
	return &apipb.TerminalAttachResult{
		Attachment: &apipb.AttachmentHandle{
			Resource:  &apipb.ResourceHandle{OpaqueToken: []byte("attachment"), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: cloneTestSession(), Generation: 1},
			Terminal:  &apipb.TerminalRef{EndpointId: "studio", TerminalId: "term-1"},
			Operation: &apipb.OperationStamp{Session: cloneTestSession(), OperationId: "attach-1"},
			SurfaceId: "surface", ViewId: "view",
		},
		Mode: apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR, ResizePolicy: apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		Size: &apipb.TerminalSize{Cols: 80, Rows: 24},
	}
}

func terminalAPIContext(requestID string) *apipb.RequestContext {
	return &apipb.RequestContext{
		RequestId: requestID, ApiVersion: &apipb.ApiVersion{Major: 1}, Session: cloneTestSession(),
	}
}

func cloneTestSession() *apipb.EndpointSessionStamp {
	return &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 7}
}
