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

import 'package:protobuf/protobuf.dart' as $pb;

import 'edge_config.pb.dart' as $3;
import 'operator.pb.dart' as $5;
import 'operator.pbjson.dart';

export 'operator.pb.dart';

abstract class OperatorServiceBase extends $pb.GeneratedService {
  $async.Future<$5.GetOperatorOverviewResponse> getOverview(
      $pb.ServerContext ctx, $5.GetOperatorOverviewRequest request);
  $async.Future<$5.ListOperatorAccountsResponse> listAccounts(
      $pb.ServerContext ctx, $5.ListOperatorAccountsRequest request);
  $async.Future<$5.GetOperatorAccountResponse> getAccount(
      $pb.ServerContext ctx, $5.GetOperatorAccountRequest request);
  $async.Future<$5.ProvisionAccountResponse> provisionAccount(
      $pb.ServerContext ctx, $5.ProvisionAccountRequest request);
  $async.Future<$5.ResetAccountSetupResponse> resetAccountSetup(
      $pb.ServerContext ctx, $5.ResetAccountSetupRequest request);
  $async.Future<$5.ListRuntimeSessionsResponse> listRuntimeSessions(
      $pb.ServerContext ctx, $5.ListRuntimeSessionsRequest request);
  $async.Future<$5.ListOperatorOrdersResponse> listOrders(
      $pb.ServerContext ctx, $5.ListOperatorOrdersRequest request);
  $async.Future<$5.ListOperatorSubscriptionsResponse> listSubscriptions(
      $pb.ServerContext ctx, $5.ListOperatorSubscriptionsRequest request);
  $async.Future<$5.ListOperatorUsageResponse> listUsage(
      $pb.ServerContext ctx, $5.ListOperatorUsageRequest request);
  $async.Future<$5.ListOperatorAuditResponse> listAudit(
      $pb.ServerContext ctx, $5.ListOperatorAuditRequest request);
  $async.Future<$5.SetAccountStateResponse> setAccountState(
      $pb.ServerContext ctx, $5.SetAccountStateRequest request);
  $async.Future<$5.SetAccountRoleResponse> setAccountRole(
      $pb.ServerContext ctx, $5.SetAccountRoleRequest request);
  $async.Future<$5.DisconnectDaemonResponse> disconnectDaemon(
      $pb.ServerContext ctx, $5.DisconnectDaemonRequest request);
  $async.Future<$5.DisconnectSessionResponse> disconnectSession(
      $pb.ServerContext ctx, $5.DisconnectSessionRequest request);
  $async.Future<$3.DeleteEdgeResponse> deleteEdge(
      $pb.ServerContext ctx, $3.DeleteEdgeRequest request);
  $async.Future<$3.CreateEdgeIdentityRecoveryResponse>
      createEdgeIdentityRecovery(
          $pb.ServerContext ctx, $3.CreateEdgeIdentityRecoveryRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'GetOverview':
        return $5.GetOperatorOverviewRequest();
      case 'ListAccounts':
        return $5.ListOperatorAccountsRequest();
      case 'GetAccount':
        return $5.GetOperatorAccountRequest();
      case 'ProvisionAccount':
        return $5.ProvisionAccountRequest();
      case 'ResetAccountSetup':
        return $5.ResetAccountSetupRequest();
      case 'ListRuntimeSessions':
        return $5.ListRuntimeSessionsRequest();
      case 'ListOrders':
        return $5.ListOperatorOrdersRequest();
      case 'ListSubscriptions':
        return $5.ListOperatorSubscriptionsRequest();
      case 'ListUsage':
        return $5.ListOperatorUsageRequest();
      case 'ListAudit':
        return $5.ListOperatorAuditRequest();
      case 'SetAccountState':
        return $5.SetAccountStateRequest();
      case 'SetAccountRole':
        return $5.SetAccountRoleRequest();
      case 'DisconnectDaemon':
        return $5.DisconnectDaemonRequest();
      case 'DisconnectSession':
        return $5.DisconnectSessionRequest();
      case 'DeleteEdge':
        return $3.DeleteEdgeRequest();
      case 'CreateEdgeIdentityRecovery':
        return $3.CreateEdgeIdentityRecoveryRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'GetOverview':
        return getOverview(ctx, request as $5.GetOperatorOverviewRequest);
      case 'ListAccounts':
        return listAccounts(ctx, request as $5.ListOperatorAccountsRequest);
      case 'GetAccount':
        return getAccount(ctx, request as $5.GetOperatorAccountRequest);
      case 'ProvisionAccount':
        return provisionAccount(ctx, request as $5.ProvisionAccountRequest);
      case 'ResetAccountSetup':
        return resetAccountSetup(ctx, request as $5.ResetAccountSetupRequest);
      case 'ListRuntimeSessions':
        return listRuntimeSessions(
            ctx, request as $5.ListRuntimeSessionsRequest);
      case 'ListOrders':
        return listOrders(ctx, request as $5.ListOperatorOrdersRequest);
      case 'ListSubscriptions':
        return listSubscriptions(
            ctx, request as $5.ListOperatorSubscriptionsRequest);
      case 'ListUsage':
        return listUsage(ctx, request as $5.ListOperatorUsageRequest);
      case 'ListAudit':
        return listAudit(ctx, request as $5.ListOperatorAuditRequest);
      case 'SetAccountState':
        return setAccountState(ctx, request as $5.SetAccountStateRequest);
      case 'SetAccountRole':
        return setAccountRole(ctx, request as $5.SetAccountRoleRequest);
      case 'DisconnectDaemon':
        return disconnectDaemon(ctx, request as $5.DisconnectDaemonRequest);
      case 'DisconnectSession':
        return disconnectSession(ctx, request as $5.DisconnectSessionRequest);
      case 'DeleteEdge':
        return deleteEdge(ctx, request as $3.DeleteEdgeRequest);
      case 'CreateEdgeIdentityRecovery':
        return createEdgeIdentityRecovery(
            ctx, request as $3.CreateEdgeIdentityRecoveryRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => OperatorServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => OperatorServiceBase$messageJson;
}
