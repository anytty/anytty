import 'package:anytty_native/src/features/terminal/domain/history_store.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('prepends older rows while retaining the logical viewport anchor', () {
    final initial = mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(30, 'current-a'), _row(31, 'current-b')],
        topLineId: 30,
      ),
    ) as HistoryMerged;

    final outcome = mergeHistoryWindow(
      current: initial.history,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
        rows: [_row(10, 'older-a'), _row(20, 'older-b')],
        topLineId: 10,
      ),
    );

    expect(outcome, isA<HistoryMerged>());
    final merged = outcome as HistoryMerged;
    expect(merged.prependedRows, 2);
    expect(merged.history.rows.map((row) => row.logicalLineId.toInt()), [
      10,
      20,
      30,
      31,
    ]);
    expect(merged.history.anchor.logicalLineId.toInt(), 30);
    expect(merged.history.anchor.cellOffset, 3);
  });

  test('prepend inherits the frozen anchor when the page omits it', () {
    final initial = mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(30, 'current')],
        topLineId: 30,
      ),
    ) as HistoryMerged;
    final older = _window(
      operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
      rows: [_row(20, 'older')],
      topLineId: 20,
    )..clearViewportAnchor();

    final outcome = mergeHistoryWindow(
      current: initial.history,
      incoming: older,
    );

    expect(outcome, isA<HistoryMerged>());
    final merged = outcome as HistoryMerged;
    expect(merged.prependedRows, 1);
    expect(merged.history.anchor.logicalLineId.toInt(), 30);
    expect(merged.history.anchor.cellOffset, 3);
  });

  test('stops frozen history growth at the resident visual-row limit', () {
    final initial = mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(30, 'current-a'), _row(31, 'current-b')],
        topLineId: 30,
      ),
      maximumRows: 3,
    ) as HistoryMerged;

    final merged = mergeHistoryWindow(
      current: initial.history,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
        rows: [_row(10, 'oldest'), _row(20, 'older'), _row(25, 'nearest')],
        topLineId: 10,
      ),
      maximumRows: 3,
    ) as HistoryMerged;

    expect(merged.prependedRows, 1);
    expect(merged.history.rows.map((row) => row.logicalLineId.toInt()), [
      25,
      30,
      31,
    ]);
    expect(merged.history.hasMore, isFalse);
  });

  test('bounds an oversized replacement window to its newest rows', () {
    final merged = mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [
          _row(10, 'oldest'),
          _row(20, 'older'),
          _row(30, 'newer'),
          _row(40, 'newest'),
        ],
        topLineId: 30,
      ),
      maximumRows: 3,
    ) as HistoryMerged;

    expect(merged.history.rows.map((row) => row.logicalLineId.toInt()), [
      20,
      30,
      40,
    ]);
    expect(merged.history.hasMore, isFalse);
  });

  test('replace still requires a viewport anchor', () {
    final incoming = _window(
      operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
      rows: [_row(30, 'current')],
      topLineId: 30,
    )..clearViewportAnchor();

    final outcome = mergeHistoryWindow(current: null, incoming: incoming);

    expect(outcome, isA<HistoryRejected>());
    expect(
      (outcome as HistoryRejected).reason,
      'incomplete frozen history metadata',
    );
  });

  test('rejects prepend from another frozen generation', () {
    final current = (mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(30, 'current')],
        topLineId: 30,
      ),
    ) as HistoryMerged).history;

    final stale = _window(
      operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
      rows: [_row(20, 'stale')],
      topLineId: 20,
    )..historyGeneration = Int64(8);

    expect(
      mergeHistoryWindow(current: current, incoming: stale),
      isA<HistoryRejected>(),
    );
  });

  test('rejects server append because mobile history grows at the head', () {
    final outcome = mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_APPEND,
        rows: [_row(10, 'unexpected')],
        topLineId: 10,
      ),
    );

    expect(outcome, isA<HistoryRejected>());
  });

  test('rejects a prepend that repeats the current boundary row', () {
    final current = (mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(20, 'current-a'), _row(30, 'current-b')],
        topLineId: 20,
      ),
    ) as HistoryMerged).history;

    final outcome = mergeHistoryWindow(
      current: current,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
        rows: [_row(10, 'older'), _row(20, 'duplicate')],
        topLineId: 10,
      ),
    );

    expect(outcome, isA<HistoryRejected>());
  });

  test('rejects rows that are out of order within a prepend page', () {
    final current = (mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(30, 'current')],
        topLineId: 30,
      ),
    ) as HistoryMerged).history;

    final outcome = mergeHistoryWindow(
      current: current,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
        rows: [_row(20, 'newer'), _row(10, 'older')],
        topLineId: 20,
      ),
    );

    expect(outcome, isA<HistoryRejected>());
  });

  test('classifies only frozen history identity failures for reload', () {
    expect(frozenHistoryRequiresReload('stale frozen history window'), isTrue);
    expect(frozenHistoryRequiresReload('history token expired'), isTrue);
    expect(frozenHistoryRequiresReload('network timeout'), isFalse);
    expect(frozenHistoryRequiresReload('logical line is too large'), isFalse);
  });

  test('reflows logical source rows at the requested local columns', () {
    final merged = mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(42, 'abcdefghij')],
        topLineId: 42,
        cols: 4,
      ),
    ) as HistoryMerged;

    expect(merged.history.rows.map(_rowText), ['abcd', 'efgh', 'ij']);
    expect(merged.history.rows.map((row) => row.rowInLine), [0, 1, 2]);
    expect(merged.history.rows.map((row) => row.wrapped), [true, true, false]);
  });

  test('keeps wide graphemes intact while reflowing', () {
    final source = HistoryRow(
      logicalLineId: Int64(42),
      row: ScreenRow(
        cells: [
          ScreenCell(content: 'a', width: 1),
          ScreenCell(content: '\u4e2d', width: 2),
          ScreenCell(content: 'b', width: 1),
        ],
      ),
    );

    final rows = reflowHistoryRows([source], 3);

    expect(rows.map(_rowText), ['a\u4e2d', 'b']);
    expect(rows.first.row.cells.map((cell) => cell.width), [1, 2]);
    expect(rows.last.rowInLine, 1);
  });

  test('materializes styled blank footprints without losing style', () {
    final source = HistoryRow(
      logicalLineId: Int64(42),
      row: ScreenRow(
        cells: [
          ScreenCell(content: 'X', width: 1),
          ScreenCell(
            width: 3,
            style: CellStyle(background: 'ansi:4'),
            linkUrl: 'https://example.test',
          ),
          ScreenCell(content: 'Y', width: 1),
        ],
      ),
    );

    final rows = reflowHistoryRows([source], 2);

    expect(rows.map(_rowText), ['X ', '  ', 'Y']);
    final blanks = rows
        .expand((row) => row.row.cells)
        .where((cell) => cell.content == ' ');
    expect(blanks, hasLength(3));
    expect(blanks.every((cell) => cell.style.background == 'ansi:4'), isTrue);
    expect(blanks.every((cell) => cell.linkUrl.isEmpty), isTrue);
  });

  test('preserves fixed-grid frame geometry at narrower local columns', () {
    final source = HistoryRow(
      logicalLineId: Int64(42),
      fixedGrid: true,
      screenCols: 12,
      row: ScreenRow(cells: [ScreenCell(content: 'abcdefghijkl', width: 12)]),
    );

    final rows = reflowHistoryRows([source], 4);

    expect(rows, hasLength(1));
    expect(_rowText(rows.single), 'abcdefghijkl');
    expect(rows.single.row.cells.single.width, 12);
    expect(rows.single.screenCols, 12);
  });

  test('reports visual rows inserted by an older logical page', () {
    final current = (mergeHistoryWindow(
      current: null,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
        rows: [_row(30, 'tail')],
        topLineId: 30,
        cols: 4,
      ),
    ) as HistoryMerged).history;

    final merged = mergeHistoryWindow(
      current: current,
      incoming: _window(
        operation: HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
        rows: [_row(20, 'abcdefgh')],
        topLineId: 20,
        cols: 4,
      ),
    ) as HistoryMerged;

    expect(merged.prependedRows, 2);
    expect(merged.history.rows.map(_rowText), ['abcd', 'efgh', 'tail']);
  });
}

HistoryWindowResult _window({
  required HistoryWindowOperation operation,
  required List<HistoryRow> rows,
  required int topLineId,
  int cols = 80,
}) {
  return HistoryWindowResult(
    token: 'frozen-1',
    operation: operation,
    size: TerminalSize(cols: cols, rows: 24),
    rows: rows,
    historyGeneration: Int64(7),
    logicalTotal: 42,
    hasMore: true,
    viewportAnchor: HistoryViewportAnchor(
      topLineId: Int64(topLineId),
      topCellOffset: 3,
    ),
  );
}

String _rowText(HistoryRow row) =>
    row.row.cells.map((cell) => cell.content).join();

HistoryRow _row(int lineId, String text) {
  return HistoryRow(
    row: ScreenRow(
      cells: [ScreenCell(content: text, width: text.length)],
    ),
    logicalLineId: Int64(lineId),
  );
}
