// This is a generated file - do not edit.
//
// Generated from cloud/v1/directory.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'directory.pb.dart' as $3;
import 'directory.pbjson.dart';
import 'enrollment.pb.dart' as $1;

export 'directory.pb.dart';

abstract class DirectoryServiceBase extends $pb.GeneratedService {
  $async.Future<$1.IdentityChallenge> beginClientRoute(
      $pb.ServerContext ctx, $3.BeginClientRouteRequest request);
  $async.Future<$3.ResolveClientRouteResponse> resolveClientRoute(
      $pb.ServerContext ctx, $3.ResolveClientRouteRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'BeginClientRoute':
        return $3.BeginClientRouteRequest();
      case 'ResolveClientRoute':
        return $3.ResolveClientRouteRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'BeginClientRoute':
        return beginClientRoute(ctx, request as $3.BeginClientRouteRequest);
      case 'ResolveClientRoute':
        return resolveClientRoute(ctx, request as $3.ResolveClientRouteRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => DirectoryServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => DirectoryServiceBase$messageJson;
}
