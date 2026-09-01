// This is a generated file - do not edit.
//
// Generated from cloud/v1/enrollment.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// DaemonState 是 daemon identity 的持久生命周期。DELETED 是不可恢复的终态墓碑。
class DaemonState extends $pb.ProtobufEnum {
  static const DaemonState DAEMON_STATE_UNSPECIFIED =
      DaemonState._(0, _omitEnumNames ? '' : 'DAEMON_STATE_UNSPECIFIED');
  static const DaemonState DAEMON_STATE_ACTIVE =
      DaemonState._(1, _omitEnumNames ? '' : 'DAEMON_STATE_ACTIVE');
  static const DaemonState DAEMON_STATE_BLOCKED =
      DaemonState._(2, _omitEnumNames ? '' : 'DAEMON_STATE_BLOCKED');
  static const DaemonState DAEMON_STATE_DELETED =
      DaemonState._(3, _omitEnumNames ? '' : 'DAEMON_STATE_DELETED');

  static const $core.List<DaemonState> values = <DaemonState>[
    DAEMON_STATE_UNSPECIFIED,
    DAEMON_STATE_ACTIVE,
    DAEMON_STATE_BLOCKED,
    DAEMON_STATE_DELETED,
  ];

  static final $core.List<DaemonState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static DaemonState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const DaemonState._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
