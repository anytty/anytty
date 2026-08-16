package state

import (
	"strconv"

	"github.com/anytty/anytty/proto/apipb"
	"google.golang.org/protobuf/proto"
)

const (
	TerminalResizeRoleOwner    = "owner"
	TerminalResizeRoleFollower = "follower"
	TerminalResizeRoleObserver = "observer"

	TerminalViewAlignStart  = "start"
	TerminalViewAlignCenter = "center"
	TerminalViewAlignEnd    = "end"
	TerminalViewAlignBase   = "base"

	TerminalViewLayoutAuto   = "auto"
	TerminalViewLayoutFit    = "fit"
	TerminalViewLayoutCenter = "center"
)

// TerminalViewStore 是 pane/floating 到 core-v2 attachment 的 reducer-owned 连接视图状态。
// Terminal 本身仍是共享 process/lifecycle/history truth，view 只保存 UI 连接身份和请求状态。
type TerminalViewStore struct {
	Views         map[string]TerminalViewBinding
	PaneViews     map[string]string
	FloatingViews map[string]string
	NextOperation uint64 `json:"-"`
}

// TerminalViewBinding 是 pane/floating 到 owning daemon terminal 的连接意图和运行时 attach 投影。
// EndpointID + TerminalID 是跨 endpoint 真值；Channel、SurfaceID、OwnerViewID 等字段只属于当前 TUI 运行时和 daemon attachment 回包，不是 workbench storage 的长期 truth。
type TerminalViewBinding struct {
	ViewID              string                      `json:"viewId"`
	SurfaceID           string                      `json:"surfaceId,omitempty"`
	EndpointID          EndpointID                  `json:"endpointId,omitempty"`
	TerminalID          string                      `json:"terminalId"`
	Channel             uint16                      `json:"channel,omitempty"`
	Layout              TerminalViewLayout          `json:"layout,omitempty"`
	ResizeRole          string                      `json:"resizeRole,omitempty"`
	DesiredCols         int                         `json:"desiredCols,omitempty"`
	DesiredRows         int                         `json:"desiredRows,omitempty"`
	RequestSeq          uint64                      `json:"requestSeq,omitempty"`
	LastError           string                      `json:"lastError,omitempty"`
	PaneID              string                      `json:"paneId,omitempty"`
	FloatingID          string                      `json:"floatingId,omitempty"`
	Attached            bool                        `json:"attached"`
	CanResize           bool                        `json:"canResize,omitempty"`
	SizeLocked          bool                        `json:"sizeLocked,omitempty"`
	ControlReason       string                      `json:"controlReason,omitempty"`
	OwnerSurfaceID      string                      `json:"ownerSurfaceId,omitempty"`
	OwnerViewID         string                      `json:"ownerViewId,omitempty"`
	ResizeEpoch         uint64                      `json:"resizeEpoch,omitempty"`
	ResizePending       bool                        `json:"resizePending,omitempty"`
	OwnerAcquirePending bool                        `json:"-"`
	AttachPending       bool                        `json:"attachPending,omitempty"`
	Unresolved          bool                        `json:"unresolved,omitempty"`
	UnresolvedReason    string                      `json:"unresolvedReason,omitempty"`
	Session             *apipb.EndpointSessionStamp `json:"-"`
	OperationID         string                      `json:"-"`
	AttachCandidate     *TerminalAttachCandidate    `json:"-"`
}

// TerminalAttachCandidate 是 reducer-owned 的未提交 attach operation。
// candidate 与 committed binding 分离；只有 operation ID 精确匹配的结果才能 commit，迟到成功必须按其返回 stamp/resource cleanup。
type TerminalAttachCandidate struct {
	OperationID string
	EndpointID  EndpointID
	TerminalID  string
	SurfaceID   string
	ViewID      string
	ResizeRole  string
	DesiredCols int
	DesiredRows int
	PaneID      string
	FloatingID  string
	HadBinding  bool
}

// NextTerminalOperation 分配 TUI 进程内唯一 operation ID，用于 input/paste/resize/detach 与 attach candidate correlation。
// 该序号只属于 reducer runtime，不进入 workbench storage，也不代替 Proto EndpointSessionStamp。
func (store TerminalViewStore) NextTerminalOperation(kind, viewID string) (TerminalViewStore, string) {
	store.NextOperation++
	return store, kind + ":" + viewID + ":" + strconv.FormatUint(store.NextOperation, 10)
}

// TerminalViewLayout 是 pane/floating 的 view-local 内容布局状态。
// 它不改变共享 terminal process、history truth 或 PTY size ownership。
type TerminalViewLayout struct {
	SizeLocked bool   `json:"sizeLocked,omitempty"`
	Mode       string `json:"mode,omitempty"`
	PanX       int    `json:"panX,omitempty"`
	PanY       int    `json:"panY,omitempty"`
	AlignX     string `json:"alignX,omitempty"`
	AlignY     string `json:"alignY,omitempty"`
}

type TerminalViewLayoutCommand struct {
	Action string
	Mode   string
	AlignX string
	AlignY string
	DeltaX int
	DeltaY int
}

type TerminalViewResizeDecision struct {
	Binding TerminalViewBinding
	Allowed bool
	Changed bool
	Seq     uint64
	Reason  string
}

// NewPaneTerminalView 构造默认 local endpoint 的 pane terminal view binding。
// 这是旧单 daemon 调用边界的兼容入口；新增多 endpoint 路径应使用 NewEndpointPaneTerminalView。
func NewPaneTerminalView(paneID string, terminalID string, channel uint16, cols int, rows int, resizeRole string, surfaceID string, viewID string, canResize bool) TerminalViewBinding {
	return NewEndpointPaneTerminalView(DefaultEndpointID, paneID, terminalID, channel, cols, rows, resizeRole, surfaceID, viewID, canResize)
}

// NewEndpointPaneTerminalView 构造带明确 endpoint 的 pane terminal view binding。
// 调用方必须把 endpoint 作为 terminal 连接意图的一部分传入，后续 owner transfer、size lock、remove 和 restore 都按该 TerminalRef 作用域执行。
func NewEndpointPaneTerminalView(endpointID EndpointID, paneID string, terminalID string, channel uint16, cols int, rows int, resizeRole string, surfaceID string, viewID string, canResize bool) TerminalViewBinding {
	if viewID == "" {
		viewID = TerminalPaneViewID(paneID)
	}
	if surfaceID == "" {
		surfaceID = "tui"
	}
	resizeRole = normalizeTerminalResizeRole(resizeRole)
	return TerminalViewBinding{ViewID: viewID, SurfaceID: surfaceID, EndpointID: NormalizeEndpointID(endpointID), TerminalID: terminalID, Channel: channel, ResizeRole: resizeRole, DesiredCols: cols, DesiredRows: rows, PaneID: paneID, Attached: terminalID != "" && channel != 0, CanResize: canResize}
}

// NewFloatingTerminalView 构造默认 local endpoint 的 floating terminal view binding。
// 这是旧单 daemon 调用边界的兼容入口；新增多 endpoint 路径应使用 NewEndpointFloatingTerminalView。
func NewFloatingTerminalView(floatingID string, paneID string, terminalID string, channel uint16, cols int, rows int, resizeRole string, surfaceID string, viewID string, canResize bool) TerminalViewBinding {
	return NewEndpointFloatingTerminalView(DefaultEndpointID, floatingID, paneID, terminalID, channel, cols, rows, resizeRole, surfaceID, viewID, canResize)
}

// NewEndpointFloatingTerminalView 构造带明确 endpoint 的 floating terminal view binding。
// floating 和 tiled pane 共享同一 TerminalRef 语义；不同 endpoint 下的同名 TerminalID 不会互相抢 owner 或互相移除。
func NewEndpointFloatingTerminalView(endpointID EndpointID, floatingID string, paneID string, terminalID string, channel uint16, cols int, rows int, resizeRole string, surfaceID string, viewID string, canResize bool) TerminalViewBinding {
	if viewID == "" {
		viewID = TerminalFloatingViewID(floatingID)
	}
	if surfaceID == "" {
		surfaceID = "tui"
	}
	resizeRole = normalizeTerminalResizeRole(resizeRole)
	return TerminalViewBinding{ViewID: viewID, SurfaceID: surfaceID, EndpointID: NormalizeEndpointID(endpointID), TerminalID: terminalID, Channel: channel, ResizeRole: resizeRole, DesiredCols: cols, DesiredRows: rows, FloatingID: floatingID, PaneID: paneID, Attached: terminalID != "" && channel != 0, CanResize: canResize}
}

func TerminalPaneViewID(paneID string) string {
	if paneID == "" {
		paneID = DefaultPaneID
	}
	return "pane:" + paneID
}

func TerminalFloatingViewID(floatingID string) string {
	if floatingID == "" {
		floatingID = "floating"
	}
	return "floating:" + floatingID
}

func (store TerminalViewStore) BindPane(binding TerminalViewBinding) TerminalViewStore {
	binding = binding.withDefaultEndpoint()
	if binding.PaneID == "" || binding.TerminalID == "" {
		return store
	}
	if binding.ViewID == "" {
		binding.ViewID = TerminalPaneViewID(binding.PaneID)
	}
	return store.bind(binding)
}

func (store TerminalViewStore) BindFloating(binding TerminalViewBinding) TerminalViewStore {
	binding = binding.withDefaultEndpoint()
	if binding.FloatingID == "" || binding.TerminalID == "" {
		return store
	}
	if binding.ViewID == "" {
		binding.ViewID = TerminalFloatingViewID(binding.FloatingID)
	}
	return store.bind(binding)
}

func (store TerminalViewStore) bind(binding TerminalViewBinding) TerminalViewStore {
	binding = binding.withDefaultEndpoint()
	ref := binding.TerminalRef()
	binding.ResizeRole = normalizeTerminalResizeRole(binding.ResizeRole)
	binding.Layout = binding.Layout.Normalize()
	binding.Attached = binding.TerminalID != "" && binding.Channel != 0
	binding = binding.applyTerminalSizeLockProjection(binding.SizeLocked || store.terminalSizeLocked(ref))
	store.Views = cloneTerminalViewBindings(store.Views)
	store.PaneViews = cloneTerminalViewIDs(store.PaneViews)
	store.FloatingViews = cloneTerminalViewIDs(store.FloatingViews)
	if binding.ResizeRole == TerminalResizeRoleOwner {
		_, hadDifferentOwner := store.ownerIdentityBinding(ref)
		if owner, ok := store.ownerIdentityBinding(ref); ok && owner.ViewID == binding.ViewID {
			hadDifferentOwner = false
		}
		if existing, ok := store.Views[binding.ViewID]; ok && existing.TerminalRef().Equal(ref) && !existing.HasResizeOwner() {
			// 中文说明：attach result 可能把 follower 投影成 owner；这同样需要下一帧主动校验 PTY size。
			binding.ResizePending = true
		}
		if hadDifferentOwner {
			// 中文说明：新建 view 直接以 owner 写入时也属于 owner transfer，不能等后续输入输出才同步尺寸。
			binding.ResizePending = true
		}
	}
	if binding.ResizeRole == TerminalResizeRoleOwner {
		store.demoteResizeOwnersLocked(ref, binding.ViewID)
	}
	store.Views[binding.ViewID] = binding
	if binding.PaneID != "" {
		store.PaneViews[binding.PaneID] = binding.ViewID
	}
	if binding.FloatingID != "" {
		store.FloatingViews[binding.FloatingID] = binding.ViewID
	}
	return store
}

func (store TerminalViewStore) demoteResizeOwnersLocked(ref TerminalRef, exceptViewID string) {
	ref = ref.Normalize()
	if ref.Empty() {
		return
	}
	for candidateID, candidate := range store.Views {
		if candidateID == exceptViewID || !candidate.TerminalRef().Equal(ref) || candidate.ResizeRole != TerminalResizeRoleOwner {
			continue
		}
		candidate.ResizeRole = TerminalResizeRoleFollower
		candidate.CanResize = false
		candidate.ResizePending = false
		store.Views[candidateID] = candidate
	}
}

func (store TerminalViewStore) ApplyPaneLayoutCommand(paneID string, command TerminalViewLayoutCommand) (TerminalViewStore, TerminalViewBinding, bool) {
	return store.ApplyViewLayoutCommand(store.PaneViews[paneID], command)
}

func (store TerminalViewStore) ApplyFloatingLayoutCommand(floatingID string, command TerminalViewLayoutCommand) (TerminalViewStore, TerminalViewBinding, bool) {
	return store.ApplyViewLayoutCommand(store.FloatingViews[floatingID], command)
}

func (store TerminalViewStore) ApplyViewLayoutCommand(viewID string, command TerminalViewLayoutCommand) (TerminalViewStore, TerminalViewBinding, bool) {
	binding, ok := store.Views[viewID]
	if !ok || binding.TerminalID == "" {
		return store, TerminalViewBinding{}, false
	}
	next := binding
	next.Layout = next.Layout.Apply(command)
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = next
	return store, next, true
}

func (store TerminalViewStore) DetachPane(paneID string) TerminalViewStore {
	viewID := store.PaneViews[paneID]
	if viewID == "" {
		return store
	}
	binding, ok := store.Views[viewID]
	if !ok {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	store.PaneViews = cloneTerminalViewIDs(store.PaneViews)
	delete(store.Views, viewID)
	delete(store.PaneViews, paneID)
	store.clearDetachedOwnerProjectionLocked(binding)
	return store
}

func (store TerminalViewStore) DetachFloating(floatingID string) TerminalViewStore {
	viewID := store.FloatingViews[floatingID]
	if viewID == "" {
		return store
	}
	binding, ok := store.Views[viewID]
	if !ok {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	store.FloatingViews = cloneTerminalViewIDs(store.FloatingViews)
	delete(store.Views, viewID)
	delete(store.FloatingViews, floatingID)
	store.clearDetachedOwnerProjectionLocked(binding)
	return store
}

// RemoveTerminal 移除默认 local endpoint 下指定 TerminalID 的所有 view binding。
// 该方法只用于旧单 endpoint 调用边界；跨 endpoint 路径必须使用 RemoveTerminalRef。
func (store TerminalViewStore) RemoveTerminal(terminalID string) TerminalViewStore {
	return store.RemoveTerminalRef(LocalTerminalRef(terminalID))
}

// RemoveTerminalRef 按完整 TerminalRef 移除 view binding。
// 它是 terminal lifecycle/remove 回投进入 reducer 后的 endpoint-aware 删除边界，不会影响其他 endpoint 下同名 terminal。
func (store TerminalViewStore) RemoveTerminalRef(ref TerminalRef) TerminalViewStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	store.PaneViews = cloneTerminalViewIDs(store.PaneViews)
	store.FloatingViews = cloneTerminalViewIDs(store.FloatingViews)
	for viewID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		delete(store.Views, viewID)
		if binding.PaneID != "" {
			delete(store.PaneViews, binding.PaneID)
		}
		if binding.FloatingID != "" {
			delete(store.FloatingViews, binding.FloatingID)
		}
	}
	return store
}

// ApplyWorkbenchEndpointResolution 根据当前 endpoint registry 给 workbench 恢复的 binding 标记 unresolved。
// 该方法只改变 TUI 本地连接意图投影；缺失、禁用或 manual endpoint 的 pane/floating 必须保留在 layout 中，不能被删除或自动改绑。
func (store TerminalViewStore) ApplyWorkbenchEndpointResolution(endpoints EndpointStore) TerminalViewStore {
	if len(store.Views) == 0 || !endpoints.HasItems() {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	for viewID, binding := range store.Views {
		unresolved, reason := workbenchBindingUnresolvedReason(binding, endpoints)
		binding.Unresolved = unresolved
		binding.UnresolvedReason = reason
		if unresolved {
			binding.Attached = false
			binding.Channel = 0
			binding.AttachPending = false
			binding.ResizePending = false
			binding = binding.clearRuntimeResizeControl()
			binding.LastError = "endpoint " + reason
		}
		store.Views[viewID] = binding
	}
	return store
}

// MarkEndpointRuntimeError 把同一个 endpoint 下的所有 pane/floating view 标成连接错误。
// 这是 endpoint transport 生命周期回投进入 view 投影的边界；它只清理本 TUI 的 attachment/channel，
// 保留 TerminalRef 连接意图，等待 endpoint 恢复后通过正常 attach/reconnect 重新建立。
func (store TerminalViewStore) MarkEndpointRuntimeError(endpointID EndpointID, message string) TerminalViewStore {
	endpointID = NormalizeEndpointID(endpointID)
	if endpointID == "" || len(store.Views) == 0 {
		return store
	}
	if message == "" {
		message = "endpoint offline"
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	for viewID, binding := range store.Views {
		if NormalizeEndpointID(binding.EndpointID) != endpointID || binding.TerminalID == "" {
			continue
		}
		binding.Channel = 0
		binding.Attached = false
		binding.AttachPending = false
		binding.ResizePending = false
		binding = binding.clearRuntimeResizeControl()
		binding.LastError = message
		store.Views[viewID] = binding
	}
	return store
}

// MarkViewRuntimeError 把单个 terminal view 标成运行时连接错误。
// 这是用户对某个断线 pane 发起 reconnect 后失败的局部回投边界；它只清理该 view 的
// attachment/channel，不删除 TerminalRef 连接意图，也不影响同 endpoint 的其他 pane。
func (store TerminalViewStore) MarkViewRuntimeError(viewID string, message string) TerminalViewStore {
	if viewID == "" || len(store.Views) == 0 {
		return store
	}
	if message == "" {
		message = "terminal view disconnected"
	}
	binding, ok := store.Views[viewID]
	if !ok || binding.TerminalID == "" {
		return store
	}
	binding.Channel = 0
	binding.Attached = false
	binding.AttachPending = false
	binding.ResizePending = false
	binding = binding.clearRuntimeResizeControl()
	binding.LastError = message
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = binding
	return store
}

// MarkTerminalRefRuntimeError 把同一个 endpoint terminal 的所有 pane/floating view 标成运行时错误。
// 调用方通常来自 live surface、input 或 invalidation 失败；这些失败属于 terminal/ref 的连接链路，
// 不能只写全局 Session/Surface，否则非 active view 或后续 render 投影会丢失 pane 级错误原因。
func (store TerminalViewStore) MarkTerminalRefRuntimeError(ref TerminalRef, message string) TerminalViewStore {
	ref = ref.Normalize()
	if ref.Empty() || len(store.Views) == 0 {
		return store
	}
	if message == "" {
		message = "terminal view disconnected"
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	for viewID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		binding.Channel = 0
		binding.Attached = false
		binding.AttachPending = false
		binding.ResizePending = false
		binding = binding.clearRuntimeResizeControl()
		binding.LastError = message
		store.Views[viewID] = binding
	}
	return store
}

func workbenchBindingUnresolvedReason(binding TerminalViewBinding, endpoints EndpointStore) (bool, string) {
	ref := binding.TerminalRef()
	if ref.Empty() {
		return false, ""
	}
	endpoint, ok := endpoints.DisplayEndpoint(ref.EndpointID)
	if !ok {
		return false, ""
	}
	status := endpoint.DisplayStatus()
	switch status {
	case EndpointStatusDisabled, EndpointStatusUnregistered:
		return true, string(status)
	case EndpointStatusManual:
		return true, string(status)
	}
	return false, ""
}

func (store TerminalViewStore) PaneBinding(paneID string) (TerminalViewBinding, bool) {
	viewID := store.PaneViews[paneID]
	if viewID == "" {
		return TerminalViewBinding{}, false
	}
	binding, ok := store.Views[viewID]
	return binding, ok
}

func (store TerminalViewStore) PaneViewID(paneID string) string {
	if paneID == "" {
		return ""
	}
	if viewID := store.PaneViews[paneID]; viewID != "" {
		return viewID
	}
	return TerminalPaneViewID(paneID)
}

func (store TerminalViewStore) FloatingBinding(floatingID string) (TerminalViewBinding, bool) {
	viewID := store.FloatingViews[floatingID]
	if viewID == "" {
		return TerminalViewBinding{}, false
	}
	binding, ok := store.Views[viewID]
	return binding, ok
}

func (store TerminalViewStore) FloatingViewID(floatingID string) string {
	if floatingID == "" {
		return ""
	}
	if viewID := store.FloatingViews[floatingID]; viewID != "" {
		return viewID
	}
	return TerminalFloatingViewID(floatingID)
}

func (store TerminalViewStore) BindingsForTerminal(terminalID string) []TerminalViewBinding {
	return store.BindingsForTerminalRef(LocalTerminalRef(terminalID))
}

// BindingsForTerminalRef 返回绑定到同一个 endpoint terminal 的所有 pane/floating view。
// attachment count、owner transfer 和 storage restore 都必须使用该 endpoint-aware 查询，避免远端同名 TerminalID 被错误合并。
func (store TerminalViewStore) BindingsForTerminalRef(ref TerminalRef) []TerminalViewBinding {
	ref = ref.Normalize()
	if ref.Empty() {
		return nil
	}
	bindings := make([]TerminalViewBinding, 0)
	for _, binding := range store.Views {
		if binding.TerminalRef().Equal(ref) {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

func (store TerminalViewStore) Bindings() []TerminalViewBinding {
	bindings := make([]TerminalViewBinding, 0, len(store.Views))
	for _, binding := range store.Views {
		bindings = append(bindings, binding)
	}
	return bindings
}

// AttachedBindingCountForEndpoint returns the number of live TUI attachments for one Endpoint.
// Pending or disconnected view intents do not keep a disabled transport alive.
func (store TerminalViewStore) AttachedBindingCountForEndpoint(endpointID EndpointID) int {
	endpointID = NormalizeEndpointID(endpointID)
	count := 0
	for _, binding := range store.Views {
		if NormalizeEndpointID(binding.EndpointID) == endpointID && binding.Attached && binding.Channel != 0 {
			count++
		}
	}
	return count
}

// BeginAttach 创建独立 candidate 并返回唯一 operation ID；已有 committed channel 不会被覆盖。
func (store TerminalViewStore) BeginAttach(binding TerminalViewBinding) (TerminalViewStore, TerminalAttachCandidate) {
	binding = binding.withDefaultEndpoint()
	if binding.TerminalID == "" || binding.ViewID == "" {
		return store, TerminalAttachCandidate{}
	}
	var operationID string
	store, operationID = store.NextTerminalOperation("attach", binding.ViewID)
	candidate := TerminalAttachCandidate{
		OperationID: operationID, EndpointID: binding.EndpointID,
		TerminalID: binding.TerminalID, SurfaceID: binding.SurfaceID, ViewID: binding.ViewID, ResizeRole: binding.ResizeRole,
		DesiredCols: binding.DesiredCols, DesiredRows: binding.DesiredRows, PaneID: binding.PaneID, FloatingID: binding.FloatingID,
	}
	existing, hasExisting := store.Views[binding.ViewID]
	if hasExisting {
		candidate.HadBinding = true
		if candidate.PaneID == "" {
			candidate.PaneID = existing.PaneID
		}
		if candidate.FloatingID == "" {
			candidate.FloatingID = existing.FloatingID
		}
		if candidate.DesiredCols <= 0 {
			candidate.DesiredCols = existing.DesiredCols
		}
		if candidate.DesiredRows <= 0 {
			candidate.DesiredRows = existing.DesiredRows
		}
		if candidate.ResizeRole == "" {
			candidate.ResizeRole = existing.ResizeRole
		}
		if candidate.SurfaceID == "" {
			candidate.SurfaceID = existing.SurfaceID
		}
		existing.AttachPending = true
		existing.AttachCandidate = &candidate
		store.Views = cloneTerminalViewBindings(store.Views)
		store.Views[binding.ViewID] = existing
		return store, candidate
	}
	binding.Channel = 0
	binding.Attached = false
	binding.AttachPending = true
	binding.AttachCandidate = &candidate
	binding.CanResize = false
	binding.LastError = ""
	if binding.PaneID != "" {
		return store.BindPane(binding), candidate
	}
	if binding.FloatingID != "" {
		return store.BindFloating(binding), candidate
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[binding.ViewID] = binding
	return store, candidate
}

// FailAttach 只结束 operation ID 精确匹配的 candidate；已有 committed binding 保持可用且不写入 candidate error。
func (store TerminalViewStore) FailAttach(viewID, operationID, message string) (TerminalViewStore, bool) {
	binding, ok := store.Views[viewID]
	if !ok || binding.AttachCandidate == nil || binding.AttachCandidate.OperationID != operationID {
		return store, false
	}
	hadBinding := binding.AttachCandidate.HadBinding
	binding.AttachPending = false
	binding.AttachCandidate = nil
	if !binding.Attached && !hadBinding {
		store.Views = cloneTerminalViewBindings(store.Views)
		store.PaneViews = cloneTerminalViewIDs(store.PaneViews)
		store.FloatingViews = cloneTerminalViewIDs(store.FloatingViews)
		delete(store.Views, viewID)
		if binding.PaneID != "" {
			delete(store.PaneViews, binding.PaneID)
		}
		if binding.FloatingID != "" {
			delete(store.FloatingViews, binding.FloatingID)
		}
		return store, true
	}
	if !binding.Attached {
		binding.LastError = message
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = binding
	return store, true
}

// CommitAttach 原子提交当前 candidate，并返回需要在 commit 后精确 detach 的 previous committed binding。
func (store TerminalViewStore) CommitAttach(viewID, operationID string, next TerminalViewBinding) (TerminalViewStore, TerminalViewBinding, bool) {
	current, ok := store.Views[viewID]
	if !ok || current.AttachCandidate == nil || current.AttachCandidate.OperationID != operationID {
		return store, TerminalViewBinding{}, false
	}
	previous := TerminalViewBinding{}
	if current.Attached {
		previous = current
	}
	if next.PaneID == "" {
		next.PaneID = current.AttachCandidate.PaneID
	}
	if next.FloatingID == "" {
		next.FloatingID = current.AttachCandidate.FloatingID
	}
	next.Layout = current.Layout
	next.AttachPending = false
	next.AttachCandidate = nil
	next.LastError = ""
	if next.PaneID != "" {
		return store.BindPane(next), previous, true
	}
	if next.FloatingID != "" {
		return store.BindFloating(next), previous, true
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = next
	return store, previous, true
}

// AttachmentSession 返回 committed binding 的 Proto session stamp 快照。
func (binding TerminalViewBinding) AttachmentSession() *apipb.EndpointSessionStamp {
	if binding.Session == nil {
		return nil
	}
	return proto.Clone(binding.Session).(*apipb.EndpointSessionStamp)
}

func (store TerminalViewStore) MarkTerminalReattaching(terminalID string) TerminalViewStore {
	return store.MarkTerminalRefReattaching(LocalTerminalRef(terminalID))
}

// MarkTerminalRefReattaching 清除指定 endpoint terminal 的旧 attach channel，同时保留 pane/floating 连接意图。
// restart/reconnect 只能影响 owning endpoint；其他 endpoint 下同名 terminal 的 channel 必须保持有效。
func (store TerminalViewStore) MarkTerminalRefReattaching(ref TerminalRef) TerminalViewStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	for viewID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		// restart 后旧 channel 已经属于退出前的 attachment；保留 view 绑定意图，
		// 但必须让输入路径重新 attach 当前 view 后再发送。
		binding.Channel = 0
		binding.Attached = false
		binding.AttachPending = false
		binding.AttachCandidate = nil
		binding.Session = nil
		binding.OperationID = ""
		binding = binding.clearRuntimeResizeControl()
		binding.LastError = ""
		store.Views[viewID] = binding
	}
	return store
}

func (store TerminalViewStore) OwnerBinding(terminalID string) (TerminalViewBinding, bool) {
	return store.OwnerBindingRef(LocalTerminalRef(terminalID))
}

// OwnerBindingRef 返回指定 TerminalRef 的 authoritative resize owner。
// 该查询只在同一个 endpoint terminal 内寻找 owner，不会因为 daemon-local TerminalID 相同而跨机器串扰。
func (store TerminalViewStore) OwnerBindingRef(ref TerminalRef) (TerminalViewBinding, bool) {
	ref = ref.Normalize()
	for _, binding := range store.Views {
		if binding.TerminalRef().Equal(ref) && binding.HasAuthoritativeResizeOwner() {
			return binding, true
		}
	}
	return TerminalViewBinding{}, false
}

func (store TerminalViewStore) ownerIdentityBinding(ref TerminalRef) (TerminalViewBinding, bool) {
	ref = ref.Normalize()
	for _, binding := range store.Views {
		if binding.TerminalRef().Equal(ref) && binding.HasResizeOwner() {
			return binding, true
		}
	}
	return TerminalViewBinding{}, false
}

func (store TerminalViewStore) terminalSizeLocked(ref TerminalRef) bool {
	ref = ref.Normalize()
	if ref.Empty() {
		return false
	}
	for _, binding := range store.Views {
		if binding.TerminalRef().Equal(ref) && binding.SizeLocked {
			return true
		}
	}
	return false
}

func (store TerminalViewStore) clearDetachedOwnerProjectionLocked(detached TerminalViewBinding) {
	if !detached.HasResizeOwner() {
		return
	}
	ref := detached.TerminalRef()
	for viewID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		binding.ResizeRole = TerminalResizeRoleFollower
		binding.CanResize = false
		binding.OwnerSurfaceID = ""
		binding.OwnerViewID = ""
		if !binding.SizeLocked {
			binding.ControlReason = ""
		}
		binding.ResizePending = false
		binding.OwnerAcquirePending = false
		store.Views[viewID] = binding
	}
}

func (binding TerminalViewBinding) clearRuntimeResizeControl() TerminalViewBinding {
	binding.ResizeRole = TerminalResizeRoleFollower
	binding.CanResize = false
	binding.SizeLocked = false
	binding.ControlReason = ""
	binding.OwnerSurfaceID = ""
	binding.OwnerViewID = ""
	binding.ResizeEpoch = 0
	binding.OwnerAcquirePending = false
	return binding
}

func (store TerminalViewStore) TransferResizeOwner(viewID string) TerminalViewStore {
	target, ok := store.Views[viewID]
	if !ok || target.TerminalID == "" {
		return store
	}
	ref := target.TerminalRef()
	store.Views = cloneTerminalViewBindings(store.Views)
	locked := store.terminalSizeLocked(ref)
	for candidateID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		if candidateID == viewID {
			becameOwner := !binding.HasResizeOwner()
			binding.ResizeRole = TerminalResizeRoleOwner
			// 中文说明：本地抢 owner 后不能继续保留旧 core owner identity；
			// 否则后续 layout pass 仍会把当前 view 判成 follower，再把 resize 打回旧 owner。
			binding.OwnerSurfaceID = ""
			binding.OwnerViewID = ""
			binding.ControlReason = ""
			binding = binding.applyTerminalSizeLockProjection(locked)
			if !locked {
				binding.CanResize = true
			}
			if becameOwner {
				// 中文说明：主动抢 owner 是 attachment ownership 变化，尺寸相同也要校验一次。
				binding.ResizePending = true
			}
		} else if binding.ResizeRole == TerminalResizeRoleOwner {
			binding.ResizeRole = TerminalResizeRoleFollower
			binding.CanResize = false
			binding.ResizePending = false
			binding = binding.applyTerminalSizeLockProjection(locked)
		} else {
			binding = binding.applyTerminalSizeLockProjection(locked)
		}
		store.Views[candidateID] = binding
	}
	return store
}

func (store TerminalViewStore) TransferPaneResizeOwner(paneID string) TerminalViewStore {
	return store.TransferResizeOwner(store.PaneViews[paneID])
}

func (store TerminalViewStore) RequestPaneResize(paneID string, cols int, rows int) (TerminalViewStore, TerminalViewResizeDecision) {
	viewID := store.PaneViews[paneID]
	return store.RequestViewResize(viewID, cols, rows)
}

func (store TerminalViewStore) RequestViewResize(viewID string, cols int, rows int) (TerminalViewStore, TerminalViewResizeDecision) {
	binding, ok := store.Views[viewID]
	if !ok || binding.TerminalID == "" {
		return store, TerminalViewResizeDecision{Reason: "missing-view"}
	}
	decision := TerminalViewResizeDecision{Binding: binding}
	if !binding.Attached || binding.Channel == 0 {
		decision.Reason = "not-attached"
		return store, decision
	}
	for _, candidate := range store.BindingsForTerminalRef(binding.TerminalRef()) {
		if candidate.OwnerAcquirePending {
			decision.Reason = "owner-transfer-pending"
			return store, decision
		}
	}
	if binding.SizeLocked {
		decision.Reason = "size-locked"
		return store, decision
	}
	if binding.ResizeRole != TerminalResizeRoleOwner || !binding.CanResize {
		decision.Reason = "not-owner"
		return store, decision
	}
	decision.Allowed = true
	if binding.DesiredCols == cols && binding.DesiredRows == rows && !binding.ResizePending {
		decision.Reason = "unchanged"
		return store, decision
	}
	binding.DesiredCols = cols
	binding.DesiredRows = rows
	binding.ResizePending = false
	binding.RequestSeq++
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = binding
	decision.Binding = binding
	decision.Changed = true
	decision.Seq = binding.RequestSeq
	return store, decision
}

func (store TerminalViewStore) RequestViewResizeOwner(viewID string, cols int, rows int) (TerminalViewStore, TerminalViewResizeDecision) {
	binding, ok := store.Views[viewID]
	if !ok || binding.TerminalID == "" {
		return store, TerminalViewResizeDecision{Reason: "missing-view"}
	}
	decision := TerminalViewResizeDecision{Binding: binding}
	if binding.OwnerAcquirePending {
		decision.Reason = "owner-transfer-pending"
		return store, decision
	}
	if !binding.Attached || binding.Channel == 0 {
		decision.Reason = "not-attached"
		return store, decision
	}
	binding.DesiredCols = cols
	binding.DesiredRows = rows
	binding.OwnerAcquirePending = true
	binding.RequestSeq++
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = binding
	decision.Binding = binding
	decision.Allowed = true
	decision.Changed = true
	decision.Seq = binding.RequestSeq
	return store, decision
}

func (store TerminalViewStore) ApplyResizeResult(viewID string, seq uint64, cols int, rows int, lastError string) (TerminalViewStore, bool) {
	binding, ok := store.Views[viewID]
	if !ok || binding.IsStaleResizeResult(seq) {
		return store, false
	}
	binding.DesiredCols = cols
	binding.DesiredRows = rows
	binding.ResizePending = false
	binding.OwnerAcquirePending = false
	binding.LastError = lastError
	store.Views = cloneTerminalViewBindings(store.Views)
	store.Views[viewID] = binding
	return store, true
}

type TerminalResizeControlProjection struct {
	CanResize      bool
	SizeLocked     bool
	ControlReason  string
	OwnerSurfaceID string
	OwnerViewID    string
	ResizeEpoch    uint64
	ResizeRole     string
	SurfaceID      string
	ViewID         string
}

func (store TerminalViewStore) ApplyResizeControl(viewID string, projection TerminalResizeControlProjection) (TerminalViewStore, bool) {
	binding, ok := store.Views[viewID]
	if !ok {
		return store, false
	}
	binding.OwnerAcquirePending = false
	binding.CanResize = projection.CanResize
	binding.SizeLocked = projection.SizeLocked
	binding.ControlReason = projection.ControlReason
	binding.OwnerSurfaceID = projection.OwnerSurfaceID
	binding.OwnerViewID = projection.OwnerViewID
	binding.ResizeEpoch = projection.ResizeEpoch
	if projection.ResizeRole != "" {
		wasOwner := binding.HasResizeOwner()
		binding.ResizeRole = normalizeTerminalResizeRole(projection.ResizeRole)
		if !wasOwner && binding.HasResizeOwner() {
			// 中文说明：core 投影把当前 view 提升为 owner 后，下一次 layout pass 必须主动校验尺寸。
			binding.ResizePending = true
		}
	}
	if projection.SurfaceID != "" {
		binding.SurfaceID = projection.SurfaceID
	}
	if projection.ViewID != "" {
		binding.ViewID = projection.ViewID
	}
	binding = binding.applyTerminalSizeLockProjection(binding.SizeLocked)
	store.Views = cloneTerminalViewBindings(store.Views)
	if binding.HasResizeOwner() {
		store.demoteResizeOwnersLocked(binding.TerminalRef(), viewID)
	}
	store.Views[viewID] = binding
	return store, true
}

func (store TerminalViewStore) ApplyTerminalResizeControl(terminalID string, projection TerminalResizeControlProjection) TerminalViewStore {
	return store.ApplyTerminalRefResizeControl(LocalTerminalRef(terminalID), projection)
}

// ApplyTerminalRefResizeControl 按 endpoint terminal 应用 daemon resize owner 投影。
// projection 来自 owning daemon，因此只能更新同一个 TerminalRef 下的 view binding。
func (store TerminalViewStore) ApplyTerminalRefResizeControl(ref TerminalRef, projection TerminalResizeControlProjection) TerminalViewStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	if store.terminalResizeControlProjectionStale(ref, projection) {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	for viewID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		viewProjection := projection
		viewProjection.SurfaceID = binding.SurfaceID
		viewProjection.ViewID = binding.ViewID
		if binding.Attached && binding.Channel != 0 && projection.OwnerViewID == binding.ViewID && projection.OwnerSurfaceID == binding.SurfaceID {
			viewProjection.ResizeRole = TerminalResizeRoleOwner
			viewProjection.CanResize = !projection.SizeLocked
		} else {
			viewProjection.ResizeRole = TerminalResizeRoleFollower
			viewProjection.CanResize = false
		}
		binding.CanResize = viewProjection.CanResize
		binding.SizeLocked = viewProjection.SizeLocked
		binding.ControlReason = viewProjection.ControlReason
		binding.OwnerSurfaceID = viewProjection.OwnerSurfaceID
		binding.OwnerViewID = viewProjection.OwnerViewID
		binding.ResizeEpoch = viewProjection.ResizeEpoch
		binding.OwnerAcquirePending = false
		wasOwner := binding.HasResizeOwner()
		binding.ResizeRole = normalizeTerminalResizeRole(viewProjection.ResizeRole)
		if !binding.HasResizeOwner() {
			binding.ResizePending = false
		} else if !wasOwner {
			// 中文说明：外部投影把当前 view 提升为 owner 后，下一轮布局要校验一次 PTY 尺寸。
			binding.ResizePending = true
		}
		binding = binding.applyTerminalSizeLockProjection(binding.SizeLocked)
		store.Views[viewID] = binding
	}
	return store
}

func (store TerminalViewStore) terminalResizeControlProjectionStale(ref TerminalRef, projection TerminalResizeControlProjection) bool {
	ref = ref.Normalize()
	var maxEpoch uint64
	var localOwner TerminalViewBinding
	var hasLocalOwner bool
	var projectedLocalOwner TerminalViewBinding
	var hasProjectedLocalOwner bool
	for _, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		if binding.HasResizeOwner() {
			localOwner = binding
			hasLocalOwner = true
		}
		if projection.OwnerViewID != "" && projection.OwnerSurfaceID != "" &&
			binding.ViewID == projection.OwnerViewID && binding.SurfaceID == projection.OwnerSurfaceID {
			projectedLocalOwner = binding
			hasProjectedLocalOwner = true
		}
		if binding.ResizeEpoch > maxEpoch {
			maxEpoch = binding.ResizeEpoch
		}
		if binding.ResizeRole == TerminalResizeRoleOwner && binding.ResizePending && projection.OwnerViewID != "" {
			if projection.OwnerViewID != binding.ViewID || projection.OwnerSurfaceID != binding.SurfaceID {
				return true
			}
		}
	}
	if hasLocalOwner && hasProjectedLocalOwner && !projectedLocalOwner.HasResizeOwner() &&
		(localOwner.ViewID != projectedLocalOwner.ViewID || localOwner.SurfaceID != projectedLocalOwner.SurfaceID) {
		// 中文说明：本 TUI 刚把 owner 转到另一个 view 后，旧 owner 的异步 resize/metadata
		// 可能带着更晚的 epoch 回来；它不能把本地 follower 再提升回 owner。
		return true
	}
	return projection.ResizeEpoch != 0 && maxEpoch > projection.ResizeEpoch
}

func (store TerminalViewStore) ApplyTerminalSizeLock(terminalID string, locked bool) TerminalViewStore {
	return store.ApplyTerminalRefSizeLock(LocalTerminalRef(terminalID), locked)
}

// ApplyTerminalRefSizeLock 把 terminal 级 size lock 投影到同一个 TerminalRef 的所有 view。
// size lock 是 daemon terminal 级状态，不得传播到其他 endpoint 下同名 terminal。
func (store TerminalViewStore) ApplyTerminalRefSizeLock(ref TerminalRef, locked bool) TerminalViewStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store.Views = cloneTerminalViewBindings(store.Views)
	for viewID, binding := range store.Views {
		if !binding.TerminalRef().Equal(ref) {
			continue
		}
		binding = binding.applyTerminalSizeLockProjection(locked)
		store.Views[viewID] = binding
	}
	return store
}

func (binding TerminalViewBinding) applyTerminalSizeLockProjection(locked bool) TerminalViewBinding {
	binding.SizeLocked = locked
	if locked {
		// 中文说明：Size lock 是 terminal 级最高优先级；owner 身份可以存在，但不能恢复 PTY resize 权限。
		if binding.HasResizeOwner() {
			binding.ResizeRole = TerminalResizeRoleOwner
		}
		binding.CanResize = false
		binding.ControlReason = "size_locked"
		return binding
	}
	if binding.ControlReason == "size_locked" {
		binding.ControlReason = ""
		if binding.HasResizeOwner() {
			binding.CanResize = true
		}
	}
	return binding
}

// TerminalRef 返回该 view binding 连接意图的 endpoint-aware terminal 身份。
// 如果 binding 来自旧 snapshot 或旧测试数据而缺少 EndpointID，返回值会显式归入默认 local endpoint。
func (binding TerminalViewBinding) TerminalRef() TerminalRef {
	return NewTerminalRef(binding.EndpointID, binding.TerminalID)
}

func (binding TerminalViewBinding) withDefaultEndpoint() TerminalViewBinding {
	binding.EndpointID = NormalizeEndpointID(binding.EndpointID)
	return binding
}

func (binding TerminalViewBinding) HasAuthoritativeResizeOwner() bool {
	if !binding.CanResize {
		return false
	}
	return binding.HasResizeOwner()
}

func (binding TerminalViewBinding) HasProjectedResizeOwner() bool {
	if binding.ResizeRole != TerminalResizeRoleOwner {
		return false
	}
	if binding.OwnerViewID == "" || binding.OwnerSurfaceID == "" {
		return false
	}
	return binding.OwnerViewID == binding.ViewID && binding.OwnerSurfaceID == binding.SurfaceID
}

func (binding TerminalViewBinding) HasResizeOwner() bool {
	if binding.ResizeRole != TerminalResizeRoleOwner {
		return false
	}
	if binding.OwnerViewID != "" {
		if binding.OwnerViewID != binding.ViewID {
			return false
		}
		// 中文说明：不同 TUI 实例可以有相同 logical ViewID；core owner 必须同时匹配 surface。
		return binding.OwnerSurfaceID != "" && binding.OwnerSurfaceID == binding.SurfaceID
	}
	return true
}

func (binding TerminalViewBinding) IsStaleResizeResult(seq uint64) bool {
	return seq != 0 && seq < binding.RequestSeq
}

func (layout TerminalViewLayout) Normalize() TerminalViewLayout {
	if layout.Mode == "" {
		layout.Mode = TerminalViewLayoutAuto
	}
	if layout.AlignX == "" {
		layout.AlignX = TerminalViewAlignStart
	}
	if layout.AlignY == "" {
		layout.AlignY = TerminalViewAlignStart
	}
	return layout
}

func (layout TerminalViewLayout) Apply(command TerminalViewLayoutCommand) TerminalViewLayout {
	layout = layout.Normalize()
	switch command.Action {
	case "toggle-lock":
		layout.SizeLocked = !layout.SizeLocked
	case "toggle-layout":
		layout.Mode = nextTerminalViewLayoutMode(layout.Mode)
	case "pan":
		layout.PanX += command.DeltaX
		layout.PanY += command.DeltaY
	case "align":
		// 中文说明：align 是显式 view-local 布局指令；必须切出 full-center mode，
		// 否则 render 会继续强制双轴居中，导致后续 0/$/^/B 看起来不生效。
		layout.Mode = TerminalViewLayoutAuto
		if command.AlignX != "" {
			layout.AlignX = normalizeTerminalViewAlign(command.AlignX)
		}
		if command.AlignY != "" {
			layout.AlignY = normalizeTerminalViewAlign(command.AlignY)
		}
	case "center":
		layout.Mode = TerminalViewLayoutCenter
		layout.PanX = 0
		layout.PanY = 0
		layout.AlignX = TerminalViewAlignCenter
		layout.AlignY = TerminalViewAlignCenter
	case "center-x":
		layout.Mode = TerminalViewLayoutAuto
		layout.PanX = 0
		layout.PanY = 0
		layout.AlignX = TerminalViewAlignCenter
		layout.AlignY = TerminalViewAlignStart
	case "center-y":
		layout.Mode = TerminalViewLayoutAuto
		layout.PanX = 0
		layout.PanY = 0
		layout.AlignX = TerminalViewAlignStart
		layout.AlignY = TerminalViewAlignCenter
	case "reset":
		layout = TerminalViewLayout{}.Normalize()
	}
	return layout.Normalize()
}

func nextTerminalViewLayoutMode(mode string) string {
	switch mode {
	case TerminalViewLayoutAuto:
		return TerminalViewLayoutFit
	case TerminalViewLayoutFit:
		return TerminalViewLayoutCenter
	default:
		return TerminalViewLayoutAuto
	}
}

func normalizeTerminalViewAlign(align string) string {
	switch align {
	case TerminalViewAlignCenter, TerminalViewAlignEnd, TerminalViewAlignBase:
		return align
	default:
		return TerminalViewAlignStart
	}
}

func normalizeTerminalResizeRole(role string) string {
	switch role {
	case TerminalResizeRoleFollower, TerminalResizeRoleObserver:
		return role
	default:
		return TerminalResizeRoleOwner
	}
}

func cloneTerminalViewBindings(values map[string]TerminalViewBinding) map[string]TerminalViewBinding {
	cloned := make(map[string]TerminalViewBinding, len(values)+1)
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneTerminalViewIDs(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values)+1)
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
