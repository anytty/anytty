import 'package:anytty_native/src/features/terminal/data/terminal_settings_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'normalizes persisted terminal settings without a scrollback budget',
    () {
      final settings = TerminalSettings.fromJson({
        'fontSize': 90,
        'fontFamily': 'missing-font',
        'themeId': 'missing-theme',
        'scrollInertia': 'short',
        'scrollback': 50000,
        'scrollbackPrefetchThresholdRows': 42,
        'keyboardMode': 'overlay',
      });

      expect(settings.fontSize, 32);
      expect(settings.fontFamily, 'JetBrainsMonoNerd');
      expect(settings.themeId, 'anytty-dark');
      expect(settings.scrollInertia, 25);
      expect(settings.historyPrefetchThresholdRows, 42);
      expect(settings.keyboardMode, TerminalKeyboardMode.shift);
      expect(settings.toJson(), isNot(contains('scrollback')));
    },
  );

  test('keeps renderer metrics proportional to the selected font size', () {
    final small = defaultTerminalSettings.copyWith(fontSize: 8).metrics;
    final large = defaultTerminalSettings.copyWith(fontSize: 32).metrics;

    expect(small.cellWidth / small.fontSize, closeTo(8.44 / 14, 0.000001));
    expect(large.rowHeight / large.fontSize, closeTo(20 / 14, 0.000001));
  });

  test('preserves all legacy terminal palettes', () {
    expect(terminalThemes, hasLength(17));
    expect(terminalThemes.map((theme) => theme.id), contains('tokyo-night'));
    expect(terminalThemes.map((theme) => theme.id), contains('github-light'));
    expect(terminalThemes.every((theme) => theme.ansi.length == 16), isTrue);
  });

  test('preserves all legacy terminal font choices', () {
    expect(terminalFontFamilies, hasLength(6));
    expect(
      terminalFontFamilies,
      containsAll([
        'JetBrainsMonoNerd',
        'FiraCodeNerd',
        'CascadiaCodeNerd',
        'HackNerd',
        'IosevkaNerd',
        'monospace',
      ]),
    );
    expect(
      defaultTerminalSettings.copyWith(fontFamily: 'IosevkaNerd').fontFamily,
      'IosevkaNerd',
    );
  });

  test('provides a distinct display label for every terminal font', () {
    final labels = terminalFontFamilies.map(terminalFontLabel).toList();

    expect(labels, hasLength(terminalFontFamilies.length));
    expect(labels.toSet(), hasLength(terminalFontFamilies.length));
    expect(labels, containsAll(['Fira Code', 'System Mono']));
  });

  test('persists normalized native settings', () async {
    SharedPreferences.setMockInitialValues({});
    const store = TerminalSettingsStore();
    final expected = defaultTerminalSettings.copyWith(
      fontSize: 18,
      themeId: 'dracula',
      cursorBlink: false,
    );

    expect(await store.save(expected), expected);
    expect(await store.load(), expected);
  });

  test('interpolates the legacy inertia profile continuously', () {
    expect(resolveTerminalMomentumProfile(0).enabled, isFalse);
    expect(resolveTerminalMomentumProfile(25).deceleration, 0.95);
    expect(resolveTerminalMomentumProfile(60).minimumVelocity, 20);
    expect(
      resolveTerminalMomentumProfile(50).deceleration,
      greaterThan(resolveTerminalMomentumProfile(49).deceleration),
    );
  });

  test('uses TUI-aware keyboard layout only in automatic mode', () {
    expect(
      resolveTerminalKeyboardMode(
        TerminalKeyboardMode.automatic,
        alternateScreen: false,
      ),
      TerminalKeyboardMode.shift,
    );
    expect(
      resolveTerminalKeyboardMode(
        TerminalKeyboardMode.automatic,
        alternateScreen: true,
      ),
      TerminalKeyboardMode.resize,
    );
    expect(
      resolveTerminalKeyboardMode(
        TerminalKeyboardMode.shift,
        alternateScreen: true,
      ),
      TerminalKeyboardMode.shift,
    );
  });
}
