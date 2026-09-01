import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'background_preferences.dart';

abstract interface class BackgroundPreferencesStorage {
  Future<String?> read(String key);

  Future<bool> write(String key, String value);
}

final class SharedBackgroundPreferencesStorage
    implements BackgroundPreferencesStorage {
  const SharedBackgroundPreferencesStorage();

  @override
  Future<String?> read(String key) async =>
      (await SharedPreferences.getInstance()).getString(key);

  @override
  Future<bool> write(String key, String value) async =>
      (await SharedPreferences.getInstance()).setString(key, value);
}

final class BackgroundPreferencesStore {
  const BackgroundPreferencesStore({
    this.storage = const SharedBackgroundPreferencesStorage(),
  });

  static const storageKey = 'anytty.background.native.v1';

  final BackgroundPreferencesStorage storage;

  Future<BackgroundPreferences> load() async {
    final raw = await storage.read(storageKey);
    if (raw == null || raw.isEmpty) return BackgroundPreferences.defaults;
    try {
      final value = jsonDecode(raw);
      if (value is! Map<String, dynamic>) {
        return BackgroundPreferences.defaults;
      }
      return BackgroundPreferences.fromJson(value);
    } catch (_) {
      return BackgroundPreferences.defaults;
    }
  }

  Future<BackgroundPreferences> save(BackgroundPreferences preferences) async {
    final saved = await storage.write(
      storageKey,
      jsonEncode(preferences.toJson()),
    );
    if (!saved) throw StateError('Background preferences could not be saved');
    return preferences;
  }
}
