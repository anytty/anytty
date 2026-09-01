import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

final class TerminalQuickKeysLayoutStore {
  const TerminalQuickKeysLayoutStore();

  static const storageKey = 'anytty.terminal.quick-keys.layout.native.v1';

  Future<List<String>> load() async {
    final preferences = await SharedPreferences.getInstance();
    final raw = preferences.getString(storageKey);
    if (raw == null || raw.isEmpty) return const [];
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) return const [];
      final values = decoded['ids'];
      if (values is! List) return const [];
      return _normalize(values.whereType<String>());
    } catch (_) {
      return const [];
    }
  }

  Future<List<String>> save(Iterable<String> ids) async {
    final normalized = _normalize(ids);
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setString(
      storageKey,
      jsonEncode({'version': 1, 'ids': normalized}),
    );
    if (!saved) throw StateError('Quick Keys layout could not be saved');
    return normalized;
  }

  List<String> _normalize(Iterable<String> ids) {
    final seen = <String>{};
    return List.unmodifiable([
      for (final id in ids)
        if (id.isNotEmpty && seen.add(id)) id,
    ]);
  }
}
