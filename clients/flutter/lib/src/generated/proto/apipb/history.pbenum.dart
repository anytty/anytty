// This is a generated file - do not edit.
//
// Generated from apipb/history.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class HistoryWindowMode extends $pb.ProtobufEnum {
  static const HistoryWindowMode HISTORY_WINDOW_MODE_UNSPECIFIED =
      HistoryWindowMode._(
          0, _omitEnumNames ? '' : 'HISTORY_WINDOW_MODE_UNSPECIFIED');
  static const HistoryWindowMode HISTORY_WINDOW_MODE_LATEST =
      HistoryWindowMode._(
          1, _omitEnumNames ? '' : 'HISTORY_WINDOW_MODE_LATEST');
  static const HistoryWindowMode HISTORY_WINDOW_MODE_OLDER =
      HistoryWindowMode._(2, _omitEnumNames ? '' : 'HISTORY_WINDOW_MODE_OLDER');
  static const HistoryWindowMode HISTORY_WINDOW_MODE_NEWER =
      HistoryWindowMode._(3, _omitEnumNames ? '' : 'HISTORY_WINDOW_MODE_NEWER');
  static const HistoryWindowMode HISTORY_WINDOW_MODE_OLDEST =
      HistoryWindowMode._(
          4, _omitEnumNames ? '' : 'HISTORY_WINDOW_MODE_OLDEST');

  static const $core.List<HistoryWindowMode> values = <HistoryWindowMode>[
    HISTORY_WINDOW_MODE_UNSPECIFIED,
    HISTORY_WINDOW_MODE_LATEST,
    HISTORY_WINDOW_MODE_OLDER,
    HISTORY_WINDOW_MODE_NEWER,
    HISTORY_WINDOW_MODE_OLDEST,
  ];

  static final $core.List<HistoryWindowMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static HistoryWindowMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const HistoryWindowMode._(super.value, super.name);
}

class HistoryWindowOperation extends $pb.ProtobufEnum {
  static const HistoryWindowOperation HISTORY_WINDOW_OPERATION_UNSPECIFIED =
      HistoryWindowOperation._(
          0, _omitEnumNames ? '' : 'HISTORY_WINDOW_OPERATION_UNSPECIFIED');
  static const HistoryWindowOperation HISTORY_WINDOW_OPERATION_REPLACE =
      HistoryWindowOperation._(
          1, _omitEnumNames ? '' : 'HISTORY_WINDOW_OPERATION_REPLACE');
  static const HistoryWindowOperation HISTORY_WINDOW_OPERATION_PREPEND =
      HistoryWindowOperation._(
          2, _omitEnumNames ? '' : 'HISTORY_WINDOW_OPERATION_PREPEND');
  static const HistoryWindowOperation HISTORY_WINDOW_OPERATION_APPEND =
      HistoryWindowOperation._(
          3, _omitEnumNames ? '' : 'HISTORY_WINDOW_OPERATION_APPEND');

  static const $core.List<HistoryWindowOperation> values =
      <HistoryWindowOperation>[
    HISTORY_WINDOW_OPERATION_UNSPECIFIED,
    HISTORY_WINDOW_OPERATION_REPLACE,
    HISTORY_WINDOW_OPERATION_PREPEND,
    HISTORY_WINDOW_OPERATION_APPEND,
  ];

  static final $core.List<HistoryWindowOperation?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static HistoryWindowOperation? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const HistoryWindowOperation._(super.value, super.name);
}

class HistorySearchDirection extends $pb.ProtobufEnum {
  static const HistorySearchDirection HISTORY_SEARCH_DIRECTION_UNSPECIFIED =
      HistorySearchDirection._(
          0, _omitEnumNames ? '' : 'HISTORY_SEARCH_DIRECTION_UNSPECIFIED');
  static const HistorySearchDirection HISTORY_SEARCH_DIRECTION_FORWARD =
      HistorySearchDirection._(
          1, _omitEnumNames ? '' : 'HISTORY_SEARCH_DIRECTION_FORWARD');
  static const HistorySearchDirection HISTORY_SEARCH_DIRECTION_BACKWARD =
      HistorySearchDirection._(
          2, _omitEnumNames ? '' : 'HISTORY_SEARCH_DIRECTION_BACKWARD');

  static const $core.List<HistorySearchDirection> values =
      <HistorySearchDirection>[
    HISTORY_SEARCH_DIRECTION_UNSPECIFIED,
    HISTORY_SEARCH_DIRECTION_FORWARD,
    HISTORY_SEARCH_DIRECTION_BACKWARD,
  ];

  static final $core.List<HistorySearchDirection?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static HistorySearchDirection? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const HistorySearchDirection._(super.value, super.name);
}

class HistorySearchMode extends $pb.ProtobufEnum {
  static const HistorySearchMode HISTORY_SEARCH_MODE_UNSPECIFIED =
      HistorySearchMode._(
          0, _omitEnumNames ? '' : 'HISTORY_SEARCH_MODE_UNSPECIFIED');
  static const HistorySearchMode HISTORY_SEARCH_MODE_TEXT =
      HistorySearchMode._(1, _omitEnumNames ? '' : 'HISTORY_SEARCH_MODE_TEXT');
  static const HistorySearchMode HISTORY_SEARCH_MODE_GLOB =
      HistorySearchMode._(2, _omitEnumNames ? '' : 'HISTORY_SEARCH_MODE_GLOB');
  static const HistorySearchMode HISTORY_SEARCH_MODE_REGEX =
      HistorySearchMode._(3, _omitEnumNames ? '' : 'HISTORY_SEARCH_MODE_REGEX');

  static const $core.List<HistorySearchMode> values = <HistorySearchMode>[
    HISTORY_SEARCH_MODE_UNSPECIFIED,
    HISTORY_SEARCH_MODE_TEXT,
    HISTORY_SEARCH_MODE_GLOB,
    HISTORY_SEARCH_MODE_REGEX,
  ];

  static final $core.List<HistorySearchMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static HistorySearchMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const HistorySearchMode._(super.value, super.name);
}

class HistoryCursorSegment extends $pb.ProtobufEnum {
  static const HistoryCursorSegment HISTORY_CURSOR_SEGMENT_UNSPECIFIED =
      HistoryCursorSegment._(
          0, _omitEnumNames ? '' : 'HISTORY_CURSOR_SEGMENT_UNSPECIFIED');
  static const HistoryCursorSegment HISTORY_CURSOR_SEGMENT_COMMITTED =
      HistoryCursorSegment._(
          1, _omitEnumNames ? '' : 'HISTORY_CURSOR_SEGMENT_COMMITTED');
  static const HistoryCursorSegment
      HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME = HistoryCursorSegment._(2,
          _omitEnumNames ? '' : 'HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME');
  static const HistoryCursorSegment
      HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME = HistoryCursorSegment._(
          3,
          _omitEnumNames
              ? ''
              : 'HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME');
  static const HistoryCursorSegment HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME =
      HistoryCursorSegment._(
          4, _omitEnumNames ? '' : 'HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME');

  static const $core.List<HistoryCursorSegment> values = <HistoryCursorSegment>[
    HISTORY_CURSOR_SEGMENT_UNSPECIFIED,
    HISTORY_CURSOR_SEGMENT_COMMITTED,
    HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME,
    HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME,
    HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME,
  ];

  static final $core.List<HistoryCursorSegment?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static HistoryCursorSegment? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const HistoryCursorSegment._(super.value, super.name);
}

class RowOwnership extends $pb.ProtobufEnum {
  static const RowOwnership ROW_OWNERSHIP_UNSPECIFIED =
      RowOwnership._(0, _omitEnumNames ? '' : 'ROW_OWNERSHIP_UNSPECIFIED');
  static const RowOwnership ROW_OWNERSHIP_PERSISTED =
      RowOwnership._(1, _omitEnumNames ? '' : 'ROW_OWNERSHIP_PERSISTED');
  static const RowOwnership ROW_OWNERSHIP_LIVE_TAIL_RECLAIMED = RowOwnership._(
      2, _omitEnumNames ? '' : 'ROW_OWNERSHIP_LIVE_TAIL_RECLAIMED');
  static const RowOwnership ROW_OWNERSHIP_LIVE_TAIL_LIVE =
      RowOwnership._(3, _omitEnumNames ? '' : 'ROW_OWNERSHIP_LIVE_TAIL_LIVE');
  static const RowOwnership ROW_OWNERSHIP_SCREEN =
      RowOwnership._(4, _omitEnumNames ? '' : 'ROW_OWNERSHIP_SCREEN');

  static const $core.List<RowOwnership> values = <RowOwnership>[
    ROW_OWNERSHIP_UNSPECIFIED,
    ROW_OWNERSHIP_PERSISTED,
    ROW_OWNERSHIP_LIVE_TAIL_RECLAIMED,
    ROW_OWNERSHIP_LIVE_TAIL_LIVE,
    ROW_OWNERSHIP_SCREEN,
  ];

  static final $core.List<RowOwnership?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static RowOwnership? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RowOwnership._(super.value, super.name);
}

class CursorShape extends $pb.ProtobufEnum {
  static const CursorShape CURSOR_SHAPE_UNSPECIFIED =
      CursorShape._(0, _omitEnumNames ? '' : 'CURSOR_SHAPE_UNSPECIFIED');
  static const CursorShape CURSOR_SHAPE_BLOCK =
      CursorShape._(1, _omitEnumNames ? '' : 'CURSOR_SHAPE_BLOCK');
  static const CursorShape CURSOR_SHAPE_UNDERLINE =
      CursorShape._(2, _omitEnumNames ? '' : 'CURSOR_SHAPE_UNDERLINE');
  static const CursorShape CURSOR_SHAPE_BAR =
      CursorShape._(3, _omitEnumNames ? '' : 'CURSOR_SHAPE_BAR');

  static const $core.List<CursorShape> values = <CursorShape>[
    CURSOR_SHAPE_UNSPECIFIED,
    CURSOR_SHAPE_BLOCK,
    CURSOR_SHAPE_UNDERLINE,
    CURSOR_SHAPE_BAR,
  ];

  static final $core.List<CursorShape?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static CursorShape? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CursorShape._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
