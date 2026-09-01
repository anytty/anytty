// This is a generated file - do not edit.
//
// Generated from cloud/v1/client_gateway.proto.

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
    as $2;

import 'common.pbjson.dart' as $0;
import 'usage.pbjson.dart' as $1;

@$core.Deprecated('Use selectedCloudPathDescriptor instead')
const SelectedCloudPath$json = {
  '1': 'SelectedCloudPath',
  '2': [
    {'1': 'SELECTED_CLOUD_PATH_UNSPECIFIED', '2': 0},
    {'1': 'SELECTED_CLOUD_PATH_DIRECT', '2': 1},
    {'1': 'SELECTED_CLOUD_PATH_RELAY', '2': 2},
  ],
};

/// Descriptor for `SelectedCloudPath`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List selectedCloudPathDescriptor = $convert.base64Decode(
    'ChFTZWxlY3RlZENsb3VkUGF0aBIjCh9TRUxFQ1RFRF9DTE9VRF9QQVRIX1VOU1BFQ0lGSUVEEA'
    'ASHgoaU0VMRUNURURfQ0xPVURfUEFUSF9ESVJFQ1QQARIdChlTRUxFQ1RFRF9DTE9VRF9QQVRI'
    'X1JFTEFZEAI=');

@$core.Deprecated('Use cloudPathDecisionDescriptor instead')
const CloudPathDecision$json = {
  '1': 'CloudPathDecision',
  '2': [
    {'1': 'CLOUD_PATH_DECISION_UNSPECIFIED', '2': 0},
    {'1': 'CLOUD_PATH_DECISION_CONFIRM_DIRECT', '2': 1},
    {'1': 'CLOUD_PATH_DECISION_CONFIRM_RELAY', '2': 2},
    {'1': 'CLOUD_PATH_DECISION_ABANDON', '2': 3},
  ],
};

/// Descriptor for `CloudPathDecision`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List cloudPathDecisionDescriptor = $convert.base64Decode(
    'ChFDbG91ZFBhdGhEZWNpc2lvbhIjCh9DTE9VRF9QQVRIX0RFQ0lTSU9OX1VOU1BFQ0lGSUVEEA'
    'ASJgoiQ0xPVURfUEFUSF9ERUNJU0lPTl9DT05GSVJNX0RJUkVDVBABEiUKIUNMT1VEX1BBVEhf'
    'REVDSVNJT05fQ09ORklSTV9SRUxBWRACEh8KG0NMT1VEX1BBVEhfREVDSVNJT05fQUJBTkRPTh'
    'AD');

@$core.Deprecated('Use signalSessionCloseCodeDescriptor instead')
const SignalSessionCloseCode$json = {
  '1': 'SignalSessionCloseCode',
  '2': [
    {'1': 'SIGNAL_SESSION_CLOSE_CODE_UNSPECIFIED', '2': 0},
    {'1': 'SIGNAL_SESSION_CLOSE_CODE_ADMIN_DISCONNECT', '2': 1},
  ],
};

/// Descriptor for `SignalSessionCloseCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List signalSessionCloseCodeDescriptor = $convert.base64Decode(
    'ChZTaWduYWxTZXNzaW9uQ2xvc2VDb2RlEikKJVNJR05BTF9TRVNTSU9OX0NMT1NFX0NPREVfVU'
    '5TUEVDSUZJRUQQABIuCipTSUdOQUxfU0VTU0lPTl9DTE9TRV9DT0RFX0FETUlOX0RJU0NPTk5F'
    'Q1QQAQ==');

@$core.Deprecated('Use cloudICECandidateDescriptor instead')
const CloudICECandidate$json = {
  '1': 'CloudICECandidate',
  '2': [
    {'1': 'candidate', '3': 1, '4': 1, '5': 9, '10': 'candidate'},
    {'1': 'sdp_mid', '3': 2, '4': 1, '5': 9, '10': 'sdpMid'},
    {'1': 'sdp_mline_index', '3': 3, '4': 1, '5': 13, '10': 'sdpMlineIndex'},
    {
      '1': 'username_fragment',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'usernameFragment'
    },
  ],
};

/// Descriptor for `CloudICECandidate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudICECandidateDescriptor = $convert.base64Decode(
    'ChFDbG91ZElDRUNhbmRpZGF0ZRIcCgljYW5kaWRhdGUYASABKAlSCWNhbmRpZGF0ZRIXCgdzZH'
    'BfbWlkGAIgASgJUgZzZHBNaWQSJgoPc2RwX21saW5lX2luZGV4GAMgASgNUg1zZHBNbGluZUlu'
    'ZGV4EisKEXVzZXJuYW1lX2ZyYWdtZW50GAQgASgJUhB1c2VybmFtZUZyYWdtZW50');

@$core.Deprecated('Use pairingAdmissionDescriptor instead')
const PairingAdmission$json = {
  '1': 'PairingAdmission',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_public_key',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
    {
      '1': 'pairing_claim_sha256',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'pairingClaimSha256'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
  ],
};

/// Descriptor for `PairingAdmission`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingAdmissionDescriptor = $convert.base64Decode(
    'ChBQYWlyaW5nQWRtaXNzaW9uEhsKCWRhZW1vbl9pZBgBIAEoCVIIZGFlbW9uSWQSGwoJZGV2aW'
    'NlX2lkGAIgASgJUghkZXZpY2VJZBIqChFkZXZpY2VfcHVibGljX2tleRgDIAEoDFIPZGV2aWNl'
    'UHVibGljS2V5EjAKFHBhaXJpbmdfY2xhaW1fc2hhMjU2GAQgASgMUhJwYWlyaW5nQ2xhaW1TaG'
    'EyNTYSLwoUZXhwaXJlc19hdF91bml4X25hbm8YBSABKANSEWV4cGlyZXNBdFVuaXhOYW5v');

@$core.Deprecated('Use clientHelloDescriptor instead')
const ClientHello$json = {
  '1': 'ClientHello',
  '2': [
    {
      '1': 'client_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {'1': 'client_proof', '3': 3, '4': 1, '5': 12, '10': 'clientProof'},
    {
      '1': 'product',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.ClientProduct',
      '10': 'product'
    },
    {'1': 'software_version', '3': 5, '4': 1, '5': 9, '10': 'softwareVersion'},
    {
      '1': 'attempt_generation',
      '3': 6,
      '4': 1,
      '5': 4,
      '10': 'attemptGeneration'
    },
    {
      '1': 'relay_preference',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.RelayPreference',
      '10': 'relayPreference'
    },
    {'1': 'presence_probe', '3': 8, '4': 1, '5': 8, '10': 'presenceProbe'},
    {
      '1': 'cloud_route_grant',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEnvelope',
      '9': 0,
      '10': 'cloudRouteGrant'
    },
    {
      '1': 'pairing_admission',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PairingAdmission',
      '9': 0,
      '10': 'pairingAdmission'
    },
  ],
  '8': [
    {'1': 'authorization'},
  ],
};

/// Descriptor for `ClientHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientHelloDescriptor = $convert.base64Decode(
    'CgtDbGllbnRIZWxsbxIqChFjbGllbnRfcHVibGljX2tleRgCIAEoDFIPY2xpZW50UHVibGljS2'
    'V5EiEKDGNsaWVudF9wcm9vZhgDIAEoDFILY2xpZW50UHJvb2YSOAoHcHJvZHVjdBgEIAEoDjIe'
    'LmFueXR0eS5jbG91ZC52MS5DbGllbnRQcm9kdWN0Ugdwcm9kdWN0EikKEHNvZnR3YXJlX3Zlcn'
    'Npb24YBSABKAlSD3NvZnR3YXJlVmVyc2lvbhItChJhdHRlbXB0X2dlbmVyYXRpb24YBiABKARS'
    'EWF0dGVtcHRHZW5lcmF0aW9uEksKEHJlbGF5X3ByZWZlcmVuY2UYByABKA4yIC5hbnl0dHkuY2'
    'xvdWQudjEuUmVsYXlQcmVmZXJlbmNlUg9yZWxheVByZWZlcmVuY2USJQoOcHJlc2VuY2VfcHJv'
    'YmUYCCABKAhSDXByZXNlbmNlUHJvYmUSTQoRY2xvdWRfcm91dGVfZ3JhbnQYCiABKAsyHy5hbn'
    'l0dHkuY2xvdWQudjEuU2lnbmVkRW52ZWxvcGVIAFIPY2xvdWRSb3V0ZUdyYW50ElAKEXBhaXJp'
    'bmdfYWRtaXNzaW9uGAsgASgLMiEuYW55dHR5LmNsb3VkLnYxLlBhaXJpbmdBZG1pc3Npb25IAF'
    'IQcGFpcmluZ0FkbWlzc2lvbkIPCg1hdXRob3JpemF0aW9u');

@$core.Deprecated('Use clientReadyDescriptor instead')
const ClientReady$json = {
  '1': 'ClientReady',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'relay',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayICEConfig',
      '10': 'relay'
    },
    {
      '1': 'relay_failure',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudEntitlementFailure',
      '10': 'relayFailure'
    },
  ],
};

/// Descriptor for `ClientReady`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientReadyDescriptor = $convert.base64Decode(
    'CgtDbGllbnRSZWFkeRIdCgpzZXNzaW9uX2lkGAEgASgJUglzZXNzaW9uSWQSHgoKZ2VuZXJhdG'
    'lvbhgCIAEoBFIKZ2VuZXJhdGlvbhI1CgVyZWxheRgDIAEoCzIfLmFueXR0eS5jbG91ZC52MS5S'
    'ZWxheUlDRUNvbmZpZ1IFcmVsYXkSTQoNcmVsYXlfZmFpbHVyZRgEIAEoCzIoLmFueXR0eS5jbG'
    '91ZC52MS5DbG91ZEVudGl0bGVtZW50RmFpbHVyZVIMcmVsYXlGYWlsdXJl');

@$core.Deprecated('Use clientPathDecisionDescriptor instead')
const ClientPathDecision$json = {
  '1': 'ClientPathDecision',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'decision_id', '3': 2, '4': 1, '5': 9, '10': 'decisionId'},
    {
      '1': 'decision',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudPathDecision',
      '10': 'decision'
    },
  ],
};

/// Descriptor for `ClientPathDecision`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientPathDecisionDescriptor = $convert.base64Decode(
    'ChJDbGllbnRQYXRoRGVjaXNpb24SHQoKc2Vzc2lvbl9pZBgBIAEoCVIJc2Vzc2lvbklkEh8KC2'
    'RlY2lzaW9uX2lkGAIgASgJUgpkZWNpc2lvbklkEj4KCGRlY2lzaW9uGAMgASgOMiIuYW55dHR5'
    'LmNsb3VkLnYxLkNsb3VkUGF0aERlY2lzaW9uUghkZWNpc2lvbg==');

@$core.Deprecated('Use edgePathDecisionAckDescriptor instead')
const EdgePathDecisionAck$json = {
  '1': 'EdgePathDecisionAck',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'decision_id', '3': 2, '4': 1, '5': 9, '10': 'decisionId'},
    {
      '1': 'decision',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudPathDecision',
      '10': 'decision'
    },
  ],
};

/// Descriptor for `EdgePathDecisionAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgePathDecisionAckDescriptor = $convert.base64Decode(
    'ChNFZGdlUGF0aERlY2lzaW9uQWNrEh0KCnNlc3Npb25faWQYASABKAlSCXNlc3Npb25JZBIfCg'
    'tkZWNpc2lvbl9pZBgCIAEoCVIKZGVjaXNpb25JZBI+CghkZWNpc2lvbhgDIAEoDjIiLmFueXR0'
    'eS5jbG91ZC52MS5DbG91ZFBhdGhEZWNpc2lvblIIZGVjaXNpb24=');

@$core.Deprecated('Use clientSessionReleaseDescriptor instead')
const ClientSessionRelease$json = {
  '1': 'ClientSessionRelease',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'release_id', '3': 2, '4': 1, '5': 9, '10': 'releaseId'},
  ],
};

/// Descriptor for `ClientSessionRelease`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientSessionReleaseDescriptor = $convert.base64Decode(
    'ChRDbGllbnRTZXNzaW9uUmVsZWFzZRIdCgpzZXNzaW9uX2lkGAEgASgJUglzZXNzaW9uSWQSHQ'
    'oKcmVsZWFzZV9pZBgCIAEoCVIJcmVsZWFzZUlk');

@$core.Deprecated('Use edgeSessionReleaseAckDescriptor instead')
const EdgeSessionReleaseAck$json = {
  '1': 'EdgeSessionReleaseAck',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'release_id', '3': 2, '4': 1, '5': 9, '10': 'releaseId'},
  ],
};

/// Descriptor for `EdgeSessionReleaseAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeSessionReleaseAckDescriptor = $convert.base64Decode(
    'ChVFZGdlU2Vzc2lvblJlbGVhc2VBY2sSHQoKc2Vzc2lvbl9pZBgBIAEoCVIJc2Vzc2lvbklkEh'
    '0KCnJlbGVhc2VfaWQYAiABKAlSCXJlbGVhc2VJZA==');

@$core.Deprecated('Use clientOfferDescriptor instead')
const ClientOffer$json = {
  '1': 'ClientOffer',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'offer_sdp', '3': 2, '4': 1, '5': 9, '10': 'offerSdp'},
    {
      '1': 'candidates',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudICECandidate',
      '10': 'candidates'
    },
  ],
};

/// Descriptor for `ClientOffer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientOfferDescriptor = $convert.base64Decode(
    'CgtDbGllbnRPZmZlchIdCgpzZXNzaW9uX2lkGAEgASgJUglzZXNzaW9uSWQSGwoJb2ZmZXJfc2'
    'RwGAIgASgJUghvZmZlclNkcBJCCgpjYW5kaWRhdGVzGAMgAygLMiIuYW55dHR5LmNsb3VkLnYx'
    'LkNsb3VkSUNFQ2FuZGlkYXRlUgpjYW5kaWRhdGVz');

@$core.Deprecated('Use edgeAnswerDescriptor instead')
const EdgeAnswer$json = {
  '1': 'EdgeAnswer',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'answer_sdp', '3': 2, '4': 1, '5': 9, '10': 'answerSdp'},
    {
      '1': 'candidates',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudICECandidate',
      '10': 'candidates'
    },
  ],
};

/// Descriptor for `EdgeAnswer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeAnswerDescriptor = $convert.base64Decode(
    'CgpFZGdlQW5zd2VyEh0KCnNlc3Npb25faWQYASABKAlSCXNlc3Npb25JZBIdCgphbnN3ZXJfc2'
    'RwGAIgASgJUglhbnN3ZXJTZHASQgoKY2FuZGlkYXRlcxgDIAMoCzIiLmFueXR0eS5jbG91ZC52'
    'MS5DbG91ZElDRUNhbmRpZGF0ZVIKY2FuZGlkYXRlcw==');

@$core.Deprecated('Use signalRejectedDescriptor instead')
const SignalRejected$json = {
  '1': 'SignalRejected',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'code', '3': 2, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
    {
      '1': 'entitlement_failure',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudEntitlementFailure',
      '10': 'entitlementFailure'
    },
  ],
};

/// Descriptor for `SignalRejected`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signalRejectedDescriptor = $convert.base64Decode(
    'Cg5TaWduYWxSZWplY3RlZBIdCgpzZXNzaW9uX2lkGAEgASgJUglzZXNzaW9uSWQSEgoEY29kZR'
    'gCIAEoCVIEY29kZRIYCgdtZXNzYWdlGAMgASgJUgdtZXNzYWdlElkKE2VudGl0bGVtZW50X2Zh'
    'aWx1cmUYBCABKAsyKC5hbnl0dHkuY2xvdWQudjEuQ2xvdWRFbnRpdGxlbWVudEZhaWx1cmVSEm'
    'VudGl0bGVtZW50RmFpbHVyZQ==');

@$core.Deprecated('Use signalSessionClosedDescriptor instead')
const SignalSessionClosed$json = {
  '1': 'SignalSessionClosed',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 9, '10': 'sessionId'},
    {
      '1': 'code',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SignalSessionCloseCode',
      '10': 'code'
    },
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `SignalSessionClosed`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signalSessionClosedDescriptor = $convert.base64Decode(
    'ChNTaWduYWxTZXNzaW9uQ2xvc2VkEh0KCnNlc3Npb25faWQYASABKAlSCXNlc3Npb25JZBI7Cg'
    'Rjb2RlGAIgASgOMicuYW55dHR5LmNsb3VkLnYxLlNpZ25hbFNlc3Npb25DbG9zZUNvZGVSBGNv'
    'ZGUSGAoHbWVzc2FnZRgDIAEoCVIHbWVzc2FnZQ==');

@$core.Deprecated('Use daemonPresenceDescriptor instead')
const DaemonPresence$json = {
  '1': 'DaemonPresence',
  '2': [
    {'1': 'online', '3': 1, '4': 1, '5': 8, '10': 'online'},
  ],
};

/// Descriptor for `DaemonPresence`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonPresenceDescriptor = $convert
    .base64Decode('Cg5EYWVtb25QcmVzZW5jZRIWCgZvbmxpbmUYASABKAhSBm9ubGluZQ==');

@$core.Deprecated('Use clientSignalDescriptor instead')
const ClientSignal$json = {
  '1': 'ClientSignal',
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
      '6': '.anytty.cloud.v1.ClientHello',
      '9': 0,
      '10': 'hello'
    },
    {
      '1': 'offer',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientOffer',
      '9': 0,
      '10': 'offer'
    },
    {
      '1': 'path_decision',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientPathDecision',
      '9': 0,
      '10': 'pathDecision'
    },
    {
      '1': 'session_release',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientSessionRelease',
      '9': 0,
      '10': 'sessionRelease'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `ClientSignal`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientSignalDescriptor = $convert.base64Decode(
    'CgxDbGllbnRTaWduYWwSKQoQcHJvdG9jb2xfdmVyc2lvbhgBIAEoDVIPcHJvdG9jb2xWZXJzaW'
    '9uEh0KCm1lc3NhZ2VfaWQYAiABKAlSCW1lc3NhZ2VJZBIbCglzZW5kZXJfaWQYAyABKAlSCHNl'
    'bmRlcklkEhcKB2Jvb3RfaWQYBCABKAlSBmJvb3RJZBIjCg1jb25uZWN0aW9uX2lkGAUgASgJUg'
    'xjb25uZWN0aW9uSWQSHQoKc3RyZWFtX3NlcRgGIAEoBFIJc3RyZWFtU2VxEjMKB3NlbnRfYXQY'
    'ByABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgZzZW50QXQSNAoFaGVsbG8YFCABKA'
    'syHC5hbnl0dHkuY2xvdWQudjEuQ2xpZW50SGVsbG9IAFIFaGVsbG8SNAoFb2ZmZXIYFSABKAsy'
    'HC5hbnl0dHkuY2xvdWQudjEuQ2xpZW50T2ZmZXJIAFIFb2ZmZXISSgoNcGF0aF9kZWNpc2lvbh'
    'gWIAEoCzIjLmFueXR0eS5jbG91ZC52MS5DbGllbnRQYXRoRGVjaXNpb25IAFIMcGF0aERlY2lz'
    'aW9uElAKD3Nlc3Npb25fcmVsZWFzZRgXIAEoCzIlLmFueXR0eS5jbG91ZC52MS5DbGllbnRTZX'
    'NzaW9uUmVsZWFzZUgAUg5zZXNzaW9uUmVsZWFzZUIJCgdwYXlsb2Fk');

@$core.Deprecated('Use edgeSignalDescriptor instead')
const EdgeSignal$json = {
  '1': 'EdgeSignal',
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
      '1': 'ready',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.ClientReady',
      '9': 0,
      '10': 'ready'
    },
    {
      '1': 'answer',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeAnswer',
      '9': 0,
      '10': 'answer'
    },
    {
      '1': 'rejected',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignalRejected',
      '9': 0,
      '10': 'rejected'
    },
    {
      '1': 'challenge',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeChallenge',
      '9': 0,
      '10': 'challenge'
    },
    {
      '1': 'closed',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignalSessionClosed',
      '9': 0,
      '10': 'closed'
    },
    {
      '1': 'presence',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonPresence',
      '9': 0,
      '10': 'presence'
    },
    {
      '1': 'path_decision_ack',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePathDecisionAck',
      '9': 0,
      '10': 'pathDecisionAck'
    },
    {
      '1': 'session_release_ack',
      '3': 27,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeSessionReleaseAck',
      '9': 0,
      '10': 'sessionReleaseAck'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `EdgeSignal`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeSignalDescriptor = $convert.base64Decode(
    'CgpFZGdlU2lnbmFsEikKEHByb3RvY29sX3ZlcnNpb24YASABKA1SD3Byb3RvY29sVmVyc2lvbh'
    'IdCgptZXNzYWdlX2lkGAIgASgJUgltZXNzYWdlSWQSGwoJc2VuZGVyX2lkGAMgASgJUghzZW5k'
    'ZXJJZBIXCgdib290X2lkGAQgASgJUgZib290SWQSIwoNY29ubmVjdGlvbl9pZBgFIAEoCVIMY2'
    '9ubmVjdGlvbklkEh0KCnN0cmVhbV9zZXEYBiABKARSCXN0cmVhbVNlcRIzCgdzZW50X2F0GAcg'
    'ASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIGc2VudEF0EjQKBXJlYWR5GBQgASgLMh'
    'wuYW55dHR5LmNsb3VkLnYxLkNsaWVudFJlYWR5SABSBXJlYWR5EjUKBmFuc3dlchgVIAEoCzIb'
    'LmFueXR0eS5jbG91ZC52MS5FZGdlQW5zd2VySABSBmFuc3dlchI9CghyZWplY3RlZBgWIAEoCz'
    'IfLmFueXR0eS5jbG91ZC52MS5TaWduYWxSZWplY3RlZEgAUghyZWplY3RlZBI+CgljaGFsbGVu'
    'Z2UYFyABKAsyHi5hbnl0dHkuY2xvdWQudjEuRWRnZUNoYWxsZW5nZUgAUgljaGFsbGVuZ2USPg'
    'oGY2xvc2VkGBggASgLMiQuYW55dHR5LmNsb3VkLnYxLlNpZ25hbFNlc3Npb25DbG9zZWRIAFIG'
    'Y2xvc2VkEj0KCHByZXNlbmNlGBkgASgLMh8uYW55dHR5LmNsb3VkLnYxLkRhZW1vblByZXNlbm'
    'NlSABSCHByZXNlbmNlElIKEXBhdGhfZGVjaXNpb25fYWNrGBogASgLMiQuYW55dHR5LmNsb3Vk'
    'LnYxLkVkZ2VQYXRoRGVjaXNpb25BY2tIAFIPcGF0aERlY2lzaW9uQWNrElgKE3Nlc3Npb25fcm'
    'VsZWFzZV9hY2sYGyABKAsyJi5hbnl0dHkuY2xvdWQudjEuRWRnZVNlc3Npb25SZWxlYXNlQWNr'
    'SABSEXNlc3Npb25SZWxlYXNlQWNrQgkKB3BheWxvYWQ=');

const $core.Map<$core.String, $core.dynamic> ClientGatewayServiceBase$json = {
  '1': 'ClientGateway',
  '2': [
    {
      '1': 'Connect',
      '2': '.anytty.cloud.v1.ClientSignal',
      '3': '.anytty.cloud.v1.EdgeSignal',
      '5': true,
      '6': true
    },
  ],
};

@$core.Deprecated('Use clientGatewayServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    ClientGatewayServiceBase$messageJson = {
  '.anytty.cloud.v1.ClientSignal': ClientSignal$json,
  '.google.protobuf.Timestamp': $2.Timestamp$json,
  '.anytty.cloud.v1.ClientHello': ClientHello$json,
  '.anytty.cloud.v1.SignedEnvelope': $0.SignedEnvelope$json,
  '.anytty.cloud.v1.PairingAdmission': PairingAdmission$json,
  '.anytty.cloud.v1.ClientOffer': ClientOffer$json,
  '.anytty.cloud.v1.CloudICECandidate': CloudICECandidate$json,
  '.anytty.cloud.v1.ClientPathDecision': ClientPathDecision$json,
  '.anytty.cloud.v1.ClientSessionRelease': ClientSessionRelease$json,
  '.anytty.cloud.v1.EdgeSignal': EdgeSignal$json,
  '.anytty.cloud.v1.ClientReady': ClientReady$json,
  '.anytty.cloud.v1.RelayICEConfig': $1.RelayICEConfig$json,
  '.anytty.cloud.v1.CloudEntitlementFailure': $0.CloudEntitlementFailure$json,
  '.anytty.cloud.v1.EdgeAnswer': EdgeAnswer$json,
  '.anytty.cloud.v1.SignalRejected': SignalRejected$json,
  '.anytty.cloud.v1.EdgeChallenge': $0.EdgeChallenge$json,
  '.anytty.cloud.v1.SignalSessionClosed': SignalSessionClosed$json,
  '.anytty.cloud.v1.DaemonPresence': DaemonPresence$json,
  '.anytty.cloud.v1.EdgePathDecisionAck': EdgePathDecisionAck$json,
  '.anytty.cloud.v1.EdgeSessionReleaseAck': EdgeSessionReleaseAck$json,
};

/// Descriptor for `ClientGateway`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List clientGatewayServiceDescriptor = $convert.base64Decode(
    'Cg1DbGllbnRHYXRld2F5EkkKB0Nvbm5lY3QSHS5hbnl0dHkuY2xvdWQudjEuQ2xpZW50U2lnbm'
    'FsGhsuYW55dHR5LmNsb3VkLnYxLkVkZ2VTaWduYWwoATAB');
