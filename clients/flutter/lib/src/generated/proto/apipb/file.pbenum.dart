// This is a generated file - do not edit.
//
// Generated from apipb/file.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class FileEntryType extends $pb.ProtobufEnum {
  static const FileEntryType FILE_ENTRY_TYPE_UNSPECIFIED =
      FileEntryType._(0, _omitEnumNames ? '' : 'FILE_ENTRY_TYPE_UNSPECIFIED');
  static const FileEntryType FILE_ENTRY_TYPE_FILE =
      FileEntryType._(1, _omitEnumNames ? '' : 'FILE_ENTRY_TYPE_FILE');
  static const FileEntryType FILE_ENTRY_TYPE_DIRECTORY =
      FileEntryType._(2, _omitEnumNames ? '' : 'FILE_ENTRY_TYPE_DIRECTORY');
  static const FileEntryType FILE_ENTRY_TYPE_SYMLINK =
      FileEntryType._(3, _omitEnumNames ? '' : 'FILE_ENTRY_TYPE_SYMLINK');
  static const FileEntryType FILE_ENTRY_TYPE_OTHER =
      FileEntryType._(4, _omitEnumNames ? '' : 'FILE_ENTRY_TYPE_OTHER');

  static const $core.List<FileEntryType> values = <FileEntryType>[
    FILE_ENTRY_TYPE_UNSPECIFIED,
    FILE_ENTRY_TYPE_FILE,
    FILE_ENTRY_TYPE_DIRECTORY,
    FILE_ENTRY_TYPE_SYMLINK,
    FILE_ENTRY_TYPE_OTHER,
  ];

  static final $core.List<FileEntryType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static FileEntryType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const FileEntryType._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
