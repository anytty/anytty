// This is a generated file - do not edit.
//
// Generated from apipb/common.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'common.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'common.pbenum.dart';

class ApiVersion extends $pb.GeneratedMessage {
  factory ApiVersion({
    $core.int? major,
    $core.int? minor,
  }) {
    final result = create();
    if (major != null) result.major = major;
    if (minor != null) result.minor = minor;
    return result;
  }

  ApiVersion._();

  factory ApiVersion.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ApiVersion.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ApiVersion',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'major', fieldType: $pb.PbFieldType.OU3)
    ..aI(2, _omitFieldNames ? '' : 'minor', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApiVersion clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApiVersion copyWith(void Function(ApiVersion) updates) =>
      super.copyWith((message) => updates(message as ApiVersion)) as ApiVersion;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ApiVersion create() => ApiVersion._();
  @$core.override
  ApiVersion createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ApiVersion getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ApiVersion>(create);
  static ApiVersion? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get major => $_getIZ(0);
  @$pb.TagNumber(1)
  set major($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMajor() => $_has(0);
  @$pb.TagNumber(1)
  void clearMajor() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get minor => $_getIZ(1);
  @$pb.TagNumber(2)
  set minor($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMinor() => $_has(1);
  @$pb.TagNumber(2)
  void clearMinor() => $_clearField(2);
}

class EndpointSessionStamp extends $pb.GeneratedMessage {
  factory EndpointSessionStamp({
    $core.String? endpointId,
    $core.String? routeId,
    $fixnum.Int64? generation,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (routeId != null) result.routeId = routeId;
    if (generation != null) result.generation = generation;
    return result;
  }

  EndpointSessionStamp._();

  factory EndpointSessionStamp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EndpointSessionStamp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EndpointSessionStamp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'routeId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSessionStamp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EndpointSessionStamp copyWith(void Function(EndpointSessionStamp) updates) =>
      super.copyWith((message) => updates(message as EndpointSessionStamp))
          as EndpointSessionStamp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EndpointSessionStamp create() => EndpointSessionStamp._();
  @$core.override
  EndpointSessionStamp createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EndpointSessionStamp getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EndpointSessionStamp>(create);
  static EndpointSessionStamp? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get routeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set routeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRouteId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRouteId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get generation => $_getI64(2);
  @$pb.TagNumber(3)
  set generation($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasGeneration() => $_has(2);
  @$pb.TagNumber(3)
  void clearGeneration() => $_clearField(3);
}

class OperationStamp extends $pb.GeneratedMessage {
  factory OperationStamp({
    EndpointSessionStamp? session,
    $core.String? operationId,
  }) {
    final result = create();
    if (session != null) result.session = session;
    if (operationId != null) result.operationId = operationId;
    return result;
  }

  OperationStamp._();

  factory OperationStamp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OperationStamp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OperationStamp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<EndpointSessionStamp>(1, _omitFieldNames ? '' : 'session',
        subBuilder: EndpointSessionStamp.create)
    ..aOS(2, _omitFieldNames ? '' : 'operationId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperationStamp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperationStamp copyWith(void Function(OperationStamp) updates) =>
      super.copyWith((message) => updates(message as OperationStamp))
          as OperationStamp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OperationStamp create() => OperationStamp._();
  @$core.override
  OperationStamp createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OperationStamp getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OperationStamp>(create);
  static OperationStamp? _defaultInstance;

  @$pb.TagNumber(1)
  EndpointSessionStamp get session => $_getN(0);
  @$pb.TagNumber(1)
  set session(EndpointSessionStamp value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSession() => $_has(0);
  @$pb.TagNumber(1)
  void clearSession() => $_clearField(1);
  @$pb.TagNumber(1)
  EndpointSessionStamp ensureSession() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get operationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set operationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOperationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearOperationId() => $_clearField(2);
}

class ResourceHandle extends $pb.GeneratedMessage {
  factory ResourceHandle({
    $core.List<$core.int>? opaqueToken,
    ResourceKind? kind,
    EndpointSessionStamp? session,
    $fixnum.Int64? generation,
  }) {
    final result = create();
    if (opaqueToken != null) result.opaqueToken = opaqueToken;
    if (kind != null) result.kind = kind;
    if (session != null) result.session = session;
    if (generation != null) result.generation = generation;
    return result;
  }

  ResourceHandle._();

  factory ResourceHandle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResourceHandle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResourceHandle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'opaqueToken', $pb.PbFieldType.OY)
    ..aE<ResourceKind>(2, _omitFieldNames ? '' : 'kind',
        enumValues: ResourceKind.values)
    ..aOM<EndpointSessionStamp>(3, _omitFieldNames ? '' : 'session',
        subBuilder: EndpointSessionStamp.create)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceHandle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceHandle copyWith(void Function(ResourceHandle) updates) =>
      super.copyWith((message) => updates(message as ResourceHandle))
          as ResourceHandle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResourceHandle create() => ResourceHandle._();
  @$core.override
  ResourceHandle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResourceHandle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResourceHandle>(create);
  static ResourceHandle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get opaqueToken => $_getN(0);
  @$pb.TagNumber(1)
  set opaqueToken($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOpaqueToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearOpaqueToken() => $_clearField(1);

  @$pb.TagNumber(2)
  ResourceKind get kind => $_getN(1);
  @$pb.TagNumber(2)
  set kind(ResourceKind value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointSessionStamp get session => $_getN(2);
  @$pb.TagNumber(3)
  set session(EndpointSessionStamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSession() => $_has(2);
  @$pb.TagNumber(3)
  void clearSession() => $_clearField(3);
  @$pb.TagNumber(3)
  EndpointSessionStamp ensureSession() => $_ensure(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get generation => $_getI64(3);
  @$pb.TagNumber(4)
  set generation($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasGeneration() => $_has(3);
  @$pb.TagNumber(4)
  void clearGeneration() => $_clearField(4);
}

class RequestContext extends $pb.GeneratedMessage {
  factory RequestContext({
    $core.String? requestId,
    ApiVersion? apiVersion,
    EndpointSessionStamp? session,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (apiVersion != null) result.apiVersion = apiVersion;
    if (session != null) result.session = session;
    return result;
  }

  RequestContext._();

  factory RequestContext.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RequestContext.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestContext',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<ApiVersion>(2, _omitFieldNames ? '' : 'apiVersion',
        subBuilder: ApiVersion.create)
    ..aOM<EndpointSessionStamp>(4, _omitFieldNames ? '' : 'session',
        subBuilder: EndpointSessionStamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestContext clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestContext copyWith(void Function(RequestContext) updates) =>
      super.copyWith((message) => updates(message as RequestContext))
          as RequestContext;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RequestContext create() => RequestContext._();
  @$core.override
  RequestContext createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RequestContext getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RequestContext>(create);
  static RequestContext? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  ApiVersion get apiVersion => $_getN(1);
  @$pb.TagNumber(2)
  set apiVersion(ApiVersion value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasApiVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearApiVersion() => $_clearField(2);
  @$pb.TagNumber(2)
  ApiVersion ensureApiVersion() => $_ensure(1);

  @$pb.TagNumber(4)
  EndpointSessionStamp get session => $_getN(2);
  @$pb.TagNumber(4)
  set session(EndpointSessionStamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSession() => $_has(2);
  @$pb.TagNumber(4)
  void clearSession() => $_clearField(4);
  @$pb.TagNumber(4)
  EndpointSessionStamp ensureSession() => $_ensure(2);
}

class ValidationErrorDetail extends $pb.GeneratedMessage {
  factory ValidationErrorDetail({
    $core.String? field_1,
    $core.String? reason,
  }) {
    final result = create();
    if (field_1 != null) result.field_1 = field_1;
    if (reason != null) result.reason = reason;
    return result;
  }

  ValidationErrorDetail._();

  factory ValidationErrorDetail.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ValidationErrorDetail.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ValidationErrorDetail',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'field')
    ..aOS(2, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ValidationErrorDetail clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ValidationErrorDetail copyWith(
          void Function(ValidationErrorDetail) updates) =>
      super.copyWith((message) => updates(message as ValidationErrorDetail))
          as ValidationErrorDetail;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ValidationErrorDetail create() => ValidationErrorDetail._();
  @$core.override
  ValidationErrorDetail createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ValidationErrorDetail getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ValidationErrorDetail>(create);
  static ValidationErrorDetail? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get field_1 => $_getSZ(0);
  @$pb.TagNumber(1)
  set field_1($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasField_1() => $_has(0);
  @$pb.TagNumber(1)
  void clearField_1() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get reason => $_getSZ(1);
  @$pb.TagNumber(2)
  set reason($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReason() => $_has(1);
  @$pb.TagNumber(2)
  void clearReason() => $_clearField(2);
}

class StaleSessionErrorDetail extends $pb.GeneratedMessage {
  factory StaleSessionErrorDetail({
    EndpointSessionStamp? requested,
    $fixnum.Int64? currentGeneration,
  }) {
    final result = create();
    if (requested != null) result.requested = requested;
    if (currentGeneration != null) result.currentGeneration = currentGeneration;
    return result;
  }

  StaleSessionErrorDetail._();

  factory StaleSessionErrorDetail.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StaleSessionErrorDetail.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StaleSessionErrorDetail',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<EndpointSessionStamp>(1, _omitFieldNames ? '' : 'requested',
        subBuilder: EndpointSessionStamp.create)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'currentGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StaleSessionErrorDetail clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StaleSessionErrorDetail copyWith(
          void Function(StaleSessionErrorDetail) updates) =>
      super.copyWith((message) => updates(message as StaleSessionErrorDetail))
          as StaleSessionErrorDetail;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StaleSessionErrorDetail create() => StaleSessionErrorDetail._();
  @$core.override
  StaleSessionErrorDetail createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StaleSessionErrorDetail getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StaleSessionErrorDetail>(create);
  static StaleSessionErrorDetail? _defaultInstance;

  @$pb.TagNumber(1)
  EndpointSessionStamp get requested => $_getN(0);
  @$pb.TagNumber(1)
  set requested(EndpointSessionStamp value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRequested() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequested() => $_clearField(1);
  @$pb.TagNumber(1)
  EndpointSessionStamp ensureRequested() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get currentGeneration => $_getI64(1);
  @$pb.TagNumber(2)
  set currentGeneration($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCurrentGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearCurrentGeneration() => $_clearField(2);
}

class ResourceErrorDetail extends $pb.GeneratedMessage {
  factory ResourceErrorDetail({
    ResourceHandle? resource,
  }) {
    final result = create();
    if (resource != null) result.resource = resource;
    return result;
  }

  ResourceErrorDetail._();

  factory ResourceErrorDetail.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResourceErrorDetail.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResourceErrorDetail',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<ResourceHandle>(1, _omitFieldNames ? '' : 'resource',
        subBuilder: ResourceHandle.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceErrorDetail clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceErrorDetail copyWith(void Function(ResourceErrorDetail) updates) =>
      super.copyWith((message) => updates(message as ResourceErrorDetail))
          as ResourceErrorDetail;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResourceErrorDetail create() => ResourceErrorDetail._();
  @$core.override
  ResourceErrorDetail createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResourceErrorDetail getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResourceErrorDetail>(create);
  static ResourceErrorDetail? _defaultInstance;

  @$pb.TagNumber(1)
  ResourceHandle get resource => $_getN(0);
  @$pb.TagNumber(1)
  set resource(ResourceHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasResource() => $_has(0);
  @$pb.TagNumber(1)
  void clearResource() => $_clearField(1);
  @$pb.TagNumber(1)
  ResourceHandle ensureResource() => $_ensure(0);
}

class OutputSyncLostErrorDetail extends $pb.GeneratedMessage {
  factory OutputSyncLostErrorDetail({
    $core.String? terminalId,
    $core.String? consumer,
    $fixnum.Int64? parserEpoch,
    $fixnum.Int64? droppedBytes,
    $fixnum.Int64? gapAfterLine,
  }) {
    final result = create();
    if (terminalId != null) result.terminalId = terminalId;
    if (consumer != null) result.consumer = consumer;
    if (parserEpoch != null) result.parserEpoch = parserEpoch;
    if (droppedBytes != null) result.droppedBytes = droppedBytes;
    if (gapAfterLine != null) result.gapAfterLine = gapAfterLine;
    return result;
  }

  OutputSyncLostErrorDetail._();

  factory OutputSyncLostErrorDetail.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OutputSyncLostErrorDetail.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OutputSyncLostErrorDetail',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'terminalId')
    ..aOS(2, _omitFieldNames ? '' : 'consumer')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'parserEpoch', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'droppedBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'gapAfterLine', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OutputSyncLostErrorDetail clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OutputSyncLostErrorDetail copyWith(
          void Function(OutputSyncLostErrorDetail) updates) =>
      super.copyWith((message) => updates(message as OutputSyncLostErrorDetail))
          as OutputSyncLostErrorDetail;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OutputSyncLostErrorDetail create() => OutputSyncLostErrorDetail._();
  @$core.override
  OutputSyncLostErrorDetail createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OutputSyncLostErrorDetail getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OutputSyncLostErrorDetail>(create);
  static OutputSyncLostErrorDetail? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get terminalId => $_getSZ(0);
  @$pb.TagNumber(1)
  set terminalId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminalId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminalId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get consumer => $_getSZ(1);
  @$pb.TagNumber(2)
  set consumer($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConsumer() => $_has(1);
  @$pb.TagNumber(2)
  void clearConsumer() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get parserEpoch => $_getI64(2);
  @$pb.TagNumber(3)
  set parserEpoch($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasParserEpoch() => $_has(2);
  @$pb.TagNumber(3)
  void clearParserEpoch() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get droppedBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set droppedBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDroppedBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearDroppedBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get gapAfterLine => $_getI64(4);
  @$pb.TagNumber(5)
  set gapAfterLine($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGapAfterLine() => $_has(4);
  @$pb.TagNumber(5)
  void clearGapAfterLine() => $_clearField(5);
}

enum ApiError_Detail {
  validation,
  staleSession,
  resource,
  outputSyncLost,
  notSet
}

class ApiError extends $pb.GeneratedMessage {
  factory ApiError({
    ApiErrorCode? code,
    $core.String? message,
    $core.bool? retryable,
    $core.bool? attempted,
    ValidationErrorDetail? validation,
    StaleSessionErrorDetail? staleSession,
    ResourceErrorDetail? resource,
    OutputSyncLostErrorDetail? outputSyncLost,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    if (retryable != null) result.retryable = retryable;
    if (attempted != null) result.attempted = attempted;
    if (validation != null) result.validation = validation;
    if (staleSession != null) result.staleSession = staleSession;
    if (resource != null) result.resource = resource;
    if (outputSyncLost != null) result.outputSyncLost = outputSyncLost;
    return result;
  }

  ApiError._();

  factory ApiError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ApiError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ApiError_Detail> _ApiError_DetailByTag = {
    10: ApiError_Detail.validation,
    11: ApiError_Detail.staleSession,
    12: ApiError_Detail.resource,
    13: ApiError_Detail.outputSyncLost,
    0: ApiError_Detail.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ApiError',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 12, 13])
    ..aE<ApiErrorCode>(1, _omitFieldNames ? '' : 'code',
        enumValues: ApiErrorCode.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aOB(3, _omitFieldNames ? '' : 'retryable')
    ..aOB(4, _omitFieldNames ? '' : 'attempted')
    ..aOM<ValidationErrorDetail>(10, _omitFieldNames ? '' : 'validation',
        subBuilder: ValidationErrorDetail.create)
    ..aOM<StaleSessionErrorDetail>(11, _omitFieldNames ? '' : 'staleSession',
        subBuilder: StaleSessionErrorDetail.create)
    ..aOM<ResourceErrorDetail>(12, _omitFieldNames ? '' : 'resource',
        subBuilder: ResourceErrorDetail.create)
    ..aOM<OutputSyncLostErrorDetail>(
        13, _omitFieldNames ? '' : 'outputSyncLost',
        subBuilder: OutputSyncLostErrorDetail.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApiError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApiError copyWith(void Function(ApiError) updates) =>
      super.copyWith((message) => updates(message as ApiError)) as ApiError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ApiError create() => ApiError._();
  @$core.override
  ApiError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ApiError getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ApiError>(create);
  static ApiError? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  ApiError_Detail whichDetail() => _ApiError_DetailByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  void clearDetail() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  ApiErrorCode get code => $_getN(0);
  @$pb.TagNumber(1)
  set code(ApiErrorCode value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get retryable => $_getBF(2);
  @$pb.TagNumber(3)
  set retryable($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRetryable() => $_has(2);
  @$pb.TagNumber(3)
  void clearRetryable() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get attempted => $_getBF(3);
  @$pb.TagNumber(4)
  set attempted($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAttempted() => $_has(3);
  @$pb.TagNumber(4)
  void clearAttempted() => $_clearField(4);

  @$pb.TagNumber(10)
  ValidationErrorDetail get validation => $_getN(4);
  @$pb.TagNumber(10)
  set validation(ValidationErrorDetail value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasValidation() => $_has(4);
  @$pb.TagNumber(10)
  void clearValidation() => $_clearField(10);
  @$pb.TagNumber(10)
  ValidationErrorDetail ensureValidation() => $_ensure(4);

  @$pb.TagNumber(11)
  StaleSessionErrorDetail get staleSession => $_getN(5);
  @$pb.TagNumber(11)
  set staleSession(StaleSessionErrorDetail value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasStaleSession() => $_has(5);
  @$pb.TagNumber(11)
  void clearStaleSession() => $_clearField(11);
  @$pb.TagNumber(11)
  StaleSessionErrorDetail ensureStaleSession() => $_ensure(5);

  @$pb.TagNumber(12)
  ResourceErrorDetail get resource => $_getN(6);
  @$pb.TagNumber(12)
  set resource(ResourceErrorDetail value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasResource() => $_has(6);
  @$pb.TagNumber(12)
  void clearResource() => $_clearField(12);
  @$pb.TagNumber(12)
  ResourceErrorDetail ensureResource() => $_ensure(6);

  @$pb.TagNumber(13)
  OutputSyncLostErrorDetail get outputSyncLost => $_getN(7);
  @$pb.TagNumber(13)
  set outputSyncLost(OutputSyncLostErrorDetail value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasOutputSyncLost() => $_has(7);
  @$pb.TagNumber(13)
  void clearOutputSyncLost() => $_clearField(13);
  @$pb.TagNumber(13)
  OutputSyncLostErrorDetail ensureOutputSyncLost() => $_ensure(7);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
