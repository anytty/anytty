// This is a generated file - do not edit.
//
// Generated from cloud/v1/runtime.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// ClientProduct 表示建立 Cloud 信令连接的官方客户端形态。
class ClientProduct extends $pb.ProtobufEnum {
  static const ClientProduct CLIENT_PRODUCT_UNSPECIFIED =
      ClientProduct._(0, _omitEnumNames ? '' : 'CLIENT_PRODUCT_UNSPECIFIED');
  static const ClientProduct CLIENT_PRODUCT_TUI =
      ClientProduct._(1, _omitEnumNames ? '' : 'CLIENT_PRODUCT_TUI');
  static const ClientProduct CLIENT_PRODUCT_CLI =
      ClientProduct._(2, _omitEnumNames ? '' : 'CLIENT_PRODUCT_CLI');
  static const ClientProduct CLIENT_PRODUCT_ANDROID =
      ClientProduct._(3, _omitEnumNames ? '' : 'CLIENT_PRODUCT_ANDROID');
  static const ClientProduct CLIENT_PRODUCT_IOS =
      ClientProduct._(4, _omitEnumNames ? '' : 'CLIENT_PRODUCT_IOS');
  static const ClientProduct CLIENT_PRODUCT_DESKTOP_GUI =
      ClientProduct._(5, _omitEnumNames ? '' : 'CLIENT_PRODUCT_DESKTOP_GUI');

  static const $core.List<ClientProduct> values = <ClientProduct>[
    CLIENT_PRODUCT_UNSPECIFIED,
    CLIENT_PRODUCT_TUI,
    CLIENT_PRODUCT_CLI,
    CLIENT_PRODUCT_ANDROID,
    CLIENT_PRODUCT_IOS,
    CLIENT_PRODUCT_DESKTOP_GUI,
  ];

  static final $core.List<ClientProduct?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static ClientProduct? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ClientProduct._(super.value, super.name);
}

/// CloudClientAccessMode 区分普通 capability signaling 与只允许首次配对的 bootstrap signaling。
/// PAIRING 只负责建立 PairingExchange DataChannel，不能表达 terminal 权限。
class CloudClientAccessMode extends $pb.ProtobufEnum {
  static const CloudClientAccessMode CLOUD_CLIENT_ACCESS_MODE_UNSPECIFIED =
      CloudClientAccessMode._(
          0, _omitEnumNames ? '' : 'CLOUD_CLIENT_ACCESS_MODE_UNSPECIFIED');
  static const CloudClientAccessMode CLOUD_CLIENT_ACCESS_MODE_CAPABILITY =
      CloudClientAccessMode._(
          1, _omitEnumNames ? '' : 'CLOUD_CLIENT_ACCESS_MODE_CAPABILITY');
  static const CloudClientAccessMode CLOUD_CLIENT_ACCESS_MODE_PAIRING =
      CloudClientAccessMode._(
          2, _omitEnumNames ? '' : 'CLOUD_CLIENT_ACCESS_MODE_PAIRING');

  static const $core.List<CloudClientAccessMode> values =
      <CloudClientAccessMode>[
    CLOUD_CLIENT_ACCESS_MODE_UNSPECIFIED,
    CLOUD_CLIENT_ACCESS_MODE_CAPABILITY,
    CLOUD_CLIENT_ACCESS_MODE_PAIRING,
  ];

  static final $core.List<CloudClientAccessMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static CloudClientAccessMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CloudClientAccessMode._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
