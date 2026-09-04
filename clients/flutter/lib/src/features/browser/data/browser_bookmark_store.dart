import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'browser_storage_keys.dart';

final class BrowserBookmark {
  const BrowserBookmark({required this.url, required this.title});

  final String url;
  final String title;

  Map<String, Object?> toJson() => {'url': url, 'title': title};

  factory BrowserBookmark.fromJson(Map<String, Object?> json) {
    final url = json['url'];
    final title = json['title'];
    if (url is! String || url.trim().isEmpty) {
      throw const FormatException('Browser bookmark URL is required');
    }
    return BrowserBookmark(
      url: url.trim(),
      title: title is String ? title.trim() : '',
    );
  }

  @override
  bool operator ==(Object other) =>
      other is BrowserBookmark && other.url == url && other.title == title;

  @override
  int get hashCode => Object.hash(url, title);
}

abstract interface class BrowserBookmarkStore {
  Future<List<BrowserBookmark>> load();

  Future<void> add(BrowserBookmark bookmark);

  Future<void> remove(String url);

  Future<void> clear();
}

final class SharedPreferencesBrowserBookmarkStore
    implements BrowserBookmarkStore {
  const SharedPreferencesBrowserBookmarkStore({this.limit = 24, this.scope});

  static const _legacyKey = 'browser.bookmarks.v1';
  static Future<void> _writeQueue = Future<void>.value();

  final int limit;
  final String? scope;

  String get _key =>
      scope == null ? _legacyKey : browserDeviceStorageKey(_legacyKey, scope!);

  @override
  Future<List<BrowserBookmark>> load() async {
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
  Future<void> add(BrowserBookmark bookmark) => _enqueue(() async {
    final preferences = await SharedPreferences.getInstance();
    final entries = _decode(preferences.getString(_key));
    final next = [
      bookmark,
      ...entries.where((item) => item.url != bookmark.url),
    ].take(limit).toList(growable: false);
    await preferences.setString(
      _key,
      jsonEncode(next.map((item) => item.toJson()).toList(growable: false)),
    );
  });

  @override
  Future<void> remove(String url) => _enqueue(() async {
    final preferences = await SharedPreferences.getInstance();
    final entries = _decode(preferences.getString(_key));
    final next = entries.where((item) => item.url != url).toList();
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

  List<BrowserBookmark> _decode(String? value) {
    if (value == null || value.isEmpty) return const [];
    try {
      final decoded = jsonDecode(value);
      if (decoded is! List) return const [];
      final entries = <BrowserBookmark>[];
      for (final item in decoded) {
        if (item is! Map) continue;
        try {
          entries.add(BrowserBookmark.fromJson(item.cast<String, Object?>()));
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
