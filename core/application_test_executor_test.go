package core

import (
	"context"
	"errors"
	"time"

	"github.com/anytty/anytty/proto/apipb"
)

type applicationTestExecutor struct {
	port ApplicationSessionPort
}

func applicationTestExecutorFactory(port ApplicationSessionPort) ApplicationExecutor {
	return &applicationTestExecutor{port: port}
}

func (executor *applicationTestExecutor) Execute(ctx context.Context, command *apipb.CommandEnvelope) *apipb.ResultEnvelope {
	request := command.GetContext()
	result := &apipb.ResultEnvelope{RequestId: request.GetRequestId(), OriginSession: cloneApplicationTestSession(request.GetSession())}
	lease, err := executor.port.AcquireApplication(ctx, applicationTestAdmission(command))
	if err != nil {
		result.Result = &apipb.ResultEnvelope_Error{Error: applicationTestError(err)}
		return result
	}
	defer lease.Release()

	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDefaults:
		defaults, err := executor.port.ApplicationTerminalDefaults(ctx)
		if err != nil {
			return applicationTestFailure(result, err)
		}
		result.Result = &apipb.ResultEnvelope_TerminalDefaults{TerminalDefaults: &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{DefaultCommand: defaults.DefaultCommand, DefaultCwd: defaults.DefaultCWD, Platform: defaults.Platform}}}
	case *apipb.CommandEnvelope_TerminalCreate:
		spec := value.TerminalCreate.GetTerminal()
		info, err := executor.port.ApplicationTerminalCreate(ctx, TerminalRecord{ID: spec.GetTerminalId(), Name: spec.GetName(), Command: append([]string(nil), spec.GetCommand()...), Tags: cloneStringMap(spec.GetTags()), Size: Size{Cols: uint16(spec.GetSize().GetCols()), Rows: uint16(spec.GetSize().GetRows())}, Options: TerminalCreateOptions{Dir: spec.GetCwd(), Env: append([]string(nil), spec.GetEnv()...), ScrollbackSize: int(spec.GetScrollbackRows()), ScrollbackMaxBytes: spec.GetScrollbackMaxBytes(), ScrollbackMaxAge: time.Duration(spec.GetScrollbackMaxAgeSeconds()) * time.Second}})
		if err != nil {
			return applicationTestFailure(result, err)
		}
		result.Result = &apipb.ResultEnvelope_TerminalCreate{TerminalCreate: &apipb.TerminalCreateResult{Terminal: applicationTestTerminalInfo(request.GetSession().GetEndpointId(), info, executor.port.ApplicationTerminalAttachmentCount(info.ID))}}
	case *apipb.CommandEnvelope_TerminalList:
		items, err := executor.port.ApplicationTerminalList(ctx)
		if err != nil {
			return applicationTestFailure(result, err)
		}
		listed := &apipb.TerminalListResult{Terminals: make([]*apipb.TerminalInfo, 0, len(items))}
		for _, item := range items {
			listed.Terminals = append(listed.Terminals, applicationTestTerminalInfo(request.GetSession().GetEndpointId(), item, executor.port.ApplicationTerminalAttachmentCount(item.ID)))
		}
		result.Result = &apipb.ResultEnvelope_TerminalList{TerminalList: listed}
	case *apipb.CommandEnvelope_TerminalGet:
		info, err := executor.port.ApplicationTerminalGet(ctx, value.TerminalGet.GetTerminal().GetTerminalId())
		if err != nil {
			return applicationTestFailure(result, err)
		}
		result.Result = &apipb.ResultEnvelope_TerminalGet{TerminalGet: &apipb.TerminalGetResult{Terminal: applicationTestTerminalInfo(request.GetSession().GetEndpointId(), info, executor.port.ApplicationTerminalAttachmentCount(info.ID))}}
	case *apipb.CommandEnvelope_TerminalRestart:
		return applicationTestAck(result, executor.port.ApplicationTerminalRestart(ctx, value.TerminalRestart.GetTerminal().GetTerminalId()))
	case *apipb.CommandEnvelope_TerminalKill:
		return applicationTestAck(result, executor.port.ApplicationTerminalKill(ctx, value.TerminalKill.GetTerminal().GetTerminalId()))
	case *apipb.CommandEnvelope_TerminalRemove:
		return applicationTestAck(result, executor.port.ApplicationTerminalRemove(ctx, value.TerminalRemove.GetTerminal().GetTerminalId()))
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		return applicationTestAck(result, executor.port.ApplicationTerminalSetMetadata(ctx, value.TerminalSetMetadata.GetTerminal().GetTerminalId(), value.TerminalSetMetadata.GetName(), value.TerminalSetMetadata.GetTags()))
	case *apipb.CommandEnvelope_TerminalSetTags:
		return applicationTestAck(result, executor.port.ApplicationTerminalSetTags(ctx, value.TerminalSetTags.GetTerminal().GetTerminalId(), value.TerminalSetTags.GetTags()))
	case *apipb.CommandEnvelope_TerminalAttach:
		attach := value.TerminalAttach
		transaction, err := executor.port.ApplicationTerminalAttach(ctx, TerminalAttachmentRequest{TerminalID: attach.GetTerminal().GetTerminalId(), Mode: applicationTestAttachmentMode(attach.GetMode()), ResizePolicy: applicationTestResizePolicy(attach.GetResizePolicy()), SurfaceID: attach.GetSurfaceId(), ViewID: attach.GetViewId()})
		if err != nil {
			return applicationTestFailure(result, err)
		}
		attachment := transaction.Result()
		if err := transaction.Commit(ctx); err != nil {
			_ = transaction.Rollback(ctx)
			return applicationTestFailure(result, err)
		}
		result.Result = &apipb.ResultEnvelope_TerminalAttach{TerminalAttach: &apipb.TerminalAttachResult{Attachment: &apipb.AttachmentHandle{Resource: &apipb.ResourceHandle{OpaqueToken: attachment.Token, Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: cloneApplicationTestSession(request.GetSession()), Generation: 1}, Terminal: attach.GetTerminal(), Operation: attach.GetOperation(), SurfaceId: attach.GetSurfaceId(), ViewId: attach.GetViewId()}, Mode: attach.GetMode(), ResizePolicy: attach.GetResizePolicy(), Size: applicationTestSize(attachment.Size), ResizeControl: applicationTestResizeControl(attachment.ResizeControl)}}
	case *apipb.CommandEnvelope_TerminalDetach:
		return applicationTestAck(result, executor.port.ApplicationTerminalDetach(ctx, value.TerminalDetach.GetAttachment().GetOpaqueToken()))
	case *apipb.CommandEnvelope_TerminalInput:
		return applicationTestAck(result, executor.port.ApplicationTerminalInput(ctx, value.TerminalInput.GetAttachment().GetOpaqueToken(), value.TerminalInput.GetData()))
	case *apipb.CommandEnvelope_TerminalResize:
		resized, err := executor.port.ApplicationTerminalResize(ctx, value.TerminalResize.GetAttachment().GetOpaqueToken(), Size{Cols: uint16(value.TerminalResize.GetSize().GetCols()), Rows: uint16(value.TerminalResize.GetSize().GetRows())}, applicationTestResizePolicy(value.TerminalResize.GetResizePolicy()), value.TerminalResize.GetTakeOwnership(), value.TerminalResize.GetExpectedOwnerEpoch())
		if err != nil {
			return applicationTestFailure(result, err)
		}
		result.Result = &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{Size: applicationTestSize(resized.Size), Resized: resized.Resized, ResizeControl: applicationTestResizeControl(resized.ResizeControl)}}
	case *apipb.CommandEnvelope_TerminalResizeLock:
		resized, err := executor.port.ApplicationTerminalResizeLock(ctx, value.TerminalResizeLock.GetAttachment().GetOpaqueToken(), value.TerminalResizeLock.GetLocked())
		if err != nil {
			return applicationTestFailure(result, err)
		}
		result.Result = &apipb.ResultEnvelope_TerminalResize{TerminalResize: &apipb.TerminalResizeResult{Size: applicationTestSize(resized.Size), Resized: resized.Resized, ResizeControl: applicationTestResizeControl(resized.ResizeControl)}}
	case *apipb.CommandEnvelope_PathListDirectories:
		paths, err := executor.port.ApplicationPathListDirectories(ctx, value.PathListDirectories.GetPrefix(), int(value.PathListDirectories.GetLimit()))
		if err != nil {
			return applicationTestFailure(result, err)
		}
		mapped := &apipb.PathListDirectoriesResult{BasePath: paths.BasePath, Missing: paths.Missing, Truncated: paths.Truncated}
		for _, entry := range paths.Entries {
			mapped.Entries = append(mapped.Entries, &apipb.PathDirectoryEntry{Name: entry.Name, Path: entry.Path})
		}
		result.Result = &apipb.ResultEnvelope_PathListDirectories{PathListDirectories: mapped}
	case *apipb.CommandEnvelope_ReleaseResource:
		return applicationTestAck(result, executor.port.ReleaseApplicationResource(ctx, value.ReleaseResource.GetResource().GetOpaqueToken()))
	default:
		return applicationTestFailure(result, ErrApplicationUnsupportedCapability)
	}
	return result
}

func applicationTestAdmission(command *apipb.CommandEnvelope) ApplicationAdmission {
	admission := ApplicationAdmission{Capability: ApplicationCapabilityTerminalLifecycle}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalList:
		admission.Capability = ApplicationCapabilityTerminalInventory
	case *apipb.CommandEnvelope_TerminalGet:
		admission.TerminalID = value.TerminalGet.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalRestart:
		admission.TerminalID = value.TerminalRestart.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalKill:
		admission.TerminalID = value.TerminalKill.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalRemove:
		admission.TerminalID = value.TerminalRemove.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		admission.TerminalID = value.TerminalSetMetadata.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalSetTags:
		admission.TerminalID = value.TerminalSetTags.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalAttach:
		admission.Capability, admission.TerminalID = ApplicationCapabilityTerminalAttachment, value.TerminalAttach.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalDetach:
		admission.Capability, admission.ResourceToken = ApplicationCapabilityTerminalAttachment, value.TerminalDetach.GetAttachment().GetOpaqueToken()
	case *apipb.CommandEnvelope_TerminalInput:
		admission.Capability, admission.ResourceToken = ApplicationCapabilityTerminalAttachment, value.TerminalInput.GetAttachment().GetOpaqueToken()
	case *apipb.CommandEnvelope_TerminalResize:
		admission.Capability, admission.ResourceToken = ApplicationCapabilityTerminalAttachment, value.TerminalResize.GetAttachment().GetOpaqueToken()
	case *apipb.CommandEnvelope_TerminalResizeLock:
		admission.Capability, admission.ResourceToken = ApplicationCapabilityTerminalAttachment, value.TerminalResizeLock.GetAttachment().GetOpaqueToken()
	case *apipb.CommandEnvelope_PathListDirectories:
		admission.Capability = ApplicationCapabilityPathQuery
	case *apipb.CommandEnvelope_ReleaseResource:
		admission.Capability, admission.ResourceToken = ApplicationCapabilityResourceLifecycle, value.ReleaseResource.GetResource().GetOpaqueToken()
	}
	return admission
}

func applicationTestAck(result *apipb.ResultEnvelope, err error) *apipb.ResultEnvelope {
	if err != nil {
		return applicationTestFailure(result, err)
	}
	result.Result = &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}
	return result
}

func applicationTestFailure(result *apipb.ResultEnvelope, err error) *apipb.ResultEnvelope {
	result.Result = &apipb.ResultEnvelope_Error{Error: applicationTestError(err)}
	return result
}

func applicationTestError(err error) *apipb.ApiError {
	code := apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL
	switch {
	case errors.Is(err, ErrApplicationForbidden):
		code = apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN
	case errors.Is(err, ErrApplicationUnsupportedCapability):
		code = apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_CAPABILITY
	case errors.Is(err, ErrTerminalNotFound):
		code = apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND
	case errors.Is(err, ErrDuplicateTerminal), errors.Is(err, ErrTerminalExited):
		code = apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT
	case errors.Is(err, ErrInvalidTerminalID), errors.Is(err, ErrInvalidCommand), errors.Is(err, ErrInvalidServerSize):
		code = apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST
	}
	return &apipb.ApiError{Code: code, Message: err.Error(), Attempted: true}
}

func applicationTestTerminalInfo(endpointID string, info TerminalInfo, attachmentCount int) *apipb.TerminalInfo {
	state := apipb.TerminalState_TERMINAL_STATE_CREATED
	switch info.State {
	case TerminalStateRunning:
		state = apipb.TerminalState_TERMINAL_STATE_RUNNING
	case TerminalStateExited:
		state = apipb.TerminalState_TERMINAL_STATE_EXITED
	case TerminalStateRemoved:
		state = apipb.TerminalState_TERMINAL_STATE_REMOVED
	}
	out := &apipb.TerminalInfo{Ref: &apipb.TerminalRef{EndpointId: endpointID, TerminalId: info.ID}, Name: info.Name, Command: info.Command, Tags: info.Tags, Size: applicationTestSize(info.Size), State: state, Cwd: info.CWD, LiveCwd: info.LiveCWD, CreatedAtUnixNano: info.CreatedAt.UnixNano(), ExitedAtUnixNano: info.ExitedAt.UnixNano(), AttachmentCount: int32(attachmentCount)}
	if info.ExitCode != nil {
		value := int32(*info.ExitCode)
		out.ExitCode = &value
	}
	return out
}

func applicationTestResizeControl(control *TerminalResizeControl) *apipb.ResizeControl {
	if control == nil {
		return nil
	}
	reason := apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER
	switch control.Reason {
	case TerminalResizeReasonOwner:
		reason = apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER
	case TerminalResizeReasonObserver:
		reason = apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER
	case TerminalResizeReasonSizeLocked:
		reason = apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED
	}
	out := &apipb.ResizeControl{CanResize: control.CanResize, Reason: reason, SizeLocked: control.SizeLocked, SurfaceId: control.SurfaceID, OwnerSurfaceId: control.OwnerSurfaceID, OwnerViewId: control.OwnerViewID}
	if ownership := control.ResizeOwnership; ownership != nil {
		out.Ownership = &apipb.ResizeOwnership{OwnerAttachmentId: ownership.OwnerAttachmentID, OwnerSurfaceId: ownership.OwnerSurfaceID, OwnerViewId: ownership.OwnerViewID, Size: applicationTestSize(ownership.Size), SizeLocked: ownership.SizeLocked, Epoch: ownership.Epoch}
	}
	return out
}

func applicationTestAttachmentMode(mode apipb.AttachmentMode) TerminalAttachmentMode {
	if mode == apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER {
		return TerminalAttachmentModeObserver
	}
	return TerminalAttachmentModeCollaborator
}

func applicationTestResizePolicy(policy apipb.ResizePolicy) TerminalResizePolicy {
	switch policy {
	case apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER:
		return TerminalResizePolicyFollower
	case apipb.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return TerminalResizePolicyObserver
	default:
		return TerminalResizePolicyOwner
	}
}

func applicationTestSize(size Size) *apipb.TerminalSize {
	return &apipb.TerminalSize{Cols: uint32(size.Cols), Rows: uint32(size.Rows)}
}

func cloneApplicationTestSession(session *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if session == nil {
		return nil
	}
	return &apipb.EndpointSessionStamp{EndpointId: session.GetEndpointId(), RouteId: session.GetRouteId(), Generation: session.GetGeneration()}
}
