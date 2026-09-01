import 'dart:convert';

import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/files/presentation/path_bookmarks_sheet.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets(
    'keeps the editor visible above the keyboard and unwinds sheets',
    (tester) async {
      tester.view.physicalSize = const Size(360, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(() {
        tester.view.resetPhysicalSize();
        tester.view.resetDevicePixelRatio();
        tester.view.resetViewInsets();
      });
      SharedPreferences.setMockInitialValues({
        'anytty.path-bookmarks.v1:endpoint-a': jsonEncode([
          {
            'id': 'bookmark-1',
            'path': '/srv/app',
            'label': 'Production',
            'createdAt': '2026-01-01T00:00:00.000Z',
            'updatedAt': '2026-01-01T00:00:00.000Z',
            'version': 1,
          },
        ]),
      });

      await tester.pumpWidget(
        MaterialApp(
          theme: anyttyTheme(Brightness.light),
          home: Builder(
            builder: (context) => Scaffold(
              body: Center(
                child: FilledButton(
                  onPressed: () => showPathBookmarks(
                    context: context,
                    endpointId: 'endpoint-a',
                    currentPath: '/srv/app',
                  ),
                  child: const Text('Open bookmarks'),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open bookmarks'));
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Edit bookmark Production'));
      await tester.pumpAndSettle();

      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pumpAndSettle();

      expect(find.text('Edit bookmark'), findsOneWidget);
      expect(find.text('Remove bookmark'), findsOneWidget);
      expect(
        tester.getBottomLeft(find.text('Remove bookmark')).dy,
        lessThan(500),
      );

      expect(await tester.binding.handlePopRoute(), isTrue);
      await tester.pumpAndSettle();
      expect(find.text('Edit bookmark'), findsNothing);
      expect(find.text('Bookmarks'), findsOneWidget);

      expect(await tester.binding.handlePopRoute(), isTrue);
      await tester.pumpAndSettle();
      expect(find.text('Bookmarks'), findsNothing);
    },
  );
}
