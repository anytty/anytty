import 'package:anytty_native/src/app/background_preferences.dart';
import 'package:anytty_native/src/app/background_preferences_store.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'background preferences default to connection retention without alerts',
    () {
      expect(BackgroundPreferences.defaults.keepConnections, isTrue);
      expect(BackgroundPreferences.defaults.notifications, isFalse);
    },
  );

  test('background preferences persist and reload', () async {
    final storage = _MemoryStorage();
    final store = BackgroundPreferencesStore(storage: storage);
    const value = BackgroundPreferences(
      keepConnections: false,
      notifications: true,
    );

    expect(await store.save(value), value);
    expect(await store.load(), value);
    expect(storage.values, contains(BackgroundPreferencesStore.storageKey));
  });

  test('invalid background preferences fall back without throwing', () async {
    final storage = _MemoryStorage()
      ..values[BackgroundPreferencesStore.storageKey] = '{invalid';
    final store = BackgroundPreferencesStore(storage: storage);

    expect(await store.load(), BackgroundPreferences.defaults);
  });

  test('failed background preference writes surface an error', () async {
    final store = BackgroundPreferencesStore(
      storage: _MemoryStorage(writeSucceeds: false),
    );

    await expectLater(
      store.save(BackgroundPreferences.defaults),
      throwsStateError,
    );
  });
}

final class _MemoryStorage implements BackgroundPreferencesStorage {
  _MemoryStorage({this.writeSucceeds = true});

  final bool writeSucceeds;
  final Map<String, String> values = <String, String>{};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<bool> write(String key, String value) async {
    if (writeSucceeds) values[key] = value;
    return writeSucceeds;
  }
}
