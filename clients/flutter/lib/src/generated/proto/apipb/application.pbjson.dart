// This is a generated file - do not edit.
//
// Generated from apipb/application.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use cancelOperationCommandDescriptor instead')
const CancelOperationCommand$json = {
  '1': 'CancelOperationCommand',
  '2': [
    {
      '1': 'operation',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `CancelOperationCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cancelOperationCommandDescriptor =
    $convert.base64Decode(
        'ChZDYW5jZWxPcGVyYXRpb25Db21tYW5kEjsKCW9wZXJhdGlvbhgCIAEoCzIdLmFueXR0eS5hcG'
        'kudjEuT3BlcmF0aW9uU3RhbXBSCW9wZXJhdGlvbkoECAEQAg==');

@$core.Deprecated('Use releaseResourceCommandDescriptor instead')
const ReleaseResourceCommand$json = {
  '1': 'ReleaseResourceCommand',
  '2': [
    {
      '1': 'resource',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `ReleaseResourceCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List releaseResourceCommandDescriptor =
    $convert.base64Decode(
        'ChZSZWxlYXNlUmVzb3VyY2VDb21tYW5kEjkKCHJlc291cmNlGAIgASgLMh0uYW55dHR5LmFwaS'
        '52MS5SZXNvdXJjZUhhbmRsZVIIcmVzb3VyY2VKBAgBEAI=');

@$core.Deprecated('Use browserProxyOpenCommandDescriptor instead')
const BrowserProxyOpenCommand$json = {
  '1': 'BrowserProxyOpenCommand',
  '2': [
    {'1': 'host', '3': 1, '4': 1, '5': 9, '10': 'host'},
    {'1': 'port', '3': 2, '4': 1, '5': 13, '10': 'port'},
  ],
};

/// Descriptor for `BrowserProxyOpenCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List browserProxyOpenCommandDescriptor =
    $convert.base64Decode(
        'ChdCcm93c2VyUHJveHlPcGVuQ29tbWFuZBISCgRob3N0GAEgASgJUgRob3N0EhIKBHBvcnQYAi'
        'ABKA1SBHBvcnQ=');

@$core.Deprecated('Use commandEnvelopeDescriptor instead')
const CommandEnvelope$json = {
  '1': 'CommandEnvelope',
  '2': [
    {
      '1': 'context',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RequestContext',
      '10': 'context'
    },
    {
      '1': 'cancel_operation',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.CancelOperationCommand',
      '9': 0,
      '10': 'cancelOperation'
    },
    {
      '1': 'release_resource',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ReleaseResourceCommand',
      '9': 0,
      '10': 'releaseResource'
    },
    {
      '1': 'terminal_defaults',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalDefaultsCommand',
      '9': 0,
      '10': 'terminalDefaults'
    },
    {
      '1': 'terminal_create',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalCreateCommand',
      '9': 0,
      '10': 'terminalCreate'
    },
    {
      '1': 'terminal_list',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalListCommand',
      '9': 0,
      '10': 'terminalList'
    },
    {
      '1': 'terminal_get',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalGetCommand',
      '9': 0,
      '10': 'terminalGet'
    },
    {
      '1': 'terminal_restart',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRestartCommand',
      '9': 0,
      '10': 'terminalRestart'
    },
    {
      '1': 'terminal_kill',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalKillCommand',
      '9': 0,
      '10': 'terminalKill'
    },
    {
      '1': 'terminal_remove',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalRemoveCommand',
      '9': 0,
      '10': 'terminalRemove'
    },
    {
      '1': 'terminal_set_metadata',
      '3': 27,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSetMetadataCommand',
      '9': 0,
      '10': 'terminalSetMetadata'
    },
    {
      '1': 'terminal_set_tags',
      '3': 28,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalSetTagsCommand',
      '9': 0,
      '10': 'terminalSetTags'
    },
    {
      '1': 'terminal_attach',
      '3': 29,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalAttachCommand',
      '9': 0,
      '10': 'terminalAttach'
    },
    {
      '1': 'terminal_detach',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalDetachCommand',
      '9': 0,
      '10': 'terminalDetach'
    },
    {
      '1': 'terminal_input',
      '3': 31,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalInputCommand',
      '9': 0,
      '10': 'terminalInput'
    },
    {
      '1': 'terminal_resize',
      '3': 32,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalResizeCommand',
      '9': 0,
      '10': 'terminalResize'
    },
    {
      '1': 'terminal_resize_lock',
      '3': 33,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalResizeLockCommand',
      '9': 0,
      '10': 'terminalResizeLock'
    },
    {
      '1': 'path_list_directories',
      '3': 34,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PathListDirectoriesCommand',
      '9': 0,
      '10': 'pathListDirectories'
    },
    {
      '1': 'history_window',
      '3': 40,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryWindowCommand',
      '9': 0,
      '10': 'historyWindow'
    },
    {
      '1': 'history_copy',
      '3': 41,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryCopyCommand',
      '9': 0,
      '10': 'historyCopy'
    },
    {
      '1': 'history_release',
      '3': 42,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryReleaseCommand',
      '9': 0,
      '10': 'historyRelease'
    },
    {
      '1': 'history_backlog_status',
      '3': 43,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryBacklogStatusCommand',
      '9': 0,
      '10': 'historyBacklogStatus'
    },
    {
      '1': 'live_screen_next',
      '3': 44,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.LiveScreenNextCommand',
      '9': 0,
      '10': 'liveScreenNext'
    },
    {
      '1': 'history_search',
      '3': 45,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistorySearchCommand',
      '9': 0,
      '10': 'historySearch'
    },
    {
      '1': 'event_subscribe',
      '3': 46,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EventSubscribeCommand',
      '9': 0,
      '10': 'eventSubscribe'
    },
    {
      '1': 'file_list',
      '3': 60,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileListCommand',
      '9': 0,
      '10': 'fileList'
    },
    {
      '1': 'file_stat',
      '3': 61,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileStatCommand',
      '9': 0,
      '10': 'fileStat'
    },
    {
      '1': 'file_preview',
      '3': 62,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FilePreviewCommand',
      '9': 0,
      '10': 'filePreview'
    },
    {
      '1': 'file_mkdir',
      '3': 63,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileMkdirCommand',
      '9': 0,
      '10': 'fileMkdir'
    },
    {
      '1': 'file_rename',
      '3': 64,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileRenameCommand',
      '9': 0,
      '10': 'fileRename'
    },
    {
      '1': 'file_delete',
      '3': 65,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileDeleteCommand',
      '9': 0,
      '10': 'fileDelete'
    },
    {
      '1': 'file_copy',
      '3': 66,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileCopyCommand',
      '9': 0,
      '10': 'fileCopy'
    },
    {
      '1': 'file_move',
      '3': 67,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileMoveCommand',
      '9': 0,
      '10': 'fileMove'
    },
    {
      '1': 'file_download_open',
      '3': 68,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileDownloadOpenCommand',
      '9': 0,
      '10': 'fileDownloadOpen'
    },
    {
      '1': 'file_upload_open',
      '3': 69,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileUploadOpenCommand',
      '9': 0,
      '10': 'fileUploadOpen'
    },
    {
      '1': 'file_transfer_cancel',
      '3': 70,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileTransferCancelCommand',
      '9': 0,
      '10': 'fileTransferCancel'
    },
    {
      '1': 'storage_get',
      '3': 80,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageGetCommand',
      '9': 0,
      '10': 'storageGet'
    },
    {
      '1': 'storage_put',
      '3': 81,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StoragePutCommand',
      '9': 0,
      '10': 'storagePut'
    },
    {
      '1': 'storage_delete',
      '3': 82,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageDeleteCommand',
      '9': 0,
      '10': 'storageDelete'
    },
    {
      '1': 'storage_list',
      '3': 83,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageListCommand',
      '9': 0,
      '10': 'storageList'
    },
    {
      '1': 'client_access_identity',
      '3': 100,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessIdentityCommand',
      '9': 0,
      '10': 'clientAccessIdentity'
    },
    {
      '1': 'client_access_list',
      '3': 101,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessListCommand',
      '9': 0,
      '10': 'clientAccessList'
    },
    {
      '1': 'client_access_ticket_create',
      '3': 102,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessTicketCreateCommand',
      '9': 0,
      '10': 'clientAccessTicketCreate'
    },
    {
      '1': 'client_access_revoke',
      '3': 103,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessRevokeCommand',
      '9': 0,
      '10': 'clientAccessRevoke'
    },
    {
      '1': 'remote_status',
      '3': 110,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteStatusCommand',
      '9': 0,
      '10': 'remoteStatus'
    },
    {
      '1': 'remote_pair_start',
      '3': 111,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemotePairStartCommand',
      '9': 0,
      '10': 'remotePairStart'
    },
    {
      '1': 'remote_local_enable',
      '3': 112,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteLocalEnableCommand',
      '9': 0,
      '10': 'remoteLocalEnable'
    },
    {
      '1': 'remote_local_status',
      '3': 113,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteLocalStatusCommand',
      '9': 0,
      '10': 'remoteLocalStatus'
    },
    {
      '1': 'remote_local_disable',
      '3': 114,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteLocalDisableCommand',
      '9': 0,
      '10': 'remoteLocalDisable'
    },
    {
      '1': 'remote_cloud_edges',
      '3': 115,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudEdgesCommand',
      '9': 0,
      '10': 'remoteCloudEdges'
    },
    {
      '1': 'remote_cloud_prefer_edge',
      '3': 116,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudPreferEdgeCommand',
      '9': 0,
      '10': 'remoteCloudPreferEdge'
    },
    {
      '1': 'remote_cloud_reselect_edge',
      '3': 117,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudReselectEdgeCommand',
      '9': 0,
      '10': 'remoteCloudReselectEdge'
    },
    {
      '1': 'remote_cloud_status',
      '3': 118,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudStatusCommand',
      '9': 0,
      '10': 'remoteCloudStatus'
    },
    {
      '1': 'remote_cloud_enable',
      '3': 119,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudEnableCommand',
      '9': 0,
      '10': 'remoteCloudEnable'
    },
    {
      '1': 'remote_cloud_disable',
      '3': 120,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudDisableCommand',
      '9': 0,
      '10': 'remoteCloudDisable'
    },
    {
      '1': 'browser_proxy_open',
      '3': 121,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.BrowserProxyOpenCommand',
      '9': 0,
      '10': 'browserProxyOpen'
    },
  ],
  '8': [
    {'1': 'command'},
  ],
};

/// Descriptor for `CommandEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commandEnvelopeDescriptor = $convert.base64Decode(
    'Cg9Db21tYW5kRW52ZWxvcGUSNwoHY29udGV4dBgBIAEoCzIdLmFueXR0eS5hcGkudjEuUmVxdW'
    'VzdENvbnRleHRSB2NvbnRleHQSUgoQY2FuY2VsX29wZXJhdGlvbhgKIAEoCzIlLmFueXR0eS5h'
    'cGkudjEuQ2FuY2VsT3BlcmF0aW9uQ29tbWFuZEgAUg9jYW5jZWxPcGVyYXRpb24SUgoQcmVsZW'
    'FzZV9yZXNvdXJjZRgLIAEoCzIlLmFueXR0eS5hcGkudjEuUmVsZWFzZVJlc291cmNlQ29tbWFu'
    'ZEgAUg9yZWxlYXNlUmVzb3VyY2USVQoRdGVybWluYWxfZGVmYXVsdHMYFCABKAsyJi5hbnl0dH'
    'kuYXBpLnYxLlRlcm1pbmFsRGVmYXVsdHNDb21tYW5kSABSEHRlcm1pbmFsRGVmYXVsdHMSTwoP'
    'dGVybWluYWxfY3JlYXRlGBUgASgLMiQuYW55dHR5LmFwaS52MS5UZXJtaW5hbENyZWF0ZUNvbW'
    '1hbmRIAFIOdGVybWluYWxDcmVhdGUSSQoNdGVybWluYWxfbGlzdBgWIAEoCzIiLmFueXR0eS5h'
    'cGkudjEuVGVybWluYWxMaXN0Q29tbWFuZEgAUgx0ZXJtaW5hbExpc3QSRgoMdGVybWluYWxfZ2'
    'V0GBcgASgLMiEuYW55dHR5LmFwaS52MS5UZXJtaW5hbEdldENvbW1hbmRIAFILdGVybWluYWxH'
    'ZXQSUgoQdGVybWluYWxfcmVzdGFydBgYIAEoCzIlLmFueXR0eS5hcGkudjEuVGVybWluYWxSZX'
    'N0YXJ0Q29tbWFuZEgAUg90ZXJtaW5hbFJlc3RhcnQSSQoNdGVybWluYWxfa2lsbBgZIAEoCzIi'
    'LmFueXR0eS5hcGkudjEuVGVybWluYWxLaWxsQ29tbWFuZEgAUgx0ZXJtaW5hbEtpbGwSTwoPdG'
    'VybWluYWxfcmVtb3ZlGBogASgLMiQuYW55dHR5LmFwaS52MS5UZXJtaW5hbFJlbW92ZUNvbW1h'
    'bmRIAFIOdGVybWluYWxSZW1vdmUSXwoVdGVybWluYWxfc2V0X21ldGFkYXRhGBsgASgLMikuYW'
    '55dHR5LmFwaS52MS5UZXJtaW5hbFNldE1ldGFkYXRhQ29tbWFuZEgAUhN0ZXJtaW5hbFNldE1l'
    'dGFkYXRhElMKEXRlcm1pbmFsX3NldF90YWdzGBwgASgLMiUuYW55dHR5LmFwaS52MS5UZXJtaW'
    '5hbFNldFRhZ3NDb21tYW5kSABSD3Rlcm1pbmFsU2V0VGFncxJPCg90ZXJtaW5hbF9hdHRhY2gY'
    'HSABKAsyJC5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsQXR0YWNoQ29tbWFuZEgAUg50ZXJtaW5hbE'
    'F0dGFjaBJPCg90ZXJtaW5hbF9kZXRhY2gYHiABKAsyJC5hbnl0dHkuYXBpLnYxLlRlcm1pbmFs'
    'RGV0YWNoQ29tbWFuZEgAUg50ZXJtaW5hbERldGFjaBJMCg50ZXJtaW5hbF9pbnB1dBgfIAEoCz'
    'IjLmFueXR0eS5hcGkudjEuVGVybWluYWxJbnB1dENvbW1hbmRIAFINdGVybWluYWxJbnB1dBJP'
    'Cg90ZXJtaW5hbF9yZXNpemUYICABKAsyJC5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsUmVzaXplQ2'
    '9tbWFuZEgAUg50ZXJtaW5hbFJlc2l6ZRJcChR0ZXJtaW5hbF9yZXNpemVfbG9jaxghIAEoCzIo'
    'LmFueXR0eS5hcGkudjEuVGVybWluYWxSZXNpemVMb2NrQ29tbWFuZEgAUhJ0ZXJtaW5hbFJlc2'
    'l6ZUxvY2sSXwoVcGF0aF9saXN0X2RpcmVjdG9yaWVzGCIgASgLMikuYW55dHR5LmFwaS52MS5Q'
    'YXRoTGlzdERpcmVjdG9yaWVzQ29tbWFuZEgAUhNwYXRoTGlzdERpcmVjdG9yaWVzEkwKDmhpc3'
    'Rvcnlfd2luZG93GCggASgLMiMuYW55dHR5LmFwaS52MS5IaXN0b3J5V2luZG93Q29tbWFuZEgA'
    'Ug1oaXN0b3J5V2luZG93EkYKDGhpc3RvcnlfY29weRgpIAEoCzIhLmFueXR0eS5hcGkudjEuSG'
    'lzdG9yeUNvcHlDb21tYW5kSABSC2hpc3RvcnlDb3B5Ek8KD2hpc3RvcnlfcmVsZWFzZRgqIAEo'
    'CzIkLmFueXR0eS5hcGkudjEuSGlzdG9yeVJlbGVhc2VDb21tYW5kSABSDmhpc3RvcnlSZWxlYX'
    'NlEmIKFmhpc3RvcnlfYmFja2xvZ19zdGF0dXMYKyABKAsyKi5hbnl0dHkuYXBpLnYxLkhpc3Rv'
    'cnlCYWNrbG9nU3RhdHVzQ29tbWFuZEgAUhRoaXN0b3J5QmFja2xvZ1N0YXR1cxJQChBsaXZlX3'
    'NjcmVlbl9uZXh0GCwgASgLMiQuYW55dHR5LmFwaS52MS5MaXZlU2NyZWVuTmV4dENvbW1hbmRI'
    'AFIObGl2ZVNjcmVlbk5leHQSTAoOaGlzdG9yeV9zZWFyY2gYLSABKAsyIy5hbnl0dHkuYXBpLn'
    'YxLkhpc3RvcnlTZWFyY2hDb21tYW5kSABSDWhpc3RvcnlTZWFyY2gSTwoPZXZlbnRfc3Vic2Ny'
    'aWJlGC4gASgLMiQuYW55dHR5LmFwaS52MS5FdmVudFN1YnNjcmliZUNvbW1hbmRIAFIOZXZlbn'
    'RTdWJzY3JpYmUSPQoJZmlsZV9saXN0GDwgASgLMh4uYW55dHR5LmFwaS52MS5GaWxlTGlzdENv'
    'bW1hbmRIAFIIZmlsZUxpc3QSPQoJZmlsZV9zdGF0GD0gASgLMh4uYW55dHR5LmFwaS52MS5GaW'
    'xlU3RhdENvbW1hbmRIAFIIZmlsZVN0YXQSRgoMZmlsZV9wcmV2aWV3GD4gASgLMiEuYW55dHR5'
    'LmFwaS52MS5GaWxlUHJldmlld0NvbW1hbmRIAFILZmlsZVByZXZpZXcSQAoKZmlsZV9ta2Rpch'
    'g/IAEoCzIfLmFueXR0eS5hcGkudjEuRmlsZU1rZGlyQ29tbWFuZEgAUglmaWxlTWtkaXISQwoL'
    'ZmlsZV9yZW5hbWUYQCABKAsyIC5hbnl0dHkuYXBpLnYxLkZpbGVSZW5hbWVDb21tYW5kSABSCm'
    'ZpbGVSZW5hbWUSQwoLZmlsZV9kZWxldGUYQSABKAsyIC5hbnl0dHkuYXBpLnYxLkZpbGVEZWxl'
    'dGVDb21tYW5kSABSCmZpbGVEZWxldGUSPQoJZmlsZV9jb3B5GEIgASgLMh4uYW55dHR5LmFwaS'
    '52MS5GaWxlQ29weUNvbW1hbmRIAFIIZmlsZUNvcHkSPQoJZmlsZV9tb3ZlGEMgASgLMh4uYW55'
    'dHR5LmFwaS52MS5GaWxlTW92ZUNvbW1hbmRIAFIIZmlsZU1vdmUSVgoSZmlsZV9kb3dubG9hZF'
    '9vcGVuGEQgASgLMiYuYW55dHR5LmFwaS52MS5GaWxlRG93bmxvYWRPcGVuQ29tbWFuZEgAUhBm'
    'aWxlRG93bmxvYWRPcGVuElAKEGZpbGVfdXBsb2FkX29wZW4YRSABKAsyJC5hbnl0dHkuYXBpLn'
    'YxLkZpbGVVcGxvYWRPcGVuQ29tbWFuZEgAUg5maWxlVXBsb2FkT3BlbhJcChRmaWxlX3RyYW5z'
    'ZmVyX2NhbmNlbBhGIAEoCzIoLmFueXR0eS5hcGkudjEuRmlsZVRyYW5zZmVyQ2FuY2VsQ29tbW'
    'FuZEgAUhJmaWxlVHJhbnNmZXJDYW5jZWwSQwoLc3RvcmFnZV9nZXQYUCABKAsyIC5hbnl0dHku'
    'YXBpLnYxLlN0b3JhZ2VHZXRDb21tYW5kSABSCnN0b3JhZ2VHZXQSQwoLc3RvcmFnZV9wdXQYUS'
    'ABKAsyIC5hbnl0dHkuYXBpLnYxLlN0b3JhZ2VQdXRDb21tYW5kSABSCnN0b3JhZ2VQdXQSTAoO'
    'c3RvcmFnZV9kZWxldGUYUiABKAsyIy5hbnl0dHkuYXBpLnYxLlN0b3JhZ2VEZWxldGVDb21tYW'
    '5kSABSDXN0b3JhZ2VEZWxldGUSRgoMc3RvcmFnZV9saXN0GFMgASgLMiEuYW55dHR5LmFwaS52'
    'MS5TdG9yYWdlTGlzdENvbW1hbmRIAFILc3RvcmFnZUxpc3QSYgoWY2xpZW50X2FjY2Vzc19pZG'
    'VudGl0eRhkIAEoCzIqLmFueXR0eS5hcGkudjEuQ2xpZW50QWNjZXNzSWRlbnRpdHlDb21tYW5k'
    'SABSFGNsaWVudEFjY2Vzc0lkZW50aXR5ElYKEmNsaWVudF9hY2Nlc3NfbGlzdBhlIAEoCzImLm'
    'FueXR0eS5hcGkudjEuQ2xpZW50QWNjZXNzTGlzdENvbW1hbmRIAFIQY2xpZW50QWNjZXNzTGlz'
    'dBJvChtjbGllbnRfYWNjZXNzX3RpY2tldF9jcmVhdGUYZiABKAsyLi5hbnl0dHkuYXBpLnYxLk'
    'NsaWVudEFjY2Vzc1RpY2tldENyZWF0ZUNvbW1hbmRIAFIYY2xpZW50QWNjZXNzVGlja2V0Q3Jl'
    'YXRlElwKFGNsaWVudF9hY2Nlc3NfcmV2b2tlGGcgASgLMiguYW55dHR5LmFwaS52MS5DbGllbn'
    'RBY2Nlc3NSZXZva2VDb21tYW5kSABSEmNsaWVudEFjY2Vzc1Jldm9rZRJJCg1yZW1vdGVfc3Rh'
    'dHVzGG4gASgLMiIuYW55dHR5LmFwaS52MS5SZW1vdGVTdGF0dXNDb21tYW5kSABSDHJlbW90ZV'
    'N0YXR1cxJTChFyZW1vdGVfcGFpcl9zdGFydBhvIAEoCzIlLmFueXR0eS5hcGkudjEuUmVtb3Rl'
    'UGFpclN0YXJ0Q29tbWFuZEgAUg9yZW1vdGVQYWlyU3RhcnQSWQoTcmVtb3RlX2xvY2FsX2VuYW'
    'JsZRhwIAEoCzInLmFueXR0eS5hcGkudjEuUmVtb3RlTG9jYWxFbmFibGVDb21tYW5kSABSEXJl'
    'bW90ZUxvY2FsRW5hYmxlElkKE3JlbW90ZV9sb2NhbF9zdGF0dXMYcSABKAsyJy5hbnl0dHkuYX'
    'BpLnYxLlJlbW90ZUxvY2FsU3RhdHVzQ29tbWFuZEgAUhFyZW1vdGVMb2NhbFN0YXR1cxJcChRy'
    'ZW1vdGVfbG9jYWxfZGlzYWJsZRhyIAEoCzIoLmFueXR0eS5hcGkudjEuUmVtb3RlTG9jYWxEaX'
    'NhYmxlQ29tbWFuZEgAUhJyZW1vdGVMb2NhbERpc2FibGUSVgoScmVtb3RlX2Nsb3VkX2VkZ2Vz'
    'GHMgASgLMiYuYW55dHR5LmFwaS52MS5SZW1vdGVDbG91ZEVkZ2VzQ29tbWFuZEgAUhByZW1vdG'
    'VDbG91ZEVkZ2VzEmYKGHJlbW90ZV9jbG91ZF9wcmVmZXJfZWRnZRh0IAEoCzIrLmFueXR0eS5h'
    'cGkudjEuUmVtb3RlQ2xvdWRQcmVmZXJFZGdlQ29tbWFuZEgAUhVyZW1vdGVDbG91ZFByZWZlck'
    'VkZ2USbAoacmVtb3RlX2Nsb3VkX3Jlc2VsZWN0X2VkZ2UYdSABKAsyLS5hbnl0dHkuYXBpLnYx'
    'LlJlbW90ZUNsb3VkUmVzZWxlY3RFZGdlQ29tbWFuZEgAUhdyZW1vdGVDbG91ZFJlc2VsZWN0RW'
    'RnZRJZChNyZW1vdGVfY2xvdWRfc3RhdHVzGHYgASgLMicuYW55dHR5LmFwaS52MS5SZW1vdGVD'
    'bG91ZFN0YXR1c0NvbW1hbmRIAFIRcmVtb3RlQ2xvdWRTdGF0dXMSWQoTcmVtb3RlX2Nsb3VkX2'
    'VuYWJsZRh3IAEoCzInLmFueXR0eS5hcGkudjEuUmVtb3RlQ2xvdWRFbmFibGVDb21tYW5kSABS'
    'EXJlbW90ZUNsb3VkRW5hYmxlElwKFHJlbW90ZV9jbG91ZF9kaXNhYmxlGHggASgLMiguYW55dH'
    'R5LmFwaS52MS5SZW1vdGVDbG91ZERpc2FibGVDb21tYW5kSABSEnJlbW90ZUNsb3VkRGlzYWJs'
    'ZRJWChJicm93c2VyX3Byb3h5X29wZW4YeSABKAsyJi5hbnl0dHkuYXBpLnYxLkJyb3dzZXJQcm'
    '94eU9wZW5Db21tYW5kSABSEGJyb3dzZXJQcm94eU9wZW5CCQoHY29tbWFuZA==');

@$core.Deprecated('Use acknowledgeResultDescriptor instead')
const AcknowledgeResult$json = {
  '1': 'AcknowledgeResult',
};

/// Descriptor for `AcknowledgeResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List acknowledgeResultDescriptor =
    $convert.base64Decode('ChFBY2tub3dsZWRnZVJlc3VsdA==');

@$core.Deprecated('Use resultEnvelopeDescriptor instead')
const ResultEnvelope$json = {
  '1': 'ResultEnvelope',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {
      '1': 'origin_session',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'originSession'
    },
    {
      '1': 'acknowledge',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.AcknowledgeResult',
      '9': 0,
      '10': 'acknowledge'
    },
    {
      '1': 'error',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiError',
      '9': 0,
      '10': 'error'
    },
    {
      '1': 'terminal_defaults',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalDefaultsResult',
      '9': 0,
      '10': 'terminalDefaults'
    },
    {
      '1': 'terminal_create',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalCreateResult',
      '9': 0,
      '10': 'terminalCreate'
    },
    {
      '1': 'terminal_list',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalListResult',
      '9': 0,
      '10': 'terminalList'
    },
    {
      '1': 'terminal_get',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalGetResult',
      '9': 0,
      '10': 'terminalGet'
    },
    {
      '1': 'terminal_attach',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalAttachResult',
      '9': 0,
      '10': 'terminalAttach'
    },
    {
      '1': 'terminal_resize',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalResizeResult',
      '9': 0,
      '10': 'terminalResize'
    },
    {
      '1': 'path_list_directories',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.PathListDirectoriesResult',
      '9': 0,
      '10': 'pathListDirectories'
    },
    {
      '1': 'history_window',
      '3': 40,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryWindowResult',
      '9': 0,
      '10': 'historyWindow'
    },
    {
      '1': 'history_copy',
      '3': 41,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryCopyResult',
      '9': 0,
      '10': 'historyCopy'
    },
    {
      '1': 'history_backlog_status',
      '3': 42,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistoryBacklogStatusResult',
      '9': 0,
      '10': 'historyBacklogStatus'
    },
    {
      '1': 'live_screen',
      '3': 43,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.NativeScreenResult',
      '9': 0,
      '10': 'liveScreen'
    },
    {
      '1': 'history_search',
      '3': 44,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.HistorySearchResult',
      '9': 0,
      '10': 'historySearch'
    },
    {
      '1': 'event_subscription',
      '3': 45,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EventSubscriptionResult',
      '9': 0,
      '10': 'eventSubscription'
    },
    {
      '1': 'file_list',
      '3': 60,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileListResult',
      '9': 0,
      '10': 'fileList'
    },
    {
      '1': 'file_stat',
      '3': 61,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileStatResult',
      '9': 0,
      '10': 'fileStat'
    },
    {
      '1': 'file_preview',
      '3': 62,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FilePreviewResult',
      '9': 0,
      '10': 'filePreview'
    },
    {
      '1': 'file_operation',
      '3': 63,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileOperationResult',
      '9': 0,
      '10': 'fileOperation'
    },
    {
      '1': 'file_batch',
      '3': 64,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileBatchResult',
      '9': 0,
      '10': 'fileBatch'
    },
    {
      '1': 'file_transfer_open',
      '3': 65,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileTransferOpenResult',
      '9': 0,
      '10': 'fileTransferOpen'
    },
    {
      '1': 'file_transfer_cancel',
      '3': 66,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileTransferCancelResult',
      '9': 0,
      '10': 'fileTransferCancel'
    },
    {
      '1': 'storage_get',
      '3': 80,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageGetResult',
      '9': 0,
      '10': 'storageGet'
    },
    {
      '1': 'storage_put',
      '3': 81,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StoragePutResult',
      '9': 0,
      '10': 'storagePut'
    },
    {
      '1': 'storage_delete',
      '3': 82,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageDeleteResult',
      '9': 0,
      '10': 'storageDelete'
    },
    {
      '1': 'storage_list',
      '3': 83,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageListResult',
      '9': 0,
      '10': 'storageList'
    },
    {
      '1': 'client_access_identity',
      '3': 100,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessIdentityResult',
      '9': 0,
      '10': 'clientAccessIdentity'
    },
    {
      '1': 'client_access_list',
      '3': 101,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessListResult',
      '9': 0,
      '10': 'clientAccessList'
    },
    {
      '1': 'client_access_ticket_create',
      '3': 102,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessTicketCreateResult',
      '9': 0,
      '10': 'clientAccessTicketCreate'
    },
    {
      '1': 'client_access_revoke',
      '3': 103,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ClientAccessRevokeResult',
      '9': 0,
      '10': 'clientAccessRevoke'
    },
    {
      '1': 'remote_status',
      '3': 110,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteStatusResult',
      '9': 0,
      '10': 'remoteStatus'
    },
    {
      '1': 'remote_pair_start',
      '3': 111,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemotePairStartResult',
      '9': 0,
      '10': 'remotePairStart'
    },
    {
      '1': 'remote_local_status',
      '3': 112,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteLocalStatusResult',
      '9': 0,
      '10': 'remoteLocalStatus'
    },
    {
      '1': 'remote_cloud_edges',
      '3': 113,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudEdgesResult',
      '9': 0,
      '10': 'remoteCloudEdges'
    },
    {
      '1': 'remote_cloud_status',
      '3': 114,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.RemoteCloudStatusResult',
      '9': 0,
      '10': 'remoteCloudStatus'
    },
    {
      '1': 'browser_proxy_open',
      '3': 115,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.BrowserProxyOpenResult',
      '9': 0,
      '10': 'browserProxyOpen'
    },
  ],
  '8': [
    {'1': 'result'},
  ],
};

/// Descriptor for `ResultEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resultEnvelopeDescriptor = $convert.base64Decode(
    'Cg5SZXN1bHRFbnZlbG9wZRIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSSgoOb3JpZ2'
    'luX3Nlc3Npb24YAiABKAsyIy5hbnl0dHkuYXBpLnYxLkVuZHBvaW50U2Vzc2lvblN0YW1wUg1v'
    'cmlnaW5TZXNzaW9uEkQKC2Fja25vd2xlZGdlGAogASgLMiAuYW55dHR5LmFwaS52MS5BY2tub3'
    'dsZWRnZVJlc3VsdEgAUgthY2tub3dsZWRnZRIvCgVlcnJvchgLIAEoCzIXLmFueXR0eS5hcGku'
    'djEuQXBpRXJyb3JIAFIFZXJyb3ISVAoRdGVybWluYWxfZGVmYXVsdHMYFCABKAsyJS5hbnl0dH'
    'kuYXBpLnYxLlRlcm1pbmFsRGVmYXVsdHNSZXN1bHRIAFIQdGVybWluYWxEZWZhdWx0cxJOCg90'
    'ZXJtaW5hbF9jcmVhdGUYFSABKAsyIy5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsQ3JlYXRlUmVzdW'
    'x0SABSDnRlcm1pbmFsQ3JlYXRlEkgKDXRlcm1pbmFsX2xpc3QYFiABKAsyIS5hbnl0dHkuYXBp'
    'LnYxLlRlcm1pbmFsTGlzdFJlc3VsdEgAUgx0ZXJtaW5hbExpc3QSRQoMdGVybWluYWxfZ2V0GB'
    'cgASgLMiAuYW55dHR5LmFwaS52MS5UZXJtaW5hbEdldFJlc3VsdEgAUgt0ZXJtaW5hbEdldBJO'
    'Cg90ZXJtaW5hbF9hdHRhY2gYGCABKAsyIy5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsQXR0YWNoUm'
    'VzdWx0SABSDnRlcm1pbmFsQXR0YWNoEk4KD3Rlcm1pbmFsX3Jlc2l6ZRgZIAEoCzIjLmFueXR0'
    'eS5hcGkudjEuVGVybWluYWxSZXNpemVSZXN1bHRIAFIOdGVybWluYWxSZXNpemUSXgoVcGF0aF'
    '9saXN0X2RpcmVjdG9yaWVzGBogASgLMiguYW55dHR5LmFwaS52MS5QYXRoTGlzdERpcmVjdG9y'
    'aWVzUmVzdWx0SABSE3BhdGhMaXN0RGlyZWN0b3JpZXMSSwoOaGlzdG9yeV93aW5kb3cYKCABKA'
    'syIi5hbnl0dHkuYXBpLnYxLkhpc3RvcnlXaW5kb3dSZXN1bHRIAFINaGlzdG9yeVdpbmRvdxJF'
    'CgxoaXN0b3J5X2NvcHkYKSABKAsyIC5hbnl0dHkuYXBpLnYxLkhpc3RvcnlDb3B5UmVzdWx0SA'
    'BSC2hpc3RvcnlDb3B5EmEKFmhpc3RvcnlfYmFja2xvZ19zdGF0dXMYKiABKAsyKS5hbnl0dHku'
    'YXBpLnYxLkhpc3RvcnlCYWNrbG9nU3RhdHVzUmVzdWx0SABSFGhpc3RvcnlCYWNrbG9nU3RhdH'
    'VzEkQKC2xpdmVfc2NyZWVuGCsgASgLMiEuYW55dHR5LmFwaS52MS5OYXRpdmVTY3JlZW5SZXN1'
    'bHRIAFIKbGl2ZVNjcmVlbhJLCg5oaXN0b3J5X3NlYXJjaBgsIAEoCzIiLmFueXR0eS5hcGkudj'
    'EuSGlzdG9yeVNlYXJjaFJlc3VsdEgAUg1oaXN0b3J5U2VhcmNoElcKEmV2ZW50X3N1YnNjcmlw'
    'dGlvbhgtIAEoCzImLmFueXR0eS5hcGkudjEuRXZlbnRTdWJzY3JpcHRpb25SZXN1bHRIAFIRZX'
    'ZlbnRTdWJzY3JpcHRpb24SPAoJZmlsZV9saXN0GDwgASgLMh0uYW55dHR5LmFwaS52MS5GaWxl'
    'TGlzdFJlc3VsdEgAUghmaWxlTGlzdBI8CglmaWxlX3N0YXQYPSABKAsyHS5hbnl0dHkuYXBpLn'
    'YxLkZpbGVTdGF0UmVzdWx0SABSCGZpbGVTdGF0EkUKDGZpbGVfcHJldmlldxg+IAEoCzIgLmFu'
    'eXR0eS5hcGkudjEuRmlsZVByZXZpZXdSZXN1bHRIAFILZmlsZVByZXZpZXcSSwoOZmlsZV9vcG'
    'VyYXRpb24YPyABKAsyIi5hbnl0dHkuYXBpLnYxLkZpbGVPcGVyYXRpb25SZXN1bHRIAFINZmls'
    'ZU9wZXJhdGlvbhI/CgpmaWxlX2JhdGNoGEAgASgLMh4uYW55dHR5LmFwaS52MS5GaWxlQmF0Y2'
    'hSZXN1bHRIAFIJZmlsZUJhdGNoElUKEmZpbGVfdHJhbnNmZXJfb3BlbhhBIAEoCzIlLmFueXR0'
    'eS5hcGkudjEuRmlsZVRyYW5zZmVyT3BlblJlc3VsdEgAUhBmaWxlVHJhbnNmZXJPcGVuElsKFG'
    'ZpbGVfdHJhbnNmZXJfY2FuY2VsGEIgASgLMicuYW55dHR5LmFwaS52MS5GaWxlVHJhbnNmZXJD'
    'YW5jZWxSZXN1bHRIAFISZmlsZVRyYW5zZmVyQ2FuY2VsEkIKC3N0b3JhZ2VfZ2V0GFAgASgLMh'
    '8uYW55dHR5LmFwaS52MS5TdG9yYWdlR2V0UmVzdWx0SABSCnN0b3JhZ2VHZXQSQgoLc3RvcmFn'
    'ZV9wdXQYUSABKAsyHy5hbnl0dHkuYXBpLnYxLlN0b3JhZ2VQdXRSZXN1bHRIAFIKc3RvcmFnZV'
    'B1dBJLCg5zdG9yYWdlX2RlbGV0ZRhSIAEoCzIiLmFueXR0eS5hcGkudjEuU3RvcmFnZURlbGV0'
    'ZVJlc3VsdEgAUg1zdG9yYWdlRGVsZXRlEkUKDHN0b3JhZ2VfbGlzdBhTIAEoCzIgLmFueXR0eS'
    '5hcGkudjEuU3RvcmFnZUxpc3RSZXN1bHRIAFILc3RvcmFnZUxpc3QSYQoWY2xpZW50X2FjY2Vz'
    'c19pZGVudGl0eRhkIAEoCzIpLmFueXR0eS5hcGkudjEuQ2xpZW50QWNjZXNzSWRlbnRpdHlSZX'
    'N1bHRIAFIUY2xpZW50QWNjZXNzSWRlbnRpdHkSVQoSY2xpZW50X2FjY2Vzc19saXN0GGUgASgL'
    'MiUuYW55dHR5LmFwaS52MS5DbGllbnRBY2Nlc3NMaXN0UmVzdWx0SABSEGNsaWVudEFjY2Vzc0'
    'xpc3QSbgobY2xpZW50X2FjY2Vzc190aWNrZXRfY3JlYXRlGGYgASgLMi0uYW55dHR5LmFwaS52'
    'MS5DbGllbnRBY2Nlc3NUaWNrZXRDcmVhdGVSZXN1bHRIAFIYY2xpZW50QWNjZXNzVGlja2V0Q3'
    'JlYXRlElsKFGNsaWVudF9hY2Nlc3NfcmV2b2tlGGcgASgLMicuYW55dHR5LmFwaS52MS5DbGll'
    'bnRBY2Nlc3NSZXZva2VSZXN1bHRIAFISY2xpZW50QWNjZXNzUmV2b2tlEkgKDXJlbW90ZV9zdG'
    'F0dXMYbiABKAsyIS5hbnl0dHkuYXBpLnYxLlJlbW90ZVN0YXR1c1Jlc3VsdEgAUgxyZW1vdGVT'
    'dGF0dXMSUgoRcmVtb3RlX3BhaXJfc3RhcnQYbyABKAsyJC5hbnl0dHkuYXBpLnYxLlJlbW90ZV'
    'BhaXJTdGFydFJlc3VsdEgAUg9yZW1vdGVQYWlyU3RhcnQSWAoTcmVtb3RlX2xvY2FsX3N0YXR1'
    'cxhwIAEoCzImLmFueXR0eS5hcGkudjEuUmVtb3RlTG9jYWxTdGF0dXNSZXN1bHRIAFIRcmVtb3'
    'RlTG9jYWxTdGF0dXMSVQoScmVtb3RlX2Nsb3VkX2VkZ2VzGHEgASgLMiUuYW55dHR5LmFwaS52'
    'MS5SZW1vdGVDbG91ZEVkZ2VzUmVzdWx0SABSEHJlbW90ZUNsb3VkRWRnZXMSWAoTcmVtb3RlX2'
    'Nsb3VkX3N0YXR1cxhyIAEoCzImLmFueXR0eS5hcGkudjEuUmVtb3RlQ2xvdWRTdGF0dXNSZXN1'
    'bHRIAFIRcmVtb3RlQ2xvdWRTdGF0dXMSVQoSYnJvd3Nlcl9wcm94eV9vcGVuGHMgASgLMiUuYW'
    '55dHR5LmFwaS52MS5Ccm93c2VyUHJveHlPcGVuUmVzdWx0SABSEGJyb3dzZXJQcm94eU9wZW5C'
    'CAoGcmVzdWx0');

@$core.Deprecated('Use browserProxyOpenResultDescriptor instead')
const BrowserProxyOpenResult$json = {
  '1': 'BrowserProxyOpenResult',
  '2': [
    {
      '1': 'resource',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
  ],
};

/// Descriptor for `BrowserProxyOpenResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List browserProxyOpenResultDescriptor =
    $convert.base64Decode(
        'ChZCcm93c2VyUHJveHlPcGVuUmVzdWx0EjkKCHJlc291cmNlGAEgASgLMh0uYW55dHR5LmFwaS'
        '52MS5SZXNvdXJjZUhhbmRsZVIIcmVzb3VyY2U=');

@$core.Deprecated('Use operationCancelledEventDescriptor instead')
const OperationCancelledEvent$json = {
  '1': 'OperationCancelledEvent',
  '2': [
    {
      '1': 'operation',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
  ],
};

/// Descriptor for `OperationCancelledEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operationCancelledEventDescriptor =
    $convert.base64Decode(
        'ChdPcGVyYXRpb25DYW5jZWxsZWRFdmVudBI7CglvcGVyYXRpb24YASABKAsyHS5hbnl0dHkuYX'
        'BpLnYxLk9wZXJhdGlvblN0YW1wUglvcGVyYXRpb24=');

@$core.Deprecated('Use resourceReleasedEventDescriptor instead')
const ResourceReleasedEvent$json = {
  '1': 'ResourceReleasedEvent',
  '2': [
    {
      '1': 'resource',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
  ],
};

/// Descriptor for `ResourceReleasedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resourceReleasedEventDescriptor = $convert.base64Decode(
    'ChVSZXNvdXJjZVJlbGVhc2VkRXZlbnQSOQoIcmVzb3VyY2UYASABKAsyHS5hbnl0dHkuYXBpLn'
    'YxLlJlc291cmNlSGFuZGxlUghyZXNvdXJjZQ==');

@$core.Deprecated('Use eventEnvelopeDescriptor instead')
const EventEnvelope$json = {
  '1': 'EventEnvelope',
  '2': [
    {'1': 'event_id', '3': 1, '4': 1, '5': 9, '10': 'eventId'},
    {
      '1': 'timestamp_unix_nano',
      '3': 2,
      '4': 1,
      '5': 3,
      '10': 'timestampUnixNano'
    },
    {
      '1': 'api_version',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ApiVersion',
      '10': 'apiVersion'
    },
    {
      '1': 'origin_session',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.EndpointSessionStamp',
      '10': 'originSession'
    },
    {
      '1': 'subscription',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'subscription'
    },
    {
      '1': 'operation_cancelled',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationCancelledEvent',
      '9': 0,
      '10': 'operationCancelled'
    },
    {
      '1': 'resource_released',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceReleasedEvent',
      '9': 0,
      '10': 'resourceReleased'
    },
    {
      '1': 'terminal_lifecycle',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.TerminalLifecycleEvent',
      '9': 0,
      '10': 'terminalLifecycle'
    },
    {
      '1': 'storage_changed',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.StorageChangedEvent',
      '9': 0,
      '10': 'storageChanged'
    },
    {
      '1': 'file_transfer_completed',
      '3': 40,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileTransferCompletedEvent',
      '9': 0,
      '10': 'fileTransferCompleted'
    },
  ],
  '8': [
    {'1': 'event'},
  ],
  '9': [
    {'1': 21, '2': 22},
    {'1': 22, '2': 23},
    {'1': 23, '2': 24},
  ],
};

/// Descriptor for `EventEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eventEnvelopeDescriptor = $convert.base64Decode(
    'Cg1FdmVudEVudmVsb3BlEhkKCGV2ZW50X2lkGAEgASgJUgdldmVudElkEi4KE3RpbWVzdGFtcF'
    '91bml4X25hbm8YAiABKANSEXRpbWVzdGFtcFVuaXhOYW5vEjoKC2FwaV92ZXJzaW9uGAMgASgL'
    'MhkuYW55dHR5LmFwaS52MS5BcGlWZXJzaW9uUgphcGlWZXJzaW9uEkoKDm9yaWdpbl9zZXNzaW'
    '9uGAQgASgLMiMuYW55dHR5LmFwaS52MS5FbmRwb2ludFNlc3Npb25TdGFtcFINb3JpZ2luU2Vz'
    'c2lvbhJBCgxzdWJzY3JpcHRpb24YBSABKAsyHS5hbnl0dHkuYXBpLnYxLlJlc291cmNlSGFuZG'
    'xlUgxzdWJzY3JpcHRpb24SWQoTb3BlcmF0aW9uX2NhbmNlbGxlZBgKIAEoCzImLmFueXR0eS5h'
    'cGkudjEuT3BlcmF0aW9uQ2FuY2VsbGVkRXZlbnRIAFISb3BlcmF0aW9uQ2FuY2VsbGVkElMKEX'
    'Jlc291cmNlX3JlbGVhc2VkGAsgASgLMiQuYW55dHR5LmFwaS52MS5SZXNvdXJjZVJlbGVhc2Vk'
    'RXZlbnRIAFIQcmVzb3VyY2VSZWxlYXNlZBJWChJ0ZXJtaW5hbF9saWZlY3ljbGUYFCABKAsyJS'
    '5hbnl0dHkuYXBpLnYxLlRlcm1pbmFsTGlmZWN5Y2xlRXZlbnRIAFIRdGVybWluYWxMaWZlY3lj'
    'bGUSTQoPc3RvcmFnZV9jaGFuZ2VkGB4gASgLMiIuYW55dHR5LmFwaS52MS5TdG9yYWdlQ2hhbm'
    'dlZEV2ZW50SABSDnN0b3JhZ2VDaGFuZ2VkEmMKF2ZpbGVfdHJhbnNmZXJfY29tcGxldGVkGCgg'
    'ASgLMikuYW55dHR5LmFwaS52MS5GaWxlVHJhbnNmZXJDb21wbGV0ZWRFdmVudEgAUhVmaWxlVH'
    'JhbnNmZXJDb21wbGV0ZWRCBwoFZXZlbnRKBAgVEBZKBAgWEBdKBAgXEBg=');
