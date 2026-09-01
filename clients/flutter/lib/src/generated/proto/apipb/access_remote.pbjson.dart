// This is a generated file - do not edit.
//
// Generated from apipb/access_remote.proto.

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

@$core.Deprecated('Use clientAccessIdentityCommandDescriptor instead')
const ClientAccessIdentityCommand$json = {
  '1': 'ClientAccessIdentityCommand',
  '2': [
    {'1': 'challenge', '3': 2, '4': 1, '5': 12, '10': 'challenge'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `ClientAccessIdentityCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessIdentityCommandDescriptor =
    $convert.base64Decode(
        'ChtDbGllbnRBY2Nlc3NJZGVudGl0eUNvbW1hbmQSHAoJY2hhbGxlbmdlGAIgASgMUgljaGFsbG'
        'VuZ2VKBAgBEAI=');

@$core.Deprecated('Use clientAccessListCommandDescriptor instead')
const ClientAccessListCommand$json = {
  '1': 'ClientAccessListCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `ClientAccessListCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessListCommandDescriptor =
    $convert.base64Decode('ChdDbGllbnRBY2Nlc3NMaXN0Q29tbWFuZEoECAEQAg==');

@$core.Deprecated('Use clientAccessTicketCreateCommandDescriptor instead')
const ClientAccessTicketCreateCommand$json = {
  '1': 'ClientAccessTicketCreateCommand',
  '2': [
    {
      '1': 'request',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessTicketCreateRequest',
      '10': 'request'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `ClientAccessTicketCreateCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessTicketCreateCommandDescriptor =
    $convert.base64Decode(
        'Ch9DbGllbnRBY2Nlc3NUaWNrZXRDcmVhdGVDb21tYW5kElAKB3JlcXVlc3QYAiABKAsyNi5hbn'
        'l0dHkucmVtb3RlLmF1dGgudjEuQ2xpZW50QWNjZXNzVGlja2V0Q3JlYXRlUmVxdWVzdFIHcmVx'
        'dWVzdEoECAEQAg==');

@$core.Deprecated('Use clientAccessRevokeCommandDescriptor instead')
const ClientAccessRevokeCommand$json = {
  '1': 'ClientAccessRevokeCommand',
  '2': [
    {
      '1': 'request',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessRevokeRequest',
      '10': 'request'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `ClientAccessRevokeCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessRevokeCommandDescriptor = $convert.base64Decode(
    'ChlDbGllbnRBY2Nlc3NSZXZva2VDb21tYW5kEkoKB3JlcXVlc3QYAiABKAsyMC5hbnl0dHkucm'
    'Vtb3RlLmF1dGgudjEuQ2xpZW50QWNjZXNzUmV2b2tlUmVxdWVzdFIHcmVxdWVzdEoECAEQAg==');

@$core.Deprecated('Use clientAccessIdentityResultDescriptor instead')
const ClientAccessIdentityResult$json = {
  '1': 'ClientAccessIdentityResult',
  '2': [
    {
      '1': 'identity',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessIdentityResult',
      '10': 'identity'
    },
    {'1': 'challenge', '3': 2, '4': 1, '5': 12, '10': 'challenge'},
    {'1': 'proof', '3': 3, '4': 1, '5': 12, '10': 'proof'},
  ],
};

/// Descriptor for `ClientAccessIdentityResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessIdentityResultDescriptor =
    $convert.base64Decode(
        'ChpDbGllbnRBY2Nlc3NJZGVudGl0eVJlc3VsdBJNCghpZGVudGl0eRgBIAEoCzIxLmFueXR0eS'
        '5yZW1vdGUuYXV0aC52MS5DbGllbnRBY2Nlc3NJZGVudGl0eVJlc3VsdFIIaWRlbnRpdHkSHAoJ'
        'Y2hhbGxlbmdlGAIgASgMUgljaGFsbGVuZ2USFAoFcHJvb2YYAyABKAxSBXByb29m');

@$core.Deprecated('Use clientAccessListResultDescriptor instead')
const ClientAccessListResult$json = {
  '1': 'ClientAccessListResult',
  '2': [
    {
      '1': 'access',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessListResult',
      '10': 'access'
    },
  ],
};

/// Descriptor for `ClientAccessListResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessListResultDescriptor =
    $convert.base64Decode(
        'ChZDbGllbnRBY2Nlc3NMaXN0UmVzdWx0EkUKBmFjY2VzcxgBIAEoCzItLmFueXR0eS5yZW1vdG'
        'UuYXV0aC52MS5DbGllbnRBY2Nlc3NMaXN0UmVzdWx0UgZhY2Nlc3M=');

@$core.Deprecated('Use clientAccessTicketCreateResultDescriptor instead')
const ClientAccessTicketCreateResult$json = {
  '1': 'ClientAccessTicketCreateResult',
  '2': [
    {
      '1': 'ticket',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessTicketCreateResult',
      '10': 'ticket'
    },
  ],
};

/// Descriptor for `ClientAccessTicketCreateResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessTicketCreateResultDescriptor =
    $convert.base64Decode(
        'Ch5DbGllbnRBY2Nlc3NUaWNrZXRDcmVhdGVSZXN1bHQSTQoGdGlja2V0GAEgASgLMjUuYW55dH'
        'R5LnJlbW90ZS5hdXRoLnYxLkNsaWVudEFjY2Vzc1RpY2tldENyZWF0ZVJlc3VsdFIGdGlja2V0');

@$core.Deprecated('Use clientAccessRevokeResultDescriptor instead')
const ClientAccessRevokeResult$json = {
  '1': 'ClientAccessRevokeResult',
  '2': [
    {
      '1': 'record',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessRecord',
      '10': 'record'
    },
  ],
};

/// Descriptor for `ClientAccessRevokeResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessRevokeResultDescriptor =
    $convert.base64Decode(
        'ChhDbGllbnRBY2Nlc3NSZXZva2VSZXN1bHQSQQoGcmVjb3JkGAEgASgLMikuYW55dHR5LnJlbW'
        '90ZS5hdXRoLnYxLkNsaWVudEFjY2Vzc1JlY29yZFIGcmVjb3Jk');

@$core.Deprecated('Use remoteStatusCommandDescriptor instead')
const RemoteStatusCommand$json = {
  '1': 'RemoteStatusCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteStatusCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteStatusCommandDescriptor =
    $convert.base64Decode('ChNSZW1vdGVTdGF0dXNDb21tYW5kSgQIARAC');

@$core.Deprecated('Use remotePairStartCommandDescriptor instead')
const RemotePairStartCommand$json = {
  '1': 'RemotePairStartCommand',
  '2': [
    {'1': 'local_pair_url', '3': 2, '4': 1, '5': 9, '10': 'localPairUrl'},
    {'1': 'ttl_seconds', '3': 3, '4': 1, '5': 5, '10': 'ttlSeconds'},
    {'1': 'auth_ttl_seconds', '3': 4, '4': 1, '5': 5, '10': 'authTtlSeconds'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemotePairStartCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remotePairStartCommandDescriptor = $convert.base64Decode(
    'ChZSZW1vdGVQYWlyU3RhcnRDb21tYW5kEiQKDmxvY2FsX3BhaXJfdXJsGAIgASgJUgxsb2NhbF'
    'BhaXJVcmwSHwoLdHRsX3NlY29uZHMYAyABKAVSCnR0bFNlY29uZHMSKAoQYXV0aF90dGxfc2Vj'
    'b25kcxgEIAEoBVIOYXV0aFR0bFNlY29uZHNKBAgBEAI=');

@$core.Deprecated('Use remoteLocalEnableCommandDescriptor instead')
const RemoteLocalEnableCommand$json = {
  '1': 'RemoteLocalEnableCommand',
  '2': [
    {'1': 'local_web_address', '3': 2, '4': 1, '5': 9, '10': 'localWebAddress'},
    {'1': 'ice_tcp_address', '3': 3, '4': 1, '5': 9, '10': 'iceTcpAddress'},
    {'1': 'hub_urls', '3': 4, '4': 3, '5': 9, '10': 'hubUrls'},
    {'1': 'control_url', '3': 5, '4': 1, '5': 9, '10': 'controlUrl'},
    {'1': 'access_token', '3': 6, '4': 1, '5': 9, '10': 'accessToken'},
    {'1': 'region', '3': 7, '4': 1, '5': 9, '10': 'region'},
    {
      '1': 'local_web_password',
      '3': 8,
      '4': 1,
      '5': 12,
      '10': 'localWebPassword'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteLocalEnableCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteLocalEnableCommandDescriptor = $convert.base64Decode(
    'ChhSZW1vdGVMb2NhbEVuYWJsZUNvbW1hbmQSKgoRbG9jYWxfd2ViX2FkZHJlc3MYAiABKAlSD2'
    'xvY2FsV2ViQWRkcmVzcxImCg9pY2VfdGNwX2FkZHJlc3MYAyABKAlSDWljZVRjcEFkZHJlc3MS'
    'GQoIaHViX3VybHMYBCADKAlSB2h1YlVybHMSHwoLY29udHJvbF91cmwYBSABKAlSCmNvbnRyb2'
    'xVcmwSIQoMYWNjZXNzX3Rva2VuGAYgASgJUgthY2Nlc3NUb2tlbhIWCgZyZWdpb24YByABKAlS'
    'BnJlZ2lvbhIsChJsb2NhbF93ZWJfcGFzc3dvcmQYCCABKAxSEGxvY2FsV2ViUGFzc3dvcmRKBA'
    'gBEAI=');

@$core.Deprecated('Use remoteLocalStatusCommandDescriptor instead')
const RemoteLocalStatusCommand$json = {
  '1': 'RemoteLocalStatusCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteLocalStatusCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteLocalStatusCommandDescriptor =
    $convert.base64Decode('ChhSZW1vdGVMb2NhbFN0YXR1c0NvbW1hbmRKBAgBEAI=');

@$core.Deprecated('Use remoteLocalDisableCommandDescriptor instead')
const RemoteLocalDisableCommand$json = {
  '1': 'RemoteLocalDisableCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteLocalDisableCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteLocalDisableCommandDescriptor =
    $convert.base64Decode('ChlSZW1vdGVMb2NhbERpc2FibGVDb21tYW5kSgQIARAC');

@$core.Deprecated('Use remoteCloudStatusCommandDescriptor instead')
const RemoteCloudStatusCommand$json = {
  '1': 'RemoteCloudStatusCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteCloudStatusCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudStatusCommandDescriptor =
    $convert.base64Decode('ChhSZW1vdGVDbG91ZFN0YXR1c0NvbW1hbmRKBAgBEAI=');

@$core.Deprecated('Use remoteCloudEnableCommandDescriptor instead')
const RemoteCloudEnableCommand$json = {
  '1': 'RemoteCloudEnableCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteCloudEnableCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudEnableCommandDescriptor =
    $convert.base64Decode('ChhSZW1vdGVDbG91ZEVuYWJsZUNvbW1hbmRKBAgBEAI=');

@$core.Deprecated('Use remoteCloudDisableCommandDescriptor instead')
const RemoteCloudDisableCommand$json = {
  '1': 'RemoteCloudDisableCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteCloudDisableCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudDisableCommandDescriptor =
    $convert.base64Decode('ChlSZW1vdGVDbG91ZERpc2FibGVDb21tYW5kSgQIARAC');

@$core.Deprecated('Use remoteCloudEdgesCommandDescriptor instead')
const RemoteCloudEdgesCommand$json = {
  '1': 'RemoteCloudEdgesCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteCloudEdgesCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudEdgesCommandDescriptor =
    $convert.base64Decode('ChdSZW1vdGVDbG91ZEVkZ2VzQ29tbWFuZEoECAEQAg==');

@$core.Deprecated('Use remoteCloudPreferEdgeCommandDescriptor instead')
const RemoteCloudPreferEdgeCommand$json = {
  '1': 'RemoteCloudPreferEdgeCommand',
  '2': [
    {'1': 'edge_id', '3': 2, '4': 1, '5': 9, '10': 'edgeId'},
    {
      '1': 'expected_preference_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'expectedPreferenceRevision'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteCloudPreferEdgeCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudPreferEdgeCommandDescriptor =
    $convert.base64Decode(
        'ChxSZW1vdGVDbG91ZFByZWZlckVkZ2VDb21tYW5kEhcKB2VkZ2VfaWQYAiABKAlSBmVkZ2VJZB'
        'JAChxleHBlY3RlZF9wcmVmZXJlbmNlX3JldmlzaW9uGAMgASgEUhpleHBlY3RlZFByZWZlcmVu'
        'Y2VSZXZpc2lvbkoECAEQAg==');

@$core.Deprecated('Use remoteCloudReselectEdgeCommandDescriptor instead')
const RemoteCloudReselectEdgeCommand$json = {
  '1': 'RemoteCloudReselectEdgeCommand',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `RemoteCloudReselectEdgeCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudReselectEdgeCommandDescriptor = $convert
    .base64Decode('Ch5SZW1vdGVDbG91ZFJlc2VsZWN0RWRnZUNvbW1hbmRKBAgBEAI=');

@$core.Deprecated('Use remoteStatusResultDescriptor instead')
const RemoteStatusResult$json = {
  '1': 'RemoteStatusResult',
  '2': [
    {'1': 'state', '3': 1, '4': 1, '5': 9, '10': 'state'},
    {'1': 'detail', '3': 2, '4': 1, '5': 9, '10': 'detail'},
    {'1': 'device_id', '3': 3, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'device_name', '3': 4, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'control_url', '3': 5, '4': 1, '5': 9, '10': 'controlUrl'},
    {'1': 'hub_url', '3': 6, '4': 1, '5': 9, '10': 'hubUrl'},
    {'1': 'hub_urls', '3': 7, '4': 3, '5': 9, '10': 'hubUrls'},
    {'1': 'data_directory', '3': 8, '4': 1, '5': 9, '10': 'dataDirectory'},
    {'1': 'mode', '3': 9, '4': 1, '5': 9, '10': 'mode'},
    {'1': 'allow_lan', '3': 10, '4': 1, '5': 8, '10': 'allowLan'},
    {'1': 'terminal_count', '3': 11, '4': 1, '5': 5, '10': 'terminalCount'},
    {
      '1': 'updated_at_unix_nano',
      '3': 12,
      '4': 1,
      '5': 3,
      '10': 'updatedAtUnixNano'
    },
  ],
};

/// Descriptor for `RemoteStatusResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteStatusResultDescriptor = $convert.base64Decode(
    'ChJSZW1vdGVTdGF0dXNSZXN1bHQSFAoFc3RhdGUYASABKAlSBXN0YXRlEhYKBmRldGFpbBgCIA'
    'EoCVIGZGV0YWlsEhsKCWRldmljZV9pZBgDIAEoCVIIZGV2aWNlSWQSHwoLZGV2aWNlX25hbWUY'
    'BCABKAlSCmRldmljZU5hbWUSHwoLY29udHJvbF91cmwYBSABKAlSCmNvbnRyb2xVcmwSFwoHaH'
    'ViX3VybBgGIAEoCVIGaHViVXJsEhkKCGh1Yl91cmxzGAcgAygJUgdodWJVcmxzEiUKDmRhdGFf'
    'ZGlyZWN0b3J5GAggASgJUg1kYXRhRGlyZWN0b3J5EhIKBG1vZGUYCSABKAlSBG1vZGUSGwoJYW'
    'xsb3dfbGFuGAogASgIUghhbGxvd0xhbhIlCg50ZXJtaW5hbF9jb3VudBgLIAEoBVINdGVybWlu'
    'YWxDb3VudBIvChR1cGRhdGVkX2F0X3VuaXhfbmFubxgMIAEoA1IRdXBkYXRlZEF0VW5peE5hbm'
    '8=');

@$core.Deprecated('Use remotePairStartResultDescriptor instead')
const RemotePairStartResult$json = {
  '1': 'RemotePairStartResult',
  '2': [
    {'1': 'type', '3': 1, '4': 1, '5': 9, '10': 'type'},
    {'1': 'machine_id', '3': 2, '4': 1, '5': 9, '10': 'machineId'},
    {'1': 'machine_name', '3': 3, '4': 1, '5': 9, '10': 'machineName'},
    {'1': 'local_pair_url', '3': 4, '4': 1, '5': 9, '10': 'localPairUrl'},
    {'1': 'pair_session_id', '3': 5, '4': 1, '5': 9, '10': 'pairSessionId'},
    {'1': 'pair_secret', '3': 6, '4': 1, '5': 9, '10': 'pairSecret'},
    {
      '1': 'answer_proof_secret',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'answerProofSecret'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 8,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
  ],
};

/// Descriptor for `RemotePairStartResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remotePairStartResultDescriptor = $convert.base64Decode(
    'ChVSZW1vdGVQYWlyU3RhcnRSZXN1bHQSEgoEdHlwZRgBIAEoCVIEdHlwZRIdCgptYWNoaW5lX2'
    'lkGAIgASgJUgltYWNoaW5lSWQSIQoMbWFjaGluZV9uYW1lGAMgASgJUgttYWNoaW5lTmFtZRIk'
    'Cg5sb2NhbF9wYWlyX3VybBgEIAEoCVIMbG9jYWxQYWlyVXJsEiYKD3BhaXJfc2Vzc2lvbl9pZB'
    'gFIAEoCVINcGFpclNlc3Npb25JZBIfCgtwYWlyX3NlY3JldBgGIAEoCVIKcGFpclNlY3JldBIu'
    'ChNhbnN3ZXJfcHJvb2Zfc2VjcmV0GAcgASgJUhFhbnN3ZXJQcm9vZlNlY3JldBIvChRleHBpcm'
    'VzX2F0X3VuaXhfbmFubxgIIAEoA1IRZXhwaXJlc0F0VW5peE5hbm8=');

@$core.Deprecated('Use remoteLocalStatusResultDescriptor instead')
const RemoteLocalStatusResult$json = {
  '1': 'RemoteLocalStatusResult',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'http_url', '3': 2, '4': 1, '5': 9, '10': 'httpUrl'},
    {'1': 'local_web_address', '3': 3, '4': 1, '5': 9, '10': 'localWebAddress'},
    {'1': 'local_pair_url', '3': 4, '4': 1, '5': 9, '10': 'localPairUrl'},
    {'1': 'ice_tcp_enabled', '3': 5, '4': 1, '5': 8, '10': 'iceTcpEnabled'},
    {'1': 'ice_tcp_address', '3': 6, '4': 1, '5': 9, '10': 'iceTcpAddress'},
    {'1': 'ice_tcp_port', '3': 7, '4': 1, '5': 5, '10': 'iceTcpPort'},
    {
      '1': 'updated_at_unix_nano',
      '3': 8,
      '4': 1,
      '5': 3,
      '10': 'updatedAtUnixNano'
    },
    {
      '1': 'password_protected',
      '3': 9,
      '4': 1,
      '5': 8,
      '10': 'passwordProtected'
    },
  ],
};

/// Descriptor for `RemoteLocalStatusResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteLocalStatusResultDescriptor = $convert.base64Decode(
    'ChdSZW1vdGVMb2NhbFN0YXR1c1Jlc3VsdBIYCgdlbmFibGVkGAEgASgIUgdlbmFibGVkEhkKCG'
    'h0dHBfdXJsGAIgASgJUgdodHRwVXJsEioKEWxvY2FsX3dlYl9hZGRyZXNzGAMgASgJUg9sb2Nh'
    'bFdlYkFkZHJlc3MSJAoObG9jYWxfcGFpcl91cmwYBCABKAlSDGxvY2FsUGFpclVybBImCg9pY2'
    'VfdGNwX2VuYWJsZWQYBSABKAhSDWljZVRjcEVuYWJsZWQSJgoPaWNlX3RjcF9hZGRyZXNzGAYg'
    'ASgJUg1pY2VUY3BBZGRyZXNzEiAKDGljZV90Y3BfcG9ydBgHIAEoBVIKaWNlVGNwUG9ydBIvCh'
    'R1cGRhdGVkX2F0X3VuaXhfbmFubxgIIAEoA1IRdXBkYXRlZEF0VW5peE5hbm8SLQoScGFzc3dv'
    'cmRfcHJvdGVjdGVkGAkgASgIUhFwYXNzd29yZFByb3RlY3RlZA==');

@$core.Deprecated('Use remoteCloudStatusResultDescriptor instead')
const RemoteCloudStatusResult$json = {
  '1': 'RemoteCloudStatusResult',
  '2': [
    {'1': 'enrolled', '3': 1, '4': 1, '5': 8, '10': 'enrolled'},
    {'1': 'enabled', '3': 2, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'running', '3': 3, '4': 1, '5': 8, '10': 'running'},
    {'1': 'state', '3': 4, '4': 1, '5': 9, '10': 'state'},
    {'1': 'detail', '3': 5, '4': 1, '5': 9, '10': 'detail'},
    {'1': 'daemon_id', '3': 6, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'account_id', '3': 7, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'edge_id', '3': 8, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'edge_name', '3': 9, '4': 1, '5': 9, '10': 'edgeName'},
    {'1': 'edge_region', '3': 10, '4': 1, '5': 9, '10': 'edgeRegion'},
    {'1': 'public_endpoint', '3': 11, '4': 1, '5': 9, '10': 'publicEndpoint'},
    {'1': 'server_name', '3': 12, '4': 1, '5': 9, '10': 'serverName'},
    {'1': 'lifecycle_state', '3': 13, '4': 1, '5': 9, '10': 'lifecycleState'},
    {
      '1': 'lifecycle_revision',
      '3': 14,
      '4': 1,
      '5': 4,
      '10': 'lifecycleRevision'
    },
    {'1': 'ready', '3': 15, '4': 1, '5': 8, '10': 'ready'},
    {'1': 'active_sessions', '3': 16, '4': 1, '5': 5, '10': 'activeSessions'},
    {
      '1': 'enrolled_at_unix_nano',
      '3': 17,
      '4': 1,
      '5': 3,
      '10': 'enrolledAtUnixNano'
    },
    {
      '1': 'updated_at_unix_nano',
      '3': 18,
      '4': 1,
      '5': 3,
      '10': 'updatedAtUnixNano'
    },
    {'1': 'record_path', '3': 19, '4': 1, '5': 9, '10': 'recordPath'},
    {'1': 'disabled_path', '3': 20, '4': 1, '5': 9, '10': 'disabledPath'},
  ],
};

/// Descriptor for `RemoteCloudStatusResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudStatusResultDescriptor = $convert.base64Decode(
    'ChdSZW1vdGVDbG91ZFN0YXR1c1Jlc3VsdBIaCghlbnJvbGxlZBgBIAEoCFIIZW5yb2xsZWQSGA'
    'oHZW5hYmxlZBgCIAEoCFIHZW5hYmxlZBIYCgdydW5uaW5nGAMgASgIUgdydW5uaW5nEhQKBXN0'
    'YXRlGAQgASgJUgVzdGF0ZRIWCgZkZXRhaWwYBSABKAlSBmRldGFpbBIbCglkYWVtb25faWQYBi'
    'ABKAlSCGRhZW1vbklkEh0KCmFjY291bnRfaWQYByABKAlSCWFjY291bnRJZBIXCgdlZGdlX2lk'
    'GAggASgJUgZlZGdlSWQSGwoJZWRnZV9uYW1lGAkgASgJUghlZGdlTmFtZRIfCgtlZGdlX3JlZ2'
    'lvbhgKIAEoCVIKZWRnZVJlZ2lvbhInCg9wdWJsaWNfZW5kcG9pbnQYCyABKAlSDnB1YmxpY0Vu'
    'ZHBvaW50Eh8KC3NlcnZlcl9uYW1lGAwgASgJUgpzZXJ2ZXJOYW1lEicKD2xpZmVjeWNsZV9zdG'
    'F0ZRgNIAEoCVIObGlmZWN5Y2xlU3RhdGUSLQoSbGlmZWN5Y2xlX3JldmlzaW9uGA4gASgEUhFs'
    'aWZlY3ljbGVSZXZpc2lvbhIUCgVyZWFkeRgPIAEoCFIFcmVhZHkSJwoPYWN0aXZlX3Nlc3Npb2'
    '5zGBAgASgFUg5hY3RpdmVTZXNzaW9ucxIxChVlbnJvbGxlZF9hdF91bml4X25hbm8YESABKANS'
    'EmVucm9sbGVkQXRVbml4TmFubxIvChR1cGRhdGVkX2F0X3VuaXhfbmFubxgSIAEoA1IRdXBkYX'
    'RlZEF0VW5peE5hbm8SHwoLcmVjb3JkX3BhdGgYEyABKAlSCnJlY29yZFBhdGgSIwoNZGlzYWJs'
    'ZWRfcGF0aBgUIAEoCVIMZGlzYWJsZWRQYXRo');

@$core.Deprecated('Use remoteCloudEdgesResultDescriptor instead')
const RemoteCloudEdgesResult$json = {
  '1': 'RemoteCloudEdgesResult',
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

/// Descriptor for `RemoteCloudEdgesResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List remoteCloudEdgesResultDescriptor =
    $convert.base64Decode(
        'ChZSZW1vdGVDbG91ZEVkZ2VzUmVzdWx0EkIKCXNlbGVjdGlvbhgBIAEoCzIkLmFueXR0eS5jbG'
        '91ZC52MS5EYWVtb25FZGdlU2VsZWN0aW9uUglzZWxlY3Rpb24=');
