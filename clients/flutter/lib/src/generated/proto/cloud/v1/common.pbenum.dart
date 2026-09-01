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

import 'package:protobuf/protobuf.dart' as $pb;

/// CloudEntitlementErrorCode 是跨 Controller、Edge 和客户端稳定传播的商业准入失败。
/// 客户端只能按枚举展示本地化文案，不得解析服务端 message。
class CloudEntitlementErrorCode extends $pb.ProtobufEnum {
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_UNSPECIFIED = CloudEntitlementErrorCode._(
          0, _omitEnumNames ? '' : 'CLOUD_ENTITLEMENT_ERROR_CODE_UNSPECIFIED');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED =
      CloudEntitlementErrorCode._(
          1,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_NOT_IN_PLAN =
      CloudEntitlementErrorCode._(
          2,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_NOT_IN_PLAN');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_QUOTA_EXHAUSTED =
      CloudEntitlementErrorCode._(
          3,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_QUOTA_EXHAUSTED');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED =
      CloudEntitlementErrorCode._(
          4,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE =
      CloudEntitlementErrorCode._(
          5,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_REGION_UNAVAILABLE =
      CloudEntitlementErrorCode._(
          6,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_REGION_UNAVAILABLE');
  static const CloudEntitlementErrorCode
      CLOUD_ENTITLEMENT_ERROR_CODE_SERVICE_UNAVAILABLE =
      CloudEntitlementErrorCode._(
          7,
          _omitEnumNames
              ? ''
              : 'CLOUD_ENTITLEMENT_ERROR_CODE_SERVICE_UNAVAILABLE');

  static const $core.List<CloudEntitlementErrorCode> values =
      <CloudEntitlementErrorCode>[
    CLOUD_ENTITLEMENT_ERROR_CODE_UNSPECIFIED,
    CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED,
    CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_NOT_IN_PLAN,
    CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_QUOTA_EXHAUSTED,
    CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED,
    CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE,
    CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_REGION_UNAVAILABLE,
    CLOUD_ENTITLEMENT_ERROR_CODE_SERVICE_UNAVAILABLE,
  ];

  static final $core.List<CloudEntitlementErrorCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 7);
  static CloudEntitlementErrorCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CloudEntitlementErrorCode._(super.value, super.name);
}

/// EdgeChallengeTarget 把一次 challenge 限定到唯一 Gateway 协议，禁止跨流类型复用。
class EdgeChallengeTarget extends $pb.ProtobufEnum {
  static const EdgeChallengeTarget EDGE_CHALLENGE_TARGET_UNSPECIFIED =
      EdgeChallengeTarget._(
          0, _omitEnumNames ? '' : 'EDGE_CHALLENGE_TARGET_UNSPECIFIED');
  static const EdgeChallengeTarget EDGE_CHALLENGE_TARGET_AGENT_GATEWAY =
      EdgeChallengeTarget._(
          1, _omitEnumNames ? '' : 'EDGE_CHALLENGE_TARGET_AGENT_GATEWAY');
  static const EdgeChallengeTarget EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY =
      EdgeChallengeTarget._(
          2, _omitEnumNames ? '' : 'EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY');

  static const $core.List<EdgeChallengeTarget> values = <EdgeChallengeTarget>[
    EDGE_CHALLENGE_TARGET_UNSPECIFIED,
    EDGE_CHALLENGE_TARGET_AGENT_GATEWAY,
    EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY,
  ];

  static final $core.List<EdgeChallengeTarget?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static EdgeChallengeTarget? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EdgeChallengeTarget._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
