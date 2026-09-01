// This is a generated file - do not edit.
//
// Generated from cloud/v1/agent_gateway.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $4;

import 'client_gateway.pb.dart' as $1;
import 'common.pb.dart' as $0;
import 'enrollment.pb.dart' as $3;
import 'runtime.pbenum.dart' as $5;
import 'usage.pb.dart' as $2;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class AgentHello extends $pb.GeneratedMessage {
  factory AgentHello({
    $0.SignedEnvelope? daemonBinding,
    $core.List<$core.int>? deviceProof,
    $core.String? softwareVersion,
    $fixnum.Int64? attemptGeneration,
  }) {
    final result = create();
    if (daemonBinding != null) result.daemonBinding = daemonBinding;
    if (deviceProof != null) result.deviceProof = deviceProof;
    if (softwareVersion != null) result.softwareVersion = softwareVersion;
    if (attemptGeneration != null) result.attemptGeneration = attemptGeneration;
    return result;
  }

  AgentHello._();

  factory AgentHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentHello',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$0.SignedEnvelope>(1, _omitFieldNames ? '' : 'daemonBinding',
        subBuilder: $0.SignedEnvelope.create)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'deviceProof', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'softwareVersion')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'attemptGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentHello copyWith(void Function(AgentHello) updates) =>
      super.copyWith((message) => updates(message as AgentHello)) as AgentHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentHello create() => AgentHello._();
  @$core.override
  AgentHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentHello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentHello>(create);
  static AgentHello? _defaultInstance;

  @$pb.TagNumber(1)
  $0.SignedEnvelope get daemonBinding => $_getN(0);
  @$pb.TagNumber(1)
  set daemonBinding($0.SignedEnvelope value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonBinding() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonBinding() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.SignedEnvelope ensureDaemonBinding() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.List<$core.int> get deviceProof => $_getN(1);
  @$pb.TagNumber(2)
  set deviceProof($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceProof() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceProof() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get softwareVersion => $_getSZ(2);
  @$pb.TagNumber(3)
  set softwareVersion($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSoftwareVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearSoftwareVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get attemptGeneration => $_getI64(3);
  @$pb.TagNumber(4)
  set attemptGeneration($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAttemptGeneration() => $_has(3);
  @$pb.TagNumber(4)
  void clearAttemptGeneration() => $_clearField(4);
}

class AgentHeartbeat extends $pb.GeneratedMessage {
  factory AgentHeartbeat({
    $fixnum.Int64? generation,
  }) {
    final result = create();
    if (generation != null) result.generation = generation;
    return result;
  }

  AgentHeartbeat._();

  factory AgentHeartbeat.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentHeartbeat.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentHeartbeat',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentHeartbeat clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentHeartbeat copyWith(void Function(AgentHeartbeat) updates) =>
      super.copyWith((message) => updates(message as AgentHeartbeat))
          as AgentHeartbeat;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentHeartbeat create() => AgentHeartbeat._();
  @$core.override
  AgentHeartbeat createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentHeartbeat getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentHeartbeat>(create);
  static AgentHeartbeat? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get generation => $_getI64(0);
  @$pb.TagNumber(1)
  set generation($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGeneration() => $_has(0);
  @$pb.TagNumber(1)
  void clearGeneration() => $_clearField(1);
}

class AgentOffer extends $pb.GeneratedMessage {
  factory AgentOffer({
    $core.String? correlationId,
    $core.String? sessionId,
    $fixnum.Int64? agentGeneration,
    $core.List<$core.int>? clientPublicKey,
    $core.String? offerSdp,
    $core.Iterable<$1.CloudICECandidate>? candidates,
    $2.RelayICEConfig? relay,
    $5.CloudClientAccessMode? accessMode,
    $core.List<$core.int>? pairingClaimSha256,
  }) {
    final result = create();
    if (correlationId != null) result.correlationId = correlationId;
    if (sessionId != null) result.sessionId = sessionId;
    if (agentGeneration != null) result.agentGeneration = agentGeneration;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (offerSdp != null) result.offerSdp = offerSdp;
    if (candidates != null) result.candidates.addAll(candidates);
    if (relay != null) result.relay = relay;
    if (accessMode != null) result.accessMode = accessMode;
    if (pairingClaimSha256 != null)
      result.pairingClaimSha256 = pairingClaimSha256;
    return result;
  }

  AgentOffer._();

  factory AgentOffer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentOffer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentOffer',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'correlationId')
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'agentGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..aOS(5, _omitFieldNames ? '' : 'offerSdp')
    ..pPM<$1.CloudICECandidate>(6, _omitFieldNames ? '' : 'candidates',
        subBuilder: $1.CloudICECandidate.create)
    ..aOM<$2.RelayICEConfig>(7, _omitFieldNames ? '' : 'relay',
        subBuilder: $2.RelayICEConfig.create)
    ..aE<$5.CloudClientAccessMode>(8, _omitFieldNames ? '' : 'accessMode',
        enumValues: $5.CloudClientAccessMode.values)
    ..a<$core.List<$core.int>>(
        9, _omitFieldNames ? '' : 'pairingClaimSha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentOffer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentOffer copyWith(void Function(AgentOffer) updates) =>
      super.copyWith((message) => updates(message as AgentOffer)) as AgentOffer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentOffer create() => AgentOffer._();
  @$core.override
  AgentOffer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentOffer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentOffer>(create);
  static AgentOffer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get correlationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set correlationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCorrelationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCorrelationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get agentGeneration => $_getI64(2);
  @$pb.TagNumber(3)
  set agentGeneration($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAgentGeneration() => $_has(2);
  @$pb.TagNumber(3)
  void clearAgentGeneration() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get clientPublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasClientPublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearClientPublicKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get offerSdp => $_getSZ(4);
  @$pb.TagNumber(5)
  set offerSdp($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOfferSdp() => $_has(4);
  @$pb.TagNumber(5)
  void clearOfferSdp() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<$1.CloudICECandidate> get candidates => $_getList(5);

  @$pb.TagNumber(7)
  $2.RelayICEConfig get relay => $_getN(6);
  @$pb.TagNumber(7)
  set relay($2.RelayICEConfig value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasRelay() => $_has(6);
  @$pb.TagNumber(7)
  void clearRelay() => $_clearField(7);
  @$pb.TagNumber(7)
  $2.RelayICEConfig ensureRelay() => $_ensure(6);

  @$pb.TagNumber(8)
  $5.CloudClientAccessMode get accessMode => $_getN(7);
  @$pb.TagNumber(8)
  set accessMode($5.CloudClientAccessMode value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasAccessMode() => $_has(7);
  @$pb.TagNumber(8)
  void clearAccessMode() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.List<$core.int> get pairingClaimSha256 => $_getN(8);
  @$pb.TagNumber(9)
  set pairingClaimSha256($core.List<$core.int> value) => $_setBytes(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPairingClaimSha256() => $_has(8);
  @$pb.TagNumber(9)
  void clearPairingClaimSha256() => $_clearField(9);
}

/// AgentAuthorize 要求 owning daemon 在产生 TURN credential 前按本地 AccessStore 预检客户端。
class AgentAuthorize extends $pb.GeneratedMessage {
  factory AgentAuthorize({
    $core.String? correlationId,
    $core.String? sessionId,
    $fixnum.Int64? agentGeneration,
    $core.List<$core.int>? clientPublicKey,
    $5.ClientProduct? product,
    $5.CloudClientAccessMode? accessMode,
    $core.List<$core.int>? pairingClaimSha256,
  }) {
    final result = create();
    if (correlationId != null) result.correlationId = correlationId;
    if (sessionId != null) result.sessionId = sessionId;
    if (agentGeneration != null) result.agentGeneration = agentGeneration;
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (product != null) result.product = product;
    if (accessMode != null) result.accessMode = accessMode;
    if (pairingClaimSha256 != null)
      result.pairingClaimSha256 = pairingClaimSha256;
    return result;
  }

  AgentAuthorize._();

  factory AgentAuthorize.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentAuthorize.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentAuthorize',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'correlationId')
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'agentGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..aE<$5.ClientProduct>(5, _omitFieldNames ? '' : 'product',
        enumValues: $5.ClientProduct.values)
    ..aE<$5.CloudClientAccessMode>(6, _omitFieldNames ? '' : 'accessMode',
        enumValues: $5.CloudClientAccessMode.values)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'pairingClaimSha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentAuthorize clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentAuthorize copyWith(void Function(AgentAuthorize) updates) =>
      super.copyWith((message) => updates(message as AgentAuthorize))
          as AgentAuthorize;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentAuthorize create() => AgentAuthorize._();
  @$core.override
  AgentAuthorize createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentAuthorize getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentAuthorize>(create);
  static AgentAuthorize? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get correlationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set correlationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCorrelationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCorrelationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get agentGeneration => $_getI64(2);
  @$pb.TagNumber(3)
  set agentGeneration($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAgentGeneration() => $_has(2);
  @$pb.TagNumber(3)
  void clearAgentGeneration() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get clientPublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasClientPublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearClientPublicKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $5.ClientProduct get product => $_getN(4);
  @$pb.TagNumber(5)
  set product($5.ClientProduct value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasProduct() => $_has(4);
  @$pb.TagNumber(5)
  void clearProduct() => $_clearField(5);

  @$pb.TagNumber(6)
  $5.CloudClientAccessMode get accessMode => $_getN(5);
  @$pb.TagNumber(6)
  set accessMode($5.CloudClientAccessMode value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasAccessMode() => $_has(5);
  @$pb.TagNumber(6)
  void clearAccessMode() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get pairingClaimSha256 => $_getN(6);
  @$pb.TagNumber(7)
  set pairingClaimSha256($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasPairingClaimSha256() => $_has(6);
  @$pb.TagNumber(7)
  void clearPairingClaimSha256() => $_clearField(7);
}

class AgentAuthorizationResult extends $pb.GeneratedMessage {
  factory AgentAuthorizationResult({
    $core.String? correlationId,
    $core.String? sessionId,
    $core.bool? authorized,
    $core.String? code,
    $core.String? message,
  }) {
    final result = create();
    if (correlationId != null) result.correlationId = correlationId;
    if (sessionId != null) result.sessionId = sessionId;
    if (authorized != null) result.authorized = authorized;
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  AgentAuthorizationResult._();

  factory AgentAuthorizationResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentAuthorizationResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentAuthorizationResult',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'correlationId')
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..aOB(3, _omitFieldNames ? '' : 'authorized')
    ..aOS(4, _omitFieldNames ? '' : 'code')
    ..aOS(5, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentAuthorizationResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentAuthorizationResult copyWith(
          void Function(AgentAuthorizationResult) updates) =>
      super.copyWith((message) => updates(message as AgentAuthorizationResult))
          as AgentAuthorizationResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentAuthorizationResult create() => AgentAuthorizationResult._();
  @$core.override
  AgentAuthorizationResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentAuthorizationResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentAuthorizationResult>(create);
  static AgentAuthorizationResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get correlationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set correlationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCorrelationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCorrelationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get authorized => $_getBF(2);
  @$pb.TagNumber(3)
  set authorized($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAuthorized() => $_has(2);
  @$pb.TagNumber(3)
  void clearAuthorized() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get code => $_getSZ(3);
  @$pb.TagNumber(4)
  set code($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCode() => $_has(3);
  @$pb.TagNumber(4)
  void clearCode() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get message => $_getSZ(4);
  @$pb.TagNumber(5)
  set message($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearMessage() => $_clearField(5);
}

class AgentAnswer extends $pb.GeneratedMessage {
  factory AgentAnswer({
    $core.String? correlationId,
    $core.String? sessionId,
    $core.String? answerSdp,
    $core.Iterable<$1.CloudICECandidate>? candidates,
  }) {
    final result = create();
    if (correlationId != null) result.correlationId = correlationId;
    if (sessionId != null) result.sessionId = sessionId;
    if (answerSdp != null) result.answerSdp = answerSdp;
    if (candidates != null) result.candidates.addAll(candidates);
    return result;
  }

  AgentAnswer._();

  factory AgentAnswer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentAnswer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentAnswer',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'correlationId')
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..aOS(3, _omitFieldNames ? '' : 'answerSdp')
    ..pPM<$1.CloudICECandidate>(4, _omitFieldNames ? '' : 'candidates',
        subBuilder: $1.CloudICECandidate.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentAnswer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentAnswer copyWith(void Function(AgentAnswer) updates) =>
      super.copyWith((message) => updates(message as AgentAnswer))
          as AgentAnswer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentAnswer create() => AgentAnswer._();
  @$core.override
  AgentAnswer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentAnswer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentAnswer>(create);
  static AgentAnswer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get correlationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set correlationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCorrelationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCorrelationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get answerSdp => $_getSZ(2);
  @$pb.TagNumber(3)
  set answerSdp($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAnswerSdp() => $_has(2);
  @$pb.TagNumber(3)
  void clearAnswerSdp() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$1.CloudICECandidate> get candidates => $_getList(3);
}

class AgentSignalRejected extends $pb.GeneratedMessage {
  factory AgentSignalRejected({
    $core.String? correlationId,
    $core.String? sessionId,
    $core.String? code,
    $core.String? message,
  }) {
    final result = create();
    if (correlationId != null) result.correlationId = correlationId;
    if (sessionId != null) result.sessionId = sessionId;
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  AgentSignalRejected._();

  factory AgentSignalRejected.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentSignalRejected.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentSignalRejected',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'correlationId')
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..aOS(3, _omitFieldNames ? '' : 'code')
    ..aOS(4, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentSignalRejected clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentSignalRejected copyWith(void Function(AgentSignalRejected) updates) =>
      super.copyWith((message) => updates(message as AgentSignalRejected))
          as AgentSignalRejected;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentSignalRejected create() => AgentSignalRejected._();
  @$core.override
  AgentSignalRejected createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentSignalRejected getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentSignalRejected>(create);
  static AgentSignalRejected? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get correlationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set correlationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCorrelationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCorrelationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get code => $_getSZ(2);
  @$pb.TagNumber(3)
  set code($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCode() => $_has(2);
  @$pb.TagNumber(3)
  void clearCode() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get message => $_getSZ(3);
  @$pb.TagNumber(4)
  set message($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearMessage() => $_clearField(4);
}

class AgentReady extends $pb.GeneratedMessage {
  factory AgentReady({
    $fixnum.Int64? generation,
    $0.HeartbeatPolicy? heartbeat,
    $3.DaemonStateRecord? daemonState,
  }) {
    final result = create();
    if (generation != null) result.generation = generation;
    if (heartbeat != null) result.heartbeat = heartbeat;
    if (daemonState != null) result.daemonState = daemonState;
    return result;
  }

  AgentReady._();

  factory AgentReady.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentReady.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentReady',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.HeartbeatPolicy>(2, _omitFieldNames ? '' : 'heartbeat',
        subBuilder: $0.HeartbeatPolicy.create)
    ..aOM<$3.DaemonStateRecord>(3, _omitFieldNames ? '' : 'daemonState',
        subBuilder: $3.DaemonStateRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentReady clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentReady copyWith(void Function(AgentReady) updates) =>
      super.copyWith((message) => updates(message as AgentReady)) as AgentReady;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentReady create() => AgentReady._();
  @$core.override
  AgentReady createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentReady getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentReady>(create);
  static AgentReady? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get generation => $_getI64(0);
  @$pb.TagNumber(1)
  set generation($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasGeneration() => $_has(0);
  @$pb.TagNumber(1)
  void clearGeneration() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.HeartbeatPolicy get heartbeat => $_getN(1);
  @$pb.TagNumber(2)
  set heartbeat($0.HeartbeatPolicy value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasHeartbeat() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeartbeat() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.HeartbeatPolicy ensureHeartbeat() => $_ensure(1);

  @$pb.TagNumber(3)
  $3.DaemonStateRecord get daemonState => $_getN(2);
  @$pb.TagNumber(3)
  set daemonState($3.DaemonStateRecord value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonState() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonState() => $_clearField(3);
  @$pb.TagNumber(3)
  $3.DaemonStateRecord ensureDaemonState() => $_ensure(2);
}

/// DaemonLifecycleCommand 要求当前 Agent generation 应用最新持久状态。
class DaemonLifecycleCommand extends $pb.GeneratedMessage {
  factory DaemonLifecycleCommand({
    $3.DaemonStateRecord? daemonState,
    $fixnum.Int64? agentGeneration,
  }) {
    final result = create();
    if (daemonState != null) result.daemonState = daemonState;
    if (agentGeneration != null) result.agentGeneration = agentGeneration;
    return result;
  }

  DaemonLifecycleCommand._();

  factory DaemonLifecycleCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonLifecycleCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonLifecycleCommand',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$3.DaemonStateRecord>(1, _omitFieldNames ? '' : 'daemonState',
        subBuilder: $3.DaemonStateRecord.create)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'agentGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonLifecycleCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonLifecycleCommand copyWith(
          void Function(DaemonLifecycleCommand) updates) =>
      super.copyWith((message) => updates(message as DaemonLifecycleCommand))
          as DaemonLifecycleCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonLifecycleCommand create() => DaemonLifecycleCommand._();
  @$core.override
  DaemonLifecycleCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonLifecycleCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonLifecycleCommand>(create);
  static DaemonLifecycleCommand? _defaultInstance;

  @$pb.TagNumber(1)
  $3.DaemonStateRecord get daemonState => $_getN(0);
  @$pb.TagNumber(1)
  set daemonState($3.DaemonStateRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonState() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonState() => $_clearField(1);
  @$pb.TagNumber(1)
  $3.DaemonStateRecord ensureDaemonState() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get agentGeneration => $_getI64(1);
  @$pb.TagNumber(2)
  set agentGeneration($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAgentGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearAgentGeneration() => $_clearField(2);
}

/// DaemonLifecycleResult 在 Cloud peers 已按状态收敛后返回。
class DaemonLifecycleResult extends $pb.GeneratedMessage {
  factory DaemonLifecycleResult({
    $3.DaemonStateRecord? daemonState,
    $fixnum.Int64? agentGeneration,
    $core.bool? applied,
    $core.String? errorMessage,
  }) {
    final result = create();
    if (daemonState != null) result.daemonState = daemonState;
    if (agentGeneration != null) result.agentGeneration = agentGeneration;
    if (applied != null) result.applied = applied;
    if (errorMessage != null) result.errorMessage = errorMessage;
    return result;
  }

  DaemonLifecycleResult._();

  factory DaemonLifecycleResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonLifecycleResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonLifecycleResult',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$3.DaemonStateRecord>(1, _omitFieldNames ? '' : 'daemonState',
        subBuilder: $3.DaemonStateRecord.create)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'agentGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(3, _omitFieldNames ? '' : 'applied')
    ..aOS(4, _omitFieldNames ? '' : 'errorMessage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonLifecycleResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonLifecycleResult copyWith(
          void Function(DaemonLifecycleResult) updates) =>
      super.copyWith((message) => updates(message as DaemonLifecycleResult))
          as DaemonLifecycleResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonLifecycleResult create() => DaemonLifecycleResult._();
  @$core.override
  DaemonLifecycleResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonLifecycleResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonLifecycleResult>(create);
  static DaemonLifecycleResult? _defaultInstance;

  @$pb.TagNumber(1)
  $3.DaemonStateRecord get daemonState => $_getN(0);
  @$pb.TagNumber(1)
  set daemonState($3.DaemonStateRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonState() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonState() => $_clearField(1);
  @$pb.TagNumber(1)
  $3.DaemonStateRecord ensureDaemonState() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get agentGeneration => $_getI64(1);
  @$pb.TagNumber(2)
  set agentGeneration($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAgentGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearAgentGeneration() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get applied => $_getBF(2);
  @$pb.TagNumber(3)
  set applied($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasApplied() => $_has(2);
  @$pb.TagNumber(3)
  void clearApplied() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get errorMessage => $_getSZ(3);
  @$pb.TagNumber(4)
  set errorMessage($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasErrorMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearErrorMessage() => $_clearField(4);
}

/// DaemonEdgeReselectCommand 让 daemon 立即测速并刷新 binding；不会重启 daemon 进程。
class DaemonEdgeReselectCommand extends $pb.GeneratedMessage {
  factory DaemonEdgeReselectCommand({
    $fixnum.Int64? agentGeneration,
    $fixnum.Int64? preferenceRevision,
  }) {
    final result = create();
    if (agentGeneration != null) result.agentGeneration = agentGeneration;
    if (preferenceRevision != null)
      result.preferenceRevision = preferenceRevision;
    return result;
  }

  DaemonEdgeReselectCommand._();

  factory DaemonEdgeReselectCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonEdgeReselectCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonEdgeReselectCommand',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'agentGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'preferenceRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeReselectCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeReselectCommand copyWith(
          void Function(DaemonEdgeReselectCommand) updates) =>
      super.copyWith((message) => updates(message as DaemonEdgeReselectCommand))
          as DaemonEdgeReselectCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonEdgeReselectCommand create() => DaemonEdgeReselectCommand._();
  @$core.override
  DaemonEdgeReselectCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonEdgeReselectCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonEdgeReselectCommand>(create);
  static DaemonEdgeReselectCommand? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get agentGeneration => $_getI64(0);
  @$pb.TagNumber(1)
  set agentGeneration($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAgentGeneration() => $_has(0);
  @$pb.TagNumber(1)
  void clearAgentGeneration() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get preferenceRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set preferenceRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPreferenceRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearPreferenceRevision() => $_clearField(2);
}

enum AgentEvent_Payload {
  hello,
  heartbeat,
  answer,
  rejected,
  authorization,
  lifecycleResult,
  notSet
}

class AgentEvent extends $pb.GeneratedMessage {
  factory AgentEvent({
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? senderId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? streamSeq,
    $4.Timestamp? sentAt,
    AgentHello? hello,
    AgentHeartbeat? heartbeat,
    AgentAnswer? answer,
    AgentSignalRejected? rejected,
    AgentAuthorizationResult? authorization,
    DaemonLifecycleResult? lifecycleResult,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (messageId != null) result.messageId = messageId;
    if (senderId != null) result.senderId = senderId;
    if (bootId != null) result.bootId = bootId;
    if (connectionId != null) result.connectionId = connectionId;
    if (streamSeq != null) result.streamSeq = streamSeq;
    if (sentAt != null) result.sentAt = sentAt;
    if (hello != null) result.hello = hello;
    if (heartbeat != null) result.heartbeat = heartbeat;
    if (answer != null) result.answer = answer;
    if (rejected != null) result.rejected = rejected;
    if (authorization != null) result.authorization = authorization;
    if (lifecycleResult != null) result.lifecycleResult = lifecycleResult;
    return result;
  }

  AgentEvent._();

  factory AgentEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, AgentEvent_Payload>
      _AgentEvent_PayloadByTag = {
    20: AgentEvent_Payload.hello,
    21: AgentEvent_Payload.heartbeat,
    22: AgentEvent_Payload.answer,
    23: AgentEvent_Payload.rejected,
    24: AgentEvent_Payload.authorization,
    25: AgentEvent_Payload.lifecycleResult,
    0: AgentEvent_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [20, 21, 22, 23, 24, 25])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'messageId')
    ..aOS(3, _omitFieldNames ? '' : 'senderId')
    ..aOS(4, _omitFieldNames ? '' : 'bootId')
    ..aOS(5, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$4.Timestamp>(7, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $4.Timestamp.create)
    ..aOM<AgentHello>(20, _omitFieldNames ? '' : 'hello',
        subBuilder: AgentHello.create)
    ..aOM<AgentHeartbeat>(21, _omitFieldNames ? '' : 'heartbeat',
        subBuilder: AgentHeartbeat.create)
    ..aOM<AgentAnswer>(22, _omitFieldNames ? '' : 'answer',
        subBuilder: AgentAnswer.create)
    ..aOM<AgentSignalRejected>(23, _omitFieldNames ? '' : 'rejected',
        subBuilder: AgentSignalRejected.create)
    ..aOM<AgentAuthorizationResult>(24, _omitFieldNames ? '' : 'authorization',
        subBuilder: AgentAuthorizationResult.create)
    ..aOM<DaemonLifecycleResult>(25, _omitFieldNames ? '' : 'lifecycleResult',
        subBuilder: DaemonLifecycleResult.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentEvent copyWith(void Function(AgentEvent) updates) =>
      super.copyWith((message) => updates(message as AgentEvent)) as AgentEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentEvent create() => AgentEvent._();
  @$core.override
  AgentEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentEvent>(create);
  static AgentEvent? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  AgentEvent_Payload whichPayload() =>
      _AgentEvent_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.int get protocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set protocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get messageId => $_getSZ(1);
  @$pb.TagNumber(2)
  set messageId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessageId() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessageId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get senderId => $_getSZ(2);
  @$pb.TagNumber(3)
  set senderId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSenderId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSenderId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get bootId => $_getSZ(3);
  @$pb.TagNumber(4)
  set bootId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBootId() => $_has(3);
  @$pb.TagNumber(4)
  void clearBootId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get connectionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set connectionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasConnectionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearConnectionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get streamSeq => $_getI64(5);
  @$pb.TagNumber(6)
  set streamSeq($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasStreamSeq() => $_has(5);
  @$pb.TagNumber(6)
  void clearStreamSeq() => $_clearField(6);

  @$pb.TagNumber(7)
  $4.Timestamp get sentAt => $_getN(6);
  @$pb.TagNumber(7)
  set sentAt($4.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSentAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearSentAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $4.Timestamp ensureSentAt() => $_ensure(6);

  @$pb.TagNumber(20)
  AgentHello get hello => $_getN(7);
  @$pb.TagNumber(20)
  set hello(AgentHello value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasHello() => $_has(7);
  @$pb.TagNumber(20)
  void clearHello() => $_clearField(20);
  @$pb.TagNumber(20)
  AgentHello ensureHello() => $_ensure(7);

  @$pb.TagNumber(21)
  AgentHeartbeat get heartbeat => $_getN(8);
  @$pb.TagNumber(21)
  set heartbeat(AgentHeartbeat value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasHeartbeat() => $_has(8);
  @$pb.TagNumber(21)
  void clearHeartbeat() => $_clearField(21);
  @$pb.TagNumber(21)
  AgentHeartbeat ensureHeartbeat() => $_ensure(8);

  @$pb.TagNumber(22)
  AgentAnswer get answer => $_getN(9);
  @$pb.TagNumber(22)
  set answer(AgentAnswer value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasAnswer() => $_has(9);
  @$pb.TagNumber(22)
  void clearAnswer() => $_clearField(22);
  @$pb.TagNumber(22)
  AgentAnswer ensureAnswer() => $_ensure(9);

  @$pb.TagNumber(23)
  AgentSignalRejected get rejected => $_getN(10);
  @$pb.TagNumber(23)
  set rejected(AgentSignalRejected value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasRejected() => $_has(10);
  @$pb.TagNumber(23)
  void clearRejected() => $_clearField(23);
  @$pb.TagNumber(23)
  AgentSignalRejected ensureRejected() => $_ensure(10);

  @$pb.TagNumber(24)
  AgentAuthorizationResult get authorization => $_getN(11);
  @$pb.TagNumber(24)
  set authorization(AgentAuthorizationResult value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasAuthorization() => $_has(11);
  @$pb.TagNumber(24)
  void clearAuthorization() => $_clearField(24);
  @$pb.TagNumber(24)
  AgentAuthorizationResult ensureAuthorization() => $_ensure(11);

  @$pb.TagNumber(25)
  DaemonLifecycleResult get lifecycleResult => $_getN(12);
  @$pb.TagNumber(25)
  set lifecycleResult(DaemonLifecycleResult value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasLifecycleResult() => $_has(12);
  @$pb.TagNumber(25)
  void clearLifecycleResult() => $_clearField(25);
  @$pb.TagNumber(25)
  DaemonLifecycleResult ensureLifecycleResult() => $_ensure(12);
}

enum EdgeCommand_Payload {
  ready,
  offer,
  authorize,
  challenge,
  lifecycle,
  edgeReselect,
  notSet
}

class EdgeCommand extends $pb.GeneratedMessage {
  factory EdgeCommand({
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? senderId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? streamSeq,
    $4.Timestamp? sentAt,
    AgentReady? ready,
    AgentOffer? offer,
    AgentAuthorize? authorize,
    $0.EdgeChallenge? challenge,
    DaemonLifecycleCommand? lifecycle,
    DaemonEdgeReselectCommand? edgeReselect,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (messageId != null) result.messageId = messageId;
    if (senderId != null) result.senderId = senderId;
    if (bootId != null) result.bootId = bootId;
    if (connectionId != null) result.connectionId = connectionId;
    if (streamSeq != null) result.streamSeq = streamSeq;
    if (sentAt != null) result.sentAt = sentAt;
    if (ready != null) result.ready = ready;
    if (offer != null) result.offer = offer;
    if (authorize != null) result.authorize = authorize;
    if (challenge != null) result.challenge = challenge;
    if (lifecycle != null) result.lifecycle = lifecycle;
    if (edgeReselect != null) result.edgeReselect = edgeReselect;
    return result;
  }

  EdgeCommand._();

  factory EdgeCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EdgeCommand_Payload>
      _EdgeCommand_PayloadByTag = {
    20: EdgeCommand_Payload.ready,
    21: EdgeCommand_Payload.offer,
    22: EdgeCommand_Payload.authorize,
    23: EdgeCommand_Payload.challenge,
    24: EdgeCommand_Payload.lifecycle,
    25: EdgeCommand_Payload.edgeReselect,
    0: EdgeCommand_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeCommand',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [20, 21, 22, 23, 24, 25])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'messageId')
    ..aOS(3, _omitFieldNames ? '' : 'senderId')
    ..aOS(4, _omitFieldNames ? '' : 'bootId')
    ..aOS(5, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$4.Timestamp>(7, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $4.Timestamp.create)
    ..aOM<AgentReady>(20, _omitFieldNames ? '' : 'ready',
        subBuilder: AgentReady.create)
    ..aOM<AgentOffer>(21, _omitFieldNames ? '' : 'offer',
        subBuilder: AgentOffer.create)
    ..aOM<AgentAuthorize>(22, _omitFieldNames ? '' : 'authorize',
        subBuilder: AgentAuthorize.create)
    ..aOM<$0.EdgeChallenge>(23, _omitFieldNames ? '' : 'challenge',
        subBuilder: $0.EdgeChallenge.create)
    ..aOM<DaemonLifecycleCommand>(24, _omitFieldNames ? '' : 'lifecycle',
        subBuilder: DaemonLifecycleCommand.create)
    ..aOM<DaemonEdgeReselectCommand>(25, _omitFieldNames ? '' : 'edgeReselect',
        subBuilder: DaemonEdgeReselectCommand.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeCommand copyWith(void Function(EdgeCommand) updates) =>
      super.copyWith((message) => updates(message as EdgeCommand))
          as EdgeCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeCommand create() => EdgeCommand._();
  @$core.override
  EdgeCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeCommand>(create);
  static EdgeCommand? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  EdgeCommand_Payload whichPayload() =>
      _EdgeCommand_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.int get protocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set protocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get messageId => $_getSZ(1);
  @$pb.TagNumber(2)
  set messageId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessageId() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessageId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get senderId => $_getSZ(2);
  @$pb.TagNumber(3)
  set senderId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSenderId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSenderId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get bootId => $_getSZ(3);
  @$pb.TagNumber(4)
  set bootId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBootId() => $_has(3);
  @$pb.TagNumber(4)
  void clearBootId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get connectionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set connectionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasConnectionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearConnectionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get streamSeq => $_getI64(5);
  @$pb.TagNumber(6)
  set streamSeq($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasStreamSeq() => $_has(5);
  @$pb.TagNumber(6)
  void clearStreamSeq() => $_clearField(6);

  @$pb.TagNumber(7)
  $4.Timestamp get sentAt => $_getN(6);
  @$pb.TagNumber(7)
  set sentAt($4.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSentAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearSentAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $4.Timestamp ensureSentAt() => $_ensure(6);

  @$pb.TagNumber(20)
  AgentReady get ready => $_getN(7);
  @$pb.TagNumber(20)
  set ready(AgentReady value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasReady() => $_has(7);
  @$pb.TagNumber(20)
  void clearReady() => $_clearField(20);
  @$pb.TagNumber(20)
  AgentReady ensureReady() => $_ensure(7);

  @$pb.TagNumber(21)
  AgentOffer get offer => $_getN(8);
  @$pb.TagNumber(21)
  set offer(AgentOffer value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasOffer() => $_has(8);
  @$pb.TagNumber(21)
  void clearOffer() => $_clearField(21);
  @$pb.TagNumber(21)
  AgentOffer ensureOffer() => $_ensure(8);

  @$pb.TagNumber(22)
  AgentAuthorize get authorize => $_getN(9);
  @$pb.TagNumber(22)
  set authorize(AgentAuthorize value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasAuthorize() => $_has(9);
  @$pb.TagNumber(22)
  void clearAuthorize() => $_clearField(22);
  @$pb.TagNumber(22)
  AgentAuthorize ensureAuthorize() => $_ensure(9);

  @$pb.TagNumber(23)
  $0.EdgeChallenge get challenge => $_getN(10);
  @$pb.TagNumber(23)
  set challenge($0.EdgeChallenge value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasChallenge() => $_has(10);
  @$pb.TagNumber(23)
  void clearChallenge() => $_clearField(23);
  @$pb.TagNumber(23)
  $0.EdgeChallenge ensureChallenge() => $_ensure(10);

  @$pb.TagNumber(24)
  DaemonLifecycleCommand get lifecycle => $_getN(11);
  @$pb.TagNumber(24)
  set lifecycle(DaemonLifecycleCommand value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasLifecycle() => $_has(11);
  @$pb.TagNumber(24)
  void clearLifecycle() => $_clearField(24);
  @$pb.TagNumber(24)
  DaemonLifecycleCommand ensureLifecycle() => $_ensure(11);

  @$pb.TagNumber(25)
  DaemonEdgeReselectCommand get edgeReselect => $_getN(12);
  @$pb.TagNumber(25)
  set edgeReselect(DaemonEdgeReselectCommand value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasEdgeReselect() => $_has(12);
  @$pb.TagNumber(25)
  void clearEdgeReselect() => $_clearField(25);
  @$pb.TagNumber(25)
  DaemonEdgeReselectCommand ensureEdgeReselect() => $_ensure(12);
}

/// AgentGateway 是 daemon 到单个 Edge 的唯一长连接控制流。
class AgentGatewayApi {
  final $pb.RpcClient _client;

  AgentGatewayApi(this._client);

  $async.Future<EdgeCommand> connect(
          $pb.ClientContext? ctx, AgentEvent request) =>
      _client.invoke<EdgeCommand>(
          ctx, 'AgentGateway', 'Connect', request, EdgeCommand());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
