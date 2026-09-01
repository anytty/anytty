// This is a generated file - do not edit.
//
// Generated from apipb/access_remote.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import '../cloud/v1/enrollment.pb.dart' as $1;
import '../remoteauthpb/remote_auth.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class ClientAccessIdentityCommand extends $pb.GeneratedMessage {
  factory ClientAccessIdentityCommand({
    $core.List<$core.int>? challenge,
  }) {
    final result = create();
    if (challenge != null) result.challenge = challenge;
    return result;
  }

  ClientAccessIdentityCommand._();

  factory ClientAccessIdentityCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessIdentityCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessIdentityCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'challenge', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessIdentityCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessIdentityCommand copyWith(
          void Function(ClientAccessIdentityCommand) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessIdentityCommand))
          as ClientAccessIdentityCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessIdentityCommand create() =>
      ClientAccessIdentityCommand._();
  @$core.override
  ClientAccessIdentityCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessIdentityCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessIdentityCommand>(create);
  static ClientAccessIdentityCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.List<$core.int> get challenge => $_getN(0);
  @$pb.TagNumber(2)
  set challenge($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(2)
  $core.bool hasChallenge() => $_has(0);
  @$pb.TagNumber(2)
  void clearChallenge() => $_clearField(2);
}

class ClientAccessListCommand extends $pb.GeneratedMessage {
  factory ClientAccessListCommand() => create();

  ClientAccessListCommand._();

  factory ClientAccessListCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessListCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessListCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessListCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessListCommand copyWith(
          void Function(ClientAccessListCommand) updates) =>
      super.copyWith((message) => updates(message as ClientAccessListCommand))
          as ClientAccessListCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessListCommand create() => ClientAccessListCommand._();
  @$core.override
  ClientAccessListCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessListCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessListCommand>(create);
  static ClientAccessListCommand? _defaultInstance;
}

class ClientAccessTicketCreateCommand extends $pb.GeneratedMessage {
  factory ClientAccessTicketCreateCommand({
    $0.ClientAccessTicketCreateRequest? request,
  }) {
    final result = create();
    if (request != null) result.request = request;
    return result;
  }

  ClientAccessTicketCreateCommand._();

  factory ClientAccessTicketCreateCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessTicketCreateCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessTicketCreateCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ClientAccessTicketCreateRequest>(
        2, _omitFieldNames ? '' : 'request',
        subBuilder: $0.ClientAccessTicketCreateRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateCommand copyWith(
          void Function(ClientAccessTicketCreateCommand) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessTicketCreateCommand))
          as ClientAccessTicketCreateCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateCommand create() =>
      ClientAccessTicketCreateCommand._();
  @$core.override
  ClientAccessTicketCreateCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessTicketCreateCommand>(
          create);
  static ClientAccessTicketCreateCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ClientAccessTicketCreateRequest get request => $_getN(0);
  @$pb.TagNumber(2)
  set request($0.ClientAccessTicketCreateRequest value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRequest() => $_has(0);
  @$pb.TagNumber(2)
  void clearRequest() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ClientAccessTicketCreateRequest ensureRequest() => $_ensure(0);
}

class ClientAccessRevokeCommand extends $pb.GeneratedMessage {
  factory ClientAccessRevokeCommand({
    $0.ClientAccessRevokeRequest? request,
  }) {
    final result = create();
    if (request != null) result.request = request;
    return result;
  }

  ClientAccessRevokeCommand._();

  factory ClientAccessRevokeCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessRevokeCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessRevokeCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ClientAccessRevokeRequest>(2, _omitFieldNames ? '' : 'request',
        subBuilder: $0.ClientAccessRevokeRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRevokeCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRevokeCommand copyWith(
          void Function(ClientAccessRevokeCommand) updates) =>
      super.copyWith((message) => updates(message as ClientAccessRevokeCommand))
          as ClientAccessRevokeCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessRevokeCommand create() => ClientAccessRevokeCommand._();
  @$core.override
  ClientAccessRevokeCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessRevokeCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessRevokeCommand>(create);
  static ClientAccessRevokeCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ClientAccessRevokeRequest get request => $_getN(0);
  @$pb.TagNumber(2)
  set request($0.ClientAccessRevokeRequest value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRequest() => $_has(0);
  @$pb.TagNumber(2)
  void clearRequest() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ClientAccessRevokeRequest ensureRequest() => $_ensure(0);
}

class ClientAccessIdentityResult extends $pb.GeneratedMessage {
  factory ClientAccessIdentityResult({
    $0.ClientAccessIdentityResult? identity,
    $core.List<$core.int>? challenge,
    $core.List<$core.int>? proof,
  }) {
    final result = create();
    if (identity != null) result.identity = identity;
    if (challenge != null) result.challenge = challenge;
    if (proof != null) result.proof = proof;
    return result;
  }

  ClientAccessIdentityResult._();

  factory ClientAccessIdentityResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessIdentityResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessIdentityResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ClientAccessIdentityResult>(1, _omitFieldNames ? '' : 'identity',
        subBuilder: $0.ClientAccessIdentityResult.create)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'challenge', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'proof', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessIdentityResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessIdentityResult copyWith(
          void Function(ClientAccessIdentityResult) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessIdentityResult))
          as ClientAccessIdentityResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessIdentityResult create() => ClientAccessIdentityResult._();
  @$core.override
  ClientAccessIdentityResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessIdentityResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessIdentityResult>(create);
  static ClientAccessIdentityResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.ClientAccessIdentityResult get identity => $_getN(0);
  @$pb.TagNumber(1)
  set identity($0.ClientAccessIdentityResult value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentity() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentity() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.ClientAccessIdentityResult ensureIdentity() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.List<$core.int> get challenge => $_getN(1);
  @$pb.TagNumber(2)
  set challenge($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChallenge() => $_has(1);
  @$pb.TagNumber(2)
  void clearChallenge() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get proof => $_getN(2);
  @$pb.TagNumber(3)
  set proof($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasProof() => $_has(2);
  @$pb.TagNumber(3)
  void clearProof() => $_clearField(3);
}

class ClientAccessListResult extends $pb.GeneratedMessage {
  factory ClientAccessListResult({
    $0.ClientAccessListResult? access,
  }) {
    final result = create();
    if (access != null) result.access = access;
    return result;
  }

  ClientAccessListResult._();

  factory ClientAccessListResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessListResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessListResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ClientAccessListResult>(1, _omitFieldNames ? '' : 'access',
        subBuilder: $0.ClientAccessListResult.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessListResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessListResult copyWith(
          void Function(ClientAccessListResult) updates) =>
      super.copyWith((message) => updates(message as ClientAccessListResult))
          as ClientAccessListResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessListResult create() => ClientAccessListResult._();
  @$core.override
  ClientAccessListResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessListResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessListResult>(create);
  static ClientAccessListResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.ClientAccessListResult get access => $_getN(0);
  @$pb.TagNumber(1)
  set access($0.ClientAccessListResult value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccess() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccess() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.ClientAccessListResult ensureAccess() => $_ensure(0);
}

class ClientAccessTicketCreateResult extends $pb.GeneratedMessage {
  factory ClientAccessTicketCreateResult({
    $0.ClientAccessTicketCreateResult? ticket,
  }) {
    final result = create();
    if (ticket != null) result.ticket = ticket;
    return result;
  }

  ClientAccessTicketCreateResult._();

  factory ClientAccessTicketCreateResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessTicketCreateResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessTicketCreateResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ClientAccessTicketCreateResult>(1, _omitFieldNames ? '' : 'ticket',
        subBuilder: $0.ClientAccessTicketCreateResult.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessTicketCreateResult copyWith(
          void Function(ClientAccessTicketCreateResult) updates) =>
      super.copyWith(
              (message) => updates(message as ClientAccessTicketCreateResult))
          as ClientAccessTicketCreateResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateResult create() =>
      ClientAccessTicketCreateResult._();
  @$core.override
  ClientAccessTicketCreateResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessTicketCreateResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessTicketCreateResult>(create);
  static ClientAccessTicketCreateResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.ClientAccessTicketCreateResult get ticket => $_getN(0);
  @$pb.TagNumber(1)
  set ticket($0.ClientAccessTicketCreateResult value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTicket() => $_has(0);
  @$pb.TagNumber(1)
  void clearTicket() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.ClientAccessTicketCreateResult ensureTicket() => $_ensure(0);
}

class ClientAccessRevokeResult extends $pb.GeneratedMessage {
  factory ClientAccessRevokeResult({
    $0.ClientAccessRecord? record,
  }) {
    final result = create();
    if (record != null) result.record = record;
    return result;
  }

  ClientAccessRevokeResult._();

  factory ClientAccessRevokeResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientAccessRevokeResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientAccessRevokeResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ClientAccessRecord>(1, _omitFieldNames ? '' : 'record',
        subBuilder: $0.ClientAccessRecord.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRevokeResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientAccessRevokeResult copyWith(
          void Function(ClientAccessRevokeResult) updates) =>
      super.copyWith((message) => updates(message as ClientAccessRevokeResult))
          as ClientAccessRevokeResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientAccessRevokeResult create() => ClientAccessRevokeResult._();
  @$core.override
  ClientAccessRevokeResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientAccessRevokeResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientAccessRevokeResult>(create);
  static ClientAccessRevokeResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.ClientAccessRecord get record => $_getN(0);
  @$pb.TagNumber(1)
  set record($0.ClientAccessRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRecord() => $_has(0);
  @$pb.TagNumber(1)
  void clearRecord() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.ClientAccessRecord ensureRecord() => $_ensure(0);
}

class RemoteStatusCommand extends $pb.GeneratedMessage {
  factory RemoteStatusCommand() => create();

  RemoteStatusCommand._();

  factory RemoteStatusCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteStatusCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteStatusCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteStatusCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteStatusCommand copyWith(void Function(RemoteStatusCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteStatusCommand))
          as RemoteStatusCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteStatusCommand create() => RemoteStatusCommand._();
  @$core.override
  RemoteStatusCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteStatusCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteStatusCommand>(create);
  static RemoteStatusCommand? _defaultInstance;
}

class RemotePairStartCommand extends $pb.GeneratedMessage {
  factory RemotePairStartCommand({
    $core.String? localPairUrl,
    $core.int? ttlSeconds,
    $core.int? authTtlSeconds,
  }) {
    final result = create();
    if (localPairUrl != null) result.localPairUrl = localPairUrl;
    if (ttlSeconds != null) result.ttlSeconds = ttlSeconds;
    if (authTtlSeconds != null) result.authTtlSeconds = authTtlSeconds;
    return result;
  }

  RemotePairStartCommand._();

  factory RemotePairStartCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemotePairStartCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemotePairStartCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(2, _omitFieldNames ? '' : 'localPairUrl')
    ..aI(3, _omitFieldNames ? '' : 'ttlSeconds')
    ..aI(4, _omitFieldNames ? '' : 'authTtlSeconds')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemotePairStartCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemotePairStartCommand copyWith(
          void Function(RemotePairStartCommand) updates) =>
      super.copyWith((message) => updates(message as RemotePairStartCommand))
          as RemotePairStartCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemotePairStartCommand create() => RemotePairStartCommand._();
  @$core.override
  RemotePairStartCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemotePairStartCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemotePairStartCommand>(create);
  static RemotePairStartCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get localPairUrl => $_getSZ(0);
  @$pb.TagNumber(2)
  set localPairUrl($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasLocalPairUrl() => $_has(0);
  @$pb.TagNumber(2)
  void clearLocalPairUrl() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get ttlSeconds => $_getIZ(1);
  @$pb.TagNumber(3)
  set ttlSeconds($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(3)
  $core.bool hasTtlSeconds() => $_has(1);
  @$pb.TagNumber(3)
  void clearTtlSeconds() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get authTtlSeconds => $_getIZ(2);
  @$pb.TagNumber(4)
  set authTtlSeconds($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(4)
  $core.bool hasAuthTtlSeconds() => $_has(2);
  @$pb.TagNumber(4)
  void clearAuthTtlSeconds() => $_clearField(4);
}

class RemoteLocalEnableCommand extends $pb.GeneratedMessage {
  factory RemoteLocalEnableCommand({
    $core.String? localWebAddress,
    $core.String? iceTcpAddress,
    $core.Iterable<$core.String>? hubUrls,
    $core.String? controlUrl,
    $core.String? accessToken,
    $core.String? region,
    $core.List<$core.int>? localWebPassword,
  }) {
    final result = create();
    if (localWebAddress != null) result.localWebAddress = localWebAddress;
    if (iceTcpAddress != null) result.iceTcpAddress = iceTcpAddress;
    if (hubUrls != null) result.hubUrls.addAll(hubUrls);
    if (controlUrl != null) result.controlUrl = controlUrl;
    if (accessToken != null) result.accessToken = accessToken;
    if (region != null) result.region = region;
    if (localWebPassword != null) result.localWebPassword = localWebPassword;
    return result;
  }

  RemoteLocalEnableCommand._();

  factory RemoteLocalEnableCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteLocalEnableCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteLocalEnableCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(2, _omitFieldNames ? '' : 'localWebAddress')
    ..aOS(3, _omitFieldNames ? '' : 'iceTcpAddress')
    ..pPS(4, _omitFieldNames ? '' : 'hubUrls')
    ..aOS(5, _omitFieldNames ? '' : 'controlUrl')
    ..aOS(6, _omitFieldNames ? '' : 'accessToken')
    ..aOS(7, _omitFieldNames ? '' : 'region')
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'localWebPassword', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalEnableCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalEnableCommand copyWith(
          void Function(RemoteLocalEnableCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteLocalEnableCommand))
          as RemoteLocalEnableCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteLocalEnableCommand create() => RemoteLocalEnableCommand._();
  @$core.override
  RemoteLocalEnableCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteLocalEnableCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteLocalEnableCommand>(create);
  static RemoteLocalEnableCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get localWebAddress => $_getSZ(0);
  @$pb.TagNumber(2)
  set localWebAddress($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasLocalWebAddress() => $_has(0);
  @$pb.TagNumber(2)
  void clearLocalWebAddress() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get iceTcpAddress => $_getSZ(1);
  @$pb.TagNumber(3)
  set iceTcpAddress($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasIceTcpAddress() => $_has(1);
  @$pb.TagNumber(3)
  void clearIceTcpAddress() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get hubUrls => $_getList(2);

  @$pb.TagNumber(5)
  $core.String get controlUrl => $_getSZ(3);
  @$pb.TagNumber(5)
  set controlUrl($core.String value) => $_setString(3, value);
  @$pb.TagNumber(5)
  $core.bool hasControlUrl() => $_has(3);
  @$pb.TagNumber(5)
  void clearControlUrl() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get accessToken => $_getSZ(4);
  @$pb.TagNumber(6)
  set accessToken($core.String value) => $_setString(4, value);
  @$pb.TagNumber(6)
  $core.bool hasAccessToken() => $_has(4);
  @$pb.TagNumber(6)
  void clearAccessToken() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get region => $_getSZ(5);
  @$pb.TagNumber(7)
  set region($core.String value) => $_setString(5, value);
  @$pb.TagNumber(7)
  $core.bool hasRegion() => $_has(5);
  @$pb.TagNumber(7)
  void clearRegion() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.int> get localWebPassword => $_getN(6);
  @$pb.TagNumber(8)
  set localWebPassword($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(8)
  $core.bool hasLocalWebPassword() => $_has(6);
  @$pb.TagNumber(8)
  void clearLocalWebPassword() => $_clearField(8);
}

class RemoteLocalStatusCommand extends $pb.GeneratedMessage {
  factory RemoteLocalStatusCommand() => create();

  RemoteLocalStatusCommand._();

  factory RemoteLocalStatusCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteLocalStatusCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteLocalStatusCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalStatusCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalStatusCommand copyWith(
          void Function(RemoteLocalStatusCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteLocalStatusCommand))
          as RemoteLocalStatusCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteLocalStatusCommand create() => RemoteLocalStatusCommand._();
  @$core.override
  RemoteLocalStatusCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteLocalStatusCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteLocalStatusCommand>(create);
  static RemoteLocalStatusCommand? _defaultInstance;
}

class RemoteLocalDisableCommand extends $pb.GeneratedMessage {
  factory RemoteLocalDisableCommand() => create();

  RemoteLocalDisableCommand._();

  factory RemoteLocalDisableCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteLocalDisableCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteLocalDisableCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalDisableCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalDisableCommand copyWith(
          void Function(RemoteLocalDisableCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteLocalDisableCommand))
          as RemoteLocalDisableCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteLocalDisableCommand create() => RemoteLocalDisableCommand._();
  @$core.override
  RemoteLocalDisableCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteLocalDisableCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteLocalDisableCommand>(create);
  static RemoteLocalDisableCommand? _defaultInstance;
}

class RemoteCloudStatusCommand extends $pb.GeneratedMessage {
  factory RemoteCloudStatusCommand() => create();

  RemoteCloudStatusCommand._();

  factory RemoteCloudStatusCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudStatusCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudStatusCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudStatusCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudStatusCommand copyWith(
          void Function(RemoteCloudStatusCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteCloudStatusCommand))
          as RemoteCloudStatusCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudStatusCommand create() => RemoteCloudStatusCommand._();
  @$core.override
  RemoteCloudStatusCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudStatusCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudStatusCommand>(create);
  static RemoteCloudStatusCommand? _defaultInstance;
}

class RemoteCloudEnableCommand extends $pb.GeneratedMessage {
  factory RemoteCloudEnableCommand() => create();

  RemoteCloudEnableCommand._();

  factory RemoteCloudEnableCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudEnableCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudEnableCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudEnableCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudEnableCommand copyWith(
          void Function(RemoteCloudEnableCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteCloudEnableCommand))
          as RemoteCloudEnableCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudEnableCommand create() => RemoteCloudEnableCommand._();
  @$core.override
  RemoteCloudEnableCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudEnableCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudEnableCommand>(create);
  static RemoteCloudEnableCommand? _defaultInstance;
}

class RemoteCloudDisableCommand extends $pb.GeneratedMessage {
  factory RemoteCloudDisableCommand() => create();

  RemoteCloudDisableCommand._();

  factory RemoteCloudDisableCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudDisableCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudDisableCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudDisableCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudDisableCommand copyWith(
          void Function(RemoteCloudDisableCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteCloudDisableCommand))
          as RemoteCloudDisableCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudDisableCommand create() => RemoteCloudDisableCommand._();
  @$core.override
  RemoteCloudDisableCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudDisableCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudDisableCommand>(create);
  static RemoteCloudDisableCommand? _defaultInstance;
}

class RemoteCloudEdgesCommand extends $pb.GeneratedMessage {
  factory RemoteCloudEdgesCommand() => create();

  RemoteCloudEdgesCommand._();

  factory RemoteCloudEdgesCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudEdgesCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudEdgesCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudEdgesCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudEdgesCommand copyWith(
          void Function(RemoteCloudEdgesCommand) updates) =>
      super.copyWith((message) => updates(message as RemoteCloudEdgesCommand))
          as RemoteCloudEdgesCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudEdgesCommand create() => RemoteCloudEdgesCommand._();
  @$core.override
  RemoteCloudEdgesCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudEdgesCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudEdgesCommand>(create);
  static RemoteCloudEdgesCommand? _defaultInstance;
}

class RemoteCloudPreferEdgeCommand extends $pb.GeneratedMessage {
  factory RemoteCloudPreferEdgeCommand({
    $core.String? edgeId,
    $fixnum.Int64? expectedPreferenceRevision,
  }) {
    final result = create();
    if (edgeId != null) result.edgeId = edgeId;
    if (expectedPreferenceRevision != null)
      result.expectedPreferenceRevision = expectedPreferenceRevision;
    return result;
  }

  RemoteCloudPreferEdgeCommand._();

  factory RemoteCloudPreferEdgeCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudPreferEdgeCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudPreferEdgeCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(2, _omitFieldNames ? '' : 'edgeId')
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'expectedPreferenceRevision',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudPreferEdgeCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudPreferEdgeCommand copyWith(
          void Function(RemoteCloudPreferEdgeCommand) updates) =>
      super.copyWith(
              (message) => updates(message as RemoteCloudPreferEdgeCommand))
          as RemoteCloudPreferEdgeCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudPreferEdgeCommand create() =>
      RemoteCloudPreferEdgeCommand._();
  @$core.override
  RemoteCloudPreferEdgeCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudPreferEdgeCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudPreferEdgeCommand>(create);
  static RemoteCloudPreferEdgeCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get edgeId => $_getSZ(0);
  @$pb.TagNumber(2)
  set edgeId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasEdgeId() => $_has(0);
  @$pb.TagNumber(2)
  void clearEdgeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get expectedPreferenceRevision => $_getI64(1);
  @$pb.TagNumber(3)
  set expectedPreferenceRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedPreferenceRevision() => $_has(1);
  @$pb.TagNumber(3)
  void clearExpectedPreferenceRevision() => $_clearField(3);
}

class RemoteCloudReselectEdgeCommand extends $pb.GeneratedMessage {
  factory RemoteCloudReselectEdgeCommand() => create();

  RemoteCloudReselectEdgeCommand._();

  factory RemoteCloudReselectEdgeCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudReselectEdgeCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudReselectEdgeCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudReselectEdgeCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudReselectEdgeCommand copyWith(
          void Function(RemoteCloudReselectEdgeCommand) updates) =>
      super.copyWith(
              (message) => updates(message as RemoteCloudReselectEdgeCommand))
          as RemoteCloudReselectEdgeCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudReselectEdgeCommand create() =>
      RemoteCloudReselectEdgeCommand._();
  @$core.override
  RemoteCloudReselectEdgeCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudReselectEdgeCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudReselectEdgeCommand>(create);
  static RemoteCloudReselectEdgeCommand? _defaultInstance;
}

class RemoteStatusResult extends $pb.GeneratedMessage {
  factory RemoteStatusResult({
    $core.String? state,
    $core.String? detail,
    $core.String? deviceId,
    $core.String? deviceName,
    $core.String? controlUrl,
    $core.String? hubUrl,
    $core.Iterable<$core.String>? hubUrls,
    $core.String? dataDirectory,
    $core.String? mode,
    $core.bool? allowLan,
    $core.int? terminalCount,
    $fixnum.Int64? updatedAtUnixNano,
  }) {
    final result = create();
    if (state != null) result.state = state;
    if (detail != null) result.detail = detail;
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceName != null) result.deviceName = deviceName;
    if (controlUrl != null) result.controlUrl = controlUrl;
    if (hubUrl != null) result.hubUrl = hubUrl;
    if (hubUrls != null) result.hubUrls.addAll(hubUrls);
    if (dataDirectory != null) result.dataDirectory = dataDirectory;
    if (mode != null) result.mode = mode;
    if (allowLan != null) result.allowLan = allowLan;
    if (terminalCount != null) result.terminalCount = terminalCount;
    if (updatedAtUnixNano != null) result.updatedAtUnixNano = updatedAtUnixNano;
    return result;
  }

  RemoteStatusResult._();

  factory RemoteStatusResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteStatusResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteStatusResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'state')
    ..aOS(2, _omitFieldNames ? '' : 'detail')
    ..aOS(3, _omitFieldNames ? '' : 'deviceId')
    ..aOS(4, _omitFieldNames ? '' : 'deviceName')
    ..aOS(5, _omitFieldNames ? '' : 'controlUrl')
    ..aOS(6, _omitFieldNames ? '' : 'hubUrl')
    ..pPS(7, _omitFieldNames ? '' : 'hubUrls')
    ..aOS(8, _omitFieldNames ? '' : 'dataDirectory')
    ..aOS(9, _omitFieldNames ? '' : 'mode')
    ..aOB(10, _omitFieldNames ? '' : 'allowLan')
    ..aI(11, _omitFieldNames ? '' : 'terminalCount')
    ..aInt64(12, _omitFieldNames ? '' : 'updatedAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteStatusResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteStatusResult copyWith(void Function(RemoteStatusResult) updates) =>
      super.copyWith((message) => updates(message as RemoteStatusResult))
          as RemoteStatusResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteStatusResult create() => RemoteStatusResult._();
  @$core.override
  RemoteStatusResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteStatusResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteStatusResult>(create);
  static RemoteStatusResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get state => $_getSZ(0);
  @$pb.TagNumber(1)
  set state($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasState() => $_has(0);
  @$pb.TagNumber(1)
  void clearState() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get detail => $_getSZ(1);
  @$pb.TagNumber(2)
  set detail($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDetail() => $_has(1);
  @$pb.TagNumber(2)
  void clearDetail() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get deviceName => $_getSZ(3);
  @$pb.TagNumber(4)
  set deviceName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDeviceName() => $_has(3);
  @$pb.TagNumber(4)
  void clearDeviceName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get controlUrl => $_getSZ(4);
  @$pb.TagNumber(5)
  set controlUrl($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasControlUrl() => $_has(4);
  @$pb.TagNumber(5)
  void clearControlUrl() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get hubUrl => $_getSZ(5);
  @$pb.TagNumber(6)
  set hubUrl($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasHubUrl() => $_has(5);
  @$pb.TagNumber(6)
  void clearHubUrl() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get hubUrls => $_getList(6);

  @$pb.TagNumber(8)
  $core.String get dataDirectory => $_getSZ(7);
  @$pb.TagNumber(8)
  set dataDirectory($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasDataDirectory() => $_has(7);
  @$pb.TagNumber(8)
  void clearDataDirectory() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get mode => $_getSZ(8);
  @$pb.TagNumber(9)
  set mode($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasMode() => $_has(8);
  @$pb.TagNumber(9)
  void clearMode() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get allowLan => $_getBF(9);
  @$pb.TagNumber(10)
  set allowLan($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasAllowLan() => $_has(9);
  @$pb.TagNumber(10)
  void clearAllowLan() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get terminalCount => $_getIZ(10);
  @$pb.TagNumber(11)
  set terminalCount($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasTerminalCount() => $_has(10);
  @$pb.TagNumber(11)
  void clearTerminalCount() => $_clearField(11);

  @$pb.TagNumber(12)
  $fixnum.Int64 get updatedAtUnixNano => $_getI64(11);
  @$pb.TagNumber(12)
  set updatedAtUnixNano($fixnum.Int64 value) => $_setInt64(11, value);
  @$pb.TagNumber(12)
  $core.bool hasUpdatedAtUnixNano() => $_has(11);
  @$pb.TagNumber(12)
  void clearUpdatedAtUnixNano() => $_clearField(12);
}

class RemotePairStartResult extends $pb.GeneratedMessage {
  factory RemotePairStartResult({
    $core.String? type,
    $core.String? machineId,
    $core.String? machineName,
    $core.String? localPairUrl,
    $core.String? pairSessionId,
    $core.String? pairSecret,
    $core.String? answerProofSecret,
    $fixnum.Int64? expiresAtUnixNano,
  }) {
    final result = create();
    if (type != null) result.type = type;
    if (machineId != null) result.machineId = machineId;
    if (machineName != null) result.machineName = machineName;
    if (localPairUrl != null) result.localPairUrl = localPairUrl;
    if (pairSessionId != null) result.pairSessionId = pairSessionId;
    if (pairSecret != null) result.pairSecret = pairSecret;
    if (answerProofSecret != null) result.answerProofSecret = answerProofSecret;
    if (expiresAtUnixNano != null) result.expiresAtUnixNano = expiresAtUnixNano;
    return result;
  }

  RemotePairStartResult._();

  factory RemotePairStartResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemotePairStartResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemotePairStartResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'type')
    ..aOS(2, _omitFieldNames ? '' : 'machineId')
    ..aOS(3, _omitFieldNames ? '' : 'machineName')
    ..aOS(4, _omitFieldNames ? '' : 'localPairUrl')
    ..aOS(5, _omitFieldNames ? '' : 'pairSessionId')
    ..aOS(6, _omitFieldNames ? '' : 'pairSecret')
    ..aOS(7, _omitFieldNames ? '' : 'answerProofSecret')
    ..aInt64(8, _omitFieldNames ? '' : 'expiresAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemotePairStartResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemotePairStartResult copyWith(
          void Function(RemotePairStartResult) updates) =>
      super.copyWith((message) => updates(message as RemotePairStartResult))
          as RemotePairStartResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemotePairStartResult create() => RemotePairStartResult._();
  @$core.override
  RemotePairStartResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemotePairStartResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemotePairStartResult>(create);
  static RemotePairStartResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get type => $_getSZ(0);
  @$pb.TagNumber(1)
  set type($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(1)
  void clearType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get machineId => $_getSZ(1);
  @$pb.TagNumber(2)
  set machineId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMachineId() => $_has(1);
  @$pb.TagNumber(2)
  void clearMachineId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get machineName => $_getSZ(2);
  @$pb.TagNumber(3)
  set machineName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMachineName() => $_has(2);
  @$pb.TagNumber(3)
  void clearMachineName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get localPairUrl => $_getSZ(3);
  @$pb.TagNumber(4)
  set localPairUrl($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLocalPairUrl() => $_has(3);
  @$pb.TagNumber(4)
  void clearLocalPairUrl() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get pairSessionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set pairSessionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPairSessionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearPairSessionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get pairSecret => $_getSZ(5);
  @$pb.TagNumber(6)
  set pairSecret($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPairSecret() => $_has(5);
  @$pb.TagNumber(6)
  void clearPairSecret() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get answerProofSecret => $_getSZ(6);
  @$pb.TagNumber(7)
  set answerProofSecret($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAnswerProofSecret() => $_has(6);
  @$pb.TagNumber(7)
  void clearAnswerProofSecret() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get expiresAtUnixNano => $_getI64(7);
  @$pb.TagNumber(8)
  set expiresAtUnixNano($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasExpiresAtUnixNano() => $_has(7);
  @$pb.TagNumber(8)
  void clearExpiresAtUnixNano() => $_clearField(8);
}

class RemoteLocalStatusResult extends $pb.GeneratedMessage {
  factory RemoteLocalStatusResult({
    $core.bool? enabled,
    $core.String? httpUrl,
    $core.String? localWebAddress,
    $core.String? localPairUrl,
    $core.bool? iceTcpEnabled,
    $core.String? iceTcpAddress,
    $core.int? iceTcpPort,
    $fixnum.Int64? updatedAtUnixNano,
    $core.bool? passwordProtected,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (httpUrl != null) result.httpUrl = httpUrl;
    if (localWebAddress != null) result.localWebAddress = localWebAddress;
    if (localPairUrl != null) result.localPairUrl = localPairUrl;
    if (iceTcpEnabled != null) result.iceTcpEnabled = iceTcpEnabled;
    if (iceTcpAddress != null) result.iceTcpAddress = iceTcpAddress;
    if (iceTcpPort != null) result.iceTcpPort = iceTcpPort;
    if (updatedAtUnixNano != null) result.updatedAtUnixNano = updatedAtUnixNano;
    if (passwordProtected != null) result.passwordProtected = passwordProtected;
    return result;
  }

  RemoteLocalStatusResult._();

  factory RemoteLocalStatusResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteLocalStatusResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteLocalStatusResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..aOS(2, _omitFieldNames ? '' : 'httpUrl')
    ..aOS(3, _omitFieldNames ? '' : 'localWebAddress')
    ..aOS(4, _omitFieldNames ? '' : 'localPairUrl')
    ..aOB(5, _omitFieldNames ? '' : 'iceTcpEnabled')
    ..aOS(6, _omitFieldNames ? '' : 'iceTcpAddress')
    ..aI(7, _omitFieldNames ? '' : 'iceTcpPort')
    ..aInt64(8, _omitFieldNames ? '' : 'updatedAtUnixNano')
    ..aOB(9, _omitFieldNames ? '' : 'passwordProtected')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalStatusResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteLocalStatusResult copyWith(
          void Function(RemoteLocalStatusResult) updates) =>
      super.copyWith((message) => updates(message as RemoteLocalStatusResult))
          as RemoteLocalStatusResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteLocalStatusResult create() => RemoteLocalStatusResult._();
  @$core.override
  RemoteLocalStatusResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteLocalStatusResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteLocalStatusResult>(create);
  static RemoteLocalStatusResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get httpUrl => $_getSZ(1);
  @$pb.TagNumber(2)
  set httpUrl($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHttpUrl() => $_has(1);
  @$pb.TagNumber(2)
  void clearHttpUrl() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get localWebAddress => $_getSZ(2);
  @$pb.TagNumber(3)
  set localWebAddress($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLocalWebAddress() => $_has(2);
  @$pb.TagNumber(3)
  void clearLocalWebAddress() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get localPairUrl => $_getSZ(3);
  @$pb.TagNumber(4)
  set localPairUrl($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLocalPairUrl() => $_has(3);
  @$pb.TagNumber(4)
  void clearLocalPairUrl() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get iceTcpEnabled => $_getBF(4);
  @$pb.TagNumber(5)
  set iceTcpEnabled($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasIceTcpEnabled() => $_has(4);
  @$pb.TagNumber(5)
  void clearIceTcpEnabled() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get iceTcpAddress => $_getSZ(5);
  @$pb.TagNumber(6)
  set iceTcpAddress($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIceTcpAddress() => $_has(5);
  @$pb.TagNumber(6)
  void clearIceTcpAddress() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get iceTcpPort => $_getIZ(6);
  @$pb.TagNumber(7)
  set iceTcpPort($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasIceTcpPort() => $_has(6);
  @$pb.TagNumber(7)
  void clearIceTcpPort() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get updatedAtUnixNano => $_getI64(7);
  @$pb.TagNumber(8)
  set updatedAtUnixNano($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasUpdatedAtUnixNano() => $_has(7);
  @$pb.TagNumber(8)
  void clearUpdatedAtUnixNano() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get passwordProtected => $_getBF(8);
  @$pb.TagNumber(9)
  set passwordProtected($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPasswordProtected() => $_has(8);
  @$pb.TagNumber(9)
  void clearPasswordProtected() => $_clearField(9);
}

class RemoteCloudStatusResult extends $pb.GeneratedMessage {
  factory RemoteCloudStatusResult({
    $core.bool? enrolled,
    $core.bool? enabled,
    $core.bool? running,
    $core.String? state,
    $core.String? detail,
    $core.String? daemonId,
    $core.String? accountId,
    $core.String? edgeId,
    $core.String? edgeName,
    $core.String? edgeRegion,
    $core.String? publicEndpoint,
    $core.String? serverName,
    $core.String? lifecycleState,
    $fixnum.Int64? lifecycleRevision,
    $core.bool? ready,
    $core.int? activeSessions,
    $fixnum.Int64? enrolledAtUnixNano,
    $fixnum.Int64? updatedAtUnixNano,
    $core.String? recordPath,
    $core.String? disabledPath,
  }) {
    final result = create();
    if (enrolled != null) result.enrolled = enrolled;
    if (enabled != null) result.enabled = enabled;
    if (running != null) result.running = running;
    if (state != null) result.state = state;
    if (detail != null) result.detail = detail;
    if (daemonId != null) result.daemonId = daemonId;
    if (accountId != null) result.accountId = accountId;
    if (edgeId != null) result.edgeId = edgeId;
    if (edgeName != null) result.edgeName = edgeName;
    if (edgeRegion != null) result.edgeRegion = edgeRegion;
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    if (serverName != null) result.serverName = serverName;
    if (lifecycleState != null) result.lifecycleState = lifecycleState;
    if (lifecycleRevision != null) result.lifecycleRevision = lifecycleRevision;
    if (ready != null) result.ready = ready;
    if (activeSessions != null) result.activeSessions = activeSessions;
    if (enrolledAtUnixNano != null)
      result.enrolledAtUnixNano = enrolledAtUnixNano;
    if (updatedAtUnixNano != null) result.updatedAtUnixNano = updatedAtUnixNano;
    if (recordPath != null) result.recordPath = recordPath;
    if (disabledPath != null) result.disabledPath = disabledPath;
    return result;
  }

  RemoteCloudStatusResult._();

  factory RemoteCloudStatusResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudStatusResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudStatusResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enrolled')
    ..aOB(2, _omitFieldNames ? '' : 'enabled')
    ..aOB(3, _omitFieldNames ? '' : 'running')
    ..aOS(4, _omitFieldNames ? '' : 'state')
    ..aOS(5, _omitFieldNames ? '' : 'detail')
    ..aOS(6, _omitFieldNames ? '' : 'daemonId')
    ..aOS(7, _omitFieldNames ? '' : 'accountId')
    ..aOS(8, _omitFieldNames ? '' : 'edgeId')
    ..aOS(9, _omitFieldNames ? '' : 'edgeName')
    ..aOS(10, _omitFieldNames ? '' : 'edgeRegion')
    ..aOS(11, _omitFieldNames ? '' : 'publicEndpoint')
    ..aOS(12, _omitFieldNames ? '' : 'serverName')
    ..aOS(13, _omitFieldNames ? '' : 'lifecycleState')
    ..a<$fixnum.Int64>(
        14, _omitFieldNames ? '' : 'lifecycleRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(15, _omitFieldNames ? '' : 'ready')
    ..aI(16, _omitFieldNames ? '' : 'activeSessions')
    ..aInt64(17, _omitFieldNames ? '' : 'enrolledAtUnixNano')
    ..aInt64(18, _omitFieldNames ? '' : 'updatedAtUnixNano')
    ..aOS(19, _omitFieldNames ? '' : 'recordPath')
    ..aOS(20, _omitFieldNames ? '' : 'disabledPath')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudStatusResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudStatusResult copyWith(
          void Function(RemoteCloudStatusResult) updates) =>
      super.copyWith((message) => updates(message as RemoteCloudStatusResult))
          as RemoteCloudStatusResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudStatusResult create() => RemoteCloudStatusResult._();
  @$core.override
  RemoteCloudStatusResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudStatusResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudStatusResult>(create);
  static RemoteCloudStatusResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enrolled => $_getBF(0);
  @$pb.TagNumber(1)
  set enrolled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnrolled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnrolled() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get enabled => $_getBF(1);
  @$pb.TagNumber(2)
  set enabled($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEnabled() => $_has(1);
  @$pb.TagNumber(2)
  void clearEnabled() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get running => $_getBF(2);
  @$pb.TagNumber(3)
  set running($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRunning() => $_has(2);
  @$pb.TagNumber(3)
  void clearRunning() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get state => $_getSZ(3);
  @$pb.TagNumber(4)
  set state($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasState() => $_has(3);
  @$pb.TagNumber(4)
  void clearState() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get detail => $_getSZ(4);
  @$pb.TagNumber(5)
  set detail($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDetail() => $_has(4);
  @$pb.TagNumber(5)
  void clearDetail() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get daemonId => $_getSZ(5);
  @$pb.TagNumber(6)
  set daemonId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDaemonId() => $_has(5);
  @$pb.TagNumber(6)
  void clearDaemonId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get accountId => $_getSZ(6);
  @$pb.TagNumber(7)
  set accountId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAccountId() => $_has(6);
  @$pb.TagNumber(7)
  void clearAccountId() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get edgeId => $_getSZ(7);
  @$pb.TagNumber(8)
  set edgeId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasEdgeId() => $_has(7);
  @$pb.TagNumber(8)
  void clearEdgeId() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get edgeName => $_getSZ(8);
  @$pb.TagNumber(9)
  set edgeName($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasEdgeName() => $_has(8);
  @$pb.TagNumber(9)
  void clearEdgeName() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get edgeRegion => $_getSZ(9);
  @$pb.TagNumber(10)
  set edgeRegion($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasEdgeRegion() => $_has(9);
  @$pb.TagNumber(10)
  void clearEdgeRegion() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get publicEndpoint => $_getSZ(10);
  @$pb.TagNumber(11)
  set publicEndpoint($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasPublicEndpoint() => $_has(10);
  @$pb.TagNumber(11)
  void clearPublicEndpoint() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get serverName => $_getSZ(11);
  @$pb.TagNumber(12)
  set serverName($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasServerName() => $_has(11);
  @$pb.TagNumber(12)
  void clearServerName() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get lifecycleState => $_getSZ(12);
  @$pb.TagNumber(13)
  set lifecycleState($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasLifecycleState() => $_has(12);
  @$pb.TagNumber(13)
  void clearLifecycleState() => $_clearField(13);

  @$pb.TagNumber(14)
  $fixnum.Int64 get lifecycleRevision => $_getI64(13);
  @$pb.TagNumber(14)
  set lifecycleRevision($fixnum.Int64 value) => $_setInt64(13, value);
  @$pb.TagNumber(14)
  $core.bool hasLifecycleRevision() => $_has(13);
  @$pb.TagNumber(14)
  void clearLifecycleRevision() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.bool get ready => $_getBF(14);
  @$pb.TagNumber(15)
  set ready($core.bool value) => $_setBool(14, value);
  @$pb.TagNumber(15)
  $core.bool hasReady() => $_has(14);
  @$pb.TagNumber(15)
  void clearReady() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.int get activeSessions => $_getIZ(15);
  @$pb.TagNumber(16)
  set activeSessions($core.int value) => $_setSignedInt32(15, value);
  @$pb.TagNumber(16)
  $core.bool hasActiveSessions() => $_has(15);
  @$pb.TagNumber(16)
  void clearActiveSessions() => $_clearField(16);

  @$pb.TagNumber(17)
  $fixnum.Int64 get enrolledAtUnixNano => $_getI64(16);
  @$pb.TagNumber(17)
  set enrolledAtUnixNano($fixnum.Int64 value) => $_setInt64(16, value);
  @$pb.TagNumber(17)
  $core.bool hasEnrolledAtUnixNano() => $_has(16);
  @$pb.TagNumber(17)
  void clearEnrolledAtUnixNano() => $_clearField(17);

  @$pb.TagNumber(18)
  $fixnum.Int64 get updatedAtUnixNano => $_getI64(17);
  @$pb.TagNumber(18)
  set updatedAtUnixNano($fixnum.Int64 value) => $_setInt64(17, value);
  @$pb.TagNumber(18)
  $core.bool hasUpdatedAtUnixNano() => $_has(17);
  @$pb.TagNumber(18)
  void clearUpdatedAtUnixNano() => $_clearField(18);

  @$pb.TagNumber(19)
  $core.String get recordPath => $_getSZ(18);
  @$pb.TagNumber(19)
  set recordPath($core.String value) => $_setString(18, value);
  @$pb.TagNumber(19)
  $core.bool hasRecordPath() => $_has(18);
  @$pb.TagNumber(19)
  void clearRecordPath() => $_clearField(19);

  @$pb.TagNumber(20)
  $core.String get disabledPath => $_getSZ(19);
  @$pb.TagNumber(20)
  set disabledPath($core.String value) => $_setString(19, value);
  @$pb.TagNumber(20)
  $core.bool hasDisabledPath() => $_has(19);
  @$pb.TagNumber(20)
  void clearDisabledPath() => $_clearField(20);
}

class RemoteCloudEdgesResult extends $pb.GeneratedMessage {
  factory RemoteCloudEdgesResult({
    $1.DaemonEdgeSelection? selection,
  }) {
    final result = create();
    if (selection != null) result.selection = selection;
    return result;
  }

  RemoteCloudEdgesResult._();

  factory RemoteCloudEdgesResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoteCloudEdgesResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoteCloudEdgesResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$1.DaemonEdgeSelection>(1, _omitFieldNames ? '' : 'selection',
        subBuilder: $1.DaemonEdgeSelection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudEdgesResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoteCloudEdgesResult copyWith(
          void Function(RemoteCloudEdgesResult) updates) =>
      super.copyWith((message) => updates(message as RemoteCloudEdgesResult))
          as RemoteCloudEdgesResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoteCloudEdgesResult create() => RemoteCloudEdgesResult._();
  @$core.override
  RemoteCloudEdgesResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoteCloudEdgesResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoteCloudEdgesResult>(create);
  static RemoteCloudEdgesResult? _defaultInstance;

  @$pb.TagNumber(1)
  $1.DaemonEdgeSelection get selection => $_getN(0);
  @$pb.TagNumber(1)
  set selection($1.DaemonEdgeSelection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSelection() => $_has(0);
  @$pb.TagNumber(1)
  void clearSelection() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.DaemonEdgeSelection ensureSelection() => $_ensure(0);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
