import 'dart:convert';

import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/runtime_diagnostics.dart';

void main() {
  test(
    'builds an allowlisted report without endpoint or route identifiers',
    () {
      final recorder = RuntimeDiagnosticsRecorder()
        ..recordRenderLatency(
          session: EndpointSessionStamp(
            endpointId: 'private-endpoint',
            routeId: 'private-route',
            generation: Int64(12),
          ),
          latency: const Duration(milliseconds: 18),
        )
        ..recordOutputSyncLoss(droppedBytes: 4096, parserEpoch: 7)
        ..recordOutputSyncLoss(droppedBytes: 512, parserEpoch: 6);

      final report = recorder.buildRedactedReport(
        activeOperationHandles: 3,
        activeSessionHandles: 2,
        activeStreamHandles: 1,
        capturedAt: DateTime.utc(2026, 8, 31, 10),
        session: EndpointSessionStamp(
          endpointId: 'private-endpoint',
          routeId: 'private-route',
          generation: Int64(12),
        ),
        supervisor: EndpointSupervisorProjection(
          endpointId: 'private-endpoint',
          phase: 'ready',
          controlRevision: Int64(22),
          attemptId: Int64(9),
          session: EndpointSessionStamp(
            endpointId: 'private-endpoint',
            routeId: 'private-route',
            generation: Int64(12),
          ),
          errorCode: 'unavailable',
          message: 'secret transport error',
          probeCount: Int64(2),
          dialCount: Int64(3),
          backoffCount: Int64(1),
        ),
      );

      expect(report, isNot(contains('private-endpoint')));
      expect(report, isNot(contains('private-route')));
      expect(report, isNot(contains('secret transport error')));
      final decoded = jsonDecode(report) as Map<String, Object?>;
      expect(decoded['redacted'], isTrue);
      expect(decoded['session'], {
        'endpoint': '<redacted>',
        'route': '<redacted>',
        'generation': 12,
      });
      expect(decoded['resources'], {
        'operation_handles': 3,
        'session_handles': 2,
        'stream_handles': 1,
      });
      expect(decoded['output'], {'dropped_bytes': 4608, 'parser_epoch': 7});
      expect(
        (decoded['render'] as Map<String, Object?>)['latest_latency_ms'],
        18,
      );
      expect((decoded['supervisor'] as Map<String, Object?>)['attempt'], 9);
    },
  );

  test('retains only the newest bounded lifecycle events', () {
    final recorder = RuntimeDiagnosticsRecorder();
    for (var generation = 1; generation <= 80; generation += 1) {
      recorder.recordEvent(
        RuntimeDiagnosticEventKind.connectionPhaseChanged,
        generation: generation,
      );
    }

    final decoded = jsonDecode(
      recorder.buildRedactedReport(
        activeOperationHandles: 0,
        activeSessionHandles: 0,
        activeStreamHandles: 0,
      ),
    ) as Map<String, Object?>;
    final events = decoded['recent_events'] as List<Object?>;
    expect(events, hasLength(RuntimeDiagnosticsRecorder.maximumEvents));
    expect((events.first as Map<String, Object?>)['generation'], 17);
    expect((events.last as Map<String, Object?>)['generation'], 80);
  });
}
