// This is a generated file - do not edit.
//
// Generated from apipb/runtime.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'common.pb.dart' as $0;
import 'runtime.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'runtime.pbenum.dart';

class EndpointProbeRequest extends $pb.GeneratedMessage {
  factory EndpointProbeRequest({
    $core.String? endpointId,
    $core.String? routeOverride,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (routeOverride != null) result.routeOverride = routeOverride;
    return result;
  }

  EndpointProbeRequest._();

  factory EndpointProbeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointProbeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointProbeRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'routeOverride')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointProbeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointProbeRequest copyWith(void Function(EndpointProbeRequest) updates) =>
      super.copyWith((message) => updates(message as EndpointProbeRequest))
          as EndpointProbeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointProbeRequest create() => EndpointProbeRequest._();
  @$core.override
  EndpointProbeRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointProbeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointProbeRequest>(create);
  static EndpointProbeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get routeOverride => $_getSZ(1);
  @$pb.TagNumber(2)
  set routeOverride($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRouteOverride() => $_has(1);
  @$pb.TagNumber(2)
  void clearRouteOverride() => $_clearField(2);
}

class EndpointProbeResult extends $pb.GeneratedMessage {
  factory EndpointProbeResult({
    $0.EndpointSessionStamp? session,
    $core.String? observedPath,
    $core.String? routeSelectionReason,
  }) {
    final result = create();
    if (session != null) result.session = session;
    if (observedPath != null) result.observedPath = observedPath;
    if (routeSelectionReason != null)
      result.routeSelectionReason = routeSelectionReason;
    return result;
  }

  EndpointProbeResult._();

  factory EndpointProbeResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointProbeResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointProbeResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.EndpointSessionStamp>(1, _omitFieldNames ? '' : 'session',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOS(2, _omitFieldNames ? '' : 'observedPath')
    ..aOS(3, _omitFieldNames ? '' : 'routeSelectionReason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointProbeResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointProbeResult copyWith(void Function(EndpointProbeResult) updates) =>
      super.copyWith((message) => updates(message as EndpointProbeResult))
          as EndpointProbeResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointProbeResult create() => EndpointProbeResult._();
  @$core.override
  EndpointProbeResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointProbeResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointProbeResult>(create);
  static EndpointProbeResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.EndpointSessionStamp get session => $_getN(0);
  @$pb.TagNumber(1)
  set session($0.EndpointSessionStamp value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSession() => $_has(0);
  @$pb.TagNumber(1)
  void clearSession() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.EndpointSessionStamp ensureSession() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get observedPath => $_getSZ(1);
  @$pb.TagNumber(2)
  set observedPath($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasObservedPath() => $_has(1);
  @$pb.TagNumber(2)
  void clearObservedPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get routeSelectionReason => $_getSZ(2);
  @$pb.TagNumber(3)
  set routeSelectionReason($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRouteSelectionReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearRouteSelectionReason() => $_clearField(3);
}

class EndpointRuntimeEvent extends $pb.GeneratedMessage {
  factory EndpointRuntimeEvent({
    $core.String? endpointId,
    EndpointRuntimePhase? phase,
    $0.EndpointSessionStamp? session,
    $0.ApiError? error,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (phase != null) result.phase = phase;
    if (session != null) result.session = session;
    if (error != null) result.error = error;
    return result;
  }

  EndpointRuntimeEvent._();

  factory EndpointRuntimeEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointRuntimeEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointRuntimeEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aE<EndpointRuntimePhase>(2, _omitFieldNames ? '' : 'phase',
        enumValues: EndpointRuntimePhase.values)
    ..aOM<$0.EndpointSessionStamp>(3, _omitFieldNames ? '' : 'session',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOM<$0.ApiError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRuntimeEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointRuntimeEvent copyWith(void Function(EndpointRuntimeEvent) updates) =>
      super.copyWith((message) => updates(message as EndpointRuntimeEvent))
          as EndpointRuntimeEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointRuntimeEvent create() => EndpointRuntimeEvent._();
  @$core.override
  EndpointRuntimeEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointRuntimeEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointRuntimeEvent>(create);
  static EndpointRuntimeEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  EndpointRuntimePhase get phase => $_getN(1);
  @$pb.TagNumber(2)
  set phase(EndpointRuntimePhase value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPhase() => $_has(1);
  @$pb.TagNumber(2)
  void clearPhase() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.EndpointSessionStamp get session => $_getN(2);
  @$pb.TagNumber(3)
  set session($0.EndpointSessionStamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSession() => $_has(2);
  @$pb.TagNumber(3)
  void clearSession() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.EndpointSessionStamp ensureSession() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error($0.ApiError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.ApiError ensureError() => $_ensure(3);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
