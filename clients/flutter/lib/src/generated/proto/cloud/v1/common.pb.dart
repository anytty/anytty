// This is a generated file - do not edit.
//
// Generated from cloud/v1/common.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/duration.pb.dart'
    as $1;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $0;

import 'common.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'common.pbenum.dart';

/// VerificationKey 是 Controller 下发给 Edge 的公开验签密钥。
/// public_key 只能包含公钥编码，不得携带私钥或 KMS 凭据。
class VerificationKey extends $pb.GeneratedMessage {
  factory VerificationKey({
    $core.String? keyId,
    $core.String? algorithm,
    $core.List<$core.int>? publicKey,
  }) {
    final result = create();
    if (keyId != null) result.keyId = keyId;
    if (algorithm != null) result.algorithm = algorithm;
    if (publicKey != null) result.publicKey = publicKey;
    return result;
  }

  VerificationKey._();

  factory VerificationKey.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory VerificationKey.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'VerificationKey',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'keyId')
    ..aOS(2, _omitFieldNames ? '' : 'algorithm')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'publicKey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VerificationKey clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VerificationKey copyWith(void Function(VerificationKey) updates) =>
      super.copyWith((message) => updates(message as VerificationKey))
          as VerificationKey;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static VerificationKey create() => VerificationKey._();
  @$core.override
  VerificationKey createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static VerificationKey getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<VerificationKey>(create);
  static VerificationKey? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get keyId => $_getSZ(0);
  @$pb.TagNumber(1)
  set keyId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKeyId() => $_has(0);
  @$pb.TagNumber(1)
  void clearKeyId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get algorithm => $_getSZ(1);
  @$pb.TagNumber(2)
  set algorithm($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAlgorithm() => $_has(1);
  @$pb.TagNumber(2)
  void clearAlgorithm() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get publicKey => $_getN(2);
  @$pb.TagNumber(3)
  set publicKey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearPublicKey() => $_clearField(3);
}

/// KeyBundle 是 Controller 发布并由 Edge 原子持久化的完整 binding 验签 keyset。
/// revision 只随 keyset owner 单调变化；issued_at/expires_at 限制离线 admission 窗口。
class KeyBundle extends $pb.GeneratedMessage {
  factory KeyBundle({
    $fixnum.Int64? revision,
    $0.Timestamp? issuedAt,
    $0.Timestamp? expiresAt,
    $core.Iterable<VerificationKey>? keys,
  }) {
    final result = create();
    if (revision != null) result.revision = revision;
    if (issuedAt != null) result.issuedAt = issuedAt;
    if (expiresAt != null) result.expiresAt = expiresAt;
    if (keys != null) result.keys.addAll(keys);
    return result;
  }

  KeyBundle._();

  factory KeyBundle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory KeyBundle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'KeyBundle',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(2, _omitFieldNames ? '' : 'issuedAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..pPM<VerificationKey>(4, _omitFieldNames ? '' : 'keys',
        subBuilder: VerificationKey.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyBundle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyBundle copyWith(void Function(KeyBundle) updates) =>
      super.copyWith((message) => updates(message as KeyBundle)) as KeyBundle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static KeyBundle create() => KeyBundle._();
  @$core.override
  KeyBundle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static KeyBundle getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<KeyBundle>(create);
  static KeyBundle? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get revision => $_getI64(0);
  @$pb.TagNumber(1)
  set revision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevision() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.Timestamp get issuedAt => $_getN(1);
  @$pb.TagNumber(2)
  set issuedAt($0.Timestamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasIssuedAt() => $_has(1);
  @$pb.TagNumber(2)
  void clearIssuedAt() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.Timestamp ensureIssuedAt() => $_ensure(1);

  @$pb.TagNumber(3)
  $0.Timestamp get expiresAt => $_getN(2);
  @$pb.TagNumber(3)
  set expiresAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasExpiresAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpiresAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureExpiresAt() => $_ensure(2);

  @$pb.TagNumber(4)
  $pb.PbList<VerificationKey> get keys => $_getList(3);
}

/// HeartbeatPolicy 约束 EdgeControl 的心跳频率和 Controller 失联判定。
class HeartbeatPolicy extends $pb.GeneratedMessage {
  factory HeartbeatPolicy({
    $1.Duration? interval,
    $1.Duration? timeout,
  }) {
    final result = create();
    if (interval != null) result.interval = interval;
    if (timeout != null) result.timeout = timeout;
    return result;
  }

  HeartbeatPolicy._();

  factory HeartbeatPolicy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HeartbeatPolicy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HeartbeatPolicy',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.Duration>(1, _omitFieldNames ? '' : 'interval',
        subBuilder: $1.Duration.create)
    ..aOM<$1.Duration>(2, _omitFieldNames ? '' : 'timeout',
        subBuilder: $1.Duration.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HeartbeatPolicy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HeartbeatPolicy copyWith(void Function(HeartbeatPolicy) updates) =>
      super.copyWith((message) => updates(message as HeartbeatPolicy))
          as HeartbeatPolicy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HeartbeatPolicy create() => HeartbeatPolicy._();
  @$core.override
  HeartbeatPolicy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HeartbeatPolicy getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HeartbeatPolicy>(create);
  static HeartbeatPolicy? _defaultInstance;

  @$pb.TagNumber(1)
  $1.Duration get interval => $_getN(0);
  @$pb.TagNumber(1)
  set interval($1.Duration value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasInterval() => $_has(0);
  @$pb.TagNumber(1)
  void clearInterval() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.Duration ensureInterval() => $_ensure(0);

  @$pb.TagNumber(2)
  $1.Duration get timeout => $_getN(1);
  @$pb.TagNumber(2)
  set timeout($1.Duration value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTimeout() => $_has(1);
  @$pb.TagNumber(2)
  void clearTimeout() => $_clearField(2);
  @$pb.TagNumber(2)
  $1.Duration ensureTimeout() => $_ensure(1);
}

/// SignedEnvelope 使用 domain-separated Ed25519 签名保护确定性 Proto payload。
class SignedEnvelope extends $pb.GeneratedMessage {
  factory SignedEnvelope({
    $core.String? keyId,
    $core.List<$core.int>? payload,
    $core.List<$core.int>? signature,
  }) {
    final result = create();
    if (keyId != null) result.keyId = keyId;
    if (payload != null) result.payload = payload;
    if (signature != null) result.signature = signature;
    return result;
  }

  SignedEnvelope._();

  factory SignedEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SignedEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SignedEnvelope',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'keyId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'payload', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignedEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignedEnvelope copyWith(void Function(SignedEnvelope) updates) =>
      super.copyWith((message) => updates(message as SignedEnvelope))
          as SignedEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SignedEnvelope create() => SignedEnvelope._();
  @$core.override
  SignedEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SignedEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SignedEnvelope>(create);
  static SignedEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get keyId => $_getSZ(0);
  @$pb.TagNumber(1)
  set keyId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKeyId() => $_has(0);
  @$pb.TagNumber(1)
  void clearKeyId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get payload => $_getN(1);
  @$pb.TagNumber(2)
  set payload($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPayload() => $_has(1);
  @$pb.TagNumber(2)
  void clearPayload() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get signature => $_getN(2);
  @$pb.TagNumber(3)
  set signature($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSignature() => $_has(2);
  @$pb.TagNumber(3)
  void clearSignature() => $_clearField(3);
}

/// CloudEntitlementFailure 可安全投影给未登录 App，不包含账号或订阅 identity。
class CloudEntitlementFailure extends $pb.GeneratedMessage {
  factory CloudEntitlementFailure({
    CloudEntitlementErrorCode? code,
    $core.String? message,
    $fixnum.Int64? limit,
    $fixnum.Int64? used,
    $fixnum.Int64? remainingBytes,
    $0.Timestamp? periodEnd,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    if (limit != null) result.limit = limit;
    if (used != null) result.used = used;
    if (remainingBytes != null) result.remainingBytes = remainingBytes;
    if (periodEnd != null) result.periodEnd = periodEnd;
    return result;
  }

  CloudEntitlementFailure._();

  factory CloudEntitlementFailure.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudEntitlementFailure.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudEntitlementFailure',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aE<CloudEntitlementErrorCode>(1, _omitFieldNames ? '' : 'code',
        enumValues: CloudEntitlementErrorCode.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'limit', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(4, _omitFieldNames ? '' : 'used', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'remainingBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'periodEnd',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudEntitlementFailure clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudEntitlementFailure copyWith(
          void Function(CloudEntitlementFailure) updates) =>
      super.copyWith((message) => updates(message as CloudEntitlementFailure))
          as CloudEntitlementFailure;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudEntitlementFailure create() => CloudEntitlementFailure._();
  @$core.override
  CloudEntitlementFailure createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudEntitlementFailure getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudEntitlementFailure>(create);
  static CloudEntitlementFailure? _defaultInstance;

  @$pb.TagNumber(1)
  CloudEntitlementErrorCode get code => $_getN(0);
  @$pb.TagNumber(1)
  set code(CloudEntitlementErrorCode value) => $_setField(1, value);
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

  @$pb.TagNumber(3)
  $fixnum.Int64 get limit => $_getI64(2);
  @$pb.TagNumber(3)
  set limit($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLimit() => $_has(2);
  @$pb.TagNumber(3)
  void clearLimit() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get used => $_getI64(3);
  @$pb.TagNumber(4)
  set used($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUsed() => $_has(3);
  @$pb.TagNumber(4)
  void clearUsed() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get remainingBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set remainingBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRemainingBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearRemainingBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get periodEnd => $_getN(5);
  @$pb.TagNumber(6)
  set periodEnd($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasPeriodEnd() => $_has(5);
  @$pb.TagNumber(6)
  void clearPeriodEnd() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensurePeriodEnd() => $_ensure(5);
}

/// EdgeChallenge 是 Edge 在每条 Gateway stream 上先发的单次新鲜度证明材料。
/// nonce 固定为 32 bytes，stream_id 由 Edge 为该 stream 唯一生成，期限固定为 10 秒。
class EdgeChallenge extends $pb.GeneratedMessage {
  factory EdgeChallenge({
    $core.List<$core.int>? nonce,
    $core.String? edgeId,
    $core.String? edgeBootId,
    $core.String? streamId,
    $0.Timestamp? issuedAt,
    $0.Timestamp? expiresAt,
    EdgeChallengeTarget? target,
  }) {
    final result = create();
    if (nonce != null) result.nonce = nonce;
    if (edgeId != null) result.edgeId = edgeId;
    if (edgeBootId != null) result.edgeBootId = edgeBootId;
    if (streamId != null) result.streamId = streamId;
    if (issuedAt != null) result.issuedAt = issuedAt;
    if (expiresAt != null) result.expiresAt = expiresAt;
    if (target != null) result.target = target;
    return result;
  }

  EdgeChallenge._();

  factory EdgeChallenge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeChallenge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeChallenge',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'nonce', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'edgeId')
    ..aOS(3, _omitFieldNames ? '' : 'edgeBootId')
    ..aOS(4, _omitFieldNames ? '' : 'streamId')
    ..aOM<$0.Timestamp>(5, _omitFieldNames ? '' : 'issuedAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..aE<EdgeChallengeTarget>(7, _omitFieldNames ? '' : 'target',
        enumValues: EdgeChallengeTarget.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeChallenge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeChallenge copyWith(void Function(EdgeChallenge) updates) =>
      super.copyWith((message) => updates(message as EdgeChallenge))
          as EdgeChallenge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeChallenge create() => EdgeChallenge._();
  @$core.override
  EdgeChallenge createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeChallenge getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeChallenge>(create);
  static EdgeChallenge? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get nonce => $_getN(0);
  @$pb.TagNumber(1)
  set nonce($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasNonce() => $_has(0);
  @$pb.TagNumber(1)
  void clearNonce() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get edgeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set edgeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEdgeId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEdgeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get edgeBootId => $_getSZ(2);
  @$pb.TagNumber(3)
  set edgeBootId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEdgeBootId() => $_has(2);
  @$pb.TagNumber(3)
  void clearEdgeBootId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get streamId => $_getSZ(3);
  @$pb.TagNumber(4)
  set streamId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStreamId() => $_has(3);
  @$pb.TagNumber(4)
  void clearStreamId() => $_clearField(4);

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

  @$pb.TagNumber(7)
  EdgeChallengeTarget get target => $_getN(6);
  @$pb.TagNumber(7)
  set target(EdgeChallengeTarget value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasTarget() => $_has(6);
  @$pb.TagNumber(7)
  void clearTarget() => $_clearField(7);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
