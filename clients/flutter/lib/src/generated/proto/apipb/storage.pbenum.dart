// This is a generated file - do not edit.
//
// Generated from apipb/storage.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class StorageScope extends $pb.ProtobufEnum {
  static const StorageScope STORAGE_SCOPE_UNSPECIFIED =
      StorageScope._(0, _omitEnumNames ? '' : 'STORAGE_SCOPE_UNSPECIFIED');
  static const StorageScope STORAGE_SCOPE_PUBLIC =
      StorageScope._(1, _omitEnumNames ? '' : 'STORAGE_SCOPE_PUBLIC');
  static const StorageScope STORAGE_SCOPE_PRIVATE =
      StorageScope._(2, _omitEnumNames ? '' : 'STORAGE_SCOPE_PRIVATE');

  static const $core.List<StorageScope> values = <StorageScope>[
    STORAGE_SCOPE_UNSPECIFIED,
    STORAGE_SCOPE_PUBLIC,
    STORAGE_SCOPE_PRIVATE,
  ];

  static final $core.List<StorageScope?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static StorageScope? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const StorageScope._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
