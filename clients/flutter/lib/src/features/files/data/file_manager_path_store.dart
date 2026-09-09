import 'package:shared_preferences/shared_preferences.dart';

import '../domain/file_path.dart';

final class FileManagerPathStore {
  const FileManagerPathStore();

  static const storagePrefix = 'anytty.file-manager.last-path.v1';

  static Future<void> _writeQueue = Future<void>.value();

  Future<String?> load(String endpointId) async {
    if (endpointId.trim().isEmpty) return null;
    await _writeQueue;
    final preferences = await SharedPreferences.getInstance();
    final path = preferences.getString(_storageKey(endpointId))?.trim();
    if (path == null || path.isEmpty) return null;
    return normalizeFilePath(path);
  }

  Future<void> save(String endpointId, String path) {
    if (endpointId.trim().isEmpty) {
      throw StateError('Endpoint id is required');
    }
    final normalized = normalizeFilePath(path);
    return _enqueue(() async {
      final preferences = await SharedPreferences.getInstance();
      final saved = await preferences.setString(
        _storageKey(endpointId),
        normalized,
      );
      if (!saved) throw StateError('File manager path could not be saved');
    });
  }

  Future<void> remove(String endpointId) {
    if (endpointId.trim().isEmpty) return Future<void>.value();
    return _enqueue(() async {
      final preferences = await SharedPreferences.getInstance();
      await preferences.remove(_storageKey(endpointId));
    });
  }

  Future<void> _enqueue(Future<void> Function() operation) {
    final next = _writeQueue.then((_) => operation());
    _writeQueue = next.catchError((_) {});
    return next;
  }

  String _storageKey(String endpointId) =>
      '$storagePrefix:${Uri.encodeComponent(endpointId)}';
}
