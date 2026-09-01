// This is a generated file - do not edit.
//
// Generated from cloud/v1/ticket.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// DaemonCapability 是 enrollment 时授予 daemon 的 Edge 侧能力，不表达 terminal 权限。
class DaemonCapability extends $pb.ProtobufEnum {
  static const DaemonCapability DAEMON_CAPABILITY_UNSPECIFIED =
      DaemonCapability._(
          0, _omitEnumNames ? '' : 'DAEMON_CAPABILITY_UNSPECIFIED');
  static const DaemonCapability DAEMON_CAPABILITY_SIGNALING =
      DaemonCapability._(
          1, _omitEnumNames ? '' : 'DAEMON_CAPABILITY_SIGNALING');

  static const $core.List<DaemonCapability> values = <DaemonCapability>[
    DAEMON_CAPABILITY_UNSPECIFIED,
    DAEMON_CAPABILITY_SIGNALING,
  ];

  static final $core.List<DaemonCapability?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 1);
  static DaemonCapability? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const DaemonCapability._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
