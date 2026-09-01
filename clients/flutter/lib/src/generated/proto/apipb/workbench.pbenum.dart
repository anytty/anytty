// This is a generated file - do not edit.
//
// Generated from apipb/workbench.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class WorkbenchSplitDirection extends $pb.ProtobufEnum {
  static const WorkbenchSplitDirection WORKBENCH_SPLIT_DIRECTION_UNSPECIFIED =
      WorkbenchSplitDirection._(
          0, _omitEnumNames ? '' : 'WORKBENCH_SPLIT_DIRECTION_UNSPECIFIED');
  static const WorkbenchSplitDirection WORKBENCH_SPLIT_DIRECTION_HORIZONTAL =
      WorkbenchSplitDirection._(
          1, _omitEnumNames ? '' : 'WORKBENCH_SPLIT_DIRECTION_HORIZONTAL');
  static const WorkbenchSplitDirection WORKBENCH_SPLIT_DIRECTION_VERTICAL =
      WorkbenchSplitDirection._(
          2, _omitEnumNames ? '' : 'WORKBENCH_SPLIT_DIRECTION_VERTICAL');

  static const $core.List<WorkbenchSplitDirection> values =
      <WorkbenchSplitDirection>[
    WORKBENCH_SPLIT_DIRECTION_UNSPECIFIED,
    WORKBENCH_SPLIT_DIRECTION_HORIZONTAL,
    WORKBENCH_SPLIT_DIRECTION_VERTICAL,
  ];

  static final $core.List<WorkbenchSplitDirection?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static WorkbenchSplitDirection? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const WorkbenchSplitDirection._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
