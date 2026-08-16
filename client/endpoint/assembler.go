package endpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EndpointCandidate 是 Cloud、bootstrap、manual 或 share 向 EndpointAssembler 提交的已验证输入。
// Source 只决定 provenance/覆盖优先级；Identity 才决定能否与 SavedEndpointRegistry 合并。
type EndpointCandidate struct {
	Source                EndpointSource         `json:"source"`
	Identity              DaemonIdentity         `json:"identity"`
	SuggestedLabel        string                 `json:"suggested_label,omitempty"`
	Platform              string                 `json:"platform,omitempty"`
	Routes                []AccessRoute          `json:"routes,omitempty"`
	ConnectMode           ConnectMode            `json:"connect_mode,omitempty"`
	SelectionPolicy       *SelectionPolicy       `json:"selection_policy,omitempty"`
	ApplyClientPolicy     bool                   `json:"apply_client_policy"`
	CredentialDescriptors []CredentialDescriptor `json:"credential_descriptors,omitempty"`
}

// ConfirmedIdentityBinding 表示用户已确认把一个 identity 为空的本地/SSH Endpoint 绑定到已验证 daemon identity。
// Identity 必须同时出现在本次 assembler candidates 中；该输入只表达本地确认结果，不能由 Cloud/bootstrap/share payload 自行指定。
type ConfirmedIdentityBinding struct {
	EndpointID EndpointID     `json:"endpoint_id"`
	Identity   DaemonIdentity `json:"identity"`
}

// EndpointAssemblerInput 汇总当前 SavedEndpointRegistry 与经过各自安全边界验证的候选。
// LAN candidate 只参与内存地址规划，不得作为持久 Routes 输入；调用方应使用 LocalDiscoveryCandidate 单独保存。
type EndpointAssemblerInput struct {
	Registry                  Registry                   `json:"registry"`
	Candidates                []EndpointCandidate        `json:"candidates"`
	ConfirmedIdentityBindings []ConfirmedIdentityBinding `json:"confirmed_identity_bindings,omitempty"`
}

// EndpointAssemblerResult 是纯合并事务的结果。
// ResolvedEndpointIDs 与原 Candidates 顺序一一对应，调用方只有在 secure credential 写入成功后才能原子发布 Registry。
type EndpointAssemblerResult struct {
	Registry              Registry               `json:"registry"`
	ResolvedEndpointIDs   []EndpointID           `json:"resolved_endpoint_ids"`
	CredentialDescriptors []CredentialDescriptor `json:"credential_descriptors"`
}

// AssembleEndpoints 按 DeviceFingerprint + DeviceID 合并候选并返回新的 registry snapshot。
// 相同 DeviceID/不同 fingerprint、相同 fingerprint/不同 DeviceID、RouteID 换 kind 均 fail closed；label、地址和来源类型永远不能换 pin。
func AssembleEndpoints(input EndpointAssemblerInput) (EndpointAssemblerResult, error) {
	registry, err := input.Registry.Normalize()
	if err != nil {
		return EndpointAssemblerResult{}, err
	}
	registry = cloneRegistry(registry)
	resolved := make([]EndpointID, len(input.Candidates))
	credentials := make([]CredentialDescriptor, 0)
	type indexedCandidate struct {
		index     int
		candidate EndpointCandidate
	}
	candidates := make([]indexedCandidate, len(input.Candidates))
	candidateIdentities := make(map[string]struct{}, len(input.Candidates))
	for index, candidate := range input.Candidates {
		if candidate.Source == SourceLAN || !validSource(candidate.Source) {
			return EndpointAssemblerResult{}, connectionError(ErrorConfig, "candidate %d has invalid persistent source %q", index, candidate.Source)
		}
		if err := candidate.Identity.Validate(true); err != nil {
			return EndpointAssemblerResult{}, fmt.Errorf("candidate %d: %w", index, err)
		}
		if err := validateCandidateClientPolicy(candidate); err != nil {
			return EndpointAssemblerResult{}, fmt.Errorf("candidate %d: %w", index, err)
		}
		candidates[index] = indexedCandidate{index: index, candidate: cloneCandidate(candidate)}
		candidateIdentities[identityKey(candidate.Identity)] = struct{}{}
	}
	if err := applyConfirmedIdentityBindings(&registry, input.ConfirmedIdentityBindings, candidateIdentities); err != nil {
		return EndpointAssemblerResult{}, err
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i].candidate, candidates[j].candidate
		if left.Identity.DeviceFingerprint != right.Identity.DeviceFingerprint {
			return left.Identity.DeviceFingerprint < right.Identity.DeviceFingerprint
		}
		if sourceRank(left.Source) != sourceRank(right.Source) {
			return sourceRank(left.Source) < sourceRank(right.Source)
		}
		return candidateSortKey(left) < candidateSortKey(right)
	})
	for _, item := range candidates {
		candidate := item.candidate
		endpointID, endpoint, found, err := findEndpointByIdentity(registry, candidate.Identity)
		if err != nil {
			return EndpointAssemblerResult{}, err
		}
		applyPolicy := candidateOwnsClientPolicy(candidate)
		if !found {
			endpointID = deriveEndpointID(candidate.Identity.DeviceFingerprint, registry.Endpoints)
			endpoint = Endpoint{
				ID: endpointID, Label: strings.TrimSpace(candidate.SuggestedLabel), LabelSource: candidate.Source,
				DaemonIdentity: candidate.Identity, Platform: candidate.Platform, ConnectMode: ConnectOnDemand, Enabled: true,
				SelectionPolicy: SelectionPolicy{}, Routes: map[RouteID]AccessRoute{},
			}
			if endpoint.Label == "" {
				endpoint.Label = candidate.Identity.DeviceID
			}
		} else if applyPolicy && endpoint.LabelSource != SourceUser && sourceRank(candidate.Source) >= sourceRank(endpoint.LabelSource) && strings.TrimSpace(candidate.SuggestedLabel) != "" {
			endpoint.Label = strings.TrimSpace(candidate.SuggestedLabel)
			endpoint.LabelSource = candidate.Source
		}
		if candidate.Platform != "" {
			endpoint.Platform = candidate.Platform
		}
		if applyPolicy && candidate.ConnectMode != "" {
			endpoint.ConnectMode = candidate.ConnectMode
		}
		if applyPolicy && candidate.SelectionPolicy != nil {
			endpoint.SelectionPolicy = *candidate.SelectionPolicy
		}
		routes := append([]AccessRoute(nil), candidate.Routes...)
		sort.SliceStable(routes, func(i, j int) bool { return routes[i].ID < routes[j].ID })
		for _, route := range routes {
			if err := validateIdentifier("route", string(route.ID)); err != nil {
				return EndpointAssemblerResult{}, fmt.Errorf("candidate %d: %w", item.index, err)
			}
			if route.Source != "" && route.Source != candidate.Source {
				return EndpointAssemblerResult{}, connectionError(ErrorConfig, "candidate %d route %q source %q does not match candidate source %q", item.index, route.ID, route.Source, candidate.Source)
			}
			if route.PolicySource != "" && route.PolicySource != candidate.Source {
				return EndpointAssemblerResult{}, connectionError(ErrorConfig, "candidate %d route %q policy source %q does not match candidate source %q", item.index, route.ID, route.PolicySource, candidate.Source)
			}
			route.Source = candidate.Source
			route.PolicySource = candidate.Source
			route = route.withDefaults()
			if err := route.Validate(candidate.Identity); err != nil {
				return EndpointAssemblerResult{}, fmt.Errorf("candidate %d: %w", item.index, err)
			}
			if existing, ok := endpoint.Routes[route.ID]; ok {
				if existing.Kind != route.Kind {
					return EndpointAssemblerResult{}, connectionError(ErrorRouteConflict, "endpoint %q route %q changes kind from %q to %q", endpoint.ID, route.ID, existing.Kind, route.Kind)
				}
				route = mergeRoute(existing, route, applyPolicy, candidate.Source == SourceShare && candidate.ApplyClientPolicy)
			} else if !applyPolicy {
				// 外部来源只提供 route 配置；新 route 的默认启用策略仍归当前客户端所有。
				route.PolicySource = SourceLocal
				if hasPrioritizedAutomaticRoute(endpoint.Routes) {
					// 已有分组策略时，新 route 先保持手动可选，等用户显式纳入竞速。
					route.ManualOnly = true
				}
			}
			endpoint.Routes[route.ID] = route
		}
		registry.Endpoints[endpointID] = endpoint
		if registry.Default == "" {
			registry.Default = endpointID
		}
		resolved[item.index] = endpointID
		for _, descriptor := range candidate.CredentialDescriptors {
			if err := validateCredentialDescriptor(descriptor); err != nil {
				return EndpointAssemblerResult{}, fmt.Errorf("candidate %d: %w", item.index, err)
			}
			credentials = append(credentials, descriptor)
		}
	}
	normalized, err := registry.Normalize()
	if err != nil {
		return EndpointAssemblerResult{}, err
	}
	credentials, err = normalizeCredentialDescriptors(credentials)
	if err != nil {
		return EndpointAssemblerResult{}, err
	}
	return EndpointAssemblerResult{Registry: normalized, ResolvedEndpointIDs: resolved, CredentialDescriptors: credentials}, nil
}

func findEndpointByIdentity(registry Registry, identity DaemonIdentity) (EndpointID, Endpoint, bool, error) {
	var fingerprintMatch Endpoint
	var fingerprintFound bool
	for _, endpoint := range registry.Endpoints {
		current := endpoint.DaemonIdentity
		if current.Empty() {
			continue
		}
		if current.DeviceFingerprint == identity.DeviceFingerprint {
			if current.DeviceID != identity.DeviceID {
				return "", Endpoint{}, false, connectionError(ErrorIdentityConflict, "fingerprint %q is pinned to device_id %q, not %q", identity.DeviceFingerprint, current.DeviceID, identity.DeviceID)
			}
			fingerprintMatch, fingerprintFound = endpoint, true
		}
		if current.DeviceID == identity.DeviceID && current.DeviceFingerprint != identity.DeviceFingerprint {
			return "", Endpoint{}, false, connectionError(ErrorIdentityConflict, "device_id %q is pinned to a different fingerprint", identity.DeviceID)
		}
	}
	if fingerprintFound {
		return fingerprintMatch.ID, cloneEndpoint(fingerprintMatch), true, nil
	}
	return "", Endpoint{}, false, nil
}

func applyConfirmedIdentityBindings(registry *Registry, bindings []ConfirmedIdentityBinding, candidateIdentities map[string]struct{}) error {
	ordered := append([]ConfirmedIdentityBinding(nil), bindings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].EndpointID != ordered[j].EndpointID {
			return ordered[i].EndpointID < ordered[j].EndpointID
		}
		return identityKey(ordered[i].Identity) < identityKey(ordered[j].Identity)
	})
	seenEndpoints := make(map[EndpointID]struct{}, len(ordered))
	seenIdentities := make(map[string]struct{}, len(ordered))
	for _, binding := range ordered {
		if err := validateIdentifier("confirmed identity binding endpoint", string(binding.EndpointID)); err != nil {
			return err
		}
		if err := binding.Identity.Validate(true); err != nil {
			return fmt.Errorf("confirmed identity binding for endpoint %q: %w", binding.EndpointID, err)
		}
		key := identityKey(binding.Identity)
		if _, verified := candidateIdentities[key]; !verified {
			return connectionError(ErrorConfig, "confirmed identity binding for endpoint %q has no matching verified candidate", binding.EndpointID)
		}
		if _, duplicate := seenEndpoints[binding.EndpointID]; duplicate {
			return connectionError(ErrorIdentityConflict, "endpoint %q has multiple confirmed identity bindings", binding.EndpointID)
		}
		if _, duplicate := seenIdentities[key]; duplicate {
			return connectionError(ErrorIdentityConflict, "daemon identity is confirmed for multiple endpoints")
		}
		seenEndpoints[binding.EndpointID] = struct{}{}
		seenIdentities[key] = struct{}{}

		endpoint, exists := registry.Endpoints[binding.EndpointID]
		if !exists {
			return connectionError(ErrorConfig, "confirmed identity binding endpoint %q does not exist", binding.EndpointID)
		}
		if !endpoint.DaemonIdentity.Empty() {
			if endpoint.DaemonIdentity != binding.Identity {
				return connectionError(ErrorIdentityConflict, "endpoint %q is already pinned to a different daemon identity", binding.EndpointID)
			}
			continue
		}
		existingID, _, found, err := findEndpointByIdentity(*registry, binding.Identity)
		if err != nil {
			return err
		}
		if found && existingID != binding.EndpointID {
			return connectionError(ErrorIdentityConflict, "daemon identity is already pinned to endpoint %q", existingID)
		}
		endpoint.DaemonIdentity = binding.Identity
		registry.Endpoints[binding.EndpointID] = endpoint
	}
	if len(ordered) == 0 {
		return nil
	}
	normalized, err := registry.Normalize()
	if err != nil {
		return err
	}
	*registry = normalized
	return nil
}

func identityKey(identity DaemonIdentity) string {
	return identity.DeviceID + "\x00" + identity.DeviceFingerprint
}

func mergeRoute(existing, incoming AccessRoute, applyPolicy, forcePolicy bool) AccessRoute {
	merged := cloneRoute(existing)
	if sourceRank(incoming.Source) >= sourceRank(existing.Source) {
		enabled, manualOnly, priority, policySource := existing.Enabled, existing.ManualOnly, clonePriority(existing.Priority), existing.PolicySource
		merged = cloneRoute(incoming)
		merged.Enabled, merged.ManualOnly, merged.Priority = enabled, manualOnly, priority
		merged.PolicySource = policySource
	}
	if applyPolicy && (forcePolicy || sourceRank(incoming.PolicySource) >= sourceRank(existing.PolicySource)) {
		merged.Enabled, merged.ManualOnly, merged.Priority = incoming.Enabled, incoming.ManualOnly, clonePriority(incoming.Priority)
		merged.PolicySource = incoming.PolicySource
	}
	return merged
}

func hasPrioritizedAutomaticRoute(routes map[RouteID]AccessRoute) bool {
	for _, route := range routes {
		if route.Enabled && !route.ManualOnly && route.Priority != nil {
			return true
		}
	}
	return false
}

func deriveEndpointID(fingerprint string, endpoints map[EndpointID]Endpoint) EndpointID {
	digest := sha256.Sum256([]byte(strings.TrimSpace(fingerprint)))
	encoded := hex.EncodeToString(digest[:])
	for length := 12; length <= len(encoded); length += 4 {
		candidate := EndpointID("daemon-" + encoded[:length])
		if _, exists := endpoints[candidate]; !exists {
			return candidate
		}
	}
	return EndpointID("daemon-" + encoded)
}

func sourceRank(source EndpointSource) int {
	switch source {
	case SourceLAN:
		return 0
	case SourceCloud:
		return 10
	case SourceBootstrap:
		return 20
	case SourceLocal:
		return 25
	case SourceManual:
		return 30
	case SourceShare:
		return 40
	case SourceUser:
		return 50
	default:
		return -1
	}
}

func candidateSortKey(candidate EndpointCandidate) string {
	candidate.SuggestedLabel = strings.TrimSpace(candidate.SuggestedLabel)
	sort.SliceStable(candidate.Routes, func(i, j int) bool {
		if candidate.Routes[i].ID != candidate.Routes[j].ID {
			return candidate.Routes[i].ID < candidate.Routes[j].ID
		}
		return candidate.Routes[i].Kind < candidate.Routes[j].Kind
	})
	sort.SliceStable(candidate.CredentialDescriptors, func(i, j int) bool {
		if candidate.CredentialDescriptors[i].DescriptorID != candidate.CredentialDescriptors[j].DescriptorID {
			return candidate.CredentialDescriptors[i].DescriptorID < candidate.CredentialDescriptors[j].DescriptorID
		}
		return candidate.CredentialDescriptors[i].Kind < candidate.CredentialDescriptors[j].Kind
	})
	payload, _ := json.Marshal(candidate)
	return string(payload)
}

func validateCandidateClientPolicy(candidate EndpointCandidate) error {
	if !validEndpointPlatform(candidate.Platform) {
		return connectionError(ErrorConfig, "candidate platform is invalid")
	}
	if candidate.ApplyClientPolicy && candidate.Source != SourceShare {
		return connectionError(ErrorConfig, "only a confirmed share candidate may apply imported client policy")
	}
	ownsPolicy := candidateOwnsClientPolicy(candidate)
	if candidate.ConnectMode != "" {
		switch candidate.ConnectMode {
		case ConnectAuto, ConnectOnDemand, ConnectManual:
		default:
			return connectionError(ErrorConfig, "candidate has unknown connect_mode %q", candidate.ConnectMode)
		}
		if !ownsPolicy {
			return connectionError(ErrorConfig, "candidate source %q cannot change connect_mode", candidate.Source)
		}
	}
	if candidate.SelectionPolicy != nil {
		if !ownsPolicy {
			return connectionError(ErrorConfig, "candidate source %q cannot change selection policy", candidate.Source)
		}
		if !candidate.SelectionPolicy.HedgeDelayConfigured && candidate.SelectionPolicy.HedgeDelay != 0 {
			return connectionError(ErrorConfig, "candidate hedge_delay must be zero when it is not configured")
		}
		if candidate.SelectionPolicy.HedgeDelayConfigured && (candidate.SelectionPolicy.HedgeDelay < 0 || candidate.SelectionPolicy.HedgeDelay > 30*time.Second || candidate.SelectionPolicy.HedgeDelay%time.Millisecond != 0) {
			return connectionError(ErrorConfig, "candidate hedge_delay must be a whole millisecond between 0 and 30s")
		}
		switch candidate.SelectionPolicy.RoutePreference {
		case "", RoutePreferenceAuto, RoutePreferenceDirect, RoutePreferenceSSH, RoutePreferenceManagedCloud:
		default:
			return connectionError(ErrorConfig, "candidate has unknown route_preference %q", candidate.SelectionPolicy.RoutePreference)
		}
	}
	for _, route := range candidate.Routes {
		if ownsPolicy {
			continue
		}
		if !route.Enabled || route.ManualOnly || route.Priority != nil {
			return connectionError(ErrorConfig, "candidate source %q cannot import route selection policy", candidate.Source)
		}
	}
	return nil
}

func candidateOwnsClientPolicy(candidate EndpointCandidate) bool {
	switch candidate.Source {
	case SourceLocal, SourceManual, SourceUser:
		return true
	case SourceShare:
		return candidate.ApplyClientPolicy
	default:
		return false
	}
}

func validateCredentialDescriptor(descriptor CredentialDescriptor) error {
	if err := validateIdentifier("credential descriptor", descriptor.DescriptorID); err != nil {
		return connectionError(ErrorConfig, "credential descriptor requires a single-line id")
	}
	switch descriptor.Kind {
	case CredentialSSHPrivateKey, CredentialSSHPassword:
		return nil
	case CredentialSSHAgent, CredentialCapabilityGrant, CredentialCloudProfile:
		if descriptor.Exportable {
			return connectionError(ErrorConfig, "credential descriptor %q cannot mark %q exportable", descriptor.DescriptorID, descriptor.Kind)
		}
		return nil
	default:
		return connectionError(ErrorConfig, "credential descriptor %q has unknown kind %q", descriptor.DescriptorID, descriptor.Kind)
	}
}

func normalizeCredentialDescriptors(values []CredentialDescriptor) ([]CredentialDescriptor, error) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].DescriptorID != values[j].DescriptorID {
			return values[i].DescriptorID < values[j].DescriptorID
		}
		return values[i].Kind < values[j].Kind
	})
	out := make([]CredentialDescriptor, 0, len(values))
	seen := map[string]CredentialDescriptor{}
	for _, value := range values {
		if existing, ok := seen[value.DescriptorID]; ok {
			if existing.Kind != value.Kind || existing.Exportable != value.Exportable {
				return nil, connectionError(ErrorConfig, "credential descriptor %q is defined inconsistently", value.DescriptorID)
			}
			continue
		}
		seen[value.DescriptorID] = value
		out = append(out, value)
	}
	return out, nil
}

func cloneCandidate(candidate EndpointCandidate) EndpointCandidate {
	candidate.Routes = append([]AccessRoute(nil), candidate.Routes...)
	for index := range candidate.Routes {
		candidate.Routes[index] = cloneRoute(candidate.Routes[index])
	}
	if candidate.SelectionPolicy != nil {
		selection := *candidate.SelectionPolicy
		candidate.SelectionPolicy = &selection
	}
	candidate.CredentialDescriptors = append([]CredentialDescriptor(nil), candidate.CredentialDescriptors...)
	return candidate
}

func cloneRegistry(registry Registry) Registry {
	out := Registry{Version: registry.Version, Default: registry.Default, Endpoints: make(map[EndpointID]Endpoint, len(registry.Endpoints))}
	for id, endpoint := range registry.Endpoints {
		out.Endpoints[id] = cloneEndpoint(endpoint)
	}
	return out
}
