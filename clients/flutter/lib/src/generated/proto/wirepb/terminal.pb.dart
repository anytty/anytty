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
    final result = Hello._();
    if (version != null) result.version = version;
    if (client != null) result.client = client;
    if (server != null) result.server = server;
    return result;
  }

  Hello._();

  factory Hello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      Hello()..mergeFromBuffer(data, registry);
  factory Hello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      Hello()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Hello',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: Hello.$_createMessage)
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
  @$core.Deprecated('Use Hello() / Hello.new instead')
  static Hello create() => Hello._();
  static $pb.GeneratedMessage $_createMessage() => Hello._();
  @$core.override
  Hello createEmptyInstance() => Hello._();
  @$core.pragma('dart2js:noInline')
  static Hello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Hello>(Hello.$_createMessage);
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
    final result = SessionClose._();
    if (version != null) result.version = version;
    return result;
  }

  SessionClose._();

  factory SessionClose.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SessionClose()..mergeFromBuffer(data, registry);
  factory SessionClose.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SessionClose()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SessionClose',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: SessionClose.$_createMessage)
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
  @$core.Deprecated('Use SessionClose() / SessionClose.new instead')
  static SessionClose create() => SessionClose._();
  static $pb.GeneratedMessage $_createMessage() => SessionClose._();
  @$core.override
  SessionClose createEmptyInstance() => SessionClose._();
  @$core.pragma('dart2js:noInline')
  static SessionClose getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SessionClose>(
          SessionClose.$_createMessage);
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
    final result = RequestCancel._();
    if (id != null) result.id = id;
    return result;
  }

  RequestCancel._();

  factory RequestCancel.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RequestCancel()..mergeFromBuffer(data, registry);
  factory RequestCancel.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RequestCancel()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestCancel',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: RequestCancel.$_createMessage)
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
  @$core.Deprecated('Use RequestCancel() / RequestCancel.new instead')
  static RequestCancel create() => RequestCancel._();
  static $pb.GeneratedMessage $_createMessage() => RequestCancel._();
  @$core.override
  RequestCancel createEmptyInstance() => RequestCancel._();
  @$core.pragma('dart2js:noInline')
  static RequestCancel getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RequestCancel>(
          RequestCancel.$_createMessage);
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
    final result = RequestEnvelope._();
    if (id != null) result.id = id;
    if (method != null) result.method = method;
    if (params != null) result.params = params;
    return result;
  }

  RequestEnvelope._();

  factory RequestEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RequestEnvelope()..mergeFromBuffer(data, registry);
  factory RequestEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RequestEnvelope()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: RequestEnvelope.$_createMessage)
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
  @$core.Deprecated('Use RequestEnvelope() / RequestEnvelope.new instead')
  static RequestEnvelope create() => RequestEnvelope._();
  static $pb.GeneratedMessage $_createMessage() => RequestEnvelope._();
  @$core.override
  RequestEnvelope createEmptyInstance() => RequestEnvelope._();
  @$core.pragma('dart2js:noInline')
  static RequestEnvelope getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RequestEnvelope>(
          RequestEnvelope.$_createMessage);
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
    final result$ = ResponseEnvelope._();
    if (id != null) result$.id = id;
    if (result != null) result$.result = result;
    return result$;
  }

  ResponseEnvelope._();

  factory ResponseEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ResponseEnvelope()..mergeFromBuffer(data, registry);
  factory ResponseEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ResponseEnvelope()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResponseEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: ResponseEnvelope.$_createMessage)
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
  @$core.Deprecated('Use ResponseEnvelope() / ResponseEnvelope.new instead')
  static ResponseEnvelope create() => ResponseEnvelope._();
  static $pb.GeneratedMessage $_createMessage() => ResponseEnvelope._();
  @$core.override
  ResponseEnvelope createEmptyInstance() => ResponseEnvelope._();
  @$core.pragma('dart2js:noInline')
  static ResponseEnvelope getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ResponseEnvelope>(
          ResponseEnvelope.$_createMessage);
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
    final result = ProtocolError._();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  ProtocolError._();

  factory ProtocolError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ProtocolError()..mergeFromBuffer(data, registry);
  factory ProtocolError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ProtocolError()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProtocolError',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: ProtocolError.$_createMessage)
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
  @$core.Deprecated('Use ProtocolError() / ProtocolError.new instead')
  static ProtocolError create() => ProtocolError._();
  static $pb.GeneratedMessage $_createMessage() => ProtocolError._();
  @$core.override
  ProtocolError createEmptyInstance() => ProtocolError._();
  @$core.pragma('dart2js:noInline')
  static ProtocolError getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ProtocolError>(
          ProtocolError.$_createMessage);
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
    final result = ErrorEnvelope._();
    if (id != null) result.id = id;
    if (error != null) result.error = error;
    return result;
  }

  ErrorEnvelope._();

  factory ErrorEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ErrorEnvelope()..mergeFromBuffer(data, registry);
  factory ErrorEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ErrorEnvelope()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ErrorEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: ErrorEnvelope.$_createMessage)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ProtocolError>(2, _omitFieldNames ? '' : 'error',
        subBuilder: ProtocolError.$_createMessage)
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
  @$core.Deprecated('Use ErrorEnvelope() / ErrorEnvelope.new instead')
  static ErrorEnvelope create() => ErrorEnvelope._();
  static $pb.GeneratedMessage $_createMessage() => ErrorEnvelope._();
  @$core.override
  ErrorEnvelope createEmptyInstance() => ErrorEnvelope._();
  @$core.pragma('dart2js:noInline')
  static ErrorEnvelope getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ErrorEnvelope>(
          ErrorEnvelope.$_createMessage);
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
    $core.String? encoding,
  }) {
    final result = FileTransferData._();
    if (offset != null) result.offset = offset;
    if (data != null) result.data = data;
    if (encoding != null) result.encoding = encoding;
    return result;
  }

  FileTransferData._();

  factory FileTransferData.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferData()..mergeFromBuffer(data, registry);
  factory FileTransferData.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferData()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferData',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: FileTransferData.$_createMessage)
    ..aInt64(1, _omitFieldNames ? '' : 'offset')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'encoding')
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
  @$core.Deprecated('Use FileTransferData() / FileTransferData.new instead')
  static FileTransferData create() => FileTransferData._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferData._();
  @$core.override
  FileTransferData createEmptyInstance() => FileTransferData._();
  @$core.pragma('dart2js:noInline')
  static FileTransferData getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileTransferData>(
          FileTransferData.$_createMessage);
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

  /// encoding 是当前帧 data 的编码（""=identity，"zstd"=独立 zstd frame）。
  /// 旧接收端忽略该字段；未协商压缩时始终为空，payload 字节不变。
  @$pb.TagNumber(3)
  $core.String get encoding => $_getSZ(2);
  @$pb.TagNumber(3)
  set encoding($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEncoding() => $_has(2);
  @$pb.TagNumber(3)
  void clearEncoding() => $_clearField(3);
}

class FileTransferAck extends $pb.GeneratedMessage {
  factory FileTransferAck({
    $fixnum.Int64? offset,
    $fixnum.Int64? windowBytes,
    $fixnum.Int64? transferredBytes,
    $fixnum.Int64? totalBytes,
    $fixnum.Int64? elapsedMillis,
  }) {
    final result = FileTransferAck._();
    if (offset != null) result.offset = offset;
    if (windowBytes != null) result.windowBytes = windowBytes;
    if (transferredBytes != null) result.transferredBytes = transferredBytes;
    if (totalBytes != null) result.totalBytes = totalBytes;
    if (elapsedMillis != null) result.elapsedMillis = elapsedMillis;
    return result;
  }

  FileTransferAck._();

  factory FileTransferAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferAck()..mergeFromBuffer(data, registry);
  factory FileTransferAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferAck()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferAck',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: FileTransferAck.$_createMessage)
    ..aInt64(1, _omitFieldNames ? '' : 'offset')
    ..aInt64(2, _omitFieldNames ? '' : 'windowBytes')
    ..aInt64(3, _omitFieldNames ? '' : 'transferredBytes')
    ..aInt64(4, _omitFieldNames ? '' : 'totalBytes')
    ..aInt64(5, _omitFieldNames ? '' : 'elapsedMillis')
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
  @$core.Deprecated('Use FileTransferAck() / FileTransferAck.new instead')
  static FileTransferAck create() => FileTransferAck._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferAck._();
  @$core.override
  FileTransferAck createEmptyInstance() => FileTransferAck._();
  @$core.pragma('dart2js:noInline')
  static FileTransferAck getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileTransferAck>(
          FileTransferAck.$_createMessage);
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

  /// 以下为可选结构化进度：transferred_bytes/total_bytes 是发送方的权威进度，
  /// elapsed_millis 是 transfer 打开以来的毫秒数。0 = 未启用（旧行为的精确字节）。
  @$pb.TagNumber(3)
  $fixnum.Int64 get transferredBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set transferredBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTransferredBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearTransferredBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get totalBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set totalBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTotalBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearTotalBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get elapsedMillis => $_getI64(4);
  @$pb.TagNumber(5)
  set elapsedMillis($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasElapsedMillis() => $_has(4);
  @$pb.TagNumber(5)
  void clearElapsedMillis() => $_clearField(5);
}

class FileTransferFinish extends $pb.GeneratedMessage {
  factory FileTransferFinish({
    $fixnum.Int64? size,
    $core.List<$core.int>? sha256,
    $fixnum.Int64? elapsedMillis,
  }) {
    final result = FileTransferFinish._();
    if (size != null) result.size = size;
    if (sha256 != null) result.sha256 = sha256;
    if (elapsedMillis != null) result.elapsedMillis = elapsedMillis;
    return result;
  }

  FileTransferFinish._();

  factory FileTransferFinish.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferFinish()..mergeFromBuffer(data, registry);
  factory FileTransferFinish.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferFinish()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferFinish',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: FileTransferFinish.$_createMessage)
    ..aInt64(1, _omitFieldNames ? '' : 'size')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'sha256', $pb.PbFieldType.OY)
    ..aInt64(3, _omitFieldNames ? '' : 'elapsedMillis')
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
  @$core.Deprecated('Use FileTransferFinish() / FileTransferFinish.new instead')
  static FileTransferFinish create() => FileTransferFinish._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferFinish._();
  @$core.override
  FileTransferFinish createEmptyInstance() => FileTransferFinish._();
  @$core.pragma('dart2js:noInline')
  static FileTransferFinish getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferFinish>(
          FileTransferFinish.$_createMessage);
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

  /// elapsed_millis 是可选的发送方耗时；0 = 未报告（旧行为字节不变）。
  @$pb.TagNumber(3)
  $fixnum.Int64 get elapsedMillis => $_getI64(2);
  @$pb.TagNumber(3)
  set elapsedMillis($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasElapsedMillis() => $_has(2);
  @$pb.TagNumber(3)
  void clearElapsedMillis() => $_clearField(3);
}

class FileTransferResult extends $pb.GeneratedMessage {
  factory FileTransferResult({
    $core.String? path,
    $fixnum.Int64? size,
    $core.List<$core.int>? sha256,
  }) {
    final result = FileTransferResult._();
    if (path != null) result.path = path;
    if (size != null) result.size = size;
    if (sha256 != null) result.sha256 = sha256;
    return result;
  }

  FileTransferResult._();

  factory FileTransferResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferResult()..mergeFromBuffer(data, registry);
  factory FileTransferResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'anytty.protocol.wirepb'),
      createEmptyInstance: FileTransferResult.$_createMessage)
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
  @$core.Deprecated('Use FileTransferResult() / FileTransferResult.new instead')
  static FileTransferResult create() => FileTransferResult._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferResult._();
  @$core.override
  FileTransferResult createEmptyInstance() => FileTransferResult._();
  @$core.pragma('dart2js:noInline')
  static FileTransferResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferResult>(
          FileTransferResult.$_createMessage);
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
