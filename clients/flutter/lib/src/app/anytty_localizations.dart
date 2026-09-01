import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

final class AnyttyLocalizations {
  const AnyttyLocalizations(this.locale);

  final Locale locale;

  bool get isChinese => locale.languageCode == 'zh';

  String text({required String en, required String zh}) => isChinese ? zh : en;

  static AnyttyLocalizations of(BuildContext context) =>
      Localizations.of<AnyttyLocalizations>(context, AnyttyLocalizations) ??
      const AnyttyLocalizations(Locale('en'));

  static const delegate = _AnyttyLocalizationsDelegate();
  static const supportedLocales = [Locale('en'), Locale('zh', 'CN')];
}

String anyttyText(
  BuildContext context, {
  required String en,
  required String zh,
}) => AnyttyLocalizations.of(context).text(en: en, zh: zh);

final class _AnyttyLocalizationsDelegate
    extends LocalizationsDelegate<AnyttyLocalizations> {
  const _AnyttyLocalizationsDelegate();

  @override
  bool isSupported(Locale locale) =>
      const {'en', 'zh'}.contains(locale.languageCode);

  @override
  Future<AnyttyLocalizations> load(Locale locale) =>
      SynchronousFuture(AnyttyLocalizations(locale));

  @override
  bool shouldReload(_AnyttyLocalizationsDelegate old) => false;
}
