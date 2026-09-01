import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';

import 'package:anytty_native/src/app/startup_recovery.dart';

void main() {
  test('keeps only bounded allowlisted startup diagnostics', () {
    final recorder = StartupDiagnosticsRecorder();
    for (var index = 0; index < 12; index += 1) {
      recorder.beginRuntimeAttempt();
      recorder.recordFailure(
        StartupStage.registry,
        StateError('private-endpoint-$index /Users/private/config'),
      );
    }

    final report = recorder.buildRedactedReport(
      capturedAt: DateTime.utc(2026, 8, 31, 12),
    );
    expect(report, isNot(contains('private-endpoint')));
    expect(report, isNot(contains('/Users/private')));
    final decoded = jsonDecode(report) as Map<String, Object?>;
    expect(decoded['redacted'], isTrue);
    expect(decoded['startup_attempts'], 12);
    expect(decoded['last_failure'], 'localData');
    expect(
      decoded['recent_events'],
      hasLength(StartupDiagnosticsRecorder.maximumEvents),
    );
    expect(
      recorder.safeFailureMessage,
      'Saved local connection data could not be loaded.',
    );
  });
}
