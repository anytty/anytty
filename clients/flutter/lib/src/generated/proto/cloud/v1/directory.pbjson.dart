// This is a generated file - do not edit.
//
// Generated from cloud/v1/directory.proto.

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

import 'package:protobuf/well_known_types/google/protobuf/timestamp.pbjson.dart'
    as $2;

import 'common.pbjson.dart' as $0;
import 'enrollment.pbjson.dart' as $1;

@$core.Deprecated('Use beginClientRouteRequestDescriptor instead')
const BeginClientRouteRequest$json = {
  '1': 'BeginClientRouteRequest',
  '2': [
    {
      '1': 'cloud_route_grant',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SignedEnvelope',
      '10': 'cloudRouteGrant'
    },
  ],
};

/// Descriptor for `BeginClientRouteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List beginClientRouteRequestDescriptor =
    $convert.base64Decode(
        'ChdCZWdpbkNsaWVudFJvdXRlUmVxdWVzdBJLChFjbG91ZF9yb3V0ZV9ncmFudBgBIAEoCzIfLm'
        'FueXR0eS5jbG91ZC52MS5TaWduZWRFbnZlbG9wZVIPY2xvdWRSb3V0ZUdyYW50');

@$core.Deprecated('Use resolveClientRouteRequestDescriptor instead')
const ResolveClientRouteRequest$json = {
  '1': 'ResolveClientRouteRequest',
  '2': [
    {'1': 'challenge_id', '3': 1, '4': 1, '5': 9, '10': 'challengeId'},
    {'1': 'request_id', '3': 2, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'client_proof', '3': 3, '4': 1, '5': 12, '10': 'clientProof'},
  ],
};

/// Descriptor for `ResolveClientRouteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resolveClientRouteRequestDescriptor = $convert.base64Decode(
    'ChlSZXNvbHZlQ2xpZW50Um91dGVSZXF1ZXN0EiEKDGNoYWxsZW5nZV9pZBgBIAEoCVILY2hhbG'
    'xlbmdlSWQSHQoKcmVxdWVzdF9pZBgCIAEoCVIJcmVxdWVzdElkEiEKDGNsaWVudF9wcm9vZhgD'
    'IAEoDFILY2xpZW50UHJvb2Y=');

@$core.Deprecated('Use resolveClientRouteResponseDescriptor instead')
const ResolveClientRouteResponse$json = {
  '1': 'ResolveClientRouteResponse',
  '2': [
    {
      '1': 'edge_locator',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EdgeLocator',
      '10': 'edgeLocator'
    },
  ],
};

/// Descriptor for `ResolveClientRouteResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resolveClientRouteResponseDescriptor =
    $convert.base64Decode(
        'ChpSZXNvbHZlQ2xpZW50Um91dGVSZXNwb25zZRI/CgxlZGdlX2xvY2F0b3IYASABKAsyHC5hbn'
        'l0dHkuY2xvdWQudjEuRWRnZUxvY2F0b3JSC2VkZ2VMb2NhdG9y');

const $core.Map<$core.String, $core.dynamic> DirectoryServiceBase$json = {
  '1': 'DirectoryService',
  '2': [
    {
      '1': 'BeginClientRoute',
      '2': '.anytty.cloud.v1.BeginClientRouteRequest',
      '3': '.anytty.cloud.v1.IdentityChallenge'
    },
    {
      '1': 'ResolveClientRoute',
      '2': '.anytty.cloud.v1.ResolveClientRouteRequest',
      '3': '.anytty.cloud.v1.ResolveClientRouteResponse'
    },
  ],
};

@$core.Deprecated('Use directoryServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    DirectoryServiceBase$messageJson = {
  '.anytty.cloud.v1.BeginClientRouteRequest': BeginClientRouteRequest$json,
  '.anytty.cloud.v1.SignedEnvelope': $0.SignedEnvelope$json,
  '.anytty.cloud.v1.IdentityChallenge': $1.IdentityChallenge$json,
  '.google.protobuf.Timestamp': $2.Timestamp$json,
  '.anytty.cloud.v1.ResolveClientRouteRequest': ResolveClientRouteRequest$json,
  '.anytty.cloud.v1.ResolveClientRouteResponse':
      ResolveClientRouteResponse$json,
  '.anytty.cloud.v1.EdgeLocator': $1.EdgeLocator$json,
};

/// Descriptor for `DirectoryService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List directoryServiceDescriptor = $convert.base64Decode(
    'ChBEaXJlY3RvcnlTZXJ2aWNlEmAKEEJlZ2luQ2xpZW50Um91dGUSKC5hbnl0dHkuY2xvdWQudj'
    'EuQmVnaW5DbGllbnRSb3V0ZVJlcXVlc3QaIi5hbnl0dHkuY2xvdWQudjEuSWRlbnRpdHlDaGFs'
    'bGVuZ2USbQoSUmVzb2x2ZUNsaWVudFJvdXRlEiouYW55dHR5LmNsb3VkLnYxLlJlc29sdmVDbG'
    'llbnRSb3V0ZVJlcXVlc3QaKy5hbnl0dHkuY2xvdWQudjEuUmVzb2x2ZUNsaWVudFJvdXRlUmVz'
    'cG9uc2U=');
