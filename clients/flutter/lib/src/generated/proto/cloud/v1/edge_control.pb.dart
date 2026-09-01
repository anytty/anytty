// This is a generated file - do not edit.
//
// Generated from cloud/v1/edge_control.proto.

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
    as $3;

import 'certificate.pb.dart' as $0;
import 'common.pb.dart' as $1;
import 'edge_config.pb.dart' as $6;
import 'edge_control.pbenum.dart';
import 'enrollment.pb.dart' as $4;
import 'runtime.pb.dart' as $2;
import 'usage.pb.dart' as $5;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'edge_control.pbenum.dart';

/// EdgeHello 是 EdgeControl 新连接的第一个且唯一一个注册消息。
/// edge_id 必须与 mTLS 客户端证书中的 Edge URI SAN 一致。
class EdgeHello extends $pb.GeneratedMessage {
  factory EdgeHello({
    $core.String? edgeId,
    $core.String? softwareVersion,
    $core.Iterable<EdgeCapability>? capabilities,
    $fixnum.Int64? desiredConfigVersion,
    $0.EdgePublicCertificateStatus? publicCertificate,
  }) {
    final result = create();
    if (edgeId != null) result.edgeId = edgeId;
    if (softwareVersion != null) result.softwareVersion = softwareVersion;
    if (capabilities != null) result.capabilities.addAll(capabilities);
    if (desiredConfigVersion != null)
      result.desiredConfigVersion = desiredConfigVersion;
    if (publicCertificate != null) result.publicCertificate = publicCertificate;
    return result;
  }

  EdgeHello._();

  factory EdgeHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeHello',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..aOS(2, _omitFieldNames ? '' : 'softwareVersion')
    ..pc<EdgeCapability>(
        3, _omitFieldNames ? '' : 'capabilities', $pb.PbFieldType.KE,
        valueOf: EdgeCapability.valueOf,
        enumValues: EdgeCapability.values,
        defaultEnumValue: EdgeCapability.EDGE_CAPABILITY_UNSPECIFIED)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'desiredConfigVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.EdgePublicCertificateStatus>(
        5, _omitFieldNames ? '' : 'publicCertificate',
        subBuilder: $0.EdgePublicCertificateStatus.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeHello copyWith(void Function(EdgeHello) updates) =>
      super.copyWith((message) => updates(message as EdgeHello)) as EdgeHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeHello create() => EdgeHello._();
  @$core.override
  EdgeHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeHello getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EdgeHello>(create);
  static EdgeHello? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get softwareVersion => $_getSZ(1);
  @$pb.TagNumber(2)
  set softwareVersion($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSoftwareVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearSoftwareVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<EdgeCapability> get capabilities => $_getList(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get desiredConfigVersion => $_getI64(3);
  @$pb.TagNumber(4)
  set desiredConfigVersion($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDesiredConfigVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearDesiredConfigVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.EdgePublicCertificateStatus get publicCertificate => $_getN(4);
  @$pb.TagNumber(5)
  set publicCertificate($0.EdgePublicCertificateStatus value) =>
      $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasPublicCertificate() => $_has(4);
  @$pb.TagNumber(5)
  void clearPublicCertificate() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.EdgePublicCertificateStatus ensurePublicCertificate() => $_ensure(4);
}

/// EdgeWelcome 表示 Controller 已接受当前 mTLS 连接和协议版本。
/// connection_id 由 Edge 每次重连生成，Controller 接受后在外层 ControllerCommand 中回显。
class EdgeWelcome extends $pb.GeneratedMessage {
  factory EdgeWelcome({
    $core.int? acceptedProtocolVersion,
    $1.HeartbeatPolicy? heartbeat,
    $1.KeyBundle? bindingKeyBundle,
  }) {
    final result = create();
    if (acceptedProtocolVersion != null)
      result.acceptedProtocolVersion = acceptedProtocolVersion;
    if (heartbeat != null) result.heartbeat = heartbeat;
    if (bindingKeyBundle != null) result.bindingKeyBundle = bindingKeyBundle;
    return result;
  }

  EdgeWelcome._();

  factory EdgeWelcome.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeWelcome.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeWelcome',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'acceptedProtocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOM<$1.HeartbeatPolicy>(2, _omitFieldNames ? '' : 'heartbeat',
        subBuilder: $1.HeartbeatPolicy.create)
    ..aOM<$1.KeyBundle>(3, _omitFieldNames ? '' : 'bindingKeyBundle',
        subBuilder: $1.KeyBundle.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeWelcome clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeWelcome copyWith(void Function(EdgeWelcome) updates) =>
      super.copyWith((message) => updates(message as EdgeWelcome))
          as EdgeWelcome;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeWelcome create() => EdgeWelcome._();
  @$core.override
  EdgeWelcome createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeWelcome getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeWelcome>(create);
  static EdgeWelcome? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get acceptedProtocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set acceptedProtocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAcceptedProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearAcceptedProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $1.HeartbeatPolicy get heartbeat => $_getN(1);
  @$pb.TagNumber(2)
  set heartbeat($1.HeartbeatPolicy value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasHeartbeat() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeartbeat() => $_clearField(2);
  @$pb.TagNumber(2)
  $1.HeartbeatPolicy ensureHeartbeat() => $_ensure(1);

  @$pb.TagNumber(3)
  $1.KeyBundle get bindingKeyBundle => $_getN(2);
  @$pb.TagNumber(3)
  set bindingKeyBundle($1.KeyBundle value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasBindingKeyBundle() => $_has(2);
  @$pb.TagNumber(3)
  void clearBindingKeyBundle() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.KeyBundle ensureBindingKeyBundle() => $_ensure(2);
}

/// SnapshotBegin 开始当前 connection generation 的临时快照命名空间。
class SnapshotBegin extends $pb.GeneratedMessage {
  factory SnapshotBegin({
    $core.String? snapshotId,
    $fixnum.Int64? revision,
  }) {
    final result = create();
    if (snapshotId != null) result.snapshotId = snapshotId;
    if (revision != null) result.revision = revision;
    return result;
  }

  SnapshotBegin._();

  factory SnapshotBegin.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SnapshotBegin.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SnapshotBegin',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'snapshotId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotBegin clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotBegin copyWith(void Function(SnapshotBegin) updates) =>
      super.copyWith((message) => updates(message as SnapshotBegin))
          as SnapshotBegin;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SnapshotBegin create() => SnapshotBegin._();
  @$core.override
  SnapshotBegin createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SnapshotBegin getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SnapshotBegin>(create);
  static SnapshotBegin? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get snapshotId => $_getSZ(0);
  @$pb.TagNumber(1)
  set snapshotId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSnapshotId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSnapshotId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get revision => $_getI64(1);
  @$pb.TagNumber(2)
  set revision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);
}

/// SnapshotChunk 分块传输快照对象；chunk_index 必须从零严格递增。
class SnapshotChunk extends $pb.GeneratedMessage {
  factory SnapshotChunk({
    $core.String? snapshotId,
    $core.int? chunkIndex,
    $core.Iterable<$2.AgentPresence>? agents,
    $core.Iterable<$2.ClientSessionSummary>? sessions,
  }) {
    final result = create();
    if (snapshotId != null) result.snapshotId = snapshotId;
    if (chunkIndex != null) result.chunkIndex = chunkIndex;
    if (agents != null) result.agents.addAll(agents);
    if (sessions != null) result.sessions.addAll(sessions);
    return result;
  }

  SnapshotChunk._();

  factory SnapshotChunk.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SnapshotChunk.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SnapshotChunk',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'snapshotId')
    ..aI(2, _omitFieldNames ? '' : 'chunkIndex', fieldType: $pb.PbFieldType.OU3)
    ..pPM<$2.AgentPresence>(3, _omitFieldNames ? '' : 'agents',
        subBuilder: $2.AgentPresence.create)
    ..pPM<$2.ClientSessionSummary>(4, _omitFieldNames ? '' : 'sessions',
        subBuilder: $2.ClientSessionSummary.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotChunk clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotChunk copyWith(void Function(SnapshotChunk) updates) =>
      super.copyWith((message) => updates(message as SnapshotChunk))
          as SnapshotChunk;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SnapshotChunk create() => SnapshotChunk._();
  @$core.override
  SnapshotChunk createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SnapshotChunk getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SnapshotChunk>(create);
  static SnapshotChunk? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get snapshotId => $_getSZ(0);
  @$pb.TagNumber(1)
  set snapshotId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSnapshotId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSnapshotId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get chunkIndex => $_getIZ(1);
  @$pb.TagNumber(2)
  set chunkIndex($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChunkIndex() => $_has(1);
  @$pb.TagNumber(2)
  void clearChunkIndex() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$2.AgentPresence> get agents => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbList<$2.ClientSessionSummary> get sessions => $_getList(3);
}

/// SnapshotEnd 携带完整投影的确定性摘要，Controller 校验后才原子发布。
class SnapshotEnd extends $pb.GeneratedMessage {
  factory SnapshotEnd({
    $core.String? snapshotId,
    $fixnum.Int64? revision,
    $core.int? chunkCount,
    $core.List<$core.int>? digest,
  }) {
    final result = create();
    if (snapshotId != null) result.snapshotId = snapshotId;
    if (revision != null) result.revision = revision;
    if (chunkCount != null) result.chunkCount = chunkCount;
    if (digest != null) result.digest = digest;
    return result;
  }

  SnapshotEnd._();

  factory SnapshotEnd.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SnapshotEnd.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SnapshotEnd',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'snapshotId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(3, _omitFieldNames ? '' : 'chunkCount', fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'digest', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotEnd clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotEnd copyWith(void Function(SnapshotEnd) updates) =>
      super.copyWith((message) => updates(message as SnapshotEnd))
          as SnapshotEnd;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SnapshotEnd create() => SnapshotEnd._();
  @$core.override
  SnapshotEnd createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SnapshotEnd getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SnapshotEnd>(create);
  static SnapshotEnd? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get snapshotId => $_getSZ(0);
  @$pb.TagNumber(1)
  set snapshotId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSnapshotId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSnapshotId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get revision => $_getI64(1);
  @$pb.TagNumber(2)
  set revision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get chunkCount => $_getIZ(2);
  @$pb.TagNumber(3)
  set chunkCount($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasChunkCount() => $_has(2);
  @$pb.TagNumber(3)
  void clearChunkCount() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get digest => $_getN(3);
  @$pb.TagNumber(4)
  set digest($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDigest() => $_has(3);
  @$pb.TagNumber(4)
  void clearDigest() => $_clearField(4);
}

/// EdgeHeartbeat 只报告可合并的最新 runtime revision。
class EdgeHeartbeat extends $pb.GeneratedMessage {
  factory EdgeHeartbeat({
    $fixnum.Int64? runtimeRevision,
  }) {
    final result = create();
    if (runtimeRevision != null) result.runtimeRevision = runtimeRevision;
    return result;
  }

  EdgeHeartbeat._();

  factory EdgeHeartbeat.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeHeartbeat.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeHeartbeat',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'runtimeRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeHeartbeat clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeHeartbeat copyWith(void Function(EdgeHeartbeat) updates) =>
      super.copyWith((message) => updates(message as EdgeHeartbeat))
          as EdgeHeartbeat;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeHeartbeat create() => EdgeHeartbeat._();
  @$core.override
  EdgeHeartbeat createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeHeartbeat getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeHeartbeat>(create);
  static EdgeHeartbeat? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get runtimeRevision => $_getI64(0);
  @$pb.TagNumber(1)
  set runtimeRevision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRuntimeRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearRuntimeRevision() => $_clearField(1);
}

/// SnapshotAccepted 表示 Controller 已原子发布当前快照。
class SnapshotAccepted extends $pb.GeneratedMessage {
  factory SnapshotAccepted({
    $core.String? snapshotId,
    $fixnum.Int64? revision,
  }) {
    final result = create();
    if (snapshotId != null) result.snapshotId = snapshotId;
    if (revision != null) result.revision = revision;
    return result;
  }

  SnapshotAccepted._();

  factory SnapshotAccepted.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SnapshotAccepted.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SnapshotAccepted',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'snapshotId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotAccepted clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SnapshotAccepted copyWith(void Function(SnapshotAccepted) updates) =>
      super.copyWith((message) => updates(message as SnapshotAccepted))
          as SnapshotAccepted;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SnapshotAccepted create() => SnapshotAccepted._();
  @$core.override
  SnapshotAccepted createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SnapshotAccepted getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SnapshotAccepted>(create);
  static SnapshotAccepted? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get snapshotId => $_getSZ(0);
  @$pb.TagNumber(1)
  set snapshotId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSnapshotId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSnapshotId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get revision => $_getI64(1);
  @$pb.TagNumber(2)
  set revision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);
}

/// ResyncRequired 要求 Edge 丢弃当前发送游标并重新获取一致性快照。
class ResyncRequired extends $pb.GeneratedMessage {
  factory ResyncRequired({
    $fixnum.Int64? expectedRevision,
    $core.String? reason,
  }) {
    final result = create();
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (reason != null) result.reason = reason;
    return result;
  }

  ResyncRequired._();

  factory ResyncRequired.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResyncRequired.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResyncRequired',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(2, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResyncRequired clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResyncRequired copyWith(void Function(ResyncRequired) updates) =>
      super.copyWith((message) => updates(message as ResyncRequired))
          as ResyncRequired;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResyncRequired create() => ResyncRequired._();
  @$core.override
  ResyncRequired createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResyncRequired getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResyncRequired>(create);
  static ResyncRequired? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get expectedRevision => $_getI64(0);
  @$pb.TagNumber(1)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExpectedRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearExpectedRevision() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get reason => $_getSZ(1);
  @$pb.TagNumber(2)
  set reason($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReason() => $_has(1);
  @$pb.TagNumber(2)
  void clearReason() => $_clearField(2);
}

/// ConfigApplied 是 Edge 验签、校验并原子缓存 desired config 后的结果。
class ConfigApplied extends $pb.GeneratedMessage {
  factory ConfigApplied({
    $fixnum.Int64? version,
    $core.bool? applied,
    $core.String? errorCode,
  }) {
    final result = create();
    if (version != null) result.version = version;
    if (applied != null) result.applied = applied;
    if (errorCode != null) result.errorCode = errorCode;
    return result;
  }

  ConfigApplied._();

  factory ConfigApplied.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConfigApplied.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConfigApplied',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(2, _omitFieldNames ? '' : 'applied')
    ..aOS(3, _omitFieldNames ? '' : 'errorCode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConfigApplied clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConfigApplied copyWith(void Function(ConfigApplied) updates) =>
      super.copyWith((message) => updates(message as ConfigApplied))
          as ConfigApplied;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConfigApplied create() => ConfigApplied._();
  @$core.override
  ConfigApplied createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConfigApplied getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConfigApplied>(create);
  static ConfigApplied? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get version => $_getI64(0);
  @$pb.TagNumber(1)
  set version($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get applied => $_getBF(1);
  @$pb.TagNumber(2)
  set applied($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasApplied() => $_has(1);
  @$pb.TagNumber(2)
  void clearApplied() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get errorCode => $_getSZ(2);
  @$pb.TagNumber(3)
  set errorCode($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasErrorCode() => $_has(2);
  @$pb.TagNumber(3)
  void clearErrorCode() => $_clearField(3);
}

/// EdgeIdentityRenewRequest 只允许在已通过当前 EdgeIdentity mTLS 认证的控制流内发送。
/// CSR 必须绑定同一 Edge URI SAN；新私钥始终留在 Edge 本机。
class EdgeIdentityRenewRequest extends $pb.GeneratedMessage {
  factory EdgeIdentityRenewRequest({
    $core.String? requestId,
    $core.List<$core.int>? csrPem,
    $core.List<$core.int>? currentCertificateSha256,
    $3.Timestamp? requestedAt,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (csrPem != null) result.csrPem = csrPem;
    if (currentCertificateSha256 != null)
      result.currentCertificateSha256 = currentCertificateSha256;
    if (requestedAt != null) result.requestedAt = requestedAt;
    return result;
  }

  EdgeIdentityRenewRequest._();

  factory EdgeIdentityRenewRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeIdentityRenewRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeIdentityRenewRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'csrPem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(3,
        _omitFieldNames ? '' : 'currentCertificateSha256', $pb.PbFieldType.OY)
    ..aOM<$3.Timestamp>(4, _omitFieldNames ? '' : 'requestedAt',
        subBuilder: $3.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeIdentityRenewRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeIdentityRenewRequest copyWith(
          void Function(EdgeIdentityRenewRequest) updates) =>
      super.copyWith((message) => updates(message as EdgeIdentityRenewRequest))
          as EdgeIdentityRenewRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeIdentityRenewRequest create() => EdgeIdentityRenewRequest._();
  @$core.override
  EdgeIdentityRenewRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeIdentityRenewRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeIdentityRenewRequest>(create);
  static EdgeIdentityRenewRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get csrPem => $_getN(1);
  @$pb.TagNumber(2)
  set csrPem($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCsrPem() => $_has(1);
  @$pb.TagNumber(2)
  void clearCsrPem() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get currentCertificateSha256 => $_getN(2);
  @$pb.TagNumber(3)
  set currentCertificateSha256($core.List<$core.int> value) =>
      $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCurrentCertificateSha256() => $_has(2);
  @$pb.TagNumber(3)
  void clearCurrentCertificateSha256() => $_clearField(3);

  @$pb.TagNumber(4)
  $3.Timestamp get requestedAt => $_getN(3);
  @$pb.TagNumber(4)
  set requestedAt($3.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRequestedAt() => $_has(3);
  @$pb.TagNumber(4)
  void clearRequestedAt() => $_clearField(4);
  @$pb.TagNumber(4)
  $3.Timestamp ensureRequestedAt() => $_ensure(3);
}

/// EdgeIdentityRenewResponse 返回 Controller Edge CA 签发的新 clientAuth 叶证书。
class EdgeIdentityRenewResponse extends $pb.GeneratedMessage {
  factory EdgeIdentityRenewResponse({
    $core.String? requestId,
    $core.List<$core.int>? certificatePem,
    $core.List<$core.int>? certificateSha256,
    $3.Timestamp? notAfter,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (certificatePem != null) result.certificatePem = certificatePem;
    if (certificateSha256 != null) result.certificateSha256 = certificateSha256;
    if (notAfter != null) result.notAfter = notAfter;
    return result;
  }

  EdgeIdentityRenewResponse._();

  factory EdgeIdentityRenewResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeIdentityRenewResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeIdentityRenewResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'certificatePem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'certificateSha256', $pb.PbFieldType.OY)
    ..aOM<$3.Timestamp>(4, _omitFieldNames ? '' : 'notAfter',
        subBuilder: $3.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeIdentityRenewResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeIdentityRenewResponse copyWith(
          void Function(EdgeIdentityRenewResponse) updates) =>
      super.copyWith((message) => updates(message as EdgeIdentityRenewResponse))
          as EdgeIdentityRenewResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeIdentityRenewResponse create() => EdgeIdentityRenewResponse._();
  @$core.override
  EdgeIdentityRenewResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeIdentityRenewResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeIdentityRenewResponse>(create);
  static EdgeIdentityRenewResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get certificatePem => $_getN(1);
  @$pb.TagNumber(2)
  set certificatePem($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCertificatePem() => $_has(1);
  @$pb.TagNumber(2)
  void clearCertificatePem() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get certificateSha256 => $_getN(2);
  @$pb.TagNumber(3)
  set certificateSha256($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCertificateSha256() => $_has(2);
  @$pb.TagNumber(3)
  void clearCertificateSha256() => $_clearField(3);

  @$pb.TagNumber(4)
  $3.Timestamp get notAfter => $_getN(3);
  @$pb.TagNumber(4)
  set notAfter($3.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasNotAfter() => $_has(3);
  @$pb.TagNumber(4)
  void clearNotAfter() => $_clearField(4);
  @$pb.TagNumber(4)
  $3.Timestamp ensureNotAfter() => $_ensure(3);
}

/// EdgeIdentityApplied 是 Edge 完成证书/私钥原子持久化和内存热切换后的回执。
class EdgeIdentityApplied extends $pb.GeneratedMessage {
  factory EdgeIdentityApplied({
    $core.String? requestId,
    $core.List<$core.int>? certificateSha256,
    $3.Timestamp? notAfter,
    $core.bool? applied,
    $core.String? errorCode,
    $core.String? errorMessage,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (certificateSha256 != null) result.certificateSha256 = certificateSha256;
    if (notAfter != null) result.notAfter = notAfter;
    if (applied != null) result.applied = applied;
    if (errorCode != null) result.errorCode = errorCode;
    if (errorMessage != null) result.errorMessage = errorMessage;
    return result;
  }

  EdgeIdentityApplied._();

  factory EdgeIdentityApplied.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeIdentityApplied.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeIdentityApplied',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'certificateSha256', $pb.PbFieldType.OY)
    ..aOM<$3.Timestamp>(3, _omitFieldNames ? '' : 'notAfter',
        subBuilder: $3.Timestamp.create)
    ..aOB(4, _omitFieldNames ? '' : 'applied')
    ..aOS(5, _omitFieldNames ? '' : 'errorCode')
    ..aOS(6, _omitFieldNames ? '' : 'errorMessage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeIdentityApplied clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeIdentityApplied copyWith(void Function(EdgeIdentityApplied) updates) =>
      super.copyWith((message) => updates(message as EdgeIdentityApplied))
          as EdgeIdentityApplied;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeIdentityApplied create() => EdgeIdentityApplied._();
  @$core.override
  EdgeIdentityApplied createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeIdentityApplied getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeIdentityApplied>(create);
  static EdgeIdentityApplied? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get certificateSha256 => $_getN(1);
  @$pb.TagNumber(2)
  set certificateSha256($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCertificateSha256() => $_has(1);
  @$pb.TagNumber(2)
  void clearCertificateSha256() => $_clearField(2);

  @$pb.TagNumber(3)
  $3.Timestamp get notAfter => $_getN(2);
  @$pb.TagNumber(3)
  set notAfter($3.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasNotAfter() => $_has(2);
  @$pb.TagNumber(3)
  void clearNotAfter() => $_clearField(3);
  @$pb.TagNumber(3)
  $3.Timestamp ensureNotAfter() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.bool get applied => $_getBF(3);
  @$pb.TagNumber(4)
  set applied($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasApplied() => $_has(3);
  @$pb.TagNumber(4)
  void clearApplied() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get errorCode => $_getSZ(4);
  @$pb.TagNumber(5)
  set errorCode($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasErrorCode() => $_has(4);
  @$pb.TagNumber(5)
  void clearErrorCode() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get errorMessage => $_getSZ(5);
  @$pb.TagNumber(6)
  set errorMessage($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasErrorMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearErrorMessage() => $_clearField(6);
}

/// DaemonStateSnapshot 在 EdgeControl 建连时原子替换 Edge 的内存状态表。
class DaemonStateSnapshot extends $pb.GeneratedMessage {
  factory DaemonStateSnapshot({
    $core.Iterable<$4.DaemonStateRecord>? daemons,
  }) {
    final result = create();
    if (daemons != null) result.daemons.addAll(daemons);
    return result;
  }

  DaemonStateSnapshot._();

  factory DaemonStateSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateSnapshot',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<$4.DaemonStateRecord>(1, _omitFieldNames ? '' : 'daemons',
        subBuilder: $4.DaemonStateRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSnapshot copyWith(void Function(DaemonStateSnapshot) updates) =>
      super.copyWith((message) => updates(message as DaemonStateSnapshot))
          as DaemonStateSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateSnapshot create() => DaemonStateSnapshot._();
  @$core.override
  DaemonStateSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateSnapshot>(create);
  static DaemonStateSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$4.DaemonStateRecord> get daemons => $_getList(0);
}

/// DaemonStateSyncRequest reports only daemon connections currently owned by
/// this Edge generation. Controller returns the durable records in bounded
/// chunks so EdgeControl reconnect cost does not grow with the global fleet.
class DaemonStateSyncRequest extends $pb.GeneratedMessage {
  factory DaemonStateSyncRequest({
    $core.String? syncId,
    $core.Iterable<$core.String>? daemonIds,
  }) {
    final result = create();
    if (syncId != null) result.syncId = syncId;
    if (daemonIds != null) result.daemonIds.addAll(daemonIds);
    return result;
  }

  DaemonStateSyncRequest._();

  factory DaemonStateSyncRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateSyncRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateSyncRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'syncId')
    ..pPS(2, _omitFieldNames ? '' : 'daemonIds')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSyncRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSyncRequest copyWith(
          void Function(DaemonStateSyncRequest) updates) =>
      super.copyWith((message) => updates(message as DaemonStateSyncRequest))
          as DaemonStateSyncRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateSyncRequest create() => DaemonStateSyncRequest._();
  @$core.override
  DaemonStateSyncRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateSyncRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateSyncRequest>(create);
  static DaemonStateSyncRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get syncId => $_getSZ(0);
  @$pb.TagNumber(1)
  set syncId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSyncId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSyncId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get daemonIds => $_getList(1);
}

class DaemonStateSyncChunk extends $pb.GeneratedMessage {
  factory DaemonStateSyncChunk({
    $core.String? syncId,
    $core.int? chunkIndex,
    $core.Iterable<$4.DaemonStateRecord>? daemons,
  }) {
    final result = create();
    if (syncId != null) result.syncId = syncId;
    if (chunkIndex != null) result.chunkIndex = chunkIndex;
    if (daemons != null) result.daemons.addAll(daemons);
    return result;
  }

  DaemonStateSyncChunk._();

  factory DaemonStateSyncChunk.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateSyncChunk.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateSyncChunk',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'syncId')
    ..aI(2, _omitFieldNames ? '' : 'chunkIndex', fieldType: $pb.PbFieldType.OU3)
    ..pPM<$4.DaemonStateRecord>(3, _omitFieldNames ? '' : 'daemons',
        subBuilder: $4.DaemonStateRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSyncChunk clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSyncChunk copyWith(void Function(DaemonStateSyncChunk) updates) =>
      super.copyWith((message) => updates(message as DaemonStateSyncChunk))
          as DaemonStateSyncChunk;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateSyncChunk create() => DaemonStateSyncChunk._();
  @$core.override
  DaemonStateSyncChunk createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateSyncChunk getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateSyncChunk>(create);
  static DaemonStateSyncChunk? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get syncId => $_getSZ(0);
  @$pb.TagNumber(1)
  set syncId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSyncId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSyncId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get chunkIndex => $_getIZ(1);
  @$pb.TagNumber(2)
  set chunkIndex($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChunkIndex() => $_has(1);
  @$pb.TagNumber(2)
  void clearChunkIndex() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$4.DaemonStateRecord> get daemons => $_getList(2);
}

class DaemonStateSyncEnd extends $pb.GeneratedMessage {
  factory DaemonStateSyncEnd({
    $core.String? syncId,
    $core.int? chunkCount,
  }) {
    final result = create();
    if (syncId != null) result.syncId = syncId;
    if (chunkCount != null) result.chunkCount = chunkCount;
    return result;
  }

  DaemonStateSyncEnd._();

  factory DaemonStateSyncEnd.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateSyncEnd.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateSyncEnd',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'syncId')
    ..aI(2, _omitFieldNames ? '' : 'chunkCount', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSyncEnd clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateSyncEnd copyWith(void Function(DaemonStateSyncEnd) updates) =>
      super.copyWith((message) => updates(message as DaemonStateSyncEnd))
          as DaemonStateSyncEnd;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateSyncEnd create() => DaemonStateSyncEnd._();
  @$core.override
  DaemonStateSyncEnd createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateSyncEnd getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateSyncEnd>(create);
  static DaemonStateSyncEnd? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get syncId => $_getSZ(0);
  @$pb.TagNumber(1)
  set syncId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSyncId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSyncId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get chunkCount => $_getIZ(1);
  @$pb.TagNumber(2)
  set chunkCount($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChunkCount() => $_has(1);
  @$pb.TagNumber(2)
  void clearChunkCount() => $_clearField(2);
}

class DaemonStateDelta extends $pb.GeneratedMessage {
  factory DaemonStateDelta({
    $4.DaemonStateRecord? daemon,
  }) {
    final result = create();
    if (daemon != null) result.daemon = daemon;
    return result;
  }

  DaemonStateDelta._();

  factory DaemonStateDelta.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateDelta.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateDelta',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$4.DaemonStateRecord>(1, _omitFieldNames ? '' : 'daemon',
        subBuilder: $4.DaemonStateRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateDelta clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateDelta copyWith(void Function(DaemonStateDelta) updates) =>
      super.copyWith((message) => updates(message as DaemonStateDelta))
          as DaemonStateDelta;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateDelta create() => DaemonStateDelta._();
  @$core.override
  DaemonStateDelta createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateDelta getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateDelta>(create);
  static DaemonStateDelta? _defaultInstance;

  @$pb.TagNumber(1)
  $4.DaemonStateRecord get daemon => $_getN(0);
  @$pb.TagNumber(1)
  set daemon($4.DaemonStateRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemon() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemon() => $_clearField(1);
  @$pb.TagNumber(1)
  $4.DaemonStateRecord ensureDaemon() => $_ensure(0);
}

class DaemonStateQuery extends $pb.GeneratedMessage {
  factory DaemonStateQuery({
    $core.String? requestId,
    $core.String? daemonId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (daemonId != null) result.daemonId = daemonId;
    return result;
  }

  DaemonStateQuery._();

  factory DaemonStateQuery.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateQuery.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateQuery',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'daemonId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateQuery clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateQuery copyWith(void Function(DaemonStateQuery) updates) =>
      super.copyWith((message) => updates(message as DaemonStateQuery))
          as DaemonStateQuery;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateQuery create() => DaemonStateQuery._();
  @$core.override
  DaemonStateQuery createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateQuery getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateQuery>(create);
  static DaemonStateQuery? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get daemonId => $_getSZ(1);
  @$pb.TagNumber(2)
  set daemonId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonId() => $_clearField(2);
}

class DaemonStateQueryResult extends $pb.GeneratedMessage {
  factory DaemonStateQueryResult({
    $core.String? requestId,
    $core.String? daemonId,
    $core.bool? found,
    $4.DaemonStateRecord? daemon,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (daemonId != null) result.daemonId = daemonId;
    if (found != null) result.found = found;
    if (daemon != null) result.daemon = daemon;
    return result;
  }

  DaemonStateQueryResult._();

  factory DaemonStateQueryResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonStateQueryResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonStateQueryResult',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'daemonId')
    ..aOB(3, _omitFieldNames ? '' : 'found')
    ..aOM<$4.DaemonStateRecord>(4, _omitFieldNames ? '' : 'daemon',
        subBuilder: $4.DaemonStateRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateQueryResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonStateQueryResult copyWith(
          void Function(DaemonStateQueryResult) updates) =>
      super.copyWith((message) => updates(message as DaemonStateQueryResult))
          as DaemonStateQueryResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonStateQueryResult create() => DaemonStateQueryResult._();
  @$core.override
  DaemonStateQueryResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonStateQueryResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonStateQueryResult>(create);
  static DaemonStateQueryResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get daemonId => $_getSZ(1);
  @$pb.TagNumber(2)
  set daemonId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDaemonId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDaemonId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get found => $_getBF(2);
  @$pb.TagNumber(3)
  set found($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFound() => $_has(2);
  @$pb.TagNumber(3)
  void clearFound() => $_clearField(3);

  @$pb.TagNumber(4)
  $4.DaemonStateRecord get daemon => $_getN(3);
  @$pb.TagNumber(4)
  set daemon($4.DaemonStateRecord value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemon() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemon() => $_clearField(4);
  @$pb.TagNumber(4)
  $4.DaemonStateRecord ensureDaemon() => $_ensure(3);
}

/// DaemonConnectionAdmissionRequest 在 Presence 发布前为一个认证连接申请全局账号名额。
/// release=true 释放尚未被 Presence 增量消费的准入；同一 daemon 的重连不重复计数。
class DaemonConnectionAdmissionRequest extends $pb.GeneratedMessage {
  factory DaemonConnectionAdmissionRequest({
    $core.String? requestId,
    $core.String? admissionId,
    $core.String? daemonId,
    $core.String? accountId,
    $core.String? agentConnectionId,
    $core.bool? release,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (admissionId != null) result.admissionId = admissionId;
    if (daemonId != null) result.daemonId = daemonId;
    if (accountId != null) result.accountId = accountId;
    if (agentConnectionId != null) result.agentConnectionId = agentConnectionId;
    if (release != null) result.release = release;
    return result;
  }

  DaemonConnectionAdmissionRequest._();

  factory DaemonConnectionAdmissionRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonConnectionAdmissionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonConnectionAdmissionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'admissionId')
    ..aOS(3, _omitFieldNames ? '' : 'daemonId')
    ..aOS(4, _omitFieldNames ? '' : 'accountId')
    ..aOS(5, _omitFieldNames ? '' : 'agentConnectionId')
    ..aOB(6, _omitFieldNames ? '' : 'release')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonConnectionAdmissionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonConnectionAdmissionRequest copyWith(
          void Function(DaemonConnectionAdmissionRequest) updates) =>
      super.copyWith(
              (message) => updates(message as DaemonConnectionAdmissionRequest))
          as DaemonConnectionAdmissionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonConnectionAdmissionRequest create() =>
      DaemonConnectionAdmissionRequest._();
  @$core.override
  DaemonConnectionAdmissionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonConnectionAdmissionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonConnectionAdmissionRequest>(
          create);
  static DaemonConnectionAdmissionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get admissionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set admissionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAdmissionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAdmissionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get daemonId => $_getSZ(2);
  @$pb.TagNumber(3)
  set daemonId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get accountId => $_getSZ(3);
  @$pb.TagNumber(4)
  set accountId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAccountId() => $_has(3);
  @$pb.TagNumber(4)
  void clearAccountId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get agentConnectionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set agentConnectionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAgentConnectionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearAgentConnectionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get release => $_getBF(5);
  @$pb.TagNumber(6)
  set release($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRelease() => $_has(5);
  @$pb.TagNumber(6)
  void clearRelease() => $_clearField(6);
}

class DaemonConnectionAdmissionResponse extends $pb.GeneratedMessage {
  factory DaemonConnectionAdmissionResponse({
    $core.String? requestId,
    $core.String? admissionId,
    DaemonConnectionAdmissionResult? result,
    $core.int? limit,
    $core.String? message,
    $1.CloudEntitlementFailure? entitlementFailure,
  }) {
    final result$ = create();
    if (requestId != null) result$.requestId = requestId;
    if (admissionId != null) result$.admissionId = admissionId;
    if (result != null) result$.result = result;
    if (limit != null) result$.limit = limit;
    if (message != null) result$.message = message;
    if (entitlementFailure != null)
      result$.entitlementFailure = entitlementFailure;
    return result$;
  }

  DaemonConnectionAdmissionResponse._();

  factory DaemonConnectionAdmissionResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DaemonConnectionAdmissionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DaemonConnectionAdmissionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'admissionId')
    ..aE<DaemonConnectionAdmissionResult>(3, _omitFieldNames ? '' : 'result',
        enumValues: DaemonConnectionAdmissionResult.values)
    ..aI(4, _omitFieldNames ? '' : 'limit', fieldType: $pb.PbFieldType.OU3)
    ..aOS(5, _omitFieldNames ? '' : 'message')
    ..aOM<$1.CloudEntitlementFailure>(
        6, _omitFieldNames ? '' : 'entitlementFailure',
        subBuilder: $1.CloudEntitlementFailure.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonConnectionAdmissionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DaemonConnectionAdmissionResponse copyWith(
          void Function(DaemonConnectionAdmissionResponse) updates) =>
      super.copyWith((message) =>
              updates(message as DaemonConnectionAdmissionResponse))
          as DaemonConnectionAdmissionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DaemonConnectionAdmissionResponse create() =>
      DaemonConnectionAdmissionResponse._();
  @$core.override
  DaemonConnectionAdmissionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DaemonConnectionAdmissionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DaemonConnectionAdmissionResponse>(
          create);
  static DaemonConnectionAdmissionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get admissionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set admissionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAdmissionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAdmissionId() => $_clearField(2);

  @$pb.TagNumber(3)
  DaemonConnectionAdmissionResult get result => $_getN(2);
  @$pb.TagNumber(3)
  set result(DaemonConnectionAdmissionResult value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasResult() => $_has(2);
  @$pb.TagNumber(3)
  void clearResult() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get limit => $_getIZ(3);
  @$pb.TagNumber(4)
  set limit($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLimit() => $_has(3);
  @$pb.TagNumber(4)
  void clearLimit() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get message => $_getSZ(4);
  @$pb.TagNumber(5)
  set message($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearMessage() => $_clearField(5);

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

class CloseDaemonConnection extends $pb.GeneratedMessage {
  factory CloseDaemonConnection({
    $core.String? commandId,
    $core.String? correlationId,
    $3.Timestamp? deadline,
    $core.String? daemonId,
    $fixnum.Int64? generation,
    $core.String? reason,
  }) {
    final result = create();
    if (commandId != null) result.commandId = commandId;
    if (correlationId != null) result.correlationId = correlationId;
    if (deadline != null) result.deadline = deadline;
    if (daemonId != null) result.daemonId = daemonId;
    if (generation != null) result.generation = generation;
    if (reason != null) result.reason = reason;
    return result;
  }

  CloseDaemonConnection._();

  factory CloseDaemonConnection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloseDaemonConnection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloseDaemonConnection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'commandId')
    ..aOS(2, _omitFieldNames ? '' : 'correlationId')
    ..aOM<$3.Timestamp>(3, _omitFieldNames ? '' : 'deadline',
        subBuilder: $3.Timestamp.create)
    ..aOS(4, _omitFieldNames ? '' : 'daemonId')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloseDaemonConnection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloseDaemonConnection copyWith(
          void Function(CloseDaemonConnection) updates) =>
      super.copyWith((message) => updates(message as CloseDaemonConnection))
          as CloseDaemonConnection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloseDaemonConnection create() => CloseDaemonConnection._();
  @$core.override
  CloseDaemonConnection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloseDaemonConnection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloseDaemonConnection>(create);
  static CloseDaemonConnection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get commandId => $_getSZ(0);
  @$pb.TagNumber(1)
  set commandId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommandId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommandId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get correlationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set correlationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCorrelationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearCorrelationId() => $_clearField(2);

  @$pb.TagNumber(3)
  $3.Timestamp get deadline => $_getN(2);
  @$pb.TagNumber(3)
  set deadline($3.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDeadline() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeadline() => $_clearField(3);
  @$pb.TagNumber(3)
  $3.Timestamp ensureDeadline() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get daemonId => $_getSZ(3);
  @$pb.TagNumber(4)
  set daemonId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemonId() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemonId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get generation => $_getI64(4);
  @$pb.TagNumber(5)
  set generation($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGeneration() => $_has(4);
  @$pb.TagNumber(5)
  void clearGeneration() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get reason => $_getSZ(5);
  @$pb.TagNumber(6)
  set reason($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasReason() => $_has(5);
  @$pb.TagNumber(6)
  void clearReason() => $_clearField(6);
}

class CloseClientSession extends $pb.GeneratedMessage {
  factory CloseClientSession({
    $core.String? commandId,
    $core.String? correlationId,
    $3.Timestamp? deadline,
    $core.String? sessionId,
    $fixnum.Int64? generation,
    $core.String? reason,
  }) {
    final result = create();
    if (commandId != null) result.commandId = commandId;
    if (correlationId != null) result.correlationId = correlationId;
    if (deadline != null) result.deadline = deadline;
    if (sessionId != null) result.sessionId = sessionId;
    if (generation != null) result.generation = generation;
    if (reason != null) result.reason = reason;
    return result;
  }

  CloseClientSession._();

  factory CloseClientSession.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloseClientSession.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloseClientSession',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'commandId')
    ..aOS(2, _omitFieldNames ? '' : 'correlationId')
    ..aOM<$3.Timestamp>(3, _omitFieldNames ? '' : 'deadline',
        subBuilder: $3.Timestamp.create)
    ..aOS(4, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloseClientSession clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloseClientSession copyWith(void Function(CloseClientSession) updates) =>
      super.copyWith((message) => updates(message as CloseClientSession))
          as CloseClientSession;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloseClientSession create() => CloseClientSession._();
  @$core.override
  CloseClientSession createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloseClientSession getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloseClientSession>(create);
  static CloseClientSession? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get commandId => $_getSZ(0);
  @$pb.TagNumber(1)
  set commandId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommandId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommandId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get correlationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set correlationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCorrelationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearCorrelationId() => $_clearField(2);

  @$pb.TagNumber(3)
  $3.Timestamp get deadline => $_getN(2);
  @$pb.TagNumber(3)
  set deadline($3.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDeadline() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeadline() => $_clearField(3);
  @$pb.TagNumber(3)
  $3.Timestamp ensureDeadline() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get sessionId => $_getSZ(3);
  @$pb.TagNumber(4)
  set sessionId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSessionId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSessionId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get generation => $_getI64(4);
  @$pb.TagNumber(5)
  set generation($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGeneration() => $_has(4);
  @$pb.TagNumber(5)
  void clearGeneration() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get reason => $_getSZ(5);
  @$pb.TagNumber(6)
  set reason($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasReason() => $_has(5);
  @$pb.TagNumber(6)
  void clearReason() => $_clearField(6);
}

class ReselectDaemonEdge extends $pb.GeneratedMessage {
  factory ReselectDaemonEdge({
    $core.String? commandId,
    $core.String? correlationId,
    $3.Timestamp? deadline,
    $core.String? daemonId,
    $fixnum.Int64? generation,
    $fixnum.Int64? preferenceRevision,
  }) {
    final result = create();
    if (commandId != null) result.commandId = commandId;
    if (correlationId != null) result.correlationId = correlationId;
    if (deadline != null) result.deadline = deadline;
    if (daemonId != null) result.daemonId = daemonId;
    if (generation != null) result.generation = generation;
    if (preferenceRevision != null)
      result.preferenceRevision = preferenceRevision;
    return result;
  }

  ReselectDaemonEdge._();

  factory ReselectDaemonEdge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReselectDaemonEdge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReselectDaemonEdge',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'commandId')
    ..aOS(2, _omitFieldNames ? '' : 'correlationId')
    ..aOM<$3.Timestamp>(3, _omitFieldNames ? '' : 'deadline',
        subBuilder: $3.Timestamp.create)
    ..aOS(4, _omitFieldNames ? '' : 'daemonId')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'preferenceRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReselectDaemonEdge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReselectDaemonEdge copyWith(void Function(ReselectDaemonEdge) updates) =>
      super.copyWith((message) => updates(message as ReselectDaemonEdge))
          as ReselectDaemonEdge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReselectDaemonEdge create() => ReselectDaemonEdge._();
  @$core.override
  ReselectDaemonEdge createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReselectDaemonEdge getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReselectDaemonEdge>(create);
  static ReselectDaemonEdge? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get commandId => $_getSZ(0);
  @$pb.TagNumber(1)
  set commandId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommandId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommandId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get correlationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set correlationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCorrelationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearCorrelationId() => $_clearField(2);

  @$pb.TagNumber(3)
  $3.Timestamp get deadline => $_getN(2);
  @$pb.TagNumber(3)
  set deadline($3.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasDeadline() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeadline() => $_clearField(3);
  @$pb.TagNumber(3)
  $3.Timestamp ensureDeadline() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get daemonId => $_getSZ(3);
  @$pb.TagNumber(4)
  set daemonId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemonId() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemonId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get generation => $_getI64(4);
  @$pb.TagNumber(5)
  set generation($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGeneration() => $_has(4);
  @$pb.TagNumber(5)
  void clearGeneration() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get preferenceRevision => $_getI64(5);
  @$pb.TagNumber(6)
  set preferenceRevision($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPreferenceRevision() => $_has(5);
  @$pb.TagNumber(6)
  void clearPreferenceRevision() => $_clearField(6);
}

class EdgeCommandResult extends $pb.GeneratedMessage {
  factory EdgeCommandResult({
    $core.String? commandId,
    $core.String? correlationId,
    CommandResultCode? code,
    $core.String? message,
    $3.Timestamp? completedAt,
  }) {
    final result = create();
    if (commandId != null) result.commandId = commandId;
    if (correlationId != null) result.correlationId = correlationId;
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    if (completedAt != null) result.completedAt = completedAt;
    return result;
  }

  EdgeCommandResult._();

  factory EdgeCommandResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeCommandResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeCommandResult',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'commandId')
    ..aOS(2, _omitFieldNames ? '' : 'correlationId')
    ..aE<CommandResultCode>(3, _omitFieldNames ? '' : 'code',
        enumValues: CommandResultCode.values)
    ..aOS(4, _omitFieldNames ? '' : 'message')
    ..aOM<$3.Timestamp>(5, _omitFieldNames ? '' : 'completedAt',
        subBuilder: $3.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeCommandResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeCommandResult copyWith(void Function(EdgeCommandResult) updates) =>
      super.copyWith((message) => updates(message as EdgeCommandResult))
          as EdgeCommandResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeCommandResult create() => EdgeCommandResult._();
  @$core.override
  EdgeCommandResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeCommandResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeCommandResult>(create);
  static EdgeCommandResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get commandId => $_getSZ(0);
  @$pb.TagNumber(1)
  set commandId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommandId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommandId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get correlationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set correlationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCorrelationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearCorrelationId() => $_clearField(2);

  @$pb.TagNumber(3)
  CommandResultCode get code => $_getN(2);
  @$pb.TagNumber(3)
  set code(CommandResultCode value) => $_setField(3, value);
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

  @$pb.TagNumber(5)
  $3.Timestamp get completedAt => $_getN(4);
  @$pb.TagNumber(5)
  set completedAt($3.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasCompletedAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearCompletedAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $3.Timestamp ensureCompletedAt() => $_ensure(4);
}

enum EdgeEvent_Payload {
  hello,
  snapshotBegin,
  snapshotChunk,
  snapshotEnd,
  runtimeDelta,
  heartbeat,
  configApplied,
  relayReserve,
  commandResult,
  publicCertificateApplied,
  relayRenew,
  relaySettle,
  relayQuery,
  daemonStateQuery,
  identityRenew,
  identityApplied,
  daemonConnectionAdmission,
  relayAuthorize,
  relayUsageBatch,
  daemonStateSync,
  publicCertificateRenew,
  notSet
}

/// EdgeEvent 是 Edge 向 Controller 发送的单调序列 envelope。
/// hello 之后只允许快照、严格连续增量和可合并心跳。
class EdgeEvent extends $pb.GeneratedMessage {
  factory EdgeEvent({
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? senderId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? streamSeq,
    $3.Timestamp? sentAt,
    EdgeHello? hello,
    SnapshotBegin? snapshotBegin,
    SnapshotChunk? snapshotChunk,
    SnapshotEnd? snapshotEnd,
    $2.RuntimeDelta? runtimeDelta,
    EdgeHeartbeat? heartbeat,
    ConfigApplied? configApplied,
    $5.RelayReserveRequest? relayReserve,
    EdgeCommandResult? commandResult,
    $0.EdgePublicCertificateApplied? publicCertificateApplied,
    $5.RelayRenewRequest? relayRenew,
    $5.RelaySettlement? relaySettle,
    $5.RelayQueryRequest? relayQuery,
    DaemonStateQuery? daemonStateQuery,
    EdgeIdentityRenewRequest? identityRenew,
    EdgeIdentityApplied? identityApplied,
    DaemonConnectionAdmissionRequest? daemonConnectionAdmission,
    $5.RelayAuthorizeRequest? relayAuthorize,
    $5.RelayUsageBatch? relayUsageBatch,
    DaemonStateSyncRequest? daemonStateSync,
    $0.EdgePublicCertificateRenewRequest? publicCertificateRenew,
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
    if (snapshotBegin != null) result.snapshotBegin = snapshotBegin;
    if (snapshotChunk != null) result.snapshotChunk = snapshotChunk;
    if (snapshotEnd != null) result.snapshotEnd = snapshotEnd;
    if (runtimeDelta != null) result.runtimeDelta = runtimeDelta;
    if (heartbeat != null) result.heartbeat = heartbeat;
    if (configApplied != null) result.configApplied = configApplied;
    if (relayReserve != null) result.relayReserve = relayReserve;
    if (commandResult != null) result.commandResult = commandResult;
    if (publicCertificateApplied != null)
      result.publicCertificateApplied = publicCertificateApplied;
    if (relayRenew != null) result.relayRenew = relayRenew;
    if (relaySettle != null) result.relaySettle = relaySettle;
    if (relayQuery != null) result.relayQuery = relayQuery;
    if (daemonStateQuery != null) result.daemonStateQuery = daemonStateQuery;
    if (identityRenew != null) result.identityRenew = identityRenew;
    if (identityApplied != null) result.identityApplied = identityApplied;
    if (daemonConnectionAdmission != null)
      result.daemonConnectionAdmission = daemonConnectionAdmission;
    if (relayAuthorize != null) result.relayAuthorize = relayAuthorize;
    if (relayUsageBatch != null) result.relayUsageBatch = relayUsageBatch;
    if (daemonStateSync != null) result.daemonStateSync = daemonStateSync;
    if (publicCertificateRenew != null)
      result.publicCertificateRenew = publicCertificateRenew;
    return result;
  }

  EdgeEvent._();

  factory EdgeEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EdgeEvent_Payload> _EdgeEvent_PayloadByTag =
      {
    20: EdgeEvent_Payload.hello,
    21: EdgeEvent_Payload.snapshotBegin,
    22: EdgeEvent_Payload.snapshotChunk,
    23: EdgeEvent_Payload.snapshotEnd,
    24: EdgeEvent_Payload.runtimeDelta,
    25: EdgeEvent_Payload.heartbeat,
    26: EdgeEvent_Payload.configApplied,
    28: EdgeEvent_Payload.relayReserve,
    29: EdgeEvent_Payload.commandResult,
    30: EdgeEvent_Payload.publicCertificateApplied,
    31: EdgeEvent_Payload.relayRenew,
    32: EdgeEvent_Payload.relaySettle,
    33: EdgeEvent_Payload.relayQuery,
    34: EdgeEvent_Payload.daemonStateQuery,
    35: EdgeEvent_Payload.identityRenew,
    36: EdgeEvent_Payload.identityApplied,
    37: EdgeEvent_Payload.daemonConnectionAdmission,
    38: EdgeEvent_Payload.relayAuthorize,
    39: EdgeEvent_Payload.relayUsageBatch,
    40: EdgeEvent_Payload.daemonStateSync,
    41: EdgeEvent_Payload.publicCertificateRenew,
    0: EdgeEvent_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      20,
      21,
      22,
      23,
      24,
      25,
      26,
      28,
      29,
      30,
      31,
      32,
      33,
      34,
      35,
      36,
      37,
      38,
      39,
      40,
      41
    ])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'messageId')
    ..aOS(3, _omitFieldNames ? '' : 'senderId')
    ..aOS(4, _omitFieldNames ? '' : 'bootId')
    ..aOS(5, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$3.Timestamp>(7, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $3.Timestamp.create)
    ..aOM<EdgeHello>(20, _omitFieldNames ? '' : 'hello',
        subBuilder: EdgeHello.create)
    ..aOM<SnapshotBegin>(21, _omitFieldNames ? '' : 'snapshotBegin',
        subBuilder: SnapshotBegin.create)
    ..aOM<SnapshotChunk>(22, _omitFieldNames ? '' : 'snapshotChunk',
        subBuilder: SnapshotChunk.create)
    ..aOM<SnapshotEnd>(23, _omitFieldNames ? '' : 'snapshotEnd',
        subBuilder: SnapshotEnd.create)
    ..aOM<$2.RuntimeDelta>(24, _omitFieldNames ? '' : 'runtimeDelta',
        subBuilder: $2.RuntimeDelta.create)
    ..aOM<EdgeHeartbeat>(25, _omitFieldNames ? '' : 'heartbeat',
        subBuilder: EdgeHeartbeat.create)
    ..aOM<ConfigApplied>(26, _omitFieldNames ? '' : 'configApplied',
        subBuilder: ConfigApplied.create)
    ..aOM<$5.RelayReserveRequest>(28, _omitFieldNames ? '' : 'relayReserve',
        subBuilder: $5.RelayReserveRequest.create)
    ..aOM<EdgeCommandResult>(29, _omitFieldNames ? '' : 'commandResult',
        subBuilder: EdgeCommandResult.create)
    ..aOM<$0.EdgePublicCertificateApplied>(
        30, _omitFieldNames ? '' : 'publicCertificateApplied',
        subBuilder: $0.EdgePublicCertificateApplied.create)
    ..aOM<$5.RelayRenewRequest>(31, _omitFieldNames ? '' : 'relayRenew',
        subBuilder: $5.RelayRenewRequest.create)
    ..aOM<$5.RelaySettlement>(32, _omitFieldNames ? '' : 'relaySettle',
        subBuilder: $5.RelaySettlement.create)
    ..aOM<$5.RelayQueryRequest>(33, _omitFieldNames ? '' : 'relayQuery',
        subBuilder: $5.RelayQueryRequest.create)
    ..aOM<DaemonStateQuery>(34, _omitFieldNames ? '' : 'daemonStateQuery',
        subBuilder: DaemonStateQuery.create)
    ..aOM<EdgeIdentityRenewRequest>(35, _omitFieldNames ? '' : 'identityRenew',
        subBuilder: EdgeIdentityRenewRequest.create)
    ..aOM<EdgeIdentityApplied>(36, _omitFieldNames ? '' : 'identityApplied',
        subBuilder: EdgeIdentityApplied.create)
    ..aOM<DaemonConnectionAdmissionRequest>(
        37, _omitFieldNames ? '' : 'daemonConnectionAdmission',
        subBuilder: DaemonConnectionAdmissionRequest.create)
    ..aOM<$5.RelayAuthorizeRequest>(38, _omitFieldNames ? '' : 'relayAuthorize',
        subBuilder: $5.RelayAuthorizeRequest.create)
    ..aOM<$5.RelayUsageBatch>(39, _omitFieldNames ? '' : 'relayUsageBatch',
        subBuilder: $5.RelayUsageBatch.create)
    ..aOM<DaemonStateSyncRequest>(40, _omitFieldNames ? '' : 'daemonStateSync',
        subBuilder: DaemonStateSyncRequest.create)
    ..aOM<$0.EdgePublicCertificateRenewRequest>(
        41, _omitFieldNames ? '' : 'publicCertificateRenew',
        subBuilder: $0.EdgePublicCertificateRenewRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeEvent copyWith(void Function(EdgeEvent) updates) =>
      super.copyWith((message) => updates(message as EdgeEvent)) as EdgeEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeEvent create() => EdgeEvent._();
  @$core.override
  EdgeEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeEvent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EdgeEvent>(create);
  static EdgeEvent? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  @$pb.TagNumber(33)
  @$pb.TagNumber(34)
  @$pb.TagNumber(35)
  @$pb.TagNumber(36)
  @$pb.TagNumber(37)
  @$pb.TagNumber(38)
  @$pb.TagNumber(39)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
  EdgeEvent_Payload whichPayload() => _EdgeEvent_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  @$pb.TagNumber(33)
  @$pb.TagNumber(34)
  @$pb.TagNumber(35)
  @$pb.TagNumber(36)
  @$pb.TagNumber(37)
  @$pb.TagNumber(38)
  @$pb.TagNumber(39)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
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
  $3.Timestamp get sentAt => $_getN(6);
  @$pb.TagNumber(7)
  set sentAt($3.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSentAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearSentAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $3.Timestamp ensureSentAt() => $_ensure(6);

  @$pb.TagNumber(20)
  EdgeHello get hello => $_getN(7);
  @$pb.TagNumber(20)
  set hello(EdgeHello value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasHello() => $_has(7);
  @$pb.TagNumber(20)
  void clearHello() => $_clearField(20);
  @$pb.TagNumber(20)
  EdgeHello ensureHello() => $_ensure(7);

  @$pb.TagNumber(21)
  SnapshotBegin get snapshotBegin => $_getN(8);
  @$pb.TagNumber(21)
  set snapshotBegin(SnapshotBegin value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasSnapshotBegin() => $_has(8);
  @$pb.TagNumber(21)
  void clearSnapshotBegin() => $_clearField(21);
  @$pb.TagNumber(21)
  SnapshotBegin ensureSnapshotBegin() => $_ensure(8);

  @$pb.TagNumber(22)
  SnapshotChunk get snapshotChunk => $_getN(9);
  @$pb.TagNumber(22)
  set snapshotChunk(SnapshotChunk value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasSnapshotChunk() => $_has(9);
  @$pb.TagNumber(22)
  void clearSnapshotChunk() => $_clearField(22);
  @$pb.TagNumber(22)
  SnapshotChunk ensureSnapshotChunk() => $_ensure(9);

  @$pb.TagNumber(23)
  SnapshotEnd get snapshotEnd => $_getN(10);
  @$pb.TagNumber(23)
  set snapshotEnd(SnapshotEnd value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasSnapshotEnd() => $_has(10);
  @$pb.TagNumber(23)
  void clearSnapshotEnd() => $_clearField(23);
  @$pb.TagNumber(23)
  SnapshotEnd ensureSnapshotEnd() => $_ensure(10);

  @$pb.TagNumber(24)
  $2.RuntimeDelta get runtimeDelta => $_getN(11);
  @$pb.TagNumber(24)
  set runtimeDelta($2.RuntimeDelta value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasRuntimeDelta() => $_has(11);
  @$pb.TagNumber(24)
  void clearRuntimeDelta() => $_clearField(24);
  @$pb.TagNumber(24)
  $2.RuntimeDelta ensureRuntimeDelta() => $_ensure(11);

  @$pb.TagNumber(25)
  EdgeHeartbeat get heartbeat => $_getN(12);
  @$pb.TagNumber(25)
  set heartbeat(EdgeHeartbeat value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasHeartbeat() => $_has(12);
  @$pb.TagNumber(25)
  void clearHeartbeat() => $_clearField(25);
  @$pb.TagNumber(25)
  EdgeHeartbeat ensureHeartbeat() => $_ensure(12);

  @$pb.TagNumber(26)
  ConfigApplied get configApplied => $_getN(13);
  @$pb.TagNumber(26)
  set configApplied(ConfigApplied value) => $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasConfigApplied() => $_has(13);
  @$pb.TagNumber(26)
  void clearConfigApplied() => $_clearField(26);
  @$pb.TagNumber(26)
  ConfigApplied ensureConfigApplied() => $_ensure(13);

  @$pb.TagNumber(28)
  $5.RelayReserveRequest get relayReserve => $_getN(14);
  @$pb.TagNumber(28)
  set relayReserve($5.RelayReserveRequest value) => $_setField(28, value);
  @$pb.TagNumber(28)
  $core.bool hasRelayReserve() => $_has(14);
  @$pb.TagNumber(28)
  void clearRelayReserve() => $_clearField(28);
  @$pb.TagNumber(28)
  $5.RelayReserveRequest ensureRelayReserve() => $_ensure(14);

  @$pb.TagNumber(29)
  EdgeCommandResult get commandResult => $_getN(15);
  @$pb.TagNumber(29)
  set commandResult(EdgeCommandResult value) => $_setField(29, value);
  @$pb.TagNumber(29)
  $core.bool hasCommandResult() => $_has(15);
  @$pb.TagNumber(29)
  void clearCommandResult() => $_clearField(29);
  @$pb.TagNumber(29)
  EdgeCommandResult ensureCommandResult() => $_ensure(15);

  @$pb.TagNumber(30)
  $0.EdgePublicCertificateApplied get publicCertificateApplied => $_getN(16);
  @$pb.TagNumber(30)
  set publicCertificateApplied($0.EdgePublicCertificateApplied value) =>
      $_setField(30, value);
  @$pb.TagNumber(30)
  $core.bool hasPublicCertificateApplied() => $_has(16);
  @$pb.TagNumber(30)
  void clearPublicCertificateApplied() => $_clearField(30);
  @$pb.TagNumber(30)
  $0.EdgePublicCertificateApplied ensurePublicCertificateApplied() =>
      $_ensure(16);

  @$pb.TagNumber(31)
  $5.RelayRenewRequest get relayRenew => $_getN(17);
  @$pb.TagNumber(31)
  set relayRenew($5.RelayRenewRequest value) => $_setField(31, value);
  @$pb.TagNumber(31)
  $core.bool hasRelayRenew() => $_has(17);
  @$pb.TagNumber(31)
  void clearRelayRenew() => $_clearField(31);
  @$pb.TagNumber(31)
  $5.RelayRenewRequest ensureRelayRenew() => $_ensure(17);

  @$pb.TagNumber(32)
  $5.RelaySettlement get relaySettle => $_getN(18);
  @$pb.TagNumber(32)
  set relaySettle($5.RelaySettlement value) => $_setField(32, value);
  @$pb.TagNumber(32)
  $core.bool hasRelaySettle() => $_has(18);
  @$pb.TagNumber(32)
  void clearRelaySettle() => $_clearField(32);
  @$pb.TagNumber(32)
  $5.RelaySettlement ensureRelaySettle() => $_ensure(18);

  @$pb.TagNumber(33)
  $5.RelayQueryRequest get relayQuery => $_getN(19);
  @$pb.TagNumber(33)
  set relayQuery($5.RelayQueryRequest value) => $_setField(33, value);
  @$pb.TagNumber(33)
  $core.bool hasRelayQuery() => $_has(19);
  @$pb.TagNumber(33)
  void clearRelayQuery() => $_clearField(33);
  @$pb.TagNumber(33)
  $5.RelayQueryRequest ensureRelayQuery() => $_ensure(19);

  @$pb.TagNumber(34)
  DaemonStateQuery get daemonStateQuery => $_getN(20);
  @$pb.TagNumber(34)
  set daemonStateQuery(DaemonStateQuery value) => $_setField(34, value);
  @$pb.TagNumber(34)
  $core.bool hasDaemonStateQuery() => $_has(20);
  @$pb.TagNumber(34)
  void clearDaemonStateQuery() => $_clearField(34);
  @$pb.TagNumber(34)
  DaemonStateQuery ensureDaemonStateQuery() => $_ensure(20);

  @$pb.TagNumber(35)
  EdgeIdentityRenewRequest get identityRenew => $_getN(21);
  @$pb.TagNumber(35)
  set identityRenew(EdgeIdentityRenewRequest value) => $_setField(35, value);
  @$pb.TagNumber(35)
  $core.bool hasIdentityRenew() => $_has(21);
  @$pb.TagNumber(35)
  void clearIdentityRenew() => $_clearField(35);
  @$pb.TagNumber(35)
  EdgeIdentityRenewRequest ensureIdentityRenew() => $_ensure(21);

  @$pb.TagNumber(36)
  EdgeIdentityApplied get identityApplied => $_getN(22);
  @$pb.TagNumber(36)
  set identityApplied(EdgeIdentityApplied value) => $_setField(36, value);
  @$pb.TagNumber(36)
  $core.bool hasIdentityApplied() => $_has(22);
  @$pb.TagNumber(36)
  void clearIdentityApplied() => $_clearField(36);
  @$pb.TagNumber(36)
  EdgeIdentityApplied ensureIdentityApplied() => $_ensure(22);

  @$pb.TagNumber(37)
  DaemonConnectionAdmissionRequest get daemonConnectionAdmission => $_getN(23);
  @$pb.TagNumber(37)
  set daemonConnectionAdmission(DaemonConnectionAdmissionRequest value) =>
      $_setField(37, value);
  @$pb.TagNumber(37)
  $core.bool hasDaemonConnectionAdmission() => $_has(23);
  @$pb.TagNumber(37)
  void clearDaemonConnectionAdmission() => $_clearField(37);
  @$pb.TagNumber(37)
  DaemonConnectionAdmissionRequest ensureDaemonConnectionAdmission() =>
      $_ensure(23);

  @$pb.TagNumber(38)
  $5.RelayAuthorizeRequest get relayAuthorize => $_getN(24);
  @$pb.TagNumber(38)
  set relayAuthorize($5.RelayAuthorizeRequest value) => $_setField(38, value);
  @$pb.TagNumber(38)
  $core.bool hasRelayAuthorize() => $_has(24);
  @$pb.TagNumber(38)
  void clearRelayAuthorize() => $_clearField(38);
  @$pb.TagNumber(38)
  $5.RelayAuthorizeRequest ensureRelayAuthorize() => $_ensure(24);

  @$pb.TagNumber(39)
  $5.RelayUsageBatch get relayUsageBatch => $_getN(25);
  @$pb.TagNumber(39)
  set relayUsageBatch($5.RelayUsageBatch value) => $_setField(39, value);
  @$pb.TagNumber(39)
  $core.bool hasRelayUsageBatch() => $_has(25);
  @$pb.TagNumber(39)
  void clearRelayUsageBatch() => $_clearField(39);
  @$pb.TagNumber(39)
  $5.RelayUsageBatch ensureRelayUsageBatch() => $_ensure(25);

  @$pb.TagNumber(40)
  DaemonStateSyncRequest get daemonStateSync => $_getN(26);
  @$pb.TagNumber(40)
  set daemonStateSync(DaemonStateSyncRequest value) => $_setField(40, value);
  @$pb.TagNumber(40)
  $core.bool hasDaemonStateSync() => $_has(26);
  @$pb.TagNumber(40)
  void clearDaemonStateSync() => $_clearField(40);
  @$pb.TagNumber(40)
  DaemonStateSyncRequest ensureDaemonStateSync() => $_ensure(26);

  @$pb.TagNumber(41)
  $0.EdgePublicCertificateRenewRequest get publicCertificateRenew => $_getN(27);
  @$pb.TagNumber(41)
  set publicCertificateRenew($0.EdgePublicCertificateRenewRequest value) =>
      $_setField(41, value);
  @$pb.TagNumber(41)
  $core.bool hasPublicCertificateRenew() => $_has(27);
  @$pb.TagNumber(41)
  void clearPublicCertificateRenew() => $_clearField(41);
  @$pb.TagNumber(41)
  $0.EdgePublicCertificateRenewRequest ensurePublicCertificateRenew() =>
      $_ensure(27);
}

enum ControllerCommand_Payload {
  welcome,
  snapshotAccepted,
  resyncRequired,
  desiredConfig,
  bindingKeyBundle,
  relayReserve,
  closeDaemon,
  closeSession,
  publicCertificateRenew,
  relayRenew,
  relaySettle,
  relayQuery,
  daemonStateDelta,
  daemonStateQueryResult,
  reselectDaemonEdge,
  identityRenew,
  daemonConnectionAdmission,
  relayAuthorize,
  relayUsageAck,
  relayAccountAction,
  daemonStateSyncChunk,
  daemonStateSyncEnd,
  notSet
}

/// ControllerCommand 是 Controller 向 Edge 发送的单调序列 envelope。
/// welcome 之后由 snapshot_accepted 或 resync_required 驱动同步状态机。
class ControllerCommand extends $pb.GeneratedMessage {
  factory ControllerCommand({
    $core.int? protocolVersion,
    $core.String? messageId,
    $core.String? senderId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? streamSeq,
    $3.Timestamp? sentAt,
    EdgeWelcome? welcome,
    SnapshotAccepted? snapshotAccepted,
    ResyncRequired? resyncRequired,
    $6.SignedEdgeDesiredConfig? desiredConfig,
    $1.KeyBundle? bindingKeyBundle,
    $5.RelayReserveResponse? relayReserve,
    CloseDaemonConnection? closeDaemon,
    CloseClientSession? closeSession,
    $0.EdgePublicCertificateRenewResponse? publicCertificateRenew,
    $5.RelayRenewResponse? relayRenew,
    $5.RelaySettlementAck? relaySettle,
    $5.RelayQueryResponse? relayQuery,
    DaemonStateDelta? daemonStateDelta,
    DaemonStateQueryResult? daemonStateQueryResult,
    ReselectDaemonEdge? reselectDaemonEdge,
    EdgeIdentityRenewResponse? identityRenew,
    DaemonConnectionAdmissionResponse? daemonConnectionAdmission,
    $5.RelayAuthorizeResponse? relayAuthorize,
    $5.RelayUsageAck? relayUsageAck,
    $5.RelayAccountAction? relayAccountAction,
    DaemonStateSyncChunk? daemonStateSyncChunk,
    DaemonStateSyncEnd? daemonStateSyncEnd,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (messageId != null) result.messageId = messageId;
    if (senderId != null) result.senderId = senderId;
    if (bootId != null) result.bootId = bootId;
    if (connectionId != null) result.connectionId = connectionId;
    if (streamSeq != null) result.streamSeq = streamSeq;
    if (sentAt != null) result.sentAt = sentAt;
    if (welcome != null) result.welcome = welcome;
    if (snapshotAccepted != null) result.snapshotAccepted = snapshotAccepted;
    if (resyncRequired != null) result.resyncRequired = resyncRequired;
    if (desiredConfig != null) result.desiredConfig = desiredConfig;
    if (bindingKeyBundle != null) result.bindingKeyBundle = bindingKeyBundle;
    if (relayReserve != null) result.relayReserve = relayReserve;
    if (closeDaemon != null) result.closeDaemon = closeDaemon;
    if (closeSession != null) result.closeSession = closeSession;
    if (publicCertificateRenew != null)
      result.publicCertificateRenew = publicCertificateRenew;
    if (relayRenew != null) result.relayRenew = relayRenew;
    if (relaySettle != null) result.relaySettle = relaySettle;
    if (relayQuery != null) result.relayQuery = relayQuery;
    if (daemonStateDelta != null) result.daemonStateDelta = daemonStateDelta;
    if (daemonStateQueryResult != null)
      result.daemonStateQueryResult = daemonStateQueryResult;
    if (reselectDaemonEdge != null)
      result.reselectDaemonEdge = reselectDaemonEdge;
    if (identityRenew != null) result.identityRenew = identityRenew;
    if (daemonConnectionAdmission != null)
      result.daemonConnectionAdmission = daemonConnectionAdmission;
    if (relayAuthorize != null) result.relayAuthorize = relayAuthorize;
    if (relayUsageAck != null) result.relayUsageAck = relayUsageAck;
    if (relayAccountAction != null)
      result.relayAccountAction = relayAccountAction;
    if (daemonStateSyncChunk != null)
      result.daemonStateSyncChunk = daemonStateSyncChunk;
    if (daemonStateSyncEnd != null)
      result.daemonStateSyncEnd = daemonStateSyncEnd;
    return result;
  }

  ControllerCommand._();

  factory ControllerCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ControllerCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ControllerCommand_Payload>
      _ControllerCommand_PayloadByTag = {
    20: ControllerCommand_Payload.welcome,
    21: ControllerCommand_Payload.snapshotAccepted,
    22: ControllerCommand_Payload.resyncRequired,
    23: ControllerCommand_Payload.desiredConfig,
    24: ControllerCommand_Payload.bindingKeyBundle,
    25: ControllerCommand_Payload.relayReserve,
    26: ControllerCommand_Payload.closeDaemon,
    27: ControllerCommand_Payload.closeSession,
    28: ControllerCommand_Payload.publicCertificateRenew,
    29: ControllerCommand_Payload.relayRenew,
    30: ControllerCommand_Payload.relaySettle,
    31: ControllerCommand_Payload.relayQuery,
    32: ControllerCommand_Payload.daemonStateDelta,
    33: ControllerCommand_Payload.daemonStateQueryResult,
    34: ControllerCommand_Payload.reselectDaemonEdge,
    35: ControllerCommand_Payload.identityRenew,
    36: ControllerCommand_Payload.daemonConnectionAdmission,
    37: ControllerCommand_Payload.relayAuthorize,
    38: ControllerCommand_Payload.relayUsageAck,
    39: ControllerCommand_Payload.relayAccountAction,
    40: ControllerCommand_Payload.daemonStateSyncChunk,
    41: ControllerCommand_Payload.daemonStateSyncEnd,
    0: ControllerCommand_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ControllerCommand',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      20,
      21,
      22,
      23,
      24,
      25,
      26,
      27,
      28,
      29,
      30,
      31,
      32,
      33,
      34,
      35,
      36,
      37,
      38,
      39,
      40,
      41
    ])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'messageId')
    ..aOS(3, _omitFieldNames ? '' : 'senderId')
    ..aOS(4, _omitFieldNames ? '' : 'bootId')
    ..aOS(5, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'streamSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$3.Timestamp>(7, _omitFieldNames ? '' : 'sentAt',
        subBuilder: $3.Timestamp.create)
    ..aOM<EdgeWelcome>(20, _omitFieldNames ? '' : 'welcome',
        subBuilder: EdgeWelcome.create)
    ..aOM<SnapshotAccepted>(21, _omitFieldNames ? '' : 'snapshotAccepted',
        subBuilder: SnapshotAccepted.create)
    ..aOM<ResyncRequired>(22, _omitFieldNames ? '' : 'resyncRequired',
        subBuilder: ResyncRequired.create)
    ..aOM<$6.SignedEdgeDesiredConfig>(
        23, _omitFieldNames ? '' : 'desiredConfig',
        subBuilder: $6.SignedEdgeDesiredConfig.create)
    ..aOM<$1.KeyBundle>(24, _omitFieldNames ? '' : 'bindingKeyBundle',
        subBuilder: $1.KeyBundle.create)
    ..aOM<$5.RelayReserveResponse>(25, _omitFieldNames ? '' : 'relayReserve',
        subBuilder: $5.RelayReserveResponse.create)
    ..aOM<CloseDaemonConnection>(26, _omitFieldNames ? '' : 'closeDaemon',
        subBuilder: CloseDaemonConnection.create)
    ..aOM<CloseClientSession>(27, _omitFieldNames ? '' : 'closeSession',
        subBuilder: CloseClientSession.create)
    ..aOM<$0.EdgePublicCertificateRenewResponse>(
        28, _omitFieldNames ? '' : 'publicCertificateRenew',
        subBuilder: $0.EdgePublicCertificateRenewResponse.create)
    ..aOM<$5.RelayRenewResponse>(29, _omitFieldNames ? '' : 'relayRenew',
        subBuilder: $5.RelayRenewResponse.create)
    ..aOM<$5.RelaySettlementAck>(30, _omitFieldNames ? '' : 'relaySettle',
        subBuilder: $5.RelaySettlementAck.create)
    ..aOM<$5.RelayQueryResponse>(31, _omitFieldNames ? '' : 'relayQuery',
        subBuilder: $5.RelayQueryResponse.create)
    ..aOM<DaemonStateDelta>(32, _omitFieldNames ? '' : 'daemonStateDelta',
        subBuilder: DaemonStateDelta.create)
    ..aOM<DaemonStateQueryResult>(
        33, _omitFieldNames ? '' : 'daemonStateQueryResult',
        subBuilder: DaemonStateQueryResult.create)
    ..aOM<ReselectDaemonEdge>(34, _omitFieldNames ? '' : 'reselectDaemonEdge',
        subBuilder: ReselectDaemonEdge.create)
    ..aOM<EdgeIdentityRenewResponse>(35, _omitFieldNames ? '' : 'identityRenew',
        subBuilder: EdgeIdentityRenewResponse.create)
    ..aOM<DaemonConnectionAdmissionResponse>(
        36, _omitFieldNames ? '' : 'daemonConnectionAdmission',
        subBuilder: DaemonConnectionAdmissionResponse.create)
    ..aOM<$5.RelayAuthorizeResponse>(
        37, _omitFieldNames ? '' : 'relayAuthorize',
        subBuilder: $5.RelayAuthorizeResponse.create)
    ..aOM<$5.RelayUsageAck>(38, _omitFieldNames ? '' : 'relayUsageAck',
        subBuilder: $5.RelayUsageAck.create)
    ..aOM<$5.RelayAccountAction>(
        39, _omitFieldNames ? '' : 'relayAccountAction',
        subBuilder: $5.RelayAccountAction.create)
    ..aOM<DaemonStateSyncChunk>(
        40, _omitFieldNames ? '' : 'daemonStateSyncChunk',
        subBuilder: DaemonStateSyncChunk.create)
    ..aOM<DaemonStateSyncEnd>(41, _omitFieldNames ? '' : 'daemonStateSyncEnd',
        subBuilder: DaemonStateSyncEnd.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ControllerCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ControllerCommand copyWith(void Function(ControllerCommand) updates) =>
      super.copyWith((message) => updates(message as ControllerCommand))
          as ControllerCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ControllerCommand create() => ControllerCommand._();
  @$core.override
  ControllerCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ControllerCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ControllerCommand>(create);
  static ControllerCommand? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  @$pb.TagNumber(33)
  @$pb.TagNumber(34)
  @$pb.TagNumber(35)
  @$pb.TagNumber(36)
  @$pb.TagNumber(37)
  @$pb.TagNumber(38)
  @$pb.TagNumber(39)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
  ControllerCommand_Payload whichPayload() =>
      _ControllerCommand_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  @$pb.TagNumber(33)
  @$pb.TagNumber(34)
  @$pb.TagNumber(35)
  @$pb.TagNumber(36)
  @$pb.TagNumber(37)
  @$pb.TagNumber(38)
  @$pb.TagNumber(39)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
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
  $3.Timestamp get sentAt => $_getN(6);
  @$pb.TagNumber(7)
  set sentAt($3.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasSentAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearSentAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $3.Timestamp ensureSentAt() => $_ensure(6);

  @$pb.TagNumber(20)
  EdgeWelcome get welcome => $_getN(7);
  @$pb.TagNumber(20)
  set welcome(EdgeWelcome value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasWelcome() => $_has(7);
  @$pb.TagNumber(20)
  void clearWelcome() => $_clearField(20);
  @$pb.TagNumber(20)
  EdgeWelcome ensureWelcome() => $_ensure(7);

  @$pb.TagNumber(21)
  SnapshotAccepted get snapshotAccepted => $_getN(8);
  @$pb.TagNumber(21)
  set snapshotAccepted(SnapshotAccepted value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasSnapshotAccepted() => $_has(8);
  @$pb.TagNumber(21)
  void clearSnapshotAccepted() => $_clearField(21);
  @$pb.TagNumber(21)
  SnapshotAccepted ensureSnapshotAccepted() => $_ensure(8);

  @$pb.TagNumber(22)
  ResyncRequired get resyncRequired => $_getN(9);
  @$pb.TagNumber(22)
  set resyncRequired(ResyncRequired value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasResyncRequired() => $_has(9);
  @$pb.TagNumber(22)
  void clearResyncRequired() => $_clearField(22);
  @$pb.TagNumber(22)
  ResyncRequired ensureResyncRequired() => $_ensure(9);

  @$pb.TagNumber(23)
  $6.SignedEdgeDesiredConfig get desiredConfig => $_getN(10);
  @$pb.TagNumber(23)
  set desiredConfig($6.SignedEdgeDesiredConfig value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasDesiredConfig() => $_has(10);
  @$pb.TagNumber(23)
  void clearDesiredConfig() => $_clearField(23);
  @$pb.TagNumber(23)
  $6.SignedEdgeDesiredConfig ensureDesiredConfig() => $_ensure(10);

  @$pb.TagNumber(24)
  $1.KeyBundle get bindingKeyBundle => $_getN(11);
  @$pb.TagNumber(24)
  set bindingKeyBundle($1.KeyBundle value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasBindingKeyBundle() => $_has(11);
  @$pb.TagNumber(24)
  void clearBindingKeyBundle() => $_clearField(24);
  @$pb.TagNumber(24)
  $1.KeyBundle ensureBindingKeyBundle() => $_ensure(11);

  @$pb.TagNumber(25)
  $5.RelayReserveResponse get relayReserve => $_getN(12);
  @$pb.TagNumber(25)
  set relayReserve($5.RelayReserveResponse value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasRelayReserve() => $_has(12);
  @$pb.TagNumber(25)
  void clearRelayReserve() => $_clearField(25);
  @$pb.TagNumber(25)
  $5.RelayReserveResponse ensureRelayReserve() => $_ensure(12);

  @$pb.TagNumber(26)
  CloseDaemonConnection get closeDaemon => $_getN(13);
  @$pb.TagNumber(26)
  set closeDaemon(CloseDaemonConnection value) => $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasCloseDaemon() => $_has(13);
  @$pb.TagNumber(26)
  void clearCloseDaemon() => $_clearField(26);
  @$pb.TagNumber(26)
  CloseDaemonConnection ensureCloseDaemon() => $_ensure(13);

  @$pb.TagNumber(27)
  CloseClientSession get closeSession => $_getN(14);
  @$pb.TagNumber(27)
  set closeSession(CloseClientSession value) => $_setField(27, value);
  @$pb.TagNumber(27)
  $core.bool hasCloseSession() => $_has(14);
  @$pb.TagNumber(27)
  void clearCloseSession() => $_clearField(27);
  @$pb.TagNumber(27)
  CloseClientSession ensureCloseSession() => $_ensure(14);

  @$pb.TagNumber(28)
  $0.EdgePublicCertificateRenewResponse get publicCertificateRenew =>
      $_getN(15);
  @$pb.TagNumber(28)
  set publicCertificateRenew($0.EdgePublicCertificateRenewResponse value) =>
      $_setField(28, value);
  @$pb.TagNumber(28)
  $core.bool hasPublicCertificateRenew() => $_has(15);
  @$pb.TagNumber(28)
  void clearPublicCertificateRenew() => $_clearField(28);
  @$pb.TagNumber(28)
  $0.EdgePublicCertificateRenewResponse ensurePublicCertificateRenew() =>
      $_ensure(15);

  @$pb.TagNumber(29)
  $5.RelayRenewResponse get relayRenew => $_getN(16);
  @$pb.TagNumber(29)
  set relayRenew($5.RelayRenewResponse value) => $_setField(29, value);
  @$pb.TagNumber(29)
  $core.bool hasRelayRenew() => $_has(16);
  @$pb.TagNumber(29)
  void clearRelayRenew() => $_clearField(29);
  @$pb.TagNumber(29)
  $5.RelayRenewResponse ensureRelayRenew() => $_ensure(16);

  @$pb.TagNumber(30)
  $5.RelaySettlementAck get relaySettle => $_getN(17);
  @$pb.TagNumber(30)
  set relaySettle($5.RelaySettlementAck value) => $_setField(30, value);
  @$pb.TagNumber(30)
  $core.bool hasRelaySettle() => $_has(17);
  @$pb.TagNumber(30)
  void clearRelaySettle() => $_clearField(30);
  @$pb.TagNumber(30)
  $5.RelaySettlementAck ensureRelaySettle() => $_ensure(17);

  @$pb.TagNumber(31)
  $5.RelayQueryResponse get relayQuery => $_getN(18);
  @$pb.TagNumber(31)
  set relayQuery($5.RelayQueryResponse value) => $_setField(31, value);
  @$pb.TagNumber(31)
  $core.bool hasRelayQuery() => $_has(18);
  @$pb.TagNumber(31)
  void clearRelayQuery() => $_clearField(31);
  @$pb.TagNumber(31)
  $5.RelayQueryResponse ensureRelayQuery() => $_ensure(18);

  @$pb.TagNumber(32)
  DaemonStateDelta get daemonStateDelta => $_getN(19);
  @$pb.TagNumber(32)
  set daemonStateDelta(DaemonStateDelta value) => $_setField(32, value);
  @$pb.TagNumber(32)
  $core.bool hasDaemonStateDelta() => $_has(19);
  @$pb.TagNumber(32)
  void clearDaemonStateDelta() => $_clearField(32);
  @$pb.TagNumber(32)
  DaemonStateDelta ensureDaemonStateDelta() => $_ensure(19);

  @$pb.TagNumber(33)
  DaemonStateQueryResult get daemonStateQueryResult => $_getN(20);
  @$pb.TagNumber(33)
  set daemonStateQueryResult(DaemonStateQueryResult value) =>
      $_setField(33, value);
  @$pb.TagNumber(33)
  $core.bool hasDaemonStateQueryResult() => $_has(20);
  @$pb.TagNumber(33)
  void clearDaemonStateQueryResult() => $_clearField(33);
  @$pb.TagNumber(33)
  DaemonStateQueryResult ensureDaemonStateQueryResult() => $_ensure(20);

  @$pb.TagNumber(34)
  ReselectDaemonEdge get reselectDaemonEdge => $_getN(21);
  @$pb.TagNumber(34)
  set reselectDaemonEdge(ReselectDaemonEdge value) => $_setField(34, value);
  @$pb.TagNumber(34)
  $core.bool hasReselectDaemonEdge() => $_has(21);
  @$pb.TagNumber(34)
  void clearReselectDaemonEdge() => $_clearField(34);
  @$pb.TagNumber(34)
  ReselectDaemonEdge ensureReselectDaemonEdge() => $_ensure(21);

  @$pb.TagNumber(35)
  EdgeIdentityRenewResponse get identityRenew => $_getN(22);
  @$pb.TagNumber(35)
  set identityRenew(EdgeIdentityRenewResponse value) => $_setField(35, value);
  @$pb.TagNumber(35)
  $core.bool hasIdentityRenew() => $_has(22);
  @$pb.TagNumber(35)
  void clearIdentityRenew() => $_clearField(35);
  @$pb.TagNumber(35)
  EdgeIdentityRenewResponse ensureIdentityRenew() => $_ensure(22);

  @$pb.TagNumber(36)
  DaemonConnectionAdmissionResponse get daemonConnectionAdmission => $_getN(23);
  @$pb.TagNumber(36)
  set daemonConnectionAdmission(DaemonConnectionAdmissionResponse value) =>
      $_setField(36, value);
  @$pb.TagNumber(36)
  $core.bool hasDaemonConnectionAdmission() => $_has(23);
  @$pb.TagNumber(36)
  void clearDaemonConnectionAdmission() => $_clearField(36);
  @$pb.TagNumber(36)
  DaemonConnectionAdmissionResponse ensureDaemonConnectionAdmission() =>
      $_ensure(23);

  @$pb.TagNumber(37)
  $5.RelayAuthorizeResponse get relayAuthorize => $_getN(24);
  @$pb.TagNumber(37)
  set relayAuthorize($5.RelayAuthorizeResponse value) => $_setField(37, value);
  @$pb.TagNumber(37)
  $core.bool hasRelayAuthorize() => $_has(24);
  @$pb.TagNumber(37)
  void clearRelayAuthorize() => $_clearField(37);
  @$pb.TagNumber(37)
  $5.RelayAuthorizeResponse ensureRelayAuthorize() => $_ensure(24);

  @$pb.TagNumber(38)
  $5.RelayUsageAck get relayUsageAck => $_getN(25);
  @$pb.TagNumber(38)
  set relayUsageAck($5.RelayUsageAck value) => $_setField(38, value);
  @$pb.TagNumber(38)
  $core.bool hasRelayUsageAck() => $_has(25);
  @$pb.TagNumber(38)
  void clearRelayUsageAck() => $_clearField(38);
  @$pb.TagNumber(38)
  $5.RelayUsageAck ensureRelayUsageAck() => $_ensure(25);

  @$pb.TagNumber(39)
  $5.RelayAccountAction get relayAccountAction => $_getN(26);
  @$pb.TagNumber(39)
  set relayAccountAction($5.RelayAccountAction value) => $_setField(39, value);
  @$pb.TagNumber(39)
  $core.bool hasRelayAccountAction() => $_has(26);
  @$pb.TagNumber(39)
  void clearRelayAccountAction() => $_clearField(39);
  @$pb.TagNumber(39)
  $5.RelayAccountAction ensureRelayAccountAction() => $_ensure(26);

  @$pb.TagNumber(40)
  DaemonStateSyncChunk get daemonStateSyncChunk => $_getN(27);
  @$pb.TagNumber(40)
  set daemonStateSyncChunk(DaemonStateSyncChunk value) => $_setField(40, value);
  @$pb.TagNumber(40)
  $core.bool hasDaemonStateSyncChunk() => $_has(27);
  @$pb.TagNumber(40)
  void clearDaemonStateSyncChunk() => $_clearField(40);
  @$pb.TagNumber(40)
  DaemonStateSyncChunk ensureDaemonStateSyncChunk() => $_ensure(27);

  @$pb.TagNumber(41)
  DaemonStateSyncEnd get daemonStateSyncEnd => $_getN(28);
  @$pb.TagNumber(41)
  set daemonStateSyncEnd(DaemonStateSyncEnd value) => $_setField(41, value);
  @$pb.TagNumber(41)
  $core.bool hasDaemonStateSyncEnd() => $_has(28);
  @$pb.TagNumber(41)
  void clearDaemonStateSyncEnd() => $_clearField(41);
  @$pb.TagNumber(41)
  DaemonStateSyncEnd ensureDaemonStateSyncEnd() => $_ensure(28);
}

/// EdgeControl 是 Edge 与 Controller 之间唯一长连接控制流。
class EdgeControlApi {
  final $pb.RpcClient _client;

  EdgeControlApi(this._client);

  $async.Future<ControllerCommand> connect(
          $pb.ClientContext? ctx, EdgeEvent request) =>
      _client.invoke<ControllerCommand>(
          ctx, 'EdgeControl', 'Connect', request, ControllerCommand());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
