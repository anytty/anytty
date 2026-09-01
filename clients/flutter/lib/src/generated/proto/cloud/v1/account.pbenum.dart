// This is a generated file - do not edit.
//
// Generated from cloud/v1/account.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// AccountState 是账号持久状态；只有 active 账号可以登录或持有 refresh token。
class AccountState extends $pb.ProtobufEnum {
  static const AccountState ACCOUNT_STATE_UNSPECIFIED =
      AccountState._(0, _omitEnumNames ? '' : 'ACCOUNT_STATE_UNSPECIFIED');
  static const AccountState ACCOUNT_STATE_PENDING =
      AccountState._(1, _omitEnumNames ? '' : 'ACCOUNT_STATE_PENDING');
  static const AccountState ACCOUNT_STATE_ACTIVE =
      AccountState._(2, _omitEnumNames ? '' : 'ACCOUNT_STATE_ACTIVE');
  static const AccountState ACCOUNT_STATE_DISABLED =
      AccountState._(3, _omitEnumNames ? '' : 'ACCOUNT_STATE_DISABLED');

  static const $core.List<AccountState> values = <AccountState>[
    ACCOUNT_STATE_UNSPECIFIED,
    ACCOUNT_STATE_PENDING,
    ACCOUNT_STATE_ACTIVE,
    ACCOUNT_STATE_DISABLED,
  ];

  static final $core.List<AccountState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static AccountState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const AccountState._(super.value, super.name);
}

/// AccountRole 是 Controller RBAC 的稳定角色，不由浏览器自行推断权限。
class AccountRole extends $pb.ProtobufEnum {
  static const AccountRole ACCOUNT_ROLE_UNSPECIFIED =
      AccountRole._(0, _omitEnumNames ? '' : 'ACCOUNT_ROLE_UNSPECIFIED');
  static const AccountRole ACCOUNT_ROLE_USER =
      AccountRole._(1, _omitEnumNames ? '' : 'ACCOUNT_ROLE_USER');
  static const AccountRole ACCOUNT_ROLE_OPERATOR =
      AccountRole._(2, _omitEnumNames ? '' : 'ACCOUNT_ROLE_OPERATOR');
  static const AccountRole ACCOUNT_ROLE_ADMIN =
      AccountRole._(3, _omitEnumNames ? '' : 'ACCOUNT_ROLE_ADMIN');

  static const $core.List<AccountRole> values = <AccountRole>[
    ACCOUNT_ROLE_UNSPECIFIED,
    ACCOUNT_ROLE_USER,
    ACCOUNT_ROLE_OPERATOR,
    ACCOUNT_ROLE_ADMIN,
  ];

  static final $core.List<AccountRole?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static AccountRole? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const AccountRole._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
