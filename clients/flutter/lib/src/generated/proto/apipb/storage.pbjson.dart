// This is a generated file - do not edit.
//
// Generated from apipb/storage.proto.

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

@$core.Deprecated('Use storageScopeDescriptor instead')
const StorageScope$json = {
  '1': 'StorageScope',
  '2': [
    {'1': 'STORAGE_SCOPE_UNSPECIFIED', '2': 0},
    {'1': 'STORAGE_SCOPE_PUBLIC', '2': 1},
    {'1': 'STORAGE_SCOPE_PRIVATE', '2': 2},
  ],
};

/// Descriptor for `StorageScope`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List storageScopeDescriptor = $convert.base64Decode(
    'CgxTdG9yYWdlU2NvcGUSHQoZU1RPUkFHRV9TQ09QRV9VTlNQRUNJRklFRBAAEhgKFFNUT1JBR0'
    'VfU0NPUEVfUFVCTElDEAESGQoVU1RPUkFHRV9TQ09QRV9QUklWQVRFEAI=');

@$core.Deprecated('Use storageKeyDescriptor instead')
const StorageKey$json = {
  '1': 'StorageKey',
  '2': [
    {'1': 'app_id', '3': 1, '4': 1, '5': 9, '10': 'appId'},
    {
      '1': 'scope',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.StorageScope',
      '10': 'scope'
    },
    {'1': 'owner_id', '3': 3, '4': 1, '5': 9, '10': 'ownerId'},
    {'1': 'key', '3': 4, '4': 1, '5': 9, '10': 'key'},
  ],
};

/// Descriptor for `StorageKey`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageKeyDescriptor = $convert.base64Decode(
    'CgpTdG9yYWdlS2V5EhUKBmFwcF9pZBgBIAEoCVIFYXBwSWQSMQoFc2NvcGUYAiABKA4yGy5hbn'
    'l0dHkuYXBpLnYxLlN0b3JhZ2VTY29wZVIFc2NvcGUSGQoIb3duZXJfaWQYAyABKAlSB293bmVy'
    'SWQSEAoDa2V5GAQgASgJUgNrZXk=');

@$core.Deprecated('Use storageEntryDescriptor instead')
const StorageEntry$json = {
  '1': 'StorageEntry',
  '2': [
    {
      '1': 'key',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageKey',
      '10': 'key'
    },
    {'1': 'value', '3': 2, '4': 1, '5': 12, '10': 'value'},
    {'1': 'version', '3': 3, '4': 1, '5': 4, '10': 'version'},
    {
      '1': 'updated_at_unix_nano',
      '3': 4,
      '4': 1,
      '5': 3,
      '10': 'updatedAtUnixNano'
    },
  ],
};

/// Descriptor for `StorageEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageEntryDescriptor = $convert.base64Decode(
    'CgxTdG9yYWdlRW50cnkSKwoDa2V5GAEgASgLMhkuYW55dHR5LmFwaS52MS5TdG9yYWdlS2V5Ug'
    'NrZXkSFAoFdmFsdWUYAiABKAxSBXZhbHVlEhgKB3ZlcnNpb24YAyABKARSB3ZlcnNpb24SLwoU'
    'dXBkYXRlZF9hdF91bml4X25hbm8YBCABKANSEXVwZGF0ZWRBdFVuaXhOYW5v');

@$core.Deprecated('Use storageVersionFenceDescriptor instead')
const StorageVersionFence$json = {
  '1': 'StorageVersionFence',
  '2': [
    {'1': 'check_version', '3': 1, '4': 1, '5': 8, '10': 'checkVersion'},
    {'1': 'expected_version', '3': 2, '4': 1, '5': 4, '10': 'expectedVersion'},
  ],
};

/// Descriptor for `StorageVersionFence`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageVersionFenceDescriptor = $convert.base64Decode(
    'ChNTdG9yYWdlVmVyc2lvbkZlbmNlEiMKDWNoZWNrX3ZlcnNpb24YASABKAhSDGNoZWNrVmVyc2'
    'lvbhIpChBleHBlY3RlZF92ZXJzaW9uGAIgASgEUg9leHBlY3RlZFZlcnNpb24=');

@$core.Deprecated('Use storageGetCommandDescriptor instead')
const StorageGetCommand$json = {
  '1': 'StorageGetCommand',
  '2': [
    {
      '1': 'key',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageKey',
      '10': 'key'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `StorageGetCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageGetCommandDescriptor = $convert.base64Decode(
    'ChFTdG9yYWdlR2V0Q29tbWFuZBIrCgNrZXkYAiABKAsyGS5hbnl0dHkuYXBpLnYxLlN0b3JhZ2'
    'VLZXlSA2tleUoECAEQAg==');

@$core.Deprecated('Use storagePutCommandDescriptor instead')
const StoragePutCommand$json = {
  '1': 'StoragePutCommand',
  '2': [
    {
      '1': 'key',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageKey',
      '10': 'key'
    },
    {'1': 'value', '3': 3, '4': 1, '5': 12, '10': 'value'},
    {
      '1': 'version',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageVersionFence',
      '10': 'version'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `StoragePutCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storagePutCommandDescriptor = $convert.base64Decode(
    'ChFTdG9yYWdlUHV0Q29tbWFuZBIrCgNrZXkYAiABKAsyGS5hbnl0dHkuYXBpLnYxLlN0b3JhZ2'
    'VLZXlSA2tleRIUCgV2YWx1ZRgDIAEoDFIFdmFsdWUSPAoHdmVyc2lvbhgEIAEoCzIiLmFueXR0'
    'eS5hcGkudjEuU3RvcmFnZVZlcnNpb25GZW5jZVIHdmVyc2lvbkoECAEQAg==');

@$core.Deprecated('Use storageDeleteCommandDescriptor instead')
const StorageDeleteCommand$json = {
  '1': 'StorageDeleteCommand',
  '2': [
    {
      '1': 'key',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageKey',
      '10': 'key'
    },
    {
      '1': 'version',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageVersionFence',
      '10': 'version'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `StorageDeleteCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageDeleteCommandDescriptor = $convert.base64Decode(
    'ChRTdG9yYWdlRGVsZXRlQ29tbWFuZBIrCgNrZXkYAiABKAsyGS5hbnl0dHkuYXBpLnYxLlN0b3'
    'JhZ2VLZXlSA2tleRI8Cgd2ZXJzaW9uGAMgASgLMiIuYW55dHR5LmFwaS52MS5TdG9yYWdlVmVy'
    'c2lvbkZlbmNlUgd2ZXJzaW9uSgQIARAC');

@$core.Deprecated('Use storageListCommandDescriptor instead')
const StorageListCommand$json = {
  '1': 'StorageListCommand',
  '2': [
    {'1': 'app_id', '3': 2, '4': 1, '5': 9, '10': 'appId'},
    {
      '1': 'scope',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.StorageScope',
      '10': 'scope'
    },
    {'1': 'owner_id', '3': 4, '4': 1, '5': 9, '10': 'ownerId'},
    {'1': 'prefix', '3': 5, '4': 1, '5': 9, '10': 'prefix'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `StorageListCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageListCommandDescriptor = $convert.base64Decode(
    'ChJTdG9yYWdlTGlzdENvbW1hbmQSFQoGYXBwX2lkGAIgASgJUgVhcHBJZBIxCgVzY29wZRgDIA'
    'EoDjIbLmFueXR0eS5hcGkudjEuU3RvcmFnZVNjb3BlUgVzY29wZRIZCghvd25lcl9pZBgEIAEo'
    'CVIHb3duZXJJZBIWCgZwcmVmaXgYBSABKAlSBnByZWZpeEoECAEQAg==');

@$core.Deprecated('Use storageGetResultDescriptor instead')
const StorageGetResult$json = {
  '1': 'StorageGetResult',
  '2': [
    {
      '1': 'entry',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageEntry',
      '10': 'entry'
    },
  ],
};

/// Descriptor for `StorageGetResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageGetResultDescriptor = $convert.base64Decode(
    'ChBTdG9yYWdlR2V0UmVzdWx0EjEKBWVudHJ5GAEgASgLMhsuYW55dHR5LmFwaS52MS5TdG9yYW'
    'dlRW50cnlSBWVudHJ5');

@$core.Deprecated('Use storagePutResultDescriptor instead')
const StoragePutResult$json = {
  '1': 'StoragePutResult',
  '2': [
    {
      '1': 'entry',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageEntry',
      '10': 'entry'
    },
  ],
};

/// Descriptor for `StoragePutResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storagePutResultDescriptor = $convert.base64Decode(
    'ChBTdG9yYWdlUHV0UmVzdWx0EjEKBWVudHJ5GAEgASgLMhsuYW55dHR5LmFwaS52MS5TdG9yYW'
    'dlRW50cnlSBWVudHJ5');

@$core.Deprecated('Use storageDeleteResultDescriptor instead')
const StorageDeleteResult$json = {
  '1': 'StorageDeleteResult',
  '2': [
    {
      '1': 'key',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageKey',
      '10': 'key'
    },
    {'1': 'deleted', '3': 2, '4': 1, '5': 8, '10': 'deleted'},
    {'1': 'version', '3': 3, '4': 1, '5': 4, '10': 'version'},
  ],
};

/// Descriptor for `StorageDeleteResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageDeleteResultDescriptor = $convert.base64Decode(
    'ChNTdG9yYWdlRGVsZXRlUmVzdWx0EisKA2tleRgBIAEoCzIZLmFueXR0eS5hcGkudjEuU3Rvcm'
    'FnZUtleVIDa2V5EhgKB2RlbGV0ZWQYAiABKAhSB2RlbGV0ZWQSGAoHdmVyc2lvbhgDIAEoBFIH'
    'dmVyc2lvbg==');

@$core.Deprecated('Use storageListResultDescriptor instead')
const StorageListResult$json = {
  '1': 'StorageListResult',
  '2': [
    {
      '1': 'entries',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.StorageEntry',
      '10': 'entries'
    },
  ],
};

/// Descriptor for `StorageListResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageListResultDescriptor = $convert.base64Decode(
    'ChFTdG9yYWdlTGlzdFJlc3VsdBI1CgdlbnRyaWVzGAEgAygLMhsuYW55dHR5LmFwaS52MS5TdG'
    '9yYWdlRW50cnlSB2VudHJpZXM=');

@$core.Deprecated('Use storageChangedEventDescriptor instead')
const StorageChangedEvent$json = {
  '1': 'StorageChangedEvent',
  '2': [
    {
      '1': 'key',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageKey',
      '10': 'key'
    },
    {'1': 'version', '3': 2, '4': 1, '5': 4, '10': 'version'},
    {'1': 'operation', '3': 3, '4': 1, '5': 9, '10': 'operation'},
  ],
};

/// Descriptor for `StorageChangedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageChangedEventDescriptor = $convert.base64Decode(
    'ChNTdG9yYWdlQ2hhbmdlZEV2ZW50EisKA2tleRgBIAEoCzIZLmFueXR0eS5hcGkudjEuU3Rvcm'
    'FnZUtleVIDa2V5EhgKB3ZlcnNpb24YAiABKARSB3ZlcnNpb24SHAoJb3BlcmF0aW9uGAMgASgJ'
    'UglvcGVyYXRpb24=');
