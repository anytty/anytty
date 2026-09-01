import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:anytty_native/src/app/anytty_app.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/app/startup_recovery.dart';

void main() {
  testWidgets('recovers startup without exposing private failure details', (
    tester,
  ) async {
    anyttyRouter.go('/');
    const secret = 'endpoint-secret /Users/private/registry';
    final diagnostics = StartupDiagnosticsRecorder()
      ..beginRuntimeAttempt()
      ..recordFailure(StartupStage.registry, StateError(secret));
    final reset = _FakeStartupReset();
    var registryLoads = 0;
    String? clipboardText;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          clipboardText =
              (call.arguments as Map<Object?, Object?>)['text'] as String?;
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith((ref) async {
            registryLoads += 1;
            throw StateError(secret);
          }),
          startupDiagnosticsProvider.overrideWithValue(diagnostics),
          startupLocalResetProvider.overrideWithValue(reset),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('endpoint-secret'), findsNothing);
    expect(find.textContaining('/Users/private'), findsNothing);
    expect(
      find.text('Saved local connection data could not be loaded.'),
      findsOneWidget,
    );
    for (final finder in [
      find.widgetWithText(FilledButton, 'Retry startup'),
      find.widgetWithText(OutlinedButton, 'Copy redacted diagnostics'),
      find.widgetWithText(TextButton, 'Reset saved devices'),
    ]) {
      expect(tester.getSize(finder).height, greaterThanOrEqualTo(48));
    }

    await tester.tap(find.text('Copy redacted diagnostics'));
    await tester.pump();
    expect(clipboardText, isNotNull);
    expect(clipboardText, isNot(contains(secret)));

    await tester.tap(find.text('Retry startup'));
    await tester.pumpAndSettle();
    expect(registryLoads, greaterThanOrEqualTo(2));

    await tester.tap(find.text('Reset saved devices'));
    await tester.pumpAndSettle();
    expect(find.text('Reset saved devices?'), findsOneWidget);
    await tester.tap(find.widgetWithText(FilledButton, 'Reset'));
    await tester.pumpAndSettle();
    expect(reset.calls, 1);
    expect(find.textContaining('endpoint-secret'), findsNothing);
  });
}

final class _FakeStartupReset implements StartupLocalReset {
  int calls = 0;

  @override
  Future<void> reset() async {
    calls += 1;
  }
}
