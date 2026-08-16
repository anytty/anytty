package endpoint

import (
	"fmt"
	"sort"
	"time"

	"github.com/anytty/anytty/proto/remoteauthpb"
)

// EndpointFromProto 把跨 binding/进程传入的 generated EndpointConfigV1 转为 Go Client Engine 领域模型。
// Proto 是字段与版本真值；该入口拒绝 unknown field、旧版本、重复 RouteID、非法 identity 和不完整的 kind-specific 配置。
func EndpointFromProto(config *remoteauthpb.EndpointConfigV1) (Endpoint, error) {
	if config == nil || config.GetSchemaVersion() != EndpointConfigVersion || hasUnknownFields(config.ProtoReflect()) {
		return Endpoint{}, connectionError(ErrorUnsupportedVersion, "endpoint proto version is unsupported")
	}
	identity := DaemonIdentity{}
	if wireIdentity := config.GetIdentity(); wireIdentity != nil {
		if err := validateWireIdentity(wireIdentity, false); err != nil {
			return Endpoint{}, fmt.Errorf("endpoint proto identity: %w", err)
		}
		identity = DaemonIdentity{DeviceID: wireIdentity.GetDeviceId(), DeviceFingerprint: wireIdentity.GetDeviceFingerprint()}
	}
	connectMode := mapWireConnectMode(config.GetConnectMode())
	if connectMode == "" {
		return Endpoint{}, connectionError(ErrorConfig, "endpoint proto connect_mode is invalid")
	}
	selection := SelectionPolicy{}
	if policy := config.GetSelectionPolicy(); policy != nil {
		if policy.GetHedgeDelayMillis() > 30_000 || (!policy.GetHedgeDelayConfigured() && policy.GetHedgeDelayMillis() != 0) {
			return Endpoint{}, connectionError(ErrorConfig, "endpoint proto selection policy is invalid")
		}
		selection = SelectionPolicy{
			HedgeDelayConfigured: policy.GetHedgeDelayConfigured(), HedgeDelay: time.Duration(policy.GetHedgeDelayMillis()) * time.Millisecond,
			RoutePreference: mapWireRoutePreference(policy.GetRoutePreference()),
		}
		if selection.RoutePreference == "" {
			return Endpoint{}, connectionError(ErrorConfig, "endpoint proto route_preference is invalid")
		}
	}
	model := Endpoint{
		ID: EndpointID(config.GetEndpointId()), Label: config.GetLabel(), LabelSource: mapWireSource(config.GetLabelSource()),
		DaemonIdentity: identity, Platform: config.GetPlatform(), ConnectMode: connectMode, Enabled: config.GetEnabled(), SelectionPolicy: selection,
		Routes: make(map[RouteID]AccessRoute, len(config.GetRoutes())),
	}
	for _, wireRoute := range config.GetRoutes() {
		if wireRoute == nil || wireRoute.GetSchemaVersion() != RouteConfigVersion {
			return Endpoint{}, connectionError(ErrorUnsupportedVersion, "endpoint route proto version is unsupported")
		}
		route, err := accessRouteFromWire(wireRoute, wireRoute.GetEnabled())
		if err != nil {
			return Endpoint{}, err
		}
		if wireRoute.GetSource() == remoteauthpb.EndpointSource_ENDPOINT_SOURCE_UNSPECIFIED {
			route.Source = SourceManual
		}
		if wireRoute.GetPolicySource() == remoteauthpb.EndpointSource_ENDPOINT_SOURCE_UNSPECIFIED {
			route.PolicySource = route.Source
		}
		if _, duplicate := model.Routes[route.ID]; duplicate {
			return Endpoint{}, connectionError(ErrorRouteConflict, "endpoint proto repeats route_id %q", route.ID)
		}
		model.Routes[route.ID] = route
	}
	return model.withDefaults().normalizedAndValidated()
}

// EndpointToProto 把规范化 Go Endpoint 投影为 generated EndpointConfigV1。
// 返回值只包含持久配置，不包含 runtime winner、session、credential body、terminal 或 UI state。
func EndpointToProto(endpoint Endpoint) (*remoteauthpb.EndpointConfigV1, error) {
	normalized, err := endpoint.normalizedAndValidated()
	if err != nil {
		return nil, err
	}
	config := &remoteauthpb.EndpointConfigV1{
		SchemaVersion: EndpointConfigVersion,
		EndpointId:    string(normalized.ID), Label: normalized.Label, LabelSource: wireSource(normalized.LabelSource),
		ConnectMode: wireConnectMode(normalized.ConnectMode), Enabled: normalized.Enabled,
		Platform: normalized.Platform,
		SelectionPolicy: &remoteauthpb.EndpointSelectionPolicy{
			HedgeDelayConfigured: normalized.SelectionPolicy.HedgeDelayConfigured,
			HedgeDelayMillis:     uint64(normalized.SelectionPolicy.HedgeDelay / time.Millisecond),
			RoutePreference:      wireRoutePreference(normalized.SelectionPolicy.RoutePreference),
		},
	}
	if !normalized.DaemonIdentity.Empty() {
		config.Identity = &remoteauthpb.EndpointDaemonIdentity{
			DeviceId: normalized.DaemonIdentity.DeviceID, DeviceFingerprint: normalized.DaemonIdentity.DeviceFingerprint,
		}
	}
	for _, route := range normalized.RouteList() {
		wireRoute, err := routeToProto(route)
		if err != nil {
			return nil, err
		}
		config.Routes = append(config.Routes, wireRoute)
	}
	return config, nil
}

// RegistryFromProto 把 generated EndpointRegistryV1 转为 Go Client Engine registry snapshot。
// EndpointID 重复、default 不存在、identity 冲突和任一 route 配置失败都会使整个事务失败，不发布部分结果。
func RegistryFromProto(config *remoteauthpb.EndpointRegistryV1) (Registry, error) {
	if config == nil || config.GetSchemaVersion() != EndpointRegistryContractVersion || hasUnknownFields(config.ProtoReflect()) {
		return Registry{}, connectionError(ErrorUnsupportedVersion, "endpoint registry proto version is unsupported")
	}
	registry := Registry{Version: RegistryVersion, Default: EndpointID(config.GetDefaultEndpointId()), Endpoints: make(map[EndpointID]Endpoint, len(config.GetEndpoints()))}
	for _, wireEndpoint := range config.GetEndpoints() {
		endpoint, err := EndpointFromProto(wireEndpoint)
		if err != nil {
			return Registry{}, err
		}
		if _, duplicate := registry.Endpoints[endpoint.ID]; duplicate {
			return Registry{}, connectionError(ErrorConfig, "endpoint registry proto repeats endpoint_id %q", endpoint.ID)
		}
		registry.Endpoints[endpoint.ID] = endpoint
	}
	return registry.Normalize()
}

// RegistryToProto 把完整 Go registry 稳定排序后投影为 generated EndpointRegistryV1。
// 该入口用于 JNI/C ABI、未来 WASM 和配置导入导出，不生成 JSON 镜像 DTO。
func RegistryToProto(registry Registry) (*remoteauthpb.EndpointRegistryV1, error) {
	normalized, err := registry.Normalize()
	if err != nil {
		return nil, err
	}
	config := &remoteauthpb.EndpointRegistryV1{SchemaVersion: EndpointRegistryContractVersion, DefaultEndpointId: string(normalized.Default)}
	ids := make([]string, 0, len(normalized.Endpoints))
	for id := range normalized.Endpoints {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, id := range ids {
		wireEndpoint, err := EndpointToProto(normalized.Endpoints[EndpointID(id)])
		if err != nil {
			return nil, err
		}
		config.Endpoints = append(config.Endpoints, wireEndpoint)
	}
	return config, nil
}

func (endpoint Endpoint) normalizedAndValidated() (Endpoint, error) {
	endpoint = endpoint.withDefaults()
	if err := endpoint.Validate(); err != nil {
		return Endpoint{}, err
	}
	return endpoint, nil
}

func routeToProto(route AccessRoute) (*remoteauthpb.EndpointRouteConfigV1, error) {
	config := &remoteauthpb.EndpointRouteConfigV1{
		SchemaVersion: RouteConfigVersion, RouteId: string(route.ID), Enabled: route.Enabled, ManualOnly: route.ManualOnly,
		CredentialRef: route.CredentialRef, Source: wireSource(route.Source), PolicySource: wireSource(route.PolicySource), DisplayName: route.DisplayName,
	}
	if route.Priority != nil {
		priority := int32(*route.Priority)
		config.Priority = &priority
	}
	switch route.Kind {
	case RouteLocalUnix:
		config.Route = &remoteauthpb.EndpointRouteConfigV1_LocalUnix{LocalUnix: &remoteauthpb.LocalUnixRouteConfig{Socket: route.Socket}}
	case RouteDirectWebRTCTCP:
		config.Route = &remoteauthpb.EndpointRouteConfigV1_DirectWebrtcTcp{DirectWebrtcTcp: &remoteauthpb.DirectWebRTCTCPRouteConfig{
			SignalingAddresses: append([]string(nil), route.SignalingAddresses...), IceTcpAddresses: append([]string(nil), route.ICETCPAddresses...),
			AdvertisedAddresses: append([]string(nil), route.AdvertisedAddresses...), ServerName: route.ServerName,
		}}
	case RouteSSHWebRTCTCP:
		sshConfig := &remoteauthpb.SSHWebRTCTCPRouteConfig{
			Host: route.Host, Port: uint32(route.Port), User: route.User, ProxyJump: route.ProxyJump,
			HostKeyFingerprints: append([]string(nil), route.HostKeyFingerprints...), RemoteSignalingAddress: route.RemoteSignalingAddress,
			RemoteIceTcpAddress: route.RemoteICETCPAddress, SshCredentialRef: route.SSHCredentialRef,
		}
		if descriptor := route.CredentialDescriptor; descriptor != nil {
			sshConfig.CredentialDescriptor = &remoteauthpb.EndpointCredentialDescriptor{
				DescriptorId: descriptor.DescriptorID, Kind: wireCredentialKind(descriptor.Kind), Exportable: descriptor.Exportable,
			}
		}
		config.Route = &remoteauthpb.EndpointRouteConfigV1_SshWebrtcTcp{SshWebrtcTcp: sshConfig}
	case RouteManagedWebRTC:
		config.Route = &remoteauthpb.EndpointRouteConfigV1_ManagedWebrtc{ManagedWebrtc: &remoteauthpb.ManagedWebRTCRouteConfig{
			TargetDeviceId: route.TargetDeviceID, AccountProfileRef: route.AccountProfileRef, RelayMode: wireRelayMode(route.RelayMode),
			RelayTransport: wireRelayTransport(route.RelayTransport),
		}}
	default:
		return nil, connectionError(ErrorConfig, "route %q has unknown kind %q", route.ID, route.Kind)
	}
	return config, nil
}

func mapWireConnectMode(mode remoteauthpb.EndpointConnectMode) ConnectMode {
	switch mode {
	case remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_AUTO:
		return ConnectAuto
	case remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_ON_DEMAND:
		return ConnectOnDemand
	case remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_MANUAL:
		return ConnectManual
	default:
		return ""
	}
}

func wireConnectMode(mode ConnectMode) remoteauthpb.EndpointConnectMode {
	switch mode {
	case ConnectAuto:
		return remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_AUTO
	case ConnectOnDemand:
		return remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_ON_DEMAND
	case ConnectManual:
		return remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_MANUAL
	default:
		return remoteauthpb.EndpointConnectMode_ENDPOINT_CONNECT_MODE_UNSPECIFIED
	}
}

func wireSource(source EndpointSource) remoteauthpb.EndpointSource {
	switch source {
	case SourceLocal:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_LOCAL
	case SourceCloud:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_CLOUD
	case SourceBootstrap:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_BOOTSTRAP
	case SourceManual:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_MANUAL
	case SourceShare:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_SHARE
	case SourceLAN:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_LAN
	case SourceUser:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_USER
	default:
		return remoteauthpb.EndpointSource_ENDPOINT_SOURCE_UNSPECIFIED
	}
}

func wireRelayMode(mode RelayMode) remoteauthpb.ManagedWebRTCRelayMode {
	switch mode {
	case RelayAuto:
		return remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_AUTO
	case RelayDirect:
		return remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_DIRECT
	case RelayOnly:
		return remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY
	case RelaySmart:
		return remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_SMART_ROUTE
	default:
		return remoteauthpb.ManagedWebRTCRelayMode_MANAGED_WEBRTC_RELAY_MODE_UNSPECIFIED
	}
}

func mapWireRoutePreference(value remoteauthpb.EndpointRoutePreference) RoutePreference {
	switch value {
	case remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_UNSPECIFIED,
		remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_AUTO:
		return RoutePreferenceAuto
	case remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_DIRECT:
		return RoutePreferenceDirect
	case remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_SSH:
		return RoutePreferenceSSH
	case remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD:
		return RoutePreferenceManagedCloud
	default:
		return ""
	}
}

func wireRoutePreference(value RoutePreference) remoteauthpb.EndpointRoutePreference {
	switch value {
	case RoutePreferenceAuto:
		return remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_AUTO
	case RoutePreferenceDirect:
		return remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_DIRECT
	case RoutePreferenceSSH:
		return remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_SSH
	case RoutePreferenceManagedCloud:
		return remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD
	default:
		return remoteauthpb.EndpointRoutePreference_ENDPOINT_ROUTE_PREFERENCE_UNSPECIFIED
	}
}

func mapWireRelayTransport(value remoteauthpb.ManagedWebRTCRelayTransport) RelayTransport {
	switch value {
	case remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_UNSPECIFIED,
		remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO:
		return RelayTransportAuto
	case remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_UDP:
		return RelayTransportUDP
	case remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_TCP:
		return RelayTransportTCP
	default:
		return ""
	}
}

func wireRelayTransport(value RelayTransport) remoteauthpb.ManagedWebRTCRelayTransport {
	switch value {
	case RelayTransportAuto:
		return remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO
	case RelayTransportUDP:
		return remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_UDP
	case RelayTransportTCP:
		return remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_TCP
	default:
		return remoteauthpb.ManagedWebRTCRelayTransport_MANAGED_WEBRTC_RELAY_TRANSPORT_UNSPECIFIED
	}
}

func wireCredentialKind(kind CredentialKind) remoteauthpb.EndpointCredentialKind {
	switch kind {
	case CredentialSSHAgent:
		return remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_AGENT
	case CredentialSSHPrivateKey:
		return remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY
	case CredentialSSHPassword:
		return remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD
	case CredentialCapabilityGrant:
		return remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_CAPABILITY_GRANT
	case CredentialCloudProfile:
		return remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_CLOUD_PROFILE
	default:
		return remoteauthpb.EndpointCredentialKind_ENDPOINT_CREDENTIAL_KIND_UNSPECIFIED
	}
}
