// This is a generated file - do not edit.
//
// Generated from bindingpb/client_binding.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import '../apipb/application.pb.dart' as $2;
import '../apipb/common.pb.dart' as $0;
import '../remoteauthpb/remote_auth.pb.dart' as $1;
import 'client_binding.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'client_binding.pbenum.dart';

/// ConnectionSnapshot 是同一个 Go-owned ReadySession 的只读诊断投影。
/// 它只携带 selected candidate pair 的 IP/port，禁止携带 SDP、credential、账号或 terminal 数据。
class ConnectionSnapshot extends $pb.GeneratedMessage {
  factory ConnectionSnapshot({
    $core.String? routeId,
    ConnectionRouteKind? routeKind,
    ConnectionObservedPath? observedPath,
    $core.String? selectionReason,
    $fixnum.Int64? sampledAtUnixNano,
    $fixnum.Int64? roundTripNanos,
    ConnectionCandidateType? localCandidateType,
    ConnectionCandidateType? remoteCandidateType,
    ConnectionTransport? localProtocol,
    ConnectionTransport? remoteProtocol,
    ConnectionTransport? relayTransport,
    $core.String? networkClass,
    $fixnum.Int64? bytesSent,
    $fixnum.Int64? bytesReceived,
    $fixnum.Int64? packetsSent,
    $fixnum.Int64? lossEvents,
    $core.bool? connected,
    $core.String? localIp,
    $core.String? remoteIp,
    $core.int? localPort,
    $core.int? remotePort,
    $core.String? candidatePairId,
    $core.String? localRelatedIp,
    $core.int? localRelatedPort,
    $core.String? remoteRelatedIp,
    $core.int? remoteRelatedPort,
  }) {
    final result = create();
    if (routeId != null) result.routeId = routeId;
    if (routeKind != null) result.routeKind = routeKind;
    if (observedPath != null) result.observedPath = observedPath;
    if (selectionReason != null) result.selectionReason = selectionReason;
    if (sampledAtUnixNano != null) result.sampledAtUnixNano = sampledAtUnixNano;
    if (roundTripNanos != null) result.roundTripNanos = roundTripNanos;
    if (localCandidateType != null)
      result.localCandidateType = localCandidateType;
    if (remoteCandidateType != null)
      result.remoteCandidateType = remoteCandidateType;
    if (localProtocol != null) result.localProtocol = localProtocol;
    if (remoteProtocol != null) result.remoteProtocol = remoteProtocol;
    if (relayTransport != null) result.relayTransport = relayTransport;
    if (networkClass != null) result.networkClass = networkClass;
    if (bytesSent != null) result.bytesSent = bytesSent;
    if (bytesReceived != null) result.bytesReceived = bytesReceived;
    if (packetsSent != null) result.packetsSent = packetsSent;
    if (lossEvents != null) result.lossEvents = lossEvents;
    if (connected != null) result.connected = connected;
    if (localIp != null) result.localIp = localIp;
    if (remoteIp != null) result.remoteIp = remoteIp;
    if (localPort != null) result.localPort = localPort;
    if (remotePort != null) result.remotePort = remotePort;
    if (candidatePairId != null) result.candidatePairId = candidatePairId;
    if (localRelatedIp != null) result.localRelatedIp = localRelatedIp;
    if (localRelatedPort != null) result.localRelatedPort = localRelatedPort;
    if (remoteRelatedIp != null) result.remoteRelatedIp = remoteRelatedIp;
    if (remoteRelatedPort != null) result.remoteRelatedPort = remoteRelatedPort;
    return result;
  }

  ConnectionSnapshot._();

  factory ConnectionSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionSnapshot',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'routeId')
    ..aE<ConnectionRouteKind>(2, _omitFieldNames ? '' : 'routeKind',
        enumValues: ConnectionRouteKind.values)
    ..aE<ConnectionObservedPath>(3, _omitFieldNames ? '' : 'observedPath',
        enumValues: ConnectionObservedPath.values)
    ..aOS(4, _omitFieldNames ? '' : 'selectionReason')
    ..aInt64(5, _omitFieldNames ? '' : 'sampledAtUnixNano')
    ..aInt64(6, _omitFieldNames ? '' : 'roundTripNanos')
    ..aE<ConnectionCandidateType>(
        7, _omitFieldNames ? '' : 'localCandidateType',
        enumValues: ConnectionCandidateType.values)
    ..aE<ConnectionCandidateType>(
        8, _omitFieldNames ? '' : 'remoteCandidateType',
        enumValues: ConnectionCandidateType.values)
    ..aE<ConnectionTransport>(9, _omitFieldNames ? '' : 'localProtocol',
        enumValues: ConnectionTransport.values)
    ..aE<ConnectionTransport>(10, _omitFieldNames ? '' : 'remoteProtocol',
        enumValues: ConnectionTransport.values)
    ..aE<ConnectionTransport>(11, _omitFieldNames ? '' : 'relayTransport',
        enumValues: ConnectionTransport.values)
    ..aOS(12, _omitFieldNames ? '' : 'networkClass')
    ..a<$fixnum.Int64>(
        13, _omitFieldNames ? '' : 'bytesSent', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        14, _omitFieldNames ? '' : 'bytesReceived', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        15, _omitFieldNames ? '' : 'packetsSent', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        16, _omitFieldNames ? '' : 'lossEvents', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(17, _omitFieldNames ? '' : 'connected')
    ..aOS(18, _omitFieldNames ? '' : 'localIp')
    ..aOS(19, _omitFieldNames ? '' : 'remoteIp')
    ..aI(20, _omitFieldNames ? '' : 'localPort', fieldType: $pb.PbFieldType.OU3)
    ..aI(21, _omitFieldNames ? '' : 'remotePort',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(22, _omitFieldNames ? '' : 'candidatePairId')
    ..aOS(23, _omitFieldNames ? '' : 'localRelatedIp')
    ..aI(24, _omitFieldNames ? '' : 'localRelatedPort',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(25, _omitFieldNames ? '' : 'remoteRelatedIp')
    ..aI(26, _omitFieldNames ? '' : 'remoteRelatedPort',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionSnapshot copyWith(void Function(ConnectionSnapshot) updates) =>
      super.copyWith((message) => updates(message as ConnectionSnapshot))
          as ConnectionSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionSnapshot create() => ConnectionSnapshot._();
  @$core.override
  ConnectionSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionSnapshot>(create);
  static ConnectionSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get routeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set routeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRouteId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRouteId() => $_clearField(1);

  @$pb.TagNumber(2)
  ConnectionRouteKind get routeKind => $_getN(1);
  @$pb.TagNumber(2)
  set routeKind(ConnectionRouteKind value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRouteKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearRouteKind() => $_clearField(2);

  @$pb.TagNumber(3)
  ConnectionObservedPath get observedPath => $_getN(2);
  @$pb.TagNumber(3)
  set observedPath(ConnectionObservedPath value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasObservedPath() => $_has(2);
  @$pb.TagNumber(3)
  void clearObservedPath() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get selectionReason => $_getSZ(3);
  @$pb.TagNumber(4)
  set selectionReason($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSelectionReason() => $_has(3);
  @$pb.TagNumber(4)
  void clearSelectionReason() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get sampledAtUnixNano => $_getI64(4);
  @$pb.TagNumber(5)
  set sampledAtUnixNano($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSampledAtUnixNano() => $_has(4);
  @$pb.TagNumber(5)
  void clearSampledAtUnixNano() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get roundTripNanos => $_getI64(5);
  @$pb.TagNumber(6)
  set roundTripNanos($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRoundTripNanos() => $_has(5);
  @$pb.TagNumber(6)
  void clearRoundTripNanos() => $_clearField(6);

  @$pb.TagNumber(7)
  ConnectionCandidateType get localCandidateType => $_getN(6);
  @$pb.TagNumber(7)
  set localCandidateType(ConnectionCandidateType value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasLocalCandidateType() => $_has(6);
  @$pb.TagNumber(7)
  void clearLocalCandidateType() => $_clearField(7);

  @$pb.TagNumber(8)
  ConnectionCandidateType get remoteCandidateType => $_getN(7);
  @$pb.TagNumber(8)
  set remoteCandidateType(ConnectionCandidateType value) =>
      $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasRemoteCandidateType() => $_has(7);
  @$pb.TagNumber(8)
  void clearRemoteCandidateType() => $_clearField(8);

  @$pb.TagNumber(9)
  ConnectionTransport get localProtocol => $_getN(8);
  @$pb.TagNumber(9)
  set localProtocol(ConnectionTransport value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasLocalProtocol() => $_has(8);
  @$pb.TagNumber(9)
  void clearLocalProtocol() => $_clearField(9);

  @$pb.TagNumber(10)
  ConnectionTransport get remoteProtocol => $_getN(9);
  @$pb.TagNumber(10)
  set remoteProtocol(ConnectionTransport value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasRemoteProtocol() => $_has(9);
  @$pb.TagNumber(10)
  void clearRemoteProtocol() => $_clearField(10);

  @$pb.TagNumber(11)
  ConnectionTransport get relayTransport => $_getN(10);
  @$pb.TagNumber(11)
  set relayTransport(ConnectionTransport value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasRelayTransport() => $_has(10);
  @$pb.TagNumber(11)
  void clearRelayTransport() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get networkClass => $_getSZ(11);
  @$pb.TagNumber(12)
  set networkClass($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasNetworkClass() => $_has(11);
  @$pb.TagNumber(12)
  void clearNetworkClass() => $_clearField(12);

  @$pb.TagNumber(13)
  $fixnum.Int64 get bytesSent => $_getI64(12);
  @$pb.TagNumber(13)
  set bytesSent($fixnum.Int64 value) => $_setInt64(12, value);
  @$pb.TagNumber(13)
  $core.bool hasBytesSent() => $_has(12);
  @$pb.TagNumber(13)
  void clearBytesSent() => $_clearField(13);

  @$pb.TagNumber(14)
  $fixnum.Int64 get bytesReceived => $_getI64(13);
  @$pb.TagNumber(14)
  set bytesReceived($fixnum.Int64 value) => $_setInt64(13, value);
  @$pb.TagNumber(14)
  $core.bool hasBytesReceived() => $_has(13);
  @$pb.TagNumber(14)
  void clearBytesReceived() => $_clearField(14);

  @$pb.TagNumber(15)
  $fixnum.Int64 get packetsSent => $_getI64(14);
  @$pb.TagNumber(15)
  set packetsSent($fixnum.Int64 value) => $_setInt64(14, value);
  @$pb.TagNumber(15)
  $core.bool hasPacketsSent() => $_has(14);
  @$pb.TagNumber(15)
  void clearPacketsSent() => $_clearField(15);

  @$pb.TagNumber(16)
  $fixnum.Int64 get lossEvents => $_getI64(15);
  @$pb.TagNumber(16)
  set lossEvents($fixnum.Int64 value) => $_setInt64(15, value);
  @$pb.TagNumber(16)
  $core.bool hasLossEvents() => $_has(15);
  @$pb.TagNumber(16)
  void clearLossEvents() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.bool get connected => $_getBF(16);
  @$pb.TagNumber(17)
  set connected($core.bool value) => $_setBool(16, value);
  @$pb.TagNumber(17)
  $core.bool hasConnected() => $_has(16);
  @$pb.TagNumber(17)
  void clearConnected() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.String get localIp => $_getSZ(17);
  @$pb.TagNumber(18)
  set localIp($core.String value) => $_setString(17, value);
  @$pb.TagNumber(18)
  $core.bool hasLocalIp() => $_has(17);
  @$pb.TagNumber(18)
  void clearLocalIp() => $_clearField(18);

  @$pb.TagNumber(19)
  $core.String get remoteIp => $_getSZ(18);
  @$pb.TagNumber(19)
  set remoteIp($core.String value) => $_setString(18, value);
  @$pb.TagNumber(19)
  $core.bool hasRemoteIp() => $_has(18);
  @$pb.TagNumber(19)
  void clearRemoteIp() => $_clearField(19);

  @$pb.TagNumber(20)
  $core.int get localPort => $_getIZ(19);
  @$pb.TagNumber(20)
  set localPort($core.int value) => $_setUnsignedInt32(19, value);
  @$pb.TagNumber(20)
  $core.bool hasLocalPort() => $_has(19);
  @$pb.TagNumber(20)
  void clearLocalPort() => $_clearField(20);

  @$pb.TagNumber(21)
  $core.int get remotePort => $_getIZ(20);
  @$pb.TagNumber(21)
  set remotePort($core.int value) => $_setUnsignedInt32(20, value);
  @$pb.TagNumber(21)
  $core.bool hasRemotePort() => $_has(20);
  @$pb.TagNumber(21)
  void clearRemotePort() => $_clearField(21);

  @$pb.TagNumber(22)
  $core.String get candidatePairId => $_getSZ(21);
  @$pb.TagNumber(22)
  set candidatePairId($core.String value) => $_setString(21, value);
  @$pb.TagNumber(22)
  $core.bool hasCandidatePairId() => $_has(21);
  @$pb.TagNumber(22)
  void clearCandidatePairId() => $_clearField(22);

  @$pb.TagNumber(23)
  $core.String get localRelatedIp => $_getSZ(22);
  @$pb.TagNumber(23)
  set localRelatedIp($core.String value) => $_setString(22, value);
  @$pb.TagNumber(23)
  $core.bool hasLocalRelatedIp() => $_has(22);
  @$pb.TagNumber(23)
  void clearLocalRelatedIp() => $_clearField(23);

  @$pb.TagNumber(24)
  $core.int get localRelatedPort => $_getIZ(23);
  @$pb.TagNumber(24)
  set localRelatedPort($core.int value) => $_setUnsignedInt32(23, value);
  @$pb.TagNumber(24)
  $core.bool hasLocalRelatedPort() => $_has(23);
  @$pb.TagNumber(24)
  void clearLocalRelatedPort() => $_clearField(24);

  @$pb.TagNumber(25)
  $core.String get remoteRelatedIp => $_getSZ(24);
  @$pb.TagNumber(25)
  set remoteRelatedIp($core.String value) => $_setString(24, value);
  @$pb.TagNumber(25)
  $core.bool hasRemoteRelatedIp() => $_has(24);
  @$pb.TagNumber(25)
  void clearRemoteRelatedIp() => $_clearField(25);

  @$pb.TagNumber(26)
  $core.int get remoteRelatedPort => $_getIZ(25);
  @$pb.TagNumber(26)
  set remoteRelatedPort($core.int value) => $_setUnsignedInt32(25, value);
  @$pb.TagNumber(26)
  $core.bool hasRemoteRelatedPort() => $_has(25);
  @$pb.TagNumber(26)
  void clearRemoteRelatedPort() => $_clearField(26);
}

/// ConnectionPolicy 是 Endpoint selection policy 中允许用户调整的稳定子集。
/// 持久化和 planner validation 都由 Go Client Engine 持有，平台 UI 只能提交用户意图。
class ConnectionPolicy extends $pb.GeneratedMessage {
  factory ConnectionPolicy({
    $1.EndpointRoutePreference? routePreference,
    $1.ManagedWebRTCRelayMode? cloudRelayMode,
    $1.ManagedWebRTCRelayTransport? relayTransport,
  }) {
    final result = create();
    if (routePreference != null) result.routePreference = routePreference;
    if (cloudRelayMode != null) result.cloudRelayMode = cloudRelayMode;
    if (relayTransport != null) result.relayTransport = relayTransport;
    return result;
  }

  ConnectionPolicy._();

  factory ConnectionPolicy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicy',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aE<$1.EndpointRoutePreference>(
        1, _omitFieldNames ? '' : 'routePreference',
        enumValues: $1.EndpointRoutePreference.values)
    ..aE<$1.ManagedWebRTCRelayMode>(2, _omitFieldNames ? '' : 'cloudRelayMode',
        enumValues: $1.ManagedWebRTCRelayMode.values)
    ..aE<$1.ManagedWebRTCRelayTransport>(
        3, _omitFieldNames ? '' : 'relayTransport',
        enumValues: $1.ManagedWebRTCRelayTransport.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicy copyWith(void Function(ConnectionPolicy) updates) =>
      super.copyWith((message) => updates(message as ConnectionPolicy))
          as ConnectionPolicy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicy create() => ConnectionPolicy._();
  @$core.override
  ConnectionPolicy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicy getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicy>(create);
  static ConnectionPolicy? _defaultInstance;

  @$pb.TagNumber(1)
  $1.EndpointRoutePreference get routePreference => $_getN(0);
  @$pb.TagNumber(1)
  set routePreference($1.EndpointRoutePreference value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRoutePreference() => $_has(0);
  @$pb.TagNumber(1)
  void clearRoutePreference() => $_clearField(1);

  /// cloud_relay_mode 约束 managed Cloud Route 使用 P2P、Relay 或自动选择；持久真值在 Go Endpoint registry。
  @$pb.TagNumber(2)
  $1.ManagedWebRTCRelayMode get cloudRelayMode => $_getN(1);
  @$pb.TagNumber(2)
  set cloudRelayMode($1.ManagedWebRTCRelayMode value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCloudRelayMode() => $_has(1);
  @$pb.TagNumber(2)
  void clearCloudRelayMode() => $_clearField(2);

  /// relay_transport 只在 Cloud Relay 可用时收缩 TURN UDP/TCP；UI 不得把它保存成第二份连接状态。
  @$pb.TagNumber(3)
  $1.ManagedWebRTCRelayTransport get relayTransport => $_getN(2);
  @$pb.TagNumber(3)
  set relayTransport($1.ManagedWebRTCRelayTransport value) =>
      $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRelayTransport() => $_has(2);
  @$pb.TagNumber(3)
  void clearRelayTransport() => $_clearField(3);
}

class ConnectionPolicyRouteAvailability extends $pb.GeneratedMessage {
  factory ConnectionPolicyRouteAvailability({
    ConnectionRouteKind? routeKind,
    $core.bool? available,
    ConnectionPolicyAvailabilityReason? reason,
  }) {
    final result = create();
    if (routeKind != null) result.routeKind = routeKind;
    if (available != null) result.available = available;
    if (reason != null) result.reason = reason;
    return result;
  }

  ConnectionPolicyRouteAvailability._();

  factory ConnectionPolicyRouteAvailability.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicyRouteAvailability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicyRouteAvailability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aE<ConnectionRouteKind>(1, _omitFieldNames ? '' : 'routeKind',
        enumValues: ConnectionRouteKind.values)
    ..aOB(2, _omitFieldNames ? '' : 'available')
    ..aE<ConnectionPolicyAvailabilityReason>(3, _omitFieldNames ? '' : 'reason',
        enumValues: ConnectionPolicyAvailabilityReason.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyRouteAvailability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyRouteAvailability copyWith(
          void Function(ConnectionPolicyRouteAvailability) updates) =>
      super.copyWith((message) =>
              updates(message as ConnectionPolicyRouteAvailability))
          as ConnectionPolicyRouteAvailability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyRouteAvailability create() =>
      ConnectionPolicyRouteAvailability._();
  @$core.override
  ConnectionPolicyRouteAvailability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyRouteAvailability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicyRouteAvailability>(
          create);
  static ConnectionPolicyRouteAvailability? _defaultInstance;

  @$pb.TagNumber(1)
  ConnectionRouteKind get routeKind => $_getN(0);
  @$pb.TagNumber(1)
  set routeKind(ConnectionRouteKind value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRouteKind() => $_has(0);
  @$pb.TagNumber(1)
  void clearRouteKind() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get available => $_getBF(1);
  @$pb.TagNumber(2)
  set available($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAvailable() => $_has(1);
  @$pb.TagNumber(2)
  void clearAvailable() => $_clearField(2);

  @$pb.TagNumber(3)
  ConnectionPolicyAvailabilityReason get reason => $_getN(2);
  @$pb.TagNumber(3)
  set reason(ConnectionPolicyAvailabilityReason value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearReason() => $_clearField(3);
}

class ConnectionPolicyState extends $pb.GeneratedMessage {
  factory ConnectionPolicyState({
    ConnectionPolicy? policy,
    $core.Iterable<ConnectionPolicyRouteAvailability>? routes,
  }) {
    final result = create();
    if (policy != null) result.policy = policy;
    if (routes != null) result.routes.addAll(routes);
    return result;
  }

  ConnectionPolicyState._();

  factory ConnectionPolicyState.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicyState.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicyState',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOM<ConnectionPolicy>(1, _omitFieldNames ? '' : 'policy',
        subBuilder: ConnectionPolicy.create)
    ..pPM<ConnectionPolicyRouteAvailability>(2, _omitFieldNames ? '' : 'routes',
        subBuilder: ConnectionPolicyRouteAvailability.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyState clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyState copyWith(
          void Function(ConnectionPolicyState) updates) =>
      super.copyWith((message) => updates(message as ConnectionPolicyState))
          as ConnectionPolicyState;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyState create() => ConnectionPolicyState._();
  @$core.override
  ConnectionPolicyState createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyState getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicyState>(create);
  static ConnectionPolicyState? _defaultInstance;

  @$pb.TagNumber(1)
  ConnectionPolicy get policy => $_getN(0);
  @$pb.TagNumber(1)
  set policy(ConnectionPolicy value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPolicy() => $_has(0);
  @$pb.TagNumber(1)
  void clearPolicy() => $_clearField(1);
  @$pb.TagNumber(1)
  ConnectionPolicy ensurePolicy() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<ConnectionPolicyRouteAvailability> get routes => $_getList(1);
}

class ConnectionPolicyGetRequest extends $pb.GeneratedMessage {
  factory ConnectionPolicyGetRequest({
    $core.String? requestId,
    $core.String? endpointId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    return result;
  }

  ConnectionPolicyGetRequest._();

  factory ConnectionPolicyGetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicyGetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicyGetRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyGetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyGetRequest copyWith(
          void Function(ConnectionPolicyGetRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ConnectionPolicyGetRequest))
          as ConnectionPolicyGetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyGetRequest create() => ConnectionPolicyGetRequest._();
  @$core.override
  ConnectionPolicyGetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyGetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicyGetRequest>(create);
  static ConnectionPolicyGetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);
}

class ConnectionPolicyGetResult extends $pb.GeneratedMessage {
  factory ConnectionPolicyGetResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    ConnectionPolicyState? state,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (state != null) result.state = state;
    if (error != null) result.error = error;
    return result;
  }

  ConnectionPolicyGetResult._();

  factory ConnectionPolicyGetResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicyGetResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicyGetResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ConnectionPolicyState>(3, _omitFieldNames ? '' : 'state',
        subBuilder: ConnectionPolicyState.create)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyGetResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyGetResult copyWith(
          void Function(ConnectionPolicyGetResult) updates) =>
      super.copyWith((message) => updates(message as ConnectionPolicyGetResult))
          as ConnectionPolicyGetResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyGetResult create() => ConnectionPolicyGetResult._();
  @$core.override
  ConnectionPolicyGetResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyGetResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicyGetResult>(create);
  static ConnectionPolicyGetResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  ConnectionPolicyState get state => $_getN(2);
  @$pb.TagNumber(3)
  set state(ConnectionPolicyState value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasState() => $_has(2);
  @$pb.TagNumber(3)
  void clearState() => $_clearField(3);
  @$pb.TagNumber(3)
  ConnectionPolicyState ensureState() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

class ConnectionPolicyApplyRequest extends $pb.GeneratedMessage {
  factory ConnectionPolicyApplyRequest({
    $core.String? requestId,
    $core.String? endpointId,
    ConnectionPolicy? policy,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    if (policy != null) result.policy = policy;
    return result;
  }

  ConnectionPolicyApplyRequest._();

  factory ConnectionPolicyApplyRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicyApplyRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicyApplyRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..aOM<ConnectionPolicy>(3, _omitFieldNames ? '' : 'policy',
        subBuilder: ConnectionPolicy.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyApplyRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyApplyRequest copyWith(
          void Function(ConnectionPolicyApplyRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ConnectionPolicyApplyRequest))
          as ConnectionPolicyApplyRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyApplyRequest create() =>
      ConnectionPolicyApplyRequest._();
  @$core.override
  ConnectionPolicyApplyRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyApplyRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicyApplyRequest>(create);
  static ConnectionPolicyApplyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);

  @$pb.TagNumber(3)
  ConnectionPolicy get policy => $_getN(2);
  @$pb.TagNumber(3)
  set policy(ConnectionPolicy value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasPolicy() => $_has(2);
  @$pb.TagNumber(3)
  void clearPolicy() => $_clearField(3);
  @$pb.TagNumber(3)
  ConnectionPolicy ensurePolicy() => $_ensure(2);
}

class ConnectionPolicyApplyResult extends $pb.GeneratedMessage {
  factory ConnectionPolicyApplyResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    ConnectionPolicyState? state,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (state != null) result.state = state;
    if (error != null) result.error = error;
    return result;
  }

  ConnectionPolicyApplyResult._();

  factory ConnectionPolicyApplyResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionPolicyApplyResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionPolicyApplyResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ConnectionPolicyState>(3, _omitFieldNames ? '' : 'state',
        subBuilder: ConnectionPolicyState.create)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyApplyResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionPolicyApplyResult copyWith(
          void Function(ConnectionPolicyApplyResult) updates) =>
      super.copyWith(
              (message) => updates(message as ConnectionPolicyApplyResult))
          as ConnectionPolicyApplyResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyApplyResult create() =>
      ConnectionPolicyApplyResult._();
  @$core.override
  ConnectionPolicyApplyResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionPolicyApplyResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionPolicyApplyResult>(create);
  static ConnectionPolicyApplyResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  ConnectionPolicyState get state => $_getN(2);
  @$pb.TagNumber(3)
  set state(ConnectionPolicyState value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasState() => $_has(2);
  @$pb.TagNumber(3)
  void clearState() => $_clearField(3);
  @$pb.TagNumber(3)
  ConnectionPolicyState ensureState() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

/// ConnectionSnapshotGetRequest 重新采样当前 session handle 的实际 transport 状态。
/// session_handle 属于同一 Engine generation；stale 或 closing handle 必须显式失败。
class ConnectionSnapshotGetRequest extends $pb.GeneratedMessage {
  factory ConnectionSnapshotGetRequest({
    $core.String? requestId,
    $fixnum.Int64? sessionHandle,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    return result;
  }

  ConnectionSnapshotGetRequest._();

  factory ConnectionSnapshotGetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionSnapshotGetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionSnapshotGetRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionSnapshotGetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionSnapshotGetRequest copyWith(
          void Function(ConnectionSnapshotGetRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ConnectionSnapshotGetRequest))
          as ConnectionSnapshotGetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionSnapshotGetRequest create() =>
      ConnectionSnapshotGetRequest._();
  @$core.override
  ConnectionSnapshotGetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionSnapshotGetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionSnapshotGetRequest>(create);
  static ConnectionSnapshotGetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get sessionHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionHandle() => $_clearField(2);
}

class ConnectionSnapshotGetResult extends $pb.GeneratedMessage {
  factory ConnectionSnapshotGetResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $fixnum.Int64? sessionHandle,
    ConnectionSnapshot? connection,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    if (connection != null) result.connection = connection;
    if (error != null) result.error = error;
    return result;
  }

  ConnectionSnapshotGetResult._();

  factory ConnectionSnapshotGetResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionSnapshotGetResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionSnapshotGetResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ConnectionSnapshot>(4, _omitFieldNames ? '' : 'connection',
        subBuilder: ConnectionSnapshot.create)
    ..aOM<$0.ApiError>(5, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionSnapshotGetResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionSnapshotGetResult copyWith(
          void Function(ConnectionSnapshotGetResult) updates) =>
      super.copyWith(
              (message) => updates(message as ConnectionSnapshotGetResult))
          as ConnectionSnapshotGetResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionSnapshotGetResult create() =>
      ConnectionSnapshotGetResult._();
  @$core.override
  ConnectionSnapshotGetResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionSnapshotGetResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionSnapshotGetResult>(create);
  static ConnectionSnapshotGetResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get sessionHandle => $_getI64(2);
  @$pb.TagNumber(3)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionHandle() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionHandle() => $_clearField(3);

  @$pb.TagNumber(4)
  ConnectionSnapshot get connection => $_getN(3);
  @$pb.TagNumber(4)
  set connection(ConnectionSnapshot value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasConnection() => $_has(3);
  @$pb.TagNumber(4)
  void clearConnection() => $_clearField(4);
  @$pb.TagNumber(4)
  ConnectionSnapshot ensureConnection() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.ApiError get error => $_getN(4);
  @$pb.TagNumber(5)
  set error($0.ApiError value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasError() => $_has(4);
  @$pb.TagNumber(5)
  void clearError() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.ApiError ensureError() => $_ensure(4);
}

/// SessionInvalidateRequest removes the exact Go-owned endpoint generation behind
/// a binding session. It is reserved for confirmed platform network changes; a
/// normal CloseSession still releases only the caller's consumer lease.
class SessionInvalidateRequest extends $pb.GeneratedMessage {
  factory SessionInvalidateRequest({
    $core.String? requestId,
    $fixnum.Int64? sessionHandle,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    return result;
  }

  SessionInvalidateRequest._();

  factory SessionInvalidateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SessionInvalidateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SessionInvalidateRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionInvalidateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionInvalidateRequest copyWith(
          void Function(SessionInvalidateRequest) updates) =>
      super.copyWith((message) => updates(message as SessionInvalidateRequest))
          as SessionInvalidateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SessionInvalidateRequest create() => SessionInvalidateRequest._();
  @$core.override
  SessionInvalidateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SessionInvalidateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SessionInvalidateRequest>(create);
  static SessionInvalidateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get sessionHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionHandle() => $_clearField(2);
}

class SessionInvalidateResult extends $pb.GeneratedMessage {
  factory SessionInvalidateResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $fixnum.Int64? sessionHandle,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    if (error != null) result.error = error;
    return result;
  }

  SessionInvalidateResult._();

  factory SessionInvalidateResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SessionInvalidateResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SessionInvalidateResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionInvalidateResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionInvalidateResult copyWith(
          void Function(SessionInvalidateResult) updates) =>
      super.copyWith((message) => updates(message as SessionInvalidateResult))
          as SessionInvalidateResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SessionInvalidateResult create() => SessionInvalidateResult._();
  @$core.override
  SessionInvalidateResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SessionInvalidateResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SessionInvalidateResult>(create);
  static SessionInvalidateResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get sessionHandle => $_getI64(2);
  @$pb.TagNumber(3)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionHandle() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionHandle() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

/// EndpointDisconnectRequest is the explicit user disconnect operation. Unlike
/// cancelling OpenSession, it waits behind any in-flight endpoint acquisition,
/// fences that generation, and closes the pooled physical session.
class EndpointDisconnectRequest extends $pb.GeneratedMessage {
  factory EndpointDisconnectRequest({
    $core.String? requestId,
    $core.String? endpointId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    return result;
  }

  EndpointDisconnectRequest._();

  factory EndpointDisconnectRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointDisconnectRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointDisconnectRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDisconnectRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDisconnectRequest copyWith(
          void Function(EndpointDisconnectRequest) updates) =>
      super.copyWith((message) => updates(message as EndpointDisconnectRequest))
          as EndpointDisconnectRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointDisconnectRequest create() => EndpointDisconnectRequest._();
  @$core.override
  EndpointDisconnectRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointDisconnectRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointDisconnectRequest>(create);
  static EndpointDisconnectRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);
}

class EndpointDisconnectResult extends $pb.GeneratedMessage {
  factory EndpointDisconnectResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $core.String? endpointId,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpointId != null) result.endpointId = endpointId;
    if (error != null) result.error = error;
    return result;
  }

  EndpointDisconnectResult._();

  factory EndpointDisconnectResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointDisconnectResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointDisconnectResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'endpointId')
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDisconnectResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDisconnectResult copyWith(
          void Function(EndpointDisconnectResult) updates) =>
      super.copyWith((message) => updates(message as EndpointDisconnectResult))
          as EndpointDisconnectResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointDisconnectResult create() => EndpointDisconnectResult._();
  @$core.override
  EndpointDisconnectResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointDisconnectResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointDisconnectResult>(create);
  static EndpointDisconnectResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get endpointId => $_getSZ(2);
  @$pb.TagNumber(3)
  set endpointId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpointId() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpointId() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

/// EndpointCloudPresenceGet asks the endpoint's cached Edge directly whether
/// the paired daemon currently owns an authenticated AgentGateway connection.
class EndpointCloudPresenceGetRequest extends $pb.GeneratedMessage {
  factory EndpointCloudPresenceGetRequest({
    $core.String? requestId,
    $core.String? endpointId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    return result;
  }

  EndpointCloudPresenceGetRequest._();

  factory EndpointCloudPresenceGetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointCloudPresenceGetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointCloudPresenceGetRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointCloudPresenceGetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointCloudPresenceGetRequest copyWith(
          void Function(EndpointCloudPresenceGetRequest) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointCloudPresenceGetRequest))
          as EndpointCloudPresenceGetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointCloudPresenceGetRequest create() =>
      EndpointCloudPresenceGetRequest._();
  @$core.override
  EndpointCloudPresenceGetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointCloudPresenceGetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointCloudPresenceGetRequest>(
          create);
  static EndpointCloudPresenceGetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);
}

class EndpointCloudPresenceGetResult extends $pb.GeneratedMessage {
  factory EndpointCloudPresenceGetResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $core.String? endpointId,
    $core.bool? online,
    $0.ApiError? error,
    $core.String? deviceId,
    $core.String? deviceFingerprint,
    $core.String? daemonId,
    $core.String? edgeId,
    $core.String? edgeName,
    $core.String? edgeRegion,
    $core.String? edgePublicEndpoint,
    $core.String? edgeServerName,
    $core.String? locatorSource,
    $core.bool? refreshedFromController,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpointId != null) result.endpointId = endpointId;
    if (online != null) result.online = online;
    if (error != null) result.error = error;
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    if (daemonId != null) result.daemonId = daemonId;
    if (edgeId != null) result.edgeId = edgeId;
    if (edgeName != null) result.edgeName = edgeName;
    if (edgeRegion != null) result.edgeRegion = edgeRegion;
    if (edgePublicEndpoint != null)
      result.edgePublicEndpoint = edgePublicEndpoint;
    if (edgeServerName != null) result.edgeServerName = edgeServerName;
    if (locatorSource != null) result.locatorSource = locatorSource;
    if (refreshedFromController != null)
      result.refreshedFromController = refreshedFromController;
    return result;
  }

  EndpointCloudPresenceGetResult._();

  factory EndpointCloudPresenceGetResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointCloudPresenceGetResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointCloudPresenceGetResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'endpointId')
    ..aOB(4, _omitFieldNames ? '' : 'online')
    ..aOM<$0.ApiError>(5, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..aOS(6, _omitFieldNames ? '' : 'deviceId')
    ..aOS(7, _omitFieldNames ? '' : 'deviceFingerprint')
    ..aOS(8, _omitFieldNames ? '' : 'daemonId')
    ..aOS(9, _omitFieldNames ? '' : 'edgeId')
    ..aOS(10, _omitFieldNames ? '' : 'edgeName')
    ..aOS(11, _omitFieldNames ? '' : 'edgeRegion')
    ..aOS(12, _omitFieldNames ? '' : 'edgePublicEndpoint')
    ..aOS(13, _omitFieldNames ? '' : 'edgeServerName')
    ..aOS(14, _omitFieldNames ? '' : 'locatorSource')
    ..aOB(15, _omitFieldNames ? '' : 'refreshedFromController')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointCloudPresenceGetResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointCloudPresenceGetResult copyWith(
          void Function(EndpointCloudPresenceGetResult) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointCloudPresenceGetResult))
          as EndpointCloudPresenceGetResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointCloudPresenceGetResult create() =>
      EndpointCloudPresenceGetResult._();
  @$core.override
  EndpointCloudPresenceGetResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointCloudPresenceGetResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointCloudPresenceGetResult>(create);
  static EndpointCloudPresenceGetResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get endpointId => $_getSZ(2);
  @$pb.TagNumber(3)
  set endpointId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpointId() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpointId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get online => $_getBF(3);
  @$pb.TagNumber(4)
  set online($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOnline() => $_has(3);
  @$pb.TagNumber(4)
  void clearOnline() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.ApiError get error => $_getN(4);
  @$pb.TagNumber(5)
  set error($0.ApiError value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasError() => $_has(4);
  @$pb.TagNumber(5)
  void clearError() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.ApiError ensureError() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get deviceId => $_getSZ(5);
  @$pb.TagNumber(6)
  set deviceId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDeviceId() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeviceId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get deviceFingerprint => $_getSZ(6);
  @$pb.TagNumber(7)
  set deviceFingerprint($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasDeviceFingerprint() => $_has(6);
  @$pb.TagNumber(7)
  void clearDeviceFingerprint() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get daemonId => $_getSZ(7);
  @$pb.TagNumber(8)
  set daemonId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasDaemonId() => $_has(7);
  @$pb.TagNumber(8)
  void clearDaemonId() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get edgeId => $_getSZ(8);
  @$pb.TagNumber(9)
  set edgeId($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasEdgeId() => $_has(8);
  @$pb.TagNumber(9)
  void clearEdgeId() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get edgeName => $_getSZ(9);
  @$pb.TagNumber(10)
  set edgeName($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasEdgeName() => $_has(9);
  @$pb.TagNumber(10)
  void clearEdgeName() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get edgeRegion => $_getSZ(10);
  @$pb.TagNumber(11)
  set edgeRegion($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasEdgeRegion() => $_has(10);
  @$pb.TagNumber(11)
  void clearEdgeRegion() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get edgePublicEndpoint => $_getSZ(11);
  @$pb.TagNumber(12)
  set edgePublicEndpoint($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasEdgePublicEndpoint() => $_has(11);
  @$pb.TagNumber(12)
  void clearEdgePublicEndpoint() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get edgeServerName => $_getSZ(12);
  @$pb.TagNumber(13)
  set edgeServerName($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasEdgeServerName() => $_has(12);
  @$pb.TagNumber(13)
  void clearEdgeServerName() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get locatorSource => $_getSZ(13);
  @$pb.TagNumber(14)
  set locatorSource($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasLocatorSource() => $_has(13);
  @$pb.TagNumber(14)
  void clearLocatorSource() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.bool get refreshedFromController => $_getBF(14);
  @$pb.TagNumber(15)
  set refreshedFromController($core.bool value) => $_setBool(14, value);
  @$pb.TagNumber(15)
  $core.bool hasRefreshedFromController() => $_has(14);
  @$pb.TagNumber(15)
  void clearRefreshedFromController() => $_clearField(15);
}

class OpenSessionRequest extends $pb.GeneratedMessage {
  factory OpenSessionRequest({
    $core.String? requestId,
    $core.String? endpointId,
    $core.String? routeOverride,
    ConnectIntent? intent,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    if (routeOverride != null) result.routeOverride = routeOverride;
    if (intent != null) result.intent = intent;
    return result;
  }

  OpenSessionRequest._();

  factory OpenSessionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OpenSessionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OpenSessionRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..aOS(3, _omitFieldNames ? '' : 'routeOverride')
    ..aE<ConnectIntent>(4, _omitFieldNames ? '' : 'intent',
        enumValues: ConnectIntent.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenSessionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenSessionRequest copyWith(void Function(OpenSessionRequest) updates) =>
      super.copyWith((message) => updates(message as OpenSessionRequest))
          as OpenSessionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OpenSessionRequest create() => OpenSessionRequest._();
  @$core.override
  OpenSessionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OpenSessionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OpenSessionRequest>(create);
  static OpenSessionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get routeOverride => $_getSZ(2);
  @$pb.TagNumber(3)
  set routeOverride($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRouteOverride() => $_has(2);
  @$pb.TagNumber(3)
  void clearRouteOverride() => $_clearField(3);

  @$pb.TagNumber(4)
  ConnectIntent get intent => $_getN(3);
  @$pb.TagNumber(4)
  set intent(ConnectIntent value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasIntent() => $_has(3);
  @$pb.TagNumber(4)
  void clearIntent() => $_clearField(4);
}

class ImportPairingRequest extends $pb.GeneratedMessage {
  factory ImportPairingRequest({
    $core.String? requestId,
    $core.String? portablePayload,
    $core.String? expectedEndpointId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (portablePayload != null) result.portablePayload = portablePayload;
    if (expectedEndpointId != null)
      result.expectedEndpointId = expectedEndpointId;
    return result;
  }

  ImportPairingRequest._();

  factory ImportPairingRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ImportPairingRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ImportPairingRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'portablePayload')
    ..aOS(3, _omitFieldNames ? '' : 'expectedEndpointId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImportPairingRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImportPairingRequest copyWith(void Function(ImportPairingRequest) updates) =>
      super.copyWith((message) => updates(message as ImportPairingRequest))
          as ImportPairingRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ImportPairingRequest create() => ImportPairingRequest._();
  @$core.override
  ImportPairingRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ImportPairingRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ImportPairingRequest>(create);
  static ImportPairingRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get portablePayload => $_getSZ(1);
  @$pb.TagNumber(2)
  set portablePayload($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPortablePayload() => $_has(1);
  @$pb.TagNumber(2)
  void clearPortablePayload() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get expectedEndpointId => $_getSZ(2);
  @$pb.TagNumber(3)
  set expectedEndpointId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedEndpointId() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpectedEndpointId() => $_clearField(3);
}

class ImportPairingResult extends $pb.GeneratedMessage {
  factory ImportPairingResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $1.EndpointConfigV1? endpoint,
    $core.String? ticketId,
    $core.String? clientKeyFingerprint,
    $fixnum.Int64? expiresAtUnixNano,
    $core.bool? authorizationRequired,
    $0.ApiError? error,
    $1.EndpointRegistryV1? registry,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpoint != null) result.endpoint = endpoint;
    if (ticketId != null) result.ticketId = ticketId;
    if (clientKeyFingerprint != null)
      result.clientKeyFingerprint = clientKeyFingerprint;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (authorizationRequired != null)
      result.authorizationRequired = authorizationRequired;
    if (error != null) result.error = error;
    if (registry != null) result.registry = registry;
    return result;
  }

  ImportPairingResult._();

  factory ImportPairingResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ImportPairingResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ImportPairingResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$1.EndpointConfigV1>(3, _omitFieldNames ? '' : 'endpoint',
        subBuilder: $1.EndpointConfigV1.create)
    ..aOS(4, _omitFieldNames ? '' : 'ticketId')
    ..aOS(5, _omitFieldNames ? '' : 'clientKeyFingerprint')
    ..aInt64(6, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..aOB(7, _omitFieldNames ? '' : 'authorizationRequired')
    ..aOM<$0.ApiError>(8, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..aOM<$1.EndpointRegistryV1>(9, _omitFieldNames ? '' : 'registry',
        subBuilder: $1.EndpointRegistryV1.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImportPairingResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImportPairingResult copyWith(void Function(ImportPairingResult) updates) =>
      super.copyWith((message) => updates(message as ImportPairingResult))
          as ImportPairingResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ImportPairingResult create() => ImportPairingResult._();
  @$core.override
  ImportPairingResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ImportPairingResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ImportPairingResult>(create);
  static ImportPairingResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $1.EndpointConfigV1 get endpoint => $_getN(2);
  @$pb.TagNumber(3)
  set endpoint($1.EndpointConfigV1 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpoint() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpoint() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.EndpointConfigV1 ensureEndpoint() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get ticketId => $_getSZ(3);
  @$pb.TagNumber(4)
  set ticketId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTicketId() => $_has(3);
  @$pb.TagNumber(4)
  void clearTicketId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get clientKeyFingerprint => $_getSZ(4);
  @$pb.TagNumber(5)
  set clientKeyFingerprint($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasClientKeyFingerprint() => $_has(4);
  @$pb.TagNumber(5)
  void clearClientKeyFingerprint() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasExpiresAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearExpiresAtUnixNano() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get authorizationRequired => $_getBF(6);
  @$pb.TagNumber(7)
  set authorizationRequired($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAuthorizationRequired() => $_has(6);
  @$pb.TagNumber(7)
  void clearAuthorizationRequired() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.ApiError get error => $_getN(7);
  @$pb.TagNumber(8)
  set error($0.ApiError value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasError() => $_has(7);
  @$pb.TagNumber(8)
  void clearError() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.ApiError ensureError() => $_ensure(7);

  @$pb.TagNumber(9)
  $1.EndpointRegistryV1 get registry => $_getN(8);
  @$pb.TagNumber(9)
  set registry($1.EndpointRegistryV1 value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasRegistry() => $_has(8);
  @$pb.TagNumber(9)
  void clearRegistry() => $_clearField(9);
  @$pb.TagNumber(9)
  $1.EndpointRegistryV1 ensureRegistry() => $_ensure(8);
}

class DeleteCredentialRequest extends $pb.GeneratedMessage {
  factory DeleteCredentialRequest({
    $core.String? requestId,
    $core.String? credentialRef,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (credentialRef != null) result.credentialRef = credentialRef;
    return result;
  }

  DeleteCredentialRequest._();

  factory DeleteCredentialRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeleteCredentialRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeleteCredentialRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'credentialRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteCredentialRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteCredentialRequest copyWith(
          void Function(DeleteCredentialRequest) updates) =>
      super.copyWith((message) => updates(message as DeleteCredentialRequest))
          as DeleteCredentialRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeleteCredentialRequest create() => DeleteCredentialRequest._();
  @$core.override
  DeleteCredentialRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeleteCredentialRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeleteCredentialRequest>(create);
  static DeleteCredentialRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get credentialRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set credentialRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCredentialRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearCredentialRef() => $_clearField(2);
}

class DeleteCredentialResult extends $pb.GeneratedMessage {
  factory DeleteCredentialResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (error != null) result.error = error;
    return result;
  }

  DeleteCredentialResult._();

  factory DeleteCredentialResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeleteCredentialResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeleteCredentialResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.ApiError>(3, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteCredentialResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteCredentialResult copyWith(
          void Function(DeleteCredentialResult) updates) =>
      super.copyWith((message) => updates(message as DeleteCredentialResult))
          as DeleteCredentialResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeleteCredentialResult create() => DeleteCredentialResult._();
  @$core.override
  DeleteCredentialResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeleteCredentialResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeleteCredentialResult>(create);
  static DeleteCredentialResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.ApiError get error => $_getN(2);
  @$pb.TagNumber(3)
  set error($0.ApiError value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasError() => $_has(2);
  @$pb.TagNumber(3)
  void clearError() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.ApiError ensureError() => $_ensure(2);
}

class EndpointRegistryGetRequest extends $pb.GeneratedMessage {
  factory EndpointRegistryGetRequest({
    $core.String? requestId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    return result;
  }

  EndpointRegistryGetRequest._();

  factory EndpointRegistryGetRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRegistryGetRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRegistryGetRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryGetRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryGetRequest copyWith(
          void Function(EndpointRegistryGetRequest) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointRegistryGetRequest))
          as EndpointRegistryGetRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRegistryGetRequest create() => EndpointRegistryGetRequest._();
  @$core.override
  EndpointRegistryGetRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRegistryGetRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRegistryGetRequest>(create);
  static EndpointRegistryGetRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);
}

class EndpointRegistryGetResult extends $pb.GeneratedMessage {
  factory EndpointRegistryGetResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $1.EndpointRegistryV1? registry,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (registry != null) result.registry = registry;
    if (error != null) result.error = error;
    return result;
  }

  EndpointRegistryGetResult._();

  factory EndpointRegistryGetResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRegistryGetResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRegistryGetResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$1.EndpointRegistryV1>(3, _omitFieldNames ? '' : 'registry',
        subBuilder: $1.EndpointRegistryV1.create)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryGetResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryGetResult copyWith(
          void Function(EndpointRegistryGetResult) updates) =>
      super.copyWith((message) => updates(message as EndpointRegistryGetResult))
          as EndpointRegistryGetResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRegistryGetResult create() => EndpointRegistryGetResult._();
  @$core.override
  EndpointRegistryGetResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRegistryGetResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRegistryGetResult>(create);
  static EndpointRegistryGetResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $1.EndpointRegistryV1 get registry => $_getN(2);
  @$pb.TagNumber(3)
  set registry($1.EndpointRegistryV1 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRegistry() => $_has(2);
  @$pb.TagNumber(3)
  void clearRegistry() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.EndpointRegistryV1 ensureRegistry() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

class EndpointUpsertRequest extends $pb.GeneratedMessage {
  factory EndpointUpsertRequest({
    $core.String? requestId,
    $1.EndpointConfigV1? endpoint,
    $core.bool? makeDefault,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpoint != null) result.endpoint = endpoint;
    if (makeDefault != null) result.makeDefault = makeDefault;
    return result;
  }

  EndpointUpsertRequest._();

  factory EndpointUpsertRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointUpsertRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointUpsertRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<$1.EndpointConfigV1>(2, _omitFieldNames ? '' : 'endpoint',
        subBuilder: $1.EndpointConfigV1.create)
    ..aOB(3, _omitFieldNames ? '' : 'makeDefault')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointUpsertRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointUpsertRequest copyWith(
          void Function(EndpointUpsertRequest) updates) =>
      super.copyWith((message) => updates(message as EndpointUpsertRequest))
          as EndpointUpsertRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointUpsertRequest create() => EndpointUpsertRequest._();
  @$core.override
  EndpointUpsertRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointUpsertRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointUpsertRequest>(create);
  static EndpointUpsertRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $1.EndpointConfigV1 get endpoint => $_getN(1);
  @$pb.TagNumber(2)
  set endpoint($1.EndpointConfigV1 value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpoint() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpoint() => $_clearField(2);
  @$pb.TagNumber(2)
  $1.EndpointConfigV1 ensureEndpoint() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.bool get makeDefault => $_getBF(2);
  @$pb.TagNumber(3)
  set makeDefault($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMakeDefault() => $_has(2);
  @$pb.TagNumber(3)
  void clearMakeDefault() => $_clearField(3);
}

class EndpointUpsertResult extends $pb.GeneratedMessage {
  factory EndpointUpsertResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $1.EndpointConfigV1? endpoint,
    $1.EndpointRegistryV1? registry,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpoint != null) result.endpoint = endpoint;
    if (registry != null) result.registry = registry;
    if (error != null) result.error = error;
    return result;
  }

  EndpointUpsertResult._();

  factory EndpointUpsertResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointUpsertResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointUpsertResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$1.EndpointConfigV1>(3, _omitFieldNames ? '' : 'endpoint',
        subBuilder: $1.EndpointConfigV1.create)
    ..aOM<$1.EndpointRegistryV1>(4, _omitFieldNames ? '' : 'registry',
        subBuilder: $1.EndpointRegistryV1.create)
    ..aOM<$0.ApiError>(5, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointUpsertResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointUpsertResult copyWith(void Function(EndpointUpsertResult) updates) =>
      super.copyWith((message) => updates(message as EndpointUpsertResult))
          as EndpointUpsertResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointUpsertResult create() => EndpointUpsertResult._();
  @$core.override
  EndpointUpsertResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointUpsertResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointUpsertResult>(create);
  static EndpointUpsertResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $1.EndpointConfigV1 get endpoint => $_getN(2);
  @$pb.TagNumber(3)
  set endpoint($1.EndpointConfigV1 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpoint() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpoint() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.EndpointConfigV1 ensureEndpoint() => $_ensure(2);

  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 get registry => $_getN(3);
  @$pb.TagNumber(4)
  set registry($1.EndpointRegistryV1 value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRegistry() => $_has(3);
  @$pb.TagNumber(4)
  void clearRegistry() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 ensureRegistry() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.ApiError get error => $_getN(4);
  @$pb.TagNumber(5)
  set error($0.ApiError value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasError() => $_has(4);
  @$pb.TagNumber(5)
  void clearError() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.ApiError ensureError() => $_ensure(4);
}

class EndpointDeleteRequest extends $pb.GeneratedMessage {
  factory EndpointDeleteRequest({
    $core.String? requestId,
    $core.String? endpointId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    return result;
  }

  EndpointDeleteRequest._();

  factory EndpointDeleteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointDeleteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointDeleteRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDeleteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDeleteRequest copyWith(
          void Function(EndpointDeleteRequest) updates) =>
      super.copyWith((message) => updates(message as EndpointDeleteRequest))
          as EndpointDeleteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointDeleteRequest create() => EndpointDeleteRequest._();
  @$core.override
  EndpointDeleteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointDeleteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointDeleteRequest>(create);
  static EndpointDeleteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);
}

class EndpointDeleteResult extends $pb.GeneratedMessage {
  factory EndpointDeleteResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $core.String? endpointId,
    $1.EndpointRegistryV1? registry,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpointId != null) result.endpointId = endpointId;
    if (registry != null) result.registry = registry;
    if (error != null) result.error = error;
    return result;
  }

  EndpointDeleteResult._();

  factory EndpointDeleteResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointDeleteResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointDeleteResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'endpointId')
    ..aOM<$1.EndpointRegistryV1>(4, _omitFieldNames ? '' : 'registry',
        subBuilder: $1.EndpointRegistryV1.create)
    ..aOM<$0.ApiError>(5, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDeleteResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointDeleteResult copyWith(void Function(EndpointDeleteResult) updates) =>
      super.copyWith((message) => updates(message as EndpointDeleteResult))
          as EndpointDeleteResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointDeleteResult create() => EndpointDeleteResult._();
  @$core.override
  EndpointDeleteResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointDeleteResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointDeleteResult>(create);
  static EndpointDeleteResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get endpointId => $_getSZ(2);
  @$pb.TagNumber(3)
  set endpointId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpointId() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpointId() => $_clearField(3);

  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 get registry => $_getN(3);
  @$pb.TagNumber(4)
  set registry($1.EndpointRegistryV1 value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRegistry() => $_has(3);
  @$pb.TagNumber(4)
  void clearRegistry() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 ensureRegistry() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.ApiError get error => $_getN(4);
  @$pb.TagNumber(5)
  set error($0.ApiError value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasError() => $_has(4);
  @$pb.TagNumber(5)
  void clearError() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.ApiError ensureError() => $_ensure(4);
}

/// EndpointShareReceiveRequest 通过 Go Client Engine 接收一次性 TLS share bundle，只生成待确认预览。
class EndpointShareReceiveRequest extends $pb.GeneratedMessage {
  factory EndpointShareReceiveRequest({
    $core.String? requestId,
    $core.String? portableOffer,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (portableOffer != null) result.portableOffer = portableOffer;
    return result;
  }

  EndpointShareReceiveRequest._();

  factory EndpointShareReceiveRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointShareReceiveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointShareReceiveRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'portableOffer')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareReceiveRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareReceiveRequest copyWith(
          void Function(EndpointShareReceiveRequest) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointShareReceiveRequest))
          as EndpointShareReceiveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointShareReceiveRequest create() =>
      EndpointShareReceiveRequest._();
  @$core.override
  EndpointShareReceiveRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointShareReceiveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointShareReceiveRequest>(create);
  static EndpointShareReceiveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get portableOffer => $_getSZ(1);
  @$pb.TagNumber(2)
  set portableOffer($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPortableOffer() => $_has(1);
  @$pb.TagNumber(2)
  void clearPortableOffer() => $_clearField(2);
}

class EndpointShareRouteDiff extends $pb.GeneratedMessage {
  factory EndpointShareRouteDiff({
    $core.String? routeId,
    $core.String? routeKind,
    $core.String? action,
  }) {
    final result = create();
    if (routeId != null) result.routeId = routeId;
    if (routeKind != null) result.routeKind = routeKind;
    if (action != null) result.action = action;
    return result;
  }

  EndpointShareRouteDiff._();

  factory EndpointShareRouteDiff.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointShareRouteDiff.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointShareRouteDiff',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'routeId')
    ..aOS(2, _omitFieldNames ? '' : 'routeKind')
    ..aOS(3, _omitFieldNames ? '' : 'action')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareRouteDiff clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareRouteDiff copyWith(
          void Function(EndpointShareRouteDiff) updates) =>
      super.copyWith((message) => updates(message as EndpointShareRouteDiff))
          as EndpointShareRouteDiff;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointShareRouteDiff create() => EndpointShareRouteDiff._();
  @$core.override
  EndpointShareRouteDiff createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointShareRouteDiff getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointShareRouteDiff>(create);
  static EndpointShareRouteDiff? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get routeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set routeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRouteId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRouteId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get routeKind => $_getSZ(1);
  @$pb.TagNumber(2)
  set routeKind($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRouteKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearRouteKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get action => $_getSZ(2);
  @$pb.TagNumber(3)
  set action($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAction() => $_has(2);
  @$pb.TagNumber(3)
  void clearAction() => $_clearField(3);
}

/// EndpointSharePreview 是 Go 根据当前 registry 与已验证 share bundle 计算的确认信息。
class EndpointSharePreview extends $pb.GeneratedMessage {
  factory EndpointSharePreview({
    $core.String? importToken,
    $core.String? endpointId,
    $core.String? label,
    $1.EndpointDaemonIdentity? identity,
    $core.Iterable<EndpointShareRouteDiff>? routeDiffs,
    $core.bool? connectModeChanged,
    $core.bool? selectionPolicyChanged,
    $core.Iterable<$1.EndpointCredentialDescriptor>? credentialDescriptors,
    $fixnum.Int64? expiresAtUnixNano,
  }) {
    final result = create();
    if (importToken != null) result.importToken = importToken;
    if (endpointId != null) result.endpointId = endpointId;
    if (label != null) result.label = label;
    if (identity != null) result.identity = identity;
    if (routeDiffs != null) result.routeDiffs.addAll(routeDiffs);
    if (connectModeChanged != null)
      result.connectModeChanged = connectModeChanged;
    if (selectionPolicyChanged != null)
      result.selectionPolicyChanged = selectionPolicyChanged;
    if (credentialDescriptors != null)
      result.credentialDescriptors.addAll(credentialDescriptors);
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    return result;
  }

  EndpointSharePreview._();

  factory EndpointSharePreview.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSharePreview.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSharePreview',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'importToken')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..aOS(3, _omitFieldNames ? '' : 'label')
    ..aOM<$1.EndpointDaemonIdentity>(4, _omitFieldNames ? '' : 'identity',
        subBuilder: $1.EndpointDaemonIdentity.create)
    ..pPM<EndpointShareRouteDiff>(5, _omitFieldNames ? '' : 'routeDiffs',
        subBuilder: EndpointShareRouteDiff.create)
    ..aOB(6, _omitFieldNames ? '' : 'connectModeChanged')
    ..aOB(7, _omitFieldNames ? '' : 'selectionPolicyChanged')
    ..pPM<$1.EndpointCredentialDescriptor>(
        8, _omitFieldNames ? '' : 'credentialDescriptors',
        subBuilder: $1.EndpointCredentialDescriptor.create)
    ..aInt64(9, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSharePreview clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSharePreview copyWith(void Function(EndpointSharePreview) updates) =>
      super.copyWith((message) => updates(message as EndpointSharePreview))
          as EndpointSharePreview;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSharePreview create() => EndpointSharePreview._();
  @$core.override
  EndpointSharePreview createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSharePreview getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSharePreview>(create);
  static EndpointSharePreview? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get importToken => $_getSZ(0);
  @$pb.TagNumber(1)
  set importToken($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasImportToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearImportToken() => $_clearField(1);

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
  $1.EndpointDaemonIdentity get identity => $_getN(3);
  @$pb.TagNumber(4)
  set identity($1.EndpointDaemonIdentity value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasIdentity() => $_has(3);
  @$pb.TagNumber(4)
  void clearIdentity() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.EndpointDaemonIdentity ensureIdentity() => $_ensure(3);

  @$pb.TagNumber(5)
  $pb.PbList<EndpointShareRouteDiff> get routeDiffs => $_getList(4);

  @$pb.TagNumber(6)
  $core.bool get connectModeChanged => $_getBF(5);
  @$pb.TagNumber(6)
  set connectModeChanged($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasConnectModeChanged() => $_has(5);
  @$pb.TagNumber(6)
  void clearConnectModeChanged() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get selectionPolicyChanged => $_getBF(6);
  @$pb.TagNumber(7)
  set selectionPolicyChanged($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSelectionPolicyChanged() => $_has(6);
  @$pb.TagNumber(7)
  void clearSelectionPolicyChanged() => $_clearField(7);

  @$pb.TagNumber(8)
  $pb.PbList<$1.EndpointCredentialDescriptor> get credentialDescriptors =>
      $_getList(7);

  @$pb.TagNumber(9)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(8);
  @$pb.TagNumber(9)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasExpiresAtUnixNano() => $_has(8);
  @$pb.TagNumber(9)
  void clearExpiresAtUnixNano() => $_clearField(9);
}

class EndpointShareReceiveResult extends $pb.GeneratedMessage {
  factory EndpointShareReceiveResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    EndpointSharePreview? preview,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (preview != null) result.preview = preview;
    if (error != null) result.error = error;
    return result;
  }

  EndpointShareReceiveResult._();

  factory EndpointShareReceiveResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointShareReceiveResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointShareReceiveResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<EndpointSharePreview>(3, _omitFieldNames ? '' : 'preview',
        subBuilder: EndpointSharePreview.create)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareReceiveResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareReceiveResult copyWith(
          void Function(EndpointShareReceiveResult) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointShareReceiveResult))
          as EndpointShareReceiveResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointShareReceiveResult create() => EndpointShareReceiveResult._();
  @$core.override
  EndpointShareReceiveResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointShareReceiveResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointShareReceiveResult>(create);
  static EndpointShareReceiveResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointSharePreview get preview => $_getN(2);
  @$pb.TagNumber(3)
  set preview(EndpointSharePreview value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasPreview() => $_has(2);
  @$pb.TagNumber(3)
  void clearPreview() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointSharePreview ensurePreview() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

/// EndpointShareCommitRequest 仅提交当前 generation 内尚未过期的 preview token。
class EndpointShareCommitRequest extends $pb.GeneratedMessage {
  factory EndpointShareCommitRequest({
    $core.String? requestId,
    $core.String? importToken,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (importToken != null) result.importToken = importToken;
    return result;
  }

  EndpointShareCommitRequest._();

  factory EndpointShareCommitRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointShareCommitRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointShareCommitRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'importToken')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareCommitRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareCommitRequest copyWith(
          void Function(EndpointShareCommitRequest) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointShareCommitRequest))
          as EndpointShareCommitRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointShareCommitRequest create() => EndpointShareCommitRequest._();
  @$core.override
  EndpointShareCommitRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointShareCommitRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointShareCommitRequest>(create);
  static EndpointShareCommitRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get importToken => $_getSZ(1);
  @$pb.TagNumber(2)
  set importToken($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasImportToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearImportToken() => $_clearField(2);
}

class EndpointShareCommitResult extends $pb.GeneratedMessage {
  factory EndpointShareCommitResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $1.EndpointConfigV1? endpoint,
    $1.EndpointRegistryV1? registry,
    $core.bool? authorizationRequired,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpoint != null) result.endpoint = endpoint;
    if (registry != null) result.registry = registry;
    if (authorizationRequired != null)
      result.authorizationRequired = authorizationRequired;
    if (error != null) result.error = error;
    return result;
  }

  EndpointShareCommitResult._();

  factory EndpointShareCommitResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointShareCommitResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointShareCommitResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$1.EndpointConfigV1>(3, _omitFieldNames ? '' : 'endpoint',
        subBuilder: $1.EndpointConfigV1.create)
    ..aOM<$1.EndpointRegistryV1>(4, _omitFieldNames ? '' : 'registry',
        subBuilder: $1.EndpointRegistryV1.create)
    ..aOB(5, _omitFieldNames ? '' : 'authorizationRequired')
    ..aOM<$0.ApiError>(6, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareCommitResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointShareCommitResult copyWith(
          void Function(EndpointShareCommitResult) updates) =>
      super.copyWith((message) => updates(message as EndpointShareCommitResult))
          as EndpointShareCommitResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointShareCommitResult create() => EndpointShareCommitResult._();
  @$core.override
  EndpointShareCommitResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointShareCommitResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointShareCommitResult>(create);
  static EndpointShareCommitResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $1.EndpointConfigV1 get endpoint => $_getN(2);
  @$pb.TagNumber(3)
  set endpoint($1.EndpointConfigV1 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpoint() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpoint() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.EndpointConfigV1 ensureEndpoint() => $_ensure(2);

  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 get registry => $_getN(3);
  @$pb.TagNumber(4)
  set registry($1.EndpointRegistryV1 value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRegistry() => $_has(3);
  @$pb.TagNumber(4)
  void clearRegistry() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 ensureRegistry() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.bool get authorizationRequired => $_getBF(4);
  @$pb.TagNumber(5)
  set authorizationRequired($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAuthorizationRequired() => $_has(4);
  @$pb.TagNumber(5)
  void clearAuthorizationRequired() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.ApiError get error => $_getN(5);
  @$pb.TagNumber(6)
  set error($0.ApiError value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasError() => $_has(5);
  @$pb.TagNumber(6)
  void clearError() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.ApiError ensureError() => $_ensure(5);
}

/// SSHCredentialProvisionRequest 让 Go Client Engine 为已存在的 SSH Route 准备目标平台 signer。
/// endpoint_id/route_id 只定位 Go registry 中的 Route；平台不会收到或解析 Endpoint 配置。
class SSHCredentialProvisionRequest extends $pb.GeneratedMessage {
  factory SSHCredentialProvisionRequest({
    $core.String? requestId,
    $core.String? endpointId,
    $core.String? routeId,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (endpointId != null) result.endpointId = endpointId;
    if (routeId != null) result.routeId = routeId;
    return result;
  }

  SSHCredentialProvisionRequest._();

  factory SSHCredentialProvisionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialProvisionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialProvisionRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointId')
    ..aOS(3, _omitFieldNames ? '' : 'routeId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialProvisionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialProvisionRequest copyWith(
          void Function(SSHCredentialProvisionRequest) updates) =>
      super.copyWith(
              (message) => updates(message as SSHCredentialProvisionRequest))
          as SSHCredentialProvisionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialProvisionRequest create() =>
      SSHCredentialProvisionRequest._();
  @$core.override
  SSHCredentialProvisionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialProvisionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialProvisionRequest>(create);
  static SSHCredentialProvisionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointId => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get routeId => $_getSZ(2);
  @$pb.TagNumber(3)
  set routeId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRouteId() => $_has(2);
  @$pb.TagNumber(3)
  void clearRouteId() => $_clearField(3);
}

/// SSHCredentialProvisionResult 返回更新后的 Go registry projection 和可安装到 SSH server 的公钥。
/// private key 始终留在平台 secure signer 中，不进入 Proto payload。
class SSHCredentialProvisionResult extends $pb.GeneratedMessage {
  factory SSHCredentialProvisionResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $1.EndpointConfigV1? endpoint,
    $1.EndpointRegistryV1? registry,
    $core.String? credentialRef,
    $core.String? authorizedKey,
    $core.String? keyFingerprint,
    $0.ApiError? error,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpoint != null) result.endpoint = endpoint;
    if (registry != null) result.registry = registry;
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (authorizedKey != null) result.authorizedKey = authorizedKey;
    if (keyFingerprint != null) result.keyFingerprint = keyFingerprint;
    if (error != null) result.error = error;
    return result;
  }

  SSHCredentialProvisionResult._();

  factory SSHCredentialProvisionResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialProvisionResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialProvisionResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$1.EndpointConfigV1>(3, _omitFieldNames ? '' : 'endpoint',
        subBuilder: $1.EndpointConfigV1.create)
    ..aOM<$1.EndpointRegistryV1>(4, _omitFieldNames ? '' : 'registry',
        subBuilder: $1.EndpointRegistryV1.create)
    ..aOS(5, _omitFieldNames ? '' : 'credentialRef')
    ..aOS(6, _omitFieldNames ? '' : 'authorizedKey')
    ..aOS(7, _omitFieldNames ? '' : 'keyFingerprint')
    ..aOM<$0.ApiError>(8, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialProvisionResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialProvisionResult copyWith(
          void Function(SSHCredentialProvisionResult) updates) =>
      super.copyWith(
              (message) => updates(message as SSHCredentialProvisionResult))
          as SSHCredentialProvisionResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialProvisionResult create() =>
      SSHCredentialProvisionResult._();
  @$core.override
  SSHCredentialProvisionResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialProvisionResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialProvisionResult>(create);
  static SSHCredentialProvisionResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $1.EndpointConfigV1 get endpoint => $_getN(2);
  @$pb.TagNumber(3)
  set endpoint($1.EndpointConfigV1 value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpoint() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpoint() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.EndpointConfigV1 ensureEndpoint() => $_ensure(2);

  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 get registry => $_getN(3);
  @$pb.TagNumber(4)
  set registry($1.EndpointRegistryV1 value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRegistry() => $_has(3);
  @$pb.TagNumber(4)
  void clearRegistry() => $_clearField(4);
  @$pb.TagNumber(4)
  $1.EndpointRegistryV1 ensureRegistry() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.String get credentialRef => $_getSZ(4);
  @$pb.TagNumber(5)
  set credentialRef($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCredentialRef() => $_has(4);
  @$pb.TagNumber(5)
  void clearCredentialRef() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get authorizedKey => $_getSZ(5);
  @$pb.TagNumber(6)
  set authorizedKey($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAuthorizedKey() => $_has(5);
  @$pb.TagNumber(6)
  void clearAuthorizedKey() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get keyFingerprint => $_getSZ(6);
  @$pb.TagNumber(7)
  set keyFingerprint($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasKeyFingerprint() => $_has(6);
  @$pb.TagNumber(7)
  void clearKeyFingerprint() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.ApiError get error => $_getN(7);
  @$pb.TagNumber(8)
  set error($0.ApiError value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasError() => $_has(7);
  @$pb.TagNumber(8)
  void clearError() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.ApiError ensureError() => $_ensure(7);
}

enum EngineCommand_Command {
  importPairing,
  deleteCredential,
  endpointRegistryGet,
  endpointUpsert,
  endpointDelete,
  endpointShareReceive,
  endpointShareCommit,
  sshCredentialProvision,
  connectionPolicyGet,
  connectionPolicyApply,
  connectionSnapshotGet,
  sessionInvalidate,
  endpointDisconnect,
  endpointCloudPresenceGet,
  notSet
}

class EngineCommand extends $pb.GeneratedMessage {
  factory EngineCommand({
    ImportPairingRequest? importPairing,
    DeleteCredentialRequest? deleteCredential,
    EndpointRegistryGetRequest? endpointRegistryGet,
    EndpointUpsertRequest? endpointUpsert,
    EndpointDeleteRequest? endpointDelete,
    EndpointShareReceiveRequest? endpointShareReceive,
    EndpointShareCommitRequest? endpointShareCommit,
    SSHCredentialProvisionRequest? sshCredentialProvision,
    ConnectionPolicyGetRequest? connectionPolicyGet,
    ConnectionPolicyApplyRequest? connectionPolicyApply,
    ConnectionSnapshotGetRequest? connectionSnapshotGet,
    SessionInvalidateRequest? sessionInvalidate,
    EndpointDisconnectRequest? endpointDisconnect,
    EndpointCloudPresenceGetRequest? endpointCloudPresenceGet,
  }) {
    final result = create();
    if (importPairing != null) result.importPairing = importPairing;
    if (deleteCredential != null) result.deleteCredential = deleteCredential;
    if (endpointRegistryGet != null)
      result.endpointRegistryGet = endpointRegistryGet;
    if (endpointUpsert != null) result.endpointUpsert = endpointUpsert;
    if (endpointDelete != null) result.endpointDelete = endpointDelete;
    if (endpointShareReceive != null)
      result.endpointShareReceive = endpointShareReceive;
    if (endpointShareCommit != null)
      result.endpointShareCommit = endpointShareCommit;
    if (sshCredentialProvision != null)
      result.sshCredentialProvision = sshCredentialProvision;
    if (connectionPolicyGet != null)
      result.connectionPolicyGet = connectionPolicyGet;
    if (connectionPolicyApply != null)
      result.connectionPolicyApply = connectionPolicyApply;
    if (connectionSnapshotGet != null)
      result.connectionSnapshotGet = connectionSnapshotGet;
    if (sessionInvalidate != null) result.sessionInvalidate = sessionInvalidate;
    if (endpointDisconnect != null)
      result.endpointDisconnect = endpointDisconnect;
    if (endpointCloudPresenceGet != null)
      result.endpointCloudPresenceGet = endpointCloudPresenceGet;
    return result;
  }

  EngineCommand._();

  factory EngineCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EngineCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EngineCommand_Command>
      _EngineCommand_CommandByTag = {
    1: EngineCommand_Command.importPairing,
    2: EngineCommand_Command.deleteCredential,
    3: EngineCommand_Command.endpointRegistryGet,
    4: EngineCommand_Command.endpointUpsert,
    5: EngineCommand_Command.endpointDelete,
    6: EngineCommand_Command.endpointShareReceive,
    7: EngineCommand_Command.endpointShareCommit,
    8: EngineCommand_Command.sshCredentialProvision,
    9: EngineCommand_Command.connectionPolicyGet,
    10: EngineCommand_Command.connectionPolicyApply,
    11: EngineCommand_Command.connectionSnapshotGet,
    12: EngineCommand_Command.sessionInvalidate,
    13: EngineCommand_Command.endpointDisconnect,
    14: EngineCommand_Command.endpointCloudPresenceGet,
    0: EngineCommand_Command.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EngineCommand',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14])
    ..aOM<ImportPairingRequest>(1, _omitFieldNames ? '' : 'importPairing',
        subBuilder: ImportPairingRequest.create)
    ..aOM<DeleteCredentialRequest>(2, _omitFieldNames ? '' : 'deleteCredential',
        subBuilder: DeleteCredentialRequest.create)
    ..aOM<EndpointRegistryGetRequest>(
        3, _omitFieldNames ? '' : 'endpointRegistryGet',
        subBuilder: EndpointRegistryGetRequest.create)
    ..aOM<EndpointUpsertRequest>(4, _omitFieldNames ? '' : 'endpointUpsert',
        subBuilder: EndpointUpsertRequest.create)
    ..aOM<EndpointDeleteRequest>(5, _omitFieldNames ? '' : 'endpointDelete',
        subBuilder: EndpointDeleteRequest.create)
    ..aOM<EndpointShareReceiveRequest>(
        6, _omitFieldNames ? '' : 'endpointShareReceive',
        subBuilder: EndpointShareReceiveRequest.create)
    ..aOM<EndpointShareCommitRequest>(
        7, _omitFieldNames ? '' : 'endpointShareCommit',
        subBuilder: EndpointShareCommitRequest.create)
    ..aOM<SSHCredentialProvisionRequest>(
        8, _omitFieldNames ? '' : 'sshCredentialProvision',
        subBuilder: SSHCredentialProvisionRequest.create)
    ..aOM<ConnectionPolicyGetRequest>(
        9, _omitFieldNames ? '' : 'connectionPolicyGet',
        subBuilder: ConnectionPolicyGetRequest.create)
    ..aOM<ConnectionPolicyApplyRequest>(
        10, _omitFieldNames ? '' : 'connectionPolicyApply',
        subBuilder: ConnectionPolicyApplyRequest.create)
    ..aOM<ConnectionSnapshotGetRequest>(
        11, _omitFieldNames ? '' : 'connectionSnapshotGet',
        subBuilder: ConnectionSnapshotGetRequest.create)
    ..aOM<SessionInvalidateRequest>(
        12, _omitFieldNames ? '' : 'sessionInvalidate',
        subBuilder: SessionInvalidateRequest.create)
    ..aOM<EndpointDisconnectRequest>(
        13, _omitFieldNames ? '' : 'endpointDisconnect',
        subBuilder: EndpointDisconnectRequest.create)
    ..aOM<EndpointCloudPresenceGetRequest>(
        14, _omitFieldNames ? '' : 'endpointCloudPresenceGet',
        subBuilder: EndpointCloudPresenceGetRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EngineCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EngineCommand copyWith(void Function(EngineCommand) updates) =>
      super.copyWith((message) => updates(message as EngineCommand))
          as EngineCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EngineCommand create() => EngineCommand._();
  @$core.override
  EngineCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EngineCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EngineCommand>(create);
  static EngineCommand? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  EngineCommand_Command whichCommand() =>
      _EngineCommand_CommandByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  void clearCommand() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  ImportPairingRequest get importPairing => $_getN(0);
  @$pb.TagNumber(1)
  set importPairing(ImportPairingRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasImportPairing() => $_has(0);
  @$pb.TagNumber(1)
  void clearImportPairing() => $_clearField(1);
  @$pb.TagNumber(1)
  ImportPairingRequest ensureImportPairing() => $_ensure(0);

  @$pb.TagNumber(2)
  DeleteCredentialRequest get deleteCredential => $_getN(1);
  @$pb.TagNumber(2)
  set deleteCredential(DeleteCredentialRequest value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasDeleteCredential() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeleteCredential() => $_clearField(2);
  @$pb.TagNumber(2)
  DeleteCredentialRequest ensureDeleteCredential() => $_ensure(1);

  @$pb.TagNumber(3)
  EndpointRegistryGetRequest get endpointRegistryGet => $_getN(2);
  @$pb.TagNumber(3)
  set endpointRegistryGet(EndpointRegistryGetRequest value) =>
      $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpointRegistryGet() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpointRegistryGet() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointRegistryGetRequest ensureEndpointRegistryGet() => $_ensure(2);

  @$pb.TagNumber(4)
  EndpointUpsertRequest get endpointUpsert => $_getN(3);
  @$pb.TagNumber(4)
  set endpointUpsert(EndpointUpsertRequest value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasEndpointUpsert() => $_has(3);
  @$pb.TagNumber(4)
  void clearEndpointUpsert() => $_clearField(4);
  @$pb.TagNumber(4)
  EndpointUpsertRequest ensureEndpointUpsert() => $_ensure(3);

  @$pb.TagNumber(5)
  EndpointDeleteRequest get endpointDelete => $_getN(4);
  @$pb.TagNumber(5)
  set endpointDelete(EndpointDeleteRequest value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasEndpointDelete() => $_has(4);
  @$pb.TagNumber(5)
  void clearEndpointDelete() => $_clearField(5);
  @$pb.TagNumber(5)
  EndpointDeleteRequest ensureEndpointDelete() => $_ensure(4);

  @$pb.TagNumber(6)
  EndpointShareReceiveRequest get endpointShareReceive => $_getN(5);
  @$pb.TagNumber(6)
  set endpointShareReceive(EndpointShareReceiveRequest value) =>
      $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasEndpointShareReceive() => $_has(5);
  @$pb.TagNumber(6)
  void clearEndpointShareReceive() => $_clearField(6);
  @$pb.TagNumber(6)
  EndpointShareReceiveRequest ensureEndpointShareReceive() => $_ensure(5);

  @$pb.TagNumber(7)
  EndpointShareCommitRequest get endpointShareCommit => $_getN(6);
  @$pb.TagNumber(7)
  set endpointShareCommit(EndpointShareCommitRequest value) =>
      $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasEndpointShareCommit() => $_has(6);
  @$pb.TagNumber(7)
  void clearEndpointShareCommit() => $_clearField(7);
  @$pb.TagNumber(7)
  EndpointShareCommitRequest ensureEndpointShareCommit() => $_ensure(6);

  @$pb.TagNumber(8)
  SSHCredentialProvisionRequest get sshCredentialProvision => $_getN(7);
  @$pb.TagNumber(8)
  set sshCredentialProvision(SSHCredentialProvisionRequest value) =>
      $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasSshCredentialProvision() => $_has(7);
  @$pb.TagNumber(8)
  void clearSshCredentialProvision() => $_clearField(8);
  @$pb.TagNumber(8)
  SSHCredentialProvisionRequest ensureSshCredentialProvision() => $_ensure(7);

  @$pb.TagNumber(9)
  ConnectionPolicyGetRequest get connectionPolicyGet => $_getN(8);
  @$pb.TagNumber(9)
  set connectionPolicyGet(ConnectionPolicyGetRequest value) =>
      $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasConnectionPolicyGet() => $_has(8);
  @$pb.TagNumber(9)
  void clearConnectionPolicyGet() => $_clearField(9);
  @$pb.TagNumber(9)
  ConnectionPolicyGetRequest ensureConnectionPolicyGet() => $_ensure(8);

  @$pb.TagNumber(10)
  ConnectionPolicyApplyRequest get connectionPolicyApply => $_getN(9);
  @$pb.TagNumber(10)
  set connectionPolicyApply(ConnectionPolicyApplyRequest value) =>
      $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasConnectionPolicyApply() => $_has(9);
  @$pb.TagNumber(10)
  void clearConnectionPolicyApply() => $_clearField(10);
  @$pb.TagNumber(10)
  ConnectionPolicyApplyRequest ensureConnectionPolicyApply() => $_ensure(9);

  @$pb.TagNumber(11)
  ConnectionSnapshotGetRequest get connectionSnapshotGet => $_getN(10);
  @$pb.TagNumber(11)
  set connectionSnapshotGet(ConnectionSnapshotGetRequest value) =>
      $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasConnectionSnapshotGet() => $_has(10);
  @$pb.TagNumber(11)
  void clearConnectionSnapshotGet() => $_clearField(11);
  @$pb.TagNumber(11)
  ConnectionSnapshotGetRequest ensureConnectionSnapshotGet() => $_ensure(10);

  @$pb.TagNumber(12)
  SessionInvalidateRequest get sessionInvalidate => $_getN(11);
  @$pb.TagNumber(12)
  set sessionInvalidate(SessionInvalidateRequest value) =>
      $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasSessionInvalidate() => $_has(11);
  @$pb.TagNumber(12)
  void clearSessionInvalidate() => $_clearField(12);
  @$pb.TagNumber(12)
  SessionInvalidateRequest ensureSessionInvalidate() => $_ensure(11);

  @$pb.TagNumber(13)
  EndpointDisconnectRequest get endpointDisconnect => $_getN(12);
  @$pb.TagNumber(13)
  set endpointDisconnect(EndpointDisconnectRequest value) =>
      $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasEndpointDisconnect() => $_has(12);
  @$pb.TagNumber(13)
  void clearEndpointDisconnect() => $_clearField(13);
  @$pb.TagNumber(13)
  EndpointDisconnectRequest ensureEndpointDisconnect() => $_ensure(12);

  @$pb.TagNumber(14)
  EndpointCloudPresenceGetRequest get endpointCloudPresenceGet => $_getN(13);
  @$pb.TagNumber(14)
  set endpointCloudPresenceGet(EndpointCloudPresenceGetRequest value) =>
      $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasEndpointCloudPresenceGet() => $_has(13);
  @$pb.TagNumber(14)
  void clearEndpointCloudPresenceGet() => $_clearField(14);
  @$pb.TagNumber(14)
  EndpointCloudPresenceGetRequest ensureEndpointCloudPresenceGet() =>
      $_ensure(13);
}

class OpenSessionResult extends $pb.GeneratedMessage {
  factory OpenSessionResult({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $fixnum.Int64? sessionHandle,
    $0.EndpointSessionStamp? session,
    $0.ApiError? error,
    ConnectionSnapshot? connection,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    if (session != null) result.session = session;
    if (error != null) result.error = error;
    if (connection != null) result.connection = connection;
    return result;
  }

  OpenSessionResult._();

  factory OpenSessionResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OpenSessionResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OpenSessionResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.EndpointSessionStamp>(4, _omitFieldNames ? '' : 'session',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOM<$0.ApiError>(5, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..aOM<ConnectionSnapshot>(6, _omitFieldNames ? '' : 'connection',
        subBuilder: ConnectionSnapshot.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenSessionResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenSessionResult copyWith(void Function(OpenSessionResult) updates) =>
      super.copyWith((message) => updates(message as OpenSessionResult))
          as OpenSessionResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OpenSessionResult create() => OpenSessionResult._();
  @$core.override
  OpenSessionResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OpenSessionResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OpenSessionResult>(create);
  static OpenSessionResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get sessionHandle => $_getI64(2);
  @$pb.TagNumber(3)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionHandle() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionHandle() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.EndpointSessionStamp get session => $_getN(3);
  @$pb.TagNumber(4)
  set session($0.EndpointSessionStamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSession() => $_has(3);
  @$pb.TagNumber(4)
  void clearSession() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.EndpointSessionStamp ensureSession() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.ApiError get error => $_getN(4);
  @$pb.TagNumber(5)
  set error($0.ApiError value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasError() => $_has(4);
  @$pb.TagNumber(5)
  void clearError() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.ApiError ensureError() => $_ensure(4);

  @$pb.TagNumber(6)
  ConnectionSnapshot get connection => $_getN(5);
  @$pb.TagNumber(6)
  set connection(ConnectionSnapshot value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasConnection() => $_has(5);
  @$pb.TagNumber(6)
  void clearConnection() => $_clearField(6);
  @$pb.TagNumber(6)
  ConnectionSnapshot ensureConnection() => $_ensure(5);
}

class ExecuteResult extends $pb.GeneratedMessage {
  factory ExecuteResult({
    $fixnum.Int64? operationHandle,
    $fixnum.Int64? sessionHandle,
    $2.ResultEnvelope? result,
    $0.ApiError? error,
  }) {
    final result$ = create();
    if (operationHandle != null) result$.operationHandle = operationHandle;
    if (sessionHandle != null) result$.sessionHandle = sessionHandle;
    if (result != null) result$.result = result;
    if (error != null) result$.error = error;
    return result$;
  }

  ExecuteResult._();

  factory ExecuteResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ExecuteResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ExecuteResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$2.ResultEnvelope>(3, _omitFieldNames ? '' : 'result',
        subBuilder: $2.ResultEnvelope.create)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExecuteResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExecuteResult copyWith(void Function(ExecuteResult) updates) =>
      super.copyWith((message) => updates(message as ExecuteResult))
          as ExecuteResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ExecuteResult create() => ExecuteResult._();
  @$core.override
  ExecuteResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ExecuteResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ExecuteResult>(create);
  static ExecuteResult? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get operationHandle => $_getI64(0);
  @$pb.TagNumber(1)
  set operationHandle($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOperationHandle() => $_has(0);
  @$pb.TagNumber(1)
  void clearOperationHandle() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get sessionHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $2.ResultEnvelope get result => $_getN(2);
  @$pb.TagNumber(3)
  set result($2.ResultEnvelope value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasResult() => $_has(2);
  @$pb.TagNumber(3)
  void clearResult() => $_clearField(3);
  @$pb.TagNumber(3)
  $2.ResultEnvelope ensureResult() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

class ApplicationEvent extends $pb.GeneratedMessage {
  factory ApplicationEvent({
    $fixnum.Int64? sessionHandle,
    $2.EventEnvelope? event,
  }) {
    final result = create();
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    if (event != null) result.event = event;
    return result;
  }

  ApplicationEvent._();

  factory ApplicationEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ApplicationEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ApplicationEvent',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$2.EventEnvelope>(2, _omitFieldNames ? '' : 'event',
        subBuilder: $2.EventEnvelope.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApplicationEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApplicationEvent copyWith(void Function(ApplicationEvent) updates) =>
      super.copyWith((message) => updates(message as ApplicationEvent))
          as ApplicationEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ApplicationEvent create() => ApplicationEvent._();
  @$core.override
  ApplicationEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ApplicationEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ApplicationEvent>(create);
  static ApplicationEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get sessionHandle => $_getI64(0);
  @$pb.TagNumber(1)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionHandle() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionHandle() => $_clearField(1);

  @$pb.TagNumber(2)
  $2.EventEnvelope get event => $_getN(1);
  @$pb.TagNumber(2)
  set event($2.EventEnvelope value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasEvent() => $_has(1);
  @$pb.TagNumber(2)
  void clearEvent() => $_clearField(2);
  @$pb.TagNumber(2)
  $2.EventEnvelope ensureEvent() => $_ensure(1);
}

class OpenResourceStreamRequest extends $pb.GeneratedMessage {
  factory OpenResourceStreamRequest({
    $0.ResourceHandle? resource,
    $fixnum.Int64? initialUploadOffset,
  }) {
    final result = create();
    if (resource != null) result.resource = resource;
    if (initialUploadOffset != null)
      result.initialUploadOffset = initialUploadOffset;
    return result;
  }

  OpenResourceStreamRequest._();

  factory OpenResourceStreamRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OpenResourceStreamRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OpenResourceStreamRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(1, _omitFieldNames ? '' : 'resource',
        subBuilder: $0.ResourceHandle.create)
    ..aInt64(2, _omitFieldNames ? '' : 'initialUploadOffset')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenResourceStreamRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenResourceStreamRequest copyWith(
          void Function(OpenResourceStreamRequest) updates) =>
      super.copyWith((message) => updates(message as OpenResourceStreamRequest))
          as OpenResourceStreamRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OpenResourceStreamRequest create() => OpenResourceStreamRequest._();
  @$core.override
  OpenResourceStreamRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OpenResourceStreamRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OpenResourceStreamRequest>(create);
  static OpenResourceStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $0.ResourceHandle get resource => $_getN(0);
  @$pb.TagNumber(1)
  set resource($0.ResourceHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasResource() => $_has(0);
  @$pb.TagNumber(1)
  void clearResource() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.ResourceHandle ensureResource() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get initialUploadOffset => $_getI64(1);
  @$pb.TagNumber(2)
  set initialUploadOffset($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasInitialUploadOffset() => $_has(1);
  @$pb.TagNumber(2)
  void clearInitialUploadOffset() => $_clearField(2);
}

class ResourceStreamFrame extends $pb.GeneratedMessage {
  factory ResourceStreamFrame({
    $fixnum.Int64? streamHandle,
    ResourceStreamFrameType? type,
    $core.List<$core.int>? payload,
  }) {
    final result = create();
    if (streamHandle != null) result.streamHandle = streamHandle;
    if (type != null) result.type = type;
    if (payload != null) result.payload = payload;
    return result;
  }

  ResourceStreamFrame._();

  factory ResourceStreamFrame.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResourceStreamFrame.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResourceStreamFrame',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'streamHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<ResourceStreamFrameType>(2, _omitFieldNames ? '' : 'type',
        enumValues: ResourceStreamFrameType.values)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'payload', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceStreamFrame clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceStreamFrame copyWith(void Function(ResourceStreamFrame) updates) =>
      super.copyWith((message) => updates(message as ResourceStreamFrame))
          as ResourceStreamFrame;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResourceStreamFrame create() => ResourceStreamFrame._();
  @$core.override
  ResourceStreamFrame createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResourceStreamFrame getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResourceStreamFrame>(create);
  static ResourceStreamFrame? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get streamHandle => $_getI64(0);
  @$pb.TagNumber(1)
  set streamHandle($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamHandle() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamHandle() => $_clearField(1);

  @$pb.TagNumber(2)
  ResourceStreamFrameType get type => $_getN(1);
  @$pb.TagNumber(2)
  set type(ResourceStreamFrameType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get payload => $_getN(2);
  @$pb.TagNumber(3)
  set payload($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearPayload() => $_clearField(3);
}

class ResourceStreamClosedEvent extends $pb.GeneratedMessage {
  factory ResourceStreamClosedEvent({
    $fixnum.Int64? streamHandle,
    $0.ApiError? error,
  }) {
    final result = create();
    if (streamHandle != null) result.streamHandle = streamHandle;
    if (error != null) result.error = error;
    return result;
  }

  ResourceStreamClosedEvent._();

  factory ResourceStreamClosedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResourceStreamClosedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResourceStreamClosedEvent',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'streamHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.ApiError>(2, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceStreamClosedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceStreamClosedEvent copyWith(
          void Function(ResourceStreamClosedEvent) updates) =>
      super.copyWith((message) => updates(message as ResourceStreamClosedEvent))
          as ResourceStreamClosedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResourceStreamClosedEvent create() => ResourceStreamClosedEvent._();
  @$core.override
  ResourceStreamClosedEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResourceStreamClosedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResourceStreamClosedEvent>(create);
  static ResourceStreamClosedEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get streamHandle => $_getI64(0);
  @$pb.TagNumber(1)
  set streamHandle($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamHandle() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamHandle() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.ApiError get error => $_getN(1);
  @$pb.TagNumber(2)
  set error($0.ApiError value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ApiError ensureError() => $_ensure(1);
}

class SessionClosedEvent extends $pb.GeneratedMessage {
  factory SessionClosedEvent({
    $fixnum.Int64? sessionHandle,
    $0.EndpointSessionStamp? session,
    $0.ApiError? error,
  }) {
    final result = create();
    if (sessionHandle != null) result.sessionHandle = sessionHandle;
    if (session != null) result.session = session;
    if (error != null) result.error = error;
    return result;
  }

  SessionClosedEvent._();

  factory SessionClosedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SessionClosedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SessionClosedEvent',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'sessionHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.EndpointSessionStamp>(2, _omitFieldNames ? '' : 'session',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOM<$0.ApiError>(3, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionClosedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionClosedEvent copyWith(void Function(SessionClosedEvent) updates) =>
      super.copyWith((message) => updates(message as SessionClosedEvent))
          as SessionClosedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SessionClosedEvent create() => SessionClosedEvent._();
  @$core.override
  SessionClosedEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SessionClosedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SessionClosedEvent>(create);
  static SessionClosedEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get sessionHandle => $_getI64(0);
  @$pb.TagNumber(1)
  set sessionHandle($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionHandle() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionHandle() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.EndpointSessionStamp get session => $_getN(1);
  @$pb.TagNumber(2)
  set session($0.EndpointSessionStamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasSession() => $_has(1);
  @$pb.TagNumber(2)
  void clearSession() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.EndpointSessionStamp ensureSession() => $_ensure(1);

  @$pb.TagNumber(3)
  $0.ApiError get error => $_getN(2);
  @$pb.TagNumber(3)
  set error($0.ApiError value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasError() => $_has(2);
  @$pb.TagNumber(3)
  void clearError() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.ApiError ensureError() => $_ensure(2);
}

/// EndpointConnectionEvent forwards the Go-owned lifecycle for one OpenSession
/// operation. Policy intent is deliberately absent; observed_path is populated
/// only when the runtime has selected a ready transport.
class EndpointConnectionEvent extends $pb.GeneratedMessage {
  factory EndpointConnectionEvent({
    $core.String? requestId,
    $fixnum.Int64? operationHandle,
    $core.String? endpointId,
    $0.EndpointSessionStamp? session,
    EndpointConnectionPhase? phase,
    ConnectionObservedPath? observedPath,
    $core.String? routeSelectionReason,
    $0.ApiError? error,
    ConnectionRouteKind? attemptedRouteKind,
    $core.String? connectionStage,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (operationHandle != null) result.operationHandle = operationHandle;
    if (endpointId != null) result.endpointId = endpointId;
    if (session != null) result.session = session;
    if (phase != null) result.phase = phase;
    if (observedPath != null) result.observedPath = observedPath;
    if (routeSelectionReason != null)
      result.routeSelectionReason = routeSelectionReason;
    if (error != null) result.error = error;
    if (attemptedRouteKind != null)
      result.attemptedRouteKind = attemptedRouteKind;
    if (connectionStage != null) result.connectionStage = connectionStage;
    return result;
  }

  EndpointConnectionEvent._();

  factory EndpointConnectionEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointConnectionEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointConnectionEvent',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'operationHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'endpointId')
    ..aOM<$0.EndpointSessionStamp>(4, _omitFieldNames ? '' : 'session',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aE<EndpointConnectionPhase>(5, _omitFieldNames ? '' : 'phase',
        enumValues: EndpointConnectionPhase.values)
    ..aE<ConnectionObservedPath>(6, _omitFieldNames ? '' : 'observedPath',
        enumValues: ConnectionObservedPath.values)
    ..aOS(7, _omitFieldNames ? '' : 'routeSelectionReason')
    ..aOM<$0.ApiError>(8, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..aE<ConnectionRouteKind>(9, _omitFieldNames ? '' : 'attemptedRouteKind',
        enumValues: ConnectionRouteKind.values)
    ..aOS(10, _omitFieldNames ? '' : 'connectionStage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointConnectionEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointConnectionEvent copyWith(
          void Function(EndpointConnectionEvent) updates) =>
      super.copyWith((message) => updates(message as EndpointConnectionEvent))
          as EndpointConnectionEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointConnectionEvent create() => EndpointConnectionEvent._();
  @$core.override
  EndpointConnectionEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointConnectionEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointConnectionEvent>(create);
  static EndpointConnectionEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get operationHandle => $_getI64(1);
  @$pb.TagNumber(2)
  set operationHandle($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationHandle() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationHandle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get endpointId => $_getSZ(2);
  @$pb.TagNumber(3)
  set endpointId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEndpointId() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndpointId() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.EndpointSessionStamp get session => $_getN(3);
  @$pb.TagNumber(4)
  set session($0.EndpointSessionStamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSession() => $_has(3);
  @$pb.TagNumber(4)
  void clearSession() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.EndpointSessionStamp ensureSession() => $_ensure(3);

  @$pb.TagNumber(5)
  EndpointConnectionPhase get phase => $_getN(4);
  @$pb.TagNumber(5)
  set phase(EndpointConnectionPhase value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasPhase() => $_has(4);
  @$pb.TagNumber(5)
  void clearPhase() => $_clearField(5);

  @$pb.TagNumber(6)
  ConnectionObservedPath get observedPath => $_getN(5);
  @$pb.TagNumber(6)
  set observedPath(ConnectionObservedPath value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasObservedPath() => $_has(5);
  @$pb.TagNumber(6)
  void clearObservedPath() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get routeSelectionReason => $_getSZ(6);
  @$pb.TagNumber(7)
  set routeSelectionReason($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRouteSelectionReason() => $_has(6);
  @$pb.TagNumber(7)
  void clearRouteSelectionReason() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.ApiError get error => $_getN(7);
  @$pb.TagNumber(8)
  set error($0.ApiError value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasError() => $_has(7);
  @$pb.TagNumber(8)
  void clearError() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.ApiError ensureError() => $_ensure(7);

  /// attempted_route_kind identifies the concrete route whose live stage is
  /// being reported. It may differ between adjacent events in an AUTO race.
  @$pb.TagNumber(9)
  ConnectionRouteKind get attemptedRouteKind => $_getN(8);
  @$pb.TagNumber(9)
  set attemptedRouteKind(ConnectionRouteKind value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasAttemptedRouteKind() => $_has(8);
  @$pb.TagNumber(9)
  void clearAttemptedRouteKind() => $_clearField(9);

  /// connection_stage is a stable runtime stage identifier. Display text and
  /// localization remain owned by the client UI.
  @$pb.TagNumber(10)
  $core.String get connectionStage => $_getSZ(9);
  @$pb.TagNumber(10)
  set connectionStage($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasConnectionStage() => $_has(9);
  @$pb.TagNumber(10)
  void clearConnectionStage() => $_clearField(10);
}

enum EventEnvelope_Event {
  openSession,
  execute,
  application,
  sessionClosed,
  importPairing,
  deleteCredential,
  resourceStreamFrame,
  resourceStreamClosed,
  endpointRegistryGet,
  endpointUpsert,
  endpointDelete,
  endpointShareReceive,
  endpointShareCommit,
  sshCredentialProvision,
  connectionPolicyGet,
  connectionPolicyApply,
  connectionSnapshotGet,
  sessionInvalidate,
  endpointConnection,
  endpointDisconnect,
  endpointCloudPresenceGet,
  notSet
}

class EventEnvelope extends $pb.GeneratedMessage {
  factory EventEnvelope({
    $core.int? abiVersion,
    $fixnum.Int64? sequence,
    OpenSessionResult? openSession,
    ExecuteResult? execute,
    ApplicationEvent? application,
    SessionClosedEvent? sessionClosed,
    ImportPairingResult? importPairing,
    DeleteCredentialResult? deleteCredential,
    ResourceStreamFrame? resourceStreamFrame,
    ResourceStreamClosedEvent? resourceStreamClosed,
    EndpointRegistryGetResult? endpointRegistryGet,
    EndpointUpsertResult? endpointUpsert,
    EndpointDeleteResult? endpointDelete,
    EndpointShareReceiveResult? endpointShareReceive,
    EndpointShareCommitResult? endpointShareCommit,
    SSHCredentialProvisionResult? sshCredentialProvision,
    ConnectionPolicyGetResult? connectionPolicyGet,
    ConnectionPolicyApplyResult? connectionPolicyApply,
    ConnectionSnapshotGetResult? connectionSnapshotGet,
    SessionInvalidateResult? sessionInvalidate,
    EndpointConnectionEvent? endpointConnection,
    EndpointDisconnectResult? endpointDisconnect,
    EndpointCloudPresenceGetResult? endpointCloudPresenceGet,
  }) {
    final result = create();
    if (abiVersion != null) result.abiVersion = abiVersion;
    if (sequence != null) result.sequence = sequence;
    if (openSession != null) result.openSession = openSession;
    if (execute != null) result.execute = execute;
    if (application != null) result.application = application;
    if (sessionClosed != null) result.sessionClosed = sessionClosed;
    if (importPairing != null) result.importPairing = importPairing;
    if (deleteCredential != null) result.deleteCredential = deleteCredential;
    if (resourceStreamFrame != null)
      result.resourceStreamFrame = resourceStreamFrame;
    if (resourceStreamClosed != null)
      result.resourceStreamClosed = resourceStreamClosed;
    if (endpointRegistryGet != null)
      result.endpointRegistryGet = endpointRegistryGet;
    if (endpointUpsert != null) result.endpointUpsert = endpointUpsert;
    if (endpointDelete != null) result.endpointDelete = endpointDelete;
    if (endpointShareReceive != null)
      result.endpointShareReceive = endpointShareReceive;
    if (endpointShareCommit != null)
      result.endpointShareCommit = endpointShareCommit;
    if (sshCredentialProvision != null)
      result.sshCredentialProvision = sshCredentialProvision;
    if (connectionPolicyGet != null)
      result.connectionPolicyGet = connectionPolicyGet;
    if (connectionPolicyApply != null)
      result.connectionPolicyApply = connectionPolicyApply;
    if (connectionSnapshotGet != null)
      result.connectionSnapshotGet = connectionSnapshotGet;
    if (sessionInvalidate != null) result.sessionInvalidate = sessionInvalidate;
    if (endpointConnection != null)
      result.endpointConnection = endpointConnection;
    if (endpointDisconnect != null)
      result.endpointDisconnect = endpointDisconnect;
    if (endpointCloudPresenceGet != null)
      result.endpointCloudPresenceGet = endpointCloudPresenceGet;
    return result;
  }

  EventEnvelope._();

  factory EventEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EventEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EventEnvelope_Event>
      _EventEnvelope_EventByTag = {
    10: EventEnvelope_Event.openSession,
    11: EventEnvelope_Event.execute,
    12: EventEnvelope_Event.application,
    13: EventEnvelope_Event.sessionClosed,
    14: EventEnvelope_Event.importPairing,
    15: EventEnvelope_Event.deleteCredential,
    16: EventEnvelope_Event.resourceStreamFrame,
    17: EventEnvelope_Event.resourceStreamClosed,
    18: EventEnvelope_Event.endpointRegistryGet,
    19: EventEnvelope_Event.endpointUpsert,
    20: EventEnvelope_Event.endpointDelete,
    21: EventEnvelope_Event.endpointShareReceive,
    22: EventEnvelope_Event.endpointShareCommit,
    23: EventEnvelope_Event.sshCredentialProvision,
    24: EventEnvelope_Event.connectionPolicyGet,
    25: EventEnvelope_Event.connectionPolicyApply,
    26: EventEnvelope_Event.connectionSnapshotGet,
    27: EventEnvelope_Event.sessionInvalidate,
    28: EventEnvelope_Event.endpointConnection,
    29: EventEnvelope_Event.endpointDisconnect,
    30: EventEnvelope_Event.endpointCloudPresenceGet,
    0: EventEnvelope_Event.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EventEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      10,
      11,
      12,
      13,
      14,
      15,
      16,
      17,
      18,
      19,
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
      30
    ])
    ..aI(1, _omitFieldNames ? '' : 'abiVersion', fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'sequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<OpenSessionResult>(10, _omitFieldNames ? '' : 'openSession',
        subBuilder: OpenSessionResult.create)
    ..aOM<ExecuteResult>(11, _omitFieldNames ? '' : 'execute',
        subBuilder: ExecuteResult.create)
    ..aOM<ApplicationEvent>(12, _omitFieldNames ? '' : 'application',
        subBuilder: ApplicationEvent.create)
    ..aOM<SessionClosedEvent>(13, _omitFieldNames ? '' : 'sessionClosed',
        subBuilder: SessionClosedEvent.create)
    ..aOM<ImportPairingResult>(14, _omitFieldNames ? '' : 'importPairing',
        subBuilder: ImportPairingResult.create)
    ..aOM<DeleteCredentialResult>(15, _omitFieldNames ? '' : 'deleteCredential',
        subBuilder: DeleteCredentialResult.create)
    ..aOM<ResourceStreamFrame>(16, _omitFieldNames ? '' : 'resourceStreamFrame',
        subBuilder: ResourceStreamFrame.create)
    ..aOM<ResourceStreamClosedEvent>(
        17, _omitFieldNames ? '' : 'resourceStreamClosed',
        subBuilder: ResourceStreamClosedEvent.create)
    ..aOM<EndpointRegistryGetResult>(
        18, _omitFieldNames ? '' : 'endpointRegistryGet',
        subBuilder: EndpointRegistryGetResult.create)
    ..aOM<EndpointUpsertResult>(19, _omitFieldNames ? '' : 'endpointUpsert',
        subBuilder: EndpointUpsertResult.create)
    ..aOM<EndpointDeleteResult>(20, _omitFieldNames ? '' : 'endpointDelete',
        subBuilder: EndpointDeleteResult.create)
    ..aOM<EndpointShareReceiveResult>(
        21, _omitFieldNames ? '' : 'endpointShareReceive',
        subBuilder: EndpointShareReceiveResult.create)
    ..aOM<EndpointShareCommitResult>(
        22, _omitFieldNames ? '' : 'endpointShareCommit',
        subBuilder: EndpointShareCommitResult.create)
    ..aOM<SSHCredentialProvisionResult>(
        23, _omitFieldNames ? '' : 'sshCredentialProvision',
        subBuilder: SSHCredentialProvisionResult.create)
    ..aOM<ConnectionPolicyGetResult>(
        24, _omitFieldNames ? '' : 'connectionPolicyGet',
        subBuilder: ConnectionPolicyGetResult.create)
    ..aOM<ConnectionPolicyApplyResult>(
        25, _omitFieldNames ? '' : 'connectionPolicyApply',
        subBuilder: ConnectionPolicyApplyResult.create)
    ..aOM<ConnectionSnapshotGetResult>(
        26, _omitFieldNames ? '' : 'connectionSnapshotGet',
        subBuilder: ConnectionSnapshotGetResult.create)
    ..aOM<SessionInvalidateResult>(
        27, _omitFieldNames ? '' : 'sessionInvalidate',
        subBuilder: SessionInvalidateResult.create)
    ..aOM<EndpointConnectionEvent>(
        28, _omitFieldNames ? '' : 'endpointConnection',
        subBuilder: EndpointConnectionEvent.create)
    ..aOM<EndpointDisconnectResult>(
        29, _omitFieldNames ? '' : 'endpointDisconnect',
        subBuilder: EndpointDisconnectResult.create)
    ..aOM<EndpointCloudPresenceGetResult>(
        30, _omitFieldNames ? '' : 'endpointCloudPresenceGet',
        subBuilder: EndpointCloudPresenceGetResult.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventEnvelope copyWith(void Function(EventEnvelope) updates) =>
      super.copyWith((message) => updates(message as EventEnvelope))
          as EventEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EventEnvelope create() => EventEnvelope._();
  @$core.override
  EventEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EventEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EventEnvelope>(create);
  static EventEnvelope? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  @$pb.TagNumber(19)
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
  EventEnvelope_Event whichEvent() =>
      _EventEnvelope_EventByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  @$pb.TagNumber(19)
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
  void clearEvent() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.int get abiVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set abiVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAbiVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearAbiVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get sequence => $_getI64(1);
  @$pb.TagNumber(2)
  set sequence($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSequence() => $_has(1);
  @$pb.TagNumber(2)
  void clearSequence() => $_clearField(2);

  @$pb.TagNumber(10)
  OpenSessionResult get openSession => $_getN(2);
  @$pb.TagNumber(10)
  set openSession(OpenSessionResult value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasOpenSession() => $_has(2);
  @$pb.TagNumber(10)
  void clearOpenSession() => $_clearField(10);
  @$pb.TagNumber(10)
  OpenSessionResult ensureOpenSession() => $_ensure(2);

  @$pb.TagNumber(11)
  ExecuteResult get execute => $_getN(3);
  @$pb.TagNumber(11)
  set execute(ExecuteResult value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasExecute() => $_has(3);
  @$pb.TagNumber(11)
  void clearExecute() => $_clearField(11);
  @$pb.TagNumber(11)
  ExecuteResult ensureExecute() => $_ensure(3);

  @$pb.TagNumber(12)
  ApplicationEvent get application => $_getN(4);
  @$pb.TagNumber(12)
  set application(ApplicationEvent value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasApplication() => $_has(4);
  @$pb.TagNumber(12)
  void clearApplication() => $_clearField(12);
  @$pb.TagNumber(12)
  ApplicationEvent ensureApplication() => $_ensure(4);

  @$pb.TagNumber(13)
  SessionClosedEvent get sessionClosed => $_getN(5);
  @$pb.TagNumber(13)
  set sessionClosed(SessionClosedEvent value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasSessionClosed() => $_has(5);
  @$pb.TagNumber(13)
  void clearSessionClosed() => $_clearField(13);
  @$pb.TagNumber(13)
  SessionClosedEvent ensureSessionClosed() => $_ensure(5);

  @$pb.TagNumber(14)
  ImportPairingResult get importPairing => $_getN(6);
  @$pb.TagNumber(14)
  set importPairing(ImportPairingResult value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasImportPairing() => $_has(6);
  @$pb.TagNumber(14)
  void clearImportPairing() => $_clearField(14);
  @$pb.TagNumber(14)
  ImportPairingResult ensureImportPairing() => $_ensure(6);

  @$pb.TagNumber(15)
  DeleteCredentialResult get deleteCredential => $_getN(7);
  @$pb.TagNumber(15)
  set deleteCredential(DeleteCredentialResult value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasDeleteCredential() => $_has(7);
  @$pb.TagNumber(15)
  void clearDeleteCredential() => $_clearField(15);
  @$pb.TagNumber(15)
  DeleteCredentialResult ensureDeleteCredential() => $_ensure(7);

  @$pb.TagNumber(16)
  ResourceStreamFrame get resourceStreamFrame => $_getN(8);
  @$pb.TagNumber(16)
  set resourceStreamFrame(ResourceStreamFrame value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasResourceStreamFrame() => $_has(8);
  @$pb.TagNumber(16)
  void clearResourceStreamFrame() => $_clearField(16);
  @$pb.TagNumber(16)
  ResourceStreamFrame ensureResourceStreamFrame() => $_ensure(8);

  @$pb.TagNumber(17)
  ResourceStreamClosedEvent get resourceStreamClosed => $_getN(9);
  @$pb.TagNumber(17)
  set resourceStreamClosed(ResourceStreamClosedEvent value) =>
      $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasResourceStreamClosed() => $_has(9);
  @$pb.TagNumber(17)
  void clearResourceStreamClosed() => $_clearField(17);
  @$pb.TagNumber(17)
  ResourceStreamClosedEvent ensureResourceStreamClosed() => $_ensure(9);

  @$pb.TagNumber(18)
  EndpointRegistryGetResult get endpointRegistryGet => $_getN(10);
  @$pb.TagNumber(18)
  set endpointRegistryGet(EndpointRegistryGetResult value) =>
      $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasEndpointRegistryGet() => $_has(10);
  @$pb.TagNumber(18)
  void clearEndpointRegistryGet() => $_clearField(18);
  @$pb.TagNumber(18)
  EndpointRegistryGetResult ensureEndpointRegistryGet() => $_ensure(10);

  @$pb.TagNumber(19)
  EndpointUpsertResult get endpointUpsert => $_getN(11);
  @$pb.TagNumber(19)
  set endpointUpsert(EndpointUpsertResult value) => $_setField(19, value);
  @$pb.TagNumber(19)
  $core.bool hasEndpointUpsert() => $_has(11);
  @$pb.TagNumber(19)
  void clearEndpointUpsert() => $_clearField(19);
  @$pb.TagNumber(19)
  EndpointUpsertResult ensureEndpointUpsert() => $_ensure(11);

  @$pb.TagNumber(20)
  EndpointDeleteResult get endpointDelete => $_getN(12);
  @$pb.TagNumber(20)
  set endpointDelete(EndpointDeleteResult value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasEndpointDelete() => $_has(12);
  @$pb.TagNumber(20)
  void clearEndpointDelete() => $_clearField(20);
  @$pb.TagNumber(20)
  EndpointDeleteResult ensureEndpointDelete() => $_ensure(12);

  @$pb.TagNumber(21)
  EndpointShareReceiveResult get endpointShareReceive => $_getN(13);
  @$pb.TagNumber(21)
  set endpointShareReceive(EndpointShareReceiveResult value) =>
      $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasEndpointShareReceive() => $_has(13);
  @$pb.TagNumber(21)
  void clearEndpointShareReceive() => $_clearField(21);
  @$pb.TagNumber(21)
  EndpointShareReceiveResult ensureEndpointShareReceive() => $_ensure(13);

  @$pb.TagNumber(22)
  EndpointShareCommitResult get endpointShareCommit => $_getN(14);
  @$pb.TagNumber(22)
  set endpointShareCommit(EndpointShareCommitResult value) =>
      $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasEndpointShareCommit() => $_has(14);
  @$pb.TagNumber(22)
  void clearEndpointShareCommit() => $_clearField(22);
  @$pb.TagNumber(22)
  EndpointShareCommitResult ensureEndpointShareCommit() => $_ensure(14);

  @$pb.TagNumber(23)
  SSHCredentialProvisionResult get sshCredentialProvision => $_getN(15);
  @$pb.TagNumber(23)
  set sshCredentialProvision(SSHCredentialProvisionResult value) =>
      $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasSshCredentialProvision() => $_has(15);
  @$pb.TagNumber(23)
  void clearSshCredentialProvision() => $_clearField(23);
  @$pb.TagNumber(23)
  SSHCredentialProvisionResult ensureSshCredentialProvision() => $_ensure(15);

  @$pb.TagNumber(24)
  ConnectionPolicyGetResult get connectionPolicyGet => $_getN(16);
  @$pb.TagNumber(24)
  set connectionPolicyGet(ConnectionPolicyGetResult value) =>
      $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasConnectionPolicyGet() => $_has(16);
  @$pb.TagNumber(24)
  void clearConnectionPolicyGet() => $_clearField(24);
  @$pb.TagNumber(24)
  ConnectionPolicyGetResult ensureConnectionPolicyGet() => $_ensure(16);

  @$pb.TagNumber(25)
  ConnectionPolicyApplyResult get connectionPolicyApply => $_getN(17);
  @$pb.TagNumber(25)
  set connectionPolicyApply(ConnectionPolicyApplyResult value) =>
      $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasConnectionPolicyApply() => $_has(17);
  @$pb.TagNumber(25)
  void clearConnectionPolicyApply() => $_clearField(25);
  @$pb.TagNumber(25)
  ConnectionPolicyApplyResult ensureConnectionPolicyApply() => $_ensure(17);

  @$pb.TagNumber(26)
  ConnectionSnapshotGetResult get connectionSnapshotGet => $_getN(18);
  @$pb.TagNumber(26)
  set connectionSnapshotGet(ConnectionSnapshotGetResult value) =>
      $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasConnectionSnapshotGet() => $_has(18);
  @$pb.TagNumber(26)
  void clearConnectionSnapshotGet() => $_clearField(26);
  @$pb.TagNumber(26)
  ConnectionSnapshotGetResult ensureConnectionSnapshotGet() => $_ensure(18);

  @$pb.TagNumber(27)
  SessionInvalidateResult get sessionInvalidate => $_getN(19);
  @$pb.TagNumber(27)
  set sessionInvalidate(SessionInvalidateResult value) => $_setField(27, value);
  @$pb.TagNumber(27)
  $core.bool hasSessionInvalidate() => $_has(19);
  @$pb.TagNumber(27)
  void clearSessionInvalidate() => $_clearField(27);
  @$pb.TagNumber(27)
  SessionInvalidateResult ensureSessionInvalidate() => $_ensure(19);

  @$pb.TagNumber(28)
  EndpointConnectionEvent get endpointConnection => $_getN(20);
  @$pb.TagNumber(28)
  set endpointConnection(EndpointConnectionEvent value) =>
      $_setField(28, value);
  @$pb.TagNumber(28)
  $core.bool hasEndpointConnection() => $_has(20);
  @$pb.TagNumber(28)
  void clearEndpointConnection() => $_clearField(28);
  @$pb.TagNumber(28)
  EndpointConnectionEvent ensureEndpointConnection() => $_ensure(20);

  @$pb.TagNumber(29)
  EndpointDisconnectResult get endpointDisconnect => $_getN(21);
  @$pb.TagNumber(29)
  set endpointDisconnect(EndpointDisconnectResult value) =>
      $_setField(29, value);
  @$pb.TagNumber(29)
  $core.bool hasEndpointDisconnect() => $_has(21);
  @$pb.TagNumber(29)
  void clearEndpointDisconnect() => $_clearField(29);
  @$pb.TagNumber(29)
  EndpointDisconnectResult ensureEndpointDisconnect() => $_ensure(21);

  @$pb.TagNumber(30)
  EndpointCloudPresenceGetResult get endpointCloudPresenceGet => $_getN(22);
  @$pb.TagNumber(30)
  set endpointCloudPresenceGet(EndpointCloudPresenceGetResult value) =>
      $_setField(30, value);
  @$pb.TagNumber(30)
  $core.bool hasEndpointCloudPresenceGet() => $_has(22);
  @$pb.TagNumber(30)
  void clearEndpointCloudPresenceGet() => $_clearField(30);
  @$pb.TagNumber(30)
  EndpointCloudPresenceGetResult ensureEndpointCloudPresenceGet() =>
      $_ensure(22);
}

class CredentialResolveRequest extends $pb.GeneratedMessage {
  factory CredentialResolveRequest({
    $core.String? endpointId,
    $core.String? credentialRef,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (credentialRef != null) result.credentialRef = credentialRef;
    return result;
  }

  CredentialResolveRequest._();

  factory CredentialResolveRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialResolveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialResolveRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'credentialRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialResolveRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialResolveRequest copyWith(
          void Function(CredentialResolveRequest) updates) =>
      super.copyWith((message) => updates(message as CredentialResolveRequest))
          as CredentialResolveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialResolveRequest create() => CredentialResolveRequest._();
  @$core.override
  CredentialResolveRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialResolveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialResolveRequest>(create);
  static CredentialResolveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get credentialRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set credentialRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCredentialRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearCredentialRef() => $_clearField(2);
}

class CredentialPrepareRequest extends $pb.GeneratedMessage {
  factory CredentialPrepareRequest({
    $core.String? endpointId,
    $core.String? credentialRef,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (credentialRef != null) result.credentialRef = credentialRef;
    return result;
  }

  CredentialPrepareRequest._();

  factory CredentialPrepareRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialPrepareRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialPrepareRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'credentialRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialPrepareRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialPrepareRequest copyWith(
          void Function(CredentialPrepareRequest) updates) =>
      super.copyWith((message) => updates(message as CredentialPrepareRequest))
          as CredentialPrepareRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialPrepareRequest create() => CredentialPrepareRequest._();
  @$core.override
  CredentialPrepareRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialPrepareRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialPrepareRequest>(create);
  static CredentialPrepareRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get credentialRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set credentialRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCredentialRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearCredentialRef() => $_clearField(2);
}

class CredentialDeleteRequest extends $pb.GeneratedMessage {
  factory CredentialDeleteRequest({
    $core.String? credentialRef,
  }) {
    final result = create();
    if (credentialRef != null) result.credentialRef = credentialRef;
    return result;
  }

  CredentialDeleteRequest._();

  factory CredentialDeleteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialDeleteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialDeleteRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'credentialRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialDeleteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialDeleteRequest copyWith(
          void Function(CredentialDeleteRequest) updates) =>
      super.copyWith((message) => updates(message as CredentialDeleteRequest))
          as CredentialDeleteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialDeleteRequest create() => CredentialDeleteRequest._();
  @$core.override
  CredentialDeleteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialDeleteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialDeleteRequest>(create);
  static CredentialDeleteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get credentialRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set credentialRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCredentialRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearCredentialRef() => $_clearField(1);
}

class CredentialBindRequest extends $pb.GeneratedMessage {
  factory CredentialBindRequest({
    $core.String? endpointId,
    $core.String? credentialRef,
    $core.String? capabilityGrant,
    $core.List<$core.int>? cloudRouteGrant,
    $core.List<$core.int>? cloudEdgeLocator,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (capabilityGrant != null) result.capabilityGrant = capabilityGrant;
    if (cloudRouteGrant != null) result.cloudRouteGrant = cloudRouteGrant;
    if (cloudEdgeLocator != null) result.cloudEdgeLocator = cloudEdgeLocator;
    return result;
  }

  CredentialBindRequest._();

  factory CredentialBindRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialBindRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialBindRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'credentialRef')
    ..aOS(3, _omitFieldNames ? '' : 'capabilityGrant')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'cloudRouteGrant', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'cloudEdgeLocator', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialBindRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialBindRequest copyWith(
          void Function(CredentialBindRequest) updates) =>
      super.copyWith((message) => updates(message as CredentialBindRequest))
          as CredentialBindRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialBindRequest create() => CredentialBindRequest._();
  @$core.override
  CredentialBindRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialBindRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialBindRequest>(create);
  static CredentialBindRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get credentialRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set credentialRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCredentialRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearCredentialRef() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get capabilityGrant => $_getSZ(2);
  @$pb.TagNumber(3)
  set capabilityGrant($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCapabilityGrant() => $_has(2);
  @$pb.TagNumber(3)
  void clearCapabilityGrant() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get cloudRouteGrant => $_getN(3);
  @$pb.TagNumber(4)
  set cloudRouteGrant($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCloudRouteGrant() => $_has(3);
  @$pb.TagNumber(4)
  void clearCloudRouteGrant() => $_clearField(4);

  /// cloud_edge_locator 是 enrollment、配对或 Controller 纠偏返回的 EdgeLocator protobuf；不是授权真值。
  @$pb.TagNumber(5)
  $core.List<$core.int> get cloudEdgeLocator => $_getN(4);
  @$pb.TagNumber(5)
  set cloudEdgeLocator($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCloudEdgeLocator() => $_has(4);
  @$pb.TagNumber(5)
  void clearCloudEdgeLocator() => $_clearField(5);
}

class CredentialRecord extends $pb.GeneratedMessage {
  factory CredentialRecord({
    $core.String? endpointId,
    $core.String? credentialRef,
    $core.List<$core.int>? publicKey,
    $core.String? keyFingerprint,
    $core.String? capabilityGrant,
    $core.bool? newlyCreated,
    $core.List<$core.int>? cloudRouteGrant,
    $core.List<$core.int>? cloudEdgeLocator,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (publicKey != null) result.publicKey = publicKey;
    if (keyFingerprint != null) result.keyFingerprint = keyFingerprint;
    if (capabilityGrant != null) result.capabilityGrant = capabilityGrant;
    if (newlyCreated != null) result.newlyCreated = newlyCreated;
    if (cloudRouteGrant != null) result.cloudRouteGrant = cloudRouteGrant;
    if (cloudEdgeLocator != null) result.cloudEdgeLocator = cloudEdgeLocator;
    return result;
  }

  CredentialRecord._();

  factory CredentialRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialRecord',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'credentialRef')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'publicKey', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'keyFingerprint')
    ..aOS(5, _omitFieldNames ? '' : 'capabilityGrant')
    ..aOB(6, _omitFieldNames ? '' : 'newlyCreated')
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'cloudRouteGrant', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'cloudEdgeLocator', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialRecord copyWith(void Function(CredentialRecord) updates) =>
      super.copyWith((message) => updates(message as CredentialRecord))
          as CredentialRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialRecord create() => CredentialRecord._();
  @$core.override
  CredentialRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialRecord>(create);
  static CredentialRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get credentialRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set credentialRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCredentialRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearCredentialRef() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get publicKey => $_getN(2);
  @$pb.TagNumber(3)
  set publicKey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearPublicKey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get keyFingerprint => $_getSZ(3);
  @$pb.TagNumber(4)
  set keyFingerprint($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasKeyFingerprint() => $_has(3);
  @$pb.TagNumber(4)
  void clearKeyFingerprint() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get capabilityGrant => $_getSZ(4);
  @$pb.TagNumber(5)
  set capabilityGrant($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCapabilityGrant() => $_has(4);
  @$pb.TagNumber(5)
  void clearCapabilityGrant() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get newlyCreated => $_getBF(5);
  @$pb.TagNumber(6)
  set newlyCreated($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasNewlyCreated() => $_has(5);
  @$pb.TagNumber(6)
  void clearNewlyCreated() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get cloudRouteGrant => $_getN(6);
  @$pb.TagNumber(7)
  set cloudRouteGrant($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCloudRouteGrant() => $_has(6);
  @$pb.TagNumber(7)
  void clearCloudRouteGrant() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.int> get cloudEdgeLocator => $_getN(7);
  @$pb.TagNumber(8)
  set cloudEdgeLocator($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCloudEdgeLocator() => $_has(7);
  @$pb.TagNumber(8)
  void clearCloudEdgeLocator() => $_clearField(8);
}

class CredentialSignRequest extends $pb.GeneratedMessage {
  factory CredentialSignRequest({
    $core.String? credentialRef,
    $core.List<$core.int>? payload,
  }) {
    final result = create();
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (payload != null) result.payload = payload;
    return result;
  }

  CredentialSignRequest._();

  factory CredentialSignRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialSignRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialSignRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'credentialRef')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'payload', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialSignRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialSignRequest copyWith(
          void Function(CredentialSignRequest) updates) =>
      super.copyWith((message) => updates(message as CredentialSignRequest))
          as CredentialSignRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialSignRequest create() => CredentialSignRequest._();
  @$core.override
  CredentialSignRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialSignRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialSignRequest>(create);
  static CredentialSignRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get credentialRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set credentialRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCredentialRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearCredentialRef() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get payload => $_getN(1);
  @$pb.TagNumber(2)
  set payload($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPayload() => $_has(1);
  @$pb.TagNumber(2)
  void clearPayload() => $_clearField(2);
}

class CredentialSignResponse extends $pb.GeneratedMessage {
  factory CredentialSignResponse({
    $core.List<$core.int>? signature,
  }) {
    final result = create();
    if (signature != null) result.signature = signature;
    return result;
  }

  CredentialSignResponse._();

  factory CredentialSignResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CredentialSignResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CredentialSignResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialSignResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CredentialSignResponse copyWith(
          void Function(CredentialSignResponse) updates) =>
      super.copyWith((message) => updates(message as CredentialSignResponse))
          as CredentialSignResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CredentialSignResponse create() => CredentialSignResponse._();
  @$core.override
  CredentialSignResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CredentialSignResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CredentialSignResponse>(create);
  static CredentialSignResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get signature => $_getN(0);
  @$pb.TagNumber(1)
  set signature($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSignature() => $_has(0);
  @$pb.TagNumber(1)
  void clearSignature() => $_clearField(1);
}

/// CloudProfileResolveRequest 只解析本机账号/构建配置中的 Controller TLS locator。
/// 平台不得在该请求内执行 Directory、ClientGateway、WebRTC 或认证逻辑。
class CloudProfileResolveRequest extends $pb.GeneratedMessage {
  factory CloudProfileResolveRequest({
    $core.String? accountProfileRef,
  }) {
    final result = create();
    if (accountProfileRef != null) result.accountProfileRef = accountProfileRef;
    return result;
  }

  CloudProfileResolveRequest._();

  factory CloudProfileResolveRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudProfileResolveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudProfileResolveRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountProfileRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudProfileResolveRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudProfileResolveRequest copyWith(
          void Function(CloudProfileResolveRequest) updates) =>
      super.copyWith(
              (message) => updates(message as CloudProfileResolveRequest))
          as CloudProfileResolveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudProfileResolveRequest create() => CloudProfileResolveRequest._();
  @$core.override
  CloudProfileResolveRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudProfileResolveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudProfileResolveRequest>(create);
  static CloudProfileResolveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountProfileRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountProfileRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountProfileRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountProfileRef() => $_clearField(1);
}

class CloudProfileRecord extends $pb.GeneratedMessage {
  factory CloudProfileRecord({
    $core.String? accountProfileRef,
    $core.String? controllerAddress,
    $core.String? controllerServerName,
    $core.List<$core.int>? controllerCaPem,
  }) {
    final result = create();
    if (accountProfileRef != null) result.accountProfileRef = accountProfileRef;
    if (controllerAddress != null) result.controllerAddress = controllerAddress;
    if (controllerServerName != null)
      result.controllerServerName = controllerServerName;
    if (controllerCaPem != null) result.controllerCaPem = controllerCaPem;
    return result;
  }

  CloudProfileRecord._();

  factory CloudProfileRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudProfileRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudProfileRecord',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountProfileRef')
    ..aOS(2, _omitFieldNames ? '' : 'controllerAddress')
    ..aOS(3, _omitFieldNames ? '' : 'controllerServerName')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'controllerCaPem', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudProfileRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudProfileRecord copyWith(void Function(CloudProfileRecord) updates) =>
      super.copyWith((message) => updates(message as CloudProfileRecord))
          as CloudProfileRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudProfileRecord create() => CloudProfileRecord._();
  @$core.override
  CloudProfileRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudProfileRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudProfileRecord>(create);
  static CloudProfileRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountProfileRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountProfileRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountProfileRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountProfileRef() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get controllerAddress => $_getSZ(1);
  @$pb.TagNumber(2)
  set controllerAddress($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasControllerAddress() => $_has(1);
  @$pb.TagNumber(2)
  void clearControllerAddress() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get controllerServerName => $_getSZ(2);
  @$pb.TagNumber(3)
  set controllerServerName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasControllerServerName() => $_has(2);
  @$pb.TagNumber(3)
  void clearControllerServerName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get controllerCaPem => $_getN(3);
  @$pb.TagNumber(4)
  set controllerCaPem($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasControllerCaPem() => $_has(3);
  @$pb.TagNumber(4)
  void clearControllerCaPem() => $_clearField(4);
}

/// SSHCredentialLookupRequest 查询或显式创建一个平台不可导出的 SSH signer。
/// create_if_missing=false 只能查询，planner 不得因为能力探测而创建新 key。
class SSHCredentialLookupRequest extends $pb.GeneratedMessage {
  factory SSHCredentialLookupRequest({
    $core.String? credentialRef,
    $core.bool? createIfMissing,
  }) {
    final result = create();
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (createIfMissing != null) result.createIfMissing = createIfMissing;
    return result;
  }

  SSHCredentialLookupRequest._();

  factory SSHCredentialLookupRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialLookupRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialLookupRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'credentialRef')
    ..aOB(2, _omitFieldNames ? '' : 'createIfMissing')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialLookupRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialLookupRequest copyWith(
          void Function(SSHCredentialLookupRequest) updates) =>
      super.copyWith(
              (message) => updates(message as SSHCredentialLookupRequest))
          as SSHCredentialLookupRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialLookupRequest create() => SSHCredentialLookupRequest._();
  @$core.override
  SSHCredentialLookupRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialLookupRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialLookupRequest>(create);
  static SSHCredentialLookupRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get credentialRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set credentialRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCredentialRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearCredentialRef() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get createIfMissing => $_getBF(1);
  @$pb.TagNumber(2)
  set createIfMissing($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCreateIfMissing() => $_has(1);
  @$pb.TagNumber(2)
  void clearCreateIfMissing() => $_clearField(2);
}

class SSHCredentialDeleteRequest extends $pb.GeneratedMessage {
  factory SSHCredentialDeleteRequest({
    $core.String? credentialRef,
  }) {
    final result = create();
    if (credentialRef != null) result.credentialRef = credentialRef;
    return result;
  }

  SSHCredentialDeleteRequest._();

  factory SSHCredentialDeleteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialDeleteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialDeleteRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'credentialRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialDeleteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialDeleteRequest copyWith(
          void Function(SSHCredentialDeleteRequest) updates) =>
      super.copyWith(
              (message) => updates(message as SSHCredentialDeleteRequest))
          as SSHCredentialDeleteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialDeleteRequest create() => SSHCredentialDeleteRequest._();
  @$core.override
  SSHCredentialDeleteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialDeleteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialDeleteRequest>(create);
  static SSHCredentialDeleteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get credentialRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set credentialRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCredentialRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearCredentialRef() => $_clearField(1);
}

/// SSHCredentialRecord 只返回 PKIX 公钥；private key handle 和 key body 不跨平台边界。
class SSHCredentialRecord extends $pb.GeneratedMessage {
  factory SSHCredentialRecord({
    $core.String? credentialRef,
    $core.List<$core.int>? publicKeyPkix,
    $core.bool? newlyCreated,
  }) {
    final result = create();
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (publicKeyPkix != null) result.publicKeyPkix = publicKeyPkix;
    if (newlyCreated != null) result.newlyCreated = newlyCreated;
    return result;
  }

  SSHCredentialRecord._();

  factory SSHCredentialRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialRecord',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'credentialRef')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'publicKeyPkix', $pb.PbFieldType.OY)
    ..aOB(3, _omitFieldNames ? '' : 'newlyCreated')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialRecord copyWith(void Function(SSHCredentialRecord) updates) =>
      super.copyWith((message) => updates(message as SSHCredentialRecord))
          as SSHCredentialRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialRecord create() => SSHCredentialRecord._();
  @$core.override
  SSHCredentialRecord createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialRecord getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialRecord>(create);
  static SSHCredentialRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get credentialRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set credentialRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCredentialRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearCredentialRef() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get publicKeyPkix => $_getN(1);
  @$pb.TagNumber(2)
  set publicKeyPkix($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPublicKeyPkix() => $_has(1);
  @$pb.TagNumber(2)
  void clearPublicKeyPkix() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get newlyCreated => $_getBF(2);
  @$pb.TagNumber(3)
  set newlyCreated($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNewlyCreated() => $_has(2);
  @$pb.TagNumber(3)
  void clearNewlyCreated() => $_clearField(3);
}

/// SSHCredentialSignRequest 让平台 signer 对 Go SSH 实现已计算的 digest 签名。
/// 当前 Android contract 只允许 ECDSA P-256 + SHA-256。
class SSHCredentialSignRequest extends $pb.GeneratedMessage {
  factory SSHCredentialSignRequest({
    $core.String? credentialRef,
    $core.List<$core.int>? digest,
    $core.String? hash,
  }) {
    final result = create();
    if (credentialRef != null) result.credentialRef = credentialRef;
    if (digest != null) result.digest = digest;
    if (hash != null) result.hash = hash;
    return result;
  }

  SSHCredentialSignRequest._();

  factory SSHCredentialSignRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialSignRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialSignRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'credentialRef')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'digest', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'hash')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialSignRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialSignRequest copyWith(
          void Function(SSHCredentialSignRequest) updates) =>
      super.copyWith((message) => updates(message as SSHCredentialSignRequest))
          as SSHCredentialSignRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialSignRequest create() => SSHCredentialSignRequest._();
  @$core.override
  SSHCredentialSignRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialSignRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialSignRequest>(create);
  static SSHCredentialSignRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get credentialRef => $_getSZ(0);
  @$pb.TagNumber(1)
  set credentialRef($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCredentialRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearCredentialRef() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get digest => $_getN(1);
  @$pb.TagNumber(2)
  set digest($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDigest() => $_has(1);
  @$pb.TagNumber(2)
  void clearDigest() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get hash => $_getSZ(2);
  @$pb.TagNumber(3)
  set hash($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasHash() => $_has(2);
  @$pb.TagNumber(3)
  void clearHash() => $_clearField(3);
}

class SSHCredentialSignResponse extends $pb.GeneratedMessage {
  factory SSHCredentialSignResponse({
    $core.List<$core.int>? signature,
  }) {
    final result = create();
    if (signature != null) result.signature = signature;
    return result;
  }

  SSHCredentialSignResponse._();

  factory SSHCredentialSignResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SSHCredentialSignResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SSHCredentialSignResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialSignResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SSHCredentialSignResponse copyWith(
          void Function(SSHCredentialSignResponse) updates) =>
      super.copyWith((message) => updates(message as SSHCredentialSignResponse))
          as SSHCredentialSignResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SSHCredentialSignResponse create() => SSHCredentialSignResponse._();
  @$core.override
  SSHCredentialSignResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SSHCredentialSignResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SSHCredentialSignResponse>(create);
  static SSHCredentialSignResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get signature => $_getN(0);
  @$pb.TagNumber(1)
  set signature($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSignature() => $_has(0);
  @$pb.TagNumber(1)
  void clearSignature() => $_clearField(1);
}

/// EndpointRegistryLoad/Store 是平台 opaque blob primitive；平台不得解析 registry 字段或建立第二份索引。
class EndpointRegistryLoadRequest extends $pb.GeneratedMessage {
  factory EndpointRegistryLoadRequest() => create();

  EndpointRegistryLoadRequest._();

  factory EndpointRegistryLoadRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRegistryLoadRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRegistryLoadRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryLoadRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryLoadRequest copyWith(
          void Function(EndpointRegistryLoadRequest) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointRegistryLoadRequest))
          as EndpointRegistryLoadRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRegistryLoadRequest create() =>
      EndpointRegistryLoadRequest._();
  @$core.override
  EndpointRegistryLoadRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRegistryLoadRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRegistryLoadRequest>(create);
  static EndpointRegistryLoadRequest? _defaultInstance;
}

class EndpointRegistryStoreRequest extends $pb.GeneratedMessage {
  factory EndpointRegistryStoreRequest({
    $core.List<$core.int>? registryProto,
    $core.Iterable<$core.String>? deleteCredentialRefs,
  }) {
    final result = create();
    if (registryProto != null) result.registryProto = registryProto;
    if (deleteCredentialRefs != null)
      result.deleteCredentialRefs.addAll(deleteCredentialRefs);
    return result;
  }

  EndpointRegistryStoreRequest._();

  factory EndpointRegistryStoreRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRegistryStoreRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRegistryStoreRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'registryProto', $pb.PbFieldType.OY)
    ..pPS(2, _omitFieldNames ? '' : 'deleteCredentialRefs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryStoreRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryStoreRequest copyWith(
          void Function(EndpointRegistryStoreRequest) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointRegistryStoreRequest))
          as EndpointRegistryStoreRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRegistryStoreRequest create() =>
      EndpointRegistryStoreRequest._();
  @$core.override
  EndpointRegistryStoreRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRegistryStoreRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRegistryStoreRequest>(create);
  static EndpointRegistryStoreRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get registryProto => $_getN(0);
  @$pb.TagNumber(1)
  set registryProto($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRegistryProto() => $_has(0);
  @$pb.TagNumber(1)
  void clearRegistryProto() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get deleteCredentialRefs => $_getList(1);
}

class EndpointRegistryLoaded extends $pb.GeneratedMessage {
  factory EndpointRegistryLoaded({
    $core.List<$core.int>? registryProto,
  }) {
    final result = create();
    if (registryProto != null) result.registryProto = registryProto;
    return result;
  }

  EndpointRegistryLoaded._();

  factory EndpointRegistryLoaded.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRegistryLoaded.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRegistryLoaded',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'registryProto', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryLoaded clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRegistryLoaded copyWith(
          void Function(EndpointRegistryLoaded) updates) =>
      super.copyWith((message) => updates(message as EndpointRegistryLoaded))
          as EndpointRegistryLoaded;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRegistryLoaded create() => EndpointRegistryLoaded._();
  @$core.override
  EndpointRegistryLoaded createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRegistryLoaded getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRegistryLoaded>(create);
  static EndpointRegistryLoaded? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get registryProto => $_getN(0);
  @$pb.TagNumber(1)
  set registryProto($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRegistryProto() => $_has(0);
  @$pb.TagNumber(1)
  void clearRegistryProto() => $_clearField(1);
}

/// LocalDiscoveryLookupRequest asks the platform-owned DNS-SD cache for
/// short-lived locators matching an already pinned daemon identity.
class LocalDiscoveryLookupRequest extends $pb.GeneratedMessage {
  factory LocalDiscoveryLookupRequest({
    $core.String? deviceId,
    $core.String? deviceFingerprint,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceFingerprint != null) result.deviceFingerprint = deviceFingerprint;
    return result;
  }

  LocalDiscoveryLookupRequest._();

  factory LocalDiscoveryLookupRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LocalDiscoveryLookupRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LocalDiscoveryLookupRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceFingerprint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalDiscoveryLookupRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalDiscoveryLookupRequest copyWith(
          void Function(LocalDiscoveryLookupRequest) updates) =>
      super.copyWith(
              (message) => updates(message as LocalDiscoveryLookupRequest))
          as LocalDiscoveryLookupRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LocalDiscoveryLookupRequest create() =>
      LocalDiscoveryLookupRequest._();
  @$core.override
  LocalDiscoveryLookupRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LocalDiscoveryLookupRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LocalDiscoveryLookupRequest>(create);
  static LocalDiscoveryLookupRequest? _defaultInstance;

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
}

class LocalDiscoveryCandidate extends $pb.GeneratedMessage {
  factory LocalDiscoveryCandidate({
    $core.String? address,
    $core.int? port,
    $core.int? protocolVersion,
    $fixnum.Int64? expiresAtUnixNano,
    $fixnum.Int64? networkHandle,
  }) {
    final result = create();
    if (address != null) result.address = address;
    if (port != null) result.port = port;
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    if (networkHandle != null) result.networkHandle = networkHandle;
    return result;
  }

  LocalDiscoveryCandidate._();

  factory LocalDiscoveryCandidate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LocalDiscoveryCandidate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LocalDiscoveryCandidate',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'address')
    ..aI(2, _omitFieldNames ? '' : 'port', fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aInt64(4, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'networkHandle', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalDiscoveryCandidate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalDiscoveryCandidate copyWith(
          void Function(LocalDiscoveryCandidate) updates) =>
      super.copyWith((message) => updates(message as LocalDiscoveryCandidate))
          as LocalDiscoveryCandidate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LocalDiscoveryCandidate create() => LocalDiscoveryCandidate._();
  @$core.override
  LocalDiscoveryCandidate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LocalDiscoveryCandidate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LocalDiscoveryCandidate>(create);
  static LocalDiscoveryCandidate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get address => $_getSZ(0);
  @$pb.TagNumber(1)
  set address($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAddress() => $_has(0);
  @$pb.TagNumber(1)
  void clearAddress() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get port => $_getIZ(1);
  @$pb.TagNumber(2)
  set port($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPort() => $_has(1);
  @$pb.TagNumber(2)
  void clearPort() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get protocolVersion => $_getIZ(2);
  @$pb.TagNumber(3)
  set protocolVersion($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasProtocolVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearProtocolVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(3);
  @$pb.TagNumber(4)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasExpiresAtUnixNano() => $_has(3);
  @$pb.TagNumber(4)
  void clearExpiresAtUnixNano() => $_clearField(4);

  /// network_handle identifies the Android Network that delivered this mDNS
  /// candidate. It is transient routing metadata, never endpoint identity.
  @$pb.TagNumber(5)
  $fixnum.Int64 get networkHandle => $_getI64(4);
  @$pb.TagNumber(5)
  set networkHandle($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasNetworkHandle() => $_has(4);
  @$pb.TagNumber(5)
  void clearNetworkHandle() => $_clearField(5);
}

class LocalDiscoveryLookupResult extends $pb.GeneratedMessage {
  factory LocalDiscoveryLookupResult({
    $core.Iterable<LocalDiscoveryCandidate>? candidates,
  }) {
    final result = create();
    if (candidates != null) result.candidates.addAll(candidates);
    return result;
  }

  LocalDiscoveryLookupResult._();

  factory LocalDiscoveryLookupResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LocalDiscoveryLookupResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LocalDiscoveryLookupResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..pPM<LocalDiscoveryCandidate>(1, _omitFieldNames ? '' : 'candidates',
        subBuilder: LocalDiscoveryCandidate.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalDiscoveryLookupResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocalDiscoveryLookupResult copyWith(
          void Function(LocalDiscoveryLookupResult) updates) =>
      super.copyWith(
              (message) => updates(message as LocalDiscoveryLookupResult))
          as LocalDiscoveryLookupResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LocalDiscoveryLookupResult create() => LocalDiscoveryLookupResult._();
  @$core.override
  LocalDiscoveryLookupResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LocalDiscoveryLookupResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LocalDiscoveryLookupResult>(create);
  static LocalDiscoveryLookupResult? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<LocalDiscoveryCandidate> get candidates => $_getList(0);
}

/// PlatformEvent 保留窄 binding 的异步平台入口。
/// 旧 Cloud/WebRTC 浏览器 primitive 已删除，后续能力必须由新 Proto 重新分配字段。
class PlatformEvent extends $pb.GeneratedMessage {
  factory PlatformEvent() => create();

  PlatformEvent._();

  factory PlatformEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PlatformEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PlatformEvent',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlatformEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlatformEvent copyWith(void Function(PlatformEvent) updates) =>
      super.copyWith((message) => updates(message as PlatformEvent))
          as PlatformEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PlatformEvent create() => PlatformEvent._();
  @$core.override
  PlatformEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PlatformEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PlatformEvent>(create);
  static PlatformEvent? _defaultInstance;
}

enum PlatformRequest_Request {
  credentialResolve,
  credentialPrepare,
  credentialDelete,
  credentialSign,
  credentialBind,
  endpointRegistryLoad,
  endpointRegistryStore,
  sshCredentialLookup,
  sshCredentialSign,
  sshCredentialDelete,
  cloudProfileResolve,
  localDiscoveryLookup,
  notSet
}

class PlatformRequest extends $pb.GeneratedMessage {
  factory PlatformRequest({
    $fixnum.Int64? requestId,
    CredentialResolveRequest? credentialResolve,
    CredentialPrepareRequest? credentialPrepare,
    CredentialDeleteRequest? credentialDelete,
    CredentialSignRequest? credentialSign,
    CredentialBindRequest? credentialBind,
    EndpointRegistryLoadRequest? endpointRegistryLoad,
    EndpointRegistryStoreRequest? endpointRegistryStore,
    SSHCredentialLookupRequest? sshCredentialLookup,
    SSHCredentialSignRequest? sshCredentialSign,
    SSHCredentialDeleteRequest? sshCredentialDelete,
    CloudProfileResolveRequest? cloudProfileResolve,
    LocalDiscoveryLookupRequest? localDiscoveryLookup,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (credentialResolve != null) result.credentialResolve = credentialResolve;
    if (credentialPrepare != null) result.credentialPrepare = credentialPrepare;
    if (credentialDelete != null) result.credentialDelete = credentialDelete;
    if (credentialSign != null) result.credentialSign = credentialSign;
    if (credentialBind != null) result.credentialBind = credentialBind;
    if (endpointRegistryLoad != null)
      result.endpointRegistryLoad = endpointRegistryLoad;
    if (endpointRegistryStore != null)
      result.endpointRegistryStore = endpointRegistryStore;
    if (sshCredentialLookup != null)
      result.sshCredentialLookup = sshCredentialLookup;
    if (sshCredentialSign != null) result.sshCredentialSign = sshCredentialSign;
    if (sshCredentialDelete != null)
      result.sshCredentialDelete = sshCredentialDelete;
    if (cloudProfileResolve != null)
      result.cloudProfileResolve = cloudProfileResolve;
    if (localDiscoveryLookup != null)
      result.localDiscoveryLookup = localDiscoveryLookup;
    return result;
  }

  PlatformRequest._();

  factory PlatformRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PlatformRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PlatformRequest_Request>
      _PlatformRequest_RequestByTag = {
    10: PlatformRequest_Request.credentialResolve,
    11: PlatformRequest_Request.credentialPrepare,
    12: PlatformRequest_Request.credentialDelete,
    13: PlatformRequest_Request.credentialSign,
    14: PlatformRequest_Request.credentialBind,
    15: PlatformRequest_Request.endpointRegistryLoad,
    16: PlatformRequest_Request.endpointRegistryStore,
    17: PlatformRequest_Request.sshCredentialLookup,
    18: PlatformRequest_Request.sshCredentialSign,
    19: PlatformRequest_Request.sshCredentialDelete,
    20: PlatformRequest_Request.cloudProfileResolve,
    21: PlatformRequest_Request.localDiscoveryLookup,
    0: PlatformRequest_Request.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PlatformRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21])
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<CredentialResolveRequest>(
        10, _omitFieldNames ? '' : 'credentialResolve',
        subBuilder: CredentialResolveRequest.create)
    ..aOM<CredentialPrepareRequest>(
        11, _omitFieldNames ? '' : 'credentialPrepare',
        subBuilder: CredentialPrepareRequest.create)
    ..aOM<CredentialDeleteRequest>(
        12, _omitFieldNames ? '' : 'credentialDelete',
        subBuilder: CredentialDeleteRequest.create)
    ..aOM<CredentialSignRequest>(13, _omitFieldNames ? '' : 'credentialSign',
        subBuilder: CredentialSignRequest.create)
    ..aOM<CredentialBindRequest>(14, _omitFieldNames ? '' : 'credentialBind',
        subBuilder: CredentialBindRequest.create)
    ..aOM<EndpointRegistryLoadRequest>(
        15, _omitFieldNames ? '' : 'endpointRegistryLoad',
        subBuilder: EndpointRegistryLoadRequest.create)
    ..aOM<EndpointRegistryStoreRequest>(
        16, _omitFieldNames ? '' : 'endpointRegistryStore',
        subBuilder: EndpointRegistryStoreRequest.create)
    ..aOM<SSHCredentialLookupRequest>(
        17, _omitFieldNames ? '' : 'sshCredentialLookup',
        subBuilder: SSHCredentialLookupRequest.create)
    ..aOM<SSHCredentialSignRequest>(
        18, _omitFieldNames ? '' : 'sshCredentialSign',
        subBuilder: SSHCredentialSignRequest.create)
    ..aOM<SSHCredentialDeleteRequest>(
        19, _omitFieldNames ? '' : 'sshCredentialDelete',
        subBuilder: SSHCredentialDeleteRequest.create)
    ..aOM<CloudProfileResolveRequest>(
        20, _omitFieldNames ? '' : 'cloudProfileResolve',
        subBuilder: CloudProfileResolveRequest.create)
    ..aOM<LocalDiscoveryLookupRequest>(
        21, _omitFieldNames ? '' : 'localDiscoveryLookup',
        subBuilder: LocalDiscoveryLookupRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlatformRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlatformRequest copyWith(void Function(PlatformRequest) updates) =>
      super.copyWith((message) => updates(message as PlatformRequest))
          as PlatformRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PlatformRequest create() => PlatformRequest._();
  @$core.override
  PlatformRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PlatformRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PlatformRequest>(create);
  static PlatformRequest? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  @$pb.TagNumber(19)
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  PlatformRequest_Request whichRequest() =>
      _PlatformRequest_RequestByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  @$pb.TagNumber(19)
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  void clearRequest() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(10)
  CredentialResolveRequest get credentialResolve => $_getN(1);
  @$pb.TagNumber(10)
  set credentialResolve(CredentialResolveRequest value) =>
      $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasCredentialResolve() => $_has(1);
  @$pb.TagNumber(10)
  void clearCredentialResolve() => $_clearField(10);
  @$pb.TagNumber(10)
  CredentialResolveRequest ensureCredentialResolve() => $_ensure(1);

  @$pb.TagNumber(11)
  CredentialPrepareRequest get credentialPrepare => $_getN(2);
  @$pb.TagNumber(11)
  set credentialPrepare(CredentialPrepareRequest value) =>
      $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasCredentialPrepare() => $_has(2);
  @$pb.TagNumber(11)
  void clearCredentialPrepare() => $_clearField(11);
  @$pb.TagNumber(11)
  CredentialPrepareRequest ensureCredentialPrepare() => $_ensure(2);

  @$pb.TagNumber(12)
  CredentialDeleteRequest get credentialDelete => $_getN(3);
  @$pb.TagNumber(12)
  set credentialDelete(CredentialDeleteRequest value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasCredentialDelete() => $_has(3);
  @$pb.TagNumber(12)
  void clearCredentialDelete() => $_clearField(12);
  @$pb.TagNumber(12)
  CredentialDeleteRequest ensureCredentialDelete() => $_ensure(3);

  @$pb.TagNumber(13)
  CredentialSignRequest get credentialSign => $_getN(4);
  @$pb.TagNumber(13)
  set credentialSign(CredentialSignRequest value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasCredentialSign() => $_has(4);
  @$pb.TagNumber(13)
  void clearCredentialSign() => $_clearField(13);
  @$pb.TagNumber(13)
  CredentialSignRequest ensureCredentialSign() => $_ensure(4);

  @$pb.TagNumber(14)
  CredentialBindRequest get credentialBind => $_getN(5);
  @$pb.TagNumber(14)
  set credentialBind(CredentialBindRequest value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasCredentialBind() => $_has(5);
  @$pb.TagNumber(14)
  void clearCredentialBind() => $_clearField(14);
  @$pb.TagNumber(14)
  CredentialBindRequest ensureCredentialBind() => $_ensure(5);

  @$pb.TagNumber(15)
  EndpointRegistryLoadRequest get endpointRegistryLoad => $_getN(6);
  @$pb.TagNumber(15)
  set endpointRegistryLoad(EndpointRegistryLoadRequest value) =>
      $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasEndpointRegistryLoad() => $_has(6);
  @$pb.TagNumber(15)
  void clearEndpointRegistryLoad() => $_clearField(15);
  @$pb.TagNumber(15)
  EndpointRegistryLoadRequest ensureEndpointRegistryLoad() => $_ensure(6);

  @$pb.TagNumber(16)
  EndpointRegistryStoreRequest get endpointRegistryStore => $_getN(7);
  @$pb.TagNumber(16)
  set endpointRegistryStore(EndpointRegistryStoreRequest value) =>
      $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasEndpointRegistryStore() => $_has(7);
  @$pb.TagNumber(16)
  void clearEndpointRegistryStore() => $_clearField(16);
  @$pb.TagNumber(16)
  EndpointRegistryStoreRequest ensureEndpointRegistryStore() => $_ensure(7);

  @$pb.TagNumber(17)
  SSHCredentialLookupRequest get sshCredentialLookup => $_getN(8);
  @$pb.TagNumber(17)
  set sshCredentialLookup(SSHCredentialLookupRequest value) =>
      $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasSshCredentialLookup() => $_has(8);
  @$pb.TagNumber(17)
  void clearSshCredentialLookup() => $_clearField(17);
  @$pb.TagNumber(17)
  SSHCredentialLookupRequest ensureSshCredentialLookup() => $_ensure(8);

  @$pb.TagNumber(18)
  SSHCredentialSignRequest get sshCredentialSign => $_getN(9);
  @$pb.TagNumber(18)
  set sshCredentialSign(SSHCredentialSignRequest value) =>
      $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasSshCredentialSign() => $_has(9);
  @$pb.TagNumber(18)
  void clearSshCredentialSign() => $_clearField(18);
  @$pb.TagNumber(18)
  SSHCredentialSignRequest ensureSshCredentialSign() => $_ensure(9);

  @$pb.TagNumber(19)
  SSHCredentialDeleteRequest get sshCredentialDelete => $_getN(10);
  @$pb.TagNumber(19)
  set sshCredentialDelete(SSHCredentialDeleteRequest value) =>
      $_setField(19, value);
  @$pb.TagNumber(19)
  $core.bool hasSshCredentialDelete() => $_has(10);
  @$pb.TagNumber(19)
  void clearSshCredentialDelete() => $_clearField(19);
  @$pb.TagNumber(19)
  SSHCredentialDeleteRequest ensureSshCredentialDelete() => $_ensure(10);

  @$pb.TagNumber(20)
  CloudProfileResolveRequest get cloudProfileResolve => $_getN(11);
  @$pb.TagNumber(20)
  set cloudProfileResolve(CloudProfileResolveRequest value) =>
      $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasCloudProfileResolve() => $_has(11);
  @$pb.TagNumber(20)
  void clearCloudProfileResolve() => $_clearField(20);
  @$pb.TagNumber(20)
  CloudProfileResolveRequest ensureCloudProfileResolve() => $_ensure(11);

  @$pb.TagNumber(21)
  LocalDiscoveryLookupRequest get localDiscoveryLookup => $_getN(12);
  @$pb.TagNumber(21)
  set localDiscoveryLookup(LocalDiscoveryLookupRequest value) =>
      $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasLocalDiscoveryLookup() => $_has(12);
  @$pb.TagNumber(21)
  void clearLocalDiscoveryLookup() => $_clearField(21);
  @$pb.TagNumber(21)
  LocalDiscoveryLookupRequest ensureLocalDiscoveryLookup() => $_ensure(12);
}

enum PlatformResponse_Response {
  credential,
  credentialSign,
  endpointRegistry,
  sshCredential,
  sshCredentialSign,
  cloudProfile,
  localDiscovery,
  notSet
}

class PlatformResponse extends $pb.GeneratedMessage {
  factory PlatformResponse({
    $fixnum.Int64? requestId,
    $0.ApiError? error,
    CredentialRecord? credential,
    CredentialSignResponse? credentialSign,
    EndpointRegistryLoaded? endpointRegistry,
    SSHCredentialRecord? sshCredential,
    SSHCredentialSignResponse? sshCredentialSign,
    CloudProfileRecord? cloudProfile,
    LocalDiscoveryLookupResult? localDiscovery,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (error != null) result.error = error;
    if (credential != null) result.credential = credential;
    if (credentialSign != null) result.credentialSign = credentialSign;
    if (endpointRegistry != null) result.endpointRegistry = endpointRegistry;
    if (sshCredential != null) result.sshCredential = sshCredential;
    if (sshCredentialSign != null) result.sshCredentialSign = sshCredentialSign;
    if (cloudProfile != null) result.cloudProfile = cloudProfile;
    if (localDiscovery != null) result.localDiscovery = localDiscovery;
    return result;
  }

  PlatformResponse._();

  factory PlatformResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PlatformResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PlatformResponse_Response>
      _PlatformResponse_ResponseByTag = {
    10: PlatformResponse_Response.credential,
    11: PlatformResponse_Response.credentialSign,
    12: PlatformResponse_Response.endpointRegistry,
    13: PlatformResponse_Response.sshCredential,
    14: PlatformResponse_Response.sshCredentialSign,
    15: PlatformResponse_Response.cloudProfile,
    16: PlatformResponse_Response.localDiscovery,
    0: PlatformResponse_Response.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PlatformResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 12, 13, 14, 15, 16])
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.ApiError>(2, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..aOM<CredentialRecord>(10, _omitFieldNames ? '' : 'credential',
        subBuilder: CredentialRecord.create)
    ..aOM<CredentialSignResponse>(11, _omitFieldNames ? '' : 'credentialSign',
        subBuilder: CredentialSignResponse.create)
    ..aOM<EndpointRegistryLoaded>(12, _omitFieldNames ? '' : 'endpointRegistry',
        subBuilder: EndpointRegistryLoaded.create)
    ..aOM<SSHCredentialRecord>(13, _omitFieldNames ? '' : 'sshCredential',
        subBuilder: SSHCredentialRecord.create)
    ..aOM<SSHCredentialSignResponse>(
        14, _omitFieldNames ? '' : 'sshCredentialSign',
        subBuilder: SSHCredentialSignResponse.create)
    ..aOM<CloudProfileRecord>(15, _omitFieldNames ? '' : 'cloudProfile',
        subBuilder: CloudProfileRecord.create)
    ..aOM<LocalDiscoveryLookupResult>(
        16, _omitFieldNames ? '' : 'localDiscovery',
        subBuilder: LocalDiscoveryLookupResult.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlatformResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlatformResponse copyWith(void Function(PlatformResponse) updates) =>
      super.copyWith((message) => updates(message as PlatformResponse))
          as PlatformResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PlatformResponse create() => PlatformResponse._();
  @$core.override
  PlatformResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PlatformResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PlatformResponse>(create);
  static PlatformResponse? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  PlatformResponse_Response whichResponse() =>
      _PlatformResponse_ResponseByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  void clearResponse() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.ApiError get error => $_getN(1);
  @$pb.TagNumber(2)
  set error($0.ApiError value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ApiError ensureError() => $_ensure(1);

  @$pb.TagNumber(10)
  CredentialRecord get credential => $_getN(2);
  @$pb.TagNumber(10)
  set credential(CredentialRecord value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasCredential() => $_has(2);
  @$pb.TagNumber(10)
  void clearCredential() => $_clearField(10);
  @$pb.TagNumber(10)
  CredentialRecord ensureCredential() => $_ensure(2);

  @$pb.TagNumber(11)
  CredentialSignResponse get credentialSign => $_getN(3);
  @$pb.TagNumber(11)
  set credentialSign(CredentialSignResponse value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasCredentialSign() => $_has(3);
  @$pb.TagNumber(11)
  void clearCredentialSign() => $_clearField(11);
  @$pb.TagNumber(11)
  CredentialSignResponse ensureCredentialSign() => $_ensure(3);

  @$pb.TagNumber(12)
  EndpointRegistryLoaded get endpointRegistry => $_getN(4);
  @$pb.TagNumber(12)
  set endpointRegistry(EndpointRegistryLoaded value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasEndpointRegistry() => $_has(4);
  @$pb.TagNumber(12)
  void clearEndpointRegistry() => $_clearField(12);
  @$pb.TagNumber(12)
  EndpointRegistryLoaded ensureEndpointRegistry() => $_ensure(4);

  @$pb.TagNumber(13)
  SSHCredentialRecord get sshCredential => $_getN(5);
  @$pb.TagNumber(13)
  set sshCredential(SSHCredentialRecord value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasSshCredential() => $_has(5);
  @$pb.TagNumber(13)
  void clearSshCredential() => $_clearField(13);
  @$pb.TagNumber(13)
  SSHCredentialRecord ensureSshCredential() => $_ensure(5);

  @$pb.TagNumber(14)
  SSHCredentialSignResponse get sshCredentialSign => $_getN(6);
  @$pb.TagNumber(14)
  set sshCredentialSign(SSHCredentialSignResponse value) =>
      $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasSshCredentialSign() => $_has(6);
  @$pb.TagNumber(14)
  void clearSshCredentialSign() => $_clearField(14);
  @$pb.TagNumber(14)
  SSHCredentialSignResponse ensureSshCredentialSign() => $_ensure(6);

  @$pb.TagNumber(15)
  CloudProfileRecord get cloudProfile => $_getN(7);
  @$pb.TagNumber(15)
  set cloudProfile(CloudProfileRecord value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasCloudProfile() => $_has(7);
  @$pb.TagNumber(15)
  void clearCloudProfile() => $_clearField(15);
  @$pb.TagNumber(15)
  CloudProfileRecord ensureCloudProfile() => $_ensure(7);

  @$pb.TagNumber(16)
  LocalDiscoveryLookupResult get localDiscovery => $_getN(8);
  @$pb.TagNumber(16)
  set localDiscovery(LocalDiscoveryLookupResult value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasLocalDiscovery() => $_has(8);
  @$pb.TagNumber(16)
  void clearLocalDiscovery() => $_clearField(16);
  @$pb.TagNumber(16)
  LocalDiscoveryLookupResult ensureLocalDiscovery() => $_ensure(8);
}

class PTYStreamSyncLost extends $pb.GeneratedMessage {
  factory PTYStreamSyncLost({
    $fixnum.Int64? droppedBytes,
  }) {
    final result = create();
    if (droppedBytes != null) result.droppedBytes = droppedBytes;
    return result;
  }

  PTYStreamSyncLost._();

  factory PTYStreamSyncLost.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PTYStreamSyncLost.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PTYStreamSyncLost',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'droppedBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PTYStreamSyncLost clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PTYStreamSyncLost copyWith(void Function(PTYStreamSyncLost) updates) =>
      super.copyWith((message) => updates(message as PTYStreamSyncLost))
          as PTYStreamSyncLost;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PTYStreamSyncLost create() => PTYStreamSyncLost._();
  @$core.override
  PTYStreamSyncLost createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PTYStreamSyncLost getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PTYStreamSyncLost>(create);
  static PTYStreamSyncLost? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get droppedBytes => $_getI64(0);
  @$pb.TagNumber(1)
  set droppedBytes($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDroppedBytes() => $_has(0);
  @$pb.TagNumber(1)
  void clearDroppedBytes() => $_clearField(1);
}

class PTYStreamClosed extends $pb.GeneratedMessage {
  factory PTYStreamClosed({
    $core.int? exitCode,
  }) {
    final result = create();
    if (exitCode != null) result.exitCode = exitCode;
    return result;
  }

  PTYStreamClosed._();

  factory PTYStreamClosed.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PTYStreamClosed.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PTYStreamClosed',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'exitCode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PTYStreamClosed clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PTYStreamClosed copyWith(void Function(PTYStreamClosed) updates) =>
      super.copyWith((message) => updates(message as PTYStreamClosed))
          as PTYStreamClosed;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PTYStreamClosed create() => PTYStreamClosed._();
  @$core.override
  PTYStreamClosed createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PTYStreamClosed getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PTYStreamClosed>(create);
  static PTYStreamClosed? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get exitCode => $_getIZ(0);
  @$pb.TagNumber(1)
  set exitCode($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExitCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearExitCode() => $_clearField(1);
}

class EndpointSupervisorDemand extends $pb.GeneratedMessage {
  factory EndpointSupervisorDemand({
    $core.String? endpointId,
    EndpointSupervisorMode? mode,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (mode != null) result.mode = mode;
    return result;
  }

  EndpointSupervisorDemand._();

  factory EndpointSupervisorDemand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSupervisorDemand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSupervisorDemand',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aE<EndpointSupervisorMode>(2, _omitFieldNames ? '' : 'mode',
        enumValues: EndpointSupervisorMode.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorDemand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorDemand copyWith(
          void Function(EndpointSupervisorDemand) updates) =>
      super.copyWith((message) => updates(message as EndpointSupervisorDemand))
          as EndpointSupervisorDemand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorDemand create() => EndpointSupervisorDemand._();
  @$core.override
  EndpointSupervisorDemand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorDemand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSupervisorDemand>(create);
  static EndpointSupervisorDemand? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  EndpointSupervisorMode get mode => $_getN(1);
  @$pb.TagNumber(2)
  set mode(EndpointSupervisorMode value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasMode() => $_has(1);
  @$pb.TagNumber(2)
  void clearMode() => $_clearField(2);
}

class EndpointSupervisorDemandSnapshot extends $pb.GeneratedMessage {
  factory EndpointSupervisorDemandSnapshot({
    $core.String? attachmentId,
    $fixnum.Int64? demandRevision,
    $core.Iterable<EndpointSupervisorDemand>? endpoints,
  }) {
    final result = create();
    if (attachmentId != null) result.attachmentId = attachmentId;
    if (demandRevision != null) result.demandRevision = demandRevision;
    if (endpoints != null) result.endpoints.addAll(endpoints);
    return result;
  }

  EndpointSupervisorDemandSnapshot._();

  factory EndpointSupervisorDemandSnapshot.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSupervisorDemandSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSupervisorDemandSnapshot',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'attachmentId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'demandRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPM<EndpointSupervisorDemand>(3, _omitFieldNames ? '' : 'endpoints',
        subBuilder: EndpointSupervisorDemand.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorDemandSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorDemandSnapshot copyWith(
          void Function(EndpointSupervisorDemandSnapshot) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointSupervisorDemandSnapshot))
          as EndpointSupervisorDemandSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorDemandSnapshot create() =>
      EndpointSupervisorDemandSnapshot._();
  @$core.override
  EndpointSupervisorDemandSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorDemandSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSupervisorDemandSnapshot>(
          create);
  static EndpointSupervisorDemandSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get attachmentId => $_getSZ(0);
  @$pb.TagNumber(1)
  set attachmentId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAttachmentId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAttachmentId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get demandRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set demandRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDemandRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearDemandRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<EndpointSupervisorDemand> get endpoints => $_getList(2);
}

/// Network metadata is only a wake-up hint. connected=false pauses dialing but
/// does not itself invalidate a retained physical winner.
class EndpointSupervisorHostSignal extends $pb.GeneratedMessage {
  factory EndpointSupervisorHostSignal({
    $fixnum.Int64? revision,
    $core.bool? connected,
    $core.String? reason,
    $core.bool? foreground,
  }) {
    final result = create();
    if (revision != null) result.revision = revision;
    if (connected != null) result.connected = connected;
    if (reason != null) result.reason = reason;
    if (foreground != null) result.foreground = foreground;
    return result;
  }

  EndpointSupervisorHostSignal._();

  factory EndpointSupervisorHostSignal.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSupervisorHostSignal.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSupervisorHostSignal',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(2, _omitFieldNames ? '' : 'connected')
    ..aOS(3, _omitFieldNames ? '' : 'reason')
    ..aOB(4, _omitFieldNames ? '' : 'foreground')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorHostSignal clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorHostSignal copyWith(
          void Function(EndpointSupervisorHostSignal) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointSupervisorHostSignal))
          as EndpointSupervisorHostSignal;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorHostSignal create() =>
      EndpointSupervisorHostSignal._();
  @$core.override
  EndpointSupervisorHostSignal createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorHostSignal getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSupervisorHostSignal>(create);
  static EndpointSupervisorHostSignal? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get revision => $_getI64(0);
  @$pb.TagNumber(1)
  set revision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevision() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get connected => $_getBF(1);
  @$pb.TagNumber(2)
  set connected($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConnected() => $_has(1);
  @$pb.TagNumber(2)
  void clearConnected() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get reason => $_getSZ(2);
  @$pb.TagNumber(3)
  set reason($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearReason() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get foreground => $_getBF(3);
  @$pb.TagNumber(4)
  set foreground($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasForeground() => $_has(3);
  @$pb.TagNumber(4)
  void clearForeground() => $_clearField(4);
}

class EndpointSupervisorProjection extends $pb.GeneratedMessage {
  factory EndpointSupervisorProjection({
    $core.String? endpointId,
    EndpointSupervisorMode? mode,
    $core.String? phase,
    $fixnum.Int64? controlRevision,
    $fixnum.Int64? attemptId,
    $0.EndpointSessionStamp? session,
    $core.String? errorCode,
    $core.String? message,
    $fixnum.Int64? probeCount,
    $fixnum.Int64? dialCount,
    $fixnum.Int64? backoffCount,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (mode != null) result.mode = mode;
    if (phase != null) result.phase = phase;
    if (controlRevision != null) result.controlRevision = controlRevision;
    if (attemptId != null) result.attemptId = attemptId;
    if (session != null) result.session = session;
    if (errorCode != null) result.errorCode = errorCode;
    if (message != null) result.message = message;
    if (probeCount != null) result.probeCount = probeCount;
    if (dialCount != null) result.dialCount = dialCount;
    if (backoffCount != null) result.backoffCount = backoffCount;
    return result;
  }

  EndpointSupervisorProjection._();

  factory EndpointSupervisorProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSupervisorProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSupervisorProjection',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aE<EndpointSupervisorMode>(2, _omitFieldNames ? '' : 'mode',
        enumValues: EndpointSupervisorMode.values)
    ..aOS(3, _omitFieldNames ? '' : 'phase')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'controlRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'attemptId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.EndpointSessionStamp>(6, _omitFieldNames ? '' : 'session',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOS(7, _omitFieldNames ? '' : 'errorCode')
    ..aOS(8, _omitFieldNames ? '' : 'message')
    ..a<$fixnum.Int64>(
        9, _omitFieldNames ? '' : 'probeCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        10, _omitFieldNames ? '' : 'dialCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        11, _omitFieldNames ? '' : 'backoffCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorProjection copyWith(
          void Function(EndpointSupervisorProjection) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointSupervisorProjection))
          as EndpointSupervisorProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorProjection create() =>
      EndpointSupervisorProjection._();
  @$core.override
  EndpointSupervisorProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSupervisorProjection>(create);
  static EndpointSupervisorProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  EndpointSupervisorMode get mode => $_getN(1);
  @$pb.TagNumber(2)
  set mode(EndpointSupervisorMode value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasMode() => $_has(1);
  @$pb.TagNumber(2)
  void clearMode() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get phase => $_getSZ(2);
  @$pb.TagNumber(3)
  set phase($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPhase() => $_has(2);
  @$pb.TagNumber(3)
  void clearPhase() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get controlRevision => $_getI64(3);
  @$pb.TagNumber(4)
  set controlRevision($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasControlRevision() => $_has(3);
  @$pb.TagNumber(4)
  void clearControlRevision() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get attemptId => $_getI64(4);
  @$pb.TagNumber(5)
  set attemptId($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAttemptId() => $_has(4);
  @$pb.TagNumber(5)
  void clearAttemptId() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.EndpointSessionStamp get session => $_getN(5);
  @$pb.TagNumber(6)
  set session($0.EndpointSessionStamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasSession() => $_has(5);
  @$pb.TagNumber(6)
  void clearSession() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.EndpointSessionStamp ensureSession() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.String get errorCode => $_getSZ(6);
  @$pb.TagNumber(7)
  set errorCode($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasErrorCode() => $_has(6);
  @$pb.TagNumber(7)
  void clearErrorCode() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get message => $_getSZ(7);
  @$pb.TagNumber(8)
  set message($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasMessage() => $_has(7);
  @$pb.TagNumber(8)
  void clearMessage() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get probeCount => $_getI64(8);
  @$pb.TagNumber(9)
  set probeCount($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasProbeCount() => $_has(8);
  @$pb.TagNumber(9)
  void clearProbeCount() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get dialCount => $_getI64(9);
  @$pb.TagNumber(10)
  set dialCount($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasDialCount() => $_has(9);
  @$pb.TagNumber(10)
  void clearDialCount() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get backoffCount => $_getI64(10);
  @$pb.TagNumber(11)
  set backoffCount($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasBackoffCount() => $_has(10);
  @$pb.TagNumber(11)
  void clearBackoffCount() => $_clearField(11);
}

class EndpointSupervisorSnapshot extends $pb.GeneratedMessage {
  factory EndpointSupervisorSnapshot({
    $core.Iterable<EndpointSupervisorProjection>? endpoints,
  }) {
    final result = create();
    if (endpoints != null) result.endpoints.addAll(endpoints);
    return result;
  }

  EndpointSupervisorSnapshot._();

  factory EndpointSupervisorSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSupervisorSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSupervisorSnapshot',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.client.binding.v1'),
      createEmptyInstance: create)
    ..pPM<EndpointSupervisorProjection>(1, _omitFieldNames ? '' : 'endpoints',
        subBuilder: EndpointSupervisorProjection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSupervisorSnapshot copyWith(
          void Function(EndpointSupervisorSnapshot) updates) =>
      super.copyWith(
              (message) => updates(message as EndpointSupervisorSnapshot))
          as EndpointSupervisorSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorSnapshot create() => EndpointSupervisorSnapshot._();
  @$core.override
  EndpointSupervisorSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSupervisorSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSupervisorSnapshot>(create);
  static EndpointSupervisorSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<EndpointSupervisorProjection> get endpoints => $_getList(0);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
