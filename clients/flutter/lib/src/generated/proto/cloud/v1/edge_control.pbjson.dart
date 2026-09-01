// This is a generated file - do not edit.
//
// Generated from cloud/v1/edge_control.proto.

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

import 'package:protobuf/well_known_types/google/protobuf/duration.pbjson.dart'
    as $7;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pbjson.dart'
    as $3;

import 'certificate.pbjson.dart' as $0;
import 'common.pbjson.dart' as $1;
import 'edge_config.pbjson.dart' as $6;
import 'enrollment.pbjson.dart' as $4;
import 'runtime.pbjson.dart' as $2;
import 'usage.pbjson.dart' as $5;

@$core.Deprecated('Use edgeCapabilityDescriptor instead')
const EdgeCapability$json = {
  '1': 'EdgeCapability',
  '2': [
    {'1': 'EDGE_CAPABILITY_UNSPECIFIED', '2': 0},
    {'1': 'EDGE_CAPABILITY_CONTROL_STREAM', '2': 1},
    {'1': 'EDGE_CAPABILITY_RELAY', '2': 2},
    {'1': 'EDGE_CAPABILITY_RESERVATION_JOURNAL', '2': 3},
    {'1': 'EDGE_CAPABILITY_PUBLIC_CERTIFICATE_ROTATION', '2': 4},
    {'1': 'EDGE_CAPABILITY_DAEMON_LIFECYCLE_POLICY', '2': 5},
    {'1': 'EDGE_CAPABILITY_DAEMON_EDGE_RESELECTION', '2': 6},
    {'1': 'EDGE_CAPABILITY_IDENTITY_CERTIFICATE_ROTATION', '2': 7},
    {'1': 'EDGE_CAPABILITY_DAEMON_CONNECTION_ADMISSION', '2': 8},
    {'1': 'EDGE_CAPABILITY_RELAY_USAGE_BATCH_V1', '2': 9},
    {'1': 'EDGE_CAPABILITY_RELAY_LOCAL_ADMISSION_V1', '2': 10},
    {'1': 'EDGE_CAPABILITY_SCOPED_DAEMON_STATE_SYNC', '2': 11},
  ],
};

/// Descriptor for `EdgeCapability`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List edgeCapabilityDescriptor = $convert.base64Decode(
    'Cg5FZGdlQ2FwYWJpbGl0eRIfChtFREdFX0NBUEFCSUxJVFlfVU5TUEVDSUZJRUQQABIiCh5FRE'
    'dFX0NBUEFCSUxJVFlfQ09OVFJPTF9TVFJFQU0QARIZChVFREdFX0NBUEFCSUxJVFlfUkVMQVkQ'
    'AhInCiNFREdFX0NBUEFCSUxJVFlfUkVTRVJWQVRJT05fSk9VUk5BTBADEi8KK0VER0VfQ0FQQU'
    'JJTElUWV9QVUJMSUNfQ0VSVElGSUNBVEVfUk9UQVRJT04QBBIrCidFREdFX0NBUEFCSUxJVFlf'
    'REFFTU9OX0xJRkVDWUNMRV9QT0xJQ1kQBRIrCidFREdFX0NBUEFCSUxJVFlfREFFTU9OX0VER0'
    'VfUkVTRUxFQ1RJT04QBhIxCi1FREdFX0NBUEFCSUxJVFlfSURFTlRJVFlfQ0VSVElGSUNBVEVf'
    'Uk9UQVRJT04QBxIvCitFREdFX0NBUEFCSUxJVFlfREFFTU9OX0NPTk5FQ1RJT05fQURNSVNTSU'
    '9OEAgSKAokRURHRV9DQVBBQklMSVRZX1JFTEFZX1VTQUdFX0JBVENIX1YxEAkSLAooRURHRV9D'
    'QVBBQklMSVRZX1JFTEFZX0xPQ0FMX0FETUlTU0lPTl9WMRAKEiwKKEVER0VfQ0FQQUJJTElUWV'
    '9TQ09QRURfREFFTU9OX1NUQVRFX1NZTkMQCw==');

@$core.Deprecated('Use daemonConnectionAdmissionResultDescriptor instead')
const DaemonConnectionAdmissionResult$json = {
  '1': 'DaemonConnectionAdmissionResult',
  '2': [
    {'1': 'DAEMON_CONNECTION_ADMISSION_RESULT_UNSPECIFIED', '2': 0},
    {'1': 'DAEMON_CONNECTION_ADMISSION_RESULT_ADMITTED', '2': 1},
    {'1': 'DAEMON_CONNECTION_ADMISSION_RESULT_RELEASED', '2': 2},
    {'1': 'DAEMON_CONNECTION_ADMISSION_RESULT_LIMIT_REACHED', '2': 3},
    {'1': 'DAEMON_CONNECTION_ADMISSION_RESULT_REJECTED', '2': 4},
    {'1': 'DAEMON_CONNECTION_ADMISSION_RESULT_UNAVAILABLE', '2': 5},
  ],
};

/// Descriptor for `DaemonConnectionAdmissionResult`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List daemonConnectionAdmissionResultDescriptor = $convert.base64Decode(
    'Ch9EYWVtb25Db25uZWN0aW9uQWRtaXNzaW9uUmVzdWx0EjIKLkRBRU1PTl9DT05ORUNUSU9OX0'
    'FETUlTU0lPTl9SRVNVTFRfVU5TUEVDSUZJRUQQABIvCitEQUVNT05fQ09OTkVDVElPTl9BRE1J'
    'U1NJT05fUkVTVUxUX0FETUlUVEVEEAESLworREFFTU9OX0NPTk5FQ1RJT05fQURNSVNTSU9OX1'
    'JFU1VMVF9SRUxFQVNFRBACEjQKMERBRU1PTl9DT05ORUNUSU9OX0FETUlTU0lPTl9SRVNVTFRf'
    'TElNSVRfUkVBQ0hFRBADEi8KK0RBRU1PTl9DT05ORUNUSU9OX0FETUlTU0lPTl9SRVNVTFRfUk'
    'VKRUNURUQQBBIyCi5EQUVNT05fQ09OTkVDVElPTl9BRE1JU1NJT05fUkVTVUxUX1VOQVZBSUxB'
    'QkxFEAU=');

@$core.Deprecated('Use commandResultCodeDescriptor instead')
const CommandResultCode$json = {
  '1': 'CommandResultCode',
  '2': [
    {'1': 'COMMAND_RESULT_CODE_UNSPECIFIED', '2': 0},
    {'1': 'COMMAND_RESULT_CODE_APPLIED', '2': 1},
    {'1': 'COMMAND_RESULT_CODE_REJECTED', '2': 2},
    {'1': 'COMMAND_RESULT_CODE_STALE', '2': 3},
  ],
};

/// Descriptor for `CommandResultCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List commandResultCodeDescriptor = $convert.base64Decode(
    'ChFDb21tYW5kUmVzdWx0Q29kZRIjCh9DT01NQU5EX1JFU1VMVF9DT0RFX1VOU1BFQ0lGSUVEEA'
    'ASHwobQ09NTUFORF9SRVNVTFRfQ09ERV9BUFBMSUVEEAESIAocQ09NTUFORF9SRVNVTFRfQ09E'
    'RV9SRUpFQ1RFRBACEh0KGUNPTU1BTkRfUkVTVUxUX0NPREVfU1RBTEUQAw==');

@$core.Deprecated('Use edgeHelloDescriptor instead')
const EdgeHello$json = {
  '1': 'EdgeHello',
  '2': [
    {'1': 'edge_id', '3': 1, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'software_version', '3': 2, '4': 1, '5': 9, '10': 'softwareVersion'},
    {
      '1': 'capabilities',
      '3': 3,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.EdgeCapability',
      '10': 'capabilities'
    },
    {
      '1': 'desired_config_version',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'desiredConfigVersion'
    },
    {
      '1': 'public_certificate',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePublicCertificateStatus',
      '10': 'publicCertificate'
    },
  ],
  '9': [
    {'1': 6, '2': 7},
  ],
};

/// Descriptor for `EdgeHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeHelloDescriptor = $convert.base64Decode(
    'CglFZGdlSGVsbG8SFwoHZWRnZV9pZBgBIAEoCVIGZWRnZUlkEikKEHNvZnR3YXJlX3ZlcnNpb2'
    '4YAiABKAlSD3NvZnR3YXJlVmVyc2lvbhJDCgxjYXBhYmlsaXRpZXMYAyADKA4yHy5hbnl0dHku'
    'Y2xvdWQudjEuRWRnZUNhcGFiaWxpdHlSDGNhcGFiaWxpdGllcxI0ChZkZXNpcmVkX2NvbmZpZ1'
    '92ZXJzaW9uGAQgASgEUhRkZXNpcmVkQ29uZmlnVmVyc2lvbhJbChJwdWJsaWNfY2VydGlmaWNh'
    'dGUYBSABKAsyLC5hbnl0dHkuY2xvdWQudjEuRWRnZVB1YmxpY0NlcnRpZmljYXRlU3RhdHVzUh'
    'FwdWJsaWNDZXJ0aWZpY2F0ZUoECAYQBw==');

@$core.Deprecated('Use edgeWelcomeDescriptor instead')
const EdgeWelcome$json = {
  '1': 'EdgeWelcome',
  '2': [
    {
      '1': 'accepted_protocol_version',
      '3': 1,
      '4': 1,
      '5': 13,
      '10': 'acceptedProtocolVersion'
    },
    {
      '1': 'heartbeat',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.HeartbeatPolicy',
      '10': 'heartbeat'
    },
    {
      '1': 'binding_key_bundle',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.KeyBundle',
      '10': 'bindingKeyBundle'
    },
  ],
  '9': [
    {'1': 4, '2': 5},
  ],
};

/// Descriptor for `EdgeWelcome`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeWelcomeDescriptor = $convert.base64Decode(
    'CgtFZGdlV2VsY29tZRI6ChlhY2NlcHRlZF9wcm90b2NvbF92ZXJzaW9uGAEgASgNUhdhY2NlcH'
    'RlZFByb3RvY29sVmVyc2lvbhI+CgloZWFydGJlYXQYAiABKAsyIC5hbnl0dHkuY2xvdWQudjEu'
    'SGVhcnRiZWF0UG9saWN5UgloZWFydGJlYXQSSAoSYmluZGluZ19rZXlfYnVuZGxlGAMgASgLMh'
    'ouYW55dHR5LmNsb3VkLnYxLktleUJ1bmRsZVIQYmluZGluZ0tleUJ1bmRsZUoECAQQBQ==');

@$core.Deprecated('Use snapshotBeginDescriptor instead')
const SnapshotBegin$json = {
  '1': 'SnapshotBegin',
  '2': [
    {'1': 'snapshot_id', '3': 1, '4': 1, '5': 9, '10': 'snapshotId'},
    {'1': 'revision', '3': 2, '4': 1, '5': 4, '10': 'revision'},
  ],
};

/// Descriptor for `SnapshotBegin`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List snapshotBeginDescriptor = $convert.base64Decode(
    'Cg1TbmFwc2hvdEJlZ2luEh8KC3NuYXBzaG90X2lkGAEgASgJUgpzbmFwc2hvdElkEhoKCHJldm'
    'lzaW9uGAIgASgEUghyZXZpc2lvbg==');

@$core.Deprecated('Use snapshotChunkDescriptor instead')
const SnapshotChunk$json = {
  '1': 'SnapshotChunk',
  '2': [
    {'1': 'snapshot_id', '3': 1, '4': 1, '5': 9, '10': 'snapshotId'},
    {'1': 'chunk_index', '3': 2, '4': 1, '5': 13, '10': 'chunkIndex'},
    {
      '1': 'agents',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentPresence',
      '10': 'agents'
    },
    {
      '1': 'sessions',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientSessionSummary',
      '10': 'sessions'
    },
  ],
  '9': [
    {'1': 5, '2': 6},
  ],
};

/// Descriptor for `SnapshotChunk`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List snapshotChunkDescriptor = $convert.base64Decode(
    'Cg1TbmFwc2hvdENodW5rEh8KC3NuYXBzaG90X2lkGAEgASgJUgpzbmFwc2hvdElkEh8KC2NodW'
    '5rX2luZGV4GAIgASgNUgpjaHVua0luZGV4EjYKBmFnZW50cxgDIAMoCzIeLmFueXR0eS5jbG91'
    'ZC52MS5BZ2VudFByZXNlbmNlUgZhZ2VudHMSQQoIc2Vzc2lvbnMYBCADKAsyJS5hbnl0dHkuY2'
    'xvdWQudjEuQ2xpZW50U2Vzc2lvblN1bW1hcnlSCHNlc3Npb25zSgQIBRAG');

@$core.Deprecated('Use snapshotEndDescriptor instead')
const SnapshotEnd$json = {
  '1': 'SnapshotEnd',
  '2': [
    {'1': 'snapshot_id', '3': 1, '4': 1, '5': 9, '10': 'snapshotId'},
    {'1': 'revision', '3': 2, '4': 1, '5': 4, '10': 'revision'},
    {'1': 'chunk_count', '3': 3, '4': 1, '5': 13, '10': 'chunkCount'},
    {'1': 'digest', '3': 4, '4': 1, '5': 12, '10': 'digest'},
  ],
};

/// Descriptor for `SnapshotEnd`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List snapshotEndDescriptor = $convert.base64Decode(
    'CgtTbmFwc2hvdEVuZBIfCgtzbmFwc2hvdF9pZBgBIAEoCVIKc25hcHNob3RJZBIaCghyZXZpc2'
    'lvbhgCIAEoBFIIcmV2aXNpb24SHwoLY2h1bmtfY291bnQYAyABKA1SCmNodW5rQ291bnQSFgoG'
    'ZGlnZXN0GAQgASgMUgZkaWdlc3Q=');

@$core.Deprecated('Use edgeHeartbeatDescriptor instead')
const EdgeHeartbeat$json = {
  '1': 'EdgeHeartbeat',
  '2': [
    {'1': 'runtime_revision', '3': 1, '4': 1, '5': 4, '10': 'runtimeRevision'},
  ],
};

/// Descriptor for `EdgeHeartbeat`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeHeartbeatDescriptor = $convert.base64Decode(
    'Cg1FZGdlSGVhcnRiZWF0EikKEHJ1bnRpbWVfcmV2aXNpb24YASABKARSD3J1bnRpbWVSZXZpc2'
    'lvbg==');

@$core.Deprecated('Use snapshotAcceptedDescriptor instead')
const SnapshotAccepted$json = {
  '1': 'SnapshotAccepted',
  '2': [
    {'1': 'snapshot_id', '3': 1, '4': 1, '5': 9, '10': 'snapshotId'},
    {'1': 'revision', '3': 2, '4': 1, '5': 4, '10': 'revision'},
  ],
};

/// Descriptor for `SnapshotAccepted`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List snapshotAcceptedDescriptor = $convert.base64Decode(
    'ChBTbmFwc2hvdEFjY2VwdGVkEh8KC3NuYXBzaG90X2lkGAEgASgJUgpzbmFwc2hvdElkEhoKCH'
    'JldmlzaW9uGAIgASgEUghyZXZpc2lvbg==');

@$core.Deprecated('Use resyncRequiredDescriptor instead')
const ResyncRequired$json = {
  '1': 'ResyncRequired',
  '2': [
    {
      '1': 'expected_revision',
      '3': 1,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {'1': 'reason', '3': 2, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `ResyncRequired`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resyncRequiredDescriptor = $convert.base64Decode(
    'Cg5SZXN5bmNSZXF1aXJlZBIrChFleHBlY3RlZF9yZXZpc2lvbhgBIAEoBFIQZXhwZWN0ZWRSZX'
    'Zpc2lvbhIWCgZyZWFzb24YAiABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use configAppliedDescriptor instead')
const ConfigApplied$json = {
  '1': 'ConfigApplied',
  '2': [
    {'1': 'version', '3': 1, '4': 1, '5': 4, '10': 'version'},
    {'1': 'applied', '3': 2, '4': 1, '5': 8, '10': 'applied'},
    {'1': 'error_code', '3': 3, '4': 1, '5': 9, '10': 'errorCode'},
  ],
};

/// Descriptor for `ConfigApplied`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List configAppliedDescriptor = $convert.base64Decode(
    'Cg1Db25maWdBcHBsaWVkEhgKB3ZlcnNpb24YASABKARSB3ZlcnNpb24SGAoHYXBwbGllZBgCIA'
    'EoCFIHYXBwbGllZBIdCgplcnJvcl9jb2RlGAMgASgJUgllcnJvckNvZGU=');

@$core.Deprecated('Use edgeIdentityRenewRequestDescriptor instead')
const EdgeIdentityRenewRequest$json = {
  '1': 'EdgeIdentityRenewRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'csr_pem', '3': 2, '4': 1, '5': 12, '10': 'csrPem'},
    {
      '1': 'current_certificate_sha256',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'currentCertificateSha256'
    },
    {
      '1': 'requested_at',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'requestedAt'
    },
  ],
};

/// Descriptor for `EdgeIdentityRenewRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeIdentityRenewRequestDescriptor = $convert.base64Decode(
    'ChhFZGdlSWRlbnRpdHlSZW5ld1JlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdE'
    'lkEhcKB2Nzcl9wZW0YAiABKAxSBmNzclBlbRI8ChpjdXJyZW50X2NlcnRpZmljYXRlX3NoYTI1'
    'NhgDIAEoDFIYY3VycmVudENlcnRpZmljYXRlU2hhMjU2Ej0KDHJlcXVlc3RlZF9hdBgEIAEoCz'
    'IaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC3JlcXVlc3RlZEF0');

@$core.Deprecated('Use edgeIdentityRenewResponseDescriptor instead')
const EdgeIdentityRenewResponse$json = {
  '1': 'EdgeIdentityRenewResponse',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'certificate_pem', '3': 2, '4': 1, '5': 12, '10': 'certificatePem'},
    {
      '1': 'certificate_sha256',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'certificateSha256'
    },
    {
      '1': 'not_after',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'notAfter'
    },
  ],
};

/// Descriptor for `EdgeIdentityRenewResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeIdentityRenewResponseDescriptor = $convert.base64Decode(
    'ChlFZGdlSWRlbnRpdHlSZW5ld1Jlc3BvbnNlEh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3'
    'RJZBInCg9jZXJ0aWZpY2F0ZV9wZW0YAiABKAxSDmNlcnRpZmljYXRlUGVtEi0KEmNlcnRpZmlj'
    'YXRlX3NoYTI1NhgDIAEoDFIRY2VydGlmaWNhdGVTaGEyNTYSNwoJbm90X2FmdGVyGAQgASgLMh'
    'ouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIIbm90QWZ0ZXI=');

@$core.Deprecated('Use edgeIdentityAppliedDescriptor instead')
const EdgeIdentityApplied$json = {
  '1': 'EdgeIdentityApplied',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
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
    {'1': 'applied', '3': 4, '4': 1, '5': 8, '10': 'applied'},
    {'1': 'error_code', '3': 5, '4': 1, '5': 9, '10': 'errorCode'},
    {'1': 'error_message', '3': 6, '4': 1, '5': 9, '10': 'errorMessage'},
  ],
};

/// Descriptor for `EdgeIdentityApplied`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeIdentityAppliedDescriptor = $convert.base64Decode(
    'ChNFZGdlSWRlbnRpdHlBcHBsaWVkEh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZBItCh'
    'JjZXJ0aWZpY2F0ZV9zaGEyNTYYAiABKAxSEWNlcnRpZmljYXRlU2hhMjU2EjcKCW5vdF9hZnRl'
    'chgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCG5vdEFmdGVyEhgKB2FwcGxpZW'
    'QYBCABKAhSB2FwcGxpZWQSHQoKZXJyb3JfY29kZRgFIAEoCVIJZXJyb3JDb2RlEiMKDWVycm9y'
    'X21lc3NhZ2UYBiABKAlSDGVycm9yTWVzc2FnZQ==');

@$core.Deprecated('Use daemonStateSnapshotDescriptor instead')
const DaemonStateSnapshot$json = {
  '1': 'DaemonStateSnapshot',
  '2': [
    {
      '1': 'daemons',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemons'
    },
  ],
};

/// Descriptor for `DaemonStateSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateSnapshotDescriptor = $convert.base64Decode(
    'ChNEYWVtb25TdGF0ZVNuYXBzaG90EjwKB2RhZW1vbnMYASADKAsyIi5hbnl0dHkuY2xvdWQudj'
    'EuRGFlbW9uU3RhdGVSZWNvcmRSB2RhZW1vbnM=');

@$core.Deprecated('Use daemonStateSyncRequestDescriptor instead')
const DaemonStateSyncRequest$json = {
  '1': 'DaemonStateSyncRequest',
  '2': [
    {'1': 'sync_id', '3': 1, '4': 1, '5': 9, '10': 'syncId'},
    {'1': 'daemon_ids', '3': 2, '4': 3, '5': 9, '10': 'daemonIds'},
  ],
};

/// Descriptor for `DaemonStateSyncRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateSyncRequestDescriptor =
    $convert.base64Decode(
        'ChZEYWVtb25TdGF0ZVN5bmNSZXF1ZXN0EhcKB3N5bmNfaWQYASABKAlSBnN5bmNJZBIdCgpkYW'
        'Vtb25faWRzGAIgAygJUglkYWVtb25JZHM=');

@$core.Deprecated('Use daemonStateSyncChunkDescriptor instead')
const DaemonStateSyncChunk$json = {
  '1': 'DaemonStateSyncChunk',
  '2': [
    {'1': 'sync_id', '3': 1, '4': 1, '5': 9, '10': 'syncId'},
    {'1': 'chunk_index', '3': 2, '4': 1, '5': 13, '10': 'chunkIndex'},
    {
      '1': 'daemons',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemons'
    },
  ],
};

/// Descriptor for `DaemonStateSyncChunk`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateSyncChunkDescriptor = $convert.base64Decode(
    'ChREYWVtb25TdGF0ZVN5bmNDaHVuaxIXCgdzeW5jX2lkGAEgASgJUgZzeW5jSWQSHwoLY2h1bm'
    'tfaW5kZXgYAiABKA1SCmNodW5rSW5kZXgSPAoHZGFlbW9ucxgDIAMoCzIiLmFueXR0eS5jbG91'
    'ZC52MS5EYWVtb25TdGF0ZVJlY29yZFIHZGFlbW9ucw==');

@$core.Deprecated('Use daemonStateSyncEndDescriptor instead')
const DaemonStateSyncEnd$json = {
  '1': 'DaemonStateSyncEnd',
  '2': [
    {'1': 'sync_id', '3': 1, '4': 1, '5': 9, '10': 'syncId'},
    {'1': 'chunk_count', '3': 2, '4': 1, '5': 13, '10': 'chunkCount'},
  ],
};

/// Descriptor for `DaemonStateSyncEnd`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateSyncEndDescriptor = $convert.base64Decode(
    'ChJEYWVtb25TdGF0ZVN5bmNFbmQSFwoHc3luY19pZBgBIAEoCVIGc3luY0lkEh8KC2NodW5rX2'
    'NvdW50GAIgASgNUgpjaHVua0NvdW50');

@$core.Deprecated('Use daemonStateDeltaDescriptor instead')
const DaemonStateDelta$json = {
  '1': 'DaemonStateDelta',
  '2': [
    {
      '1': 'daemon',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemon'
    },
  ],
};

/// Descriptor for `DaemonStateDelta`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateDeltaDescriptor = $convert.base64Decode(
    'ChBEYWVtb25TdGF0ZURlbHRhEjoKBmRhZW1vbhgBIAEoCzIiLmFueXR0eS5jbG91ZC52MS5EYW'
    'Vtb25TdGF0ZVJlY29yZFIGZGFlbW9u');

@$core.Deprecated('Use daemonStateQueryDescriptor instead')
const DaemonStateQuery$json = {
  '1': 'DaemonStateQuery',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'daemon_id', '3': 2, '4': 1, '5': 9, '10': 'daemonId'},
  ],
};

/// Descriptor for `DaemonStateQuery`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateQueryDescriptor = $convert.base64Decode(
    'ChBEYWVtb25TdGF0ZVF1ZXJ5Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZBIbCglkYW'
    'Vtb25faWQYAiABKAlSCGRhZW1vbklk');

@$core.Deprecated('Use daemonStateQueryResultDescriptor instead')
const DaemonStateQueryResult$json = {
  '1': 'DaemonStateQueryResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'daemon_id', '3': 2, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'found', '3': 3, '4': 1, '5': 8, '10': 'found'},
    {
      '1': 'daemon',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemon'
    },
  ],
};

/// Descriptor for `DaemonStateQueryResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonStateQueryResultDescriptor = $convert.base64Decode(
    'ChZEYWVtb25TdGF0ZVF1ZXJ5UmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZB'
    'IbCglkYWVtb25faWQYAiABKAlSCGRhZW1vbklkEhQKBWZvdW5kGAMgASgIUgVmb3VuZBI6CgZk'
    'YWVtb24YBCABKAsyIi5hbnl0dHkuY2xvdWQudjEuRGFlbW9uU3RhdGVSZWNvcmRSBmRhZW1vbg'
    '==');

@$core.Deprecated('Use daemonConnectionAdmissionRequestDescriptor instead')
const DaemonConnectionAdmissionRequest$json = {
  '1': 'DaemonConnectionAdmissionRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'admission_id', '3': 2, '4': 1, '5': 9, '10': 'admissionId'},
    {'1': 'daemon_id', '3': 3, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'account_id', '3': 4, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'agent_connection_id',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'agentConnectionId'
    },
    {'1': 'release', '3': 6, '4': 1, '5': 8, '10': 'release'},
  ],
};

/// Descriptor for `DaemonConnectionAdmissionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonConnectionAdmissionRequestDescriptor = $convert.base64Decode(
    'CiBEYWVtb25Db25uZWN0aW9uQWRtaXNzaW9uUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUg'
    'lyZXF1ZXN0SWQSIQoMYWRtaXNzaW9uX2lkGAIgASgJUgthZG1pc3Npb25JZBIbCglkYWVtb25f'
    'aWQYAyABKAlSCGRhZW1vbklkEh0KCmFjY291bnRfaWQYBCABKAlSCWFjY291bnRJZBIuChNhZ2'
    'VudF9jb25uZWN0aW9uX2lkGAUgASgJUhFhZ2VudENvbm5lY3Rpb25JZBIYCgdyZWxlYXNlGAYg'
    'ASgIUgdyZWxlYXNl');

@$core.Deprecated('Use daemonConnectionAdmissionResponseDescriptor instead')
const DaemonConnectionAdmissionResponse$json = {
  '1': 'DaemonConnectionAdmissionResponse',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'admission_id', '3': 2, '4': 1, '5': 9, '10': 'admissionId'},
    {
      '1': 'result',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.DaemonConnectionAdmissionResult',
      '10': 'result'
    },
    {'1': 'limit', '3': 4, '4': 1, '5': 13, '10': 'limit'},
    {'1': 'message', '3': 5, '4': 1, '5': 9, '10': 'message'},
    {
      '1': 'entitlement_failure',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudEntitlementFailure',
      '10': 'entitlementFailure'
    },
  ],
};

/// Descriptor for `DaemonConnectionAdmissionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonConnectionAdmissionResponseDescriptor = $convert.base64Decode(
    'CiFEYWVtb25Db25uZWN0aW9uQWRtaXNzaW9uUmVzcG9uc2USHQoKcmVxdWVzdF9pZBgBIAEoCV'
    'IJcmVxdWVzdElkEiEKDGFkbWlzc2lvbl9pZBgCIAEoCVILYWRtaXNzaW9uSWQSSAoGcmVzdWx0'
    'GAMgASgOMjAuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkNvbm5lY3Rpb25BZG1pc3Npb25SZXN1bH'
    'RSBnJlc3VsdBIUCgVsaW1pdBgEIAEoDVIFbGltaXQSGAoHbWVzc2FnZRgFIAEoCVIHbWVzc2Fn'
    'ZRJZChNlbnRpdGxlbWVudF9mYWlsdXJlGAYgASgLMiguYW55dHR5LmNsb3VkLnYxLkNsb3VkRW'
    '50aXRsZW1lbnRGYWlsdXJlUhJlbnRpdGxlbWVudEZhaWx1cmU=');

@$core.Deprecated('Use closeDaemonConnectionDescriptor instead')
const CloseDaemonConnection$json = {
  '1': 'CloseDaemonConnection',
  '2': [
    {'1': 'command_id', '3': 1, '4': 1, '5': 9, '10': 'commandId'},
    {'1': 'correlation_id', '3': 2, '4': 1, '5': 9, '10': 'correlationId'},
    {
      '1': 'deadline',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'deadline'
    },
    {'1': 'daemon_id', '3': 4, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'generation', '3': 5, '4': 1, '5': 4, '10': 'generation'},
    {'1': 'reason', '3': 6, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `CloseDaemonConnection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List closeDaemonConnectionDescriptor = $convert.base64Decode(
    'ChVDbG9zZURhZW1vbkNvbm5lY3Rpb24SHQoKY29tbWFuZF9pZBgBIAEoCVIJY29tbWFuZElkEi'
    'UKDmNvcnJlbGF0aW9uX2lkGAIgASgJUg1jb3JyZWxhdGlvbklkEjYKCGRlYWRsaW5lGAMgASgL'
    'MhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIIZGVhZGxpbmUSGwoJZGFlbW9uX2lkGAQgAS'
    'gJUghkYWVtb25JZBIeCgpnZW5lcmF0aW9uGAUgASgEUgpnZW5lcmF0aW9uEhYKBnJlYXNvbhgG'
    'IAEoCVIGcmVhc29u');

@$core.Deprecated('Use closeClientSessionDescriptor instead')
const CloseClientSession$json = {
  '1': 'CloseClientSession',
  '2': [
    {'1': 'command_id', '3': 1, '4': 1, '5': 9, '10': 'commandId'},
    {'1': 'correlation_id', '3': 2, '4': 1, '5': 9, '10': 'correlationId'},
    {
      '1': 'deadline',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'deadline'
    },
    {'1': 'session_id', '3': 4, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'generation', '3': 5, '4': 1, '5': 4, '10': 'generation'},
    {'1': 'reason', '3': 6, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `CloseClientSession`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List closeClientSessionDescriptor = $convert.base64Decode(
    'ChJDbG9zZUNsaWVudFNlc3Npb24SHQoKY29tbWFuZF9pZBgBIAEoCVIJY29tbWFuZElkEiUKDm'
    'NvcnJlbGF0aW9uX2lkGAIgASgJUg1jb3JyZWxhdGlvbklkEjYKCGRlYWRsaW5lGAMgASgLMhou'
    'Z29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIIZGVhZGxpbmUSHQoKc2Vzc2lvbl9pZBgEIAEoCV'
    'IJc2Vzc2lvbklkEh4KCmdlbmVyYXRpb24YBSABKARSCmdlbmVyYXRpb24SFgoGcmVhc29uGAYg'
    'ASgJUgZyZWFzb24=');

@$core.Deprecated('Use reselectDaemonEdgeDescriptor instead')
const ReselectDaemonEdge$json = {
  '1': 'ReselectDaemonEdge',
  '2': [
    {'1': 'command_id', '3': 1, '4': 1, '5': 9, '10': 'commandId'},
    {'1': 'correlation_id', '3': 2, '4': 1, '5': 9, '10': 'correlationId'},
    {
      '1': 'deadline',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'deadline'
    },
    {'1': 'daemon_id', '3': 4, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'generation', '3': 5, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'preference_revision',
      '3': 6,
      '4': 1,
      '5': 4,
      '10': 'preferenceRevision'
    },
  ],
};

/// Descriptor for `ReselectDaemonEdge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List reselectDaemonEdgeDescriptor = $convert.base64Decode(
    'ChJSZXNlbGVjdERhZW1vbkVkZ2USHQoKY29tbWFuZF9pZBgBIAEoCVIJY29tbWFuZElkEiUKDm'
    'NvcnJlbGF0aW9uX2lkGAIgASgJUg1jb3JyZWxhdGlvbklkEjYKCGRlYWRsaW5lGAMgASgLMhou'
    'Z29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIIZGVhZGxpbmUSGwoJZGFlbW9uX2lkGAQgASgJUg'
    'hkYWVtb25JZBIeCgpnZW5lcmF0aW9uGAUgASgEUgpnZW5lcmF0aW9uEi8KE3ByZWZlcmVuY2Vf'
    'cmV2aXNpb24YBiABKARSEnByZWZlcmVuY2VSZXZpc2lvbg==');

@$core.Deprecated('Use edgeCommandResultDescriptor instead')
const EdgeCommandResult$json = {
  '1': 'EdgeCommandResult',
  '2': [
    {'1': 'command_id', '3': 1, '4': 1, '5': 9, '10': 'commandId'},
    {'1': 'correlation_id', '3': 2, '4': 1, '5': 9, '10': 'correlationId'},
    {
      '1': 'code',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CommandResultCode',
      '10': 'code'
    },
    {'1': 'message', '3': 4, '4': 1, '5': 9, '10': 'message'},
    {
      '1': 'completed_at',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'completedAt'
    },
  ],
};

/// Descriptor for `EdgeCommandResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeCommandResultDescriptor = $convert.base64Decode(
    'ChFFZGdlQ29tbWFuZFJlc3VsdBIdCgpjb21tYW5kX2lkGAEgASgJUgljb21tYW5kSWQSJQoOY2'
    '9ycmVsYXRpb25faWQYAiABKAlSDWNvcnJlbGF0aW9uSWQSNgoEY29kZRgDIAEoDjIiLmFueXR0'
    'eS5jbG91ZC52MS5Db21tYW5kUmVzdWx0Q29kZVIEY29kZRIYCgdtZXNzYWdlGAQgASgJUgdtZX'
    'NzYWdlEj0KDGNvbXBsZXRlZF9hdBgFIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBS'
    'C2NvbXBsZXRlZEF0');

@$core.Deprecated('Use edgeEventDescriptor instead')
const EdgeEvent$json = {
  '1': 'EdgeEvent',
  '2': [
    {'1': 'protocol_version', '3': 1, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'message_id', '3': 2, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'sender_id', '3': 3, '4': 1, '5': 9, '10': 'senderId'},
    {'1': 'boot_id', '3': 4, '4': 1, '5': 9, '10': 'bootId'},
    {'1': 'connection_id', '3': 5, '4': 1, '5': 9, '10': 'connectionId'},
    {'1': 'stream_seq', '3': 6, '4': 1, '5': 4, '10': 'streamSeq'},
    {
      '1': 'sent_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'sentAt'
    },
    {
      '1': 'hello',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeHello',
      '9': 0,
      '10': 'hello'
    },
    {
      '1': 'snapshot_begin',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SnapshotBegin',
      '9': 0,
      '10': 'snapshotBegin'
    },
    {
      '1': 'snapshot_chunk',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SnapshotChunk',
      '9': 0,
      '10': 'snapshotChunk'
    },
    {
      '1': 'snapshot_end',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SnapshotEnd',
      '9': 0,
      '10': 'snapshotEnd'
    },
    {
      '1': 'runtime_delta',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RuntimeDelta',
      '9': 0,
      '10': 'runtimeDelta'
    },
    {
      '1': 'heartbeat',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeHeartbeat',
      '9': 0,
      '10': 'heartbeat'
    },
    {
      '1': 'config_applied',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ConfigApplied',
      '9': 0,
      '10': 'configApplied'
    },
    {
      '1': 'relay_reserve',
      '3': 28,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayReserveRequest',
      '9': 0,
      '10': 'relayReserve'
    },
    {
      '1': 'command_result',
      '3': 29,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeCommandResult',
      '9': 0,
      '10': 'commandResult'
    },
    {
      '1': 'public_certificate_applied',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePublicCertificateApplied',
      '9': 0,
      '10': 'publicCertificateApplied'
    },
    {
      '1': 'relay_renew',
      '3': 31,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayRenewRequest',
      '9': 0,
      '10': 'relayRenew'
    },
    {
      '1': 'relay_settle',
      '3': 32,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelaySettlement',
      '9': 0,
      '10': 'relaySettle'
    },
    {
      '1': 'relay_query',
      '3': 33,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayQueryRequest',
      '9': 0,
      '10': 'relayQuery'
    },
    {
      '1': 'daemon_state_query',
      '3': 34,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateQuery',
      '9': 0,
      '10': 'daemonStateQuery'
    },
    {
      '1': 'identity_renew',
      '3': 35,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeIdentityRenewRequest',
      '9': 0,
      '10': 'identityRenew'
    },
    {
      '1': 'identity_applied',
      '3': 36,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeIdentityApplied',
      '9': 0,
      '10': 'identityApplied'
    },
    {
      '1': 'daemon_connection_admission',
      '3': 37,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonConnectionAdmissionRequest',
      '9': 0,
      '10': 'daemonConnectionAdmission'
    },
    {
      '1': 'relay_authorize',
      '3': 38,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayAuthorizeRequest',
      '9': 0,
      '10': 'relayAuthorize'
    },
    {
      '1': 'relay_usage_batch',
      '3': 39,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayUsageBatch',
      '9': 0,
      '10': 'relayUsageBatch'
    },
    {
      '1': 'daemon_state_sync',
      '3': 40,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateSyncRequest',
      '9': 0,
      '10': 'daemonStateSync'
    },
    {
      '1': 'public_certificate_renew',
      '3': 41,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePublicCertificateRenewRequest',
      '9': 0,
      '10': 'publicCertificateRenew'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `EdgeEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeEventDescriptor = $convert.base64Decode(
    'CglFZGdlRXZlbnQSKQoQcHJvdG9jb2xfdmVyc2lvbhgBIAEoDVIPcHJvdG9jb2xWZXJzaW9uEh'
    '0KCm1lc3NhZ2VfaWQYAiABKAlSCW1lc3NhZ2VJZBIbCglzZW5kZXJfaWQYAyABKAlSCHNlbmRl'
    'cklkEhcKB2Jvb3RfaWQYBCABKAlSBmJvb3RJZBIjCg1jb25uZWN0aW9uX2lkGAUgASgJUgxjb2'
    '5uZWN0aW9uSWQSHQoKc3RyZWFtX3NlcRgGIAEoBFIJc3RyZWFtU2VxEjMKB3NlbnRfYXQYByAB'
    'KAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgZzZW50QXQSMgoFaGVsbG8YFCABKAsyGi'
    '5hbnl0dHkuY2xvdWQudjEuRWRnZUhlbGxvSABSBWhlbGxvEkcKDnNuYXBzaG90X2JlZ2luGBUg'
    'ASgLMh4uYW55dHR5LmNsb3VkLnYxLlNuYXBzaG90QmVnaW5IAFINc25hcHNob3RCZWdpbhJHCg'
    '5zbmFwc2hvdF9jaHVuaxgWIAEoCzIeLmFueXR0eS5jbG91ZC52MS5TbmFwc2hvdENodW5rSABS'
    'DXNuYXBzaG90Q2h1bmsSQQoMc25hcHNob3RfZW5kGBcgASgLMhwuYW55dHR5LmNsb3VkLnYxLl'
    'NuYXBzaG90RW5kSABSC3NuYXBzaG90RW5kEkQKDXJ1bnRpbWVfZGVsdGEYGCABKAsyHS5hbnl0'
    'dHkuY2xvdWQudjEuUnVudGltZURlbHRhSABSDHJ1bnRpbWVEZWx0YRI+CgloZWFydGJlYXQYGS'
    'ABKAsyHi5hbnl0dHkuY2xvdWQudjEuRWRnZUhlYXJ0YmVhdEgAUgloZWFydGJlYXQSRwoOY29u'
    'ZmlnX2FwcGxpZWQYGiABKAsyHi5hbnl0dHkuY2xvdWQudjEuQ29uZmlnQXBwbGllZEgAUg1jb2'
    '5maWdBcHBsaWVkEksKDXJlbGF5X3Jlc2VydmUYHCABKAsyJC5hbnl0dHkuY2xvdWQudjEuUmVs'
    'YXlSZXNlcnZlUmVxdWVzdEgAUgxyZWxheVJlc2VydmUSSwoOY29tbWFuZF9yZXN1bHQYHSABKA'
    'syIi5hbnl0dHkuY2xvdWQudjEuRWRnZUNvbW1hbmRSZXN1bHRIAFINY29tbWFuZFJlc3VsdBJt'
    'ChpwdWJsaWNfY2VydGlmaWNhdGVfYXBwbGllZBgeIAEoCzItLmFueXR0eS5jbG91ZC52MS5FZG'
    'dlUHVibGljQ2VydGlmaWNhdGVBcHBsaWVkSABSGHB1YmxpY0NlcnRpZmljYXRlQXBwbGllZBJF'
    'CgtyZWxheV9yZW5ldxgfIAEoCzIiLmFueXR0eS5jbG91ZC52MS5SZWxheVJlbmV3UmVxdWVzdE'
    'gAUgpyZWxheVJlbmV3EkUKDHJlbGF5X3NldHRsZRggIAEoCzIgLmFueXR0eS5jbG91ZC52MS5S'
    'ZWxheVNldHRsZW1lbnRIAFILcmVsYXlTZXR0bGUSRQoLcmVsYXlfcXVlcnkYISABKAsyIi5hbn'
    'l0dHkuY2xvdWQudjEuUmVsYXlRdWVyeVJlcXVlc3RIAFIKcmVsYXlRdWVyeRJRChJkYWVtb25f'
    'c3RhdGVfcXVlcnkYIiABKAsyIS5hbnl0dHkuY2xvdWQudjEuRGFlbW9uU3RhdGVRdWVyeUgAUh'
    'BkYWVtb25TdGF0ZVF1ZXJ5ElIKDmlkZW50aXR5X3JlbmV3GCMgASgLMikuYW55dHR5LmNsb3Vk'
    'LnYxLkVkZ2VJZGVudGl0eVJlbmV3UmVxdWVzdEgAUg1pZGVudGl0eVJlbmV3ElEKEGlkZW50aX'
    'R5X2FwcGxpZWQYJCABKAsyJC5hbnl0dHkuY2xvdWQudjEuRWRnZUlkZW50aXR5QXBwbGllZEgA'
    'Ug9pZGVudGl0eUFwcGxpZWQScwobZGFlbW9uX2Nvbm5lY3Rpb25fYWRtaXNzaW9uGCUgASgLMj'
    'EuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkNvbm5lY3Rpb25BZG1pc3Npb25SZXF1ZXN0SABSGWRh'
    'ZW1vbkNvbm5lY3Rpb25BZG1pc3Npb24SUQoPcmVsYXlfYXV0aG9yaXplGCYgASgLMiYuYW55dH'
    'R5LmNsb3VkLnYxLlJlbGF5QXV0aG9yaXplUmVxdWVzdEgAUg5yZWxheUF1dGhvcml6ZRJOChFy'
    'ZWxheV91c2FnZV9iYXRjaBgnIAEoCzIgLmFueXR0eS5jbG91ZC52MS5SZWxheVVzYWdlQmF0Y2'
    'hIAFIPcmVsYXlVc2FnZUJhdGNoElUKEWRhZW1vbl9zdGF0ZV9zeW5jGCggASgLMicuYW55dHR5'
    'LmNsb3VkLnYxLkRhZW1vblN0YXRlU3luY1JlcXVlc3RIAFIPZGFlbW9uU3RhdGVTeW5jEm4KGH'
    'B1YmxpY19jZXJ0aWZpY2F0ZV9yZW5ldxgpIAEoCzIyLmFueXR0eS5jbG91ZC52MS5FZGdlUHVi'
    'bGljQ2VydGlmaWNhdGVSZW5ld1JlcXVlc3RIAFIWcHVibGljQ2VydGlmaWNhdGVSZW5ld0IJCg'
    'dwYXlsb2Fk');

@$core.Deprecated('Use controllerCommandDescriptor instead')
const ControllerCommand$json = {
  '1': 'ControllerCommand',
  '2': [
    {'1': 'protocol_version', '3': 1, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'message_id', '3': 2, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'sender_id', '3': 3, '4': 1, '5': 9, '10': 'senderId'},
    {'1': 'boot_id', '3': 4, '4': 1, '5': 9, '10': 'bootId'},
    {'1': 'connection_id', '3': 5, '4': 1, '5': 9, '10': 'connectionId'},
    {'1': 'stream_seq', '3': 6, '4': 1, '5': 4, '10': 'streamSeq'},
    {
      '1': 'sent_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'sentAt'
    },
    {
      '1': 'welcome',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeWelcome',
      '9': 0,
      '10': 'welcome'
    },
    {
      '1': 'snapshot_accepted',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SnapshotAccepted',
      '9': 0,
      '10': 'snapshotAccepted'
    },
    {
      '1': 'resync_required',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ResyncRequired',
      '9': 0,
      '10': 'resyncRequired'
    },
    {
      '1': 'desired_config',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEdgeDesiredConfig',
      '9': 0,
      '10': 'desiredConfig'
    },
    {
      '1': 'binding_key_bundle',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.KeyBundle',
      '9': 0,
      '10': 'bindingKeyBundle'
    },
    {
      '1': 'relay_reserve',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayReserveResponse',
      '9': 0,
      '10': 'relayReserve'
    },
    {
      '1': 'close_daemon',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloseDaemonConnection',
      '9': 0,
      '10': 'closeDaemon'
    },
    {
      '1': 'close_session',
      '3': 27,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloseClientSession',
      '9': 0,
      '10': 'closeSession'
    },
    {
      '1': 'public_certificate_renew',
      '3': 28,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePublicCertificateRenewResponse',
      '9': 0,
      '10': 'publicCertificateRenew'
    },
    {
      '1': 'relay_renew',
      '3': 29,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayRenewResponse',
      '9': 0,
      '10': 'relayRenew'
    },
    {
      '1': 'relay_settle',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelaySettlementAck',
      '9': 0,
      '10': 'relaySettle'
    },
    {
      '1': 'relay_query',
      '3': 31,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayQueryResponse',
      '9': 0,
      '10': 'relayQuery'
    },
    {
      '1': 'daemon_state_delta',
      '3': 32,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateDelta',
      '9': 0,
      '10': 'daemonStateDelta'
    },
    {
      '1': 'daemon_state_query_result',
      '3': 33,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateQueryResult',
      '9': 0,
      '10': 'daemonStateQueryResult'
    },
    {
      '1': 'reselect_daemon_edge',
      '3': 34,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ReselectDaemonEdge',
      '9': 0,
      '10': 'reselectDaemonEdge'
    },
    {
      '1': 'identity_renew',
      '3': 35,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeIdentityRenewResponse',
      '9': 0,
      '10': 'identityRenew'
    },
    {
      '1': 'daemon_connection_admission',
      '3': 36,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonConnectionAdmissionResponse',
      '9': 0,
      '10': 'daemonConnectionAdmission'
    },
    {
      '1': 'relay_authorize',
      '3': 37,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayAuthorizeResponse',
      '9': 0,
      '10': 'relayAuthorize'
    },
    {
      '1': 'relay_usage_ack',
      '3': 38,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayUsageAck',
      '9': 0,
      '10': 'relayUsageAck'
    },
    {
      '1': 'relay_account_action',
      '3': 39,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayAccountAction',
      '9': 0,
      '10': 'relayAccountAction'
    },
    {
      '1': 'daemon_state_sync_chunk',
      '3': 40,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateSyncChunk',
      '9': 0,
      '10': 'daemonStateSyncChunk'
    },
    {
      '1': 'daemon_state_sync_end',
      '3': 41,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateSyncEnd',
      '9': 0,
      '10': 'daemonStateSyncEnd'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `ControllerCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List controllerCommandDescriptor = $convert.base64Decode(
    'ChFDb250cm9sbGVyQ29tbWFuZBIpChBwcm90b2NvbF92ZXJzaW9uGAEgASgNUg9wcm90b2NvbF'
    'ZlcnNpb24SHQoKbWVzc2FnZV9pZBgCIAEoCVIJbWVzc2FnZUlkEhsKCXNlbmRlcl9pZBgDIAEo'
    'CVIIc2VuZGVySWQSFwoHYm9vdF9pZBgEIAEoCVIGYm9vdElkEiMKDWNvbm5lY3Rpb25faWQYBS'
    'ABKAlSDGNvbm5lY3Rpb25JZBIdCgpzdHJlYW1fc2VxGAYgASgEUglzdHJlYW1TZXESMwoHc2Vu'
    'dF9hdBgHIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSBnNlbnRBdBI4Cgd3ZWxjb2'
    '1lGBQgASgLMhwuYW55dHR5LmNsb3VkLnYxLkVkZ2VXZWxjb21lSABSB3dlbGNvbWUSUAoRc25h'
    'cHNob3RfYWNjZXB0ZWQYFSABKAsyIS5hbnl0dHkuY2xvdWQudjEuU25hcHNob3RBY2NlcHRlZE'
    'gAUhBzbmFwc2hvdEFjY2VwdGVkEkoKD3Jlc3luY19yZXF1aXJlZBgWIAEoCzIfLmFueXR0eS5j'
    'bG91ZC52MS5SZXN5bmNSZXF1aXJlZEgAUg5yZXN5bmNSZXF1aXJlZBJRCg5kZXNpcmVkX2Nvbm'
    'ZpZxgXIAEoCzIoLmFueXR0eS5jbG91ZC52MS5TaWduZWRFZGdlRGVzaXJlZENvbmZpZ0gAUg1k'
    'ZXNpcmVkQ29uZmlnEkoKEmJpbmRpbmdfa2V5X2J1bmRsZRgYIAEoCzIaLmFueXR0eS5jbG91ZC'
    '52MS5LZXlCdW5kbGVIAFIQYmluZGluZ0tleUJ1bmRsZRJMCg1yZWxheV9yZXNlcnZlGBkgASgL'
    'MiUuYW55dHR5LmNsb3VkLnYxLlJlbGF5UmVzZXJ2ZVJlc3BvbnNlSABSDHJlbGF5UmVzZXJ2ZR'
    'JLCgxjbG9zZV9kYWVtb24YGiABKAsyJi5hbnl0dHkuY2xvdWQudjEuQ2xvc2VEYWVtb25Db25u'
    'ZWN0aW9uSABSC2Nsb3NlRGFlbW9uEkoKDWNsb3NlX3Nlc3Npb24YGyABKAsyIy5hbnl0dHkuY2'
    'xvdWQudjEuQ2xvc2VDbGllbnRTZXNzaW9uSABSDGNsb3NlU2Vzc2lvbhJvChhwdWJsaWNfY2Vy'
    'dGlmaWNhdGVfcmVuZXcYHCABKAsyMy5hbnl0dHkuY2xvdWQudjEuRWRnZVB1YmxpY0NlcnRpZm'
    'ljYXRlUmVuZXdSZXNwb25zZUgAUhZwdWJsaWNDZXJ0aWZpY2F0ZVJlbmV3EkYKC3JlbGF5X3Jl'
    'bmV3GB0gASgLMiMuYW55dHR5LmNsb3VkLnYxLlJlbGF5UmVuZXdSZXNwb25zZUgAUgpyZWxheV'
    'JlbmV3EkgKDHJlbGF5X3NldHRsZRgeIAEoCzIjLmFueXR0eS5jbG91ZC52MS5SZWxheVNldHRs'
    'ZW1lbnRBY2tIAFILcmVsYXlTZXR0bGUSRgoLcmVsYXlfcXVlcnkYHyABKAsyIy5hbnl0dHkuY2'
    'xvdWQudjEuUmVsYXlRdWVyeVJlc3BvbnNlSABSCnJlbGF5UXVlcnkSUQoSZGFlbW9uX3N0YXRl'
    'X2RlbHRhGCAgASgLMiEuYW55dHR5LmNsb3VkLnYxLkRhZW1vblN0YXRlRGVsdGFIAFIQZGFlbW'
    '9uU3RhdGVEZWx0YRJkChlkYWVtb25fc3RhdGVfcXVlcnlfcmVzdWx0GCEgASgLMicuYW55dHR5'
    'LmNsb3VkLnYxLkRhZW1vblN0YXRlUXVlcnlSZXN1bHRIAFIWZGFlbW9uU3RhdGVRdWVyeVJlc3'
    'VsdBJXChRyZXNlbGVjdF9kYWVtb25fZWRnZRgiIAEoCzIjLmFueXR0eS5jbG91ZC52MS5SZXNl'
    'bGVjdERhZW1vbkVkZ2VIAFIScmVzZWxlY3REYWVtb25FZGdlElMKDmlkZW50aXR5X3JlbmV3GC'
    'MgASgLMiouYW55dHR5LmNsb3VkLnYxLkVkZ2VJZGVudGl0eVJlbmV3UmVzcG9uc2VIAFINaWRl'
    'bnRpdHlSZW5ldxJ0ChtkYWVtb25fY29ubmVjdGlvbl9hZG1pc3Npb24YJCABKAsyMi5hbnl0dH'
    'kuY2xvdWQudjEuRGFlbW9uQ29ubmVjdGlvbkFkbWlzc2lvblJlc3BvbnNlSABSGWRhZW1vbkNv'
    'bm5lY3Rpb25BZG1pc3Npb24SUgoPcmVsYXlfYXV0aG9yaXplGCUgASgLMicuYW55dHR5LmNsb3'
    'VkLnYxLlJlbGF5QXV0aG9yaXplUmVzcG9uc2VIAFIOcmVsYXlBdXRob3JpemUSSAoPcmVsYXlf'
    'dXNhZ2VfYWNrGCYgASgLMh4uYW55dHR5LmNsb3VkLnYxLlJlbGF5VXNhZ2VBY2tIAFINcmVsYX'
    'lVc2FnZUFjaxJXChRyZWxheV9hY2NvdW50X2FjdGlvbhgnIAEoCzIjLmFueXR0eS5jbG91ZC52'
    'MS5SZWxheUFjY291bnRBY3Rpb25IAFIScmVsYXlBY2NvdW50QWN0aW9uEl4KF2RhZW1vbl9zdG'
    'F0ZV9zeW5jX2NodW5rGCggASgLMiUuYW55dHR5LmNsb3VkLnYxLkRhZW1vblN0YXRlU3luY0No'
    'dW5rSABSFGRhZW1vblN0YXRlU3luY0NodW5rElgKFWRhZW1vbl9zdGF0ZV9zeW5jX2VuZBgpIA'
    'EoCzIjLmFueXR0eS5jbG91ZC52MS5EYWVtb25TdGF0ZVN5bmNFbmRIAFISZGFlbW9uU3RhdGVT'
    'eW5jRW5kQgkKB3BheWxvYWQ=');

const $core.Map<$core.String, $core.dynamic> EdgeControlServiceBase$json = {
  '1': 'EdgeControl',
  '2': [
    {
      '1': 'Connect',
      '2': '.anytty.cloud.v1.EdgeEvent',
      '3': '.anytty.cloud.v1.ControllerCommand',
      '5': true,
      '6': true
    },
  ],
};

@$core.Deprecated('Use edgeControlServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    EdgeControlServiceBase$messageJson = {
  '.anytty.cloud.v1.EdgeEvent': EdgeEvent$json,
  '.google.protobuf.Timestamp': $3.Timestamp$json,
  '.anytty.cloud.v1.EdgeHello': EdgeHello$json,
  '.anytty.cloud.v1.EdgePublicCertificateStatus':
      $0.EdgePublicCertificateStatus$json,
  '.anytty.cloud.v1.SnapshotBegin': SnapshotBegin$json,
  '.anytty.cloud.v1.SnapshotChunk': SnapshotChunk$json,
  '.anytty.cloud.v1.AgentPresence': $2.AgentPresence$json,
  '.anytty.cloud.v1.ClientSessionSummary': $2.ClientSessionSummary$json,
  '.anytty.cloud.v1.SnapshotEnd': SnapshotEnd$json,
  '.anytty.cloud.v1.RuntimeDelta': $2.RuntimeDelta$json,
  '.anytty.cloud.v1.AgentRemoved': $2.AgentRemoved$json,
  '.anytty.cloud.v1.ClientSessionRemoved': $2.ClientSessionRemoved$json,
  '.anytty.cloud.v1.EdgeHeartbeat': EdgeHeartbeat$json,
  '.anytty.cloud.v1.ConfigApplied': ConfigApplied$json,
  '.anytty.cloud.v1.RelayReserveRequest': $5.RelayReserveRequest$json,
  '.anytty.cloud.v1.EdgeCommandResult': EdgeCommandResult$json,
  '.anytty.cloud.v1.EdgePublicCertificateApplied':
      $0.EdgePublicCertificateApplied$json,
  '.anytty.cloud.v1.RelayRenewRequest': $5.RelayRenewRequest$json,
  '.anytty.cloud.v1.RelaySettlement': $5.RelaySettlement$json,
  '.anytty.cloud.v1.RelayQueryRequest': $5.RelayQueryRequest$json,
  '.anytty.cloud.v1.DaemonStateQuery': DaemonStateQuery$json,
  '.anytty.cloud.v1.EdgeIdentityRenewRequest': EdgeIdentityRenewRequest$json,
  '.anytty.cloud.v1.EdgeIdentityApplied': EdgeIdentityApplied$json,
  '.anytty.cloud.v1.DaemonConnectionAdmissionRequest':
      DaemonConnectionAdmissionRequest$json,
  '.anytty.cloud.v1.RelayAuthorizeRequest': $5.RelayAuthorizeRequest$json,
  '.anytty.cloud.v1.RelayUsageBatch': $5.RelayUsageBatch$json,
  '.anytty.cloud.v1.RelayUsageSample': $5.RelayUsageSample$json,
  '.anytty.cloud.v1.DaemonStateSyncRequest': DaemonStateSyncRequest$json,
  '.anytty.cloud.v1.EdgePublicCertificateRenewRequest':
      $0.EdgePublicCertificateRenewRequest$json,
  '.anytty.cloud.v1.ControllerCommand': ControllerCommand$json,
  '.anytty.cloud.v1.EdgeWelcome': EdgeWelcome$json,
  '.anytty.cloud.v1.HeartbeatPolicy': $1.HeartbeatPolicy$json,
  '.google.protobuf.Duration': $7.Duration$json,
  '.anytty.cloud.v1.KeyBundle': $1.KeyBundle$json,
  '.anytty.cloud.v1.VerificationKey': $1.VerificationKey$json,
  '.anytty.cloud.v1.SnapshotAccepted': SnapshotAccepted$json,
  '.anytty.cloud.v1.ResyncRequired': ResyncRequired$json,
  '.anytty.cloud.v1.SignedEdgeDesiredConfig': $6.SignedEdgeDesiredConfig$json,
  '.anytty.cloud.v1.RelayReserveResponse': $5.RelayReserveResponse$json,
  '.anytty.cloud.v1.RelayGrant': $5.RelayGrant$json,
  '.anytty.cloud.v1.RelayPolicySnapshot': $5.RelayPolicySnapshot$json,
  '.anytty.cloud.v1.RelaySettlementAck': $5.RelaySettlementAck$json,
  '.anytty.cloud.v1.CloudEntitlementFailure': $1.CloudEntitlementFailure$json,
  '.anytty.cloud.v1.CloseDaemonConnection': CloseDaemonConnection$json,
  '.anytty.cloud.v1.CloseClientSession': CloseClientSession$json,
  '.anytty.cloud.v1.EdgePublicCertificateRenewResponse':
      $0.EdgePublicCertificateRenewResponse$json,
  '.anytty.cloud.v1.RelayRenewResponse': $5.RelayRenewResponse$json,
  '.anytty.cloud.v1.RelayQueryResponse': $5.RelayQueryResponse$json,
  '.anytty.cloud.v1.DaemonStateDelta': DaemonStateDelta$json,
  '.anytty.cloud.v1.DaemonStateRecord': $4.DaemonStateRecord$json,
  '.anytty.cloud.v1.DaemonStateQueryResult': DaemonStateQueryResult$json,
  '.anytty.cloud.v1.ReselectDaemonEdge': ReselectDaemonEdge$json,
  '.anytty.cloud.v1.EdgeIdentityRenewResponse': EdgeIdentityRenewResponse$json,
  '.anytty.cloud.v1.DaemonConnectionAdmissionResponse':
      DaemonConnectionAdmissionResponse$json,
  '.anytty.cloud.v1.RelayAuthorizeResponse': $5.RelayAuthorizeResponse$json,
  '.anytty.cloud.v1.RelayRuntimePolicy': $5.RelayRuntimePolicy$json,
  '.anytty.cloud.v1.RelayUsageAck': $5.RelayUsageAck$json,
  '.anytty.cloud.v1.RelayAccountAction': $5.RelayAccountAction$json,
  '.anytty.cloud.v1.DaemonStateSyncChunk': DaemonStateSyncChunk$json,
  '.anytty.cloud.v1.DaemonStateSyncEnd': DaemonStateSyncEnd$json,
};

/// Descriptor for `EdgeControl`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List edgeControlServiceDescriptor = $convert.base64Decode(
    'CgtFZGdlQ29udHJvbBJNCgdDb25uZWN0EhouYW55dHR5LmNsb3VkLnYxLkVkZ2VFdmVudBoiLm'
    'FueXR0eS5jbG91ZC52MS5Db250cm9sbGVyQ29tbWFuZCgBMAE=');
