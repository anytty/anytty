// This is a generated file - do not edit.
//
// Generated from wirepb/terminal.proto.

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

@$core.Deprecated('Use helloDescriptor instead')
const Hello$json = {
  '1': 'Hello',
  '2': [
    {'1': 'version', '3': 1, '4': 1, '5': 13, '10': 'version'},
    {'1': 'client', '3': 2, '4': 1, '5': 9, '10': 'client'},
    {'1': 'server', '3': 3, '4': 1, '5': 9, '10': 'server'},
  ],
};

/// Descriptor for `Hello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List helloDescriptor = $convert.base64Decode(
    'CgVIZWxsbxIYCgd2ZXJzaW9uGAEgASgNUgd2ZXJzaW9uEhYKBmNsaWVudBgCIAEoCVIGY2xpZW'
    '50EhYKBnNlcnZlchgDIAEoCVIGc2VydmVy');

@$core.Deprecated('Use sessionCloseDescriptor instead')
const SessionClose$json = {
  '1': 'SessionClose',
  '2': [
    {'1': 'version', '3': 1, '4': 1, '5': 13, '10': 'version'},
  ],
};

/// Descriptor for `SessionClose`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sessionCloseDescriptor = $convert
    .base64Decode('CgxTZXNzaW9uQ2xvc2USGAoHdmVyc2lvbhgBIAEoDVIHdmVyc2lvbg==');

@$core.Deprecated('Use requestCancelDescriptor instead')
const RequestCancel$json = {
  '1': 'RequestCancel',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 4, '10': 'id'},
  ],
};

/// Descriptor for `RequestCancel`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestCancelDescriptor =
    $convert.base64Decode('Cg1SZXF1ZXN0Q2FuY2VsEg4KAmlkGAEgASgEUgJpZA==');

@$core.Deprecated('Use requestEnvelopeDescriptor instead')
const RequestEnvelope$json = {
  '1': 'RequestEnvelope',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 4, '10': 'id'},
    {'1': 'method', '3': 2, '4': 1, '5': 9, '10': 'method'},
    {'1': 'params', '3': 3, '4': 1, '5': 12, '10': 'params'},
  ],
};

/// Descriptor for `RequestEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestEnvelopeDescriptor = $convert.base64Decode(
    'Cg9SZXF1ZXN0RW52ZWxvcGUSDgoCaWQYASABKARSAmlkEhYKBm1ldGhvZBgCIAEoCVIGbWV0aG'
    '9kEhYKBnBhcmFtcxgDIAEoDFIGcGFyYW1z');

@$core.Deprecated('Use responseEnvelopeDescriptor instead')
const ResponseEnvelope$json = {
  '1': 'ResponseEnvelope',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 4, '10': 'id'},
    {'1': 'result', '3': 2, '4': 1, '5': 12, '10': 'result'},
  ],
};

/// Descriptor for `ResponseEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List responseEnvelopeDescriptor = $convert.base64Decode(
    'ChBSZXNwb25zZUVudmVsb3BlEg4KAmlkGAEgASgEUgJpZBIWCgZyZXN1bHQYAiABKAxSBnJlc3'
    'VsdA==');

@$core.Deprecated('Use protocolErrorDescriptor instead')
const ProtocolError$json = {
  '1': 'ProtocolError',
  '2': [
    {'1': 'code', '3': 1, '4': 1, '5': 5, '10': 'code'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ProtocolError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List protocolErrorDescriptor = $convert.base64Decode(
    'Cg1Qcm90b2NvbEVycm9yEhIKBGNvZGUYASABKAVSBGNvZGUSGAoHbWVzc2FnZRgCIAEoCVIHbW'
    'Vzc2FnZQ==');

@$core.Deprecated('Use errorEnvelopeDescriptor instead')
const ErrorEnvelope$json = {
  '1': 'ErrorEnvelope',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 4, '10': 'id'},
    {
      '1': 'error',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.protocol.wirepb.ProtocolError',
      '10': 'error'
    },
  ],
};

/// Descriptor for `ErrorEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List errorEnvelopeDescriptor = $convert.base64Decode(
    'Cg1FcnJvckVudmVsb3BlEg4KAmlkGAEgASgEUgJpZBI7CgVlcnJvchgCIAEoCzIlLmFueXR0eS'
    '5wcm90b2NvbC53aXJlcGIuUHJvdG9jb2xFcnJvclIFZXJyb3I=');

@$core.Deprecated('Use fileTransferDataDescriptor instead')
const FileTransferData$json = {
  '1': 'FileTransferData',
  '2': [
    {'1': 'offset', '3': 1, '4': 1, '5': 3, '10': 'offset'},
    {'1': 'data', '3': 2, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `FileTransferData`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferDataDescriptor = $convert.base64Decode(
    'ChBGaWxlVHJhbnNmZXJEYXRhEhYKBm9mZnNldBgBIAEoA1IGb2Zmc2V0EhIKBGRhdGEYAiABKA'
    'xSBGRhdGE=');

@$core.Deprecated('Use fileTransferAckDescriptor instead')
const FileTransferAck$json = {
  '1': 'FileTransferAck',
  '2': [
    {'1': 'offset', '3': 1, '4': 1, '5': 3, '10': 'offset'},
    {'1': 'window_bytes', '3': 2, '4': 1, '5': 3, '10': 'windowBytes'},
  ],
};

/// Descriptor for `FileTransferAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferAckDescriptor = $convert.base64Decode(
    'Cg9GaWxlVHJhbnNmZXJBY2sSFgoGb2Zmc2V0GAEgASgDUgZvZmZzZXQSIQoMd2luZG93X2J5dG'
    'VzGAIgASgDUgt3aW5kb3dCeXRlcw==');

@$core.Deprecated('Use fileTransferFinishDescriptor instead')
const FileTransferFinish$json = {
  '1': 'FileTransferFinish',
  '2': [
    {'1': 'size', '3': 1, '4': 1, '5': 3, '10': 'size'},
    {'1': 'sha256', '3': 2, '4': 1, '5': 12, '10': 'sha256'},
  ],
};

/// Descriptor for `FileTransferFinish`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferFinishDescriptor = $convert.base64Decode(
    'ChJGaWxlVHJhbnNmZXJGaW5pc2gSEgoEc2l6ZRgBIAEoA1IEc2l6ZRIWCgZzaGEyNTYYAiABKA'
    'xSBnNoYTI1Ng==');

@$core.Deprecated('Use fileTransferResultDescriptor instead')
const FileTransferResult$json = {
  '1': 'FileTransferResult',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {'1': 'size', '3': 2, '4': 1, '5': 3, '10': 'size'},
    {'1': 'sha256', '3': 3, '4': 1, '5': 12, '10': 'sha256'},
  ],
};

/// Descriptor for `FileTransferResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferResultDescriptor = $convert.base64Decode(
    'ChJGaWxlVHJhbnNmZXJSZXN1bHQSEgoEcGF0aBgBIAEoCVIEcGF0aBISCgRzaXplGAIgASgDUg'
    'RzaXplEhYKBnNoYTI1NhgDIAEoDFIGc2hhMjU2');
