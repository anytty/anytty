import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_ui_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('derives the complete route palette from the terminal theme', () {
    final palette = terminalUiPalette(resolveTerminalTheme('tokyo-night'));
    final theme = terminalUiThemeData(resolveTerminalTheme('tokyo-night'));

    expect(palette.background, const Color(0xff1a1b26));
    expect(palette.text, const Color(0xffa9b1d6));
    expect(palette.accent, const Color(0xff7dcfff));
    expect(theme.brightness, Brightness.dark);
    expect(theme.scaffoldBackgroundColor, palette.background);
    expect(theme.extension<AnyttyPalette>()?.accent, palette.accent);
  });
}
