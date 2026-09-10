import 'package:anytty_native/src/features/terminal/presentation/terminal_history_scroll_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final physics in <ScrollPhysics>[
    const ClampingScrollPhysics(),
    const BouncingScrollPhysics(),
  ]) {
    for (final fling in [false, true]) {
      testWidgets(
        'prepend preserves the first painted viewport and ongoing ${fling ? 'fling' : 'drag'} ($physics)',
        (tester) async {
          final controller = TerminalHistoryScrollController();
          addTearDown(controller.dispose);
          var prepended = 0;
          final paintedOffsets = <double>[];
          late StateSetter rebuild;
          await tester.pumpWidget(
            MaterialApp(
              home: StatefulBuilder(
                builder: (context, setState) {
                  rebuild = setState;
                  return CustomPaint(
                    foregroundPainter: _OffsetRecorder(
                      controller,
                      paintedOffsets,
                    ),
                    child: ListView.builder(
                      controller: controller,
                      physics: physics,
                      itemExtent: 20,
                      itemCount: 200 + prepended,
                      itemBuilder: (_, index) =>
                          Text('row ${index - prepended}'),
                    ),
                  );
                },
              ),
            ),
          );
          controller.jumpTo(2000);
          await tester.pump();
          final requestOffset = controller.offset;
          final gesture = await tester.startGesture(const Offset(200, 200));
          await gesture.moveBy(const Offset(0, 60));
          await tester.pump(const Duration(milliseconds: 16));
          await gesture.moveBy(const Offset(0, 60));
          await tester.pump(const Duration(milliseconds: 16));
          if (fling) {
            await gesture.up();
            await tester.fling(
              find.byType(ListView),
              const Offset(0, 120),
              1000,
            );
            await tester.pump(const Duration(milliseconds: 16));
          }
          final responseOffset = controller.offset;
          expect(responseOffset, lessThan(requestOffset));
          expect(controller.position.isScrollingNotifier.value, isTrue);

          // The response arrives after the user has moved during the request.
          controller.preservePrepend(512 * 20);
          rebuild(() => prepended += 512);
          if (!fling) await gesture.moveBy(const Offset(0, 20));
          final layoutOffset = controller.offset;
          paintedOffsets.clear();
          await tester.pump();

          // Check the actual first paint, not just the post-frame scroll offset.
          expect(paintedOffsets, isNotEmpty);
          expect(paintedOffsets.first, closeTo(layoutOffset + 512 * 20, 0.01));
          expect(controller.offset, closeTo(layoutOffset + 512 * 20, 0.01));
          expect(controller.position.isScrollingNotifier.value, isTrue);
          final correctedOffset = controller.offset;
          if (fling) {
            await tester.pump(const Duration(milliseconds: 32));
          } else {
            await gesture.moveBy(const Offset(0, 30));
            await tester.pump();
          }
          expect(controller.offset, lessThan(correctedOffset));
          expect(controller.offset, greaterThan(correctedOffset - 200));
          if (!fling) await gesture.up();
          await tester.pumpAndSettle();
          final settledOffset = controller.offset;
          rebuild(() {});
          await tester.pump();
          expect(controller.offset, settledOffset);
        },
      );
    }
  }
}

final class _OffsetRecorder extends CustomPainter {
  _OffsetRecorder(this.controller, this.offsets);

  final ScrollController controller;
  final List<double> offsets;

  @override
  void paint(Canvas canvas, Size size) => offsets.add(controller.offset);

  @override
  bool shouldRepaint(_OffsetRecorder oldDelegate) => true;
}
