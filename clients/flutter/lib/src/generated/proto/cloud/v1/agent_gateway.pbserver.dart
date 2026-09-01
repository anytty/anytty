// This is a generated file - do not edit.
//
// Generated from cloud/v1/agent_gateway.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'agent_gateway.pb.dart' as $7;
import 'agent_gateway.pbjson.dart';

export 'agent_gateway.pb.dart';

abstract class AgentGatewayServiceBase extends $pb.GeneratedService {
  $async.Future<$7.EdgeCommand> connect(
      $pb.ServerContext ctx, $7.AgentEvent request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'Connect':
        return $7.AgentEvent();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'Connect':
        return connect(ctx, request as $7.AgentEvent);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      AgentGatewayServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => AgentGatewayServiceBase$messageJson;
}
