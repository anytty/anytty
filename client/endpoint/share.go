package endpoint

import (
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

// ShareRouteDiff 描述 share bundle 相对当前 registry 的单条 Route 变化。
// 它只是用户确认投影，不参与 merge；最终结果仍必须由 AssembleEndpoints 在提交时重新计算。
type ShareRouteDiff struct {
	RouteID RouteID
	Kind    RouteKind
	Action  string
}

// ShareDiff 是接收端在原子导入前展示的 Go-owned 配置差异。
// EndpointID 是按当前 registry 解析的本地引用，daemon identity 才是跨客户端归并真值。
type ShareDiff struct {
	EndpointID             EndpointID
	Label                  string
	Identity               DaemonIdentity
	Routes                 []ShareRouteDiff
	ConnectModeChanged     bool
	SelectionPolicyChanged bool
	CredentialDescriptors  []CredentialDescriptor
}

// NewClientEndpointShareBundle 从已规范化 Endpoint 构造 config-only share bundle。
// 本地 Unix Route、credential ref、SSH secret ref、Cloud profile ref 与 CapabilityGrant 不会进入 wire contract。
func NewClientEndpointShareBundle(target Endpoint, transferID string, now time.Time, ttl time.Duration) (*ClientEndpointShareBundleV1, error) {
	if ttl <= 0 {
		return nil, connectionError(ErrorConfig, "endpoint share ttl must be positive")
	}
	normalized, err := target.normalizedAndValidated()
	if err != nil {
		return nil, err
	}
	if normalized.DaemonIdentity.Empty() {
		return nil, connectionError(ErrorConfig, "endpoint share requires a pinned daemon identity")
	}
	now = now.UTC()
	bundle := &remoteauthpb.ClientEndpointShareBundleV1{
		SchemaVersion: ClientEndpointShareBundleVersion,
		TransferId:    strings.TrimSpace(transferID),
		Identity: &remoteauthpb.EndpointDaemonIdentity{
			DeviceId: normalized.DaemonIdentity.DeviceID, DeviceFingerprint: normalized.DaemonIdentity.DeviceFingerprint,
		},
		SuggestedLabel: normalized.Label,
		ConnectMode:    wireConnectMode(normalized.ConnectMode),
		SelectionPolicy: &remoteauthpb.EndpointSelectionPolicy{
			HedgeDelayConfigured: normalized.SelectionPolicy.HedgeDelayConfigured,
			HedgeDelayMillis:     uint64(normalized.SelectionPolicy.HedgeDelay / time.Millisecond),
			RoutePreference:      wireRoutePreference(normalized.SelectionPolicy.RoutePreference),
		},
		IssuedAtUnixNano:  now.UnixNano(),
		ExpiresAtUnixNano: now.Add(ttl).UnixNano(),
	}
	descriptors := make(map[string]*remoteauthpb.EndpointCredentialDescriptor)
	for _, route := range normalized.RouteList() {
		if route.Kind == RouteLocalUnix {
			continue
		}
		wire, err := routeToProto(route)
		if err != nil {
			return nil, err
		}
		wire.CredentialRef = ""
		wire.Source = remoteauthpb.EndpointSource_ENDPOINT_SOURCE_UNSPECIFIED
		wire.PolicySource = remoteauthpb.EndpointSource_ENDPOINT_SOURCE_UNSPECIFIED
		switch route.Kind {
		case RouteDirectWebRTCTCP:
			descriptors[string(route.ID)+"-capability"] = shareCredentialDescriptor(string(route.ID)+"-capability", CredentialCapabilityGrant)
		case RouteSSHWebRTCTCP:
			wire.GetSshWebrtcTcp().SshCredentialRef = ""
			if descriptor := route.CredentialDescriptor; descriptor != nil {
				wire.GetSshWebrtcTcp().CredentialDescriptor = &remoteauthpb.EndpointCredentialDescriptor{
					DescriptorId: descriptor.DescriptorID, Kind: wireCredentialKind(descriptor.Kind), Exportable: descriptor.Exportable,
				}
				descriptors[descriptor.DescriptorID] = wire.GetSshWebrtcTcp().GetCredentialDescriptor()
			}
			descriptors[string(route.ID)+"-capability"] = shareCredentialDescriptor(string(route.ID)+"-capability", CredentialCapabilityGrant)
		case RouteManagedWebRTC:
			wire.GetManagedWebrtc().AccountProfileRef = ""
			descriptors[string(route.ID)+"-cloud-profile"] = shareCredentialDescriptor(string(route.ID)+"-cloud-profile", CredentialCloudProfile)
			descriptors[string(route.ID)+"-capability"] = shareCredentialDescriptor(string(route.ID)+"-capability", CredentialCapabilityGrant)
		}
		bundle.Routes = append(bundle.Routes, wire)
	}
	if len(bundle.Routes) == 0 {
		return nil, connectionError(ErrorConfig, "endpoint share has no portable Route")
	}
	keys := make([]string, 0, len(descriptors))
	for key := range descriptors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		bundle.CredentialDescriptors = append(bundle.CredentialDescriptors, descriptors[key])
	}
	if _, err := MarshalClientEndpointShareBundle(bundle); err != nil {
		return nil, err
	}
	return bundle, nil
}

func shareCredentialDescriptor(id string, kind CredentialKind) *remoteauthpb.EndpointCredentialDescriptor {
	return &remoteauthpb.EndpointCredentialDescriptor{DescriptorId: id, Kind: wireCredentialKind(kind), Exportable: false}
}

// EndpointCandidateFromShareBundle 把已通过 TLS share session 验证的 bundle 投影为 assembler candidate。
// config-only bundle 不授予 terminal 权限；credential descriptor 只说明接收端后续需要准备的本地能力。
func EndpointCandidateFromShareBundle(bundle *ClientEndpointShareBundleV1) (EndpointCandidate, error) {
	if err := validateClientEndpointShareBundle(bundle); err != nil {
		return EndpointCandidate{}, err
	}
	policy := SelectionPolicy{}
	if wire := bundle.GetSelectionPolicy(); wire != nil {
		policy = SelectionPolicy{HedgeDelayConfigured: wire.GetHedgeDelayConfigured(), HedgeDelay: time.Duration(wire.GetHedgeDelayMillis()) * time.Millisecond}
	}
	candidate := EndpointCandidate{
		Source:         SourceShare,
		Identity:       DaemonIdentity{DeviceID: bundle.GetIdentity().GetDeviceId(), DeviceFingerprint: bundle.GetIdentity().GetDeviceFingerprint()},
		SuggestedLabel: bundle.GetSuggestedLabel(), ConnectMode: mapWireConnectMode(bundle.GetConnectMode()), SelectionPolicy: &policy, ApplyClientPolicy: true,
	}
	for _, wire := range bundle.GetRoutes() {
		route, err := accessRouteFromWire(wire, wire.GetEnabled())
		if err != nil {
			return EndpointCandidate{}, err
		}
		route.Source, route.PolicySource = SourceShare, SourceShare
		candidate.Routes = append(candidate.Routes, route)
	}
	for _, wire := range bundle.GetCredentialDescriptors() {
		candidate.CredentialDescriptors = append(candidate.CredentialDescriptors, CredentialDescriptor{
			DescriptorID: wire.GetDescriptorId(), Kind: mapWireCredentialKind(wire.GetKind()), Exportable: wire.GetExportable(),
		})
	}
	return candidate, nil
}

// PreviewShare 通过同一 assembler 计算待确认 diff，但不修改或持久化 registry。
func PreviewShare(registry Registry, candidate EndpointCandidate) (ShareDiff, error) {
	assembled, err := AssembleEndpoints(EndpointAssemblerInput{Registry: registry, Candidates: []EndpointCandidate{candidate}})
	if err != nil {
		return ShareDiff{}, err
	}
	id := assembled.ResolvedEndpointIDs[0]
	after := assembled.Registry.Endpoints[id]
	before, existed := registry.Endpoints[id]
	diff := ShareDiff{
		EndpointID: id, Label: after.Label, Identity: after.DaemonIdentity,
		ConnectModeChanged:     !existed || before.ConnectMode != after.ConnectMode,
		SelectionPolicyChanged: !existed || before.SelectionPolicy != after.SelectionPolicy,
		CredentialDescriptors:  append([]CredentialDescriptor(nil), assembled.CredentialDescriptors...),
	}
	for _, route := range candidate.Routes {
		action := "add"
		if existing, ok := before.Routes[route.ID]; ok {
			action = "unchanged"
			if !reflect.DeepEqual(existing, after.Routes[route.ID]) {
				action = "update"
			}
		}
		diff.Routes = append(diff.Routes, ShareRouteDiff{RouteID: route.ID, Kind: route.Kind, Action: action})
	}
	sort.Slice(diff.Routes, func(i, j int) bool { return diff.Routes[i].RouteID < diff.Routes[j].RouteID })
	return diff, nil
}
