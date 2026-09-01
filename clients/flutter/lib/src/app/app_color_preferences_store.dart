import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'app_color_preferences.dart';

abstract interface class AppColorPreferencesStorage {
  Future<String?> read(String key);

  Future<bool> write(String key, String value);
}

final class SharedAppColorPreferencesStorage
    implements AppColorPreferencesStorage {
  const SharedAppColorPreferencesStorage();

  @override
  Future<String?> read(String key) async =>
      (await SharedPreferences.getInstance()).getString(key);

  @override
  Future<bool> write(String key, String value) async =>
      (await SharedPreferences.getInstance()).setString(key, value);
}

final class AppColorPreferencesStore {
  const AppColorPreferencesStore({
    this.storage = const SharedAppColorPreferencesStorage(),
  });

  static const storageKey = 'anytty.app.colors.v1';

  final AppColorPreferencesStorage storage;

  Future<AppColorPreferences> load() async {
    final raw = await storage.read(storageKey);
    if (raw == null) return AppColorPreferences.defaults;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) {
        return AppColorPreferences.defaults;
      }
      return AppColorPreferences.fromJson(decoded);
    } on FormatException {
      return AppColorPreferences.defaults;
    }
  }

  Future<AppColorPreferences> save(AppColorPreferences preferences) async {
    final saved = await storage.write(
      storageKey,
      jsonEncode(preferences.toJson()),
    );
    if (!saved) throw StateError('App color preferences could not be saved');
    return preferences;
  }
}
