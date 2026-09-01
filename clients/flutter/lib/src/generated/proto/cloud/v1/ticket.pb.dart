// This is a generated file - do not edit.
//
// Generated from cloud/v1/ticket.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $0;

import 'common.pb.dart' as $1;
import 'runtime.pbenum.dart' as $2;
import 'ticket.pbenum.dart';
import 'usage.pbenum.dart' as $3;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'ticket.pbenum.dart';

/// DaemonBindingClaims 是 enrollment 的持久结果。它只绑定 daemon 身份和目标 Edge，
/// 不是 bearer token；每次 AgentGateway 连接仍必须证明 DeviceIdentity 私钥持有权。
class DaemonBindingClaims extends $pb.GeneratedMessage {
  factory DaemonBindingClaims({
    $core.String? bindingId,
    $core.String? daemonId,
    $core.String? accountId,
    $core.String? edgeId,
    $core.String? deviceId,
    $core.List<$core.int>? devicePublicKey,
    $core.Iterable<DaemonCapability>? capabilities,
    $0.Timestamp? issuedAt,
    $0.Timestamp? expiresAt,
    $core.List<$core.int>? edgeLocatorSha256,
  }) {
    final result = create();
    if (bindingId != null) result.bindingId = bindingId;
    if (daemonId != null) result.daemonId = daemonId;
    if (accountId != null) result.accountId = accountId;
    if (edgeId != null) result.edgeId = edgeId;
    if (deviceId != null) result.deviceId = deviceId;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    if (capabilities != null) result.capabilities.addAll(capabilities);
    if (issuedAt != null) result.issuedAt = issuedAt;
    if (expiresAt != null) result.expiresAt = expiresAt;
    if (edgeLocatorSha256 != null) result.edgeLocatorSha256 = edgeLocatorSha256;
    return result;
  }

  DaemonBindingClaims._();

  factory DaemonBindingClaims.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonBindingClaims.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonBindingClaims',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'bindingId')
    ..aOS(2, _omitFieldNames ? '' : 'daemonId')
    ..aOS(3, _omitFieldNames ? '' : 'accountId')
    ..aOS(4, _omitFieldNames ? '' : 'edgeId')
    ..aOS(5, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..pc<DaemonCapability>(
        7, _omitFieldNames ? '' : 'capabilities', $pb.PbFieldType.KE,
        valueOf: DaemonCapability.valueOf,
        enumValues: DaemonCapability.values,
        defaultEnumValue: DaemonCapability.DAEMON_CAPABILITY_UNSPECIFIED)
    ..aOM<$0.Timestamp>(8, _omitFieldNames ? '' : 'issuedAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..a<$core.List<$core.int>>(
        12, _omitFieldNames ? '' : 'edgeLocatorSha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonBindingClaims clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonBindingClaims copyWith(void Function(DaemonBindingClaims) updates) =>
      super.copyWith((message) => updates(message as DaemonBindingClaims))
          as DaemonBindingClaims;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonBindingClaims create() => DaemonBindingClaims._();
  @$core.override
  DaemonBindingClaims createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonBindingClaims getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonBindingClaims>(create);
  static DaemonBindingClaims? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get bindingId => $_getSZ(0);
  @$pb.TagNumber(1)
  set bindingId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBindingId() => $_has(0);
  @$pb.TagNumber(1)
  void clearBindingId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get daemonId => $_getSZ(1);
  @$pb.TagNumber(2)
  set daemonId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get accountId => $_getSZ(2);
  @$pb.TagNumber(3)
  set accountId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAccountId() => $_has(2);
  @$pb.TagNumber(3)
  void clearAccountId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get edgeId => $_getSZ(3);
  @$pb.TagNumber(4)
  set edgeId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEdgeId() => $_has(3);
  @$pb.TagNumber(4)
  void clearEdgeId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get deviceId => $_getSZ(4);
  @$pb.TagNumber(5)
  set deviceId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDeviceId() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeviceId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get devicePublicKey => $_getN(5);
  @$pb.TagNumber(6)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDevicePublicKey() => $_has(5);
  @$pb.TagNumber(6)
  void clearDevicePublicKey() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbList<DaemonCapability> get capabilities => $_getList(6);

  @$pb.TagNumber(8)
  $0.Timestamp get issuedAt => $_getN(7);
  @$pb.TagNumber(8)
  set issuedAt($0.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasIssuedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearIssuedAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Timestamp ensureIssuedAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $0.Timestamp get expiresAt => $_getN(8);
  @$pb.TagNumber(9)
  set expiresAt($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasExpiresAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearExpiresAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureExpiresAt() => $_ensure(8);

  @$pb.TagNumber(12)
  $core.List<$core.int> get edgeLocatorSha256 => $_getN(9);
  @$pb.TagNumber(12)
  set edgeLocatorSha256($core.List<$core.int> value) => $_setBytes(9, value);
  @$pb.TagNumber(12)
  $core.bool hasEdgeLocatorSha256() => $_has(9);
  @$pb.TagNumber(12)
  void clearEdgeLocatorSha256() => $_clearField(12);
}

/// AgentHelloProofInput 覆盖 Edge 单次 challenge、完整 binding envelope 摘要和 AgentHello 中除 proof 自身外的全部字段。
class AgentHelloProofInput extends $pb.GeneratedMessage {
  factory AgentHelloProofInput({
    $core.List<$core.int>? bindingEnvelopeSha256,
    $core.String? daemonId,
    $core.String? daemonBootId,
    $core.String? daemonSessionId,
    $1.EdgeChallenge? challenge,
    $core.int? protocolVersion,
    $core.String? messageId,
    $fixnum.Int64? streamSeq,
    $0.Timestamp? sentAt,
    $core.String? softwareVersion,
    $fixnum.Int64? attemptGeneration,
  }) {
    final result = create();
    if (bindingEnvelopeSha256 != null)
      result.bindingEnvelopeSha256 = bindingEnvelopeSha256;
    if (daemonId != null) result.daemonId = daemonId;
    if (daemonBootId != null) result.daemonBootId = daemonBootId;
    if (daemonSessionId != null) result.daemonSessionId = daemonSessionId;
    if (challenge != null) result.challenge = challenge;
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (messageId != null) result.messageId = messageId;
    if (streamSeq != null) result.streamSeq = streamSeq;
    if (sentAt != null) result.sentAt = sentAt;
    if (softwareVersion != null) result.softwareVersion = softwareVersion;
    if (attemptGeneration != null) result.attemptGeneration = attemptGeneration;
    return result;
  }

  AgentHelloProofInput._();

  factory AgentHelloProofInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentHelloProofInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentHelloProofInput',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'bindingEnvelopeSha256', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'daemonId')
    ..aOS(3, _omitFieldNames ? '' : 'daemonBootId')
    ..aOS(4, _omitFieldNames ? '' : 'daemonSessionId')
    ..aOM<$1.EdgeChallenge>(5, _omitFieldNames ? '' : 'challenge',
        subBuilder: $1.EdgeChallenge.create)
    ..aI(6, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(7, _omitFieldNames ? '' : 'messageId')
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $0.Timestamp.create)
    ..aOS(10, _omitFieldNames ? '' : 'softwareVersion')
    ..a<$fixnum.Int64>(
        11, _omitFieldNames ? '' : 'attemptGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentHelloProofInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentHelloProofInput copyWith(void Function(AgentHelloProofInput) updates) =>
      super.copyWith((message) => updates(message as AgentHelloProofInput))
          as AgentHelloProofInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentHelloProofInput create() => AgentHelloProofInput._();
  @$core.override
  AgentHelloProofInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentHelloProofInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentHelloProofInput>(create);
  static AgentHelloProofInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get bindingEnvelopeSha256 => $_getN(0);
  @$pb.TagNumber(1)
  set bindingEnvelopeSha256($core.List<$core.int> value) =>
      $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBindingEnvelopeSha256() => $_has(0);
  @$pb.TagNumber(1)
  void clearBindingEnvelopeSha256() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get daemonId => $_getSZ(1);
  @$pb.TagNumber(2)
  set daemonId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get daemonBootId => $_getSZ(2);
  @$pb.TagNumber(3)
  set daemonBootId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonBootId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonBootId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get daemonSessionId => $_getSZ(3);
  @$pb.TagNumber(4)
  set daemonSessionId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemonSessionId() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemonSessionId() => $_clearField(4);

  @$pb.TagNumber(5)
  $1.EdgeChallenge get challenge => $_getN(4);
  @$pb.TagNumber(5)
  set challenge($1.EdgeChallenge value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasChallenge() => $_has(4);
  @$pb.TagNumber(5)
  void clearChallenge() => $_clearField(5);
  @$pb.TagNumber(5)
  $1.EdgeChallenge ensureChallenge() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.int get protocolVersion => $_getIZ(5);
  @$pb.TagNumber(6)
  set protocolVersion($core.int value) => $_setUnsignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasProtocolVersion() => $_has(5);
  @$pb.TagNumber(6)
  void clearProtocolVersion() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get messageId => $_getSZ(6);
  @$pb.TagNumber(7)
  set messageId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMessageId() => $_has(6);
  @$pb.TagNumber(7)
  void clearMessageId() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get streamSeq => $_getI64(7);
  @$pb.TagNumber(8)
  set streamSeq($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasStreamSeq() => $_has(7);
  @$pb.TagNumber(8)
  void clearStreamSeq() => $_clearField(8);

  @$pb.TagNumber(9)
  $0.Timestamp get sentAt => $_getN(8);
  @$pb.TagNumber(9)
  set sentAt($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasSentAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearSentAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureSentAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $core.String get softwareVersion => $_getSZ(9);
  @$pb.TagNumber(10)
  set softwareVersion($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasSoftwareVersion() => $_has(9);
  @$pb.TagNumber(10)
  void clearSoftwareVersion() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get attemptGeneration => $_getI64(10);
  @$pb.TagNumber(11)
  set attemptGeneration($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasAttemptGeneration() => $_has(10);
  @$pb.TagNumber(11)
  void clearAttemptGeneration() => $_clearField(11);
}

/// CloudRouteGrantClaims 由 owning daemon 的 DeviceIdentity 签发，只授权发现和信令。
/// 该消息不得出现 terminal、scope、CapabilityGrant、命令或文件字段。
class CloudRouteGrantClaims extends $pb.GeneratedMessage {
  factory CloudRouteGrantClaims({
    $core.String? grantId,
    $core.String? daemonId,
    $core.List<$core.int>? clientPublicKey,
    $2.ClientProduct? product,
    $0.Timestamp? issuedAt,
    $0.Timestamp? expiresAt,
  }) {
    final result = create();
    if (grantId != null) result.grantId = grantId;
    if (daemonId != null) result.daemonId = daemonId;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (product != null) result.product = product;
    if (issuedAt != null) result.issuedAt = issuedAt;
    if (expiresAt != null) result.expiresAt = expiresAt;
    return result;
  }

  CloudRouteGrantClaims._();

  factory CloudRouteGrantClaims.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudRouteGrantClaims.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudRouteGrantClaims',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'grantId')
    ..aOS(2, _omitFieldNames ? '' : 'daemonId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..aE<$2.ClientProduct>(4, _omitFieldNames ? '' : 'product',
        enumValues: $2.ClientProduct.values)
    ..aOM<$0.Timestamp>(5, _omitFieldNames ? '' : 'issuedAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudRouteGrantClaims clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudRouteGrantClaims copyWith(
          void Function(CloudRouteGrantClaims) updates) =>
      super.copyWith((message) => updates(message as CloudRouteGrantClaims))
          as CloudRouteGrantClaims;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudRouteGrantClaims create() => CloudRouteGrantClaims._();
  @$core.override
  CloudRouteGrantClaims createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudRouteGrantClaims getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudRouteGrantClaims>(create);
  static CloudRouteGrantClaims? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get grantId => $_getSZ(0);
  @$pb.TagNumber(1)
  set grantId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGrantId() => $_has(0);
  @$pb.TagNumber(1)
  void clearGrantId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get daemonId => $_getSZ(1);
  @$pb.TagNumber(2)
  set daemonId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get clientPublicKey => $_getN(2);
  @$pb.TagNumber(3)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasClientPublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearClientPublicKey() => $_clearField(3);

  @$pb.TagNumber(4)
  $2.ClientProduct get product => $_getN(3);
  @$pb.TagNumber(4)
  set product($2.ClientProduct value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasProduct() => $_has(3);
  @$pb.TagNumber(4)
  void clearProduct() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.Timestamp get issuedAt => $_getN(4);
  @$pb.TagNumber(5)
  set issuedAt($0.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasIssuedAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearIssuedAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Timestamp ensureIssuedAt() => $_ensure(4);

  @$pb.TagNumber(6)
  $0.Timestamp get expiresAt => $_getN(5);
  @$pb.TagNumber(6)
  set expiresAt($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasExpiresAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearExpiresAt() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureExpiresAt() => $_ensure(5);
}

/// ClientRouteProofInput 把客户端 proof 绑定到 Controller nonce、grant 和本次解析请求。
class ClientRouteProofInput extends $pb.GeneratedMessage {
  factory ClientRouteProofInput({
    $core.String? challengeId,
    $core.List<$core.int>? challenge,
    $core.List<$core.int>? grantPayloadSha256,
    $core.String? requestId,
  }) {
    final result = create();
    if (challengeId != null) result.challengeId = challengeId;
    if (challenge != null) result.challenge = challenge;
    if (grantPayloadSha256 != null)
      result.grantPayloadSha256 = grantPayloadSha256;
    if (requestId != null) result.requestId = requestId;
    return result;
  }

  ClientRouteProofInput._();

  factory ClientRouteProofInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientRouteProofInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientRouteProofInput',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'challengeId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'challenge', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'grantPayloadSha256', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientRouteProofInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientRouteProofInput copyWith(
          void Function(ClientRouteProofInput) updates) =>
      super.copyWith((message) => updates(message as ClientRouteProofInput))
          as ClientRouteProofInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientRouteProofInput create() => ClientRouteProofInput._();
  @$core.override
  ClientRouteProofInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientRouteProofInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientRouteProofInput>(create);
  static ClientRouteProofInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get challengeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set challengeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChallengeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearChallengeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get challenge => $_getN(1);
  @$pb.TagNumber(2)
  set challenge($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChallenge() => $_has(1);
  @$pb.TagNumber(2)
  void clearChallenge() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get grantPayloadSha256 => $_getN(2);
  @$pb.TagNumber(3)
  set grantPayloadSha256($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasGrantPayloadSha256() => $_has(2);
  @$pb.TagNumber(3)
  void clearGrantPayloadSha256() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get requestId => $_getSZ(3);
  @$pb.TagNumber(4)
  set requestId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRequestId() => $_has(3);
  @$pb.TagNumber(4)
  void clearRequestId() => $_clearField(4);
}

/// GatewayClientHelloProofInput 对两种 authorization 使用同一 v2 transcript。
/// authorization_sha256 是对应 SignedEnvelope 或 PairingAdmission 的确定性 protobuf 摘要。
class GatewayClientHelloProofInput extends $pb.GeneratedMessage {
  factory GatewayClientHelloProofInput({
    $1.EdgeChallenge? challenge,
    $core.List<$core.int>? authorizationSha256,
    $2.CloudClientAccessMode? accessMode,
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? clientId,
    $core.String? clientBootId,
    $core.String? clientSessionId,
    $fixnum.Int64? streamSeq,
    $0.Timestamp? sentAt,
    $core.List<$core.int>? clientPublicKey,
    $2.ClientProduct? product,
    $core.String? softwareVersion,
    $fixnum.Int64? attemptGeneration,
    $3.RelayPreference? relayPreference,
    $core.bool? presenceProbe,
  }) {
    final result = create();
    if (challenge != null) result.challenge = challenge;
    if (authorizationSha256 != null)
      result.authorizationSha256 = authorizationSha256;
    if (accessMode != null) result.accessMode = accessMode;
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (messageId != null) result.messageId = messageId;
    if (clientId != null) result.clientId = clientId;
    if (clientBootId != null) result.clientBootId = clientBootId;
    if (clientSessionId != null) result.clientSessionId = clientSessionId;
    if (streamSeq != null) result.streamSeq = streamSeq;
    if (sentAt != null) result.sentAt = sentAt;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (product != null) result.product = product;
    if (softwareVersion != null) result.softwareVersion = softwareVersion;
    if (attemptGeneration != null) result.attemptGeneration = attemptGeneration;
    if (relayPreference != null) result.relayPreference = relayPreference;
    if (presenceProbe != null) result.presenceProbe = presenceProbe;
    return result;
  }

  GatewayClientHelloProofInput._();

  factory GatewayClientHelloProofInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GatewayClientHelloProofInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GatewayClientHelloProofInput',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.EdgeChallenge>(1, _omitFieldNames ? '' : 'challenge',
        subBuilder: $1.EdgeChallenge.create)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'authorizationSha256', $pb.PbFieldType.OY)
    ..aE<$2.CloudClientAccessMode>(3, _omitFieldNames ? '' : 'accessMode',
        enumValues: $2.CloudClientAccessMode.values)
    ..aI(4, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(5, _omitFieldNames ? '' : 'messageId')
    ..aOS(6, _omitFieldNames ? '' : 'clientId')
    ..aOS(7, _omitFieldNames ? '' : 'clientBootId')
    ..aOS(8, _omitFieldNames ? '' : 'clientSessionId')
    ..a<$fixnum.Int64>(
        9, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $0.Timestamp.create)
    ..a<$core.List<$core.int>>(
        11, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..aE<$2.ClientProduct>(12, _omitFieldNames ? '' : 'product',
        enumValues: $2.ClientProduct.values)
    ..aOS(13, _omitFieldNames ? '' : 'softwareVersion')
    ..a<$fixnum.Int64>(
        14, _omitFieldNames ? '' : 'attemptGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<$3.RelayPreference>(15, _omitFieldNames ? '' : 'relayPreference',
        enumValues: $3.RelayPreference.values)
    ..aOB(16, _omitFieldNames ? '' : 'presenceProbe')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GatewayClientHelloProofInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GatewayClientHelloProofInput copyWith(
          void Function(GatewayClientHelloProofInput) updates) =>
      super.copyWith(
              (message) => updates(message as GatewayClientHelloProofInput))
          as GatewayClientHelloProofInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GatewayClientHelloProofInput create() =>
      GatewayClientHelloProofInput._();
  @$core.override
  GatewayClientHelloProofInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GatewayClientHelloProofInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GatewayClientHelloProofInput>(create);
  static GatewayClientHelloProofInput? _defaultInstance;

  @$pb.TagNumber(1)
  $1.EdgeChallenge get challenge => $_getN(0);
  @$pb.TagNumber(1)
  set challenge($1.EdgeChallenge value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasChallenge() => $_has(0);
  @$pb.TagNumber(1)
  void clearChallenge() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.EdgeChallenge ensureChallenge() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.List<$core.int> get authorizationSha256 => $_getN(1);
  @$pb.TagNumber(2)
  set authorizationSha256($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAuthorizationSha256() => $_has(1);
  @$pb.TagNumber(2)
  void clearAuthorizationSha256() => $_clearField(2);

  @$pb.TagNumber(3)
  $2.CloudClientAccessMode get accessMode => $_getN(2);
  @$pb.TagNumber(3)
  set accessMode($2.CloudClientAccessMode value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasAccessMode() => $_has(2);
  @$pb.TagNumber(3)
  void clearAccessMode() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get protocolVersion => $_getIZ(3);
  @$pb.TagNumber(4)
  set protocolVersion($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProtocolVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearProtocolVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get messageId => $_getSZ(4);
  @$pb.TagNumber(5)
  set messageId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMessageId() => $_has(4);
  @$pb.TagNumber(5)
  void clearMessageId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get clientId => $_getSZ(5);
  @$pb.TagNumber(6)
  set clientId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasClientId() => $_has(5);
  @$pb.TagNumber(6)
  void clearClientId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get clientBootId => $_getSZ(6);
  @$pb.TagNumber(7)
  set clientBootId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasClientBootId() => $_has(6);
  @$pb.TagNumber(7)
  void clearClientBootId() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get clientSessionId => $_getSZ(7);
  @$pb.TagNumber(8)
  set clientSessionId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasClientSessionId() => $_has(7);
  @$pb.TagNumber(8)
  void clearClientSessionId() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get streamSeq => $_getI64(8);
  @$pb.TagNumber(9)
  set streamSeq($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasStreamSeq() => $_has(8);
  @$pb.TagNumber(9)
  void clearStreamSeq() => $_clearField(9);

  @$pb.TagNumber(10)
  $0.Timestamp get sentAt => $_getN(9);
  @$pb.TagNumber(10)
  set sentAt($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasSentAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearSentAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensureSentAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $core.List<$core.int> get clientPublicKey => $_getN(10);
  @$pb.TagNumber(11)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(10, value);
  @$pb.TagNumber(11)
  $core.bool hasClientPublicKey() => $_has(10);
  @$pb.TagNumber(11)
  void clearClientPublicKey() => $_clearField(11);

  @$pb.TagNumber(12)
  $2.ClientProduct get product => $_getN(11);
  @$pb.TagNumber(12)
  set product($2.ClientProduct value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasProduct() => $_has(11);
  @$pb.TagNumber(12)
  void clearProduct() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get softwareVersion => $_getSZ(12);
  @$pb.TagNumber(13)
  set softwareVersion($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasSoftwareVersion() => $_has(12);
  @$pb.TagNumber(13)
  void clearSoftwareVersion() => $_clearField(13);

  @$pb.TagNumber(14)
  $fixnum.Int64 get attemptGeneration => $_getI64(13);
  @$pb.TagNumber(14)
  set attemptGeneration($fixnum.Int64 value) => $_setInt64(13, value);
  @$pb.TagNumber(14)
  $core.bool hasAttemptGeneration() => $_has(13);
  @$pb.TagNumber(14)
  void clearAttemptGeneration() => $_clearField(14);

  @$pb.TagNumber(15)
  $3.RelayPreference get relayPreference => $_getN(14);
  @$pb.TagNumber(15)
  set relayPreference($3.RelayPreference value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasRelayPreference() => $_has(14);
  @$pb.TagNumber(15)
  void clearRelayPreference() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.bool get presenceProbe => $_getBF(15);
  @$pb.TagNumber(16)
  set presenceProbe($core.bool value) => $_setBool(15, value);
  @$pb.TagNumber(16)
  $core.bool hasPresenceProbe() => $_has(15);
  @$pb.TagNumber(16)
  void clearPresenceProbe() => $_clearField(16);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
