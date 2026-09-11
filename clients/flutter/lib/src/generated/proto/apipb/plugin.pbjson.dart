// This is a generated file - do not edit.
//
// Generated from apipb/plugin.proto.

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

@$core.Deprecated('Use pluginAddressDescriptor instead')
const PluginAddress$json = {
  '1': 'PluginAddress',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'tui_instance_id', '3': 2, '4': 1, '5': 9, '10': 'tuiInstanceId'},
    {'1': 'plugin_id', '3': 3, '4': 1, '5': 9, '10': 'pluginId'},
    {
      '1': 'plugin_instance_id',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'pluginInstanceId'
    },
    {
      '1': 'registration_epoch',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'registrationEpoch'
    },
  ],
};

/// Descriptor for `PluginAddress`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginAddressDescriptor = $convert.base64Decode(
    'Cg1QbHVnaW5BZGRyZXNzEhsKCWRhZW1vbl9pZBgBIAEoCVIIZGFlbW9uSWQSJgoPdHVpX2luc3'
    'RhbmNlX2lkGAIgASgJUg10dWlJbnN0YW5jZUlkEhsKCXBsdWdpbl9pZBgDIAEoCVIIcGx1Z2lu'
    'SWQSLAoScGx1Z2luX2luc3RhbmNlX2lkGAQgASgJUhBwbHVnaW5JbnN0YW5jZUlkEi0KEnJlZ2'
    'lzdHJhdGlvbl9lcG9jaBgFIAEoBFIRcmVnaXN0cmF0aW9uRXBvY2g=');

@$core.Deprecated('Use pluginRegisterRequestDescriptor instead')
const PluginRegisterRequest$json = {
  '1': 'PluginRegisterRequest',
  '2': [
    {
      '1': 'address',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAddress',
      '10': 'address'
    },
    {'1': 'topics', '3': 2, '4': 3, '5': 9, '10': 'topics'},
    {
      '1': 'allow_peer_messages',
      '3': 3,
      '4': 1,
      '5': 8,
      '10': 'allowPeerMessages'
    },
    {'1': 'daemon_service', '3': 4, '4': 1, '5': 8, '10': 'daemonService'},
    {'1': 'previous_lease', '3': 5, '4': 1, '5': 12, '10': 'previousLease'},
  ],
};

/// Descriptor for `PluginRegisterRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginRegisterRequestDescriptor = $convert.base64Decode(
    'ChVQbHVnaW5SZWdpc3RlclJlcXVlc3QSNgoHYWRkcmVzcxgBIAEoCzIcLmFueXR0eS5hcGkudj'
    'EuUGx1Z2luQWRkcmVzc1IHYWRkcmVzcxIWCgZ0b3BpY3MYAiADKAlSBnRvcGljcxIuChNhbGxv'
    'd19wZWVyX21lc3NhZ2VzGAMgASgIUhFhbGxvd1BlZXJNZXNzYWdlcxIlCg5kYWVtb25fc2Vydm'
    'ljZRgEIAEoCFINZGFlbW9uU2VydmljZRIlCg5wcmV2aW91c19sZWFzZRgFIAEoDFINcHJldmlv'
    'dXNMZWFzZQ==');

@$core.Deprecated('Use pluginRegistrationDescriptor instead')
const PluginRegistration$json = {
  '1': 'PluginRegistration',
  '2': [
    {
      '1': 'address',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAddress',
      '10': 'address'
    },
    {'1': 'source_lease', '3': 2, '4': 1, '5': 12, '10': 'sourceLease'},
    {'1': 'boot_epoch', '3': 3, '4': 1, '5': 9, '10': 'bootEpoch'},
    {
      '1': 'peers',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginAddress',
      '10': 'peers'
    },
  ],
};

/// Descriptor for `PluginRegistration`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginRegistrationDescriptor = $convert.base64Decode(
    'ChJQbHVnaW5SZWdpc3RyYXRpb24SNgoHYWRkcmVzcxgBIAEoCzIcLmFueXR0eS5hcGkudjEuUG'
    'x1Z2luQWRkcmVzc1IHYWRkcmVzcxIhCgxzb3VyY2VfbGVhc2UYAiABKAxSC3NvdXJjZUxlYXNl'
    'Eh0KCmJvb3RfZXBvY2gYAyABKAlSCWJvb3RFcG9jaBIyCgVwZWVycxgEIAMoCzIcLmFueXR0eS'
    '5hcGkudjEuUGx1Z2luQWRkcmVzc1IFcGVlcnM=');

@$core.Deprecated('Use pluginSendRequestDescriptor instead')
const PluginSendRequest$json = {
  '1': 'PluginSendRequest',
  '2': [
    {'1': 'source_lease', '3': 1, '4': 1, '5': 12, '10': 'sourceLease'},
    {
      '1': 'message',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginMessage',
      '10': 'message'
    },
  ],
};

/// Descriptor for `PluginSendRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginSendRequestDescriptor = $convert.base64Decode(
    'ChFQbHVnaW5TZW5kUmVxdWVzdBIhCgxzb3VyY2VfbGVhc2UYASABKAxSC3NvdXJjZUxlYXNlEj'
    'YKB21lc3NhZ2UYAiABKAsyHC5hbnl0dHkuYXBpLnYxLlBsdWdpbk1lc3NhZ2VSB21lc3NhZ2U=');

@$core.Deprecated('Use pluginReceiveRequestDescriptor instead')
const PluginReceiveRequest$json = {
  '1': 'PluginReceiveRequest',
  '2': [
    {'1': 'source_lease', '3': 1, '4': 1, '5': 12, '10': 'sourceLease'},
    {'1': 'wait_millis', '3': 2, '4': 1, '5': 13, '10': 'waitMillis'},
    {'1': 'max_messages', '3': 3, '4': 1, '5': 13, '10': 'maxMessages'},
  ],
};

/// Descriptor for `PluginReceiveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginReceiveRequestDescriptor = $convert.base64Decode(
    'ChRQbHVnaW5SZWNlaXZlUmVxdWVzdBIhCgxzb3VyY2VfbGVhc2UYASABKAxSC3NvdXJjZUxlYX'
    'NlEh8KC3dhaXRfbWlsbGlzGAIgASgNUgp3YWl0TWlsbGlzEiEKDG1heF9tZXNzYWdlcxgDIAEo'
    'DVILbWF4TWVzc2FnZXM=');

@$core.Deprecated('Use pluginUnregisterRequestDescriptor instead')
const PluginUnregisterRequest$json = {
  '1': 'PluginUnregisterRequest',
  '2': [
    {'1': 'source_lease', '3': 1, '4': 1, '5': 12, '10': 'sourceLease'},
  ],
};

/// Descriptor for `PluginUnregisterRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUnregisterRequestDescriptor =
    $convert.base64Decode(
        'ChdQbHVnaW5VbnJlZ2lzdGVyUmVxdWVzdBIhCgxzb3VyY2VfbGVhc2UYASABKAxSC3NvdXJjZU'
        'xlYXNl');

@$core.Deprecated('Use pluginBatchDescriptor instead')
const PluginBatch$json = {
  '1': 'PluginBatch',
  '2': [
    {
      '1': 'messages',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginMessage',
      '10': 'messages'
    },
    {'1': 'resync_required', '3': 2, '4': 1, '5': 8, '10': 'resyncRequired'},
  ],
};

/// Descriptor for `PluginBatch`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginBatchDescriptor = $convert.base64Decode(
    'CgtQbHVnaW5CYXRjaBI4CghtZXNzYWdlcxgBIAMoCzIcLmFueXR0eS5hcGkudjEuUGx1Z2luTW'
    'Vzc2FnZVIIbWVzc2FnZXMSJwoPcmVzeW5jX3JlcXVpcmVkGAIgASgIUg5yZXN5bmNSZXF1aXJl'
    'ZA==');

@$core.Deprecated('Use pluginAckDescriptor instead')
const PluginAck$json = {
  '1': 'PluginAck',
  '2': [
    {'1': 'delivered', '3': 1, '4': 1, '5': 13, '10': 'delivered'},
  ],
};

/// Descriptor for `PluginAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginAckDescriptor = $convert
    .base64Decode('CglQbHVnaW5BY2sSHAoJZGVsaXZlcmVkGAEgASgNUglkZWxpdmVyZWQ=');

@$core.Deprecated('Use pluginErrorDescriptor instead')
const PluginError$json = {
  '1': 'PluginError',
  '2': [
    {'1': 'code', '3': 1, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `PluginError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginErrorDescriptor = $convert.base64Decode(
    'CgtQbHVnaW5FcnJvchISCgRjb2RlGAEgASgJUgRjb2RlEhgKB21lc3NhZ2UYAiABKAlSB21lc3'
    'NhZ2U=');

@$core.Deprecated('Use pluginCommandDescriptor instead')
const PluginCommand$json = {
  '1': 'PluginCommand',
  '2': [
    {
      '1': 'register',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginRegisterRequest',
      '9': 0,
      '10': 'register'
    },
    {
      '1': 'send',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginSendRequest',
      '9': 0,
      '10': 'send'
    },
    {
      '1': 'receive',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginReceiveRequest',
      '9': 0,
      '10': 'receive'
    },
    {
      '1': 'unregister',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUnregisterRequest',
      '9': 0,
      '10': 'unregister'
    },
    {
      '1': 'state',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginStateRequest',
      '9': 0,
      '10': 'state'
    },
  ],
  '8': [
    {'1': 'command'},
  ],
};

/// Descriptor for `PluginCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginCommandDescriptor = $convert.base64Decode(
    'Cg1QbHVnaW5Db21tYW5kEkIKCHJlZ2lzdGVyGAEgASgLMiQuYW55dHR5LmFwaS52MS5QbHVnaW'
    '5SZWdpc3RlclJlcXVlc3RIAFIIcmVnaXN0ZXISNgoEc2VuZBgCIAEoCzIgLmFueXR0eS5hcGku'
    'djEuUGx1Z2luU2VuZFJlcXVlc3RIAFIEc2VuZBI/CgdyZWNlaXZlGAMgASgLMiMuYW55dHR5Lm'
    'FwaS52MS5QbHVnaW5SZWNlaXZlUmVxdWVzdEgAUgdyZWNlaXZlEkgKCnVucmVnaXN0ZXIYBCAB'
    'KAsyJi5hbnl0dHkuYXBpLnYxLlBsdWdpblVucmVnaXN0ZXJSZXF1ZXN0SABSCnVucmVnaXN0ZX'
    'ISOQoFc3RhdGUYBSABKAsyIS5hbnl0dHkuYXBpLnYxLlBsdWdpblN0YXRlUmVxdWVzdEgAUgVz'
    'dGF0ZUIJCgdjb21tYW5k');

@$core.Deprecated('Use pluginResultDescriptor instead')
const PluginResult$json = {
  '1': 'PluginResult',
  '2': [
    {
      '1': 'registration',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginRegistration',
      '9': 0,
      '10': 'registration'
    },
    {
      '1': 'ack',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAck',
      '9': 0,
      '10': 'ack'
    },
    {
      '1': 'batch',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginBatch',
      '9': 0,
      '10': 'batch'
    },
    {
      '1': 'error',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginError',
      '9': 0,
      '10': 'error'
    },
    {
      '1': 'state',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginStateSnapshot',
      '9': 0,
      '10': 'state'
    },
  ],
  '8': [
    {'1': 'result'},
  ],
};

/// Descriptor for `PluginResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginResultDescriptor = $convert.base64Decode(
    'CgxQbHVnaW5SZXN1bHQSRwoMcmVnaXN0cmF0aW9uGAEgASgLMiEuYW55dHR5LmFwaS52MS5QbH'
    'VnaW5SZWdpc3RyYXRpb25IAFIMcmVnaXN0cmF0aW9uEiwKA2FjaxgCIAEoCzIYLmFueXR0eS5h'
    'cGkudjEuUGx1Z2luQWNrSABSA2FjaxIyCgViYXRjaBgDIAEoCzIaLmFueXR0eS5hcGkudjEuUG'
    'x1Z2luQmF0Y2hIAFIFYmF0Y2gSMgoFZXJyb3IYBCABKAsyGi5hbnl0dHkuYXBpLnYxLlBsdWdp'
    'bkVycm9ySABSBWVycm9yEjoKBXN0YXRlGAUgASgLMiIuYW55dHR5LmFwaS52MS5QbHVnaW5TdG'
    'F0ZVNuYXBzaG90SABSBXN0YXRlQggKBnJlc3VsdA==');

@$core.Deprecated('Use pluginMessageDescriptor instead')
const PluginMessage$json = {
  '1': 'PluginMessage',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'trace_id', '3': 2, '4': 1, '5': 9, '10': 'traceId'},
    {
      '1': 'source',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAddress',
      '10': 'source'
    },
    {
      '1': 'destination',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAddress',
      '10': 'destination'
    },
    {'1': 'topic', '3': 5, '4': 1, '5': 9, '10': 'topic'},
    {
      '1': 'deadline_unix_millis',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'deadlineUnixMillis'
    },
    {'1': 'idempotency_key', '3': 7, '4': 1, '5': 9, '10': 'idempotencyKey'},
    {
      '1': 'delivery_sequence',
      '3': 8,
      '4': 1,
      '5': 4,
      '10': 'deliverySequence'
    },
    {
      '1': 'interaction',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiInteraction',
      '9': 0,
      '10': 'interaction'
    },
    {
      '1': 'operation',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiOperation',
      '9': 0,
      '10': 'operation'
    },
    {
      '1': 'mount_update',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiMountUpdate',
      '9': 0,
      '10': 'mountUpdate'
    },
    {
      '1': 'reply',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginReply',
      '9': 0,
      '10': 'reply'
    },
    {
      '1': 'agent_report',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAgentReport',
      '9': 0,
      '10': 'agentReport'
    },
    {
      '1': 'agent_snapshot',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAgentSnapshot',
      '9': 0,
      '10': 'agentSnapshot'
    },
    {
      '1': 'payload',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginPayload',
      '9': 0,
      '10': 'payload'
    },
    {
      '1': 'state_changed',
      '3': 27,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginStateSnapshot',
      '9': 0,
      '10': 'stateChanged'
    },
    {
      '1': 'init',
      '3': 28,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiInit',
      '9': 0,
      '10': 'init'
    },
    {
      '1': 'ui_query',
      '3': 29,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiQuery',
      '9': 0,
      '10': 'uiQuery'
    },
  ],
  '8': [
    {'1': 'body'},
  ],
};

/// Descriptor for `PluginMessage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginMessageDescriptor = $convert.base64Decode(
    'Cg1QbHVnaW5NZXNzYWdlEh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZBIZCgh0cmFjZV'
    '9pZBgCIAEoCVIHdHJhY2VJZBI0CgZzb3VyY2UYAyABKAsyHC5hbnl0dHkuYXBpLnYxLlBsdWdp'
    'bkFkZHJlc3NSBnNvdXJjZRI+CgtkZXN0aW5hdGlvbhgEIAEoCzIcLmFueXR0eS5hcGkudjEuUG'
    'x1Z2luQWRkcmVzc1ILZGVzdGluYXRpb24SFAoFdG9waWMYBSABKAlSBXRvcGljEjAKFGRlYWRs'
    'aW5lX3VuaXhfbWlsbGlzGAYgASgDUhJkZWFkbGluZVVuaXhNaWxsaXMSJwoPaWRlbXBvdGVuY3'
    'lfa2V5GAcgASgJUg5pZGVtcG90ZW5jeUtleRIrChFkZWxpdmVyeV9zZXF1ZW5jZRgIIAEoBFIQ'
    'ZGVsaXZlcnlTZXF1ZW5jZRJGCgtpbnRlcmFjdGlvbhgUIAEoCzIiLmFueXR0eS5hcGkudjEuUG'
    'x1Z2luVWlJbnRlcmFjdGlvbkgAUgtpbnRlcmFjdGlvbhJACglvcGVyYXRpb24YFSABKAsyIC5h'
    'bnl0dHkuYXBpLnYxLlBsdWdpblVpT3BlcmF0aW9uSABSCW9wZXJhdGlvbhJHCgxtb3VudF91cG'
    'RhdGUYFiABKAsyIi5hbnl0dHkuYXBpLnYxLlBsdWdpblVpTW91bnRVcGRhdGVIAFILbW91bnRV'
    'cGRhdGUSMgoFcmVwbHkYFyABKAsyGi5hbnl0dHkuYXBpLnYxLlBsdWdpblJlcGx5SABSBXJlcG'
    'x5EkUKDGFnZW50X3JlcG9ydBgYIAEoCzIgLmFueXR0eS5hcGkudjEuUGx1Z2luQWdlbnRSZXBv'
    'cnRIAFILYWdlbnRSZXBvcnQSSwoOYWdlbnRfc25hcHNob3QYGSABKAsyIi5hbnl0dHkuYXBpLn'
    'YxLlBsdWdpbkFnZW50U25hcHNob3RIAFINYWdlbnRTbmFwc2hvdBI4CgdwYXlsb2FkGBogASgL'
    'MhwuYW55dHR5LmFwaS52MS5QbHVnaW5QYXlsb2FkSABSB3BheWxvYWQSSQoNc3RhdGVfY2hhbm'
    'dlZBgbIAEoCzIiLmFueXR0eS5hcGkudjEuUGx1Z2luU3RhdGVTbmFwc2hvdEgAUgxzdGF0ZUNo'
    'YW5nZWQSMQoEaW5pdBgcIAEoCzIbLmFueXR0eS5hcGkudjEuUGx1Z2luVWlJbml0SABSBGluaX'
    'QSOQoIdWlfcXVlcnkYHSABKAsyHC5hbnl0dHkuYXBpLnYxLlBsdWdpblVpUXVlcnlIAFIHdWlR'
    'dWVyeUIGCgRib2R5');

@$core.Deprecated('Use pluginPayloadDescriptor instead')
const PluginPayload$json = {
  '1': 'PluginPayload',
  '2': [
    {'1': 'schema', '3': 1, '4': 1, '5': 9, '10': 'schema'},
    {'1': 'version', '3': 2, '4': 1, '5': 13, '10': 'version'},
    {'1': 'data', '3': 3, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `PluginPayload`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginPayloadDescriptor = $convert.base64Decode(
    'Cg1QbHVnaW5QYXlsb2FkEhYKBnNjaGVtYRgBIAEoCVIGc2NoZW1hEhgKB3ZlcnNpb24YAiABKA'
    '1SB3ZlcnNpb24SEgoEZGF0YRgDIAEoDFIEZGF0YQ==');

@$core.Deprecated('Use pluginReplyDescriptor instead')
const PluginReply$json = {
  '1': 'PluginReply',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'error',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginError',
      '10': 'error'
    },
    {
      '1': 'value',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginPayload',
      '10': 'value'
    },
    {
      '1': 'ui_snapshot',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiSnapshot',
      '10': 'uiSnapshot'
    },
  ],
};

/// Descriptor for `PluginReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginReplyDescriptor = $convert.base64Decode(
    'CgtQbHVnaW5SZXBseRIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSMAoFZXJyb3IYAi'
    'ABKAsyGi5hbnl0dHkuYXBpLnYxLlBsdWdpbkVycm9yUgVlcnJvchIyCgV2YWx1ZRgDIAEoCzIc'
    'LmFueXR0eS5hcGkudjEuUGx1Z2luUGF5bG9hZFIFdmFsdWUSQAoLdWlfc25hcHNob3QYBCABKA'
    'syHy5hbnl0dHkuYXBpLnYxLlBsdWdpblVpU25hcHNob3RSCnVpU25hcHNob3Q=');

@$core.Deprecated('Use pluginTerminalRefDescriptor instead')
const PluginTerminalRef$json = {
  '1': 'PluginTerminalRef',
  '2': [
    {'1': 'daemon_id', '3': 1, '4': 1, '5': 9, '10': 'daemonId'},
    {'1': 'terminal_id', '3': 2, '4': 1, '5': 9, '10': 'terminalId'},
  ],
};

/// Descriptor for `PluginTerminalRef`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginTerminalRefDescriptor = $convert.base64Decode(
    'ChFQbHVnaW5UZXJtaW5hbFJlZhIbCglkYWVtb25faWQYASABKAlSCGRhZW1vbklkEh8KC3Rlcm'
    '1pbmFsX2lkGAIgASgJUgp0ZXJtaW5hbElk');

@$core.Deprecated('Use pluginWorkspaceOwnerDescriptor instead')
const PluginWorkspaceOwner$json = {
  '1': 'PluginWorkspaceOwner',
  '2': [
    {'1': 'workspace_id', '3': 1, '4': 1, '5': 9, '10': 'workspaceId'},
  ],
};

/// Descriptor for `PluginWorkspaceOwner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginWorkspaceOwnerDescriptor = $convert.base64Decode(
    'ChRQbHVnaW5Xb3Jrc3BhY2VPd25lchIhCgx3b3Jrc3BhY2VfaWQYASABKAlSC3dvcmtzcGFjZU'
    'lk');

@$core.Deprecated('Use pluginTabOwnerDescriptor instead')
const PluginTabOwner$json = {
  '1': 'PluginTabOwner',
  '2': [
    {'1': 'workspace_id', '3': 1, '4': 1, '5': 9, '10': 'workspaceId'},
    {'1': 'tab_id', '3': 2, '4': 1, '5': 9, '10': 'tabId'},
  ],
};

/// Descriptor for `PluginTabOwner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginTabOwnerDescriptor = $convert.base64Decode(
    'Cg5QbHVnaW5UYWJPd25lchIhCgx3b3Jrc3BhY2VfaWQYASABKAlSC3dvcmtzcGFjZUlkEhUKBn'
    'RhYl9pZBgCIAEoCVIFdGFiSWQ=');

@$core.Deprecated('Use pluginPanelOwnerDescriptor instead')
const PluginPanelOwner$json = {
  '1': 'PluginPanelOwner',
  '2': [
    {'1': 'workspace_id', '3': 1, '4': 1, '5': 9, '10': 'workspaceId'},
    {'1': 'tab_id', '3': 2, '4': 1, '5': 9, '10': 'tabId'},
    {'1': 'pane_id', '3': 3, '4': 1, '5': 9, '10': 'paneId'},
  ],
};

/// Descriptor for `PluginPanelOwner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginPanelOwnerDescriptor = $convert.base64Decode(
    'ChBQbHVnaW5QYW5lbE93bmVyEiEKDHdvcmtzcGFjZV9pZBgBIAEoCVILd29ya3NwYWNlSWQSFQ'
    'oGdGFiX2lkGAIgASgJUgV0YWJJZBIXCgdwYW5lX2lkGAMgASgJUgZwYW5lSWQ=');

@$core.Deprecated('Use pluginFloatingOwnerDescriptor instead')
const PluginFloatingOwner$json = {
  '1': 'PluginFloatingOwner',
  '2': [
    {'1': 'workspace_id', '3': 1, '4': 1, '5': 9, '10': 'workspaceId'},
    {'1': 'floating_id', '3': 2, '4': 1, '5': 9, '10': 'floatingId'},
    {'1': 'tab_id', '3': 3, '4': 1, '5': 9, '10': 'tabId'},
  ],
};

/// Descriptor for `PluginFloatingOwner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginFloatingOwnerDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5GbG9hdGluZ093bmVyEiEKDHdvcmtzcGFjZV9pZBgBIAEoCVILd29ya3NwYWNlSW'
    'QSHwoLZmxvYXRpbmdfaWQYAiABKAlSCmZsb2F0aW5nSWQSFQoGdGFiX2lkGAMgASgJUgV0YWJJ'
    'ZA==');

@$core.Deprecated('Use pluginMountOwnerDescriptor instead')
const PluginMountOwner$json = {
  '1': 'PluginMountOwner',
  '2': [
    {
      '1': 'workspace',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginWorkspaceOwner',
      '9': 0,
      '10': 'workspace'
    },
    {
      '1': 'tab',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTabOwner',
      '9': 0,
      '10': 'tab'
    },
    {
      '1': 'panel',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginPanelOwner',
      '9': 0,
      '10': 'panel'
    },
    {
      '1': 'floating',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginFloatingOwner',
      '9': 0,
      '10': 'floating'
    },
  ],
  '8': [
    {'1': 'owner'},
  ],
};

/// Descriptor for `PluginMountOwner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginMountOwnerDescriptor = $convert.base64Decode(
    'ChBQbHVnaW5Nb3VudE93bmVyEkMKCXdvcmtzcGFjZRgBIAEoCzIjLmFueXR0eS5hcGkudjEuUG'
    'x1Z2luV29ya3NwYWNlT3duZXJIAFIJd29ya3NwYWNlEjEKA3RhYhgCIAEoCzIdLmFueXR0eS5h'
    'cGkudjEuUGx1Z2luVGFiT3duZXJIAFIDdGFiEjcKBXBhbmVsGAMgASgLMh8uYW55dHR5LmFwaS'
    '52MS5QbHVnaW5QYW5lbE93bmVySABSBXBhbmVsEkAKCGZsb2F0aW5nGAQgASgLMiIuYW55dHR5'
    'LmFwaS52MS5QbHVnaW5GbG9hdGluZ093bmVySABSCGZsb2F0aW5nQgcKBW93bmVy');

@$core.Deprecated('Use pluginTargetContextDescriptor instead')
const PluginTargetContext$json = {
  '1': 'PluginTargetContext',
  '2': [
    {'1': 'context_id', '3': 1, '4': 1, '5': 9, '10': 'contextId'},
    {'1': 'tui_instance_id', '3': 2, '4': 1, '5': 9, '10': 'tuiInstanceId'},
    {'1': 'workspace_id', '3': 3, '4': 1, '5': 9, '10': 'workspaceId'},
    {'1': 'tab_id', '3': 4, '4': 1, '5': 9, '10': 'tabId'},
    {'1': 'pane_id', '3': 5, '4': 1, '5': 9, '10': 'paneId'},
    {'1': 'binding_revision', '3': 6, '4': 1, '5': 4, '10': 'bindingRevision'},
    {'1': 'mount_id', '3': 7, '4': 1, '5': 9, '10': 'mountId'},
    {'1': 'floating_id', '3': 8, '4': 1, '5': 9, '10': 'floatingId'},
  ],
};

/// Descriptor for `PluginTargetContext`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginTargetContextDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5UYXJnZXRDb250ZXh0Eh0KCmNvbnRleHRfaWQYASABKAlSCWNvbnRleHRJZBImCg'
    '90dWlfaW5zdGFuY2VfaWQYAiABKAlSDXR1aUluc3RhbmNlSWQSIQoMd29ya3NwYWNlX2lkGAMg'
    'ASgJUgt3b3Jrc3BhY2VJZBIVCgZ0YWJfaWQYBCABKAlSBXRhYklkEhcKB3BhbmVfaWQYBSABKA'
    'lSBnBhbmVJZBIpChBiaW5kaW5nX3JldmlzaW9uGAYgASgEUg9iaW5kaW5nUmV2aXNpb24SGQoI'
    'bW91bnRfaWQYByABKAlSB21vdW50SWQSHwoLZmxvYXRpbmdfaWQYCCABKAlSCmZsb2F0aW5nSW'
    'Q=');

@$core.Deprecated('Use pluginUiInteractionDescriptor instead')
const PluginUiInteraction$json = {
  '1': 'PluginUiInteraction',
  '2': [
    {'1': 'mount_id', '3': 1, '4': 1, '5': 9, '10': 'mountId'},
    {'1': 'mount_revision', '3': 2, '4': 1, '5': 4, '10': 'mountRevision'},
    {'1': 'node_id', '3': 3, '4': 1, '5': 9, '10': 'nodeId'},
    {'1': 'action_id', '3': 4, '4': 1, '5': 9, '10': 'actionId'},
    {'1': 'item_id', '3': 5, '4': 1, '5': 9, '10': 'itemId'},
    {'1': 'value', '3': 6, '4': 1, '5': 9, '10': 'value'},
    {
      '1': 'context',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTargetContext',
      '10': 'context'
    },
    {'1': 'kind', '3': 8, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'modifiers', '3': 9, '4': 3, '5': 9, '10': 'modifiers'},
    {
      '1': 'values',
      '3': 10,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiInteraction.ValuesEntry',
      '10': 'values'
    },
  ],
  '3': [PluginUiInteraction_ValuesEntry$json],
};

@$core.Deprecated('Use pluginUiInteractionDescriptor instead')
const PluginUiInteraction_ValuesEntry$json = {
  '1': 'ValuesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `PluginUiInteraction`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiInteractionDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5VaUludGVyYWN0aW9uEhkKCG1vdW50X2lkGAEgASgJUgdtb3VudElkEiUKDm1vdW'
    '50X3JldmlzaW9uGAIgASgEUg1tb3VudFJldmlzaW9uEhcKB25vZGVfaWQYAyABKAlSBm5vZGVJ'
    'ZBIbCglhY3Rpb25faWQYBCABKAlSCGFjdGlvbklkEhcKB2l0ZW1faWQYBSABKAlSBml0ZW1JZB'
    'IUCgV2YWx1ZRgGIAEoCVIFdmFsdWUSPAoHY29udGV4dBgHIAEoCzIiLmFueXR0eS5hcGkudjEu'
    'UGx1Z2luVGFyZ2V0Q29udGV4dFIHY29udGV4dBISCgRraW5kGAggASgJUgRraW5kEhwKCW1vZG'
    'lmaWVycxgJIAMoCVIJbW9kaWZpZXJzEkYKBnZhbHVlcxgKIAMoCzIuLmFueXR0eS5hcGkudjEu'
    'UGx1Z2luVWlJbnRlcmFjdGlvbi5WYWx1ZXNFbnRyeVIGdmFsdWVzGjkKC1ZhbHVlc0VudHJ5Eh'
    'AKA2tleRgBIAEoCVIDa2V5EhQKBXZhbHVlGAIgASgJUgV2YWx1ZToCOAE=');

@$core.Deprecated('Use pluginPaneBindDescriptor instead')
const PluginPaneBind$json = {
  '1': 'PluginPaneBind',
  '2': [
    {
      '1': 'terminal',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTerminalRef',
      '10': 'terminal'
    },
  ],
};

/// Descriptor for `PluginPaneBind`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginPaneBindDescriptor = $convert.base64Decode(
    'Cg5QbHVnaW5QYW5lQmluZBI8Cgh0ZXJtaW5hbBgBIAEoCzIgLmFueXR0eS5hcGkudjEuUGx1Z2'
    'luVGVybWluYWxSZWZSCHRlcm1pbmFs');

@$core.Deprecated('Use pluginUiNotificationDescriptor instead')
const PluginUiNotification$json = {
  '1': 'PluginUiNotification',
  '2': [
    {'1': 'title', '3': 1, '4': 1, '5': 9, '10': 'title'},
    {'1': 'body', '3': 2, '4': 1, '5': 9, '10': 'body'},
    {'1': 'severity', '3': 3, '4': 1, '5': 9, '10': 'severity'},
  ],
};

/// Descriptor for `PluginUiNotification`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiNotificationDescriptor = $convert.base64Decode(
    'ChRQbHVnaW5VaU5vdGlmaWNhdGlvbhIUCgV0aXRsZRgBIAEoCVIFdGl0bGUSEgoEYm9keRgCIA'
    'EoCVIEYm9keRIaCghzZXZlcml0eRgDIAEoCVIIc2V2ZXJpdHk=');

@$core.Deprecated('Use pluginUiOperationDescriptor instead')
const PluginUiOperation$json = {
  '1': 'PluginUiOperation',
  '2': [
    {
      '1': 'context',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTargetContext',
      '10': 'context'
    },
    {
      '1': 'bind',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginPaneBind',
      '9': 0,
      '10': 'bind'
    },
    {
      '1': 'notification',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiNotification',
      '9': 0,
      '10': 'notification'
    },
  ],
  '8': [
    {'1': 'operation'},
  ],
};

/// Descriptor for `PluginUiOperation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiOperationDescriptor = $convert.base64Decode(
    'ChFQbHVnaW5VaU9wZXJhdGlvbhI8Cgdjb250ZXh0GAEgASgLMiIuYW55dHR5LmFwaS52MS5QbH'
    'VnaW5UYXJnZXRDb250ZXh0Ugdjb250ZXh0EjMKBGJpbmQYCiABKAsyHS5hbnl0dHkuYXBpLnYx'
    'LlBsdWdpblBhbmVCaW5kSABSBGJpbmQSSQoMbm90aWZpY2F0aW9uGAsgASgLMiMuYW55dHR5Lm'
    'FwaS52MS5QbHVnaW5VaU5vdGlmaWNhdGlvbkgAUgxub3RpZmljYXRpb25CCwoJb3BlcmF0aW9u');

@$core.Deprecated('Use pluginUiActionDescriptor instead')
const PluginUiAction$json = {
  '1': 'PluginUiAction',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'label', '3': 2, '4': 1, '5': 9, '10': 'label'},
    {'1': 'default_key', '3': 3, '4': 1, '5': 9, '10': 'defaultKey'},
    {'1': 'enabled', '3': 4, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'scope', '3': 5, '4': 1, '5': 9, '10': 'scope'},
  ],
};

/// Descriptor for `PluginUiAction`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiActionDescriptor = $convert.base64Decode(
    'Cg5QbHVnaW5VaUFjdGlvbhIOCgJpZBgBIAEoCVICaWQSFAoFbGFiZWwYAiABKAlSBWxhYmVsEh'
    '8KC2RlZmF1bHRfa2V5GAMgASgJUgpkZWZhdWx0S2V5EhgKB2VuYWJsZWQYBCABKAhSB2VuYWJs'
    'ZWQSFAoFc2NvcGUYBSABKAlSBXNjb3Bl');

@$core.Deprecated('Use pluginUiStyleDescriptor instead')
const PluginUiStyle$json = {
  '1': 'PluginUiStyle',
  '2': [
    {'1': 'foreground_role', '3': 1, '4': 1, '5': 9, '10': 'foregroundRole'},
    {'1': 'background_role', '3': 2, '4': 1, '5': 9, '10': 'backgroundRole'},
    {'1': 'bold', '3': 3, '4': 1, '5': 8, '10': 'bold'},
    {'1': 'dim', '3': 4, '4': 1, '5': 8, '10': 'dim'},
  ],
};

/// Descriptor for `PluginUiStyle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiStyleDescriptor = $convert.base64Decode(
    'Cg1QbHVnaW5VaVN0eWxlEicKD2ZvcmVncm91bmRfcm9sZRgBIAEoCVIOZm9yZWdyb3VuZFJvbG'
    'USJwoPYmFja2dyb3VuZF9yb2xlGAIgASgJUg5iYWNrZ3JvdW5kUm9sZRISCgRib2xkGAMgASgI'
    'UgRib2xkEhAKA2RpbRgEIAEoCFIDZGlt');

@$core.Deprecated('Use pluginUiLayoutDescriptor instead')
const PluginUiLayout$json = {
  '1': 'PluginUiLayout',
  '2': [
    {'1': 'padding_top', '3': 1, '4': 1, '5': 13, '10': 'paddingTop'},
    {'1': 'padding_right', '3': 2, '4': 1, '5': 13, '10': 'paddingRight'},
    {'1': 'padding_bottom', '3': 3, '4': 1, '5': 13, '10': 'paddingBottom'},
    {'1': 'padding_left', '3': 4, '4': 1, '5': 13, '10': 'paddingLeft'},
    {'1': 'gap_after', '3': 5, '4': 1, '5': 13, '10': 'gapAfter'},
  ],
};

/// Descriptor for `PluginUiLayout`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiLayoutDescriptor = $convert.base64Decode(
    'Cg5QbHVnaW5VaUxheW91dBIfCgtwYWRkaW5nX3RvcBgBIAEoDVIKcGFkZGluZ1RvcBIjCg1wYW'
    'RkaW5nX3JpZ2h0GAIgASgNUgxwYWRkaW5nUmlnaHQSJQoOcGFkZGluZ19ib3R0b20YAyABKA1S'
    'DXBhZGRpbmdCb3R0b20SIQoMcGFkZGluZ19sZWZ0GAQgASgNUgtwYWRkaW5nTGVmdBIbCglnYX'
    'BfYWZ0ZXIYBSABKA1SCGdhcEFmdGVy');

@$core.Deprecated('Use pluginUiNodeDescriptor instead')
const PluginUiNode$json = {
  '1': 'PluginUiNode',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'kind', '3': 2, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'text', '3': 3, '4': 1, '5': 9, '10': 'text'},
    {'1': 'status', '3': 4, '4': 1, '5': 9, '10': 'status'},
    {'1': 'action_id', '3': 5, '4': 1, '5': 9, '10': 'actionId'},
    {'1': 'item_id', '3': 6, '4': 1, '5': 9, '10': 'itemId'},
    {'1': 'disabled', '3': 7, '4': 1, '5': 8, '10': 'disabled'},
    {
      '1': 'children',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiNode',
      '10': 'children'
    },
    {'1': 'progress', '3': 9, '4': 1, '5': 1, '10': 'progress'},
    {
      '1': 'terminal',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTerminalRef',
      '10': 'terminal'
    },
    {'1': 'value', '3': 11, '4': 1, '5': 9, '10': 'value'},
    {'1': 'placeholder', '3': 12, '4': 1, '5': 9, '10': 'placeholder'},
    {
      '1': 'style',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiStyle',
      '10': 'style'
    },
    {
      '1': 'selected_style',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiStyle',
      '10': 'selectedStyle'
    },
    {
      '1': 'layout',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiLayout',
      '10': 'layout'
    },
    {'1': 'description', '3': 16, '4': 1, '5': 9, '10': 'description'},
  ],
};

/// Descriptor for `PluginUiNode`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiNodeDescriptor = $convert.base64Decode(
    'CgxQbHVnaW5VaU5vZGUSDgoCaWQYASABKAlSAmlkEhIKBGtpbmQYAiABKAlSBGtpbmQSEgoEdG'
    'V4dBgDIAEoCVIEdGV4dBIWCgZzdGF0dXMYBCABKAlSBnN0YXR1cxIbCglhY3Rpb25faWQYBSAB'
    'KAlSCGFjdGlvbklkEhcKB2l0ZW1faWQYBiABKAlSBml0ZW1JZBIaCghkaXNhYmxlZBgHIAEoCF'
    'IIZGlzYWJsZWQSNwoIY2hpbGRyZW4YCCADKAsyGy5hbnl0dHkuYXBpLnYxLlBsdWdpblVpTm9k'
    'ZVIIY2hpbGRyZW4SGgoIcHJvZ3Jlc3MYCSABKAFSCHByb2dyZXNzEjwKCHRlcm1pbmFsGAogAS'
    'gLMiAuYW55dHR5LmFwaS52MS5QbHVnaW5UZXJtaW5hbFJlZlIIdGVybWluYWwSFAoFdmFsdWUY'
    'CyABKAlSBXZhbHVlEiAKC3BsYWNlaG9sZGVyGAwgASgJUgtwbGFjZWhvbGRlchIyCgVzdHlsZR'
    'gNIAEoCzIcLmFueXR0eS5hcGkudjEuUGx1Z2luVWlTdHlsZVIFc3R5bGUSQwoOc2VsZWN0ZWRf'
    'c3R5bGUYDiABKAsyHC5hbnl0dHkuYXBpLnYxLlBsdWdpblVpU3R5bGVSDXNlbGVjdGVkU3R5bG'
    'USNQoGbGF5b3V0GA8gASgLMh0uYW55dHR5LmFwaS52MS5QbHVnaW5VaUxheW91dFIGbGF5b3V0'
    'EiAKC2Rlc2NyaXB0aW9uGBAgASgJUgtkZXNjcmlwdGlvbg==');

@$core.Deprecated('Use pluginUiMountUpdateDescriptor instead')
const PluginUiMountUpdate$json = {
  '1': 'PluginUiMountUpdate',
  '2': [
    {'1': 'mount_id', '3': 1, '4': 1, '5': 9, '10': 'mountId'},
    {
      '1': 'owner',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginMountOwner',
      '10': 'owner'
    },
    {'1': 'slot', '3': 3, '4': 1, '5': 9, '10': 'slot'},
    {
      '1': 'expected_revision',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {'1': 'revision', '3': 5, '4': 1, '5': 4, '10': 'revision'},
    {'1': 'title', '3': 6, '4': 1, '5': 9, '10': 'title'},
    {
      '1': 'root',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiNode',
      '10': 'root'
    },
    {
      '1': 'actions',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiAction',
      '10': 'actions'
    },
    {'1': 'close', '3': 9, '4': 1, '5': 8, '10': 'close'},
    {'1': 'focus', '3': 10, '4': 1, '5': 8, '10': 'focus'},
    {'1': 'preferred_width', '3': 11, '4': 1, '5': 13, '10': 'preferredWidth'},
    {'1': 'min_width', '3': 12, '4': 1, '5': 13, '10': 'minWidth'},
  ],
};

/// Descriptor for `PluginUiMountUpdate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiMountUpdateDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5VaU1vdW50VXBkYXRlEhkKCG1vdW50X2lkGAEgASgJUgdtb3VudElkEjUKBW93bm'
    'VyGAIgASgLMh8uYW55dHR5LmFwaS52MS5QbHVnaW5Nb3VudE93bmVyUgVvd25lchISCgRzbG90'
    'GAMgASgJUgRzbG90EisKEWV4cGVjdGVkX3JldmlzaW9uGAQgASgEUhBleHBlY3RlZFJldmlzaW'
    '9uEhoKCHJldmlzaW9uGAUgASgEUghyZXZpc2lvbhIUCgV0aXRsZRgGIAEoCVIFdGl0bGUSLwoE'
    'cm9vdBgHIAEoCzIbLmFueXR0eS5hcGkudjEuUGx1Z2luVWlOb2RlUgRyb290EjcKB2FjdGlvbn'
    'MYCCADKAsyHS5hbnl0dHkuYXBpLnYxLlBsdWdpblVpQWN0aW9uUgdhY3Rpb25zEhQKBWNsb3Nl'
    'GAkgASgIUgVjbG9zZRIUCgVmb2N1cxgKIAEoCFIFZm9jdXMSJwoPcHJlZmVycmVkX3dpZHRoGA'
    'sgASgNUg5wcmVmZXJyZWRXaWR0aBIbCgltaW5fd2lkdGgYDCABKA1SCG1pbldpZHRo');

@$core.Deprecated('Use pluginAgentReportDescriptor instead')
const PluginAgentReport$json = {
  '1': 'PluginAgentReport',
  '2': [
    {'1': 'agent_id', '3': 1, '4': 1, '5': 9, '10': 'agentId'},
    {'1': 'provider', '3': 2, '4': 1, '5': 9, '10': 'provider'},
    {'1': 'session_id', '3': 3, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'sequence', '3': 4, '4': 1, '5': 4, '10': 'sequence'},
    {'1': 'state', '3': 5, '4': 1, '5': 9, '10': 'state'},
    {'1': 'title', '3': 6, '4': 1, '5': 9, '10': 'title'},
    {'1': 'cwd', '3': 7, '4': 1, '5': 9, '10': 'cwd'},
    {
      '1': 'terminal',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTerminalRef',
      '10': 'terminal'
    },
    {
      '1': 'observed_unix_millis',
      '3': 9,
      '4': 1,
      '5': 3,
      '10': 'observedUnixMillis'
    },
    {'1': 'event', '3': 10, '4': 1, '5': 9, '10': 'event'},
    {'1': 'detail', '3': 11, '4': 1, '5': 9, '10': 'detail'},
    {'1': 'source_epoch', '3': 12, '4': 1, '5': 4, '10': 'sourceEpoch'},
    {'1': 'permission_id', '3': 13, '4': 1, '5': 9, '10': 'permissionId'},
    {'1': 'turn_id', '3': 14, '4': 1, '5': 9, '10': 'turnId'},
    {
      '1': 'pending_permission_ids',
      '3': 15,
      '4': 3,
      '5': 9,
      '10': 'pendingPermissionIds'
    },
    {'1': 'base_state', '3': 16, '4': 1, '5': 9, '10': 'baseState'},
    {'1': 'full_state', '3': 17, '4': 1, '5': 8, '10': 'fullState'},
    {'1': 'stale', '3': 18, '4': 1, '5': 8, '10': 'stale'},
  ],
};

/// Descriptor for `PluginAgentReport`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginAgentReportDescriptor = $convert.base64Decode(
    'ChFQbHVnaW5BZ2VudFJlcG9ydBIZCghhZ2VudF9pZBgBIAEoCVIHYWdlbnRJZBIaCghwcm92aW'
    'RlchgCIAEoCVIIcHJvdmlkZXISHQoKc2Vzc2lvbl9pZBgDIAEoCVIJc2Vzc2lvbklkEhoKCHNl'
    'cXVlbmNlGAQgASgEUghzZXF1ZW5jZRIUCgVzdGF0ZRgFIAEoCVIFc3RhdGUSFAoFdGl0bGUYBi'
    'ABKAlSBXRpdGxlEhAKA2N3ZBgHIAEoCVIDY3dkEjwKCHRlcm1pbmFsGAggASgLMiAuYW55dHR5'
    'LmFwaS52MS5QbHVnaW5UZXJtaW5hbFJlZlIIdGVybWluYWwSMAoUb2JzZXJ2ZWRfdW5peF9taW'
    'xsaXMYCSABKANSEm9ic2VydmVkVW5peE1pbGxpcxIUCgVldmVudBgKIAEoCVIFZXZlbnQSFgoG'
    'ZGV0YWlsGAsgASgJUgZkZXRhaWwSIQoMc291cmNlX2Vwb2NoGAwgASgEUgtzb3VyY2VFcG9jaB'
    'IjCg1wZXJtaXNzaW9uX2lkGA0gASgJUgxwZXJtaXNzaW9uSWQSFwoHdHVybl9pZBgOIAEoCVIG'
    'dHVybklkEjQKFnBlbmRpbmdfcGVybWlzc2lvbl9pZHMYDyADKAlSFHBlbmRpbmdQZXJtaXNzaW'
    '9uSWRzEh0KCmJhc2Vfc3RhdGUYECABKAlSCWJhc2VTdGF0ZRIdCgpmdWxsX3N0YXRlGBEgASgI'
    'UglmdWxsU3RhdGUSFAoFc3RhbGUYEiABKAhSBXN0YWxl');

@$core.Deprecated('Use pluginAgentSnapshotDescriptor instead')
const PluginAgentSnapshot$json = {
  '1': 'PluginAgentSnapshot',
  '2': [
    {
      '1': 'agents',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginAgentReport',
      '10': 'agents'
    },
    {'1': 'revision', '3': 2, '4': 1, '5': 4, '10': 'revision'},
  ],
};

/// Descriptor for `PluginAgentSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginAgentSnapshotDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5BZ2VudFNuYXBzaG90EjgKBmFnZW50cxgBIAMoCzIgLmFueXR0eS5hcGkudjEuUG'
    'x1Z2luQWdlbnRSZXBvcnRSBmFnZW50cxIaCghyZXZpc2lvbhgCIAEoBFIIcmV2aXNpb24=');

@$core.Deprecated('Use pluginStateRequestDescriptor instead')
const PluginStateRequest$json = {
  '1': 'PluginStateRequest',
  '2': [
    {'1': 'source_lease', '3': 1, '4': 1, '5': 12, '10': 'sourceLease'},
    {'1': 'collection', '3': 2, '4': 1, '5': 9, '10': 'collection'},
    {
      '1': 'get',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginStateGet',
      '9': 0,
      '10': 'get'
    },
    {
      '1': 'put',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginStatePut',
      '9': 0,
      '10': 'put'
    },
    {
      '1': 'watch',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginStateWatch',
      '9': 0,
      '10': 'watch'
    },
  ],
  '8': [
    {'1': 'operation'},
  ],
};

/// Descriptor for `PluginStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStateRequestDescriptor = $convert.base64Decode(
    'ChJQbHVnaW5TdGF0ZVJlcXVlc3QSIQoMc291cmNlX2xlYXNlGAEgASgMUgtzb3VyY2VMZWFzZR'
    'IeCgpjb2xsZWN0aW9uGAIgASgJUgpjb2xsZWN0aW9uEjEKA2dldBgKIAEoCzIdLmFueXR0eS5h'
    'cGkudjEuUGx1Z2luU3RhdGVHZXRIAFIDZ2V0EjEKA3B1dBgLIAEoCzIdLmFueXR0eS5hcGkudj'
    'EuUGx1Z2luU3RhdGVQdXRIAFIDcHV0EjcKBXdhdGNoGAwgASgLMh8uYW55dHR5LmFwaS52MS5Q'
    'bHVnaW5TdGF0ZVdhdGNoSABSBXdhdGNoQgsKCW9wZXJhdGlvbg==');

@$core.Deprecated('Use pluginStateGetDescriptor instead')
const PluginStateGet$json = {
  '1': 'PluginStateGet',
};

/// Descriptor for `PluginStateGet`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStateGetDescriptor =
    $convert.base64Decode('Cg5QbHVnaW5TdGF0ZUdldA==');

@$core.Deprecated('Use pluginStatePutDescriptor instead')
const PluginStatePut$json = {
  '1': 'PluginStatePut',
  '2': [
    {
      '1': 'expected_revision',
      '3': 1,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginPayload',
      '10': 'value'
    },
  ],
};

/// Descriptor for `PluginStatePut`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStatePutDescriptor = $convert.base64Decode(
    'Cg5QbHVnaW5TdGF0ZVB1dBIrChFleHBlY3RlZF9yZXZpc2lvbhgBIAEoBFIQZXhwZWN0ZWRSZX'
    'Zpc2lvbhIyCgV2YWx1ZRgCIAEoCzIcLmFueXR0eS5hcGkudjEuUGx1Z2luUGF5bG9hZFIFdmFs'
    'dWU=');

@$core.Deprecated('Use pluginStateWatchDescriptor instead')
const PluginStateWatch$json = {
  '1': 'PluginStateWatch',
  '2': [
    {'1': 'cancel', '3': 1, '4': 1, '5': 8, '10': 'cancel'},
  ],
};

/// Descriptor for `PluginStateWatch`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStateWatchDescriptor = $convert
    .base64Decode('ChBQbHVnaW5TdGF0ZVdhdGNoEhYKBmNhbmNlbBgBIAEoCFIGY2FuY2Vs');

@$core.Deprecated('Use pluginStateSnapshotDescriptor instead')
const PluginStateSnapshot$json = {
  '1': 'PluginStateSnapshot',
  '2': [
    {'1': 'collection', '3': 1, '4': 1, '5': 9, '10': 'collection'},
    {'1': 'revision', '3': 2, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'value',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginPayload',
      '10': 'value'
    },
    {'1': 'boot_epoch', '3': 4, '4': 1, '5': 9, '10': 'bootEpoch'},
  ],
};

/// Descriptor for `PluginStateSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginStateSnapshotDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5TdGF0ZVNuYXBzaG90Eh4KCmNvbGxlY3Rpb24YASABKAlSCmNvbGxlY3Rpb24SGg'
    'oIcmV2aXNpb24YAiABKARSCHJldmlzaW9uEjIKBXZhbHVlGAMgASgLMhwuYW55dHR5LmFwaS52'
    'MS5QbHVnaW5QYXlsb2FkUgV2YWx1ZRIdCgpib290X2Vwb2NoGAQgASgJUglib290RXBvY2g=');

@$core.Deprecated('Use pluginBridgeFrameDescriptor instead')
const PluginBridgeFrame$json = {
  '1': 'PluginBridgeFrame',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'via_endpoint_id', '3': 4, '4': 1, '5': 9, '10': 'viaEndpointId'},
    {
      '1': 'deadline_unix_millis',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'deadlineUnixMillis'
    },
    {
      '1': 'command',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginCommand',
      '9': 0,
      '10': 'command'
    },
    {
      '1': 'result',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginResult',
      '9': 0,
      '10': 'result'
    },
    {
      '1': 'cancel_request_id',
      '3': 6,
      '4': 1,
      '5': 9,
      '9': 0,
      '10': 'cancelRequestId'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `PluginBridgeFrame`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginBridgeFrameDescriptor = $convert.base64Decode(
    'ChFQbHVnaW5CcmlkZ2VGcmFtZRIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSJgoPdm'
    'lhX2VuZHBvaW50X2lkGAQgASgJUg12aWFFbmRwb2ludElkEjAKFGRlYWRsaW5lX3VuaXhfbWls'
    'bGlzGAUgASgDUhJkZWFkbGluZVVuaXhNaWxsaXMSOAoHY29tbWFuZBgCIAEoCzIcLmFueXR0eS'
    '5hcGkudjEuUGx1Z2luQ29tbWFuZEgAUgdjb21tYW5kEjUKBnJlc3VsdBgDIAEoCzIbLmFueXR0'
    'eS5hcGkudjEuUGx1Z2luUmVzdWx0SABSBnJlc3VsdBIsChFjYW5jZWxfcmVxdWVzdF9pZBgGIA'
    'EoCUgAUg9jYW5jZWxSZXF1ZXN0SWRCCQoHcGF5bG9hZA==');

@$core.Deprecated('Use pluginUiInitDescriptor instead')
const PluginUiInit$json = {
  '1': 'PluginUiInit',
  '2': [
    {
      '1': 'owners',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginMountOwner',
      '10': 'owners'
    },
    {
      '1': 'host',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginAddress',
      '10': 'host'
    },
    {
      '1': 'context',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTargetContext',
      '10': 'context'
    },
    {
      '1': 'mount_revisions',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiInit.MountRevisionsEntry',
      '10': 'mountRevisions'
    },
    {
      '1': 'reconcile_request_id',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'reconcileRequestId'
    },
  ],
  '3': [PluginUiInit_MountRevisionsEntry$json],
};

@$core.Deprecated('Use pluginUiInitDescriptor instead')
const PluginUiInit_MountRevisionsEntry$json = {
  '1': 'MountRevisionsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 4, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `PluginUiInit`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiInitDescriptor = $convert.base64Decode(
    'CgxQbHVnaW5VaUluaXQSNwoGb3duZXJzGAEgAygLMh8uYW55dHR5LmFwaS52MS5QbHVnaW5Nb3'
    'VudE93bmVyUgZvd25lcnMSMAoEaG9zdBgCIAEoCzIcLmFueXR0eS5hcGkudjEuUGx1Z2luQWRk'
    'cmVzc1IEaG9zdBI8Cgdjb250ZXh0GAMgASgLMiIuYW55dHR5LmFwaS52MS5QbHVnaW5UYXJnZX'
    'RDb250ZXh0Ugdjb250ZXh0ElgKD21vdW50X3JldmlzaW9ucxgEIAMoCzIvLmFueXR0eS5hcGku'
    'djEuUGx1Z2luVWlJbml0Lk1vdW50UmV2aXNpb25zRW50cnlSDm1vdW50UmV2aXNpb25zEjAKFH'
    'JlY29uY2lsZV9yZXF1ZXN0X2lkGAUgASgJUhJyZWNvbmNpbGVSZXF1ZXN0SWQaQQoTTW91bnRS'
    'ZXZpc2lvbnNFbnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoBFIFdmFsdWU6Aj'
    'gB');

@$core.Deprecated('Use pluginUiSnapshotQueryDescriptor instead')
const PluginUiSnapshotQuery$json = {
  '1': 'PluginUiSnapshotQuery',
};

/// Descriptor for `PluginUiSnapshotQuery`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiSnapshotQueryDescriptor =
    $convert.base64Decode('ChVQbHVnaW5VaVNuYXBzaG90UXVlcnk=');

@$core.Deprecated('Use pluginUiQueryDescriptor instead')
const PluginUiQuery$json = {
  '1': 'PluginUiQuery',
  '2': [
    {
      '1': 'snapshot',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginUiSnapshotQuery',
      '9': 0,
      '10': 'snapshot'
    },
  ],
  '8': [
    {'1': 'query'},
  ],
};

/// Descriptor for `PluginUiQuery`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiQueryDescriptor = $convert.base64Decode(
    'Cg1QbHVnaW5VaVF1ZXJ5EkIKCHNuYXBzaG90GAEgASgLMiQuYW55dHR5LmFwaS52MS5QbHVnaW'
    '5VaVNuYXBzaG90UXVlcnlIAFIIc25hcHNob3RCBwoFcXVlcnk=');

@$core.Deprecated('Use pluginPanelSnapshotDescriptor instead')
const PluginPanelSnapshot$json = {
  '1': 'PluginPanelSnapshot',
  '2': [
    {
      '1': 'owner',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginMountOwner',
      '10': 'owner'
    },
    {
      '1': 'terminal',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTerminalRef',
      '10': 'terminal'
    },
    {'1': 'view_id', '3': 3, '4': 1, '5': 9, '10': 'viewId'},
    {'1': 'focused', '3': 4, '4': 1, '5': 8, '10': 'focused'},
  ],
};

/// Descriptor for `PluginPanelSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginPanelSnapshotDescriptor = $convert.base64Decode(
    'ChNQbHVnaW5QYW5lbFNuYXBzaG90EjUKBW93bmVyGAEgASgLMh8uYW55dHR5LmFwaS52MS5QbH'
    'VnaW5Nb3VudE93bmVyUgVvd25lchI8Cgh0ZXJtaW5hbBgCIAEoCzIgLmFueXR0eS5hcGkudjEu'
    'UGx1Z2luVGVybWluYWxSZWZSCHRlcm1pbmFsEhcKB3ZpZXdfaWQYAyABKAlSBnZpZXdJZBIYCg'
    'dmb2N1c2VkGAQgASgIUgdmb2N1c2Vk');

@$core.Deprecated('Use pluginUiSnapshotDescriptor instead')
const PluginUiSnapshot$json = {
  '1': 'PluginUiSnapshot',
  '2': [
    {'1': 'tui_instance_id', '3': 1, '4': 1, '5': 9, '10': 'tuiInstanceId'},
    {
      '1': 'owners',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginMountOwner',
      '10': 'owners'
    },
    {
      '1': 'panels',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.PluginPanelSnapshot',
      '10': 'panels'
    },
    {
      '1': 'active_context',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PluginTargetContext',
      '10': 'activeContext'
    },
    {'1': 'revision', '3': 5, '4': 1, '5': 4, '10': 'revision'},
  ],
};

/// Descriptor for `PluginUiSnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pluginUiSnapshotDescriptor = $convert.base64Decode(
    'ChBQbHVnaW5VaVNuYXBzaG90EiYKD3R1aV9pbnN0YW5jZV9pZBgBIAEoCVINdHVpSW5zdGFuY2'
    'VJZBI3CgZvd25lcnMYAiADKAsyHy5hbnl0dHkuYXBpLnYxLlBsdWdpbk1vdW50T3duZXJSBm93'
    'bmVycxI6CgZwYW5lbHMYAyADKAsyIi5hbnl0dHkuYXBpLnYxLlBsdWdpblBhbmVsU25hcHNob3'
    'RSBnBhbmVscxJJCg5hY3RpdmVfY29udGV4dBgEIAEoCzIiLmFueXR0eS5hcGkudjEuUGx1Z2lu'
    'VGFyZ2V0Q29udGV4dFINYWN0aXZlQ29udGV4dBIaCghyZXZpc2lvbhgFIAEoBFIIcmV2aXNpb2'
    '4=');
