// This is a generated file - do not edit.
//
// Generated from cloud/v1/usage.proto.

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
import 'usage.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'usage.pbenum.dart';

/// RelayPolicySnapshot contains only inputs that actually authorize Relay.
/// Both Controller and Edge hash its deterministic protobuf encoding.
class RelayPolicySnapshot extends $pb.GeneratedMessage {
  factory RelayPolicySnapshot({
    $core.String? accountId,
    $fixnum.Int64? accountRevision,
    $core.String? accountState,
    $core.String? subscriptionId,
    $fixnum.Int64? subscriptionRevision,
    $core.String? subscriptionState,
    $0.Timestamp? periodStart,
    $0.Timestamp? periodEnd,
    $core.String? planId,
    $fixnum.Int64? planVersion,
    $fixnum.Int64? planRevision,
    $core.bool? relayEnabled,
    $fixnum.Int64? relayMaxBytesPerPeriod,
    $fixnum.Int64? relayMaxBytesPerSession,
    $fixnum.Int64? relayMaxRateBytesPerSecond,
    $core.int? relayMaxConcurrency,
    $core.Iterable<$core.String>? allowedRegions,
    $core.String? edgeId,
    $fixnum.Int64? edgeRevision,
    $core.bool? edgeEnabled,
    $core.String? edgeRegion,
    $core.String? daemonId,
    $fixnum.Int64? daemonStateRevision,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (accountRevision != null) result.accountRevision = accountRevision;
    if (accountState != null) result.accountState = accountState;
    if (subscriptionId != null) result.subscriptionId = subscriptionId;
    if (subscriptionRevision != null)
      result.subscriptionRevision = subscriptionRevision;
    if (subscriptionState != null) result.subscriptionState = subscriptionState;
    if (periodStart != null) result.periodStart = periodStart;
    if (periodEnd != null) result.periodEnd = periodEnd;
    if (planId != null) result.planId = planId;
    if (planVersion != null) result.planVersion = planVersion;
    if (planRevision != null) result.planRevision = planRevision;
    if (relayEnabled != null) result.relayEnabled = relayEnabled;
    if (relayMaxBytesPerPeriod != null)
      result.relayMaxBytesPerPeriod = relayMaxBytesPerPeriod;
    if (relayMaxBytesPerSession != null)
      result.relayMaxBytesPerSession = relayMaxBytesPerSession;
    if (relayMaxRateBytesPerSecond != null)
      result.relayMaxRateBytesPerSecond = relayMaxRateBytesPerSecond;
    if (relayMaxConcurrency != null)
      result.relayMaxConcurrency = relayMaxConcurrency;
    if (allowedRegions != null) result.allowedRegions.addAll(allowedRegions);
    if (edgeId != null) result.edgeId = edgeId;
    if (edgeRevision != null) result.edgeRevision = edgeRevision;
    if (edgeEnabled != null) result.edgeEnabled = edgeEnabled;
    if (edgeRegion != null) result.edgeRegion = edgeRegion;
    if (daemonId != null) result.daemonId = daemonId;
    if (daemonStateRevision != null)
      result.daemonStateRevision = daemonStateRevision;
    return result;
  }

  RelayPolicySnapshot._();

  factory RelayPolicySnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayPolicySnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayPolicySnapshot',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'accountRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'accountState')
    ..aOS(4, _omitFieldNames ? '' : 'subscriptionId')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'subscriptionRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'subscriptionState')
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'periodStart',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(8, _omitFieldNames ? '' : 'periodEnd',
        subBuilder: $0.Timestamp.create)
    ..aOS(9, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        10, _omitFieldNames ? '' : 'planVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        11, _omitFieldNames ? '' : 'planRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(12, _omitFieldNames ? '' : 'relayEnabled')
    ..a<$fixnum.Int64>(13, _omitFieldNames ? '' : 'relayMaxBytesPerPeriod',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(14, _omitFieldNames ? '' : 'relayMaxBytesPerSession',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(15, _omitFieldNames ? '' : 'relayMaxRateBytesPerSecond',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(16, _omitFieldNames ? '' : 'relayMaxConcurrency',
        fieldType: $pb.PbFieldType.OU3)
    ..pPS(17, _omitFieldNames ? '' : 'allowedRegions')
    ..aOS(18, _omitFieldNames ? '' : 'edgeId')
    ..a<$fixnum.Int64>(
        19, _omitFieldNames ? '' : 'edgeRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(20, _omitFieldNames ? '' : 'edgeEnabled')
    ..aOS(21, _omitFieldNames ? '' : 'edgeRegion')
    ..aOS(22, _omitFieldNames ? '' : 'daemonId')
    ..a<$fixnum.Int64>(
        23, _omitFieldNames ? '' : 'daemonStateRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayPolicySnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayPolicySnapshot copyWith(void Function(RelayPolicySnapshot) updates) =>
      super.copyWith((message) => updates(message as RelayPolicySnapshot))
          as RelayPolicySnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayPolicySnapshot create() => RelayPolicySnapshot._();
  @$core.override
  RelayPolicySnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayPolicySnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayPolicySnapshot>(create);
  static RelayPolicySnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get accountRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set accountRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get accountState => $_getSZ(2);
  @$pb.TagNumber(3)
  set accountState($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAccountState() => $_has(2);
  @$pb.TagNumber(3)
  void clearAccountState() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get subscriptionId => $_getSZ(3);
  @$pb.TagNumber(4)
  set subscriptionId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSubscriptionId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSubscriptionId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get subscriptionRevision => $_getI64(4);
  @$pb.TagNumber(5)
  set subscriptionRevision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSubscriptionRevision() => $_has(4);
  @$pb.TagNumber(5)
  void clearSubscriptionRevision() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get subscriptionState => $_getSZ(5);
  @$pb.TagNumber(6)
  set subscriptionState($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasSubscriptionState() => $_has(5);
  @$pb.TagNumber(6)
  void clearSubscriptionState() => $_clearField(6);

  @$pb.TagNumber(7)
  $0.Timestamp get periodStart => $_getN(6);
  @$pb.TagNumber(7)
  set periodStart($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasPeriodStart() => $_has(6);
  @$pb.TagNumber(7)
  void clearPeriodStart() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensurePeriodStart() => $_ensure(6);

  @$pb.TagNumber(8)
  $0.Timestamp get periodEnd => $_getN(7);
  @$pb.TagNumber(8)
  set periodEnd($0.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasPeriodEnd() => $_has(7);
  @$pb.TagNumber(8)
  void clearPeriodEnd() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Timestamp ensurePeriodEnd() => $_ensure(7);

  @$pb.TagNumber(9)
  $core.String get planId => $_getSZ(8);
  @$pb.TagNumber(9)
  set planId($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPlanId() => $_has(8);
  @$pb.TagNumber(9)
  void clearPlanId() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get planVersion => $_getI64(9);
  @$pb.TagNumber(10)
  set planVersion($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasPlanVersion() => $_has(9);
  @$pb.TagNumber(10)
  void clearPlanVersion() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get planRevision => $_getI64(10);
  @$pb.TagNumber(11)
  set planRevision($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasPlanRevision() => $_has(10);
  @$pb.TagNumber(11)
  void clearPlanRevision() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get relayEnabled => $_getBF(11);
  @$pb.TagNumber(12)
  set relayEnabled($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasRelayEnabled() => $_has(11);
  @$pb.TagNumber(12)
  void clearRelayEnabled() => $_clearField(12);

  @$pb.TagNumber(13)
  $fixnum.Int64 get relayMaxBytesPerPeriod => $_getI64(12);
  @$pb.TagNumber(13)
  set relayMaxBytesPerPeriod($fixnum.Int64 value) => $_setInt64(12, value);
  @$pb.TagNumber(13)
  $core.bool hasRelayMaxBytesPerPeriod() => $_has(12);
  @$pb.TagNumber(13)
  void clearRelayMaxBytesPerPeriod() => $_clearField(13);

  @$pb.TagNumber(14)
  $fixnum.Int64 get relayMaxBytesPerSession => $_getI64(13);
  @$pb.TagNumber(14)
  set relayMaxBytesPerSession($fixnum.Int64 value) => $_setInt64(13, value);
  @$pb.TagNumber(14)
  $core.bool hasRelayMaxBytesPerSession() => $_has(13);
  @$pb.TagNumber(14)
  void clearRelayMaxBytesPerSession() => $_clearField(14);

  @$pb.TagNumber(15)
  $fixnum.Int64 get relayMaxRateBytesPerSecond => $_getI64(14);
  @$pb.TagNumber(15)
  set relayMaxRateBytesPerSecond($fixnum.Int64 value) => $_setInt64(14, value);
  @$pb.TagNumber(15)
  $core.bool hasRelayMaxRateBytesPerSecond() => $_has(14);
  @$pb.TagNumber(15)
  void clearRelayMaxRateBytesPerSecond() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.int get relayMaxConcurrency => $_getIZ(15);
  @$pb.TagNumber(16)
  set relayMaxConcurrency($core.int value) => $_setUnsignedInt32(15, value);
  @$pb.TagNumber(16)
  $core.bool hasRelayMaxConcurrency() => $_has(15);
  @$pb.TagNumber(16)
  void clearRelayMaxConcurrency() => $_clearField(16);

  @$pb.TagNumber(17)
  $pb.PbList<$core.String> get allowedRegions => $_getList(16);

  @$pb.TagNumber(18)
  $core.String get edgeId => $_getSZ(17);
  @$pb.TagNumber(18)
  set edgeId($core.String value) => $_setString(17, value);
  @$pb.TagNumber(18)
  $core.bool hasEdgeId() => $_has(17);
  @$pb.TagNumber(18)
  void clearEdgeId() => $_clearField(18);

  @$pb.TagNumber(19)
  $fixnum.Int64 get edgeRevision => $_getI64(18);
  @$pb.TagNumber(19)
  set edgeRevision($fixnum.Int64 value) => $_setInt64(18, value);
  @$pb.TagNumber(19)
  $core.bool hasEdgeRevision() => $_has(18);
  @$pb.TagNumber(19)
  void clearEdgeRevision() => $_clearField(19);

  @$pb.TagNumber(20)
  $core.bool get edgeEnabled => $_getBF(19);
  @$pb.TagNumber(20)
  set edgeEnabled($core.bool value) => $_setBool(19, value);
  @$pb.TagNumber(20)
  $core.bool hasEdgeEnabled() => $_has(19);
  @$pb.TagNumber(20)
  void clearEdgeEnabled() => $_clearField(20);

  @$pb.TagNumber(21)
  $core.String get edgeRegion => $_getSZ(20);
  @$pb.TagNumber(21)
  set edgeRegion($core.String value) => $_setString(20, value);
  @$pb.TagNumber(21)
  $core.bool hasEdgeRegion() => $_has(20);
  @$pb.TagNumber(21)
  void clearEdgeRegion() => $_clearField(21);

  @$pb.TagNumber(22)
  $core.String get daemonId => $_getSZ(21);
  @$pb.TagNumber(22)
  set daemonId($core.String value) => $_setString(21, value);
  @$pb.TagNumber(22)
  $core.bool hasDaemonId() => $_has(21);
  @$pb.TagNumber(22)
  void clearDaemonId() => $_clearField(22);

  @$pb.TagNumber(23)
  $fixnum.Int64 get daemonStateRevision => $_getI64(22);
  @$pb.TagNumber(23)
  set daemonStateRevision($fixnum.Int64 value) => $_setInt64(22, value);
  @$pb.TagNumber(23)
  $core.bool hasDaemonStateRevision() => $_has(22);
  @$pb.TagNumber(23)
  void clearDaemonStateRevision() => $_clearField(23);
}

class RelayReserveRequest extends $pb.GeneratedMessage {
  factory RelayReserveRequest({
    $core.String? reservationId,
    $core.String? accountId,
    $core.String? daemonId,
    $core.String? clientId,
    $core.String? sessionId,
    $0.Timestamp? observedAt,
    $core.List<$core.int>? requestDigest,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (accountId != null) result.accountId = accountId;
    if (daemonId != null) result.daemonId = daemonId;
    if (clientId != null) result.clientId = clientId;
    if (sessionId != null) result.sessionId = sessionId;
    if (observedAt != null) result.observedAt = observedAt;
    if (requestDigest != null) result.requestDigest = requestDigest;
    return result;
  }

  RelayReserveRequest._();

  factory RelayReserveRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayReserveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayReserveRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'daemonId')
    ..aOS(4, _omitFieldNames ? '' : 'clientId')
    ..aOS(5, _omitFieldNames ? '' : 'sessionId')
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'observedAt',
        subBuilder: $0.Timestamp.create)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'requestDigest', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayReserveRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayReserveRequest copyWith(void Function(RelayReserveRequest) updates) =>
      super.copyWith((message) => updates(message as RelayReserveRequest))
          as RelayReserveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayReserveRequest create() => RelayReserveRequest._();
  @$core.override
  RelayReserveRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayReserveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayReserveRequest>(create);
  static RelayReserveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get daemonId => $_getSZ(2);
  @$pb.TagNumber(3)
  set daemonId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get clientId => $_getSZ(3);
  @$pb.TagNumber(4)
  set clientId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasClientId() => $_has(3);
  @$pb.TagNumber(4)
  void clearClientId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get sessionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set sessionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSessionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearSessionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get observedAt => $_getN(5);
  @$pb.TagNumber(6)
  set observedAt($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasObservedAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearObservedAt() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureObservedAt() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.List<$core.int> get requestDigest => $_getN(6);
  @$pb.TagNumber(7)
  set requestDigest($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRequestDigest() => $_has(6);
  @$pb.TagNumber(7)
  void clearRequestDigest() => $_clearField(7);
}

class RelayGrant extends $pb.GeneratedMessage {
  factory RelayGrant({
    $core.String? reservationId,
    $core.String? sessionId,
    $fixnum.Int64? reservedBytes,
    $fixnum.Int64? maxRateBytesPerSecond,
    $fixnum.Int64? renewSequence,
    $0.Timestamp? authorizedUntil,
    $core.List<$core.int>? policyDigest,
    RelayPolicySnapshot? policy,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (sessionId != null) result.sessionId = sessionId;
    if (reservedBytes != null) result.reservedBytes = reservedBytes;
    if (maxRateBytesPerSecond != null)
      result.maxRateBytesPerSecond = maxRateBytesPerSecond;
    if (renewSequence != null) result.renewSequence = renewSequence;
    if (authorizedUntil != null) result.authorizedUntil = authorizedUntil;
    if (policyDigest != null) result.policyDigest = policyDigest;
    if (policy != null) result.policy = policy;
    return result;
  }

  RelayGrant._();

  factory RelayGrant.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayGrant.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayGrant',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'reservedBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'maxRateBytesPerSecond', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'renewSequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'authorizedUntil',
        subBuilder: $0.Timestamp.create)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'policyDigest', $pb.PbFieldType.OY)
    ..aOM<RelayPolicySnapshot>(8, _omitFieldNames ? '' : 'policy',
        subBuilder: RelayPolicySnapshot.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayGrant clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayGrant copyWith(void Function(RelayGrant) updates) =>
      super.copyWith((message) => updates(message as RelayGrant)) as RelayGrant;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayGrant create() => RelayGrant._();
  @$core.override
  RelayGrant createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayGrant getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayGrant>(create);
  static RelayGrant? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get reservedBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set reservedBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReservedBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearReservedBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get maxRateBytesPerSecond => $_getI64(3);
  @$pb.TagNumber(4)
  set maxRateBytesPerSecond($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMaxRateBytesPerSecond() => $_has(3);
  @$pb.TagNumber(4)
  void clearMaxRateBytesPerSecond() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get renewSequence => $_getI64(4);
  @$pb.TagNumber(5)
  set renewSequence($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRenewSequence() => $_has(4);
  @$pb.TagNumber(5)
  void clearRenewSequence() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get authorizedUntil => $_getN(5);
  @$pb.TagNumber(6)
  set authorizedUntil($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasAuthorizedUntil() => $_has(5);
  @$pb.TagNumber(6)
  void clearAuthorizedUntil() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureAuthorizedUntil() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.List<$core.int> get policyDigest => $_getN(6);
  @$pb.TagNumber(7)
  set policyDigest($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasPolicyDigest() => $_has(6);
  @$pb.TagNumber(7)
  void clearPolicyDigest() => $_clearField(7);

  @$pb.TagNumber(8)
  RelayPolicySnapshot get policy => $_getN(7);
  @$pb.TagNumber(8)
  set policy(RelayPolicySnapshot value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasPolicy() => $_has(7);
  @$pb.TagNumber(8)
  void clearPolicy() => $_clearField(8);
  @$pb.TagNumber(8)
  RelayPolicySnapshot ensurePolicy() => $_ensure(7);
}

class RelayReserveResponse extends $pb.GeneratedMessage {
  factory RelayReserveResponse({
    $core.String? reservationId,
    $core.List<$core.int>? requestDigest,
    RelayResponseCode? code,
    RelayGrant? grant,
    RelaySettlementAck? terminal,
    $core.String? errorMessage,
    $1.CloudEntitlementFailure? entitlementFailure,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (requestDigest != null) result.requestDigest = requestDigest;
    if (code != null) result.code = code;
    if (grant != null) result.grant = grant;
    if (terminal != null) result.terminal = terminal;
    if (errorMessage != null) result.errorMessage = errorMessage;
    if (entitlementFailure != null)
      result.entitlementFailure = entitlementFailure;
    return result;
  }

  RelayReserveResponse._();

  factory RelayReserveResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayReserveResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayReserveResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'requestDigest', $pb.PbFieldType.OY)
    ..aE<RelayResponseCode>(3, _omitFieldNames ? '' : 'code',
        enumValues: RelayResponseCode.values)
    ..aOM<RelayGrant>(4, _omitFieldNames ? '' : 'grant',
        subBuilder: RelayGrant.create)
    ..aOM<RelaySettlementAck>(5, _omitFieldNames ? '' : 'terminal',
        subBuilder: RelaySettlementAck.create)
    ..aOS(6, _omitFieldNames ? '' : 'errorMessage')
    ..aOM<$1.CloudEntitlementFailure>(
        7, _omitFieldNames ? '' : 'entitlementFailure',
        subBuilder: $1.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayReserveResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayReserveResponse copyWith(void Function(RelayReserveResponse) updates) =>
      super.copyWith((message) => updates(message as RelayReserveResponse))
          as RelayReserveResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayReserveResponse create() => RelayReserveResponse._();
  @$core.override
  RelayReserveResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayReserveResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayReserveResponse>(create);
  static RelayReserveResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get requestDigest => $_getN(1);
  @$pb.TagNumber(2)
  set requestDigest($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRequestDigest() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestDigest() => $_clearField(2);

  @$pb.TagNumber(3)
  RelayResponseCode get code => $_getN(2);
  @$pb.TagNumber(3)
  set code(RelayResponseCode value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCode() => $_has(2);
  @$pb.TagNumber(3)
  void clearCode() => $_clearField(3);

  @$pb.TagNumber(4)
  RelayGrant get grant => $_getN(3);
  @$pb.TagNumber(4)
  set grant(RelayGrant value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasGrant() => $_has(3);
  @$pb.TagNumber(4)
  void clearGrant() => $_clearField(4);
  @$pb.TagNumber(4)
  RelayGrant ensureGrant() => $_ensure(3);

  @$pb.TagNumber(5)
  RelaySettlementAck get terminal => $_getN(4);
  @$pb.TagNumber(5)
  set terminal(RelaySettlementAck value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasTerminal() => $_has(4);
  @$pb.TagNumber(5)
  void clearTerminal() => $_clearField(5);
  @$pb.TagNumber(5)
  RelaySettlementAck ensureTerminal() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get errorMessage => $_getSZ(5);
  @$pb.TagNumber(6)
  set errorMessage($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasErrorMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearErrorMessage() => $_clearField(6);

  @$pb.TagNumber(7)
  $1.CloudEntitlementFailure get entitlementFailure => $_getN(6);
  @$pb.TagNumber(7)
  set entitlementFailure($1.CloudEntitlementFailure value) =>
      $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasEntitlementFailure() => $_has(6);
  @$pb.TagNumber(7)
  void clearEntitlementFailure() => $_clearField(7);
  @$pb.TagNumber(7)
  $1.CloudEntitlementFailure ensureEntitlementFailure() => $_ensure(6);
}

class RelayRenewRequest extends $pb.GeneratedMessage {
  factory RelayRenewRequest({
    $core.String? reservationId,
    $fixnum.Int64? renewSequence,
    $core.List<$core.int>? policyDigest,
    $0.Timestamp? observedAt,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (renewSequence != null) result.renewSequence = renewSequence;
    if (policyDigest != null) result.policyDigest = policyDigest;
    if (observedAt != null) result.observedAt = observedAt;
    return result;
  }

  RelayRenewRequest._();

  factory RelayRenewRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayRenewRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayRenewRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'renewSequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'policyDigest', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(4, _omitFieldNames ? '' : 'observedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayRenewRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayRenewRequest copyWith(void Function(RelayRenewRequest) updates) =>
      super.copyWith((message) => updates(message as RelayRenewRequest))
          as RelayRenewRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayRenewRequest create() => RelayRenewRequest._();
  @$core.override
  RelayRenewRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayRenewRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayRenewRequest>(create);
  static RelayRenewRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get renewSequence => $_getI64(1);
  @$pb.TagNumber(2)
  set renewSequence($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRenewSequence() => $_has(1);
  @$pb.TagNumber(2)
  void clearRenewSequence() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get policyDigest => $_getN(2);
  @$pb.TagNumber(3)
  set policyDigest($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPolicyDigest() => $_has(2);
  @$pb.TagNumber(3)
  void clearPolicyDigest() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.Timestamp get observedAt => $_getN(3);
  @$pb.TagNumber(4)
  set observedAt($0.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasObservedAt() => $_has(3);
  @$pb.TagNumber(4)
  void clearObservedAt() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Timestamp ensureObservedAt() => $_ensure(3);
}

class RelayRenewResponse extends $pb.GeneratedMessage {
  factory RelayRenewResponse({
    $core.String? reservationId,
    $fixnum.Int64? renewSequence,
    RelayResponseCode? code,
    RelayGrant? grant,
    RelaySettlementAck? terminal,
    $core.String? errorMessage,
    $1.CloudEntitlementFailure? entitlementFailure,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (renewSequence != null) result.renewSequence = renewSequence;
    if (code != null) result.code = code;
    if (grant != null) result.grant = grant;
    if (terminal != null) result.terminal = terminal;
    if (errorMessage != null) result.errorMessage = errorMessage;
    if (entitlementFailure != null)
      result.entitlementFailure = entitlementFailure;
    return result;
  }

  RelayRenewResponse._();

  factory RelayRenewResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayRenewResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayRenewResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'renewSequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<RelayResponseCode>(3, _omitFieldNames ? '' : 'code',
        enumValues: RelayResponseCode.values)
    ..aOM<RelayGrant>(4, _omitFieldNames ? '' : 'grant',
        subBuilder: RelayGrant.create)
    ..aOM<RelaySettlementAck>(5, _omitFieldNames ? '' : 'terminal',
        subBuilder: RelaySettlementAck.create)
    ..aOS(6, _omitFieldNames ? '' : 'errorMessage')
    ..aOM<$1.CloudEntitlementFailure>(
        7, _omitFieldNames ? '' : 'entitlementFailure',
        subBuilder: $1.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayRenewResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayRenewResponse copyWith(void Function(RelayRenewResponse) updates) =>
      super.copyWith((message) => updates(message as RelayRenewResponse))
          as RelayRenewResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayRenewResponse create() => RelayRenewResponse._();
  @$core.override
  RelayRenewResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayRenewResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayRenewResponse>(create);
  static RelayRenewResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get renewSequence => $_getI64(1);
  @$pb.TagNumber(2)
  set renewSequence($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRenewSequence() => $_has(1);
  @$pb.TagNumber(2)
  void clearRenewSequence() => $_clearField(2);

  @$pb.TagNumber(3)
  RelayResponseCode get code => $_getN(2);
  @$pb.TagNumber(3)
  set code(RelayResponseCode value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCode() => $_has(2);
  @$pb.TagNumber(3)
  void clearCode() => $_clearField(3);

  @$pb.TagNumber(4)
  RelayGrant get grant => $_getN(3);
  @$pb.TagNumber(4)
  set grant(RelayGrant value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasGrant() => $_has(3);
  @$pb.TagNumber(4)
  void clearGrant() => $_clearField(4);
  @$pb.TagNumber(4)
  RelayGrant ensureGrant() => $_ensure(3);

  @$pb.TagNumber(5)
  RelaySettlementAck get terminal => $_getN(4);
  @$pb.TagNumber(5)
  set terminal(RelaySettlementAck value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasTerminal() => $_has(4);
  @$pb.TagNumber(5)
  void clearTerminal() => $_clearField(5);
  @$pb.TagNumber(5)
  RelaySettlementAck ensureTerminal() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get errorMessage => $_getSZ(5);
  @$pb.TagNumber(6)
  set errorMessage($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasErrorMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearErrorMessage() => $_clearField(6);

  @$pb.TagNumber(7)
  $1.CloudEntitlementFailure get entitlementFailure => $_getN(6);
  @$pb.TagNumber(7)
  set entitlementFailure($1.CloudEntitlementFailure value) =>
      $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasEntitlementFailure() => $_has(6);
  @$pb.TagNumber(7)
  void clearEntitlementFailure() => $_clearField(7);
  @$pb.TagNumber(7)
  $1.CloudEntitlementFailure ensureEntitlementFailure() => $_ensure(6);
}

class RelaySettlement extends $pb.GeneratedMessage {
  factory RelaySettlement({
    $core.String? reservationId,
    RelaySettlementKind? kind,
    $fixnum.Int64? ingressBytes,
    $fixnum.Int64? egressBytes,
    $core.List<$core.int>? policyDigest,
    $0.Timestamp? observedAt,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (kind != null) result.kind = kind;
    if (ingressBytes != null) result.ingressBytes = ingressBytes;
    if (egressBytes != null) result.egressBytes = egressBytes;
    if (policyDigest != null) result.policyDigest = policyDigest;
    if (observedAt != null) result.observedAt = observedAt;
    return result;
  }

  RelaySettlement._();

  factory RelaySettlement.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelaySettlement.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelaySettlement',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..aE<RelaySettlementKind>(2, _omitFieldNames ? '' : 'kind',
        enumValues: RelaySettlementKind.values)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'ingressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'egressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'policyDigest', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'observedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelaySettlement clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelaySettlement copyWith(void Function(RelaySettlement) updates) =>
      super.copyWith((message) => updates(message as RelaySettlement))
          as RelaySettlement;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelaySettlement create() => RelaySettlement._();
  @$core.override
  RelaySettlement createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelaySettlement getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelaySettlement>(create);
  static RelaySettlement? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  RelaySettlementKind get kind => $_getN(1);
  @$pb.TagNumber(2)
  set kind(RelaySettlementKind value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get ingressBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set ingressBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIngressBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearIngressBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get egressBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set egressBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEgressBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearEgressBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get policyDigest => $_getN(4);
  @$pb.TagNumber(5)
  set policyDigest($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPolicyDigest() => $_has(4);
  @$pb.TagNumber(5)
  void clearPolicyDigest() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get observedAt => $_getN(5);
  @$pb.TagNumber(6)
  set observedAt($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasObservedAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearObservedAt() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureObservedAt() => $_ensure(5);
}

class RelaySettlementAck extends $pb.GeneratedMessage {
  factory RelaySettlementAck({
    $core.String? reservationId,
    RelaySettlementKind? kind,
    $fixnum.Int64? ingressBytes,
    $fixnum.Int64? egressBytes,
    $fixnum.Int64? recoveryBytes,
    $core.List<$core.int>? policyDigest,
    $0.Timestamp? observedAt,
    $0.Timestamp? settledAt,
    RelayResponseCode? code,
    $core.String? errorMessage,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (kind != null) result.kind = kind;
    if (ingressBytes != null) result.ingressBytes = ingressBytes;
    if (egressBytes != null) result.egressBytes = egressBytes;
    if (recoveryBytes != null) result.recoveryBytes = recoveryBytes;
    if (policyDigest != null) result.policyDigest = policyDigest;
    if (observedAt != null) result.observedAt = observedAt;
    if (settledAt != null) result.settledAt = settledAt;
    if (code != null) result.code = code;
    if (errorMessage != null) result.errorMessage = errorMessage;
    return result;
  }

  RelaySettlementAck._();

  factory RelaySettlementAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelaySettlementAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelaySettlementAck',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..aE<RelaySettlementKind>(2, _omitFieldNames ? '' : 'kind',
        enumValues: RelaySettlementKind.values)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'ingressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'egressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'recoveryBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'policyDigest', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'observedAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(8, _omitFieldNames ? '' : 'settledAt',
        subBuilder: $0.Timestamp.create)
    ..aE<RelayResponseCode>(9, _omitFieldNames ? '' : 'code',
        enumValues: RelayResponseCode.values)
    ..aOS(10, _omitFieldNames ? '' : 'errorMessage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelaySettlementAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelaySettlementAck copyWith(void Function(RelaySettlementAck) updates) =>
      super.copyWith((message) => updates(message as RelaySettlementAck))
          as RelaySettlementAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelaySettlementAck create() => RelaySettlementAck._();
  @$core.override
  RelaySettlementAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelaySettlementAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelaySettlementAck>(create);
  static RelaySettlementAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  RelaySettlementKind get kind => $_getN(1);
  @$pb.TagNumber(2)
  set kind(RelaySettlementKind value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get ingressBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set ingressBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIngressBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearIngressBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get egressBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set egressBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEgressBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearEgressBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get recoveryBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set recoveryBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRecoveryBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearRecoveryBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get policyDigest => $_getN(5);
  @$pb.TagNumber(6)
  set policyDigest($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPolicyDigest() => $_has(5);
  @$pb.TagNumber(6)
  void clearPolicyDigest() => $_clearField(6);

  @$pb.TagNumber(7)
  $0.Timestamp get observedAt => $_getN(6);
  @$pb.TagNumber(7)
  set observedAt($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasObservedAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearObservedAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensureObservedAt() => $_ensure(6);

  @$pb.TagNumber(8)
  $0.Timestamp get settledAt => $_getN(7);
  @$pb.TagNumber(8)
  set settledAt($0.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasSettledAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearSettledAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Timestamp ensureSettledAt() => $_ensure(7);

  @$pb.TagNumber(9)
  RelayResponseCode get code => $_getN(8);
  @$pb.TagNumber(9)
  set code(RelayResponseCode value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasCode() => $_has(8);
  @$pb.TagNumber(9)
  void clearCode() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get errorMessage => $_getSZ(9);
  @$pb.TagNumber(10)
  set errorMessage($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasErrorMessage() => $_has(9);
  @$pb.TagNumber(10)
  void clearErrorMessage() => $_clearField(10);
}

class RelayQueryRequest extends $pb.GeneratedMessage {
  factory RelayQueryRequest({
    $core.String? reservationId,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    return result;
  }

  RelayQueryRequest._();

  factory RelayQueryRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayQueryRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayQueryRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayQueryRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayQueryRequest copyWith(void Function(RelayQueryRequest) updates) =>
      super.copyWith((message) => updates(message as RelayQueryRequest))
          as RelayQueryRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayQueryRequest create() => RelayQueryRequest._();
  @$core.override
  RelayQueryRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayQueryRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayQueryRequest>(create);
  static RelayQueryRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);
}

class RelayQueryResponse extends $pb.GeneratedMessage {
  factory RelayQueryResponse({
    $core.String? reservationId,
    RelayResponseCode? code,
    RelayGrant? grant,
    RelaySettlementAck? terminal,
    $core.String? errorMessage,
    $1.CloudEntitlementFailure? entitlementFailure,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (code != null) result.code = code;
    if (grant != null) result.grant = grant;
    if (terminal != null) result.terminal = terminal;
    if (errorMessage != null) result.errorMessage = errorMessage;
    if (entitlementFailure != null)
      result.entitlementFailure = entitlementFailure;
    return result;
  }

  RelayQueryResponse._();

  factory RelayQueryResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayQueryResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayQueryResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..aE<RelayResponseCode>(2, _omitFieldNames ? '' : 'code',
        enumValues: RelayResponseCode.values)
    ..aOM<RelayGrant>(3, _omitFieldNames ? '' : 'grant',
        subBuilder: RelayGrant.create)
    ..aOM<RelaySettlementAck>(4, _omitFieldNames ? '' : 'terminal',
        subBuilder: RelaySettlementAck.create)
    ..aOS(5, _omitFieldNames ? '' : 'errorMessage')
    ..aOM<$1.CloudEntitlementFailure>(
        6, _omitFieldNames ? '' : 'entitlementFailure',
        subBuilder: $1.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayQueryResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayQueryResponse copyWith(void Function(RelayQueryResponse) updates) =>
      super.copyWith((message) => updates(message as RelayQueryResponse))
          as RelayQueryResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayQueryResponse create() => RelayQueryResponse._();
  @$core.override
  RelayQueryResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayQueryResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayQueryResponse>(create);
  static RelayQueryResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  RelayResponseCode get code => $_getN(1);
  @$pb.TagNumber(2)
  set code(RelayResponseCode value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCode() => $_has(1);
  @$pb.TagNumber(2)
  void clearCode() => $_clearField(2);

  @$pb.TagNumber(3)
  RelayGrant get grant => $_getN(2);
  @$pb.TagNumber(3)
  set grant(RelayGrant value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasGrant() => $_has(2);
  @$pb.TagNumber(3)
  void clearGrant() => $_clearField(3);
  @$pb.TagNumber(3)
  RelayGrant ensureGrant() => $_ensure(2);

  @$pb.TagNumber(4)
  RelaySettlementAck get terminal => $_getN(3);
  @$pb.TagNumber(4)
  set terminal(RelaySettlementAck value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasTerminal() => $_has(3);
  @$pb.TagNumber(4)
  void clearTerminal() => $_clearField(4);
  @$pb.TagNumber(4)
  RelaySettlementAck ensureTerminal() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.String get errorMessage => $_getSZ(4);
  @$pb.TagNumber(5)
  set errorMessage($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasErrorMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearErrorMessage() => $_clearField(5);

  @$pb.TagNumber(6)
  $1.CloudEntitlementFailure get entitlementFailure => $_getN(5);
  @$pb.TagNumber(6)
  set entitlementFailure($1.CloudEntitlementFailure value) =>
      $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasEntitlementFailure() => $_has(5);
  @$pb.TagNumber(6)
  void clearEntitlementFailure() => $_clearField(6);
  @$pb.TagNumber(6)
  $1.CloudEntitlementFailure ensureEntitlementFailure() => $_ensure(5);
}

/// RelayRuntimePolicy is a versioned commercial policy snapshot used by an
/// Edge for local Relay admission. It does not reserve bytes or require renewal.
class RelayRuntimePolicy extends $pb.GeneratedMessage {
  factory RelayRuntimePolicy({
    $core.String? accountId,
    $core.String? subscriptionId,
    $core.String? planId,
    $fixnum.Int64? policyRevision,
    $core.bool? relayEnabled,
    $fixnum.Int64? relayMaxRateBytesPerSecond,
    $core.int? relayMaxConcurrency,
    $fixnum.Int64? relayQuotaBytes,
    $0.Timestamp? periodStart,
    $0.Timestamp? periodEnd,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (subscriptionId != null) result.subscriptionId = subscriptionId;
    if (planId != null) result.planId = planId;
    if (policyRevision != null) result.policyRevision = policyRevision;
    if (relayEnabled != null) result.relayEnabled = relayEnabled;
    if (relayMaxRateBytesPerSecond != null)
      result.relayMaxRateBytesPerSecond = relayMaxRateBytesPerSecond;
    if (relayMaxConcurrency != null)
      result.relayMaxConcurrency = relayMaxConcurrency;
    if (relayQuotaBytes != null) result.relayQuotaBytes = relayQuotaBytes;
    if (periodStart != null) result.periodStart = periodStart;
    if (periodEnd != null) result.periodEnd = periodEnd;
    return result;
  }

  RelayRuntimePolicy._();

  factory RelayRuntimePolicy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayRuntimePolicy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayRuntimePolicy',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOS(2, _omitFieldNames ? '' : 'subscriptionId')
    ..aOS(3, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'policyRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(5, _omitFieldNames ? '' : 'relayEnabled')
    ..a<$fixnum.Int64>(6, _omitFieldNames ? '' : 'relayMaxRateBytesPerSecond',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(7, _omitFieldNames ? '' : 'relayMaxConcurrency',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'relayQuotaBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'periodStart',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'periodEnd',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayRuntimePolicy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayRuntimePolicy copyWith(void Function(RelayRuntimePolicy) updates) =>
      super.copyWith((message) => updates(message as RelayRuntimePolicy))
          as RelayRuntimePolicy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayRuntimePolicy create() => RelayRuntimePolicy._();
  @$core.override
  RelayRuntimePolicy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayRuntimePolicy getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayRuntimePolicy>(create);
  static RelayRuntimePolicy? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get subscriptionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set subscriptionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSubscriptionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSubscriptionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get planId => $_getSZ(2);
  @$pb.TagNumber(3)
  set planId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlanId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlanId() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get policyRevision => $_getI64(3);
  @$pb.TagNumber(4)
  set policyRevision($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPolicyRevision() => $_has(3);
  @$pb.TagNumber(4)
  void clearPolicyRevision() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get relayEnabled => $_getBF(4);
  @$pb.TagNumber(5)
  set relayEnabled($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRelayEnabled() => $_has(4);
  @$pb.TagNumber(5)
  void clearRelayEnabled() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get relayMaxRateBytesPerSecond => $_getI64(5);
  @$pb.TagNumber(6)
  set relayMaxRateBytesPerSecond($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRelayMaxRateBytesPerSecond() => $_has(5);
  @$pb.TagNumber(6)
  void clearRelayMaxRateBytesPerSecond() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get relayMaxConcurrency => $_getIZ(6);
  @$pb.TagNumber(7)
  set relayMaxConcurrency($core.int value) => $_setUnsignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRelayMaxConcurrency() => $_has(6);
  @$pb.TagNumber(7)
  void clearRelayMaxConcurrency() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get relayQuotaBytes => $_getI64(7);
  @$pb.TagNumber(8)
  set relayQuotaBytes($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRelayQuotaBytes() => $_has(7);
  @$pb.TagNumber(8)
  void clearRelayQuotaBytes() => $_clearField(8);

  @$pb.TagNumber(9)
  $0.Timestamp get periodStart => $_getN(8);
  @$pb.TagNumber(9)
  set periodStart($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasPeriodStart() => $_has(8);
  @$pb.TagNumber(9)
  void clearPeriodStart() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensurePeriodStart() => $_ensure(8);

  @$pb.TagNumber(10)
  $0.Timestamp get periodEnd => $_getN(9);
  @$pb.TagNumber(10)
  set periodEnd($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasPeriodEnd() => $_has(9);
  @$pb.TagNumber(10)
  void clearPeriodEnd() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensurePeriodEnd() => $_ensure(9);
}

/// RelayAuthorizeRequest is the optional fast-path cache fill used when an Edge
/// has no local account policy. It never creates a durable reservation.
class RelayAuthorizeRequest extends $pb.GeneratedMessage {
  factory RelayAuthorizeRequest({
    $core.String? requestId,
    $core.String? accountId,
    $core.String? daemonId,
    $core.String? sessionId,
    $0.Timestamp? observedAt,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (accountId != null) result.accountId = accountId;
    if (daemonId != null) result.daemonId = daemonId;
    if (sessionId != null) result.sessionId = sessionId;
    if (observedAt != null) result.observedAt = observedAt;
    return result;
  }

  RelayAuthorizeRequest._();

  factory RelayAuthorizeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayAuthorizeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayAuthorizeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'daemonId')
    ..aOS(4, _omitFieldNames ? '' : 'sessionId')
    ..aOM<$0.Timestamp>(5, _omitFieldNames ? '' : 'observedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayAuthorizeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayAuthorizeRequest copyWith(
          void Function(RelayAuthorizeRequest) updates) =>
      super.copyWith((message) => updates(message as RelayAuthorizeRequest))
          as RelayAuthorizeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayAuthorizeRequest create() => RelayAuthorizeRequest._();
  @$core.override
  RelayAuthorizeRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayAuthorizeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayAuthorizeRequest>(create);
  static RelayAuthorizeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get daemonId => $_getSZ(2);
  @$pb.TagNumber(3)
  set daemonId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get sessionId => $_getSZ(3);
  @$pb.TagNumber(4)
  set sessionId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSessionId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSessionId() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.Timestamp get observedAt => $_getN(4);
  @$pb.TagNumber(5)
  set observedAt($0.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasObservedAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearObservedAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Timestamp ensureObservedAt() => $_ensure(4);
}

class RelayAuthorizeResponse extends $pb.GeneratedMessage {
  factory RelayAuthorizeResponse({
    $core.String? requestId,
    RelayRuntimePolicy? policy,
    $1.CloudEntitlementFailure? entitlementFailure,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (policy != null) result.policy = policy;
    if (entitlementFailure != null)
      result.entitlementFailure = entitlementFailure;
    return result;
  }

  RelayAuthorizeResponse._();

  factory RelayAuthorizeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayAuthorizeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayAuthorizeResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<RelayRuntimePolicy>(2, _omitFieldNames ? '' : 'policy',
        subBuilder: RelayRuntimePolicy.create)
    ..aOM<$1.CloudEntitlementFailure>(
        3, _omitFieldNames ? '' : 'entitlementFailure',
        subBuilder: $1.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayAuthorizeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayAuthorizeResponse copyWith(
          void Function(RelayAuthorizeResponse) updates) =>
      super.copyWith((message) => updates(message as RelayAuthorizeResponse))
          as RelayAuthorizeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayAuthorizeResponse create() => RelayAuthorizeResponse._();
  @$core.override
  RelayAuthorizeResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayAuthorizeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayAuthorizeResponse>(create);
  static RelayAuthorizeResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  RelayRuntimePolicy get policy => $_getN(1);
  @$pb.TagNumber(2)
  set policy(RelayRuntimePolicy value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPolicy() => $_has(1);
  @$pb.TagNumber(2)
  void clearPolicy() => $_clearField(2);
  @$pb.TagNumber(2)
  RelayRuntimePolicy ensurePolicy() => $_ensure(1);

  @$pb.TagNumber(3)
  $1.CloudEntitlementFailure get entitlementFailure => $_getN(2);
  @$pb.TagNumber(3)
  set entitlementFailure($1.CloudEntitlementFailure value) =>
      $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEntitlementFailure() => $_has(2);
  @$pb.TagNumber(3)
  void clearEntitlementFailure() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.CloudEntitlementFailure ensureEntitlementFailure() => $_ensure(2);
}

/// RelayUsageSample is cumulative for one (Edge boot, account) counter epoch.
/// Retrying the same or an older value is therefore idempotent.
class RelayUsageSample extends $pb.GeneratedMessage {
  factory RelayUsageSample({
    $core.String? accountId,
    $fixnum.Int64? cumulativeEgressBytes,
    $0.Timestamp? sampledAt,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (cumulativeEgressBytes != null)
      result.cumulativeEgressBytes = cumulativeEgressBytes;
    if (sampledAt != null) result.sampledAt = sampledAt;
    return result;
  }

  RelayUsageSample._();

  factory RelayUsageSample.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayUsageSample.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayUsageSample',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'cumulativeEgressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'sampledAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayUsageSample clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayUsageSample copyWith(void Function(RelayUsageSample) updates) =>
      super.copyWith((message) => updates(message as RelayUsageSample))
          as RelayUsageSample;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayUsageSample create() => RelayUsageSample._();
  @$core.override
  RelayUsageSample createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayUsageSample getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayUsageSample>(create);
  static RelayUsageSample? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get cumulativeEgressBytes => $_getI64(1);
  @$pb.TagNumber(2)
  set cumulativeEgressBytes($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCumulativeEgressBytes() => $_has(1);
  @$pb.TagNumber(2)
  void clearCumulativeEgressBytes() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get sampledAt => $_getN(2);
  @$pb.TagNumber(3)
  set sampledAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSampledAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearSampledAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureSampledAt() => $_ensure(2);
}

class RelayUsageBatch extends $pb.GeneratedMessage {
  factory RelayUsageBatch({
    $fixnum.Int64? batchSequence,
    $core.Iterable<RelayUsageSample>? samples,
  }) {
    final result = create();
    if (batchSequence != null) result.batchSequence = batchSequence;
    if (samples != null) result.samples.addAll(samples);
    return result;
  }

  RelayUsageBatch._();

  factory RelayUsageBatch.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayUsageBatch.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayUsageBatch',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'batchSequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPM<RelayUsageSample>(2, _omitFieldNames ? '' : 'samples',
        subBuilder: RelayUsageSample.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayUsageBatch clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayUsageBatch copyWith(void Function(RelayUsageBatch) updates) =>
      super.copyWith((message) => updates(message as RelayUsageBatch))
          as RelayUsageBatch;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayUsageBatch create() => RelayUsageBatch._();
  @$core.override
  RelayUsageBatch createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayUsageBatch getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayUsageBatch>(create);
  static RelayUsageBatch? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get batchSequence => $_getI64(0);
  @$pb.TagNumber(1)
  set batchSequence($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBatchSequence() => $_has(0);
  @$pb.TagNumber(1)
  void clearBatchSequence() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<RelayUsageSample> get samples => $_getList(1);
}

class RelayAccountAction extends $pb.GeneratedMessage {
  factory RelayAccountAction({
    $core.String? accountId,
    RelayAccountActionType? action,
    $fixnum.Int64? actionRevision,
    $fixnum.Int64? usedBytes,
    $fixnum.Int64? quotaBytes,
    $0.Timestamp? periodStart,
    $0.Timestamp? periodEnd,
    $core.String? reason,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (action != null) result.action = action;
    if (actionRevision != null) result.actionRevision = actionRevision;
    if (usedBytes != null) result.usedBytes = usedBytes;
    if (quotaBytes != null) result.quotaBytes = quotaBytes;
    if (periodStart != null) result.periodStart = periodStart;
    if (periodEnd != null) result.periodEnd = periodEnd;
    if (reason != null) result.reason = reason;
    return result;
  }

  RelayAccountAction._();

  factory RelayAccountAction.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayAccountAction.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayAccountAction',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aE<RelayAccountActionType>(2, _omitFieldNames ? '' : 'action',
        enumValues: RelayAccountActionType.values)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'actionRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'usedBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'quotaBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'periodStart',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'periodEnd',
        subBuilder: $0.Timestamp.create)
    ..aOS(8, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayAccountAction clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayAccountAction copyWith(void Function(RelayAccountAction) updates) =>
      super.copyWith((message) => updates(message as RelayAccountAction))
          as RelayAccountAction;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayAccountAction create() => RelayAccountAction._();
  @$core.override
  RelayAccountAction createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayAccountAction getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayAccountAction>(create);
  static RelayAccountAction? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  RelayAccountActionType get action => $_getN(1);
  @$pb.TagNumber(2)
  set action(RelayAccountActionType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasAction() => $_has(1);
  @$pb.TagNumber(2)
  void clearAction() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get actionRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set actionRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasActionRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearActionRevision() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get usedBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set usedBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUsedBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearUsedBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get quotaBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set quotaBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasQuotaBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearQuotaBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get periodStart => $_getN(5);
  @$pb.TagNumber(6)
  set periodStart($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasPeriodStart() => $_has(5);
  @$pb.TagNumber(6)
  void clearPeriodStart() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensurePeriodStart() => $_ensure(5);

  @$pb.TagNumber(7)
  $0.Timestamp get periodEnd => $_getN(6);
  @$pb.TagNumber(7)
  set periodEnd($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasPeriodEnd() => $_has(6);
  @$pb.TagNumber(7)
  void clearPeriodEnd() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensurePeriodEnd() => $_ensure(6);

  @$pb.TagNumber(8)
  $core.String get reason => $_getSZ(7);
  @$pb.TagNumber(8)
  set reason($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasReason() => $_has(7);
  @$pb.TagNumber(8)
  void clearReason() => $_clearField(8);
}

class RelayUsageAck extends $pb.GeneratedMessage {
  factory RelayUsageAck({
    $fixnum.Int64? batchSequence,
    $core.Iterable<RelayAccountAction>? actions,
    $0.Timestamp? processedAt,
  }) {
    final result = create();
    if (batchSequence != null) result.batchSequence = batchSequence;
    if (actions != null) result.actions.addAll(actions);
    if (processedAt != null) result.processedAt = processedAt;
    return result;
  }

  RelayUsageAck._();

  factory RelayUsageAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayUsageAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayUsageAck',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'batchSequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPM<RelayAccountAction>(2, _omitFieldNames ? '' : 'actions',
        subBuilder: RelayAccountAction.create)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'processedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayUsageAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayUsageAck copyWith(void Function(RelayUsageAck) updates) =>
      super.copyWith((message) => updates(message as RelayUsageAck))
          as RelayUsageAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayUsageAck create() => RelayUsageAck._();
  @$core.override
  RelayUsageAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayUsageAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayUsageAck>(create);
  static RelayUsageAck? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get batchSequence => $_getI64(0);
  @$pb.TagNumber(1)
  set batchSequence($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBatchSequence() => $_has(0);
  @$pb.TagNumber(1)
  void clearBatchSequence() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<RelayAccountAction> get actions => $_getList(1);

  @$pb.TagNumber(3)
  $0.Timestamp get processedAt => $_getN(2);
  @$pb.TagNumber(3)
  set processedAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasProcessedAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearProcessedAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureProcessedAt() => $_ensure(2);
}

/// RelayICEConfig is derived from a committed Controller grant. It is durable
/// before leaving the Edge and never contains commercial authority of its own.
class RelayICEConfig extends $pb.GeneratedMessage {
  factory RelayICEConfig({
    $core.String? reservationId,
    $core.Iterable<$core.String>? urls,
    $core.String? username,
    $core.String? credential,
    $0.Timestamp? expiresAt,
  }) {
    final result = create();
    if (reservationId != null) result.reservationId = reservationId;
    if (urls != null) result.urls.addAll(urls);
    if (username != null) result.username = username;
    if (credential != null) result.credential = credential;
    if (expiresAt != null) result.expiresAt = expiresAt;
    return result;
  }

  RelayICEConfig._();

  factory RelayICEConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayICEConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayICEConfig',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reservationId')
    ..pPS(2, _omitFieldNames ? '' : 'urls')
    ..aOS(3, _omitFieldNames ? '' : 'username')
    ..aOS(4, _omitFieldNames ? '' : 'credential')
    ..aOM<$0.Timestamp>(5, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayICEConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayICEConfig copyWith(void Function(RelayICEConfig) updates) =>
      super.copyWith((message) => updates(message as RelayICEConfig))
          as RelayICEConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayICEConfig create() => RelayICEConfig._();
  @$core.override
  RelayICEConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayICEConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayICEConfig>(create);
  static RelayICEConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reservationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reservationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReservationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReservationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get urls => $_getList(1);

  @$pb.TagNumber(3)
  $core.String get username => $_getSZ(2);
  @$pb.TagNumber(3)
  set username($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUsername() => $_has(2);
  @$pb.TagNumber(3)
  void clearUsername() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get credential => $_getSZ(3);
  @$pb.TagNumber(4)
  set credential($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCredential() => $_has(3);
  @$pb.TagNumber(4)
  void clearCredential() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.Timestamp get expiresAt => $_getN(4);
  @$pb.TagNumber(5)
  set expiresAt($0.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasExpiresAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearExpiresAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Timestamp ensureExpiresAt() => $_ensure(4);
}

/// RelayJournalRecord is the single durable bbolt value for one reservation.
class RelayJournalRecord extends $pb.GeneratedMessage {
  factory RelayJournalRecord({
    $core.int? schemaVersion,
    RelayJournalStage? stage,
    RelayReserveRequest? reserveRequest,
    RelayGrant? grant,
    $fixnum.Int64? pendingRenewSequence,
    RelaySettlement? settlement,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (stage != null) result.stage = stage;
    if (reserveRequest != null) result.reserveRequest = reserveRequest;
    if (grant != null) result.grant = grant;
    if (pendingRenewSequence != null)
      result.pendingRenewSequence = pendingRenewSequence;
    if (settlement != null) result.settlement = settlement;
    return result;
  }

  RelayJournalRecord._();

  factory RelayJournalRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RelayJournalRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RelayJournalRecord',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aE<RelayJournalStage>(2, _omitFieldNames ? '' : 'stage',
        enumValues: RelayJournalStage.values)
    ..aOM<RelayReserveRequest>(3, _omitFieldNames ? '' : 'reserveRequest',
        subBuilder: RelayReserveRequest.create)
    ..aOM<RelayGrant>(4, _omitFieldNames ? '' : 'grant',
        subBuilder: RelayGrant.create)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'pendingRenewSequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<RelaySettlement>(6, _omitFieldNames ? '' : 'settlement',
        subBuilder: RelaySettlement.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayJournalRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RelayJournalRecord copyWith(void Function(RelayJournalRecord) updates) =>
      super.copyWith((message) => updates(message as RelayJournalRecord))
          as RelayJournalRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RelayJournalRecord create() => RelayJournalRecord._();
  @$core.override
  RelayJournalRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RelayJournalRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RelayJournalRecord>(create);
  static RelayJournalRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  RelayJournalStage get stage => $_getN(1);
  @$pb.TagNumber(2)
  set stage(RelayJournalStage value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasStage() => $_has(1);
  @$pb.TagNumber(2)
  void clearStage() => $_clearField(2);

  @$pb.TagNumber(3)
  RelayReserveRequest get reserveRequest => $_getN(2);
  @$pb.TagNumber(3)
  set reserveRequest(RelayReserveRequest value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasReserveRequest() => $_has(2);
  @$pb.TagNumber(3)
  void clearReserveRequest() => $_clearField(3);
  @$pb.TagNumber(3)
  RelayReserveRequest ensureReserveRequest() => $_ensure(2);

  @$pb.TagNumber(4)
  RelayGrant get grant => $_getN(3);
  @$pb.TagNumber(4)
  set grant(RelayGrant value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasGrant() => $_has(3);
  @$pb.TagNumber(4)
  void clearGrant() => $_clearField(4);
  @$pb.TagNumber(4)
  RelayGrant ensureGrant() => $_ensure(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get pendingRenewSequence => $_getI64(4);
  @$pb.TagNumber(5)
  set pendingRenewSequence($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPendingRenewSequence() => $_has(4);
  @$pb.TagNumber(5)
  void clearPendingRenewSequence() => $_clearField(5);

  @$pb.TagNumber(6)
  RelaySettlement get settlement => $_getN(5);
  @$pb.TagNumber(6)
  set settlement(RelaySettlement value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasSettlement() => $_has(5);
  @$pb.TagNumber(6)
  void clearSettlement() => $_clearField(6);
  @$pb.TagNumber(6)
  RelaySettlement ensureSettlement() => $_ensure(5);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
