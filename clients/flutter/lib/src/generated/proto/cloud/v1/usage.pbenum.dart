// This is a generated file - do not edit.
//
// Generated from cloud/v1/usage.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// RelayPreference is the ICE policy for one Cloud connection attempt.
class RelayPreference extends $pb.ProtobufEnum {
  static const RelayPreference RELAY_PREFERENCE_UNSPECIFIED = RelayPreference._(
      0, _omitEnumNames ? '' : 'RELAY_PREFERENCE_UNSPECIFIED');
  static const RelayPreference RELAY_PREFERENCE_AUTO =
      RelayPreference._(1, _omitEnumNames ? '' : 'RELAY_PREFERENCE_AUTO');
  static const RelayPreference RELAY_PREFERENCE_DIRECT_ONLY = RelayPreference._(
      2, _omitEnumNames ? '' : 'RELAY_PREFERENCE_DIRECT_ONLY');
  static const RelayPreference RELAY_PREFERENCE_RELAY_ONLY =
      RelayPreference._(3, _omitEnumNames ? '' : 'RELAY_PREFERENCE_RELAY_ONLY');

  static const $core.List<RelayPreference> values = <RelayPreference>[
    RELAY_PREFERENCE_UNSPECIFIED,
    RELAY_PREFERENCE_AUTO,
    RELAY_PREFERENCE_DIRECT_ONLY,
    RELAY_PREFERENCE_RELAY_ONLY,
  ];

  static final $core.List<RelayPreference?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static RelayPreference? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RelayPreference._(super.value, super.name);
}

/// RelayTransport is the physical TURN transport used by an Edge allocation.
class RelayTransport extends $pb.ProtobufEnum {
  static const RelayTransport RELAY_TRANSPORT_UNSPECIFIED =
      RelayTransport._(0, _omitEnumNames ? '' : 'RELAY_TRANSPORT_UNSPECIFIED');
  static const RelayTransport RELAY_TRANSPORT_UDP =
      RelayTransport._(1, _omitEnumNames ? '' : 'RELAY_TRANSPORT_UDP');
  static const RelayTransport RELAY_TRANSPORT_TCP =
      RelayTransport._(2, _omitEnumNames ? '' : 'RELAY_TRANSPORT_TCP');
  static const RelayTransport RELAY_TRANSPORT_TLS =
      RelayTransport._(3, _omitEnumNames ? '' : 'RELAY_TRANSPORT_TLS');

  static const $core.List<RelayTransport> values = <RelayTransport>[
    RELAY_TRANSPORT_UNSPECIFIED,
    RELAY_TRANSPORT_UDP,
    RELAY_TRANSPORT_TCP,
    RELAY_TRANSPORT_TLS,
  ];

  static final $core.List<RelayTransport?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static RelayTransport? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RelayTransport._(super.value, super.name);
}

class RelaySettlementKind extends $pb.ProtobufEnum {
  static const RelaySettlementKind RELAY_SETTLEMENT_KIND_UNSPECIFIED =
      RelaySettlementKind._(
          0, _omitEnumNames ? '' : 'RELAY_SETTLEMENT_KIND_UNSPECIFIED');
  static const RelaySettlementKind RELAY_SETTLEMENT_KIND_EXACT =
      RelaySettlementKind._(
          1, _omitEnumNames ? '' : 'RELAY_SETTLEMENT_KIND_EXACT');
  static const RelaySettlementKind RELAY_SETTLEMENT_KIND_RECOVERY_MAX =
      RelaySettlementKind._(
          2, _omitEnumNames ? '' : 'RELAY_SETTLEMENT_KIND_RECOVERY_MAX');

  static const $core.List<RelaySettlementKind> values = <RelaySettlementKind>[
    RELAY_SETTLEMENT_KIND_UNSPECIFIED,
    RELAY_SETTLEMENT_KIND_EXACT,
    RELAY_SETTLEMENT_KIND_RECOVERY_MAX,
  ];

  static final $core.List<RelaySettlementKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static RelaySettlementKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RelaySettlementKind._(super.value, super.name);
}

class RelayResponseCode extends $pb.ProtobufEnum {
  static const RelayResponseCode RELAY_RESPONSE_CODE_UNSPECIFIED =
      RelayResponseCode._(
          0, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_UNSPECIFIED');
  static const RelayResponseCode RELAY_RESPONSE_CODE_APPLIED =
      RelayResponseCode._(
          1, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_APPLIED');
  static const RelayResponseCode RELAY_RESPONSE_CODE_REPLAY =
      RelayResponseCode._(
          2, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_REPLAY');
  static const RelayResponseCode RELAY_RESPONSE_CODE_TERMINAL =
      RelayResponseCode._(
          3, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_TERMINAL');
  static const RelayResponseCode RELAY_RESPONSE_CODE_REJECTED =
      RelayResponseCode._(
          4, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_REJECTED');
  static const RelayResponseCode RELAY_RESPONSE_CODE_CONFLICT =
      RelayResponseCode._(
          5, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_CONFLICT');
  static const RelayResponseCode RELAY_RESPONSE_CODE_UNAVAILABLE =
      RelayResponseCode._(
          6, _omitEnumNames ? '' : 'RELAY_RESPONSE_CODE_UNAVAILABLE');

  static const $core.List<RelayResponseCode> values = <RelayResponseCode>[
    RELAY_RESPONSE_CODE_UNSPECIFIED,
    RELAY_RESPONSE_CODE_APPLIED,
    RELAY_RESPONSE_CODE_REPLAY,
    RELAY_RESPONSE_CODE_TERMINAL,
    RELAY_RESPONSE_CODE_REJECTED,
    RELAY_RESPONSE_CODE_CONFLICT,
    RELAY_RESPONSE_CODE_UNAVAILABLE,
  ];

  static final $core.List<RelayResponseCode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static RelayResponseCode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RelayResponseCode._(super.value, super.name);
}

class RelayAccountActionType extends $pb.ProtobufEnum {
  static const RelayAccountActionType RELAY_ACCOUNT_ACTION_TYPE_UNSPECIFIED =
      RelayAccountActionType._(
          0, _omitEnumNames ? '' : 'RELAY_ACCOUNT_ACTION_TYPE_UNSPECIFIED');
  static const RelayAccountActionType RELAY_ACCOUNT_ACTION_TYPE_ALLOW =
      RelayAccountActionType._(
          1, _omitEnumNames ? '' : 'RELAY_ACCOUNT_ACTION_TYPE_ALLOW');
  static const RelayAccountActionType RELAY_ACCOUNT_ACTION_TYPE_DENY_AND_CLOSE =
      RelayAccountActionType._(
          2, _omitEnumNames ? '' : 'RELAY_ACCOUNT_ACTION_TYPE_DENY_AND_CLOSE');

  static const $core.List<RelayAccountActionType> values =
      <RelayAccountActionType>[
    RELAY_ACCOUNT_ACTION_TYPE_UNSPECIFIED,
    RELAY_ACCOUNT_ACTION_TYPE_ALLOW,
    RELAY_ACCOUNT_ACTION_TYPE_DENY_AND_CLOSE,
  ];

  static final $core.List<RelayAccountActionType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static RelayAccountActionType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RelayAccountActionType._(super.value, super.name);
}

class RelayJournalStage extends $pb.ProtobufEnum {
  static const RelayJournalStage RELAY_JOURNAL_STAGE_UNSPECIFIED =
      RelayJournalStage._(
          0, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_UNSPECIFIED');
  static const RelayJournalStage RELAY_JOURNAL_STAGE_REQUESTED =
      RelayJournalStage._(
          1, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_REQUESTED');
  static const RelayJournalStage RELAY_JOURNAL_STAGE_HELD_UNEXPOSED =
      RelayJournalStage._(
          2, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_HELD_UNEXPOSED');
  static const RelayJournalStage RELAY_JOURNAL_STAGE_EXPOSED =
      RelayJournalStage._(
          3, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_EXPOSED');
  static const RelayJournalStage RELAY_JOURNAL_STAGE_RENEW_PENDING =
      RelayJournalStage._(
          4, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_RENEW_PENDING');
  static const RelayJournalStage RELAY_JOURNAL_STAGE_CLOSING =
      RelayJournalStage._(
          5, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_CLOSING');
  static const RelayJournalStage RELAY_JOURNAL_STAGE_SETTLEMENT_DURABLE =
      RelayJournalStage._(
          6, _omitEnumNames ? '' : 'RELAY_JOURNAL_STAGE_SETTLEMENT_DURABLE');

  static const $core.List<RelayJournalStage> values = <RelayJournalStage>[
    RELAY_JOURNAL_STAGE_UNSPECIFIED,
    RELAY_JOURNAL_STAGE_REQUESTED,
    RELAY_JOURNAL_STAGE_HELD_UNEXPOSED,
    RELAY_JOURNAL_STAGE_EXPOSED,
    RELAY_JOURNAL_STAGE_RENEW_PENDING,
    RELAY_JOURNAL_STAGE_CLOSING,
    RELAY_JOURNAL_STAGE_SETTLEMENT_DURABLE,
  ];

  static final $core.List<RelayJournalStage?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static RelayJournalStage? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const RelayJournalStage._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
