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

import 'package:protobuf/protobuf.dart' as $pb;

class ConnectIntent extends $pb.ProtobufEnum {
  static const ConnectIntent CONNECT_INTENT_UNSPECIFIED =
      ConnectIntent._(0, _omitEnumNames ? '' : 'CONNECT_INTENT_UNSPECIFIED');
  static const ConnectIntent CONNECT_INTENT_INTERACTIVE =
      ConnectIntent._(1, _omitEnumNames ? '' : 'CONNECT_INTENT_INTERACTIVE');
  static const ConnectIntent CONNECT_INTENT_BACKGROUND =
      ConnectIntent._(2, _omitEnumNames ? '' : 'CONNECT_INTENT_BACKGROUND');
  static const ConnectIntent CONNECT_INTENT_PROBE =
      ConnectIntent._(3, _omitEnumNames ? '' : 'CONNECT_INTENT_PROBE');

  static const $core.List<ConnectIntent> values = <ConnectIntent>[
    CONNECT_INTENT_UNSPECIFIED,
    CONNECT_INTENT_INTERACTIVE,
    CONNECT_INTENT_BACKGROUND,
    CONNECT_INTENT_PROBE,
  ];

  static final $core.List<ConnectIntent?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static ConnectIntent? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ConnectIntent._(super.value, super.name);
}

class ResourceStreamFrameType extends $pb.ProtobufEnum {
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_UNSPECIFIED =
      ResourceStreamFrameType._(
          0, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_UNSPECIFIED');
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_FILE_DATA =
      ResourceStreamFrameType._(
          1, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_FILE_DATA');
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_FILE_ACK =
      ResourceStreamFrameType._(
          2, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_FILE_ACK');
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH =
      ResourceStreamFrameType._(
          3, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH');
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT =
      ResourceStreamFrameType._(
          4, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT');
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_ERROR =
      ResourceStreamFrameType._(
          5, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_ERROR');
  static const ResourceStreamFrameType
      RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO = ResourceStreamFrameType._(6,
          _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO');

  /// PTY_OUTPUT payload 是未经文本解码或 ANSI 解析的原始 PTY bytes。
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT =
      ResourceStreamFrameType._(
          7, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT');

  /// PTY_SYNC_LOST payload 是 serialized PTYStreamSyncLost；收到后必须丢弃本地解析状态。
  static const ResourceStreamFrameType
      RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST = ResourceStreamFrameType._(
          8, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST');

  /// PTY_CLOSED payload 是 serialized PTYStreamClosed。
  static const ResourceStreamFrameType RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED =
      ResourceStreamFrameType._(
          9, _omitEnumNames ? '' : 'RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED');

  static const $core.List<ResourceStreamFrameType> values =
      <ResourceStreamFrameType>[
    RESOURCE_STREAM_FRAME_TYPE_UNSPECIFIED,
    RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
    RESOURCE_STREAM_FRAME_TYPE_FILE_ACK,
    RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH,
    RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT,
    RESOURCE_STREAM_FRAME_TYPE_ERROR,
    RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO,
    RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT,
    RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST,
    RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED,
  ];

  static final $core.List<ResourceStreamFrameType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 9);
  static ResourceStreamFrameType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ResourceStreamFrameType._(super.value, super.name);
}

/// ConnectionRouteKind 是 ReadySession 实际获胜的持久 Route kind 投影。
class ConnectionRouteKind extends $pb.ProtobufEnum {
  static const ConnectionRouteKind CONNECTION_ROUTE_KIND_UNSPECIFIED =
      ConnectionRouteKind._(
          0, _omitEnumNames ? '' : 'CONNECTION_ROUTE_KIND_UNSPECIFIED');
  static const ConnectionRouteKind CONNECTION_ROUTE_KIND_LOCAL =
      ConnectionRouteKind._(
          1, _omitEnumNames ? '' : 'CONNECTION_ROUTE_KIND_LOCAL');
  static const ConnectionRouteKind CONNECTION_ROUTE_KIND_DIRECT =
      ConnectionRouteKind._(
          2, _omitEnumNames ? '' : 'CONNECTION_ROUTE_KIND_DIRECT');
  static const ConnectionRouteKind CONNECTION_ROUTE_KIND_SSH =
      ConnectionRouteKind._(
          3, _omitEnumNames ? '' : 'CONNECTION_ROUTE_KIND_SSH');
  static const ConnectionRouteKind CONNECTION_ROUTE_KIND_CLOUD =
      ConnectionRouteKind._(
          5, _omitEnumNames ? '' : 'CONNECTION_ROUTE_KIND_CLOUD');

  static const $core.List<ConnectionRouteKind> values = <ConnectionRouteKind>[
    CONNECTION_ROUTE_KIND_UNSPECIFIED,
    CONNECTION_ROUTE_KIND_LOCAL,
    CONNECTION_ROUTE_KIND_DIRECT,
    CONNECTION_ROUTE_KIND_SSH,
    CONNECTION_ROUTE_KIND_CLOUD,
  ];

  static final $core.List<ConnectionRouteKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static ConnectionRouteKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ConnectionRouteKind._(super.value, super.name);
}

/// ConnectionObservedPath 是已建立 WebRTC candidate pair 的中性路径投影。
class ConnectionObservedPath extends $pb.ProtobufEnum {
  static const ConnectionObservedPath CONNECTION_OBSERVED_PATH_UNSPECIFIED =
      ConnectionObservedPath._(
          0, _omitEnumNames ? '' : 'CONNECTION_OBSERVED_PATH_UNSPECIFIED');
  static const ConnectionObservedPath CONNECTION_OBSERVED_PATH_DIRECT =
      ConnectionObservedPath._(
          1, _omitEnumNames ? '' : 'CONNECTION_OBSERVED_PATH_DIRECT');
  static const ConnectionObservedPath CONNECTION_OBSERVED_PATH_SINGLE_RELAY =
      ConnectionObservedPath._(
          2, _omitEnumNames ? '' : 'CONNECTION_OBSERVED_PATH_SINGLE_RELAY');

  static const $core.List<ConnectionObservedPath> values =
      <ConnectionObservedPath>[
    CONNECTION_OBSERVED_PATH_UNSPECIFIED,
    CONNECTION_OBSERVED_PATH_DIRECT,
    CONNECTION_OBSERVED_PATH_SINGLE_RELAY,
  ];

  static final $core.List<ConnectionObservedPath?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static ConnectionObservedPath? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ConnectionObservedPath._(super.value, super.name);
}

/// ConnectionCandidateType 是 selected ICE candidate pair 的脱敏候选类型。
class ConnectionCandidateType extends $pb.ProtobufEnum {
  static const ConnectionCandidateType CONNECTION_CANDIDATE_TYPE_UNSPECIFIED =
      ConnectionCandidateType._(
          0, _omitEnumNames ? '' : 'CONNECTION_CANDIDATE_TYPE_UNSPECIFIED');
  static const ConnectionCandidateType CONNECTION_CANDIDATE_TYPE_HOST =
      ConnectionCandidateType._(
          1, _omitEnumNames ? '' : 'CONNECTION_CANDIDATE_TYPE_HOST');
  static const ConnectionCandidateType
      CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE = ConnectionCandidateType._(2,
          _omitEnumNames ? '' : 'CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE');
  static const ConnectionCandidateType
      CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE = ConnectionCandidateType._(
          3, _omitEnumNames ? '' : 'CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE');
  static const ConnectionCandidateType CONNECTION_CANDIDATE_TYPE_RELAY =
      ConnectionCandidateType._(
          4, _omitEnumNames ? '' : 'CONNECTION_CANDIDATE_TYPE_RELAY');

  static const $core.List<ConnectionCandidateType> values =
      <ConnectionCandidateType>[
    CONNECTION_CANDIDATE_TYPE_UNSPECIFIED,
    CONNECTION_CANDIDATE_TYPE_HOST,
    CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE,
    CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE,
    CONNECTION_CANDIDATE_TYPE_RELAY,
  ];

  static final $core.List<ConnectionCandidateType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static ConnectionCandidateType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ConnectionCandidateType._(super.value, super.name);
}

/// ConnectionTransport 是 selected candidate 或 TURN allocation 可证明的传输协议。
class ConnectionTransport extends $pb.ProtobufEnum {
  static const ConnectionTransport CONNECTION_TRANSPORT_UNSPECIFIED =
      ConnectionTransport._(
          0, _omitEnumNames ? '' : 'CONNECTION_TRANSPORT_UNSPECIFIED');
  static const ConnectionTransport CONNECTION_TRANSPORT_UDP =
      ConnectionTransport._(
          1, _omitEnumNames ? '' : 'CONNECTION_TRANSPORT_UDP');
  static const ConnectionTransport CONNECTION_TRANSPORT_TCP =
      ConnectionTransport._(
          2, _omitEnumNames ? '' : 'CONNECTION_TRANSPORT_TCP');

  static const $core.List<ConnectionTransport> values = <ConnectionTransport>[
    CONNECTION_TRANSPORT_UNSPECIFIED,
    CONNECTION_TRANSPORT_UDP,
    CONNECTION_TRANSPORT_TCP,
  ];

  static final $core.List<ConnectionTransport?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static ConnectionTransport? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ConnectionTransport._(super.value, super.name);
}

/// ConnectionPolicyAvailabilityReason 是 Go planner 对指定 Route kind 不可选原因的稳定投影。
/// UI 只负责展示；不得依据 Route 字段、平台类型或登录状态自行推断可用性。
class ConnectionPolicyAvailabilityReason extends $pb.ProtobufEnum {
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_UNSPECIFIED =
      ConnectionPolicyAvailabilityReason._(
          0,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_UNSPECIFIED');
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE =
      ConnectionPolicyAvailabilityReason._(
          1,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE');
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_NOT_CONFIGURED =
      ConnectionPolicyAvailabilityReason._(
          2,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_NOT_CONFIGURED');
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_DISABLED =
      ConnectionPolicyAvailabilityReason._(
          3,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_DISABLED');
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED =
      ConnectionPolicyAvailabilityReason._(
          4,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED');
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE =
      ConnectionPolicyAvailabilityReason._(
          5,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE');
  static const ConnectionPolicyAvailabilityReason
      CONNECTION_POLICY_AVAILABILITY_REASON_CLOUD_UNAVAILABLE =
      ConnectionPolicyAvailabilityReason._(
          6,
          _omitEnumNames
              ? ''
              : 'CONNECTION_POLICY_AVAILABILITY_REASON_CLOUD_UNAVAILABLE');

  static const $core.List<ConnectionPolicyAvailabilityReason> values =
      <ConnectionPolicyAvailabilityReason>[
    CONNECTION_POLICY_AVAILABILITY_REASON_UNSPECIFIED,
    CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE,
    CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_NOT_CONFIGURED,
    CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_DISABLED,
    CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED,
    CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE,
    CONNECTION_POLICY_AVAILABILITY_REASON_CLOUD_UNAVAILABLE,
  ];

  static final $core.List<ConnectionPolicyAvailabilityReason?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static ConnectionPolicyAvailabilityReason? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ConnectionPolicyAvailabilityReason._(super.value, super.name);
}

class EndpointConnectionPhase extends $pb.ProtobufEnum {
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_UNSPECIFIED =
      EndpointConnectionPhase._(
          0, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_UNSPECIFIED');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_IDLE =
      EndpointConnectionPhase._(
          1, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_IDLE');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_PLANNING =
      EndpointConnectionPhase._(
          2, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_PLANNING');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_RESOLVING =
      EndpointConnectionPhase._(
          3, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_RESOLVING');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_SIGNALING =
      EndpointConnectionPhase._(
          4, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_SIGNALING');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_CONNECTING =
      EndpointConnectionPhase._(
          5, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_CONNECTING');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_AUTHORIZING =
      EndpointConnectionPhase._(
          6, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_AUTHORIZING');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_READY =
      EndpointConnectionPhase._(
          7, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_READY');
  static const EndpointConnectionPhase ENDPOINT_CONNECTION_PHASE_OFFLINE =
      EndpointConnectionPhase._(
          8, _omitEnumNames ? '' : 'ENDPOINT_CONNECTION_PHASE_OFFLINE');

  static const $core.List<EndpointConnectionPhase> values =
      <EndpointConnectionPhase>[
    ENDPOINT_CONNECTION_PHASE_UNSPECIFIED,
    ENDPOINT_CONNECTION_PHASE_IDLE,
    ENDPOINT_CONNECTION_PHASE_PLANNING,
    ENDPOINT_CONNECTION_PHASE_RESOLVING,
    ENDPOINT_CONNECTION_PHASE_SIGNALING,
    ENDPOINT_CONNECTION_PHASE_CONNECTING,
    ENDPOINT_CONNECTION_PHASE_AUTHORIZING,
    ENDPOINT_CONNECTION_PHASE_READY,
    ENDPOINT_CONNECTION_PHASE_OFFLINE,
  ];

  static final $core.List<EndpointConnectionPhase?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 8);
  static EndpointConnectionPhase? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointConnectionPhase._(super.value, super.name);
}

/// EndpointSupervisorMode is selected by the Android rollout policy for each
/// demanded endpoint. Shadow records decisions while TS remains authoritative;
/// takeover moves probe/dial/backoff ownership into Go.
class EndpointSupervisorMode extends $pb.ProtobufEnum {
  static const EndpointSupervisorMode ENDPOINT_SUPERVISOR_MODE_UNSPECIFIED =
      EndpointSupervisorMode._(
          0, _omitEnumNames ? '' : 'ENDPOINT_SUPERVISOR_MODE_UNSPECIFIED');
  static const EndpointSupervisorMode ENDPOINT_SUPERVISOR_MODE_SHADOW =
      EndpointSupervisorMode._(
          1, _omitEnumNames ? '' : 'ENDPOINT_SUPERVISOR_MODE_SHADOW');
  static const EndpointSupervisorMode ENDPOINT_SUPERVISOR_MODE_TAKEOVER =
      EndpointSupervisorMode._(
          2, _omitEnumNames ? '' : 'ENDPOINT_SUPERVISOR_MODE_TAKEOVER');

  static const $core.List<EndpointSupervisorMode> values =
      <EndpointSupervisorMode>[
    ENDPOINT_SUPERVISOR_MODE_UNSPECIFIED,
    ENDPOINT_SUPERVISOR_MODE_SHADOW,
    ENDPOINT_SUPERVISOR_MODE_TAKEOVER,
  ];

  static final $core.List<EndpointSupervisorMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static EndpointSupervisorMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointSupervisorMode._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
