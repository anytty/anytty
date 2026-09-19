package files

import (
	"errors"
	"time"
)

var (
	// ErrInvalidUploadResume 表示客户端把非 resume namespace 的 opaque token 用作续传凭据。
	// 它属于请求校验失败，必须映射为 INVALID_REQUEST，不得退化为内部错误。
	ErrInvalidUploadResume = errors.New("file upload resume credential is invalid")
	// ErrTransferNotFound 表示 transfer 凭据无法解析或不属于当前 access 实例。
	ErrTransferNotFound = errors.New("file transfer was not found")
	// ErrTransferUnavailable 表示 resume 记录、临时文件或目标状态已失效，不能安全续传。
	ErrTransferUnavailable = errors.New("file transfer is unavailable")
	// ErrDone 由 stream handler 返回，表示该 stream 已到达终态，调用方必须回收 binding。
	ErrDone = errors.New("file transfer stream is done")
)

// ListRequest 表达目录分页请求。
type ListRequest struct {
	Path   string
	Cursor string
	Limit  int
}

// PathRequest 表达单路径 mutation。
type PathRequest struct {
	Path      string
	Recursive bool
}

// Entry 是文件系统 metadata 投影。
type Entry struct {
	Path       string
	Name       string
	Type       string
	Size       int64
	Mode       uint32
	ModifiedAt time.Time
	LinkTarget string
}

// ListResult 是单次目录枚举窗口。
type ListResult struct {
	Path       string
	Entries    []Entry
	NextCursor string
}

// PreviewRequest 请求有界读取普通文件前缀。
type PreviewRequest struct {
	Path     string
	MaxBytes int64
}

// PreviewResult 是有界内容预览，不是下载通道。
type PreviewResult struct {
	Entry     Entry
	MIMEType  string
	Content   []byte
	Truncated bool
	SHA256    []byte
}

// RenameRequest 表达单个原子重命名请求。
type RenameRequest struct {
	Path      string
	NewPath   string
	Overwrite bool
}

// CopyMoveRequest 表达复制或移动到目标目录的批量请求。
type CopyMoveRequest struct {
	Paths     []string
	TargetDir string
	Overwrite bool
}

// OperationResult 表达一个 mutation 的确定结果。
type OperationResult struct {
	Path         string
	TargetPath   string
	Success      bool
	ErrorCode    string
	ErrorMessage string
}

// BatchResult 保存批量 mutation 的逐项结果。
type BatchResult struct {
	Results []OperationResult
}

// DownloadRequest 打开可续传下载并固定源文件 identity。
type DownloadRequest struct {
	Path               string
	Offset             int64
	ExpectedSize       int64
	ExpectedModifiedAt time.Time
	// AcceptCompression 是客户端可解压的 data frame 编码（如 "zstd"）；空=identity。
	AcceptCompression []string
	// ProgressIntervalBytes 请求结构化进度合并窗口；0=关闭（默认，字节行为不变）。
	ProgressIntervalBytes int64
}

// UploadRequest 打开可续传上传。
type UploadRequest struct {
	Path                string
	Size                int64
	Overwrite           bool
	ResumeTransferToken []byte
	// AcceptCompression 是客户端可编码的 data frame 编码（如 "zstd"）；空=identity。
	AcceptCompression []string
	// ProgressIntervalBytes 请求服务端在 ack 上合并结构化进度；0=关闭（默认）。
	ProgressIntervalBytes int64
}

// CancelRequest 携带二选一的 transfer 销毁凭据。
// ResourceToken 只属于当前 session；UploadResumeToken 可跨 session 销毁未完成上传。
type CancelRequest struct {
	ResourceToken     []byte
	UploadResumeToken []byte
}

// Transfer 是 transfer open 的元数据投影；token 内嵌 channel，不暴露给其他层解析。
type Transfer struct {
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
	// ContentEncoding 是本次 data stream 的编码（""=identity）。
	ContentEncoding string
	// ProgressIntervalBytes 是实际启用的进度合并窗口；0=未启用。
	ProgressIntervalBytes int64
}

// CancelResult 表示本次调用是否取消了活动 transfer。
type CancelResult struct {
	Cancelled bool
}
