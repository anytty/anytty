// This is a generated file - do not edit.
//
// Generated from remoteauthpb/remote_auth.proto.

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

@$core.Deprecated('Use authErrorCodeDescriptor instead')
const AuthErrorCode$json = {
  '1': 'AuthErrorCode',
  '2': [
    {'1': 'AUTH_ERROR_CODE_UNSPECIFIED', '2': 0},
    {'1': 'AUTH_ERROR_CODE_PROTOCOL', '2': 1},
    {'1': 'AUTH_ERROR_CODE_DEVICE_IDENTITY_MISMATCH', '2': 2},
    {'1': 'AUTH_ERROR_CODE_CAPABILITY_INVALID', '2': 3},
    {'1': 'AUTH_ERROR_CODE_CAPABILITY_EXPIRED', '2': 4},
    {'1': 'AUTH_ERROR_CODE_CAPABILITY_REVOKED', '2': 5},
    {'1': 'AUTH_ERROR_CODE_CAPABILITY_PROOF_INVALID', '2': 6},
    {'1': 'AUTH_ERROR_CODE_SCOPE_INVALID', '2': 7},
    {'1': 'AUTH_ERROR_CODE_REPLAYED', '2': 8},
    {'1': 'AUTH_ERROR_CODE_INTERNAL', '2': 9},
    {'1': 'AUTH_ERROR_CODE_SUBJECT_KEY_MISMATCH', '2': 10},
    {'1': 'AUTH_ERROR_CODE_PAIRING_TICKET_INVALID', '2': 11},
    {'1': 'AUTH_ERROR_CODE_PAIRING_TICKET_EXPIRED', '2': 12},
    {'1': 'AUTH_ERROR_CODE_PAIRING_TICKET_CONSUMED', '2': 13},
  ],
};

/// Descriptor for `AuthErrorCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List authErrorCodeDescriptor = $convert.base64Decode(
    'Cg1BdXRoRXJyb3JDb2RlEh8KG0FVVEhfRVJST1JfQ09ERV9VTlNQRUNJRklFRBAAEhwKGEFVVE'
    'hfRVJST1JfQ09ERV9QUk9UT0NPTBABEiwKKEFVVEhfRVJST1JfQ09ERV9ERVZJQ0VfSURFTlRJ'
    'VFlfTUlTTUFUQ0gQAhImCiJBVVRIX0VSUk9SX0NPREVfQ0FQQUJJTElUWV9JTlZBTElEEAMSJg'
    'oiQVVUSF9FUlJPUl9DT0RFX0NBUEFCSUxJVFlfRVhQSVJFRBAEEiYKIkFVVEhfRVJST1JfQ09E'
    'RV9DQVBBQklMSVRZX1JFVk9LRUQQBRIsCihBVVRIX0VSUk9SX0NPREVfQ0FQQUJJTElUWV9QUk'
    '9PRl9JTlZBTElEEAYSIQodQVVUSF9FUlJPUl9DT0RFX1NDT1BFX0lOVkFMSUQQBxIcChhBVVRI'
    'X0VSUk9SX0NPREVfUkVQTEFZRUQQCBIcChhBVVRIX0VSUk9SX0NPREVfSU5URVJOQUwQCRIoCi'
    'RBVVRIX0VSUk9SX0NPREVfU1VCSkVDVF9LRVlfTUlTTUFUQ0gQChIqCiZBVVRIX0VSUk9SX0NP'
    'REVfUEFJUklOR19USUNLRVRfSU5WQUxJRBALEioKJkFVVEhfRVJST1JfQ09ERV9QQUlSSU5HX1'
    'RJQ0tFVF9FWFBJUkVEEAwSKwonQVVUSF9FUlJPUl9DT0RFX1BBSVJJTkdfVElDS0VUX0NPTlNV'
    'TUVEEA0=');

@$core.Deprecated('Use scopeKindDescriptor instead')
const ScopeKind$json = {
  '1': 'ScopeKind',
  '2': [
    {'1': 'SCOPE_KIND_UNSPECIFIED', '2': 0},
    {'1': 'SCOPE_KIND_DAEMON', '2': 1},
    {'1': 'SCOPE_KIND_TERMINAL', '2': 2},
    {'1': 'SCOPE_KIND_MACHINE_EVENTS', '2': 3},
  ],
};

/// Descriptor for `ScopeKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List scopeKindDescriptor = $convert.base64Decode(
    'CglTY29wZUtpbmQSGgoWU0NPUEVfS0lORF9VTlNQRUNJRklFRBAAEhUKEVNDT1BFX0tJTkRfRE'
    'FFTU9OEAESFwoTU0NPUEVfS0lORF9URVJNSU5BTBACEh0KGVNDT1BFX0tJTkRfTUFDSElORV9F'
    'VkVOVFMQAw==');

@$core.Deprecated('Use channelBindingKindDescriptor instead')
const ChannelBindingKind$json = {
  '1': 'ChannelBindingKind',
  '2': [
    {'1': 'CHANNEL_BINDING_KIND_UNSPECIFIED', '2': 0},
    {'1': 'CHANNEL_BINDING_KIND_DIRECT_TLS', '2': 1},
    {'1': 'CHANNEL_BINDING_KIND_DTLS', '2': 2},
    {'1': 'CHANNEL_BINDING_KIND_LOCAL_UNIX', '2': 3},
  ],
};

/// Descriptor for `ChannelBindingKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List channelBindingKindDescriptor = $convert.base64Decode(
    'ChJDaGFubmVsQmluZGluZ0tpbmQSJAogQ0hBTk5FTF9CSU5ESU5HX0tJTkRfVU5TUEVDSUZJRU'
    'QQABIjCh9DSEFOTkVMX0JJTkRJTkdfS0lORF9ESVJFQ1RfVExTEAESHQoZQ0hBTk5FTF9CSU5E'
    'SU5HX0tJTkRfRFRMUxACEiMKH0NIQU5ORUxfQklORElOR19LSU5EX0xPQ0FMX1VOSVgQAw==');

@$core.Deprecated('Use authOpenKindDescriptor instead')
const AuthOpenKind$json = {
  '1': 'AuthOpenKind',
  '2': [
    {'1': 'AUTH_OPEN_KIND_UNSPECIFIED', '2': 0},
    {'1': 'AUTH_OPEN_KIND_CAPABILITY', '2': 1},
    {'1': 'AUTH_OPEN_KIND_PAIRING', '2': 2},
  ],
};

/// Descriptor for `AuthOpenKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List authOpenKindDescriptor = $convert.base64Decode(
    'CgxBdXRoT3BlbktpbmQSHgoaQVVUSF9PUEVOX0tJTkRfVU5TUEVDSUZJRUQQABIdChlBVVRIX0'
    '9QRU5fS0lORF9DQVBBQklMSVRZEAESGgoWQVVUSF9PUEVOX0tJTkRfUEFJUklORxAC');

@$core.Deprecated('Use directSignalingErrorCodeDescriptor instead')
const DirectSignalingErrorCode$json = {
  '1': 'DirectSignalingErrorCode',
  '2': [
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_UNSPECIFIED', '2': 0},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_PROTOCOL', '2': 1},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_EXPIRED', '2': 2},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_REPLAYED', '2': 3},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_IDENTITY_MISMATCH', '2': 4},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_INTERNAL', '2': 5},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_OVERLOADED', '2': 6},
    {'1': 'DIRECT_SIGNALING_ERROR_CODE_AUTHORIZATION', '2': 7},
  ],
};

/// Descriptor for `DirectSignalingErrorCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List directSignalingErrorCodeDescriptor = $convert.base64Decode(
    'ChhEaXJlY3RTaWduYWxpbmdFcnJvckNvZGUSKwonRElSRUNUX1NJR05BTElOR19FUlJPUl9DT0'
    'RFX1VOU1BFQ0lGSUVEEAASKAokRElSRUNUX1NJR05BTElOR19FUlJPUl9DT0RFX1BST1RPQ09M'
    'EAESJwojRElSRUNUX1NJR05BTElOR19FUlJPUl9DT0RFX0VYUElSRUQQAhIoCiRESVJFQ1RfU0'
    'lHTkFMSU5HX0VSUk9SX0NPREVfUkVQTEFZRUQQAxIxCi1ESVJFQ1RfU0lHTkFMSU5HX0VSUk9S'
    'X0NPREVfSURFTlRJVFlfTUlTTUFUQ0gQBBIoCiRESVJFQ1RfU0lHTkFMSU5HX0VSUk9SX0NPRE'
    'VfSU5URVJOQUwQBRIqCiZESVJFQ1RfU0lHTkFMSU5HX0VSUk9SX0NPREVfT1ZFUkxPQURFRBAG'
    'Ei0KKURJUkVDVF9TSUdOQUxJTkdfRVJST1JfQ09ERV9BVVRIT1JJWkFUSU9OEAc=');

@$core.Deprecated('Use endpointConnectModeDescriptor instead')
const EndpointConnectMode$json = {
  '1': 'EndpointConnectMode',
  '2': [
    {'1': 'ENDPOINT_CONNECT_MODE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_CONNECT_MODE_AUTO', '2': 1},
    {'1': 'ENDPOINT_CONNECT_MODE_ON_DEMAND', '2': 2},
    {'1': 'ENDPOINT_CONNECT_MODE_MANUAL', '2': 3},
  ],
};

/// Descriptor for `EndpointConnectMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointConnectModeDescriptor = $convert.base64Decode(
    'ChNFbmRwb2ludENvbm5lY3RNb2RlEiUKIUVORFBPSU5UX0NPTk5FQ1RfTU9ERV9VTlNQRUNJRk'
    'lFRBAAEh4KGkVORFBPSU5UX0NPTk5FQ1RfTU9ERV9BVVRPEAESIwofRU5EUE9JTlRfQ09OTkVD'
    'VF9NT0RFX09OX0RFTUFORBACEiAKHEVORFBPSU5UX0NPTk5FQ1RfTU9ERV9NQU5VQUwQAw==');

@$core.Deprecated('Use managedWebRTCRelayModeDescriptor instead')
const ManagedWebRTCRelayMode$json = {
  '1': 'ManagedWebRTCRelayMode',
  '2': [
    {'1': 'MANAGED_WEBRTC_RELAY_MODE_UNSPECIFIED', '2': 0},
    {'1': 'MANAGED_WEBRTC_RELAY_MODE_AUTO', '2': 1},
    {'1': 'MANAGED_WEBRTC_RELAY_MODE_DIRECT', '2': 2},
    {'1': 'MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY', '2': 3},
    {'1': 'MANAGED_WEBRTC_RELAY_MODE_SMART_ROUTE', '2': 4},
  ],
};

/// Descriptor for `ManagedWebRTCRelayMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List managedWebRTCRelayModeDescriptor = $convert.base64Decode(
    'ChZNYW5hZ2VkV2ViUlRDUmVsYXlNb2RlEikKJU1BTkFHRURfV0VCUlRDX1JFTEFZX01PREVfVU'
    '5TUEVDSUZJRUQQABIiCh5NQU5BR0VEX1dFQlJUQ19SRUxBWV9NT0RFX0FVVE8QARIkCiBNQU5B'
    'R0VEX1dFQlJUQ19SRUxBWV9NT0RFX0RJUkVDVBACEigKJE1BTkFHRURfV0VCUlRDX1JFTEFZX0'
    '1PREVfUkVMQVlfT05MWRADEikKJU1BTkFHRURfV0VCUlRDX1JFTEFZX01PREVfU01BUlRfUk9V'
    'VEUQBA==');

@$core.Deprecated('Use endpointRoutePreferenceDescriptor instead')
const EndpointRoutePreference$json = {
  '1': 'EndpointRoutePreference',
  '2': [
    {'1': 'ENDPOINT_ROUTE_PREFERENCE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_ROUTE_PREFERENCE_AUTO', '2': 1},
    {'1': 'ENDPOINT_ROUTE_PREFERENCE_DIRECT', '2': 2},
    {'1': 'ENDPOINT_ROUTE_PREFERENCE_SSH', '2': 3},
    {'1': 'ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD', '2': 4},
  ],
};

/// Descriptor for `EndpointRoutePreference`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointRoutePreferenceDescriptor = $convert.base64Decode(
    'ChdFbmRwb2ludFJvdXRlUHJlZmVyZW5jZRIpCiVFTkRQT0lOVF9ST1VURV9QUkVGRVJFTkNFX1'
    'VOU1BFQ0lGSUVEEAASIgoeRU5EUE9JTlRfUk9VVEVfUFJFRkVSRU5DRV9BVVRPEAESJAogRU5E'
    'UE9JTlRfUk9VVEVfUFJFRkVSRU5DRV9ESVJFQ1QQAhIhCh1FTkRQT0lOVF9ST1VURV9QUkVGRV'
    'JFTkNFX1NTSBADEisKJ0VORFBPSU5UX1JPVVRFX1BSRUZFUkVOQ0VfTUFOQUdFRF9DTE9VRBAE');

@$core.Deprecated('Use managedWebRTCRelayTransportDescriptor instead')
const ManagedWebRTCRelayTransport$json = {
  '1': 'ManagedWebRTCRelayTransport',
  '2': [
    {'1': 'MANAGED_WEBRTC_RELAY_TRANSPORT_UNSPECIFIED', '2': 0},
    {'1': 'MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO', '2': 1},
    {'1': 'MANAGED_WEBRTC_RELAY_TRANSPORT_UDP', '2': 2},
    {'1': 'MANAGED_WEBRTC_RELAY_TRANSPORT_TCP', '2': 3},
  ],
};

/// Descriptor for `ManagedWebRTCRelayTransport`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List managedWebRTCRelayTransportDescriptor = $convert.base64Decode(
    'ChtNYW5hZ2VkV2ViUlRDUmVsYXlUcmFuc3BvcnQSLgoqTUFOQUdFRF9XRUJSVENfUkVMQVlfVF'
    'JBTlNQT1JUX1VOU1BFQ0lGSUVEEAASJwojTUFOQUdFRF9XRUJSVENfUkVMQVlfVFJBTlNQT1JU'
    'X0FVVE8QARImCiJNQU5BR0VEX1dFQlJUQ19SRUxBWV9UUkFOU1BPUlRfVURQEAISJgoiTUFOQU'
    'dFRF9XRUJSVENfUkVMQVlfVFJBTlNQT1JUX1RDUBAD');

@$core.Deprecated('Use endpointSourceDescriptor instead')
const EndpointSource$json = {
  '1': 'EndpointSource',
  '2': [
    {'1': 'ENDPOINT_SOURCE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_SOURCE_LAN', '2': 1},
    {'1': 'ENDPOINT_SOURCE_CLOUD', '2': 2},
    {'1': 'ENDPOINT_SOURCE_BOOTSTRAP', '2': 3},
    {'1': 'ENDPOINT_SOURCE_LOCAL', '2': 4},
    {'1': 'ENDPOINT_SOURCE_MANUAL', '2': 5},
    {'1': 'ENDPOINT_SOURCE_SHARE', '2': 6},
    {'1': 'ENDPOINT_SOURCE_USER', '2': 7},
  ],
};

/// Descriptor for `EndpointSource`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointSourceDescriptor = $convert.base64Decode(
    'Cg5FbmRwb2ludFNvdXJjZRIfChtFTkRQT0lOVF9TT1VSQ0VfVU5TUEVDSUZJRUQQABIXChNFTk'
    'RQT0lOVF9TT1VSQ0VfTEFOEAESGQoVRU5EUE9JTlRfU09VUkNFX0NMT1VEEAISHQoZRU5EUE9J'
    'TlRfU09VUkNFX0JPT1RTVFJBUBADEhkKFUVORFBPSU5UX1NPVVJDRV9MT0NBTBAEEhoKFkVORF'
    'BPSU5UX1NPVVJDRV9NQU5VQUwQBRIZChVFTkRQT0lOVF9TT1VSQ0VfU0hBUkUQBhIYChRFTkRQ'
    'T0lOVF9TT1VSQ0VfVVNFUhAH');

@$core.Deprecated('Use endpointCredentialKindDescriptor instead')
const EndpointCredentialKind$json = {
  '1': 'EndpointCredentialKind',
  '2': [
    {'1': 'ENDPOINT_CREDENTIAL_KIND_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_CREDENTIAL_KIND_SSH_AGENT', '2': 1},
    {'1': 'ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY', '2': 2},
    {'1': 'ENDPOINT_CREDENTIAL_KIND_SSH_PASSWORD', '2': 3},
    {'1': 'ENDPOINT_CREDENTIAL_KIND_CAPABILITY_GRANT', '2': 4},
    {'1': 'ENDPOINT_CREDENTIAL_KIND_CLOUD_PROFILE', '2': 5},
  ],
};

/// Descriptor for `EndpointCredentialKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointCredentialKindDescriptor = $convert.base64Decode(
    'ChZFbmRwb2ludENyZWRlbnRpYWxLaW5kEigKJEVORFBPSU5UX0NSRURFTlRJQUxfS0lORF9VTl'
    'NQRUNJRklFRBAAEiYKIkVORFBPSU5UX0NSRURFTlRJQUxfS0lORF9TU0hfQUdFTlQQARIsCihF'
    'TkRQT0lOVF9DUkVERU5USUFMX0tJTkRfU1NIX1BSSVZBVEVfS0VZEAISKQolRU5EUE9JTlRfQ1'
    'JFREVOVElBTF9LSU5EX1NTSF9QQVNTV09SRBADEi0KKUVORFBPSU5UX0NSRURFTlRJQUxfS0lO'
    'RF9DQVBBQklMSVRZX0dSQU5UEAQSKgomRU5EUE9JTlRfQ1JFREVOVElBTF9LSU5EX0NMT1VEX1'
    'BST0ZJTEUQBQ==');

@$core.Deprecated('Use channelBindingDescriptor instead')
const ChannelBinding$json = {
  '1': 'ChannelBinding',
  '2': [
    {
      '1': 'kind',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.ChannelBindingKind',
      '10': 'kind'
    },
    {'1': 'binding_hash', '3': 2, '4': 1, '5': 12, '10': 'bindingHash'},
  ],
};

/// Descriptor for `ChannelBinding`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List channelBindingDescriptor = $convert.base64Decode(
    'Cg5DaGFubmVsQmluZGluZxI9CgRraW5kGAEgASgOMikuYW55dHR5LnJlbW90ZS5hdXRoLnYxLk'
    'NoYW5uZWxCaW5kaW5nS2luZFIEa2luZBIhCgxiaW5kaW5nX2hhc2gYAiABKAxSC2JpbmRpbmdI'
    'YXNo');

@$core.Deprecated('Use authEnvelopeDescriptor instead')
const AuthEnvelope$json = {
  '1': 'AuthEnvelope',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {'1': 'auth_session_id', '3': 3, '4': 1, '5': 9, '10': 'authSessionId'},
    {
      '1': 'device_hello',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.DeviceHello',
      '9': 0,
      '10': 'deviceHello'
    },
    {
      '1': 'capability_open',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.CapabilityOpen',
      '9': 0,
      '10': 'capabilityOpen'
    },
    {
      '1': 'capability_accepted',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.CapabilityAccepted',
      '9': 0,
      '10': 'capabilityAccepted'
    },
    {
      '1': 'capability_rejected',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.CapabilityRejected',
      '9': 0,
      '10': 'capabilityRejected'
    },
    {
      '1': 'pairing_open',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingOpen',
      '9': 0,
      '10': 'pairingOpen'
    },
    {
      '1': 'pairing_accepted',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingAccepted',
      '9': 0,
      '10': 'pairingAccepted'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `AuthEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List authEnvelopeDescriptor = $convert.base64Decode(
    'CgxBdXRoRW52ZWxvcGUSGgoIcHJvdG9jb2wYASABKAlSCHByb3RvY29sEhgKB3ZlcnNpb24YAi'
    'ABKA1SB3ZlcnNpb24SJgoPYXV0aF9zZXNzaW9uX2lkGAMgASgJUg1hdXRoU2Vzc2lvbklkEkcK'
    'DGRldmljZV9oZWxsbxgEIAEoCzIiLmFueXR0eS5yZW1vdGUuYXV0aC52MS5EZXZpY2VIZWxsb0'
    'gAUgtkZXZpY2VIZWxsbxJQCg9jYXBhYmlsaXR5X29wZW4YBSABKAsyJS5hbnl0dHkucmVtb3Rl'
    'LmF1dGgudjEuQ2FwYWJpbGl0eU9wZW5IAFIOY2FwYWJpbGl0eU9wZW4SXAoTY2FwYWJpbGl0eV'
    '9hY2NlcHRlZBgGIAEoCzIpLmFueXR0eS5yZW1vdGUuYXV0aC52MS5DYXBhYmlsaXR5QWNjZXB0'
    'ZWRIAFISY2FwYWJpbGl0eUFjY2VwdGVkElwKE2NhcGFiaWxpdHlfcmVqZWN0ZWQYByABKAsyKS'
    '5hbnl0dHkucmVtb3RlLmF1dGgudjEuQ2FwYWJpbGl0eVJlamVjdGVkSABSEmNhcGFiaWxpdHlS'
    'ZWplY3RlZBJHCgxwYWlyaW5nX29wZW4YCCABKAsyIi5hbnl0dHkucmVtb3RlLmF1dGgudjEuUG'
    'FpcmluZ09wZW5IAFILcGFpcmluZ09wZW4SUwoQcGFpcmluZ19hY2NlcHRlZBgJIAEoCzImLmFu'
    'eXR0eS5yZW1vdGUuYXV0aC52MS5QYWlyaW5nQWNjZXB0ZWRIAFIPcGFpcmluZ0FjY2VwdGVkQg'
    'kKB3BheWxvYWQ=');

@$core.Deprecated('Use deviceHelloDescriptor instead')
const DeviceHello$json = {
  '1': 'DeviceHello',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
    {
      '1': 'device_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {'1': 'server_nonce', '3': 4, '4': 1, '5': 12, '10': 'serverNonce'},
    {
      '1': 'channel_binding',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ChannelBinding',
      '10': 'channelBinding'
    },
    {
      '1': 'issued_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {'1': 'signature', '3': 7, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `DeviceHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceHelloDescriptor = $convert.base64Decode(
    'CgtEZXZpY2VIZWxsbxIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlkEioKEWRldmljZV9wdW'
    'JsaWNfa2V5GAIgASgMUg9kZXZpY2VQdWJsaWNLZXkSLQoSZGV2aWNlX2ZpbmdlcnByaW50GAMg'
    'ASgJUhFkZXZpY2VGaW5nZXJwcmludBIhCgxzZXJ2ZXJfbm9uY2UYBCABKAxSC3NlcnZlck5vbm'
    'NlEk4KD2NoYW5uZWxfYmluZGluZxgFIAEoCzIlLmFueXR0eS5yZW1vdGUuYXV0aC52MS5DaGFu'
    'bmVsQmluZGluZ1IOY2hhbm5lbEJpbmRpbmcSLQoTaXNzdWVkX2F0X3VuaXhfbmFubxgGIAEoA1'
    'IQaXNzdWVkQXRVbml4TmFubxIcCglzaWduYXR1cmUYByABKAxSCXNpZ25hdHVyZQ==');

@$core.Deprecated('Use capabilityOpenDescriptor instead')
const CapabilityOpen$json = {
  '1': 'CapabilityOpen',
  '2': [
    {'1': 'grant', '3': 1, '4': 1, '5': 9, '10': 'grant'},
    {
      '1': 'client_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {'1': 'client_nonce', '3': 3, '4': 1, '5': 12, '10': 'clientNonce'},
    {'1': 'proof', '3': 4, '4': 1, '5': 12, '10': 'proof'},
  ],
};

/// Descriptor for `CapabilityOpen`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilityOpenDescriptor = $convert.base64Decode(
    'Cg5DYXBhYmlsaXR5T3BlbhIUCgVncmFudBgBIAEoCVIFZ3JhbnQSKgoRY2xpZW50X3B1YmxpY1'
    '9rZXkYAiABKAxSD2NsaWVudFB1YmxpY0tleRIhCgxjbGllbnRfbm9uY2UYAyABKAxSC2NsaWVu'
    'dE5vbmNlEhQKBXByb29mGAQgASgMUgVwcm9vZg==');

@$core.Deprecated('Use pairingOpenDescriptor instead')
const PairingOpen$json = {
  '1': 'PairingOpen',
  '2': [
    {
      '1': 'pairing_claim_offer',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'pairingClaimOffer'
    },
    {
      '1': 'client_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {'1': 'client_label', '3': 3, '4': 1, '5': 9, '10': 'clientLabel'},
    {'1': 'client_nonce', '3': 4, '4': 1, '5': 12, '10': 'clientNonce'},
    {'1': 'proof', '3': 5, '4': 1, '5': 12, '10': 'proof'},
    {'1': 'client_product', '3': 6, '4': 1, '5': 13, '10': 'clientProduct'},
  ],
};

/// Descriptor for `PairingOpen`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingOpenDescriptor = $convert.base64Decode(
    'CgtQYWlyaW5nT3BlbhIuChNwYWlyaW5nX2NsYWltX29mZmVyGAEgASgMUhFwYWlyaW5nQ2xhaW'
    '1PZmZlchIqChFjbGllbnRfcHVibGljX2tleRgCIAEoDFIPY2xpZW50UHVibGljS2V5EiEKDGNs'
    'aWVudF9sYWJlbBgDIAEoCVILY2xpZW50TGFiZWwSIQoMY2xpZW50X25vbmNlGAQgASgMUgtjbG'
    'llbnROb25jZRIUCgVwcm9vZhgFIAEoDFIFcHJvb2YSJQoOY2xpZW50X3Byb2R1Y3QYBiABKA1S'
    'DWNsaWVudFByb2R1Y3Q=');

@$core.Deprecated('Use scopeSummaryDescriptor instead')
const ScopeSummary$json = {
  '1': 'ScopeSummary',
  '2': [
    {
      '1': 'kind',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.ScopeKind',
      '10': 'kind'
    },
    {'1': 'terminal_id', '3': 2, '4': 1, '5': 9, '10': 'terminalId'},
    {
      '1': 'manage_client_access',
      '3': 3,
      '4': 1,
      '5': 8,
      '10': 'manageClientAccess'
    },
  ],
};

/// Descriptor for `ScopeSummary`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List scopeSummaryDescriptor = $convert.base64Decode(
    'CgxTY29wZVN1bW1hcnkSNAoEa2luZBgBIAEoDjIgLmFueXR0eS5yZW1vdGUuYXV0aC52MS5TY2'
    '9wZUtpbmRSBGtpbmQSHwoLdGVybWluYWxfaWQYAiABKAlSCnRlcm1pbmFsSWQSMAoUbWFuYWdl'
    'X2NsaWVudF9hY2Nlc3MYAyABKAhSEm1hbmFnZUNsaWVudEFjY2Vzcw==');

@$core.Deprecated('Use clientAccessScopeDescriptor instead')
const ClientAccessScope$json = {
  '1': 'ClientAccessScope',
  '2': [
    {'1': 'allow_daemon', '3': 1, '4': 1, '5': 8, '10': 'allowDaemon'},
    {'1': 'terminal_id', '3': 2, '4': 1, '5': 9, '10': 'terminalId'},
    {
      '1': 'machine_events_only',
      '3': 3,
      '4': 1,
      '5': 8,
      '10': 'machineEventsOnly'
    },
    {
      '1': 'file_read_metadata',
      '3': 4,
      '4': 1,
      '5': 8,
      '10': 'fileReadMetadata'
    },
    {'1': 'file_read_content', '3': 5, '4': 1, '5': 8, '10': 'fileReadContent'},
    {
      '1': 'file_write_content',
      '3': 6,
      '4': 1,
      '5': 8,
      '10': 'fileWriteContent'
    },
    {'1': 'file_mutate', '3': 7, '4': 1, '5': 8, '10': 'fileMutate'},
    {
      '1': 'manage_client_access',
      '3': 8,
      '4': 1,
      '5': 8,
      '10': 'manageClientAccess'
    },
  ],
};

/// Descriptor for `ClientAccessScope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessScopeDescriptor = $convert.base64Decode(
    'ChFDbGllbnRBY2Nlc3NTY29wZRIhCgxhbGxvd19kYWVtb24YASABKAhSC2FsbG93RGFlbW9uEh'
    '8KC3Rlcm1pbmFsX2lkGAIgASgJUgp0ZXJtaW5hbElkEi4KE21hY2hpbmVfZXZlbnRzX29ubHkY'
    'AyABKAhSEW1hY2hpbmVFdmVudHNPbmx5EiwKEmZpbGVfcmVhZF9tZXRhZGF0YRgEIAEoCFIQZm'
    'lsZVJlYWRNZXRhZGF0YRIqChFmaWxlX3JlYWRfY29udGVudBgFIAEoCFIPZmlsZVJlYWRDb250'
    'ZW50EiwKEmZpbGVfd3JpdGVfY29udGVudBgGIAEoCFIQZmlsZVdyaXRlQ29udGVudBIfCgtmaW'
    'xlX211dGF0ZRgHIAEoCFIKZmlsZU11dGF0ZRIwChRtYW5hZ2VfY2xpZW50X2FjY2VzcxgIIAEo'
    'CFISbWFuYWdlQ2xpZW50QWNjZXNz');

@$core.Deprecated('Use clientAccessIdentityResultDescriptor instead')
const ClientAccessIdentityResult$json = {
  '1': 'ClientAccessIdentityResult',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_fingerprint',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {
      '1': 'device_public_key',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
  ],
};

/// Descriptor for `ClientAccessIdentityResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessIdentityResultDescriptor =
    $convert.base64Decode(
        'ChpDbGllbnRBY2Nlc3NJZGVudGl0eVJlc3VsdBIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZU'
        'lkEi0KEmRldmljZV9maW5nZXJwcmludBgCIAEoCVIRZGV2aWNlRmluZ2VycHJpbnQSKgoRZGV2'
        'aWNlX3B1YmxpY19rZXkYAyABKAxSD2RldmljZVB1YmxpY0tleQ==');

@$core.Deprecated('Use deviceIdentityProofInputDescriptor instead')
const DeviceIdentityProofInput$json = {
  '1': 'DeviceIdentityProofInput',
  '2': [
    {'1': 'domain', '3': 1, '4': 1, '5': 9, '10': 'domain'},
    {'1': 'challenge', '3': 2, '4': 1, '5': 12, '10': 'challenge'},
    {'1': 'device_id', '3': 3, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_fingerprint',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {
      '1': 'device_public_key',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
  ],
};

/// Descriptor for `DeviceIdentityProofInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceIdentityProofInputDescriptor = $convert.base64Decode(
    'ChhEZXZpY2VJZGVudGl0eVByb29mSW5wdXQSFgoGZG9tYWluGAEgASgJUgZkb21haW4SHAoJY2'
    'hhbGxlbmdlGAIgASgMUgljaGFsbGVuZ2USGwoJZGV2aWNlX2lkGAMgASgJUghkZXZpY2VJZBIt'
    'ChJkZXZpY2VfZmluZ2VycHJpbnQYBCABKAlSEWRldmljZUZpbmdlcnByaW50EioKEWRldmljZV'
    '9wdWJsaWNfa2V5GAUgASgMUg9kZXZpY2VQdWJsaWNLZXk=');

@$core.Deprecated('Use clientAccessTicketCreateRequestDescriptor instead')
const ClientAccessTicketCreateRequest$json = {
  '1': 'ClientAccessTicketCreateRequest',
  '2': [
    {'1': 'label', '3': 1, '4': 1, '5': 9, '10': 'label'},
    {
      '1': 'scope',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessScope',
      '10': 'scope'
    },
    {
      '1': 'ticket_ttl_seconds',
      '3': 3,
      '4': 1,
      '5': 3,
      '10': 'ticketTtlSeconds'
    },
    {
      '1': 'grant_lifetime_seconds',
      '3': 4,
      '4': 1,
      '5': 3,
      '10': 'grantLifetimeSeconds'
    },
    {
      '1': 'routes',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRouteConfigV1',
      '10': 'routes'
    },
    {'1': 'access_label', '3': 6, '4': 1, '5': 9, '10': 'accessLabel'},
  ],
};

/// Descriptor for `ClientAccessTicketCreateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessTicketCreateRequestDescriptor = $convert.base64Decode(
    'Ch9DbGllbnRBY2Nlc3NUaWNrZXRDcmVhdGVSZXF1ZXN0EhQKBWxhYmVsGAEgASgJUgVsYWJlbB'
    'I+CgVzY29wZRgCIAEoCzIoLmFueXR0eS5yZW1vdGUuYXV0aC52MS5DbGllbnRBY2Nlc3NTY29w'
    'ZVIFc2NvcGUSLAoSdGlja2V0X3R0bF9zZWNvbmRzGAMgASgDUhB0aWNrZXRUdGxTZWNvbmRzEj'
    'QKFmdyYW50X2xpZmV0aW1lX3NlY29uZHMYBCABKANSFGdyYW50TGlmZXRpbWVTZWNvbmRzEkQK'
    'BnJvdXRlcxgFIAMoCzIsLmFueXR0eS5yZW1vdGUuYXV0aC52MS5FbmRwb2ludFJvdXRlQ29uZm'
    'lnVjFSBnJvdXRlcxIhCgxhY2Nlc3NfbGFiZWwYBiABKAlSC2FjY2Vzc0xhYmVs');

@$core.Deprecated('Use clientAccessTicketCreateResultDescriptor instead')
const ClientAccessTicketCreateResult$json = {
  '1': 'ClientAccessTicketCreateResult',
  '2': [
    {'1': 'ticket_id', '3': 1, '4': 1, '5': 9, '10': 'ticketId'},
    {
      '1': 'expires_at_unix_nano',
      '3': 2,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'claim_offer', '3': 3, '4': 1, '5': 12, '10': 'claimOffer'},
    {'1': 'claim_code', '3': 4, '4': 1, '5': 9, '10': 'claimCode'},
  ],
};

/// Descriptor for `ClientAccessTicketCreateResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessTicketCreateResultDescriptor =
    $convert.base64Decode(
        'Ch5DbGllbnRBY2Nlc3NUaWNrZXRDcmVhdGVSZXN1bHQSGwoJdGlja2V0X2lkGAEgASgJUgh0aW'
        'NrZXRJZBIvChRleHBpcmVzX2F0X3VuaXhfbmFubxgCIAEoA1IRZXhwaXJlc0F0VW5peE5hbm8S'
        'HwoLY2xhaW1fb2ZmZXIYAyABKAxSCmNsYWltT2ZmZXISHQoKY2xhaW1fY29kZRgEIAEoCVIJY2'
        'xhaW1Db2Rl');

@$core.Deprecated('Use clientAccessRevokeRequestDescriptor instead')
const ClientAccessRevokeRequest$json = {
  '1': 'ClientAccessRevokeRequest',
  '2': [
    {'1': 'grant_id', '3': 1, '4': 1, '5': 9, '10': 'grantId'},
  ],
};

/// Descriptor for `ClientAccessRevokeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessRevokeRequestDescriptor =
    $convert.base64Decode(
        'ChlDbGllbnRBY2Nlc3NSZXZva2VSZXF1ZXN0EhkKCGdyYW50X2lkGAEgASgJUgdncmFudElk');

@$core.Deprecated('Use clientAccessRecordDescriptor instead')
const ClientAccessRecord$json = {
  '1': 'ClientAccessRecord',
  '2': [
    {'1': 'grant_id', '3': 1, '4': 1, '5': 9, '10': 'grantId'},
    {'1': 'revocation_id', '3': 2, '4': 1, '5': 9, '10': 'revocationId'},
    {
      '1': 'subject_key_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'subjectKeyFingerprint'
    },
    {'1': 'client_label', '3': 4, '4': 1, '5': 9, '10': 'clientLabel'},
    {
      '1': 'scope',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessScope',
      '10': 'scope'
    },
    {
      '1': 'issued_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {
      '1': 'revoked_at_unix_nano',
      '3': 8,
      '4': 1,
      '5': 3,
      '10': 'revokedAtUnixNano'
    },
    {'1': 'access_label', '3': 9, '4': 1, '5': 9, '10': 'accessLabel'},
  ],
};

/// Descriptor for `ClientAccessRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessRecordDescriptor = $convert.base64Decode(
    'ChJDbGllbnRBY2Nlc3NSZWNvcmQSGQoIZ3JhbnRfaWQYASABKAlSB2dyYW50SWQSIwoNcmV2b2'
    'NhdGlvbl9pZBgCIAEoCVIMcmV2b2NhdGlvbklkEjYKF3N1YmplY3Rfa2V5X2ZpbmdlcnByaW50'
    'GAMgASgJUhVzdWJqZWN0S2V5RmluZ2VycHJpbnQSIQoMY2xpZW50X2xhYmVsGAQgASgJUgtjbG'
    'llbnRMYWJlbBI+CgVzY29wZRgFIAEoCzIoLmFueXR0eS5yZW1vdGUuYXV0aC52MS5DbGllbnRB'
    'Y2Nlc3NTY29wZVIFc2NvcGUSLQoTaXNzdWVkX2F0X3VuaXhfbmFubxgGIAEoA1IQaXNzdWVkQX'
    'RVbml4TmFubxIvChRleHBpcmVzX2F0X3VuaXhfbmFubxgHIAEoA1IRZXhwaXJlc0F0VW5peE5h'
    'bm8SLwoUcmV2b2tlZF9hdF91bml4X25hbm8YCCABKANSEXJldm9rZWRBdFVuaXhOYW5vEiEKDG'
    'FjY2Vzc19sYWJlbBgJIAEoCVILYWNjZXNzTGFiZWw=');

@$core.Deprecated('Use clientAccessListResultDescriptor instead')
const ClientAccessListResult$json = {
  '1': 'ClientAccessListResult',
  '2': [
    {
      '1': 'records',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientAccessRecord',
      '10': 'records'
    },
  ],
};

/// Descriptor for `ClientAccessListResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientAccessListResultDescriptor =
    $convert.base64Decode(
        'ChZDbGllbnRBY2Nlc3NMaXN0UmVzdWx0EkMKB3JlY29yZHMYASADKAsyKS5hbnl0dHkucmVtb3'
        'RlLmF1dGgudjEuQ2xpZW50QWNjZXNzUmVjb3JkUgdyZWNvcmRz');

@$core.Deprecated('Use capabilityAcceptedDescriptor instead')
const CapabilityAccepted$json = {
  '1': 'CapabilityAccepted',
  '2': [
    {'1': 'grant_id', '3': 1, '4': 1, '5': 9, '10': 'grantId'},
    {
      '1': 'scope',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ScopeSummary',
      '10': 'scope'
    },
    {
      '1': 'subject_key_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'subjectKeyFingerprint'
    },
  ],
};

/// Descriptor for `CapabilityAccepted`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilityAcceptedDescriptor = $convert.base64Decode(
    'ChJDYXBhYmlsaXR5QWNjZXB0ZWQSGQoIZ3JhbnRfaWQYASABKAlSB2dyYW50SWQSOQoFc2NvcG'
    'UYAiABKAsyIy5hbnl0dHkucmVtb3RlLmF1dGgudjEuU2NvcGVTdW1tYXJ5UgVzY29wZRI2Chdz'
    'dWJqZWN0X2tleV9maW5nZXJwcmludBgDIAEoCVIVc3ViamVjdEtleUZpbmdlcnByaW50');

@$core.Deprecated('Use pairingAcceptedDescriptor instead')
const PairingAccepted$json = {
  '1': 'PairingAccepted',
  '2': [
    {'1': 'grant', '3': 1, '4': 1, '5': 9, '10': 'grant'},
    {'1': 'delivery_receipt', '3': 2, '4': 1, '5': 9, '10': 'deliveryReceipt'},
    {
      '1': 'subject_key_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'subjectKeyFingerprint'
    },
    {
      '1': 'scope',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ScopeSummary',
      '10': 'scope'
    },
    {'1': 'pairing_bundle', '3': 5, '4': 1, '5': 12, '10': 'pairingBundle'},
    {
      '1': 'cloud_route_grant',
      '3': 6,
      '4': 1,
      '5': 12,
      '10': 'cloudRouteGrant'
    },
    {
      '1': 'cloud_edge_locator',
      '3': 7,
      '4': 1,
      '5': 12,
      '10': 'cloudEdgeLocator'
    },
  ],
};

/// Descriptor for `PairingAccepted`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingAcceptedDescriptor = $convert.base64Decode(
    'Cg9QYWlyaW5nQWNjZXB0ZWQSFAoFZ3JhbnQYASABKAlSBWdyYW50EikKEGRlbGl2ZXJ5X3JlY2'
    'VpcHQYAiABKAlSD2RlbGl2ZXJ5UmVjZWlwdBI2ChdzdWJqZWN0X2tleV9maW5nZXJwcmludBgD'
    'IAEoCVIVc3ViamVjdEtleUZpbmdlcnByaW50EjkKBXNjb3BlGAQgASgLMiMuYW55dHR5LnJlbW'
    '90ZS5hdXRoLnYxLlNjb3BlU3VtbWFyeVIFc2NvcGUSJQoOcGFpcmluZ19idW5kbGUYBSABKAxS'
    'DXBhaXJpbmdCdW5kbGUSKgoRY2xvdWRfcm91dGVfZ3JhbnQYBiABKAxSD2Nsb3VkUm91dGVHcm'
    'FudBIsChJjbG91ZF9lZGdlX2xvY2F0b3IYByABKAxSEGNsb3VkRWRnZUxvY2F0b3I=');

@$core.Deprecated('Use capabilityRejectedDescriptor instead')
const CapabilityRejected$json = {
  '1': 'CapabilityRejected',
  '2': [
    {
      '1': 'code',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.AuthErrorCode',
      '10': 'code'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `CapabilityRejected`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilityRejectedDescriptor = $convert.base64Decode(
    'ChJDYXBhYmlsaXR5UmVqZWN0ZWQSOAoEY29kZRgBIAEoDjIkLmFueXR0eS5yZW1vdGUuYXV0aC'
    '52MS5BdXRoRXJyb3JDb2RlUgRjb2RlEhgKB21lc3NhZ2UYAiABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use deviceHelloSignatureInputDescriptor instead')
const DeviceHelloSignatureInput$json = {
  '1': 'DeviceHelloSignatureInput',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {'1': 'auth_session_id', '3': 3, '4': 1, '5': 9, '10': 'authSessionId'},
    {'1': 'device_id', '3': 4, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_public_key',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
    {
      '1': 'device_fingerprint',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {'1': 'server_nonce', '3': 7, '4': 1, '5': 12, '10': 'serverNonce'},
    {
      '1': 'channel_binding',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ChannelBinding',
      '10': 'channelBinding'
    },
    {
      '1': 'issued_at_unix_nano',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
  ],
};

/// Descriptor for `DeviceHelloSignatureInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceHelloSignatureInputDescriptor = $convert.base64Decode(
    'ChlEZXZpY2VIZWxsb1NpZ25hdHVyZUlucHV0EhoKCHByb3RvY29sGAEgASgJUghwcm90b2NvbB'
    'IYCgd2ZXJzaW9uGAIgASgNUgd2ZXJzaW9uEiYKD2F1dGhfc2Vzc2lvbl9pZBgDIAEoCVINYXV0'
    'aFNlc3Npb25JZBIbCglkZXZpY2VfaWQYBCABKAlSCGRldmljZUlkEioKEWRldmljZV9wdWJsaW'
    'Nfa2V5GAUgASgMUg9kZXZpY2VQdWJsaWNLZXkSLQoSZGV2aWNlX2ZpbmdlcnByaW50GAYgASgJ'
    'UhFkZXZpY2VGaW5nZXJwcmludBIhCgxzZXJ2ZXJfbm9uY2UYByABKAxSC3NlcnZlck5vbmNlEk'
    '4KD2NoYW5uZWxfYmluZGluZxgIIAEoCzIlLmFueXR0eS5yZW1vdGUuYXV0aC52MS5DaGFubmVs'
    'QmluZGluZ1IOY2hhbm5lbEJpbmRpbmcSLQoTaXNzdWVkX2F0X3VuaXhfbmFubxgJIAEoA1IQaX'
    'NzdWVkQXRVbml4TmFubw==');

@$core.Deprecated('Use clientProofInputDescriptor instead')
const ClientProofInput$json = {
  '1': 'ClientProofInput',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {'1': 'auth_session_id', '3': 3, '4': 1, '5': 9, '10': 'authSessionId'},
    {'1': 'server_nonce', '3': 4, '4': 1, '5': 12, '10': 'serverNonce'},
    {'1': 'client_nonce', '3': 5, '4': 1, '5': 12, '10': 'clientNonce'},
    {
      '1': 'channel_binding',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ChannelBinding',
      '10': 'channelBinding'
    },
    {
      '1': 'credential_sha256',
      '3': 7,
      '4': 1,
      '5': 12,
      '10': 'credentialSha256'
    },
    {
      '1': 'client_public_key',
      '3': 8,
      '4': 1,
      '5': 12,
      '10': 'clientPublicKey'
    },
    {
      '1': 'open_kind',
      '3': 9,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.AuthOpenKind',
      '10': 'openKind'
    },
  ],
};

/// Descriptor for `ClientProofInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientProofInputDescriptor = $convert.base64Decode(
    'ChBDbGllbnRQcm9vZklucHV0EhoKCHByb3RvY29sGAEgASgJUghwcm90b2NvbBIYCgd2ZXJzaW'
    '9uGAIgASgNUgd2ZXJzaW9uEiYKD2F1dGhfc2Vzc2lvbl9pZBgDIAEoCVINYXV0aFNlc3Npb25J'
    'ZBIhCgxzZXJ2ZXJfbm9uY2UYBCABKAxSC3NlcnZlck5vbmNlEiEKDGNsaWVudF9ub25jZRgFIA'
    'EoDFILY2xpZW50Tm9uY2USTgoPY2hhbm5lbF9iaW5kaW5nGAYgASgLMiUuYW55dHR5LnJlbW90'
    'ZS5hdXRoLnYxLkNoYW5uZWxCaW5kaW5nUg5jaGFubmVsQmluZGluZxIrChFjcmVkZW50aWFsX3'
    'NoYTI1NhgHIAEoDFIQY3JlZGVudGlhbFNoYTI1NhIqChFjbGllbnRfcHVibGljX2tleRgIIAEo'
    'DFIPY2xpZW50UHVibGljS2V5EkAKCW9wZW5fa2luZBgJIAEoDjIjLmFueXR0eS5yZW1vdGUuYX'
    'V0aC52MS5BdXRoT3BlbktpbmRSCG9wZW5LaW5k');

@$core.Deprecated('Use directIceCandidateDescriptor instead')
const DirectIceCandidate$json = {
  '1': 'DirectIceCandidate',
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

/// Descriptor for `DirectIceCandidate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directIceCandidateDescriptor = $convert.base64Decode(
    'ChJEaXJlY3RJY2VDYW5kaWRhdGUSHAoJY2FuZGlkYXRlGAEgASgJUgljYW5kaWRhdGUSFwoHc2'
    'RwX21pZBgCIAEoCVIGc2RwTWlkEiYKD3NkcF9tbGluZV9pbmRleBgDIAEoDVINc2RwTWxpbmVJ'
    'bmRleBIrChF1c2VybmFtZV9mcmFnbWVudBgEIAEoCVIQdXNlcm5hbWVGcmFnbWVudA==');

@$core.Deprecated('Use directSignalingRequestV2Descriptor instead')
const DirectSignalingRequestV2$json = {
  '1': 'DirectSignalingRequestV2',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'request_id', '3': 2, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'expected_device_id',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'expectedDeviceId'
    },
    {
      '1': 'expected_device_fingerprint',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'expectedDeviceFingerprint'
    },
    {'1': 'offer_sdp', '3': 5, '4': 1, '5': 9, '10': 'offerSdp'},
    {
      '1': 'issued_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'grant_id', '3': 8, '4': 1, '5': 9, '10': 'grantId'},
    {
      '1': 'grant_expires_at_unix_nano',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'grantExpiresAtUnixNano'
    },
    {
      '1': 'pairing_claim_digest',
      '3': 10,
      '4': 1,
      '5': 12,
      '10': 'pairingClaimDigest'
    },
    {
      '1': 'pairing_client_public_key',
      '3': 11,
      '4': 1,
      '5': 12,
      '10': 'pairingClientPublicKey'
    },
    {
      '1': 'pairing_expires_at_unix_nano',
      '3': 12,
      '4': 1,
      '5': 3,
      '10': 'pairingExpiresAtUnixNano'
    },
  ],
};

/// Descriptor for `DirectSignalingRequestV2`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directSignalingRequestV2Descriptor = $convert.base64Decode(
    'ChhEaXJlY3RTaWduYWxpbmdSZXF1ZXN0VjISJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDXNjaG'
    'VtYVZlcnNpb24SHQoKcmVxdWVzdF9pZBgCIAEoCVIJcmVxdWVzdElkEiwKEmV4cGVjdGVkX2Rl'
    'dmljZV9pZBgDIAEoCVIQZXhwZWN0ZWREZXZpY2VJZBI+ChtleHBlY3RlZF9kZXZpY2VfZmluZ2'
    'VycHJpbnQYBCABKAlSGWV4cGVjdGVkRGV2aWNlRmluZ2VycHJpbnQSGwoJb2ZmZXJfc2RwGAUg'
    'ASgJUghvZmZlclNkcBItChNpc3N1ZWRfYXRfdW5peF9uYW5vGAYgASgDUhBpc3N1ZWRBdFVuaX'
    'hOYW5vEi8KFGV4cGlyZXNfYXRfdW5peF9uYW5vGAcgASgDUhFleHBpcmVzQXRVbml4TmFubxIZ'
    'CghncmFudF9pZBgIIAEoCVIHZ3JhbnRJZBI6ChpncmFudF9leHBpcmVzX2F0X3VuaXhfbmFubx'
    'gJIAEoA1IWZ3JhbnRFeHBpcmVzQXRVbml4TmFubxIwChRwYWlyaW5nX2NsYWltX2RpZ2VzdBgK'
    'IAEoDFIScGFpcmluZ0NsYWltRGlnZXN0EjkKGXBhaXJpbmdfY2xpZW50X3B1YmxpY19rZXkYCy'
    'ABKAxSFnBhaXJpbmdDbGllbnRQdWJsaWNLZXkSPgoccGFpcmluZ19leHBpcmVzX2F0X3VuaXhf'
    'bmFubxgMIAEoA1IYcGFpcmluZ0V4cGlyZXNBdFVuaXhOYW5v');

@$core.Deprecated('Use directSignalingAnswerV2Descriptor instead')
const DirectSignalingAnswerV2$json = {
  '1': 'DirectSignalingAnswerV2',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'request_id', '3': 2, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'identity',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointDaemonIdentity',
      '10': 'identity'
    },
    {'1': 'answer_sdp', '3': 4, '4': 1, '5': 9, '10': 'answerSdp'},
    {
      '1': 'candidates',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.DirectIceCandidate',
      '10': 'candidates'
    },
    {
      '1': 'issued_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'signature', '3': 8, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `DirectSignalingAnswerV2`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directSignalingAnswerV2Descriptor = $convert.base64Decode(
    'ChdEaXJlY3RTaWduYWxpbmdBbnN3ZXJWMhIlCg5zY2hlbWFfdmVyc2lvbhgBIAEoDVINc2NoZW'
    '1hVmVyc2lvbhIdCgpyZXF1ZXN0X2lkGAIgASgJUglyZXF1ZXN0SWQSSQoIaWRlbnRpdHkYAyAB'
    'KAsyLS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnREYWVtb25JZGVudGl0eVIIaWRlbn'
    'RpdHkSHQoKYW5zd2VyX3NkcBgEIAEoCVIJYW5zd2VyU2RwEkkKCmNhbmRpZGF0ZXMYBSADKAsy'
    'KS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRGlyZWN0SWNlQ2FuZGlkYXRlUgpjYW5kaWRhdGVzEi'
    '0KE2lzc3VlZF9hdF91bml4X25hbm8YBiABKANSEGlzc3VlZEF0VW5peE5hbm8SLwoUZXhwaXJl'
    'c19hdF91bml4X25hbm8YByABKANSEWV4cGlyZXNBdFVuaXhOYW5vEhwKCXNpZ25hdHVyZRgIIA'
    'EoDFIJc2lnbmF0dXJl');

@$core.Deprecated('Use directSignalingAnswerSignatureInputDescriptor instead')
const DirectSignalingAnswerSignatureInput$json = {
  '1': 'DirectSignalingAnswerSignatureInput',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {
      '1': 'answer',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.DirectSignalingAnswerV2',
      '10': 'answer'
    },
  ],
};

/// Descriptor for `DirectSignalingAnswerSignatureInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directSignalingAnswerSignatureInputDescriptor =
    $convert.base64Decode(
        'CiNEaXJlY3RTaWduYWxpbmdBbnN3ZXJTaWduYXR1cmVJbnB1dBIaCghwcm90b2NvbBgBIAEoCV'
        'IIcHJvdG9jb2wSGAoHdmVyc2lvbhgCIAEoDVIHdmVyc2lvbhJGCgZhbnN3ZXIYAyABKAsyLi5h'
        'bnl0dHkucmVtb3RlLmF1dGgudjEuRGlyZWN0U2lnbmFsaW5nQW5zd2VyVjJSBmFuc3dlcg==');

@$core.Deprecated('Use directSignalingErrorV2Descriptor instead')
const DirectSignalingErrorV2$json = {
  '1': 'DirectSignalingErrorV2',
  '2': [
    {
      '1': 'code',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.DirectSignalingErrorCode',
      '10': 'code'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `DirectSignalingErrorV2`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directSignalingErrorV2Descriptor = $convert.base64Decode(
    'ChZEaXJlY3RTaWduYWxpbmdFcnJvclYyEkMKBGNvZGUYASABKA4yLy5hbnl0dHkucmVtb3RlLm'
    'F1dGgudjEuRGlyZWN0U2lnbmFsaW5nRXJyb3JDb2RlUgRjb2RlEhgKB21lc3NhZ2UYAiABKAlS'
    'B21lc3NhZ2U=');

@$core.Deprecated('Use directSignalingResponseV2Descriptor instead')
const DirectSignalingResponseV2$json = {
  '1': 'DirectSignalingResponseV2',
  '2': [
    {
      '1': 'answer',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.DirectSignalingAnswerV2',
      '9': 0,
      '10': 'answer'
    },
    {
      '1': 'error',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.DirectSignalingErrorV2',
      '9': 0,
      '10': 'error'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `DirectSignalingResponseV2`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directSignalingResponseV2Descriptor = $convert.base64Decode(
    'ChlEaXJlY3RTaWduYWxpbmdSZXNwb25zZVYyEkgKBmFuc3dlchgBIAEoCzIuLmFueXR0eS5yZW'
    '1vdGUuYXV0aC52MS5EaXJlY3RTaWduYWxpbmdBbnN3ZXJWMkgAUgZhbnN3ZXISRQoFZXJyb3IY'
    'AiABKAsyLS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRGlyZWN0U2lnbmFsaW5nRXJyb3JWMkgAUg'
    'VlcnJvckIJCgdwYXlsb2Fk');

@$core.Deprecated('Use endpointDaemonIdentityDescriptor instead')
const EndpointDaemonIdentity$json = {
  '1': 'EndpointDaemonIdentity',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
    {
      '1': 'device_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
  ],
};

/// Descriptor for `EndpointDaemonIdentity`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointDaemonIdentityDescriptor = $convert.base64Decode(
    'ChZFbmRwb2ludERhZW1vbklkZW50aXR5EhsKCWRldmljZV9pZBgBIAEoCVIIZGV2aWNlSWQSKg'
    'oRZGV2aWNlX3B1YmxpY19rZXkYAiABKAxSD2RldmljZVB1YmxpY0tleRItChJkZXZpY2VfZmlu'
    'Z2VycHJpbnQYAyABKAlSEWRldmljZUZpbmdlcnByaW50');

@$core.Deprecated('Use endpointSelectionPolicyDescriptor instead')
const EndpointSelectionPolicy$json = {
  '1': 'EndpointSelectionPolicy',
  '2': [
    {
      '1': 'hedge_delay_configured',
      '3': 1,
      '4': 1,
      '5': 8,
      '10': 'hedgeDelayConfigured'
    },
    {
      '1': 'hedge_delay_millis',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'hedgeDelayMillis'
    },
    {
      '1': 'route_preference',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointRoutePreference',
      '10': 'routePreference'
    },
  ],
};

/// Descriptor for `EndpointSelectionPolicy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSelectionPolicyDescriptor = $convert.base64Decode(
    'ChdFbmRwb2ludFNlbGVjdGlvblBvbGljeRI0ChZoZWRnZV9kZWxheV9jb25maWd1cmVkGAEgAS'
    'gIUhRoZWRnZURlbGF5Q29uZmlndXJlZBIsChJoZWRnZV9kZWxheV9taWxsaXMYAiABKARSEGhl'
    'ZGdlRGVsYXlNaWxsaXMSWQoQcm91dGVfcHJlZmVyZW5jZRgDIAEoDjIuLmFueXR0eS5yZW1vdG'
    'UuYXV0aC52MS5FbmRwb2ludFJvdXRlUHJlZmVyZW5jZVIPcm91dGVQcmVmZXJlbmNl');

@$core.Deprecated('Use endpointCredentialDescriptorDescriptor instead')
const EndpointCredentialDescriptor$json = {
  '1': 'EndpointCredentialDescriptor',
  '2': [
    {'1': 'descriptor_id', '3': 1, '4': 1, '5': 9, '10': 'descriptorId'},
    {
      '1': 'kind',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointCredentialKind',
      '10': 'kind'
    },
    {'1': 'exportable', '3': 3, '4': 1, '5': 8, '10': 'exportable'},
  ],
};

/// Descriptor for `EndpointCredentialDescriptor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointCredentialDescriptorDescriptor = $convert.base64Decode(
    'ChxFbmRwb2ludENyZWRlbnRpYWxEZXNjcmlwdG9yEiMKDWRlc2NyaXB0b3JfaWQYASABKAlSDG'
    'Rlc2NyaXB0b3JJZBJBCgRraW5kGAIgASgOMi0uYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVuZHBv'
    'aW50Q3JlZGVudGlhbEtpbmRSBGtpbmQSHgoKZXhwb3J0YWJsZRgDIAEoCFIKZXhwb3J0YWJsZQ'
    '==');

@$core.Deprecated('Use localUnixRouteConfigDescriptor instead')
const LocalUnixRouteConfig$json = {
  '1': 'LocalUnixRouteConfig',
  '2': [
    {'1': 'socket', '3': 1, '4': 1, '5': 9, '10': 'socket'},
  ],
};

/// Descriptor for `LocalUnixRouteConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List localUnixRouteConfigDescriptor =
    $convert.base64Decode(
        'ChRMb2NhbFVuaXhSb3V0ZUNvbmZpZxIWCgZzb2NrZXQYASABKAlSBnNvY2tldA==');

@$core.Deprecated('Use directWebRTCTCPRouteConfigDescriptor instead')
const DirectWebRTCTCPRouteConfig$json = {
  '1': 'DirectWebRTCTCPRouteConfig',
  '2': [
    {
      '1': 'signaling_addresses',
      '3': 1,
      '4': 3,
      '5': 9,
      '10': 'signalingAddresses'
    },
    {'1': 'ice_tcp_addresses', '3': 2, '4': 3, '5': 9, '10': 'iceTcpAddresses'},
    {
      '1': 'advertised_addresses',
      '3': 3,
      '4': 3,
      '5': 9,
      '10': 'advertisedAddresses'
    },
    {'1': 'server_name', '3': 4, '4': 1, '5': 9, '10': 'serverName'},
  ],
};

/// Descriptor for `DirectWebRTCTCPRouteConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List directWebRTCTCPRouteConfigDescriptor = $convert.base64Decode(
    'ChpEaXJlY3RXZWJSVENUQ1BSb3V0ZUNvbmZpZxIvChNzaWduYWxpbmdfYWRkcmVzc2VzGAEgAy'
    'gJUhJzaWduYWxpbmdBZGRyZXNzZXMSKgoRaWNlX3RjcF9hZGRyZXNzZXMYAiADKAlSD2ljZVRj'
    'cEFkZHJlc3NlcxIxChRhZHZlcnRpc2VkX2FkZHJlc3NlcxgDIAMoCVITYWR2ZXJ0aXNlZEFkZH'
    'Jlc3NlcxIfCgtzZXJ2ZXJfbmFtZRgEIAEoCVIKc2VydmVyTmFtZQ==');

@$core.Deprecated('Use sSHWebRTCTCPRouteConfigDescriptor instead')
const SSHWebRTCTCPRouteConfig$json = {
  '1': 'SSHWebRTCTCPRouteConfig',
  '2': [
    {'1': 'host', '3': 1, '4': 1, '5': 9, '10': 'host'},
    {'1': 'port', '3': 2, '4': 1, '5': 13, '10': 'port'},
    {'1': 'user', '3': 3, '4': 1, '5': 9, '10': 'user'},
    {
      '1': 'host_key_fingerprints',
      '3': 4,
      '4': 3,
      '5': 9,
      '10': 'hostKeyFingerprints'
    },
    {'1': 'proxy_jump', '3': 5, '4': 1, '5': 9, '10': 'proxyJump'},
    {
      '1': 'credential_descriptor',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointCredentialDescriptor',
      '10': 'credentialDescriptor'
    },
    {
      '1': 'remote_signaling_address',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'remoteSignalingAddress'
    },
    {
      '1': 'remote_ice_tcp_address',
      '3': 8,
      '4': 1,
      '5': 9,
      '10': 'remoteIceTcpAddress'
    },
    {
      '1': 'ssh_credential_ref',
      '3': 9,
      '4': 1,
      '5': 9,
      '10': 'sshCredentialRef'
    },
  ],
};

/// Descriptor for `SSHWebRTCTCPRouteConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHWebRTCTCPRouteConfigDescriptor = $convert.base64Decode(
    'ChdTU0hXZWJSVENUQ1BSb3V0ZUNvbmZpZxISCgRob3N0GAEgASgJUgRob3N0EhIKBHBvcnQYAi'
    'ABKA1SBHBvcnQSEgoEdXNlchgDIAEoCVIEdXNlchIyChVob3N0X2tleV9maW5nZXJwcmludHMY'
    'BCADKAlSE2hvc3RLZXlGaW5nZXJwcmludHMSHQoKcHJveHlfanVtcBgFIAEoCVIJcHJveHlKdW'
    '1wEmgKFWNyZWRlbnRpYWxfZGVzY3JpcHRvchgGIAEoCzIzLmFueXR0eS5yZW1vdGUuYXV0aC52'
    'MS5FbmRwb2ludENyZWRlbnRpYWxEZXNjcmlwdG9yUhRjcmVkZW50aWFsRGVzY3JpcHRvchI4Ch'
    'hyZW1vdGVfc2lnbmFsaW5nX2FkZHJlc3MYByABKAlSFnJlbW90ZVNpZ25hbGluZ0FkZHJlc3MS'
    'MwoWcmVtb3RlX2ljZV90Y3BfYWRkcmVzcxgIIAEoCVITcmVtb3RlSWNlVGNwQWRkcmVzcxIsCh'
    'Jzc2hfY3JlZGVudGlhbF9yZWYYCSABKAlSEHNzaENyZWRlbnRpYWxSZWY=');

@$core.Deprecated('Use managedWebRTCRouteConfigDescriptor instead')
const ManagedWebRTCRouteConfig$json = {
  '1': 'ManagedWebRTCRouteConfig',
  '2': [
    {'1': 'target_device_id', '3': 1, '4': 1, '5': 9, '10': 'targetDeviceId'},
    {
      '1': 'account_profile_ref',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'accountProfileRef'
    },
    {
      '1': 'relay_mode',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.ManagedWebRTCRelayMode',
      '10': 'relayMode'
    },
    {
      '1': 'relay_transport',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.ManagedWebRTCRelayTransport',
      '10': 'relayTransport'
    },
  ],
};

/// Descriptor for `ManagedWebRTCRouteConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List managedWebRTCRouteConfigDescriptor = $convert.base64Decode(
    'ChhNYW5hZ2VkV2ViUlRDUm91dGVDb25maWcSKAoQdGFyZ2V0X2RldmljZV9pZBgBIAEoCVIOdG'
    'FyZ2V0RGV2aWNlSWQSLgoTYWNjb3VudF9wcm9maWxlX3JlZhgCIAEoCVIRYWNjb3VudFByb2Zp'
    'bGVSZWYSTAoKcmVsYXlfbW9kZRgDIAEoDjItLmFueXR0eS5yZW1vdGUuYXV0aC52MS5NYW5hZ2'
    'VkV2ViUlRDUmVsYXlNb2RlUglyZWxheU1vZGUSWwoPcmVsYXlfdHJhbnNwb3J0GAQgASgOMjIu'
    'YW55dHR5LnJlbW90ZS5hdXRoLnYxLk1hbmFnZWRXZWJSVENSZWxheVRyYW5zcG9ydFIOcmVsYX'
    'lUcmFuc3BvcnQ=');

@$core.Deprecated('Use endpointRouteConfigV1Descriptor instead')
const EndpointRouteConfigV1$json = {
  '1': 'EndpointRouteConfigV1',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'route_id', '3': 2, '4': 1, '5': 9, '10': 'routeId'},
    {'1': 'enabled', '3': 3, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'manual_only', '3': 4, '4': 1, '5': 8, '10': 'manualOnly'},
    {
      '1': 'priority',
      '3': 5,
      '4': 1,
      '5': 5,
      '9': 1,
      '10': 'priority',
      '17': true
    },
    {'1': 'credential_ref', '3': 6, '4': 1, '5': 9, '10': 'credentialRef'},
    {
      '1': 'source',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointSource',
      '10': 'source'
    },
    {
      '1': 'policy_source',
      '3': 8,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointSource',
      '10': 'policySource'
    },
    {'1': 'display_name', '3': 9, '4': 1, '5': 9, '10': 'displayName'},
    {
      '1': 'local_unix',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.LocalUnixRouteConfig',
      '9': 0,
      '10': 'localUnix'
    },
    {
      '1': 'direct_webrtc_tcp',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.DirectWebRTCTCPRouteConfig',
      '9': 0,
      '10': 'directWebrtcTcp'
    },
    {
      '1': 'ssh_webrtc_tcp',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.SSHWebRTCTCPRouteConfig',
      '9': 0,
      '10': 'sshWebrtcTcp'
    },
    {
      '1': 'managed_webrtc',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ManagedWebRTCRouteConfig',
      '9': 0,
      '10': 'managedWebrtc'
    },
  ],
  '8': [
    {'1': 'route'},
    {'1': '_priority'},
  ],
};

/// Descriptor for `EndpointRouteConfigV1`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRouteConfigV1Descriptor = $convert.base64Decode(
    'ChVFbmRwb2ludFJvdXRlQ29uZmlnVjESJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDXNjaGVtYV'
    'ZlcnNpb24SGQoIcm91dGVfaWQYAiABKAlSB3JvdXRlSWQSGAoHZW5hYmxlZBgDIAEoCFIHZW5h'
    'YmxlZBIfCgttYW51YWxfb25seRgEIAEoCFIKbWFudWFsT25seRIfCghwcmlvcml0eRgFIAEoBU'
    'gBUghwcmlvcml0eYgBARIlCg5jcmVkZW50aWFsX3JlZhgGIAEoCVINY3JlZGVudGlhbFJlZhI9'
    'CgZzb3VyY2UYByABKA4yJS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRTb3VyY2VSBn'
    'NvdXJjZRJKCg1wb2xpY3lfc291cmNlGAggASgOMiUuYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVu'
    'ZHBvaW50U291cmNlUgxwb2xpY3lTb3VyY2USIQoMZGlzcGxheV9uYW1lGAkgASgJUgtkaXNwbG'
    'F5TmFtZRJMCgpsb2NhbF91bml4GBQgASgLMisuYW55dHR5LnJlbW90ZS5hdXRoLnYxLkxvY2Fs'
    'VW5peFJvdXRlQ29uZmlnSABSCWxvY2FsVW5peBJfChFkaXJlY3Rfd2VicnRjX3RjcBgVIAEoCz'
    'IxLmFueXR0eS5yZW1vdGUuYXV0aC52MS5EaXJlY3RXZWJSVENUQ1BSb3V0ZUNvbmZpZ0gAUg9k'
    'aXJlY3RXZWJydGNUY3ASVgoOc3NoX3dlYnJ0Y190Y3AYFiABKAsyLi5hbnl0dHkucmVtb3RlLm'
    'F1dGgudjEuU1NIV2ViUlRDVENQUm91dGVDb25maWdIAFIMc3NoV2VicnRjVGNwElgKDm1hbmFn'
    'ZWRfd2VicnRjGBcgASgLMi8uYW55dHR5LnJlbW90ZS5hdXRoLnYxLk1hbmFnZWRXZWJSVENSb3'
    'V0ZUNvbmZpZ0gAUg1tYW5hZ2VkV2VicnRjQgcKBXJvdXRlQgsKCV9wcmlvcml0eQ==');

@$core.Deprecated('Use pairingDirectRouteSeedDescriptor instead')
const PairingDirectRouteSeed$json = {
  '1': 'PairingDirectRouteSeed',
  '2': [
    {
      '1': 'signaling_address',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'signalingAddress'
    },
    {'1': 'ice_tcp_address', '3': 2, '4': 1, '5': 9, '10': 'iceTcpAddress'},
    {'1': 'server_name', '3': 3, '4': 1, '5': 9, '10': 'serverName'},
  ],
};

/// Descriptor for `PairingDirectRouteSeed`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingDirectRouteSeedDescriptor = $convert.base64Decode(
    'ChZQYWlyaW5nRGlyZWN0Um91dGVTZWVkEisKEXNpZ25hbGluZ19hZGRyZXNzGAEgASgJUhBzaW'
    'duYWxpbmdBZGRyZXNzEiYKD2ljZV90Y3BfYWRkcmVzcxgCIAEoCVINaWNlVGNwQWRkcmVzcxIf'
    'CgtzZXJ2ZXJfbmFtZRgDIAEoCVIKc2VydmVyTmFtZQ==');

@$core.Deprecated('Use pairingManagedRouteSeedDescriptor instead')
const PairingManagedRouteSeed$json = {
  '1': 'PairingManagedRouteSeed',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'edge_id', '3': 2, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'public_endpoint', '3': 3, '4': 1, '5': 9, '10': 'publicEndpoint'},
    {'1': 'server_name', '3': 4, '4': 1, '5': 9, '10': 'serverName'},
    {
      '1': 'ca_certificate_der_sha256',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'caCertificateDerSha256'
    },
  ],
};

/// Descriptor for `PairingManagedRouteSeed`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingManagedRouteSeedDescriptor = $convert.base64Decode(
    'ChdQYWlyaW5nTWFuYWdlZFJvdXRlU2VlZBIbCglkYWVtb25faWQYASABKAlSCGRhZW1vbklkEh'
    'cKB2VkZ2VfaWQYAiABKAlSBmVkZ2VJZBInCg9wdWJsaWNfZW5kcG9pbnQYAyABKAlSDnB1Ymxp'
    'Y0VuZHBvaW50Eh8KC3NlcnZlcl9uYW1lGAQgASgJUgpzZXJ2ZXJOYW1lEjkKGWNhX2NlcnRpZm'
    'ljYXRlX2Rlcl9zaGEyNTYYBSABKAxSFmNhQ2VydGlmaWNhdGVEZXJTaGEyNTY=');

@$core.Deprecated('Use pairingSSHRouteSeedDescriptor instead')
const PairingSSHRouteSeed$json = {
  '1': 'PairingSSHRouteSeed',
  '2': [
    {'1': 'host', '3': 1, '4': 1, '5': 9, '10': 'host'},
    {'1': 'port', '3': 2, '4': 1, '5': 13, '10': 'port'},
    {'1': 'user', '3': 3, '4': 1, '5': 9, '10': 'user'},
    {
      '1': 'host_key_fingerprints',
      '3': 4,
      '4': 3,
      '5': 9,
      '10': 'hostKeyFingerprints'
    },
    {'1': 'proxy_jump', '3': 5, '4': 1, '5': 9, '10': 'proxyJump'},
    {
      '1': 'remote_signaling_address',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'remoteSignalingAddress'
    },
    {
      '1': 'remote_ice_tcp_address',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'remoteIceTcpAddress'
    },
    {
      '1': 'credential_kind',
      '3': 8,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointCredentialKind',
      '10': 'credentialKind'
    },
  ],
};

/// Descriptor for `PairingSSHRouteSeed`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingSSHRouteSeedDescriptor = $convert.base64Decode(
    'ChNQYWlyaW5nU1NIUm91dGVTZWVkEhIKBGhvc3QYASABKAlSBGhvc3QSEgoEcG9ydBgCIAEoDV'
    'IEcG9ydBISCgR1c2VyGAMgASgJUgR1c2VyEjIKFWhvc3Rfa2V5X2ZpbmdlcnByaW50cxgEIAMo'
    'CVITaG9zdEtleUZpbmdlcnByaW50cxIdCgpwcm94eV9qdW1wGAUgASgJUglwcm94eUp1bXASOA'
    'oYcmVtb3RlX3NpZ25hbGluZ19hZGRyZXNzGAYgASgJUhZyZW1vdGVTaWduYWxpbmdBZGRyZXNz'
    'EjMKFnJlbW90ZV9pY2VfdGNwX2FkZHJlc3MYByABKAlSE3JlbW90ZUljZVRjcEFkZHJlc3MSVg'
    'oPY3JlZGVudGlhbF9raW5kGAggASgOMi0uYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVuZHBvaW50'
    'Q3JlZGVudGlhbEtpbmRSDmNyZWRlbnRpYWxLaW5k');

@$core.Deprecated('Use pairingRouteSeedDescriptor instead')
const PairingRouteSeed$json = {
  '1': 'PairingRouteSeed',
  '2': [
    {
      '1': 'direct_webrtc_tcp',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingDirectRouteSeed',
      '9': 0,
      '10': 'directWebrtcTcp'
    },
    {
      '1': 'managed_webrtc',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingManagedRouteSeed',
      '9': 0,
      '10': 'managedWebrtc'
    },
    {
      '1': 'ssh_webrtc_tcp',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingSSHRouteSeed',
      '9': 0,
      '10': 'sshWebrtcTcp'
    },
    {'1': 'route_id', '3': 4, '4': 1, '5': 9, '10': 'routeId'},
    {'1': 'display_name', '3': 5, '4': 1, '5': 9, '10': 'displayName'},
    {
      '1': 'priority',
      '3': 6,
      '4': 1,
      '5': 5,
      '9': 1,
      '10': 'priority',
      '17': true
    },
  ],
  '8': [
    {'1': 'route'},
    {'1': '_priority'},
  ],
};

/// Descriptor for `PairingRouteSeed`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingRouteSeedDescriptor = $convert.base64Decode(
    'ChBQYWlyaW5nUm91dGVTZWVkElsKEWRpcmVjdF93ZWJydGNfdGNwGAEgASgLMi0uYW55dHR5Ln'
    'JlbW90ZS5hdXRoLnYxLlBhaXJpbmdEaXJlY3RSb3V0ZVNlZWRIAFIPZGlyZWN0V2VicnRjVGNw'
    'ElcKDm1hbmFnZWRfd2VicnRjGAIgASgLMi4uYW55dHR5LnJlbW90ZS5hdXRoLnYxLlBhaXJpbm'
    'dNYW5hZ2VkUm91dGVTZWVkSABSDW1hbmFnZWRXZWJydGMSUgoOc3NoX3dlYnJ0Y190Y3AYAyAB'
    'KAsyKi5hbnl0dHkucmVtb3RlLmF1dGgudjEuUGFpcmluZ1NTSFJvdXRlU2VlZEgAUgxzc2hXZW'
    'JydGNUY3ASGQoIcm91dGVfaWQYBCABKAlSB3JvdXRlSWQSIQoMZGlzcGxheV9uYW1lGAUgASgJ'
    'UgtkaXNwbGF5TmFtZRIfCghwcmlvcml0eRgGIAEoBUgBUghwcmlvcml0eYgBAUIHCgVyb3V0ZU'
    'ILCglfcHJpb3JpdHk=');

@$core.Deprecated('Use pairingClaimOfferDescriptor instead')
const PairingClaimOffer$json = {
  '1': 'PairingClaimOffer',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'claim', '3': 2, '4': 1, '5': 12, '10': 'claim'},
    {'1': 'device_id', '3': 3, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'devicePublicKey'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {
      '1': 'routes',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingRouteSeed',
      '10': 'routes'
    },
  ],
};

/// Descriptor for `PairingClaimOffer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingClaimOfferDescriptor = $convert.base64Decode(
    'ChFQYWlyaW5nQ2xhaW1PZmZlchIlCg5zY2hlbWFfdmVyc2lvbhgBIAEoDVINc2NoZW1hVmVyc2'
    'lvbhIUCgVjbGFpbRgCIAEoDFIFY2xhaW0SGwoJZGV2aWNlX2lkGAMgASgJUghkZXZpY2VJZBIq'
    'ChFkZXZpY2VfcHVibGljX2tleRgEIAEoDFIPZGV2aWNlUHVibGljS2V5Ei8KFGV4cGlyZXNfYX'
    'RfdW5peF9uYW5vGAUgASgDUhFleHBpcmVzQXRVbml4TmFubxI/CgZyb3V0ZXMYBiADKAsyJy5h'
    'bnl0dHkucmVtb3RlLmF1dGgudjEuUGFpcmluZ1JvdXRlU2VlZFIGcm91dGVz');

@$core.Deprecated('Use endpointConfigV1Descriptor instead')
const EndpointConfigV1$json = {
  '1': 'EndpointConfigV1',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'label', '3': 3, '4': 1, '5': 9, '10': 'label'},
    {
      '1': 'label_source',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointSource',
      '10': 'labelSource'
    },
    {
      '1': 'identity',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointDaemonIdentity',
      '10': 'identity'
    },
    {
      '1': 'connect_mode',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointConnectMode',
      '10': 'connectMode'
    },
    {'1': 'enabled', '3': 7, '4': 1, '5': 8, '10': 'enabled'},
    {
      '1': 'selection_policy',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointSelectionPolicy',
      '10': 'selectionPolicy'
    },
    {
      '1': 'routes',
      '3': 9,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRouteConfigV1',
      '10': 'routes'
    },
    {'1': 'platform', '3': 10, '4': 1, '5': 9, '10': 'platform'},
  ],
};

/// Descriptor for `EndpointConfigV1`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointConfigV1Descriptor = $convert.base64Decode(
    'ChBFbmRwb2ludENvbmZpZ1YxEiUKDnNjaGVtYV92ZXJzaW9uGAEgASgNUg1zY2hlbWFWZXJzaW'
    '9uEh8KC2VuZHBvaW50X2lkGAIgASgJUgplbmRwb2ludElkEhQKBWxhYmVsGAMgASgJUgVsYWJl'
    'bBJICgxsYWJlbF9zb3VyY2UYBCABKA4yJS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbn'
    'RTb3VyY2VSC2xhYmVsU291cmNlEkkKCGlkZW50aXR5GAUgASgLMi0uYW55dHR5LnJlbW90ZS5h'
    'dXRoLnYxLkVuZHBvaW50RGFlbW9uSWRlbnRpdHlSCGlkZW50aXR5Ek0KDGNvbm5lY3RfbW9kZR'
    'gGIAEoDjIqLmFueXR0eS5yZW1vdGUuYXV0aC52MS5FbmRwb2ludENvbm5lY3RNb2RlUgtjb25u'
    'ZWN0TW9kZRIYCgdlbmFibGVkGAcgASgIUgdlbmFibGVkElkKEHNlbGVjdGlvbl9wb2xpY3kYCC'
    'ABKAsyLi5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRTZWxlY3Rpb25Qb2xpY3lSD3Nl'
    'bGVjdGlvblBvbGljeRJECgZyb3V0ZXMYCSADKAsyLC5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW'
    '5kcG9pbnRSb3V0ZUNvbmZpZ1YxUgZyb3V0ZXMSGgoIcGxhdGZvcm0YCiABKAlSCHBsYXRmb3Jt');

@$core.Deprecated('Use endpointRegistryV1Descriptor instead')
const EndpointRegistryV1$json = {
  '1': 'EndpointRegistryV1',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {
      '1': 'default_endpoint_id',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'defaultEndpointId'
    },
    {
      '1': 'endpoints',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointConfigV1',
      '10': 'endpoints'
    },
  ],
};

/// Descriptor for `EndpointRegistryV1`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRegistryV1Descriptor = $convert.base64Decode(
    'ChJFbmRwb2ludFJlZ2lzdHJ5VjESJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDXNjaGVtYVZlcn'
    'Npb24SLgoTZGVmYXVsdF9lbmRwb2ludF9pZBgCIAEoCVIRZGVmYXVsdEVuZHBvaW50SWQSRQoJ'
    'ZW5kcG9pbnRzGAMgAygLMicuYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVuZHBvaW50Q29uZmlnVj'
    'FSCWVuZHBvaW50cw==');

@$core.Deprecated('Use pairingTicketDescriptorDescriptor instead')
const PairingTicketDescriptor$json = {
  '1': 'PairingTicketDescriptor',
  '2': [
    {'1': 'ticket_id', '3': 1, '4': 1, '5': 9, '10': 'ticketId'},
    {'1': 'scope_ceiling', '3': 2, '4': 3, '5': 9, '10': 'scopeCeiling'},
    {
      '1': 'expires_at_unix_nano',
      '3': 3,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'nonce', '3': 4, '4': 1, '5': 12, '10': 'nonce'},
    {'1': 'max_redemptions', '3': 5, '4': 1, '5': 13, '10': 'maxRedemptions'},
    {'1': 'signature', '3': 6, '4': 1, '5': 12, '10': 'signature'},
    {
      '1': 'issued_at_unix_nano',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {
      '1': 'grant_lifetime_seconds',
      '3': 8,
      '4': 1,
      '5': 3,
      '10': 'grantLifetimeSeconds'
    },
  ],
};

/// Descriptor for `PairingTicketDescriptor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingTicketDescriptorDescriptor = $convert.base64Decode(
    'ChdQYWlyaW5nVGlja2V0RGVzY3JpcHRvchIbCgl0aWNrZXRfaWQYASABKAlSCHRpY2tldElkEi'
    'MKDXNjb3BlX2NlaWxpbmcYAiADKAlSDHNjb3BlQ2VpbGluZxIvChRleHBpcmVzX2F0X3VuaXhf'
    'bmFubxgDIAEoA1IRZXhwaXJlc0F0VW5peE5hbm8SFAoFbm9uY2UYBCABKAxSBW5vbmNlEicKD2'
    '1heF9yZWRlbXB0aW9ucxgFIAEoDVIObWF4UmVkZW1wdGlvbnMSHAoJc2lnbmF0dXJlGAYgASgM'
    'UglzaWduYXR1cmUSLQoTaXNzdWVkX2F0X3VuaXhfbmFubxgHIAEoA1IQaXNzdWVkQXRVbml4Tm'
    'FubxI0ChZncmFudF9saWZldGltZV9zZWNvbmRzGAggASgDUhRncmFudExpZmV0aW1lU2Vjb25k'
    'cw==');

@$core.Deprecated('Use endpointAuthorizationBootstrapDescriptor instead')
const EndpointAuthorizationBootstrap$json = {
  '1': 'EndpointAuthorizationBootstrap',
  '2': [
    {
      '1': 'pairing_ticket',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingTicketDescriptor',
      '9': 0,
      '10': 'pairingTicket'
    },
    {'1': 'bound_grant', '3': 2, '4': 1, '5': 12, '9': 0, '10': 'boundGrant'},
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `EndpointAuthorizationBootstrap`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointAuthorizationBootstrapDescriptor =
    $convert.base64Decode(
        'Ch5FbmRwb2ludEF1dGhvcml6YXRpb25Cb290c3RyYXASVwoOcGFpcmluZ190aWNrZXQYASABKA'
        'syLi5hbnl0dHkucmVtb3RlLmF1dGgudjEuUGFpcmluZ1RpY2tldERlc2NyaXB0b3JIAFINcGFp'
        'cmluZ1RpY2tldBIhCgtib3VuZF9ncmFudBgCIAEoDEgAUgpib3VuZEdyYW50QgkKB3BheWxvYW'
        'Q=');

@$core.Deprecated('Use endpointBootstrapBundleV2Descriptor instead')
const EndpointBootstrapBundleV2$json = {
  '1': 'EndpointBootstrapBundleV2',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'bundle_id', '3': 2, '4': 1, '5': 9, '10': 'bundleId'},
    {
      '1': 'identity',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointDaemonIdentity',
      '10': 'identity'
    },
    {'1': 'suggested_label', '3': 4, '4': 1, '5': 9, '10': 'suggestedLabel'},
    {
      '1': 'routes',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRouteConfigV1',
      '10': 'routes'
    },
    {
      '1': 'authorization',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointAuthorizationBootstrap',
      '10': 'authorization'
    },
    {
      '1': 'issued_at_unix_nano',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 8,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'bundle_signature', '3': 9, '4': 1, '5': 12, '10': 'bundleSignature'},
    {'1': 'platform', '3': 10, '4': 1, '5': 9, '10': 'platform'},
  ],
};

/// Descriptor for `EndpointBootstrapBundleV2`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointBootstrapBundleV2Descriptor = $convert.base64Decode(
    'ChlFbmRwb2ludEJvb3RzdHJhcEJ1bmRsZVYyEiUKDnNjaGVtYV92ZXJzaW9uGAEgASgNUg1zY2'
    'hlbWFWZXJzaW9uEhsKCWJ1bmRsZV9pZBgCIAEoCVIIYnVuZGxlSWQSSQoIaWRlbnRpdHkYAyAB'
    'KAsyLS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnREYWVtb25JZGVudGl0eVIIaWRlbn'
    'RpdHkSJwoPc3VnZ2VzdGVkX2xhYmVsGAQgASgJUg5zdWdnZXN0ZWRMYWJlbBJECgZyb3V0ZXMY'
    'BSADKAsyLC5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRSb3V0ZUNvbmZpZ1YxUgZyb3'
    'V0ZXMSWwoNYXV0aG9yaXphdGlvbhgGIAEoCzI1LmFueXR0eS5yZW1vdGUuYXV0aC52MS5FbmRw'
    'b2ludEF1dGhvcml6YXRpb25Cb290c3RyYXBSDWF1dGhvcml6YXRpb24SLQoTaXNzdWVkX2F0X3'
    'VuaXhfbmFubxgHIAEoA1IQaXNzdWVkQXRVbml4TmFubxIvChRleHBpcmVzX2F0X3VuaXhfbmFu'
    'bxgIIAEoA1IRZXhwaXJlc0F0VW5peE5hbm8SKQoQYnVuZGxlX3NpZ25hdHVyZRgJIAEoDFIPYn'
    'VuZGxlU2lnbmF0dXJlEhoKCHBsYXRmb3JtGAogASgJUghwbGF0Zm9ybQ==');

@$core.Deprecated('Use pairingTicketSignatureInputDescriptor instead')
const PairingTicketSignatureInput$json = {
  '1': 'PairingTicketSignatureInput',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {'1': 'issuer_device_id', '3': 3, '4': 1, '5': 9, '10': 'issuerDeviceId'},
    {
      '1': 'issuer_device_fingerprint',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'issuerDeviceFingerprint'
    },
    {
      '1': 'ticket',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.PairingTicketDescriptor',
      '10': 'ticket'
    },
  ],
};

/// Descriptor for `PairingTicketSignatureInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingTicketSignatureInputDescriptor = $convert.base64Decode(
    'ChtQYWlyaW5nVGlja2V0U2lnbmF0dXJlSW5wdXQSGgoIcHJvdG9jb2wYASABKAlSCHByb3RvY2'
    '9sEhgKB3ZlcnNpb24YAiABKA1SB3ZlcnNpb24SKAoQaXNzdWVyX2RldmljZV9pZBgDIAEoCVIO'
    'aXNzdWVyRGV2aWNlSWQSOgoZaXNzdWVyX2RldmljZV9maW5nZXJwcmludBgEIAEoCVIXaXNzdW'
    'VyRGV2aWNlRmluZ2VycHJpbnQSRgoGdGlja2V0GAUgASgLMi4uYW55dHR5LnJlbW90ZS5hdXRo'
    'LnYxLlBhaXJpbmdUaWNrZXREZXNjcmlwdG9yUgZ0aWNrZXQ=');

@$core.Deprecated('Use endpointBootstrapSignatureInputDescriptor instead')
const EndpointBootstrapSignatureInput$json = {
  '1': 'EndpointBootstrapSignatureInput',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {
      '1': 'bundle',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointBootstrapBundleV2',
      '10': 'bundle'
    },
  ],
};

/// Descriptor for `EndpointBootstrapSignatureInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointBootstrapSignatureInputDescriptor =
    $convert.base64Decode(
        'Ch9FbmRwb2ludEJvb3RzdHJhcFNpZ25hdHVyZUlucHV0EhoKCHByb3RvY29sGAEgASgJUghwcm'
        '90b2NvbBIYCgd2ZXJzaW9uGAIgASgNUgd2ZXJzaW9uEkgKBmJ1bmRsZRgDIAEoCzIwLmFueXR0'
        'eS5yZW1vdGUuYXV0aC52MS5FbmRwb2ludEJvb3RzdHJhcEJ1bmRsZVYyUgZidW5kbGU=');

@$core.Deprecated('Use clientEndpointShareBundleV1Descriptor instead')
const ClientEndpointShareBundleV1$json = {
  '1': 'ClientEndpointShareBundleV1',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'transfer_id', '3': 2, '4': 1, '5': 9, '10': 'transferId'},
    {
      '1': 'identity',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointDaemonIdentity',
      '10': 'identity'
    },
    {'1': 'suggested_label', '3': 4, '4': 1, '5': 9, '10': 'suggestedLabel'},
    {
      '1': 'routes',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRouteConfigV1',
      '10': 'routes'
    },
    {
      '1': 'connect_mode',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointConnectMode',
      '10': 'connectMode'
    },
    {
      '1': 'selection_policy',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointSelectionPolicy',
      '10': 'selectionPolicy'
    },
    {
      '1': 'credential_descriptors',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointCredentialDescriptor',
      '10': 'credentialDescriptors'
    },
    {'1': 'bound_grant', '3': 9, '4': 1, '5': 12, '10': 'boundGrant'},
    {
      '1': 'issued_at_unix_nano',
      '3': 10,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixNano'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 11,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'platform', '3': 12, '4': 1, '5': 9, '10': 'platform'},
  ],
};

/// Descriptor for `ClientEndpointShareBundleV1`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientEndpointShareBundleV1Descriptor = $convert.base64Decode(
    'ChtDbGllbnRFbmRwb2ludFNoYXJlQnVuZGxlVjESJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDX'
    'NjaGVtYVZlcnNpb24SHwoLdHJhbnNmZXJfaWQYAiABKAlSCnRyYW5zZmVySWQSSQoIaWRlbnRp'
    'dHkYAyABKAsyLS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnREYWVtb25JZGVudGl0eV'
    'IIaWRlbnRpdHkSJwoPc3VnZ2VzdGVkX2xhYmVsGAQgASgJUg5zdWdnZXN0ZWRMYWJlbBJECgZy'
    'b3V0ZXMYBSADKAsyLC5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRSb3V0ZUNvbmZpZ1'
    'YxUgZyb3V0ZXMSTQoMY29ubmVjdF9tb2RlGAYgASgOMiouYW55dHR5LnJlbW90ZS5hdXRoLnYx'
    'LkVuZHBvaW50Q29ubmVjdE1vZGVSC2Nvbm5lY3RNb2RlElkKEHNlbGVjdGlvbl9wb2xpY3kYBy'
    'ABKAsyLi5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRTZWxlY3Rpb25Qb2xpY3lSD3Nl'
    'bGVjdGlvblBvbGljeRJqChZjcmVkZW50aWFsX2Rlc2NyaXB0b3JzGAggAygLMjMuYW55dHR5Ln'
    'JlbW90ZS5hdXRoLnYxLkVuZHBvaW50Q3JlZGVudGlhbERlc2NyaXB0b3JSFWNyZWRlbnRpYWxE'
    'ZXNjcmlwdG9ycxIfCgtib3VuZF9ncmFudBgJIAEoDFIKYm91bmRHcmFudBItChNpc3N1ZWRfYX'
    'RfdW5peF9uYW5vGAogASgDUhBpc3N1ZWRBdFVuaXhOYW5vEi8KFGV4cGlyZXNfYXRfdW5peF9u'
    'YW5vGAsgASgDUhFleHBpcmVzQXRVbml4TmFubxIaCghwbGF0Zm9ybRgMIAEoCVIIcGxhdGZvcm'
    '0=');

@$core.Deprecated('Use shareSessionOfferDescriptor instead')
const ShareSessionOffer$json = {
  '1': 'ShareSessionOffer',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'transfer_id', '3': 2, '4': 1, '5': 9, '10': 'transferId'},
    {
      '1': 'listener_addresses',
      '3': 3,
      '4': 3,
      '5': 9,
      '10': 'listenerAddresses'
    },
    {
      '1': 'ephemeral_certificate_sha256',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'ephemeralCertificateSha256'
    },
    {
      '1': 'one_time_session_secret',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'oneTimeSessionSecret'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
  ],
};

/// Descriptor for `ShareSessionOffer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareSessionOfferDescriptor = $convert.base64Decode(
    'ChFTaGFyZVNlc3Npb25PZmZlchIlCg5zY2hlbWFfdmVyc2lvbhgBIAEoDVINc2NoZW1hVmVyc2'
    'lvbhIfCgt0cmFuc2Zlcl9pZBgCIAEoCVIKdHJhbnNmZXJJZBItChJsaXN0ZW5lcl9hZGRyZXNz'
    'ZXMYAyADKAlSEWxpc3RlbmVyQWRkcmVzc2VzEkAKHGVwaGVtZXJhbF9jZXJ0aWZpY2F0ZV9zaG'
    'EyNTYYBCABKAlSGmVwaGVtZXJhbENlcnRpZmljYXRlU2hhMjU2EjUKF29uZV90aW1lX3Nlc3Np'
    'b25fc2VjcmV0GAUgASgMUhRvbmVUaW1lU2Vzc2lvblNlY3JldBIvChRleHBpcmVzX2F0X3VuaX'
    'hfbmFubxgGIAEoA1IRZXhwaXJlc0F0VW5peE5hbm8=');

@$core.Deprecated('Use shareReceiverHelloDescriptor instead')
const ShareReceiverHello$json = {
  '1': 'ShareReceiverHello',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'transfer_id', '3': 2, '4': 1, '5': 9, '10': 'transferId'},
    {
      '1': 'one_time_session_secret',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'oneTimeSessionSecret'
    },
    {
      '1': 'receiver_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'receiverPublicKey'
    },
    {'1': 'receiver_nonce', '3': 5, '4': 1, '5': 12, '10': 'receiverNonce'},
  ],
};

/// Descriptor for `ShareReceiverHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareReceiverHelloDescriptor = $convert.base64Decode(
    'ChJTaGFyZVJlY2VpdmVySGVsbG8SJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDXNjaGVtYVZlcn'
    'Npb24SHwoLdHJhbnNmZXJfaWQYAiABKAlSCnRyYW5zZmVySWQSNQoXb25lX3RpbWVfc2Vzc2lv'
    'bl9zZWNyZXQYAyABKAxSFG9uZVRpbWVTZXNzaW9uU2VjcmV0Ei4KE3JlY2VpdmVyX3B1YmxpY1'
    '9rZXkYBCABKAxSEXJlY2VpdmVyUHVibGljS2V5EiUKDnJlY2VpdmVyX25vbmNlGAUgASgMUg1y'
    'ZWNlaXZlck5vbmNl');

@$core.Deprecated('Use shareSenderChallengeDescriptor instead')
const ShareSenderChallenge$json = {
  '1': 'ShareSenderChallenge',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'transfer_id', '3': 2, '4': 1, '5': 9, '10': 'transferId'},
    {'1': 'receiver_nonce', '3': 3, '4': 1, '5': 12, '10': 'receiverNonce'},
    {'1': 'sender_nonce', '3': 4, '4': 1, '5': 12, '10': 'senderNonce'},
    {
      '1': 'expires_at_unix_nano',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
  ],
};

/// Descriptor for `ShareSenderChallenge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareSenderChallengeDescriptor = $convert.base64Decode(
    'ChRTaGFyZVNlbmRlckNoYWxsZW5nZRIlCg5zY2hlbWFfdmVyc2lvbhgBIAEoDVINc2NoZW1hVm'
    'Vyc2lvbhIfCgt0cmFuc2Zlcl9pZBgCIAEoCVIKdHJhbnNmZXJJZBIlCg5yZWNlaXZlcl9ub25j'
    'ZRgDIAEoDFINcmVjZWl2ZXJOb25jZRIhCgxzZW5kZXJfbm9uY2UYBCABKAxSC3NlbmRlck5vbm'
    'NlEi8KFGV4cGlyZXNfYXRfdW5peF9uYW5vGAUgASgDUhFleHBpcmVzQXRVbml4TmFubw==');

@$core.Deprecated('Use shareReceiverProofInputDescriptor instead')
const ShareReceiverProofInput$json = {
  '1': 'ShareReceiverProofInput',
  '2': [
    {'1': 'protocol', '3': 1, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {'1': 'transfer_id', '3': 3, '4': 1, '5': 9, '10': 'transferId'},
    {'1': 'receiver_nonce', '3': 4, '4': 1, '5': 12, '10': 'receiverNonce'},
    {'1': 'sender_nonce', '3': 5, '4': 1, '5': 12, '10': 'senderNonce'},
    {
      '1': 'ephemeral_certificate_sha256',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'ephemeralCertificateSha256'
    },
    {
      '1': 'one_time_session_secret_sha256',
      '3': 7,
      '4': 1,
      '5': 12,
      '10': 'oneTimeSessionSecretSha256'
    },
  ],
};

/// Descriptor for `ShareReceiverProofInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareReceiverProofInputDescriptor = $convert.base64Decode(
    'ChdTaGFyZVJlY2VpdmVyUHJvb2ZJbnB1dBIaCghwcm90b2NvbBgBIAEoCVIIcHJvdG9jb2wSGA'
    'oHdmVyc2lvbhgCIAEoDVIHdmVyc2lvbhIfCgt0cmFuc2Zlcl9pZBgDIAEoCVIKdHJhbnNmZXJJ'
    'ZBIlCg5yZWNlaXZlcl9ub25jZRgEIAEoDFINcmVjZWl2ZXJOb25jZRIhCgxzZW5kZXJfbm9uY2'
    'UYBSABKAxSC3NlbmRlck5vbmNlEkAKHGVwaGVtZXJhbF9jZXJ0aWZpY2F0ZV9zaGEyNTYYBiAB'
    'KAlSGmVwaGVtZXJhbENlcnRpZmljYXRlU2hhMjU2EkIKHm9uZV90aW1lX3Nlc3Npb25fc2Vjcm'
    'V0X3NoYTI1NhgHIAEoDFIab25lVGltZVNlc3Npb25TZWNyZXRTaGEyNTY=');

@$core.Deprecated('Use shareReceiverProofDescriptor instead')
const ShareReceiverProof$json = {
  '1': 'ShareReceiverProof',
  '2': [
    {'1': 'schema_version', '3': 1, '4': 1, '5': 13, '10': 'schemaVersion'},
    {'1': 'transfer_id', '3': 2, '4': 1, '5': 9, '10': 'transferId'},
    {'1': 'signature', '3': 3, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `ShareReceiverProof`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareReceiverProofDescriptor = $convert.base64Decode(
    'ChJTaGFyZVJlY2VpdmVyUHJvb2YSJQoOc2NoZW1hX3ZlcnNpb24YASABKA1SDXNjaGVtYVZlcn'
    'Npb24SHwoLdHJhbnNmZXJfaWQYAiABKAlSCnRyYW5zZmVySWQSHAoJc2lnbmF0dXJlGAMgASgM'
    'UglzaWduYXR1cmU=');

@$core.Deprecated('Use shareSessionClientEnvelopeDescriptor instead')
const ShareSessionClientEnvelope$json = {
  '1': 'ShareSessionClientEnvelope',
  '2': [
    {
      '1': 'hello',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ShareReceiverHello',
      '9': 0,
      '10': 'hello'
    },
    {
      '1': 'proof',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ShareReceiverProof',
      '9': 0,
      '10': 'proof'
    },
  ],
  '8': [
    {'1': 'message'},
  ],
};

/// Descriptor for `ShareSessionClientEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareSessionClientEnvelopeDescriptor = $convert.base64Decode(
    'ChpTaGFyZVNlc3Npb25DbGllbnRFbnZlbG9wZRJBCgVoZWxsbxgBIAEoCzIpLmFueXR0eS5yZW'
    '1vdGUuYXV0aC52MS5TaGFyZVJlY2VpdmVySGVsbG9IAFIFaGVsbG8SQQoFcHJvb2YYAiABKAsy'
    'KS5hbnl0dHkucmVtb3RlLmF1dGgudjEuU2hhcmVSZWNlaXZlclByb29mSABSBXByb29mQgkKB2'
    '1lc3NhZ2U=');

@$core.Deprecated('Use shareSessionErrorDescriptor instead')
const ShareSessionError$json = {
  '1': 'ShareSessionError',
  '2': [
    {'1': 'code', '3': 1, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ShareSessionError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareSessionErrorDescriptor = $convert.base64Decode(
    'ChFTaGFyZVNlc3Npb25FcnJvchISCgRjb2RlGAEgASgJUgRjb2RlEhgKB21lc3NhZ2UYAiABKA'
    'lSB21lc3NhZ2U=');

@$core.Deprecated('Use shareSessionServerEnvelopeDescriptor instead')
const ShareSessionServerEnvelope$json = {
  '1': 'ShareSessionServerEnvelope',
  '2': [
    {
      '1': 'challenge',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ShareSenderChallenge',
      '9': 0,
      '10': 'challenge'
    },
    {
      '1': 'bundle',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ClientEndpointShareBundleV1',
      '9': 0,
      '10': 'bundle'
    },
    {
      '1': 'error',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.ShareSessionError',
      '9': 0,
      '10': 'error'
    },
  ],
  '8': [
    {'1': 'message'},
  ],
};

/// Descriptor for `ShareSessionServerEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List shareSessionServerEnvelopeDescriptor = $convert.base64Decode(
    'ChpTaGFyZVNlc3Npb25TZXJ2ZXJFbnZlbG9wZRJLCgljaGFsbGVuZ2UYASABKAsyKy5hbnl0dH'
    'kucmVtb3RlLmF1dGgudjEuU2hhcmVTZW5kZXJDaGFsbGVuZ2VIAFIJY2hhbGxlbmdlEkwKBmJ1'
    'bmRsZRgCIAEoCzIyLmFueXR0eS5yZW1vdGUuYXV0aC52MS5DbGllbnRFbmRwb2ludFNoYXJlQn'
    'VuZGxlVjFIAFIGYnVuZGxlEkAKBWVycm9yGAMgASgLMiguYW55dHR5LnJlbW90ZS5hdXRoLnYx'
    'LlNoYXJlU2Vzc2lvbkVycm9ySABSBWVycm9yQgkKB21lc3NhZ2U=');
