import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../domain/browser_session.dart';

abstract interface class BrowserSessionStore {
  Future<BrowserSessionSnapshot?> load(String sessionId);

  Future<void> save(BrowserSessionSnapshot snapshot);

  Future<void> remove(String sessionId);
}

final class SharedPreferencesBrowserSessionStore
    implements BrowserSessionStore {
  const SharedPreferencesBrowserSessionStore();

  static const _key = 'browser.session.snapshots.v1';

  static Future<void> _writeQueue = Future<void>.value();

  @override
  Future<BrowserSessionSnapshot?> load(String sessionId) async {
    await _writeQueue;
    final preferences = await SharedPreferences.getInstance();
    final all = _decode(preferences.getString(_key));
    final value = all[sessionId];
    if (value == null) return null;
    try {
      return BrowserSessionSnapshot.fromJson(value);
    } on FormatException {
      return null;
    }
  }

  @override
  Future<void> save(BrowserSessionSnapshot snapshot) async {
    return _enqueue(() async {
      final preferences = await SharedPreferences.getInstance();
      final all = _decode(preferences.getString(_key));
      all[snapshot.sessionId] = snapshot.toJson();
      await preferences.setString(_key, jsonEncode(all));
    });
  }

  @override
  Future<void> remove(String sessionId) async {
    return _enqueue(() async {
      final preferences = await SharedPreferences.getInstance();
      final all = _decode(preferences.getString(_key));
      if (all.remove(sessionId) == null) return;
      await preferences.setString(_key, jsonEncode(all));
    });
  }

  Future<void> _enqueue(Future<void> Function() operation) {
    final next = _writeQueue.then((_) => operation());
    _writeQueue = next.catchError((_) {});
    return next;
  }

  Map<String, Map<String, Object?>> _decode(String? value) {
    if (value == null || value.isEmpty) return {};
    try {
      final decoded = jsonDecode(value);
      if (decoded is! Map) return {};
      return decoded.map<String, Map<String, Object?>>((key, value) {
        if (value is! Map) {
          throw const FormatException('Invalid browser session map');
        }
        return MapEntry(key.toString(), value.cast<String, Object?>());
      });
    } on FormatException {
      return {};
    } on TypeError {
      return {};
    }
  }
}
