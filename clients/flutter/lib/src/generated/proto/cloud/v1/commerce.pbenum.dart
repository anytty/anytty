// This is a generated file - do not edit.
//
// Generated from cloud/v1/commerce.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class PlanState extends $pb.ProtobufEnum {
  static const PlanState PLAN_STATE_UNSPECIFIED =
      PlanState._(0, _omitEnumNames ? '' : 'PLAN_STATE_UNSPECIFIED');
  static const PlanState PLAN_STATE_DRAFT =
      PlanState._(1, _omitEnumNames ? '' : 'PLAN_STATE_DRAFT');
  static const PlanState PLAN_STATE_PUBLISHED =
      PlanState._(2, _omitEnumNames ? '' : 'PLAN_STATE_PUBLISHED');
  static const PlanState PLAN_STATE_RETIRED =
      PlanState._(3, _omitEnumNames ? '' : 'PLAN_STATE_RETIRED');

  static const $core.List<PlanState> values = <PlanState>[
    PLAN_STATE_UNSPECIFIED,
    PLAN_STATE_DRAFT,
    PLAN_STATE_PUBLISHED,
    PLAN_STATE_RETIRED,
  ];

  static final $core.List<PlanState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static PlanState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PlanState._(super.value, super.name);
}

class OrderStatus extends $pb.ProtobufEnum {
  static const OrderStatus ORDER_STATUS_UNSPECIFIED =
      OrderStatus._(0, _omitEnumNames ? '' : 'ORDER_STATUS_UNSPECIFIED');
  static const OrderStatus ORDER_STATUS_PENDING =
      OrderStatus._(1, _omitEnumNames ? '' : 'ORDER_STATUS_PENDING');
  static const OrderStatus ORDER_STATUS_PAID =
      OrderStatus._(2, _omitEnumNames ? '' : 'ORDER_STATUS_PAID');
  static const OrderStatus ORDER_STATUS_PAYMENT_FAILED =
      OrderStatus._(3, _omitEnumNames ? '' : 'ORDER_STATUS_PAYMENT_FAILED');
  static const OrderStatus ORDER_STATUS_REFUNDED =
      OrderStatus._(4, _omitEnumNames ? '' : 'ORDER_STATUS_REFUNDED');
  static const OrderStatus ORDER_STATUS_REVOKED =
      OrderStatus._(5, _omitEnumNames ? '' : 'ORDER_STATUS_REVOKED');

  static const $core.List<OrderStatus> values = <OrderStatus>[
    ORDER_STATUS_UNSPECIFIED,
    ORDER_STATUS_PENDING,
    ORDER_STATUS_PAID,
    ORDER_STATUS_PAYMENT_FAILED,
    ORDER_STATUS_REFUNDED,
    ORDER_STATUS_REVOKED,
  ];

  static final $core.List<OrderStatus?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static OrderStatus? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const OrderStatus._(super.value, super.name);
}

class PaymentAttemptStatus extends $pb.ProtobufEnum {
  static const PaymentAttemptStatus PAYMENT_ATTEMPT_STATUS_UNSPECIFIED =
      PaymentAttemptStatus._(
          0, _omitEnumNames ? '' : 'PAYMENT_ATTEMPT_STATUS_UNSPECIFIED');
  static const PaymentAttemptStatus PAYMENT_ATTEMPT_STATUS_PENDING =
      PaymentAttemptStatus._(
          1, _omitEnumNames ? '' : 'PAYMENT_ATTEMPT_STATUS_PENDING');
  static const PaymentAttemptStatus PAYMENT_ATTEMPT_STATUS_SUCCEEDED =
      PaymentAttemptStatus._(
          2, _omitEnumNames ? '' : 'PAYMENT_ATTEMPT_STATUS_SUCCEEDED');
  static const PaymentAttemptStatus PAYMENT_ATTEMPT_STATUS_FAILED =
      PaymentAttemptStatus._(
          3, _omitEnumNames ? '' : 'PAYMENT_ATTEMPT_STATUS_FAILED');

  static const $core.List<PaymentAttemptStatus> values = <PaymentAttemptStatus>[
    PAYMENT_ATTEMPT_STATUS_UNSPECIFIED,
    PAYMENT_ATTEMPT_STATUS_PENDING,
    PAYMENT_ATTEMPT_STATUS_SUCCEEDED,
    PAYMENT_ATTEMPT_STATUS_FAILED,
  ];

  static final $core.List<PaymentAttemptStatus?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static PaymentAttemptStatus? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PaymentAttemptStatus._(super.value, super.name);
}

class PaymentEventType extends $pb.ProtobufEnum {
  static const PaymentEventType PAYMENT_EVENT_TYPE_UNSPECIFIED =
      PaymentEventType._(
          0, _omitEnumNames ? '' : 'PAYMENT_EVENT_TYPE_UNSPECIFIED');
  static const PaymentEventType PAYMENT_EVENT_TYPE_SUCCEEDED =
      PaymentEventType._(
          1, _omitEnumNames ? '' : 'PAYMENT_EVENT_TYPE_SUCCEEDED');
  static const PaymentEventType PAYMENT_EVENT_TYPE_FAILED =
      PaymentEventType._(2, _omitEnumNames ? '' : 'PAYMENT_EVENT_TYPE_FAILED');
  static const PaymentEventType PAYMENT_EVENT_TYPE_REFUNDED =
      PaymentEventType._(
          3, _omitEnumNames ? '' : 'PAYMENT_EVENT_TYPE_REFUNDED');
  static const PaymentEventType PAYMENT_EVENT_TYPE_REVOKED =
      PaymentEventType._(4, _omitEnumNames ? '' : 'PAYMENT_EVENT_TYPE_REVOKED');
  static const PaymentEventType PAYMENT_EVENT_TYPE_CHARGEBACK =
      PaymentEventType._(
          5, _omitEnumNames ? '' : 'PAYMENT_EVENT_TYPE_CHARGEBACK');

  static const $core.List<PaymentEventType> values = <PaymentEventType>[
    PAYMENT_EVENT_TYPE_UNSPECIFIED,
    PAYMENT_EVENT_TYPE_SUCCEEDED,
    PAYMENT_EVENT_TYPE_FAILED,
    PAYMENT_EVENT_TYPE_REFUNDED,
    PAYMENT_EVENT_TYPE_REVOKED,
    PAYMENT_EVENT_TYPE_CHARGEBACK,
  ];

  static final $core.List<PaymentEventType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static PaymentEventType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PaymentEventType._(super.value, super.name);
}

class SubscriptionState extends $pb.ProtobufEnum {
  static const SubscriptionState SUBSCRIPTION_STATE_UNSPECIFIED =
      SubscriptionState._(
          0, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_UNSPECIFIED');
  static const SubscriptionState SUBSCRIPTION_STATE_ACTIVE =
      SubscriptionState._(1, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_ACTIVE');
  static const SubscriptionState SUBSCRIPTION_STATE_CANCEL_AT_PERIOD_END =
      SubscriptionState._(
          2, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_CANCEL_AT_PERIOD_END');
  static const SubscriptionState SUBSCRIPTION_STATE_CANCELED =
      SubscriptionState._(
          3, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_CANCELED');
  static const SubscriptionState SUBSCRIPTION_STATE_SUSPENDED =
      SubscriptionState._(
          4, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_SUSPENDED');
  static const SubscriptionState SUBSCRIPTION_STATE_EXPIRED =
      SubscriptionState._(
          5, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_EXPIRED');
  static const SubscriptionState SUBSCRIPTION_STATE_PAST_DUE =
      SubscriptionState._(
          6, _omitEnumNames ? '' : 'SUBSCRIPTION_STATE_PAST_DUE');

  static const $core.List<SubscriptionState> values = <SubscriptionState>[
    SUBSCRIPTION_STATE_UNSPECIFIED,
    SUBSCRIPTION_STATE_ACTIVE,
    SUBSCRIPTION_STATE_CANCEL_AT_PERIOD_END,
    SUBSCRIPTION_STATE_CANCELED,
    SUBSCRIPTION_STATE_SUSPENDED,
    SUBSCRIPTION_STATE_EXPIRED,
    SUBSCRIPTION_STATE_PAST_DUE,
  ];

  static final $core.List<SubscriptionState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static SubscriptionState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const SubscriptionState._(super.value, super.name);
}

class SubscriptionTransition extends $pb.ProtobufEnum {
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_UNSPECIFIED =
      SubscriptionTransition._(
          0, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_UNSPECIFIED');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_ACTIVATE =
      SubscriptionTransition._(
          1, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_ACTIVATE');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_RENEW =
      SubscriptionTransition._(
          2, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_RENEW');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_UPGRADE =
      SubscriptionTransition._(
          3, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_UPGRADE');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_DOWNGRADE =
      SubscriptionTransition._(
          4, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_DOWNGRADE');
  static const SubscriptionTransition
      SUBSCRIPTION_TRANSITION_CANCEL_AT_PERIOD_END = SubscriptionTransition._(5,
          _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_CANCEL_AT_PERIOD_END');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_RESUME =
      SubscriptionTransition._(
          6, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_RESUME');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_SUSPEND =
      SubscriptionTransition._(
          7, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_SUSPEND');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_RESTORE =
      SubscriptionTransition._(
          8, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_RESTORE');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_EXPIRE =
      SubscriptionTransition._(
          9, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_EXPIRE');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_REFUND =
      SubscriptionTransition._(
          10, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_REFUND');
  static const SubscriptionTransition SUBSCRIPTION_TRANSITION_REVOKE =
      SubscriptionTransition._(
          11, _omitEnumNames ? '' : 'SUBSCRIPTION_TRANSITION_REVOKE');

  static const $core.List<SubscriptionTransition> values =
      <SubscriptionTransition>[
    SUBSCRIPTION_TRANSITION_UNSPECIFIED,
    SUBSCRIPTION_TRANSITION_ACTIVATE,
    SUBSCRIPTION_TRANSITION_RENEW,
    SUBSCRIPTION_TRANSITION_UPGRADE,
    SUBSCRIPTION_TRANSITION_DOWNGRADE,
    SUBSCRIPTION_TRANSITION_CANCEL_AT_PERIOD_END,
    SUBSCRIPTION_TRANSITION_RESUME,
    SUBSCRIPTION_TRANSITION_SUSPEND,
    SUBSCRIPTION_TRANSITION_RESTORE,
    SUBSCRIPTION_TRANSITION_EXPIRE,
    SUBSCRIPTION_TRANSITION_REFUND,
    SUBSCRIPTION_TRANSITION_REVOKE,
  ];

  static final $core.List<SubscriptionTransition?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 11);
  static SubscriptionTransition? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const SubscriptionTransition._(super.value, super.name);
}

class EntitlementState extends $pb.ProtobufEnum {
  static const EntitlementState ENTITLEMENT_STATE_UNSPECIFIED =
      EntitlementState._(
          0, _omitEnumNames ? '' : 'ENTITLEMENT_STATE_UNSPECIFIED');
  static const EntitlementState ENTITLEMENT_STATE_ACTIVE =
      EntitlementState._(1, _omitEnumNames ? '' : 'ENTITLEMENT_STATE_ACTIVE');
  static const EntitlementState ENTITLEMENT_STATE_SUSPENDED =
      EntitlementState._(
          2, _omitEnumNames ? '' : 'ENTITLEMENT_STATE_SUSPENDED');
  static const EntitlementState ENTITLEMENT_STATE_EXPIRED =
      EntitlementState._(3, _omitEnumNames ? '' : 'ENTITLEMENT_STATE_EXPIRED');

  static const $core.List<EntitlementState> values = <EntitlementState>[
    ENTITLEMENT_STATE_UNSPECIFIED,
    ENTITLEMENT_STATE_ACTIVE,
    ENTITLEMENT_STATE_SUSPENDED,
    ENTITLEMENT_STATE_EXPIRED,
  ];

  static final $core.List<EntitlementState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static EntitlementState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const EntitlementState._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
