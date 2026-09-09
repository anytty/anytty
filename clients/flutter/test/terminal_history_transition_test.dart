import 'package:anytty_native/src/features/terminal/presentation/terminal_canvas.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_history_transition.dart';
import 'package:anytty_native/src/shared/presentation/anytty_brand_mark.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget surface({
    bool active = true,
    bool ready = false,
    VoidCallback? onCancel,
    bool reducedMotion = false,
    double textScale = 1,
    Color background = Colors.black,
  }) => MaterialApp(
    home: MediaQuery(
      data: MediaQueryData(
        disableAnimations: reducedMotion,
        textScaler: TextScaler.linear(textScale),
      ),
      child: Scaffold(
        body: TerminalHistoryTransition(
          active: active,
          ready: ready,
          background: background,
          foreground: background == Colors.black ? Colors.white : Colors.black,
          onCancel: onCancel,
          fallback: const Text('Live output'),
          child: const Text('History output'),
        ),
      ),
    ),
  );

  bool revealed(WidgetTester tester) => tester
      .widget<TerminalHistoryPresentation>(
        find.byType(TerminalHistoryPresentation),
      )
      .ready;

  testWidgets('fast history stays covered for 700ms then fades out', (
    tester,
  ) async {
    await tester.pumpWidget(surface());
    expect(find.byType(AnyttyBrandLoader), findsOneWidget);
    final position = tester.getRect(find.byType(AnyttyBrandLoader));
    await tester.pumpWidget(surface(ready: true));
    await tester.pump(const Duration(milliseconds: 699));
    expect(revealed(tester), isFalse);
    expect(tester.getRect(find.byType(AnyttyBrandLoader)), position);
    await tester.pump(const Duration(milliseconds: 1));
    expect(revealed(tester), isTrue);
    await tester.pumpAndSettle();
    expect(find.byType(AnyttyBrandLoader), findsNothing);
  });

  testWidgets(
    'slow history waits for positioned content without another delay',
    (tester) async {
      await tester.pumpWidget(surface());
      await tester.pump(const Duration(seconds: 3));
      expect(revealed(tester), isFalse);
      expect(find.byType(AnyttyBrandLoader), findsOneWidget);
      await tester.pumpWidget(surface(ready: true));
      expect(revealed(tester), isTrue);
      await tester.pumpAndSettle();
      expect(find.byType(AnyttyBrandLoader), findsNothing);
      await tester.pumpWidget(surface(ready: true));
      expect(find.byType(AnyttyBrandLoader), findsNothing);
    },
  );

  testWidgets(
    'cancel is available immediately and reentry gets a fresh timer',
    (tester) async {
      var cancelled = false;
      await tester.pumpWidget(surface(onCancel: () => cancelled = true));
      await tester.tap(find.byType(IconButton));
      expect(cancelled, isTrue);
      await tester.pump(const Duration(milliseconds: 500));
      await tester.pumpWidget(surface(active: false));
      expect(find.byType(AnyttyBrandLoader), findsNothing);
      await tester.pumpWidget(surface(ready: true));
      await tester.pump(const Duration(milliseconds: 200));
      expect(revealed(tester), isFalse);
      await tester.pump(const Duration(milliseconds: 500));
      expect(revealed(tester), isTrue);
      await tester.pumpAndSettle();
    },
  );

  testWidgets('failure or disposal removes the pending entry timer', (
    tester,
  ) async {
    await tester.pumpWidget(surface());
    await tester.pumpWidget(surface(active: false));
    await tester.pump(const Duration(seconds: 1));
    expect(find.byType(AnyttyBrandLoader), findsNothing);
    expect(revealed(tester), isFalse);
    await tester.pumpWidget(surface());
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 1));
    expect(tester.takeException(), isNull);
  });

  for (final size in [const Size(375, 667), const Size(667, 375)]) {
    for (final background in [Colors.black, Colors.white]) {
      testWidgets('reduced motion and large type at $size / $background', (
        tester,
      ) async {
        tester.view.physicalSize = size;
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        await tester.pumpWidget(
          surface(
            ready: true,
            reducedMotion: true,
            textScale: 3,
            background: background,
            onCancel: () {},
          ),
        );
        expect(revealed(tester), isFalse);
        expect(tester.takeException(), isNull);
        await tester.pump(const Duration(milliseconds: 700));
        await tester.pump();
        expect(revealed(tester), isTrue);
        expect(find.byType(AnyttyBrandLoader), findsNothing);
        expect(tester.takeException(), isNull);
      });
    }
  }
}
