package state

import (
	"sort"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
)

const (
	// EndpointTransportLocal 表示通过本机 unix socket 访问 daemon。
	// 该值只用于 TUI reducer/view-model 展示，真实 dial 仍由 client runtime 和 transport adapter 负责。
	EndpointTransportLocal EndpointTransportKind = "local-unix"
	// EndpointTransportSSH 表示通过 SSH 访问远端 daemon。
	// SSH host key 与认证结果不由 label 或 endpoint id 表达，必须在 transport 连接阶段处理。
	EndpointTransportSSH EndpointTransportKind = "ssh-webrtc-tcp"
	// EndpointTransportHubP2P 表示 managed WebRTC transport 的 UI 投影。
	// 展示层只消费连接阶段和实际路径，DeviceIdentity 与 CapabilityGrant 仍由公开 remote transport 验证。
	EndpointTransportHubP2P EndpointTransportKind = "managed-webrtc"
	// EndpointTransportMulti 是多 route Endpoint 的纯展示摘要。
	// 它不参与 dial；真实 route 集合始终来自 EndpointItem.Routes。
	EndpointTransportMulti EndpointTransportKind = "multi-route"
)

const (
	// EndpointConnectAuto 表示 endpoint 会在启动或恢复需要时自动连接。
	// 它来自 endpoints.yaml 的 connect_mode，不表示当前已经 online。
	EndpointConnectAuto EndpointConnectMode = "auto"
	// EndpointConnectOnDemand 表示 endpoint 只在 picker 展开、搜索命中或可见 restore 需要时连接。
	EndpointConnectOnDemand EndpointConnectMode = "on_demand"
	// EndpointConnectManual 表示 endpoint 只响应显式 connect action。
	EndpointConnectManual EndpointConnectMode = "manual"
)

const (
	// EndpointStatusUnknown 表示 reducer 尚未收到 registry 或连接状态。
	// renderer 不应把它解释成 online，只能展示为未初始化状态。
	EndpointStatusUnknown EndpointStatusKind = ""
	// EndpointStatusConnected 表示该 endpoint 最近一次 list/connect 成功。
	// 这只是 TUI 已知的运行时投影，terminal lifecycle truth 仍在 owning daemon。
	EndpointStatusConnected EndpointStatusKind = "connected"
	// EndpointStatusConnecting 表示 client runtime 正在建立连接。
	// 该值只提供展示状态，真实连接流程按 CONN003 的 client runtime 切片接入。
	EndpointStatusConnecting EndpointStatusKind = "connecting"
	// EndpointStatusAuto 表示 auto endpoint 已配置但当前没有已知连接结果。
	// 它和 connect mode 同名，是为了让 picker 能展示启动期的连接策略状态。
	EndpointStatusAuto EndpointStatusKind = "auto"
	// EndpointStatusOnDemand 表示 on_demand endpoint 当前处于待需连接状态。
	EndpointStatusOnDemand EndpointStatusKind = "on_demand"
	// EndpointStatusManual 表示 manual endpoint 当前等待显式 connect action。
	EndpointStatusManual EndpointStatusKind = "manual"
	// EndpointStatusDisabled 表示 registry 中该 endpoint 被禁用。
	// 禁用不会自动断开已有 session，但禁止未来自动连接、恢复和创建。
	EndpointStatusDisabled EndpointStatusKind = "disabled"
	// EndpointStatusOffline 表示该 endpoint 最近一次连接或 list 失败。
	// 该失败只能影响本 endpoint，不能清空其他 endpoint 的 terminal pool。
	EndpointStatusOffline EndpointStatusKind = "offline"
	// EndpointStatusReconnectRequired 表示 registry reload 后 dial identity 已变化。
	// 当前 session 不能热切换，必须等显式 reconnect/disconnect 后再使用新配置。
	EndpointStatusReconnectRequired EndpointStatusKind = "reconnect-required"
	// EndpointStatusUnregistered 表示运行时仍有该 endpoint 的 session 或 terminal，但 registry 已删除。
	// 已有展示可以继续存在，重启后 workbench binding 应进入 unresolved。
	EndpointStatusUnregistered EndpointStatusKind = "unregistered"
)

const (
	// EndpointErrorUnknown 表示当前 endpoint 没有可分类错误，或错误还未进入 reducer。
	EndpointErrorUnknown EndpointErrorKind = ""
	// EndpointErrorTransportClosed 表示已有 transport 主动关闭，例如 SSH 子进程退出。
	// 这是 endpoint 连接生命周期事件，不应作为全局 toast 独占展示。
	EndpointErrorTransportClosed EndpointErrorKind = "transport-closed"
	// EndpointErrorTransportDial 表示 transport 建连失败，通常发生在 lazy dial 或 auto connect 阶段。
	EndpointErrorTransportDial EndpointErrorKind = "transport-dial"
	// EndpointErrorAuth 表示认证失败；该分类只服务 UI 展示，不替代 SSH/transport 层的安全判断。
	EndpointErrorAuth EndpointErrorKind = "auth"
	// EndpointErrorHostKey 表示 host key 校验失败，必须作为高风险 endpoint 状态展示。
	EndpointErrorHostKey EndpointErrorKind = "host-key"
	// EndpointErrorRemoteDaemon 表示远端 anytty daemon、stdio-proxy 或 socket 不可用。
	EndpointErrorRemoteDaemon EndpointErrorKind = "remote-daemon"
	// EndpointErrorProtocol 表示 transport 已连通但 protocol 层返回错误或断开。
	EndpointErrorProtocol EndpointErrorKind = "protocol"
	// EndpointErrorConfig 表示 registry、disabled endpoint 或 unsupported transport 这类本地配置错误。
	EndpointErrorConfig EndpointErrorKind = "config"
	// EndpointErrorUnavailable 表示无法进一步分类的连接不可达错误。
	EndpointErrorUnavailable EndpointErrorKind = "unavailable"
	// EndpointErrorEntitlement 表示 Cloud 套餐、Relay 流量或并发限制，不应被展示为普通网络故障。
	EndpointErrorEntitlement EndpointErrorKind = "cloud-entitlement"
)

// EndpointTransportKind 是 TUI 展示层使用的 transport 枚举。
// 它只表达到达 daemon 的方式，不承担安全身份、认证状态或 host key 校验。
type EndpointTransportKind string

// EndpointConnectMode 是 TUI 展示层使用的连接策略枚举。
// 它来自 connection registry，只影响未来连接时机，不热切换已连接 session。
type EndpointConnectMode string

// EndpointStatusKind 是 client runtime adapter 回投给 reducer 的运行时展示状态。
// renderer 只能消费该状态；真正的 terminal list、history 和 lifecycle 仍属于 owning daemon。
type EndpointStatusKind string

// EndpointErrorKind 是 endpoint-scoped 失败类型投影。
// 它来自 transport/protocol/service 错误分类，只用于 TUI 展示和局部状态标记；
// renderer 不得根据它重试、fallback 或改写 terminal lifecycle truth。
type EndpointErrorKind string

// EndpointRouteItem 是 reducer-owned 的脱敏 route 配置投影。
// DialIdentity 只用于判断 registry reload 是否要求 reconnect，不包含 credential body 或 runtime Transport。
type EndpointRouteItem struct {
	ID                 endpointdomain.RouteID
	Kind               EndpointTransportKind
	Enabled            bool
	ManualOnly         bool
	Priority           *int
	RelayMode          endpointdomain.RelayMode
	RelayTransport     endpointdomain.RelayTransport
	AvailabilityKnown  bool
	Available          bool
	AvailabilityReason endpointdomain.RouteAvailabilityReason
	DialIdentity       endpointdomain.DialIdentity
}

// EndpointConnectionSnapshot 是 TUI 对当前 ReadySession selected pair 的只读投影。
type EndpointConnectionSnapshot struct {
	SampledAt           time.Time
	RoundTrip           time.Duration
	LocalCandidateType  string
	RemoteCandidateType string
	LocalAddress        string
	RemoteAddress       string
	LocalPort           uint16
	RemotePort          uint16
	LocalProtocol       string
	RemoteProtocol      string
	RelayTransport      string
	NetworkClass        string
	BytesSent           uint64
	BytesReceived       uint64
	PacketsSent         uint64
	LossEvents          uint64
	Connected           bool
}

// EndpointConnectionPolicy is the TUI port projection of the shared next-session policy.
type EndpointConnectionPolicy struct {
	RoutePreference endpointdomain.RoutePreference
	CloudRelayMode  endpointdomain.RelayMode
	RelayTransport  endpointdomain.RelayTransport
}

// EndpointItem 是 reducer-owned endpoint 展示投影。
// ID 是 workbench/路由主键；Label/Transport/ConnectMode 来自 registry；
// Status/LastError 来自 client runtime adapter 的运行时消息；DefaultCommand/DefaultCWD
// 来自 owning daemon 的 path.defaults，不得由 TUI 本地环境推断。
type EndpointItem struct {
	ID                EndpointID
	Label             string
	DeviceID          string
	DeviceFingerprint string
	Routes            []EndpointRouteItem
	// Transport 是从 Routes 派生的单 route/multi-route 展示摘要，不是持久连接真值。
	Transport   EndpointTransportKind
	ConnectMode EndpointConnectMode
	Enabled     bool
	// RoutePreference 来自共享 Endpoint registry；TUI 只展示该策略，NETUX001 App 与 Go planner 消费同一真值。
	RoutePreference endpointdomain.RoutePreference
	// ActiveRouteID/ConnectionGeneration 来自 SessionOwner event stamp，不能由 route priority 或当前列表位置推断。
	ActiveRouteID        endpointdomain.RouteID
	ConnectionGeneration uint64
	// ConnectionPhase 是 managed WebRTC 当前连接阶段；local/SSH 保持空值。
	// 它来自 endpoint runtime event，只服务 picker/manager 展示，不参与 endpoint identity 或 capability 判断。
	ConnectionPhase EndpointConnectionPhase
	// ObservedPath 是当前 managed WebRTC session 的 direct/single_relay/relay_mesh 投影。
	// registry reload 不提供该值；断线或新 session 会由 endpoint runtime event 更新。
	ObservedPath string
	// RouteSelectionReason 是当前 SmartRoute session 的稳定公开原因；非 SmartRoute endpoint 保持为空。
	RouteSelectionReason string
	ConnectionSnapshot   EndpointConnectionSnapshot
	Status               EndpointStatusKind
	LastError            string
	LastErrorKind        EndpointErrorKind
	TerminalCount        int
	ReconnectRequired    bool
	Unregistered         bool
	DefaultCommand       []string
	DefaultCWD           string
	DefaultsLoaded       bool
	DefaultsError        string
}

// EndpointStore 是 TUI/client 侧 endpoint registry 与运行时连接状态的 reducer-owned 投影。
// endpoints.yaml 是期望状态，已连接 session 是运行时事实；该 store 只保存两者 diff 后的 UI 可见结果。
type EndpointStore struct {
	Items []EndpointItem
}

// ApplyConnectionProjection 合并 adapter 返回的最新 registry/planner policy，同时保留 SessionOwner runtime 状态。
// projection 不得覆盖 active route、generation、phase、path 或 endpoint-scoped error；这些字段只能由 runtime event 更新。
func (store EndpointStore) ApplyConnectionProjection(projection EndpointStore) EndpointStore {
	next := projection.Normalize()
	for index := range next.Items {
		previous, ok := store.Endpoint(next.Items[index].ID)
		if !ok {
			continue
		}
		item := next.Items[index]
		item.Status = previous.Status
		item.LastError = previous.LastError
		item.LastErrorKind = previous.LastErrorKind
		item.TerminalCount = previous.TerminalCount
		item.ReconnectRequired = previous.ReconnectRequired || previous.RequiresReconnect(item)
		item.ConnectionPhase = previous.ConnectionPhase
		item.ActiveRouteID = previous.ActiveRouteID
		item.ConnectionGeneration = previous.ConnectionGeneration
		item.ObservedPath = previous.ObservedPath
		item.RouteSelectionReason = previous.RouteSelectionReason
		item.ConnectionSnapshot = previous.ConnectionSnapshot
		item.DefaultCommand = append([]string(nil), previous.DefaultCommand...)
		item.DefaultCWD = previous.DefaultCWD
		item.DefaultsLoaded = previous.DefaultsLoaded
		item.DefaultsError = previous.DefaultsError
		next.Items[index] = item
	}
	return next.Normalize()
}

// EndpointItemFromEndpoint 把共享 Endpoint registry 配置转换成 TUI 脱敏投影。
// 该转换不做网络 IO，也不验证凭据；失败和 host key 只能由后续 transport 连接消息回投。
func EndpointItemFromEndpoint(endpoint endpointdomain.Endpoint) EndpointItem {
	endpointID := EndpointID(strings.TrimSpace(string(endpoint.ID)))
	if endpointID == "" {
		endpointID = DefaultEndpointID
	}
	item := EndpointItem{
		ID: NormalizeEndpointID(endpointID), Label: strings.TrimSpace(endpoint.Label),
		DeviceID: strings.TrimSpace(endpoint.DaemonIdentity.DeviceID), DeviceFingerprint: strings.TrimSpace(endpoint.DaemonIdentity.DeviceFingerprint),
		ConnectMode: EndpointConnectMode(strings.TrimSpace(string(endpoint.ConnectMode))), Enabled: endpoint.Enabled,
		RoutePreference: endpoint.SelectionPolicy.RoutePreference,
	}
	for _, route := range endpoint.RouteList() {
		item.Routes = append(item.Routes, EndpointRouteItem{
			ID: route.ID, Kind: EndpointTransportKind(route.Kind), Enabled: route.Enabled, ManualOnly: route.ManualOnly,
			Priority: cloneEndpointPriority(route.Priority), RelayMode: route.RelayMode, RelayTransport: route.RelayTransport, DialIdentity: route.DialIdentity(),
		})
	}
	return item.withDefaults()
}

// ApplyConnectionRegistry 用新的 endpoints.yaml 快照更新 endpoint 展示配置。
// 已存在的运行时状态会按 endpoint id 保留；被删除但仍有运行时状态的 endpoint 会标记为 unregistered。
func (store EndpointStore) ApplyConnectionRegistry(registry endpointdomain.Registry) EndpointStore {
	next := EndpointStore{}
	seen := map[EndpointID]struct{}{}
	for _, endpoint := range registry.List() {
		item := EndpointItemFromEndpoint(endpoint).withDefaults()
		if previous, ok := store.Endpoint(item.ID); ok {
			item.Status = previous.Status
			item.LastError = previous.LastError
			item.LastErrorKind = previous.LastErrorKind
			item.TerminalCount = previous.TerminalCount
			item.ReconnectRequired = previous.ReconnectRequired || previous.RequiresReconnect(item)
			item.ObservedPath = previous.ObservedPath
			item.RouteSelectionReason = previous.RouteSelectionReason
			item.ConnectionPhase = previous.ConnectionPhase
			item.ActiveRouteID = previous.ActiveRouteID
			item.ConnectionGeneration = previous.ConnectionGeneration
			item.ConnectionSnapshot = previous.ConnectionSnapshot
			item.DefaultCommand = append([]string(nil), previous.DefaultCommand...)
			item.DefaultCWD = previous.DefaultCWD
			item.DefaultsLoaded = previous.DefaultsLoaded
			item.DefaultsError = previous.DefaultsError
		}
		next.Items = append(next.Items, item)
		seen[item.ID] = struct{}{}
	}
	for _, item := range store.Items {
		item = item.withDefaults()
		if _, ok := seen[item.ID]; ok {
			continue
		}
		item.Unregistered = true
		item.Status = EndpointStatusUnregistered
		next.Items = append(next.Items, item)
	}
	return next.Normalize()
}

// Normalize 返回带有稳定排序和默认字段的 EndpointStore。
// 它不合并不同 endpoint，也不会把 label 当成 identity；排序只为了 UI 和测试稳定。
func (store EndpointStore) Normalize() EndpointStore {
	if len(store.Items) == 0 {
		return EndpointStore{}
	}
	items := cloneEndpointItems(store.Items)
	for index := range items {
		items[index] = items[index].withDefaults()
	}
	sort.SliceStable(items, func(i, j int) bool {
		return string(items[i].ID) < string(items[j].ID)
	})
	out := make([]EndpointItem, 0, len(items))
	seen := map[EndpointID]struct{}{}
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		out = append(out, item)
		seen[item.ID] = struct{}{}
	}
	return EndpointStore{Items: out}
}

// HasItems 表示 reducer 已经加载了 endpoint registry 或运行时 endpoint 状态。
// renderer 用它决定是否启用多 endpoint 分组展示，避免旧单本地测试在未接 registry 前抖动。
func (store EndpointStore) HasItems() bool {
	return len(store.Items) > 0
}

// Endpoint 返回指定 endpoint 的展示投影。
// 找不到时返回 false；调用方若需要展示 orphan terminal，应显式使用 UnregisteredEndpoint。
func (store EndpointStore) Endpoint(endpointID EndpointID) (EndpointItem, bool) {
	endpointID = NormalizeEndpointID(endpointID)
	for _, item := range store.Items {
		item = item.withDefaults()
		if item.ID == endpointID {
			return item, true
		}
	}
	return EndpointItem{}, false
}

// DisplayEndpoint 返回适合 row view-model 使用的 endpoint 投影。
// 已加载 registry 时，缺失 endpoint 会被标记为 unregistered；未加载 registry 时返回 false 保持旧本地紧凑展示。
func (store EndpointStore) DisplayEndpoint(endpointID EndpointID) (EndpointItem, bool) {
	endpointID = NormalizeEndpointID(endpointID)
	if item, ok := store.Endpoint(endpointID); ok {
		return item, true
	}
	if !store.HasItems() {
		return EndpointItem{}, false
	}
	return UnregisteredEndpoint(endpointID), true
}

// Upsert 插入或替换一个 endpoint 展示投影。
// 它用于测试 harness 和后续 client runtime adapter 消息回投；重复 ID 只保留最后一次写入的状态。
func (store EndpointStore) Upsert(item EndpointItem) EndpointStore {
	item = item.withDefaults()
	items := cloneEndpointItems(store.Items)
	replaced := false
	for index := range items {
		if NormalizeEndpointID(items[index].ID) == item.ID {
			items[index] = item
			replaced = true
			break
		}
	}
	if !replaced {
		items = append(items, item)
	}
	return (EndpointStore{Items: items}).Normalize()
}

// MarkTerminalListResult 记录某个 endpoint 的 terminal list 结果。
// 成功只更新该 endpoint 的数量和 connected 状态；失败只把该 endpoint 标为 offline，不影响其他 endpoint 的列表真值。
func (store EndpointStore) MarkTerminalListResult(endpointID EndpointID, terminalCount int, err string) EndpointStore {
	return store.MarkRuntimeStatus(endpointID, EndpointStatusUnknown, EndpointErrorUnknown, terminalCount, err)
}

// MarkRuntimeStatus 记录 client runtime adapter 主动回投的连接状态。
// Status/ErrorKind 来自 endpoint-scoped service/transport 事件；该方法只更新对应 endpoint，
// 不删除 terminal pool、workbench binding 或其他 endpoint 的状态。
func (store EndpointStore) MarkRuntimeStatus(endpointID EndpointID, status EndpointStatusKind, errorKind EndpointErrorKind, terminalCount int, err string) EndpointStore {
	endpointID = NormalizeEndpointID(endpointID)
	item, ok := store.DisplayEndpoint(endpointID)
	if !ok {
		item = DefaultLocalEndpoint()
		item.ID = endpointID
	}
	item.TerminalCount = terminalCount
	item.LastError = strings.TrimSpace(err)
	item.LastErrorKind = NormalizeEndpointErrorKind(errorKind)
	if item.LastError != "" {
		if item.LastErrorKind == EndpointErrorUnknown {
			item.LastErrorKind = ClassifyEndpointErrorText(item.LastError)
		}
		if status == EndpointStatusUnknown {
			status = EndpointStatusOffline
		}
	} else if status == EndpointStatusUnknown {
		status = EndpointStatusConnected
	}
	if item.LastError == "" && status == EndpointStatusConnected {
		item.LastErrorKind = EndpointErrorUnknown
	}
	item.Status = status
	return store.Upsert(item)
}

// MarkManagedRoute 原子更新单个 managed endpoint 的实际 WebRTC 路径和 SmartRoute 原因。
// 空路径会同时清除旧原因；非法组合保持原状态，避免 renderer 看到新路径和旧诊断混合。
func (store EndpointStore) MarkManagedRoute(endpointID EndpointID, observedPath, selectionReason string) EndpointStore {
	endpointID = NormalizeEndpointID(endpointID)
	item, ok := store.DisplayEndpoint(endpointID)
	if !ok {
		return store
	}
	switch observedPath = strings.TrimSpace(observedPath); observedPath {
	case "", "direct", "single_relay", "relay_mesh":
	default:
		return store
	}
	selectionReason = strings.TrimSpace(selectionReason)
	if observedPath == "" {
		selectionReason = ""
	} else if selectionReason != "" && !isKnownRouteSelectionReason(selectionReason) {
		return store
	}
	item.ObservedPath = observedPath
	item.RouteSelectionReason = selectionReason
	return store.Upsert(item)
}

// MarkRuntimeConnection 原子更新 SessionOwner 发布的 winner route、generation 与实际 managed path。
// generation 小于当前值时保持原状态；新 generation 在 Ready 前清空旧 winner，Offline 只清理当前 generation 的连接详情。
func (store EndpointStore) MarkRuntimeConnection(endpointID EndpointID, routeID string, generation uint64, status EndpointStatusKind, observedPath, selectionReason string) EndpointStore {
	endpointID = NormalizeEndpointID(endpointID)
	item, ok := store.DisplayEndpoint(endpointID)
	if !ok || (item.ConnectionGeneration > 0 && generation == 0) || (generation > 0 && generation < item.ConnectionGeneration) {
		return store
	}
	if generation > item.ConnectionGeneration {
		item.ConnectionGeneration = generation
		item.ActiveRouteID = ""
		item.ObservedPath = ""
		item.RouteSelectionReason = ""
		item.ConnectionSnapshot = EndpointConnectionSnapshot{}
	}
	switch status {
	case EndpointStatusConnected:
		if generation > 0 {
			item.ConnectionGeneration = generation
		}
		item.ActiveRouteID = endpointdomain.RouteID(strings.TrimSpace(routeID))
		switch observedPath = strings.TrimSpace(observedPath); observedPath {
		case "", "direct", "single_relay", "relay_mesh":
			item.ObservedPath = observedPath
		default:
			item.ObservedPath = ""
		}
		selectionReason = strings.TrimSpace(selectionReason)
		if selectionReason == "" || isKnownRouteSelectionReason(selectionReason) {
			item.RouteSelectionReason = selectionReason
		} else {
			item.RouteSelectionReason = ""
		}
	case EndpointStatusOffline:
		item.ActiveRouteID = ""
		item.ObservedPath = ""
		item.RouteSelectionReason = ""
		item.ConnectionSnapshot = EndpointConnectionSnapshot{}
	}
	return store.Upsert(item)
}

// ApplyConnectionSnapshot replaces diagnostics for the selected endpoint without changing lifecycle truth.
func (store EndpointStore) ApplyConnectionSnapshot(endpointID EndpointID, snapshot EndpointConnectionSnapshot) EndpointStore {
	item, ok := store.DisplayEndpoint(NormalizeEndpointID(endpointID))
	if !ok {
		return store
	}
	snapshot.LocalAddress = strings.TrimSpace(snapshot.LocalAddress)
	snapshot.RemoteAddress = strings.TrimSpace(snapshot.RemoteAddress)
	snapshot.LocalCandidateType = strings.TrimSpace(snapshot.LocalCandidateType)
	snapshot.RemoteCandidateType = strings.TrimSpace(snapshot.RemoteCandidateType)
	snapshot.LocalProtocol = strings.TrimSpace(snapshot.LocalProtocol)
	snapshot.RemoteProtocol = strings.TrimSpace(snapshot.RemoteProtocol)
	snapshot.RelayTransport = strings.TrimSpace(snapshot.RelayTransport)
	snapshot.NetworkClass = strings.TrimSpace(snapshot.NetworkClass)
	item.ConnectionSnapshot = snapshot
	return store.Upsert(item)
}

// ApplyRouteAvailability 合并 Go planner policy 对单条 Route 的可用性投影。
// 未返回的 Route 保持 unknown；reducer/renderer 不得按 kind、credential ref 或错误文本补推断。
func (store EndpointStore) ApplyRouteAvailability(endpointID EndpointID, availability []endpointdomain.RouteAvailability) EndpointStore {
	item, ok := store.DisplayEndpoint(endpointID)
	if !ok {
		return store
	}
	byID := make(map[endpointdomain.RouteID]endpointdomain.RouteAvailability, len(availability))
	for _, value := range availability {
		byID[value.RouteID] = value
	}
	for index := range item.Routes {
		value, ok := byID[item.Routes[index].ID]
		item.Routes[index].AvailabilityKnown = ok
		item.Routes[index].Available = ok && value.Available
		if ok {
			item.Routes[index].AvailabilityReason = value.Reason
		} else {
			item.Routes[index].AvailabilityReason = ""
		}
	}
	return store.Upsert(item)
}

// MarkConnectionPhase 更新单个 managed endpoint 的公开连接阶段。
// 未知阶段被忽略；connected/failed 只描述最近一次 dial 结果，Status 仍由 endpoint runtime 消息独立维护。
func (store EndpointStore) MarkConnectionPhase(endpointID EndpointID, phase EndpointConnectionPhase) EndpointStore {
	switch phase {
	case EndpointConnectionIdle, EndpointConnectionResolving, EndpointConnectionSignaling,
		EndpointConnectionConnecting, EndpointConnectionAuthorizing,
		EndpointConnectionConnected, EndpointConnectionFailed:
	default:
		return store
	}
	item, ok := store.DisplayEndpoint(endpointID)
	if !ok || !item.hasRouteKind(EndpointTransportHubP2P) {
		return store
	}
	item.ConnectionPhase = phase
	return store.Upsert(item)
}

func isKnownRouteSelectionReason(reason string) bool {
	switch reason {
	case "first_ready", "route_override", "current_winner", "initial_best", "only_viable", "lower_loss", "direct_unstable", "lower_latency", "lower_score",
		"cost_guard", "minimum_hold", "cooldown", "hysteresis_hold", "insufficient_improvement",
		"current_unavailable", "current_best":
		return true
	default:
		return false
	}
}

// ApplyDefaults 记录某个 endpoint daemon 返回的创建默认值。
// 成功结果成为后续 create prompt/submit 的 truth；失败只记录错误，不用本地环境补洞。
func (store EndpointStore) ApplyDefaults(endpointID EndpointID, command []string, cwd string, err string) EndpointStore {
	endpointID = NormalizeEndpointID(endpointID)
	item, ok := store.DisplayEndpoint(endpointID)
	if !ok {
		item = DefaultLocalEndpoint()
		item.ID = endpointID
	}
	item.DefaultsError = strings.TrimSpace(err)
	if item.DefaultsError == "" {
		item.DefaultCommand = append([]string(nil), command...)
		item.DefaultCWD = strings.TrimSpace(cwd)
		item.DefaultsLoaded = true
	}
	return store.Upsert(item)
}

// DefaultLocalEndpoint 返回缺省 local endpoint 的展示投影。
// 该默认值只用于 TUI state/view-model，不代表已经完成 local socket 连接。
func DefaultLocalEndpoint() EndpointItem {
	return EndpointItem{
		ID:        DefaultEndpointID,
		Label:     string(DefaultEndpointID),
		Transport: EndpointTransportLocal,
		Routes: []EndpointRouteItem{{
			ID: endpointdomain.DefaultLocalRouteID, Kind: EndpointTransportLocal, Enabled: true,
			DialIdentity: endpointdomain.AccessRoute{ID: endpointdomain.DefaultLocalRouteID, Kind: endpointdomain.RouteLocalUnix, Enabled: true, Socket: "auto"}.DialIdentity(),
		}},
		ConnectMode: EndpointConnectAuto,
		Enabled:     true,
		Status:      EndpointStatusAuto,
	}
}

// UnregisteredEndpoint 返回 registry 缺失但运行时仍可见的 endpoint 投影。
// 它用于删除 connection 后继续展示已有 session/terminal，避免 layout 或 terminal pool 被错误清空。
func UnregisteredEndpoint(endpointID EndpointID) EndpointItem {
	return EndpointItem{
		ID:           NormalizeEndpointID(endpointID),
		Label:        string(NormalizeEndpointID(endpointID)),
		Status:       EndpointStatusUnregistered,
		Enabled:      false,
		Unregistered: true,
	}
}

// DisplayLabel 返回 endpoint 在 picker/manager 中显示的机器名称。
// label 缺失时退回 endpoint id；该值只用于 UI，不参与安全身份判断。
func (item EndpointItem) DisplayLabel() string {
	item = item.withDefaults()
	if strings.TrimSpace(item.Label) != "" {
		return strings.TrimSpace(item.Label)
	}
	return string(item.ID)
}

// DisplayStatus 返回 endpoint 当前应展示的状态。
// reconnect-required、unregistered 和 disabled 具有最高优先级，避免连接策略掩盖运行时风险。
func (item EndpointItem) DisplayStatus() EndpointStatusKind {
	item = item.withDefaults()
	switch {
	case item.Unregistered:
		return EndpointStatusUnregistered
	case item.ReconnectRequired:
		return EndpointStatusReconnectRequired
	case !item.Enabled:
		return EndpointStatusDisabled
	case item.Status != EndpointStatusUnknown:
		return item.Status
	}
	switch item.ConnectMode {
	case EndpointConnectOnDemand:
		return EndpointStatusOnDemand
	case EndpointConnectManual:
		return EndpointStatusManual
	case EndpointConnectAuto:
		return EndpointStatusAuto
	default:
		return EndpointStatusUnknown
	}
}

// DisplayErrorLabel 返回 endpoint 错误类型和原始摘要的组合展示文本。
// 错误类型先显示，便于 picker/manager/workbench 在有限宽度内区分 auth、host-key、transport closed 等失败。
func (item EndpointItem) DisplayErrorLabel() string {
	item = item.withDefaults()
	errText := strings.TrimSpace(item.LastError)
	errKind := NormalizeEndpointErrorKind(item.LastErrorKind)
	if errKind == EndpointErrorUnknown && errText != "" {
		errKind = ClassifyEndpointErrorText(errText)
	}
	if errKind == EndpointErrorUnknown {
		return errText
	}
	if errText == "" {
		return string(errKind)
	}
	return string(errKind) + ": " + errText
}

// NormalizeEndpointErrorKind 把未知错误类型归入空值，避免 renderer 展示拼写错误的状态。
func NormalizeEndpointErrorKind(kind EndpointErrorKind) EndpointErrorKind {
	switch kind {
	case EndpointErrorTransportClosed,
		EndpointErrorTransportDial,
		EndpointErrorAuth,
		EndpointErrorHostKey,
		EndpointErrorRemoteDaemon,
		EndpointErrorProtocol,
		EndpointErrorConfig,
		EndpointErrorUnavailable,
		EndpointErrorEntitlement:
		return kind
	default:
		return EndpointErrorUnknown
	}
}

// ClassifyEndpointErrorText 根据服务层错误摘要生成 endpoint 错误类型。
// 该分类只影响 UI 标志；真实认证、host key 和 transport 行为仍由 connection/transport/protocol 层负责。
func ClassifyEndpointErrorText(text string) EndpointErrorKind {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch {
	case lower == "":
		return EndpointErrorUnknown
	case strings.Contains(lower, "host key") ||
		strings.Contains(lower, "remote host identification") ||
		strings.Contains(lower, "known_hosts"):
		return EndpointErrorHostKey
	case strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "publickey") ||
		strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "auth"):
		return EndpointErrorAuth
	case strings.Contains(lower, "stdio-proxy") ||
		strings.Contains(lower, "remote socket") ||
		strings.Contains(lower, "daemon") ||
		strings.Contains(lower, "no such file or directory"):
		return EndpointErrorRemoteDaemon
	case strings.Contains(lower, "ssh transport closed") ||
		strings.Contains(lower, "transport closed") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "connection refused"):
		return EndpointErrorTransportClosed
	case strings.Contains(lower, "dial") ||
		strings.Contains(lower, "connect timeout") ||
		strings.Contains(lower, "no route to host") ||
		strings.Contains(lower, "network is unreachable"):
		return EndpointErrorTransportDial
	case strings.Contains(lower, "protocol error") ||
		strings.Contains(lower, "unexpected frame") ||
		strings.Contains(lower, "eof"):
		return EndpointErrorProtocol
	case strings.Contains(lower, "not registered") ||
		strings.Contains(lower, "disabled") ||
		(strings.Contains(lower, "transport") || strings.Contains(lower, "route")) && strings.Contains(lower, "not connected"):
		return EndpointErrorConfig
	default:
		return EndpointErrorUnavailable
	}
}

// RequiresReconnect 判断 registry reload 后是否改变了 endpoint dial identity。
// 该判断只覆盖 TUI 展示字段，完整 dial identity 仍由 client/endpoint.Endpoint 负责。
func (item EndpointItem) RequiresReconnect(next EndpointItem) bool {
	item = item.withDefaults()
	next = next.withDefaults()
	if item.DeviceID != next.DeviceID || item.DeviceFingerprint != next.DeviceFingerprint || len(item.Routes) != len(next.Routes) {
		return true
	}
	for index := range item.Routes {
		if item.Routes[index].ID != next.Routes[index].ID || item.Routes[index].DialIdentity != next.Routes[index].DialIdentity {
			return true
		}
	}
	return false
}

func (item EndpointItem) withDefaults() EndpointItem {
	item.ID = NormalizeEndpointID(item.ID)
	item.DefaultCommand = append([]string(nil), item.DefaultCommand...)
	item.Routes = cloneEndpointRoutes(item.Routes)
	sort.SliceStable(item.Routes, func(i, j int) bool { return item.Routes[i].ID < item.Routes[j].ID })
	if strings.TrimSpace(item.Label) == "" {
		item.Label = string(item.ID)
	}
	if len(item.Routes) > 0 {
		kinds := map[EndpointTransportKind]struct{}{}
		for _, route := range item.Routes {
			kinds[route.Kind] = struct{}{}
		}
		if len(kinds) == 1 {
			for kind := range kinds {
				item.Transport = kind
			}
		} else if len(kinds) > 1 {
			item.Transport = EndpointTransportMulti
		}
	} else if strings.TrimSpace(string(item.Transport)) == "" {
		if item.ID == DefaultEndpointID {
			item.Transport = EndpointTransportLocal
		}
	}
	if strings.TrimSpace(string(item.ConnectMode)) == "" {
		if item.ID == DefaultEndpointID {
			item.ConnectMode = EndpointConnectAuto
		} else {
			item.ConnectMode = EndpointConnectOnDemand
		}
	}
	return item
}

func cloneEndpointItems(items []EndpointItem) []EndpointItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]EndpointItem, len(items))
	copy(cloned, items)
	for index := range cloned {
		cloned[index].DefaultCommand = append([]string(nil), cloned[index].DefaultCommand...)
		cloned[index].Routes = cloneEndpointRoutes(cloned[index].Routes)
	}
	return cloned
}

func (item EndpointItem) hasRouteKind(kind EndpointTransportKind) bool {
	for _, route := range item.Routes {
		if route.Kind == kind {
			return true
		}
	}
	return item.Transport == kind
}

func cloneEndpointRoutes(routes []EndpointRouteItem) []EndpointRouteItem {
	if routes == nil {
		return nil
	}
	out := make([]EndpointRouteItem, len(routes))
	copy(out, routes)
	for index := range out {
		out[index].Priority = cloneEndpointPriority(out[index].Priority)
	}
	return out
}

func cloneEndpointPriority(priority *int) *int {
	if priority == nil {
		return nil
	}
	value := *priority
	return &value
}
