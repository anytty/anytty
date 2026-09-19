package core

import "errors"

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
	// ApplicationCapabilityBrowserProxy 表示 session-bound daemon-side TCP proxy。
	ApplicationCapabilityBrowserProxy
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

// TerminalDefaults 是 daemon 默认 shell 与 cwd 的 core-native 查询结果。
type TerminalDefaults struct {
	DefaultCommand []string
	DefaultCWD     string
	Platform       string
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
// Token 由 owning session registry 验证，调用方只能原样传回。
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

// TerminalResizeResult 是 resize/lock 操作后的 daemon authoritative 状态。
type TerminalResizeResult struct {
	Size          Size
	Resized       bool
	ResizeControl *TerminalResizeControl
}

// BrowserProxy 是一个已经由 daemon 拨号并绑定到当前 session 的 TCP resource。
type BrowserProxy struct {
	Token              []byte
	ReceiveWindowBytes uint32
	SendWindowBytes    uint32
}

// BrowserProxyMaximumReceiveWindow 是 browser proxy 允许协商的最大窗口。
const BrowserProxyMaximumReceiveWindow uint32 = 1 << 20
