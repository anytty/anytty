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

import 'package:protobuf/protobuf.dart' as $pb;

import 'commerce.pb.dart' as $1;
import 'commerce.pbjson.dart';

export 'commerce.pb.dart';

abstract class CommerceServiceBase extends $pb.GeneratedService {
  $async.Future<$1.ListPlansResponse> listPlans(
      $pb.ServerContext ctx, $1.ListPlansRequest request);
  $async.Future<$1.CreatePlanVersionResponse> createPlanVersion(
      $pb.ServerContext ctx, $1.CreatePlanVersionRequest request);
  $async.Future<$1.PublishPlanVersionResponse> publishPlanVersion(
      $pb.ServerContext ctx, $1.PublishPlanVersionRequest request);
  $async.Future<$1.CreateOrderResponse> createOrder(
      $pb.ServerContext ctx, $1.CreateOrderRequest request);
  $async.Future<$1.ApplyPaymentEventResponse> applyPaymentEvent(
      $pb.ServerContext ctx, $1.ApplyPaymentEventRequest request);
  $async.Future<$1.TransitionSubscriptionResponse> transitionSubscription(
      $pb.ServerContext ctx, $1.TransitionSubscriptionRequest request);
  $async.Future<$1.GetAccountCommerceResponse> getAccountCommerce(
      $pb.ServerContext ctx, $1.GetAccountCommerceRequest request);
  $async.Future<$1.CreateOrderResponse> createMyOrder(
      $pb.ServerContext ctx, $1.CreateMyOrderRequest request);
  $async.Future<$1.GetAccountCommerceResponse> getMyCommerce(
      $pb.ServerContext ctx, $1.GetMyCommerceRequest request);
  $async.Future<$1.TransitionSubscriptionResponse> changeMySubscription(
      $pb.ServerContext ctx, $1.ChangeMySubscriptionRequest request);
  $async.Future<$1.ApplyPaymentEventResponse> completeDevelopmentPayment(
      $pb.ServerContext ctx, $1.CompleteDevelopmentPaymentRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'ListPlans':
        return $1.ListPlansRequest();
      case 'CreatePlanVersion':
        return $1.CreatePlanVersionRequest();
      case 'PublishPlanVersion':
        return $1.PublishPlanVersionRequest();
      case 'CreateOrder':
        return $1.CreateOrderRequest();
      case 'ApplyPaymentEvent':
        return $1.ApplyPaymentEventRequest();
      case 'TransitionSubscription':
        return $1.TransitionSubscriptionRequest();
      case 'GetAccountCommerce':
        return $1.GetAccountCommerceRequest();
      case 'CreateMyOrder':
        return $1.CreateMyOrderRequest();
      case 'GetMyCommerce':
        return $1.GetMyCommerceRequest();
      case 'ChangeMySubscription':
        return $1.ChangeMySubscriptionRequest();
      case 'CompleteDevelopmentPayment':
        return $1.CompleteDevelopmentPaymentRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'ListPlans':
        return listPlans(ctx, request as $1.ListPlansRequest);
      case 'CreatePlanVersion':
        return createPlanVersion(ctx, request as $1.CreatePlanVersionRequest);
      case 'PublishPlanVersion':
        return publishPlanVersion(ctx, request as $1.PublishPlanVersionRequest);
      case 'CreateOrder':
        return createOrder(ctx, request as $1.CreateOrderRequest);
      case 'ApplyPaymentEvent':
        return applyPaymentEvent(ctx, request as $1.ApplyPaymentEventRequest);
      case 'TransitionSubscription':
        return transitionSubscription(
            ctx, request as $1.TransitionSubscriptionRequest);
      case 'GetAccountCommerce':
        return getAccountCommerce(ctx, request as $1.GetAccountCommerceRequest);
      case 'CreateMyOrder':
        return createMyOrder(ctx, request as $1.CreateMyOrderRequest);
      case 'GetMyCommerce':
        return getMyCommerce(ctx, request as $1.GetMyCommerceRequest);
      case 'ChangeMySubscription':
        return changeMySubscription(
            ctx, request as $1.ChangeMySubscriptionRequest);
      case 'CompleteDevelopmentPayment':
        return completeDevelopmentPayment(
            ctx, request as $1.CompleteDevelopmentPaymentRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => CommerceServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => CommerceServiceBase$messageJson;
}
