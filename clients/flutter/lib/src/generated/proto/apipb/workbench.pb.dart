// This is a generated file - do not edit.
//
// Generated from apipb/workbench.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'workbench.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'workbench.pbenum.dart';

class WorkbenchValue extends $pb.GeneratedMessage {
  factory WorkbenchValue({
    $core.int? schemaVersion,
    $core.String? activeWorkspaceId,
    $core.Iterable<WorkbenchWorkspace>? workspaces,
  }) {
    final result = create();
    if (schemaVersion != null) result.schemaVersion = schemaVersion;
    if (activeWorkspaceId != null) result.activeWorkspaceId = activeWorkspaceId;
    if (workspaces != null) result.workspaces.addAll(workspaces);
    return result;
  }

  WorkbenchValue._();

  factory WorkbenchValue.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WorkbenchValue.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkbenchValue',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'schemaVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'activeWorkspaceId')
    ..pPM<WorkbenchWorkspace>(3, _omitFieldNames ? '' : 'workspaces',
        subBuilder: WorkbenchWorkspace.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchValue clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchValue copyWith(void Function(WorkbenchValue) updates) =>
      super.copyWith((message) => updates(message as WorkbenchValue))
          as WorkbenchValue;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkbenchValue create() => WorkbenchValue._();
  @$core.override
  WorkbenchValue createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WorkbenchValue getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WorkbenchValue>(create);
  static WorkbenchValue? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get schemaVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set schemaVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchemaVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchemaVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get activeWorkspaceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set activeWorkspaceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasActiveWorkspaceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearActiveWorkspaceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<WorkbenchWorkspace> get workspaces => $_getList(2);
}

class WorkbenchWorkspace extends $pb.GeneratedMessage {
  factory WorkbenchWorkspace({
    $core.String? id,
    $core.String? name,
    $core.String? activeTabId,
    $core.Iterable<WorkbenchTab>? tabs,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (activeTabId != null) result.activeTabId = activeTabId;
    if (tabs != null) result.tabs.addAll(tabs);
    return result;
  }

  WorkbenchWorkspace._();

  factory WorkbenchWorkspace.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WorkbenchWorkspace.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkbenchWorkspace',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'activeTabId')
    ..pPM<WorkbenchTab>(4, _omitFieldNames ? '' : 'tabs',
        subBuilder: WorkbenchTab.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchWorkspace clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchWorkspace copyWith(void Function(WorkbenchWorkspace) updates) =>
      super.copyWith((message) => updates(message as WorkbenchWorkspace))
          as WorkbenchWorkspace;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkbenchWorkspace create() => WorkbenchWorkspace._();
  @$core.override
  WorkbenchWorkspace createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WorkbenchWorkspace getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WorkbenchWorkspace>(create);
  static WorkbenchWorkspace? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get activeTabId => $_getSZ(2);
  @$pb.TagNumber(3)
  set activeTabId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasActiveTabId() => $_has(2);
  @$pb.TagNumber(3)
  void clearActiveTabId() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<WorkbenchTab> get tabs => $_getList(3);
}

class WorkbenchTab extends $pb.GeneratedMessage {
  factory WorkbenchTab({
    $core.String? id,
    $core.String? title,
    $core.String? activePaneId,
    $core.Iterable<WorkbenchPane>? panes,
    WorkbenchSplitNode? rootSplit,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (title != null) result.title = title;
    if (activePaneId != null) result.activePaneId = activePaneId;
    if (panes != null) result.panes.addAll(panes);
    if (rootSplit != null) result.rootSplit = rootSplit;
    return result;
  }

  WorkbenchTab._();

  factory WorkbenchTab.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WorkbenchTab.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkbenchTab',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'title')
    ..aOS(3, _omitFieldNames ? '' : 'activePaneId')
    ..pPM<WorkbenchPane>(4, _omitFieldNames ? '' : 'panes',
        subBuilder: WorkbenchPane.create)
    ..aOM<WorkbenchSplitNode>(5, _omitFieldNames ? '' : 'rootSplit',
        subBuilder: WorkbenchSplitNode.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchTab clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchTab copyWith(void Function(WorkbenchTab) updates) =>
      super.copyWith((message) => updates(message as WorkbenchTab))
          as WorkbenchTab;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkbenchTab create() => WorkbenchTab._();
  @$core.override
  WorkbenchTab createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WorkbenchTab getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WorkbenchTab>(create);
  static WorkbenchTab? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get title => $_getSZ(1);
  @$pb.TagNumber(2)
  set title($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTitle() => $_has(1);
  @$pb.TagNumber(2)
  void clearTitle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get activePaneId => $_getSZ(2);
  @$pb.TagNumber(3)
  set activePaneId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasActivePaneId() => $_has(2);
  @$pb.TagNumber(3)
  void clearActivePaneId() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<WorkbenchPane> get panes => $_getList(3);

  @$pb.TagNumber(5)
  WorkbenchSplitNode get rootSplit => $_getN(4);
  @$pb.TagNumber(5)
  set rootSplit(WorkbenchSplitNode value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasRootSplit() => $_has(4);
  @$pb.TagNumber(5)
  void clearRootSplit() => $_clearField(5);
  @$pb.TagNumber(5)
  WorkbenchSplitNode ensureRootSplit() => $_ensure(4);
}

class WorkbenchPane extends $pb.GeneratedMessage {
  factory WorkbenchPane({
    $core.String? id,
    $core.String? title,
    $core.String? terminalId,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (title != null) result.title = title;
    if (terminalId != null) result.terminalId = terminalId;
    return result;
  }

  WorkbenchPane._();

  factory WorkbenchPane.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WorkbenchPane.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkbenchPane',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'title')
    ..aOS(3, _omitFieldNames ? '' : 'terminalId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchPane clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchPane copyWith(void Function(WorkbenchPane) updates) =>
      super.copyWith((message) => updates(message as WorkbenchPane))
          as WorkbenchPane;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkbenchPane create() => WorkbenchPane._();
  @$core.override
  WorkbenchPane createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WorkbenchPane getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WorkbenchPane>(create);
  static WorkbenchPane? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get title => $_getSZ(1);
  @$pb.TagNumber(2)
  set title($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTitle() => $_has(1);
  @$pb.TagNumber(2)
  void clearTitle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get terminalId => $_getSZ(2);
  @$pb.TagNumber(3)
  set terminalId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTerminalId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTerminalId() => $_clearField(3);
}

class WorkbenchSplitNode extends $pb.GeneratedMessage {
  factory WorkbenchSplitNode({
    $core.String? paneId,
    WorkbenchSplitDirection? direction,
    $core.Iterable<WorkbenchSplitNode>? children,
    $core.double? ratio,
    $core.int? biasCells,
    $core.String? fixedPaneId,
    $core.int? fixedCols,
    $core.int? fixedRows,
  }) {
    final result = create();
    if (paneId != null) result.paneId = paneId;
    if (direction != null) result.direction = direction;
    if (children != null) result.children.addAll(children);
    if (ratio != null) result.ratio = ratio;
    if (biasCells != null) result.biasCells = biasCells;
    if (fixedPaneId != null) result.fixedPaneId = fixedPaneId;
    if (fixedCols != null) result.fixedCols = fixedCols;
    if (fixedRows != null) result.fixedRows = fixedRows;
    return result;
  }

  WorkbenchSplitNode._();

  factory WorkbenchSplitNode.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WorkbenchSplitNode.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkbenchSplitNode',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'paneId')
    ..aE<WorkbenchSplitDirection>(2, _omitFieldNames ? '' : 'direction',
        enumValues: WorkbenchSplitDirection.values)
    ..pPM<WorkbenchSplitNode>(3, _omitFieldNames ? '' : 'children',
        subBuilder: WorkbenchSplitNode.create)
    ..aD(4, _omitFieldNames ? '' : 'ratio')
    ..aI(5, _omitFieldNames ? '' : 'biasCells')
    ..aOS(6, _omitFieldNames ? '' : 'fixedPaneId')
    ..aI(7, _omitFieldNames ? '' : 'fixedCols')
    ..aI(8, _omitFieldNames ? '' : 'fixedRows')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchSplitNode clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkbenchSplitNode copyWith(void Function(WorkbenchSplitNode) updates) =>
      super.copyWith((message) => updates(message as WorkbenchSplitNode))
          as WorkbenchSplitNode;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkbenchSplitNode create() => WorkbenchSplitNode._();
  @$core.override
  WorkbenchSplitNode createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WorkbenchSplitNode getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WorkbenchSplitNode>(create);
  static WorkbenchSplitNode? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get paneId => $_getSZ(0);
  @$pb.TagNumber(1)
  set paneId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPaneId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPaneId() => $_clearField(1);

  @$pb.TagNumber(2)
  WorkbenchSplitDirection get direction => $_getN(1);
  @$pb.TagNumber(2)
  set direction(WorkbenchSplitDirection value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasDirection() => $_has(1);
  @$pb.TagNumber(2)
  void clearDirection() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<WorkbenchSplitNode> get children => $_getList(2);

  @$pb.TagNumber(4)
  $core.double get ratio => $_getN(3);
  @$pb.TagNumber(4)
  set ratio($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRatio() => $_has(3);
  @$pb.TagNumber(4)
  void clearRatio() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get biasCells => $_getIZ(4);
  @$pb.TagNumber(5)
  set biasCells($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBiasCells() => $_has(4);
  @$pb.TagNumber(5)
  void clearBiasCells() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get fixedPaneId => $_getSZ(5);
  @$pb.TagNumber(6)
  set fixedPaneId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFixedPaneId() => $_has(5);
  @$pb.TagNumber(6)
  void clearFixedPaneId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get fixedCols => $_getIZ(6);
  @$pb.TagNumber(7)
  set fixedCols($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasFixedCols() => $_has(6);
  @$pb.TagNumber(7)
  void clearFixedCols() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get fixedRows => $_getIZ(7);
  @$pb.TagNumber(8)
  set fixedRows($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasFixedRows() => $_has(7);
  @$pb.TagNumber(8)
  void clearFixedRows() => $_clearField(8);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
