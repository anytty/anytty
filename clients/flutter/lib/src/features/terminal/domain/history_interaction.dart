import 'dart:collection';

import 'package:fixnum/fixnum.dart';

import '../../../generated/proto/apipb/history.pb.dart';
import 'history_store.dart';

typedef HistoryRowRange = ({int start, int end});
typedef HistoryPrefetchPlan = ({int thresholdRows, int requestRows});

HistoryPrefetchPlan adaptiveHistoryPrefetchPlan({
  required int baseThresholdRows,
  required double upwardVelocityRowsPerSecond,
}) {
  final base = baseThresholdRows.clamp(0, 1000);
  final velocity = upwardVelocityRowsPerSecond.isFinite
      ? upwardVelocityRowsPerSecond.clamp(0.0, 1000.0)
      : 0.0;
  final leadRows = (maximumHistoryWindowRequestRows + velocity * 2.5)
      .round()
      .clamp(
        maximumHistoryWindowRequestRows,
        maximumHistoryWindowRequestRows * 3,
      );
  return (
    thresholdRows: (base + leadRows).clamp(
      base,
      maximumHistoryWindowRequestRows * 3,
    ),
    requestRows: maximumHistoryWindowRequestRows,
  );
}

bool historyRowIsVisibleInViewport({
  required int rowIndex,
  required double rowHeight,
  required double scrollOffset,
  required double viewportExtent,
}) {
  if (rowIndex < 0 ||
      !rowHeight.isFinite ||
      rowHeight <= 0 ||
      !scrollOffset.isFinite ||
      !viewportExtent.isFinite ||
      viewportExtent <= 0) {
    return false;
  }
  const tolerance = 0.5;
  final rowTop = rowIndex * rowHeight;
  final rowBottom = rowTop + rowHeight;
  return rowTop >= scrollOffset - tolerance &&
      rowBottom <= scrollOffset + viewportExtent + tolerance;
}

final class HistoryCellPoint {
  const HistoryCellPoint({required this.lineId, required this.column});

  final Int64 lineId;
  final int column;
}

final class HistorySelection {
  const HistorySelection({required this.anchor, required this.focus});

  final HistoryCellPoint anchor;
  final HistoryCellPoint focus;
}

({HistoryRange match, bool wrapped})? adjacentHistorySearchMatch({
  required List<HistoryRange> matches,
  required HistoryRange? current,
  required bool forward,
}) {
  if (matches.isEmpty) return null;
  if (current == null) {
    return (match: forward ? matches.first : matches.last, wrapped: false);
  }
  final currentIndex = matches.indexWhere(
    (match) => sameHistoryRange(match, current),
  );
  if (currentIndex < 0) {
    return (match: forward ? matches.first : matches.last, wrapped: false);
  }
  final nextIndex = forward ? currentIndex + 1 : currentIndex - 1;
  if (nextIndex >= matches.length) {
    return (match: matches.first, wrapped: true);
  }
  if (nextIndex < 0) {
    return (match: matches.last, wrapped: true);
  }
  return (match: matches[nextIndex], wrapped: false);
}

bool sameHistoryRange(HistoryRange left, HistoryRange right) =>
    left.startLineId == right.startLineId &&
    left.startCol == right.startCol &&
    left.endLineId == right.endLineId &&
    left.endCol == right.endCol;

final class HistoryRowLayout {
  const HistoryRowLayout({
    required this.index,
    required this.lineId,
    required this.lineOrder,
    required this.logicalStartColumn,
    required this.width,
  });

  final int index;
  final Int64 lineId;
  final int lineOrder;
  final int logicalStartColumn;
  final int width;

  int get logicalEndColumn => logicalStartColumn + width;
}

final class HistoryLayout {
  HistoryLayout._({
    required List<HistoryRowLayout> rows,
    required Map<Int64, int> lineOrder,
    required Map<Int64, int> lineWidths,
    required Map<Int64, int> firstRowByLine,
  }) : rows = UnmodifiableListView(rows),
       lineOrder = UnmodifiableMapView(lineOrder),
       lineWidths = UnmodifiableMapView(lineWidths),
       firstRowByLine = UnmodifiableMapView(firstRowByLine);

  final UnmodifiableListView<HistoryRowLayout> rows;
  final UnmodifiableMapView<Int64, int> lineOrder;
  final UnmodifiableMapView<Int64, int> lineWidths;
  final UnmodifiableMapView<Int64, int> firstRowByLine;

  static HistoryLayout fromHistory(FrozenHistory history) {
    final rows = <HistoryRowLayout>[];
    final orders = <Int64, int>{};
    final widths = <Int64, int>{};
    final firstRows = <Int64, int>{};
    HistoryRowLayout? previous;

    for (var index = 0; index < history.rows.length; index += 1) {
      final row = history.rows[index];
      final lineId = row.logicalLineId;
      firstRows.putIfAbsent(lineId, () => index);
      final order = orders.putIfAbsent(lineId, () => orders.length);
      final width = historyScreenRowWidth(row.row);
      final continuesPrevious =
          previous != null &&
          previous.lineId == lineId &&
          history.rows[index - 1].rowInLine + 1 == row.rowInLine;
      final start = row.rowInLine <= 0
          ? 0
          : continuesPrevious
          ? previous.logicalEndColumn
          : row.rowInLine * history.cols;
      final layout = HistoryRowLayout(
        index: index,
        lineId: lineId,
        lineOrder: order,
        logicalStartColumn: start,
        width: width,
      );
      rows.add(layout);
      previous = layout;
      final end = layout.logicalEndColumn;
      if (end > (widths[lineId] ?? 0)) widths[lineId] = end;
    }
    return HistoryLayout._(
      rows: rows,
      lineOrder: orders,
      lineWidths: widths,
      firstRowByLine: firstRows,
    );
  }
}

int historyViewportAnchorRow(FrozenHistory history) {
  if (history.anchor.atEnd) return history.rows.length;
  var remaining = history.anchor.cellOffset.clamp(0, 1 << 30);
  var found = false;
  for (var index = 0; index < history.rows.length; index += 1) {
    final row = history.rows[index];
    if (row.logicalLineId != history.anchor.logicalLineId) {
      if (found) break;
      continue;
    }
    found = true;
    if (remaining == 0) return index;
    final width = historyScreenRowWidth(row.row);
    if (remaining < width) return index;
    remaining -= width;
  }
  return history.rows.length;
}

int historyViewportTailRows({
  required FrozenHistory history,
  required int viewportRows,
}) {
  if (viewportRows <= 0) return 0;
  final anchorRow = historyViewportAnchorRow(history);
  final rowsAfterAnchor = history.rows.length - anchorRow;
  return (viewportRows - rowsAfterAnchor).clamp(0, viewportRows);
}

HistoryCellPoint historyPointForRowColumn({
  required HistoryLayout layout,
  required int rowIndex,
  required int localColumn,
}) {
  if (layout.rows.isEmpty) {
    throw StateError('History has no selectable rows');
  }
  final row = layout.rows[rowIndex.clamp(0, layout.rows.length - 1)];
  return HistoryCellPoint(
    lineId: row.lineId,
    column: row.logicalStartColumn + localColumn.clamp(0, row.width),
  );
}

HistorySelection selectAllHistory(HistoryLayout layout) {
  if (layout.rows.isEmpty) {
    throw StateError('History has no selectable rows');
  }
  final first = layout.rows.first;
  final last = layout.rows.last;
  return HistorySelection(
    anchor: HistoryCellPoint(
      lineId: first.lineId,
      column: first.logicalStartColumn,
    ),
    focus: HistoryCellPoint(lineId: last.lineId, column: last.logicalEndColumn),
  );
}

HistorySelection selectVisibleHistory({
  required HistoryLayout layout,
  required int firstRow,
  required int lastRow,
}) {
  if (layout.rows.isEmpty) {
    throw StateError('History has no selectable rows');
  }
  final first = layout.rows[firstRow.clamp(0, layout.rows.length - 1)];
  final last = layout.rows[lastRow.clamp(first.index, layout.rows.length - 1)];
  return HistorySelection(
    anchor: HistoryCellPoint(
      lineId: first.lineId,
      column: first.logicalStartColumn,
    ),
    focus: HistoryCellPoint(lineId: last.lineId, column: last.logicalEndColumn),
  );
}

HistoryRange normalizeHistorySelection({
  required HistoryLayout layout,
  required HistorySelection selection,
}) {
  final anchor = _clampPoint(layout, selection.anchor);
  final focus = _clampPoint(layout, selection.focus);
  final anchorOrder = layout.lineOrder[anchor.lineId]!;
  final focusOrder = layout.lineOrder[focus.lineId]!;
  final forward =
      anchorOrder < focusOrder ||
      (anchorOrder == focusOrder && anchor.column <= focus.column);
  final start = forward ? anchor : focus;
  final end = forward ? focus : anchor;
  return HistoryRange(
    startLineId: start.lineId,
    startCol: start.column,
    endLineId: end.lineId,
    endCol: end.column,
  );
}

HistoryRowRange? projectHistoryRangeToRow({
  required HistoryLayout layout,
  required int rowIndex,
  required HistoryRange range,
}) {
  if (rowIndex < 0 || rowIndex >= layout.rows.length) return null;
  final row = layout.rows[rowIndex];
  final startOrder = layout.lineOrder[range.startLineId];
  final endOrder = layout.lineOrder[range.endLineId];
  if (startOrder == null || endOrder == null || startOrder > endOrder) {
    return null;
  }
  if (row.lineOrder < startOrder || row.lineOrder > endOrder) return null;

  var start = row.logicalStartColumn;
  var end = row.logicalEndColumn;
  if (row.lineOrder == startOrder) start = range.startCol;
  if (row.lineOrder == endOrder) end = range.endCol;
  final localStart = (start - row.logicalStartColumn).clamp(0, row.width);
  final localEnd = (end - row.logicalStartColumn).clamp(0, row.width);
  if (localEnd <= localStart) return null;
  return (start: localStart, end: localEnd);
}

int? historyRowIndexForRange({
  required HistoryLayout layout,
  required HistoryRange range,
}) {
  final firstRow = layout.firstRowByLine[range.startLineId];
  final endOrder = layout.lineOrder[range.endLineId];
  if (firstRow == null || endOrder == null) return null;
  for (var index = firstRow; index < layout.rows.length; index += 1) {
    if (layout.rows[index].lineOrder > endOrder) break;
    if (projectHistoryRangeToRow(
          layout: layout,
          rowIndex: index,
          range: range,
        ) !=
        null) {
      return index;
    }
  }
  return null;
}

Map<int, List<HistoryRowRange>> projectHistoryRangesToRows({
  required HistoryLayout layout,
  required List<HistoryRange> ranges,
}) {
  final projected = <int, List<HistoryRowRange>>{};
  for (final range in ranges) {
    final firstRow = layout.firstRowByLine[range.startLineId];
    final endOrder = layout.lineOrder[range.endLineId];
    if (firstRow == null || endOrder == null) continue;
    for (var index = firstRow; index < layout.rows.length; index += 1) {
      if (layout.rows[index].lineOrder > endOrder) break;
      final rowRange = projectHistoryRangeToRow(
        layout: layout,
        rowIndex: index,
        range: range,
      );
      if (rowRange != null) {
        (projected[index] ??= []).add(rowRange);
      }
    }
  }
  for (final ranges in projected.values) {
    ranges.sort((left, right) => left.start.compareTo(right.start));
  }
  return projected;
}

HistoryCellPoint _clampPoint(HistoryLayout layout, HistoryCellPoint point) {
  final width = layout.lineWidths[point.lineId];
  if (width == null) {
    throw StateError('History selection is outside the loaded window');
  }
  return HistoryCellPoint(
    lineId: point.lineId,
    column: point.column.clamp(0, width),
  );
}

int historyScreenRowWidth(ScreenRow row) {
  return row.cells.fold(
    0,
    (width, cell) => width + cell.width.clamp(0, 1 << 20),
  );
}
