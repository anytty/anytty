package core

import (
	"fmt"
	"strings"
	"time"
)

// TransportScope 只约束一条 protocol session 的可见能力，不保存 terminal truth。
type TransportScope struct {
	// GrantID 与 GrantExpiresAt 是远程 transport 已验证 CapabilityGrant 的规范身份和可选绝对期限。
	// GrantExpiresAt 为零表示授权持续到 daemon 主动撤销；local owner 不携带二者。
	GrantID        string
	GrantExpiresAt time.Time
	// PrincipalID 标识当前已验证授权主体，仅用于绑定可恢复资源；不得作为 terminal 或账号 truth。
	// local listener 使用固定 local principal，remote DataChannel 使用已签名 subject key fingerprint。
	PrincipalID string
	// LocalOwner 表示该 session 来自当前用户拥有的 daemon-local listener。
	// 它只由 core 本地 listener 设置，用于 DeviceIdentity、pairing ticket 和 client access 管理；远程 grant 永远不能声明该字段。
	LocalOwner bool
	// AllowDaemon 表示该 session 拥有当前 daemon 的完整 protocol 能力。
	// 该字段必须由本地 listener 或已验证的 daemon-level capability 显式设置；零值不能代表无限权限。
	AllowDaemon bool
	// TerminalID 把 session 限制到单个 daemon-local terminal。
	// terminal lifecycle 和 history truth 仍由 core-v2 持有，scope 只在 protocol method/stream 入口执行授权。
	TerminalID string
	// MachineEventsOnly 只允许订阅 daemon 的受限 terminal lifecycle 事件。
	// 它不能与 AllowDaemon 或 TerminalID 组合，也不能访问 storage、history、input 或 terminal management method。
	MachineEventsOnly bool
	// FileReadMetadata 允许读取 daemon 文件系统的目录项和 lstat metadata。
	// 权限必须由 local listener 或已验证 grant 显式赋予，不能从 AllowDaemon 推导。
	FileReadMetadata bool
	// FileReadContent 允许有界预览文件内容；它不包含上传或 mutation 权限。
	FileReadContent bool
	// FileMutate 允许 mkdir、rename、delete、copy 和 move。
	FileMutate bool
	// FileWriteContent 允许创建、恢复并完成上传 transfer；它不隐含 mutation 权限。
	FileWriteContent bool
	// ManageClientAccess 允许调用 remote.access.* 管理其他客户端的 bound grant。
	// 它只能来自 CapabilityGrant v2 的显式 signed scope；AllowDaemon、terminal 或 file 权限均不能隐式推出该能力。
	ManageClientAccess bool
}

func fullDaemonTransportScope() TransportScope {
	return TransportScope{PrincipalID: "local", LocalOwner: true, AllowDaemon: true, FileReadMetadata: true, FileReadContent: true, FileWriteContent: true, FileMutate: true, ManageClientAccess: true}
}

func (scope TransportScope) normalized() TransportScope {
	scope.GrantID = strings.TrimSpace(scope.GrantID)
	scope.GrantExpiresAt = scope.GrantExpiresAt.UTC()
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	scope.TerminalID = strings.TrimSpace(scope.TerminalID)
	return scope
}

func (scope TransportScope) validate() error {
	if scope.LocalOwner {
		if scope.GrantID != "" || !scope.GrantExpiresAt.IsZero() {
			return fmt.Errorf("local owner transport cannot carry remote grant")
		}
	} else if scope.GrantID == "" {
		return fmt.Errorf("remote transport requires grant identity")
	}
	capabilities := 0
	if scope.AllowDaemon {
		capabilities++
	}
	if scope.TerminalID != "" {
		capabilities++
	}
	if scope.MachineEventsOnly {
		capabilities++
	}
	if capabilities == 0 {
		return fmt.Errorf("transport scope requires explicit capability")
	}
	if capabilities != 1 {
		return fmt.Errorf("transport scope capabilities are mutually exclusive")
	}
	if (scope.FileReadMetadata || scope.FileReadContent || scope.FileWriteContent || scope.FileMutate) && !scope.AllowDaemon {
		return fmt.Errorf("file permissions require daemon scope")
	}
	if (scope.FileReadMetadata || scope.FileReadContent || scope.FileWriteContent || scope.FileMutate) && scope.PrincipalID == "" {
		return fmt.Errorf("file permissions require verified principal")
	}
	if scope.LocalOwner && (!scope.AllowDaemon || scope.PrincipalID == "") {
		return fmt.Errorf("local owner scope requires verified daemon capability")
	}
	if scope.ManageClientAccess && scope.PrincipalID == "" {
		return fmt.Errorf("client access management requires verified principal")
	}
	return nil
}

func (scope TransportScope) unrestricted() bool {
	return scope.AllowDaemon
}

func (scope TransportScope) requireTerminal(method string, terminalID string) error {
	if terminalID == "" {
		return fmt.Errorf("transport scope terminal %q denies %s without terminal_id", scope.TerminalID, method)
	}
	if terminalID != scope.TerminalID {
		return fmt.Errorf("transport scope terminal %q denies %s for terminal %q", scope.TerminalID, method, terminalID)
	}
	return nil
}

func (scope TransportScope) allowsAttachment(attachment protocolAttachment) error {
	if scope.unrestricted() {
		return nil
	}
	if scope.MachineEventsOnly {
		return fmt.Errorf("transport scope machine-events-only denies stream channel %d", attachment.Channel)
	}
	return scope.requireTerminal("stream", attachment.TerminalID)
}
