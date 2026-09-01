// This is a generated file - do not edit.
//
// Generated from apipb/events.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'common.pb.dart' as $1;
import 'events.pbenum.dart';
import 'storage.pbenum.dart' as $2;
import 'terminal.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'events.pbenum.dart';

class EventSubscribeCommand extends $pb.GeneratedMessage {
  factory EventSubscribeCommand({
    $core.Iterable<ApplicationEventType>? types,
    $0.TerminalRef? terminal,
    $core.String? storageAppId,
    $2.StorageScope? storageScope,
    $core.String? storageOwnerId,
    $core.String? storageKeyPrefix,
  }) {
    final result = create();
    if (types != null) result.types.addAll(types);
    if (terminal != null) result.terminal = terminal;
    if (storageAppId != null) result.storageAppId = storageAppId;
    if (storageScope != null) result.storageScope = storageScope;
    if (storageOwnerId != null) result.storageOwnerId = storageOwnerId;
    if (storageKeyPrefix != null) result.storageKeyPrefix = storageKeyPrefix;
    return result;
  }

  EventSubscribeCommand._();

  factory EventSubscribeCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EventSubscribeCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EventSubscribeCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pc<ApplicationEventType>(
        2, _omitFieldNames ? '' : 'types', $pb.PbFieldType.KE,
        valueOf: ApplicationEventType.valueOf,
        enumValues: ApplicationEventType.values,
        defaultEnumValue:
            ApplicationEventType.APPLICATION_EVENT_TYPE_UNSPECIFIED)
    ..aOM<$0.TerminalRef>(3, _omitFieldNames ? '' : 'terminal',
        subBuilder: $0.TerminalRef.create)
    ..aOS(4, _omitFieldNames ? '' : 'storageAppId')
    ..aE<$2.StorageScope>(5, _omitFieldNames ? '' : 'storageScope',
        enumValues: $2.StorageScope.values)
    ..aOS(6, _omitFieldNames ? '' : 'storageOwnerId')
    ..aOS(7, _omitFieldNames ? '' : 'storageKeyPrefix')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventSubscribeCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventSubscribeCommand copyWith(
          void Function(EventSubscribeCommand) updates) =>
      super.copyWith((message) => updates(message as EventSubscribeCommand))
          as EventSubscribeCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EventSubscribeCommand create() => EventSubscribeCommand._();
  @$core.override
  EventSubscribeCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EventSubscribeCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EventSubscribeCommand>(create);
  static EventSubscribeCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $pb.PbList<ApplicationEventType> get types => $_getList(0);

  @$pb.TagNumber(3)
  $0.TerminalRef get terminal => $_getN(1);
  @$pb.TagNumber(3)
  set terminal($0.TerminalRef value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasTerminal() => $_has(1);
  @$pb.TagNumber(3)
  void clearTerminal() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.TerminalRef ensureTerminal() => $_ensure(1);

  @$pb.TagNumber(4)
  $core.String get storageAppId => $_getSZ(2);
  @$pb.TagNumber(4)
  set storageAppId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(4)
  $core.bool hasStorageAppId() => $_has(2);
  @$pb.TagNumber(4)
  void clearStorageAppId() => $_clearField(4);

  @$pb.TagNumber(5)
  $2.StorageScope get storageScope => $_getN(3);
  @$pb.TagNumber(5)
  set storageScope($2.StorageScope value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasStorageScope() => $_has(3);
  @$pb.TagNumber(5)
  void clearStorageScope() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get storageOwnerId => $_getSZ(4);
  @$pb.TagNumber(6)
  set storageOwnerId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(6)
  $core.bool hasStorageOwnerId() => $_has(4);
  @$pb.TagNumber(6)
  void clearStorageOwnerId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get storageKeyPrefix => $_getSZ(5);
  @$pb.TagNumber(7)
  set storageKeyPrefix($core.String value) => $_setString(5, value);
  @$pb.TagNumber(7)
  $core.bool hasStorageKeyPrefix() => $_has(5);
  @$pb.TagNumber(7)
  void clearStorageKeyPrefix() => $_clearField(7);
}

class EventSubscriptionResult extends $pb.GeneratedMessage {
  factory EventSubscriptionResult({
    $1.ResourceHandle? subscription,
  }) {
    final result = create();
    if (subscription != null) result.subscription = subscription;
    return result;
  }

  EventSubscriptionResult._();

  factory EventSubscriptionResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EventSubscriptionResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EventSubscriptionResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$1.ResourceHandle>(1, _omitFieldNames ? '' : 'subscription',
        subBuilder: $1.ResourceHandle.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventSubscriptionResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventSubscriptionResult copyWith(
          void Function(EventSubscriptionResult) updates) =>
      super.copyWith((message) => updates(message as EventSubscriptionResult))
          as EventSubscriptionResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EventSubscriptionResult create() => EventSubscriptionResult._();
  @$core.override
  EventSubscriptionResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EventSubscriptionResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EventSubscriptionResult>(create);
  static EventSubscriptionResult? _defaultInstance;

  @$pb.TagNumber(1)
  $1.ResourceHandle get subscription => $_getN(0);
  @$pb.TagNumber(1)
  set subscription($1.ResourceHandle value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSubscription() => $_has(0);
  @$pb.TagNumber(1)
  void clearSubscription() => $_clearField(1);
  @$pb.TagNumber(1)
  $1.ResourceHandle ensureSubscription() => $_ensure(0);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
