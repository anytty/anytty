// This is a generated file - do not edit.
//
// Generated from cloud/v1/certificate.proto.

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

@$core.Deprecated('Use edgePublicCertificateStatusDescriptor instead')
const EdgePublicCertificateStatus$json = {
  '1': 'EdgePublicCertificateStatus',
  '2': [
    {'1': 'public_endpoint', '3': 1, '4': 1, '5': 9, '10': 'publicEndpoint'},
    {
      '1': 'certificate_sha256',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'certificateSha256'
    },
    {
      '1': 'not_before',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'notBefore'
    },
    {
      '1': 'not_after',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'notAfter'
    },
    {'1': 'renewal_pending', '3': 5, '4': 1, '5': 8, '10': 'renewalPending'},
    {'1': 'last_error_code', '3': 6, '4': 1, '5': 9, '10': 'lastErrorCode'},
    {
      '1': 'last_error_message',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'lastErrorMessage'
    },
    {
      '1': 'last_attempt_at',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'lastAttemptAt'
    },
    {
      '1': 'applied_at',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'appliedAt'
    },
  ],
};

/// Descriptor for `EdgePublicCertificateStatus`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgePublicCertificateStatusDescriptor = $convert.base64Decode(
    'ChtFZGdlUHVibGljQ2VydGlmaWNhdGVTdGF0dXMSJwoPcHVibGljX2VuZHBvaW50GAEgASgJUg'
    '5wdWJsaWNFbmRwb2ludBItChJjZXJ0aWZpY2F0ZV9zaGEyNTYYAiABKAxSEWNlcnRpZmljYXRl'
    'U2hhMjU2EjkKCm5vdF9iZWZvcmUYAyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUg'
    'lub3RCZWZvcmUSNwoJbm90X2FmdGVyGAQgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFt'
    'cFIIbm90QWZ0ZXISJwoPcmVuZXdhbF9wZW5kaW5nGAUgASgIUg5yZW5ld2FsUGVuZGluZxImCg'
    '9sYXN0X2Vycm9yX2NvZGUYBiABKAlSDWxhc3RFcnJvckNvZGUSLAoSbGFzdF9lcnJvcl9tZXNz'
    'YWdlGAcgASgJUhBsYXN0RXJyb3JNZXNzYWdlEkIKD2xhc3RfYXR0ZW1wdF9hdBgIIAEoCzIaLm'
    'dvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSDWxhc3RBdHRlbXB0QXQSOQoKYXBwbGllZF9hdBgJ'
    'IAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCWFwcGxpZWRBdA==');

@$core.Deprecated('Use edgePublicCertificateRenewRequestDescriptor instead')
const EdgePublicCertificateRenewRequest$json = {
  '1': 'EdgePublicCertificateRenewRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'csr_pem', '3': 2, '4': 1, '5': 12, '10': 'csrPem'},
    {
      '1': 'current_certificate_sha256',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'currentCertificateSha256'
    },
    {
      '1': 'requested_at',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'requestedAt'
    },
  ],
};

/// Descriptor for `EdgePublicCertificateRenewRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgePublicCertificateRenewRequestDescriptor =
    $convert.base64Decode(
        'CiFFZGdlUHVibGljQ2VydGlmaWNhdGVSZW5ld1JlcXVlc3QSHQoKcmVxdWVzdF9pZBgBIAEoCV'
        'IJcmVxdWVzdElkEhcKB2Nzcl9wZW0YAiABKAxSBmNzclBlbRI8ChpjdXJyZW50X2NlcnRpZmlj'
        'YXRlX3NoYTI1NhgDIAEoDFIYY3VycmVudENlcnRpZmljYXRlU2hhMjU2Ej0KDHJlcXVlc3RlZF'
        '9hdBgEIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC3JlcXVlc3RlZEF0');

@$core.Deprecated('Use edgePublicCertificateRenewResponseDescriptor instead')
const EdgePublicCertificateRenewResponse$json = {
  '1': 'EdgePublicCertificateRenewResponse',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'certificate_pem', '3': 2, '4': 1, '5': 12, '10': 'certificatePem'},
    {
      '1': 'certificate_sha256',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'certificateSha256'
    },
    {
      '1': 'not_before',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'notBefore'
    },
    {
      '1': 'not_after',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'notAfter'
    },
  ],
};

/// Descriptor for `EdgePublicCertificateRenewResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgePublicCertificateRenewResponseDescriptor =
    $convert.base64Decode(
        'CiJFZGdlUHVibGljQ2VydGlmaWNhdGVSZW5ld1Jlc3BvbnNlEh0KCnJlcXVlc3RfaWQYASABKA'
        'lSCXJlcXVlc3RJZBInCg9jZXJ0aWZpY2F0ZV9wZW0YAiABKAxSDmNlcnRpZmljYXRlUGVtEi0K'
        'EmNlcnRpZmljYXRlX3NoYTI1NhgDIAEoDFIRY2VydGlmaWNhdGVTaGEyNTYSOQoKbm90X2JlZm'
        '9yZRgEIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCW5vdEJlZm9yZRI3Cglub3Rf'
        'YWZ0ZXIYBSABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUghub3RBZnRlcg==');

@$core.Deprecated('Use edgePublicCertificateAppliedDescriptor instead')
const EdgePublicCertificateApplied$json = {
  '1': 'EdgePublicCertificateApplied',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'status',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgePublicCertificateStatus',
      '10': 'status'
    },
    {'1': 'applied', '3': 3, '4': 1, '5': 8, '10': 'applied'},
    {'1': 'error_code', '3': 4, '4': 1, '5': 9, '10': 'errorCode'},
    {'1': 'error_message', '3': 5, '4': 1, '5': 9, '10': 'errorMessage'},
  ],
};

/// Descriptor for `EdgePublicCertificateApplied`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgePublicCertificateAppliedDescriptor = $convert.base64Decode(
    'ChxFZGdlUHVibGljQ2VydGlmaWNhdGVBcHBsaWVkEh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcX'
    'Vlc3RJZBJECgZzdGF0dXMYAiABKAsyLC5hbnl0dHkuY2xvdWQudjEuRWRnZVB1YmxpY0NlcnRp'
    'ZmljYXRlU3RhdHVzUgZzdGF0dXMSGAoHYXBwbGllZBgDIAEoCFIHYXBwbGllZBIdCgplcnJvcl'
    '9jb2RlGAQgASgJUgllcnJvckNvZGUSIwoNZXJyb3JfbWVzc2FnZRgFIAEoCVIMZXJyb3JNZXNz'
    'YWdl');
