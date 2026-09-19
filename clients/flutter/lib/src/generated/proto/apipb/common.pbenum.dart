// This is a generated file - do not edit.
//
// Generated from apipb/common.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class ApiCapability extends $pb.ProtobufEnum {
  static const ApiCapability API_CAPABILITY_UNSPECIFIED =
      ApiCapability._(0, _omitEnumNames ? '' : 'API_CAPABILITY_UNSPECIFIED');
  static const ApiCapability API_CAPABILITY_TYPED_ERRORS =
      ApiCapability._(1, _omitEnumNames ? '' : 'API_CAPABILITY_TYPED_ERRORS');
  static const ApiCapability API_CAPABILITY_ENDPOINT_SESSION_FENCE =
      ApiCapability._(
          2, _omitEnumNames ? '' : 'API_CAPABILITY_ENDPOINT_SESSION_FENCE');
  static const ApiCapability API_CAPABILITY_OPERATION_CANCELLATION =
      ApiCapability._(
          3, _omitEnumNames ? '' : 'API_CAPABILITY_OPERATION_CANCELLATION');
  static const ApiCapability API_CAPABILITY_RESOURCE_LIFECYCLE =
      ApiCapability._(
          4, _omitEnumNames ? '' : 'API_CAPABILITY_RESOURCE_LIFECYCLE');
  static const ApiCapability API_CAPABILITY_TERMINAL_LIFECYCLE =
      ApiCapability._(
          5, _omitEnumNames ? '' : 'API_CAPABILITY_TERMINAL_LIFECYCLE');
  static const ApiCapability API_CAPABILITY_TERMINAL_ATTACHMENT =
      ApiCapability._(
          6, _omitEnumNames ? '' : 'API_CAPABILITY_TERMINAL_ATTACHMENT');
  static const ApiCapability API_CAPABILITY_PATH_QUERY =
      ApiCapability._(7, _omitEnumNames ? '' : 'API_CAPABILITY_PATH_QUERY');
  static const ApiCapability API_CAPABILITY_HISTORY =
      ApiCapability._(8, _omitEnumNames ? '' : 'API_CAPABILITY_HISTORY');
  static const ApiCapability API_CAPABILITY_LIVE_SCREEN =
      ApiCapability._(9, _omitEnumNames ? '' : 'API_CAPABILITY_LIVE_SCREEN');
  static const ApiCapability API_CAPABILITY_FILE =
      ApiCapability._(10, _omitEnumNames ? '' : 'API_CAPABILITY_FILE');
  static const ApiCapability API_CAPABILITY_STORAGE =
      ApiCapability._(11, _omitEnumNames ? '' : 'API_CAPABILITY_STORAGE');
  static const ApiCapability API_CAPABILITY_EVENT_SUBSCRIPTION =
      ApiCapability._(
          12, _omitEnumNames ? '' : 'API_CAPABILITY_EVENT_SUBSCRIPTION');
  static const ApiCapability API_CAPABILITY_CLIENT_ACCESS =
      ApiCapability._(13, _omitEnumNames ? '' : 'API_CAPABILITY_CLIENT_ACCESS');
  static const ApiCapability API_CAPABILITY_REMOTE_CONTROL = ApiCapability._(
      14, _omitEnumNames ? '' : 'API_CAPABILITY_REMOTE_CONTROL');
  static const ApiCapability API_CAPABILITY_BROWSER_PROXY =
      ApiCapability._(15, _omitEnumNames ? '' : 'API_CAPABILITY_BROWSER_PROXY');

  static const $core.List<ApiCapability> values = <ApiCapability>[
    API_CAPABILITY_UNSPECIFIED,
    API_CAPABILITY_TYPED_ERRORS,
    API_CAPABILITY_ENDPOINT_SESSION_FENCE,
    API_CAPABILITY_OPERATION_CANCELLATION,
    API_CAPABILITY_RESOURCE_LIFECYCLE,
    API_CAPABILITY_TERMINAL_LIFECYCLE,
    API_CAPABILITY_TERMINAL_ATTACHMENT,
    API_CAPABILITY_PATH_QUERY,
    API_CAPABILITY_HISTORY,
    API_CAPABILITY_LIVE_SCREEN,
    API_CAPABILITY_FILE,
    API_CAPABILITY_STORAGE,
    API_CAPABILITY_EVENT_SUBSCRIPTION,
    API_CAPABILITY_CLIENT_ACCESS,
    API_CAPABILITY_REMOTE_CONTROL,
    API_CAPABILITY_BROWSER_PROXY,
  ];

  static final $core.List<ApiCapability?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 15);
  static ApiCapability? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ApiCapability._(super.value, super.name);
}

class ApiErrorCode extends $pb.ProtobufEnum {
  static const ApiErrorCode API_ERROR_CODE_UNSPECIFIED =
      ApiErrorCode._(0, _omitEnumNames ? '' : 'API_ERROR_CODE_UNSPECIFIED');
  static const ApiErrorCode API_ERROR_CODE_INVALID_REQUEST =
      ApiErrorCode._(1, _omitEnumNames ? '' : 'API_ERROR_CODE_INVALID_REQUEST');
  static const ApiErrorCode API_ERROR_CODE_UNSUPPORTED_VERSION = ApiErrorCode._(
      2, _omitEnumNames ? '' : 'API_ERROR_CODE_UNSUPPORTED_VERSION');
  static const ApiErrorCode API_ERROR_CODE_UNSUPPORTED_CAPABILITY =
      ApiErrorCode._(
          3, _omitEnumNames ? '' : 'API_ERROR_CODE_UNSUPPORTED_CAPABILITY');
  static const ApiErrorCode API_ERROR_CODE_UNAUTHORIZED =
      ApiErrorCode._(4, _omitEnumNames ? '' : 'API_ERROR_CODE_UNAUTHORIZED');
  static const ApiErrorCode API_ERROR_CODE_FORBIDDEN =
      ApiErrorCode._(5, _omitEnumNames ? '' : 'API_ERROR_CODE_FORBIDDEN');
  static const ApiErrorCode API_ERROR_CODE_NOT_FOUND =
      ApiErrorCode._(6, _omitEnumNames ? '' : 'API_ERROR_CODE_NOT_FOUND');
  static const ApiErrorCode API_ERROR_CODE_CONFLICT =
      ApiErrorCode._(7, _omitEnumNames ? '' : 'API_ERROR_CODE_CONFLICT');
  static const ApiErrorCode API_ERROR_CODE_STALE_SESSION =
      ApiErrorCode._(8, _omitEnumNames ? '' : 'API_ERROR_CODE_STALE_SESSION');
  static const ApiErrorCode API_ERROR_CODE_CANCELLED =
      ApiErrorCode._(9, _omitEnumNames ? '' : 'API_ERROR_CODE_CANCELLED');
  static const ApiErrorCode API_ERROR_CODE_UNAVAILABLE =
      ApiErrorCode._(10, _omitEnumNames ? '' : 'API_ERROR_CODE_UNAVAILABLE');
  static const ApiErrorCode API_ERROR_CODE_INTERNAL =
      ApiErrorCode._(11, _omitEnumNames ? '' : 'API_ERROR_CODE_INTERNAL');
  static const ApiErrorCode API_ERROR_CODE_ENTITLEMENT_DENIED = ApiErrorCode._(
      12, _omitEnumNames ? '' : 'API_ERROR_CODE_ENTITLEMENT_DENIED');
  static const ApiErrorCode API_ERROR_CODE_RESOURCE_EXHAUSTED = ApiErrorCode._(
      13, _omitEnumNames ? '' : 'API_ERROR_CODE_RESOURCE_EXHAUSTED');
  static const ApiErrorCode API_ERROR_CODE_STALE_RESOURCE =
      ApiErrorCode._(14, _omitEnumNames ? '' : 'API_ERROR_CODE_STALE_RESOURCE');
  static const ApiErrorCode API_ERROR_CODE_DAEMON_BLOCKED =
      ApiErrorCode._(15, _omitEnumNames ? '' : 'API_ERROR_CODE_DAEMON_BLOCKED');
  static const ApiErrorCode API_ERROR_CODE_DAEMON_DELETED =
      ApiErrorCode._(16, _omitEnumNames ? '' : 'API_ERROR_CODE_DAEMON_DELETED');
  static const ApiErrorCode API_ERROR_CODE_RELAY_NOT_IN_PLAN = ApiErrorCode._(
      17, _omitEnumNames ? '' : 'API_ERROR_CODE_RELAY_NOT_IN_PLAN');
  static const ApiErrorCode API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED =
      ApiErrorCode._(
          18, _omitEnumNames ? '' : 'API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED');
  static const ApiErrorCode API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED =
      ApiErrorCode._(19,
          _omitEnumNames ? '' : 'API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED');
  static const ApiErrorCode API_ERROR_CODE_SUBSCRIPTION_INACTIVE =
      ApiErrorCode._(
          20, _omitEnumNames ? '' : 'API_ERROR_CODE_SUBSCRIPTION_INACTIVE');
  static const ApiErrorCode API_ERROR_CODE_RELAY_REGION_UNAVAILABLE =
      ApiErrorCode._(
          21, _omitEnumNames ? '' : 'API_ERROR_CODE_RELAY_REGION_UNAVAILABLE');

  static const $core.List<ApiErrorCode> values = <ApiErrorCode>[
    API_ERROR_CODE_UNSPECIFIED,
    API_ERROR_CODE_INVALID_REQUEST,
    API_ERROR_CODE_UNSUPPORTED_VERSION,
    API_ERROR_CODE_UNSUPPORTED_CAPABILITY,
    API_ERROR_CODE_UNAUTHORIZED,
    API_ERROR_CODE_FORBIDDEN,
    API_ERROR_CODE_NOT_FOUND,
    API_ERROR_CODE_CONFLICT,
    API_ERROR_CODE_STALE_SESSION,
    API_ERROR_CODE_CANCELLED,
    API_ERROR_CODE_UNAVAILABLE,
    API_ERROR_CODE_INTERNAL,
    API_ERROR_CODE_ENTITLEMENT_DENIED,
    API_ERROR_CODE_RESOURCE_EXHAUSTED,
    API_ERROR_CODE_STALE_RESOURCE,
    API_ERROR_CODE_DAEMON_BLOCKED,
    API_ERROR_CODE_DAEMON_DELETED,
    API_ERROR_CODE_RELAY_NOT_IN_PLAN,
    API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED,
    API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED,
    API_ERROR_CODE_SUBSCRIPTION_INACTIVE,
    API_ERROR_CODE_RELAY_REGION_UNAVAILABLE,
  ];

  static final $core.List<ApiErrorCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 21);
  static ApiErrorCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ApiErrorCode._(super.value, super.name);
}

class ResourceKind extends $pb.ProtobufEnum {
  static const ResourceKind RESOURCE_KIND_UNSPECIFIED =
      ResourceKind._(0, _omitEnumNames ? '' : 'RESOURCE_KIND_UNSPECIFIED');
  static const ResourceKind RESOURCE_KIND_OPERATION =
      ResourceKind._(1, _omitEnumNames ? '' : 'RESOURCE_KIND_OPERATION');
  static const ResourceKind RESOURCE_KIND_SUBSCRIPTION =
      ResourceKind._(2, _omitEnumNames ? '' : 'RESOURCE_KIND_SUBSCRIPTION');
  static const ResourceKind RESOURCE_KIND_TERMINAL_ATTACHMENT = ResourceKind._(
      3, _omitEnumNames ? '' : 'RESOURCE_KIND_TERMINAL_ATTACHMENT');
  static const ResourceKind RESOURCE_KIND_HISTORY_WINDOW =
      ResourceKind._(4, _omitEnumNames ? '' : 'RESOURCE_KIND_HISTORY_WINDOW');
  static const ResourceKind RESOURCE_KIND_FILE_TRANSFER =
      ResourceKind._(5, _omitEnumNames ? '' : 'RESOURCE_KIND_FILE_TRANSFER');
  static const ResourceKind RESOURCE_KIND_BROWSER_PROXY =
      ResourceKind._(6, _omitEnumNames ? '' : 'RESOURCE_KIND_BROWSER_PROXY');

  static const $core.List<ResourceKind> values = <ResourceKind>[
    RESOURCE_KIND_UNSPECIFIED,
    RESOURCE_KIND_OPERATION,
    RESOURCE_KIND_SUBSCRIPTION,
    RESOURCE_KIND_TERMINAL_ATTACHMENT,
    RESOURCE_KIND_HISTORY_WINDOW,
    RESOURCE_KIND_FILE_TRANSFER,
    RESOURCE_KIND_BROWSER_PROXY,
  ];

  static final $core.List<ResourceKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static ResourceKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ResourceKind._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
