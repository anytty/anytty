// This is a generated file - do not edit.
//
// Generated from cloud/v1/account.proto.

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

@$core.Deprecated('Use accountStateDescriptor instead')
const AccountState$json = {
  '1': 'AccountState',
  '2': [
    {'1': 'ACCOUNT_STATE_UNSPECIFIED', '2': 0},
    {'1': 'ACCOUNT_STATE_PENDING', '2': 1},
    {'1': 'ACCOUNT_STATE_ACTIVE', '2': 2},
    {'1': 'ACCOUNT_STATE_DISABLED', '2': 3},
  ],
};

/// Descriptor for `AccountState`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List accountStateDescriptor = $convert.base64Decode(
    'CgxBY2NvdW50U3RhdGUSHQoZQUNDT1VOVF9TVEFURV9VTlNQRUNJRklFRBAAEhkKFUFDQ09VTl'
    'RfU1RBVEVfUEVORElORxABEhgKFEFDQ09VTlRfU1RBVEVfQUNUSVZFEAISGgoWQUNDT1VOVF9T'
    'VEFURV9ESVNBQkxFRBAD');

@$core.Deprecated('Use accountRoleDescriptor instead')
const AccountRole$json = {
  '1': 'AccountRole',
  '2': [
    {'1': 'ACCOUNT_ROLE_UNSPECIFIED', '2': 0},
    {'1': 'ACCOUNT_ROLE_USER', '2': 1},
    {'1': 'ACCOUNT_ROLE_OPERATOR', '2': 2},
    {'1': 'ACCOUNT_ROLE_ADMIN', '2': 3},
  ],
};

/// Descriptor for `AccountRole`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List accountRoleDescriptor = $convert.base64Decode(
    'CgtBY2NvdW50Um9sZRIcChhBQ0NPVU5UX1JPTEVfVU5TUEVDSUZJRUQQABIVChFBQ0NPVU5UX1'
    'JPTEVfVVNFUhABEhkKFUFDQ09VTlRfUk9MRV9PUEVSQVRPUhACEhYKEkFDQ09VTlRfUk9MRV9B'
    'RE1JThAD');

@$core.Deprecated('Use accountProfileDescriptor instead')
const AccountProfile$json = {
  '1': 'AccountProfile',
  '2': [
    {'1': 'account_id', '3': 1, '4': 1, '5': 9, '10': 'accountId'},
    {'1': 'email', '3': 2, '4': 1, '5': 9, '10': 'email'},
    {'1': 'display_name', '3': 3, '4': 1, '5': 9, '10': 'displayName'},
    {
      '1': 'state',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountState',
      '10': 'state'
    },
    {'1': 'revision', '3': 5, '4': 1, '5': 4, '10': 'revision'},
    {
      '1': 'created_at',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createdAt'
    },
    {
      '1': 'updated_at',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'updatedAt'
    },
    {'1': 'email_verified', '3': 8, '4': 1, '5': 8, '10': 'emailVerified'},
  ],
};

/// Descriptor for `AccountProfile`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List accountProfileDescriptor = $convert.base64Decode(
    'Cg5BY2NvdW50UHJvZmlsZRIdCgphY2NvdW50X2lkGAEgASgJUglhY2NvdW50SWQSFAoFZW1haW'
    'wYAiABKAlSBWVtYWlsEiEKDGRpc3BsYXlfbmFtZRgDIAEoCVILZGlzcGxheU5hbWUSMwoFc3Rh'
    'dGUYBCABKA4yHS5hbnl0dHkuY2xvdWQudjEuQWNjb3VudFN0YXRlUgVzdGF0ZRIaCghyZXZpc2'
    'lvbhgFIAEoBFIIcmV2aXNpb24SOQoKY3JlYXRlZF9hdBgGIAEoCzIaLmdvb2dsZS5wcm90b2J1'
    'Zi5UaW1lc3RhbXBSCWNyZWF0ZWRBdBI5Cgp1cGRhdGVkX2F0GAcgASgLMhouZ29vZ2xlLnByb3'
    'RvYnVmLlRpbWVzdGFtcFIJdXBkYXRlZEF0EiUKDmVtYWlsX3ZlcmlmaWVkGAggASgIUg1lbWFp'
    'bFZlcmlmaWVk');

@$core.Deprecated('Use accountTokenCredentialDescriptor instead')
const AccountTokenCredential$json = {
  '1': 'AccountTokenCredential',
  '2': [
    {'1': 'refresh_id', '3': 1, '4': 1, '5': 9, '10': 'refreshId'},
    {'1': 'access_token', '3': 2, '4': 1, '5': 9, '10': 'accessToken'},
    {'1': 'refresh_token', '3': 3, '4': 1, '5': 12, '10': 'refreshToken'},
    {
      '1': 'access_expires_at',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'accessExpiresAt'
    },
    {
      '1': 'refresh_expires_at',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'refreshExpiresAt'
    },
    {'1': 'csrf_token', '3': 6, '4': 1, '5': 12, '10': 'csrfToken'},
  ],
};

/// Descriptor for `AccountTokenCredential`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List accountTokenCredentialDescriptor = $convert.base64Decode(
    'ChZBY2NvdW50VG9rZW5DcmVkZW50aWFsEh0KCnJlZnJlc2hfaWQYASABKAlSCXJlZnJlc2hJZB'
    'IhCgxhY2Nlc3NfdG9rZW4YAiABKAlSC2FjY2Vzc1Rva2VuEiMKDXJlZnJlc2hfdG9rZW4YAyAB'
    'KAxSDHJlZnJlc2hUb2tlbhJGChFhY2Nlc3NfZXhwaXJlc19hdBgEIAEoCzIaLmdvb2dsZS5wcm'
    '90b2J1Zi5UaW1lc3RhbXBSD2FjY2Vzc0V4cGlyZXNBdBJIChJyZWZyZXNoX2V4cGlyZXNfYXQY'
    'BSABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUhByZWZyZXNoRXhwaXJlc0F0Eh0KCm'
    'NzcmZfdG9rZW4YBiABKAxSCWNzcmZUb2tlbg==');

@$core.Deprecated('Use loginAccountRequestDescriptor instead')
const LoginAccountRequest$json = {
  '1': 'LoginAccountRequest',
  '2': [
    {'1': 'login', '3': 1, '4': 1, '5': 9, '10': 'login'},
    {'1': 'password', '3': 2, '4': 1, '5': 9, '10': 'password'},
  ],
};

/// Descriptor for `LoginAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List loginAccountRequestDescriptor = $convert.base64Decode(
    'ChNMb2dpbkFjY291bnRSZXF1ZXN0EhQKBWxvZ2luGAEgASgJUgVsb2dpbhIaCghwYXNzd29yZB'
    'gCIAEoCVIIcGFzc3dvcmQ=');

@$core.Deprecated('Use loginAccountResponseDescriptor instead')
const LoginAccountResponse$json = {
  '1': 'LoginAccountResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {
      '1': 'roles',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'roles'
    },
    {
      '1': 'credential',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountTokenCredential',
      '10': 'credential'
    },
  ],
};

/// Descriptor for `LoginAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List loginAccountResponseDescriptor = $convert.base64Decode(
    'ChRMb2dpbkFjY291bnRSZXNwb25zZRI5CgdhY2NvdW50GAEgASgLMh8uYW55dHR5LmNsb3VkLn'
    'YxLkFjY291bnRQcm9maWxlUgdhY2NvdW50EjIKBXJvbGVzGAIgAygOMhwuYW55dHR5LmNsb3Vk'
    'LnYxLkFjY291bnRSb2xlUgVyb2xlcxJHCgpjcmVkZW50aWFsGAMgASgLMicuYW55dHR5LmNsb3'
    'VkLnYxLkFjY291bnRUb2tlbkNyZWRlbnRpYWxSCmNyZWRlbnRpYWw=');

@$core.Deprecated('Use refreshAccountTokenRequestDescriptor instead')
const RefreshAccountTokenRequest$json = {
  '1': 'RefreshAccountTokenRequest',
  '2': [
    {'1': 'refresh_token', '3': 1, '4': 1, '5': 12, '10': 'refreshToken'},
  ],
};

/// Descriptor for `RefreshAccountTokenRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List refreshAccountTokenRequestDescriptor =
    $convert.base64Decode(
        'ChpSZWZyZXNoQWNjb3VudFRva2VuUmVxdWVzdBIjCg1yZWZyZXNoX3Rva2VuGAEgASgMUgxyZW'
        'ZyZXNoVG9rZW4=');

@$core.Deprecated('Use refreshAccountTokenResponseDescriptor instead')
const RefreshAccountTokenResponse$json = {
  '1': 'RefreshAccountTokenResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {
      '1': 'roles',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'roles'
    },
    {
      '1': 'credential',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountTokenCredential',
      '10': 'credential'
    },
  ],
};

/// Descriptor for `RefreshAccountTokenResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List refreshAccountTokenResponseDescriptor = $convert.base64Decode(
    'ChtSZWZyZXNoQWNjb3VudFRva2VuUmVzcG9uc2USOQoHYWNjb3VudBgBIAEoCzIfLmFueXR0eS'
    '5jbG91ZC52MS5BY2NvdW50UHJvZmlsZVIHYWNjb3VudBIyCgVyb2xlcxgCIAMoDjIcLmFueXR0'
    'eS5jbG91ZC52MS5BY2NvdW50Um9sZVIFcm9sZXMSRwoKY3JlZGVudGlhbBgDIAEoCzInLmFueX'
    'R0eS5jbG91ZC52MS5BY2NvdW50VG9rZW5DcmVkZW50aWFsUgpjcmVkZW50aWFs');

@$core.Deprecated('Use logoutAccountRequestDescriptor instead')
const LogoutAccountRequest$json = {
  '1': 'LogoutAccountRequest',
  '2': [
    {
      '1': 'all_refresh_tokens',
      '3': 1,
      '4': 1,
      '5': 8,
      '10': 'allRefreshTokens'
    },
  ],
};

/// Descriptor for `LogoutAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logoutAccountRequestDescriptor = $convert.base64Decode(
    'ChRMb2dvdXRBY2NvdW50UmVxdWVzdBIsChJhbGxfcmVmcmVzaF90b2tlbnMYASABKAhSEGFsbF'
    'JlZnJlc2hUb2tlbnM=');

@$core.Deprecated('Use logoutAccountResponseDescriptor instead')
const LogoutAccountResponse$json = {
  '1': 'LogoutAccountResponse',
};

/// Descriptor for `LogoutAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logoutAccountResponseDescriptor =
    $convert.base64Decode('ChVMb2dvdXRBY2NvdW50UmVzcG9uc2U=');

@$core.Deprecated('Use getCurrentAccountRequestDescriptor instead')
const GetCurrentAccountRequest$json = {
  '1': 'GetCurrentAccountRequest',
};

/// Descriptor for `GetCurrentAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCurrentAccountRequestDescriptor =
    $convert.base64Decode('ChhHZXRDdXJyZW50QWNjb3VudFJlcXVlc3Q=');

@$core.Deprecated('Use getCurrentAccountResponseDescriptor instead')
const GetCurrentAccountResponse$json = {
  '1': 'GetCurrentAccountResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {
      '1': 'roles',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'roles'
    },
  ],
  '9': [
    {'1': 3, '2': 4},
  ],
};

/// Descriptor for `GetCurrentAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCurrentAccountResponseDescriptor = $convert.base64Decode(
    'ChlHZXRDdXJyZW50QWNjb3VudFJlc3BvbnNlEjkKB2FjY291bnQYASABKAsyHy5hbnl0dHkuY2'
    'xvdWQudjEuQWNjb3VudFByb2ZpbGVSB2FjY291bnQSMgoFcm9sZXMYAiADKA4yHC5hbnl0dHku'
    'Y2xvdWQudjEuQWNjb3VudFJvbGVSBXJvbGVzSgQIAxAE');

@$core.Deprecated('Use accountRefreshTokenProjectionDescriptor instead')
const AccountRefreshTokenProjection$json = {
  '1': 'AccountRefreshTokenProjection',
  '2': [
    {'1': 'refresh_id', '3': 1, '4': 1, '5': 9, '10': 'refreshId'},
    {'1': 'current', '3': 2, '4': 1, '5': 8, '10': 'current'},
    {
      '1': 'created_at',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'createdAt'
    },
    {
      '1': 'expires_at',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Timestamp',
      '10': 'expiresAt'
    },
    {'1': 'revision', '3': 6, '4': 1, '5': 4, '10': 'revision'},
  ],
  '9': [
    {'1': 5, '2': 6},
  ],
};

/// Descriptor for `AccountRefreshTokenProjection`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List accountRefreshTokenProjectionDescriptor = $convert.base64Decode(
    'Ch1BY2NvdW50UmVmcmVzaFRva2VuUHJvamVjdGlvbhIdCgpyZWZyZXNoX2lkGAEgASgJUglyZW'
    'ZyZXNoSWQSGAoHY3VycmVudBgCIAEoCFIHY3VycmVudBI5CgpjcmVhdGVkX2F0GAMgASgLMhou'
    'Z29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJY3JlYXRlZEF0EjkKCmV4cGlyZXNfYXQYBCABKA'
    'syGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUglleHBpcmVzQXQSGgoIcmV2aXNpb24YBiAB'
    'KARSCHJldmlzaW9uSgQIBRAG');

@$core.Deprecated('Use listAccountRefreshTokensRequestDescriptor instead')
const ListAccountRefreshTokensRequest$json = {
  '1': 'ListAccountRefreshTokensRequest',
};

/// Descriptor for `ListAccountRefreshTokensRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listAccountRefreshTokensRequestDescriptor =
    $convert.base64Decode('Ch9MaXN0QWNjb3VudFJlZnJlc2hUb2tlbnNSZXF1ZXN0');

@$core.Deprecated('Use listAccountRefreshTokensResponseDescriptor instead')
const ListAccountRefreshTokensResponse$json = {
  '1': 'ListAccountRefreshTokensResponse',
  '2': [
    {
      '1': 'refresh_tokens',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountRefreshTokenProjection',
      '10': 'refreshTokens'
    },
  ],
};

/// Descriptor for `ListAccountRefreshTokensResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listAccountRefreshTokensResponseDescriptor =
    $convert.base64Decode(
        'CiBMaXN0QWNjb3VudFJlZnJlc2hUb2tlbnNSZXNwb25zZRJVCg5yZWZyZXNoX3Rva2VucxgBIA'
        'MoCzIuLmFueXR0eS5jbG91ZC52MS5BY2NvdW50UmVmcmVzaFRva2VuUHJvamVjdGlvblINcmVm'
        'cmVzaFRva2Vucw==');

@$core.Deprecated('Use changeAccountPasswordRequestDescriptor instead')
const ChangeAccountPasswordRequest$json = {
  '1': 'ChangeAccountPasswordRequest',
  '2': [
    {'1': 'new_password', '3': 2, '4': 1, '5': 9, '10': 'newPassword'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `ChangeAccountPasswordRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeAccountPasswordRequestDescriptor =
    $convert.base64Decode(
        'ChxDaGFuZ2VBY2NvdW50UGFzc3dvcmRSZXF1ZXN0EiEKDG5ld19wYXNzd29yZBgCIAEoCVILbm'
        'V3UGFzc3dvcmRKBAgBEAI=');

@$core.Deprecated('Use changeAccountPasswordResponseDescriptor instead')
const ChangeAccountPasswordResponse$json = {
  '1': 'ChangeAccountPasswordResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
  ],
};

/// Descriptor for `ChangeAccountPasswordResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List changeAccountPasswordResponseDescriptor =
    $convert.base64Decode(
        'Ch1DaGFuZ2VBY2NvdW50UGFzc3dvcmRSZXNwb25zZRI5CgdhY2NvdW50GAEgASgLMh8uYW55dH'
        'R5LmNsb3VkLnYxLkFjY291bnRQcm9maWxlUgdhY2NvdW50');

@$core.Deprecated('Use redeemAccountSetupRequestDescriptor instead')
const RedeemAccountSetupRequest$json = {
  '1': 'RedeemAccountSetupRequest',
  '2': [
    {'1': 'setup_credential', '3': 1, '4': 1, '5': 9, '10': 'setupCredential'},
    {'1': 'new_password', '3': 2, '4': 1, '5': 9, '10': 'newPassword'},
  ],
};

/// Descriptor for `RedeemAccountSetupRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List redeemAccountSetupRequestDescriptor =
    $convert.base64Decode(
        'ChlSZWRlZW1BY2NvdW50U2V0dXBSZXF1ZXN0EikKEHNldHVwX2NyZWRlbnRpYWwYASABKAlSD3'
        'NldHVwQ3JlZGVudGlhbBIhCgxuZXdfcGFzc3dvcmQYAiABKAlSC25ld1Bhc3N3b3Jk');

@$core.Deprecated('Use redeemAccountSetupResponseDescriptor instead')
const RedeemAccountSetupResponse$json = {
  '1': 'RedeemAccountSetupResponse',
  '2': [
    {
      '1': 'account',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountProfile',
      '10': 'account'
    },
    {
      '1': 'roles',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.anytty.cloud.v1.AccountRole',
      '10': 'roles'
    },
    {
      '1': 'credential',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.cloud.v1.AccountTokenCredential',
      '10': 'credential'
    },
  ],
};

/// Descriptor for `RedeemAccountSetupResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List redeemAccountSetupResponseDescriptor = $convert.base64Decode(
    'ChpSZWRlZW1BY2NvdW50U2V0dXBSZXNwb25zZRI5CgdhY2NvdW50GAEgASgLMh8uYW55dHR5Lm'
    'Nsb3VkLnYxLkFjY291bnRQcm9maWxlUgdhY2NvdW50EjIKBXJvbGVzGAIgAygOMhwuYW55dHR5'
    'LmNsb3VkLnYxLkFjY291bnRSb2xlUgVyb2xlcxJHCgpjcmVkZW50aWFsGAMgASgLMicuYW55dH'
    'R5LmNsb3VkLnYxLkFjY291bnRUb2tlbkNyZWRlbnRpYWxSCmNyZWRlbnRpYWw=');

@$core.Deprecated('Use revokeAccountRefreshTokenRequestDescriptor instead')
const RevokeAccountRefreshTokenRequest$json = {
  '1': 'RevokeAccountRefreshTokenRequest',
  '2': [
    {'1': 'refresh_id', '3': 1, '4': 1, '5': 9, '10': 'refreshId'},
  ],
};

/// Descriptor for `RevokeAccountRefreshTokenRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List revokeAccountRefreshTokenRequestDescriptor =
    $convert.base64Decode(
        'CiBSZXZva2VBY2NvdW50UmVmcmVzaFRva2VuUmVxdWVzdBIdCgpyZWZyZXNoX2lkGAEgASgJUg'
        'lyZWZyZXNoSWQ=');

@$core.Deprecated('Use revokeAccountRefreshTokenResponseDescriptor instead')
const RevokeAccountRefreshTokenResponse$json = {
  '1': 'RevokeAccountRefreshTokenResponse',
};

/// Descriptor for `RevokeAccountRefreshTokenResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List revokeAccountRefreshTokenResponseDescriptor =
    $convert.base64Decode('CiFSZXZva2VBY2NvdW50UmVmcmVzaFRva2VuUmVzcG9uc2U=');

@$core.Deprecated('Use deleteAccountRequestDescriptor instead')
const DeleteAccountRequest$json = {
  '1': 'DeleteAccountRequest',
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `DeleteAccountRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteAccountRequestDescriptor =
    $convert.base64Decode('ChREZWxldGVBY2NvdW50UmVxdWVzdEoECAEQAg==');

@$core.Deprecated('Use deleteAccountResponseDescriptor instead')
const DeleteAccountResponse$json = {
  '1': 'DeleteAccountResponse',
};

/// Descriptor for `DeleteAccountResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteAccountResponseDescriptor =
    $convert.base64Decode('ChVEZWxldGVBY2NvdW50UmVzcG9uc2U=');

const $core.Map<$core.String, $core.dynamic> AccountServiceBase$json = {
  '1': 'AccountService',
  '2': [
    {
      '1': 'Login',
      '2': '.anytty.cloud.v1.LoginAccountRequest',
      '3': '.anytty.cloud.v1.LoginAccountResponse'
    },
    {
      '1': 'Refresh',
      '2': '.anytty.cloud.v1.RefreshAccountTokenRequest',
      '3': '.anytty.cloud.v1.RefreshAccountTokenResponse'
    },
    {
      '1': 'Logout',
      '2': '.anytty.cloud.v1.LogoutAccountRequest',
      '3': '.anytty.cloud.v1.LogoutAccountResponse'
    },
    {
      '1': 'GetCurrent',
      '2': '.anytty.cloud.v1.GetCurrentAccountRequest',
      '3': '.anytty.cloud.v1.GetCurrentAccountResponse'
    },
    {
      '1': 'ListRefreshTokens',
      '2': '.anytty.cloud.v1.ListAccountRefreshTokensRequest',
      '3': '.anytty.cloud.v1.ListAccountRefreshTokensResponse'
    },
    {
      '1': 'ChangePassword',
      '2': '.anytty.cloud.v1.ChangeAccountPasswordRequest',
      '3': '.anytty.cloud.v1.ChangeAccountPasswordResponse'
    },
    {
      '1': 'RedeemAccountSetup',
      '2': '.anytty.cloud.v1.RedeemAccountSetupRequest',
      '3': '.anytty.cloud.v1.RedeemAccountSetupResponse'
    },
    {
      '1': 'RevokeRefreshToken',
      '2': '.anytty.cloud.v1.RevokeAccountRefreshTokenRequest',
      '3': '.anytty.cloud.v1.RevokeAccountRefreshTokenResponse'
    },
    {
      '1': 'DeleteAccount',
      '2': '.anytty.cloud.v1.DeleteAccountRequest',
      '3': '.anytty.cloud.v1.DeleteAccountResponse'
    },
  ],
};

@$core.Deprecated('Use accountServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    AccountServiceBase$messageJson = {
  '.anytty.cloud.v1.LoginAccountRequest': LoginAccountRequest$json,
  '.anytty.cloud.v1.LoginAccountResponse': LoginAccountResponse$json,
  '.anytty.cloud.v1.AccountProfile': AccountProfile$json,
  '.google.protobuf.Timestamp': $0.Timestamp$json,
  '.anytty.cloud.v1.AccountTokenCredential': AccountTokenCredential$json,
  '.anytty.cloud.v1.RefreshAccountTokenRequest':
      RefreshAccountTokenRequest$json,
  '.anytty.cloud.v1.RefreshAccountTokenResponse':
      RefreshAccountTokenResponse$json,
  '.anytty.cloud.v1.LogoutAccountRequest': LogoutAccountRequest$json,
  '.anytty.cloud.v1.LogoutAccountResponse': LogoutAccountResponse$json,
  '.anytty.cloud.v1.GetCurrentAccountRequest': GetCurrentAccountRequest$json,
  '.anytty.cloud.v1.GetCurrentAccountResponse': GetCurrentAccountResponse$json,
  '.anytty.cloud.v1.ListAccountRefreshTokensRequest':
      ListAccountRefreshTokensRequest$json,
  '.anytty.cloud.v1.ListAccountRefreshTokensResponse':
      ListAccountRefreshTokensResponse$json,
  '.anytty.cloud.v1.AccountRefreshTokenProjection':
      AccountRefreshTokenProjection$json,
  '.anytty.cloud.v1.ChangeAccountPasswordRequest':
      ChangeAccountPasswordRequest$json,
  '.anytty.cloud.v1.ChangeAccountPasswordResponse':
      ChangeAccountPasswordResponse$json,
  '.anytty.cloud.v1.RedeemAccountSetupRequest': RedeemAccountSetupRequest$json,
  '.anytty.cloud.v1.RedeemAccountSetupResponse':
      RedeemAccountSetupResponse$json,
  '.anytty.cloud.v1.RevokeAccountRefreshTokenRequest':
      RevokeAccountRefreshTokenRequest$json,
  '.anytty.cloud.v1.RevokeAccountRefreshTokenResponse':
      RevokeAccountRefreshTokenResponse$json,
  '.anytty.cloud.v1.DeleteAccountRequest': DeleteAccountRequest$json,
  '.anytty.cloud.v1.DeleteAccountResponse': DeleteAccountResponse$json,
};

/// Descriptor for `AccountService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List accountServiceDescriptor = $convert.base64Decode(
    'Cg5BY2NvdW50U2VydmljZRJUCgVMb2dpbhIkLmFueXR0eS5jbG91ZC52MS5Mb2dpbkFjY291bn'
    'RSZXF1ZXN0GiUuYW55dHR5LmNsb3VkLnYxLkxvZ2luQWNjb3VudFJlc3BvbnNlEmQKB1JlZnJl'
    'c2gSKy5hbnl0dHkuY2xvdWQudjEuUmVmcmVzaEFjY291bnRUb2tlblJlcXVlc3QaLC5hbnl0dH'
    'kuY2xvdWQudjEuUmVmcmVzaEFjY291bnRUb2tlblJlc3BvbnNlElcKBkxvZ291dBIlLmFueXR0'
    'eS5jbG91ZC52MS5Mb2dvdXRBY2NvdW50UmVxdWVzdBomLmFueXR0eS5jbG91ZC52MS5Mb2dvdX'
    'RBY2NvdW50UmVzcG9uc2USYwoKR2V0Q3VycmVudBIpLmFueXR0eS5jbG91ZC52MS5HZXRDdXJy'
    'ZW50QWNjb3VudFJlcXVlc3QaKi5hbnl0dHkuY2xvdWQudjEuR2V0Q3VycmVudEFjY291bnRSZX'
    'Nwb25zZRJ4ChFMaXN0UmVmcmVzaFRva2VucxIwLmFueXR0eS5jbG91ZC52MS5MaXN0QWNjb3Vu'
    'dFJlZnJlc2hUb2tlbnNSZXF1ZXN0GjEuYW55dHR5LmNsb3VkLnYxLkxpc3RBY2NvdW50UmVmcm'
    'VzaFRva2Vuc1Jlc3BvbnNlEm8KDkNoYW5nZVBhc3N3b3JkEi0uYW55dHR5LmNsb3VkLnYxLkNo'
    'YW5nZUFjY291bnRQYXNzd29yZFJlcXVlc3QaLi5hbnl0dHkuY2xvdWQudjEuQ2hhbmdlQWNjb3'
    'VudFBhc3N3b3JkUmVzcG9uc2USbQoSUmVkZWVtQWNjb3VudFNldHVwEiouYW55dHR5LmNsb3Vk'
    'LnYxLlJlZGVlbUFjY291bnRTZXR1cFJlcXVlc3QaKy5hbnl0dHkuY2xvdWQudjEuUmVkZWVtQW'
    'Njb3VudFNldHVwUmVzcG9uc2USewoSUmV2b2tlUmVmcmVzaFRva2VuEjEuYW55dHR5LmNsb3Vk'
    'LnYxLlJldm9rZUFjY291bnRSZWZyZXNoVG9rZW5SZXF1ZXN0GjIuYW55dHR5LmNsb3VkLnYxLl'
    'Jldm9rZUFjY291bnRSZWZyZXNoVG9rZW5SZXNwb25zZRJeCg1EZWxldGVBY2NvdW50EiUuYW55'
    'dHR5LmNsb3VkLnYxLkRlbGV0ZUFjY291bnRSZXF1ZXN0GiYuYW55dHR5LmNsb3VkLnYxLkRlbG'
    'V0ZUFjY291bnRSZXNwb25zZQ==');
