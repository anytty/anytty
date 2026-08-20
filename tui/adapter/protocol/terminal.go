package protocoladapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/tui/port"
	"github.com/anytty/anytty/tui/state"
	"google.golang.org/protobuf/proto"
)

// ProtocolTerminalClient 是 tui-v3 service adapter 依赖的 core-v2 protocol 边界。
// live screen next 方法只透传 observed native revision 补 wake 边沿；adapter
// 不把 renderer/FrameSink 状态写回 core，也不从 protocol 侧推断 history truth。
type ProtocolTerminalClient interface {
	clientruntime.ProtoApplicationExecutor
	ApplicationAttachmentChannel(*apipb.ResourceHandle) (uint16, bool)
	ApplicationAttachment(uint16) (*apipb.ResourceHandle, bool)
}

// ProtocolTerminalServiceAdapter 把 TUI-v3 terminal service 契约映射到 anytty protocol。
type ProtocolTerminalServiceAdapter struct {
	Client      ProtocolTerminalClient
	Application *clientruntime.ApplicationSession
}

// NewProtocolTerminalServiceAdapter 把一个已完成 Hello/auth 的 protocol client 绑定到不可变 runtime generation。
// 构造失败时调用方必须放弃该 connection，不能用空 application session 继续运行或 fallback 到旧 method。
func NewProtocolTerminalServiceAdapter(client ProtocolTerminalClient, stamp clientruntime.EndpointSessionStamp) (ProtocolTerminalServiceAdapter, error) {
	if client == nil {
		return ProtocolTerminalServiceAdapter{}, port.ErrMissingTerminalClient
	}
	application, err := clientruntime.NewApplicationSession(stamp, client)
	if err != nil {
		return ProtocolTerminalServiceAdapter{}, err
	}
	return ProtocolTerminalServiceAdapter{Client: client, Application: application}, nil
}

func (adapter ProtocolTerminalServiceAdapter) Attach(ctx context.Context, req port.TerminalAttachRequest) (port.TerminalAttachResult, error) {
	if adapter.Client == nil || adapter.Application == nil {
		return port.TerminalAttachResult{}, port.ErrMissingTerminalClient
	}
	if req.OperationID == "" {
		return port.TerminalAttachResult{}, operationValidationError(clientruntime.ErrorInvalidRequest, "terminal attach operation identity is required")
	}
	if err := adapter.Application.ValidateCurrent(); err != nil {
		return port.TerminalAttachResult{}, err
	}
	finishRPC := perftrace.Measure("tui.protocol.terminal_attach.rpc")
	result, err := adapter.Application.TerminalAttach(ctx, &apipb.TerminalAttachCommand{
		Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID},
		Mode:     attachmentModeToProto(req.Mode), ResizePolicy: resizePolicyToProto(req.ResizePolicy), SurfaceId: req.SurfaceID, ViewId: req.ViewID,
		Operation: &apipb.OperationStamp{OperationId: req.OperationID},
	})
	finishRPC(0)
	if err != nil {
		return port.TerminalAttachResult{}, err
	}
	channel, ok := adapter.Client.ApplicationAttachmentChannel(result.GetAttachment().GetResource())
	if !ok {
		return port.TerminalAttachResult{}, fmt.Errorf("terminal attachment has no protocol stream binding")
	}
	out := port.TerminalAttachResult{
		EndpointID:   req.EndpointID,
		TerminalID:   req.TerminalID,
		Channel:      channel,
		Cols:         int(result.GetSize().GetCols()),
		Rows:         int(result.GetSize().GetRows()),
		CanResize:    true,
		ResizePolicy: resizePolicyFromProto(result.GetResizePolicy()),
		SurfaceID:    req.SurfaceID,
		ViewID:       req.ViewID,
		Session:      adapter.Application.ProtoStamp(),
		OperationID:  req.OperationID,
	}
	if result.GetResizeControl() != nil {
		applyProtoResizeControlToAttach(&out, result.GetResizeControl())
	}
	return out, nil
}

func (adapter ProtocolTerminalServiceAdapter) Detach(ctx context.Context, req port.TerminalDetachRequest) error {
	if adapter.Client == nil || adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	if err := adapter.validateBoundOperation(req.Session, req.OperationID); err != nil {
		return err
	}
	resource, ok := adapter.Client.ApplicationAttachment(req.Channel)
	if !ok {
		return fmt.Errorf("terminal detach requires active attachment resource")
	}
	return adapter.Application.TerminalDetach(ctx, &apipb.TerminalDetachCommand{Attachment: resource, Operation: &apipb.OperationStamp{OperationId: req.OperationID}})
}

func (adapter ProtocolTerminalServiceAdapter) List(ctx context.Context, req port.TerminalListRequest) (port.TerminalListResult, error) {
	if adapter.Application == nil {
		return port.TerminalListResult{}, port.ErrMissingTerminalClient
	}
	finishRPC := perftrace.Measure("tui.protocol.terminal_list.rpc")
	result, err := adapter.Application.TerminalList(ctx, &apipb.TerminalListCommand{})
	finishRPC(0)
	if err != nil {
		return port.TerminalListResult{}, err
	}
	finishConvert := perftrace.Measure("tui.protocol.terminal_list.convert")
	items := make([]port.TerminalPoolItem, 0, len(result.GetTerminals()))
	for _, terminal := range result.GetTerminals() {
		items = append(items, port.TerminalPoolItem{
			EndpointID: req.EndpointID, TerminalID: terminal.GetRef().GetTerminalId(), Title: terminalTitleFromProto(terminal), State: terminalStateFromProto(terminal.GetState()), CWD: terminal.GetCwd(),
			Command: append([]string(nil), terminal.GetCommand()...), Tags: cloneStringMap(terminal.GetTags()), ExitCode: int32PointerToInt(terminal.ExitCode), ExitedAt: unixNanoTime(terminal.GetExitedAtUnixNano()),
			Cols: int(terminal.GetSize().GetCols()), Rows: int(terminal.GetSize().GetRows()), AttachmentCount: int(terminal.GetAttachmentCount()),
			Resources: port.TerminalResourceUsage{
				PID: int(terminal.GetResources().GetPid()), CPUPercentX100: int(terminal.GetResources().GetCpuPercentX100()), MemoryBytes: terminal.GetResources().GetMemoryBytes(), SampledAt: unixNanoTime(terminal.GetResources().GetSampledAtUnixNano()),
			},
			ResourceHistory: terminalResourceHistoryFromProto(terminal.GetResourceHistory()),
		})
	}
	finishConvert(len(items))
	return port.TerminalListResult{Items: items}, nil
}

func terminalResourceHistoryFromProto(samples []*apipb.TerminalResourceUsage) []port.TerminalResourceUsage {
	if len(samples) == 0 {
		return nil
	}
	out := make([]port.TerminalResourceUsage, 0, len(samples))
	for _, sample := range samples {
		if sample == nil {
			continue
		}
		out = append(out, port.TerminalResourceUsage{
			PID: int(sample.GetPid()), CPUPercentX100: int(sample.GetCpuPercentX100()), MemoryBytes: sample.GetMemoryBytes(), SampledAt: unixNanoTime(sample.GetSampledAtUnixNano()),
		})
	}
	return out
}

func (adapter ProtocolTerminalServiceAdapter) Create(ctx context.Context, req port.TerminalCreateRequest) (port.TerminalCreateResult, error) {
	if adapter.Application == nil {
		return port.TerminalCreateResult{}, port.ErrMissingTerminalClient
	}
	command := append([]string(nil), req.Command...)
	if len(command) == 0 {
		// 中文说明：默认 command 属于目标 daemon endpoint；adapter 这里没有
		// endpoint 默认值 truth，不能退回 TUI 进程本地 SHELL。
		return port.TerminalCreateResult{}, fmt.Errorf("terminal create command is required")
	}
	finishRPC := perftrace.Measure("tui.protocol.terminal_create.rpc")
	result, err := adapter.Application.TerminalCreate(ctx, &apipb.TerminalCreateCommand{Terminal: &apipb.TerminalCreateSpec{
		TerminalId: req.TerminalID, Name: req.Title, Command: command, Tags: cloneStringMap(req.Tags), Cwd: req.CWD, Size: &apipb.TerminalSize{Cols: uint32(req.Cols), Rows: uint32(req.Rows)},
	}})
	finishRPC(0)
	if err != nil {
		return port.TerminalCreateResult{}, err
	}
	terminalID := result.GetTerminal().GetRef().GetTerminalId()
	if terminalID == "" {
		terminalID = req.TerminalID
	}
	return port.TerminalCreateResult{EndpointID: req.EndpointID, TerminalID: terminalID, State: terminalStateFromProto(result.GetTerminal().GetState())}, nil
}

func (adapter ProtocolTerminalServiceAdapter) Restart(ctx context.Context, req port.TerminalRestartRequest) error {
	if adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	return adapter.Application.TerminalRestart(ctx, &apipb.TerminalRestartCommand{Terminal: terminalRef(req.EndpointID, req.TerminalID)})
}

func (adapter ProtocolTerminalServiceAdapter) Reconnect(ctx context.Context, req port.TerminalReconnectRequest) (port.TerminalAttachResult, error) {
	return adapter.Attach(ctx, port.TerminalAttachRequest{
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

func (adapter ProtocolTerminalServiceAdapter) Kill(ctx context.Context, req port.TerminalKillRequest) error {
	if adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	return adapter.Application.TerminalKill(ctx, &apipb.TerminalKillCommand{Terminal: terminalRef(req.EndpointID, req.TerminalID)})
}

func (adapter ProtocolTerminalServiceAdapter) Remove(ctx context.Context, req port.TerminalRemoveRequest) error {
	if adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	return adapter.Application.TerminalRemove(ctx, &apipb.TerminalRemoveCommand{Terminal: terminalRef(req.EndpointID, req.TerminalID)})
}

func (adapter ProtocolTerminalServiceAdapter) EditMetadata(ctx context.Context, req port.TerminalEditMetadataRequest) error {
	if adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	return adapter.Application.TerminalSetMetadata(ctx, &apipb.TerminalSetMetadataCommand{Terminal: terminalRef(req.EndpointID, req.TerminalID), Name: req.Title, Tags: cloneStringMap(req.Tags)})
}

func (adapter ProtocolTerminalServiceAdapter) EditTags(ctx context.Context, req port.TerminalEditTagsRequest) error {
	if adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	return adapter.Application.TerminalSetTags(ctx, &apipb.TerminalSetTagsCommand{Terminal: terminalRef(req.EndpointID, req.TerminalID), Tags: cloneStringMap(req.Tags)})
}

func (adapter ProtocolTerminalServiceAdapter) SendInput(ctx context.Context, req port.TerminalInputRequest) error {
	if adapter.Client == nil || adapter.Application == nil {
		return port.ErrMissingTerminalClient
	}
	if req.Channel == 0 {
		return fmt.Errorf("terminal input requires attached channel")
	}
	if err := adapter.validateBoundOperation(req.Session, req.OperationID); err != nil {
		return err
	}
	resource, ok := adapter.Client.ApplicationAttachment(req.Channel)
	if !ok {
		return fmt.Errorf("terminal input requires active attachment resource")
	}
	return adapter.Application.TerminalInput(ctx, &apipb.TerminalInputCommand{Attachment: resource, Operation: &apipb.OperationStamp{OperationId: req.OperationID}, Data: append([]byte(nil), req.Bytes...)})
}

func (adapter ProtocolTerminalServiceAdapter) Resize(ctx context.Context, req port.TerminalResizeRequest) (port.TerminalResizeResult, error) {
	if adapter.Client == nil || adapter.Application == nil {
		return port.TerminalResizeResult{}, port.ErrMissingTerminalClient
	}
	if req.Channel == 0 {
		return port.TerminalResizeResult{}, fmt.Errorf("terminal resize requires attached channel")
	}
	if err := adapter.validateBoundOperation(req.Session, req.OperationID); err != nil {
		return port.TerminalResizeResult{}, err
	}
	resource, ok := adapter.Client.ApplicationAttachment(req.Channel)
	if !ok {
		return port.TerminalResizeResult{}, fmt.Errorf("terminal resize requires active attachment resource")
	}
	result, err := adapter.Application.TerminalResize(ctx, &apipb.TerminalResizeCommand{Attachment: resource, Operation: &apipb.OperationStamp{OperationId: req.OperationID}, Size: &apipb.TerminalSize{Cols: uint32(req.Cols), Rows: uint32(req.Rows)}, ResizePolicy: resizePolicyToProto(req.ResizePolicy), TakeOwnership: req.TakeOwnership, ExpectedOwnerEpoch: req.ExpectedOwnerEpoch})
	if err != nil {
		return port.TerminalResizeResult{}, err
	}
	return terminalResizeResultFromProto(req, result), nil
}

func (adapter ProtocolTerminalServiceAdapter) ResizeLock(ctx context.Context, req port.TerminalResizeLockRequest) (port.TerminalResizeResult, error) {
	if adapter.Client == nil || adapter.Application == nil {
		return port.TerminalResizeResult{}, port.ErrMissingTerminalClient
	}
	if req.Channel == 0 {
		return port.TerminalResizeResult{}, fmt.Errorf("terminal resize lock requires attached channel")
	}
	if err := adapter.validateBoundOperation(req.Session, req.OperationID); err != nil {
		return port.TerminalResizeResult{}, err
	}
	resource, ok := adapter.Client.ApplicationAttachment(req.Channel)
	if !ok {
		return port.TerminalResizeResult{}, fmt.Errorf("terminal resize lock requires active attachment resource")
	}
	result, err := adapter.Application.TerminalResizeLock(ctx, &apipb.TerminalResizeLockCommand{Attachment: resource, Operation: &apipb.OperationStamp{OperationId: req.OperationID}, Locked: req.Locked})
	if err != nil {
		return port.TerminalResizeResult{}, err
	}
	return terminalResizeResultFromProto(port.TerminalResizeRequest{
		EndpointID: req.EndpointID, TerminalID: req.TerminalID, SurfaceID: req.SurfaceID,
		ViewID: req.ViewID, Session: req.Session, OperationID: req.OperationID,
	}, result), nil
}

func applyProtoResizeControlToAttach(out *port.TerminalAttachResult, control *apipb.ResizeControl) {
	if out == nil || control == nil {
		return
	}
	out.CanResize = control.CanResize
	out.SizeLocked = control.SizeLocked
	out.ControlReason = resizeControlReasonFromProto(control.GetReason())
	out.ResizePolicy = resizePolicyFromProtoControl(control, out.ResizePolicy)
	out.OwnerSurfaceID = control.GetOwnerSurfaceId()
	out.OwnerViewID = control.GetOwnerViewId()
	if ownership := control.GetOwnership(); ownership != nil {
		out.SizeLocked = ownership.GetSizeLocked()
		out.ResizeEpoch = ownership.GetEpoch()
		if ownership.GetSize() != nil {
			out.Cols = int(ownership.GetSize().GetCols())
			out.Rows = int(ownership.GetSize().GetRows())
		}
	}
}

func terminalResizeResultFromProto(req port.TerminalResizeRequest, result *apipb.TerminalResizeResult) port.TerminalResizeResult {
	out := port.TerminalResizeResult{EndpointID: req.EndpointID, TerminalID: req.TerminalID, Cols: req.Cols, Rows: req.Rows, ResizePolicy: req.ResizePolicy, SurfaceID: req.SurfaceID, ViewID: req.ViewID, Session: cloneEndpointSessionStamp(req.Session), OperationID: req.OperationID}
	if result == nil {
		return out
	}
	out.Resized = result.GetResized()
	out.Cols = int(result.GetSize().GetCols())
	out.Rows = int(result.GetSize().GetRows())
	if control := result.GetResizeControl(); control != nil {
		out.CanResize = control.GetCanResize()
		out.SizeLocked = control.GetSizeLocked()
		out.ControlReason = resizeControlReasonFromProto(control.GetReason())
		out.ResizePolicy = resizePolicyFromProtoControl(control, out.ResizePolicy)
		out.OwnerSurfaceID = control.GetOwnerSurfaceId()
		out.OwnerViewID = control.GetOwnerViewId()
		if ownership := control.GetOwnership(); ownership != nil {
			out.SizeLocked = ownership.GetSizeLocked()
			out.ResizeEpoch = ownership.GetEpoch()
			if ownership.GetSize() != nil {
				out.Cols = int(ownership.GetSize().GetCols())
				out.Rows = int(ownership.GetSize().GetRows())
			}
		}
	}
	return out
}

func (adapter ProtocolTerminalServiceAdapter) validateBoundOperation(session *apipb.EndpointSessionStamp, operationID string) error {
	if operationID == "" {
		return operationValidationError(clientruntime.ErrorInvalidRequest, "terminal operation identity is required")
	}
	current := adapter.Application.ProtoStamp()
	if session == nil || !proto.Equal(session, current) {
		return operationValidationError(clientruntime.ErrorStaleSession, "terminal attachment session stamp is stale")
	}
	return adapter.Application.ValidateCurrent()
}

func operationValidationError(code clientruntime.ErrorCode, message string) error {
	return &clientruntime.Error{Code: code, Message: message, Attempted: false}
}

func cloneEndpointSessionStamp(stamp *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if stamp == nil {
		return nil
	}
	return proto.Clone(stamp).(*apipb.EndpointSessionStamp)
}

func resizePolicyFromProtoControl(control *apipb.ResizeControl, fallback string) string {
	if control == nil {
		return fallback
	}
	// core-v2 的 ResizeControl.Reason 是 attachment 这次 attach/ensure 后的权威角色；
	// 不能继续沿用请求里的 ResizePolicy，否则 UI 会显示半旧的 owner/follower 状态。
	switch control.GetReason() {
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER, apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED:
		return state.TerminalResizeRoleOwner
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER:
		return state.TerminalResizeRoleObserver
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER:
		return state.TerminalResizeRoleFollower
	default:
		if control.CanResize {
			return state.TerminalResizeRoleOwner
		}
		return fallback
	}
}

func (adapter ProtocolTerminalServiceAdapter) LiveSurface(ctx context.Context, req port.TerminalSurfaceRequest) (port.TerminalSurfaceResult, error) {
	if adapter.Application == nil {
		return port.TerminalSurfaceResult{}, port.ErrMissingTerminalClient
	}
	finishRPC := perftrace.Measure("tui.protocol.live_surface.rpc")
	snapshot, err := adapter.Application.LiveScreenNext(ctx, &apipb.LiveScreenNextCommand{
		Terminal:         &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID},
		ObservedRevision: req.ObservedRevision,
	})
	finishRPC(0)
	if err != nil {
		return port.TerminalSurfaceResult{}, err
	}
	finishConvert := perftrace.Measure("tui.protocol.live_surface_convert")
	// 中文说明：v3 live display 的 screen truth 只能来自 core latest native screen；
	// lifecycle 由 list/state event 单独承载，不能再混入 snapshot RPC。
	liveSnapshot := liveSurfaceSnapshotFromProto(req.TerminalID, snapshot)
	finishConvert(liveSnapshotApproxBytes(liveSnapshot))
	return port.TerminalSurfaceResult{
		Ready:          true,
		Snapshot:       liveSnapshot,
		LifecycleKnown: false,
	}, nil
}

func (adapter ProtocolTerminalServiceAdapter) LiveScreenNext(ctx context.Context, req port.TerminalSurfaceRequest) (port.TerminalSurfaceResult, error) {
	if adapter.Application == nil {
		return port.TerminalSurfaceResult{}, port.ErrMissingTerminalClient
	}
	finishRPC := perftrace.Measure("tui.protocol.live_screen_next.rpc")
	event, err := adapter.Application.LiveScreenNext(ctx, &apipb.LiveScreenNextCommand{Terminal: &apipb.TerminalRef{EndpointId: string(req.EndpointID), TerminalId: req.TerminalID}, ObservedRevision: req.ObservedRevision})
	finishRPC(0)
	if err != nil {
		return port.TerminalSurfaceResult{}, err
	}
	if event == nil {
		return port.TerminalSurfaceResult{}, context.Canceled
	}
	snapshot := liveSurfaceSnapshotFromProto(req.TerminalID, event)
	snapshot.EndpointID = req.EndpointID
	if snapshot.TerminalID == "" {
		snapshot.TerminalID = event.GetTerminal().GetTerminalId()
	}
	perftrace.Count("tui.protocol.live_screen_next", liveSnapshotApproxBytes(snapshot))
	return port.TerminalSurfaceResult{Snapshot: snapshot, Ready: true}, nil
}

func (adapter ProtocolTerminalServiceAdapter) WatchTerminalEvents(ctx context.Context, endpointID state.EndpointID) (<-chan port.TerminalLiveEvent, error) {
	if adapter.Application == nil {
		return nil, port.ErrMissingTerminalClient
	}
	_, events, err := adapter.Application.EventSubscribe(ctx, &apipb.EventSubscribeCommand{Types: []apipb.ApplicationEventType{
		apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
	}})
	if err != nil {
		return nil, err
	}
	out := make(chan port.TerminalLiveEvent, 32)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case envelope, ok := <-events:
				if !ok {
					return
				}
				event, ok := terminalAttachmentEventFromProto(endpointID, envelope)
				if !ok {
					continue
				}
				select {
				case out <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func terminalAttachmentEventFromProto(endpointID state.EndpointID, envelope *apipb.EventEnvelope) (port.TerminalLiveEvent, bool) {
	lifecycle := envelope.GetTerminalLifecycle()
	if lifecycle == nil || !lifecycle.GetAttachmentProjection() {
		return port.TerminalLiveEvent{}, false
	}
	terminal := lifecycle.GetTerminal()
	if terminal.GetRef().GetTerminalId() == "" {
		return port.TerminalLiveEvent{}, false
	}
	if terminal.GetRef().GetEndpointId() != "" {
		endpointID = state.EndpointID(terminal.GetRef().GetEndpointId())
	}
	out := port.TerminalLiveEvent{
		EndpointID:           state.NormalizeEndpointID(endpointID),
		TerminalID:           terminal.GetRef().GetTerminalId(),
		Tags:                 cloneStringMap(terminal.GetTags()),
		Metadata:             true,
		AttachmentProjection: true,
		AttachmentCount:      int(terminal.GetAttachmentCount()),
	}
	if control := lifecycle.GetResizeControl(); control != nil {
		out.OwnerSurfaceID = control.GetOwnerSurfaceId()
		out.OwnerViewID = control.GetOwnerViewId()
		out.SizeLocked = control.GetSizeLocked()
		if ownership := control.GetOwnership(); ownership != nil {
			out.OwnerSurfaceID = ownership.GetOwnerSurfaceId()
			out.OwnerViewID = ownership.GetOwnerViewId()
			out.ResizeEpoch = ownership.GetEpoch()
			out.SizeLocked = ownership.GetSizeLocked()
			if ownership.GetSize() != nil {
				out.Cols = int(ownership.GetSize().GetCols())
				out.Rows = int(ownership.GetSize().GetRows())
			}
		}
	}
	return out, true
}

func liveSurfaceSnapshotFromProto(terminalID string, snapshot *apipb.NativeScreenResult) state.LiveSurfaceSnapshot {
	if snapshot == nil {
		return state.LiveSurfaceSnapshot{TerminalID: terminalID}
	}
	result := state.LiveSurfaceSnapshot{
		TerminalID:   terminalID,
		Revision:     snapshot.GetLiveRevision(),
		BaseRevision: snapshot.GetBaseRevision(),
		Cols:         int(snapshot.GetSize().GetCols()),
		Rows:         int(snapshot.GetSize().GetRows()),
		FullReplace:  snapshot.GetFullReplace(),
	}
	if cursor := snapshot.GetCursor(); cursor != nil {
		result.Cursor = state.LiveCursor{Visible: cursor.GetVisible(), Row: int(cursor.GetRow()), Col: int(cursor.GetCol()), Shape: strings.ToLower(strings.TrimPrefix(cursor.GetShape().String(), "CURSOR_SHAPE_"))}
	}
	if modes := snapshot.GetModes(); modes != nil {
		result.Modes = state.LiveTerminalModes{MouseTracking: modes.GetMouseTracking(), MouseX10: modes.GetMouseX10(), MouseNormal: modes.GetMouseNormal(), MouseButton: modes.GetMouseButtonEvent(), MouseAny: modes.GetMouseAnyEvent(), MouseSGR: modes.GetMouseSgr(), BracketedPaste: modes.GetBracketedPaste()}
	}
	for _, rowCopy := range snapshot.GetRowCopies() {
		result.RowCopies = append(result.RowCopies, state.LiveRowCopy{SourceRow: int(rowCopy.GetSourceRow()), DestinationRow: int(rowCopy.GetDestinationRow()), Count: int(rowCopy.GetCount())})
	}
	for _, replacement := range snapshot.GetRowReplacements() {
		row := replacement.GetRow()
		cells := make([]state.LiveCell, 0, len(row.GetCells()))
		for _, cell := range row.GetCells() {
			style := cell.GetStyle()
			cells = append(cells, state.LiveCell{Text: cell.GetContent(), Width: int(cell.GetWidth()), FG: style.GetForeground(), BG: style.GetBackground(), Bold: style.GetBold(), Italic: style.GetItalic(), Underline: style.GetUnderline(), Blink: style.GetBlink(), Reverse: style.GetReverse(), Strikethrough: style.GetStrikethrough(), LinkURL: cell.GetLinkUrl(), LinkParams: cell.GetLinkParams()})
		}
		result.Screen = append(result.Screen, cells)
		result.ChangedRows = append(result.ChangedRows, int(replacement.GetRowIndex()))
	}
	return result
}

func liveSnapshotApproxBytes(snapshot state.LiveSurfaceSnapshot) int {
	total := 0
	for _, line := range snapshot.Lines {
		total += len(line)
	}
	for _, row := range snapshot.Screen {
		for _, cell := range row {
			total += len(cell.Text)
		}
	}
	return total
}

func terminalRef(endpointID state.EndpointID, terminalID string) *apipb.TerminalRef {
	return &apipb.TerminalRef{EndpointId: string(endpointID), TerminalId: terminalID}
}

func terminalTitleFromProto(terminal *apipb.TerminalInfo) string {
	if terminal.GetName() != "" {
		return terminal.GetName()
	}
	if terminal.GetRef().GetTerminalId() != "" {
		return terminal.GetRef().GetTerminalId()
	}
	return "terminal"
}

func int32PointerToInt(value *int32) *int {
	if value == nil {
		return nil
	}
	cloned := int(*value)
	return &cloned
}

func unixNanoTime(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, value)
}

func terminalStateFromProto(value apipb.TerminalState) string {
	switch value {
	case apipb.TerminalState_TERMINAL_STATE_CREATED:
		return "created"
	case apipb.TerminalState_TERMINAL_STATE_RUNNING:
		return "running"
	case apipb.TerminalState_TERMINAL_STATE_EXITED:
		return "exited"
	case apipb.TerminalState_TERMINAL_STATE_REMOVED:
		return "removed"
	default:
		return ""
	}
}

func attachmentModeToProto(value string) apipb.AttachmentMode {
	if strings.EqualFold(value, "observer") {
		return apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER
	}
	return apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR
}

func resizePolicyToProto(value string) apipb.ResizePolicy {
	switch value {
	case state.TerminalResizeRoleFollower:
		return apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER
	case state.TerminalResizeRoleObserver:
		return apipb.ResizePolicy_RESIZE_POLICY_OBSERVER
	default:
		return apipb.ResizePolicy_RESIZE_POLICY_OWNER
	}
}

func resizePolicyFromProto(value apipb.ResizePolicy) string {
	switch value {
	case apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER:
		return state.TerminalResizeRoleFollower
	case apipb.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return state.TerminalResizeRoleObserver
	default:
		return state.TerminalResizeRoleOwner
	}
}

func resizeControlReasonFromProto(value apipb.ResizeControlReason) string {
	switch value {
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER:
		return "owner"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER:
		return "follower"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER:
		return "observer"
	case apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED:
		return "size_locked"
	default:
		return ""
	}
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
