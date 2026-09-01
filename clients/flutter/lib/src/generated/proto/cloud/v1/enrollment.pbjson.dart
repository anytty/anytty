// This is a generated file - do not edit.
//
// Generated from cloud/v1/enrollment.proto.

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

import 'package:protobuf/well_known_types/google/protobuf/timestamp.pbjson.dart'
    as $0;

import 'common.pbjson.dart' as $1;

@$core.Deprecated('Use daemonStateDescriptor instead')
const DaemonState$json = {
  '1': 'DaemonState',
  '2': [
    {'1': 'DAEMON_STATE_UNSPECIFIED', '2': 0},
    {'1': 'DAEMON_STATE_ACTIVE', '2': 1},
    {'1': 'DAEMON_STATE_BLOCKED', '2': 2},
    {'1': 'DAEMON_STATE_DELETED', '2': 3},
  ],
};

/// Descriptor for `DaemonState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List daemonStateDescriptor = $convert.base64Decode(
    'CgtEYWVtb25TdGF0ZRIcChhEQUVNT05fU1RBVEVfVU5TUEVDSUZJRUQQABIXChNEQUVNT05fU1'
    'RBVEVfQUNUSVZFEAESGAoUREFFTU9OX1NUQVRFX0JMT0NLRUQQAhIYChREQUVNT05fU1RBVEVf'
    'REVMRVRFRBAD');

@$core.Deprecated('Use daemonRecordDescriptor instead')
const DaemonRecord$json = {
  '1': 'DaemonRecord',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'account_name', '3': 3, '4': 1, '5': 9, '10': 'accountName'},
    {'1': 'display_name', '3': 4, '4': 1, '5': 9, '10': 'displayName'},
    {'1': 'device_id', '3': 5, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_fingerprint',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {
      '1': 'state',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.DaemonState',
      '10': 'state'
    },
    {'1': 'state_revision', '3': 8, '4': 1, '5': 4, '10': 'stateRevision'},
    {
      '1': 'created_at',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createdAt'
    },
    {
      '1': 'updated_at',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'updatedAt'
    },
    {
      '1': 'preferred_edge_id',
      '3': 11,
      '4': 1,
      '5': 9,
      '10': 'preferredEdgeId'
    },
    {
      '1': 'edge_preference_revision',
      '3': 12,
      '4': 1,
      '5': 4,
      '10': 'edgePreferenceRevision'
    },
    {
      '1': 'edge_preference_updated_at',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'edgePreferenceUpdatedAt'
    },
  ],
};

/// Descriptor for `DaemonRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonRecordDescriptor = $convert.base64Decode(
    'CgxEYWVtb25SZWNvcmQSGwoJZGFlbW9uX2lkGAEgASgJUghkYWVtb25JZBIdCgphY2NvdW50X2'
    'lkGAIgASgJUglhY2NvdW50SWQSIQoMYWNjb3VudF9uYW1lGAMgASgJUgthY2NvdW50TmFtZRIh'
    'CgxkaXNwbGF5X25hbWUYBCABKAlSC2Rpc3BsYXlOYW1lEhsKCWRldmljZV9pZBgFIAEoCVIIZG'
    'V2aWNlSWQSLQoSZGV2aWNlX2ZpbmdlcnByaW50GAYgASgJUhFkZXZpY2VGaW5nZXJwcmludBIy'
    'CgVzdGF0ZRgHIAEoDjIcLmFueXR0eS5jbG91ZC52MS5EYWVtb25TdGF0ZVIFc3RhdGUSJQoOc3'
    'RhdGVfcmV2aXNpb24YCCABKARSDXN0YXRlUmV2aXNpb24SOQoKY3JlYXRlZF9hdBgJIAEoCzIa'
    'Lmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCWNyZWF0ZWRBdBI5Cgp1cGRhdGVkX2F0GAogAS'
    'gLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJdXBkYXRlZEF0EioKEXByZWZlcnJlZF9l'
    'ZGdlX2lkGAsgASgJUg9wcmVmZXJyZWRFZGdlSWQSOAoYZWRnZV9wcmVmZXJlbmNlX3JldmlzaW'
    '9uGAwgASgEUhZlZGdlUHJlZmVyZW5jZVJldmlzaW9uElcKGmVkZ2VfcHJlZmVyZW5jZV91cGRh'
    'dGVkX2F0GA0gASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIXZWRnZVByZWZlcmVuY2'
    'VVcGRhdGVkQXQ=');

@$core.Deprecated('Use daemonStateRecordDescriptor instead')
const DaemonStateRecord$json = {
  '1': 'DaemonStateRecord',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {
      '1': 'state',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.DaemonState',
      '10': 'state'
    },
    {'1': 'state_revision', '3': 3, '4': 1, '5': 4, '10': 'stateRevision'},
  ],
};

/// Descriptor for `DaemonStateRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateRecordDescriptor = $convert.base64Decode(
    'ChFEYWVtb25TdGF0ZVJlY29yZBIbCglkYWVtb25faWQYASABKAlSCGRhZW1vbklkEjIKBXN0YX'
    'RlGAIgASgOMhwuYW55dHR5LmNsb3VkLnYxLkRhZW1vblN0YXRlUgVzdGF0ZRIlCg5zdGF0ZV9y'
    'ZXZpc2lvbhgDIAEoBFINc3RhdGVSZXZpc2lvbg==');

@$core.Deprecated('Use daemonRuntimeProjectionDescriptor instead')
const DaemonRuntimeProjection$json = {
  '1': 'DaemonRuntimeProjection',
  '2': [
    {'1': 'online', '3': 1, '4': 1, '5': 8, '10': 'online'},
    {'1': 'edge_id', '3': 2, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'edge_name', '3': 3, '4': 1, '5': 9, '10': 'edgeName'},
    {'1': 'edge_region', '3': 4, '4': 1, '5': 9, '10': 'edgeRegion'},
    {'1': 'boot_id', '3': 5, '4': 1, '5': 9, '10': 'bootId'},
    {'1': 'connection_id', '3': 6, '4': 1, '5': 9, '10': 'connectionId'},
    {'1': 'generation', '3': 7, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'edge_public_endpoint',
      '3': 8,
      '4': 1,
      '5': 9,
      '10': 'edgePublicEndpoint'
    },
  ],
};

/// Descriptor for `DaemonRuntimeProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonRuntimeProjectionDescriptor = $convert.base64Decode(
    'ChdEYWVtb25SdW50aW1lUHJvamVjdGlvbhIWCgZvbmxpbmUYASABKAhSBm9ubGluZRIXCgdlZG'
    'dlX2lkGAIgASgJUgZlZGdlSWQSGwoJZWRnZV9uYW1lGAMgASgJUghlZGdlTmFtZRIfCgtlZGdl'
    'X3JlZ2lvbhgEIAEoCVIKZWRnZVJlZ2lvbhIXCgdib290X2lkGAUgASgJUgZib290SWQSIwoNY2'
    '9ubmVjdGlvbl9pZBgGIAEoCVIMY29ubmVjdGlvbklkEh4KCmdlbmVyYXRpb24YByABKARSCmdl'
    'bmVyYXRpb24SMAoUZWRnZV9wdWJsaWNfZW5kcG9pbnQYCCABKAlSEmVkZ2VQdWJsaWNFbmRwb2'
    'ludA==');

@$core.Deprecated('Use managedDaemonDescriptor instead')
const ManagedDaemon$json = {
  '1': 'ManagedDaemon',
  '2': [
    {
      '1': 'daemon',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonRecord',
      '10': 'daemon'
    },
    {
      '1': 'runtime',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonRuntimeProjection',
      '10': 'runtime'
    },
  ],
};

/// Descriptor for `ManagedDaemon`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List managedDaemonDescriptor = $convert.base64Decode(
    'Cg1NYW5hZ2VkRGFlbW9uEjUKBmRhZW1vbhgBIAEoCzIdLmFueXR0eS5jbG91ZC52MS5EYWVtb2'
    '5SZWNvcmRSBmRhZW1vbhJCCgdydW50aW1lGAIgASgLMiguYW55dHR5LmNsb3VkLnYxLkRhZW1v'
    'blJ1bnRpbWVQcm9qZWN0aW9uUgdydW50aW1l');

@$core.Deprecated('Use daemonEdgeMeasurementDescriptor instead')
const DaemonEdgeMeasurement$json = {
  '1': 'DaemonEdgeMeasurement',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'reachable', '3': 2, '4': 1, '5': 8, '10': 'reachable'},
    {
      '1': 'connect_latency_ms',
      '3': 3,
      '4': 1,
      '5': 13,
      '10': 'connectLatencyMs'
    },
    {
      '1': 'connection_failure_rate',
      '3': 4,
      '4': 1,
      '5': 1,
      '10': 'connectionFailureRate'
    },
    {'1': 'sample_count', '3': 5, '4': 1, '5': 13, '10': 'sampleCount'},
    {
      '1': 'measured_at',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'measuredAt'
    },
  ],
};

/// Descriptor for `DaemonEdgeMeasurement`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonEdgeMeasurementDescriptor = $convert.base64Decode(
    'ChVEYWVtb25FZGdlTWVhc3VyZW1lbnQSFwoHZWRnZV9pZBgBIAEoCVIGZWRnZUlkEhwKCXJlYW'
    'NoYWJsZRgCIAEoCFIJcmVhY2hhYmxlEiwKEmNvbm5lY3RfbGF0ZW5jeV9tcxgDIAEoDVIQY29u'
    'bmVjdExhdGVuY3lNcxI2Chdjb25uZWN0aW9uX2ZhaWx1cmVfcmF0ZRgEIAEoAVIVY29ubmVjdG'
    'lvbkZhaWx1cmVSYXRlEiEKDHNhbXBsZV9jb3VudBgFIAEoDVILc2FtcGxlQ291bnQSOwoLbWVh'
    'c3VyZWRfYXQYBiABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgptZWFzdXJlZEF0');

@$core.Deprecated('Use daemonEdgeCandidateDescriptor instead')
const DaemonEdgeCandidate$json = {
  '1': 'DaemonEdgeCandidate',
  '2': [
    {
      '1': 'locator',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeLocator',
      '10': 'locator'
    },
    {'1': 'online', '3': 2, '4': 1, '5': 8, '10': 'online'},
    {'1': 'eligible', '3': 3, '4': 1, '5': 8, '10': 'eligible'},
    {'1': 'preferred', '3': 6, '4': 1, '5': 8, '10': 'preferred'},
    {'1': 'current', '3': 7, '4': 1, '5': 8, '10': 'current'},
    {
      '1': 'measurement',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeMeasurement',
      '10': 'measurement'
    },
    {'1': 'score', '3': 9, '4': 1, '5': 1, '10': 'score'},
    {'1': 'status', '3': 10, '4': 1, '5': 9, '10': 'status'},
  ],
  '9': [
    {'1': 4, '2': 5},
    {'1': 5, '2': 6},
  ],
  '10': ['agent_count', 'capacity'],
};

/// Descriptor for `DaemonEdgeCandidate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonEdgeCandidateDescriptor = $convert.base64Decode(
    'ChNEYWVtb25FZGdlQ2FuZGlkYXRlEjYKB2xvY2F0b3IYASABKAsyHC5hbnl0dHkuY2xvdWQudj'
    'EuRWRnZUxvY2F0b3JSB2xvY2F0b3ISFgoGb25saW5lGAIgASgIUgZvbmxpbmUSGgoIZWxpZ2li'
    'bGUYAyABKAhSCGVsaWdpYmxlEhwKCXByZWZlcnJlZBgGIAEoCFIJcHJlZmVycmVkEhgKB2N1cn'
    'JlbnQYByABKAhSB2N1cnJlbnQSSAoLbWVhc3VyZW1lbnQYCCABKAsyJi5hbnl0dHkuY2xvdWQu'
    'djEuRGFlbW9uRWRnZU1lYXN1cmVtZW50UgttZWFzdXJlbWVudBIUCgVzY29yZRgJIAEoAVIFc2'
    'NvcmUSFgoGc3RhdHVzGAogASgJUgZzdGF0dXNKBAgEEAVKBAgFEAZSC2FnZW50X2NvdW50Ughj'
    'YXBhY2l0eQ==');

@$core.Deprecated('Use daemonEdgeSelectionDescriptor instead')
const DaemonEdgeSelection$json = {
  '1': 'DaemonEdgeSelection',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'preferred_edge_id', '3': 2, '4': 1, '5': 9, '10': 'preferredEdgeId'},
    {
      '1': 'preference_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'preferenceRevision'
    },
    {'1': 'current_edge_id', '3': 4, '4': 1, '5': 9, '10': 'currentEdgeId'},
    {'1': 'selected_edge_id', '3': 5, '4': 1, '5': 9, '10': 'selectedEdgeId'},
    {
      '1': 'candidates',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeCandidate',
      '10': 'candidates'
    },
    {
      '1': 'evaluated_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'evaluatedAt'
    },
  ],
};

/// Descriptor for `DaemonEdgeSelection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonEdgeSelectionDescriptor = $convert.base64Decode(
    'ChNEYWVtb25FZGdlU2VsZWN0aW9uEhsKCWRhZW1vbl9pZBgBIAEoCVIIZGFlbW9uSWQSKgoRcH'
    'JlZmVycmVkX2VkZ2VfaWQYAiABKAlSD3ByZWZlcnJlZEVkZ2VJZBIvChNwcmVmZXJlbmNlX3Jl'
    'dmlzaW9uGAMgASgEUhJwcmVmZXJlbmNlUmV2aXNpb24SJgoPY3VycmVudF9lZGdlX2lkGAQgAS'
    'gJUg1jdXJyZW50RWRnZUlkEigKEHNlbGVjdGVkX2VkZ2VfaWQYBSABKAlSDnNlbGVjdGVkRWRn'
    'ZUlkEkQKCmNhbmRpZGF0ZXMYBiADKAsyJC5hbnl0dHkuY2xvdWQudjEuRGFlbW9uRWRnZUNhbm'
    'RpZGF0ZVIKY2FuZGlkYXRlcxI9CgxldmFsdWF0ZWRfYXQYByABKAsyGi5nb29nbGUucHJvdG9i'
    'dWYuVGltZXN0YW1wUgtldmFsdWF0ZWRBdA==');

@$core.Deprecated('Use createDaemonEnrollmentRequestDescriptor instead')
const CreateDaemonEnrollmentRequest$json = {
  '1': 'CreateDaemonEnrollmentRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'account_name', '3': 2, '4': 1, '5': 9, '10': 'accountName'},
    {'1': 'daemon_name', '3': 3, '4': 1, '5': 9, '10': 'daemonName'},
  ],
};

/// Descriptor for `CreateDaemonEnrollmentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createDaemonEnrollmentRequestDescriptor =
    $convert.base64Decode(
        'Ch1DcmVhdGVEYWVtb25FbnJvbGxtZW50UmVxdWVzdBIdCgphY2NvdW50X2lkGAEgASgJUglhY2'
        'NvdW50SWQSIQoMYWNjb3VudF9uYW1lGAIgASgJUgthY2NvdW50TmFtZRIfCgtkYWVtb25fbmFt'
        'ZRgDIAEoCVIKZGFlbW9uTmFtZQ==');

@$core.Deprecated('Use createDaemonEnrollmentResponseDescriptor instead')
const CreateDaemonEnrollmentResponse$json = {
  '1': 'CreateDaemonEnrollmentResponse',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'enrollment_code', '3': 2, '4': 1, '5': 9, '10': 'enrollmentCode'},
    {
      '1': 'expires_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
    {'1': 'enroll_command', '3': 4, '4': 1, '5': 9, '10': 'enrollCommand'},
    {'1': 'daemon_count', '3': 5, '4': 1, '5': 13, '10': 'daemonCount'},
    {'1': 'daemon_limit', '3': 6, '4': 1, '5': 13, '10': 'daemonLimit'},
  ],
};

/// Descriptor for `CreateDaemonEnrollmentResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createDaemonEnrollmentResponseDescriptor = $convert.base64Decode(
    'Ch5DcmVhdGVEYWVtb25FbnJvbGxtZW50UmVzcG9uc2USHQoKYWNjb3VudF9pZBgBIAEoCVIJYW'
    'Njb3VudElkEicKD2Vucm9sbG1lbnRfY29kZRgCIAEoCVIOZW5yb2xsbWVudENvZGUSOQoKZXhw'
    'aXJlc19hdBgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCWV4cGlyZXNBdBIlCg'
    '5lbnJvbGxfY29tbWFuZBgEIAEoCVINZW5yb2xsQ29tbWFuZBIhCgxkYWVtb25fY291bnQYBSAB'
    'KA1SC2RhZW1vbkNvdW50EiEKDGRhZW1vbl9saW1pdBgGIAEoDVILZGFlbW9uTGltaXQ=');

@$core.Deprecated('Use listDaemonsRequestDescriptor instead')
const ListDaemonsRequest$json = {
  '1': 'ListDaemonsRequest',
  '2': [
    {'1': 'page_size', '3': 1, '4': 1, '5': 13, '10': 'pageSize'},
    {'1': 'cursor', '3': 2, '4': 1, '5': 9, '10': 'cursor'},
    {'1': 'query', '3': 3, '4': 1, '5': 9, '10': 'query'},
    {'1': 'edge_id', '3': 4, '4': 1, '5': 9, '10': 'edgeId'},
  ],
};

/// Descriptor for `ListDaemonsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listDaemonsRequestDescriptor = $convert.base64Decode(
    'ChJMaXN0RGFlbW9uc1JlcXVlc3QSGwoJcGFnZV9zaXplGAEgASgNUghwYWdlU2l6ZRIWCgZjdX'
    'Jzb3IYAiABKAlSBmN1cnNvchIUCgVxdWVyeRgDIAEoCVIFcXVlcnkSFwoHZWRnZV9pZBgEIAEo'
    'CVIGZWRnZUlk');

@$core.Deprecated('Use listDaemonsResponseDescriptor instead')
const ListDaemonsResponse$json = {
  '1': 'ListDaemonsResponse',
  '2': [
    {
      '1': 'daemons',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.ManagedDaemon',
      '10': 'daemons'
    },
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListDaemonsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listDaemonsResponseDescriptor = $convert.base64Decode(
    'ChNMaXN0RGFlbW9uc1Jlc3BvbnNlEjgKB2RhZW1vbnMYASADKAsyHi5hbnl0dHkuY2xvdWQudj'
    'EuTWFuYWdlZERhZW1vblIHZGFlbW9ucxIfCgtuZXh0X2N1cnNvchgCIAEoCVIKbmV4dEN1cnNv'
    'cg==');

@$core.Deprecated('Use createMyDaemonEnrollmentRequestDescriptor instead')
const CreateMyDaemonEnrollmentRequest$json = {
  '1': 'CreateMyDaemonEnrollmentRequest',
  '2': [
    {'1': 'daemon_name', '3': 1, '4': 1, '5': 9, '10': 'daemonName'},
  ],
};

/// Descriptor for `CreateMyDaemonEnrollmentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createMyDaemonEnrollmentRequestDescriptor =
    $convert.base64Decode(
        'Ch9DcmVhdGVNeURhZW1vbkVucm9sbG1lbnRSZXF1ZXN0Eh8KC2RhZW1vbl9uYW1lGAEgASgJUg'
        'pkYWVtb25OYW1l');

@$core.Deprecated('Use listMyDaemonsRequestDescriptor instead')
const ListMyDaemonsRequest$json = {
  '1': 'ListMyDaemonsRequest',
};

/// Descriptor for `ListMyDaemonsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listMyDaemonsRequestDescriptor =
    $convert.base64Decode('ChRMaXN0TXlEYWVtb25zUmVxdWVzdA==');

@$core.Deprecated('Use listMyDaemonsResponseDescriptor instead')
const ListMyDaemonsResponse$json = {
  '1': 'ListMyDaemonsResponse',
  '2': [
    {
      '1': 'daemons',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.ManagedDaemon',
      '10': 'daemons'
    },
    {'1': 'daemon_count', '3': 2, '4': 1, '5': 13, '10': 'daemonCount'},
    {'1': 'daemon_limit', '3': 3, '4': 1, '5': 13, '10': 'daemonLimit'},
  ],
};

/// Descriptor for `ListMyDaemonsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listMyDaemonsResponseDescriptor = $convert.base64Decode(
    'ChVMaXN0TXlEYWVtb25zUmVzcG9uc2USOAoHZGFlbW9ucxgBIAMoCzIeLmFueXR0eS5jbG91ZC'
    '52MS5NYW5hZ2VkRGFlbW9uUgdkYWVtb25zEiEKDGRhZW1vbl9jb3VudBgCIAEoDVILZGFlbW9u'
    'Q291bnQSIQoMZGFlbW9uX2xpbWl0GAMgASgNUgtkYWVtb25MaW1pdA==');

@$core.Deprecated('Use changeMyDaemonStateRequestDescriptor instead')
const ChangeMyDaemonStateRequest$json = {
  '1': 'ChangeMyDaemonStateRequest',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {
      '1': 'target_state',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.DaemonState',
      '10': 'targetState'
    },
    {
      '1': 'expected_state_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'expectedStateRevision'
    },
    {'1': 'reason', '3': 4, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `ChangeMyDaemonStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeMyDaemonStateRequestDescriptor = $convert.base64Decode(
    'ChpDaGFuZ2VNeURhZW1vblN0YXRlUmVxdWVzdBIbCglkYWVtb25faWQYASABKAlSCGRhZW1vbk'
    'lkEj8KDHRhcmdldF9zdGF0ZRgCIAEoDjIcLmFueXR0eS5jbG91ZC52MS5EYWVtb25TdGF0ZVIL'
    'dGFyZ2V0U3RhdGUSNgoXZXhwZWN0ZWRfc3RhdGVfcmV2aXNpb24YAyABKARSFWV4cGVjdGVkU3'
    'RhdGVSZXZpc2lvbhIWCgZyZWFzb24YBCABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use changeMyDaemonStateResponseDescriptor instead')
const ChangeMyDaemonStateResponse$json = {
  '1': 'ChangeMyDaemonStateResponse',
  '2': [
    {
      '1': 'daemon',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonRecord',
      '10': 'daemon'
    },
  ],
};

/// Descriptor for `ChangeMyDaemonStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeMyDaemonStateResponseDescriptor =
    $convert.base64Decode(
        'ChtDaGFuZ2VNeURhZW1vblN0YXRlUmVzcG9uc2USNQoGZGFlbW9uGAEgASgLMh0uYW55dHR5Lm'
        'Nsb3VkLnYxLkRhZW1vblJlY29yZFIGZGFlbW9u');

@$core.Deprecated('Use listMyDaemonEdgesRequestDescriptor instead')
const ListMyDaemonEdgesRequest$json = {
  '1': 'ListMyDaemonEdgesRequest',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
  ],
};

/// Descriptor for `ListMyDaemonEdgesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listMyDaemonEdgesRequestDescriptor =
    $convert.base64Decode(
        'ChhMaXN0TXlEYWVtb25FZGdlc1JlcXVlc3QSGwoJZGFlbW9uX2lkGAEgASgJUghkYWVtb25JZA'
        '==');

@$core.Deprecated('Use listMyDaemonEdgesResponseDescriptor instead')
const ListMyDaemonEdgesResponse$json = {
  '1': 'ListMyDaemonEdgesResponse',
  '2': [
    {
      '1': 'selection',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeSelection',
      '10': 'selection'
    },
  ],
};

/// Descriptor for `ListMyDaemonEdgesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listMyDaemonEdgesResponseDescriptor =
    $convert.base64Decode(
        'ChlMaXN0TXlEYWVtb25FZGdlc1Jlc3BvbnNlEkIKCXNlbGVjdGlvbhgBIAEoCzIkLmFueXR0eS'
        '5jbG91ZC52MS5EYWVtb25FZGdlU2VsZWN0aW9uUglzZWxlY3Rpb24=');

@$core.Deprecated('Use changeMyDaemonEdgePreferenceRequestDescriptor instead')
const ChangeMyDaemonEdgePreferenceRequest$json = {
  '1': 'ChangeMyDaemonEdgePreferenceRequest',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'preferred_edge_id', '3': 2, '4': 1, '5': 9, '10': 'preferredEdgeId'},
    {
      '1': 'expected_preference_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'expectedPreferenceRevision'
    },
    {'1': 'reselect_now', '3': 4, '4': 1, '5': 8, '10': 'reselectNow'},
  ],
};

/// Descriptor for `ChangeMyDaemonEdgePreferenceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeMyDaemonEdgePreferenceRequestDescriptor =
    $convert.base64Decode(
        'CiNDaGFuZ2VNeURhZW1vbkVkZ2VQcmVmZXJlbmNlUmVxdWVzdBIbCglkYWVtb25faWQYASABKA'
        'lSCGRhZW1vbklkEioKEXByZWZlcnJlZF9lZGdlX2lkGAIgASgJUg9wcmVmZXJyZWRFZGdlSWQS'
        'QAocZXhwZWN0ZWRfcHJlZmVyZW5jZV9yZXZpc2lvbhgDIAEoBFIaZXhwZWN0ZWRQcmVmZXJlbm'
        'NlUmV2aXNpb24SIQoMcmVzZWxlY3Rfbm93GAQgASgIUgtyZXNlbGVjdE5vdw==');

@$core.Deprecated('Use changeMyDaemonEdgePreferenceResponseDescriptor instead')
const ChangeMyDaemonEdgePreferenceResponse$json = {
  '1': 'ChangeMyDaemonEdgePreferenceResponse',
  '2': [
    {
      '1': 'selection',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeSelection',
      '10': 'selection'
    },
    {
      '1': 'reselect_accepted',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'reselectAccepted'
    },
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ChangeMyDaemonEdgePreferenceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeMyDaemonEdgePreferenceResponseDescriptor =
    $convert.base64Decode(
        'CiRDaGFuZ2VNeURhZW1vbkVkZ2VQcmVmZXJlbmNlUmVzcG9uc2USQgoJc2VsZWN0aW9uGAEgAS'
        'gLMiQuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkVkZ2VTZWxlY3Rpb25SCXNlbGVjdGlvbhIrChFy'
        'ZXNlbGVjdF9hY2NlcHRlZBgCIAEoCFIQcmVzZWxlY3RBY2NlcHRlZBIYCgdtZXNzYWdlGAMgAS'
        'gJUgdtZXNzYWdl');

@$core.Deprecated('Use reselectMyDaemonEdgeRequestDescriptor instead')
const ReselectMyDaemonEdgeRequest$json = {
  '1': 'ReselectMyDaemonEdgeRequest',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
  ],
};

/// Descriptor for `ReselectMyDaemonEdgeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List reselectMyDaemonEdgeRequestDescriptor =
    $convert.base64Decode(
        'ChtSZXNlbGVjdE15RGFlbW9uRWRnZVJlcXVlc3QSGwoJZGFlbW9uX2lkGAEgASgJUghkYWVtb2'
        '5JZA==');

@$core.Deprecated('Use reselectMyDaemonEdgeResponseDescriptor instead')
const ReselectMyDaemonEdgeResponse$json = {
  '1': 'ReselectMyDaemonEdgeResponse',
  '2': [
    {
      '1': 'selection',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeSelection',
      '10': 'selection'
    },
    {
      '1': 'reselect_accepted',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'reselectAccepted'
    },
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ReselectMyDaemonEdgeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List reselectMyDaemonEdgeResponseDescriptor = $convert.base64Decode(
    'ChxSZXNlbGVjdE15RGFlbW9uRWRnZVJlc3BvbnNlEkIKCXNlbGVjdGlvbhgBIAEoCzIkLmFueX'
    'R0eS5jbG91ZC52MS5EYWVtb25FZGdlU2VsZWN0aW9uUglzZWxlY3Rpb24SKwoRcmVzZWxlY3Rf'
    'YWNjZXB0ZWQYAiABKAhSEHJlc2VsZWN0QWNjZXB0ZWQSGAoHbWVzc2FnZRgDIAEoCVIHbWVzc2'
    'FnZQ==');

@$core.Deprecated('Use beginDaemonEnrollmentRequestDescriptor instead')
const BeginDaemonEnrollmentRequest$json = {
  '1': 'BeginDaemonEnrollmentRequest',
  '2': [
    {'1': 'enrollment_code', '3': 1, '4': 1, '5': 9, '10': 'enrollmentCode'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {
      '1': 'device_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
  ],
};

/// Descriptor for `BeginDaemonEnrollmentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List beginDaemonEnrollmentRequestDescriptor = $convert.base64Decode(
    'ChxCZWdpbkRhZW1vbkVucm9sbG1lbnRSZXF1ZXN0EicKD2Vucm9sbG1lbnRfY29kZRgBIAEoCV'
    'IOZW5yb2xsbWVudENvZGUSGwoJZGV2aWNlX2lkGAIgASgJUghkZXZpY2VJZBItChJkZXZpY2Vf'
    'ZmluZ2VycHJpbnQYAyABKAlSEWRldmljZUZpbmdlcnByaW50EioKEWRldmljZV9wdWJsaWNfa2'
    'V5GAQgASgMUg9kZXZpY2VQdWJsaWNLZXk=');

@$core.Deprecated('Use identityChallengeDescriptor instead')
const IdentityChallenge$json = {
  '1': 'IdentityChallenge',
  '2': [
    {'1': 'challenge_id', '3': 1, '4': 1, '5': 9, '10': 'challengeId'},
    {'1': 'challenge', '3': 2, '4': 1, '5': 12, '10': 'challenge'},
    {
      '1': 'expires_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
  ],
};

/// Descriptor for `IdentityChallenge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List identityChallengeDescriptor = $convert.base64Decode(
    'ChFJZGVudGl0eUNoYWxsZW5nZRIhCgxjaGFsbGVuZ2VfaWQYASABKAlSC2NoYWxsZW5nZUlkEh'
    'wKCWNoYWxsZW5nZRgCIAEoDFIJY2hhbGxlbmdlEjkKCmV4cGlyZXNfYXQYAyABKAsyGi5nb29n'
    'bGUucHJvdG9idWYuVGltZXN0YW1wUglleHBpcmVzQXQ=');

@$core.Deprecated('Use daemonEnrollmentChallengeDescriptor instead')
const DaemonEnrollmentChallenge$json = {
  '1': 'DaemonEnrollmentChallenge',
  '2': [
    {
      '1': 'identity_challenge',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.IdentityChallenge',
      '10': 'identityChallenge'
    },
    {
      '1': 'edge_candidates',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeCandidate',
      '10': 'edgeCandidates'
    },
  ],
};

/// Descriptor for `DaemonEnrollmentChallenge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonEnrollmentChallengeDescriptor = $convert.base64Decode(
    'ChlEYWVtb25FbnJvbGxtZW50Q2hhbGxlbmdlElEKEmlkZW50aXR5X2NoYWxsZW5nZRgBIAEoCz'
    'IiLmFueXR0eS5jbG91ZC52MS5JZGVudGl0eUNoYWxsZW5nZVIRaWRlbnRpdHlDaGFsbGVuZ2US'
    'TQoPZWRnZV9jYW5kaWRhdGVzGAIgAygLMiQuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkVkZ2VDYW'
    '5kaWRhdGVSDmVkZ2VDYW5kaWRhdGVz');

@$core.Deprecated('Use completeDaemonEnrollmentRequestDescriptor instead')
const CompleteDaemonEnrollmentRequest$json = {
  '1': 'CompleteDaemonEnrollmentRequest',
  '2': [
    {'1': 'challenge_id', '3': 1, '4': 1, '5': 9, '10': 'challengeId'},
    {'1': 'device_proof', '3': 2, '4': 1, '5': 12, '10': 'deviceProof'},
    {
      '1': 'edge_measurements',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeMeasurement',
      '10': 'edgeMeasurements'
    },
  ],
};

/// Descriptor for `CompleteDaemonEnrollmentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List completeDaemonEnrollmentRequestDescriptor =
    $convert.base64Decode(
        'Ch9Db21wbGV0ZURhZW1vbkVucm9sbG1lbnRSZXF1ZXN0EiEKDGNoYWxsZW5nZV9pZBgBIAEoCV'
        'ILY2hhbGxlbmdlSWQSIQoMZGV2aWNlX3Byb29mGAIgASgMUgtkZXZpY2VQcm9vZhJTChFlZGdl'
        'X21lYXN1cmVtZW50cxgDIAMoCzImLmFueXR0eS5jbG91ZC52MS5EYWVtb25FZGdlTWVhc3VyZW'
        '1lbnRSEGVkZ2VNZWFzdXJlbWVudHM=');

@$core.Deprecated('Use edgeLocatorDescriptor instead')
const EdgeLocator$json = {
  '1': 'EdgeLocator',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'region', '3': 3, '4': 1, '5': 9, '10': 'region'},
    {'1': 'public_endpoint', '3': 4, '4': 1, '5': 9, '10': 'publicEndpoint'},
    {'1': 'server_name', '3': 5, '4': 1, '5': 9, '10': 'serverName'},
    {
      '1': 'ca_certificate_pem',
      '3': 6,
      '4': 1,
      '5': 12,
      '10': 'caCertificatePem'
    },
    {'1': 'revision', '3': 7, '4': 1, '5': 4, '10': 'revision'},
  ],
};

/// Descriptor for `EdgeLocator`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeLocatorDescriptor = $convert.base64Decode(
    'CgtFZGdlTG9jYXRvchIXCgdlZGdlX2lkGAEgASgJUgZlZGdlSWQSEgoEbmFtZRgCIAEoCVIEbm'
    'FtZRIWCgZyZWdpb24YAyABKAlSBnJlZ2lvbhInCg9wdWJsaWNfZW5kcG9pbnQYBCABKAlSDnB1'
    'YmxpY0VuZHBvaW50Eh8KC3NlcnZlcl9uYW1lGAUgASgJUgpzZXJ2ZXJOYW1lEiwKEmNhX2Nlcn'
    'RpZmljYXRlX3BlbRgGIAEoDFIQY2FDZXJ0aWZpY2F0ZVBlbRIaCghyZXZpc2lvbhgHIAEoBFII'
    'cmV2aXNpb24=');

@$core.Deprecated('Use completeDaemonEnrollmentResponseDescriptor instead')
const CompleteDaemonEnrollmentResponse$json = {
  '1': 'CompleteDaemonEnrollmentResponse',
  '2': [
    {
      '1': 'daemon',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonRecord',
      '10': 'daemon'
    },
    {
      '1': 'daemon_binding',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEnvelope',
      '10': 'daemonBinding'
    },
    {
      '1': 'edge_locator',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeLocator',
      '10': 'edgeLocator'
    },
    {'1': 'daemon_count', '3': 4, '4': 1, '5': 13, '10': 'daemonCount'},
    {'1': 'daemon_limit', '3': 5, '4': 1, '5': 13, '10': 'daemonLimit'},
    {
      '1': 'edge_selection',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeSelection',
      '10': 'edgeSelection'
    },
  ],
};

/// Descriptor for `CompleteDaemonEnrollmentResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List completeDaemonEnrollmentResponseDescriptor = $convert.base64Decode(
    'CiBDb21wbGV0ZURhZW1vbkVucm9sbG1lbnRSZXNwb25zZRI1CgZkYWVtb24YASABKAsyHS5hbn'
    'l0dHkuY2xvdWQudjEuRGFlbW9uUmVjb3JkUgZkYWVtb24SRgoOZGFlbW9uX2JpbmRpbmcYAiAB'
    'KAsyHy5hbnl0dHkuY2xvdWQudjEuU2lnbmVkRW52ZWxvcGVSDWRhZW1vbkJpbmRpbmcSPwoMZW'
    'RnZV9sb2NhdG9yGAMgASgLMhwuYW55dHR5LmNsb3VkLnYxLkVkZ2VMb2NhdG9yUgtlZGdlTG9j'
    'YXRvchIhCgxkYWVtb25fY291bnQYBCABKA1SC2RhZW1vbkNvdW50EiEKDGRhZW1vbl9saW1pdB'
    'gFIAEoDVILZGFlbW9uTGltaXQSSwoOZWRnZV9zZWxlY3Rpb24YBiABKAsyJC5hbnl0dHkuY2xv'
    'dWQudjEuRGFlbW9uRWRnZVNlbGVjdGlvblINZWRnZVNlbGVjdGlvbg==');

@$core.Deprecated('Use beginDaemonBindingRefreshRequestDescriptor instead')
const BeginDaemonBindingRefreshRequest$json = {
  '1': 'BeginDaemonBindingRefreshRequest',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
  ],
};

/// Descriptor for `BeginDaemonBindingRefreshRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List beginDaemonBindingRefreshRequestDescriptor =
    $convert.base64Decode(
        'CiBCZWdpbkRhZW1vbkJpbmRpbmdSZWZyZXNoUmVxdWVzdBIbCglkYWVtb25faWQYASABKAlSCG'
        'RhZW1vbklk');

@$core.Deprecated('Use completeDaemonBindingRefreshRequestDescriptor instead')
const CompleteDaemonBindingRefreshRequest$json = {
  '1': 'CompleteDaemonBindingRefreshRequest',
  '2': [
    {'1': 'challenge_id', '3': 1, '4': 1, '5': 9, '10': 'challengeId'},
    {'1': 'device_proof', '3': 2, '4': 1, '5': 12, '10': 'deviceProof'},
    {
      '1': 'edge_measurements',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeMeasurement',
      '10': 'edgeMeasurements'
    },
    {
      '1': 'change_preference',
      '3': 4,
      '4': 1,
      '5': 8,
      '10': 'changePreference'
    },
    {'1': 'preferred_edge_id', '3': 5, '4': 1, '5': 9, '10': 'preferredEdgeId'},
    {
      '1': 'expected_preference_revision',
      '3': 6,
      '4': 1,
      '5': 4,
      '10': 'expectedPreferenceRevision'
    },
  ],
};

/// Descriptor for `CompleteDaemonBindingRefreshRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List completeDaemonBindingRefreshRequestDescriptor = $convert.base64Decode(
    'CiNDb21wbGV0ZURhZW1vbkJpbmRpbmdSZWZyZXNoUmVxdWVzdBIhCgxjaGFsbGVuZ2VfaWQYAS'
    'ABKAlSC2NoYWxsZW5nZUlkEiEKDGRldmljZV9wcm9vZhgCIAEoDFILZGV2aWNlUHJvb2YSUwoR'
    'ZWRnZV9tZWFzdXJlbWVudHMYAyADKAsyJi5hbnl0dHkuY2xvdWQudjEuRGFlbW9uRWRnZU1lYX'
    'N1cmVtZW50UhBlZGdlTWVhc3VyZW1lbnRzEisKEWNoYW5nZV9wcmVmZXJlbmNlGAQgASgIUhBj'
    'aGFuZ2VQcmVmZXJlbmNlEioKEXByZWZlcnJlZF9lZGdlX2lkGAUgASgJUg9wcmVmZXJyZWRFZG'
    'dlSWQSQAocZXhwZWN0ZWRfcHJlZmVyZW5jZV9yZXZpc2lvbhgGIAEoBFIaZXhwZWN0ZWRQcmVm'
    'ZXJlbmNlUmV2aXNpb24=');

@$core.Deprecated('Use refreshDaemonBindingResponseDescriptor instead')
const RefreshDaemonBindingResponse$json = {
  '1': 'RefreshDaemonBindingResponse',
  '2': [
    {
      '1': 'daemon',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonRecord',
      '10': 'daemon'
    },
    {
      '1': 'daemon_binding',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEnvelope',
      '10': 'daemonBinding'
    },
    {
      '1': 'edge_locator',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeLocator',
      '10': 'edgeLocator'
    },
    {
      '1': 'edge_selection',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeSelection',
      '10': 'edgeSelection'
    },
  ],
};

/// Descriptor for `RefreshDaemonBindingResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List refreshDaemonBindingResponseDescriptor = $convert.base64Decode(
    'ChxSZWZyZXNoRGFlbW9uQmluZGluZ1Jlc3BvbnNlEjUKBmRhZW1vbhgBIAEoCzIdLmFueXR0eS'
    '5jbG91ZC52MS5EYWVtb25SZWNvcmRSBmRhZW1vbhJGCg5kYWVtb25fYmluZGluZxgCIAEoCzIf'
    'LmFueXR0eS5jbG91ZC52MS5TaWduZWRFbnZlbG9wZVINZGFlbW9uQmluZGluZxI/CgxlZGdlX2'
    'xvY2F0b3IYAyABKAsyHC5hbnl0dHkuY2xvdWQudjEuRWRnZUxvY2F0b3JSC2VkZ2VMb2NhdG9y'
    'EksKDmVkZ2Vfc2VsZWN0aW9uGAQgASgLMiQuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkVkZ2VTZW'
    'xlY3Rpb25SDWVkZ2VTZWxlY3Rpb24=');

const $core.Map<$core.String, $core.dynamic> EnrollmentServiceBase$json = {
  '1': 'EnrollmentService',
  '2': [
    {
      '1': 'BeginDaemonEnrollment',
      '2': '.anytty.cloud.v1.BeginDaemonEnrollmentRequest',
      '3': '.anytty.cloud.v1.DaemonEnrollmentChallenge'
    },
    {
      '1': 'CompleteDaemonEnrollment',
      '2': '.anytty.cloud.v1.CompleteDaemonEnrollmentRequest',
      '3': '.anytty.cloud.v1.CompleteDaemonEnrollmentResponse'
    },
    {
      '1': 'BeginDaemonBindingRefresh',
      '2': '.anytty.cloud.v1.BeginDaemonBindingRefreshRequest',
      '3': '.anytty.cloud.v1.IdentityChallenge'
    },
    {
      '1': 'CompleteDaemonBindingRefresh',
      '2': '.anytty.cloud.v1.CompleteDaemonBindingRefreshRequest',
      '3': '.anytty.cloud.v1.RefreshDaemonBindingResponse'
    },
  ],
};

@$core.Deprecated('Use enrollmentServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    EnrollmentServiceBase$messageJson = {
  '.anytty.cloud.v1.BeginDaemonEnrollmentRequest':
      BeginDaemonEnrollmentRequest$json,
  '.anytty.cloud.v1.DaemonEnrollmentChallenge': DaemonEnrollmentChallenge$json,
  '.anytty.cloud.v1.IdentityChallenge': IdentityChallenge$json,
  '.google.protobuf.Timestamp': $0.Timestamp$json,
  '.anytty.cloud.v1.DaemonEdgeCandidate': DaemonEdgeCandidate$json,
  '.anytty.cloud.v1.EdgeLocator': EdgeLocator$json,
  '.anytty.cloud.v1.DaemonEdgeMeasurement': DaemonEdgeMeasurement$json,
  '.anytty.cloud.v1.CompleteDaemonEnrollmentRequest':
      CompleteDaemonEnrollmentRequest$json,
  '.anytty.cloud.v1.CompleteDaemonEnrollmentResponse':
      CompleteDaemonEnrollmentResponse$json,
  '.anytty.cloud.v1.DaemonRecord': DaemonRecord$json,
  '.anytty.cloud.v1.SignedEnvelope': $1.SignedEnvelope$json,
  '.anytty.cloud.v1.DaemonEdgeSelection': DaemonEdgeSelection$json,
  '.anytty.cloud.v1.BeginDaemonBindingRefreshRequest':
      BeginDaemonBindingRefreshRequest$json,
  '.anytty.cloud.v1.CompleteDaemonBindingRefreshRequest':
      CompleteDaemonBindingRefreshRequest$json,
  '.anytty.cloud.v1.RefreshDaemonBindingResponse':
      RefreshDaemonBindingResponse$json,
};

/// Descriptor for `EnrollmentService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List enrollmentServiceDescriptor = $convert.base64Decode(
    'ChFFbnJvbGxtZW50U2VydmljZRJyChVCZWdpbkRhZW1vbkVucm9sbG1lbnQSLS5hbnl0dHkuY2'
    'xvdWQudjEuQmVnaW5EYWVtb25FbnJvbGxtZW50UmVxdWVzdBoqLmFueXR0eS5jbG91ZC52MS5E'
    'YWVtb25FbnJvbGxtZW50Q2hhbGxlbmdlEn8KGENvbXBsZXRlRGFlbW9uRW5yb2xsbWVudBIwLm'
    'FueXR0eS5jbG91ZC52MS5Db21wbGV0ZURhZW1vbkVucm9sbG1lbnRSZXF1ZXN0GjEuYW55dHR5'
    'LmNsb3VkLnYxLkNvbXBsZXRlRGFlbW9uRW5yb2xsbWVudFJlc3BvbnNlEnIKGUJlZ2luRGFlbW'
    '9uQmluZGluZ1JlZnJlc2gSMS5hbnl0dHkuY2xvdWQudjEuQmVnaW5EYWVtb25CaW5kaW5nUmVm'
    'cmVzaFJlcXVlc3QaIi5hbnl0dHkuY2xvdWQudjEuSWRlbnRpdHlDaGFsbGVuZ2USgwEKHENvbX'
    'BsZXRlRGFlbW9uQmluZGluZ1JlZnJlc2gSNC5hbnl0dHkuY2xvdWQudjEuQ29tcGxldGVEYWVt'
    'b25CaW5kaW5nUmVmcmVzaFJlcXVlc3QaLS5hbnl0dHkuY2xvdWQudjEuUmVmcmVzaERhZW1vbk'
    'JpbmRpbmdSZXNwb25zZQ==');

const $core.Map<$core.String, $core.dynamic> DaemonManagementServiceBase$json =
    {
  '1': 'DaemonManagementService',
  '2': [
    {
      '1': 'CreateMyEnrollment',
      '2': '.anytty.cloud.v1.CreateMyDaemonEnrollmentRequest',
      '3': '.anytty.cloud.v1.CreateDaemonEnrollmentResponse'
    },
    {
      '1': 'ListMyDaemons',
      '2': '.anytty.cloud.v1.ListMyDaemonsRequest',
      '3': '.anytty.cloud.v1.ListMyDaemonsResponse'
    },
    {
      '1': 'ChangeMyDaemonState',
      '2': '.anytty.cloud.v1.ChangeMyDaemonStateRequest',
      '3': '.anytty.cloud.v1.ChangeMyDaemonStateResponse'
    },
    {
      '1': 'ListMyDaemonEdges',
      '2': '.anytty.cloud.v1.ListMyDaemonEdgesRequest',
      '3': '.anytty.cloud.v1.ListMyDaemonEdgesResponse'
    },
    {
      '1': 'ChangeMyDaemonEdgePreference',
      '2': '.anytty.cloud.v1.ChangeMyDaemonEdgePreferenceRequest',
      '3': '.anytty.cloud.v1.ChangeMyDaemonEdgePreferenceResponse'
    },
    {
      '1': 'ReselectMyDaemonEdge',
      '2': '.anytty.cloud.v1.ReselectMyDaemonEdgeRequest',
      '3': '.anytty.cloud.v1.ReselectMyDaemonEdgeResponse'
    },
  ],
};

@$core.Deprecated('Use daemonManagementServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    DaemonManagementServiceBase$messageJson = {
  '.anytty.cloud.v1.CreateMyDaemonEnrollmentRequest':
      CreateMyDaemonEnrollmentRequest$json,
  '.anytty.cloud.v1.CreateDaemonEnrollmentResponse':
      CreateDaemonEnrollmentResponse$json,
  '.google.protobuf.Timestamp': $0.Timestamp$json,
  '.anytty.cloud.v1.ListMyDaemonsRequest': ListMyDaemonsRequest$json,
  '.anytty.cloud.v1.ListMyDaemonsResponse': ListMyDaemonsResponse$json,
  '.anytty.cloud.v1.ManagedDaemon': ManagedDaemon$json,
  '.anytty.cloud.v1.DaemonRecord': DaemonRecord$json,
  '.anytty.cloud.v1.DaemonRuntimeProjection': DaemonRuntimeProjection$json,
  '.anytty.cloud.v1.ChangeMyDaemonStateRequest':
      ChangeMyDaemonStateRequest$json,
  '.anytty.cloud.v1.ChangeMyDaemonStateResponse':
      ChangeMyDaemonStateResponse$json,
  '.anytty.cloud.v1.ListMyDaemonEdgesRequest': ListMyDaemonEdgesRequest$json,
  '.anytty.cloud.v1.ListMyDaemonEdgesResponse': ListMyDaemonEdgesResponse$json,
  '.anytty.cloud.v1.DaemonEdgeSelection': DaemonEdgeSelection$json,
  '.anytty.cloud.v1.DaemonEdgeCandidate': DaemonEdgeCandidate$json,
  '.anytty.cloud.v1.EdgeLocator': EdgeLocator$json,
  '.anytty.cloud.v1.DaemonEdgeMeasurement': DaemonEdgeMeasurement$json,
  '.anytty.cloud.v1.ChangeMyDaemonEdgePreferenceRequest':
      ChangeMyDaemonEdgePreferenceRequest$json,
  '.anytty.cloud.v1.ChangeMyDaemonEdgePreferenceResponse':
      ChangeMyDaemonEdgePreferenceResponse$json,
  '.anytty.cloud.v1.ReselectMyDaemonEdgeRequest':
      ReselectMyDaemonEdgeRequest$json,
  '.anytty.cloud.v1.ReselectMyDaemonEdgeResponse':
      ReselectMyDaemonEdgeResponse$json,
};

/// Descriptor for `DaemonManagementService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List daemonManagementServiceDescriptor = $convert.base64Decode(
    'ChdEYWVtb25NYW5hZ2VtZW50U2VydmljZRJ3ChJDcmVhdGVNeUVucm9sbG1lbnQSMC5hbnl0dH'
    'kuY2xvdWQudjEuQ3JlYXRlTXlEYWVtb25FbnJvbGxtZW50UmVxdWVzdBovLmFueXR0eS5jbG91'
    'ZC52MS5DcmVhdGVEYWVtb25FbnJvbGxtZW50UmVzcG9uc2USXgoNTGlzdE15RGFlbW9ucxIlLm'
    'FueXR0eS5jbG91ZC52MS5MaXN0TXlEYWVtb25zUmVxdWVzdBomLmFueXR0eS5jbG91ZC52MS5M'
    'aXN0TXlEYWVtb25zUmVzcG9uc2UScAoTQ2hhbmdlTXlEYWVtb25TdGF0ZRIrLmFueXR0eS5jbG'
    '91ZC52MS5DaGFuZ2VNeURhZW1vblN0YXRlUmVxdWVzdBosLmFueXR0eS5jbG91ZC52MS5DaGFu'
    'Z2VNeURhZW1vblN0YXRlUmVzcG9uc2USagoRTGlzdE15RGFlbW9uRWRnZXMSKS5hbnl0dHkuY2'
    'xvdWQudjEuTGlzdE15RGFlbW9uRWRnZXNSZXF1ZXN0GiouYW55dHR5LmNsb3VkLnYxLkxpc3RN'
    'eURhZW1vbkVkZ2VzUmVzcG9uc2USiwEKHENoYW5nZU15RGFlbW9uRWRnZVByZWZlcmVuY2USNC'
    '5hbnl0dHkuY2xvdWQudjEuQ2hhbmdlTXlEYWVtb25FZGdlUHJlZmVyZW5jZVJlcXVlc3QaNS5h'
    'bnl0dHkuY2xvdWQudjEuQ2hhbmdlTXlEYWVtb25FZGdlUHJlZmVyZW5jZVJlc3BvbnNlEnMKFF'
    'Jlc2VsZWN0TXlEYWVtb25FZGdlEiwuYW55dHR5LmNsb3VkLnYxLlJlc2VsZWN0TXlEYWVtb25F'
    'ZGdlUmVxdWVzdBotLmFueXR0eS5jbG91ZC52MS5SZXNlbGVjdE15RGFlbW9uRWRnZVJlc3Bvbn'
    'Nl');
