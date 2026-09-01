import 'package:flutter/material.dart';

enum AppAppearance { light, dark, system }

extension AppAppearanceThemeMode on AppAppearance {
  ThemeMode get themeMode => switch (this) {
    AppAppearance.dark => ThemeMode.dark,
    AppAppearance.light => ThemeMode.light,
    AppAppearance.system => ThemeMode.system,
  };

  String get storageValue => name;
}

AppAppearance parseAppAppearance(Object? value) => switch (value) {
  'light' => AppAppearance.light,
  'system' => AppAppearance.system,
  _ => AppAppearance.dark,
};
