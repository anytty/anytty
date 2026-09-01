import 'dart:convert';

import 'package:anytty_native/src/app/app_color_preferences.dart';
import 'package:anytty_native/src/app/app_color_preferences_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('color preferences default to the accepted cyan palette', () async {
    final storage = _MemoryColorStorage();
    final store = AppColorPreferencesStore(storage: storage);

    expect(await store.load(), AppColorPreferences.defaults);
    expect(AppColorPreferencesStore.storageKey, 'anytty.app.colors.v1');
    expect(colorHex(AppColorPreferences.defaults.accent), '#32D5D0');
  });

  test('color preferences persist and reload RGB tokens', () async {
    final storage = _MemoryColorStorage();
    final store = AppColorPreferencesStore(storage: storage);
    const colors = AppColorPreferences(
      accent: Color(0xff32b9ef),
      darkBackground: Color(0xff121820),
      darkSurface: Color(0xff1c2630),
    );

    expect(await store.save(colors), colors);
    expect(await store.load(), colors);
    expect(jsonDecode(storage.values[AppColorPreferencesStore.storageKey]!), {
      'version': 1,
      'accent': '#32B9EF',
      'darkBackground': '#121820',
      'darkSurface': '#1C2630',
    });
  });

  test('malformed color preferences fall back without throwing', () async {
    final storage = _MemoryColorStorage()
      ..values[AppColorPreferencesStore.storageKey] = '{bad json';
    final store = AppColorPreferencesStore(storage: storage);

    expect(await store.load(), AppColorPreferences.defaults);
  });

  test('failed color writes surface an error', () async {
    final store = AppColorPreferencesStore(
      storage: _MemoryColorStorage(writeSucceeds: false),
    );

    await expectLater(
      store.save(AppColorPreferences.defaults),
      throwsStateError,
    );
  });
}

final class _MemoryColorStorage implements AppColorPreferencesStorage {
  _MemoryColorStorage({this.writeSucceeds = true});

  final bool writeSucceeds;
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<bool> write(String key, String value) async {
    if (writeSucceeds) values[key] = value;
    return writeSucceeds;
  }
}
