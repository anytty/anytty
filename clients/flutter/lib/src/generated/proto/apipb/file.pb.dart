// This is a generated file - do not edit.
//
// Generated from apipb/file.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'common.pb.dart' as $0;
import 'file.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'file.pbenum.dart';

class FileEntry extends $pb.GeneratedMessage {
  factory FileEntry({
    $core.String? path,
    $core.String? name,
    FileEntryType? type,
    $fixnum.Int64? size,
    $core.int? mode,
    $fixnum.Int64? modifiedAtUnixNano,
    $core.String? linkTarget,
  }) {
    final result = FileEntry._();
    if (path != null) result.path = path;
    if (name != null) result.name = name;
    if (type != null) result.type = type;
    if (size != null) result.size = size;
    if (mode != null) result.mode = mode;
    if (modifiedAtUnixNano != null)
      result.modifiedAtUnixNano = modifiedAtUnixNano;
    if (linkTarget != null) result.linkTarget = linkTarget;
    return result;
  }

  FileEntry._();

  factory FileEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileEntry()..mergeFromBuffer(data, registry);
  factory FileEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileEntry()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileEntry',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileEntry.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'path')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aE<FileEntryType>(3, _omitFieldNames ? '' : 'type',
        enumValues: FileEntryType.values)
    ..aInt64(4, _omitFieldNames ? '' : 'size')
    ..aI(5, _omitFieldNames ? '' : 'mode', fieldType: $pb.PbFieldType.OU3)
    ..aInt64(6, _omitFieldNames ? '' : 'modifiedAtUnixNano')
    ..aOS(7, _omitFieldNames ? '' : 'linkTarget')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileEntry copyWith(void Function(FileEntry) updates) =>
      super.copyWith((message) => updates(message as FileEntry)) as FileEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileEntry() / FileEntry.new instead')
  static FileEntry create() => FileEntry._();
  static $pb.GeneratedMessage $_createMessage() => FileEntry._();
  @$core.override
  FileEntry createEmptyInstance() => FileEntry._();
  @$core.pragma('dart2js:noInline')
  static FileEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileEntry>(FileEntry.$_createMessage);
  static FileEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(1)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(1)
  void clearPath() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  FileEntryType get type => $_getN(2);
  @$pb.TagNumber(3)
  set type(FileEntryType value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasType() => $_has(2);
  @$pb.TagNumber(3)
  void clearType() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get size => $_getI64(3);
  @$pb.TagNumber(4)
  set size($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSize() => $_has(3);
  @$pb.TagNumber(4)
  void clearSize() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get mode => $_getIZ(4);
  @$pb.TagNumber(5)
  set mode($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMode() => $_has(4);
  @$pb.TagNumber(5)
  void clearMode() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get modifiedAtUnixNano => $_getI64(5);
  @$pb.TagNumber(6)
  set modifiedAtUnixNano($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasModifiedAtUnixNano() => $_has(5);
  @$pb.TagNumber(6)
  void clearModifiedAtUnixNano() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get linkTarget => $_getSZ(6);
  @$pb.TagNumber(7)
  set linkTarget($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasLinkTarget() => $_has(6);
  @$pb.TagNumber(7)
  void clearLinkTarget() => $_clearField(7);
}

/// FileUploadResumeHandle 是 owning daemon 签发的短期 opaque 上传凭据。
/// 它由已验证 principal 约束，可跨 protocol session 用于续传或销毁未完成上传，但不能打开 stream 或 release session-bound resource。
class FileUploadResumeHandle extends $pb.GeneratedMessage {
  factory FileUploadResumeHandle({
    $core.List<$core.int>? opaqueToken,
  }) {
    final result = FileUploadResumeHandle._();
    if (opaqueToken != null) result.opaqueToken = opaqueToken;
    return result;
  }

  FileUploadResumeHandle._();

  factory FileUploadResumeHandle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileUploadResumeHandle()..mergeFromBuffer(data, registry);
  factory FileUploadResumeHandle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileUploadResumeHandle()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileUploadResumeHandle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileUploadResumeHandle.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'opaqueToken', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileUploadResumeHandle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileUploadResumeHandle copyWith(
          void Function(FileUploadResumeHandle) updates) =>
      super.copyWith((message) => updates(message as FileUploadResumeHandle))
          as FileUploadResumeHandle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileUploadResumeHandle() / FileUploadResumeHandle.new instead')
  static FileUploadResumeHandle create() => FileUploadResumeHandle._();
  static $pb.GeneratedMessage $_createMessage() => FileUploadResumeHandle._();
  @$core.override
  FileUploadResumeHandle createEmptyInstance() => FileUploadResumeHandle._();
  @$core.pragma('dart2js:noInline')
  static FileUploadResumeHandle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileUploadResumeHandle>(
          FileUploadResumeHandle.$_createMessage);
  static FileUploadResumeHandle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get opaqueToken => $_getN(0);
  @$pb.TagNumber(1)
  set opaqueToken($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOpaqueToken() => $_has(0);
  @$pb.TagNumber(1)
  void clearOpaqueToken() => $_clearField(1);
}

class FileListCommand extends $pb.GeneratedMessage {
  factory FileListCommand({
    $core.String? path,
    $core.String? cursor,
    $core.int? limit,
  }) {
    final result = FileListCommand._();
    if (path != null) result.path = path;
    if (cursor != null) result.cursor = cursor;
    if (limit != null) result.limit = limit;
    return result;
  }

  FileListCommand._();

  factory FileListCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileListCommand()..mergeFromBuffer(data, registry);
  factory FileListCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileListCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileListCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileListCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aOS(3, _omitFieldNames ? '' : 'cursor')
    ..aI(4, _omitFieldNames ? '' : 'limit')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileListCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileListCommand copyWith(void Function(FileListCommand) updates) =>
      super.copyWith((message) => updates(message as FileListCommand))
          as FileListCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileListCommand() / FileListCommand.new instead')
  static FileListCommand create() => FileListCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileListCommand._();
  @$core.override
  FileListCommand createEmptyInstance() => FileListCommand._();
  @$core.pragma('dart2js:noInline')
  static FileListCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileListCommand>(
          FileListCommand.$_createMessage);
  static FileListCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get cursor => $_getSZ(1);
  @$pb.TagNumber(3)
  set cursor($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasCursor() => $_has(1);
  @$pb.TagNumber(3)
  void clearCursor() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get limit => $_getIZ(2);
  @$pb.TagNumber(4)
  set limit($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(4)
  $core.bool hasLimit() => $_has(2);
  @$pb.TagNumber(4)
  void clearLimit() => $_clearField(4);
}

class FileStatCommand extends $pb.GeneratedMessage {
  factory FileStatCommand({
    $core.String? path,
  }) {
    final result = FileStatCommand._();
    if (path != null) result.path = path;
    return result;
  }

  FileStatCommand._();

  factory FileStatCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileStatCommand()..mergeFromBuffer(data, registry);
  factory FileStatCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileStatCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileStatCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileStatCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileStatCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileStatCommand copyWith(void Function(FileStatCommand) updates) =>
      super.copyWith((message) => updates(message as FileStatCommand))
          as FileStatCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileStatCommand() / FileStatCommand.new instead')
  static FileStatCommand create() => FileStatCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileStatCommand._();
  @$core.override
  FileStatCommand createEmptyInstance() => FileStatCommand._();
  @$core.pragma('dart2js:noInline')
  static FileStatCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileStatCommand>(
          FileStatCommand.$_createMessage);
  static FileStatCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);
}

class FilePreviewCommand extends $pb.GeneratedMessage {
  factory FilePreviewCommand({
    $core.String? path,
    $fixnum.Int64? maxBytes,
  }) {
    final result = FilePreviewCommand._();
    if (path != null) result.path = path;
    if (maxBytes != null) result.maxBytes = maxBytes;
    return result;
  }

  FilePreviewCommand._();

  factory FilePreviewCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FilePreviewCommand()..mergeFromBuffer(data, registry);
  factory FilePreviewCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FilePreviewCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FilePreviewCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FilePreviewCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aInt64(3, _omitFieldNames ? '' : 'maxBytes')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FilePreviewCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FilePreviewCommand copyWith(void Function(FilePreviewCommand) updates) =>
      super.copyWith((message) => updates(message as FilePreviewCommand))
          as FilePreviewCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FilePreviewCommand() / FilePreviewCommand.new instead')
  static FilePreviewCommand create() => FilePreviewCommand._();
  static $pb.GeneratedMessage $_createMessage() => FilePreviewCommand._();
  @$core.override
  FilePreviewCommand createEmptyInstance() => FilePreviewCommand._();
  @$core.pragma('dart2js:noInline')
  static FilePreviewCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FilePreviewCommand>(
          FilePreviewCommand.$_createMessage);
  static FilePreviewCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get maxBytes => $_getI64(1);
  @$pb.TagNumber(3)
  set maxBytes($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(3)
  $core.bool hasMaxBytes() => $_has(1);
  @$pb.TagNumber(3)
  void clearMaxBytes() => $_clearField(3);
}

class FileMkdirCommand extends $pb.GeneratedMessage {
  factory FileMkdirCommand({
    $core.String? path,
    $core.bool? recursive,
  }) {
    final result = FileMkdirCommand._();
    if (path != null) result.path = path;
    if (recursive != null) result.recursive = recursive;
    return result;
  }

  FileMkdirCommand._();

  factory FileMkdirCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileMkdirCommand()..mergeFromBuffer(data, registry);
  factory FileMkdirCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileMkdirCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileMkdirCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileMkdirCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aOB(3, _omitFieldNames ? '' : 'recursive')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileMkdirCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileMkdirCommand copyWith(void Function(FileMkdirCommand) updates) =>
      super.copyWith((message) => updates(message as FileMkdirCommand))
          as FileMkdirCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileMkdirCommand() / FileMkdirCommand.new instead')
  static FileMkdirCommand create() => FileMkdirCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileMkdirCommand._();
  @$core.override
  FileMkdirCommand createEmptyInstance() => FileMkdirCommand._();
  @$core.pragma('dart2js:noInline')
  static FileMkdirCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileMkdirCommand>(
          FileMkdirCommand.$_createMessage);
  static FileMkdirCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get recursive => $_getBF(1);
  @$pb.TagNumber(3)
  set recursive($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(3)
  $core.bool hasRecursive() => $_has(1);
  @$pb.TagNumber(3)
  void clearRecursive() => $_clearField(3);
}

class FileRenameCommand extends $pb.GeneratedMessage {
  factory FileRenameCommand({
    $core.String? path,
    $core.String? newPath,
    $core.bool? overwrite,
  }) {
    final result = FileRenameCommand._();
    if (path != null) result.path = path;
    if (newPath != null) result.newPath = newPath;
    if (overwrite != null) result.overwrite = overwrite;
    return result;
  }

  FileRenameCommand._();

  factory FileRenameCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileRenameCommand()..mergeFromBuffer(data, registry);
  factory FileRenameCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileRenameCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileRenameCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileRenameCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aOS(3, _omitFieldNames ? '' : 'newPath')
    ..aOB(4, _omitFieldNames ? '' : 'overwrite')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileRenameCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileRenameCommand copyWith(void Function(FileRenameCommand) updates) =>
      super.copyWith((message) => updates(message as FileRenameCommand))
          as FileRenameCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileRenameCommand() / FileRenameCommand.new instead')
  static FileRenameCommand create() => FileRenameCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileRenameCommand._();
  @$core.override
  FileRenameCommand createEmptyInstance() => FileRenameCommand._();
  @$core.pragma('dart2js:noInline')
  static FileRenameCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileRenameCommand>(
          FileRenameCommand.$_createMessage);
  static FileRenameCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get newPath => $_getSZ(1);
  @$pb.TagNumber(3)
  set newPath($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasNewPath() => $_has(1);
  @$pb.TagNumber(3)
  void clearNewPath() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get overwrite => $_getBF(2);
  @$pb.TagNumber(4)
  set overwrite($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(4)
  $core.bool hasOverwrite() => $_has(2);
  @$pb.TagNumber(4)
  void clearOverwrite() => $_clearField(4);
}

class FileDeleteCommand extends $pb.GeneratedMessage {
  factory FileDeleteCommand({
    $core.String? path,
    $core.bool? recursive,
  }) {
    final result = FileDeleteCommand._();
    if (path != null) result.path = path;
    if (recursive != null) result.recursive = recursive;
    return result;
  }

  FileDeleteCommand._();

  factory FileDeleteCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileDeleteCommand()..mergeFromBuffer(data, registry);
  factory FileDeleteCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileDeleteCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileDeleteCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileDeleteCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aOB(3, _omitFieldNames ? '' : 'recursive')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileDeleteCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileDeleteCommand copyWith(void Function(FileDeleteCommand) updates) =>
      super.copyWith((message) => updates(message as FileDeleteCommand))
          as FileDeleteCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileDeleteCommand() / FileDeleteCommand.new instead')
  static FileDeleteCommand create() => FileDeleteCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileDeleteCommand._();
  @$core.override
  FileDeleteCommand createEmptyInstance() => FileDeleteCommand._();
  @$core.pragma('dart2js:noInline')
  static FileDeleteCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileDeleteCommand>(
          FileDeleteCommand.$_createMessage);
  static FileDeleteCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get recursive => $_getBF(1);
  @$pb.TagNumber(3)
  set recursive($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(3)
  $core.bool hasRecursive() => $_has(1);
  @$pb.TagNumber(3)
  void clearRecursive() => $_clearField(3);
}

class FileCopyCommand extends $pb.GeneratedMessage {
  factory FileCopyCommand({
    $core.Iterable<$core.String>? paths,
    $core.String? targetDirectory,
    $core.bool? overwrite,
  }) {
    final result = FileCopyCommand._();
    if (paths != null) result.paths.addAll(paths);
    if (targetDirectory != null) result.targetDirectory = targetDirectory;
    if (overwrite != null) result.overwrite = overwrite;
    return result;
  }

  FileCopyCommand._();

  factory FileCopyCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileCopyCommand()..mergeFromBuffer(data, registry);
  factory FileCopyCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileCopyCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileCopyCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileCopyCommand.$_createMessage)
    ..pPS(2, _omitFieldNames ? '' : 'paths')
    ..aOS(3, _omitFieldNames ? '' : 'targetDirectory')
    ..aOB(4, _omitFieldNames ? '' : 'overwrite')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileCopyCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileCopyCommand copyWith(void Function(FileCopyCommand) updates) =>
      super.copyWith((message) => updates(message as FileCopyCommand))
          as FileCopyCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileCopyCommand() / FileCopyCommand.new instead')
  static FileCopyCommand create() => FileCopyCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileCopyCommand._();
  @$core.override
  FileCopyCommand createEmptyInstance() => FileCopyCommand._();
  @$core.pragma('dart2js:noInline')
  static FileCopyCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileCopyCommand>(
          FileCopyCommand.$_createMessage);
  static FileCopyCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get paths => $_getList(0);

  @$pb.TagNumber(3)
  $core.String get targetDirectory => $_getSZ(1);
  @$pb.TagNumber(3)
  set targetDirectory($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasTargetDirectory() => $_has(1);
  @$pb.TagNumber(3)
  void clearTargetDirectory() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get overwrite => $_getBF(2);
  @$pb.TagNumber(4)
  set overwrite($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(4)
  $core.bool hasOverwrite() => $_has(2);
  @$pb.TagNumber(4)
  void clearOverwrite() => $_clearField(4);
}

class FileMoveCommand extends $pb.GeneratedMessage {
  factory FileMoveCommand({
    $core.Iterable<$core.String>? paths,
    $core.String? targetDirectory,
    $core.bool? overwrite,
  }) {
    final result = FileMoveCommand._();
    if (paths != null) result.paths.addAll(paths);
    if (targetDirectory != null) result.targetDirectory = targetDirectory;
    if (overwrite != null) result.overwrite = overwrite;
    return result;
  }

  FileMoveCommand._();

  factory FileMoveCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileMoveCommand()..mergeFromBuffer(data, registry);
  factory FileMoveCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileMoveCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileMoveCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileMoveCommand.$_createMessage)
    ..pPS(2, _omitFieldNames ? '' : 'paths')
    ..aOS(3, _omitFieldNames ? '' : 'targetDirectory')
    ..aOB(4, _omitFieldNames ? '' : 'overwrite')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileMoveCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileMoveCommand copyWith(void Function(FileMoveCommand) updates) =>
      super.copyWith((message) => updates(message as FileMoveCommand))
          as FileMoveCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileMoveCommand() / FileMoveCommand.new instead')
  static FileMoveCommand create() => FileMoveCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileMoveCommand._();
  @$core.override
  FileMoveCommand createEmptyInstance() => FileMoveCommand._();
  @$core.pragma('dart2js:noInline')
  static FileMoveCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileMoveCommand>(
          FileMoveCommand.$_createMessage);
  static FileMoveCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get paths => $_getList(0);

  @$pb.TagNumber(3)
  $core.String get targetDirectory => $_getSZ(1);
  @$pb.TagNumber(3)
  set targetDirectory($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasTargetDirectory() => $_has(1);
  @$pb.TagNumber(3)
  void clearTargetDirectory() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get overwrite => $_getBF(2);
  @$pb.TagNumber(4)
  set overwrite($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(4)
  $core.bool hasOverwrite() => $_has(2);
  @$pb.TagNumber(4)
  void clearOverwrite() => $_clearField(4);
}

class FileDownloadOpenCommand extends $pb.GeneratedMessage {
  factory FileDownloadOpenCommand({
    $core.String? path,
    $fixnum.Int64? offset,
    $fixnum.Int64? expectedSize,
    $fixnum.Int64? expectedModifiedAtUnixNano,
    $0.OperationStamp? operation,
    $core.Iterable<$core.String>? acceptCompression,
    $fixnum.Int64? progressIntervalBytes,
  }) {
    final result = FileDownloadOpenCommand._();
    if (path != null) result.path = path;
    if (offset != null) result.offset = offset;
    if (expectedSize != null) result.expectedSize = expectedSize;
    if (expectedModifiedAtUnixNano != null)
      result.expectedModifiedAtUnixNano = expectedModifiedAtUnixNano;
    if (operation != null) result.operation = operation;
    if (acceptCompression != null)
      result.acceptCompression.addAll(acceptCompression);
    if (progressIntervalBytes != null)
      result.progressIntervalBytes = progressIntervalBytes;
    return result;
  }

  FileDownloadOpenCommand._();

  factory FileDownloadOpenCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileDownloadOpenCommand()..mergeFromBuffer(data, registry);
  factory FileDownloadOpenCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileDownloadOpenCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileDownloadOpenCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileDownloadOpenCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aInt64(3, _omitFieldNames ? '' : 'offset')
    ..aInt64(4, _omitFieldNames ? '' : 'expectedSize')
    ..aInt64(5, _omitFieldNames ? '' : 'expectedModifiedAtUnixNano')
    ..aOM<$0.OperationStamp>(6, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.$_createMessage)
    ..pPS(7, _omitFieldNames ? '' : 'acceptCompression')
    ..aInt64(8, _omitFieldNames ? '' : 'progressIntervalBytes')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileDownloadOpenCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileDownloadOpenCommand copyWith(
          void Function(FileDownloadOpenCommand) updates) =>
      super.copyWith((message) => updates(message as FileDownloadOpenCommand))
          as FileDownloadOpenCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileDownloadOpenCommand() / FileDownloadOpenCommand.new instead')
  static FileDownloadOpenCommand create() => FileDownloadOpenCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileDownloadOpenCommand._();
  @$core.override
  FileDownloadOpenCommand createEmptyInstance() => FileDownloadOpenCommand._();
  @$core.pragma('dart2js:noInline')
  static FileDownloadOpenCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileDownloadOpenCommand>(
          FileDownloadOpenCommand.$_createMessage);
  static FileDownloadOpenCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get offset => $_getI64(1);
  @$pb.TagNumber(3)
  set offset($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(3)
  $core.bool hasOffset() => $_has(1);
  @$pb.TagNumber(3)
  void clearOffset() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get expectedSize => $_getI64(2);
  @$pb.TagNumber(4)
  set expectedSize($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(4)
  $core.bool hasExpectedSize() => $_has(2);
  @$pb.TagNumber(4)
  void clearExpectedSize() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get expectedModifiedAtUnixNano => $_getI64(3);
  @$pb.TagNumber(5)
  set expectedModifiedAtUnixNano($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(5)
  $core.bool hasExpectedModifiedAtUnixNano() => $_has(3);
  @$pb.TagNumber(5)
  void clearExpectedModifiedAtUnixNano() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.OperationStamp get operation => $_getN(4);
  @$pb.TagNumber(6)
  set operation($0.OperationStamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasOperation() => $_has(4);
  @$pb.TagNumber(6)
  void clearOperation() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.OperationStamp ensureOperation() => $_ensure(4);

  /// accept_compression 是客户端可解压的 data frame 编码列表（当前支持 "zstd"）。
  /// 空 = 只接受 identity；旧客户端不发该字段，服务端行为完全不变。
  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get acceptCompression => $_getList(5);

  /// progress_interval_bytes 请求服务端按字节窗口合并结构化进度；0 = 关闭（默认）。
  @$pb.TagNumber(8)
  $fixnum.Int64 get progressIntervalBytes => $_getI64(6);
  @$pb.TagNumber(8)
  set progressIntervalBytes($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(8)
  $core.bool hasProgressIntervalBytes() => $_has(6);
  @$pb.TagNumber(8)
  void clearProgressIntervalBytes() => $_clearField(8);
}

class FileUploadOpenCommand extends $pb.GeneratedMessage {
  factory FileUploadOpenCommand({
    $core.String? path,
    $fixnum.Int64? size,
    $core.bool? overwrite,
    FileUploadResumeHandle? resume,
    $0.OperationStamp? operation,
    $core.Iterable<$core.String>? acceptCompression,
    $fixnum.Int64? progressIntervalBytes,
  }) {
    final result = FileUploadOpenCommand._();
    if (path != null) result.path = path;
    if (size != null) result.size = size;
    if (overwrite != null) result.overwrite = overwrite;
    if (resume != null) result.resume = resume;
    if (operation != null) result.operation = operation;
    if (acceptCompression != null)
      result.acceptCompression.addAll(acceptCompression);
    if (progressIntervalBytes != null)
      result.progressIntervalBytes = progressIntervalBytes;
    return result;
  }

  FileUploadOpenCommand._();

  factory FileUploadOpenCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileUploadOpenCommand()..mergeFromBuffer(data, registry);
  factory FileUploadOpenCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileUploadOpenCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileUploadOpenCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileUploadOpenCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aInt64(3, _omitFieldNames ? '' : 'size')
    ..aOB(4, _omitFieldNames ? '' : 'overwrite')
    ..aOM<FileUploadResumeHandle>(5, _omitFieldNames ? '' : 'resume',
        subBuilder: FileUploadResumeHandle.$_createMessage)
    ..aOM<$0.OperationStamp>(6, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.$_createMessage)
    ..pPS(7, _omitFieldNames ? '' : 'acceptCompression')
    ..aInt64(8, _omitFieldNames ? '' : 'progressIntervalBytes')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileUploadOpenCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileUploadOpenCommand copyWith(
          void Function(FileUploadOpenCommand) updates) =>
      super.copyWith((message) => updates(message as FileUploadOpenCommand))
          as FileUploadOpenCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileUploadOpenCommand() / FileUploadOpenCommand.new instead')
  static FileUploadOpenCommand create() => FileUploadOpenCommand._();
  static $pb.GeneratedMessage $_createMessage() => FileUploadOpenCommand._();
  @$core.override
  FileUploadOpenCommand createEmptyInstance() => FileUploadOpenCommand._();
  @$core.pragma('dart2js:noInline')
  static FileUploadOpenCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileUploadOpenCommand>(
          FileUploadOpenCommand.$_createMessage);
  static FileUploadOpenCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get size => $_getI64(1);
  @$pb.TagNumber(3)
  set size($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(3)
  $core.bool hasSize() => $_has(1);
  @$pb.TagNumber(3)
  void clearSize() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get overwrite => $_getBF(2);
  @$pb.TagNumber(4)
  set overwrite($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(4)
  $core.bool hasOverwrite() => $_has(2);
  @$pb.TagNumber(4)
  void clearOverwrite() => $_clearField(4);

  @$pb.TagNumber(5)
  FileUploadResumeHandle get resume => $_getN(3);
  @$pb.TagNumber(5)
  set resume(FileUploadResumeHandle value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasResume() => $_has(3);
  @$pb.TagNumber(5)
  void clearResume() => $_clearField(5);
  @$pb.TagNumber(5)
  FileUploadResumeHandle ensureResume() => $_ensure(3);

  @$pb.TagNumber(6)
  $0.OperationStamp get operation => $_getN(4);
  @$pb.TagNumber(6)
  set operation($0.OperationStamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasOperation() => $_has(4);
  @$pb.TagNumber(6)
  void clearOperation() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.OperationStamp ensureOperation() => $_ensure(4);

  /// accept_compression 是客户端可编码的 data frame 编码列表（当前支持 "zstd"）。
  /// 服务端在 handle.content_encoding 中选择一个；空 = identity，旧行为不变。
  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get acceptCompression => $_getList(5);

  /// progress_interval_bytes 请求服务端在 ack 上合并结构化进度；0 = 关闭（默认）。
  @$pb.TagNumber(8)
  $fixnum.Int64 get progressIntervalBytes => $_getI64(6);
  @$pb.TagNumber(8)
  set progressIntervalBytes($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(8)
  $core.bool hasProgressIntervalBytes() => $_has(6);
  @$pb.TagNumber(8)
  void clearProgressIntervalBytes() => $_clearField(8);
}

class FileTransferCancelCommand extends $pb.GeneratedMessage {
  factory FileTransferCancelCommand({
    $0.ResourceHandle? transfer,
    $0.OperationStamp? operation,
    FileUploadResumeHandle? uploadResume,
  }) {
    final result = FileTransferCancelCommand._();
    if (transfer != null) result.transfer = transfer;
    if (operation != null) result.operation = operation;
    if (uploadResume != null) result.uploadResume = uploadResume;
    return result;
  }

  FileTransferCancelCommand._();

  factory FileTransferCancelCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferCancelCommand()..mergeFromBuffer(data, registry);
  factory FileTransferCancelCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferCancelCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferCancelCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileTransferCancelCommand.$_createMessage)
    ..aOM<$0.ResourceHandle>(2, _omitFieldNames ? '' : 'transfer',
        subBuilder: $0.ResourceHandle.$_createMessage)
    ..aOM<$0.OperationStamp>(3, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.$_createMessage)
    ..aOM<FileUploadResumeHandle>(4, _omitFieldNames ? '' : 'uploadResume',
        subBuilder: FileUploadResumeHandle.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferCancelCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferCancelCommand copyWith(
          void Function(FileTransferCancelCommand) updates) =>
      super.copyWith((message) => updates(message as FileTransferCancelCommand))
          as FileTransferCancelCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileTransferCancelCommand() / FileTransferCancelCommand.new instead')
  static FileTransferCancelCommand create() => FileTransferCancelCommand._();
  static $pb.GeneratedMessage $_createMessage() =>
      FileTransferCancelCommand._();
  @$core.override
  FileTransferCancelCommand createEmptyInstance() =>
      FileTransferCancelCommand._();
  @$core.pragma('dart2js:noInline')
  static FileTransferCancelCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferCancelCommand>(
          FileTransferCancelCommand.$_createMessage);
  static FileTransferCancelCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ResourceHandle get transfer => $_getN(0);
  @$pb.TagNumber(2)
  set transfer($0.ResourceHandle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTransfer() => $_has(0);
  @$pb.TagNumber(2)
  void clearTransfer() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ResourceHandle ensureTransfer() => $_ensure(0);

  @$pb.TagNumber(3)
  $0.OperationStamp get operation => $_getN(1);
  @$pb.TagNumber(3)
  set operation($0.OperationStamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasOperation() => $_has(1);
  @$pb.TagNumber(3)
  void clearOperation() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.OperationStamp ensureOperation() => $_ensure(1);

  @$pb.TagNumber(4)
  FileUploadResumeHandle get uploadResume => $_getN(2);
  @$pb.TagNumber(4)
  set uploadResume(FileUploadResumeHandle value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasUploadResume() => $_has(2);
  @$pb.TagNumber(4)
  void clearUploadResume() => $_clearField(4);
  @$pb.TagNumber(4)
  FileUploadResumeHandle ensureUploadResume() => $_ensure(2);
}

class FileListResult extends $pb.GeneratedMessage {
  factory FileListResult({
    $core.String? path,
    $core.Iterable<FileEntry>? entries,
    $core.String? nextCursor,
  }) {
    final result = FileListResult._();
    if (path != null) result.path = path;
    if (entries != null) result.entries.addAll(entries);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  FileListResult._();

  factory FileListResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileListResult()..mergeFromBuffer(data, registry);
  factory FileListResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileListResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileListResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileListResult.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'path')
    ..pPM<FileEntry>(2, _omitFieldNames ? '' : 'entries',
        subBuilder: FileEntry.$_createMessage)
    ..aOS(3, _omitFieldNames ? '' : 'nextCursor')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileListResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileListResult copyWith(void Function(FileListResult) updates) =>
      super.copyWith((message) => updates(message as FileListResult))
          as FileListResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileListResult() / FileListResult.new instead')
  static FileListResult create() => FileListResult._();
  static $pb.GeneratedMessage $_createMessage() => FileListResult._();
  @$core.override
  FileListResult createEmptyInstance() => FileListResult._();
  @$core.pragma('dart2js:noInline')
  static FileListResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileListResult>(
          FileListResult.$_createMessage);
  static FileListResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(1)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(1)
  void clearPath() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<FileEntry> get entries => $_getList(1);

  @$pb.TagNumber(3)
  $core.String get nextCursor => $_getSZ(2);
  @$pb.TagNumber(3)
  set nextCursor($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNextCursor() => $_has(2);
  @$pb.TagNumber(3)
  void clearNextCursor() => $_clearField(3);
}

class FileStatResult extends $pb.GeneratedMessage {
  factory FileStatResult({
    FileEntry? entry,
  }) {
    final result = FileStatResult._();
    if (entry != null) result.entry = entry;
    return result;
  }

  FileStatResult._();

  factory FileStatResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileStatResult()..mergeFromBuffer(data, registry);
  factory FileStatResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileStatResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileStatResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileStatResult.$_createMessage)
    ..aOM<FileEntry>(1, _omitFieldNames ? '' : 'entry',
        subBuilder: FileEntry.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileStatResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileStatResult copyWith(void Function(FileStatResult) updates) =>
      super.copyWith((message) => updates(message as FileStatResult))
          as FileStatResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileStatResult() / FileStatResult.new instead')
  static FileStatResult create() => FileStatResult._();
  static $pb.GeneratedMessage $_createMessage() => FileStatResult._();
  @$core.override
  FileStatResult createEmptyInstance() => FileStatResult._();
  @$core.pragma('dart2js:noInline')
  static FileStatResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileStatResult>(
          FileStatResult.$_createMessage);
  static FileStatResult? _defaultInstance;

  @$pb.TagNumber(1)
  FileEntry get entry => $_getN(0);
  @$pb.TagNumber(1)
  set entry(FileEntry value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEntry() => $_has(0);
  @$pb.TagNumber(1)
  void clearEntry() => $_clearField(1);
  @$pb.TagNumber(1)
  FileEntry ensureEntry() => $_ensure(0);
}

class FilePreviewResult extends $pb.GeneratedMessage {
  factory FilePreviewResult({
    FileEntry? entry,
    $core.String? mimeType,
    $core.List<$core.int>? content,
    $core.bool? truncated,
    $core.List<$core.int>? sha256,
  }) {
    final result = FilePreviewResult._();
    if (entry != null) result.entry = entry;
    if (mimeType != null) result.mimeType = mimeType;
    if (content != null) result.content = content;
    if (truncated != null) result.truncated = truncated;
    if (sha256 != null) result.sha256 = sha256;
    return result;
  }

  FilePreviewResult._();

  factory FilePreviewResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FilePreviewResult()..mergeFromBuffer(data, registry);
  factory FilePreviewResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FilePreviewResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FilePreviewResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FilePreviewResult.$_createMessage)
    ..aOM<FileEntry>(1, _omitFieldNames ? '' : 'entry',
        subBuilder: FileEntry.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'mimeType')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'content', $pb.PbFieldType.OY)
    ..aOB(4, _omitFieldNames ? '' : 'truncated')
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'sha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FilePreviewResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FilePreviewResult copyWith(void Function(FilePreviewResult) updates) =>
      super.copyWith((message) => updates(message as FilePreviewResult))
          as FilePreviewResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FilePreviewResult() / FilePreviewResult.new instead')
  static FilePreviewResult create() => FilePreviewResult._();
  static $pb.GeneratedMessage $_createMessage() => FilePreviewResult._();
  @$core.override
  FilePreviewResult createEmptyInstance() => FilePreviewResult._();
  @$core.pragma('dart2js:noInline')
  static FilePreviewResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FilePreviewResult>(
          FilePreviewResult.$_createMessage);
  static FilePreviewResult? _defaultInstance;

  @$pb.TagNumber(1)
  FileEntry get entry => $_getN(0);
  @$pb.TagNumber(1)
  set entry(FileEntry value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEntry() => $_has(0);
  @$pb.TagNumber(1)
  void clearEntry() => $_clearField(1);
  @$pb.TagNumber(1)
  FileEntry ensureEntry() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get mimeType => $_getSZ(1);
  @$pb.TagNumber(2)
  set mimeType($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMimeType() => $_has(1);
  @$pb.TagNumber(2)
  void clearMimeType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get content => $_getN(2);
  @$pb.TagNumber(3)
  set content($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasContent() => $_has(2);
  @$pb.TagNumber(3)
  void clearContent() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get truncated => $_getBF(3);
  @$pb.TagNumber(4)
  set truncated($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTruncated() => $_has(3);
  @$pb.TagNumber(4)
  void clearTruncated() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get sha256 => $_getN(4);
  @$pb.TagNumber(5)
  set sha256($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSha256() => $_has(4);
  @$pb.TagNumber(5)
  void clearSha256() => $_clearField(5);
}

class FileOperationResult extends $pb.GeneratedMessage {
  factory FileOperationResult({
    $core.String? path,
    $core.String? targetPath,
    $core.bool? success,
    $core.String? errorCode,
    $core.String? errorMessage,
  }) {
    final result = FileOperationResult._();
    if (path != null) result.path = path;
    if (targetPath != null) result.targetPath = targetPath;
    if (success != null) result.success = success;
    if (errorCode != null) result.errorCode = errorCode;
    if (errorMessage != null) result.errorMessage = errorMessage;
    return result;
  }

  FileOperationResult._();

  factory FileOperationResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileOperationResult()..mergeFromBuffer(data, registry);
  factory FileOperationResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileOperationResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileOperationResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileOperationResult.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'path')
    ..aOS(2, _omitFieldNames ? '' : 'targetPath')
    ..aOB(3, _omitFieldNames ? '' : 'success')
    ..aOS(4, _omitFieldNames ? '' : 'errorCode')
    ..aOS(5, _omitFieldNames ? '' : 'errorMessage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileOperationResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileOperationResult copyWith(void Function(FileOperationResult) updates) =>
      super.copyWith((message) => updates(message as FileOperationResult))
          as FileOperationResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use FileOperationResult() / FileOperationResult.new instead')
  static FileOperationResult create() => FileOperationResult._();
  static $pb.GeneratedMessage $_createMessage() => FileOperationResult._();
  @$core.override
  FileOperationResult createEmptyInstance() => FileOperationResult._();
  @$core.pragma('dart2js:noInline')
  static FileOperationResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileOperationResult>(
          FileOperationResult.$_createMessage);
  static FileOperationResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get path => $_getSZ(0);
  @$pb.TagNumber(1)
  set path($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPath() => $_has(0);
  @$pb.TagNumber(1)
  void clearPath() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get targetPath => $_getSZ(1);
  @$pb.TagNumber(2)
  set targetPath($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTargetPath() => $_has(1);
  @$pb.TagNumber(2)
  void clearTargetPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get success => $_getBF(2);
  @$pb.TagNumber(3)
  set success($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSuccess() => $_has(2);
  @$pb.TagNumber(3)
  void clearSuccess() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get errorCode => $_getSZ(3);
  @$pb.TagNumber(4)
  set errorCode($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasErrorCode() => $_has(3);
  @$pb.TagNumber(4)
  void clearErrorCode() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get errorMessage => $_getSZ(4);
  @$pb.TagNumber(5)
  set errorMessage($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasErrorMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearErrorMessage() => $_clearField(5);
}

class FileBatchResult extends $pb.GeneratedMessage {
  factory FileBatchResult({
    $core.Iterable<FileOperationResult>? results,
  }) {
    final result = FileBatchResult._();
    if (results != null) result.results.addAll(results);
    return result;
  }

  FileBatchResult._();

  factory FileBatchResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileBatchResult()..mergeFromBuffer(data, registry);
  factory FileBatchResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileBatchResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileBatchResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileBatchResult.$_createMessage)
    ..pPM<FileOperationResult>(1, _omitFieldNames ? '' : 'results',
        subBuilder: FileOperationResult.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileBatchResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileBatchResult copyWith(void Function(FileBatchResult) updates) =>
      super.copyWith((message) => updates(message as FileBatchResult))
          as FileBatchResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileBatchResult() / FileBatchResult.new instead')
  static FileBatchResult create() => FileBatchResult._();
  static $pb.GeneratedMessage $_createMessage() => FileBatchResult._();
  @$core.override
  FileBatchResult createEmptyInstance() => FileBatchResult._();
  @$core.pragma('dart2js:noInline')
  static FileBatchResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FileBatchResult>(
          FileBatchResult.$_createMessage);
  static FileBatchResult? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<FileOperationResult> get results => $_getList(0);
}

class FileTransferHandle extends $pb.GeneratedMessage {
  factory FileTransferHandle({
    $0.ResourceHandle? resource,
    $core.String? path,
    $fixnum.Int64? offset,
    $fixnum.Int64? size,
    $fixnum.Int64? modifiedAtUnixNano,
    $0.OperationStamp? operation,
    FileUploadResumeHandle? resume,
    $core.int? chunkBytes,
    $fixnum.Int64? windowBytes,
    $core.String? contentEncoding,
    $fixnum.Int64? progressIntervalBytes,
  }) {
    final result = FileTransferHandle._();
    if (resource != null) result.resource = resource;
    if (path != null) result.path = path;
    if (offset != null) result.offset = offset;
    if (size != null) result.size = size;
    if (modifiedAtUnixNano != null)
      result.modifiedAtUnixNano = modifiedAtUnixNano;
    if (operation != null) result.operation = operation;
    if (resume != null) result.resume = resume;
    if (chunkBytes != null) result.chunkBytes = chunkBytes;
    if (windowBytes != null) result.windowBytes = windowBytes;
    if (contentEncoding != null) result.contentEncoding = contentEncoding;
    if (progressIntervalBytes != null)
      result.progressIntervalBytes = progressIntervalBytes;
    return result;
  }

  FileTransferHandle._();

  factory FileTransferHandle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferHandle()..mergeFromBuffer(data, registry);
  factory FileTransferHandle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferHandle()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferHandle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileTransferHandle.$_createMessage)
    ..aOM<$0.ResourceHandle>(1, _omitFieldNames ? '' : 'resource',
        subBuilder: $0.ResourceHandle.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..aInt64(3, _omitFieldNames ? '' : 'offset')
    ..aInt64(4, _omitFieldNames ? '' : 'size')
    ..aInt64(5, _omitFieldNames ? '' : 'modifiedAtUnixNano')
    ..aOM<$0.OperationStamp>(6, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.$_createMessage)
    ..aOM<FileUploadResumeHandle>(7, _omitFieldNames ? '' : 'resume',
        subBuilder: FileUploadResumeHandle.$_createMessage)
    ..aI(8, _omitFieldNames ? '' : 'chunkBytes', fieldType: $pb.PbFieldType.OU3)
    ..aInt64(9, _omitFieldNames ? '' : 'windowBytes')
    ..aOS(10, _omitFieldNames ? '' : 'contentEncoding')
    ..aInt64(11, _omitFieldNames ? '' : 'progressIntervalBytes')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferHandle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferHandle copyWith(void Function(FileTransferHandle) updates) =>
      super.copyWith((message) => updates(message as FileTransferHandle))
          as FileTransferHandle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FileTransferHandle() / FileTransferHandle.new instead')
  static FileTransferHandle create() => FileTransferHandle._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferHandle._();
  @$core.override
  FileTransferHandle createEmptyInstance() => FileTransferHandle._();
  @$core.pragma('dart2js:noInline')
  static FileTransferHandle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferHandle>(
          FileTransferHandle.$_createMessage);
  static FileTransferHandle? _defaultInstance;

  @$pb.TagNumber(1)
  $0.ResourceHandle get resource => $_getN(0);
  @$pb.TagNumber(1)
  set resource($0.ResourceHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasResource() => $_has(0);
  @$pb.TagNumber(1)
  void clearResource() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.ResourceHandle ensureResource() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(1);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(1);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get offset => $_getI64(2);
  @$pb.TagNumber(3)
  set offset($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOffset() => $_has(2);
  @$pb.TagNumber(3)
  void clearOffset() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get size => $_getI64(3);
  @$pb.TagNumber(4)
  set size($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSize() => $_has(3);
  @$pb.TagNumber(4)
  void clearSize() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get modifiedAtUnixNano => $_getI64(4);
  @$pb.TagNumber(5)
  set modifiedAtUnixNano($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasModifiedAtUnixNano() => $_has(4);
  @$pb.TagNumber(5)
  void clearModifiedAtUnixNano() => $_clearField(5);

  @$pb.TagNumber(6)
  $0.OperationStamp get operation => $_getN(5);
  @$pb.TagNumber(6)
  set operation($0.OperationStamp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasOperation() => $_has(5);
  @$pb.TagNumber(6)
  void clearOperation() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.OperationStamp ensureOperation() => $_ensure(5);

  @$pb.TagNumber(7)
  FileUploadResumeHandle get resume => $_getN(6);
  @$pb.TagNumber(7)
  set resume(FileUploadResumeHandle value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasResume() => $_has(6);
  @$pb.TagNumber(7)
  void clearResume() => $_clearField(7);
  @$pb.TagNumber(7)
  FileUploadResumeHandle ensureResume() => $_ensure(6);

  @$pb.TagNumber(8)
  $core.int get chunkBytes => $_getIZ(7);
  @$pb.TagNumber(8)
  set chunkBytes($core.int value) => $_setUnsignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasChunkBytes() => $_has(7);
  @$pb.TagNumber(8)
  void clearChunkBytes() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get windowBytes => $_getI64(8);
  @$pb.TagNumber(9)
  set windowBytes($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasWindowBytes() => $_has(8);
  @$pb.TagNumber(9)
  void clearWindowBytes() => $_clearField(9);

  /// content_encoding 是服务端为本次 transfer data stream 选定的编码（""=identity）。
  /// 旧客户端忽略该字段；未协商时为空，帧字节与旧行为一致。
  @$pb.TagNumber(10)
  $core.String get contentEncoding => $_getSZ(9);
  @$pb.TagNumber(10)
  set contentEncoding($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasContentEncoding() => $_has(9);
  @$pb.TagNumber(10)
  void clearContentEncoding() => $_clearField(10);

  /// progress_interval_bytes 回显服务端实际采用的进度合并窗口；0 = 未启用。
  @$pb.TagNumber(11)
  $fixnum.Int64 get progressIntervalBytes => $_getI64(10);
  @$pb.TagNumber(11)
  set progressIntervalBytes($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasProgressIntervalBytes() => $_has(10);
  @$pb.TagNumber(11)
  void clearProgressIntervalBytes() => $_clearField(11);
}

class FileTransferOpenResult extends $pb.GeneratedMessage {
  factory FileTransferOpenResult({
    FileTransferHandle? transfer,
  }) {
    final result = FileTransferOpenResult._();
    if (transfer != null) result.transfer = transfer;
    return result;
  }

  FileTransferOpenResult._();

  factory FileTransferOpenResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferOpenResult()..mergeFromBuffer(data, registry);
  factory FileTransferOpenResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferOpenResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferOpenResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileTransferOpenResult.$_createMessage)
    ..aOM<FileTransferHandle>(1, _omitFieldNames ? '' : 'transfer',
        subBuilder: FileTransferHandle.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferOpenResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferOpenResult copyWith(
          void Function(FileTransferOpenResult) updates) =>
      super.copyWith((message) => updates(message as FileTransferOpenResult))
          as FileTransferOpenResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileTransferOpenResult() / FileTransferOpenResult.new instead')
  static FileTransferOpenResult create() => FileTransferOpenResult._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferOpenResult._();
  @$core.override
  FileTransferOpenResult createEmptyInstance() => FileTransferOpenResult._();
  @$core.pragma('dart2js:noInline')
  static FileTransferOpenResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferOpenResult>(
          FileTransferOpenResult.$_createMessage);
  static FileTransferOpenResult? _defaultInstance;

  @$pb.TagNumber(1)
  FileTransferHandle get transfer => $_getN(0);
  @$pb.TagNumber(1)
  set transfer(FileTransferHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTransfer() => $_has(0);
  @$pb.TagNumber(1)
  void clearTransfer() => $_clearField(1);
  @$pb.TagNumber(1)
  FileTransferHandle ensureTransfer() => $_ensure(0);
}

class FileTransferCancelResult extends $pb.GeneratedMessage {
  factory FileTransferCancelResult({
    $core.bool? cancelled,
  }) {
    final result = FileTransferCancelResult._();
    if (cancelled != null) result.cancelled = cancelled;
    return result;
  }

  FileTransferCancelResult._();

  factory FileTransferCancelResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferCancelResult()..mergeFromBuffer(data, registry);
  factory FileTransferCancelResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferCancelResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferCancelResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileTransferCancelResult.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'cancelled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferCancelResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferCancelResult copyWith(
          void Function(FileTransferCancelResult) updates) =>
      super.copyWith((message) => updates(message as FileTransferCancelResult))
          as FileTransferCancelResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileTransferCancelResult() / FileTransferCancelResult.new instead')
  static FileTransferCancelResult create() => FileTransferCancelResult._();
  static $pb.GeneratedMessage $_createMessage() => FileTransferCancelResult._();
  @$core.override
  FileTransferCancelResult createEmptyInstance() =>
      FileTransferCancelResult._();
  @$core.pragma('dart2js:noInline')
  static FileTransferCancelResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferCancelResult>(
          FileTransferCancelResult.$_createMessage);
  static FileTransferCancelResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get cancelled => $_getBF(0);
  @$pb.TagNumber(1)
  set cancelled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCancelled() => $_has(0);
  @$pb.TagNumber(1)
  void clearCancelled() => $_clearField(1);
}

class FileTransferCompletedEvent extends $pb.GeneratedMessage {
  factory FileTransferCompletedEvent({
    FileTransferHandle? transfer,
    $fixnum.Int64? size,
    $core.List<$core.int>? sha256,
  }) {
    final result = FileTransferCompletedEvent._();
    if (transfer != null) result.transfer = transfer;
    if (size != null) result.size = size;
    if (sha256 != null) result.sha256 = sha256;
    return result;
  }

  FileTransferCompletedEvent._();

  factory FileTransferCompletedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferCompletedEvent()..mergeFromBuffer(data, registry);
  factory FileTransferCompletedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FileTransferCompletedEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileTransferCompletedEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: FileTransferCompletedEvent.$_createMessage)
    ..aOM<FileTransferHandle>(1, _omitFieldNames ? '' : 'transfer',
        subBuilder: FileTransferHandle.$_createMessage)
    ..aInt64(2, _omitFieldNames ? '' : 'size')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'sha256', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferCompletedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileTransferCompletedEvent copyWith(
          void Function(FileTransferCompletedEvent) updates) =>
      super.copyWith(
              (message) => updates(message as FileTransferCompletedEvent))
          as FileTransferCompletedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use FileTransferCompletedEvent() / FileTransferCompletedEvent.new instead')
  static FileTransferCompletedEvent create() => FileTransferCompletedEvent._();
  static $pb.GeneratedMessage $_createMessage() =>
      FileTransferCompletedEvent._();
  @$core.override
  FileTransferCompletedEvent createEmptyInstance() =>
      FileTransferCompletedEvent._();
  @$core.pragma('dart2js:noInline')
  static FileTransferCompletedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileTransferCompletedEvent>(
          FileTransferCompletedEvent.$_createMessage);
  static FileTransferCompletedEvent? _defaultInstance;

  @$pb.TagNumber(1)
  FileTransferHandle get transfer => $_getN(0);
  @$pb.TagNumber(1)
  set transfer(FileTransferHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTransfer() => $_has(0);
  @$pb.TagNumber(1)
  void clearTransfer() => $_clearField(1);
  @$pb.TagNumber(1)
  FileTransferHandle ensureTransfer() => $_ensure(0);

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
