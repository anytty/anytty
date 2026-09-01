// This is a generated file - do not edit.
//
// Generated from apipb/runtime.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class EndpointRuntimePhase extends $pb.ProtobufEnum {
  static const EndpointRuntimePhase ENDPOINT_RUNTIME_PHASE_UNSPECIFIED =
      EndpointRuntimePhase._(
          0, _omitEnumNames ? '' : 'ENDPOINT_RUNTIME_PHASE_UNSPECIFIED');
  static const EndpointRuntimePhase ENDPOINT_RUNTIME_PHASE_CONNECTING =
      EndpointRuntimePhase._(
          1, _omitEnumNames ? '' : 'ENDPOINT_RUNTIME_PHASE_CONNECTING');
  static const EndpointRuntimePhase ENDPOINT_RUNTIME_PHASE_READY =
      EndpointRuntimePhase._(
          2, _omitEnumNames ? '' : 'ENDPOINT_RUNTIME_PHASE_READY');
  static const EndpointRuntimePhase ENDPOINT_RUNTIME_PHASE_FAILED =
      EndpointRuntimePhase._(
          3, _omitEnumNames ? '' : 'ENDPOINT_RUNTIME_PHASE_FAILED');
  static const EndpointRuntimePhase ENDPOINT_RUNTIME_PHASE_CLOSED =
      EndpointRuntimePhase._(
          4, _omitEnumNames ? '' : 'ENDPOINT_RUNTIME_PHASE_CLOSED');

  static const $core.List<EndpointRuntimePhase> values = <EndpointRuntimePhase>[
    ENDPOINT_RUNTIME_PHASE_UNSPECIFIED,
    ENDPOINT_RUNTIME_PHASE_CONNECTING,
    ENDPOINT_RUNTIME_PHASE_READY,
    ENDPOINT_RUNTIME_PHASE_FAILED,
    ENDPOINT_RUNTIME_PHASE_CLOSED,
  ];

  static final $core.List<EndpointRuntimePhase?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static EndpointRuntimePhase? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EndpointRuntimePhase._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
