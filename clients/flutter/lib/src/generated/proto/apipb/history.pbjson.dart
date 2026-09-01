// This is a generated file - do not edit.
//
// Generated from apipb/history.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use historyWindowModeDescriptor instead')
const HistoryWindowMode$json = {
  '1': 'HistoryWindowMode',
  '2': [
    {'1': 'HISTORY_WINDOW_MODE_UNSPECIFIED', '2': 0},
    {'1': 'HISTORY_WINDOW_MODE_LATEST', '2': 1},
    {'1': 'HISTORY_WINDOW_MODE_OLDER', '2': 2},
    {'1': 'HISTORY_WINDOW_MODE_NEWER', '2': 3},
    {'1': 'HISTORY_WINDOW_MODE_OLDEST', '2': 4},
  ],
};

/// Descriptor for `HistoryWindowMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List historyWindowModeDescriptor = $convert.base64Decode(
    'ChFIaXN0b3J5V2luZG93TW9kZRIjCh9ISVNUT1JZX1dJTkRPV19NT0RFX1VOU1BFQ0lGSUVEEA'
    'ASHgoaSElTVE9SWV9XSU5ET1dfTU9ERV9MQVRFU1QQARIdChlISVNUT1JZX1dJTkRPV19NT0RF'
    'X09MREVSEAISHQoZSElTVE9SWV9XSU5ET1dfTU9ERV9ORVdFUhADEh4KGkhJU1RPUllfV0lORE'
    '9XX01PREVfT0xERVNUEAQ=');

@$core.Deprecated('Use historyWindowOperationDescriptor instead')
const HistoryWindowOperation$json = {
  '1': 'HistoryWindowOperation',
  '2': [
    {'1': 'HISTORY_WINDOW_OPERATION_UNSPECIFIED', '2': 0},
    {'1': 'HISTORY_WINDOW_OPERATION_REPLACE', '2': 1},
    {'1': 'HISTORY_WINDOW_OPERATION_PREPEND', '2': 2},
    {'1': 'HISTORY_WINDOW_OPERATION_APPEND', '2': 3},
  ],
};

/// Descriptor for `HistoryWindowOperation`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List historyWindowOperationDescriptor = $convert.base64Decode(
    'ChZIaXN0b3J5V2luZG93T3BlcmF0aW9uEigKJEhJU1RPUllfV0lORE9XX09QRVJBVElPTl9VTl'
    'NQRUNJRklFRBAAEiQKIEhJU1RPUllfV0lORE9XX09QRVJBVElPTl9SRVBMQUNFEAESJAogSElT'
    'VE9SWV9XSU5ET1dfT1BFUkFUSU9OX1BSRVBFTkQQAhIjCh9ISVNUT1JZX1dJTkRPV19PUEVSQV'
    'RJT05fQVBQRU5EEAM=');

@$core.Deprecated('Use historySearchDirectionDescriptor instead')
const HistorySearchDirection$json = {
  '1': 'HistorySearchDirection',
  '2': [
    {'1': 'HISTORY_SEARCH_DIRECTION_UNSPECIFIED', '2': 0},
    {'1': 'HISTORY_SEARCH_DIRECTION_FORWARD', '2': 1},
    {'1': 'HISTORY_SEARCH_DIRECTION_BACKWARD', '2': 2},
  ],
};

/// Descriptor for `HistorySearchDirection`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List historySearchDirectionDescriptor = $convert.base64Decode(
    'ChZIaXN0b3J5U2VhcmNoRGlyZWN0aW9uEigKJEhJU1RPUllfU0VBUkNIX0RJUkVDVElPTl9VTl'
    'NQRUNJRklFRBAAEiQKIEhJU1RPUllfU0VBUkNIX0RJUkVDVElPTl9GT1JXQVJEEAESJQohSElT'
    'VE9SWV9TRUFSQ0hfRElSRUNUSU9OX0JBQ0tXQVJEEAI=');

@$core.Deprecated('Use historySearchModeDescriptor instead')
const HistorySearchMode$json = {
  '1': 'HistorySearchMode',
  '2': [
    {'1': 'HISTORY_SEARCH_MODE_UNSPECIFIED', '2': 0},
    {'1': 'HISTORY_SEARCH_MODE_TEXT', '2': 1},
    {'1': 'HISTORY_SEARCH_MODE_GLOB', '2': 2},
    {'1': 'HISTORY_SEARCH_MODE_REGEX', '2': 3},
  ],
};

/// Descriptor for `HistorySearchMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List historySearchModeDescriptor = $convert.base64Decode(
    'ChFIaXN0b3J5U2VhcmNoTW9kZRIjCh9ISVNUT1JZX1NFQVJDSF9NT0RFX1VOU1BFQ0lGSUVEEA'
    'ASHAoYSElTVE9SWV9TRUFSQ0hfTU9ERV9URVhUEAESHAoYSElTVE9SWV9TRUFSQ0hfTU9ERV9H'
    'TE9CEAISHQoZSElTVE9SWV9TRUFSQ0hfTU9ERV9SRUdFWBAD');

@$core.Deprecated('Use historyCursorSegmentDescriptor instead')
const HistoryCursorSegment$json = {
  '1': 'HistoryCursorSegment',
  '2': [
    {'1': 'HISTORY_CURSOR_SEGMENT_UNSPECIFIED', '2': 0},
    {'1': 'HISTORY_CURSOR_SEGMENT_COMMITTED', '2': 1},
    {'1': 'HISTORY_CURSOR_SEGMENT_CURRENT_PRIMARY_FRAME', '2': 2},
    {'1': 'HISTORY_CURSOR_SEGMENT_ARCHIVED_PRIMARY_FRAME', '2': 3},
    {'1': 'HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME', '2': 4},
  ],
};

/// Descriptor for `HistoryCursorSegment`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List historyCursorSegmentDescriptor = $convert.base64Decode(
    'ChRIaXN0b3J5Q3Vyc29yU2VnbWVudBImCiJISVNUT1JZX0NVUlNPUl9TRUdNRU5UX1VOU1BFQ0'
    'lGSUVEEAASJAogSElTVE9SWV9DVVJTT1JfU0VHTUVOVF9DT01NSVRURUQQARIwCixISVNUT1JZ'
    'X0NVUlNPUl9TRUdNRU5UX0NVUlJFTlRfUFJJTUFSWV9GUkFNRRACEjEKLUhJU1RPUllfQ1VSU0'
    '9SX1NFR01FTlRfQVJDSElWRURfUFJJTUFSWV9GUkFNRRADEiwKKEhJU1RPUllfQ1VSU09SX1NF'
    'R01FTlRfQ1VSUkVOVF9BTFRfRlJBTUUQBA==');

@$core.Deprecated('Use rowOwnershipDescriptor instead')
const RowOwnership$json = {
  '1': 'RowOwnership',
  '2': [
    {'1': 'ROW_OWNERSHIP_UNSPECIFIED', '2': 0},
    {'1': 'ROW_OWNERSHIP_PERSISTED', '2': 1},
    {'1': 'ROW_OWNERSHIP_LIVE_TAIL_RECLAIMED', '2': 2},
    {'1': 'ROW_OWNERSHIP_LIVE_TAIL_LIVE', '2': 3},
    {'1': 'ROW_OWNERSHIP_SCREEN', '2': 4},
  ],
};

/// Descriptor for `RowOwnership`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List rowOwnershipDescriptor = $convert.base64Decode(
    'CgxSb3dPd25lcnNoaXASHQoZUk9XX09XTkVSU0hJUF9VTlNQRUNJRklFRBAAEhsKF1JPV19PV0'
    '5FUlNISVBfUEVSU0lTVEVEEAESJQohUk9XX09XTkVSU0hJUF9MSVZFX1RBSUxfUkVDTEFJTUVE'
    'EAISIAocUk9XX09XTkVSU0hJUF9MSVZFX1RBSUxfTElWRRADEhgKFFJPV19PV05FUlNISVBfU0'
    'NSRUVOEAQ=');

@$core.Deprecated('Use cursorShapeDescriptor instead')
const CursorShape$json = {
  '1': 'CursorShape',
  '2': [
    {'1': 'CURSOR_SHAPE_UNSPECIFIED', '2': 0},
    {'1': 'CURSOR_SHAPE_BLOCK', '2': 1},
    {'1': 'CURSOR_SHAPE_UNDERLINE', '2': 2},
    {'1': 'CURSOR_SHAPE_BAR', '2': 3},
  ],
};

/// Descriptor for `CursorShape`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List cursorShapeDescriptor = $convert.base64Decode(
    'CgtDdXJzb3JTaGFwZRIcChhDVVJTT1JfU0hBUEVfVU5TUEVDSUZJRUQQABIWChJDVVJTT1JfU0'
    'hBUEVfQkxPQ0sQARIaChZDVVJTT1JfU0hBUEVfVU5ERVJMSU5FEAISFAoQQ1VSU09SX1NIQVBF'
    'X0JBUhAD');

@$core.Deprecated('Use cellStyleDescriptor instead')
const CellStyle$json = {
  '1': 'CellStyle',
  '2': [
    {'1': 'foreground', '3': 1, '4': 1, '5': 9, '10': 'foreground'},
    {'1': 'background', '3': 2, '4': 1, '5': 9, '10': 'background'},
    {'1': 'bold', '3': 3, '4': 1, '5': 8, '10': 'bold'},
    {'1': 'italic', '3': 4, '4': 1, '5': 8, '10': 'italic'},
    {'1': 'underline', '3': 5, '4': 1, '5': 8, '10': 'underline'},
    {'1': 'blink', '3': 6, '4': 1, '5': 8, '10': 'blink'},
    {'1': 'reverse', '3': 7, '4': 1, '5': 8, '10': 'reverse'},
    {'1': 'strikethrough', '3': 8, '4': 1, '5': 8, '10': 'strikethrough'},
  ],
};

/// Descriptor for `CellStyle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cellStyleDescriptor = $convert.base64Decode(
    'CglDZWxsU3R5bGUSHgoKZm9yZWdyb3VuZBgBIAEoCVIKZm9yZWdyb3VuZBIeCgpiYWNrZ3JvdW'
    '5kGAIgASgJUgpiYWNrZ3JvdW5kEhIKBGJvbGQYAyABKAhSBGJvbGQSFgoGaXRhbGljGAQgASgI'
    'UgZpdGFsaWMSHAoJdW5kZXJsaW5lGAUgASgIUgl1bmRlcmxpbmUSFAoFYmxpbmsYBiABKAhSBW'
    'JsaW5rEhgKB3JldmVyc2UYByABKAhSB3JldmVyc2USJAoNc3RyaWtldGhyb3VnaBgIIAEoCFIN'
    'c3RyaWtldGhyb3VnaA==');

@$core.Deprecated('Use screenCellDescriptor instead')
const ScreenCell$json = {
  '1': 'ScreenCell',
  '2': [
    {'1': 'content', '3': 1, '4': 1, '5': 9, '10': 'content'},
    {'1': 'width', '3': 2, '4': 1, '5': 5, '10': 'width'},
    {
      '1': 'style',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.CellStyle',
      '10': 'style'
    },
    {'1': 'link_url', '3': 4, '4': 1, '5': 9, '10': 'linkUrl'},
    {'1': 'link_params', '3': 5, '4': 1, '5': 9, '10': 'linkParams'},
  ],
};

/// Descriptor for `ScreenCell`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List screenCellDescriptor = $convert.base64Decode(
    'CgpTY3JlZW5DZWxsEhgKB2NvbnRlbnQYASABKAlSB2NvbnRlbnQSFAoFd2lkdGgYAiABKAVSBX'
    'dpZHRoEi4KBXN0eWxlGAMgASgLMhguYW55dHR5LmFwaS52MS5DZWxsU3R5bGVSBXN0eWxlEhkK'
    'CGxpbmtfdXJsGAQgASgJUgdsaW5rVXJsEh8KC2xpbmtfcGFyYW1zGAUgASgJUgpsaW5rUGFyYW'
    '1z');

@$core.Deprecated('Use screenRowDescriptor instead')
const ScreenRow$json = {
  '1': 'ScreenRow',
  '2': [
    {
      '1': 'cells',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.ScreenCell',
      '10': 'cells'
    },
    {
      '1': 'tail_fill',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.CellStyle',
      '10': 'tailFill'
    },
    {'1': 'wrapped', '3': 3, '4': 1, '5': 8, '10': 'wrapped'},
  ],
};

/// Descriptor for `ScreenRow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List screenRowDescriptor = $convert.base64Decode(
    'CglTY3JlZW5Sb3cSLwoFY2VsbHMYASADKAsyGS5hbnl0dHkuYXBpLnYxLlNjcmVlbkNlbGxSBW'
    'NlbGxzEjUKCXRhaWxfZmlsbBgCIAEoCzIYLmFueXR0eS5hcGkudjEuQ2VsbFN0eWxlUgh0YWls'
    'RmlsbBIYCgd3cmFwcGVkGAMgASgIUgd3cmFwcGVk');

@$core.Deprecated('Use terminalCursorDescriptor instead')
const TerminalCursor$json = {
  '1': 'TerminalCursor',
  '2': [
    {'1': 'row', '3': 1, '4': 1, '5': 5, '10': 'row'},
    {'1': 'col', '3': 2, '4': 1, '5': 5, '10': 'col'},
    {'1': 'visible', '3': 3, '4': 1, '5': 8, '10': 'visible'},
    {
      '1': 'shape',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.CursorShape',
      '10': 'shape'
    },
    {'1': 'blink', '3': 5, '4': 1, '5': 8, '10': 'blink'},
  ],
};

/// Descriptor for `TerminalCursor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalCursorDescriptor = $convert.base64Decode(
    'Cg5UZXJtaW5hbEN1cnNvchIQCgNyb3cYASABKAVSA3JvdxIQCgNjb2wYAiABKAVSA2NvbBIYCg'
    'd2aXNpYmxlGAMgASgIUgd2aXNpYmxlEjAKBXNoYXBlGAQgASgOMhouYW55dHR5LmFwaS52MS5D'
    'dXJzb3JTaGFwZVIFc2hhcGUSFAoFYmxpbmsYBSABKAhSBWJsaW5r');

@$core.Deprecated('Use terminalModesDescriptor instead')
const TerminalModes$json = {
  '1': 'TerminalModes',
  '2': [
    {'1': 'alternate_screen', '3': 1, '4': 1, '5': 8, '10': 'alternateScreen'},
    {'1': 'alternate_scroll', '3': 2, '4': 1, '5': 8, '10': 'alternateScroll'},
    {'1': 'mouse_tracking', '3': 3, '4': 1, '5': 8, '10': 'mouseTracking'},
    {'1': 'mouse_x10', '3': 4, '4': 1, '5': 8, '10': 'mouseX10'},
    {'1': 'mouse_normal', '3': 5, '4': 1, '5': 8, '10': 'mouseNormal'},
    {
      '1': 'mouse_button_event',
      '3': 6,
      '4': 1,
      '5': 8,
      '10': 'mouseButtonEvent'
    },
    {'1': 'mouse_any_event', '3': 7, '4': 1, '5': 8, '10': 'mouseAnyEvent'},
    {'1': 'mouse_sgr', '3': 8, '4': 1, '5': 8, '10': 'mouseSgr'},
    {'1': 'bracketed_paste', '3': 9, '4': 1, '5': 8, '10': 'bracketedPaste'},
    {
      '1': 'application_cursor',
      '3': 10,
      '4': 1,
      '5': 8,
      '10': 'applicationCursor'
    },
    {'1': 'auto_wrap', '3': 11, '4': 1, '5': 8, '10': 'autoWrap'},
  ],
};

/// Descriptor for `TerminalModes`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalModesDescriptor = $convert.base64Decode(
    'Cg1UZXJtaW5hbE1vZGVzEikKEGFsdGVybmF0ZV9zY3JlZW4YASABKAhSD2FsdGVybmF0ZVNjcm'
    'VlbhIpChBhbHRlcm5hdGVfc2Nyb2xsGAIgASgIUg9hbHRlcm5hdGVTY3JvbGwSJQoObW91c2Vf'
    'dHJhY2tpbmcYAyABKAhSDW1vdXNlVHJhY2tpbmcSGwoJbW91c2VfeDEwGAQgASgIUghtb3VzZV'
    'gxMBIhCgxtb3VzZV9ub3JtYWwYBSABKAhSC21vdXNlTm9ybWFsEiwKEm1vdXNlX2J1dHRvbl9l'
    'dmVudBgGIAEoCFIQbW91c2VCdXR0b25FdmVudBImCg9tb3VzZV9hbnlfZXZlbnQYByABKAhSDW'
    '1vdXNlQW55RXZlbnQSGwoJbW91c2Vfc2dyGAggASgIUghtb3VzZVNnchInCg9icmFja2V0ZWRf'
    'cGFzdGUYCSABKAhSDmJyYWNrZXRlZFBhc3RlEi0KEmFwcGxpY2F0aW9uX2N1cnNvchgKIAEoCF'
    'IRYXBwbGljYXRpb25DdXJzb3ISGwoJYXV0b193cmFwGAsgASgIUghhdXRvV3JhcA==');

@$core.Deprecated('Use historyCursorDescriptor instead')
const HistoryCursor$json = {
  '1': 'HistoryCursor',
  '2': [
    {'1': 'line_id', '3': 1, '4': 1, '5': 4, '10': 'lineId'},
    {'1': 'row_in_line', '3': 2, '4': 1, '5': 5, '10': 'rowInLine'},
    {
      '1': 'segment',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.HistoryCursorSegment',
      '10': 'segment'
    },
  ],
  '9': [
    {'1': 3, '2': 4},
  ],
};

/// Descriptor for `HistoryCursor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyCursorDescriptor = $convert.base64Decode(
    'Cg1IaXN0b3J5Q3Vyc29yEhcKB2xpbmVfaWQYASABKARSBmxpbmVJZBIeCgtyb3dfaW5fbGluZR'
    'gCIAEoBVIJcm93SW5MaW5lEj0KB3NlZ21lbnQYBCABKA4yIy5hbnl0dHkuYXBpLnYxLkhpc3Rv'
    'cnlDdXJzb3JTZWdtZW50UgdzZWdtZW50SgQIAxAE');

@$core.Deprecated('Use historyRangeDescriptor instead')
const HistoryRange$json = {
  '1': 'HistoryRange',
  '2': [
    {'1': 'start_line_id', '3': 1, '4': 1, '5': 4, '10': 'startLineId'},
    {'1': 'start_col', '3': 2, '4': 1, '5': 5, '10': 'startCol'},
    {'1': 'end_line_id', '3': 3, '4': 1, '5': 4, '10': 'endLineId'},
    {'1': 'end_col', '3': 4, '4': 1, '5': 5, '10': 'endCol'},
  ],
};

/// Descriptor for `HistoryRange`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyRangeDescriptor = $convert.base64Decode(
    'CgxIaXN0b3J5UmFuZ2USIgoNc3RhcnRfbGluZV9pZBgBIAEoBFILc3RhcnRMaW5lSWQSGwoJc3'
    'RhcnRfY29sGAIgASgFUghzdGFydENvbBIeCgtlbmRfbGluZV9pZBgDIAEoBFIJZW5kTGluZUlk'
    'EhcKB2VuZF9jb2wYBCABKAVSBmVuZENvbA==');

@$core.Deprecated('Use historyWindowCommandDescriptor instead')
const HistoryWindowCommand$json = {
  '1': 'HistoryWindowCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {
      '1': 'mode',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.HistoryWindowMode',
      '10': 'mode'
    },
    {'1': 'limit', '3': 5, '4': 1, '5': 5, '10': 'limit'},
    {'1': 'cols', '3': 6, '4': 1, '5': 5, '10': 'cols'},
    {'1': 'token', '3': 7, '4': 1, '5': 9, '10': 'token'},
    {
      '1': 'history_generation',
      '3': 8,
      '4': 1,
      '5': 4,
      '10': 'historyGeneration'
    },
    {
      '1': 'before_cursor',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryCursor',
      '10': 'beforeCursor'
    },
    {
      '1': 'after_cursor',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryCursor',
      '10': 'afterCursor'
    },
    {
      '1': 'boundary_first_line_id',
      '3': 11,
      '4': 1,
      '5': 4,
      '10': 'boundaryFirstLineId'
    },
    {
      '1': 'boundary_last_line_id',
      '3': 12,
      '4': 1,
      '5': 4,
      '10': 'boundaryLastLineId'
    },
    {
      '1': 'range',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryRange',
      '10': 'range'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
    {'1': 4, '2': 5},
  ],
};

/// Descriptor for `HistoryWindowCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyWindowCommandDescriptor = $convert.base64Decode(
    'ChRIaXN0b3J5V2luZG93Q29tbWFuZBI2Cgh0ZXJtaW5hbBgCIAEoCzIaLmFueXR0eS5hcGkudj'
    'EuVGVybWluYWxSZWZSCHRlcm1pbmFsEjQKBG1vZGUYAyABKA4yIC5hbnl0dHkuYXBpLnYxLkhp'
    'c3RvcnlXaW5kb3dNb2RlUgRtb2RlEhQKBWxpbWl0GAUgASgFUgVsaW1pdBISCgRjb2xzGAYgAS'
    'gFUgRjb2xzEhQKBXRva2VuGAcgASgJUgV0b2tlbhItChJoaXN0b3J5X2dlbmVyYXRpb24YCCAB'
    'KARSEWhpc3RvcnlHZW5lcmF0aW9uEkEKDWJlZm9yZV9jdXJzb3IYCSABKAsyHC5hbnl0dHkuYX'
    'BpLnYxLkhpc3RvcnlDdXJzb3JSDGJlZm9yZUN1cnNvchI/CgxhZnRlcl9jdXJzb3IYCiABKAsy'
    'HC5hbnl0dHkuYXBpLnYxLkhpc3RvcnlDdXJzb3JSC2FmdGVyQ3Vyc29yEjMKFmJvdW5kYXJ5X2'
    'ZpcnN0X2xpbmVfaWQYCyABKARSE2JvdW5kYXJ5Rmlyc3RMaW5lSWQSMQoVYm91bmRhcnlfbGFz'
    'dF9saW5lX2lkGAwgASgEUhJib3VuZGFyeUxhc3RMaW5lSWQSMQoFcmFuZ2UYDSABKAsyGy5hbn'
    'l0dHkuYXBpLnYxLkhpc3RvcnlSYW5nZVIFcmFuZ2VKBAgBEAJKBAgEEAU=');

@$core.Deprecated('Use historyCopyCommandDescriptor instead')
const HistoryCopyCommand$json = {
  '1': 'HistoryCopyCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {
      '1': 'window',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryWindowCommand',
      '10': 'window'
    },
    {'1': 'max_lines', '3': 4, '4': 1, '5': 5, '10': 'maxLines'},
    {'1': 'max_bytes', '3': 5, '4': 1, '5': 5, '10': 'maxBytes'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `HistoryCopyCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyCopyCommandDescriptor = $convert.base64Decode(
    'ChJIaXN0b3J5Q29weUNvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLnYxLl'
    'Rlcm1pbmFsUmVmUgh0ZXJtaW5hbBI7CgZ3aW5kb3cYAyABKAsyIy5hbnl0dHkuYXBpLnYxLkhp'
    'c3RvcnlXaW5kb3dDb21tYW5kUgZ3aW5kb3cSGwoJbWF4X2xpbmVzGAQgASgFUghtYXhMaW5lcx'
    'IbCgltYXhfYnl0ZXMYBSABKAVSCG1heEJ5dGVzSgQIARAC');

@$core.Deprecated('Use historyTextPositionDescriptor instead')
const HistoryTextPosition$json = {
  '1': 'HistoryTextPosition',
  '2': [
    {'1': 'line_id', '3': 1, '4': 1, '5': 4, '10': 'lineId'},
    {'1': 'col', '3': 2, '4': 1, '5': 5, '10': 'col'},
  ],
};

/// Descriptor for `HistoryTextPosition`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyTextPositionDescriptor = $convert.base64Decode(
    'ChNIaXN0b3J5VGV4dFBvc2l0aW9uEhcKB2xpbmVfaWQYASABKARSBmxpbmVJZBIQCgNjb2wYAi'
    'ABKAVSA2NvbA==');

@$core.Deprecated('Use historySearchCommandDescriptor instead')
const HistorySearchCommand$json = {
  '1': 'HistorySearchCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'token', '3': 3, '4': 1, '5': 9, '10': 'token'},
    {
      '1': 'history_generation',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'historyGeneration'
    },
    {'1': 'query', '3': 5, '4': 1, '5': 9, '10': 'query'},
    {
      '1': 'direction',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.HistorySearchDirection',
      '10': 'direction'
    },
    {'1': 'cols', '3': 7, '4': 1, '5': 5, '10': 'cols'},
    {'1': 'limit', '3': 8, '4': 1, '5': 5, '10': 'limit'},
    {
      '1': 'start',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryTextPosition',
      '10': 'start'
    },
    {
      '1': 'mode',
      '3': 10,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.HistorySearchMode',
      '10': 'mode'
    },
    {'1': 'context_before', '3': 11, '4': 1, '5': 5, '10': 'contextBefore'},
    {'1': 'scan', '3': 12, '4': 1, '5': 8, '10': 'scan'},
    {'1': 'max_matches', '3': 13, '4': 1, '5': 5, '10': 'maxMatches'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `HistorySearchCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historySearchCommandDescriptor = $convert.base64Decode(
    'ChRIaXN0b3J5U2VhcmNoQ29tbWFuZBI2Cgh0ZXJtaW5hbBgCIAEoCzIaLmFueXR0eS5hcGkudj'
    'EuVGVybWluYWxSZWZSCHRlcm1pbmFsEhQKBXRva2VuGAMgASgJUgV0b2tlbhItChJoaXN0b3J5'
    'X2dlbmVyYXRpb24YBCABKARSEWhpc3RvcnlHZW5lcmF0aW9uEhQKBXF1ZXJ5GAUgASgJUgVxdW'
    'VyeRJDCglkaXJlY3Rpb24YBiABKA4yJS5hbnl0dHkuYXBpLnYxLkhpc3RvcnlTZWFyY2hEaXJl'
    'Y3Rpb25SCWRpcmVjdGlvbhISCgRjb2xzGAcgASgFUgRjb2xzEhQKBWxpbWl0GAggASgFUgVsaW'
    '1pdBI4CgVzdGFydBgJIAEoCzIiLmFueXR0eS5hcGkudjEuSGlzdG9yeVRleHRQb3NpdGlvblIF'
    'c3RhcnQSNAoEbW9kZRgKIAEoDjIgLmFueXR0eS5hcGkudjEuSGlzdG9yeVNlYXJjaE1vZGVSBG'
    '1vZGUSJQoOY29udGV4dF9iZWZvcmUYCyABKAVSDWNvbnRleHRCZWZvcmUSEgoEc2NhbhgMIAEo'
    'CFIEc2NhbhIfCgttYXhfbWF0Y2hlcxgNIAEoBVIKbWF4TWF0Y2hlc0oECAEQAg==');

@$core.Deprecated('Use historyReleaseCommandDescriptor instead')
const HistoryReleaseCommand$json = {
  '1': 'HistoryReleaseCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'token', '3': 3, '4': 1, '5': 9, '10': 'token'},
    {
      '1': 'history_generation',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'historyGeneration'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `HistoryReleaseCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyReleaseCommandDescriptor = $convert.base64Decode(
    'ChVIaXN0b3J5UmVsZWFzZUNvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLn'
    'YxLlRlcm1pbmFsUmVmUgh0ZXJtaW5hbBIUCgV0b2tlbhgDIAEoCVIFdG9rZW4SLQoSaGlzdG9y'
    'eV9nZW5lcmF0aW9uGAQgASgEUhFoaXN0b3J5R2VuZXJhdGlvbkoECAEQAg==');

@$core.Deprecated('Use historyBacklogStatusCommandDescriptor instead')
const HistoryBacklogStatusCommand$json = {
  '1': 'HistoryBacklogStatusCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `HistoryBacklogStatusCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyBacklogStatusCommandDescriptor =
    $convert.base64Decode(
        'ChtIaXN0b3J5QmFja2xvZ1N0YXR1c0NvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dH'
        'kuYXBpLnYxLlRlcm1pbmFsUmVmUgh0ZXJtaW5hbEoECAEQAg==');

@$core.Deprecated('Use historyLineSpanDescriptor instead')
const HistoryLineSpan$json = {
  '1': 'HistoryLineSpan',
  '2': [
    {'1': 'start_row', '3': 1, '4': 1, '5': 5, '10': 'startRow'},
    {'1': 'end_row', '3': 2, '4': 1, '5': 5, '10': 'endRow'},
    {'1': 'row_kind', '3': 3, '4': 1, '5': 9, '10': 'rowKind'},
    {'1': 'logical_line_id', '3': 4, '4': 1, '5': 4, '10': 'logicalLineId'},
    {'1': 'session_id', '3': 5, '4': 1, '5': 4, '10': 'sessionId'},
    {'1': 'frame_id', '3': 6, '4': 1, '5': 4, '10': 'frameId'},
    {'1': 'fixed_grid', '3': 7, '4': 1, '5': 8, '10': 'fixedGrid'},
    {'1': 'screen_cols', '3': 8, '4': 1, '5': 5, '10': 'screenCols'},
    {
      '1': 'timestamp_start_unix_nano',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'timestampStartUnixNano'
    },
    {
      '1': 'timestamp_end_unix_nano',
      '3': 10,
      '4': 1,
      '5': 3,
      '10': 'timestampEndUnixNano'
    },
    {'1': 'clipped_before', '3': 11, '4': 1, '5': 8, '10': 'clippedBefore'},
    {'1': 'clipped_after', '3': 12, '4': 1, '5': 8, '10': 'clippedAfter'},
  ],
};

/// Descriptor for `HistoryLineSpan`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyLineSpanDescriptor = $convert.base64Decode(
    'Cg9IaXN0b3J5TGluZVNwYW4SGwoJc3RhcnRfcm93GAEgASgFUghzdGFydFJvdxIXCgdlbmRfcm'
    '93GAIgASgFUgZlbmRSb3cSGQoIcm93X2tpbmQYAyABKAlSB3Jvd0tpbmQSJgoPbG9naWNhbF9s'
    'aW5lX2lkGAQgASgEUg1sb2dpY2FsTGluZUlkEh0KCnNlc3Npb25faWQYBSABKARSCXNlc3Npb2'
    '5JZBIZCghmcmFtZV9pZBgGIAEoBFIHZnJhbWVJZBIdCgpmaXhlZF9ncmlkGAcgASgIUglmaXhl'
    'ZEdyaWQSHwoLc2NyZWVuX2NvbHMYCCABKAVSCnNjcmVlbkNvbHMSOQoZdGltZXN0YW1wX3N0YX'
    'J0X3VuaXhfbmFubxgJIAEoA1IWdGltZXN0YW1wU3RhcnRVbml4TmFubxI1Chd0aW1lc3RhbXBf'
    'ZW5kX3VuaXhfbmFubxgKIAEoA1IUdGltZXN0YW1wRW5kVW5peE5hbm8SJQoOY2xpcHBlZF9iZW'
    'ZvcmUYCyABKAhSDWNsaXBwZWRCZWZvcmUSIwoNY2xpcHBlZF9hZnRlchgMIAEoCFIMY2xpcHBl'
    'ZEFmdGVy');

@$core.Deprecated('Use historyRowDescriptor instead')
const HistoryRow$json = {
  '1': 'HistoryRow',
  '2': [
    {
      '1': 'row',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ScreenRow',
      '10': 'row'
    },
    {
      '1': 'timestamp_unix_nano',
      '3': 2,
      '4': 1,
      '5': 3,
      '10': 'timestampUnixNano'
    },
    {'1': 'row_kind', '3': 3, '4': 1, '5': 9, '10': 'rowKind'},
    {'1': 'wrapped', '3': 4, '4': 1, '5': 8, '10': 'wrapped'},
    {
      '1': 'ownership',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.RowOwnership',
      '10': 'ownership'
    },
    {
      '1': 'segment',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.HistoryCursorSegment',
      '10': 'segment'
    },
    {'1': 'session_id', '3': 7, '4': 1, '5': 4, '10': 'sessionId'},
    {'1': 'frame_id', '3': 8, '4': 1, '5': 4, '10': 'frameId'},
    {'1': 'fixed_grid', '3': 9, '4': 1, '5': 8, '10': 'fixedGrid'},
    {'1': 'screen_cols', '3': 10, '4': 1, '5': 5, '10': 'screenCols'},
    {'1': 'screen_rows', '3': 11, '4': 1, '5': 5, '10': 'screenRows'},
    {'1': 'screen_row_set', '3': 12, '4': 1, '5': 8, '10': 'screenRowSet'},
    {'1': 'logical_line_id', '3': 14, '4': 1, '5': 4, '10': 'logicalLineId'},
    {'1': 'row_in_line', '3': 15, '4': 1, '5': 5, '10': 'rowInLine'},
  ],
  '9': [
    {'1': 13, '2': 14},
  ],
};

/// Descriptor for `HistoryRow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyRowDescriptor = $convert.base64Decode(
    'CgpIaXN0b3J5Um93EioKA3JvdxgBIAEoCzIYLmFueXR0eS5hcGkudjEuU2NyZWVuUm93UgNyb3'
    'cSLgoTdGltZXN0YW1wX3VuaXhfbmFubxgCIAEoA1IRdGltZXN0YW1wVW5peE5hbm8SGQoIcm93'
    'X2tpbmQYAyABKAlSB3Jvd0tpbmQSGAoHd3JhcHBlZBgEIAEoCFIHd3JhcHBlZBI5Cglvd25lcn'
    'NoaXAYBSABKA4yGy5hbnl0dHkuYXBpLnYxLlJvd093bmVyc2hpcFIJb3duZXJzaGlwEj0KB3Nl'
    'Z21lbnQYBiABKA4yIy5hbnl0dHkuYXBpLnYxLkhpc3RvcnlDdXJzb3JTZWdtZW50UgdzZWdtZW'
    '50Eh0KCnNlc3Npb25faWQYByABKARSCXNlc3Npb25JZBIZCghmcmFtZV9pZBgIIAEoBFIHZnJh'
    'bWVJZBIdCgpmaXhlZF9ncmlkGAkgASgIUglmaXhlZEdyaWQSHwoLc2NyZWVuX2NvbHMYCiABKA'
    'VSCnNjcmVlbkNvbHMSHwoLc2NyZWVuX3Jvd3MYCyABKAVSCnNjcmVlblJvd3MSJAoOc2NyZWVu'
    'X3Jvd19zZXQYDCABKAhSDHNjcmVlblJvd1NldBImCg9sb2dpY2FsX2xpbmVfaWQYDiABKARSDW'
    'xvZ2ljYWxMaW5lSWQSHgoLcm93X2luX2xpbmUYDyABKAVSCXJvd0luTGluZUoECA0QDg==');

@$core.Deprecated('Use historyViewportAnchorDescriptor instead')
const HistoryViewportAnchor$json = {
  '1': 'HistoryViewportAnchor',
  '2': [
    {'1': 'top_line_id', '3': 1, '4': 1, '5': 4, '10': 'topLineId'},
    {'1': 'top_cell_offset', '3': 2, '4': 1, '5': 5, '10': 'topCellOffset'},
    {'1': 'at_end', '3': 3, '4': 1, '5': 8, '10': 'atEnd'},
    {'1': 'screen_cols', '3': 4, '4': 1, '5': 13, '10': 'screenCols'},
    {'1': 'screen_rows', '3': 5, '4': 1, '5': 13, '10': 'screenRows'},
  ],
};

/// Descriptor for `HistoryViewportAnchor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyViewportAnchorDescriptor = $convert.base64Decode(
    'ChVIaXN0b3J5Vmlld3BvcnRBbmNob3ISHgoLdG9wX2xpbmVfaWQYASABKARSCXRvcExpbmVJZB'
    'ImCg90b3BfY2VsbF9vZmZzZXQYAiABKAVSDXRvcENlbGxPZmZzZXQSFQoGYXRfZW5kGAMgASgI'
    'UgVhdEVuZBIfCgtzY3JlZW5fY29scxgEIAEoDVIKc2NyZWVuQ29scxIfCgtzY3JlZW5fcm93cx'
    'gFIAEoDVIKc2NyZWVuUm93cw==');

@$core.Deprecated('Use historyWindowResultDescriptor instead')
const HistoryWindowResult$json = {
  '1': 'HistoryWindowResult',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'token', '3': 2, '4': 1, '5': 9, '10': 'token'},
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.HistoryWindowOperation',
      '10': 'operation'
    },
    {
      '1': 'size',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSize',
      '10': 'size'
    },
    {
      '1': 'rows',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.HistoryRow',
      '10': 'rows'
    },
    {
      '1': 'lines',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.HistoryLineSpan',
      '10': 'lines'
    },
    {'1': 'loaded_rows', '3': 8, '4': 1, '5': 5, '10': 'loadedRows'},
    {'1': 'total_rows', '3': 9, '4': 1, '5': 5, '10': 'totalRows'},
    {'1': 'loaded_lines', '3': 10, '4': 1, '5': 5, '10': 'loadedLines'},
    {'1': 'logical_total', '3': 11, '4': 1, '5': 5, '10': 'logicalTotal'},
    {'1': 'has_more', '3': 12, '4': 1, '5': 8, '10': 'hasMore'},
    {
      '1': 'history_generation',
      '3': 13,
      '4': 1,
      '5': 4,
      '10': 'historyGeneration'
    },
    {'1': 'first_row_id', '3': 14, '4': 1, '5': 4, '10': 'firstRowId'},
    {'1': 'last_row_id', '3': 15, '4': 1, '5': 4, '10': 'lastRowId'},
    {'1': 'first_line_id', '3': 16, '4': 1, '5': 4, '10': 'firstLineId'},
    {'1': 'last_line_id', '3': 17, '4': 1, '5': 4, '10': 'lastLineId'},
    {
      '1': 'cursor',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryCursor',
      '10': 'cursor'
    },
    {
      '1': 'timestamp_unix_nano',
      '3': 19,
      '4': 1,
      '5': 3,
      '10': 'timestampUnixNano'
    },
    {
      '1': 'viewport_anchor',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryViewportAnchor',
      '10': 'viewportAnchor'
    },
  ],
  '9': [
    {'1': 7, '2': 8},
  ],
};

/// Descriptor for `HistoryWindowResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyWindowResultDescriptor = $convert.base64Decode(
    'ChNIaXN0b3J5V2luZG93UmVzdWx0EjYKCHRlcm1pbmFsGAEgASgLMhouYW55dHR5LmFwaS52MS'
    '5UZXJtaW5hbFJlZlIIdGVybWluYWwSFAoFdG9rZW4YAiABKAlSBXRva2VuEkMKCW9wZXJhdGlv'
    'bhgDIAEoDjIlLmFueXR0eS5hcGkudjEuSGlzdG9yeVdpbmRvd09wZXJhdGlvblIJb3BlcmF0aW'
    '9uEi8KBHNpemUYBCABKAsyGy5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsU2l6ZVIEc2l6ZRItCgRy'
    'b3dzGAUgAygLMhkuYW55dHR5LmFwaS52MS5IaXN0b3J5Um93UgRyb3dzEjQKBWxpbmVzGAYgAy'
    'gLMh4uYW55dHR5LmFwaS52MS5IaXN0b3J5TGluZVNwYW5SBWxpbmVzEh8KC2xvYWRlZF9yb3dz'
    'GAggASgFUgpsb2FkZWRSb3dzEh0KCnRvdGFsX3Jvd3MYCSABKAVSCXRvdGFsUm93cxIhCgxsb2'
    'FkZWRfbGluZXMYCiABKAVSC2xvYWRlZExpbmVzEiMKDWxvZ2ljYWxfdG90YWwYCyABKAVSDGxv'
    'Z2ljYWxUb3RhbBIZCghoYXNfbW9yZRgMIAEoCFIHaGFzTW9yZRItChJoaXN0b3J5X2dlbmVyYX'
    'Rpb24YDSABKARSEWhpc3RvcnlHZW5lcmF0aW9uEiAKDGZpcnN0X3Jvd19pZBgOIAEoBFIKZmly'
    'c3RSb3dJZBIeCgtsYXN0X3Jvd19pZBgPIAEoBFIJbGFzdFJvd0lkEiIKDWZpcnN0X2xpbmVfaW'
    'QYECABKARSC2ZpcnN0TGluZUlkEiAKDGxhc3RfbGluZV9pZBgRIAEoBFIKbGFzdExpbmVJZBI0'
    'CgZjdXJzb3IYEiABKAsyHC5hbnl0dHkuYXBpLnYxLkhpc3RvcnlDdXJzb3JSBmN1cnNvchIuCh'
    'N0aW1lc3RhbXBfdW5peF9uYW5vGBMgASgDUhF0aW1lc3RhbXBVbml4TmFubxJNCg92aWV3cG9y'
    'dF9hbmNob3IYFCABKAsyJC5hbnl0dHkuYXBpLnYxLkhpc3RvcnlWaWV3cG9ydEFuY2hvclIOdm'
    'lld3BvcnRBbmNob3JKBAgHEAg=');

@$core.Deprecated('Use historyCopyResultDescriptor instead')
const HistoryCopyResult$json = {
  '1': 'HistoryCopyResult',
  '2': [
    {'1': 'text', '3': 1, '4': 1, '5': 9, '10': 'text'},
    {
      '1': 'next',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryTextPosition',
      '10': 'next'
    },
    {'1': 'done', '3': 3, '4': 1, '5': 8, '10': 'done'},
  ],
};

/// Descriptor for `HistoryCopyResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyCopyResultDescriptor = $convert.base64Decode(
    'ChFIaXN0b3J5Q29weVJlc3VsdBISCgR0ZXh0GAEgASgJUgR0ZXh0EjYKBG5leHQYAiABKAsyIi'
    '5hbnl0dHkuYXBpLnYxLkhpc3RvcnlUZXh0UG9zaXRpb25SBG5leHQSEgoEZG9uZRgDIAEoCFIE'
    'ZG9uZQ==');

@$core.Deprecated('Use historySearchResultDescriptor instead')
const HistorySearchResult$json = {
  '1': 'HistorySearchResult',
  '2': [
    {'1': 'found', '3': 1, '4': 1, '5': 8, '10': 'found'},
    {
      '1': 'match',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryRange',
      '10': 'match'
    },
    {
      '1': 'window',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryWindowResult',
      '10': 'window'
    },
    {'1': 'wrapped', '3': 4, '4': 1, '5': 8, '10': 'wrapped'},
    {
      '1': 'scan_matches',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.HistoryRange',
      '10': 'scanMatches'
    },
    {
      '1': 'scan_next',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryTextPosition',
      '10': 'scanNext'
    },
    {'1': 'scan_done', '3': 7, '4': 1, '5': 8, '10': 'scanDone'},
  ],
};

/// Descriptor for `HistorySearchResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historySearchResultDescriptor = $convert.base64Decode(
    'ChNIaXN0b3J5U2VhcmNoUmVzdWx0EhQKBWZvdW5kGAEgASgIUgVmb3VuZBIxCgVtYXRjaBgCIA'
    'EoCzIbLmFueXR0eS5hcGkudjEuSGlzdG9yeVJhbmdlUgVtYXRjaBI6CgZ3aW5kb3cYAyABKAsy'
    'Ii5hbnl0dHkuYXBpLnYxLkhpc3RvcnlXaW5kb3dSZXN1bHRSBndpbmRvdxIYCgd3cmFwcGVkGA'
    'QgASgIUgd3cmFwcGVkEj4KDHNjYW5fbWF0Y2hlcxgFIAMoCzIbLmFueXR0eS5hcGkudjEuSGlz'
    'dG9yeVJhbmdlUgtzY2FuTWF0Y2hlcxI/CglzY2FuX25leHQYBiABKAsyIi5hbnl0dHkuYXBpLn'
    'YxLkhpc3RvcnlUZXh0UG9zaXRpb25SCHNjYW5OZXh0EhsKCXNjYW5fZG9uZRgHIAEoCFIIc2Nh'
    'bkRvbmU=');

@$core.Deprecated('Use historyBacklogStatusResultDescriptor instead')
const HistoryBacklogStatusResult$json = {
  '1': 'HistoryBacklogStatusResult',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'history_enabled', '3': 2, '4': 1, '5': 8, '10': 'historyEnabled'},
    {
      '1': 'output_buffer_policy',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'outputBufferPolicy'
    },
    {
      '1': 'buffer_capacity_bytes',
      '3': 4,
      '4': 1,
      '5': 3,
      '10': 'bufferCapacityBytes'
    },
    {'1': 'resident_bytes', '3': 5, '4': 1, '5': 3, '10': 'residentBytes'},
    {
      '1': 'aggregate_resident_bytes',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'aggregateResidentBytes'
    },
    {
      '1': 'aggregate_budget_bytes',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'aggregateBudgetBytes'
    },
    {'1': 'dropped_bytes', '3': 8, '4': 1, '5': 4, '10': 'droppedBytes'},
    {'1': 'gap_count', '3': 9, '4': 1, '5': 4, '10': 'gapCount'},
    {
      '1': 'output_buffer_wait_nanos',
      '3': 10,
      '4': 1,
      '5': 3,
      '10': 'outputBufferWaitNanos'
    },
    {'1': 'unavailable', '3': 11, '4': 1, '5': 8, '10': 'unavailable'},
    {
      '1': 'unavailable_reason',
      '3': 12,
      '4': 1,
      '5': 9,
      '10': 'unavailableReason'
    },
    {'1': 'closed', '3': 13, '4': 1, '5': 8, '10': 'closed'},
  ],
};

/// Descriptor for `HistoryBacklogStatusResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List historyBacklogStatusResultDescriptor = $convert.base64Decode(
    'ChpIaXN0b3J5QmFja2xvZ1N0YXR1c1Jlc3VsdBI2Cgh0ZXJtaW5hbBgBIAEoCzIaLmFueXR0eS'
    '5hcGkudjEuVGVybWluYWxSZWZSCHRlcm1pbmFsEicKD2hpc3RvcnlfZW5hYmxlZBgCIAEoCFIO'
    'aGlzdG9yeUVuYWJsZWQSMAoUb3V0cHV0X2J1ZmZlcl9wb2xpY3kYAyABKAlSEm91dHB1dEJ1Zm'
    'ZlclBvbGljeRIyChVidWZmZXJfY2FwYWNpdHlfYnl0ZXMYBCABKANSE2J1ZmZlckNhcGFjaXR5'
    'Qnl0ZXMSJQoOcmVzaWRlbnRfYnl0ZXMYBSABKANSDXJlc2lkZW50Qnl0ZXMSOAoYYWdncmVnYX'
    'RlX3Jlc2lkZW50X2J5dGVzGAYgASgDUhZhZ2dyZWdhdGVSZXNpZGVudEJ5dGVzEjQKFmFnZ3Jl'
    'Z2F0ZV9idWRnZXRfYnl0ZXMYByABKANSFGFnZ3JlZ2F0ZUJ1ZGdldEJ5dGVzEiMKDWRyb3BwZW'
    'RfYnl0ZXMYCCABKARSDGRyb3BwZWRCeXRlcxIbCglnYXBfY291bnQYCSABKARSCGdhcENvdW50'
    'EjcKGG91dHB1dF9idWZmZXJfd2FpdF9uYW5vcxgKIAEoA1IVb3V0cHV0QnVmZmVyV2FpdE5hbm'
    '9zEiAKC3VuYXZhaWxhYmxlGAsgASgIUgt1bmF2YWlsYWJsZRItChJ1bmF2YWlsYWJsZV9yZWFz'
    'b24YDCABKAlSEXVuYXZhaWxhYmxlUmVhc29uEhYKBmNsb3NlZBgNIAEoCFIGY2xvc2Vk');

@$core.Deprecated('Use liveScreenNextCommandDescriptor instead')
const LiveScreenNextCommand$json = {
  '1': 'LiveScreenNextCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {
      '1': 'observed_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'observedRevision'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `LiveScreenNextCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List liveScreenNextCommandDescriptor = $convert.base64Decode(
    'ChVMaXZlU2NyZWVuTmV4dENvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLn'
    'YxLlRlcm1pbmFsUmVmUgh0ZXJtaW5hbBIrChFvYnNlcnZlZF9yZXZpc2lvbhgDIAEoBFIQb2Jz'
    'ZXJ2ZWRSZXZpc2lvbkoECAEQAg==');

@$core.Deprecated('Use screenRowCopyDescriptor instead')
const ScreenRowCopy$json = {
  '1': 'ScreenRowCopy',
  '2': [
    {'1': 'source_row', '3': 1, '4': 1, '5': 5, '10': 'sourceRow'},
    {'1': 'destination_row', '3': 2, '4': 1, '5': 5, '10': 'destinationRow'},
    {'1': 'count', '3': 3, '4': 1, '5': 5, '10': 'count'},
  ],
};

/// Descriptor for `ScreenRowCopy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List screenRowCopyDescriptor = $convert.base64Decode(
    'Cg1TY3JlZW5Sb3dDb3B5Eh0KCnNvdXJjZV9yb3cYASABKAVSCXNvdXJjZVJvdxInCg9kZXN0aW'
    '5hdGlvbl9yb3cYAiABKAVSDmRlc3RpbmF0aW9uUm93EhQKBWNvdW50GAMgASgFUgVjb3VudA==');

@$core.Deprecated('Use screenRowReplaceDescriptor instead')
const ScreenRowReplace$json = {
  '1': 'ScreenRowReplace',
  '2': [
    {'1': 'row_index', '3': 1, '4': 1, '5': 5, '10': 'rowIndex'},
    {
      '1': 'row',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ScreenRow',
      '10': 'row'
    },
  ],
};

/// Descriptor for `ScreenRowReplace`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List screenRowReplaceDescriptor = $convert.base64Decode(
    'ChBTY3JlZW5Sb3dSZXBsYWNlEhsKCXJvd19pbmRleBgBIAEoBVIIcm93SW5kZXgSKgoDcm93GA'
    'IgASgLMhguYW55dHR5LmFwaS52MS5TY3JlZW5Sb3dSA3Jvdw==');

@$core.Deprecated('Use nativeScreenResultDescriptor instead')
const NativeScreenResult$json = {
  '1': 'NativeScreenResult',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'live_revision', '3': 2, '4': 1, '5': 4, '10': 'liveRevision'},
    {
      '1': 'size',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSize',
      '10': 'size'
    },
    {
      '1': 'row_replacements',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.ScreenRowReplace',
      '10': 'rowReplacements'
    },
    {'1': 'alternate_screen', '3': 5, '4': 1, '5': 8, '10': 'alternateScreen'},
    {
      '1': 'cursor',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalCursor',
      '10': 'cursor'
    },
    {
      '1': 'modes',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalModes',
      '10': 'modes'
    },
    {
      '1': 'timestamp_unix_nano',
      '3': 8,
      '4': 1,
      '5': 3,
      '10': 'timestampUnixNano'
    },
    {'1': 'base_revision', '3': 9, '4': 1, '5': 4, '10': 'baseRevision'},
    {
      '1': 'row_copies',
      '3': 10,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.ScreenRowCopy',
      '10': 'rowCopies'
    },
    {'1': 'full_replace', '3': 11, '4': 1, '5': 8, '10': 'fullReplace'},
  ],
};

/// Descriptor for `NativeScreenResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List nativeScreenResultDescriptor = $convert.base64Decode(
    'ChJOYXRpdmVTY3JlZW5SZXN1bHQSNgoIdGVybWluYWwYASABKAsyGi5hbnl0dHkuYXBpLnYxLl'
    'Rlcm1pbmFsUmVmUgh0ZXJtaW5hbBIjCg1saXZlX3JldmlzaW9uGAIgASgEUgxsaXZlUmV2aXNp'
    'b24SLwoEc2l6ZRgDIAEoCzIbLmFueXR0eS5hcGkudjEuVGVybWluYWxTaXplUgRzaXplEkoKEH'
    'Jvd19yZXBsYWNlbWVudHMYBCADKAsyHy5hbnl0dHkuYXBpLnYxLlNjcmVlblJvd1JlcGxhY2VS'
    'D3Jvd1JlcGxhY2VtZW50cxIpChBhbHRlcm5hdGVfc2NyZWVuGAUgASgIUg9hbHRlcm5hdGVTY3'
    'JlZW4SNQoGY3Vyc29yGAYgASgLMh0uYW55dHR5LmFwaS52MS5UZXJtaW5hbEN1cnNvclIGY3Vy'
    'c29yEjIKBW1vZGVzGAcgASgLMhwuYW55dHR5LmFwaS52MS5UZXJtaW5hbE1vZGVzUgVtb2Rlcx'
    'IuChN0aW1lc3RhbXBfdW5peF9uYW5vGAggASgDUhF0aW1lc3RhbXBVbml4TmFubxIjCg1iYXNl'
    'X3JldmlzaW9uGAkgASgEUgxiYXNlUmV2aXNpb24SOwoKcm93X2NvcGllcxgKIAMoCzIcLmFueX'
    'R0eS5hcGkudjEuU2NyZWVuUm93Q29weVIJcm93Q29waWVzEiEKDGZ1bGxfcmVwbGFjZRgLIAEo'
    'CFILZnVsbFJlcGxhY2U=');
