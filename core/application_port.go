package core

import (
	"context"
	"errors"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
)

// ApplicationCapability 表示 connection admission 需要的 core-native 能力类别。
// 它只参与 daemon 内部授权，不是公共 capability schema 的第二份真值。
type ApplicationCapability uint8

// ApplicationResourceKind 只用于 connection admission 选择 owning registry，不复制公共资源字段。
type ApplicationResourceKind uint8

const (
	// ApplicationResourceKindSubscription 表示 session-owned event subscription。
	ApplicationResourceKindSubscription ApplicationResourceKind = iota + 1
)

const (
	// ApplicationCapabilityResourceLifecycle 表示释放 session-owned resource。
	ApplicationCapabilityResourceLifecycle ApplicationCapability = iota + 1
	// ApplicationCapabilityTerminalLifecycle 表示 terminal lifecycle 与 metadata 操作。
	ApplicationCapabilityTerminalLifecycle
	// ApplicationCapabilityTerminalInventory 表示按 connection scope 过滤的 terminal inventory 查询。
	ApplicationCapabilityTerminalInventory
	// ApplicationCapabilityTerminalAttachment 表示 attachment、input 与 resize 操作。
	ApplicationCapabilityTerminalAttachment
	// ApplicationCapabilityPathQuery 表示 daemon 文件系统 path 查询。
	ApplicationCapabilityPathQuery
	// ApplicationCapabilityHistory 表示 authoritative history 查询。
	ApplicationCapabilityHistory
	// ApplicationCapabilityLiveScreen 表示 native live screen 查询。
	ApplicationCapabilityLiveScreen
	// ApplicationCapabilityFile 表示 daemon 文件系统操作。
	ApplicationCapabilityFile
	// ApplicationCapabilityStorage 表示 daemon opaque storage 操作。
	ApplicationCapabilityStorage
	// ApplicationCapabilityEventSubscription 表示 application event 订阅。
	ApplicationCapabilityEventSubscription
	// ApplicationCapabilityClientAccess 表示 daemon client access 管理。
	ApplicationCapabilityClientAccess
	// ApplicationCapabilityRemoteControl 表示 local-owner remote runtime 控制。
	ApplicationCapabilityRemoteControl
)

var (
	// ErrApplicationForbidden 表示连接身份有效，但请求不在 immutable transport scope 内。
	ErrApplicationForbidden = errors.New("application request is forbidden")
	// ErrApplicationUnsupportedCapability 表示连接没有协商请求所需能力。
	ErrApplicationUnsupportedCapability = errors.New("application capability is unsupported")
	// ErrApplicationCancellationUnavailable 表示 daemon 尚未发布 operation cancellation registry。
	ErrApplicationCancellationUnavailable = errors.New("application operation cancellation is unavailable")
	// ErrProtocolResourceExhausted 表示当前 protocol session 已达到具体资源上限。
	ErrProtocolResourceExhausted = errors.New("protocol session resource capacity is exhausted")
)

// ApplicationAdmission 是 API Layer 映射后的 connection-bound 授权输入。
// TerminalID 与 ResourceToken 只能二选一表达目标；core 不读取 Proto command 推断权限。
type ApplicationAdmission struct {
	Capability                 ApplicationCapability
	TerminalID                 string
	ResourceToken              []byte
	ResourceKind               ApplicationResourceKind
	FileOperation              string
	MachineLifecycleEventsOnly bool
}

// ApplicationAdmissionLease 覆盖一次 controller 执行期，保证连接 scope 不在校验后失效。
type ApplicationAdmissionLease interface {
	Release()
}

// ApplicationEventEncoder 把 core-native event 转成公共事件 payload。
// 实现由 API Mapping 注入；core 只负责 subscription lifecycle 与 framing send。
type ApplicationEventEncoder func(Event, []byte) ([]byte, error)

// TerminalDefaults 是 daemon 默认 shell 与 cwd 的 core-native 查询结果。
type TerminalDefaults struct {
	DefaultCommand []string
	DefaultCWD     string
}

// PathDirectoryEntry 是 path completion 查询返回的 core-native 目录项。
type PathDirectoryEntry struct {
	Name string
	Path string
}

// PathDirectories 是一次 path completion 窗口，不拥有文件系统 truth。
type PathDirectories struct {
	BasePath  string
	Entries   []PathDirectoryEntry
	Missing   bool
	Truncated bool
}

// TerminalAttachmentMode 是 core attachment registry 的交互权限模式。
type TerminalAttachmentMode string

const (
	// TerminalAttachmentModeCollaborator 允许 input，并参与 resize arbitration。
	TerminalAttachmentModeCollaborator TerminalAttachmentMode = "collaborator"
	// TerminalAttachmentModeObserver 只观察 terminal 输出。
	TerminalAttachmentModeObserver TerminalAttachmentMode = "observer"
)

// TerminalResizePolicy 是 core attachment registry 的 resize 角色。
type TerminalResizePolicy string

const (
	// TerminalResizePolicyOwner 请求成为 resize owner。
	TerminalResizePolicyOwner TerminalResizePolicy = "owner"
	// TerminalResizePolicyFollower 跟随当前 owner size。
	TerminalResizePolicyFollower TerminalResizePolicy = "follower"
	// TerminalResizePolicyObserver 不参与 resize ownership。
	TerminalResizePolicyObserver TerminalResizePolicy = "observer"
)

// TerminalResizeReason 解释 daemon 返回的 resize control 决策。
type TerminalResizeReason string

const (
	// TerminalResizeReasonOwner 表示当前 attachment 是 owner。
	TerminalResizeReasonOwner TerminalResizeReason = "owner"
	// TerminalResizeReasonFollower 表示当前 attachment 跟随 owner。
	TerminalResizeReasonFollower TerminalResizeReason = "follower"
	// TerminalResizeReasonObserver 表示 observer 不允许 resize。
	TerminalResizeReasonObserver TerminalResizeReason = "observer"
	// TerminalResizeReasonSizeLocked 表示 owner size 被显式锁定。
	TerminalResizeReasonSizeLocked TerminalResizeReason = "size_locked"
)

// TerminalAttachmentRequest 是建立 daemon attachment 的 core-native 输入。
type TerminalAttachmentRequest struct {
	TerminalID   string
	Mode         TerminalAttachmentMode
	ResizePolicy TerminalResizePolicy
	SurfaceID    string
	ViewID       string
}

// TerminalResizeOwnership 是 daemon attachment registry 的 resize owner 投影。
type TerminalResizeOwnership struct {
	OwnerAttachmentID string
	OwnerSurfaceID    string
	OwnerViewID       string
	Size              Size
	SizeLocked        bool
	Epoch             uint64
}

// TerminalResizeControl 是 resize arbitration 的 core-native 结果。
type TerminalResizeControl struct {
	CanResize       bool
	Reason          TerminalResizeReason
	SizeLocked      bool
	SurfaceID       string
	OwnerSurfaceID  string
	OwnerViewID     string
	ResizeOwnership *TerminalResizeOwnership
}

// TerminalAttachment 是尚未或已经发布的 daemon attachment 投影。
// Token 由 owning protocol session registry 验证，调用方只能原样传回。
type TerminalAttachment struct {
	Token         []byte
	TerminalID    string
	Mode          TerminalAttachmentMode
	ResizePolicy  TerminalResizePolicy
	SurfaceID     string
	ViewID        string
	Size          Size
	ResizeControl *TerminalResizeControl
}

// TerminalAttachmentTransaction 持有尚未发布的 attachment。
// API Layer 完成公开 handle 校验后调用 Commit，失败路径调用 Rollback。
type TerminalAttachmentTransaction interface {
	Result() TerminalAttachment
	Commit(context.Context) error
	Rollback(context.Context) error
}

// TerminalResizeResult 是 resize/lock 操作后的 daemon authoritative 状态。
type TerminalResizeResult struct {
	Size          Size
	Resized       bool
	ResizeControl *TerminalResizeControl
}

// ApplicationSessionPort 是单条 protocol connection 暴露给 API Layer 的 core-native 窄边界。
// 实现绑定 immutable transport scope 和 session-owned resource registry，不包含 Proto 字段转换。
type ApplicationSessionPort interface {
	// AcquireApplication 原子校验 connection scope，并返回覆盖 controller 执行期的 lease。
	AcquireApplication(context.Context, ApplicationAdmission) (ApplicationAdmissionLease, error)
	// CancelApplicationOperation 取消当前 session owning operation；未发布能力时必须 fail closed。
	CancelApplicationOperation(context.Context, string) error
	// ReleaseApplicationResource 按 opaque token 释放当前 session owning resource。
	ReleaseApplicationResource(context.Context, []byte) error
	// ApplicationTerminalDefaults 返回 owning daemon 机器的 shell/cwd 默认值。
	ApplicationTerminalDefaults(context.Context) (TerminalDefaults, error)
	// ApplicationTerminalCreate 把 core record 交给 terminal lifecycle owner。
	ApplicationTerminalCreate(context.Context, TerminalRecord) (TerminalInfo, error)
	// ApplicationTerminalList 返回按当前 connection scope 过滤的 terminal inventory。
	ApplicationTerminalList(context.Context) ([]TerminalInfo, error)
	// ApplicationTerminalGet 返回单个 daemon-local terminal snapshot。
	ApplicationTerminalGet(context.Context, string) (TerminalInfo, error)
	// ApplicationTerminalAttachmentCount 返回 daemon registry 中的活动 attachment 数量。
	ApplicationTerminalAttachmentCount(string) int
	// ApplicationTerminalRestart 按保存的 process specification 重启 terminal。
	ApplicationTerminalRestart(context.Context, string) error
	// ApplicationTerminalKill 终止 process，但保留 record/history。
	ApplicationTerminalKill(context.Context, string) error
	// ApplicationTerminalRemove 删除满足 lifecycle 条件的 terminal record。
	ApplicationTerminalRemove(context.Context, string) error
	// ApplicationTerminalSetMetadata 原子更新 terminal name/tags。
	ApplicationTerminalSetMetadata(context.Context, string, string, map[string]string) error
	// ApplicationTerminalSetTags 替换 tags，同时保留 daemon-owned name。
	ApplicationTerminalSetTags(context.Context, string, map[string]string) error
	// ApplicationTerminalAttach 创建 pending attachment transaction。
	ApplicationTerminalAttach(context.Context, TerminalAttachmentRequest) (TerminalAttachmentTransaction, error)
	// ApplicationTerminalDetach 释放 opaque token owning attachment。
	ApplicationTerminalDetach(context.Context, []byte) error
	// ApplicationTerminalInput 向 token owning attachment 写入 bytes，失败不得隐式重放。
	ApplicationTerminalInput(context.Context, []byte, []byte) error
	// ApplicationTerminalResize 协调 resize ownership 并返回 authoritative control。
	ApplicationTerminalResize(context.Context, []byte, Size, TerminalResizePolicy) (TerminalResizeResult, error)
	// ApplicationTerminalResizeLock 修改 owner size lock 并返回最终 control。
	ApplicationTerminalResizeLock(context.Context, []byte, bool) (TerminalResizeResult, error)
	// ApplicationPathListDirectories 查询 owning daemon 文件系统目录候选。
	ApplicationPathListDirectories(context.Context, string, int) (PathDirectories, error)
	// ApplicationHistoryWindow 从 core authoritative history truth 查询 tokenized window。
	ApplicationHistoryWindow(context.Context, history.HistoryWindowRequest) (history.HistoryWindow, error)
	// ApplicationHistoryCopy 从 frozen history token 复制文本。
	ApplicationHistoryCopy(context.Context, history.HistoryCopyRequest) (string, error)
	ApplicationHistoryCopyChunk(context.Context, history.HistoryCopyChunkRequest) (history.HistoryCopyChunkResult, error)
	// ApplicationHistorySearch searches a frozen history token and returns one match window.
	ApplicationHistorySearch(context.Context, history.HistorySearchRequest) (history.HistorySearchResult, error)
	// ApplicationHistoryRelease 释放当前 terminal 的 frozen history token。
	ApplicationHistoryRelease(context.Context, string, history.HistoryToken) error
	// ApplicationHistoryBacklogStatus 返回 history ingest 的诊断状态。
	ApplicationHistoryBacklogStatus(context.Context, string) (HistoryBacklogStatus, error)
	// ApplicationLiveScreenNext 返回 observed revision 之后的 daemon latest screen；
	// observed 等于当前 revision 时等待下一次变化。
	ApplicationLiveScreenNext(context.Context, string, LiveRevision) (NativeScreenSnapshot, error)
	// ApplicationEventSubscribe 建立 session-owned application event subscription。
	ApplicationEventSubscribe(context.Context, EventFilter, ApplicationEventEncoder) ([]byte, error)
	// ApplicationFileList 返回 daemon-owned directory window。
	ApplicationFileList(context.Context, FileListRequest) (FileListResult, error)
	// ApplicationFileStat 返回 daemon-owned path metadata。
	ApplicationFileStat(context.Context, FilePathRequest) (FileEntry, error)
	// ApplicationFilePreview 返回有界文件内容预览。
	ApplicationFilePreview(context.Context, FilePreviewRequest) (FilePreviewResult, error)
	// ApplicationFileMkdir 创建 daemon-owned directory。
	ApplicationFileMkdir(context.Context, FilePathRequest) FileOperationResult
	// ApplicationFileRename 原子重命名 daemon-owned path。
	ApplicationFileRename(context.Context, FileRenameRequest) FileOperationResult
	// ApplicationFileDelete 删除 daemon-owned path。
	ApplicationFileDelete(context.Context, FilePathRequest) FileOperationResult
	// ApplicationFileCopy 批量复制 daemon-owned paths。
	ApplicationFileCopy(context.Context, FileCopyMoveRequest) FileBatchResult
	// ApplicationFileMove 批量移动 daemon-owned paths。
	ApplicationFileMove(context.Context, FileCopyMoveRequest) FileBatchResult
	// ApplicationFileDownloadOpen 创建 session-bound download transfer。
	ApplicationFileDownloadOpen(context.Context, FileDownloadOpenRequest) (FileTransfer, error)
	// ApplicationFileUploadOpen 创建或恢复 session-bound upload transfer。
	ApplicationFileUploadOpen(context.Context, FileUploadOpenRequest) (FileTransfer, error)
	// ApplicationFileTransferCancel 按 current-session resource 或 principal-bound upload resume 凭据取消 transfer。
	ApplicationFileTransferCancel(context.Context, FileTransferCancelRequest) (FileTransferCancelResult, error)
	// ApplicationStorageGet 返回 daemon opaque storage entry。
	ApplicationStorageGet(context.Context, string, StorageScope, string, string) (StorageEntry, error)
	// ApplicationStoragePut 执行 opaque value CAS put。
	ApplicationStoragePut(context.Context, StoragePutRequest) (StorageEntry, error)
	// ApplicationStorageDelete 执行 opaque value CAS delete。
	ApplicationStorageDelete(context.Context, StorageDeleteRequest) (StorageDeleteResult, error)
	// ApplicationStorageList 返回稳定 storage key window。
	ApplicationStorageList(context.Context, string, StorageScope, string, string) []StorageEntry
	// ApplicationClientAccessIdentity 返回 daemon DeviceIdentity 公开投影。
	ApplicationClientAccessIdentity(context.Context, []byte) (ClientAccessIdentity, error)
	// ApplicationClientAccessList 返回 daemon 持久化 grant 脱敏投影。
	ApplicationClientAccessList(context.Context) ([]ClientAccessRecord, error)
	// ApplicationClientAccessCreateTicket 原子签发并登记一次性 pairing ticket。
	ApplicationClientAccessCreateTicket(context.Context, ClientAccessTicketRequest) (ClientAccessTicket, error)
	// ApplicationClientAccessRevoke 按 grant ID 持久化撤销。
	ApplicationClientAccessRevoke(context.Context, string) (ClientAccessRecord, error)
	// ApplicationRemoteStatus 返回 remote runtime 状态。
	ApplicationRemoteStatus(context.Context) (RemoteStatus, error)
	// ApplicationRemotePairStart 启动本地配对会话。
	ApplicationRemotePairStart(context.Context, RemotePairStartRequest) (RemotePairStartResult, error)
	// ApplicationRemoteLocalEnable 启用 local remote runtime。
	ApplicationRemoteLocalEnable(context.Context, RemoteLocalEnableRequest) (RemoteLocalStatus, error)
	// ApplicationRemoteLocalStatus 返回 local remote runtime 状态。
	ApplicationRemoteLocalStatus(context.Context) (RemoteLocalStatus, error)
	// ApplicationRemoteLocalDisable 关闭 local remote runtime。
	ApplicationRemoteLocalDisable(context.Context) (RemoteLocalStatus, error)
	ApplicationRemoteCloudEdges(context.Context) (RemoteCloudEdgeSelection, error)
	ApplicationRemoteCloudPreferEdge(context.Context, string, uint64) (RemoteCloudEdgeSelection, error)
	ApplicationRemoteCloudReselectEdge(context.Context) (RemoteCloudEdgeSelection, error)
}

// ApplicationExecutor 执行 framing 已解码的公共 Proto command。
// 实现由 API Layer 注入；core protocol session 不持有具体 controller 或 mapping。
type ApplicationExecutor interface {
	// Execute 执行单个完整 Proto envelope，并始终返回带 correlation 的 result。
	Execute(context.Context, *apipb.CommandEnvelope) *apipb.ResultEnvelope
}

// ApplicationExecutorFactory 为每条 ready protocol connection 建立同寿命 API executor。
type ApplicationExecutorFactory func(ApplicationSessionPort) ApplicationExecutor
