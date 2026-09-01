// This is a generated file - do not edit.
//
// Generated from cloud/v1/usage.proto.

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

@$core.Deprecated('Use relayPreferenceDescriptor instead')
const RelayPreference$json = {
  '1': 'RelayPreference',
  '2': [
    {'1': 'RELAY_PREFERENCE_UNSPECIFIED', '2': 0},
    {'1': 'RELAY_PREFERENCE_AUTO', '2': 1},
    {'1': 'RELAY_PREFERENCE_DIRECT_ONLY', '2': 2},
    {'1': 'RELAY_PREFERENCE_RELAY_ONLY', '2': 3},
  ],
};

/// Descriptor for `RelayPreference`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List relayPreferenceDescriptor = $convert.base64Decode(
    'Cg9SZWxheVByZWZlcmVuY2USIAocUkVMQVlfUFJFRkVSRU5DRV9VTlNQRUNJRklFRBAAEhkKFV'
    'JFTEFZX1BSRUZFUkVOQ0VfQVVUTxABEiAKHFJFTEFZX1BSRUZFUkVOQ0VfRElSRUNUX09OTFkQ'
    'AhIfChtSRUxBWV9QUkVGRVJFTkNFX1JFTEFZX09OTFkQAw==');

@$core.Deprecated('Use relayTransportDescriptor instead')
const RelayTransport$json = {
  '1': 'RelayTransport',
  '2': [
    {'1': 'RELAY_TRANSPORT_UNSPECIFIED', '2': 0},
    {'1': 'RELAY_TRANSPORT_UDP', '2': 1},
    {'1': 'RELAY_TRANSPORT_TCP', '2': 2},
    {'1': 'RELAY_TRANSPORT_TLS', '2': 3},
  ],
};

/// Descriptor for `RelayTransport`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List relayTransportDescriptor = $convert.base64Decode(
    'Cg5SZWxheVRyYW5zcG9ydBIfChtSRUxBWV9UUkFOU1BPUlRfVU5TUEVDSUZJRUQQABIXChNSRU'
    'xBWV9UUkFOU1BPUlRfVURQEAESFwoTUkVMQVlfVFJBTlNQT1JUX1RDUBACEhcKE1JFTEFZX1RS'
    'QU5TUE9SVF9UTFMQAw==');

@$core.Deprecated('Use relaySettlementKindDescriptor instead')
const RelaySettlementKind$json = {
  '1': 'RelaySettlementKind',
  '2': [
    {'1': 'RELAY_SETTLEMENT_KIND_UNSPECIFIED', '2': 0},
    {'1': 'RELAY_SETTLEMENT_KIND_EXACT', '2': 1},
    {'1': 'RELAY_SETTLEMENT_KIND_RECOVERY_MAX', '2': 2},
  ],
};

/// Descriptor for `RelaySettlementKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List relaySettlementKindDescriptor = $convert.base64Decode(
    'ChNSZWxheVNldHRsZW1lbnRLaW5kEiUKIVJFTEFZX1NFVFRMRU1FTlRfS0lORF9VTlNQRUNJRk'
    'lFRBAAEh8KG1JFTEFZX1NFVFRMRU1FTlRfS0lORF9FWEFDVBABEiYKIlJFTEFZX1NFVFRMRU1F'
    'TlRfS0lORF9SRUNPVkVSWV9NQVgQAg==');

@$core.Deprecated('Use relayResponseCodeDescriptor instead')
const RelayResponseCode$json = {
  '1': 'RelayResponseCode',
  '2': [
    {'1': 'RELAY_RESPONSE_CODE_UNSPECIFIED', '2': 0},
    {'1': 'RELAY_RESPONSE_CODE_APPLIED', '2': 1},
    {'1': 'RELAY_RESPONSE_CODE_REPLAY', '2': 2},
    {'1': 'RELAY_RESPONSE_CODE_TERMINAL', '2': 3},
    {'1': 'RELAY_RESPONSE_CODE_REJECTED', '2': 4},
    {'1': 'RELAY_RESPONSE_CODE_CONFLICT', '2': 5},
    {'1': 'RELAY_RESPONSE_CODE_UNAVAILABLE', '2': 6},
  ],
};

/// Descriptor for `RelayResponseCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List relayResponseCodeDescriptor = $convert.base64Decode(
    'ChFSZWxheVJlc3BvbnNlQ29kZRIjCh9SRUxBWV9SRVNQT05TRV9DT0RFX1VOU1BFQ0lGSUVEEA'
    'ASHwobUkVMQVlfUkVTUE9OU0VfQ09ERV9BUFBMSUVEEAESHgoaUkVMQVlfUkVTUE9OU0VfQ09E'
    'RV9SRVBMQVkQAhIgChxSRUxBWV9SRVNQT05TRV9DT0RFX1RFUk1JTkFMEAMSIAocUkVMQVlfUk'
    'VTUE9OU0VfQ09ERV9SRUpFQ1RFRBAEEiAKHFJFTEFZX1JFU1BPTlNFX0NPREVfQ09ORkxJQ1QQ'
    'BRIjCh9SRUxBWV9SRVNQT05TRV9DT0RFX1VOQVZBSUxBQkxFEAY=');

@$core.Deprecated('Use relayAccountActionTypeDescriptor instead')
const RelayAccountActionType$json = {
  '1': 'RelayAccountActionType',
  '2': [
    {'1': 'RELAY_ACCOUNT_ACTION_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'RELAY_ACCOUNT_ACTION_TYPE_ALLOW', '2': 1},
    {'1': 'RELAY_ACCOUNT_ACTION_TYPE_DENY_AND_CLOSE', '2': 2},
  ],
};

/// Descriptor for `RelayAccountActionType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List relayAccountActionTypeDescriptor = $convert.base64Decode(
    'ChZSZWxheUFjY291bnRBY3Rpb25UeXBlEikKJVJFTEFZX0FDQ09VTlRfQUNUSU9OX1RZUEVfVU'
    '5TUEVDSUZJRUQQABIjCh9SRUxBWV9BQ0NPVU5UX0FDVElPTl9UWVBFX0FMTE9XEAESLAooUkVM'
    'QVlfQUNDT1VOVF9BQ1RJT05fVFlQRV9ERU5ZX0FORF9DTE9TRRAC');

@$core.Deprecated('Use relayJournalStageDescriptor instead')
const RelayJournalStage$json = {
  '1': 'RelayJournalStage',
  '2': [
    {'1': 'RELAY_JOURNAL_STAGE_UNSPECIFIED', '2': 0},
    {'1': 'RELAY_JOURNAL_STAGE_REQUESTED', '2': 1},
    {'1': 'RELAY_JOURNAL_STAGE_HELD_UNEXPOSED', '2': 2},
    {'1': 'RELAY_JOURNAL_STAGE_EXPOSED', '2': 3},
    {'1': 'RELAY_JOURNAL_STAGE_RENEW_PENDING', '2': 4},
    {'1': 'RELAY_JOURNAL_STAGE_CLOSING', '2': 5},
    {'1': 'RELAY_JOURNAL_STAGE_SETTLEMENT_DURABLE', '2': 6},
  ],
};

/// Descriptor for `RelayJournalStage`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List relayJournalStageDescriptor = $convert.base64Decode(
    'ChFSZWxheUpvdXJuYWxTdGFnZRIjCh9SRUxBWV9KT1VSTkFMX1NUQUdFX1VOU1BFQ0lGSUVEEA'
    'ASIQodUkVMQVlfSk9VUk5BTF9TVEFHRV9SRVFVRVNURUQQARImCiJSRUxBWV9KT1VSTkFMX1NU'
    'QUdFX0hFTERfVU5FWFBPU0VEEAISHwobUkVMQVlfSk9VUk5BTF9TVEFHRV9FWFBPU0VEEAMSJQ'
    'ohUkVMQVlfSk9VUk5BTF9TVEFHRV9SRU5FV19QRU5ESU5HEAQSHwobUkVMQVlfSk9VUk5BTF9T'
    'VEFHRV9DTE9TSU5HEAUSKgomUkVMQVlfSk9VUk5BTF9TVEFHRV9TRVRUTEVNRU5UX0RVUkFCTE'
    'UQBg==');

@$core.Deprecated('Use relayPolicySnapshotDescriptor instead')
const RelayPolicySnapshot$json = {
  '1': 'RelayPolicySnapshot',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'account_revision', '3': 2, '4': 1, '5': 4, '10': 'accountRevision'},
    {'1': 'account_state', '3': 3, '4': 1, '5': 9, '10': 'accountState'},
    {'1': 'subscription_id', '3': 4, '4': 1, '5': 9, '10': 'subscriptionId'},
    {
      '1': 'subscription_revision',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'subscriptionRevision'
    },
    {
      '1': 'subscription_state',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'subscriptionState'
    },
    {
      '1': 'period_start',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodStart'
    },
    {
      '1': 'period_end',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodEnd'
    },
    {'1': 'plan_id', '3': 9, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'plan_version', '3': 10, '4': 1, '5': 4, '10': 'planVersion'},
    {'1': 'plan_revision', '3': 11, '4': 1, '5': 4, '10': 'planRevision'},
    {'1': 'relay_enabled', '3': 12, '4': 1, '5': 8, '10': 'relayEnabled'},
    {
      '1': 'relay_max_bytes_per_period',
      '3': 13,
      '4': 1,
      '5': 4,
      '10': 'relayMaxBytesPerPeriod'
    },
    {
      '1': 'relay_max_bytes_per_session',
      '3': 14,
      '4': 1,
      '5': 4,
      '10': 'relayMaxBytesPerSession'
    },
    {
      '1': 'relay_max_rate_bytes_per_second',
      '3': 15,
      '4': 1,
      '5': 4,
      '10': 'relayMaxRateBytesPerSecond'
    },
    {
      '1': 'relay_max_concurrency',
      '3': 16,
      '4': 1,
      '5': 13,
      '10': 'relayMaxConcurrency'
    },
    {'1': 'allowed_regions', '3': 17, '4': 3, '5': 9, '10': 'allowedRegions'},
    {'1': 'edge_id', '3': 18, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'edge_revision', '3': 19, '4': 1, '5': 4, '10': 'edgeRevision'},
    {'1': 'edge_enabled', '3': 20, '4': 1, '5': 8, '10': 'edgeEnabled'},
    {'1': 'edge_region', '3': 21, '4': 1, '5': 9, '10': 'edgeRegion'},
    {'1': 'daemon_id', '3': 22, '4': 1, '5': 9, '10': 'daemonId'},
    {
      '1': 'daemon_state_revision',
      '3': 23,
      '4': 1,
      '5': 4,
      '10': 'daemonStateRevision'
    },
  ],
  '9': [
    {'1': 24, '2': 29},
  ],
};

/// Descriptor for `RelayPolicySnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayPolicySnapshotDescriptor = $convert.base64Decode(
    'ChNSZWxheVBvbGljeVNuYXBzaG90Eh0KCmFjY291bnRfaWQYASABKAlSCWFjY291bnRJZBIpCh'
    'BhY2NvdW50X3JldmlzaW9uGAIgASgEUg9hY2NvdW50UmV2aXNpb24SIwoNYWNjb3VudF9zdGF0'
    'ZRgDIAEoCVIMYWNjb3VudFN0YXRlEicKD3N1YnNjcmlwdGlvbl9pZBgEIAEoCVIOc3Vic2NyaX'
    'B0aW9uSWQSMwoVc3Vic2NyaXB0aW9uX3JldmlzaW9uGAUgASgEUhRzdWJzY3JpcHRpb25SZXZp'
    'c2lvbhItChJzdWJzY3JpcHRpb25fc3RhdGUYBiABKAlSEXN1YnNjcmlwdGlvblN0YXRlEj0KDH'
    'BlcmlvZF9zdGFydBgHIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC3BlcmlvZFN0'
    'YXJ0EjkKCnBlcmlvZF9lbmQYCCABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUglwZX'
    'Jpb2RFbmQSFwoHcGxhbl9pZBgJIAEoCVIGcGxhbklkEiEKDHBsYW5fdmVyc2lvbhgKIAEoBFIL'
    'cGxhblZlcnNpb24SIwoNcGxhbl9yZXZpc2lvbhgLIAEoBFIMcGxhblJldmlzaW9uEiMKDXJlbG'
    'F5X2VuYWJsZWQYDCABKAhSDHJlbGF5RW5hYmxlZBI6ChpyZWxheV9tYXhfYnl0ZXNfcGVyX3Bl'
    'cmlvZBgNIAEoBFIWcmVsYXlNYXhCeXRlc1BlclBlcmlvZBI8ChtyZWxheV9tYXhfYnl0ZXNfcG'
    'VyX3Nlc3Npb24YDiABKARSF3JlbGF5TWF4Qnl0ZXNQZXJTZXNzaW9uEkMKH3JlbGF5X21heF9y'
    'YXRlX2J5dGVzX3Blcl9zZWNvbmQYDyABKARSGnJlbGF5TWF4UmF0ZUJ5dGVzUGVyU2Vjb25kEj'
    'IKFXJlbGF5X21heF9jb25jdXJyZW5jeRgQIAEoDVITcmVsYXlNYXhDb25jdXJyZW5jeRInCg9h'
    'bGxvd2VkX3JlZ2lvbnMYESADKAlSDmFsbG93ZWRSZWdpb25zEhcKB2VkZ2VfaWQYEiABKAlSBm'
    'VkZ2VJZBIjCg1lZGdlX3JldmlzaW9uGBMgASgEUgxlZGdlUmV2aXNpb24SIQoMZWRnZV9lbmFi'
    'bGVkGBQgASgIUgtlZGdlRW5hYmxlZBIfCgtlZGdlX3JlZ2lvbhgVIAEoCVIKZWRnZVJlZ2lvbh'
    'IbCglkYWVtb25faWQYFiABKAlSCGRhZW1vbklkEjIKFWRhZW1vbl9zdGF0ZV9yZXZpc2lvbhgX'
    'IAEoBFITZGFlbW9uU3RhdGVSZXZpc2lvbkoECBgQHQ==');

@$core.Deprecated('Use relayReserveRequestDescriptor instead')
const RelayReserveRequest$json = {
  '1': 'RelayReserveRequest',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'daemon_id', '3': 3, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'client_id', '3': 4, '4': 1, '5': 9, '10': 'clientId'},
    {'1': 'session_id', '3': 5, '4': 1, '5': 9, '10': 'sessionId'},
    {
      '1': 'observed_at',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'observedAt'
    },
    {'1': 'request_digest', '3': 7, '4': 1, '5': 12, '10': 'requestDigest'},
  ],
};

/// Descriptor for `RelayReserveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayReserveRequestDescriptor = $convert.base64Decode(
    'ChNSZWxheVJlc2VydmVSZXF1ZXN0EiUKDnJlc2VydmF0aW9uX2lkGAEgASgJUg1yZXNlcnZhdG'
    'lvbklkEh0KCmFjY291bnRfaWQYAiABKAlSCWFjY291bnRJZBIbCglkYWVtb25faWQYAyABKAlS'
    'CGRhZW1vbklkEhsKCWNsaWVudF9pZBgEIAEoCVIIY2xpZW50SWQSHQoKc2Vzc2lvbl9pZBgFIA'
    'EoCVIJc2Vzc2lvbklkEjsKC29ic2VydmVkX2F0GAYgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRp'
    'bWVzdGFtcFIKb2JzZXJ2ZWRBdBIlCg5yZXF1ZXN0X2RpZ2VzdBgHIAEoDFINcmVxdWVzdERpZ2'
    'VzdA==');

@$core.Deprecated('Use relayGrantDescriptor instead')
const RelayGrant$json = {
  '1': 'RelayGrant',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'reserved_bytes', '3': 3, '4': 1, '5': 4, '10': 'reservedBytes'},
    {
      '1': 'max_rate_bytes_per_second',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'maxRateBytesPerSecond'
    },
    {'1': 'renew_sequence', '3': 5, '4': 1, '5': 4, '10': 'renewSequence'},
    {
      '1': 'authorized_until',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'authorizedUntil'
    },
    {'1': 'policy_digest', '3': 7, '4': 1, '5': 12, '10': 'policyDigest'},
    {
      '1': 'policy',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayPolicySnapshot',
      '10': 'policy'
    },
  ],
};

/// Descriptor for `RelayGrant`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayGrantDescriptor = $convert.base64Decode(
    'CgpSZWxheUdyYW50EiUKDnJlc2VydmF0aW9uX2lkGAEgASgJUg1yZXNlcnZhdGlvbklkEh0KCn'
    'Nlc3Npb25faWQYAiABKAlSCXNlc3Npb25JZBIlCg5yZXNlcnZlZF9ieXRlcxgDIAEoBFINcmVz'
    'ZXJ2ZWRCeXRlcxI4ChltYXhfcmF0ZV9ieXRlc19wZXJfc2Vjb25kGAQgASgEUhVtYXhSYXRlQn'
    'l0ZXNQZXJTZWNvbmQSJQoOcmVuZXdfc2VxdWVuY2UYBSABKARSDXJlbmV3U2VxdWVuY2USRQoQ'
    'YXV0aG9yaXplZF91bnRpbBgGIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSD2F1dG'
    'hvcml6ZWRVbnRpbBIjCg1wb2xpY3lfZGlnZXN0GAcgASgMUgxwb2xpY3lEaWdlc3QSPAoGcG9s'
    'aWN5GAggASgLMiQuYW55dHR5LmNsb3VkLnYxLlJlbGF5UG9saWN5U25hcHNob3RSBnBvbGljeQ'
    '==');

@$core.Deprecated('Use relayReserveResponseDescriptor instead')
const RelayReserveResponse$json = {
  '1': 'RelayReserveResponse',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {'1': 'request_digest', '3': 2, '4': 1, '5': 12, '10': 'requestDigest'},
    {
      '1': 'code',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayResponseCode',
      '10': 'code'
    },
    {
      '1': 'grant',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayGrant',
      '10': 'grant'
    },
    {
      '1': 'terminal',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelaySettlementAck',
      '10': 'terminal'
    },
    {'1': 'error_message', '3': 6, '4': 1, '5': 9, '10': 'errorMessage'},
    {
      '1': 'entitlement_failure',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudEntitlementFailure',
      '10': 'entitlementFailure'
    },
  ],
};

/// Descriptor for `RelayReserveResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayReserveResponseDescriptor = $convert.base64Decode(
    'ChRSZWxheVJlc2VydmVSZXNwb25zZRIlCg5yZXNlcnZhdGlvbl9pZBgBIAEoCVINcmVzZXJ2YX'
    'Rpb25JZBIlCg5yZXF1ZXN0X2RpZ2VzdBgCIAEoDFINcmVxdWVzdERpZ2VzdBI2CgRjb2RlGAMg'
    'ASgOMiIuYW55dHR5LmNsb3VkLnYxLlJlbGF5UmVzcG9uc2VDb2RlUgRjb2RlEjEKBWdyYW50GA'
    'QgASgLMhsuYW55dHR5LmNsb3VkLnYxLlJlbGF5R3JhbnRSBWdyYW50Ej8KCHRlcm1pbmFsGAUg'
    'ASgLMiMuYW55dHR5LmNsb3VkLnYxLlJlbGF5U2V0dGxlbWVudEFja1IIdGVybWluYWwSIwoNZX'
    'Jyb3JfbWVzc2FnZRgGIAEoCVIMZXJyb3JNZXNzYWdlElkKE2VudGl0bGVtZW50X2ZhaWx1cmUY'
    'ByABKAsyKC5hbnl0dHkuY2xvdWQudjEuQ2xvdWRFbnRpdGxlbWVudEZhaWx1cmVSEmVudGl0bG'
    'VtZW50RmFpbHVyZQ==');

@$core.Deprecated('Use relayRenewRequestDescriptor instead')
const RelayRenewRequest$json = {
  '1': 'RelayRenewRequest',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {'1': 'renew_sequence', '3': 2, '4': 1, '5': 4, '10': 'renewSequence'},
    {'1': 'policy_digest', '3': 3, '4': 1, '5': 12, '10': 'policyDigest'},
    {
      '1': 'observed_at',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'observedAt'
    },
  ],
};

/// Descriptor for `RelayRenewRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayRenewRequestDescriptor = $convert.base64Decode(
    'ChFSZWxheVJlbmV3UmVxdWVzdBIlCg5yZXNlcnZhdGlvbl9pZBgBIAEoCVINcmVzZXJ2YXRpb2'
    '5JZBIlCg5yZW5ld19zZXF1ZW5jZRgCIAEoBFINcmVuZXdTZXF1ZW5jZRIjCg1wb2xpY3lfZGln'
    'ZXN0GAMgASgMUgxwb2xpY3lEaWdlc3QSOwoLb2JzZXJ2ZWRfYXQYBCABKAsyGi5nb29nbGUucH'
    'JvdG9idWYuVGltZXN0YW1wUgpvYnNlcnZlZEF0');

@$core.Deprecated('Use relayRenewResponseDescriptor instead')
const RelayRenewResponse$json = {
  '1': 'RelayRenewResponse',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {'1': 'renew_sequence', '3': 2, '4': 1, '5': 4, '10': 'renewSequence'},
    {
      '1': 'code',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayResponseCode',
      '10': 'code'
    },
    {
      '1': 'grant',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayGrant',
      '10': 'grant'
    },
    {
      '1': 'terminal',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelaySettlementAck',
      '10': 'terminal'
    },
    {'1': 'error_message', '3': 6, '4': 1, '5': 9, '10': 'errorMessage'},
    {
      '1': 'entitlement_failure',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudEntitlementFailure',
      '10': 'entitlementFailure'
    },
  ],
};

/// Descriptor for `RelayRenewResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayRenewResponseDescriptor = $convert.base64Decode(
    'ChJSZWxheVJlbmV3UmVzcG9uc2USJQoOcmVzZXJ2YXRpb25faWQYASABKAlSDXJlc2VydmF0aW'
    '9uSWQSJQoOcmVuZXdfc2VxdWVuY2UYAiABKARSDXJlbmV3U2VxdWVuY2USNgoEY29kZRgDIAEo'
    'DjIiLmFueXR0eS5jbG91ZC52MS5SZWxheVJlc3BvbnNlQ29kZVIEY29kZRIxCgVncmFudBgEIA'
    'EoCzIbLmFueXR0eS5jbG91ZC52MS5SZWxheUdyYW50UgVncmFudBI/Cgh0ZXJtaW5hbBgFIAEo'
    'CzIjLmFueXR0eS5jbG91ZC52MS5SZWxheVNldHRsZW1lbnRBY2tSCHRlcm1pbmFsEiMKDWVycm'
    '9yX21lc3NhZ2UYBiABKAlSDGVycm9yTWVzc2FnZRJZChNlbnRpdGxlbWVudF9mYWlsdXJlGAcg'
    'ASgLMiguYW55dHR5LmNsb3VkLnYxLkNsb3VkRW50aXRsZW1lbnRGYWlsdXJlUhJlbnRpdGxlbW'
    'VudEZhaWx1cmU=');

@$core.Deprecated('Use relaySettlementDescriptor instead')
const RelaySettlement$json = {
  '1': 'RelaySettlement',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {
      '1': 'kind',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelaySettlementKind',
      '10': 'kind'
    },
    {'1': 'ingress_bytes', '3': 3, '4': 1, '5': 4, '10': 'ingressBytes'},
    {'1': 'egress_bytes', '3': 4, '4': 1, '5': 4, '10': 'egressBytes'},
    {'1': 'policy_digest', '3': 5, '4': 1, '5': 12, '10': 'policyDigest'},
    {
      '1': 'observed_at',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'observedAt'
    },
  ],
};

/// Descriptor for `RelaySettlement`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relaySettlementDescriptor = $convert.base64Decode(
    'Cg9SZWxheVNldHRsZW1lbnQSJQoOcmVzZXJ2YXRpb25faWQYASABKAlSDXJlc2VydmF0aW9uSW'
    'QSOAoEa2luZBgCIAEoDjIkLmFueXR0eS5jbG91ZC52MS5SZWxheVNldHRsZW1lbnRLaW5kUgRr'
    'aW5kEiMKDWluZ3Jlc3NfYnl0ZXMYAyABKARSDGluZ3Jlc3NCeXRlcxIhCgxlZ3Jlc3NfYnl0ZX'
    'MYBCABKARSC2VncmVzc0J5dGVzEiMKDXBvbGljeV9kaWdlc3QYBSABKAxSDHBvbGljeURpZ2Vz'
    'dBI7CgtvYnNlcnZlZF9hdBgGIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCm9ic2'
    'VydmVkQXQ=');

@$core.Deprecated('Use relaySettlementAckDescriptor instead')
const RelaySettlementAck$json = {
  '1': 'RelaySettlementAck',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {
      '1': 'kind',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelaySettlementKind',
      '10': 'kind'
    },
    {'1': 'ingress_bytes', '3': 3, '4': 1, '5': 4, '10': 'ingressBytes'},
    {'1': 'egress_bytes', '3': 4, '4': 1, '5': 4, '10': 'egressBytes'},
    {'1': 'recovery_bytes', '3': 5, '4': 1, '5': 4, '10': 'recoveryBytes'},
    {'1': 'policy_digest', '3': 6, '4': 1, '5': 12, '10': 'policyDigest'},
    {
      '1': 'observed_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'observedAt'
    },
    {
      '1': 'settled_at',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'settledAt'
    },
    {
      '1': 'code',
      '3': 9,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayResponseCode',
      '10': 'code'
    },
    {'1': 'error_message', '3': 10, '4': 1, '5': 9, '10': 'errorMessage'},
  ],
};

/// Descriptor for `RelaySettlementAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relaySettlementAckDescriptor = $convert.base64Decode(
    'ChJSZWxheVNldHRsZW1lbnRBY2sSJQoOcmVzZXJ2YXRpb25faWQYASABKAlSDXJlc2VydmF0aW'
    '9uSWQSOAoEa2luZBgCIAEoDjIkLmFueXR0eS5jbG91ZC52MS5SZWxheVNldHRsZW1lbnRLaW5k'
    'UgRraW5kEiMKDWluZ3Jlc3NfYnl0ZXMYAyABKARSDGluZ3Jlc3NCeXRlcxIhCgxlZ3Jlc3NfYn'
    'l0ZXMYBCABKARSC2VncmVzc0J5dGVzEiUKDnJlY292ZXJ5X2J5dGVzGAUgASgEUg1yZWNvdmVy'
    'eUJ5dGVzEiMKDXBvbGljeV9kaWdlc3QYBiABKAxSDHBvbGljeURpZ2VzdBI7CgtvYnNlcnZlZF'
    '9hdBgHIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCm9ic2VydmVkQXQSOQoKc2V0'
    'dGxlZF9hdBgIIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCXNldHRsZWRBdBI2Cg'
    'Rjb2RlGAkgASgOMiIuYW55dHR5LmNsb3VkLnYxLlJlbGF5UmVzcG9uc2VDb2RlUgRjb2RlEiMK'
    'DWVycm9yX21lc3NhZ2UYCiABKAlSDGVycm9yTWVzc2FnZQ==');

@$core.Deprecated('Use relayQueryRequestDescriptor instead')
const RelayQueryRequest$json = {
  '1': 'RelayQueryRequest',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
  ],
};

/// Descriptor for `RelayQueryRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayQueryRequestDescriptor = $convert.base64Decode(
    'ChFSZWxheVF1ZXJ5UmVxdWVzdBIlCg5yZXNlcnZhdGlvbl9pZBgBIAEoCVINcmVzZXJ2YXRpb2'
    '5JZA==');

@$core.Deprecated('Use relayQueryResponseDescriptor instead')
const RelayQueryResponse$json = {
  '1': 'RelayQueryResponse',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {
      '1': 'code',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayResponseCode',
      '10': 'code'
    },
    {
      '1': 'grant',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayGrant',
      '10': 'grant'
    },
    {
      '1': 'terminal',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelaySettlementAck',
      '10': 'terminal'
    },
    {'1': 'error_message', '3': 5, '4': 1, '5': 9, '10': 'errorMessage'},
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

/// Descriptor for `RelayQueryResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayQueryResponseDescriptor = $convert.base64Decode(
    'ChJSZWxheVF1ZXJ5UmVzcG9uc2USJQoOcmVzZXJ2YXRpb25faWQYASABKAlSDXJlc2VydmF0aW'
    '9uSWQSNgoEY29kZRgCIAEoDjIiLmFueXR0eS5jbG91ZC52MS5SZWxheVJlc3BvbnNlQ29kZVIE'
    'Y29kZRIxCgVncmFudBgDIAEoCzIbLmFueXR0eS5jbG91ZC52MS5SZWxheUdyYW50UgVncmFudB'
    'I/Cgh0ZXJtaW5hbBgEIAEoCzIjLmFueXR0eS5jbG91ZC52MS5SZWxheVNldHRsZW1lbnRBY2tS'
    'CHRlcm1pbmFsEiMKDWVycm9yX21lc3NhZ2UYBSABKAlSDGVycm9yTWVzc2FnZRJZChNlbnRpdG'
    'xlbWVudF9mYWlsdXJlGAYgASgLMiguYW55dHR5LmNsb3VkLnYxLkNsb3VkRW50aXRsZW1lbnRG'
    'YWlsdXJlUhJlbnRpdGxlbWVudEZhaWx1cmU=');

@$core.Deprecated('Use relayRuntimePolicyDescriptor instead')
const RelayRuntimePolicy$json = {
  '1': 'RelayRuntimePolicy',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'subscription_id', '3': 2, '4': 1, '5': 9, '10': 'subscriptionId'},
    {'1': 'plan_id', '3': 3, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'policy_revision', '3': 4, '4': 1, '5': 4, '10': 'policyRevision'},
    {'1': 'relay_enabled', '3': 5, '4': 1, '5': 8, '10': 'relayEnabled'},
    {
      '1': 'relay_max_rate_bytes_per_second',
      '3': 6,
      '4': 1,
      '5': 4,
      '10': 'relayMaxRateBytesPerSecond'
    },
    {
      '1': 'relay_max_concurrency',
      '3': 7,
      '4': 1,
      '5': 13,
      '10': 'relayMaxConcurrency'
    },
    {'1': 'relay_quota_bytes', '3': 8, '4': 1, '5': 4, '10': 'relayQuotaBytes'},
    {
      '1': 'period_start',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodStart'
    },
    {
      '1': 'period_end',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodEnd'
    },
  ],
};

/// Descriptor for `RelayRuntimePolicy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayRuntimePolicyDescriptor = $convert.base64Decode(
    'ChJSZWxheVJ1bnRpbWVQb2xpY3kSHQoKYWNjb3VudF9pZBgBIAEoCVIJYWNjb3VudElkEicKD3'
    'N1YnNjcmlwdGlvbl9pZBgCIAEoCVIOc3Vic2NyaXB0aW9uSWQSFwoHcGxhbl9pZBgDIAEoCVIG'
    'cGxhbklkEicKD3BvbGljeV9yZXZpc2lvbhgEIAEoBFIOcG9saWN5UmV2aXNpb24SIwoNcmVsYX'
    'lfZW5hYmxlZBgFIAEoCFIMcmVsYXlFbmFibGVkEkMKH3JlbGF5X21heF9yYXRlX2J5dGVzX3Bl'
    'cl9zZWNvbmQYBiABKARSGnJlbGF5TWF4UmF0ZUJ5dGVzUGVyU2Vjb25kEjIKFXJlbGF5X21heF'
    '9jb25jdXJyZW5jeRgHIAEoDVITcmVsYXlNYXhDb25jdXJyZW5jeRIqChFyZWxheV9xdW90YV9i'
    'eXRlcxgIIAEoBFIPcmVsYXlRdW90YUJ5dGVzEj0KDHBlcmlvZF9zdGFydBgJIAEoCzIaLmdvb2'
    'dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC3BlcmlvZFN0YXJ0EjkKCnBlcmlvZF9lbmQYCiABKAsy'
    'Gi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUglwZXJpb2RFbmQ=');

@$core.Deprecated('Use relayAuthorizeRequestDescriptor instead')
const RelayAuthorizeRequest$json = {
  '1': 'RelayAuthorizeRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'daemon_id', '3': 3, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'session_id', '3': 4, '4': 1, '5': 9, '10': 'sessionId'},
    {
      '1': 'observed_at',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'observedAt'
    },
  ],
};

/// Descriptor for `RelayAuthorizeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayAuthorizeRequestDescriptor = $convert.base64Decode(
    'ChVSZWxheUF1dGhvcml6ZVJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdElkEh'
    '0KCmFjY291bnRfaWQYAiABKAlSCWFjY291bnRJZBIbCglkYWVtb25faWQYAyABKAlSCGRhZW1v'
    'bklkEh0KCnNlc3Npb25faWQYBCABKAlSCXNlc3Npb25JZBI7CgtvYnNlcnZlZF9hdBgFIAEoCz'
    'IaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCm9ic2VydmVkQXQ=');

@$core.Deprecated('Use relayAuthorizeResponseDescriptor instead')
const RelayAuthorizeResponse$json = {
  '1': 'RelayAuthorizeResponse',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'policy',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayRuntimePolicy',
      '10': 'policy'
    },
    {
      '1': 'entitlement_failure',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudEntitlementFailure',
      '10': 'entitlementFailure'
    },
  ],
};

/// Descriptor for `RelayAuthorizeResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayAuthorizeResponseDescriptor = $convert.base64Decode(
    'ChZSZWxheUF1dGhvcml6ZVJlc3BvbnNlEh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZB'
    'I7CgZwb2xpY3kYAiABKAsyIy5hbnl0dHkuY2xvdWQudjEuUmVsYXlSdW50aW1lUG9saWN5UgZw'
    'b2xpY3kSWQoTZW50aXRsZW1lbnRfZmFpbHVyZRgDIAEoCzIoLmFueXR0eS5jbG91ZC52MS5DbG'
    '91ZEVudGl0bGVtZW50RmFpbHVyZVISZW50aXRsZW1lbnRGYWlsdXJl');

@$core.Deprecated('Use relayUsageSampleDescriptor instead')
const RelayUsageSample$json = {
  '1': 'RelayUsageSample',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'cumulative_egress_bytes',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'cumulativeEgressBytes'
    },
    {
      '1': 'sampled_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'sampledAt'
    },
  ],
};

/// Descriptor for `RelayUsageSample`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayUsageSampleDescriptor = $convert.base64Decode(
    'ChBSZWxheVVzYWdlU2FtcGxlEh0KCmFjY291bnRfaWQYASABKAlSCWFjY291bnRJZBI2ChdjdW'
    '11bGF0aXZlX2VncmVzc19ieXRlcxgCIAEoBFIVY3VtdWxhdGl2ZUVncmVzc0J5dGVzEjkKCnNh'
    'bXBsZWRfYXQYAyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUglzYW1wbGVkQXQ=');

@$core.Deprecated('Use relayUsageBatchDescriptor instead')
const RelayUsageBatch$json = {
  '1': 'RelayUsageBatch',
  '2': [
    {'1': 'batch_sequence', '3': 1, '4': 1, '5': 4, '10': 'batchSequence'},
    {
      '1': 'samples',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayUsageSample',
      '10': 'samples'
    },
  ],
};

/// Descriptor for `RelayUsageBatch`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayUsageBatchDescriptor = $convert.base64Decode(
    'Cg9SZWxheVVzYWdlQmF0Y2gSJQoOYmF0Y2hfc2VxdWVuY2UYASABKARSDWJhdGNoU2VxdWVuY2'
    'USOwoHc2FtcGxlcxgCIAMoCzIhLmFueXR0eS5jbG91ZC52MS5SZWxheVVzYWdlU2FtcGxlUgdz'
    'YW1wbGVz');

@$core.Deprecated('Use relayAccountActionDescriptor instead')
const RelayAccountAction$json = {
  '1': 'RelayAccountAction',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'action',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayAccountActionType',
      '10': 'action'
    },
    {'1': 'action_revision', '3': 3, '4': 1, '5': 4, '10': 'actionRevision'},
    {'1': 'used_bytes', '3': 4, '4': 1, '5': 4, '10': 'usedBytes'},
    {'1': 'quota_bytes', '3': 5, '4': 1, '5': 4, '10': 'quotaBytes'},
    {
      '1': 'period_start',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodStart'
    },
    {
      '1': 'period_end',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodEnd'
    },
    {'1': 'reason', '3': 8, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `RelayAccountAction`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayAccountActionDescriptor = $convert.base64Decode(
    'ChJSZWxheUFjY291bnRBY3Rpb24SHQoKYWNjb3VudF9pZBgBIAEoCVIJYWNjb3VudElkEj8KBm'
    'FjdGlvbhgCIAEoDjInLmFueXR0eS5jbG91ZC52MS5SZWxheUFjY291bnRBY3Rpb25UeXBlUgZh'
    'Y3Rpb24SJwoPYWN0aW9uX3JldmlzaW9uGAMgASgEUg5hY3Rpb25SZXZpc2lvbhIdCgp1c2VkX2'
    'J5dGVzGAQgASgEUgl1c2VkQnl0ZXMSHwoLcXVvdGFfYnl0ZXMYBSABKARSCnF1b3RhQnl0ZXMS'
    'PQoMcGVyaW9kX3N0YXJ0GAYgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFILcGVyaW'
    '9kU3RhcnQSOQoKcGVyaW9kX2VuZBgHIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBS'
    'CXBlcmlvZEVuZBIWCgZyZWFzb24YCCABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use relayUsageAckDescriptor instead')
const RelayUsageAck$json = {
  '1': 'RelayUsageAck',
  '2': [
    {'1': 'batch_sequence', '3': 1, '4': 1, '5': 4, '10': 'batchSequence'},
    {
      '1': 'actions',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayAccountAction',
      '10': 'actions'
    },
    {
      '1': 'processed_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'processedAt'
    },
  ],
};

/// Descriptor for `RelayUsageAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayUsageAckDescriptor = $convert.base64Decode(
    'Cg1SZWxheVVzYWdlQWNrEiUKDmJhdGNoX3NlcXVlbmNlGAEgASgEUg1iYXRjaFNlcXVlbmNlEj'
    '0KB2FjdGlvbnMYAiADKAsyIy5hbnl0dHkuY2xvdWQudjEuUmVsYXlBY2NvdW50QWN0aW9uUgdh'
    'Y3Rpb25zEj0KDHByb2Nlc3NlZF9hdBgDIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbX'
    'BSC3Byb2Nlc3NlZEF0');

@$core.Deprecated('Use relayICEConfigDescriptor instead')
const RelayICEConfig$json = {
  '1': 'RelayICEConfig',
  '2': [
    {'1': 'reservation_id', '3': 1, '4': 1, '5': 9, '10': 'reservationId'},
    {'1': 'urls', '3': 2, '4': 3, '5': 9, '10': 'urls'},
    {'1': 'username', '3': 3, '4': 1, '5': 9, '10': 'username'},
    {'1': 'credential', '3': 4, '4': 1, '5': 9, '10': 'credential'},
    {
      '1': 'expires_at',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
  ],
};

/// Descriptor for `RelayICEConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayICEConfigDescriptor = $convert.base64Decode(
    'Cg5SZWxheUlDRUNvbmZpZxIlCg5yZXNlcnZhdGlvbl9pZBgBIAEoCVINcmVzZXJ2YXRpb25JZB'
    'ISCgR1cmxzGAIgAygJUgR1cmxzEhoKCHVzZXJuYW1lGAMgASgJUgh1c2VybmFtZRIeCgpjcmVk'
    'ZW50aWFsGAQgASgJUgpjcmVkZW50aWFsEjkKCmV4cGlyZXNfYXQYBSABKAsyGi5nb29nbGUucH'
    'JvdG9idWYuVGltZXN0YW1wUglleHBpcmVzQXQ=');

@$core.Deprecated('Use relayJournalRecordDescriptor instead')
const RelayJournalRecord$json = {
  '1': 'RelayJournalRecord',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {
      '1': 'stage',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayJournalStage',
      '10': 'stage'
    },
    {
      '1': 'reserve_request',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayReserveRequest',
      '10': 'reserveRequest'
    },
    {
      '1': 'grant',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayGrant',
      '10': 'grant'
    },
    {
      '1': 'pending_renew_sequence',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'pendingRenewSequence'
    },
    {
      '1': 'settlement',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelaySettlement',
      '10': 'settlement'
    },
  ],
};

/// Descriptor for `RelayJournalRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List relayJournalRecordDescriptor = $convert.base64Decode(
    'ChJSZWxheUpvdXJuYWxSZWNvcmQSJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDXNjaGVtYVZlcn'
    'Npb24SOAoFc3RhZ2UYAiABKA4yIi5hbnl0dHkuY2xvdWQudjEuUmVsYXlKb3VybmFsU3RhZ2VS'
    'BXN0YWdlEk0KD3Jlc2VydmVfcmVxdWVzdBgDIAEoCzIkLmFueXR0eS5jbG91ZC52MS5SZWxheV'
    'Jlc2VydmVSZXF1ZXN0Ug5yZXNlcnZlUmVxdWVzdBIxCgVncmFudBgEIAEoCzIbLmFueXR0eS5j'
    'bG91ZC52MS5SZWxheUdyYW50UgVncmFudBI0ChZwZW5kaW5nX3JlbmV3X3NlcXVlbmNlGAUgAS'
    'gEUhRwZW5kaW5nUmVuZXdTZXF1ZW5jZRJACgpzZXR0bGVtZW50GAYgASgLMiAuYW55dHR5LmNs'
    'b3VkLnYxLlJlbGF5U2V0dGxlbWVudFIKc2V0dGxlbWVudA==');
