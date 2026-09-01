import 'dart:collection';
import 'dart:convert';

import '../generated/proto/apipb/common.pb.dart';
import '../generated/proto/bindingpb/client_binding.pb.dart';

enum RuntimeDiagnosticEventKind {
  sessionOpened,
  sessionClosed,
  connectionPhaseChanged,
  streamOpened,
  streamClosed,
  outputSyncLost,
}

abstract interface class RuntimeDiagnosticsSink {
  void recordRenderLatency({
    required EndpointSessionStamp session,
    required Duration latency,
  });
}

final class RuntimeDiagnosticsRecorder implements RuntimeDiagnosticsSink {
  static const maximumEvents = 64;

  final Queue<({DateTime at, RuntimeDiagnosticEventKind kind, int generation})>
  _events = Queue();
  int _renderSamples = 0;
  int _renderLatencyTotalMicros = 0;
  int _latestRenderLatencyMicros = 0;
  int _maximumRenderLatencyMicros = 0;
  int _droppedBytes = 0;
  int _parserEpoch = 0;

  void recordEvent(RuntimeDiagnosticEventKind kind, {int generation = 0}) {
    if (_events.length == maximumEvents) _events.removeFirst();
    _events.addLast((
      at: DateTime.now().toUtc(),
      kind: kind,
      generation: generation,
    ));
  }

  @override
  void recordRenderLatency({
    required EndpointSessionStamp session,
    required Duration latency,
  }) {
    final micros = latency.inMicroseconds.clamp(0, 60 * 1000 * 1000);
    _renderSamples += 1;
    _renderLatencyTotalMicros += micros;
    _latestRenderLatencyMicros = micros;
    if (micros > _maximumRenderLatencyMicros) {
      _maximumRenderLatencyMicros = micros;
    }
  }

  void recordOutputSyncLoss({required int droppedBytes, int parserEpoch = 0}) {
    _droppedBytes += droppedBytes.clamp(0, 1 << 40);
    if (parserEpoch > _parserEpoch) _parserEpoch = parserEpoch;
    recordEvent(RuntimeDiagnosticEventKind.outputSyncLost);
  }

  String buildRedactedReport({
    required int activeOperationHandles,
    required int activeSessionHandles,
    required int activeStreamHandles,
    EndpointSessionStamp? session,
    EndpointSupervisorProjection? supervisor,
    DateTime? capturedAt,
  }) {
    final activeGeneration = session?.generation.toInt() ?? 0;
    final averageRenderMicros = _renderSamples == 0
        ? 0
        : _renderLatencyTotalMicros ~/ _renderSamples;
    final report = <String, Object?>{
      'schema': 1,
      'redacted': true,
      'captured_at': (capturedAt ?? DateTime.now()).toUtc().toIso8601String(),
      'session': <String, Object?>{
        'endpoint': session == null ? null : '<redacted>',
        'route': session == null ? null : '<redacted>',
        'generation': activeGeneration,
      },
      'supervisor': supervisor == null
          ? null
          : <String, Object?>{
              'phase': supervisor.phase,
              'control_revision': supervisor.controlRevision.toInt(),
              'attempt': supervisor.attemptId.toInt(),
              'generation': supervisor.hasSession()
                  ? supervisor.session.generation.toInt()
                  : 0,
              'error_code': supervisor.errorCode,
              'probe_count': supervisor.probeCount.toInt(),
              'dial_count': supervisor.dialCount.toInt(),
              'backoff_count': supervisor.backoffCount.toInt(),
            },
      'resources': <String, int>{
        'operation_handles': activeOperationHandles,
        'session_handles': activeSessionHandles,
        'stream_handles': activeStreamHandles,
      },
      'render': <String, num>{
        'samples': _renderSamples,
        'latest_latency_ms': _milliseconds(_latestRenderLatencyMicros),
        'average_latency_ms': _milliseconds(averageRenderMicros),
        'maximum_latency_ms': _milliseconds(_maximumRenderLatencyMicros),
      },
      'output': <String, int>{
        'dropped_bytes': _droppedBytes,
        'parser_epoch': _parserEpoch,
      },
      'recent_events': [
        for (final event in _events)
          <String, Object>{
            'at': event.at.toIso8601String(),
            'kind': event.kind.name,
            'generation': event.generation,
          },
      ],
    };
    return const JsonEncoder.withIndent('  ').convert(report);
  }
}

double _milliseconds(int microseconds) => microseconds / 1000;
