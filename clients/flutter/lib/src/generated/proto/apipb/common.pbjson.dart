// This is a generated file - do not edit.
//
// Generated from apipb/common.proto.

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

@$core.Deprecated('Use apiCapabilityDescriptor instead')
const ApiCapability$json = {
  '1': 'ApiCapability',
  '2': [
    {'1': 'API_CAPABILITY_UNSPECIFIED', '2': 0},
    {'1': 'API_CAPABILITY_TYPED_ERRORS', '2': 1},
    {'1': 'API_CAPABILITY_ENDPOINT_SESSION_FENCE', '2': 2},
    {'1': 'API_CAPABILITY_OPERATION_CANCELLATION', '2': 3},
    {'1': 'API_CAPABILITY_RESOURCE_LIFECYCLE', '2': 4},
    {'1': 'API_CAPABILITY_TERMINAL_LIFECYCLE', '2': 5},
    {'1': 'API_CAPABILITY_TERMINAL_ATTACHMENT', '2': 6},
    {'1': 'API_CAPABILITY_PATH_QUERY', '2': 7},
    {'1': 'API_CAPABILITY_HISTORY', '2': 8},
    {'1': 'API_CAPABILITY_LIVE_SCREEN', '2': 9},
    {'1': 'API_CAPABILITY_FILE', '2': 10},
    {'1': 'API_CAPABILITY_STORAGE', '2': 11},
    {'1': 'API_CAPABILITY_EVENT_SUBSCRIPTION', '2': 12},
    {'1': 'API_CAPABILITY_CLIENT_ACCESS', '2': 13},
    {'1': 'API_CAPABILITY_REMOTE_CONTROL', '2': 14},
    {'1': 'API_CAPABILITY_BROWSER_PROXY', '2': 15},
  ],
};

/// Descriptor for `ApiCapability`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List apiCapabilityDescriptor = $convert.base64Decode(
    'Cg1BcGlDYXBhYmlsaXR5Eh4KGkFQSV9DQVBBQklMSVRZX1VOU1BFQ0lGSUVEEAASHwobQVBJX0'
    'NBUEFCSUxJVFlfVFlQRURfRVJST1JTEAESKQolQVBJX0NBUEFCSUxJVFlfRU5EUE9JTlRfU0VT'
    'U0lPTl9GRU5DRRACEikKJUFQSV9DQVBBQklMSVRZX09QRVJBVElPTl9DQU5DRUxMQVRJT04QAx'
    'IlCiFBUElfQ0FQQUJJTElUWV9SRVNPVVJDRV9MSUZFQ1lDTEUQBBIlCiFBUElfQ0FQQUJJTElU'
    'WV9URVJNSU5BTF9MSUZFQ1lDTEUQBRImCiJBUElfQ0FQQUJJTElUWV9URVJNSU5BTF9BVFRBQ0'
    'hNRU5UEAYSHQoZQVBJX0NBUEFCSUxJVFlfUEFUSF9RVUVSWRAHEhoKFkFQSV9DQVBBQklMSVRZ'
    'X0hJU1RPUlkQCBIeChpBUElfQ0FQQUJJTElUWV9MSVZFX1NDUkVFThAJEhcKE0FQSV9DQVBBQk'
    'lMSVRZX0ZJTEUQChIaChZBUElfQ0FQQUJJTElUWV9TVE9SQUdFEAsSJQohQVBJX0NBUEFCSUxJ'
    'VFlfRVZFTlRfU1VCU0NSSVBUSU9OEAwSIAocQVBJX0NBUEFCSUxJVFlfQ0xJRU5UX0FDQ0VTUx'
    'ANEiEKHUFQSV9DQVBBQklMSVRZX1JFTU9URV9DT05UUk9MEA4SIAocQVBJX0NBUEFCSUxJVFlf'
    'QlJPV1NFUl9QUk9YWRAP');

@$core.Deprecated('Use apiErrorCodeDescriptor instead')
const ApiErrorCode$json = {
  '1': 'ApiErrorCode',
  '2': [
    {'1': 'API_ERROR_CODE_UNSPECIFIED', '2': 0},
    {'1': 'API_ERROR_CODE_INVALID_REQUEST', '2': 1},
    {'1': 'API_ERROR_CODE_UNSUPPORTED_VERSION', '2': 2},
    {'1': 'API_ERROR_CODE_UNSUPPORTED_CAPABILITY', '2': 3},
    {'1': 'API_ERROR_CODE_UNAUTHORIZED', '2': 4},
    {'1': 'API_ERROR_CODE_FORBIDDEN', '2': 5},
    {'1': 'API_ERROR_CODE_NOT_FOUND', '2': 6},
    {'1': 'API_ERROR_CODE_CONFLICT', '2': 7},
    {'1': 'API_ERROR_CODE_STALE_SESSION', '2': 8},
    {'1': 'API_ERROR_CODE_CANCELLED', '2': 9},
    {'1': 'API_ERROR_CODE_UNAVAILABLE', '2': 10},
    {'1': 'API_ERROR_CODE_INTERNAL', '2': 11},
    {'1': 'API_ERROR_CODE_ENTITLEMENT_DENIED', '2': 12},
    {'1': 'API_ERROR_CODE_RESOURCE_EXHAUSTED', '2': 13},
    {'1': 'API_ERROR_CODE_STALE_RESOURCE', '2': 14},
    {'1': 'API_ERROR_CODE_DAEMON_BLOCKED', '2': 15},
    {'1': 'API_ERROR_CODE_DAEMON_DELETED', '2': 16},
    {'1': 'API_ERROR_CODE_RELAY_NOT_IN_PLAN', '2': 17},
    {'1': 'API_ERROR_CODE_RELAY_QUOTA_EXHAUSTED', '2': 18},
    {'1': 'API_ERROR_CODE_RELAY_CONCURRENCY_EXHAUSTED', '2': 19},
    {'1': 'API_ERROR_CODE_SUBSCRIPTION_INACTIVE', '2': 20},
    {'1': 'API_ERROR_CODE_RELAY_REGION_UNAVAILABLE', '2': 21},
  ],
};

/// Descriptor for `ApiErrorCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List apiErrorCodeDescriptor = $convert.base64Decode(
    'CgxBcGlFcnJvckNvZGUSHgoaQVBJX0VSUk9SX0NPREVfVU5TUEVDSUZJRUQQABIiCh5BUElfRV'
    'JST1JfQ09ERV9JTlZBTElEX1JFUVVFU1QQARImCiJBUElfRVJST1JfQ09ERV9VTlNVUFBPUlRF'
    'RF9WRVJTSU9OEAISKQolQVBJX0VSUk9SX0NPREVfVU5TVVBQT1JURURfQ0FQQUJJTElUWRADEh'
    '8KG0FQSV9FUlJPUl9DT0RFX1VOQVVUSE9SSVpFRBAEEhwKGEFQSV9FUlJPUl9DT0RFX0ZPUkJJ'
    'RERFThAFEhwKGEFQSV9FUlJPUl9DT0RFX05PVF9GT1VORBAGEhsKF0FQSV9FUlJPUl9DT0RFX0'
    'NPTkZMSUNUEAcSIAocQVBJX0VSUk9SX0NPREVfU1RBTEVfU0VTU0lPThAIEhwKGEFQSV9FUlJP'
    'Ul9DT0RFX0NBTkNFTExFRBAJEh4KGkFQSV9FUlJPUl9DT0RFX1VOQVZBSUxBQkxFEAoSGwoXQV'
    'BJX0VSUk9SX0NPREVfSU5URVJOQUwQCxIlCiFBUElfRVJST1JfQ09ERV9FTlRJVExFTUVOVF9E'
    'RU5JRUQQDBIlCiFBUElfRVJST1JfQ09ERV9SRVNPVVJDRV9FWEhBVVNURUQQDRIhCh1BUElfRV'
    'JST1JfQ09ERV9TVEFMRV9SRVNPVVJDRRAOEiEKHUFQSV9FUlJPUl9DT0RFX0RBRU1PTl9CTE9D'
    'S0VEEA8SIQodQVBJX0VSUk9SX0NPREVfREFFTU9OX0RFTEVURUQQEBIkCiBBUElfRVJST1JfQ0'
    '9ERV9SRUxBWV9OT1RfSU5fUExBThAREigKJEFQSV9FUlJPUl9DT0RFX1JFTEFZX1FVT1RBX0VY'
    'SEFVU1RFRBASEi4KKkFQSV9FUlJPUl9DT0RFX1JFTEFZX0NPTkNVUlJFTkNZX0VYSEFVU1RFRB'
    'ATEigKJEFQSV9FUlJPUl9DT0RFX1NVQlNDUklQVElPTl9JTkFDVElWRRAUEisKJ0FQSV9FUlJP'
    'Ul9DT0RFX1JFTEFZX1JFR0lPTl9VTkFWQUlMQUJMRRAV');

@$core.Deprecated('Use resourceKindDescriptor instead')
const ResourceKind$json = {
  '1': 'ResourceKind',
  '2': [
    {'1': 'RESOURCE_KIND_UNSPECIFIED', '2': 0},
    {'1': 'RESOURCE_KIND_OPERATION', '2': 1},
    {'1': 'RESOURCE_KIND_SUBSCRIPTION', '2': 2},
    {'1': 'RESOURCE_KIND_TERMINAL_ATTACHMENT', '2': 3},
    {'1': 'RESOURCE_KIND_HISTORY_WINDOW', '2': 4},
    {'1': 'RESOURCE_KIND_FILE_TRANSFER', '2': 5},
    {'1': 'RESOURCE_KIND_BROWSER_PROXY', '2': 6},
  ],
};

/// Descriptor for `ResourceKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List resourceKindDescriptor = $convert.base64Decode(
    'CgxSZXNvdXJjZUtpbmQSHQoZUkVTT1VSQ0VfS0lORF9VTlNQRUNJRklFRBAAEhsKF1JFU09VUk'
    'NFX0tJTkRfT1BFUkFUSU9OEAESHgoaUkVTT1VSQ0VfS0lORF9TVUJTQ1JJUFRJT04QAhIlCiFS'
    'RVNPVVJDRV9LSU5EX1RFUk1JTkFMX0FUVEFDSE1FTlQQAxIgChxSRVNPVVJDRV9LSU5EX0hJU1'
    'RPUllfV0lORE9XEAQSHwobUkVTT1VSQ0VfS0lORF9GSUxFX1RSQU5TRkVSEAUSHwobUkVTT1VS'
    'Q0VfS0lORF9CUk9XU0VSX1BST1hZEAY=');

@$core.Deprecated('Use apiVersionDescriptor instead')
const ApiVersion$json = {
  '1': 'ApiVersion',
  '2': [
    {'1': 'major', '3': 1, '4': 1, '5': 13, '10': 'major'},
    {'1': 'minor', '3': 2, '4': 1, '5': 13, '10': 'minor'},
  ],
};

/// Descriptor for `ApiVersion`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List apiVersionDescriptor = $convert.base64Decode(
    'CgpBcGlWZXJzaW9uEhQKBW1ham9yGAEgASgNUgVtYWpvchIUCgVtaW5vchgCIAEoDVIFbWlub3'
    'I=');

@$core.Deprecated('Use endpointSessionStampDescriptor instead')
const EndpointSessionStamp$json = {
  '1': 'EndpointSessionStamp',
  '2': [
    {'1': 'endpoint_id', '3': 1, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'route_id', '3': 2, '4': 1, '5': 9, '10': 'routeId'},
    {'1': 'generation', '3': 3, '4': 1, '5': 4, '10': 'generation'},
  ],
};

/// Descriptor for `EndpointSessionStamp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List endpointSessionStampDescriptor = $convert.base64Decode(
    'ChRFbmRwb2ludFNlc3Npb25TdGFtcBIfCgtlbmRwb2ludF9pZBgBIAEoCVIKZW5kcG9pbnRJZB'
    'IZCghyb3V0ZV9pZBgCIAEoCVIHcm91dGVJZBIeCgpnZW5lcmF0aW9uGAMgASgEUgpnZW5lcmF0'
    'aW9u');

@$core.Deprecated('Use operationStampDescriptor instead')
const OperationStamp$json = {
  '1': 'OperationStamp',
  '2': [
    {
      '1': 'session',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {'1': 'operation_id', '3': 2, '4': 1, '5': 9, '10': 'operationId'},
  ],
};

/// Descriptor for `OperationStamp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operationStampDescriptor = $convert.base64Decode(
    'Cg5PcGVyYXRpb25TdGFtcBI9CgdzZXNzaW9uGAEgASgLMiMuYW55dHR5LmFwaS52MS5FbmRwb2'
    'ludFNlc3Npb25TdGFtcFIHc2Vzc2lvbhIhCgxvcGVyYXRpb25faWQYAiABKAlSC29wZXJhdGlv'
    'bklk');

@$core.Deprecated('Use resourceHandleDescriptor instead')
const ResourceHandle$json = {
  '1': 'ResourceHandle',
  '2': [
    {'1': 'opaque_token', '3': 1, '4': 1, '5': 12, '10': 'opaqueToken'},
    {
      '1': 'kind',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.ResourceKind',
      '10': 'kind'
    },
    {
      '1': 'session',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
    {'1': 'generation', '3': 4, '4': 1, '5': 4, '10': 'generation'},
  ],
};

/// Descriptor for `ResourceHandle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resourceHandleDescriptor = $convert.base64Decode(
    'Cg5SZXNvdXJjZUhhbmRsZRIhCgxvcGFxdWVfdG9rZW4YASABKAxSC29wYXF1ZVRva2VuEi8KBG'
    'tpbmQYAiABKA4yGy5hbnl0dHkuYXBpLnYxLlJlc291cmNlS2luZFIEa2luZBI9CgdzZXNzaW9u'
    'GAMgASgLMiMuYW55dHR5LmFwaS52MS5FbmRwb2ludFNlc3Npb25TdGFtcFIHc2Vzc2lvbhIeCg'
    'pnZW5lcmF0aW9uGAQgASgEUgpnZW5lcmF0aW9u');

@$core.Deprecated('Use requestContextDescriptor instead')
const RequestContext$json = {
  '1': 'RequestContext',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'api_version',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiVersion',
      '10': 'apiVersion'
    },
    {
      '1': 'session',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'session'
    },
  ],
  '9': [
    {'1': 3, '2': 4},
  ],
};

/// Descriptor for `RequestContext`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestContextDescriptor = $convert.base64Decode(
    'Cg5SZXF1ZXN0Q29udGV4dBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSOgoLYXBpX3'
    'ZlcnNpb24YAiABKAsyGS5hbnl0dHkuYXBpLnYxLkFwaVZlcnNpb25SCmFwaVZlcnNpb24SPQoH'
    'c2Vzc2lvbhgEIAEoCzIjLmFueXR0eS5hcGkudjEuRW5kcG9pbnRTZXNzaW9uU3RhbXBSB3Nlc3'
    'Npb25KBAgDEAQ=');

@$core.Deprecated('Use validationErrorDetailDescriptor instead')
const ValidationErrorDetail$json = {
  '1': 'ValidationErrorDetail',
  '2': [
    {'1': 'field', '3': 1, '4': 1, '5': 9, '10': 'field'},
    {'1': 'reason', '3': 2, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `ValidationErrorDetail`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List validationErrorDetailDescriptor = $convert.base64Decode(
    'ChVWYWxpZGF0aW9uRXJyb3JEZXRhaWwSFAoFZmllbGQYASABKAlSBWZpZWxkEhYKBnJlYXNvbh'
    'gCIAEoCVIGcmVhc29u');

@$core.Deprecated('Use staleSessionErrorDetailDescriptor instead')
const StaleSessionErrorDetail$json = {
  '1': 'StaleSessionErrorDetail',
  '2': [
    {
      '1': 'requested',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'requested'
    },
    {
      '1': 'current_generation',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'currentGeneration'
    },
  ],
};

/// Descriptor for `StaleSessionErrorDetail`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List staleSessionErrorDetailDescriptor = $convert.base64Decode(
    'ChdTdGFsZVNlc3Npb25FcnJvckRldGFpbBJBCglyZXF1ZXN0ZWQYASABKAsyIy5hbnl0dHkuYX'
    'BpLnYxLkVuZHBvaW50U2Vzc2lvblN0YW1wUglyZXF1ZXN0ZWQSLQoSY3VycmVudF9nZW5lcmF0'
    'aW9uGAIgASgEUhFjdXJyZW50R2VuZXJhdGlvbg==');

@$core.Deprecated('Use resourceErrorDetailDescriptor instead')
const ResourceErrorDetail$json = {
  '1': 'ResourceErrorDetail',
  '2': [
    {
      '1': 'resource',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
  ],
};

/// Descriptor for `ResourceErrorDetail`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resourceErrorDetailDescriptor = $convert.base64Decode(
    'ChNSZXNvdXJjZUVycm9yRGV0YWlsEjkKCHJlc291cmNlGAEgASgLMh0uYW55dHR5LmFwaS52MS'
    '5SZXNvdXJjZUhhbmRsZVIIcmVzb3VyY2U=');

@$core.Deprecated('Use outputSyncLostErrorDetailDescriptor instead')
const OutputSyncLostErrorDetail$json = {
  '1': 'OutputSyncLostErrorDetail',
  '2': [
    {'1': 'terminal_id', '3': 1, '4': 1, '5': 9, '10': 'terminalId'},
    {'1': 'consumer', '3': 2, '4': 1, '5': 9, '10': 'consumer'},
    {'1': 'parser_epoch', '3': 3, '4': 1, '5': 4, '10': 'parserEpoch'},
    {'1': 'dropped_bytes', '3': 4, '4': 1, '5': 4, '10': 'droppedBytes'},
    {'1': 'gap_after_line', '3': 5, '4': 1, '5': 4, '10': 'gapAfterLine'},
  ],
};

/// Descriptor for `OutputSyncLostErrorDetail`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List outputSyncLostErrorDetailDescriptor = $convert.base64Decode(
    'ChlPdXRwdXRTeW5jTG9zdEVycm9yRGV0YWlsEh8KC3Rlcm1pbmFsX2lkGAEgASgJUgp0ZXJtaW'
    '5hbElkEhoKCGNvbnN1bWVyGAIgASgJUghjb25zdW1lchIhCgxwYXJzZXJfZXBvY2gYAyABKARS'
    'C3BhcnNlckVwb2NoEiMKDWRyb3BwZWRfYnl0ZXMYBCABKARSDGRyb3BwZWRCeXRlcxIkCg5nYX'
    'BfYWZ0ZXJfbGluZRgFIAEoBFIMZ2FwQWZ0ZXJMaW5l');

@$core.Deprecated('Use apiErrorDescriptor instead')
const ApiError$json = {
  '1': 'ApiError',
  '2': [
    {
      '1': 'code',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.ApiErrorCode',
      '10': 'code'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'retryable', '3': 3, '4': 1, '5': 8, '10': 'retryable'},
    {'1': 'attempted', '3': 4, '4': 1, '5': 8, '10': 'attempted'},
    {
      '1': 'validation',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ValidationErrorDetail',
      '9': 0,
      '10': 'validation'
    },
    {
      '1': 'stale_session',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StaleSessionErrorDetail',
      '9': 0,
      '10': 'staleSession'
    },
    {
      '1': 'resource',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceErrorDetail',
      '9': 0,
      '10': 'resource'
    },
    {
      '1': 'output_sync_lost',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OutputSyncLostErrorDetail',
      '9': 0,
      '10': 'outputSyncLost'
    },
  ],
  '8': [
    {'1': 'detail'},
  ],
};

/// Descriptor for `ApiError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List apiErrorDescriptor = $convert.base64Decode(
    'CghBcGlFcnJvchIvCgRjb2RlGAEgASgOMhsuYW55dHR5LmFwaS52MS5BcGlFcnJvckNvZGVSBG'
    'NvZGUSGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZRIcCglyZXRyeWFibGUYAyABKAhSCXJldHJ5'
    'YWJsZRIcCglhdHRlbXB0ZWQYBCABKAhSCWF0dGVtcHRlZBJGCgp2YWxpZGF0aW9uGAogASgLMi'
    'QuYW55dHR5LmFwaS52MS5WYWxpZGF0aW9uRXJyb3JEZXRhaWxIAFIKdmFsaWRhdGlvbhJNCg1z'
    'dGFsZV9zZXNzaW9uGAsgASgLMiYuYW55dHR5LmFwaS52MS5TdGFsZVNlc3Npb25FcnJvckRldG'
    'FpbEgAUgxzdGFsZVNlc3Npb24SQAoIcmVzb3VyY2UYDCABKAsyIi5hbnl0dHkuYXBpLnYxLlJl'
    'c291cmNlRXJyb3JEZXRhaWxIAFIIcmVzb3VyY2USVAoQb3V0cHV0X3N5bmNfbG9zdBgNIAEoCz'
    'IoLmFueXR0eS5hcGkudjEuT3V0cHV0U3luY0xvc3RFcnJvckRldGFpbEgAUg5vdXRwdXRTeW5j'
    'TG9zdEIICgZkZXRhaWw=');
