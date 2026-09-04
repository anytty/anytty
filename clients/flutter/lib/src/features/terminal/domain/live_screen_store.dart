import 'dart:collection';

import 'package:fixnum/fixnum.dart';

import '../../../generated/proto/apipb/history.pb.dart';
import '../../../generated/proto/apipb/terminal.pb.dart';

final class CanonicalLiveScreen {
  CanonicalLiveScreen({
    required TerminalRef terminal,
    required this.connectionGeneration,
    required this.revision,
    required this.cols,
    required this.rows,
    required List<ScreenRow> screenRows,
    required this.alternateScreen,
    required TerminalCursor? cursor,
    required TerminalModes? modes,
    required this.timestampUnixNano,
  }) : terminal = terminal.deepCopy(),
       screenRows = UnmodifiableListView(
         screenRows.map((row) => row.deepCopy()),
       ),
       cursor = cursor?.deepCopy(),
       modes = modes?.deepCopy();

  CanonicalLiveScreen._owned({
    required TerminalRef terminal,
    required this.connectionGeneration,
    required this.revision,
    required this.cols,
    required this.rows,
    required List<ScreenRow> screenRows,
    required this.alternateScreen,
    required TerminalCursor? cursor,
    required TerminalModes? modes,
    required this.timestampUnixNano,
  }) : terminal = terminal.deepCopy(),
       screenRows = UnmodifiableListView(screenRows),
       cursor = cursor?.deepCopy(),
       modes = modes?.deepCopy();

  final TerminalRef terminal;
  final Int64 connectionGeneration;
  final Int64 revision;
  final int cols;
  final int rows;
  final UnmodifiableListView<ScreenRow> screenRows;
  final bool alternateScreen;
  final TerminalCursor? cursor;
  final TerminalModes? modes;
  final Int64 timestampUnixNano;
}

final class LiveScreenDamage {
  const LiveScreenDamage({
    required this.fullReplace,
    required this.changedRows,
  });

  final bool fullReplace;
  final List<int> changedRows;
}

sealed class LiveScreenMergeOutcome {
  const LiveScreenMergeOutcome();
}

final class LiveScreenMerged extends LiveScreenMergeOutcome {
  const LiveScreenMerged({required this.screen, required this.damage});

  final CanonicalLiveScreen screen;
  final LiveScreenDamage damage;
}

final class LiveScreenRejected extends LiveScreenMergeOutcome {
  const LiveScreenRejected({
    required this.reason,
    required this.requestFullReplace,
  });

  final String reason;
  final bool requestFullReplace;
}

LiveScreenMergeOutcome mergeLiveScreen({
  required CanonicalLiveScreen? current,
  required NativeScreenResult incoming,
  required Int64 connectionGeneration,
  required TerminalRef expectedTerminal,
}) {
  final terminal = incoming.hasTerminal()
      ? incoming.terminal
      : expectedTerminal;
  if (!_sameTerminal(terminal, expectedTerminal)) {
    return const LiveScreenRejected(
      reason: 'terminal identity mismatch',
      requestFullReplace: false,
    );
  }

  final cols = incoming.hasSize() ? incoming.size.cols : 0;
  final rows = incoming.hasSize() ? incoming.size.rows : 0;
  if (cols < 0 || rows < 0) {
    return const LiveScreenRejected(
      reason: 'negative terminal dimensions',
      requestFullReplace: true,
    );
  }

  if (current != null &&
      current.connectionGeneration == connectionGeneration &&
      incoming.liveRevision < current.revision) {
    return const LiveScreenRejected(
      reason: 'stale live revision',
      requestFullReplace: false,
    );
  }

  if (!incoming.fullReplace) {
    if (current == null ||
        current.connectionGeneration != connectionGeneration ||
        incoming.baseRevision != current.revision ||
        current.cols != cols ||
        current.rows != rows) {
      return const LiveScreenRejected(
        reason: 'delta base is unavailable',
        requestFullReplace: true,
      );
    }
  }

  // Delta frames only replace a small number of rows. Keep unchanged rows by
  // reference and copy the rows that are changed before publishing the frame.
  final baseRows = incoming.fullReplace
      ? List<ScreenRow>.generate(rows, (_) => ScreenRow())
      : List<ScreenRow>.of(current!.screenRows);
  final nextRows = List<ScreenRow>.of(baseRows);
  final changedRows = <int>{};
  final copiedDestinations = <int>{};

  for (final copy in incoming.rowCopies) {
    if (incoming.fullReplace ||
        copy.count < 0 ||
        copy.sourceRow < 0 ||
        copy.destinationRow < 0 ||
        copy.sourceRow + copy.count > rows ||
        copy.destinationRow + copy.count > rows) {
      return const LiveScreenRejected(
        reason: 'invalid row copy',
        requestFullReplace: true,
      );
    }
    for (var offset = 0; offset < copy.count; offset += 1) {
      final destination = copy.destinationRow + offset;
      if (!copiedDestinations.add(destination)) {
        return const LiveScreenRejected(
          reason: 'overlapping row copies',
          requestFullReplace: true,
        );
      }
      nextRows[destination] = baseRows[copy.sourceRow + offset].deepCopy();
      changedRows.add(destination);
    }
  }

  final replacedRows = <int>{};
  for (final replacement in incoming.rowReplacements) {
    if (replacement.rowIndex < 0 ||
        replacement.rowIndex >= rows ||
        !replacement.hasRow() ||
        !replacedRows.add(replacement.rowIndex)) {
      return const LiveScreenRejected(
        reason: 'invalid row replacement',
        requestFullReplace: true,
      );
    }
    nextRows[replacement.rowIndex] = replacement.row.deepCopy();
    changedRows.add(replacement.rowIndex);
  }

  if (incoming.fullReplace && replacedRows.length != rows) {
    return const LiveScreenRejected(
      reason: 'incomplete full replacement',
      requestFullReplace: true,
    );
  }

  final sortedDamage = changedRows.toList()..sort();
  return LiveScreenMerged(
    screen: CanonicalLiveScreen._owned(
      terminal: terminal,
      connectionGeneration: connectionGeneration,
      revision: incoming.liveRevision,
      cols: cols,
      rows: rows,
      screenRows: nextRows,
      alternateScreen: incoming.alternateScreen,
      cursor: incoming.hasCursor() ? incoming.cursor : null,
      modes: incoming.hasModes() ? incoming.modes : null,
      timestampUnixNano: incoming.timestampUnixNano,
    ),
    damage: LiveScreenDamage(
      fullReplace: incoming.fullReplace,
      changedRows: List.unmodifiable(sortedDamage),
    ),
  );
}

bool _sameTerminal(TerminalRef left, TerminalRef right) {
  return left.endpointId == right.endpointId &&
      left.terminalId == right.terminalId;
}
