// This is a generated file - do not edit.
//
// Generated from cloud/v1/operator.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class RuntimeCommandResult extends $pb.ProtobufEnum {
  static const RuntimeCommandResult RUNTIME_COMMAND_RESULT_UNSPECIFIED =
      RuntimeCommandResult._(
          0, _omitEnumNames ? '' : 'RUNTIME_COMMAND_RESULT_UNSPECIFIED');
  static const RuntimeCommandResult RUNTIME_COMMAND_RESULT_APPLIED =
      RuntimeCommandResult._(
          1, _omitEnumNames ? '' : 'RUNTIME_COMMAND_RESULT_APPLIED');
  static const RuntimeCommandResult RUNTIME_COMMAND_RESULT_REJECTED =
      RuntimeCommandResult._(
          2, _omitEnumNames ? '' : 'RUNTIME_COMMAND_RESULT_REJECTED');
  static const RuntimeCommandResult RUNTIME_COMMAND_RESULT_STALE =
      RuntimeCommandResult._(
          3, _omitEnumNames ? '' : 'RUNTIME_COMMAND_RESULT_STALE');
  static const RuntimeCommandResult RUNTIME_COMMAND_RESULT_TIMEOUT =
      RuntimeCommandResult._(
          4, _omitEnumNames ? '' : 'RUNTIME_COMMAND_RESULT_TIMEOUT');
  static const RuntimeCommandResult RUNTIME_COMMAND_RESULT_UNAVAILABLE =
      RuntimeCommandResult._(
          5, _omitEnumNames ? '' : 'RUNTIME_COMMAND_RESULT_UNAVAILABLE');

  static const $core.List<RuntimeCommandResult> values = <RuntimeCommandResult>[
    RUNTIME_COMMAND_RESULT_UNSPECIFIED,
    RUNTIME_COMMAND_RESULT_APPLIED,
    RUNTIME_COMMAND_RESULT_REJECTED,
    RUNTIME_COMMAND_RESULT_STALE,
    RUNTIME_COMMAND_RESULT_TIMEOUT,
    RUNTIME_COMMAND_RESULT_UNAVAILABLE,
  ];

  static final $core.List<RuntimeCommandResult?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static RuntimeCommandResult? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RuntimeCommandResult._(super.value, super.name);
}

class OperatorEventOperation extends $pb.ProtobufEnum {
  static const OperatorEventOperation OPERATOR_EVENT_OPERATION_UNSPECIFIED =
      OperatorEventOperation._(
          0, _omitEnumNames ? '' : 'OPERATOR_EVENT_OPERATION_UNSPECIFIED');
  static const OperatorEventOperation OPERATOR_EVENT_OPERATION_UPSERT =
      OperatorEventOperation._(
          1, _omitEnumNames ? '' : 'OPERATOR_EVENT_OPERATION_UPSERT');
  static const OperatorEventOperation OPERATOR_EVENT_OPERATION_DELETE =
      OperatorEventOperation._(
          2, _omitEnumNames ? '' : 'OPERATOR_EVENT_OPERATION_DELETE');
  static const OperatorEventOperation OPERATOR_EVENT_OPERATION_RESET =
      OperatorEventOperation._(
          3, _omitEnumNames ? '' : 'OPERATOR_EVENT_OPERATION_RESET');

  static const $core.List<OperatorEventOperation> values =
      <OperatorEventOperation>[
    OPERATOR_EVENT_OPERATION_UNSPECIFIED,
    OPERATOR_EVENT_OPERATION_UPSERT,
    OPERATOR_EVENT_OPERATION_DELETE,
    OPERATOR_EVENT_OPERATION_RESET,
  ];

  static final $core.List<OperatorEventOperation?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static OperatorEventOperation? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const OperatorEventOperation._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
