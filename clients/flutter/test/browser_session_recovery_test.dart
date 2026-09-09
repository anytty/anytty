import 'dart:async';

import 'package:anytty_native/src/features/browser/domain/browser_session_recovery.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('immediate failure yields and retries with bounded backoff', (
    tester,
  ) async {
    var attempts = 0;
    var errors = 0;
    final recovery = BrowserSessionRecovery(
      recover: () async {
        attempts++;
        throw StateError('offline');
      },
      needsRecovery: () => true,
      onError: (_, _) => errors++,
    )..start();
    await tester.pump(Duration.zero);
    expect(attempts, 1);
    await tester.pump(const Duration(milliseconds: 249));
    expect(attempts, 1);
    await tester.pump(const Duration(milliseconds: 1));
    expect(attempts, 2);
    for (final milliseconds in [500, 1000, 2000, 4000, 5000, 5000]) {
      final before = attempts;
      await tester.pump(Duration(milliseconds: milliseconds));
      expect(attempts, before + 1);
    }
    expect(errors, attempts);
    recovery.cancel();
    await tester.pump(const Duration(minutes: 1));
    expect(attempts, 8);
  });

  testWidgets(
    'coalesces attempts and cancellation cannot restart an in-flight retry',
    (tester) async {
      final pending = Completer<void>();
      var attempts = 0;
      final recovery = BrowserSessionRecovery(
        recover: () {
          attempts++;
          return pending.future;
        },
        needsRecovery: () => true,
        onError: (_, _) => fail('unexpected error'),
      )..start();
      recovery.start();
      await tester.pump(Duration.zero);
      await tester.pump(const Duration(seconds: 10));
      expect(attempts, 1);
      recovery.cancel();
      pending.complete();
      await tester.pump(Duration.zero);
      await tester.pump(const Duration(seconds: 10));
      expect(attempts, 1);
      expect(recovery.isRunning, isFalse);
    },
  );

  testWidgets('stops on recovery or endpoint change', (tester) async {
    var needed = true;
    var attempts = 0;
    final recovery = BrowserSessionRecovery(
      recover: () async {
        attempts++;
        needed = false;
      },
      needsRecovery: () => needed,
      onError: (_, _) => fail('unexpected error'),
    )..start();
    await tester.pump(Duration.zero);
    await tester.pump(const Duration(seconds: 10));
    expect(attempts, 1);
    expect(recovery.isRunning, isFalse);
    recovery.cancel();

    final switched = BrowserSessionRecovery(
      recover: () async {
        attempts++;
      },
      needsRecovery: () => false,
      onError: (_, _) => fail('unexpected error'),
    )..start();
    await tester.pump(Duration.zero);
    expect(attempts, 1);
    expect(switched.isRunning, isFalse);
  });
}
