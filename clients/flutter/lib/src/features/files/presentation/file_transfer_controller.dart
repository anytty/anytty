import 'dart:async';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';

import '../../../generated/proto/apipb/file.pb.dart' as api;
import '../../../native/native_file_export.dart';
import '../../terminal/data/endpoint_session_client.dart';
import '../data/endpoint_file_transfer.dart';
import '../domain/file_path.dart';

enum FileTransferDirection { download, upload }

enum FileTransferStatus {
  pending,
  transferring,
  paused,
  saving,
  completed,
  failed,
  cancelled,
}

typedef EndpointTransferSessionResolver =
    Future<EndpointSessionClient> Function(String endpointId);

final class FileTransferViewItem {
  FileTransferViewItem({
    required this.id,
    required this.endpointId,
    required this.endpointLabel,
    required this.name,
    required this.remotePath,
    required this.direction,
    required this.totalBytes,
  });

  final String id;
  final String endpointId;
  final String endpointLabel;
  final String name;
  final String remotePath;
  final FileTransferDirection direction;
  final int totalBytes;
  int transferredBytes = 0;
  FileTransferStatus status = FileTransferStatus.pending;
  String? error;
  Uri? savedUri;
  String? uploadSourcePath;
  bool ownsUploadSource = false;
  String? downloadCacheDirectoryPath;
  int remoteModifiedAtUnixNano = 0;
  api.FileUploadResumeHandle? uploadResume;
  bool downloadReady = false;
  bool preserveAfterInterruption = true;
  Completer<void> cancellation = Completer<void>();
  Future<void>? attempt;

  bool get active =>
      status == FileTransferStatus.pending ||
      status == FileTransferStatus.transferring ||
      status == FileTransferStatus.saving;

  bool get pausable =>
      status == FileTransferStatus.pending ||
      status == FileTransferStatus.transferring;

  bool get resumable =>
      status == FileTransferStatus.paused ||
      status == FileTransferStatus.failed;
}

final class FileTransferController extends ChangeNotifier {
  FileTransferController({
    required this.session,
    FileExporter? fileExporter,
    FileOpener? fileOpener,
    Iterable<FileTransferViewItem> initialItems = const [],
  }) : fileExporter = fileExporter ?? const NativeFileExporter(),
       fileOpener = fileOpener ?? const NativeFileOpener() {
    _items.addAll(initialItems);
  }

  final EndpointTransferSessionResolver session;
  final FileExporter fileExporter;
  final FileOpener fileOpener;
  final List<FileTransferViewItem> _items = [];
  final Map<String, int> _uploadCompletionRevisions = {};
  Timer? _progressNotifyTimer;
  var _sequence = 0;
  bool _disposed = false;

  List<FileTransferViewItem> get items => List.unmodifiable(_items.reversed);

  bool get hasActiveTransfers => _items.any((item) => item.active);

  int uploadCompletionRevision(String endpointId) =>
      _uploadCompletionRevisions[endpointId] ?? 0;

  Future<void> startDownload({
    required String endpointId,
    required String endpointLabel,
    required String remotePath,
    required String name,
    required int size,
  }) async {
    final item = _add(
      endpointId: endpointId,
      endpointLabel: endpointLabel,
      name: name,
      remotePath: remotePath,
      direction: FileTransferDirection.download,
      totalBytes: size,
    );
    await _launchDownload(item);
  }

  Future<void> _launchDownload(FileTransferViewItem item) {
    final cancellation = item.cancellation;
    late final Future<void> attempt;
    attempt = _runDownload(item, cancellation).whenComplete(() {
      if (identical(item.attempt, attempt)) item.attempt = null;
    });
    item.attempt = attempt;
    return attempt;
  }

  Future<void> _runDownload(
    FileTransferViewItem item,
    Completer<void> cancellation,
  ) async {
    File? temporary;
    Directory? transferDirectory;
    try {
      final cache = await getTemporaryDirectory();
      item.downloadCacheDirectoryPath ??=
          '${cache.path}/anytty-downloads/${item.id}';
      transferDirectory = Directory(item.downloadCacheDirectoryPath!);
      await transferDirectory.create(recursive: true);
      final localName = _safeLocalFileName(item.name);
      temporary = File('${transferDirectory.path}/$localName');
      if (!item.downloadReady || !await temporary.exists()) {
        item.downloadReady = false;
        _update(item, status: FileTransferStatus.transferring);
        final client = await session(item.endpointId);
        await EndpointFileTransfer(client).downloadToFile(
          remotePath: item.remotePath,
          destination: temporary,
          resume: true,
          expectedSize: item.totalBytes,
          expectedModifiedAtUnixNano: item.remoteModifiedAtUnixNano,
          cancelWhen: cancellation.future,
          onOpened: (transfer) {
            item.remoteModifiedAtUnixNano = transfer.modifiedAtUnixNano.toInt();
          },
          preserveOnInterruption: () => item.preserveAfterInterruption,
          onProgress: (transferred, total) {
            item.transferredBytes = transferred;
            _notifyProgress();
          },
        );
        item.downloadReady = true;
      }
      if (cancellation.isCompleted) {
        if (item.status != FileTransferStatus.paused) {
          _update(item, status: FileTransferStatus.cancelled);
        }
        return;
      }
      _update(item, status: FileTransferStatus.saving);
      final uri = await fileExporter.exportFile(
        source: temporary,
        fileName: localName,
        mimeType: fileMimeType(localName),
      );
      if (cancellation.isCompleted) {
        if (item.status != FileTransferStatus.paused) {
          _update(item, status: FileTransferStatus.cancelled);
        }
        return;
      }
      if (uri == null) {
        _update(
          item,
          status: FileTransferStatus.cancelled,
          error: 'Save cancelled',
        );
        return;
      }
      item
        ..transferredBytes = item.totalBytes
        ..savedUri = uri
        ..downloadReady = false;
      _update(item, status: FileTransferStatus.completed);
    } catch (error) {
      _updateFailure(item, error, cancellation);
    } finally {
      if (item.status == FileTransferStatus.completed ||
          item.status == FileTransferStatus.cancelled) {
        await _cleanupDownloadCache(item);
      }
    }
  }

  Future<bool> pickAndUpload({
    required String endpointId,
    required String endpointLabel,
    required String remoteDirectory,
  }) async {
    final picked = await FilePicker.pickFiles(type: FileType.any);
    if (picked.isEmpty || _disposed) return false;
    for (final file in picked) {
      unawaited(
        _startUpload(
          file,
          endpointId: endpointId,
          endpointLabel: endpointLabel,
          remoteDirectory: remoteDirectory,
        ),
      );
    }
    return true;
  }

  Future<void> _startUpload(
    PlatformFile picked, {
    required String endpointId,
    required String endpointLabel,
    required String remoteDirectory,
  }) async {
    final pickedSize = await picked.length();
    if (_disposed) return;
    final remotePath = joinFilePath(remoteDirectory, picked.name);
    final item = _add(
      endpointId: endpointId,
      endpointLabel: endpointLabel,
      name: picked.name,
      remotePath: remotePath,
      direction: FileTransferDirection.upload,
      totalBytes: pickedSize,
    );
    try {
      if (picked.path case final path? when path.isNotEmpty) {
        final cache = await getTemporaryDirectory();
        item
          ..uploadSourcePath = path
          ..ownsUploadSource = _isWithinDirectory(path, cache.path);
      } else {
        final cache = await getTemporaryDirectory();
        final transferDirectory = Directory(
          '${cache.path}/anytty-uploads/${item.id}',
        );
        await transferDirectory.create(recursive: true);
        final materialized = File(
          '${transferDirectory.path}/${_safeLocalFileName(picked.name)}',
        );
        final output = materialized.openWrite();
        try {
          await for (final chunk in picked.readAsByteStream()) {
            output.add(chunk);
          }
        } finally {
          await output.close();
        }
        item
          ..uploadSourcePath = materialized.path
          ..ownsUploadSource = true;
      }
      if (item.cancellation.isCompleted) {
        _update(item, status: FileTransferStatus.cancelled);
        await _cleanupUploadSource(item);
        return;
      }
      await _launchUpload(item);
    } catch (error) {
      _updateFailure(item, error, item.cancellation);
    }
  }

  Future<void> _launchUpload(FileTransferViewItem item) {
    final cancellation = item.cancellation;
    late final Future<void> attempt;
    attempt = _runUpload(item, cancellation).whenComplete(() {
      if (identical(item.attempt, attempt)) item.attempt = null;
    });
    item.attempt = attempt;
    return attempt;
  }

  Future<void> _runUpload(
    FileTransferViewItem item,
    Completer<void> cancellation,
  ) async {
    final sourcePath = item.uploadSourcePath;
    if (sourcePath == null || sourcePath.isEmpty) {
      _update(
        item,
        status: FileTransferStatus.failed,
        error: 'Upload source is no longer available',
      );
      return;
    }
    final source = File(sourcePath);
    try {
      if (!await source.exists()) {
        throw const FileSystemException('Upload source is no longer available');
      }
      _update(item, status: FileTransferStatus.transferring);
      final client = await session(item.endpointId);
      await EndpointFileTransfer(client).uploadFile(
        source: source,
        remotePath: item.remotePath,
        resume: item.uploadResume,
        cancelWhen: cancellation.future,
        onOpened: (transfer) {
          item.uploadResume = transfer.hasResume()
              ? transfer.resume.deepCopy()
              : null;
        },
        preserveOnInterruption: () => item.preserveAfterInterruption,
        onProgress: (transferred, total) {
          item.transferredBytes = transferred;
          _notifyProgress();
        },
      );
      item
        ..transferredBytes = item.totalBytes
        ..uploadResume = null;
      _update(item, status: FileTransferStatus.completed);
      _uploadCompletionRevisions.update(
        item.endpointId,
        (revision) => revision + 1,
        ifAbsent: () => 1,
      );
      _notify();
    } catch (error) {
      _updateFailure(item, error, cancellation);
    } finally {
      if (item.status == FileTransferStatus.completed ||
          item.status == FileTransferStatus.cancelled) {
        await _cleanupUploadSource(item);
      }
    }
  }

  void retry(String id) {
    final item = _items.where((candidate) => candidate.id == id).firstOrNull;
    if (item == null || item.status != FileTransferStatus.failed || _disposed) {
      return;
    }
    unawaited(_resumeItem(item));
  }

  Future<void> openDownloadedFile(String id) async {
    final item = _items.where((candidate) => candidate.id == id).firstOrNull;
    if (item == null ||
        item.direction != FileTransferDirection.download ||
        item.status != FileTransferStatus.completed ||
        item.savedUri == null) {
      throw StateError('Completed download is unavailable');
    }
    await fileOpener.openFile(
      uri: item.savedUri!,
      fileName: item.name,
      mimeType: fileMimeType(item.name),
    );
  }

  void pause(String id) {
    final item = _items.where((candidate) => candidate.id == id).firstOrNull;
    if (item == null || !item.pausable) return;
    item.preserveAfterInterruption = true;
    _update(item, status: FileTransferStatus.paused);
    if (!item.cancellation.isCompleted) item.cancellation.complete();
  }

  void resume(String id) {
    final item = _items.where((candidate) => candidate.id == id).firstOrNull;
    if (item == null || !item.resumable || _disposed) return;
    unawaited(_resumeItem(item));
  }

  Future<void> _resumeItem(FileTransferViewItem item) async {
    final requestedStatus = item.status;
    if (!item.resumable) return;
    await item.attempt;
    if (_disposed || item.status != requestedStatus || !_items.contains(item)) {
      return;
    }
    item
      ..preserveAfterInterruption = true
      ..cancellation = Completer<void>()
      ..savedUri = null;
    _update(item, status: FileTransferStatus.pending);
    if (item.direction == FileTransferDirection.download) {
      await _launchDownload(item);
    } else {
      await _launchUpload(item);
    }
  }

  void cancel(String id) {
    final item = _items.where((candidate) => candidate.id == id).firstOrNull;
    if (item == null ||
        (!item.active && item.status != FileTransferStatus.paused)) {
      return;
    }
    item.preserveAfterInterruption = false;
    if (!item.cancellation.isCompleted) item.cancellation.complete();
    _update(item, status: FileTransferStatus.cancelled);
    unawaited(_discardTransferArtifacts(item));
  }

  void dismiss(String id) {
    final removed = _items
        .where((item) => item.id == id && !item.active)
        .toList();
    _items.removeWhere((item) => removed.contains(item));
    for (final item in removed) {
      unawaited(_discardTransferArtifacts(item));
    }
    _notify();
  }

  void clearCompleted() {
    final removed = _items
        .where((item) => item.status == FileTransferStatus.completed)
        .toList();
    _items.removeWhere(removed.contains);
    for (final item in removed) {
      unawaited(_discardTransferArtifacts(item));
    }
    _notify();
  }

  void clearFailed() {
    final removed = _items
        .where(
          (item) =>
              item.status == FileTransferStatus.failed ||
              item.status == FileTransferStatus.cancelled,
        )
        .toList();
    _items.removeWhere(removed.contains);
    for (final item in removed) {
      unawaited(_discardTransferArtifacts(item));
    }
    _notify();
  }

  Future<void> _cleanupUploadSource(FileTransferViewItem item) async {
    if (!item.ownsUploadSource || item.uploadSourcePath == null) return;
    final source = File(item.uploadSourcePath!);
    item
      ..uploadSourcePath = null
      ..ownsUploadSource = false;
    if (await source.exists()) await source.delete();
    final parent = source.parent;
    if (await parent.exists()) {
      try {
        await parent.delete();
      } on FileSystemException {
        // A picker may group multiple selected files in the same cache folder.
      }
    }
  }

  Future<void> _cleanupDownloadCache(FileTransferViewItem item) async {
    final path = item.downloadCacheDirectoryPath;
    item
      ..downloadCacheDirectoryPath = null
      ..downloadReady = false;
    if (path == null) return;
    final directory = Directory(path);
    if (await directory.exists()) await directory.delete(recursive: true);
  }

  Future<void> _discardTransferArtifacts(FileTransferViewItem item) async {
    await item.attempt;
    if (item.uploadResume case final resume?) {
      try {
        final client = await session(item.endpointId);
        await client.cancelFileTransfer(
          api.FileTransferHandle(resume: resume.deepCopy()),
        );
      } catch (_) {
        // Local removal remains available if the detached remote expired.
      }
      item.uploadResume = null;
    }
    await _cleanupUploadSource(item);
    await _cleanupDownloadCache(item);
  }

  FileTransferViewItem _add({
    required String endpointId,
    required String endpointLabel,
    required String name,
    required String remotePath,
    required FileTransferDirection direction,
    required int totalBytes,
  }) {
    _sequence += 1;
    final item = FileTransferViewItem(
      id: '${DateTime.now().microsecondsSinceEpoch}-$_sequence',
      endpointId: endpointId,
      endpointLabel: endpointLabel,
      name: name,
      remotePath: remotePath,
      direction: direction,
      totalBytes: totalBytes,
    );
    _items.add(item);
    _notify();
    return item;
  }

  void _update(
    FileTransferViewItem item, {
    required FileTransferStatus status,
    String? error,
  }) {
    item
      ..status = status
      ..error = error;
    _notify();
  }

  void _updateFailure(
    FileTransferViewItem item,
    Object error,
    Completer<void> cancellation,
  ) {
    if (cancellation.isCompleted) {
      if (item.status != FileTransferStatus.paused &&
          item.status != FileTransferStatus.cancelled) {
        _update(item, status: FileTransferStatus.cancelled);
      }
    } else {
      _update(item, status: FileTransferStatus.failed, error: error.toString());
    }
  }

  void _notify() {
    _progressNotifyTimer?.cancel();
    _progressNotifyTimer = null;
    if (!_disposed) notifyListeners();
  }

  void _notifyProgress() {
    if (_disposed || _progressNotifyTimer != null) return;
    _progressNotifyTimer = Timer(const Duration(milliseconds: 33), () {
      _progressNotifyTimer = null;
      if (!_disposed) notifyListeners();
    });
  }

  @override
  void dispose() {
    _disposed = true;
    _progressNotifyTimer?.cancel();
    for (final item in _items.where(
      (candidate) =>
          candidate.active || candidate.status == FileTransferStatus.paused,
    )) {
      item
        ..preserveAfterInterruption = false
        ..status = FileTransferStatus.cancelled;
      if (!item.cancellation.isCompleted) item.cancellation.complete();
      unawaited(_discardTransferArtifacts(item));
    }
    super.dispose();
  }
}

bool _isWithinDirectory(String filePath, String directoryPath) {
  final file = File(filePath).absolute.path;
  final directory = Directory(directoryPath).absolute.path;
  return file.startsWith('$directory${Platform.pathSeparator}');
}

String _safeLocalFileName(String name) {
  final leaf = fileBasename(name).trim();
  final sanitized = leaf.replaceAll(RegExp(r'[\x00-\x1f/:*?"<>|]'), '_');
  if (sanitized.isEmpty || sanitized == '.' || sanitized == '..') {
    return 'download';
  }
  return sanitized;
}
