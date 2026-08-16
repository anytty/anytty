// Package connection owns the shared client-side Endpoint/Route registry model.
package endpoint

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/anytty/anytty/shared/userdirs"
)

const (
	// RegistryVersion 是破坏性多 Route registry 的唯一受支持版本。
	// v1 把一个 endpoint 与一个 transport 绑定，不能兼容读取，否则会重新制造两套连接真值。
	RegistryVersion = 3
	// RouteConfigVersion 是 generated EndpointRouteConfigV1 的唯一当前版本。
	RouteConfigVersion uint32 = 1
	// EndpointConfigVersion 是 generated EndpointConfigV1 的唯一当前版本。
	EndpointConfigVersion uint32 = 1
	// EndpointRegistryContractVersion 是 generated EndpointRegistryV1 的唯一当前版本。
	EndpointRegistryContractVersion uint32 = 1
	// MaxRegistryBytes 限制普通配置文件大小，防止错误文件或不受信输入耗尽客户端内存。
	MaxRegistryBytes = 1 << 20
	// DefaultFileName 是 CLI/TUI 共享连接注册表的默认文件名。
	// v2 使用独立文件，避免旧 v1 connections schema 阻断默认本地入口；旧文件不迁移、不覆盖。
	DefaultFileName = "endpoints.yaml"
	// DefaultEndpointID 是缺省本地 daemon endpoint 的稳定客户端引用。
	DefaultEndpointID EndpointID = "local"
	// DefaultLocalRouteID 是缺省本地 unix route 的稳定 route 引用。
	DefaultLocalRouteID RouteID = "local"
)

const (
	// RouteLocalUnix 表示当前客户端通过本机 unix socket 到达 daemon。
	RouteLocalUnix RouteKind = "local-unix"
	// RouteSSHWebRTCTCP 表示通过 Go SSH direct-tcpip tunnel 到达 daemon 的 WebRTC ICE-TCP listener。
	RouteSSHWebRTCTCP RouteKind = "ssh-webrtc-tcp"
	// RouteDirectWebRTCTCP 表示通过 daemon embedded signaling 与 ICE-TCP 建立 WebRTC DataChannel。
	RouteDirectWebRTCTCP RouteKind = "direct-webrtc-tcp"
	// RouteManagedWebRTC 表示通过 Cloud Companion/Hub 建立 managed WebRTC DataChannel。
	RouteManagedWebRTC RouteKind = "managed-webrtc"
)

const (
	// ConnectAuto 表示客户端启动时可以主动连接 Endpoint。
	ConnectAuto ConnectMode = "auto"
	// ConnectOnDemand 表示只在用户或可见 restore 需要 Endpoint 时连接。
	ConnectOnDemand ConnectMode = "on_demand"
	// ConnectManual 表示只有显式 connect action 才能连接 Endpoint。
	ConnectManual ConnectMode = "manual"
)

const (
	// RelayAuto 表示 managed route 使用标准 direct/single Relay 策略。
	RelayAuto RelayMode = "auto"
	// RelayDirect 表示 managed route 只允许 WebRTC direct candidate。
	RelayDirect RelayMode = "direct"
	// RelayOnly 表示 managed route 只允许 single Relay，用于诊断或受限网络。
	RelayOnly RelayMode = "relay_only"
	// RelaySmart 表示显式使用私有 SmartRoute 计划。
	RelaySmart RelayMode = "smart_route"
)

const (
	// PathDirect 是 managed WebRTC 内部最终使用的端到端 direct path。
	PathDirect Path = "direct"
	// PathSingleRelay 是 managed WebRTC 内部最终使用的单 Relay path。
	PathSingleRelay Path = "single_relay"
)

const (
	// SourceLocal 表示客户端自身创建的本机 unix route。
	SourceLocal EndpointSource = "local"
	// SourceCloud 表示 Cloud directory 提供的 managed route projection。
	SourceCloud EndpointSource = "cloud"
	// SourceBootstrap 表示 daemon-signed EndpointBootstrapBundle 提供的配置。
	SourceBootstrap EndpointSource = "bootstrap"
	// SourceManual 表示用户手工验证并提交的 route 配置。
	SourceManual EndpointSource = "manual"
	// SourceShare 表示用户确认的 ClientEndpointShareBundle 导入。
	SourceShare EndpointSource = "share"
	// SourceLAN 表示只存在于内存 TTL cache 的 LAN discovery candidate。
	SourceLAN EndpointSource = "lan"
	// SourceUser 表示用户明确修改的 label 或 selection policy。
	SourceUser EndpointSource = "user"
)

const (
	// CredentialSSHAgent 表示 route 依赖目标平台上的 SSH agent。
	CredentialSSHAgent CredentialKind = "ssh-agent"
	// CredentialSSHPrivateKey 表示 route 依赖目标平台 secure store 中的 SSH private key。
	CredentialSSHPrivateKey CredentialKind = "ssh-private-key"
	// CredentialSSHPassword 表示 route 依赖目标平台 secure store 中的 SSH password。
	CredentialSSHPassword CredentialKind = "ssh-password"
	// CredentialCapabilityGrant 表示 direct/managed route 依赖 daemon capability credential。
	CredentialCapabilityGrant CredentialKind = "capability-grant"
	// CredentialCloudProfile 表示 managed route 依赖本机 Cloud account profile。
	CredentialCloudProfile CredentialKind = "cloud-profile"
)

const (
	// ErrorConfig 表示 registry 或 route 配置本身不合法。
	ErrorConfig ErrorCode = "config_invalid"
	// ErrorSizeLimit 表示输入超过公开 contract 的大小上限。
	ErrorSizeLimit ErrorCode = "size_limit"
	// ErrorUnsupportedVersion 表示输入 schema 版本不受支持。
	ErrorUnsupportedVersion ErrorCode = "unsupported_version"
	// ErrorIdentityConflict 表示 DeviceID 与 DeviceFingerprint 不能安全归并。
	ErrorIdentityConflict ErrorCode = "identity_conflict"
	// ErrorRouteConflict 表示相同 RouteID 被用于不同 RouteKind。
	ErrorRouteConflict ErrorCode = "route_conflict"
	// ErrorRouteSelectionRequired 表示 CONN003 planner 接入前不能隐式选择多条 eligible route。
	ErrorRouteSelectionRequired ErrorCode = "route_selection_required"
	// ErrorRouteUnavailable 表示没有满足当前请求的启用 route。
	ErrorRouteUnavailable ErrorCode = "route_unavailable"
	// ErrorCredentialRequired 表示 route 配置存在但目标平台缺少本地 credential。
	ErrorCredentialRequired ErrorCode = "credential_required"
	// ErrorAuthorizationRequired 表示 endpoint 已发现但尚无 daemon-local capability。
	ErrorAuthorizationRequired ErrorCode = "authorization_required"
)

// EndpointID 是客户端本地稳定引用，用于 registry、TerminalRef 和 UI 路由。
// 它不是安全身份；不同来源能否合并只能由 DaemonIdentity 的 DeviceFingerprint 决定。
type EndpointID string

// RouteID 是一个 Endpoint 内的稳定 route 配置引用。
// RouteID 只在所属 Endpoint 内唯一，不能代替 daemon identity 或 runtime AttemptID。
type RouteID string

// RouteKind 描述到达 Endpoint 的持久配置类型。
// 它不是一次运行时 Transport，也不能承载 managed WebRTC 的 direct/single_relay Path。
type RouteKind string

// ConnectMode 描述 Endpoint 的客户端连接时机策略。
// 它只影响未来连接，不改变已建立 session、terminal lifecycle 或 history truth。
type ConnectMode string

// RoutePreference 描述用户对 Endpoint planner 的顶层 Route 约束。
// 它只影响新 generation 的计划，不修改 route 本身，也不允许强制模式回退到其它 kind。
type RoutePreference string

// RelayMode 描述 managed WebRTC route 内部允许的 ICE/Relay 策略。
// 外层 Endpoint route 竞速不由该值决定。
type RelayMode string

// RelayTransport 描述 managed WebRTC Route 允许使用的 TURN transport。
// 它不代表当前系统网络；实际 transport 只能由 selected ICE candidate pair 证明。
type RelayTransport string

// Path 只描述 managed WebRTC transport 内部实际经过的网络路径。
// local Unix、Direct/SSH WebRTC TCP route 不得伪装成 managed PathDirect。
type Path string

// EndpointSource 记录 route/label 的来源，用于确定性合并和审计。
// 来源不是 endpoint kind，也不能单独触发 identity 合并或换 pin。
type EndpointSource string

// CredentialKind 描述 route secure-store 引用或可迁移 credential descriptor 所需的凭据类别。
// registry 只保存 descriptor/ref，永远不保存 credential body。
type CredentialKind string

// ErrorCode 是 Go/Kotlin/TypeScript 必须一致解释的稳定连接错误分类。
// detail 只能用于本地诊断，不能包含 secret 或把 identity conflict 降级为普通网络错误。
type ErrorCode string

// AttemptID 标识一次 route attempt；它与持久 EndpointID/RouteID 分离。
type AttemptID string

// SessionGeneration 标识 Endpoint 当前 session 世代。
// 旧世代的 live/history/input/file 结果必须由 session owner 拒绝。
type SessionGeneration uint64

// SessionLifecycle 是 EndpointSession 的客户端运行时阶段；它不表示 daemon terminal lifecycle。
type SessionLifecycle string

const (
	// SessionReady 表示该 endpoint session 已完成 identity、authorization 和 protocol Hello。
	SessionReady SessionLifecycle = "ready"
	// SessionClosing 表示 session owner 已开始取消/关闭当前 protocol bundle。
	SessionClosing SessionLifecycle = "closing"
	// SessionClosed 表示当前 generation 已关闭，任何迟到回包都必须拒绝。
	SessionClosed SessionLifecycle = "closed"
)

const (
	// RoutePreferenceAuto 允许所有合格 Route 按 priority/full race 参与计划。
	RoutePreferenceAuto RoutePreference = "auto"
	// RoutePreferenceDirect 只允许 daemon embedded signaling + ICE-TCP Route。
	RoutePreferenceDirect RoutePreference = "direct"
	// RoutePreferenceSSH 只允许 SSH tunnel Route。
	RoutePreferenceSSH RoutePreference = "ssh"
	// RoutePreferenceManagedCloud 只允许 AnyTTY Cloud managed Route。
	RoutePreferenceManagedCloud RoutePreference = "managed_cloud"
)

const (
	// RelayTransportAuto 保留 Relay lease 明确提供的 UDP/TCP transport。
	RelayTransportAuto RelayTransport = "auto"
	// RelayTransportUDP 只允许 TURN/UDP。
	RelayTransportUDP RelayTransport = "udp"
	// RelayTransportTCP 只允许 TURN/TCP。
	RelayTransportTCP RelayTransport = "tcp"
)

// Error 是统一连接领域的稳定失败。
// Code 供 CLI/TUI/App 投影恢复动作，Message 必须保持脱敏且不得用于分支解析。
type Error struct {
	Code    ErrorCode
	Message string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

// IsCode 判断错误链顶层是否为指定统一连接错误码。
func IsCode(err error, code ErrorCode) bool {
	var value *Error
	return errors.As(err, &value) && value.Code == code
}

func connectionError(code ErrorCode, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// DaemonIdentity 是跨来源、跨 route 合并同一 daemon 的安全锚点。
// DeviceID 只用于目录/路由；DeviceFingerprint 是长期 public key 的规范化 pin，二者必须成对出现。
type DaemonIdentity struct {
	DeviceID          string `json:"device_id"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

// Empty 表示当前 Endpoint 尚未经过可验证 daemon identity handshake。
// CONN001 只允许既有 local/SSH 单 route 处于该状态；自动跨来源合并永远要求完整 identity。
func (identity DaemonIdentity) Empty() bool {
	return identity.DeviceID == "" && identity.DeviceFingerprint == ""
}

// Validate 校验 DeviceID 与 fingerprint 必须同时存在且不含空白。
// required=true 用于 assembler、Direct WebRTC TCP 和 managed route，禁止用裸 DeviceID 或展示字段建立信任。
func (identity DaemonIdentity) Validate(required bool) error {
	deviceID := strings.TrimSpace(identity.DeviceID)
	fingerprint := strings.TrimSpace(identity.DeviceFingerprint)
	if deviceID == "" && fingerprint == "" && !required {
		if identity.Empty() {
			return nil
		}
		return connectionError(ErrorConfig, "empty daemon identity fields must use exact empty strings")
	}
	if deviceID == "" || fingerprint == "" {
		return connectionError(ErrorConfig, "daemon identity requires both device_id and device_fingerprint")
	}
	invalidIdentityRune := func(value rune) bool { return unicode.IsSpace(value) || unicode.IsControl(value) }
	if identity.DeviceID != deviceID || identity.DeviceFingerprint != fingerprint || strings.IndexFunc(deviceID, invalidIdentityRune) >= 0 || strings.IndexFunc(fingerprint, invalidIdentityRune) >= 0 {
		return connectionError(ErrorConfig, "daemon identity fields must not contain whitespace or control characters")
	}
	return nil
}

// SelectionPolicy 是客户端本地 route 选择策略。
// 未配置 priority 时 HedgeDelay 不改变 full-race；存在 priority 时 CONN003 planner 使用它启动下一分组。
type SelectionPolicy struct {
	HedgeDelay           time.Duration   `json:"hedge_delay_ns"`
	HedgeDelayConfigured bool            `json:"hedge_delay_configured"`
	RoutePreference      RoutePreference `json:"route_preference"`
}

// CredentialDescriptor 是可迁移但不包含 secret 的目标端凭据说明。
// DescriptorID 只在一次 assembler/share 事务内标识待解析凭据；它不是源平台 credential ref，
// 也不能携带 SSH password/private key、Cloud token、CapabilityGrant 或其他 credential body。
type CredentialDescriptor struct {
	DescriptorID string         `json:"descriptor_id"`
	Kind         CredentialKind `json:"kind"`
	Exportable   bool           `json:"exportable"`
}

// AccessRoute 是到达一个 Endpoint 的持久配置。
// Kind-specific 字段由 Validate 严格互斥；Enabled/ManualOnly/Priority 属于客户端 selection policy，不能由 discovery 静默覆盖。
type AccessRoute struct {
	ID               RouteID        `json:"route_id"`
	DisplayName      string         `json:"display_name,omitempty"`
	Kind             RouteKind      `json:"kind"`
	Enabled          bool           `json:"enabled"`
	ManualOnly       bool           `json:"manual_only"`
	Priority         *int           `json:"priority,omitempty"`
	CredentialRef    string         `json:"credential_ref,omitempty"`
	SSHCredentialRef string         `json:"ssh_credential_ref,omitempty"`
	Source           EndpointSource `json:"source"`
	PolicySource     EndpointSource `json:"policy_source"`

	Socket string `json:"socket,omitempty"`

	Host                   string                `json:"host,omitempty"`
	Port                   uint16                `json:"port,omitempty"`
	User                   string                `json:"user,omitempty"`
	ProxyJump              string                `json:"proxy_jump,omitempty"`
	HostKeyFingerprints    []string              `json:"host_key_fingerprints,omitempty"`
	CredentialDescriptor   *CredentialDescriptor `json:"credential_descriptor,omitempty"`
	RemoteSignalingAddress string                `json:"remote_signaling_address,omitempty"`
	RemoteICETCPAddress    string                `json:"remote_ice_tcp_address,omitempty"`

	SignalingAddresses  []string `json:"signaling_addresses,omitempty"`
	ICETCPAddresses     []string `json:"ice_tcp_addresses,omitempty"`
	AdvertisedAddresses []string `json:"advertised_addresses,omitempty"`
	ServerName          string   `json:"server_name,omitempty"`
	// NetworkHandle is transient platform routing metadata for a discovered LAN
	// route. It is never persisted or accepted from a pairing/share payload.
	NetworkHandle uint64 `json:"-"`

	TargetDeviceID    string         `json:"target_device_id,omitempty"`
	AccountProfileRef string         `json:"account_profile_ref,omitempty"`
	RelayMode         RelayMode      `json:"relay_mode,omitempty"`
	RelayTransport    RelayTransport `json:"relay_transport,omitempty"`
}

// Endpoint 表示当前客户端要访问的一个逻辑 daemon。
// Routes 只是到达方式；terminal lifecycle/history/file truth 始终由该 daemon 的 core-v2 持有。
type Endpoint struct {
	ID              EndpointID              `json:"endpoint_id"`
	Label           string                  `json:"label"`
	LabelSource     EndpointSource          `json:"label_source"`
	DaemonIdentity  DaemonIdentity          `json:"daemon_identity"`
	Platform        string                  `json:"platform,omitempty"`
	ConnectMode     ConnectMode             `json:"connect_mode"`
	Enabled         bool                    `json:"enabled"`
	SelectionPolicy SelectionPolicy         `json:"selection_policy"`
	Routes          map[RouteID]AccessRoute `json:"routes"`
}

// Registry 是 CLI/TUI 共享的 Endpoint 期望状态真值。
// 已建立的 Transport/ReadyPeerSession、attempt phase、Path 和错误不进入该文件。
type Registry struct {
	Version   int                     `json:"version"`
	Default   EndpointID              `json:"default"`
	Endpoints map[EndpointID]Endpoint `json:"endpoints"`
}

// ConnectIntent 描述本次连接要求的用户动作和最小 capability。
// planner 只用它过滤 route；它不拥有 terminal lifecycle，也不能扩大 daemon grant scope。
type ConnectIntent struct {
	Kind           string   `json:"kind"`
	TerminalID     string   `json:"terminal_id,omitempty"`
	RequiredScopes []string `json:"required_scopes,omitempty"`
}

// RouteAttempt 是 planner 交给具体 route dialer 的不可变输入。
// 它只冻结 EndpointID、期望 daemon identity 和唯一选中 route；不得再携带包含 Routes 的完整 Endpoint 副本形成第二份 route 真值。
type RouteAttempt struct {
	AttemptID        AttemptID         `json:"attempt_id"`
	EndpointID       EndpointID        `json:"endpoint_id"`
	ExpectedIdentity DaemonIdentity    `json:"expected_daemon_identity"`
	Route            AccessRoute       `json:"route"`
	Intent           ConnectIntent     `json:"intent"`
	Generation       SessionGeneration `json:"session_generation"`
}

// ReadyPeerSession 表示 transport、daemon identity、authorization 和 protocol Hello 均已成功。
// 只有该状态可以参加 winner CAS；底层 socket/SSH/WebRTC ready 均不能提前胜出。
type ReadyPeerSession struct {
	AttemptID          AttemptID         `json:"attempt_id"`
	EndpointID         EndpointID        `json:"endpoint_id"`
	RouteID            RouteID           `json:"route_id"`
	Generation         SessionGeneration `json:"session_generation"`
	VerifiedIdentity   DaemonIdentity    `json:"verified_daemon_identity"`
	AuthorizationScope []string          `json:"authorization_scope"`
	ProtocolVersion    uint32            `json:"protocol_version"`
	ObservedPath       Path              `json:"observed_path,omitempty"`
}

// EndpointSession 是客户端当前唯一活动的 protocol session 投影。
// 它只引用 ReadyPeerSession 结果，不复制 core terminal/history/file truth。
type EndpointSession struct {
	EndpointID         EndpointID        `json:"endpoint_id"`
	RouteID            RouteID           `json:"route_id"`
	Generation         SessionGeneration `json:"session_generation"`
	Identity           DaemonIdentity    `json:"verified_daemon_identity"`
	AuthorizationScope []string          `json:"authorization_scope"`
	ProtocolVersion    uint32            `json:"protocol_version"`
	Path               Path              `json:"observed_path,omitempty"`
	Lifecycle          SessionLifecycle  `json:"lifecycle"`
}

// LocalDiscoveryCandidate 是 LAN discovery 的短期内存候选。
// Address/label/DeviceID 都不是信任锚点；最终必须由 DTLS DataChannel auth 验证 pinned fingerprint。
type LocalDiscoveryCandidate struct {
	ClaimedIdentity    DaemonIdentity `json:"claimed_identity"`
	Address            string         `json:"address"`
	Port               uint16         `json:"port"`
	ProtocolVersion    uint32         `json:"protocol_version"`
	AnnouncementExpiry time.Time      `json:"announcement_expiry"`
	Signature          []byte         `json:"signature,omitempty"`
}

// Validate 校验 LAN discovery candidate 的内存 TTL 与结构边界。
// announcement 即使带签名也只是地址 side proof；最终连接仍必须通过 DTLS DataChannel auth 验证 Endpoint pin。
func (candidate LocalDiscoveryCandidate) Validate(now time.Time) error {
	if err := candidate.ClaimedIdentity.Validate(true); err != nil {
		return fmt.Errorf("local discovery identity: %w", err)
	}
	if strings.TrimSpace(candidate.Address) == "" || strings.ContainsAny(candidate.Address, "\r\n") || candidate.Port == 0 || candidate.ProtocolVersion == 0 {
		return connectionError(ErrorConfig, "local discovery candidate address or protocol is invalid")
	}
	if !candidate.AnnouncementExpiry.After(now) {
		return connectionError(ErrorConfig, "local discovery candidate is expired")
	}
	if len(candidate.Signature) != 0 && len(candidate.Signature) != ed25519.SignatureSize {
		return connectionError(ErrorConfig, "local discovery candidate signature length is invalid")
	}
	return nil
}

// DialIdentity 是单条 route 的运行时连接身份。
// priority/enabled/manual-only/source 不进入该值；它们只影响未来 selection，不要求热切换当前 session。
type DialIdentity struct {
	Kind                   RouteKind
	CredentialRef          string
	SSHCredentialRef       string
	Socket                 string
	Host                   string
	Port                   uint16
	User                   string
	ProxyJump              string
	CredentialDescriptor   string
	RemoteSignalingAddress string
	RemoteICETCPAddress    string
	HostKeyFingerprints    string
	SignalingAddresses     string
	ICETCPAddresses        string
	AdvertisedAddresses    string
	ServerName             string
	TargetDeviceID         string
	AccountProfileRef      string
	RelayMode              RelayMode
}

// DefaultPath 返回 connection registry 默认读取路径。
// 该路径归 CLI/TUI 共享 Endpoint registry 所有，不属于 TUI-only 配置。
func DefaultPath() string {
	return filepath.Join(userdirs.ConfigHome(), "anytty", DefaultFileName)
}

// DefaultRegistry 返回缺少配置时的单 Endpoint/单 local-unix route registry。
// 它保持当前本机用户链路可用，但不把 local route 当成 Endpoint 安全身份。
func DefaultRegistry() Registry {
	label := "local"
	if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
		label = strings.TrimSpace(hostname)
	}
	endpoint := Endpoint{
		ID: DefaultEndpointID, Label: label, LabelSource: SourceLocal, ConnectMode: ConnectAuto, Enabled: true,
		Routes: map[RouteID]AccessRoute{
			DefaultLocalRouteID: {ID: DefaultLocalRouteID, Kind: RouteLocalUnix, Enabled: true, Source: SourceLocal, PolicySource: SourceLocal, Socket: "auto"},
		},
	}
	return Registry{Version: RegistryVersion, Default: DefaultEndpointID, Endpoints: map[EndpointID]Endpoint{DefaultEndpointID: endpoint}}
}

// Load 读取 v2 Endpoint registry。
// 默认路径缺失返回 DefaultRegistry；显式路径缺失或 v1 schema 均直接失败，不做兼容 fallback。
func Load(path string) (Registry, error) {
	explicit := strings.TrimSpace(path) != ""
	if !explicit {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return DefaultRegistry(), nil
		}
		return Registry{}, err
	}
	registry, err := Parse(data)
	if err != nil {
		return Registry{}, fmt.Errorf("parse endpoint registry %q: %w", path, err)
	}
	return registry, nil
}

// Parse 严格解析 v2 endpoints.yaml。
// 未知字段、重复文档、旧 `connections` schema、非法 route 组合和超过 1 MiB 的输入全部 fail closed。
func Parse(data []byte) (Registry, error) {
	return parseRegistry(data)
}

// Normalize 返回深拷贝且稳定排序语义的 registry。
// 它补齐 Endpoint 内部安全默认值并校验 identity、route kind、priority 完整性和 default endpoint。
// 空 registry 保持为空；只有默认配置路径不存在时，Load 才创建本机 local Endpoint。
func (registry Registry) Normalize() (Registry, error) {
	if registry.Version == 0 {
		registry.Version = RegistryVersion
	}
	if registry.Version != RegistryVersion {
		return Registry{}, connectionError(ErrorUnsupportedVersion, "unsupported endpoint registry version %d", registry.Version)
	}
	if len(registry.Endpoints) == 0 {
		if normalizeEndpointID(registry.Default) != "" {
			return Registry{}, connectionError(ErrorConfig, "empty endpoint registry cannot define default endpoint %q", registry.Default)
		}
		return Registry{Version: RegistryVersion, Endpoints: map[EndpointID]Endpoint{}}, nil
	}
	normalized := Registry{Version: RegistryVersion, Default: normalizeEndpointID(registry.Default), Endpoints: make(map[EndpointID]Endpoint, len(registry.Endpoints))}
	endpointKeys := make([]string, 0, len(registry.Endpoints))
	for key := range registry.Endpoints {
		endpointKeys = append(endpointKeys, string(key))
	}
	sort.Strings(endpointKeys)
	deviceIDs := make(map[string]EndpointID, len(registry.Endpoints))
	fingerprints := make(map[string]EndpointID, len(registry.Endpoints))
	for _, keyValue := range endpointKeys {
		key := EndpointID(keyValue)
		endpoint := registry.Endpoints[key]
		if err := validateIdentifier("endpoint", string(key)); err != nil {
			return Registry{}, err
		}
		key = normalizeEndpointID(key)
		if endpoint.ID == "" {
			endpoint.ID = key
		} else if err := validateIdentifier("endpoint", string(endpoint.ID)); err != nil {
			return Registry{}, err
		}
		endpoint.ID = normalizeEndpointID(endpoint.ID)
		if key != endpoint.ID {
			return Registry{}, connectionError(ErrorConfig, "endpoint key %q does not match id %q", key, endpoint.ID)
		}
		seenRoutes := make(map[RouteID]struct{}, len(endpoint.Routes))
		for routeKey, route := range endpoint.Routes {
			if err := validateIdentifier("route", string(routeKey)); err != nil {
				return Registry{}, fmt.Errorf("endpoint %q: %w", endpoint.ID, err)
			}
			routeID := route.ID
			if routeID == "" {
				routeID = routeKey
			} else if err := validateIdentifier("route", string(routeID)); err != nil {
				return Registry{}, fmt.Errorf("endpoint %q: %w", endpoint.ID, err)
			}
			routeID = normalizeRouteID(routeID)
			if normalizeRouteID(routeKey) != routeID {
				return Registry{}, connectionError(ErrorConfig, "endpoint %q route key %q does not match id %q", endpoint.ID, routeKey, routeID)
			}
			if _, duplicate := seenRoutes[routeID]; duplicate {
				return Registry{}, connectionError(ErrorConfig, "endpoint %q repeats route id %q", endpoint.ID, routeID)
			}
			seenRoutes[routeID] = struct{}{}
		}
		endpoint = endpoint.withDefaults()
		if err := endpoint.Validate(); err != nil {
			return Registry{}, err
		}
		if _, duplicate := normalized.Endpoints[endpoint.ID]; duplicate {
			return Registry{}, connectionError(ErrorConfig, "endpoint registry repeats endpoint id %q", endpoint.ID)
		}
		if !endpoint.DaemonIdentity.Empty() {
			if existingID, duplicate := deviceIDs[endpoint.DaemonIdentity.DeviceID]; duplicate {
				existing := normalized.Endpoints[existingID].DaemonIdentity
				if existing.DeviceFingerprint == endpoint.DaemonIdentity.DeviceFingerprint {
					return Registry{}, connectionError(ErrorIdentityConflict, "endpoints %q and %q repeat daemon identity", existingID, endpoint.ID)
				}
				return Registry{}, connectionError(ErrorIdentityConflict, "device_id %q is pinned to multiple fingerprints", endpoint.DaemonIdentity.DeviceID)
			}
			if existingID, duplicate := fingerprints[endpoint.DaemonIdentity.DeviceFingerprint]; duplicate {
				existing := normalized.Endpoints[existingID].DaemonIdentity
				return Registry{}, connectionError(ErrorIdentityConflict, "fingerprint %q is pinned to device_id values %q and %q", endpoint.DaemonIdentity.DeviceFingerprint, existing.DeviceID, endpoint.DaemonIdentity.DeviceID)
			}
			deviceIDs[endpoint.DaemonIdentity.DeviceID] = endpoint.ID
			fingerprints[endpoint.DaemonIdentity.DeviceFingerprint] = endpoint.ID
		}
		normalized.Endpoints[endpoint.ID] = endpoint
	}
	if normalized.Default == "" {
		return Registry{}, connectionError(ErrorConfig, "non-empty endpoint registry requires an explicit default endpoint")
	}
	endpoint, ok := normalized.Endpoints[normalized.Default]
	if !ok {
		return Registry{}, connectionError(ErrorConfig, "default endpoint %q not found", normalized.Default)
	}
	if !endpoint.Enabled {
		return Registry{}, connectionError(ErrorConfig, "default endpoint %q is disabled", normalized.Default)
	}
	return normalized, nil
}

// List 返回按 EndpointID 稳定排序的 Endpoint 深拷贝。
// CLI/TUI 必须使用稳定顺序，避免 map 顺序造成配置输出和 picker 抖动。
func (registry Registry) List() []Endpoint {
	ids := make([]string, 0, len(registry.Endpoints))
	for id := range registry.Endpoints {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	out := make([]Endpoint, 0, len(ids))
	for _, value := range ids {
		out = append(out, cloneEndpoint(registry.Endpoints[EndpointID(value)]))
	}
	return out
}

// DefaultEndpoint 返回当前 default Endpoint。
// false 表示 registry 尚未规范化或 default 指向缺失条目。
func (registry Registry) DefaultEndpoint() (Endpoint, bool) {
	endpoint, ok := registry.Endpoints[registry.Default]
	return cloneEndpoint(endpoint), ok
}

// Validate 校验 Endpoint identity、route 集合和 selection policy。
// 本方法不做网络 IO；host-key、TLS certificate 和 daemon proof 只能在具体安全握手中验证。
func (endpoint Endpoint) Validate() error {
	if err := validateIdentifier("endpoint", string(endpoint.ID)); err != nil {
		return err
	}
	switch endpoint.ConnectMode {
	case ConnectAuto, ConnectOnDemand, ConnectManual:
	default:
		return connectionError(ErrorConfig, "endpoint %q has unknown connect_mode %q", endpoint.ID, endpoint.ConnectMode)
	}
	if !endpoint.SelectionPolicy.HedgeDelayConfigured && endpoint.SelectionPolicy.HedgeDelay != 0 {
		return connectionError(ErrorConfig, "endpoint %q hedge_delay must be zero when it is not configured", endpoint.ID)
	}
	if endpoint.SelectionPolicy.HedgeDelayConfigured && (endpoint.SelectionPolicy.HedgeDelay < 0 || endpoint.SelectionPolicy.HedgeDelay > 30*time.Second || endpoint.SelectionPolicy.HedgeDelay%time.Millisecond != 0) {
		return connectionError(ErrorConfig, "endpoint %q hedge_delay must be a whole millisecond between 0 and 30s", endpoint.ID)
	}
	switch endpoint.SelectionPolicy.RoutePreference {
	case "", RoutePreferenceAuto, RoutePreferenceDirect, RoutePreferenceSSH, RoutePreferenceManagedCloud:
	default:
		return connectionError(ErrorConfig, "endpoint %q has unknown route_preference %q", endpoint.ID, endpoint.SelectionPolicy.RoutePreference)
	}
	if !validSource(endpoint.LabelSource) {
		return connectionError(ErrorConfig, "endpoint %q has unknown label source %q", endpoint.ID, endpoint.LabelSource)
	}
	if err := endpoint.DaemonIdentity.Validate(false); err != nil {
		return fmt.Errorf("endpoint %q: %w", endpoint.ID, err)
	}
	if !validEndpointPlatform(endpoint.Platform) {
		return connectionError(ErrorConfig, "endpoint %q platform is invalid", endpoint.ID)
	}
	if len(endpoint.Routes) == 0 {
		return connectionError(ErrorConfig, "endpoint %q requires at least one route", endpoint.ID)
	}
	anyPriority := false
	allPriority := true
	for key, route := range endpoint.Routes {
		key = normalizeRouteID(key)
		if route.ID == "" {
			route.ID = key
		}
		route.ID = normalizeRouteID(route.ID)
		if key != "" && key != route.ID {
			return connectionError(ErrorConfig, "endpoint %q route key %q does not match id %q", endpoint.ID, key, route.ID)
		}
		if err := route.Validate(endpoint.DaemonIdentity); err != nil {
			return fmt.Errorf("endpoint %q: %w", endpoint.ID, err)
		}
		if route.Enabled && !route.ManualOnly {
			anyPriority = anyPriority || route.Priority != nil
			allPriority = allPriority && route.Priority != nil
		}
	}
	if anyPriority && !allPriority {
		return connectionError(ErrorConfig, "endpoint %q must configure priority on every enabled automatic route", endpoint.ID)
	}
	return nil
}

// Validate 校验 route kind-specific 字段互斥、引用格式和 identity 前置条件。
// credential body 不进入该结构；缺少 credential ref 可作为 authorization_required/credential_required 投影保存。
func (route AccessRoute) Validate(identity DaemonIdentity) error {
	if err := validateIdentifier("route", string(route.ID)); err != nil {
		return err
	}
	if strings.TrimSpace(route.DisplayName) != route.DisplayName || len(route.DisplayName) > 128 {
		return connectionError(ErrorConfig, "route %q display_name is invalid", route.ID)
	}
	if route.Priority != nil && (*route.Priority < 0 || int64(*route.Priority) > int64(1<<31-1)) {
		return connectionError(ErrorConfig, "route %q priority must be between 0 and 2147483647", route.ID)
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "credential_ref", value: route.CredentialRef},
		{name: "ssh_credential_ref", value: route.SSHCredentialRef},
		{name: "socket", value: route.Socket},
		{name: "host", value: route.Host},
		{name: "user", value: route.User},
		{name: "proxy_jump", value: route.ProxyJump},
		{name: "remote_signaling_address", value: route.RemoteSignalingAddress},
		{name: "remote_ice_tcp_address", value: route.RemoteICETCPAddress},
		{name: "server_name", value: route.ServerName},
		{name: "target_device_id", value: route.TargetDeviceID},
		{name: "account_profile_ref", value: route.AccountProfileRef},
	} {
		if err := validateCanonicalRouteText(route.ID, field.name, field.value, false); err != nil {
			return err
		}
	}
	if err := validateCanonicalRouteList(route.ID, "host_key_fingerprints", route.HostKeyFingerprints, false); err != nil {
		return err
	}
	if err := validateCanonicalRouteList(route.ID, "signaling_addresses", route.SignalingAddresses, route.Kind == RouteDirectWebRTCTCP); err != nil {
		return err
	}
	if err := validateCanonicalRouteList(route.ID, "ice_tcp_addresses", route.ICETCPAddresses, route.Kind == RouteDirectWebRTCTCP); err != nil {
		return err
	}
	if err := validateCanonicalRouteList(route.ID, "advertised_addresses", route.AdvertisedAddresses, false); err != nil {
		return err
	}
	if route.CredentialDescriptor != nil {
		if route.Kind != RouteSSHWebRTCTCP {
			return connectionError(ErrorConfig, "route %q contains credential_descriptor outside ssh-webrtc-tcp", route.ID)
		}
		if err := validateCredentialDescriptor(*route.CredentialDescriptor); err != nil {
			return fmt.Errorf("route %q credential_descriptor: %w", route.ID, err)
		}
	}
	if !validSource(route.Source) || !validSource(route.PolicySource) {
		return connectionError(ErrorConfig, "route %q has an unknown source", route.ID)
	}
	switch route.Kind {
	case RouteLocalUnix:
		if route.Socket == "" {
			return connectionError(ErrorConfig, "local-unix route %q requires socket", route.ID)
		}
		if route.hasSSHFields() || route.hasDirectFields() || route.hasManagedFields() || route.CredentialRef != "" {
			return connectionError(ErrorConfig, "local-unix route %q contains fields owned by another route kind", route.ID)
		}
	case RouteSSHWebRTCTCP:
		if route.Host == "" || route.RemoteSignalingAddress == "" || route.RemoteICETCPAddress == "" {
			return connectionError(ErrorConfig, "ssh-webrtc-tcp route %q requires host and remote signaling/ICE-TCP addresses", route.ID)
		}
		if route.Socket != "" || route.hasDirectFields() || route.hasManagedFields() {
			return connectionError(ErrorConfig, "ssh-webrtc-tcp route %q contains fields owned by another route kind", route.ID)
		}
	case RouteDirectWebRTCTCP:
		if len(route.SignalingAddresses) == 0 || len(route.ICETCPAddresses) == 0 {
			return connectionError(ErrorConfig, "direct-webrtc-tcp route %q requires signaling and ICE-TCP addresses", route.ID)
		}
		if err := identity.Validate(true); err != nil {
			return fmt.Errorf("direct-webrtc-tcp route %q: %w", route.ID, err)
		}
		if route.Socket != "" || route.hasSSHFields() || route.hasManagedFields() {
			return connectionError(ErrorConfig, "direct-webrtc-tcp route %q contains fields owned by another route kind", route.ID)
		}
	case RouteManagedWebRTC:
		if route.TargetDeviceID == "" {
			return connectionError(ErrorConfig, "managed-webrtc route %q requires target_device_id", route.ID)
		}
		if err := identity.Validate(true); err != nil {
			return fmt.Errorf("managed-webrtc route %q: %w", route.ID, err)
		}
		if route.TargetDeviceID != identity.DeviceID {
			return connectionError(ErrorConfig, "managed-webrtc route %q target_device_id does not match endpoint identity", route.ID)
		}
		switch route.RelayMode {
		case RelayAuto, RelayDirect, RelayOnly, RelaySmart:
		default:
			return connectionError(ErrorConfig, "managed-webrtc route %q has unknown relay_mode %q", route.ID, route.RelayMode)
		}
		switch route.RelayTransport {
		case "", RelayTransportAuto, RelayTransportUDP, RelayTransportTCP:
		default:
			return connectionError(ErrorConfig, "managed-webrtc route %q has unknown relay_transport %q", route.ID, route.RelayTransport)
		}
		if route.Socket != "" || route.hasSSHFields() || route.hasDirectFields() {
			return connectionError(ErrorConfig, "managed-webrtc route %q contains fields owned by another route kind", route.ID)
		}
	default:
		return connectionError(ErrorConfig, "route %q has unknown kind %q", route.ID, route.Kind)
	}
	return nil
}

// RouteList 返回按 RouteID 稳定排序的 route 深拷贝。
func (endpoint Endpoint) RouteList() []AccessRoute {
	ids := make([]string, 0, len(endpoint.Routes))
	for id := range endpoint.Routes {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	out := make([]AccessRoute, 0, len(ids))
	for _, value := range ids {
		out = append(out, cloneRoute(endpoint.Routes[RouteID(value)]))
	}
	return out
}

// Route 返回指定 route 的深拷贝。
func (endpoint Endpoint) Route(id RouteID) (AccessRoute, bool) {
	route, ok := endpoint.Routes[normalizeRouteID(id)]
	return cloneRoute(route), ok
}

// DialIdentity 返回 route 的连接身份，用于 registry reload 后判断当前 session 是否需要重连。
func (route AccessRoute) DialIdentity() DialIdentity {
	hostKeys := append([]string(nil), route.HostKeyFingerprints...)
	signalingAddresses := append([]string(nil), route.SignalingAddresses...)
	iceTCPAddresses := append([]string(nil), route.ICETCPAddresses...)
	advertisedAddresses := append([]string(nil), route.AdvertisedAddresses...)
	sort.Strings(hostKeys)
	sort.Strings(signalingAddresses)
	sort.Strings(iceTCPAddresses)
	sort.Strings(advertisedAddresses)
	descriptor := ""
	if route.CredentialDescriptor != nil {
		descriptor = route.CredentialDescriptor.DescriptorID + "\x00" + string(route.CredentialDescriptor.Kind)
	}
	return DialIdentity{
		Kind: route.Kind, CredentialRef: strings.TrimSpace(route.CredentialRef), SSHCredentialRef: strings.TrimSpace(route.SSHCredentialRef), Socket: strings.TrimSpace(route.Socket),
		Host: strings.TrimSpace(route.Host), Port: route.Port, User: strings.TrimSpace(route.User), ProxyJump: strings.TrimSpace(route.ProxyJump),
		CredentialDescriptor: descriptor, RemoteSignalingAddress: strings.TrimSpace(route.RemoteSignalingAddress),
		RemoteICETCPAddress: strings.TrimSpace(route.RemoteICETCPAddress), HostKeyFingerprints: strings.Join(hostKeys, "\x00"),
		SignalingAddresses: strings.Join(signalingAddresses, "\x00"), ICETCPAddresses: strings.Join(iceTCPAddresses, "\x00"),
		AdvertisedAddresses: strings.Join(advertisedAddresses, "\x00"), ServerName: strings.TrimSpace(route.ServerName),
		TargetDeviceID: strings.TrimSpace(route.TargetDeviceID), AccountProfileRef: strings.TrimSpace(route.AccountProfileRef), RelayMode: route.RelayMode,
	}
}

// RequiresReconnect 判断 Endpoint identity 或任一 route dial identity 是否变化。
// label、selection priority、route enabled/manual-only 和 connect mode 只影响展示/未来连接，不热切换当前 session；
// Endpoint enabled 由持有当前 session 的进程单独执行 disable/drain lifecycle，不属于 reconnect 判定。
func (endpoint Endpoint) RequiresReconnect(next Endpoint) bool {
	if endpoint.DaemonIdentity != next.DaemonIdentity || len(endpoint.Routes) != len(next.Routes) {
		return true
	}
	for id, route := range endpoint.Routes {
		nextRoute, ok := next.Routes[id]
		if !ok || route.DialIdentity() != nextRoute.DialIdentity() {
			return true
		}
	}
	return false
}

// DisplayChanged 判断用户可见 label 是否变化。
func (endpoint Endpoint) DisplayChanged(next Endpoint) bool {
	return strings.TrimSpace(endpoint.Label) != strings.TrimSpace(next.Label)
}

// NewLocalEndpoint 构造单 local-unix route Endpoint。
// 该构造器用于 CLI/TUI/default harness，不会伪造 DaemonIdentity。
func NewLocalEndpoint(id EndpointID, label, socket string, mode ConnectMode) Endpoint {
	if id == "" {
		id = DefaultEndpointID
	}
	routeID := RouteID("local")
	return Endpoint{ID: id, Label: label, LabelSource: SourceManual, ConnectMode: mode, Enabled: true, Routes: map[RouteID]AccessRoute{
		routeID: {ID: routeID, Kind: RouteLocalUnix, Enabled: true, Source: SourceManual, PolicySource: SourceManual, Socket: socket},
	}}
}

// NewSSHEndpoint 构造单 ssh-webrtc-tcp route Endpoint。
// host 可以是 OpenSSH alias；credential body 和 known_hosts 内容不进入 registry。
func NewSSHEndpoint(id EndpointID, label, host, sshCredentialRef, remoteSignalingAddress, remoteICETCPAddress string, mode ConnectMode) Endpoint {
	routeID := RouteID("ssh")
	return Endpoint{ID: id, Label: label, LabelSource: SourceManual, ConnectMode: mode, Enabled: true, Routes: map[RouteID]AccessRoute{
		routeID: {ID: routeID, Kind: RouteSSHWebRTCTCP, Enabled: true, Source: SourceManual, PolicySource: SourceManual, Host: host, SSHCredentialRef: sshCredentialRef, RemoteSignalingAddress: remoteSignalingAddress, RemoteICETCPAddress: remoteICETCPAddress},
	}}
}

// NewManagedEndpoint 构造单 managed-webrtc route Endpoint。
// identity pin 属于 Endpoint；TargetDeviceID 只属于 Cloud route，二者不能互相替代。
func NewManagedEndpoint(id EndpointID, label string, identity DaemonIdentity, targetDeviceID, credentialRef string, relayMode RelayMode, mode ConnectMode) Endpoint {
	routeID := RouteID("cloud")
	return Endpoint{ID: id, Label: label, LabelSource: SourceCloud, DaemonIdentity: identity, ConnectMode: mode, Enabled: true, Routes: map[RouteID]AccessRoute{
		routeID: {ID: routeID, Kind: RouteManagedWebRTC, Enabled: true, Source: SourceCloud, PolicySource: SourceLocal, TargetDeviceID: targetDeviceID, CredentialRef: credentialRef, RelayMode: relayMode},
	}}
}

func (endpoint Endpoint) withDefaults() Endpoint {
	endpoint.ID = normalizeEndpointID(endpoint.ID)
	endpoint.Platform = strings.ToLower(strings.TrimSpace(endpoint.Platform))
	if strings.TrimSpace(endpoint.Label) == "" {
		endpoint.Label = string(endpoint.ID)
	}
	if endpoint.LabelSource == "" {
		endpoint.LabelSource = SourceManual
	}
	if endpoint.ConnectMode == "" {
		if endpoint.ID == DefaultEndpointID {
			endpoint.ConnectMode = ConnectAuto
		} else {
			endpoint.ConnectMode = ConnectOnDemand
		}
	}
	routes := make(map[RouteID]AccessRoute, len(endpoint.Routes))
	for key, route := range endpoint.Routes {
		key = normalizeRouteID(key)
		if route.ID == "" {
			route.ID = key
		}
		route = route.withDefaults()
		routes[route.ID] = route
	}
	endpoint.Routes = routes
	return endpoint
}

func validEndpointPlatform(value string) bool {
	if value == "" {
		return true
	}
	if value != strings.ToLower(strings.TrimSpace(value)) || len(value) > 32 {
		return false
	}
	for _, current := range value {
		if (current < 'a' || current > 'z') && (current < '0' || current > '9') && current != '_' && current != '-' {
			return false
		}
	}
	return true
}

func (route AccessRoute) withDefaults() AccessRoute {
	route.ID = normalizeRouteID(route.ID)
	if route.Source == "" {
		route.Source = SourceManual
	}
	if route.PolicySource == "" {
		route.PolicySource = route.Source
	}
	switch route.Kind {
	case RouteLocalUnix:
		if route.Socket == "" {
			route.Socket = "auto"
		}
	case RouteSSHWebRTCTCP:
		if route.Port == 0 {
			route.Port = 22
		}
	case RouteManagedWebRTC:
		if route.RelayMode == "" {
			route.RelayMode = RelayAuto
		}
		if route.RelayTransport == "" {
			route.RelayTransport = RelayTransportAuto
		}
	}
	route.HostKeyFingerprints = normalizeStrings(route.HostKeyFingerprints)
	route.SignalingAddresses = normalizeStrings(route.SignalingAddresses)
	route.ICETCPAddresses = normalizeStrings(route.ICETCPAddresses)
	route.AdvertisedAddresses = normalizeStrings(route.AdvertisedAddresses)
	return route
}

func (route AccessRoute) hasSSHFields() bool {
	return strings.TrimSpace(route.Host) != "" || route.Port != 0 || strings.TrimSpace(route.User) != "" ||
		strings.TrimSpace(route.ProxyJump) != "" || strings.TrimSpace(route.RemoteSignalingAddress) != "" ||
		strings.TrimSpace(route.RemoteICETCPAddress) != "" || strings.TrimSpace(route.SSHCredentialRef) != "" ||
		len(route.HostKeyFingerprints) != 0 || route.CredentialDescriptor != nil
}

func (route AccessRoute) hasDirectFields() bool {
	return len(route.SignalingAddresses) != 0 || len(route.ICETCPAddresses) != 0 || len(route.AdvertisedAddresses) != 0 || strings.TrimSpace(route.ServerName) != ""
}

func (route AccessRoute) hasManagedFields() bool {
	return strings.TrimSpace(route.TargetDeviceID) != "" || strings.TrimSpace(route.AccountProfileRef) != "" || route.RelayMode != ""
}

func validSource(source EndpointSource) bool {
	switch source {
	case SourceLocal, SourceCloud, SourceBootstrap, SourceManual, SourceShare, SourceLAN, SourceUser:
		return true
	default:
		return false
	}
}

func normalizeEndpointID(id EndpointID) EndpointID {
	return EndpointID(strings.TrimSpace(string(id)))
}

func normalizeRouteID(id RouteID) RouteID {
	return RouteID(strings.TrimSpace(string(id)))
}

func validateIdentifier(kind, value string) error {
	if value == "" {
		return connectionError(ErrorConfig, "%s id is required", kind)
	}
	if len(value) > 128 {
		return connectionError(ErrorConfig, "%s id is longer than 128 characters", kind)
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return connectionError(ErrorConfig, "%s id %q contains invalid character %q", kind, value, r)
		}
	}
	return nil
}

func normalizeStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func validateCanonicalRouteText(routeID RouteID, field, value string, required bool) error {
	if value == "" {
		if required {
			return connectionError(ErrorConfig, "route %q field %s is required", routeID, field)
		}
		return nil
	}
	if value != strings.TrimSpace(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return connectionError(ErrorConfig, "route %q field %s is not canonical", routeID, field)
	}
	return nil
}

func validateCanonicalRouteList(routeID RouteID, field string, values []string, required bool) error {
	if required && len(values) == 0 {
		return connectionError(ErrorConfig, "route %q field %s requires at least one value", routeID, field)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateCanonicalRouteText(routeID, field, value, true); err != nil {
			return err
		}
		if _, duplicate := seen[value]; duplicate {
			return connectionError(ErrorConfig, "route %q field %s repeats value %q", routeID, field, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func cloneEndpoint(endpoint Endpoint) Endpoint {
	endpoint.Routes = cloneRoutes(endpoint.Routes)
	return endpoint
}

func cloneRoutes(routes map[RouteID]AccessRoute) map[RouteID]AccessRoute {
	if routes == nil {
		return nil
	}
	out := make(map[RouteID]AccessRoute, len(routes))
	for id, route := range routes {
		out[id] = cloneRoute(route)
	}
	return out
}

func cloneRoute(route AccessRoute) AccessRoute {
	if route.Priority != nil {
		priority := *route.Priority
		route.Priority = &priority
	}
	route.HostKeyFingerprints = append([]string(nil), route.HostKeyFingerprints...)
	route.SignalingAddresses = append([]string(nil), route.SignalingAddresses...)
	route.ICETCPAddresses = append([]string(nil), route.ICETCPAddresses...)
	route.AdvertisedAddresses = append([]string(nil), route.AdvertisedAddresses...)
	if route.CredentialDescriptor != nil {
		descriptor := *route.CredentialDescriptor
		route.CredentialDescriptor = &descriptor
	}
	return route
}
