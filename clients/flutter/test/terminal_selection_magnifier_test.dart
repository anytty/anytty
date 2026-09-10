import 'package:anytty_native/src/features/terminal/domain/history_interaction.dart';
import 'package:anytty_native/src/features/terminal/domain/history_store.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_canvas.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final viewport in [const Size(375, 500), const Size(667, 240)]) {
    testWidgets('magnifier follows selection and hides on exit in $viewport', (
      tester,
    ) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);
      final rows = [
        for (var i = 0; i < 60; i++)
          HistoryRow(
            logicalLineId: Int64(i + 1),
            row: ScreenRow(
              cells: [ScreenCell(content: 'hello world', width: 11)],
            ),
          ),
      ];
      final layout = HistoryLayout.fromHistory(
        FrozenHistory(
          token: 'magnifier',
          generation: Int64.ONE,
          cols: 40,
          rows: rows,
          anchor: HistoryAnchor(
            logicalLineId: Int64(60),
            cellOffset: 0,
            atEnd: true,
          ),
          hasMore: false,
          logicalTotal: 60,
        ),
      );
      var enabled = true;
      late StateSetter rebuild;
      await tester.pumpWidget(
        MaterialApp(
          home: Center(
            child: SizedBox(
              width: viewport.width,
              height: viewport.height,
              child: StatefulBuilder(
                builder: (context, setState) {
                  rebuild = setState;
                  return TerminalHistoryCanvas(
                    rows: rows,
                    cols: 40,
                    scrollController: controller,
                    layout: layout,
                    selectionEnabled: enabled,
                  );
                },
              ),
            ),
          ),
        ),
      );
      final canvas = tester.getRect(find.byType(TerminalHistoryCanvas));
      final lens = find.byKey(const ValueKey('terminal-selection-magnifier'));
      expect(lens, findsNothing);
      var point = canvas.center;
      var gesture = await tester.startGesture(point);
      point += const Offset(30, 0);
      await gesture.moveTo(point);
      await tester.pump();
      expect(lens, findsOneWidget);
      void checkSource() {
        final magnifier = tester.widget<RawMagnifier>(lens);
        final bounds = tester.getRect(lens);
        expect(bounds.left, greaterThanOrEqualTo(canvas.left));
        expect(bounds.right, lessThanOrEqualTo(canvas.right));
        expect(bounds.top, greaterThanOrEqualTo(canvas.top));
        expect(bounds.bottom, lessThanOrEqualTo(canvas.bottom));
        expect(
          (bounds.center + magnifier.focalPointOffset - point).distance,
          lessThan(0.01),
        );
        expect(magnifier.magnificationScale, greaterThan(1));
      }

      checkSource();
      expect(tester.getRect(lens).bottom, lessThan(point.dy));
      point = canvas.topLeft + const Offset(2, 2);
      await gesture.moveTo(point);
      await tester.pump();
      checkSource();
      expect(tester.getRect(lens).top, greaterThan(point.dy));
      point = canvas.bottomRight - const Offset(2, 2);
      await gesture.moveTo(point);
      await tester.pump();
      checkSource();
      final before = controller.offset;
      await tester.pump(const Duration(milliseconds: 180));
      expect(controller.offset, greaterThan(before));
      checkSource();
      await gesture.up();
      await tester.pump();
      expect(lens, findsNothing);
      gesture = await tester.startGesture(canvas.center);
      await gesture.moveBy(const Offset(30, 0));
      await tester.pump();
      expect(lens, findsOneWidget);
      await gesture.cancel();
      await tester.pump();
      expect(lens, findsNothing);
      gesture = await tester.startGesture(canvas.center);
      await gesture.moveBy(const Offset(30, 0));
      await tester.pump();
      expect(lens, findsOneWidget);
      final second = await tester.startGesture(
        canvas.center + const Offset(0, 40),
      );
      await second.moveBy(const Offset(0, -30));
      await tester.pump();
      expect(lens, findsNothing);
      await second.up();
      await gesture.up();
      await tester.pump();
      gesture = await tester.startGesture(canvas.center);
      await gesture.moveBy(const Offset(30, 0));
      await tester.pump();
      expect(lens, findsOneWidget);
      rebuild(() => enabled = false);
      await tester.pump();
      expect(lens, findsNothing);
      await gesture.up();
      await tester.pumpWidget(const SizedBox.shrink());
    });
  }
}
