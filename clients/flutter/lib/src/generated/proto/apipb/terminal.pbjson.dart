// This is a generated file - do not edit.
//
// Generated from apipb/terminal.proto.

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

@$core.Deprecated('Use terminalStateDescriptor instead')
const TerminalState$json = {
  '1': 'TerminalState',
  '2': [
    {'1': 'TERMINAL_STATE_UNSPECIFIED', '2': 0},
    {'1': 'TERMINAL_STATE_CREATED', '2': 1},
    {'1': 'TERMINAL_STATE_RUNNING', '2': 2},
    {'1': 'TERMINAL_STATE_EXITED', '2': 3},
    {'1': 'TERMINAL_STATE_REMOVED', '2': 4},
  ],
};

/// Descriptor for `TerminalState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List terminalStateDescriptor = $convert.base64Decode(
    'Cg1UZXJtaW5hbFN0YXRlEh4KGlRFUk1JTkFMX1NUQVRFX1VOU1BFQ0lGSUVEEAASGgoWVEVSTU'
    'lOQUxfU1RBVEVfQ1JFQVRFRBABEhoKFlRFUk1JTkFMX1NUQVRFX1JVTk5JTkcQAhIZChVURVJN'
    'SU5BTF9TVEFURV9FWElURUQQAxIaChZURVJNSU5BTF9TVEFURV9SRU1PVkVEEAQ=');

@$core.Deprecated('Use attachmentModeDescriptor instead')
const AttachmentMode$json = {
  '1': 'AttachmentMode',
  '2': [
    {'1': 'ATTACHMENT_MODE_UNSPECIFIED', '2': 0},
    {'1': 'ATTACHMENT_MODE_COLLABORATOR', '2': 1},
    {'1': 'ATTACHMENT_MODE_OBSERVER', '2': 2},
  ],
};

/// Descriptor for `AttachmentMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List attachmentModeDescriptor = $convert.base64Decode(
    'Cg5BdHRhY2htZW50TW9kZRIfChtBVFRBQ0hNRU5UX01PREVfVU5TUEVDSUZJRUQQABIgChxBVF'
    'RBQ0hNRU5UX01PREVfQ09MTEFCT1JBVE9SEAESHAoYQVRUQUNITUVOVF9NT0RFX09CU0VSVkVS'
    'EAI=');

@$core.Deprecated('Use resizePolicyDescriptor instead')
const ResizePolicy$json = {
  '1': 'ResizePolicy',
  '2': [
    {'1': 'RESIZE_POLICY_UNSPECIFIED', '2': 0},
    {'1': 'RESIZE_POLICY_OWNER', '2': 1},
    {'1': 'RESIZE_POLICY_FOLLOWER', '2': 2},
    {'1': 'RESIZE_POLICY_OBSERVER', '2': 3},
  ],
};

/// Descriptor for `ResizePolicy`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List resizePolicyDescriptor = $convert.base64Decode(
    'CgxSZXNpemVQb2xpY3kSHQoZUkVTSVpFX1BPTElDWV9VTlNQRUNJRklFRBAAEhcKE1JFU0laRV'
    '9QT0xJQ1lfT1dORVIQARIaChZSRVNJWkVfUE9MSUNZX0ZPTExPV0VSEAISGgoWUkVTSVpFX1BP'
    'TElDWV9PQlNFUlZFUhAD');

@$core.Deprecated('Use resizeControlReasonDescriptor instead')
const ResizeControlReason$json = {
  '1': 'ResizeControlReason',
  '2': [
    {'1': 'RESIZE_CONTROL_REASON_UNSPECIFIED', '2': 0},
    {'1': 'RESIZE_CONTROL_REASON_OWNER', '2': 1},
    {'1': 'RESIZE_CONTROL_REASON_FOLLOWER', '2': 2},
    {'1': 'RESIZE_CONTROL_REASON_OBSERVER', '2': 3},
    {'1': 'RESIZE_CONTROL_REASON_SIZE_LOCKED', '2': 4},
  ],
};

/// Descriptor for `ResizeControlReason`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List resizeControlReasonDescriptor = $convert.base64Decode(
    'ChNSZXNpemVDb250cm9sUmVhc29uEiUKIVJFU0laRV9DT05UUk9MX1JFQVNPTl9VTlNQRUNJRk'
    'lFRBAAEh8KG1JFU0laRV9DT05UUk9MX1JFQVNPTl9PV05FUhABEiIKHlJFU0laRV9DT05UUk9M'
    'X1JFQVNPTl9GT0xMT1dFUhACEiIKHlJFU0laRV9DT05UUk9MX1JFQVNPTl9PQlNFUlZFUhADEi'
    'UKIVJFU0laRV9DT05UUk9MX1JFQVNPTl9TSVpFX0xPQ0tFRBAE');

@$core.Deprecated('Use terminalRefDescriptor instead')
const TerminalRef$json = {
  '1': 'TerminalRef',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'terminal_id', '3': 2, '4': 1, '5': 9, '10': 'terminalId'},
  ],
};

/// Descriptor for `TerminalRef`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalRefDescriptor = $convert.base64Decode(
    'CgtUZXJtaW5hbFJlZhIfCgtlbmRwb2ludF9pZBgBIAEoCVIKZW5kcG9pbnRJZBIfCgt0ZXJtaW'
    '5hbF9pZBgCIAEoCVIKdGVybWluYWxJZA==');

@$core.Deprecated('Use terminalSizeDescriptor instead')
const TerminalSize$json = {
  '1': 'TerminalSize',
  '2': [
    {'1': 'cols', '3': 1, '4': 1, '5': 13, '10': 'cols'},
    {'1': 'rows', '3': 2, '4': 1, '5': 13, '10': 'rows'},
  ],
};

/// Descriptor for `TerminalSize`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalSizeDescriptor = $convert.base64Decode(
    'CgxUZXJtaW5hbFNpemUSEgoEY29scxgBIAEoDVIEY29scxISCgRyb3dzGAIgASgNUgRyb3dz');

@$core.Deprecated('Use terminalResourceUsageDescriptor instead')
const TerminalResourceUsage$json = {
  '1': 'TerminalResourceUsage',
  '2': [
    {'1': 'pid', '3': 1, '4': 1, '5': 5, '10': 'pid'},
    {'1': 'cpu_percent_x100', '3': 2, '4': 1, '5': 5, '10': 'cpuPercentX100'},
    {'1': 'memory_bytes', '3': 3, '4': 1, '5': 4, '10': 'memoryBytes'},
    {
      '1': 'sampled_at_unix_nano',
      '3': 4,
      '4': 1,
      '5': 3,
      '10': 'sampledAtUnixNano'
    },
  ],
};

/// Descriptor for `TerminalResourceUsage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalResourceUsageDescriptor = $convert.base64Decode(
    'ChVUZXJtaW5hbFJlc291cmNlVXNhZ2USEAoDcGlkGAEgASgFUgNwaWQSKAoQY3B1X3BlcmNlbn'
    'RfeDEwMBgCIAEoBVIOY3B1UGVyY2VudFgxMDASIQoMbWVtb3J5X2J5dGVzGAMgASgEUgttZW1v'
    'cnlCeXRlcxIvChRzYW1wbGVkX2F0X3VuaXhfbmFubxgEIAEoA1IRc2FtcGxlZEF0VW5peE5hbm'
    '8=');

@$core.Deprecated('Use terminalInfoDescriptor instead')
const TerminalInfo$json = {
  '1': 'TerminalInfo',
  '2': [
    {
      '1': 'ref',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'ref'
    },
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'command', '3': 3, '4': 3, '5': 9, '10': 'command'},
    {
      '1': 'tags',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.TerminalInfo.TagsEntry',
      '10': 'tags'
    },
    {
      '1': 'size',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSize',
      '10': 'size'
    },
    {
      '1': 'state',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.TerminalState',
      '10': 'state'
    },
    {'1': 'cwd', '3': 7, '4': 1, '5': 9, '10': 'cwd'},
    {'1': 'live_cwd', '3': 8, '4': 1, '5': 9, '10': 'liveCwd'},
    {
      '1': 'created_at_unix_nano',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'createdAtUnixNano'
    },
    {
      '1': 'exit_code',
      '3': 10,
      '4': 1,
      '5': 5,
      '9': 0,
      '10': 'exitCode',
      '17': true
    },
    {
      '1': 'exited_at_unix_nano',
      '3': 11,
      '4': 1,
      '5': 3,
      '10': 'exitedAtUnixNano'
    },
    {'1': 'attachment_count', '3': 12, '4': 1, '5': 5, '10': 'attachmentCount'},
    {
      '1': 'resources',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalResourceUsage',
      '10': 'resources'
    },
    {
      '1': 'resource_history',
      '3': 14,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.TerminalResourceUsage',
      '10': 'resourceHistory'
    },
    {
      '1': 'foreground_process',
      '3': 15,
      '4': 1,
      '5': 9,
      '10': 'foregroundProcess'
    },
    {
      '1': 'last_output_at_unix_nano',
      '3': 16,
      '4': 1,
      '5': 3,
      '10': 'lastOutputAtUnixNano'
    },
    {'1': 'foreground_cwd', '3': 17, '4': 1, '5': 9, '10': 'foregroundCwd'},
  ],
  '3': [TerminalInfo_TagsEntry$json],
  '8': [
    {'1': '_exit_code'},
  ],
};

@$core.Deprecated('Use terminalInfoDescriptor instead')
const TerminalInfo_TagsEntry$json = {
  '1': 'TagsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `TerminalInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalInfoDescriptor = $convert.base64Decode(
    'CgxUZXJtaW5hbEluZm8SLAoDcmVmGAEgASgLMhouYW55dHR5LmFwaS52MS5UZXJtaW5hbFJlZl'
    'IDcmVmEhIKBG5hbWUYAiABKAlSBG5hbWUSGAoHY29tbWFuZBgDIAMoCVIHY29tbWFuZBI5CgR0'
    'YWdzGAQgAygLMiUuYW55dHR5LmFwaS52MS5UZXJtaW5hbEluZm8uVGFnc0VudHJ5UgR0YWdzEi'
    '8KBHNpemUYBSABKAsyGy5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsU2l6ZVIEc2l6ZRIyCgVzdGF0'
    'ZRgGIAEoDjIcLmFueXR0eS5hcGkudjEuVGVybWluYWxTdGF0ZVIFc3RhdGUSEAoDY3dkGAcgAS'
    'gJUgNjd2QSGQoIbGl2ZV9jd2QYCCABKAlSB2xpdmVDd2QSLwoUY3JlYXRlZF9hdF91bml4X25h'
    'bm8YCSABKANSEWNyZWF0ZWRBdFVuaXhOYW5vEiAKCWV4aXRfY29kZRgKIAEoBUgAUghleGl0Q2'
    '9kZYgBARItChNleGl0ZWRfYXRfdW5peF9uYW5vGAsgASgDUhBleGl0ZWRBdFVuaXhOYW5vEikK'
    'EGF0dGFjaG1lbnRfY291bnQYDCABKAVSD2F0dGFjaG1lbnRDb3VudBJCCglyZXNvdXJjZXMYDS'
    'ABKAsyJC5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsUmVzb3VyY2VVc2FnZVIJcmVzb3VyY2VzEk8K'
    'EHJlc291cmNlX2hpc3RvcnkYDiADKAsyJC5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsUmVzb3VyY2'
    'VVc2FnZVIPcmVzb3VyY2VIaXN0b3J5Ei0KEmZvcmVncm91bmRfcHJvY2VzcxgPIAEoCVIRZm9y'
    'ZWdyb3VuZFByb2Nlc3MSNgoYbGFzdF9vdXRwdXRfYXRfdW5peF9uYW5vGBAgASgDUhRsYXN0T3'
    'V0cHV0QXRVbml4TmFubxIlCg5mb3JlZ3JvdW5kX2N3ZBgRIAEoCVINZm9yZWdyb3VuZEN3ZBo3'
    'CglUYWdzRW50cnkSEAoDa2V5GAEgASgJUgNrZXkSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AU'
    'IMCgpfZXhpdF9jb2Rl');

@$core.Deprecated('Use terminalCreateSpecDescriptor instead')
const TerminalCreateSpec$json = {
  '1': 'TerminalCreateSpec',
  '2': [
    {'1': 'terminal_id', '3': 1, '4': 1, '5': 9, '10': 'terminalId'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'command', '3': 3, '4': 3, '5': 9, '10': 'command'},
    {
      '1': 'tags',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.TerminalCreateSpec.TagsEntry',
      '10': 'tags'
    },
    {
      '1': 'size',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSize',
      '10': 'size'
    },
    {'1': 'cwd', '3': 6, '4': 1, '5': 9, '10': 'cwd'},
    {'1': 'env', '3': 7, '4': 3, '5': 9, '10': 'env'},
    {'1': 'scrollback_rows', '3': 8, '4': 1, '5': 5, '10': 'scrollbackRows'},
    {
      '1': 'scrollback_max_bytes',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'scrollbackMaxBytes'
    },
    {
      '1': 'scrollback_max_age_seconds',
      '3': 10,
      '4': 1,
      '5': 3,
      '10': 'scrollbackMaxAgeSeconds'
    },
  ],
  '3': [TerminalCreateSpec_TagsEntry$json],
};

@$core.Deprecated('Use terminalCreateSpecDescriptor instead')
const TerminalCreateSpec_TagsEntry$json = {
  '1': 'TagsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `TerminalCreateSpec`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalCreateSpecDescriptor = $convert.base64Decode(
    'ChJUZXJtaW5hbENyZWF0ZVNwZWMSHwoLdGVybWluYWxfaWQYASABKAlSCnRlcm1pbmFsSWQSEg'
    'oEbmFtZRgCIAEoCVIEbmFtZRIYCgdjb21tYW5kGAMgAygJUgdjb21tYW5kEj8KBHRhZ3MYBCAD'
    'KAsyKy5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsQ3JlYXRlU3BlYy5UYWdzRW50cnlSBHRhZ3MSLw'
    'oEc2l6ZRgFIAEoCzIbLmFueXR0eS5hcGkudjEuVGVybWluYWxTaXplUgRzaXplEhAKA2N3ZBgG'
    'IAEoCVIDY3dkEhAKA2VudhgHIAMoCVIDZW52EicKD3Njcm9sbGJhY2tfcm93cxgIIAEoBVIOc2'
    'Nyb2xsYmFja1Jvd3MSMAoUc2Nyb2xsYmFja19tYXhfYnl0ZXMYCSABKANSEnNjcm9sbGJhY2tN'
    'YXhCeXRlcxI7ChpzY3JvbGxiYWNrX21heF9hZ2Vfc2Vjb25kcxgKIAEoA1IXc2Nyb2xsYmFja0'
    '1heEFnZVNlY29uZHMaNwoJVGFnc0VudHJ5EhAKA2tleRgBIAEoCVIDa2V5EhQKBXZhbHVlGAIg'
    'ASgJUgV2YWx1ZToCOAE=');

@$core.Deprecated('Use terminalDefaultsDescriptor instead')
const TerminalDefaults$json = {
  '1': 'TerminalDefaults',
  '2': [
    {'1': 'default_command', '3': 1, '4': 3, '5': 9, '10': 'defaultCommand'},
    {'1': 'default_cwd', '3': 2, '4': 1, '5': 9, '10': 'defaultCwd'},
    {'1': 'platform', '3': 3, '4': 1, '5': 9, '10': 'platform'},
  ],
};

/// Descriptor for `TerminalDefaults`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalDefaultsDescriptor = $convert.base64Decode(
    'ChBUZXJtaW5hbERlZmF1bHRzEicKD2RlZmF1bHRfY29tbWFuZBgBIAMoCVIOZGVmYXVsdENvbW'
    '1hbmQSHwoLZGVmYXVsdF9jd2QYAiABKAlSCmRlZmF1bHRDd2QSGgoIcGxhdGZvcm0YAyABKAlS'
    'CHBsYXRmb3Jt');

@$core.Deprecated('Use resizeOwnershipDescriptor instead')
const ResizeOwnership$json = {
  '1': 'ResizeOwnership',
  '2': [
    {
      '1': 'owner_attachment_id',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'ownerAttachmentId'
    },
    {'1': 'owner_surface_id', '3': 2, '4': 1, '5': 9, '10': 'ownerSurfaceId'},
    {'1': 'owner_view_id', '3': 3, '4': 1, '5': 9, '10': 'ownerViewId'},
    {
      '1': 'size',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSize',
      '10': 'size'
    },
    {'1': 'size_locked', '3': 5, '4': 1, '5': 8, '10': 'sizeLocked'},
    {'1': 'epoch', '3': 6, '4': 1, '5': 4, '10': 'epoch'},
  ],
};

/// Descriptor for `ResizeOwnership`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resizeOwnershipDescriptor = $convert.base64Decode(
    'Cg9SZXNpemVPd25lcnNoaXASLgoTb3duZXJfYXR0YWNobWVudF9pZBgBIAEoCVIRb3duZXJBdH'
    'RhY2htZW50SWQSKAoQb3duZXJfc3VyZmFjZV9pZBgCIAEoCVIOb3duZXJTdXJmYWNlSWQSIgoN'
    'b3duZXJfdmlld19pZBgDIAEoCVILb3duZXJWaWV3SWQSLwoEc2l6ZRgEIAEoCzIbLmFueXR0eS'
    '5hcGkudjEuVGVybWluYWxTaXplUgRzaXplEh8KC3NpemVfbG9ja2VkGAUgASgIUgpzaXplTG9j'
    'a2VkEhQKBWVwb2NoGAYgASgEUgVlcG9jaA==');

@$core.Deprecated('Use resizeControlDescriptor instead')
const ResizeControl$json = {
  '1': 'ResizeControl',
  '2': [
    {'1': 'can_resize', '3': 1, '4': 1, '5': 8, '10': 'canResize'},
    {
      '1': 'reason',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.ResizeControlReason',
      '10': 'reason'
    },
    {'1': 'size_locked', '3': 3, '4': 1, '5': 8, '10': 'sizeLocked'},
    {'1': 'surface_id', '3': 4, '4': 1, '5': 9, '10': 'surfaceId'},
    {'1': 'owner_surface_id', '3': 5, '4': 1, '5': 9, '10': 'ownerSurfaceId'},
    {'1': 'owner_view_id', '3': 6, '4': 1, '5': 9, '10': 'ownerViewId'},
    {
      '1': 'ownership',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResizeOwnership',
      '10': 'ownership'
    },
  ],
};

/// Descriptor for `ResizeControl`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resizeControlDescriptor = $convert.base64Decode(
    'Cg1SZXNpemVDb250cm9sEh0KCmNhbl9yZXNpemUYASABKAhSCWNhblJlc2l6ZRI6CgZyZWFzb2'
    '4YAiABKA4yIi5hbnl0dHkuYXBpLnYxLlJlc2l6ZUNvbnRyb2xSZWFzb25SBnJlYXNvbhIfCgtz'
    'aXplX2xvY2tlZBgDIAEoCFIKc2l6ZUxvY2tlZBIdCgpzdXJmYWNlX2lkGAQgASgJUglzdXJmYW'
    'NlSWQSKAoQb3duZXJfc3VyZmFjZV9pZBgFIAEoCVIOb3duZXJTdXJmYWNlSWQSIgoNb3duZXJf'
    'dmlld19pZBgGIAEoCVILb3duZXJWaWV3SWQSPAoJb3duZXJzaGlwGAcgASgLMh4uYW55dHR5Lm'
    'FwaS52MS5SZXNpemVPd25lcnNoaXBSCW93bmVyc2hpcA==');

@$core.Deprecated('Use attachmentHandleDescriptor instead')
const AttachmentHandle$json = {
  '1': 'AttachmentHandle',
  '2': [
    {
      '1': 'resource',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
    {'1': 'surface_id', '3': 4, '4': 1, '5': 9, '10': 'surfaceId'},
    {'1': 'view_id', '3': 5, '4': 1, '5': 9, '10': 'viewId'},
  ],
};

/// Descriptor for `AttachmentHandle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List attachmentHandleDescriptor = $convert.base64Decode(
    'ChBBdHRhY2htZW50SGFuZGxlEjkKCHJlc291cmNlGAEgASgLMh0uYW55dHR5LmFwaS52MS5SZX'
    'NvdXJjZUhhbmRsZVIIcmVzb3VyY2USNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLnYx'
    'LlRlcm1pbmFsUmVmUgh0ZXJtaW5hbBI7CglvcGVyYXRpb24YAyABKAsyHS5hbnl0dHkuYXBpLn'
    'YxLk9wZXJhdGlvblN0YW1wUglvcGVyYXRpb24SHQoKc3VyZmFjZV9pZBgEIAEoCVIJc3VyZmFj'
    'ZUlkEhcKB3ZpZXdfaWQYBSABKAlSBnZpZXdJZA==');

@$core.Deprecated('Use pathDirectoryEntryDescriptor instead')
const PathDirectoryEntry$json = {
  '1': 'PathDirectoryEntry',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
  ],
};

/// Descriptor for `PathDirectoryEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pathDirectoryEntryDescriptor = $convert.base64Decode(
    'ChJQYXRoRGlyZWN0b3J5RW50cnkSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgRwYXRoGAIgASgJUg'
    'RwYXRo');

@$core.Deprecated('Use terminalDefaultsCommandDescriptor instead')
const TerminalDefaultsCommand$json = {
  '1': 'TerminalDefaultsCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalDefaultsCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalDefaultsCommandDescriptor =
    $convert.base64Decode('ChdUZXJtaW5hbERlZmF1bHRzQ29tbWFuZEoECAEQAg==');

@$core.Deprecated('Use terminalCreateCommandDescriptor instead')
const TerminalCreateCommand$json = {
  '1': 'TerminalCreateCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalCreateSpec',
      '10': 'terminal'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalCreateCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalCreateCommandDescriptor = $convert.base64Decode(
    'ChVUZXJtaW5hbENyZWF0ZUNvbW1hbmQSPQoIdGVybWluYWwYAiABKAsyIS5hbnl0dHkuYXBpLn'
    'YxLlRlcm1pbmFsQ3JlYXRlU3BlY1IIdGVybWluYWxKBAgBEAI=');

@$core.Deprecated('Use terminalListCommandDescriptor instead')
const TerminalListCommand$json = {
  '1': 'TerminalListCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalListCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalListCommandDescriptor =
    $convert.base64Decode('ChNUZXJtaW5hbExpc3RDb21tYW5kSgQIARAC');

@$core.Deprecated('Use terminalGetCommandDescriptor instead')
const TerminalGetCommand$json = {
  '1': 'TerminalGetCommand',
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

/// Descriptor for `TerminalGetCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalGetCommandDescriptor = $convert.base64Decode(
    'ChJUZXJtaW5hbEdldENvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLnYxLl'
    'Rlcm1pbmFsUmVmUgh0ZXJtaW5hbEoECAEQAg==');

@$core.Deprecated('Use terminalRestartCommandDescriptor instead')
const TerminalRestartCommand$json = {
  '1': 'TerminalRestartCommand',
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

/// Descriptor for `TerminalRestartCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalRestartCommandDescriptor =
    $convert.base64Decode(
        'ChZUZXJtaW5hbFJlc3RhcnRDb21tYW5kEjYKCHRlcm1pbmFsGAIgASgLMhouYW55dHR5LmFwaS'
        '52MS5UZXJtaW5hbFJlZlIIdGVybWluYWxKBAgBEAI=');

@$core.Deprecated('Use terminalKillCommandDescriptor instead')
const TerminalKillCommand$json = {
  '1': 'TerminalKillCommand',
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

/// Descriptor for `TerminalKillCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalKillCommandDescriptor = $convert.base64Decode(
    'ChNUZXJtaW5hbEtpbGxDb21tYW5kEjYKCHRlcm1pbmFsGAIgASgLMhouYW55dHR5LmFwaS52MS'
    '5UZXJtaW5hbFJlZlIIdGVybWluYWxKBAgBEAI=');

@$core.Deprecated('Use terminalRemoveCommandDescriptor instead')
const TerminalRemoveCommand$json = {
  '1': 'TerminalRemoveCommand',
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

/// Descriptor for `TerminalRemoveCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalRemoveCommandDescriptor = $convert.base64Decode(
    'ChVUZXJtaW5hbFJlbW92ZUNvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLn'
    'YxLlRlcm1pbmFsUmVmUgh0ZXJtaW5hbEoECAEQAg==');

@$core.Deprecated('Use terminalSetMetadataCommandDescriptor instead')
const TerminalSetMetadataCommand$json = {
  '1': 'TerminalSetMetadataCommand',
  '2': [
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'name', '3': 3, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'tags',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSetMetadataCommand.TagsEntry',
      '10': 'tags'
    },
  ],
  '3': [TerminalSetMetadataCommand_TagsEntry$json],
  '9': [
    {'1': 1, '2': 2},
  ],
};

@$core.Deprecated('Use terminalSetMetadataCommandDescriptor instead')
const TerminalSetMetadataCommand_TagsEntry$json = {
  '1': 'TagsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `TerminalSetMetadataCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalSetMetadataCommandDescriptor = $convert.base64Decode(
    'ChpUZXJtaW5hbFNldE1ldGFkYXRhQ29tbWFuZBI2Cgh0ZXJtaW5hbBgCIAEoCzIaLmFueXR0eS'
    '5hcGkudjEuVGVybWluYWxSZWZSCHRlcm1pbmFsEhIKBG5hbWUYAyABKAlSBG5hbWUSRwoEdGFn'
    'cxgEIAMoCzIzLmFueXR0eS5hcGkudjEuVGVybWluYWxTZXRNZXRhZGF0YUNvbW1hbmQuVGFnc0'
    'VudHJ5UgR0YWdzGjcKCVRhZ3NFbnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEo'
    'CVIFdmFsdWU6AjgBSgQIARAC');

@$core.Deprecated('Use terminalSetTagsCommandDescriptor instead')
const TerminalSetTagsCommand$json = {
  '1': 'TerminalSetTagsCommand',
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
      '1': 'tags',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSetTagsCommand.TagsEntry',
      '10': 'tags'
    },
  ],
  '3': [TerminalSetTagsCommand_TagsEntry$json],
  '9': [
    {'1': 1, '2': 2},
  ],
};

@$core.Deprecated('Use terminalSetTagsCommandDescriptor instead')
const TerminalSetTagsCommand_TagsEntry$json = {
  '1': 'TagsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `TerminalSetTagsCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalSetTagsCommandDescriptor = $convert.base64Decode(
    'ChZUZXJtaW5hbFNldFRhZ3NDb21tYW5kEjYKCHRlcm1pbmFsGAIgASgLMhouYW55dHR5LmFwaS'
    '52MS5UZXJtaW5hbFJlZlIIdGVybWluYWwSQwoEdGFncxgDIAMoCzIvLmFueXR0eS5hcGkudjEu'
    'VGVybWluYWxTZXRUYWdzQ29tbWFuZC5UYWdzRW50cnlSBHRhZ3MaNwoJVGFnc0VudHJ5EhAKA2'
    'tleRgBIAEoCVIDa2V5EhQKBXZhbHVlGAIgASgJUgV2YWx1ZToCOAFKBAgBEAI=');

@$core.Deprecated('Use terminalAttachCommandDescriptor instead')
const TerminalAttachCommand$json = {
  '1': 'TerminalAttachCommand',
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
      '6': '.anytty.api.v1.AttachmentMode',
      '10': 'mode'
    },
    {
      '1': 'resize_policy',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.ResizePolicy',
      '10': 'resizePolicy'
    },
    {'1': 'surface_id', '3': 5, '4': 1, '5': 9, '10': 'surfaceId'},
    {'1': 'view_id', '3': 6, '4': 1, '5': 9, '10': 'viewId'},
    {
      '1': 'operation',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalAttachCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalAttachCommandDescriptor = $convert.base64Decode(
    'ChVUZXJtaW5hbEF0dGFjaENvbW1hbmQSNgoIdGVybWluYWwYAiABKAsyGi5hbnl0dHkuYXBpLn'
    'YxLlRlcm1pbmFsUmVmUgh0ZXJtaW5hbBIxCgRtb2RlGAMgASgOMh0uYW55dHR5LmFwaS52MS5B'
    'dHRhY2htZW50TW9kZVIEbW9kZRJACg1yZXNpemVfcG9saWN5GAQgASgOMhsuYW55dHR5LmFwaS'
    '52MS5SZXNpemVQb2xpY3lSDHJlc2l6ZVBvbGljeRIdCgpzdXJmYWNlX2lkGAUgASgJUglzdXJm'
    'YWNlSWQSFwoHdmlld19pZBgGIAEoCVIGdmlld0lkEjsKCW9wZXJhdGlvbhgHIAEoCzIdLmFueX'
    'R0eS5hcGkudjEuT3BlcmF0aW9uU3RhbXBSCW9wZXJhdGlvbkoECAEQAg==');

@$core.Deprecated('Use terminalDetachCommandDescriptor instead')
const TerminalDetachCommand$json = {
  '1': 'TerminalDetachCommand',
  '2': [
    {
      '1': 'attachment',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'attachment'
    },
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalDetachCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalDetachCommandDescriptor = $convert.base64Decode(
    'ChVUZXJtaW5hbERldGFjaENvbW1hbmQSPQoKYXR0YWNobWVudBgCIAEoCzIdLmFueXR0eS5hcG'
    'kudjEuUmVzb3VyY2VIYW5kbGVSCmF0dGFjaG1lbnQSOwoJb3BlcmF0aW9uGAMgASgLMh0uYW55'
    'dHR5LmFwaS52MS5PcGVyYXRpb25TdGFtcFIJb3BlcmF0aW9uSgQIARAC');

@$core.Deprecated('Use terminalInputCommandDescriptor instead')
const TerminalInputCommand$json = {
  '1': 'TerminalInputCommand',
  '2': [
    {
      '1': 'attachment',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'attachment'
    },
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
    {'1': 'data', '3': 4, '4': 1, '5': 12, '10': 'data'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalInputCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalInputCommandDescriptor = $convert.base64Decode(
    'ChRUZXJtaW5hbElucHV0Q29tbWFuZBI9CgphdHRhY2htZW50GAIgASgLMh0uYW55dHR5LmFwaS'
    '52MS5SZXNvdXJjZUhhbmRsZVIKYXR0YWNobWVudBI7CglvcGVyYXRpb24YAyABKAsyHS5hbnl0'
    'dHkuYXBpLnYxLk9wZXJhdGlvblN0YW1wUglvcGVyYXRpb24SEgoEZGF0YRgEIAEoDFIEZGF0YU'
    'oECAEQAg==');

@$core.Deprecated('Use terminalResizeCommandDescriptor instead')
const TerminalResizeCommand$json = {
  '1': 'TerminalResizeCommand',
  '2': [
    {
      '1': 'attachment',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'attachment'
    },
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
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
      '1': 'resize_policy',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.ResizePolicy',
      '10': 'resizePolicy'
    },
    {'1': 'take_ownership', '3': 6, '4': 1, '5': 8, '10': 'takeOwnership'},
    {
      '1': 'expected_owner_epoch',
      '3': 7,
      '4': 1,
      '5': 4,
      '10': 'expectedOwnerEpoch'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalResizeCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalResizeCommandDescriptor = $convert.base64Decode(
    'ChVUZXJtaW5hbFJlc2l6ZUNvbW1hbmQSPQoKYXR0YWNobWVudBgCIAEoCzIdLmFueXR0eS5hcG'
    'kudjEuUmVzb3VyY2VIYW5kbGVSCmF0dGFjaG1lbnQSOwoJb3BlcmF0aW9uGAMgASgLMh0uYW55'
    'dHR5LmFwaS52MS5PcGVyYXRpb25TdGFtcFIJb3BlcmF0aW9uEi8KBHNpemUYBCABKAsyGy5hbn'
    'l0dHkuYXBpLnYxLlRlcm1pbmFsU2l6ZVIEc2l6ZRJACg1yZXNpemVfcG9saWN5GAUgASgOMhsu'
    'YW55dHR5LmFwaS52MS5SZXNpemVQb2xpY3lSDHJlc2l6ZVBvbGljeRIlCg50YWtlX293bmVyc2'
    'hpcBgGIAEoCFINdGFrZU93bmVyc2hpcBIwChRleHBlY3RlZF9vd25lcl9lcG9jaBgHIAEoBFIS'
    'ZXhwZWN0ZWRPd25lckVwb2NoSgQIARAC');

@$core.Deprecated('Use terminalResizeLockCommandDescriptor instead')
const TerminalResizeLockCommand$json = {
  '1': 'TerminalResizeLockCommand',
  '2': [
    {
      '1': 'attachment',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'attachment'
    },
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
    {'1': 'locked', '3': 4, '4': 1, '5': 8, '10': 'locked'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `TerminalResizeLockCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalResizeLockCommandDescriptor = $convert.base64Decode(
    'ChlUZXJtaW5hbFJlc2l6ZUxvY2tDb21tYW5kEj0KCmF0dGFjaG1lbnQYAiABKAsyHS5hbnl0dH'
    'kuYXBpLnYxLlJlc291cmNlSGFuZGxlUgphdHRhY2htZW50EjsKCW9wZXJhdGlvbhgDIAEoCzId'
    'LmFueXR0eS5hcGkudjEuT3BlcmF0aW9uU3RhbXBSCW9wZXJhdGlvbhIWCgZsb2NrZWQYBCABKA'
    'hSBmxvY2tlZEoECAEQAg==');

@$core.Deprecated('Use pathListDirectoriesCommandDescriptor instead')
const PathListDirectoriesCommand$json = {
  '1': 'PathListDirectoriesCommand',
  '2': [
    {'1': 'prefix', '3': 2, '4': 1, '5': 9, '10': 'prefix'},
    {'1': 'limit', '3': 3, '4': 1, '5': 5, '10': 'limit'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `PathListDirectoriesCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pathListDirectoriesCommandDescriptor =
    $convert.base64Decode(
        'ChpQYXRoTGlzdERpcmVjdG9yaWVzQ29tbWFuZBIWCgZwcmVmaXgYAiABKAlSBnByZWZpeBIUCg'
        'VsaW1pdBgDIAEoBVIFbGltaXRKBAgBEAI=');

@$core.Deprecated('Use terminalCreateResultDescriptor instead')
const TerminalCreateResult$json = {
  '1': 'TerminalCreateResult',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalInfo',
      '10': 'terminal'
    },
  ],
};

/// Descriptor for `TerminalCreateResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalCreateResultDescriptor = $convert.base64Decode(
    'ChRUZXJtaW5hbENyZWF0ZVJlc3VsdBI3Cgh0ZXJtaW5hbBgBIAEoCzIbLmFueXR0eS5hcGkudj'
    'EuVGVybWluYWxJbmZvUgh0ZXJtaW5hbA==');

@$core.Deprecated('Use terminalListResultDescriptor instead')
const TerminalListResult$json = {
  '1': 'TerminalListResult',
  '2': [
    {
      '1': 'terminals',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.TerminalInfo',
      '10': 'terminals'
    },
  ],
};

/// Descriptor for `TerminalListResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalListResultDescriptor = $convert.base64Decode(
    'ChJUZXJtaW5hbExpc3RSZXN1bHQSOQoJdGVybWluYWxzGAEgAygLMhsuYW55dHR5LmFwaS52MS'
    '5UZXJtaW5hbEluZm9SCXRlcm1pbmFscw==');

@$core.Deprecated('Use terminalGetResultDescriptor instead')
const TerminalGetResult$json = {
  '1': 'TerminalGetResult',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalInfo',
      '10': 'terminal'
    },
  ],
};

/// Descriptor for `TerminalGetResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalGetResultDescriptor = $convert.base64Decode(
    'ChFUZXJtaW5hbEdldFJlc3VsdBI3Cgh0ZXJtaW5hbBgBIAEoCzIbLmFueXR0eS5hcGkudjEuVG'
    'VybWluYWxJbmZvUgh0ZXJtaW5hbA==');

@$core.Deprecated('Use terminalDefaultsResultDescriptor instead')
const TerminalDefaultsResult$json = {
  '1': 'TerminalDefaultsResult',
  '2': [
    {
      '1': 'defaults',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalDefaults',
      '10': 'defaults'
    },
  ],
};

/// Descriptor for `TerminalDefaultsResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalDefaultsResultDescriptor =
    $convert.base64Decode(
        'ChZUZXJtaW5hbERlZmF1bHRzUmVzdWx0EjsKCGRlZmF1bHRzGAEgASgLMh8uYW55dHR5LmFwaS'
        '52MS5UZXJtaW5hbERlZmF1bHRzUghkZWZhdWx0cw==');

@$core.Deprecated('Use terminalAttachResultDescriptor instead')
const TerminalAttachResult$json = {
  '1': 'TerminalAttachResult',
  '2': [
    {
      '1': 'attachment',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.AttachmentHandle',
      '10': 'attachment'
    },
    {
      '1': 'mode',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.AttachmentMode',
      '10': 'mode'
    },
    {
      '1': 'resize_policy',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.ResizePolicy',
      '10': 'resizePolicy'
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
      '1': 'resize_control',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResizeControl',
      '10': 'resizeControl'
    },
  ],
};

/// Descriptor for `TerminalAttachResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalAttachResultDescriptor = $convert.base64Decode(
    'ChRUZXJtaW5hbEF0dGFjaFJlc3VsdBI/CgphdHRhY2htZW50GAEgASgLMh8uYW55dHR5LmFwaS'
    '52MS5BdHRhY2htZW50SGFuZGxlUgphdHRhY2htZW50EjEKBG1vZGUYAiABKA4yHS5hbnl0dHku'
    'YXBpLnYxLkF0dGFjaG1lbnRNb2RlUgRtb2RlEkAKDXJlc2l6ZV9wb2xpY3kYAyABKA4yGy5hbn'
    'l0dHkuYXBpLnYxLlJlc2l6ZVBvbGljeVIMcmVzaXplUG9saWN5Ei8KBHNpemUYBCABKAsyGy5h'
    'bnl0dHkuYXBpLnYxLlRlcm1pbmFsU2l6ZVIEc2l6ZRJDCg5yZXNpemVfY29udHJvbBgFIAEoCz'
    'IcLmFueXR0eS5hcGkudjEuUmVzaXplQ29udHJvbFINcmVzaXplQ29udHJvbA==');

@$core.Deprecated('Use terminalResizeResultDescriptor instead')
const TerminalResizeResult$json = {
  '1': 'TerminalResizeResult',
  '2': [
    {
      '1': 'size',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSize',
      '10': 'size'
    },
    {'1': 'resized', '3': 2, '4': 1, '5': 8, '10': 'resized'},
    {
      '1': 'resize_control',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResizeControl',
      '10': 'resizeControl'
    },
  ],
};

/// Descriptor for `TerminalResizeResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalResizeResultDescriptor = $convert.base64Decode(
    'ChRUZXJtaW5hbFJlc2l6ZVJlc3VsdBIvCgRzaXplGAEgASgLMhsuYW55dHR5LmFwaS52MS5UZX'
    'JtaW5hbFNpemVSBHNpemUSGAoHcmVzaXplZBgCIAEoCFIHcmVzaXplZBJDCg5yZXNpemVfY29u'
    'dHJvbBgDIAEoCzIcLmFueXR0eS5hcGkudjEuUmVzaXplQ29udHJvbFINcmVzaXplQ29udHJvbA'
    '==');

@$core.Deprecated('Use pathListDirectoriesResultDescriptor instead')
const PathListDirectoriesResult$json = {
  '1': 'PathListDirectoriesResult',
  '2': [
    {'1': 'base_path', '3': 1, '4': 1, '5': 9, '10': 'basePath'},
    {
      '1': 'entries',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PathDirectoryEntry',
      '10': 'entries'
    },
    {'1': 'missing', '3': 3, '4': 1, '5': 8, '10': 'missing'},
    {'1': 'truncated', '3': 4, '4': 1, '5': 8, '10': 'truncated'},
  ],
};

/// Descriptor for `PathListDirectoriesResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pathListDirectoriesResultDescriptor = $convert.base64Decode(
    'ChlQYXRoTGlzdERpcmVjdG9yaWVzUmVzdWx0EhsKCWJhc2VfcGF0aBgBIAEoCVIIYmFzZVBhdG'
    'gSOwoHZW50cmllcxgCIAMoCzIhLmFueXR0eS5hcGkudjEuUGF0aERpcmVjdG9yeUVudHJ5Ugdl'
    'bnRyaWVzEhgKB21pc3NpbmcYAyABKAhSB21pc3NpbmcSHAoJdHJ1bmNhdGVkGAQgASgIUgl0cn'
    'VuY2F0ZWQ=');

@$core.Deprecated('Use terminalLifecycleEventDescriptor instead')
const TerminalLifecycleEvent$json = {
  '1': 'TerminalLifecycleEvent',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalInfo',
      '10': 'terminal'
    },
    {
      '1': 'attachment_projection',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'attachmentProjection'
    },
    {
      '1': 'resize_control',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResizeControl',
      '10': 'resizeControl'
    },
    {'1': 'resize_epoch', '3': 4, '4': 1, '5': 4, '10': 'resizeEpoch'},
  ],
};

/// Descriptor for `TerminalLifecycleEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalLifecycleEventDescriptor = $convert.base64Decode(
    'ChZUZXJtaW5hbExpZmVjeWNsZUV2ZW50EjcKCHRlcm1pbmFsGAEgASgLMhsuYW55dHR5LmFwaS'
    '52MS5UZXJtaW5hbEluZm9SCHRlcm1pbmFsEjMKFWF0dGFjaG1lbnRfcHJvamVjdGlvbhgCIAEo'
    'CFIUYXR0YWNobWVudFByb2plY3Rpb24SQwoOcmVzaXplX2NvbnRyb2wYAyABKAsyHC5hbnl0dH'
    'kuYXBpLnYxLlJlc2l6ZUNvbnRyb2xSDXJlc2l6ZUNvbnRyb2wSIQoMcmVzaXplX2Vwb2NoGAQg'
    'ASgEUgtyZXNpemVFcG9jaA==');

@$core.Deprecated('Use terminalResizeControlEventDescriptor instead')
const TerminalResizeControlEvent$json = {
  '1': 'TerminalResizeControlEvent',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {
      '1': 'resize_control',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResizeControl',
      '10': 'resizeControl'
    },
  ],
};

/// Descriptor for `TerminalResizeControlEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalResizeControlEventDescriptor =
    $convert.base64Decode(
        'ChpUZXJtaW5hbFJlc2l6ZUNvbnRyb2xFdmVudBI2Cgh0ZXJtaW5hbBgBIAEoCzIaLmFueXR0eS'
        '5hcGkudjEuVGVybWluYWxSZWZSCHRlcm1pbmFsEkMKDnJlc2l6ZV9jb250cm9sGAIgASgLMhwu'
        'YW55dHR5LmFwaS52MS5SZXNpemVDb250cm9sUg1yZXNpemVDb250cm9s');
