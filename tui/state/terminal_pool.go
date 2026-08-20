package state

import (
	"strings"
	"time"
)

type TerminalPoolStatus string

const (
	TerminalPoolIdle    TerminalPoolStatus = ""
	TerminalPoolLoading TerminalPoolStatus = "loading"
	TerminalPoolReady   TerminalPoolStatus = "ready"
	TerminalPoolError   TerminalPoolStatus = "error"
)

// TerminalPoolStore 是 reducer-owned terminal inventory 投影。
// Items 来自 endpoint/daemon list 结果，Last*Ref 字段记录最近一次 endpoint-aware 操作结果；旧 Last*ID 仅兼容当前 local 单 daemon UI 路径。
type TerminalPoolStore struct {
	Status           TerminalPoolStatus
	Items            []TerminalPoolItem
	ResourceHistory  map[TerminalRef][]TerminalResourceUsage
	RequestSeq       uint64
	AppliedSeq       uint64
	LastError        string
	LastCreatedID    string
	LastAttachedID   string
	LastKilledID     string
	LastRemovedID    string
	LastEditedID     string
	LastRestartedID  string
	LastCreatedRef   TerminalRef
	LastAttachedRef  TerminalRef
	LastKilledRef    TerminalRef
	LastRemovedRef   TerminalRef
	LastEditedRef    TerminalRef
	LastRestartedRef TerminalRef
}

// TerminalPoolItem 是 terminal manager/picker 消费的 endpoint-aware terminal 摘要。
// EndpointID + TerminalID 是列表项身份；生命周期、资源和尺寸字段仍是 owning daemon list 响应的只读投影。
type TerminalPoolItem struct {
	EndpointID      EndpointID
	TerminalID      string
	Title           string
	State           string
	CWD             string
	Command         []string
	Tags            map[string]string
	ExitCode        *int
	ExitedAt        time.Time
	LastOutputAt    time.Time
	Cols            int
	Rows            int
	AttachmentCount int
	Resources       TerminalResourceUsage
	ResourceHistory []TerminalResourceUsage
	Attached        bool
}

// TerminalResourceUsage 是 TUI reducer 持有的 terminal 资源诊断投影；
// 真值来自 core list 响应，renderer 只能展示它，不能据此推断 terminal 生命周期。
type TerminalResourceUsage struct {
	PID            int
	CPUPercentX100 int
	MemoryBytes    uint64
	SampledAt      time.Time
}

const terminalResourceHistoryLimit = 64

func (store TerminalPoolStore) RequestList() TerminalPoolStore {
	store.RequestSeq++
	store.Status = TerminalPoolLoading
	store.LastError = ""
	return store
}

// RequestRefresh 只推进 Terminal Manager 的后台 list 序号，不切换 loading 状态。
// 它用于资源/preview 的周期性诊断刷新；TerminalPoolStore 的列表真值仍来自下一次 ApplyList。
func (store TerminalPoolStore) RequestRefresh() TerminalPoolStore {
	store.RequestSeq++
	return store
}

func (store TerminalPoolStore) ApplyList(seq uint64, items []TerminalPoolItem, err string) (TerminalPoolStore, bool) {
	if store.IsStale(seq) {
		return store, false
	}
	store.AppliedSeq = seq
	if err != "" {
		store.Status = TerminalPoolError
		store.LastError = err
		return store, true
	}
	store.Status = TerminalPoolReady
	store.Items = normalizeTerminalPoolItems(cloneTerminalPoolItems(items))
	store.ResourceHistory = reconcileTerminalResourceHistory(store.ResourceHistory, store.Items, nil)
	store.LastError = ""
	return store, true
}

// ApplyEndpointList 应用单个 endpoint 的 terminal list 结果。
// 该入口服务多 endpoint runtime projection：成功只替换同 endpoint 条目，失败不清空任何 terminal，避免局部离线污染其他 daemon 的列表真值。
func (store TerminalPoolStore) ApplyEndpointList(endpointID EndpointID, items []TerminalPoolItem, err string) TerminalPoolStore {
	endpointID = NormalizeEndpointID(endpointID)
	if strings.TrimSpace(err) != "" {
		return store
	}
	nextItems := make([]TerminalPoolItem, 0, len(store.Items)+len(items))
	scopedItems := make([]TerminalPoolItem, 0, len(items))
	for _, item := range store.Items {
		item = normalizeTerminalPoolItem(item)
		if item.EndpointID == endpointID {
			continue
		}
		nextItems = append(nextItems, item)
	}
	for _, item := range items {
		item = normalizeTerminalPoolItem(item)
		item.EndpointID = endpointID
		if item.TerminalID == "" {
			continue
		}
		nextItems = append(nextItems, item)
		scopedItems = append(scopedItems, item)
	}
	store.Status = TerminalPoolReady
	store.Items = normalizeTerminalPoolItems(cloneTerminalPoolItems(nextItems))
	store.ResourceHistory = reconcileTerminalResourceHistory(store.ResourceHistory, scopedItems, &endpointID)
	store.LastError = ""
	return store
}

// ApplyCreated 记录默认 local endpoint 的 create 结果。
// 新增跨 endpoint 创建路径应调用 ApplyCreatedRef，避免同名 TerminalID 串扰。
func (store TerminalPoolStore) ApplyCreated(terminalID string, err string) TerminalPoolStore {
	return store.ApplyCreatedRef(LocalTerminalRef(terminalID), err)
}

// ApplyCreatedRef 记录指定 endpoint terminal 的 create 结果。
// 它同时维护旧 LastCreatedID，供当前本地 UI 文案和测试继续读取 daemon-local ID。
func (store TerminalPoolStore) ApplyCreatedRef(ref TerminalRef, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastCreatedID = ref.TerminalID
	store.LastCreatedRef = ref
	store.LastError = ""
	return store
}

// ApplyAttached 记录默认 local endpoint 的 attach 结果。
// 新增跨 endpoint attach 路径应调用 ApplyAttachedRef。
func (store TerminalPoolStore) ApplyAttached(terminalID string, err string) TerminalPoolStore {
	return store.ApplyAttachedRef(LocalTerminalRef(terminalID), err)
}

// ApplyAttachedRef 按 TerminalRef 标记最近 attach 的 terminal。
// 只有同一个 endpoint 的同一个 terminal 会被标记为 attached；其他 endpoint 下同名 TerminalID 不会被误命中。
func (store TerminalPoolStore) ApplyAttachedRef(ref TerminalRef, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastAttachedID = ref.TerminalID
	store.LastAttachedRef = ref
	store.LastError = ""
	store.Items = markTerminalPoolAttached(store.Items, ref)
	return store
}

// ApplyRestarted 记录默认 local endpoint 的 restart 结果。
func (store TerminalPoolStore) ApplyRestarted(terminalID string, err string) TerminalPoolStore {
	return store.ApplyRestartedRef(LocalTerminalRef(terminalID), err)
}

// ApplyRestartedRef 记录指定 endpoint terminal 的 restart 结果。
func (store TerminalPoolStore) ApplyRestartedRef(ref TerminalRef, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastRestartedID = ref.TerminalID
	store.LastRestartedRef = ref
	store.LastError = ""
	return store
}

// ApplyKilled 记录默认 local endpoint 的 kill 结果。
func (store TerminalPoolStore) ApplyKilled(terminalID string, err string) TerminalPoolStore {
	return store.ApplyKilledRef(LocalTerminalRef(terminalID), err)
}

// ApplyKilledRef 记录指定 endpoint terminal 的 kill 结果。
func (store TerminalPoolStore) ApplyKilledRef(ref TerminalRef, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastKilledID = ref.TerminalID
	store.LastKilledRef = ref
	store.LastError = ""
	return store
}

// ApplyRemoved 记录默认 local endpoint 的 remove 结果。
func (store TerminalPoolStore) ApplyRemoved(terminalID string, err string) TerminalPoolStore {
	return store.ApplyRemovedRef(LocalTerminalRef(terminalID), err)
}

// ApplyRemovedRef 删除指定 TerminalRef 的列表投影。
// 删除范围严格限制在 endpoint + terminal，避免远端和本地同名 TerminalID 互相清除。
func (store TerminalPoolStore) ApplyRemovedRef(ref TerminalRef, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastRemovedID = ref.TerminalID
	store.LastRemovedRef = ref
	store.LastError = ""
	store.Items = removeTerminalPoolItem(store.Items, ref)
	store.ResourceHistory = removeTerminalResourceHistory(store.ResourceHistory, ref)
	return store
}

// ApplyEdited 在 terminal metadata 服务确认成功后更新 Terminal Manager 的本地列表投影；
// TerminalPoolStore 只持有 TUI reducer-owned inventory projection，后台 list 刷新仍是最终校准来源。
func (store TerminalPoolStore) ApplyEdited(terminalID string, title string, tags map[string]string, err string) TerminalPoolStore {
	return store.ApplyEditedRef(LocalTerminalRef(terminalID), title, tags, err)
}

// ApplyEditedRef 在指定 endpoint terminal metadata 服务确认成功后更新本地列表投影。
// title/tags 只更新对应 TerminalRef，不会碰撞其他 endpoint 下同名 terminal。
func (store TerminalPoolStore) ApplyEditedRef(ref TerminalRef, title string, tags map[string]string, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastEditedID = ref.TerminalID
	store.LastEditedRef = ref
	store.LastError = ""
	store.Items = updateTerminalPoolItemMetadata(store.Items, ref, title, tags)
	return store
}

func (store TerminalPoolStore) ApplyTagsEdited(terminalID string, tags map[string]string, err string) TerminalPoolStore {
	return store.ApplyTagsEditedRef(LocalTerminalRef(terminalID), tags, err)
}

// ApplyTagsEditedRef 在指定 endpoint terminal tags 服务确认成功后更新本地列表投影。
func (store TerminalPoolStore) ApplyTagsEditedRef(ref TerminalRef, tags map[string]string, err string) TerminalPoolStore {
	ref = ref.Normalize()
	if err != "" {
		store.LastError = err
		store.Status = TerminalPoolError
		return store
	}
	store.LastEditedID = ref.TerminalID
	store.LastEditedRef = ref
	store.LastError = ""
	store.Items = updateTerminalPoolItemTags(store.Items, ref, tags)
	return store
}

func (store TerminalPoolStore) ApplyAttachmentProjection(terminalID string, attachmentCount int) TerminalPoolStore {
	return store.ApplyAttachmentProjectionRef(LocalTerminalRef(terminalID), attachmentCount)
}

// ApplyAttachmentProjectionRef 更新指定 endpoint terminal 的 attachment count 投影。
func (store TerminalPoolStore) ApplyAttachmentProjectionRef(ref TerminalRef, attachmentCount int) TerminalPoolStore {
	ref = ref.Normalize()
	if ref.Empty() {
		return store
	}
	store.Items = updateTerminalPoolItemAttachmentCount(store.Items, ref, attachmentCount)
	return store
}

func (store TerminalPoolStore) IsStale(seq uint64) bool {
	return seq != 0 && seq < store.RequestSeq
}

func cloneTerminalPoolItems(items []TerminalPoolItem) []TerminalPoolItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]TerminalPoolItem, len(items))
	for i, item := range items {
		cloned[i] = item
		cloned[i].Command = append([]string(nil), item.Command...)
		cloned[i].Tags = cloneStringMap(item.Tags)
		cloned[i].ResourceHistory = cloneTerminalResourceSamples(item.ResourceHistory)
		if item.ExitCode != nil {
			code := *item.ExitCode
			cloned[i].ExitCode = &code
		}
	}
	return cloned
}

// ResourceSamples returns the bounded, chronological resource samples for one terminal.
func (store TerminalPoolStore) ResourceSamples(ref TerminalRef) []TerminalResourceUsage {
	ref = ref.Normalize()
	return cloneTerminalResourceSamples(store.ResourceHistory[ref])
}

func reconcileTerminalResourceHistory(history map[TerminalRef][]TerminalResourceUsage, items []TerminalPoolItem, endpointID *EndpointID) map[TerminalRef][]TerminalResourceUsage {
	next := make(map[TerminalRef][]TerminalResourceUsage)
	if endpointID != nil {
		scope := NormalizeEndpointID(*endpointID)
		for ref, samples := range history {
			ref = ref.Normalize()
			if ref.EndpointID != scope && len(samples) > 0 {
				next[ref] = cloneTerminalResourceSamples(samples)
			}
		}
	}
	for _, item := range items {
		item = normalizeTerminalPoolItem(item)
		ref := item.TerminalRef()
		if ref.Empty() {
			continue
		}
		samples := cloneTerminalResourceSamples(history[ref])
		if len(item.ResourceHistory) > 0 {
			samples = normalizeTerminalResourceSamples(item.ResourceHistory)
		}
		sample := item.Resources
		if !sample.SampledAt.IsZero() && terminalResourceSampleIsNewer(samples, sample) {
			if len(samples) > 0 && samples[len(samples)-1].PID > 0 && sample.PID > 0 && samples[len(samples)-1].PID != sample.PID {
				samples = nil
			}
			samples = append(samples, sample)
			if len(samples) > terminalResourceHistoryLimit {
				samples = cloneTerminalResourceSamples(samples[len(samples)-terminalResourceHistoryLimit:])
			}
		}
		if len(samples) > 0 {
			next[ref] = samples
		}
	}
	if len(next) == 0 {
		return nil
	}
	return next
}

func normalizeTerminalResourceSamples(samples []TerminalResourceUsage) []TerminalResourceUsage {
	var normalized []TerminalResourceUsage
	for _, sample := range samples {
		if sample.SampledAt.IsZero() || !terminalResourceSampleIsNewer(normalized, sample) {
			continue
		}
		if len(normalized) > 0 && normalized[len(normalized)-1].PID > 0 && sample.PID > 0 && normalized[len(normalized)-1].PID != sample.PID {
			normalized = nil
		}
		normalized = append(normalized, sample)
		if len(normalized) > terminalResourceHistoryLimit {
			normalized = normalized[len(normalized)-terminalResourceHistoryLimit:]
		}
	}
	return cloneTerminalResourceSamples(normalized)
}

func terminalResourceSampleIsNewer(samples []TerminalResourceUsage, sample TerminalResourceUsage) bool {
	return len(samples) == 0 || sample.SampledAt.After(samples[len(samples)-1].SampledAt)
}

func removeTerminalResourceHistory(history map[TerminalRef][]TerminalResourceUsage, ref TerminalRef) map[TerminalRef][]TerminalResourceUsage {
	ref = ref.Normalize()
	if len(history) == 0 {
		return nil
	}
	next := make(map[TerminalRef][]TerminalResourceUsage, len(history)-1)
	for existingRef, samples := range history {
		existingRef = existingRef.Normalize()
		if existingRef.Equal(ref) || len(samples) == 0 {
			continue
		}
		next[existingRef] = cloneTerminalResourceSamples(samples)
	}
	if len(next) == 0 {
		return nil
	}
	return next
}

func cloneTerminalResourceSamples(samples []TerminalResourceUsage) []TerminalResourceUsage {
	if len(samples) == 0 {
		return nil
	}
	return append([]TerminalResourceUsage(nil), samples...)
}

func markTerminalPoolAttached(items []TerminalPoolItem, ref TerminalRef) []TerminalPoolItem {
	ref = ref.Normalize()
	cloned := cloneTerminalPoolItems(items)
	for index := range cloned {
		cloned[index] = normalizeTerminalPoolItem(cloned[index])
		cloned[index].Attached = cloned[index].TerminalRef().Equal(ref)
	}
	return cloned
}

func removeTerminalPoolItem(items []TerminalPoolItem, ref TerminalRef) []TerminalPoolItem {
	ref = ref.Normalize()
	if ref.Empty() {
		return cloneTerminalPoolItems(items)
	}
	out := make([]TerminalPoolItem, 0, len(items))
	for _, item := range items {
		item = normalizeTerminalPoolItem(item)
		if item.TerminalRef().Equal(ref) {
			continue
		}
		out = append(out, item)
	}
	return cloneTerminalPoolItems(out)
}

func updateTerminalPoolItemTags(items []TerminalPoolItem, ref TerminalRef, tags map[string]string) []TerminalPoolItem {
	ref = ref.Normalize()
	cloned := cloneTerminalPoolItems(items)
	for index := range cloned {
		cloned[index] = normalizeTerminalPoolItem(cloned[index])
		if cloned[index].TerminalRef().Equal(ref) {
			cloned[index].Tags = cloneStringMap(tags)
		}
	}
	return cloned
}

func updateTerminalPoolItemMetadata(items []TerminalPoolItem, ref TerminalRef, title string, tags map[string]string) []TerminalPoolItem {
	ref = ref.Normalize()
	cloned := cloneTerminalPoolItems(items)
	for index := range cloned {
		cloned[index] = normalizeTerminalPoolItem(cloned[index])
		if cloned[index].TerminalRef().Equal(ref) {
			if title != "" {
				cloned[index].Title = title
			}
			cloned[index].Tags = cloneStringMap(tags)
		}
	}
	return cloned
}

func updateTerminalPoolItemAttachmentCount(items []TerminalPoolItem, ref TerminalRef, attachmentCount int) []TerminalPoolItem {
	ref = ref.Normalize()
	cloned := cloneTerminalPoolItems(items)
	for index := range cloned {
		cloned[index] = normalizeTerminalPoolItem(cloned[index])
		if cloned[index].TerminalRef().Equal(ref) {
			cloned[index].AttachmentCount = attachmentCount
		}
	}
	return cloned
}

// TerminalRef 返回该 terminal pool 条目的 endpoint-aware 身份。
// list 结果如果来自旧本地 service 而没有 EndpointID，会被归入默认 local endpoint。
func (item TerminalPoolItem) TerminalRef() TerminalRef {
	return NewTerminalRef(item.EndpointID, item.TerminalID)
}

func normalizeTerminalPoolItems(items []TerminalPoolItem) []TerminalPoolItem {
	for index := range items {
		items[index] = normalizeTerminalPoolItem(items[index])
	}
	return items
}

func normalizeTerminalPoolItem(item TerminalPoolItem) TerminalPoolItem {
	item.EndpointID = NormalizeEndpointID(item.EndpointID)
	return item
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

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
