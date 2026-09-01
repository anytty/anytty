// This is a generated file - do not edit.
//
// Generated from cloud/v1/common.proto.

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

@$core.Deprecated('Use cloudEntitlementErrorCodeDescriptor instead')
const CloudEntitlementErrorCode$json = {
  '1': 'CloudEntitlementErrorCode',
  '2': [
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_UNSPECIFIED', '2': 0},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED', '2': 1},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_NOT_IN_PLAN', '2': 2},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_QUOTA_EXHAUSTED', '2': 3},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED', '2': 4},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE', '2': 5},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_RELAY_REGION_UNAVAILABLE', '2': 6},
    {'1': 'CLOUD_ENTITLEMENT_ERROR_CODE_SERVICE_UNAVAILABLE', '2': 7},
  ],
};

/// Descriptor for `CloudEntitlementErrorCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List cloudEntitlementErrorCodeDescriptor = $convert.base64Decode(
    'ChlDbG91ZEVudGl0bGVtZW50RXJyb3JDb2RlEiwKKENMT1VEX0VOVElUTEVNRU5UX0VSUk9SX0'
    'NPREVfVU5TUEVDSUZJRUQQABI3CjNDTE9VRF9FTlRJVExFTUVOVF9FUlJPUl9DT0RFX0RBRU1P'
    'Tl9MSU1JVF9FWEhBVVNURUQQARIyCi5DTE9VRF9FTlRJVExFTUVOVF9FUlJPUl9DT0RFX1JFTE'
    'FZX05PVF9JTl9QTEFOEAISNgoyQ0xPVURfRU5USVRMRU1FTlRfRVJST1JfQ09ERV9SRUxBWV9R'
    'VU9UQV9FWEhBVVNURUQQAxI8CjhDTE9VRF9FTlRJVExFTUVOVF9FUlJPUl9DT0RFX1JFTEFZX0'
    'NPTkNVUlJFTkNZX0VYSEFVU1RFRBAEEjYKMkNMT1VEX0VOVElUTEVNRU5UX0VSUk9SX0NPREVf'
    'U1VCU0NSSVBUSU9OX0lOQUNUSVZFEAUSOQo1Q0xPVURfRU5USVRMRU1FTlRfRVJST1JfQ09ERV'
    '9SRUxBWV9SRUdJT05fVU5BVkFJTEFCTEUQBhI0CjBDTE9VRF9FTlRJVExFTUVOVF9FUlJPUl9D'
    'T0RFX1NFUlZJQ0VfVU5BVkFJTEFCTEUQBw==');

@$core.Deprecated('Use edgeChallengeTargetDescriptor instead')
const EdgeChallengeTarget$json = {
  '1': 'EdgeChallengeTarget',
  '2': [
    {'1': 'EDGE_CHALLENGE_TARGET_UNSPECIFIED', '2': 0},
    {'1': 'EDGE_CHALLENGE_TARGET_AGENT_GATEWAY', '2': 1},
    {'1': 'EDGE_CHALLENGE_TARGET_CLIENT_GATEWAY', '2': 2},
  ],
};

/// Descriptor for `EdgeChallengeTarget`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List edgeChallengeTargetDescriptor = $convert.base64Decode(
    'ChNFZGdlQ2hhbGxlbmdlVGFyZ2V0EiUKIUVER0VfQ0hBTExFTkdFX1RBUkdFVF9VTlNQRUNJRk'
    'lFRBAAEicKI0VER0VfQ0hBTExFTkdFX1RBUkdFVF9BR0VOVF9HQVRFV0FZEAESKAokRURHRV9D'
    'SEFMTEVOR0VfVEFSR0VUX0NMSUVOVF9HQVRFV0FZEAI=');

@$core.Deprecated('Use verificationKeyDescriptor instead')
const VerificationKey$json = {
  '1': 'VerificationKey',
  '2': [
    {'1': 'key_id', '3': 1, '4': 1, '5': 9, '10': 'keyId'},
    {'1': 'algorithm', '3': 2, '4': 1, '5': 9, '10': 'algorithm'},
    {'1': 'public_key', '3': 3, '4': 1, '5': 12, '10': 'publicKey'},
  ],
};

/// Descriptor for `VerificationKey`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List verificationKeyDescriptor = $convert.base64Decode(
    'Cg9WZXJpZmljYXRpb25LZXkSFQoGa2V5X2lkGAEgASgJUgVrZXlJZBIcCglhbGdvcml0aG0YAi'
    'ABKAlSCWFsZ29yaXRobRIdCgpwdWJsaWNfa2V5GAMgASgMUglwdWJsaWNLZXk=');

@$core.Deprecated('Use keyBundleDescriptor instead')
const KeyBundle$json = {
  '1': 'KeyBundle',
  '2': [
    {'1': 'revision', '3': 1, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'issued_at',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'issuedAt'
    },
    {
      '1': 'expires_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
    {
      '1': 'keys',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.VerificationKey',
      '10': 'keys'
    },
  ],
};

/// Descriptor for `KeyBundle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List keyBundleDescriptor = $convert.base64Decode(
    'CglLZXlCdW5kbGUSGgoIcmV2aXNpb24YASABKARSCHJldmlzaW9uEjcKCWlzc3VlZF9hdBgCIA'
    'EoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCGlzc3VlZEF0EjkKCmV4cGlyZXNfYXQY'
    'AyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUglleHBpcmVzQXQSNAoEa2V5cxgEIA'
    'MoCzIgLmFueXR0eS5jbG91ZC52MS5WZXJpZmljYXRpb25LZXlSBGtleXM=');

@$core.Deprecated('Use heartbeatPolicyDescriptor instead')
const HeartbeatPolicy$json = {
  '1': 'HeartbeatPolicy',
  '2': [
    {
      '1': 'interval',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Duration',
      '10': 'interval'
    },
    {
      '1': 'timeout',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Duration',
      '10': 'timeout'
    },
  ],
};

/// Descriptor for `HeartbeatPolicy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List heartbeatPolicyDescriptor = $convert.base64Decode(
    'Cg9IZWFydGJlYXRQb2xpY3kSNQoIaW50ZXJ2YWwYASABKAsyGS5nb29nbGUucHJvdG9idWYuRH'
    'VyYXRpb25SCGludGVydmFsEjMKB3RpbWVvdXQYAiABKAsyGS5nb29nbGUucHJvdG9idWYuRHVy'
    'YXRpb25SB3RpbWVvdXQ=');

@$core.Deprecated('Use signedEnvelopeDescriptor instead')
const SignedEnvelope$json = {
  '1': 'SignedEnvelope',
  '2': [
    {'1': 'key_id', '3': 1, '4': 1, '5': 9, '10': 'keyId'},
    {'1': 'payload', '3': 2, '4': 1, '5': 12, '10': 'payload'},
    {'1': 'signature', '3': 3, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `SignedEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signedEnvelopeDescriptor = $convert.base64Decode(
    'Cg5TaWduZWRFbnZlbG9wZRIVCgZrZXlfaWQYASABKAlSBWtleUlkEhgKB3BheWxvYWQYAiABKA'
    'xSB3BheWxvYWQSHAoJc2lnbmF0dXJlGAMgASgMUglzaWduYXR1cmU=');

@$core.Deprecated('Use cloudEntitlementFailureDescriptor instead')
const CloudEntitlementFailure$json = {
  '1': 'CloudEntitlementFailure',
  '2': [
    {
      '1': 'code',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudEntitlementErrorCode',
      '10': 'code'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'limit', '3': 3, '4': 1, '5': 4, '10': 'limit'},
    {'1': 'used', '3': 4, '4': 1, '5': 4, '10': 'used'},
    {'1': 'remaining_bytes', '3': 5, '4': 1, '5': 4, '10': 'remainingBytes'},
    {
      '1': 'period_end',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodEnd'
    },
  ],
};

/// Descriptor for `CloudEntitlementFailure`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudEntitlementFailureDescriptor = $convert.base64Decode(
    'ChdDbG91ZEVudGl0bGVtZW50RmFpbHVyZRI+CgRjb2RlGAEgASgOMiouYW55dHR5LmNsb3VkLn'
    'YxLkNsb3VkRW50aXRsZW1lbnRFcnJvckNvZGVSBGNvZGUSGAoHbWVzc2FnZRgCIAEoCVIHbWVz'
    'c2FnZRIUCgVsaW1pdBgDIAEoBFIFbGltaXQSEgoEdXNlZBgEIAEoBFIEdXNlZBInCg9yZW1haW'
    '5pbmdfYnl0ZXMYBSABKARSDnJlbWFpbmluZ0J5dGVzEjkKCnBlcmlvZF9lbmQYBiABKAsyGi5n'
    'b29nbGUucHJvdG9idWYuVGltZXN0YW1wUglwZXJpb2RFbmQ=');

@$core.Deprecated('Use edgeChallengeDescriptor instead')
const EdgeChallenge$json = {
  '1': 'EdgeChallenge',
  '2': [
    {'1': 'nonce', '3': 1, '4': 1, '5': 12, '10': 'nonce'},
    {'1': 'edge_id', '3': 2, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'edge_boot_id', '3': 3, '4': 1, '5': 9, '10': 'edgeBootId'},
    {'1': 'stream_id', '3': 4, '4': 1, '5': 9, '10': 'streamId'},
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
    {
      '1': 'target',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.EdgeChallengeTarget',
      '10': 'target'
    },
  ],
};

/// Descriptor for `EdgeChallenge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeChallengeDescriptor = $convert.base64Decode(
    'Cg1FZGdlQ2hhbGxlbmdlEhQKBW5vbmNlGAEgASgMUgVub25jZRIXCgdlZGdlX2lkGAIgASgJUg'
    'ZlZGdlSWQSIAoMZWRnZV9ib290X2lkGAMgASgJUgplZGdlQm9vdElkEhsKCXN0cmVhbV9pZBgE'
    'IAEoCVIIc3RyZWFtSWQSNwoJaXNzdWVkX2F0GAUgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbW'
    'VzdGFtcFIIaXNzdWVkQXQSOQoKZXhwaXJlc19hdBgGIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5U'
    'aW1lc3RhbXBSCWV4cGlyZXNBdBI8CgZ0YXJnZXQYByABKA4yJC5hbnl0dHkuY2xvdWQudjEuRW'
    'RnZUNoYWxsZW5nZVRhcmdldFIGdGFyZ2V0');
