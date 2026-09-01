import 'package:shared_preferences/shared_preferences.dart';

final class TerminalPinStore {
  const TerminalPinStore();

  static const _prefix = 'anytty.terminal-pins.v2:';

  Future<List<String>> load(String endpointId) async {
    if (endpointId.isEmpty) return const [];
    final preferences = await SharedPreferences.getInstance();
    return _normalize(preferences.getStringList('$_prefix$endpointId') ?? []);
  }

  Future<List<String>> save(String endpointId, List<String> terminalIds) async {
    final normalized = _normalize(terminalIds);
    if (endpointId.isEmpty) return normalized;
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setStringList(
      '$_prefix$endpointId',
      normalized,
    );
    if (!saved) throw StateError('Terminal pin order could not be saved');
    return normalized;
  }

  List<String> _normalize(List<String> values) {
    final seen = <String>{};
    return values
        .where((value) => value.isNotEmpty && seen.add(value))
        .toList(growable: false);
  }
}
