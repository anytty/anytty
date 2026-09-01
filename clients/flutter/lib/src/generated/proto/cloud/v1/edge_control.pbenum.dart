// This is a generated file - do not edit.
//
// Generated from cloud/v1/edge_control.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// EdgeCapability 是 Edge 进程已实现并愿意承诺的版本化能力。
class EdgeCapability extends $pb.ProtobufEnum {
  static const EdgeCapability EDGE_CAPABILITY_UNSPECIFIED =
      EdgeCapability._(0, _omitEnumNames ? '' : 'EDGE_CAPABILITY_UNSPECIFIED');
  static const EdgeCapability EDGE_CAPABILITY_CONTROL_STREAM = EdgeCapability._(
      1, _omitEnumNames ? '' : 'EDGE_CAPABILITY_CONTROL_STREAM');
  static const EdgeCapability EDGE_CAPABILITY_RELAY =
      EdgeCapability._(2, _omitEnumNames ? '' : 'EDGE_CAPABILITY_RELAY');
  static const EdgeCapability EDGE_CAPABILITY_RESERVATION_JOURNAL =
      EdgeCapability._(
          3, _omitEnumNames ? '' : 'EDGE_CAPABILITY_RESERVATION_JOURNAL');
  static const EdgeCapability EDGE_CAPABILITY_PUBLIC_CERTIFICATE_ROTATION =
      EdgeCapability._(4,
          _omitEnumNames ? '' : 'EDGE_CAPABILITY_PUBLIC_CERTIFICATE_ROTATION');
  static const EdgeCapability EDGE_CAPABILITY_DAEMON_LIFECYCLE_POLICY =
      EdgeCapability._(
          5, _omitEnumNames ? '' : 'EDGE_CAPABILITY_DAEMON_LIFECYCLE_POLICY');
  static const EdgeCapability EDGE_CAPABILITY_DAEMON_EDGE_RESELECTION =
      EdgeCapability._(
          6, _omitEnumNames ? '' : 'EDGE_CAPABILITY_DAEMON_EDGE_RESELECTION');
  static const EdgeCapability EDGE_CAPABILITY_IDENTITY_CERTIFICATE_ROTATION =
      EdgeCapability._(
          7,
          _omitEnumNames
              ? ''
              : 'EDGE_CAPABILITY_IDENTITY_CERTIFICATE_ROTATION');
  static const EdgeCapability EDGE_CAPABILITY_DAEMON_CONNECTION_ADMISSION =
      EdgeCapability._(8,
          _omitEnumNames ? '' : 'EDGE_CAPABILITY_DAEMON_CONNECTION_ADMISSION');
  static const EdgeCapability EDGE_CAPABILITY_RELAY_USAGE_BATCH_V1 =
      EdgeCapability._(
          9, _omitEnumNames ? '' : 'EDGE_CAPABILITY_RELAY_USAGE_BATCH_V1');
  static const EdgeCapability EDGE_CAPABILITY_RELAY_LOCAL_ADMISSION_V1 =
      EdgeCapability._(
          10, _omitEnumNames ? '' : 'EDGE_CAPABILITY_RELAY_LOCAL_ADMISSION_V1');
  static const EdgeCapability EDGE_CAPABILITY_SCOPED_DAEMON_STATE_SYNC =
      EdgeCapability._(
          11, _omitEnumNames ? '' : 'EDGE_CAPABILITY_SCOPED_DAEMON_STATE_SYNC');

  static const $core.List<EdgeCapability> values = <EdgeCapability>[
    EDGE_CAPABILITY_UNSPECIFIED,
    EDGE_CAPABILITY_CONTROL_STREAM,
    EDGE_CAPABILITY_RELAY,
    EDGE_CAPABILITY_RESERVATION_JOURNAL,
    EDGE_CAPABILITY_PUBLIC_CERTIFICATE_ROTATION,
    EDGE_CAPABILITY_DAEMON_LIFECYCLE_POLICY,
    EDGE_CAPABILITY_DAEMON_EDGE_RESELECTION,
    EDGE_CAPABILITY_IDENTITY_CERTIFICATE_ROTATION,
    EDGE_CAPABILITY_DAEMON_CONNECTION_ADMISSION,
    EDGE_CAPABILITY_RELAY_USAGE_BATCH_V1,
    EDGE_CAPABILITY_RELAY_LOCAL_ADMISSION_V1,
    EDGE_CAPABILITY_SCOPED_DAEMON_STATE_SYNC,
  ];

  static final $core.List<EdgeCapability?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 11);
  static EdgeCapability? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EdgeCapability._(super.value, super.name);
}

class DaemonConnectionAdmissionResult extends $pb.ProtobufEnum {
  static const DaemonConnectionAdmissionResult
      DAEMON_CONNECTION_ADMISSION_RESULT_UNSPECIFIED =
      DaemonConnectionAdmissionResult._(
          0,
          _omitEnumNames
              ? ''
              : 'DAEMON_CONNECTION_ADMISSION_RESULT_UNSPECIFIED');
  static const DaemonConnectionAdmissionResult
      DAEMON_CONNECTION_ADMISSION_RESULT_ADMITTED =
      DaemonConnectionAdmissionResult._(1,
          _omitEnumNames ? '' : 'DAEMON_CONNECTION_ADMISSION_RESULT_ADMITTED');
  static const DaemonConnectionAdmissionResult
      DAEMON_CONNECTION_ADMISSION_RESULT_RELEASED =
      DaemonConnectionAdmissionResult._(2,
          _omitEnumNames ? '' : 'DAEMON_CONNECTION_ADMISSION_RESULT_RELEASED');
  static const DaemonConnectionAdmissionResult
      DAEMON_CONNECTION_ADMISSION_RESULT_LIMIT_REACHED =
      DaemonConnectionAdmissionResult._(
          3,
          _omitEnumNames
              ? ''
              : 'DAEMON_CONNECTION_ADMISSION_RESULT_LIMIT_REACHED');
  static const DaemonConnectionAdmissionResult
      DAEMON_CONNECTION_ADMISSION_RESULT_REJECTED =
      DaemonConnectionAdmissionResult._(4,
          _omitEnumNames ? '' : 'DAEMON_CONNECTION_ADMISSION_RESULT_REJECTED');
  static const DaemonConnectionAdmissionResult
      DAEMON_CONNECTION_ADMISSION_RESULT_UNAVAILABLE =
      DaemonConnectionAdmissionResult._(
          5,
          _omitEnumNames
              ? ''
              : 'DAEMON_CONNECTION_ADMISSION_RESULT_UNAVAILABLE');

  static const $core.List<DaemonConnectionAdmissionResult> values =
      <DaemonConnectionAdmissionResult>[
    DAEMON_CONNECTION_ADMISSION_RESULT_UNSPECIFIED,
    DAEMON_CONNECTION_ADMISSION_RESULT_ADMITTED,
    DAEMON_CONNECTION_ADMISSION_RESULT_RELEASED,
    DAEMON_CONNECTION_ADMISSION_RESULT_LIMIT_REACHED,
    DAEMON_CONNECTION_ADMISSION_RESULT_REJECTED,
    DAEMON_CONNECTION_ADMISSION_RESULT_UNAVAILABLE,
  ];

  static final $core.List<DaemonConnectionAdmissionResult?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static DaemonConnectionAdmissionResult? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const DaemonConnectionAdmissionResult._(super.value, super.name);
}

class CommandResultCode extends $pb.ProtobufEnum {
  static const CommandResultCode COMMAND_RESULT_CODE_UNSPECIFIED =
      CommandResultCode._(
          0, _omitEnumNames ? '' : 'COMMAND_RESULT_CODE_UNSPECIFIED');
  static const CommandResultCode COMMAND_RESULT_CODE_APPLIED =
      CommandResultCode._(
          1, _omitEnumNames ? '' : 'COMMAND_RESULT_CODE_APPLIED');
  static const CommandResultCode COMMAND_RESULT_CODE_REJECTED =
      CommandResultCode._(
          2, _omitEnumNames ? '' : 'COMMAND_RESULT_CODE_REJECTED');
  static const CommandResultCode COMMAND_RESULT_CODE_STALE =
      CommandResultCode._(3, _omitEnumNames ? '' : 'COMMAND_RESULT_CODE_STALE');

  static const $core.List<CommandResultCode> values = <CommandResultCode>[
    COMMAND_RESULT_CODE_UNSPECIFIED,
    COMMAND_RESULT_CODE_APPLIED,
    COMMAND_RESULT_CODE_REJECTED,
    COMMAND_RESULT_CODE_STALE,
  ];

  static final $core.List<CommandResultCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static CommandResultCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CommandResultCode._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
