package apilayer

import (
	"context"
	"errors"

	apimapping "github.com/anytty/anytty/api_mapping"
	corev2 "github.com/anytty/anytty/core"
	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

// CoreApplicationExecutorFactory 为一条 ready protocol connection 装配 API Layer。
// core 只提供 connection-bound native port；Proto validation、mapping 与错误分类均停留在本层。
func CoreApplicationExecutorFactory(port corev2.ApplicationSessionPort) corev2.ApplicationExecutor {
	adapter := &coreApplicationAdapter{port: port}
	return NewPlatformService(adapter, adapter, adapter, adapter, adapter)
}

type coreApplicationAdapter struct {
	port corev2.ApplicationSessionPort
}

func (adapter *coreApplicationAdapter) Acquire(ctx context.Context, command *apipb.CommandEnvelope, capability apipb.ApiCapability) (AdmissionLease, error) {
	lease, err := adapter.port.AcquireApplication(ctx, apimapping.ApplicationAdmissionFromCommand(command, capability))
	if err == nil {
		return lease, nil
	}
	switch {
	case errors.Is(err, corev2.ErrApplicationForbidden):
		return nil, ErrAdmissionForbidden
	case errors.Is(err, corev2.ErrApplicationUnsupportedCapability):
		return nil, ErrAdmissionUnsupportedCapability
	default:
		return nil, apimapping.CoreError(err)
	}
}

func (adapter *coreApplicationAdapter) CancelOperation(ctx context.Context, operation *apipb.OperationStamp) error {
	return apimapping.CoreError(adapter.port.CancelApplicationOperation(ctx, operation.GetOperationId()))
}

func (adapter *coreApplicationAdapter) ReleaseResource(ctx context.Context, resource *apipb.ResourceHandle) error {
	return apimapping.CoreError(adapter.port.ReleaseApplicationResource(ctx, resource.GetOpaqueToken()))
}

func (adapter *coreApplicationAdapter) TerminalDefaults(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.TerminalDefaultsCommand) (*apipb.TerminalDefaultsResult, error) {
	defaults, err := adapter.port.ApplicationTerminalDefaults(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.TerminalDefaultsToProto(defaults), nil
}

func (adapter *coreApplicationAdapter) TerminalCreate(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.TerminalCreateCommand) (*apipb.TerminalCreateResult, error) {
	record, err := apimapping.TerminalRecordFromProto(command.GetTerminal())
	if err != nil {
		return nil, err
	}
	info, err := adapter.port.ApplicationTerminalCreate(ctx, record)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	projection, err := apimapping.TerminalInfoToProto(origin.GetEndpointId(), info, adapter.port.ApplicationTerminalAttachmentCount(info.ID))
	if err != nil {
		return nil, err
	}
	return apimapping.TerminalCreateToProto(projection), nil
}

func (adapter *coreApplicationAdapter) TerminalList(ctx context.Context, origin *apipb.EndpointSessionStamp, _ *apipb.TerminalListCommand) (*apipb.TerminalListResult, error) {
	items, err := adapter.port.ApplicationTerminalList(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	terminals := make([]*apipb.TerminalInfo, 0, len(items))
	for _, item := range items {
		projection, err := apimapping.TerminalInfoToProto(origin.GetEndpointId(), item, adapter.port.ApplicationTerminalAttachmentCount(item.ID))
		if err != nil {
			return nil, err
		}
		terminals = append(terminals, projection)
	}
	return apimapping.TerminalListToProto(terminals), nil
}

func (adapter *coreApplicationAdapter) TerminalGet(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.TerminalGetCommand) (*apipb.TerminalGetResult, error) {
	info, err := adapter.port.ApplicationTerminalGet(ctx, command.GetTerminal().GetTerminalId())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	projection, err := apimapping.TerminalInfoToProto(origin.GetEndpointId(), info, adapter.port.ApplicationTerminalAttachmentCount(info.ID))
	if err != nil {
		return nil, err
	}
	return apimapping.TerminalGetToProto(projection), nil
}

func (adapter *coreApplicationAdapter) TerminalRestart(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalRestartCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalRestart(ctx, command.GetTerminal().GetTerminalId()))
}

func (adapter *coreApplicationAdapter) TerminalKill(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalKillCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalKill(ctx, command.GetTerminal().GetTerminalId()))
}

func (adapter *coreApplicationAdapter) TerminalRemove(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalRemoveCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalRemove(ctx, command.GetTerminal().GetTerminalId()))
}

func (adapter *coreApplicationAdapter) TerminalSetMetadata(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalSetMetadataCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalSetMetadata(ctx, command.GetTerminal().GetTerminalId(), command.GetName(), command.GetTags()))
}

func (adapter *coreApplicationAdapter) TerminalSetTags(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalSetTagsCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalSetTags(ctx, command.GetTerminal().GetTerminalId(), command.GetTags()))
}

func (adapter *coreApplicationAdapter) TerminalAttach(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.TerminalAttachCommand) (TerminalAttachTransaction, error) {
	transaction, err := adapter.port.ApplicationTerminalAttach(ctx, apimapping.TerminalAttachmentRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return &coreAttachmentTransaction{
		transaction: transaction,
		origin:      proto.Clone(origin).(*apipb.EndpointSessionStamp),
		command:     proto.Clone(command).(*apipb.TerminalAttachCommand),
	}, nil
}

func (adapter *coreApplicationAdapter) TerminalDetach(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalDetachCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalDetach(ctx, command.GetAttachment().GetOpaqueToken()))
}

func (adapter *coreApplicationAdapter) TerminalInput(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalInputCommand) error {
	return apimapping.CoreError(adapter.port.ApplicationTerminalInput(ctx, command.GetAttachment().GetOpaqueToken(), command.GetData()))
}

func (adapter *coreApplicationAdapter) TerminalResize(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalResizeCommand) (*apipb.TerminalResizeResult, error) {
	result, err := adapter.port.ApplicationTerminalResize(ctx, command.GetAttachment().GetOpaqueToken(), apimapping.TerminalSizeFromProto(command.GetSize()), apimapping.ResizePolicyToCore(command.GetResizePolicy()), command.GetTakeOwnership(), command.GetExpectedOwnerEpoch())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.TerminalResizeResultToProto(result), nil
}

func (adapter *coreApplicationAdapter) TerminalResizeLock(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.TerminalResizeLockCommand) (*apipb.TerminalResizeResult, error) {
	result, err := adapter.port.ApplicationTerminalResizeLock(ctx, command.GetAttachment().GetOpaqueToken(), command.GetLocked())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.TerminalResizeResultToProto(result), nil
}

func (adapter *coreApplicationAdapter) PathListDirectories(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.PathListDirectoriesCommand) (*apipb.PathListDirectoriesResult, error) {
	result, err := adapter.port.ApplicationPathListDirectories(ctx, command.GetPrefix(), int(command.GetLimit()))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.PathDirectoriesToProto(result), nil
}

func (adapter *coreApplicationAdapter) HistoryWindow(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	result, err := adapter.port.ApplicationHistoryWindow(ctx, apimapping.HistoryWindowRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.HistoryWindowToProto(origin.GetEndpointId(), result), nil
}

func (adapter *coreApplicationAdapter) HistoryCopy(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	if command.GetMaxLines() > 0 || command.GetMaxBytes() > 0 {
		result, err := adapter.port.ApplicationHistoryCopyChunk(ctx, apimapping.HistoryCopyChunkRequestFromProto(command))
		if err != nil {
			return nil, apimapping.CoreError(err)
		}
		return apimapping.HistoryCopyChunkToProto(result), nil
	}
	text, err := adapter.port.ApplicationHistoryCopy(ctx, apimapping.HistoryCopyRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.HistoryCopyToProto(text), nil
}

func (adapter *coreApplicationAdapter) HistorySearch(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.HistorySearchCommand) (*apipb.HistorySearchResult, error) {
	result, err := adapter.port.ApplicationHistorySearch(ctx, apimapping.HistorySearchRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.HistorySearchToProto(origin.GetEndpointId(), result), nil
}

func (adapter *coreApplicationAdapter) HistoryRelease(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.HistoryReleaseCommand) (*apipb.AcknowledgeResult, error) {
	if err := adapter.port.ApplicationHistoryRelease(ctx, command.GetTerminal().GetTerminalId(), history.HistoryToken(command.GetToken())); err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.AcknowledgeToProto(), nil
}

func (adapter *coreApplicationAdapter) HistoryBacklogStatus(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.HistoryBacklogStatusCommand) (*apipb.HistoryBacklogStatusResult, error) {
	result, err := adapter.port.ApplicationHistoryBacklogStatus(ctx, command.GetTerminal().GetTerminalId())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.HistoryBacklogToProto(origin.GetEndpointId(), result), nil
}

func (adapter *coreApplicationAdapter) LiveScreenNext(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.LiveScreenNextCommand) (*apipb.NativeScreenResult, error) {
	snapshot, err := adapter.port.ApplicationLiveScreenNext(ctx, command.GetTerminal().GetTerminalId(), corev2.LiveRevision(command.GetObservedRevision()))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.NativeScreenToProto(origin.GetEndpointId(), snapshot), nil
}

func (adapter *coreApplicationAdapter) EventSubscribe(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.EventSubscribeCommand) (*apipb.EventSubscriptionResult, error) {
	eventOrigin := proto.Clone(origin).(*apipb.EndpointSessionStamp)
	token, err := adapter.port.ApplicationEventSubscribe(ctx, apimapping.EventFilterFromProto(command), func(event corev2.Event, subscriptionToken []byte) ([]byte, error) {
		return apimapping.EncodeEventEnvelope(eventOrigin.GetEndpointId(), eventOrigin, subscriptionToken, event)
	})
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.EventSubscriptionToProto(origin, token), nil
}

type coreAttachmentTransaction struct {
	transaction corev2.TerminalAttachmentTransaction
	origin      *apipb.EndpointSessionStamp
	command     *apipb.TerminalAttachCommand
}

func (transaction *coreAttachmentTransaction) Result() *apipb.TerminalAttachResult {
	if transaction == nil || transaction.transaction == nil {
		return nil
	}
	return apimapping.TerminalAttachmentToProto(transaction.origin, transaction.command, transaction.transaction.Result())
}

func (transaction *coreAttachmentTransaction) Commit(ctx context.Context) error {
	return apimapping.CoreError(transaction.transaction.Commit(ctx))
}

func (transaction *coreAttachmentTransaction) Rollback(ctx context.Context) error {
	return apimapping.CoreError(transaction.transaction.Rollback(ctx))
}

func (adapter *coreApplicationAdapter) FileList(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileListCommand) (*apipb.FileListResult, error) {
	result, err := adapter.port.ApplicationFileList(ctx, apimapping.FileListRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.FileListToProto(result), nil
}
func (adapter *coreApplicationAdapter) FileStat(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileStatCommand) (*apipb.FileStatResult, error) {
	result, err := adapter.port.ApplicationFileStat(ctx, apimapping.FilePathRequestFromProto(command.GetPath(), false))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.FileStatToProto(result), nil
}
func (adapter *coreApplicationAdapter) FilePreview(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FilePreviewCommand) (*apipb.FilePreviewResult, error) {
	result, err := adapter.port.ApplicationFilePreview(ctx, apimapping.FilePreviewRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.FilePreviewToProto(result), nil
}
func (adapter *coreApplicationAdapter) FileMkdir(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileMkdirCommand) (*apipb.FileOperationResult, error) {
	return apimapping.FileOperationToProto(adapter.port.ApplicationFileMkdir(ctx, apimapping.FilePathRequestFromProto(command.GetPath(), command.GetRecursive()))), nil
}
func (adapter *coreApplicationAdapter) FileRename(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileRenameCommand) (*apipb.FileOperationResult, error) {
	return apimapping.FileOperationToProto(adapter.port.ApplicationFileRename(ctx, apimapping.FileRenameRequestFromProto(command))), nil
}
func (adapter *coreApplicationAdapter) FileDelete(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileDeleteCommand) (*apipb.FileOperationResult, error) {
	return apimapping.FileOperationToProto(adapter.port.ApplicationFileDelete(ctx, apimapping.FilePathRequestFromProto(command.GetPath(), command.GetRecursive()))), nil
}
func (adapter *coreApplicationAdapter) FileCopy(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileCopyCommand) (*apipb.FileBatchResult, error) {
	return apimapping.FileBatchToProto(adapter.port.ApplicationFileCopy(ctx, apimapping.FileCopyRequestFromProto(command))), nil
}
func (adapter *coreApplicationAdapter) FileMove(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileMoveCommand) (*apipb.FileBatchResult, error) {
	return apimapping.FileBatchToProto(adapter.port.ApplicationFileMove(ctx, apimapping.FileMoveRequestFromProto(command))), nil
}
func (adapter *coreApplicationAdapter) FileDownloadOpen(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.FileDownloadOpenCommand) (*apipb.FileTransferOpenResult, error) {
	result, err := adapter.port.ApplicationFileDownloadOpen(ctx, apimapping.FileDownloadRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.FileTransferToProto(origin, command.GetOperation(), result), nil
}
func (adapter *coreApplicationAdapter) FileUploadOpen(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.FileUploadOpenCommand) (*apipb.FileTransferOpenResult, error) {
	result, err := adapter.port.ApplicationFileUploadOpen(ctx, apimapping.FileUploadRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.FileTransferToProto(origin, command.GetOperation(), result), nil
}
func (adapter *coreApplicationAdapter) FileTransferCancel(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.FileTransferCancelCommand) (*apipb.FileTransferCancelResult, error) {
	result, err := adapter.port.ApplicationFileTransferCancel(ctx, apimapping.FileTransferCancelRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.FileTransferCancelToProto(result), nil
}

func (adapter *coreApplicationAdapter) BrowserProxyOpen(ctx context.Context, origin *apipb.EndpointSessionStamp, command *apipb.BrowserProxyOpenCommand) (*apipb.BrowserProxyOpenResult, error) {
	proxy, err := adapter.port.ApplicationBrowserProxyOpen(ctx, command.GetHost(), uint16(command.GetPort()))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.BrowserProxyToProto(origin, proxy), nil
}
func (adapter *coreApplicationAdapter) StorageGet(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.StorageGetCommand) (*apipb.StorageGetResult, error) {
	appID, scope, ownerID, key := apimapping.StorageKeyFromProto(command.GetKey())
	result, err := adapter.port.ApplicationStorageGet(ctx, appID, scope, ownerID, key)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.StorageGetToProto(result), nil
}
func (adapter *coreApplicationAdapter) StoragePut(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.StoragePutCommand) (*apipb.StoragePutResult, error) {
	result, err := adapter.port.ApplicationStoragePut(ctx, apimapping.StoragePutFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.StoragePutToProto(result), nil
}
func (adapter *coreApplicationAdapter) StorageDelete(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.StorageDeleteCommand) (*apipb.StorageDeleteResult, error) {
	result, err := adapter.port.ApplicationStorageDelete(ctx, apimapping.StorageDeleteFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.StorageDeleteToProto(result), nil
}
func (adapter *coreApplicationAdapter) StorageList(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.StorageListCommand) (*apipb.StorageListResult, error) {
	return apimapping.StorageListToProto(adapter.port.ApplicationStorageList(ctx, command.GetAppId(), apimapping.StorageScopeFromProto(command.GetScope()), command.GetOwnerId(), command.GetPrefix())), nil
}

func (adapter *coreApplicationAdapter) ClientAccessIdentity(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.ClientAccessIdentityCommand) (*apipb.ClientAccessIdentityResult, error) {
	result, err := adapter.port.ApplicationClientAccessIdentity(ctx, command.GetChallenge())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.ClientAccessIdentityResultToProto(result), nil
}

func (adapter *coreApplicationAdapter) ClientAccessList(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.ClientAccessListCommand) (*apipb.ClientAccessListResult, error) {
	records, err := adapter.port.ApplicationClientAccessList(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.ClientAccessListToProto(records), nil
}

func (adapter *coreApplicationAdapter) ClientAccessTicketCreate(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.ClientAccessTicketCreateCommand) (*apipb.ClientAccessTicketCreateResult, error) {
	result, err := adapter.port.ApplicationClientAccessCreateTicket(ctx, apimapping.ClientAccessTicketRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.ClientAccessTicketResultToProto(result), nil
}

func (adapter *coreApplicationAdapter) ClientAccessRevoke(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.ClientAccessRevokeCommand) (*apipb.ClientAccessRevokeResult, error) {
	result, err := adapter.port.ApplicationClientAccessRevoke(ctx, command.GetRequest().GetGrantId())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.ClientAccessRevokeToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteStatus(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteStatusCommand) (*apipb.RemoteStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteStatus(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemotePairStart(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.RemotePairStartCommand) (*apipb.RemotePairStartResult, error) {
	result, err := adapter.port.ApplicationRemotePairStart(ctx, apimapping.RemotePairStartRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemotePairStartToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteLocalEnable(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.RemoteLocalEnableCommand) (*apipb.RemoteLocalStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteLocalEnable(ctx, apimapping.RemoteLocalEnableRequestFromProto(command))
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteLocalStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteLocalStatus(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteLocalStatusCommand) (*apipb.RemoteLocalStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteLocalStatus(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteLocalStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteLocalDisable(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteLocalDisableCommand) (*apipb.RemoteLocalStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteLocalDisable(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteLocalStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteCloudStatus(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteCloudStatusCommand) (*apipb.RemoteCloudStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteCloudStatus(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteCloudStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteCloudEnable(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteCloudEnableCommand) (*apipb.RemoteCloudStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteCloudEnable(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteCloudStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteCloudDisable(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteCloudDisableCommand) (*apipb.RemoteCloudStatusResult, error) {
	result, err := adapter.port.ApplicationRemoteCloudDisable(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteCloudStatusToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteCloudEdges(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteCloudEdgesCommand) (*apipb.RemoteCloudEdgesResult, error) {
	result, err := adapter.port.ApplicationRemoteCloudEdges(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteCloudEdgeSelectionToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteCloudPreferEdge(ctx context.Context, _ *apipb.EndpointSessionStamp, command *apipb.RemoteCloudPreferEdgeCommand) (*apipb.RemoteCloudEdgesResult, error) {
	result, err := adapter.port.ApplicationRemoteCloudPreferEdge(ctx, command.GetEdgeId(), command.GetExpectedPreferenceRevision())
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteCloudEdgeSelectionToProto(result), nil
}

func (adapter *coreApplicationAdapter) RemoteCloudReselectEdge(ctx context.Context, _ *apipb.EndpointSessionStamp, _ *apipb.RemoteCloudReselectEdgeCommand) (*apipb.RemoteCloudEdgesResult, error) {
	result, err := adapter.port.ApplicationRemoteCloudReselectEdge(ctx)
	if err != nil {
		return nil, apimapping.CoreError(err)
	}
	return apimapping.RemoteCloudEdgeSelectionToProto(result), nil
}
