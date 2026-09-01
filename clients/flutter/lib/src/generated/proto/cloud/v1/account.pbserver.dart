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

import 'package:protobuf/protobuf.dart' as $pb;

import 'account.pb.dart' as $1;
import 'account.pbjson.dart';

export 'account.pb.dart';

abstract class AccountServiceBase extends $pb.GeneratedService {
  $async.Future<$1.LoginAccountResponse> login(
      $pb.ServerContext ctx, $1.LoginAccountRequest request);
  $async.Future<$1.RefreshAccountTokenResponse> refresh(
      $pb.ServerContext ctx, $1.RefreshAccountTokenRequest request);
  $async.Future<$1.LogoutAccountResponse> logout(
      $pb.ServerContext ctx, $1.LogoutAccountRequest request);
  $async.Future<$1.GetCurrentAccountResponse> getCurrent(
      $pb.ServerContext ctx, $1.GetCurrentAccountRequest request);
  $async.Future<$1.VerifyRecentAuthenticationResponse>
      verifyRecentAuthentication(
          $pb.ServerContext ctx, $1.VerifyRecentAuthenticationRequest request);
  $async.Future<$1.ListAccountRefreshTokensResponse> listRefreshTokens(
      $pb.ServerContext ctx, $1.ListAccountRefreshTokensRequest request);
  $async.Future<$1.ChangeAccountPasswordResponse> changePassword(
      $pb.ServerContext ctx, $1.ChangeAccountPasswordRequest request);
  $async.Future<$1.RedeemAccountSetupResponse> redeemAccountSetup(
      $pb.ServerContext ctx, $1.RedeemAccountSetupRequest request);
  $async.Future<$1.RevokeAccountRefreshTokenResponse> revokeRefreshToken(
      $pb.ServerContext ctx, $1.RevokeAccountRefreshTokenRequest request);
  $async.Future<$1.DeleteAccountResponse> deleteAccount(
      $pb.ServerContext ctx, $1.DeleteAccountRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'Login':
        return $1.LoginAccountRequest();
      case 'Refresh':
        return $1.RefreshAccountTokenRequest();
      case 'Logout':
        return $1.LogoutAccountRequest();
      case 'GetCurrent':
        return $1.GetCurrentAccountRequest();
      case 'VerifyRecentAuthentication':
        return $1.VerifyRecentAuthenticationRequest();
      case 'ListRefreshTokens':
        return $1.ListAccountRefreshTokensRequest();
      case 'ChangePassword':
        return $1.ChangeAccountPasswordRequest();
      case 'RedeemAccountSetup':
        return $1.RedeemAccountSetupRequest();
      case 'RevokeRefreshToken':
        return $1.RevokeAccountRefreshTokenRequest();
      case 'DeleteAccount':
        return $1.DeleteAccountRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'Login':
        return login(ctx, request as $1.LoginAccountRequest);
      case 'Refresh':
        return refresh(ctx, request as $1.RefreshAccountTokenRequest);
      case 'Logout':
        return logout(ctx, request as $1.LogoutAccountRequest);
      case 'GetCurrent':
        return getCurrent(ctx, request as $1.GetCurrentAccountRequest);
      case 'VerifyRecentAuthentication':
        return verifyRecentAuthentication(
            ctx, request as $1.VerifyRecentAuthenticationRequest);
      case 'ListRefreshTokens':
        return listRefreshTokens(
            ctx, request as $1.ListAccountRefreshTokensRequest);
      case 'ChangePassword':
        return changePassword(ctx, request as $1.ChangeAccountPasswordRequest);
      case 'RedeemAccountSetup':
        return redeemAccountSetup(ctx, request as $1.RedeemAccountSetupRequest);
      case 'RevokeRefreshToken':
        return revokeRefreshToken(
            ctx, request as $1.RevokeAccountRefreshTokenRequest);
      case 'DeleteAccount':
        return deleteAccount(ctx, request as $1.DeleteAccountRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => AccountServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => AccountServiceBase$messageJson;
}
