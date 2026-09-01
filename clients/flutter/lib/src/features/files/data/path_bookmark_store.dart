import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../domain/file_path.dart';

final class PathBookmark {
  const PathBookmark({
    required this.id,
    required this.path,
    required this.label,
    required this.createdAt,
    required this.updatedAt,
    required this.version,
  });

  final String id;
  final String path;
  final String label;
  final DateTime createdAt;
  final DateTime updatedAt;
  final int version;

  Map<String, Object> toJson() => {
    'id': id,
    'path': path,
    'label': label,
    'createdAt': createdAt.toUtc().toIso8601String(),
    'updatedAt': updatedAt.toUtc().toIso8601String(),
    'version': version,
  };
}

final class PathBookmarkStore {
  const PathBookmarkStore();

  static const _prefix = 'anytty.path-bookmarks.v1:';

  Future<List<PathBookmark>> load(String endpointId) async {
    if (endpointId.isEmpty) return const [];
    final preferences = await SharedPreferences.getInstance();
    return _decode(preferences.getString('$_prefix$endpointId'));
  }

  Future<PathBookmark> add(
    String endpointId,
    String path, {
    String? label,
  }) async {
    final bookmarks = await load(endpointId);
    final normalizedPath = normalizeFilePath(path);
    final existing = bookmarks
        .where((bookmark) => bookmark.path == normalizedPath)
        .firstOrNull;
    final now = DateTime.now().toUtc();
    if (existing != null) {
      final updated = PathBookmark(
        id: existing.id,
        path: existing.path,
        label: label?.trim().isNotEmpty == true
            ? label!.trim()
            : existing.label,
        createdAt: existing.createdAt,
        updatedAt: now,
        version: existing.version + 1,
      );
      await _replace(endpointId, bookmarks, updated);
      return updated;
    }
    final bookmark = PathBookmark(
      id: _createId(normalizedPath, now),
      path: normalizedPath,
      label: label?.trim().isNotEmpty == true
          ? label!.trim()
          : pathBookmarkLabel(normalizedPath),
      createdAt: now,
      updatedAt: now,
      version: 1,
    );
    await _write(endpointId, [...bookmarks, bookmark]);
    return bookmark;
  }

  Future<PathBookmark> rename(
    String endpointId,
    String id,
    String label,
  ) async {
    final bookmarks = await load(endpointId);
    final current = bookmarks
        .where((bookmark) => bookmark.id == id)
        .firstOrNull;
    if (current == null) throw StateError('Bookmark not found');
    final normalizedLabel = label.trim();
    final updated = PathBookmark(
      id: current.id,
      path: current.path,
      label: normalizedLabel.isEmpty
          ? pathBookmarkLabel(current.path)
          : normalizedLabel,
      createdAt: current.createdAt,
      updatedAt: DateTime.now().toUtc(),
      version: current.version + 1,
    );
    await _replace(endpointId, bookmarks, updated);
    return updated;
  }

  Future<void> remove(String endpointId, String id) async {
    final bookmarks = await load(endpointId);
    await _write(
      endpointId,
      bookmarks.where((bookmark) => bookmark.id != id).toList(),
    );
  }

  Future<void> _replace(
    String endpointId,
    List<PathBookmark> bookmarks,
    PathBookmark updated,
  ) => _write(
    endpointId,
    bookmarks
        .map((bookmark) => bookmark.id == updated.id ? updated : bookmark)
        .toList(),
  );

  Future<void> _write(String endpointId, List<PathBookmark> bookmarks) async {
    if (endpointId.isEmpty) throw StateError('Endpoint id is required');
    final preferences = await SharedPreferences.getInstance();
    final sorted = _sort(bookmarks);
    final saved = await preferences.setString(
      '$_prefix$endpointId',
      jsonEncode(sorted.map((bookmark) => bookmark.toJson()).toList()),
    );
    if (!saved) throw StateError('Path bookmarks could not be saved');
  }
}

String pathBookmarkLabel(String path) {
  final normalized = normalizeFilePath(path);
  if (normalized == '/') return '/';
  final label = fileBasename(normalized);
  return label.isEmpty ? normalized : label;
}

List<PathBookmark> _decode(String? raw) {
  if (raw == null) return const [];
  try {
    final value = jsonDecode(raw);
    if (value is! List) return const [];
    return _sort(
      value.whereType<Map>().map(_decodeBookmark).whereType<PathBookmark>(),
    );
  } catch (_) {
    return const [];
  }
}

PathBookmark? _decodeBookmark(Map<Object?, Object?> value) {
  final id = value['id']?.toString().trim() ?? '';
  final rawPath = value['path']?.toString().trim() ?? '';
  if (id.isEmpty || rawPath.isEmpty) return null;
  final path = normalizeFilePath(rawPath);
  final createdAt = DateTime.tryParse(value['createdAt']?.toString() ?? '');
  final updatedAt = DateTime.tryParse(value['updatedAt']?.toString() ?? '');
  final version = value['version'];
  return PathBookmark(
    id: id,
    path: path,
    label: value['label']?.toString().trim().isNotEmpty == true
        ? value['label'].toString().trim()
        : pathBookmarkLabel(path),
    createdAt: createdAt?.toUtc() ?? DateTime.fromMillisecondsSinceEpoch(0),
    updatedAt: updatedAt?.toUtc() ?? DateTime.fromMillisecondsSinceEpoch(0),
    version: version is num && version.isFinite ? version.toInt() : 1,
  );
}

List<PathBookmark> _sort(Iterable<PathBookmark> bookmarks) {
  final result = bookmarks.toList(growable: false);
  result.sort((left, right) {
    final label = left.label.toLowerCase().compareTo(right.label.toLowerCase());
    return label == 0 ? left.path.compareTo(right.path) : label;
  });
  return result;
}

String _createId(String path, DateTime now) {
  final prefix = Uri.encodeComponent(path).replaceAll('%', '~');
  return '$prefix~${now.microsecondsSinceEpoch.toRadixString(36)}';
}
