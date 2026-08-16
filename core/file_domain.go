package core

import (
	"errors"
	"time"
)

var (
	// ErrInvalidFileUploadResume 表示客户端把非 resume namespace 的 opaque token 用作续传凭据。
	// 该错误属于请求校验失败；不得退化为内部错误，也不得尝试按 active resource token 解释。
	ErrInvalidFileUploadResume = errors.New("file upload resume credential is invalid")
)

// FilePathRequest 表达 owning daemon 文件系统上的单路径 mutation。
type FilePathRequest struct {
	Path      string
	Recursive bool
}

// FileListRequest 表达 daemon-owned 目录分页请求。
type FileListRequest struct {
	Path   string
	Cursor string
	Limit  int
}

// FileEntry 是 daemon 文件系统 metadata 的 core-native 投影。
type FileEntry struct {
	Path       string
	Name       string
	Type       string
	Size       int64
	Mode       uint32
	ModifiedAt time.Time
	LinkTarget string
}

// FileListResult 是单次目录枚举窗口。
type FileListResult struct {
	Path       string
	Entries    []FileEntry
	NextCursor string
}

// FilePreviewRequest 请求有界读取普通文件前缀。
type FilePreviewRequest struct {
	Path     string
	MaxBytes int64
}

// FilePreviewResult 是有界内容预览，不是下载通道。
type FilePreviewResult struct {
	Entry     FileEntry
	MIMEType  string
	Content   []byte
	Truncated bool
}

// FileRenameRequest 表达单个原子重命名请求。
type FileRenameRequest struct {
	Path      string
	NewPath   string
	Overwrite bool
}

// FileCopyMoveRequest 表达复制或移动到目标目录的批量请求。
type FileCopyMoveRequest struct {
	Paths     []string
	TargetDir string
	Overwrite bool
}

// FileOperationResult 表达一个文件 mutation 的确定结果。
type FileOperationResult struct {
	Path         string
	TargetPath   string
	Success      bool
	ErrorCode    string
	ErrorMessage string
}

// FileBatchResult 保存批量 mutation 的逐项结果。
type FileBatchResult struct {
	Results []FileOperationResult
}

// FileDownloadOpenRequest 打开可续传下载并固定源文件 identity。
type FileDownloadOpenRequest struct {
	Path               string
	Offset             int64
	ExpectedSize       int64
	ExpectedModifiedAt time.Time
}

// FileUploadOpenRequest 打开 daemon-owned 临时上传。
type FileUploadOpenRequest struct {
	Path                string
	Size                int64
	Overwrite           bool
	ResumeTransferToken []byte
}

// FileTransferCancelRequest 携带二选一的 transfer 销毁凭据。
// ResourceToken 只属于当前 protocol session；UploadResumeToken 由 principal 约束，可在新 session 中销毁未完成上传。
type FileTransferCancelRequest struct {
	ResourceToken     []byte
	UploadResumeToken []byte
}

// FileTransfer 是 session-local stream binding 的 core-native 结果。
// Channel、window 和 chunk 只供 protocol binding 使用，不进入公共 Proto API。
type FileTransfer struct {
	ID          string
	Channel     uint16
	Path        string
	Offset      int64
	Size        int64
	ModifiedAt  time.Time
	WindowBytes int64
	ChunkBytes  int
	OpaqueToken []byte
	ResumeToken []byte
}

// FileTransferCancelResult 表示本次调用是否取消了活动 transfer。
type FileTransferCancelResult struct {
	Cancelled bool
}
