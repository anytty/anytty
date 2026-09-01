import 'dart:async';
import 'dart:io';

import 'package:cryptography/dart.dart';
import 'package:fixnum/fixnum.dart';

import '../../../generated/proto/apipb/file.pb.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/wirepb/terminal.pb.dart' as wire;
import '../../../native/anytty_resource_stream.dart';
import '../../../native/binding_operation.dart';
import '../../terminal/data/endpoint_session_client.dart';

typedef FileTransferProgressCallback = void Function(
  int transferredBytes,
  int totalBytes,
);

final class FileTransferException implements Exception {
  const FileTransferException(this.message, {this.code});

  final String message;
  final int? code;

  @override
  String toString() => message;
}

final class EndpointFileTransfer {
  const EndpointFileTransfer(this._session);

  static const _maximumChunkBytes = 1024 * 1024;
  static const _frameTimeout = Duration(seconds: 45);

  final EndpointSessionClient _session;

  Future<wire.FileTransferResult> downloadToFile({
    required String remotePath,
    required File destination,
    bool overwrite = false,
    bool resume = false,
    int expectedSize = 0,
    int expectedModifiedAtUnixNano = 0,
    Future<void>? cancelWhen,
    FileTransferProgressCallback? onProgress,
    void Function(FileTransferHandle transfer)? onOpened,
    bool Function()? preserveOnInterruption,
  }) async {
    if (!overwrite && await destination.exists()) {
      throw FileSystemException('Destination already exists', destination.path);
    }
    await destination.parent.create(recursive: true);
    final temporary = File('${destination.path}.part');
    if (!resume && await temporary.exists()) await temporary.delete();
    RandomAccessFile? output;
    AnyttyResourceStream? stream;
    FileTransferHandle? transfer;
    var completed = false;
    var cancelled = false;
    try {
      var resumeOffset = resume && await temporary.exists()
          ? await temporary.length()
          : 0;
      try {
        transfer = await _session.openFileDownload(
          remotePath,
          offset: resumeOffset,
          expectedSize: resumeOffset > 0 ? expectedSize : 0,
          expectedModifiedAtUnixNano: resumeOffset > 0
              ? expectedModifiedAtUnixNano
              : 0,
        );
      } catch (error) {
        if (resumeOffset <= 0 || !_isStaleDownloadError(error)) rethrow;
        await temporary.delete();
        resumeOffset = 0;
        transfer = await _session.openFileDownload(remotePath);
      }
      _validateTransfer(transfer, expectedSize: null, allowResume: true);
      if (transfer.offset.toInt() != resumeOffset) {
        throw const FileTransferException(
          'Daemon did not accept the download resume offset',
        );
      }
      onOpened?.call(transfer.deepCopy());
      final total = transfer.size.toInt();
      final digest = const DartSha256().newHashSink();
      if (resumeOffset > 0) {
        await for (final chunk in temporary.openRead()) {
          digest.add(chunk);
        }
      }
      output = await temporary.open(
        mode: resumeOffset > 0 ? FileMode.append : FileMode.write,
      );
      stream = await _session.openFileResourceStream(transfer);
      if (cancelWhen != null) {
        unawaited(
          cancelWhen.then((_) {
            if (completed) return;
            cancelled = true;
            try {
              stream?.close();
            } catch (_) {
              // The transfer cleanup below owns the final error and release.
            }
          }),
        );
      }
      var offset = transfer.offset.toInt();
      var bytesSinceAcknowledgement = 0;
      onProgress?.call(offset, total);
      await for (final frame in stream.frames.timeout(_frameTimeout)) {
        if (cancelled) throw const BindingOperationCancelledException();
        switch (frame.type) {
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA:
            final data = wire.FileTransferData.fromBuffer(frame.payload);
            if (data.offset.toInt() != offset || data.data.isEmpty) {
              throw FileTransferException(
                'Invalid download data at offset $offset',
              );
            }
            await output!.writeFrom(data.data);
            digest.add(data.data);
            offset += data.data.length;
            bytesSinceAcknowledgement += data.data.length;
            if (bytesSinceAcknowledgement >= transfer.windowBytes.toInt()) {
              stream.send(
                ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_ACK,
                wire.FileTransferAck(
                  offset: Int64(offset),
                  windowBytes: Int64(bytesSinceAcknowledgement),
                ).writeToBuffer(),
              );
              bytesSinceAcknowledgement = 0;
            }
            onProgress?.call(offset, total);
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH:
            final finish = wire.FileTransferFinish.fromBuffer(frame.payload);
            digest.close();
            final hash = await digest.hash();
            if (finish.size.toInt() != offset ||
                finish.size != transfer.size ||
                !_bytesEqual(finish.sha256, hash.bytes)) {
              throw const FileTransferException(
                'Download size or SHA-256 mismatch',
              );
            }
            await output!.flush();
            await output.close();
            output = null;
            if (overwrite && await destination.exists()) {
              await destination.delete();
            }
            await temporary.rename(destination.path);
            completed = true;
            onProgress?.call(offset, total);
            return wire.FileTransferResult(
              path: transfer.path,
              size: finish.size,
              sha256: finish.sha256,
            );
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_ERROR:
            throw _decodeStreamError(frame.payload);
          default:
            throw FileTransferException(
              'Unexpected download frame ${frame.type.name}',
            );
        }
      }
      if (cancelled) throw const BindingOperationCancelledException();
      await _throwIfStreamFailed(stream);
      throw const FileTransferException('Download stream closed before finish');
    } finally {
      await output?.close();
      if (stream != null) await _closeStream(stream);
      if (!completed && transfer != null) {
        if (preserveOnInterruption?.call() == true) {
          await _releaseTransfer(transfer);
        } else {
          await _cancelTransfer(transfer);
        }
      }
      if (!completed &&
          preserveOnInterruption?.call() != true &&
          await temporary.exists()) {
        await temporary.delete();
      }
    }
  }

  Future<wire.FileTransferResult> uploadFile({
    required File source,
    required String remotePath,
    bool overwrite = false,
    FileUploadResumeHandle? resume,
    Future<void>? cancelWhen,
    FileTransferProgressCallback? onProgress,
    void Function(FileTransferHandle transfer)? onOpened,
    bool Function()? preserveOnInterruption,
  }) async {
    final stat = await source.stat();
    if (stat.type != FileSystemEntityType.file) {
      throw FileSystemException('Upload source must be a file', source.path);
    }
    final total = stat.size;
    AnyttyResourceStream? stream;
    FileTransferHandle? transfer;
    RandomAccessFile? input;
    var completed = false;
    var cancelled = false;
    try {
      transfer = await _session.openFileUpload(
        path: remotePath,
        size: total,
        overwrite: overwrite,
        resume: resume,
      );
      _validateTransfer(transfer, expectedSize: total, allowResume: true);
      onOpened?.call(transfer.deepCopy());
      final chunkBytes = transfer.chunkBytes;
      if (chunkBytes <= 0 || chunkBytes > _maximumChunkBytes) {
        throw FileTransferException('Invalid upload chunk size $chunkBytes');
      }
      input = await source.open();
      final digest = const DartSha256().newHashSink();
      var offset = 0;
      final resumeOffset = transfer.offset.toInt();
      while (offset < resumeOffset) {
        final prefix = await input.read(
          (resumeOffset - offset).clamp(1, chunkBytes),
        );
        if (prefix.isEmpty) {
          throw const FileTransferException('Upload source ended early');
        }
        digest.add(prefix);
        offset += prefix.length;
      }
      stream = await _session.openFileResourceStream(transfer);
      if (cancelWhen != null) {
        unawaited(
          cancelWhen.then((_) {
            if (completed) return;
            cancelled = true;
            try {
              stream?.close();
            } catch (_) {
              // The transfer cleanup below owns the final error and release.
            }
          }),
        );
      }
      final frames = StreamIterator(stream.frames);
      onProgress?.call(offset, total);
      while (offset < total) {
        if (cancelled) throw const BindingOperationCancelledException();
        final data = await input.read((total - offset).clamp(1, chunkBytes));
        if (data.isEmpty) {
          throw const FileTransferException('Upload source ended early');
        }
        digest.add(data);
        stream.send(
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
          wire.FileTransferData(
            offset: Int64(offset),
            data: data,
          ).writeToBuffer(),
        );
        final frame = await _nextFrame(frames, stream);
        if (frame.type ==
            ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_ERROR) {
          throw _decodeStreamError(frame.payload);
        }
        if (frame.type !=
            ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_ACK) {
          throw FileTransferException(
            'Unexpected upload frame ${frame.type.name}',
          );
        }
        final ack = wire.FileTransferAck.fromBuffer(frame.payload);
        if (ack.offset.toInt() != offset + data.length) {
          throw const FileTransferException('Invalid upload acknowledgement');
        }
        offset = ack.offset.toInt();
        onProgress?.call(offset, total);
      }
      digest.close();
      final hash = await digest.hash();
      stream.send(
        ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH,
        wire.FileTransferFinish(
          size: Int64(total),
          sha256: hash.bytes,
        ).writeToBuffer(),
      );
      final resultFrame = await _nextFrame(frames, stream);
      if (resultFrame.type ==
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_ERROR) {
        throw _decodeStreamError(resultFrame.payload);
      }
      if (resultFrame.type !=
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT) {
        throw FileTransferException(
          'Unexpected upload completion ${resultFrame.type.name}',
        );
      }
      final result = wire.FileTransferResult.fromBuffer(resultFrame.payload);
      if (result.size.toInt() != total ||
          !_bytesEqual(result.sha256, hash.bytes)) {
        throw const FileTransferException(
          'Upload completion size or SHA-256 mismatch',
        );
      }
      completed = true;
      onProgress?.call(total, total);
      return result.deepCopy();
    } finally {
      if (stream != null) await _closeStream(stream);
      await input?.close();
      if (!completed && transfer != null) {
        if (preserveOnInterruption?.call() == true) {
          await _releaseTransfer(transfer);
        } else {
          await _cancelTransfer(transfer);
        }
      }
    }
  }

  Future<ResourceStreamFrame> _nextFrame(
    StreamIterator<ResourceStreamFrame> frames,
    AnyttyResourceStream stream,
  ) async {
    final available = await frames.moveNext().timeout(_frameTimeout);
    if (available) return frames.current.deepCopy();
    await _throwIfStreamFailed(stream);
    throw const FileTransferException('File transfer stream closed');
  }

  Future<void> _throwIfStreamFailed(AnyttyResourceStream stream) async {
    final closed = await stream.closed;
    if (closed.hasError()) {
      throw FileTransferException(
        closed.error.message.isEmpty
            ? 'File transfer stream failed'
            : closed.error.message,
        code: closed.error.code.value,
      );
    }
  }

  Future<void> _closeStream(AnyttyResourceStream stream) async {
    try {
      stream.close();
    } catch (_) {
      return;
    }
    try {
      await stream.closed.timeout(const Duration(seconds: 3));
    } catch (_) {
      // The binding still owns idempotent handle cleanup during shutdown.
    }
  }

  Future<void> _cancelTransfer(FileTransferHandle transfer) async {
    try {
      await _session.cancelFileTransfer(transfer);
    } catch (_) {
      // Preserve the transfer failure that caused cancellation.
    }
  }

  Future<void> _releaseTransfer(FileTransferHandle transfer) async {
    try {
      await _session.releaseFileTransfer(transfer);
    } catch (_) {
      // Preserve the transfer failure that caused resource release.
    }
  }

  static void _validateTransfer(
    FileTransferHandle transfer, {
    required int? expectedSize,
    required bool allowResume,
  }) {
    final size = transfer.size.toInt();
    final offset = transfer.offset.toInt();
    if (!transfer.hasResource() ||
        transfer.path.trim().isEmpty ||
        size < 0 ||
        (expectedSize != null && size != expectedSize) ||
        offset < 0 ||
        offset > size ||
        (!allowResume && offset != 0) ||
        transfer.chunkBytes <= 0 ||
        transfer.chunkBytes > _maximumChunkBytes ||
        transfer.windowBytes <= Int64.ZERO) {
      throw const FileTransferException(
        'Daemon returned invalid file transfer metadata',
      );
    }
  }
}

FileTransferException _decodeStreamError(List<int> payload) {
  try {
    final envelope = wire.ErrorEnvelope.fromBuffer(payload);
    if (!envelope.hasError() || envelope.error.message.isEmpty) {
      throw const FormatException('incomplete error payload');
    }
    return FileTransferException(
      envelope.error.message,
      code: envelope.error.code,
    );
  } catch (error) {
    if (error is FileTransferException) return error;
    return const FileTransferException('Invalid file transfer error response');
  }
}

bool _bytesEqual(List<int> left, List<int> right) {
  if (left.length != right.length) return false;
  for (var index = 0; index < left.length; index += 1) {
    if (left[index] != right[index]) return false;
  }
  return true;
}

bool _isStaleDownloadError(Object error) {
  final message = error.toString().toLowerCase();
  return message.contains('stale download source') ||
      message.contains('invalid download offset');
}
