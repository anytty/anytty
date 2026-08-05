package endpoint

import "fmt"

// RouteAvailabilityReason 是 Go planner policy 对持久 Route 当前不可参与连接的稳定原因。
// 它只描述配置、平台 primitive、credential 索引和 managed eligibility，不包含网络探测结果或 secret。
type RouteAvailabilityReason string

const (
	// RouteAvailabilityAvailable 表示当前 Route 具备进入 planner 的配置与平台前置条件。
	RouteAvailabilityAvailable RouteAvailabilityReason = "available"
	// RouteAvailabilityDisabled 表示用户或配置禁用了该 Route。
	RouteAvailabilityDisabled RouteAvailabilityReason = "route_disabled"
	// RouteAvailabilityPlatformUnsupported 表示当前客户端没有该 Route kind 的连接 primitive。
	RouteAvailabilityPlatformUnsupported RouteAvailabilityReason = "platform_unsupported"
	// RouteAvailabilityCredentialUnavailable 表示 secure-store 中缺少 Route 引用的 credential。
	RouteAvailabilityCredentialUnavailable RouteAvailabilityReason = "credential_unavailable"
	// RouteAvailabilityCloudUnavailable 表示 managed Route 已配置，但当前账号或 Cloud eligibility 不允许使用。
	RouteAvailabilityCloudUnavailable RouteAvailabilityReason = "cloud_unavailable"
)

// RouteAvailability 是单条持久 Route 的 planner 前置条件投影。
// RouteID/Kind 来自 registry；Available/Reason 必须由 EvaluateRouteAvailability 计算，UI 不得自行推断。
type RouteAvailability struct {
	RouteID   RouteID
	Kind      RouteKind
	Available bool
	Reason    RouteAvailabilityReason
}

// RouteAvailabilityRequest 是 Route 可用性纯评估的完整输入。
// Endpoint 是用户持久配置；PlanningEndpoint 是 composition root 应用 managed eligibility 后的副本。
type RouteAvailabilityRequest struct {
	Endpoint                Endpoint
	PlanningEndpoint        Endpoint
	SupportedRouteKinds     []RouteKind
	AvailableCredentialRefs []string
}

// ConnectionPolicy 是允许客户端调整的稳定选路子集；它只作用于下一代 session。
type ConnectionPolicy struct {
	RoutePreference RoutePreference
	CloudRelayMode  RelayMode
	RelayTransport  RelayTransport
}

// SetEndpointEnabled 在 registry 副本中更新 Endpoint 开关，并在关闭当前 default 时
// 优先选择仍启用的 local Endpoint，否则选择稳定排序后的首个启用 Endpoint。
// 该操作只改变客户端连接资格，不删除 Endpoint、Route 或 credential 引用。
func SetEndpointEnabled(registry Registry, endpointID EndpointID, enabled bool) (Registry, error) {
	next, err := registry.Normalize()
	if err != nil {
		return Registry{}, err
	}
	endpointID = normalizeEndpointID(endpointID)
	target, ok := next.Endpoints[endpointID]
	if !ok {
		return Registry{}, connectionError(ErrorConfig, "endpoint %q does not exist", endpointID)
	}
	target.Enabled = enabled
	next.Endpoints[endpointID] = target
	if enabled || next.Default != endpointID {
		return next.Normalize()
	}

	var fallback EndpointID
	for _, candidate := range next.List() {
		if !candidate.Enabled {
			continue
		}
		if fallback == "" {
			fallback = candidate.ID
		}
		for _, route := range candidate.RouteList() {
			if route.Enabled && route.Kind == RouteLocalUnix {
				next.Default = candidate.ID
				return next.Normalize()
			}
		}
	}
	if fallback == "" {
		return Registry{}, connectionError(ErrorConfig, "endpoint registry requires at least one enabled default endpoint")
	}
	next.Default = fallback
	return next.Normalize()
}

// SetConnectionPolicy 在 registry 深拷贝中原子更新 Endpoint 选路偏好和全部 managed Route 的 Relay 约束。
func SetConnectionPolicy(registry Registry, endpointID EndpointID, policy ConnectionPolicy) (Registry, error) {
	switch policy.RoutePreference {
	case RoutePreferenceAuto, RoutePreferenceDirect, RoutePreferenceSSH, RoutePreferenceManagedCloud:
	default:
		return Registry{}, connectionError(ErrorConfig, "unknown route preference %q", policy.RoutePreference)
	}
	switch policy.CloudRelayMode {
	case RelayAuto, RelayDirect, RelayOnly, RelaySmart:
	default:
		return Registry{}, connectionError(ErrorConfig, "unknown Cloud relay mode %q", policy.CloudRelayMode)
	}
	switch policy.RelayTransport {
	case RelayTransportAuto, RelayTransportUDP, RelayTransportTCP:
	default:
		return Registry{}, connectionError(ErrorConfig, "unknown Relay transport %q", policy.RelayTransport)
	}
	next, err := registry.Normalize()
	if err != nil {
		return Registry{}, err
	}
	target, ok := next.Endpoints[normalizeEndpointID(endpointID)]
	if !ok {
		return Registry{}, connectionError(ErrorConfig, "endpoint %q does not exist", endpointID)
	}
	target.SelectionPolicy.RoutePreference = policy.RoutePreference
	for routeID, route := range target.Routes {
		if route.Kind != RouteManagedWebRTC {
			continue
		}
		route.RelayMode = policy.CloudRelayMode
		route.RelayTransport = policy.RelayTransport
		route.PolicySource = SourceUser
		target.Routes[routeID] = route
	}
	next.Endpoints[target.ID] = target
	return next.Normalize()
}

// EvaluateRouteAvailability 使用与 RouteSelectionPlanner 相同的 route kind 和 credential 规则生成稳定投影。
// 该函数不 dial、不读取 secure store、不访问 Cloud；调用方必须先提供当前 generation 的能力索引。
func EvaluateRouteAvailability(request RouteAvailabilityRequest) ([]RouteAvailability, error) {
	target := cloneEndpoint(request.Endpoint)
	if err := target.Validate(); err != nil {
		return nil, err
	}
	planningTarget := cloneEndpoint(request.PlanningEndpoint)
	if planningTarget.ID == "" {
		planningTarget = cloneEndpoint(target)
	}
	if planningTarget.ID != target.ID {
		return nil, connectionError(ErrorConfig, "route availability endpoint %q does not match planning endpoint %q", target.ID, planningTarget.ID)
	}
	if err := planningTarget.Validate(); err != nil {
		return nil, err
	}
	supported, err := routeKindSet(request.SupportedRouteKinds)
	if err != nil {
		return nil, err
	}
	credentials, err := credentialRefSet(request.AvailableCredentialRefs)
	if err != nil {
		return nil, err
	}
	result := make([]RouteAvailability, 0, len(target.Routes))
	for _, route := range target.RouteList() {
		item := RouteAvailability{RouteID: route.ID, Kind: route.Kind, Reason: RouteAvailabilityDisabled}
		if !route.Enabled {
			result = append(result, item)
			continue
		}
		if route.Kind == RouteManagedWebRTC {
			planned, ok := planningTarget.Routes[route.ID]
			if !ok || !planned.Enabled {
				item.Reason = RouteAvailabilityCloudUnavailable
				result = append(result, item)
				continue
			}
		}
		if _, ok := supported[route.Kind]; !ok {
			item.Reason = RouteAvailabilityPlatformUnsupported
			result = append(result, item)
			continue
		}
		if missingRouteCredential(route, credentials) != "" {
			item.Reason = RouteAvailabilityCredentialUnavailable
			result = append(result, item)
			continue
		}
		item.Available = true
		item.Reason = RouteAvailabilityAvailable
		result = append(result, item)
	}
	return result, nil
}

// SetAutomaticRoutePriorities 在 registry 深拷贝中原子替换一个 Endpoint 的完整自动 Route 优先级集合。
// priorities 必须精确覆盖全部 enabled 且非 manual-only Route；全部 nil 表示 full race，全部非 nil 表示按较小数值优先、相同数值并发。
func SetAutomaticRoutePriorities(registry Registry, endpointID EndpointID, priorities map[RouteID]*int) (Registry, error) {
	next, err := registry.Normalize()
	if err != nil {
		return Registry{}, err
	}
	target, ok := next.Endpoints[normalizeEndpointID(endpointID)]
	if !ok {
		return Registry{}, connectionError(ErrorConfig, "endpoint %q does not exist", endpointID)
	}
	automatic := make(map[RouteID]struct{})
	for _, route := range target.RouteList() {
		if route.Enabled && !route.ManualOnly {
			automatic[route.ID] = struct{}{}
		}
	}
	if len(priorities) != len(automatic) {
		return Registry{}, connectionError(ErrorConfig, "endpoint %q route priority update must cover every enabled automatic route", target.ID)
	}
	for routeID := range priorities {
		if _, ok := automatic[routeID]; !ok {
			return Registry{}, connectionError(ErrorConfig, "endpoint %q route %q is not an enabled automatic route", target.ID, routeID)
		}
	}
	anyPriority, allPriority := false, true
	for routeID, priority := range priorities {
		anyPriority = anyPriority || priority != nil
		allPriority = allPriority && priority != nil
		route := target.Routes[routeID]
		route.Priority = clonePriority(priority)
		route.PolicySource = SourceUser
		target.Routes[routeID] = route
	}
	if anyPriority && !allPriority {
		return Registry{}, fmt.Errorf("endpoint %q route priorities must be all empty or all numeric", target.ID)
	}
	next.Endpoints[target.ID] = target
	return next.Normalize()
}
