package apimapping

import (
	"time"

	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/proto/apipb"
)

// ValidateFileStorageCommand 校验 file/storage command 的边界字段和 operation/resource fence。
func ValidateFileStorageCommand(command *apipb.CommandEnvelope) error {
	requestContext := RequestContextForCommand(command)
	if err := ValidateRequestContext(requestContext); err != nil {
		return err
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_FileList:
		if err := validatePath(value.FileList.GetPath()); err != nil {
			return err
		}
		if value.FileList.GetLimit() < 0 || value.FileList.GetLimit() > 1000 {
			return validation("file_list.limit", "must be between zero and 1000")
		}
		if len(value.FileList.GetCursor()) > 4096 {
			return validation("file_list.cursor", "exceeds 4096 bytes")
		}
	case *apipb.CommandEnvelope_FileStat:
		return validatePath(value.FileStat.GetPath())
	case *apipb.CommandEnvelope_FilePreview:
		if err := validatePath(value.FilePreview.GetPath()); err != nil {
			return err
		}
		if value.FilePreview.GetMaxBytes() < 0 {
			return validation("file_preview.max_bytes", "must not be negative")
		}
	case *apipb.CommandEnvelope_FileMkdir:
		return validatePath(value.FileMkdir.GetPath())
	case *apipb.CommandEnvelope_FileRename:
		if err := validatePath(value.FileRename.GetPath()); err != nil {
			return err
		}
		return validatePath(value.FileRename.GetNewPath())
	case *apipb.CommandEnvelope_FileDelete:
		return validatePath(value.FileDelete.GetPath())
	case *apipb.CommandEnvelope_FileCopy:
		return validateFileBatch(value.FileCopy.GetPaths(), value.FileCopy.GetTargetDirectory())
	case *apipb.CommandEnvelope_FileMove:
		return validateFileBatch(value.FileMove.GetPaths(), value.FileMove.GetTargetDirectory())
	case *apipb.CommandEnvelope_FileDownloadOpen:
		if err := validatePath(value.FileDownloadOpen.GetPath()); err != nil {
			return err
		}
		if value.FileDownloadOpen.GetOffset() < 0 || value.FileDownloadOpen.GetExpectedSize() < 0 {
			return validation("file_download_open", "offset and expected_size must not be negative")
		}
		return ValidateOperationStamp(value.FileDownloadOpen.GetOperation(), requestContext.GetSession())
	case *apipb.CommandEnvelope_FileUploadOpen:
		if err := validatePath(value.FileUploadOpen.GetPath()); err != nil {
			return err
		}
		if value.FileUploadOpen.GetSize() < 0 {
			return validation("file_upload_open.size", "must not be negative")
		}
		if resume := value.FileUploadOpen.GetResume(); resume != nil {
			if len(resume.GetOpaqueToken()) == 0 || len(resume.GetOpaqueToken()) > 256 {
				return validation("file_upload_open.resume", "opaque_token must contain at most 256 bytes")
			}
		}
		return ValidateOperationStamp(value.FileUploadOpen.GetOperation(), requestContext.GetSession())
	case *apipb.CommandEnvelope_FileTransferCancel:
		return validateFileTransferCancel(value.FileTransferCancel, requestContext)
	case *apipb.CommandEnvelope_StorageGet:
		return validateStorageKey(value.StorageGet.GetKey())
	case *apipb.CommandEnvelope_StoragePut:
		if err := validateStorageKey(value.StoragePut.GetKey()); err != nil {
			return err
		}
		return validateStorageVersion(value.StoragePut.GetVersion())
	case *apipb.CommandEnvelope_StorageDelete:
		if err := validateStorageKey(value.StorageDelete.GetKey()); err != nil {
			return err
		}
		return validateStorageVersion(value.StorageDelete.GetVersion())
	case *apipb.CommandEnvelope_StorageList:
		if value.StorageList.GetAppId() == "" || value.StorageList.GetScope() == apipb.StorageScope_STORAGE_SCOPE_UNSPECIFIED {
			return validation("storage_list", "app_id and scope are required")
		}
		if len(value.StorageList.GetAppId()) > 256 || len(value.StorageList.GetOwnerId()) > 256 || len(value.StorageList.GetPrefix()) > 4096 {
			return validation("storage_list", "contains an oversized field")
		}
	default:
		return validation("command", "is not a file or storage command")
	}
	return nil
}

// FileListRequestFromProto 映射目录分页请求。
func FileListRequestFromProto(command *apipb.FileListCommand) corev2.FileListRequest {
	return corev2.FileListRequest{Path: command.GetPath(), Cursor: command.GetCursor(), Limit: int(command.GetLimit())}
}

// FilePathRequestFromProto 映射单路径请求。
func FilePathRequestFromProto(path string, recursive bool) corev2.FilePathRequest {
	return corev2.FilePathRequest{Path: path, Recursive: recursive}
}

// FilePreviewRequestFromProto 映射有界预览请求。
func FilePreviewRequestFromProto(command *apipb.FilePreviewCommand) corev2.FilePreviewRequest {
	return corev2.FilePreviewRequest{Path: command.GetPath(), MaxBytes: command.GetMaxBytes()}
}

// FileRenameRequestFromProto 映射 rename 请求。
func FileRenameRequestFromProto(command *apipb.FileRenameCommand) corev2.FileRenameRequest {
	return corev2.FileRenameRequest{Path: command.GetPath(), NewPath: command.GetNewPath(), Overwrite: command.GetOverwrite()}
}

// FileCopyRequestFromProto 映射 copy 请求。
func FileCopyRequestFromProto(command *apipb.FileCopyCommand) corev2.FileCopyMoveRequest {
	return corev2.FileCopyMoveRequest{Paths: append([]string(nil), command.GetPaths()...), TargetDir: command.GetTargetDirectory(), Overwrite: command.GetOverwrite()}
}

// FileMoveRequestFromProto 映射 move 请求。
func FileMoveRequestFromProto(command *apipb.FileMoveCommand) corev2.FileCopyMoveRequest {
	return corev2.FileCopyMoveRequest{Paths: append([]string(nil), command.GetPaths()...), TargetDir: command.GetTargetDirectory(), Overwrite: command.GetOverwrite()}
}

// FileDownloadRequestFromProto 映射 download open 请求。
func FileDownloadRequestFromProto(command *apipb.FileDownloadOpenCommand) corev2.FileDownloadOpenRequest {
	return corev2.FileDownloadOpenRequest{Path: command.GetPath(), Offset: command.GetOffset(), ExpectedSize: command.GetExpectedSize(), ExpectedModifiedAt: timeFromUnixNano(command.GetExpectedModifiedAtUnixNano())}
}

// FileUploadRequestFromProto 映射 upload open 请求。
func FileUploadRequestFromProto(command *apipb.FileUploadOpenCommand) corev2.FileUploadOpenRequest {
	return corev2.FileUploadOpenRequest{
		Path:                command.GetPath(),
		Size:                command.GetSize(),
		Overwrite:           command.GetOverwrite(),
		ResumeTransferToken: cloneBytes(command.GetResume().GetOpaqueToken()),
	}
}

// FileTransferCancelRequestFromProto 将 Proto 二选一销毁凭据映射到 core；不解释 opaque token 内容。
func FileTransferCancelRequestFromProto(command *apipb.FileTransferCancelCommand) corev2.FileTransferCancelRequest {
	return corev2.FileTransferCancelRequest{
		ResourceToken:     cloneBytes(command.GetTransfer().GetOpaqueToken()),
		UploadResumeToken: cloneBytes(command.GetUploadResume().GetOpaqueToken()),
	}
}

// FileEntryToProto 映射 daemon file metadata。
func FileEntryToProto(entry corev2.FileEntry) *apipb.FileEntry {
	return &apipb.FileEntry{Path: entry.Path, Name: entry.Name, Type: fileEntryTypeToProto(entry.Type), Size: entry.Size, Mode: entry.Mode, ModifiedAtUnixNano: unixNanoOrZero(entry.ModifiedAt), LinkTarget: entry.LinkTarget}
}

// FileStatToProto 包装单个 daemon file metadata projection。
func FileStatToProto(entry corev2.FileEntry) *apipb.FileStatResult {
	return &apipb.FileStatResult{Entry: FileEntryToProto(entry)}
}

// FileListToProto 映射目录窗口。
func FileListToProto(result corev2.FileListResult) *apipb.FileListResult {
	out := &apipb.FileListResult{Path: result.Path, NextCursor: result.NextCursor}
	for _, entry := range result.Entries {
		out.Entries = append(out.Entries, FileEntryToProto(entry))
	}
	return out
}

// FilePreviewToProto 映射有界预览。
func FilePreviewToProto(result corev2.FilePreviewResult) *apipb.FilePreviewResult {
	return &apipb.FilePreviewResult{Entry: FileEntryToProto(result.Entry), MimeType: result.MIMEType, Content: cloneBytes(result.Content), Truncated: result.Truncated, Sha256: cloneBytes(result.SHA256)}
}

// FileOperationToProto 映射单项 mutation 结果。
func FileOperationToProto(result corev2.FileOperationResult) *apipb.FileOperationResult {
	return &apipb.FileOperationResult{Path: result.Path, TargetPath: result.TargetPath, Success: result.Success, ErrorCode: result.ErrorCode, ErrorMessage: result.ErrorMessage}
}

// FileBatchToProto 映射批量 mutation 结果。
func FileBatchToProto(result corev2.FileBatchResult) *apipb.FileBatchResult {
	out := &apipb.FileBatchResult{}
	for _, item := range result.Results {
		out.Results = append(out.Results, FileOperationToProto(item))
	}
	return out
}

// FileTransferToProto 映射 session-bound transfer；内部 channel 隐藏在 resource token，流控上限通过 Proto 显式返回跨端 consumer。
func FileTransferToProto(origin *apipb.EndpointSessionStamp, operation *apipb.OperationStamp, transfer corev2.FileTransfer) *apipb.FileTransferOpenResult {
	handle := &apipb.FileTransferHandle{Resource: &apipb.ResourceHandle{OpaqueToken: cloneBytes(transfer.OpaqueToken), Kind: apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER, Session: cloneSessionStamp(origin), Generation: 1}, Path: transfer.Path, Offset: transfer.Offset, Size: transfer.Size, ModifiedAtUnixNano: unixNanoOrZero(transfer.ModifiedAt), Operation: cloneOperationStamp(operation), ChunkBytes: uint32(transfer.ChunkBytes), WindowBytes: transfer.WindowBytes}
	if len(transfer.ResumeToken) > 0 {
		handle.Resume = &apipb.FileUploadResumeHandle{OpaqueToken: cloneBytes(transfer.ResumeToken)}
	}
	return &apipb.FileTransferOpenResult{Transfer: handle}
}

// FileTransferCancelToProto 映射 transfer cancellation result。
func FileTransferCancelToProto(result corev2.FileTransferCancelResult) *apipb.FileTransferCancelResult {
	return &apipb.FileTransferCancelResult{Cancelled: result.Cancelled}
}

// StorageKeyFromProto 映射 opaque storage identity。
func StorageKeyFromProto(key *apipb.StorageKey) (string, corev2.StorageScope, string, string) {
	return key.GetAppId(), storageScopeFromProto(key.GetScope()), key.GetOwnerId(), key.GetKey()
}

// StoragePutFromProto 映射 storage CAS put。
func StoragePutFromProto(command *apipb.StoragePutCommand) corev2.StoragePutRequest {
	appID, scope, ownerID, key := StorageKeyFromProto(command.GetKey())
	return corev2.StoragePutRequest{AppID: appID, Scope: scope, OwnerID: ownerID, Key: key, Value: cloneBytes(command.GetValue()), CheckVersion: command.GetVersion().GetCheckVersion(), ExpectedVersion: command.GetVersion().GetExpectedVersion()}
}

// StorageDeleteFromProto 映射 storage CAS delete。
func StorageDeleteFromProto(command *apipb.StorageDeleteCommand) corev2.StorageDeleteRequest {
	appID, scope, ownerID, key := StorageKeyFromProto(command.GetKey())
	return corev2.StorageDeleteRequest{AppID: appID, Scope: scope, OwnerID: ownerID, Key: key, CheckVersion: command.GetVersion().GetCheckVersion(), ExpectedVersion: command.GetVersion().GetExpectedVersion()}
}

// StorageEntryToProto 映射 opaque storage entry，不解释 value。
func StorageEntryToProto(entry corev2.StorageEntry) *apipb.StorageEntry {
	return &apipb.StorageEntry{Key: &apipb.StorageKey{AppId: entry.AppID, Scope: storageScopeToProto(entry.Scope), OwnerId: entry.OwnerID, Key: entry.Key}, Value: cloneBytes(entry.Value), Version: entry.Version, UpdatedAtUnixNano: unixNanoOrZero(entry.UpdatedAt)}
}

// StorageDeleteToProto 映射 delete 结果。
func StorageDeleteToProto(result corev2.StorageDeleteResult) *apipb.StorageDeleteResult {
	return &apipb.StorageDeleteResult{Key: &apipb.StorageKey{AppId: result.AppID, Scope: storageScopeToProto(result.Scope), OwnerId: result.OwnerID, Key: result.Key}, Deleted: result.Deleted, Version: result.Version}
}

// StorageGetToProto 包装 storage entry projection。
func StorageGetToProto(entry corev2.StorageEntry) *apipb.StorageGetResult {
	return &apipb.StorageGetResult{Entry: StorageEntryToProto(entry)}
}

// StoragePutToProto 包装 storage entry projection。
func StoragePutToProto(entry corev2.StorageEntry) *apipb.StoragePutResult {
	return &apipb.StoragePutResult{Entry: StorageEntryToProto(entry)}
}

// StorageListToProto 映射 core storage entry window。
func StorageListToProto(entries []corev2.StorageEntry) *apipb.StorageListResult {
	result := &apipb.StorageListResult{Entries: make([]*apipb.StorageEntry, 0, len(entries))}
	for _, entry := range entries {
		result.Entries = append(result.Entries, StorageEntryToProto(entry))
	}
	return result
}

// StorageScopeFromProto 映射公共 storage scope enum。
func StorageScopeFromProto(scope apipb.StorageScope) corev2.StorageScope {
	return storageScopeFromProto(scope)
}

func validatePath(path string) error {
	if path == "" {
		return validation("file.path", "is required")
	}
	if len(path) > 4096 {
		return validation("file.path", "exceeds 4096 bytes")
	}
	return nil
}
func validateFileBatch(paths []string, target string) error {
	if len(paths) == 0 {
		return validation("file.paths", "must not be empty")
	}
	if len(paths) > 1000 {
		return validation("file.paths", "exceeds 1000 entries")
	}
	for _, path := range paths {
		if err := validatePath(path); err != nil {
			return err
		}
	}
	return validatePath(target)
}
func validateFileTransferOperation(resource *apipb.ResourceHandle, operation *apipb.OperationStamp, contextMessage *apipb.RequestContext) error {
	if err := ValidateResourceHandle(resource); err != nil {
		return err
	}
	if resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER || !SessionStampsEqual(resource.GetSession(), contextMessage.GetSession()) {
		return validation("file_transfer", "must be a current-session file transfer")
	}
	return ValidateOperationStamp(operation, contextMessage.GetSession())
}
func validateFileTransferCancel(command *apipb.FileTransferCancelCommand, contextMessage *apipb.RequestContext) error {
	if command == nil {
		return validation("file_transfer_cancel", "is required")
	}
	hasTransfer := command.GetTransfer() != nil
	hasResume := command.GetUploadResume() != nil
	if hasTransfer == hasResume {
		return validation("file_transfer_cancel", "requires exactly one transfer or upload_resume credential")
	}
	if hasTransfer {
		return validateFileTransferOperation(command.GetTransfer(), command.GetOperation(), contextMessage)
	}
	token := command.GetUploadResume().GetOpaqueToken()
	if len(token) == 0 || len(token) > 256 {
		return validation("file_transfer_cancel.upload_resume", "opaque_token must contain at most 256 bytes")
	}
	return ValidateOperationStamp(command.GetOperation(), contextMessage.GetSession())
}
func validateStorageKey(key *apipb.StorageKey) error {
	if key == nil || key.GetAppId() == "" || key.GetKey() == "" || key.GetScope() == apipb.StorageScope_STORAGE_SCOPE_UNSPECIFIED {
		return validation("storage.key", "app_id, scope and key are required")
	}
	if len(key.GetAppId()) > 256 || len(key.GetOwnerId()) > 256 || len(key.GetKey()) > 4096 {
		return validation("storage.key", "contains an oversized field")
	}
	return nil
}
func validateStorageVersion(version *apipb.StorageVersionFence) error {
	if version == nil {
		return nil
	}
	if !version.GetCheckVersion() && version.GetExpectedVersion() != 0 {
		return validation("storage.version", "expected_version requires check_version")
	}
	return nil
}
func storageScopeFromProto(scope apipb.StorageScope) corev2.StorageScope {
	if scope == apipb.StorageScope_STORAGE_SCOPE_PRIVATE {
		return corev2.StorageScopePrivate
	}
	return corev2.StorageScopePublic
}
func storageScopeToProto(scope corev2.StorageScope) apipb.StorageScope {
	if scope == corev2.StorageScopePrivate {
		return apipb.StorageScope_STORAGE_SCOPE_PRIVATE
	}
	return apipb.StorageScope_STORAGE_SCOPE_PUBLIC
}
func fileEntryTypeToProto(value string) apipb.FileEntryType {
	switch value {
	case "file":
		return apipb.FileEntryType_FILE_ENTRY_TYPE_FILE
	case "dir":
		return apipb.FileEntryType_FILE_ENTRY_TYPE_DIRECTORY
	case "symlink":
		return apipb.FileEntryType_FILE_ENTRY_TYPE_SYMLINK
	default:
		return apipb.FileEntryType_FILE_ENTRY_TYPE_OTHER
	}
}
func timeFromUnixNano(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value).UTC()
}
