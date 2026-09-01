// This is a generated file - do not edit.
//
// Generated from cloud/v1/client_gateway.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class SelectedCloudPath extends $pb.ProtobufEnum {
  static const SelectedCloudPath SELECTED_CLOUD_PATH_UNSPECIFIED =
      SelectedCloudPath._(
          0, _omitEnumNames ? '' : 'SELECTED_CLOUD_PATH_UNSPECIFIED');
  static const SelectedCloudPath SELECTED_CLOUD_PATH_DIRECT =
      SelectedCloudPath._(
          1, _omitEnumNames ? '' : 'SELECTED_CLOUD_PATH_DIRECT');
  static const SelectedCloudPath SELECTED_CLOUD_PATH_RELAY =
      SelectedCloudPath._(2, _omitEnumNames ? '' : 'SELECTED_CLOUD_PATH_RELAY');

  static const $core.List<SelectedCloudPath> values = <SelectedCloudPath>[
    SELECTED_CLOUD_PATH_UNSPECIFIED,
    SELECTED_CLOUD_PATH_DIRECT,
    SELECTED_CLOUD_PATH_RELAY,
  ];

  static final $core.List<SelectedCloudPath?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static SelectedCloudPath? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const SelectedCloudPath._(super.value, super.name);
}

class CloudPathDecision extends $pb.ProtobufEnum {
  static const CloudPathDecision CLOUD_PATH_DECISION_UNSPECIFIED =
      CloudPathDecision._(
          0, _omitEnumNames ? '' : 'CLOUD_PATH_DECISION_UNSPECIFIED');
  static const CloudPathDecision CLOUD_PATH_DECISION_CONFIRM_DIRECT =
      CloudPathDecision._(
          1, _omitEnumNames ? '' : 'CLOUD_PATH_DECISION_CONFIRM_DIRECT');
  static const CloudPathDecision CLOUD_PATH_DECISION_CONFIRM_RELAY =
      CloudPathDecision._(
          2, _omitEnumNames ? '' : 'CLOUD_PATH_DECISION_CONFIRM_RELAY');
  static const CloudPathDecision CLOUD_PATH_DECISION_ABANDON =
      CloudPathDecision._(
          3, _omitEnumNames ? '' : 'CLOUD_PATH_DECISION_ABANDON');

  static const $core.List<CloudPathDecision> values = <CloudPathDecision>[
    CLOUD_PATH_DECISION_UNSPECIFIED,
    CLOUD_PATH_DECISION_CONFIRM_DIRECT,
    CLOUD_PATH_DECISION_CONFIRM_RELAY,
    CLOUD_PATH_DECISION_ABANDON,
  ];

  static final $core.List<CloudPathDecision?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static CloudPathDecision? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CloudPathDecision._(super.value, super.name);
}

class SignalSessionCloseCode extends $pb.ProtobufEnum {
  static const SignalSessionCloseCode SIGNAL_SESSION_CLOSE_CODE_UNSPECIFIED =
      SignalSessionCloseCode._(
          0, _omitEnumNames ? '' : 'SIGNAL_SESSION_CLOSE_CODE_UNSPECIFIED');
  static const SignalSessionCloseCode
      SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT = SignalSessionCloseCode._(1,
          _omitEnumNames ? '' : 'SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT');

  static const $core.List<SignalSessionCloseCode> values =
      <SignalSessionCloseCode>[
    SIGNAL_SESSION_CLOSE_CODE_UNSPECIFIED,
    SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT,
  ];

  static final $core.List<SignalSessionCloseCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 1);
  static SignalSessionCloseCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const SignalSessionCloseCode._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
