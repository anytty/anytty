// This is a generated file - do not edit.
//
// Generated from wirepb/terminal.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// wirepb 只拥有 DataChannel framing 与 file resource stream payload。
/// application command/result/event 必须使用 apipb，不得在本包重新定义。
class Hello extends $pb.GeneratedMessage {
  factory Hello({
    $core.int? version,
    $core.String? client,
    $core.String? server,
  }) {
    final result = create();
    if (version != null) result.version = version;
    if (client != null) result.client = client;
    if (server != null) result.server = server;
    return result;
  }

  Hello._();

  factory Hello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Hello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Hello',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'client')
    ..aOS(3, _omitFieldNames ? '' : 'server')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Hello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Hello copyWith(void Function(Hello) updates) =>
      super.copyWith((message) => updates(message as Hello)) as Hello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Hello create() => Hello._();
  @$core.override
  Hello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Hello getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Hello>(create);
  static Hello? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get version => $_getIZ(0);
  @$pb.TagNumber(1)
  set version($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get client => $_getSZ(1);
  @$pb.TagNumber(2)
  set client($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasClient() => $_has(1);
  @$pb.TagNumber(2)
  void clearClient() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get server => $_getSZ(2);
  @$pb.TagNumber(3)
  set server($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasServer() => $_has(2);
  @$pb.TagNumber(3)
  void clearServer() => $_clearField(3);
}

/// SessionClose 是已完成 Hello 的客户端主动结束当前 protocol session 的单向 control frame。
/// daemon 收到后必须先释放 request/resource 并关闭 transport；它不改变 Endpoint、terminal 或 grant lifecycle。
class SessionClose extends $pb.GeneratedMessage {
  factory SessionClose({
    $core.int? version,
  }) {
    final result = create();
    if (version != null) result.version = version;
    return result;
  }

  SessionClose._();

  factory SessionClose.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SessionClose.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SessionClose',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionClose clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SessionClose copyWith(void Function(SessionClose) updates) =>
      super.copyWith((message) => updates(message as SessionClose))
          as SessionClose;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SessionClose create() => SessionClose._();
  @$core.override
  SessionClose createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SessionClose getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SessionClose>(create);
  static SessionClose? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get version => $_getIZ(0);
  @$pb.TagNumber(1)
  set version($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearVersion() => $_clearField(1);
}

/// RequestCancel cancels one in-flight control request without closing the
/// protocol session. Unknown or already completed IDs are an idempotent no-op.
class RequestCancel extends $pb.GeneratedMessage {
  factory RequestCancel({
    $fixnum.Int64? id,
  }) {
    final result = create();
    if (id != null) result.id = id;
    return result;
  }

  RequestCancel._();

  factory RequestCancel.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RequestCancel.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestCancel',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestCancel clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestCancel copyWith(void Function(RequestCancel) updates) =>
      super.copyWith((message) => updates(message as RequestCancel))
          as RequestCancel;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RequestCancel create() => RequestCancel._();
  @$core.override
  RequestCancel createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RequestCancel getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RequestCancel>(create);
  static RequestCancel? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get id => $_getI64(0);
  @$pb.TagNumber(1)
  set id($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);
}

class RequestEnvelope extends $pb.GeneratedMessage {
  factory RequestEnvelope({
    $fixnum.Int64? id,
    $core.String? method,
    $core.List<$core.int>? params,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (method != null) result.method = method;
    if (params != null) result.params = params;
    return result;
  }

  RequestEnvelope._();

  factory RequestEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RequestEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(2, _omitFieldNames ? '' : 'method')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'params', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestEnvelope copyWith(void Function(RequestEnvelope) updates) =>
      super.copyWith((message) => updates(message as RequestEnvelope))
          as RequestEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RequestEnvelope create() => RequestEnvelope._();
  @$core.override
  RequestEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RequestEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RequestEnvelope>(create);
  static RequestEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get id => $_getI64(0);
  @$pb.TagNumber(1)
  set id($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get method => $_getSZ(1);
  @$pb.TagNumber(2)
  set method($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMethod() => $_has(1);
  @$pb.TagNumber(2)
  void clearMethod() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get params => $_getN(2);
  @$pb.TagNumber(3)
  set params($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasParams() => $_has(2);
  @$pb.TagNumber(3)
  void clearParams() => $_clearField(3);
}

class ResponseEnvelope extends $pb.GeneratedMessage {
  factory ResponseEnvelope({
    $fixnum.Int64? id,
    $core.List<$core.int>? result,
  }) {
    final result$ = create();
    if (id != null) result$.id = id;
    if (result != null) result$.result = result;
    return result$;
  }

  ResponseEnvelope._();

  factory ResponseEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResponseEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResponseEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'result', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResponseEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResponseEnvelope copyWith(void Function(ResponseEnvelope) updates) =>
      super.copyWith((message) => updates(message as ResponseEnvelope))
          as ResponseEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResponseEnvelope create() => ResponseEnvelope._();
  @$core.override
  ResponseEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResponseEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResponseEnvelope>(create);
  static ResponseEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get id => $_getI64(0);
  @$pb.TagNumber(1)
  set id($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get result => $_getN(1);
  @$pb.TagNumber(2)
  set result($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasResult() => $_has(1);
  @$pb.TagNumber(2)
  void clearResult() => $_clearField(2);
}

class ProtocolError extends $pb.GeneratedMessage {
  factory ProtocolError({
    $core.int? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  ProtocolError._();

  factory ProtocolError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProtocolError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProtocolError',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'code')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProtocolError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProtocolError copyWith(void Function(ProtocolError) updates) =>
      super.copyWith((message) => updates(message as ProtocolError))
          as ProtocolError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProtocolError create() => ProtocolError._();
  @$core.override
  ProtocolError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProtocolError getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProtocolError>(create);
  static ProtocolError? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get code => $_getIZ(0);
  @$pb.TagNumber(1)
  set code($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class ErrorEnvelope extends $pb.GeneratedMessage {
  factory ErrorEnvelope({
    $fixnum.Int64? id,
    ProtocolError? error,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (error != null) result.error = error;
    return result;
  }

  ErrorEnvelope._();

  factory ErrorEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ErrorEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ErrorEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ProtocolError>(2, _omitFieldNames ? '' : 'error',
        subBuilder: ProtocolError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ErrorEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ErrorEnvelope copyWith(void Function(ErrorEnvelope) updates) =>
      super.copyWith((message) => updates(message as ErrorEnvelope))
          as ErrorEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ErrorEnvelope create() => ErrorEnvelope._();
  @$core.override
  ErrorEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ErrorEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ErrorEnvelope>(create);
  static ErrorEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get id => $_getI64(0);
  @$pb.TagNumber(1)
  set id($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  ProtocolError get error => $_getN(1);
  @$pb.TagNumber(2)
  set error(ProtocolError value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
  @$pb.TagNumber(2)
  ProtocolError ensureError() => $_ensure(1);
}

class FileTransferData extends $pb.GeneratedMessage {
  factory FileTransferData({
    $fixnum.Int64? offset,
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (offset != null) result.offset = offset;
    if (data != null) result.data = data;
    return result;
  }

  FileTransferData._();

  factory FileTransferData.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FileTransferData.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferData',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'offset')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferData clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferData copyWith(void Function(FileTransferData) updates) =>
      super.copyWith((message) => updates(message as FileTransferData))
          as FileTransferData;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FileTransferData create() => FileTransferData._();
  @$core.override
  FileTransferData createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FileTransferData getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferData>(create);
  static FileTransferData? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get offset => $_getI64(0);
  @$pb.TagNumber(1)
  set offset($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOffset() => $_has(0);
  @$pb.TagNumber(1)
  void clearOffset() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get data => $_getN(1);
  @$pb.TagNumber(2)
  set data($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasData() => $_has(1);
  @$pb.TagNumber(2)
  void clearData() => $_clearField(2);
}

class FileTransferAck extends $pb.GeneratedMessage {
  factory FileTransferAck({
    $fixnum.Int64? offset,
    $fixnum.Int64? windowBytes,
  }) {
    final result = create();
    if (offset != null) result.offset = offset;
    if (windowBytes != null) result.windowBytes = windowBytes;
    return result;
  }

  FileTransferAck._();

  factory FileTransferAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FileTransferAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferAck',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'offset')
    ..aInt64(2, _omitFieldNames ? '' : 'windowBytes')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferAck copyWith(void Function(FileTransferAck) updates) =>
      super.copyWith((message) => updates(message as FileTransferAck))
          as FileTransferAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FileTransferAck create() => FileTransferAck._();
  @$core.override
  FileTransferAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FileTransferAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferAck>(create);
  static FileTransferAck? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get offset => $_getI64(0);
  @$pb.TagNumber(1)
  set offset($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOffset() => $_has(0);
  @$pb.TagNumber(1)
  void clearOffset() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get windowBytes => $_getI64(1);
  @$pb.TagNumber(2)
  set windowBytes($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWindowBytes() => $_has(1);
  @$pb.TagNumber(2)
  void clearWindowBytes() => $_clearField(2);
}

class FileTransferFinish extends $pb.GeneratedMessage {
  factory FileTransferFinish({
    $fixnum.Int64? size,
    $core.List<$core.int>? sha256,
  }) {
    final result = create();
    if (size != null) result.size = size;
    if (sha256 != null) result.sha256 = sha256;
    return result;
  }

  FileTransferFinish._();

  factory FileTransferFinish.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FileTransferFinish.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferFinish',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'size')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'sha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferFinish clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferFinish copyWith(void Function(FileTransferFinish) updates) =>
      super.copyWith((message) => updates(message as FileTransferFinish))
          as FileTransferFinish;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FileTransferFinish create() => FileTransferFinish._();
  @$core.override
  FileTransferFinish createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FileTransferFinish getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferFinish>(create);
  static FileTransferFinish? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get size => $_getI64(0);
  @$pb.TagNumber(1)
  set size($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSize() => $_has(0);
  @$pb.TagNumber(1)
  void clearSize() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get sha256 => $_getN(1);
  @$pb.TagNumber(2)
  set sha256($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSha256() => $_has(1);
  @$pb.TagNumber(2)
  void clearSha256() => $_clearField(2);
}

class FileTransferResult extends $pb.GeneratedMessage {
  factory FileTransferResult({
    $core.String? path,
    $fixnum.Int64? size,
    $core.List<$core.int>? sha256,
  }) {
    final result = create();
    if (path != null) result.path = path;
    if (size != null) result.size = size;
    if (sha256 != null) result.sha256 = sha256;
    return result;
  }

  FileTransferResult._();

  factory FileTransferResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FileTransferResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'path')
    ..aInt64(2, _omitFieldNames ? '' : 'size')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'sha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferResult copyWith(void Function(FileTransferResult) updates) =>
      super.copyWith((message) => updates(message as FileTransferResult))
          as FileTransferResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FileTransferResult create() => FileTransferResult._();
  @$core.override
  FileTransferResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FileTransferResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferResult>(create);
  static FileTransferResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(1)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(1)
  void clearPath() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get size => $_getI64(1);
  @$pb.TagNumber(2)
  set size($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSize() => $_has(1);
  @$pb.TagNumber(2)
  void clearSize() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get sha256 => $_getN(2);
  @$pb.TagNumber(3)
  set sha256($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSha256() => $_has(2);
  @$pb.TagNumber(3)
  void clearSha256() => $_clearField(3);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
