// This is a generated file - do not edit.
//
// Generated from cloud/v1/edge_config.proto.

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

import 'certificate.pb.dart' as $1;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// EdgeDesiredConfig 是 Controller 持久化并签名下发的节点部署意图。
class EdgeDesiredConfig extends $pb.GeneratedMessage {
  factory EdgeDesiredConfig({
    $core.String? edgeId,
    $fixnum.Int64? version,
    $core.String? name,
    $core.String? region,
    $fixnum.Int64? capacity,
    $core.String? publicEndpoint,
    $core.bool? enabled,
  }) {
    final result = EdgeDesiredConfig._();
    if (edgeId != null) result.edgeId = edgeId;
    if (version != null) result.version = version;
    if (name != null) result.name = name;
    if (region != null) result.region = region;
    if (capacity != null) result.capacity = capacity;
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  EdgeDesiredConfig._();

  factory EdgeDesiredConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      EdgeDesiredConfig()..mergeFromBuffer(data, registry);
  factory EdgeDesiredConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      EdgeDesiredConfig()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeDesiredConfig',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: EdgeDesiredConfig.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'name')
    ..aOS(4, _omitFieldNames ? '' : 'region')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'capacity', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'publicEndpoint')
    ..aOB(7, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeDesiredConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeDesiredConfig copyWith(void Function(EdgeDesiredConfig) updates) =>
      super.copyWith((message) => updates(message as EdgeDesiredConfig))
          as EdgeDesiredConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use EdgeDesiredConfig() / EdgeDesiredConfig.new instead')
  static EdgeDesiredConfig create() => EdgeDesiredConfig._();
  static $pb.GeneratedMessage $_createMessage() => EdgeDesiredConfig._();
  @$core.override
  EdgeDesiredConfig createEmptyInstance() => EdgeDesiredConfig._();
  @$core.pragma('dart2js:noInline')
  static EdgeDesiredConfig getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EdgeDesiredConfig>(
          EdgeDesiredConfig.$_createMessage);
  static EdgeDesiredConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get version => $_getI64(1);
  @$pb.TagNumber(2)
  set version($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get name => $_getSZ(2);
  @$pb.TagNumber(3)
  set name($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasName() => $_has(2);
  @$pb.TagNumber(3)
  void clearName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get region => $_getSZ(3);
  @$pb.TagNumber(4)
  set region($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRegion() => $_has(3);
  @$pb.TagNumber(4)
  void clearRegion() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get capacity => $_getI64(4);
  @$pb.TagNumber(5)
  set capacity($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCapacity() => $_has(4);
  @$pb.TagNumber(5)
  void clearCapacity() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get publicEndpoint => $_getSZ(5);
  @$pb.TagNumber(6)
  set publicEndpoint($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPublicEndpoint() => $_has(5);
  @$pb.TagNumber(6)
  void clearPublicEndpoint() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get enabled => $_getBF(6);
  @$pb.TagNumber(7)
  set enabled($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasEnabled() => $_has(6);
  @$pb.TagNumber(7)
  void clearEnabled() => $_clearField(7);
}

/// SignedEdgeDesiredConfig 使用独立签名域保护配置，Edge 只持有公开验签密钥。
class SignedEdgeDesiredConfig extends $pb.GeneratedMessage {
  factory SignedEdgeDesiredConfig({
    $core.String? keyId,
    $core.List<$core.int>? payload,
    $core.List<$core.int>? signature,
  }) {
    final result = SignedEdgeDesiredConfig._();
    if (keyId != null) result.keyId = keyId;
    if (payload != null) result.payload = payload;
    if (signature != null) result.signature = signature;
    return result;
  }

  SignedEdgeDesiredConfig._();

  factory SignedEdgeDesiredConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SignedEdgeDesiredConfig()..mergeFromBuffer(data, registry);
  factory SignedEdgeDesiredConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SignedEdgeDesiredConfig()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SignedEdgeDesiredConfig',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: SignedEdgeDesiredConfig.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'keyId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'payload', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignedEdgeDesiredConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignedEdgeDesiredConfig copyWith(
          void Function(SignedEdgeDesiredConfig) updates) =>
      super.copyWith((message) => updates(message as SignedEdgeDesiredConfig))
          as SignedEdgeDesiredConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use SignedEdgeDesiredConfig() / SignedEdgeDesiredConfig.new instead')
  static SignedEdgeDesiredConfig create() => SignedEdgeDesiredConfig._();
  static $pb.GeneratedMessage $_createMessage() => SignedEdgeDesiredConfig._();
  @$core.override
  SignedEdgeDesiredConfig createEmptyInstance() => SignedEdgeDesiredConfig._();
  @$core.pragma('dart2js:noInline')
  static SignedEdgeDesiredConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SignedEdgeDesiredConfig>(
          SignedEdgeDesiredConfig.$_createMessage);
  static SignedEdgeDesiredConfig? _defaultInstance;

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

/// EdgeRuntimeProjection 来自纯内存 Directory，不得写回 Edge 配置表。
class EdgeRuntimeProjection extends $pb.GeneratedMessage {
  factory EdgeRuntimeProjection({
    $core.bool? online,
    $core.String? bootId,
    $core.String? connectionId,
    $core.String? softwareVersion,
    $fixnum.Int64? runtimeRevision,
    $fixnum.Int64? agentCount,
    $fixnum.Int64? sessionCount,
    $0.Timestamp? connectedAt,
    $0.Timestamp? lastHeartbeat,
  }) {
    final result = EdgeRuntimeProjection._();
    if (online != null) result.online = online;
    if (bootId != null) result.bootId = bootId;
    if (connectionId != null) result.connectionId = connectionId;
    if (softwareVersion != null) result.softwareVersion = softwareVersion;
    if (runtimeRevision != null) result.runtimeRevision = runtimeRevision;
    if (agentCount != null) result.agentCount = agentCount;
    if (sessionCount != null) result.sessionCount = sessionCount;
    if (connectedAt != null) result.connectedAt = connectedAt;
    if (lastHeartbeat != null) result.lastHeartbeat = lastHeartbeat;
    return result;
  }

  EdgeRuntimeProjection._();

  factory EdgeRuntimeProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      EdgeRuntimeProjection()..mergeFromBuffer(data, registry);
  factory EdgeRuntimeProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      EdgeRuntimeProjection()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeRuntimeProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: EdgeRuntimeProjection.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'online')
    ..aOS(2, _omitFieldNames ? '' : 'bootId')
    ..aOS(3, _omitFieldNames ? '' : 'connectionId')
    ..aOS(4, _omitFieldNames ? '' : 'softwareVersion')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'runtimeRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'agentCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'sessionCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(8, _omitFieldNames ? '' : 'connectedAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'lastHeartbeat',
        subBuilder: $0.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeRuntimeProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeRuntimeProjection copyWith(
          void Function(EdgeRuntimeProjection) updates) =>
      super.copyWith((message) => updates(message as EdgeRuntimeProjection))
          as EdgeRuntimeProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use EdgeRuntimeProjection() / EdgeRuntimeProjection.new instead')
  static EdgeRuntimeProjection create() => EdgeRuntimeProjection._();
  static $pb.GeneratedMessage $_createMessage() => EdgeRuntimeProjection._();
  @$core.override
  EdgeRuntimeProjection createEmptyInstance() => EdgeRuntimeProjection._();
  @$core.pragma('dart2js:noInline')
  static EdgeRuntimeProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeRuntimeProjection>(
          EdgeRuntimeProjection.$_createMessage);
  static EdgeRuntimeProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get online => $_getBF(0);
  @$pb.TagNumber(1)
  set online($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOnline() => $_has(0);
  @$pb.TagNumber(1)
  void clearOnline() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get bootId => $_getSZ(1);
  @$pb.TagNumber(2)
  set bootId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBootId() => $_has(1);
  @$pb.TagNumber(2)
  void clearBootId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get connectionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set connectionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasConnectionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearConnectionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get softwareVersion => $_getSZ(3);
  @$pb.TagNumber(4)
  set softwareVersion($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSoftwareVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearSoftwareVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get runtimeRevision => $_getI64(4);
  @$pb.TagNumber(5)
  set runtimeRevision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRuntimeRevision() => $_has(4);
  @$pb.TagNumber(5)
  void clearRuntimeRevision() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get agentCount => $_getI64(5);
  @$pb.TagNumber(6)
  set agentCount($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAgentCount() => $_has(5);
  @$pb.TagNumber(6)
  void clearAgentCount() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get sessionCount => $_getI64(6);
  @$pb.TagNumber(7)
  set sessionCount($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSessionCount() => $_has(6);
  @$pb.TagNumber(7)
  void clearSessionCount() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.Timestamp get connectedAt => $_getN(7);
  @$pb.TagNumber(8)
  set connectedAt($0.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasConnectedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearConnectedAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Timestamp ensureConnectedAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $0.Timestamp get lastHeartbeat => $_getN(8);
  @$pb.TagNumber(9)
  set lastHeartbeat($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasLastHeartbeat() => $_has(8);
  @$pb.TagNumber(9)
  void clearLastHeartbeat() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureLastHeartbeat() => $_ensure(8);
}

/// ManagedEdge 合并持久 desired state 与只读 runtime projection。
class ManagedEdge extends $pb.GeneratedMessage {
  factory ManagedEdge({
    EdgeDesiredConfig? config,
    $fixnum.Int64? configRevision,
    EdgeRuntimeProjection? runtime,
    $1.EdgePublicCertificateStatus? publicCertificate,
  }) {
    final result = ManagedEdge._();
    if (config != null) result.config = config;
    if (configRevision != null) result.configRevision = configRevision;
    if (runtime != null) result.runtime = runtime;
    if (publicCertificate != null) result.publicCertificate = publicCertificate;
    return result;
  }

  ManagedEdge._();

  factory ManagedEdge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ManagedEdge()..mergeFromBuffer(data, registry);
  factory ManagedEdge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ManagedEdge()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ManagedEdge',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ManagedEdge.$_createMessage)
    ..aOM<EdgeDesiredConfig>(1, _omitFieldNames ? '' : 'config',
        subBuilder: EdgeDesiredConfig.$_createMessage)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'configRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<EdgeRuntimeProjection>(3, _omitFieldNames ? '' : 'runtime',
        subBuilder: EdgeRuntimeProjection.$_createMessage)
    ..aOM<$1.EdgePublicCertificateStatus>(
        4, _omitFieldNames ? '' : 'publicCertificate',
        subBuilder: $1.EdgePublicCertificateStatus.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ManagedEdge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ManagedEdge copyWith(void Function(ManagedEdge) updates) =>
      super.copyWith((message) => updates(message as ManagedEdge))
          as ManagedEdge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ManagedEdge() / ManagedEdge.new instead')
  static ManagedEdge create() => ManagedEdge._();
  static $pb.GeneratedMessage $_createMessage() => ManagedEdge._();
  @$core.override
  ManagedEdge createEmptyInstance() => ManagedEdge._();
  @$core.pragma('dart2js:noInline')
  static ManagedEdge getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ManagedEdge>(
          ManagedEdge.$_createMessage);
  static ManagedEdge? _defaultInstance;

  @$pb.TagNumber(1)
  EdgeDesiredConfig get config => $_getN(0);
  @$pb.TagNumber(1)
  set config(EdgeDesiredConfig value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfig() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfig() => $_clearField(1);
  @$pb.TagNumber(1)
  EdgeDesiredConfig ensureConfig() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get configRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set configRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConfigRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfigRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  EdgeRuntimeProjection get runtime => $_getN(2);
  @$pb.TagNumber(3)
  set runtime(EdgeRuntimeProjection value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRuntime() => $_has(2);
  @$pb.TagNumber(3)
  void clearRuntime() => $_clearField(3);
  @$pb.TagNumber(3)
  EdgeRuntimeProjection ensureRuntime() => $_ensure(2);

  @$pb.TagNumber(4)
  $1.EdgePublicCertificateStatus get publicCertificate => $_getN(3);
  @$pb.TagNumber(4)
  set publicCertificate($1.EdgePublicCertificateStatus value) =>
      $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasPublicCertificate() => $_has(3);
  @$pb.TagNumber(4)
  void clearPublicCertificate() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.EdgePublicCertificateStatus ensurePublicCertificate() => $_ensure(3);
}

class ListEdgesRequest extends $pb.GeneratedMessage {
  factory ListEdgesRequest() => ListEdgesRequest._();

  ListEdgesRequest._();

  factory ListEdgesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListEdgesRequest()..mergeFromBuffer(data, registry);
  factory ListEdgesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListEdgesRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListEdgesRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ListEdgesRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListEdgesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListEdgesRequest copyWith(void Function(ListEdgesRequest) updates) =>
      super.copyWith((message) => updates(message as ListEdgesRequest))
          as ListEdgesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ListEdgesRequest() / ListEdgesRequest.new instead')
  static ListEdgesRequest create() => ListEdgesRequest._();
  static $pb.GeneratedMessage $_createMessage() => ListEdgesRequest._();
  @$core.override
  ListEdgesRequest createEmptyInstance() => ListEdgesRequest._();
  @$core.pragma('dart2js:noInline')
  static ListEdgesRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListEdgesRequest>(
          ListEdgesRequest.$_createMessage);
  static ListEdgesRequest? _defaultInstance;
}

class ListEdgesResponse extends $pb.GeneratedMessage {
  factory ListEdgesResponse({
    $core.Iterable<ManagedEdge>? edges,
  }) {
    final result = ListEdgesResponse._();
    if (edges != null) result.edges.addAll(edges);
    return result;
  }

  ListEdgesResponse._();

  factory ListEdgesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListEdgesResponse()..mergeFromBuffer(data, registry);
  factory ListEdgesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListEdgesResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListEdgesResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ListEdgesResponse.$_createMessage)
    ..pPM<ManagedEdge>(1, _omitFieldNames ? '' : 'edges',
        subBuilder: ManagedEdge.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListEdgesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListEdgesResponse copyWith(void Function(ListEdgesResponse) updates) =>
      super.copyWith((message) => updates(message as ListEdgesResponse))
          as ListEdgesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ListEdgesResponse() / ListEdgesResponse.new instead')
  static ListEdgesResponse create() => ListEdgesResponse._();
  static $pb.GeneratedMessage $_createMessage() => ListEdgesResponse._();
  @$core.override
  ListEdgesResponse createEmptyInstance() => ListEdgesResponse._();
  @$core.pragma('dart2js:noInline')
  static ListEdgesResponse getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListEdgesResponse>(
          ListEdgesResponse.$_createMessage);
  static ListEdgesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ManagedEdge> get edges => $_getList(0);
}

class CreateEdgeRequest extends $pb.GeneratedMessage {
  factory CreateEdgeRequest({
    $core.String? name,
    $core.String? region,
    $fixnum.Int64? capacity,
    $core.String? publicEndpoint,
  }) {
    final result = CreateEdgeRequest._();
    if (name != null) result.name = name;
    if (region != null) result.region = region;
    if (capacity != null) result.capacity = capacity;
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    return result;
  }

  CreateEdgeRequest._();

  factory CreateEdgeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeRequest()..mergeFromBuffer(data, registry);
  factory CreateEdgeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateEdgeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: CreateEdgeRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'region')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'capacity', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(4, _omitFieldNames ? '' : 'publicEndpoint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeRequest copyWith(void Function(CreateEdgeRequest) updates) =>
      super.copyWith((message) => updates(message as CreateEdgeRequest))
          as CreateEdgeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CreateEdgeRequest() / CreateEdgeRequest.new instead')
  static CreateEdgeRequest create() => CreateEdgeRequest._();
  static $pb.GeneratedMessage $_createMessage() => CreateEdgeRequest._();
  @$core.override
  CreateEdgeRequest createEmptyInstance() => CreateEdgeRequest._();
  @$core.pragma('dart2js:noInline')
  static CreateEdgeRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CreateEdgeRequest>(
          CreateEdgeRequest.$_createMessage);
  static CreateEdgeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get region => $_getSZ(1);
  @$pb.TagNumber(2)
  set region($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRegion() => $_has(1);
  @$pb.TagNumber(2)
  void clearRegion() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get capacity => $_getI64(2);
  @$pb.TagNumber(3)
  set capacity($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCapacity() => $_has(2);
  @$pb.TagNumber(3)
  void clearCapacity() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get publicEndpoint => $_getSZ(3);
  @$pb.TagNumber(4)
  set publicEndpoint($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPublicEndpoint() => $_has(3);
  @$pb.TagNumber(4)
  void clearPublicEndpoint() => $_clearField(4);
}

class CreateEdgeResponse extends $pb.GeneratedMessage {
  factory CreateEdgeResponse({
    ManagedEdge? edge,
    $core.String? installCommand,
    $0.Timestamp? claimExpiresAt,
  }) {
    final result = CreateEdgeResponse._();
    if (edge != null) result.edge = edge;
    if (installCommand != null) result.installCommand = installCommand;
    if (claimExpiresAt != null) result.claimExpiresAt = claimExpiresAt;
    return result;
  }

  CreateEdgeResponse._();

  factory CreateEdgeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeResponse()..mergeFromBuffer(data, registry);
  factory CreateEdgeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateEdgeResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: CreateEdgeResponse.$_createMessage)
    ..aOM<ManagedEdge>(1, _omitFieldNames ? '' : 'edge',
        subBuilder: ManagedEdge.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'installCommand')
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'claimExpiresAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeResponse copyWith(void Function(CreateEdgeResponse) updates) =>
      super.copyWith((message) => updates(message as CreateEdgeResponse))
          as CreateEdgeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CreateEdgeResponse() / CreateEdgeResponse.new instead')
  static CreateEdgeResponse create() => CreateEdgeResponse._();
  static $pb.GeneratedMessage $_createMessage() => CreateEdgeResponse._();
  @$core.override
  CreateEdgeResponse createEmptyInstance() => CreateEdgeResponse._();
  @$core.pragma('dart2js:noInline')
  static CreateEdgeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateEdgeResponse>(
          CreateEdgeResponse.$_createMessage);
  static CreateEdgeResponse? _defaultInstance;

  @$pb.TagNumber(1)
  ManagedEdge get edge => $_getN(0);
  @$pb.TagNumber(1)
  set edge(ManagedEdge value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEdge() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdge() => $_clearField(1);
  @$pb.TagNumber(1)
  ManagedEdge ensureEdge() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get installCommand => $_getSZ(1);
  @$pb.TagNumber(2)
  set installCommand($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasInstallCommand() => $_has(1);
  @$pb.TagNumber(2)
  void clearInstallCommand() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get claimExpiresAt => $_getN(2);
  @$pb.TagNumber(3)
  set claimExpiresAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasClaimExpiresAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearClaimExpiresAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureClaimExpiresAt() => $_ensure(2);
}

class UpdateEdgeRequest extends $pb.GeneratedMessage {
  factory UpdateEdgeRequest({
    $core.String? edgeId,
    $fixnum.Int64? expectedRevision,
    $core.String? name,
    $core.String? region,
    $fixnum.Int64? capacity,
    $core.String? publicEndpoint,
    $core.bool? enabled,
  }) {
    final result = UpdateEdgeRequest._();
    if (edgeId != null) result.edgeId = edgeId;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (name != null) result.name = name;
    if (region != null) result.region = region;
    if (capacity != null) result.capacity = capacity;
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  UpdateEdgeRequest._();

  factory UpdateEdgeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      UpdateEdgeRequest()..mergeFromBuffer(data, registry);
  factory UpdateEdgeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      UpdateEdgeRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateEdgeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: UpdateEdgeRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'name')
    ..aOS(4, _omitFieldNames ? '' : 'region')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'capacity', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'publicEndpoint')
    ..aOB(7, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateEdgeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateEdgeRequest copyWith(void Function(UpdateEdgeRequest) updates) =>
      super.copyWith((message) => updates(message as UpdateEdgeRequest))
          as UpdateEdgeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use UpdateEdgeRequest() / UpdateEdgeRequest.new instead')
  static UpdateEdgeRequest create() => UpdateEdgeRequest._();
  static $pb.GeneratedMessage $_createMessage() => UpdateEdgeRequest._();
  @$core.override
  UpdateEdgeRequest createEmptyInstance() => UpdateEdgeRequest._();
  @$core.pragma('dart2js:noInline')
  static UpdateEdgeRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<UpdateEdgeRequest>(
          UpdateEdgeRequest.$_createMessage);
  static UpdateEdgeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get expectedRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExpectedRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpectedRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get name => $_getSZ(2);
  @$pb.TagNumber(3)
  set name($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasName() => $_has(2);
  @$pb.TagNumber(3)
  void clearName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get region => $_getSZ(3);
  @$pb.TagNumber(4)
  set region($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRegion() => $_has(3);
  @$pb.TagNumber(4)
  void clearRegion() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get capacity => $_getI64(4);
  @$pb.TagNumber(5)
  set capacity($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCapacity() => $_has(4);
  @$pb.TagNumber(5)
  void clearCapacity() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get publicEndpoint => $_getSZ(5);
  @$pb.TagNumber(6)
  set publicEndpoint($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPublicEndpoint() => $_has(5);
  @$pb.TagNumber(6)
  void clearPublicEndpoint() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get enabled => $_getBF(6);
  @$pb.TagNumber(7)
  set enabled($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasEnabled() => $_has(6);
  @$pb.TagNumber(7)
  void clearEnabled() => $_clearField(7);
}

class UpdateEdgeResponse extends $pb.GeneratedMessage {
  factory UpdateEdgeResponse({
    ManagedEdge? edge,
  }) {
    final result = UpdateEdgeResponse._();
    if (edge != null) result.edge = edge;
    return result;
  }

  UpdateEdgeResponse._();

  factory UpdateEdgeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      UpdateEdgeResponse()..mergeFromBuffer(data, registry);
  factory UpdateEdgeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      UpdateEdgeResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateEdgeResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: UpdateEdgeResponse.$_createMessage)
    ..aOM<ManagedEdge>(1, _omitFieldNames ? '' : 'edge',
        subBuilder: ManagedEdge.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateEdgeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateEdgeResponse copyWith(void Function(UpdateEdgeResponse) updates) =>
      super.copyWith((message) => updates(message as UpdateEdgeResponse))
          as UpdateEdgeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use UpdateEdgeResponse() / UpdateEdgeResponse.new instead')
  static UpdateEdgeResponse create() => UpdateEdgeResponse._();
  static $pb.GeneratedMessage $_createMessage() => UpdateEdgeResponse._();
  @$core.override
  UpdateEdgeResponse createEmptyInstance() => UpdateEdgeResponse._();
  @$core.pragma('dart2js:noInline')
  static UpdateEdgeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UpdateEdgeResponse>(
          UpdateEdgeResponse.$_createMessage);
  static UpdateEdgeResponse? _defaultInstance;

  @$pb.TagNumber(1)
  ManagedEdge get edge => $_getN(0);
  @$pb.TagNumber(1)
  set edge(ManagedEdge value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEdge() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdge() => $_clearField(1);
  @$pb.TagNumber(1)
  ManagedEdge ensureEdge() => $_ensure(0);
}

class RegenerateEdgeInstallRequest extends $pb.GeneratedMessage {
  factory RegenerateEdgeInstallRequest({
    $core.String? edgeId,
  }) {
    final result = RegenerateEdgeInstallRequest._();
    if (edgeId != null) result.edgeId = edgeId;
    return result;
  }

  RegenerateEdgeInstallRequest._();

  factory RegenerateEdgeInstallRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegenerateEdgeInstallRequest()..mergeFromBuffer(data, registry);
  factory RegenerateEdgeInstallRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegenerateEdgeInstallRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegenerateEdgeInstallRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RegenerateEdgeInstallRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegenerateEdgeInstallRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegenerateEdgeInstallRequest copyWith(
          void Function(RegenerateEdgeInstallRequest) updates) =>
      super.copyWith(
              (message) => updates(message as RegenerateEdgeInstallRequest))
          as RegenerateEdgeInstallRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RegenerateEdgeInstallRequest() / RegenerateEdgeInstallRequest.new instead')
  static RegenerateEdgeInstallRequest create() =>
      RegenerateEdgeInstallRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      RegenerateEdgeInstallRequest._();
  @$core.override
  RegenerateEdgeInstallRequest createEmptyInstance() =>
      RegenerateEdgeInstallRequest._();
  @$core.pragma('dart2js:noInline')
  static RegenerateEdgeInstallRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegenerateEdgeInstallRequest>(
          RegenerateEdgeInstallRequest.$_createMessage);
  static RegenerateEdgeInstallRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);
}

class RegenerateEdgeInstallResponse extends $pb.GeneratedMessage {
  factory RegenerateEdgeInstallResponse({
    ManagedEdge? edge,
    $core.String? installCommand,
    $0.Timestamp? claimExpiresAt,
  }) {
    final result = RegenerateEdgeInstallResponse._();
    if (edge != null) result.edge = edge;
    if (installCommand != null) result.installCommand = installCommand;
    if (claimExpiresAt != null) result.claimExpiresAt = claimExpiresAt;
    return result;
  }

  RegenerateEdgeInstallResponse._();

  factory RegenerateEdgeInstallResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegenerateEdgeInstallResponse()..mergeFromBuffer(data, registry);
  factory RegenerateEdgeInstallResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegenerateEdgeInstallResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegenerateEdgeInstallResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RegenerateEdgeInstallResponse.$_createMessage)
    ..aOM<ManagedEdge>(1, _omitFieldNames ? '' : 'edge',
        subBuilder: ManagedEdge.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'installCommand')
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'claimExpiresAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegenerateEdgeInstallResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegenerateEdgeInstallResponse copyWith(
          void Function(RegenerateEdgeInstallResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RegenerateEdgeInstallResponse))
          as RegenerateEdgeInstallResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RegenerateEdgeInstallResponse() / RegenerateEdgeInstallResponse.new instead')
  static RegenerateEdgeInstallResponse create() =>
      RegenerateEdgeInstallResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      RegenerateEdgeInstallResponse._();
  @$core.override
  RegenerateEdgeInstallResponse createEmptyInstance() =>
      RegenerateEdgeInstallResponse._();
  @$core.pragma('dart2js:noInline')
  static RegenerateEdgeInstallResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegenerateEdgeInstallResponse>(
          RegenerateEdgeInstallResponse.$_createMessage);
  static RegenerateEdgeInstallResponse? _defaultInstance;

  @$pb.TagNumber(1)
  ManagedEdge get edge => $_getN(0);
  @$pb.TagNumber(1)
  set edge(ManagedEdge value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEdge() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdge() => $_clearField(1);
  @$pb.TagNumber(1)
  ManagedEdge ensureEdge() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get installCommand => $_getSZ(1);
  @$pb.TagNumber(2)
  set installCommand($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasInstallCommand() => $_has(1);
  @$pb.TagNumber(2)
  void clearInstallCommand() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get claimExpiresAt => $_getN(2);
  @$pb.TagNumber(3)
  set claimExpiresAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasClaimExpiresAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearClaimExpiresAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureClaimExpiresAt() => $_ensure(2);
}

class DeleteEdgeRequest extends $pb.GeneratedMessage {
  factory DeleteEdgeRequest({
    $core.String? edgeId,
    $fixnum.Int64? expectedRevision,
    $core.String? reason,
  }) {
    final result = DeleteEdgeRequest._();
    if (edgeId != null) result.edgeId = edgeId;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (reason != null) result.reason = reason;
    return result;
  }

  DeleteEdgeRequest._();

  factory DeleteEdgeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteEdgeRequest()..mergeFromBuffer(data, registry);
  factory DeleteEdgeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteEdgeRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeleteEdgeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: DeleteEdgeRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteEdgeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteEdgeRequest copyWith(void Function(DeleteEdgeRequest) updates) =>
      super.copyWith((message) => updates(message as DeleteEdgeRequest))
          as DeleteEdgeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use DeleteEdgeRequest() / DeleteEdgeRequest.new instead')
  static DeleteEdgeRequest create() => DeleteEdgeRequest._();
  static $pb.GeneratedMessage $_createMessage() => DeleteEdgeRequest._();
  @$core.override
  DeleteEdgeRequest createEmptyInstance() => DeleteEdgeRequest._();
  @$core.pragma('dart2js:noInline')
  static DeleteEdgeRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DeleteEdgeRequest>(
          DeleteEdgeRequest.$_createMessage);
  static DeleteEdgeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get expectedRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExpectedRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpectedRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get reason => $_getSZ(2);
  @$pb.TagNumber(3)
  set reason($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearReason() => $_clearField(3);
}

class DeleteEdgeResponse extends $pb.GeneratedMessage {
  factory DeleteEdgeResponse() => DeleteEdgeResponse._();

  DeleteEdgeResponse._();

  factory DeleteEdgeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteEdgeResponse()..mergeFromBuffer(data, registry);
  factory DeleteEdgeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteEdgeResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeleteEdgeResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: DeleteEdgeResponse.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteEdgeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteEdgeResponse copyWith(void Function(DeleteEdgeResponse) updates) =>
      super.copyWith((message) => updates(message as DeleteEdgeResponse))
          as DeleteEdgeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use DeleteEdgeResponse() / DeleteEdgeResponse.new instead')
  static DeleteEdgeResponse create() => DeleteEdgeResponse._();
  static $pb.GeneratedMessage $_createMessage() => DeleteEdgeResponse._();
  @$core.override
  DeleteEdgeResponse createEmptyInstance() => DeleteEdgeResponse._();
  @$core.pragma('dart2js:noInline')
  static DeleteEdgeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeleteEdgeResponse>(
          DeleteEdgeResponse.$_createMessage);
  static DeleteEdgeResponse? _defaultInstance;
}

/// RegisterEdgeRequest 由安装后的 Edge 发送；两个私钥都不会离开 Edge。
class RegisterEdgeRequest extends $pb.GeneratedMessage {
  factory RegisterEdgeRequest({
    $core.String? edgeId,
    $core.String? bootstrapToken,
    $core.List<$core.int>? identityCsrPem,
    $core.List<$core.int>? publicCsrPem,
  }) {
    final result = RegisterEdgeRequest._();
    if (edgeId != null) result.edgeId = edgeId;
    if (bootstrapToken != null) result.bootstrapToken = bootstrapToken;
    if (identityCsrPem != null) result.identityCsrPem = identityCsrPem;
    if (publicCsrPem != null) result.publicCsrPem = publicCsrPem;
    return result;
  }

  RegisterEdgeRequest._();

  factory RegisterEdgeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegisterEdgeRequest()..mergeFromBuffer(data, registry);
  factory RegisterEdgeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegisterEdgeRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterEdgeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RegisterEdgeRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..aOS(2, _omitFieldNames ? '' : 'bootstrapToken')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'identityCsrPem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'publicCsrPem', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterEdgeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterEdgeRequest copyWith(void Function(RegisterEdgeRequest) updates) =>
      super.copyWith((message) => updates(message as RegisterEdgeRequest))
          as RegisterEdgeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use RegisterEdgeRequest() / RegisterEdgeRequest.new instead')
  static RegisterEdgeRequest create() => RegisterEdgeRequest._();
  static $pb.GeneratedMessage $_createMessage() => RegisterEdgeRequest._();
  @$core.override
  RegisterEdgeRequest createEmptyInstance() => RegisterEdgeRequest._();
  @$core.pragma('dart2js:noInline')
  static RegisterEdgeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterEdgeRequest>(
          RegisterEdgeRequest.$_createMessage);
  static RegisterEdgeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get bootstrapToken => $_getSZ(1);
  @$pb.TagNumber(2)
  set bootstrapToken($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBootstrapToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearBootstrapToken() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get identityCsrPem => $_getN(2);
  @$pb.TagNumber(3)
  set identityCsrPem($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIdentityCsrPem() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdentityCsrPem() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get publicCsrPem => $_getN(3);
  @$pb.TagNumber(4)
  set publicCsrPem($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPublicCsrPem() => $_has(3);
  @$pb.TagNumber(4)
  void clearPublicCsrPem() => $_clearField(4);
}

/// RegisterEdgeResponse 返回本 Edge 专属证书和已签名配置。
class RegisterEdgeResponse extends $pb.GeneratedMessage {
  factory RegisterEdgeResponse({
    $core.String? edgeId,
    $core.List<$core.int>? identityCertificatePem,
    $core.List<$core.int>? publicCertificatePem,
    $core.List<$core.int>? edgeCaCertificatePem,
    $core.String? controllerAddress,
    $core.String? controllerServerName,
    SignedEdgeDesiredConfig? desiredConfig,
    $core.String? configKeyId,
    $core.List<$core.int>? configSigningPublicKey,
  }) {
    final result = RegisterEdgeResponse._();
    if (edgeId != null) result.edgeId = edgeId;
    if (identityCertificatePem != null)
      result.identityCertificatePem = identityCertificatePem;
    if (publicCertificatePem != null)
      result.publicCertificatePem = publicCertificatePem;
    if (edgeCaCertificatePem != null)
      result.edgeCaCertificatePem = edgeCaCertificatePem;
    if (controllerAddress != null) result.controllerAddress = controllerAddress;
    if (controllerServerName != null)
      result.controllerServerName = controllerServerName;
    if (desiredConfig != null) result.desiredConfig = desiredConfig;
    if (configKeyId != null) result.configKeyId = configKeyId;
    if (configSigningPublicKey != null)
      result.configSigningPublicKey = configSigningPublicKey;
    return result;
  }

  RegisterEdgeResponse._();

  factory RegisterEdgeResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegisterEdgeResponse()..mergeFromBuffer(data, registry);
  factory RegisterEdgeResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RegisterEdgeResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterEdgeResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RegisterEdgeResponse.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'identityCertificatePem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'publicCertificatePem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'edgeCaCertificatePem', $pb.PbFieldType.OY)
    ..aOS(6, _omitFieldNames ? '' : 'controllerAddress')
    ..aOS(7, _omitFieldNames ? '' : 'controllerServerName')
    ..aOM<SignedEdgeDesiredConfig>(8, _omitFieldNames ? '' : 'desiredConfig',
        subBuilder: SignedEdgeDesiredConfig.$_createMessage)
    ..aOS(9, _omitFieldNames ? '' : 'configKeyId')
    ..a<$core.List<$core.int>>(
        10, _omitFieldNames ? '' : 'configSigningPublicKey', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterEdgeResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterEdgeResponse copyWith(void Function(RegisterEdgeResponse) updates) =>
      super.copyWith((message) => updates(message as RegisterEdgeResponse))
          as RegisterEdgeResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RegisterEdgeResponse() / RegisterEdgeResponse.new instead')
  static RegisterEdgeResponse create() => RegisterEdgeResponse._();
  static $pb.GeneratedMessage $_createMessage() => RegisterEdgeResponse._();
  @$core.override
  RegisterEdgeResponse createEmptyInstance() => RegisterEdgeResponse._();
  @$core.pragma('dart2js:noInline')
  static RegisterEdgeResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterEdgeResponse>(
          RegisterEdgeResponse.$_createMessage);
  static RegisterEdgeResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get identityCertificatePem => $_getN(1);
  @$pb.TagNumber(2)
  set identityCertificatePem($core.List<$core.int> value) =>
      $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIdentityCertificatePem() => $_has(1);
  @$pb.TagNumber(2)
  void clearIdentityCertificatePem() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get publicCertificatePem => $_getN(2);
  @$pb.TagNumber(3)
  set publicCertificatePem($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPublicCertificatePem() => $_has(2);
  @$pb.TagNumber(3)
  void clearPublicCertificatePem() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get edgeCaCertificatePem => $_getN(3);
  @$pb.TagNumber(4)
  set edgeCaCertificatePem($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEdgeCaCertificatePem() => $_has(3);
  @$pb.TagNumber(4)
  void clearEdgeCaCertificatePem() => $_clearField(4);

  @$pb.TagNumber(6)
  $core.String get controllerAddress => $_getSZ(4);
  @$pb.TagNumber(6)
  set controllerAddress($core.String value) => $_setString(4, value);
  @$pb.TagNumber(6)
  $core.bool hasControllerAddress() => $_has(4);
  @$pb.TagNumber(6)
  void clearControllerAddress() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get controllerServerName => $_getSZ(5);
  @$pb.TagNumber(7)
  set controllerServerName($core.String value) => $_setString(5, value);
  @$pb.TagNumber(7)
  $core.bool hasControllerServerName() => $_has(5);
  @$pb.TagNumber(7)
  void clearControllerServerName() => $_clearField(7);

  @$pb.TagNumber(8)
  SignedEdgeDesiredConfig get desiredConfig => $_getN(6);
  @$pb.TagNumber(8)
  set desiredConfig(SignedEdgeDesiredConfig value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasDesiredConfig() => $_has(6);
  @$pb.TagNumber(8)
  void clearDesiredConfig() => $_clearField(8);
  @$pb.TagNumber(8)
  SignedEdgeDesiredConfig ensureDesiredConfig() => $_ensure(6);

  @$pb.TagNumber(9)
  $core.String get configKeyId => $_getSZ(7);
  @$pb.TagNumber(9)
  set configKeyId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(9)
  $core.bool hasConfigKeyId() => $_has(7);
  @$pb.TagNumber(9)
  void clearConfigKeyId() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.List<$core.int> get configSigningPublicKey => $_getN(8);
  @$pb.TagNumber(10)
  set configSigningPublicKey($core.List<$core.int> value) =>
      $_setBytes(8, value);
  @$pb.TagNumber(10)
  $core.bool hasConfigSigningPublicKey() => $_has(8);
  @$pb.TagNumber(10)
  void clearConfigSigningPublicKey() => $_clearField(10);
}

class CreateEdgeIdentityRecoveryRequest extends $pb.GeneratedMessage {
  factory CreateEdgeIdentityRecoveryRequest({
    $core.String? edgeId,
    $core.String? reason,
  }) {
    final result = CreateEdgeIdentityRecoveryRequest._();
    if (edgeId != null) result.edgeId = edgeId;
    if (reason != null) result.reason = reason;
    return result;
  }

  CreateEdgeIdentityRecoveryRequest._();

  factory CreateEdgeIdentityRecoveryRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeIdentityRecoveryRequest()..mergeFromBuffer(data, registry);
  factory CreateEdgeIdentityRecoveryRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeIdentityRecoveryRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateEdgeIdentityRecoveryRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: CreateEdgeIdentityRecoveryRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..aOS(2, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeIdentityRecoveryRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeIdentityRecoveryRequest copyWith(
          void Function(CreateEdgeIdentityRecoveryRequest) updates) =>
      super.copyWith((message) =>
              updates(message as CreateEdgeIdentityRecoveryRequest))
          as CreateEdgeIdentityRecoveryRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use CreateEdgeIdentityRecoveryRequest() / CreateEdgeIdentityRecoveryRequest.new instead')
  static CreateEdgeIdentityRecoveryRequest create() =>
      CreateEdgeIdentityRecoveryRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      CreateEdgeIdentityRecoveryRequest._();
  @$core.override
  CreateEdgeIdentityRecoveryRequest createEmptyInstance() =>
      CreateEdgeIdentityRecoveryRequest._();
  @$core.pragma('dart2js:noInline')
  static CreateEdgeIdentityRecoveryRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateEdgeIdentityRecoveryRequest>(
          CreateEdgeIdentityRecoveryRequest.$_createMessage);
  static CreateEdgeIdentityRecoveryRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get reason => $_getSZ(1);
  @$pb.TagNumber(2)
  set reason($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReason() => $_has(1);
  @$pb.TagNumber(2)
  void clearReason() => $_clearField(2);
}

class CreateEdgeIdentityRecoveryResponse extends $pb.GeneratedMessage {
  factory CreateEdgeIdentityRecoveryResponse({
    $core.String? recoveryToken,
    $0.Timestamp? expiresAt,
  }) {
    final result = CreateEdgeIdentityRecoveryResponse._();
    if (recoveryToken != null) result.recoveryToken = recoveryToken;
    if (expiresAt != null) result.expiresAt = expiresAt;
    return result;
  }

  CreateEdgeIdentityRecoveryResponse._();

  factory CreateEdgeIdentityRecoveryResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeIdentityRecoveryResponse()..mergeFromBuffer(data, registry);
  factory CreateEdgeIdentityRecoveryResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateEdgeIdentityRecoveryResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateEdgeIdentityRecoveryResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: CreateEdgeIdentityRecoveryResponse.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'recoveryToken')
    ..aOM<$0.Timestamp>(2, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeIdentityRecoveryResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateEdgeIdentityRecoveryResponse copyWith(
          void Function(CreateEdgeIdentityRecoveryResponse) updates) =>
      super.copyWith((message) =>
              updates(message as CreateEdgeIdentityRecoveryResponse))
          as CreateEdgeIdentityRecoveryResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use CreateEdgeIdentityRecoveryResponse() / CreateEdgeIdentityRecoveryResponse.new instead')
  static CreateEdgeIdentityRecoveryResponse create() =>
      CreateEdgeIdentityRecoveryResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      CreateEdgeIdentityRecoveryResponse._();
  @$core.override
  CreateEdgeIdentityRecoveryResponse createEmptyInstance() =>
      CreateEdgeIdentityRecoveryResponse._();
  @$core.pragma('dart2js:noInline')
  static CreateEdgeIdentityRecoveryResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateEdgeIdentityRecoveryResponse>(
          CreateEdgeIdentityRecoveryResponse.$_createMessage);
  static CreateEdgeIdentityRecoveryResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get recoveryToken => $_getSZ(0);
  @$pb.TagNumber(1)
  set recoveryToken($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRecoveryToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearRecoveryToken() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.Timestamp get expiresAt => $_getN(1);
  @$pb.TagNumber(2)
  set expiresAt($0.Timestamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasExpiresAt() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpiresAt() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.Timestamp ensureExpiresAt() => $_ensure(1);
}

/// RecoverEdgeIdentityRequest is the only unauthenticated-by-mTLS identity path.
/// Its high-entropy token is operator-created, short-lived, one-time, and Edge-bound.
class RecoverEdgeIdentityRequest extends $pb.GeneratedMessage {
  factory RecoverEdgeIdentityRequest({
    $core.String? edgeId,
    $core.String? recoveryToken,
    $core.List<$core.int>? identityCsrPem,
  }) {
    final result = RecoverEdgeIdentityRequest._();
    if (edgeId != null) result.edgeId = edgeId;
    if (recoveryToken != null) result.recoveryToken = recoveryToken;
    if (identityCsrPem != null) result.identityCsrPem = identityCsrPem;
    return result;
  }

  RecoverEdgeIdentityRequest._();

  factory RecoverEdgeIdentityRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RecoverEdgeIdentityRequest()..mergeFromBuffer(data, registry);
  factory RecoverEdgeIdentityRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RecoverEdgeIdentityRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RecoverEdgeIdentityRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RecoverEdgeIdentityRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'edgeId')
    ..aOS(2, _omitFieldNames ? '' : 'recoveryToken')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'identityCsrPem', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RecoverEdgeIdentityRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RecoverEdgeIdentityRequest copyWith(
          void Function(RecoverEdgeIdentityRequest) updates) =>
      super.copyWith(
              (message) => updates(message as RecoverEdgeIdentityRequest))
          as RecoverEdgeIdentityRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RecoverEdgeIdentityRequest() / RecoverEdgeIdentityRequest.new instead')
  static RecoverEdgeIdentityRequest create() => RecoverEdgeIdentityRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      RecoverEdgeIdentityRequest._();
  @$core.override
  RecoverEdgeIdentityRequest createEmptyInstance() =>
      RecoverEdgeIdentityRequest._();
  @$core.pragma('dart2js:noInline')
  static RecoverEdgeIdentityRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RecoverEdgeIdentityRequest>(
          RecoverEdgeIdentityRequest.$_createMessage);
  static RecoverEdgeIdentityRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get recoveryToken => $_getSZ(1);
  @$pb.TagNumber(2)
  set recoveryToken($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRecoveryToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearRecoveryToken() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get identityCsrPem => $_getN(2);
  @$pb.TagNumber(3)
  set identityCsrPem($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIdentityCsrPem() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdentityCsrPem() => $_clearField(3);
}

class RecoverEdgeIdentityResponse extends $pb.GeneratedMessage {
  factory RecoverEdgeIdentityResponse({
    $core.List<$core.int>? identityCertificatePem,
    $core.List<$core.int>? certificateSha256,
    $0.Timestamp? notAfter,
  }) {
    final result = RecoverEdgeIdentityResponse._();
    if (identityCertificatePem != null)
      result.identityCertificatePem = identityCertificatePem;
    if (certificateSha256 != null) result.certificateSha256 = certificateSha256;
    if (notAfter != null) result.notAfter = notAfter;
    return result;
  }

  RecoverEdgeIdentityResponse._();

  factory RecoverEdgeIdentityResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RecoverEdgeIdentityResponse()..mergeFromBuffer(data, registry);
  factory RecoverEdgeIdentityResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RecoverEdgeIdentityResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RecoverEdgeIdentityResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RecoverEdgeIdentityResponse.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'identityCertificatePem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'certificateSha256', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'notAfter',
        subBuilder: $0.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RecoverEdgeIdentityResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RecoverEdgeIdentityResponse copyWith(
          void Function(RecoverEdgeIdentityResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RecoverEdgeIdentityResponse))
          as RecoverEdgeIdentityResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RecoverEdgeIdentityResponse() / RecoverEdgeIdentityResponse.new instead')
  static RecoverEdgeIdentityResponse create() =>
      RecoverEdgeIdentityResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      RecoverEdgeIdentityResponse._();
  @$core.override
  RecoverEdgeIdentityResponse createEmptyInstance() =>
      RecoverEdgeIdentityResponse._();
  @$core.pragma('dart2js:noInline')
  static RecoverEdgeIdentityResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RecoverEdgeIdentityResponse>(
          RecoverEdgeIdentityResponse.$_createMessage);
  static RecoverEdgeIdentityResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get identityCertificatePem => $_getN(0);
  @$pb.TagNumber(1)
  set identityCertificatePem($core.List<$core.int> value) =>
      $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentityCertificatePem() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentityCertificatePem() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get certificateSha256 => $_getN(1);
  @$pb.TagNumber(2)
  set certificateSha256($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCertificateSha256() => $_has(1);
  @$pb.TagNumber(2)
  void clearCertificateSha256() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get notAfter => $_getN(2);
  @$pb.TagNumber(3)
  set notAfter($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasNotAfter() => $_has(2);
  @$pb.TagNumber(3)
  void clearNotAfter() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureNotAfter() => $_ensure(2);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
