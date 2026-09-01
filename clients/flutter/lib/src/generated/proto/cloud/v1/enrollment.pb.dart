// This is a generated file - do not edit.
//
// Generated from cloud/v1/enrollment.proto.

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
    as $0;

import 'common.pb.dart' as $1;
import 'enrollment.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'enrollment.pbenum.dart';

/// DaemonRecord 是 Controller 持久 daemon identity 的公开投影，不包含私钥或实时拓扑。
class DaemonRecord extends $pb.GeneratedMessage {
  factory DaemonRecord({
    $core.String? daemonId,
    $core.String? accountId,
    $core.String? accountName,
    $core.String? displayName,
    $core.String? deviceId,
    $core.String? deviceFingerprint,
    DaemonState? state,
    $fixnum.Int64? stateRevision,
    $0.Timestamp? createdAt,
    $0.Timestamp? updatedAt,
    $core.String? preferredEdgeId,
    $fixnum.Int64? edgePreferenceRevision,
    $0.Timestamp? edgePreferenceUpdatedAt,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (accountId != null) result.accountId = accountId;
    if (accountName != null) result.accountName = accountName;
    if (displayName != null) result.displayName = displayName;
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (state != null) result.state = state;
    if (stateRevision != null) result.stateRevision = stateRevision;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (preferredEdgeId != null) result.preferredEdgeId = preferredEdgeId;
    if (edgePreferenceRevision != null)
      result.edgePreferenceRevision = edgePreferenceRevision;
    if (edgePreferenceUpdatedAt != null)
      result.edgePreferenceUpdatedAt = edgePreferenceUpdatedAt;
    return result;
  }

  DaemonRecord._();

  factory DaemonRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonRecord',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'accountName')
    ..aOS(4, _omitFieldNames ? '' : 'displayName')
    ..aOS(5, _omitFieldNames ? '' : 'deviceId')
    ..aOS(6, _omitFieldNames ? '' : 'deviceFingerprint')
    ..aE<DaemonState>(7, _omitFieldNames ? '' : 'state',
        enumValues: DaemonState.values)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'stateRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $0.Timestamp.create)
    ..aOS(11, _omitFieldNames ? '' : 'preferredEdgeId')
    ..a<$fixnum.Int64>(12, _omitFieldNames ? '' : 'edgePreferenceRevision',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(13, _omitFieldNames ? '' : 'edgePreferenceUpdatedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonRecord copyWith(void Function(DaemonRecord) updates) =>
      super.copyWith((message) => updates(message as DaemonRecord))
          as DaemonRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonRecord create() => DaemonRecord._();
  @$core.override
  DaemonRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonRecord>(create);
  static DaemonRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get accountName => $_getSZ(2);
  @$pb.TagNumber(3)
  set accountName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAccountName() => $_has(2);
  @$pb.TagNumber(3)
  void clearAccountName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get displayName => $_getSZ(3);
  @$pb.TagNumber(4)
  set displayName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDisplayName() => $_has(3);
  @$pb.TagNumber(4)
  void clearDisplayName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get deviceId => $_getSZ(4);
  @$pb.TagNumber(5)
  set deviceId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDeviceId() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeviceId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get deviceFingerprint => $_getSZ(5);
  @$pb.TagNumber(6)
  set deviceFingerprint($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDeviceFingerprint() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeviceFingerprint() => $_clearField(6);

  @$pb.TagNumber(7)
  DaemonState get state => $_getN(6);
  @$pb.TagNumber(7)
  set state(DaemonState value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasState() => $_has(6);
  @$pb.TagNumber(7)
  void clearState() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get stateRevision => $_getI64(7);
  @$pb.TagNumber(8)
  set stateRevision($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasStateRevision() => $_has(7);
  @$pb.TagNumber(8)
  void clearStateRevision() => $_clearField(8);

  @$pb.TagNumber(9)
  $0.Timestamp get createdAt => $_getN(8);
  @$pb.TagNumber(9)
  set createdAt($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasCreatedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearCreatedAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureCreatedAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $0.Timestamp get updatedAt => $_getN(9);
  @$pb.TagNumber(10)
  set updatedAt($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasUpdatedAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearUpdatedAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensureUpdatedAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $core.String get preferredEdgeId => $_getSZ(10);
  @$pb.TagNumber(11)
  set preferredEdgeId($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasPreferredEdgeId() => $_has(10);
  @$pb.TagNumber(11)
  void clearPreferredEdgeId() => $_clearField(11);

  @$pb.TagNumber(12)
  $fixnum.Int64 get edgePreferenceRevision => $_getI64(11);
  @$pb.TagNumber(12)
  set edgePreferenceRevision($fixnum.Int64 value) => $_setInt64(11, value);
  @$pb.TagNumber(12)
  $core.bool hasEdgePreferenceRevision() => $_has(11);
  @$pb.TagNumber(12)
  void clearEdgePreferenceRevision() => $_clearField(12);

  @$pb.TagNumber(13)
  $0.Timestamp get edgePreferenceUpdatedAt => $_getN(12);
  @$pb.TagNumber(13)
  set edgePreferenceUpdatedAt($0.Timestamp value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasEdgePreferenceUpdatedAt() => $_has(12);
  @$pb.TagNumber(13)
  void clearEdgePreferenceUpdatedAt() => $_clearField(13);
  @$pb.TagNumber(13)
  $0.Timestamp ensureEdgePreferenceUpdatedAt() => $_ensure(12);
}

/// DaemonStateRecord 是 Controller 向 Edge 广播的最小持久状态。
class DaemonStateRecord extends $pb.GeneratedMessage {
  factory DaemonStateRecord({
    $core.String? daemonId,
    DaemonState? state,
    $fixnum.Int64? stateRevision,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (state != null) result.state = state;
    if (stateRevision != null) result.stateRevision = stateRevision;
    return result;
  }

  DaemonStateRecord._();

  factory DaemonStateRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateRecord',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aE<DaemonState>(2, _omitFieldNames ? '' : 'state',
        enumValues: DaemonState.values)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'stateRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateRecord copyWith(void Function(DaemonStateRecord) updates) =>
      super.copyWith((message) => updates(message as DaemonStateRecord))
          as DaemonStateRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateRecord create() => DaemonStateRecord._();
  @$core.override
  DaemonStateRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateRecord>(create);
  static DaemonStateRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  DaemonState get state => $_getN(1);
  @$pb.TagNumber(2)
  set state(DaemonState value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get stateRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set stateRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasStateRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearStateRevision() => $_clearField(3);
}

/// DaemonRuntimeProjection 来自 Controller Directory 内存，Controller 重启后必须由 Edge 重建。
class DaemonRuntimeProjection extends $pb.GeneratedMessage {
  factory DaemonRuntimeProjection({
    $core.bool? online,
    $core.String? edgeId,
    $core.String? edgeName,
    $core.String? edgeRegion,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? generation,
    $core.String? edgePublicEndpoint,
  }) {
    final result = create();
    if (online != null) result.online = online;
    if (edgeId != null) result.edgeId = edgeId;
    if (edgeName != null) result.edgeName = edgeName;
    if (edgeRegion != null) result.edgeRegion = edgeRegion;
    if (bootId != null) result.bootId = bootId;
    if (connectionId != null) result.connectionId = connectionId;
    if (generation != null) result.generation = generation;
    if (edgePublicEndpoint != null)
      result.edgePublicEndpoint = edgePublicEndpoint;
    return result;
  }

  DaemonRuntimeProjection._();

  factory DaemonRuntimeProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonRuntimeProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonRuntimeProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'online')
    ..aOS(2, _omitFieldNames ? '' : 'edgeId')
    ..aOS(3, _omitFieldNames ? '' : 'edgeName')
    ..aOS(4, _omitFieldNames ? '' : 'edgeRegion')
    ..aOS(5, _omitFieldNames ? '' : 'bootId')
    ..aOS(6, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(8, _omitFieldNames ? '' : 'edgePublicEndpoint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonRuntimeProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonRuntimeProjection copyWith(
          void Function(DaemonRuntimeProjection) updates) =>
      super.copyWith((message) => updates(message as DaemonRuntimeProjection))
          as DaemonRuntimeProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonRuntimeProjection create() => DaemonRuntimeProjection._();
  @$core.override
  DaemonRuntimeProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonRuntimeProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonRuntimeProjection>(create);
  static DaemonRuntimeProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get online => $_getBF(0);
  @$pb.TagNumber(1)
  set online($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOnline() => $_has(0);
  @$pb.TagNumber(1)
  void clearOnline() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get edgeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set edgeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEdgeId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEdgeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get edgeName => $_getSZ(2);
  @$pb.TagNumber(3)
  set edgeName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEdgeName() => $_has(2);
  @$pb.TagNumber(3)
  void clearEdgeName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get edgeRegion => $_getSZ(3);
  @$pb.TagNumber(4)
  set edgeRegion($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEdgeRegion() => $_has(3);
  @$pb.TagNumber(4)
  void clearEdgeRegion() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get bootId => $_getSZ(4);
  @$pb.TagNumber(5)
  set bootId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBootId() => $_has(4);
  @$pb.TagNumber(5)
  void clearBootId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get connectionId => $_getSZ(5);
  @$pb.TagNumber(6)
  set connectionId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasConnectionId() => $_has(5);
  @$pb.TagNumber(6)
  void clearConnectionId() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get generation => $_getI64(6);
  @$pb.TagNumber(7)
  set generation($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasGeneration() => $_has(6);
  @$pb.TagNumber(7)
  void clearGeneration() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get edgePublicEndpoint => $_getSZ(7);
  @$pb.TagNumber(8)
  set edgePublicEndpoint($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasEdgePublicEndpoint() => $_has(7);
  @$pb.TagNumber(8)
  void clearEdgePublicEndpoint() => $_clearField(8);
}

class ManagedDaemon extends $pb.GeneratedMessage {
  factory ManagedDaemon({
    DaemonRecord? daemon,
    DaemonRuntimeProjection? runtime,
  }) {
    final result = create();
    if (daemon != null) result.daemon = daemon;
    if (runtime != null) result.runtime = runtime;
    return result;
  }

  ManagedDaemon._();

  factory ManagedDaemon.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ManagedDaemon.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ManagedDaemon',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonRecord>(1, _omitFieldNames ? '' : 'daemon',
        subBuilder: DaemonRecord.create)
    ..aOM<DaemonRuntimeProjection>(2, _omitFieldNames ? '' : 'runtime',
        subBuilder: DaemonRuntimeProjection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ManagedDaemon clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ManagedDaemon copyWith(void Function(ManagedDaemon) updates) =>
      super.copyWith((message) => updates(message as ManagedDaemon))
          as ManagedDaemon;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ManagedDaemon create() => ManagedDaemon._();
  @$core.override
  ManagedDaemon createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ManagedDaemon getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ManagedDaemon>(create);
  static ManagedDaemon? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonRecord get daemon => $_getN(0);
  @$pb.TagNumber(1)
  set daemon(DaemonRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemon() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemon() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonRecord ensureDaemon() => $_ensure(0);

  @$pb.TagNumber(2)
  DaemonRuntimeProjection get runtime => $_getN(1);
  @$pb.TagNumber(2)
  set runtime(DaemonRuntimeProjection value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRuntime() => $_has(1);
  @$pb.TagNumber(2)
  void clearRuntime() => $_clearField(2);
  @$pb.TagNumber(2)
  DaemonRuntimeProjection ensureRuntime() => $_ensure(1);
}

/// DaemonEdgeMeasurement 是 daemon 从自身网络位置测得的短期连接质量。
/// connection_failure_rate 是 TCP/TLS/gRPC 探测失败率，不表示 UDP 丢包率。
class DaemonEdgeMeasurement extends $pb.GeneratedMessage {
  factory DaemonEdgeMeasurement({
    $core.String? edgeId,
    $core.bool? reachable,
    $core.int? connectLatencyMs,
    $core.double? connectionFailureRate,
    $core.int? sampleCount,
    $0.Timestamp? measuredAt,
  }) {
    final result = create();
    if (edgeId != null) result.edgeId = edgeId;
    if (reachable != null) result.reachable = reachable;
    if (connectLatencyMs != null) result.connectLatencyMs = connectLatencyMs;
    if (connectionFailureRate != null)
      result.connectionFailureRate = connectionFailureRate;
    if (sampleCount != null) result.sampleCount = sampleCount;
    if (measuredAt != null) result.measuredAt = measuredAt;
    return result;
  }

  DaemonEdgeMeasurement._();

  factory DaemonEdgeMeasurement.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonEdgeMeasurement.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonEdgeMeasurement',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..aOB(2, _omitFieldNames ? '' : 'reachable')
    ..aI(3, _omitFieldNames ? '' : 'connectLatencyMs',
        fieldType: $pb.PbFieldType.OU3)
    ..aD(4, _omitFieldNames ? '' : 'connectionFailureRate')
    ..aI(5, _omitFieldNames ? '' : 'sampleCount',
        fieldType: $pb.PbFieldType.OU3)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'measuredAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeMeasurement clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeMeasurement copyWith(
          void Function(DaemonEdgeMeasurement) updates) =>
      super.copyWith((message) => updates(message as DaemonEdgeMeasurement))
          as DaemonEdgeMeasurement;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonEdgeMeasurement create() => DaemonEdgeMeasurement._();
  @$core.override
  DaemonEdgeMeasurement createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonEdgeMeasurement getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonEdgeMeasurement>(create);
  static DaemonEdgeMeasurement? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get reachable => $_getBF(1);
  @$pb.TagNumber(2)
  set reachable($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReachable() => $_has(1);
  @$pb.TagNumber(2)
  void clearReachable() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get connectLatencyMs => $_getIZ(2);
  @$pb.TagNumber(3)
  set connectLatencyMs($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasConnectLatencyMs() => $_has(2);
  @$pb.TagNumber(3)
  void clearConnectLatencyMs() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get connectionFailureRate => $_getN(3);
  @$pb.TagNumber(4)
  set connectionFailureRate($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasConnectionFailureRate() => $_has(3);
  @$pb.TagNumber(4)
  void clearConnectionFailureRate() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get sampleCount => $_getIZ(4);
  @$pb.TagNumber(5)
  set sampleCount($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSampleCount() => $_has(4);
  @$pb.TagNumber(5)
  void clearSampleCount() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get measuredAt => $_getN(5);
  @$pb.TagNumber(6)
  set measuredAt($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasMeasuredAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearMeasuredAt() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureMeasuredAt() => $_ensure(5);
}

/// DaemonEdgeCandidate 是面向 daemon owner 的安全 Edge 投影。
class DaemonEdgeCandidate extends $pb.GeneratedMessage {
  factory DaemonEdgeCandidate({
    EdgeLocator? locator,
    $core.bool? online,
    $core.bool? eligible,
    $core.bool? preferred,
    $core.bool? current,
    DaemonEdgeMeasurement? measurement,
    $core.double? score,
    $core.String? status,
  }) {
    final result = create();
    if (locator != null) result.locator = locator;
    if (online != null) result.online = online;
    if (eligible != null) result.eligible = eligible;
    if (preferred != null) result.preferred = preferred;
    if (current != null) result.current = current;
    if (measurement != null) result.measurement = measurement;
    if (score != null) result.score = score;
    if (status != null) result.status = status;
    return result;
  }

  DaemonEdgeCandidate._();

  factory DaemonEdgeCandidate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonEdgeCandidate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonEdgeCandidate',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<EdgeLocator>(1, _omitFieldNames ? '' : 'locator',
        subBuilder: EdgeLocator.create)
    ..aOB(2, _omitFieldNames ? '' : 'online')
    ..aOB(3, _omitFieldNames ? '' : 'eligible')
    ..aOB(6, _omitFieldNames ? '' : 'preferred')
    ..aOB(7, _omitFieldNames ? '' : 'current')
    ..aOM<DaemonEdgeMeasurement>(8, _omitFieldNames ? '' : 'measurement',
        subBuilder: DaemonEdgeMeasurement.create)
    ..aD(9, _omitFieldNames ? '' : 'score')
    ..aOS(10, _omitFieldNames ? '' : 'status')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeCandidate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeCandidate copyWith(void Function(DaemonEdgeCandidate) updates) =>
      super.copyWith((message) => updates(message as DaemonEdgeCandidate))
          as DaemonEdgeCandidate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonEdgeCandidate create() => DaemonEdgeCandidate._();
  @$core.override
  DaemonEdgeCandidate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonEdgeCandidate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonEdgeCandidate>(create);
  static DaemonEdgeCandidate? _defaultInstance;

  @$pb.TagNumber(1)
  EdgeLocator get locator => $_getN(0);
  @$pb.TagNumber(1)
  set locator(EdgeLocator value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasLocator() => $_has(0);
  @$pb.TagNumber(1)
  void clearLocator() => $_clearField(1);
  @$pb.TagNumber(1)
  EdgeLocator ensureLocator() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get online => $_getBF(1);
  @$pb.TagNumber(2)
  set online($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOnline() => $_has(1);
  @$pb.TagNumber(2)
  void clearOnline() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get eligible => $_getBF(2);
  @$pb.TagNumber(3)
  set eligible($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEligible() => $_has(2);
  @$pb.TagNumber(3)
  void clearEligible() => $_clearField(3);

  @$pb.TagNumber(6)
  $core.bool get preferred => $_getBF(3);
  @$pb.TagNumber(6)
  set preferred($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(6)
  $core.bool hasPreferred() => $_has(3);
  @$pb.TagNumber(6)
  void clearPreferred() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get current => $_getBF(4);
  @$pb.TagNumber(7)
  set current($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(7)
  $core.bool hasCurrent() => $_has(4);
  @$pb.TagNumber(7)
  void clearCurrent() => $_clearField(7);

  @$pb.TagNumber(8)
  DaemonEdgeMeasurement get measurement => $_getN(5);
  @$pb.TagNumber(8)
  set measurement(DaemonEdgeMeasurement value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasMeasurement() => $_has(5);
  @$pb.TagNumber(8)
  void clearMeasurement() => $_clearField(8);
  @$pb.TagNumber(8)
  DaemonEdgeMeasurement ensureMeasurement() => $_ensure(5);

  @$pb.TagNumber(9)
  $core.double get score => $_getN(6);
  @$pb.TagNumber(9)
  set score($core.double value) => $_setDouble(6, value);
  @$pb.TagNumber(9)
  $core.bool hasScore() => $_has(6);
  @$pb.TagNumber(9)
  void clearScore() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get status => $_getSZ(7);
  @$pb.TagNumber(10)
  set status($core.String value) => $_setString(7, value);
  @$pb.TagNumber(10)
  $core.bool hasStatus() => $_has(7);
  @$pb.TagNumber(10)
  void clearStatus() => $_clearField(10);
}

class DaemonEdgeSelection extends $pb.GeneratedMessage {
  factory DaemonEdgeSelection({
    $core.String? daemonId,
    $core.String? preferredEdgeId,
    $fixnum.Int64? preferenceRevision,
    $core.String? currentEdgeId,
    $core.String? selectedEdgeId,
    $core.Iterable<DaemonEdgeCandidate>? candidates,
    $0.Timestamp? evaluatedAt,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (preferredEdgeId != null) result.preferredEdgeId = preferredEdgeId;
    if (preferenceRevision != null)
      result.preferenceRevision = preferenceRevision;
    if (currentEdgeId != null) result.currentEdgeId = currentEdgeId;
    if (selectedEdgeId != null) result.selectedEdgeId = selectedEdgeId;
    if (candidates != null) result.candidates.addAll(candidates);
    if (evaluatedAt != null) result.evaluatedAt = evaluatedAt;
    return result;
  }

  DaemonEdgeSelection._();

  factory DaemonEdgeSelection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonEdgeSelection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonEdgeSelection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'preferredEdgeId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'preferenceRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(4, _omitFieldNames ? '' : 'currentEdgeId')
    ..aOS(5, _omitFieldNames ? '' : 'selectedEdgeId')
    ..pPM<DaemonEdgeCandidate>(6, _omitFieldNames ? '' : 'candidates',
        subBuilder: DaemonEdgeCandidate.create)
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'evaluatedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeSelection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEdgeSelection copyWith(void Function(DaemonEdgeSelection) updates) =>
      super.copyWith((message) => updates(message as DaemonEdgeSelection))
          as DaemonEdgeSelection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonEdgeSelection create() => DaemonEdgeSelection._();
  @$core.override
  DaemonEdgeSelection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonEdgeSelection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonEdgeSelection>(create);
  static DaemonEdgeSelection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get preferredEdgeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set preferredEdgeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPreferredEdgeId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPreferredEdgeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get preferenceRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set preferenceRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPreferenceRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearPreferenceRevision() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get currentEdgeId => $_getSZ(3);
  @$pb.TagNumber(4)
  set currentEdgeId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCurrentEdgeId() => $_has(3);
  @$pb.TagNumber(4)
  void clearCurrentEdgeId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get selectedEdgeId => $_getSZ(4);
  @$pb.TagNumber(5)
  set selectedEdgeId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSelectedEdgeId() => $_has(4);
  @$pb.TagNumber(5)
  void clearSelectedEdgeId() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<DaemonEdgeCandidate> get candidates => $_getList(5);

  @$pb.TagNumber(7)
  $0.Timestamp get evaluatedAt => $_getN(6);
  @$pb.TagNumber(7)
  set evaluatedAt($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasEvaluatedAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearEvaluatedAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensureEvaluatedAt() => $_ensure(6);
}

class CreateDaemonEnrollmentRequest extends $pb.GeneratedMessage {
  factory CreateDaemonEnrollmentRequest({
    $core.String? accountId,
    $core.String? accountName,
    $core.String? daemonName,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (accountName != null) result.accountName = accountName;
    if (daemonName != null) result.daemonName = daemonName;
    return result;
  }

  CreateDaemonEnrollmentRequest._();

  factory CreateDaemonEnrollmentRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateDaemonEnrollmentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateDaemonEnrollmentRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOS(2, _omitFieldNames ? '' : 'accountName')
    ..aOS(3, _omitFieldNames ? '' : 'daemonName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateDaemonEnrollmentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateDaemonEnrollmentRequest copyWith(
          void Function(CreateDaemonEnrollmentRequest) updates) =>
      super.copyWith(
              (message) => updates(message as CreateDaemonEnrollmentRequest))
          as CreateDaemonEnrollmentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateDaemonEnrollmentRequest create() =>
      CreateDaemonEnrollmentRequest._();
  @$core.override
  CreateDaemonEnrollmentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateDaemonEnrollmentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateDaemonEnrollmentRequest>(create);
  static CreateDaemonEnrollmentRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountName => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountName() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get daemonName => $_getSZ(2);
  @$pb.TagNumber(3)
  set daemonName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonName() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonName() => $_clearField(3);
}

class CreateDaemonEnrollmentResponse extends $pb.GeneratedMessage {
  factory CreateDaemonEnrollmentResponse({
    $core.String? accountId,
    $core.String? enrollmentCode,
    $0.Timestamp? expiresAt,
    $core.String? enrollCommand,
    $core.int? daemonCount,
    $core.int? daemonLimit,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (enrollmentCode != null) result.enrollmentCode = enrollmentCode;
    if (expiresAt != null) result.expiresAt = expiresAt;
    if (enrollCommand != null) result.enrollCommand = enrollCommand;
    if (daemonCount != null) result.daemonCount = daemonCount;
    if (daemonLimit != null) result.daemonLimit = daemonLimit;
    return result;
  }

  CreateDaemonEnrollmentResponse._();

  factory CreateDaemonEnrollmentResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateDaemonEnrollmentResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateDaemonEnrollmentResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOS(2, _omitFieldNames ? '' : 'enrollmentCode')
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..aOS(4, _omitFieldNames ? '' : 'enrollCommand')
    ..aI(5, _omitFieldNames ? '' : 'daemonCount',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(6, _omitFieldNames ? '' : 'daemonLimit',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateDaemonEnrollmentResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateDaemonEnrollmentResponse copyWith(
          void Function(CreateDaemonEnrollmentResponse) updates) =>
      super.copyWith(
              (message) => updates(message as CreateDaemonEnrollmentResponse))
          as CreateDaemonEnrollmentResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateDaemonEnrollmentResponse create() =>
      CreateDaemonEnrollmentResponse._();
  @$core.override
  CreateDaemonEnrollmentResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateDaemonEnrollmentResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateDaemonEnrollmentResponse>(create);
  static CreateDaemonEnrollmentResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get enrollmentCode => $_getSZ(1);
  @$pb.TagNumber(2)
  set enrollmentCode($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEnrollmentCode() => $_has(1);
  @$pb.TagNumber(2)
  void clearEnrollmentCode() => $_clearField(2);

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
  $core.String get enrollCommand => $_getSZ(3);
  @$pb.TagNumber(4)
  set enrollCommand($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEnrollCommand() => $_has(3);
  @$pb.TagNumber(4)
  void clearEnrollCommand() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get daemonCount => $_getIZ(4);
  @$pb.TagNumber(5)
  set daemonCount($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDaemonCount() => $_has(4);
  @$pb.TagNumber(5)
  void clearDaemonCount() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get daemonLimit => $_getIZ(5);
  @$pb.TagNumber(6)
  set daemonLimit($core.int value) => $_setUnsignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDaemonLimit() => $_has(5);
  @$pb.TagNumber(6)
  void clearDaemonLimit() => $_clearField(6);
}

class ListDaemonsRequest extends $pb.GeneratedMessage {
  factory ListDaemonsRequest({
    $core.int? pageSize,
    $core.String? cursor,
    $core.String? query,
    $core.String? edgeId,
  }) {
    final result = create();
    if (pageSize != null) result.pageSize = pageSize;
    if (cursor != null) result.cursor = cursor;
    if (query != null) result.query = query;
    if (edgeId != null) result.edgeId = edgeId;
    return result;
  }

  ListDaemonsRequest._();

  factory ListDaemonsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListDaemonsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListDaemonsRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'pageSize', fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'cursor')
    ..aOS(3, _omitFieldNames ? '' : 'query')
    ..aOS(4, _omitFieldNames ? '' : 'edgeId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListDaemonsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListDaemonsRequest copyWith(void Function(ListDaemonsRequest) updates) =>
      super.copyWith((message) => updates(message as ListDaemonsRequest))
          as ListDaemonsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListDaemonsRequest create() => ListDaemonsRequest._();
  @$core.override
  ListDaemonsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListDaemonsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListDaemonsRequest>(create);
  static ListDaemonsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get pageSize => $_getIZ(0);
  @$pb.TagNumber(1)
  set pageSize($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPageSize() => $_has(0);
  @$pb.TagNumber(1)
  void clearPageSize() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get cursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set cursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearCursor() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get query => $_getSZ(2);
  @$pb.TagNumber(3)
  set query($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasQuery() => $_has(2);
  @$pb.TagNumber(3)
  void clearQuery() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get edgeId => $_getSZ(3);
  @$pb.TagNumber(4)
  set edgeId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEdgeId() => $_has(3);
  @$pb.TagNumber(4)
  void clearEdgeId() => $_clearField(4);
}

class ListDaemonsResponse extends $pb.GeneratedMessage {
  factory ListDaemonsResponse({
    $core.Iterable<ManagedDaemon>? daemons,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (daemons != null) result.daemons.addAll(daemons);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListDaemonsResponse._();

  factory ListDaemonsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListDaemonsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListDaemonsResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<ManagedDaemon>(1, _omitFieldNames ? '' : 'daemons',
        subBuilder: ManagedDaemon.create)
    ..aOS(2, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListDaemonsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListDaemonsResponse copyWith(void Function(ListDaemonsResponse) updates) =>
      super.copyWith((message) => updates(message as ListDaemonsResponse))
          as ListDaemonsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListDaemonsResponse create() => ListDaemonsResponse._();
  @$core.override
  ListDaemonsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListDaemonsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListDaemonsResponse>(create);
  static ListDaemonsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ManagedDaemon> get daemons => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
}

/// CreateMyDaemonEnrollmentRequest 只允许用户命名 daemon；账号来自认证 session。
class CreateMyDaemonEnrollmentRequest extends $pb.GeneratedMessage {
  factory CreateMyDaemonEnrollmentRequest({
    $core.String? daemonName,
  }) {
    final result = create();
    if (daemonName != null) result.daemonName = daemonName;
    return result;
  }

  CreateMyDaemonEnrollmentRequest._();

  factory CreateMyDaemonEnrollmentRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateMyDaemonEnrollmentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateMyDaemonEnrollmentRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateMyDaemonEnrollmentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateMyDaemonEnrollmentRequest copyWith(
          void Function(CreateMyDaemonEnrollmentRequest) updates) =>
      super.copyWith(
              (message) => updates(message as CreateMyDaemonEnrollmentRequest))
          as CreateMyDaemonEnrollmentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateMyDaemonEnrollmentRequest create() =>
      CreateMyDaemonEnrollmentRequest._();
  @$core.override
  CreateMyDaemonEnrollmentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateMyDaemonEnrollmentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateMyDaemonEnrollmentRequest>(
          create);
  static CreateMyDaemonEnrollmentRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonName => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonName() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonName() => $_clearField(1);
}

class ListMyDaemonsRequest extends $pb.GeneratedMessage {
  factory ListMyDaemonsRequest() => create();

  ListMyDaemonsRequest._();

  factory ListMyDaemonsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListMyDaemonsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListMyDaemonsRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonsRequest copyWith(void Function(ListMyDaemonsRequest) updates) =>
      super.copyWith((message) => updates(message as ListMyDaemonsRequest))
          as ListMyDaemonsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListMyDaemonsRequest create() => ListMyDaemonsRequest._();
  @$core.override
  ListMyDaemonsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListMyDaemonsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListMyDaemonsRequest>(create);
  static ListMyDaemonsRequest? _defaultInstance;
}

class ListMyDaemonsResponse extends $pb.GeneratedMessage {
  factory ListMyDaemonsResponse({
    $core.Iterable<ManagedDaemon>? daemons,
    $core.int? daemonCount,
    $core.int? daemonLimit,
  }) {
    final result = create();
    if (daemons != null) result.daemons.addAll(daemons);
    if (daemonCount != null) result.daemonCount = daemonCount;
    if (daemonLimit != null) result.daemonLimit = daemonLimit;
    return result;
  }

  ListMyDaemonsResponse._();

  factory ListMyDaemonsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListMyDaemonsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListMyDaemonsResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<ManagedDaemon>(1, _omitFieldNames ? '' : 'daemons',
        subBuilder: ManagedDaemon.create)
    ..aI(2, _omitFieldNames ? '' : 'daemonCount',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'daemonLimit',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonsResponse copyWith(
          void Function(ListMyDaemonsResponse) updates) =>
      super.copyWith((message) => updates(message as ListMyDaemonsResponse))
          as ListMyDaemonsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListMyDaemonsResponse create() => ListMyDaemonsResponse._();
  @$core.override
  ListMyDaemonsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListMyDaemonsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListMyDaemonsResponse>(create);
  static ListMyDaemonsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ManagedDaemon> get daemons => $_getList(0);

  @$pb.TagNumber(2)
  $core.int get daemonCount => $_getIZ(1);
  @$pb.TagNumber(2)
  set daemonCount($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonCount() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonCount() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get daemonLimit => $_getIZ(2);
  @$pb.TagNumber(3)
  set daemonLimit($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonLimit() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonLimit() => $_clearField(3);
}

class ChangeMyDaemonStateRequest extends $pb.GeneratedMessage {
  factory ChangeMyDaemonStateRequest({
    $core.String? daemonId,
    DaemonState? targetState,
    $fixnum.Int64? expectedStateRevision,
    $core.String? reason,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (targetState != null) result.targetState = targetState;
    if (expectedStateRevision != null)
      result.expectedStateRevision = expectedStateRevision;
    if (reason != null) result.reason = reason;
    return result;
  }

  ChangeMyDaemonStateRequest._();

  factory ChangeMyDaemonStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ChangeMyDaemonStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeMyDaemonStateRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aE<DaemonState>(2, _omitFieldNames ? '' : 'targetState',
        enumValues: DaemonState.values)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'expectedStateRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(4, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonStateRequest copyWith(
          void Function(ChangeMyDaemonStateRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ChangeMyDaemonStateRequest))
          as ChangeMyDaemonStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonStateRequest create() => ChangeMyDaemonStateRequest._();
  @$core.override
  ChangeMyDaemonStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ChangeMyDaemonStateRequest>(create);
  static ChangeMyDaemonStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  DaemonState get targetState => $_getN(1);
  @$pb.TagNumber(2)
  set targetState(DaemonState value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTargetState() => $_has(1);
  @$pb.TagNumber(2)
  void clearTargetState() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get expectedStateRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set expectedStateRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedStateRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpectedStateRevision() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get reason => $_getSZ(3);
  @$pb.TagNumber(4)
  set reason($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReason() => $_has(3);
  @$pb.TagNumber(4)
  void clearReason() => $_clearField(4);
}

class ChangeMyDaemonStateResponse extends $pb.GeneratedMessage {
  factory ChangeMyDaemonStateResponse({
    DaemonRecord? daemon,
  }) {
    final result = create();
    if (daemon != null) result.daemon = daemon;
    return result;
  }

  ChangeMyDaemonStateResponse._();

  factory ChangeMyDaemonStateResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ChangeMyDaemonStateResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeMyDaemonStateResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonRecord>(1, _omitFieldNames ? '' : 'daemon',
        subBuilder: DaemonRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonStateResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonStateResponse copyWith(
          void Function(ChangeMyDaemonStateResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ChangeMyDaemonStateResponse))
          as ChangeMyDaemonStateResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonStateResponse create() =>
      ChangeMyDaemonStateResponse._();
  @$core.override
  ChangeMyDaemonStateResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonStateResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ChangeMyDaemonStateResponse>(create);
  static ChangeMyDaemonStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonRecord get daemon => $_getN(0);
  @$pb.TagNumber(1)
  set daemon(DaemonRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemon() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemon() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonRecord ensureDaemon() => $_ensure(0);
}

class ListMyDaemonEdgesRequest extends $pb.GeneratedMessage {
  factory ListMyDaemonEdgesRequest({
    $core.String? daemonId,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    return result;
  }

  ListMyDaemonEdgesRequest._();

  factory ListMyDaemonEdgesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListMyDaemonEdgesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListMyDaemonEdgesRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonEdgesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonEdgesRequest copyWith(
          void Function(ListMyDaemonEdgesRequest) updates) =>
      super.copyWith((message) => updates(message as ListMyDaemonEdgesRequest))
          as ListMyDaemonEdgesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListMyDaemonEdgesRequest create() => ListMyDaemonEdgesRequest._();
  @$core.override
  ListMyDaemonEdgesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListMyDaemonEdgesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListMyDaemonEdgesRequest>(create);
  static ListMyDaemonEdgesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);
}

class ListMyDaemonEdgesResponse extends $pb.GeneratedMessage {
  factory ListMyDaemonEdgesResponse({
    DaemonEdgeSelection? selection,
  }) {
    final result = create();
    if (selection != null) result.selection = selection;
    return result;
  }

  ListMyDaemonEdgesResponse._();

  factory ListMyDaemonEdgesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListMyDaemonEdgesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListMyDaemonEdgesResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonEdgeSelection>(1, _omitFieldNames ? '' : 'selection',
        subBuilder: DaemonEdgeSelection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonEdgesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListMyDaemonEdgesResponse copyWith(
          void Function(ListMyDaemonEdgesResponse) updates) =>
      super.copyWith((message) => updates(message as ListMyDaemonEdgesResponse))
          as ListMyDaemonEdgesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListMyDaemonEdgesResponse create() => ListMyDaemonEdgesResponse._();
  @$core.override
  ListMyDaemonEdgesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListMyDaemonEdgesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListMyDaemonEdgesResponse>(create);
  static ListMyDaemonEdgesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonEdgeSelection get selection => $_getN(0);
  @$pb.TagNumber(1)
  set selection(DaemonEdgeSelection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSelection() => $_has(0);
  @$pb.TagNumber(1)
  void clearSelection() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonEdgeSelection ensureSelection() => $_ensure(0);
}

class ChangeMyDaemonEdgePreferenceRequest extends $pb.GeneratedMessage {
  factory ChangeMyDaemonEdgePreferenceRequest({
    $core.String? daemonId,
    $core.String? preferredEdgeId,
    $fixnum.Int64? expectedPreferenceRevision,
    $core.bool? reselectNow,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (preferredEdgeId != null) result.preferredEdgeId = preferredEdgeId;
    if (expectedPreferenceRevision != null)
      result.expectedPreferenceRevision = expectedPreferenceRevision;
    if (reselectNow != null) result.reselectNow = reselectNow;
    return result;
  }

  ChangeMyDaemonEdgePreferenceRequest._();

  factory ChangeMyDaemonEdgePreferenceRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ChangeMyDaemonEdgePreferenceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeMyDaemonEdgePreferenceRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'preferredEdgeId')
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'expectedPreferenceRevision',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(4, _omitFieldNames ? '' : 'reselectNow')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonEdgePreferenceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonEdgePreferenceRequest copyWith(
          void Function(ChangeMyDaemonEdgePreferenceRequest) updates) =>
      super.copyWith((message) =>
              updates(message as ChangeMyDaemonEdgePreferenceRequest))
          as ChangeMyDaemonEdgePreferenceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonEdgePreferenceRequest create() =>
      ChangeMyDaemonEdgePreferenceRequest._();
  @$core.override
  ChangeMyDaemonEdgePreferenceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonEdgePreferenceRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<
          ChangeMyDaemonEdgePreferenceRequest>(create);
  static ChangeMyDaemonEdgePreferenceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get preferredEdgeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set preferredEdgeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPreferredEdgeId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPreferredEdgeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get expectedPreferenceRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set expectedPreferenceRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedPreferenceRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpectedPreferenceRevision() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get reselectNow => $_getBF(3);
  @$pb.TagNumber(4)
  set reselectNow($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReselectNow() => $_has(3);
  @$pb.TagNumber(4)
  void clearReselectNow() => $_clearField(4);
}

class ChangeMyDaemonEdgePreferenceResponse extends $pb.GeneratedMessage {
  factory ChangeMyDaemonEdgePreferenceResponse({
    DaemonEdgeSelection? selection,
    $core.bool? reselectAccepted,
    $core.String? message,
  }) {
    final result = create();
    if (selection != null) result.selection = selection;
    if (reselectAccepted != null) result.reselectAccepted = reselectAccepted;
    if (message != null) result.message = message;
    return result;
  }

  ChangeMyDaemonEdgePreferenceResponse._();

  factory ChangeMyDaemonEdgePreferenceResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ChangeMyDaemonEdgePreferenceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeMyDaemonEdgePreferenceResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonEdgeSelection>(1, _omitFieldNames ? '' : 'selection',
        subBuilder: DaemonEdgeSelection.create)
    ..aOB(2, _omitFieldNames ? '' : 'reselectAccepted')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonEdgePreferenceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMyDaemonEdgePreferenceResponse copyWith(
          void Function(ChangeMyDaemonEdgePreferenceResponse) updates) =>
      super.copyWith((message) =>
              updates(message as ChangeMyDaemonEdgePreferenceResponse))
          as ChangeMyDaemonEdgePreferenceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonEdgePreferenceResponse create() =>
      ChangeMyDaemonEdgePreferenceResponse._();
  @$core.override
  ChangeMyDaemonEdgePreferenceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ChangeMyDaemonEdgePreferenceResponse getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<
          ChangeMyDaemonEdgePreferenceResponse>(create);
  static ChangeMyDaemonEdgePreferenceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonEdgeSelection get selection => $_getN(0);
  @$pb.TagNumber(1)
  set selection(DaemonEdgeSelection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSelection() => $_has(0);
  @$pb.TagNumber(1)
  void clearSelection() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonEdgeSelection ensureSelection() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get reselectAccepted => $_getBF(1);
  @$pb.TagNumber(2)
  set reselectAccepted($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReselectAccepted() => $_has(1);
  @$pb.TagNumber(2)
  void clearReselectAccepted() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
}

class ReselectMyDaemonEdgeRequest extends $pb.GeneratedMessage {
  factory ReselectMyDaemonEdgeRequest({
    $core.String? daemonId,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    return result;
  }

  ReselectMyDaemonEdgeRequest._();

  factory ReselectMyDaemonEdgeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReselectMyDaemonEdgeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReselectMyDaemonEdgeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReselectMyDaemonEdgeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReselectMyDaemonEdgeRequest copyWith(
          void Function(ReselectMyDaemonEdgeRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ReselectMyDaemonEdgeRequest))
          as ReselectMyDaemonEdgeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReselectMyDaemonEdgeRequest create() =>
      ReselectMyDaemonEdgeRequest._();
  @$core.override
  ReselectMyDaemonEdgeRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReselectMyDaemonEdgeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReselectMyDaemonEdgeRequest>(create);
  static ReselectMyDaemonEdgeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);
}

class ReselectMyDaemonEdgeResponse extends $pb.GeneratedMessage {
  factory ReselectMyDaemonEdgeResponse({
    DaemonEdgeSelection? selection,
    $core.bool? reselectAccepted,
    $core.String? message,
  }) {
    final result = create();
    if (selection != null) result.selection = selection;
    if (reselectAccepted != null) result.reselectAccepted = reselectAccepted;
    if (message != null) result.message = message;
    return result;
  }

  ReselectMyDaemonEdgeResponse._();

  factory ReselectMyDaemonEdgeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReselectMyDaemonEdgeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReselectMyDaemonEdgeResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonEdgeSelection>(1, _omitFieldNames ? '' : 'selection',
        subBuilder: DaemonEdgeSelection.create)
    ..aOB(2, _omitFieldNames ? '' : 'reselectAccepted')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReselectMyDaemonEdgeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReselectMyDaemonEdgeResponse copyWith(
          void Function(ReselectMyDaemonEdgeResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ReselectMyDaemonEdgeResponse))
          as ReselectMyDaemonEdgeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReselectMyDaemonEdgeResponse create() =>
      ReselectMyDaemonEdgeResponse._();
  @$core.override
  ReselectMyDaemonEdgeResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReselectMyDaemonEdgeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReselectMyDaemonEdgeResponse>(create);
  static ReselectMyDaemonEdgeResponse? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonEdgeSelection get selection => $_getN(0);
  @$pb.TagNumber(1)
  set selection(DaemonEdgeSelection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSelection() => $_has(0);
  @$pb.TagNumber(1)
  void clearSelection() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonEdgeSelection ensureSelection() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get reselectAccepted => $_getBF(1);
  @$pb.TagNumber(2)
  set reselectAccepted($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReselectAccepted() => $_has(1);
  @$pb.TagNumber(2)
  void clearReselectAccepted() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
}

class BeginDaemonEnrollmentRequest extends $pb.GeneratedMessage {
  factory BeginDaemonEnrollmentRequest({
    $core.String? enrollmentCode,
    $core.String? deviceId,
    $core.String? deviceFingerprint,
    $core.List<$core.int>? devicePublicKey,
  }) {
    final result = create();
    if (enrollmentCode != null) result.enrollmentCode = enrollmentCode;
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (devicePublicKey != null) result.devicePublicKey = devicePublicKey;
    return result;
  }

  BeginDaemonEnrollmentRequest._();

  factory BeginDaemonEnrollmentRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BeginDaemonEnrollmentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BeginDaemonEnrollmentRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'enrollmentCode')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aOS(3, _omitFieldNames ? '' : 'deviceFingerprint')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'devicePublicKey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BeginDaemonEnrollmentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BeginDaemonEnrollmentRequest copyWith(
          void Function(BeginDaemonEnrollmentRequest) updates) =>
      super.copyWith(
              (message) => updates(message as BeginDaemonEnrollmentRequest))
          as BeginDaemonEnrollmentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BeginDaemonEnrollmentRequest create() =>
      BeginDaemonEnrollmentRequest._();
  @$core.override
  BeginDaemonEnrollmentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BeginDaemonEnrollmentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BeginDaemonEnrollmentRequest>(create);
  static BeginDaemonEnrollmentRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get enrollmentCode => $_getSZ(0);
  @$pb.TagNumber(1)
  set enrollmentCode($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnrollmentCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnrollmentCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceFingerprint() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get devicePublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set devicePublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDevicePublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearDevicePublicKey() => $_clearField(4);
}

class IdentityChallenge extends $pb.GeneratedMessage {
  factory IdentityChallenge({
    $core.String? challengeId,
    $core.List<$core.int>? challenge,
    $0.Timestamp? expiresAt,
  }) {
    final result = create();
    if (challengeId != null) result.challengeId = challengeId;
    if (challenge != null) result.challenge = challenge;
    if (expiresAt != null) result.expiresAt = expiresAt;
    return result;
  }

  IdentityChallenge._();

  factory IdentityChallenge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory IdentityChallenge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IdentityChallenge',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'challengeId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'challenge', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IdentityChallenge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IdentityChallenge copyWith(void Function(IdentityChallenge) updates) =>
      super.copyWith((message) => updates(message as IdentityChallenge))
          as IdentityChallenge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static IdentityChallenge create() => IdentityChallenge._();
  @$core.override
  IdentityChallenge createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static IdentityChallenge getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<IdentityChallenge>(create);
  static IdentityChallenge? _defaultInstance;

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
  $0.Timestamp get expiresAt => $_getN(2);
  @$pb.TagNumber(3)
  set expiresAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasExpiresAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpiresAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureExpiresAt() => $_ensure(2);
}

class DaemonEnrollmentChallenge extends $pb.GeneratedMessage {
  factory DaemonEnrollmentChallenge({
    IdentityChallenge? identityChallenge,
    $core.Iterable<DaemonEdgeCandidate>? edgeCandidates,
  }) {
    final result = create();
    if (identityChallenge != null) result.identityChallenge = identityChallenge;
    if (edgeCandidates != null) result.edgeCandidates.addAll(edgeCandidates);
    return result;
  }

  DaemonEnrollmentChallenge._();

  factory DaemonEnrollmentChallenge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonEnrollmentChallenge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonEnrollmentChallenge',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<IdentityChallenge>(1, _omitFieldNames ? '' : 'identityChallenge',
        subBuilder: IdentityChallenge.create)
    ..pPM<DaemonEdgeCandidate>(2, _omitFieldNames ? '' : 'edgeCandidates',
        subBuilder: DaemonEdgeCandidate.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEnrollmentChallenge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonEnrollmentChallenge copyWith(
          void Function(DaemonEnrollmentChallenge) updates) =>
      super.copyWith((message) => updates(message as DaemonEnrollmentChallenge))
          as DaemonEnrollmentChallenge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonEnrollmentChallenge create() => DaemonEnrollmentChallenge._();
  @$core.override
  DaemonEnrollmentChallenge createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonEnrollmentChallenge getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonEnrollmentChallenge>(create);
  static DaemonEnrollmentChallenge? _defaultInstance;

  @$pb.TagNumber(1)
  IdentityChallenge get identityChallenge => $_getN(0);
  @$pb.TagNumber(1)
  set identityChallenge(IdentityChallenge value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentityChallenge() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentityChallenge() => $_clearField(1);
  @$pb.TagNumber(1)
  IdentityChallenge ensureIdentityChallenge() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<DaemonEdgeCandidate> get edgeCandidates => $_getList(1);
}

class CompleteDaemonEnrollmentRequest extends $pb.GeneratedMessage {
  factory CompleteDaemonEnrollmentRequest({
    $core.String? challengeId,
    $core.List<$core.int>? deviceProof,
    $core.Iterable<DaemonEdgeMeasurement>? edgeMeasurements,
  }) {
    final result = create();
    if (challengeId != null) result.challengeId = challengeId;
    if (deviceProof != null) result.deviceProof = deviceProof;
    if (edgeMeasurements != null)
      result.edgeMeasurements.addAll(edgeMeasurements);
    return result;
  }

  CompleteDaemonEnrollmentRequest._();

  factory CompleteDaemonEnrollmentRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CompleteDaemonEnrollmentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CompleteDaemonEnrollmentRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'challengeId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'deviceProof', $pb.PbFieldType.OY)
    ..pPM<DaemonEdgeMeasurement>(3, _omitFieldNames ? '' : 'edgeMeasurements',
        subBuilder: DaemonEdgeMeasurement.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDaemonEnrollmentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDaemonEnrollmentRequest copyWith(
          void Function(CompleteDaemonEnrollmentRequest) updates) =>
      super.copyWith(
              (message) => updates(message as CompleteDaemonEnrollmentRequest))
          as CompleteDaemonEnrollmentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CompleteDaemonEnrollmentRequest create() =>
      CompleteDaemonEnrollmentRequest._();
  @$core.override
  CompleteDaemonEnrollmentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CompleteDaemonEnrollmentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CompleteDaemonEnrollmentRequest>(
          create);
  static CompleteDaemonEnrollmentRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get challengeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set challengeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChallengeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearChallengeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get deviceProof => $_getN(1);
  @$pb.TagNumber(2)
  set deviceProof($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceProof() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceProof() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<DaemonEdgeMeasurement> get edgeMeasurements => $_getList(2);
}

/// EdgeLocator 是 daemon 和已授权客户端可持久化的公开连接位置，不包含授权秘密。
class EdgeLocator extends $pb.GeneratedMessage {
  factory EdgeLocator({
    $core.String? edgeId,
    $core.String? name,
    $core.String? region,
    $core.String? publicEndpoint,
    $core.String? serverName,
    $core.List<$core.int>? caCertificatePem,
    $fixnum.Int64? revision,
  }) {
    final result = create();
    if (edgeId != null) result.edgeId = edgeId;
    if (name != null) result.name = name;
    if (region != null) result.region = region;
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    if (serverName != null) result.serverName = serverName;
    if (caCertificatePem != null) result.caCertificatePem = caCertificatePem;
    if (revision != null) result.revision = revision;
    return result;
  }

  EdgeLocator._();

  factory EdgeLocator.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeLocator.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeLocator',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'region')
    ..aOS(4, _omitFieldNames ? '' : 'publicEndpoint')
    ..aOS(5, _omitFieldNames ? '' : 'serverName')
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'caCertificatePem', $pb.PbFieldType.OY)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeLocator clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeLocator copyWith(void Function(EdgeLocator) updates) =>
      super.copyWith((message) => updates(message as EdgeLocator))
          as EdgeLocator;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeLocator create() => EdgeLocator._();
  @$core.override
  EdgeLocator createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeLocator getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeLocator>(create);
  static EdgeLocator? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get region => $_getSZ(2);
  @$pb.TagNumber(3)
  set region($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRegion() => $_has(2);
  @$pb.TagNumber(3)
  void clearRegion() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get publicEndpoint => $_getSZ(3);
  @$pb.TagNumber(4)
  set publicEndpoint($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPublicEndpoint() => $_has(3);
  @$pb.TagNumber(4)
  void clearPublicEndpoint() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get serverName => $_getSZ(4);
  @$pb.TagNumber(5)
  set serverName($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasServerName() => $_has(4);
  @$pb.TagNumber(5)
  void clearServerName() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get caCertificatePem => $_getN(5);
  @$pb.TagNumber(6)
  set caCertificatePem($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCaCertificatePem() => $_has(5);
  @$pb.TagNumber(6)
  void clearCaCertificatePem() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get revision => $_getI64(6);
  @$pb.TagNumber(7)
  set revision($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRevision() => $_has(6);
  @$pb.TagNumber(7)
  void clearRevision() => $_clearField(7);
}

class CompleteDaemonEnrollmentResponse extends $pb.GeneratedMessage {
  factory CompleteDaemonEnrollmentResponse({
    DaemonRecord? daemon,
    $1.SignedEnvelope? daemonBinding,
    EdgeLocator? edgeLocator,
    $core.int? daemonCount,
    $core.int? daemonLimit,
    DaemonEdgeSelection? edgeSelection,
  }) {
    final result = create();
    if (daemon != null) result.daemon = daemon;
    if (daemonBinding != null) result.daemonBinding = daemonBinding;
    if (edgeLocator != null) result.edgeLocator = edgeLocator;
    if (daemonCount != null) result.daemonCount = daemonCount;
    if (daemonLimit != null) result.daemonLimit = daemonLimit;
    if (edgeSelection != null) result.edgeSelection = edgeSelection;
    return result;
  }

  CompleteDaemonEnrollmentResponse._();

  factory CompleteDaemonEnrollmentResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CompleteDaemonEnrollmentResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CompleteDaemonEnrollmentResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonRecord>(1, _omitFieldNames ? '' : 'daemon',
        subBuilder: DaemonRecord.create)
    ..aOM<$1.SignedEnvelope>(2, _omitFieldNames ? '' : 'daemonBinding',
        subBuilder: $1.SignedEnvelope.create)
    ..aOM<EdgeLocator>(3, _omitFieldNames ? '' : 'edgeLocator',
        subBuilder: EdgeLocator.create)
    ..aI(4, _omitFieldNames ? '' : 'daemonCount',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(5, _omitFieldNames ? '' : 'daemonLimit',
        fieldType: $pb.PbFieldType.OU3)
    ..aOM<DaemonEdgeSelection>(6, _omitFieldNames ? '' : 'edgeSelection',
        subBuilder: DaemonEdgeSelection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDaemonEnrollmentResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDaemonEnrollmentResponse copyWith(
          void Function(CompleteDaemonEnrollmentResponse) updates) =>
      super.copyWith(
              (message) => updates(message as CompleteDaemonEnrollmentResponse))
          as CompleteDaemonEnrollmentResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CompleteDaemonEnrollmentResponse create() =>
      CompleteDaemonEnrollmentResponse._();
  @$core.override
  CompleteDaemonEnrollmentResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CompleteDaemonEnrollmentResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CompleteDaemonEnrollmentResponse>(
          create);
  static CompleteDaemonEnrollmentResponse? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonRecord get daemon => $_getN(0);
  @$pb.TagNumber(1)
  set daemon(DaemonRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemon() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemon() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonRecord ensureDaemon() => $_ensure(0);

  @$pb.TagNumber(2)
  $1.SignedEnvelope get daemonBinding => $_getN(1);
  @$pb.TagNumber(2)
  set daemonBinding($1.SignedEnvelope value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonBinding() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonBinding() => $_clearField(2);
  @$pb.TagNumber(2)
  $1.SignedEnvelope ensureDaemonBinding() => $_ensure(1);

  @$pb.TagNumber(3)
  EdgeLocator get edgeLocator => $_getN(2);
  @$pb.TagNumber(3)
  set edgeLocator(EdgeLocator value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEdgeLocator() => $_has(2);
  @$pb.TagNumber(3)
  void clearEdgeLocator() => $_clearField(3);
  @$pb.TagNumber(3)
  EdgeLocator ensureEdgeLocator() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.int get daemonCount => $_getIZ(3);
  @$pb.TagNumber(4)
  set daemonCount($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemonCount() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemonCount() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get daemonLimit => $_getIZ(4);
  @$pb.TagNumber(5)
  set daemonLimit($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDaemonLimit() => $_has(4);
  @$pb.TagNumber(5)
  void clearDaemonLimit() => $_clearField(5);

  @$pb.TagNumber(6)
  DaemonEdgeSelection get edgeSelection => $_getN(5);
  @$pb.TagNumber(6)
  set edgeSelection(DaemonEdgeSelection value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasEdgeSelection() => $_has(5);
  @$pb.TagNumber(6)
  void clearEdgeSelection() => $_clearField(6);
  @$pb.TagNumber(6)
  DaemonEdgeSelection ensureEdgeSelection() => $_ensure(5);
}

/// BeginDaemonBindingRefreshRequest 只标识现有 daemon；身份和 Edge 选择都由 Controller 的当前真值决定。
class BeginDaemonBindingRefreshRequest extends $pb.GeneratedMessage {
  factory BeginDaemonBindingRefreshRequest({
    $core.String? daemonId,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    return result;
  }

  BeginDaemonBindingRefreshRequest._();

  factory BeginDaemonBindingRefreshRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BeginDaemonBindingRefreshRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BeginDaemonBindingRefreshRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BeginDaemonBindingRefreshRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BeginDaemonBindingRefreshRequest copyWith(
          void Function(BeginDaemonBindingRefreshRequest) updates) =>
      super.copyWith(
              (message) => updates(message as BeginDaemonBindingRefreshRequest))
          as BeginDaemonBindingRefreshRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BeginDaemonBindingRefreshRequest create() =>
      BeginDaemonBindingRefreshRequest._();
  @$core.override
  BeginDaemonBindingRefreshRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BeginDaemonBindingRefreshRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BeginDaemonBindingRefreshRequest>(
          create);
  static BeginDaemonBindingRefreshRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);
}

class CompleteDaemonBindingRefreshRequest extends $pb.GeneratedMessage {
  factory CompleteDaemonBindingRefreshRequest({
    $core.String? challengeId,
    $core.List<$core.int>? deviceProof,
    $core.Iterable<DaemonEdgeMeasurement>? edgeMeasurements,
    $core.bool? changePreference,
    $core.String? preferredEdgeId,
    $fixnum.Int64? expectedPreferenceRevision,
  }) {
    final result = create();
    if (challengeId != null) result.challengeId = challengeId;
    if (deviceProof != null) result.deviceProof = deviceProof;
    if (edgeMeasurements != null)
      result.edgeMeasurements.addAll(edgeMeasurements);
    if (changePreference != null) result.changePreference = changePreference;
    if (preferredEdgeId != null) result.preferredEdgeId = preferredEdgeId;
    if (expectedPreferenceRevision != null)
      result.expectedPreferenceRevision = expectedPreferenceRevision;
    return result;
  }

  CompleteDaemonBindingRefreshRequest._();

  factory CompleteDaemonBindingRefreshRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CompleteDaemonBindingRefreshRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CompleteDaemonBindingRefreshRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'challengeId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'deviceProof', $pb.PbFieldType.OY)
    ..pPM<DaemonEdgeMeasurement>(3, _omitFieldNames ? '' : 'edgeMeasurements',
        subBuilder: DaemonEdgeMeasurement.create)
    ..aOB(4, _omitFieldNames ? '' : 'changePreference')
    ..aOS(5, _omitFieldNames ? '' : 'preferredEdgeId')
    ..a<$fixnum.Int64>(6, _omitFieldNames ? '' : 'expectedPreferenceRevision',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDaemonBindingRefreshRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDaemonBindingRefreshRequest copyWith(
          void Function(CompleteDaemonBindingRefreshRequest) updates) =>
      super.copyWith((message) =>
              updates(message as CompleteDaemonBindingRefreshRequest))
          as CompleteDaemonBindingRefreshRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CompleteDaemonBindingRefreshRequest create() =>
      CompleteDaemonBindingRefreshRequest._();
  @$core.override
  CompleteDaemonBindingRefreshRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CompleteDaemonBindingRefreshRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<
          CompleteDaemonBindingRefreshRequest>(create);
  static CompleteDaemonBindingRefreshRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get challengeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set challengeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChallengeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearChallengeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get deviceProof => $_getN(1);
  @$pb.TagNumber(2)
  set deviceProof($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceProof() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceProof() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<DaemonEdgeMeasurement> get edgeMeasurements => $_getList(2);

  @$pb.TagNumber(4)
  $core.bool get changePreference => $_getBF(3);
  @$pb.TagNumber(4)
  set changePreference($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasChangePreference() => $_has(3);
  @$pb.TagNumber(4)
  void clearChangePreference() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get preferredEdgeId => $_getSZ(4);
  @$pb.TagNumber(5)
  set preferredEdgeId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPreferredEdgeId() => $_has(4);
  @$pb.TagNumber(5)
  void clearPreferredEdgeId() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get expectedPreferenceRevision => $_getI64(5);
  @$pb.TagNumber(6)
  set expectedPreferenceRevision($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasExpectedPreferenceRevision() => $_has(5);
  @$pb.TagNumber(6)
  void clearExpectedPreferenceRevision() => $_clearField(6);
}

/// RefreshDaemonBindingResponse 对 ACTIVE/BLOCKED daemon 返回新的 binding 和 locator。
/// DELETED 只返回 daemon 状态，使离线 daemon 能清理本地 enrollment。
class RefreshDaemonBindingResponse extends $pb.GeneratedMessage {
  factory RefreshDaemonBindingResponse({
    DaemonRecord? daemon,
    $1.SignedEnvelope? daemonBinding,
    EdgeLocator? edgeLocator,
    DaemonEdgeSelection? edgeSelection,
  }) {
    final result = create();
    if (daemon != null) result.daemon = daemon;
    if (daemonBinding != null) result.daemonBinding = daemonBinding;
    if (edgeLocator != null) result.edgeLocator = edgeLocator;
    if (edgeSelection != null) result.edgeSelection = edgeSelection;
    return result;
  }

  RefreshDaemonBindingResponse._();

  factory RefreshDaemonBindingResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RefreshDaemonBindingResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RefreshDaemonBindingResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<DaemonRecord>(1, _omitFieldNames ? '' : 'daemon',
        subBuilder: DaemonRecord.create)
    ..aOM<$1.SignedEnvelope>(2, _omitFieldNames ? '' : 'daemonBinding',
        subBuilder: $1.SignedEnvelope.create)
    ..aOM<EdgeLocator>(3, _omitFieldNames ? '' : 'edgeLocator',
        subBuilder: EdgeLocator.create)
    ..aOM<DaemonEdgeSelection>(4, _omitFieldNames ? '' : 'edgeSelection',
        subBuilder: DaemonEdgeSelection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RefreshDaemonBindingResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RefreshDaemonBindingResponse copyWith(
          void Function(RefreshDaemonBindingResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RefreshDaemonBindingResponse))
          as RefreshDaemonBindingResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RefreshDaemonBindingResponse create() =>
      RefreshDaemonBindingResponse._();
  @$core.override
  RefreshDaemonBindingResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RefreshDaemonBindingResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RefreshDaemonBindingResponse>(create);
  static RefreshDaemonBindingResponse? _defaultInstance;

  @$pb.TagNumber(1)
  DaemonRecord get daemon => $_getN(0);
  @$pb.TagNumber(1)
  set daemon(DaemonRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemon() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemon() => $_clearField(1);
  @$pb.TagNumber(1)
  DaemonRecord ensureDaemon() => $_ensure(0);

  @$pb.TagNumber(2)
  $1.SignedEnvelope get daemonBinding => $_getN(1);
  @$pb.TagNumber(2)
  set daemonBinding($1.SignedEnvelope value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonBinding() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonBinding() => $_clearField(2);
  @$pb.TagNumber(2)
  $1.SignedEnvelope ensureDaemonBinding() => $_ensure(1);

  @$pb.TagNumber(3)
  EdgeLocator get edgeLocator => $_getN(2);
  @$pb.TagNumber(3)
  set edgeLocator(EdgeLocator value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEdgeLocator() => $_has(2);
  @$pb.TagNumber(3)
  void clearEdgeLocator() => $_clearField(3);
  @$pb.TagNumber(3)
  EdgeLocator ensureEdgeLocator() => $_ensure(2);

  @$pb.TagNumber(4)
  DaemonEdgeSelection get edgeSelection => $_getN(3);
  @$pb.TagNumber(4)
  set edgeSelection(DaemonEdgeSelection value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasEdgeSelection() => $_has(3);
  @$pb.TagNumber(4)
  void clearEdgeSelection() => $_clearField(4);
  @$pb.TagNumber(4)
  DaemonEdgeSelection ensureEdgeSelection() => $_ensure(3);
}

/// EnrollmentService 负责一次性注册，以及现有 DeviceIdentity 在 Edge 位置失效后的 binding 刷新。
class EnrollmentServiceApi {
  final $pb.RpcClient _client;

  EnrollmentServiceApi(this._client);

  $async.Future<DaemonEnrollmentChallenge> beginDaemonEnrollment(
          $pb.ClientContext? ctx, BeginDaemonEnrollmentRequest request) =>
      _client.invoke<DaemonEnrollmentChallenge>(ctx, 'EnrollmentService',
          'BeginDaemonEnrollment', request, DaemonEnrollmentChallenge());
  $async.Future<CompleteDaemonEnrollmentResponse> completeDaemonEnrollment(
          $pb.ClientContext? ctx, CompleteDaemonEnrollmentRequest request) =>
      _client.invoke<CompleteDaemonEnrollmentResponse>(
          ctx,
          'EnrollmentService',
          'CompleteDaemonEnrollment',
          request,
          CompleteDaemonEnrollmentResponse());
  $async.Future<IdentityChallenge> beginDaemonBindingRefresh(
          $pb.ClientContext? ctx, BeginDaemonBindingRefreshRequest request) =>
      _client.invoke<IdentityChallenge>(ctx, 'EnrollmentService',
          'BeginDaemonBindingRefresh', request, IdentityChallenge());
  $async.Future<RefreshDaemonBindingResponse> completeDaemonBindingRefresh(
          $pb.ClientContext? ctx,
          CompleteDaemonBindingRefreshRequest request) =>
      _client.invoke<RefreshDaemonBindingResponse>(
          ctx,
          'EnrollmentService',
          'CompleteDaemonBindingRefresh',
          request,
          RefreshDaemonBindingResponse());
}

/// DaemonManagementService 是登录用户管理自己 daemon 的 API；实现必须从 session 推导账号。
class DaemonManagementServiceApi {
  final $pb.RpcClient _client;

  DaemonManagementServiceApi(this._client);

  $async.Future<CreateDaemonEnrollmentResponse> createMyEnrollment(
          $pb.ClientContext? ctx, CreateMyDaemonEnrollmentRequest request) =>
      _client.invoke<CreateDaemonEnrollmentResponse>(
          ctx,
          'DaemonManagementService',
          'CreateMyEnrollment',
          request,
          CreateDaemonEnrollmentResponse());
  $async.Future<ListMyDaemonsResponse> listMyDaemons(
          $pb.ClientContext? ctx, ListMyDaemonsRequest request) =>
      _client.invoke<ListMyDaemonsResponse>(ctx, 'DaemonManagementService',
          'ListMyDaemons', request, ListMyDaemonsResponse());
  $async.Future<ChangeMyDaemonStateResponse> changeMyDaemonState(
          $pb.ClientContext? ctx, ChangeMyDaemonStateRequest request) =>
      _client.invoke<ChangeMyDaemonStateResponse>(
          ctx,
          'DaemonManagementService',
          'ChangeMyDaemonState',
          request,
          ChangeMyDaemonStateResponse());
  $async.Future<ListMyDaemonEdgesResponse> listMyDaemonEdges(
          $pb.ClientContext? ctx, ListMyDaemonEdgesRequest request) =>
      _client.invoke<ListMyDaemonEdgesResponse>(ctx, 'DaemonManagementService',
          'ListMyDaemonEdges', request, ListMyDaemonEdgesResponse());
  $async.Future<ChangeMyDaemonEdgePreferenceResponse>
      changeMyDaemonEdgePreference($pb.ClientContext? ctx,
              ChangeMyDaemonEdgePreferenceRequest request) =>
          _client.invoke<ChangeMyDaemonEdgePreferenceResponse>(
              ctx,
              'DaemonManagementService',
              'ChangeMyDaemonEdgePreference',
              request,
              ChangeMyDaemonEdgePreferenceResponse());
  $async.Future<ReselectMyDaemonEdgeResponse> reselectMyDaemonEdge(
          $pb.ClientContext? ctx, ReselectMyDaemonEdgeRequest request) =>
      _client.invoke<ReselectMyDaemonEdgeResponse>(
          ctx,
          'DaemonManagementService',
          'ReselectMyDaemonEdge',
          request,
          ReselectMyDaemonEdgeResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
