// This is a generated file - do not edit.
//
// Generated from bindingpb/client_binding.proto.

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

@$core.Deprecated('Use connectIntentDescriptor instead')
const ConnectIntent$json = {
  '1': 'ConnectIntent',
  '2': [
    {'1': 'CONNECT_INTENT_UNSPECIFIED', '2': 0},
    {'1': 'CONNECT_INTENT_INTERACTIVE', '2': 1},
    {'1': 'CONNECT_INTENT_BACKGROUND', '2': 2},
    {'1': 'CONNECT_INTENT_PROBE', '2': 3},
  ],
};

/// Descriptor for `ConnectIntent`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List connectIntentDescriptor = $convert.base64Decode(
    'Cg1Db25uZWN0SW50ZW50Eh4KGkNPTk5FQ1RfSU5URU5UX1VOU1BFQ0lGSUVEEAASHgoaQ09OTk'
    'VDVF9JTlRFTlRfSU5URVJBQ1RJVkUQARIdChlDT05ORUNUX0lOVEVOVF9CQUNLR1JPVU5EEAIS'
    'GAoUQ09OTkVDVF9JTlRFTlRfUFJPQkUQAw==');

@$core.Deprecated('Use resourceStreamFrameTypeDescriptor instead')
const ResourceStreamFrameType$json = {
  '1': 'ResourceStreamFrameType',
  '2': [
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_FILE_DATA', '2': 1},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_FILE_ACK', '2': 2},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH', '2': 3},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT', '2': 4},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_ERROR', '2': 5},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH_AUTO', '2': 6},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_PTY_OUTPUT', '2': 7},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST', '2': 8},
    {'1': 'RESOURCE_STREAM_FRAME_TYPE_PTY_CLOSED', '2': 9},
  ],
};

/// Descriptor for `ResourceStreamFrameType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List resourceStreamFrameTypeDescriptor = $convert.base64Decode(
    'ChdSZXNvdXJjZVN0cmVhbUZyYW1lVHlwZRIqCiZSRVNPVVJDRV9TVFJFQU1fRlJBTUVfVFlQRV'
    '9VTlNQRUNJRklFRBAAEigKJFJFU09VUkNFX1NUUkVBTV9GUkFNRV9UWVBFX0ZJTEVfREFUQRAB'
    'EicKI1JFU09VUkNFX1NUUkVBTV9GUkFNRV9UWVBFX0ZJTEVfQUNLEAISKgomUkVTT1VSQ0VfU1'
    'RSRUFNX0ZSQU1FX1RZUEVfRklMRV9GSU5JU0gQAxIqCiZSRVNPVVJDRV9TVFJFQU1fRlJBTUVf'
    'VFlQRV9GSUxFX1JFU1VMVBAEEiQKIFJFU09VUkNFX1NUUkVBTV9GUkFNRV9UWVBFX0VSUk9SEA'
    'USLworUkVTT1VSQ0VfU1RSRUFNX0ZSQU1FX1RZUEVfRklMRV9GSU5JU0hfQVVUTxAGEikKJVJF'
    'U09VUkNFX1NUUkVBTV9GUkFNRV9UWVBFX1BUWV9PVVRQVVQQBxIsCihSRVNPVVJDRV9TVFJFQU'
    '1fRlJBTUVfVFlQRV9QVFlfU1lOQ19MT1NUEAgSKQolUkVTT1VSQ0VfU1RSRUFNX0ZSQU1FX1RZ'
    'UEVfUFRZX0NMT1NFRBAJ');

@$core.Deprecated('Use connectionRouteKindDescriptor instead')
const ConnectionRouteKind$json = {
  '1': 'ConnectionRouteKind',
  '2': [
    {'1': 'CONNECTION_ROUTE_KIND_UNSPECIFIED', '2': 0},
    {'1': 'CONNECTION_ROUTE_KIND_LOCAL', '2': 1},
    {'1': 'CONNECTION_ROUTE_KIND_DIRECT', '2': 2},
    {'1': 'CONNECTION_ROUTE_KIND_SSH', '2': 3},
    {'1': 'CONNECTION_ROUTE_KIND_CLOUD', '2': 5},
  ],
  '4': [
    {'1': 4, '2': 4},
  ],
  '5': ['CONNECTION_ROUTE_KIND_MANAGED_CLOUD'],
};

/// Descriptor for `ConnectionRouteKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List connectionRouteKindDescriptor = $convert.base64Decode(
    'ChNDb25uZWN0aW9uUm91dGVLaW5kEiUKIUNPTk5FQ1RJT05fUk9VVEVfS0lORF9VTlNQRUNJRk'
    'lFRBAAEh8KG0NPTk5FQ1RJT05fUk9VVEVfS0lORF9MT0NBTBABEiAKHENPTk5FQ1RJT05fUk9V'
    'VEVfS0lORF9ESVJFQ1QQAhIdChlDT05ORUNUSU9OX1JPVVRFX0tJTkRfU1NIEAMSHwobQ09OTk'
    'VDVElPTl9ST1VURV9LSU5EX0NMT1VEEAUiBAgEEAQqI0NPTk5FQ1RJT05fUk9VVEVfS0lORF9N'
    'QU5BR0VEX0NMT1VE');

@$core.Deprecated('Use connectionObservedPathDescriptor instead')
const ConnectionObservedPath$json = {
  '1': 'ConnectionObservedPath',
  '2': [
    {'1': 'CONNECTION_OBSERVED_PATH_UNSPECIFIED', '2': 0},
    {'1': 'CONNECTION_OBSERVED_PATH_DIRECT', '2': 1},
    {'1': 'CONNECTION_OBSERVED_PATH_SINGLE_RELAY', '2': 2},
  ],
};

/// Descriptor for `ConnectionObservedPath`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List connectionObservedPathDescriptor = $convert.base64Decode(
    'ChZDb25uZWN0aW9uT2JzZXJ2ZWRQYXRoEigKJENPTk5FQ1RJT05fT0JTRVJWRURfUEFUSF9VTl'
    'NQRUNJRklFRBAAEiMKH0NPTk5FQ1RJT05fT0JTRVJWRURfUEFUSF9ESVJFQ1QQARIpCiVDT05O'
    'RUNUSU9OX09CU0VSVkVEX1BBVEhfU0lOR0xFX1JFTEFZEAI=');

@$core.Deprecated('Use connectionCandidateTypeDescriptor instead')
const ConnectionCandidateType$json = {
  '1': 'ConnectionCandidateType',
  '2': [
    {'1': 'CONNECTION_CANDIDATE_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'CONNECTION_CANDIDATE_TYPE_HOST', '2': 1},
    {'1': 'CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE', '2': 2},
    {'1': 'CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE', '2': 3},
    {'1': 'CONNECTION_CANDIDATE_TYPE_RELAY', '2': 4},
  ],
};

/// Descriptor for `ConnectionCandidateType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List connectionCandidateTypeDescriptor = $convert.base64Decode(
    'ChdDb25uZWN0aW9uQ2FuZGlkYXRlVHlwZRIpCiVDT05ORUNUSU9OX0NBTkRJREFURV9UWVBFX1'
    'VOU1BFQ0lGSUVEEAASIgoeQ09OTkVDVElPTl9DQU5ESURBVEVfVFlQRV9IT1NUEAESLgoqQ09O'
    'TkVDVElPTl9DQU5ESURBVEVfVFlQRV9TRVJWRVJfUkVGTEVYSVZFEAISLAooQ09OTkVDVElPTl'
    '9DQU5ESURBVEVfVFlQRV9QRUVSX1JFRkxFWElWRRADEiMKH0NPTk5FQ1RJT05fQ0FORElEQVRF'
    'X1RZUEVfUkVMQVkQBA==');

@$core.Deprecated('Use connectionTransportDescriptor instead')
const ConnectionTransport$json = {
  '1': 'ConnectionTransport',
  '2': [
    {'1': 'CONNECTION_TRANSPORT_UNSPECIFIED', '2': 0},
    {'1': 'CONNECTION_TRANSPORT_UDP', '2': 1},
    {'1': 'CONNECTION_TRANSPORT_TCP', '2': 2},
  ],
};

/// Descriptor for `ConnectionTransport`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List connectionTransportDescriptor = $convert.base64Decode(
    'ChNDb25uZWN0aW9uVHJhbnNwb3J0EiQKIENPTk5FQ1RJT05fVFJBTlNQT1JUX1VOU1BFQ0lGSU'
    'VEEAASHAoYQ09OTkVDVElPTl9UUkFOU1BPUlRfVURQEAESHAoYQ09OTkVDVElPTl9UUkFOU1BP'
    'UlRfVENQEAI=');

@$core.Deprecated('Use connectionPolicyAvailabilityReasonDescriptor instead')
const ConnectionPolicyAvailabilityReason$json = {
  '1': 'ConnectionPolicyAvailabilityReason',
  '2': [
    {'1': 'CONNECTION_POLICY_AVAILABILITY_REASON_UNSPECIFIED', '2': 0},
    {'1': 'CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE', '2': 1},
    {'1': 'CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_NOT_CONFIGURED', '2': 2},
    {'1': 'CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_DISABLED', '2': 3},
    {'1': 'CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED', '2': 4},
    {
      '1': 'CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE',
      '2': 5
    },
    {'1': 'CONNECTION_POLICY_AVAILABILITY_REASON_CLOUD_UNAVAILABLE', '2': 6},
  ],
};

/// Descriptor for `ConnectionPolicyAvailabilityReason`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List connectionPolicyAvailabilityReasonDescriptor = $convert.base64Decode(
    'CiJDb25uZWN0aW9uUG9saWN5QXZhaWxhYmlsaXR5UmVhc29uEjUKMUNPTk5FQ1RJT05fUE9MSU'
    'NZX0FWQUlMQUJJTElUWV9SRUFTT05fVU5TUEVDSUZJRUQQABIzCi9DT05ORUNUSU9OX1BPTElD'
    'WV9BVkFJTEFCSUxJVFlfUkVBU09OX0FWQUlMQUJMRRABEj4KOkNPTk5FQ1RJT05fUE9MSUNZX0'
    'FWQUlMQUJJTElUWV9SRUFTT05fUk9VVEVfTk9UX0NPTkZJR1VSRUQQAhI4CjRDT05ORUNUSU9O'
    'X1BPTElDWV9BVkFJTEFCSUxJVFlfUkVBU09OX1JPVVRFX0RJU0FCTEVEEAMSPgo6Q09OTkVDVE'
    'lPTl9QT0xJQ1lfQVZBSUxBQklMSVRZX1JFQVNPTl9QTEFURk9STV9VTlNVUFBPUlRFRBAEEkAK'
    'PENPTk5FQ1RJT05fUE9MSUNZX0FWQUlMQUJJTElUWV9SRUFTT05fQ1JFREVOVElBTF9VTkFWQU'
    'lMQUJMRRAFEjsKN0NPTk5FQ1RJT05fUE9MSUNZX0FWQUlMQUJJTElUWV9SRUFTT05fQ0xPVURf'
    'VU5BVkFJTEFCTEUQBg==');

@$core.Deprecated('Use endpointConnectionPhaseDescriptor instead')
const EndpointConnectionPhase$json = {
  '1': 'EndpointConnectionPhase',
  '2': [
    {'1': 'ENDPOINT_CONNECTION_PHASE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_CONNECTION_PHASE_IDLE', '2': 1},
    {'1': 'ENDPOINT_CONNECTION_PHASE_PLANNING', '2': 2},
    {'1': 'ENDPOINT_CONNECTION_PHASE_RESOLVING', '2': 3},
    {'1': 'ENDPOINT_CONNECTION_PHASE_SIGNALING', '2': 4},
    {'1': 'ENDPOINT_CONNECTION_PHASE_CONNECTING', '2': 5},
    {'1': 'ENDPOINT_CONNECTION_PHASE_AUTHORIZING', '2': 6},
    {'1': 'ENDPOINT_CONNECTION_PHASE_READY', '2': 7},
    {'1': 'ENDPOINT_CONNECTION_PHASE_OFFLINE', '2': 8},
  ],
};

/// Descriptor for `EndpointConnectionPhase`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointConnectionPhaseDescriptor = $convert.base64Decode(
    'ChdFbmRwb2ludENvbm5lY3Rpb25QaGFzZRIpCiVFTkRQT0lOVF9DT05ORUNUSU9OX1BIQVNFX1'
    'VOU1BFQ0lGSUVEEAASIgoeRU5EUE9JTlRfQ09OTkVDVElPTl9QSEFTRV9JRExFEAESJgoiRU5E'
    'UE9JTlRfQ09OTkVDVElPTl9QSEFTRV9QTEFOTklORxACEicKI0VORFBPSU5UX0NPTk5FQ1RJT0'
    '5fUEhBU0VfUkVTT0xWSU5HEAMSJwojRU5EUE9JTlRfQ09OTkVDVElPTl9QSEFTRV9TSUdOQUxJ'
    'TkcQBBIoCiRFTkRQT0lOVF9DT05ORUNUSU9OX1BIQVNFX0NPTk5FQ1RJTkcQBRIpCiVFTkRQT0'
    'lOVF9DT05ORUNUSU9OX1BIQVNFX0FVVEhPUklaSU5HEAYSIwofRU5EUE9JTlRfQ09OTkVDVElP'
    'Tl9QSEFTRV9SRUFEWRAHEiUKIUVORFBPSU5UX0NPTk5FQ1RJT05fUEhBU0VfT0ZGTElORRAI');

@$core.Deprecated('Use endpointSupervisorModeDescriptor instead')
const EndpointSupervisorMode$json = {
  '1': 'EndpointSupervisorMode',
  '2': [
    {'1': 'ENDPOINT_SUPERVISOR_MODE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_SUPERVISOR_MODE_SHADOW', '2': 1},
    {'1': 'ENDPOINT_SUPERVISOR_MODE_TAKEOVER', '2': 2},
  ],
};

/// Descriptor for `EndpointSupervisorMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointSupervisorModeDescriptor = $convert.base64Decode(
    'ChZFbmRwb2ludFN1cGVydmlzb3JNb2RlEigKJEVORFBPSU5UX1NVUEVSVklTT1JfTU9ERV9VTl'
    'NQRUNJRklFRBAAEiMKH0VORFBPSU5UX1NVUEVSVklTT1JfTU9ERV9TSEFET1cQARIlCiFFTkRQ'
    'T0lOVF9TVVBFUlZJU09SX01PREVfVEFLRU9WRVIQAg==');

@$core.Deprecated('Use connectionSnapshotDescriptor instead')
const ConnectionSnapshot$json = {
  '1': 'ConnectionSnapshot',
  '2': [
    {'1': 'route_id', '3': 1, '4': 1, '5': 9, '10': 'routeId'},
    {
      '1': 'route_kind',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionRouteKind',
      '10': 'routeKind'
    },
    {
      '1': 'observed_path',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionObservedPath',
      '10': 'observedPath'
    },
    {'1': 'selection_reason', '3': 4, '4': 1, '5': 9, '10': 'selectionReason'},
    {
      '1': 'sampled_at_unix_nano',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'sampledAtUnixNano'
    },
    {'1': 'round_trip_nanos', '3': 6, '4': 1, '5': 3, '10': 'roundTripNanos'},
    {
      '1': 'local_candidate_type',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionCandidateType',
      '10': 'localCandidateType'
    },
    {
      '1': 'remote_candidate_type',
      '3': 8,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionCandidateType',
      '10': 'remoteCandidateType'
    },
    {
      '1': 'local_protocol',
      '3': 9,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionTransport',
      '10': 'localProtocol'
    },
    {
      '1': 'remote_protocol',
      '3': 10,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionTransport',
      '10': 'remoteProtocol'
    },
    {
      '1': 'relay_transport',
      '3': 11,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionTransport',
      '10': 'relayTransport'
    },
    {'1': 'network_class', '3': 12, '4': 1, '5': 9, '10': 'networkClass'},
    {'1': 'bytes_sent', '3': 13, '4': 1, '5': 4, '10': 'bytesSent'},
    {'1': 'bytes_received', '3': 14, '4': 1, '5': 4, '10': 'bytesReceived'},
    {'1': 'packets_sent', '3': 15, '4': 1, '5': 4, '10': 'packetsSent'},
    {'1': 'loss_events', '3': 16, '4': 1, '5': 4, '10': 'lossEvents'},
    {'1': 'connected', '3': 17, '4': 1, '5': 8, '10': 'connected'},
    {'1': 'local_ip', '3': 18, '4': 1, '5': 9, '10': 'localIp'},
    {'1': 'remote_ip', '3': 19, '4': 1, '5': 9, '10': 'remoteIp'},
    {'1': 'local_port', '3': 20, '4': 1, '5': 13, '10': 'localPort'},
    {'1': 'remote_port', '3': 21, '4': 1, '5': 13, '10': 'remotePort'},
    {
      '1': 'candidate_pair_id',
      '3': 22,
      '4': 1,
      '5': 9,
      '10': 'candidatePairId'
    },
    {'1': 'local_related_ip', '3': 23, '4': 1, '5': 9, '10': 'localRelatedIp'},
    {
      '1': 'local_related_port',
      '3': 24,
      '4': 1,
      '5': 13,
      '10': 'localRelatedPort'
    },
    {
      '1': 'remote_related_ip',
      '3': 25,
      '4': 1,
      '5': 9,
      '10': 'remoteRelatedIp'
    },
    {
      '1': 'remote_related_port',
      '3': 26,
      '4': 1,
      '5': 13,
      '10': 'remoteRelatedPort'
    },
  ],
};

/// Descriptor for `ConnectionSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionSnapshotDescriptor = $convert.base64Decode(
    'ChJDb25uZWN0aW9uU25hcHNob3QSGQoIcm91dGVfaWQYASABKAlSB3JvdXRlSWQSTAoKcm91dG'
    'Vfa2luZBgCIAEoDjItLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5Db25uZWN0aW9uUm91dGVL'
    'aW5kUglyb3V0ZUtpbmQSVQoNb2JzZXJ2ZWRfcGF0aBgDIAEoDjIwLmFueXR0eS5jbGllbnQuYm'
    'luZGluZy52MS5Db25uZWN0aW9uT2JzZXJ2ZWRQYXRoUgxvYnNlcnZlZFBhdGgSKQoQc2VsZWN0'
    'aW9uX3JlYXNvbhgEIAEoCVIPc2VsZWN0aW9uUmVhc29uEi8KFHNhbXBsZWRfYXRfdW5peF9uYW'
    '5vGAUgASgDUhFzYW1wbGVkQXRVbml4TmFubxIoChByb3VuZF90cmlwX25hbm9zGAYgASgDUg5y'
    'b3VuZFRyaXBOYW5vcxJjChRsb2NhbF9jYW5kaWRhdGVfdHlwZRgHIAEoDjIxLmFueXR0eS5jbG'
    'llbnQuYmluZGluZy52MS5Db25uZWN0aW9uQ2FuZGlkYXRlVHlwZVISbG9jYWxDYW5kaWRhdGVU'
    'eXBlEmUKFXJlbW90ZV9jYW5kaWRhdGVfdHlwZRgIIAEoDjIxLmFueXR0eS5jbGllbnQuYmluZG'
    'luZy52MS5Db25uZWN0aW9uQ2FuZGlkYXRlVHlwZVITcmVtb3RlQ2FuZGlkYXRlVHlwZRJUCg5s'
    'b2NhbF9wcm90b2NvbBgJIAEoDjItLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5Db25uZWN0aW'
    '9uVHJhbnNwb3J0Ug1sb2NhbFByb3RvY29sElYKD3JlbW90ZV9wcm90b2NvbBgKIAEoDjItLmFu'
    'eXR0eS5jbGllbnQuYmluZGluZy52MS5Db25uZWN0aW9uVHJhbnNwb3J0Ug5yZW1vdGVQcm90b2'
    'NvbBJWCg9yZWxheV90cmFuc3BvcnQYCyABKA4yLS5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEu'
    'Q29ubmVjdGlvblRyYW5zcG9ydFIOcmVsYXlUcmFuc3BvcnQSIwoNbmV0d29ya19jbGFzcxgMIA'
    'EoCVIMbmV0d29ya0NsYXNzEh0KCmJ5dGVzX3NlbnQYDSABKARSCWJ5dGVzU2VudBIlCg5ieXRl'
    'c19yZWNlaXZlZBgOIAEoBFINYnl0ZXNSZWNlaXZlZBIhCgxwYWNrZXRzX3NlbnQYDyABKARSC3'
    'BhY2tldHNTZW50Eh8KC2xvc3NfZXZlbnRzGBAgASgEUgpsb3NzRXZlbnRzEhwKCWNvbm5lY3Rl'
    'ZBgRIAEoCFIJY29ubmVjdGVkEhkKCGxvY2FsX2lwGBIgASgJUgdsb2NhbElwEhsKCXJlbW90ZV'
    '9pcBgTIAEoCVIIcmVtb3RlSXASHQoKbG9jYWxfcG9ydBgUIAEoDVIJbG9jYWxQb3J0Eh8KC3Jl'
    'bW90ZV9wb3J0GBUgASgNUgpyZW1vdGVQb3J0EioKEWNhbmRpZGF0ZV9wYWlyX2lkGBYgASgJUg'
    '9jYW5kaWRhdGVQYWlySWQSKAoQbG9jYWxfcmVsYXRlZF9pcBgXIAEoCVIObG9jYWxSZWxhdGVk'
    'SXASLAoSbG9jYWxfcmVsYXRlZF9wb3J0GBggASgNUhBsb2NhbFJlbGF0ZWRQb3J0EioKEXJlbW'
    '90ZV9yZWxhdGVkX2lwGBkgASgJUg9yZW1vdGVSZWxhdGVkSXASLgoTcmVtb3RlX3JlbGF0ZWRf'
    'cG9ydBgaIAEoDVIRcmVtb3RlUmVsYXRlZFBvcnQ=');

@$core.Deprecated('Use connectionPolicyDescriptor instead')
const ConnectionPolicy$json = {
  '1': 'ConnectionPolicy',
  '2': [
    {
      '1': 'route_preference',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.EndpointRoutePreference',
      '10': 'routePreference'
    },
    {
      '1': 'cloud_relay_mode',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.ManagedWebRTCRelayMode',
      '10': 'cloudRelayMode'
    },
    {
      '1': 'relay_transport',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.remote.auth.v1.ManagedWebRTCRelayTransport',
      '10': 'relayTransport'
    },
  ],
};

/// Descriptor for `ConnectionPolicy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyDescriptor = $convert.base64Decode(
    'ChBDb25uZWN0aW9uUG9saWN5ElkKEHJvdXRlX3ByZWZlcmVuY2UYASABKA4yLi5hbnl0dHkucm'
    'Vtb3RlLmF1dGgudjEuRW5kcG9pbnRSb3V0ZVByZWZlcmVuY2VSD3JvdXRlUHJlZmVyZW5jZRJX'
    'ChBjbG91ZF9yZWxheV9tb2RlGAIgASgOMi0uYW55dHR5LnJlbW90ZS5hdXRoLnYxLk1hbmFnZW'
    'RXZWJSVENSZWxheU1vZGVSDmNsb3VkUmVsYXlNb2RlElsKD3JlbGF5X3RyYW5zcG9ydBgDIAEo'
    'DjIyLmFueXR0eS5yZW1vdGUuYXV0aC52MS5NYW5hZ2VkV2ViUlRDUmVsYXlUcmFuc3BvcnRSDn'
    'JlbGF5VHJhbnNwb3J0');

@$core.Deprecated('Use connectionPolicyRouteAvailabilityDescriptor instead')
const ConnectionPolicyRouteAvailability$json = {
  '1': 'ConnectionPolicyRouteAvailability',
  '2': [
    {
      '1': 'route_kind',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionRouteKind',
      '10': 'routeKind'
    },
    {'1': 'available', '3': 2, '4': 1, '5': 8, '10': 'available'},
    {
      '1': 'reason',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionPolicyAvailabilityReason',
      '10': 'reason'
    },
  ],
};

/// Descriptor for `ConnectionPolicyRouteAvailability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyRouteAvailabilityDescriptor = $convert.base64Decode(
    'CiFDb25uZWN0aW9uUG9saWN5Um91dGVBdmFpbGFiaWxpdHkSTAoKcm91dGVfa2luZBgBIAEoDj'
    'ItLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5Db25uZWN0aW9uUm91dGVLaW5kUglyb3V0ZUtp'
    'bmQSHAoJYXZhaWxhYmxlGAIgASgIUglhdmFpbGFibGUSVAoGcmVhc29uGAMgASgOMjwuYW55dH'
    'R5LmNsaWVudC5iaW5kaW5nLnYxLkNvbm5lY3Rpb25Qb2xpY3lBdmFpbGFiaWxpdHlSZWFzb25S'
    'BnJlYXNvbg==');

@$core.Deprecated('Use connectionPolicyStateDescriptor instead')
const ConnectionPolicyState$json = {
  '1': 'ConnectionPolicyState',
  '2': [
    {
      '1': 'policy',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicy',
      '10': 'policy'
    },
    {
      '1': 'routes',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyRouteAvailability',
      '10': 'routes'
    },
  ],
};

/// Descriptor for `ConnectionPolicyState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyStateDescriptor = $convert.base64Decode(
    'ChVDb25uZWN0aW9uUG9saWN5U3RhdGUSQgoGcG9saWN5GAEgASgLMiouYW55dHR5LmNsaWVudC'
    '5iaW5kaW5nLnYxLkNvbm5lY3Rpb25Qb2xpY3lSBnBvbGljeRJTCgZyb3V0ZXMYAiADKAsyOy5h'
    'bnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ29ubmVjdGlvblBvbGljeVJvdXRlQXZhaWxhYmlsaX'
    'R5UgZyb3V0ZXM=');

@$core.Deprecated('Use connectionPolicyGetRequestDescriptor instead')
const ConnectionPolicyGetRequest$json = {
  '1': 'ConnectionPolicyGetRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
  ],
};

/// Descriptor for `ConnectionPolicyGetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyGetRequestDescriptor =
    $convert.base64Decode(
        'ChpDb25uZWN0aW9uUG9saWN5R2V0UmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZX'
        'N0SWQSHwoLZW5kcG9pbnRfaWQYAiABKAlSCmVuZHBvaW50SWQ=');

@$core.Deprecated('Use connectionPolicyGetResultDescriptor instead')
const ConnectionPolicyGetResult$json = {
  '1': 'ConnectionPolicyGetResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'state',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyState',
      '10': 'state'
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

/// Descriptor for `ConnectionPolicyGetResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyGetResultDescriptor = $convert.base64Decode(
    'ChlDb25uZWN0aW9uUG9saWN5R2V0UmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3'
    'RJZBIpChBvcGVyYXRpb25faGFuZGxlGAIgASgEUg9vcGVyYXRpb25IYW5kbGUSRQoFc3RhdGUY'
    'AyABKAsyLy5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ29ubmVjdGlvblBvbGljeVN0YXRlUg'
    'VzdGF0ZRItCgVlcnJvchgEIAEoCzIXLmFueXR0eS5hcGkudjEuQXBpRXJyb3JSBWVycm9y');

@$core.Deprecated('Use connectionPolicyApplyRequestDescriptor instead')
const ConnectionPolicyApplyRequest$json = {
  '1': 'ConnectionPolicyApplyRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
    {
      '1': 'policy',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicy',
      '10': 'policy'
    },
  ],
};

/// Descriptor for `ConnectionPolicyApplyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyApplyRequestDescriptor =
    $convert.base64Decode(
        'ChxDb25uZWN0aW9uUG9saWN5QXBwbHlSZXF1ZXN0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcX'
        'Vlc3RJZBIfCgtlbmRwb2ludF9pZBgCIAEoCVIKZW5kcG9pbnRJZBJCCgZwb2xpY3kYAyABKAsy'
        'Ki5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ29ubmVjdGlvblBvbGljeVIGcG9saWN5');

@$core.Deprecated('Use connectionPolicyApplyResultDescriptor instead')
const ConnectionPolicyApplyResult$json = {
  '1': 'ConnectionPolicyApplyResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'state',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyState',
      '10': 'state'
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

/// Descriptor for `ConnectionPolicyApplyResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionPolicyApplyResultDescriptor = $convert.base64Decode(
    'ChtDb25uZWN0aW9uUG9saWN5QXBwbHlSZXN1bHQSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdW'
    'VzdElkEikKEG9wZXJhdGlvbl9oYW5kbGUYAiABKARSD29wZXJhdGlvbkhhbmRsZRJFCgVzdGF0'
    'ZRgDIAEoCzIvLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5Db25uZWN0aW9uUG9saWN5U3RhdG'
    'VSBXN0YXRlEi0KBWVycm9yGAQgASgLMhcuYW55dHR5LmFwaS52MS5BcGlFcnJvclIFZXJyb3I=');

@$core.Deprecated('Use connectionSnapshotGetRequestDescriptor instead')
const ConnectionSnapshotGetRequest$json = {
  '1': 'ConnectionSnapshotGetRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'session_handle', '3': 2, '4': 1, '5': 4, '10': 'sessionHandle'},
  ],
};

/// Descriptor for `ConnectionSnapshotGetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionSnapshotGetRequestDescriptor =
    $convert.base64Decode(
        'ChxDb25uZWN0aW9uU25hcHNob3RHZXRSZXF1ZXN0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcX'
        'Vlc3RJZBIlCg5zZXNzaW9uX2hhbmRsZRgCIAEoBFINc2Vzc2lvbkhhbmRsZQ==');

@$core.Deprecated('Use connectionSnapshotGetResultDescriptor instead')
const ConnectionSnapshotGetResult$json = {
  '1': 'ConnectionSnapshotGetResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'session_handle', '3': 3, '4': 1, '5': 4, '10': 'sessionHandle'},
    {
      '1': 'connection',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionSnapshot',
      '10': 'connection'
    },
    {
      '1': 'error',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `ConnectionSnapshotGetResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionSnapshotGetResultDescriptor = $convert.base64Decode(
    'ChtDb25uZWN0aW9uU25hcHNob3RHZXRSZXN1bHQSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdW'
    'VzdElkEikKEG9wZXJhdGlvbl9oYW5kbGUYAiABKARSD29wZXJhdGlvbkhhbmRsZRIlCg5zZXNz'
    'aW9uX2hhbmRsZRgDIAEoBFINc2Vzc2lvbkhhbmRsZRJMCgpjb25uZWN0aW9uGAQgASgLMiwuYW'
    '55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkNvbm5lY3Rpb25TbmFwc2hvdFIKY29ubmVjdGlvbhIt'
    'CgVlcnJvchgFIAEoCzIXLmFueXR0eS5hcGkudjEuQXBpRXJyb3JSBWVycm9y');

@$core.Deprecated('Use sessionInvalidateRequestDescriptor instead')
const SessionInvalidateRequest$json = {
  '1': 'SessionInvalidateRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'session_handle', '3': 2, '4': 1, '5': 4, '10': 'sessionHandle'},
  ],
};

/// Descriptor for `SessionInvalidateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sessionInvalidateRequestDescriptor =
    $convert.base64Decode(
        'ChhTZXNzaW9uSW52YWxpZGF0ZVJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdE'
        'lkEiUKDnNlc3Npb25faGFuZGxlGAIgASgEUg1zZXNzaW9uSGFuZGxl');

@$core.Deprecated('Use sessionInvalidateResultDescriptor instead')
const SessionInvalidateResult$json = {
  '1': 'SessionInvalidateResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'session_handle', '3': 3, '4': 1, '5': 4, '10': 'sessionHandle'},
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

/// Descriptor for `SessionInvalidateResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sessionInvalidateResultDescriptor = $convert.base64Decode(
    'ChdTZXNzaW9uSW52YWxpZGF0ZVJlc3VsdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SW'
    'QSKQoQb3BlcmF0aW9uX2hhbmRsZRgCIAEoBFIPb3BlcmF0aW9uSGFuZGxlEiUKDnNlc3Npb25f'
    'aGFuZGxlGAMgASgEUg1zZXNzaW9uSGFuZGxlEi0KBWVycm9yGAQgASgLMhcuYW55dHR5LmFwaS'
    '52MS5BcGlFcnJvclIFZXJyb3I=');

@$core.Deprecated('Use endpointDisconnectRequestDescriptor instead')
const EndpointDisconnectRequest$json = {
  '1': 'EndpointDisconnectRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
  ],
};

/// Descriptor for `EndpointDisconnectRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointDisconnectRequestDescriptor =
    $convert.base64Decode(
        'ChlFbmRwb2ludERpc2Nvbm5lY3RSZXF1ZXN0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3'
        'RJZBIfCgtlbmRwb2ludF9pZBgCIAEoCVIKZW5kcG9pbnRJZA==');

@$core.Deprecated('Use endpointDisconnectResultDescriptor instead')
const EndpointDisconnectResult$json = {
  '1': 'EndpointDisconnectResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'endpoint_id', '3': 3, '4': 1, '5': 9, '10': 'endpointId'},
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

/// Descriptor for `EndpointDisconnectResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointDisconnectResultDescriptor = $convert.base64Decode(
    'ChhFbmRwb2ludERpc2Nvbm5lY3RSZXN1bHQSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdE'
    'lkEikKEG9wZXJhdGlvbl9oYW5kbGUYAiABKARSD29wZXJhdGlvbkhhbmRsZRIfCgtlbmRwb2lu'
    'dF9pZBgDIAEoCVIKZW5kcG9pbnRJZBItCgVlcnJvchgEIAEoCzIXLmFueXR0eS5hcGkudjEuQX'
    'BpRXJyb3JSBWVycm9y');

@$core.Deprecated('Use endpointCloudPresenceGetRequestDescriptor instead')
const EndpointCloudPresenceGetRequest$json = {
  '1': 'EndpointCloudPresenceGetRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
  ],
};

/// Descriptor for `EndpointCloudPresenceGetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointCloudPresenceGetRequestDescriptor =
    $convert.base64Decode(
        'Ch9FbmRwb2ludENsb3VkUHJlc2VuY2VHZXRSZXF1ZXN0Eh0KCnJlcXVlc3RfaWQYASABKAlSCX'
        'JlcXVlc3RJZBIfCgtlbmRwb2ludF9pZBgCIAEoCVIKZW5kcG9pbnRJZA==');

@$core.Deprecated('Use endpointCloudPresenceGetResultDescriptor instead')
const EndpointCloudPresenceGetResult$json = {
  '1': 'EndpointCloudPresenceGetResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'endpoint_id', '3': 3, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'online', '3': 4, '4': 1, '5': 8, '10': 'online'},
    {
      '1': 'error',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
    {'1': 'device_id', '3': 6, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_fingerprint',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
    {'1': 'daemon_id', '3': 8, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'edge_id', '3': 9, '4': 1, '5': 9, '10': 'edgeId'},
    {'1': 'edge_name', '3': 10, '4': 1, '5': 9, '10': 'edgeName'},
    {'1': 'edge_region', '3': 11, '4': 1, '5': 9, '10': 'edgeRegion'},
    {
      '1': 'edge_public_endpoint',
      '3': 12,
      '4': 1,
      '5': 9,
      '10': 'edgePublicEndpoint'
    },
    {'1': 'edge_server_name', '3': 13, '4': 1, '5': 9, '10': 'edgeServerName'},
    {'1': 'locator_source', '3': 14, '4': 1, '5': 9, '10': 'locatorSource'},
    {
      '1': 'refreshed_from_controller',
      '3': 15,
      '4': 1,
      '5': 8,
      '10': 'refreshedFromController'
    },
  ],
};

/// Descriptor for `EndpointCloudPresenceGetResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointCloudPresenceGetResultDescriptor = $convert.base64Decode(
    'Ch5FbmRwb2ludENsb3VkUHJlc2VuY2VHZXRSZXN1bHQSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcm'
    'VxdWVzdElkEikKEG9wZXJhdGlvbl9oYW5kbGUYAiABKARSD29wZXJhdGlvbkhhbmRsZRIfCgtl'
    'bmRwb2ludF9pZBgDIAEoCVIKZW5kcG9pbnRJZBIWCgZvbmxpbmUYBCABKAhSBm9ubGluZRItCg'
    'VlcnJvchgFIAEoCzIXLmFueXR0eS5hcGkudjEuQXBpRXJyb3JSBWVycm9yEhsKCWRldmljZV9p'
    'ZBgGIAEoCVIIZGV2aWNlSWQSLQoSZGV2aWNlX2ZpbmdlcnByaW50GAcgASgJUhFkZXZpY2VGaW'
    '5nZXJwcmludBIbCglkYWVtb25faWQYCCABKAlSCGRhZW1vbklkEhcKB2VkZ2VfaWQYCSABKAlS'
    'BmVkZ2VJZBIbCgllZGdlX25hbWUYCiABKAlSCGVkZ2VOYW1lEh8KC2VkZ2VfcmVnaW9uGAsgAS'
    'gJUgplZGdlUmVnaW9uEjAKFGVkZ2VfcHVibGljX2VuZHBvaW50GAwgASgJUhJlZGdlUHVibGlj'
    'RW5kcG9pbnQSKAoQZWRnZV9zZXJ2ZXJfbmFtZRgNIAEoCVIOZWRnZVNlcnZlck5hbWUSJQoObG'
    '9jYXRvcl9zb3VyY2UYDiABKAlSDWxvY2F0b3JTb3VyY2USOgoZcmVmcmVzaGVkX2Zyb21fY29u'
    'dHJvbGxlchgPIAEoCFIXcmVmcmVzaGVkRnJvbUNvbnRyb2xsZXI=');

@$core.Deprecated('Use openSessionRequestDescriptor instead')
const OpenSessionRequest$json = {
  '1': 'OpenSessionRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'route_override', '3': 3, '4': 1, '5': 9, '10': 'routeOverride'},
    {
      '1': 'intent',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectIntent',
      '10': 'intent'
    },
  ],
  '9': [
    {'1': 5, '2': 6},
  ],
};

/// Descriptor for `OpenSessionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List openSessionRequestDescriptor = $convert.base64Decode(
    'ChJPcGVuU2Vzc2lvblJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdElkEh8KC2'
    'VuZHBvaW50X2lkGAIgASgJUgplbmRwb2ludElkEiUKDnJvdXRlX292ZXJyaWRlGAMgASgJUg1y'
    'b3V0ZU92ZXJyaWRlEj8KBmludGVudBgEIAEoDjInLmFueXR0eS5jbGllbnQuYmluZGluZy52MS'
    '5Db25uZWN0SW50ZW50UgZpbnRlbnRKBAgFEAY=');

@$core.Deprecated('Use importPairingRequestDescriptor instead')
const ImportPairingRequest$json = {
  '1': 'ImportPairingRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'portable_payload', '3': 2, '4': 1, '5': 9, '10': 'portablePayload'},
    {
      '1': 'expected_endpoint_id',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'expectedEndpointId'
    },
  ],
};

/// Descriptor for `ImportPairingRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List importPairingRequestDescriptor = $convert.base64Decode(
    'ChRJbXBvcnRQYWlyaW5nUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSKQ'
    'oQcG9ydGFibGVfcGF5bG9hZBgCIAEoCVIPcG9ydGFibGVQYXlsb2FkEjAKFGV4cGVjdGVkX2Vu'
    'ZHBvaW50X2lkGAMgASgJUhJleHBlY3RlZEVuZHBvaW50SWQ=');

@$core.Deprecated('Use importPairingResultDescriptor instead')
const ImportPairingResult$json = {
  '1': 'ImportPairingResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'endpoint',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointConfigV1',
      '10': 'endpoint'
    },
    {'1': 'ticket_id', '3': 4, '4': 1, '5': 9, '10': 'ticketId'},
    {
      '1': 'client_key_fingerprint',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'clientKeyFingerprint'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {
      '1': 'authorization_required',
      '3': 7,
      '4': 1,
      '5': 8,
      '10': 'authorizationRequired'
    },
    {
      '1': 'error',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
    {
      '1': 'registry',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRegistryV1',
      '10': 'registry'
    },
  ],
};

/// Descriptor for `ImportPairingResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List importPairingResultDescriptor = $convert.base64Decode(
    'ChNJbXBvcnRQYWlyaW5nUmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZBIpCh'
    'BvcGVyYXRpb25faGFuZGxlGAIgASgEUg9vcGVyYXRpb25IYW5kbGUSQwoIZW5kcG9pbnQYAyAB'
    'KAsyJy5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRDb25maWdWMVIIZW5kcG9pbnQSGw'
    'oJdGlja2V0X2lkGAQgASgJUgh0aWNrZXRJZBI0ChZjbGllbnRfa2V5X2ZpbmdlcnByaW50GAUg'
    'ASgJUhRjbGllbnRLZXlGaW5nZXJwcmludBIvChRleHBpcmVzX2F0X3VuaXhfbmFubxgGIAEoA1'
    'IRZXhwaXJlc0F0VW5peE5hbm8SNQoWYXV0aG9yaXphdGlvbl9yZXF1aXJlZBgHIAEoCFIVYXV0'
    'aG9yaXphdGlvblJlcXVpcmVkEi0KBWVycm9yGAggASgLMhcuYW55dHR5LmFwaS52MS5BcGlFcn'
    'JvclIFZXJyb3ISRQoIcmVnaXN0cnkYCSABKAsyKS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5k'
    'cG9pbnRSZWdpc3RyeVYxUghyZWdpc3RyeQ==');

@$core.Deprecated('Use deleteCredentialRequestDescriptor instead')
const DeleteCredentialRequest$json = {
  '1': 'DeleteCredentialRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'credential_ref', '3': 2, '4': 1, '5': 9, '10': 'credentialRef'},
  ],
};

/// Descriptor for `DeleteCredentialRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteCredentialRequestDescriptor =
    $convert.base64Decode(
        'ChdEZWxldGVDcmVkZW50aWFsUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SW'
        'QSJQoOY3JlZGVudGlhbF9yZWYYAiABKAlSDWNyZWRlbnRpYWxSZWY=');

@$core.Deprecated('Use deleteCredentialResultDescriptor instead')
const DeleteCredentialResult$json = {
  '1': 'DeleteCredentialResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'error',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `DeleteCredentialResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteCredentialResultDescriptor = $convert.base64Decode(
    'ChZEZWxldGVDcmVkZW50aWFsUmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZB'
    'IpChBvcGVyYXRpb25faGFuZGxlGAIgASgEUg9vcGVyYXRpb25IYW5kbGUSLQoFZXJyb3IYAyAB'
    'KAsyFy5hbnl0dHkuYXBpLnYxLkFwaUVycm9yUgVlcnJvcg==');

@$core.Deprecated('Use endpointRegistryGetRequestDescriptor instead')
const EndpointRegistryGetRequest$json = {
  '1': 'EndpointRegistryGetRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
  ],
};

/// Descriptor for `EndpointRegistryGetRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRegistryGetRequestDescriptor =
    $convert.base64Decode(
        'ChpFbmRwb2ludFJlZ2lzdHJ5R2V0UmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZX'
        'N0SWQ=');

@$core.Deprecated('Use endpointRegistryGetResultDescriptor instead')
const EndpointRegistryGetResult$json = {
  '1': 'EndpointRegistryGetResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'registry',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRegistryV1',
      '10': 'registry'
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

/// Descriptor for `EndpointRegistryGetResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRegistryGetResultDescriptor = $convert.base64Decode(
    'ChlFbmRwb2ludFJlZ2lzdHJ5R2V0UmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3'
    'RJZBIpChBvcGVyYXRpb25faGFuZGxlGAIgASgEUg9vcGVyYXRpb25IYW5kbGUSRQoIcmVnaXN0'
    'cnkYAyABKAsyKS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRSZWdpc3RyeVYxUghyZW'
    'dpc3RyeRItCgVlcnJvchgEIAEoCzIXLmFueXR0eS5hcGkudjEuQXBpRXJyb3JSBWVycm9y');

@$core.Deprecated('Use endpointUpsertRequestDescriptor instead')
const EndpointUpsertRequest$json = {
  '1': 'EndpointUpsertRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'endpoint',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointConfigV1',
      '10': 'endpoint'
    },
    {'1': 'make_default', '3': 3, '4': 1, '5': 8, '10': 'makeDefault'},
  ],
};

/// Descriptor for `EndpointUpsertRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointUpsertRequestDescriptor = $convert.base64Decode(
    'ChVFbmRwb2ludFVwc2VydFJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdElkEk'
    'MKCGVuZHBvaW50GAIgASgLMicuYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVuZHBvaW50Q29uZmln'
    'VjFSCGVuZHBvaW50EiEKDG1ha2VfZGVmYXVsdBgDIAEoCFILbWFrZURlZmF1bHQ=');

@$core.Deprecated('Use endpointUpsertResultDescriptor instead')
const EndpointUpsertResult$json = {
  '1': 'EndpointUpsertResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'endpoint',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointConfigV1',
      '10': 'endpoint'
    },
    {
      '1': 'registry',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRegistryV1',
      '10': 'registry'
    },
    {
      '1': 'error',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `EndpointUpsertResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointUpsertResultDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludFVwc2VydFJlc3VsdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSKQ'
    'oQb3BlcmF0aW9uX2hhbmRsZRgCIAEoBFIPb3BlcmF0aW9uSGFuZGxlEkMKCGVuZHBvaW50GAMg'
    'ASgLMicuYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVuZHBvaW50Q29uZmlnVjFSCGVuZHBvaW50Ek'
    'UKCHJlZ2lzdHJ5GAQgASgLMikuYW55dHR5LnJlbW90ZS5hdXRoLnYxLkVuZHBvaW50UmVnaXN0'
    'cnlWMVIIcmVnaXN0cnkSLQoFZXJyb3IYBSABKAsyFy5hbnl0dHkuYXBpLnYxLkFwaUVycm9yUg'
    'VlcnJvcg==');

@$core.Deprecated('Use endpointDeleteRequestDescriptor instead')
const EndpointDeleteRequest$json = {
  '1': 'EndpointDeleteRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
  ],
};

/// Descriptor for `EndpointDeleteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointDeleteRequestDescriptor = $convert.base64Decode(
    'ChVFbmRwb2ludERlbGV0ZVJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdElkEh'
    '8KC2VuZHBvaW50X2lkGAIgASgJUgplbmRwb2ludElk');

@$core.Deprecated('Use endpointDeleteResultDescriptor instead')
const EndpointDeleteResult$json = {
  '1': 'EndpointDeleteResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'endpoint_id', '3': 3, '4': 1, '5': 9, '10': 'endpointId'},
    {
      '1': 'registry',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRegistryV1',
      '10': 'registry'
    },
    {
      '1': 'error',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `EndpointDeleteResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointDeleteResultDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludERlbGV0ZVJlc3VsdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSKQ'
    'oQb3BlcmF0aW9uX2hhbmRsZRgCIAEoBFIPb3BlcmF0aW9uSGFuZGxlEh8KC2VuZHBvaW50X2lk'
    'GAMgASgJUgplbmRwb2ludElkEkUKCHJlZ2lzdHJ5GAQgASgLMikuYW55dHR5LnJlbW90ZS5hdX'
    'RoLnYxLkVuZHBvaW50UmVnaXN0cnlWMVIIcmVnaXN0cnkSLQoFZXJyb3IYBSABKAsyFy5hbnl0'
    'dHkuYXBpLnYxLkFwaUVycm9yUgVlcnJvcg==');

@$core.Deprecated('Use endpointShareReceiveRequestDescriptor instead')
const EndpointShareReceiveRequest$json = {
  '1': 'EndpointShareReceiveRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'portable_offer', '3': 2, '4': 1, '5': 9, '10': 'portableOffer'},
  ],
};

/// Descriptor for `EndpointShareReceiveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointShareReceiveRequestDescriptor =
    $convert.base64Decode(
        'ChtFbmRwb2ludFNoYXJlUmVjZWl2ZVJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdW'
        'VzdElkEiUKDnBvcnRhYmxlX29mZmVyGAIgASgJUg1wb3J0YWJsZU9mZmVy');

@$core.Deprecated('Use endpointShareRouteDiffDescriptor instead')
const EndpointShareRouteDiff$json = {
  '1': 'EndpointShareRouteDiff',
  '2': [
    {'1': 'route_id', '3': 1, '4': 1, '5': 9, '10': 'routeId'},
    {'1': 'route_kind', '3': 2, '4': 1, '5': 9, '10': 'routeKind'},
    {'1': 'action', '3': 3, '4': 1, '5': 9, '10': 'action'},
  ],
};

/// Descriptor for `EndpointShareRouteDiff`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointShareRouteDiffDescriptor = $convert.base64Decode(
    'ChZFbmRwb2ludFNoYXJlUm91dGVEaWZmEhkKCHJvdXRlX2lkGAEgASgJUgdyb3V0ZUlkEh0KCn'
    'JvdXRlX2tpbmQYAiABKAlSCXJvdXRlS2luZBIWCgZhY3Rpb24YAyABKAlSBmFjdGlvbg==');

@$core.Deprecated('Use endpointSharePreviewDescriptor instead')
const EndpointSharePreview$json = {
  '1': 'EndpointSharePreview',
  '2': [
    {'1': 'import_token', '3': 1, '4': 1, '5': 9, '10': 'importToken'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'label', '3': 3, '4': 1, '5': 9, '10': 'label'},
    {
      '1': 'identity',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointDaemonIdentity',
      '10': 'identity'
    },
    {
      '1': 'route_diffs',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointShareRouteDiff',
      '10': 'routeDiffs'
    },
    {
      '1': 'connect_mode_changed',
      '3': 6,
      '4': 1,
      '5': 8,
      '10': 'connectModeChanged'
    },
    {
      '1': 'selection_policy_changed',
      '3': 7,
      '4': 1,
      '5': 8,
      '10': 'selectionPolicyChanged'
    },
    {
      '1': 'credential_descriptors',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointCredentialDescriptor',
      '10': 'credentialDescriptors'
    },
    {
      '1': 'expires_at_unix_nano',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
  ],
};

/// Descriptor for `EndpointSharePreview`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSharePreviewDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludFNoYXJlUHJldmlldxIhCgxpbXBvcnRfdG9rZW4YASABKAlSC2ltcG9ydFRva2'
    'VuEh8KC2VuZHBvaW50X2lkGAIgASgJUgplbmRwb2ludElkEhQKBWxhYmVsGAMgASgJUgVsYWJl'
    'bBJJCghpZGVudGl0eRgEIAEoCzItLmFueXR0eS5yZW1vdGUuYXV0aC52MS5FbmRwb2ludERhZW'
    '1vbklkZW50aXR5UghpZGVudGl0eRJRCgtyb3V0ZV9kaWZmcxgFIAMoCzIwLmFueXR0eS5jbGll'
    'bnQuYmluZGluZy52MS5FbmRwb2ludFNoYXJlUm91dGVEaWZmUgpyb3V0ZURpZmZzEjAKFGNvbm'
    '5lY3RfbW9kZV9jaGFuZ2VkGAYgASgIUhJjb25uZWN0TW9kZUNoYW5nZWQSOAoYc2VsZWN0aW9u'
    'X3BvbGljeV9jaGFuZ2VkGAcgASgIUhZzZWxlY3Rpb25Qb2xpY3lDaGFuZ2VkEmoKFmNyZWRlbn'
    'RpYWxfZGVzY3JpcHRvcnMYCCADKAsyMy5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRD'
    'cmVkZW50aWFsRGVzY3JpcHRvclIVY3JlZGVudGlhbERlc2NyaXB0b3JzEi8KFGV4cGlyZXNfYX'
    'RfdW5peF9uYW5vGAkgASgDUhFleHBpcmVzQXRVbml4TmFubw==');

@$core.Deprecated('Use endpointShareReceiveResultDescriptor instead')
const EndpointShareReceiveResult$json = {
  '1': 'EndpointShareReceiveResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'preview',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointSharePreview',
      '10': 'preview'
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

/// Descriptor for `EndpointShareReceiveResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointShareReceiveResultDescriptor = $convert.base64Decode(
    'ChpFbmRwb2ludFNoYXJlUmVjZWl2ZVJlc3VsdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZX'
    'N0SWQSKQoQb3BlcmF0aW9uX2hhbmRsZRgCIAEoBFIPb3BlcmF0aW9uSGFuZGxlEkgKB3ByZXZp'
    'ZXcYAyABKAsyLi5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRW5kcG9pbnRTaGFyZVByZXZpZX'
    'dSB3ByZXZpZXcSLQoFZXJyb3IYBCABKAsyFy5hbnl0dHkuYXBpLnYxLkFwaUVycm9yUgVlcnJv'
    'cg==');

@$core.Deprecated('Use endpointShareCommitRequestDescriptor instead')
const EndpointShareCommitRequest$json = {
  '1': 'EndpointShareCommitRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'import_token', '3': 2, '4': 1, '5': 9, '10': 'importToken'},
  ],
};

/// Descriptor for `EndpointShareCommitRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointShareCommitRequestDescriptor =
    $convert.base64Decode(
        'ChpFbmRwb2ludFNoYXJlQ29tbWl0UmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZX'
        'N0SWQSIQoMaW1wb3J0X3Rva2VuGAIgASgJUgtpbXBvcnRUb2tlbg==');

@$core.Deprecated('Use endpointShareCommitResultDescriptor instead')
const EndpointShareCommitResult$json = {
  '1': 'EndpointShareCommitResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'endpoint',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointConfigV1',
      '10': 'endpoint'
    },
    {
      '1': 'registry',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRegistryV1',
      '10': 'registry'
    },
    {
      '1': 'authorization_required',
      '3': 5,
      '4': 1,
      '5': 8,
      '10': 'authorizationRequired'
    },
    {
      '1': 'error',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `EndpointShareCommitResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointShareCommitResultDescriptor = $convert.base64Decode(
    'ChlFbmRwb2ludFNoYXJlQ29tbWl0UmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3'
    'RJZBIpChBvcGVyYXRpb25faGFuZGxlGAIgASgEUg9vcGVyYXRpb25IYW5kbGUSQwoIZW5kcG9p'
    'bnQYAyABKAsyJy5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRDb25maWdWMVIIZW5kcG'
    '9pbnQSRQoIcmVnaXN0cnkYBCABKAsyKS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRS'
    'ZWdpc3RyeVYxUghyZWdpc3RyeRI1ChZhdXRob3JpemF0aW9uX3JlcXVpcmVkGAUgASgIUhVhdX'
    'Rob3JpemF0aW9uUmVxdWlyZWQSLQoFZXJyb3IYBiABKAsyFy5hbnl0dHkuYXBpLnYxLkFwaUVy'
    'cm9yUgVlcnJvcg==');

@$core.Deprecated('Use sSHCredentialProvisionRequestDescriptor instead')
const SSHCredentialProvisionRequest$json = {
  '1': 'SSHCredentialProvisionRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'endpoint_id', '3': 2, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'route_id', '3': 3, '4': 1, '5': 9, '10': 'routeId'},
  ],
};

/// Descriptor for `SSHCredentialProvisionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialProvisionRequestDescriptor =
    $convert.base64Decode(
        'Ch1TU0hDcmVkZW50aWFsUHJvdmlzaW9uUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZX'
        'F1ZXN0SWQSHwoLZW5kcG9pbnRfaWQYAiABKAlSCmVuZHBvaW50SWQSGQoIcm91dGVfaWQYAyAB'
        'KAlSB3JvdXRlSWQ=');

@$core.Deprecated('Use sSHCredentialProvisionResultDescriptor instead')
const SSHCredentialProvisionResult$json = {
  '1': 'SSHCredentialProvisionResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {
      '1': 'endpoint',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointConfigV1',
      '10': 'endpoint'
    },
    {
      '1': 'registry',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.remote.auth.v1.EndpointRegistryV1',
      '10': 'registry'
    },
    {'1': 'credential_ref', '3': 5, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'authorized_key', '3': 6, '4': 1, '5': 9, '10': 'authorizedKey'},
    {'1': 'key_fingerprint', '3': 7, '4': 1, '5': 9, '10': 'keyFingerprint'},
    {
      '1': 'error',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `SSHCredentialProvisionResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialProvisionResultDescriptor = $convert.base64Decode(
    'ChxTU0hDcmVkZW50aWFsUHJvdmlzaW9uUmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcX'
    'Vlc3RJZBIpChBvcGVyYXRpb25faGFuZGxlGAIgASgEUg9vcGVyYXRpb25IYW5kbGUSQwoIZW5k'
    'cG9pbnQYAyABKAsyJy5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9pbnRDb25maWdWMVIIZW'
    '5kcG9pbnQSRQoIcmVnaXN0cnkYBCABKAsyKS5hbnl0dHkucmVtb3RlLmF1dGgudjEuRW5kcG9p'
    'bnRSZWdpc3RyeVYxUghyZWdpc3RyeRIlCg5jcmVkZW50aWFsX3JlZhgFIAEoCVINY3JlZGVudG'
    'lhbFJlZhIlCg5hdXRob3JpemVkX2tleRgGIAEoCVINYXV0aG9yaXplZEtleRInCg9rZXlfZmlu'
    'Z2VycHJpbnQYByABKAlSDmtleUZpbmdlcnByaW50Ei0KBWVycm9yGAggASgLMhcuYW55dHR5Lm'
    'FwaS52MS5BcGlFcnJvclIFZXJyb3I=');

@$core.Deprecated('Use engineCommandDescriptor instead')
const EngineCommand$json = {
  '1': 'EngineCommand',
  '2': [
    {
      '1': 'import_pairing',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ImportPairingRequest',
      '9': 0,
      '10': 'importPairing'
    },
    {
      '1': 'delete_credential',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.DeleteCredentialRequest',
      '9': 0,
      '10': 'deleteCredential'
    },
    {
      '1': 'endpoint_registry_get',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointRegistryGetRequest',
      '9': 0,
      '10': 'endpointRegistryGet'
    },
    {
      '1': 'endpoint_upsert',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointUpsertRequest',
      '9': 0,
      '10': 'endpointUpsert'
    },
    {
      '1': 'endpoint_delete',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointDeleteRequest',
      '9': 0,
      '10': 'endpointDelete'
    },
    {
      '1': 'endpoint_share_receive',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointShareReceiveRequest',
      '9': 0,
      '10': 'endpointShareReceive'
    },
    {
      '1': 'endpoint_share_commit',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointShareCommitRequest',
      '9': 0,
      '10': 'endpointShareCommit'
    },
    {
      '1': 'ssh_credential_provision',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialProvisionRequest',
      '9': 0,
      '10': 'sshCredentialProvision'
    },
    {
      '1': 'connection_policy_get',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyGetRequest',
      '9': 0,
      '10': 'connectionPolicyGet'
    },
    {
      '1': 'connection_policy_apply',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyApplyRequest',
      '9': 0,
      '10': 'connectionPolicyApply'
    },
    {
      '1': 'connection_snapshot_get',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionSnapshotGetRequest',
      '9': 0,
      '10': 'connectionSnapshotGet'
    },
    {
      '1': 'session_invalidate',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SessionInvalidateRequest',
      '9': 0,
      '10': 'sessionInvalidate'
    },
    {
      '1': 'endpoint_disconnect',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointDisconnectRequest',
      '9': 0,
      '10': 'endpointDisconnect'
    },
    {
      '1': 'endpoint_cloud_presence_get',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointCloudPresenceGetRequest',
      '9': 0,
      '10': 'endpointCloudPresenceGet'
    },
  ],
  '8': [
    {'1': 'command'},
  ],
};

/// Descriptor for `EngineCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List engineCommandDescriptor = $convert.base64Decode(
    'Cg1FbmdpbmVDb21tYW5kElcKDmltcG9ydF9wYWlyaW5nGAEgASgLMi4uYW55dHR5LmNsaWVudC'
    '5iaW5kaW5nLnYxLkltcG9ydFBhaXJpbmdSZXF1ZXN0SABSDWltcG9ydFBhaXJpbmcSYAoRZGVs'
    'ZXRlX2NyZWRlbnRpYWwYAiABKAsyMS5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRGVsZXRlQ3'
    'JlZGVudGlhbFJlcXVlc3RIAFIQZGVsZXRlQ3JlZGVudGlhbBJqChVlbmRwb2ludF9yZWdpc3Ry'
    'eV9nZXQYAyABKAsyNC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRW5kcG9pbnRSZWdpc3RyeU'
    'dldFJlcXVlc3RIAFITZW5kcG9pbnRSZWdpc3RyeUdldBJaCg9lbmRwb2ludF91cHNlcnQYBCAB'
    'KAsyLy5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRW5kcG9pbnRVcHNlcnRSZXF1ZXN0SABSDm'
    'VuZHBvaW50VXBzZXJ0EloKD2VuZHBvaW50X2RlbGV0ZRgFIAEoCzIvLmFueXR0eS5jbGllbnQu'
    'YmluZGluZy52MS5FbmRwb2ludERlbGV0ZVJlcXVlc3RIAFIOZW5kcG9pbnREZWxldGUSbQoWZW'
    '5kcG9pbnRfc2hhcmVfcmVjZWl2ZRgGIAEoCzI1LmFueXR0eS5jbGllbnQuYmluZGluZy52MS5F'
    'bmRwb2ludFNoYXJlUmVjZWl2ZVJlcXVlc3RIAFIUZW5kcG9pbnRTaGFyZVJlY2VpdmUSagoVZW'
    '5kcG9pbnRfc2hhcmVfY29tbWl0GAcgASgLMjQuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkVu'
    'ZHBvaW50U2hhcmVDb21taXRSZXF1ZXN0SABSE2VuZHBvaW50U2hhcmVDb21taXQScwoYc3NoX2'
    'NyZWRlbnRpYWxfcHJvdmlzaW9uGAggASgLMjcuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLlNT'
    'SENyZWRlbnRpYWxQcm92aXNpb25SZXF1ZXN0SABSFnNzaENyZWRlbnRpYWxQcm92aXNpb24Sag'
    'oVY29ubmVjdGlvbl9wb2xpY3lfZ2V0GAkgASgLMjQuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYx'
    'LkNvbm5lY3Rpb25Qb2xpY3lHZXRSZXF1ZXN0SABSE2Nvbm5lY3Rpb25Qb2xpY3lHZXQScAoXY2'
    '9ubmVjdGlvbl9wb2xpY3lfYXBwbHkYCiABKAsyNi5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEu'
    'Q29ubmVjdGlvblBvbGljeUFwcGx5UmVxdWVzdEgAUhVjb25uZWN0aW9uUG9saWN5QXBwbHkScA'
    'oXY29ubmVjdGlvbl9zbmFwc2hvdF9nZXQYCyABKAsyNi5hbnl0dHkuY2xpZW50LmJpbmRpbmcu'
    'djEuQ29ubmVjdGlvblNuYXBzaG90R2V0UmVxdWVzdEgAUhVjb25uZWN0aW9uU25hcHNob3RHZX'
    'QSYwoSc2Vzc2lvbl9pbnZhbGlkYXRlGAwgASgLMjIuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYx'
    'LlNlc3Npb25JbnZhbGlkYXRlUmVxdWVzdEgAUhFzZXNzaW9uSW52YWxpZGF0ZRJmChNlbmRwb2'
    'ludF9kaXNjb25uZWN0GA0gASgLMjMuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkVuZHBvaW50'
    'RGlzY29ubmVjdFJlcXVlc3RIAFISZW5kcG9pbnREaXNjb25uZWN0EnoKG2VuZHBvaW50X2Nsb3'
    'VkX3ByZXNlbmNlX2dldBgOIAEoCzI5LmFueXR0eS5jbGllbnQuYmluZGluZy52MS5FbmRwb2lu'
    'dENsb3VkUHJlc2VuY2VHZXRSZXF1ZXN0SABSGGVuZHBvaW50Q2xvdWRQcmVzZW5jZUdldEIJCg'
    'djb21tYW5k');

@$core.Deprecated('Use openSessionResultDescriptor instead')
const OpenSessionResult$json = {
  '1': 'OpenSessionResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'session_handle', '3': 3, '4': 1, '5': 4, '10': 'sessionHandle'},
    {
      '1': 'session',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {
      '1': 'error',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
    {
      '1': 'connection',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionSnapshot',
      '10': 'connection'
    },
  ],
};

/// Descriptor for `OpenSessionResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List openSessionResultDescriptor = $convert.base64Decode(
    'ChFPcGVuU2Vzc2lvblJlc3VsdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSKQoQb3'
    'BlcmF0aW9uX2hhbmRsZRgCIAEoBFIPb3BlcmF0aW9uSGFuZGxlEiUKDnNlc3Npb25faGFuZGxl'
    'GAMgASgEUg1zZXNzaW9uSGFuZGxlEj0KB3Nlc3Npb24YBCABKAsyIy5hbnl0dHkuYXBpLnYxLk'
    'VuZHBvaW50U2Vzc2lvblN0YW1wUgdzZXNzaW9uEi0KBWVycm9yGAUgASgLMhcuYW55dHR5LmFw'
    'aS52MS5BcGlFcnJvclIFZXJyb3ISTAoKY29ubmVjdGlvbhgGIAEoCzIsLmFueXR0eS5jbGllbn'
    'QuYmluZGluZy52MS5Db25uZWN0aW9uU25hcHNob3RSCmNvbm5lY3Rpb24=');

@$core.Deprecated('Use executeResultDescriptor instead')
const ExecuteResult$json = {
  '1': 'ExecuteResult',
  '2': [
    {'1': 'operation_handle', '3': 1, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'session_handle', '3': 2, '4': 1, '5': 4, '10': 'sessionHandle'},
    {
      '1': 'result',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResultEnvelope',
      '10': 'result'
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

/// Descriptor for `ExecuteResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List executeResultDescriptor = $convert.base64Decode(
    'Cg1FeGVjdXRlUmVzdWx0EikKEG9wZXJhdGlvbl9oYW5kbGUYASABKARSD29wZXJhdGlvbkhhbm'
    'RsZRIlCg5zZXNzaW9uX2hhbmRsZRgCIAEoBFINc2Vzc2lvbkhhbmRsZRI1CgZyZXN1bHQYAyAB'
    'KAsyHS5hbnl0dHkuYXBpLnYxLlJlc3VsdEVudmVsb3BlUgZyZXN1bHQSLQoFZXJyb3IYBCABKA'
    'syFy5hbnl0dHkuYXBpLnYxLkFwaUVycm9yUgVlcnJvcg==');

@$core.Deprecated('Use applicationEventDescriptor instead')
const ApplicationEvent$json = {
  '1': 'ApplicationEvent',
  '2': [
    {'1': 'session_handle', '3': 1, '4': 1, '5': 4, '10': 'sessionHandle'},
    {
      '1': 'event',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EventEnvelope',
      '10': 'event'
    },
  ],
};

/// Descriptor for `ApplicationEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List applicationEventDescriptor = $convert.base64Decode(
    'ChBBcHBsaWNhdGlvbkV2ZW50EiUKDnNlc3Npb25faGFuZGxlGAEgASgEUg1zZXNzaW9uSGFuZG'
    'xlEjIKBWV2ZW50GAIgASgLMhwuYW55dHR5LmFwaS52MS5FdmVudEVudmVsb3BlUgVldmVudA==');

@$core.Deprecated('Use openResourceStreamRequestDescriptor instead')
const OpenResourceStreamRequest$json = {
  '1': 'OpenResourceStreamRequest',
  '2': [
    {
      '1': 'resource',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
    {
      '1': 'initial_upload_offset',
      '3': 2,
      '4': 1,
      '5': 3,
      '10': 'initialUploadOffset'
    },
  ],
};

/// Descriptor for `OpenResourceStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List openResourceStreamRequestDescriptor = $convert.base64Decode(
    'ChlPcGVuUmVzb3VyY2VTdHJlYW1SZXF1ZXN0EjkKCHJlc291cmNlGAEgASgLMh0uYW55dHR5Lm'
    'FwaS52MS5SZXNvdXJjZUhhbmRsZVIIcmVzb3VyY2USMgoVaW5pdGlhbF91cGxvYWRfb2Zmc2V0'
    'GAIgASgDUhNpbml0aWFsVXBsb2FkT2Zmc2V0');

@$core.Deprecated('Use resourceStreamFrameDescriptor instead')
const ResourceStreamFrame$json = {
  '1': 'ResourceStreamFrame',
  '2': [
    {'1': 'stream_handle', '3': 1, '4': 1, '5': 4, '10': 'streamHandle'},
    {
      '1': 'type',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ResourceStreamFrameType',
      '10': 'type'
    },
    {'1': 'payload', '3': 3, '4': 1, '5': 12, '10': 'payload'},
  ],
};

/// Descriptor for `ResourceStreamFrame`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resourceStreamFrameDescriptor = $convert.base64Decode(
    'ChNSZXNvdXJjZVN0cmVhbUZyYW1lEiMKDXN0cmVhbV9oYW5kbGUYASABKARSDHN0cmVhbUhhbm'
    'RsZRJFCgR0eXBlGAIgASgOMjEuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLlJlc291cmNlU3Ry'
    'ZWFtRnJhbWVUeXBlUgR0eXBlEhgKB3BheWxvYWQYAyABKAxSB3BheWxvYWQ=');

@$core.Deprecated('Use resourceStreamClosedEventDescriptor instead')
const ResourceStreamClosedEvent$json = {
  '1': 'ResourceStreamClosedEvent',
  '2': [
    {'1': 'stream_handle', '3': 1, '4': 1, '5': 4, '10': 'streamHandle'},
    {
      '1': 'error',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `ResourceStreamClosedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resourceStreamClosedEventDescriptor = $convert.base64Decode(
    'ChlSZXNvdXJjZVN0cmVhbUNsb3NlZEV2ZW50EiMKDXN0cmVhbV9oYW5kbGUYASABKARSDHN0cm'
    'VhbUhhbmRsZRItCgVlcnJvchgCIAEoCzIXLmFueXR0eS5hcGkudjEuQXBpRXJyb3JSBWVycm9y');

@$core.Deprecated('Use sessionClosedEventDescriptor instead')
const SessionClosedEvent$json = {
  '1': 'SessionClosedEvent',
  '2': [
    {'1': 'session_handle', '3': 1, '4': 1, '5': 4, '10': 'sessionHandle'},
    {
      '1': 'session',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {
      '1': 'error',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `SessionClosedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sessionClosedEventDescriptor = $convert.base64Decode(
    'ChJTZXNzaW9uQ2xvc2VkRXZlbnQSJQoOc2Vzc2lvbl9oYW5kbGUYASABKARSDXNlc3Npb25IYW'
    '5kbGUSPQoHc2Vzc2lvbhgCIAEoCzIjLmFueXR0eS5hcGkudjEuRW5kcG9pbnRTZXNzaW9uU3Rh'
    'bXBSB3Nlc3Npb24SLQoFZXJyb3IYAyABKAsyFy5hbnl0dHkuYXBpLnYxLkFwaUVycm9yUgVlcn'
    'Jvcg==');

@$core.Deprecated('Use endpointConnectionEventDescriptor instead')
const EndpointConnectionEvent$json = {
  '1': 'EndpointConnectionEvent',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'operation_handle', '3': 2, '4': 1, '5': 4, '10': 'operationHandle'},
    {'1': 'endpoint_id', '3': 3, '4': 1, '5': 9, '10': 'endpointId'},
    {
      '1': 'session',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {
      '1': 'phase',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.EndpointConnectionPhase',
      '10': 'phase'
    },
    {
      '1': 'observed_path',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionObservedPath',
      '10': 'observedPath'
    },
    {
      '1': 'route_selection_reason',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'routeSelectionReason'
    },
    {
      '1': 'error',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
    {
      '1': 'attempted_route_kind',
      '3': 9,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.ConnectionRouteKind',
      '10': 'attemptedRouteKind'
    },
    {'1': 'connection_stage', '3': 10, '4': 1, '5': 9, '10': 'connectionStage'},
  ],
};

/// Descriptor for `EndpointConnectionEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointConnectionEventDescriptor = $convert.base64Decode(
    'ChdFbmRwb2ludENvbm5lY3Rpb25FdmVudBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SW'
    'QSKQoQb3BlcmF0aW9uX2hhbmRsZRgCIAEoBFIPb3BlcmF0aW9uSGFuZGxlEh8KC2VuZHBvaW50'
    'X2lkGAMgASgJUgplbmRwb2ludElkEj0KB3Nlc3Npb24YBCABKAsyIy5hbnl0dHkuYXBpLnYxLk'
    'VuZHBvaW50U2Vzc2lvblN0YW1wUgdzZXNzaW9uEkcKBXBoYXNlGAUgASgOMjEuYW55dHR5LmNs'
    'aWVudC5iaW5kaW5nLnYxLkVuZHBvaW50Q29ubmVjdGlvblBoYXNlUgVwaGFzZRJVCg1vYnNlcn'
    'ZlZF9wYXRoGAYgASgOMjAuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkNvbm5lY3Rpb25PYnNl'
    'cnZlZFBhdGhSDG9ic2VydmVkUGF0aBI0ChZyb3V0ZV9zZWxlY3Rpb25fcmVhc29uGAcgASgJUh'
    'Ryb3V0ZVNlbGVjdGlvblJlYXNvbhItCgVlcnJvchgIIAEoCzIXLmFueXR0eS5hcGkudjEuQXBp'
    'RXJyb3JSBWVycm9yEl8KFGF0dGVtcHRlZF9yb3V0ZV9raW5kGAkgASgOMi0uYW55dHR5LmNsaW'
    'VudC5iaW5kaW5nLnYxLkNvbm5lY3Rpb25Sb3V0ZUtpbmRSEmF0dGVtcHRlZFJvdXRlS2luZBIp'
    'ChBjb25uZWN0aW9uX3N0YWdlGAogASgJUg9jb25uZWN0aW9uU3RhZ2U=');

@$core.Deprecated('Use eventEnvelopeDescriptor instead')
const EventEnvelope$json = {
  '1': 'EventEnvelope',
  '2': [
    {'1': 'abi_version', '3': 1, '4': 1, '5': 13, '10': 'abiVersion'},
    {'1': 'sequence', '3': 2, '4': 1, '5': 4, '10': 'sequence'},
    {
      '1': 'open_session',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.OpenSessionResult',
      '9': 0,
      '10': 'openSession'
    },
    {
      '1': 'execute',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ExecuteResult',
      '9': 0,
      '10': 'execute'
    },
    {
      '1': 'application',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ApplicationEvent',
      '9': 0,
      '10': 'application'
    },
    {
      '1': 'session_closed',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SessionClosedEvent',
      '9': 0,
      '10': 'sessionClosed'
    },
    {
      '1': 'import_pairing',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ImportPairingResult',
      '9': 0,
      '10': 'importPairing'
    },
    {
      '1': 'delete_credential',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.DeleteCredentialResult',
      '9': 0,
      '10': 'deleteCredential'
    },
    {
      '1': 'resource_stream_frame',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ResourceStreamFrame',
      '9': 0,
      '10': 'resourceStreamFrame'
    },
    {
      '1': 'resource_stream_closed',
      '3': 17,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ResourceStreamClosedEvent',
      '9': 0,
      '10': 'resourceStreamClosed'
    },
    {
      '1': 'endpoint_registry_get',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointRegistryGetResult',
      '9': 0,
      '10': 'endpointRegistryGet'
    },
    {
      '1': 'endpoint_upsert',
      '3': 19,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointUpsertResult',
      '9': 0,
      '10': 'endpointUpsert'
    },
    {
      '1': 'endpoint_delete',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointDeleteResult',
      '9': 0,
      '10': 'endpointDelete'
    },
    {
      '1': 'endpoint_share_receive',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointShareReceiveResult',
      '9': 0,
      '10': 'endpointShareReceive'
    },
    {
      '1': 'endpoint_share_commit',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointShareCommitResult',
      '9': 0,
      '10': 'endpointShareCommit'
    },
    {
      '1': 'ssh_credential_provision',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialProvisionResult',
      '9': 0,
      '10': 'sshCredentialProvision'
    },
    {
      '1': 'connection_policy_get',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyGetResult',
      '9': 0,
      '10': 'connectionPolicyGet'
    },
    {
      '1': 'connection_policy_apply',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionPolicyApplyResult',
      '9': 0,
      '10': 'connectionPolicyApply'
    },
    {
      '1': 'connection_snapshot_get',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.ConnectionSnapshotGetResult',
      '9': 0,
      '10': 'connectionSnapshotGet'
    },
    {
      '1': 'session_invalidate',
      '3': 27,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SessionInvalidateResult',
      '9': 0,
      '10': 'sessionInvalidate'
    },
    {
      '1': 'endpoint_connection',
      '3': 28,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointConnectionEvent',
      '9': 0,
      '10': 'endpointConnection'
    },
    {
      '1': 'endpoint_disconnect',
      '3': 29,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointDisconnectResult',
      '9': 0,
      '10': 'endpointDisconnect'
    },
    {
      '1': 'endpoint_cloud_presence_get',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointCloudPresenceGetResult',
      '9': 0,
      '10': 'endpointCloudPresenceGet'
    },
  ],
  '8': [
    {'1': 'event'},
  ],
};

/// Descriptor for `EventEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eventEnvelopeDescriptor = $convert.base64Decode(
    'Cg1FdmVudEVudmVsb3BlEh8KC2FiaV92ZXJzaW9uGAEgASgNUgphYmlWZXJzaW9uEhoKCHNlcX'
    'VlbmNlGAIgASgEUghzZXF1ZW5jZRJQCgxvcGVuX3Nlc3Npb24YCiABKAsyKy5hbnl0dHkuY2xp'
    'ZW50LmJpbmRpbmcudjEuT3BlblNlc3Npb25SZXN1bHRIAFILb3BlblNlc3Npb24SQwoHZXhlY3'
    'V0ZRgLIAEoCzInLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5FeGVjdXRlUmVzdWx0SABSB2V4'
    'ZWN1dGUSTgoLYXBwbGljYXRpb24YDCABKAsyKi5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQX'
    'BwbGljYXRpb25FdmVudEgAUgthcHBsaWNhdGlvbhJVCg5zZXNzaW9uX2Nsb3NlZBgNIAEoCzIs'
    'LmFueXR0eS5jbGllbnQuYmluZGluZy52MS5TZXNzaW9uQ2xvc2VkRXZlbnRIAFINc2Vzc2lvbk'
    'Nsb3NlZBJWCg5pbXBvcnRfcGFpcmluZxgOIAEoCzItLmFueXR0eS5jbGllbnQuYmluZGluZy52'
    'MS5JbXBvcnRQYWlyaW5nUmVzdWx0SABSDWltcG9ydFBhaXJpbmcSXwoRZGVsZXRlX2NyZWRlbn'
    'RpYWwYDyABKAsyMC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRGVsZXRlQ3JlZGVudGlhbFJl'
    'c3VsdEgAUhBkZWxldGVDcmVkZW50aWFsEmMKFXJlc291cmNlX3N0cmVhbV9mcmFtZRgQIAEoCz'
    'ItLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5SZXNvdXJjZVN0cmVhbUZyYW1lSABSE3Jlc291'
    'cmNlU3RyZWFtRnJhbWUSawoWcmVzb3VyY2Vfc3RyZWFtX2Nsb3NlZBgRIAEoCzIzLmFueXR0eS'
    '5jbGllbnQuYmluZGluZy52MS5SZXNvdXJjZVN0cmVhbUNsb3NlZEV2ZW50SABSFHJlc291cmNl'
    'U3RyZWFtQ2xvc2VkEmkKFWVuZHBvaW50X3JlZ2lzdHJ5X2dldBgSIAEoCzIzLmFueXR0eS5jbG'
    'llbnQuYmluZGluZy52MS5FbmRwb2ludFJlZ2lzdHJ5R2V0UmVzdWx0SABSE2VuZHBvaW50UmVn'
    'aXN0cnlHZXQSWQoPZW5kcG9pbnRfdXBzZXJ0GBMgASgLMi4uYW55dHR5LmNsaWVudC5iaW5kaW'
    '5nLnYxLkVuZHBvaW50VXBzZXJ0UmVzdWx0SABSDmVuZHBvaW50VXBzZXJ0ElkKD2VuZHBvaW50'
    'X2RlbGV0ZRgUIAEoCzIuLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5FbmRwb2ludERlbGV0ZV'
    'Jlc3VsdEgAUg5lbmRwb2ludERlbGV0ZRJsChZlbmRwb2ludF9zaGFyZV9yZWNlaXZlGBUgASgL'
    'MjQuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkVuZHBvaW50U2hhcmVSZWNlaXZlUmVzdWx0SA'
    'BSFGVuZHBvaW50U2hhcmVSZWNlaXZlEmkKFWVuZHBvaW50X3NoYXJlX2NvbW1pdBgWIAEoCzIz'
    'LmFueXR0eS5jbGllbnQuYmluZGluZy52MS5FbmRwb2ludFNoYXJlQ29tbWl0UmVzdWx0SABSE2'
    'VuZHBvaW50U2hhcmVDb21taXQScgoYc3NoX2NyZWRlbnRpYWxfcHJvdmlzaW9uGBcgASgLMjYu'
    'YW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLlNTSENyZWRlbnRpYWxQcm92aXNpb25SZXN1bHRIAF'
    'IWc3NoQ3JlZGVudGlhbFByb3Zpc2lvbhJpChVjb25uZWN0aW9uX3BvbGljeV9nZXQYGCABKAsy'
    'My5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ29ubmVjdGlvblBvbGljeUdldFJlc3VsdEgAUh'
    'Njb25uZWN0aW9uUG9saWN5R2V0Em8KF2Nvbm5lY3Rpb25fcG9saWN5X2FwcGx5GBkgASgLMjUu'
    'YW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkNvbm5lY3Rpb25Qb2xpY3lBcHBseVJlc3VsdEgAUh'
    'Vjb25uZWN0aW9uUG9saWN5QXBwbHkSbwoXY29ubmVjdGlvbl9zbmFwc2hvdF9nZXQYGiABKAsy'
    'NS5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ29ubmVjdGlvblNuYXBzaG90R2V0UmVzdWx0SA'
    'BSFWNvbm5lY3Rpb25TbmFwc2hvdEdldBJiChJzZXNzaW9uX2ludmFsaWRhdGUYGyABKAsyMS5h'
    'bnl0dHkuY2xpZW50LmJpbmRpbmcudjEuU2Vzc2lvbkludmFsaWRhdGVSZXN1bHRIAFIRc2Vzc2'
    'lvbkludmFsaWRhdGUSZAoTZW5kcG9pbnRfY29ubmVjdGlvbhgcIAEoCzIxLmFueXR0eS5jbGll'
    'bnQuYmluZGluZy52MS5FbmRwb2ludENvbm5lY3Rpb25FdmVudEgAUhJlbmRwb2ludENvbm5lY3'
    'Rpb24SZQoTZW5kcG9pbnRfZGlzY29ubmVjdBgdIAEoCzIyLmFueXR0eS5jbGllbnQuYmluZGlu'
    'Zy52MS5FbmRwb2ludERpc2Nvbm5lY3RSZXN1bHRIAFISZW5kcG9pbnREaXNjb25uZWN0EnkKG2'
    'VuZHBvaW50X2Nsb3VkX3ByZXNlbmNlX2dldBgeIAEoCzI4LmFueXR0eS5jbGllbnQuYmluZGlu'
    'Zy52MS5FbmRwb2ludENsb3VkUHJlc2VuY2VHZXRSZXN1bHRIAFIYZW5kcG9pbnRDbG91ZFByZX'
    'NlbmNlR2V0QgcKBWV2ZW50');

@$core.Deprecated('Use credentialResolveRequestDescriptor instead')
const CredentialResolveRequest$json = {
  '1': 'CredentialResolveRequest',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'credential_ref', '3': 2, '4': 1, '5': 9, '10': 'credentialRef'},
  ],
};

/// Descriptor for `CredentialResolveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialResolveRequestDescriptor =
    $convert.base64Decode(
        'ChhDcmVkZW50aWFsUmVzb2x2ZVJlcXVlc3QSHwoLZW5kcG9pbnRfaWQYASABKAlSCmVuZHBvaW'
        '50SWQSJQoOY3JlZGVudGlhbF9yZWYYAiABKAlSDWNyZWRlbnRpYWxSZWY=');

@$core.Deprecated('Use credentialPrepareRequestDescriptor instead')
const CredentialPrepareRequest$json = {
  '1': 'CredentialPrepareRequest',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'credential_ref', '3': 2, '4': 1, '5': 9, '10': 'credentialRef'},
  ],
};

/// Descriptor for `CredentialPrepareRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialPrepareRequestDescriptor =
    $convert.base64Decode(
        'ChhDcmVkZW50aWFsUHJlcGFyZVJlcXVlc3QSHwoLZW5kcG9pbnRfaWQYASABKAlSCmVuZHBvaW'
        '50SWQSJQoOY3JlZGVudGlhbF9yZWYYAiABKAlSDWNyZWRlbnRpYWxSZWY=');

@$core.Deprecated('Use credentialDeleteRequestDescriptor instead')
const CredentialDeleteRequest$json = {
  '1': 'CredentialDeleteRequest',
  '2': [
    {'1': 'credential_ref', '3': 1, '4': 1, '5': 9, '10': 'credentialRef'},
  ],
};

/// Descriptor for `CredentialDeleteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialDeleteRequestDescriptor =
    $convert.base64Decode(
        'ChdDcmVkZW50aWFsRGVsZXRlUmVxdWVzdBIlCg5jcmVkZW50aWFsX3JlZhgBIAEoCVINY3JlZG'
        'VudGlhbFJlZg==');

@$core.Deprecated('Use credentialBindRequestDescriptor instead')
const CredentialBindRequest$json = {
  '1': 'CredentialBindRequest',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'credential_ref', '3': 2, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'capability_grant', '3': 3, '4': 1, '5': 9, '10': 'capabilityGrant'},
    {
      '1': 'cloud_route_grant',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'cloudRouteGrant'
    },
    {
      '1': 'cloud_edge_locator',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'cloudEdgeLocator'
    },
  ],
};

/// Descriptor for `CredentialBindRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialBindRequestDescriptor = $convert.base64Decode(
    'ChVDcmVkZW50aWFsQmluZFJlcXVlc3QSHwoLZW5kcG9pbnRfaWQYASABKAlSCmVuZHBvaW50SW'
    'QSJQoOY3JlZGVudGlhbF9yZWYYAiABKAlSDWNyZWRlbnRpYWxSZWYSKQoQY2FwYWJpbGl0eV9n'
    'cmFudBgDIAEoCVIPY2FwYWJpbGl0eUdyYW50EioKEWNsb3VkX3JvdXRlX2dyYW50GAQgASgMUg'
    '9jbG91ZFJvdXRlR3JhbnQSLAoSY2xvdWRfZWRnZV9sb2NhdG9yGAUgASgMUhBjbG91ZEVkZ2VM'
    'b2NhdG9y');

@$core.Deprecated('Use credentialRecordDescriptor instead')
const CredentialRecord$json = {
  '1': 'CredentialRecord',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'credential_ref', '3': 2, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'public_key', '3': 3, '4': 1, '5': 12, '10': 'publicKey'},
    {'1': 'key_fingerprint', '3': 4, '4': 1, '5': 9, '10': 'keyFingerprint'},
    {'1': 'capability_grant', '3': 5, '4': 1, '5': 9, '10': 'capabilityGrant'},
    {'1': 'newly_created', '3': 6, '4': 1, '5': 8, '10': 'newlyCreated'},
    {
      '1': 'cloud_route_grant',
      '3': 7,
      '4': 1,
      '5': 12,
      '10': 'cloudRouteGrant'
    },
    {
      '1': 'cloud_edge_locator',
      '3': 8,
      '4': 1,
      '5': 12,
      '10': 'cloudEdgeLocator'
    },
  ],
};

/// Descriptor for `CredentialRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialRecordDescriptor = $convert.base64Decode(
    'ChBDcmVkZW50aWFsUmVjb3JkEh8KC2VuZHBvaW50X2lkGAEgASgJUgplbmRwb2ludElkEiUKDm'
    'NyZWRlbnRpYWxfcmVmGAIgASgJUg1jcmVkZW50aWFsUmVmEh0KCnB1YmxpY19rZXkYAyABKAxS'
    'CXB1YmxpY0tleRInCg9rZXlfZmluZ2VycHJpbnQYBCABKAlSDmtleUZpbmdlcnByaW50EikKEG'
    'NhcGFiaWxpdHlfZ3JhbnQYBSABKAlSD2NhcGFiaWxpdHlHcmFudBIjCg1uZXdseV9jcmVhdGVk'
    'GAYgASgIUgxuZXdseUNyZWF0ZWQSKgoRY2xvdWRfcm91dGVfZ3JhbnQYByABKAxSD2Nsb3VkUm'
    '91dGVHcmFudBIsChJjbG91ZF9lZGdlX2xvY2F0b3IYCCABKAxSEGNsb3VkRWRnZUxvY2F0b3I=');

@$core.Deprecated('Use credentialSignRequestDescriptor instead')
const CredentialSignRequest$json = {
  '1': 'CredentialSignRequest',
  '2': [
    {'1': 'credential_ref', '3': 1, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'payload', '3': 2, '4': 1, '5': 12, '10': 'payload'},
  ],
};

/// Descriptor for `CredentialSignRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialSignRequestDescriptor = $convert.base64Decode(
    'ChVDcmVkZW50aWFsU2lnblJlcXVlc3QSJQoOY3JlZGVudGlhbF9yZWYYASABKAlSDWNyZWRlbn'
    'RpYWxSZWYSGAoHcGF5bG9hZBgCIAEoDFIHcGF5bG9hZA==');

@$core.Deprecated('Use credentialSignResponseDescriptor instead')
const CredentialSignResponse$json = {
  '1': 'CredentialSignResponse',
  '2': [
    {'1': 'signature', '3': 1, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `CredentialSignResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List credentialSignResponseDescriptor =
    $convert.base64Decode(
        'ChZDcmVkZW50aWFsU2lnblJlc3BvbnNlEhwKCXNpZ25hdHVyZRgBIAEoDFIJc2lnbmF0dXJl');

@$core.Deprecated('Use cloudProfileResolveRequestDescriptor instead')
const CloudProfileResolveRequest$json = {
  '1': 'CloudProfileResolveRequest',
  '2': [
    {
      '1': 'account_profile_ref',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'accountProfileRef'
    },
  ],
};

/// Descriptor for `CloudProfileResolveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudProfileResolveRequestDescriptor =
    $convert.base64Decode(
        'ChpDbG91ZFByb2ZpbGVSZXNvbHZlUmVxdWVzdBIuChNhY2NvdW50X3Byb2ZpbGVfcmVmGAEgAS'
        'gJUhFhY2NvdW50UHJvZmlsZVJlZg==');

@$core.Deprecated('Use cloudProfileRecordDescriptor instead')
const CloudProfileRecord$json = {
  '1': 'CloudProfileRecord',
  '2': [
    {
      '1': 'account_profile_ref',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'accountProfileRef'
    },
    {
      '1': 'controller_address',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'controllerAddress'
    },
    {
      '1': 'controller_server_name',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'controllerServerName'
    },
    {
      '1': 'controller_ca_pem',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'controllerCaPem'
    },
  ],
};

/// Descriptor for `CloudProfileRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudProfileRecordDescriptor = $convert.base64Decode(
    'ChJDbG91ZFByb2ZpbGVSZWNvcmQSLgoTYWNjb3VudF9wcm9maWxlX3JlZhgBIAEoCVIRYWNjb3'
    'VudFByb2ZpbGVSZWYSLQoSY29udHJvbGxlcl9hZGRyZXNzGAIgASgJUhFjb250cm9sbGVyQWRk'
    'cmVzcxI0ChZjb250cm9sbGVyX3NlcnZlcl9uYW1lGAMgASgJUhRjb250cm9sbGVyU2VydmVyTm'
    'FtZRIqChFjb250cm9sbGVyX2NhX3BlbRgEIAEoDFIPY29udHJvbGxlckNhUGVt');

@$core.Deprecated('Use sSHCredentialLookupRequestDescriptor instead')
const SSHCredentialLookupRequest$json = {
  '1': 'SSHCredentialLookupRequest',
  '2': [
    {'1': 'credential_ref', '3': 1, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'create_if_missing', '3': 2, '4': 1, '5': 8, '10': 'createIfMissing'},
  ],
};

/// Descriptor for `SSHCredentialLookupRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialLookupRequestDescriptor =
    $convert.base64Decode(
        'ChpTU0hDcmVkZW50aWFsTG9va3VwUmVxdWVzdBIlCg5jcmVkZW50aWFsX3JlZhgBIAEoCVINY3'
        'JlZGVudGlhbFJlZhIqChFjcmVhdGVfaWZfbWlzc2luZxgCIAEoCFIPY3JlYXRlSWZNaXNzaW5n');

@$core.Deprecated('Use sSHCredentialDeleteRequestDescriptor instead')
const SSHCredentialDeleteRequest$json = {
  '1': 'SSHCredentialDeleteRequest',
  '2': [
    {'1': 'credential_ref', '3': 1, '4': 1, '5': 9, '10': 'credentialRef'},
  ],
};

/// Descriptor for `SSHCredentialDeleteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialDeleteRequestDescriptor =
    $convert.base64Decode(
        'ChpTU0hDcmVkZW50aWFsRGVsZXRlUmVxdWVzdBIlCg5jcmVkZW50aWFsX3JlZhgBIAEoCVINY3'
        'JlZGVudGlhbFJlZg==');

@$core.Deprecated('Use sSHCredentialRecordDescriptor instead')
const SSHCredentialRecord$json = {
  '1': 'SSHCredentialRecord',
  '2': [
    {'1': 'credential_ref', '3': 1, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'public_key_pkix', '3': 2, '4': 1, '5': 12, '10': 'publicKeyPkix'},
    {'1': 'newly_created', '3': 3, '4': 1, '5': 8, '10': 'newlyCreated'},
  ],
};

/// Descriptor for `SSHCredentialRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialRecordDescriptor = $convert.base64Decode(
    'ChNTU0hDcmVkZW50aWFsUmVjb3JkEiUKDmNyZWRlbnRpYWxfcmVmGAEgASgJUg1jcmVkZW50aW'
    'FsUmVmEiYKD3B1YmxpY19rZXlfcGtpeBgCIAEoDFINcHVibGljS2V5UGtpeBIjCg1uZXdseV9j'
    'cmVhdGVkGAMgASgIUgxuZXdseUNyZWF0ZWQ=');

@$core.Deprecated('Use sSHCredentialSignRequestDescriptor instead')
const SSHCredentialSignRequest$json = {
  '1': 'SSHCredentialSignRequest',
  '2': [
    {'1': 'credential_ref', '3': 1, '4': 1, '5': 9, '10': 'credentialRef'},
    {'1': 'digest', '3': 2, '4': 1, '5': 12, '10': 'digest'},
    {'1': 'hash', '3': 3, '4': 1, '5': 9, '10': 'hash'},
  ],
};

/// Descriptor for `SSHCredentialSignRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialSignRequestDescriptor = $convert.base64Decode(
    'ChhTU0hDcmVkZW50aWFsU2lnblJlcXVlc3QSJQoOY3JlZGVudGlhbF9yZWYYASABKAlSDWNyZW'
    'RlbnRpYWxSZWYSFgoGZGlnZXN0GAIgASgMUgZkaWdlc3QSEgoEaGFzaBgDIAEoCVIEaGFzaA==');

@$core.Deprecated('Use sSHCredentialSignResponseDescriptor instead')
const SSHCredentialSignResponse$json = {
  '1': 'SSHCredentialSignResponse',
  '2': [
    {'1': 'signature', '3': 1, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `SSHCredentialSignResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sSHCredentialSignResponseDescriptor =
    $convert.base64Decode(
        'ChlTU0hDcmVkZW50aWFsU2lnblJlc3BvbnNlEhwKCXNpZ25hdHVyZRgBIAEoDFIJc2lnbmF0dX'
        'Jl');

@$core.Deprecated('Use endpointRegistryLoadRequestDescriptor instead')
const EndpointRegistryLoadRequest$json = {
  '1': 'EndpointRegistryLoadRequest',
};

/// Descriptor for `EndpointRegistryLoadRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRegistryLoadRequestDescriptor =
    $convert.base64Decode('ChtFbmRwb2ludFJlZ2lzdHJ5TG9hZFJlcXVlc3Q=');

@$core.Deprecated('Use endpointRegistryStoreRequestDescriptor instead')
const EndpointRegistryStoreRequest$json = {
  '1': 'EndpointRegistryStoreRequest',
  '2': [
    {'1': 'registry_proto', '3': 1, '4': 1, '5': 12, '10': 'registryProto'},
    {
      '1': 'delete_credential_refs',
      '3': 2,
      '4': 3,
      '5': 9,
      '10': 'deleteCredentialRefs'
    },
  ],
};

/// Descriptor for `EndpointRegistryStoreRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRegistryStoreRequestDescriptor =
    $convert.base64Decode(
        'ChxFbmRwb2ludFJlZ2lzdHJ5U3RvcmVSZXF1ZXN0EiUKDnJlZ2lzdHJ5X3Byb3RvGAEgASgMUg'
        '1yZWdpc3RyeVByb3RvEjQKFmRlbGV0ZV9jcmVkZW50aWFsX3JlZnMYAiADKAlSFGRlbGV0ZUNy'
        'ZWRlbnRpYWxSZWZz');

@$core.Deprecated('Use endpointRegistryLoadedDescriptor instead')
const EndpointRegistryLoaded$json = {
  '1': 'EndpointRegistryLoaded',
  '2': [
    {'1': 'registry_proto', '3': 1, '4': 1, '5': 12, '10': 'registryProto'},
  ],
};

/// Descriptor for `EndpointRegistryLoaded`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointRegistryLoadedDescriptor =
    $convert.base64Decode(
        'ChZFbmRwb2ludFJlZ2lzdHJ5TG9hZGVkEiUKDnJlZ2lzdHJ5X3Byb3RvGAEgASgMUg1yZWdpc3'
        'RyeVByb3Rv');

@$core.Deprecated('Use localDiscoveryLookupRequestDescriptor instead')
const LocalDiscoveryLookupRequest$json = {
  '1': 'LocalDiscoveryLookupRequest',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'device_fingerprint',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'deviceFingerprint'
    },
  ],
};

/// Descriptor for `LocalDiscoveryLookupRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List localDiscoveryLookupRequestDescriptor =
    $convert.base64Decode(
        'ChtMb2NhbERpc2NvdmVyeUxvb2t1cFJlcXVlc3QSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2'
        'VJZBItChJkZXZpY2VfZmluZ2VycHJpbnQYAiABKAlSEWRldmljZUZpbmdlcnByaW50');

@$core.Deprecated('Use localDiscoveryCandidateDescriptor instead')
const LocalDiscoveryCandidate$json = {
  '1': 'LocalDiscoveryCandidate',
  '2': [
    {'1': 'address', '3': 1, '4': 1, '5': 9, '10': 'address'},
    {'1': 'port', '3': 2, '4': 1, '5': 13, '10': 'port'},
    {'1': 'protocol_version', '3': 3, '4': 1, '5': 13, '10': 'protocolVersion'},
    {
      '1': 'expires_at_unix_nano',
      '3': 4,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixNano'
    },
    {'1': 'network_handle', '3': 5, '4': 1, '5': 4, '10': 'networkHandle'},
  ],
};

/// Descriptor for `LocalDiscoveryCandidate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List localDiscoveryCandidateDescriptor = $convert.base64Decode(
    'ChdMb2NhbERpc2NvdmVyeUNhbmRpZGF0ZRIYCgdhZGRyZXNzGAEgASgJUgdhZGRyZXNzEhIKBH'
    'BvcnQYAiABKA1SBHBvcnQSKQoQcHJvdG9jb2xfdmVyc2lvbhgDIAEoDVIPcHJvdG9jb2xWZXJz'
    'aW9uEi8KFGV4cGlyZXNfYXRfdW5peF9uYW5vGAQgASgDUhFleHBpcmVzQXRVbml4TmFubxIlCg'
    '5uZXR3b3JrX2hhbmRsZRgFIAEoBFINbmV0d29ya0hhbmRsZQ==');

@$core.Deprecated('Use localDiscoveryLookupResultDescriptor instead')
const LocalDiscoveryLookupResult$json = {
  '1': 'LocalDiscoveryLookupResult',
  '2': [
    {
      '1': 'candidates',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.client.binding.v1.LocalDiscoveryCandidate',
      '10': 'candidates'
    },
  ],
};

/// Descriptor for `LocalDiscoveryLookupResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List localDiscoveryLookupResultDescriptor =
    $convert.base64Decode(
        'ChpMb2NhbERpc2NvdmVyeUxvb2t1cFJlc3VsdBJRCgpjYW5kaWRhdGVzGAEgAygLMjEuYW55dH'
        'R5LmNsaWVudC5iaW5kaW5nLnYxLkxvY2FsRGlzY292ZXJ5Q2FuZGlkYXRlUgpjYW5kaWRhdGVz');

@$core.Deprecated('Use platformEventDescriptor instead')
const PlatformEvent$json = {
  '1': 'PlatformEvent',
  '9': [
    {'1': 10, '2': 13},
  ],
};

/// Descriptor for `PlatformEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List platformEventDescriptor =
    $convert.base64Decode('Cg1QbGF0Zm9ybUV2ZW50SgQIChAN');

@$core.Deprecated('Use platformRequestDescriptor instead')
const PlatformRequest$json = {
  '1': 'PlatformRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
    {
      '1': 'credential_resolve',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialResolveRequest',
      '9': 0,
      '10': 'credentialResolve'
    },
    {
      '1': 'credential_prepare',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialPrepareRequest',
      '9': 0,
      '10': 'credentialPrepare'
    },
    {
      '1': 'credential_delete',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialDeleteRequest',
      '9': 0,
      '10': 'credentialDelete'
    },
    {
      '1': 'credential_sign',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialSignRequest',
      '9': 0,
      '10': 'credentialSign'
    },
    {
      '1': 'credential_bind',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialBindRequest',
      '9': 0,
      '10': 'credentialBind'
    },
    {
      '1': 'endpoint_registry_load',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointRegistryLoadRequest',
      '9': 0,
      '10': 'endpointRegistryLoad'
    },
    {
      '1': 'endpoint_registry_store',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointRegistryStoreRequest',
      '9': 0,
      '10': 'endpointRegistryStore'
    },
    {
      '1': 'ssh_credential_lookup',
      '3': 17,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialLookupRequest',
      '9': 0,
      '10': 'sshCredentialLookup'
    },
    {
      '1': 'ssh_credential_sign',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialSignRequest',
      '9': 0,
      '10': 'sshCredentialSign'
    },
    {
      '1': 'ssh_credential_delete',
      '3': 19,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialDeleteRequest',
      '9': 0,
      '10': 'sshCredentialDelete'
    },
    {
      '1': 'cloud_profile_resolve',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CloudProfileResolveRequest',
      '9': 0,
      '10': 'cloudProfileResolve'
    },
    {
      '1': 'local_discovery_lookup',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.LocalDiscoveryLookupRequest',
      '9': 0,
      '10': 'localDiscoveryLookup'
    },
  ],
  '8': [
    {'1': 'request'},
  ],
  '9': [
    {'1': 22, '2': 27},
    {'1': 30, '2': 39},
  ],
};

/// Descriptor for `PlatformRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List platformRequestDescriptor = $convert.base64Decode(
    'Cg9QbGF0Zm9ybVJlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoBFIJcmVxdWVzdElkEmMKEmNyZW'
    'RlbnRpYWxfcmVzb2x2ZRgKIAEoCzIyLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5DcmVkZW50'
    'aWFsUmVzb2x2ZVJlcXVlc3RIAFIRY3JlZGVudGlhbFJlc29sdmUSYwoSY3JlZGVudGlhbF9wcm'
    'VwYXJlGAsgASgLMjIuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkNyZWRlbnRpYWxQcmVwYXJl'
    'UmVxdWVzdEgAUhFjcmVkZW50aWFsUHJlcGFyZRJgChFjcmVkZW50aWFsX2RlbGV0ZRgMIAEoCz'
    'IxLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5DcmVkZW50aWFsRGVsZXRlUmVxdWVzdEgAUhBj'
    'cmVkZW50aWFsRGVsZXRlEloKD2NyZWRlbnRpYWxfc2lnbhgNIAEoCzIvLmFueXR0eS5jbGllbn'
    'QuYmluZGluZy52MS5DcmVkZW50aWFsU2lnblJlcXVlc3RIAFIOY3JlZGVudGlhbFNpZ24SWgoP'
    'Y3JlZGVudGlhbF9iaW5kGA4gASgLMi8uYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkNyZWRlbn'
    'RpYWxCaW5kUmVxdWVzdEgAUg5jcmVkZW50aWFsQmluZBJtChZlbmRwb2ludF9yZWdpc3RyeV9s'
    'b2FkGA8gASgLMjUuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkVuZHBvaW50UmVnaXN0cnlMb2'
    'FkUmVxdWVzdEgAUhRlbmRwb2ludFJlZ2lzdHJ5TG9hZBJwChdlbmRwb2ludF9yZWdpc3RyeV9z'
    'dG9yZRgQIAEoCzI2LmFueXR0eS5jbGllbnQuYmluZGluZy52MS5FbmRwb2ludFJlZ2lzdHJ5U3'
    'RvcmVSZXF1ZXN0SABSFWVuZHBvaW50UmVnaXN0cnlTdG9yZRJqChVzc2hfY3JlZGVudGlhbF9s'
    'b29rdXAYESABKAsyNC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuU1NIQ3JlZGVudGlhbExvb2'
    't1cFJlcXVlc3RIAFITc3NoQ3JlZGVudGlhbExvb2t1cBJkChNzc2hfY3JlZGVudGlhbF9zaWdu'
    'GBIgASgLMjIuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLlNTSENyZWRlbnRpYWxTaWduUmVxdW'
    'VzdEgAUhFzc2hDcmVkZW50aWFsU2lnbhJqChVzc2hfY3JlZGVudGlhbF9kZWxldGUYEyABKAsy'
    'NC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuU1NIQ3JlZGVudGlhbERlbGV0ZVJlcXVlc3RIAF'
    'ITc3NoQ3JlZGVudGlhbERlbGV0ZRJqChVjbG91ZF9wcm9maWxlX3Jlc29sdmUYFCABKAsyNC5h'
    'bnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ2xvdWRQcm9maWxlUmVzb2x2ZVJlcXVlc3RIAFITY2'
    'xvdWRQcm9maWxlUmVzb2x2ZRJtChZsb2NhbF9kaXNjb3ZlcnlfbG9va3VwGBUgASgLMjUuYW55'
    'dHR5LmNsaWVudC5iaW5kaW5nLnYxLkxvY2FsRGlzY292ZXJ5TG9va3VwUmVxdWVzdEgAUhRsb2'
    'NhbERpc2NvdmVyeUxvb2t1cEIJCgdyZXF1ZXN0SgQIFhAbSgQIHhAn');

@$core.Deprecated('Use platformResponseDescriptor instead')
const PlatformResponse$json = {
  '1': 'PlatformResponse',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
    {
      '1': 'error',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '10': 'error'
    },
    {
      '1': 'credential',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialRecord',
      '9': 0,
      '10': 'credential'
    },
    {
      '1': 'credential_sign',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CredentialSignResponse',
      '9': 0,
      '10': 'credentialSign'
    },
    {
      '1': 'endpoint_registry',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointRegistryLoaded',
      '9': 0,
      '10': 'endpointRegistry'
    },
    {
      '1': 'ssh_credential',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialRecord',
      '9': 0,
      '10': 'sshCredential'
    },
    {
      '1': 'ssh_credential_sign',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.SSHCredentialSignResponse',
      '9': 0,
      '10': 'sshCredentialSign'
    },
    {
      '1': 'cloud_profile',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.CloudProfileRecord',
      '9': 0,
      '10': 'cloudProfile'
    },
    {
      '1': 'local_discovery',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.anytty.client.binding.v1.LocalDiscoveryLookupResult',
      '9': 0,
      '10': 'localDiscovery'
    },
  ],
  '8': [
    {'1': 'response'},
  ],
  '9': [
    {'1': 20, '2': 27},
    {'1': 30, '2': 35},
  ],
};

/// Descriptor for `PlatformResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List platformResponseDescriptor = $convert.base64Decode(
    'ChBQbGF0Zm9ybVJlc3BvbnNlEh0KCnJlcXVlc3RfaWQYASABKARSCXJlcXVlc3RJZBItCgVlcn'
    'JvchgCIAEoCzIXLmFueXR0eS5hcGkudjEuQXBpRXJyb3JSBWVycm9yEkwKCmNyZWRlbnRpYWwY'
    'CiABKAsyKi5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ3JlZGVudGlhbFJlY29yZEgAUgpjcm'
    'VkZW50aWFsElsKD2NyZWRlbnRpYWxfc2lnbhgLIAEoCzIwLmFueXR0eS5jbGllbnQuYmluZGlu'
    'Zy52MS5DcmVkZW50aWFsU2lnblJlc3BvbnNlSABSDmNyZWRlbnRpYWxTaWduEl8KEWVuZHBvaW'
    '50X3JlZ2lzdHJ5GAwgASgLMjAuYW55dHR5LmNsaWVudC5iaW5kaW5nLnYxLkVuZHBvaW50UmVn'
    'aXN0cnlMb2FkZWRIAFIQZW5kcG9pbnRSZWdpc3RyeRJWCg5zc2hfY3JlZGVudGlhbBgNIAEoCz'
    'ItLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5TU0hDcmVkZW50aWFsUmVjb3JkSABSDXNzaENy'
    'ZWRlbnRpYWwSZQoTc3NoX2NyZWRlbnRpYWxfc2lnbhgOIAEoCzIzLmFueXR0eS5jbGllbnQuYm'
    'luZGluZy52MS5TU0hDcmVkZW50aWFsU2lnblJlc3BvbnNlSABSEXNzaENyZWRlbnRpYWxTaWdu'
    'ElMKDWNsb3VkX3Byb2ZpbGUYDyABKAsyLC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuQ2xvdW'
    'RQcm9maWxlUmVjb3JkSABSDGNsb3VkUHJvZmlsZRJfCg9sb2NhbF9kaXNjb3ZlcnkYECABKAsy'
    'NC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuTG9jYWxEaXNjb3ZlcnlMb29rdXBSZXN1bHRIAF'
    'IObG9jYWxEaXNjb3ZlcnlCCgoIcmVzcG9uc2VKBAgUEBtKBAgeECM=');

@$core.Deprecated('Use pTYStreamSyncLostDescriptor instead')
const PTYStreamSyncLost$json = {
  '1': 'PTYStreamSyncLost',
  '2': [
    {'1': 'dropped_bytes', '3': 1, '4': 1, '5': 4, '10': 'droppedBytes'},
  ],
};

/// Descriptor for `PTYStreamSyncLost`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pTYStreamSyncLostDescriptor = $convert.base64Decode(
    'ChFQVFlTdHJlYW1TeW5jTG9zdBIjCg1kcm9wcGVkX2J5dGVzGAEgASgEUgxkcm9wcGVkQnl0ZX'
    'M=');

@$core.Deprecated('Use pTYStreamClosedDescriptor instead')
const PTYStreamClosed$json = {
  '1': 'PTYStreamClosed',
  '2': [
    {'1': 'exit_code', '3': 1, '4': 1, '5': 5, '10': 'exitCode'},
  ],
};

/// Descriptor for `PTYStreamClosed`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pTYStreamClosedDescriptor = $convert.base64Decode(
    'Cg9QVFlTdHJlYW1DbG9zZWQSGwoJZXhpdF9jb2RlGAEgASgFUghleGl0Q29kZQ==');

@$core.Deprecated('Use endpointSupervisorDemandDescriptor instead')
const EndpointSupervisorDemand$json = {
  '1': 'EndpointSupervisorDemand',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {
      '1': 'mode',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.EndpointSupervisorMode',
      '10': 'mode'
    },
  ],
};

/// Descriptor for `EndpointSupervisorDemand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSupervisorDemandDescriptor = $convert.base64Decode(
    'ChhFbmRwb2ludFN1cGVydmlzb3JEZW1hbmQSHwoLZW5kcG9pbnRfaWQYASABKAlSCmVuZHBvaW'
    '50SWQSRAoEbW9kZRgCIAEoDjIwLmFueXR0eS5jbGllbnQuYmluZGluZy52MS5FbmRwb2ludFN1'
    'cGVydmlzb3JNb2RlUgRtb2Rl');

@$core.Deprecated('Use endpointSupervisorDemandSnapshotDescriptor instead')
const EndpointSupervisorDemandSnapshot$json = {
  '1': 'EndpointSupervisorDemandSnapshot',
  '2': [
    {'1': 'attachment_id', '3': 1, '4': 1, '5': 9, '10': 'attachmentId'},
    {'1': 'demand_revision', '3': 2, '4': 1, '5': 4, '10': 'demandRevision'},
    {
      '1': 'endpoints',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointSupervisorDemand',
      '10': 'endpoints'
    },
  ],
};

/// Descriptor for `EndpointSupervisorDemandSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSupervisorDemandSnapshotDescriptor =
    $convert.base64Decode(
        'CiBFbmRwb2ludFN1cGVydmlzb3JEZW1hbmRTbmFwc2hvdBIjCg1hdHRhY2htZW50X2lkGAEgAS'
        'gJUgxhdHRhY2htZW50SWQSJwoPZGVtYW5kX3JldmlzaW9uGAIgASgEUg5kZW1hbmRSZXZpc2lv'
        'bhJQCgllbmRwb2ludHMYAyADKAsyMi5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRW5kcG9pbn'
        'RTdXBlcnZpc29yRGVtYW5kUgllbmRwb2ludHM=');

@$core.Deprecated('Use endpointSupervisorHostSignalDescriptor instead')
const EndpointSupervisorHostSignal$json = {
  '1': 'EndpointSupervisorHostSignal',
  '2': [
    {'1': 'revision', '3': 1, '4': 1, '5': 4, '10': 'revision'},
    {'1': 'connected', '3': 2, '4': 1, '5': 8, '10': 'connected'},
    {'1': 'reason', '3': 3, '4': 1, '5': 9, '10': 'reason'},
    {'1': 'foreground', '3': 4, '4': 1, '5': 8, '10': 'foreground'},
  ],
};

/// Descriptor for `EndpointSupervisorHostSignal`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSupervisorHostSignalDescriptor =
    $convert.base64Decode(
        'ChxFbmRwb2ludFN1cGVydmlzb3JIb3N0U2lnbmFsEhoKCHJldmlzaW9uGAEgASgEUghyZXZpc2'
        'lvbhIcCgljb25uZWN0ZWQYAiABKAhSCWNvbm5lY3RlZBIWCgZyZWFzb24YAyABKAlSBnJlYXNv'
        'bhIeCgpmb3JlZ3JvdW5kGAQgASgIUgpmb3JlZ3JvdW5k');

@$core.Deprecated('Use endpointSupervisorProjectionDescriptor instead')
const EndpointSupervisorProjection$json = {
  '1': 'EndpointSupervisorProjection',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {
      '1': 'mode',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.client.binding.v1.EndpointSupervisorMode',
      '10': 'mode'
    },
    {'1': 'phase', '3': 3, '4': 1, '5': 9, '10': 'phase'},
    {'1': 'control_revision', '3': 4, '4': 1, '5': 4, '10': 'controlRevision'},
    {'1': 'attempt_id', '3': 5, '4': 1, '5': 4, '10': 'attemptId'},
    {
      '1': 'session',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {'1': 'error_code', '3': 7, '4': 1, '5': 9, '10': 'errorCode'},
    {'1': 'message', '3': 8, '4': 1, '5': 9, '10': 'message'},
    {'1': 'probe_count', '3': 9, '4': 1, '5': 4, '10': 'probeCount'},
    {'1': 'dial_count', '3': 10, '4': 1, '5': 4, '10': 'dialCount'},
    {'1': 'backoff_count', '3': 11, '4': 1, '5': 4, '10': 'backoffCount'},
  ],
};

/// Descriptor for `EndpointSupervisorProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSupervisorProjectionDescriptor = $convert.base64Decode(
    'ChxFbmRwb2ludFN1cGVydmlzb3JQcm9qZWN0aW9uEh8KC2VuZHBvaW50X2lkGAEgASgJUgplbm'
    'Rwb2ludElkEkQKBG1vZGUYAiABKA4yMC5hbnl0dHkuY2xpZW50LmJpbmRpbmcudjEuRW5kcG9p'
    'bnRTdXBlcnZpc29yTW9kZVIEbW9kZRIUCgVwaGFzZRgDIAEoCVIFcGhhc2USKQoQY29udHJvbF'
    '9yZXZpc2lvbhgEIAEoBFIPY29udHJvbFJldmlzaW9uEh0KCmF0dGVtcHRfaWQYBSABKARSCWF0'
    'dGVtcHRJZBI9CgdzZXNzaW9uGAYgASgLMiMuYW55dHR5LmFwaS52MS5FbmRwb2ludFNlc3Npb2'
    '5TdGFtcFIHc2Vzc2lvbhIdCgplcnJvcl9jb2RlGAcgASgJUgllcnJvckNvZGUSGAoHbWVzc2Fn'
    'ZRgIIAEoCVIHbWVzc2FnZRIfCgtwcm9iZV9jb3VudBgJIAEoBFIKcHJvYmVDb3VudBIdCgpkaW'
    'FsX2NvdW50GAogASgEUglkaWFsQ291bnQSIwoNYmFja29mZl9jb3VudBgLIAEoBFIMYmFja29m'
    'ZkNvdW50');

@$core.Deprecated('Use endpointSupervisorSnapshotDescriptor instead')
const EndpointSupervisorSnapshot$json = {
  '1': 'EndpointSupervisorSnapshot',
  '2': [
    {
      '1': 'endpoints',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.client.binding.v1.EndpointSupervisorProjection',
      '10': 'endpoints'
    },
  ],
};

/// Descriptor for `EndpointSupervisorSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSupervisorSnapshotDescriptor =
    $convert.base64Decode(
        'ChpFbmRwb2ludFN1cGVydmlzb3JTbmFwc2hvdBJUCgllbmRwb2ludHMYASADKAsyNi5hbnl0dH'
        'kuY2xpZW50LmJpbmRpbmcudjEuRW5kcG9pbnRTdXBlcnZpc29yUHJvamVjdGlvblIJZW5kcG9p'
        'bnRz');
