// This is a generated file - do not edit.
//
// Generated from apipb/plugin.proto.

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

/// All application plugin traffic is routed by the selected authenticated daemon.
class PluginAddress extends $pb.GeneratedMessage {
  factory PluginAddress({
    $core.String? daemonId,
    $core.String? tuiInstanceId,
    $core.String? pluginId,
    $core.String? pluginInstanceId,
    $fixnum.Int64? registrationEpoch,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (tuiInstanceId != null) result.tuiInstanceId = tuiInstanceId;
    if (pluginId != null) result.pluginId = pluginId;
    if (pluginInstanceId != null) result.pluginInstanceId = pluginInstanceId;
    if (registrationEpoch != null) result.registrationEpoch = registrationEpoch;
    return result;
  }

  PluginAddress._();

  factory PluginAddress.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginAddress.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginAddress',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'tuiInstanceId')
    ..aOS(3, _omitFieldNames ? '' : 'pluginId')
    ..aOS(4, _omitFieldNames ? '' : 'pluginInstanceId')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'registrationEpoch', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAddress clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAddress copyWith(void Function(PluginAddress) updates) =>
      super.copyWith((message) => updates(message as PluginAddress))
          as PluginAddress;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginAddress create() => PluginAddress._();
  @$core.override
  PluginAddress createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginAddress getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginAddress>(create);
  static PluginAddress? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tuiInstanceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tuiInstanceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTuiInstanceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTuiInstanceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get pluginId => $_getSZ(2);
  @$pb.TagNumber(3)
  set pluginId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPluginId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPluginId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get pluginInstanceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set pluginInstanceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPluginInstanceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearPluginInstanceId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get registrationEpoch => $_getI64(4);
  @$pb.TagNumber(5)
  set registrationEpoch($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRegistrationEpoch() => $_has(4);
  @$pb.TagNumber(5)
  void clearRegistrationEpoch() => $_clearField(5);
}

class PluginRegisterRequest extends $pb.GeneratedMessage {
  factory PluginRegisterRequest({
    PluginAddress? address,
    $core.Iterable<$core.String>? topics,
    $core.bool? allowPeerMessages,
    $core.bool? daemonService,
    $core.List<$core.int>? previousLease,
  }) {
    final result = create();
    if (address != null) result.address = address;
    if (topics != null) result.topics.addAll(topics);
    if (allowPeerMessages != null) result.allowPeerMessages = allowPeerMessages;
    if (daemonService != null) result.daemonService = daemonService;
    if (previousLease != null) result.previousLease = previousLease;
    return result;
  }

  PluginRegisterRequest._();

  factory PluginRegisterRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginRegisterRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginRegisterRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<PluginAddress>(1, _omitFieldNames ? '' : 'address',
        subBuilder: PluginAddress.create)
    ..pPS(2, _omitFieldNames ? '' : 'topics')
    ..aOB(3, _omitFieldNames ? '' : 'allowPeerMessages')
    ..aOB(4, _omitFieldNames ? '' : 'daemonService')
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'previousLease', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginRegisterRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginRegisterRequest copyWith(
          void Function(PluginRegisterRequest) updates) =>
      super.copyWith((message) => updates(message as PluginRegisterRequest))
          as PluginRegisterRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginRegisterRequest create() => PluginRegisterRequest._();
  @$core.override
  PluginRegisterRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginRegisterRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginRegisterRequest>(create);
  static PluginRegisterRequest? _defaultInstance;

  @$pb.TagNumber(1)
  PluginAddress get address => $_getN(0);
  @$pb.TagNumber(1)
  set address(PluginAddress value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAddress() => $_has(0);
  @$pb.TagNumber(1)
  void clearAddress() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginAddress ensureAddress() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get topics => $_getList(1);

  /// Cross-TUI delivery requires explicit target opt-in. Self-delivery is allowed.
  @$pb.TagNumber(3)
  $core.bool get allowPeerMessages => $_getBF(2);
  @$pb.TagNumber(3)
  set allowPeerMessages($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAllowPeerMessages() => $_has(2);
  @$pb.TagNumber(3)
  void clearAllowPeerMessages() => $_clearField(3);

  /// Only a daemon-local owner may register a background service.
  @$pb.TagNumber(4)
  $core.bool get daemonService => $_getBF(3);
  @$pb.TagNumber(4)
  set daemonService($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDaemonService() => $_has(3);
  @$pb.TagNumber(4)
  void clearDaemonService() => $_clearField(4);

  /// Explicit authenticated handoff; never copied into public addresses.
  @$pb.TagNumber(5)
  $core.List<$core.int> get previousLease => $_getN(4);
  @$pb.TagNumber(5)
  set previousLease($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPreviousLease() => $_has(4);
  @$pb.TagNumber(5)
  void clearPreviousLease() => $_clearField(5);
}

class PluginRegistration extends $pb.GeneratedMessage {
  factory PluginRegistration({
    PluginAddress? address,
    $core.List<$core.int>? sourceLease,
    $core.String? bootEpoch,
    $core.Iterable<PluginAddress>? peers,
  }) {
    final result = create();
    if (address != null) result.address = address;
    if (sourceLease != null) result.sourceLease = sourceLease;
    if (bootEpoch != null) result.bootEpoch = bootEpoch;
    if (peers != null) result.peers.addAll(peers);
    return result;
  }

  PluginRegistration._();

  factory PluginRegistration.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginRegistration.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginRegistration',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<PluginAddress>(1, _omitFieldNames ? '' : 'address',
        subBuilder: PluginAddress.create)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'sourceLease', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'bootEpoch')
    ..pPM<PluginAddress>(4, _omitFieldNames ? '' : 'peers',
        subBuilder: PluginAddress.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginRegistration clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginRegistration copyWith(void Function(PluginRegistration) updates) =>
      super.copyWith((message) => updates(message as PluginRegistration))
          as PluginRegistration;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginRegistration create() => PluginRegistration._();
  @$core.override
  PluginRegistration createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginRegistration getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginRegistration>(create);
  static PluginRegistration? _defaultInstance;

  @$pb.TagNumber(1)
  PluginAddress get address => $_getN(0);
  @$pb.TagNumber(1)
  set address(PluginAddress value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasAddress() => $_has(0);
  @$pb.TagNumber(1)
  void clearAddress() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginAddress ensureAddress() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.List<$core.int> get sourceLease => $_getN(1);
  @$pb.TagNumber(2)
  set sourceLease($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSourceLease() => $_has(1);
  @$pb.TagNumber(2)
  void clearSourceLease() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get bootEpoch => $_getSZ(2);
  @$pb.TagNumber(3)
  set bootEpoch($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBootEpoch() => $_has(2);
  @$pb.TagNumber(3)
  void clearBootEpoch() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<PluginAddress> get peers => $_getList(3);
}

class PluginSendRequest extends $pb.GeneratedMessage {
  factory PluginSendRequest({
    $core.List<$core.int>? sourceLease,
    PluginMessage? message,
  }) {
    final result = create();
    if (sourceLease != null) result.sourceLease = sourceLease;
    if (message != null) result.message = message;
    return result;
  }

  PluginSendRequest._();

  factory PluginSendRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginSendRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginSendRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'sourceLease', $pb.PbFieldType.OY)
    ..aOM<PluginMessage>(2, _omitFieldNames ? '' : 'message',
        subBuilder: PluginMessage.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginSendRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginSendRequest copyWith(void Function(PluginSendRequest) updates) =>
      super.copyWith((message) => updates(message as PluginSendRequest))
          as PluginSendRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginSendRequest create() => PluginSendRequest._();
  @$core.override
  PluginSendRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginSendRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginSendRequest>(create);
  static PluginSendRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get sourceLease => $_getN(0);
  @$pb.TagNumber(1)
  set sourceLease($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSourceLease() => $_has(0);
  @$pb.TagNumber(1)
  void clearSourceLease() => $_clearField(1);

  @$pb.TagNumber(2)
  PluginMessage get message => $_getN(1);
  @$pb.TagNumber(2)
  set message(PluginMessage value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginMessage ensureMessage() => $_ensure(1);
}

class PluginReceiveRequest extends $pb.GeneratedMessage {
  factory PluginReceiveRequest({
    $core.List<$core.int>? sourceLease,
    $core.int? waitMillis,
    $core.int? maxMessages,
  }) {
    final result = create();
    if (sourceLease != null) result.sourceLease = sourceLease;
    if (waitMillis != null) result.waitMillis = waitMillis;
    if (maxMessages != null) result.maxMessages = maxMessages;
    return result;
  }

  PluginReceiveRequest._();

  factory PluginReceiveRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginReceiveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginReceiveRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'sourceLease', $pb.PbFieldType.OY)
    ..aI(2, _omitFieldNames ? '' : 'waitMillis', fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'maxMessages',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginReceiveRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginReceiveRequest copyWith(void Function(PluginReceiveRequest) updates) =>
      super.copyWith((message) => updates(message as PluginReceiveRequest))
          as PluginReceiveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginReceiveRequest create() => PluginReceiveRequest._();
  @$core.override
  PluginReceiveRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginReceiveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginReceiveRequest>(create);
  static PluginReceiveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get sourceLease => $_getN(0);
  @$pb.TagNumber(1)
  set sourceLease($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSourceLease() => $_has(0);
  @$pb.TagNumber(1)
  void clearSourceLease() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get waitMillis => $_getIZ(1);
  @$pb.TagNumber(2)
  set waitMillis($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWaitMillis() => $_has(1);
  @$pb.TagNumber(2)
  void clearWaitMillis() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get maxMessages => $_getIZ(2);
  @$pb.TagNumber(3)
  set maxMessages($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMaxMessages() => $_has(2);
  @$pb.TagNumber(3)
  void clearMaxMessages() => $_clearField(3);
}

class PluginUnregisterRequest extends $pb.GeneratedMessage {
  factory PluginUnregisterRequest({
    $core.List<$core.int>? sourceLease,
  }) {
    final result = create();
    if (sourceLease != null) result.sourceLease = sourceLease;
    return result;
  }

  PluginUnregisterRequest._();

  factory PluginUnregisterRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUnregisterRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUnregisterRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'sourceLease', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUnregisterRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUnregisterRequest copyWith(
          void Function(PluginUnregisterRequest) updates) =>
      super.copyWith((message) => updates(message as PluginUnregisterRequest))
          as PluginUnregisterRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUnregisterRequest create() => PluginUnregisterRequest._();
  @$core.override
  PluginUnregisterRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUnregisterRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUnregisterRequest>(create);
  static PluginUnregisterRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get sourceLease => $_getN(0);
  @$pb.TagNumber(1)
  set sourceLease($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSourceLease() => $_has(0);
  @$pb.TagNumber(1)
  void clearSourceLease() => $_clearField(1);
}

class PluginBatch extends $pb.GeneratedMessage {
  factory PluginBatch({
    $core.Iterable<PluginMessage>? messages,
    $core.bool? resyncRequired,
  }) {
    final result = create();
    if (messages != null) result.messages.addAll(messages);
    if (resyncRequired != null) result.resyncRequired = resyncRequired;
    return result;
  }

  PluginBatch._();

  factory PluginBatch.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginBatch.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginBatch',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pPM<PluginMessage>(1, _omitFieldNames ? '' : 'messages',
        subBuilder: PluginMessage.create)
    ..aOB(2, _omitFieldNames ? '' : 'resyncRequired')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginBatch clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginBatch copyWith(void Function(PluginBatch) updates) =>
      super.copyWith((message) => updates(message as PluginBatch))
          as PluginBatch;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginBatch create() => PluginBatch._();
  @$core.override
  PluginBatch createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginBatch getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginBatch>(create);
  static PluginBatch? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<PluginMessage> get messages => $_getList(0);

  @$pb.TagNumber(2)
  $core.bool get resyncRequired => $_getBF(1);
  @$pb.TagNumber(2)
  set resyncRequired($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasResyncRequired() => $_has(1);
  @$pb.TagNumber(2)
  void clearResyncRequired() => $_clearField(2);
}

class PluginAck extends $pb.GeneratedMessage {
  factory PluginAck({
    $core.int? delivered,
  }) {
    final result = create();
    if (delivered != null) result.delivered = delivered;
    return result;
  }

  PluginAck._();

  factory PluginAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginAck',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'delivered', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAck copyWith(void Function(PluginAck) updates) =>
      super.copyWith((message) => updates(message as PluginAck)) as PluginAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginAck create() => PluginAck._();
  @$core.override
  PluginAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginAck getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PluginAck>(create);
  static PluginAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get delivered => $_getIZ(0);
  @$pb.TagNumber(1)
  set delivered($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDelivered() => $_has(0);
  @$pb.TagNumber(1)
  void clearDelivered() => $_clearField(1);
}

class PluginError extends $pb.GeneratedMessage {
  factory PluginError({
    $core.String? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  PluginError._();

  factory PluginError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginError',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'code')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginError copyWith(void Function(PluginError) updates) =>
      super.copyWith((message) => updates(message as PluginError))
          as PluginError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginError create() => PluginError._();
  @$core.override
  PluginError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginError getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginError>(create);
  static PluginError? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get code => $_getSZ(0);
  @$pb.TagNumber(1)
  set code($core.String value) => $_setString(0, value);
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

enum PluginCommand_Command {
  register,
  send,
  receive,
  unregister,
  state,
  notSet
}

class PluginCommand extends $pb.GeneratedMessage {
  factory PluginCommand({
    PluginRegisterRequest? register,
    PluginSendRequest? send,
    PluginReceiveRequest? receive,
    PluginUnregisterRequest? unregister,
    PluginStateRequest? state,
  }) {
    final result = create();
    if (register != null) result.register = register;
    if (send != null) result.send = send;
    if (receive != null) result.receive = receive;
    if (unregister != null) result.unregister = unregister;
    if (state != null) result.state = state;
    return result;
  }

  PluginCommand._();

  factory PluginCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginCommand_Command>
      _PluginCommand_CommandByTag = {
    1: PluginCommand_Command.register,
    2: PluginCommand_Command.send,
    3: PluginCommand_Command.receive,
    4: PluginCommand_Command.unregister,
    5: PluginCommand_Command.state,
    0: PluginCommand_Command.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3, 4, 5])
    ..aOM<PluginRegisterRequest>(1, _omitFieldNames ? '' : 'register',
        subBuilder: PluginRegisterRequest.create)
    ..aOM<PluginSendRequest>(2, _omitFieldNames ? '' : 'send',
        subBuilder: PluginSendRequest.create)
    ..aOM<PluginReceiveRequest>(3, _omitFieldNames ? '' : 'receive',
        subBuilder: PluginReceiveRequest.create)
    ..aOM<PluginUnregisterRequest>(4, _omitFieldNames ? '' : 'unregister',
        subBuilder: PluginUnregisterRequest.create)
    ..aOM<PluginStateRequest>(5, _omitFieldNames ? '' : 'state',
        subBuilder: PluginStateRequest.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginCommand copyWith(void Function(PluginCommand) updates) =>
      super.copyWith((message) => updates(message as PluginCommand))
          as PluginCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginCommand create() => PluginCommand._();
  @$core.override
  PluginCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginCommand>(create);
  static PluginCommand? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  PluginCommand_Command whichCommand() =>
      _PluginCommand_CommandByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  void clearCommand() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PluginRegisterRequest get register => $_getN(0);
  @$pb.TagNumber(1)
  set register(PluginRegisterRequest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRegister() => $_has(0);
  @$pb.TagNumber(1)
  void clearRegister() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginRegisterRequest ensureRegister() => $_ensure(0);

  @$pb.TagNumber(2)
  PluginSendRequest get send => $_getN(1);
  @$pb.TagNumber(2)
  set send(PluginSendRequest value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasSend() => $_has(1);
  @$pb.TagNumber(2)
  void clearSend() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginSendRequest ensureSend() => $_ensure(1);

  @$pb.TagNumber(3)
  PluginReceiveRequest get receive => $_getN(2);
  @$pb.TagNumber(3)
  set receive(PluginReceiveRequest value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasReceive() => $_has(2);
  @$pb.TagNumber(3)
  void clearReceive() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginReceiveRequest ensureReceive() => $_ensure(2);

  @$pb.TagNumber(4)
  PluginUnregisterRequest get unregister => $_getN(3);
  @$pb.TagNumber(4)
  set unregister(PluginUnregisterRequest value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasUnregister() => $_has(3);
  @$pb.TagNumber(4)
  void clearUnregister() => $_clearField(4);
  @$pb.TagNumber(4)
  PluginUnregisterRequest ensureUnregister() => $_ensure(3);

  @$pb.TagNumber(5)
  PluginStateRequest get state => $_getN(4);
  @$pb.TagNumber(5)
  set state(PluginStateRequest value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasState() => $_has(4);
  @$pb.TagNumber(5)
  void clearState() => $_clearField(5);
  @$pb.TagNumber(5)
  PluginStateRequest ensureState() => $_ensure(4);
}

enum PluginResult_Result { registration, ack, batch, error, state, notSet }

class PluginResult extends $pb.GeneratedMessage {
  factory PluginResult({
    PluginRegistration? registration,
    PluginAck? ack,
    PluginBatch? batch,
    PluginError? error,
    PluginStateSnapshot? state,
  }) {
    final result = create();
    if (registration != null) result.registration = registration;
    if (ack != null) result.ack = ack;
    if (batch != null) result.batch = batch;
    if (error != null) result.error = error;
    if (state != null) result.state = state;
    return result;
  }

  PluginResult._();

  factory PluginResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginResult_Result>
      _PluginResult_ResultByTag = {
    1: PluginResult_Result.registration,
    2: PluginResult_Result.ack,
    3: PluginResult_Result.batch,
    4: PluginResult_Result.error,
    5: PluginResult_Result.state,
    0: PluginResult_Result.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3, 4, 5])
    ..aOM<PluginRegistration>(1, _omitFieldNames ? '' : 'registration',
        subBuilder: PluginRegistration.create)
    ..aOM<PluginAck>(2, _omitFieldNames ? '' : 'ack',
        subBuilder: PluginAck.create)
    ..aOM<PluginBatch>(3, _omitFieldNames ? '' : 'batch',
        subBuilder: PluginBatch.create)
    ..aOM<PluginError>(4, _omitFieldNames ? '' : 'error',
        subBuilder: PluginError.create)
    ..aOM<PluginStateSnapshot>(5, _omitFieldNames ? '' : 'state',
        subBuilder: PluginStateSnapshot.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginResult copyWith(void Function(PluginResult) updates) =>
      super.copyWith((message) => updates(message as PluginResult))
          as PluginResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginResult create() => PluginResult._();
  @$core.override
  PluginResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginResult>(create);
  static PluginResult? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  PluginResult_Result whichResult() =>
      _PluginResult_ResultByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  void clearResult() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PluginRegistration get registration => $_getN(0);
  @$pb.TagNumber(1)
  set registration(PluginRegistration value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRegistration() => $_has(0);
  @$pb.TagNumber(1)
  void clearRegistration() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginRegistration ensureRegistration() => $_ensure(0);

  @$pb.TagNumber(2)
  PluginAck get ack => $_getN(1);
  @$pb.TagNumber(2)
  set ack(PluginAck value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasAck() => $_has(1);
  @$pb.TagNumber(2)
  void clearAck() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginAck ensureAck() => $_ensure(1);

  @$pb.TagNumber(3)
  PluginBatch get batch => $_getN(2);
  @$pb.TagNumber(3)
  set batch(PluginBatch value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasBatch() => $_has(2);
  @$pb.TagNumber(3)
  void clearBatch() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginBatch ensureBatch() => $_ensure(2);

  @$pb.TagNumber(4)
  PluginError get error => $_getN(3);
  @$pb.TagNumber(4)
  set error(PluginError value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(4)
  void clearError() => $_clearField(4);
  @$pb.TagNumber(4)
  PluginError ensureError() => $_ensure(3);

  @$pb.TagNumber(5)
  PluginStateSnapshot get state => $_getN(4);
  @$pb.TagNumber(5)
  set state(PluginStateSnapshot value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasState() => $_has(4);
  @$pb.TagNumber(5)
  void clearState() => $_clearField(5);
  @$pb.TagNumber(5)
  PluginStateSnapshot ensureState() => $_ensure(4);
}

enum PluginMessage_Body {
  interaction,
  operation,
  mountUpdate,
  reply,
  agentReport,
  agentSnapshot,
  payload,
  stateChanged,
  init,
  uiQuery,
  notSet
}

class PluginMessage extends $pb.GeneratedMessage {
  factory PluginMessage({
    $core.String? requestId,
    $core.String? traceId,
    PluginAddress? source,
    PluginAddress? destination,
    $core.String? topic,
    $fixnum.Int64? deadlineUnixMillis,
    $core.String? idempotencyKey,
    $fixnum.Int64? deliverySequence,
    PluginUiInteraction? interaction,
    PluginUiOperation? operation,
    PluginUiMountUpdate? mountUpdate,
    PluginReply? reply,
    PluginAgentReport? agentReport,
    PluginAgentSnapshot? agentSnapshot,
    PluginPayload? payload,
    PluginStateSnapshot? stateChanged,
    PluginUiInit? init,
    PluginUiQuery? uiQuery,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (traceId != null) result.traceId = traceId;
    if (source != null) result.source = source;
    if (destination != null) result.destination = destination;
    if (topic != null) result.topic = topic;
    if (deadlineUnixMillis != null)
      result.deadlineUnixMillis = deadlineUnixMillis;
    if (idempotencyKey != null) result.idempotencyKey = idempotencyKey;
    if (deliverySequence != null) result.deliverySequence = deliverySequence;
    if (interaction != null) result.interaction = interaction;
    if (operation != null) result.operation = operation;
    if (mountUpdate != null) result.mountUpdate = mountUpdate;
    if (reply != null) result.reply = reply;
    if (agentReport != null) result.agentReport = agentReport;
    if (agentSnapshot != null) result.agentSnapshot = agentSnapshot;
    if (payload != null) result.payload = payload;
    if (stateChanged != null) result.stateChanged = stateChanged;
    if (init != null) result.init = init;
    if (uiQuery != null) result.uiQuery = uiQuery;
    return result;
  }

  PluginMessage._();

  factory PluginMessage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginMessage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginMessage_Body>
      _PluginMessage_BodyByTag = {
    20: PluginMessage_Body.interaction,
    21: PluginMessage_Body.operation,
    22: PluginMessage_Body.mountUpdate,
    23: PluginMessage_Body.reply,
    24: PluginMessage_Body.agentReport,
    25: PluginMessage_Body.agentSnapshot,
    26: PluginMessage_Body.payload,
    27: PluginMessage_Body.stateChanged,
    28: PluginMessage_Body.init,
    29: PluginMessage_Body.uiQuery,
    0: PluginMessage_Body.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginMessage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [20, 21, 22, 23, 24, 25, 26, 27, 28, 29])
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'traceId')
    ..aOM<PluginAddress>(3, _omitFieldNames ? '' : 'source',
        subBuilder: PluginAddress.create)
    ..aOM<PluginAddress>(4, _omitFieldNames ? '' : 'destination',
        subBuilder: PluginAddress.create)
    ..aOS(5, _omitFieldNames ? '' : 'topic')
    ..aInt64(6, _omitFieldNames ? '' : 'deadlineUnixMillis')
    ..aOS(7, _omitFieldNames ? '' : 'idempotencyKey')
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'deliverySequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<PluginUiInteraction>(20, _omitFieldNames ? '' : 'interaction',
        subBuilder: PluginUiInteraction.create)
    ..aOM<PluginUiOperation>(21, _omitFieldNames ? '' : 'operation',
        subBuilder: PluginUiOperation.create)
    ..aOM<PluginUiMountUpdate>(22, _omitFieldNames ? '' : 'mountUpdate',
        subBuilder: PluginUiMountUpdate.create)
    ..aOM<PluginReply>(23, _omitFieldNames ? '' : 'reply',
        subBuilder: PluginReply.create)
    ..aOM<PluginAgentReport>(24, _omitFieldNames ? '' : 'agentReport',
        subBuilder: PluginAgentReport.create)
    ..aOM<PluginAgentSnapshot>(25, _omitFieldNames ? '' : 'agentSnapshot',
        subBuilder: PluginAgentSnapshot.create)
    ..aOM<PluginPayload>(26, _omitFieldNames ? '' : 'payload',
        subBuilder: PluginPayload.create)
    ..aOM<PluginStateSnapshot>(27, _omitFieldNames ? '' : 'stateChanged',
        subBuilder: PluginStateSnapshot.create)
    ..aOM<PluginUiInit>(28, _omitFieldNames ? '' : 'init',
        subBuilder: PluginUiInit.create)
    ..aOM<PluginUiQuery>(29, _omitFieldNames ? '' : 'uiQuery',
        subBuilder: PluginUiQuery.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginMessage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginMessage copyWith(void Function(PluginMessage) updates) =>
      super.copyWith((message) => updates(message as PluginMessage))
          as PluginMessage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginMessage create() => PluginMessage._();
  @$core.override
  PluginMessage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginMessage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginMessage>(create);
  static PluginMessage? _defaultInstance;

  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  PluginMessage_Body whichBody() => _PluginMessage_BodyByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  void clearBody() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get traceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set traceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTraceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTraceId() => $_clearField(2);

  /// Source is daemon-stamped, never accepted from an untrusted sender.
  @$pb.TagNumber(3)
  PluginAddress get source => $_getN(2);
  @$pb.TagNumber(3)
  set source(PluginAddress value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSource() => $_has(2);
  @$pb.TagNumber(3)
  void clearSource() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginAddress ensureSource() => $_ensure(2);

  @$pb.TagNumber(4)
  PluginAddress get destination => $_getN(3);
  @$pb.TagNumber(4)
  set destination(PluginAddress value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasDestination() => $_has(3);
  @$pb.TagNumber(4)
  void clearDestination() => $_clearField(4);
  @$pb.TagNumber(4)
  PluginAddress ensureDestination() => $_ensure(3);

  /// Nonempty topic means subscription broadcast; only state/business events qualify.
  @$pb.TagNumber(5)
  $core.String get topic => $_getSZ(4);
  @$pb.TagNumber(5)
  set topic($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTopic() => $_has(4);
  @$pb.TagNumber(5)
  void clearTopic() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get deadlineUnixMillis => $_getI64(5);
  @$pb.TagNumber(6)
  set deadlineUnixMillis($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDeadlineUnixMillis() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeadlineUnixMillis() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get idempotencyKey => $_getSZ(6);
  @$pb.TagNumber(7)
  set idempotencyKey($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasIdempotencyKey() => $_has(6);
  @$pb.TagNumber(7)
  void clearIdempotencyKey() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get deliverySequence => $_getI64(7);
  @$pb.TagNumber(8)
  set deliverySequence($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasDeliverySequence() => $_has(7);
  @$pb.TagNumber(8)
  void clearDeliverySequence() => $_clearField(8);

  @$pb.TagNumber(20)
  PluginUiInteraction get interaction => $_getN(8);
  @$pb.TagNumber(20)
  set interaction(PluginUiInteraction value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasInteraction() => $_has(8);
  @$pb.TagNumber(20)
  void clearInteraction() => $_clearField(20);
  @$pb.TagNumber(20)
  PluginUiInteraction ensureInteraction() => $_ensure(8);

  @$pb.TagNumber(21)
  PluginUiOperation get operation => $_getN(9);
  @$pb.TagNumber(21)
  set operation(PluginUiOperation value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasOperation() => $_has(9);
  @$pb.TagNumber(21)
  void clearOperation() => $_clearField(21);
  @$pb.TagNumber(21)
  PluginUiOperation ensureOperation() => $_ensure(9);

  @$pb.TagNumber(22)
  PluginUiMountUpdate get mountUpdate => $_getN(10);
  @$pb.TagNumber(22)
  set mountUpdate(PluginUiMountUpdate value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasMountUpdate() => $_has(10);
  @$pb.TagNumber(22)
  void clearMountUpdate() => $_clearField(22);
  @$pb.TagNumber(22)
  PluginUiMountUpdate ensureMountUpdate() => $_ensure(10);

  @$pb.TagNumber(23)
  PluginReply get reply => $_getN(11);
  @$pb.TagNumber(23)
  set reply(PluginReply value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasReply() => $_has(11);
  @$pb.TagNumber(23)
  void clearReply() => $_clearField(23);
  @$pb.TagNumber(23)
  PluginReply ensureReply() => $_ensure(11);

  @$pb.TagNumber(24)
  PluginAgentReport get agentReport => $_getN(12);
  @$pb.TagNumber(24)
  set agentReport(PluginAgentReport value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasAgentReport() => $_has(12);
  @$pb.TagNumber(24)
  void clearAgentReport() => $_clearField(24);
  @$pb.TagNumber(24)
  PluginAgentReport ensureAgentReport() => $_ensure(12);

  @$pb.TagNumber(25)
  PluginAgentSnapshot get agentSnapshot => $_getN(13);
  @$pb.TagNumber(25)
  set agentSnapshot(PluginAgentSnapshot value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasAgentSnapshot() => $_has(13);
  @$pb.TagNumber(25)
  void clearAgentSnapshot() => $_clearField(25);
  @$pb.TagNumber(25)
  PluginAgentSnapshot ensureAgentSnapshot() => $_ensure(13);

  @$pb.TagNumber(26)
  PluginPayload get payload => $_getN(14);
  @$pb.TagNumber(26)
  set payload(PluginPayload value) => $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasPayload() => $_has(14);
  @$pb.TagNumber(26)
  void clearPayload() => $_clearField(26);
  @$pb.TagNumber(26)
  PluginPayload ensurePayload() => $_ensure(14);

  @$pb.TagNumber(27)
  PluginStateSnapshot get stateChanged => $_getN(15);
  @$pb.TagNumber(27)
  set stateChanged(PluginStateSnapshot value) => $_setField(27, value);
  @$pb.TagNumber(27)
  $core.bool hasStateChanged() => $_has(15);
  @$pb.TagNumber(27)
  void clearStateChanged() => $_clearField(27);
  @$pb.TagNumber(27)
  PluginStateSnapshot ensureStateChanged() => $_ensure(15);

  @$pb.TagNumber(28)
  PluginUiInit get init => $_getN(16);
  @$pb.TagNumber(28)
  set init(PluginUiInit value) => $_setField(28, value);
  @$pb.TagNumber(28)
  $core.bool hasInit() => $_has(16);
  @$pb.TagNumber(28)
  void clearInit() => $_clearField(28);
  @$pb.TagNumber(28)
  PluginUiInit ensureInit() => $_ensure(16);

  @$pb.TagNumber(29)
  PluginUiQuery get uiQuery => $_getN(17);
  @$pb.TagNumber(29)
  set uiQuery(PluginUiQuery value) => $_setField(29, value);
  @$pb.TagNumber(29)
  $core.bool hasUiQuery() => $_has(17);
  @$pb.TagNumber(29)
  void clearUiQuery() => $_clearField(29);
  @$pb.TagNumber(29)
  PluginUiQuery ensureUiQuery() => $_ensure(17);
}

class PluginPayload extends $pb.GeneratedMessage {
  factory PluginPayload({
    $core.String? schema,
    $core.int? version,
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (schema != null) result.schema = schema;
    if (version != null) result.version = version;
    if (data != null) result.data = data;
    return result;
  }

  PluginPayload._();

  factory PluginPayload.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginPayload.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginPayload',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'schema')
    ..aI(2, _omitFieldNames ? '' : 'version', fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPayload clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPayload copyWith(void Function(PluginPayload) updates) =>
      super.copyWith((message) => updates(message as PluginPayload))
          as PluginPayload;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginPayload create() => PluginPayload._();
  @$core.override
  PluginPayload createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginPayload getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginPayload>(create);
  static PluginPayload? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get schema => $_getSZ(0);
  @$pb.TagNumber(1)
  set schema($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSchema() => $_has(0);
  @$pb.TagNumber(1)
  void clearSchema() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get version => $_getIZ(1);
  @$pb.TagNumber(2)
  set version($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get data => $_getN(2);
  @$pb.TagNumber(3)
  set data($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasData() => $_has(2);
  @$pb.TagNumber(3)
  void clearData() => $_clearField(3);
}

class PluginReply extends $pb.GeneratedMessage {
  factory PluginReply({
    $core.String? requestId,
    PluginError? error,
    PluginPayload? value,
    PluginUiSnapshot? uiSnapshot,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (error != null) result.error = error;
    if (value != null) result.value = value;
    if (uiSnapshot != null) result.uiSnapshot = uiSnapshot;
    return result;
  }

  PluginReply._();

  factory PluginReply.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginReply.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginReply',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<PluginError>(2, _omitFieldNames ? '' : 'error',
        subBuilder: PluginError.create)
    ..aOM<PluginPayload>(3, _omitFieldNames ? '' : 'value',
        subBuilder: PluginPayload.create)
    ..aOM<PluginUiSnapshot>(4, _omitFieldNames ? '' : 'uiSnapshot',
        subBuilder: PluginUiSnapshot.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginReply clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginReply copyWith(void Function(PluginReply) updates) =>
      super.copyWith((message) => updates(message as PluginReply))
          as PluginReply;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginReply create() => PluginReply._();
  @$core.override
  PluginReply createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginReply getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginReply>(create);
  static PluginReply? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  PluginError get error => $_getN(1);
  @$pb.TagNumber(2)
  set error(PluginError value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginError ensureError() => $_ensure(1);

  @$pb.TagNumber(3)
  PluginPayload get value => $_getN(2);
  @$pb.TagNumber(3)
  set value(PluginPayload value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasValue() => $_has(2);
  @$pb.TagNumber(3)
  void clearValue() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginPayload ensureValue() => $_ensure(2);

  @$pb.TagNumber(4)
  PluginUiSnapshot get uiSnapshot => $_getN(3);
  @$pb.TagNumber(4)
  set uiSnapshot(PluginUiSnapshot value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasUiSnapshot() => $_has(3);
  @$pb.TagNumber(4)
  void clearUiSnapshot() => $_clearField(4);
  @$pb.TagNumber(4)
  PluginUiSnapshot ensureUiSnapshot() => $_ensure(3);
}

class PluginTerminalRef extends $pb.GeneratedMessage {
  factory PluginTerminalRef({
    $core.String? daemonId,
    $core.String? terminalId,
  }) {
    final result = create();
    if (daemonId != null) result.daemonId = daemonId;
    if (terminalId != null) result.terminalId = terminalId;
    return result;
  }

  PluginTerminalRef._();

  factory PluginTerminalRef.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginTerminalRef.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginTerminalRef',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'daemonId')
    ..aOS(2, _omitFieldNames ? '' : 'terminalId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginTerminalRef clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginTerminalRef copyWith(void Function(PluginTerminalRef) updates) =>
      super.copyWith((message) => updates(message as PluginTerminalRef))
          as PluginTerminalRef;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginTerminalRef create() => PluginTerminalRef._();
  @$core.override
  PluginTerminalRef createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginTerminalRef getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginTerminalRef>(create);
  static PluginTerminalRef? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get daemonId => $_getSZ(0);
  @$pb.TagNumber(1)
  set daemonId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDaemonId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDaemonId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get terminalId => $_getSZ(1);
  @$pb.TagNumber(2)
  set terminalId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminalId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTerminalId() => $_clearField(2);
}

class PluginWorkspaceOwner extends $pb.GeneratedMessage {
  factory PluginWorkspaceOwner({
    $core.String? workspaceId,
  }) {
    final result = create();
    if (workspaceId != null) result.workspaceId = workspaceId;
    return result;
  }

  PluginWorkspaceOwner._();

  factory PluginWorkspaceOwner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginWorkspaceOwner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginWorkspaceOwner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'workspaceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginWorkspaceOwner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginWorkspaceOwner copyWith(void Function(PluginWorkspaceOwner) updates) =>
      super.copyWith((message) => updates(message as PluginWorkspaceOwner))
          as PluginWorkspaceOwner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginWorkspaceOwner create() => PluginWorkspaceOwner._();
  @$core.override
  PluginWorkspaceOwner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginWorkspaceOwner getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginWorkspaceOwner>(create);
  static PluginWorkspaceOwner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get workspaceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set workspaceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkspaceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkspaceId() => $_clearField(1);
}

class PluginTabOwner extends $pb.GeneratedMessage {
  factory PluginTabOwner({
    $core.String? workspaceId,
    $core.String? tabId,
  }) {
    final result = create();
    if (workspaceId != null) result.workspaceId = workspaceId;
    if (tabId != null) result.tabId = tabId;
    return result;
  }

  PluginTabOwner._();

  factory PluginTabOwner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginTabOwner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginTabOwner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'workspaceId')
    ..aOS(2, _omitFieldNames ? '' : 'tabId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginTabOwner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginTabOwner copyWith(void Function(PluginTabOwner) updates) =>
      super.copyWith((message) => updates(message as PluginTabOwner))
          as PluginTabOwner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginTabOwner create() => PluginTabOwner._();
  @$core.override
  PluginTabOwner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginTabOwner getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginTabOwner>(create);
  static PluginTabOwner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get workspaceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set workspaceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkspaceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkspaceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tabId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tabId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTabId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTabId() => $_clearField(2);
}

class PluginPanelOwner extends $pb.GeneratedMessage {
  factory PluginPanelOwner({
    $core.String? workspaceId,
    $core.String? tabId,
    $core.String? paneId,
  }) {
    final result = create();
    if (workspaceId != null) result.workspaceId = workspaceId;
    if (tabId != null) result.tabId = tabId;
    if (paneId != null) result.paneId = paneId;
    return result;
  }

  PluginPanelOwner._();

  factory PluginPanelOwner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginPanelOwner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginPanelOwner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'workspaceId')
    ..aOS(2, _omitFieldNames ? '' : 'tabId')
    ..aOS(3, _omitFieldNames ? '' : 'paneId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPanelOwner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPanelOwner copyWith(void Function(PluginPanelOwner) updates) =>
      super.copyWith((message) => updates(message as PluginPanelOwner))
          as PluginPanelOwner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginPanelOwner create() => PluginPanelOwner._();
  @$core.override
  PluginPanelOwner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginPanelOwner getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginPanelOwner>(create);
  static PluginPanelOwner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get workspaceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set workspaceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkspaceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkspaceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tabId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tabId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTabId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTabId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get paneId => $_getSZ(2);
  @$pb.TagNumber(3)
  set paneId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPaneId() => $_has(2);
  @$pb.TagNumber(3)
  void clearPaneId() => $_clearField(3);
}

class PluginFloatingOwner extends $pb.GeneratedMessage {
  factory PluginFloatingOwner({
    $core.String? workspaceId,
    $core.String? floatingId,
    $core.String? tabId,
  }) {
    final result = create();
    if (workspaceId != null) result.workspaceId = workspaceId;
    if (floatingId != null) result.floatingId = floatingId;
    if (tabId != null) result.tabId = tabId;
    return result;
  }

  PluginFloatingOwner._();

  factory PluginFloatingOwner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginFloatingOwner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginFloatingOwner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'workspaceId')
    ..aOS(2, _omitFieldNames ? '' : 'floatingId')
    ..aOS(3, _omitFieldNames ? '' : 'tabId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginFloatingOwner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginFloatingOwner copyWith(void Function(PluginFloatingOwner) updates) =>
      super.copyWith((message) => updates(message as PluginFloatingOwner))
          as PluginFloatingOwner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginFloatingOwner create() => PluginFloatingOwner._();
  @$core.override
  PluginFloatingOwner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginFloatingOwner getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginFloatingOwner>(create);
  static PluginFloatingOwner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get workspaceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set workspaceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkspaceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkspaceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get floatingId => $_getSZ(1);
  @$pb.TagNumber(2)
  set floatingId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasFloatingId() => $_has(1);
  @$pb.TagNumber(2)
  void clearFloatingId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get tabId => $_getSZ(2);
  @$pb.TagNumber(3)
  set tabId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTabId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTabId() => $_clearField(3);
}

enum PluginMountOwner_Owner { workspace, tab, panel, floating, notSet }

class PluginMountOwner extends $pb.GeneratedMessage {
  factory PluginMountOwner({
    PluginWorkspaceOwner? workspace,
    PluginTabOwner? tab,
    PluginPanelOwner? panel,
    PluginFloatingOwner? floating,
  }) {
    final result = create();
    if (workspace != null) result.workspace = workspace;
    if (tab != null) result.tab = tab;
    if (panel != null) result.panel = panel;
    if (floating != null) result.floating = floating;
    return result;
  }

  PluginMountOwner._();

  factory PluginMountOwner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginMountOwner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginMountOwner_Owner>
      _PluginMountOwner_OwnerByTag = {
    1: PluginMountOwner_Owner.workspace,
    2: PluginMountOwner_Owner.tab,
    3: PluginMountOwner_Owner.panel,
    4: PluginMountOwner_Owner.floating,
    0: PluginMountOwner_Owner.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginMountOwner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3, 4])
    ..aOM<PluginWorkspaceOwner>(1, _omitFieldNames ? '' : 'workspace',
        subBuilder: PluginWorkspaceOwner.create)
    ..aOM<PluginTabOwner>(2, _omitFieldNames ? '' : 'tab',
        subBuilder: PluginTabOwner.create)
    ..aOM<PluginPanelOwner>(3, _omitFieldNames ? '' : 'panel',
        subBuilder: PluginPanelOwner.create)
    ..aOM<PluginFloatingOwner>(4, _omitFieldNames ? '' : 'floating',
        subBuilder: PluginFloatingOwner.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginMountOwner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginMountOwner copyWith(void Function(PluginMountOwner) updates) =>
      super.copyWith((message) => updates(message as PluginMountOwner))
          as PluginMountOwner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginMountOwner create() => PluginMountOwner._();
  @$core.override
  PluginMountOwner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginMountOwner getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginMountOwner>(create);
  static PluginMountOwner? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  PluginMountOwner_Owner whichOwner() =>
      _PluginMountOwner_OwnerByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  void clearOwner() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PluginWorkspaceOwner get workspace => $_getN(0);
  @$pb.TagNumber(1)
  set workspace(PluginWorkspaceOwner value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkspace() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkspace() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginWorkspaceOwner ensureWorkspace() => $_ensure(0);

  @$pb.TagNumber(2)
  PluginTabOwner get tab => $_getN(1);
  @$pb.TagNumber(2)
  set tab(PluginTabOwner value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTab() => $_has(1);
  @$pb.TagNumber(2)
  void clearTab() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginTabOwner ensureTab() => $_ensure(1);

  @$pb.TagNumber(3)
  PluginPanelOwner get panel => $_getN(2);
  @$pb.TagNumber(3)
  set panel(PluginPanelOwner value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasPanel() => $_has(2);
  @$pb.TagNumber(3)
  void clearPanel() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginPanelOwner ensurePanel() => $_ensure(2);

  @$pb.TagNumber(4)
  PluginFloatingOwner get floating => $_getN(3);
  @$pb.TagNumber(4)
  set floating(PluginFloatingOwner value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasFloating() => $_has(3);
  @$pb.TagNumber(4)
  void clearFloating() => $_clearField(4);
  @$pb.TagNumber(4)
  PluginFloatingOwner ensureFloating() => $_ensure(3);
}

class PluginTargetContext extends $pb.GeneratedMessage {
  factory PluginTargetContext({
    $core.String? contextId,
    $core.String? tuiInstanceId,
    $core.String? workspaceId,
    $core.String? tabId,
    $core.String? paneId,
    $fixnum.Int64? bindingRevision,
    $core.String? mountId,
    $core.String? floatingId,
  }) {
    final result = create();
    if (contextId != null) result.contextId = contextId;
    if (tuiInstanceId != null) result.tuiInstanceId = tuiInstanceId;
    if (workspaceId != null) result.workspaceId = workspaceId;
    if (tabId != null) result.tabId = tabId;
    if (paneId != null) result.paneId = paneId;
    if (bindingRevision != null) result.bindingRevision = bindingRevision;
    if (mountId != null) result.mountId = mountId;
    if (floatingId != null) result.floatingId = floatingId;
    return result;
  }

  PluginTargetContext._();

  factory PluginTargetContext.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginTargetContext.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginTargetContext',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'contextId')
    ..aOS(2, _omitFieldNames ? '' : 'tuiInstanceId')
    ..aOS(3, _omitFieldNames ? '' : 'workspaceId')
    ..aOS(4, _omitFieldNames ? '' : 'tabId')
    ..aOS(5, _omitFieldNames ? '' : 'paneId')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'bindingRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(7, _omitFieldNames ? '' : 'mountId')
    ..aOS(8, _omitFieldNames ? '' : 'floatingId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginTargetContext clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginTargetContext copyWith(void Function(PluginTargetContext) updates) =>
      super.copyWith((message) => updates(message as PluginTargetContext))
          as PluginTargetContext;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginTargetContext create() => PluginTargetContext._();
  @$core.override
  PluginTargetContext createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginTargetContext getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginTargetContext>(create);
  static PluginTargetContext? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get contextId => $_getSZ(0);
  @$pb.TagNumber(1)
  set contextId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasContextId() => $_has(0);
  @$pb.TagNumber(1)
  void clearContextId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get tuiInstanceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set tuiInstanceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTuiInstanceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTuiInstanceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get workspaceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set workspaceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasWorkspaceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearWorkspaceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get tabId => $_getSZ(3);
  @$pb.TagNumber(4)
  set tabId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTabId() => $_has(3);
  @$pb.TagNumber(4)
  void clearTabId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get paneId => $_getSZ(4);
  @$pb.TagNumber(5)
  set paneId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPaneId() => $_has(4);
  @$pb.TagNumber(5)
  void clearPaneId() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get bindingRevision => $_getI64(5);
  @$pb.TagNumber(6)
  set bindingRevision($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasBindingRevision() => $_has(5);
  @$pb.TagNumber(6)
  void clearBindingRevision() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get mountId => $_getSZ(6);
  @$pb.TagNumber(7)
  set mountId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMountId() => $_has(6);
  @$pb.TagNumber(7)
  void clearMountId() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get floatingId => $_getSZ(7);
  @$pb.TagNumber(8)
  set floatingId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasFloatingId() => $_has(7);
  @$pb.TagNumber(8)
  void clearFloatingId() => $_clearField(8);
}

class PluginUiInteraction extends $pb.GeneratedMessage {
  factory PluginUiInteraction({
    $core.String? mountId,
    $fixnum.Int64? mountRevision,
    $core.String? nodeId,
    $core.String? actionId,
    $core.String? itemId,
    $core.String? value,
    PluginTargetContext? context,
    $core.String? kind,
    $core.Iterable<$core.String>? modifiers,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? values,
  }) {
    final result = create();
    if (mountId != null) result.mountId = mountId;
    if (mountRevision != null) result.mountRevision = mountRevision;
    if (nodeId != null) result.nodeId = nodeId;
    if (actionId != null) result.actionId = actionId;
    if (itemId != null) result.itemId = itemId;
    if (value != null) result.value = value;
    if (context != null) result.context = context;
    if (kind != null) result.kind = kind;
    if (modifiers != null) result.modifiers.addAll(modifiers);
    if (values != null) result.values.addEntries(values);
    return result;
  }

  PluginUiInteraction._();

  factory PluginUiInteraction.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiInteraction.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiInteraction',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'mountId')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'mountRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'nodeId')
    ..aOS(4, _omitFieldNames ? '' : 'actionId')
    ..aOS(5, _omitFieldNames ? '' : 'itemId')
    ..aOS(6, _omitFieldNames ? '' : 'value')
    ..aOM<PluginTargetContext>(7, _omitFieldNames ? '' : 'context',
        subBuilder: PluginTargetContext.create)
    ..aOS(8, _omitFieldNames ? '' : 'kind')
    ..pPS(9, _omitFieldNames ? '' : 'modifiers')
    ..m<$core.String, $core.String>(10, _omitFieldNames ? '' : 'values',
        entryClassName: 'PluginUiInteraction.ValuesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('anytty.api.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiInteraction clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiInteraction copyWith(void Function(PluginUiInteraction) updates) =>
      super.copyWith((message) => updates(message as PluginUiInteraction))
          as PluginUiInteraction;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiInteraction create() => PluginUiInteraction._();
  @$core.override
  PluginUiInteraction createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiInteraction getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiInteraction>(create);
  static PluginUiInteraction? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get mountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set mountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMountId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get mountRevision => $_getI64(1);
  @$pb.TagNumber(2)
  set mountRevision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMountRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearMountRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get nodeId => $_getSZ(2);
  @$pb.TagNumber(3)
  set nodeId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNodeId() => $_has(2);
  @$pb.TagNumber(3)
  void clearNodeId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get actionId => $_getSZ(3);
  @$pb.TagNumber(4)
  set actionId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasActionId() => $_has(3);
  @$pb.TagNumber(4)
  void clearActionId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get itemId => $_getSZ(4);
  @$pb.TagNumber(5)
  set itemId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasItemId() => $_has(4);
  @$pb.TagNumber(5)
  void clearItemId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get value => $_getSZ(5);
  @$pb.TagNumber(6)
  set value($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasValue() => $_has(5);
  @$pb.TagNumber(6)
  void clearValue() => $_clearField(6);

  @$pb.TagNumber(7)
  PluginTargetContext get context => $_getN(6);
  @$pb.TagNumber(7)
  set context(PluginTargetContext value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasContext() => $_has(6);
  @$pb.TagNumber(7)
  void clearContext() => $_clearField(7);
  @$pb.TagNumber(7)
  PluginTargetContext ensureContext() => $_ensure(6);

  @$pb.TagNumber(8)
  $core.String get kind => $_getSZ(7);
  @$pb.TagNumber(8)
  set kind($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasKind() => $_has(7);
  @$pb.TagNumber(8)
  void clearKind() => $_clearField(8);

  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get modifiers => $_getList(8);

  @$pb.TagNumber(10)
  $pb.PbMap<$core.String, $core.String> get values => $_getMap(9);
}

class PluginPaneBind extends $pb.GeneratedMessage {
  factory PluginPaneBind({
    PluginTerminalRef? terminal,
  }) {
    final result = create();
    if (terminal != null) result.terminal = terminal;
    return result;
  }

  PluginPaneBind._();

  factory PluginPaneBind.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginPaneBind.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginPaneBind',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<PluginTerminalRef>(1, _omitFieldNames ? '' : 'terminal',
        subBuilder: PluginTerminalRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPaneBind clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPaneBind copyWith(void Function(PluginPaneBind) updates) =>
      super.copyWith((message) => updates(message as PluginPaneBind))
          as PluginPaneBind;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginPaneBind create() => PluginPaneBind._();
  @$core.override
  PluginPaneBind createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginPaneBind getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginPaneBind>(create);
  static PluginPaneBind? _defaultInstance;

  @$pb.TagNumber(1)
  PluginTerminalRef get terminal => $_getN(0);
  @$pb.TagNumber(1)
  set terminal(PluginTerminalRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTerminal() => $_has(0);
  @$pb.TagNumber(1)
  void clearTerminal() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginTerminalRef ensureTerminal() => $_ensure(0);
}

class PluginUiNotification extends $pb.GeneratedMessage {
  factory PluginUiNotification({
    $core.String? title,
    $core.String? body,
    $core.String? severity,
  }) {
    final result = create();
    if (title != null) result.title = title;
    if (body != null) result.body = body;
    if (severity != null) result.severity = severity;
    return result;
  }

  PluginUiNotification._();

  factory PluginUiNotification.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiNotification.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiNotification',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'title')
    ..aOS(2, _omitFieldNames ? '' : 'body')
    ..aOS(3, _omitFieldNames ? '' : 'severity')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiNotification clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiNotification copyWith(void Function(PluginUiNotification) updates) =>
      super.copyWith((message) => updates(message as PluginUiNotification))
          as PluginUiNotification;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiNotification create() => PluginUiNotification._();
  @$core.override
  PluginUiNotification createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiNotification getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiNotification>(create);
  static PluginUiNotification? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get title => $_getSZ(0);
  @$pb.TagNumber(1)
  set title($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTitle() => $_has(0);
  @$pb.TagNumber(1)
  void clearTitle() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get body => $_getSZ(1);
  @$pb.TagNumber(2)
  set body($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBody() => $_has(1);
  @$pb.TagNumber(2)
  void clearBody() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get severity => $_getSZ(2);
  @$pb.TagNumber(3)
  set severity($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSeverity() => $_has(2);
  @$pb.TagNumber(3)
  void clearSeverity() => $_clearField(3);
}

enum PluginUiOperation_Operation { bind, notification, notSet }

class PluginUiOperation extends $pb.GeneratedMessage {
  factory PluginUiOperation({
    PluginTargetContext? context,
    PluginPaneBind? bind,
    PluginUiNotification? notification,
  }) {
    final result = create();
    if (context != null) result.context = context;
    if (bind != null) result.bind = bind;
    if (notification != null) result.notification = notification;
    return result;
  }

  PluginUiOperation._();

  factory PluginUiOperation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiOperation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginUiOperation_Operation>
      _PluginUiOperation_OperationByTag = {
    10: PluginUiOperation_Operation.bind,
    11: PluginUiOperation_Operation.notification,
    0: PluginUiOperation_Operation.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiOperation',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11])
    ..aOM<PluginTargetContext>(1, _omitFieldNames ? '' : 'context',
        subBuilder: PluginTargetContext.create)
    ..aOM<PluginPaneBind>(10, _omitFieldNames ? '' : 'bind',
        subBuilder: PluginPaneBind.create)
    ..aOM<PluginUiNotification>(11, _omitFieldNames ? '' : 'notification',
        subBuilder: PluginUiNotification.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiOperation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiOperation copyWith(void Function(PluginUiOperation) updates) =>
      super.copyWith((message) => updates(message as PluginUiOperation))
          as PluginUiOperation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiOperation create() => PluginUiOperation._();
  @$core.override
  PluginUiOperation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiOperation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiOperation>(create);
  static PluginUiOperation? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  PluginUiOperation_Operation whichOperation() =>
      _PluginUiOperation_OperationByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  void clearOperation() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PluginTargetContext get context => $_getN(0);
  @$pb.TagNumber(1)
  set context(PluginTargetContext value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasContext() => $_has(0);
  @$pb.TagNumber(1)
  void clearContext() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginTargetContext ensureContext() => $_ensure(0);

  @$pb.TagNumber(10)
  PluginPaneBind get bind => $_getN(1);
  @$pb.TagNumber(10)
  set bind(PluginPaneBind value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasBind() => $_has(1);
  @$pb.TagNumber(10)
  void clearBind() => $_clearField(10);
  @$pb.TagNumber(10)
  PluginPaneBind ensureBind() => $_ensure(1);

  @$pb.TagNumber(11)
  PluginUiNotification get notification => $_getN(2);
  @$pb.TagNumber(11)
  set notification(PluginUiNotification value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasNotification() => $_has(2);
  @$pb.TagNumber(11)
  void clearNotification() => $_clearField(11);
  @$pb.TagNumber(11)
  PluginUiNotification ensureNotification() => $_ensure(2);
}

class PluginUiAction extends $pb.GeneratedMessage {
  factory PluginUiAction({
    $core.String? id,
    $core.String? label,
    $core.String? defaultKey,
    $core.bool? enabled,
    $core.String? scope,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (label != null) result.label = label;
    if (defaultKey != null) result.defaultKey = defaultKey;
    if (enabled != null) result.enabled = enabled;
    if (scope != null) result.scope = scope;
    return result;
  }

  PluginUiAction._();

  factory PluginUiAction.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiAction.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiAction',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'label')
    ..aOS(3, _omitFieldNames ? '' : 'defaultKey')
    ..aOB(4, _omitFieldNames ? '' : 'enabled')
    ..aOS(5, _omitFieldNames ? '' : 'scope')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiAction clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiAction copyWith(void Function(PluginUiAction) updates) =>
      super.copyWith((message) => updates(message as PluginUiAction))
          as PluginUiAction;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiAction create() => PluginUiAction._();
  @$core.override
  PluginUiAction createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiAction getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiAction>(create);
  static PluginUiAction? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get label => $_getSZ(1);
  @$pb.TagNumber(2)
  set label($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLabel() => $_has(1);
  @$pb.TagNumber(2)
  void clearLabel() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get defaultKey => $_getSZ(2);
  @$pb.TagNumber(3)
  set defaultKey($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDefaultKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearDefaultKey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get enabled => $_getBF(3);
  @$pb.TagNumber(4)
  set enabled($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEnabled() => $_has(3);
  @$pb.TagNumber(4)
  void clearEnabled() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get scope => $_getSZ(4);
  @$pb.TagNumber(5)
  set scope($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasScope() => $_has(4);
  @$pb.TagNumber(5)
  void clearScope() => $_clearField(5);
}

/// Theme-aware semantic roles. The host maps these roles to its active theme;
/// plugins do not emit ANSI escape sequences or reach into host renderer code.
class PluginUiStyle extends $pb.GeneratedMessage {
  factory PluginUiStyle({
    $core.String? foregroundRole,
    $core.String? backgroundRole,
    $core.bool? bold,
    $core.bool? dim,
  }) {
    final result = create();
    if (foregroundRole != null) result.foregroundRole = foregroundRole;
    if (backgroundRole != null) result.backgroundRole = backgroundRole;
    if (bold != null) result.bold = bold;
    if (dim != null) result.dim = dim;
    return result;
  }

  PluginUiStyle._();

  factory PluginUiStyle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiStyle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiStyle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'foregroundRole')
    ..aOS(2, _omitFieldNames ? '' : 'backgroundRole')
    ..aOB(3, _omitFieldNames ? '' : 'bold')
    ..aOB(4, _omitFieldNames ? '' : 'dim')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiStyle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiStyle copyWith(void Function(PluginUiStyle) updates) =>
      super.copyWith((message) => updates(message as PluginUiStyle))
          as PluginUiStyle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiStyle create() => PluginUiStyle._();
  @$core.override
  PluginUiStyle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiStyle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiStyle>(create);
  static PluginUiStyle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get foregroundRole => $_getSZ(0);
  @$pb.TagNumber(1)
  set foregroundRole($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasForegroundRole() => $_has(0);
  @$pb.TagNumber(1)
  void clearForegroundRole() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get backgroundRole => $_getSZ(1);
  @$pb.TagNumber(2)
  set backgroundRole($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBackgroundRole() => $_has(1);
  @$pb.TagNumber(2)
  void clearBackgroundRole() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get bold => $_getBF(2);
  @$pb.TagNumber(3)
  set bold($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBold() => $_has(2);
  @$pb.TagNumber(3)
  void clearBold() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get dim => $_getBF(3);
  @$pb.TagNumber(4)
  set dim($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDim() => $_has(3);
  @$pb.TagNumber(4)
  void clearDim() => $_clearField(4);
}

class PluginUiLayout extends $pb.GeneratedMessage {
  factory PluginUiLayout({
    $core.int? paddingTop,
    $core.int? paddingRight,
    $core.int? paddingBottom,
    $core.int? paddingLeft,
    $core.int? gapAfter,
  }) {
    final result = create();
    if (paddingTop != null) result.paddingTop = paddingTop;
    if (paddingRight != null) result.paddingRight = paddingRight;
    if (paddingBottom != null) result.paddingBottom = paddingBottom;
    if (paddingLeft != null) result.paddingLeft = paddingLeft;
    if (gapAfter != null) result.gapAfter = gapAfter;
    return result;
  }

  PluginUiLayout._();

  factory PluginUiLayout.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiLayout.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiLayout',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'paddingTop', fieldType: $pb.PbFieldType.OU3)
    ..aI(2, _omitFieldNames ? '' : 'paddingRight',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'paddingBottom',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(4, _omitFieldNames ? '' : 'paddingLeft',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(5, _omitFieldNames ? '' : 'gapAfter', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiLayout clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiLayout copyWith(void Function(PluginUiLayout) updates) =>
      super.copyWith((message) => updates(message as PluginUiLayout))
          as PluginUiLayout;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiLayout create() => PluginUiLayout._();
  @$core.override
  PluginUiLayout createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiLayout getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiLayout>(create);
  static PluginUiLayout? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get paddingTop => $_getIZ(0);
  @$pb.TagNumber(1)
  set paddingTop($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPaddingTop() => $_has(0);
  @$pb.TagNumber(1)
  void clearPaddingTop() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get paddingRight => $_getIZ(1);
  @$pb.TagNumber(2)
  set paddingRight($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPaddingRight() => $_has(1);
  @$pb.TagNumber(2)
  void clearPaddingRight() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get paddingBottom => $_getIZ(2);
  @$pb.TagNumber(3)
  set paddingBottom($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPaddingBottom() => $_has(2);
  @$pb.TagNumber(3)
  void clearPaddingBottom() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get paddingLeft => $_getIZ(3);
  @$pb.TagNumber(4)
  set paddingLeft($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPaddingLeft() => $_has(3);
  @$pb.TagNumber(4)
  void clearPaddingLeft() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get gapAfter => $_getIZ(4);
  @$pb.TagNumber(5)
  set gapAfter($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasGapAfter() => $_has(4);
  @$pb.TagNumber(5)
  void clearGapAfter() => $_clearField(5);
}

class PluginUiNode extends $pb.GeneratedMessage {
  factory PluginUiNode({
    $core.String? id,
    $core.String? kind,
    $core.String? text,
    $core.String? status,
    $core.String? actionId,
    $core.String? itemId,
    $core.bool? disabled,
    $core.Iterable<PluginUiNode>? children,
    $core.double? progress,
    PluginTerminalRef? terminal,
    $core.String? value,
    $core.String? placeholder,
    PluginUiStyle? style,
    PluginUiStyle? selectedStyle,
    PluginUiLayout? layout,
    $core.String? description,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (kind != null) result.kind = kind;
    if (text != null) result.text = text;
    if (status != null) result.status = status;
    if (actionId != null) result.actionId = actionId;
    if (itemId != null) result.itemId = itemId;
    if (disabled != null) result.disabled = disabled;
    if (children != null) result.children.addAll(children);
    if (progress != null) result.progress = progress;
    if (terminal != null) result.terminal = terminal;
    if (value != null) result.value = value;
    if (placeholder != null) result.placeholder = placeholder;
    if (style != null) result.style = style;
    if (selectedStyle != null) result.selectedStyle = selectedStyle;
    if (layout != null) result.layout = layout;
    if (description != null) result.description = description;
    return result;
  }

  PluginUiNode._();

  factory PluginUiNode.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiNode.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiNode',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'kind')
    ..aOS(3, _omitFieldNames ? '' : 'text')
    ..aOS(4, _omitFieldNames ? '' : 'status')
    ..aOS(5, _omitFieldNames ? '' : 'actionId')
    ..aOS(6, _omitFieldNames ? '' : 'itemId')
    ..aOB(7, _omitFieldNames ? '' : 'disabled')
    ..pPM<PluginUiNode>(8, _omitFieldNames ? '' : 'children',
        subBuilder: PluginUiNode.create)
    ..aD(9, _omitFieldNames ? '' : 'progress')
    ..aOM<PluginTerminalRef>(10, _omitFieldNames ? '' : 'terminal',
        subBuilder: PluginTerminalRef.create)
    ..aOS(11, _omitFieldNames ? '' : 'value')
    ..aOS(12, _omitFieldNames ? '' : 'placeholder')
    ..aOM<PluginUiStyle>(13, _omitFieldNames ? '' : 'style',
        subBuilder: PluginUiStyle.create)
    ..aOM<PluginUiStyle>(14, _omitFieldNames ? '' : 'selectedStyle',
        subBuilder: PluginUiStyle.create)
    ..aOM<PluginUiLayout>(15, _omitFieldNames ? '' : 'layout',
        subBuilder: PluginUiLayout.create)
    ..aOS(16, _omitFieldNames ? '' : 'description')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiNode clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiNode copyWith(void Function(PluginUiNode) updates) =>
      super.copyWith((message) => updates(message as PluginUiNode))
          as PluginUiNode;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiNode create() => PluginUiNode._();
  @$core.override
  PluginUiNode createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiNode getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiNode>(create);
  static PluginUiNode? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  /// text, badge, progress, button, list, row, card, gap, column, table, tree, form.
  @$pb.TagNumber(2)
  $core.String get kind => $_getSZ(1);
  @$pb.TagNumber(2)
  set kind($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get text => $_getSZ(2);
  @$pb.TagNumber(3)
  set text($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasText() => $_has(2);
  @$pb.TagNumber(3)
  void clearText() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get status => $_getSZ(3);
  @$pb.TagNumber(4)
  set status($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get actionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set actionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasActionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearActionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get itemId => $_getSZ(5);
  @$pb.TagNumber(6)
  set itemId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasItemId() => $_has(5);
  @$pb.TagNumber(6)
  void clearItemId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get disabled => $_getBF(6);
  @$pb.TagNumber(7)
  set disabled($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasDisabled() => $_has(6);
  @$pb.TagNumber(7)
  void clearDisabled() => $_clearField(7);

  @$pb.TagNumber(8)
  $pb.PbList<PluginUiNode> get children => $_getList(7);

  @$pb.TagNumber(9)
  $core.double get progress => $_getN(8);
  @$pb.TagNumber(9)
  set progress($core.double value) => $_setDouble(8, value);
  @$pb.TagNumber(9)
  $core.bool hasProgress() => $_has(8);
  @$pb.TagNumber(9)
  void clearProgress() => $_clearField(9);

  @$pb.TagNumber(10)
  PluginTerminalRef get terminal => $_getN(9);
  @$pb.TagNumber(10)
  set terminal(PluginTerminalRef value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasTerminal() => $_has(9);
  @$pb.TagNumber(10)
  void clearTerminal() => $_clearField(10);
  @$pb.TagNumber(10)
  PluginTerminalRef ensureTerminal() => $_ensure(9);

  @$pb.TagNumber(11)
  $core.String get value => $_getSZ(10);
  @$pb.TagNumber(11)
  set value($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasValue() => $_has(10);
  @$pb.TagNumber(11)
  void clearValue() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get placeholder => $_getSZ(11);
  @$pb.TagNumber(12)
  set placeholder($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasPlaceholder() => $_has(11);
  @$pb.TagNumber(12)
  void clearPlaceholder() => $_clearField(12);

  @$pb.TagNumber(13)
  PluginUiStyle get style => $_getN(12);
  @$pb.TagNumber(13)
  set style(PluginUiStyle value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasStyle() => $_has(12);
  @$pb.TagNumber(13)
  void clearStyle() => $_clearField(13);
  @$pb.TagNumber(13)
  PluginUiStyle ensureStyle() => $_ensure(12);

  @$pb.TagNumber(14)
  PluginUiStyle get selectedStyle => $_getN(13);
  @$pb.TagNumber(14)
  set selectedStyle(PluginUiStyle value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasSelectedStyle() => $_has(13);
  @$pb.TagNumber(14)
  void clearSelectedStyle() => $_clearField(14);
  @$pb.TagNumber(14)
  PluginUiStyle ensureSelectedStyle() => $_ensure(13);

  @$pb.TagNumber(15)
  PluginUiLayout get layout => $_getN(14);
  @$pb.TagNumber(15)
  set layout(PluginUiLayout value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasLayout() => $_has(14);
  @$pb.TagNumber(15)
  void clearLayout() => $_clearField(15);
  @$pb.TagNumber(15)
  PluginUiLayout ensureLayout() => $_ensure(14);

  @$pb.TagNumber(16)
  $core.String get description => $_getSZ(15);
  @$pb.TagNumber(16)
  set description($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasDescription() => $_has(15);
  @$pb.TagNumber(16)
  void clearDescription() => $_clearField(16);
}

class PluginUiMountUpdate extends $pb.GeneratedMessage {
  factory PluginUiMountUpdate({
    $core.String? mountId,
    PluginMountOwner? owner,
    $core.String? slot,
    $fixnum.Int64? expectedRevision,
    $fixnum.Int64? revision,
    $core.String? title,
    PluginUiNode? root,
    $core.Iterable<PluginUiAction>? actions,
    $core.bool? close,
    $core.bool? focus,
    $core.int? preferredWidth,
    $core.int? minWidth,
  }) {
    final result = create();
    if (mountId != null) result.mountId = mountId;
    if (owner != null) result.owner = owner;
    if (slot != null) result.slot = slot;
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (revision != null) result.revision = revision;
    if (title != null) result.title = title;
    if (root != null) result.root = root;
    if (actions != null) result.actions.addAll(actions);
    if (close != null) result.close = close;
    if (focus != null) result.focus = focus;
    if (preferredWidth != null) result.preferredWidth = preferredWidth;
    if (minWidth != null) result.minWidth = minWidth;
    return result;
  }

  PluginUiMountUpdate._();

  factory PluginUiMountUpdate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiMountUpdate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiMountUpdate',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'mountId')
    ..aOM<PluginMountOwner>(2, _omitFieldNames ? '' : 'owner',
        subBuilder: PluginMountOwner.create)
    ..aOS(3, _omitFieldNames ? '' : 'slot')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'title')
    ..aOM<PluginUiNode>(7, _omitFieldNames ? '' : 'root',
        subBuilder: PluginUiNode.create)
    ..pPM<PluginUiAction>(8, _omitFieldNames ? '' : 'actions',
        subBuilder: PluginUiAction.create)
    ..aOB(9, _omitFieldNames ? '' : 'close')
    ..aOB(10, _omitFieldNames ? '' : 'focus')
    ..aI(11, _omitFieldNames ? '' : 'preferredWidth',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(12, _omitFieldNames ? '' : 'minWidth', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiMountUpdate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiMountUpdate copyWith(void Function(PluginUiMountUpdate) updates) =>
      super.copyWith((message) => updates(message as PluginUiMountUpdate))
          as PluginUiMountUpdate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiMountUpdate create() => PluginUiMountUpdate._();
  @$core.override
  PluginUiMountUpdate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiMountUpdate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiMountUpdate>(create);
  static PluginUiMountUpdate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get mountId => $_getSZ(0);
  @$pb.TagNumber(1)
  set mountId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMountId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMountId() => $_clearField(1);

  @$pb.TagNumber(2)
  PluginMountOwner get owner => $_getN(1);
  @$pb.TagNumber(2)
  set owner(PluginMountOwner value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasOwner() => $_has(1);
  @$pb.TagNumber(2)
  void clearOwner() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginMountOwner ensureOwner() => $_ensure(1);

  /// Owner is structural; slot is placement within that owner.
  @$pb.TagNumber(3)
  $core.String get slot => $_getSZ(2);
  @$pb.TagNumber(3)
  set slot($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSlot() => $_has(2);
  @$pb.TagNumber(3)
  void clearSlot() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get expectedRevision => $_getI64(3);
  @$pb.TagNumber(4)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasExpectedRevision() => $_has(3);
  @$pb.TagNumber(4)
  void clearExpectedRevision() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get revision => $_getI64(4);
  @$pb.TagNumber(5)
  set revision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRevision() => $_has(4);
  @$pb.TagNumber(5)
  void clearRevision() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get title => $_getSZ(5);
  @$pb.TagNumber(6)
  set title($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasTitle() => $_has(5);
  @$pb.TagNumber(6)
  void clearTitle() => $_clearField(6);

  @$pb.TagNumber(7)
  PluginUiNode get root => $_getN(6);
  @$pb.TagNumber(7)
  set root(PluginUiNode value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasRoot() => $_has(6);
  @$pb.TagNumber(7)
  void clearRoot() => $_clearField(7);
  @$pb.TagNumber(7)
  PluginUiNode ensureRoot() => $_ensure(6);

  @$pb.TagNumber(8)
  $pb.PbList<PluginUiAction> get actions => $_getList(7);

  @$pb.TagNumber(9)
  $core.bool get close => $_getBF(8);
  @$pb.TagNumber(9)
  set close($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasClose() => $_has(8);
  @$pb.TagNumber(9)
  void clearClose() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get focus => $_getBF(9);
  @$pb.TagNumber(10)
  set focus($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasFocus() => $_has(9);
  @$pb.TagNumber(10)
  void clearFocus() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get preferredWidth => $_getIZ(10);
  @$pb.TagNumber(11)
  set preferredWidth($core.int value) => $_setUnsignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasPreferredWidth() => $_has(10);
  @$pb.TagNumber(11)
  void clearPreferredWidth() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.int get minWidth => $_getIZ(11);
  @$pb.TagNumber(12)
  set minWidth($core.int value) => $_setUnsignedInt32(11, value);
  @$pb.TagNumber(12)
  $core.bool hasMinWidth() => $_has(11);
  @$pb.TagNumber(12)
  void clearMinWidth() => $_clearField(12);
}

class PluginAgentReport extends $pb.GeneratedMessage {
  factory PluginAgentReport({
    $core.String? agentId,
    $core.String? provider,
    $core.String? sessionId,
    $fixnum.Int64? sequence,
    $core.String? state,
    $core.String? title,
    $core.String? cwd,
    PluginTerminalRef? terminal,
    $fixnum.Int64? observedUnixMillis,
    $core.String? event,
    $core.String? detail,
    $fixnum.Int64? sourceEpoch,
    $core.String? permissionId,
    $core.String? turnId,
    $core.Iterable<$core.String>? pendingPermissionIds,
    $core.String? baseState,
    $core.bool? fullState,
    $core.bool? stale,
  }) {
    final result = create();
    if (agentId != null) result.agentId = agentId;
    if (provider != null) result.provider = provider;
    if (sessionId != null) result.sessionId = sessionId;
    if (sequence != null) result.sequence = sequence;
    if (state != null) result.state = state;
    if (title != null) result.title = title;
    if (cwd != null) result.cwd = cwd;
    if (terminal != null) result.terminal = terminal;
    if (observedUnixMillis != null)
      result.observedUnixMillis = observedUnixMillis;
    if (event != null) result.event = event;
    if (detail != null) result.detail = detail;
    if (sourceEpoch != null) result.sourceEpoch = sourceEpoch;
    if (permissionId != null) result.permissionId = permissionId;
    if (turnId != null) result.turnId = turnId;
    if (pendingPermissionIds != null)
      result.pendingPermissionIds.addAll(pendingPermissionIds);
    if (baseState != null) result.baseState = baseState;
    if (fullState != null) result.fullState = fullState;
    if (stale != null) result.stale = stale;
    return result;
  }

  PluginAgentReport._();

  factory PluginAgentReport.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginAgentReport.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginAgentReport',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'agentId')
    ..aOS(2, _omitFieldNames ? '' : 'provider')
    ..aOS(3, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'sequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(5, _omitFieldNames ? '' : 'state')
    ..aOS(6, _omitFieldNames ? '' : 'title')
    ..aOS(7, _omitFieldNames ? '' : 'cwd')
    ..aOM<PluginTerminalRef>(8, _omitFieldNames ? '' : 'terminal',
        subBuilder: PluginTerminalRef.create)
    ..aInt64(9, _omitFieldNames ? '' : 'observedUnixMillis')
    ..aOS(10, _omitFieldNames ? '' : 'event')
    ..aOS(11, _omitFieldNames ? '' : 'detail')
    ..a<$fixnum.Int64>(
        12, _omitFieldNames ? '' : 'sourceEpoch', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(13, _omitFieldNames ? '' : 'permissionId')
    ..aOS(14, _omitFieldNames ? '' : 'turnId')
    ..pPS(15, _omitFieldNames ? '' : 'pendingPermissionIds')
    ..aOS(16, _omitFieldNames ? '' : 'baseState')
    ..aOB(17, _omitFieldNames ? '' : 'fullState')
    ..aOB(18, _omitFieldNames ? '' : 'stale')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAgentReport clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAgentReport copyWith(void Function(PluginAgentReport) updates) =>
      super.copyWith((message) => updates(message as PluginAgentReport))
          as PluginAgentReport;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginAgentReport create() => PluginAgentReport._();
  @$core.override
  PluginAgentReport createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginAgentReport getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginAgentReport>(create);
  static PluginAgentReport? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get agentId => $_getSZ(0);
  @$pb.TagNumber(1)
  set agentId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAgentId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAgentId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get provider => $_getSZ(1);
  @$pb.TagNumber(2)
  set provider($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasProvider() => $_has(1);
  @$pb.TagNumber(2)
  void clearProvider() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set sessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get sequence => $_getI64(3);
  @$pb.TagNumber(4)
  set sequence($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSequence() => $_has(3);
  @$pb.TagNumber(4)
  void clearSequence() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get state => $_getSZ(4);
  @$pb.TagNumber(5)
  set state($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasState() => $_has(4);
  @$pb.TagNumber(5)
  void clearState() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get title => $_getSZ(5);
  @$pb.TagNumber(6)
  set title($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasTitle() => $_has(5);
  @$pb.TagNumber(6)
  void clearTitle() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get cwd => $_getSZ(6);
  @$pb.TagNumber(7)
  set cwd($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCwd() => $_has(6);
  @$pb.TagNumber(7)
  void clearCwd() => $_clearField(7);

  @$pb.TagNumber(8)
  PluginTerminalRef get terminal => $_getN(7);
  @$pb.TagNumber(8)
  set terminal(PluginTerminalRef value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasTerminal() => $_has(7);
  @$pb.TagNumber(8)
  void clearTerminal() => $_clearField(8);
  @$pb.TagNumber(8)
  PluginTerminalRef ensureTerminal() => $_ensure(7);

  @$pb.TagNumber(9)
  $fixnum.Int64 get observedUnixMillis => $_getI64(8);
  @$pb.TagNumber(9)
  set observedUnixMillis($fixnum.Int64 value) => $_setInt64(8, value);
  @$pb.TagNumber(9)
  $core.bool hasObservedUnixMillis() => $_has(8);
  @$pb.TagNumber(9)
  void clearObservedUnixMillis() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get event => $_getSZ(9);
  @$pb.TagNumber(10)
  set event($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasEvent() => $_has(9);
  @$pb.TagNumber(10)
  void clearEvent() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get detail => $_getSZ(10);
  @$pb.TagNumber(11)
  set detail($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasDetail() => $_has(10);
  @$pb.TagNumber(11)
  void clearDetail() => $_clearField(11);

  @$pb.TagNumber(12)
  $fixnum.Int64 get sourceEpoch => $_getI64(11);
  @$pb.TagNumber(12)
  set sourceEpoch($fixnum.Int64 value) => $_setInt64(11, value);
  @$pb.TagNumber(12)
  $core.bool hasSourceEpoch() => $_has(11);
  @$pb.TagNumber(12)
  void clearSourceEpoch() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get permissionId => $_getSZ(12);
  @$pb.TagNumber(13)
  set permissionId($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasPermissionId() => $_has(12);
  @$pb.TagNumber(13)
  void clearPermissionId() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get turnId => $_getSZ(13);
  @$pb.TagNumber(14)
  set turnId($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasTurnId() => $_has(13);
  @$pb.TagNumber(14)
  void clearTurnId() => $_clearField(14);

  @$pb.TagNumber(15)
  $pb.PbList<$core.String> get pendingPermissionIds => $_getList(14);

  @$pb.TagNumber(16)
  $core.String get baseState => $_getSZ(15);
  @$pb.TagNumber(16)
  set baseState($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasBaseState() => $_has(15);
  @$pb.TagNumber(16)
  void clearBaseState() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.bool get fullState => $_getBF(16);
  @$pb.TagNumber(17)
  set fullState($core.bool value) => $_setBool(16, value);
  @$pb.TagNumber(17)
  $core.bool hasFullState() => $_has(16);
  @$pb.TagNumber(17)
  void clearFullState() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.bool get stale => $_getBF(17);
  @$pb.TagNumber(18)
  set stale($core.bool value) => $_setBool(17, value);
  @$pb.TagNumber(18)
  $core.bool hasStale() => $_has(17);
  @$pb.TagNumber(18)
  void clearStale() => $_clearField(18);
}

class PluginAgentSnapshot extends $pb.GeneratedMessage {
  factory PluginAgentSnapshot({
    $core.Iterable<PluginAgentReport>? agents,
    $fixnum.Int64? revision,
  }) {
    final result = create();
    if (agents != null) result.agents.addAll(agents);
    if (revision != null) result.revision = revision;
    return result;
  }

  PluginAgentSnapshot._();

  factory PluginAgentSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginAgentSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginAgentSnapshot',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pPM<PluginAgentReport>(1, _omitFieldNames ? '' : 'agents',
        subBuilder: PluginAgentReport.create)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAgentSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginAgentSnapshot copyWith(void Function(PluginAgentSnapshot) updates) =>
      super.copyWith((message) => updates(message as PluginAgentSnapshot))
          as PluginAgentSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginAgentSnapshot create() => PluginAgentSnapshot._();
  @$core.override
  PluginAgentSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginAgentSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginAgentSnapshot>(create);
  static PluginAgentSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<PluginAgentReport> get agents => $_getList(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get revision => $_getI64(1);
  @$pb.TagNumber(2)
  set revision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);
}

enum PluginStateRequest_Operation { get, put, watch, notSet }

/// State changes are full versioned snapshots, so slow consumers can resnapshot.
/// Collection access is isolated by the registration's plugin namespace.
class PluginStateRequest extends $pb.GeneratedMessage {
  factory PluginStateRequest({
    $core.List<$core.int>? sourceLease,
    $core.String? collection,
    PluginStateGet? get,
    PluginStatePut? put,
    PluginStateWatch? watch,
  }) {
    final result = create();
    if (sourceLease != null) result.sourceLease = sourceLease;
    if (collection != null) result.collection = collection;
    if (get != null) result.get = get;
    if (put != null) result.put = put;
    if (watch != null) result.watch = watch;
    return result;
  }

  PluginStateRequest._();

  factory PluginStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginStateRequest_Operation>
      _PluginStateRequest_OperationByTag = {
    10: PluginStateRequest_Operation.get,
    11: PluginStateRequest_Operation.put,
    12: PluginStateRequest_Operation.watch,
    0: PluginStateRequest_Operation.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginStateRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 12])
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'sourceLease', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'collection')
    ..aOM<PluginStateGet>(10, _omitFieldNames ? '' : 'get',
        subBuilder: PluginStateGet.create)
    ..aOM<PluginStatePut>(11, _omitFieldNames ? '' : 'put',
        subBuilder: PluginStatePut.create)
    ..aOM<PluginStateWatch>(12, _omitFieldNames ? '' : 'watch',
        subBuilder: PluginStateWatch.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateRequest copyWith(void Function(PluginStateRequest) updates) =>
      super.copyWith((message) => updates(message as PluginStateRequest))
          as PluginStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStateRequest create() => PluginStateRequest._();
  @$core.override
  PluginStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginStateRequest>(create);
  static PluginStateRequest? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  PluginStateRequest_Operation whichOperation() =>
      _PluginStateRequest_OperationByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  void clearOperation() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.List<$core.int> get sourceLease => $_getN(0);
  @$pb.TagNumber(1)
  set sourceLease($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSourceLease() => $_has(0);
  @$pb.TagNumber(1)
  void clearSourceLease() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get collection => $_getSZ(1);
  @$pb.TagNumber(2)
  set collection($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCollection() => $_has(1);
  @$pb.TagNumber(2)
  void clearCollection() => $_clearField(2);

  @$pb.TagNumber(10)
  PluginStateGet get get => $_getN(2);
  @$pb.TagNumber(10)
  set get(PluginStateGet value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasGet() => $_has(2);
  @$pb.TagNumber(10)
  void clearGet() => $_clearField(10);
  @$pb.TagNumber(10)
  PluginStateGet ensureGet() => $_ensure(2);

  @$pb.TagNumber(11)
  PluginStatePut get put => $_getN(3);
  @$pb.TagNumber(11)
  set put(PluginStatePut value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasPut() => $_has(3);
  @$pb.TagNumber(11)
  void clearPut() => $_clearField(11);
  @$pb.TagNumber(11)
  PluginStatePut ensurePut() => $_ensure(3);

  @$pb.TagNumber(12)
  PluginStateWatch get watch => $_getN(4);
  @$pb.TagNumber(12)
  set watch(PluginStateWatch value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasWatch() => $_has(4);
  @$pb.TagNumber(12)
  void clearWatch() => $_clearField(12);
  @$pb.TagNumber(12)
  PluginStateWatch ensureWatch() => $_ensure(4);
}

class PluginStateGet extends $pb.GeneratedMessage {
  factory PluginStateGet() => create();

  PluginStateGet._();

  factory PluginStateGet.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginStateGet.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginStateGet',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateGet clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateGet copyWith(void Function(PluginStateGet) updates) =>
      super.copyWith((message) => updates(message as PluginStateGet))
          as PluginStateGet;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStateGet create() => PluginStateGet._();
  @$core.override
  PluginStateGet createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginStateGet getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginStateGet>(create);
  static PluginStateGet? _defaultInstance;
}

class PluginStatePut extends $pb.GeneratedMessage {
  factory PluginStatePut({
    $fixnum.Int64? expectedRevision,
    PluginPayload? value,
  }) {
    final result = create();
    if (expectedRevision != null) result.expectedRevision = expectedRevision;
    if (value != null) result.value = value;
    return result;
  }

  PluginStatePut._();

  factory PluginStatePut.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginStatePut.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginStatePut',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'expectedRevision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<PluginPayload>(2, _omitFieldNames ? '' : 'value',
        subBuilder: PluginPayload.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStatePut clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStatePut copyWith(void Function(PluginStatePut) updates) =>
      super.copyWith((message) => updates(message as PluginStatePut))
          as PluginStatePut;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStatePut create() => PluginStatePut._();
  @$core.override
  PluginStatePut createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginStatePut getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginStatePut>(create);
  static PluginStatePut? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get expectedRevision => $_getI64(0);
  @$pb.TagNumber(1)
  set expectedRevision($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExpectedRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearExpectedRevision() => $_clearField(1);

  @$pb.TagNumber(2)
  PluginPayload get value => $_getN(1);
  @$pb.TagNumber(2)
  set value(PluginPayload value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(2)
  void clearValue() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginPayload ensureValue() => $_ensure(1);
}

class PluginStateWatch extends $pb.GeneratedMessage {
  factory PluginStateWatch({
    $core.bool? cancel,
  }) {
    final result = create();
    if (cancel != null) result.cancel = cancel;
    return result;
  }

  PluginStateWatch._();

  factory PluginStateWatch.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginStateWatch.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginStateWatch',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'cancel')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateWatch clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateWatch copyWith(void Function(PluginStateWatch) updates) =>
      super.copyWith((message) => updates(message as PluginStateWatch))
          as PluginStateWatch;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStateWatch create() => PluginStateWatch._();
  @$core.override
  PluginStateWatch createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginStateWatch getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginStateWatch>(create);
  static PluginStateWatch? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get cancel => $_getBF(0);
  @$pb.TagNumber(1)
  set cancel($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCancel() => $_has(0);
  @$pb.TagNumber(1)
  void clearCancel() => $_clearField(1);
}

class PluginStateSnapshot extends $pb.GeneratedMessage {
  factory PluginStateSnapshot({
    $core.String? collection,
    $fixnum.Int64? revision,
    PluginPayload? value,
    $core.String? bootEpoch,
  }) {
    final result = create();
    if (collection != null) result.collection = collection;
    if (revision != null) result.revision = revision;
    if (value != null) result.value = value;
    if (bootEpoch != null) result.bootEpoch = bootEpoch;
    return result;
  }

  PluginStateSnapshot._();

  factory PluginStateSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginStateSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginStateSnapshot',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'collection')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<PluginPayload>(3, _omitFieldNames ? '' : 'value',
        subBuilder: PluginPayload.create)
    ..aOS(4, _omitFieldNames ? '' : 'bootEpoch')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginStateSnapshot copyWith(void Function(PluginStateSnapshot) updates) =>
      super.copyWith((message) => updates(message as PluginStateSnapshot))
          as PluginStateSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginStateSnapshot create() => PluginStateSnapshot._();
  @$core.override
  PluginStateSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginStateSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginStateSnapshot>(create);
  static PluginStateSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get collection => $_getSZ(0);
  @$pb.TagNumber(1)
  set collection($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCollection() => $_has(0);
  @$pb.TagNumber(1)
  void clearCollection() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get revision => $_getI64(1);
  @$pb.TagNumber(2)
  set revision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  PluginPayload get value => $_getN(2);
  @$pb.TagNumber(3)
  set value(PluginPayload value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasValue() => $_has(2);
  @$pb.TagNumber(3)
  void clearValue() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginPayload ensureValue() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get bootEpoch => $_getSZ(3);
  @$pb.TagNumber(4)
  set bootEpoch($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBootEpoch() => $_has(3);
  @$pb.TagNumber(4)
  void clearBootEpoch() => $_clearField(4);
}

enum PluginBridgeFrame_Payload { command, result, cancelRequestId, notSet }

/// Length-delimited local SDK transport; host forwards every command to daemon.
class PluginBridgeFrame extends $pb.GeneratedMessage {
  factory PluginBridgeFrame({
    $core.String? requestId,
    PluginCommand? command,
    PluginResult? result,
    $core.String? viaEndpointId,
    $fixnum.Int64? deadlineUnixMillis,
    $core.String? cancelRequestId,
  }) {
    final result$ = create();
    if (requestId != null) result$.requestId = requestId;
    if (command != null) result$.command = command;
    if (result != null) result$.result = result;
    if (viaEndpointId != null) result$.viaEndpointId = viaEndpointId;
    if (deadlineUnixMillis != null)
      result$.deadlineUnixMillis = deadlineUnixMillis;
    if (cancelRequestId != null) result$.cancelRequestId = cancelRequestId;
    return result$;
  }

  PluginBridgeFrame._();

  factory PluginBridgeFrame.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginBridgeFrame.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginBridgeFrame_Payload>
      _PluginBridgeFrame_PayloadByTag = {
    2: PluginBridgeFrame_Payload.command,
    3: PluginBridgeFrame_Payload.result,
    6: PluginBridgeFrame_Payload.cancelRequestId,
    0: PluginBridgeFrame_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginBridgeFrame',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [2, 3, 6])
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<PluginCommand>(2, _omitFieldNames ? '' : 'command',
        subBuilder: PluginCommand.create)
    ..aOM<PluginResult>(3, _omitFieldNames ? '' : 'result',
        subBuilder: PluginResult.create)
    ..aOS(4, _omitFieldNames ? '' : 'viaEndpointId')
    ..aInt64(5, _omitFieldNames ? '' : 'deadlineUnixMillis')
    ..aOS(6, _omitFieldNames ? '' : 'cancelRequestId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginBridgeFrame clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginBridgeFrame copyWith(void Function(PluginBridgeFrame) updates) =>
      super.copyWith((message) => updates(message as PluginBridgeFrame))
          as PluginBridgeFrame;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginBridgeFrame create() => PluginBridgeFrame._();
  @$core.override
  PluginBridgeFrame createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginBridgeFrame getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginBridgeFrame>(create);
  static PluginBridgeFrame? _defaultInstance;

  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(6)
  PluginBridgeFrame_Payload whichPayload() =>
      _PluginBridgeFrame_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(6)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  PluginCommand get command => $_getN(1);
  @$pb.TagNumber(2)
  set command(PluginCommand value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCommand() => $_has(1);
  @$pb.TagNumber(2)
  void clearCommand() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginCommand ensureCommand() => $_ensure(1);

  @$pb.TagNumber(3)
  PluginResult get result => $_getN(2);
  @$pb.TagNumber(3)
  set result(PluginResult value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasResult() => $_has(2);
  @$pb.TagNumber(3)
  void clearResult() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginResult ensureResult() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get viaEndpointId => $_getSZ(3);
  @$pb.TagNumber(4)
  set viaEndpointId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasViaEndpointId() => $_has(3);
  @$pb.TagNumber(4)
  void clearViaEndpointId() => $_clearField(4);

  /// Local host bridge deadline; zero is a 30-second compatibility default.
  @$pb.TagNumber(5)
  $fixnum.Int64 get deadlineUnixMillis => $_getI64(4);
  @$pb.TagNumber(5)
  set deadlineUnixMillis($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDeadlineUnixMillis() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeadlineUnixMillis() => $_clearField(5);

  /// Cancels the matching in-flight bridge request without consuming a worker.
  @$pb.TagNumber(6)
  $core.String get cancelRequestId => $_getSZ(5);
  @$pb.TagNumber(6)
  set cancelRequestId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCancelRequestId() => $_has(5);
  @$pb.TagNumber(6)
  void clearCancelRequestId() => $_clearField(6);
}

class PluginUiInit extends $pb.GeneratedMessage {
  factory PluginUiInit({
    $core.Iterable<PluginMountOwner>? owners,
    PluginAddress? host,
    PluginTargetContext? context,
    $core.Iterable<$core.MapEntry<$core.String, $fixnum.Int64>>? mountRevisions,
    $core.String? reconcileRequestId,
  }) {
    final result = create();
    if (owners != null) result.owners.addAll(owners);
    if (host != null) result.host = host;
    if (context != null) result.context = context;
    if (mountRevisions != null)
      result.mountRevisions.addEntries(mountRevisions);
    if (reconcileRequestId != null)
      result.reconcileRequestId = reconcileRequestId;
    return result;
  }

  PluginUiInit._();

  factory PluginUiInit.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiInit.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiInit',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..pPM<PluginMountOwner>(1, _omitFieldNames ? '' : 'owners',
        subBuilder: PluginMountOwner.create)
    ..aOM<PluginAddress>(2, _omitFieldNames ? '' : 'host',
        subBuilder: PluginAddress.create)
    ..aOM<PluginTargetContext>(3, _omitFieldNames ? '' : 'context',
        subBuilder: PluginTargetContext.create)
    ..m<$core.String, $fixnum.Int64>(4, _omitFieldNames ? '' : 'mountRevisions',
        entryClassName: 'PluginUiInit.MountRevisionsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OU6,
        packageName: const $pb.PackageName('anytty.api.v1'))
    ..aOS(5, _omitFieldNames ? '' : 'reconcileRequestId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiInit clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiInit copyWith(void Function(PluginUiInit) updates) =>
      super.copyWith((message) => updates(message as PluginUiInit))
          as PluginUiInit;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiInit create() => PluginUiInit._();
  @$core.override
  PluginUiInit createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiInit getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiInit>(create);
  static PluginUiInit? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<PluginMountOwner> get owners => $_getList(0);

  @$pb.TagNumber(2)
  PluginAddress get host => $_getN(1);
  @$pb.TagNumber(2)
  set host(PluginAddress value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasHost() => $_has(1);
  @$pb.TagNumber(2)
  void clearHost() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginAddress ensureHost() => $_ensure(1);

  @$pb.TagNumber(3)
  PluginTargetContext get context => $_getN(2);
  @$pb.TagNumber(3)
  set context(PluginTargetContext value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasContext() => $_has(2);
  @$pb.TagNumber(3)
  void clearContext() => $_clearField(3);
  @$pb.TagNumber(3)
  PluginTargetContext ensureContext() => $_ensure(2);

  /// Host-authoritative revisions scoped to the requesting plugin and daemon.
  @$pb.TagNumber(4)
  $pb.PbMap<$core.String, $fixnum.Int64> get mountRevisions => $_getMap(3);

  /// Echoed by the host so late reconciliation responses cannot replace newer ACKs.
  @$pb.TagNumber(5)
  $core.String get reconcileRequestId => $_getSZ(4);
  @$pb.TagNumber(5)
  set reconcileRequestId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasReconcileRequestId() => $_has(4);
  @$pb.TagNumber(5)
  void clearReconcileRequestId() => $_clearField(5);
}

class PluginUiSnapshotQuery extends $pb.GeneratedMessage {
  factory PluginUiSnapshotQuery() => create();

  PluginUiSnapshotQuery._();

  factory PluginUiSnapshotQuery.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiSnapshotQuery.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiSnapshotQuery',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiSnapshotQuery clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiSnapshotQuery copyWith(
          void Function(PluginUiSnapshotQuery) updates) =>
      super.copyWith((message) => updates(message as PluginUiSnapshotQuery))
          as PluginUiSnapshotQuery;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiSnapshotQuery create() => PluginUiSnapshotQuery._();
  @$core.override
  PluginUiSnapshotQuery createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiSnapshotQuery getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiSnapshotQuery>(create);
  static PluginUiSnapshotQuery? _defaultInstance;
}

enum PluginUiQuery_Query { snapshot, notSet }

class PluginUiQuery extends $pb.GeneratedMessage {
  factory PluginUiQuery({
    PluginUiSnapshotQuery? snapshot,
  }) {
    final result = create();
    if (snapshot != null) result.snapshot = snapshot;
    return result;
  }

  PluginUiQuery._();

  factory PluginUiQuery.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiQuery.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PluginUiQuery_Query>
      _PluginUiQuery_QueryByTag = {
    1: PluginUiQuery_Query.snapshot,
    0: PluginUiQuery_Query.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiQuery',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [1])
    ..aOM<PluginUiSnapshotQuery>(1, _omitFieldNames ? '' : 'snapshot',
        subBuilder: PluginUiSnapshotQuery.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiQuery clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiQuery copyWith(void Function(PluginUiQuery) updates) =>
      super.copyWith((message) => updates(message as PluginUiQuery))
          as PluginUiQuery;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiQuery create() => PluginUiQuery._();
  @$core.override
  PluginUiQuery createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiQuery getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiQuery>(create);
  static PluginUiQuery? _defaultInstance;

  @$pb.TagNumber(1)
  PluginUiQuery_Query whichQuery() =>
      _PluginUiQuery_QueryByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  void clearQuery() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  PluginUiSnapshotQuery get snapshot => $_getN(0);
  @$pb.TagNumber(1)
  set snapshot(PluginUiSnapshotQuery value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasSnapshot() => $_has(0);
  @$pb.TagNumber(1)
  void clearSnapshot() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginUiSnapshotQuery ensureSnapshot() => $_ensure(0);
}

class PluginPanelSnapshot extends $pb.GeneratedMessage {
  factory PluginPanelSnapshot({
    PluginMountOwner? owner,
    PluginTerminalRef? terminal,
    $core.String? viewId,
    $core.bool? focused,
  }) {
    final result = create();
    if (owner != null) result.owner = owner;
    if (terminal != null) result.terminal = terminal;
    if (viewId != null) result.viewId = viewId;
    if (focused != null) result.focused = focused;
    return result;
  }

  PluginPanelSnapshot._();

  factory PluginPanelSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginPanelSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginPanelSnapshot',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<PluginMountOwner>(1, _omitFieldNames ? '' : 'owner',
        subBuilder: PluginMountOwner.create)
    ..aOM<PluginTerminalRef>(2, _omitFieldNames ? '' : 'terminal',
        subBuilder: PluginTerminalRef.create)
    ..aOS(3, _omitFieldNames ? '' : 'viewId')
    ..aOB(4, _omitFieldNames ? '' : 'focused')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPanelSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginPanelSnapshot copyWith(void Function(PluginPanelSnapshot) updates) =>
      super.copyWith((message) => updates(message as PluginPanelSnapshot))
          as PluginPanelSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginPanelSnapshot create() => PluginPanelSnapshot._();
  @$core.override
  PluginPanelSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginPanelSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginPanelSnapshot>(create);
  static PluginPanelSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  PluginMountOwner get owner => $_getN(0);
  @$pb.TagNumber(1)
  set owner(PluginMountOwner value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasOwner() => $_has(0);
  @$pb.TagNumber(1)
  void clearOwner() => $_clearField(1);
  @$pb.TagNumber(1)
  PluginMountOwner ensureOwner() => $_ensure(0);

  @$pb.TagNumber(2)
  PluginTerminalRef get terminal => $_getN(1);
  @$pb.TagNumber(2)
  set terminal(PluginTerminalRef value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTerminal() => $_has(1);
  @$pb.TagNumber(2)
  void clearTerminal() => $_clearField(2);
  @$pb.TagNumber(2)
  PluginTerminalRef ensureTerminal() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.String get viewId => $_getSZ(2);
  @$pb.TagNumber(3)
  set viewId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasViewId() => $_has(2);
  @$pb.TagNumber(3)
  void clearViewId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get focused => $_getBF(3);
  @$pb.TagNumber(4)
  set focused($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasFocused() => $_has(3);
  @$pb.TagNumber(4)
  void clearFocused() => $_clearField(4);
}

class PluginUiSnapshot extends $pb.GeneratedMessage {
  factory PluginUiSnapshot({
    $core.String? tuiInstanceId,
    $core.Iterable<PluginMountOwner>? owners,
    $core.Iterable<PluginPanelSnapshot>? panels,
    PluginTargetContext? activeContext,
    $fixnum.Int64? revision,
  }) {
    final result = create();
    if (tuiInstanceId != null) result.tuiInstanceId = tuiInstanceId;
    if (owners != null) result.owners.addAll(owners);
    if (panels != null) result.panels.addAll(panels);
    if (activeContext != null) result.activeContext = activeContext;
    if (revision != null) result.revision = revision;
    return result;
  }

  PluginUiSnapshot._();

  factory PluginUiSnapshot.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PluginUiSnapshot.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PluginUiSnapshot',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'tuiInstanceId')
    ..pPM<PluginMountOwner>(2, _omitFieldNames ? '' : 'owners',
        subBuilder: PluginMountOwner.create)
    ..pPM<PluginPanelSnapshot>(3, _omitFieldNames ? '' : 'panels',
        subBuilder: PluginPanelSnapshot.create)
    ..aOM<PluginTargetContext>(4, _omitFieldNames ? '' : 'activeContext',
        subBuilder: PluginTargetContext.create)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiSnapshot clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PluginUiSnapshot copyWith(void Function(PluginUiSnapshot) updates) =>
      super.copyWith((message) => updates(message as PluginUiSnapshot))
          as PluginUiSnapshot;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PluginUiSnapshot create() => PluginUiSnapshot._();
  @$core.override
  PluginUiSnapshot createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PluginUiSnapshot getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PluginUiSnapshot>(create);
  static PluginUiSnapshot? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get tuiInstanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set tuiInstanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTuiInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTuiInstanceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<PluginMountOwner> get owners => $_getList(1);

  @$pb.TagNumber(3)
  $pb.PbList<PluginPanelSnapshot> get panels => $_getList(2);

  @$pb.TagNumber(4)
  PluginTargetContext get activeContext => $_getN(3);
  @$pb.TagNumber(4)
  set activeContext(PluginTargetContext value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasActiveContext() => $_has(3);
  @$pb.TagNumber(4)
  void clearActiveContext() => $_clearField(4);
  @$pb.TagNumber(4)
  PluginTargetContext ensureActiveContext() => $_ensure(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get revision => $_getI64(4);
  @$pb.TagNumber(5)
  set revision($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRevision() => $_has(4);
  @$pb.TagNumber(5)
  void clearRevision() => $_clearField(5);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
