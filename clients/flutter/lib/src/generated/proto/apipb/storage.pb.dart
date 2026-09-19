// This is a generated file - do not edit.
//
// Generated from apipb/storage.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'storage.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'storage.pbenum.dart';

class StorageKey extends $pb.GeneratedMessage {
  factory StorageKey({
    $core.String? appId,
    StorageScope? scope,
    $core.String? ownerId,
    $core.String? key,
  }) {
    final result = StorageKey._();
    if (appId != null) result.appId = appId;
    if (scope != null) result.scope = scope;
    if (ownerId != null) result.ownerId = ownerId;
    if (key != null) result.key = key;
    return result;
  }

  StorageKey._();

  factory StorageKey.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageKey()..mergeFromBuffer(data, registry);
  factory StorageKey.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageKey()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageKey',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageKey.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'appId')
    ..aE<StorageScope>(2, _omitFieldNames ? '' : 'scope',
        enumValues: StorageScope.values)
    ..aOS(3, _omitFieldNames ? '' : 'ownerId')
    ..aOS(4, _omitFieldNames ? '' : 'key')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageKey clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageKey copyWith(void Function(StorageKey) updates) =>
      super.copyWith((message) => updates(message as StorageKey)) as StorageKey;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StorageKey() / StorageKey.new instead')
  static StorageKey create() => StorageKey._();
  static $pb.GeneratedMessage $_createMessage() => StorageKey._();
  @$core.override
  StorageKey createEmptyInstance() => StorageKey._();
  @$core.pragma('dart2js:noInline')
  static StorageKey getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StorageKey>(StorageKey.$_createMessage);
  static StorageKey? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get appId => $_getSZ(0);
  @$pb.TagNumber(1)
  set appId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAppId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAppId() => $_clearField(1);

  @$pb.TagNumber(2)
  StorageScope get scope => $_getN(1);
  @$pb.TagNumber(2)
  set scope(StorageScope value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasScope() => $_has(1);
  @$pb.TagNumber(2)
  void clearScope() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get ownerId => $_getSZ(2);
  @$pb.TagNumber(3)
  set ownerId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOwnerId() => $_has(2);
  @$pb.TagNumber(3)
  void clearOwnerId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get key => $_getSZ(3);
  @$pb.TagNumber(4)
  set key($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearKey() => $_clearField(4);
}

class StorageEntry extends $pb.GeneratedMessage {
  factory StorageEntry({
    StorageKey? key,
    $core.List<$core.int>? value,
    $fixnum.Int64? version,
    $fixnum.Int64? updatedAtUnixNano,
  }) {
    final result = StorageEntry._();
    if (key != null) result.key = key;
    if (value != null) result.value = value;
    if (version != null) result.version = version;
    if (updatedAtUnixNano != null) result.updatedAtUnixNano = updatedAtUnixNano;
    return result;
  }

  StorageEntry._();

  factory StorageEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageEntry()..mergeFromBuffer(data, registry);
  factory StorageEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageEntry()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageEntry',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageEntry.$_createMessage)
    ..aOM<StorageKey>(1, _omitFieldNames ? '' : 'key',
        subBuilder: StorageKey.$_createMessage)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'value', $pb.PbFieldType.OY)
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aInt64(4, _omitFieldNames ? '' : 'updatedAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageEntry copyWith(void Function(StorageEntry) updates) =>
      super.copyWith((message) => updates(message as StorageEntry))
          as StorageEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StorageEntry() / StorageEntry.new instead')
  static StorageEntry create() => StorageEntry._();
  static $pb.GeneratedMessage $_createMessage() => StorageEntry._();
  @$core.override
  StorageEntry createEmptyInstance() => StorageEntry._();
  @$core.pragma('dart2js:noInline')
  static StorageEntry getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StorageEntry>(
          StorageEntry.$_createMessage);
  static StorageEntry? _defaultInstance;

  @$pb.TagNumber(1)
  StorageKey get key => $_getN(0);
  @$pb.TagNumber(1)
  set key(StorageKey value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(1)
  void clearKey() => $_clearField(1);
  @$pb.TagNumber(1)
  StorageKey ensureKey() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.List<$core.int> get value => $_getN(1);
  @$pb.TagNumber(2)
  set value($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(2)
  void clearValue() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get version => $_getI64(2);
  @$pb.TagNumber(3)
  set version($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get updatedAtUnixNano => $_getI64(3);
  @$pb.TagNumber(4)
  set updatedAtUnixNano($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUpdatedAtUnixNano() => $_has(3);
  @$pb.TagNumber(4)
  void clearUpdatedAtUnixNano() => $_clearField(4);
}

class StorageVersionFence extends $pb.GeneratedMessage {
  factory StorageVersionFence({
    $core.bool? checkVersion,
    $fixnum.Int64? expectedVersion,
  }) {
    final result = StorageVersionFence._();
    if (checkVersion != null) result.checkVersion = checkVersion;
    if (expectedVersion != null) result.expectedVersion = expectedVersion;
    return result;
  }

  StorageVersionFence._();

  factory StorageVersionFence.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageVersionFence()..mergeFromBuffer(data, registry);
  factory StorageVersionFence.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageVersionFence()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageVersionFence',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageVersionFence.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'checkVersion')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'expectedVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageVersionFence clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageVersionFence copyWith(void Function(StorageVersionFence) updates) =>
      super.copyWith((message) => updates(message as StorageVersionFence))
          as StorageVersionFence;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use StorageVersionFence() / StorageVersionFence.new instead')
  static StorageVersionFence create() => StorageVersionFence._();
  static $pb.GeneratedMessage $_createMessage() => StorageVersionFence._();
  @$core.override
  StorageVersionFence createEmptyInstance() => StorageVersionFence._();
  @$core.pragma('dart2js:noInline')
  static StorageVersionFence getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StorageVersionFence>(
          StorageVersionFence.$_createMessage);
  static StorageVersionFence? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get checkVersion => $_getBF(0);
  @$pb.TagNumber(1)
  set checkVersion($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCheckVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearCheckVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get expectedVersion => $_getI64(1);
  @$pb.TagNumber(2)
  set expectedVersion($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExpectedVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpectedVersion() => $_clearField(2);
}

class StorageGetCommand extends $pb.GeneratedMessage {
  factory StorageGetCommand({
    StorageKey? key,
  }) {
    final result = StorageGetCommand._();
    if (key != null) result.key = key;
    return result;
  }

  StorageGetCommand._();

  factory StorageGetCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageGetCommand()..mergeFromBuffer(data, registry);
  factory StorageGetCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageGetCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageGetCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageGetCommand.$_createMessage)
    ..aOM<StorageKey>(2, _omitFieldNames ? '' : 'key',
        subBuilder: StorageKey.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageGetCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageGetCommand copyWith(void Function(StorageGetCommand) updates) =>
      super.copyWith((message) => updates(message as StorageGetCommand))
          as StorageGetCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StorageGetCommand() / StorageGetCommand.new instead')
  static StorageGetCommand create() => StorageGetCommand._();
  static $pb.GeneratedMessage $_createMessage() => StorageGetCommand._();
  @$core.override
  StorageGetCommand createEmptyInstance() => StorageGetCommand._();
  @$core.pragma('dart2js:noInline')
  static StorageGetCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StorageGetCommand>(
          StorageGetCommand.$_createMessage);
  static StorageGetCommand? _defaultInstance;

  @$pb.TagNumber(2)
  StorageKey get key => $_getN(0);
  @$pb.TagNumber(2)
  set key(StorageKey value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(2)
  void clearKey() => $_clearField(2);
  @$pb.TagNumber(2)
  StorageKey ensureKey() => $_ensure(0);
}

class StoragePutCommand extends $pb.GeneratedMessage {
  factory StoragePutCommand({
    StorageKey? key,
    $core.List<$core.int>? value,
    StorageVersionFence? version,
  }) {
    final result = StoragePutCommand._();
    if (key != null) result.key = key;
    if (value != null) result.value = value;
    if (version != null) result.version = version;
    return result;
  }

  StoragePutCommand._();

  factory StoragePutCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StoragePutCommand()..mergeFromBuffer(data, registry);
  factory StoragePutCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StoragePutCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoragePutCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StoragePutCommand.$_createMessage)
    ..aOM<StorageKey>(2, _omitFieldNames ? '' : 'key',
        subBuilder: StorageKey.$_createMessage)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'value', $pb.PbFieldType.OY)
    ..aOM<StorageVersionFence>(4, _omitFieldNames ? '' : 'version',
        subBuilder: StorageVersionFence.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoragePutCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoragePutCommand copyWith(void Function(StoragePutCommand) updates) =>
      super.copyWith((message) => updates(message as StoragePutCommand))
          as StoragePutCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StoragePutCommand() / StoragePutCommand.new instead')
  static StoragePutCommand create() => StoragePutCommand._();
  static $pb.GeneratedMessage $_createMessage() => StoragePutCommand._();
  @$core.override
  StoragePutCommand createEmptyInstance() => StoragePutCommand._();
  @$core.pragma('dart2js:noInline')
  static StoragePutCommand getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StoragePutCommand>(
          StoragePutCommand.$_createMessage);
  static StoragePutCommand? _defaultInstance;

  @$pb.TagNumber(2)
  StorageKey get key => $_getN(0);
  @$pb.TagNumber(2)
  set key(StorageKey value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(2)
  void clearKey() => $_clearField(2);
  @$pb.TagNumber(2)
  StorageKey ensureKey() => $_ensure(0);

  @$pb.TagNumber(3)
  $core.List<$core.int> get value => $_getN(1);
  @$pb.TagNumber(3)
  set value($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(3)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(3)
  void clearValue() => $_clearField(3);

  @$pb.TagNumber(4)
  StorageVersionFence get version => $_getN(2);
  @$pb.TagNumber(4)
  set version(StorageVersionFence value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(4)
  void clearVersion() => $_clearField(4);
  @$pb.TagNumber(4)
  StorageVersionFence ensureVersion() => $_ensure(2);
}

class StorageDeleteCommand extends $pb.GeneratedMessage {
  factory StorageDeleteCommand({
    StorageKey? key,
    StorageVersionFence? version,
  }) {
    final result = StorageDeleteCommand._();
    if (key != null) result.key = key;
    if (version != null) result.version = version;
    return result;
  }

  StorageDeleteCommand._();

  factory StorageDeleteCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageDeleteCommand()..mergeFromBuffer(data, registry);
  factory StorageDeleteCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageDeleteCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageDeleteCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageDeleteCommand.$_createMessage)
    ..aOM<StorageKey>(2, _omitFieldNames ? '' : 'key',
        subBuilder: StorageKey.$_createMessage)
    ..aOM<StorageVersionFence>(3, _omitFieldNames ? '' : 'version',
        subBuilder: StorageVersionFence.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageDeleteCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageDeleteCommand copyWith(void Function(StorageDeleteCommand) updates) =>
      super.copyWith((message) => updates(message as StorageDeleteCommand))
          as StorageDeleteCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use StorageDeleteCommand() / StorageDeleteCommand.new instead')
  static StorageDeleteCommand create() => StorageDeleteCommand._();
  static $pb.GeneratedMessage $_createMessage() => StorageDeleteCommand._();
  @$core.override
  StorageDeleteCommand createEmptyInstance() => StorageDeleteCommand._();
  @$core.pragma('dart2js:noInline')
  static StorageDeleteCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StorageDeleteCommand>(
          StorageDeleteCommand.$_createMessage);
  static StorageDeleteCommand? _defaultInstance;

  @$pb.TagNumber(2)
  StorageKey get key => $_getN(0);
  @$pb.TagNumber(2)
  set key(StorageKey value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(2)
  void clearKey() => $_clearField(2);
  @$pb.TagNumber(2)
  StorageKey ensureKey() => $_ensure(0);

  @$pb.TagNumber(3)
  StorageVersionFence get version => $_getN(1);
  @$pb.TagNumber(3)
  set version(StorageVersionFence value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(3)
  void clearVersion() => $_clearField(3);
  @$pb.TagNumber(3)
  StorageVersionFence ensureVersion() => $_ensure(1);
}

class StorageListCommand extends $pb.GeneratedMessage {
  factory StorageListCommand({
    $core.String? appId,
    StorageScope? scope,
    $core.String? ownerId,
    $core.String? prefix,
  }) {
    final result = StorageListCommand._();
    if (appId != null) result.appId = appId;
    if (scope != null) result.scope = scope;
    if (ownerId != null) result.ownerId = ownerId;
    if (prefix != null) result.prefix = prefix;
    return result;
  }

  StorageListCommand._();

  factory StorageListCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageListCommand()..mergeFromBuffer(data, registry);
  factory StorageListCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageListCommand()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageListCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageListCommand.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'appId')
    ..aE<StorageScope>(3, _omitFieldNames ? '' : 'scope',
        enumValues: StorageScope.values)
    ..aOS(4, _omitFieldNames ? '' : 'ownerId')
    ..aOS(5, _omitFieldNames ? '' : 'prefix')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageListCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageListCommand copyWith(void Function(StorageListCommand) updates) =>
      super.copyWith((message) => updates(message as StorageListCommand))
          as StorageListCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StorageListCommand() / StorageListCommand.new instead')
  static StorageListCommand create() => StorageListCommand._();
  static $pb.GeneratedMessage $_createMessage() => StorageListCommand._();
  @$core.override
  StorageListCommand createEmptyInstance() => StorageListCommand._();
  @$core.pragma('dart2js:noInline')
  static StorageListCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StorageListCommand>(
          StorageListCommand.$_createMessage);
  static StorageListCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get appId => $_getSZ(0);
  @$pb.TagNumber(2)
  set appId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasAppId() => $_has(0);
  @$pb.TagNumber(2)
  void clearAppId() => $_clearField(2);

  @$pb.TagNumber(3)
  StorageScope get scope => $_getN(1);
  @$pb.TagNumber(3)
  set scope(StorageScope value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasScope() => $_has(1);
  @$pb.TagNumber(3)
  void clearScope() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get ownerId => $_getSZ(2);
  @$pb.TagNumber(4)
  set ownerId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(4)
  $core.bool hasOwnerId() => $_has(2);
  @$pb.TagNumber(4)
  void clearOwnerId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get prefix => $_getSZ(3);
  @$pb.TagNumber(5)
  set prefix($core.String value) => $_setString(3, value);
  @$pb.TagNumber(5)
  $core.bool hasPrefix() => $_has(3);
  @$pb.TagNumber(5)
  void clearPrefix() => $_clearField(5);
}

class StorageGetResult extends $pb.GeneratedMessage {
  factory StorageGetResult({
    StorageEntry? entry,
  }) {
    final result = StorageGetResult._();
    if (entry != null) result.entry = entry;
    return result;
  }

  StorageGetResult._();

  factory StorageGetResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageGetResult()..mergeFromBuffer(data, registry);
  factory StorageGetResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageGetResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageGetResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageGetResult.$_createMessage)
    ..aOM<StorageEntry>(1, _omitFieldNames ? '' : 'entry',
        subBuilder: StorageEntry.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageGetResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageGetResult copyWith(void Function(StorageGetResult) updates) =>
      super.copyWith((message) => updates(message as StorageGetResult))
          as StorageGetResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StorageGetResult() / StorageGetResult.new instead')
  static StorageGetResult create() => StorageGetResult._();
  static $pb.GeneratedMessage $_createMessage() => StorageGetResult._();
  @$core.override
  StorageGetResult createEmptyInstance() => StorageGetResult._();
  @$core.pragma('dart2js:noInline')
  static StorageGetResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StorageGetResult>(
          StorageGetResult.$_createMessage);
  static StorageGetResult? _defaultInstance;

  @$pb.TagNumber(1)
  StorageEntry get entry => $_getN(0);
  @$pb.TagNumber(1)
  set entry(StorageEntry value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEntry() => $_has(0);
  @$pb.TagNumber(1)
  void clearEntry() => $_clearField(1);
  @$pb.TagNumber(1)
  StorageEntry ensureEntry() => $_ensure(0);
}

class StoragePutResult extends $pb.GeneratedMessage {
  factory StoragePutResult({
    StorageEntry? entry,
  }) {
    final result = StoragePutResult._();
    if (entry != null) result.entry = entry;
    return result;
  }

  StoragePutResult._();

  factory StoragePutResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StoragePutResult()..mergeFromBuffer(data, registry);
  factory StoragePutResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StoragePutResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StoragePutResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StoragePutResult.$_createMessage)
    ..aOM<StorageEntry>(1, _omitFieldNames ? '' : 'entry',
        subBuilder: StorageEntry.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoragePutResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StoragePutResult copyWith(void Function(StoragePutResult) updates) =>
      super.copyWith((message) => updates(message as StoragePutResult))
          as StoragePutResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StoragePutResult() / StoragePutResult.new instead')
  static StoragePutResult create() => StoragePutResult._();
  static $pb.GeneratedMessage $_createMessage() => StoragePutResult._();
  @$core.override
  StoragePutResult createEmptyInstance() => StoragePutResult._();
  @$core.pragma('dart2js:noInline')
  static StoragePutResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StoragePutResult>(
          StoragePutResult.$_createMessage);
  static StoragePutResult? _defaultInstance;

  @$pb.TagNumber(1)
  StorageEntry get entry => $_getN(0);
  @$pb.TagNumber(1)
  set entry(StorageEntry value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEntry() => $_has(0);
  @$pb.TagNumber(1)
  void clearEntry() => $_clearField(1);
  @$pb.TagNumber(1)
  StorageEntry ensureEntry() => $_ensure(0);
}

class StorageDeleteResult extends $pb.GeneratedMessage {
  factory StorageDeleteResult({
    StorageKey? key,
    $core.bool? deleted,
    $fixnum.Int64? version,
  }) {
    final result = StorageDeleteResult._();
    if (key != null) result.key = key;
    if (deleted != null) result.deleted = deleted;
    if (version != null) result.version = version;
    return result;
  }

  StorageDeleteResult._();

  factory StorageDeleteResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageDeleteResult()..mergeFromBuffer(data, registry);
  factory StorageDeleteResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageDeleteResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageDeleteResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageDeleteResult.$_createMessage)
    ..aOM<StorageKey>(1, _omitFieldNames ? '' : 'key',
        subBuilder: StorageKey.$_createMessage)
    ..aOB(2, _omitFieldNames ? '' : 'deleted')
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageDeleteResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageDeleteResult copyWith(void Function(StorageDeleteResult) updates) =>
      super.copyWith((message) => updates(message as StorageDeleteResult))
          as StorageDeleteResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use StorageDeleteResult() / StorageDeleteResult.new instead')
  static StorageDeleteResult create() => StorageDeleteResult._();
  static $pb.GeneratedMessage $_createMessage() => StorageDeleteResult._();
  @$core.override
  StorageDeleteResult createEmptyInstance() => StorageDeleteResult._();
  @$core.pragma('dart2js:noInline')
  static StorageDeleteResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StorageDeleteResult>(
          StorageDeleteResult.$_createMessage);
  static StorageDeleteResult? _defaultInstance;

  @$pb.TagNumber(1)
  StorageKey get key => $_getN(0);
  @$pb.TagNumber(1)
  set key(StorageKey value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(1)
  void clearKey() => $_clearField(1);
  @$pb.TagNumber(1)
  StorageKey ensureKey() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get deleted => $_getBF(1);
  @$pb.TagNumber(2)
  set deleted($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeleted() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeleted() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get version => $_getI64(2);
  @$pb.TagNumber(3)
  set version($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearVersion() => $_clearField(3);
}

class StorageListResult extends $pb.GeneratedMessage {
  factory StorageListResult({
    $core.Iterable<StorageEntry>? entries,
  }) {
    final result = StorageListResult._();
    if (entries != null) result.entries.addAll(entries);
    return result;
  }

  StorageListResult._();

  factory StorageListResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageListResult()..mergeFromBuffer(data, registry);
  factory StorageListResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageListResult()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageListResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageListResult.$_createMessage)
    ..pPM<StorageEntry>(1, _omitFieldNames ? '' : 'entries',
        subBuilder: StorageEntry.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageListResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageListResult copyWith(void Function(StorageListResult) updates) =>
      super.copyWith((message) => updates(message as StorageListResult))
          as StorageListResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StorageListResult() / StorageListResult.new instead')
  static StorageListResult create() => StorageListResult._();
  static $pb.GeneratedMessage $_createMessage() => StorageListResult._();
  @$core.override
  StorageListResult createEmptyInstance() => StorageListResult._();
  @$core.pragma('dart2js:noInline')
  static StorageListResult getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StorageListResult>(
          StorageListResult.$_createMessage);
  static StorageListResult? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<StorageEntry> get entries => $_getList(0);
}

class StorageChangedEvent extends $pb.GeneratedMessage {
  factory StorageChangedEvent({
    StorageKey? key,
    $fixnum.Int64? version,
    $core.String? operation,
  }) {
    final result = StorageChangedEvent._();
    if (key != null) result.key = key;
    if (version != null) result.version = version;
    if (operation != null) result.operation = operation;
    return result;
  }

  StorageChangedEvent._();

  factory StorageChangedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageChangedEvent()..mergeFromBuffer(data, registry);
  factory StorageChangedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StorageChangedEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StorageChangedEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: StorageChangedEvent.$_createMessage)
    ..aOM<StorageKey>(1, _omitFieldNames ? '' : 'key',
        subBuilder: StorageKey.$_createMessage)
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'operation')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageChangedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StorageChangedEvent copyWith(void Function(StorageChangedEvent) updates) =>
      super.copyWith((message) => updates(message as StorageChangedEvent))
          as StorageChangedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use StorageChangedEvent() / StorageChangedEvent.new instead')
  static StorageChangedEvent create() => StorageChangedEvent._();
  static $pb.GeneratedMessage $_createMessage() => StorageChangedEvent._();
  @$core.override
  StorageChangedEvent createEmptyInstance() => StorageChangedEvent._();
  @$core.pragma('dart2js:noInline')
  static StorageChangedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StorageChangedEvent>(
          StorageChangedEvent.$_createMessage);
  static StorageChangedEvent? _defaultInstance;

  @$pb.TagNumber(1)
  StorageKey get key => $_getN(0);
  @$pb.TagNumber(1)
  set key(StorageKey value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(1)
  void clearKey() => $_clearField(1);
  @$pb.TagNumber(1)
  StorageKey ensureKey() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get version => $_getI64(1);
  @$pb.TagNumber(2)
  set version($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get operation => $_getSZ(2);
  @$pb.TagNumber(3)
  set operation($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOperation() => $_has(2);
  @$pb.TagNumber(3)
  void clearOperation() => $_clearField(3);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
