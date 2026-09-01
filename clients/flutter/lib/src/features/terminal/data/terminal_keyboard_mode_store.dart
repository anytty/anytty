import 'package:shared_preferences/shared_preferences.dart';

import '../domain/terminal_settings.dart';

final class TerminalKeyboardModeStore {
  const TerminalKeyboardModeStore();

  static const storagePrefix = 'anytty.terminal.keyboard-mode.v1';

  Future<TerminalKeyboardMode?> load({
    required String endpointId,
    required String terminalId,
  }) async {
    final preferences = await SharedPreferences.getInstance();
    final raw = preferences.getString(_storageKey(endpointId, terminalId));
    return switch (raw) {
      'auto' || 'automatic' => TerminalKeyboardMode.automatic,
      'resize' => TerminalKeyboardMode.resize,
      'shift' || 'cover' || 'overlay' => TerminalKeyboardMode.shift,
      _ => null,
    };
  }

  Future<TerminalKeyboardMode> save({
    required String endpointId,
    required String terminalId,
    required TerminalKeyboardMode mode,
  }) async {
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setString(
      _storageKey(endpointId, terminalId),
      _storedName(mode),
    );
    if (!saved) throw StateError('Terminal keyboard mode could not be saved');
    return mode;
  }

  String _storageKey(String endpointId, String terminalId) =>
      '$storagePrefix:${Uri.encodeComponent(endpointId)}:'
      '${Uri.encodeComponent(terminalId)}';

  String _storedName(TerminalKeyboardMode mode) => switch (mode) {
    TerminalKeyboardMode.automatic => 'auto',
    TerminalKeyboardMode.resize => 'resize',
    TerminalKeyboardMode.shift => 'shift',
  };
}
