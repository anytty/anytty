package testkit

import (
	"context"
	"sync"

	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
)

type FakeCoreClient struct {
	LatestResponses []port.HistoryResult
	OlderResponses  []port.HistoryResult
	NewerResponses  []port.HistoryResult
	OldestResponses []port.HistoryResult
	CopyResponses   []port.HistoryCopyRangeResult
	SearchResponses []port.HistorySearchResult
	LatestRequests  []port.HistoryLatestRequest
	OlderRequests   []port.HistoryOlderRequest
	NewerRequests   []port.HistoryNewerRequest
	OldestRequests  []port.HistoryOldestRequest
	CopyRequests    []port.HistoryCopyRangeRequest
	SearchRequests  []port.HistorySearchRequest
	ReleaseRequests []port.HistoryReleaseRequest
	ReleaseErr      error
}

func (client *FakeCoreClient) HistoryLatest(_ context.Context, req port.HistoryLatestRequest) (port.HistoryResult, error) {
	client.LatestRequests = append(client.LatestRequests, req)
	if len(client.LatestResponses) == 0 {
		return port.HistoryResult{}, port.ErrMissingHistoryResponse
	}
	result := client.LatestResponses[0]
	client.LatestResponses = client.LatestResponses[1:]
	result.RequestID = req.RequestID
	return result, nil
}

func (client *FakeCoreClient) HistoryOlder(_ context.Context, req port.HistoryOlderRequest) (port.HistoryResult, error) {
	client.OlderRequests = append(client.OlderRequests, req)
	if len(client.OlderResponses) == 0 {
		return port.HistoryResult{}, port.ErrMissingHistoryResponse
	}
	result := client.OlderResponses[0]
	client.OlderResponses = client.OlderResponses[1:]
	result.RequestID = req.RequestID
	return result, nil
}

func (client *FakeCoreClient) HistoryNewer(_ context.Context, req port.HistoryNewerRequest) (port.HistoryResult, error) {
	client.NewerRequests = append(client.NewerRequests, req)
	if len(client.NewerResponses) == 0 {
		return port.HistoryResult{}, port.ErrMissingHistoryResponse
	}
	result := client.NewerResponses[0]
	client.NewerResponses = client.NewerResponses[1:]
	result.RequestID = req.RequestID
	return result, nil
}

func (client *FakeCoreClient) HistoryOldest(_ context.Context, req port.HistoryOldestRequest) (port.HistoryResult, error) {
	client.OldestRequests = append(client.OldestRequests, req)
	if len(client.OldestResponses) == 0 {
		return port.HistoryResult{}, port.ErrMissingHistoryResponse
	}
	result := client.OldestResponses[0]
	client.OldestResponses = client.OldestResponses[1:]
	result.RequestID = req.RequestID
	return result, nil
}

func (client *FakeCoreClient) HistoryCopyRange(_ context.Context, req port.HistoryCopyRangeRequest) (port.HistoryCopyRangeResult, error) {
	client.CopyRequests = append(client.CopyRequests, req)
	if len(client.CopyResponses) == 0 {
		return port.HistoryCopyRangeResult{}, port.ErrMissingHistoryResponse
	}
	result := client.CopyResponses[0]
	client.CopyResponses = client.CopyResponses[1:]
	return result, nil
}

func (client *FakeCoreClient) HistorySearch(_ context.Context, req port.HistorySearchRequest) (port.HistorySearchResult, error) {
	client.SearchRequests = append(client.SearchRequests, req)
	if len(client.SearchResponses) == 0 {
		return port.HistorySearchResult{RequestID: req.RequestID}, port.ErrMissingHistoryResponse
	}
	result := client.SearchResponses[0]
	client.SearchResponses = client.SearchResponses[1:]
	result.RequestID = req.RequestID
	return result, nil
}

func (client *FakeCoreClient) ReleaseHistory(_ context.Context, req port.HistoryReleaseRequest) error {
	client.ReleaseRequests = append(client.ReleaseRequests, req)
	return client.ReleaseErr
}

type FakeTerminalService struct {
	liveScreenNextMu       sync.Mutex
	liveScreenNextRequests []port.TerminalSurfaceRequest
	AttachResult           port.TerminalAttachResult
	ListResult             port.TerminalListResult
	CreateResult           port.TerminalCreateResult
	SurfaceResult          port.TerminalSurfaceResult
	AttachErr              error
	ListErr                error
	CreateErr              error
	RestartErr             error
	ReconnectErr           error
	KillErr                error
	RemoveErr              error
	EditErr                error
	EditTagsErr            error
	InputErr               error
	ResizeErr              error
	ResizeResult           port.TerminalResizeResult
	ResizeLockErr          error
	ResizeLockResult       port.TerminalResizeResult
	SurfaceErr             error
	LiveScreenNextCh       chan port.TerminalSurfaceResult
	LiveScreenNextErr      error
	PathResult             port.PathListDirectoriesResult
	PathErr                error
	PathDefaultsResult     port.PathDefaultsResult
	PathDefaultsErr        error
	Attaches               []port.TerminalAttachRequest
	Detaches               []port.TerminalDetachRequest
	Lists                  []port.TerminalListRequest
	Creates                []port.TerminalCreateRequest
	Restarts               []port.TerminalRestartRequest
	Reconnects             []port.TerminalReconnectRequest
	Kills                  []port.TerminalKillRequest
	Removes                []port.TerminalRemoveRequest
	Edits                  []port.TerminalEditMetadataRequest
	TagEdits               []port.TerminalEditTagsRequest
	Inputs                 []port.TerminalInputRequest
	Resizes                []port.TerminalResizeRequest
	ResizeLocks            []port.TerminalResizeLockRequest
	Surfaces               []port.TerminalSurfaceRequest
	PathRequests           []port.PathListDirectoriesRequest
	PathDefaultsRequests   []port.PathDefaultsRequest
}

type FakeWorkbenchStorageService struct {
	LoadResult     port.WorkbenchStorageLoadResult
	LoadErr        error
	SaveResult     port.WorkbenchStorageSaveResult
	SaveErr        error
	CurrentVersion uint64
	WatchCh        chan port.WorkbenchStorageEvent
	WatchErr       error
	Loads          []state.WorkbenchStorageRef
	Saves          []port.WorkbenchStorageSaveRequest
	Watches        []state.WorkbenchStorageRef
}

type FakeClipboardStorageService struct {
	LoadResult     port.ClipboardStorageLoadResult
	LoadErr        error
	SaveResult     port.ClipboardStorageSaveResult
	SaveErr        error
	CurrentVersion uint64
	WatchCh        chan port.ClipboardStorageEvent
	WatchErr       error
	Loads          []state.ClipboardStorageRef
	Saves          []port.ClipboardStorageSaveRequest
	Watches        []state.ClipboardStorageRef
}

func (service *FakeWorkbenchStorageService) LoadWorkbench(_ context.Context, ref state.WorkbenchStorageRef) (port.WorkbenchStorageLoadResult, error) {
	service.Loads = append(service.Loads, ref)
	if service.LoadErr != nil {
		return port.WorkbenchStorageLoadResult{}, service.LoadErr
	}
	if service.LoadResult.Found && service.CurrentVersion == 0 {
		service.CurrentVersion = service.LoadResult.Version
	}
	return service.LoadResult, nil
}

func (service *FakeWorkbenchStorageService) SaveWorkbench(_ context.Context, req port.WorkbenchStorageSaveRequest) (port.WorkbenchStorageSaveResult, error) {
	req.Snapshot = cloneWorkbenchStorageSnapshot(req.Snapshot)
	service.Saves = append(service.Saves, req)
	if service.SaveErr != nil {
		return port.WorkbenchStorageSaveResult{}, service.SaveErr
	}
	if req.CheckVersion && service.CurrentVersion != req.ExpectedVersion {
		return port.WorkbenchStorageSaveResult{}, port.ErrWorkbenchStorageConflict
	}
	result := service.SaveResult
	if result.Ref.AppID == "" {
		result.Ref = req.Ref
	}
	if result.Version == 0 {
		result.Version = req.ExpectedVersion + 1
	}
	service.CurrentVersion = result.Version
	return result, nil
}

func (service *FakeWorkbenchStorageService) WatchWorkbench(_ context.Context, ref state.WorkbenchStorageRef) (<-chan port.WorkbenchStorageEvent, error) {
	service.Watches = append(service.Watches, ref)
	if service.WatchErr != nil {
		return nil, service.WatchErr
	}
	if service.WatchCh == nil {
		service.WatchCh = make(chan port.WorkbenchStorageEvent, 16)
	}
	return service.WatchCh, nil
}

func (service *FakeClipboardStorageService) LoadClipboard(_ context.Context, ref state.ClipboardStorageRef) (port.ClipboardStorageLoadResult, error) {
	service.Loads = append(service.Loads, ref)
	if service.LoadErr != nil {
		return port.ClipboardStorageLoadResult{}, service.LoadErr
	}
	return service.LoadResult, nil
}

func (service *FakeClipboardStorageService) SaveClipboard(_ context.Context, req port.ClipboardStorageSaveRequest) (port.ClipboardStorageSaveResult, error) {
	req.Snapshot = cloneClipboardStorageSnapshot(req.Snapshot)
	service.Saves = append(service.Saves, req)
	if service.SaveErr != nil {
		return port.ClipboardStorageSaveResult{}, service.SaveErr
	}
	if req.CheckVersion && service.CurrentVersion != req.ExpectedVersion {
		return port.ClipboardStorageSaveResult{}, port.ErrClipboardStorageConflict
	}
	result := service.SaveResult
	if result.Ref.AppID == "" {
		result.Ref = req.Ref
	}
	if result.Version == 0 {
		result.Version = req.ExpectedVersion + 1
	}
	service.CurrentVersion = result.Version
	return result, nil
}

func (service *FakeClipboardStorageService) WatchClipboard(_ context.Context, ref state.ClipboardStorageRef) (<-chan port.ClipboardStorageEvent, error) {
	service.Watches = append(service.Watches, ref)
	if service.WatchErr != nil {
		return nil, service.WatchErr
	}
	if service.WatchCh == nil {
		service.WatchCh = make(chan port.ClipboardStorageEvent, 16)
	}
	return service.WatchCh, nil
}

func (service *FakeTerminalService) Attach(_ context.Context, req port.TerminalAttachRequest) (port.TerminalAttachResult, error) {
	service.Attaches = append(service.Attaches, req)
	if service.AttachErr != nil {
		return port.TerminalAttachResult{}, service.AttachErr
	}
	result := service.AttachResult
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = req.TerminalID
	}
	if result.Cols == 0 {
		result.Cols = req.Cols
	}
	if result.Rows == 0 {
		result.Rows = req.Rows
	}
	if result.ResizePolicy == "" {
		result.ResizePolicy = req.ResizePolicy
	}
	if result.SurfaceID == "" {
		result.SurfaceID = req.SurfaceID
	}
	if result.ViewID == "" {
		result.ViewID = req.ViewID
	}
	if result.Session == nil {
		result.Session = &apipb.EndpointSessionStamp{EndpointId: string(state.NormalizeEndpointID(req.EndpointID)), RouteId: "test", Generation: 1}
	}
	if result.OperationID == "" {
		result.OperationID = req.OperationID
	}
	if !result.SizeLocked && result.ControlReason == "" && result.ResizePolicy == state.TerminalResizeRoleOwner {
		result.CanResize = true
	}
	if result.CanResize && result.ResizePolicy == state.TerminalResizeRoleOwner {
		if result.OwnerSurfaceID == "" {
			result.OwnerSurfaceID = result.SurfaceID
		}
		if result.OwnerViewID == "" {
			result.OwnerViewID = result.ViewID
		}
	}
	return result, nil
}

func (service *FakeTerminalService) Detach(_ context.Context, req port.TerminalDetachRequest) error {
	service.Detaches = append(service.Detaches, req)
	return nil
}

func (service *FakeTerminalService) List(_ context.Context, req port.TerminalListRequest) (port.TerminalListResult, error) {
	service.Lists = append(service.Lists, req)
	if service.ListErr != nil {
		return port.TerminalListResult{}, service.ListErr
	}
	return port.TerminalListResult{Items: cloneTerminalPoolItems(service.ListResult.Items)}, nil
}

func (service *FakeTerminalService) Create(_ context.Context, req port.TerminalCreateRequest) (port.TerminalCreateResult, error) {
	req.Tags = cloneStringMap(req.Tags)
	req.Command = append([]string(nil), req.Command...)
	service.Creates = append(service.Creates, req)
	if service.CreateErr != nil {
		return port.TerminalCreateResult{}, service.CreateErr
	}
	result := service.CreateResult
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = req.TerminalID
	}
	if result.State == "" {
		result.State = "running"
	}
	return result, nil
}

func (service *FakeTerminalService) Restart(_ context.Context, req port.TerminalRestartRequest) error {
	service.Restarts = append(service.Restarts, req)
	return service.RestartErr
}

func (service *FakeTerminalService) Reconnect(ctx context.Context, req port.TerminalReconnectRequest) (port.TerminalAttachResult, error) {
	service.Reconnects = append(service.Reconnects, req)
	if service.ReconnectErr != nil {
		return port.TerminalAttachResult{}, service.ReconnectErr
	}
	return service.Attach(ctx, port.TerminalAttachRequest{
		EndpointID:   req.EndpointID,
		TerminalID:   req.TerminalID,
		Cols:         req.Cols,
		Rows:         req.Rows,
		Mode:         req.Mode,
		ResizePolicy: req.ResizePolicy,
		SurfaceID:    req.SurfaceID,
		ViewID:       req.ViewID,
		OperationID:  req.OperationID,
	})
}

func (service *FakeTerminalService) Kill(_ context.Context, req port.TerminalKillRequest) error {
	service.Kills = append(service.Kills, req)
	return service.KillErr
}

func (service *FakeTerminalService) Remove(_ context.Context, req port.TerminalRemoveRequest) error {
	service.Removes = append(service.Removes, req)
	return service.RemoveErr
}

func (service *FakeTerminalService) EditMetadata(_ context.Context, req port.TerminalEditMetadataRequest) error {
	service.Edits = append(service.Edits, port.TerminalEditMetadataRequest{
		EndpointID: req.EndpointID,
		TerminalID: req.TerminalID,
		Title:      req.Title,
		Tags:       cloneStringMap(req.Tags),
	})
	return service.EditErr
}

func (service *FakeTerminalService) EditTags(_ context.Context, req port.TerminalEditTagsRequest) error {
	service.TagEdits = append(service.TagEdits, port.TerminalEditTagsRequest{
		EndpointID: req.EndpointID,
		TerminalID: req.TerminalID,
		Tags:       cloneStringMap(req.Tags),
	})
	if service.EditTagsErr != nil {
		return service.EditTagsErr
	}
	return service.EditErr
}

func (service *FakeTerminalService) SendInput(_ context.Context, req port.TerminalInputRequest) error {
	service.Inputs = append(service.Inputs, req)
	return service.InputErr
}

func (service *FakeTerminalService) Resize(_ context.Context, req port.TerminalResizeRequest) (port.TerminalResizeResult, error) {
	service.Resizes = append(service.Resizes, req)
	if service.ResizeErr != nil {
		return port.TerminalResizeResult{}, service.ResizeErr
	}
	result := service.ResizeResult
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = req.TerminalID
	}
	if result.Cols == 0 {
		result.Cols = req.Cols
	}
	if result.Rows == 0 {
		result.Rows = req.Rows
	}
	if result.ResizePolicy == "" {
		result.ResizePolicy = req.ResizePolicy
	}
	if result.SurfaceID == "" {
		result.SurfaceID = req.SurfaceID
	}
	if result.ViewID == "" {
		result.ViewID = req.ViewID
	}
	if result.Session == nil {
		result.Session = req.Session
	}
	if result.OperationID == "" {
		result.OperationID = req.OperationID
	}
	if !result.SizeLocked && result.ControlReason == "" {
		result.Resized = true
		result.CanResize = true
	}
	if result.CanResize && result.ResizePolicy == state.TerminalResizeRoleOwner {
		if result.OwnerSurfaceID == "" {
			result.OwnerSurfaceID = result.SurfaceID
		}
		if result.OwnerViewID == "" {
			result.OwnerViewID = result.ViewID
		}
	}
	return result, nil
}

func (service *FakeTerminalService) ResizeLock(_ context.Context, req port.TerminalResizeLockRequest) (port.TerminalResizeResult, error) {
	service.ResizeLocks = append(service.ResizeLocks, req)
	if service.ResizeLockErr != nil {
		return port.TerminalResizeResult{}, service.ResizeLockErr
	}
	result := service.ResizeLockResult
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	if result.TerminalID == "" {
		result.TerminalID = req.TerminalID
	}
	result.SizeLocked = req.Locked
	result.CanResize = !req.Locked
	result.ResizePolicy = state.TerminalResizeRoleOwner
	result.SurfaceID = req.SurfaceID
	result.ViewID = req.ViewID
	result.OwnerSurfaceID = req.SurfaceID
	result.OwnerViewID = req.ViewID
	result.Session = req.Session
	result.OperationID = req.OperationID
	if req.Locked {
		result.ControlReason = "size_locked"
	} else {
		result.ControlReason = "owner"
	}
	return result, nil
}

func (service *FakeTerminalService) ListDirectories(_ context.Context, req port.PathListDirectoriesRequest) (port.PathListDirectoriesResult, error) {
	service.PathRequests = append(service.PathRequests, req)
	if service.PathErr != nil {
		return port.PathListDirectoriesResult{}, service.PathErr
	}
	result := service.PathResult
	result.Entries = clonePathDirectoryEntries(result.Entries)
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	return result, nil
}

func (service *FakeTerminalService) Defaults(_ context.Context, req port.PathDefaultsRequest) (port.PathDefaultsResult, error) {
	service.PathDefaultsRequests = append(service.PathDefaultsRequests, req)
	if service.PathDefaultsErr != nil {
		return port.PathDefaultsResult{}, service.PathDefaultsErr
	}
	result := service.PathDefaultsResult
	result.DefaultCommand = append([]string(nil), result.DefaultCommand...)
	if result.EndpointID == "" {
		result.EndpointID = req.EndpointID
	}
	return result, nil
}

func (service *FakeTerminalService) LiveSurface(_ context.Context, req port.TerminalSurfaceRequest) (port.TerminalSurfaceResult, error) {
	service.Surfaces = append(service.Surfaces, req)
	if service.SurfaceErr != nil {
		return port.TerminalSurfaceResult{}, service.SurfaceErr
	}
	result := service.SurfaceResult
	if result.Snapshot.EndpointID == "" {
		result.Snapshot.EndpointID = req.EndpointID
	}
	if result.Snapshot.TerminalID == "" {
		result.Snapshot.TerminalID = req.TerminalID
	}
	if result.Snapshot.Cols == 0 {
		result.Snapshot.Cols = req.Cols
	}
	if result.Snapshot.Rows == 0 {
		result.Snapshot.Rows = req.Rows
	}
	if result.Ready || len(result.Snapshot.Lines) > 0 || len(result.Snapshot.Screen) > 0 || result.Snapshot.Cursor.Visible {
		result.Ready = true
	}
	return result, nil
}

func (service *FakeTerminalService) LiveScreenNext(ctx context.Context, req port.TerminalSurfaceRequest) (port.TerminalSurfaceResult, error) {
	service.liveScreenNextMu.Lock()
	service.liveScreenNextRequests = append(service.liveScreenNextRequests, req)
	service.liveScreenNextMu.Unlock()
	if service.LiveScreenNextErr != nil {
		return port.TerminalSurfaceResult{}, service.LiveScreenNextErr
	}
	if service.LiveScreenNextCh != nil {
		select {
		case result, ok := <-service.LiveScreenNextCh:
			if !ok {
				return port.TerminalSurfaceResult{}, context.Canceled
			}
			if result.Snapshot.EndpointID == "" {
				result.Snapshot.EndpointID = req.EndpointID
			}
			if result.Snapshot.TerminalID == "" {
				result.Snapshot.TerminalID = req.TerminalID
			}
			return result, nil
		case <-ctx.Done():
			return port.TerminalSurfaceResult{}, ctx.Err()
		}
	}
	<-ctx.Done()
	return port.TerminalSurfaceResult{}, ctx.Err()
}

// LiveScreenNextRequestsSnapshot 返回 fake 已观测到的 one-shot live wake 请求快照。
// LiveScreenNext 可由异步 effect 调用；测试只能读取本方法返回的副本，禁止直接共享可变 slice。
func (service *FakeTerminalService) LiveScreenNextRequestsSnapshot() []port.TerminalSurfaceRequest {
	service.liveScreenNextMu.Lock()
	defer service.liveScreenNextMu.Unlock()
	return append([]port.TerminalSurfaceRequest(nil), service.liveScreenNextRequests...)
}

func clonePathDirectoryEntries(entries []port.PathDirectoryEntry) []port.PathDirectoryEntry {
	if len(entries) == 0 {
		return nil
	}
	cloned := make([]port.PathDirectoryEntry, len(entries))
	copy(cloned, entries)
	return cloned
}

func cloneTerminalPoolItems(items []port.TerminalPoolItem) []port.TerminalPoolItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]port.TerminalPoolItem, len(items))
	for i, item := range items {
		cloned[i] = item
		cloned[i].Command = append([]string(nil), item.Command...)
		if item.ExitCode != nil {
			code := *item.ExitCode
			cloned[i].ExitCode = &code
		}
		if len(item.Tags) > 0 {
			cloned[i].Tags = make(map[string]string, len(item.Tags))
			for key, value := range item.Tags {
				cloned[i].Tags[key] = value
			}
		}
	}
	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

type FakeClipboardService struct {
	ReadResult port.ClipboardReadResult
	ReadErr    error
	WriteErr   error
	LastCopied string
	Writes     []port.ClipboardWriteRequest
}

func (service *FakeClipboardService) Read(_ context.Context) (port.ClipboardReadResult, error) {
	if service.ReadErr != nil {
		return port.ClipboardReadResult{}, service.ReadErr
	}
	return service.ReadResult, nil
}

func (service *FakeClipboardService) Write(_ context.Context, req port.ClipboardWriteRequest) error {
	service.Writes = append(service.Writes, req)
	if service.WriteErr != nil {
		return service.WriteErr
	}
	service.LastCopied = req.Text
	return nil
}

func (service *FakeClipboardService) LastCopy() string {
	return service.LastCopied
}
