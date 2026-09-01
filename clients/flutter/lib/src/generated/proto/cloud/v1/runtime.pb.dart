// This is a generated file - do not edit.
//
// Generated from cloud/v1/runtime.proto.

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

import 'runtime.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'runtime.pbenum.dart';

/// AgentPresence 是 Edge 内存中一个已认证 daemon 连接的实时投影。
/// R2 只由测试事件生成；R4 才由真实 AgentGateway 写入。
class AgentPresence extends $pb.GeneratedMessage {
  factory AgentPresence({
    $core.String? daemonId,
    $core.String? accountId,
    $core.String? bootId,
    $core.String? connectionId,
    $fixnum.Int64? generation,
    $core.String? bindingId,
    $0.Timestamp? bindingIssuedAt,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (accountId != null) result.accountId = accountId;
    if (bootId != null) result.bootId = bootId;
    if (connectionId != null) result.connectionId = connectionId;
    if (generation != null) result.generation = generation;
    if (bindingId != null) result.bindingId = bindingId;
    if (bindingIssuedAt != null) result.bindingIssuedAt = bindingIssuedAt;
    return result;
  }

  AgentPresence._();

  factory AgentPresence.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentPresence.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentPresence',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'bootId')
    ..aOS(4, _omitFieldNames ? '' : 'connectionId')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'bindingId')
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'bindingIssuedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentPresence clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentPresence copyWith(void Function(AgentPresence) updates) =>
      super.copyWith((message) => updates(message as AgentPresence))
          as AgentPresence;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentPresence create() => AgentPresence._();
  @$core.override
  AgentPresence createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentPresence getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentPresence>(create);
  static AgentPresence? _defaultInstance;

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
  $core.String get bootId => $_getSZ(2);
  @$pb.TagNumber(3)
  set bootId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBootId() => $_has(2);
  @$pb.TagNumber(3)
  void clearBootId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get connectionId => $_getSZ(3);
  @$pb.TagNumber(4)
  set connectionId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasConnectionId() => $_has(3);
  @$pb.TagNumber(4)
  void clearConnectionId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get generation => $_getI64(4);
  @$pb.TagNumber(5)
  set generation($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGeneration() => $_has(4);
  @$pb.TagNumber(5)
  void clearGeneration() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get bindingId => $_getSZ(5);
  @$pb.TagNumber(6)
  set bindingId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasBindingId() => $_has(5);
  @$pb.TagNumber(6)
  void clearBindingId() => $_clearField(6);

  @$pb.TagNumber(7)
  $0.Timestamp get bindingIssuedAt => $_getN(6);
  @$pb.TagNumber(7)
  set bindingIssuedAt($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasBindingIssuedAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearBindingIssuedAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensureBindingIssuedAt() => $_ensure(6);
}

/// ClientSessionSummary 是 Edge 内存中一个客户端信令会话的实时投影。
/// 该消息不包含 SDP、ICE、CapabilityGrant 或 terminal 业务内容。
class ClientSessionSummary extends $pb.GeneratedMessage {
  factory ClientSessionSummary({
    $core.String? sessionId,
    $core.String? accountId,
    $core.String? daemonId,
    $core.String? clientId,
    ClientProduct? product,
    $fixnum.Int64? generation,
    CloudClientAccessMode? accessMode,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (accountId != null) result.accountId = accountId;
    if (daemonId != null) result.daemonId = daemonId;
    if (clientId != null) result.clientId = clientId;
    if (product != null) result.product = product;
    if (generation != null) result.generation = generation;
    if (accessMode != null) result.accessMode = accessMode;
    return result;
  }

  ClientSessionSummary._();

  factory ClientSessionSummary.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientSessionSummary.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientSessionSummary',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'daemonId')
    ..aOS(4, _omitFieldNames ? '' : 'clientId')
    ..aE<ClientProduct>(5, _omitFieldNames ? '' : 'product',
        enumValues: ClientProduct.values)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<CloudClientAccessMode>(7, _omitFieldNames ? '' : 'accessMode',
        enumValues: CloudClientAccessMode.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSessionSummary clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSessionSummary copyWith(void Function(ClientSessionSummary) updates) =>
      super.copyWith((message) => updates(message as ClientSessionSummary))
          as ClientSessionSummary;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientSessionSummary create() => ClientSessionSummary._();
  @$core.override
  ClientSessionSummary createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientSessionSummary getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientSessionSummary>(create);
  static ClientSessionSummary? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

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
  ClientProduct get product => $_getN(4);
  @$pb.TagNumber(5)
  set product(ClientProduct value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasProduct() => $_has(4);
  @$pb.TagNumber(5)
  void clearProduct() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get generation => $_getI64(5);
  @$pb.TagNumber(6)
  set generation($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasGeneration() => $_has(5);
  @$pb.TagNumber(6)
  void clearGeneration() => $_clearField(6);

  @$pb.TagNumber(7)
  CloudClientAccessMode get accessMode => $_getN(6);
  @$pb.TagNumber(7)
  set accessMode(CloudClientAccessMode value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasAccessMode() => $_has(6);
  @$pb.TagNumber(7)
  void clearAccessMode() => $_clearField(7);
}

/// RuntimeSnapshot 是 Edge 在某个单调 revision 上的一致性运行时投影。
class RuntimeSnapshot extends $pb.GeneratedMessage {
  factory RuntimeSnapshot({
    $fixnum.Int64? revision,
    $core.Iterable<AgentPresence>? agents,
    $core.Iterable<ClientSessionSummary>? sessions,
  }) {
    final result = create();
    if (revision != null) result.revision = revision;
    if (agents != null) result.agents.addAll(agents);
    if (sessions != null) result.sessions.addAll(sessions);
    return result;
  }

  RuntimeSnapshot._();

  factory RuntimeSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeSnapshot',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPM<AgentPresence>(2, _omitFieldNames ? '' : 'agents',
        subBuilder: AgentPresence.create)
    ..pPM<ClientSessionSummary>(3, _omitFieldNames ? '' : 'sessions',
        subBuilder: ClientSessionSummary.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeSnapshot copyWith(void Function(RuntimeSnapshot) updates) =>
      super.copyWith((message) => updates(message as RuntimeSnapshot))
          as RuntimeSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeSnapshot create() => RuntimeSnapshot._();
  @$core.override
  RuntimeSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeSnapshot>(create);
  static RuntimeSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get revision => $_getI64(0);
  @$pb.TagNumber(1)
  set revision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevision() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<AgentPresence> get agents => $_getList(1);

  @$pb.TagNumber(3)
  $pb.PbList<ClientSessionSummary> get sessions => $_getList(2);
}

class AgentRemoved extends $pb.GeneratedMessage {
  factory AgentRemoved({
    $core.String? daemonId,
    $fixnum.Int64? generation,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (generation != null) result.generation = generation;
    return result;
  }

  AgentRemoved._();

  factory AgentRemoved.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AgentRemoved.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AgentRemoved',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentRemoved clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AgentRemoved copyWith(void Function(AgentRemoved) updates) =>
      super.copyWith((message) => updates(message as AgentRemoved))
          as AgentRemoved;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AgentRemoved create() => AgentRemoved._();
  @$core.override
  AgentRemoved createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AgentRemoved getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AgentRemoved>(create);
  static AgentRemoved? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get generation => $_getI64(1);
  @$pb.TagNumber(2)
  set generation($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearGeneration() => $_clearField(2);
}

class ClientSessionRemoved extends $pb.GeneratedMessage {
  factory ClientSessionRemoved({
    $core.String? sessionId,
    $fixnum.Int64? generation,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (generation != null) result.generation = generation;
    return result;
  }

  ClientSessionRemoved._();

  factory ClientSessionRemoved.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientSessionRemoved.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientSessionRemoved',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSessionRemoved clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientSessionRemoved copyWith(void Function(ClientSessionRemoved) updates) =>
      super.copyWith((message) => updates(message as ClientSessionRemoved))
          as ClientSessionRemoved;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientSessionRemoved create() => ClientSessionRemoved._();
  @$core.override
  ClientSessionRemoved createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientSessionRemoved getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientSessionRemoved>(create);
  static ClientSessionRemoved? _defaultInstance;

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
}

enum RuntimeDelta_Change {
  agentUpserted,
  agentRemoved,
  sessionUpserted,
  sessionRemoved,
  notSet
}

/// RuntimeDelta 是快照 revision 之后严格连续的一次运行时变更。
class RuntimeDelta extends $pb.GeneratedMessage {
  factory RuntimeDelta({
    $fixnum.Int64? revision,
    AgentPresence? agentUpserted,
    AgentRemoved? agentRemoved,
    ClientSessionSummary? sessionUpserted,
    ClientSessionRemoved? sessionRemoved,
  }) {
    final result = create();
    if (revision != null) result.revision = revision;
    if (agentUpserted != null) result.agentUpserted = agentUpserted;
    if (agentRemoved != null) result.agentRemoved = agentRemoved;
    if (sessionUpserted != null) result.sessionUpserted = sessionUpserted;
    if (sessionRemoved != null) result.sessionRemoved = sessionRemoved;
    return result;
  }

  RuntimeDelta._();

  factory RuntimeDelta.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeDelta.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, RuntimeDelta_Change>
      _RuntimeDelta_ChangeByTag = {
    10: RuntimeDelta_Change.agentUpserted,
    11: RuntimeDelta_Change.agentRemoved,
    12: RuntimeDelta_Change.sessionUpserted,
    13: RuntimeDelta_Change.sessionRemoved,
    0: RuntimeDelta_Change.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeDelta',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 12, 13])
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<AgentPresence>(10, _omitFieldNames ? '' : 'agentUpserted',
        subBuilder: AgentPresence.create)
    ..aOM<AgentRemoved>(11, _omitFieldNames ? '' : 'agentRemoved',
        subBuilder: AgentRemoved.create)
    ..aOM<ClientSessionSummary>(12, _omitFieldNames ? '' : 'sessionUpserted',
        subBuilder: ClientSessionSummary.create)
    ..aOM<ClientSessionRemoved>(13, _omitFieldNames ? '' : 'sessionRemoved',
        subBuilder: ClientSessionRemoved.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeDelta clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeDelta copyWith(void Function(RuntimeDelta) updates) =>
      super.copyWith((message) => updates(message as RuntimeDelta))
          as RuntimeDelta;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeDelta create() => RuntimeDelta._();
  @$core.override
  RuntimeDelta createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeDelta getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeDelta>(create);
  static RuntimeDelta? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  RuntimeDelta_Change whichChange() =>
      _RuntimeDelta_ChangeByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  void clearChange() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $fixnum.Int64 get revision => $_getI64(0);
  @$pb.TagNumber(1)
  set revision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevision() => $_clearField(1);

  @$pb.TagNumber(10)
  AgentPresence get agentUpserted => $_getN(1);
  @$pb.TagNumber(10)
  set agentUpserted(AgentPresence value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasAgentUpserted() => $_has(1);
  @$pb.TagNumber(10)
  void clearAgentUpserted() => $_clearField(10);
  @$pb.TagNumber(10)
  AgentPresence ensureAgentUpserted() => $_ensure(1);

  @$pb.TagNumber(11)
  AgentRemoved get agentRemoved => $_getN(2);
  @$pb.TagNumber(11)
  set agentRemoved(AgentRemoved value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasAgentRemoved() => $_has(2);
  @$pb.TagNumber(11)
  void clearAgentRemoved() => $_clearField(11);
  @$pb.TagNumber(11)
  AgentRemoved ensureAgentRemoved() => $_ensure(2);

  @$pb.TagNumber(12)
  ClientSessionSummary get sessionUpserted => $_getN(3);
  @$pb.TagNumber(12)
  set sessionUpserted(ClientSessionSummary value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasSessionUpserted() => $_has(3);
  @$pb.TagNumber(12)
  void clearSessionUpserted() => $_clearField(12);
  @$pb.TagNumber(12)
  ClientSessionSummary ensureSessionUpserted() => $_ensure(3);

  @$pb.TagNumber(13)
  ClientSessionRemoved get sessionRemoved => $_getN(4);
  @$pb.TagNumber(13)
  set sessionRemoved(ClientSessionRemoved value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasSessionRemoved() => $_has(4);
  @$pb.TagNumber(13)
  void clearSessionRemoved() => $_clearField(13);
  @$pb.TagNumber(13)
  ClientSessionRemoved ensureSessionRemoved() => $_ensure(4);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
