import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/app_appearance.dart';
import 'package:anytty_native/src/app/app_appearance_store.dart';
import 'package:anytty_native/src/app/app_color_preferences.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AppAppearanceStore', () {
    test('defaults to dark to match the existing client', () async {
      final preferences = _MemoryAppearancePreferences();
      final store = AppAppearanceStore(preferences: preferences);

      expect(await store.load(), AppAppearance.dark);
      expect(AppAppearanceStore.storageKey, 'anytty.app.theme.v1');
    });

    test('persists and reloads each appearance mode', () async {
      final preferences = _MemoryAppearancePreferences();
      final store = AppAppearanceStore(preferences: preferences);

      for (final appearance in AppAppearance.values) {
        expect(await store.save(appearance), appearance);
        expect(await store.load(), appearance);
      }
    });

    test('rejects an unsuccessful preference write', () async {
      final store = AppAppearanceStore(
        preferences: _MemoryAppearancePreferences(writeSucceeds: false),
      );

      await expectLater(store.save(AppAppearance.light), throwsStateError);
    });
  });

  test('theme tokens use the adjustable cross-platform palette', () {
    final darkTheme = anyttyTheme(Brightness.dark);
    final lightTheme = anyttyTheme(Brightness.light);
    final dark = darkTheme.extension<AnyttyPalette>()!;
    final light = lightTheme.extension<AnyttyPalette>()!;

    expect(dark.background, AppColorPreferences.defaults.darkBackground);
    expect(dark.surface, AppColorPreferences.defaults.darkSurface);
    expect(dark.accent, AppColorPreferences.defaults.accent);
    expect(light.background, const Color(0xfff2f5f3));
    expect(light.surface, const Color(0xfffbfdfc));
    expect(darkTheme.colorScheme.primary, dark.accent);
    expect(lightTheme.colorScheme.primary, light.accent);
    expect(darkTheme.snackBarTheme.backgroundColor, dark.surfaceRaised);
    expect(darkTheme.snackBarTheme.contentTextStyle?.color, dark.text);
    expect(lightTheme.snackBarTheme.backgroundColor, light.surfaceRaised);
    expect(lightTheme.snackBarTheme.contentTextStyle?.color, light.text);
    expect(
      dark.text.computeLuminance(),
      greaterThan(dark.background.computeLuminance()),
    );
    expect(
      light.text.computeLuminance(),
      lessThan(light.background.computeLuminance()),
    );
  });

  test('custom RGB values drive dark workspace surfaces', () {
    const colors = AppColorPreferences(
      accent: Color(0xff819cff),
      darkBackground: Color(0xff182030),
      darkSurface: Color(0xff202a3c),
    );
    final theme = anyttyTheme(Brightness.dark, colors);
    final palette = theme.extension<AnyttyPalette>()!;

    expect(palette.background, colors.darkBackground);
    expect(palette.surface, colors.darkSurface);
    expect(palette.accent, colors.accent);
    expect(theme.scaffoldBackgroundColor, colors.darkBackground);
    expect(theme.colorScheme.surface, colors.darkSurface);
    expect(theme.colorScheme.primary, colors.accent);
  });
}

final class _MemoryAppearancePreferences implements AppAppearancePreferences {
  _MemoryAppearancePreferences({this.writeSucceeds = true});

  final bool writeSucceeds;
  final Map<String, String> _values = {};

  @override
  Future<String?> read(String key) async => _values[key];

  @override
  Future<bool> write(String key, String value) async {
    if (writeSucceeds) _values[key] = value;
    return writeSucceeds;
  }
}
