// This is a generated file - do not edit.
//
// Generated from apipb/file.proto.

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

@$core.Deprecated('Use fileEntryTypeDescriptor instead')
const FileEntryType$json = {
  '1': 'FileEntryType',
  '2': [
    {'1': 'FILE_ENTRY_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'FILE_ENTRY_TYPE_FILE', '2': 1},
    {'1': 'FILE_ENTRY_TYPE_DIRECTORY', '2': 2},
    {'1': 'FILE_ENTRY_TYPE_SYMLINK', '2': 3},
    {'1': 'FILE_ENTRY_TYPE_OTHER', '2': 4},
  ],
};

/// Descriptor for `FileEntryType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List fileEntryTypeDescriptor = $convert.base64Decode(
    'Cg1GaWxlRW50cnlUeXBlEh8KG0ZJTEVfRU5UUllfVFlQRV9VTlNQRUNJRklFRBAAEhgKFEZJTE'
    'VfRU5UUllfVFlQRV9GSUxFEAESHQoZRklMRV9FTlRSWV9UWVBFX0RJUkVDVE9SWRACEhsKF0ZJ'
    'TEVfRU5UUllfVFlQRV9TWU1MSU5LEAMSGQoVRklMRV9FTlRSWV9UWVBFX09USEVSEAQ=');

@$core.Deprecated('Use fileEntryDescriptor instead')
const FileEntry$json = {
  '1': 'FileEntry',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'type',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.anytty.api.v1.FileEntryType',
      '10': 'type'
    },
    {'1': 'size', '3': 4, '4': 1, '5': 3, '10': 'size'},
    {'1': 'mode', '3': 5, '4': 1, '5': 13, '10': 'mode'},
    {
      '1': 'modified_at_unix_nano',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'modifiedAtUnixNano'
    },
    {'1': 'link_target', '3': 7, '4': 1, '5': 9, '10': 'linkTarget'},
  ],
};

/// Descriptor for `FileEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileEntryDescriptor = $convert.base64Decode(
    'CglGaWxlRW50cnkSEgoEcGF0aBgBIAEoCVIEcGF0aBISCgRuYW1lGAIgASgJUgRuYW1lEjAKBH'
    'R5cGUYAyABKA4yHC5hbnl0dHkuYXBpLnYxLkZpbGVFbnRyeVR5cGVSBHR5cGUSEgoEc2l6ZRgE'
    'IAEoA1IEc2l6ZRISCgRtb2RlGAUgASgNUgRtb2RlEjEKFW1vZGlmaWVkX2F0X3VuaXhfbmFubx'
    'gGIAEoA1ISbW9kaWZpZWRBdFVuaXhOYW5vEh8KC2xpbmtfdGFyZ2V0GAcgASgJUgpsaW5rVGFy'
    'Z2V0');

@$core.Deprecated('Use fileUploadResumeHandleDescriptor instead')
const FileUploadResumeHandle$json = {
  '1': 'FileUploadResumeHandle',
  '2': [
    {'1': 'opaque_token', '3': 1, '4': 1, '5': 12, '10': 'opaqueToken'},
  ],
};

/// Descriptor for `FileUploadResumeHandle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileUploadResumeHandleDescriptor =
    $convert.base64Decode(
        'ChZGaWxlVXBsb2FkUmVzdW1lSGFuZGxlEiEKDG9wYXF1ZV90b2tlbhgBIAEoDFILb3BhcXVlVG'
        '9rZW4=');

@$core.Deprecated('Use fileListCommandDescriptor instead')
const FileListCommand$json = {
  '1': 'FileListCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'cursor', '3': 3, '4': 1, '5': 9, '10': 'cursor'},
    {'1': 'limit', '3': 4, '4': 1, '5': 5, '10': 'limit'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileListCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileListCommandDescriptor = $convert.base64Decode(
    'Cg9GaWxlTGlzdENvbW1hbmQSEgoEcGF0aBgCIAEoCVIEcGF0aBIWCgZjdXJzb3IYAyABKAlSBm'
    'N1cnNvchIUCgVsaW1pdBgEIAEoBVIFbGltaXRKBAgBEAI=');

@$core.Deprecated('Use fileStatCommandDescriptor instead')
const FileStatCommand$json = {
  '1': 'FileStatCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileStatCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileStatCommandDescriptor = $convert.base64Decode(
    'Cg9GaWxlU3RhdENvbW1hbmQSEgoEcGF0aBgCIAEoCVIEcGF0aEoECAEQAg==');

@$core.Deprecated('Use filePreviewCommandDescriptor instead')
const FilePreviewCommand$json = {
  '1': 'FilePreviewCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'max_bytes', '3': 3, '4': 1, '5': 3, '10': 'maxBytes'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FilePreviewCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List filePreviewCommandDescriptor = $convert.base64Decode(
    'ChJGaWxlUHJldmlld0NvbW1hbmQSEgoEcGF0aBgCIAEoCVIEcGF0aBIbCgltYXhfYnl0ZXMYAy'
    'ABKANSCG1heEJ5dGVzSgQIARAC');

@$core.Deprecated('Use fileMkdirCommandDescriptor instead')
const FileMkdirCommand$json = {
  '1': 'FileMkdirCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'recursive', '3': 3, '4': 1, '5': 8, '10': 'recursive'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileMkdirCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileMkdirCommandDescriptor = $convert.base64Decode(
    'ChBGaWxlTWtkaXJDb21tYW5kEhIKBHBhdGgYAiABKAlSBHBhdGgSHAoJcmVjdXJzaXZlGAMgAS'
    'gIUglyZWN1cnNpdmVKBAgBEAI=');

@$core.Deprecated('Use fileRenameCommandDescriptor instead')
const FileRenameCommand$json = {
  '1': 'FileRenameCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'new_path', '3': 3, '4': 1, '5': 9, '10': 'newPath'},
    {'1': 'overwrite', '3': 4, '4': 1, '5': 8, '10': 'overwrite'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileRenameCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileRenameCommandDescriptor = $convert.base64Decode(
    'ChFGaWxlUmVuYW1lQ29tbWFuZBISCgRwYXRoGAIgASgJUgRwYXRoEhkKCG5ld19wYXRoGAMgAS'
    'gJUgduZXdQYXRoEhwKCW92ZXJ3cml0ZRgEIAEoCFIJb3ZlcndyaXRlSgQIARAC');

@$core.Deprecated('Use fileDeleteCommandDescriptor instead')
const FileDeleteCommand$json = {
  '1': 'FileDeleteCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'recursive', '3': 3, '4': 1, '5': 8, '10': 'recursive'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileDeleteCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileDeleteCommandDescriptor = $convert.base64Decode(
    'ChFGaWxlRGVsZXRlQ29tbWFuZBISCgRwYXRoGAIgASgJUgRwYXRoEhwKCXJlY3Vyc2l2ZRgDIA'
    'EoCFIJcmVjdXJzaXZlSgQIARAC');

@$core.Deprecated('Use fileCopyCommandDescriptor instead')
const FileCopyCommand$json = {
  '1': 'FileCopyCommand',
  '2': [
    {'1': 'paths', '3': 2, '4': 3, '5': 9, '10': 'paths'},
    {'1': 'target_directory', '3': 3, '4': 1, '5': 9, '10': 'targetDirectory'},
    {'1': 'overwrite', '3': 4, '4': 1, '5': 8, '10': 'overwrite'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileCopyCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileCopyCommandDescriptor = $convert.base64Decode(
    'Cg9GaWxlQ29weUNvbW1hbmQSFAoFcGF0aHMYAiADKAlSBXBhdGhzEikKEHRhcmdldF9kaXJlY3'
    'RvcnkYAyABKAlSD3RhcmdldERpcmVjdG9yeRIcCglvdmVyd3JpdGUYBCABKAhSCW92ZXJ3cml0'
    'ZUoECAEQAg==');

@$core.Deprecated('Use fileMoveCommandDescriptor instead')
const FileMoveCommand$json = {
  '1': 'FileMoveCommand',
  '2': [
    {'1': 'paths', '3': 2, '4': 3, '5': 9, '10': 'paths'},
    {'1': 'target_directory', '3': 3, '4': 1, '5': 9, '10': 'targetDirectory'},
    {'1': 'overwrite', '3': 4, '4': 1, '5': 8, '10': 'overwrite'},
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileMoveCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileMoveCommandDescriptor = $convert.base64Decode(
    'Cg9GaWxlTW92ZUNvbW1hbmQSFAoFcGF0aHMYAiADKAlSBXBhdGhzEikKEHRhcmdldF9kaXJlY3'
    'RvcnkYAyABKAlSD3RhcmdldERpcmVjdG9yeRIcCglvdmVyd3JpdGUYBCABKAhSCW92ZXJ3cml0'
    'ZUoECAEQAg==');

@$core.Deprecated('Use fileDownloadOpenCommandDescriptor instead')
const FileDownloadOpenCommand$json = {
  '1': 'FileDownloadOpenCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'offset', '3': 3, '4': 1, '5': 3, '10': 'offset'},
    {'1': 'expected_size', '3': 4, '4': 1, '5': 3, '10': 'expectedSize'},
    {
      '1': 'expected_modified_at_unix_nano',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'expectedModifiedAtUnixNano'
    },
    {
      '1': 'operation',
      '3': 6,
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

/// Descriptor for `FileDownloadOpenCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileDownloadOpenCommandDescriptor = $convert.base64Decode(
    'ChdGaWxlRG93bmxvYWRPcGVuQ29tbWFuZBISCgRwYXRoGAIgASgJUgRwYXRoEhYKBm9mZnNldB'
    'gDIAEoA1IGb2Zmc2V0EiMKDWV4cGVjdGVkX3NpemUYBCABKANSDGV4cGVjdGVkU2l6ZRJCCh5l'
    'eHBlY3RlZF9tb2RpZmllZF9hdF91bml4X25hbm8YBSABKANSGmV4cGVjdGVkTW9kaWZpZWRBdF'
    'VuaXhOYW5vEjsKCW9wZXJhdGlvbhgGIAEoCzIdLmFueXR0eS5hcGkudjEuT3BlcmF0aW9uU3Rh'
    'bXBSCW9wZXJhdGlvbkoECAEQAg==');

@$core.Deprecated('Use fileUploadOpenCommandDescriptor instead')
const FileUploadOpenCommand$json = {
  '1': 'FileUploadOpenCommand',
  '2': [
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'size', '3': 3, '4': 1, '5': 3, '10': 'size'},
    {'1': 'overwrite', '3': 4, '4': 1, '5': 8, '10': 'overwrite'},
    {
      '1': 'resume',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileUploadResumeHandle',
      '10': 'resume'
    },
    {
      '1': 'operation',
      '3': 6,
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

/// Descriptor for `FileUploadOpenCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileUploadOpenCommandDescriptor = $convert.base64Decode(
    'ChVGaWxlVXBsb2FkT3BlbkNvbW1hbmQSEgoEcGF0aBgCIAEoCVIEcGF0aBISCgRzaXplGAMgAS'
    'gDUgRzaXplEhwKCW92ZXJ3cml0ZRgEIAEoCFIJb3ZlcndyaXRlEj0KBnJlc3VtZRgFIAEoCzIl'
    'LmFueXR0eS5hcGkudjEuRmlsZVVwbG9hZFJlc3VtZUhhbmRsZVIGcmVzdW1lEjsKCW9wZXJhdG'
    'lvbhgGIAEoCzIdLmFueXR0eS5hcGkudjEuT3BlcmF0aW9uU3RhbXBSCW9wZXJhdGlvbkoECAEQ'
    'Ag==');

@$core.Deprecated('Use fileTransferCancelCommandDescriptor instead')
const FileTransferCancelCommand$json = {
  '1': 'FileTransferCancelCommand',
  '2': [
    {
      '1': 'transfer',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'transfer'
    },
    {
      '1': 'operation',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
    {
      '1': 'upload_resume',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileUploadResumeHandle',
      '10': 'uploadResume'
    },
  ],
  '9': [
    {'1': 1, '2': 2},
  ],
};

/// Descriptor for `FileTransferCancelCommand`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferCancelCommandDescriptor = $convert.base64Decode(
    'ChlGaWxlVHJhbnNmZXJDYW5jZWxDb21tYW5kEjkKCHRyYW5zZmVyGAIgASgLMh0uYW55dHR5Lm'
    'FwaS52MS5SZXNvdXJjZUhhbmRsZVIIdHJhbnNmZXISOwoJb3BlcmF0aW9uGAMgASgLMh0uYW55'
    'dHR5LmFwaS52MS5PcGVyYXRpb25TdGFtcFIJb3BlcmF0aW9uEkoKDXVwbG9hZF9yZXN1bWUYBC'
    'ABKAsyJS5hbnl0dHkuYXBpLnYxLkZpbGVVcGxvYWRSZXN1bWVIYW5kbGVSDHVwbG9hZFJlc3Vt'
    'ZUoECAEQAg==');

@$core.Deprecated('Use fileListResultDescriptor instead')
const FileListResult$json = {
  '1': 'FileListResult',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {
      '1': 'entries',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.FileEntry',
      '10': 'entries'
    },
    {'1': 'next_cursor', '3': 3, '4': 1, '5': 9, '10': 'nextCursor'},
  ],
};

/// Descriptor for `FileListResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileListResultDescriptor = $convert.base64Decode(
    'Cg5GaWxlTGlzdFJlc3VsdBISCgRwYXRoGAEgASgJUgRwYXRoEjIKB2VudHJpZXMYAiADKAsyGC'
    '5hbnl0dHkuYXBpLnYxLkZpbGVFbnRyeVIHZW50cmllcxIfCgtuZXh0X2N1cnNvchgDIAEoCVIK'
    'bmV4dEN1cnNvcg==');

@$core.Deprecated('Use fileStatResultDescriptor instead')
const FileStatResult$json = {
  '1': 'FileStatResult',
  '2': [
    {
      '1': 'entry',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileEntry',
      '10': 'entry'
    },
  ],
};

/// Descriptor for `FileStatResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileStatResultDescriptor = $convert.base64Decode(
    'Cg5GaWxlU3RhdFJlc3VsdBIuCgVlbnRyeRgBIAEoCzIYLmFueXR0eS5hcGkudjEuRmlsZUVudH'
    'J5UgVlbnRyeQ==');

@$core.Deprecated('Use filePreviewResultDescriptor instead')
const FilePreviewResult$json = {
  '1': 'FilePreviewResult',
  '2': [
    {
      '1': 'entry',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileEntry',
      '10': 'entry'
    },
    {'1': 'mime_type', '3': 2, '4': 1, '5': 9, '10': 'mimeType'},
    {'1': 'content', '3': 3, '4': 1, '5': 12, '10': 'content'},
    {'1': 'truncated', '3': 4, '4': 1, '5': 8, '10': 'truncated'},
    {'1': 'sha256', '3': 5, '4': 1, '5': 12, '10': 'sha256'},
  ],
};

/// Descriptor for `FilePreviewResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List filePreviewResultDescriptor = $convert.base64Decode(
    'ChFGaWxlUHJldmlld1Jlc3VsdBIuCgVlbnRyeRgBIAEoCzIYLmFueXR0eS5hcGkudjEuRmlsZU'
    'VudHJ5UgVlbnRyeRIbCgltaW1lX3R5cGUYAiABKAlSCG1pbWVUeXBlEhgKB2NvbnRlbnQYAyAB'
    'KAxSB2NvbnRlbnQSHAoJdHJ1bmNhdGVkGAQgASgIUgl0cnVuY2F0ZWQSFgoGc2hhMjU2GAUgAS'
    'gMUgZzaGEyNTY=');

@$core.Deprecated('Use fileOperationResultDescriptor instead')
const FileOperationResult$json = {
  '1': 'FileOperationResult',
  '2': [
    {'1': 'path', '3': 1, '4': 1, '5': 9, '10': 'path'},
    {'1': 'target_path', '3': 2, '4': 1, '5': 9, '10': 'targetPath'},
    {'1': 'success', '3': 3, '4': 1, '5': 8, '10': 'success'},
    {'1': 'error_code', '3': 4, '4': 1, '5': 9, '10': 'errorCode'},
    {'1': 'error_message', '3': 5, '4': 1, '5': 9, '10': 'errorMessage'},
  ],
};

/// Descriptor for `FileOperationResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileOperationResultDescriptor = $convert.base64Decode(
    'ChNGaWxlT3BlcmF0aW9uUmVzdWx0EhIKBHBhdGgYASABKAlSBHBhdGgSHwoLdGFyZ2V0X3BhdG'
    'gYAiABKAlSCnRhcmdldFBhdGgSGAoHc3VjY2VzcxgDIAEoCFIHc3VjY2VzcxIdCgplcnJvcl9j'
    'b2RlGAQgASgJUgllcnJvckNvZGUSIwoNZXJyb3JfbWVzc2FnZRgFIAEoCVIMZXJyb3JNZXNzYW'
    'dl');

@$core.Deprecated('Use fileBatchResultDescriptor instead')
const FileBatchResult$json = {
  '1': 'FileBatchResult',
  '2': [
    {
      '1': 'results',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.anytty.api.v1.FileOperationResult',
      '10': 'results'
    },
  ],
};

/// Descriptor for `FileBatchResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileBatchResultDescriptor = $convert.base64Decode(
    'Cg9GaWxlQmF0Y2hSZXN1bHQSPAoHcmVzdWx0cxgBIAMoCzIiLmFueXR0eS5hcGkudjEuRmlsZU'
    '9wZXJhdGlvblJlc3VsdFIHcmVzdWx0cw==');

@$core.Deprecated('Use fileTransferHandleDescriptor instead')
const FileTransferHandle$json = {
  '1': 'FileTransferHandle',
  '2': [
    {
      '1': 'resource',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.ResourceHandle',
      '10': 'resource'
    },
    {'1': 'path', '3': 2, '4': 1, '5': 9, '10': 'path'},
    {'1': 'offset', '3': 3, '4': 1, '5': 3, '10': 'offset'},
    {'1': 'size', '3': 4, '4': 1, '5': 3, '10': 'size'},
    {
      '1': 'modified_at_unix_nano',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'modifiedAtUnixNano'
    },
    {
      '1': 'operation',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.OperationStamp',
      '10': 'operation'
    },
    {
      '1': 'resume',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileUploadResumeHandle',
      '10': 'resume'
    },
    {'1': 'chunk_bytes', '3': 8, '4': 1, '5': 13, '10': 'chunkBytes'},
    {'1': 'window_bytes', '3': 9, '4': 1, '5': 3, '10': 'windowBytes'},
  ],
};

/// Descriptor for `FileTransferHandle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferHandleDescriptor = $convert.base64Decode(
    'ChJGaWxlVHJhbnNmZXJIYW5kbGUSOQoIcmVzb3VyY2UYASABKAsyHS5hbnl0dHkuYXBpLnYxLl'
    'Jlc291cmNlSGFuZGxlUghyZXNvdXJjZRISCgRwYXRoGAIgASgJUgRwYXRoEhYKBm9mZnNldBgD'
    'IAEoA1IGb2Zmc2V0EhIKBHNpemUYBCABKANSBHNpemUSMQoVbW9kaWZpZWRfYXRfdW5peF9uYW'
    '5vGAUgASgDUhJtb2RpZmllZEF0VW5peE5hbm8SOwoJb3BlcmF0aW9uGAYgASgLMh0uYW55dHR5'
    'LmFwaS52MS5PcGVyYXRpb25TdGFtcFIJb3BlcmF0aW9uEj0KBnJlc3VtZRgHIAEoCzIlLmFueX'
    'R0eS5hcGkudjEuRmlsZVVwbG9hZFJlc3VtZUhhbmRsZVIGcmVzdW1lEh8KC2NodW5rX2J5dGVz'
    'GAggASgNUgpjaHVua0J5dGVzEiEKDHdpbmRvd19ieXRlcxgJIAEoA1ILd2luZG93Qnl0ZXM=');

@$core.Deprecated('Use fileTransferOpenResultDescriptor instead')
const FileTransferOpenResult$json = {
  '1': 'FileTransferOpenResult',
  '2': [
    {
      '1': 'transfer',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileTransferHandle',
      '10': 'transfer'
    },
  ],
};

/// Descriptor for `FileTransferOpenResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferOpenResultDescriptor =
    $convert.base64Decode(
        'ChZGaWxlVHJhbnNmZXJPcGVuUmVzdWx0Ej0KCHRyYW5zZmVyGAEgASgLMiEuYW55dHR5LmFwaS'
        '52MS5GaWxlVHJhbnNmZXJIYW5kbGVSCHRyYW5zZmVy');

@$core.Deprecated('Use fileTransferCancelResultDescriptor instead')
const FileTransferCancelResult$json = {
  '1': 'FileTransferCancelResult',
  '2': [
    {'1': 'cancelled', '3': 1, '4': 1, '5': 8, '10': 'cancelled'},
  ],
};

/// Descriptor for `FileTransferCancelResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferCancelResultDescriptor =
    $convert.base64Decode(
        'ChhGaWxlVHJhbnNmZXJDYW5jZWxSZXN1bHQSHAoJY2FuY2VsbGVkGAEgASgIUgljYW5jZWxsZW'
        'Q=');

@$core.Deprecated('Use fileTransferCompletedEventDescriptor instead')
const FileTransferCompletedEvent$json = {
  '1': 'FileTransferCompletedEvent',
  '2': [
    {
      '1': 'transfer',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.anytty.api.v1.FileTransferHandle',
      '10': 'transfer'
    },
    {'1': 'size', '3': 2, '4': 1, '5': 3, '10': 'size'},
    {'1': 'sha256', '3': 3, '4': 1, '5': 12, '10': 'sha256'},
  ],
};

/// Descriptor for `FileTransferCompletedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileTransferCompletedEventDescriptor =
    $convert.base64Decode(
        'ChpGaWxlVHJhbnNmZXJDb21wbGV0ZWRFdmVudBI9Cgh0cmFuc2ZlchgBIAEoCzIhLmFueXR0eS'
        '5hcGkudjEuRmlsZVRyYW5zZmVySGFuZGxlUgh0cmFuc2ZlchISCgRzaXplGAIgASgDUgRzaXpl'
        'EhYKBnNoYTI1NhgDIAEoDFIGc2hhMjU2');
