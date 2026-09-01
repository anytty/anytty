import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../domain/terminal_quick_action.dart';

final class TerminalQuickActionStore {
  const TerminalQuickActionStore();

  static const storageKey = 'anytty.terminal.quick-actions.native.v1';

  Future<List<TerminalQuickAction>> load() async {
    final preferences = await SharedPreferences.getInstance();
    final raw = preferences.getString(storageKey);
    if (raw == null || raw.isEmpty) return defaultTerminalQuickActions;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) return defaultTerminalQuickActions;
      final values = decoded['actions'];
      if (values is! List) return defaultTerminalQuickActions;
      final seen = <String>{};
      final actions = <TerminalQuickAction>[];
      for (final value in values) {
        if (value is! Map) continue;
        final action = TerminalQuickAction.fromJson(
          value.map((key, value) => MapEntry(key.toString(), value)),
        );
        if (action != null && seen.add(action.id)) actions.add(action);
      }
      return List.unmodifiable(actions);
    } catch (_) {
      return defaultTerminalQuickActions;
    }
  }

  Future<List<TerminalQuickAction>> save(
    List<TerminalQuickAction> actions,
  ) async {
    final normalized = _normalize(actions);
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setString(
      storageKey,
      jsonEncode({
        'version': 1,
        'actions': normalized.map((action) => action.toJson()).toList(),
      }),
    );
    if (!saved) throw StateError('Terminal quick actions could not be saved');
    return normalized;
  }

  List<TerminalQuickAction> _normalize(List<TerminalQuickAction> actions) {
    final seen = <String>{};
    return List.unmodifiable([
      for (final action in actions)
        if (action.isValid && seen.add(action.id)) action,
    ]);
  }
}
