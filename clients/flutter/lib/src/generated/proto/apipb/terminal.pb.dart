// This is a generated file - do not edit.
//
// Generated from apipb/terminal.proto.

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
import 'terminal.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'terminal.pbenum.dart';

class TerminalRef extends $pb.GeneratedMessage {
  factory TerminalRef({
    $core.String? endpointId,
    $core.String? terminalId,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (terminalId != null) result.terminalId = terminalId;
    return result;
  }

  TerminalRef._();

  factory TerminalRef.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalRef.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalRef',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'terminalId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalRef clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalRef copyWith(void Function(TerminalRef) updates) =>
      super.copyWith((message) => updates(message as TerminalRef))
          as TerminalRef;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalRef create() => TerminalRef._();
  @$core.override
  TerminalRef createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalRef getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalRef>(create);
  static TerminalRef? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get terminalId => $_getSZ(1);
  @$pb.TagNumber(2)
  set terminalId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminalId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTerminalId() => $_clearField(2);
}

class TerminalSize extends $pb.GeneratedMessage {
  factory TerminalSize({
    $core.int? cols,
    $core.int? rows,
  }) {
    final result = create();
    if (cols != null) result.cols = cols;
    if (rows != null) result.rows = rows;
    return result;
  }

  TerminalSize._();

  factory TerminalSize.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalSize.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalSize',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'cols', fieldType: $pb.PbFieldType.OU3)
    ..aI(2, _omitFieldNames ? '' : 'rows', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalSize clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalSize copyWith(void Function(TerminalSize) updates) =>
      super.copyWith((message) => updates(message as TerminalSize))
          as TerminalSize;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalSize create() => TerminalSize._();
  @$core.override
  TerminalSize createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalSize getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalSize>(create);
  static TerminalSize? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get cols => $_getIZ(0);
  @$pb.TagNumber(1)
  set cols($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCols() => $_has(0);
  @$pb.TagNumber(1)
  void clearCols() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get rows => $_getIZ(1);
  @$pb.TagNumber(2)
  set rows($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRows() => $_has(1);
  @$pb.TagNumber(2)
  void clearRows() => $_clearField(2);
}

class TerminalResourceUsage extends $pb.GeneratedMessage {
  factory TerminalResourceUsage({
    $core.int? pid,
    $core.int? cpuPercentX100,
    $fixnum.Int64? memoryBytes,
    $fixnum.Int64? sampledAtUnixNano,
  }) {
    final result = create();
    if (pid != null) result.pid = pid;
    if (cpuPercentX100 != null) result.cpuPercentX100 = cpuPercentX100;
    if (memoryBytes != null) result.memoryBytes = memoryBytes;
    if (sampledAtUnixNano != null) result.sampledAtUnixNano = sampledAtUnixNano;
    return result;
  }

  TerminalResourceUsage._();

  factory TerminalResourceUsage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalResourceUsage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalResourceUsage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'pid')
    ..aI(2, _omitFieldNames ? '' : 'cpuPercentX100')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'memoryBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aInt64(4, _omitFieldNames ? '' : 'sampledAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResourceUsage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResourceUsage copyWith(
          void Function(TerminalResourceUsage) updates) =>
      super.copyWith((message) => updates(message as TerminalResourceUsage))
          as TerminalResourceUsage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalResourceUsage create() => TerminalResourceUsage._();
  @$core.override
  TerminalResourceUsage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalResourceUsage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalResourceUsage>(create);
  static TerminalResourceUsage? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get pid => $_getIZ(0);
  @$pb.TagNumber(1)
  set pid($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPid() => $_has(0);
  @$pb.TagNumber(1)
  void clearPid() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get cpuPercentX100 => $_getIZ(1);
  @$pb.TagNumber(2)
  set cpuPercentX100($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCpuPercentX100() => $_has(1);
  @$pb.TagNumber(2)
  void clearCpuPercentX100() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get memoryBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set memoryBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMemoryBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearMemoryBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get sampledAtUnixNano => $_getI64(3);
  @$pb.TagNumber(4)
  set sampledAtUnixNano($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSampledAtUnixNano() => $_has(3);
  @$pb.TagNumber(4)
  void clearSampledAtUnixNano() => $_clearField(4);
}

class TerminalInfo extends $pb.GeneratedMessage {
  factory TerminalInfo({
    TerminalRef? ref,
    $core.String? name,
    $core.Iterable<$core.String>? command,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? tags,
    TerminalSize? size,
    TerminalState? state,
    $core.String? cwd,
    $core.String? liveCwd,
    $fixnum.Int64? createdAtUnixNano,
    $core.int? exitCode,
    $fixnum.Int64? exitedAtUnixNano,
    $core.int? attachmentCount,
    TerminalResourceUsage? resources,
    $core.Iterable<TerminalResourceUsage>? resourceHistory,
    $core.String? foregroundProcess,
    $fixnum.Int64? lastOutputAtUnixNano,
    $core.String? foregroundCwd,
  }) {
    final result = create();
    if (ref != null) result.ref = ref;
    if (name != null) result.name = name;
    if (command != null) result.command.addAll(command);
    if (tags != null) result.tags.addEntries(tags);
    if (size != null) result.size = size;
    if (state != null) result.state = state;
    if (cwd != null) result.cwd = cwd;
    if (liveCwd != null) result.liveCwd = liveCwd;
    if (createdAtUnixNano != null) result.createdAtUnixNano = createdAtUnixNano;
    if (exitCode != null) result.exitCode = exitCode;
    if (exitedAtUnixNano != null) result.exitedAtUnixNano = exitedAtUnixNano;
    if (attachmentCount != null) result.attachmentCount = attachmentCount;
    if (resources != null) result.resources = resources;
    if (resourceHistory != null) result.resourceHistory.addAll(resourceHistory);
    if (foregroundProcess != null) result.foregroundProcess = foregroundProcess;
    if (lastOutputAtUnixNano != null)
      result.lastOutputAtUnixNano = lastOutputAtUnixNano;
    if (foregroundCwd != null) result.foregroundCwd = foregroundCwd;
    return result;
  }

  TerminalInfo._();

  factory TerminalInfo.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalInfo.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalInfo',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(1, _omitFieldNames ? '' : 'ref',
        subBuilder: TerminalRef.create)
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..pPS(3, _omitFieldNames ? '' : 'command')
    ..m<$core.String, $core.String>(4, _omitFieldNames ? '' : 'tags',
        entryClassName: 'TerminalInfo.TagsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('anytty.api.v1'))
    ..aOM<TerminalSize>(5, _omitFieldNames ? '' : 'size',
        subBuilder: TerminalSize.create)
    ..aE<TerminalState>(6, _omitFieldNames ? '' : 'state',
        enumValues: TerminalState.values)
    ..aOS(7, _omitFieldNames ? '' : 'cwd')
    ..aOS(8, _omitFieldNames ? '' : 'liveCwd')
    ..aInt64(9, _omitFieldNames ? '' : 'createdAtUnixNano')
    ..aI(10, _omitFieldNames ? '' : 'exitCode')
    ..aInt64(11, _omitFieldNames ? '' : 'exitedAtUnixNano')
    ..aI(12, _omitFieldNames ? '' : 'attachmentCount')
    ..aOM<TerminalResourceUsage>(13, _omitFieldNames ? '' : 'resources',
        subBuilder: TerminalResourceUsage.create)
    ..pPM<TerminalResourceUsage>(14, _omitFieldNames ? '' : 'resourceHistory',
        subBuilder: TerminalResourceUsage.create)
    ..aOS(15, _omitFieldNames ? '' : 'foregroundProcess')
    ..aInt64(16, _omitFieldNames ? '' : 'lastOutputAtUnixNano')
    ..aOS(17, _omitFieldNames ? '' : 'foregroundCwd')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalInfo clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalInfo copyWith(void Function(TerminalInfo) updates) =>
      super.copyWith((message) => updates(message as TerminalInfo))
          as TerminalInfo;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalInfo create() => TerminalInfo._();
  @$core.override
  TerminalInfo createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalInfo getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalInfo>(create);
  static TerminalInfo? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalRef get ref => $_getN(0);
  @$pb.TagNumber(1)
  set ref(TerminalRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRef() => $_has(0);
  @$pb.TagNumber(1)
  void clearRef() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalRef ensureRef() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get command => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbMap<$core.String, $core.String> get tags => $_getMap(3);

  @$pb.TagNumber(5)
  TerminalSize get size => $_getN(4);
  @$pb.TagNumber(5)
  set size(TerminalSize value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasSize() => $_has(4);
  @$pb.TagNumber(5)
  void clearSize() => $_clearField(5);
  @$pb.TagNumber(5)
  TerminalSize ensureSize() => $_ensure(4);

  @$pb.TagNumber(6)
  TerminalState get state => $_getN(5);
  @$pb.TagNumber(6)
  set state(TerminalState value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasState() => $_has(5);
  @$pb.TagNumber(6)
  void clearState() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get cwd => $_getSZ(6);
  @$pb.TagNumber(7)
  set cwd($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCwd() => $_has(6);
  @$pb.TagNumber(7)
  void clearCwd() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get liveCwd => $_getSZ(7);
  @$pb.TagNumber(8)
  set liveCwd($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasLiveCwd() => $_has(7);
  @$pb.TagNumber(8)
  void clearLiveCwd() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get createdAtUnixNano => $_getI64(8);
  @$pb.TagNumber(9)
  set createdAtUnixNano($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasCreatedAtUnixNano() => $_has(8);
  @$pb.TagNumber(9)
  void clearCreatedAtUnixNano() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get exitCode => $_getIZ(9);
  @$pb.TagNumber(10)
  set exitCode($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasExitCode() => $_has(9);
  @$pb.TagNumber(10)
  void clearExitCode() => $_clearField(10);

  @$pb.TagNumber(11)
  $fixnum.Int64 get exitedAtUnixNano => $_getI64(10);
  @$pb.TagNumber(11)
  set exitedAtUnixNano($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasExitedAtUnixNano() => $_has(10);
  @$pb.TagNumber(11)
  void clearExitedAtUnixNano() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.int get attachmentCount => $_getIZ(11);
  @$pb.TagNumber(12)
  set attachmentCount($core.int value) => $_setSignedInt32(11, value);
  @$pb.TagNumber(12)
  $core.bool hasAttachmentCount() => $_has(11);
  @$pb.TagNumber(12)
  void clearAttachmentCount() => $_clearField(12);

  @$pb.TagNumber(13)
  TerminalResourceUsage get resources => $_getN(12);
  @$pb.TagNumber(13)
  set resources(TerminalResourceUsage value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasResources() => $_has(12);
  @$pb.TagNumber(13)
  void clearResources() => $_clearField(13);
  @$pb.TagNumber(13)
  TerminalResourceUsage ensureResources() => $_ensure(12);

  @$pb.TagNumber(14)
  $pb.PbList<TerminalResourceUsage> get resourceHistory => $_getList(13);

  /// foreground_process 是 daemon 归一化后的前台进程名，不包含命令行参数。
  @$pb.TagNumber(15)
  $core.String get foregroundProcess => $_getSZ(14);
  @$pb.TagNumber(15)
  set foregroundProcess($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasForegroundProcess() => $_has(14);
  @$pb.TagNumber(15)
  void clearForegroundProcess() => $_clearField(15);

  /// last_output_at_unix_nano 是该 terminal 最近一次产生非空 PTY 输出的时间。
  @$pb.TagNumber(16)
  $fixnum.Int64 get lastOutputAtUnixNano => $_getI64(15);
  @$pb.TagNumber(16)
  set lastOutputAtUnixNano($fixnum.Int64 value) => $_setInt64(15, value);
  @$pb.TagNumber(16)
  $core.bool hasLastOutputAtUnixNano() => $_has(15);
  @$pb.TagNumber(16)
  void clearLastOutputAtUnixNano() => $_clearField(16);

  /// foreground_cwd 是 daemon 在查询时从 PTY 前台进程读取的真实工作目录。
  @$pb.TagNumber(17)
  $core.String get foregroundCwd => $_getSZ(16);
  @$pb.TagNumber(17)
  set foregroundCwd($core.String value) => $_setString(16, value);
  @$pb.TagNumber(17)
  $core.bool hasForegroundCwd() => $_has(16);
  @$pb.TagNumber(17)
  void clearForegroundCwd() => $_clearField(17);
}

class TerminalCreateSpec extends $pb.GeneratedMessage {
  factory TerminalCreateSpec({
    $core.String? terminalId,
    $core.String? name,
    $core.Iterable<$core.String>? command,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? tags,
    TerminalSize? size,
    $core.String? cwd,
    $core.Iterable<$core.String>? env,
    $core.int? scrollbackRows,
    $fixnum.Int64? scrollbackMaxBytes,
    $fixnum.Int64? scrollbackMaxAgeSeconds,
  }) {
    final result = create();
    if (terminalId != null) result.terminalId = terminalId;
    if (name != null) result.name = name;
    if (command != null) result.command.addAll(command);
    if (tags != null) result.tags.addEntries(tags);
    if (size != null) result.size = size;
    if (cwd != null) result.cwd = cwd;
    if (env != null) result.env.addAll(env);
    if (scrollbackRows != null) result.scrollbackRows = scrollbackRows;
    if (scrollbackMaxBytes != null)
      result.scrollbackMaxBytes = scrollbackMaxBytes;
    if (scrollbackMaxAgeSeconds != null)
      result.scrollbackMaxAgeSeconds = scrollbackMaxAgeSeconds;
    return result;
  }

  TerminalCreateSpec._();

  factory TerminalCreateSpec.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalCreateSpec.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalCreateSpec',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'terminalId')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..pPS(3, _omitFieldNames ? '' : 'command')
    ..m<$core.String, $core.String>(4, _omitFieldNames ? '' : 'tags',
        entryClassName: 'TerminalCreateSpec.TagsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('anytty.api.v1'))
    ..aOM<TerminalSize>(5, _omitFieldNames ? '' : 'size',
        subBuilder: TerminalSize.create)
    ..aOS(6, _omitFieldNames ? '' : 'cwd')
    ..pPS(7, _omitFieldNames ? '' : 'env')
    ..aI(8, _omitFieldNames ? '' : 'scrollbackRows')
    ..aInt64(9, _omitFieldNames ? '' : 'scrollbackMaxBytes')
    ..aInt64(10, _omitFieldNames ? '' : 'scrollbackMaxAgeSeconds')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCreateSpec clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCreateSpec copyWith(void Function(TerminalCreateSpec) updates) =>
      super.copyWith((message) => updates(message as TerminalCreateSpec))
          as TerminalCreateSpec;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalCreateSpec create() => TerminalCreateSpec._();
  @$core.override
  TerminalCreateSpec createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalCreateSpec getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalCreateSpec>(create);
  static TerminalCreateSpec? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get terminalId => $_getSZ(0);
  @$pb.TagNumber(1)
  set terminalId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminalId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminalId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get command => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbMap<$core.String, $core.String> get tags => $_getMap(3);

  @$pb.TagNumber(5)
  TerminalSize get size => $_getN(4);
  @$pb.TagNumber(5)
  set size(TerminalSize value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasSize() => $_has(4);
  @$pb.TagNumber(5)
  void clearSize() => $_clearField(5);
  @$pb.TagNumber(5)
  TerminalSize ensureSize() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get cwd => $_getSZ(5);
  @$pb.TagNumber(6)
  set cwd($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCwd() => $_has(5);
  @$pb.TagNumber(6)
  void clearCwd() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get env => $_getList(6);

  @$pb.TagNumber(8)
  $core.int get scrollbackRows => $_getIZ(7);
  @$pb.TagNumber(8)
  set scrollbackRows($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasScrollbackRows() => $_has(7);
  @$pb.TagNumber(8)
  void clearScrollbackRows() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get scrollbackMaxBytes => $_getI64(8);
  @$pb.TagNumber(9)
  set scrollbackMaxBytes($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasScrollbackMaxBytes() => $_has(8);
  @$pb.TagNumber(9)
  void clearScrollbackMaxBytes() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get scrollbackMaxAgeSeconds => $_getI64(9);
  @$pb.TagNumber(10)
  set scrollbackMaxAgeSeconds($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasScrollbackMaxAgeSeconds() => $_has(9);
  @$pb.TagNumber(10)
  void clearScrollbackMaxAgeSeconds() => $_clearField(10);
}

class TerminalDefaults extends $pb.GeneratedMessage {
  factory TerminalDefaults({
    $core.Iterable<$core.String>? defaultCommand,
    $core.String? defaultCwd,
    $core.String? platform,
  }) {
    final result = create();
    if (defaultCommand != null) result.defaultCommand.addAll(defaultCommand);
    if (defaultCwd != null) result.defaultCwd = defaultCwd;
    if (platform != null) result.platform = platform;
    return result;
  }

  TerminalDefaults._();

  factory TerminalDefaults.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalDefaults.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalDefaults',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'defaultCommand')
    ..aOS(2, _omitFieldNames ? '' : 'defaultCwd')
    ..aOS(3, _omitFieldNames ? '' : 'platform')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDefaults clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDefaults copyWith(void Function(TerminalDefaults) updates) =>
      super.copyWith((message) => updates(message as TerminalDefaults))
          as TerminalDefaults;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalDefaults create() => TerminalDefaults._();
  @$core.override
  TerminalDefaults createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalDefaults getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalDefaults>(create);
  static TerminalDefaults? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get defaultCommand => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get defaultCwd => $_getSZ(1);
  @$pb.TagNumber(2)
  set defaultCwd($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDefaultCwd() => $_has(1);
  @$pb.TagNumber(2)
  void clearDefaultCwd() => $_clearField(2);

  /// platform 是 owning daemon 的规范化 OS family（例如 darwin/linux/windows）。
  @$pb.TagNumber(3)
  $core.String get platform => $_getSZ(2);
  @$pb.TagNumber(3)
  set platform($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlatform() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlatform() => $_clearField(3);
}

class ResizeOwnership extends $pb.GeneratedMessage {
  factory ResizeOwnership({
    $core.String? ownerAttachmentId,
    $core.String? ownerSurfaceId,
    $core.String? ownerViewId,
    TerminalSize? size,
    $core.bool? sizeLocked,
    $fixnum.Int64? epoch,
  }) {
    final result = create();
    if (ownerAttachmentId != null) result.ownerAttachmentId = ownerAttachmentId;
    if (ownerSurfaceId != null) result.ownerSurfaceId = ownerSurfaceId;
    if (ownerViewId != null) result.ownerViewId = ownerViewId;
    if (size != null) result.size = size;
    if (sizeLocked != null) result.sizeLocked = sizeLocked;
    if (epoch != null) result.epoch = epoch;
    return result;
  }

  ResizeOwnership._();

  factory ResizeOwnership.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResizeOwnership.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResizeOwnership',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'ownerAttachmentId')
    ..aOS(2, _omitFieldNames ? '' : 'ownerSurfaceId')
    ..aOS(3, _omitFieldNames ? '' : 'ownerViewId')
    ..aOM<TerminalSize>(4, _omitFieldNames ? '' : 'size',
        subBuilder: TerminalSize.create)
    ..aOB(5, _omitFieldNames ? '' : 'sizeLocked')
    ..a<$fixnum.Int64>(6, _omitFieldNames ? '' : 'epoch', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResizeOwnership clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResizeOwnership copyWith(void Function(ResizeOwnership) updates) =>
      super.copyWith((message) => updates(message as ResizeOwnership))
          as ResizeOwnership;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResizeOwnership create() => ResizeOwnership._();
  @$core.override
  ResizeOwnership createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResizeOwnership getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResizeOwnership>(create);
  static ResizeOwnership? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get ownerAttachmentId => $_getSZ(0);
  @$pb.TagNumber(1)
  set ownerAttachmentId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOwnerAttachmentId() => $_has(0);
  @$pb.TagNumber(1)
  void clearOwnerAttachmentId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get ownerSurfaceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set ownerSurfaceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOwnerSurfaceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearOwnerSurfaceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get ownerViewId => $_getSZ(2);
  @$pb.TagNumber(3)
  set ownerViewId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOwnerViewId() => $_has(2);
  @$pb.TagNumber(3)
  void clearOwnerViewId() => $_clearField(3);

  @$pb.TagNumber(4)
  TerminalSize get size => $_getN(3);
  @$pb.TagNumber(4)
  set size(TerminalSize value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSize() => $_has(3);
  @$pb.TagNumber(4)
  void clearSize() => $_clearField(4);
  @$pb.TagNumber(4)
  TerminalSize ensureSize() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.bool get sizeLocked => $_getBF(4);
  @$pb.TagNumber(5)
  set sizeLocked($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSizeLocked() => $_has(4);
  @$pb.TagNumber(5)
  void clearSizeLocked() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get epoch => $_getI64(5);
  @$pb.TagNumber(6)
  set epoch($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasEpoch() => $_has(5);
  @$pb.TagNumber(6)
  void clearEpoch() => $_clearField(6);
}

class ResizeControl extends $pb.GeneratedMessage {
  factory ResizeControl({
    $core.bool? canResize,
    ResizeControlReason? reason,
    $core.bool? sizeLocked,
    $core.String? surfaceId,
    $core.String? ownerSurfaceId,
    $core.String? ownerViewId,
    ResizeOwnership? ownership,
  }) {
    final result = create();
    if (canResize != null) result.canResize = canResize;
    if (reason != null) result.reason = reason;
    if (sizeLocked != null) result.sizeLocked = sizeLocked;
    if (surfaceId != null) result.surfaceId = surfaceId;
    if (ownerSurfaceId != null) result.ownerSurfaceId = ownerSurfaceId;
    if (ownerViewId != null) result.ownerViewId = ownerViewId;
    if (ownership != null) result.ownership = ownership;
    return result;
  }

  ResizeControl._();

  factory ResizeControl.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResizeControl.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResizeControl',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'canResize')
    ..aE<ResizeControlReason>(2, _omitFieldNames ? '' : 'reason',
        enumValues: ResizeControlReason.values)
    ..aOB(3, _omitFieldNames ? '' : 'sizeLocked')
    ..aOS(4, _omitFieldNames ? '' : 'surfaceId')
    ..aOS(5, _omitFieldNames ? '' : 'ownerSurfaceId')
    ..aOS(6, _omitFieldNames ? '' : 'ownerViewId')
    ..aOM<ResizeOwnership>(7, _omitFieldNames ? '' : 'ownership',
        subBuilder: ResizeOwnership.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResizeControl clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResizeControl copyWith(void Function(ResizeControl) updates) =>
      super.copyWith((message) => updates(message as ResizeControl))
          as ResizeControl;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResizeControl create() => ResizeControl._();
  @$core.override
  ResizeControl createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResizeControl getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResizeControl>(create);
  static ResizeControl? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get canResize => $_getBF(0);
  @$pb.TagNumber(1)
  set canResize($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCanResize() => $_has(0);
  @$pb.TagNumber(1)
  void clearCanResize() => $_clearField(1);

  @$pb.TagNumber(2)
  ResizeControlReason get reason => $_getN(1);
  @$pb.TagNumber(2)
  set reason(ResizeControlReason value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasReason() => $_has(1);
  @$pb.TagNumber(2)
  void clearReason() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get sizeLocked => $_getBF(2);
  @$pb.TagNumber(3)
  set sizeLocked($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSizeLocked() => $_has(2);
  @$pb.TagNumber(3)
  void clearSizeLocked() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get surfaceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set surfaceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSurfaceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSurfaceId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get ownerSurfaceId => $_getSZ(4);
  @$pb.TagNumber(5)
  set ownerSurfaceId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOwnerSurfaceId() => $_has(4);
  @$pb.TagNumber(5)
  void clearOwnerSurfaceId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get ownerViewId => $_getSZ(5);
  @$pb.TagNumber(6)
  set ownerViewId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasOwnerViewId() => $_has(5);
  @$pb.TagNumber(6)
  void clearOwnerViewId() => $_clearField(6);

  @$pb.TagNumber(7)
  ResizeOwnership get ownership => $_getN(6);
  @$pb.TagNumber(7)
  set ownership(ResizeOwnership value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasOwnership() => $_has(6);
  @$pb.TagNumber(7)
  void clearOwnership() => $_clearField(7);
  @$pb.TagNumber(7)
  ResizeOwnership ensureOwnership() => $_ensure(6);
}

class AttachmentHandle extends $pb.GeneratedMessage {
  factory AttachmentHandle({
    $0.ResourceHandle? resource,
    TerminalRef? terminal,
    $0.OperationStamp? operation,
    $core.String? surfaceId,
    $core.String? viewId,
  }) {
    final result = create();
    if (resource != null) result.resource = resource;
    if (terminal != null) result.terminal = terminal;
    if (operation != null) result.operation = operation;
    if (surfaceId != null) result.surfaceId = surfaceId;
    if (viewId != null) result.viewId = viewId;
    return result;
  }

  AttachmentHandle._();

  factory AttachmentHandle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AttachmentHandle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AttachmentHandle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(1, _omitFieldNames ? '' : 'resource',
        subBuilder: $0.ResourceHandle.create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..aOM<$0.OperationStamp>(3, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..aOS(4, _omitFieldNames ? '' : 'surfaceId')
    ..aOS(5, _omitFieldNames ? '' : 'viewId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AttachmentHandle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AttachmentHandle copyWith(void Function(AttachmentHandle) updates) =>
      super.copyWith((message) => updates(message as AttachmentHandle))
          as AttachmentHandle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AttachmentHandle create() => AttachmentHandle._();
  @$core.override
  AttachmentHandle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AttachmentHandle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AttachmentHandle>(create);
  static AttachmentHandle? _defaultInstance;

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
  TerminalRef get terminal => $_getN(1);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(1);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(1);

  @$pb.TagNumber(3)
  $0.OperationStamp get operation => $_getN(2);
  @$pb.TagNumber(3)
  set operation($0.OperationStamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasOperation() => $_has(2);
  @$pb.TagNumber(3)
  void clearOperation() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.OperationStamp ensureOperation() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get surfaceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set surfaceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSurfaceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSurfaceId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get viewId => $_getSZ(4);
  @$pb.TagNumber(5)
  set viewId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasViewId() => $_has(4);
  @$pb.TagNumber(5)
  void clearViewId() => $_clearField(5);
}

class PathDirectoryEntry extends $pb.GeneratedMessage {
  factory PathDirectoryEntry({
    $core.String? name,
    $core.String? path,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (path != null) result.path = path;
    return result;
  }

  PathDirectoryEntry._();

  factory PathDirectoryEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PathDirectoryEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PathDirectoryEntry',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'path')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PathDirectoryEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PathDirectoryEntry copyWith(void Function(PathDirectoryEntry) updates) =>
      super.copyWith((message) => updates(message as PathDirectoryEntry))
          as PathDirectoryEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PathDirectoryEntry create() => PathDirectoryEntry._();
  @$core.override
  PathDirectoryEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PathDirectoryEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PathDirectoryEntry>(create);
  static PathDirectoryEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get path => $_getSZ(1);
  @$pb.TagNumber(2)
  set path($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPath() => $_has(1);
  @$pb.TagNumber(2)
  void clearPath() => $_clearField(2);
}

class TerminalDefaultsCommand extends $pb.GeneratedMessage {
  factory TerminalDefaultsCommand() => create();

  TerminalDefaultsCommand._();

  factory TerminalDefaultsCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalDefaultsCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalDefaultsCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDefaultsCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDefaultsCommand copyWith(
          void Function(TerminalDefaultsCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalDefaultsCommand))
          as TerminalDefaultsCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalDefaultsCommand create() => TerminalDefaultsCommand._();
  @$core.override
  TerminalDefaultsCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalDefaultsCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalDefaultsCommand>(create);
  static TerminalDefaultsCommand? _defaultInstance;
}

class TerminalCreateCommand extends $pb.GeneratedMessage {
  factory TerminalCreateCommand({
    TerminalCreateSpec? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalCreateCommand._();

  factory TerminalCreateCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalCreateCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalCreateCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalCreateSpec>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalCreateSpec.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCreateCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCreateCommand copyWith(
          void Function(TerminalCreateCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalCreateCommand))
          as TerminalCreateCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalCreateCommand create() => TerminalCreateCommand._();
  @$core.override
  TerminalCreateCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalCreateCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalCreateCommand>(create);
  static TerminalCreateCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalCreateSpec get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalCreateSpec value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalCreateSpec ensureTerminal() => $_ensure(0);
}

class TerminalListCommand extends $pb.GeneratedMessage {
  factory TerminalListCommand() => create();

  TerminalListCommand._();

  factory TerminalListCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalListCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalListCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalListCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalListCommand copyWith(void Function(TerminalListCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalListCommand))
          as TerminalListCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalListCommand create() => TerminalListCommand._();
  @$core.override
  TerminalListCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalListCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalListCommand>(create);
  static TerminalListCommand? _defaultInstance;
}

class TerminalGetCommand extends $pb.GeneratedMessage {
  factory TerminalGetCommand({
    TerminalRef? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalGetCommand._();

  factory TerminalGetCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalGetCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalGetCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalGetCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalGetCommand copyWith(void Function(TerminalGetCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalGetCommand))
          as TerminalGetCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalGetCommand create() => TerminalGetCommand._();
  @$core.override
  TerminalGetCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalGetCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalGetCommand>(create);
  static TerminalGetCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);
}

class TerminalRestartCommand extends $pb.GeneratedMessage {
  factory TerminalRestartCommand({
    TerminalRef? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalRestartCommand._();

  factory TerminalRestartCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalRestartCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalRestartCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalRestartCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalRestartCommand copyWith(
          void Function(TerminalRestartCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalRestartCommand))
          as TerminalRestartCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalRestartCommand create() => TerminalRestartCommand._();
  @$core.override
  TerminalRestartCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalRestartCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalRestartCommand>(create);
  static TerminalRestartCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);
}

class TerminalKillCommand extends $pb.GeneratedMessage {
  factory TerminalKillCommand({
    TerminalRef? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalKillCommand._();

  factory TerminalKillCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalKillCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalKillCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalKillCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalKillCommand copyWith(void Function(TerminalKillCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalKillCommand))
          as TerminalKillCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalKillCommand create() => TerminalKillCommand._();
  @$core.override
  TerminalKillCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalKillCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalKillCommand>(create);
  static TerminalKillCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);
}

class TerminalRemoveCommand extends $pb.GeneratedMessage {
  factory TerminalRemoveCommand({
    TerminalRef? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalRemoveCommand._();

  factory TerminalRemoveCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalRemoveCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalRemoveCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalRemoveCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalRemoveCommand copyWith(
          void Function(TerminalRemoveCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalRemoveCommand))
          as TerminalRemoveCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalRemoveCommand create() => TerminalRemoveCommand._();
  @$core.override
  TerminalRemoveCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalRemoveCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalRemoveCommand>(create);
  static TerminalRemoveCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);
}

class TerminalSetMetadataCommand extends $pb.GeneratedMessage {
  factory TerminalSetMetadataCommand({
    TerminalRef? terminal,
    $core.String? name,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? tags,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (name != null) result.name = name;
    if (tags != null) result.tags.addEntries(tags);
    return result;
  }

  TerminalSetMetadataCommand._();

  factory TerminalSetMetadataCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalSetMetadataCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalSetMetadataCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..aOS(3, _omitFieldNames ? '' : 'name')
    ..m<$core.String, $core.String>(4, _omitFieldNames ? '' : 'tags',
        entryClassName: 'TerminalSetMetadataCommand.TagsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('anytty.api.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalSetMetadataCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalSetMetadataCommand copyWith(
          void Function(TerminalSetMetadataCommand) updates) =>
      super.copyWith(
              (message) => updates(message as TerminalSetMetadataCommand))
          as TerminalSetMetadataCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalSetMetadataCommand create() => TerminalSetMetadataCommand._();
  @$core.override
  TerminalSetMetadataCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalSetMetadataCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalSetMetadataCommand>(create);
  static TerminalSetMetadataCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(3)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(3)
  void clearName() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbMap<$core.String, $core.String> get tags => $_getMap(2);
}

class TerminalSetTagsCommand extends $pb.GeneratedMessage {
  factory TerminalSetTagsCommand({
    TerminalRef? terminal,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? tags,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (tags != null) result.tags.addEntries(tags);
    return result;
  }

  TerminalSetTagsCommand._();

  factory TerminalSetTagsCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalSetTagsCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalSetTagsCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'tags',
        entryClassName: 'TerminalSetTagsCommand.TagsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('anytty.api.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalSetTagsCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalSetTagsCommand copyWith(
          void Function(TerminalSetTagsCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalSetTagsCommand))
          as TerminalSetTagsCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalSetTagsCommand create() => TerminalSetTagsCommand._();
  @$core.override
  TerminalSetTagsCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalSetTagsCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalSetTagsCommand>(create);
  static TerminalSetTagsCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.String> get tags => $_getMap(1);
}

class TerminalAttachCommand extends $pb.GeneratedMessage {
  factory TerminalAttachCommand({
    TerminalRef? terminal,
    AttachmentMode? mode,
    ResizePolicy? resizePolicy,
    $core.String? surfaceId,
    $core.String? viewId,
    $0.OperationStamp? operation,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (mode != null) result.mode = mode;
    if (resizePolicy != null) result.resizePolicy = resizePolicy;
    if (surfaceId != null) result.surfaceId = surfaceId;
    if (viewId != null) result.viewId = viewId;
    if (operation != null) result.operation = operation;
    return result;
  }

  TerminalAttachCommand._();

  factory TerminalAttachCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalAttachCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalAttachCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..aE<AttachmentMode>(3, _omitFieldNames ? '' : 'mode',
        enumValues: AttachmentMode.values)
    ..aE<ResizePolicy>(4, _omitFieldNames ? '' : 'resizePolicy',
        enumValues: ResizePolicy.values)
    ..aOS(5, _omitFieldNames ? '' : 'surfaceId')
    ..aOS(6, _omitFieldNames ? '' : 'viewId')
    ..aOM<$0.OperationStamp>(7, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalAttachCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalAttachCommand copyWith(
          void Function(TerminalAttachCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalAttachCommand))
          as TerminalAttachCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalAttachCommand create() => TerminalAttachCommand._();
  @$core.override
  TerminalAttachCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalAttachCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalAttachCommand>(create);
  static TerminalAttachCommand? _defaultInstance;

  @$pb.TagNumber(2)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal(TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  AttachmentMode get mode => $_getN(1);
  @$pb.TagNumber(3)
  set mode(AttachmentMode value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasMode() => $_has(1);
  @$pb.TagNumber(3)
  void clearMode() => $_clearField(3);

  @$pb.TagNumber(4)
  ResizePolicy get resizePolicy => $_getN(2);
  @$pb.TagNumber(4)
  set resizePolicy(ResizePolicy value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasResizePolicy() => $_has(2);
  @$pb.TagNumber(4)
  void clearResizePolicy() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get surfaceId => $_getSZ(3);
  @$pb.TagNumber(5)
  set surfaceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(5)
  $core.bool hasSurfaceId() => $_has(3);
  @$pb.TagNumber(5)
  void clearSurfaceId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get viewId => $_getSZ(4);
  @$pb.TagNumber(6)
  set viewId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(6)
  $core.bool hasViewId() => $_has(4);
  @$pb.TagNumber(6)
  void clearViewId() => $_clearField(6);

  @$pb.TagNumber(7)
  $0.OperationStamp get operation => $_getN(5);
  @$pb.TagNumber(7)
  set operation($0.OperationStamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasOperation() => $_has(5);
  @$pb.TagNumber(7)
  void clearOperation() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.OperationStamp ensureOperation() => $_ensure(5);
}

class TerminalDetachCommand extends $pb.GeneratedMessage {
  factory TerminalDetachCommand({
    $0.ResourceHandle? attachment,
    $0.OperationStamp? operation,
  }) {
    final result = create();
    if (attachment != null) result.attachment = attachment;
    if (operation != null) result.operation = operation;
    return result;
  }

  TerminalDetachCommand._();

  factory TerminalDetachCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalDetachCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalDetachCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(2, _omitFieldNames ? '' : 'attachment',
        subBuilder: $0.ResourceHandle.create)
    ..aOM<$0.OperationStamp>(3, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDetachCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDetachCommand copyWith(
          void Function(TerminalDetachCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalDetachCommand))
          as TerminalDetachCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalDetachCommand create() => TerminalDetachCommand._();
  @$core.override
  TerminalDetachCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalDetachCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalDetachCommand>(create);
  static TerminalDetachCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ResourceHandle get attachment => $_getN(0);
  @$pb.TagNumber(2)
  set attachment($0.ResourceHandle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasAttachment() => $_has(0);
  @$pb.TagNumber(2)
  void clearAttachment() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ResourceHandle ensureAttachment() => $_ensure(0);

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
}

class TerminalInputCommand extends $pb.GeneratedMessage {
  factory TerminalInputCommand({
    $0.ResourceHandle? attachment,
    $0.OperationStamp? operation,
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (attachment != null) result.attachment = attachment;
    if (operation != null) result.operation = operation;
    if (data != null) result.data = data;
    return result;
  }

  TerminalInputCommand._();

  factory TerminalInputCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalInputCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalInputCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(2, _omitFieldNames ? '' : 'attachment',
        subBuilder: $0.ResourceHandle.create)
    ..aOM<$0.OperationStamp>(3, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalInputCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalInputCommand copyWith(void Function(TerminalInputCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalInputCommand))
          as TerminalInputCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalInputCommand create() => TerminalInputCommand._();
  @$core.override
  TerminalInputCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalInputCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalInputCommand>(create);
  static TerminalInputCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ResourceHandle get attachment => $_getN(0);
  @$pb.TagNumber(2)
  set attachment($0.ResourceHandle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasAttachment() => $_has(0);
  @$pb.TagNumber(2)
  void clearAttachment() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ResourceHandle ensureAttachment() => $_ensure(0);

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
  $core.List<$core.int> get data => $_getN(2);
  @$pb.TagNumber(4)
  set data($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(4)
  $core.bool hasData() => $_has(2);
  @$pb.TagNumber(4)
  void clearData() => $_clearField(4);
}

class TerminalResizeCommand extends $pb.GeneratedMessage {
  factory TerminalResizeCommand({
    $0.ResourceHandle? attachment,
    $0.OperationStamp? operation,
    TerminalSize? size,
    ResizePolicy? resizePolicy,
    $core.bool? takeOwnership,
    $fixnum.Int64? expectedOwnerEpoch,
  }) {
    final result = create();
    if (attachment != null) result.attachment = attachment;
    if (operation != null) result.operation = operation;
    if (size != null) result.size = size;
    if (resizePolicy != null) result.resizePolicy = resizePolicy;
    if (takeOwnership != null) result.takeOwnership = takeOwnership;
    if (expectedOwnerEpoch != null)
      result.expectedOwnerEpoch = expectedOwnerEpoch;
    return result;
  }

  TerminalResizeCommand._();

  factory TerminalResizeCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalResizeCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalResizeCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(2, _omitFieldNames ? '' : 'attachment',
        subBuilder: $0.ResourceHandle.create)
    ..aOM<$0.OperationStamp>(3, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..aOM<TerminalSize>(4, _omitFieldNames ? '' : 'size',
        subBuilder: TerminalSize.create)
    ..aE<ResizePolicy>(5, _omitFieldNames ? '' : 'resizePolicy',
        enumValues: ResizePolicy.values)
    ..aOB(6, _omitFieldNames ? '' : 'takeOwnership')
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'expectedOwnerEpoch', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeCommand copyWith(
          void Function(TerminalResizeCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalResizeCommand))
          as TerminalResizeCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalResizeCommand create() => TerminalResizeCommand._();
  @$core.override
  TerminalResizeCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalResizeCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalResizeCommand>(create);
  static TerminalResizeCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ResourceHandle get attachment => $_getN(0);
  @$pb.TagNumber(2)
  set attachment($0.ResourceHandle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasAttachment() => $_has(0);
  @$pb.TagNumber(2)
  void clearAttachment() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ResourceHandle ensureAttachment() => $_ensure(0);

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
  TerminalSize get size => $_getN(2);
  @$pb.TagNumber(4)
  set size(TerminalSize value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSize() => $_has(2);
  @$pb.TagNumber(4)
  void clearSize() => $_clearField(4);
  @$pb.TagNumber(4)
  TerminalSize ensureSize() => $_ensure(2);

  @$pb.TagNumber(5)
  ResizePolicy get resizePolicy => $_getN(3);
  @$pb.TagNumber(5)
  set resizePolicy(ResizePolicy value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasResizePolicy() => $_has(3);
  @$pb.TagNumber(5)
  void clearResizePolicy() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get takeOwnership => $_getBF(4);
  @$pb.TagNumber(6)
  set takeOwnership($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(6)
  $core.bool hasTakeOwnership() => $_has(4);
  @$pb.TagNumber(6)
  void clearTakeOwnership() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get expectedOwnerEpoch => $_getI64(5);
  @$pb.TagNumber(7)
  set expectedOwnerEpoch($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(7)
  $core.bool hasExpectedOwnerEpoch() => $_has(5);
  @$pb.TagNumber(7)
  void clearExpectedOwnerEpoch() => $_clearField(7);
}

class TerminalResizeLockCommand extends $pb.GeneratedMessage {
  factory TerminalResizeLockCommand({
    $0.ResourceHandle? attachment,
    $0.OperationStamp? operation,
    $core.bool? locked,
  }) {
    final result = create();
    if (attachment != null) result.attachment = attachment;
    if (operation != null) result.operation = operation;
    if (locked != null) result.locked = locked;
    return result;
  }

  TerminalResizeLockCommand._();

  factory TerminalResizeLockCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalResizeLockCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalResizeLockCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(2, _omitFieldNames ? '' : 'attachment',
        subBuilder: $0.ResourceHandle.create)
    ..aOM<$0.OperationStamp>(3, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..aOB(4, _omitFieldNames ? '' : 'locked')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeLockCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeLockCommand copyWith(
          void Function(TerminalResizeLockCommand) updates) =>
      super.copyWith((message) => updates(message as TerminalResizeLockCommand))
          as TerminalResizeLockCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalResizeLockCommand create() => TerminalResizeLockCommand._();
  @$core.override
  TerminalResizeLockCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalResizeLockCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalResizeLockCommand>(create);
  static TerminalResizeLockCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ResourceHandle get attachment => $_getN(0);
  @$pb.TagNumber(2)
  set attachment($0.ResourceHandle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasAttachment() => $_has(0);
  @$pb.TagNumber(2)
  void clearAttachment() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ResourceHandle ensureAttachment() => $_ensure(0);

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
  $core.bool get locked => $_getBF(2);
  @$pb.TagNumber(4)
  set locked($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(4)
  $core.bool hasLocked() => $_has(2);
  @$pb.TagNumber(4)
  void clearLocked() => $_clearField(4);
}

class PathListDirectoriesCommand extends $pb.GeneratedMessage {
  factory PathListDirectoriesCommand({
    $core.String? prefix,
    $core.int? limit,
  }) {
    final result = create();
    if (prefix != null) result.prefix = prefix;
    if (limit != null) result.limit = limit;
    return result;
  }

  PathListDirectoriesCommand._();

  factory PathListDirectoriesCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PathListDirectoriesCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PathListDirectoriesCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(2, _omitFieldNames ? '' : 'prefix')
    ..aI(3, _omitFieldNames ? '' : 'limit')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PathListDirectoriesCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PathListDirectoriesCommand copyWith(
          void Function(PathListDirectoriesCommand) updates) =>
      super.copyWith(
              (message) => updates(message as PathListDirectoriesCommand))
          as PathListDirectoriesCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PathListDirectoriesCommand create() => PathListDirectoriesCommand._();
  @$core.override
  PathListDirectoriesCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PathListDirectoriesCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PathListDirectoriesCommand>(create);
  static PathListDirectoriesCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get prefix => $_getSZ(0);
  @$pb.TagNumber(2)
  set prefix($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasPrefix() => $_has(0);
  @$pb.TagNumber(2)
  void clearPrefix() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get limit => $_getIZ(1);
  @$pb.TagNumber(3)
  set limit($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(3)
  $core.bool hasLimit() => $_has(1);
  @$pb.TagNumber(3)
  void clearLimit() => $_clearField(3);
}

class TerminalCreateResult extends $pb.GeneratedMessage {
  factory TerminalCreateResult({
    TerminalInfo? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalCreateResult._();

  factory TerminalCreateResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalCreateResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalCreateResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalInfo>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalInfo.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCreateResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCreateResult copyWith(void Function(TerminalCreateResult) updates) =>
      super.copyWith((message) => updates(message as TerminalCreateResult))
          as TerminalCreateResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalCreateResult create() => TerminalCreateResult._();
  @$core.override
  TerminalCreateResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalCreateResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalCreateResult>(create);
  static TerminalCreateResult? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalInfo get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal(TerminalInfo value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalInfo ensureTerminal() => $_ensure(0);
}

class TerminalListResult extends $pb.GeneratedMessage {
  factory TerminalListResult({
    $core.Iterable<TerminalInfo>? terminals,
  }) {
    final result = create();
    if (terminals != null) result.terminals.addAll(terminals);
    return result;
  }

  TerminalListResult._();

  factory TerminalListResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalListResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalListResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pPM<TerminalInfo>(1, _omitFieldNames ? '' : 'terminals',
        subBuilder: TerminalInfo.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalListResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalListResult copyWith(void Function(TerminalListResult) updates) =>
      super.copyWith((message) => updates(message as TerminalListResult))
          as TerminalListResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalListResult create() => TerminalListResult._();
  @$core.override
  TerminalListResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalListResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalListResult>(create);
  static TerminalListResult? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<TerminalInfo> get terminals => $_getList(0);
}

class TerminalGetResult extends $pb.GeneratedMessage {
  factory TerminalGetResult({
    TerminalInfo? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  TerminalGetResult._();

  factory TerminalGetResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalGetResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalGetResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalInfo>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalInfo.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalGetResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalGetResult copyWith(void Function(TerminalGetResult) updates) =>
      super.copyWith((message) => updates(message as TerminalGetResult))
          as TerminalGetResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalGetResult create() => TerminalGetResult._();
  @$core.override
  TerminalGetResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalGetResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalGetResult>(create);
  static TerminalGetResult? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalInfo get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal(TerminalInfo value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalInfo ensureTerminal() => $_ensure(0);
}

class TerminalDefaultsResult extends $pb.GeneratedMessage {
  factory TerminalDefaultsResult({
    TerminalDefaults? defaults,
  }) {
    final result = create();
    if (defaults != null) result.defaults = defaults;
    return result;
  }

  TerminalDefaultsResult._();

  factory TerminalDefaultsResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalDefaultsResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalDefaultsResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalDefaults>(1, _omitFieldNames ? '' : 'defaults',
        subBuilder: TerminalDefaults.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDefaultsResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalDefaultsResult copyWith(
          void Function(TerminalDefaultsResult) updates) =>
      super.copyWith((message) => updates(message as TerminalDefaultsResult))
          as TerminalDefaultsResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalDefaultsResult create() => TerminalDefaultsResult._();
  @$core.override
  TerminalDefaultsResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalDefaultsResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalDefaultsResult>(create);
  static TerminalDefaultsResult? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalDefaults get defaults => $_getN(0);
  @$pb.TagNumber(1)
  set defaults(TerminalDefaults value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDefaults() => $_has(0);
  @$pb.TagNumber(1)
  void clearDefaults() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalDefaults ensureDefaults() => $_ensure(0);
}

class TerminalAttachResult extends $pb.GeneratedMessage {
  factory TerminalAttachResult({
    AttachmentHandle? attachment,
    AttachmentMode? mode,
    ResizePolicy? resizePolicy,
    TerminalSize? size,
    ResizeControl? resizeControl,
  }) {
    final result = create();
    if (attachment != null) result.attachment = attachment;
    if (mode != null) result.mode = mode;
    if (resizePolicy != null) result.resizePolicy = resizePolicy;
    if (size != null) result.size = size;
    if (resizeControl != null) result.resizeControl = resizeControl;
    return result;
  }

  TerminalAttachResult._();

  factory TerminalAttachResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalAttachResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalAttachResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<AttachmentHandle>(1, _omitFieldNames ? '' : 'attachment',
        subBuilder: AttachmentHandle.create)
    ..aE<AttachmentMode>(2, _omitFieldNames ? '' : 'mode',
        enumValues: AttachmentMode.values)
    ..aE<ResizePolicy>(3, _omitFieldNames ? '' : 'resizePolicy',
        enumValues: ResizePolicy.values)
    ..aOM<TerminalSize>(4, _omitFieldNames ? '' : 'size',
        subBuilder: TerminalSize.create)
    ..aOM<ResizeControl>(5, _omitFieldNames ? '' : 'resizeControl',
        subBuilder: ResizeControl.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalAttachResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalAttachResult copyWith(void Function(TerminalAttachResult) updates) =>
      super.copyWith((message) => updates(message as TerminalAttachResult))
          as TerminalAttachResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalAttachResult create() => TerminalAttachResult._();
  @$core.override
  TerminalAttachResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalAttachResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalAttachResult>(create);
  static TerminalAttachResult? _defaultInstance;

  @$pb.TagNumber(1)
  AttachmentHandle get attachment => $_getN(0);
  @$pb.TagNumber(1)
  set attachment(AttachmentHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAttachment() => $_has(0);
  @$pb.TagNumber(1)
  void clearAttachment() => $_clearField(1);
  @$pb.TagNumber(1)
  AttachmentHandle ensureAttachment() => $_ensure(0);

  @$pb.TagNumber(2)
  AttachmentMode get mode => $_getN(1);
  @$pb.TagNumber(2)
  set mode(AttachmentMode value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasMode() => $_has(1);
  @$pb.TagNumber(2)
  void clearMode() => $_clearField(2);

  @$pb.TagNumber(3)
  ResizePolicy get resizePolicy => $_getN(2);
  @$pb.TagNumber(3)
  set resizePolicy(ResizePolicy value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasResizePolicy() => $_has(2);
  @$pb.TagNumber(3)
  void clearResizePolicy() => $_clearField(3);

  @$pb.TagNumber(4)
  TerminalSize get size => $_getN(3);
  @$pb.TagNumber(4)
  set size(TerminalSize value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSize() => $_has(3);
  @$pb.TagNumber(4)
  void clearSize() => $_clearField(4);
  @$pb.TagNumber(4)
  TerminalSize ensureSize() => $_ensure(3);

  @$pb.TagNumber(5)
  ResizeControl get resizeControl => $_getN(4);
  @$pb.TagNumber(5)
  set resizeControl(ResizeControl value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasResizeControl() => $_has(4);
  @$pb.TagNumber(5)
  void clearResizeControl() => $_clearField(5);
  @$pb.TagNumber(5)
  ResizeControl ensureResizeControl() => $_ensure(4);
}

class TerminalResizeResult extends $pb.GeneratedMessage {
  factory TerminalResizeResult({
    TerminalSize? size,
    $core.bool? resized,
    ResizeControl? resizeControl,
  }) {
    final result = create();
    if (size != null) result.size = size;
    if (resized != null) result.resized = resized;
    if (resizeControl != null) result.resizeControl = resizeControl;
    return result;
  }

  TerminalResizeResult._();

  factory TerminalResizeResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalResizeResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalResizeResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalSize>(1, _omitFieldNames ? '' : 'size',
        subBuilder: TerminalSize.create)
    ..aOB(2, _omitFieldNames ? '' : 'resized')
    ..aOM<ResizeControl>(3, _omitFieldNames ? '' : 'resizeControl',
        subBuilder: ResizeControl.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeResult copyWith(void Function(TerminalResizeResult) updates) =>
      super.copyWith((message) => updates(message as TerminalResizeResult))
          as TerminalResizeResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalResizeResult create() => TerminalResizeResult._();
  @$core.override
  TerminalResizeResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalResizeResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalResizeResult>(create);
  static TerminalResizeResult? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalSize get size => $_getN(0);
  @$pb.TagNumber(1)
  set size(TerminalSize value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSize() => $_has(0);
  @$pb.TagNumber(1)
  void clearSize() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalSize ensureSize() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get resized => $_getBF(1);
  @$pb.TagNumber(2)
  set resized($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasResized() => $_has(1);
  @$pb.TagNumber(2)
  void clearResized() => $_clearField(2);

  @$pb.TagNumber(3)
  ResizeControl get resizeControl => $_getN(2);
  @$pb.TagNumber(3)
  set resizeControl(ResizeControl value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasResizeControl() => $_has(2);
  @$pb.TagNumber(3)
  void clearResizeControl() => $_clearField(3);
  @$pb.TagNumber(3)
  ResizeControl ensureResizeControl() => $_ensure(2);
}

class PathListDirectoriesResult extends $pb.GeneratedMessage {
  factory PathListDirectoriesResult({
    $core.String? basePath,
    $core.Iterable<PathDirectoryEntry>? entries,
    $core.bool? missing,
    $core.bool? truncated,
  }) {
    final result = create();
    if (basePath != null) result.basePath = basePath;
    if (entries != null) result.entries.addAll(entries);
    if (missing != null) result.missing = missing;
    if (truncated != null) result.truncated = truncated;
    return result;
  }

  PathListDirectoriesResult._();

  factory PathListDirectoriesResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PathListDirectoriesResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PathListDirectoriesResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'basePath')
    ..pPM<PathDirectoryEntry>(2, _omitFieldNames ? '' : 'entries',
        subBuilder: PathDirectoryEntry.create)
    ..aOB(3, _omitFieldNames ? '' : 'missing')
    ..aOB(4, _omitFieldNames ? '' : 'truncated')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PathListDirectoriesResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PathListDirectoriesResult copyWith(
          void Function(PathListDirectoriesResult) updates) =>
      super.copyWith((message) => updates(message as PathListDirectoriesResult))
          as PathListDirectoriesResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PathListDirectoriesResult create() => PathListDirectoriesResult._();
  @$core.override
  PathListDirectoriesResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PathListDirectoriesResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PathListDirectoriesResult>(create);
  static PathListDirectoriesResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get basePath => $_getSZ(0);
  @$pb.TagNumber(1)
  set basePath($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBasePath() => $_has(0);
  @$pb.TagNumber(1)
  void clearBasePath() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<PathDirectoryEntry> get entries => $_getList(1);

  @$pb.TagNumber(3)
  $core.bool get missing => $_getBF(2);
  @$pb.TagNumber(3)
  set missing($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMissing() => $_has(2);
  @$pb.TagNumber(3)
  void clearMissing() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get truncated => $_getBF(3);
  @$pb.TagNumber(4)
  set truncated($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTruncated() => $_has(3);
  @$pb.TagNumber(4)
  void clearTruncated() => $_clearField(4);
}

class TerminalLifecycleEvent extends $pb.GeneratedMessage {
  factory TerminalLifecycleEvent({
    TerminalInfo? terminal,
    $core.bool? attachmentProjection,
    ResizeControl? resizeControl,
    $fixnum.Int64? resizeEpoch,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (attachmentProjection != null)
      result.attachmentProjection = attachmentProjection;
    if (resizeControl != null) result.resizeControl = resizeControl;
    if (resizeEpoch != null) result.resizeEpoch = resizeEpoch;
    return result;
  }

  TerminalLifecycleEvent._();

  factory TerminalLifecycleEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalLifecycleEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalLifecycleEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalInfo>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalInfo.create)
    ..aOB(2, _omitFieldNames ? '' : 'attachmentProjection')
    ..aOM<ResizeControl>(3, _omitFieldNames ? '' : 'resizeControl',
        subBuilder: ResizeControl.create)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'resizeEpoch', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalLifecycleEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalLifecycleEvent copyWith(
          void Function(TerminalLifecycleEvent) updates) =>
      super.copyWith((message) => updates(message as TerminalLifecycleEvent))
          as TerminalLifecycleEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalLifecycleEvent create() => TerminalLifecycleEvent._();
  @$core.override
  TerminalLifecycleEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalLifecycleEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalLifecycleEvent>(create);
  static TerminalLifecycleEvent? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalInfo get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal(TerminalInfo value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalInfo ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get attachmentProjection => $_getBF(1);
  @$pb.TagNumber(2)
  set attachmentProjection($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAttachmentProjection() => $_has(1);
  @$pb.TagNumber(2)
  void clearAttachmentProjection() => $_clearField(2);

  @$pb.TagNumber(3)
  ResizeControl get resizeControl => $_getN(2);
  @$pb.TagNumber(3)
  set resizeControl(ResizeControl value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasResizeControl() => $_has(2);
  @$pb.TagNumber(3)
  void clearResizeControl() => $_clearField(3);
  @$pb.TagNumber(3)
  ResizeControl ensureResizeControl() => $_ensure(2);

  /// resize_epoch versions the complete owner projection, including no-owner state.
  @$pb.TagNumber(4)
  $fixnum.Int64 get resizeEpoch => $_getI64(3);
  @$pb.TagNumber(4)
  set resizeEpoch($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasResizeEpoch() => $_has(3);
  @$pb.TagNumber(4)
  void clearResizeEpoch() => $_clearField(4);
}

class TerminalResizeControlEvent extends $pb.GeneratedMessage {
  factory TerminalResizeControlEvent({
    TerminalRef? terminal,
    ResizeControl? resizeControl,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (resizeControl != null) result.resizeControl = resizeControl;
    return result;
  }

  TerminalResizeControlEvent._();

  factory TerminalResizeControlEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalResizeControlEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalResizeControlEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<TerminalRef>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: TerminalRef.create)
    ..aOM<ResizeControl>(2, _omitFieldNames ? '' : 'resizeControl',
        subBuilder: ResizeControl.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeControlEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalResizeControlEvent copyWith(
          void Function(TerminalResizeControlEvent) updates) =>
      super.copyWith(
              (message) => updates(message as TerminalResizeControlEvent))
          as TerminalResizeControlEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalResizeControlEvent create() => TerminalResizeControlEvent._();
  @$core.override
  TerminalResizeControlEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalResizeControlEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalResizeControlEvent>(create);
  static TerminalResizeControlEvent? _defaultInstance;

  @$pb.TagNumber(1)
  TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal(TerminalRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(2)
  ResizeControl get resizeControl => $_getN(1);
  @$pb.TagNumber(2)
  set resizeControl(ResizeControl value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasResizeControl() => $_has(1);
  @$pb.TagNumber(2)
  void clearResizeControl() => $_clearField(2);
  @$pb.TagNumber(2)
  ResizeControl ensureResizeControl() => $_ensure(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
