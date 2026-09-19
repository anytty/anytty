package server

import (
	"context"
	"errors"
	"strconv"

	"github.com/anytty/anytty/access/files"
	apimapping "github.com/anytty/anytty/api_mapping"
	"github.com/anytty/anytty/proto/access/apipb"
)

// executeFiles 在 access 本地终结 file.* 命令（Phase 2 迁移）。
// 路径解析、根目录约束与 transfer 状态全部由 access/files 拥有；resource token
// 由 access 以同一不透明形状签发，不暴露任何 provider token。
func (session *session) executeFiles(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if err := apimapping.ValidateFileStorageCommand(command); err != nil {
		return nil, &routeError{apiError: apimapping.ErrorToProto(err, false)}
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_FileList:
		result, err := session.server.files.List(fileListRequestFromProto(value.FileList))
		if err != nil {
			return nil, fileRouteError(err)
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileList{FileList: fileListResultToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileStat:
		entry, err := session.server.files.Stat(filePathRequestFromProto(value.FileStat.GetPath(), false))
		if err != nil {
			return nil, fileRouteError(err)
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileStat{FileStat: fileStatResultToProto(entry)}
		return envelope, nil
	case *apipb.CommandEnvelope_FilePreview:
		result, err := session.server.files.Preview(filePreviewRequestFromProto(value.FilePreview))
		if err != nil {
			return nil, fileRouteError(err)
		}
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FilePreview{FilePreview: filePreviewResultToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileMkdir:
		result := session.server.files.Mkdir(filePathRequestFromProto(value.FileMkdir.GetPath(), value.FileMkdir.GetRecursive()))
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileOperation{FileOperation: fileOperationToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileRename:
		result := session.server.files.Rename(fileRenameRequestFromProto(value.FileRename))
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileOperation{FileOperation: fileOperationToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileDelete:
		result := session.server.files.Delete(filePathRequestFromProto(value.FileDelete.GetPath(), value.FileDelete.GetRecursive()))
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileOperation{FileOperation: fileOperationToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileCopy:
		result := session.server.files.Copy(fileCopyMoveRequestFromProto(value.FileCopy))
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileBatch{FileBatch: fileBatchResultToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileMove:
		result := session.server.files.Move(fileMoveRequestFromProto(value.FileMove))
		envelope := newResultEnvelope(command)
		envelope.Result = &apipb.ResultEnvelope_FileBatch{FileBatch: fileBatchResultToProto(result)}
		return envelope, nil
	case *apipb.CommandEnvelope_FileDownloadOpen:
		return session.openLocalDownload(ctx, command, value.FileDownloadOpen)
	case *apipb.CommandEnvelope_FileUploadOpen:
		return session.openLocalUpload(ctx, command, value.FileUploadOpen)
	case *apipb.CommandEnvelope_FileTransferCancel:
		return session.cancelLocalTransfer(command, value.FileTransferCancel)
	default:
		return nil, errUnsupportedCommand
	}
}

func (session *session) openLocalDownload(ctx context.Context, command *apipb.CommandEnvelope, request *apipb.FileDownloadOpenCommand) (*apipb.ResultEnvelope, error) {
	channel, err := session.allocateLocalChannel()
	if err != nil {
		return nil, &routeError{apiError: &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED, Message: err.Error(), Retryable: true}}
	}
	transfer, handle, err := session.server.files.OpenDownload(session.lifetimeContext(), fileDownloadRequestFromProto(request), channel, sessionStream{session: session, channel: channel})
	if err != nil {
		session.releaseLocalChannel(channel)
		return nil, fileRouteError(err)
	}
	session.registerLocalBinding(apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, transfer.OpaqueToken, channel, transfer.ID, handle)
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: fileTransferToProto(command.GetContext().GetSession(), request.GetOperation(), transfer)}
	return envelope, nil
}

func (session *session) openLocalUpload(ctx context.Context, command *apipb.CommandEnvelope, request *apipb.FileUploadOpenCommand) (*apipb.ResultEnvelope, error) {
	channel, err := session.allocateLocalChannel()
	if err != nil {
		return nil, &routeError{apiError: &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED, Message: err.Error(), Retryable: true}}
	}
	transfer, handle, err := session.server.files.OpenUpload(session.lifetimeContext(), fileUploadRequestFromProto(request), channel, sessionStream{session: session, channel: channel}, sessionOwnerID(command))
	if err != nil {
		session.releaseLocalChannel(channel)
		return nil, fileRouteError(err)
	}
	session.registerLocalBinding(apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, transfer.OpaqueToken, channel, transfer.ID, handle)
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_FileTransferOpen{FileTransferOpen: fileTransferToProto(command.GetContext().GetSession(), request.GetOperation(), transfer)}
	return envelope, nil
}

func (session *session) cancelLocalTransfer(command *apipb.CommandEnvelope, request *apipb.FileTransferCancelCommand) (*apipb.ResultEnvelope, error) {
	cancelRequest := fileTransferCancelRequestFromProto(request)
	cancelled := false
	if len(cancelRequest.UploadResumeToken) > 0 {
		result, err := session.server.files.CancelResume(cancelRequest.UploadResumeToken)
		if err != nil {
			return nil, fileRouteError(err)
		}
		cancelled = result
	} else {
		binding := session.localBindingForToken(cancelRequest.ResourceToken)
		if binding == nil {
			return nil, fileRouteError(files.ErrTransferNotFound)
		}
		cancelled = session.server.files.CancelResource(binding.localID)
		session.retireBinding(binding)
	}
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_FileTransferCancel{FileTransferCancel: &apipb.FileTransferCancelResult{Cancelled: cancelled}}
	return envelope, nil
}

// sessionOwnerID 是 upload 记录的 owner 绑定：当前 client generation。
func sessionOwnerID(command *apipb.CommandEnvelope) string {
	stamp := command.GetContext().GetSession()
	if stamp == nil {
		return ""
	}
	return stamp.GetEndpointId() + "|" + stamp.GetRouteId() + "|" + strconv.FormatUint(stamp.GetGeneration(), 10)
}

// fileRouteError 把 access/files 错误分类为公共 typed error，语义与 api_mapping.CoreError 对齐。
func fileRouteError(err error) *routeError {
	apiError := &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL, Message: err.Error()}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED, Message: err.Error()}
	case errors.Is(err, files.ErrInvalidUploadResume):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST, Message: err.Error()}
	case errors.Is(err, files.ErrTransferNotFound):
		apiError = &apipb.ApiError{Code: apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND, Message: err.Error()}
	}
	return &routeError{apiError: apiError}
}
