import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../domain/terminal_settings.dart';

final class TerminalSettingsStore {
  const TerminalSettingsStore();

  static const storageKey = 'anytty.terminal.settings.native.v1';

  Future<TerminalSettings> load() async {
    final preferences = await SharedPreferences.getInstance();
    final raw = preferences.getString(storageKey);
    if (raw == null || raw.isEmpty) return defaultTerminalSettings;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) return defaultTerminalSettings;
      return TerminalSettings.fromJson(decoded);
    } catch (_) {
      return defaultTerminalSettings;
    }
  }

  Future<TerminalSettings> save(TerminalSettings settings) async {
    final normalized = settings.normalized();
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setString(
      storageKey,
      jsonEncode(normalized.toJson()),
    );
    if (!saved) throw StateError('Terminal settings could not be saved');
    return normalized;
  }
}
