import 'dart:collection';

import 'package:fixnum/fixnum.dart';

import '../../../generated/proto/apipb/history.pb.dart';
import 'terminal_text_width.dart';

const maximumFrozenHistoryRows = 8192;
const maximumHistoryWindowRequestRows = 512;

final class HistoryAnchor {
  const HistoryAnchor({
    required this.logicalLineId,
    required this.cellOffset,
    required this.atEnd,
    this.screenCols = 0,
    this.screenRows = 0,
  });

  final Int64 logicalLineId;
  final int cellOffset;
  final bool atEnd;
  final int screenCols;
  final int screenRows;

  factory HistoryAnchor.fromProto(HistoryViewportAnchor value) {
    return HistoryAnchor(
      logicalLineId: value.topLineId,
      cellOffset: value.topCellOffset,
      atEnd: value.atEnd,
      screenCols: value.screenCols,
      screenRows: value.screenRows,
    );
  }
}

final class FrozenHistory {
  FrozenHistory({
    required this.token,
    required this.generation,
    required this.cols,
    required List<HistoryRow> rows,
    required this.anchor,
    required this.hasMore,
    required this.logicalTotal,
  }) : rows = UnmodifiableListView(rows.map((row) => row.deepCopy()));

  final String token;
  final Int64 generation;
  final int cols;
  final UnmodifiableListView<HistoryRow> rows;
  final HistoryAnchor anchor;
  final bool hasMore;
  final int logicalTotal;
}

sealed class HistoryMergeOutcome {
  const HistoryMergeOutcome();
}

bool frozenHistoryRequiresReload(String message) {
  final normalized = message.toLowerCase();
  return normalized.contains('stale frozen history') ||
      normalized.contains('history token') ||
      normalized.contains('token expired') ||
      normalized.contains('history generation');
}

final class HistoryMerged extends HistoryMergeOutcome {
  const HistoryMerged({required this.history, required this.prependedRows});

  final FrozenHistory history;
  final int prependedRows;
}

final class HistoryRejected extends HistoryMergeOutcome {
  const HistoryRejected(this.reason);

  final String reason;
}

HistoryMergeOutcome mergeHistoryWindow({
  required FrozenHistory? current,
  required HistoryWindowResult incoming,
  int maximumRows = maximumFrozenHistoryRows,
}) {
  if (maximumRows <= 0) {
    return const HistoryRejected('frozen history row limit is invalid');
  }
  if (incoming.token.isEmpty ||
      incoming.historyGeneration == Int64.ZERO ||
      !incoming.hasSize() ||
      incoming.size.cols <= 0) {
    return const HistoryRejected('incomplete frozen history metadata');
  }

  final operation = incoming.operation;
  if (operation == HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE) {
    if (!incoming.hasViewportAnchor()) {
      return const HistoryRejected('incomplete frozen history metadata');
    }
    final projectedRows = reflowHistoryRows(incoming.rows, incoming.size.cols);
    final truncated = projectedRows.length > maximumRows;
    final rows = truncated
        ? projectedRows.sublist(projectedRows.length - maximumRows)
        : projectedRows;
    return HistoryMerged(
      history: FrozenHistory(
        token: incoming.token,
        generation: incoming.historyGeneration,
        cols: incoming.size.cols,
        rows: rows,
        anchor: HistoryAnchor.fromProto(incoming.viewportAnchor),
        hasMore: !truncated && incoming.hasMore,
        logicalTotal: incoming.logicalTotal,
      ),
      prependedRows: 0,
    );
  }

  if (operation != HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND) {
    return const HistoryRejected('unsupported history window operation');
  }
  if (current == null ||
      current.token != incoming.token ||
      current.generation != incoming.historyGeneration ||
      current.cols != incoming.size.cols) {
    return const HistoryRejected('stale frozen history window');
  }

  if (!_historyRowsOrdered(incoming.rows)) {
    return const HistoryRejected('prepended history rows are out of order');
  }
  final incomingRows = reflowHistoryRows(incoming.rows, incoming.size.cols);
  final currentKeys = current.rows.map(_historyRowKey).toSet();
  if (incomingRows.any((row) => currentKeys.contains(_historyRowKey(row)))) {
    return const HistoryRejected('prepended history overlaps current rows');
  }
  if (incomingRows.isNotEmpty && current.rows.isNotEmpty) {
    final incomingLast = incomingRows.last;
    final currentFirst = current.rows.first;
    if (incomingLast.segment == currentFirst.segment &&
        (incomingLast.logicalLineId > currentFirst.logicalLineId ||
            (incomingLast.logicalLineId == currentFirst.logicalLineId &&
                incomingLast.rowInLine >= currentFirst.rowInLine))) {
      return const HistoryRejected(
        'prepended history does not precede current rows',
      );
    }
  }

  final remainingCapacity = (maximumRows - current.rows.length).clamp(
    0,
    maximumRows,
  );
  final acceptedRows = incomingRows.length > remainingCapacity
      ? incomingRows.sublist(incomingRows.length - remainingCapacity)
      : incomingRows;
  // FrozenHistory takes the defensive copies. Keeping this intermediate list
  // borrowed avoids cloning the entire resident window twice on every page.
  final combined = <HistoryRow>[...acceptedRows, ...current.rows];
  return HistoryMerged(
    history: FrozenHistory(
      token: current.token,
      generation: current.generation,
      cols: current.cols,
      rows: combined,
      anchor: current.anchor,
      hasMore:
          acceptedRows.length == incomingRows.length &&
          combined.length < maximumRows &&
          incoming.hasMore,
      logicalTotal: incoming.logicalTotal,
    ),
    prependedRows: acceptedRows.length,
  );
}

List<HistoryRow> reflowHistoryRows(Iterable<HistoryRow> source, int cols) {
  final width = cols > 0 ? cols : 80;
  return [for (final row in source) ..._reflowHistoryRow(row, width)];
}

List<HistoryRow> _reflowHistoryRow(HistoryRow source, int cols) {
  if (source.fixedGrid) {
    return [source.deepCopy()];
  }

  final cells = source.row.cells.expand(_splitHistoryCell).toList();
  if (cells.isEmpty) {
    final row = source.deepCopy()
      ..rowInLine = 0
      ..row = ScreenRow()
      ..wrapped = source.wrapped;
    if (source.row.hasTailFill()) {
      row.row.tailFill = source.row.tailFill.deepCopy();
    }
    return [row];
  }

  final rows = <HistoryRow>[];
  var current = <ScreenCell>[];
  var currentWidth = 0;

  void flush() {
    final row = source.deepCopy()
      ..rowInLine = rows.length
      ..row = ScreenRow(cells: current.map((cell) => cell.deepCopy()))
      ..wrapped = true;
    rows.add(row);
    current = <ScreenCell>[];
    currentWidth = 0;
  }

  for (final cell in cells) {
    final cellWidth = cell.width.clamp(1, 1 << 20);
    if (currentWidth > 0 && currentWidth + cellWidth > cols) flush();
    current.add(cell);
    currentWidth += cellWidth;
    if (currentWidth >= cols) flush();
  }
  if (current.isNotEmpty || rows.isEmpty) flush();
  final last = rows.last..wrapped = source.wrapped;
  if (source.row.hasTailFill()) {
    last.row.tailFill = source.row.tailFill.deepCopy();
  }
  return rows;
}

Iterable<ScreenCell> _splitHistoryCell(ScreenCell source) sync* {
  final authoritativeWidth = source.width.clamp(0, 1 << 20);
  if (authoritativeWidth <= 0) return;
  if (source.content.isEmpty) {
    for (var index = 0; index < authoritativeWidth; index += 1) {
      yield source.deepCopy()
        ..content = ' '
        ..width = 1
        ..clearLinkUrl()
        ..clearLinkParams();
    }
    return;
  }

  final graphemes = terminalGraphemeClusters(source.content);
  if (graphemes.length <= 1) {
    yield source.deepCopy()..width = authoritativeWidth;
    return;
  }

  final naturalWidths = [
    for (final grapheme in graphemes) terminalGraphemeCellWidth(grapheme),
  ];
  final naturalWidth = naturalWidths.fold(0, (sum, width) => sum + width);
  final equalRunWidth = authoritativeWidth % graphemes.length == 0
      ? authoritativeWidth ~/ graphemes.length
      : 0;
  for (var index = 0; index < graphemes.length; index += 1) {
    final width = equalRunWidth > 0 ? equalRunWidth : naturalWidths[index];
    if (width <= 0) continue;
    yield source.deepCopy()
      ..content = graphemes[index]
      ..width = width;
  }
  if (equalRunWidth == 0 && authoritativeWidth > naturalWidth) {
    for (var index = naturalWidth; index < authoritativeWidth; index += 1) {
      yield source.deepCopy()
        ..content = ' '
        ..width = 1
        ..clearLinkUrl()
        ..clearLinkParams();
    }
  }
}

bool _historyRowsOrdered(Iterable<HistoryRow> rows) {
  HistoryRow? previous;
  for (final row in rows) {
    if (previous != null &&
        previous.segment == row.segment &&
        (previous.logicalLineId > row.logicalLineId ||
            (previous.logicalLineId == row.logicalLineId &&
                previous.rowInLine >= row.rowInLine))) {
      return false;
    }
    previous = row;
  }
  return true;
}

String _historyRowKey(HistoryRow row) {
  return '${row.segment.value}:${row.logicalLineId}:${row.rowInLine}';
}
