import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:anytty_native/src/features/terminal/domain/history_interaction.dart';
import 'package:anytty_native/src/features/terminal/domain/history_store.dart';
import 'package:anytty_native/src/features/terminal/domain/live_screen_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_canvas.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('keeps the live frame visible until history is positioned', (
    tester,
  ) async {
    Widget buildPresentation({
      required bool historyLoaded,
      required bool ready,
    }) {
      return MaterialApp(
        home: SizedBox(
          width: 200,
          height: 120,
          child: TerminalHistoryPresentation(
            ready: ready,
            fallback: const ColoredBox(
              key: ValueKey('live-fallback'),
              color: Colors.red,
            ),
            child: historyLoaded
                ? const ColoredBox(
                    key: ValueKey('positioned-history'),
                    color: Colors.green,
                  )
                : const SizedBox.expand(key: ValueKey('history-placeholder')),
          ),
        ),
      );
    }

    await tester.pumpWidget(
      buildPresentation(historyLoaded: false, ready: false),
    );

    expect(find.byKey(const ValueKey('live-fallback')), findsOneWidget);
    final fallbackLayer = tester.element(
      find.byKey(const ValueKey('terminal-history-fallback-layer')),
    );
    final historyLayer = tester.element(
      find.byKey(const ValueKey('terminal-history-content-layer')),
    );

    await tester.pumpWidget(
      buildPresentation(historyLoaded: true, ready: false),
    );

    expect(
      tester.element(
        find.byKey(const ValueKey('terminal-history-fallback-layer')),
      ),
      same(fallbackLayer),
    );
    expect(
      tester.element(
        find.byKey(const ValueKey('terminal-history-content-layer')),
      ),
      same(historyLayer),
    );
    final hiddenHistory = tester.widget<Opacity>(
      find.ancestor(
        of: find.byKey(const ValueKey('positioned-history')),
        matching: find.byType(Opacity),
      ),
    );
    expect(hiddenHistory.opacity, 0);

    await tester.pumpWidget(
      buildPresentation(historyLoaded: true, ready: true),
    );

    expect(find.byKey(const ValueKey('live-fallback')), findsNothing);
    expect(
      tester.element(
        find.byKey(const ValueKey('terminal-history-content-layer')),
      ),
      same(historyLayer),
    );
    final visibleHistory = tester.widget<Opacity>(
      find.ancestor(
        of: find.byKey(const ValueKey('positioned-history')),
        matching: find.byType(Opacity),
      ),
    );
    expect(visibleHistory.opacity, 1);
  });

  test('keeps terminal semantics bounded on complete graphemes', () {
    const family = '\u{1F468}\u200D\u{1F469}\u200D\u{1F467}\u200D\u{1F466}';
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 80,
      rows: 40,
      screenRows: [
        for (var index = 0; index < 40; index += 1)
          ScreenRow(
            cells: [ScreenCell(content: 'line $index $family', width: 16)],
          ),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    final value = terminalScreenSemanticValue(
      screen,
      maxLines: 3,
      maxCharacters: 96,
    );

    expect(value, startsWith('Earlier visible output omitted.'));
    expect(value, contains('line 39 $family'));
    expect(value, isNot(contains('line 0')));
    expect(value.characters.length, lessThanOrEqualTo(96));
  });

  test('keeps the latest output when the semantic budget is very small', () {
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 20,
      rows: 2,
      screenRows: [
        ScreenRow(cells: [ScreenCell(content: 'older', width: 5)]),
        ScreenRow(cells: [ScreenCell(content: 'latest', width: 6)]),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    final value = terminalScreenSemanticValue(
      screen,
      maxLines: 1,
      maxCharacters: 4,
    );

    expect(value, 'test');
    expect(value.characters.length, 4);
  });

  testWidgets('exposes the current visible terminal to accessibility', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 20,
      rows: 2,
      screenRows: [
        ScreenRow(
          cells: [
            ScreenCell(content: 'prompt', width: 6),
            ScreenCell(width: 2),
            ScreenCell(content: 'ready', width: 5),
          ],
        ),
        ScreenRow(cells: [ScreenCell(content: 'latest output', width: 13)]),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox(
          width: 200,
          height: 40,
          child: TerminalCanvas(screen: screen),
        ),
      ),
    );

    final node = tester.getSemantics(
      find.byKey(const ValueKey('terminal-output-semantics')),
    );
    expect(node.label, 'Terminal output, 20 columns by 2 rows');
    expect(node.value, 'prompt  ready\nlatest output');
    expect(node.childrenCount, 0);
    semantics.dispose();
  });

  testWidgets('only creates a horizontal scroller when the canvas overflows', (
    tester,
  ) async {
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 20,
      rows: 1,
      screenRows: [
        ScreenRow(cells: [ScreenCell(content: 'ready', width: 5)]),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    Future<void> pumpAtWidth(double width) => tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: width,
            height: terminalRowHeight,
            child: TerminalCanvas(screen: screen),
          ),
        ),
      ),
    );

    await pumpAtWidth(200);
    expect(find.byType(SingleChildScrollView), findsNothing);

    await pumpAtWidth(100);
    expect(find.byType(SingleChildScrollView), findsOneWidget);
  });

  testWidgets('acknowledges a replacement snapshot with the same revision', (
    tester,
  ) async {
    CanonicalLiveScreen screen(String content) => CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64(7),
      cols: 12,
      rows: 1,
      screenRows: [
        ScreenRow(
          cells: [ScreenCell(content: content, width: content.length)],
        ),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    var current = screen('first');
    late StateSetter update;
    final presented = <int>[];
    await tester.pumpWidget(
      MaterialApp(
        home: StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return SizedBox(
              width: 120,
              height: 20,
              child: TerminalCanvas(
                screen: current,
                onPresented: presented.add,
              ),
            );
          },
        ),
      ),
    );
    await tester.pump();
    expect(presented, [7]);

    update(() => current = screen('second'));
    await tester.pump();
    await tester.pump();

    expect(presented, [7, 7]);
  });

  testWidgets('paints styled snapshot cells and cursor into nonblank pixels', (
    tester,
  ) async {
    const boundaryKey = ValueKey('terminal-boundary');
    int? presentedRevision;
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64(7),
      cols: 32,
      rows: 3,
      screenRows: [
        ScreenRow(
          cells: [
            ScreenCell(
              content: r'lozzow@studio:~ $',
              width: 18,
              style: CellStyle(foreground: 'ansi:6', bold: true),
            ),
          ],
        ),
        ScreenRow(
          cells: [
            ScreenCell(
              content: 'AnyTTY',
              width: 6,
              style: CellStyle(foreground: '#fafafa', background: '#a84f16'),
            ),
            ScreenCell(content: ' snapshot', width: 9),
          ],
        ),
        ScreenRow(
          cells: [
            ScreenCell(content: 'wide:', width: 5),
            ScreenCell(content: '\u754c', width: 2),
          ],
        ),
      ],
      alternateScreen: false,
      cursor: TerminalCursor(
        row: 0,
        col: 19,
        visible: true,
        shape: CursorShape.CURSOR_SHAPE_BAR,
      ),
      modes: TerminalModes(autoWrap: true),
      timestampUnixNano: Int64.ZERO,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: RepaintBoundary(
            key: boundaryKey,
            child: SizedBox(
              width: 300,
              height: 60,
              child: TerminalCanvas(
                screen: screen,
                onPresented: (revision) => presentedRevision = revision,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();
    expect(tester.takeException(), isNull);
    expect(presentedRevision, 7);

    final boundary = tester.renderObject<RenderRepaintBoundary>(
      find.byKey(boundaryKey),
    );
    final data = await tester.runAsync(() async {
      final image = await boundary.toImage(pixelRatio: 1);
      return image.toByteData(format: ui.ImageByteFormat.rawRgba);
    });
    expect(data, isNotNull);
    final bytes = data!.buffer.asUint8List();
    var nonBackground = 0;
    var warmPixels = 0;
    for (var index = 0; index < bytes.length; index += 4) {
      final red = bytes[index];
      final green = bytes[index + 1];
      final blue = bytes[index + 2];
      if (red > 18 || green > 18 || blue > 20) nonBackground += 1;
      if (red > 100 && red > green * 1.3 && green > blue) warmPixels += 1;
    }
    expect(nonBackground, greaterThan(500));
    expect(warmPixels, greaterThan(100));
  });

  testWidgets('paints frozen history rows without ANSI reconstruction', (
    tester,
  ) async {
    const boundaryKey = ValueKey('history-boundary');
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      HistoryRow(
        logicalLineId: Int64(10),
        row: ScreenRow(
          cells: [
            ScreenCell(
              content: 'older output',
              width: 12,
              style: CellStyle(foreground: 'ansi:11'),
            ),
          ],
        ),
      ),
      HistoryRow(
        logicalLineId: Int64(20),
        row: ScreenRow(
          cells: [ScreenCell(content: 'latest output', width: 13)],
        ),
      ),
    ];
    final history = FrozenHistory(
      token: 'token-1',
      generation: Int64.ONE,
      cols: 24,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64.ZERO,
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: rows.length,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: RepaintBoundary(
            key: boundaryKey,
            child: SizedBox(
              width: 240,
              height: 60,
              child: TerminalHistoryCanvas(
                rows: rows,
                cols: 24,
                scrollController: controller,
                layout: HistoryLayout.fromHistory(history),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();
    expect(
      find.byKey(const ValueKey('terminal-history-canvas')),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);

    final boundary = tester.renderObject<RenderRepaintBoundary>(
      find.byKey(boundaryKey),
    );
    final data = await tester.runAsync(() async {
      final image = await boundary.toImage(pixelRatio: 1);
      return image.toByteData(format: ui.ImageByteFormat.rawRgba);
    });
    expect(data, isNotNull);
    final bytes = data!.buffer.asUint8List();
    var brightPixels = 0;
    for (var index = 0; index < bytes.length; index += 4) {
      if (bytes[index] > 80 || bytes[index + 1] > 80 || bytes[index + 2] > 80) {
        brightPixels += 1;
      }
    }
    expect(brightPixels, greaterThan(100));
  });

  testWidgets('exposes visible history rows and their selected state', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      HistoryRow(
        logicalLineId: Int64(10),
        row: ScreenRow(cells: [ScreenCell(content: 'older output', width: 12)]),
      ),
      HistoryRow(
        logicalLineId: Int64(20),
        row: ScreenRow(
          cells: [ScreenCell(content: 'latest output', width: 13)],
        ),
      ),
    ];
    final history = FrozenHistory(
      token: 'token-1',
      generation: Int64.ONE,
      cols: 24,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64.ZERO,
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: rows.length,
    );
    final layout = HistoryLayout.fromHistory(history);

    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox(
          width: 240,
          height: 60,
          child: TerminalHistoryCanvas(
            rows: rows,
            cols: 24,
            scrollController: controller,
            layout: layout,
            selection: HistorySelection(
              anchor: HistoryCellPoint(lineId: Int64(20), column: 0),
              focus: HistoryCellPoint(lineId: Int64(20), column: 13),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final older = tester.getSemantics(
      find.byKey(const ValueKey('terminal-history-line-0')),
    );
    final latest = tester.getSemantics(
      find.byKey(const ValueKey('terminal-history-line-1')),
    );
    expect(older.label, 'History line 1');
    expect(older.value, 'older output');
    expect(older.flagsCollection.isSelected, ui.Tristate.isFalse);
    expect(latest.label, 'History line 2');
    expect(latest.value, 'latest output');
    expect(latest.flagsCollection.isSelected, ui.Tristate.isTrue);
    semantics.dispose();
  });

  testWidgets('keeps fixed-grid history horizontally inspectable', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      HistoryRow(
        logicalLineId: Int64(10),
        fixedGrid: true,
        screenCols: 12,
        row: ScreenRow(cells: [ScreenCell(content: 'abcdefghijkl', width: 12)]),
      ),
    ];
    final history = FrozenHistory(
      token: 'token-fixed-grid',
      generation: Int64.ONE,
      cols: 4,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64(10),
        cellOffset: 0,
        atEnd: false,
      ),
      hasMore: false,
      logicalTotal: 1,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 4 * terminalCellWidth,
            height: terminalRowHeight,
            child: TerminalHistoryCanvas(
              rows: rows,
              cols: 4,
              scrollController: controller,
              layout: HistoryLayout.fromHistory(history),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final horizontal = tester.widget<SingleChildScrollView>(
      find.byType(SingleChildScrollView),
    );
    expect(horizontal.controller?.position.maxScrollExtent, greaterThan(0));
  });

  testWidgets('virtual history tail lets the frozen anchor reach the top', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      HistoryRow(
        logicalLineId: Int64(10),
        row: ScreenRow(cells: [ScreenCell(content: 'older', width: 5)]),
      ),
    ];
    final history = FrozenHistory(
      token: 'token-tail',
      generation: Int64.ONE,
      cols: 12,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64.ZERO,
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: 1,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 12 * terminalCellWidth,
            height: 4 * terminalRowHeight,
            child: TerminalHistoryCanvas(
              rows: rows,
              cols: 12,
              scrollController: controller,
              layout: HistoryLayout.fromHistory(history),
              trailingRows: historyViewportTailRows(
                history: history,
                viewportRows: 4,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(controller.position.maxScrollExtent, terminalRowHeight);
  });

  testWidgets('maps a live canvas tap to the canonical link cell', (
    tester,
  ) async {
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 12,
      rows: 1,
      screenRows: [
        ScreenRow(
          cells: [
            ScreenCell(content: 'ab', width: 2),
            ScreenCell(
              content: 'docs',
              width: 4,
              linkUrl: 'https://example.com/docs',
            ),
          ],
        ),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );
    String? opened;
    var blankTaps = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 120,
            height: terminalRowHeight,
            child: TerminalCanvas(
              screen: screen,
              onLinkTap: (link) => opened = link.url,
              onBlankTap: () => blankTaps += 1,
            ),
          ),
        ),
      ),
    );

    final origin = tester.getTopLeft(find.byType(TerminalCanvas));
    await tester.tapAt(origin + const Offset(terminalCellWidth * 3, 10));
    expect(opened, 'https://example.com/docs');
    await tester.tapAt(origin + const Offset(terminalCellWidth * 8, 10));
    expect(blankTaps, 1);
  });

  testWidgets('treats file paths as plain terminal text', (tester) async {
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'shell'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 24,
      rows: 1,
      screenRows: [
        ScreenRow(
          cells: [ScreenCell(content: 'see src/main.dart:12', width: 20)],
        ),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );
    var terminalTaps = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 24 * terminalCellWidth,
            height: terminalRowHeight,
            child: TerminalCanvas(
              screen: screen,
              onTerminalTap: (_) => terminalTaps += 1,
            ),
          ),
        ),
      ),
    );

    final origin = tester.getTopLeft(find.byType(TerminalCanvas));
    await tester.tapAt(origin + const Offset(terminalCellWidth * 8, 10));

    expect(terminalTaps, 1);
  });

  testWidgets('routes a plain short tap with terminal-local coordinates', (
    tester,
  ) async {
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'editor'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 12,
      rows: 2,
      screenRows: [
        ScreenRow(cells: [ScreenCell(content: 'tree', width: 4)]),
        ScreenRow(cells: [ScreenCell(content: 'file', width: 4)]),
      ],
      alternateScreen: true,
      cursor: null,
      modes: TerminalModes(mouseTracking: true, mouseNormal: true),
      timestampUnixNano: Int64.ZERO,
    );
    Offset? terminalTap;
    var blankTaps = 0;
    var starts = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 120,
            height: 40,
            child: TerminalCanvas(
              screen: screen,
              onTerminalTap: (position) => terminalTap = position,
              onBlankTap: () => blankTaps += 1,
              onInteractionStart: () => starts += 1,
            ),
          ),
        ),
      ),
    );

    final origin = tester.getTopLeft(find.byType(TerminalCanvas));
    await tester.tapAt(origin + const Offset(25, 30));

    expect(starts, 1);
    expect(terminalTap?.dx, closeTo(25, 0.1));
    expect(terminalTap?.dy, closeTo(30, 0.1));
    expect(blankTaps, 0);
  });

  testWidgets('does not turn a long touch or cancelled drag into a tap', (
    tester,
  ) async {
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'editor'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 12,
      rows: 4,
      screenRows: [
        for (var index = 0; index < 4; index += 1)
          ScreenRow(cells: [ScreenCell(content: 'row$index', width: 4)]),
      ],
      alternateScreen: true,
      cursor: null,
      modes: TerminalModes(mouseTracking: true, mouseNormal: true),
      timestampUnixNano: Int64.ZERO,
    );
    var taps = 0;
    var cancellations = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 120,
            height: 80,
            child: TerminalCanvas(
              screen: screen,
              onTerminalTap: (_) => taps += 1,
              onInteractionCancel: () => cancellations += 1,
              onVerticalDragCancel: () => cancellations += 1,
            ),
          ),
        ),
      ),
    );

    final origin = tester.getTopLeft(find.byType(TerminalCanvas));
    final longTouch = await tester.startGesture(origin + const Offset(20, 20));
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await longTouch.up();
    await tester.pump();
    expect(taps, 0);

    final cancelled = await tester.startGesture(origin + const Offset(20, 60));
    await cancelled.moveBy(const Offset(0, -30));
    await tester.pump();
    await cancelled.cancel();
    await tester.pump();
    expect(taps, 0);
    expect(cancellations, greaterThanOrEqualTo(1));
  });

  test('fills terminal block elements to the exact cell edges', () {
    const bounds = Rect.fromLTWH(10, 20, 8, 16);

    expect(terminalBlockElementRects(0x2588, bounds), [bounds]);
    expect(terminalBlockElementRects(0x2584, bounds), [
      const Rect.fromLTWH(10, 28, 8, 8),
    ]);
    expect(terminalBlockElementRects(0x258c, bounds), [
      const Rect.fromLTWH(10, 20, 4, 16),
    ]);
    expect(terminalBlockElementRects(0x259a, bounds), [
      const Rect.fromLTWH(10, 20, 4, 8),
      const Rect.fromLTWH(14, 28, 4, 8),
    ]);
    expect(terminalBlockElementRects('A'.codeUnitAt(0), bounds), isNull);
    expect(terminalFontFamilyFallback.first, 'JetBrainsMonoNerd');
    expect(terminalGraphemeCellWidth('A'), 1);
    expect(terminalGraphemeCellWidth('\u754c'), 2);
    expect(terminalGraphemeCellWidth('e\u0301'), 1);
    expect(terminalGraphemeCellWidth('\u{1f4bb}'), 2);
    expect(terminalShadeOpacity(0x2591), 0.25);
    expect(terminalShadeOpacity(0x2592), 0.5);
    expect(terminalShadeOpacity(0x2593), 0.75);
  });

  test('realigns mixed fallback-font text to terminal columns', () {
    expect(terminalGridTextRuns('\u540c\u6b65\u7f29jjj', 9), [
      (text: '\u540c', column: 0, width: 2),
      (text: '\u6b65', column: 2, width: 2),
      (text: '\u7f29', column: 4, width: 2),
      (text: 'jjj', column: 6, width: 3),
    ]);
  });

  test('matches the cursor span to wide terminal glyphs', () {
    final row = ScreenRow(
      cells: [
        ScreenCell(content: 'a\u4e2db', width: 4),
        ScreenCell(width: 2),
      ],
    );

    expect(terminalCursorCellSpan(row, 0), (column: 0, width: 1));
    expect(terminalCursorCellSpan(row, 1), (column: 1, width: 2));
    expect(terminalCursorCellSpan(row, 2), (column: 1, width: 2));
    expect(terminalCursorCellSpan(row, 3), (column: 3, width: 1));
    expect(terminalCursorCellSpan(row, 4), (column: 4, width: 1));
    expect(terminalCursorCellSpan(row, 8), (column: 8, width: 1));
  });

  testWidgets('fills grid glyphs embedded inside a coalesced screen run', (
    tester,
  ) async {
    const boundaryKey = ValueKey('block-run-boundary');
    final screen = CanonicalLiveScreen(
      terminal: TerminalRef(endpointId: 'studio', terminalId: 'blocks'),
      connectionGeneration: Int64.ONE,
      revision: Int64.ONE,
      cols: 4,
      rows: 2,
      screenRows: [
        ScreenRow(cells: [ScreenCell(content: ' █│', width: 3)]),
        ScreenRow(cells: [ScreenCell(content: ' █│', width: 3)]),
      ],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: RepaintBoundary(
            key: boundaryKey,
            child: SizedBox(
              width: terminalCellWidth * 4,
              height: terminalRowHeight * 2,
              child: TerminalCanvas(screen: screen),
            ),
          ),
        ),
      ),
    );
    await tester.pump();
    final boundary = tester.renderObject<RenderRepaintBoundary>(
      find.byKey(boundaryKey),
    );
    final data = await tester.runAsync(() async {
      final image = await boundary.toImage(pixelRatio: 1);
      return image.toByteData(format: ui.ImageByteFormat.rawRgba);
    });
    final bytes = data!.buffer.asUint8List();
    final width = (terminalCellWidth * 4).ceil();

    int luminanceAt(int x, int y) {
      final offset = (y * width + x) * 4;
      return bytes[offset] + bytes[offset + 1] + bytes[offset + 2];
    }

    int maxLuminanceNear(double x, int y) {
      final center = x.round();
      return [
        for (var delta = -2; delta <= 2; delta += 1)
          luminanceAt((center + delta).clamp(0, width - 1), y),
      ].reduce(math.max);
    }

    final blockX = (terminalCellWidth * 1.5).floor();
    final lineCenterX = terminalCellWidth * 2.5;
    for (final y in [0, 1, 18, 19, 20, 21, 38, 39]) {
      expect(
        luminanceAt(blockX, y),
        greaterThan(300),
        reason: 'block row pixel $y',
      );
      expect(
        maxLuminanceNear(lineCenterX, y),
        greaterThan(300),
        reason: 'line row pixel $y',
      );
    }
  });

  testWidgets('maps history selection drags to logical display columns', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      HistoryRow(
        logicalLineId: Int64(41),
        row: ScreenRow(cells: [ScreenCell(content: 'abcdefghij', width: 10)]),
      ),
    ];
    final history = FrozenHistory(
      token: 'token-1',
      generation: Int64.ONE,
      cols: 12,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64(41),
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: 1,
    );
    final layout = HistoryLayout.fromHistory(history);
    HistorySelection? selection;

    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox(
          width: 120,
          height: 60,
          child: TerminalHistoryCanvas(
            rows: rows,
            cols: 12,
            scrollController: controller,
            layout: layout,
            selectionEnabled: true,
            onSelectionChanged: (value) => selection = value,
          ),
        ),
      ),
    );
    await tester.pump();
    final origin = tester.getTopLeft(
      find.byKey(const ValueKey('terminal-history-canvas')),
    );
    await tester.dragFrom(origin + const Offset(9, 10), const Offset(42, 0));

    expect(selection, isNotNull);
    final range = normalizeHistorySelection(
      layout: layout,
      selection: selection!,
    );
    expect(range.startLineId, Int64(41));
    expect(range.endLineId, Int64(41));
    expect(range.startCol, greaterThanOrEqualTo(1));
    expect(range.endCol, greaterThan(range.startCol));
  });

  testWidgets('updates history highlights without rebuilding its parent', (
    tester,
  ) async {
    final controller = ScrollController();
    final highlights = TerminalHistoryHighlights();
    addTearDown(controller.dispose);
    addTearDown(highlights.dispose);
    final rows = [
      HistoryRow(
        logicalLineId: Int64(41),
        row: ScreenRow(cells: [ScreenCell(content: 'alpha beta', width: 10)]),
      ),
    ];
    final history = FrozenHistory(
      token: 'highlight-token',
      generation: Int64.ONE,
      cols: 12,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64(41),
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: 1,
    );
    final layout = HistoryLayout.fromHistory(history);
    var parentBuilds = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) {
            parentBuilds += 1;
            return SizedBox(
              width: 120,
              height: 40,
              child: TerminalHistoryCanvas(
                rows: rows,
                cols: 12,
                scrollController: controller,
                layout: layout,
                highlights: highlights,
              ),
            );
          },
        ),
      ),
    );
    expect(parentBuilds, 1);

    highlights.update(
      selection: HistorySelection(
        anchor: HistoryCellPoint(lineId: Int64(41), column: 0),
        focus: HistoryCellPoint(lineId: Int64(41), column: 5),
      ),
      layout: layout,
      searchMatch: HistoryRange(
        startLineId: Int64(41),
        startCol: 6,
        endLineId: Int64(41),
        endCol: 10,
      ),
    );
    await tester.pump();

    expect(parentBuilds, 1);
    final row = tester.getSemantics(
      find.byKey(const ValueKey('terminal-history-line-0')),
    );
    expect(row.flagsCollection.isSelected, ui.Tristate.isTrue);
  });

  testWidgets(
    'keeps the first selection anchor and extends it on the next tap',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);
      final rows = [
        HistoryRow(
          logicalLineId: Int64(51),
          row: ScreenRow(
            cells: [ScreenCell(content: 'abcdefghijkl', width: 12)],
          ),
        ),
      ];
      final history = FrozenHistory(
        token: 'token-2',
        generation: Int64.ONE,
        cols: 12,
        rows: rows,
        anchor: HistoryAnchor(
          logicalLineId: Int64(51),
          cellOffset: 0,
          atEnd: true,
        ),
        hasMore: false,
        logicalTotal: 1,
      );
      final layout = HistoryLayout.fromHistory(history);
      HistorySelection? selection;

      await tester.pumpWidget(
        MaterialApp(
          home: StatefulBuilder(
            builder: (context, setState) => SizedBox(
              width: 120,
              height: 40,
              child: TerminalHistoryCanvas(
                rows: rows,
                cols: 12,
                scrollController: controller,
                layout: layout,
                selection: selection,
                selectionEnabled: true,
                onSelectionChanged: (value) =>
                    setState(() => selection = value),
              ),
            ),
          ),
        ),
      );
      final origin = tester.getTopLeft(
        find.byKey(const ValueKey('terminal-history-canvas')),
      );
      await tester.tapAt(origin + const Offset(10, 10));
      await tester.pump();
      final anchor = selection!.anchor;
      await tester.tapAt(origin + const Offset(60, 10));
      await tester.pump();

      expect(selection!.anchor, anchor);
      final range = normalizeHistorySelection(
        layout: layout,
        selection: selection!,
      );
      expect(range.startCol, 1);
      expect(range.endCol, greaterThanOrEqualTo(7));
    },
  );

  testWidgets('uses two fingers to scroll without corrupting selection', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      for (var index = 0; index < 40; index += 1)
        HistoryRow(
          logicalLineId: Int64(100 + index),
          row: ScreenRow(cells: [ScreenCell(content: 'line $index', width: 7)]),
        ),
    ];
    final history = FrozenHistory(
      token: 'token-3',
      generation: Int64.ONE,
      cols: 12,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64(139),
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: 40,
    );
    final layout = HistoryLayout.fromHistory(history);
    final selection = HistorySelection(
      anchor: historyPointForRowColumn(
        layout: layout,
        rowIndex: 0,
        localColumn: 0,
      ),
      focus: historyPointForRowColumn(
        layout: layout,
        rowIndex: 0,
        localColumn: 4,
      ),
    );
    final changes = <HistorySelection>[];

    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox(
          width: 120,
          height: 80,
          child: TerminalHistoryCanvas(
            rows: rows,
            cols: 12,
            scrollController: controller,
            layout: layout,
            selection: selection,
            selectionEnabled: true,
            onSelectionChanged: changes.add,
          ),
        ),
      ),
    );
    await tester.pump();
    final origin = tester.getTopLeft(
      find.byKey(const ValueKey('terminal-history-canvas')),
    );
    final first = await tester.startGesture(origin + const Offset(30, 65));
    final second = await tester.startGesture(origin + const Offset(80, 65));
    await tester.pump();
    await first.moveBy(const Offset(0, -40));
    await second.moveBy(const Offset(0, -40));
    await tester.pump();
    await first.up();
    await second.up();
    await tester.pump();

    expect(controller.offset, greaterThan(0));
    expect(changes, isEmpty);
  });

  testWidgets('keeps auto-scrolling while selection is held at an edge', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      for (var index = 0; index < 40; index += 1)
        HistoryRow(
          logicalLineId: Int64(200 + index),
          row: ScreenRow(cells: [ScreenCell(content: 'line $index', width: 7)]),
        ),
    ];
    final history = FrozenHistory(
      token: 'token-4',
      generation: Int64.ONE,
      cols: 12,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64(239),
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: false,
      logicalTotal: 40,
    );
    final changes = <HistorySelection>[];

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 120,
            height: 80,
            child: TerminalHistoryCanvas(
              rows: rows,
              cols: 12,
              scrollController: controller,
              layout: HistoryLayout.fromHistory(history),
              selectionEnabled: true,
              onSelectionChanged: changes.add,
            ),
          ),
        ),
      ),
    );
    await tester.pump();
    final origin = tester.getTopLeft(
      find.byKey(const ValueKey('terminal-history-canvas')),
    );
    final gesture = await tester.startGesture(origin + const Offset(30, 20));
    await gesture.moveBy(const Offset(0, 30));
    await tester.pump();
    await gesture.moveBy(const Offset(0, 20));
    await tester.pump();
    await gesture.moveBy(const Offset(0, 5));
    await tester.pump();
    final initialOffset = controller.offset;
    expect(changes, isNotEmpty, reason: 'the drag must enter selection mode');
    expect(
      changes.length,
      greaterThan(1),
      reason: 'the accepted drag must continue updating the selection',
    );
    expect(controller.position.maxScrollExtent, greaterThan(0));

    await tester.pump(const Duration(milliseconds: 240));

    expect(
      controller.offset,
      greaterThan(initialOffset),
      reason: 'selection should keep scrolling after the drag stops moving',
    );
    expect(changes.length, greaterThan(1));

    await gesture.up();
    await tester.pump();
    final releasedOffset = controller.offset;
    await tester.pump(const Duration(milliseconds: 240));
    expect(controller.offset, releasedOffset);
  });

  testWidgets('extends selection into earlier rows at the top edge', (
    tester,
  ) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final rows = [
      for (var index = 0; index < 60; index += 1)
        HistoryRow(
          logicalLineId: Int64(300 + index),
          row: ScreenRow(cells: [ScreenCell(content: 'line $index', width: 7)]),
        ),
    ];
    final history = FrozenHistory(
      token: 'token-5',
      generation: Int64.ONE,
      cols: 12,
      rows: rows,
      anchor: HistoryAnchor(
        logicalLineId: Int64(359),
        cellOffset: 0,
        atEnd: true,
      ),
      hasMore: true,
      logicalTotal: 120,
    );
    final layout = HistoryLayout.fromHistory(history);
    HistorySelection? selection;

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: StatefulBuilder(
            builder: (context, setState) => SizedBox(
              width: 120,
              height: 80,
              child: TerminalHistoryCanvas(
                rows: rows,
                cols: 12,
                scrollController: controller,
                layout: layout,
                selection: selection,
                selectionEnabled: true,
                canLoadOlder: true,
                onSelectionChanged: (value) =>
                    setState(() => selection = value),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();
    controller.jumpTo(20 * defaultTerminalSettings.metrics.rowHeight);
    await tester.pump();
    final initialOffset = controller.offset;
    final origin = tester.getTopLeft(
      find.byKey(const ValueKey('terminal-history-canvas')),
    );
    final gesture = await tester.startGesture(origin + const Offset(30, 65));
    await gesture.moveTo(origin + const Offset(30, 2));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 240));

    expect(controller.offset, lessThan(initialOffset));
    expect(selection, isNotNull);
    final range = normalizeHistorySelection(
      layout: layout,
      selection: selection!,
    );
    expect(range.startLineId, lessThan(range.endLineId));

    await gesture.up();
  });
}
