import 'dart:io';

import 'package:flutter/services.dart';
import 'package:mime/mime.dart';

abstract interface class FileExporter {
  Future<Uri?> exportFile({
    required File source,
    required String fileName,
    required String mimeType,
  });
}

abstract interface class FileOpener {
  Future<void> openFile({
    required Uri uri,
    required String fileName,
    required String mimeType,
  });
}

final class NativeFileExporter implements FileExporter {
  const NativeFileExporter({
    this._channel = const MethodChannel('com.anytty.app/files'),
  });

  final MethodChannel _channel;

  @override
  Future<Uri?> exportFile({
    required File source,
    required String fileName,
    required String mimeType,
  }) async {
    final value = await _channel.invokeMethod<String>('exportFile', {
      'sourcePath': source.path,
      'fileName': fileName,
      'mimeType': mimeType,
    });
    if (value == null || value.isEmpty) return null;
    return Uri.parse(value);
  }
}

final class NativeFileOpener implements FileOpener {
  const NativeFileOpener({
    this._channel = const MethodChannel('com.anytty.app/files'),
  });

  final MethodChannel _channel;

  @override
  Future<void> openFile({
    required Uri uri,
    required String fileName,
    required String mimeType,
  }) async {
    await _channel.invokeMethod<void>('openFile', {
      'uri': uri.toString(),
      'fileName': fileName,
      'mimeType': mimeType,
    });
  }
}

String fileMimeType(String fileName) =>
    lookupMimeType(fileName) ?? 'application/octet-stream';
