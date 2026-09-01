import 'package:anytty_native/src/features/terminal/domain/live_screen_store.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final terminal = TerminalRef(endpointId: 'endpoint-1', terminalId: 'term-1');

  test('merges row copies from the unchanged base before replacements', () {
    final baseline = mergeLiveScreen(
      current: null,
      incoming: NativeScreenResult(
        terminal: terminal,
        liveRevision: Int64(7),
        size: TerminalSize(cols: 8, rows: 3),
        rowReplacements: [
          _replacement(0, 'zero'),
          _replacement(1, 'one'),
          _replacement(2, 'two'),
        ],
        fullReplace: true,
      ),
      connectionGeneration: Int64(4),
      expectedTerminal: terminal,
    ) as LiveScreenMerged;

    final merged = mergeLiveScreen(
      current: baseline.screen,
      incoming: NativeScreenResult(
        terminal: terminal,
        liveRevision: Int64(8),
        baseRevision: Int64(7),
        size: TerminalSize(cols: 8, rows: 3),
        rowCopies: [ScreenRowCopy(sourceRow: 0, destinationRow: 1, count: 2)],
        rowReplacements: [_replacement(0, 'new')],
      ),
      connectionGeneration: Int64(4),
      expectedTerminal: terminal,
    );

    expect(merged, isA<LiveScreenMerged>());
    final screen = (merged as LiveScreenMerged).screen;
    expect(_text(screen.screenRows[0]), 'new');
    expect(_text(screen.screenRows[1]), 'zero');
    expect(_text(screen.screenRows[2]), 'one');
    expect(merged.damage.changedRows, [0, 1, 2]);
  });

  test('rejects a delta whose base revision does not match', () {
    final current = CanonicalLiveScreen(
      terminal: terminal,
      connectionGeneration: Int64(2),
      revision: Int64(9),
      cols: 80,
      rows: 1,
      screenRows: [ScreenRow()],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64.ZERO,
    );

    final outcome = mergeLiveScreen(
      current: current,
      incoming: NativeScreenResult(
        terminal: terminal,
        liveRevision: Int64(11),
        baseRevision: Int64(8),
        size: TerminalSize(cols: 80, rows: 1),
      ),
      connectionGeneration: Int64(2),
      expectedTerminal: terminal,
    );

    expect(outcome, isA<LiveScreenRejected>());
    expect((outcome as LiveScreenRejected).requestFullReplace, isTrue);
  });

  test('requires every row in a full replacement', () {
    final outcome = mergeLiveScreen(
      current: null,
      incoming: NativeScreenResult(
        terminal: terminal,
        liveRevision: Int64.ONE,
        size: TerminalSize(cols: 8, rows: 2),
        rowReplacements: [_replacement(0, 'only one')],
        fullReplace: true,
      ),
      connectionGeneration: Int64.ONE,
      expectedTerminal: terminal,
    );

    expect(outcome, isA<LiveScreenRejected>());
  });
}

ScreenRowReplace _replacement(int index, String text) {
  return ScreenRowReplace(
    rowIndex: index,
    row: ScreenRow(
      cells: [ScreenCell(content: text, width: text.length)],
    ),
  );
}

String _text(ScreenRow row) => row.cells.map((cell) => cell.content).join();
