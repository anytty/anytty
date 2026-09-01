// This is a generated file - do not edit.
//
// Generated from cloud/v1/operator.proto.

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

import 'account.pbjson.dart' as $1;
import 'commerce.pbjson.dart' as $2;
import 'edge_config.pbjson.dart' as $3;

@$core.Deprecated('Use runtimeCommandResultDescriptor instead')
const RuntimeCommandResult$json = {
  '1': 'RuntimeCommandResult',
  '2': [
    {'1': 'RUNTIME_COMMAND_RESULT_UNSPECIFIED', '2': 0},
    {'1': 'RUNTIME_COMMAND_RESULT_APPLIED', '2': 1},
    {'1': 'RUNTIME_COMMAND_RESULT_REJECTED', '2': 2},
    {'1': 'RUNTIME_COMMAND_RESULT_STALE', '2': 3},
    {'1': 'RUNTIME_COMMAND_RESULT_TIMEOUT', '2': 4},
    {'1': 'RUNTIME_COMMAND_RESULT_UNAVAILABLE', '2': 5},
  ],
};

/// Descriptor for `RuntimeCommandResult`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List runtimeCommandResultDescriptor = $convert.base64Decode(
    'ChRSdW50aW1lQ29tbWFuZFJlc3VsdBImCiJSVU5USU1FX0NPTU1BTkRfUkVTVUxUX1VOU1BFQ0'
    'lGSUVEEAASIgoeUlVOVElNRV9DT01NQU5EX1JFU1VMVF9BUFBMSUVEEAESIwofUlVOVElNRV9D'
    'T01NQU5EX1JFU1VMVF9SRUpFQ1RFRBACEiAKHFJVTlRJTUVfQ09NTUFORF9SRVNVTFRfU1RBTE'
    'UQAxIiCh5SVU5USU1FX0NPTU1BTkRfUkVTVUxUX1RJTUVPVVQQBBImCiJSVU5USU1FX0NPTU1B'
    'TkRfUkVTVUxUX1VOQVZBSUxBQkxFEAU=');

@$core.Deprecated('Use operatorEventOperationDescriptor instead')
const OperatorEventOperation$json = {
  '1': 'OperatorEventOperation',
  '2': [
    {'1': 'OPERATOR_EVENT_OPERATION_UNSPECIFIED', '2': 0},
    {'1': 'OPERATOR_EVENT_OPERATION_UPSERT', '2': 1},
    {'1': 'OPERATOR_EVENT_OPERATION_DELETE', '2': 2},
    {'1': 'OPERATOR_EVENT_OPERATION_RESET', '2': 3},
  ],
};

/// Descriptor for `OperatorEventOperation`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List operatorEventOperationDescriptor = $convert.base64Decode(
    'ChZPcGVyYXRvckV2ZW50T3BlcmF0aW9uEigKJE9QRVJBVE9SX0VWRU5UX09QRVJBVElPTl9VTl'
    'NQRUNJRklFRBAAEiMKH09QRVJBVE9SX0VWRU5UX09QRVJBVElPTl9VUFNFUlQQARIjCh9PUEVS'
    'QVRPUl9FVkVOVF9PUEVSQVRJT05fREVMRVRFEAISIgoeT1BFUkFUT1JfRVZFTlRfT1BFUkFUSU'
    '9OX1JFU0VUEAM=');

@$core.Deprecated('Use pageRequestDescriptor instead')
const PageRequest$json = {
  '1': 'PageRequest',
  '2': [
    {'1': 'page_size', '3': 1, '4': 1, '5': 13, '10': 'pageSize'},
    {'1': 'cursor', '3': 2, '4': 1, '5': 9, '10': 'cursor'},
    {'1': 'query', '3': 3, '4': 1, '5': 9, '10': 'query'},
    {'1': 'sort', '3': 4, '4': 1, '5': 9, '10': 'sort'},
  ],
};

/// Descriptor for `PageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pageRequestDescriptor = $convert.base64Decode(
    'CgtQYWdlUmVxdWVzdBIbCglwYWdlX3NpemUYASABKA1SCHBhZ2VTaXplEhYKBmN1cnNvchgCIA'
    'EoCVIGY3Vyc29yEhQKBXF1ZXJ5GAMgASgJUgVxdWVyeRISCgRzb3J0GAQgASgJUgRzb3J0');

@$core.Deprecated('Use operatorOverviewDescriptor instead')
const OperatorOverview$json = {
  '1': 'OperatorOverview',
  '2': [
    {'1': 'edge_total', '3': 1, '4': 1, '5': 4, '10': 'edgeTotal'},
    {'1': 'edge_online', '3': 2, '4': 1, '5': 4, '10': 'edgeOnline'},
    {'1': 'daemon_total', '3': 3, '4': 1, '5': 4, '10': 'daemonTotal'},
    {'1': 'daemon_online', '3': 4, '4': 1, '5': 4, '10': 'daemonOnline'},
    {
      '1': 'client_session_online',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'clientSessionOnline'
    },
    {
      '1': 'relay_bytes_current_period',
      '3': 8,
      '4': 1,
      '5': 4,
      '10': 'relayBytesCurrentPeriod'
    },
    {
      '1': 'controller_instance_id',
      '3': 9,
      '4': 1,
      '5': 9,
      '10': 'controllerInstanceId'
    },
    {
      '1': 'generated_at',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'generatedAt'
    },
  ],
  '9': [
    {'1': 6, '2': 7},
    {'1': 7, '2': 8},
  ],
};

/// Descriptor for `OperatorOverview`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operatorOverviewDescriptor = $convert.base64Decode(
    'ChBPcGVyYXRvck92ZXJ2aWV3Eh0KCmVkZ2VfdG90YWwYASABKARSCWVkZ2VUb3RhbBIfCgtlZG'
    'dlX29ubGluZRgCIAEoBFIKZWRnZU9ubGluZRIhCgxkYWVtb25fdG90YWwYAyABKARSC2RhZW1v'
    'blRvdGFsEiMKDWRhZW1vbl9vbmxpbmUYBCABKARSDGRhZW1vbk9ubGluZRIyChVjbGllbnRfc2'
    'Vzc2lvbl9vbmxpbmUYBSABKARSE2NsaWVudFNlc3Npb25PbmxpbmUSOwoacmVsYXlfYnl0ZXNf'
    'Y3VycmVudF9wZXJpb2QYCCABKARSF3JlbGF5Qnl0ZXNDdXJyZW50UGVyaW9kEjQKFmNvbnRyb2'
    'xsZXJfaW5zdGFuY2VfaWQYCSABKAlSFGNvbnRyb2xsZXJJbnN0YW5jZUlkEj0KDGdlbmVyYXRl'
    'ZF9hdBgKIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC2dlbmVyYXRlZEF0SgQIBh'
    'AHSgQIBxAI');

@$core.Deprecated('Use accountSummaryDescriptor instead')
const AccountSummary$json = {
  '1': 'AccountSummary',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {
      '1': 'roles',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'roles'
    },
    {'1': 'daemon_count', '3': 3, '4': 1, '5': 4, '10': 'daemonCount'},
    {
      '1': 'subscription',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SubscriptionProjection',
      '10': 'subscription'
    },
    {
      '1': 'entitlement',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EffectiveEntitlement',
      '10': 'entitlement'
    },
    {
      '1': 'usage',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.UsagePeriodProjection',
      '10': 'usage'
    },
  ],
};

/// Descriptor for `AccountSummary`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List accountSummaryDescriptor = $convert.base64Decode(
    'Cg5BY2NvdW50U3VtbWFyeRI5CgdhY2NvdW50GAEgASgLMh8uYW55dHR5LmNsb3VkLnYxLkFjY2'
    '91bnRQcm9maWxlUgdhY2NvdW50EjIKBXJvbGVzGAIgAygOMhwuYW55dHR5LmNsb3VkLnYxLkFj'
    'Y291bnRSb2xlUgVyb2xlcxIhCgxkYWVtb25fY291bnQYAyABKARSC2RhZW1vbkNvdW50EksKDH'
    'N1YnNjcmlwdGlvbhgEIAEoCzInLmFueXR0eS5jbG91ZC52MS5TdWJzY3JpcHRpb25Qcm9qZWN0'
    'aW9uUgxzdWJzY3JpcHRpb24SRwoLZW50aXRsZW1lbnQYBSABKAsyJS5hbnl0dHkuY2xvdWQudj'
    'EuRWZmZWN0aXZlRW50aXRsZW1lbnRSC2VudGl0bGVtZW50EjwKBXVzYWdlGAYgASgLMiYuYW55'
    'dHR5LmNsb3VkLnYxLlVzYWdlUGVyaW9kUHJvamVjdGlvblIFdXNhZ2U=');

@$core.Deprecated('Use runtimeSessionProjectionDescriptor instead')
const RuntimeSessionProjection$json = {
  '1': 'RuntimeSessionProjection',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'daemon_id', '3': 3, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'edge_id', '3': 4, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'client_id', '3': 5, '4': 1, '5': 9, '10': 'clientId'},
    {
      '1': 'product',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.ClientProduct',
      '10': 'product'
    },
    {'1': 'generation', '3': 8, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'connected_at',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'connectedAt'
    },
    {
      '1': 'account_display_name',
      '3': 10,
      '4': 1,
      '5': 9,
      '10': 'accountDisplayName'
    },
    {'1': 'account_email', '3': 11, '4': 1, '5': 9, '10': 'accountEmail'},
    {
      '1': 'daemon_display_name',
      '3': 12,
      '4': 1,
      '5': 9,
      '10': 'daemonDisplayName'
    },
    {'1': 'edge_name', '3': 13, '4': 1, '5': 9, '10': 'edgeName'},
  ],
  '9': [
    {'1': 7, '2': 8},
  ],
};

/// Descriptor for `RuntimeSessionProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeSessionProjectionDescriptor = $convert.base64Decode(
    'ChhSdW50aW1lU2Vzc2lvblByb2plY3Rpb24SHQoKc2Vzc2lvbl9pZBgBIAEoCVIJc2Vzc2lvbk'
    'lkEh0KCmFjY291bnRfaWQYAiABKAlSCWFjY291bnRJZBIbCglkYWVtb25faWQYAyABKAlSCGRh'
    'ZW1vbklkEhcKB2VkZ2VfaWQYBCABKAlSBmVkZ2VJZBIbCgljbGllbnRfaWQYBSABKAlSCGNsaW'
    'VudElkEjgKB3Byb2R1Y3QYBiABKA4yHi5hbnl0dHkuY2xvdWQudjEuQ2xpZW50UHJvZHVjdFIH'
    'cHJvZHVjdBIeCgpnZW5lcmF0aW9uGAggASgEUgpnZW5lcmF0aW9uEj0KDGNvbm5lY3RlZF9hdB'
    'gJIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC2Nvbm5lY3RlZEF0EjAKFGFjY291'
    'bnRfZGlzcGxheV9uYW1lGAogASgJUhJhY2NvdW50RGlzcGxheU5hbWUSIwoNYWNjb3VudF9lbW'
    'FpbBgLIAEoCVIMYWNjb3VudEVtYWlsEi4KE2RhZW1vbl9kaXNwbGF5X25hbWUYDCABKAlSEWRh'
    'ZW1vbkRpc3BsYXlOYW1lEhsKCWVkZ2VfbmFtZRgNIAEoCVIIZWRnZU5hbWVKBAgHEAg=');

@$core.Deprecated('Use operatorAuditEventDescriptor instead')
const OperatorAuditEvent$json = {
  '1': 'OperatorAuditEvent',
  '2': [
    {'1': 'audit_id', '3': 1, '4': 1, '5': 9, '10': 'auditId'},
    {'1': 'actor_account_id', '3': 2, '4': 1, '5': 9, '10': 'actorAccountId'},
    {
      '1': 'actor_display_name',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'actorDisplayName'
    },
    {'1': 'action', '3': 4, '4': 1, '5': 9, '10': 'action'},
    {'1': 'resource_type', '3': 5, '4': 1, '5': 9, '10': 'resourceType'},
    {'1': 'resource_id', '3': 6, '4': 1, '5': 9, '10': 'resourceId'},
    {'1': 'reason', '3': 7, '4': 1, '5': 9, '10': 'reason'},
    {'1': 'result', '3': 8, '4': 1, '5': 9, '10': 'result'},
    {'1': 'correlation_id', '3': 9, '4': 1, '5': 9, '10': 'correlationId'},
    {
      '1': 'occurred_at',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'occurredAt'
    },
  ],
};

/// Descriptor for `OperatorAuditEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operatorAuditEventDescriptor = $convert.base64Decode(
    'ChJPcGVyYXRvckF1ZGl0RXZlbnQSGQoIYXVkaXRfaWQYASABKAlSB2F1ZGl0SWQSKAoQYWN0b3'
    'JfYWNjb3VudF9pZBgCIAEoCVIOYWN0b3JBY2NvdW50SWQSLAoSYWN0b3JfZGlzcGxheV9uYW1l'
    'GAMgASgJUhBhY3RvckRpc3BsYXlOYW1lEhYKBmFjdGlvbhgEIAEoCVIGYWN0aW9uEiMKDXJlc2'
    '91cmNlX3R5cGUYBSABKAlSDHJlc291cmNlVHlwZRIfCgtyZXNvdXJjZV9pZBgGIAEoCVIKcmVz'
    'b3VyY2VJZBIWCgZyZWFzb24YByABKAlSBnJlYXNvbhIWCgZyZXN1bHQYCCABKAlSBnJlc3VsdB'
    'IlCg5jb3JyZWxhdGlvbl9pZBgJIAEoCVINY29ycmVsYXRpb25JZBI7CgtvY2N1cnJlZF9hdBgK'
    'IAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCm9jY3VycmVkQXQ=');

@$core.Deprecated('Use getOperatorOverviewRequestDescriptor instead')
const GetOperatorOverviewRequest$json = {
  '1': 'GetOperatorOverviewRequest',
};

/// Descriptor for `GetOperatorOverviewRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOperatorOverviewRequestDescriptor =
    $convert.base64Decode('ChpHZXRPcGVyYXRvck92ZXJ2aWV3UmVxdWVzdA==');

@$core.Deprecated('Use getOperatorOverviewResponseDescriptor instead')
const GetOperatorOverviewResponse$json = {
  '1': 'GetOperatorOverviewResponse',
  '2': [
    {
      '1': 'overview',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.OperatorOverview',
      '10': 'overview'
    },
  ],
};

/// Descriptor for `GetOperatorOverviewResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOperatorOverviewResponseDescriptor =
    $convert.base64Decode(
        'ChtHZXRPcGVyYXRvck92ZXJ2aWV3UmVzcG9uc2USPQoIb3ZlcnZpZXcYASABKAsyIS5hbnl0dH'
        'kuY2xvdWQudjEuT3BlcmF0b3JPdmVydmlld1IIb3ZlcnZpZXc=');

@$core.Deprecated('Use listOperatorAccountsRequestDescriptor instead')
const ListOperatorAccountsRequest$json = {
  '1': 'ListOperatorAccountsRequest',
  '2': [
    {
      '1': 'page',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PageRequest',
      '10': 'page'
    },
  ],
};

/// Descriptor for `ListOperatorAccountsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorAccountsRequestDescriptor =
    $convert.base64Decode(
        'ChtMaXN0T3BlcmF0b3JBY2NvdW50c1JlcXVlc3QSMAoEcGFnZRgBIAEoCzIcLmFueXR0eS5jbG'
        '91ZC52MS5QYWdlUmVxdWVzdFIEcGFnZQ==');

@$core.Deprecated('Use listOperatorAccountsResponseDescriptor instead')
const ListOperatorAccountsResponse$json = {
  '1': 'ListOperatorAccountsResponse',
  '2': [
    {
      '1': 'accounts',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountSummary',
      '10': 'accounts'
    },
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListOperatorAccountsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorAccountsResponseDescriptor =
    $convert.base64Decode(
        'ChxMaXN0T3BlcmF0b3JBY2NvdW50c1Jlc3BvbnNlEjsKCGFjY291bnRzGAEgAygLMh8uYW55dH'
        'R5LmNsb3VkLnYxLkFjY291bnRTdW1tYXJ5UghhY2NvdW50cxIfCgtuZXh0X2N1cnNvchgCIAEo'
        'CVIKbmV4dEN1cnNvcg==');

@$core.Deprecated('Use getOperatorAccountRequestDescriptor instead')
const GetOperatorAccountRequest$json = {
  '1': 'GetOperatorAccountRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
  ],
};

/// Descriptor for `GetOperatorAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOperatorAccountRequestDescriptor =
    $convert.base64Decode(
        'ChlHZXRPcGVyYXRvckFjY291bnRSZXF1ZXN0Eh0KCmFjY291bnRfaWQYASABKAlSCWFjY291bn'
        'RJZA==');

@$core.Deprecated('Use getOperatorAccountResponseDescriptor instead')
const GetOperatorAccountResponse$json = {
  '1': 'GetOperatorAccountResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountSummary',
      '10': 'account'
    },
  ],
};

/// Descriptor for `GetOperatorAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOperatorAccountResponseDescriptor =
    $convert.base64Decode(
        'ChpHZXRPcGVyYXRvckFjY291bnRSZXNwb25zZRI5CgdhY2NvdW50GAEgASgLMh8uYW55dHR5Lm'
        'Nsb3VkLnYxLkFjY291bnRTdW1tYXJ5UgdhY2NvdW50');

@$core.Deprecated('Use provisionAccountRequestDescriptor instead')
const ProvisionAccountRequest$json = {
  '1': 'ProvisionAccountRequest',
  '2': [
    {'1': 'email', '3': 1, '4': 1, '5': 9, '10': 'email'},
    {'1': 'display_name', '3': 2, '4': 1, '5': 9, '10': 'displayName'},
    {'1': 'reason', '3': 3, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `ProvisionAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List provisionAccountRequestDescriptor = $convert.base64Decode(
    'ChdQcm92aXNpb25BY2NvdW50UmVxdWVzdBIUCgVlbWFpbBgBIAEoCVIFZW1haWwSIQoMZGlzcG'
    'xheV9uYW1lGAIgASgJUgtkaXNwbGF5TmFtZRIWCgZyZWFzb24YAyABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use provisionAccountResponseDescriptor instead')
const ProvisionAccountResponse$json = {
  '1': 'ProvisionAccountResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {'1': 'setup_credential', '3': 2, '4': 1, '5': 9, '10': 'setupCredential'},
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

/// Descriptor for `ProvisionAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List provisionAccountResponseDescriptor = $convert.base64Decode(
    'ChhQcm92aXNpb25BY2NvdW50UmVzcG9uc2USOQoHYWNjb3VudBgBIAEoCzIfLmFueXR0eS5jbG'
    '91ZC52MS5BY2NvdW50UHJvZmlsZVIHYWNjb3VudBIpChBzZXR1cF9jcmVkZW50aWFsGAIgASgJ'
    'Ug9zZXR1cENyZWRlbnRpYWwSOQoKZXhwaXJlc19hdBgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi'
    '5UaW1lc3RhbXBSCWV4cGlyZXNBdA==');

@$core.Deprecated('Use resetAccountSetupRequestDescriptor instead')
const ResetAccountSetupRequest$json = {
  '1': 'ResetAccountSetupRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'reason', '3': 2, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `ResetAccountSetupRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resetAccountSetupRequestDescriptor =
    $convert.base64Decode(
        'ChhSZXNldEFjY291bnRTZXR1cFJlcXVlc3QSHQoKYWNjb3VudF9pZBgBIAEoCVIJYWNjb3VudE'
        'lkEhYKBnJlYXNvbhgCIAEoCVIGcmVhc29u');

@$core.Deprecated('Use resetAccountSetupResponseDescriptor instead')
const ResetAccountSetupResponse$json = {
  '1': 'ResetAccountSetupResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {'1': 'setup_credential', '3': 2, '4': 1, '5': 9, '10': 'setupCredential'},
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

/// Descriptor for `ResetAccountSetupResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resetAccountSetupResponseDescriptor = $convert.base64Decode(
    'ChlSZXNldEFjY291bnRTZXR1cFJlc3BvbnNlEjkKB2FjY291bnQYASABKAsyHy5hbnl0dHkuY2'
    'xvdWQudjEuQWNjb3VudFByb2ZpbGVSB2FjY291bnQSKQoQc2V0dXBfY3JlZGVudGlhbBgCIAEo'
    'CVIPc2V0dXBDcmVkZW50aWFsEjkKCmV4cGlyZXNfYXQYAyABKAsyGi5nb29nbGUucHJvdG9idW'
    'YuVGltZXN0YW1wUglleHBpcmVzQXQ=');

@$core.Deprecated('Use listRuntimeSessionsRequestDescriptor instead')
const ListRuntimeSessionsRequest$json = {
  '1': 'ListRuntimeSessionsRequest',
  '2': [
    {
      '1': 'page',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PageRequest',
      '10': 'page'
    },
  ],
};

/// Descriptor for `ListRuntimeSessionsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listRuntimeSessionsRequestDescriptor =
    $convert.base64Decode(
        'ChpMaXN0UnVudGltZVNlc3Npb25zUmVxdWVzdBIwCgRwYWdlGAEgASgLMhwuYW55dHR5LmNsb3'
        'VkLnYxLlBhZ2VSZXF1ZXN0UgRwYWdl');

@$core.Deprecated('Use listRuntimeSessionsResponseDescriptor instead')
const ListRuntimeSessionsResponse$json = {
  '1': 'ListRuntimeSessionsResponse',
  '2': [
    {
      '1': 'sessions',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.RuntimeSessionProjection',
      '10': 'sessions'
    },
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListRuntimeSessionsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listRuntimeSessionsResponseDescriptor =
    $convert.base64Decode(
        'ChtMaXN0UnVudGltZVNlc3Npb25zUmVzcG9uc2USRQoIc2Vzc2lvbnMYASADKAsyKS5hbnl0dH'
        'kuY2xvdWQudjEuUnVudGltZVNlc3Npb25Qcm9qZWN0aW9uUghzZXNzaW9ucxIfCgtuZXh0X2N1'
        'cnNvchgCIAEoCVIKbmV4dEN1cnNvcg==');

@$core.Deprecated('Use listOperatorOrdersRequestDescriptor instead')
const ListOperatorOrdersRequest$json = {
  '1': 'ListOperatorOrdersRequest',
  '2': [
    {
      '1': 'page',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PageRequest',
      '10': 'page'
    },
  ],
};

/// Descriptor for `ListOperatorOrdersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorOrdersRequestDescriptor =
    $convert.base64Decode(
        'ChlMaXN0T3BlcmF0b3JPcmRlcnNSZXF1ZXN0EjAKBHBhZ2UYASABKAsyHC5hbnl0dHkuY2xvdW'
        'QudjEuUGFnZVJlcXVlc3RSBHBhZ2U=');

@$core.Deprecated('Use listOperatorOrdersResponseDescriptor instead')
const ListOperatorOrdersResponse$json = {
  '1': 'ListOperatorOrdersResponse',
  '2': [
    {
      '1': 'orders',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.OrderProjection',
      '10': 'orders'
    },
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListOperatorOrdersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorOrdersResponseDescriptor =
    $convert.base64Decode(
        'ChpMaXN0T3BlcmF0b3JPcmRlcnNSZXNwb25zZRI4CgZvcmRlcnMYASADKAsyIC5hbnl0dHkuY2'
        'xvdWQudjEuT3JkZXJQcm9qZWN0aW9uUgZvcmRlcnMSHwoLbmV4dF9jdXJzb3IYAiABKAlSCm5l'
        'eHRDdXJzb3I=');

@$core.Deprecated('Use listOperatorSubscriptionsRequestDescriptor instead')
const ListOperatorSubscriptionsRequest$json = {
  '1': 'ListOperatorSubscriptionsRequest',
  '2': [
    {
      '1': 'page',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PageRequest',
      '10': 'page'
    },
  ],
};

/// Descriptor for `ListOperatorSubscriptionsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorSubscriptionsRequestDescriptor =
    $convert.base64Decode(
        'CiBMaXN0T3BlcmF0b3JTdWJzY3JpcHRpb25zUmVxdWVzdBIwCgRwYWdlGAEgASgLMhwuYW55dH'
        'R5LmNsb3VkLnYxLlBhZ2VSZXF1ZXN0UgRwYWdl');

@$core.Deprecated('Use listOperatorSubscriptionsResponseDescriptor instead')
const ListOperatorSubscriptionsResponse$json = {
  '1': 'ListOperatorSubscriptionsResponse',
  '2': [
    {
      '1': 'subscriptions',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.SubscriptionProjection',
      '10': 'subscriptions'
    },
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListOperatorSubscriptionsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorSubscriptionsResponseDescriptor =
    $convert.base64Decode(
        'CiFMaXN0T3BlcmF0b3JTdWJzY3JpcHRpb25zUmVzcG9uc2USTQoNc3Vic2NyaXB0aW9ucxgBIA'
        'MoCzInLmFueXR0eS5jbG91ZC52MS5TdWJzY3JpcHRpb25Qcm9qZWN0aW9uUg1zdWJzY3JpcHRp'
        'b25zEh8KC25leHRfY3Vyc29yGAIgASgJUgpuZXh0Q3Vyc29y');

@$core.Deprecated('Use listOperatorUsageRequestDescriptor instead')
const ListOperatorUsageRequest$json = {
  '1': 'ListOperatorUsageRequest',
  '2': [
    {
      '1': 'page',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PageRequest',
      '10': 'page'
    },
  ],
};

/// Descriptor for `ListOperatorUsageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorUsageRequestDescriptor =
    $convert.base64Decode(
        'ChhMaXN0T3BlcmF0b3JVc2FnZVJlcXVlc3QSMAoEcGFnZRgBIAEoCzIcLmFueXR0eS5jbG91ZC'
        '52MS5QYWdlUmVxdWVzdFIEcGFnZQ==');

@$core.Deprecated('Use listOperatorUsageResponseDescriptor instead')
const ListOperatorUsageResponse$json = {
  '1': 'ListOperatorUsageResponse',
  '2': [
    {
      '1': 'accounts',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.UsagePeriodProjection',
      '10': 'accounts'
    },
    {'1': 'next_cursor', '3': 3, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
  '9': [
    {'1': 2, '2': 3},
  ],
};

/// Descriptor for `ListOperatorUsageResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorUsageResponseDescriptor = $convert.base64Decode(
    'ChlMaXN0T3BlcmF0b3JVc2FnZVJlc3BvbnNlEkIKCGFjY291bnRzGAEgAygLMiYuYW55dHR5Lm'
    'Nsb3VkLnYxLlVzYWdlUGVyaW9kUHJvamVjdGlvblIIYWNjb3VudHMSHwoLbmV4dF9jdXJzb3IY'
    'AyABKAlSCm5leHRDdXJzb3JKBAgCEAM=');

@$core.Deprecated('Use listOperatorAuditRequestDescriptor instead')
const ListOperatorAuditRequest$json = {
  '1': 'ListOperatorAuditRequest',
  '2': [
    {
      '1': 'page',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PageRequest',
      '10': 'page'
    },
  ],
};

/// Descriptor for `ListOperatorAuditRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorAuditRequestDescriptor =
    $convert.base64Decode(
        'ChhMaXN0T3BlcmF0b3JBdWRpdFJlcXVlc3QSMAoEcGFnZRgBIAEoCzIcLmFueXR0eS5jbG91ZC'
        '52MS5QYWdlUmVxdWVzdFIEcGFnZQ==');

@$core.Deprecated('Use listOperatorAuditResponseDescriptor instead')
const ListOperatorAuditResponse$json = {
  '1': 'ListOperatorAuditResponse',
  '2': [
    {
      '1': 'events',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.OperatorAuditEvent',
      '10': 'events'
    },
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListOperatorAuditResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperatorAuditResponseDescriptor = $convert.base64Decode(
    'ChlMaXN0T3BlcmF0b3JBdWRpdFJlc3BvbnNlEjsKBmV2ZW50cxgBIAMoCzIjLmFueXR0eS5jbG'
    '91ZC52MS5PcGVyYXRvckF1ZGl0RXZlbnRSBmV2ZW50cxIfCgtuZXh0X2N1cnNvchgCIAEoCVIK'
    'bmV4dEN1cnNvcg==');

@$core.Deprecated('Use setAccountStateRequestDescriptor instead')
const SetAccountStateRequest$json = {
  '1': 'SetAccountStateRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'state',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountState',
      '10': 'state'
    },
    {
      '1': 'expected_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {'1': 'reason', '3': 4, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `SetAccountStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setAccountStateRequestDescriptor = $convert.base64Decode(
    'ChZTZXRBY2NvdW50U3RhdGVSZXF1ZXN0Eh0KCmFjY291bnRfaWQYASABKAlSCWFjY291bnRJZB'
    'IzCgVzdGF0ZRgCIAEoDjIdLmFueXR0eS5jbG91ZC52MS5BY2NvdW50U3RhdGVSBXN0YXRlEisK'
    'EWV4cGVjdGVkX3JldmlzaW9uGAMgASgEUhBleHBlY3RlZFJldmlzaW9uEhYKBnJlYXNvbhgEIA'
    'EoCVIGcmVhc29u');

@$core.Deprecated('Use setAccountStateResponseDescriptor instead')
const SetAccountStateResponse$json = {
  '1': 'SetAccountStateResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
  ],
};

/// Descriptor for `SetAccountStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setAccountStateResponseDescriptor =
    $convert.base64Decode(
        'ChdTZXRBY2NvdW50U3RhdGVSZXNwb25zZRI5CgdhY2NvdW50GAEgASgLMh8uYW55dHR5LmNsb3'
        'VkLnYxLkFjY291bnRQcm9maWxlUgdhY2NvdW50');

@$core.Deprecated('Use setAccountRoleRequestDescriptor instead')
const SetAccountRoleRequest$json = {
  '1': 'SetAccountRoleRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'role',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'role'
    },
    {'1': 'enabled', '3': 3, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'reason', '3': 4, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `SetAccountRoleRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setAccountRoleRequestDescriptor = $convert.base64Decode(
    'ChVTZXRBY2NvdW50Um9sZVJlcXVlc3QSHQoKYWNjb3VudF9pZBgBIAEoCVIJYWNjb3VudElkEj'
    'AKBHJvbGUYAiABKA4yHC5hbnl0dHkuY2xvdWQudjEuQWNjb3VudFJvbGVSBHJvbGUSGAoHZW5h'
    'YmxlZBgDIAEoCFIHZW5hYmxlZBIWCgZyZWFzb24YBCABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use setAccountRoleResponseDescriptor instead')
const SetAccountRoleResponse$json = {
  '1': 'SetAccountRoleResponse',
  '2': [
    {
      '1': 'roles',
      '3': 1,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'roles'
    },
  ],
};

/// Descriptor for `SetAccountRoleResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setAccountRoleResponseDescriptor =
    $convert.base64Decode(
        'ChZTZXRBY2NvdW50Um9sZVJlc3BvbnNlEjIKBXJvbGVzGAEgAygOMhwuYW55dHR5LmNsb3VkLn'
        'YxLkFjY291bnRSb2xlUgVyb2xlcw==');

@$core.Deprecated('Use disconnectDaemonRequestDescriptor instead')
const DisconnectDaemonRequest$json = {
  '1': 'DisconnectDaemonRequest',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
    {'1': 'reason', '3': 3, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `DisconnectDaemonRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List disconnectDaemonRequestDescriptor = $convert.base64Decode(
    'ChdEaXNjb25uZWN0RGFlbW9uUmVxdWVzdBIbCglkYWVtb25faWQYASABKAlSCGRhZW1vbklkEh'
    '4KCmdlbmVyYXRpb24YAiABKARSCmdlbmVyYXRpb24SFgoGcmVhc29uGAMgASgJUgZyZWFzb24=');

@$core.Deprecated('Use disconnectDaemonResponseDescriptor instead')
const DisconnectDaemonResponse$json = {
  '1': 'DisconnectDaemonResponse',
  '2': [
    {
      '1': 'result',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RuntimeCommandResult',
      '10': 'result'
    },
  ],
};

/// Descriptor for `DisconnectDaemonResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List disconnectDaemonResponseDescriptor =
    $convert.base64Decode(
        'ChhEaXNjb25uZWN0RGFlbW9uUmVzcG9uc2USPQoGcmVzdWx0GAEgASgOMiUuYW55dHR5LmNsb3'
        'VkLnYxLlJ1bnRpbWVDb21tYW5kUmVzdWx0UgZyZXN1bHQ=');

@$core.Deprecated('Use disconnectSessionRequestDescriptor instead')
const DisconnectSessionRequest$json = {
  '1': 'DisconnectSessionRequest',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
    {'1': 'reason', '3': 3, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `DisconnectSessionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List disconnectSessionRequestDescriptor = $convert.base64Decode(
    'ChhEaXNjb25uZWN0U2Vzc2lvblJlcXVlc3QSHQoKc2Vzc2lvbl9pZBgBIAEoCVIJc2Vzc2lvbk'
    'lkEh4KCmdlbmVyYXRpb24YAiABKARSCmdlbmVyYXRpb24SFgoGcmVhc29uGAMgASgJUgZyZWFz'
    'b24=');

@$core.Deprecated('Use disconnectSessionResponseDescriptor instead')
const DisconnectSessionResponse$json = {
  '1': 'DisconnectSessionResponse',
  '2': [
    {
      '1': 'result',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RuntimeCommandResult',
      '10': 'result'
    },
  ],
};

/// Descriptor for `DisconnectSessionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List disconnectSessionResponseDescriptor =
    $convert.base64Decode(
        'ChlEaXNjb25uZWN0U2Vzc2lvblJlc3BvbnNlEj0KBnJlc3VsdBgBIAEoDjIlLmFueXR0eS5jbG'
        '91ZC52MS5SdW50aW1lQ29tbWFuZFJlc3VsdFIGcmVzdWx0');

@$core.Deprecated('Use operatorRuntimeEventDescriptor instead')
const OperatorRuntimeEvent$json = {
  '1': 'OperatorRuntimeEvent',
  '2': [
    {
      '1': 'controller_instance_id',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'controllerInstanceId'
    },
    {'1': 'event_seq', '3': 2, '4': 1, '5': 4, '10': 'eventSeq'},
    {'1': 'resource_kind', '3': 3, '4': 1, '5': 9, '10': 'resourceKind'},
    {'1': 'resource_id', '3': 4, '4': 1, '5': 9, '10': 'resourceId'},
    {
      '1': 'operation',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.OperatorEventOperation',
      '10': 'operation'
    },
    {
      '1': 'occurred_at',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'occurredAt'
    },
  ],
};

/// Descriptor for `OperatorRuntimeEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operatorRuntimeEventDescriptor = $convert.base64Decode(
    'ChRPcGVyYXRvclJ1bnRpbWVFdmVudBI0ChZjb250cm9sbGVyX2luc3RhbmNlX2lkGAEgASgJUh'
    'Rjb250cm9sbGVySW5zdGFuY2VJZBIbCglldmVudF9zZXEYAiABKARSCGV2ZW50U2VxEiMKDXJl'
    'c291cmNlX2tpbmQYAyABKAlSDHJlc291cmNlS2luZBIfCgtyZXNvdXJjZV9pZBgEIAEoCVIKcm'
    'Vzb3VyY2VJZBJFCglvcGVyYXRpb24YBSABKA4yJy5hbnl0dHkuY2xvdWQudjEuT3BlcmF0b3JF'
    'dmVudE9wZXJhdGlvblIJb3BlcmF0aW9uEjsKC29jY3VycmVkX2F0GAYgASgLMhouZ29vZ2xlLn'
    'Byb3RvYnVmLlRpbWVzdGFtcFIKb2NjdXJyZWRBdA==');

const $core.Map<$core.String, $core.dynamic> OperatorServiceBase$json = {
  '1': 'OperatorService',
  '2': [
    {
      '1': 'GetOverview',
      '2': '.anytty.cloud.v1.GetOperatorOverviewRequest',
      '3': '.anytty.cloud.v1.GetOperatorOverviewResponse'
    },
    {
      '1': 'ListAccounts',
      '2': '.anytty.cloud.v1.ListOperatorAccountsRequest',
      '3': '.anytty.cloud.v1.ListOperatorAccountsResponse'
    },
    {
      '1': 'GetAccount',
      '2': '.anytty.cloud.v1.GetOperatorAccountRequest',
      '3': '.anytty.cloud.v1.GetOperatorAccountResponse'
    },
    {
      '1': 'ProvisionAccount',
      '2': '.anytty.cloud.v1.ProvisionAccountRequest',
      '3': '.anytty.cloud.v1.ProvisionAccountResponse'
    },
    {
      '1': 'ResetAccountSetup',
      '2': '.anytty.cloud.v1.ResetAccountSetupRequest',
      '3': '.anytty.cloud.v1.ResetAccountSetupResponse'
    },
    {
      '1': 'ListRuntimeSessions',
      '2': '.anytty.cloud.v1.ListRuntimeSessionsRequest',
      '3': '.anytty.cloud.v1.ListRuntimeSessionsResponse'
    },
    {
      '1': 'ListOrders',
      '2': '.anytty.cloud.v1.ListOperatorOrdersRequest',
      '3': '.anytty.cloud.v1.ListOperatorOrdersResponse'
    },
    {
      '1': 'ListSubscriptions',
      '2': '.anytty.cloud.v1.ListOperatorSubscriptionsRequest',
      '3': '.anytty.cloud.v1.ListOperatorSubscriptionsResponse'
    },
    {
      '1': 'ListUsage',
      '2': '.anytty.cloud.v1.ListOperatorUsageRequest',
      '3': '.anytty.cloud.v1.ListOperatorUsageResponse'
    },
    {
      '1': 'ListAudit',
      '2': '.anytty.cloud.v1.ListOperatorAuditRequest',
      '3': '.anytty.cloud.v1.ListOperatorAuditResponse'
    },
    {
      '1': 'SetAccountState',
      '2': '.anytty.cloud.v1.SetAccountStateRequest',
      '3': '.anytty.cloud.v1.SetAccountStateResponse'
    },
    {
      '1': 'SetAccountRole',
      '2': '.anytty.cloud.v1.SetAccountRoleRequest',
      '3': '.anytty.cloud.v1.SetAccountRoleResponse'
    },
    {
      '1': 'DisconnectDaemon',
      '2': '.anytty.cloud.v1.DisconnectDaemonRequest',
      '3': '.anytty.cloud.v1.DisconnectDaemonResponse'
    },
    {
      '1': 'DisconnectSession',
      '2': '.anytty.cloud.v1.DisconnectSessionRequest',
      '3': '.anytty.cloud.v1.DisconnectSessionResponse'
    },
    {
      '1': 'DeleteEdge',
      '2': '.anytty.cloud.v1.DeleteEdgeRequest',
      '3': '.anytty.cloud.v1.DeleteEdgeResponse'
    },
    {
      '1': 'CreateEdgeIdentityRecovery',
      '2': '.anytty.cloud.v1.CreateEdgeIdentityRecoveryRequest',
      '3': '.anytty.cloud.v1.CreateEdgeIdentityRecoveryResponse'
    },
  ],
};

@$core.Deprecated('Use operatorServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    OperatorServiceBase$messageJson = {
  '.anytty.cloud.v1.GetOperatorOverviewRequest':
      GetOperatorOverviewRequest$json,
  '.anytty.cloud.v1.GetOperatorOverviewResponse':
      GetOperatorOverviewResponse$json,
  '.anytty.cloud.v1.OperatorOverview': OperatorOverview$json,
  '.google.protobuf.Timestamp': $0.Timestamp$json,
  '.anytty.cloud.v1.ListOperatorAccountsRequest':
      ListOperatorAccountsRequest$json,
  '.anytty.cloud.v1.PageRequest': PageRequest$json,
  '.anytty.cloud.v1.ListOperatorAccountsResponse':
      ListOperatorAccountsResponse$json,
  '.anytty.cloud.v1.AccountSummary': AccountSummary$json,
  '.anytty.cloud.v1.AccountProfile': $1.AccountProfile$json,
  '.anytty.cloud.v1.SubscriptionProjection': $2.SubscriptionProjection$json,
  '.anytty.cloud.v1.EffectiveEntitlement': $2.EffectiveEntitlement$json,
  '.anytty.cloud.v1.CloudCapability': $2.CloudCapability$json,
  '.anytty.cloud.v1.UsagePeriodProjection': $2.UsagePeriodProjection$json,
  '.anytty.cloud.v1.GetOperatorAccountRequest': GetOperatorAccountRequest$json,
  '.anytty.cloud.v1.GetOperatorAccountResponse':
      GetOperatorAccountResponse$json,
  '.anytty.cloud.v1.ProvisionAccountRequest': ProvisionAccountRequest$json,
  '.anytty.cloud.v1.ProvisionAccountResponse': ProvisionAccountResponse$json,
  '.anytty.cloud.v1.ResetAccountSetupRequest': ResetAccountSetupRequest$json,
  '.anytty.cloud.v1.ResetAccountSetupResponse': ResetAccountSetupResponse$json,
  '.anytty.cloud.v1.ListRuntimeSessionsRequest':
      ListRuntimeSessionsRequest$json,
  '.anytty.cloud.v1.ListRuntimeSessionsResponse':
      ListRuntimeSessionsResponse$json,
  '.anytty.cloud.v1.RuntimeSessionProjection': RuntimeSessionProjection$json,
  '.anytty.cloud.v1.ListOperatorOrdersRequest': ListOperatorOrdersRequest$json,
  '.anytty.cloud.v1.ListOperatorOrdersResponse':
      ListOperatorOrdersResponse$json,
  '.anytty.cloud.v1.OrderProjection': $2.OrderProjection$json,
  '.anytty.cloud.v1.Money': $2.Money$json,
  '.anytty.cloud.v1.ListOperatorSubscriptionsRequest':
      ListOperatorSubscriptionsRequest$json,
  '.anytty.cloud.v1.ListOperatorSubscriptionsResponse':
      ListOperatorSubscriptionsResponse$json,
  '.anytty.cloud.v1.ListOperatorUsageRequest': ListOperatorUsageRequest$json,
  '.anytty.cloud.v1.ListOperatorUsageResponse': ListOperatorUsageResponse$json,
  '.anytty.cloud.v1.ListOperatorAuditRequest': ListOperatorAuditRequest$json,
  '.anytty.cloud.v1.ListOperatorAuditResponse': ListOperatorAuditResponse$json,
  '.anytty.cloud.v1.OperatorAuditEvent': OperatorAuditEvent$json,
  '.anytty.cloud.v1.SetAccountStateRequest': SetAccountStateRequest$json,
  '.anytty.cloud.v1.SetAccountStateResponse': SetAccountStateResponse$json,
  '.anytty.cloud.v1.SetAccountRoleRequest': SetAccountRoleRequest$json,
  '.anytty.cloud.v1.SetAccountRoleResponse': SetAccountRoleResponse$json,
  '.anytty.cloud.v1.DisconnectDaemonRequest': DisconnectDaemonRequest$json,
  '.anytty.cloud.v1.DisconnectDaemonResponse': DisconnectDaemonResponse$json,
  '.anytty.cloud.v1.DisconnectSessionRequest': DisconnectSessionRequest$json,
  '.anytty.cloud.v1.DisconnectSessionResponse': DisconnectSessionResponse$json,
  '.anytty.cloud.v1.DeleteEdgeRequest': $3.DeleteEdgeRequest$json,
  '.anytty.cloud.v1.DeleteEdgeResponse': $3.DeleteEdgeResponse$json,
  '.anytty.cloud.v1.CreateEdgeIdentityRecoveryRequest':
      $3.CreateEdgeIdentityRecoveryRequest$json,
  '.anytty.cloud.v1.CreateEdgeIdentityRecoveryResponse':
      $3.CreateEdgeIdentityRecoveryResponse$json,
};

/// Descriptor for `OperatorService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List operatorServiceDescriptor = $convert.base64Decode(
    'Cg9PcGVyYXRvclNlcnZpY2USaAoLR2V0T3ZlcnZpZXcSKy5hbnl0dHkuY2xvdWQudjEuR2V0T3'
    'BlcmF0b3JPdmVydmlld1JlcXVlc3QaLC5hbnl0dHkuY2xvdWQudjEuR2V0T3BlcmF0b3JPdmVy'
    'dmlld1Jlc3BvbnNlEmsKDExpc3RBY2NvdW50cxIsLmFueXR0eS5jbG91ZC52MS5MaXN0T3Blcm'
    'F0b3JBY2NvdW50c1JlcXVlc3QaLS5hbnl0dHkuY2xvdWQudjEuTGlzdE9wZXJhdG9yQWNjb3Vu'
    'dHNSZXNwb25zZRJlCgpHZXRBY2NvdW50EiouYW55dHR5LmNsb3VkLnYxLkdldE9wZXJhdG9yQW'
    'Njb3VudFJlcXVlc3QaKy5hbnl0dHkuY2xvdWQudjEuR2V0T3BlcmF0b3JBY2NvdW50UmVzcG9u'
    'c2USZwoQUHJvdmlzaW9uQWNjb3VudBIoLmFueXR0eS5jbG91ZC52MS5Qcm92aXNpb25BY2NvdW'
    '50UmVxdWVzdBopLmFueXR0eS5jbG91ZC52MS5Qcm92aXNpb25BY2NvdW50UmVzcG9uc2USagoR'
    'UmVzZXRBY2NvdW50U2V0dXASKS5hbnl0dHkuY2xvdWQudjEuUmVzZXRBY2NvdW50U2V0dXBSZX'
    'F1ZXN0GiouYW55dHR5LmNsb3VkLnYxLlJlc2V0QWNjb3VudFNldHVwUmVzcG9uc2UScAoTTGlz'
    'dFJ1bnRpbWVTZXNzaW9ucxIrLmFueXR0eS5jbG91ZC52MS5MaXN0UnVudGltZVNlc3Npb25zUm'
    'VxdWVzdBosLmFueXR0eS5jbG91ZC52MS5MaXN0UnVudGltZVNlc3Npb25zUmVzcG9uc2USZQoK'
    'TGlzdE9yZGVycxIqLmFueXR0eS5jbG91ZC52MS5MaXN0T3BlcmF0b3JPcmRlcnNSZXF1ZXN0Gi'
    'suYW55dHR5LmNsb3VkLnYxLkxpc3RPcGVyYXRvck9yZGVyc1Jlc3BvbnNlEnoKEUxpc3RTdWJz'
    'Y3JpcHRpb25zEjEuYW55dHR5LmNsb3VkLnYxLkxpc3RPcGVyYXRvclN1YnNjcmlwdGlvbnNSZX'
    'F1ZXN0GjIuYW55dHR5LmNsb3VkLnYxLkxpc3RPcGVyYXRvclN1YnNjcmlwdGlvbnNSZXNwb25z'
    'ZRJiCglMaXN0VXNhZ2USKS5hbnl0dHkuY2xvdWQudjEuTGlzdE9wZXJhdG9yVXNhZ2VSZXF1ZX'
    'N0GiouYW55dHR5LmNsb3VkLnYxLkxpc3RPcGVyYXRvclVzYWdlUmVzcG9uc2USYgoJTGlzdEF1'
    'ZGl0EikuYW55dHR5LmNsb3VkLnYxLkxpc3RPcGVyYXRvckF1ZGl0UmVxdWVzdBoqLmFueXR0eS'
    '5jbG91ZC52MS5MaXN0T3BlcmF0b3JBdWRpdFJlc3BvbnNlEmQKD1NldEFjY291bnRTdGF0ZRIn'
    'LmFueXR0eS5jbG91ZC52MS5TZXRBY2NvdW50U3RhdGVSZXF1ZXN0GiguYW55dHR5LmNsb3VkLn'
    'YxLlNldEFjY291bnRTdGF0ZVJlc3BvbnNlEmEKDlNldEFjY291bnRSb2xlEiYuYW55dHR5LmNs'
    'b3VkLnYxLlNldEFjY291bnRSb2xlUmVxdWVzdBonLmFueXR0eS5jbG91ZC52MS5TZXRBY2NvdW'
    '50Um9sZVJlc3BvbnNlEmcKEERpc2Nvbm5lY3REYWVtb24SKC5hbnl0dHkuY2xvdWQudjEuRGlz'
    'Y29ubmVjdERhZW1vblJlcXVlc3QaKS5hbnl0dHkuY2xvdWQudjEuRGlzY29ubmVjdERhZW1vbl'
    'Jlc3BvbnNlEmoKEURpc2Nvbm5lY3RTZXNzaW9uEikuYW55dHR5LmNsb3VkLnYxLkRpc2Nvbm5l'
    'Y3RTZXNzaW9uUmVxdWVzdBoqLmFueXR0eS5jbG91ZC52MS5EaXNjb25uZWN0U2Vzc2lvblJlc3'
    'BvbnNlElUKCkRlbGV0ZUVkZ2USIi5hbnl0dHkuY2xvdWQudjEuRGVsZXRlRWRnZVJlcXVlc3Qa'
    'Iy5hbnl0dHkuY2xvdWQudjEuRGVsZXRlRWRnZVJlc3BvbnNlEoUBChpDcmVhdGVFZGdlSWRlbn'
    'RpdHlSZWNvdmVyeRIyLmFueXR0eS5jbG91ZC52MS5DcmVhdGVFZGdlSWRlbnRpdHlSZWNvdmVy'
    'eVJlcXVlc3QaMy5hbnl0dHkuY2xvdWQudjEuQ3JlYXRlRWRnZUlkZW50aXR5UmVjb3ZlcnlSZX'
    'Nwb25zZQ==');
