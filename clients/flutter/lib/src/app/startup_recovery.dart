import 'dart:collection';
import 'dart:convert';

import 'package:flutter/services.dart';

import '../native/anytty_client_engine.dart';
import '../native/endpoint_registry_platform.dart';

enum StartupStage { runtime, registry }

enum StartupFailureKind { nativeRuntime, localData, platformService, unknown }

enum StartupDiagnosticEventKind {
  runtimeAttempt,
  runtimeReady,
  runtimeFailed,
  registryReady,
  registryFailed,
  localReset,
}

final class StartupDiagnosticsRecorder {
  static const maximumEvents = 8;

  final Queue<
    ({
      DateTime at,
      StartupDiagnosticEventKind kind,
      StartupFailureKind? failure,
    })
  >
  _events = Queue();
  int _attempts = 0;
  StartupFailureKind _lastFailure = StartupFailureKind.unknown;

  int get attempts => _attempts;
  StartupFailureKind get lastFailure => _lastFailure;

  String get safeFailureMessage => switch (_lastFailure) {
    StartupFailureKind.nativeRuntime =>
      'The local terminal engine could not start.',
    StartupFailureKind.localData =>
      'Saved local connection data could not be loaded.',
    StartupFailureKind.platformService =>
      'A required device service is unavailable.',
    StartupFailureKind.unknown => 'AnyTTY could not finish starting.',
  };

  void beginRuntimeAttempt() {
    _attempts += 1;
    _record(StartupDiagnosticEventKind.runtimeAttempt);
  }

  void recordReady(StartupStage stage) {
    _record(
      stage == StartupStage.runtime
          ? StartupDiagnosticEventKind.runtimeReady
          : StartupDiagnosticEventKind.registryReady,
    );
  }

  void recordFailure(StartupStage stage, Object error) {
    _lastFailure = _classify(error);
    _record(
      stage == StartupStage.runtime
          ? StartupDiagnosticEventKind.runtimeFailed
          : StartupDiagnosticEventKind.registryFailed,
      failure: _lastFailure,
    );
  }

  void recordLocalReset() => _record(StartupDiagnosticEventKind.localReset);

  String buildRedactedReport({DateTime? capturedAt}) {
    return const JsonEncoder.withIndent('  ').convert(<String, Object?>{
      'schema': 1,
      'redacted': true,
      'captured_at': (capturedAt ?? DateTime.now()).toUtc().toIso8601String(),
      'startup_attempts': _attempts,
      'last_failure': _lastFailure.name,
      'recent_events': [
        for (final event in _events)
          <String, Object?>{
            'at': event.at.toIso8601String(),
            'kind': event.kind.name,
            'failure': event.failure?.name,
          },
      ],
    });
  }

  void _record(StartupDiagnosticEventKind kind, {StartupFailureKind? failure}) {
    if (_events.length == maximumEvents) _events.removeFirst();
    _events.addLast((at: DateTime.now().toUtc(), kind: kind, failure: failure));
  }
}

abstract interface class StartupLocalReset {
  Future<void> reset();
}

final class RegistryStartupLocalReset implements StartupLocalReset {
  const RegistryStartupLocalReset([this._store]);

  final EndpointRegistryBlobStore? _store;

  @override
  Future<void> reset() =>
      (_store ?? PreferencesEndpointRegistryBlobStore()).clear();
}

StartupFailureKind _classify(Object error) {
  if (error is AnyttyNativeException) return StartupFailureKind.nativeRuntime;
  if (error is FormatException || error is StateError) {
    return StartupFailureKind.localData;
  }
  if (error is PlatformException || error is MissingPluginException) {
    return StartupFailureKind.platformService;
  }
  return StartupFailureKind.unknown;
}
