// This is a generated file - do not edit.
//
// Generated from cloud/v1/account.proto.

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

import 'account.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'account.pbenum.dart';

/// AccountProfile 是账号持久投影，不包含密码 verifier 或 token secret。
class AccountProfile extends $pb.GeneratedMessage {
  factory AccountProfile({
    $core.String? accountId,
    $core.String? email,
    $core.String? displayName,
    AccountState? state,
    $fixnum.Int64? revision,
    $0.Timestamp? createdAt,
    $0.Timestamp? updatedAt,
    $core.bool? emailVerified,
  }) {
    final result = AccountProfile._();
    if (accountId != null) result.accountId = accountId;
    if (email != null) result.email = email;
    if (displayName != null) result.displayName = displayName;
    if (state != null) result.state = state;
    if (revision != null) result.revision = revision;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (emailVerified != null) result.emailVerified = emailVerified;
    return result;
  }

  AccountProfile._();

  factory AccountProfile.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AccountProfile()..mergeFromBuffer(data, registry);
  factory AccountProfile.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AccountProfile()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AccountProfile',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: AccountProfile.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'accountId')
    ..aOS(2, _omitFieldNames ? '' : 'email')
    ..aOS(3, _omitFieldNames ? '' : 'displayName')
    ..aE<AccountState>(4, _omitFieldNames ? '' : 'state',
        enumValues: AccountState.values)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.Timestamp>(6, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..aOM<$0.Timestamp>(7, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..aOB(8, _omitFieldNames ? '' : 'emailVerified')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountProfile clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountProfile copyWith(void Function(AccountProfile) updates) =>
      super.copyWith((message) => updates(message as AccountProfile))
          as AccountProfile;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use AccountProfile() / AccountProfile.new instead')
  static AccountProfile create() => AccountProfile._();
  static $pb.GeneratedMessage $_createMessage() => AccountProfile._();
  @$core.override
  AccountProfile createEmptyInstance() => AccountProfile._();
  @$core.pragma('dart2js:noInline')
  static AccountProfile getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<AccountProfile>(
          AccountProfile.$_createMessage);
  static AccountProfile? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get accountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set accountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get email => $_getSZ(1);
  @$pb.TagNumber(2)
  set email($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEmail() => $_has(1);
  @$pb.TagNumber(2)
  void clearEmail() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get displayName => $_getSZ(2);
  @$pb.TagNumber(3)
  set displayName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDisplayName() => $_has(2);
  @$pb.TagNumber(3)
  void clearDisplayName() => $_clearField(3);

  @$pb.TagNumber(4)
  AccountState get state => $_getN(3);
  @$pb.TagNumber(4)
  set state(AccountState value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasState() => $_has(3);
  @$pb.TagNumber(4)
  void clearState() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get revision => $_getI64(4);
  @$pb.TagNumber(5)
  set revision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRevision() => $_has(4);
  @$pb.TagNumber(5)
  void clearRevision() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.Timestamp get createdAt => $_getN(5);
  @$pb.TagNumber(6)
  set createdAt($0.Timestamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCreatedAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearCreatedAt() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.Timestamp ensureCreatedAt() => $_ensure(5);

  @$pb.TagNumber(7)
  $0.Timestamp get updatedAt => $_getN(6);
  @$pb.TagNumber(7)
  set updatedAt($0.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasUpdatedAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearUpdatedAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Timestamp ensureUpdatedAt() => $_ensure(6);

  @$pb.TagNumber(8)
  $core.bool get emailVerified => $_getBF(7);
  @$pb.TagNumber(8)
  set emailVerified($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasEmailVerified() => $_has(7);
  @$pb.TagNumber(8)
  void clearEmailVerified() => $_clearField(8);
}

/// AccountTokenCredential 只在登录或轮换 refresh token 时返回原始 token。
class AccountTokenCredential extends $pb.GeneratedMessage {
  factory AccountTokenCredential({
    $core.String? refreshId,
    $core.String? accessToken,
    $core.List<$core.int>? refreshToken,
    $0.Timestamp? accessExpiresAt,
    $0.Timestamp? refreshExpiresAt,
    $core.List<$core.int>? csrfToken,
  }) {
    final result = AccountTokenCredential._();
    if (refreshId != null) result.refreshId = refreshId;
    if (accessToken != null) result.accessToken = accessToken;
    if (refreshToken != null) result.refreshToken = refreshToken;
    if (accessExpiresAt != null) result.accessExpiresAt = accessExpiresAt;
    if (refreshExpiresAt != null) result.refreshExpiresAt = refreshExpiresAt;
    if (csrfToken != null) result.csrfToken = csrfToken;
    return result;
  }

  AccountTokenCredential._();

  factory AccountTokenCredential.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AccountTokenCredential()..mergeFromBuffer(data, registry);
  factory AccountTokenCredential.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AccountTokenCredential()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AccountTokenCredential',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: AccountTokenCredential.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'refreshId')
    ..aOS(2, _omitFieldNames ? '' : 'accessToken')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'refreshToken', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(4, _omitFieldNames ? '' : 'accessExpiresAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..aOM<$0.Timestamp>(5, _omitFieldNames ? '' : 'refreshExpiresAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'csrfToken', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountTokenCredential clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountTokenCredential copyWith(
          void Function(AccountTokenCredential) updates) =>
      super.copyWith((message) => updates(message as AccountTokenCredential))
          as AccountTokenCredential;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use AccountTokenCredential() / AccountTokenCredential.new instead')
  static AccountTokenCredential create() => AccountTokenCredential._();
  static $pb.GeneratedMessage $_createMessage() => AccountTokenCredential._();
  @$core.override
  AccountTokenCredential createEmptyInstance() => AccountTokenCredential._();
  @$core.pragma('dart2js:noInline')
  static AccountTokenCredential getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AccountTokenCredential>(
          AccountTokenCredential.$_createMessage);
  static AccountTokenCredential? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get refreshId => $_getSZ(0);
  @$pb.TagNumber(1)
  set refreshId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRefreshId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRefreshId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get accessToken => $_getSZ(1);
  @$pb.TagNumber(2)
  set accessToken($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAccessToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearAccessToken() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get refreshToken => $_getN(2);
  @$pb.TagNumber(3)
  set refreshToken($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRefreshToken() => $_has(2);
  @$pb.TagNumber(3)
  void clearRefreshToken() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.Timestamp get accessExpiresAt => $_getN(3);
  @$pb.TagNumber(4)
  set accessExpiresAt($0.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasAccessExpiresAt() => $_has(3);
  @$pb.TagNumber(4)
  void clearAccessExpiresAt() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Timestamp ensureAccessExpiresAt() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.Timestamp get refreshExpiresAt => $_getN(4);
  @$pb.TagNumber(5)
  set refreshExpiresAt($0.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasRefreshExpiresAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearRefreshExpiresAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Timestamp ensureRefreshExpiresAt() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.List<$core.int> get csrfToken => $_getN(5);
  @$pb.TagNumber(6)
  set csrfToken($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCsrfToken() => $_has(5);
  @$pb.TagNumber(6)
  void clearCsrfToken() => $_clearField(6);
}

class LoginAccountRequest extends $pb.GeneratedMessage {
  factory LoginAccountRequest({
    $core.String? login,
    $core.String? password,
  }) {
    final result = LoginAccountRequest._();
    if (login != null) result.login = login;
    if (password != null) result.password = password;
    return result;
  }

  LoginAccountRequest._();

  factory LoginAccountRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LoginAccountRequest()..mergeFromBuffer(data, registry);
  factory LoginAccountRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LoginAccountRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LoginAccountRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: LoginAccountRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'login')
    ..aOS(2, _omitFieldNames ? '' : 'password')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginAccountRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginAccountRequest copyWith(void Function(LoginAccountRequest) updates) =>
      super.copyWith((message) => updates(message as LoginAccountRequest))
          as LoginAccountRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use LoginAccountRequest() / LoginAccountRequest.new instead')
  static LoginAccountRequest create() => LoginAccountRequest._();
  static $pb.GeneratedMessage $_createMessage() => LoginAccountRequest._();
  @$core.override
  LoginAccountRequest createEmptyInstance() => LoginAccountRequest._();
  @$core.pragma('dart2js:noInline')
  static LoginAccountRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LoginAccountRequest>(
          LoginAccountRequest.$_createMessage);
  static LoginAccountRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get login => $_getSZ(0);
  @$pb.TagNumber(1)
  set login($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLogin() => $_has(0);
  @$pb.TagNumber(1)
  void clearLogin() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get password => $_getSZ(1);
  @$pb.TagNumber(2)
  set password($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPassword() => $_has(1);
  @$pb.TagNumber(2)
  void clearPassword() => $_clearField(2);
}

class LoginAccountResponse extends $pb.GeneratedMessage {
  factory LoginAccountResponse({
    AccountProfile? account,
    $core.Iterable<AccountRole>? roles,
    AccountTokenCredential? credential,
  }) {
    final result = LoginAccountResponse._();
    if (account != null) result.account = account;
    if (roles != null) result.roles.addAll(roles);
    if (credential != null) result.credential = credential;
    return result;
  }

  LoginAccountResponse._();

  factory LoginAccountResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LoginAccountResponse()..mergeFromBuffer(data, registry);
  factory LoginAccountResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LoginAccountResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LoginAccountResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: LoginAccountResponse.$_createMessage)
    ..aOM<AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: AccountProfile.$_createMessage)
    ..pc<AccountRole>(2, _omitFieldNames ? '' : 'roles', $pb.PbFieldType.KE,
        valueOf: AccountRole.valueOf,
        enumValues: AccountRole.values,
        defaultEnumValue: AccountRole.ACCOUNT_ROLE_UNSPECIFIED)
    ..aOM<AccountTokenCredential>(3, _omitFieldNames ? '' : 'credential',
        subBuilder: AccountTokenCredential.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginAccountResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LoginAccountResponse copyWith(void Function(LoginAccountResponse) updates) =>
      super.copyWith((message) => updates(message as LoginAccountResponse))
          as LoginAccountResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use LoginAccountResponse() / LoginAccountResponse.new instead')
  static LoginAccountResponse create() => LoginAccountResponse._();
  static $pb.GeneratedMessage $_createMessage() => LoginAccountResponse._();
  @$core.override
  LoginAccountResponse createEmptyInstance() => LoginAccountResponse._();
  @$core.pragma('dart2js:noInline')
  static LoginAccountResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LoginAccountResponse>(
          LoginAccountResponse.$_createMessage);
  static LoginAccountResponse? _defaultInstance;

  @$pb.TagNumber(1)
  AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account(AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<AccountRole> get roles => $_getList(1);

  @$pb.TagNumber(3)
  AccountTokenCredential get credential => $_getN(2);
  @$pb.TagNumber(3)
  set credential(AccountTokenCredential value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCredential() => $_has(2);
  @$pb.TagNumber(3)
  void clearCredential() => $_clearField(3);
  @$pb.TagNumber(3)
  AccountTokenCredential ensureCredential() => $_ensure(2);
}

class RefreshAccountTokenRequest extends $pb.GeneratedMessage {
  factory RefreshAccountTokenRequest({
    $core.List<$core.int>? refreshToken,
  }) {
    final result = RefreshAccountTokenRequest._();
    if (refreshToken != null) result.refreshToken = refreshToken;
    return result;
  }

  RefreshAccountTokenRequest._();

  factory RefreshAccountTokenRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RefreshAccountTokenRequest()..mergeFromBuffer(data, registry);
  factory RefreshAccountTokenRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RefreshAccountTokenRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RefreshAccountTokenRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RefreshAccountTokenRequest.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'refreshToken', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RefreshAccountTokenRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RefreshAccountTokenRequest copyWith(
          void Function(RefreshAccountTokenRequest) updates) =>
      super.copyWith(
              (message) => updates(message as RefreshAccountTokenRequest))
          as RefreshAccountTokenRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RefreshAccountTokenRequest() / RefreshAccountTokenRequest.new instead')
  static RefreshAccountTokenRequest create() => RefreshAccountTokenRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      RefreshAccountTokenRequest._();
  @$core.override
  RefreshAccountTokenRequest createEmptyInstance() =>
      RefreshAccountTokenRequest._();
  @$core.pragma('dart2js:noInline')
  static RefreshAccountTokenRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RefreshAccountTokenRequest>(
          RefreshAccountTokenRequest.$_createMessage);
  static RefreshAccountTokenRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get refreshToken => $_getN(0);
  @$pb.TagNumber(1)
  set refreshToken($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRefreshToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearRefreshToken() => $_clearField(1);
}

class RefreshAccountTokenResponse extends $pb.GeneratedMessage {
  factory RefreshAccountTokenResponse({
    AccountProfile? account,
    $core.Iterable<AccountRole>? roles,
    AccountTokenCredential? credential,
  }) {
    final result = RefreshAccountTokenResponse._();
    if (account != null) result.account = account;
    if (roles != null) result.roles.addAll(roles);
    if (credential != null) result.credential = credential;
    return result;
  }

  RefreshAccountTokenResponse._();

  factory RefreshAccountTokenResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RefreshAccountTokenResponse()..mergeFromBuffer(data, registry);
  factory RefreshAccountTokenResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RefreshAccountTokenResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RefreshAccountTokenResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RefreshAccountTokenResponse.$_createMessage)
    ..aOM<AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: AccountProfile.$_createMessage)
    ..pc<AccountRole>(2, _omitFieldNames ? '' : 'roles', $pb.PbFieldType.KE,
        valueOf: AccountRole.valueOf,
        enumValues: AccountRole.values,
        defaultEnumValue: AccountRole.ACCOUNT_ROLE_UNSPECIFIED)
    ..aOM<AccountTokenCredential>(3, _omitFieldNames ? '' : 'credential',
        subBuilder: AccountTokenCredential.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RefreshAccountTokenResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RefreshAccountTokenResponse copyWith(
          void Function(RefreshAccountTokenResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RefreshAccountTokenResponse))
          as RefreshAccountTokenResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RefreshAccountTokenResponse() / RefreshAccountTokenResponse.new instead')
  static RefreshAccountTokenResponse create() =>
      RefreshAccountTokenResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      RefreshAccountTokenResponse._();
  @$core.override
  RefreshAccountTokenResponse createEmptyInstance() =>
      RefreshAccountTokenResponse._();
  @$core.pragma('dart2js:noInline')
  static RefreshAccountTokenResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RefreshAccountTokenResponse>(
          RefreshAccountTokenResponse.$_createMessage);
  static RefreshAccountTokenResponse? _defaultInstance;

  @$pb.TagNumber(1)
  AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account(AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<AccountRole> get roles => $_getList(1);

  @$pb.TagNumber(3)
  AccountTokenCredential get credential => $_getN(2);
  @$pb.TagNumber(3)
  set credential(AccountTokenCredential value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCredential() => $_has(2);
  @$pb.TagNumber(3)
  void clearCredential() => $_clearField(3);
  @$pb.TagNumber(3)
  AccountTokenCredential ensureCredential() => $_ensure(2);
}

class LogoutAccountRequest extends $pb.GeneratedMessage {
  factory LogoutAccountRequest({
    $core.bool? allRefreshTokens,
  }) {
    final result = LogoutAccountRequest._();
    if (allRefreshTokens != null) result.allRefreshTokens = allRefreshTokens;
    return result;
  }

  LogoutAccountRequest._();

  factory LogoutAccountRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LogoutAccountRequest()..mergeFromBuffer(data, registry);
  factory LogoutAccountRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LogoutAccountRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LogoutAccountRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: LogoutAccountRequest.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'allRefreshTokens')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutAccountRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutAccountRequest copyWith(void Function(LogoutAccountRequest) updates) =>
      super.copyWith((message) => updates(message as LogoutAccountRequest))
          as LogoutAccountRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use LogoutAccountRequest() / LogoutAccountRequest.new instead')
  static LogoutAccountRequest create() => LogoutAccountRequest._();
  static $pb.GeneratedMessage $_createMessage() => LogoutAccountRequest._();
  @$core.override
  LogoutAccountRequest createEmptyInstance() => LogoutAccountRequest._();
  @$core.pragma('dart2js:noInline')
  static LogoutAccountRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LogoutAccountRequest>(
          LogoutAccountRequest.$_createMessage);
  static LogoutAccountRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get allRefreshTokens => $_getBF(0);
  @$pb.TagNumber(1)
  set allRefreshTokens($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAllRefreshTokens() => $_has(0);
  @$pb.TagNumber(1)
  void clearAllRefreshTokens() => $_clearField(1);
}

class LogoutAccountResponse extends $pb.GeneratedMessage {
  factory LogoutAccountResponse() => LogoutAccountResponse._();

  LogoutAccountResponse._();

  factory LogoutAccountResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LogoutAccountResponse()..mergeFromBuffer(data, registry);
  factory LogoutAccountResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      LogoutAccountResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LogoutAccountResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: LogoutAccountResponse.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutAccountResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogoutAccountResponse copyWith(
          void Function(LogoutAccountResponse) updates) =>
      super.copyWith((message) => updates(message as LogoutAccountResponse))
          as LogoutAccountResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use LogoutAccountResponse() / LogoutAccountResponse.new instead')
  static LogoutAccountResponse create() => LogoutAccountResponse._();
  static $pb.GeneratedMessage $_createMessage() => LogoutAccountResponse._();
  @$core.override
  LogoutAccountResponse createEmptyInstance() => LogoutAccountResponse._();
  @$core.pragma('dart2js:noInline')
  static LogoutAccountResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LogoutAccountResponse>(
          LogoutAccountResponse.$_createMessage);
  static LogoutAccountResponse? _defaultInstance;
}

class GetCurrentAccountRequest extends $pb.GeneratedMessage {
  factory GetCurrentAccountRequest() => GetCurrentAccountRequest._();

  GetCurrentAccountRequest._();

  factory GetCurrentAccountRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetCurrentAccountRequest()..mergeFromBuffer(data, registry);
  factory GetCurrentAccountRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetCurrentAccountRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCurrentAccountRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: GetCurrentAccountRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCurrentAccountRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCurrentAccountRequest copyWith(
          void Function(GetCurrentAccountRequest) updates) =>
      super.copyWith((message) => updates(message as GetCurrentAccountRequest))
          as GetCurrentAccountRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetCurrentAccountRequest() / GetCurrentAccountRequest.new instead')
  static GetCurrentAccountRequest create() => GetCurrentAccountRequest._();
  static $pb.GeneratedMessage $_createMessage() => GetCurrentAccountRequest._();
  @$core.override
  GetCurrentAccountRequest createEmptyInstance() =>
      GetCurrentAccountRequest._();
  @$core.pragma('dart2js:noInline')
  static GetCurrentAccountRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCurrentAccountRequest>(
          GetCurrentAccountRequest.$_createMessage);
  static GetCurrentAccountRequest? _defaultInstance;
}

class GetCurrentAccountResponse extends $pb.GeneratedMessage {
  factory GetCurrentAccountResponse({
    AccountProfile? account,
    $core.Iterable<AccountRole>? roles,
  }) {
    final result = GetCurrentAccountResponse._();
    if (account != null) result.account = account;
    if (roles != null) result.roles.addAll(roles);
    return result;
  }

  GetCurrentAccountResponse._();

  factory GetCurrentAccountResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetCurrentAccountResponse()..mergeFromBuffer(data, registry);
  factory GetCurrentAccountResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetCurrentAccountResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCurrentAccountResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: GetCurrentAccountResponse.$_createMessage)
    ..aOM<AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: AccountProfile.$_createMessage)
    ..pc<AccountRole>(2, _omitFieldNames ? '' : 'roles', $pb.PbFieldType.KE,
        valueOf: AccountRole.valueOf,
        enumValues: AccountRole.values,
        defaultEnumValue: AccountRole.ACCOUNT_ROLE_UNSPECIFIED)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCurrentAccountResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCurrentAccountResponse copyWith(
          void Function(GetCurrentAccountResponse) updates) =>
      super.copyWith((message) => updates(message as GetCurrentAccountResponse))
          as GetCurrentAccountResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetCurrentAccountResponse() / GetCurrentAccountResponse.new instead')
  static GetCurrentAccountResponse create() => GetCurrentAccountResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      GetCurrentAccountResponse._();
  @$core.override
  GetCurrentAccountResponse createEmptyInstance() =>
      GetCurrentAccountResponse._();
  @$core.pragma('dart2js:noInline')
  static GetCurrentAccountResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCurrentAccountResponse>(
          GetCurrentAccountResponse.$_createMessage);
  static GetCurrentAccountResponse? _defaultInstance;

  @$pb.TagNumber(1)
  AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account(AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<AccountRole> get roles => $_getList(1);
}

/// AccountRefreshTokenProjection 是用户可见的持久登录凭据元数据，不包含 token 摘要或原始值。
class AccountRefreshTokenProjection extends $pb.GeneratedMessage {
  factory AccountRefreshTokenProjection({
    $core.String? refreshId,
    $core.bool? current,
    $0.Timestamp? createdAt,
    $0.Timestamp? expiresAt,
    $fixnum.Int64? revision,
  }) {
    final result = AccountRefreshTokenProjection._();
    if (refreshId != null) result.refreshId = refreshId;
    if (current != null) result.current = current;
    if (createdAt != null) result.createdAt = createdAt;
    if (expiresAt != null) result.expiresAt = expiresAt;
    if (revision != null) result.revision = revision;
    return result;
  }

  AccountRefreshTokenProjection._();

  factory AccountRefreshTokenProjection.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AccountRefreshTokenProjection()..mergeFromBuffer(data, registry);
  factory AccountRefreshTokenProjection.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AccountRefreshTokenProjection()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AccountRefreshTokenProjection',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: AccountRefreshTokenProjection.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'refreshId')
    ..aOB(2, _omitFieldNames ? '' : 'current')
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..aOM<$0.Timestamp>(4, _omitFieldNames ? '' : 'expiresAt',
        subBuilder: $0.Timestamp.$_createMessage)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountRefreshTokenProjection clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AccountRefreshTokenProjection copyWith(
          void Function(AccountRefreshTokenProjection) updates) =>
      super.copyWith(
              (message) => updates(message as AccountRefreshTokenProjection))
          as AccountRefreshTokenProjection;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use AccountRefreshTokenProjection() / AccountRefreshTokenProjection.new instead')
  static AccountRefreshTokenProjection create() =>
      AccountRefreshTokenProjection._();
  static $pb.GeneratedMessage $_createMessage() =>
      AccountRefreshTokenProjection._();
  @$core.override
  AccountRefreshTokenProjection createEmptyInstance() =>
      AccountRefreshTokenProjection._();
  @$core.pragma('dart2js:noInline')
  static AccountRefreshTokenProjection getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AccountRefreshTokenProjection>(
          AccountRefreshTokenProjection.$_createMessage);
  static AccountRefreshTokenProjection? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get refreshId => $_getSZ(0);
  @$pb.TagNumber(1)
  set refreshId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRefreshId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRefreshId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get current => $_getBF(1);
  @$pb.TagNumber(2)
  set current($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCurrent() => $_has(1);
  @$pb.TagNumber(2)
  void clearCurrent() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get createdAt => $_getN(2);
  @$pb.TagNumber(3)
  set createdAt($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCreatedAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearCreatedAt() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureCreatedAt() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.Timestamp get expiresAt => $_getN(3);
  @$pb.TagNumber(4)
  set expiresAt($0.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasExpiresAt() => $_has(3);
  @$pb.TagNumber(4)
  void clearExpiresAt() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Timestamp ensureExpiresAt() => $_ensure(3);

  @$pb.TagNumber(6)
  $fixnum.Int64 get revision => $_getI64(4);
  @$pb.TagNumber(6)
  set revision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(6)
  $core.bool hasRevision() => $_has(4);
  @$pb.TagNumber(6)
  void clearRevision() => $_clearField(6);
}

class ListAccountRefreshTokensRequest extends $pb.GeneratedMessage {
  factory ListAccountRefreshTokensRequest() =>
      ListAccountRefreshTokensRequest._();

  ListAccountRefreshTokensRequest._();

  factory ListAccountRefreshTokensRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListAccountRefreshTokensRequest()..mergeFromBuffer(data, registry);
  factory ListAccountRefreshTokensRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListAccountRefreshTokensRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListAccountRefreshTokensRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ListAccountRefreshTokensRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListAccountRefreshTokensRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListAccountRefreshTokensRequest copyWith(
          void Function(ListAccountRefreshTokensRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ListAccountRefreshTokensRequest))
          as ListAccountRefreshTokensRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListAccountRefreshTokensRequest() / ListAccountRefreshTokensRequest.new instead')
  static ListAccountRefreshTokensRequest create() =>
      ListAccountRefreshTokensRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      ListAccountRefreshTokensRequest._();
  @$core.override
  ListAccountRefreshTokensRequest createEmptyInstance() =>
      ListAccountRefreshTokensRequest._();
  @$core.pragma('dart2js:noInline')
  static ListAccountRefreshTokensRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListAccountRefreshTokensRequest>(
          ListAccountRefreshTokensRequest.$_createMessage);
  static ListAccountRefreshTokensRequest? _defaultInstance;
}

class ListAccountRefreshTokensResponse extends $pb.GeneratedMessage {
  factory ListAccountRefreshTokensResponse({
    $core.Iterable<AccountRefreshTokenProjection>? refreshTokens,
  }) {
    final result = ListAccountRefreshTokensResponse._();
    if (refreshTokens != null) result.refreshTokens.addAll(refreshTokens);
    return result;
  }

  ListAccountRefreshTokensResponse._();

  factory ListAccountRefreshTokensResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListAccountRefreshTokensResponse()..mergeFromBuffer(data, registry);
  factory ListAccountRefreshTokensResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListAccountRefreshTokensResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListAccountRefreshTokensResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ListAccountRefreshTokensResponse.$_createMessage)
    ..pPM<AccountRefreshTokenProjection>(
        1, _omitFieldNames ? '' : 'refreshTokens',
        subBuilder: AccountRefreshTokenProjection.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListAccountRefreshTokensResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListAccountRefreshTokensResponse copyWith(
          void Function(ListAccountRefreshTokensResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ListAccountRefreshTokensResponse))
          as ListAccountRefreshTokensResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListAccountRefreshTokensResponse() / ListAccountRefreshTokensResponse.new instead')
  static ListAccountRefreshTokensResponse create() =>
      ListAccountRefreshTokensResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      ListAccountRefreshTokensResponse._();
  @$core.override
  ListAccountRefreshTokensResponse createEmptyInstance() =>
      ListAccountRefreshTokensResponse._();
  @$core.pragma('dart2js:noInline')
  static ListAccountRefreshTokensResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListAccountRefreshTokensResponse>(
          ListAccountRefreshTokensResponse.$_createMessage);
  static ListAccountRefreshTokensResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<AccountRefreshTokenProjection> get refreshTokens => $_getList(0);
}

class ChangeAccountPasswordRequest extends $pb.GeneratedMessage {
  factory ChangeAccountPasswordRequest({
    $core.String? newPassword,
  }) {
    final result = ChangeAccountPasswordRequest._();
    if (newPassword != null) result.newPassword = newPassword;
    return result;
  }

  ChangeAccountPasswordRequest._();

  factory ChangeAccountPasswordRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ChangeAccountPasswordRequest()..mergeFromBuffer(data, registry);
  factory ChangeAccountPasswordRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ChangeAccountPasswordRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeAccountPasswordRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ChangeAccountPasswordRequest.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'newPassword')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeAccountPasswordRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeAccountPasswordRequest copyWith(
          void Function(ChangeAccountPasswordRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ChangeAccountPasswordRequest))
          as ChangeAccountPasswordRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ChangeAccountPasswordRequest() / ChangeAccountPasswordRequest.new instead')
  static ChangeAccountPasswordRequest create() =>
      ChangeAccountPasswordRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      ChangeAccountPasswordRequest._();
  @$core.override
  ChangeAccountPasswordRequest createEmptyInstance() =>
      ChangeAccountPasswordRequest._();
  @$core.pragma('dart2js:noInline')
  static ChangeAccountPasswordRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ChangeAccountPasswordRequest>(
          ChangeAccountPasswordRequest.$_createMessage);
  static ChangeAccountPasswordRequest? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get newPassword => $_getSZ(0);
  @$pb.TagNumber(2)
  set newPassword($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasNewPassword() => $_has(0);
  @$pb.TagNumber(2)
  void clearNewPassword() => $_clearField(2);
}

class ChangeAccountPasswordResponse extends $pb.GeneratedMessage {
  factory ChangeAccountPasswordResponse({
    AccountProfile? account,
  }) {
    final result = ChangeAccountPasswordResponse._();
    if (account != null) result.account = account;
    return result;
  }

  ChangeAccountPasswordResponse._();

  factory ChangeAccountPasswordResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ChangeAccountPasswordResponse()..mergeFromBuffer(data, registry);
  factory ChangeAccountPasswordResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ChangeAccountPasswordResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChangeAccountPasswordResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: ChangeAccountPasswordResponse.$_createMessage)
    ..aOM<AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: AccountProfile.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeAccountPasswordResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChangeAccountPasswordResponse copyWith(
          void Function(ChangeAccountPasswordResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ChangeAccountPasswordResponse))
          as ChangeAccountPasswordResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ChangeAccountPasswordResponse() / ChangeAccountPasswordResponse.new instead')
  static ChangeAccountPasswordResponse create() =>
      ChangeAccountPasswordResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      ChangeAccountPasswordResponse._();
  @$core.override
  ChangeAccountPasswordResponse createEmptyInstance() =>
      ChangeAccountPasswordResponse._();
  @$core.pragma('dart2js:noInline')
  static ChangeAccountPasswordResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ChangeAccountPasswordResponse>(
          ChangeAccountPasswordResponse.$_createMessage);
  static ChangeAccountPasswordResponse? _defaultInstance;

  @$pb.TagNumber(1)
  AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account(AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  AccountProfile ensureAccount() => $_ensure(0);
}

class RedeemAccountSetupRequest extends $pb.GeneratedMessage {
  factory RedeemAccountSetupRequest({
    $core.String? setupCredential,
    $core.String? newPassword,
  }) {
    final result = RedeemAccountSetupRequest._();
    if (setupCredential != null) result.setupCredential = setupCredential;
    if (newPassword != null) result.newPassword = newPassword;
    return result;
  }

  RedeemAccountSetupRequest._();

  factory RedeemAccountSetupRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RedeemAccountSetupRequest()..mergeFromBuffer(data, registry);
  factory RedeemAccountSetupRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RedeemAccountSetupRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RedeemAccountSetupRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RedeemAccountSetupRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'setupCredential')
    ..aOS(2, _omitFieldNames ? '' : 'newPassword')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RedeemAccountSetupRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RedeemAccountSetupRequest copyWith(
          void Function(RedeemAccountSetupRequest) updates) =>
      super.copyWith((message) => updates(message as RedeemAccountSetupRequest))
          as RedeemAccountSetupRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RedeemAccountSetupRequest() / RedeemAccountSetupRequest.new instead')
  static RedeemAccountSetupRequest create() => RedeemAccountSetupRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      RedeemAccountSetupRequest._();
  @$core.override
  RedeemAccountSetupRequest createEmptyInstance() =>
      RedeemAccountSetupRequest._();
  @$core.pragma('dart2js:noInline')
  static RedeemAccountSetupRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RedeemAccountSetupRequest>(
          RedeemAccountSetupRequest.$_createMessage);
  static RedeemAccountSetupRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get setupCredential => $_getSZ(0);
  @$pb.TagNumber(1)
  set setupCredential($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSetupCredential() => $_has(0);
  @$pb.TagNumber(1)
  void clearSetupCredential() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get newPassword => $_getSZ(1);
  @$pb.TagNumber(2)
  set newPassword($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNewPassword() => $_has(1);
  @$pb.TagNumber(2)
  void clearNewPassword() => $_clearField(2);
}

class RedeemAccountSetupResponse extends $pb.GeneratedMessage {
  factory RedeemAccountSetupResponse({
    AccountProfile? account,
    $core.Iterable<AccountRole>? roles,
    AccountTokenCredential? credential,
  }) {
    final result = RedeemAccountSetupResponse._();
    if (account != null) result.account = account;
    if (roles != null) result.roles.addAll(roles);
    if (credential != null) result.credential = credential;
    return result;
  }

  RedeemAccountSetupResponse._();

  factory RedeemAccountSetupResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RedeemAccountSetupResponse()..mergeFromBuffer(data, registry);
  factory RedeemAccountSetupResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RedeemAccountSetupResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RedeemAccountSetupResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RedeemAccountSetupResponse.$_createMessage)
    ..aOM<AccountProfile>(1, _omitFieldNames ? '' : 'account',
        subBuilder: AccountProfile.$_createMessage)
    ..pc<AccountRole>(2, _omitFieldNames ? '' : 'roles', $pb.PbFieldType.KE,
        valueOf: AccountRole.valueOf,
        enumValues: AccountRole.values,
        defaultEnumValue: AccountRole.ACCOUNT_ROLE_UNSPECIFIED)
    ..aOM<AccountTokenCredential>(3, _omitFieldNames ? '' : 'credential',
        subBuilder: AccountTokenCredential.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RedeemAccountSetupResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RedeemAccountSetupResponse copyWith(
          void Function(RedeemAccountSetupResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RedeemAccountSetupResponse))
          as RedeemAccountSetupResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RedeemAccountSetupResponse() / RedeemAccountSetupResponse.new instead')
  static RedeemAccountSetupResponse create() => RedeemAccountSetupResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      RedeemAccountSetupResponse._();
  @$core.override
  RedeemAccountSetupResponse createEmptyInstance() =>
      RedeemAccountSetupResponse._();
  @$core.pragma('dart2js:noInline')
  static RedeemAccountSetupResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RedeemAccountSetupResponse>(
          RedeemAccountSetupResponse.$_createMessage);
  static RedeemAccountSetupResponse? _defaultInstance;

  @$pb.TagNumber(1)
  AccountProfile get account => $_getN(0);
  @$pb.TagNumber(1)
  set account(AccountProfile value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAccount() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccount() => $_clearField(1);
  @$pb.TagNumber(1)
  AccountProfile ensureAccount() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<AccountRole> get roles => $_getList(1);

  @$pb.TagNumber(3)
  AccountTokenCredential get credential => $_getN(2);
  @$pb.TagNumber(3)
  set credential(AccountTokenCredential value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCredential() => $_has(2);
  @$pb.TagNumber(3)
  void clearCredential() => $_clearField(3);
  @$pb.TagNumber(3)
  AccountTokenCredential ensureCredential() => $_ensure(2);
}

class RevokeAccountRefreshTokenRequest extends $pb.GeneratedMessage {
  factory RevokeAccountRefreshTokenRequest({
    $core.String? refreshId,
  }) {
    final result = RevokeAccountRefreshTokenRequest._();
    if (refreshId != null) result.refreshId = refreshId;
    return result;
  }

  RevokeAccountRefreshTokenRequest._();

  factory RevokeAccountRefreshTokenRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RevokeAccountRefreshTokenRequest()..mergeFromBuffer(data, registry);
  factory RevokeAccountRefreshTokenRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RevokeAccountRefreshTokenRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RevokeAccountRefreshTokenRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RevokeAccountRefreshTokenRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'refreshId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RevokeAccountRefreshTokenRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RevokeAccountRefreshTokenRequest copyWith(
          void Function(RevokeAccountRefreshTokenRequest) updates) =>
      super.copyWith(
              (message) => updates(message as RevokeAccountRefreshTokenRequest))
          as RevokeAccountRefreshTokenRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RevokeAccountRefreshTokenRequest() / RevokeAccountRefreshTokenRequest.new instead')
  static RevokeAccountRefreshTokenRequest create() =>
      RevokeAccountRefreshTokenRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      RevokeAccountRefreshTokenRequest._();
  @$core.override
  RevokeAccountRefreshTokenRequest createEmptyInstance() =>
      RevokeAccountRefreshTokenRequest._();
  @$core.pragma('dart2js:noInline')
  static RevokeAccountRefreshTokenRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RevokeAccountRefreshTokenRequest>(
          RevokeAccountRefreshTokenRequest.$_createMessage);
  static RevokeAccountRefreshTokenRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get refreshId => $_getSZ(0);
  @$pb.TagNumber(1)
  set refreshId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRefreshId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRefreshId() => $_clearField(1);
}

class RevokeAccountRefreshTokenResponse extends $pb.GeneratedMessage {
  factory RevokeAccountRefreshTokenResponse() =>
      RevokeAccountRefreshTokenResponse._();

  RevokeAccountRefreshTokenResponse._();

  factory RevokeAccountRefreshTokenResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RevokeAccountRefreshTokenResponse()..mergeFromBuffer(data, registry);
  factory RevokeAccountRefreshTokenResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RevokeAccountRefreshTokenResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RevokeAccountRefreshTokenResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: RevokeAccountRefreshTokenResponse.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RevokeAccountRefreshTokenResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RevokeAccountRefreshTokenResponse copyWith(
          void Function(RevokeAccountRefreshTokenResponse) updates) =>
      super.copyWith((message) =>
              updates(message as RevokeAccountRefreshTokenResponse))
          as RevokeAccountRefreshTokenResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RevokeAccountRefreshTokenResponse() / RevokeAccountRefreshTokenResponse.new instead')
  static RevokeAccountRefreshTokenResponse create() =>
      RevokeAccountRefreshTokenResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      RevokeAccountRefreshTokenResponse._();
  @$core.override
  RevokeAccountRefreshTokenResponse createEmptyInstance() =>
      RevokeAccountRefreshTokenResponse._();
  @$core.pragma('dart2js:noInline')
  static RevokeAccountRefreshTokenResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RevokeAccountRefreshTokenResponse>(
          RevokeAccountRefreshTokenResponse.$_createMessage);
  static RevokeAccountRefreshTokenResponse? _defaultInstance;
}

/// DeleteAccountRequest 由已登录账号所有者提交，永久删除账号及其个人数据（GDPR）。
class DeleteAccountRequest extends $pb.GeneratedMessage {
  factory DeleteAccountRequest() => DeleteAccountRequest._();

  DeleteAccountRequest._();

  factory DeleteAccountRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteAccountRequest()..mergeFromBuffer(data, registry);
  factory DeleteAccountRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteAccountRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeleteAccountRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: DeleteAccountRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteAccountRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteAccountRequest copyWith(void Function(DeleteAccountRequest) updates) =>
      super.copyWith((message) => updates(message as DeleteAccountRequest))
          as DeleteAccountRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use DeleteAccountRequest() / DeleteAccountRequest.new instead')
  static DeleteAccountRequest create() => DeleteAccountRequest._();
  static $pb.GeneratedMessage $_createMessage() => DeleteAccountRequest._();
  @$core.override
  DeleteAccountRequest createEmptyInstance() => DeleteAccountRequest._();
  @$core.pragma('dart2js:noInline')
  static DeleteAccountRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeleteAccountRequest>(
          DeleteAccountRequest.$_createMessage);
  static DeleteAccountRequest? _defaultInstance;
}

class DeleteAccountResponse extends $pb.GeneratedMessage {
  factory DeleteAccountResponse() => DeleteAccountResponse._();

  DeleteAccountResponse._();

  factory DeleteAccountResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteAccountResponse()..mergeFromBuffer(data, registry);
  factory DeleteAccountResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeleteAccountResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeleteAccountResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: DeleteAccountResponse.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteAccountResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeleteAccountResponse copyWith(
          void Function(DeleteAccountResponse) updates) =>
      super.copyWith((message) => updates(message as DeleteAccountResponse))
          as DeleteAccountResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use DeleteAccountResponse() / DeleteAccountResponse.new instead')
  static DeleteAccountResponse create() => DeleteAccountResponse._();
  static $pb.GeneratedMessage $_createMessage() => DeleteAccountResponse._();
  @$core.override
  DeleteAccountResponse createEmptyInstance() => DeleteAccountResponse._();
  @$core.pragma('dart2js:noInline')
  static DeleteAccountResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeleteAccountResponse>(
          DeleteAccountResponse.$_createMessage);
  static DeleteAccountResponse? _defaultInstance;
}

/// AccountService 是账号、credential、Access JWT 和 Refresh token 的唯一公共应用 API。
class AccountServiceApi {
  final $pb.RpcClient _client;

  AccountServiceApi(this._client);

  $async.Future<LoginAccountResponse> login(
          $pb.ClientContext? ctx, LoginAccountRequest request) =>
      _client.invoke<LoginAccountResponse>(
          ctx, 'AccountService', 'Login', request, LoginAccountResponse());
  $async.Future<RefreshAccountTokenResponse> refresh(
          $pb.ClientContext? ctx, RefreshAccountTokenRequest request) =>
      _client.invoke<RefreshAccountTokenResponse>(ctx, 'AccountService',
          'Refresh', request, RefreshAccountTokenResponse());
  $async.Future<LogoutAccountResponse> logout(
          $pb.ClientContext? ctx, LogoutAccountRequest request) =>
      _client.invoke<LogoutAccountResponse>(
          ctx, 'AccountService', 'Logout', request, LogoutAccountResponse());
  $async.Future<GetCurrentAccountResponse> getCurrent(
          $pb.ClientContext? ctx, GetCurrentAccountRequest request) =>
      _client.invoke<GetCurrentAccountResponse>(ctx, 'AccountService',
          'GetCurrent', request, GetCurrentAccountResponse());
  $async.Future<ListAccountRefreshTokensResponse> listRefreshTokens(
          $pb.ClientContext? ctx, ListAccountRefreshTokensRequest request) =>
      _client.invoke<ListAccountRefreshTokensResponse>(ctx, 'AccountService',
          'ListRefreshTokens', request, ListAccountRefreshTokensResponse());
  $async.Future<ChangeAccountPasswordResponse> changePassword(
          $pb.ClientContext? ctx, ChangeAccountPasswordRequest request) =>
      _client.invoke<ChangeAccountPasswordResponse>(ctx, 'AccountService',
          'ChangePassword', request, ChangeAccountPasswordResponse());
  $async.Future<RedeemAccountSetupResponse> redeemAccountSetup(
          $pb.ClientContext? ctx, RedeemAccountSetupRequest request) =>
      _client.invoke<RedeemAccountSetupResponse>(ctx, 'AccountService',
          'RedeemAccountSetup', request, RedeemAccountSetupResponse());
  $async.Future<RevokeAccountRefreshTokenResponse> revokeRefreshToken(
          $pb.ClientContext? ctx, RevokeAccountRefreshTokenRequest request) =>
      _client.invoke<RevokeAccountRefreshTokenResponse>(ctx, 'AccountService',
          'RevokeRefreshToken', request, RevokeAccountRefreshTokenResponse());
  $async.Future<DeleteAccountResponse> deleteAccount(
          $pb.ClientContext? ctx, DeleteAccountRequest request) =>
      _client.invoke<DeleteAccountResponse>(ctx, 'AccountService',
          'DeleteAccount', request, DeleteAccountResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
