import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'browser_storage_keys.dart';

final class BrowserHistoryEntry {
  const BrowserHistoryEntry({required this.url, required this.title});

  final String url;
  final String title;

  Map<String, Object?> toJson() => {'url': url, 'title': title};

  factory BrowserHistoryEntry.fromJson(Map<String, Object?> json) {
    final url = json['url'];
    final title = json['title'];
    if (url is! String || url.trim().isEmpty) {
      throw const FormatException('Browser history URL is required');
    }
    return BrowserHistoryEntry(
      url: url.trim(),
      title: title is String ? title.trim() : '',
    );
  }

  @override
  bool operator ==(Object other) =>
      other is BrowserHistoryEntry && other.url == url && other.title == title;

  @override
  int get hashCode => Object.hash(url, title);
}

abstract interface class BrowserHistoryStore {
  Future<List<BrowserHistoryEntry>> load();

  Future<void> add(BrowserHistoryEntry entry);

  Future<void> clear();
}

final class SharedPreferencesBrowserHistoryStore
    implements BrowserHistoryStore {
  const SharedPreferencesBrowserHistoryStore({this.limit = 20, this.scope});

  static const _legacyKey = 'browser.address.history.v1';
  static Future<void> _writeQueue = Future<void>.value();

  final int limit;
  final String? scope;

  String get _key =>
      scope == null ? _legacyKey : browserDeviceStorageKey(_legacyKey, scope!);

  @override
  Future<List<BrowserHistoryEntry>> load() async {
    await _writeQueue;
    final preferences = await SharedPreferences.getInstance();
    final scopedValue = preferences.getString(_key);
    if (scope != null && scopedValue == null) {
      final legacyValue = preferences.getString(_legacyKey);
      if (legacyValue != null) {
        final entries = _decode(legacyValue);
        await preferences.setString(
          _key,
          jsonEncode(
            entries.map((item) => item.toJson()).toList(growable: false),
          ),
        );
        await preferences.remove(_legacyKey);
        return entries;
      }
    }
    return _decode(scopedValue);
  }

  @override
  Future<void> add(BrowserHistoryEntry entry) => _enqueue(() async {
    final preferences = await SharedPreferences.getInstance();
    final entries = _decode(preferences.getString(_key));
    final next = [
      entry,
      ...entries.where((item) => item.url != entry.url),
    ].take(limit).toList(growable: false);
    await preferences.setString(
      _key,
      jsonEncode(next.map((item) => item.toJson()).toList(growable: false)),
    );
  });

  @override
  Future<void> clear() => _enqueue(() async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.remove(_key);
  });

  Future<void> _enqueue(Future<void> Function() operation) {
    final next = _writeQueue.then((_) => operation());
    _writeQueue = next.catchError((_) {});
    return next;
  }

  List<BrowserHistoryEntry> _decode(String? value) {
    if (value == null || value.isEmpty) return const [];
    try {
      final decoded = jsonDecode(value);
      if (decoded is! List) return const [];
      final entries = <BrowserHistoryEntry>[];
      for (final item in decoded) {
        if (item is! Map) continue;
        try {
          entries.add(
            BrowserHistoryEntry.fromJson(item.cast<String, Object?>()),
          );
        } on FormatException {
          continue;
        }
      }
      return entries.take(limit).toList(growable: false);
    } on FormatException {
      return const [];
    } on TypeError {
      return const [];
    }
  }
}
