// This is a generated file - do not edit.
//
// Generated from apipb/events.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class ApplicationEventType extends $pb.ProtobufEnum {
  static const ApplicationEventType APPLICATION_EVENT_TYPE_UNSPECIFIED =
      ApplicationEventType._(
          0, _omitEnumNames ? '' : 'APPLICATION_EVENT_TYPE_UNSPECIFIED');
  static const ApplicationEventType APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE =
      ApplicationEventType._(
          1, _omitEnumNames ? '' : 'APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE');
  static const ApplicationEventType APPLICATION_EVENT_TYPE_STORAGE_CHANGED =
      ApplicationEventType._(
          5, _omitEnumNames ? '' : 'APPLICATION_EVENT_TYPE_STORAGE_CHANGED');

  static const $core.List<ApplicationEventType> values = <ApplicationEventType>[
    APPLICATION_EVENT_TYPE_UNSPECIFIED,
    APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE,
    APPLICATION_EVENT_TYPE_STORAGE_CHANGED,
  ];

  static final $core.Map<$core.int, ApplicationEventType> _byValue =
      $pb.ProtobufEnum.initByValue(values);
  static ApplicationEventType? valueOf($core.int value) => _byValue[value];

  const ApplicationEventType._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
