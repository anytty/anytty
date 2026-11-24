import 'package:anytty_native/src/features/terminal/domain/live_screen_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_canvas.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

ScreenRow _row(String text) => ScreenRow(
  cells: [
    ScreenCell(content: text, width: text.runes.length.clamp(1, 8)),
  ],
);

ScreenRowReplace _replacement(int index, String text) =>
    ScreenRowReplace(rowIndex: index, row: _row(text));

final _terminal = TerminalRef(endpointId: 'e', terminalId: 't');

CanonicalLiveScreen _merge(
  CanonicalLiveScreen? current,
  NativeScreenResult incoming,
) {
  final outcome = mergeLiveScreen(
    current: current,
    incoming: incoming,
    connectionGeneration: Int64(1),
    expectedTerminal: _terminal,
  );
  return (outcome as LiveScreenMerged).screen;
}

void main() {
  testWidgets('行级画面缓存只重录脏行', (tester) async {
    // 8 行基线
    final baseline = _merge(
      null,
      NativeScreenResult(
        terminal: _terminal,
        liveRevision: Int64(1),
        size: TerminalSize(cols: 40, rows: 8),
        rowReplacements: [
          for (var index = 0; index < 8; index += 1)
            _replacement(index, 'row$index'),
        ],
        fullReplace: true,
      ),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 400,
            height: 300,
            child: TerminalCanvas(
              screen: baseline,
              settings: defaultTerminalSettings,
            ),
          ),
        ),
      ),
    );
    debugTakeTerminalPaintStats();

    // 只改 2 行：其余行必须命中缓存
    final next = _merge(
      baseline,
      NativeScreenResult(
        terminal: _terminal,
        liveRevision: Int64(2),
        baseRevision: Int64(1),
        size: TerminalSize(cols: 40, rows: 8),
        rowReplacements: [_replacement(0, 'new0'), _replacement(5, 'new5')],
      ),
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 400,
            height: 300,
            child: TerminalCanvas(
              screen: next,
              settings: defaultTerminalSettings,
            ),
          ),
        ),
      ),
    );

    final stats = debugTakeTerminalPaintStats();
    expect(stats.recorded, 2, reason: '只有 2 行变化，应该只重录 2 行');
    expect(stats.hits, 6, reason: '其余 6 行应直接复用缓存的 Picture');
  });

  testWidgets('主题或字号变化会让整屏重新录制', (tester) async {
    final baseline = _merge(
      null,
      NativeScreenResult(
        terminal: _terminal,
        liveRevision: Int64(1),
        size: TerminalSize(cols: 40, rows: 4),
        rowReplacements: [
          for (var index = 0; index < 4; index += 1)
            _replacement(index, 'row$index'),
        ],
        fullReplace: true,
      ),
    );

    Future<void> pump(TerminalSettings settings) => tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 400,
            height: 300,
            child: TerminalCanvas(screen: baseline, settings: settings),
          ),
        ),
      ),
    );

    await pump(defaultTerminalSettings);
    debugTakeTerminalPaintStats();

    await pump(defaultTerminalSettings.copyWith(fontSize: 18));
    final stats = debugTakeTerminalPaintStats();
    expect(stats.recorded, 4, reason: 'cell 尺寸变化必须整屏重录');
  });
}
