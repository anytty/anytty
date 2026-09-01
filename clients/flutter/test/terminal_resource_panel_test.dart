import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_resource_panel.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'combines history and current snapshot without duplicating the tail',
    () {
      final tail = _usage(220, 192 * 1024 * 1024, 2);
      final terminal = TerminalInfo(
        resourceHistory: [_usage(120, 180 * 1024 * 1024, 1), tail],
        resources: tail.deepCopy(),
      );

      final samples = terminalResourceSamples(terminal);

      expect(samples, hasLength(2));
      expect(samples.last.cpuPercentX100, 220);
    },
  );

  test('formats resource sizes for compact metric labels', () {
    expect(formatResourceBytes(512), '512 B');
    expect(formatResourceBytes(1536), '1.5 KiB');
    expect(formatResourceBytes(128 * 1024 * 1024), '128.0 MiB');
  });

  test('summarizes current running terminal resource snapshots', () {
    final terminals = [
      TerminalInfo(
        state: TerminalState.TERMINAL_STATE_RUNNING,
        resources: _usage(120, 180 * 1024 * 1024, 1),
      ),
      TerminalInfo(
        state: TerminalState.TERMINAL_STATE_RUNNING,
        resources: _usage(350, 220 * 1024 * 1024, 2),
      ),
      TerminalInfo(
        state: TerminalState.TERMINAL_STATE_EXITED,
        resources: _usage(9000, 1024 * 1024 * 1024, 3),
      ),
    ];

    final totals = terminalResourceTotals(terminals);

    expect(totals?.cpuX100, 470);
    expect(totals?.memoryBytes, 400 * 1024 * 1024);
    expect(totals?.reportingCount, 2);
    expect(totals?.runningCount, 2);
  });

  test('right-aligns terminal histories into an aggregate resource curve', () {
    final terminals = [
      TerminalInfo(
        state: TerminalState.TERMINAL_STATE_RUNNING,
        resourceHistory: [_usage(100, 100, 1), _usage(200, 200, 2)],
        resources: _usage(300, 300, 3),
      ),
      TerminalInfo(
        state: TerminalState.TERMINAL_STATE_RUNNING,
        resourceHistory: [_usage(400, 400, 2)],
        resources: _usage(500, 500, 3),
      ),
    ];

    final series = terminalAggregateResourceSeries(terminals);

    expect(series.cpuPercent, [1, 6, 8]);
    expect(series.memoryBytes, [100, 600, 800]);
  });

  testWidgets('keeps per-terminal resource metrics compact on a narrow row', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(180, 80);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final terminal = TerminalInfo(
      ref: TerminalRef(endpointId: 'studio', terminalId: 'build'),
      resources: _usage(420, 224 * 1024 * 1024, 3),
    );
    var opened = false;

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: Scaffold(
          body: Align(
            alignment: Alignment.topLeft,
            child: TerminalResourceInlineSummary(
              terminal: terminal,
              onTap: () => opened = true,
            ),
          ),
        ),
      ),
    );

    expect(find.text('CPU'), findsOneWidget);
    expect(find.text('4.2%'), findsOneWidget);
    expect(find.text('RAM'), findsOneWidget);
    expect(find.text('224.0 MiB'), findsOneWidget);
    expect(
      tester.getSize(find.byKey(const ValueKey('resources-build'))).width,
      180,
    );
    expect(
      find.byKey(const ValueKey('terminal-resource-sparkline-cpu-build')),
      findsOneWidget,
    );
    expect(
      find.byKey(const ValueKey('terminal-resource-sparkline-memory-build')),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);

    await tester.tap(find.byKey(const ValueKey('resources-build')));
    expect(opened, isTrue);
  });

  testWidgets('shows compact metrics and opens snapshot details', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 720);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final terminal = TerminalInfo(
      ref: TerminalRef(endpointId: 'studio', terminalId: 'build'),
      name: 'Build agent',
      resourceHistory: [
        _usage(120, 180 * 1024 * 1024, 1),
        _usage(350, 210 * 1024 * 1024, 2),
      ],
      resources: _usage(420, 224 * 1024 * 1024, 3),
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: Builder(
          builder: (context) => Scaffold(
            body: Padding(
              padding: const EdgeInsets.all(12),
              child: TerminalResourceSummary(
                terminal: terminal,
                onTap: () => showTerminalResourceDetails(context, terminal),
              ),
            ),
          ),
        ),
      ),
    );

    expect(find.text('4.2%'), findsOneWidget);
    expect(find.text('224.0 MiB'), findsOneWidget);
    expect(
      tester.getSize(find.byKey(const ValueKey('resources-build'))).height,
      greaterThanOrEqualTo(48),
    );
    expect(tester.takeException(), isNull);

    await tester.tap(find.byKey(const ValueKey('resources-build')));
    await tester.pumpAndSettle();

    final barrier = tester
        .widgetList<ModalBarrier>(find.byType(ModalBarrier))
        .where((value) => value.color != null)
        .last;
    expect(barrier.color!.a, lessThan(0.3));

    expect(find.text('Resource snapshots · 3 samples'), findsOneWidget);
    expect(find.text('CPU history'), findsOneWidget);
    expect(find.text('Memory history'), findsOneWidget);
    expect(
      find.textContaining('latest terminal inventory snapshot'),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('opens CPU and memory sampling curves from the resource strip', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(393, 852);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final terminals = [
      TerminalInfo(
        state: TerminalState.TERMINAL_STATE_RUNNING,
        resourceHistory: [_usage(120, 180 * 1024 * 1024, 1)],
        resources: _usage(420, 224 * 1024 * 1024, 2),
      ),
    ];

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: Scaffold(body: TerminalResourceStrip(terminals: terminals)),
      ),
    );

    await tester.tap(find.byKey(const ValueKey('terminal-resource-strip-cpu')));
    await tester.pumpAndSettle();
    expect(find.text('CPU sampling'), findsWidgets);
    expect(find.text('2 aggregate samples'), findsOneWidget);

    await tester.tap(find.text('Memory').last);
    await tester.pumpAndSettle();
    expect(find.text('Memory sampling'), findsWidgets);
    expect(tester.takeException(), isNull);
  });
}

TerminalResourceUsage _usage(int cpuX100, int memoryBytes, int sampledAt) =>
    TerminalResourceUsage(
      pid: 4242,
      cpuPercentX100: cpuX100,
      memoryBytes: Int64(memoryBytes),
      sampledAtUnixNano: Int64(sampledAt),
    );
