import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/features/browser/data/browser_session_store.dart';
import 'package:anytty_native/src/features/browser/domain/browser_session.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_new_tab_page.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_address_bar.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_session_screen.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_petal_menu.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  testWidgets(
    'closing the only unloaded tab shows and persists the start page',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      tester.view.physicalSize = const Size(400, 850);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        const MethodChannel('com.anytty.app/browser-proxy'),
        (_) async => null,
      );
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          const MethodChannel('com.anytty.app/browser-proxy'),
          null,
        ),
      );
      final store = _SessionStore();
      final container = ProviderContainer(
        overrides: [
          endpointSessionProvider('offline')
              .overrideWith((ref) => throw StateError('Device offline')),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: anyttyTheme(Brightness.light),
            home: BrowserSessionScreen(
              endpointId: 'offline',
              sessionStore: store,
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));
      expect(find.byType(TerminalPetalMenuRegion), findsNothing);
      expect(find.byType(BrowserNewTabPage), findsNothing);
      await tester.tap(find.byTooltip('Tabs'));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));
      await tester.tap(find.byTooltip('Close tab'));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));
      await tester.pump();
      expect(find.byType(BrowserNewTabPage), findsOneWidget);
      expect(
        tester
            .widget<BrowserAddressBar>(find.byType(BrowserAddressBar))
            .controller
            .text,
        isEmpty,
      );
      expect(store.snapshot.url, isEmpty);
      expect(store.snapshot.tabs.single.url, isEmpty);
      tester.view.physicalSize = const Size(1000, 400);
      await tester.pump();
      expect(find.byIcon(Icons.keyboard_return_rounded), findsOneWidget);
      expect(find.byIcon(Icons.arrow_back_rounded), findsOneWidget);
      expect(
        find.descendant(
          of: find.byType(AppBar),
          matching: find.byIcon(Icons.arrow_forward_rounded),
        ),
        findsOneWidget,
      );
      expect(find.byTooltip('Back to terminal'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pump();
      container.dispose();
    },
  );
}

class _SessionStore implements BrowserSessionStore {
  BrowserSessionSnapshot snapshot = BrowserSessionSnapshot.empty(
    sessionId: 'offline',
    endpointId: 'offline',
    endpointLabel: 'Offline',
  ).copyWith(url: 'http://192.168.1.1/unloaded', title: 'Unloaded page');

  @override
  Future<BrowserSessionSnapshot?> load(String sessionId) async => snapshot;
  @override
  Future<void> save(BrowserSessionSnapshot value) async {
    snapshot = value;
  }

  @override
  Future<void> remove(String sessionId) async {}
}
