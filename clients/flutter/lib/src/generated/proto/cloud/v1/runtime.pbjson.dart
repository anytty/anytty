// This is a generated file - do not edit.
//
// Generated from cloud/v1/runtime.proto.

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

@$core.Deprecated('Use clientProductDescriptor instead')
const ClientProduct$json = {
  '1': 'ClientProduct',
  '2': [
    {'1': 'CLIENT_PRODUCT_UNSPECIFIED', '2': 0},
    {'1': 'CLIENT_PRODUCT_TUI', '2': 1},
    {'1': 'CLIENT_PRODUCT_CLI', '2': 2},
    {'1': 'CLIENT_PRODUCT_ANDROID', '2': 3},
    {'1': 'CLIENT_PRODUCT_IOS', '2': 4},
    {'1': 'CLIENT_PRODUCT_DESKTOP_GUI', '2': 5},
  ],
};

/// Descriptor for `ClientProduct`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List clientProductDescriptor = $convert.base64Decode(
    'Cg1DbGllbnRQcm9kdWN0Eh4KGkNMSUVOVF9QUk9EVUNUX1VOU1BFQ0lGSUVEEAASFgoSQ0xJRU'
    '5UX1BST0RVQ1RfVFVJEAESFgoSQ0xJRU5UX1BST0RVQ1RfQ0xJEAISGgoWQ0xJRU5UX1BST0RV'
    'Q1RfQU5EUk9JRBADEhYKEkNMSUVOVF9QUk9EVUNUX0lPUxAEEh4KGkNMSUVOVF9QUk9EVUNUX0'
    'RFU0tUT1BfR1VJEAU=');

@$core.Deprecated('Use cloudClientAccessModeDescriptor instead')
const CloudClientAccessMode$json = {
  '1': 'CloudClientAccessMode',
  '2': [
    {'1': 'CLOUD_CLIENT_ACCESS_MODE_UNSPECIFIED', '2': 0},
    {'1': 'CLOUD_CLIENT_ACCESS_MODE_CAPABILITY', '2': 1},
    {'1': 'CLOUD_CLIENT_ACCESS_MODE_PAIRING', '2': 2},
  ],
};

/// Descriptor for `CloudClientAccessMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List cloudClientAccessModeDescriptor = $convert.base64Decode(
    'ChVDbG91ZENsaWVudEFjY2Vzc01vZGUSKAokQ0xPVURfQ0xJRU5UX0FDQ0VTU19NT0RFX1VOU1'
    'BFQ0lGSUVEEAASJwojQ0xPVURfQ0xJRU5UX0FDQ0VTU19NT0RFX0NBUEFCSUxJVFkQARIkCiBD'
    'TE9VRF9DTElFTlRfQUNDRVNTX01PREVfUEFJUklORxAC');

@$core.Deprecated('Use agentPresenceDescriptor instead')
const AgentPresence$json = {
  '1': 'AgentPresence',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'boot_id', '3': 3, '4': 1, '5': 9, '10': 'bootId'},
    {'1': 'connection_id', '3': 4, '4': 1, '5': 9, '10': 'connectionId'},
    {'1': 'generation', '3': 5, '4': 1, '5': 4, '10': 'generation'},
    {'1': 'binding_id', '3': 6, '4': 1, '5': 9, '10': 'bindingId'},
    {
      '1': 'binding_issued_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'bindingIssuedAt'
    },
  ],
};

/// Descriptor for `AgentPresence`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentPresenceDescriptor = $convert.base64Decode(
    'Cg1BZ2VudFByZXNlbmNlEhsKCWRhZW1vbl9pZBgBIAEoCVIIZGFlbW9uSWQSHQoKYWNjb3VudF'
    '9pZBgCIAEoCVIJYWNjb3VudElkEhcKB2Jvb3RfaWQYAyABKAlSBmJvb3RJZBIjCg1jb25uZWN0'
    'aW9uX2lkGAQgASgJUgxjb25uZWN0aW9uSWQSHgoKZ2VuZXJhdGlvbhgFIAEoBFIKZ2VuZXJhdG'
    'lvbhIdCgpiaW5kaW5nX2lkGAYgASgJUgliaW5kaW5nSWQSRgoRYmluZGluZ19pc3N1ZWRfYXQY'
    'ByABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUg9iaW5kaW5nSXNzdWVkQXQ=');

@$core.Deprecated('Use clientSessionSummaryDescriptor instead')
const ClientSessionSummary$json = {
  '1': 'ClientSessionSummary',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'daemon_id', '3': 3, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'client_id', '3': 4, '4': 1, '5': 9, '10': 'clientId'},
    {
      '1': 'product',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.ClientProduct',
      '10': 'product'
    },
    {'1': 'generation', '3': 6, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'access_mode',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudClientAccessMode',
      '10': 'accessMode'
    },
  ],
};

/// Descriptor for `ClientSessionSummary`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientSessionSummaryDescriptor = $convert.base64Decode(
    'ChRDbGllbnRTZXNzaW9uU3VtbWFyeRIdCgpzZXNzaW9uX2lkGAEgASgJUglzZXNzaW9uSWQSHQ'
    'oKYWNjb3VudF9pZBgCIAEoCVIJYWNjb3VudElkEhsKCWRhZW1vbl9pZBgDIAEoCVIIZGFlbW9u'
    'SWQSGwoJY2xpZW50X2lkGAQgASgJUghjbGllbnRJZBI4Cgdwcm9kdWN0GAUgASgOMh4uYW55dH'
    'R5LmNsb3VkLnYxLkNsaWVudFByb2R1Y3RSB3Byb2R1Y3QSHgoKZ2VuZXJhdGlvbhgGIAEoBFIK'
    'Z2VuZXJhdGlvbhJHCgthY2Nlc3NfbW9kZRgHIAEoDjImLmFueXR0eS5jbG91ZC52MS5DbG91ZE'
    'NsaWVudEFjY2Vzc01vZGVSCmFjY2Vzc01vZGU=');

@$core.Deprecated('Use runtimeSnapshotDescriptor instead')
const RuntimeSnapshot$json = {
  '1': 'RuntimeSnapshot',
  '2': [
    {'1': 'revision', '3': 1, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'agents',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentPresence',
      '10': 'agents'
    },
    {
      '1': 'sessions',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientSessionSummary',
      '10': 'sessions'
    },
  ],
  '9': [
    {'1': 4, '2': 5},
  ],
};

/// Descriptor for `RuntimeSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeSnapshotDescriptor = $convert.base64Decode(
    'Cg9SdW50aW1lU25hcHNob3QSGgoIcmV2aXNpb24YASABKARSCHJldmlzaW9uEjYKBmFnZW50cx'
    'gCIAMoCzIeLmFueXR0eS5jbG91ZC52MS5BZ2VudFByZXNlbmNlUgZhZ2VudHMSQQoIc2Vzc2lv'
    'bnMYAyADKAsyJS5hbnl0dHkuY2xvdWQudjEuQ2xpZW50U2Vzc2lvblN1bW1hcnlSCHNlc3Npb2'
    '5zSgQIBBAF');

@$core.Deprecated('Use agentRemovedDescriptor instead')
const AgentRemoved$json = {
  '1': 'AgentRemoved',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
  ],
};

/// Descriptor for `AgentRemoved`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentRemovedDescriptor = $convert.base64Decode(
    'CgxBZ2VudFJlbW92ZWQSGwoJZGFlbW9uX2lkGAEgASgJUghkYWVtb25JZBIeCgpnZW5lcmF0aW'
    '9uGAIgASgEUgpnZW5lcmF0aW9u');

@$core.Deprecated('Use clientSessionRemovedDescriptor instead')
const ClientSessionRemoved$json = {
  '1': 'ClientSessionRemoved',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
  ],
};

/// Descriptor for `ClientSessionRemoved`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientSessionRemovedDescriptor = $convert.base64Decode(
    'ChRDbGllbnRTZXNzaW9uUmVtb3ZlZBIdCgpzZXNzaW9uX2lkGAEgASgJUglzZXNzaW9uSWQSHg'
    'oKZ2VuZXJhdGlvbhgCIAEoBFIKZ2VuZXJhdGlvbg==');

@$core.Deprecated('Use runtimeDeltaDescriptor instead')
const RuntimeDelta$json = {
  '1': 'RuntimeDelta',
  '2': [
    {'1': 'revision', '3': 1, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'agent_upserted',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentPresence',
      '9': 0,
      '10': 'agentUpserted'
    },
    {
      '1': 'agent_removed',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentRemoved',
      '9': 0,
      '10': 'agentRemoved'
    },
    {
      '1': 'session_upserted',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientSessionSummary',
      '9': 0,
      '10': 'sessionUpserted'
    },
    {
      '1': 'session_removed',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientSessionRemoved',
      '9': 0,
      '10': 'sessionRemoved'
    },
  ],
  '8': [
    {'1': 'change'},
  ],
  '9': [
    {'1': 14, '2': 15},
    {'1': 15, '2': 16},
  ],
};

/// Descriptor for `RuntimeDelta`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeDeltaDescriptor = $convert.base64Decode(
    'CgxSdW50aW1lRGVsdGESGgoIcmV2aXNpb24YASABKARSCHJldmlzaW9uEkcKDmFnZW50X3Vwc2'
    'VydGVkGAogASgLMh4uYW55dHR5LmNsb3VkLnYxLkFnZW50UHJlc2VuY2VIAFINYWdlbnRVcHNl'
    'cnRlZBJECg1hZ2VudF9yZW1vdmVkGAsgASgLMh0uYW55dHR5LmNsb3VkLnYxLkFnZW50UmVtb3'
    'ZlZEgAUgxhZ2VudFJlbW92ZWQSUgoQc2Vzc2lvbl91cHNlcnRlZBgMIAEoCzIlLmFueXR0eS5j'
    'bG91ZC52MS5DbGllbnRTZXNzaW9uU3VtbWFyeUgAUg9zZXNzaW9uVXBzZXJ0ZWQSUAoPc2Vzc2'
    'lvbl9yZW1vdmVkGA0gASgLMiUuYW55dHR5LmNsb3VkLnYxLkNsaWVudFNlc3Npb25SZW1vdmVk'
    'SABSDnNlc3Npb25SZW1vdmVkQggKBmNoYW5nZUoECA4QD0oECA8QEA==');
