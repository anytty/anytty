import 'package:flutter/material.dart';

@immutable
final class AppColorPreferences {
  const AppColorPreferences({
    required this.accent,
    required this.darkBackground,
    required this.darkSurface,
  });

  static const version = 1;

  static const defaults = AppColorPreferences(
    accent: Color(0xff32d5d0),
    darkBackground: Color(0xff0e1012),
    darkSurface: Color(0xff181b1e),
  );

  final Color accent;
  final Color darkBackground;
  final Color darkSurface;

  AppColorPreferences copyWith({
    Color? accent,
    Color? darkBackground,
    Color? darkSurface,
  }) => AppColorPreferences(
    accent: accent ?? this.accent,
    darkBackground: darkBackground ?? this.darkBackground,
    darkSurface: darkSurface ?? this.darkSurface,
  );

  Map<String, Object> toJson() => {
    'version': version,
    'accent': colorHex(accent),
    'darkBackground': colorHex(darkBackground),
    'darkSurface': colorHex(darkSurface),
  };

  factory AppColorPreferences.fromJson(Map<String, Object?> json) {
    if (json['version'] != version) return defaults;
    return AppColorPreferences(
      accent: _parseColor(json['accent'], defaults.accent),
      darkBackground: _parseColor(
        json['darkBackground'],
        defaults.darkBackground,
      ),
      darkSurface: _parseColor(json['darkSurface'], defaults.darkSurface),
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AppColorPreferences &&
          accent == other.accent &&
          darkBackground == other.darkBackground &&
          darkSurface == other.darkSurface;

  @override
  int get hashCode => Object.hash(accent, darkBackground, darkSurface);
}

String colorHex(Color color) {
  final value = color.toARGB32() & 0x00ffffff;
  return '#${value.toRadixString(16).padLeft(6, '0').toUpperCase()}';
}

Color _parseColor(Object? value, Color fallback) {
  if (value is! String) return fallback;
  final hex = value.trim().replaceFirst('#', '');
  if (hex.length != 6) return fallback;
  final parsed = int.tryParse(hex, radix: 16);
  return parsed == null ? fallback : Color(0xff000000 | parsed);
}
