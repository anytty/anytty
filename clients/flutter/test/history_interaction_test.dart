import 'package:anytty_native/src/features/terminal/domain/history_interaction.dart';
import 'package:anytty_native/src/features/terminal/domain/history_store.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'expands history prefetch lead and request size for fast upward scroll',
    () {
      final slow = adaptiveHistoryPrefetchPlan(
        baseThresholdRows: 30,
        upwardVelocityRowsPerSecond: 0,
      );
      final fast = adaptiveHistoryPrefetchPlan(
        baseThresholdRows: 30,
        upwardVelocityRowsPerSecond: 240,
      );

      expect(slow, (thresholdRows: 542, requestRows: 512));
      expect(fast.thresholdRows, greaterThan(slow.thresholdRows));
      expect(fast.requestRows, 512);
      expect(fast.thresholdRows, 1142);
    },
  );

  test('bounds invalid history prefetch velocity', () {
    expect(
      adaptiveHistoryPrefetchPlan(
        baseThresholdRows: 30,
        upwardVelocityRowsPerSecond: double.infinity,
      ),
      (thresholdRows: 542, requestRows: 512),
    );
  });

  test('caps the fast-scroll history buffer at three API pages', () {
    expect(
      adaptiveHistoryPrefetchPlan(
        baseThresholdRows: 30,
        upwardVelocityRowsPerSecond: 1000,
      ),
      (thresholdRows: 1536, requestRows: 512),
    );
  });

  test('keeps an already visible search result in the current viewport', () {
    expect(
      historyRowIsVisibleInViewport(
        rowIndex: 12,
        rowHeight: 18,
        scrollOffset: 180,
        viewportExtent: 108,
      ),
      isTrue,
    );
    expect(
      historyRowIsVisibleInViewport(
        rowIndex: 16,
        rowHeight: 18,
        scrollOffset: 180,
        viewportExtent: 108,
      ),
      isFalse,
    );
    expect(
      historyRowIsVisibleInViewport(
        rowIndex: 9,
        rowHeight: 18,
        scrollOffset: 180,
        viewportExtent: 108,
      ),
      isFalse,
    );
  });

  test('moves through cached search matches and wraps locally', () {
    final first = HistoryRange(
      startLineId: Int64(41),
      startCol: 1,
      endLineId: Int64(41),
      endCol: 4,
    );
    final second = HistoryRange(
      startLineId: Int64(42),
      startCol: 2,
      endLineId: Int64(42),
      endCol: 5,
    );

    expect(
      adjacentHistorySearchMatch(
        matches: [first, second],
        current: first.deepCopy(),
        forward: true,
      ),
      (match: second, wrapped: false),
    );
    expect(
      adjacentHistorySearchMatch(
        matches: [first, second],
        current: first,
        forward: false,
      ),
      (match: second, wrapped: true),
    );
  });

  test('maps wrapped rows to exact logical display columns', () {
    final history = _history([
      _row(41, 0, [('a', 1), ('b', 1), ('c', 1), ('d', 1)]),
      _row(41, 1, [('中', 2), ('e', 1)]),
    ], cols: 5);

    final layout = HistoryLayout.fromHistory(history);

    expect(layout.rows[0].logicalStartColumn, 0);
    expect(layout.rows[1].logicalStartColumn, 4);
    expect(
      historyPointForRowColumn(
        layout: layout,
        rowIndex: 1,
        localColumn: 2,
      ).column,
      6,
    );
  });

  test('normalizes reversed selection and clamps it to loaded line widths', () {
    final layout = HistoryLayout.fromHistory(
      _history([_textRow(41, 'alpha'), _textRow(42, 'beta')]),
    );

    final range = normalizeHistorySelection(
      layout: layout,
      selection: HistorySelection(
        anchor: HistoryCellPoint(lineId: Int64(42), column: 99),
        focus: HistoryCellPoint(lineId: Int64(41), column: -1),
      ),
    );

    expect(range.startLineId, Int64(41));
    expect(range.startCol, 0);
    expect(range.endLineId, Int64(42));
    expect(range.endCol, 4);
  });

  test('projects a logical selection across wrapped rows', () {
    final layout = HistoryLayout.fromHistory(
      _history([
        _textRow(41, 'abcd', rowInLine: 0),
        _textRow(41, 'efg', rowInLine: 1),
        _textRow(42, 'tail'),
      ], cols: 5),
    );
    final range = HistoryRange(
      startLineId: Int64(41),
      startCol: 2,
      endLineId: Int64(42),
      endCol: 2,
    );

    expect(
      projectHistoryRangeToRow(layout: layout, rowIndex: 0, range: range),
      (start: 2, end: 4),
    );
    expect(
      projectHistoryRangeToRow(layout: layout, rowIndex: 1, range: range),
      (start: 0, end: 3),
    );
    expect(
      projectHistoryRangeToRow(layout: layout, rowIndex: 2, range: range),
      (start: 0, end: 2),
    );
  });

  test('indexes a match directly to its wrapped visual row', () {
    final layout = HistoryLayout.fromHistory(
      _history([
        _textRow(41, 'abcd', rowInLine: 0),
        _textRow(41, 'efgh', rowInLine: 1),
        _textRow(42, 'tail'),
      ], cols: 4),
    );
    final match = HistoryRange(
      startLineId: Int64(41),
      startCol: 5,
      endLineId: Int64(41),
      endCol: 7,
    );

    expect(historyRowIndexForRange(layout: layout, range: match), 1);
  });

  test('groups search matches by visual row in display order', () {
    final layout = HistoryLayout.fromHistory(
      _history([
        _textRow(41, 'abcd', rowInLine: 0),
        _textRow(41, 'efgh', rowInLine: 1),
        _textRow(42, 'tail'),
      ], cols: 4),
    );
    final ranges = [
      HistoryRange(
        startLineId: Int64(42),
        startCol: 1,
        endLineId: Int64(42),
        endCol: 3,
      ),
      HistoryRange(
        startLineId: Int64(41),
        startCol: 6,
        endLineId: Int64(41),
        endCol: 8,
      ),
      HistoryRange(
        startLineId: Int64(41),
        startCol: 4,
        endLineId: Int64(41),
        endCol: 5,
      ),
    ];

    final indexed = projectHistoryRangesToRows(layout: layout, ranges: ranges);

    expect(indexed.keys, unorderedEquals([1, 2]));
    expect(indexed[1], [(start: 0, end: 1), (start: 2, end: 4)]);
    expect(indexed[2], [(start: 1, end: 3)]);
  });

  test('maps an exact viewport cell boundary to the next visual row', () {
    final history = _history(
      [
        _textRow(41, 'older'),
        _textRow(42, 'abcdef', rowInLine: 0),
        _textRow(42, 'ghijkl', rowInLine: 1),
        _textRow(42, 'mnopqr', rowInLine: 2),
      ],
      cols: 6,
      anchor: HistoryAnchor(
        logicalLineId: Int64(42),
        cellOffset: 6,
        atEnd: false,
      ),
    );

    expect(historyViewportAnchorRow(history), 2);
  });

  test('keeps a viewport anchor inside the row containing a wide glyph', () {
    final history = _history(
      [
        _row(42, 0, [('a', 1), ('\u4e2d', 2)]),
        _row(42, 1, [('b', 1)]),
      ],
      cols: 3,
      anchor: HistoryAnchor(
        logicalLineId: Int64(42),
        cellOffset: 2,
        atEnd: false,
      ),
    );

    expect(historyViewportAnchorRow(history), 0);
  });

  test(
    'adds only enough virtual tail rows to keep the frozen anchor at top',
    () {
      final history = _history(
        [_textRow(41, 'older'), _textRow(42, 'current')],
        anchor: HistoryAnchor(
          logicalLineId: Int64(42),
          cellOffset: 0,
          atEnd: false,
        ),
      );

      expect(historyViewportTailRows(history: history, viewportRows: 4), 3);
    },
  );

  test('anchors an empty live viewport after all frozen history rows', () {
    final history = _history(
      [_textRow(41, 'older')],
      anchor: HistoryAnchor(
        logicalLineId: Int64.ZERO,
        cellOffset: 0,
        atEnd: true,
      ),
    );

    expect(historyViewportAnchorRow(history), 1);
    expect(historyViewportTailRows(history: history, viewportRows: 4), 4);
  });
}

FrozenHistory _history(
  List<HistoryRow> rows, {
  int cols = 80,
  HistoryAnchor? anchor,
}) {
  return FrozenHistory(
    token: 'token-1',
    generation: Int64(7),
    cols: cols,
    rows: rows,
    anchor:
        anchor ??
        HistoryAnchor(
          logicalLineId: rows.first.logicalLineId,
          cellOffset: 0,
          atEnd: true,
        ),
    hasMore: false,
    logicalTotal: rows.length,
  );
}

HistoryRow _textRow(int lineId, String text, {int rowInLine = 0}) {
  return _row(lineId, rowInLine, [(text, text.length)]);
}

HistoryRow _row(int lineId, int rowInLine, List<(String, int)> cells) {
  return HistoryRow(
    logicalLineId: Int64(lineId),
    rowInLine: rowInLine,
    row: ScreenRow(
      cells: [
        for (final cell in cells) ScreenCell(content: cell.$1, width: cell.$2),
      ],
    ),
  );
}
