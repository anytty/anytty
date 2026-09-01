import 'dart:convert';

import 'package:anytty_native/src/app/anytty_localizations.dart';
import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/features/settings/presentation/terminal_petal_menu_settings_screen.dart';
import 'package:anytty_native/src/features/terminal/data/terminal_petal_menu_preferences_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_petal_menu_preferences.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  testWidgets('reorders the current eight-slot ring and offers undo', (
    tester,
  ) async {
    _useView(tester, const Size(390, 844));
    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();

    expect(find.text('Petal menu'), findsOneWidget);
    expect(find.text('Current menu'), findsOneWidget);
    expect(find.text('Action library'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('petal-editor-action-search')),
      findsOneWidget,
    );
    expect(find.byKey(const ValueKey('petal-editor-page-next')), findsNothing);

    await tester.tap(find.byKey(const ValueKey('petal-menu-haptics')));
    await tester.pumpAndSettle();
    await _longPressDrag(
      tester,
      find.byKey(const ValueKey('petal-editor-action-paste')),
      find.byKey(const ValueKey('petal-editor-action-search')),
    );

    var value = await _preferences(tester);
    expect(value.hapticsEnabled, isFalse);
    expect(value.visibleRootActionIds.take(4), [
      'history',
      'paste',
      'search',
      'selection',
    ]);
    expect(find.text('Undo'), findsOneWidget);

    await tester.tap(find.text('Undo'));
    await tester.pumpAndSettle();
    value = await _preferences(tester);
    expect(value.visibleRootActionIds.take(4), [
      'history',
      'search',
      'selection',
      'paste',
    ]);
  });

  testWidgets('opens the second and third petal rings directly', (
    tester,
  ) async {
    _useView(tester, const Size(390, 844));
    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const ValueKey('petal-editor-action-more')));
    await tester.pumpAndSettle();
    expect(
      find.byKey(const ValueKey('petal-editor-action-input-tools')),
      findsOneWidget,
    );
    expect(find.text('Level 2'), findsOneWidget);

    await tester.tap(
      find.byKey(const ValueKey('petal-editor-action-input-tools')),
    );
    await tester.pumpAndSettle();
    expect(
      find.byKey(const ValueKey('petal-editor-action-tab')),
      findsOneWidget,
    );
    expect(find.text('Level 3'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('petal-editor-hub')));
    await tester.pumpAndSettle();
    expect(
      find.byKey(const ValueKey('petal-editor-action-input-tools')),
      findsOneWidget,
    );
  });

  testWidgets('drags actions out to the library and back into an empty slot', (
    tester,
  ) async {
    _useView(tester, const Size(390, 844));
    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();

    await _longPressDrag(
      tester,
      find.byKey(const ValueKey('petal-editor-action-search')),
      find.byKey(const ValueKey('petal-action-library-drop-zone')),
    );
    var value = await _preferences(tester);
    expect(value.hiddenActionIds, contains('search'));
    expect(
      find.byKey(const ValueKey('petal-library-action-search')),
      findsOneWidget,
    );
    final feedback = tester.widget<SnackBar>(find.byType(SnackBar));
    expect(feedback.persist, isFalse);
    expect(feedback.duration, const Duration(seconds: 4));
    expect(find.text('Search moved to the library'), findsOneWidget);
    await tester.pump(const Duration(seconds: 4));
    await tester.pumpAndSettle();
    expect(find.text('Search moved to the library'), findsNothing);

    await _longPressDrag(
      tester,
      find.byKey(const ValueKey('petal-library-action-search')),
      find.byKey(const ValueKey('petal-editor-slot-7')),
    );

    value = await _preferences(tester);
    expect(value.hiddenActionIds, isNot(contains('search')));
    expect(value.visibleRootActionIds.last, 'search');
  });

  testWidgets('builds child and grandchild rings by dragging library actions', (
    tester,
  ) async {
    _useView(tester, const Size(390, 844));
    final seeded = TerminalPetalMenuPreferences.defaults.copyWith(
      hiddenActionIds: const {'search', 'history'},
    );
    SharedPreferences.setMockInitialValues({
      TerminalPetalMenuPreferencesStore.storageKey: jsonEncode(seeded.toJson()),
    });
    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();

    await _longPressDrag(
      tester,
      find.byKey(const ValueKey('petal-library-action-search')),
      find.byKey(const ValueKey('petal-editor-action-selection')),
    );
    expect(
      find.byKey(const ValueKey('petal-editor-action-search')),
      findsOneWidget,
    );
    expect(find.text('Level 2'), findsOneWidget);

    await _longPressDrag(
      tester,
      find.byKey(const ValueKey('petal-library-action-history')),
      find.byKey(const ValueKey('petal-editor-action-search')),
    );
    expect(
      find.byKey(const ValueKey('petal-editor-action-history')),
      findsOneWidget,
    );
    expect(find.text('Level 3'), findsOneWidget);

    var value = await _preferences(tester);
    expect(value.layout.firstWhere((item) => item.id == 'search').depth, 1);
    expect(value.layout.firstWhere((item) => item.id == 'history').depth, 2);

    await _longPressDrag(
      tester,
      find.byKey(const ValueKey('petal-editor-action-history')),
      find.byKey(const ValueKey('petal-editor-hub')),
    );
    value = await _preferences(tester);
    expect(value.layout.firstWhere((item) => item.id == 'history').depth, 1);
    expect(find.text('Level 2'), findsOneWidget);
  });

  testWidgets('confirms before restoring the built-in petal configuration', (
    tester,
  ) async {
    _useView(tester, const Size(390, 844));
    final seeded = TerminalPetalMenuPreferences.defaults.copyWith(
      hapticsEnabled: false,
      hiddenActionIds: const {'search'},
    );
    SharedPreferences.setMockInitialValues({
      TerminalPetalMenuPreferencesStore.storageKey: jsonEncode(seeded.toJson()),
    });
    await tester.pumpWidget(const ProviderScope(child: _Harness()));
    await tester.pumpAndSettle();

    final restore = find.byKey(const ValueKey('petal-menu-restore-defaults'));
    await tester.tap(restore);
    await tester.pumpAndSettle();
    expect(find.text('Restore the default petal menu?'), findsOneWidget);

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(await _preferences(tester), seeded);

    await tester.tap(restore);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('petal-menu-restore-confirm')));
    await tester.pumpAndSettle();

    expect(await _preferences(tester), TerminalPetalMenuPreferences.defaults);
    expect(find.text('Petal menu restored'), findsOneWidget);
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
      find.byKey(const ValueKey('petal-layout-editor')),
      180,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    const firstPageIds = <String>[
      'history',
      'search',
      'selection',
      'paste',
      'quick-keys',
      'keyboard',
      'resources',
      'more',
    ];
    final centers = [
      for (final id in firstPageIds)
        tester.getCenter(find.byKey(ValueKey('petal-editor-action-$id'))),
    ];
    for (var first = 0; first < centers.length; first += 1) {
      for (var second = first + 1; second < centers.length; second += 1) {
        expect(
          (centers[first] - centers[second]).distance,
          greaterThanOrEqualTo(55),
        );
      }
    }
  });
}

void _useView(WidgetTester tester, Size size) {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
}

Future<void> _longPressDrag(
  WidgetTester tester,
  Finder source,
  Finder target,
) async {
  final gesture = await tester.startGesture(tester.getCenter(source));
  await tester.pump(const Duration(milliseconds: 320));
  await gesture.moveTo(tester.getCenter(target));
  await tester.pump(const Duration(milliseconds: 60));
  await gesture.up();
  await tester.pumpAndSettle();
}

Future<TerminalPetalMenuPreferences> _preferences(WidgetTester tester) {
  final container = ProviderScope.containerOf(
    tester.element(find.byType(TerminalPetalMenuSettingsScreen)),
  );
  return container.read(terminalPetalMenuPreferencesProvider.future);
}

final class _Harness extends StatelessWidget {
  const _Harness();

  @override
  Widget build(BuildContext context) => MaterialApp(
    theme: anyttyTheme(Brightness.dark),
    localizationsDelegates: const [
      AnyttyLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: AnyttyLocalizations.supportedLocales,
    home: const TerminalPetalMenuSettingsScreen(),
  );
}
