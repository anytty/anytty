// This is a generated file - do not edit.
//
// Generated from cloud/v1/commerce.proto.

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

import 'commerce.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'commerce.pbenum.dart';

/// Money 使用最小货币单位，避免浮点金额进入交易状态机。
class Money extends $pb.GeneratedMessage {
  factory Money({
    $core.String? currency,
    $fixnum.Int64? minorUnits,
  }) {
    final result = create();
    if (currency != null) result.currency = currency;
    if (minorUnits != null) result.minorUnits = minorUnits;
    return result;
  }

  Money._();

  factory Money.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Money.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Money',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'currency')
    ..aInt64(2, _omitFieldNames ? '' : 'minorUnits')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Money clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Money copyWith(void Function(Money) updates) =>
      super.copyWith((message) => updates(message as Money)) as Money;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Money create() => Money._();
  @$core.override
  Money createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Money getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Money>(create);
  static Money? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get currency => $_getSZ(0);
  @$pb.TagNumber(1)
  set currency($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCurrency() => $_has(0);
  @$pb.TagNumber(1)
  void clearCurrency() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get minorUnits => $_getI64(1);
  @$pb.TagNumber(2)
  set minorUnits($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMinorUnits() => $_has(1);
  @$pb.TagNumber(2)
  void clearMinorUnits() => $_clearField(2);
}

/// CloudCapability 是套餐和运行时票据共同消费的机器限制。
class CloudCapability extends $pb.GeneratedMessage {
  factory CloudCapability({
    $core.bool? managedP2pEnabled,
    $core.int? managedP2pMaxConcurrency,
    $core.bool? relayEnabled,
    $core.int? relayMaxConcurrency,
    $fixnum.Int64? relayMaxBytesPerPeriod,
    $fixnum.Int64? relayMaxBytesPerLease,
    $fixnum.Int64? relayMaxRateBytesPerSecond,
    $core.int? cloudDaemonLimit,
    $core.Iterable<$core.String>? allowedRegions,
  }) {
    final result = create();
    if (managedP2pEnabled != null) result.managedP2pEnabled = managedP2pEnabled;
    if (managedP2pMaxConcurrency != null)
      result.managedP2pMaxConcurrency = managedP2pMaxConcurrency;
    if (relayEnabled != null) result.relayEnabled = relayEnabled;
    if (relayMaxConcurrency != null)
      result.relayMaxConcurrency = relayMaxConcurrency;
    if (relayMaxBytesPerPeriod != null)
      result.relayMaxBytesPerPeriod = relayMaxBytesPerPeriod;
    if (relayMaxBytesPerLease != null)
      result.relayMaxBytesPerLease = relayMaxBytesPerLease;
    if (relayMaxRateBytesPerSecond != null)
      result.relayMaxRateBytesPerSecond = relayMaxRateBytesPerSecond;
    if (cloudDaemonLimit != null) result.cloudDaemonLimit = cloudDaemonLimit;
    if (allowedRegions != null) result.allowedRegions.addAll(allowedRegions);
    return result;
  }

  CloudCapability._();

  factory CloudCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudCapability',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'managedP2pEnabled')
    ..aI(2, _omitFieldNames ? '' : 'managedP2pMaxConcurrency',
        fieldType: $pb.PbFieldType.OU3)
    ..aOB(3, _omitFieldNames ? '' : 'relayEnabled')
    ..aI(4, _omitFieldNames ? '' : 'relayMaxConcurrency',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'relayMaxBytesPerPeriod', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'relayMaxBytesPerLease', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(7, _omitFieldNames ? '' : 'relayMaxRateBytesPerSecond',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(8, _omitFieldNames ? '' : 'cloudDaemonLimit',
        fieldType: $pb.PbFieldType.OU3)
    ..pPS(9, _omitFieldNames ? '' : 'allowedRegions')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudCapability copyWith(void Function(CloudCapability) updates) =>
      super.copyWith((message) => updates(message as CloudCapability))
          as CloudCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudCapability create() => CloudCapability._();
  @$core.override
  CloudCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudCapability>(create);
  static CloudCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get managedP2pEnabled => $_getBF(0);
  @$pb.TagNumber(1)
  set managedP2pEnabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasManagedP2pEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearManagedP2pEnabled() => $_clearField(1);

  /// 兼容旧套餐数据；Managed P2P 不执行并发数量限制。
  @$pb.TagNumber(2)
  $core.int get managedP2pMaxConcurrency => $_getIZ(1);
  @$pb.TagNumber(2)
  set managedP2pMaxConcurrency($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasManagedP2pMaxConcurrency() => $_has(1);
  @$pb.TagNumber(2)
  void clearManagedP2pMaxConcurrency() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get relayEnabled => $_getBF(2);
  @$pb.TagNumber(3)
  set relayEnabled($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRelayEnabled() => $_has(2);
  @$pb.TagNumber(3)
  void clearRelayEnabled() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get relayMaxConcurrency => $_getIZ(3);
  @$pb.TagNumber(4)
  set relayMaxConcurrency($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRelayMaxConcurrency() => $_has(3);
  @$pb.TagNumber(4)
  void clearRelayMaxConcurrency() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get relayMaxBytesPerPeriod => $_getI64(4);
  @$pb.TagNumber(5)
  set relayMaxBytesPerPeriod($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRelayMaxBytesPerPeriod() => $_has(4);
  @$pb.TagNumber(5)
  void clearRelayMaxBytesPerPeriod() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get relayMaxBytesPerLease => $_getI64(5);
  @$pb.TagNumber(6)
  set relayMaxBytesPerLease($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRelayMaxBytesPerLease() => $_has(5);
  @$pb.TagNumber(6)
  void clearRelayMaxBytesPerLease() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get relayMaxRateBytesPerSecond => $_getI64(6);
  @$pb.TagNumber(7)
  set relayMaxRateBytesPerSecond($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRelayMaxRateBytesPerSecond() => $_has(6);
  @$pb.TagNumber(7)
  void clearRelayMaxRateBytesPerSecond() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get cloudDaemonLimit => $_getIZ(7);
  @$pb.TagNumber(8)
  set cloudDaemonLimit($core.int value) => $_setUnsignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCloudDaemonLimit() => $_has(7);
  @$pb.TagNumber(8)
  void clearCloudDaemonLimit() => $_clearField(8);

  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get allowedRegions => $_getList(8);
}

/// PlanDefinition 是一个不可变套餐版本；已发布版本只能退休，不能原地改写。
class PlanDefinition extends $pb.GeneratedMessage {
  factory PlanDefinition({
    $core.String? planId,
    $fixnum.Int64? version,
    $core.String? name,
    $core.String? description,
    PlanState? state,
    $core.int? billingPeriodDays,
    Money? monthlyPrice,
    Money? yearlyPrice,
    CloudCapability? capability,
    $fixnum.Int64? revision,
    $0.Timestamp? createdAt,
    $0.Timestamp? publishedAt,
  }) {
    final result = create();
    if (planId != null) result.planId = planId;
    if (version != null) result.version = version;
    if (name != null) result.name = name;
    if (description != null) result.description = description;
    if (state != null) result.state = state;
    if (billingPeriodDays != null) result.billingPeriodDays = billingPeriodDays;
    if (monthlyPrice != null) result.monthlyPrice = monthlyPrice;
    if (yearlyPrice != null) result.yearlyPrice = yearlyPrice;
    if (capability != null) result.capability = capability;
    if (revision != null) result.revision = revision;
    if (createdAt != null) result.createdAt = createdAt;
    if (publishedAt != null) result.publishedAt = publishedAt;
    return result;
  }

  PlanDefinition._();

  factory PlanDefinition.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PlanDefinition.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PlanDefinition',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'name')
    ..aOS(4, _omitFieldNames ? '' : 'description')
    ..aE<PlanState>(5, _omitFieldNames ? '' : 'state',
        enumValues: PlanState.values)
    ..aI(6, _omitFieldNames ? '' : 'billingPeriodDays',
        fieldType: $pb.PbFieldType.OU3)
    ..aOM<Money>(7, _omitFieldNames ? '' : 'monthlyPrice',
        subBuilder: Money.create)
    ..aOM<Money>(8, _omitFieldNames ? '' : 'yearlyPrice',
        subBuilder: Money.create)
    ..aOM<CloudCapability>(9, _omitFieldNames ? '' : 'capability',
        subBuilder: CloudCapability.create)
    ..a<$fixnum.Int64>(
        10, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(11, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(12, _omitFieldNames ? '' : 'publishedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlanDefinition clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlanDefinition copyWith(void Function(PlanDefinition) updates) =>
      super.copyWith((message) => updates(message as PlanDefinition))
          as PlanDefinition;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PlanDefinition create() => PlanDefinition._();
  @$core.override
  PlanDefinition createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PlanDefinition getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PlanDefinition>(create);
  static PlanDefinition? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get planId => $_getSZ(0);
  @$pb.TagNumber(1)
  set planId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlanId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlanId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get version => $_getI64(1);
  @$pb.TagNumber(2)
  set version($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get name => $_getSZ(2);
  @$pb.TagNumber(3)
  set name($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasName() => $_has(2);
  @$pb.TagNumber(3)
  void clearName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get description => $_getSZ(3);
  @$pb.TagNumber(4)
  set description($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDescription() => $_has(3);
  @$pb.TagNumber(4)
  void clearDescription() => $_clearField(4);

  @$pb.TagNumber(5)
  PlanState get state => $_getN(4);
  @$pb.TagNumber(5)
  set state(PlanState value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasState() => $_has(4);
  @$pb.TagNumber(5)
  void clearState() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get billingPeriodDays => $_getIZ(5);
  @$pb.TagNumber(6)
  set billingPeriodDays($core.int value) => $_setUnsignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasBillingPeriodDays() => $_has(5);
  @$pb.TagNumber(6)
  void clearBillingPeriodDays() => $_clearField(6);

  @$pb.TagNumber(7)
  Money get monthlyPrice => $_getN(6);
  @$pb.TagNumber(7)
  set monthlyPrice(Money value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasMonthlyPrice() => $_has(6);
  @$pb.TagNumber(7)
  void clearMonthlyPrice() => $_clearField(7);
  @$pb.TagNumber(7)
  Money ensureMonthlyPrice() => $_ensure(6);

  @$pb.TagNumber(8)
  Money get yearlyPrice => $_getN(7);
  @$pb.TagNumber(8)
  set yearlyPrice(Money value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasYearlyPrice() => $_has(7);
  @$pb.TagNumber(8)
  void clearYearlyPrice() => $_clearField(8);
  @$pb.TagNumber(8)
  Money ensureYearlyPrice() => $_ensure(7);

  @$pb.TagNumber(9)
  CloudCapability get capability => $_getN(8);
  @$pb.TagNumber(9)
  set capability(CloudCapability value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasCapability() => $_has(8);
  @$pb.TagNumber(9)
  void clearCapability() => $_clearField(9);
  @$pb.TagNumber(9)
  CloudCapability ensureCapability() => $_ensure(8);

  @$pb.TagNumber(10)
  $fixnum.Int64 get revision => $_getI64(9);
  @$pb.TagNumber(10)
  set revision($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasRevision() => $_has(9);
  @$pb.TagNumber(10)
  void clearRevision() => $_clearField(10);

  @$pb.TagNumber(11)
  $0.Timestamp get createdAt => $_getN(10);
  @$pb.TagNumber(11)
  set createdAt($0.Timestamp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasCreatedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearCreatedAt() => $_clearField(11);
  @$pb.TagNumber(11)
  $0.Timestamp ensureCreatedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $0.Timestamp get publishedAt => $_getN(11);
  @$pb.TagNumber(12)
  set publishedAt($0.Timestamp value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasPublishedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearPublishedAt() => $_clearField(12);
  @$pb.TagNumber(12)
  $0.Timestamp ensurePublishedAt() => $_ensure(11);
}

class OrderProjection extends $pb.GeneratedMessage {
  factory OrderProjection({
    $core.String? orderId,
    $core.String? accountId,
    $core.String? planId,
    $fixnum.Int64? planVersion,
    OrderStatus? status,
    Money? amount,
    $core.String? provider,
    $core.String? providerReference,
    $core.String? idempotencyKey,
    SubscriptionTransition? requestedTransition,
    $fixnum.Int64? revision,
    $0.Timestamp? createdAt,
    $0.Timestamp? settledAt,
    $core.String? accountDisplayName,
    $core.String? accountEmail,
    $core.String? planName,
  }) {
    final result = create();
    if (orderId != null) result.orderId = orderId;
    if (accountId != null) result.accountId = accountId;
    if (planId != null) result.planId = planId;
    if (planVersion != null) result.planVersion = planVersion;
    if (status != null) result.status = status;
    if (amount != null) result.amount = amount;
    if (provider != null) result.provider = provider;
    if (providerReference != null) result.providerReference = providerReference;
    if (idempotencyKey != null) result.idempotencyKey = idempotencyKey;
    if (requestedTransition != null)
      result.requestedTransition = requestedTransition;
    if (revision != null) result.revision = revision;
    if (createdAt != null) result.createdAt = createdAt;
    if (settledAt != null) result.settledAt = settledAt;
    if (accountDisplayName != null)
      result.accountDisplayName = accountDisplayName;
    if (accountEmail != null) result.accountEmail = accountEmail;
    if (planName != null) result.planName = planName;
    return result;
  }

  OrderProjection._();

  factory OrderProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OrderProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OrderProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'orderId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'planVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<OrderStatus>(5, _omitFieldNames ? '' : 'status',
        enumValues: OrderStatus.values)
    ..aOM<Money>(6, _omitFieldNames ? '' : 'amount', subBuilder: Money.create)
    ..aOS(7, _omitFieldNames ? '' : 'provider')
    ..aOS(8, _omitFieldNames ? '' : 'providerReference')
    ..aOS(9, _omitFieldNames ? '' : 'idempotencyKey')
    ..aE<SubscriptionTransition>(
        10, _omitFieldNames ? '' : 'requestedTransition',
        enumValues: SubscriptionTransition.values)
    ..a<$fixnum.Int64>(
        11, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(12, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(13, _omitFieldNames ? '' : 'settledAt',
        subBuilder: $0.Timestamp.create)
    ..aOS(14, _omitFieldNames ? '' : 'accountDisplayName')
    ..aOS(15, _omitFieldNames ? '' : 'accountEmail')
    ..aOS(16, _omitFieldNames ? '' : 'planName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OrderProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OrderProjection copyWith(void Function(OrderProjection) updates) =>
      super.copyWith((message) => updates(message as OrderProjection))
          as OrderProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OrderProjection create() => OrderProjection._();
  @$core.override
  OrderProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OrderProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OrderProjection>(create);
  static OrderProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get orderId => $_getSZ(0);
  @$pb.TagNumber(1)
  set orderId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOrderId() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrderId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get planId => $_getSZ(2);
  @$pb.TagNumber(3)
  set planId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlanId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlanId() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get planVersion => $_getI64(3);
  @$pb.TagNumber(4)
  set planVersion($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlanVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlanVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  OrderStatus get status => $_getN(4);
  @$pb.TagNumber(5)
  set status(OrderStatus value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearStatus() => $_clearField(5);

  @$pb.TagNumber(6)
  Money get amount => $_getN(5);
  @$pb.TagNumber(6)
  set amount(Money value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasAmount() => $_has(5);
  @$pb.TagNumber(6)
  void clearAmount() => $_clearField(6);
  @$pb.TagNumber(6)
  Money ensureAmount() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.String get provider => $_getSZ(6);
  @$pb.TagNumber(7)
  set provider($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasProvider() => $_has(6);
  @$pb.TagNumber(7)
  void clearProvider() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get providerReference => $_getSZ(7);
  @$pb.TagNumber(8)
  set providerReference($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasProviderReference() => $_has(7);
  @$pb.TagNumber(8)
  void clearProviderReference() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get idempotencyKey => $_getSZ(8);
  @$pb.TagNumber(9)
  set idempotencyKey($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasIdempotencyKey() => $_has(8);
  @$pb.TagNumber(9)
  void clearIdempotencyKey() => $_clearField(9);

  @$pb.TagNumber(10)
  SubscriptionTransition get requestedTransition => $_getN(9);
  @$pb.TagNumber(10)
  set requestedTransition(SubscriptionTransition value) =>
      $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasRequestedTransition() => $_has(9);
  @$pb.TagNumber(10)
  void clearRequestedTransition() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get revision => $_getI64(10);
  @$pb.TagNumber(11)
  set revision($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasRevision() => $_has(10);
  @$pb.TagNumber(11)
  void clearRevision() => $_clearField(11);

  @$pb.TagNumber(12)
  $0.Timestamp get createdAt => $_getN(11);
  @$pb.TagNumber(12)
  set createdAt($0.Timestamp value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasCreatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearCreatedAt() => $_clearField(12);
  @$pb.TagNumber(12)
  $0.Timestamp ensureCreatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $0.Timestamp get settledAt => $_getN(12);
  @$pb.TagNumber(13)
  set settledAt($0.Timestamp value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasSettledAt() => $_has(12);
  @$pb.TagNumber(13)
  void clearSettledAt() => $_clearField(13);
  @$pb.TagNumber(13)
  $0.Timestamp ensureSettledAt() => $_ensure(12);

  @$pb.TagNumber(14)
  $core.String get accountDisplayName => $_getSZ(13);
  @$pb.TagNumber(14)
  set accountDisplayName($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasAccountDisplayName() => $_has(13);
  @$pb.TagNumber(14)
  void clearAccountDisplayName() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get accountEmail => $_getSZ(14);
  @$pb.TagNumber(15)
  set accountEmail($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasAccountEmail() => $_has(14);
  @$pb.TagNumber(15)
  void clearAccountEmail() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get planName => $_getSZ(15);
  @$pb.TagNumber(16)
  set planName($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasPlanName() => $_has(15);
  @$pb.TagNumber(16)
  void clearPlanName() => $_clearField(16);
}

class PaymentAttemptProjection extends $pb.GeneratedMessage {
  factory PaymentAttemptProjection({
    $core.String? paymentAttemptId,
    $core.String? orderId,
    $core.String? accountId,
    $core.String? provider,
    $core.String? providerReference,
    PaymentAttemptStatus? status,
    $fixnum.Int64? revision,
    $0.Timestamp? createdAt,
    $0.Timestamp? updatedAt,
  }) {
    final result = create();
    if (paymentAttemptId != null) result.paymentAttemptId = paymentAttemptId;
    if (orderId != null) result.orderId = orderId;
    if (accountId != null) result.accountId = accountId;
    if (provider != null) result.provider = provider;
    if (providerReference != null) result.providerReference = providerReference;
    if (status != null) result.status = status;
    if (revision != null) result.revision = revision;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    return result;
  }

  PaymentAttemptProjection._();

  factory PaymentAttemptProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PaymentAttemptProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PaymentAttemptProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'paymentAttemptId')
    ..aOS(2, _omitFieldNames ? '' : 'orderId')
    ..aOS(3, _omitFieldNames ? '' : 'accountId')
    ..aOS(4, _omitFieldNames ? '' : 'provider')
    ..aOS(5, _omitFieldNames ? '' : 'providerReference')
    ..aE<PaymentAttemptStatus>(6, _omitFieldNames ? '' : 'status',
        enumValues: PaymentAttemptStatus.values)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(8, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PaymentAttemptProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PaymentAttemptProjection copyWith(
          void Function(PaymentAttemptProjection) updates) =>
      super.copyWith((message) => updates(message as PaymentAttemptProjection))
          as PaymentAttemptProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PaymentAttemptProjection create() => PaymentAttemptProjection._();
  @$core.override
  PaymentAttemptProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PaymentAttemptProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PaymentAttemptProjection>(create);
  static PaymentAttemptProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get paymentAttemptId => $_getSZ(0);
  @$pb.TagNumber(1)
  set paymentAttemptId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPaymentAttemptId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPaymentAttemptId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get orderId => $_getSZ(1);
  @$pb.TagNumber(2)
  set orderId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOrderId() => $_has(1);
  @$pb.TagNumber(2)
  void clearOrderId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get accountId => $_getSZ(2);
  @$pb.TagNumber(3)
  set accountId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAccountId() => $_has(2);
  @$pb.TagNumber(3)
  void clearAccountId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get provider => $_getSZ(3);
  @$pb.TagNumber(4)
  set provider($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProvider() => $_has(3);
  @$pb.TagNumber(4)
  void clearProvider() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get providerReference => $_getSZ(4);
  @$pb.TagNumber(5)
  set providerReference($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProviderReference() => $_has(4);
  @$pb.TagNumber(5)
  void clearProviderReference() => $_clearField(5);

  @$pb.TagNumber(6)
  PaymentAttemptStatus get status => $_getN(5);
  @$pb.TagNumber(6)
  set status(PaymentAttemptStatus value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasStatus() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatus() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get revision => $_getI64(6);
  @$pb.TagNumber(7)
  set revision($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRevision() => $_has(6);
  @$pb.TagNumber(7)
  void clearRevision() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.Timestamp get createdAt => $_getN(7);
  @$pb.TagNumber(8)
  set createdAt($0.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasCreatedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearCreatedAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Timestamp ensureCreatedAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $0.Timestamp get updatedAt => $_getN(8);
  @$pb.TagNumber(9)
  set updatedAt($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasUpdatedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearUpdatedAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureUpdatedAt() => $_ensure(8);
}

class SubscriptionProjection extends $pb.GeneratedMessage {
  factory SubscriptionProjection({
    $core.String? subscriptionId,
    $core.String? accountId,
    $core.String? planId,
    $fixnum.Int64? planVersion,
    $core.String? sourceOrderId,
    SubscriptionState? state,
    $core.bool? cancelAtPeriodEnd,
    $fixnum.Int64? revision,
    $0.Timestamp? periodStart,
    $0.Timestamp? periodEnd,
    $0.Timestamp? updatedAt,
    $core.String? planName,
    $core.String? provider,
    $core.String? accountDisplayName,
    $core.String? accountEmail,
  }) {
    final result = create();
    if (subscriptionId != null) result.subscriptionId = subscriptionId;
    if (accountId != null) result.accountId = accountId;
    if (planId != null) result.planId = planId;
    if (planVersion != null) result.planVersion = planVersion;
    if (sourceOrderId != null) result.sourceOrderId = sourceOrderId;
    if (state != null) result.state = state;
    if (cancelAtPeriodEnd != null) result.cancelAtPeriodEnd = cancelAtPeriodEnd;
    if (revision != null) result.revision = revision;
    if (periodStart != null) result.periodStart = periodStart;
    if (periodEnd != null) result.periodEnd = periodEnd;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (planName != null) result.planName = planName;
    if (provider != null) result.provider = provider;
    if (accountDisplayName != null)
      result.accountDisplayName = accountDisplayName;
    if (accountEmail != null) result.accountEmail = accountEmail;
    return result;
  }

  SubscriptionProjection._();

  factory SubscriptionProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SubscriptionProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SubscriptionProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'subscriptionId')
    ..aOS(2, _omitFieldNames ? '' : 'accountId')
    ..aOS(3, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'planVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(5, _omitFieldNames ? '' : 'sourceOrderId')
    ..aE<SubscriptionState>(6, _omitFieldNames ? '' : 'state',
        enumValues: SubscriptionState.values)
    ..aOB(7, _omitFieldNames ? '' : 'cancelAtPeriodEnd')
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'periodStart',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'periodEnd',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(11, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $0.Timestamp.create)
    ..aOS(12, _omitFieldNames ? '' : 'planName')
    ..aOS(13, _omitFieldNames ? '' : 'provider')
    ..aOS(14, _omitFieldNames ? '' : 'accountDisplayName')
    ..aOS(15, _omitFieldNames ? '' : 'accountEmail')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SubscriptionProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SubscriptionProjection copyWith(
          void Function(SubscriptionProjection) updates) =>
      super.copyWith((message) => updates(message as SubscriptionProjection))
          as SubscriptionProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SubscriptionProjection create() => SubscriptionProjection._();
  @$core.override
  SubscriptionProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SubscriptionProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SubscriptionProjection>(create);
  static SubscriptionProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get subscriptionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set subscriptionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSubscriptionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSubscriptionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accountId => $_getSZ(1);
  @$pb.TagNumber(2)
  set accountId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccountId() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccountId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get planId => $_getSZ(2);
  @$pb.TagNumber(3)
  set planId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlanId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlanId() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get planVersion => $_getI64(3);
  @$pb.TagNumber(4)
  set planVersion($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlanVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlanVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get sourceOrderId => $_getSZ(4);
  @$pb.TagNumber(5)
  set sourceOrderId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSourceOrderId() => $_has(4);
  @$pb.TagNumber(5)
  void clearSourceOrderId() => $_clearField(5);

  @$pb.TagNumber(6)
  SubscriptionState get state => $_getN(5);
  @$pb.TagNumber(6)
  set state(SubscriptionState value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasState() => $_has(5);
  @$pb.TagNumber(6)
  void clearState() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get cancelAtPeriodEnd => $_getBF(6);
  @$pb.TagNumber(7)
  set cancelAtPeriodEnd($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCancelAtPeriodEnd() => $_has(6);
  @$pb.TagNumber(7)
  void clearCancelAtPeriodEnd() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get revision => $_getI64(7);
  @$pb.TagNumber(8)
  set revision($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRevision() => $_has(7);
  @$pb.TagNumber(8)
  void clearRevision() => $_clearField(8);

  @$pb.TagNumber(9)
  $0.Timestamp get periodStart => $_getN(8);
  @$pb.TagNumber(9)
  set periodStart($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasPeriodStart() => $_has(8);
  @$pb.TagNumber(9)
  void clearPeriodStart() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensurePeriodStart() => $_ensure(8);

  @$pb.TagNumber(10)
  $0.Timestamp get periodEnd => $_getN(9);
  @$pb.TagNumber(10)
  set periodEnd($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasPeriodEnd() => $_has(9);
  @$pb.TagNumber(10)
  void clearPeriodEnd() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensurePeriodEnd() => $_ensure(9);

  @$pb.TagNumber(11)
  $0.Timestamp get updatedAt => $_getN(10);
  @$pb.TagNumber(11)
  set updatedAt($0.Timestamp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasUpdatedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearUpdatedAt() => $_clearField(11);
  @$pb.TagNumber(11)
  $0.Timestamp ensureUpdatedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $core.String get planName => $_getSZ(11);
  @$pb.TagNumber(12)
  set planName($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasPlanName() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlanName() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get provider => $_getSZ(12);
  @$pb.TagNumber(13)
  set provider($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasProvider() => $_has(12);
  @$pb.TagNumber(13)
  void clearProvider() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get accountDisplayName => $_getSZ(13);
  @$pb.TagNumber(14)
  set accountDisplayName($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasAccountDisplayName() => $_has(13);
  @$pb.TagNumber(14)
  void clearAccountDisplayName() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get accountEmail => $_getSZ(14);
  @$pb.TagNumber(15)
  set accountEmail($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasAccountEmail() => $_has(14);
  @$pb.TagNumber(15)
  void clearAccountEmail() => $_clearField(15);
}

/// EffectiveEntitlement 是签发 Cloud 票据时冻结限制的唯一商业真值。
class EffectiveEntitlement extends $pb.GeneratedMessage {
  factory EffectiveEntitlement({
    $core.String? accountId,
    EntitlementState? state,
    $core.String? planId,
    $fixnum.Int64? planVersion,
    $core.String? subscriptionId,
    CloudCapability? capability,
    $fixnum.Int64? relayUsedBytes,
    $fixnum.Int64? relayRemainingBytes,
    $0.Timestamp? effectiveFrom,
    $0.Timestamp? effectiveUntil,
    $0.Timestamp? computedAt,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (state != null) result.state = state;
    if (planId != null) result.planId = planId;
    if (planVersion != null) result.planVersion = planVersion;
    if (subscriptionId != null) result.subscriptionId = subscriptionId;
    if (capability != null) result.capability = capability;
    if (relayUsedBytes != null) result.relayUsedBytes = relayUsedBytes;
    if (relayRemainingBytes != null)
      result.relayRemainingBytes = relayRemainingBytes;
    if (effectiveFrom != null) result.effectiveFrom = effectiveFrom;
    if (effectiveUntil != null) result.effectiveUntil = effectiveUntil;
    if (computedAt != null) result.computedAt = computedAt;
    return result;
  }

  EffectiveEntitlement._();

  factory EffectiveEntitlement.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EffectiveEntitlement.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EffectiveEntitlement',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aE<EntitlementState>(2, _omitFieldNames ? '' : 'state',
        enumValues: EntitlementState.values)
    ..aOS(3, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'planVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(5, _omitFieldNames ? '' : 'subscriptionId')
    ..aOM<CloudCapability>(6, _omitFieldNames ? '' : 'capability',
        subBuilder: CloudCapability.create)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'relayUsedBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'relayRemainingBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'effectiveFrom',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(10, _omitFieldNames ? '' : 'effectiveUntil',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(11, _omitFieldNames ? '' : 'computedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EffectiveEntitlement clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EffectiveEntitlement copyWith(void Function(EffectiveEntitlement) updates) =>
      super.copyWith((message) => updates(message as EffectiveEntitlement))
          as EffectiveEntitlement;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EffectiveEntitlement create() => EffectiveEntitlement._();
  @$core.override
  EffectiveEntitlement createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EffectiveEntitlement getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EffectiveEntitlement>(create);
  static EffectiveEntitlement? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  EntitlementState get state => $_getN(1);
  @$pb.TagNumber(2)
  set state(EntitlementState value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get planId => $_getSZ(2);
  @$pb.TagNumber(3)
  set planId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlanId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlanId() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get planVersion => $_getI64(3);
  @$pb.TagNumber(4)
  set planVersion($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlanVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlanVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get subscriptionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set subscriptionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSubscriptionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearSubscriptionId() => $_clearField(5);

  @$pb.TagNumber(6)
  CloudCapability get capability => $_getN(5);
  @$pb.TagNumber(6)
  set capability(CloudCapability value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCapability() => $_has(5);
  @$pb.TagNumber(6)
  void clearCapability() => $_clearField(6);
  @$pb.TagNumber(6)
  CloudCapability ensureCapability() => $_ensure(5);

  @$pb.TagNumber(7)
  $fixnum.Int64 get relayUsedBytes => $_getI64(6);
  @$pb.TagNumber(7)
  set relayUsedBytes($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasRelayUsedBytes() => $_has(6);
  @$pb.TagNumber(7)
  void clearRelayUsedBytes() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get relayRemainingBytes => $_getI64(7);
  @$pb.TagNumber(8)
  set relayRemainingBytes($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRelayRemainingBytes() => $_has(7);
  @$pb.TagNumber(8)
  void clearRelayRemainingBytes() => $_clearField(8);

  @$pb.TagNumber(9)
  $0.Timestamp get effectiveFrom => $_getN(8);
  @$pb.TagNumber(9)
  set effectiveFrom($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasEffectiveFrom() => $_has(8);
  @$pb.TagNumber(9)
  void clearEffectiveFrom() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureEffectiveFrom() => $_ensure(8);

  @$pb.TagNumber(10)
  $0.Timestamp get effectiveUntil => $_getN(9);
  @$pb.TagNumber(10)
  set effectiveUntil($0.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasEffectiveUntil() => $_has(9);
  @$pb.TagNumber(10)
  void clearEffectiveUntil() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.Timestamp ensureEffectiveUntil() => $_ensure(9);

  @$pb.TagNumber(11)
  $0.Timestamp get computedAt => $_getN(10);
  @$pb.TagNumber(11)
  set computedAt($0.Timestamp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasComputedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearComputedAt() => $_clearField(11);
  @$pb.TagNumber(11)
  $0.Timestamp ensureComputedAt() => $_ensure(10);
}

class UsagePeriodProjection extends $pb.GeneratedMessage {
  factory UsagePeriodProjection({
    $core.String? accountId,
    $0.Timestamp? periodStart,
    $0.Timestamp? periodEnd,
    $fixnum.Int64? relayIngressBytes,
    $fixnum.Int64? relayEgressBytes,
    $fixnum.Int64? relayTotalBytes,
    $fixnum.Int64? quotaBytes,
    $fixnum.Int64? remainingBytes,
    $fixnum.Int64? revision,
    $core.int? activeRelayReservations,
    $fixnum.Int64? relayHeldBytes,
    $core.String? accountDisplayName,
    $core.String? accountEmail,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (periodStart != null) result.periodStart = periodStart;
    if (periodEnd != null) result.periodEnd = periodEnd;
    if (relayIngressBytes != null) result.relayIngressBytes = relayIngressBytes;
    if (relayEgressBytes != null) result.relayEgressBytes = relayEgressBytes;
    if (relayTotalBytes != null) result.relayTotalBytes = relayTotalBytes;
    if (quotaBytes != null) result.quotaBytes = quotaBytes;
    if (remainingBytes != null) result.remainingBytes = remainingBytes;
    if (revision != null) result.revision = revision;
    if (activeRelayReservations != null)
      result.activeRelayReservations = activeRelayReservations;
    if (relayHeldBytes != null) result.relayHeldBytes = relayHeldBytes;
    if (accountDisplayName != null)
      result.accountDisplayName = accountDisplayName;
    if (accountEmail != null) result.accountEmail = accountEmail;
    return result;
  }

  UsagePeriodProjection._();

  factory UsagePeriodProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UsagePeriodProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UsagePeriodProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOM<$0.Timestamp>(2, _omitFieldNames ? '' : 'periodStart',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'periodEnd',
        subBuilder: $0.Timestamp.create)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'relayIngressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'relayEgressBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'relayTotalBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'quotaBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'remainingBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        9, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(10, _omitFieldNames ? '' : 'activeRelayReservations',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        11, _omitFieldNames ? '' : 'relayHeldBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(12, _omitFieldNames ? '' : 'accountDisplayName')
    ..aOS(13, _omitFieldNames ? '' : 'accountEmail')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UsagePeriodProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UsagePeriodProjection copyWith(
          void Function(UsagePeriodProjection) updates) =>
      super.copyWith((message) => updates(message as UsagePeriodProjection))
          as UsagePeriodProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UsagePeriodProjection create() => UsagePeriodProjection._();
  @$core.override
  UsagePeriodProjection createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UsagePeriodProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UsagePeriodProjection>(create);
  static UsagePeriodProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.Timestamp get periodStart => $_getN(1);
  @$pb.TagNumber(2)
  set periodStart($0.Timestamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPeriodStart() => $_has(1);
  @$pb.TagNumber(2)
  void clearPeriodStart() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.Timestamp ensurePeriodStart() => $_ensure(1);

  @$pb.TagNumber(3)
  $0.Timestamp get periodEnd => $_getN(2);
  @$pb.TagNumber(3)
  set periodEnd($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasPeriodEnd() => $_has(2);
  @$pb.TagNumber(3)
  void clearPeriodEnd() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensurePeriodEnd() => $_ensure(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get relayIngressBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set relayIngressBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRelayIngressBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearRelayIngressBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get relayEgressBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set relayEgressBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRelayEgressBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearRelayEgressBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get relayTotalBytes => $_getI64(5);
  @$pb.TagNumber(6)
  set relayTotalBytes($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRelayTotalBytes() => $_has(5);
  @$pb.TagNumber(6)
  void clearRelayTotalBytes() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get quotaBytes => $_getI64(6);
  @$pb.TagNumber(7)
  set quotaBytes($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasQuotaBytes() => $_has(6);
  @$pb.TagNumber(7)
  void clearQuotaBytes() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get remainingBytes => $_getI64(7);
  @$pb.TagNumber(8)
  set remainingBytes($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRemainingBytes() => $_has(7);
  @$pb.TagNumber(8)
  void clearRemainingBytes() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get revision => $_getI64(8);
  @$pb.TagNumber(9)
  set revision($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasRevision() => $_has(8);
  @$pb.TagNumber(9)
  void clearRevision() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get activeRelayReservations => $_getIZ(9);
  @$pb.TagNumber(10)
  set activeRelayReservations($core.int value) => $_setUnsignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasActiveRelayReservations() => $_has(9);
  @$pb.TagNumber(10)
  void clearActiveRelayReservations() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get relayHeldBytes => $_getI64(10);
  @$pb.TagNumber(11)
  set relayHeldBytes($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasRelayHeldBytes() => $_has(10);
  @$pb.TagNumber(11)
  void clearRelayHeldBytes() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get accountDisplayName => $_getSZ(11);
  @$pb.TagNumber(12)
  set accountDisplayName($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasAccountDisplayName() => $_has(11);
  @$pb.TagNumber(12)
  void clearAccountDisplayName() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get accountEmail => $_getSZ(12);
  @$pb.TagNumber(13)
  set accountEmail($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasAccountEmail() => $_has(12);
  @$pb.TagNumber(13)
  void clearAccountEmail() => $_clearField(13);
}

class ListPlansRequest extends $pb.GeneratedMessage {
  factory ListPlansRequest({
    $core.bool? includeUnpublished,
  }) {
    final result = create();
    if (includeUnpublished != null)
      result.includeUnpublished = includeUnpublished;
    return result;
  }

  ListPlansRequest._();

  factory ListPlansRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListPlansRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListPlansRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'includeUnpublished')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListPlansRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListPlansRequest copyWith(void Function(ListPlansRequest) updates) =>
      super.copyWith((message) => updates(message as ListPlansRequest))
          as ListPlansRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListPlansRequest create() => ListPlansRequest._();
  @$core.override
  ListPlansRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListPlansRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListPlansRequest>(create);
  static ListPlansRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get includeUnpublished => $_getBF(0);
  @$pb.TagNumber(1)
  set includeUnpublished($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIncludeUnpublished() => $_has(0);
  @$pb.TagNumber(1)
  void clearIncludeUnpublished() => $_clearField(1);
}

class ListPlansResponse extends $pb.GeneratedMessage {
  factory ListPlansResponse({
    $core.Iterable<PlanDefinition>? plans,
  }) {
    final result = create();
    if (plans != null) result.plans.addAll(plans);
    return result;
  }

  ListPlansResponse._();

  factory ListPlansResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListPlansResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListPlansResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..pPM<PlanDefinition>(1, _omitFieldNames ? '' : 'plans',
        subBuilder: PlanDefinition.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListPlansResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListPlansResponse copyWith(void Function(ListPlansResponse) updates) =>
      super.copyWith((message) => updates(message as ListPlansResponse))
          as ListPlansResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListPlansResponse create() => ListPlansResponse._();
  @$core.override
  ListPlansResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListPlansResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListPlansResponse>(create);
  static ListPlansResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<PlanDefinition> get plans => $_getList(0);
}

class CreatePlanVersionRequest extends $pb.GeneratedMessage {
  factory CreatePlanVersionRequest({
    $core.String? planId,
    $core.String? name,
    $core.String? description,
    $core.int? billingPeriodDays,
    Money? monthlyPrice,
    Money? yearlyPrice,
    CloudCapability? capability,
  }) {
    final result = create();
    if (planId != null) result.planId = planId;
    if (name != null) result.name = name;
    if (description != null) result.description = description;
    if (billingPeriodDays != null) result.billingPeriodDays = billingPeriodDays;
    if (monthlyPrice != null) result.monthlyPrice = monthlyPrice;
    if (yearlyPrice != null) result.yearlyPrice = yearlyPrice;
    if (capability != null) result.capability = capability;
    return result;
  }

  CreatePlanVersionRequest._();

  factory CreatePlanVersionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreatePlanVersionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreatePlanVersionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'planId')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'description')
    ..aI(4, _omitFieldNames ? '' : 'billingPeriodDays',
        fieldType: $pb.PbFieldType.OU3)
    ..aOM<Money>(5, _omitFieldNames ? '' : 'monthlyPrice',
        subBuilder: Money.create)
    ..aOM<Money>(6, _omitFieldNames ? '' : 'yearlyPrice',
        subBuilder: Money.create)
    ..aOM<CloudCapability>(7, _omitFieldNames ? '' : 'capability',
        subBuilder: CloudCapability.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreatePlanVersionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreatePlanVersionRequest copyWith(
          void Function(CreatePlanVersionRequest) updates) =>
      super.copyWith((message) => updates(message as CreatePlanVersionRequest))
          as CreatePlanVersionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreatePlanVersionRequest create() => CreatePlanVersionRequest._();
  @$core.override
  CreatePlanVersionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreatePlanVersionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreatePlanVersionRequest>(create);
  static CreatePlanVersionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get planId => $_getSZ(0);
  @$pb.TagNumber(1)
  set planId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlanId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlanId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get description => $_getSZ(2);
  @$pb.TagNumber(3)
  set description($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDescription() => $_has(2);
  @$pb.TagNumber(3)
  void clearDescription() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get billingPeriodDays => $_getIZ(3);
  @$pb.TagNumber(4)
  set billingPeriodDays($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBillingPeriodDays() => $_has(3);
  @$pb.TagNumber(4)
  void clearBillingPeriodDays() => $_clearField(4);

  @$pb.TagNumber(5)
  Money get monthlyPrice => $_getN(4);
  @$pb.TagNumber(5)
  set monthlyPrice(Money value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasMonthlyPrice() => $_has(4);
  @$pb.TagNumber(5)
  void clearMonthlyPrice() => $_clearField(5);
  @$pb.TagNumber(5)
  Money ensureMonthlyPrice() => $_ensure(4);

  @$pb.TagNumber(6)
  Money get yearlyPrice => $_getN(5);
  @$pb.TagNumber(6)
  set yearlyPrice(Money value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasYearlyPrice() => $_has(5);
  @$pb.TagNumber(6)
  void clearYearlyPrice() => $_clearField(6);
  @$pb.TagNumber(6)
  Money ensureYearlyPrice() => $_ensure(5);

  @$pb.TagNumber(7)
  CloudCapability get capability => $_getN(6);
  @$pb.TagNumber(7)
  set capability(CloudCapability value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasCapability() => $_has(6);
  @$pb.TagNumber(7)
  void clearCapability() => $_clearField(7);
  @$pb.TagNumber(7)
  CloudCapability ensureCapability() => $_ensure(6);
}

class CreatePlanVersionResponse extends $pb.GeneratedMessage {
  factory CreatePlanVersionResponse({
    PlanDefinition? plan,
  }) {
    final result = create();
    if (plan != null) result.plan = plan;
    return result;
  }

  CreatePlanVersionResponse._();

  factory CreatePlanVersionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreatePlanVersionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreatePlanVersionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PlanDefinition>(1, _omitFieldNames ? '' : 'plan',
        subBuilder: PlanDefinition.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreatePlanVersionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreatePlanVersionResponse copyWith(
          void Function(CreatePlanVersionResponse) updates) =>
      super.copyWith((message) => updates(message as CreatePlanVersionResponse))
          as CreatePlanVersionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreatePlanVersionResponse create() => CreatePlanVersionResponse._();
  @$core.override
  CreatePlanVersionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreatePlanVersionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreatePlanVersionResponse>(create);
  static CreatePlanVersionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  PlanDefinition get plan => $_getN(0);
  @$pb.TagNumber(1)
  set plan(PlanDefinition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPlan() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlan() => $_clearField(1);
  @$pb.TagNumber(1)
  PlanDefinition ensurePlan() => $_ensure(0);
}

class PublishPlanVersionRequest extends $pb.GeneratedMessage {
  factory PublishPlanVersionRequest({
    $core.String? planId,
    $fixnum.Int64? version,
    $fixnum.Int64? expectedRevision,
  }) {
    final result = create();
    if (planId != null) result.planId = planId;
    if (version != null) result.version = version;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    return result;
  }

  PublishPlanVersionRequest._();

  factory PublishPlanVersionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PublishPlanVersionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PublishPlanVersionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishPlanVersionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishPlanVersionRequest copyWith(
          void Function(PublishPlanVersionRequest) updates) =>
      super.copyWith((message) => updates(message as PublishPlanVersionRequest))
          as PublishPlanVersionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PublishPlanVersionRequest create() => PublishPlanVersionRequest._();
  @$core.override
  PublishPlanVersionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PublishPlanVersionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PublishPlanVersionRequest>(create);
  static PublishPlanVersionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get planId => $_getSZ(0);
  @$pb.TagNumber(1)
  set planId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlanId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlanId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get version => $_getI64(1);
  @$pb.TagNumber(2)
  set version($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get expectedRevision => $_getI64(2);
  @$pb.TagNumber(3)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExpectedRevision() => $_has(2);
  @$pb.TagNumber(3)
  void clearExpectedRevision() => $_clearField(3);
}

class PublishPlanVersionResponse extends $pb.GeneratedMessage {
  factory PublishPlanVersionResponse({
    PlanDefinition? plan,
  }) {
    final result = create();
    if (plan != null) result.plan = plan;
    return result;
  }

  PublishPlanVersionResponse._();

  factory PublishPlanVersionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PublishPlanVersionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PublishPlanVersionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<PlanDefinition>(1, _omitFieldNames ? '' : 'plan',
        subBuilder: PlanDefinition.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishPlanVersionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PublishPlanVersionResponse copyWith(
          void Function(PublishPlanVersionResponse) updates) =>
      super.copyWith(
              (message) => updates(message as PublishPlanVersionResponse))
          as PublishPlanVersionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PublishPlanVersionResponse create() => PublishPlanVersionResponse._();
  @$core.override
  PublishPlanVersionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PublishPlanVersionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PublishPlanVersionResponse>(create);
  static PublishPlanVersionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  PlanDefinition get plan => $_getN(0);
  @$pb.TagNumber(1)
  set plan(PlanDefinition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasPlan() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlan() => $_clearField(1);
  @$pb.TagNumber(1)
  PlanDefinition ensurePlan() => $_ensure(0);
}

class CreateOrderRequest extends $pb.GeneratedMessage {
  factory CreateOrderRequest({
    $core.String? accountId,
    $core.String? planId,
    $fixnum.Int64? planVersion,
    $core.String? provider,
    $core.String? idempotencyKey,
    SubscriptionTransition? requestedTransition,
    $core.bool? yearly,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (planId != null) result.planId = planId;
    if (planVersion != null) result.planVersion = planVersion;
    if (provider != null) result.provider = provider;
    if (idempotencyKey != null) result.idempotencyKey = idempotencyKey;
    if (requestedTransition != null)
      result.requestedTransition = requestedTransition;
    if (yearly != null) result.yearly = yearly;
    return result;
  }

  CreateOrderRequest._();

  factory CreateOrderRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateOrderRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateOrderRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOS(2, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'planVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(4, _omitFieldNames ? '' : 'provider')
    ..aOS(5, _omitFieldNames ? '' : 'idempotencyKey')
    ..aE<SubscriptionTransition>(
        6, _omitFieldNames ? '' : 'requestedTransition',
        enumValues: SubscriptionTransition.values)
    ..aOB(7, _omitFieldNames ? '' : 'yearly')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateOrderRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateOrderRequest copyWith(void Function(CreateOrderRequest) updates) =>
      super.copyWith((message) => updates(message as CreateOrderRequest))
          as CreateOrderRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateOrderRequest create() => CreateOrderRequest._();
  @$core.override
  CreateOrderRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateOrderRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateOrderRequest>(create);
  static CreateOrderRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get planId => $_getSZ(1);
  @$pb.TagNumber(2)
  set planId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPlanId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPlanId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get planVersion => $_getI64(2);
  @$pb.TagNumber(3)
  set planVersion($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlanVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlanVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get provider => $_getSZ(3);
  @$pb.TagNumber(4)
  set provider($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProvider() => $_has(3);
  @$pb.TagNumber(4)
  void clearProvider() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get idempotencyKey => $_getSZ(4);
  @$pb.TagNumber(5)
  set idempotencyKey($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasIdempotencyKey() => $_has(4);
  @$pb.TagNumber(5)
  void clearIdempotencyKey() => $_clearField(5);

  @$pb.TagNumber(6)
  SubscriptionTransition get requestedTransition => $_getN(5);
  @$pb.TagNumber(6)
  set requestedTransition(SubscriptionTransition value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasRequestedTransition() => $_has(5);
  @$pb.TagNumber(6)
  void clearRequestedTransition() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get yearly => $_getBF(6);
  @$pb.TagNumber(7)
  set yearly($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasYearly() => $_has(6);
  @$pb.TagNumber(7)
  void clearYearly() => $_clearField(7);
}

class CreateOrderResponse extends $pb.GeneratedMessage {
  factory CreateOrderResponse({
    OrderProjection? order,
    PaymentAttemptProjection? paymentAttempt,
  }) {
    final result = create();
    if (order != null) result.order = order;
    if (paymentAttempt != null) result.paymentAttempt = paymentAttempt;
    return result;
  }

  CreateOrderResponse._();

  factory CreateOrderResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateOrderResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateOrderResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<OrderProjection>(1, _omitFieldNames ? '' : 'order',
        subBuilder: OrderProjection.create)
    ..aOM<PaymentAttemptProjection>(2, _omitFieldNames ? '' : 'paymentAttempt',
        subBuilder: PaymentAttemptProjection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateOrderResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateOrderResponse copyWith(void Function(CreateOrderResponse) updates) =>
      super.copyWith((message) => updates(message as CreateOrderResponse))
          as CreateOrderResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateOrderResponse create() => CreateOrderResponse._();
  @$core.override
  CreateOrderResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateOrderResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateOrderResponse>(create);
  static CreateOrderResponse? _defaultInstance;

  @$pb.TagNumber(1)
  OrderProjection get order => $_getN(0);
  @$pb.TagNumber(1)
  set order(OrderProjection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasOrder() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrder() => $_clearField(1);
  @$pb.TagNumber(1)
  OrderProjection ensureOrder() => $_ensure(0);

  @$pb.TagNumber(2)
  PaymentAttemptProjection get paymentAttempt => $_getN(1);
  @$pb.TagNumber(2)
  set paymentAttempt(PaymentAttemptProjection value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPaymentAttempt() => $_has(1);
  @$pb.TagNumber(2)
  void clearPaymentAttempt() => $_clearField(2);
  @$pb.TagNumber(2)
  PaymentAttemptProjection ensurePaymentAttempt() => $_ensure(1);
}

class ApplyPaymentEventRequest extends $pb.GeneratedMessage {
  factory ApplyPaymentEventRequest({
    $core.String? provider,
    $core.String? providerEventId,
    $core.String? paymentAttemptId,
    $core.String? orderId,
    PaymentEventType? eventType,
    $core.String? providerReference,
    $0.Timestamp? occurredAt,
  }) {
    final result = create();
    if (provider != null) result.provider = provider;
    if (providerEventId != null) result.providerEventId = providerEventId;
    if (paymentAttemptId != null) result.paymentAttemptId = paymentAttemptId;
    if (orderId != null) result.orderId = orderId;
    if (eventType != null) result.eventType = eventType;
    if (providerReference != null) result.providerReference = providerReference;
    if (occurredAt != null) result.occurredAt = occurredAt;
    return result;
  }

  ApplyPaymentEventRequest._();

  factory ApplyPaymentEventRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ApplyPaymentEventRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ApplyPaymentEventRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'provider')
    ..aOS(2, _omitFieldNames ? '' : 'providerEventId')
    ..aOS(3, _omitFieldNames ? '' : 'paymentAttemptId')
    ..aOS(4, _omitFieldNames ? '' : 'orderId')
    ..aE<PaymentEventType>(5, _omitFieldNames ? '' : 'eventType',
        enumValues: PaymentEventType.values)
    ..aOS(6, _omitFieldNames ? '' : 'providerReference')
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'occurredAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApplyPaymentEventRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApplyPaymentEventRequest copyWith(
          void Function(ApplyPaymentEventRequest) updates) =>
      super.copyWith((message) => updates(message as ApplyPaymentEventRequest))
          as ApplyPaymentEventRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ApplyPaymentEventRequest create() => ApplyPaymentEventRequest._();
  @$core.override
  ApplyPaymentEventRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ApplyPaymentEventRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ApplyPaymentEventRequest>(create);
  static ApplyPaymentEventRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get provider => $_getSZ(0);
  @$pb.TagNumber(1)
  set provider($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProvider() => $_has(0);
  @$pb.TagNumber(1)
  void clearProvider() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get providerEventId => $_getSZ(1);
  @$pb.TagNumber(2)
  set providerEventId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasProviderEventId() => $_has(1);
  @$pb.TagNumber(2)
  void clearProviderEventId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get paymentAttemptId => $_getSZ(2);
  @$pb.TagNumber(3)
  set paymentAttemptId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPaymentAttemptId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPaymentAttemptId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get orderId => $_getSZ(3);
  @$pb.TagNumber(4)
  set orderId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOrderId() => $_has(3);
  @$pb.TagNumber(4)
  void clearOrderId() => $_clearField(4);

  @$pb.TagNumber(5)
  PaymentEventType get eventType => $_getN(4);
  @$pb.TagNumber(5)
  set eventType(PaymentEventType value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasEventType() => $_has(4);
  @$pb.TagNumber(5)
  void clearEventType() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get providerReference => $_getSZ(5);
  @$pb.TagNumber(6)
  set providerReference($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasProviderReference() => $_has(5);
  @$pb.TagNumber(6)
  void clearProviderReference() => $_clearField(6);

  @$pb.TagNumber(7)
  $0.Timestamp get occurredAt => $_getN(6);
  @$pb.TagNumber(7)
  set occurredAt($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasOccurredAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearOccurredAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensureOccurredAt() => $_ensure(6);
}

class ApplyPaymentEventResponse extends $pb.GeneratedMessage {
  factory ApplyPaymentEventResponse({
    OrderProjection? order,
    PaymentAttemptProjection? paymentAttempt,
    SubscriptionProjection? subscription,
    EffectiveEntitlement? entitlement,
    $core.bool? duplicate,
  }) {
    final result = create();
    if (order != null) result.order = order;
    if (paymentAttempt != null) result.paymentAttempt = paymentAttempt;
    if (subscription != null) result.subscription = subscription;
    if (entitlement != null) result.entitlement = entitlement;
    if (duplicate != null) result.duplicate = duplicate;
    return result;
  }

  ApplyPaymentEventResponse._();

  factory ApplyPaymentEventResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ApplyPaymentEventResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ApplyPaymentEventResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<OrderProjection>(1, _omitFieldNames ? '' : 'order',
        subBuilder: OrderProjection.create)
    ..aOM<PaymentAttemptProjection>(2, _omitFieldNames ? '' : 'paymentAttempt',
        subBuilder: PaymentAttemptProjection.create)
    ..aOM<SubscriptionProjection>(3, _omitFieldNames ? '' : 'subscription',
        subBuilder: SubscriptionProjection.create)
    ..aOM<EffectiveEntitlement>(4, _omitFieldNames ? '' : 'entitlement',
        subBuilder: EffectiveEntitlement.create)
    ..aOB(5, _omitFieldNames ? '' : 'duplicate')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApplyPaymentEventResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ApplyPaymentEventResponse copyWith(
          void Function(ApplyPaymentEventResponse) updates) =>
      super.copyWith((message) => updates(message as ApplyPaymentEventResponse))
          as ApplyPaymentEventResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ApplyPaymentEventResponse create() => ApplyPaymentEventResponse._();
  @$core.override
  ApplyPaymentEventResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ApplyPaymentEventResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ApplyPaymentEventResponse>(create);
  static ApplyPaymentEventResponse? _defaultInstance;

  @$pb.TagNumber(1)
  OrderProjection get order => $_getN(0);
  @$pb.TagNumber(1)
  set order(OrderProjection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasOrder() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrder() => $_clearField(1);
  @$pb.TagNumber(1)
  OrderProjection ensureOrder() => $_ensure(0);

  @$pb.TagNumber(2)
  PaymentAttemptProjection get paymentAttempt => $_getN(1);
  @$pb.TagNumber(2)
  set paymentAttempt(PaymentAttemptProjection value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPaymentAttempt() => $_has(1);
  @$pb.TagNumber(2)
  void clearPaymentAttempt() => $_clearField(2);
  @$pb.TagNumber(2)
  PaymentAttemptProjection ensurePaymentAttempt() => $_ensure(1);

  @$pb.TagNumber(3)
  SubscriptionProjection get subscription => $_getN(2);
  @$pb.TagNumber(3)
  set subscription(SubscriptionProjection value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSubscription() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubscription() => $_clearField(3);
  @$pb.TagNumber(3)
  SubscriptionProjection ensureSubscription() => $_ensure(2);

  @$pb.TagNumber(4)
  EffectiveEntitlement get entitlement => $_getN(3);
  @$pb.TagNumber(4)
  set entitlement(EffectiveEntitlement value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasEntitlement() => $_has(3);
  @$pb.TagNumber(4)
  void clearEntitlement() => $_clearField(4);
  @$pb.TagNumber(4)
  EffectiveEntitlement ensureEntitlement() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.bool get duplicate => $_getBF(4);
  @$pb.TagNumber(5)
  set duplicate($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDuplicate() => $_has(4);
  @$pb.TagNumber(5)
  void clearDuplicate() => $_clearField(5);
}

class TransitionSubscriptionRequest extends $pb.GeneratedMessage {
  factory TransitionSubscriptionRequest({
    $core.String? accountId,
    SubscriptionTransition? transition,
    $core.String? targetPlanId,
    $fixnum.Int64? targetPlanVersion,
    $fixnum.Int64? expectedRevision,
    $core.String? reason,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    if (transition != null) result.transition = transition;
    if (targetPlanId != null) result.targetPlanId = targetPlanId;
    if (targetPlanVersion != null) result.targetPlanVersion = targetPlanVersion;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (reason != null) result.reason = reason;
    return result;
  }

  TransitionSubscriptionRequest._();

  factory TransitionSubscriptionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransitionSubscriptionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransitionSubscriptionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aE<SubscriptionTransition>(2, _omitFieldNames ? '' : 'transition',
        enumValues: SubscriptionTransition.values)
    ..aOS(3, _omitFieldNames ? '' : 'targetPlanId')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'targetPlanVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransitionSubscriptionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransitionSubscriptionRequest copyWith(
          void Function(TransitionSubscriptionRequest) updates) =>
      super.copyWith(
              (message) => updates(message as TransitionSubscriptionRequest))
          as TransitionSubscriptionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransitionSubscriptionRequest create() =>
      TransitionSubscriptionRequest._();
  @$core.override
  TransitionSubscriptionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransitionSubscriptionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransitionSubscriptionRequest>(create);
  static TransitionSubscriptionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  SubscriptionTransition get transition => $_getN(1);
  @$pb.TagNumber(2)
  set transition(SubscriptionTransition value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTransition() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransition() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get targetPlanId => $_getSZ(2);
  @$pb.TagNumber(3)
  set targetPlanId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTargetPlanId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTargetPlanId() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get targetPlanVersion => $_getI64(3);
  @$pb.TagNumber(4)
  set targetPlanVersion($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTargetPlanVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearTargetPlanVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get expectedRevision => $_getI64(4);
  @$pb.TagNumber(5)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasExpectedRevision() => $_has(4);
  @$pb.TagNumber(5)
  void clearExpectedRevision() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get reason => $_getSZ(5);
  @$pb.TagNumber(6)
  set reason($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasReason() => $_has(5);
  @$pb.TagNumber(6)
  void clearReason() => $_clearField(6);
}

class TransitionSubscriptionResponse extends $pb.GeneratedMessage {
  factory TransitionSubscriptionResponse({
    SubscriptionProjection? subscription,
    EffectiveEntitlement? entitlement,
  }) {
    final result = create();
    if (subscription != null) result.subscription = subscription;
    if (entitlement != null) result.entitlement = entitlement;
    return result;
  }

  TransitionSubscriptionResponse._();

  factory TransitionSubscriptionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransitionSubscriptionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransitionSubscriptionResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<SubscriptionProjection>(1, _omitFieldNames ? '' : 'subscription',
        subBuilder: SubscriptionProjection.create)
    ..aOM<EffectiveEntitlement>(2, _omitFieldNames ? '' : 'entitlement',
        subBuilder: EffectiveEntitlement.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransitionSubscriptionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransitionSubscriptionResponse copyWith(
          void Function(TransitionSubscriptionResponse) updates) =>
      super.copyWith(
              (message) => updates(message as TransitionSubscriptionResponse))
          as TransitionSubscriptionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransitionSubscriptionResponse create() =>
      TransitionSubscriptionResponse._();
  @$core.override
  TransitionSubscriptionResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransitionSubscriptionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransitionSubscriptionResponse>(create);
  static TransitionSubscriptionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  SubscriptionProjection get subscription => $_getN(0);
  @$pb.TagNumber(1)
  set subscription(SubscriptionProjection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSubscription() => $_has(0);
  @$pb.TagNumber(1)
  void clearSubscription() => $_clearField(1);
  @$pb.TagNumber(1)
  SubscriptionProjection ensureSubscription() => $_ensure(0);

  @$pb.TagNumber(2)
  EffectiveEntitlement get entitlement => $_getN(1);
  @$pb.TagNumber(2)
  set entitlement(EffectiveEntitlement value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasEntitlement() => $_has(1);
  @$pb.TagNumber(2)
  void clearEntitlement() => $_clearField(2);
  @$pb.TagNumber(2)
  EffectiveEntitlement ensureEntitlement() => $_ensure(1);
}

class GetAccountCommerceRequest extends $pb.GeneratedMessage {
  factory GetAccountCommerceRequest({
    $core.String? accountId,
  }) {
    final result = create();
    if (accountId != null) result.accountId = accountId;
    return result;
  }

  GetAccountCommerceRequest._();

  factory GetAccountCommerceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetAccountCommerceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetAccountCommerceRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAccountCommerceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAccountCommerceRequest copyWith(
          void Function(GetAccountCommerceRequest) updates) =>
      super.copyWith((message) => updates(message as GetAccountCommerceRequest))
          as GetAccountCommerceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetAccountCommerceRequest create() => GetAccountCommerceRequest._();
  @$core.override
  GetAccountCommerceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetAccountCommerceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetAccountCommerceRequest>(create);
  static GetAccountCommerceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);
}

class GetAccountCommerceResponse extends $pb.GeneratedMessage {
  factory GetAccountCommerceResponse({
    SubscriptionProjection? subscription,
    EffectiveEntitlement? entitlement,
    $core.Iterable<OrderProjection>? orders,
    $core.Iterable<PaymentAttemptProjection>? paymentAttempts,
    UsagePeriodProjection? usage,
  }) {
    final result = create();
    if (subscription != null) result.subscription = subscription;
    if (entitlement != null) result.entitlement = entitlement;
    if (orders != null) result.orders.addAll(orders);
    if (paymentAttempts != null) result.paymentAttempts.addAll(paymentAttempts);
    if (usage != null) result.usage = usage;
    return result;
  }

  GetAccountCommerceResponse._();

  factory GetAccountCommerceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetAccountCommerceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetAccountCommerceResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOM<SubscriptionProjection>(1, _omitFieldNames ? '' : 'subscription',
        subBuilder: SubscriptionProjection.create)
    ..aOM<EffectiveEntitlement>(2, _omitFieldNames ? '' : 'entitlement',
        subBuilder: EffectiveEntitlement.create)
    ..pPM<OrderProjection>(3, _omitFieldNames ? '' : 'orders',
        subBuilder: OrderProjection.create)
    ..pPM<PaymentAttemptProjection>(4, _omitFieldNames ? '' : 'paymentAttempts',
        subBuilder: PaymentAttemptProjection.create)
    ..aOM<UsagePeriodProjection>(5, _omitFieldNames ? '' : 'usage',
        subBuilder: UsagePeriodProjection.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAccountCommerceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAccountCommerceResponse copyWith(
          void Function(GetAccountCommerceResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetAccountCommerceResponse))
          as GetAccountCommerceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetAccountCommerceResponse create() => GetAccountCommerceResponse._();
  @$core.override
  GetAccountCommerceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetAccountCommerceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetAccountCommerceResponse>(create);
  static GetAccountCommerceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  SubscriptionProjection get subscription => $_getN(0);
  @$pb.TagNumber(1)
  set subscription(SubscriptionProjection value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSubscription() => $_has(0);
  @$pb.TagNumber(1)
  void clearSubscription() => $_clearField(1);
  @$pb.TagNumber(1)
  SubscriptionProjection ensureSubscription() => $_ensure(0);

  @$pb.TagNumber(2)
  EffectiveEntitlement get entitlement => $_getN(1);
  @$pb.TagNumber(2)
  set entitlement(EffectiveEntitlement value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasEntitlement() => $_has(1);
  @$pb.TagNumber(2)
  void clearEntitlement() => $_clearField(2);
  @$pb.TagNumber(2)
  EffectiveEntitlement ensureEntitlement() => $_ensure(1);

  @$pb.TagNumber(3)
  $pb.PbList<OrderProjection> get orders => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbList<PaymentAttemptProjection> get paymentAttempts => $_getList(3);

  @$pb.TagNumber(5)
  UsagePeriodProjection get usage => $_getN(4);
  @$pb.TagNumber(5)
  set usage(UsagePeriodProjection value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasUsage() => $_has(4);
  @$pb.TagNumber(5)
  void clearUsage() => $_clearField(5);
  @$pb.TagNumber(5)
  UsagePeriodProjection ensureUsage() => $_ensure(4);
}

/// CreateMyOrderRequest 不接受 account_id 或 provider；两者由认证 session 和部署模式决定。
class CreateMyOrderRequest extends $pb.GeneratedMessage {
  factory CreateMyOrderRequest({
    $core.String? planId,
    $fixnum.Int64? planVersion,
    $core.String? idempotencyKey,
    SubscriptionTransition? requestedTransition,
    $core.bool? yearly,
  }) {
    final result = create();
    if (planId != null) result.planId = planId;
    if (planVersion != null) result.planVersion = planVersion;
    if (idempotencyKey != null) result.idempotencyKey = idempotencyKey;
    if (requestedTransition != null)
      result.requestedTransition = requestedTransition;
    if (yearly != null) result.yearly = yearly;
    return result;
  }

  CreateMyOrderRequest._();

  factory CreateMyOrderRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateMyOrderRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateMyOrderRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'planId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'planVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'idempotencyKey')
    ..aE<SubscriptionTransition>(
        4, _omitFieldNames ? '' : 'requestedTransition',
        enumValues: SubscriptionTransition.values)
    ..aOB(5, _omitFieldNames ? '' : 'yearly')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateMyOrderRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateMyOrderRequest copyWith(void Function(CreateMyOrderRequest) updates) =>
      super.copyWith((message) => updates(message as CreateMyOrderRequest))
          as CreateMyOrderRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateMyOrderRequest create() => CreateMyOrderRequest._();
  @$core.override
  CreateMyOrderRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateMyOrderRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateMyOrderRequest>(create);
  static CreateMyOrderRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get planId => $_getSZ(0);
  @$pb.TagNumber(1)
  set planId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlanId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlanId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get planVersion => $_getI64(1);
  @$pb.TagNumber(2)
  set planVersion($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPlanVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearPlanVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get idempotencyKey => $_getSZ(2);
  @$pb.TagNumber(3)
  set idempotencyKey($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIdempotencyKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdempotencyKey() => $_clearField(3);

  @$pb.TagNumber(4)
  SubscriptionTransition get requestedTransition => $_getN(3);
  @$pb.TagNumber(4)
  set requestedTransition(SubscriptionTransition value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRequestedTransition() => $_has(3);
  @$pb.TagNumber(4)
  void clearRequestedTransition() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get yearly => $_getBF(4);
  @$pb.TagNumber(5)
  set yearly($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasYearly() => $_has(4);
  @$pb.TagNumber(5)
  void clearYearly() => $_clearField(5);
}

class GetMyCommerceRequest extends $pb.GeneratedMessage {
  factory GetMyCommerceRequest() => create();

  GetMyCommerceRequest._();

  factory GetMyCommerceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetMyCommerceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetMyCommerceRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMyCommerceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMyCommerceRequest copyWith(void Function(GetMyCommerceRequest) updates) =>
      super.copyWith((message) => updates(message as GetMyCommerceRequest))
          as GetMyCommerceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetMyCommerceRequest create() => GetMyCommerceRequest._();
  @$core.override
  GetMyCommerceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetMyCommerceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetMyCommerceRequest>(create);
  static GetMyCommerceRequest? _defaultInstance;
}

/// ChangeMySubscriptionRequest 只承载用户可执行的取消到期和恢复动作。
class ChangeMySubscriptionRequest extends $pb.GeneratedMessage {
  factory ChangeMySubscriptionRequest({
    SubscriptionTransition? transition,
    $fixnum.Int64? expectedRevision,
  }) {
    final result = create();
    if (transition != null) result.transition = transition;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    return result;
  }

  ChangeMySubscriptionRequest._();

  factory ChangeMySubscriptionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ChangeMySubscriptionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeMySubscriptionRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aE<SubscriptionTransition>(1, _omitFieldNames ? '' : 'transition',
        enumValues: SubscriptionTransition.values)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMySubscriptionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeMySubscriptionRequest copyWith(
          void Function(ChangeMySubscriptionRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ChangeMySubscriptionRequest))
          as ChangeMySubscriptionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ChangeMySubscriptionRequest create() =>
      ChangeMySubscriptionRequest._();
  @$core.override
  ChangeMySubscriptionRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ChangeMySubscriptionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ChangeMySubscriptionRequest>(create);
  static ChangeMySubscriptionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  SubscriptionTransition get transition => $_getN(0);
  @$pb.TagNumber(1)
  set transition(SubscriptionTransition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTransition() => $_has(0);
  @$pb.TagNumber(1)
  void clearTransition() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get expectedRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExpectedRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpectedRevision() => $_clearField(2);
}

/// CompleteDevelopmentPaymentRequest 仅在显式 Development 支付模式下可用。
class CompleteDevelopmentPaymentRequest extends $pb.GeneratedMessage {
  factory CompleteDevelopmentPaymentRequest({
    $core.String? orderId,
    $core.String? paymentAttemptId,
  }) {
    final result = create();
    if (orderId != null) result.orderId = orderId;
    if (paymentAttemptId != null) result.paymentAttemptId = paymentAttemptId;
    return result;
  }

  CompleteDevelopmentPaymentRequest._();

  factory CompleteDevelopmentPaymentRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CompleteDevelopmentPaymentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CompleteDevelopmentPaymentRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'orderId')
    ..aOS(2, _omitFieldNames ? '' : 'paymentAttemptId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDevelopmentPaymentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CompleteDevelopmentPaymentRequest copyWith(
          void Function(CompleteDevelopmentPaymentRequest) updates) =>
      super.copyWith((message) =>
              updates(message as CompleteDevelopmentPaymentRequest))
          as CompleteDevelopmentPaymentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CompleteDevelopmentPaymentRequest create() =>
      CompleteDevelopmentPaymentRequest._();
  @$core.override
  CompleteDevelopmentPaymentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CompleteDevelopmentPaymentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CompleteDevelopmentPaymentRequest>(
          create);
  static CompleteDevelopmentPaymentRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get orderId => $_getSZ(0);
  @$pb.TagNumber(1)
  set orderId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOrderId() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrderId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get paymentAttemptId => $_getSZ(1);
  @$pb.TagNumber(2)
  set paymentAttemptId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPaymentAttemptId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPaymentAttemptId() => $_clearField(2);
}

/// CommerceService 拥有套餐、订单、支付、订阅和 Entitlement 状态机。
class CommerceServiceApi {
  final $pb.RpcClient _client;

  CommerceServiceApi(this._client);

  $async.Future<ListPlansResponse> listPlans(
          $pb.ClientContext? ctx, ListPlansRequest request) =>
      _client.invoke<ListPlansResponse>(
          ctx, 'CommerceService', 'ListPlans', request, ListPlansResponse());
  $async.Future<CreatePlanVersionResponse> createPlanVersion(
          $pb.ClientContext? ctx, CreatePlanVersionRequest request) =>
      _client.invoke<CreatePlanVersionResponse>(ctx, 'CommerceService',
          'CreatePlanVersion', request, CreatePlanVersionResponse());
  $async.Future<PublishPlanVersionResponse> publishPlanVersion(
          $pb.ClientContext? ctx, PublishPlanVersionRequest request) =>
      _client.invoke<PublishPlanVersionResponse>(ctx, 'CommerceService',
          'PublishPlanVersion', request, PublishPlanVersionResponse());
  $async.Future<CreateOrderResponse> createOrder(
          $pb.ClientContext? ctx, CreateOrderRequest request) =>
      _client.invoke<CreateOrderResponse>(ctx, 'CommerceService', 'CreateOrder',
          request, CreateOrderResponse());
  $async.Future<ApplyPaymentEventResponse> applyPaymentEvent(
          $pb.ClientContext? ctx, ApplyPaymentEventRequest request) =>
      _client.invoke<ApplyPaymentEventResponse>(ctx, 'CommerceService',
          'ApplyPaymentEvent', request, ApplyPaymentEventResponse());
  $async.Future<TransitionSubscriptionResponse> transitionSubscription(
          $pb.ClientContext? ctx, TransitionSubscriptionRequest request) =>
      _client.invoke<TransitionSubscriptionResponse>(ctx, 'CommerceService',
          'TransitionSubscription', request, TransitionSubscriptionResponse());
  $async.Future<GetAccountCommerceResponse> getAccountCommerce(
          $pb.ClientContext? ctx, GetAccountCommerceRequest request) =>
      _client.invoke<GetAccountCommerceResponse>(ctx, 'CommerceService',
          'GetAccountCommerce', request, GetAccountCommerceResponse());
  $async.Future<CreateOrderResponse> createMyOrder(
          $pb.ClientContext? ctx, CreateMyOrderRequest request) =>
      _client.invoke<CreateOrderResponse>(ctx, 'CommerceService',
          'CreateMyOrder', request, CreateOrderResponse());
  $async.Future<GetAccountCommerceResponse> getMyCommerce(
          $pb.ClientContext? ctx, GetMyCommerceRequest request) =>
      _client.invoke<GetAccountCommerceResponse>(ctx, 'CommerceService',
          'GetMyCommerce', request, GetAccountCommerceResponse());
  $async.Future<TransitionSubscriptionResponse> changeMySubscription(
          $pb.ClientContext? ctx, ChangeMySubscriptionRequest request) =>
      _client.invoke<TransitionSubscriptionResponse>(ctx, 'CommerceService',
          'ChangeMySubscription', request, TransitionSubscriptionResponse());
  $async.Future<ApplyPaymentEventResponse> completeDevelopmentPayment(
          $pb.ClientContext? ctx, CompleteDevelopmentPaymentRequest request) =>
      _client.invoke<ApplyPaymentEventResponse>(ctx, 'CommerceService',
          'CompleteDevelopmentPayment', request, ApplyPaymentEventResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
