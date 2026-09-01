// This is a generated file - do not edit.
//
// Generated from apipb/history.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'history.pbenum.dart';
import 'terminal.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'history.pbenum.dart';

class CellStyle extends $pb.GeneratedMessage {
  factory CellStyle({
    $core.String? foreground,
    $core.String? background,
    $core.bool? bold,
    $core.bool? italic,
    $core.bool? underline,
    $core.bool? blink,
    $core.bool? reverse,
    $core.bool? strikethrough,
  }) {
    final result = create();
    if (foreground != null) result.foreground = foreground;
    if (background != null) result.background = background;
    if (bold != null) result.bold = bold;
    if (italic != null) result.italic = italic;
    if (underline != null) result.underline = underline;
    if (blink != null) result.blink = blink;
    if (reverse != null) result.reverse = reverse;
    if (strikethrough != null) result.strikethrough = strikethrough;
    return result;
  }

  CellStyle._();

  factory CellStyle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CellStyle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CellStyle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'foreground')
    ..aOS(2, _omitFieldNames ? '' : 'background')
    ..aOB(3, _omitFieldNames ? '' : 'bold')
    ..aOB(4, _omitFieldNames ? '' : 'italic')
    ..aOB(5, _omitFieldNames ? '' : 'underline')
    ..aOB(6, _omitFieldNames ? '' : 'blink')
    ..aOB(7, _omitFieldNames ? '' : 'reverse')
    ..aOB(8, _omitFieldNames ? '' : 'strikethrough')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CellStyle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CellStyle copyWith(void Function(CellStyle) updates) =>
      super.copyWith((message) => updates(message as CellStyle)) as CellStyle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CellStyle create() => CellStyle._();
  @$core.override
  CellStyle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CellStyle getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CellStyle>(create);
  static CellStyle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get foreground => $_getSZ(0);
  @$pb.TagNumber(1)
  set foreground($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasForeground() => $_has(0);
  @$pb.TagNumber(1)
  void clearForeground() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get background => $_getSZ(1);
  @$pb.TagNumber(2)
  set background($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBackground() => $_has(1);
  @$pb.TagNumber(2)
  void clearBackground() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get bold => $_getBF(2);
  @$pb.TagNumber(3)
  set bold($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBold() => $_has(2);
  @$pb.TagNumber(3)
  void clearBold() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get italic => $_getBF(3);
  @$pb.TagNumber(4)
  set italic($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasItalic() => $_has(3);
  @$pb.TagNumber(4)
  void clearItalic() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get underline => $_getBF(4);
  @$pb.TagNumber(5)
  set underline($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasUnderline() => $_has(4);
  @$pb.TagNumber(5)
  void clearUnderline() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get blink => $_getBF(5);
  @$pb.TagNumber(6)
  set blink($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasBlink() => $_has(5);
  @$pb.TagNumber(6)
  void clearBlink() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get reverse => $_getBF(6);
  @$pb.TagNumber(7)
  set reverse($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasReverse() => $_has(6);
  @$pb.TagNumber(7)
  void clearReverse() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get strikethrough => $_getBF(7);
  @$pb.TagNumber(8)
  set strikethrough($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasStrikethrough() => $_has(7);
  @$pb.TagNumber(8)
  void clearStrikethrough() => $_clearField(8);
}

class ScreenCell extends $pb.GeneratedMessage {
  factory ScreenCell({
    $core.String? content,
    $core.int? width,
    CellStyle? style,
    $core.String? linkUrl,
    $core.String? linkParams,
  }) {
    final result = create();
    if (content != null) result.content = content;
    if (width != null) result.width = width;
    if (style != null) result.style = style;
    if (linkUrl != null) result.linkUrl = linkUrl;
    if (linkParams != null) result.linkParams = linkParams;
    return result;
  }

  ScreenCell._();

  factory ScreenCell.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScreenCell.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScreenCell',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'content')
    ..aI(2, _omitFieldNames ? '' : 'width')
    ..aOM<CellStyle>(3, _omitFieldNames ? '' : 'style',
        subBuilder: CellStyle.create)
    ..aOS(4, _omitFieldNames ? '' : 'linkUrl')
    ..aOS(5, _omitFieldNames ? '' : 'linkParams')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenCell clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenCell copyWith(void Function(ScreenCell) updates) =>
      super.copyWith((message) => updates(message as ScreenCell)) as ScreenCell;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScreenCell create() => ScreenCell._();
  @$core.override
  ScreenCell createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScreenCell getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScreenCell>(create);
  static ScreenCell? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get content => $_getSZ(0);
  @$pb.TagNumber(1)
  set content($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasContent() => $_has(0);
  @$pb.TagNumber(1)
  void clearContent() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get width => $_getIZ(1);
  @$pb.TagNumber(2)
  set width($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWidth() => $_has(1);
  @$pb.TagNumber(2)
  void clearWidth() => $_clearField(2);

  @$pb.TagNumber(3)
  CellStyle get style => $_getN(2);
  @$pb.TagNumber(3)
  set style(CellStyle value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasStyle() => $_has(2);
  @$pb.TagNumber(3)
  void clearStyle() => $_clearField(3);
  @$pb.TagNumber(3)
  CellStyle ensureStyle() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get linkUrl => $_getSZ(3);
  @$pb.TagNumber(4)
  set linkUrl($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLinkUrl() => $_has(3);
  @$pb.TagNumber(4)
  void clearLinkUrl() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get linkParams => $_getSZ(4);
  @$pb.TagNumber(5)
  set linkParams($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasLinkParams() => $_has(4);
  @$pb.TagNumber(5)
  void clearLinkParams() => $_clearField(5);
}

class ScreenRow extends $pb.GeneratedMessage {
  factory ScreenRow({
    $core.Iterable<ScreenCell>? cells,
    CellStyle? tailFill,
    $core.bool? wrapped,
  }) {
    final result = create();
    if (cells != null) result.cells.addAll(cells);
    if (tailFill != null) result.tailFill = tailFill;
    if (wrapped != null) result.wrapped = wrapped;
    return result;
  }

  ScreenRow._();

  factory ScreenRow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScreenRow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScreenRow',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pPM<ScreenCell>(1, _omitFieldNames ? '' : 'cells',
        subBuilder: ScreenCell.create)
    ..aOM<CellStyle>(2, _omitFieldNames ? '' : 'tailFill',
        subBuilder: CellStyle.create)
    ..aOB(3, _omitFieldNames ? '' : 'wrapped')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenRow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenRow copyWith(void Function(ScreenRow) updates) =>
      super.copyWith((message) => updates(message as ScreenRow)) as ScreenRow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScreenRow create() => ScreenRow._();
  @$core.override
  ScreenRow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScreenRow getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ScreenRow>(create);
  static ScreenRow? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ScreenCell> get cells => $_getList(0);

  @$pb.TagNumber(2)
  CellStyle get tailFill => $_getN(1);
  @$pb.TagNumber(2)
  set tailFill(CellStyle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTailFill() => $_has(1);
  @$pb.TagNumber(2)
  void clearTailFill() => $_clearField(2);
  @$pb.TagNumber(2)
  CellStyle ensureTailFill() => $_ensure(1);

  /// wrapped 表示本物理行通过终端软换行继续到下一行。
  @$pb.TagNumber(3)
  $core.bool get wrapped => $_getBF(2);
  @$pb.TagNumber(3)
  set wrapped($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasWrapped() => $_has(2);
  @$pb.TagNumber(3)
  void clearWrapped() => $_clearField(3);
}

class TerminalCursor extends $pb.GeneratedMessage {
  factory TerminalCursor({
    $core.int? row,
    $core.int? col,
    $core.bool? visible,
    CursorShape? shape,
    $core.bool? blink,
  }) {
    final result = create();
    if (row != null) result.row = row;
    if (col != null) result.col = col;
    if (visible != null) result.visible = visible;
    if (shape != null) result.shape = shape;
    if (blink != null) result.blink = blink;
    return result;
  }

  TerminalCursor._();

  factory TerminalCursor.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalCursor.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalCursor',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'row')
    ..aI(2, _omitFieldNames ? '' : 'col')
    ..aOB(3, _omitFieldNames ? '' : 'visible')
    ..aE<CursorShape>(4, _omitFieldNames ? '' : 'shape',
        enumValues: CursorShape.values)
    ..aOB(5, _omitFieldNames ? '' : 'blink')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCursor clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalCursor copyWith(void Function(TerminalCursor) updates) =>
      super.copyWith((message) => updates(message as TerminalCursor))
          as TerminalCursor;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalCursor create() => TerminalCursor._();
  @$core.override
  TerminalCursor createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalCursor getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalCursor>(create);
  static TerminalCursor? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get row => $_getIZ(0);
  @$pb.TagNumber(1)
  set row($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRow() => $_has(0);
  @$pb.TagNumber(1)
  void clearRow() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get col => $_getIZ(1);
  @$pb.TagNumber(2)
  set col($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCol() => $_has(1);
  @$pb.TagNumber(2)
  void clearCol() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get visible => $_getBF(2);
  @$pb.TagNumber(3)
  set visible($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasVisible() => $_has(2);
  @$pb.TagNumber(3)
  void clearVisible() => $_clearField(3);

  @$pb.TagNumber(4)
  CursorShape get shape => $_getN(3);
  @$pb.TagNumber(4)
  set shape(CursorShape value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasShape() => $_has(3);
  @$pb.TagNumber(4)
  void clearShape() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get blink => $_getBF(4);
  @$pb.TagNumber(5)
  set blink($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBlink() => $_has(4);
  @$pb.TagNumber(5)
  void clearBlink() => $_clearField(5);
}

class TerminalModes extends $pb.GeneratedMessage {
  factory TerminalModes({
    $core.bool? alternateScreen,
    $core.bool? alternateScroll,
    $core.bool? mouseTracking,
    $core.bool? mouseX10,
    $core.bool? mouseNormal,
    $core.bool? mouseButtonEvent,
    $core.bool? mouseAnyEvent,
    $core.bool? mouseSgr,
    $core.bool? bracketedPaste,
    $core.bool? applicationCursor,
    $core.bool? autoWrap,
  }) {
    final result = create();
    if (alternateScreen != null) result.alternateScreen = alternateScreen;
    if (alternateScroll != null) result.alternateScroll = alternateScroll;
    if (mouseTracking != null) result.mouseTracking = mouseTracking;
    if (mouseX10 != null) result.mouseX10 = mouseX10;
    if (mouseNormal != null) result.mouseNormal = mouseNormal;
    if (mouseButtonEvent != null) result.mouseButtonEvent = mouseButtonEvent;
    if (mouseAnyEvent != null) result.mouseAnyEvent = mouseAnyEvent;
    if (mouseSgr != null) result.mouseSgr = mouseSgr;
    if (bracketedPaste != null) result.bracketedPaste = bracketedPaste;
    if (applicationCursor != null) result.applicationCursor = applicationCursor;
    if (autoWrap != null) result.autoWrap = autoWrap;
    return result;
  }

  TerminalModes._();

  factory TerminalModes.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalModes.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalModes',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'alternateScreen')
    ..aOB(2, _omitFieldNames ? '' : 'alternateScroll')
    ..aOB(3, _omitFieldNames ? '' : 'mouseTracking')
    ..aOB(4, _omitFieldNames ? '' : 'mouseX10')
    ..aOB(5, _omitFieldNames ? '' : 'mouseNormal')
    ..aOB(6, _omitFieldNames ? '' : 'mouseButtonEvent')
    ..aOB(7, _omitFieldNames ? '' : 'mouseAnyEvent')
    ..aOB(8, _omitFieldNames ? '' : 'mouseSgr')
    ..aOB(9, _omitFieldNames ? '' : 'bracketedPaste')
    ..aOB(10, _omitFieldNames ? '' : 'applicationCursor')
    ..aOB(11, _omitFieldNames ? '' : 'autoWrap')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalModes clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalModes copyWith(void Function(TerminalModes) updates) =>
      super.copyWith((message) => updates(message as TerminalModes))
          as TerminalModes;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalModes create() => TerminalModes._();
  @$core.override
  TerminalModes createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalModes getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalModes>(create);
  static TerminalModes? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get alternateScreen => $_getBF(0);
  @$pb.TagNumber(1)
  set alternateScreen($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAlternateScreen() => $_has(0);
  @$pb.TagNumber(1)
  void clearAlternateScreen() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get alternateScroll => $_getBF(1);
  @$pb.TagNumber(2)
  set alternateScroll($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAlternateScroll() => $_has(1);
  @$pb.TagNumber(2)
  void clearAlternateScroll() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get mouseTracking => $_getBF(2);
  @$pb.TagNumber(3)
  set mouseTracking($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMouseTracking() => $_has(2);
  @$pb.TagNumber(3)
  void clearMouseTracking() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get mouseX10 => $_getBF(3);
  @$pb.TagNumber(4)
  set mouseX10($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMouseX10() => $_has(3);
  @$pb.TagNumber(4)
  void clearMouseX10() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get mouseNormal => $_getBF(4);
  @$pb.TagNumber(5)
  set mouseNormal($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMouseNormal() => $_has(4);
  @$pb.TagNumber(5)
  void clearMouseNormal() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get mouseButtonEvent => $_getBF(5);
  @$pb.TagNumber(6)
  set mouseButtonEvent($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMouseButtonEvent() => $_has(5);
  @$pb.TagNumber(6)
  void clearMouseButtonEvent() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get mouseAnyEvent => $_getBF(6);
  @$pb.TagNumber(7)
  set mouseAnyEvent($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMouseAnyEvent() => $_has(6);
  @$pb.TagNumber(7)
  void clearMouseAnyEvent() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get mouseSgr => $_getBF(7);
  @$pb.TagNumber(8)
  set mouseSgr($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasMouseSgr() => $_has(7);
  @$pb.TagNumber(8)
  void clearMouseSgr() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get bracketedPaste => $_getBF(8);
  @$pb.TagNumber(9)
  set bracketedPaste($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasBracketedPaste() => $_has(8);
  @$pb.TagNumber(9)
  void clearBracketedPaste() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get applicationCursor => $_getBF(9);
  @$pb.TagNumber(10)
  set applicationCursor($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasApplicationCursor() => $_has(9);
  @$pb.TagNumber(10)
  void clearApplicationCursor() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.bool get autoWrap => $_getBF(10);
  @$pb.TagNumber(11)
  set autoWrap($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(11)
  $core.bool hasAutoWrap() => $_has(10);
  @$pb.TagNumber(11)
  void clearAutoWrap() => $_clearField(11);
}

class HistoryCursor extends $pb.GeneratedMessage {
  factory HistoryCursor({
    $fixnum.Int64? lineId,
    $core.int? rowInLine,
    HistoryCursorSegment? segment,
  }) {
    final result = create();
    if (lineId != null) result.lineId = lineId;
    if (rowInLine != null) result.rowInLine = rowInLine;
    if (segment != null) result.segment = segment;
    return result;
  }

  HistoryCursor._();

  factory HistoryCursor.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryCursor.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryCursor',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'lineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(2, _omitFieldNames ? '' : 'rowInLine')
    ..aE<HistoryCursorSegment>(4, _omitFieldNames ? '' : 'segment',
        enumValues: HistoryCursorSegment.values)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryCursor clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryCursor copyWith(void Function(HistoryCursor) updates) =>
      super.copyWith((message) => updates(message as HistoryCursor))
          as HistoryCursor;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryCursor create() => HistoryCursor._();
  @$core.override
  HistoryCursor createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryCursor getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryCursor>(create);
  static HistoryCursor? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get lineId => $_getI64(0);
  @$pb.TagNumber(1)
  set lineId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLineId() => $_has(0);
  @$pb.TagNumber(1)
  void clearLineId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get rowInLine => $_getIZ(1);
  @$pb.TagNumber(2)
  set rowInLine($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRowInLine() => $_has(1);
  @$pb.TagNumber(2)
  void clearRowInLine() => $_clearField(2);

  @$pb.TagNumber(4)
  HistoryCursorSegment get segment => $_getN(2);
  @$pb.TagNumber(4)
  set segment(HistoryCursorSegment value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSegment() => $_has(2);
  @$pb.TagNumber(4)
  void clearSegment() => $_clearField(4);
}

class HistoryRange extends $pb.GeneratedMessage {
  factory HistoryRange({
    $fixnum.Int64? startLineId,
    $core.int? startCol,
    $fixnum.Int64? endLineId,
    $core.int? endCol,
  }) {
    final result = create();
    if (startLineId != null) result.startLineId = startLineId;
    if (startCol != null) result.startCol = startCol;
    if (endLineId != null) result.endLineId = endLineId;
    if (endCol != null) result.endCol = endCol;
    return result;
  }

  HistoryRange._();

  factory HistoryRange.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryRange.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryRange',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'startLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(2, _omitFieldNames ? '' : 'startCol')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'endLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(4, _omitFieldNames ? '' : 'endCol')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryRange clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryRange copyWith(void Function(HistoryRange) updates) =>
      super.copyWith((message) => updates(message as HistoryRange))
          as HistoryRange;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryRange create() => HistoryRange._();
  @$core.override
  HistoryRange createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryRange getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryRange>(create);
  static HistoryRange? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get startLineId => $_getI64(0);
  @$pb.TagNumber(1)
  set startLineId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStartLineId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStartLineId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get startCol => $_getIZ(1);
  @$pb.TagNumber(2)
  set startCol($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStartCol() => $_has(1);
  @$pb.TagNumber(2)
  void clearStartCol() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get endLineId => $_getI64(2);
  @$pb.TagNumber(3)
  set endLineId($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEndLineId() => $_has(2);
  @$pb.TagNumber(3)
  void clearEndLineId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get endCol => $_getIZ(3);
  @$pb.TagNumber(4)
  set endCol($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEndCol() => $_has(3);
  @$pb.TagNumber(4)
  void clearEndCol() => $_clearField(4);
}

class HistoryWindowCommand extends $pb.GeneratedMessage {
  factory HistoryWindowCommand({
    $0.TerminalRef? terminal,
    HistoryWindowMode? mode,
    $core.int? limit,
    $core.int? cols,
    $core.String? token,
    $fixnum.Int64? historyGeneration,
    HistoryCursor? beforeCursor,
    HistoryCursor? afterCursor,
    $fixnum.Int64? boundaryFirstLineId,
    $fixnum.Int64? boundaryLastLineId,
    HistoryRange? range,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (mode != null) result.mode = mode;
    if (limit != null) result.limit = limit;
    if (cols != null) result.cols = cols;
    if (token != null) result.token = token;
    if (historyGeneration != null) result.historyGeneration = historyGeneration;
    if (beforeCursor != null) result.beforeCursor = beforeCursor;
    if (afterCursor != null) result.afterCursor = afterCursor;
    if (boundaryFirstLineId != null)
      result.boundaryFirstLineId = boundaryFirstLineId;
    if (boundaryLastLineId != null)
      result.boundaryLastLineId = boundaryLastLineId;
    if (range != null) result.range = range;
    return result;
  }

  HistoryWindowCommand._();

  factory HistoryWindowCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryWindowCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryWindowCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aE<HistoryWindowMode>(3, _omitFieldNames ? '' : 'mode',
        enumValues: HistoryWindowMode.values)
    ..aI(5, _omitFieldNames ? '' : 'limit')
    ..aI(6, _omitFieldNames ? '' : 'cols')
    ..aOS(7, _omitFieldNames ? '' : 'token')
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'historyGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<HistoryCursor>(9, _omitFieldNames ? '' : 'beforeCursor',
        subBuilder: HistoryCursor.create)
    ..aOM<HistoryCursor>(10, _omitFieldNames ? '' : 'afterCursor',
        subBuilder: HistoryCursor.create)
    ..a<$fixnum.Int64>(
        11, _omitFieldNames ? '' : 'boundaryFirstLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        12, _omitFieldNames ? '' : 'boundaryLastLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<HistoryRange>(13, _omitFieldNames ? '' : 'range',
        subBuilder: HistoryRange.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryWindowCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryWindowCommand copyWith(void Function(HistoryWindowCommand) updates) =>
      super.copyWith((message) => updates(message as HistoryWindowCommand))
          as HistoryWindowCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryWindowCommand create() => HistoryWindowCommand._();
  @$core.override
  HistoryWindowCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryWindowCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryWindowCommand>(create);
  static HistoryWindowCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal($0.TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  HistoryWindowMode get mode => $_getN(1);
  @$pb.TagNumber(3)
  set mode(HistoryWindowMode value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasMode() => $_has(1);
  @$pb.TagNumber(3)
  void clearMode() => $_clearField(3);

  @$pb.TagNumber(5)
  $core.int get limit => $_getIZ(2);
  @$pb.TagNumber(5)
  set limit($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(5)
  $core.bool hasLimit() => $_has(2);
  @$pb.TagNumber(5)
  void clearLimit() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get cols => $_getIZ(3);
  @$pb.TagNumber(6)
  set cols($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(6)
  $core.bool hasCols() => $_has(3);
  @$pb.TagNumber(6)
  void clearCols() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get token => $_getSZ(4);
  @$pb.TagNumber(7)
  set token($core.String value) => $_setString(4, value);
  @$pb.TagNumber(7)
  $core.bool hasToken() => $_has(4);
  @$pb.TagNumber(7)
  void clearToken() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get historyGeneration => $_getI64(5);
  @$pb.TagNumber(8)
  set historyGeneration($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(8)
  $core.bool hasHistoryGeneration() => $_has(5);
  @$pb.TagNumber(8)
  void clearHistoryGeneration() => $_clearField(8);

  @$pb.TagNumber(9)
  HistoryCursor get beforeCursor => $_getN(6);
  @$pb.TagNumber(9)
  set beforeCursor(HistoryCursor value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasBeforeCursor() => $_has(6);
  @$pb.TagNumber(9)
  void clearBeforeCursor() => $_clearField(9);
  @$pb.TagNumber(9)
  HistoryCursor ensureBeforeCursor() => $_ensure(6);

  @$pb.TagNumber(10)
  HistoryCursor get afterCursor => $_getN(7);
  @$pb.TagNumber(10)
  set afterCursor(HistoryCursor value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasAfterCursor() => $_has(7);
  @$pb.TagNumber(10)
  void clearAfterCursor() => $_clearField(10);
  @$pb.TagNumber(10)
  HistoryCursor ensureAfterCursor() => $_ensure(7);

  @$pb.TagNumber(11)
  $fixnum.Int64 get boundaryFirstLineId => $_getI64(8);
  @$pb.TagNumber(11)
  set boundaryFirstLineId($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(11)
  $core.bool hasBoundaryFirstLineId() => $_has(8);
  @$pb.TagNumber(11)
  void clearBoundaryFirstLineId() => $_clearField(11);

  @$pb.TagNumber(12)
  $fixnum.Int64 get boundaryLastLineId => $_getI64(9);
  @$pb.TagNumber(12)
  set boundaryLastLineId($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(12)
  $core.bool hasBoundaryLastLineId() => $_has(9);
  @$pb.TagNumber(12)
  void clearBoundaryLastLineId() => $_clearField(12);

  @$pb.TagNumber(13)
  HistoryRange get range => $_getN(10);
  @$pb.TagNumber(13)
  set range(HistoryRange value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasRange() => $_has(10);
  @$pb.TagNumber(13)
  void clearRange() => $_clearField(13);
  @$pb.TagNumber(13)
  HistoryRange ensureRange() => $_ensure(10);
}

class HistoryCopyCommand extends $pb.GeneratedMessage {
  factory HistoryCopyCommand({
    $0.TerminalRef? terminal,
    HistoryWindowCommand? window,
    $core.int? maxLines,
    $core.int? maxBytes,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (window != null) result.window = window;
    if (maxLines != null) result.maxLines = maxLines;
    if (maxBytes != null) result.maxBytes = maxBytes;
    return result;
  }

  HistoryCopyCommand._();

  factory HistoryCopyCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryCopyCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryCopyCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aOM<HistoryWindowCommand>(3, _omitFieldNames ? '' : 'window',
        subBuilder: HistoryWindowCommand.create)
    ..aI(4, _omitFieldNames ? '' : 'maxLines')
    ..aI(5, _omitFieldNames ? '' : 'maxBytes')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryCopyCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryCopyCommand copyWith(void Function(HistoryCopyCommand) updates) =>
      super.copyWith((message) => updates(message as HistoryCopyCommand))
          as HistoryCopyCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryCopyCommand create() => HistoryCopyCommand._();
  @$core.override
  HistoryCopyCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryCopyCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryCopyCommand>(create);
  static HistoryCopyCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal($0.TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  HistoryWindowCommand get window => $_getN(1);
  @$pb.TagNumber(3)
  set window(HistoryWindowCommand value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasWindow() => $_has(1);
  @$pb.TagNumber(3)
  void clearWindow() => $_clearField(3);
  @$pb.TagNumber(3)
  HistoryWindowCommand ensureWindow() => $_ensure(1);

  @$pb.TagNumber(4)
  $core.int get maxLines => $_getIZ(2);
  @$pb.TagNumber(4)
  set maxLines($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(4)
  $core.bool hasMaxLines() => $_has(2);
  @$pb.TagNumber(4)
  void clearMaxLines() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get maxBytes => $_getIZ(3);
  @$pb.TagNumber(5)
  set maxBytes($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(5)
  $core.bool hasMaxBytes() => $_has(3);
  @$pb.TagNumber(5)
  void clearMaxBytes() => $_clearField(5);
}

class HistoryTextPosition extends $pb.GeneratedMessage {
  factory HistoryTextPosition({
    $fixnum.Int64? lineId,
    $core.int? col,
  }) {
    final result = create();
    if (lineId != null) result.lineId = lineId;
    if (col != null) result.col = col;
    return result;
  }

  HistoryTextPosition._();

  factory HistoryTextPosition.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryTextPosition.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryTextPosition',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'lineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(2, _omitFieldNames ? '' : 'col')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryTextPosition clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryTextPosition copyWith(void Function(HistoryTextPosition) updates) =>
      super.copyWith((message) => updates(message as HistoryTextPosition))
          as HistoryTextPosition;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryTextPosition create() => HistoryTextPosition._();
  @$core.override
  HistoryTextPosition createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryTextPosition getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryTextPosition>(create);
  static HistoryTextPosition? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get lineId => $_getI64(0);
  @$pb.TagNumber(1)
  set lineId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLineId() => $_has(0);
  @$pb.TagNumber(1)
  void clearLineId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get col => $_getIZ(1);
  @$pb.TagNumber(2)
  set col($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCol() => $_has(1);
  @$pb.TagNumber(2)
  void clearCol() => $_clearField(2);
}

class HistorySearchCommand extends $pb.GeneratedMessage {
  factory HistorySearchCommand({
    $0.TerminalRef? terminal,
    $core.String? token,
    $fixnum.Int64? historyGeneration,
    $core.String? query,
    HistorySearchDirection? direction,
    $core.int? cols,
    $core.int? limit,
    HistoryTextPosition? start,
    HistorySearchMode? mode,
    $core.int? contextBefore,
    $core.bool? scan,
    $core.int? maxMatches,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (token != null) result.token = token;
    if (historyGeneration != null) result.historyGeneration = historyGeneration;
    if (query != null) result.query = query;
    if (direction != null) result.direction = direction;
    if (cols != null) result.cols = cols;
    if (limit != null) result.limit = limit;
    if (start != null) result.start = start;
    if (mode != null) result.mode = mode;
    if (contextBefore != null) result.contextBefore = contextBefore;
    if (scan != null) result.scan = scan;
    if (maxMatches != null) result.maxMatches = maxMatches;
    return result;
  }

  HistorySearchCommand._();

  factory HistorySearchCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistorySearchCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistorySearchCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aOS(3, _omitFieldNames ? '' : 'token')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'historyGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(5, _omitFieldNames ? '' : 'query')
    ..aE<HistorySearchDirection>(6, _omitFieldNames ? '' : 'direction',
        enumValues: HistorySearchDirection.values)
    ..aI(7, _omitFieldNames ? '' : 'cols')
    ..aI(8, _omitFieldNames ? '' : 'limit')
    ..aOM<HistoryTextPosition>(9, _omitFieldNames ? '' : 'start',
        subBuilder: HistoryTextPosition.create)
    ..aE<HistorySearchMode>(10, _omitFieldNames ? '' : 'mode',
        enumValues: HistorySearchMode.values)
    ..aI(11, _omitFieldNames ? '' : 'contextBefore')
    ..aOB(12, _omitFieldNames ? '' : 'scan')
    ..aI(13, _omitFieldNames ? '' : 'maxMatches')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistorySearchCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistorySearchCommand copyWith(void Function(HistorySearchCommand) updates) =>
      super.copyWith((message) => updates(message as HistorySearchCommand))
          as HistorySearchCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistorySearchCommand create() => HistorySearchCommand._();
  @$core.override
  HistorySearchCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistorySearchCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistorySearchCommand>(create);
  static HistorySearchCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal($0.TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  $core.String get token => $_getSZ(1);
  @$pb.TagNumber(3)
  set token($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasToken() => $_has(1);
  @$pb.TagNumber(3)
  void clearToken() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get historyGeneration => $_getI64(2);
  @$pb.TagNumber(4)
  set historyGeneration($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(4)
  $core.bool hasHistoryGeneration() => $_has(2);
  @$pb.TagNumber(4)
  void clearHistoryGeneration() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get query => $_getSZ(3);
  @$pb.TagNumber(5)
  set query($core.String value) => $_setString(3, value);
  @$pb.TagNumber(5)
  $core.bool hasQuery() => $_has(3);
  @$pb.TagNumber(5)
  void clearQuery() => $_clearField(5);

  @$pb.TagNumber(6)
  HistorySearchDirection get direction => $_getN(4);
  @$pb.TagNumber(6)
  set direction(HistorySearchDirection value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasDirection() => $_has(4);
  @$pb.TagNumber(6)
  void clearDirection() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get cols => $_getIZ(5);
  @$pb.TagNumber(7)
  set cols($core.int value) => $_setSignedInt32(5, value);
  @$pb.TagNumber(7)
  $core.bool hasCols() => $_has(5);
  @$pb.TagNumber(7)
  void clearCols() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get limit => $_getIZ(6);
  @$pb.TagNumber(8)
  set limit($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(8)
  $core.bool hasLimit() => $_has(6);
  @$pb.TagNumber(8)
  void clearLimit() => $_clearField(8);

  @$pb.TagNumber(9)
  HistoryTextPosition get start => $_getN(7);
  @$pb.TagNumber(9)
  set start(HistoryTextPosition value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasStart() => $_has(7);
  @$pb.TagNumber(9)
  void clearStart() => $_clearField(9);
  @$pb.TagNumber(9)
  HistoryTextPosition ensureStart() => $_ensure(7);

  @$pb.TagNumber(10)
  HistorySearchMode get mode => $_getN(8);
  @$pb.TagNumber(10)
  set mode(HistorySearchMode value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasMode() => $_has(8);
  @$pb.TagNumber(10)
  void clearMode() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get contextBefore => $_getIZ(9);
  @$pb.TagNumber(11)
  set contextBefore($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(11)
  $core.bool hasContextBefore() => $_has(9);
  @$pb.TagNumber(11)
  void clearContextBefore() => $_clearField(11);

  /// scan returns chronological match batches without a replacement window.
  @$pb.TagNumber(12)
  $core.bool get scan => $_getBF(10);
  @$pb.TagNumber(12)
  set scan($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(12)
  $core.bool hasScan() => $_has(10);
  @$pb.TagNumber(12)
  void clearScan() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.int get maxMatches => $_getIZ(11);
  @$pb.TagNumber(13)
  set maxMatches($core.int value) => $_setSignedInt32(11, value);
  @$pb.TagNumber(13)
  $core.bool hasMaxMatches() => $_has(11);
  @$pb.TagNumber(13)
  void clearMaxMatches() => $_clearField(13);
}

class HistoryReleaseCommand extends $pb.GeneratedMessage {
  factory HistoryReleaseCommand({
    $0.TerminalRef? terminal,
    $core.String? token,
    $fixnum.Int64? historyGeneration,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (token != null) result.token = token;
    if (historyGeneration != null) result.historyGeneration = historyGeneration;
    return result;
  }

  HistoryReleaseCommand._();

  factory HistoryReleaseCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryReleaseCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryReleaseCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aOS(3, _omitFieldNames ? '' : 'token')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'historyGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryReleaseCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryReleaseCommand copyWith(
          void Function(HistoryReleaseCommand) updates) =>
      super.copyWith((message) => updates(message as HistoryReleaseCommand))
          as HistoryReleaseCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryReleaseCommand create() => HistoryReleaseCommand._();
  @$core.override
  HistoryReleaseCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryReleaseCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryReleaseCommand>(create);
  static HistoryReleaseCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal($0.TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  $core.String get token => $_getSZ(1);
  @$pb.TagNumber(3)
  set token($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasToken() => $_has(1);
  @$pb.TagNumber(3)
  void clearToken() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get historyGeneration => $_getI64(2);
  @$pb.TagNumber(4)
  set historyGeneration($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(4)
  $core.bool hasHistoryGeneration() => $_has(2);
  @$pb.TagNumber(4)
  void clearHistoryGeneration() => $_clearField(4);
}

class HistoryBacklogStatusCommand extends $pb.GeneratedMessage {
  factory HistoryBacklogStatusCommand({
    $0.TerminalRef? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  HistoryBacklogStatusCommand._();

  factory HistoryBacklogStatusCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryBacklogStatusCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryBacklogStatusCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryBacklogStatusCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryBacklogStatusCommand copyWith(
          void Function(HistoryBacklogStatusCommand) updates) =>
      super.copyWith(
              (message) => updates(message as HistoryBacklogStatusCommand))
          as HistoryBacklogStatusCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryBacklogStatusCommand create() =>
      HistoryBacklogStatusCommand._();
  @$core.override
  HistoryBacklogStatusCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryBacklogStatusCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryBacklogStatusCommand>(create);
  static HistoryBacklogStatusCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal($0.TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.TerminalRef ensureTerminal() => $_ensure(0);
}

class HistoryLineSpan extends $pb.GeneratedMessage {
  factory HistoryLineSpan({
    $core.int? startRow,
    $core.int? endRow,
    $core.String? rowKind,
    $fixnum.Int64? logicalLineId,
    $fixnum.Int64? sessionId,
    $fixnum.Int64? frameId,
    $core.bool? fixedGrid,
    $core.int? screenCols,
    $fixnum.Int64? timestampStartUnixNano,
    $fixnum.Int64? timestampEndUnixNano,
    $core.bool? clippedBefore,
    $core.bool? clippedAfter,
  }) {
    final result = create();
    if (startRow != null) result.startRow = startRow;
    if (endRow != null) result.endRow = endRow;
    if (rowKind != null) result.rowKind = rowKind;
    if (logicalLineId != null) result.logicalLineId = logicalLineId;
    if (sessionId != null) result.sessionId = sessionId;
    if (frameId != null) result.frameId = frameId;
    if (fixedGrid != null) result.fixedGrid = fixedGrid;
    if (screenCols != null) result.screenCols = screenCols;
    if (timestampStartUnixNano != null)
      result.timestampStartUnixNano = timestampStartUnixNano;
    if (timestampEndUnixNano != null)
      result.timestampEndUnixNano = timestampEndUnixNano;
    if (clippedBefore != null) result.clippedBefore = clippedBefore;
    if (clippedAfter != null) result.clippedAfter = clippedAfter;
    return result;
  }

  HistoryLineSpan._();

  factory HistoryLineSpan.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryLineSpan.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryLineSpan',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'startRow')
    ..aI(2, _omitFieldNames ? '' : 'endRow')
    ..aOS(3, _omitFieldNames ? '' : 'rowKind')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'logicalLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'sessionId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(6, _omitFieldNames ? '' : 'frameId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(7, _omitFieldNames ? '' : 'fixedGrid')
    ..aI(8, _omitFieldNames ? '' : 'screenCols')
    ..aInt64(9, _omitFieldNames ? '' : 'timestampStartUnixNano')
    ..aInt64(10, _omitFieldNames ? '' : 'timestampEndUnixNano')
    ..aOB(11, _omitFieldNames ? '' : 'clippedBefore')
    ..aOB(12, _omitFieldNames ? '' : 'clippedAfter')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryLineSpan clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryLineSpan copyWith(void Function(HistoryLineSpan) updates) =>
      super.copyWith((message) => updates(message as HistoryLineSpan))
          as HistoryLineSpan;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryLineSpan create() => HistoryLineSpan._();
  @$core.override
  HistoryLineSpan createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryLineSpan getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryLineSpan>(create);
  static HistoryLineSpan? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get startRow => $_getIZ(0);
  @$pb.TagNumber(1)
  set startRow($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStartRow() => $_has(0);
  @$pb.TagNumber(1)
  void clearStartRow() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get endRow => $_getIZ(1);
  @$pb.TagNumber(2)
  set endRow($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndRow() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndRow() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get rowKind => $_getSZ(2);
  @$pb.TagNumber(3)
  set rowKind($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRowKind() => $_has(2);
  @$pb.TagNumber(3)
  void clearRowKind() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get logicalLineId => $_getI64(3);
  @$pb.TagNumber(4)
  set logicalLineId($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLogicalLineId() => $_has(3);
  @$pb.TagNumber(4)
  void clearLogicalLineId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get sessionId => $_getI64(4);
  @$pb.TagNumber(5)
  set sessionId($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSessionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearSessionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get frameId => $_getI64(5);
  @$pb.TagNumber(6)
  set frameId($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFrameId() => $_has(5);
  @$pb.TagNumber(6)
  void clearFrameId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get fixedGrid => $_getBF(6);
  @$pb.TagNumber(7)
  set fixedGrid($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasFixedGrid() => $_has(6);
  @$pb.TagNumber(7)
  void clearFixedGrid() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get screenCols => $_getIZ(7);
  @$pb.TagNumber(8)
  set screenCols($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasScreenCols() => $_has(7);
  @$pb.TagNumber(8)
  void clearScreenCols() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get timestampStartUnixNano => $_getI64(8);
  @$pb.TagNumber(9)
  set timestampStartUnixNano($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasTimestampStartUnixNano() => $_has(8);
  @$pb.TagNumber(9)
  void clearTimestampStartUnixNano() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get timestampEndUnixNano => $_getI64(9);
  @$pb.TagNumber(10)
  set timestampEndUnixNano($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasTimestampEndUnixNano() => $_has(9);
  @$pb.TagNumber(10)
  void clearTimestampEndUnixNano() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.bool get clippedBefore => $_getBF(10);
  @$pb.TagNumber(11)
  set clippedBefore($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(11)
  $core.bool hasClippedBefore() => $_has(10);
  @$pb.TagNumber(11)
  void clearClippedBefore() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get clippedAfter => $_getBF(11);
  @$pb.TagNumber(12)
  set clippedAfter($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasClippedAfter() => $_has(11);
  @$pb.TagNumber(12)
  void clearClippedAfter() => $_clearField(12);
}

class HistoryRow extends $pb.GeneratedMessage {
  factory HistoryRow({
    ScreenRow? row,
    $fixnum.Int64? timestampUnixNano,
    $core.String? rowKind,
    $core.bool? wrapped,
    RowOwnership? ownership,
    HistoryCursorSegment? segment,
    $fixnum.Int64? sessionId,
    $fixnum.Int64? frameId,
    $core.bool? fixedGrid,
    $core.int? screenCols,
    $core.int? screenRows,
    $core.bool? screenRowSet,
    $fixnum.Int64? logicalLineId,
    $core.int? rowInLine,
  }) {
    final result = create();
    if (row != null) result.row = row;
    if (timestampUnixNano != null) result.timestampUnixNano = timestampUnixNano;
    if (rowKind != null) result.rowKind = rowKind;
    if (wrapped != null) result.wrapped = wrapped;
    if (ownership != null) result.ownership = ownership;
    if (segment != null) result.segment = segment;
    if (sessionId != null) result.sessionId = sessionId;
    if (frameId != null) result.frameId = frameId;
    if (fixedGrid != null) result.fixedGrid = fixedGrid;
    if (screenCols != null) result.screenCols = screenCols;
    if (screenRows != null) result.screenRows = screenRows;
    if (screenRowSet != null) result.screenRowSet = screenRowSet;
    if (logicalLineId != null) result.logicalLineId = logicalLineId;
    if (rowInLine != null) result.rowInLine = rowInLine;
    return result;
  }

  HistoryRow._();

  factory HistoryRow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryRow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryRow',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<ScreenRow>(1, _omitFieldNames ? '' : 'row',
        subBuilder: ScreenRow.create)
    ..aInt64(2, _omitFieldNames ? '' : 'timestampUnixNano')
    ..aOS(3, _omitFieldNames ? '' : 'rowKind')
    ..aOB(4, _omitFieldNames ? '' : 'wrapped')
    ..aE<RowOwnership>(5, _omitFieldNames ? '' : 'ownership',
        enumValues: RowOwnership.values)
    ..aE<HistoryCursorSegment>(6, _omitFieldNames ? '' : 'segment',
        enumValues: HistoryCursorSegment.values)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'sessionId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(8, _omitFieldNames ? '' : 'frameId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(9, _omitFieldNames ? '' : 'fixedGrid')
    ..aI(10, _omitFieldNames ? '' : 'screenCols')
    ..aI(11, _omitFieldNames ? '' : 'screenRows')
    ..aOB(12, _omitFieldNames ? '' : 'screenRowSet')
    ..a<$fixnum.Int64>(
        14, _omitFieldNames ? '' : 'logicalLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(15, _omitFieldNames ? '' : 'rowInLine')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryRow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryRow copyWith(void Function(HistoryRow) updates) =>
      super.copyWith((message) => updates(message as HistoryRow)) as HistoryRow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryRow create() => HistoryRow._();
  @$core.override
  HistoryRow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryRow getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryRow>(create);
  static HistoryRow? _defaultInstance;

  @$pb.TagNumber(1)
  ScreenRow get row => $_getN(0);
  @$pb.TagNumber(1)
  set row(ScreenRow value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRow() => $_has(0);
  @$pb.TagNumber(1)
  void clearRow() => $_clearField(1);
  @$pb.TagNumber(1)
  ScreenRow ensureRow() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get timestampUnixNano => $_getI64(1);
  @$pb.TagNumber(2)
  set timestampUnixNano($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTimestampUnixNano() => $_has(1);
  @$pb.TagNumber(2)
  void clearTimestampUnixNano() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get rowKind => $_getSZ(2);
  @$pb.TagNumber(3)
  set rowKind($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRowKind() => $_has(2);
  @$pb.TagNumber(3)
  void clearRowKind() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get wrapped => $_getBF(3);
  @$pb.TagNumber(4)
  set wrapped($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasWrapped() => $_has(3);
  @$pb.TagNumber(4)
  void clearWrapped() => $_clearField(4);

  @$pb.TagNumber(5)
  RowOwnership get ownership => $_getN(4);
  @$pb.TagNumber(5)
  set ownership(RowOwnership value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasOwnership() => $_has(4);
  @$pb.TagNumber(5)
  void clearOwnership() => $_clearField(5);

  @$pb.TagNumber(6)
  HistoryCursorSegment get segment => $_getN(5);
  @$pb.TagNumber(6)
  set segment(HistoryCursorSegment value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasSegment() => $_has(5);
  @$pb.TagNumber(6)
  void clearSegment() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get sessionId => $_getI64(6);
  @$pb.TagNumber(7)
  set sessionId($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasSessionId() => $_has(6);
  @$pb.TagNumber(7)
  void clearSessionId() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get frameId => $_getI64(7);
  @$pb.TagNumber(8)
  set frameId($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasFrameId() => $_has(7);
  @$pb.TagNumber(8)
  void clearFrameId() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get fixedGrid => $_getBF(8);
  @$pb.TagNumber(9)
  set fixedGrid($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasFixedGrid() => $_has(8);
  @$pb.TagNumber(9)
  void clearFixedGrid() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get screenCols => $_getIZ(9);
  @$pb.TagNumber(10)
  set screenCols($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasScreenCols() => $_has(9);
  @$pb.TagNumber(10)
  void clearScreenCols() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get screenRows => $_getIZ(10);
  @$pb.TagNumber(11)
  set screenRows($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasScreenRows() => $_has(10);
  @$pb.TagNumber(11)
  void clearScreenRows() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get screenRowSet => $_getBF(11);
  @$pb.TagNumber(12)
  set screenRowSet($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasScreenRowSet() => $_has(11);
  @$pb.TagNumber(12)
  void clearScreenRowSet() => $_clearField(12);

  @$pb.TagNumber(14)
  $fixnum.Int64 get logicalLineId => $_getI64(12);
  @$pb.TagNumber(14)
  set logicalLineId($fixnum.Int64 value) => $_setInt64(12, value);
  @$pb.TagNumber(14)
  $core.bool hasLogicalLineId() => $_has(12);
  @$pb.TagNumber(14)
  void clearLogicalLineId() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.int get rowInLine => $_getIZ(13);
  @$pb.TagNumber(15)
  set rowInLine($core.int value) => $_setSignedInt32(13, value);
  @$pb.TagNumber(15)
  $core.bool hasRowInLine() => $_has(13);
  @$pb.TagNumber(15)
  void clearRowInLine() => $_clearField(15);
}

class HistoryViewportAnchor extends $pb.GeneratedMessage {
  factory HistoryViewportAnchor({
    $fixnum.Int64? topLineId,
    $core.int? topCellOffset,
    $core.bool? atEnd,
    $core.int? screenCols,
    $core.int? screenRows,
  }) {
    final result = create();
    if (topLineId != null) result.topLineId = topLineId;
    if (topCellOffset != null) result.topCellOffset = topCellOffset;
    if (atEnd != null) result.atEnd = atEnd;
    if (screenCols != null) result.screenCols = screenCols;
    if (screenRows != null) result.screenRows = screenRows;
    return result;
  }

  HistoryViewportAnchor._();

  factory HistoryViewportAnchor.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryViewportAnchor.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryViewportAnchor',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'topLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(2, _omitFieldNames ? '' : 'topCellOffset')
    ..aOB(3, _omitFieldNames ? '' : 'atEnd')
    ..aI(4, _omitFieldNames ? '' : 'screenCols', fieldType: $pb.PbFieldType.OU3)
    ..aI(5, _omitFieldNames ? '' : 'screenRows', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryViewportAnchor clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryViewportAnchor copyWith(
          void Function(HistoryViewportAnchor) updates) =>
      super.copyWith((message) => updates(message as HistoryViewportAnchor))
          as HistoryViewportAnchor;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryViewportAnchor create() => HistoryViewportAnchor._();
  @$core.override
  HistoryViewportAnchor createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryViewportAnchor getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryViewportAnchor>(create);
  static HistoryViewportAnchor? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get topLineId => $_getI64(0);
  @$pb.TagNumber(1)
  set topLineId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTopLineId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTopLineId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get topCellOffset => $_getIZ(1);
  @$pb.TagNumber(2)
  set topCellOffset($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTopCellOffset() => $_has(1);
  @$pb.TagNumber(2)
  void clearTopCellOffset() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get atEnd => $_getBF(2);
  @$pb.TagNumber(3)
  set atEnd($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAtEnd() => $_has(2);
  @$pb.TagNumber(3)
  void clearAtEnd() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get screenCols => $_getIZ(3);
  @$pb.TagNumber(4)
  set screenCols($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasScreenCols() => $_has(3);
  @$pb.TagNumber(4)
  void clearScreenCols() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get screenRows => $_getIZ(4);
  @$pb.TagNumber(5)
  set screenRows($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasScreenRows() => $_has(4);
  @$pb.TagNumber(5)
  void clearScreenRows() => $_clearField(5);
}

class HistoryWindowResult extends $pb.GeneratedMessage {
  factory HistoryWindowResult({
    $0.TerminalRef? terminal,
    $core.String? token,
    HistoryWindowOperation? operation,
    $0.TerminalSize? size,
    $core.Iterable<HistoryRow>? rows,
    $core.Iterable<HistoryLineSpan>? lines,
    $core.int? loadedRows,
    $core.int? totalRows,
    $core.int? loadedLines,
    $core.int? logicalTotal,
    $core.bool? hasMore,
    $fixnum.Int64? historyGeneration,
    $fixnum.Int64? firstRowId,
    $fixnum.Int64? lastRowId,
    $fixnum.Int64? firstLineId,
    $fixnum.Int64? lastLineId,
    HistoryCursor? cursor,
    $fixnum.Int64? timestampUnixNano,
    HistoryViewportAnchor? viewportAnchor,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (token != null) result.token = token;
    if (operation != null) result.operation = operation;
    if (size != null) result.size = size;
    if (rows != null) result.rows.addAll(rows);
    if (lines != null) result.lines.addAll(lines);
    if (loadedRows != null) result.loadedRows = loadedRows;
    if (totalRows != null) result.totalRows = totalRows;
    if (loadedLines != null) result.loadedLines = loadedLines;
    if (logicalTotal != null) result.logicalTotal = logicalTotal;
    if (hasMore != null) result.hasMore = hasMore;
    if (historyGeneration != null) result.historyGeneration = historyGeneration;
    if (firstRowId != null) result.firstRowId = firstRowId;
    if (lastRowId != null) result.lastRowId = lastRowId;
    if (firstLineId != null) result.firstLineId = firstLineId;
    if (lastLineId != null) result.lastLineId = lastLineId;
    if (cursor != null) result.cursor = cursor;
    if (timestampUnixNano != null) result.timestampUnixNano = timestampUnixNano;
    if (viewportAnchor != null) result.viewportAnchor = viewportAnchor;
    return result;
  }

  HistoryWindowResult._();

  factory HistoryWindowResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryWindowResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryWindowResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aOS(2, _omitFieldNames ? '' : 'token')
    ..aE<HistoryWindowOperation>(3, _omitFieldNames ? '' : 'operation',
        enumValues: HistoryWindowOperation.values)
    ..aOM<$0.TerminalSize>(4, _omitFieldNames ? '' : 'size',
        subBuilder: $0.TerminalSize.create)
    ..pPM<HistoryRow>(5, _omitFieldNames ? '' : 'rows',
        subBuilder: HistoryRow.create)
    ..pPM<HistoryLineSpan>(6, _omitFieldNames ? '' : 'lines',
        subBuilder: HistoryLineSpan.create)
    ..aI(8, _omitFieldNames ? '' : 'loadedRows')
    ..aI(9, _omitFieldNames ? '' : 'totalRows')
    ..aI(10, _omitFieldNames ? '' : 'loadedLines')
    ..aI(11, _omitFieldNames ? '' : 'logicalTotal')
    ..aOB(12, _omitFieldNames ? '' : 'hasMore')
    ..a<$fixnum.Int64>(
        13, _omitFieldNames ? '' : 'historyGeneration', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        14, _omitFieldNames ? '' : 'firstRowId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        15, _omitFieldNames ? '' : 'lastRowId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        16, _omitFieldNames ? '' : 'firstLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        17, _omitFieldNames ? '' : 'lastLineId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<HistoryCursor>(18, _omitFieldNames ? '' : 'cursor',
        subBuilder: HistoryCursor.create)
    ..aInt64(19, _omitFieldNames ? '' : 'timestampUnixNano')
    ..aOM<HistoryViewportAnchor>(20, _omitFieldNames ? '' : 'viewportAnchor',
        subBuilder: HistoryViewportAnchor.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryWindowResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryWindowResult copyWith(void Function(HistoryWindowResult) updates) =>
      super.copyWith((message) => updates(message as HistoryWindowResult))
          as HistoryWindowResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryWindowResult create() => HistoryWindowResult._();
  @$core.override
  HistoryWindowResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryWindowResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryWindowResult>(create);
  static HistoryWindowResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal($0.TerminalRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get token => $_getSZ(1);
  @$pb.TagNumber(2)
  set token($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasToken() => $_has(1);
  @$pb.TagNumber(2)
  void clearToken() => $_clearField(2);

  @$pb.TagNumber(3)
  HistoryWindowOperation get operation => $_getN(2);
  @$pb.TagNumber(3)
  set operation(HistoryWindowOperation value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasOperation() => $_has(2);
  @$pb.TagNumber(3)
  void clearOperation() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.TerminalSize get size => $_getN(3);
  @$pb.TagNumber(4)
  set size($0.TerminalSize value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSize() => $_has(3);
  @$pb.TagNumber(4)
  void clearSize() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.TerminalSize ensureSize() => $_ensure(3);

  @$pb.TagNumber(5)
  $pb.PbList<HistoryRow> get rows => $_getList(4);

  @$pb.TagNumber(6)
  $pb.PbList<HistoryLineSpan> get lines => $_getList(5);

  @$pb.TagNumber(8)
  $core.int get loadedRows => $_getIZ(6);
  @$pb.TagNumber(8)
  set loadedRows($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(8)
  $core.bool hasLoadedRows() => $_has(6);
  @$pb.TagNumber(8)
  void clearLoadedRows() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.int get totalRows => $_getIZ(7);
  @$pb.TagNumber(9)
  set totalRows($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(9)
  $core.bool hasTotalRows() => $_has(7);
  @$pb.TagNumber(9)
  void clearTotalRows() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get loadedLines => $_getIZ(8);
  @$pb.TagNumber(10)
  set loadedLines($core.int value) => $_setSignedInt32(8, value);
  @$pb.TagNumber(10)
  $core.bool hasLoadedLines() => $_has(8);
  @$pb.TagNumber(10)
  void clearLoadedLines() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get logicalTotal => $_getIZ(9);
  @$pb.TagNumber(11)
  set logicalTotal($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(11)
  $core.bool hasLogicalTotal() => $_has(9);
  @$pb.TagNumber(11)
  void clearLogicalTotal() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.bool get hasMore => $_getBF(10);
  @$pb.TagNumber(12)
  set hasMore($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(12)
  $core.bool hasHasMore() => $_has(10);
  @$pb.TagNumber(12)
  void clearHasMore() => $_clearField(12);

  @$pb.TagNumber(13)
  $fixnum.Int64 get historyGeneration => $_getI64(11);
  @$pb.TagNumber(13)
  set historyGeneration($fixnum.Int64 value) => $_setInt64(11, value);
  @$pb.TagNumber(13)
  $core.bool hasHistoryGeneration() => $_has(11);
  @$pb.TagNumber(13)
  void clearHistoryGeneration() => $_clearField(13);

  @$pb.TagNumber(14)
  $fixnum.Int64 get firstRowId => $_getI64(12);
  @$pb.TagNumber(14)
  set firstRowId($fixnum.Int64 value) => $_setInt64(12, value);
  @$pb.TagNumber(14)
  $core.bool hasFirstRowId() => $_has(12);
  @$pb.TagNumber(14)
  void clearFirstRowId() => $_clearField(14);

  @$pb.TagNumber(15)
  $fixnum.Int64 get lastRowId => $_getI64(13);
  @$pb.TagNumber(15)
  set lastRowId($fixnum.Int64 value) => $_setInt64(13, value);
  @$pb.TagNumber(15)
  $core.bool hasLastRowId() => $_has(13);
  @$pb.TagNumber(15)
  void clearLastRowId() => $_clearField(15);

  @$pb.TagNumber(16)
  $fixnum.Int64 get firstLineId => $_getI64(14);
  @$pb.TagNumber(16)
  set firstLineId($fixnum.Int64 value) => $_setInt64(14, value);
  @$pb.TagNumber(16)
  $core.bool hasFirstLineId() => $_has(14);
  @$pb.TagNumber(16)
  void clearFirstLineId() => $_clearField(16);

  @$pb.TagNumber(17)
  $fixnum.Int64 get lastLineId => $_getI64(15);
  @$pb.TagNumber(17)
  set lastLineId($fixnum.Int64 value) => $_setInt64(15, value);
  @$pb.TagNumber(17)
  $core.bool hasLastLineId() => $_has(15);
  @$pb.TagNumber(17)
  void clearLastLineId() => $_clearField(17);

  @$pb.TagNumber(18)
  HistoryCursor get cursor => $_getN(16);
  @$pb.TagNumber(18)
  set cursor(HistoryCursor value) => $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasCursor() => $_has(16);
  @$pb.TagNumber(18)
  void clearCursor() => $_clearField(18);
  @$pb.TagNumber(18)
  HistoryCursor ensureCursor() => $_ensure(16);

  @$pb.TagNumber(19)
  $fixnum.Int64 get timestampUnixNano => $_getI64(17);
  @$pb.TagNumber(19)
  set timestampUnixNano($fixnum.Int64 value) => $_setInt64(17, value);
  @$pb.TagNumber(19)
  $core.bool hasTimestampUnixNano() => $_has(17);
  @$pb.TagNumber(19)
  void clearTimestampUnixNano() => $_clearField(19);

  @$pb.TagNumber(20)
  HistoryViewportAnchor get viewportAnchor => $_getN(18);
  @$pb.TagNumber(20)
  set viewportAnchor(HistoryViewportAnchor value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasViewportAnchor() => $_has(18);
  @$pb.TagNumber(20)
  void clearViewportAnchor() => $_clearField(20);
  @$pb.TagNumber(20)
  HistoryViewportAnchor ensureViewportAnchor() => $_ensure(18);
}

class HistoryCopyResult extends $pb.GeneratedMessage {
  factory HistoryCopyResult({
    $core.String? text,
    HistoryTextPosition? next,
    $core.bool? done,
  }) {
    final result = create();
    if (text != null) result.text = text;
    if (next != null) result.next = next;
    if (done != null) result.done = done;
    return result;
  }

  HistoryCopyResult._();

  factory HistoryCopyResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryCopyResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryCopyResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'text')
    ..aOM<HistoryTextPosition>(2, _omitFieldNames ? '' : 'next',
        subBuilder: HistoryTextPosition.create)
    ..aOB(3, _omitFieldNames ? '' : 'done')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryCopyResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryCopyResult copyWith(void Function(HistoryCopyResult) updates) =>
      super.copyWith((message) => updates(message as HistoryCopyResult))
          as HistoryCopyResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryCopyResult create() => HistoryCopyResult._();
  @$core.override
  HistoryCopyResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryCopyResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryCopyResult>(create);
  static HistoryCopyResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get text => $_getSZ(0);
  @$pb.TagNumber(1)
  set text($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasText() => $_has(0);
  @$pb.TagNumber(1)
  void clearText() => $_clearField(1);

  @$pb.TagNumber(2)
  HistoryTextPosition get next => $_getN(1);
  @$pb.TagNumber(2)
  set next(HistoryTextPosition value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasNext() => $_has(1);
  @$pb.TagNumber(2)
  void clearNext() => $_clearField(2);
  @$pb.TagNumber(2)
  HistoryTextPosition ensureNext() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.bool get done => $_getBF(2);
  @$pb.TagNumber(3)
  set done($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDone() => $_has(2);
  @$pb.TagNumber(3)
  void clearDone() => $_clearField(3);
}

class HistorySearchResult extends $pb.GeneratedMessage {
  factory HistorySearchResult({
    $core.bool? found,
    HistoryRange? match,
    HistoryWindowResult? window,
    $core.bool? wrapped,
    $core.Iterable<HistoryRange>? scanMatches,
    HistoryTextPosition? scanNext,
    $core.bool? scanDone,
  }) {
    final result = create();
    if (found != null) result.found = found;
    if (match != null) result.match = match;
    if (window != null) result.window = window;
    if (wrapped != null) result.wrapped = wrapped;
    if (scanMatches != null) result.scanMatches.addAll(scanMatches);
    if (scanNext != null) result.scanNext = scanNext;
    if (scanDone != null) result.scanDone = scanDone;
    return result;
  }

  HistorySearchResult._();

  factory HistorySearchResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistorySearchResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistorySearchResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'found')
    ..aOM<HistoryRange>(2, _omitFieldNames ? '' : 'match',
        subBuilder: HistoryRange.create)
    ..aOM<HistoryWindowResult>(3, _omitFieldNames ? '' : 'window',
        subBuilder: HistoryWindowResult.create)
    ..aOB(4, _omitFieldNames ? '' : 'wrapped')
    ..pPM<HistoryRange>(5, _omitFieldNames ? '' : 'scanMatches',
        subBuilder: HistoryRange.create)
    ..aOM<HistoryTextPosition>(6, _omitFieldNames ? '' : 'scanNext',
        subBuilder: HistoryTextPosition.create)
    ..aOB(7, _omitFieldNames ? '' : 'scanDone')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistorySearchResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistorySearchResult copyWith(void Function(HistorySearchResult) updates) =>
      super.copyWith((message) => updates(message as HistorySearchResult))
          as HistorySearchResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistorySearchResult create() => HistorySearchResult._();
  @$core.override
  HistorySearchResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistorySearchResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistorySearchResult>(create);
  static HistorySearchResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get found => $_getBF(0);
  @$pb.TagNumber(1)
  set found($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFound() => $_has(0);
  @$pb.TagNumber(1)
  void clearFound() => $_clearField(1);

  @$pb.TagNumber(2)
  HistoryRange get match => $_getN(1);
  @$pb.TagNumber(2)
  set match(HistoryRange value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasMatch() => $_has(1);
  @$pb.TagNumber(2)
  void clearMatch() => $_clearField(2);
  @$pb.TagNumber(2)
  HistoryRange ensureMatch() => $_ensure(1);

  @$pb.TagNumber(3)
  HistoryWindowResult get window => $_getN(2);
  @$pb.TagNumber(3)
  set window(HistoryWindowResult value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasWindow() => $_has(2);
  @$pb.TagNumber(3)
  void clearWindow() => $_clearField(3);
  @$pb.TagNumber(3)
  HistoryWindowResult ensureWindow() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.bool get wrapped => $_getBF(3);
  @$pb.TagNumber(4)
  set wrapped($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasWrapped() => $_has(3);
  @$pb.TagNumber(4)
  void clearWrapped() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<HistoryRange> get scanMatches => $_getList(4);

  @$pb.TagNumber(6)
  HistoryTextPosition get scanNext => $_getN(5);
  @$pb.TagNumber(6)
  set scanNext(HistoryTextPosition value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasScanNext() => $_has(5);
  @$pb.TagNumber(6)
  void clearScanNext() => $_clearField(6);
  @$pb.TagNumber(6)
  HistoryTextPosition ensureScanNext() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.bool get scanDone => $_getBF(6);
  @$pb.TagNumber(7)
  set scanDone($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasScanDone() => $_has(6);
  @$pb.TagNumber(7)
  void clearScanDone() => $_clearField(7);
}

class HistoryBacklogStatusResult extends $pb.GeneratedMessage {
  factory HistoryBacklogStatusResult({
    $0.TerminalRef? terminal,
    $core.bool? historyEnabled,
    $core.String? outputBufferPolicy,
    $fixnum.Int64? bufferCapacityBytes,
    $fixnum.Int64? residentBytes,
    $fixnum.Int64? aggregateResidentBytes,
    $fixnum.Int64? aggregateBudgetBytes,
    $fixnum.Int64? droppedBytes,
    $fixnum.Int64? gapCount,
    $fixnum.Int64? outputBufferWaitNanos,
    $core.bool? unavailable,
    $core.String? unavailableReason,
    $core.bool? closed,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (historyEnabled != null) result.historyEnabled = historyEnabled;
    if (outputBufferPolicy != null)
      result.outputBufferPolicy = outputBufferPolicy;
    if (bufferCapacityBytes != null)
      result.bufferCapacityBytes = bufferCapacityBytes;
    if (residentBytes != null) result.residentBytes = residentBytes;
    if (aggregateResidentBytes != null)
      result.aggregateResidentBytes = aggregateResidentBytes;
    if (aggregateBudgetBytes != null)
      result.aggregateBudgetBytes = aggregateBudgetBytes;
    if (droppedBytes != null) result.droppedBytes = droppedBytes;
    if (gapCount != null) result.gapCount = gapCount;
    if (outputBufferWaitNanos != null)
      result.outputBufferWaitNanos = outputBufferWaitNanos;
    if (unavailable != null) result.unavailable = unavailable;
    if (unavailableReason != null) result.unavailableReason = unavailableReason;
    if (closed != null) result.closed = closed;
    return result;
  }

  HistoryBacklogStatusResult._();

  factory HistoryBacklogStatusResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HistoryBacklogStatusResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HistoryBacklogStatusResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aOB(2, _omitFieldNames ? '' : 'historyEnabled')
    ..aOS(3, _omitFieldNames ? '' : 'outputBufferPolicy')
    ..aInt64(4, _omitFieldNames ? '' : 'bufferCapacityBytes')
    ..aInt64(5, _omitFieldNames ? '' : 'residentBytes')
    ..aInt64(6, _omitFieldNames ? '' : 'aggregateResidentBytes')
    ..aInt64(7, _omitFieldNames ? '' : 'aggregateBudgetBytes')
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'droppedBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        9, _omitFieldNames ? '' : 'gapCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aInt64(10, _omitFieldNames ? '' : 'outputBufferWaitNanos')
    ..aOB(11, _omitFieldNames ? '' : 'unavailable')
    ..aOS(12, _omitFieldNames ? '' : 'unavailableReason')
    ..aOB(13, _omitFieldNames ? '' : 'closed')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryBacklogStatusResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HistoryBacklogStatusResult copyWith(
          void Function(HistoryBacklogStatusResult) updates) =>
      super.copyWith(
              (message) => updates(message as HistoryBacklogStatusResult))
          as HistoryBacklogStatusResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HistoryBacklogStatusResult create() => HistoryBacklogStatusResult._();
  @$core.override
  HistoryBacklogStatusResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HistoryBacklogStatusResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HistoryBacklogStatusResult>(create);
  static HistoryBacklogStatusResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal($0.TerminalRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get historyEnabled => $_getBF(1);
  @$pb.TagNumber(2)
  set historyEnabled($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHistoryEnabled() => $_has(1);
  @$pb.TagNumber(2)
  void clearHistoryEnabled() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get outputBufferPolicy => $_getSZ(2);
  @$pb.TagNumber(3)
  set outputBufferPolicy($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOutputBufferPolicy() => $_has(2);
  @$pb.TagNumber(3)
  void clearOutputBufferPolicy() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get bufferCapacityBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set bufferCapacityBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBufferCapacityBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearBufferCapacityBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get residentBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set residentBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasResidentBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearResidentBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get aggregateResidentBytes => $_getI64(5);
  @$pb.TagNumber(6)
  set aggregateResidentBytes($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAggregateResidentBytes() => $_has(5);
  @$pb.TagNumber(6)
  void clearAggregateResidentBytes() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get aggregateBudgetBytes => $_getI64(6);
  @$pb.TagNumber(7)
  set aggregateBudgetBytes($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAggregateBudgetBytes() => $_has(6);
  @$pb.TagNumber(7)
  void clearAggregateBudgetBytes() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get droppedBytes => $_getI64(7);
  @$pb.TagNumber(8)
  set droppedBytes($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasDroppedBytes() => $_has(7);
  @$pb.TagNumber(8)
  void clearDroppedBytes() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get gapCount => $_getI64(8);
  @$pb.TagNumber(9)
  set gapCount($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasGapCount() => $_has(8);
  @$pb.TagNumber(9)
  void clearGapCount() => $_clearField(9);

  @$pb.TagNumber(10)
  $fixnum.Int64 get outputBufferWaitNanos => $_getI64(9);
  @$pb.TagNumber(10)
  set outputBufferWaitNanos($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasOutputBufferWaitNanos() => $_has(9);
  @$pb.TagNumber(10)
  void clearOutputBufferWaitNanos() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.bool get unavailable => $_getBF(10);
  @$pb.TagNumber(11)
  set unavailable($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(11)
  $core.bool hasUnavailable() => $_has(10);
  @$pb.TagNumber(11)
  void clearUnavailable() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get unavailableReason => $_getSZ(11);
  @$pb.TagNumber(12)
  set unavailableReason($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasUnavailableReason() => $_has(11);
  @$pb.TagNumber(12)
  void clearUnavailableReason() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.bool get closed => $_getBF(12);
  @$pb.TagNumber(13)
  set closed($core.bool value) => $_setBool(12, value);
  @$pb.TagNumber(13)
  $core.bool hasClosed() => $_has(12);
  @$pb.TagNumber(13)
  void clearClosed() => $_clearField(13);
}

class LiveScreenNextCommand extends $pb.GeneratedMessage {
  factory LiveScreenNextCommand({
    $0.TerminalRef? terminal,
    $fixnum.Int64? observedRevision,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (observedRevision != null) result.observedRevision = observedRevision;
    return result;
  }

  LiveScreenNextCommand._();

  factory LiveScreenNextCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LiveScreenNextCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LiveScreenNextCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'observedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LiveScreenNextCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LiveScreenNextCommand copyWith(
          void Function(LiveScreenNextCommand) updates) =>
      super.copyWith((message) => updates(message as LiveScreenNextCommand))
          as LiveScreenNextCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LiveScreenNextCommand create() => LiveScreenNextCommand._();
  @$core.override
  LiveScreenNextCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LiveScreenNextCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LiveScreenNextCommand>(create);
  static LiveScreenNextCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(2)
  set terminal($0.TerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(3)
  $fixnum.Int64 get observedRevision => $_getI64(1);
  @$pb.TagNumber(3)
  set observedRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(3)
  $core.bool hasObservedRevision() => $_has(1);
  @$pb.TagNumber(3)
  void clearObservedRevision() => $_clearField(3);
}

class ScreenRowCopy extends $pb.GeneratedMessage {
  factory ScreenRowCopy({
    $core.int? sourceRow,
    $core.int? destinationRow,
    $core.int? count,
  }) {
    final result = create();
    if (sourceRow != null) result.sourceRow = sourceRow;
    if (destinationRow != null) result.destinationRow = destinationRow;
    if (count != null) result.count = count;
    return result;
  }

  ScreenRowCopy._();

  factory ScreenRowCopy.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScreenRowCopy.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScreenRowCopy',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'sourceRow')
    ..aI(2, _omitFieldNames ? '' : 'destinationRow')
    ..aI(3, _omitFieldNames ? '' : 'count')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenRowCopy clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenRowCopy copyWith(void Function(ScreenRowCopy) updates) =>
      super.copyWith((message) => updates(message as ScreenRowCopy))
          as ScreenRowCopy;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScreenRowCopy create() => ScreenRowCopy._();
  @$core.override
  ScreenRowCopy createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScreenRowCopy getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScreenRowCopy>(create);
  static ScreenRowCopy? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get sourceRow => $_getIZ(0);
  @$pb.TagNumber(1)
  set sourceRow($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSourceRow() => $_has(0);
  @$pb.TagNumber(1)
  void clearSourceRow() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get destinationRow => $_getIZ(1);
  @$pb.TagNumber(2)
  set destinationRow($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDestinationRow() => $_has(1);
  @$pb.TagNumber(2)
  void clearDestinationRow() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get count => $_getIZ(2);
  @$pb.TagNumber(3)
  set count($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCount() => $_has(2);
  @$pb.TagNumber(3)
  void clearCount() => $_clearField(3);
}

class ScreenRowReplace extends $pb.GeneratedMessage {
  factory ScreenRowReplace({
    $core.int? rowIndex,
    ScreenRow? row,
  }) {
    final result = create();
    if (rowIndex != null) result.rowIndex = rowIndex;
    if (row != null) result.row = row;
    return result;
  }

  ScreenRowReplace._();

  factory ScreenRowReplace.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScreenRowReplace.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScreenRowReplace',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'rowIndex')
    ..aOM<ScreenRow>(2, _omitFieldNames ? '' : 'row',
        subBuilder: ScreenRow.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenRowReplace clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenRowReplace copyWith(void Function(ScreenRowReplace) updates) =>
      super.copyWith((message) => updates(message as ScreenRowReplace))
          as ScreenRowReplace;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScreenRowReplace create() => ScreenRowReplace._();
  @$core.override
  ScreenRowReplace createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScreenRowReplace getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScreenRowReplace>(create);
  static ScreenRowReplace? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get rowIndex => $_getIZ(0);
  @$pb.TagNumber(1)
  set rowIndex($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRowIndex() => $_has(0);
  @$pb.TagNumber(1)
  void clearRowIndex() => $_clearField(1);

  @$pb.TagNumber(2)
  ScreenRow get row => $_getN(1);
  @$pb.TagNumber(2)
  set row(ScreenRow value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRow() => $_has(1);
  @$pb.TagNumber(2)
  void clearRow() => $_clearField(2);
  @$pb.TagNumber(2)
  ScreenRow ensureRow() => $_ensure(1);
}

class NativeScreenResult extends $pb.GeneratedMessage {
  factory NativeScreenResult({
    $0.TerminalRef? terminal,
    $fixnum.Int64? liveRevision,
    $0.TerminalSize? size,
    $core.Iterable<ScreenRowReplace>? rowReplacements,
    $core.bool? alternateScreen,
    TerminalCursor? cursor,
    TerminalModes? modes,
    $fixnum.Int64? timestampUnixNano,
    $fixnum.Int64? baseRevision,
    $core.Iterable<ScreenRowCopy>? rowCopies,
    $core.bool? fullReplace,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    if (liveRevision != null) result.liveRevision = liveRevision;
    if (size != null) result.size = size;
    if (rowReplacements != null) result.rowReplacements.addAll(rowReplacements);
    if (alternateScreen != null) result.alternateScreen = alternateScreen;
    if (cursor != null) result.cursor = cursor;
    if (modes != null) result.modes = modes;
    if (timestampUnixNano != null) result.timestampUnixNano = timestampUnixNano;
    if (baseRevision != null) result.baseRevision = baseRevision;
    if (rowCopies != null) result.rowCopies.addAll(rowCopies);
    if (fullReplace != null) result.fullReplace = fullReplace;
    return result;
  }

  NativeScreenResult._();

  factory NativeScreenResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NativeScreenResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NativeScreenResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.TerminalRef>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'liveRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<$0.TerminalSize>(3, _omitFieldNames ? '' : 'size',
        subBuilder: $0.TerminalSize.create)
    ..pPM<ScreenRowReplace>(4, _omitFieldNames ? '' : 'rowReplacements',
        subBuilder: ScreenRowReplace.create)
    ..aOB(5, _omitFieldNames ? '' : 'alternateScreen')
    ..aOM<TerminalCursor>(6, _omitFieldNames ? '' : 'cursor',
        subBuilder: TerminalCursor.create)
    ..aOM<TerminalModes>(7, _omitFieldNames ? '' : 'modes',
        subBuilder: TerminalModes.create)
    ..aInt64(8, _omitFieldNames ? '' : 'timestampUnixNano')
    ..a<$fixnum.Int64>(
        9, _omitFieldNames ? '' : 'baseRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPM<ScreenRowCopy>(10, _omitFieldNames ? '' : 'rowCopies',
        subBuilder: ScreenRowCopy.create)
    ..aOB(11, _omitFieldNames ? '' : 'fullReplace')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NativeScreenResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NativeScreenResult copyWith(void Function(NativeScreenResult) updates) =>
      super.copyWith((message) => updates(message as NativeScreenResult))
          as NativeScreenResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NativeScreenResult create() => NativeScreenResult._();
  @$core.override
  NativeScreenResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NativeScreenResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NativeScreenResult>(create);
  static NativeScreenResult? _defaultInstance;

  @$pb.TagNumber(1)
  $0.TerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal($0.TerminalRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.TerminalRef ensureTerminal() => $_ensure(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get liveRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set liveRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLiveRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearLiveRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.TerminalSize get size => $_getN(2);
  @$pb.TagNumber(3)
  set size($0.TerminalSize value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSize() => $_has(2);
  @$pb.TagNumber(3)
  void clearSize() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.TerminalSize ensureSize() => $_ensure(2);

  @$pb.TagNumber(4)
  $pb.PbList<ScreenRowReplace> get rowReplacements => $_getList(3);

  @$pb.TagNumber(5)
  $core.bool get alternateScreen => $_getBF(4);
  @$pb.TagNumber(5)
  set alternateScreen($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAlternateScreen() => $_has(4);
  @$pb.TagNumber(5)
  void clearAlternateScreen() => $_clearField(5);

  @$pb.TagNumber(6)
  TerminalCursor get cursor => $_getN(5);
  @$pb.TagNumber(6)
  set cursor(TerminalCursor value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCursor() => $_has(5);
  @$pb.TagNumber(6)
  void clearCursor() => $_clearField(6);
  @$pb.TagNumber(6)
  TerminalCursor ensureCursor() => $_ensure(5);

  @$pb.TagNumber(7)
  TerminalModes get modes => $_getN(6);
  @$pb.TagNumber(7)
  set modes(TerminalModes value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasModes() => $_has(6);
  @$pb.TagNumber(7)
  void clearModes() => $_clearField(7);
  @$pb.TagNumber(7)
  TerminalModes ensureModes() => $_ensure(6);

  @$pb.TagNumber(8)
  $fixnum.Int64 get timestampUnixNano => $_getI64(7);
  @$pb.TagNumber(8)
  set timestampUnixNano($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasTimestampUnixNano() => $_has(7);
  @$pb.TagNumber(8)
  void clearTimestampUnixNano() => $_clearField(8);

  @$pb.TagNumber(9)
  $fixnum.Int64 get baseRevision => $_getI64(8);
  @$pb.TagNumber(9)
  set baseRevision($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasBaseRevision() => $_has(8);
  @$pb.TagNumber(9)
  void clearBaseRevision() => $_clearField(9);

  @$pb.TagNumber(10)
  $pb.PbList<ScreenRowCopy> get rowCopies => $_getList(9);

  @$pb.TagNumber(11)
  $core.bool get fullReplace => $_getBF(10);
  @$pb.TagNumber(11)
  set fullReplace($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(11)
  $core.bool hasFullReplace() => $_has(10);
  @$pb.TagNumber(11)
  void clearFullReplace() => $_clearField(11);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
