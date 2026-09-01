// This is a generated file - do not edit.
//
// Generated from cloud/v1/commerce.proto.

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
    as $0;

@$core.Deprecated('Use planStateDescriptor instead')
const PlanState$json = {
  '1': 'PlanState',
  '2': [
    {'1': 'PLAN_STATE_UNSPECIFIED', '2': 0},
    {'1': 'PLAN_STATE_DRAFT', '2': 1},
    {'1': 'PLAN_STATE_PUBLISHED', '2': 2},
    {'1': 'PLAN_STATE_RETIRED', '2': 3},
  ],
};

/// Descriptor for `PlanState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List planStateDescriptor = $convert.base64Decode(
    'CglQbGFuU3RhdGUSGgoWUExBTl9TVEFURV9VTlNQRUNJRklFRBAAEhQKEFBMQU5fU1RBVEVfRF'
    'JBRlQQARIYChRQTEFOX1NUQVRFX1BVQkxJU0hFRBACEhYKElBMQU5fU1RBVEVfUkVUSVJFRBAD');

@$core.Deprecated('Use orderStatusDescriptor instead')
const OrderStatus$json = {
  '1': 'OrderStatus',
  '2': [
    {'1': 'ORDER_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'ORDER_STATUS_PENDING', '2': 1},
    {'1': 'ORDER_STATUS_PAID', '2': 2},
    {'1': 'ORDER_STATUS_PAYMENT_FAILED', '2': 3},
    {'1': 'ORDER_STATUS_REFUNDED', '2': 4},
    {'1': 'ORDER_STATUS_REVOKED', '2': 5},
  ],
};

/// Descriptor for `OrderStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List orderStatusDescriptor = $convert.base64Decode(
    'CgtPcmRlclN0YXR1cxIcChhPUkRFUl9TVEFUVVNfVU5TUEVDSUZJRUQQABIYChRPUkRFUl9TVE'
    'FUVVNfUEVORElORxABEhUKEU9SREVSX1NUQVRVU19QQUlEEAISHwobT1JERVJfU1RBVFVTX1BB'
    'WU1FTlRfRkFJTEVEEAMSGQoVT1JERVJfU1RBVFVTX1JFRlVOREVEEAQSGAoUT1JERVJfU1RBVF'
    'VTX1JFVk9LRUQQBQ==');

@$core.Deprecated('Use paymentAttemptStatusDescriptor instead')
const PaymentAttemptStatus$json = {
  '1': 'PaymentAttemptStatus',
  '2': [
    {'1': 'PAYMENT_ATTEMPT_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'PAYMENT_ATTEMPT_STATUS_PENDING', '2': 1},
    {'1': 'PAYMENT_ATTEMPT_STATUS_SUCCEEDED', '2': 2},
    {'1': 'PAYMENT_ATTEMPT_STATUS_FAILED', '2': 3},
  ],
};

/// Descriptor for `PaymentAttemptStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List paymentAttemptStatusDescriptor = $convert.base64Decode(
    'ChRQYXltZW50QXR0ZW1wdFN0YXR1cxImCiJQQVlNRU5UX0FUVEVNUFRfU1RBVFVTX1VOU1BFQ0'
    'lGSUVEEAASIgoeUEFZTUVOVF9BVFRFTVBUX1NUQVRVU19QRU5ESU5HEAESJAogUEFZTUVOVF9B'
    'VFRFTVBUX1NUQVRVU19TVUNDRUVERUQQAhIhCh1QQVlNRU5UX0FUVEVNUFRfU1RBVFVTX0ZBSU'
    'xFRBAD');

@$core.Deprecated('Use paymentEventTypeDescriptor instead')
const PaymentEventType$json = {
  '1': 'PaymentEventType',
  '2': [
    {'1': 'PAYMENT_EVENT_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'PAYMENT_EVENT_TYPE_SUCCEEDED', '2': 1},
    {'1': 'PAYMENT_EVENT_TYPE_FAILED', '2': 2},
    {'1': 'PAYMENT_EVENT_TYPE_REFUNDED', '2': 3},
    {'1': 'PAYMENT_EVENT_TYPE_REVOKED', '2': 4},
    {'1': 'PAYMENT_EVENT_TYPE_CHARGEBACK', '2': 5},
  ],
};

/// Descriptor for `PaymentEventType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List paymentEventTypeDescriptor = $convert.base64Decode(
    'ChBQYXltZW50RXZlbnRUeXBlEiIKHlBBWU1FTlRfRVZFTlRfVFlQRV9VTlNQRUNJRklFRBAAEi'
    'AKHFBBWU1FTlRfRVZFTlRfVFlQRV9TVUNDRUVERUQQARIdChlQQVlNRU5UX0VWRU5UX1RZUEVf'
    'RkFJTEVEEAISHwobUEFZTUVOVF9FVkVOVF9UWVBFX1JFRlVOREVEEAMSHgoaUEFZTUVOVF9FVk'
    'VOVF9UWVBFX1JFVk9LRUQQBBIhCh1QQVlNRU5UX0VWRU5UX1RZUEVfQ0hBUkdFQkFDSxAF');

@$core.Deprecated('Use subscriptionStateDescriptor instead')
const SubscriptionState$json = {
  '1': 'SubscriptionState',
  '2': [
    {'1': 'SUBSCRIPTION_STATE_UNSPECIFIED', '2': 0},
    {'1': 'SUBSCRIPTION_STATE_ACTIVE', '2': 1},
    {'1': 'SUBSCRIPTION_STATE_CANCEL_AT_PERIOD_END', '2': 2},
    {'1': 'SUBSCRIPTION_STATE_CANCELED', '2': 3},
    {'1': 'SUBSCRIPTION_STATE_SUSPENDED', '2': 4},
    {'1': 'SUBSCRIPTION_STATE_EXPIRED', '2': 5},
    {'1': 'SUBSCRIPTION_STATE_PAST_DUE', '2': 6},
  ],
};

/// Descriptor for `SubscriptionState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List subscriptionStateDescriptor = $convert.base64Decode(
    'ChFTdWJzY3JpcHRpb25TdGF0ZRIiCh5TVUJTQ1JJUFRJT05fU1RBVEVfVU5TUEVDSUZJRUQQAB'
    'IdChlTVUJTQ1JJUFRJT05fU1RBVEVfQUNUSVZFEAESKwonU1VCU0NSSVBUSU9OX1NUQVRFX0NB'
    'TkNFTF9BVF9QRVJJT0RfRU5EEAISHwobU1VCU0NSSVBUSU9OX1NUQVRFX0NBTkNFTEVEEAMSIA'
    'ocU1VCU0NSSVBUSU9OX1NUQVRFX1NVU1BFTkRFRBAEEh4KGlNVQlNDUklQVElPTl9TVEFURV9F'
    'WFBJUkVEEAUSHwobU1VCU0NSSVBUSU9OX1NUQVRFX1BBU1RfRFVFEAY=');

@$core.Deprecated('Use subscriptionTransitionDescriptor instead')
const SubscriptionTransition$json = {
  '1': 'SubscriptionTransition',
  '2': [
    {'1': 'SUBSCRIPTION_TRANSITION_UNSPECIFIED', '2': 0},
    {'1': 'SUBSCRIPTION_TRANSITION_ACTIVATE', '2': 1},
    {'1': 'SUBSCRIPTION_TRANSITION_RENEW', '2': 2},
    {'1': 'SUBSCRIPTION_TRANSITION_UPGRADE', '2': 3},
    {'1': 'SUBSCRIPTION_TRANSITION_DOWNGRADE', '2': 4},
    {'1': 'SUBSCRIPTION_TRANSITION_CANCEL_AT_PERIOD_END', '2': 5},
    {'1': 'SUBSCRIPTION_TRANSITION_RESUME', '2': 6},
    {'1': 'SUBSCRIPTION_TRANSITION_SUSPEND', '2': 7},
    {'1': 'SUBSCRIPTION_TRANSITION_RESTORE', '2': 8},
    {'1': 'SUBSCRIPTION_TRANSITION_EXPIRE', '2': 9},
    {'1': 'SUBSCRIPTION_TRANSITION_REFUND', '2': 10},
    {'1': 'SUBSCRIPTION_TRANSITION_REVOKE', '2': 11},
  ],
};

/// Descriptor for `SubscriptionTransition`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List subscriptionTransitionDescriptor = $convert.base64Decode(
    'ChZTdWJzY3JpcHRpb25UcmFuc2l0aW9uEicKI1NVQlNDUklQVElPTl9UUkFOU0lUSU9OX1VOU1'
    'BFQ0lGSUVEEAASJAogU1VCU0NSSVBUSU9OX1RSQU5TSVRJT05fQUNUSVZBVEUQARIhCh1TVUJT'
    'Q1JJUFRJT05fVFJBTlNJVElPTl9SRU5FVxACEiMKH1NVQlNDUklQVElPTl9UUkFOU0lUSU9OX1'
    'VQR1JBREUQAxIlCiFTVUJTQ1JJUFRJT05fVFJBTlNJVElPTl9ET1dOR1JBREUQBBIwCixTVUJT'
    'Q1JJUFRJT05fVFJBTlNJVElPTl9DQU5DRUxfQVRfUEVSSU9EX0VORBAFEiIKHlNVQlNDUklQVE'
    'lPTl9UUkFOU0lUSU9OX1JFU1VNRRAGEiMKH1NVQlNDUklQVElPTl9UUkFOU0lUSU9OX1NVU1BF'
    'TkQQBxIjCh9TVUJTQ1JJUFRJT05fVFJBTlNJVElPTl9SRVNUT1JFEAgSIgoeU1VCU0NSSVBUSU'
    '9OX1RSQU5TSVRJT05fRVhQSVJFEAkSIgoeU1VCU0NSSVBUSU9OX1RSQU5TSVRJT05fUkVGVU5E'
    'EAoSIgoeU1VCU0NSSVBUSU9OX1RSQU5TSVRJT05fUkVWT0tFEAs=');

@$core.Deprecated('Use entitlementStateDescriptor instead')
const EntitlementState$json = {
  '1': 'EntitlementState',
  '2': [
    {'1': 'ENTITLEMENT_STATE_UNSPECIFIED', '2': 0},
    {'1': 'ENTITLEMENT_STATE_ACTIVE', '2': 1},
    {'1': 'ENTITLEMENT_STATE_SUSPENDED', '2': 2},
    {'1': 'ENTITLEMENT_STATE_EXPIRED', '2': 3},
  ],
};

/// Descriptor for `EntitlementState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List entitlementStateDescriptor = $convert.base64Decode(
    'ChBFbnRpdGxlbWVudFN0YXRlEiEKHUVOVElUTEVNRU5UX1NUQVRFX1VOU1BFQ0lGSUVEEAASHA'
    'oYRU5USVRMRU1FTlRfU1RBVEVfQUNUSVZFEAESHwobRU5USVRMRU1FTlRfU1RBVEVfU1VTUEVO'
    'REVEEAISHQoZRU5USVRMRU1FTlRfU1RBVEVfRVhQSVJFRBAD');

@$core.Deprecated('Use moneyDescriptor instead')
const Money$json = {
  '1': 'Money',
  '2': [
    {'1': 'currency', '3': 1, '4': 1, '5': 9, '10': 'currency'},
    {'1': 'minor_units', '3': 2, '4': 1, '5': 3, '10': 'minorUnits'},
  ],
};

/// Descriptor for `Money`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List moneyDescriptor = $convert.base64Decode(
    'CgVNb25leRIaCghjdXJyZW5jeRgBIAEoCVIIY3VycmVuY3kSHwoLbWlub3JfdW5pdHMYAiABKA'
    'NSCm1pbm9yVW5pdHM=');

@$core.Deprecated('Use cloudCapabilityDescriptor instead')
const CloudCapability$json = {
  '1': 'CloudCapability',
  '2': [
    {
      '1': 'managed_p2p_enabled',
      '3': 1,
      '4': 1,
      '5': 8,
      '10': 'managedP2pEnabled'
    },
    {
      '1': 'managed_p2p_max_concurrency',
      '3': 2,
      '4': 1,
      '5': 13,
      '10': 'managedP2pMaxConcurrency'
    },
    {'1': 'relay_enabled', '3': 3, '4': 1, '5': 8, '10': 'relayEnabled'},
    {
      '1': 'relay_max_concurrency',
      '3': 4,
      '4': 1,
      '5': 13,
      '10': 'relayMaxConcurrency'
    },
    {
      '1': 'relay_max_bytes_per_period',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'relayMaxBytesPerPeriod'
    },
    {
      '1': 'relay_max_bytes_per_lease',
      '3': 6,
      '4': 1,
      '5': 4,
      '10': 'relayMaxBytesPerLease'
    },
    {
      '1': 'relay_max_rate_bytes_per_second',
      '3': 7,
      '4': 1,
      '5': 4,
      '10': 'relayMaxRateBytesPerSecond'
    },
    {
      '1': 'cloud_daemon_limit',
      '3': 8,
      '4': 1,
      '5': 13,
      '10': 'cloudDaemonLimit'
    },
    {'1': 'allowed_regions', '3': 9, '4': 3, '5': 9, '10': 'allowedRegions'},
  ],
};

/// Descriptor for `CloudCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudCapabilityDescriptor = $convert.base64Decode(
    'Cg9DbG91ZENhcGFiaWxpdHkSLgoTbWFuYWdlZF9wMnBfZW5hYmxlZBgBIAEoCFIRbWFuYWdlZF'
    'AycEVuYWJsZWQSPQobbWFuYWdlZF9wMnBfbWF4X2NvbmN1cnJlbmN5GAIgASgNUhhtYW5hZ2Vk'
    'UDJwTWF4Q29uY3VycmVuY3kSIwoNcmVsYXlfZW5hYmxlZBgDIAEoCFIMcmVsYXlFbmFibGVkEj'
    'IKFXJlbGF5X21heF9jb25jdXJyZW5jeRgEIAEoDVITcmVsYXlNYXhDb25jdXJyZW5jeRI6Chpy'
    'ZWxheV9tYXhfYnl0ZXNfcGVyX3BlcmlvZBgFIAEoBFIWcmVsYXlNYXhCeXRlc1BlclBlcmlvZB'
    'I4ChlyZWxheV9tYXhfYnl0ZXNfcGVyX2xlYXNlGAYgASgEUhVyZWxheU1heEJ5dGVzUGVyTGVh'
    'c2USQwofcmVsYXlfbWF4X3JhdGVfYnl0ZXNfcGVyX3NlY29uZBgHIAEoBFIacmVsYXlNYXhSYX'
    'RlQnl0ZXNQZXJTZWNvbmQSLAoSY2xvdWRfZGFlbW9uX2xpbWl0GAggASgNUhBjbG91ZERhZW1v'
    'bkxpbWl0EicKD2FsbG93ZWRfcmVnaW9ucxgJIAMoCVIOYWxsb3dlZFJlZ2lvbnM=');

@$core.Deprecated('Use planDefinitionDescriptor instead')
const PlanDefinition$json = {
  '1': 'PlanDefinition',
  '2': [
    {'1': 'plan_id', '3': 1, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'version', '3': 2, '4': 1, '5': 4, '10': 'version'},
    {'1': 'name', '3': 3, '4': 1, '5': 9, '10': 'name'},
    {'1': 'description', '3': 4, '4': 1, '5': 9, '10': 'description'},
    {
      '1': 'state',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.PlanState',
      '10': 'state'
    },
    {
      '1': 'billing_period_days',
      '3': 6,
      '4': 1,
      '5': 13,
      '10': 'billingPeriodDays'
    },
    {
      '1': 'monthly_price',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.Money',
      '10': 'monthlyPrice'
    },
    {
      '1': 'yearly_price',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.Money',
      '10': 'yearlyPrice'
    },
    {
      '1': 'capability',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudCapability',
      '10': 'capability'
    },
    {'1': 'revision', '3': 10, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'created_at',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createdAt'
    },
    {
      '1': 'published_at',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'publishedAt'
    },
  ],
};

/// Descriptor for `PlanDefinition`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List planDefinitionDescriptor = $convert.base64Decode(
    'Cg5QbGFuRGVmaW5pdGlvbhIXCgdwbGFuX2lkGAEgASgJUgZwbGFuSWQSGAoHdmVyc2lvbhgCIA'
    'EoBFIHdmVyc2lvbhISCgRuYW1lGAMgASgJUgRuYW1lEiAKC2Rlc2NyaXB0aW9uGAQgASgJUgtk'
    'ZXNjcmlwdGlvbhIwCgVzdGF0ZRgFIAEoDjIaLmFueXR0eS5jbG91ZC52MS5QbGFuU3RhdGVSBX'
    'N0YXRlEi4KE2JpbGxpbmdfcGVyaW9kX2RheXMYBiABKA1SEWJpbGxpbmdQZXJpb2REYXlzEjsK'
    'DW1vbnRobHlfcHJpY2UYByABKAsyFi5hbnl0dHkuY2xvdWQudjEuTW9uZXlSDG1vbnRobHlQcm'
    'ljZRI5Cgx5ZWFybHlfcHJpY2UYCCABKAsyFi5hbnl0dHkuY2xvdWQudjEuTW9uZXlSC3llYXJs'
    'eVByaWNlEkAKCmNhcGFiaWxpdHkYCSABKAsyIC5hbnl0dHkuY2xvdWQudjEuQ2xvdWRDYXBhYm'
    'lsaXR5UgpjYXBhYmlsaXR5EhoKCHJldmlzaW9uGAogASgEUghyZXZpc2lvbhI5CgpjcmVhdGVk'
    'X2F0GAsgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJY3JlYXRlZEF0Ej0KDHB1Ym'
    'xpc2hlZF9hdBgMIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC3B1Ymxpc2hlZEF0');

@$core.Deprecated('Use orderProjectionDescriptor instead')
const OrderProjection$json = {
  '1': 'OrderProjection',
  '2': [
    {'1': 'order_id', '3': 1, '4': 1, '5': 9, '10': 'orderId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'plan_id', '3': 3, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'plan_version', '3': 4, '4': 1, '5': 4, '10': 'planVersion'},
    {
      '1': 'status',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.OrderStatus',
      '10': 'status'
    },
    {
      '1': 'amount',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.Money',
      '10': 'amount'
    },
    {'1': 'provider', '3': 7, '4': 1, '5': 9, '10': 'provider'},
    {
      '1': 'provider_reference',
      '3': 8,
      '4': 1,
      '5': 9,
      '10': 'providerReference'
    },
    {'1': 'idempotency_key', '3': 9, '4': 1, '5': 9, '10': 'idempotencyKey'},
    {
      '1': 'requested_transition',
      '3': 10,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SubscriptionTransition',
      '10': 'requestedTransition'
    },
    {'1': 'revision', '3': 11, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'created_at',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createdAt'
    },
    {
      '1': 'settled_at',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'settledAt'
    },
    {
      '1': 'account_display_name',
      '3': 14,
      '4': 1,
      '5': 9,
      '10': 'accountDisplayName'
    },
    {'1': 'account_email', '3': 15, '4': 1, '5': 9, '10': 'accountEmail'},
    {'1': 'plan_name', '3': 16, '4': 1, '5': 9, '10': 'planName'},
  ],
};

/// Descriptor for `OrderProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List orderProjectionDescriptor = $convert.base64Decode(
    'Cg9PcmRlclByb2plY3Rpb24SGQoIb3JkZXJfaWQYASABKAlSB29yZGVySWQSHQoKYWNjb3VudF'
    '9pZBgCIAEoCVIJYWNjb3VudElkEhcKB3BsYW5faWQYAyABKAlSBnBsYW5JZBIhCgxwbGFuX3Zl'
    'cnNpb24YBCABKARSC3BsYW5WZXJzaW9uEjQKBnN0YXR1cxgFIAEoDjIcLmFueXR0eS5jbG91ZC'
    '52MS5PcmRlclN0YXR1c1IGc3RhdHVzEi4KBmFtb3VudBgGIAEoCzIWLmFueXR0eS5jbG91ZC52'
    'MS5Nb25leVIGYW1vdW50EhoKCHByb3ZpZGVyGAcgASgJUghwcm92aWRlchItChJwcm92aWRlcl'
    '9yZWZlcmVuY2UYCCABKAlSEXByb3ZpZGVyUmVmZXJlbmNlEicKD2lkZW1wb3RlbmN5X2tleRgJ'
    'IAEoCVIOaWRlbXBvdGVuY3lLZXkSWgoUcmVxdWVzdGVkX3RyYW5zaXRpb24YCiABKA4yJy5hbn'
    'l0dHkuY2xvdWQudjEuU3Vic2NyaXB0aW9uVHJhbnNpdGlvblITcmVxdWVzdGVkVHJhbnNpdGlv'
    'bhIaCghyZXZpc2lvbhgLIAEoBFIIcmV2aXNpb24SOQoKY3JlYXRlZF9hdBgMIAEoCzIaLmdvb2'
    'dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCWNyZWF0ZWRBdBI5CgpzZXR0bGVkX2F0GA0gASgLMhou'
    'Z29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJc2V0dGxlZEF0EjAKFGFjY291bnRfZGlzcGxheV'
    '9uYW1lGA4gASgJUhJhY2NvdW50RGlzcGxheU5hbWUSIwoNYWNjb3VudF9lbWFpbBgPIAEoCVIM'
    'YWNjb3VudEVtYWlsEhsKCXBsYW5fbmFtZRgQIAEoCVIIcGxhbk5hbWU=');

@$core.Deprecated('Use paymentAttemptProjectionDescriptor instead')
const PaymentAttemptProjection$json = {
  '1': 'PaymentAttemptProjection',
  '2': [
    {
      '1': 'payment_attempt_id',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'paymentAttemptId'
    },
    {'1': 'order_id', '3': 2, '4': 1, '5': 9, '10': 'orderId'},
    {'1': 'account_id', '3': 3, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'provider', '3': 4, '4': 1, '5': 9, '10': 'provider'},
    {
      '1': 'provider_reference',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'providerReference'
    },
    {
      '1': 'status',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.PaymentAttemptStatus',
      '10': 'status'
    },
    {'1': 'revision', '3': 7, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'created_at',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createdAt'
    },
    {
      '1': 'updated_at',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'updatedAt'
    },
  ],
};

/// Descriptor for `PaymentAttemptProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List paymentAttemptProjectionDescriptor = $convert.base64Decode(
    'ChhQYXltZW50QXR0ZW1wdFByb2plY3Rpb24SLAoScGF5bWVudF9hdHRlbXB0X2lkGAEgASgJUh'
    'BwYXltZW50QXR0ZW1wdElkEhkKCG9yZGVyX2lkGAIgASgJUgdvcmRlcklkEh0KCmFjY291bnRf'
    'aWQYAyABKAlSCWFjY291bnRJZBIaCghwcm92aWRlchgEIAEoCVIIcHJvdmlkZXISLQoScHJvdm'
    'lkZXJfcmVmZXJlbmNlGAUgASgJUhFwcm92aWRlclJlZmVyZW5jZRI9CgZzdGF0dXMYBiABKA4y'
    'JS5hbnl0dHkuY2xvdWQudjEuUGF5bWVudEF0dGVtcHRTdGF0dXNSBnN0YXR1cxIaCghyZXZpc2'
    'lvbhgHIAEoBFIIcmV2aXNpb24SOQoKY3JlYXRlZF9hdBgIIAEoCzIaLmdvb2dsZS5wcm90b2J1'
    'Zi5UaW1lc3RhbXBSCWNyZWF0ZWRBdBI5Cgp1cGRhdGVkX2F0GAkgASgLMhouZ29vZ2xlLnByb3'
    'RvYnVmLlRpbWVzdGFtcFIJdXBkYXRlZEF0');

@$core.Deprecated('Use subscriptionProjectionDescriptor instead')
const SubscriptionProjection$json = {
  '1': 'SubscriptionProjection',
  '2': [
    {'1': 'subscription_id', '3': 1, '4': 1, '5': 9, '10': 'subscriptionId'},
    {'1': 'account_id', '3': 2, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'plan_id', '3': 3, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'plan_version', '3': 4, '4': 1, '5': 4, '10': 'planVersion'},
    {'1': 'source_order_id', '3': 5, '4': 1, '5': 9, '10': 'sourceOrderId'},
    {
      '1': 'state',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SubscriptionState',
      '10': 'state'
    },
    {
      '1': 'cancel_at_period_end',
      '3': 7,
      '4': 1,
      '5': 8,
      '10': 'cancelAtPeriodEnd'
    },
    {'1': 'revision', '3': 8, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'period_start',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodStart'
    },
    {
      '1': 'period_end',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodEnd'
    },
    {
      '1': 'updated_at',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'updatedAt'
    },
    {'1': 'plan_name', '3': 12, '4': 1, '5': 9, '10': 'planName'},
    {'1': 'provider', '3': 13, '4': 1, '5': 9, '10': 'provider'},
    {
      '1': 'account_display_name',
      '3': 14,
      '4': 1,
      '5': 9,
      '10': 'accountDisplayName'
    },
    {'1': 'account_email', '3': 15, '4': 1, '5': 9, '10': 'accountEmail'},
  ],
};

/// Descriptor for `SubscriptionProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List subscriptionProjectionDescriptor = $convert.base64Decode(
    'ChZTdWJzY3JpcHRpb25Qcm9qZWN0aW9uEicKD3N1YnNjcmlwdGlvbl9pZBgBIAEoCVIOc3Vic2'
    'NyaXB0aW9uSWQSHQoKYWNjb3VudF9pZBgCIAEoCVIJYWNjb3VudElkEhcKB3BsYW5faWQYAyAB'
    'KAlSBnBsYW5JZBIhCgxwbGFuX3ZlcnNpb24YBCABKARSC3BsYW5WZXJzaW9uEiYKD3NvdXJjZV'
    '9vcmRlcl9pZBgFIAEoCVINc291cmNlT3JkZXJJZBI4CgVzdGF0ZRgGIAEoDjIiLmFueXR0eS5j'
    'bG91ZC52MS5TdWJzY3JpcHRpb25TdGF0ZVIFc3RhdGUSLwoUY2FuY2VsX2F0X3BlcmlvZF9lbm'
    'QYByABKAhSEWNhbmNlbEF0UGVyaW9kRW5kEhoKCHJldmlzaW9uGAggASgEUghyZXZpc2lvbhI9'
    'CgxwZXJpb2Rfc3RhcnQYCSABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgtwZXJpb2'
    'RTdGFydBI5CgpwZXJpb2RfZW5kGAogASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJ'
    'cGVyaW9kRW5kEjkKCnVwZGF0ZWRfYXQYCyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW'
    '1wUgl1cGRhdGVkQXQSGwoJcGxhbl9uYW1lGAwgASgJUghwbGFuTmFtZRIaCghwcm92aWRlchgN'
    'IAEoCVIIcHJvdmlkZXISMAoUYWNjb3VudF9kaXNwbGF5X25hbWUYDiABKAlSEmFjY291bnREaX'
    'NwbGF5TmFtZRIjCg1hY2NvdW50X2VtYWlsGA8gASgJUgxhY2NvdW50RW1haWw=');

@$core.Deprecated('Use effectiveEntitlementDescriptor instead')
const EffectiveEntitlement$json = {
  '1': 'EffectiveEntitlement',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'state',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.EntitlementState',
      '10': 'state'
    },
    {'1': 'plan_id', '3': 3, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'plan_version', '3': 4, '4': 1, '5': 4, '10': 'planVersion'},
    {'1': 'subscription_id', '3': 5, '4': 1, '5': 9, '10': 'subscriptionId'},
    {
      '1': 'capability',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudCapability',
      '10': 'capability'
    },
    {'1': 'relay_used_bytes', '3': 7, '4': 1, '5': 4, '10': 'relayUsedBytes'},
    {
      '1': 'relay_remaining_bytes',
      '3': 8,
      '4': 1,
      '5': 4,
      '10': 'relayRemainingBytes'
    },
    {
      '1': 'effective_from',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'effectiveFrom'
    },
    {
      '1': 'effective_until',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'effectiveUntil'
    },
    {
      '1': 'computed_at',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'computedAt'
    },
  ],
};

/// Descriptor for `EffectiveEntitlement`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List effectiveEntitlementDescriptor = $convert.base64Decode(
    'ChRFZmZlY3RpdmVFbnRpdGxlbWVudBIdCgphY2NvdW50X2lkGAEgASgJUglhY2NvdW50SWQSNw'
    'oFc3RhdGUYAiABKA4yIS5hbnl0dHkuY2xvdWQudjEuRW50aXRsZW1lbnRTdGF0ZVIFc3RhdGUS'
    'FwoHcGxhbl9pZBgDIAEoCVIGcGxhbklkEiEKDHBsYW5fdmVyc2lvbhgEIAEoBFILcGxhblZlcn'
    'Npb24SJwoPc3Vic2NyaXB0aW9uX2lkGAUgASgJUg5zdWJzY3JpcHRpb25JZBJACgpjYXBhYmls'
    'aXR5GAYgASgLMiAuYW55dHR5LmNsb3VkLnYxLkNsb3VkQ2FwYWJpbGl0eVIKY2FwYWJpbGl0eR'
    'IoChByZWxheV91c2VkX2J5dGVzGAcgASgEUg5yZWxheVVzZWRCeXRlcxIyChVyZWxheV9yZW1h'
    'aW5pbmdfYnl0ZXMYCCABKARSE3JlbGF5UmVtYWluaW5nQnl0ZXMSQQoOZWZmZWN0aXZlX2Zyb2'
    '0YCSABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUg1lZmZlY3RpdmVGcm9tEkMKD2Vm'
    'ZmVjdGl2ZV91bnRpbBgKIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSDmVmZmVjdG'
    'l2ZVVudGlsEjsKC2NvbXB1dGVkX2F0GAsgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFt'
    'cFIKY29tcHV0ZWRBdA==');

@$core.Deprecated('Use usagePeriodProjectionDescriptor instead')
const UsagePeriodProjection$json = {
  '1': 'UsagePeriodProjection',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'period_start',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodStart'
    },
    {
      '1': 'period_end',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'periodEnd'
    },
    {
      '1': 'relay_ingress_bytes',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'relayIngressBytes'
    },
    {
      '1': 'relay_egress_bytes',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'relayEgressBytes'
    },
    {'1': 'relay_total_bytes', '3': 6, '4': 1, '5': 4, '10': 'relayTotalBytes'},
    {'1': 'quota_bytes', '3': 7, '4': 1, '5': 4, '10': 'quotaBytes'},
    {'1': 'remaining_bytes', '3': 8, '4': 1, '5': 4, '10': 'remainingBytes'},
    {'1': 'revision', '3': 9, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'active_relay_reservations',
      '3': 10,
      '4': 1,
      '5': 13,
      '10': 'activeRelayReservations'
    },
    {'1': 'relay_held_bytes', '3': 11, '4': 1, '5': 4, '10': 'relayHeldBytes'},
    {
      '1': 'account_display_name',
      '3': 12,
      '4': 1,
      '5': 9,
      '10': 'accountDisplayName'
    },
    {'1': 'account_email', '3': 13, '4': 1, '5': 9, '10': 'accountEmail'},
  ],
};

/// Descriptor for `UsagePeriodProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List usagePeriodProjectionDescriptor = $convert.base64Decode(
    'ChVVc2FnZVBlcmlvZFByb2plY3Rpb24SHQoKYWNjb3VudF9pZBgBIAEoCVIJYWNjb3VudElkEj'
    '0KDHBlcmlvZF9zdGFydBgCIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC3Blcmlv'
    'ZFN0YXJ0EjkKCnBlcmlvZF9lbmQYAyABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUg'
    'lwZXJpb2RFbmQSLgoTcmVsYXlfaW5ncmVzc19ieXRlcxgEIAEoBFIRcmVsYXlJbmdyZXNzQnl0'
    'ZXMSLAoScmVsYXlfZWdyZXNzX2J5dGVzGAUgASgEUhByZWxheUVncmVzc0J5dGVzEioKEXJlbG'
    'F5X3RvdGFsX2J5dGVzGAYgASgEUg9yZWxheVRvdGFsQnl0ZXMSHwoLcXVvdGFfYnl0ZXMYByAB'
    'KARSCnF1b3RhQnl0ZXMSJwoPcmVtYWluaW5nX2J5dGVzGAggASgEUg5yZW1haW5pbmdCeXRlcx'
    'IaCghyZXZpc2lvbhgJIAEoBFIIcmV2aXNpb24SOgoZYWN0aXZlX3JlbGF5X3Jlc2VydmF0aW9u'
    'cxgKIAEoDVIXYWN0aXZlUmVsYXlSZXNlcnZhdGlvbnMSKAoQcmVsYXlfaGVsZF9ieXRlcxgLIA'
    'EoBFIOcmVsYXlIZWxkQnl0ZXMSMAoUYWNjb3VudF9kaXNwbGF5X25hbWUYDCABKAlSEmFjY291'
    'bnREaXNwbGF5TmFtZRIjCg1hY2NvdW50X2VtYWlsGA0gASgJUgxhY2NvdW50RW1haWw=');

@$core.Deprecated('Use listPlansRequestDescriptor instead')
const ListPlansRequest$json = {
  '1': 'ListPlansRequest',
  '2': [
    {
      '1': 'include_unpublished',
      '3': 1,
      '4': 1,
      '5': 8,
      '10': 'includeUnpublished'
    },
  ],
};

/// Descriptor for `ListPlansRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listPlansRequestDescriptor = $convert.base64Decode(
    'ChBMaXN0UGxhbnNSZXF1ZXN0Ei8KE2luY2x1ZGVfdW5wdWJsaXNoZWQYASABKAhSEmluY2x1ZG'
    'VVbnB1Ymxpc2hlZA==');

@$core.Deprecated('Use listPlansResponseDescriptor instead')
const ListPlansResponse$json = {
  '1': 'ListPlansResponse',
  '2': [
    {
      '1': 'plans',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.PlanDefinition',
      '10': 'plans'
    },
  ],
};

/// Descriptor for `ListPlansResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listPlansResponseDescriptor = $convert.base64Decode(
    'ChFMaXN0UGxhbnNSZXNwb25zZRI1CgVwbGFucxgBIAMoCzIfLmFueXR0eS5jbG91ZC52MS5QbG'
    'FuRGVmaW5pdGlvblIFcGxhbnM=');

@$core.Deprecated('Use createPlanVersionRequestDescriptor instead')
const CreatePlanVersionRequest$json = {
  '1': 'CreatePlanVersionRequest',
  '2': [
    {'1': 'plan_id', '3': 1, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'description', '3': 3, '4': 1, '5': 9, '10': 'description'},
    {
      '1': 'billing_period_days',
      '3': 4,
      '4': 1,
      '5': 13,
      '10': 'billingPeriodDays'
    },
    {
      '1': 'monthly_price',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.Money',
      '10': 'monthlyPrice'
    },
    {
      '1': 'yearly_price',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.Money',
      '10': 'yearlyPrice'
    },
    {
      '1': 'capability',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.CloudCapability',
      '10': 'capability'
    },
  ],
};

/// Descriptor for `CreatePlanVersionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createPlanVersionRequestDescriptor = $convert.base64Decode(
    'ChhDcmVhdGVQbGFuVmVyc2lvblJlcXVlc3QSFwoHcGxhbl9pZBgBIAEoCVIGcGxhbklkEhIKBG'
    '5hbWUYAiABKAlSBG5hbWUSIAoLZGVzY3JpcHRpb24YAyABKAlSC2Rlc2NyaXB0aW9uEi4KE2Jp'
    'bGxpbmdfcGVyaW9kX2RheXMYBCABKA1SEWJpbGxpbmdQZXJpb2REYXlzEjsKDW1vbnRobHlfcH'
    'JpY2UYBSABKAsyFi5hbnl0dHkuY2xvdWQudjEuTW9uZXlSDG1vbnRobHlQcmljZRI5Cgx5ZWFy'
    'bHlfcHJpY2UYBiABKAsyFi5hbnl0dHkuY2xvdWQudjEuTW9uZXlSC3llYXJseVByaWNlEkAKCm'
    'NhcGFiaWxpdHkYByABKAsyIC5hbnl0dHkuY2xvdWQudjEuQ2xvdWRDYXBhYmlsaXR5UgpjYXBh'
    'YmlsaXR5');

@$core.Deprecated('Use createPlanVersionResponseDescriptor instead')
const CreatePlanVersionResponse$json = {
  '1': 'CreatePlanVersionResponse',
  '2': [
    {
      '1': 'plan',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PlanDefinition',
      '10': 'plan'
    },
  ],
};

/// Descriptor for `CreatePlanVersionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createPlanVersionResponseDescriptor =
    $convert.base64Decode(
        'ChlDcmVhdGVQbGFuVmVyc2lvblJlc3BvbnNlEjMKBHBsYW4YASABKAsyHy5hbnl0dHkuY2xvdW'
        'QudjEuUGxhbkRlZmluaXRpb25SBHBsYW4=');

@$core.Deprecated('Use publishPlanVersionRequestDescriptor instead')
const PublishPlanVersionRequest$json = {
  '1': 'PublishPlanVersionRequest',
  '2': [
    {'1': 'plan_id', '3': 1, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'version', '3': 2, '4': 1, '5': 4, '10': 'version'},
    {
      '1': 'expected_revision',
      '3': 3,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
  ],
};

/// Descriptor for `PublishPlanVersionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List publishPlanVersionRequestDescriptor = $convert.base64Decode(
    'ChlQdWJsaXNoUGxhblZlcnNpb25SZXF1ZXN0EhcKB3BsYW5faWQYASABKAlSBnBsYW5JZBIYCg'
    'd2ZXJzaW9uGAIgASgEUgd2ZXJzaW9uEisKEWV4cGVjdGVkX3JldmlzaW9uGAMgASgEUhBleHBl'
    'Y3RlZFJldmlzaW9u');

@$core.Deprecated('Use publishPlanVersionResponseDescriptor instead')
const PublishPlanVersionResponse$json = {
  '1': 'PublishPlanVersionResponse',
  '2': [
    {
      '1': 'plan',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PlanDefinition',
      '10': 'plan'
    },
  ],
};

/// Descriptor for `PublishPlanVersionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List publishPlanVersionResponseDescriptor =
    $convert.base64Decode(
        'ChpQdWJsaXNoUGxhblZlcnNpb25SZXNwb25zZRIzCgRwbGFuGAEgASgLMh8uYW55dHR5LmNsb3'
        'VkLnYxLlBsYW5EZWZpbml0aW9uUgRwbGFu');

@$core.Deprecated('Use createOrderRequestDescriptor instead')
const CreateOrderRequest$json = {
  '1': 'CreateOrderRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'plan_id', '3': 2, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'plan_version', '3': 3, '4': 1, '5': 4, '10': 'planVersion'},
    {'1': 'provider', '3': 4, '4': 1, '5': 9, '10': 'provider'},
    {'1': 'idempotency_key', '3': 5, '4': 1, '5': 9, '10': 'idempotencyKey'},
    {
      '1': 'requested_transition',
      '3': 6,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SubscriptionTransition',
      '10': 'requestedTransition'
    },
    {'1': 'yearly', '3': 7, '4': 1, '5': 8, '10': 'yearly'},
  ],
};

/// Descriptor for `CreateOrderRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createOrderRequestDescriptor = $convert.base64Decode(
    'ChJDcmVhdGVPcmRlclJlcXVlc3QSHQoKYWNjb3VudF9pZBgBIAEoCVIJYWNjb3VudElkEhcKB3'
    'BsYW5faWQYAiABKAlSBnBsYW5JZBIhCgxwbGFuX3ZlcnNpb24YAyABKARSC3BsYW5WZXJzaW9u'
    'EhoKCHByb3ZpZGVyGAQgASgJUghwcm92aWRlchInCg9pZGVtcG90ZW5jeV9rZXkYBSABKAlSDm'
    'lkZW1wb3RlbmN5S2V5EloKFHJlcXVlc3RlZF90cmFuc2l0aW9uGAYgASgOMicuYW55dHR5LmNs'
    'b3VkLnYxLlN1YnNjcmlwdGlvblRyYW5zaXRpb25SE3JlcXVlc3RlZFRyYW5zaXRpb24SFgoGeW'
    'Vhcmx5GAcgASgIUgZ5ZWFybHk=');

@$core.Deprecated('Use createOrderResponseDescriptor instead')
const CreateOrderResponse$json = {
  '1': 'CreateOrderResponse',
  '2': [
    {
      '1': 'order',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.OrderProjection',
      '10': 'order'
    },
    {
      '1': 'payment_attempt',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PaymentAttemptProjection',
      '10': 'paymentAttempt'
    },
  ],
};

/// Descriptor for `CreateOrderResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createOrderResponseDescriptor = $convert.base64Decode(
    'ChNDcmVhdGVPcmRlclJlc3BvbnNlEjYKBW9yZGVyGAEgASgLMiAuYW55dHR5LmNsb3VkLnYxLk'
    '9yZGVyUHJvamVjdGlvblIFb3JkZXISUgoPcGF5bWVudF9hdHRlbXB0GAIgASgLMikuYW55dHR5'
    'LmNsb3VkLnYxLlBheW1lbnRBdHRlbXB0UHJvamVjdGlvblIOcGF5bWVudEF0dGVtcHQ=');

@$core.Deprecated('Use applyPaymentEventRequestDescriptor instead')
const ApplyPaymentEventRequest$json = {
  '1': 'ApplyPaymentEventRequest',
  '2': [
    {'1': 'provider', '3': 1, '4': 1, '5': 9, '10': 'provider'},
    {'1': 'provider_event_id', '3': 2, '4': 1, '5': 9, '10': 'providerEventId'},
    {
      '1': 'payment_attempt_id',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'paymentAttemptId'
    },
    {'1': 'order_id', '3': 4, '4': 1, '5': 9, '10': 'orderId'},
    {
      '1': 'event_type',
      '3': 5,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.PaymentEventType',
      '10': 'eventType'
    },
    {
      '1': 'provider_reference',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'providerReference'
    },
    {
      '1': 'occurred_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'occurredAt'
    },
  ],
};

/// Descriptor for `ApplyPaymentEventRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List applyPaymentEventRequestDescriptor = $convert.base64Decode(
    'ChhBcHBseVBheW1lbnRFdmVudFJlcXVlc3QSGgoIcHJvdmlkZXIYASABKAlSCHByb3ZpZGVyEi'
    'oKEXByb3ZpZGVyX2V2ZW50X2lkGAIgASgJUg9wcm92aWRlckV2ZW50SWQSLAoScGF5bWVudF9h'
    'dHRlbXB0X2lkGAMgASgJUhBwYXltZW50QXR0ZW1wdElkEhkKCG9yZGVyX2lkGAQgASgJUgdvcm'
    'RlcklkEkAKCmV2ZW50X3R5cGUYBSABKA4yIS5hbnl0dHkuY2xvdWQudjEuUGF5bWVudEV2ZW50'
    'VHlwZVIJZXZlbnRUeXBlEi0KEnByb3ZpZGVyX3JlZmVyZW5jZRgGIAEoCVIRcHJvdmlkZXJSZW'
    'ZlcmVuY2USOwoLb2NjdXJyZWRfYXQYByABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1w'
    'UgpvY2N1cnJlZEF0');

@$core.Deprecated('Use applyPaymentEventResponseDescriptor instead')
const ApplyPaymentEventResponse$json = {
  '1': 'ApplyPaymentEventResponse',
  '2': [
    {
      '1': 'order',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.OrderProjection',
      '10': 'order'
    },
    {
      '1': 'payment_attempt',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.PaymentAttemptProjection',
      '10': 'paymentAttempt'
    },
    {
      '1': 'subscription',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SubscriptionProjection',
      '10': 'subscription'
    },
    {
      '1': 'entitlement',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EffectiveEntitlement',
      '10': 'entitlement'
    },
    {'1': 'duplicate', '3': 5, '4': 1, '5': 8, '10': 'duplicate'},
  ],
};

/// Descriptor for `ApplyPaymentEventResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List applyPaymentEventResponseDescriptor = $convert.base64Decode(
    'ChlBcHBseVBheW1lbnRFdmVudFJlc3BvbnNlEjYKBW9yZGVyGAEgASgLMiAuYW55dHR5LmNsb3'
    'VkLnYxLk9yZGVyUHJvamVjdGlvblIFb3JkZXISUgoPcGF5bWVudF9hdHRlbXB0GAIgASgLMiku'
    'YW55dHR5LmNsb3VkLnYxLlBheW1lbnRBdHRlbXB0UHJvamVjdGlvblIOcGF5bWVudEF0dGVtcH'
    'QSSwoMc3Vic2NyaXB0aW9uGAMgASgLMicuYW55dHR5LmNsb3VkLnYxLlN1YnNjcmlwdGlvblBy'
    'b2plY3Rpb25SDHN1YnNjcmlwdGlvbhJHCgtlbnRpdGxlbWVudBgEIAEoCzIlLmFueXR0eS5jbG'
    '91ZC52MS5FZmZlY3RpdmVFbnRpdGxlbWVudFILZW50aXRsZW1lbnQSHAoJZHVwbGljYXRlGAUg'
    'ASgIUglkdXBsaWNhdGU=');

@$core.Deprecated('Use transitionSubscriptionRequestDescriptor instead')
const TransitionSubscriptionRequest$json = {
  '1': 'TransitionSubscriptionRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {
      '1': 'transition',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SubscriptionTransition',
      '10': 'transition'
    },
    {'1': 'target_plan_id', '3': 3, '4': 1, '5': 9, '10': 'targetPlanId'},
    {
      '1': 'target_plan_version',
      '3': 4,
      '4': 1,
      '5': 4,
      '10': 'targetPlanVersion'
    },
    {
      '1': 'expected_revision',
      '3': 5,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
    {'1': 'reason', '3': 6, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `TransitionSubscriptionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transitionSubscriptionRequestDescriptor = $convert.base64Decode(
    'Ch1UcmFuc2l0aW9uU3Vic2NyaXB0aW9uUmVxdWVzdBIdCgphY2NvdW50X2lkGAEgASgJUglhY2'
    'NvdW50SWQSRwoKdHJhbnNpdGlvbhgCIAEoDjInLmFueXR0eS5jbG91ZC52MS5TdWJzY3JpcHRp'
    'b25UcmFuc2l0aW9uUgp0cmFuc2l0aW9uEiQKDnRhcmdldF9wbGFuX2lkGAMgASgJUgx0YXJnZX'
    'RQbGFuSWQSLgoTdGFyZ2V0X3BsYW5fdmVyc2lvbhgEIAEoBFIRdGFyZ2V0UGxhblZlcnNpb24S'
    'KwoRZXhwZWN0ZWRfcmV2aXNpb24YBSABKARSEGV4cGVjdGVkUmV2aXNpb24SFgoGcmVhc29uGA'
    'YgASgJUgZyZWFzb24=');

@$core.Deprecated('Use transitionSubscriptionResponseDescriptor instead')
const TransitionSubscriptionResponse$json = {
  '1': 'TransitionSubscriptionResponse',
  '2': [
    {
      '1': 'subscription',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SubscriptionProjection',
      '10': 'subscription'
    },
    {
      '1': 'entitlement',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EffectiveEntitlement',
      '10': 'entitlement'
    },
  ],
};

/// Descriptor for `TransitionSubscriptionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transitionSubscriptionResponseDescriptor =
    $convert.base64Decode(
        'Ch5UcmFuc2l0aW9uU3Vic2NyaXB0aW9uUmVzcG9uc2USSwoMc3Vic2NyaXB0aW9uGAEgASgLMi'
        'cuYW55dHR5LmNsb3VkLnYxLlN1YnNjcmlwdGlvblByb2plY3Rpb25SDHN1YnNjcmlwdGlvbhJH'
        'CgtlbnRpdGxlbWVudBgCIAEoCzIlLmFueXR0eS5jbG91ZC52MS5FZmZlY3RpdmVFbnRpdGxlbW'
        'VudFILZW50aXRsZW1lbnQ=');

@$core.Deprecated('Use getAccountCommerceRequestDescriptor instead')
const GetAccountCommerceRequest$json = {
  '1': 'GetAccountCommerceRequest',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
  ],
};

/// Descriptor for `GetAccountCommerceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getAccountCommerceRequestDescriptor =
    $convert.base64Decode(
        'ChlHZXRBY2NvdW50Q29tbWVyY2VSZXF1ZXN0Eh0KCmFjY291bnRfaWQYASABKAlSCWFjY291bn'
        'RJZA==');

@$core.Deprecated('Use getAccountCommerceResponseDescriptor instead')
const GetAccountCommerceResponse$json = {
  '1': 'GetAccountCommerceResponse',
  '2': [
    {
      '1': 'subscription',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.SubscriptionProjection',
      '10': 'subscription'
    },
    {
      '1': 'entitlement',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.EffectiveEntitlement',
      '10': 'entitlement'
    },
    {
      '1': 'orders',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.OrderProjection',
      '10': 'orders'
    },
    {
      '1': 'payment_attempts',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.PaymentAttemptProjection',
      '10': 'paymentAttempts'
    },
    {
      '1': 'usage',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.UsagePeriodProjection',
      '10': 'usage'
    },
  ],
};

/// Descriptor for `GetAccountCommerceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getAccountCommerceResponseDescriptor = $convert.base64Decode(
    'ChpHZXRBY2NvdW50Q29tbWVyY2VSZXNwb25zZRJLCgxzdWJzY3JpcHRpb24YASABKAsyJy5hbn'
    'l0dHkuY2xvdWQudjEuU3Vic2NyaXB0aW9uUHJvamVjdGlvblIMc3Vic2NyaXB0aW9uEkcKC2Vu'
    'dGl0bGVtZW50GAIgASgLMiUuYW55dHR5LmNsb3VkLnYxLkVmZmVjdGl2ZUVudGl0bGVtZW50Ug'
    'tlbnRpdGxlbWVudBI4CgZvcmRlcnMYAyADKAsyIC5hbnl0dHkuY2xvdWQudjEuT3JkZXJQcm9q'
    'ZWN0aW9uUgZvcmRlcnMSVAoQcGF5bWVudF9hdHRlbXB0cxgEIAMoCzIpLmFueXR0eS5jbG91ZC'
    '52MS5QYXltZW50QXR0ZW1wdFByb2plY3Rpb25SD3BheW1lbnRBdHRlbXB0cxI8CgV1c2FnZRgF'
    'IAEoCzImLmFueXR0eS5jbG91ZC52MS5Vc2FnZVBlcmlvZFByb2plY3Rpb25SBXVzYWdl');

@$core.Deprecated('Use createMyOrderRequestDescriptor instead')
const CreateMyOrderRequest$json = {
  '1': 'CreateMyOrderRequest',
  '2': [
    {'1': 'plan_id', '3': 1, '4': 1, '5': 9, '10': 'planId'},
    {'1': 'plan_version', '3': 2, '4': 1, '5': 4, '10': 'planVersion'},
    {'1': 'idempotency_key', '3': 3, '4': 1, '5': 9, '10': 'idempotencyKey'},
    {
      '1': 'requested_transition',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SubscriptionTransition',
      '10': 'requestedTransition'
    },
    {'1': 'yearly', '3': 5, '4': 1, '5': 8, '10': 'yearly'},
  ],
};

/// Descriptor for `CreateMyOrderRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createMyOrderRequestDescriptor = $convert.base64Decode(
    'ChRDcmVhdGVNeU9yZGVyUmVxdWVzdBIXCgdwbGFuX2lkGAEgASgJUgZwbGFuSWQSIQoMcGxhbl'
    '92ZXJzaW9uGAIgASgEUgtwbGFuVmVyc2lvbhInCg9pZGVtcG90ZW5jeV9rZXkYAyABKAlSDmlk'
    'ZW1wb3RlbmN5S2V5EloKFHJlcXVlc3RlZF90cmFuc2l0aW9uGAQgASgOMicuYW55dHR5LmNsb3'
    'VkLnYxLlN1YnNjcmlwdGlvblRyYW5zaXRpb25SE3JlcXVlc3RlZFRyYW5zaXRpb24SFgoGeWVh'
    'cmx5GAUgASgIUgZ5ZWFybHk=');

@$core.Deprecated('Use getMyCommerceRequestDescriptor instead')
const GetMyCommerceRequest$json = {
  '1': 'GetMyCommerceRequest',
};

/// Descriptor for `GetMyCommerceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getMyCommerceRequestDescriptor =
    $convert.base64Decode('ChRHZXRNeUNvbW1lcmNlUmVxdWVzdA==');

@$core.Deprecated('Use changeMySubscriptionRequestDescriptor instead')
const ChangeMySubscriptionRequest$json = {
  '1': 'ChangeMySubscriptionRequest',
  '2': [
    {
      '1': 'transition',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.SubscriptionTransition',
      '10': 'transition'
    },
    {
      '1': 'expected_revision',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'expectedRevision'
    },
  ],
};

/// Descriptor for `ChangeMySubscriptionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeMySubscriptionRequestDescriptor =
    $convert.base64Decode(
        'ChtDaGFuZ2VNeVN1YnNjcmlwdGlvblJlcXVlc3QSRwoKdHJhbnNpdGlvbhgBIAEoDjInLmFueX'
        'R0eS5jbG91ZC52MS5TdWJzY3JpcHRpb25UcmFuc2l0aW9uUgp0cmFuc2l0aW9uEisKEWV4cGVj'
        'dGVkX3JldmlzaW9uGAIgASgEUhBleHBlY3RlZFJldmlzaW9u');

@$core.Deprecated('Use completeDevelopmentPaymentRequestDescriptor instead')
const CompleteDevelopmentPaymentRequest$json = {
  '1': 'CompleteDevelopmentPaymentRequest',
  '2': [
    {'1': 'order_id', '3': 1, '4': 1, '5': 9, '10': 'orderId'},
    {
      '1': 'payment_attempt_id',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'paymentAttemptId'
    },
  ],
};

/// Descriptor for `CompleteDevelopmentPaymentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List completeDevelopmentPaymentRequestDescriptor =
    $convert.base64Decode(
        'CiFDb21wbGV0ZURldmVsb3BtZW50UGF5bWVudFJlcXVlc3QSGQoIb3JkZXJfaWQYASABKAlSB2'
        '9yZGVySWQSLAoScGF5bWVudF9hdHRlbXB0X2lkGAIgASgJUhBwYXltZW50QXR0ZW1wdElk');

const $core.Map<$core.String, $core.dynamic> CommerceServiceBase$json = {
  '1': 'CommerceService',
  '2': [
    {
      '1': 'ListPlans',
      '2': '.anytty.cloud.v1.ListPlansRequest',
      '3': '.anytty.cloud.v1.ListPlansResponse'
    },
    {
      '1': 'CreatePlanVersion',
      '2': '.anytty.cloud.v1.CreatePlanVersionRequest',
      '3': '.anytty.cloud.v1.CreatePlanVersionResponse'
    },
    {
      '1': 'PublishPlanVersion',
      '2': '.anytty.cloud.v1.PublishPlanVersionRequest',
      '3': '.anytty.cloud.v1.PublishPlanVersionResponse'
    },
    {
      '1': 'CreateOrder',
      '2': '.anytty.cloud.v1.CreateOrderRequest',
      '3': '.anytty.cloud.v1.CreateOrderResponse'
    },
    {
      '1': 'ApplyPaymentEvent',
      '2': '.anytty.cloud.v1.ApplyPaymentEventRequest',
      '3': '.anytty.cloud.v1.ApplyPaymentEventResponse'
    },
    {
      '1': 'TransitionSubscription',
      '2': '.anytty.cloud.v1.TransitionSubscriptionRequest',
      '3': '.anytty.cloud.v1.TransitionSubscriptionResponse'
    },
    {
      '1': 'GetAccountCommerce',
      '2': '.anytty.cloud.v1.GetAccountCommerceRequest',
      '3': '.anytty.cloud.v1.GetAccountCommerceResponse'
    },
    {
      '1': 'CreateMyOrder',
      '2': '.anytty.cloud.v1.CreateMyOrderRequest',
      '3': '.anytty.cloud.v1.CreateOrderResponse'
    },
    {
      '1': 'GetMyCommerce',
      '2': '.anytty.cloud.v1.GetMyCommerceRequest',
      '3': '.anytty.cloud.v1.GetAccountCommerceResponse'
    },
    {
      '1': 'ChangeMySubscription',
      '2': '.anytty.cloud.v1.ChangeMySubscriptionRequest',
      '3': '.anytty.cloud.v1.TransitionSubscriptionResponse'
    },
    {
      '1': 'CompleteDevelopmentPayment',
      '2': '.anytty.cloud.v1.CompleteDevelopmentPaymentRequest',
      '3': '.anytty.cloud.v1.ApplyPaymentEventResponse'
    },
  ],
};

@$core.Deprecated('Use commerceServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    CommerceServiceBase$messageJson = {
  '.anytty.cloud.v1.ListPlansRequest': ListPlansRequest$json,
  '.anytty.cloud.v1.ListPlansResponse': ListPlansResponse$json,
  '.anytty.cloud.v1.PlanDefinition': PlanDefinition$json,
  '.anytty.cloud.v1.Money': Money$json,
  '.anytty.cloud.v1.CloudCapability': CloudCapability$json,
  '.google.protobuf.Timestamp': $0.Timestamp$json,
  '.anytty.cloud.v1.CreatePlanVersionRequest': CreatePlanVersionRequest$json,
  '.anytty.cloud.v1.CreatePlanVersionResponse': CreatePlanVersionResponse$json,
  '.anytty.cloud.v1.PublishPlanVersionRequest': PublishPlanVersionRequest$json,
  '.anytty.cloud.v1.PublishPlanVersionResponse':
      PublishPlanVersionResponse$json,
  '.anytty.cloud.v1.CreateOrderRequest': CreateOrderRequest$json,
  '.anytty.cloud.v1.CreateOrderResponse': CreateOrderResponse$json,
  '.anytty.cloud.v1.OrderProjection': OrderProjection$json,
  '.anytty.cloud.v1.PaymentAttemptProjection': PaymentAttemptProjection$json,
  '.anytty.cloud.v1.ApplyPaymentEventRequest': ApplyPaymentEventRequest$json,
  '.anytty.cloud.v1.ApplyPaymentEventResponse': ApplyPaymentEventResponse$json,
  '.anytty.cloud.v1.SubscriptionProjection': SubscriptionProjection$json,
  '.anytty.cloud.v1.EffectiveEntitlement': EffectiveEntitlement$json,
  '.anytty.cloud.v1.TransitionSubscriptionRequest':
      TransitionSubscriptionRequest$json,
  '.anytty.cloud.v1.TransitionSubscriptionResponse':
      TransitionSubscriptionResponse$json,
  '.anytty.cloud.v1.GetAccountCommerceRequest': GetAccountCommerceRequest$json,
  '.anytty.cloud.v1.GetAccountCommerceResponse':
      GetAccountCommerceResponse$json,
  '.anytty.cloud.v1.UsagePeriodProjection': UsagePeriodProjection$json,
  '.anytty.cloud.v1.CreateMyOrderRequest': CreateMyOrderRequest$json,
  '.anytty.cloud.v1.GetMyCommerceRequest': GetMyCommerceRequest$json,
  '.anytty.cloud.v1.ChangeMySubscriptionRequest':
      ChangeMySubscriptionRequest$json,
  '.anytty.cloud.v1.CompleteDevelopmentPaymentRequest':
      CompleteDevelopmentPaymentRequest$json,
};

/// Descriptor for `CommerceService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List commerceServiceDescriptor = $convert.base64Decode(
    'Cg9Db21tZXJjZVNlcnZpY2USUgoJTGlzdFBsYW5zEiEuYW55dHR5LmNsb3VkLnYxLkxpc3RQbG'
    'Fuc1JlcXVlc3QaIi5hbnl0dHkuY2xvdWQudjEuTGlzdFBsYW5zUmVzcG9uc2USagoRQ3JlYXRl'
    'UGxhblZlcnNpb24SKS5hbnl0dHkuY2xvdWQudjEuQ3JlYXRlUGxhblZlcnNpb25SZXF1ZXN0Gi'
    'ouYW55dHR5LmNsb3VkLnYxLkNyZWF0ZVBsYW5WZXJzaW9uUmVzcG9uc2USbQoSUHVibGlzaFBs'
    'YW5WZXJzaW9uEiouYW55dHR5LmNsb3VkLnYxLlB1Ymxpc2hQbGFuVmVyc2lvblJlcXVlc3QaKy'
    '5hbnl0dHkuY2xvdWQudjEuUHVibGlzaFBsYW5WZXJzaW9uUmVzcG9uc2USWAoLQ3JlYXRlT3Jk'
    'ZXISIy5hbnl0dHkuY2xvdWQudjEuQ3JlYXRlT3JkZXJSZXF1ZXN0GiQuYW55dHR5LmNsb3VkLn'
    'YxLkNyZWF0ZU9yZGVyUmVzcG9uc2USagoRQXBwbHlQYXltZW50RXZlbnQSKS5hbnl0dHkuY2xv'
    'dWQudjEuQXBwbHlQYXltZW50RXZlbnRSZXF1ZXN0GiouYW55dHR5LmNsb3VkLnYxLkFwcGx5UG'
    'F5bWVudEV2ZW50UmVzcG9uc2USeQoWVHJhbnNpdGlvblN1YnNjcmlwdGlvbhIuLmFueXR0eS5j'
    'bG91ZC52MS5UcmFuc2l0aW9uU3Vic2NyaXB0aW9uUmVxdWVzdBovLmFueXR0eS5jbG91ZC52MS'
    '5UcmFuc2l0aW9uU3Vic2NyaXB0aW9uUmVzcG9uc2USbQoSR2V0QWNjb3VudENvbW1lcmNlEiou'
    'YW55dHR5LmNsb3VkLnYxLkdldEFjY291bnRDb21tZXJjZVJlcXVlc3QaKy5hbnl0dHkuY2xvdW'
    'QudjEuR2V0QWNjb3VudENvbW1lcmNlUmVzcG9uc2USXAoNQ3JlYXRlTXlPcmRlchIlLmFueXR0'
    'eS5jbG91ZC52MS5DcmVhdGVNeU9yZGVyUmVxdWVzdBokLmFueXR0eS5jbG91ZC52MS5DcmVhdG'
    'VPcmRlclJlc3BvbnNlEmMKDUdldE15Q29tbWVyY2USJS5hbnl0dHkuY2xvdWQudjEuR2V0TXlD'
    'b21tZXJjZVJlcXVlc3QaKy5hbnl0dHkuY2xvdWQudjEuR2V0QWNjb3VudENvbW1lcmNlUmVzcG'
    '9uc2USdQoUQ2hhbmdlTXlTdWJzY3JpcHRpb24SLC5hbnl0dHkuY2xvdWQudjEuQ2hhbmdlTXlT'
    'dWJzY3JpcHRpb25SZXF1ZXN0Gi8uYW55dHR5LmNsb3VkLnYxLlRyYW5zaXRpb25TdWJzY3JpcH'
    'Rpb25SZXNwb25zZRJ8ChpDb21wbGV0ZURldmVsb3BtZW50UGF5bWVudBIyLmFueXR0eS5jbG91'
    'ZC52MS5Db21wbGV0ZURldmVsb3BtZW50UGF5bWVudFJlcXVlc3QaKi5hbnl0dHkuY2xvdWQudj'
    'EuQXBwbHlQYXltZW50RXZlbnRSZXNwb25zZQ==');
