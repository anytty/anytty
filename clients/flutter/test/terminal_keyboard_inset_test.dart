import 'package:anytty_native/src/features/terminal/presentation/terminal_keyboard_inset.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('isolates the terminal route from Flutter IME insets', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MediaQuery(
        data: MediaQueryData(viewInsets: EdgeInsets.only(bottom: 300)),
        child: TerminalKeyboardMediaQuery(child: _ViewInsetsProbe()),
      ),
    );

    expect(find.text('0'), findsOneWidget);
  });

  testWidgets('keeps the terminal and key bar in one keyboard workspace', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(400, 800);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);
    final visualInset = ValueNotifier<double>(0);
    addTearDown(visualInset.dispose);
    var workspaceBuilds = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: TerminalKeyboardWorkspace(
          visualInset: visualInset,
          child: Builder(
            builder: (context) {
              workspaceBuilds += 1;
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Expanded(
                    child: Stack(
                      fit: StackFit.expand,
                      children: [
                        ColoredBox(key: Key('terminal'), color: Colors.black),
                        Positioned(
                          left: 8,
                          right: 8,
                          bottom: 8,
                          height: 44,
                          child: SizedBox(key: Key('search-bar')),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(
                    key: Key('key-bar'),
                    width: double.infinity,
                    height: 48,
                  ),
                ],
              );
            },
          ),
        ),
      ),
    );

    expect(tester.getRect(find.byKey(const Key('terminal'))).bottom, 752);
    expect(
      tester.getRect(find.byKey(const Key('key-bar'))),
      const Rect.fromLTWH(0, 752, 400, 48),
    );

    visualInset.value = 300;
    await tester.pump();

    final terminal = tester.getRect(find.byKey(const Key('terminal')));
    final keyBar = tester.getRect(find.byKey(const Key('key-bar')));
    final searchBar = tester.getRect(find.byKey(const Key('search-bar')));
    expect(terminal, const Rect.fromLTWH(0, -300, 400, 752));
    expect(terminal.bottom, 452);
    expect(keyBar, const Rect.fromLTWH(0, 452, 400, 48));
    expect(searchBar, const Rect.fromLTWH(8, 400, 384, 44));
    expect(terminal.bottom, keyBar.top);
    expect(searchBar.bottom, lessThan(keyBar.top));
    expect(workspaceBuilds, 1);
  });

  test('shifts a covered cursor by the minimum bounded distance', () {
    expect(
      resolveTerminalKeyboardShift(
        keyboardInset: 300,
        visibleHeight: 452,
        cursorRow: 35,
        rowHeight: 20,
      ),
      288,
    );
    expect(
      resolveTerminalKeyboardShift(
        keyboardInset: 300,
        visibleHeight: 452,
        cursorRow: 10,
        rowHeight: 20,
      ),
      0,
    );
  });

  testWidgets('freezes live frames only while the keyboard is moving', (
    tester,
  ) async {
    final visualInset = ValueNotifier<double>(0);
    addTearDown(visualInset.dispose);

    Future<void> pumpFrame({
      required double settledInset,
      required String label,
    }) => tester.pumpWidget(
      TerminalKeyboardFrameFreeze(
        visualInset: visualInset,
        settledInset: settledInset,
        child: Text(label, textDirection: TextDirection.ltr),
      ),
    );

    await pumpFrame(settledInset: 0, label: 'before');
    expect(find.text('before'), findsOneWidget);

    visualInset.value = 240;
    await pumpFrame(settledInset: 0, label: 'during');
    expect(find.text('before'), findsOneWidget);
    expect(find.text('during'), findsNothing);

    await pumpFrame(settledInset: 240, label: 'after');
    expect(find.text('after'), findsOneWidget);
  });

  testWidgets('commits one final inset after keyboard opening settles', (
    tester,
  ) async {
    final changes = <double>[];
    final stabilizer = TerminalKeyboardInsetStabilizer(
      onInsetChanged: changes.add,
    );
    addTearDown(stabilizer.dispose);

    stabilizer.update(80);
    await tester.pump(const Duration(milliseconds: 60));
    stabilizer.update(240);
    await tester.pump(const Duration(milliseconds: 119));

    expect(stabilizer.settledInset, 0);
    expect(changes, isEmpty);

    await tester.pump(const Duration(milliseconds: 1));

    expect(stabilizer.settledInset, 240);
    expect(changes, [240]);
  });

  testWidgets('keeps the settled layout until keyboard closing finishes', (
    tester,
  ) async {
    final changes = <double>[];
    final stabilizer = TerminalKeyboardInsetStabilizer(
      onInsetChanged: changes.add,
    );
    addTearDown(stabilizer.dispose);

    stabilizer.update(240);
    await tester.pump(const Duration(milliseconds: 120));
    stabilizer.update(220);

    expect(stabilizer.settledInset, 240);
    expect(changes, [240]);

    stabilizer.update(0);
    await tester.pump(const Duration(milliseconds: 200));
    expect(changes, [240, 0]);
  });
}

final class _ViewInsetsProbe extends StatelessWidget {
  const _ViewInsetsProbe();

  @override
  Widget build(BuildContext context) {
    return Text(
      MediaQuery.viewInsetsOf(context).bottom.toStringAsFixed(0),
      textDirection: TextDirection.ltr,
    );
  }
}
