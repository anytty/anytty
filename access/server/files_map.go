package server

import (
	"time"

	"github.com/anytty/anytty/access/files"
	"github.com/anytty/anytty/proto/access/apipb"
	"google.golang.org/protobuf/proto"
)

func fileListRequestFromProto(command *apipb.FileListCommand) files.ListRequest {
	return files.ListRequest{Path: command.GetPath(), Cursor: command.GetCursor(), Limit: int(command.GetLimit())}
}

func filePathRequestFromProto(path string, recursive bool) files.PathRequest {
	return files.PathRequest{Path: path, Recursive: recursive}
}

func filePreviewRequestFromProto(command *apipb.FilePreviewCommand) files.PreviewRequest {
	return files.PreviewRequest{Path: command.GetPath(), MaxBytes: command.GetMaxBytes()}
}

func fileRenameRequestFromProto(command *apipb.FileRenameCommand) files.RenameRequest {
	return files.RenameRequest{Path: command.GetPath(), NewPath: command.GetNewPath(), Overwrite: command.GetOverwrite()}
}

func fileCopyMoveRequestFromProto(command *apipb.FileCopyCommand) files.CopyMoveRequest {
	return files.CopyMoveRequest{Paths: append([]string(nil), command.GetPaths()...), TargetDir: command.GetTargetDirectory(), Overwrite: command.GetOverwrite()}
}

func fileMoveRequestFromProto(command *apipb.FileMoveCommand) files.CopyMoveRequest {
	return files.CopyMoveRequest{Paths: append([]string(nil), command.GetPaths()...), TargetDir: command.GetTargetDirectory(), Overwrite: command.GetOverwrite()}
}

func fileDownloadRequestFromProto(command *apipb.FileDownloadOpenCommand) files.DownloadRequest {
	return files.DownloadRequest{
		Path:                  command.GetPath(),
		Offset:                command.GetOffset(),
		ExpectedSize:          command.GetExpectedSize(),
		ExpectedModifiedAt:    timeFromUnixNano(command.GetExpectedModifiedAtUnixNano()),
		AcceptCompression:     append([]string(nil), command.GetAcceptCompression()...),
		ProgressIntervalBytes: command.GetProgressIntervalBytes(),
	}
}

func fileUploadRequestFromProto(command *apipb.FileUploadOpenCommand) files.UploadRequest {
	return files.UploadRequest{
		Path:                  command.GetPath(),
		Size:                  command.GetSize(),
		Overwrite:             command.GetOverwrite(),
		ResumeTransferToken:   append([]byte(nil), command.GetResume().GetOpaqueToken()...),
		AcceptCompression:     append([]string(nil), command.GetAcceptCompression()...),
		ProgressIntervalBytes: command.GetProgressIntervalBytes(),
	}
}

func fileTransferCancelRequestFromProto(command *apipb.FileTransferCancelCommand) files.CancelRequest {
	return files.CancelRequest{
		ResourceToken:     append([]byte(nil), command.GetTransfer().GetOpaqueToken()...),
		UploadResumeToken: append([]byte(nil), command.GetUploadResume().GetOpaqueToken()...),
	}
}

func fileEntryToProto(entry files.Entry) *apipb.FileEntry {
	return &apipb.FileEntry{
		Path:               entry.Path,
		Name:               entry.Name,
		Type:               fileEntryTypeToProto(entry.Type),
		Size:               entry.Size,
		Mode:               entry.Mode,
		ModifiedAtUnixNano: unixNanoOrZero(entry.ModifiedAt),
		LinkTarget:         entry.LinkTarget,
	}
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

func fileListResultToProto(result files.ListResult) *apipb.FileListResult {
	out := &apipb.FileListResult{Path: result.Path, NextCursor: result.NextCursor}
	for _, entry := range result.Entries {
		out.Entries = append(out.Entries, fileEntryToProto(entry))
	}
	return out
}

func fileStatResultToProto(entry files.Entry) *apipb.FileStatResult {
	return &apipb.FileStatResult{Entry: fileEntryToProto(entry)}
}

func filePreviewResultToProto(result files.PreviewResult) *apipb.FilePreviewResult {
	return &apipb.FilePreviewResult{
		Entry: fileEntryToProto(result.Entry), MimeType: result.MIMEType,
		Content: append([]byte(nil), result.Content...), Truncated: result.Truncated, Sha256: append([]byte(nil), result.SHA256...),
	}
}

func fileOperationToProto(result files.OperationResult) *apipb.FileOperationResult {
	return &apipb.FileOperationResult{
		Path: result.Path, TargetPath: result.TargetPath, Success: result.Success,
		ErrorCode: result.ErrorCode, ErrorMessage: result.ErrorMessage,
	}
}

func fileBatchResultToProto(result files.BatchResult) *apipb.FileBatchResult {
	out := &apipb.FileBatchResult{}
	for _, item := range result.Results {
		out.Results = append(out.Results, fileOperationToProto(item))
	}
	return out
}

func fileTransferToProto(origin *apipb.EndpointSessionStamp, operation *apipb.OperationStamp, transfer *files.Transfer) *apipb.FileTransferOpenResult {
	handle := &apipb.FileTransferHandle{
		Resource: &apipb.ResourceHandle{
			OpaqueToken: append([]byte(nil), transfer.OpaqueToken...),
			Kind:        apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER,
			Session:     cloneSessionStamp(origin),
			Generation:  1,
		},
		Path:                  transfer.Path,
		Offset:                transfer.Offset,
		Size:                  transfer.Size,
		ModifiedAtUnixNano:    unixNanoOrZero(transfer.ModifiedAt),
		Operation:             cloneOperationStamp(operation),
		ChunkBytes:            uint32(transfer.ChunkBytes),
		WindowBytes:           transfer.WindowBytes,
		ContentEncoding:       transfer.ContentEncoding,
		ProgressIntervalBytes: transfer.ProgressIntervalBytes,
	}
	if len(transfer.ResumeToken) > 0 {
		handle.Resume = &apipb.FileUploadResumeHandle{OpaqueToken: append([]byte(nil), transfer.ResumeToken...)}
	}
	return &apipb.FileTransferOpenResult{Transfer: handle}
}

func cloneOperationStamp(stamp *apipb.OperationStamp) *apipb.OperationStamp {
	if stamp == nil {
		return nil
	}
	return proto.Clone(stamp).(*apipb.OperationStamp)
}

func timeFromUnixNano(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value).UTC()
}

func unixNanoOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixNano()
}
