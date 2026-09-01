import 'package:flutter/material.dart';

import '../../../app/anytty_theme.dart';
import '../domain/terminal_settings.dart';

ThemeData terminalUiThemeData(TerminalTheme terminalTheme) {
  final brightness = terminalTheme.dark ? Brightness.dark : Brightness.light;
  final palette = terminalUiPalette(terminalTheme);
  final base = anyttyTheme(brightness);
  return base.copyWith(
    scaffoldBackgroundColor: palette.background,
    colorScheme: base.colorScheme.copyWith(
      primary: palette.accent,
      onPrimary: palette.accentText,
      surface: palette.surface,
      onSurface: palette.text,
      outline: palette.borderStrong,
      outlineVariant: palette.border,
      error: palette.danger,
      surfaceContainerLowest: palette.background,
      surfaceContainerLow: palette.surface,
      surfaceContainer: palette.surfaceRaised,
    ),
    extensions: [palette],
    appBarTheme: base.appBarTheme.copyWith(
      backgroundColor: palette.surface,
      foregroundColor: palette.text,
      systemOverlayStyle: anyttySystemUiOverlayStyle(
        brightness,
        palette: palette,
      ),
    ),
    bottomSheetTheme: base.bottomSheetTheme.copyWith(
      backgroundColor: palette.surface,
      modalBackgroundColor: palette.surface,
      modalBarrierColor: palette.overlay,
      dragHandleColor: palette.borderStrong,
    ),
  );
}

AnyttyPalette terminalUiPalette(TerminalTheme terminalTheme) {
  final background = _terminalColor(terminalTheme.background);
  final text = _terminalColor(terminalTheme.foreground);
  final surface = Color.lerp(
    background,
    text,
    terminalTheme.dark ? 0.07 : 0.04,
  )!;
  final accent = _terminalColor(terminalTheme.ansi[6]);
  return AnyttyPalette(
    background: background,
    surface: surface,
    surfaceRaised: Color.lerp(
      background,
      text,
      terminalTheme.dark ? 0.12 : 0.08,
    )!,
    border: Color.lerp(background, text, terminalTheme.dark ? 0.16 : 0.13)!,
    borderStrong: Color.lerp(
      background,
      text,
      terminalTheme.dark ? 0.28 : 0.24,
    )!,
    text: text,
    muted: Color.lerp(text, background, terminalTheme.dark ? 0.38 : 0.34)!,
    faint: Color.lerp(text, background, terminalTheme.dark ? 0.58 : 0.52)!,
    accent: accent,
    accentText: _contrastText(accent),
    success: _terminalColor(terminalTheme.ansi[2]),
    warning: _terminalColor(terminalTheme.ansi[3]),
    danger: _terminalColor(terminalTheme.ansi[1]),
    overlay: const Color(0x70000000),
  );
}

Color _terminalColor(int rgb) => Color(0xff000000 | rgb);

Color _contrastText(Color background) =>
    ThemeData.estimateBrightnessForColor(background) == Brightness.dark
    ? const Color(0xfff8faf9)
    : const Color(0xff07120f);
