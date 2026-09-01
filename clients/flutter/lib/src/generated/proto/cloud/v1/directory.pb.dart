// This is a generated file - do not edit.
//
// Generated from cloud/v1/directory.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'common.pb.dart' as $0;
import 'enrollment.pb.dart' as $1;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// BeginClientRouteRequest 只提交 DeviceIdentity 签名的发现 grant；Controller 不接收 terminal capability。
class BeginClientRouteRequest extends $pb.GeneratedMessage {
  factory BeginClientRouteRequest({
    $0.SignedEnvelope? cloudRouteGrant,
  }) {
    final result = create();
    if (cloudRouteGrant != null) result.cloudRouteGrant = cloudRouteGrant;
    return result;
  }

  BeginClientRouteRequest._();

  factory BeginClientRouteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BeginClientRouteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BeginClientRouteRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$0.SignedEnvelope>(1, _omitFieldNames ? '' : 'cloudRouteGrant',
        subBuilder: $0.SignedEnvelope.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BeginClientRouteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BeginClientRouteRequest copyWith(
          void Function(BeginClientRouteRequest) updates) =>
      super.copyWith((message) => updates(message as BeginClientRouteRequest))
          as BeginClientRouteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BeginClientRouteRequest create() => BeginClientRouteRequest._();
  @$core.override
  BeginClientRouteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BeginClientRouteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BeginClientRouteRequest>(create);
  static BeginClientRouteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $0.SignedEnvelope get cloudRouteGrant => $_getN(0);
  @$pb.TagNumber(1)
  set cloudRouteGrant($0.SignedEnvelope value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCloudRouteGrant() => $_has(0);
  @$pb.TagNumber(1)
  void clearCloudRouteGrant() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.SignedEnvelope ensureCloudRouteGrant() => $_ensure(0);
}

class ResolveClientRouteRequest extends $pb.GeneratedMessage {
  factory ResolveClientRouteRequest({
    $core.String? challengeId,
    $core.String? requestId,
    $core.List<$core.int>? clientProof,
  }) {
    final result = create();
    if (challengeId != null) result.challengeId = challengeId;
    if (requestId != null) result.requestId = requestId;
    if (clientProof != null) result.clientProof = clientProof;
    return result;
  }

  ResolveClientRouteRequest._();

  factory ResolveClientRouteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResolveClientRouteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResolveClientRouteRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'challengeId')
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'clientProof', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResolveClientRouteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResolveClientRouteRequest copyWith(
          void Function(ResolveClientRouteRequest) updates) =>
      super.copyWith((message) => updates(message as ResolveClientRouteRequest))
          as ResolveClientRouteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResolveClientRouteRequest create() => ResolveClientRouteRequest._();
  @$core.override
  ResolveClientRouteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResolveClientRouteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResolveClientRouteRequest>(create);
  static ResolveClientRouteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get challengeId => $_getSZ(0);
  @$pb.TagNumber(1)
  set challengeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChallengeId() => $_has(0);
  @$pb.TagNumber(1)
  void clearChallengeId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get clientProof => $_getN(2);
  @$pb.TagNumber(3)
  set clientProof($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasClientProof() => $_has(2);
  @$pb.TagNumber(3)
  void clearClientProof() => $_clearField(3);
}

class ResolveClientRouteResponse extends $pb.GeneratedMessage {
  factory ResolveClientRouteResponse({
    $1.EdgeLocator? edgeLocator,
  }) {
    final result = create();
    if (edgeLocator != null) result.edgeLocator = edgeLocator;
    return result;
  }

  ResolveClientRouteResponse._();

  factory ResolveClientRouteResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResolveClientRouteResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResolveClientRouteResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.EdgeLocator>(1, _omitFieldNames ? '' : 'edgeLocator',
        subBuilder: $1.EdgeLocator.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResolveClientRouteResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResolveClientRouteResponse copyWith(
          void Function(ResolveClientRouteResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ResolveClientRouteResponse))
          as ResolveClientRouteResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResolveClientRouteResponse create() => ResolveClientRouteResponse._();
  @$core.override
  ResolveClientRouteResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResolveClientRouteResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResolveClientRouteResponse>(create);
  static ResolveClientRouteResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $1.EdgeLocator get edgeLocator => $_getN(0);
  @$pb.TagNumber(1)
  set edgeLocator($1.EdgeLocator value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeLocator() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeLocator() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.EdgeLocator ensureEdgeLocator() => $_ensure(0);
}

/// DirectoryService 只在首次发现或 Edge locator 失效时解析实时 daemon Presence。
class DirectoryServiceApi {
  final $pb.RpcClient _client;

  DirectoryServiceApi(this._client);

  $async.Future<$1.IdentityChallenge> beginClientRoute(
          $pb.ClientContext? ctx, BeginClientRouteRequest request) =>
      _client.invoke<$1.IdentityChallenge>(ctx, 'DirectoryService',
          'BeginClientRoute', request, $1.IdentityChallenge());
  $async.Future<ResolveClientRouteResponse> resolveClientRoute(
          $pb.ClientContext? ctx, ResolveClientRouteRequest request) =>
      _client.invoke<ResolveClientRouteResponse>(ctx, 'DirectoryService',
          'ResolveClientRoute', request, ResolveClientRouteResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
