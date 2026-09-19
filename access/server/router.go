package server

import (
	"context"
	"errors"

	"github.com/anytty/anytty/proto/access/apipb"
)

// CommandFamily 是 access 路由的能力族。Phase 1 只有 terminal provider 一项，
// 其余族暂时透传给同一个 provider（daemon 仍有这些实现），Phase 2/4 会依次
// 在 access 本地终结。
type CommandFamily uint8

const (
	// FamilyUnknown 表示未分类 command，必须 fail closed。
	FamilyUnknown CommandFamily = iota
	// FamilyTerminal 表示 terminal.* + event/path 查询（provider）。
	FamilyTerminal
	// FamilyFile 表示 file.* 与 file transfer（Phase 2 起 access/files 本地终结）。
	FamilyFile
	// FamilyProxy 表示 browser proxy（Phase 2 起 access/proxy 本地终结）。
	FamilyProxy
	// FamilyStorage 表示 storage.*（Phase 2 起 access/storage 本地终结）。
	FamilyStorage
	// FamilyAuth 表示 client_access.* / cloud.*（Phase 4 起 access 直答）。
	FamilyAuth
)

// String 返回稳定诊断标识。
func (family CommandFamily) String() string {
	switch family {
	case FamilyTerminal:
		return "terminal"
	case FamilyFile:
		return "file"
	case FamilyProxy:
		return "proxy"
	case FamilyStorage:
		return "storage"
	case FamilyAuth:
		return "auth"
	default:
		return "unknown"
	}
}

// familyOfCommand 按 command oneof 分类，不做“整段连接转发”。
// classify 只决定路由目标，不解释 command 字段。
func familyOfCommand(command *apipb.CommandEnvelope) CommandFamily {
	switch command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalCreate,
		*apipb.CommandEnvelope_TerminalList,
		*apipb.CommandEnvelope_TerminalGet,
		*apipb.CommandEnvelope_TerminalRestart,
		*apipb.CommandEnvelope_TerminalKill,
		*apipb.CommandEnvelope_TerminalRemove,
		*apipb.CommandEnvelope_TerminalSetMetadata,
		*apipb.CommandEnvelope_TerminalSetTags,
		*apipb.CommandEnvelope_TerminalAttach,
		*apipb.CommandEnvelope_TerminalDetach,
		*apipb.CommandEnvelope_TerminalInput,
		*apipb.CommandEnvelope_TerminalResize,
		*apipb.CommandEnvelope_TerminalResizeLock,
		*apipb.CommandEnvelope_TerminalDefaults,
		*apipb.CommandEnvelope_PathListDirectories,
		*apipb.CommandEnvelope_HistoryWindow,
		*apipb.CommandEnvelope_HistoryCopy,
		*apipb.CommandEnvelope_HistorySearch,
		*apipb.CommandEnvelope_HistoryRelease,
		*apipb.CommandEnvelope_HistoryBacklogStatus,
		*apipb.CommandEnvelope_LiveScreenNext,
		*apipb.CommandEnvelope_EventSubscribe,
		*apipb.CommandEnvelope_ReleaseResource,
		*apipb.CommandEnvelope_CancelOperation:
		return FamilyTerminal
	case *apipb.CommandEnvelope_FileList,
		*apipb.CommandEnvelope_FileStat,
		*apipb.CommandEnvelope_FilePreview,
		*apipb.CommandEnvelope_FileMkdir,
		*apipb.CommandEnvelope_FileRename,
		*apipb.CommandEnvelope_FileDelete,
		*apipb.CommandEnvelope_FileCopy,
		*apipb.CommandEnvelope_FileMove,
		*apipb.CommandEnvelope_FileDownloadOpen,
		*apipb.CommandEnvelope_FileUploadOpen,
		*apipb.CommandEnvelope_FileTransferCancel:
		return FamilyFile
	case *apipb.CommandEnvelope_BrowserProxyOpen:
		return FamilyProxy
	case *apipb.CommandEnvelope_StorageGet,
		*apipb.CommandEnvelope_StoragePut,
		*apipb.CommandEnvelope_StorageDelete,
		*apipb.CommandEnvelope_StorageList:
		return FamilyStorage
	case *apipb.CommandEnvelope_ClientAccessIdentity,
		*apipb.CommandEnvelope_ClientAccessList,
		*apipb.CommandEnvelope_ClientAccessTicketCreate,
		*apipb.CommandEnvelope_ClientAccessRevoke,
		*apipb.CommandEnvelope_RemoteStatus,
		*apipb.CommandEnvelope_RemotePairStart,
		*apipb.CommandEnvelope_RemoteLocalEnable,
		*apipb.CommandEnvelope_RemoteLocalStatus,
		*apipb.CommandEnvelope_RemoteLocalDisable,
		*apipb.CommandEnvelope_RemoteCloudStatus,
		*apipb.CommandEnvelope_RemoteCloudEnable,
		*apipb.CommandEnvelope_RemoteCloudDisable,
		*apipb.CommandEnvelope_RemoteCloudEdges,
		*apipb.CommandEnvelope_RemoteCloudPreferEdge,
		*apipb.CommandEnvelope_RemoteCloudReselectEdge:
		return FamilyAuth
	default:
		return FamilyUnknown
	}
}

// routeCommand 选择 command 的执行者。
//
// Phase 1：所有 family 都交给 terminal provider（daemon 仍实现文件/转发/storage/
// client_access 代理），但分类已经显式化，Phase 2/4 只需替换对应分支为
// access 本地 handler，不需要改动 session/framing 层。
func (session *session) routeCommand(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	// access-local resource 的 release 必须在本地应答，不能转发给 provider。
	if release := command.GetReleaseResource(); release != nil {
		if binding := session.localBindingForToken(release.GetResource().GetOpaqueToken()); binding != nil {
			return session.releaseLocalBinding(command, binding)
		}
	}
	switch familyOfCommand(command) {
	case FamilyUnknown:
		return nil, errUnsupportedCommand
	case FamilyStorage:
		// Phase 2：storage.* 已在 access 本地终结。
		return session.executeStorage(ctx, command)
	case FamilyFile:
		// Phase 2：file.* 与传输已在 access 本地终结。
		return session.executeFiles(ctx, command)
	case FamilyProxy:
		// Phase 2：browser proxy 从 access 主机拨号并本地终结。
		return session.executeProxy(ctx, command)
	case FamilyAuth:
		// Phase 4：client_access.* / cloud.* 由 access 直答，不再走反向 RPC。
		if session.server.cfg.Auth != nil {
			return session.executeAuth(ctx, command)
		}
		return session.executeViaProvider(ctx, command)
	default:
		return session.executeViaProvider(ctx, command)
	}
}

// errUnsupportedCommand 表示 command oneof 不在公共 API 路由表内。
var errUnsupportedCommand = errors.New("command is unsupported or missing")

// releaseLocalBinding 在 access 本地释放一个本地资源：
// browser proxy 关闭连接；download 停止发送；upload 保留断点记录与临时文件。
func (session *session) releaseLocalBinding(command *apipb.CommandEnvelope, binding *streamBinding) (*apipb.ResultEnvelope, error) {
	session.retireBinding(binding)
	envelope := newResultEnvelope(command)
	envelope.Result = &apipb.ResultEnvelope_Acknowledge{Acknowledge: &apipb.AcknowledgeResult{}}
	return envelope, nil
}
