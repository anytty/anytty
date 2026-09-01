// This is a generated file - do not edit.
//
// Generated from cloud/v1/enrollment.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'enrollment.pb.dart' as $2;
import 'enrollment.pbjson.dart';

export 'enrollment.pb.dart';

abstract class EnrollmentServiceBase extends $pb.GeneratedService {
  $async.Future<$2.DaemonEnrollmentChallenge> beginDaemonEnrollment(
      $pb.ServerContext ctx, $2.BeginDaemonEnrollmentRequest request);
  $async.Future<$2.CompleteDaemonEnrollmentResponse> completeDaemonEnrollment(
      $pb.ServerContext ctx, $2.CompleteDaemonEnrollmentRequest request);
  $async.Future<$2.IdentityChallenge> beginDaemonBindingRefresh(
      $pb.ServerContext ctx, $2.BeginDaemonBindingRefreshRequest request);
  $async.Future<$2.RefreshDaemonBindingResponse> completeDaemonBindingRefresh(
      $pb.ServerContext ctx, $2.CompleteDaemonBindingRefreshRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'BeginDaemonEnrollment':
        return $2.BeginDaemonEnrollmentRequest();
      case 'CompleteDaemonEnrollment':
        return $2.CompleteDaemonEnrollmentRequest();
      case 'BeginDaemonBindingRefresh':
        return $2.BeginDaemonBindingRefreshRequest();
      case 'CompleteDaemonBindingRefresh':
        return $2.CompleteDaemonBindingRefreshRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'BeginDaemonEnrollment':
        return beginDaemonEnrollment(
            ctx, request as $2.BeginDaemonEnrollmentRequest);
      case 'CompleteDaemonEnrollment':
        return completeDaemonEnrollment(
            ctx, request as $2.CompleteDaemonEnrollmentRequest);
      case 'BeginDaemonBindingRefresh':
        return beginDaemonBindingRefresh(
            ctx, request as $2.BeginDaemonBindingRefreshRequest);
      case 'CompleteDaemonBindingRefresh':
        return completeDaemonBindingRefresh(
            ctx, request as $2.CompleteDaemonBindingRefreshRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      EnrollmentServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => EnrollmentServiceBase$messageJson;
}

abstract class DaemonManagementServiceBase extends $pb.GeneratedService {
  $async.Future<$2.CreateDaemonEnrollmentResponse> createMyEnrollment(
      $pb.ServerContext ctx, $2.CreateMyDaemonEnrollmentRequest request);
  $async.Future<$2.ListMyDaemonsResponse> listMyDaemons(
      $pb.ServerContext ctx, $2.ListMyDaemonsRequest request);
  $async.Future<$2.ChangeMyDaemonStateResponse> changeMyDaemonState(
      $pb.ServerContext ctx, $2.ChangeMyDaemonStateRequest request);
  $async.Future<$2.ListMyDaemonEdgesResponse> listMyDaemonEdges(
      $pb.ServerContext ctx, $2.ListMyDaemonEdgesRequest request);
  $async.Future<$2.ChangeMyDaemonEdgePreferenceResponse>
      changeMyDaemonEdgePreference($pb.ServerContext ctx,
          $2.ChangeMyDaemonEdgePreferenceRequest request);
  $async.Future<$2.ReselectMyDaemonEdgeResponse> reselectMyDaemonEdge(
      $pb.ServerContext ctx, $2.ReselectMyDaemonEdgeRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'CreateMyEnrollment':
        return $2.CreateMyDaemonEnrollmentRequest();
      case 'ListMyDaemons':
        return $2.ListMyDaemonsRequest();
      case 'ChangeMyDaemonState':
        return $2.ChangeMyDaemonStateRequest();
      case 'ListMyDaemonEdges':
        return $2.ListMyDaemonEdgesRequest();
      case 'ChangeMyDaemonEdgePreference':
        return $2.ChangeMyDaemonEdgePreferenceRequest();
      case 'ReselectMyDaemonEdge':
        return $2.ReselectMyDaemonEdgeRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'CreateMyEnrollment':
        return createMyEnrollment(
            ctx, request as $2.CreateMyDaemonEnrollmentRequest);
      case 'ListMyDaemons':
        return listMyDaemons(ctx, request as $2.ListMyDaemonsRequest);
      case 'ChangeMyDaemonState':
        return changeMyDaemonState(
            ctx, request as $2.ChangeMyDaemonStateRequest);
      case 'ListMyDaemonEdges':
        return listMyDaemonEdges(ctx, request as $2.ListMyDaemonEdgesRequest);
      case 'ChangeMyDaemonEdgePreference':
        return changeMyDaemonEdgePreference(
            ctx, request as $2.ChangeMyDaemonEdgePreferenceRequest);
      case 'ReselectMyDaemonEdge':
        return reselectMyDaemonEdge(
            ctx, request as $2.ReselectMyDaemonEdgeRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      DaemonManagementServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => DaemonManagementServiceBase$messageJson;
}
