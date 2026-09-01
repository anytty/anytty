// This is a generated file - do not edit.
//
// Generated from cloud/v1/operator.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $0;

import 'account.pb.dart' as $1;
import 'commerce.pb.dart' as $2;
import 'edge_config.pb.dart' as $3;
import 'operator.pbenum.dart';
import 'runtime.pbenum.dart' as $4;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'operator.pbenum.dart';

class PageRequest extends $pb.GeneratedMessage {
  factory PageRequest({
    $core.int? pageSize,
    $core.String? cursor,
    $core.String? query,
    $core.String? sort,
  }) {
    final result = create();
    if (pageSize != null) result.pageSize = pageSize;
    if (cursor != null) result.cursor = cursor;
    if (query != null) result.query = query;
    if (sort != null) result.sort = sort;
    return result;
  }

  PageRequest._();

  factory PageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PageRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'pageSize', fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'cursor')
    ..aOS(3, _omitFieldNames ? '' : 'query')
    ..aOS(4, _omitFieldNames ? '' : 'sort')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PageRequest copyWith(void Function(PageRequest) updates) =>
      super.copyWith((message) => updates(message as PageRequest))
          as PageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PageRequest create() => PageRequest._();
  @$core.override
  PageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PageRequest>(create);
  static PageRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get pageSize => $_getIZ(0);
  @$pb.TagNumber(1)
  set pageSize($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPageSize() => $_has(0);
  @$pb.TagNumber(1)
  void clearPageSize() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get cursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set cursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearCursor() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get query => $_getSZ(2);
  @$pb.TagNumber(3)
  set query($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasQuery() => $_has(2);
  @$pb.TagNumber(3)
  void clearQuery() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get sort => $_getSZ(3);
  @$pb.TagNumber(4)
  set sort($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSort() => $_has(3);
  @$pb.TagNumber(4)
  void clearSort() => $_clearField(4);
}

class OperatorOverview extends $pb.GeneratedMessage {
  factory OperatorOverview({
    $fixnum.Int64? edgeTotal,
    $fixnum.Int64? edgeOnline,
    $fixnum.Int64? daemonTotal,
    $fixnum.Int64? daemonOnline,
    $fixnum.Int64? clientSessionOnline,
    $fixnum.Int64? relayBytesCurrentPeriod,
    $core.String? controllerInstanceId,
    $0.Timestamp? generatedAt,
  }) {
    final result = create();
    if (edgeTotal != null) result.edgeTotal = edgeTotal;
    if (edgeOnline != null) result.edgeOnline = edgeOnline;
    if (daemonTotal != null) result.daemonTotal = daemonTotal;
    if (daemonOnline != null) result.daemonOnline = daemonOnline;
    if (clientSessionOnline != null)
      result.clientSessionOnline = clientSessionOnline;
    if (relayBytesCurrentPeriod != null)
      result.relayBytesCurrentPeriod = relayBytesCurrentPeriod;
    if (controllerInstanceId != null)
      result.controllerInstanceId = controllerInstanceId;
    if (generatedAt != null) result.generatedAt = generatedAt;
    return result;
  }

  OperatorOverview._();

  factory OperatorOverview.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OperatorOverview.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OperatorOverview',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'edgeTotal', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'edgeOnline', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'daemonTotal', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'daemonOnline', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'clientSessionOnline', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(8, _omitFieldNames ? '' : 'relayBytesCurrentPeriod',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(9, _omitFieldNames ? '' : 'controllerInstanceId')
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'generatedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperatorOverview clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperatorOverview copyWith(void Function(OperatorOverview) updates) =>
      super.copyWith((message) => updates(message as OperatorOverview))
          as OperatorOverview;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OperatorOverview create() => OperatorOverview._();
  @$core.override
  OperatorOverview createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OperatorOverview getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OperatorOverview>(create);
  static OperatorOverview? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get edgeTotal => $_getI64(0);
  @$pb.TagNumber(1)
  set edgeTotal($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEdgeTotal() => $_has(0);
  @$pb.TagNumber(1)
  void clearEdgeTotal() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get edgeOnline => $_getI64(1);
  @$pb.TagNumber(2)
  set edgeOnline($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEdgeOnline() => $_has(1);
  @$pb.TagNumber(2)
  void clearEdgeOnline() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get daemonTotal => $_getI64(2);
  @$pb.TagNumber(3)
  set daemonTotal($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonTotal() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonTotal() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get daemonOnline => $_getI64(3);
  @$pb.TagNumber(4)
  set daemonOnline($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemonOnline() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemonOnline() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get clientSessionOnline => $_getI64(4);
  @$pb.TagNumber(5)
  set clientSessionOnline($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasClientSessionOnline() => $_has(4);
  @$pb.TagNumber(5)
  void clearClientSessionOnline() => $_clearField(5);

  @$pb.TagNumber(8)
  $fixnum.Int64 get relayBytesCurrentPeriod => $_getI64(5);
  @$pb.TagNumber(8)
  set relayBytesCurrentPeriod($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(8)
  $core.bool hasRelayBytesCurrentPeriod() => $_has(5);
  @$pb.TagNumber(8)
  void clearRelayBytesCurrentPeriod() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get controllerInstanceId => $_getSZ(6);
  @$pb.TagNumber(9)
  set controllerInstanceId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(9)
  $core.bool hasControllerInstanceId() => $_has(6);
  @$pb.TagNumber(9)
  void clearControllerInstanceId() => $_clearField(9);

  @$pb.TagNumber(10)
  $0.Timestamp get generatedAt => $_getN(7);
  @$pb.TagNumber(10)
  set generatedAt($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasGeneratedAt() => $_has(7);
  @$pb.TagNumber(10)
  void clearGeneratedAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensureGeneratedAt() => $_ensure(7);
}

class AccountSummary extends $pb.GeneratedMessage {
  factory AccountSummary({
    $1.AccountProfile? account,
    $core.Iterable<$1.AccountRole>? roles,
    $fixnum.Int64? daemonCount,
    $2.SubscriptionProjection? subscription,
    $2.EffectiveEntitlement? entitlement,
    $2.UsagePeriodProjection? usage,
  }) {
    final result = create();
    if (account != null) result.account = account;
    if (roles != null) result.roles.addAll(roles);
    if (daemonCount != null) result.daemonCount = daemonCount;
    if (subscription != null) result.subscription = subscription;
    if (entitlement != null) result.entitlement = entitlement;
    if (usage != null) result.usage = usage;
    return result;
  }

  AccountSummary._();

  factory AccountSummary.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AccountSummary.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AccountSummary',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: $1.AccountProfile.create)
    ..pc<$1.AccountRole>(2, _omitFieldNames ? '' : 'roles', $pb.PbFieldType.KE,
        valueOf: $1.AccountRole.valueOf,
        enumValues: $1.AccountRole.values,
        defaultEnumValue: $1.AccountRole.ACCOUNT_ROLE_UNSPECIFIED)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'daemonCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$2.SubscriptionProjection>(4, _omitFieldNames ? '' : 'subscription',
        subBuilder: $2.SubscriptionProjection.create)
    ..aOM<$2.EffectiveEntitlement>(5, _omitFieldNames ? '' : 'entitlement',
        subBuilder: $2.EffectiveEntitlement.create)
    ..aOM<$2.UsagePeriodProjection>(6, _omitFieldNames ? '' : 'usage',
        subBuilder: $2.UsagePeriodProjection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountSummary clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountSummary copyWith(void Function(AccountSummary) updates) =>
      super.copyWith((message) => updates(message as AccountSummary))
          as AccountSummary;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AccountSummary create() => AccountSummary._();
  @$core.override
  AccountSummary createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AccountSummary getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AccountSummary>(create);
  static AccountSummary? _defaultInstance;

  @$pb.TagNumber(1)
  $1.AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account($1.AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<$1.AccountRole> get roles => $_getList(1);

  @$pb.TagNumber(3)
  $fixnum.Int64 get daemonCount => $_getI64(2);
  @$pb.TagNumber(3)
  set daemonCount($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonCount() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonCount() => $_clearField(3);

  @$pb.TagNumber(4)
  $2.SubscriptionProjection get subscription => $_getN(3);
  @$pb.TagNumber(4)
  set subscription($2.SubscriptionProjection value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSubscription() => $_has(3);
  @$pb.TagNumber(4)
  void clearSubscription() => $_clearField(4);
  @$pb.TagNumber(4)
  $2.SubscriptionProjection ensureSubscription() => $_ensure(3);

  @$pb.TagNumber(5)
  $2.EffectiveEntitlement get entitlement => $_getN(4);
  @$pb.TagNumber(5)
  set entitlement($2.EffectiveEntitlement value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasEntitlement() => $_has(4);
  @$pb.TagNumber(5)
  void clearEntitlement() => $_clearField(5);
  @$pb.TagNumber(5)
  $2.EffectiveEntitlement ensureEntitlement() => $_ensure(4);

  @$pb.TagNumber(6)
  $2.UsagePeriodProjection get usage => $_getN(5);
  @$pb.TagNumber(6)
  set usage($2.UsagePeriodProjection value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasUsage() => $_has(5);
  @$pb.TagNumber(6)
  void clearUsage() => $_clearField(6);
  @$pb.TagNumber(6)
  $2.UsagePeriodProjection ensureUsage() => $_ensure(5);
}

class RuntimeSessionProjection extends $pb.GeneratedMessage {
  factory RuntimeSessionProjection({
    $core.String? sessionId,
    $core.String? accountId,
    $core.String? daemonId,
    $core.String? edgeId,
    $core.String? clientId,
    $4.ClientProduct? product,
    $fixnum.Int64? generation,
    $0.Timestamp? connectedAt,
    $core.String? accountDisplayName,
    $core.String? accountEmail,
    $core.String? daemonDisplayName,
    $core.String? edgeName,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (accountId != null) result.accountId = accountId;
    if (daemonId != null) result.daemonId = daemonId;
    if (edgeId != null) result.edgeId = edgeId;
    if (clientId != null) result.clientId = clientId;
    if (product != null) result.product = product;
    if (generation != null) result.generation = generation;
    if (connectedAt != null) result.connectedAt = connectedAt;
    if (accountDisplayName != null)
      result.accountDisplayName = accountDisplayName;
    if (accountEmail != null) result.accountEmail = accountEmail;
    if (daemonDisplayName != null) result.daemonDisplayName = daemonDisplayName;
    if (edgeName != null) result.edgeName = edgeName;
    return result;
  }

  RuntimeSessionProjection._();

  factory RuntimeSessionProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeSessionProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeSessionProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'daemonId')
    ..aOS(4, _omitFieldNames ? '' : 'edgeId')
    ..aOS(5, _omitFieldNames ? '' : 'clientId')
    ..aE<$4.ClientProduct>(6, _omitFieldNames ? '' : 'product',
        enumValues: $4.ClientProduct.values)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'connectedAt',
        subBuilder: $0.Timestamp.create)
    ..aOS(10, _omitFieldNames ? '' : 'accountDisplayName')
    ..aOS(11, _omitFieldNames ? '' : 'accountEmail')
    ..aOS(12, _omitFieldNames ? '' : 'daemonDisplayName')
    ..aOS(13, _omitFieldNames ? '' : 'edgeName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeSessionProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeSessionProjection copyWith(
          void Function(RuntimeSessionProjection) updates) =>
      super.copyWith((message) => updates(message as RuntimeSessionProjection))
          as RuntimeSessionProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeSessionProjection create() => RuntimeSessionProjection._();
  @$core.override
  RuntimeSessionProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeSessionProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeSessionProjection>(create);
  static RuntimeSessionProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get daemonId => $_getSZ(2);
  @$pb.TagNumber(3)
  set daemonId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDaemonId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDaemonId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get edgeId => $_getSZ(3);
  @$pb.TagNumber(4)
  set edgeId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEdgeId() => $_has(3);
  @$pb.TagNumber(4)
  void clearEdgeId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get clientId => $_getSZ(4);
  @$pb.TagNumber(5)
  set clientId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasClientId() => $_has(4);
  @$pb.TagNumber(5)
  void clearClientId() => $_clearField(5);

  @$pb.TagNumber(6)
  $4.ClientProduct get product => $_getN(5);
  @$pb.TagNumber(6)
  set product($4.ClientProduct value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasProduct() => $_has(5);
  @$pb.TagNumber(6)
  void clearProduct() => $_clearField(6);

  @$pb.TagNumber(8)
  $fixnum.Int64 get generation => $_getI64(6);
  @$pb.TagNumber(8)
  set generation($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(8)
  $core.bool hasGeneration() => $_has(6);
  @$pb.TagNumber(8)
  void clearGeneration() => $_clearField(8);

  @$pb.TagNumber(9)
  $0.Timestamp get connectedAt => $_getN(7);
  @$pb.TagNumber(9)
  set connectedAt($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasConnectedAt() => $_has(7);
  @$pb.TagNumber(9)
  void clearConnectedAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureConnectedAt() => $_ensure(7);

  @$pb.TagNumber(10)
  $core.String get accountDisplayName => $_getSZ(8);
  @$pb.TagNumber(10)
  set accountDisplayName($core.String value) => $_setString(8, value);
  @$pb.TagNumber(10)
  $core.bool hasAccountDisplayName() => $_has(8);
  @$pb.TagNumber(10)
  void clearAccountDisplayName() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get accountEmail => $_getSZ(9);
  @$pb.TagNumber(11)
  set accountEmail($core.String value) => $_setString(9, value);
  @$pb.TagNumber(11)
  $core.bool hasAccountEmail() => $_has(9);
  @$pb.TagNumber(11)
  void clearAccountEmail() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get daemonDisplayName => $_getSZ(10);
  @$pb.TagNumber(12)
  set daemonDisplayName($core.String value) => $_setString(10, value);
  @$pb.TagNumber(12)
  $core.bool hasDaemonDisplayName() => $_has(10);
  @$pb.TagNumber(12)
  void clearDaemonDisplayName() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get edgeName => $_getSZ(11);
  @$pb.TagNumber(13)
  set edgeName($core.String value) => $_setString(11, value);
  @$pb.TagNumber(13)
  $core.bool hasEdgeName() => $_has(11);
  @$pb.TagNumber(13)
  void clearEdgeName() => $_clearField(13);
}

class OperatorAuditEvent extends $pb.GeneratedMessage {
  factory OperatorAuditEvent({
    $core.String? auditId,
    $core.String? actorAccountId,
    $core.String? actorDisplayName,
    $core.String? action,
    $core.String? resourceType,
    $core.String? resourceId,
    $core.String? reason,
    $core.String? result,
    $core.String? correlationId,
    $0.Timestamp? occurredAt,
  }) {
    final result$ = create();
    if (auditId != null) result$.auditId = auditId;
    if (actorAccountId != null) result$.actorAccountId = actorAccountId;
    if (actorDisplayName != null) result$.actorDisplayName = actorDisplayName;
    if (action != null) result$.action = action;
    if (resourceType != null) result$.resourceType = resourceType;
    if (resourceId != null) result$.resourceId = resourceId;
    if (reason != null) result$.reason = reason;
    if (result != null) result$.result = result;
    if (correlationId != null) result$.correlationId = correlationId;
    if (occurredAt != null) result$.occurredAt = occurredAt;
    return result$;
  }

  OperatorAuditEvent._();

  factory OperatorAuditEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OperatorAuditEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OperatorAuditEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'auditId')
    ..aOS(2, _omitFieldNames ? '' : 'actorAccountId')
    ..aOS(3, _omitFieldNames ? '' : 'actorDisplayName')
    ..aOS(4, _omitFieldNames ? '' : 'action')
    ..aOS(5, _omitFieldNames ? '' : 'resourceType')
    ..aOS(6, _omitFieldNames ? '' : 'resourceId')
    ..aOS(7, _omitFieldNames ? '' : 'reason')
    ..aOS(8, _omitFieldNames ? '' : 'result')
    ..aOS(9, _omitFieldNames ? '' : 'correlationId')
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'occurredAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperatorAuditEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperatorAuditEvent copyWith(void Function(OperatorAuditEvent) updates) =>
      super.copyWith((message) => updates(message as OperatorAuditEvent))
          as OperatorAuditEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OperatorAuditEvent create() => OperatorAuditEvent._();
  @$core.override
  OperatorAuditEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OperatorAuditEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OperatorAuditEvent>(create);
  static OperatorAuditEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get auditId => $_getSZ(0);
  @$pb.TagNumber(1)
  set auditId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAuditId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAuditId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get actorAccountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set actorAccountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasActorAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearActorAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get actorDisplayName => $_getSZ(2);
  @$pb.TagNumber(3)
  set actorDisplayName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasActorDisplayName() => $_has(2);
  @$pb.TagNumber(3)
  void clearActorDisplayName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get action => $_getSZ(3);
  @$pb.TagNumber(4)
  set action($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAction() => $_has(3);
  @$pb.TagNumber(4)
  void clearAction() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get resourceType => $_getSZ(4);
  @$pb.TagNumber(5)
  set resourceType($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasResourceType() => $_has(4);
  @$pb.TagNumber(5)
  void clearResourceType() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get resourceId => $_getSZ(5);
  @$pb.TagNumber(6)
  set resourceId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasResourceId() => $_has(5);
  @$pb.TagNumber(6)
  void clearResourceId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get reason => $_getSZ(6);
  @$pb.TagNumber(7)
  set reason($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasReason() => $_has(6);
  @$pb.TagNumber(7)
  void clearReason() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get result => $_getSZ(7);
  @$pb.TagNumber(8)
  set result($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasResult() => $_has(7);
  @$pb.TagNumber(8)
  void clearResult() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get correlationId => $_getSZ(8);
  @$pb.TagNumber(9)
  set correlationId($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasCorrelationId() => $_has(8);
  @$pb.TagNumber(9)
  void clearCorrelationId() => $_clearField(9);

  @$pb.TagNumber(10)
  $0.Timestamp get occurredAt => $_getN(9);
  @$pb.TagNumber(10)
  set occurredAt($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasOccurredAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearOccurredAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensureOccurredAt() => $_ensure(9);
}

class GetOperatorOverviewRequest extends $pb.GeneratedMessage {
  factory GetOperatorOverviewRequest() => create();

  GetOperatorOverviewRequest._();

  factory GetOperatorOverviewRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetOperatorOverviewRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetOperatorOverviewRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorOverviewRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorOverviewRequest copyWith(
          void Function(GetOperatorOverviewRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetOperatorOverviewRequest))
          as GetOperatorOverviewRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetOperatorOverviewRequest create() => GetOperatorOverviewRequest._();
  @$core.override
  GetOperatorOverviewRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetOperatorOverviewRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetOperatorOverviewRequest>(create);
  static GetOperatorOverviewRequest? _defaultInstance;
}

class GetOperatorOverviewResponse extends $pb.GeneratedMessage {
  factory GetOperatorOverviewResponse({
    OperatorOverview? overview,
  }) {
    final result = create();
    if (overview != null) result.overview = overview;
    return result;
  }

  GetOperatorOverviewResponse._();

  factory GetOperatorOverviewResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetOperatorOverviewResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetOperatorOverviewResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<OperatorOverview>(1, _omitFieldNames ? '' : 'overview',
        subBuilder: OperatorOverview.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorOverviewResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorOverviewResponse copyWith(
          void Function(GetOperatorOverviewResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetOperatorOverviewResponse))
          as GetOperatorOverviewResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetOperatorOverviewResponse create() =>
      GetOperatorOverviewResponse._();
  @$core.override
  GetOperatorOverviewResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetOperatorOverviewResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetOperatorOverviewResponse>(create);
  static GetOperatorOverviewResponse? _defaultInstance;

  @$pb.TagNumber(1)
  OperatorOverview get overview => $_getN(0);
  @$pb.TagNumber(1)
  set overview(OperatorOverview value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasOverview() => $_has(0);
  @$pb.TagNumber(1)
  void clearOverview() => $_clearField(1);
  @$pb.TagNumber(1)
  OperatorOverview ensureOverview() => $_ensure(0);
}

class ListOperatorAccountsRequest extends $pb.GeneratedMessage {
  factory ListOperatorAccountsRequest({
    PageRequest? page,
  }) {
    final result = create();
    if (page != null) result.page = page;
    return result;
  }

  ListOperatorAccountsRequest._();

  factory ListOperatorAccountsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorAccountsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorAccountsRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PageRequest>(1, _omitFieldNames ? '' : 'page',
        subBuilder: PageRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAccountsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAccountsRequest copyWith(
          void Function(ListOperatorAccountsRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ListOperatorAccountsRequest))
          as ListOperatorAccountsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorAccountsRequest create() =>
      ListOperatorAccountsRequest._();
  @$core.override
  ListOperatorAccountsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorAccountsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorAccountsRequest>(create);
  static ListOperatorAccountsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PageRequest get page => $_getN(0);
  @$pb.TagNumber(1)
  set page(PageRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPage() => $_clearField(1);
  @$pb.TagNumber(1)
  PageRequest ensurePage() => $_ensure(0);
}

class ListOperatorAccountsResponse extends $pb.GeneratedMessage {
  factory ListOperatorAccountsResponse({
    $core.Iterable<AccountSummary>? accounts,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (accounts != null) result.accounts.addAll(accounts);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListOperatorAccountsResponse._();

  factory ListOperatorAccountsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorAccountsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorAccountsResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<AccountSummary>(1, _omitFieldNames ? '' : 'accounts',
        subBuilder: AccountSummary.create)
    ..aOS(2, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAccountsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAccountsResponse copyWith(
          void Function(ListOperatorAccountsResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ListOperatorAccountsResponse))
          as ListOperatorAccountsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorAccountsResponse create() =>
      ListOperatorAccountsResponse._();
  @$core.override
  ListOperatorAccountsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorAccountsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorAccountsResponse>(create);
  static ListOperatorAccountsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<AccountSummary> get accounts => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
}

class GetOperatorAccountRequest extends $pb.GeneratedMessage {
  factory GetOperatorAccountRequest({
    $core.String? accountId,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    return result;
  }

  GetOperatorAccountRequest._();

  factory GetOperatorAccountRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetOperatorAccountRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetOperatorAccountRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorAccountRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorAccountRequest copyWith(
          void Function(GetOperatorAccountRequest) updates) =>
      super.copyWith((message) => updates(message as GetOperatorAccountRequest))
          as GetOperatorAccountRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetOperatorAccountRequest create() => GetOperatorAccountRequest._();
  @$core.override
  GetOperatorAccountRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetOperatorAccountRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetOperatorAccountRequest>(create);
  static GetOperatorAccountRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);
}

class GetOperatorAccountResponse extends $pb.GeneratedMessage {
  factory GetOperatorAccountResponse({
    AccountSummary? account,
  }) {
    final result = create();
    if (account != null) result.account = account;
    return result;
  }

  GetOperatorAccountResponse._();

  factory GetOperatorAccountResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetOperatorAccountResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetOperatorAccountResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<AccountSummary>(1, _omitFieldNames ? '' : 'account',
        subBuilder: AccountSummary.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorAccountResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetOperatorAccountResponse copyWith(
          void Function(GetOperatorAccountResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetOperatorAccountResponse))
          as GetOperatorAccountResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetOperatorAccountResponse create() => GetOperatorAccountResponse._();
  @$core.override
  GetOperatorAccountResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetOperatorAccountResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetOperatorAccountResponse>(create);
  static GetOperatorAccountResponse? _defaultInstance;

  @$pb.TagNumber(1)
  AccountSummary get account => $_getN(0);
  @$pb.TagNumber(1)
  set account(AccountSummary value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  AccountSummary ensureAccount() => $_ensure(0);
}

class ProvisionAccountRequest extends $pb.GeneratedMessage {
  factory ProvisionAccountRequest({
    $core.String? email,
    $core.String? displayName,
    $core.String? reason,
  }) {
    final result = create();
    if (email != null) result.email = email;
    if (displayName != null) result.displayName = displayName;
    if (reason != null) result.reason = reason;
    return result;
  }

  ProvisionAccountRequest._();

  factory ProvisionAccountRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProvisionAccountRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProvisionAccountRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'email')
    ..aOS(2, _omitFieldNames ? '' : 'displayName')
    ..aOS(3, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionAccountRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionAccountRequest copyWith(
          void Function(ProvisionAccountRequest) updates) =>
      super.copyWith((message) => updates(message as ProvisionAccountRequest))
          as ProvisionAccountRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProvisionAccountRequest create() => ProvisionAccountRequest._();
  @$core.override
  ProvisionAccountRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProvisionAccountRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProvisionAccountRequest>(create);
  static ProvisionAccountRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get email => $_getSZ(0);
  @$pb.TagNumber(1)
  set email($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEmail() => $_has(0);
  @$pb.TagNumber(1)
  void clearEmail() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get displayName => $_getSZ(1);
  @$pb.TagNumber(2)
  set displayName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDisplayName() => $_has(1);
  @$pb.TagNumber(2)
  void clearDisplayName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get reason => $_getSZ(2);
  @$pb.TagNumber(3)
  set reason($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearReason() => $_clearField(3);
}

class ProvisionAccountResponse extends $pb.GeneratedMessage {
  factory ProvisionAccountResponse({
    $1.AccountProfile? account,
    $core.String? setupCredential,
    $0.Timestamp? expiresAt,
  }) {
    final result = create();
    if (account != null) result.account = account;
    if (setupCredential != null) result.setupCredential = setupCredential;
    if (expiresAt != null) result.expiresAt = expiresAt;
    return result;
  }

  ProvisionAccountResponse._();

  factory ProvisionAccountResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProvisionAccountResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProvisionAccountResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: $1.AccountProfile.create)
    ..aOS(2, _omitFieldNames ? '' : 'setupCredential')
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionAccountResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionAccountResponse copyWith(
          void Function(ProvisionAccountResponse) updates) =>
      super.copyWith((message) => updates(message as ProvisionAccountResponse))
          as ProvisionAccountResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProvisionAccountResponse create() => ProvisionAccountResponse._();
  @$core.override
  ProvisionAccountResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProvisionAccountResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProvisionAccountResponse>(create);
  static ProvisionAccountResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $1.AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account($1.AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get setupCredential => $_getSZ(1);
  @$pb.TagNumber(2)
  set setupCredential($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSetupCredential() => $_has(1);
  @$pb.TagNumber(2)
  void clearSetupCredential() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get expiresAt => $_getN(2);
  @$pb.TagNumber(3)
  set expiresAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasExpiresAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpiresAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureExpiresAt() => $_ensure(2);
}

class ResetAccountSetupRequest extends $pb.GeneratedMessage {
  factory ResetAccountSetupRequest({
    $core.String? accountId,
    $core.String? reason,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (reason != null) result.reason = reason;
    return result;
  }

  ResetAccountSetupRequest._();

  factory ResetAccountSetupRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResetAccountSetupRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResetAccountSetupRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOS(2, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResetAccountSetupRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResetAccountSetupRequest copyWith(
          void Function(ResetAccountSetupRequest) updates) =>
      super.copyWith((message) => updates(message as ResetAccountSetupRequest))
          as ResetAccountSetupRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResetAccountSetupRequest create() => ResetAccountSetupRequest._();
  @$core.override
  ResetAccountSetupRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResetAccountSetupRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResetAccountSetupRequest>(create);
  static ResetAccountSetupRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get reason => $_getSZ(1);
  @$pb.TagNumber(2)
  set reason($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReason() => $_has(1);
  @$pb.TagNumber(2)
  void clearReason() => $_clearField(2);
}

class ResetAccountSetupResponse extends $pb.GeneratedMessage {
  factory ResetAccountSetupResponse({
    $1.AccountProfile? account,
    $core.String? setupCredential,
    $0.Timestamp? expiresAt,
  }) {
    final result = create();
    if (account != null) result.account = account;
    if (setupCredential != null) result.setupCredential = setupCredential;
    if (expiresAt != null) result.expiresAt = expiresAt;
    return result;
  }

  ResetAccountSetupResponse._();

  factory ResetAccountSetupResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResetAccountSetupResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResetAccountSetupResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: $1.AccountProfile.create)
    ..aOS(2, _omitFieldNames ? '' : 'setupCredential')
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResetAccountSetupResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResetAccountSetupResponse copyWith(
          void Function(ResetAccountSetupResponse) updates) =>
      super.copyWith((message) => updates(message as ResetAccountSetupResponse))
          as ResetAccountSetupResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResetAccountSetupResponse create() => ResetAccountSetupResponse._();
  @$core.override
  ResetAccountSetupResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResetAccountSetupResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResetAccountSetupResponse>(create);
  static ResetAccountSetupResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $1.AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account($1.AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get setupCredential => $_getSZ(1);
  @$pb.TagNumber(2)
  set setupCredential($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSetupCredential() => $_has(1);
  @$pb.TagNumber(2)
  void clearSetupCredential() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get expiresAt => $_getN(2);
  @$pb.TagNumber(3)
  set expiresAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasExpiresAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpiresAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureExpiresAt() => $_ensure(2);
}

class ListRuntimeSessionsRequest extends $pb.GeneratedMessage {
  factory ListRuntimeSessionsRequest({
    PageRequest? page,
  }) {
    final result = create();
    if (page != null) result.page = page;
    return result;
  }

  ListRuntimeSessionsRequest._();

  factory ListRuntimeSessionsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListRuntimeSessionsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListRuntimeSessionsRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PageRequest>(1, _omitFieldNames ? '' : 'page',
        subBuilder: PageRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListRuntimeSessionsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListRuntimeSessionsRequest copyWith(
          void Function(ListRuntimeSessionsRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ListRuntimeSessionsRequest))
          as ListRuntimeSessionsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListRuntimeSessionsRequest create() => ListRuntimeSessionsRequest._();
  @$core.override
  ListRuntimeSessionsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListRuntimeSessionsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListRuntimeSessionsRequest>(create);
  static ListRuntimeSessionsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PageRequest get page => $_getN(0);
  @$pb.TagNumber(1)
  set page(PageRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPage() => $_clearField(1);
  @$pb.TagNumber(1)
  PageRequest ensurePage() => $_ensure(0);
}

class ListRuntimeSessionsResponse extends $pb.GeneratedMessage {
  factory ListRuntimeSessionsResponse({
    $core.Iterable<RuntimeSessionProjection>? sessions,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (sessions != null) result.sessions.addAll(sessions);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListRuntimeSessionsResponse._();

  factory ListRuntimeSessionsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListRuntimeSessionsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListRuntimeSessionsResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<RuntimeSessionProjection>(1, _omitFieldNames ? '' : 'sessions',
        subBuilder: RuntimeSessionProjection.create)
    ..aOS(2, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListRuntimeSessionsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListRuntimeSessionsResponse copyWith(
          void Function(ListRuntimeSessionsResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ListRuntimeSessionsResponse))
          as ListRuntimeSessionsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListRuntimeSessionsResponse create() =>
      ListRuntimeSessionsResponse._();
  @$core.override
  ListRuntimeSessionsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListRuntimeSessionsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListRuntimeSessionsResponse>(create);
  static ListRuntimeSessionsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<RuntimeSessionProjection> get sessions => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
}

class ListOperatorOrdersRequest extends $pb.GeneratedMessage {
  factory ListOperatorOrdersRequest({
    PageRequest? page,
  }) {
    final result = create();
    if (page != null) result.page = page;
    return result;
  }

  ListOperatorOrdersRequest._();

  factory ListOperatorOrdersRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorOrdersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorOrdersRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PageRequest>(1, _omitFieldNames ? '' : 'page',
        subBuilder: PageRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorOrdersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorOrdersRequest copyWith(
          void Function(ListOperatorOrdersRequest) updates) =>
      super.copyWith((message) => updates(message as ListOperatorOrdersRequest))
          as ListOperatorOrdersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorOrdersRequest create() => ListOperatorOrdersRequest._();
  @$core.override
  ListOperatorOrdersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorOrdersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorOrdersRequest>(create);
  static ListOperatorOrdersRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PageRequest get page => $_getN(0);
  @$pb.TagNumber(1)
  set page(PageRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPage() => $_clearField(1);
  @$pb.TagNumber(1)
  PageRequest ensurePage() => $_ensure(0);
}

class ListOperatorOrdersResponse extends $pb.GeneratedMessage {
  factory ListOperatorOrdersResponse({
    $core.Iterable<$2.OrderProjection>? orders,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (orders != null) result.orders.addAll(orders);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListOperatorOrdersResponse._();

  factory ListOperatorOrdersResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorOrdersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorOrdersResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<$2.OrderProjection>(1, _omitFieldNames ? '' : 'orders',
        subBuilder: $2.OrderProjection.create)
    ..aOS(2, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorOrdersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorOrdersResponse copyWith(
          void Function(ListOperatorOrdersResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ListOperatorOrdersResponse))
          as ListOperatorOrdersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorOrdersResponse create() => ListOperatorOrdersResponse._();
  @$core.override
  ListOperatorOrdersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorOrdersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorOrdersResponse>(create);
  static ListOperatorOrdersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$2.OrderProjection> get orders => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
}

class ListOperatorSubscriptionsRequest extends $pb.GeneratedMessage {
  factory ListOperatorSubscriptionsRequest({
    PageRequest? page,
  }) {
    final result = create();
    if (page != null) result.page = page;
    return result;
  }

  ListOperatorSubscriptionsRequest._();

  factory ListOperatorSubscriptionsRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorSubscriptionsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorSubscriptionsRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PageRequest>(1, _omitFieldNames ? '' : 'page',
        subBuilder: PageRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorSubscriptionsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorSubscriptionsRequest copyWith(
          void Function(ListOperatorSubscriptionsRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ListOperatorSubscriptionsRequest))
          as ListOperatorSubscriptionsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorSubscriptionsRequest create() =>
      ListOperatorSubscriptionsRequest._();
  @$core.override
  ListOperatorSubscriptionsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorSubscriptionsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorSubscriptionsRequest>(
          create);
  static ListOperatorSubscriptionsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PageRequest get page => $_getN(0);
  @$pb.TagNumber(1)
  set page(PageRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPage() => $_clearField(1);
  @$pb.TagNumber(1)
  PageRequest ensurePage() => $_ensure(0);
}

class ListOperatorSubscriptionsResponse extends $pb.GeneratedMessage {
  factory ListOperatorSubscriptionsResponse({
    $core.Iterable<$2.SubscriptionProjection>? subscriptions,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (subscriptions != null) result.subscriptions.addAll(subscriptions);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListOperatorSubscriptionsResponse._();

  factory ListOperatorSubscriptionsResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorSubscriptionsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorSubscriptionsResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<$2.SubscriptionProjection>(1, _omitFieldNames ? '' : 'subscriptions',
        subBuilder: $2.SubscriptionProjection.create)
    ..aOS(2, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorSubscriptionsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorSubscriptionsResponse copyWith(
          void Function(ListOperatorSubscriptionsResponse) updates) =>
      super.copyWith((message) =>
              updates(message as ListOperatorSubscriptionsResponse))
          as ListOperatorSubscriptionsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorSubscriptionsResponse create() =>
      ListOperatorSubscriptionsResponse._();
  @$core.override
  ListOperatorSubscriptionsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorSubscriptionsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorSubscriptionsResponse>(
          create);
  static ListOperatorSubscriptionsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$2.SubscriptionProjection> get subscriptions => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
}

class ListOperatorUsageRequest extends $pb.GeneratedMessage {
  factory ListOperatorUsageRequest({
    PageRequest? page,
  }) {
    final result = create();
    if (page != null) result.page = page;
    return result;
  }

  ListOperatorUsageRequest._();

  factory ListOperatorUsageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorUsageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorUsageRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PageRequest>(1, _omitFieldNames ? '' : 'page',
        subBuilder: PageRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorUsageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorUsageRequest copyWith(
          void Function(ListOperatorUsageRequest) updates) =>
      super.copyWith((message) => updates(message as ListOperatorUsageRequest))
          as ListOperatorUsageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorUsageRequest create() => ListOperatorUsageRequest._();
  @$core.override
  ListOperatorUsageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorUsageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorUsageRequest>(create);
  static ListOperatorUsageRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PageRequest get page => $_getN(0);
  @$pb.TagNumber(1)
  set page(PageRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPage() => $_clearField(1);
  @$pb.TagNumber(1)
  PageRequest ensurePage() => $_ensure(0);
}

class ListOperatorUsageResponse extends $pb.GeneratedMessage {
  factory ListOperatorUsageResponse({
    $core.Iterable<$2.UsagePeriodProjection>? accounts,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (accounts != null) result.accounts.addAll(accounts);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListOperatorUsageResponse._();

  factory ListOperatorUsageResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorUsageResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorUsageResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<$2.UsagePeriodProjection>(1, _omitFieldNames ? '' : 'accounts',
        subBuilder: $2.UsagePeriodProjection.create)
    ..aOS(3, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorUsageResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorUsageResponse copyWith(
          void Function(ListOperatorUsageResponse) updates) =>
      super.copyWith((message) => updates(message as ListOperatorUsageResponse))
          as ListOperatorUsageResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorUsageResponse create() => ListOperatorUsageResponse._();
  @$core.override
  ListOperatorUsageResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorUsageResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorUsageResponse>(create);
  static ListOperatorUsageResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$2.UsagePeriodProjection> get accounts => $_getList(0);

  @$pb.TagNumber(3)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(3)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(3)
  void clearNextCursor() => $_clearField(3);
}

class ListOperatorAuditRequest extends $pb.GeneratedMessage {
  factory ListOperatorAuditRequest({
    PageRequest? page,
  }) {
    final result = create();
    if (page != null) result.page = page;
    return result;
  }

  ListOperatorAuditRequest._();

  factory ListOperatorAuditRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorAuditRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorAuditRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PageRequest>(1, _omitFieldNames ? '' : 'page',
        subBuilder: PageRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAuditRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAuditRequest copyWith(
          void Function(ListOperatorAuditRequest) updates) =>
      super.copyWith((message) => updates(message as ListOperatorAuditRequest))
          as ListOperatorAuditRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorAuditRequest create() => ListOperatorAuditRequest._();
  @$core.override
  ListOperatorAuditRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorAuditRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorAuditRequest>(create);
  static ListOperatorAuditRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PageRequest get page => $_getN(0);
  @$pb.TagNumber(1)
  set page(PageRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPage() => $_has(0);
  @$pb.TagNumber(1)
  void clearPage() => $_clearField(1);
  @$pb.TagNumber(1)
  PageRequest ensurePage() => $_ensure(0);
}

class ListOperatorAuditResponse extends $pb.GeneratedMessage {
  factory ListOperatorAuditResponse({
    $core.Iterable<OperatorAuditEvent>? events,
    $core.String? nextCursor,
  }) {
    final result = create();
    if (events != null) result.events.addAll(events);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListOperatorAuditResponse._();

  factory ListOperatorAuditResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOperatorAuditResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOperatorAuditResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<OperatorAuditEvent>(1, _omitFieldNames ? '' : 'events',
        subBuilder: OperatorAuditEvent.create)
    ..aOS(2, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAuditResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOperatorAuditResponse copyWith(
          void Function(ListOperatorAuditResponse) updates) =>
      super.copyWith((message) => updates(message as ListOperatorAuditResponse))
          as ListOperatorAuditResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOperatorAuditResponse create() => ListOperatorAuditResponse._();
  @$core.override
  ListOperatorAuditResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOperatorAuditResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOperatorAuditResponse>(create);
  static ListOperatorAuditResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<OperatorAuditEvent> get events => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get nextCursor => $_getSZ(1);
  @$pb.TagNumber(2)
  set nextCursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
}

class SetAccountStateRequest extends $pb.GeneratedMessage {
  factory SetAccountStateRequest({
    $core.String? accountId,
    $1.AccountState? state,
    $fixnum.Int64? expectedRevision,
    $core.String? reason,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (state != null) result.state = state;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (reason != null) result.reason = reason;
    return result;
  }

  SetAccountStateRequest._();

  factory SetAccountStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetAccountStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetAccountStateRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aE<$1.AccountState>(2, _omitFieldNames ? '' : 'state',
        enumValues: $1.AccountState.values)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(4, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountStateRequest copyWith(
          void Function(SetAccountStateRequest) updates) =>
      super.copyWith((message) => updates(message as SetAccountStateRequest))
          as SetAccountStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetAccountStateRequest create() => SetAccountStateRequest._();
  @$core.override
  SetAccountStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetAccountStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetAccountStateRequest>(create);
  static SetAccountStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $1.AccountState get state => $_getN(1);
  @$pb.TagNumber(2)
  set state($1.AccountState value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get expectedRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpectedRevision() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get reason => $_getSZ(3);
  @$pb.TagNumber(4)
  set reason($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReason() => $_has(3);
  @$pb.TagNumber(4)
  void clearReason() => $_clearField(4);
}

class SetAccountStateResponse extends $pb.GeneratedMessage {
  factory SetAccountStateResponse({
    $1.AccountProfile? account,
  }) {
    final result = create();
    if (account != null) result.account = account;
    return result;
  }

  SetAccountStateResponse._();

  factory SetAccountStateResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetAccountStateResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetAccountStateResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<$1.AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: $1.AccountProfile.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountStateResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountStateResponse copyWith(
          void Function(SetAccountStateResponse) updates) =>
      super.copyWith((message) => updates(message as SetAccountStateResponse))
          as SetAccountStateResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetAccountStateResponse create() => SetAccountStateResponse._();
  @$core.override
  SetAccountStateResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetAccountStateResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetAccountStateResponse>(create);
  static SetAccountStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $1.AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account($1.AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.AccountProfile ensureAccount() => $_ensure(0);
}

class SetAccountRoleRequest extends $pb.GeneratedMessage {
  factory SetAccountRoleRequest({
    $core.String? accountId,
    $1.AccountRole? role,
    $core.bool? enabled,
    $core.String? reason,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (role != null) result.role = role;
    if (enabled != null) result.enabled = enabled;
    if (reason != null) result.reason = reason;
    return result;
  }

  SetAccountRoleRequest._();

  factory SetAccountRoleRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetAccountRoleRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetAccountRoleRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aE<$1.AccountRole>(2, _omitFieldNames ? '' : 'role',
        enumValues: $1.AccountRole.values)
    ..aOB(3, _omitFieldNames ? '' : 'enabled')
    ..aOS(4, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountRoleRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountRoleRequest copyWith(
          void Function(SetAccountRoleRequest) updates) =>
      super.copyWith((message) => updates(message as SetAccountRoleRequest))
          as SetAccountRoleRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetAccountRoleRequest create() => SetAccountRoleRequest._();
  @$core.override
  SetAccountRoleRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetAccountRoleRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetAccountRoleRequest>(create);
  static SetAccountRoleRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $1.AccountRole get role => $_getN(1);
  @$pb.TagNumber(2)
  set role($1.AccountRole value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRole() => $_has(1);
  @$pb.TagNumber(2)
  void clearRole() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get enabled => $_getBF(2);
  @$pb.TagNumber(3)
  set enabled($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEnabled() => $_has(2);
  @$pb.TagNumber(3)
  void clearEnabled() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get reason => $_getSZ(3);
  @$pb.TagNumber(4)
  set reason($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReason() => $_has(3);
  @$pb.TagNumber(4)
  void clearReason() => $_clearField(4);
}

class SetAccountRoleResponse extends $pb.GeneratedMessage {
  factory SetAccountRoleResponse({
    $core.Iterable<$1.AccountRole>? roles,
  }) {
    final result = create();
    if (roles != null) result.roles.addAll(roles);
    return result;
  }

  SetAccountRoleResponse._();

  factory SetAccountRoleResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetAccountRoleResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetAccountRoleResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pc<$1.AccountRole>(1, _omitFieldNames ? '' : 'roles', $pb.PbFieldType.KE,
        valueOf: $1.AccountRole.valueOf,
        enumValues: $1.AccountRole.values,
        defaultEnumValue: $1.AccountRole.ACCOUNT_ROLE_UNSPECIFIED)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountRoleResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetAccountRoleResponse copyWith(
          void Function(SetAccountRoleResponse) updates) =>
      super.copyWith((message) => updates(message as SetAccountRoleResponse))
          as SetAccountRoleResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetAccountRoleResponse create() => SetAccountRoleResponse._();
  @$core.override
  SetAccountRoleResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetAccountRoleResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetAccountRoleResponse>(create);
  static SetAccountRoleResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$1.AccountRole> get roles => $_getList(0);
}

class DisconnectDaemonRequest extends $pb.GeneratedMessage {
  factory DisconnectDaemonRequest({
    $core.String? daemonId,
    $fixnum.Int64? generation,
    $core.String? reason,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (generation != null) result.generation = generation;
    if (reason != null) result.reason = reason;
    return result;
  }

  DisconnectDaemonRequest._();

  factory DisconnectDaemonRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DisconnectDaemonRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DisconnectDaemonRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectDaemonRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectDaemonRequest copyWith(
          void Function(DisconnectDaemonRequest) updates) =>
      super.copyWith((message) => updates(message as DisconnectDaemonRequest))
          as DisconnectDaemonRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DisconnectDaemonRequest create() => DisconnectDaemonRequest._();
  @$core.override
  DisconnectDaemonRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DisconnectDaemonRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DisconnectDaemonRequest>(create);
  static DisconnectDaemonRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get generation => $_getI64(1);
  @$pb.TagNumber(2)
  set generation($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearGeneration() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get reason => $_getSZ(2);
  @$pb.TagNumber(3)
  set reason($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearReason() => $_clearField(3);
}

class DisconnectDaemonResponse extends $pb.GeneratedMessage {
  factory DisconnectDaemonResponse({
    RuntimeCommandResult? result,
  }) {
    final result$ = create();
    if (result != null) result$.result = result;
    return result$;
  }

  DisconnectDaemonResponse._();

  factory DisconnectDaemonResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DisconnectDaemonResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DisconnectDaemonResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aE<RuntimeCommandResult>(1, _omitFieldNames ? '' : 'result',
        enumValues: RuntimeCommandResult.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectDaemonResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectDaemonResponse copyWith(
          void Function(DisconnectDaemonResponse) updates) =>
      super.copyWith((message) => updates(message as DisconnectDaemonResponse))
          as DisconnectDaemonResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DisconnectDaemonResponse create() => DisconnectDaemonResponse._();
  @$core.override
  DisconnectDaemonResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DisconnectDaemonResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DisconnectDaemonResponse>(create);
  static DisconnectDaemonResponse? _defaultInstance;

  @$pb.TagNumber(1)
  RuntimeCommandResult get result => $_getN(0);
  @$pb.TagNumber(1)
  set result(RuntimeCommandResult value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasResult() => $_has(0);
  @$pb.TagNumber(1)
  void clearResult() => $_clearField(1);
}

class DisconnectSessionRequest extends $pb.GeneratedMessage {
  factory DisconnectSessionRequest({
    $core.String? sessionId,
    $fixnum.Int64? generation,
    $core.String? reason,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (generation != null) result.generation = generation;
    if (reason != null) result.reason = reason;
    return result;
  }

  DisconnectSessionRequest._();

  factory DisconnectSessionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DisconnectSessionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DisconnectSessionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'generation', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectSessionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectSessionRequest copyWith(
          void Function(DisconnectSessionRequest) updates) =>
      super.copyWith((message) => updates(message as DisconnectSessionRequest))
          as DisconnectSessionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DisconnectSessionRequest create() => DisconnectSessionRequest._();
  @$core.override
  DisconnectSessionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DisconnectSessionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DisconnectSessionRequest>(create);
  static DisconnectSessionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sessionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set sessionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get generation => $_getI64(1);
  @$pb.TagNumber(2)
  set generation($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGeneration() => $_has(1);
  @$pb.TagNumber(2)
  void clearGeneration() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get reason => $_getSZ(2);
  @$pb.TagNumber(3)
  set reason($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReason() => $_has(2);
  @$pb.TagNumber(3)
  void clearReason() => $_clearField(3);
}

class DisconnectSessionResponse extends $pb.GeneratedMessage {
  factory DisconnectSessionResponse({
    RuntimeCommandResult? result,
  }) {
    final result$ = create();
    if (result != null) result$.result = result;
    return result$;
  }

  DisconnectSessionResponse._();

  factory DisconnectSessionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DisconnectSessionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DisconnectSessionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aE<RuntimeCommandResult>(1, _omitFieldNames ? '' : 'result',
        enumValues: RuntimeCommandResult.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectSessionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectSessionResponse copyWith(
          void Function(DisconnectSessionResponse) updates) =>
      super.copyWith((message) => updates(message as DisconnectSessionResponse))
          as DisconnectSessionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DisconnectSessionResponse create() => DisconnectSessionResponse._();
  @$core.override
  DisconnectSessionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DisconnectSessionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DisconnectSessionResponse>(create);
  static DisconnectSessionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  RuntimeCommandResult get result => $_getN(0);
  @$pb.TagNumber(1)
  set result(RuntimeCommandResult value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasResult() => $_has(0);
  @$pb.TagNumber(1)
  void clearResult() => $_clearField(1);
}

/// OperatorRuntimeEvent 是 SSE 的失效提示，不携带实时对象完整内容。
class OperatorRuntimeEvent extends $pb.GeneratedMessage {
  factory OperatorRuntimeEvent({
    $core.String? controllerInstanceId,
    $fixnum.Int64? eventSeq,
    $core.String? resourceKind,
    $core.String? resourceId,
    OperatorEventOperation? operation,
    $0.Timestamp? occurredAt,
  }) {
    final result = create();
    if (controllerInstanceId != null)
      result.controllerInstanceId = controllerInstanceId;
    if (eventSeq != null) result.eventSeq = eventSeq;
    if (resourceKind != null) result.resourceKind = resourceKind;
    if (resourceId != null) result.resourceId = resourceId;
    if (operation != null) result.operation = operation;
    if (occurredAt != null) result.occurredAt = occurredAt;
    return result;
  }

  OperatorRuntimeEvent._();

  factory OperatorRuntimeEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OperatorRuntimeEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OperatorRuntimeEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'controllerInstanceId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'eventSeq', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'resourceKind')
    ..aOS(4, _omitFieldNames ? '' : 'resourceId')
    ..aE<OperatorEventOperation>(5, _omitFieldNames ? '' : 'operation',
        enumValues: OperatorEventOperation.values)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'occurredAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperatorRuntimeEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperatorRuntimeEvent copyWith(void Function(OperatorRuntimeEvent) updates) =>
      super.copyWith((message) => updates(message as OperatorRuntimeEvent))
          as OperatorRuntimeEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OperatorRuntimeEvent create() => OperatorRuntimeEvent._();
  @$core.override
  OperatorRuntimeEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OperatorRuntimeEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OperatorRuntimeEvent>(create);
  static OperatorRuntimeEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get controllerInstanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set controllerInstanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasControllerInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearControllerInstanceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get eventSeq => $_getI64(1);
  @$pb.TagNumber(2)
  set eventSeq($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEventSeq() => $_has(1);
  @$pb.TagNumber(2)
  void clearEventSeq() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get resourceKind => $_getSZ(2);
  @$pb.TagNumber(3)
  set resourceKind($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasResourceKind() => $_has(2);
  @$pb.TagNumber(3)
  void clearResourceKind() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get resourceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set resourceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasResourceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearResourceId() => $_clearField(4);

  @$pb.TagNumber(5)
  OperatorEventOperation get operation => $_getN(4);
  @$pb.TagNumber(5)
  set operation(OperatorEventOperation value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasOperation() => $_has(4);
  @$pb.TagNumber(5)
  void clearOperation() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get occurredAt => $_getN(5);
  @$pb.TagNumber(6)
  set occurredAt($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasOccurredAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearOccurredAt() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureOccurredAt() => $_ensure(5);
}

/// OperatorService 是中文运营后台的唯一查询、持久 mutation 与实时控制 API。
class OperatorServiceApi {
  final $pb.RpcClient _client;

  OperatorServiceApi(this._client);

  $async.Future<GetOperatorOverviewResponse> getOverview(
          $pb.ClientContext? ctx, GetOperatorOverviewRequest request) =>
      _client.invoke<GetOperatorOverviewResponse>(ctx, 'OperatorService',
          'GetOverview', request, GetOperatorOverviewResponse());
  $async.Future<ListOperatorAccountsResponse> listAccounts(
          $pb.ClientContext? ctx, ListOperatorAccountsRequest request) =>
      _client.invoke<ListOperatorAccountsResponse>(ctx, 'OperatorService',
          'ListAccounts', request, ListOperatorAccountsResponse());
  $async.Future<GetOperatorAccountResponse> getAccount(
          $pb.ClientContext? ctx, GetOperatorAccountRequest request) =>
      _client.invoke<GetOperatorAccountResponse>(ctx, 'OperatorService',
          'GetAccount', request, GetOperatorAccountResponse());
  $async.Future<ProvisionAccountResponse> provisionAccount(
          $pb.ClientContext? ctx, ProvisionAccountRequest request) =>
      _client.invoke<ProvisionAccountResponse>(ctx, 'OperatorService',
          'ProvisionAccount', request, ProvisionAccountResponse());
  $async.Future<ResetAccountSetupResponse> resetAccountSetup(
          $pb.ClientContext? ctx, ResetAccountSetupRequest request) =>
      _client.invoke<ResetAccountSetupResponse>(ctx, 'OperatorService',
          'ResetAccountSetup', request, ResetAccountSetupResponse());
  $async.Future<ListRuntimeSessionsResponse> listRuntimeSessions(
          $pb.ClientContext? ctx, ListRuntimeSessionsRequest request) =>
      _client.invoke<ListRuntimeSessionsResponse>(ctx, 'OperatorService',
          'ListRuntimeSessions', request, ListRuntimeSessionsResponse());
  $async.Future<ListOperatorOrdersResponse> listOrders(
          $pb.ClientContext? ctx, ListOperatorOrdersRequest request) =>
      _client.invoke<ListOperatorOrdersResponse>(ctx, 'OperatorService',
          'ListOrders', request, ListOperatorOrdersResponse());
  $async.Future<ListOperatorSubscriptionsResponse> listSubscriptions(
          $pb.ClientContext? ctx, ListOperatorSubscriptionsRequest request) =>
      _client.invoke<ListOperatorSubscriptionsResponse>(ctx, 'OperatorService',
          'ListSubscriptions', request, ListOperatorSubscriptionsResponse());
  $async.Future<ListOperatorUsageResponse> listUsage(
          $pb.ClientContext? ctx, ListOperatorUsageRequest request) =>
      _client.invoke<ListOperatorUsageResponse>(ctx, 'OperatorService',
          'ListUsage', request, ListOperatorUsageResponse());
  $async.Future<ListOperatorAuditResponse> listAudit(
          $pb.ClientContext? ctx, ListOperatorAuditRequest request) =>
      _client.invoke<ListOperatorAuditResponse>(ctx, 'OperatorService',
          'ListAudit', request, ListOperatorAuditResponse());
  $async.Future<SetAccountStateResponse> setAccountState(
          $pb.ClientContext? ctx, SetAccountStateRequest request) =>
      _client.invoke<SetAccountStateResponse>(ctx, 'OperatorService',
          'SetAccountState', request, SetAccountStateResponse());
  $async.Future<SetAccountRoleResponse> setAccountRole(
          $pb.ClientContext? ctx, SetAccountRoleRequest request) =>
      _client.invoke<SetAccountRoleResponse>(ctx, 'OperatorService',
          'SetAccountRole', request, SetAccountRoleResponse());
  $async.Future<DisconnectDaemonResponse> disconnectDaemon(
          $pb.ClientContext? ctx, DisconnectDaemonRequest request) =>
      _client.invoke<DisconnectDaemonResponse>(ctx, 'OperatorService',
          'DisconnectDaemon', request, DisconnectDaemonResponse());
  $async.Future<DisconnectSessionResponse> disconnectSession(
          $pb.ClientContext? ctx, DisconnectSessionRequest request) =>
      _client.invoke<DisconnectSessionResponse>(ctx, 'OperatorService',
          'DisconnectSession', request, DisconnectSessionResponse());
  $async.Future<$3.DeleteEdgeResponse> deleteEdge(
          $pb.ClientContext? ctx, $3.DeleteEdgeRequest request) =>
      _client.invoke<$3.DeleteEdgeResponse>(ctx, 'OperatorService',
          'DeleteEdge', request, $3.DeleteEdgeResponse());
  $async.Future<$3.CreateEdgeIdentityRecoveryResponse>
      createEdgeIdentityRecovery($pb.ClientContext? ctx,
              $3.CreateEdgeIdentityRecoveryRequest request) =>
          _client.invoke<$3.CreateEdgeIdentityRecoveryResponse>(
              ctx,
              'OperatorService',
              'CreateEdgeIdentityRecovery',
              request,
              $3.CreateEdgeIdentityRecoveryResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
