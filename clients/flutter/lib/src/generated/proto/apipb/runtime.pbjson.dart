// This is a generated file - do not edit.
//
// Generated from apipb/runtime.proto.

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

@$core.Deprecated('Use endpointRuntimePhaseDescriptor instead')
const EndpointRuntimePhase$json = {
  '1': 'EndpointRuntimePhase',
  '2': [
    {'1': 'ENDPOINT_RUNTIME_PHASE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_RUNTIME_PHASE_CONNECTING', '2': 1},
    {'1': 'ENDPOINT_RUNTIME_PHASE_READY', '2': 2},
    {'1': 'ENDPOINT_RUNTIME_PHASE_FAILED', '2': 3},
    {'1': 'ENDPOINT_RUNTIME_PHASE_CLOSED', '2': 4},
  ],
};

/// Descriptor for `EndpointRuntimePhase`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointRuntimePhaseDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludFJ1bnRpbWVQaGFzZRImCiJFTkRQT0lOVF9SVU5USU1FX1BIQVNFX1VOU1BFQ0'
    'lGSUVEEAASJQohRU5EUE9JTlRfUlVOVElNRV9QSEFTRV9DT05ORUNUSU5HEAESIAocRU5EUE9J'
    'TlRfUlVOVElNRV9QSEFTRV9SRUFEWRACEiEKHUVORFBPSU5UX1JVTlRJTUVfUEhBU0VfRkFJTE'
    'VEEAMSIQodRU5EUE9JTlRfUlVOVElNRV9QSEFTRV9DTE9TRUQQBA==');

@$core.Deprecated('Use endpointProbeRequestDescriptor instead')
const EndpointProbeRequest$json = {
  '1': 'EndpointProbeRequest',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'route_override', '3': 2, '4': 1, '5': 9, '10': 'routeOverride'},
  ],
};

/// Descriptor for `EndpointProbeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointProbeRequestDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludFByb2JlUmVxdWVzdBIfCgtlbmRwb2ludF9pZBgBIAEoCVIKZW5kcG9pbnRJZB'
    'IlCg5yb3V0ZV9vdmVycmlkZRgCIAEoCVINcm91dGVPdmVycmlkZQ==');

@$core.Deprecated('Use endpointProbeResultDescriptor instead')
const EndpointProbeResult$json = {
  '1': 'EndpointProbeResult',
  '2': [
    {
      '1': 'session',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {'1': 'observed_path', '3': 2, '4': 1, '5': 9, '10': 'observedPath'},
    {
      '1': 'route_selection_reason',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'routeSelectionReason'
    },
  ],
};

/// Descriptor for `EndpointProbeResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointProbeResultDescriptor = $convert.base64Decode(
    'ChNFbmRwb2ludFByb2JlUmVzdWx0Ej0KB3Nlc3Npb24YASABKAsyIy5hbnl0dHkuYXBpLnYxLk'
    'VuZHBvaW50U2Vzc2lvblN0YW1wUgdzZXNzaW9uEiMKDW9ic2VydmVkX3BhdGgYAiABKAlSDG9i'
    'c2VydmVkUGF0aBI0ChZyb3V0ZV9zZWxlY3Rpb25fcmVhc29uGAMgASgJUhRyb3V0ZVNlbGVjdG'
    'lvblJlYXNvbg==');

@$core.Deprecated('Use endpointRuntimeEventDescriptor instead')
const EndpointRuntimeEvent$json = {
  '1': 'EndpointRuntimeEvent',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {
      '1': 'phase',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.EndpointRuntimePhase',
      '10': 'phase'
    },
    {
      '1': 'session',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {
      '1': 'error',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `EndpointRuntimeEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRuntimeEventDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludFJ1bnRpbWVFdmVudBIfCgtlbmRwb2ludF9pZBgBIAEoCVIKZW5kcG9pbnRJZB'
    'I5CgVwaGFzZRgCIAEoDjIjLmFueXR0eS5hcGkudjEuRW5kcG9pbnRSdW50aW1lUGhhc2VSBXBo'
    'YXNlEj0KB3Nlc3Npb24YAyABKAsyIy5hbnl0dHkuYXBpLnYxLkVuZHBvaW50U2Vzc2lvblN0YW'
    '1wUgdzZXNzaW9uEi0KBWVycm9yGAQgASgLMhcuYW55dHR5LmFwaS52MS5BcGlFcnJvclIFZXJy'
    'b3I=');
