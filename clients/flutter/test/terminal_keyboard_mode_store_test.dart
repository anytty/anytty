import 'package:anytty_native/src/features/terminal/data/terminal_keyboard_mode_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('stores a keyboard mode independently for each terminal', () async {
    SharedPreferences.setMockInitialValues({});
    const store = TerminalKeyboardModeStore();

    await store.save(
      endpointId: 'studio',
      terminalId: 'shell',
      mode: TerminalKeyboardMode.shift,
    );
    await store.save(
      endpointId: 'studio',
      terminalId: 'editor',
      mode: TerminalKeyboardMode.resize,
    );

    expect(
      await store.load(endpointId: 'studio', terminalId: 'shell'),
      TerminalKeyboardMode.shift,
    );
    expect(
      await store.load(endpointId: 'studio', terminalId: 'editor'),
      TerminalKeyboardMode.resize,
    );
    expect(await store.load(endpointId: 'server', terminalId: 'shell'), isNull);
  });

  test('reads legacy cover and overlay values as shift', () async {
    SharedPreferences.setMockInitialValues({
      'anytty.terminal.keyboard-mode.v1:legacy:shell': 'cover',
      'anytty.terminal.keyboard-mode.v1:legacy:editor': 'overlay',
    });
    const store = TerminalKeyboardModeStore();

    expect(
      await store.load(endpointId: 'legacy', terminalId: 'shell'),
      TerminalKeyboardMode.shift,
    );
    expect(
      await store.load(endpointId: 'legacy', terminalId: 'editor'),
      TerminalKeyboardMode.shift,
    );
  });
}
