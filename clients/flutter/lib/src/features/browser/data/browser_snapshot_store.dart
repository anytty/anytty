import 'dart:io';
import 'dart:typed_data';

import 'package:path_provider/path_provider.dart';

abstract interface class BrowserSnapshotStore {
  Future<String> save(String sessionId, Uint8List bytes);

  Future<Uint8List?> read(String path);

  Future<void> remove(String path);
}

final class ApplicationBrowserSnapshotStore implements BrowserSnapshotStore {
  const ApplicationBrowserSnapshotStore();

  @override
  Future<String> save(String sessionId, Uint8List bytes) async {
    final directory = await _directory();
    final destination = File('${directory.path}/${_fileName(sessionId)}');
    final temporary = File('${destination.path}.part');
    await temporary.writeAsBytes(bytes, flush: true);
    if (await destination.exists()) await destination.delete();
    await temporary.rename(destination.path);
    return destination.path;
  }

  @override
  Future<Uint8List?> read(String path) async {
    final file = File(path);
    if (!await file.exists()) return null;
    return file.readAsBytes();
  }

  @override
  Future<void> remove(String path) async {
    final file = File(path);
    if (await file.exists()) await file.delete();
  }

  Future<Directory> _directory() async {
    final support = await getApplicationSupportDirectory();
    final directory = Directory('${support.path}/browser-session-snapshots');
    await directory.create(recursive: true);
    return directory;
  }

  String _fileName(String sessionId) {
    final safe = sessionId.replaceAll(RegExp(r'[^a-zA-Z0-9._-]'), '_');
    final bounded = safe.isEmpty
        ? 'session'
        : safe.substring(0, safe.length.clamp(0, 80));
    return '$bounded.png';
  }
}
