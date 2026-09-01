import 'package:anytty_native/src/app/app_language.dart';
import 'package:anytty_native/src/app/app_language_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('follows the system when no language was selected', () async {
    SharedPreferences.setMockInitialValues({});

    expect(await const AppLanguageStore().load(), AppLanguage.system);
  });

  test('persists an explicit local language', () async {
    SharedPreferences.setMockInitialValues({});
    const store = AppLanguageStore();

    await store.save(AppLanguage.simplifiedChinese);

    expect(await store.load(), AppLanguage.simplifiedChinese);
    expect(AppLanguage.simplifiedChinese.locale?.languageCode, 'zh');
  });
}
