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

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'remote_auth.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'remote_auth.pbenum.dart';

/// ChannelBinding 是 transport adapter 从实际 TLS/DTLS/local Unix channel 取得的 SHA-256 binding。
/// binding_hash 固定为 32 bytes；SDP、route 地址或 Cloud signaling 字段不能替代该值。
class ChannelBinding extends $pb.GeneratedMessage {
  factory ChannelBinding({
    ChannelBindingKind? kind,
    $core.List<$core.int>? bindingHash,
  }) {
    final result = create();
    if (kind != null) result.kind = kind;
    if (bindingHash != null) result.bindingHash = bindingHash;
    return result;
  }

  ChannelBinding._();

  factory ChannelBinding.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ChannelBinding.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChannelBinding',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aE<ChannelBindingKind>(1, _omitFieldNames ? '' : 'kind',
        enumValues: ChannelBindingKind.values)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'bindingHash', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChannelBinding clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChannelBinding copyWith(void Function(ChannelBinding) updates) =>
      super.copyWith((message) => updates(message as ChannelBinding))
          as ChannelBinding;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ChannelBinding create() => ChannelBinding._();
  @$core.override
  ChannelBinding createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ChannelBinding getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ChannelBinding>(create);
  static ChannelBinding? _defaultInstance;

  @$pb.TagNumber(1)
  ChannelBindingKind get kind => $_getN(0);
  @$pb.TagNumber(1)
  set kind(ChannelBindingKind value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasKind() => $_has(0);
  @$pb.TagNumber(1)
  void clearKind() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get bindingHash => $_getN(1);
  @$pb.TagNumber(2)
  set bindingHash($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBindingHash() => $_has(1);
  @$pb.TagNumber(2)
  void clearBindingHash() => $_clearField(2);
}

enum AuthEnvelope_Payload {
  deviceHello,
  capabilityOpen,
  capabilityAccepted,
  capabilityRejected,
  pairingOpen,
  pairingAccepted,
  notSet
}

/// AuthEnvelope 是 DataChannel protocol 切换前唯一允许出现的 versioned frame。
/// 每条 DataChannel 使用独立 auth_session_id；CapabilityAccepted 后该 envelope 不得再次出现。
class AuthEnvelope extends $pb.GeneratedMessage {
  factory AuthEnvelope({
    $core.String? protocol,
    $core.int? version,
    $core.String? authSessionId,
    DeviceHello? deviceHello,
    CapabilityOpen? capabilityOpen,
    CapabilityAccepted? capabilityAccepted,
    CapabilityRejected? capabilityRejected,
    PairingOpen? pairingOpen,
    PairingAccepted? pairingAccepted,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (authSessionId != null) result.authSessionId = authSessionId;
    if (deviceHello != null) result.deviceHello = deviceHello;
    if (capabilityOpen != null) result.capabilityOpen = capabilityOpen;
    if (capabilityAccepted != null)
      result.capabilityAccepted = capabilityAccepted;
    if (capabilityRejected != null)
      result.capabilityRejected = capabilityRejected;
    if (pairingOpen != null) result.pairingOpen = pairingOpen;
    if (pairingAccepted != null) result.pairingAccepted = pairingAccepted;
    return result;
  }

  AuthEnvelope._();

  factory AuthEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AuthEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, AuthEnvelope_Payload>
      _AuthEnvelope_PayloadByTag = {
    4: AuthEnvelope_Payload.deviceHello,
    5: AuthEnvelope_Payload.capabilityOpen,
    6: AuthEnvelope_Payload.capabilityAccepted,
    7: AuthEnvelope_Payload.capabilityRejected,
    8: AuthEnvelope_Payload.pairingOpen,
    9: AuthEnvelope_Payload.pairingAccepted,
    0: AuthEnvelope_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AuthEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [4, 5, 6, 7, 8, 9])
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'authSessionId')
    ..aOM<DeviceHello>(4, _omitFieldNames ? '' : 'deviceHello',
        subBuilder: DeviceHello.create)
    ..aOM<CapabilityOpen>(5, _omitFieldNames ? '' : 'capabilityOpen',
        subBuilder: CapabilityOpen.create)
    ..aOM<CapabilityAccepted>(6, _omitFieldNames ? '' : 'capabilityAccepted',
        subBuilder: CapabilityAccepted.create)
    ..aOM<CapabilityRejected>(7, _omitFieldNames ? '' : 'capabilityRejected',
        subBuilder: CapabilityRejected.create)
    ..aOM<PairingOpen>(8, _omitFieldNames ? '' : 'pairingOpen',
        subBuilder: PairingOpen.create)
    ..aOM<PairingAccepted>(9, _omitFieldNames ? '' : 'pairingAccepted',
        subBuilder: PairingAccepted.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AuthEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AuthEnvelope copyWith(void Function(AuthEnvelope) updates) =>
      super.copyWith((message) => updates(message as AuthEnvelope))
          as AuthEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AuthEnvelope create() => AuthEnvelope._();
  @$core.override
  AuthEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AuthEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AuthEnvelope>(create);
  static AuthEnvelope? _defaultInstance;

  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  AuthEnvelope_Payload whichPayload() =>
      _AuthEnvelope_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get authSessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set authSessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAuthSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearAuthSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  DeviceHello get deviceHello => $_getN(3);
  @$pb.TagNumber(4)
  set deviceHello(DeviceHello value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasDeviceHello() => $_has(3);
  @$pb.TagNumber(4)
  void clearDeviceHello() => $_clearField(4);
  @$pb.TagNumber(4)
  DeviceHello ensureDeviceHello() => $_ensure(3);

  @$pb.TagNumber(5)
  CapabilityOpen get capabilityOpen => $_getN(4);
  @$pb.TagNumber(5)
  set capabilityOpen(CapabilityOpen value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasCapabilityOpen() => $_has(4);
  @$pb.TagNumber(5)
  void clearCapabilityOpen() => $_clearField(5);
  @$pb.TagNumber(5)
  CapabilityOpen ensureCapabilityOpen() => $_ensure(4);

  @$pb.TagNumber(6)
  CapabilityAccepted get capabilityAccepted => $_getN(5);
  @$pb.TagNumber(6)
  set capabilityAccepted(CapabilityAccepted value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCapabilityAccepted() => $_has(5);
  @$pb.TagNumber(6)
  void clearCapabilityAccepted() => $_clearField(6);
  @$pb.TagNumber(6)
  CapabilityAccepted ensureCapabilityAccepted() => $_ensure(5);

  @$pb.TagNumber(7)
  CapabilityRejected get capabilityRejected => $_getN(6);
  @$pb.TagNumber(7)
  set capabilityRejected(CapabilityRejected value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasCapabilityRejected() => $_has(6);
  @$pb.TagNumber(7)
  void clearCapabilityRejected() => $_clearField(7);
  @$pb.TagNumber(7)
  CapabilityRejected ensureCapabilityRejected() => $_ensure(6);

  @$pb.TagNumber(8)
  PairingOpen get pairingOpen => $_getN(7);
  @$pb.TagNumber(8)
  set pairingOpen(PairingOpen value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasPairingOpen() => $_has(7);
  @$pb.TagNumber(8)
  void clearPairingOpen() => $_clearField(8);
  @$pb.TagNumber(8)
  PairingOpen ensurePairingOpen() => $_ensure(7);

  @$pb.TagNumber(9)
  PairingAccepted get pairingAccepted => $_getN(8);
  @$pb.TagNumber(9)
  set pairingAccepted(PairingAccepted value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasPairingAccepted() => $_has(8);
  @$pb.TagNumber(9)
  void clearPairingAccepted() => $_clearField(9);
  @$pb.TagNumber(9)
  PairingAccepted ensurePairingAccepted() => $_ensure(8);
}

/// DeviceHello 是 daemon 对长期 DeviceIdentity 和本次实际 transport channel 的联合证明。
class DeviceHello extends $pb.GeneratedMessage {
  factory DeviceHello({
    $core.String? deviceId,
    $core.List<$core.int>? devicePublicKey,
    $core.String? deviceFingerprint,
    $core.List<$core.int>? serverNonce,
    ChannelBinding? channelBinding,
    $fixnum.Int64? issuedAtUnixNano,
    $core.List<$core.int>? signature,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (serverNonce != null) result.serverNonce = serverNonce;
    if (channelBinding != null) result.channelBinding = channelBinding;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (signature != null) result.signature = signature;
    return result;
  }

  DeviceHello._();

  factory DeviceHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeviceHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeviceHello',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'deviceFingerprint')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'serverNonce', $pb.PbFieldType.OY)
    ..aOM<ChannelBinding>(5, _omitFieldNames ? '' : 'channelBinding',
        subBuilder: ChannelBinding.create)
    ..aInt64(6, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceHello copyWith(void Function(DeviceHello) updates) =>
      super.copyWith((message) => updates(message as DeviceHello))
          as DeviceHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeviceHello create() => DeviceHello._();
  @$core.override
  DeviceHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeviceHello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeviceHello>(create);
  static DeviceHello? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get devicePublicKey => $_getN(1);
  @$pb.TagNumber(2)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDevicePublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearDevicePublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceFingerprint() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get serverNonce => $_getN(3);
  @$pb.TagNumber(4)
  set serverNonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasServerNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearServerNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  ChannelBinding get channelBinding => $_getN(4);
  @$pb.TagNumber(5)
  set channelBinding(ChannelBinding value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasChannelBinding() => $_has(4);
  @$pb.TagNumber(5)
  void clearChannelBinding() => $_clearField(5);
  @$pb.TagNumber(5)
  ChannelBinding ensureChannelBinding() => $_ensure(4);

  @$pb.TagNumber(6)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIssuedAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearIssuedAtUnixNano() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get signature => $_getN(6);
  @$pb.TagNumber(7)
  set signature($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSignature() => $_has(6);
  @$pb.TagNumber(7)
  void clearSignature() => $_clearField(7);
}

/// CapabilityOpen 证明 client 持有 grant 对应 ClientAccessIdentity private key，并把 proof 绑定当前 channel。
/// grant 只允许出现在端到端 direct TLS 或 DTLS DataChannel 内，不能进入 cloud signaling DTO。
class CapabilityOpen extends $pb.GeneratedMessage {
  factory CapabilityOpen({
    $core.String? grant,
    $core.List<$core.int>? clientPublicKey,
    $core.List<$core.int>? clientNonce,
    $core.List<$core.int>? proof,
  }) {
    final result = create();
    if (grant != null) result.grant = grant;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (clientNonce != null) result.clientNonce = clientNonce;
    if (proof != null) result.proof = proof;
    return result;
  }

  CapabilityOpen._();

  factory CapabilityOpen.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CapabilityOpen.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CapabilityOpen',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'grant')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'clientNonce', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'proof', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityOpen clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityOpen copyWith(void Function(CapabilityOpen) updates) =>
      super.copyWith((message) => updates(message as CapabilityOpen))
          as CapabilityOpen;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CapabilityOpen create() => CapabilityOpen._();
  @$core.override
  CapabilityOpen createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CapabilityOpen getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CapabilityOpen>(create);
  static CapabilityOpen? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get grant => $_getSZ(0);
  @$pb.TagNumber(1)
  set grant($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGrant() => $_has(0);
  @$pb.TagNumber(1)
  void clearGrant() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get clientPublicKey => $_getN(1);
  @$pb.TagNumber(2)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasClientPublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearClientPublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get clientNonce => $_getN(2);
  @$pb.TagNumber(3)
  set clientNonce($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasClientNonce() => $_has(2);
  @$pb.TagNumber(3)
  void clearClientNonce() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get proof => $_getN(3);
  @$pb.TagNumber(4)
  set proof($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProof() => $_has(3);
  @$pb.TagNumber(4)
  void clearProof() => $_clearField(4);
}

/// PairingOpen 只能兑换一次性 pairing claim，不能访问 terminal、history、file 或普通 protocol method。
/// client_public_key/proof 先证明目标 ClientAccessIdentity possession，daemon 才允许原子消费 claim。
class PairingOpen extends $pb.GeneratedMessage {
  factory PairingOpen({
    $core.List<$core.int>? pairingClaimOffer,
    $core.List<$core.int>? clientPublicKey,
    $core.String? clientLabel,
    $core.List<$core.int>? clientNonce,
    $core.List<$core.int>? proof,
    $core.int? clientProduct,
  }) {
    final result = create();
    if (pairingClaimOffer != null) result.pairingClaimOffer = pairingClaimOffer;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (clientLabel != null) result.clientLabel = clientLabel;
    if (clientNonce != null) result.clientNonce = clientNonce;
    if (proof != null) result.proof = proof;
    if (clientProduct != null) result.clientProduct = clientProduct;
    return result;
  }

  PairingOpen._();

  factory PairingOpen.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingOpen.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingOpen',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'pairingClaimOffer', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'clientLabel')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'clientNonce', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'proof', $pb.PbFieldType.OY)
    ..aI(6, _omitFieldNames ? '' : 'clientProduct',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingOpen clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingOpen copyWith(void Function(PairingOpen) updates) =>
      super.copyWith((message) => updates(message as PairingOpen))
          as PairingOpen;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingOpen create() => PairingOpen._();
  @$core.override
  PairingOpen createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingOpen getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingOpen>(create);
  static PairingOpen? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get pairingClaimOffer => $_getN(0);
  @$pb.TagNumber(1)
  set pairingClaimOffer($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPairingClaimOffer() => $_has(0);
  @$pb.TagNumber(1)
  void clearPairingClaimOffer() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get clientPublicKey => $_getN(1);
  @$pb.TagNumber(2)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasClientPublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearClientPublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get clientLabel => $_getSZ(2);
  @$pb.TagNumber(3)
  set clientLabel($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasClientLabel() => $_has(2);
  @$pb.TagNumber(3)
  void clearClientLabel() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get clientNonce => $_getN(3);
  @$pb.TagNumber(4)
  set clientNonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasClientNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearClientNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get proof => $_getN(4);
  @$pb.TagNumber(5)
  set proof($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProof() => $_has(4);
  @$pb.TagNumber(5)
  void clearProof() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get clientProduct => $_getIZ(5);
  @$pb.TagNumber(6)
  set clientProduct($core.int value) => $_setUnsignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasClientProduct() => $_has(5);
  @$pb.TagNumber(6)
  void clearClientProduct() => $_clearField(6);
}

/// ScopeSummary 是 daemon 已接受 capability 的最小无歧义投影。
class ScopeSummary extends $pb.GeneratedMessage {
  factory ScopeSummary({
    ScopeKind? kind,
    $core.String? terminalId,
    $core.bool? manageClientAccess,
  }) {
    final result = create();
    if (kind != null) result.kind = kind;
    if (terminalId != null) result.terminalId = terminalId;
    if (manageClientAccess != null)
      result.manageClientAccess = manageClientAccess;
    return result;
  }

  ScopeSummary._();

  factory ScopeSummary.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScopeSummary.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScopeSummary',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aE<ScopeKind>(1, _omitFieldNames ? '' : 'kind',
        enumValues: ScopeKind.values)
    ..aOS(2, _omitFieldNames ? '' : 'terminalId')
    ..aOB(3, _omitFieldNames ? '' : 'manageClientAccess')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScopeSummary clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScopeSummary copyWith(void Function(ScopeSummary) updates) =>
      super.copyWith((message) => updates(message as ScopeSummary))
          as ScopeSummary;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScopeSummary create() => ScopeSummary._();
  @$core.override
  ScopeSummary createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScopeSummary getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScopeSummary>(create);
  static ScopeSummary? _defaultInstance;

  @$pb.TagNumber(1)
  ScopeKind get kind => $_getN(0);
  @$pb.TagNumber(1)
  set kind(ScopeKind value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasKind() => $_has(0);
  @$pb.TagNumber(1)
  void clearKind() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get terminalId => $_getSZ(1);
  @$pb.TagNumber(2)
  set terminalId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminalId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTerminalId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get manageClientAccess => $_getBF(2);
  @$pb.TagNumber(3)
  set manageClientAccess($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasManageClientAccess() => $_has(2);
  @$pb.TagNumber(3)
  void clearManageClientAccess() => $_clearField(3);
}

/// ClientAccessScope 是 remote.access 管理 RPC 使用的完整 typed scope。
/// 它与 signed CapabilityGrant scope 一一对应，不通过 Struct 或字符串字段表复制 schema。
class ClientAccessScope extends $pb.GeneratedMessage {
  factory ClientAccessScope({
    $core.bool? allowDaemon,
    $core.String? terminalId,
    $core.bool? machineEventsOnly,
    $core.bool? fileReadMetadata,
    $core.bool? fileReadContent,
    $core.bool? fileWriteContent,
    $core.bool? fileMutate,
    $core.bool? manageClientAccess,
  }) {
    final result = create();
    if (allowDaemon != null) result.allowDaemon = allowDaemon;
    if (terminalId != null) result.terminalId = terminalId;
    if (machineEventsOnly != null) result.machineEventsOnly = machineEventsOnly;
    if (fileReadMetadata != null) result.fileReadMetadata = fileReadMetadata;
    if (fileReadContent != null) result.fileReadContent = fileReadContent;
    if (fileWriteContent != null) result.fileWriteContent = fileWriteContent;
    if (fileMutate != null) result.fileMutate = fileMutate;
    if (manageClientAccess != null)
      result.manageClientAccess = manageClientAccess;
    return result;
  }

  ClientAccessScope._();

  factory ClientAccessScope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessScope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessScope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'allowDaemon')
    ..aOS(2, _omitFieldNames ? '' : 'terminalId')
    ..aOB(3, _omitFieldNames ? '' : 'machineEventsOnly')
    ..aOB(4, _omitFieldNames ? '' : 'fileReadMetadata')
    ..aOB(5, _omitFieldNames ? '' : 'fileReadContent')
    ..aOB(6, _omitFieldNames ? '' : 'fileWriteContent')
    ..aOB(7, _omitFieldNames ? '' : 'fileMutate')
    ..aOB(8, _omitFieldNames ? '' : 'manageClientAccess')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessScope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessScope copyWith(void Function(ClientAccessScope) updates) =>
      super.copyWith((message) => updates(message as ClientAccessScope))
          as ClientAccessScope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessScope create() => ClientAccessScope._();
  @$core.override
  ClientAccessScope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessScope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessScope>(create);
  static ClientAccessScope? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get allowDaemon => $_getBF(0);
  @$pb.TagNumber(1)
  set allowDaemon($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAllowDaemon() => $_has(0);
  @$pb.TagNumber(1)
  void clearAllowDaemon() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get terminalId => $_getSZ(1);
  @$pb.TagNumber(2)
  set terminalId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminalId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTerminalId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get machineEventsOnly => $_getBF(2);
  @$pb.TagNumber(3)
  set machineEventsOnly($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMachineEventsOnly() => $_has(2);
  @$pb.TagNumber(3)
  void clearMachineEventsOnly() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get fileReadMetadata => $_getBF(3);
  @$pb.TagNumber(4)
  set fileReadMetadata($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasFileReadMetadata() => $_has(3);
  @$pb.TagNumber(4)
  void clearFileReadMetadata() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get fileReadContent => $_getBF(4);
  @$pb.TagNumber(5)
  set fileReadContent($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasFileReadContent() => $_has(4);
  @$pb.TagNumber(5)
  void clearFileReadContent() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get fileWriteContent => $_getBF(5);
  @$pb.TagNumber(6)
  set fileWriteContent($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFileWriteContent() => $_has(5);
  @$pb.TagNumber(6)
  void clearFileWriteContent() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get fileMutate => $_getBF(6);
  @$pb.TagNumber(7)
  set fileMutate($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasFileMutate() => $_has(6);
  @$pb.TagNumber(7)
  void clearFileMutate() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get manageClientAccess => $_getBF(7);
  @$pb.TagNumber(8)
  set manageClientAccess($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasManageClientAccess() => $_has(7);
  @$pb.TagNumber(8)
  void clearManageClientAccess() => $_clearField(8);
}

class ClientAccessIdentityResult extends $pb.GeneratedMessage {
  factory ClientAccessIdentityResult({
    $core.String? deviceId,
    $core.String? deviceFingerprint,
    $core.List<$core.int>? devicePublicKey,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    return result;
  }

  ClientAccessIdentityResult._();

  factory ClientAccessIdentityResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessIdentityResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessIdentityResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceFingerprint')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessIdentityResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessIdentityResult copyWith(
          void Function(ClientAccessIdentityResult) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessIdentityResult))
          as ClientAccessIdentityResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessIdentityResult create() => ClientAccessIdentityResult._();
  @$core.override
  ClientAccessIdentityResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessIdentityResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessIdentityResult>(create);
  static ClientAccessIdentityResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceFingerprint => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceFingerprint($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceFingerprint() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceFingerprint() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get devicePublicKey => $_getN(2);
  @$pb.TagNumber(3)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDevicePublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearDevicePublicKey() => $_clearField(3);
}

/// DeviceIdentityProofInput 是 local/SSH application session 上 daemon fresh identity proof 的 canonical 签名输入。
/// challenge 由客户端为本次连接随机生成；签名只证明当前响应方持有 DeviceIdentity 私钥，不替代 transport authorization。
class DeviceIdentityProofInput extends $pb.GeneratedMessage {
  factory DeviceIdentityProofInput({
    $core.String? domain,
    $core.List<$core.int>? challenge,
    $core.String? deviceId,
    $core.String? deviceFingerprint,
    $core.List<$core.int>? devicePublicKey,
  }) {
    final result = create();
    if (domain != null) result.domain = domain;
    if (challenge != null) result.challenge = challenge;
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    return result;
  }

  DeviceIdentityProofInput._();

  factory DeviceIdentityProofInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeviceIdentityProofInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeviceIdentityProofInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'domain')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'challenge', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'deviceId')
    ..aOS(4, _omitFieldNames ? '' : 'deviceFingerprint')
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceIdentityProofInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceIdentityProofInput copyWith(
          void Function(DeviceIdentityProofInput) updates) =>
      super.copyWith((message) => updates(message as DeviceIdentityProofInput))
          as DeviceIdentityProofInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeviceIdentityProofInput create() => DeviceIdentityProofInput._();
  @$core.override
  DeviceIdentityProofInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeviceIdentityProofInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeviceIdentityProofInput>(create);
  static DeviceIdentityProofInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get domain => $_getSZ(0);
  @$pb.TagNumber(1)
  set domain($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDomain() => $_has(0);
  @$pb.TagNumber(1)
  void clearDomain() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get challenge => $_getN(1);
  @$pb.TagNumber(2)
  set challenge($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChallenge() => $_has(1);
  @$pb.TagNumber(2)
  void clearChallenge() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get deviceFingerprint => $_getSZ(3);
  @$pb.TagNumber(4)
  set deviceFingerprint($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDeviceFingerprint() => $_has(3);
  @$pb.TagNumber(4)
  void clearDeviceFingerprint() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get devicePublicKey => $_getN(4);
  @$pb.TagNumber(5)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDevicePublicKey() => $_has(4);
  @$pb.TagNumber(5)
  void clearDevicePublicKey() => $_clearField(5);
}

class ClientAccessTicketCreateRequest extends $pb.GeneratedMessage {
  factory ClientAccessTicketCreateRequest({
    $core.String? label,
    ClientAccessScope? scope,
    $fixnum.Int64? ticketTtlSeconds,
    $fixnum.Int64? grantLifetimeSeconds,
    $core.Iterable<EndpointRouteConfigV1>? routes,
    $core.String? accessLabel,
  }) {
    final result = create();
    if (label != null) result.label = label;
    if (scope != null) result.scope = scope;
    if (ticketTtlSeconds != null) result.ticketTtlSeconds = ticketTtlSeconds;
    if (grantLifetimeSeconds != null)
      result.grantLifetimeSeconds = grantLifetimeSeconds;
    if (routes != null) result.routes.addAll(routes);
    if (accessLabel != null) result.accessLabel = accessLabel;
    return result;
  }

  ClientAccessTicketCreateRequest._();

  factory ClientAccessTicketCreateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessTicketCreateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessTicketCreateRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'label')
    ..aOM<ClientAccessScope>(2, _omitFieldNames ? '' : 'scope',
        subBuilder: ClientAccessScope.create)
    ..aInt64(3, _omitFieldNames ? '' : 'ticketTtlSeconds')
    ..aInt64(4, _omitFieldNames ? '' : 'grantLifetimeSeconds')
    ..pPM<EndpointRouteConfigV1>(5, _omitFieldNames ? '' : 'routes',
        subBuilder: EndpointRouteConfigV1.create)
    ..aOS(6, _omitFieldNames ? '' : 'accessLabel')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateRequest copyWith(
          void Function(ClientAccessTicketCreateRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessTicketCreateRequest))
          as ClientAccessTicketCreateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateRequest create() =>
      ClientAccessTicketCreateRequest._();
  @$core.override
  ClientAccessTicketCreateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessTicketCreateRequest>(
          create);
  static ClientAccessTicketCreateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get label => $_getSZ(0);
  @$pb.TagNumber(1)
  set label($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLabel() => $_has(0);
  @$pb.TagNumber(1)
  void clearLabel() => $_clearField(1);

  @$pb.TagNumber(2)
  ClientAccessScope get scope => $_getN(1);
  @$pb.TagNumber(2)
  set scope(ClientAccessScope value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasScope() => $_has(1);
  @$pb.TagNumber(2)
  void clearScope() => $_clearField(2);
  @$pb.TagNumber(2)
  ClientAccessScope ensureScope() => $_ensure(1);

  @$pb.TagNumber(3)
  $fixnum.Int64 get ticketTtlSeconds => $_getI64(2);
  @$pb.TagNumber(3)
  set ticketTtlSeconds($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTicketTtlSeconds() => $_has(2);
  @$pb.TagNumber(3)
  void clearTicketTtlSeconds() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get grantLifetimeSeconds => $_getI64(3);
  @$pb.TagNumber(4)
  set grantLifetimeSeconds($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasGrantLifetimeSeconds() => $_has(3);
  @$pb.TagNumber(4)
  void clearGrantLifetimeSeconds() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<EndpointRouteConfigV1> get routes => $_getList(4);

  /// access_label 由授权所有者命名，用于本地审计和撤销定位；它不同于客户端自报的 client_label。
  @$pb.TagNumber(6)
  $core.String get accessLabel => $_getSZ(5);
  @$pb.TagNumber(6)
  set accessLabel($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAccessLabel() => $_has(5);
  @$pb.TagNumber(6)
  void clearAccessLabel() => $_clearField(6);
}

class ClientAccessTicketCreateResult extends $pb.GeneratedMessage {
  factory ClientAccessTicketCreateResult({
    $core.String? ticketId,
    $fixnum.Int64? expiresAtUnixNano,
    $core.List<$core.int>? claimOffer,
    $core.String? claimCode,
  }) {
    final result = create();
    if (ticketId != null) result.ticketId = ticketId;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (claimOffer != null) result.claimOffer = claimOffer;
    if (claimCode != null) result.claimCode = claimCode;
    return result;
  }

  ClientAccessTicketCreateResult._();

  factory ClientAccessTicketCreateResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessTicketCreateResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessTicketCreateResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'ticketId')
    ..aInt64(2, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'claimOffer', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'claimCode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateResult copyWith(
          void Function(ClientAccessTicketCreateResult) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessTicketCreateResult))
          as ClientAccessTicketCreateResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateResult create() =>
      ClientAccessTicketCreateResult._();
  @$core.override
  ClientAccessTicketCreateResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessTicketCreateResult>(create);
  static ClientAccessTicketCreateResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get ticketId => $_getSZ(0);
  @$pb.TagNumber(1)
  set ticketId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTicketId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTicketId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(1);
  @$pb.TagNumber(2)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExpiresAtUnixNano() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpiresAtUnixNano() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get claimOffer => $_getN(2);
  @$pb.TagNumber(3)
  set claimOffer($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasClaimOffer() => $_has(2);
  @$pb.TagNumber(3)
  void clearClaimOffer() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get claimCode => $_getSZ(3);
  @$pb.TagNumber(4)
  set claimCode($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasClaimCode() => $_has(3);
  @$pb.TagNumber(4)
  void clearClaimCode() => $_clearField(4);
}

class ClientAccessRevokeRequest extends $pb.GeneratedMessage {
  factory ClientAccessRevokeRequest({
    $core.String? grantId,
  }) {
    final result = create();
    if (grantId != null) result.grantId = grantId;
    return result;
  }

  ClientAccessRevokeRequest._();

  factory ClientAccessRevokeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessRevokeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessRevokeRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'grantId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRevokeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRevokeRequest copyWith(
          void Function(ClientAccessRevokeRequest) updates) =>
      super.copyWith((message) => updates(message as ClientAccessRevokeRequest))
          as ClientAccessRevokeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessRevokeRequest create() => ClientAccessRevokeRequest._();
  @$core.override
  ClientAccessRevokeRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessRevokeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessRevokeRequest>(create);
  static ClientAccessRevokeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get grantId => $_getSZ(0);
  @$pb.TagNumber(1)
  set grantId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGrantId() => $_has(0);
  @$pb.TagNumber(1)
  void clearGrantId() => $_clearField(1);
}

class ClientAccessRecord extends $pb.GeneratedMessage {
  factory ClientAccessRecord({
    $core.String? grantId,
    $core.String? revocationId,
    $core.String? subjectKeyFingerprint,
    $core.String? clientLabel,
    ClientAccessScope? scope,
    $fixnum.Int64? issuedAtUnixNano,
    $fixnum.Int64? expiresAtUnixNano,
    $fixnum.Int64? revokedAtUnixNano,
    $core.String? accessLabel,
  }) {
    final result = create();
    if (grantId != null) result.grantId = grantId;
    if (revocationId != null) result.revocationId = revocationId;
    if (subjectKeyFingerprint != null)
      result.subjectKeyFingerprint = subjectKeyFingerprint;
    if (clientLabel != null) result.clientLabel = clientLabel;
    if (scope != null) result.scope = scope;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (revokedAtUnixNano != null) result.revokedAtUnixNano = revokedAtUnixNano;
    if (accessLabel != null) result.accessLabel = accessLabel;
    return result;
  }

  ClientAccessRecord._();

  factory ClientAccessRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessRecord',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'grantId')
    ..aOS(2, _omitFieldNames ? '' : 'revocationId')
    ..aOS(3, _omitFieldNames ? '' : 'subjectKeyFingerprint')
    ..aOS(4, _omitFieldNames ? '' : 'clientLabel')
    ..aOM<ClientAccessScope>(5, _omitFieldNames ? '' : 'scope',
        subBuilder: ClientAccessScope.create)
    ..aInt64(6, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..aInt64(7, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..aInt64(8, _omitFieldNames ? '' : 'revokedAtUnixNano')
    ..aOS(9, _omitFieldNames ? '' : 'accessLabel')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRecord copyWith(void Function(ClientAccessRecord) updates) =>
      super.copyWith((message) => updates(message as ClientAccessRecord))
          as ClientAccessRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessRecord create() => ClientAccessRecord._();
  @$core.override
  ClientAccessRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessRecord>(create);
  static ClientAccessRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get grantId => $_getSZ(0);
  @$pb.TagNumber(1)
  set grantId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGrantId() => $_has(0);
  @$pb.TagNumber(1)
  void clearGrantId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get revocationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set revocationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevocationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevocationId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subjectKeyFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectKeyFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectKeyFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectKeyFingerprint() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get clientLabel => $_getSZ(3);
  @$pb.TagNumber(4)
  set clientLabel($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasClientLabel() => $_has(3);
  @$pb.TagNumber(4)
  void clearClientLabel() => $_clearField(4);

  @$pb.TagNumber(5)
  ClientAccessScope get scope => $_getN(4);
  @$pb.TagNumber(5)
  set scope(ClientAccessScope value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasScope() => $_has(4);
  @$pb.TagNumber(5)
  void clearScope() => $_clearField(5);
  @$pb.TagNumber(5)
  ClientAccessScope ensureScope() => $_ensure(4);

  @$pb.TagNumber(6)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIssuedAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearIssuedAtUnixNano() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(6);
  @$pb.TagNumber(7)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasExpiresAtUnixNano() => $_has(6);
  @$pb.TagNumber(7)
  void clearExpiresAtUnixNano() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get revokedAtUnixNano => $_getI64(7);
  @$pb.TagNumber(8)
  set revokedAtUnixNano($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRevokedAtUnixNano() => $_has(7);
  @$pb.TagNumber(8)
  void clearRevokedAtUnixNano() => $_clearField(8);

  /// access_label 由签发授权的所有者控制，客户端不能在兑换时修改。
  @$pb.TagNumber(9)
  $core.String get accessLabel => $_getSZ(8);
  @$pb.TagNumber(9)
  set accessLabel($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasAccessLabel() => $_has(8);
  @$pb.TagNumber(9)
  void clearAccessLabel() => $_clearField(9);
}

class ClientAccessListResult extends $pb.GeneratedMessage {
  factory ClientAccessListResult({
    $core.Iterable<ClientAccessRecord>? records,
  }) {
    final result = create();
    if (records != null) result.records.addAll(records);
    return result;
  }

  ClientAccessListResult._();

  factory ClientAccessListResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessListResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessListResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..pPM<ClientAccessRecord>(1, _omitFieldNames ? '' : 'records',
        subBuilder: ClientAccessRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessListResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessListResult copyWith(
          void Function(ClientAccessListResult) updates) =>
      super.copyWith((message) => updates(message as ClientAccessListResult))
          as ClientAccessListResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessListResult create() => ClientAccessListResult._();
  @$core.override
  ClientAccessListResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessListResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessListResult>(create);
  static ClientAccessListResult? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ClientAccessRecord> get records => $_getList(0);
}

/// CapabilityAccepted 是 remote auth 到 AnyTTY protocol 的单向切换点。
class CapabilityAccepted extends $pb.GeneratedMessage {
  factory CapabilityAccepted({
    $core.String? grantId,
    ScopeSummary? scope,
    $core.String? subjectKeyFingerprint,
  }) {
    final result = create();
    if (grantId != null) result.grantId = grantId;
    if (scope != null) result.scope = scope;
    if (subjectKeyFingerprint != null)
      result.subjectKeyFingerprint = subjectKeyFingerprint;
    return result;
  }

  CapabilityAccepted._();

  factory CapabilityAccepted.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CapabilityAccepted.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CapabilityAccepted',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'grantId')
    ..aOM<ScopeSummary>(2, _omitFieldNames ? '' : 'scope',
        subBuilder: ScopeSummary.create)
    ..aOS(3, _omitFieldNames ? '' : 'subjectKeyFingerprint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityAccepted clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityAccepted copyWith(void Function(CapabilityAccepted) updates) =>
      super.copyWith((message) => updates(message as CapabilityAccepted))
          as CapabilityAccepted;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CapabilityAccepted create() => CapabilityAccepted._();
  @$core.override
  CapabilityAccepted createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CapabilityAccepted getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CapabilityAccepted>(create);
  static CapabilityAccepted? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get grantId => $_getSZ(0);
  @$pb.TagNumber(1)
  set grantId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGrantId() => $_has(0);
  @$pb.TagNumber(1)
  void clearGrantId() => $_clearField(1);

  @$pb.TagNumber(2)
  ScopeSummary get scope => $_getN(1);
  @$pb.TagNumber(2)
  set scope(ScopeSummary value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasScope() => $_has(1);
  @$pb.TagNumber(2)
  void clearScope() => $_clearField(2);
  @$pb.TagNumber(2)
  ScopeSummary ensureScope() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.String get subjectKeyFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectKeyFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectKeyFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectKeyFingerprint() => $_clearField(3);
}

/// PairingAccepted 返回本次原子事务已持久化的 client-bound grant 和稳定 delivery receipt。
/// 响应丢失后，相同 ticket 与相同 client key 可以取回完全相同的 grant/receipt；其他 key 必须 fail closed。
class PairingAccepted extends $pb.GeneratedMessage {
  factory PairingAccepted({
    $core.String? grant,
    $core.String? deliveryReceipt,
    $core.String? subjectKeyFingerprint,
    ScopeSummary? scope,
    $core.List<$core.int>? pairingBundle,
    $core.List<$core.int>? cloudRouteGrant,
    $core.List<$core.int>? cloudEdgeLocator,
  }) {
    final result = create();
    if (grant != null) result.grant = grant;
    if (deliveryReceipt != null) result.deliveryReceipt = deliveryReceipt;
    if (subjectKeyFingerprint != null)
      result.subjectKeyFingerprint = subjectKeyFingerprint;
    if (scope != null) result.scope = scope;
    if (pairingBundle != null) result.pairingBundle = pairingBundle;
    if (cloudRouteGrant != null) result.cloudRouteGrant = cloudRouteGrant;
    if (cloudEdgeLocator != null) result.cloudEdgeLocator = cloudEdgeLocator;
    return result;
  }

  PairingAccepted._();

  factory PairingAccepted.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingAccepted.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingAccepted',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'grant')
    ..aOS(2, _omitFieldNames ? '' : 'deliveryReceipt')
    ..aOS(3, _omitFieldNames ? '' : 'subjectKeyFingerprint')
    ..aOM<ScopeSummary>(4, _omitFieldNames ? '' : 'scope',
        subBuilder: ScopeSummary.create)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'pairingBundle', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'cloudRouteGrant', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'cloudEdgeLocator', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingAccepted clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingAccepted copyWith(void Function(PairingAccepted) updates) =>
      super.copyWith((message) => updates(message as PairingAccepted))
          as PairingAccepted;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingAccepted create() => PairingAccepted._();
  @$core.override
  PairingAccepted createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingAccepted getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingAccepted>(create);
  static PairingAccepted? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get grant => $_getSZ(0);
  @$pb.TagNumber(1)
  set grant($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGrant() => $_has(0);
  @$pb.TagNumber(1)
  void clearGrant() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deliveryReceipt => $_getSZ(1);
  @$pb.TagNumber(2)
  set deliveryReceipt($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeliveryReceipt() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeliveryReceipt() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subjectKeyFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectKeyFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectKeyFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectKeyFingerprint() => $_clearField(3);

  @$pb.TagNumber(4)
  ScopeSummary get scope => $_getN(3);
  @$pb.TagNumber(4)
  set scope(ScopeSummary value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasScope() => $_has(3);
  @$pb.TagNumber(4)
  void clearScope() => $_clearField(4);
  @$pb.TagNumber(4)
  ScopeSummary ensureScope() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.List<$core.int> get pairingBundle => $_getN(4);
  @$pb.TagNumber(5)
  set pairingBundle($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPairingBundle() => $_has(4);
  @$pb.TagNumber(5)
  void clearPairingBundle() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get cloudRouteGrant => $_getN(5);
  @$pb.TagNumber(6)
  set cloudRouteGrant($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCloudRouteGrant() => $_has(5);
  @$pb.TagNumber(6)
  void clearCloudRouteGrant() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get cloudEdgeLocator => $_getN(6);
  @$pb.TagNumber(7)
  set cloudEdgeLocator($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCloudEdgeLocator() => $_has(6);
  @$pb.TagNumber(7)
  void clearCloudEdgeLocator() => $_clearField(7);
}

/// CapabilityRejected 返回稳定错误类别和固定脱敏消息，随后 channel 必须关闭。
class CapabilityRejected extends $pb.GeneratedMessage {
  factory CapabilityRejected({
    AuthErrorCode? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  CapabilityRejected._();

  factory CapabilityRejected.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CapabilityRejected.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CapabilityRejected',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aE<AuthErrorCode>(1, _omitFieldNames ? '' : 'code',
        enumValues: AuthErrorCode.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityRejected clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityRejected copyWith(void Function(CapabilityRejected) updates) =>
      super.copyWith((message) => updates(message as CapabilityRejected))
          as CapabilityRejected;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CapabilityRejected create() => CapabilityRejected._();
  @$core.override
  CapabilityRejected createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CapabilityRejected getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CapabilityRejected>(create);
  static CapabilityRejected? _defaultInstance;

  @$pb.TagNumber(1)
  AuthErrorCode get code => $_getN(0);
  @$pb.TagNumber(1)
  set code(AuthErrorCode value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

/// DeviceHelloSignatureInput 是 Ed25519 签名的跨平台 canonical protobuf。
/// 所有实现必须拒绝 unknown field，并使用 deterministic protobuf bytes，不得改用 JSON 或字段拼接。
class DeviceHelloSignatureInput extends $pb.GeneratedMessage {
  factory DeviceHelloSignatureInput({
    $core.String? protocol,
    $core.int? version,
    $core.String? authSessionId,
    $core.String? deviceId,
    $core.List<$core.int>? devicePublicKey,
    $core.String? deviceFingerprint,
    $core.List<$core.int>? serverNonce,
    ChannelBinding? channelBinding,
    $fixnum.Int64? issuedAtUnixNano,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (authSessionId != null) result.authSessionId = authSessionId;
    if (deviceId != null) result.deviceId = deviceId;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (serverNonce != null) result.serverNonce = serverNonce;
    if (channelBinding != null) result.channelBinding = channelBinding;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    return result;
  }

  DeviceHelloSignatureInput._();

  factory DeviceHelloSignatureInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeviceHelloSignatureInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeviceHelloSignatureInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'authSessionId')
    ..aOS(4, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..aOS(6, _omitFieldNames ? '' : 'deviceFingerprint')
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'serverNonce', $pb.PbFieldType.OY)
    ..aOM<ChannelBinding>(8, _omitFieldNames ? '' : 'channelBinding',
        subBuilder: ChannelBinding.create)
    ..aInt64(9, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceHelloSignatureInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceHelloSignatureInput copyWith(
          void Function(DeviceHelloSignatureInput) updates) =>
      super.copyWith((message) => updates(message as DeviceHelloSignatureInput))
          as DeviceHelloSignatureInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeviceHelloSignatureInput create() => DeviceHelloSignatureInput._();
  @$core.override
  DeviceHelloSignatureInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeviceHelloSignatureInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeviceHelloSignatureInput>(create);
  static DeviceHelloSignatureInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get authSessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set authSessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAuthSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearAuthSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get deviceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set deviceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDeviceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearDeviceId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get devicePublicKey => $_getN(4);
  @$pb.TagNumber(5)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDevicePublicKey() => $_has(4);
  @$pb.TagNumber(5)
  void clearDevicePublicKey() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get deviceFingerprint => $_getSZ(5);
  @$pb.TagNumber(6)
  set deviceFingerprint($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDeviceFingerprint() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeviceFingerprint() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get serverNonce => $_getN(6);
  @$pb.TagNumber(7)
  set serverNonce($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasServerNonce() => $_has(6);
  @$pb.TagNumber(7)
  void clearServerNonce() => $_clearField(7);

  @$pb.TagNumber(8)
  ChannelBinding get channelBinding => $_getN(7);
  @$pb.TagNumber(8)
  set channelBinding(ChannelBinding value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasChannelBinding() => $_has(7);
  @$pb.TagNumber(8)
  void clearChannelBinding() => $_clearField(8);
  @$pb.TagNumber(8)
  ChannelBinding ensureChannelBinding() => $_ensure(7);

  @$pb.TagNumber(9)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(8);
  @$pb.TagNumber(9)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasIssuedAtUnixNano() => $_has(8);
  @$pb.TagNumber(9)
  void clearIssuedAtUnixNano() => $_clearField(9);
}

/// ClientProofInput 是 ClientAccessIdentity Ed25519 签名的跨平台 canonical protobuf。
/// credential_sha256 对 capability 使用原始 grant UTF-8 bytes、对 pairing 使用完整 canonical bootstrap protobuf bytes；open_kind 防止跨状态机重放。
class ClientProofInput extends $pb.GeneratedMessage {
  factory ClientProofInput({
    $core.String? protocol,
    $core.int? version,
    $core.String? authSessionId,
    $core.List<$core.int>? serverNonce,
    $core.List<$core.int>? clientNonce,
    ChannelBinding? channelBinding,
    $core.List<$core.int>? credentialSha256,
    $core.List<$core.int>? clientPublicKey,
    AuthOpenKind? openKind,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (authSessionId != null) result.authSessionId = authSessionId;
    if (serverNonce != null) result.serverNonce = serverNonce;
    if (clientNonce != null) result.clientNonce = clientNonce;
    if (channelBinding != null) result.channelBinding = channelBinding;
    if (credentialSha256 != null) result.credentialSha256 = credentialSha256;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (openKind != null) result.openKind = openKind;
    return result;
  }

  ClientProofInput._();

  factory ClientProofInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientProofInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientProofInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'authSessionId')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'serverNonce', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'clientNonce', $pb.PbFieldType.OY)
    ..aOM<ChannelBinding>(6, _omitFieldNames ? '' : 'channelBinding',
        subBuilder: ChannelBinding.create)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'credentialSha256', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..aE<AuthOpenKind>(9, _omitFieldNames ? '' : 'openKind',
        enumValues: AuthOpenKind.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientProofInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientProofInput copyWith(void Function(ClientProofInput) updates) =>
      super.copyWith((message) => updates(message as ClientProofInput))
          as ClientProofInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientProofInput create() => ClientProofInput._();
  @$core.override
  ClientProofInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientProofInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientProofInput>(create);
  static ClientProofInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get authSessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set authSessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAuthSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearAuthSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get serverNonce => $_getN(3);
  @$pb.TagNumber(4)
  set serverNonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasServerNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearServerNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get clientNonce => $_getN(4);
  @$pb.TagNumber(5)
  set clientNonce($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasClientNonce() => $_has(4);
  @$pb.TagNumber(5)
  void clearClientNonce() => $_clearField(5);

  @$pb.TagNumber(6)
  ChannelBinding get channelBinding => $_getN(5);
  @$pb.TagNumber(6)
  set channelBinding(ChannelBinding value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasChannelBinding() => $_has(5);
  @$pb.TagNumber(6)
  void clearChannelBinding() => $_clearField(6);
  @$pb.TagNumber(6)
  ChannelBinding ensureChannelBinding() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.List<$core.int> get credentialSha256 => $_getN(6);
  @$pb.TagNumber(7)
  set credentialSha256($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCredentialSha256() => $_has(6);
  @$pb.TagNumber(7)
  void clearCredentialSha256() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.int> get clientPublicKey => $_getN(7);
  @$pb.TagNumber(8)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasClientPublicKey() => $_has(7);
  @$pb.TagNumber(8)
  void clearClientPublicKey() => $_clearField(8);

  @$pb.TagNumber(9)
  AuthOpenKind get openKind => $_getN(8);
  @$pb.TagNumber(9)
  set openKind(AuthOpenKind value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasOpenKind() => $_has(8);
  @$pb.TagNumber(9)
  void clearOpenKind() => $_clearField(9);
}

/// DirectIceCandidate 是 Direct/SSH embedded signaling 使用的 ICE candidate wire contract。
/// candidate 只用于建立 peer，不携带授权材料，也不能成为 Endpoint identity 或 DTLS binding 真值。
class DirectIceCandidate extends $pb.GeneratedMessage {
  factory DirectIceCandidate({
    $core.String? candidate,
    $core.String? sdpMid,
    $core.int? sdpMlineIndex,
    $core.String? usernameFragment,
  }) {
    final result = create();
    if (candidate != null) result.candidate = candidate;
    if (sdpMid != null) result.sdpMid = sdpMid;
    if (sdpMlineIndex != null) result.sdpMlineIndex = sdpMlineIndex;
    if (usernameFragment != null) result.usernameFragment = usernameFragment;
    return result;
  }

  DirectIceCandidate._();

  factory DirectIceCandidate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectIceCandidate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectIceCandidate',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'candidate')
    ..aOS(2, _omitFieldNames ? '' : 'sdpMid')
    ..aI(3, _omitFieldNames ? '' : 'sdpMlineIndex',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(4, _omitFieldNames ? '' : 'usernameFragment')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectIceCandidate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectIceCandidate copyWith(void Function(DirectIceCandidate) updates) =>
      super.copyWith((message) => updates(message as DirectIceCandidate))
          as DirectIceCandidate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectIceCandidate create() => DirectIceCandidate._();
  @$core.override
  DirectIceCandidate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectIceCandidate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DirectIceCandidate>(create);
  static DirectIceCandidate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get candidate => $_getSZ(0);
  @$pb.TagNumber(1)
  set candidate($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCandidate() => $_has(0);
  @$pb.TagNumber(1)
  void clearCandidate() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sdpMid => $_getSZ(1);
  @$pb.TagNumber(2)
  set sdpMid($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSdpMid() => $_has(1);
  @$pb.TagNumber(2)
  void clearSdpMid() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get sdpMlineIndex => $_getIZ(2);
  @$pb.TagNumber(3)
  set sdpMlineIndex($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSdpMlineIndex() => $_has(2);
  @$pb.TagNumber(3)
  void clearSdpMlineIndex() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get usernameFragment => $_getSZ(3);
  @$pb.TagNumber(4)
  set usernameFragment($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUsernameFragment() => $_has(3);
  @$pb.TagNumber(4)
  void clearUsernameFragment() => $_clearField(4);
}

/// DirectSignalingRequestV2 是客户端向 daemon embedded signaling 提交的一次性短期 offer。
/// request_id 由客户端安全随机生成；daemon 必须在有效期内原子消费，并在创建 PeerConnection 前校验 Endpoint pin。
class DirectSignalingRequestV2 extends $pb.GeneratedMessage {
  factory DirectSignalingRequestV2({
    $core.int? schemaVersion,
    $core.String? requestId,
    $core.String? expectedDeviceId,
    $core.String? expectedDeviceFingerprint,
    $core.String? offerSdp,
    $fixnum.Int64? issuedAtUnixNano,
    $fixnum.Int64? expiresAtUnixNano,
    $core.String? grantId,
    $fixnum.Int64? grantExpiresAtUnixNano,
    $core.List<$core.int>? pairingClaimDigest,
    $core.List<$core.int>? pairingClientPublicKey,
    $fixnum.Int64? pairingExpiresAtUnixNano,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (requestId != null) result.requestId = requestId;
    if (expectedDeviceId != null) result.expectedDeviceId = expectedDeviceId;
    if (expectedDeviceFingerprint != null)
      result.expectedDeviceFingerprint = expectedDeviceFingerprint;
    if (offerSdp != null) result.offerSdp = offerSdp;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (grantId != null) result.grantId = grantId;
    if (grantExpiresAtUnixNano != null)
      result.grantExpiresAtUnixNano = grantExpiresAtUnixNano;
    if (pairingClaimDigest != null)
      result.pairingClaimDigest = pairingClaimDigest;
    if (pairingClientPublicKey != null)
      result.pairingClientPublicKey = pairingClientPublicKey;
    if (pairingExpiresAtUnixNano != null)
      result.pairingExpiresAtUnixNano = pairingExpiresAtUnixNano;
    return result;
  }

  DirectSignalingRequestV2._();

  factory DirectSignalingRequestV2.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectSignalingRequestV2.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectSignalingRequestV2',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..aOS(3, _omitFieldNames ? '' : 'expectedDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'expectedDeviceFingerprint')
    ..aOS(5, _omitFieldNames ? '' : 'offerSdp')
    ..aInt64(6, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..aInt64(7, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..aOS(8, _omitFieldNames ? '' : 'grantId')
    ..aInt64(9, _omitFieldNames ? '' : 'grantExpiresAtUnixNano')
    ..a<$core.List<$core.int>>(
        10, _omitFieldNames ? '' : 'pairingClaimDigest', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        11, _omitFieldNames ? '' : 'pairingClientPublicKey', $pb.PbFieldType.OY)
    ..aInt64(12, _omitFieldNames ? '' : 'pairingExpiresAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingRequestV2 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingRequestV2 copyWith(
          void Function(DirectSignalingRequestV2) updates) =>
      super.copyWith((message) => updates(message as DirectSignalingRequestV2))
          as DirectSignalingRequestV2;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectSignalingRequestV2 create() => DirectSignalingRequestV2._();
  @$core.override
  DirectSignalingRequestV2 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectSignalingRequestV2 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DirectSignalingRequestV2>(create);
  static DirectSignalingRequestV2? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get expectedDeviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set expectedDeviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpectedDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get expectedDeviceFingerprint => $_getSZ(3);
  @$pb.TagNumber(4)
  set expectedDeviceFingerprint($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasExpectedDeviceFingerprint() => $_has(3);
  @$pb.TagNumber(4)
  void clearExpectedDeviceFingerprint() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get offerSdp => $_getSZ(4);
  @$pb.TagNumber(5)
  set offerSdp($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOfferSdp() => $_has(4);
  @$pb.TagNumber(5)
  void clearOfferSdp() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIssuedAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearIssuedAtUnixNano() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(6);
  @$pb.TagNumber(7)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasExpiresAtUnixNano() => $_has(6);
  @$pb.TagNumber(7)
  void clearExpiresAtUnixNano() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get grantId => $_getSZ(7);
  @$pb.TagNumber(8)
  set grantId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasGrantId() => $_has(7);
  @$pb.TagNumber(8)
  void clearGrantId() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get grantExpiresAtUnixNano => $_getI64(8);
  @$pb.TagNumber(9)
  set grantExpiresAtUnixNano($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasGrantExpiresAtUnixNano() => $_has(8);
  @$pb.TagNumber(9)
  void clearGrantExpiresAtUnixNano() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.List<$core.int> get pairingClaimDigest => $_getN(9);
  @$pb.TagNumber(10)
  set pairingClaimDigest($core.List<$core.int> value) => $_setBytes(9, value);
  @$pb.TagNumber(10)
  $core.bool hasPairingClaimDigest() => $_has(9);
  @$pb.TagNumber(10)
  void clearPairingClaimDigest() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.List<$core.int> get pairingClientPublicKey => $_getN(10);
  @$pb.TagNumber(11)
  set pairingClientPublicKey($core.List<$core.int> value) =>
      $_setBytes(10, value);
  @$pb.TagNumber(11)
  $core.bool hasPairingClientPublicKey() => $_has(10);
  @$pb.TagNumber(11)
  void clearPairingClientPublicKey() => $_clearField(11);

  @$pb.TagNumber(12)
  $fixnum.Int64 get pairingExpiresAtUnixNano => $_getI64(11);
  @$pb.TagNumber(12)
  set pairingExpiresAtUnixNano($fixnum.Int64 value) => $_setInt64(11, value);
  @$pb.TagNumber(12)
  $core.bool hasPairingExpiresAtUnixNano() => $_has(11);
  @$pb.TagNumber(12)
  void clearPairingExpiresAtUnixNano() => $_clearField(12);
}

/// DirectSignalingAnswerV2 是 daemon 对本次 offer 返回的短期签名 answer。
/// signature 覆盖 DirectSignalingAnswerSignatureInput；客户端必须使用 Endpoint pin 对 identity 和签名同时校验。
class DirectSignalingAnswerV2 extends $pb.GeneratedMessage {
  factory DirectSignalingAnswerV2({
    $core.int? schemaVersion,
    $core.String? requestId,
    EndpointDaemonIdentity? identity,
    $core.String? answerSdp,
    $core.Iterable<DirectIceCandidate>? candidates,
    $fixnum.Int64? issuedAtUnixNano,
    $fixnum.Int64? expiresAtUnixNano,
    $core.List<$core.int>? signature,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (requestId != null) result.requestId = requestId;
    if (identity != null) result.identity = identity;
    if (answerSdp != null) result.answerSdp = answerSdp;
    if (candidates != null) result.candidates.addAll(candidates);
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (signature != null) result.signature = signature;
    return result;
  }

  DirectSignalingAnswerV2._();

  factory DirectSignalingAnswerV2.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectSignalingAnswerV2.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectSignalingAnswerV2',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..aOM<EndpointDaemonIdentity>(3, _omitFieldNames ? '' : 'identity',
        subBuilder: EndpointDaemonIdentity.create)
    ..aOS(4, _omitFieldNames ? '' : 'answerSdp')
    ..pPM<DirectIceCandidate>(5, _omitFieldNames ? '' : 'candidates',
        subBuilder: DirectIceCandidate.create)
    ..aInt64(6, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..aInt64(7, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingAnswerV2 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingAnswerV2 copyWith(
          void Function(DirectSignalingAnswerV2) updates) =>
      super.copyWith((message) => updates(message as DirectSignalingAnswerV2))
          as DirectSignalingAnswerV2;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectSignalingAnswerV2 create() => DirectSignalingAnswerV2._();
  @$core.override
  DirectSignalingAnswerV2 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectSignalingAnswerV2 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DirectSignalingAnswerV2>(create);
  static DirectSignalingAnswerV2? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointDaemonIdentity get identity => $_getN(2);
  @$pb.TagNumber(3)
  set identity(EndpointDaemonIdentity value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasIdentity() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdentity() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointDaemonIdentity ensureIdentity() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get answerSdp => $_getSZ(3);
  @$pb.TagNumber(4)
  set answerSdp($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAnswerSdp() => $_has(3);
  @$pb.TagNumber(4)
  void clearAnswerSdp() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<DirectIceCandidate> get candidates => $_getList(4);

  @$pb.TagNumber(6)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIssuedAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearIssuedAtUnixNano() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(6);
  @$pb.TagNumber(7)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasExpiresAtUnixNano() => $_has(6);
  @$pb.TagNumber(7)
  void clearExpiresAtUnixNano() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.int> get signature => $_getN(7);
  @$pb.TagNumber(8)
  set signature($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasSignature() => $_has(7);
  @$pb.TagNumber(8)
  void clearSignature() => $_clearField(8);
}

/// DirectSignalingAnswerSignatureInput 是 daemon DeviceIdentity 对 Direct answer 签名的 canonical protobuf。
/// answer.signature 必须在构造该输入前清空；deterministic protobuf bytes 是唯一签名输入。
class DirectSignalingAnswerSignatureInput extends $pb.GeneratedMessage {
  factory DirectSignalingAnswerSignatureInput({
    $core.String? protocol,
    $core.int? version,
    DirectSignalingAnswerV2? answer,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (answer != null) result.answer = answer;
    return result;
  }

  DirectSignalingAnswerSignatureInput._();

  factory DirectSignalingAnswerSignatureInput.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectSignalingAnswerSignatureInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectSignalingAnswerSignatureInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOM<DirectSignalingAnswerV2>(3, _omitFieldNames ? '' : 'answer',
        subBuilder: DirectSignalingAnswerV2.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingAnswerSignatureInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingAnswerSignatureInput copyWith(
          void Function(DirectSignalingAnswerSignatureInput) updates) =>
      super.copyWith((message) =>
              updates(message as DirectSignalingAnswerSignatureInput))
          as DirectSignalingAnswerSignatureInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectSignalingAnswerSignatureInput create() =>
      DirectSignalingAnswerSignatureInput._();
  @$core.override
  DirectSignalingAnswerSignatureInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectSignalingAnswerSignatureInput getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<
          DirectSignalingAnswerSignatureInput>(create);
  static DirectSignalingAnswerSignatureInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  DirectSignalingAnswerV2 get answer => $_getN(2);
  @$pb.TagNumber(3)
  set answer(DirectSignalingAnswerV2 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasAnswer() => $_has(2);
  @$pb.TagNumber(3)
  void clearAnswer() => $_clearField(3);
  @$pb.TagNumber(3)
  DirectSignalingAnswerV2 ensureAnswer() => $_ensure(2);
}

/// DirectSignalingErrorV2 返回不含内部状态的信令失败；错误响应后当前 TCP connection 必须关闭。
class DirectSignalingErrorV2 extends $pb.GeneratedMessage {
  factory DirectSignalingErrorV2({
    DirectSignalingErrorCode? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  DirectSignalingErrorV2._();

  factory DirectSignalingErrorV2.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectSignalingErrorV2.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectSignalingErrorV2',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aE<DirectSignalingErrorCode>(1, _omitFieldNames ? '' : 'code',
        enumValues: DirectSignalingErrorCode.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingErrorV2 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingErrorV2 copyWith(
          void Function(DirectSignalingErrorV2) updates) =>
      super.copyWith((message) => updates(message as DirectSignalingErrorV2))
          as DirectSignalingErrorV2;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectSignalingErrorV2 create() => DirectSignalingErrorV2._();
  @$core.override
  DirectSignalingErrorV2 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectSignalingErrorV2 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DirectSignalingErrorV2>(create);
  static DirectSignalingErrorV2? _defaultInstance;

  @$pb.TagNumber(1)
  DirectSignalingErrorCode get code => $_getN(0);
  @$pb.TagNumber(1)
  set code(DirectSignalingErrorCode value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

enum DirectSignalingResponseV2_Payload { answer, error, notSet }

/// DirectSignalingResponseV2 是 embedded signaling 每条 TCP connection 唯一允许返回的响应。
class DirectSignalingResponseV2 extends $pb.GeneratedMessage {
  factory DirectSignalingResponseV2({
    DirectSignalingAnswerV2? answer,
    DirectSignalingErrorV2? error,
  }) {
    final result = create();
    if (answer != null) result.answer = answer;
    if (error != null) result.error = error;
    return result;
  }

  DirectSignalingResponseV2._();

  factory DirectSignalingResponseV2.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectSignalingResponseV2.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, DirectSignalingResponseV2_Payload>
      _DirectSignalingResponseV2_PayloadByTag = {
    1: DirectSignalingResponseV2_Payload.answer,
    2: DirectSignalingResponseV2_Payload.error,
    0: DirectSignalingResponseV2_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectSignalingResponseV2',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2])
    ..aOM<DirectSignalingAnswerV2>(1, _omitFieldNames ? '' : 'answer',
        subBuilder: DirectSignalingAnswerV2.create)
    ..aOM<DirectSignalingErrorV2>(2, _omitFieldNames ? '' : 'error',
        subBuilder: DirectSignalingErrorV2.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingResponseV2 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectSignalingResponseV2 copyWith(
          void Function(DirectSignalingResponseV2) updates) =>
      super.copyWith((message) => updates(message as DirectSignalingResponseV2))
          as DirectSignalingResponseV2;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectSignalingResponseV2 create() => DirectSignalingResponseV2._();
  @$core.override
  DirectSignalingResponseV2 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectSignalingResponseV2 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DirectSignalingResponseV2>(create);
  static DirectSignalingResponseV2? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  DirectSignalingResponseV2_Payload whichPayload() =>
      _DirectSignalingResponseV2_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  DirectSignalingAnswerV2 get answer => $_getN(0);
  @$pb.TagNumber(1)
  set answer(DirectSignalingAnswerV2 value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAnswer() => $_has(0);
  @$pb.TagNumber(1)
  void clearAnswer() => $_clearField(1);
  @$pb.TagNumber(1)
  DirectSignalingAnswerV2 ensureAnswer() => $_ensure(0);

  @$pb.TagNumber(2)
  DirectSignalingErrorV2 get error => $_getN(1);
  @$pb.TagNumber(2)
  set error(DirectSignalingErrorV2 value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
  @$pb.TagNumber(2)
  DirectSignalingErrorV2 ensureError() => $_ensure(1);
}

/// EndpointDaemonIdentity 是跨来源安全归并 daemon 的唯一锚点。
/// device_public_key 只在 daemon-signed bootstrap 中必填；普通 registry 至少保存 DeviceID 与 fingerprint pin。
class EndpointDaemonIdentity extends $pb.GeneratedMessage {
  factory EndpointDaemonIdentity({
    $core.String? deviceId,
    $core.List<$core.int>? devicePublicKey,
    $core.String? deviceFingerprint,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    return result;
  }

  EndpointDaemonIdentity._();

  factory EndpointDaemonIdentity.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointDaemonIdentity.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointDaemonIdentity',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'deviceFingerprint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDaemonIdentity clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDaemonIdentity copyWith(
          void Function(EndpointDaemonIdentity) updates) =>
      super.copyWith((message) => updates(message as EndpointDaemonIdentity))
          as EndpointDaemonIdentity;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointDaemonIdentity create() => EndpointDaemonIdentity._();
  @$core.override
  EndpointDaemonIdentity createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointDaemonIdentity getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointDaemonIdentity>(create);
  static EndpointDaemonIdentity? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get devicePublicKey => $_getN(1);
  @$pb.TagNumber(2)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDevicePublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearDevicePublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceFingerprint() => $_clearField(3);
}

/// EndpointSelectionPolicy 是客户端本地竞速策略；未配置 route priority 时仍表示 full race。
class EndpointSelectionPolicy extends $pb.GeneratedMessage {
  factory EndpointSelectionPolicy({
    $core.bool? hedgeDelayConfigured,
    $fixnum.Int64? hedgeDelayMillis,
    EndpointRoutePreference? routePreference,
  }) {
    final result = create();
    if (hedgeDelayConfigured != null)
      result.hedgeDelayConfigured = hedgeDelayConfigured;
    if (hedgeDelayMillis != null) result.hedgeDelayMillis = hedgeDelayMillis;
    if (routePreference != null) result.routePreference = routePreference;
    return result;
  }

  EndpointSelectionPolicy._();

  factory EndpointSelectionPolicy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSelectionPolicy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSelectionPolicy',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'hedgeDelayConfigured')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'hedgeDelayMillis', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<EndpointRoutePreference>(3, _omitFieldNames ? '' : 'routePreference',
        enumValues: EndpointRoutePreference.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSelectionPolicy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSelectionPolicy copyWith(
          void Function(EndpointSelectionPolicy) updates) =>
      super.copyWith((message) => updates(message as EndpointSelectionPolicy))
          as EndpointSelectionPolicy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSelectionPolicy create() => EndpointSelectionPolicy._();
  @$core.override
  EndpointSelectionPolicy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSelectionPolicy getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSelectionPolicy>(create);
  static EndpointSelectionPolicy? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get hedgeDelayConfigured => $_getBF(0);
  @$pb.TagNumber(1)
  set hedgeDelayConfigured($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasHedgeDelayConfigured() => $_has(0);
  @$pb.TagNumber(1)
  void clearHedgeDelayConfigured() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get hedgeDelayMillis => $_getI64(1);
  @$pb.TagNumber(2)
  set hedgeDelayMillis($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHedgeDelayMillis() => $_has(1);
  @$pb.TagNumber(2)
  void clearHedgeDelayMillis() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointRoutePreference get routePreference => $_getN(2);
  @$pb.TagNumber(3)
  set routePreference(EndpointRoutePreference value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRoutePreference() => $_has(2);
  @$pb.TagNumber(3)
  void clearRoutePreference() => $_clearField(3);
}

/// EndpointCredentialDescriptor 描述 share 后目标端需要解析或重新创建的凭据，不携带源平台 credential ref 或 secret body。
class EndpointCredentialDescriptor extends $pb.GeneratedMessage {
  factory EndpointCredentialDescriptor({
    $core.String? descriptorId,
    EndpointCredentialKind? kind,
    $core.bool? exportable,
  }) {
    final result = create();
    if (descriptorId != null) result.descriptorId = descriptorId;
    if (kind != null) result.kind = kind;
    if (exportable != null) result.exportable = exportable;
    return result;
  }

  EndpointCredentialDescriptor._();

  factory EndpointCredentialDescriptor.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointCredentialDescriptor.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointCredentialDescriptor',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'descriptorId')
    ..aE<EndpointCredentialKind>(2, _omitFieldNames ? '' : 'kind',
        enumValues: EndpointCredentialKind.values)
    ..aOB(3, _omitFieldNames ? '' : 'exportable')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointCredentialDescriptor clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointCredentialDescriptor copyWith(
          void Function(EndpointCredentialDescriptor) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointCredentialDescriptor))
          as EndpointCredentialDescriptor;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointCredentialDescriptor create() =>
      EndpointCredentialDescriptor._();
  @$core.override
  EndpointCredentialDescriptor createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointCredentialDescriptor getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointCredentialDescriptor>(create);
  static EndpointCredentialDescriptor? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get descriptorId => $_getSZ(0);
  @$pb.TagNumber(1)
  set descriptorId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDescriptorId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDescriptorId() => $_clearField(1);

  @$pb.TagNumber(2)
  EndpointCredentialKind get kind => $_getN(1);
  @$pb.TagNumber(2)
  set kind(EndpointCredentialKind value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get exportable => $_getBF(2);
  @$pb.TagNumber(3)
  set exportable($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExportable() => $_has(2);
  @$pb.TagNumber(3)
  void clearExportable() => $_clearField(3);
}

/// LocalUnixRouteConfig 只用于同一主机上的 Go/native CLI 与 TUI。
class LocalUnixRouteConfig extends $pb.GeneratedMessage {
  factory LocalUnixRouteConfig({
    $core.String? socket,
  }) {
    final result = create();
    if (socket != null) result.socket = socket;
    return result;
  }

  LocalUnixRouteConfig._();

  factory LocalUnixRouteConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LocalUnixRouteConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LocalUnixRouteConfig',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'socket')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalUnixRouteConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalUnixRouteConfig copyWith(void Function(LocalUnixRouteConfig) updates) =>
      super.copyWith((message) => updates(message as LocalUnixRouteConfig))
          as LocalUnixRouteConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LocalUnixRouteConfig create() => LocalUnixRouteConfig._();
  @$core.override
  LocalUnixRouteConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LocalUnixRouteConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LocalUnixRouteConfig>(create);
  static LocalUnixRouteConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get socket => $_getSZ(0);
  @$pb.TagNumber(1)
  set socket($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSocket() => $_has(0);
  @$pb.TagNumber(1)
  void clearSocket() => $_clearField(1);
}

/// DirectWebRTCTCPRouteConfig 描述不依赖 AnyTTY Cloud 的 daemon embedded signaling 与 ICE-TCP locator。
/// advertised_addresses 允许 pair create 为 LAN、FRP 或其它 TCP 映射显式覆盖对外地址，但不改变 Endpoint identity。
class DirectWebRTCTCPRouteConfig extends $pb.GeneratedMessage {
  factory DirectWebRTCTCPRouteConfig({
    $core.Iterable<$core.String>? signalingAddresses,
    $core.Iterable<$core.String>? iceTcpAddresses,
    $core.Iterable<$core.String>? advertisedAddresses,
    $core.String? serverName,
  }) {
    final result = create();
    if (signalingAddresses != null)
      result.signalingAddresses.addAll(signalingAddresses);
    if (iceTcpAddresses != null) result.iceTcpAddresses.addAll(iceTcpAddresses);
    if (advertisedAddresses != null)
      result.advertisedAddresses.addAll(advertisedAddresses);
    if (serverName != null) result.serverName = serverName;
    return result;
  }

  DirectWebRTCTCPRouteConfig._();

  factory DirectWebRTCTCPRouteConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DirectWebRTCTCPRouteConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DirectWebRTCTCPRouteConfig',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'signalingAddresses')
    ..pPS(2, _omitFieldNames ? '' : 'iceTcpAddresses')
    ..pPS(3, _omitFieldNames ? '' : 'advertisedAddresses')
    ..aOS(4, _omitFieldNames ? '' : 'serverName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectWebRTCTCPRouteConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DirectWebRTCTCPRouteConfig copyWith(
          void Function(DirectWebRTCTCPRouteConfig) updates) =>
      super.copyWith(
              (message) => updates(message as DirectWebRTCTCPRouteConfig))
          as DirectWebRTCTCPRouteConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DirectWebRTCTCPRouteConfig create() => DirectWebRTCTCPRouteConfig._();
  @$core.override
  DirectWebRTCTCPRouteConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DirectWebRTCTCPRouteConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DirectWebRTCTCPRouteConfig>(create);
  static DirectWebRTCTCPRouteConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get signalingAddresses => $_getList(0);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get iceTcpAddresses => $_getList(1);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get advertisedAddresses => $_getList(2);

  @$pb.TagNumber(4)
  $core.String get serverName => $_getSZ(3);
  @$pb.TagNumber(4)
  set serverName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasServerName() => $_has(3);
  @$pb.TagNumber(4)
  void clearServerName() => $_clearField(4);
}

/// SSHWebRTCTCPRouteConfig 描述 Go SSH direct-tcpip tunnel 需要的 portable 配置。
/// credential_descriptor 只说明目标平台要解析的凭据类别；credential body 和源平台 ref 永远不得进入 wire contract。
class SSHWebRTCTCPRouteConfig extends $pb.GeneratedMessage {
  factory SSHWebRTCTCPRouteConfig({
    $core.String? host,
    $core.int? port,
    $core.String? user,
    $core.Iterable<$core.String>? hostKeyFingerprints,
    $core.String? proxyJump,
    EndpointCredentialDescriptor? credentialDescriptor,
    $core.String? remoteSignalingAddress,
    $core.String? remoteIceTcpAddress,
    $core.String? sshCredentialRef,
  }) {
    final result = create();
    if (host != null) result.host = host;
    if (port != null) result.port = port;
    if (user != null) result.user = user;
    if (hostKeyFingerprints != null)
      result.hostKeyFingerprints.addAll(hostKeyFingerprints);
    if (proxyJump != null) result.proxyJump = proxyJump;
    if (credentialDescriptor != null)
      result.credentialDescriptor = credentialDescriptor;
    if (remoteSignalingAddress != null)
      result.remoteSignalingAddress = remoteSignalingAddress;
    if (remoteIceTcpAddress != null)
      result.remoteIceTcpAddress = remoteIceTcpAddress;
    if (sshCredentialRef != null) result.sshCredentialRef = sshCredentialRef;
    return result;
  }

  SSHWebRTCTCPRouteConfig._();

  factory SSHWebRTCTCPRouteConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHWebRTCTCPRouteConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHWebRTCTCPRouteConfig',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'host')
    ..aI(2, _omitFieldNames ? '' : 'port', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'user')
    ..pPS(4, _omitFieldNames ? '' : 'hostKeyFingerprints')
    ..aOS(5, _omitFieldNames ? '' : 'proxyJump')
    ..aOM<EndpointCredentialDescriptor>(
        6, _omitFieldNames ? '' : 'credentialDescriptor',
        subBuilder: EndpointCredentialDescriptor.create)
    ..aOS(7, _omitFieldNames ? '' : 'remoteSignalingAddress')
    ..aOS(8, _omitFieldNames ? '' : 'remoteIceTcpAddress')
    ..aOS(9, _omitFieldNames ? '' : 'sshCredentialRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHWebRTCTCPRouteConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHWebRTCTCPRouteConfig copyWith(
          void Function(SSHWebRTCTCPRouteConfig) updates) =>
      super.copyWith((message) => updates(message as SSHWebRTCTCPRouteConfig))
          as SSHWebRTCTCPRouteConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHWebRTCTCPRouteConfig create() => SSHWebRTCTCPRouteConfig._();
  @$core.override
  SSHWebRTCTCPRouteConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHWebRTCTCPRouteConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHWebRTCTCPRouteConfig>(create);
  static SSHWebRTCTCPRouteConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get host => $_getSZ(0);
  @$pb.TagNumber(1)
  set host($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasHost() => $_has(0);
  @$pb.TagNumber(1)
  void clearHost() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get port => $_getIZ(1);
  @$pb.TagNumber(2)
  set port($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPort() => $_has(1);
  @$pb.TagNumber(2)
  void clearPort() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get user => $_getSZ(2);
  @$pb.TagNumber(3)
  set user($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUser() => $_has(2);
  @$pb.TagNumber(3)
  void clearUser() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get hostKeyFingerprints => $_getList(3);

  @$pb.TagNumber(5)
  $core.String get proxyJump => $_getSZ(4);
  @$pb.TagNumber(5)
  set proxyJump($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProxyJump() => $_has(4);
  @$pb.TagNumber(5)
  void clearProxyJump() => $_clearField(5);

  @$pb.TagNumber(6)
  EndpointCredentialDescriptor get credentialDescriptor => $_getN(5);
  @$pb.TagNumber(6)
  set credentialDescriptor(EndpointCredentialDescriptor value) =>
      $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCredentialDescriptor() => $_has(5);
  @$pb.TagNumber(6)
  void clearCredentialDescriptor() => $_clearField(6);
  @$pb.TagNumber(6)
  EndpointCredentialDescriptor ensureCredentialDescriptor() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.String get remoteSignalingAddress => $_getSZ(6);
  @$pb.TagNumber(7)
  set remoteSignalingAddress($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRemoteSignalingAddress() => $_has(6);
  @$pb.TagNumber(7)
  void clearRemoteSignalingAddress() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get remoteIceTcpAddress => $_getSZ(7);
  @$pb.TagNumber(8)
  set remoteIceTcpAddress($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRemoteIceTcpAddress() => $_has(7);
  @$pb.TagNumber(8)
  void clearRemoteIceTcpAddress() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get sshCredentialRef => $_getSZ(8);
  @$pb.TagNumber(9)
  set sshCredentialRef($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasSshCredentialRef() => $_has(8);
  @$pb.TagNumber(9)
  void clearSshCredentialRef() => $_clearField(9);
}

/// ManagedWebRTCRouteConfig 描述同一个 App 内由 AnyTTY Cloud 提供的可选 managed Route。
class ManagedWebRTCRouteConfig extends $pb.GeneratedMessage {
  factory ManagedWebRTCRouteConfig({
    $core.String? targetDeviceId,
    $core.String? accountProfileRef,
    ManagedWebRTCRelayMode? relayMode,
    ManagedWebRTCRelayTransport? relayTransport,
  }) {
    final result = create();
    if (targetDeviceId != null) result.targetDeviceId = targetDeviceId;
    if (accountProfileRef != null) result.accountProfileRef = accountProfileRef;
    if (relayMode != null) result.relayMode = relayMode;
    if (relayTransport != null) result.relayTransport = relayTransport;
    return result;
  }

  ManagedWebRTCRouteConfig._();

  factory ManagedWebRTCRouteConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ManagedWebRTCRouteConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ManagedWebRTCRouteConfig',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'targetDeviceId')
    ..aOS(2, _omitFieldNames ? '' : 'accountProfileRef')
    ..aE<ManagedWebRTCRelayMode>(3, _omitFieldNames ? '' : 'relayMode',
        enumValues: ManagedWebRTCRelayMode.values)
    ..aE<ManagedWebRTCRelayTransport>(
        4, _omitFieldNames ? '' : 'relayTransport',
        enumValues: ManagedWebRTCRelayTransport.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ManagedWebRTCRouteConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ManagedWebRTCRouteConfig copyWith(
          void Function(ManagedWebRTCRouteConfig) updates) =>
      super.copyWith((message) => updates(message as ManagedWebRTCRouteConfig))
          as ManagedWebRTCRouteConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ManagedWebRTCRouteConfig create() => ManagedWebRTCRouteConfig._();
  @$core.override
  ManagedWebRTCRouteConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ManagedWebRTCRouteConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ManagedWebRTCRouteConfig>(create);
  static ManagedWebRTCRouteConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get targetDeviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set targetDeviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTargetDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTargetDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountProfileRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountProfileRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountProfileRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountProfileRef() => $_clearField(2);

  @$pb.TagNumber(3)
  ManagedWebRTCRelayMode get relayMode => $_getN(2);
  @$pb.TagNumber(3)
  set relayMode(ManagedWebRTCRelayMode value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRelayMode() => $_has(2);
  @$pb.TagNumber(3)
  void clearRelayMode() => $_clearField(3);

  @$pb.TagNumber(4)
  ManagedWebRTCRelayTransport get relayTransport => $_getN(3);
  @$pb.TagNumber(4)
  set relayTransport(ManagedWebRTCRelayTransport value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRelayTransport() => $_has(3);
  @$pb.TagNumber(4)
  void clearRelayTransport() => $_clearField(4);
}

enum EndpointRouteConfigV1_Route {
  localUnix,
  directWebrtcTcp,
  sshWebrtcTcp,
  managedWebrtc,
  notSet
}

/// EndpointRouteConfigV1 是跨 Go、JNI、未来 C ABI/WASM 的唯一持久 Route schema。
/// route oneof 让 kind-specific 字段在 schema 层互斥；credential_ref 只引用当前平台 secure store。
class EndpointRouteConfigV1 extends $pb.GeneratedMessage {
  factory EndpointRouteConfigV1({
    $core.int? schemaVersion,
    $core.String? routeId,
    $core.bool? enabled,
    $core.bool? manualOnly,
    $core.int? priority,
    $core.String? credentialRef,
    EndpointSource? source,
    EndpointSource? policySource,
    $core.String? displayName,
    LocalUnixRouteConfig? localUnix,
    DirectWebRTCTCPRouteConfig? directWebrtcTcp,
    SSHWebRTCTCPRouteConfig? sshWebrtcTcp,
    ManagedWebRTCRouteConfig? managedWebrtc,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (routeId != null) result.routeId = routeId;
    if (enabled != null) result.enabled = enabled;
    if (manualOnly != null) result.manualOnly = manualOnly;
    if (priority != null) result.priority = priority;
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (source != null) result.source = source;
    if (policySource != null) result.policySource = policySource;
    if (displayName != null) result.displayName = displayName;
    if (localUnix != null) result.localUnix = localUnix;
    if (directWebrtcTcp != null) result.directWebrtcTcp = directWebrtcTcp;
    if (sshWebrtcTcp != null) result.sshWebrtcTcp = sshWebrtcTcp;
    if (managedWebrtc != null) result.managedWebrtc = managedWebrtc;
    return result;
  }

  EndpointRouteConfigV1._();

  factory EndpointRouteConfigV1.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRouteConfigV1.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EndpointRouteConfigV1_Route>
      _EndpointRouteConfigV1_RouteByTag = {
    20: EndpointRouteConfigV1_Route.localUnix,
    21: EndpointRouteConfigV1_Route.directWebrtcTcp,
    22: EndpointRouteConfigV1_Route.sshWebrtcTcp,
    23: EndpointRouteConfigV1_Route.managedWebrtc,
    0: EndpointRouteConfigV1_Route.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRouteConfigV1',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [20, 21, 22, 23])
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'routeId')
    ..aOB(3, _omitFieldNames ? '' : 'enabled')
    ..aOB(4, _omitFieldNames ? '' : 'manualOnly')
    ..aI(5, _omitFieldNames ? '' : 'priority')
    ..aOS(6, _omitFieldNames ? '' : 'credentialRef')
    ..aE<EndpointSource>(7, _omitFieldNames ? '' : 'source',
        enumValues: EndpointSource.values)
    ..aE<EndpointSource>(8, _omitFieldNames ? '' : 'policySource',
        enumValues: EndpointSource.values)
    ..aOS(9, _omitFieldNames ? '' : 'displayName')
    ..aOM<LocalUnixRouteConfig>(20, _omitFieldNames ? '' : 'localUnix',
        subBuilder: LocalUnixRouteConfig.create)
    ..aOM<DirectWebRTCTCPRouteConfig>(
        21, _omitFieldNames ? '' : 'directWebrtcTcp',
        subBuilder: DirectWebRTCTCPRouteConfig.create)
    ..aOM<SSHWebRTCTCPRouteConfig>(22, _omitFieldNames ? '' : 'sshWebrtcTcp',
        subBuilder: SSHWebRTCTCPRouteConfig.create)
    ..aOM<ManagedWebRTCRouteConfig>(23, _omitFieldNames ? '' : 'managedWebrtc',
        subBuilder: ManagedWebRTCRouteConfig.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRouteConfigV1 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRouteConfigV1 copyWith(
          void Function(EndpointRouteConfigV1) updates) =>
      super.copyWith((message) => updates(message as EndpointRouteConfigV1))
          as EndpointRouteConfigV1;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRouteConfigV1 create() => EndpointRouteConfigV1._();
  @$core.override
  EndpointRouteConfigV1 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRouteConfigV1 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRouteConfigV1>(create);
  static EndpointRouteConfigV1? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  EndpointRouteConfigV1_Route whichRoute() =>
      _EndpointRouteConfigV1_RouteByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  void clearRoute() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get routeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set routeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRouteId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRouteId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get enabled => $_getBF(2);
  @$pb.TagNumber(3)
  set enabled($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEnabled() => $_has(2);
  @$pb.TagNumber(3)
  void clearEnabled() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get manualOnly => $_getBF(3);
  @$pb.TagNumber(4)
  set manualOnly($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasManualOnly() => $_has(3);
  @$pb.TagNumber(4)
  void clearManualOnly() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get priority => $_getIZ(4);
  @$pb.TagNumber(5)
  set priority($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPriority() => $_has(4);
  @$pb.TagNumber(5)
  void clearPriority() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get credentialRef => $_getSZ(5);
  @$pb.TagNumber(6)
  set credentialRef($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCredentialRef() => $_has(5);
  @$pb.TagNumber(6)
  void clearCredentialRef() => $_clearField(6);

  @$pb.TagNumber(7)
  EndpointSource get source => $_getN(6);
  @$pb.TagNumber(7)
  set source(EndpointSource value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSource() => $_has(6);
  @$pb.TagNumber(7)
  void clearSource() => $_clearField(7);

  @$pb.TagNumber(8)
  EndpointSource get policySource => $_getN(7);
  @$pb.TagNumber(8)
  set policySource(EndpointSource value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasPolicySource() => $_has(7);
  @$pb.TagNumber(8)
  void clearPolicySource() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get displayName => $_getSZ(8);
  @$pb.TagNumber(9)
  set displayName($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasDisplayName() => $_has(8);
  @$pb.TagNumber(9)
  void clearDisplayName() => $_clearField(9);

  @$pb.TagNumber(20)
  LocalUnixRouteConfig get localUnix => $_getN(9);
  @$pb.TagNumber(20)
  set localUnix(LocalUnixRouteConfig value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasLocalUnix() => $_has(9);
  @$pb.TagNumber(20)
  void clearLocalUnix() => $_clearField(20);
  @$pb.TagNumber(20)
  LocalUnixRouteConfig ensureLocalUnix() => $_ensure(9);

  @$pb.TagNumber(21)
  DirectWebRTCTCPRouteConfig get directWebrtcTcp => $_getN(10);
  @$pb.TagNumber(21)
  set directWebrtcTcp(DirectWebRTCTCPRouteConfig value) =>
      $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasDirectWebrtcTcp() => $_has(10);
  @$pb.TagNumber(21)
  void clearDirectWebrtcTcp() => $_clearField(21);
  @$pb.TagNumber(21)
  DirectWebRTCTCPRouteConfig ensureDirectWebrtcTcp() => $_ensure(10);

  @$pb.TagNumber(22)
  SSHWebRTCTCPRouteConfig get sshWebrtcTcp => $_getN(11);
  @$pb.TagNumber(22)
  set sshWebrtcTcp(SSHWebRTCTCPRouteConfig value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasSshWebrtcTcp() => $_has(11);
  @$pb.TagNumber(22)
  void clearSshWebrtcTcp() => $_clearField(22);
  @$pb.TagNumber(22)
  SSHWebRTCTCPRouteConfig ensureSshWebrtcTcp() => $_ensure(11);

  @$pb.TagNumber(23)
  ManagedWebRTCRouteConfig get managedWebrtc => $_getN(12);
  @$pb.TagNumber(23)
  set managedWebrtc(ManagedWebRTCRouteConfig value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasManagedWebrtc() => $_has(12);
  @$pb.TagNumber(23)
  void clearManagedWebrtc() => $_clearField(23);
  @$pb.TagNumber(23)
  ManagedWebRTCRouteConfig ensureManagedWebrtc() => $_ensure(12);
}

/// PairingDirectRouteSeed 只携带建立首个 Direct pairing DataChannel 必需的单个公开 locator。
/// 完整 Route 配置在 claim 兑换后由 owning daemon 的签名 bundle 提供。
class PairingDirectRouteSeed extends $pb.GeneratedMessage {
  factory PairingDirectRouteSeed({
    $core.String? signalingAddress,
    $core.String? iceTcpAddress,
    $core.String? serverName,
  }) {
    final result = create();
    if (signalingAddress != null) result.signalingAddress = signalingAddress;
    if (iceTcpAddress != null) result.iceTcpAddress = iceTcpAddress;
    if (serverName != null) result.serverName = serverName;
    return result;
  }

  PairingDirectRouteSeed._();

  factory PairingDirectRouteSeed.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingDirectRouteSeed.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingDirectRouteSeed',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'signalingAddress')
    ..aOS(2, _omitFieldNames ? '' : 'iceTcpAddress')
    ..aOS(3, _omitFieldNames ? '' : 'serverName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingDirectRouteSeed clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingDirectRouteSeed copyWith(
          void Function(PairingDirectRouteSeed) updates) =>
      super.copyWith((message) => updates(message as PairingDirectRouteSeed))
          as PairingDirectRouteSeed;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingDirectRouteSeed create() => PairingDirectRouteSeed._();
  @$core.override
  PairingDirectRouteSeed createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingDirectRouteSeed getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingDirectRouteSeed>(create);
  static PairingDirectRouteSeed? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get signalingAddress => $_getSZ(0);
  @$pb.TagNumber(1)
  set signalingAddress($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSignalingAddress() => $_has(0);
  @$pb.TagNumber(1)
  void clearSignalingAddress() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get iceTcpAddress => $_getSZ(1);
  @$pb.TagNumber(2)
  set iceTcpAddress($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIceTcpAddress() => $_has(1);
  @$pb.TagNumber(2)
  void clearIceTcpAddress() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get serverName => $_getSZ(2);
  @$pb.TagNumber(3)
  set serverName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasServerName() => $_has(2);
  @$pb.TagNumber(3)
  void clearServerName() => $_clearField(3);
}

/// PairingManagedRouteSeed 只携带首次直连 owning Edge 所需的公开入口和 CA pin。
/// 完整 CA、EdgeLocator 与长期 RouteGrant 只在端到端 PairingAccepted 中返回。
class PairingManagedRouteSeed extends $pb.GeneratedMessage {
  factory PairingManagedRouteSeed({
    $core.String? daemonId,
    $core.String? edgeId,
    $core.String? publicEndpoint,
    $core.String? serverName,
    $core.List<$core.int>? caCertificateDerSha256,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (edgeId != null) result.edgeId = edgeId;
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    if (serverName != null) result.serverName = serverName;
    if (caCertificateDerSha256 != null)
      result.caCertificateDerSha256 = caCertificateDerSha256;
    return result;
  }

  PairingManagedRouteSeed._();

  factory PairingManagedRouteSeed.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingManagedRouteSeed.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingManagedRouteSeed',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'edgeId')
    ..aOS(3, _omitFieldNames ? '' : 'publicEndpoint')
    ..aOS(4, _omitFieldNames ? '' : 'serverName')
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'caCertificateDerSha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingManagedRouteSeed clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingManagedRouteSeed copyWith(
          void Function(PairingManagedRouteSeed) updates) =>
      super.copyWith((message) => updates(message as PairingManagedRouteSeed))
          as PairingManagedRouteSeed;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingManagedRouteSeed create() => PairingManagedRouteSeed._();
  @$core.override
  PairingManagedRouteSeed createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingManagedRouteSeed getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingManagedRouteSeed>(create);
  static PairingManagedRouteSeed? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get edgeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set edgeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEdgeId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEdgeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get publicEndpoint => $_getSZ(2);
  @$pb.TagNumber(3)
  set publicEndpoint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPublicEndpoint() => $_has(2);
  @$pb.TagNumber(3)
  void clearPublicEndpoint() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get serverName => $_getSZ(3);
  @$pb.TagNumber(4)
  set serverName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasServerName() => $_has(3);
  @$pb.TagNumber(4)
  void clearServerName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get caCertificateDerSha256 => $_getN(4);
  @$pb.TagNumber(5)
  set caCertificateDerSha256($core.List<$core.int> value) =>
      $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCaCertificateDerSha256() => $_has(4);
  @$pb.TagNumber(5)
  void clearCaCertificateDerSha256() => $_clearField(5);
}

/// PairingSSHRouteSeed 只携带建立首个 SSH tunnel pairing peer 所需的公开连接信息。
/// credential_kind 只声明接收平台需要准备的凭据类别；二维码不得携带 credential ref、密码或私钥。
class PairingSSHRouteSeed extends $pb.GeneratedMessage {
  factory PairingSSHRouteSeed({
    $core.String? host,
    $core.int? port,
    $core.String? user,
    $core.Iterable<$core.String>? hostKeyFingerprints,
    $core.String? proxyJump,
    $core.String? remoteSignalingAddress,
    $core.String? remoteIceTcpAddress,
    EndpointCredentialKind? credentialKind,
  }) {
    final result = create();
    if (host != null) result.host = host;
    if (port != null) result.port = port;
    if (user != null) result.user = user;
    if (hostKeyFingerprints != null)
      result.hostKeyFingerprints.addAll(hostKeyFingerprints);
    if (proxyJump != null) result.proxyJump = proxyJump;
    if (remoteSignalingAddress != null)
      result.remoteSignalingAddress = remoteSignalingAddress;
    if (remoteIceTcpAddress != null)
      result.remoteIceTcpAddress = remoteIceTcpAddress;
    if (credentialKind != null) result.credentialKind = credentialKind;
    return result;
  }

  PairingSSHRouteSeed._();

  factory PairingSSHRouteSeed.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingSSHRouteSeed.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingSSHRouteSeed',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'host')
    ..aI(2, _omitFieldNames ? '' : 'port', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'user')
    ..pPS(4, _omitFieldNames ? '' : 'hostKeyFingerprints')
    ..aOS(5, _omitFieldNames ? '' : 'proxyJump')
    ..aOS(6, _omitFieldNames ? '' : 'remoteSignalingAddress')
    ..aOS(7, _omitFieldNames ? '' : 'remoteIceTcpAddress')
    ..aE<EndpointCredentialKind>(8, _omitFieldNames ? '' : 'credentialKind',
        enumValues: EndpointCredentialKind.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingSSHRouteSeed clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingSSHRouteSeed copyWith(void Function(PairingSSHRouteSeed) updates) =>
      super.copyWith((message) => updates(message as PairingSSHRouteSeed))
          as PairingSSHRouteSeed;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingSSHRouteSeed create() => PairingSSHRouteSeed._();
  @$core.override
  PairingSSHRouteSeed createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingSSHRouteSeed getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingSSHRouteSeed>(create);
  static PairingSSHRouteSeed? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get host => $_getSZ(0);
  @$pb.TagNumber(1)
  set host($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasHost() => $_has(0);
  @$pb.TagNumber(1)
  void clearHost() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get port => $_getIZ(1);
  @$pb.TagNumber(2)
  set port($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPort() => $_has(1);
  @$pb.TagNumber(2)
  void clearPort() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get user => $_getSZ(2);
  @$pb.TagNumber(3)
  set user($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUser() => $_has(2);
  @$pb.TagNumber(3)
  void clearUser() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get hostKeyFingerprints => $_getList(3);

  @$pb.TagNumber(5)
  $core.String get proxyJump => $_getSZ(4);
  @$pb.TagNumber(5)
  set proxyJump($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProxyJump() => $_has(4);
  @$pb.TagNumber(5)
  void clearProxyJump() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get remoteSignalingAddress => $_getSZ(5);
  @$pb.TagNumber(6)
  set remoteSignalingAddress($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRemoteSignalingAddress() => $_has(5);
  @$pb.TagNumber(6)
  void clearRemoteSignalingAddress() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get remoteIceTcpAddress => $_getSZ(6);
  @$pb.TagNumber(7)
  set remoteIceTcpAddress($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRemoteIceTcpAddress() => $_has(6);
  @$pb.TagNumber(7)
  void clearRemoteIceTcpAddress() => $_clearField(7);

  @$pb.TagNumber(8)
  EndpointCredentialKind get credentialKind => $_getN(7);
  @$pb.TagNumber(8)
  set credentialKind(EndpointCredentialKind value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasCredentialKind() => $_has(7);
  @$pb.TagNumber(8)
  void clearCredentialKind() => $_clearField(8);
}

enum PairingRouteSeed_Route {
  directWebrtcTcp,
  managedWebrtc,
  sshWebrtcTcp,
  notSet
}

/// PairingRouteSeed 是短码中建立一次 pairing peer 所需的最小 Route 投影。
class PairingRouteSeed extends $pb.GeneratedMessage {
  factory PairingRouteSeed({
    PairingDirectRouteSeed? directWebrtcTcp,
    PairingManagedRouteSeed? managedWebrtc,
    PairingSSHRouteSeed? sshWebrtcTcp,
    $core.String? routeId,
    $core.String? displayName,
    $core.int? priority,
  }) {
    final result = create();
    if (directWebrtcTcp != null) result.directWebrtcTcp = directWebrtcTcp;
    if (managedWebrtc != null) result.managedWebrtc = managedWebrtc;
    if (sshWebrtcTcp != null) result.sshWebrtcTcp = sshWebrtcTcp;
    if (routeId != null) result.routeId = routeId;
    if (displayName != null) result.displayName = displayName;
    if (priority != null) result.priority = priority;
    return result;
  }

  PairingRouteSeed._();

  factory PairingRouteSeed.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingRouteSeed.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PairingRouteSeed_Route>
      _PairingRouteSeed_RouteByTag = {
    1: PairingRouteSeed_Route.directWebrtcTcp,
    2: PairingRouteSeed_Route.managedWebrtc,
    3: PairingRouteSeed_Route.sshWebrtcTcp,
    0: PairingRouteSeed_Route.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingRouteSeed',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3])
    ..aOM<PairingDirectRouteSeed>(1, _omitFieldNames ? '' : 'directWebrtcTcp',
        subBuilder: PairingDirectRouteSeed.create)
    ..aOM<PairingManagedRouteSeed>(2, _omitFieldNames ? '' : 'managedWebrtc',
        subBuilder: PairingManagedRouteSeed.create)
    ..aOM<PairingSSHRouteSeed>(3, _omitFieldNames ? '' : 'sshWebrtcTcp',
        subBuilder: PairingSSHRouteSeed.create)
    ..aOS(4, _omitFieldNames ? '' : 'routeId')
    ..aOS(5, _omitFieldNames ? '' : 'displayName')
    ..aI(6, _omitFieldNames ? '' : 'priority')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingRouteSeed clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingRouteSeed copyWith(void Function(PairingRouteSeed) updates) =>
      super.copyWith((message) => updates(message as PairingRouteSeed))
          as PairingRouteSeed;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingRouteSeed create() => PairingRouteSeed._();
  @$core.override
  PairingRouteSeed createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingRouteSeed getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingRouteSeed>(create);
  static PairingRouteSeed? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  PairingRouteSeed_Route whichRoute() =>
      _PairingRouteSeed_RouteByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  void clearRoute() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PairingDirectRouteSeed get directWebrtcTcp => $_getN(0);
  @$pb.TagNumber(1)
  set directWebrtcTcp(PairingDirectRouteSeed value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDirectWebrtcTcp() => $_has(0);
  @$pb.TagNumber(1)
  void clearDirectWebrtcTcp() => $_clearField(1);
  @$pb.TagNumber(1)
  PairingDirectRouteSeed ensureDirectWebrtcTcp() => $_ensure(0);

  @$pb.TagNumber(2)
  PairingManagedRouteSeed get managedWebrtc => $_getN(1);
  @$pb.TagNumber(2)
  set managedWebrtc(PairingManagedRouteSeed value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasManagedWebrtc() => $_has(1);
  @$pb.TagNumber(2)
  void clearManagedWebrtc() => $_clearField(2);
  @$pb.TagNumber(2)
  PairingManagedRouteSeed ensureManagedWebrtc() => $_ensure(1);

  @$pb.TagNumber(3)
  PairingSSHRouteSeed get sshWebrtcTcp => $_getN(2);
  @$pb.TagNumber(3)
  set sshWebrtcTcp(PairingSSHRouteSeed value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSshWebrtcTcp() => $_has(2);
  @$pb.TagNumber(3)
  void clearSshWebrtcTcp() => $_clearField(3);
  @$pb.TagNumber(3)
  PairingSSHRouteSeed ensureSshWebrtcTcp() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get routeId => $_getSZ(3);
  @$pb.TagNumber(4)
  set routeId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRouteId() => $_has(3);
  @$pb.TagNumber(4)
  void clearRouteId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get displayName => $_getSZ(4);
  @$pb.TagNumber(5)
  set displayName($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDisplayName() => $_has(4);
  @$pb.TagNumber(5)
  void clearDisplayName() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get priority => $_getIZ(5);
  @$pb.TagNumber(6)
  set priority($core.int value) => $_setSignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPriority() => $_has(5);
  @$pb.TagNumber(6)
  void clearPriority() => $_clearField(6);
}

/// PairingClaimOffer 是二维码和无摄像头输入使用的紧凑一次性 claim。
/// claim 固定 128-bit，只由 owning daemon 内存持有；消息不包含 PairingTicket、CapabilityGrant、scope 或 terminal 信息。
class PairingClaimOffer extends $pb.GeneratedMessage {
  factory PairingClaimOffer({
    $core.int? schemaVersion,
    $core.List<$core.int>? claim,
    $core.String? deviceId,
    $core.List<$core.int>? devicePublicKey,
    $fixnum.Int64? expiresAtUnixNano,
    $core.Iterable<PairingRouteSeed>? routes,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (claim != null) result.claim = claim;
    if (deviceId != null) result.deviceId = deviceId;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (routes != null) result.routes.addAll(routes);
    return result;
  }

  PairingClaimOffer._();

  factory PairingClaimOffer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingClaimOffer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingClaimOffer',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'claim', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..aInt64(5, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..pPM<PairingRouteSeed>(6, _omitFieldNames ? '' : 'routes',
        subBuilder: PairingRouteSeed.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingClaimOffer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingClaimOffer copyWith(void Function(PairingClaimOffer) updates) =>
      super.copyWith((message) => updates(message as PairingClaimOffer))
          as PairingClaimOffer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingClaimOffer create() => PairingClaimOffer._();
  @$core.override
  PairingClaimOffer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingClaimOffer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingClaimOffer>(create);
  static PairingClaimOffer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get claim => $_getN(1);
  @$pb.TagNumber(2)
  set claim($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasClaim() => $_has(1);
  @$pb.TagNumber(2)
  void clearClaim() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get devicePublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDevicePublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearDevicePublicKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(4);
  @$pb.TagNumber(5)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasExpiresAtUnixNano() => $_has(4);
  @$pb.TagNumber(5)
  void clearExpiresAtUnixNano() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<PairingRouteSeed> get routes => $_getList(5);
}

/// EndpointConfigV1 是 Go Client Engine 持久化和跨 binding 投影的 Endpoint 配置。
class EndpointConfigV1 extends $pb.GeneratedMessage {
  factory EndpointConfigV1({
    $core.int? schemaVersion,
    $core.String? endpointId,
    $core.String? label,
    EndpointSource? labelSource,
    EndpointDaemonIdentity? identity,
    EndpointConnectMode? connectMode,
    $core.bool? enabled,
    EndpointSelectionPolicy? selectionPolicy,
    $core.Iterable<EndpointRouteConfigV1>? routes,
    $core.String? platform,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (endpointId != null) result.endpointId = endpointId;
    if (label != null) result.label = label;
    if (labelSource != null) result.labelSource = labelSource;
    if (identity != null) result.identity = identity;
    if (connectMode != null) result.connectMode = connectMode;
    if (enabled != null) result.enabled = enabled;
    if (selectionPolicy != null) result.selectionPolicy = selectionPolicy;
    if (routes != null) result.routes.addAll(routes);
    if (platform != null) result.platform = platform;
    return result;
  }

  EndpointConfigV1._();

  factory EndpointConfigV1.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointConfigV1.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointConfigV1',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..aOS(3, _omitFieldNames ? '' : 'label')
    ..aE<EndpointSource>(4, _omitFieldNames ? '' : 'labelSource',
        enumValues: EndpointSource.values)
    ..aOM<EndpointDaemonIdentity>(5, _omitFieldNames ? '' : 'identity',
        subBuilder: EndpointDaemonIdentity.create)
    ..aE<EndpointConnectMode>(6, _omitFieldNames ? '' : 'connectMode',
        enumValues: EndpointConnectMode.values)
    ..aOB(7, _omitFieldNames ? '' : 'enabled')
    ..aOM<EndpointSelectionPolicy>(8, _omitFieldNames ? '' : 'selectionPolicy',
        subBuilder: EndpointSelectionPolicy.create)
    ..pPM<EndpointRouteConfigV1>(9, _omitFieldNames ? '' : 'routes',
        subBuilder: EndpointRouteConfigV1.create)
    ..aOS(10, _omitFieldNames ? '' : 'platform')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointConfigV1 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointConfigV1 copyWith(void Function(EndpointConfigV1) updates) =>
      super.copyWith((message) => updates(message as EndpointConfigV1))
          as EndpointConfigV1;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointConfigV1 create() => EndpointConfigV1._();
  @$core.override
  EndpointConfigV1 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointConfigV1 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointConfigV1>(create);
  static EndpointConfigV1? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get label => $_getSZ(2);
  @$pb.TagNumber(3)
  set label($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLabel() => $_has(2);
  @$pb.TagNumber(3)
  void clearLabel() => $_clearField(3);

  @$pb.TagNumber(4)
  EndpointSource get labelSource => $_getN(3);
  @$pb.TagNumber(4)
  set labelSource(EndpointSource value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasLabelSource() => $_has(3);
  @$pb.TagNumber(4)
  void clearLabelSource() => $_clearField(4);

  @$pb.TagNumber(5)
  EndpointDaemonIdentity get identity => $_getN(4);
  @$pb.TagNumber(5)
  set identity(EndpointDaemonIdentity value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasIdentity() => $_has(4);
  @$pb.TagNumber(5)
  void clearIdentity() => $_clearField(5);
  @$pb.TagNumber(5)
  EndpointDaemonIdentity ensureIdentity() => $_ensure(4);

  @$pb.TagNumber(6)
  EndpointConnectMode get connectMode => $_getN(5);
  @$pb.TagNumber(6)
  set connectMode(EndpointConnectMode value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasConnectMode() => $_has(5);
  @$pb.TagNumber(6)
  void clearConnectMode() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get enabled => $_getBF(6);
  @$pb.TagNumber(7)
  set enabled($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasEnabled() => $_has(6);
  @$pb.TagNumber(7)
  void clearEnabled() => $_clearField(7);

  @$pb.TagNumber(8)
  EndpointSelectionPolicy get selectionPolicy => $_getN(7);
  @$pb.TagNumber(8)
  set selectionPolicy(EndpointSelectionPolicy value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasSelectionPolicy() => $_has(7);
  @$pb.TagNumber(8)
  void clearSelectionPolicy() => $_clearField(8);
  @$pb.TagNumber(8)
  EndpointSelectionPolicy ensureSelectionPolicy() => $_ensure(7);

  @$pb.TagNumber(9)
  $pb.PbList<EndpointRouteConfigV1> get routes => $_getList(8);

  /// platform 是 daemon 在配对时声明的规范化 OS family（例如 darwin/linux/windows）。
  @$pb.TagNumber(10)
  $core.String get platform => $_getSZ(9);
  @$pb.TagNumber(10)
  set platform($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasPlatform() => $_has(9);
  @$pb.TagNumber(10)
  void clearPlatform() => $_clearField(10);
}

/// EndpointRegistryV1 是 Go Client Engine 拥有的完整 Endpoint registry contract。
class EndpointRegistryV1 extends $pb.GeneratedMessage {
  factory EndpointRegistryV1({
    $core.int? schemaVersion,
    $core.String? defaultEndpointId,
    $core.Iterable<EndpointConfigV1>? endpoints,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (defaultEndpointId != null) result.defaultEndpointId = defaultEndpointId;
    if (endpoints != null) result.endpoints.addAll(endpoints);
    return result;
  }

  EndpointRegistryV1._();

  factory EndpointRegistryV1.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRegistryV1.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRegistryV1',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'defaultEndpointId')
    ..pPM<EndpointConfigV1>(3, _omitFieldNames ? '' : 'endpoints',
        subBuilder: EndpointConfigV1.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryV1 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryV1 copyWith(void Function(EndpointRegistryV1) updates) =>
      super.copyWith((message) => updates(message as EndpointRegistryV1))
          as EndpointRegistryV1;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRegistryV1 create() => EndpointRegistryV1._();
  @$core.override
  EndpointRegistryV1 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRegistryV1 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRegistryV1>(create);
  static EndpointRegistryV1? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get defaultEndpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set defaultEndpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDefaultEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDefaultEndpointId() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<EndpointConfigV1> get endpoints => $_getList(2);
}

/// PairingTicketDescriptor 是 daemon-local 一次性授权兑换票据的公开部分。
class PairingTicketDescriptor extends $pb.GeneratedMessage {
  factory PairingTicketDescriptor({
    $core.String? ticketId,
    $core.Iterable<$core.String>? scopeCeiling,
    $fixnum.Int64? expiresAtUnixNano,
    $core.List<$core.int>? nonce,
    $core.int? maxRedemptions,
    $core.List<$core.int>? signature,
    $fixnum.Int64? issuedAtUnixNano,
    $fixnum.Int64? grantLifetimeSeconds,
  }) {
    final result = create();
    if (ticketId != null) result.ticketId = ticketId;
    if (scopeCeiling != null) result.scopeCeiling.addAll(scopeCeiling);
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (nonce != null) result.nonce = nonce;
    if (maxRedemptions != null) result.maxRedemptions = maxRedemptions;
    if (signature != null) result.signature = signature;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (grantLifetimeSeconds != null)
      result.grantLifetimeSeconds = grantLifetimeSeconds;
    return result;
  }

  PairingTicketDescriptor._();

  factory PairingTicketDescriptor.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingTicketDescriptor.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingTicketDescriptor',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'ticketId')
    ..pPS(2, _omitFieldNames ? '' : 'scopeCeiling')
    ..aInt64(3, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'nonce', $pb.PbFieldType.OY)
    ..aI(5, _omitFieldNames ? '' : 'maxRedemptions',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..aInt64(7, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..aInt64(8, _omitFieldNames ? '' : 'grantLifetimeSeconds')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingTicketDescriptor clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingTicketDescriptor copyWith(
          void Function(PairingTicketDescriptor) updates) =>
      super.copyWith((message) => updates(message as PairingTicketDescriptor))
          as PairingTicketDescriptor;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingTicketDescriptor create() => PairingTicketDescriptor._();
  @$core.override
  PairingTicketDescriptor createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingTicketDescriptor getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingTicketDescriptor>(create);
  static PairingTicketDescriptor? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get ticketId => $_getSZ(0);
  @$pb.TagNumber(1)
  set ticketId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTicketId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTicketId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get scopeCeiling => $_getList(1);

  @$pb.TagNumber(3)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(2);
  @$pb.TagNumber(3)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpiresAtUnixNano() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpiresAtUnixNano() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get nonce => $_getN(3);
  @$pb.TagNumber(4)
  set nonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get maxRedemptions => $_getIZ(4);
  @$pb.TagNumber(5)
  set maxRedemptions($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMaxRedemptions() => $_has(4);
  @$pb.TagNumber(5)
  void clearMaxRedemptions() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get signature => $_getN(5);
  @$pb.TagNumber(6)
  set signature($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasSignature() => $_has(5);
  @$pb.TagNumber(6)
  void clearSignature() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(6);
  @$pb.TagNumber(7)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasIssuedAtUnixNano() => $_has(6);
  @$pb.TagNumber(7)
  void clearIssuedAtUnixNano() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get grantLifetimeSeconds => $_getI64(7);
  @$pb.TagNumber(8)
  set grantLifetimeSeconds($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasGrantLifetimeSeconds() => $_has(7);
  @$pb.TagNumber(8)
  void clearGrantLifetimeSeconds() => $_clearField(8);
}

enum EndpointAuthorizationBootstrap_Payload {
  pairingTicket,
  boundGrant,
  notSet
}

/// EndpointAuthorizationBootstrap 只允许携带短期 PairingTicket，或双向离线扫码中已绑定接收方 key 的 grant。
class EndpointAuthorizationBootstrap extends $pb.GeneratedMessage {
  factory EndpointAuthorizationBootstrap({
    PairingTicketDescriptor? pairingTicket,
    $core.List<$core.int>? boundGrant,
  }) {
    final result = create();
    if (pairingTicket != null) result.pairingTicket = pairingTicket;
    if (boundGrant != null) result.boundGrant = boundGrant;
    return result;
  }

  EndpointAuthorizationBootstrap._();

  factory EndpointAuthorizationBootstrap.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointAuthorizationBootstrap.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EndpointAuthorizationBootstrap_Payload>
      _EndpointAuthorizationBootstrap_PayloadByTag = {
    1: EndpointAuthorizationBootstrap_Payload.pairingTicket,
    2: EndpointAuthorizationBootstrap_Payload.boundGrant,
    0: EndpointAuthorizationBootstrap_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointAuthorizationBootstrap',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2])
    ..aOM<PairingTicketDescriptor>(1, _omitFieldNames ? '' : 'pairingTicket',
        subBuilder: PairingTicketDescriptor.create)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'boundGrant', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointAuthorizationBootstrap clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointAuthorizationBootstrap copyWith(
          void Function(EndpointAuthorizationBootstrap) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointAuthorizationBootstrap))
          as EndpointAuthorizationBootstrap;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointAuthorizationBootstrap create() =>
      EndpointAuthorizationBootstrap._();
  @$core.override
  EndpointAuthorizationBootstrap createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointAuthorizationBootstrap getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointAuthorizationBootstrap>(create);
  static EndpointAuthorizationBootstrap? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  EndpointAuthorizationBootstrap_Payload whichPayload() =>
      _EndpointAuthorizationBootstrap_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PairingTicketDescriptor get pairingTicket => $_getN(0);
  @$pb.TagNumber(1)
  set pairingTicket(PairingTicketDescriptor value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPairingTicket() => $_has(0);
  @$pb.TagNumber(1)
  void clearPairingTicket() => $_clearField(1);
  @$pb.TagNumber(1)
  PairingTicketDescriptor ensurePairingTicket() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.List<$core.int> get boundGrant => $_getN(1);
  @$pb.TagNumber(2)
  set boundGrant($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBoundGrant() => $_has(1);
  @$pb.TagNumber(2)
  void clearBoundGrant() => $_clearField(2);
}

/// EndpointBootstrapBundleV2 是 daemon DeviceIdentity 签名的增量 bootstrap wire contract。
/// 它不包含客户端 priority、local-unix、Cloud token、Hub/Relay 地址、SSH credential body 或本地 credential ref。
class EndpointBootstrapBundleV2 extends $pb.GeneratedMessage {
  factory EndpointBootstrapBundleV2({
    $core.int? schemaVersion,
    $core.String? bundleId,
    EndpointDaemonIdentity? identity,
    $core.String? suggestedLabel,
    $core.Iterable<EndpointRouteConfigV1>? routes,
    EndpointAuthorizationBootstrap? authorization,
    $fixnum.Int64? issuedAtUnixNano,
    $fixnum.Int64? expiresAtUnixNano,
    $core.List<$core.int>? bundleSignature,
    $core.String? platform,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (bundleId != null) result.bundleId = bundleId;
    if (identity != null) result.identity = identity;
    if (suggestedLabel != null) result.suggestedLabel = suggestedLabel;
    if (routes != null) result.routes.addAll(routes);
    if (authorization != null) result.authorization = authorization;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (bundleSignature != null) result.bundleSignature = bundleSignature;
    if (platform != null) result.platform = platform;
    return result;
  }

  EndpointBootstrapBundleV2._();

  factory EndpointBootstrapBundleV2.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointBootstrapBundleV2.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointBootstrapBundleV2',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'bundleId')
    ..aOM<EndpointDaemonIdentity>(3, _omitFieldNames ? '' : 'identity',
        subBuilder: EndpointDaemonIdentity.create)
    ..aOS(4, _omitFieldNames ? '' : 'suggestedLabel')
    ..pPM<EndpointRouteConfigV1>(5, _omitFieldNames ? '' : 'routes',
        subBuilder: EndpointRouteConfigV1.create)
    ..aOM<EndpointAuthorizationBootstrap>(
        6, _omitFieldNames ? '' : 'authorization',
        subBuilder: EndpointAuthorizationBootstrap.create)
    ..aInt64(7, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..aInt64(8, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..a<$core.List<$core.int>>(
        9, _omitFieldNames ? '' : 'bundleSignature', $pb.PbFieldType.OY)
    ..aOS(10, _omitFieldNames ? '' : 'platform')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointBootstrapBundleV2 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointBootstrapBundleV2 copyWith(
          void Function(EndpointBootstrapBundleV2) updates) =>
      super.copyWith((message) => updates(message as EndpointBootstrapBundleV2))
          as EndpointBootstrapBundleV2;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointBootstrapBundleV2 create() => EndpointBootstrapBundleV2._();
  @$core.override
  EndpointBootstrapBundleV2 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointBootstrapBundleV2 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointBootstrapBundleV2>(create);
  static EndpointBootstrapBundleV2? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get bundleId => $_getSZ(1);
  @$pb.TagNumber(2)
  set bundleId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBundleId() => $_has(1);
  @$pb.TagNumber(2)
  void clearBundleId() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointDaemonIdentity get identity => $_getN(2);
  @$pb.TagNumber(3)
  set identity(EndpointDaemonIdentity value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasIdentity() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdentity() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointDaemonIdentity ensureIdentity() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get suggestedLabel => $_getSZ(3);
  @$pb.TagNumber(4)
  set suggestedLabel($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSuggestedLabel() => $_has(3);
  @$pb.TagNumber(4)
  void clearSuggestedLabel() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<EndpointRouteConfigV1> get routes => $_getList(4);

  @$pb.TagNumber(6)
  EndpointAuthorizationBootstrap get authorization => $_getN(5);
  @$pb.TagNumber(6)
  set authorization(EndpointAuthorizationBootstrap value) =>
      $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasAuthorization() => $_has(5);
  @$pb.TagNumber(6)
  void clearAuthorization() => $_clearField(6);
  @$pb.TagNumber(6)
  EndpointAuthorizationBootstrap ensureAuthorization() => $_ensure(5);

  @$pb.TagNumber(7)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(6);
  @$pb.TagNumber(7)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasIssuedAtUnixNano() => $_has(6);
  @$pb.TagNumber(7)
  void clearIssuedAtUnixNano() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(7);
  @$pb.TagNumber(8)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasExpiresAtUnixNano() => $_has(7);
  @$pb.TagNumber(8)
  void clearExpiresAtUnixNano() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.List<$core.int> get bundleSignature => $_getN(8);
  @$pb.TagNumber(9)
  set bundleSignature($core.List<$core.int> value) => $_setBytes(8, value);
  @$pb.TagNumber(9)
  $core.bool hasBundleSignature() => $_has(8);
  @$pb.TagNumber(9)
  void clearBundleSignature() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get platform => $_getSZ(9);
  @$pb.TagNumber(10)
  set platform($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasPlatform() => $_has(9);
  @$pb.TagNumber(10)
  void clearPlatform() => $_clearField(10);
}

/// PairingTicketSignatureInput 是 DeviceIdentity 对一次性 ticket 签名的唯一 canonical protobuf。
/// ticket.signature 必须在构造本消息前清空；issuer 字段必须与外层 bootstrap identity 完全一致。
class PairingTicketSignatureInput extends $pb.GeneratedMessage {
  factory PairingTicketSignatureInput({
    $core.String? protocol,
    $core.int? version,
    $core.String? issuerDeviceId,
    $core.String? issuerDeviceFingerprint,
    PairingTicketDescriptor? ticket,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (issuerDeviceId != null) result.issuerDeviceId = issuerDeviceId;
    if (issuerDeviceFingerprint != null)
      result.issuerDeviceFingerprint = issuerDeviceFingerprint;
    if (ticket != null) result.ticket = ticket;
    return result;
  }

  PairingTicketSignatureInput._();

  factory PairingTicketSignatureInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingTicketSignatureInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingTicketSignatureInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'issuerDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'issuerDeviceFingerprint')
    ..aOM<PairingTicketDescriptor>(5, _omitFieldNames ? '' : 'ticket',
        subBuilder: PairingTicketDescriptor.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingTicketSignatureInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingTicketSignatureInput copyWith(
          void Function(PairingTicketSignatureInput) updates) =>
      super.copyWith(
              (message) => updates(message as PairingTicketSignatureInput))
          as PairingTicketSignatureInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingTicketSignatureInput create() =>
      PairingTicketSignatureInput._();
  @$core.override
  PairingTicketSignatureInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingTicketSignatureInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingTicketSignatureInput>(create);
  static PairingTicketSignatureInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get issuerDeviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set issuerDeviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIssuerDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearIssuerDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get issuerDeviceFingerprint => $_getSZ(3);
  @$pb.TagNumber(4)
  set issuerDeviceFingerprint($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasIssuerDeviceFingerprint() => $_has(3);
  @$pb.TagNumber(4)
  void clearIssuerDeviceFingerprint() => $_clearField(4);

  @$pb.TagNumber(5)
  PairingTicketDescriptor get ticket => $_getN(4);
  @$pb.TagNumber(5)
  set ticket(PairingTicketDescriptor value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasTicket() => $_has(4);
  @$pb.TagNumber(5)
  void clearTicket() => $_clearField(5);
  @$pb.TagNumber(5)
  PairingTicketDescriptor ensureTicket() => $_ensure(4);
}

/// EndpointBootstrapSignatureInput 是 DeviceIdentity 对完整 bootstrap 签名的唯一 canonical protobuf。
/// bundle.bundle_signature 必须在构造本消息前清空；内部 PairingTicket signature 保持在签名覆盖范围内。
class EndpointBootstrapSignatureInput extends $pb.GeneratedMessage {
  factory EndpointBootstrapSignatureInput({
    $core.String? protocol,
    $core.int? version,
    EndpointBootstrapBundleV2? bundle,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (bundle != null) result.bundle = bundle;
    return result;
  }

  EndpointBootstrapSignatureInput._();

  factory EndpointBootstrapSignatureInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointBootstrapSignatureInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointBootstrapSignatureInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOM<EndpointBootstrapBundleV2>(3, _omitFieldNames ? '' : 'bundle',
        subBuilder: EndpointBootstrapBundleV2.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointBootstrapSignatureInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointBootstrapSignatureInput copyWith(
          void Function(EndpointBootstrapSignatureInput) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointBootstrapSignatureInput))
          as EndpointBootstrapSignatureInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointBootstrapSignatureInput create() =>
      EndpointBootstrapSignatureInput._();
  @$core.override
  EndpointBootstrapSignatureInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointBootstrapSignatureInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointBootstrapSignatureInput>(
          create);
  static EndpointBootstrapSignatureInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointBootstrapBundleV2 get bundle => $_getN(2);
  @$pb.TagNumber(3)
  set bundle(EndpointBootstrapBundleV2 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasBundle() => $_has(2);
  @$pb.TagNumber(3)
  void clearBundle() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointBootstrapBundleV2 ensureBundle() => $_ensure(2);
}

/// ClientEndpointShareBundleV1 是用户确认后通过实时 TLS share session 发送的客户端配置迁移 contract。
/// source EndpointID、runtime、源 credential ref、Cloud token 和源 CapabilityGrant 永远不得进入该消息。
class ClientEndpointShareBundleV1 extends $pb.GeneratedMessage {
  factory ClientEndpointShareBundleV1({
    $core.int? schemaVersion,
    $core.String? transferId,
    EndpointDaemonIdentity? identity,
    $core.String? suggestedLabel,
    $core.Iterable<EndpointRouteConfigV1>? routes,
    EndpointConnectMode? connectMode,
    EndpointSelectionPolicy? selectionPolicy,
    $core.Iterable<EndpointCredentialDescriptor>? credentialDescriptors,
    $core.List<$core.int>? boundGrant,
    $fixnum.Int64? issuedAtUnixNano,
    $fixnum.Int64? expiresAtUnixNano,
    $core.String? platform,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (transferId != null) result.transferId = transferId;
    if (identity != null) result.identity = identity;
    if (suggestedLabel != null) result.suggestedLabel = suggestedLabel;
    if (routes != null) result.routes.addAll(routes);
    if (connectMode != null) result.connectMode = connectMode;
    if (selectionPolicy != null) result.selectionPolicy = selectionPolicy;
    if (credentialDescriptors != null)
      result.credentialDescriptors.addAll(credentialDescriptors);
    if (boundGrant != null) result.boundGrant = boundGrant;
    if (issuedAtUnixNano != null) result.issuedAtUnixNano = issuedAtUnixNano;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (platform != null) result.platform = platform;
    return result;
  }

  ClientEndpointShareBundleV1._();

  factory ClientEndpointShareBundleV1.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientEndpointShareBundleV1.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientEndpointShareBundleV1',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'transferId')
    ..aOM<EndpointDaemonIdentity>(3, _omitFieldNames ? '' : 'identity',
        subBuilder: EndpointDaemonIdentity.create)
    ..aOS(4, _omitFieldNames ? '' : 'suggestedLabel')
    ..pPM<EndpointRouteConfigV1>(5, _omitFieldNames ? '' : 'routes',
        subBuilder: EndpointRouteConfigV1.create)
    ..aE<EndpointConnectMode>(6, _omitFieldNames ? '' : 'connectMode',
        enumValues: EndpointConnectMode.values)
    ..aOM<EndpointSelectionPolicy>(7, _omitFieldNames ? '' : 'selectionPolicy',
        subBuilder: EndpointSelectionPolicy.create)
    ..pPM<EndpointCredentialDescriptor>(
        8, _omitFieldNames ? '' : 'credentialDescriptors',
        subBuilder: EndpointCredentialDescriptor.create)
    ..a<$core.List<$core.int>>(
        9, _omitFieldNames ? '' : 'boundGrant', $pb.PbFieldType.OY)
    ..aInt64(10, _omitFieldNames ? '' : 'issuedAtUnixNano')
    ..aInt64(11, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..aOS(12, _omitFieldNames ? '' : 'platform')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientEndpointShareBundleV1 clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientEndpointShareBundleV1 copyWith(
          void Function(ClientEndpointShareBundleV1) updates) =>
      super.copyWith(
              (message) => updates(message as ClientEndpointShareBundleV1))
          as ClientEndpointShareBundleV1;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientEndpointShareBundleV1 create() =>
      ClientEndpointShareBundleV1._();
  @$core.override
  ClientEndpointShareBundleV1 createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientEndpointShareBundleV1 getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientEndpointShareBundleV1>(create);
  static ClientEndpointShareBundleV1? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get transferId => $_getSZ(1);
  @$pb.TagNumber(2)
  set transferId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTransferId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransferId() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointDaemonIdentity get identity => $_getN(2);
  @$pb.TagNumber(3)
  set identity(EndpointDaemonIdentity value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasIdentity() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdentity() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointDaemonIdentity ensureIdentity() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get suggestedLabel => $_getSZ(3);
  @$pb.TagNumber(4)
  set suggestedLabel($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSuggestedLabel() => $_has(3);
  @$pb.TagNumber(4)
  void clearSuggestedLabel() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<EndpointRouteConfigV1> get routes => $_getList(4);

  @$pb.TagNumber(6)
  EndpointConnectMode get connectMode => $_getN(5);
  @$pb.TagNumber(6)
  set connectMode(EndpointConnectMode value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasConnectMode() => $_has(5);
  @$pb.TagNumber(6)
  void clearConnectMode() => $_clearField(6);

  @$pb.TagNumber(7)
  EndpointSelectionPolicy get selectionPolicy => $_getN(6);
  @$pb.TagNumber(7)
  set selectionPolicy(EndpointSelectionPolicy value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSelectionPolicy() => $_has(6);
  @$pb.TagNumber(7)
  void clearSelectionPolicy() => $_clearField(7);
  @$pb.TagNumber(7)
  EndpointSelectionPolicy ensureSelectionPolicy() => $_ensure(6);

  @$pb.TagNumber(8)
  $pb.PbList<EndpointCredentialDescriptor> get credentialDescriptors =>
      $_getList(7);

  /// bound_grant 预留给 CONN002 生成并验证的目标 ClientAccessIdentity-bound grant；不得放入源客户端 bearer grant。
  @$pb.TagNumber(9)
  $core.List<$core.int> get boundGrant => $_getN(8);
  @$pb.TagNumber(9)
  set boundGrant($core.List<$core.int> value) => $_setBytes(8, value);
  @$pb.TagNumber(9)
  $core.bool hasBoundGrant() => $_has(8);
  @$pb.TagNumber(9)
  void clearBoundGrant() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get issuedAtUnixNano => $_getI64(9);
  @$pb.TagNumber(10)
  set issuedAtUnixNano($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasIssuedAtUnixNano() => $_has(9);
  @$pb.TagNumber(10)
  void clearIssuedAtUnixNano() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(10);
  @$pb.TagNumber(11)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasExpiresAtUnixNano() => $_has(10);
  @$pb.TagNumber(11)
  void clearExpiresAtUnixNano() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get platform => $_getSZ(11);
  @$pb.TagNumber(12)
  set platform($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasPlatform() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlatform() => $_clearField(12);
}

/// ShareSessionOffer 是静态二维码允许包含的全部内容。
/// endpoint 配置、credential、CapabilityGrant 和 Cloud material 只能在用户确认后的实时 TLS channel 内传输。
class ShareSessionOffer extends $pb.GeneratedMessage {
  factory ShareSessionOffer({
    $core.int? schemaVersion,
    $core.String? transferId,
    $core.Iterable<$core.String>? listenerAddresses,
    $core.String? ephemeralCertificateSha256,
    $core.List<$core.int>? oneTimeSessionSecret,
    $fixnum.Int64? expiresAtUnixNano,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (transferId != null) result.transferId = transferId;
    if (listenerAddresses != null)
      result.listenerAddresses.addAll(listenerAddresses);
    if (ephemeralCertificateSha256 != null)
      result.ephemeralCertificateSha256 = ephemeralCertificateSha256;
    if (oneTimeSessionSecret != null)
      result.oneTimeSessionSecret = oneTimeSessionSecret;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    return result;
  }

  ShareSessionOffer._();

  factory ShareSessionOffer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareSessionOffer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareSessionOffer',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'transferId')
    ..pPS(3, _omitFieldNames ? '' : 'listenerAddresses')
    ..aOS(4, _omitFieldNames ? '' : 'ephemeralCertificateSha256')
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'oneTimeSessionSecret', $pb.PbFieldType.OY)
    ..aInt64(6, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionOffer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionOffer copyWith(void Function(ShareSessionOffer) updates) =>
      super.copyWith((message) => updates(message as ShareSessionOffer))
          as ShareSessionOffer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareSessionOffer create() => ShareSessionOffer._();
  @$core.override
  ShareSessionOffer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareSessionOffer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareSessionOffer>(create);
  static ShareSessionOffer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get transferId => $_getSZ(1);
  @$pb.TagNumber(2)
  set transferId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTransferId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransferId() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get listenerAddresses => $_getList(2);

  @$pb.TagNumber(4)
  $core.String get ephemeralCertificateSha256 => $_getSZ(3);
  @$pb.TagNumber(4)
  set ephemeralCertificateSha256($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEphemeralCertificateSha256() => $_has(3);
  @$pb.TagNumber(4)
  void clearEphemeralCertificateSha256() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get oneTimeSessionSecret => $_getN(4);
  @$pb.TagNumber(5)
  set oneTimeSessionSecret($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOneTimeSessionSecret() => $_has(4);
  @$pb.TagNumber(5)
  void clearOneTimeSessionSecret() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasExpiresAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearExpiresAtUnixNano() => $_clearField(6);
}

/// ShareReceiverHello 证明接收方持有静态 offer 中的一次性 secret，并声明本次会话的临时签名公钥。
class ShareReceiverHello extends $pb.GeneratedMessage {
  factory ShareReceiverHello({
    $core.int? schemaVersion,
    $core.String? transferId,
    $core.List<$core.int>? oneTimeSessionSecret,
    $core.List<$core.int>? receiverPublicKey,
    $core.List<$core.int>? receiverNonce,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (transferId != null) result.transferId = transferId;
    if (oneTimeSessionSecret != null)
      result.oneTimeSessionSecret = oneTimeSessionSecret;
    if (receiverPublicKey != null) result.receiverPublicKey = receiverPublicKey;
    if (receiverNonce != null) result.receiverNonce = receiverNonce;
    return result;
  }

  ShareReceiverHello._();

  factory ShareReceiverHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareReceiverHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareReceiverHello',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'transferId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'oneTimeSessionSecret', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'receiverPublicKey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'receiverNonce', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareReceiverHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareReceiverHello copyWith(void Function(ShareReceiverHello) updates) =>
      super.copyWith((message) => updates(message as ShareReceiverHello))
          as ShareReceiverHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareReceiverHello create() => ShareReceiverHello._();
  @$core.override
  ShareReceiverHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareReceiverHello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareReceiverHello>(create);
  static ShareReceiverHello? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get transferId => $_getSZ(1);
  @$pb.TagNumber(2)
  set transferId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTransferId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransferId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get oneTimeSessionSecret => $_getN(2);
  @$pb.TagNumber(3)
  set oneTimeSessionSecret($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOneTimeSessionSecret() => $_has(2);
  @$pb.TagNumber(3)
  void clearOneTimeSessionSecret() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get receiverPublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set receiverPublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReceiverPublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearReceiverPublicKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get receiverNonce => $_getN(4);
  @$pb.TagNumber(5)
  set receiverNonce($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasReceiverNonce() => $_has(4);
  @$pb.TagNumber(5)
  void clearReceiverNonce() => $_clearField(5);
}

/// ShareSenderChallenge 把接收方 nonce、发送方 nonce 和 offer 有效期绑定到同一次 TLS 会话。
class ShareSenderChallenge extends $pb.GeneratedMessage {
  factory ShareSenderChallenge({
    $core.int? schemaVersion,
    $core.String? transferId,
    $core.List<$core.int>? receiverNonce,
    $core.List<$core.int>? senderNonce,
    $fixnum.Int64? expiresAtUnixNano,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (transferId != null) result.transferId = transferId;
    if (receiverNonce != null) result.receiverNonce = receiverNonce;
    if (senderNonce != null) result.senderNonce = senderNonce;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    return result;
  }

  ShareSenderChallenge._();

  factory ShareSenderChallenge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareSenderChallenge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareSenderChallenge',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'transferId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'receiverNonce', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'senderNonce', $pb.PbFieldType.OY)
    ..aInt64(5, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSenderChallenge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSenderChallenge copyWith(void Function(ShareSenderChallenge) updates) =>
      super.copyWith((message) => updates(message as ShareSenderChallenge))
          as ShareSenderChallenge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareSenderChallenge create() => ShareSenderChallenge._();
  @$core.override
  ShareSenderChallenge createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareSenderChallenge getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareSenderChallenge>(create);
  static ShareSenderChallenge? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get transferId => $_getSZ(1);
  @$pb.TagNumber(2)
  set transferId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTransferId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransferId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get receiverNonce => $_getN(2);
  @$pb.TagNumber(3)
  set receiverNonce($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReceiverNonce() => $_has(2);
  @$pb.TagNumber(3)
  void clearReceiverNonce() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get senderNonce => $_getN(3);
  @$pb.TagNumber(4)
  set senderNonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSenderNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearSenderNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(4);
  @$pb.TagNumber(5)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasExpiresAtUnixNano() => $_has(4);
  @$pb.TagNumber(5)
  void clearExpiresAtUnixNano() => $_clearField(5);
}

/// ShareReceiverProofInput 是 receiver proof 的 deterministic protobuf 签名输入。
class ShareReceiverProofInput extends $pb.GeneratedMessage {
  factory ShareReceiverProofInput({
    $core.String? protocol,
    $core.int? version,
    $core.String? transferId,
    $core.List<$core.int>? receiverNonce,
    $core.List<$core.int>? senderNonce,
    $core.String? ephemeralCertificateSha256,
    $core.List<$core.int>? oneTimeSessionSecretSha256,
  }) {
    final result = create();
    if (protocol != null) result.protocol = protocol;
    if (version != null) result.version = version;
    if (transferId != null) result.transferId = transferId;
    if (receiverNonce != null) result.receiverNonce = receiverNonce;
    if (senderNonce != null) result.senderNonce = senderNonce;
    if (ephemeralCertificateSha256 != null)
      result.ephemeralCertificateSha256 = ephemeralCertificateSha256;
    if (oneTimeSessionSecretSha256 != null)
      result.oneTimeSessionSecretSha256 = oneTimeSessionSecretSha256;
    return result;
  }

  ShareReceiverProofInput._();

  factory ShareReceiverProofInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareReceiverProofInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareReceiverProofInput',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'protocol')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOS(3, _omitFieldNames ? '' : 'transferId')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'receiverNonce', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'senderNonce', $pb.PbFieldType.OY)
    ..aOS(6, _omitFieldNames ? '' : 'ephemeralCertificateSha256')
    ..a<$core.List<$core.int>>(7,
        _omitFieldNames ? '' : 'oneTimeSessionSecretSha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareReceiverProofInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareReceiverProofInput copyWith(
          void Function(ShareReceiverProofInput) updates) =>
      super.copyWith((message) => updates(message as ShareReceiverProofInput))
          as ShareReceiverProofInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareReceiverProofInput create() => ShareReceiverProofInput._();
  @$core.override
  ShareReceiverProofInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareReceiverProofInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareReceiverProofInput>(create);
  static ShareReceiverProofInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get protocol => $_getSZ(0);
  @$pb.TagNumber(1)
  set protocol($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocol() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocol() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get transferId => $_getSZ(2);
  @$pb.TagNumber(3)
  set transferId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTransferId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTransferId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get receiverNonce => $_getN(3);
  @$pb.TagNumber(4)
  set receiverNonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReceiverNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearReceiverNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get senderNonce => $_getN(4);
  @$pb.TagNumber(5)
  set senderNonce($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSenderNonce() => $_has(4);
  @$pb.TagNumber(5)
  void clearSenderNonce() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get ephemeralCertificateSha256 => $_getSZ(5);
  @$pb.TagNumber(6)
  set ephemeralCertificateSha256($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasEphemeralCertificateSha256() => $_has(5);
  @$pb.TagNumber(6)
  void clearEphemeralCertificateSha256() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get oneTimeSessionSecretSha256 => $_getN(6);
  @$pb.TagNumber(7)
  set oneTimeSessionSecretSha256($core.List<$core.int> value) =>
      $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasOneTimeSessionSecretSha256() => $_has(6);
  @$pb.TagNumber(7)
  void clearOneTimeSessionSecretSha256() => $_clearField(7);
}

/// ShareReceiverProof 证明 challenge 的接收者持有 hello 中临时公钥对应的私钥。
class ShareReceiverProof extends $pb.GeneratedMessage {
  factory ShareReceiverProof({
    $core.int? schemaVersion,
    $core.String? transferId,
    $core.List<$core.int>? signature,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (transferId != null) result.transferId = transferId;
    if (signature != null) result.signature = signature;
    return result;
  }

  ShareReceiverProof._();

  factory ShareReceiverProof.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareReceiverProof.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareReceiverProof',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'transferId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareReceiverProof clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareReceiverProof copyWith(void Function(ShareReceiverProof) updates) =>
      super.copyWith((message) => updates(message as ShareReceiverProof))
          as ShareReceiverProof;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareReceiverProof create() => ShareReceiverProof._();
  @$core.override
  ShareReceiverProof createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareReceiverProof getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareReceiverProof>(create);
  static ShareReceiverProof? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get transferId => $_getSZ(1);
  @$pb.TagNumber(2)
  set transferId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTransferId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransferId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get signature => $_getN(2);
  @$pb.TagNumber(3)
  set signature($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSignature() => $_has(2);
  @$pb.TagNumber(3)
  void clearSignature() => $_clearField(3);
}

enum ShareSessionClientEnvelope_Message { hello, proof, notSet }

/// ShareSessionClientEnvelope 是一次性 TLS share session 的客户端 framing payload。
class ShareSessionClientEnvelope extends $pb.GeneratedMessage {
  factory ShareSessionClientEnvelope({
    ShareReceiverHello? hello,
    ShareReceiverProof? proof,
  }) {
    final result = create();
    if (hello != null) result.hello = hello;
    if (proof != null) result.proof = proof;
    return result;
  }

  ShareSessionClientEnvelope._();

  factory ShareSessionClientEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareSessionClientEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ShareSessionClientEnvelope_Message>
      _ShareSessionClientEnvelope_MessageByTag = {
    1: ShareSessionClientEnvelope_Message.hello,
    2: ShareSessionClientEnvelope_Message.proof,
    0: ShareSessionClientEnvelope_Message.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareSessionClientEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2])
    ..aOM<ShareReceiverHello>(1, _omitFieldNames ? '' : 'hello',
        subBuilder: ShareReceiverHello.create)
    ..aOM<ShareReceiverProof>(2, _omitFieldNames ? '' : 'proof',
        subBuilder: ShareReceiverProof.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionClientEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionClientEnvelope copyWith(
          void Function(ShareSessionClientEnvelope) updates) =>
      super.copyWith(
              (message) => updates(message as ShareSessionClientEnvelope))
          as ShareSessionClientEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareSessionClientEnvelope create() => ShareSessionClientEnvelope._();
  @$core.override
  ShareSessionClientEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareSessionClientEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareSessionClientEnvelope>(create);
  static ShareSessionClientEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  ShareSessionClientEnvelope_Message whichMessage() =>
      _ShareSessionClientEnvelope_MessageByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  ShareReceiverHello get hello => $_getN(0);
  @$pb.TagNumber(1)
  set hello(ShareReceiverHello value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasHello() => $_has(0);
  @$pb.TagNumber(1)
  void clearHello() => $_clearField(1);
  @$pb.TagNumber(1)
  ShareReceiverHello ensureHello() => $_ensure(0);

  @$pb.TagNumber(2)
  ShareReceiverProof get proof => $_getN(1);
  @$pb.TagNumber(2)
  set proof(ShareReceiverProof value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasProof() => $_has(1);
  @$pb.TagNumber(2)
  void clearProof() => $_clearField(2);
  @$pb.TagNumber(2)
  ShareReceiverProof ensureProof() => $_ensure(1);
}

/// ShareSessionError 是 share session fail-closed 后返回的稳定错误投影，不包含 secret 或配置内容。
class ShareSessionError extends $pb.GeneratedMessage {
  factory ShareSessionError({
    $core.String? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  ShareSessionError._();

  factory ShareSessionError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareSessionError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareSessionError',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'code')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionError copyWith(void Function(ShareSessionError) updates) =>
      super.copyWith((message) => updates(message as ShareSessionError))
          as ShareSessionError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareSessionError create() => ShareSessionError._();
  @$core.override
  ShareSessionError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareSessionError getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareSessionError>(create);
  static ShareSessionError? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get code => $_getSZ(0);
  @$pb.TagNumber(1)
  set code($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

enum ShareSessionServerEnvelope_Message { challenge, bundle, error, notSet }

/// ShareSessionServerEnvelope 是一次性 TLS share session 的服务端 framing payload。
class ShareSessionServerEnvelope extends $pb.GeneratedMessage {
  factory ShareSessionServerEnvelope({
    ShareSenderChallenge? challenge,
    ClientEndpointShareBundleV1? bundle,
    ShareSessionError? error,
  }) {
    final result = create();
    if (challenge != null) result.challenge = challenge;
    if (bundle != null) result.bundle = bundle;
    if (error != null) result.error = error;
    return result;
  }

  ShareSessionServerEnvelope._();

  factory ShareSessionServerEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShareSessionServerEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ShareSessionServerEnvelope_Message>
      _ShareSessionServerEnvelope_MessageByTag = {
    1: ShareSessionServerEnvelope_Message.challenge,
    2: ShareSessionServerEnvelope_Message.bundle,
    3: ShareSessionServerEnvelope_Message.error,
    0: ShareSessionServerEnvelope_Message.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShareSessionServerEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.remote.auth.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3])
    ..aOM<ShareSenderChallenge>(1, _omitFieldNames ? '' : 'challenge',
        subBuilder: ShareSenderChallenge.create)
    ..aOM<ClientEndpointShareBundleV1>(2, _omitFieldNames ? '' : 'bundle',
        subBuilder: ClientEndpointShareBundleV1.create)
    ..aOM<ShareSessionError>(3, _omitFieldNames ? '' : 'error',
        subBuilder: ShareSessionError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionServerEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShareSessionServerEnvelope copyWith(
          void Function(ShareSessionServerEnvelope) updates) =>
      super.copyWith(
              (message) => updates(message as ShareSessionServerEnvelope))
          as ShareSessionServerEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShareSessionServerEnvelope create() => ShareSessionServerEnvelope._();
  @$core.override
  ShareSessionServerEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShareSessionServerEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ShareSessionServerEnvelope>(create);
  static ShareSessionServerEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  ShareSessionServerEnvelope_Message whichMessage() =>
      _ShareSessionServerEnvelope_MessageByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  ShareSenderChallenge get challenge => $_getN(0);
  @$pb.TagNumber(1)
  set challenge(ShareSenderChallenge value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasChallenge() => $_has(0);
  @$pb.TagNumber(1)
  void clearChallenge() => $_clearField(1);
  @$pb.TagNumber(1)
  ShareSenderChallenge ensureChallenge() => $_ensure(0);

  @$pb.TagNumber(2)
  ClientEndpointShareBundleV1 get bundle => $_getN(1);
  @$pb.TagNumber(2)
  set bundle(ClientEndpointShareBundleV1 value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasBundle() => $_has(1);
  @$pb.TagNumber(2)
  void clearBundle() => $_clearField(2);
  @$pb.TagNumber(2)
  ClientEndpointShareBundleV1 ensureBundle() => $_ensure(1);

  @$pb.TagNumber(3)
  ShareSessionError get error => $_getN(2);
  @$pb.TagNumber(3)
  set error(ShareSessionError value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasError() => $_has(2);
  @$pb.TagNumber(3)
  void clearError() => $_clearField(3);
  @$pb.TagNumber(3)
  ShareSessionError ensureError() => $_ensure(2);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
