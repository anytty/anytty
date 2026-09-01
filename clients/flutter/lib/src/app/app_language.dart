import 'package:flutter/widgets.dart';

enum AppLanguage { system, english, simplifiedChinese }

extension AppLanguageLocale on AppLanguage {
  Locale? get locale => switch (this) {
    AppLanguage.system => null,
    AppLanguage.english => const Locale('en'),
    AppLanguage.simplifiedChinese => const Locale('zh', 'CN'),
  };

  String get storageValue => name;
}

AppLanguage parseAppLanguage(Object? value) => switch (value) {
  'english' => AppLanguage.english,
  'simplifiedChinese' => AppLanguage.simplifiedChinese,
  _ => AppLanguage.system,
};
