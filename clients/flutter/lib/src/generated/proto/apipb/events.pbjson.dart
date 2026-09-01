// This is a generated file - do not edit.
//
// Generated from apipb/events.proto.

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

@$core.Deprecated('Use applicationEventTypeDescriptor instead')
const ApplicationEventType$json = {
  '1': 'ApplicationEventType',
  '2': [
    {'1': 'APPLICATION_EVENT_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE', '2': 1},
    {'1': 'APPLICATION_EVENT_TYPE_STORAGE_CHANGED', '2': 5},
  ],
  '4': [
    {'1': 2, '2': 2},
    {'1': 3, '2': 3},
    {'1': 4, '2': 4},
  ],
};

/// Descriptor for `ApplicationEventType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List applicationEventTypeDescriptor = $convert.base64Decode(
    'ChRBcHBsaWNhdGlvbkV2ZW50VHlwZRImCiJBUFBMSUNBVElPTl9FVkVOVF9UWVBFX1VOU1BFQ0'
    'lGSUVEEAASLQopQVBQTElDQVRJT05fRVZFTlRfVFlQRV9URVJNSU5BTF9MSUZFQ1lDTEUQARIq'
    'CiZBUFBMSUNBVElPTl9FVkVOVF9UWVBFX1NUT1JBR0VfQ0hBTkdFRBAFIgQIAhACIgQIAxADIg'
    'QIBBAE');

@$core.Deprecated('Use eventSubscribeCommandDescriptor instead')
const EventSubscribeCommand$json = {
  '1': 'EventSubscribeCommand',
  '2': [
    {
      '1': 'types',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.anytty.api.v1.ApplicationEventType',
      '10': 'types'
    },
    {
      '1': 'terminal',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRef',
      '10': 'terminal'
    },
    {'1': 'storage_app_id', '3': 4, '4': 1, '5': 9, '10': 'storageAppId'},
    {
      '1': 'storage_scope',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.StorageScope',
      '10': 'storageScope'
    },
    {'1': 'storage_owner_id', '3': 6, '4': 1, '5': 9, '10': 'storageOwnerId'},
    {
      '1': 'storage_key_prefix',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'storageKeyPrefix'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `EventSubscribeCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eventSubscribeCommandDescriptor = $convert.base64Decode(
    'ChVFdmVudFN1YnNjcmliZUNvbW1hbmQSOQoFdHlwZXMYAiADKA4yIy5hbnl0dHkuYXBpLnYxLk'
    'FwcGxpY2F0aW9uRXZlbnRUeXBlUgV0eXBlcxI2Cgh0ZXJtaW5hbBgDIAEoCzIaLmFueXR0eS5h'
    'cGkudjEuVGVybWluYWxSZWZSCHRlcm1pbmFsEiQKDnN0b3JhZ2VfYXBwX2lkGAQgASgJUgxzdG'
    '9yYWdlQXBwSWQSQAoNc3RvcmFnZV9zY29wZRgFIAEoDjIbLmFueXR0eS5hcGkudjEuU3RvcmFn'
    'ZVNjb3BlUgxzdG9yYWdlU2NvcGUSKAoQc3RvcmFnZV9vd25lcl9pZBgGIAEoCVIOc3RvcmFnZU'
    '93bmVySWQSLAoSc3RvcmFnZV9rZXlfcHJlZml4GAcgASgJUhBzdG9yYWdlS2V5UHJlZml4SgQI'
    'ARAC');

@$core.Deprecated('Use eventSubscriptionResultDescriptor instead')
const EventSubscriptionResult$json = {
  '1': 'EventSubscriptionResult',
  '2': [
    {
      '1': 'subscription',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'subscription'
    },
  ],
};

/// Descriptor for `EventSubscriptionResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eventSubscriptionResultDescriptor =
    $convert.base64Decode(
        'ChdFdmVudFN1YnNjcmlwdGlvblJlc3VsdBJBCgxzdWJzY3JpcHRpb24YASABKAsyHS5hbnl0dH'
        'kuYXBpLnYxLlJlc291cmNlSGFuZGxlUgxzdWJzY3JpcHRpb24=');
