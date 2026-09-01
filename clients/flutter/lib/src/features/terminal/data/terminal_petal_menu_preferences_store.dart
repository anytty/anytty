import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../domain/terminal_petal_menu_preferences.dart';

abstract interface class TerminalPetalMenuPreferencesStorage {
  Future<String?> read(String key);

  Future<bool> write(String key, String value);
}

final class SharedTerminalPetalMenuPreferencesStorage
    implements TerminalPetalMenuPreferencesStorage {
  const SharedTerminalPetalMenuPreferencesStorage();

  @override
  Future<String?> read(String key) async =>
      (await SharedPreferences.getInstance()).getString(key);

  @override
  Future<bool> write(String key, String value) async =>
      (await SharedPreferences.getInstance()).setString(key, value);
}

final class TerminalPetalMenuPreferencesStore {
  const TerminalPetalMenuPreferencesStore({
    this.storage = const SharedTerminalPetalMenuPreferencesStorage(),
  });

  static const storageKey = 'anytty.terminal.petal-menu.native.v1';

  final TerminalPetalMenuPreferencesStorage storage;

  Future<TerminalPetalMenuPreferences> load() async {
    final raw = await storage.read(storageKey);
    if (raw == null || raw.isEmpty) {
      return TerminalPetalMenuPreferences.defaults;
    }
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) {
        return TerminalPetalMenuPreferences.defaults;
      }
      return TerminalPetalMenuPreferences.fromJson(decoded);
    } catch (_) {
      return TerminalPetalMenuPreferences.defaults;
    }
  }

  Future<TerminalPetalMenuPreferences> save(
    TerminalPetalMenuPreferences preferences,
  ) async {
    final normalized = preferences.normalized();
    final saved = await storage.write(
      storageKey,
      jsonEncode(normalized.toJson()),
    );
    if (!saved) {
      throw StateError('Terminal petal menu preferences could not be saved');
    }
    return normalized;
  }
}
