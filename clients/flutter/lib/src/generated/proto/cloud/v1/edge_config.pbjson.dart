// This is a generated file - do not edit.
//
// Generated from cloud/v1/edge_config.proto.

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

@$core.Deprecated('Use edgeDesiredConfigDescriptor instead')
const EdgeDesiredConfig$json = {
  '1': 'EdgeDesiredConfig',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'version', '3': 2, '4': 1, '5': 4, '10': 'version'},
    {'1': 'name', '3': 3, '4': 1, '5': 9, '10': 'name'},
    {'1': 'region', '3': 4, '4': 1, '5': 9, '10': 'region'},
    {'1': 'capacity', '3': 5, '4': 1, '5': 4, '10': 'capacity'},
    {'1': 'public_endpoint', '3': 6, '4': 1, '5': 9, '10': 'publicEndpoint'},
    {'1': 'enabled', '3': 7, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `EdgeDesiredConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeDesiredConfigDescriptor = $convert.base64Decode(
    'ChFFZGdlRGVzaXJlZENvbmZpZxIXCgdlZGdlX2lkGAEgASgJUgZlZGdlSWQSGAoHdmVyc2lvbh'
    'gCIAEoBFIHdmVyc2lvbhISCgRuYW1lGAMgASgJUgRuYW1lEhYKBnJlZ2lvbhgEIAEoCVIGcmVn'
    'aW9uEhoKCGNhcGFjaXR5GAUgASgEUghjYXBhY2l0eRInCg9wdWJsaWNfZW5kcG9pbnQYBiABKA'
    'lSDnB1YmxpY0VuZHBvaW50EhgKB2VuYWJsZWQYByABKAhSB2VuYWJsZWQ=');

@$core.Deprecated('Use signedEdgeDesiredConfigDescriptor instead')
const SignedEdgeDesiredConfig$json = {
  '1': 'SignedEdgeDesiredConfig',
  '2': [
    {'1': 'key_id', '3': 1, '4': 1, '5': 9, '10': 'keyId'},
    {'1': 'payload', '3': 2, '4': 1, '5': 12, '10': 'payload'},
    {'1': 'signature', '3': 3, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `SignedEdgeDesiredConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signedEdgeDesiredConfigDescriptor =
    $convert.base64Decode(
        'ChdTaWduZWRFZGdlRGVzaXJlZENvbmZpZxIVCgZrZXlfaWQYASABKAlSBWtleUlkEhgKB3BheW'
        'xvYWQYAiABKAxSB3BheWxvYWQSHAoJc2lnbmF0dXJlGAMgASgMUglzaWduYXR1cmU=');

@$core.Deprecated('Use edgeRuntimeProjectionDescriptor instead')
const EdgeRuntimeProjection$json = {
  '1': 'EdgeRuntimeProjection',
  '2': [
    {'1': 'online', '3': 1, '4': 1, '5': 8, '10': 'online'},
    {'1': 'boot_id', '3': 2, '4': 1, '5': 9, '10': 'bootId'},
    {'1': 'connection_id', '3': 3, '4': 1, '5': 9, '10': 'connectionId'},
    {'1': 'software_version', '3': 4, '4': 1, '5': 9, '10': 'softwareVersion'},
    {'1': 'runtime_revision', '3': 5, '4': 1, '5': 4, '10': 'runtimeRevision'},
    {'1': 'agent_count', '3': 6, '4': 1, '5': 4, '10': 'agentCount'},
    {'1': 'session_count', '3': 7, '4': 1, '5': 4, '10': 'sessionCount'},
    {
      '1': 'connected_at',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'connectedAt'
    },
    {
      '1': 'last_heartbeat',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'lastHeartbeat'
    },
  ],
  '9': [
    {'1': 10, '2': 11},
  ],
};

/// Descriptor for `EdgeRuntimeProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeRuntimeProjectionDescriptor = $convert.base64Decode(
    'ChVFZGdlUnVudGltZVByb2plY3Rpb24SFgoGb25saW5lGAEgASgIUgZvbmxpbmUSFwoHYm9vdF'
    '9pZBgCIAEoCVIGYm9vdElkEiMKDWNvbm5lY3Rpb25faWQYAyABKAlSDGNvbm5lY3Rpb25JZBIp'
    'ChBzb2Z0d2FyZV92ZXJzaW9uGAQgASgJUg9zb2Z0d2FyZVZlcnNpb24SKQoQcnVudGltZV9yZX'
    'Zpc2lvbhgFIAEoBFIPcnVudGltZVJldmlzaW9uEh8KC2FnZW50X2NvdW50GAYgASgEUgphZ2Vu'
    'dENvdW50EiMKDXNlc3Npb25fY291bnQYByABKARSDHNlc3Npb25Db3VudBI9Cgxjb25uZWN0ZW'
    'RfYXQYCCABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgtjb25uZWN0ZWRBdBJBCg5s'
    'YXN0X2hlYXJ0YmVhdBgJIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSDWxhc3RIZW'
    'FydGJlYXRKBAgKEAs=');

@$core.Deprecated('Use managedEdgeDescriptor instead')
const ManagedEdge$json = {
  '1': 'ManagedEdge',
  '2': [
    {
      '1': 'config',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeDesiredConfig',
      '10': 'config'
    },
    {'1': 'config_revision', '3': 2, '4': 1, '5': 4, '10': 'configRevision'},
    {
      '1': 'runtime',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeRuntimeProjection',
      '10': 'runtime'
    },
    {
      '1': 'public_certificate',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePublicCertificateStatus',
      '10': 'publicCertificate'
    },
  ],
};

/// Descriptor for `ManagedEdge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List managedEdgeDescriptor = $convert.base64Decode(
    'CgtNYW5hZ2VkRWRnZRI6CgZjb25maWcYASABKAsyIi5hbnl0dHkuY2xvdWQudjEuRWRnZURlc2'
    'lyZWRDb25maWdSBmNvbmZpZxInCg9jb25maWdfcmV2aXNpb24YAiABKARSDmNvbmZpZ1Jldmlz'
    'aW9uEkAKB3J1bnRpbWUYAyABKAsyJi5hbnl0dHkuY2xvdWQudjEuRWRnZVJ1bnRpbWVQcm9qZW'
    'N0aW9uUgdydW50aW1lElsKEnB1YmxpY19jZXJ0aWZpY2F0ZRgEIAEoCzIsLmFueXR0eS5jbG91'
    'ZC52MS5FZGdlUHVibGljQ2VydGlmaWNhdGVTdGF0dXNSEXB1YmxpY0NlcnRpZmljYXRl');

@$core.Deprecated('Use listEdgesRequestDescriptor instead')
const ListEdgesRequest$json = {
  '1': 'ListEdgesRequest',
};

/// Descriptor for `ListEdgesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listEdgesRequestDescriptor =
    $convert.base64Decode('ChBMaXN0RWRnZXNSZXF1ZXN0');

@$core.Deprecated('Use listEdgesResponseDescriptor instead')
const ListEdgesResponse$json = {
  '1': 'ListEdgesResponse',
  '2': [
    {
      '1': 'edges',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.ManagedEdge',
      '10': 'edges'
    },
  ],
};

/// Descriptor for `ListEdgesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listEdgesResponseDescriptor = $convert.base64Decode(
    'ChFMaXN0RWRnZXNSZXNwb25zZRIyCgVlZGdlcxgBIAMoCzIcLmFueXR0eS5jbG91ZC52MS5NYW'
    '5hZ2VkRWRnZVIFZWRnZXM=');

@$core.Deprecated('Use createEdgeRequestDescriptor instead')
const CreateEdgeRequest$json = {
  '1': 'CreateEdgeRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'region', '3': 2, '4': 1, '5': 9, '10': 'region'},
    {'1': 'capacity', '3': 3, '4': 1, '5': 4, '10': 'capacity'},
    {'1': 'public_endpoint', '3': 4, '4': 1, '5': 9, '10': 'publicEndpoint'},
  ],
};

/// Descriptor for `CreateEdgeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createEdgeRequestDescriptor = $convert.base64Decode(
    'ChFDcmVhdGVFZGdlUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEhYKBnJlZ2lvbhgCIAEoCV'
    'IGcmVnaW9uEhoKCGNhcGFjaXR5GAMgASgEUghjYXBhY2l0eRInCg9wdWJsaWNfZW5kcG9pbnQY'
    'BCABKAlSDnB1YmxpY0VuZHBvaW50');

@$core.Deprecated('Use createEdgeResponseDescriptor instead')
const CreateEdgeResponse$json = {
  '1': 'CreateEdgeResponse',
  '2': [
    {
      '1': 'edge',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ManagedEdge',
      '10': 'edge'
    },
    {'1': 'install_command', '3': 2, '4': 1, '5': 9, '10': 'installCommand'},
    {
      '1': 'claim_expires_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'claimExpiresAt'
    },
  ],
};

/// Descriptor for `CreateEdgeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createEdgeResponseDescriptor = $convert.base64Decode(
    'ChJDcmVhdGVFZGdlUmVzcG9uc2USMAoEZWRnZRgBIAEoCzIcLmFueXR0eS5jbG91ZC52MS5NYW'
    '5hZ2VkRWRnZVIEZWRnZRInCg9pbnN0YWxsX2NvbW1hbmQYAiABKAlSDmluc3RhbGxDb21tYW5k'
    'EkQKEGNsYWltX2V4cGlyZXNfYXQYAyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUg'
    '5jbGFpbUV4cGlyZXNBdA==');

@$core.Deprecated('Use updateEdgeRequestDescriptor instead')
const UpdateEdgeRequest$json = {
  '1': 'UpdateEdgeRequest',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {
      '1': 'expected_revision',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {'1': 'name', '3': 3, '4': 1, '5': 9, '10': 'name'},
    {'1': 'region', '3': 4, '4': 1, '5': 9, '10': 'region'},
    {'1': 'capacity', '3': 5, '4': 1, '5': 4, '10': 'capacity'},
    {'1': 'public_endpoint', '3': 6, '4': 1, '5': 9, '10': 'publicEndpoint'},
    {'1': 'enabled', '3': 7, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `UpdateEdgeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateEdgeRequestDescriptor = $convert.base64Decode(
    'ChFVcGRhdGVFZGdlUmVxdWVzdBIXCgdlZGdlX2lkGAEgASgJUgZlZGdlSWQSKwoRZXhwZWN0ZW'
    'RfcmV2aXNpb24YAiABKARSEGV4cGVjdGVkUmV2aXNpb24SEgoEbmFtZRgDIAEoCVIEbmFtZRIW'
    'CgZyZWdpb24YBCABKAlSBnJlZ2lvbhIaCghjYXBhY2l0eRgFIAEoBFIIY2FwYWNpdHkSJwoPcH'
    'VibGljX2VuZHBvaW50GAYgASgJUg5wdWJsaWNFbmRwb2ludBIYCgdlbmFibGVkGAcgASgIUgdl'
    'bmFibGVk');

@$core.Deprecated('Use updateEdgeResponseDescriptor instead')
const UpdateEdgeResponse$json = {
  '1': 'UpdateEdgeResponse',
  '2': [
    {
      '1': 'edge',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ManagedEdge',
      '10': 'edge'
    },
  ],
};

/// Descriptor for `UpdateEdgeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateEdgeResponseDescriptor = $convert.base64Decode(
    'ChJVcGRhdGVFZGdlUmVzcG9uc2USMAoEZWRnZRgBIAEoCzIcLmFueXR0eS5jbG91ZC52MS5NYW'
    '5hZ2VkRWRnZVIEZWRnZQ==');

@$core.Deprecated('Use regenerateEdgeInstallRequestDescriptor instead')
const RegenerateEdgeInstallRequest$json = {
  '1': 'RegenerateEdgeInstallRequest',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
  ],
};

/// Descriptor for `RegenerateEdgeInstallRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List regenerateEdgeInstallRequestDescriptor =
    $convert.base64Decode(
        'ChxSZWdlbmVyYXRlRWRnZUluc3RhbGxSZXF1ZXN0EhcKB2VkZ2VfaWQYASABKAlSBmVkZ2VJZA'
        '==');

@$core.Deprecated('Use regenerateEdgeInstallResponseDescriptor instead')
const RegenerateEdgeInstallResponse$json = {
  '1': 'RegenerateEdgeInstallResponse',
  '2': [
    {
      '1': 'edge',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ManagedEdge',
      '10': 'edge'
    },
    {'1': 'install_command', '3': 2, '4': 1, '5': 9, '10': 'installCommand'},
    {
      '1': 'claim_expires_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'claimExpiresAt'
    },
  ],
};

/// Descriptor for `RegenerateEdgeInstallResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List regenerateEdgeInstallResponseDescriptor = $convert.base64Decode(
    'Ch1SZWdlbmVyYXRlRWRnZUluc3RhbGxSZXNwb25zZRIwCgRlZGdlGAEgASgLMhwuYW55dHR5Lm'
    'Nsb3VkLnYxLk1hbmFnZWRFZGdlUgRlZGdlEicKD2luc3RhbGxfY29tbWFuZBgCIAEoCVIOaW5z'
    'dGFsbENvbW1hbmQSRAoQY2xhaW1fZXhwaXJlc19hdBgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi'
    '5UaW1lc3RhbXBSDmNsYWltRXhwaXJlc0F0');

@$core.Deprecated('Use deleteEdgeRequestDescriptor instead')
const DeleteEdgeRequest$json = {
  '1': 'DeleteEdgeRequest',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {
      '1': 'expected_revision',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {'1': 'reason', '3': 3, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `DeleteEdgeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteEdgeRequestDescriptor = $convert.base64Decode(
    'ChFEZWxldGVFZGdlUmVxdWVzdBIXCgdlZGdlX2lkGAEgASgJUgZlZGdlSWQSKwoRZXhwZWN0ZW'
    'RfcmV2aXNpb24YAiABKARSEGV4cGVjdGVkUmV2aXNpb24SFgoGcmVhc29uGAMgASgJUgZyZWFz'
    'b24=');

@$core.Deprecated('Use deleteEdgeResponseDescriptor instead')
const DeleteEdgeResponse$json = {
  '1': 'DeleteEdgeResponse',
};

/// Descriptor for `DeleteEdgeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteEdgeResponseDescriptor =
    $convert.base64Decode('ChJEZWxldGVFZGdlUmVzcG9uc2U=');

@$core.Deprecated('Use registerEdgeRequestDescriptor instead')
const RegisterEdgeRequest$json = {
  '1': 'RegisterEdgeRequest',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'bootstrap_token', '3': 2, '4': 1, '5': 9, '10': 'bootstrapToken'},
    {'1': 'identity_csr_pem', '3': 3, '4': 1, '5': 12, '10': 'identityCsrPem'},
    {'1': 'public_csr_pem', '3': 4, '4': 1, '5': 12, '10': 'publicCsrPem'},
  ],
};

/// Descriptor for `RegisterEdgeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerEdgeRequestDescriptor = $convert.base64Decode(
    'ChNSZWdpc3RlckVkZ2VSZXF1ZXN0EhcKB2VkZ2VfaWQYASABKAlSBmVkZ2VJZBInCg9ib290c3'
    'RyYXBfdG9rZW4YAiABKAlSDmJvb3RzdHJhcFRva2VuEigKEGlkZW50aXR5X2Nzcl9wZW0YAyAB'
    'KAxSDmlkZW50aXR5Q3NyUGVtEiQKDnB1YmxpY19jc3JfcGVtGAQgASgMUgxwdWJsaWNDc3JQZW'
    '0=');

@$core.Deprecated('Use registerEdgeResponseDescriptor instead')
const RegisterEdgeResponse$json = {
  '1': 'RegisterEdgeResponse',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {
      '1': 'identity_certificate_pem',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'identityCertificatePem'
    },
    {
      '1': 'public_certificate_pem',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'publicCertificatePem'
    },
    {
      '1': 'edge_ca_certificate_pem',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'edgeCaCertificatePem'
    },
    {
      '1': 'controller_address',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'controllerAddress'
    },
    {
      '1': 'controller_server_name',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'controllerServerName'
    },
    {
      '1': 'desired_config',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEdgeDesiredConfig',
      '10': 'desiredConfig'
    },
    {'1': 'config_key_id', '3': 9, '4': 1, '5': 9, '10': 'configKeyId'},
    {
      '1': 'config_signing_public_key',
      '3': 10,
      '4': 1,
      '5': 12,
      '10': 'configSigningPublicKey'
    },
  ],
  '9': [
    {'1': 5, '2': 6},
  ],
};

/// Descriptor for `RegisterEdgeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerEdgeResponseDescriptor = $convert.base64Decode(
    'ChRSZWdpc3RlckVkZ2VSZXNwb25zZRIXCgdlZGdlX2lkGAEgASgJUgZlZGdlSWQSOAoYaWRlbn'
    'RpdHlfY2VydGlmaWNhdGVfcGVtGAIgASgMUhZpZGVudGl0eUNlcnRpZmljYXRlUGVtEjQKFnB1'
    'YmxpY19jZXJ0aWZpY2F0ZV9wZW0YAyABKAxSFHB1YmxpY0NlcnRpZmljYXRlUGVtEjUKF2VkZ2'
    'VfY2FfY2VydGlmaWNhdGVfcGVtGAQgASgMUhRlZGdlQ2FDZXJ0aWZpY2F0ZVBlbRItChJjb250'
    'cm9sbGVyX2FkZHJlc3MYBiABKAlSEWNvbnRyb2xsZXJBZGRyZXNzEjQKFmNvbnRyb2xsZXJfc2'
    'VydmVyX25hbWUYByABKAlSFGNvbnRyb2xsZXJTZXJ2ZXJOYW1lEk8KDmRlc2lyZWRfY29uZmln'
    'GAggASgLMiguYW55dHR5LmNsb3VkLnYxLlNpZ25lZEVkZ2VEZXNpcmVkQ29uZmlnUg1kZXNpcm'
    'VkQ29uZmlnEiIKDWNvbmZpZ19rZXlfaWQYCSABKAlSC2NvbmZpZ0tleUlkEjkKGWNvbmZpZ19z'
    'aWduaW5nX3B1YmxpY19rZXkYCiABKAxSFmNvbmZpZ1NpZ25pbmdQdWJsaWNLZXlKBAgFEAY=');

@$core.Deprecated('Use createEdgeIdentityRecoveryRequestDescriptor instead')
const CreateEdgeIdentityRecoveryRequest$json = {
  '1': 'CreateEdgeIdentityRecoveryRequest',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'reason', '3': 2, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `CreateEdgeIdentityRecoveryRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createEdgeIdentityRecoveryRequestDescriptor =
    $convert.base64Decode(
        'CiFDcmVhdGVFZGdlSWRlbnRpdHlSZWNvdmVyeVJlcXVlc3QSFwoHZWRnZV9pZBgBIAEoCVIGZW'
        'RnZUlkEhYKBnJlYXNvbhgCIAEoCVIGcmVhc29u');

@$core.Deprecated('Use createEdgeIdentityRecoveryResponseDescriptor instead')
const CreateEdgeIdentityRecoveryResponse$json = {
  '1': 'CreateEdgeIdentityRecoveryResponse',
  '2': [
    {'1': 'recovery_token', '3': 1, '4': 1, '5': 9, '10': 'recoveryToken'},
    {
      '1': 'expires_at',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
  ],
};

/// Descriptor for `CreateEdgeIdentityRecoveryResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createEdgeIdentityRecoveryResponseDescriptor =
    $convert.base64Decode(
        'CiJDcmVhdGVFZGdlSWRlbnRpdHlSZWNvdmVyeVJlc3BvbnNlEiUKDnJlY292ZXJ5X3Rva2VuGA'
        'EgASgJUg1yZWNvdmVyeVRva2VuEjkKCmV4cGlyZXNfYXQYAiABKAsyGi5nb29nbGUucHJvdG9i'
        'dWYuVGltZXN0YW1wUglleHBpcmVzQXQ=');

@$core.Deprecated('Use recoverEdgeIdentityRequestDescriptor instead')
const RecoverEdgeIdentityRequest$json = {
  '1': 'RecoverEdgeIdentityRequest',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'recovery_token', '3': 2, '4': 1, '5': 9, '10': 'recoveryToken'},
    {'1': 'identity_csr_pem', '3': 3, '4': 1, '5': 12, '10': 'identityCsrPem'},
  ],
};

/// Descriptor for `RecoverEdgeIdentityRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List recoverEdgeIdentityRequestDescriptor =
    $convert.base64Decode(
        'ChpSZWNvdmVyRWRnZUlkZW50aXR5UmVxdWVzdBIXCgdlZGdlX2lkGAEgASgJUgZlZGdlSWQSJQ'
        'oOcmVjb3ZlcnlfdG9rZW4YAiABKAlSDXJlY292ZXJ5VG9rZW4SKAoQaWRlbnRpdHlfY3NyX3Bl'
        'bRgDIAEoDFIOaWRlbnRpdHlDc3JQZW0=');

@$core.Deprecated('Use recoverEdgeIdentityResponseDescriptor instead')
const RecoverEdgeIdentityResponse$json = {
  '1': 'RecoverEdgeIdentityResponse',
  '2': [
    {
      '1': 'identity_certificate_pem',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'identityCertificatePem'
    },
    {
      '1': 'certificate_sha256',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'certificateSha256'
    },
    {
      '1': 'not_after',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'notAfter'
    },
  ],
};

/// Descriptor for `RecoverEdgeIdentityResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List recoverEdgeIdentityResponseDescriptor = $convert.base64Decode(
    'ChtSZWNvdmVyRWRnZUlkZW50aXR5UmVzcG9uc2USOAoYaWRlbnRpdHlfY2VydGlmaWNhdGVfcG'
    'VtGAEgASgMUhZpZGVudGl0eUNlcnRpZmljYXRlUGVtEi0KEmNlcnRpZmljYXRlX3NoYTI1NhgC'
    'IAEoDFIRY2VydGlmaWNhdGVTaGEyNTYSNwoJbm90X2FmdGVyGAMgASgLMhouZ29vZ2xlLnByb3'
    'RvYnVmLlRpbWVzdGFtcFIIbm90QWZ0ZXI=');
