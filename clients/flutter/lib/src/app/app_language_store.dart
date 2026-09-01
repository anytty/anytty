import 'package:shared_preferences/shared_preferences.dart';

import 'app_language.dart';

final class AppLanguageStore {
  const AppLanguageStore();

  static const storageKey = 'anytty.app.language.v1';

  Future<AppLanguage> load() async {
    final preferences = await SharedPreferences.getInstance();
    return parseAppLanguage(preferences.getString(storageKey));
  }

  Future<AppLanguage> save(AppLanguage language) async {
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setString(
      storageKey,
      language.storageValue,
    );
    if (!saved) throw StateError('App language could not be saved');
    return language;
  }
}
