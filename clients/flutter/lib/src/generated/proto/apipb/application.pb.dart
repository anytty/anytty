// This is a generated file - do not edit.
//
// Generated from apipb/application.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'access_remote.pb.dart' as $6;
import 'common.pb.dart' as $0;
import 'events.pb.dart' as $3;
import 'file.pb.dart' as $4;
import 'history.pb.dart' as $2;
import 'storage.pb.dart' as $5;
import 'terminal.pb.dart' as $1;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class CancelOperationCommand extends $pb.GeneratedMessage {
  factory CancelOperationCommand({
    $0.OperationStamp? operation,
  }) {
    final result = create();
    if (operation != null) result.operation = operation;
    return result;
  }

  CancelOperationCommand._();

  factory CancelOperationCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CancelOperationCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CancelOperationCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.OperationStamp>(2, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CancelOperationCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CancelOperationCommand copyWith(
          void Function(CancelOperationCommand) updates) =>
      super.copyWith((message) => updates(message as CancelOperationCommand))
          as CancelOperationCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CancelOperationCommand create() => CancelOperationCommand._();
  @$core.override
  CancelOperationCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CancelOperationCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CancelOperationCommand>(create);
  static CancelOperationCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.OperationStamp get operation => $_getN(0);
  @$pb.TagNumber(2)
  set operation($0.OperationStamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasOperation() => $_has(0);
  @$pb.TagNumber(2)
  void clearOperation() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.OperationStamp ensureOperation() => $_ensure(0);
}

class ReleaseResourceCommand extends $pb.GeneratedMessage {
  factory ReleaseResourceCommand({
    $0.ResourceHandle? resource,
  }) {
    final result = create();
    if (resource != null) result.resource = resource;
    return result;
  }

  ReleaseResourceCommand._();

  factory ReleaseResourceCommand.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReleaseResourceCommand.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReleaseResourceCommand',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(2, _omitFieldNames ? '' : 'resource',
        subBuilder: $0.ResourceHandle.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReleaseResourceCommand clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReleaseResourceCommand copyWith(
          void Function(ReleaseResourceCommand) updates) =>
      super.copyWith((message) => updates(message as ReleaseResourceCommand))
          as ReleaseResourceCommand;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReleaseResourceCommand create() => ReleaseResourceCommand._();
  @$core.override
  ReleaseResourceCommand createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReleaseResourceCommand getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReleaseResourceCommand>(create);
  static ReleaseResourceCommand? _defaultInstance;

  @$pb.TagNumber(2)
  $0.ResourceHandle get resource => $_getN(0);
  @$pb.TagNumber(2)
  set resource($0.ResourceHandle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasResource() => $_has(0);
  @$pb.TagNumber(2)
  void clearResource() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.ResourceHandle ensureResource() => $_ensure(0);
}

enum CommandEnvelope_Command {
  cancelOperation,
  releaseResource,
  terminalDefaults,
  terminalCreate,
  terminalList,
  terminalGet,
  terminalRestart,
  terminalKill,
  terminalRemove,
  terminalSetMetadata,
  terminalSetTags,
  terminalAttach,
  terminalDetach,
  terminalInput,
  terminalResize,
  terminalResizeLock,
  pathListDirectories,
  historyWindow,
  historyCopy,
  historyRelease,
  historyBacklogStatus,
  liveScreenNext,
  historySearch,
  eventSubscribe,
  fileList,
  fileStat,
  filePreview,
  fileMkdir,
  fileRename,
  fileDelete,
  fileCopy,
  fileMove,
  fileDownloadOpen,
  fileUploadOpen,
  fileTransferCancel,
  storageGet,
  storagePut,
  storageDelete,
  storageList,
  clientAccessIdentity,
  clientAccessList,
  clientAccessTicketCreate,
  clientAccessRevoke,
  remoteStatus,
  remotePairStart,
  remoteLocalEnable,
  remoteLocalStatus,
  remoteLocalDisable,
  remoteCloudEdges,
  remoteCloudPreferEdge,
  remoteCloudReselectEdge,
  remoteCloudStatus,
  remoteCloudEnable,
  remoteCloudDisable,
  notSet
}

class CommandEnvelope extends $pb.GeneratedMessage {
  factory CommandEnvelope({
    $0.RequestContext? context,
    CancelOperationCommand? cancelOperation,
    ReleaseResourceCommand? releaseResource,
    $1.TerminalDefaultsCommand? terminalDefaults,
    $1.TerminalCreateCommand? terminalCreate,
    $1.TerminalListCommand? terminalList,
    $1.TerminalGetCommand? terminalGet,
    $1.TerminalRestartCommand? terminalRestart,
    $1.TerminalKillCommand? terminalKill,
    $1.TerminalRemoveCommand? terminalRemove,
    $1.TerminalSetMetadataCommand? terminalSetMetadata,
    $1.TerminalSetTagsCommand? terminalSetTags,
    $1.TerminalAttachCommand? terminalAttach,
    $1.TerminalDetachCommand? terminalDetach,
    $1.TerminalInputCommand? terminalInput,
    $1.TerminalResizeCommand? terminalResize,
    $1.TerminalResizeLockCommand? terminalResizeLock,
    $1.PathListDirectoriesCommand? pathListDirectories,
    $2.HistoryWindowCommand? historyWindow,
    $2.HistoryCopyCommand? historyCopy,
    $2.HistoryReleaseCommand? historyRelease,
    $2.HistoryBacklogStatusCommand? historyBacklogStatus,
    $2.LiveScreenNextCommand? liveScreenNext,
    $2.HistorySearchCommand? historySearch,
    $3.EventSubscribeCommand? eventSubscribe,
    $4.FileListCommand? fileList,
    $4.FileStatCommand? fileStat,
    $4.FilePreviewCommand? filePreview,
    $4.FileMkdirCommand? fileMkdir,
    $4.FileRenameCommand? fileRename,
    $4.FileDeleteCommand? fileDelete,
    $4.FileCopyCommand? fileCopy,
    $4.FileMoveCommand? fileMove,
    $4.FileDownloadOpenCommand? fileDownloadOpen,
    $4.FileUploadOpenCommand? fileUploadOpen,
    $4.FileTransferCancelCommand? fileTransferCancel,
    $5.StorageGetCommand? storageGet,
    $5.StoragePutCommand? storagePut,
    $5.StorageDeleteCommand? storageDelete,
    $5.StorageListCommand? storageList,
    $6.ClientAccessIdentityCommand? clientAccessIdentity,
    $6.ClientAccessListCommand? clientAccessList,
    $6.ClientAccessTicketCreateCommand? clientAccessTicketCreate,
    $6.ClientAccessRevokeCommand? clientAccessRevoke,
    $6.RemoteStatusCommand? remoteStatus,
    $6.RemotePairStartCommand? remotePairStart,
    $6.RemoteLocalEnableCommand? remoteLocalEnable,
    $6.RemoteLocalStatusCommand? remoteLocalStatus,
    $6.RemoteLocalDisableCommand? remoteLocalDisable,
    $6.RemoteCloudEdgesCommand? remoteCloudEdges,
    $6.RemoteCloudPreferEdgeCommand? remoteCloudPreferEdge,
    $6.RemoteCloudReselectEdgeCommand? remoteCloudReselectEdge,
    $6.RemoteCloudStatusCommand? remoteCloudStatus,
    $6.RemoteCloudEnableCommand? remoteCloudEnable,
    $6.RemoteCloudDisableCommand? remoteCloudDisable,
  }) {
    final result = create();
    if (context != null) result.context = context;
    if (cancelOperation != null) result.cancelOperation = cancelOperation;
    if (releaseResource != null) result.releaseResource = releaseResource;
    if (terminalDefaults != null) result.terminalDefaults = terminalDefaults;
    if (terminalCreate != null) result.terminalCreate = terminalCreate;
    if (terminalList != null) result.terminalList = terminalList;
    if (terminalGet != null) result.terminalGet = terminalGet;
    if (terminalRestart != null) result.terminalRestart = terminalRestart;
    if (terminalKill != null) result.terminalKill = terminalKill;
    if (terminalRemove != null) result.terminalRemove = terminalRemove;
    if (terminalSetMetadata != null)
      result.terminalSetMetadata = terminalSetMetadata;
    if (terminalSetTags != null) result.terminalSetTags = terminalSetTags;
    if (terminalAttach != null) result.terminalAttach = terminalAttach;
    if (terminalDetach != null) result.terminalDetach = terminalDetach;
    if (terminalInput != null) result.terminalInput = terminalInput;
    if (terminalResize != null) result.terminalResize = terminalResize;
    if (terminalResizeLock != null)
      result.terminalResizeLock = terminalResizeLock;
    if (pathListDirectories != null)
      result.pathListDirectories = pathListDirectories;
    if (historyWindow != null) result.historyWindow = historyWindow;
    if (historyCopy != null) result.historyCopy = historyCopy;
    if (historyRelease != null) result.historyRelease = historyRelease;
    if (historyBacklogStatus != null)
      result.historyBacklogStatus = historyBacklogStatus;
    if (liveScreenNext != null) result.liveScreenNext = liveScreenNext;
    if (historySearch != null) result.historySearch = historySearch;
    if (eventSubscribe != null) result.eventSubscribe = eventSubscribe;
    if (fileList != null) result.fileList = fileList;
    if (fileStat != null) result.fileStat = fileStat;
    if (filePreview != null) result.filePreview = filePreview;
    if (fileMkdir != null) result.fileMkdir = fileMkdir;
    if (fileRename != null) result.fileRename = fileRename;
    if (fileDelete != null) result.fileDelete = fileDelete;
    if (fileCopy != null) result.fileCopy = fileCopy;
    if (fileMove != null) result.fileMove = fileMove;
    if (fileDownloadOpen != null) result.fileDownloadOpen = fileDownloadOpen;
    if (fileUploadOpen != null) result.fileUploadOpen = fileUploadOpen;
    if (fileTransferCancel != null)
      result.fileTransferCancel = fileTransferCancel;
    if (storageGet != null) result.storageGet = storageGet;
    if (storagePut != null) result.storagePut = storagePut;
    if (storageDelete != null) result.storageDelete = storageDelete;
    if (storageList != null) result.storageList = storageList;
    if (clientAccessIdentity != null)
      result.clientAccessIdentity = clientAccessIdentity;
    if (clientAccessList != null) result.clientAccessList = clientAccessList;
    if (clientAccessTicketCreate != null)
      result.clientAccessTicketCreate = clientAccessTicketCreate;
    if (clientAccessRevoke != null)
      result.clientAccessRevoke = clientAccessRevoke;
    if (remoteStatus != null) result.remoteStatus = remoteStatus;
    if (remotePairStart != null) result.remotePairStart = remotePairStart;
    if (remoteLocalEnable != null) result.remoteLocalEnable = remoteLocalEnable;
    if (remoteLocalStatus != null) result.remoteLocalStatus = remoteLocalStatus;
    if (remoteLocalDisable != null)
      result.remoteLocalDisable = remoteLocalDisable;
    if (remoteCloudEdges != null) result.remoteCloudEdges = remoteCloudEdges;
    if (remoteCloudPreferEdge != null)
      result.remoteCloudPreferEdge = remoteCloudPreferEdge;
    if (remoteCloudReselectEdge != null)
      result.remoteCloudReselectEdge = remoteCloudReselectEdge;
    if (remoteCloudStatus != null) result.remoteCloudStatus = remoteCloudStatus;
    if (remoteCloudEnable != null) result.remoteCloudEnable = remoteCloudEnable;
    if (remoteCloudDisable != null)
      result.remoteCloudDisable = remoteCloudDisable;
    return result;
  }

  CommandEnvelope._();

  factory CommandEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommandEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, CommandEnvelope_Command>
      _CommandEnvelope_CommandByTag = {
    10: CommandEnvelope_Command.cancelOperation,
    11: CommandEnvelope_Command.releaseResource,
    20: CommandEnvelope_Command.terminalDefaults,
    21: CommandEnvelope_Command.terminalCreate,
    22: CommandEnvelope_Command.terminalList,
    23: CommandEnvelope_Command.terminalGet,
    24: CommandEnvelope_Command.terminalRestart,
    25: CommandEnvelope_Command.terminalKill,
    26: CommandEnvelope_Command.terminalRemove,
    27: CommandEnvelope_Command.terminalSetMetadata,
    28: CommandEnvelope_Command.terminalSetTags,
    29: CommandEnvelope_Command.terminalAttach,
    30: CommandEnvelope_Command.terminalDetach,
    31: CommandEnvelope_Command.terminalInput,
    32: CommandEnvelope_Command.terminalResize,
    33: CommandEnvelope_Command.terminalResizeLock,
    34: CommandEnvelope_Command.pathListDirectories,
    40: CommandEnvelope_Command.historyWindow,
    41: CommandEnvelope_Command.historyCopy,
    42: CommandEnvelope_Command.historyRelease,
    43: CommandEnvelope_Command.historyBacklogStatus,
    44: CommandEnvelope_Command.liveScreenNext,
    45: CommandEnvelope_Command.historySearch,
    46: CommandEnvelope_Command.eventSubscribe,
    60: CommandEnvelope_Command.fileList,
    61: CommandEnvelope_Command.fileStat,
    62: CommandEnvelope_Command.filePreview,
    63: CommandEnvelope_Command.fileMkdir,
    64: CommandEnvelope_Command.fileRename,
    65: CommandEnvelope_Command.fileDelete,
    66: CommandEnvelope_Command.fileCopy,
    67: CommandEnvelope_Command.fileMove,
    68: CommandEnvelope_Command.fileDownloadOpen,
    69: CommandEnvelope_Command.fileUploadOpen,
    70: CommandEnvelope_Command.fileTransferCancel,
    80: CommandEnvelope_Command.storageGet,
    81: CommandEnvelope_Command.storagePut,
    82: CommandEnvelope_Command.storageDelete,
    83: CommandEnvelope_Command.storageList,
    100: CommandEnvelope_Command.clientAccessIdentity,
    101: CommandEnvelope_Command.clientAccessList,
    102: CommandEnvelope_Command.clientAccessTicketCreate,
    103: CommandEnvelope_Command.clientAccessRevoke,
    110: CommandEnvelope_Command.remoteStatus,
    111: CommandEnvelope_Command.remotePairStart,
    112: CommandEnvelope_Command.remoteLocalEnable,
    113: CommandEnvelope_Command.remoteLocalStatus,
    114: CommandEnvelope_Command.remoteLocalDisable,
    115: CommandEnvelope_Command.remoteCloudEdges,
    116: CommandEnvelope_Command.remoteCloudPreferEdge,
    117: CommandEnvelope_Command.remoteCloudReselectEdge,
    118: CommandEnvelope_Command.remoteCloudStatus,
    119: CommandEnvelope_Command.remoteCloudEnable,
    120: CommandEnvelope_Command.remoteCloudDisable,
    0: CommandEnvelope_Command.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommandEnvelope',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      10,
      11,
      20,
      21,
      22,
      23,
      24,
      25,
      26,
      27,
      28,
      29,
      30,
      31,
      32,
      33,
      34,
      40,
      41,
      42,
      43,
      44,
      45,
      46,
      60,
      61,
      62,
      63,
      64,
      65,
      66,
      67,
      68,
      69,
      70,
      80,
      81,
      82,
      83,
      100,
      101,
      102,
      103,
      110,
      111,
      112,
      113,
      114,
      115,
      116,
      117,
      118,
      119,
      120
    ])
    ..aOM<$0.RequestContext>(1, _omitFieldNames ? '' : 'context',
        subBuilder: $0.RequestContext.create)
    ..aOM<CancelOperationCommand>(10, _omitFieldNames ? '' : 'cancelOperation',
        subBuilder: CancelOperationCommand.create)
    ..aOM<ReleaseResourceCommand>(11, _omitFieldNames ? '' : 'releaseResource',
        subBuilder: ReleaseResourceCommand.create)
    ..aOM<$1.TerminalDefaultsCommand>(
        20, _omitFieldNames ? '' : 'terminalDefaults',
        subBuilder: $1.TerminalDefaultsCommand.create)
    ..aOM<$1.TerminalCreateCommand>(21, _omitFieldNames ? '' : 'terminalCreate',
        subBuilder: $1.TerminalCreateCommand.create)
    ..aOM<$1.TerminalListCommand>(22, _omitFieldNames ? '' : 'terminalList',
        subBuilder: $1.TerminalListCommand.create)
    ..aOM<$1.TerminalGetCommand>(23, _omitFieldNames ? '' : 'terminalGet',
        subBuilder: $1.TerminalGetCommand.create)
    ..aOM<$1.TerminalRestartCommand>(
        24, _omitFieldNames ? '' : 'terminalRestart',
        subBuilder: $1.TerminalRestartCommand.create)
    ..aOM<$1.TerminalKillCommand>(25, _omitFieldNames ? '' : 'terminalKill',
        subBuilder: $1.TerminalKillCommand.create)
    ..aOM<$1.TerminalRemoveCommand>(26, _omitFieldNames ? '' : 'terminalRemove',
        subBuilder: $1.TerminalRemoveCommand.create)
    ..aOM<$1.TerminalSetMetadataCommand>(
        27, _omitFieldNames ? '' : 'terminalSetMetadata',
        subBuilder: $1.TerminalSetMetadataCommand.create)
    ..aOM<$1.TerminalSetTagsCommand>(
        28, _omitFieldNames ? '' : 'terminalSetTags',
        subBuilder: $1.TerminalSetTagsCommand.create)
    ..aOM<$1.TerminalAttachCommand>(29, _omitFieldNames ? '' : 'terminalAttach',
        subBuilder: $1.TerminalAttachCommand.create)
    ..aOM<$1.TerminalDetachCommand>(30, _omitFieldNames ? '' : 'terminalDetach',
        subBuilder: $1.TerminalDetachCommand.create)
    ..aOM<$1.TerminalInputCommand>(31, _omitFieldNames ? '' : 'terminalInput',
        subBuilder: $1.TerminalInputCommand.create)
    ..aOM<$1.TerminalResizeCommand>(32, _omitFieldNames ? '' : 'terminalResize',
        subBuilder: $1.TerminalResizeCommand.create)
    ..aOM<$1.TerminalResizeLockCommand>(
        33, _omitFieldNames ? '' : 'terminalResizeLock',
        subBuilder: $1.TerminalResizeLockCommand.create)
    ..aOM<$1.PathListDirectoriesCommand>(
        34, _omitFieldNames ? '' : 'pathListDirectories',
        subBuilder: $1.PathListDirectoriesCommand.create)
    ..aOM<$2.HistoryWindowCommand>(40, _omitFieldNames ? '' : 'historyWindow',
        subBuilder: $2.HistoryWindowCommand.create)
    ..aOM<$2.HistoryCopyCommand>(41, _omitFieldNames ? '' : 'historyCopy',
        subBuilder: $2.HistoryCopyCommand.create)
    ..aOM<$2.HistoryReleaseCommand>(42, _omitFieldNames ? '' : 'historyRelease',
        subBuilder: $2.HistoryReleaseCommand.create)
    ..aOM<$2.HistoryBacklogStatusCommand>(
        43, _omitFieldNames ? '' : 'historyBacklogStatus',
        subBuilder: $2.HistoryBacklogStatusCommand.create)
    ..aOM<$2.LiveScreenNextCommand>(44, _omitFieldNames ? '' : 'liveScreenNext',
        subBuilder: $2.LiveScreenNextCommand.create)
    ..aOM<$2.HistorySearchCommand>(45, _omitFieldNames ? '' : 'historySearch',
        subBuilder: $2.HistorySearchCommand.create)
    ..aOM<$3.EventSubscribeCommand>(46, _omitFieldNames ? '' : 'eventSubscribe',
        subBuilder: $3.EventSubscribeCommand.create)
    ..aOM<$4.FileListCommand>(60, _omitFieldNames ? '' : 'fileList',
        subBuilder: $4.FileListCommand.create)
    ..aOM<$4.FileStatCommand>(61, _omitFieldNames ? '' : 'fileStat',
        subBuilder: $4.FileStatCommand.create)
    ..aOM<$4.FilePreviewCommand>(62, _omitFieldNames ? '' : 'filePreview',
        subBuilder: $4.FilePreviewCommand.create)
    ..aOM<$4.FileMkdirCommand>(63, _omitFieldNames ? '' : 'fileMkdir',
        subBuilder: $4.FileMkdirCommand.create)
    ..aOM<$4.FileRenameCommand>(64, _omitFieldNames ? '' : 'fileRename',
        subBuilder: $4.FileRenameCommand.create)
    ..aOM<$4.FileDeleteCommand>(65, _omitFieldNames ? '' : 'fileDelete',
        subBuilder: $4.FileDeleteCommand.create)
    ..aOM<$4.FileCopyCommand>(66, _omitFieldNames ? '' : 'fileCopy',
        subBuilder: $4.FileCopyCommand.create)
    ..aOM<$4.FileMoveCommand>(67, _omitFieldNames ? '' : 'fileMove',
        subBuilder: $4.FileMoveCommand.create)
    ..aOM<$4.FileDownloadOpenCommand>(
        68, _omitFieldNames ? '' : 'fileDownloadOpen',
        subBuilder: $4.FileDownloadOpenCommand.create)
    ..aOM<$4.FileUploadOpenCommand>(69, _omitFieldNames ? '' : 'fileUploadOpen',
        subBuilder: $4.FileUploadOpenCommand.create)
    ..aOM<$4.FileTransferCancelCommand>(
        70, _omitFieldNames ? '' : 'fileTransferCancel',
        subBuilder: $4.FileTransferCancelCommand.create)
    ..aOM<$5.StorageGetCommand>(80, _omitFieldNames ? '' : 'storageGet',
        subBuilder: $5.StorageGetCommand.create)
    ..aOM<$5.StoragePutCommand>(81, _omitFieldNames ? '' : 'storagePut',
        subBuilder: $5.StoragePutCommand.create)
    ..aOM<$5.StorageDeleteCommand>(82, _omitFieldNames ? '' : 'storageDelete',
        subBuilder: $5.StorageDeleteCommand.create)
    ..aOM<$5.StorageListCommand>(83, _omitFieldNames ? '' : 'storageList',
        subBuilder: $5.StorageListCommand.create)
    ..aOM<$6.ClientAccessIdentityCommand>(
        100, _omitFieldNames ? '' : 'clientAccessIdentity',
        subBuilder: $6.ClientAccessIdentityCommand.create)
    ..aOM<$6.ClientAccessListCommand>(
        101, _omitFieldNames ? '' : 'clientAccessList',
        subBuilder: $6.ClientAccessListCommand.create)
    ..aOM<$6.ClientAccessTicketCreateCommand>(
        102, _omitFieldNames ? '' : 'clientAccessTicketCreate',
        subBuilder: $6.ClientAccessTicketCreateCommand.create)
    ..aOM<$6.ClientAccessRevokeCommand>(
        103, _omitFieldNames ? '' : 'clientAccessRevoke',
        subBuilder: $6.ClientAccessRevokeCommand.create)
    ..aOM<$6.RemoteStatusCommand>(110, _omitFieldNames ? '' : 'remoteStatus',
        subBuilder: $6.RemoteStatusCommand.create)
    ..aOM<$6.RemotePairStartCommand>(
        111, _omitFieldNames ? '' : 'remotePairStart',
        subBuilder: $6.RemotePairStartCommand.create)
    ..aOM<$6.RemoteLocalEnableCommand>(
        112, _omitFieldNames ? '' : 'remoteLocalEnable',
        subBuilder: $6.RemoteLocalEnableCommand.create)
    ..aOM<$6.RemoteLocalStatusCommand>(
        113, _omitFieldNames ? '' : 'remoteLocalStatus',
        subBuilder: $6.RemoteLocalStatusCommand.create)
    ..aOM<$6.RemoteLocalDisableCommand>(
        114, _omitFieldNames ? '' : 'remoteLocalDisable',
        subBuilder: $6.RemoteLocalDisableCommand.create)
    ..aOM<$6.RemoteCloudEdgesCommand>(
        115, _omitFieldNames ? '' : 'remoteCloudEdges',
        subBuilder: $6.RemoteCloudEdgesCommand.create)
    ..aOM<$6.RemoteCloudPreferEdgeCommand>(
        116, _omitFieldNames ? '' : 'remoteCloudPreferEdge',
        subBuilder: $6.RemoteCloudPreferEdgeCommand.create)
    ..aOM<$6.RemoteCloudReselectEdgeCommand>(
        117, _omitFieldNames ? '' : 'remoteCloudReselectEdge',
        subBuilder: $6.RemoteCloudReselectEdgeCommand.create)
    ..aOM<$6.RemoteCloudStatusCommand>(
        118, _omitFieldNames ? '' : 'remoteCloudStatus',
        subBuilder: $6.RemoteCloudStatusCommand.create)
    ..aOM<$6.RemoteCloudEnableCommand>(
        119, _omitFieldNames ? '' : 'remoteCloudEnable',
        subBuilder: $6.RemoteCloudEnableCommand.create)
    ..aOM<$6.RemoteCloudDisableCommand>(
        120, _omitFieldNames ? '' : 'remoteCloudDisable',
        subBuilder: $6.RemoteCloudDisableCommand.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandEnvelope copyWith(void Function(CommandEnvelope) updates) =>
      super.copyWith((message) => updates(message as CommandEnvelope))
          as CommandEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommandEnvelope create() => CommandEnvelope._();
  @$core.override
  CommandEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommandEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommandEnvelope>(create);
  static CommandEnvelope? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
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
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  @$pb.TagNumber(33)
  @$pb.TagNumber(34)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
  @$pb.TagNumber(42)
  @$pb.TagNumber(43)
  @$pb.TagNumber(44)
  @$pb.TagNumber(45)
  @$pb.TagNumber(46)
  @$pb.TagNumber(60)
  @$pb.TagNumber(61)
  @$pb.TagNumber(62)
  @$pb.TagNumber(63)
  @$pb.TagNumber(64)
  @$pb.TagNumber(65)
  @$pb.TagNumber(66)
  @$pb.TagNumber(67)
  @$pb.TagNumber(68)
  @$pb.TagNumber(69)
  @$pb.TagNumber(70)
  @$pb.TagNumber(80)
  @$pb.TagNumber(81)
  @$pb.TagNumber(82)
  @$pb.TagNumber(83)
  @$pb.TagNumber(100)
  @$pb.TagNumber(101)
  @$pb.TagNumber(102)
  @$pb.TagNumber(103)
  @$pb.TagNumber(110)
  @$pb.TagNumber(111)
  @$pb.TagNumber(112)
  @$pb.TagNumber(113)
  @$pb.TagNumber(114)
  @$pb.TagNumber(115)
  @$pb.TagNumber(116)
  @$pb.TagNumber(117)
  @$pb.TagNumber(118)
  @$pb.TagNumber(119)
  @$pb.TagNumber(120)
  CommandEnvelope_Command whichCommand() =>
      _CommandEnvelope_CommandByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
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
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  @$pb.TagNumber(33)
  @$pb.TagNumber(34)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
  @$pb.TagNumber(42)
  @$pb.TagNumber(43)
  @$pb.TagNumber(44)
  @$pb.TagNumber(45)
  @$pb.TagNumber(46)
  @$pb.TagNumber(60)
  @$pb.TagNumber(61)
  @$pb.TagNumber(62)
  @$pb.TagNumber(63)
  @$pb.TagNumber(64)
  @$pb.TagNumber(65)
  @$pb.TagNumber(66)
  @$pb.TagNumber(67)
  @$pb.TagNumber(68)
  @$pb.TagNumber(69)
  @$pb.TagNumber(70)
  @$pb.TagNumber(80)
  @$pb.TagNumber(81)
  @$pb.TagNumber(82)
  @$pb.TagNumber(83)
  @$pb.TagNumber(100)
  @$pb.TagNumber(101)
  @$pb.TagNumber(102)
  @$pb.TagNumber(103)
  @$pb.TagNumber(110)
  @$pb.TagNumber(111)
  @$pb.TagNumber(112)
  @$pb.TagNumber(113)
  @$pb.TagNumber(114)
  @$pb.TagNumber(115)
  @$pb.TagNumber(116)
  @$pb.TagNumber(117)
  @$pb.TagNumber(118)
  @$pb.TagNumber(119)
  @$pb.TagNumber(120)
  void clearCommand() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $0.RequestContext get context => $_getN(0);
  @$pb.TagNumber(1)
  set context($0.RequestContext value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasContext() => $_has(0);
  @$pb.TagNumber(1)
  void clearContext() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.RequestContext ensureContext() => $_ensure(0);

  @$pb.TagNumber(10)
  CancelOperationCommand get cancelOperation => $_getN(1);
  @$pb.TagNumber(10)
  set cancelOperation(CancelOperationCommand value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasCancelOperation() => $_has(1);
  @$pb.TagNumber(10)
  void clearCancelOperation() => $_clearField(10);
  @$pb.TagNumber(10)
  CancelOperationCommand ensureCancelOperation() => $_ensure(1);

  @$pb.TagNumber(11)
  ReleaseResourceCommand get releaseResource => $_getN(2);
  @$pb.TagNumber(11)
  set releaseResource(ReleaseResourceCommand value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasReleaseResource() => $_has(2);
  @$pb.TagNumber(11)
  void clearReleaseResource() => $_clearField(11);
  @$pb.TagNumber(11)
  ReleaseResourceCommand ensureReleaseResource() => $_ensure(2);

  @$pb.TagNumber(20)
  $1.TerminalDefaultsCommand get terminalDefaults => $_getN(3);
  @$pb.TagNumber(20)
  set terminalDefaults($1.TerminalDefaultsCommand value) =>
      $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasTerminalDefaults() => $_has(3);
  @$pb.TagNumber(20)
  void clearTerminalDefaults() => $_clearField(20);
  @$pb.TagNumber(20)
  $1.TerminalDefaultsCommand ensureTerminalDefaults() => $_ensure(3);

  @$pb.TagNumber(21)
  $1.TerminalCreateCommand get terminalCreate => $_getN(4);
  @$pb.TagNumber(21)
  set terminalCreate($1.TerminalCreateCommand value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasTerminalCreate() => $_has(4);
  @$pb.TagNumber(21)
  void clearTerminalCreate() => $_clearField(21);
  @$pb.TagNumber(21)
  $1.TerminalCreateCommand ensureTerminalCreate() => $_ensure(4);

  @$pb.TagNumber(22)
  $1.TerminalListCommand get terminalList => $_getN(5);
  @$pb.TagNumber(22)
  set terminalList($1.TerminalListCommand value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasTerminalList() => $_has(5);
  @$pb.TagNumber(22)
  void clearTerminalList() => $_clearField(22);
  @$pb.TagNumber(22)
  $1.TerminalListCommand ensureTerminalList() => $_ensure(5);

  @$pb.TagNumber(23)
  $1.TerminalGetCommand get terminalGet => $_getN(6);
  @$pb.TagNumber(23)
  set terminalGet($1.TerminalGetCommand value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasTerminalGet() => $_has(6);
  @$pb.TagNumber(23)
  void clearTerminalGet() => $_clearField(23);
  @$pb.TagNumber(23)
  $1.TerminalGetCommand ensureTerminalGet() => $_ensure(6);

  @$pb.TagNumber(24)
  $1.TerminalRestartCommand get terminalRestart => $_getN(7);
  @$pb.TagNumber(24)
  set terminalRestart($1.TerminalRestartCommand value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasTerminalRestart() => $_has(7);
  @$pb.TagNumber(24)
  void clearTerminalRestart() => $_clearField(24);
  @$pb.TagNumber(24)
  $1.TerminalRestartCommand ensureTerminalRestart() => $_ensure(7);

  @$pb.TagNumber(25)
  $1.TerminalKillCommand get terminalKill => $_getN(8);
  @$pb.TagNumber(25)
  set terminalKill($1.TerminalKillCommand value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasTerminalKill() => $_has(8);
  @$pb.TagNumber(25)
  void clearTerminalKill() => $_clearField(25);
  @$pb.TagNumber(25)
  $1.TerminalKillCommand ensureTerminalKill() => $_ensure(8);

  @$pb.TagNumber(26)
  $1.TerminalRemoveCommand get terminalRemove => $_getN(9);
  @$pb.TagNumber(26)
  set terminalRemove($1.TerminalRemoveCommand value) => $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasTerminalRemove() => $_has(9);
  @$pb.TagNumber(26)
  void clearTerminalRemove() => $_clearField(26);
  @$pb.TagNumber(26)
  $1.TerminalRemoveCommand ensureTerminalRemove() => $_ensure(9);

  @$pb.TagNumber(27)
  $1.TerminalSetMetadataCommand get terminalSetMetadata => $_getN(10);
  @$pb.TagNumber(27)
  set terminalSetMetadata($1.TerminalSetMetadataCommand value) =>
      $_setField(27, value);
  @$pb.TagNumber(27)
  $core.bool hasTerminalSetMetadata() => $_has(10);
  @$pb.TagNumber(27)
  void clearTerminalSetMetadata() => $_clearField(27);
  @$pb.TagNumber(27)
  $1.TerminalSetMetadataCommand ensureTerminalSetMetadata() => $_ensure(10);

  @$pb.TagNumber(28)
  $1.TerminalSetTagsCommand get terminalSetTags => $_getN(11);
  @$pb.TagNumber(28)
  set terminalSetTags($1.TerminalSetTagsCommand value) => $_setField(28, value);
  @$pb.TagNumber(28)
  $core.bool hasTerminalSetTags() => $_has(11);
  @$pb.TagNumber(28)
  void clearTerminalSetTags() => $_clearField(28);
  @$pb.TagNumber(28)
  $1.TerminalSetTagsCommand ensureTerminalSetTags() => $_ensure(11);

  @$pb.TagNumber(29)
  $1.TerminalAttachCommand get terminalAttach => $_getN(12);
  @$pb.TagNumber(29)
  set terminalAttach($1.TerminalAttachCommand value) => $_setField(29, value);
  @$pb.TagNumber(29)
  $core.bool hasTerminalAttach() => $_has(12);
  @$pb.TagNumber(29)
  void clearTerminalAttach() => $_clearField(29);
  @$pb.TagNumber(29)
  $1.TerminalAttachCommand ensureTerminalAttach() => $_ensure(12);

  @$pb.TagNumber(30)
  $1.TerminalDetachCommand get terminalDetach => $_getN(13);
  @$pb.TagNumber(30)
  set terminalDetach($1.TerminalDetachCommand value) => $_setField(30, value);
  @$pb.TagNumber(30)
  $core.bool hasTerminalDetach() => $_has(13);
  @$pb.TagNumber(30)
  void clearTerminalDetach() => $_clearField(30);
  @$pb.TagNumber(30)
  $1.TerminalDetachCommand ensureTerminalDetach() => $_ensure(13);

  @$pb.TagNumber(31)
  $1.TerminalInputCommand get terminalInput => $_getN(14);
  @$pb.TagNumber(31)
  set terminalInput($1.TerminalInputCommand value) => $_setField(31, value);
  @$pb.TagNumber(31)
  $core.bool hasTerminalInput() => $_has(14);
  @$pb.TagNumber(31)
  void clearTerminalInput() => $_clearField(31);
  @$pb.TagNumber(31)
  $1.TerminalInputCommand ensureTerminalInput() => $_ensure(14);

  @$pb.TagNumber(32)
  $1.TerminalResizeCommand get terminalResize => $_getN(15);
  @$pb.TagNumber(32)
  set terminalResize($1.TerminalResizeCommand value) => $_setField(32, value);
  @$pb.TagNumber(32)
  $core.bool hasTerminalResize() => $_has(15);
  @$pb.TagNumber(32)
  void clearTerminalResize() => $_clearField(32);
  @$pb.TagNumber(32)
  $1.TerminalResizeCommand ensureTerminalResize() => $_ensure(15);

  @$pb.TagNumber(33)
  $1.TerminalResizeLockCommand get terminalResizeLock => $_getN(16);
  @$pb.TagNumber(33)
  set terminalResizeLock($1.TerminalResizeLockCommand value) =>
      $_setField(33, value);
  @$pb.TagNumber(33)
  $core.bool hasTerminalResizeLock() => $_has(16);
  @$pb.TagNumber(33)
  void clearTerminalResizeLock() => $_clearField(33);
  @$pb.TagNumber(33)
  $1.TerminalResizeLockCommand ensureTerminalResizeLock() => $_ensure(16);

  @$pb.TagNumber(34)
  $1.PathListDirectoriesCommand get pathListDirectories => $_getN(17);
  @$pb.TagNumber(34)
  set pathListDirectories($1.PathListDirectoriesCommand value) =>
      $_setField(34, value);
  @$pb.TagNumber(34)
  $core.bool hasPathListDirectories() => $_has(17);
  @$pb.TagNumber(34)
  void clearPathListDirectories() => $_clearField(34);
  @$pb.TagNumber(34)
  $1.PathListDirectoriesCommand ensurePathListDirectories() => $_ensure(17);

  @$pb.TagNumber(40)
  $2.HistoryWindowCommand get historyWindow => $_getN(18);
  @$pb.TagNumber(40)
  set historyWindow($2.HistoryWindowCommand value) => $_setField(40, value);
  @$pb.TagNumber(40)
  $core.bool hasHistoryWindow() => $_has(18);
  @$pb.TagNumber(40)
  void clearHistoryWindow() => $_clearField(40);
  @$pb.TagNumber(40)
  $2.HistoryWindowCommand ensureHistoryWindow() => $_ensure(18);

  @$pb.TagNumber(41)
  $2.HistoryCopyCommand get historyCopy => $_getN(19);
  @$pb.TagNumber(41)
  set historyCopy($2.HistoryCopyCommand value) => $_setField(41, value);
  @$pb.TagNumber(41)
  $core.bool hasHistoryCopy() => $_has(19);
  @$pb.TagNumber(41)
  void clearHistoryCopy() => $_clearField(41);
  @$pb.TagNumber(41)
  $2.HistoryCopyCommand ensureHistoryCopy() => $_ensure(19);

  @$pb.TagNumber(42)
  $2.HistoryReleaseCommand get historyRelease => $_getN(20);
  @$pb.TagNumber(42)
  set historyRelease($2.HistoryReleaseCommand value) => $_setField(42, value);
  @$pb.TagNumber(42)
  $core.bool hasHistoryRelease() => $_has(20);
  @$pb.TagNumber(42)
  void clearHistoryRelease() => $_clearField(42);
  @$pb.TagNumber(42)
  $2.HistoryReleaseCommand ensureHistoryRelease() => $_ensure(20);

  @$pb.TagNumber(43)
  $2.HistoryBacklogStatusCommand get historyBacklogStatus => $_getN(21);
  @$pb.TagNumber(43)
  set historyBacklogStatus($2.HistoryBacklogStatusCommand value) =>
      $_setField(43, value);
  @$pb.TagNumber(43)
  $core.bool hasHistoryBacklogStatus() => $_has(21);
  @$pb.TagNumber(43)
  void clearHistoryBacklogStatus() => $_clearField(43);
  @$pb.TagNumber(43)
  $2.HistoryBacklogStatusCommand ensureHistoryBacklogStatus() => $_ensure(21);

  @$pb.TagNumber(44)
  $2.LiveScreenNextCommand get liveScreenNext => $_getN(22);
  @$pb.TagNumber(44)
  set liveScreenNext($2.LiveScreenNextCommand value) => $_setField(44, value);
  @$pb.TagNumber(44)
  $core.bool hasLiveScreenNext() => $_has(22);
  @$pb.TagNumber(44)
  void clearLiveScreenNext() => $_clearField(44);
  @$pb.TagNumber(44)
  $2.LiveScreenNextCommand ensureLiveScreenNext() => $_ensure(22);

  @$pb.TagNumber(45)
  $2.HistorySearchCommand get historySearch => $_getN(23);
  @$pb.TagNumber(45)
  set historySearch($2.HistorySearchCommand value) => $_setField(45, value);
  @$pb.TagNumber(45)
  $core.bool hasHistorySearch() => $_has(23);
  @$pb.TagNumber(45)
  void clearHistorySearch() => $_clearField(45);
  @$pb.TagNumber(45)
  $2.HistorySearchCommand ensureHistorySearch() => $_ensure(23);

  @$pb.TagNumber(46)
  $3.EventSubscribeCommand get eventSubscribe => $_getN(24);
  @$pb.TagNumber(46)
  set eventSubscribe($3.EventSubscribeCommand value) => $_setField(46, value);
  @$pb.TagNumber(46)
  $core.bool hasEventSubscribe() => $_has(24);
  @$pb.TagNumber(46)
  void clearEventSubscribe() => $_clearField(46);
  @$pb.TagNumber(46)
  $3.EventSubscribeCommand ensureEventSubscribe() => $_ensure(24);

  @$pb.TagNumber(60)
  $4.FileListCommand get fileList => $_getN(25);
  @$pb.TagNumber(60)
  set fileList($4.FileListCommand value) => $_setField(60, value);
  @$pb.TagNumber(60)
  $core.bool hasFileList() => $_has(25);
  @$pb.TagNumber(60)
  void clearFileList() => $_clearField(60);
  @$pb.TagNumber(60)
  $4.FileListCommand ensureFileList() => $_ensure(25);

  @$pb.TagNumber(61)
  $4.FileStatCommand get fileStat => $_getN(26);
  @$pb.TagNumber(61)
  set fileStat($4.FileStatCommand value) => $_setField(61, value);
  @$pb.TagNumber(61)
  $core.bool hasFileStat() => $_has(26);
  @$pb.TagNumber(61)
  void clearFileStat() => $_clearField(61);
  @$pb.TagNumber(61)
  $4.FileStatCommand ensureFileStat() => $_ensure(26);

  @$pb.TagNumber(62)
  $4.FilePreviewCommand get filePreview => $_getN(27);
  @$pb.TagNumber(62)
  set filePreview($4.FilePreviewCommand value) => $_setField(62, value);
  @$pb.TagNumber(62)
  $core.bool hasFilePreview() => $_has(27);
  @$pb.TagNumber(62)
  void clearFilePreview() => $_clearField(62);
  @$pb.TagNumber(62)
  $4.FilePreviewCommand ensureFilePreview() => $_ensure(27);

  @$pb.TagNumber(63)
  $4.FileMkdirCommand get fileMkdir => $_getN(28);
  @$pb.TagNumber(63)
  set fileMkdir($4.FileMkdirCommand value) => $_setField(63, value);
  @$pb.TagNumber(63)
  $core.bool hasFileMkdir() => $_has(28);
  @$pb.TagNumber(63)
  void clearFileMkdir() => $_clearField(63);
  @$pb.TagNumber(63)
  $4.FileMkdirCommand ensureFileMkdir() => $_ensure(28);

  @$pb.TagNumber(64)
  $4.FileRenameCommand get fileRename => $_getN(29);
  @$pb.TagNumber(64)
  set fileRename($4.FileRenameCommand value) => $_setField(64, value);
  @$pb.TagNumber(64)
  $core.bool hasFileRename() => $_has(29);
  @$pb.TagNumber(64)
  void clearFileRename() => $_clearField(64);
  @$pb.TagNumber(64)
  $4.FileRenameCommand ensureFileRename() => $_ensure(29);

  @$pb.TagNumber(65)
  $4.FileDeleteCommand get fileDelete => $_getN(30);
  @$pb.TagNumber(65)
  set fileDelete($4.FileDeleteCommand value) => $_setField(65, value);
  @$pb.TagNumber(65)
  $core.bool hasFileDelete() => $_has(30);
  @$pb.TagNumber(65)
  void clearFileDelete() => $_clearField(65);
  @$pb.TagNumber(65)
  $4.FileDeleteCommand ensureFileDelete() => $_ensure(30);

  @$pb.TagNumber(66)
  $4.FileCopyCommand get fileCopy => $_getN(31);
  @$pb.TagNumber(66)
  set fileCopy($4.FileCopyCommand value) => $_setField(66, value);
  @$pb.TagNumber(66)
  $core.bool hasFileCopy() => $_has(31);
  @$pb.TagNumber(66)
  void clearFileCopy() => $_clearField(66);
  @$pb.TagNumber(66)
  $4.FileCopyCommand ensureFileCopy() => $_ensure(31);

  @$pb.TagNumber(67)
  $4.FileMoveCommand get fileMove => $_getN(32);
  @$pb.TagNumber(67)
  set fileMove($4.FileMoveCommand value) => $_setField(67, value);
  @$pb.TagNumber(67)
  $core.bool hasFileMove() => $_has(32);
  @$pb.TagNumber(67)
  void clearFileMove() => $_clearField(67);
  @$pb.TagNumber(67)
  $4.FileMoveCommand ensureFileMove() => $_ensure(32);

  @$pb.TagNumber(68)
  $4.FileDownloadOpenCommand get fileDownloadOpen => $_getN(33);
  @$pb.TagNumber(68)
  set fileDownloadOpen($4.FileDownloadOpenCommand value) =>
      $_setField(68, value);
  @$pb.TagNumber(68)
  $core.bool hasFileDownloadOpen() => $_has(33);
  @$pb.TagNumber(68)
  void clearFileDownloadOpen() => $_clearField(68);
  @$pb.TagNumber(68)
  $4.FileDownloadOpenCommand ensureFileDownloadOpen() => $_ensure(33);

  @$pb.TagNumber(69)
  $4.FileUploadOpenCommand get fileUploadOpen => $_getN(34);
  @$pb.TagNumber(69)
  set fileUploadOpen($4.FileUploadOpenCommand value) => $_setField(69, value);
  @$pb.TagNumber(69)
  $core.bool hasFileUploadOpen() => $_has(34);
  @$pb.TagNumber(69)
  void clearFileUploadOpen() => $_clearField(69);
  @$pb.TagNumber(69)
  $4.FileUploadOpenCommand ensureFileUploadOpen() => $_ensure(34);

  @$pb.TagNumber(70)
  $4.FileTransferCancelCommand get fileTransferCancel => $_getN(35);
  @$pb.TagNumber(70)
  set fileTransferCancel($4.FileTransferCancelCommand value) =>
      $_setField(70, value);
  @$pb.TagNumber(70)
  $core.bool hasFileTransferCancel() => $_has(35);
  @$pb.TagNumber(70)
  void clearFileTransferCancel() => $_clearField(70);
  @$pb.TagNumber(70)
  $4.FileTransferCancelCommand ensureFileTransferCancel() => $_ensure(35);

  @$pb.TagNumber(80)
  $5.StorageGetCommand get storageGet => $_getN(36);
  @$pb.TagNumber(80)
  set storageGet($5.StorageGetCommand value) => $_setField(80, value);
  @$pb.TagNumber(80)
  $core.bool hasStorageGet() => $_has(36);
  @$pb.TagNumber(80)
  void clearStorageGet() => $_clearField(80);
  @$pb.TagNumber(80)
  $5.StorageGetCommand ensureStorageGet() => $_ensure(36);

  @$pb.TagNumber(81)
  $5.StoragePutCommand get storagePut => $_getN(37);
  @$pb.TagNumber(81)
  set storagePut($5.StoragePutCommand value) => $_setField(81, value);
  @$pb.TagNumber(81)
  $core.bool hasStoragePut() => $_has(37);
  @$pb.TagNumber(81)
  void clearStoragePut() => $_clearField(81);
  @$pb.TagNumber(81)
  $5.StoragePutCommand ensureStoragePut() => $_ensure(37);

  @$pb.TagNumber(82)
  $5.StorageDeleteCommand get storageDelete => $_getN(38);
  @$pb.TagNumber(82)
  set storageDelete($5.StorageDeleteCommand value) => $_setField(82, value);
  @$pb.TagNumber(82)
  $core.bool hasStorageDelete() => $_has(38);
  @$pb.TagNumber(82)
  void clearStorageDelete() => $_clearField(82);
  @$pb.TagNumber(82)
  $5.StorageDeleteCommand ensureStorageDelete() => $_ensure(38);

  @$pb.TagNumber(83)
  $5.StorageListCommand get storageList => $_getN(39);
  @$pb.TagNumber(83)
  set storageList($5.StorageListCommand value) => $_setField(83, value);
  @$pb.TagNumber(83)
  $core.bool hasStorageList() => $_has(39);
  @$pb.TagNumber(83)
  void clearStorageList() => $_clearField(83);
  @$pb.TagNumber(83)
  $5.StorageListCommand ensureStorageList() => $_ensure(39);

  @$pb.TagNumber(100)
  $6.ClientAccessIdentityCommand get clientAccessIdentity => $_getN(40);
  @$pb.TagNumber(100)
  set clientAccessIdentity($6.ClientAccessIdentityCommand value) =>
      $_setField(100, value);
  @$pb.TagNumber(100)
  $core.bool hasClientAccessIdentity() => $_has(40);
  @$pb.TagNumber(100)
  void clearClientAccessIdentity() => $_clearField(100);
  @$pb.TagNumber(100)
  $6.ClientAccessIdentityCommand ensureClientAccessIdentity() => $_ensure(40);

  @$pb.TagNumber(101)
  $6.ClientAccessListCommand get clientAccessList => $_getN(41);
  @$pb.TagNumber(101)
  set clientAccessList($6.ClientAccessListCommand value) =>
      $_setField(101, value);
  @$pb.TagNumber(101)
  $core.bool hasClientAccessList() => $_has(41);
  @$pb.TagNumber(101)
  void clearClientAccessList() => $_clearField(101);
  @$pb.TagNumber(101)
  $6.ClientAccessListCommand ensureClientAccessList() => $_ensure(41);

  @$pb.TagNumber(102)
  $6.ClientAccessTicketCreateCommand get clientAccessTicketCreate => $_getN(42);
  @$pb.TagNumber(102)
  set clientAccessTicketCreate($6.ClientAccessTicketCreateCommand value) =>
      $_setField(102, value);
  @$pb.TagNumber(102)
  $core.bool hasClientAccessTicketCreate() => $_has(42);
  @$pb.TagNumber(102)
  void clearClientAccessTicketCreate() => $_clearField(102);
  @$pb.TagNumber(102)
  $6.ClientAccessTicketCreateCommand ensureClientAccessTicketCreate() =>
      $_ensure(42);

  @$pb.TagNumber(103)
  $6.ClientAccessRevokeCommand get clientAccessRevoke => $_getN(43);
  @$pb.TagNumber(103)
  set clientAccessRevoke($6.ClientAccessRevokeCommand value) =>
      $_setField(103, value);
  @$pb.TagNumber(103)
  $core.bool hasClientAccessRevoke() => $_has(43);
  @$pb.TagNumber(103)
  void clearClientAccessRevoke() => $_clearField(103);
  @$pb.TagNumber(103)
  $6.ClientAccessRevokeCommand ensureClientAccessRevoke() => $_ensure(43);

  @$pb.TagNumber(110)
  $6.RemoteStatusCommand get remoteStatus => $_getN(44);
  @$pb.TagNumber(110)
  set remoteStatus($6.RemoteStatusCommand value) => $_setField(110, value);
  @$pb.TagNumber(110)
  $core.bool hasRemoteStatus() => $_has(44);
  @$pb.TagNumber(110)
  void clearRemoteStatus() => $_clearField(110);
  @$pb.TagNumber(110)
  $6.RemoteStatusCommand ensureRemoteStatus() => $_ensure(44);

  @$pb.TagNumber(111)
  $6.RemotePairStartCommand get remotePairStart => $_getN(45);
  @$pb.TagNumber(111)
  set remotePairStart($6.RemotePairStartCommand value) =>
      $_setField(111, value);
  @$pb.TagNumber(111)
  $core.bool hasRemotePairStart() => $_has(45);
  @$pb.TagNumber(111)
  void clearRemotePairStart() => $_clearField(111);
  @$pb.TagNumber(111)
  $6.RemotePairStartCommand ensureRemotePairStart() => $_ensure(45);

  @$pb.TagNumber(112)
  $6.RemoteLocalEnableCommand get remoteLocalEnable => $_getN(46);
  @$pb.TagNumber(112)
  set remoteLocalEnable($6.RemoteLocalEnableCommand value) =>
      $_setField(112, value);
  @$pb.TagNumber(112)
  $core.bool hasRemoteLocalEnable() => $_has(46);
  @$pb.TagNumber(112)
  void clearRemoteLocalEnable() => $_clearField(112);
  @$pb.TagNumber(112)
  $6.RemoteLocalEnableCommand ensureRemoteLocalEnable() => $_ensure(46);

  @$pb.TagNumber(113)
  $6.RemoteLocalStatusCommand get remoteLocalStatus => $_getN(47);
  @$pb.TagNumber(113)
  set remoteLocalStatus($6.RemoteLocalStatusCommand value) =>
      $_setField(113, value);
  @$pb.TagNumber(113)
  $core.bool hasRemoteLocalStatus() => $_has(47);
  @$pb.TagNumber(113)
  void clearRemoteLocalStatus() => $_clearField(113);
  @$pb.TagNumber(113)
  $6.RemoteLocalStatusCommand ensureRemoteLocalStatus() => $_ensure(47);

  @$pb.TagNumber(114)
  $6.RemoteLocalDisableCommand get remoteLocalDisable => $_getN(48);
  @$pb.TagNumber(114)
  set remoteLocalDisable($6.RemoteLocalDisableCommand value) =>
      $_setField(114, value);
  @$pb.TagNumber(114)
  $core.bool hasRemoteLocalDisable() => $_has(48);
  @$pb.TagNumber(114)
  void clearRemoteLocalDisable() => $_clearField(114);
  @$pb.TagNumber(114)
  $6.RemoteLocalDisableCommand ensureRemoteLocalDisable() => $_ensure(48);

  @$pb.TagNumber(115)
  $6.RemoteCloudEdgesCommand get remoteCloudEdges => $_getN(49);
  @$pb.TagNumber(115)
  set remoteCloudEdges($6.RemoteCloudEdgesCommand value) =>
      $_setField(115, value);
  @$pb.TagNumber(115)
  $core.bool hasRemoteCloudEdges() => $_has(49);
  @$pb.TagNumber(115)
  void clearRemoteCloudEdges() => $_clearField(115);
  @$pb.TagNumber(115)
  $6.RemoteCloudEdgesCommand ensureRemoteCloudEdges() => $_ensure(49);

  @$pb.TagNumber(116)
  $6.RemoteCloudPreferEdgeCommand get remoteCloudPreferEdge => $_getN(50);
  @$pb.TagNumber(116)
  set remoteCloudPreferEdge($6.RemoteCloudPreferEdgeCommand value) =>
      $_setField(116, value);
  @$pb.TagNumber(116)
  $core.bool hasRemoteCloudPreferEdge() => $_has(50);
  @$pb.TagNumber(116)
  void clearRemoteCloudPreferEdge() => $_clearField(116);
  @$pb.TagNumber(116)
  $6.RemoteCloudPreferEdgeCommand ensureRemoteCloudPreferEdge() => $_ensure(50);

  @$pb.TagNumber(117)
  $6.RemoteCloudReselectEdgeCommand get remoteCloudReselectEdge => $_getN(51);
  @$pb.TagNumber(117)
  set remoteCloudReselectEdge($6.RemoteCloudReselectEdgeCommand value) =>
      $_setField(117, value);
  @$pb.TagNumber(117)
  $core.bool hasRemoteCloudReselectEdge() => $_has(51);
  @$pb.TagNumber(117)
  void clearRemoteCloudReselectEdge() => $_clearField(117);
  @$pb.TagNumber(117)
  $6.RemoteCloudReselectEdgeCommand ensureRemoteCloudReselectEdge() =>
      $_ensure(51);

  @$pb.TagNumber(118)
  $6.RemoteCloudStatusCommand get remoteCloudStatus => $_getN(52);
  @$pb.TagNumber(118)
  set remoteCloudStatus($6.RemoteCloudStatusCommand value) =>
      $_setField(118, value);
  @$pb.TagNumber(118)
  $core.bool hasRemoteCloudStatus() => $_has(52);
  @$pb.TagNumber(118)
  void clearRemoteCloudStatus() => $_clearField(118);
  @$pb.TagNumber(118)
  $6.RemoteCloudStatusCommand ensureRemoteCloudStatus() => $_ensure(52);

  @$pb.TagNumber(119)
  $6.RemoteCloudEnableCommand get remoteCloudEnable => $_getN(53);
  @$pb.TagNumber(119)
  set remoteCloudEnable($6.RemoteCloudEnableCommand value) =>
      $_setField(119, value);
  @$pb.TagNumber(119)
  $core.bool hasRemoteCloudEnable() => $_has(53);
  @$pb.TagNumber(119)
  void clearRemoteCloudEnable() => $_clearField(119);
  @$pb.TagNumber(119)
  $6.RemoteCloudEnableCommand ensureRemoteCloudEnable() => $_ensure(53);

  @$pb.TagNumber(120)
  $6.RemoteCloudDisableCommand get remoteCloudDisable => $_getN(54);
  @$pb.TagNumber(120)
  set remoteCloudDisable($6.RemoteCloudDisableCommand value) =>
      $_setField(120, value);
  @$pb.TagNumber(120)
  $core.bool hasRemoteCloudDisable() => $_has(54);
  @$pb.TagNumber(120)
  void clearRemoteCloudDisable() => $_clearField(120);
  @$pb.TagNumber(120)
  $6.RemoteCloudDisableCommand ensureRemoteCloudDisable() => $_ensure(54);
}

class AcknowledgeResult extends $pb.GeneratedMessage {
  factory AcknowledgeResult() => create();

  AcknowledgeResult._();

  factory AcknowledgeResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AcknowledgeResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AcknowledgeResult',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AcknowledgeResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AcknowledgeResult copyWith(void Function(AcknowledgeResult) updates) =>
      super.copyWith((message) => updates(message as AcknowledgeResult))
          as AcknowledgeResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AcknowledgeResult create() => AcknowledgeResult._();
  @$core.override
  AcknowledgeResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AcknowledgeResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AcknowledgeResult>(create);
  static AcknowledgeResult? _defaultInstance;
}

enum ResultEnvelope_Result {
  acknowledge,
  error,
  terminalDefaults,
  terminalCreate,
  terminalList,
  terminalGet,
  terminalAttach,
  terminalResize,
  pathListDirectories,
  historyWindow,
  historyCopy,
  historyBacklogStatus,
  liveScreen,
  historySearch,
  eventSubscription,
  fileList,
  fileStat,
  filePreview,
  fileOperation,
  fileBatch,
  fileTransferOpen,
  fileTransferCancel,
  storageGet,
  storagePut,
  storageDelete,
  storageList,
  clientAccessIdentity,
  clientAccessList,
  clientAccessTicketCreate,
  clientAccessRevoke,
  remoteStatus,
  remotePairStart,
  remoteLocalStatus,
  remoteCloudEdges,
  remoteCloudStatus,
  notSet
}

class ResultEnvelope extends $pb.GeneratedMessage {
  factory ResultEnvelope({
    $core.String? requestId,
    $0.EndpointSessionStamp? originSession,
    AcknowledgeResult? acknowledge,
    $0.ApiError? error,
    $1.TerminalDefaultsResult? terminalDefaults,
    $1.TerminalCreateResult? terminalCreate,
    $1.TerminalListResult? terminalList,
    $1.TerminalGetResult? terminalGet,
    $1.TerminalAttachResult? terminalAttach,
    $1.TerminalResizeResult? terminalResize,
    $1.PathListDirectoriesResult? pathListDirectories,
    $2.HistoryWindowResult? historyWindow,
    $2.HistoryCopyResult? historyCopy,
    $2.HistoryBacklogStatusResult? historyBacklogStatus,
    $2.NativeScreenResult? liveScreen,
    $2.HistorySearchResult? historySearch,
    $3.EventSubscriptionResult? eventSubscription,
    $4.FileListResult? fileList,
    $4.FileStatResult? fileStat,
    $4.FilePreviewResult? filePreview,
    $4.FileOperationResult? fileOperation,
    $4.FileBatchResult? fileBatch,
    $4.FileTransferOpenResult? fileTransferOpen,
    $4.FileTransferCancelResult? fileTransferCancel,
    $5.StorageGetResult? storageGet,
    $5.StoragePutResult? storagePut,
    $5.StorageDeleteResult? storageDelete,
    $5.StorageListResult? storageList,
    $6.ClientAccessIdentityResult? clientAccessIdentity,
    $6.ClientAccessListResult? clientAccessList,
    $6.ClientAccessTicketCreateResult? clientAccessTicketCreate,
    $6.ClientAccessRevokeResult? clientAccessRevoke,
    $6.RemoteStatusResult? remoteStatus,
    $6.RemotePairStartResult? remotePairStart,
    $6.RemoteLocalStatusResult? remoteLocalStatus,
    $6.RemoteCloudEdgesResult? remoteCloudEdges,
    $6.RemoteCloudStatusResult? remoteCloudStatus,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (originSession != null) result.originSession = originSession;
    if (acknowledge != null) result.acknowledge = acknowledge;
    if (error != null) result.error = error;
    if (terminalDefaults != null) result.terminalDefaults = terminalDefaults;
    if (terminalCreate != null) result.terminalCreate = terminalCreate;
    if (terminalList != null) result.terminalList = terminalList;
    if (terminalGet != null) result.terminalGet = terminalGet;
    if (terminalAttach != null) result.terminalAttach = terminalAttach;
    if (terminalResize != null) result.terminalResize = terminalResize;
    if (pathListDirectories != null)
      result.pathListDirectories = pathListDirectories;
    if (historyWindow != null) result.historyWindow = historyWindow;
    if (historyCopy != null) result.historyCopy = historyCopy;
    if (historyBacklogStatus != null)
      result.historyBacklogStatus = historyBacklogStatus;
    if (liveScreen != null) result.liveScreen = liveScreen;
    if (historySearch != null) result.historySearch = historySearch;
    if (eventSubscription != null) result.eventSubscription = eventSubscription;
    if (fileList != null) result.fileList = fileList;
    if (fileStat != null) result.fileStat = fileStat;
    if (filePreview != null) result.filePreview = filePreview;
    if (fileOperation != null) result.fileOperation = fileOperation;
    if (fileBatch != null) result.fileBatch = fileBatch;
    if (fileTransferOpen != null) result.fileTransferOpen = fileTransferOpen;
    if (fileTransferCancel != null)
      result.fileTransferCancel = fileTransferCancel;
    if (storageGet != null) result.storageGet = storageGet;
    if (storagePut != null) result.storagePut = storagePut;
    if (storageDelete != null) result.storageDelete = storageDelete;
    if (storageList != null) result.storageList = storageList;
    if (clientAccessIdentity != null)
      result.clientAccessIdentity = clientAccessIdentity;
    if (clientAccessList != null) result.clientAccessList = clientAccessList;
    if (clientAccessTicketCreate != null)
      result.clientAccessTicketCreate = clientAccessTicketCreate;
    if (clientAccessRevoke != null)
      result.clientAccessRevoke = clientAccessRevoke;
    if (remoteStatus != null) result.remoteStatus = remoteStatus;
    if (remotePairStart != null) result.remotePairStart = remotePairStart;
    if (remoteLocalStatus != null) result.remoteLocalStatus = remoteLocalStatus;
    if (remoteCloudEdges != null) result.remoteCloudEdges = remoteCloudEdges;
    if (remoteCloudStatus != null) result.remoteCloudStatus = remoteCloudStatus;
    return result;
  }

  ResultEnvelope._();

  factory ResultEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResultEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ResultEnvelope_Result>
      _ResultEnvelope_ResultByTag = {
    10: ResultEnvelope_Result.acknowledge,
    11: ResultEnvelope_Result.error,
    20: ResultEnvelope_Result.terminalDefaults,
    21: ResultEnvelope_Result.terminalCreate,
    22: ResultEnvelope_Result.terminalList,
    23: ResultEnvelope_Result.terminalGet,
    24: ResultEnvelope_Result.terminalAttach,
    25: ResultEnvelope_Result.terminalResize,
    26: ResultEnvelope_Result.pathListDirectories,
    40: ResultEnvelope_Result.historyWindow,
    41: ResultEnvelope_Result.historyCopy,
    42: ResultEnvelope_Result.historyBacklogStatus,
    43: ResultEnvelope_Result.liveScreen,
    44: ResultEnvelope_Result.historySearch,
    45: ResultEnvelope_Result.eventSubscription,
    60: ResultEnvelope_Result.fileList,
    61: ResultEnvelope_Result.fileStat,
    62: ResultEnvelope_Result.filePreview,
    63: ResultEnvelope_Result.fileOperation,
    64: ResultEnvelope_Result.fileBatch,
    65: ResultEnvelope_Result.fileTransferOpen,
    66: ResultEnvelope_Result.fileTransferCancel,
    80: ResultEnvelope_Result.storageGet,
    81: ResultEnvelope_Result.storagePut,
    82: ResultEnvelope_Result.storageDelete,
    83: ResultEnvelope_Result.storageList,
    100: ResultEnvelope_Result.clientAccessIdentity,
    101: ResultEnvelope_Result.clientAccessList,
    102: ResultEnvelope_Result.clientAccessTicketCreate,
    103: ResultEnvelope_Result.clientAccessRevoke,
    110: ResultEnvelope_Result.remoteStatus,
    111: ResultEnvelope_Result.remotePairStart,
    112: ResultEnvelope_Result.remoteLocalStatus,
    113: ResultEnvelope_Result.remoteCloudEdges,
    114: ResultEnvelope_Result.remoteCloudStatus,
    0: ResultEnvelope_Result.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResultEnvelope',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      10,
      11,
      20,
      21,
      22,
      23,
      24,
      25,
      26,
      40,
      41,
      42,
      43,
      44,
      45,
      60,
      61,
      62,
      63,
      64,
      65,
      66,
      80,
      81,
      82,
      83,
      100,
      101,
      102,
      103,
      110,
      111,
      112,
      113,
      114
    ])
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<$0.EndpointSessionStamp>(2, _omitFieldNames ? '' : 'originSession',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOM<AcknowledgeResult>(10, _omitFieldNames ? '' : 'acknowledge',
        subBuilder: AcknowledgeResult.create)
    ..aOM<$0.ApiError>(11, _omitFieldNames ? '' : 'error',
        subBuilder: $0.ApiError.create)
    ..aOM<$1.TerminalDefaultsResult>(
        20, _omitFieldNames ? '' : 'terminalDefaults',
        subBuilder: $1.TerminalDefaultsResult.create)
    ..aOM<$1.TerminalCreateResult>(21, _omitFieldNames ? '' : 'terminalCreate',
        subBuilder: $1.TerminalCreateResult.create)
    ..aOM<$1.TerminalListResult>(22, _omitFieldNames ? '' : 'terminalList',
        subBuilder: $1.TerminalListResult.create)
    ..aOM<$1.TerminalGetResult>(23, _omitFieldNames ? '' : 'terminalGet',
        subBuilder: $1.TerminalGetResult.create)
    ..aOM<$1.TerminalAttachResult>(24, _omitFieldNames ? '' : 'terminalAttach',
        subBuilder: $1.TerminalAttachResult.create)
    ..aOM<$1.TerminalResizeResult>(25, _omitFieldNames ? '' : 'terminalResize',
        subBuilder: $1.TerminalResizeResult.create)
    ..aOM<$1.PathListDirectoriesResult>(
        26, _omitFieldNames ? '' : 'pathListDirectories',
        subBuilder: $1.PathListDirectoriesResult.create)
    ..aOM<$2.HistoryWindowResult>(40, _omitFieldNames ? '' : 'historyWindow',
        subBuilder: $2.HistoryWindowResult.create)
    ..aOM<$2.HistoryCopyResult>(41, _omitFieldNames ? '' : 'historyCopy',
        subBuilder: $2.HistoryCopyResult.create)
    ..aOM<$2.HistoryBacklogStatusResult>(
        42, _omitFieldNames ? '' : 'historyBacklogStatus',
        subBuilder: $2.HistoryBacklogStatusResult.create)
    ..aOM<$2.NativeScreenResult>(43, _omitFieldNames ? '' : 'liveScreen',
        subBuilder: $2.NativeScreenResult.create)
    ..aOM<$2.HistorySearchResult>(44, _omitFieldNames ? '' : 'historySearch',
        subBuilder: $2.HistorySearchResult.create)
    ..aOM<$3.EventSubscriptionResult>(
        45, _omitFieldNames ? '' : 'eventSubscription',
        subBuilder: $3.EventSubscriptionResult.create)
    ..aOM<$4.FileListResult>(60, _omitFieldNames ? '' : 'fileList',
        subBuilder: $4.FileListResult.create)
    ..aOM<$4.FileStatResult>(61, _omitFieldNames ? '' : 'fileStat',
        subBuilder: $4.FileStatResult.create)
    ..aOM<$4.FilePreviewResult>(62, _omitFieldNames ? '' : 'filePreview',
        subBuilder: $4.FilePreviewResult.create)
    ..aOM<$4.FileOperationResult>(63, _omitFieldNames ? '' : 'fileOperation',
        subBuilder: $4.FileOperationResult.create)
    ..aOM<$4.FileBatchResult>(64, _omitFieldNames ? '' : 'fileBatch',
        subBuilder: $4.FileBatchResult.create)
    ..aOM<$4.FileTransferOpenResult>(
        65, _omitFieldNames ? '' : 'fileTransferOpen',
        subBuilder: $4.FileTransferOpenResult.create)
    ..aOM<$4.FileTransferCancelResult>(
        66, _omitFieldNames ? '' : 'fileTransferCancel',
        subBuilder: $4.FileTransferCancelResult.create)
    ..aOM<$5.StorageGetResult>(80, _omitFieldNames ? '' : 'storageGet',
        subBuilder: $5.StorageGetResult.create)
    ..aOM<$5.StoragePutResult>(81, _omitFieldNames ? '' : 'storagePut',
        subBuilder: $5.StoragePutResult.create)
    ..aOM<$5.StorageDeleteResult>(82, _omitFieldNames ? '' : 'storageDelete',
        subBuilder: $5.StorageDeleteResult.create)
    ..aOM<$5.StorageListResult>(83, _omitFieldNames ? '' : 'storageList',
        subBuilder: $5.StorageListResult.create)
    ..aOM<$6.ClientAccessIdentityResult>(
        100, _omitFieldNames ? '' : 'clientAccessIdentity',
        subBuilder: $6.ClientAccessIdentityResult.create)
    ..aOM<$6.ClientAccessListResult>(
        101, _omitFieldNames ? '' : 'clientAccessList',
        subBuilder: $6.ClientAccessListResult.create)
    ..aOM<$6.ClientAccessTicketCreateResult>(
        102, _omitFieldNames ? '' : 'clientAccessTicketCreate',
        subBuilder: $6.ClientAccessTicketCreateResult.create)
    ..aOM<$6.ClientAccessRevokeResult>(
        103, _omitFieldNames ? '' : 'clientAccessRevoke',
        subBuilder: $6.ClientAccessRevokeResult.create)
    ..aOM<$6.RemoteStatusResult>(110, _omitFieldNames ? '' : 'remoteStatus',
        subBuilder: $6.RemoteStatusResult.create)
    ..aOM<$6.RemotePairStartResult>(
        111, _omitFieldNames ? '' : 'remotePairStart',
        subBuilder: $6.RemotePairStartResult.create)
    ..aOM<$6.RemoteLocalStatusResult>(
        112, _omitFieldNames ? '' : 'remoteLocalStatus',
        subBuilder: $6.RemoteLocalStatusResult.create)
    ..aOM<$6.RemoteCloudEdgesResult>(
        113, _omitFieldNames ? '' : 'remoteCloudEdges',
        subBuilder: $6.RemoteCloudEdgesResult.create)
    ..aOM<$6.RemoteCloudStatusResult>(
        114, _omitFieldNames ? '' : 'remoteCloudStatus',
        subBuilder: $6.RemoteCloudStatusResult.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResultEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResultEnvelope copyWith(void Function(ResultEnvelope) updates) =>
      super.copyWith((message) => updates(message as ResultEnvelope))
          as ResultEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResultEnvelope create() => ResultEnvelope._();
  @$core.override
  ResultEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResultEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResultEnvelope>(create);
  static ResultEnvelope? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
  @$pb.TagNumber(42)
  @$pb.TagNumber(43)
  @$pb.TagNumber(44)
  @$pb.TagNumber(45)
  @$pb.TagNumber(60)
  @$pb.TagNumber(61)
  @$pb.TagNumber(62)
  @$pb.TagNumber(63)
  @$pb.TagNumber(64)
  @$pb.TagNumber(65)
  @$pb.TagNumber(66)
  @$pb.TagNumber(80)
  @$pb.TagNumber(81)
  @$pb.TagNumber(82)
  @$pb.TagNumber(83)
  @$pb.TagNumber(100)
  @$pb.TagNumber(101)
  @$pb.TagNumber(102)
  @$pb.TagNumber(103)
  @$pb.TagNumber(110)
  @$pb.TagNumber(111)
  @$pb.TagNumber(112)
  @$pb.TagNumber(113)
  @$pb.TagNumber(114)
  ResultEnvelope_Result whichResult() =>
      _ResultEnvelope_ResultByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(40)
  @$pb.TagNumber(41)
  @$pb.TagNumber(42)
  @$pb.TagNumber(43)
  @$pb.TagNumber(44)
  @$pb.TagNumber(45)
  @$pb.TagNumber(60)
  @$pb.TagNumber(61)
  @$pb.TagNumber(62)
  @$pb.TagNumber(63)
  @$pb.TagNumber(64)
  @$pb.TagNumber(65)
  @$pb.TagNumber(66)
  @$pb.TagNumber(80)
  @$pb.TagNumber(81)
  @$pb.TagNumber(82)
  @$pb.TagNumber(83)
  @$pb.TagNumber(100)
  @$pb.TagNumber(101)
  @$pb.TagNumber(102)
  @$pb.TagNumber(103)
  @$pb.TagNumber(110)
  @$pb.TagNumber(111)
  @$pb.TagNumber(112)
  @$pb.TagNumber(113)
  @$pb.TagNumber(114)
  void clearResult() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.EndpointSessionStamp get originSession => $_getN(1);
  @$pb.TagNumber(2)
  set originSession($0.EndpointSessionStamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasOriginSession() => $_has(1);
  @$pb.TagNumber(2)
  void clearOriginSession() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.EndpointSessionStamp ensureOriginSession() => $_ensure(1);

  @$pb.TagNumber(10)
  AcknowledgeResult get acknowledge => $_getN(2);
  @$pb.TagNumber(10)
  set acknowledge(AcknowledgeResult value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasAcknowledge() => $_has(2);
  @$pb.TagNumber(10)
  void clearAcknowledge() => $_clearField(10);
  @$pb.TagNumber(10)
  AcknowledgeResult ensureAcknowledge() => $_ensure(2);

  @$pb.TagNumber(11)
  $0.ApiError get error => $_getN(3);
  @$pb.TagNumber(11)
  set error($0.ApiError value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasError() => $_has(3);
  @$pb.TagNumber(11)
  void clearError() => $_clearField(11);
  @$pb.TagNumber(11)
  $0.ApiError ensureError() => $_ensure(3);

  @$pb.TagNumber(20)
  $1.TerminalDefaultsResult get terminalDefaults => $_getN(4);
  @$pb.TagNumber(20)
  set terminalDefaults($1.TerminalDefaultsResult value) =>
      $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasTerminalDefaults() => $_has(4);
  @$pb.TagNumber(20)
  void clearTerminalDefaults() => $_clearField(20);
  @$pb.TagNumber(20)
  $1.TerminalDefaultsResult ensureTerminalDefaults() => $_ensure(4);

  @$pb.TagNumber(21)
  $1.TerminalCreateResult get terminalCreate => $_getN(5);
  @$pb.TagNumber(21)
  set terminalCreate($1.TerminalCreateResult value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasTerminalCreate() => $_has(5);
  @$pb.TagNumber(21)
  void clearTerminalCreate() => $_clearField(21);
  @$pb.TagNumber(21)
  $1.TerminalCreateResult ensureTerminalCreate() => $_ensure(5);

  @$pb.TagNumber(22)
  $1.TerminalListResult get terminalList => $_getN(6);
  @$pb.TagNumber(22)
  set terminalList($1.TerminalListResult value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasTerminalList() => $_has(6);
  @$pb.TagNumber(22)
  void clearTerminalList() => $_clearField(22);
  @$pb.TagNumber(22)
  $1.TerminalListResult ensureTerminalList() => $_ensure(6);

  @$pb.TagNumber(23)
  $1.TerminalGetResult get terminalGet => $_getN(7);
  @$pb.TagNumber(23)
  set terminalGet($1.TerminalGetResult value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasTerminalGet() => $_has(7);
  @$pb.TagNumber(23)
  void clearTerminalGet() => $_clearField(23);
  @$pb.TagNumber(23)
  $1.TerminalGetResult ensureTerminalGet() => $_ensure(7);

  @$pb.TagNumber(24)
  $1.TerminalAttachResult get terminalAttach => $_getN(8);
  @$pb.TagNumber(24)
  set terminalAttach($1.TerminalAttachResult value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasTerminalAttach() => $_has(8);
  @$pb.TagNumber(24)
  void clearTerminalAttach() => $_clearField(24);
  @$pb.TagNumber(24)
  $1.TerminalAttachResult ensureTerminalAttach() => $_ensure(8);

  @$pb.TagNumber(25)
  $1.TerminalResizeResult get terminalResize => $_getN(9);
  @$pb.TagNumber(25)
  set terminalResize($1.TerminalResizeResult value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasTerminalResize() => $_has(9);
  @$pb.TagNumber(25)
  void clearTerminalResize() => $_clearField(25);
  @$pb.TagNumber(25)
  $1.TerminalResizeResult ensureTerminalResize() => $_ensure(9);

  @$pb.TagNumber(26)
  $1.PathListDirectoriesResult get pathListDirectories => $_getN(10);
  @$pb.TagNumber(26)
  set pathListDirectories($1.PathListDirectoriesResult value) =>
      $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasPathListDirectories() => $_has(10);
  @$pb.TagNumber(26)
  void clearPathListDirectories() => $_clearField(26);
  @$pb.TagNumber(26)
  $1.PathListDirectoriesResult ensurePathListDirectories() => $_ensure(10);

  @$pb.TagNumber(40)
  $2.HistoryWindowResult get historyWindow => $_getN(11);
  @$pb.TagNumber(40)
  set historyWindow($2.HistoryWindowResult value) => $_setField(40, value);
  @$pb.TagNumber(40)
  $core.bool hasHistoryWindow() => $_has(11);
  @$pb.TagNumber(40)
  void clearHistoryWindow() => $_clearField(40);
  @$pb.TagNumber(40)
  $2.HistoryWindowResult ensureHistoryWindow() => $_ensure(11);

  @$pb.TagNumber(41)
  $2.HistoryCopyResult get historyCopy => $_getN(12);
  @$pb.TagNumber(41)
  set historyCopy($2.HistoryCopyResult value) => $_setField(41, value);
  @$pb.TagNumber(41)
  $core.bool hasHistoryCopy() => $_has(12);
  @$pb.TagNumber(41)
  void clearHistoryCopy() => $_clearField(41);
  @$pb.TagNumber(41)
  $2.HistoryCopyResult ensureHistoryCopy() => $_ensure(12);

  @$pb.TagNumber(42)
  $2.HistoryBacklogStatusResult get historyBacklogStatus => $_getN(13);
  @$pb.TagNumber(42)
  set historyBacklogStatus($2.HistoryBacklogStatusResult value) =>
      $_setField(42, value);
  @$pb.TagNumber(42)
  $core.bool hasHistoryBacklogStatus() => $_has(13);
  @$pb.TagNumber(42)
  void clearHistoryBacklogStatus() => $_clearField(42);
  @$pb.TagNumber(42)
  $2.HistoryBacklogStatusResult ensureHistoryBacklogStatus() => $_ensure(13);

  @$pb.TagNumber(43)
  $2.NativeScreenResult get liveScreen => $_getN(14);
  @$pb.TagNumber(43)
  set liveScreen($2.NativeScreenResult value) => $_setField(43, value);
  @$pb.TagNumber(43)
  $core.bool hasLiveScreen() => $_has(14);
  @$pb.TagNumber(43)
  void clearLiveScreen() => $_clearField(43);
  @$pb.TagNumber(43)
  $2.NativeScreenResult ensureLiveScreen() => $_ensure(14);

  @$pb.TagNumber(44)
  $2.HistorySearchResult get historySearch => $_getN(15);
  @$pb.TagNumber(44)
  set historySearch($2.HistorySearchResult value) => $_setField(44, value);
  @$pb.TagNumber(44)
  $core.bool hasHistorySearch() => $_has(15);
  @$pb.TagNumber(44)
  void clearHistorySearch() => $_clearField(44);
  @$pb.TagNumber(44)
  $2.HistorySearchResult ensureHistorySearch() => $_ensure(15);

  @$pb.TagNumber(45)
  $3.EventSubscriptionResult get eventSubscription => $_getN(16);
  @$pb.TagNumber(45)
  set eventSubscription($3.EventSubscriptionResult value) =>
      $_setField(45, value);
  @$pb.TagNumber(45)
  $core.bool hasEventSubscription() => $_has(16);
  @$pb.TagNumber(45)
  void clearEventSubscription() => $_clearField(45);
  @$pb.TagNumber(45)
  $3.EventSubscriptionResult ensureEventSubscription() => $_ensure(16);

  @$pb.TagNumber(60)
  $4.FileListResult get fileList => $_getN(17);
  @$pb.TagNumber(60)
  set fileList($4.FileListResult value) => $_setField(60, value);
  @$pb.TagNumber(60)
  $core.bool hasFileList() => $_has(17);
  @$pb.TagNumber(60)
  void clearFileList() => $_clearField(60);
  @$pb.TagNumber(60)
  $4.FileListResult ensureFileList() => $_ensure(17);

  @$pb.TagNumber(61)
  $4.FileStatResult get fileStat => $_getN(18);
  @$pb.TagNumber(61)
  set fileStat($4.FileStatResult value) => $_setField(61, value);
  @$pb.TagNumber(61)
  $core.bool hasFileStat() => $_has(18);
  @$pb.TagNumber(61)
  void clearFileStat() => $_clearField(61);
  @$pb.TagNumber(61)
  $4.FileStatResult ensureFileStat() => $_ensure(18);

  @$pb.TagNumber(62)
  $4.FilePreviewResult get filePreview => $_getN(19);
  @$pb.TagNumber(62)
  set filePreview($4.FilePreviewResult value) => $_setField(62, value);
  @$pb.TagNumber(62)
  $core.bool hasFilePreview() => $_has(19);
  @$pb.TagNumber(62)
  void clearFilePreview() => $_clearField(62);
  @$pb.TagNumber(62)
  $4.FilePreviewResult ensureFilePreview() => $_ensure(19);

  @$pb.TagNumber(63)
  $4.FileOperationResult get fileOperation => $_getN(20);
  @$pb.TagNumber(63)
  set fileOperation($4.FileOperationResult value) => $_setField(63, value);
  @$pb.TagNumber(63)
  $core.bool hasFileOperation() => $_has(20);
  @$pb.TagNumber(63)
  void clearFileOperation() => $_clearField(63);
  @$pb.TagNumber(63)
  $4.FileOperationResult ensureFileOperation() => $_ensure(20);

  @$pb.TagNumber(64)
  $4.FileBatchResult get fileBatch => $_getN(21);
  @$pb.TagNumber(64)
  set fileBatch($4.FileBatchResult value) => $_setField(64, value);
  @$pb.TagNumber(64)
  $core.bool hasFileBatch() => $_has(21);
  @$pb.TagNumber(64)
  void clearFileBatch() => $_clearField(64);
  @$pb.TagNumber(64)
  $4.FileBatchResult ensureFileBatch() => $_ensure(21);

  @$pb.TagNumber(65)
  $4.FileTransferOpenResult get fileTransferOpen => $_getN(22);
  @$pb.TagNumber(65)
  set fileTransferOpen($4.FileTransferOpenResult value) =>
      $_setField(65, value);
  @$pb.TagNumber(65)
  $core.bool hasFileTransferOpen() => $_has(22);
  @$pb.TagNumber(65)
  void clearFileTransferOpen() => $_clearField(65);
  @$pb.TagNumber(65)
  $4.FileTransferOpenResult ensureFileTransferOpen() => $_ensure(22);

  @$pb.TagNumber(66)
  $4.FileTransferCancelResult get fileTransferCancel => $_getN(23);
  @$pb.TagNumber(66)
  set fileTransferCancel($4.FileTransferCancelResult value) =>
      $_setField(66, value);
  @$pb.TagNumber(66)
  $core.bool hasFileTransferCancel() => $_has(23);
  @$pb.TagNumber(66)
  void clearFileTransferCancel() => $_clearField(66);
  @$pb.TagNumber(66)
  $4.FileTransferCancelResult ensureFileTransferCancel() => $_ensure(23);

  @$pb.TagNumber(80)
  $5.StorageGetResult get storageGet => $_getN(24);
  @$pb.TagNumber(80)
  set storageGet($5.StorageGetResult value) => $_setField(80, value);
  @$pb.TagNumber(80)
  $core.bool hasStorageGet() => $_has(24);
  @$pb.TagNumber(80)
  void clearStorageGet() => $_clearField(80);
  @$pb.TagNumber(80)
  $5.StorageGetResult ensureStorageGet() => $_ensure(24);

  @$pb.TagNumber(81)
  $5.StoragePutResult get storagePut => $_getN(25);
  @$pb.TagNumber(81)
  set storagePut($5.StoragePutResult value) => $_setField(81, value);
  @$pb.TagNumber(81)
  $core.bool hasStoragePut() => $_has(25);
  @$pb.TagNumber(81)
  void clearStoragePut() => $_clearField(81);
  @$pb.TagNumber(81)
  $5.StoragePutResult ensureStoragePut() => $_ensure(25);

  @$pb.TagNumber(82)
  $5.StorageDeleteResult get storageDelete => $_getN(26);
  @$pb.TagNumber(82)
  set storageDelete($5.StorageDeleteResult value) => $_setField(82, value);
  @$pb.TagNumber(82)
  $core.bool hasStorageDelete() => $_has(26);
  @$pb.TagNumber(82)
  void clearStorageDelete() => $_clearField(82);
  @$pb.TagNumber(82)
  $5.StorageDeleteResult ensureStorageDelete() => $_ensure(26);

  @$pb.TagNumber(83)
  $5.StorageListResult get storageList => $_getN(27);
  @$pb.TagNumber(83)
  set storageList($5.StorageListResult value) => $_setField(83, value);
  @$pb.TagNumber(83)
  $core.bool hasStorageList() => $_has(27);
  @$pb.TagNumber(83)
  void clearStorageList() => $_clearField(83);
  @$pb.TagNumber(83)
  $5.StorageListResult ensureStorageList() => $_ensure(27);

  @$pb.TagNumber(100)
  $6.ClientAccessIdentityResult get clientAccessIdentity => $_getN(28);
  @$pb.TagNumber(100)
  set clientAccessIdentity($6.ClientAccessIdentityResult value) =>
      $_setField(100, value);
  @$pb.TagNumber(100)
  $core.bool hasClientAccessIdentity() => $_has(28);
  @$pb.TagNumber(100)
  void clearClientAccessIdentity() => $_clearField(100);
  @$pb.TagNumber(100)
  $6.ClientAccessIdentityResult ensureClientAccessIdentity() => $_ensure(28);

  @$pb.TagNumber(101)
  $6.ClientAccessListResult get clientAccessList => $_getN(29);
  @$pb.TagNumber(101)
  set clientAccessList($6.ClientAccessListResult value) =>
      $_setField(101, value);
  @$pb.TagNumber(101)
  $core.bool hasClientAccessList() => $_has(29);
  @$pb.TagNumber(101)
  void clearClientAccessList() => $_clearField(101);
  @$pb.TagNumber(101)
  $6.ClientAccessListResult ensureClientAccessList() => $_ensure(29);

  @$pb.TagNumber(102)
  $6.ClientAccessTicketCreateResult get clientAccessTicketCreate => $_getN(30);
  @$pb.TagNumber(102)
  set clientAccessTicketCreate($6.ClientAccessTicketCreateResult value) =>
      $_setField(102, value);
  @$pb.TagNumber(102)
  $core.bool hasClientAccessTicketCreate() => $_has(30);
  @$pb.TagNumber(102)
  void clearClientAccessTicketCreate() => $_clearField(102);
  @$pb.TagNumber(102)
  $6.ClientAccessTicketCreateResult ensureClientAccessTicketCreate() =>
      $_ensure(30);

  @$pb.TagNumber(103)
  $6.ClientAccessRevokeResult get clientAccessRevoke => $_getN(31);
  @$pb.TagNumber(103)
  set clientAccessRevoke($6.ClientAccessRevokeResult value) =>
      $_setField(103, value);
  @$pb.TagNumber(103)
  $core.bool hasClientAccessRevoke() => $_has(31);
  @$pb.TagNumber(103)
  void clearClientAccessRevoke() => $_clearField(103);
  @$pb.TagNumber(103)
  $6.ClientAccessRevokeResult ensureClientAccessRevoke() => $_ensure(31);

  @$pb.TagNumber(110)
  $6.RemoteStatusResult get remoteStatus => $_getN(32);
  @$pb.TagNumber(110)
  set remoteStatus($6.RemoteStatusResult value) => $_setField(110, value);
  @$pb.TagNumber(110)
  $core.bool hasRemoteStatus() => $_has(32);
  @$pb.TagNumber(110)
  void clearRemoteStatus() => $_clearField(110);
  @$pb.TagNumber(110)
  $6.RemoteStatusResult ensureRemoteStatus() => $_ensure(32);

  @$pb.TagNumber(111)
  $6.RemotePairStartResult get remotePairStart => $_getN(33);
  @$pb.TagNumber(111)
  set remotePairStart($6.RemotePairStartResult value) => $_setField(111, value);
  @$pb.TagNumber(111)
  $core.bool hasRemotePairStart() => $_has(33);
  @$pb.TagNumber(111)
  void clearRemotePairStart() => $_clearField(111);
  @$pb.TagNumber(111)
  $6.RemotePairStartResult ensureRemotePairStart() => $_ensure(33);

  @$pb.TagNumber(112)
  $6.RemoteLocalStatusResult get remoteLocalStatus => $_getN(34);
  @$pb.TagNumber(112)
  set remoteLocalStatus($6.RemoteLocalStatusResult value) =>
      $_setField(112, value);
  @$pb.TagNumber(112)
  $core.bool hasRemoteLocalStatus() => $_has(34);
  @$pb.TagNumber(112)
  void clearRemoteLocalStatus() => $_clearField(112);
  @$pb.TagNumber(112)
  $6.RemoteLocalStatusResult ensureRemoteLocalStatus() => $_ensure(34);

  @$pb.TagNumber(113)
  $6.RemoteCloudEdgesResult get remoteCloudEdges => $_getN(35);
  @$pb.TagNumber(113)
  set remoteCloudEdges($6.RemoteCloudEdgesResult value) =>
      $_setField(113, value);
  @$pb.TagNumber(113)
  $core.bool hasRemoteCloudEdges() => $_has(35);
  @$pb.TagNumber(113)
  void clearRemoteCloudEdges() => $_clearField(113);
  @$pb.TagNumber(113)
  $6.RemoteCloudEdgesResult ensureRemoteCloudEdges() => $_ensure(35);

  @$pb.TagNumber(114)
  $6.RemoteCloudStatusResult get remoteCloudStatus => $_getN(36);
  @$pb.TagNumber(114)
  set remoteCloudStatus($6.RemoteCloudStatusResult value) =>
      $_setField(114, value);
  @$pb.TagNumber(114)
  $core.bool hasRemoteCloudStatus() => $_has(36);
  @$pb.TagNumber(114)
  void clearRemoteCloudStatus() => $_clearField(114);
  @$pb.TagNumber(114)
  $6.RemoteCloudStatusResult ensureRemoteCloudStatus() => $_ensure(36);
}

class OperationCancelledEvent extends $pb.GeneratedMessage {
  factory OperationCancelledEvent({
    $0.OperationStamp? operation,
  }) {
    final result = create();
    if (operation != null) result.operation = operation;
    return result;
  }

  OperationCancelledEvent._();

  factory OperationCancelledEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OperationCancelledEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OperationCancelledEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.OperationStamp>(1, _omitFieldNames ? '' : 'operation',
        subBuilder: $0.OperationStamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperationCancelledEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OperationCancelledEvent copyWith(
          void Function(OperationCancelledEvent) updates) =>
      super.copyWith((message) => updates(message as OperationCancelledEvent))
          as OperationCancelledEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OperationCancelledEvent create() => OperationCancelledEvent._();
  @$core.override
  OperationCancelledEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OperationCancelledEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OperationCancelledEvent>(create);
  static OperationCancelledEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $0.OperationStamp get operation => $_getN(0);
  @$pb.TagNumber(1)
  set operation($0.OperationStamp value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasOperation() => $_has(0);
  @$pb.TagNumber(1)
  void clearOperation() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.OperationStamp ensureOperation() => $_ensure(0);
}

class ResourceReleasedEvent extends $pb.GeneratedMessage {
  factory ResourceReleasedEvent({
    $0.ResourceHandle? resource,
  }) {
    final result = create();
    if (resource != null) result.resource = resource;
    return result;
  }

  ResourceReleasedEvent._();

  factory ResourceReleasedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResourceReleasedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResourceReleasedEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..aOM<$0.ResourceHandle>(1, _omitFieldNames ? '' : 'resource',
        subBuilder: $0.ResourceHandle.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceReleasedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResourceReleasedEvent copyWith(
          void Function(ResourceReleasedEvent) updates) =>
      super.copyWith((message) => updates(message as ResourceReleasedEvent))
          as ResourceReleasedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResourceReleasedEvent create() => ResourceReleasedEvent._();
  @$core.override
  ResourceReleasedEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResourceReleasedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResourceReleasedEvent>(create);
  static ResourceReleasedEvent? _defaultInstance;

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
}

enum EventEnvelope_Event {
  operationCancelled,
  resourceReleased,
  terminalLifecycle,
  storageChanged,
  fileTransferCompleted,
  notSet
}

class EventEnvelope extends $pb.GeneratedMessage {
  factory EventEnvelope({
    $core.String? eventId,
    $fixnum.Int64? timestampUnixNano,
    $0.ApiVersion? apiVersion,
    $0.EndpointSessionStamp? originSession,
    $0.ResourceHandle? subscription,
    OperationCancelledEvent? operationCancelled,
    ResourceReleasedEvent? resourceReleased,
    $1.TerminalLifecycleEvent? terminalLifecycle,
    $5.StorageChangedEvent? storageChanged,
    $4.FileTransferCompletedEvent? fileTransferCompleted,
  }) {
    final result = create();
    if (eventId != null) result.eventId = eventId;
    if (timestampUnixNano != null) result.timestampUnixNano = timestampUnixNano;
    if (apiVersion != null) result.apiVersion = apiVersion;
    if (originSession != null) result.originSession = originSession;
    if (subscription != null) result.subscription = subscription;
    if (operationCancelled != null)
      result.operationCancelled = operationCancelled;
    if (resourceReleased != null) result.resourceReleased = resourceReleased;
    if (terminalLifecycle != null) result.terminalLifecycle = terminalLifecycle;
    if (storageChanged != null) result.storageChanged = storageChanged;
    if (fileTransferCompleted != null)
      result.fileTransferCompleted = fileTransferCompleted;
    return result;
  }

  EventEnvelope._();

  factory EventEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EventEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, EventEnvelope_Event>
      _EventEnvelope_EventByTag = {
    10: EventEnvelope_Event.operationCancelled,
    11: EventEnvelope_Event.resourceReleased,
    20: EventEnvelope_Event.terminalLifecycle,
    30: EventEnvelope_Event.storageChanged,
    40: EventEnvelope_Event.fileTransferCompleted,
    0: EventEnvelope_Event.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EventEnvelope',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'anytty.api.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 20, 30, 40])
    ..aOS(1, _omitFieldNames ? '' : 'eventId')
    ..aInt64(2, _omitFieldNames ? '' : 'timestampUnixNano')
    ..aOM<$0.ApiVersion>(3, _omitFieldNames ? '' : 'apiVersion',
        subBuilder: $0.ApiVersion.create)
    ..aOM<$0.EndpointSessionStamp>(4, _omitFieldNames ? '' : 'originSession',
        subBuilder: $0.EndpointSessionStamp.create)
    ..aOM<$0.ResourceHandle>(5, _omitFieldNames ? '' : 'subscription',
        subBuilder: $0.ResourceHandle.create)
    ..aOM<OperationCancelledEvent>(
        10, _omitFieldNames ? '' : 'operationCancelled',
        subBuilder: OperationCancelledEvent.create)
    ..aOM<ResourceReleasedEvent>(11, _omitFieldNames ? '' : 'resourceReleased',
        subBuilder: ResourceReleasedEvent.create)
    ..aOM<$1.TerminalLifecycleEvent>(
        20, _omitFieldNames ? '' : 'terminalLifecycle',
        subBuilder: $1.TerminalLifecycleEvent.create)
    ..aOM<$5.StorageChangedEvent>(30, _omitFieldNames ? '' : 'storageChanged',
        subBuilder: $5.StorageChangedEvent.create)
    ..aOM<$4.FileTransferCompletedEvent>(
        40, _omitFieldNames ? '' : 'fileTransferCompleted',
        subBuilder: $4.FileTransferCompletedEvent.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventEnvelope copyWith(void Function(EventEnvelope) updates) =>
      super.copyWith((message) => updates(message as EventEnvelope))
          as EventEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EventEnvelope create() => EventEnvelope._();
  @$core.override
  EventEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EventEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EventEnvelope>(create);
  static EventEnvelope? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(20)
  @$pb.TagNumber(30)
  @$pb.TagNumber(40)
  EventEnvelope_Event whichEvent() =>
      _EventEnvelope_EventByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(20)
  @$pb.TagNumber(30)
  @$pb.TagNumber(40)
  void clearEvent() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get eventId => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEventId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get timestampUnixNano => $_getI64(1);
  @$pb.TagNumber(2)
  set timestampUnixNano($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTimestampUnixNano() => $_has(1);
  @$pb.TagNumber(2)
  void clearTimestampUnixNano() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.ApiVersion get apiVersion => $_getN(2);
  @$pb.TagNumber(3)
  set apiVersion($0.ApiVersion value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasApiVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearApiVersion() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.ApiVersion ensureApiVersion() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.EndpointSessionStamp get originSession => $_getN(3);
  @$pb.TagNumber(4)
  set originSession($0.EndpointSessionStamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasOriginSession() => $_has(3);
  @$pb.TagNumber(4)
  void clearOriginSession() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.EndpointSessionStamp ensureOriginSession() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.ResourceHandle get subscription => $_getN(4);
  @$pb.TagNumber(5)
  set subscription($0.ResourceHandle value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasSubscription() => $_has(4);
  @$pb.TagNumber(5)
  void clearSubscription() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.ResourceHandle ensureSubscription() => $_ensure(4);

  @$pb.TagNumber(10)
  OperationCancelledEvent get operationCancelled => $_getN(5);
  @$pb.TagNumber(10)
  set operationCancelled(OperationCancelledEvent value) =>
      $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasOperationCancelled() => $_has(5);
  @$pb.TagNumber(10)
  void clearOperationCancelled() => $_clearField(10);
  @$pb.TagNumber(10)
  OperationCancelledEvent ensureOperationCancelled() => $_ensure(5);

  @$pb.TagNumber(11)
  ResourceReleasedEvent get resourceReleased => $_getN(6);
  @$pb.TagNumber(11)
  set resourceReleased(ResourceReleasedEvent value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasResourceReleased() => $_has(6);
  @$pb.TagNumber(11)
  void clearResourceReleased() => $_clearField(11);
  @$pb.TagNumber(11)
  ResourceReleasedEvent ensureResourceReleased() => $_ensure(6);

  @$pb.TagNumber(20)
  $1.TerminalLifecycleEvent get terminalLifecycle => $_getN(7);
  @$pb.TagNumber(20)
  set terminalLifecycle($1.TerminalLifecycleEvent value) =>
      $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasTerminalLifecycle() => $_has(7);
  @$pb.TagNumber(20)
  void clearTerminalLifecycle() => $_clearField(20);
  @$pb.TagNumber(20)
  $1.TerminalLifecycleEvent ensureTerminalLifecycle() => $_ensure(7);

  @$pb.TagNumber(30)
  $5.StorageChangedEvent get storageChanged => $_getN(8);
  @$pb.TagNumber(30)
  set storageChanged($5.StorageChangedEvent value) => $_setField(30, value);
  @$pb.TagNumber(30)
  $core.bool hasStorageChanged() => $_has(8);
  @$pb.TagNumber(30)
  void clearStorageChanged() => $_clearField(30);
  @$pb.TagNumber(30)
  $5.StorageChangedEvent ensureStorageChanged() => $_ensure(8);

  @$pb.TagNumber(40)
  $4.FileTransferCompletedEvent get fileTransferCompleted => $_getN(9);
  @$pb.TagNumber(40)
  set fileTransferCompleted($4.FileTransferCompletedEvent value) =>
      $_setField(40, value);
  @$pb.TagNumber(40)
  $core.bool hasFileTransferCompleted() => $_has(9);
  @$pb.TagNumber(40)
  void clearFileTransferCompleted() => $_clearField(40);
  @$pb.TagNumber(40)
  $4.FileTransferCompletedEvent ensureFileTransferCompleted() => $_ensure(9);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
