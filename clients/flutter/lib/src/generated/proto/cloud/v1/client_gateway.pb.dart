// This is a generated file - do not edit.
//
// Generated from cloud/v1/client_gateway.proto.

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
    as $2;

import 'client_gateway.pbenum.dart';
import 'common.pb.dart' as $0;
import 'runtime.pbenum.dart' as $3;
import 'usage.pb.dart' as $1;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'client_gateway.pbenum.dart';

/// CloudICECandidate 是 Cloud 信令的中性 candidate；不携带授权或 terminal 数据。
class CloudICECandidate extends $pb.GeneratedMessage {
  factory CloudICECandidate({
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

  CloudICECandidate._();

  factory CloudICECandidate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudICECandidate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudICECandidate',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'candidate')
    ..aOS(2, _omitFieldNames ? '' : 'sdpMid')
    ..aI(3, _omitFieldNames ? '' : 'sdpMlineIndex',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(4, _omitFieldNames ? '' : 'usernameFragment')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudICECandidate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudICECandidate copyWith(void Function(CloudICECandidate) updates) =>
      super.copyWith((message) => updates(message as CloudICECandidate))
          as CloudICECandidate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudICECandidate create() => CloudICECandidate._();
  @$core.override
  CloudICECandidate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudICECandidate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudICECandidate>(create);
  static CloudICECandidate? _defaultInstance;

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

/// PairingAdmission 是二维码材料的网络投影。Edge 必须将其与当前在线 daemon binding 对齐，
/// 再向该 daemon 实时请求授权；它自身不是 grant，也不能兑换 capability。
class PairingAdmission extends $pb.GeneratedMessage {
  factory PairingAdmission({
    $core.String? daemonId,
    $core.String? deviceId,
    $core.List<$core.int>? devicePublicKey,
    $core.List<$core.int>? pairingClaimSha256,
    $fixnum.Int64? expiresAtUnixNano,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (deviceId != null) result.deviceId = deviceId;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    if (pairingClaimSha256 != null)
      result.pairingClaimSha256 = pairingClaimSha256;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    return result;
  }

  PairingAdmission._();

  factory PairingAdmission.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingAdmission.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingAdmission',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'pairingClaimSha256', $pb.PbFieldType.OY)
    ..aInt64(5, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingAdmission clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingAdmission copyWith(void Function(PairingAdmission) updates) =>
      super.copyWith((message) => updates(message as PairingAdmission))
          as PairingAdmission;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingAdmission create() => PairingAdmission._();
  @$core.override
  PairingAdmission createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingAdmission getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingAdmission>(create);
  static PairingAdmission? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get devicePublicKey => $_getN(2);
  @$pb.TagNumber(3)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDevicePublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearDevicePublicKey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get pairingClaimSha256 => $_getN(3);
  @$pb.TagNumber(4)
  set pairingClaimSha256($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPairingClaimSha256() => $_has(3);
  @$pb.TagNumber(4)
  void clearPairingClaimSha256() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(4);
  @$pb.TagNumber(5)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasExpiresAtUnixNano() => $_has(4);
  @$pb.TagNumber(5)
  void clearExpiresAtUnixNano() => $_clearField(5);
}

enum ClientHello_Authorization { cloudRouteGrant, pairingAdmission, notSet }

class ClientHello extends $pb.GeneratedMessage {
  factory ClientHello({
    $core.List<$core.int>? clientPublicKey,
    $core.List<$core.int>? clientProof,
    $3.ClientProduct? product,
    $core.String? softwareVersion,
    $fixnum.Int64? attemptGeneration,
    $1.RelayPreference? relayPreference,
    $core.bool? presenceProbe,
    $0.SignedEnvelope? cloudRouteGrant,
    PairingAdmission? pairingAdmission,
  }) {
    final result = create();
    if (clientPublicKey != null) result.clientPublicKey = clientPublicKey;
    if (clientProof != null) result.clientProof = clientProof;
    if (product != null) result.product = product;
    if (softwareVersion != null) result.softwareVersion = softwareVersion;
    if (attemptGeneration != null) result.attemptGeneration = attemptGeneration;
    if (relayPreference != null) result.relayPreference = relayPreference;
    if (presenceProbe != null) result.presenceProbe = presenceProbe;
    if (cloudRouteGrant != null) result.cloudRouteGrant = cloudRouteGrant;
    if (pairingAdmission != null) result.pairingAdmission = pairingAdmission;
    return result;
  }

  ClientHello._();

  factory ClientHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ClientHello_Authorization>
      _ClientHello_AuthorizationByTag = {
    10: ClientHello_Authorization.cloudRouteGrant,
    11: ClientHello_Authorization.pairingAdmission,
    0: ClientHello_Authorization.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientHello',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11])
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'clientPublicKey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'clientProof', $pb.PbFieldType.OY)
    ..aE<$3.ClientProduct>(4, _omitFieldNames ? '' : 'product',
        enumValues: $3.ClientProduct.values)
    ..aOS(5, _omitFieldNames ? '' : 'softwareVersion')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'attemptGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<$1.RelayPreference>(7, _omitFieldNames ? '' : 'relayPreference',
        enumValues: $1.RelayPreference.values)
    ..aOB(8, _omitFieldNames ? '' : 'presenceProbe')
    ..aOM<$0.SignedEnvelope>(10, _omitFieldNames ? '' : 'cloudRouteGrant',
        subBuilder: $0.SignedEnvelope.create)
    ..aOM<PairingAdmission>(11, _omitFieldNames ? '' : 'pairingAdmission',
        subBuilder: PairingAdmission.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientHello copyWith(void Function(ClientHello) updates) =>
      super.copyWith((message) => updates(message as ClientHello))
          as ClientHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientHello create() => ClientHello._();
  @$core.override
  ClientHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientHello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientHello>(create);
  static ClientHello? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  ClientHello_Authorization whichAuthorization() =>
      _ClientHello_AuthorizationByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  void clearAuthorization() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(2)
  $core.List<$core.int> get clientPublicKey => $_getN(0);
  @$pb.TagNumber(2)
  set clientPublicKey($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(2)
  $core.bool hasClientPublicKey() => $_has(0);
  @$pb.TagNumber(2)
  void clearClientPublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get clientProof => $_getN(1);
  @$pb.TagNumber(3)
  set clientProof($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(3)
  $core.bool hasClientProof() => $_has(1);
  @$pb.TagNumber(3)
  void clearClientProof() => $_clearField(3);

  @$pb.TagNumber(4)
  $3.ClientProduct get product => $_getN(2);
  @$pb.TagNumber(4)
  set product($3.ClientProduct value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasProduct() => $_has(2);
  @$pb.TagNumber(4)
  void clearProduct() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get softwareVersion => $_getSZ(3);
  @$pb.TagNumber(5)
  set softwareVersion($core.String value) => $_setString(3, value);
  @$pb.TagNumber(5)
  $core.bool hasSoftwareVersion() => $_has(3);
  @$pb.TagNumber(5)
  void clearSoftwareVersion() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get attemptGeneration => $_getI64(4);
  @$pb.TagNumber(6)
  set attemptGeneration($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(6)
  $core.bool hasAttemptGeneration() => $_has(4);
  @$pb.TagNumber(6)
  void clearAttemptGeneration() => $_clearField(6);

  @$pb.TagNumber(7)
  $1.RelayPreference get relayPreference => $_getN(5);
  @$pb.TagNumber(7)
  set relayPreference($1.RelayPreference value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasRelayPreference() => $_has(5);
  @$pb.TagNumber(7)
  void clearRelayPreference() => $_clearField(7);

  /// presence_probe only verifies this daemon's live AgentGateway presence.
  /// It never allocates a ClientSession, Relay reservation, or WebRTC peer.
  @$pb.TagNumber(8)
  $core.bool get presenceProbe => $_getBF(6);
  @$pb.TagNumber(8)
  set presenceProbe($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(8)
  $core.bool hasPresenceProbe() => $_has(6);
  @$pb.TagNumber(8)
  void clearPresenceProbe() => $_clearField(8);

  /// cloud_route_grant 是 owning daemon 签发给已配对 ClientAccessIdentity 的长期发现和信令授权。
  @$pb.TagNumber(10)
  $0.SignedEnvelope get cloudRouteGrant => $_getN(7);
  @$pb.TagNumber(10)
  set cloudRouteGrant($0.SignedEnvelope value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasCloudRouteGrant() => $_has(7);
  @$pb.TagNumber(10)
  void clearCloudRouteGrant() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.SignedEnvelope ensureCloudRouteGrant() => $_ensure(7);

  /// pairing_admission 只允许 Edge 向在线 daemon 请求一次性 pairing 预检。
  @$pb.TagNumber(11)
  PairingAdmission get pairingAdmission => $_getN(8);
  @$pb.TagNumber(11)
  set pairingAdmission(PairingAdmission value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasPairingAdmission() => $_has(8);
  @$pb.TagNumber(11)
  void clearPairingAdmission() => $_clearField(11);
  @$pb.TagNumber(11)
  PairingAdmission ensurePairingAdmission() => $_ensure(8);
}

class ClientReady extends $pb.GeneratedMessage {
  factory ClientReady({
    $core.String? sessionId,
    $fixnum.Int64? generation,
    $1.RelayICEConfig? relay,
    $0.CloudEntitlementFailure? relayFailure,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (generation != null) result.generation = generation;
    if (relay != null) result.relay = relay;
    if (relayFailure != null) result.relayFailure = relayFailure;
    return result;
  }

  ClientReady._();

  factory ClientReady.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientReady.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientReady',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$1.RelayICEConfig>(3, _omitFieldNames ? '' : 'relay',
        subBuilder: $1.RelayICEConfig.create)
    ..aOM<$0.CloudEntitlementFailure>(4, _omitFieldNames ? '' : 'relayFailure',
        subBuilder: $0.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientReady clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientReady copyWith(void Function(ClientReady) updates) =>
      super.copyWith((message) => updates(message as ClientReady))
          as ClientReady;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientReady create() => ClientReady._();
  @$core.override
  ClientReady createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientReady getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientReady>(create);
  static ClientReady? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get generation => $_getI64(1);
  @$pb.TagNumber(2)
  set generation($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearGeneration() => $_clearField(2);

  @$pb.TagNumber(3)
  $1.RelayICEConfig get relay => $_getN(2);
  @$pb.TagNumber(3)
  set relay($1.RelayICEConfig value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRelay() => $_has(2);
  @$pb.TagNumber(3)
  void clearRelay() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.RelayICEConfig ensureRelay() => $_ensure(2);

  /// AUTO 可以继续纯 P2P；只有 P2P 最终也失败时客户端才展示该原因。
  @$pb.TagNumber(4)
  $0.CloudEntitlementFailure get relayFailure => $_getN(3);
  @$pb.TagNumber(4)
  set relayFailure($0.CloudEntitlementFailure value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRelayFailure() => $_has(3);
  @$pb.TagNumber(4)
  void clearRelayFailure() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.CloudEntitlementFailure ensureRelayFailure() => $_ensure(3);
}

/// ClientPathDecision commits or abandons one authenticated candidate. The
/// decision_id is stable across same-stream retries and cannot be reused for a
/// different decision.
class ClientPathDecision extends $pb.GeneratedMessage {
  factory ClientPathDecision({
    $core.String? sessionId,
    $core.String? decisionId,
    CloudPathDecision? decision,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (decisionId != null) result.decisionId = decisionId;
    if (decision != null) result.decision = decision;
    return result;
  }

  ClientPathDecision._();

  factory ClientPathDecision.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientPathDecision.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientPathDecision',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'decisionId')
    ..aE<CloudPathDecision>(3, _omitFieldNames ? '' : 'decision',
        enumValues: CloudPathDecision.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientPathDecision clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientPathDecision copyWith(void Function(ClientPathDecision) updates) =>
      super.copyWith((message) => updates(message as ClientPathDecision))
          as ClientPathDecision;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientPathDecision create() => ClientPathDecision._();
  @$core.override
  ClientPathDecision createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientPathDecision getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientPathDecision>(create);
  static ClientPathDecision? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get decisionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set decisionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDecisionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDecisionId() => $_clearField(2);

  @$pb.TagNumber(3)
  CloudPathDecision get decision => $_getN(2);
  @$pb.TagNumber(3)
  set decision(CloudPathDecision value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDecision() => $_has(2);
  @$pb.TagNumber(3)
  void clearDecision() => $_clearField(3);
}

/// EdgePathDecisionAck is the only decision barrier accepted by the client.
/// Confirm freezes the chosen path without destructively releasing standby
/// Relay state because peer-reflexive ICE candidates cannot prove that TURN is
/// unused. ABANDON is acknowledged only after Relay and runtime cleanup.
class EdgePathDecisionAck extends $pb.GeneratedMessage {
  factory EdgePathDecisionAck({
    $core.String? sessionId,
    $core.String? decisionId,
    CloudPathDecision? decision,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (decisionId != null) result.decisionId = decisionId;
    if (decision != null) result.decision = decision;
    return result;
  }

  EdgePathDecisionAck._();

  factory EdgePathDecisionAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgePathDecisionAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgePathDecisionAck',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'decisionId')
    ..aE<CloudPathDecision>(3, _omitFieldNames ? '' : 'decision',
        enumValues: CloudPathDecision.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePathDecisionAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePathDecisionAck copyWith(void Function(EdgePathDecisionAck) updates) =>
      super.copyWith((message) => updates(message as EdgePathDecisionAck))
          as EdgePathDecisionAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgePathDecisionAck create() => EdgePathDecisionAck._();
  @$core.override
  EdgePathDecisionAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgePathDecisionAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgePathDecisionAck>(create);
  static EdgePathDecisionAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get decisionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set decisionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDecisionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDecisionId() => $_clearField(2);

  @$pb.TagNumber(3)
  CloudPathDecision get decision => $_getN(2);
  @$pb.TagNumber(3)
  set decision(CloudPathDecision value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDecision() => $_has(2);
  @$pb.TagNumber(3)
  void clearDecision() => $_clearField(3);
}

/// ClientSessionRelease is a separate teardown transaction for a path that was
/// already confirmed. It must not reuse or mutate the immutable PathDecision.
class ClientSessionRelease extends $pb.GeneratedMessage {
  factory ClientSessionRelease({
    $core.String? sessionId,
    $core.String? releaseId,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (releaseId != null) result.releaseId = releaseId;
    return result;
  }

  ClientSessionRelease._();

  factory ClientSessionRelease.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientSessionRelease.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientSessionRelease',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'releaseId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSessionRelease clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSessionRelease copyWith(void Function(ClientSessionRelease) updates) =>
      super.copyWith((message) => updates(message as ClientSessionRelease))
          as ClientSessionRelease;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientSessionRelease create() => ClientSessionRelease._();
  @$core.override
  ClientSessionRelease createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientSessionRelease getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientSessionRelease>(create);
  static ClientSessionRelease? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get releaseId => $_getSZ(1);
  @$pb.TagNumber(2)
  set releaseId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReleaseId() => $_has(1);
  @$pb.TagNumber(2)
  void clearReleaseId() => $_clearField(2);
}

/// EdgeSessionReleaseAck is emitted only after Relay and runtime session
/// cleanup completed. EOF and administrative close are not release ACKs.
class EdgeSessionReleaseAck extends $pb.GeneratedMessage {
  factory EdgeSessionReleaseAck({
    $core.String? sessionId,
    $core.String? releaseId,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (releaseId != null) result.releaseId = releaseId;
    return result;
  }

  EdgeSessionReleaseAck._();

  factory EdgeSessionReleaseAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeSessionReleaseAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeSessionReleaseAck',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'releaseId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeSessionReleaseAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeSessionReleaseAck copyWith(
          void Function(EdgeSessionReleaseAck) updates) =>
      super.copyWith((message) => updates(message as EdgeSessionReleaseAck))
          as EdgeSessionReleaseAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeSessionReleaseAck create() => EdgeSessionReleaseAck._();
  @$core.override
  EdgeSessionReleaseAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeSessionReleaseAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeSessionReleaseAck>(create);
  static EdgeSessionReleaseAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get releaseId => $_getSZ(1);
  @$pb.TagNumber(2)
  set releaseId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReleaseId() => $_has(1);
  @$pb.TagNumber(2)
  void clearReleaseId() => $_clearField(2);
}

class ClientOffer extends $pb.GeneratedMessage {
  factory ClientOffer({
    $core.String? sessionId,
    $core.String? offerSdp,
    $core.Iterable<CloudICECandidate>? candidates,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (offerSdp != null) result.offerSdp = offerSdp;
    if (candidates != null) result.candidates.addAll(candidates);
    return result;
  }

  ClientOffer._();

  factory ClientOffer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientOffer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientOffer',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'offerSdp')
    ..pPM<CloudICECandidate>(3, _omitFieldNames ? '' : 'candidates',
        subBuilder: CloudICECandidate.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientOffer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientOffer copyWith(void Function(ClientOffer) updates) =>
      super.copyWith((message) => updates(message as ClientOffer))
          as ClientOffer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientOffer create() => ClientOffer._();
  @$core.override
  ClientOffer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientOffer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientOffer>(create);
  static ClientOffer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get offerSdp => $_getSZ(1);
  @$pb.TagNumber(2)
  set offerSdp($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOfferSdp() => $_has(1);
  @$pb.TagNumber(2)
  void clearOfferSdp() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<CloudICECandidate> get candidates => $_getList(2);
}

class EdgeAnswer extends $pb.GeneratedMessage {
  factory EdgeAnswer({
    $core.String? sessionId,
    $core.String? answerSdp,
    $core.Iterable<CloudICECandidate>? candidates,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (answerSdp != null) result.answerSdp = answerSdp;
    if (candidates != null) result.candidates.addAll(candidates);
    return result;
  }

  EdgeAnswer._();

  factory EdgeAnswer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeAnswer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeAnswer',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'answerSdp')
    ..pPM<CloudICECandidate>(3, _omitFieldNames ? '' : 'candidates',
        subBuilder: CloudICECandidate.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeAnswer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeAnswer copyWith(void Function(EdgeAnswer) updates) =>
      super.copyWith((message) => updates(message as EdgeAnswer)) as EdgeAnswer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeAnswer create() => EdgeAnswer._();
  @$core.override
  EdgeAnswer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeAnswer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeAnswer>(create);
  static EdgeAnswer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get answerSdp => $_getSZ(1);
  @$pb.TagNumber(2)
  set answerSdp($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAnswerSdp() => $_has(1);
  @$pb.TagNumber(2)
  void clearAnswerSdp() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<CloudICECandidate> get candidates => $_getList(2);
}

class SignalRejected extends $pb.GeneratedMessage {
  factory SignalRejected({
    $core.String? sessionId,
    $core.String? code,
    $core.String? message,
    $0.CloudEntitlementFailure? entitlementFailure,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    if (entitlementFailure != null)
      result.entitlementFailure = entitlementFailure;
    return result;
  }

  SignalRejected._();

  factory SignalRejected.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SignalRejected.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SignalRejected',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'code')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..aOM<$0.CloudEntitlementFailure>(
        4, _omitFieldNames ? '' : 'entitlementFailure',
        subBuilder: $0.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalRejected clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalRejected copyWith(void Function(SignalRejected) updates) =>
      super.copyWith((message) => updates(message as SignalRejected))
          as SignalRejected;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SignalRejected create() => SignalRejected._();
  @$core.override
  SignalRejected createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SignalRejected getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SignalRejected>(create);
  static SignalRejected? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get code => $_getSZ(1);
  @$pb.TagNumber(2)
  set code($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCode() => $_has(1);
  @$pb.TagNumber(2)
  void clearCode() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.CloudEntitlementFailure get entitlementFailure => $_getN(3);
  @$pb.TagNumber(4)
  set entitlementFailure($0.CloudEntitlementFailure value) =>
      $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasEntitlementFailure() => $_has(3);
  @$pb.TagNumber(4)
  void clearEntitlementFailure() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.CloudEntitlementFailure ensureEntitlementFailure() => $_ensure(3);
}

class SignalSessionClosed extends $pb.GeneratedMessage {
  factory SignalSessionClosed({
    $core.String? sessionId,
    SignalSessionCloseCode? code,
    $core.String? message,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  SignalSessionClosed._();

  factory SignalSessionClosed.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SignalSessionClosed.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SignalSessionClosed',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aE<SignalSessionCloseCode>(2, _omitFieldNames ? '' : 'code',
        enumValues: SignalSessionCloseCode.values)
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalSessionClosed clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalSessionClosed copyWith(void Function(SignalSessionClosed) updates) =>
      super.copyWith((message) => updates(message as SignalSessionClosed))
          as SignalSessionClosed;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SignalSessionClosed create() => SignalSessionClosed._();
  @$core.override
  SignalSessionClosed createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SignalSessionClosed getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SignalSessionClosed>(create);
  static SignalSessionClosed? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  SignalSessionCloseCode get code => $_getN(1);
  @$pb.TagNumber(2)
  set code(SignalSessionCloseCode value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCode() => $_has(1);
  @$pb.TagNumber(2)
  void clearCode() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
}

/// DaemonPresence is returned only after an authenticated presence probe.
/// online=false intentionally covers offline, stale, and invalid authorization
/// so the endpoint cannot be enumerated without a valid CloudRouteGrant.
class DaemonPresence extends $pb.GeneratedMessage {
  factory DaemonPresence({
    $core.bool? online,
  }) {
    final result = create();
    if (online != null) result.online = online;
    return result;
  }

  DaemonPresence._();

  factory DaemonPresence.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonPresence.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonPresence',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'online')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonPresence clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonPresence copyWith(void Function(DaemonPresence) updates) =>
      super.copyWith((message) => updates(message as DaemonPresence))
          as DaemonPresence;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonPresence create() => DaemonPresence._();
  @$core.override
  DaemonPresence createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonPresence getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonPresence>(create);
  static DaemonPresence? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get online => $_getBF(0);
  @$pb.TagNumber(1)
  set online($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOnline() => $_has(0);
  @$pb.TagNumber(1)
  void clearOnline() => $_clearField(1);
}

enum ClientSignal_Payload { hello, offer, pathDecision, sessionRelease, notSet }

class ClientSignal extends $pb.GeneratedMessage {
  factory ClientSignal({
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? senderId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? streamSeq,
    $2.Timestamp? sentAt,
    ClientHello? hello,
    ClientOffer? offer,
    ClientPathDecision? pathDecision,
    ClientSessionRelease? sessionRelease,
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
    if (offer != null) result.offer = offer;
    if (pathDecision != null) result.pathDecision = pathDecision;
    if (sessionRelease != null) result.sessionRelease = sessionRelease;
    return result;
  }

  ClientSignal._();

  factory ClientSignal.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientSignal.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ClientSignal_Payload>
      _ClientSignal_PayloadByTag = {
    20: ClientSignal_Payload.hello,
    21: ClientSignal_Payload.offer,
    22: ClientSignal_Payload.pathDecision,
    23: ClientSignal_Payload.sessionRelease,
    0: ClientSignal_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientSignal',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [20, 21, 22, 23])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'messageId')
    ..aOS(3, _omitFieldNames ? '' : 'senderId')
    ..aOS(4, _omitFieldNames ? '' : 'bootId')
    ..aOS(5, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$2.Timestamp>(7, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $2.Timestamp.create)
    ..aOM<ClientHello>(20, _omitFieldNames ? '' : 'hello',
        subBuilder: ClientHello.create)
    ..aOM<ClientOffer>(21, _omitFieldNames ? '' : 'offer',
        subBuilder: ClientOffer.create)
    ..aOM<ClientPathDecision>(22, _omitFieldNames ? '' : 'pathDecision',
        subBuilder: ClientPathDecision.create)
    ..aOM<ClientSessionRelease>(23, _omitFieldNames ? '' : 'sessionRelease',
        subBuilder: ClientSessionRelease.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSignal clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSignal copyWith(void Function(ClientSignal) updates) =>
      super.copyWith((message) => updates(message as ClientSignal))
          as ClientSignal;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientSignal create() => ClientSignal._();
  @$core.override
  ClientSignal createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientSignal getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientSignal>(create);
  static ClientSignal? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  ClientSignal_Payload whichPayload() =>
      _ClientSignal_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
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
  $2.Timestamp get sentAt => $_getN(6);
  @$pb.TagNumber(7)
  set sentAt($2.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSentAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearSentAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $2.Timestamp ensureSentAt() => $_ensure(6);

  @$pb.TagNumber(20)
  ClientHello get hello => $_getN(7);
  @$pb.TagNumber(20)
  set hello(ClientHello value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasHello() => $_has(7);
  @$pb.TagNumber(20)
  void clearHello() => $_clearField(20);
  @$pb.TagNumber(20)
  ClientHello ensureHello() => $_ensure(7);

  @$pb.TagNumber(21)
  ClientOffer get offer => $_getN(8);
  @$pb.TagNumber(21)
  set offer(ClientOffer value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasOffer() => $_has(8);
  @$pb.TagNumber(21)
  void clearOffer() => $_clearField(21);
  @$pb.TagNumber(21)
  ClientOffer ensureOffer() => $_ensure(8);

  @$pb.TagNumber(22)
  ClientPathDecision get pathDecision => $_getN(9);
  @$pb.TagNumber(22)
  set pathDecision(ClientPathDecision value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasPathDecision() => $_has(9);
  @$pb.TagNumber(22)
  void clearPathDecision() => $_clearField(22);
  @$pb.TagNumber(22)
  ClientPathDecision ensurePathDecision() => $_ensure(9);

  @$pb.TagNumber(23)
  ClientSessionRelease get sessionRelease => $_getN(10);
  @$pb.TagNumber(23)
  set sessionRelease(ClientSessionRelease value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasSessionRelease() => $_has(10);
  @$pb.TagNumber(23)
  void clearSessionRelease() => $_clearField(23);
  @$pb.TagNumber(23)
  ClientSessionRelease ensureSessionRelease() => $_ensure(10);
}

enum EdgeSignal_Payload {
  ready,
  answer,
  rejected,
  challenge,
  closed,
  presence,
  pathDecisionAck,
  sessionReleaseAck,
  notSet
}

class EdgeSignal extends $pb.GeneratedMessage {
  factory EdgeSignal({
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? senderId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? streamSeq,
    $2.Timestamp? sentAt,
    ClientReady? ready,
    EdgeAnswer? answer,
    SignalRejected? rejected,
    $0.EdgeChallenge? challenge,
    SignalSessionClosed? closed,
    DaemonPresence? presence,
    EdgePathDecisionAck? pathDecisionAck,
    EdgeSessionReleaseAck? sessionReleaseAck,
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
    if (answer != null) result.answer = answer;
    if (rejected != null) result.rejected = rejected;
    if (challenge != null) result.challenge = challenge;
    if (closed != null) result.closed = closed;
    if (presence != null) result.presence = presence;
    if (pathDecisionAck != null) result.pathDecisionAck = pathDecisionAck;
    if (sessionReleaseAck != null) result.sessionReleaseAck = sessionReleaseAck;
    return result;
  }

  EdgeSignal._();

  factory EdgeSignal.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeSignal.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EdgeSignal_Payload>
      _EdgeSignal_PayloadByTag = {
    20: EdgeSignal_Payload.ready,
    21: EdgeSignal_Payload.answer,
    22: EdgeSignal_Payload.rejected,
    23: EdgeSignal_Payload.challenge,
    24: EdgeSignal_Payload.closed,
    25: EdgeSignal_Payload.presence,
    26: EdgeSignal_Payload.pathDecisionAck,
    27: EdgeSignal_Payload.sessionReleaseAck,
    0: EdgeSignal_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeSignal',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [20, 21, 22, 23, 24, 25, 26, 27])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'messageId')
    ..aOS(3, _omitFieldNames ? '' : 'senderId')
    ..aOS(4, _omitFieldNames ? '' : 'bootId')
    ..aOS(5, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$2.Timestamp>(7, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $2.Timestamp.create)
    ..aOM<ClientReady>(20, _omitFieldNames ? '' : 'ready',
        subBuilder: ClientReady.create)
    ..aOM<EdgeAnswer>(21, _omitFieldNames ? '' : 'answer',
        subBuilder: EdgeAnswer.create)
    ..aOM<SignalRejected>(22, _omitFieldNames ? '' : 'rejected',
        subBuilder: SignalRejected.create)
    ..aOM<$0.EdgeChallenge>(23, _omitFieldNames ? '' : 'challenge',
        subBuilder: $0.EdgeChallenge.create)
    ..aOM<SignalSessionClosed>(24, _omitFieldNames ? '' : 'closed',
        subBuilder: SignalSessionClosed.create)
    ..aOM<DaemonPresence>(25, _omitFieldNames ? '' : 'presence',
        subBuilder: DaemonPresence.create)
    ..aOM<EdgePathDecisionAck>(26, _omitFieldNames ? '' : 'pathDecisionAck',
        subBuilder: EdgePathDecisionAck.create)
    ..aOM<EdgeSessionReleaseAck>(27, _omitFieldNames ? '' : 'sessionReleaseAck',
        subBuilder: EdgeSessionReleaseAck.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeSignal clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeSignal copyWith(void Function(EdgeSignal) updates) =>
      super.copyWith((message) => updates(message as EdgeSignal)) as EdgeSignal;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeSignal create() => EdgeSignal._();
  @$core.override
  EdgeSignal createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeSignal getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeSignal>(create);
  static EdgeSignal? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  EdgeSignal_Payload whichPayload() =>
      _EdgeSignal_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
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
  $2.Timestamp get sentAt => $_getN(6);
  @$pb.TagNumber(7)
  set sentAt($2.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSentAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearSentAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $2.Timestamp ensureSentAt() => $_ensure(6);

  @$pb.TagNumber(20)
  ClientReady get ready => $_getN(7);
  @$pb.TagNumber(20)
  set ready(ClientReady value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasReady() => $_has(7);
  @$pb.TagNumber(20)
  void clearReady() => $_clearField(20);
  @$pb.TagNumber(20)
  ClientReady ensureReady() => $_ensure(7);

  @$pb.TagNumber(21)
  EdgeAnswer get answer => $_getN(8);
  @$pb.TagNumber(21)
  set answer(EdgeAnswer value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasAnswer() => $_has(8);
  @$pb.TagNumber(21)
  void clearAnswer() => $_clearField(21);
  @$pb.TagNumber(21)
  EdgeAnswer ensureAnswer() => $_ensure(8);

  @$pb.TagNumber(22)
  SignalRejected get rejected => $_getN(9);
  @$pb.TagNumber(22)
  set rejected(SignalRejected value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasRejected() => $_has(9);
  @$pb.TagNumber(22)
  void clearRejected() => $_clearField(22);
  @$pb.TagNumber(22)
  SignalRejected ensureRejected() => $_ensure(9);

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
  SignalSessionClosed get closed => $_getN(11);
  @$pb.TagNumber(24)
  set closed(SignalSessionClosed value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasClosed() => $_has(11);
  @$pb.TagNumber(24)
  void clearClosed() => $_clearField(24);
  @$pb.TagNumber(24)
  SignalSessionClosed ensureClosed() => $_ensure(11);

  @$pb.TagNumber(25)
  DaemonPresence get presence => $_getN(12);
  @$pb.TagNumber(25)
  set presence(DaemonPresence value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasPresence() => $_has(12);
  @$pb.TagNumber(25)
  void clearPresence() => $_clearField(25);
  @$pb.TagNumber(25)
  DaemonPresence ensurePresence() => $_ensure(12);

  @$pb.TagNumber(26)
  EdgePathDecisionAck get pathDecisionAck => $_getN(13);
  @$pb.TagNumber(26)
  set pathDecisionAck(EdgePathDecisionAck value) => $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasPathDecisionAck() => $_has(13);
  @$pb.TagNumber(26)
  void clearPathDecisionAck() => $_clearField(26);
  @$pb.TagNumber(26)
  EdgePathDecisionAck ensurePathDecisionAck() => $_ensure(13);

  @$pb.TagNumber(27)
  EdgeSessionReleaseAck get sessionReleaseAck => $_getN(14);
  @$pb.TagNumber(27)
  set sessionReleaseAck(EdgeSessionReleaseAck value) => $_setField(27, value);
  @$pb.TagNumber(27)
  $core.bool hasSessionReleaseAck() => $_has(14);
  @$pb.TagNumber(27)
  void clearSessionReleaseAck() => $_clearField(27);
  @$pb.TagNumber(27)
  EdgeSessionReleaseAck ensureSessionReleaseAck() => $_ensure(14);
}

/// ClientGateway 只承载 Cloud WebRTC 信令；DataChannel 建立后业务流量完全绕过该 stream。
class ClientGatewayApi {
  final $pb.RpcClient _client;

  ClientGatewayApi(this._client);

  $async.Future<EdgeSignal> connect(
          $pb.ClientContext? ctx, ClientSignal request) =>
      _client.invoke<EdgeSignal>(
          ctx, 'ClientGateway', 'Connect', request, EdgeSignal());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
