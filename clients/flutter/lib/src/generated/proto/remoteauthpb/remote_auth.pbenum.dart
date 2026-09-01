// This is a generated file - do not edit.
//
// Generated from remoteauthpb/remote_auth.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// AuthErrorCode 是端到端授权失败时允许返回给对端的稳定分类。
/// detail 必须脱敏，不能暴露 grant 是否对应某个已存在 terminal。
class AuthErrorCode extends $pb.ProtobufEnum {
  static const AuthErrorCode AUTH_ERROR_CODE_UNSPECIFIED =
      AuthErrorCode._(0, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_UNSPECIFIED');
  static const AuthErrorCode AUTH_ERROR_CODE_PROTOCOL =
      AuthErrorCode._(1, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_PROTOCOL');
  static const AuthErrorCode AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH =
      AuthErrorCode._(
          2, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH');
  static const AuthErrorCode AUTH_ERROR_CODE_CAPABILITY_INVALID =
      AuthErrorCode._(
          3, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_CAPABILITY_INVALID');
  static const AuthErrorCode AUTH_ERROR_CODE_CAPABILITY_EXPIRED =
      AuthErrorCode._(
          4, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_CAPABILITY_EXPIRED');
  static const AuthErrorCode AUTH_ERROR_CODE_CAPABILITY_REVOKED =
      AuthErrorCode._(
          5, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_CAPABILITY_REVOKED');
  static const AuthErrorCode AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID =
      AuthErrorCode._(
          6, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID');
  static const AuthErrorCode AUTH_ERROR_CODE_SCOPE_INVALID =
      AuthErrorCode._(7, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_SCOPE_INVALID');
  static const AuthErrorCode AUTH_ERROR_CODE_REPLAYED =
      AuthErrorCode._(8, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_REPLAYED');
  static const AuthErrorCode AUTH_ERROR_CODE_INTERNAL =
      AuthErrorCode._(9, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_INTERNAL');
  static const AuthErrorCode AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH =
      AuthErrorCode._(
          10, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH');
  static const AuthErrorCode AUTH_ERROR_CODE_PAIRING_TICKET_INVALID =
      AuthErrorCode._(
          11, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_PAIRING_TICKET_INVALID');
  static const AuthErrorCode AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED =
      AuthErrorCode._(
          12, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED');
  static const AuthErrorCode AUTH_ERROR_CODE_PAIRING_TICKET_CONSUMED =
      AuthErrorCode._(
          13, _omitEnumNames ? '' : 'AUTH_ERROR_CODE_PAIRING_TICKET_CONSUMED');

  static const $core.List<AuthErrorCode> values = <AuthErrorCode>[
    AUTH_ERROR_CODE_UNSPECIFIED,
    AUTH_ERROR_CODE_PROTOCOL,
    AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH,
    AUTH_ERROR_CODE_CAPABILITY_INVALID,
    AUTH_ERROR_CODE_CAPABILITY_EXPIRED,
    AUTH_ERROR_CODE_CAPABILITY_REVOKED,
    AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID,
    AUTH_ERROR_CODE_SCOPE_INVALID,
    AUTH_ERROR_CODE_REPLAYED,
    AUTH_ERROR_CODE_INTERNAL,
    AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH,
    AUTH_ERROR_CODE_PAIRING_TICKET_INVALID,
    AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED,
    AUTH_ERROR_CODE_PAIRING_TICKET_CONSUMED,
  ];

  static final $core.List<AuthErrorCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 13);
  static AuthErrorCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const AuthErrorCode._(super.value, super.name);
}

/// ScopeKind 是 capability 成功映射到 core-v2 前的规范化 scope 分类。
class ScopeKind extends $pb.ProtobufEnum {
  static const ScopeKind SCOPE_KIND_UNSPECIFIED =
      ScopeKind._(0, _omitEnumNames ? '' : 'SCOPE_KIND_UNSPECIFIED');
  static const ScopeKind SCOPE_KIND_DAEMON =
      ScopeKind._(1, _omitEnumNames ? '' : 'SCOPE_KIND_DAEMON');
  static const ScopeKind SCOPE_KIND_TERMINAL =
      ScopeKind._(2, _omitEnumNames ? '' : 'SCOPE_KIND_TERMINAL');
  static const ScopeKind SCOPE_KIND_MACHINE_EVENTS =
      ScopeKind._(3, _omitEnumNames ? '' : 'SCOPE_KIND_MACHINE_EVENTS');

  static const $core.List<ScopeKind> values = <ScopeKind>[
    SCOPE_KIND_UNSPECIFIED,
    SCOPE_KIND_DAEMON,
    SCOPE_KIND_TERMINAL,
    SCOPE_KIND_MACHINE_EVENTS,
  ];

  static final $core.List<ScopeKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static ScopeKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ScopeKind._(super.value, super.name);
}

/// ChannelBindingKind 区分实际 transport 提供的安全 channel 证明来源。
/// direct TLS 与 managed WebRTC 共用同一 canonical 结构，但不能互相替换 kind 后重放 proof。
class ChannelBindingKind extends $pb.ProtobufEnum {
  static const ChannelBindingKind CHANNEL_BINDING_KIND_UNSPECIFIED =
      ChannelBindingKind._(
          0, _omitEnumNames ? '' : 'CHANNEL_BINDING_KIND_UNSPECIFIED');
  static const ChannelBindingKind CHANNEL_BINDING_KIND_DIRECT_TLS =
      ChannelBindingKind._(
          1, _omitEnumNames ? '' : 'CHANNEL_BINDING_KIND_DIRECT_TLS');
  static const ChannelBindingKind CHANNEL_BINDING_KIND_DTLS =
      ChannelBindingKind._(
          2, _omitEnumNames ? '' : 'CHANNEL_BINDING_KIND_DTLS');
  static const ChannelBindingKind CHANNEL_BINDING_KIND_LOCAL_UNIX =
      ChannelBindingKind._(
          3, _omitEnumNames ? '' : 'CHANNEL_BINDING_KIND_LOCAL_UNIX');

  static const $core.List<ChannelBindingKind> values = <ChannelBindingKind>[
    CHANNEL_BINDING_KIND_UNSPECIFIED,
    CHANNEL_BINDING_KIND_DIRECT_TLS,
    CHANNEL_BINDING_KIND_DTLS,
    CHANNEL_BINDING_KIND_LOCAL_UNIX,
  ];

  static final $core.List<ChannelBindingKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static ChannelBindingKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ChannelBindingKind._(super.value, super.name);
}

/// AuthOpenKind 为客户端签名输入提供明确 domain separation。
class AuthOpenKind extends $pb.ProtobufEnum {
  static const AuthOpenKind AUTH_OPEN_KIND_UNSPECIFIED =
      AuthOpenKind._(0, _omitEnumNames ? '' : 'AUTH_OPEN_KIND_UNSPECIFIED');
  static const AuthOpenKind AUTH_OPEN_KIND_CAPABILITY =
      AuthOpenKind._(1, _omitEnumNames ? '' : 'AUTH_OPEN_KIND_CAPABILITY');
  static const AuthOpenKind AUTH_OPEN_KIND_PAIRING =
      AuthOpenKind._(2, _omitEnumNames ? '' : 'AUTH_OPEN_KIND_PAIRING');

  static const $core.List<AuthOpenKind> values = <AuthOpenKind>[
    AUTH_OPEN_KIND_UNSPECIFIED,
    AUTH_OPEN_KIND_CAPABILITY,
    AUTH_OPEN_KIND_PAIRING,
  ];

  static final $core.List<AuthOpenKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static AuthOpenKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const AuthOpenKind._(super.value, super.name);
}

/// DirectSignalingErrorCode 是 daemon embedded signaling 在创建 WebRTC peer 前返回的稳定失败分类。
/// 该错误只描述信令 admission，不替代 DataChannel 内的 DeviceIdentity 与 CapabilityGrant 校验。
class DirectSignalingErrorCode extends $pb.ProtobufEnum {
  static const DirectSignalingErrorCode
      DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED = DirectSignalingErrorCode._(
          0, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED');
  static const DirectSignalingErrorCode DIRECT_SIGNALING_ERROR_CODE_PROTOCOL =
      DirectSignalingErrorCode._(
          1, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_PROTOCOL');
  static const DirectSignalingErrorCode DIRECT_SIGNALING_ERROR_CODE_EXPIRED =
      DirectSignalingErrorCode._(
          2, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_EXPIRED');
  static const DirectSignalingErrorCode DIRECT_SIGNALING_ERROR_CODE_REPLAYED =
      DirectSignalingErrorCode._(
          3, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_REPLAYED');
  static const DirectSignalingErrorCode
      DIRECT_SIGNALING_ERROR_CODE_IDENTITY_MISMATCH =
      DirectSignalingErrorCode._(
          4,
          _omitEnumNames
              ? ''
              : 'DIRECT_SIGNALING_ERROR_CODE_IDENTITY_MISMATCH');
  static const DirectSignalingErrorCode DIRECT_SIGNALING_ERROR_CODE_INTERNAL =
      DirectSignalingErrorCode._(
          5, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_INTERNAL');
  static const DirectSignalingErrorCode DIRECT_SIGNALING_ERROR_CODE_OVERLOADED =
      DirectSignalingErrorCode._(
          6, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_OVERLOADED');
  static const DirectSignalingErrorCode
      DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION = DirectSignalingErrorCode._(
          7, _omitEnumNames ? '' : 'DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION');

  static const $core.List<DirectSignalingErrorCode> values =
      <DirectSignalingErrorCode>[
    DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED,
    DIRECT_SIGNALING_ERROR_CODE_PROTOCOL,
    DIRECT_SIGNALING_ERROR_CODE_EXPIRED,
    DIRECT_SIGNALING_ERROR_CODE_REPLAYED,
    DIRECT_SIGNALING_ERROR_CODE_IDENTITY_MISMATCH,
    DIRECT_SIGNALING_ERROR_CODE_INTERNAL,
    DIRECT_SIGNALING_ERROR_CODE_OVERLOADED,
    DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION,
  ];

  static final $core.List<DirectSignalingErrorCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 7);
  static DirectSignalingErrorCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const DirectSignalingErrorCode._(super.value, super.name);
}

/// EndpointConnectMode 描述客户端何时可以自动建立 Endpoint session。
class EndpointConnectMode extends $pb.ProtobufEnum {
  static const EndpointConnectMode ENDPOINT_CONNECT_MODE_UNSPECIFIED =
      EndpointConnectMode._(
          0, _omitEnumNames ? '' : 'ENDPOINT_CONNECT_MODE_UNSPECIFIED');
  static const EndpointConnectMode ENDPOINT_CONNECT_MODE_AUTO =
      EndpointConnectMode._(
          1, _omitEnumNames ? '' : 'ENDPOINT_CONNECT_MODE_AUTO');
  static const EndpointConnectMode ENDPOINT_CONNECT_MODE_ON_DEMAND =
      EndpointConnectMode._(
          2, _omitEnumNames ? '' : 'ENDPOINT_CONNECT_MODE_ON_DEMAND');
  static const EndpointConnectMode ENDPOINT_CONNECT_MODE_MANUAL =
      EndpointConnectMode._(
          3, _omitEnumNames ? '' : 'ENDPOINT_CONNECT_MODE_MANUAL');

  static const $core.List<EndpointConnectMode> values = <EndpointConnectMode>[
    ENDPOINT_CONNECT_MODE_UNSPECIFIED,
    ENDPOINT_CONNECT_MODE_AUTO,
    ENDPOINT_CONNECT_MODE_ON_DEMAND,
    ENDPOINT_CONNECT_MODE_MANUAL,
  ];

  static final $core.List<EndpointConnectMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static EndpointConnectMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointConnectMode._(super.value, super.name);
}

/// ManagedWebRTCRelayMode 只描述 AnyTTY Cloud managed route 内部允许的 ICE/Relay 策略。
class ManagedWebRTCRelayMode extends $pb.ProtobufEnum {
  static const ManagedWebRTCRelayMode MANAGED_WEBRTC_RELAY_MODE_UNSPECIFIED =
      ManagedWebRTCRelayMode._(
          0, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_MODE_UNSPECIFIED');
  static const ManagedWebRTCRelayMode MANAGED_WEBRTC_RELAY_MODE_AUTO =
      ManagedWebRTCRelayMode._(
          1, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_MODE_AUTO');
  static const ManagedWebRTCRelayMode MANAGED_WEBRTC_RELAY_MODE_DIRECT =
      ManagedWebRTCRelayMode._(
          2, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_MODE_DIRECT');
  static const ManagedWebRTCRelayMode MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY =
      ManagedWebRTCRelayMode._(
          3, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY');
  static const ManagedWebRTCRelayMode MANAGED_WEBRTC_RELAY_MODE_SMART_ROUTE =
      ManagedWebRTCRelayMode._(
          4, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_MODE_SMART_ROUTE');

  static const $core.List<ManagedWebRTCRelayMode> values =
      <ManagedWebRTCRelayMode>[
    MANAGED_WEBRTC_RELAY_MODE_UNSPECIFIED,
    MANAGED_WEBRTC_RELAY_MODE_AUTO,
    MANAGED_WEBRTC_RELAY_MODE_DIRECT,
    MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY,
    MANAGED_WEBRTC_RELAY_MODE_SMART_ROUTE,
  ];

  static final $core.List<ManagedWebRTCRelayMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static ManagedWebRTCRelayMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ManagedWebRTCRelayMode._(super.value, super.name);
}

/// EndpointRoutePreference 是用户对 Endpoint planner 的顶层 Route 约束。
/// 未指定和 AUTO 都允许 planner 按当前 route/priority 竞速；强制模式不可静默回退到其它 kind。
class EndpointRoutePreference extends $pb.ProtobufEnum {
  static const EndpointRoutePreference ENDPOINT_ROUTE_PREFERENCE_UNSPECIFIED =
      EndpointRoutePreference._(
          0, _omitEnumNames ? '' : 'ENDPOINT_ROUTE_PREFERENCE_UNSPECIFIED');
  static const EndpointRoutePreference ENDPOINT_ROUTE_PREFERENCE_AUTO =
      EndpointRoutePreference._(
          1, _omitEnumNames ? '' : 'ENDPOINT_ROUTE_PREFERENCE_AUTO');
  static const EndpointRoutePreference ENDPOINT_ROUTE_PREFERENCE_DIRECT =
      EndpointRoutePreference._(
          2, _omitEnumNames ? '' : 'ENDPOINT_ROUTE_PREFERENCE_DIRECT');
  static const EndpointRoutePreference ENDPOINT_ROUTE_PREFERENCE_SSH =
      EndpointRoutePreference._(
          3, _omitEnumNames ? '' : 'ENDPOINT_ROUTE_PREFERENCE_SSH');
  static const EndpointRoutePreference ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD =
      EndpointRoutePreference._(
          4, _omitEnumNames ? '' : 'ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD');

  static const $core.List<EndpointRoutePreference> values =
      <EndpointRoutePreference>[
    ENDPOINT_ROUTE_PREFERENCE_UNSPECIFIED,
    ENDPOINT_ROUTE_PREFERENCE_AUTO,
    ENDPOINT_ROUTE_PREFERENCE_DIRECT,
    ENDPOINT_ROUTE_PREFERENCE_SSH,
    ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD,
  ];

  static final $core.List<EndpointRoutePreference?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static EndpointRoutePreference? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointRoutePreference._(super.value, super.name);
}

/// ManagedWebRTCRelayTransport 约束 managed Route 可使用的 TURN transport。
/// 它不表达系统 Wi-Fi、移动网络或 VPN，也不推断实际选中的 candidate pair。
class ManagedWebRTCRelayTransport extends $pb.ProtobufEnum {
  static const ManagedWebRTCRelayTransport
      MANAGED_WEBRTC_RELAY_TRANSPORT_UNSPECIFIED =
      ManagedWebRTCRelayTransport._(0,
          _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_TRANSPORT_UNSPECIFIED');
  static const ManagedWebRTCRelayTransport MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO =
      ManagedWebRTCRelayTransport._(
          1, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO');
  static const ManagedWebRTCRelayTransport MANAGED_WEBRTC_RELAY_TRANSPORT_UDP =
      ManagedWebRTCRelayTransport._(
          2, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_TRANSPORT_UDP');
  static const ManagedWebRTCRelayTransport MANAGED_WEBRTC_RELAY_TRANSPORT_TCP =
      ManagedWebRTCRelayTransport._(
          3, _omitEnumNames ? '' : 'MANAGED_WEBRTC_RELAY_TRANSPORT_TCP');

  static const $core.List<ManagedWebRTCRelayTransport> values =
      <ManagedWebRTCRelayTransport>[
    MANAGED_WEBRTC_RELAY_TRANSPORT_UNSPECIFIED,
    MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO,
    MANAGED_WEBRTC_RELAY_TRANSPORT_UDP,
    MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
  ];

  static final $core.List<ManagedWebRTCRelayTransport?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static ManagedWebRTCRelayTransport? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ManagedWebRTCRelayTransport._(super.value, super.name);
}

/// EndpointSource 记录 route 或用户策略的来源，供确定性 assembler 保留更高权重的已确认信息。
class EndpointSource extends $pb.ProtobufEnum {
  static const EndpointSource ENDPOINT_SOURCE_UNSPECIFIED =
      EndpointSource._(0, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_UNSPECIFIED');
  static const EndpointSource ENDPOINT_SOURCE_LAN =
      EndpointSource._(1, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_LAN');
  static const EndpointSource ENDPOINT_SOURCE_CLOUD =
      EndpointSource._(2, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_CLOUD');
  static const EndpointSource ENDPOINT_SOURCE_BOOTSTRAP =
      EndpointSource._(3, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_BOOTSTRAP');
  static const EndpointSource ENDPOINT_SOURCE_LOCAL =
      EndpointSource._(4, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_LOCAL');
  static const EndpointSource ENDPOINT_SOURCE_MANUAL =
      EndpointSource._(5, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_MANUAL');
  static const EndpointSource ENDPOINT_SOURCE_SHARE =
      EndpointSource._(6, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_SHARE');
  static const EndpointSource ENDPOINT_SOURCE_USER =
      EndpointSource._(7, _omitEnumNames ? '' : 'ENDPOINT_SOURCE_USER');

  static const $core.List<EndpointSource> values = <EndpointSource>[
    ENDPOINT_SOURCE_UNSPECIFIED,
    ENDPOINT_SOURCE_LAN,
    ENDPOINT_SOURCE_CLOUD,
    ENDPOINT_SOURCE_BOOTSTRAP,
    ENDPOINT_SOURCE_LOCAL,
    ENDPOINT_SOURCE_MANUAL,
    ENDPOINT_SOURCE_SHARE,
    ENDPOINT_SOURCE_USER,
  ];

  static final $core.List<EndpointSource?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 7);
  static EndpointSource? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointSource._(super.value, super.name);
}

/// EndpointCredentialKind 只描述目标平台 secure store 中凭据的类别；wire contract 永远不携带凭据 body。
class EndpointCredentialKind extends $pb.ProtobufEnum {
  static const EndpointCredentialKind ENDPOINT_CREDENTIAL_KIND_UNSPECIFIED =
      EndpointCredentialKind._(
          0, _omitEnumNames ? '' : 'ENDPOINT_CREDENTIAL_KIND_UNSPECIFIED');
  static const EndpointCredentialKind ENDPOINT_CREDENTIAL_KIND_SSH_AGENT =
      EndpointCredentialKind._(
          1, _omitEnumNames ? '' : 'ENDPOINT_CREDENTIAL_KIND_SSH_AGENT');
  static const EndpointCredentialKind ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY =
      EndpointCredentialKind._(
          2, _omitEnumNames ? '' : 'ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY');
  static const EndpointCredentialKind ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD =
      EndpointCredentialKind._(
          3, _omitEnumNames ? '' : 'ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD');
  static const EndpointCredentialKind
      ENDPOINT_CREDENTIAL_KIND_CAPABILITY_GRANT = EndpointCredentialKind._(
          4, _omitEnumNames ? '' : 'ENDPOINT_CREDENTIAL_KIND_CAPABILITY_GRANT');
  static const EndpointCredentialKind ENDPOINT_CREDENTIAL_KIND_CLOUD_PROFILE =
      EndpointCredentialKind._(
          5, _omitEnumNames ? '' : 'ENDPOINT_CREDENTIAL_KIND_CLOUD_PROFILE');

  static const $core.List<EndpointCredentialKind> values =
      <EndpointCredentialKind>[
    ENDPOINT_CREDENTIAL_KIND_UNSPECIFIED,
    ENDPOINT_CREDENTIAL_KIND_SSH_AGENT,
    ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY,
    ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD,
    ENDPOINT_CREDENTIAL_KIND_CAPABILITY_GRANT,
    ENDPOINT_CREDENTIAL_KIND_CLOUD_PROFILE,
  ];

  static final $core.List<EndpointCredentialKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static EndpointCredentialKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointCredentialKind._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
