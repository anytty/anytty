// This is a generated file - do not edit.
//
// Generated from apipb/workbench.proto.

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

@$core.Deprecated('Use workbenchSplitDirectionDescriptor instead')
const WorkbenchSplitDirection$json = {
  '1': 'WorkbenchSplitDirection',
  '2': [
    {'1': 'WORKBENCH_SPLIT_DIRECTION_UNSPECIFIED', '2': 0},
    {'1': 'WORKBENCH_SPLIT_DIRECTION_HORIZONTAL', '2': 1},
    {'1': 'WORKBENCH_SPLIT_DIRECTION_VERTICAL', '2': 2},
  ],
};

/// Descriptor for `WorkbenchSplitDirection`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List workbenchSplitDirectionDescriptor = $convert.base64Decode(
    'ChdXb3JrYmVuY2hTcGxpdERpcmVjdGlvbhIpCiVXT1JLQkVOQ0hfU1BMSVRfRElSRUNUSU9OX1'
    'VOU1BFQ0lGSUVEEAASKAokV09SS0JFTkNIX1NQTElUX0RJUkVDVElPTl9IT1JJWk9OVEFMEAES'
    'JgoiV09SS0JFTkNIX1NQTElUX0RJUkVDVElPTl9WRVJUSUNBTBAC');

@$core.Deprecated('Use workbenchValueDescriptor instead')
const WorkbenchValue$json = {
  '1': 'WorkbenchValue',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {
      '1': 'active_workspace_id',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'activeWorkspaceId'
    },
    {
      '1': 'workspaces',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.WorkbenchWorkspace',
      '10': 'workspaces'
    },
  ],
};

/// Descriptor for `WorkbenchValue`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workbenchValueDescriptor = $convert.base64Decode(
    'Cg5Xb3JrYmVuY2hWYWx1ZRIlCg5zY2hlbWFfdmVyc2lvbhgBIAEoDVINc2NoZW1hVmVyc2lvbh'
    'IuChNhY3RpdmVfd29ya3NwYWNlX2lkGAIgASgJUhFhY3RpdmVXb3Jrc3BhY2VJZBJBCgp3b3Jr'
    'c3BhY2VzGAMgAygLMiEuYW55dHR5LmFwaS52MS5Xb3JrYmVuY2hXb3Jrc3BhY2VSCndvcmtzcG'
    'FjZXM=');

@$core.Deprecated('Use workbenchWorkspaceDescriptor instead')
const WorkbenchWorkspace$json = {
  '1': 'WorkbenchWorkspace',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'active_tab_id', '3': 3, '4': 1, '5': 9, '10': 'activeTabId'},
    {
      '1': 'tabs',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.WorkbenchTab',
      '10': 'tabs'
    },
  ],
};

/// Descriptor for `WorkbenchWorkspace`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workbenchWorkspaceDescriptor = $convert.base64Decode(
    'ChJXb3JrYmVuY2hXb3Jrc3BhY2USDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbW'
    'USIgoNYWN0aXZlX3RhYl9pZBgDIAEoCVILYWN0aXZlVGFiSWQSLwoEdGFicxgEIAMoCzIbLmFu'
    'eXR0eS5hcGkudjEuV29ya2JlbmNoVGFiUgR0YWJz');

@$core.Deprecated('Use workbenchTabDescriptor instead')
const WorkbenchTab$json = {
  '1': 'WorkbenchTab',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'title', '3': 2, '4': 1, '5': 9, '10': 'title'},
    {'1': 'active_pane_id', '3': 3, '4': 1, '5': 9, '10': 'activePaneId'},
    {
      '1': 'panes',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.WorkbenchPane',
      '10': 'panes'
    },
    {
      '1': 'root_split',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.WorkbenchSplitNode',
      '10': 'rootSplit'
    },
  ],
};

/// Descriptor for `WorkbenchTab`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workbenchTabDescriptor = $convert.base64Decode(
    'CgxXb3JrYmVuY2hUYWISDgoCaWQYASABKAlSAmlkEhQKBXRpdGxlGAIgASgJUgV0aXRsZRIkCg'
    '5hY3RpdmVfcGFuZV9pZBgDIAEoCVIMYWN0aXZlUGFuZUlkEjIKBXBhbmVzGAQgAygLMhwuYW55'
    'dHR5LmFwaS52MS5Xb3JrYmVuY2hQYW5lUgVwYW5lcxJACgpyb290X3NwbGl0GAUgASgLMiEuYW'
    '55dHR5LmFwaS52MS5Xb3JrYmVuY2hTcGxpdE5vZGVSCXJvb3RTcGxpdA==');

@$core.Deprecated('Use workbenchPaneDescriptor instead')
const WorkbenchPane$json = {
  '1': 'WorkbenchPane',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'title', '3': 2, '4': 1, '5': 9, '10': 'title'},
    {'1': 'terminal_id', '3': 3, '4': 1, '5': 9, '10': 'terminalId'},
  ],
};

/// Descriptor for `WorkbenchPane`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workbenchPaneDescriptor = $convert.base64Decode(
    'Cg1Xb3JrYmVuY2hQYW5lEg4KAmlkGAEgASgJUgJpZBIUCgV0aXRsZRgCIAEoCVIFdGl0bGUSHw'
    'oLdGVybWluYWxfaWQYAyABKAlSCnRlcm1pbmFsSWQ=');

@$core.Deprecated('Use workbenchSplitNodeDescriptor instead')
const WorkbenchSplitNode$json = {
  '1': 'WorkbenchSplitNode',
  '2': [
    {'1': 'pane_id', '3': 1, '4': 1, '5': 9, '10': 'paneId'},
    {
      '1': 'direction',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.WorkbenchSplitDirection',
      '10': 'direction'
    },
    {
      '1': 'children',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.WorkbenchSplitNode',
      '10': 'children'
    },
    {'1': 'ratio', '3': 4, '4': 1, '5': 1, '10': 'ratio'},
    {'1': 'bias_cells', '3': 5, '4': 1, '5': 5, '10': 'biasCells'},
    {'1': 'fixed_pane_id', '3': 6, '4': 1, '5': 9, '10': 'fixedPaneId'},
    {'1': 'fixed_cols', '3': 7, '4': 1, '5': 5, '10': 'fixedCols'},
    {'1': 'fixed_rows', '3': 8, '4': 1, '5': 5, '10': 'fixedRows'},
  ],
};

/// Descriptor for `WorkbenchSplitNode`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workbenchSplitNodeDescriptor = $convert.base64Decode(
    'ChJXb3JrYmVuY2hTcGxpdE5vZGUSFwoHcGFuZV9pZBgBIAEoCVIGcGFuZUlkEkQKCWRpcmVjdG'
    'lvbhgCIAEoDjImLmFueXR0eS5hcGkudjEuV29ya2JlbmNoU3BsaXREaXJlY3Rpb25SCWRpcmVj'
    'dGlvbhI9CghjaGlsZHJlbhgDIAMoCzIhLmFueXR0eS5hcGkudjEuV29ya2JlbmNoU3BsaXROb2'
    'RlUghjaGlsZHJlbhIUCgVyYXRpbxgEIAEoAVIFcmF0aW8SHQoKYmlhc19jZWxscxgFIAEoBVIJ'
    'Ymlhc0NlbGxzEiIKDWZpeGVkX3BhbmVfaWQYBiABKAlSC2ZpeGVkUGFuZUlkEh0KCmZpeGVkX2'
    'NvbHMYByABKAVSCWZpeGVkQ29scxIdCgpmaXhlZF9yb3dzGAggASgFUglmaXhlZFJvd3M=');
