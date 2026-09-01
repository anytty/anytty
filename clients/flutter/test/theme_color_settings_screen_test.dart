import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/app_color_preferences.dart';
import 'package:anytty_native/src/app/app_color_preferences_store.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/features/settings/presentation/theme_color_settings_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  testWidgets('edits RGB tokens and updates the global dark palette', (
    tester,
  ) async {
    _useView(tester, const Size(390, 844));
    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();

    expect(find.text('Theme colors'), findsOneWidget);
    expect(find.text('#32D5D0'), findsWidgets);
    expect(find.byKey(const ValueKey('theme-color-plane')), findsOneWidget);

    await tester.tap(find.text('Sky'));
    await tester.pumpAndSettle();
    var colors = _preferences(tester);
    expect(colors.accent, const Color(0xff32b9ef));
    var palette = Theme.of(
      tester.element(find.byType(ThemeColorSettingsScreen)),
    ).extension<AnyttyPalette>()!;
    expect(palette.accent, const Color(0xff32b9ef));

    await tester.tap(find.byKey(const ValueKey('theme-token-background')));
    await tester.pumpAndSettle();
    expect(find.text('#0E1012'), findsOneWidget);
    await tester.enterText(
      find.descendant(
        of: find.byKey(const ValueKey('theme-rgb-background-r')),
        matching: find.byType(EditableText),
      ),
      '24',
    );
    await tester.pumpAndSettle();

    colors = _preferences(tester);
    expect(colors.darkBackground, const Color(0xff181012));
    palette = Theme.of(tester.element(find.byType(ThemeColorSettingsScreen)))
        .extension<AnyttyPalette>()!;
    expect(palette.background, const Color(0xff181012));

    await tester.pump(const Duration(milliseconds: 300));
    final persisted = await const AppColorPreferencesStore().load();
    expect(persisted.darkBackground, const Color(0xff181012));
    expect(persisted.accent, const Color(0xff32b9ef));
  });

  testWidgets('remains usable on a narrow phone at 200 percent text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    tester.platformDispatcher.textScaleFactorTestValue = 2;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);

    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.byKey(const ValueKey('theme-color-plane')),
      180,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(
      tester.getSize(find.byKey(const ValueKey('theme-token-accent'))).height,
      greaterThanOrEqualTo(42),
    );
  });
}

void _useView(WidgetTester tester, Size size) {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
}

AppColorPreferences _preferences(WidgetTester tester) {
  final container = ProviderScope.containerOf(
    tester.element(find.byType(ThemeColorSettingsScreen)),
  );
  return container.read(appColorPreferencesProvider).requireValue;
}

final class _Harness extends ConsumerWidget {
  const _Harness();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colors =
        ref.watch(appColorPreferencesProvider).valueOrNull ??
        AppColorPreferences.defaults;
    return MaterialApp(
      theme: anyttyTheme(Brightness.dark, colors),
      home: const ThemeColorSettingsScreen(),
    );
  }
}
