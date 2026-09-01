// This is a generated file - do not edit.
//
// Generated from cloud/v1/agent_gateway.proto.

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
    as $6;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pbjson.dart'
    as $4;

import 'client_gateway.pbjson.dart' as $1;
import 'common.pbjson.dart' as $0;
import 'enrollment.pbjson.dart' as $3;
import 'usage.pbjson.dart' as $2;

@$core.Deprecated('Use agentHelloDescriptor instead')
const AgentHello$json = {
  '1': 'AgentHello',
  '2': [
    {
      '1': 'daemon_binding',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEnvelope',
      '10': 'daemonBinding'
    },
    {'1': 'device_proof', '3': 2, '4': 1, '5': 12, '10': 'deviceProof'},
    {'1': 'software_version', '3': 3, '4': 1, '5': 9, '10': 'softwareVersion'},
    {
      '1': 'attempt_generation',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'attemptGeneration'
    },
  ],
};

/// Descriptor for `AgentHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentHelloDescriptor = $convert.base64Decode(
    'CgpBZ2VudEhlbGxvEkYKDmRhZW1vbl9iaW5kaW5nGAEgASgLMh8uYW55dHR5LmNsb3VkLnYxLl'
    'NpZ25lZEVudmVsb3BlUg1kYWVtb25CaW5kaW5nEiEKDGRldmljZV9wcm9vZhgCIAEoDFILZGV2'
    'aWNlUHJvb2YSKQoQc29mdHdhcmVfdmVyc2lvbhgDIAEoCVIPc29mdHdhcmVWZXJzaW9uEi0KEm'
    'F0dGVtcHRfZ2VuZXJhdGlvbhgEIAEoBFIRYXR0ZW1wdEdlbmVyYXRpb24=');

@$core.Deprecated('Use agentHeartbeatDescriptor instead')
const AgentHeartbeat$json = {
  '1': 'AgentHeartbeat',
  '2': [
    {'1': 'generation', '3': 1, '4': 1, '5': 4, '10': 'generation'},
  ],
};

/// Descriptor for `AgentHeartbeat`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentHeartbeatDescriptor = $convert.base64Decode(
    'Cg5BZ2VudEhlYXJ0YmVhdBIeCgpnZW5lcmF0aW9uGAEgASgEUgpnZW5lcmF0aW9u');

@$core.Deprecated('Use agentOfferDescriptor instead')
const AgentOffer$json = {
  '1': 'AgentOffer',
  '2': [
    {'1': 'correlation_id', '3': 1, '4': 1, '5': 9, '10': 'correlationId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'agent_generation', '3': 3, '4': 1, '5': 4, '10': 'agentGeneration'},
    {
      '1': 'client_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {'1': 'offer_sdp', '3': 5, '4': 1, '5': 9, '10': 'offerSdp'},
    {
      '1': 'candidates',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudICECandidate',
      '10': 'candidates'
    },
    {
      '1': 'relay',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.RelayICEConfig',
      '10': 'relay'
    },
    {
      '1': 'access_mode',
      '3': 8,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudClientAccessMode',
      '10': 'accessMode'
    },
    {
      '1': 'pairing_claim_sha256',
      '3': 9,
      '4': 1,
      '5': 12,
      '10': 'pairingClaimSha256'
    },
  ],
};

/// Descriptor for `AgentOffer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentOfferDescriptor = $convert.base64Decode(
    'CgpBZ2VudE9mZmVyEiUKDmNvcnJlbGF0aW9uX2lkGAEgASgJUg1jb3JyZWxhdGlvbklkEh0KCn'
    'Nlc3Npb25faWQYAiABKAlSCXNlc3Npb25JZBIpChBhZ2VudF9nZW5lcmF0aW9uGAMgASgEUg9h'
    'Z2VudEdlbmVyYXRpb24SKgoRY2xpZW50X3B1YmxpY19rZXkYBCABKAxSD2NsaWVudFB1YmxpY0'
    'tleRIbCglvZmZlcl9zZHAYBSABKAlSCG9mZmVyU2RwEkIKCmNhbmRpZGF0ZXMYBiADKAsyIi5h'
    'bnl0dHkuY2xvdWQudjEuQ2xvdWRJQ0VDYW5kaWRhdGVSCmNhbmRpZGF0ZXMSNQoFcmVsYXkYBy'
    'ABKAsyHy5hbnl0dHkuY2xvdWQudjEuUmVsYXlJQ0VDb25maWdSBXJlbGF5EkcKC2FjY2Vzc19t'
    'b2RlGAggASgOMiYuYW55dHR5LmNsb3VkLnYxLkNsb3VkQ2xpZW50QWNjZXNzTW9kZVIKYWNjZX'
    'NzTW9kZRIwChRwYWlyaW5nX2NsYWltX3NoYTI1NhgJIAEoDFIScGFpcmluZ0NsYWltU2hhMjU2');

@$core.Deprecated('Use agentAuthorizeDescriptor instead')
const AgentAuthorize$json = {
  '1': 'AgentAuthorize',
  '2': [
    {'1': 'correlation_id', '3': 1, '4': 1, '5': 9, '10': 'correlationId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'agent_generation', '3': 3, '4': 1, '5': 4, '10': 'agentGeneration'},
    {
      '1': 'client_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {
      '1': 'product',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.ClientProduct',
      '10': 'product'
    },
    {
      '1': 'access_mode',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.CloudClientAccessMode',
      '10': 'accessMode'
    },
    {
      '1': 'pairing_claim_sha256',
      '3': 7,
      '4': 1,
      '5': 12,
      '10': 'pairingClaimSha256'
    },
  ],
};

/// Descriptor for `AgentAuthorize`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentAuthorizeDescriptor = $convert.base64Decode(
    'Cg5BZ2VudEF1dGhvcml6ZRIlCg5jb3JyZWxhdGlvbl9pZBgBIAEoCVINY29ycmVsYXRpb25JZB'
    'IdCgpzZXNzaW9uX2lkGAIgASgJUglzZXNzaW9uSWQSKQoQYWdlbnRfZ2VuZXJhdGlvbhgDIAEo'
    'BFIPYWdlbnRHZW5lcmF0aW9uEioKEWNsaWVudF9wdWJsaWNfa2V5GAQgASgMUg9jbGllbnRQdW'
    'JsaWNLZXkSOAoHcHJvZHVjdBgFIAEoDjIeLmFueXR0eS5jbG91ZC52MS5DbGllbnRQcm9kdWN0'
    'Ugdwcm9kdWN0EkcKC2FjY2Vzc19tb2RlGAYgASgOMiYuYW55dHR5LmNsb3VkLnYxLkNsb3VkQ2'
    'xpZW50QWNjZXNzTW9kZVIKYWNjZXNzTW9kZRIwChRwYWlyaW5nX2NsYWltX3NoYTI1NhgHIAEo'
    'DFIScGFpcmluZ0NsYWltU2hhMjU2');

@$core.Deprecated('Use agentAuthorizationResultDescriptor instead')
const AgentAuthorizationResult$json = {
  '1': 'AgentAuthorizationResult',
  '2': [
    {'1': 'correlation_id', '3': 1, '4': 1, '5': 9, '10': 'correlationId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'authorized', '3': 3, '4': 1, '5': 8, '10': 'authorized'},
    {'1': 'code', '3': 4, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 5, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `AgentAuthorizationResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentAuthorizationResultDescriptor = $convert.base64Decode(
    'ChhBZ2VudEF1dGhvcml6YXRpb25SZXN1bHQSJQoOY29ycmVsYXRpb25faWQYASABKAlSDWNvcn'
    'JlbGF0aW9uSWQSHQoKc2Vzc2lvbl9pZBgCIAEoCVIJc2Vzc2lvbklkEh4KCmF1dGhvcml6ZWQY'
    'AyABKAhSCmF1dGhvcml6ZWQSEgoEY29kZRgEIAEoCVIEY29kZRIYCgdtZXNzYWdlGAUgASgJUg'
    'dtZXNzYWdl');

@$core.Deprecated('Use agentAnswerDescriptor instead')
const AgentAnswer$json = {
  '1': 'AgentAnswer',
  '2': [
    {'1': 'correlation_id', '3': 1, '4': 1, '5': 9, '10': 'correlationId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'answer_sdp', '3': 3, '4': 1, '5': 9, '10': 'answerSdp'},
    {
      '1': 'candidates',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudICECandidate',
      '10': 'candidates'
    },
  ],
};

/// Descriptor for `AgentAnswer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentAnswerDescriptor = $convert.base64Decode(
    'CgtBZ2VudEFuc3dlchIlCg5jb3JyZWxhdGlvbl9pZBgBIAEoCVINY29ycmVsYXRpb25JZBIdCg'
    'pzZXNzaW9uX2lkGAIgASgJUglzZXNzaW9uSWQSHQoKYW5zd2VyX3NkcBgDIAEoCVIJYW5zd2Vy'
    'U2RwEkIKCmNhbmRpZGF0ZXMYBCADKAsyIi5hbnl0dHkuY2xvdWQudjEuQ2xvdWRJQ0VDYW5kaW'
    'RhdGVSCmNhbmRpZGF0ZXM=');

@$core.Deprecated('Use agentSignalRejectedDescriptor instead')
const AgentSignalRejected$json = {
  '1': 'AgentSignalRejected',
  '2': [
    {'1': 'correlation_id', '3': 1, '4': 1, '5': 9, '10': 'correlationId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'code', '3': 3, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 4, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `AgentSignalRejected`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentSignalRejectedDescriptor = $convert.base64Decode(
    'ChNBZ2VudFNpZ25hbFJlamVjdGVkEiUKDmNvcnJlbGF0aW9uX2lkGAEgASgJUg1jb3JyZWxhdG'
    'lvbklkEh0KCnNlc3Npb25faWQYAiABKAlSCXNlc3Npb25JZBISCgRjb2RlGAMgASgJUgRjb2Rl'
    'EhgKB21lc3NhZ2UYBCABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use agentReadyDescriptor instead')
const AgentReady$json = {
  '1': 'AgentReady',
  '2': [
    {'1': 'generation', '3': 1, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'heartbeat',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.HeartbeatPolicy',
      '10': 'heartbeat'
    },
    {
      '1': 'daemon_state',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemonState'
    },
  ],
};

/// Descriptor for `AgentReady`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentReadyDescriptor = $convert.base64Decode(
    'CgpBZ2VudFJlYWR5Eh4KCmdlbmVyYXRpb24YASABKARSCmdlbmVyYXRpb24SPgoJaGVhcnRiZW'
    'F0GAIgASgLMiAuYW55dHR5LmNsb3VkLnYxLkhlYXJ0YmVhdFBvbGljeVIJaGVhcnRiZWF0EkUK'
    'DGRhZW1vbl9zdGF0ZRgDIAEoCzIiLmFueXR0eS5jbG91ZC52MS5EYWVtb25TdGF0ZVJlY29yZF'
    'ILZGFlbW9uU3RhdGU=');

@$core.Deprecated('Use daemonLifecycleCommandDescriptor instead')
const DaemonLifecycleCommand$json = {
  '1': 'DaemonLifecycleCommand',
  '2': [
    {
      '1': 'daemon_state',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemonState'
    },
    {'1': 'agent_generation', '3': 2, '4': 1, '5': 4, '10': 'agentGeneration'},
  ],
};

/// Descriptor for `DaemonLifecycleCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonLifecycleCommandDescriptor = $convert.base64Decode(
    'ChZEYWVtb25MaWZlY3ljbGVDb21tYW5kEkUKDGRhZW1vbl9zdGF0ZRgBIAEoCzIiLmFueXR0eS'
    '5jbG91ZC52MS5EYWVtb25TdGF0ZVJlY29yZFILZGFlbW9uU3RhdGUSKQoQYWdlbnRfZ2VuZXJh'
    'dGlvbhgCIAEoBFIPYWdlbnRHZW5lcmF0aW9u');

@$core.Deprecated('Use daemonLifecycleResultDescriptor instead')
const DaemonLifecycleResult$json = {
  '1': 'DaemonLifecycleResult',
  '2': [
    {
      '1': 'daemon_state',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonStateRecord',
      '10': 'daemonState'
    },
    {'1': 'agent_generation', '3': 2, '4': 1, '5': 4, '10': 'agentGeneration'},
    {'1': 'applied', '3': 3, '4': 1, '5': 8, '10': 'applied'},
    {'1': 'error_message', '3': 4, '4': 1, '5': 9, '10': 'errorMessage'},
  ],
};

/// Descriptor for `DaemonLifecycleResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonLifecycleResultDescriptor = $convert.base64Decode(
    'ChVEYWVtb25MaWZlY3ljbGVSZXN1bHQSRQoMZGFlbW9uX3N0YXRlGAEgASgLMiIuYW55dHR5Lm'
    'Nsb3VkLnYxLkRhZW1vblN0YXRlUmVjb3JkUgtkYWVtb25TdGF0ZRIpChBhZ2VudF9nZW5lcmF0'
    'aW9uGAIgASgEUg9hZ2VudEdlbmVyYXRpb24SGAoHYXBwbGllZBgDIAEoCFIHYXBwbGllZBIjCg'
    '1lcnJvcl9tZXNzYWdlGAQgASgJUgxlcnJvck1lc3NhZ2U=');

@$core.Deprecated('Use daemonEdgeReselectCommandDescriptor instead')
const DaemonEdgeReselectCommand$json = {
  '1': 'DaemonEdgeReselectCommand',
  '2': [
    {'1': 'agent_generation', '3': 1, '4': 1, '5': 4, '10': 'agentGeneration'},
    {
      '1': 'preference_revision',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'preferenceRevision'
    },
  ],
};

/// Descriptor for `DaemonEdgeReselectCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List daemonEdgeReselectCommandDescriptor = $convert.base64Decode(
    'ChlEYWVtb25FZGdlUmVzZWxlY3RDb21tYW5kEikKEGFnZW50X2dlbmVyYXRpb24YASABKARSD2'
    'FnZW50R2VuZXJhdGlvbhIvChNwcmVmZXJlbmNlX3JldmlzaW9uGAIgASgEUhJwcmVmZXJlbmNl'
    'UmV2aXNpb24=');

@$core.Deprecated('Use agentEventDescriptor instead')
const AgentEvent$json = {
  '1': 'AgentEvent',
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
      '6': '.anytty.cloud.v1.AgentHello',
      '9': 0,
      '10': 'hello'
    },
    {
      '1': 'heartbeat',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentHeartbeat',
      '9': 0,
      '10': 'heartbeat'
    },
    {
      '1': 'answer',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentAnswer',
      '9': 0,
      '10': 'answer'
    },
    {
      '1': 'rejected',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentSignalRejected',
      '9': 0,
      '10': 'rejected'
    },
    {
      '1': 'authorization',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentAuthorizationResult',
      '9': 0,
      '10': 'authorization'
    },
    {
      '1': 'lifecycle_result',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonLifecycleResult',
      '9': 0,
      '10': 'lifecycleResult'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `AgentEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List agentEventDescriptor = $convert.base64Decode(
    'CgpBZ2VudEV2ZW50EikKEHByb3RvY29sX3ZlcnNpb24YASABKA1SD3Byb3RvY29sVmVyc2lvbh'
    'IdCgptZXNzYWdlX2lkGAIgASgJUgltZXNzYWdlSWQSGwoJc2VuZGVyX2lkGAMgASgJUghzZW5k'
    'ZXJJZBIXCgdib290X2lkGAQgASgJUgZib290SWQSIwoNY29ubmVjdGlvbl9pZBgFIAEoCVIMY2'
    '9ubmVjdGlvbklkEh0KCnN0cmVhbV9zZXEYBiABKARSCXN0cmVhbVNlcRIzCgdzZW50X2F0GAcg'
    'ASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIGc2VudEF0EjMKBWhlbGxvGBQgASgLMh'
    'suYW55dHR5LmNsb3VkLnYxLkFnZW50SGVsbG9IAFIFaGVsbG8SPwoJaGVhcnRiZWF0GBUgASgL'
    'Mh8uYW55dHR5LmNsb3VkLnYxLkFnZW50SGVhcnRiZWF0SABSCWhlYXJ0YmVhdBI2CgZhbnN3ZX'
    'IYFiABKAsyHC5hbnl0dHkuY2xvdWQudjEuQWdlbnRBbnN3ZXJIAFIGYW5zd2VyEkIKCHJlamVj'
    'dGVkGBcgASgLMiQuYW55dHR5LmNsb3VkLnYxLkFnZW50U2lnbmFsUmVqZWN0ZWRIAFIIcmVqZW'
    'N0ZWQSUQoNYXV0aG9yaXphdGlvbhgYIAEoCzIpLmFueXR0eS5jbG91ZC52MS5BZ2VudEF1dGhv'
    'cml6YXRpb25SZXN1bHRIAFINYXV0aG9yaXphdGlvbhJTChBsaWZlY3ljbGVfcmVzdWx0GBkgAS'
    'gLMiYuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkxpZmVjeWNsZVJlc3VsdEgAUg9saWZlY3ljbGVS'
    'ZXN1bHRCCQoHcGF5bG9hZA==');

@$core.Deprecated('Use edgeCommandDescriptor instead')
const EdgeCommand$json = {
  '1': 'EdgeCommand',
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
      '6': '.anytty.cloud.v1.AgentReady',
      '9': 0,
      '10': 'ready'
    },
    {
      '1': 'offer',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentOffer',
      '9': 0,
      '10': 'offer'
    },
    {
      '1': 'authorize',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AgentAuthorize',
      '9': 0,
      '10': 'authorize'
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
      '1': 'lifecycle',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonLifecycleCommand',
      '9': 0,
      '10': 'lifecycle'
    },
    {
      '1': 'edge_reselect',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.DaemonEdgeReselectCommand',
      '9': 0,
      '10': 'edgeReselect'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `EdgeCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeCommandDescriptor = $convert.base64Decode(
    'CgtFZGdlQ29tbWFuZBIpChBwcm90b2NvbF92ZXJzaW9uGAEgASgNUg9wcm90b2NvbFZlcnNpb2'
    '4SHQoKbWVzc2FnZV9pZBgCIAEoCVIJbWVzc2FnZUlkEhsKCXNlbmRlcl9pZBgDIAEoCVIIc2Vu'
    'ZGVySWQSFwoHYm9vdF9pZBgEIAEoCVIGYm9vdElkEiMKDWNvbm5lY3Rpb25faWQYBSABKAlSDG'
    'Nvbm5lY3Rpb25JZBIdCgpzdHJlYW1fc2VxGAYgASgEUglzdHJlYW1TZXESMwoHc2VudF9hdBgH'
    'IAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSBnNlbnRBdBIzCgVyZWFkeRgUIAEoCz'
    'IbLmFueXR0eS5jbG91ZC52MS5BZ2VudFJlYWR5SABSBXJlYWR5EjMKBW9mZmVyGBUgASgLMhsu'
    'YW55dHR5LmNsb3VkLnYxLkFnZW50T2ZmZXJIAFIFb2ZmZXISPwoJYXV0aG9yaXplGBYgASgLMh'
    '8uYW55dHR5LmNsb3VkLnYxLkFnZW50QXV0aG9yaXplSABSCWF1dGhvcml6ZRI+CgljaGFsbGVu'
    'Z2UYFyABKAsyHi5hbnl0dHkuY2xvdWQudjEuRWRnZUNoYWxsZW5nZUgAUgljaGFsbGVuZ2USRw'
    'oJbGlmZWN5Y2xlGBggASgLMicuYW55dHR5LmNsb3VkLnYxLkRhZW1vbkxpZmVjeWNsZUNvbW1h'
    'bmRIAFIJbGlmZWN5Y2xlElEKDWVkZ2VfcmVzZWxlY3QYGSABKAsyKi5hbnl0dHkuY2xvdWQudj'
    'EuRGFlbW9uRWRnZVJlc2VsZWN0Q29tbWFuZEgAUgxlZGdlUmVzZWxlY3RCCQoHcGF5bG9hZA==');

const $core.Map<$core.String, $core.dynamic> AgentGatewayServiceBase$json = {
  '1': 'AgentGateway',
  '2': [
    {
      '1': 'Connect',
      '2': '.anytty.cloud.v1.AgentEvent',
      '3': '.anytty.cloud.v1.EdgeCommand',
      '5': true,
      '6': true
    },
  ],
};

@$core.Deprecated('Use agentGatewayServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    AgentGatewayServiceBase$messageJson = {
  '.anytty.cloud.v1.AgentEvent': AgentEvent$json,
  '.google.protobuf.Timestamp': $4.Timestamp$json,
  '.anytty.cloud.v1.AgentHello': AgentHello$json,
  '.anytty.cloud.v1.SignedEnvelope': $0.SignedEnvelope$json,
  '.anytty.cloud.v1.AgentHeartbeat': AgentHeartbeat$json,
  '.anytty.cloud.v1.AgentAnswer': AgentAnswer$json,
  '.anytty.cloud.v1.CloudICECandidate': $1.CloudICECandidate$json,
  '.anytty.cloud.v1.AgentSignalRejected': AgentSignalRejected$json,
  '.anytty.cloud.v1.AgentAuthorizationResult': AgentAuthorizationResult$json,
  '.anytty.cloud.v1.DaemonLifecycleResult': DaemonLifecycleResult$json,
  '.anytty.cloud.v1.DaemonStateRecord': $3.DaemonStateRecord$json,
  '.anytty.cloud.v1.EdgeCommand': EdgeCommand$json,
  '.anytty.cloud.v1.AgentReady': AgentReady$json,
  '.anytty.cloud.v1.HeartbeatPolicy': $0.HeartbeatPolicy$json,
  '.google.protobuf.Duration': $6.Duration$json,
  '.anytty.cloud.v1.AgentOffer': AgentOffer$json,
  '.anytty.cloud.v1.RelayICEConfig': $2.RelayICEConfig$json,
  '.anytty.cloud.v1.AgentAuthorize': AgentAuthorize$json,
  '.anytty.cloud.v1.EdgeChallenge': $0.EdgeChallenge$json,
  '.anytty.cloud.v1.DaemonLifecycleCommand': DaemonLifecycleCommand$json,
  '.anytty.cloud.v1.DaemonEdgeReselectCommand': DaemonEdgeReselectCommand$json,
};

/// Descriptor for `AgentGateway`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List agentGatewayServiceDescriptor = $convert.base64Decode(
    'CgxBZ2VudEdhdGV3YXkSSAoHQ29ubmVjdBIbLmFueXR0eS5jbG91ZC52MS5BZ2VudEV2ZW50Gh'
    'wuYW55dHR5LmNsb3VkLnYxLkVkZ2VDb21tYW5kKAEwAQ==');
