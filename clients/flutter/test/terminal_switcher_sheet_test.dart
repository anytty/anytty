import 'dart:async';

import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_switcher_sheet.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('opens the terminal switcher as a bottom drawer', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.dark),
        home: Builder(
          builder: (context) => Scaffold(
            body: Center(
              child: TextButton(
                onPressed: () => showAnyttyTerminalSwitcher(
                  context: context,
                  endpoints: [
                    TerminalSwitcherEndpoint(
                      endpointId: 'local',
                      label: 'Studio',
                      current: true,
                      terminals: [_terminal('local', 'shell', name: 'Shell')],
                    ),
                  ],
                  loadTerminals: (_) async => const [],
                ),
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    final tray = find.byType(TerminalSwitcherSheet);
    expect(tray, findsOneWidget);
    expect(tester.getTopLeft(tray).dy, greaterThan(100));
    expect(
      tester.getBottomRight(tray).dy,
      moreOrLessEquals(
        tester.view.physicalSize.height / tester.view.devicePixelRatio,
        epsilon: 1,
      ),
    );
    expect(find.text('Shell'), findsOneWidget);
  });

  testWidgets('loads another device only when expanded and returns selection', (
    tester,
  ) async {
    var calls = 0;
    final remote = _terminal('remote', 'build', name: 'Build agent');

    await tester.pumpWidget(
      _Harness(
        child: TerminalSwitcherSheet(
          endpoints: [
            TerminalSwitcherEndpoint(
              endpointId: 'local',
              label: 'Studio',
              current: true,
              terminals: [_terminal('local', 'shell', name: 'Shell')],
              activeTerminalId: 'shell',
            ),
            const TerminalSwitcherEndpoint(
              endpointId: 'remote',
              label: 'Server',
              current: false,
            ),
          ],
          loadTerminals: (endpointId) async {
            calls += 1;
            expect(endpointId, 'remote');
            return [remote];
          },
        ),
      ),
    );

    expect(find.text('Shell'), findsOneWidget);
    expect(find.text('Build agent'), findsNothing);
    expect(calls, 0);

    await tester.tap(find.byKey(const ValueKey('endpoint-remote')));
    await tester.pumpAndSettle();

    expect(calls, 1);
    expect(find.text('Build agent'), findsOneWidget);

    await tester.tap(find.text('Build agent'));
    await tester.pumpAndSettle();

    final selection = _Harness.selection;
    expect(selection?.endpointId, 'remote');
    expect(selection?.endpointLabel, 'Server');
    expect(selection?.terminalId, 'build');
  });

  testWidgets('keeps a loaded inventory cached across collapse and expansion', (
    tester,
  ) async {
    var calls = 0;
    await tester.pumpWidget(
      _Harness(
        child: TerminalSwitcherSheet(
          endpoints: const [
            TerminalSwitcherEndpoint(
              endpointId: 'local',
              label: 'Studio',
              current: true,
              terminals: [],
            ),
            TerminalSwitcherEndpoint(
              endpointId: 'remote',
              label: 'Server',
              current: false,
            ),
          ],
          loadTerminals: (_) async {
            calls += 1;
            return [_terminal('remote', 'logs', name: 'Logs')];
          },
        ),
      ),
    );

    final remoteGroup = find.byKey(const ValueKey('endpoint-remote'));
    await tester.tap(remoteGroup);
    await tester.pumpAndSettle();
    await tester.tap(remoteGroup);
    await tester.pumpAndSettle();
    await tester.tap(remoteGroup);
    await tester.pumpAndSettle();

    expect(calls, 1);
    expect(find.text('Logs'), findsOneWidget);
  });

  testWidgets('shows a retry action after a device inventory error', (
    tester,
  ) async {
    var calls = 0;
    final completer = Completer<List<TerminalInfo>>();
    await tester.pumpWidget(
      _Harness(
        child: TerminalSwitcherSheet(
          endpoints: const [
            TerminalSwitcherEndpoint(
              endpointId: 'remote',
              label: 'Server',
              current: false,
            ),
          ],
          loadTerminals: (_) {
            calls += 1;
            if (calls == 1) return Future.error(StateError('offline'));
            return completer.future;
          },
        ),
      ),
    );

    await tester.tap(find.byKey(const ValueKey('endpoint-remote')));
    await tester.pumpAndSettle();
    expect(find.text('Could not load terminals'), findsOneWidget);

    await tester.tap(find.text('Retry'));
    await tester.pump();
    expect(calls, 2);

    completer.complete([_terminal('remote', 'shell', name: 'Recovered')]);
    await tester.pumpAndSettle();
    expect(find.text('Recovered'), findsOneWidget);
  });

  testWidgets('filters the switcher by state and terminal search', (
    tester,
  ) async {
    await tester.pumpWidget(
      _Harness(
        child: TerminalSwitcherSheet(
          endpoints: [
            TerminalSwitcherEndpoint(
              endpointId: 'local',
              label: 'Studio',
              current: true,
              terminals: [
                _terminal('local', 'shell', name: 'Shell'),
                _terminal(
                  'local',
                  'old-build',
                  name: 'Old build',
                  state: TerminalState.TERMINAL_STATE_EXITED,
                ),
              ],
            ),
          ],
          loadTerminals: (_) async => const [],
        ),
      ),
    );

    expect(find.text('Shell'), findsOneWidget);
    expect(find.text('Old build'), findsNothing);

    await tester.tap(find.byKey(const ValueKey('switcher-filter-all')));
    await tester.pump();
    expect(find.text('Old build'), findsOneWidget);

    await tester.enterText(
      find.byKey(const ValueKey('terminal-switcher-search-field')),
      'OdBl',
    );
    await tester.pump();
    expect(find.text('Shell'), findsNothing);
    expect(find.text('Old build'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('switcher-filter-running')));
    await tester.pump();
    expect(find.text('Old build'), findsNothing);
    expect(find.text('No matching terminals'), findsOneWidget);
  });

  testWidgets('loads unopened devices when global search starts', (
    tester,
  ) async {
    var calls = 0;
    await tester.pumpWidget(
      _Harness(
        child: TerminalSwitcherSheet(
          endpoints: const [
            TerminalSwitcherEndpoint(
              endpointId: 'local',
              label: 'Studio',
              current: true,
              terminals: [],
            ),
            TerminalSwitcherEndpoint(
              endpointId: 'remote',
              label: 'Server',
              current: false,
            ),
          ],
          loadTerminals: (_) async {
            calls += 1;
            return [_terminal('remote', 'deploy', name: 'Deploy worker')];
          },
        ),
      ),
    );

    await tester.enterText(
      find.byKey(const ValueKey('terminal-switcher-search-field')),
      'deploy',
    );
    await tester.pumpAndSettle();

    expect(calls, 1);
    expect(find.text('Deploy worker'), findsOneWidget);
  });
}

final class _Harness extends StatelessWidget {
  const _Harness({required this.child});

  final Widget child;
  static TerminalSwitcherSelection? selection;

  @override
  Widget build(BuildContext context) {
    selection = null;
    return MaterialApp(
      theme: anyttyTheme(Brightness.light),
      home: Builder(
        builder: (context) => Scaffold(
          body: Navigator(
            onGenerateRoute: (_) => MaterialPageRoute<void>(
              builder: (context) => Builder(
                builder: (context) => PopScope<TerminalSwitcherSelection>(
                  onPopInvokedWithResult: (didPop, result) {
                    if (didPop) selection = result;
                  },
                  child: child,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

TerminalInfo _terminal(
  String endpointId,
  String terminalId, {
  required String name,
  TerminalState state = TerminalState.TERMINAL_STATE_RUNNING,
}) => TerminalInfo(
  ref: TerminalRef(endpointId: endpointId, terminalId: terminalId),
  name: name,
  state: state,
  cwd: '/workspace/$terminalId',
  size: TerminalSize(cols: 96, rows: 28),
);
