// This is a generated file - do not edit.
//
// Generated from cloud/v1/ticket.proto.

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

@$core.Deprecated('Use daemonCapabilityDescriptor instead')
const DaemonCapability$json = {
  '1': 'DaemonCapability',
  '2': [
    {'1': 'DAEMON_CAPABILITY_UNSPECIFIED', '2': 0},
    {'1': 'DAEMON_CAPABILITY_SIGNALING', '2': 1},
  ],
};

/// Descriptor for `DaemonCapability`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List daemonCapabilityDescriptor = $convert.base64Decode(
    'ChBEYWVtb25DYXBhYmlsaXR5EiEKHURBRU1PTl9DQVBBQklMSVRZX1VOU1BFQ0lGSUVEEAASHw'
    'obREFFTU9OX0NBUEFCSUxJVFlfU0lHTkFMSU5HEAE=');

@$core.Deprecated('Use daemonBindingClaimsDescriptor instead')
const DaemonBindingClaims$json = {
  '1': 'DaemonBindingClaims',
  '2': [
    {'1': 'binding_id', '3': 1, '4': 1, '5': 9, '10': 'bindingId'},
    {'1': 'daemon_id', '3': 2, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'account_id', '3': 3, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'edge_id', '3': 4, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'device_id', '3': 5, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_public_key',
      '3': 6,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
    {
      '1': 'capabilities',
      '3': 7,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.DaemonCapability',
      '10': 'capabilities'
    },
    {
      '1': 'issued_at',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'issuedAt'
    },
    {
      '1': 'expires_at',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
    {
      '1': 'edge_locator_sha256',
      '3': 12,
      '4': 1,
      '5': 12,
      '10': 'edgeLocatorSha256'
    },
  ],
  '9': [
    {'1': 10, '2': 11},
    {'1': 11, '2': 12},
  ],
};

/// Descriptor for `DaemonBindingClaims`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonBindingClaimsDescriptor = $convert.base64Decode(
    'ChNEYWVtb25CaW5kaW5nQ2xhaW1zEh0KCmJpbmRpbmdfaWQYASABKAlSCWJpbmRpbmdJZBIbCg'
    'lkYWVtb25faWQYAiABKAlSCGRhZW1vbklkEh0KCmFjY291bnRfaWQYAyABKAlSCWFjY291bnRJ'
    'ZBIXCgdlZGdlX2lkGAQgASgJUgZlZGdlSWQSGwoJZGV2aWNlX2lkGAUgASgJUghkZXZpY2VJZB'
    'IqChFkZXZpY2VfcHVibGljX2tleRgGIAEoDFIPZGV2aWNlUHVibGljS2V5EkUKDGNhcGFiaWxp'
    'dGllcxgHIAMoDjIhLmFueXR0eS5jbG91ZC52MS5EYWVtb25DYXBhYmlsaXR5UgxjYXBhYmlsaX'
    'RpZXMSNwoJaXNzdWVkX2F0GAggASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIIaXNz'
    'dWVkQXQSOQoKZXhwaXJlc19hdBgJIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCW'
    'V4cGlyZXNBdBIuChNlZGdlX2xvY2F0b3Jfc2hhMjU2GAwgASgMUhFlZGdlTG9jYXRvclNoYTI1'
    'NkoECAoQC0oECAsQDA==');

@$core.Deprecated('Use agentHelloProofInputDescriptor instead')
const AgentHelloProofInput$json = {
  '1': 'AgentHelloProofInput',
  '2': [
    {
      '1': 'binding_envelope_sha256',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'bindingEnvelopeSha256'
    },
    {'1': 'daemon_id', '3': 2, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'daemon_boot_id', '3': 3, '4': 1, '5': 9, '10': 'daemonBootId'},
    {'1': 'daemon_session_id', '3': 4, '4': 1, '5': 9, '10': 'daemonSessionId'},
    {
      '1': 'challenge',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeChallenge',
      '10': 'challenge'
    },
    {'1': 'protocol_version', '3': 6, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'message_id', '3': 7, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'stream_seq', '3': 8, '4': 1, '5': 4, '10': 'streamSeq'},
    {
      '1': 'sent_at',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'sentAt'
    },
    {'1': 'software_version', '3': 10, '4': 1, '5': 9, '10': 'softwareVersion'},
    {
      '1': 'attempt_generation',
      '3': 11,
      '4': 1,
      '5': 4,
      '10': 'attemptGeneration'
    },
  ],
};

/// Descriptor for `AgentHelloProofInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentHelloProofInputDescriptor = $convert.base64Decode(
    'ChRBZ2VudEhlbGxvUHJvb2ZJbnB1dBI2ChdiaW5kaW5nX2VudmVsb3BlX3NoYTI1NhgBIAEoDF'
    'IVYmluZGluZ0VudmVsb3BlU2hhMjU2EhsKCWRhZW1vbl9pZBgCIAEoCVIIZGFlbW9uSWQSJAoO'
    'ZGFlbW9uX2Jvb3RfaWQYAyABKAlSDGRhZW1vbkJvb3RJZBIqChFkYWVtb25fc2Vzc2lvbl9pZB'
    'gEIAEoCVIPZGFlbW9uU2Vzc2lvbklkEjwKCWNoYWxsZW5nZRgFIAEoCzIeLmFueXR0eS5jbG91'
    'ZC52MS5FZGdlQ2hhbGxlbmdlUgljaGFsbGVuZ2USKQoQcHJvdG9jb2xfdmVyc2lvbhgGIAEoDV'
    'IPcHJvdG9jb2xWZXJzaW9uEh0KCm1lc3NhZ2VfaWQYByABKAlSCW1lc3NhZ2VJZBIdCgpzdHJl'
    'YW1fc2VxGAggASgEUglzdHJlYW1TZXESMwoHc2VudF9hdBgJIAEoCzIaLmdvb2dsZS5wcm90b2'
    'J1Zi5UaW1lc3RhbXBSBnNlbnRBdBIpChBzb2Z0d2FyZV92ZXJzaW9uGAogASgJUg9zb2Z0d2Fy'
    'ZVZlcnNpb24SLQoSYXR0ZW1wdF9nZW5lcmF0aW9uGAsgASgEUhFhdHRlbXB0R2VuZXJhdGlvbg'
    '==');

@$core.Deprecated('Use cloudRouteGrantClaimsDescriptor instead')
const CloudRouteGrantClaims$json = {
  '1': 'CloudRouteGrantClaims',
  '2': [
    {'1': 'grant_id', '3': 1, '4': 1, '5': 9, '10': 'grantId'},
    {'1': 'daemon_id', '3': 2, '4': 1, '5': 9, '10': 'daemonId'},
    {
      '1': 'client_public_key',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {
      '1': 'product',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.ClientProduct',
      '10': 'product'
    },
    {
      '1': 'issued_at',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'issuedAt'
    },
    {
      '1': 'expires_at',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
  ],
};

/// Descriptor for `CloudRouteGrantClaims`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudRouteGrantClaimsDescriptor = $convert.base64Decode(
    'ChVDbG91ZFJvdXRlR3JhbnRDbGFpbXMSGQoIZ3JhbnRfaWQYASABKAlSB2dyYW50SWQSGwoJZG'
    'FlbW9uX2lkGAIgASgJUghkYWVtb25JZBIqChFjbGllbnRfcHVibGljX2tleRgDIAEoDFIPY2xp'
    'ZW50UHVibGljS2V5EjgKB3Byb2R1Y3QYBCABKA4yHi5hbnl0dHkuY2xvdWQudjEuQ2xpZW50UH'
    'JvZHVjdFIHcHJvZHVjdBI3Cglpc3N1ZWRfYXQYBSABKAsyGi5nb29nbGUucHJvdG9idWYuVGlt'
    'ZXN0YW1wUghpc3N1ZWRBdBI5CgpleHBpcmVzX2F0GAYgASgLMhouZ29vZ2xlLnByb3RvYnVmLl'
    'RpbWVzdGFtcFIJZXhwaXJlc0F0');

@$core.Deprecated('Use clientRouteProofInputDescriptor instead')
const ClientRouteProofInput$json = {
  '1': 'ClientRouteProofInput',
  '2': [
    {'1': 'challenge_id', '3': 1, '4': 1, '5': 9, '10': 'challengeId'},
    {'1': 'challenge', '3': 2, '4': 1, '5': 12, '10': 'challenge'},
    {
      '1': 'grant_payload_sha256',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'grantPayloadSha256'
    },
    {'1': 'request_id', '3': 4, '4': 1, '5': 9, '10': 'requestId'},
  ],
};

/// Descriptor for `ClientRouteProofInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientRouteProofInputDescriptor = $convert.base64Decode(
    'ChVDbGllbnRSb3V0ZVByb29mSW5wdXQSIQoMY2hhbGxlbmdlX2lkGAEgASgJUgtjaGFsbGVuZ2'
    'VJZBIcCgljaGFsbGVuZ2UYAiABKAxSCWNoYWxsZW5nZRIwChRncmFudF9wYXlsb2FkX3NoYTI1'
    'NhgDIAEoDFISZ3JhbnRQYXlsb2FkU2hhMjU2Eh0KCnJlcXVlc3RfaWQYBCABKAlSCXJlcXVlc3'
    'RJZA==');

@$core.Deprecated('Use gatewayClientHelloProofInputDescriptor instead')
const GatewayClientHelloProofInput$json = {
  '1': 'GatewayClientHelloProofInput',
  '2': [
    {
      '1': 'challenge',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeChallenge',
      '10': 'challenge'
    },
    {
      '1': 'authorization_sha256',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'authorizationSha256'
    },
    {
      '1': 'access_mode',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudClientAccessMode',
      '10': 'accessMode'
    },
    {'1': 'protocol_version', '3': 4, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'message_id', '3': 5, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'client_id', '3': 6, '4': 1, '5': 9, '10': 'clientId'},
    {'1': 'client_boot_id', '3': 7, '4': 1, '5': 9, '10': 'clientBootId'},
    {'1': 'client_session_id', '3': 8, '4': 1, '5': 9, '10': 'clientSessionId'},
    {'1': 'stream_seq', '3': 9, '4': 1, '5': 4, '10': 'streamSeq'},
    {
      '1': 'sent_at',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'sentAt'
    },
    {
      '1': 'client_public_key',
      '3': 11,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {
      '1': 'product',
      '3': 12,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.ClientProduct',
      '10': 'product'
    },
    {'1': 'software_version', '3': 13, '4': 1, '5': 9, '10': 'softwareVersion'},
    {
      '1': 'attempt_generation',
      '3': 14,
      '4': 1,
      '5': 4,
      '10': 'attemptGeneration'
    },
    {
      '1': 'relay_preference',
      '3': 15,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayPreference',
      '10': 'relayPreference'
    },
    {'1': 'presence_probe', '3': 16, '4': 1, '5': 8, '10': 'presenceProbe'},
  ],
};

/// Descriptor for `GatewayClientHelloProofInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gatewayClientHelloProofInputDescriptor = $convert.base64Decode(
    'ChxHYXRld2F5Q2xpZW50SGVsbG9Qcm9vZklucHV0EjwKCWNoYWxsZW5nZRgBIAEoCzIeLmFueX'
    'R0eS5jbG91ZC52MS5FZGdlQ2hhbGxlbmdlUgljaGFsbGVuZ2USMQoUYXV0aG9yaXphdGlvbl9z'
    'aGEyNTYYAiABKAxSE2F1dGhvcml6YXRpb25TaGEyNTYSRwoLYWNjZXNzX21vZGUYAyABKA4yJi'
    '5hbnl0dHkuY2xvdWQudjEuQ2xvdWRDbGllbnRBY2Nlc3NNb2RlUgphY2Nlc3NNb2RlEikKEHBy'
    'b3RvY29sX3ZlcnNpb24YBCABKA1SD3Byb3RvY29sVmVyc2lvbhIdCgptZXNzYWdlX2lkGAUgAS'
    'gJUgltZXNzYWdlSWQSGwoJY2xpZW50X2lkGAYgASgJUghjbGllbnRJZBIkCg5jbGllbnRfYm9v'
    'dF9pZBgHIAEoCVIMY2xpZW50Qm9vdElkEioKEWNsaWVudF9zZXNzaW9uX2lkGAggASgJUg9jbG'
    'llbnRTZXNzaW9uSWQSHQoKc3RyZWFtX3NlcRgJIAEoBFIJc3RyZWFtU2VxEjMKB3NlbnRfYXQY'
    'CiABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgZzZW50QXQSKgoRY2xpZW50X3B1Ym'
    'xpY19rZXkYCyABKAxSD2NsaWVudFB1YmxpY0tleRI4Cgdwcm9kdWN0GAwgASgOMh4uYW55dHR5'
    'LmNsb3VkLnYxLkNsaWVudFByb2R1Y3RSB3Byb2R1Y3QSKQoQc29mdHdhcmVfdmVyc2lvbhgNIA'
    'EoCVIPc29mdHdhcmVWZXJzaW9uEi0KEmF0dGVtcHRfZ2VuZXJhdGlvbhgOIAEoBFIRYXR0ZW1w'
    'dEdlbmVyYXRpb24SSwoQcmVsYXlfcHJlZmVyZW5jZRgPIAEoDjIgLmFueXR0eS5jbG91ZC52MS'
    '5SZWxheVByZWZlcmVuY2VSD3JlbGF5UHJlZmVyZW5jZRIlCg5wcmVzZW5jZV9wcm9iZRgQIAEo'
    'CFINcHJlc2VuY2VQcm9iZQ==');
