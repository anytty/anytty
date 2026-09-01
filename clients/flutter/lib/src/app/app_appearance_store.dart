import 'package:shared_preferences/shared_preferences.dart';

import 'app_appearance.dart';

abstract interface class AppAppearancePreferences {
  Future<String?> read(String key);

  Future<bool> write(String key, String value);
}

final class SharedAppAppearancePreferences implements AppAppearancePreferences {
  const SharedAppAppearancePreferences();

  @override
  Future<String?> read(String key) async =>
      (await SharedPreferences.getInstance()).getString(key);

  @override
  Future<bool> write(String key, String value) async =>
      (await SharedPreferences.getInstance()).setString(key, value);
}

final class AppAppearanceStore {
  const AppAppearanceStore({
    this.preferences = const SharedAppAppearancePreferences(),
  });

  static const storageKey = 'anytty.app.theme.v1';

  final AppAppearancePreferences preferences;

  Future<AppAppearance> load() async =>
      parseAppAppearance(await preferences.read(storageKey));

  Future<AppAppearance> save(AppAppearance appearance) async {
    final saved = await preferences.write(storageKey, appearance.storageValue);
    if (!saved) throw StateError('App appearance could not be saved');
    return appearance;
  }
}
