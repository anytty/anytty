import 'dart:async';
import 'dart:isolate';
import 'dart:typed_data';

import 'package:fixnum/fixnum.dart';

import '../generated/proto/apipb/application.pb.dart'
    show CommandEnvelope, ResultEnvelope_Result;
import '../generated/proto/apipb/common.pb.dart'
    show ApiError, EndpointSessionStamp;
import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'anytty_client_engine.dart';
import 'request_id.dart';
import 'runtime_diagnostics.dart';

abstract interface class AnyttyPlatformHandler {
  Future<PlatformResponse> handle(PlatformRequest request);
}

abstract interface class AnyttyEngineRuntime {
  Stream<EventEnvelope> get events;
  Stream<int> get foregroundResumes;

  EndpointDemandLease retainEndpointDemand(String endpointId);

  int command(EngineCommand command);

  int openSession(OpenSessionRequest request);

  int execute(int sessionHandle, CommandEnvelope request);

  void cancel(int operationHandle);

  void release(int handle);

  void closeSession(int sessionHandle);
}

abstract interface class AnyttyResourceStreamRuntime {
  int openResourceStream(int sessionHandle, OpenResourceStreamRequest request);

  void sendResourceStreamFrame(int streamHandle, ResourceStreamFrame frame);

  void closeResourceStream(int streamHandle);
}

final class AnyttyRuntime
    implements
        AnyttyEngineRuntime,
        AnyttyResourceStreamRuntime,
        RuntimeDiagnosticsSink {
  AnyttyRuntime._({required this._engine, required this._platform}) {
    _endpointDemand = EndpointDemandCoordinator(
      attachmentId: 'flutter-${newRequestId()}',
      replace: (snapshot) => _engine.replaceSupervisorDemand(
        Uint8List.fromList(snapshot.writeToBuffer()),
      ),
      onPublished: (endpointIds) => _endpointDemands.add(endpointIds),
    );
  }

  final AnyttyClientEngine _engine;
  final AnyttyPlatformHandler _platform;
  final StreamController<EventEnvelope> _events =
      StreamController<EventEnvelope>.broadcast(sync: true);
  final StreamController<int> _foregroundResumes =
      StreamController<int>.broadcast();
  final StreamController<List<String>> _endpointDemands =
      StreamController<List<String>>.broadcast();
  final List<ReceivePort> _ports = [];
  final List<StreamSubscription<Object?>> _subscriptions = [];
  final List<Isolate> _isolates = [];
  final RuntimeDiagnosticsRecorder _diagnostics = RuntimeDiagnosticsRecorder();
  final Set<int> _operationHandles = {};
  final Set<int> _sessionHandles = {};
  final Set<int> _streamHandles = {};
  final Map<int, int> _sessionGenerations = {};
  final Map<int, int> _streamGenerations = {};
  final Map<String, EndpointConnectionEvent> _endpointConnectionEvents = {};
  late final EndpointDemandCoordinator _endpointDemand;
  int _hostRevision = 0;
  bool _closed = false;

  static Future<AnyttyRuntime> start({
    required AnyttyPlatformHandler platform,
  }) async {
    final engine = await AnyttyClientEngine.openAsync();
    final runtime = AnyttyRuntime._(engine: engine, platform: platform);
    try {
      await runtime._startPump(_PumpKind.events);
      await runtime._startPump(_PumpKind.platform);
      return runtime;
    } catch (_) {
      await runtime.close();
      rethrow;
    }
  }

  @override
  Stream<EventEnvelope> get events => _events.stream;

  @override
  Stream<int> get foregroundResumes => _foregroundResumes.stream;

  Stream<List<String>> get endpointDemands => _endpointDemands.stream;

  List<String> get demandedEndpointIds => _endpointDemand.endpointIds;

  EndpointSupervisorSnapshot supervisorSnapshot() =>
      EndpointSupervisorSnapshot.fromBuffer(_engine.supervisorSnapshot());

  EndpointConnectionEvent? endpointConnectionEvent(String endpointId) {
    final event = _endpointConnectionEvents[endpointId.trim()];
    return event?.deepCopy();
  }

  String buildRedactedDiagnostics({
    required String endpointId,
    EndpointSessionStamp? session,
  }) {
    final normalized = endpointId.trim();
    EndpointSupervisorProjection? projection;
    if (normalized.isNotEmpty) {
      for (final candidate in supervisorSnapshot().endpoints) {
        if (candidate.endpointId == normalized) {
          projection = candidate;
          break;
        }
      }
    }
    return _diagnostics.buildRedactedReport(
      activeOperationHandles: _operationHandles.length,
      activeSessionHandles: _sessionHandles.length,
      activeStreamHandles: _streamHandles.length,
      session: session,
      supervisor: projection,
    );
  }

  @override
  void recordRenderLatency({
    required EndpointSessionStamp session,
    required Duration latency,
  }) => _diagnostics.recordRenderLatency(session: session, latency: latency);

  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) {
    _ensureOpen();
    return _endpointDemand.retain(endpointId);
  }

  void signalNetwork({required bool connected, required String reason}) {
    _signalSupervisor(connected: connected, reason: reason, foreground: false);
  }

  void suspendForeground({required bool connected}) {
    _signalSupervisor(
      connected: connected,
      reason: 'background',
      foreground: false,
    );
  }

  Future<void> resumeForeground({required bool connected}) async {
    final revision = _signalSupervisor(
      connected: connected,
      reason: 'foreground_resume',
      foreground: true,
    );
    final engineHandle = _engine.handle;
    try {
      await Isolate.run(() {
        AnyttyClientEngine.attach(engineHandle)
            .waitSupervisorReady(timeoutMillis: 1200);
      });
    } finally {
      if (!_closed) _foregroundResumes.add(revision);
    }
  }

  @override
  int command(EngineCommand command) {
    _ensureOpen();
    return _trackOperation(
      _engine.command(Uint8List.fromList(command.writeToBuffer())),
    );
  }

  @override
  int openSession(OpenSessionRequest request) {
    _ensureOpen();
    return _trackOperation(
      _engine.openSession(Uint8List.fromList(request.writeToBuffer())),
    );
  }

  @override
  int execute(int sessionHandle, CommandEnvelope request) {
    _ensureOpen();
    return _trackOperation(
      _engine.execute(
        sessionHandle,
        Uint8List.fromList(request.writeToBuffer()),
      ),
    );
  }

  @override
  int openResourceStream(int sessionHandle, OpenResourceStreamRequest request) {
    _ensureOpen();
    final handle = _engine.openResourceStream(
      sessionHandle,
      Uint8List.fromList(request.writeToBuffer()),
    );
    _streamHandles.add(handle);
    final generation = _sessionGenerations[sessionHandle] ?? 0;
    _streamGenerations[handle] = generation;
    _diagnostics.recordEvent(
      RuntimeDiagnosticEventKind.streamOpened,
      generation: generation,
    );
    return handle;
  }

  @override
  void sendResourceStreamFrame(int streamHandle, ResourceStreamFrame frame) {
    _ensureOpen();
    _engine.sendResourceStreamFrame(
      streamHandle,
      Uint8List.fromList(frame.writeToBuffer()),
    );
  }

  @override
  void closeResourceStream(int streamHandle) {
    _ensureOpen();
    _engine.closeResourceStream(streamHandle);
  }

  @override
  void cancel(int operationHandle) => _engine.cancel(operationHandle);

  @override
  void release(int handle) {
    _engine.release(handle);
    _operationHandles.remove(handle);
    _sessionHandles.remove(handle);
    _streamHandles.remove(handle);
    _sessionGenerations.remove(handle);
    _streamGenerations.remove(handle);
  }

  @override
  void closeSession(int sessionHandle) => _engine.closeSession(sessionHandle);

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    try {
      _endpointDemand.close();
    } finally {
      _engine.close();
      _operationHandles.clear();
      _sessionHandles.clear();
      _streamHandles.clear();
      _sessionGenerations.clear();
      _streamGenerations.clear();
    }
    for (final subscription in _subscriptions) {
      await subscription.cancel();
    }
    for (final port in _ports) {
      port.close();
    }
    for (final isolate in _isolates) {
      isolate.kill(priority: Isolate.immediate);
    }
    await _events.close();
    await _foregroundResumes.close();
    await _endpointDemands.close();
  }

  Future<void> _startPump(_PumpKind kind) async {
    final port = ReceivePort('${kind.name}-pump');
    _ports.add(port);
    final ready = Completer<void>();
    SendPort? pumpControl;
    final subscription = port.listen((message) {
      if (message is _PumpReady) {
        pumpControl = message.control;
        if (!ready.isCompleted) ready.complete();
        return;
      }
      if (message is _PumpFailure) {
        if (!ready.isCompleted) {
          ready.completeError(StateError(message.message));
        } else if (!_closed) {
          _events.addError(StateError(message.message));
        }
        return;
      }
      if (message is! TransferableTypedData || _closed) return;
      final control = pumpControl;
      if (control == null) return;
      final bytes = message.materialize().asUint8List();
      if (kind == _PumpKind.events) {
        try {
          final event = EventEnvelope.fromBuffer(bytes);
          _acceptEngineEvent(event);
          _events.add(event);
        } finally {
          control.send(_PumpSignal.next);
        }
      } else {
        unawaited(_dispatchPlatformRequest(bytes, control));
      }
    });
    _subscriptions.add(subscription);
    final isolate = await Isolate.spawn(
      _runNativePump,
      _PumpStart(port.sendPort, _engine.handle, kind),
      debugName: 'anytty-${kind.name}',
    );
    _isolates.add(isolate);
    await ready.future;
  }

  Future<void> _dispatchPlatformRequest(
    Uint8List bytes,
    SendPort control,
  ) async {
    try {
      await _handlePlatform(bytes);
    } catch (error, stackTrace) {
      if (!_closed) _events.addError(error, stackTrace);
    } finally {
      control.send(_PumpSignal.next);
    }
  }

  Future<void> _handlePlatform(Uint8List bytes) async {
    if (_closed) return;
    final request = PlatformRequest.fromBuffer(bytes);
    final response = await _platform.handle(request);
    if (_closed) return;
    _engine.completePlatformRequest(
      Uint8List.fromList(response.writeToBuffer()),
    );
  }

  void _ensureOpen() {
    if (_closed) throw StateError('AnyTTY runtime is closed');
  }

  int _trackOperation(int handle) {
    _operationHandles.add(handle);
    return handle;
  }

  void _acceptEngineEvent(EventEnvelope event) {
    switch (event.whichEvent()) {
      case EventEnvelope_Event.openSession:
        final result = event.openSession;
        if (result.hasError()) _recordApiError(result.error);
        if (result.sessionHandle.toInt() > 0 && result.hasSession()) {
          final handle = result.sessionHandle.toInt();
          final generation = result.session.generation.toInt();
          _sessionHandles.add(handle);
          _sessionGenerations[handle] = generation;
          _diagnostics.recordEvent(
            RuntimeDiagnosticEventKind.sessionOpened,
            generation: generation,
          );
        }
      case EventEnvelope_Event.execute:
        final result = event.execute;
        if (result.hasError()) _recordApiError(result.error);
        if (result.hasResult() &&
            result.result.whichResult() == ResultEnvelope_Result.error) {
          _recordApiError(result.result.error);
        }
      case EventEnvelope_Event.sessionClosed:
        final closed = event.sessionClosed;
        if (closed.hasError()) _recordApiError(closed.error);
        _diagnostics.recordEvent(
          RuntimeDiagnosticEventKind.sessionClosed,
          generation: closed.hasSession()
              ? closed.session.generation.toInt()
              : (_sessionGenerations[closed.sessionHandle.toInt()] ?? 0),
        );
      case EventEnvelope_Event.resourceStreamFrame:
        final frame = event.resourceStreamFrame;
        if (frame.type ==
            ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_PTY_SYNC_LOST) {
          try {
            final syncLost = PTYStreamSyncLost.fromBuffer(frame.payload);
            _diagnostics.recordOutputSyncLoss(
              droppedBytes: syncLost.droppedBytes.toInt(),
            );
          } catch (_) {
            // Diagnostics must never interfere with resource delivery.
          }
        }
      case EventEnvelope_Event.resourceStreamClosed:
        final closed = event.resourceStreamClosed;
        if (closed.hasError()) _recordApiError(closed.error);
        _diagnostics.recordEvent(
          RuntimeDiagnosticEventKind.streamClosed,
          generation: _streamGenerations[closed.streamHandle.toInt()] ?? 0,
        );
      case EventEnvelope_Event.endpointConnection:
        final connection = event.endpointConnection;
        if (connection.endpointId.trim().isNotEmpty) {
          _endpointConnectionEvents[connection.endpointId.trim()] = connection
              .deepCopy();
        }
        if (connection.hasError()) _recordApiError(connection.error);
        _diagnostics.recordEvent(
          RuntimeDiagnosticEventKind.connectionPhaseChanged,
          generation: connection.hasSession()
              ? connection.session.generation.toInt()
              : 0,
        );
      default:
        break;
    }
  }

  void _recordApiError(ApiError error) {
    if (!error.hasOutputSyncLost()) return;
    _diagnostics.recordOutputSyncLoss(
      droppedBytes: error.outputSyncLost.droppedBytes.toInt(),
      parserEpoch: error.outputSyncLost.parserEpoch.toInt(),
    );
  }

  int _signalSupervisor({
    required bool connected,
    required String reason,
    required bool foreground,
  }) {
    _ensureOpen();
    _hostRevision += 1;
    _engine.signalSupervisor(
      Uint8List.fromList(
        EndpointSupervisorHostSignal(
          revision: Int64(_hostRevision),
          connected: connected,
          reason: reason,
          foreground: foreground,
        ).writeToBuffer(),
      ),
    );
    return _hostRevision;
  }
}

final class EndpointDemandLease {
  EndpointDemandLease(this._release);

  final void Function() _release;
  bool _released = false;

  void release() {
    if (_released) return;
    _release();
    _released = true;
  }
}

final class EndpointDemandCoordinator {
  EndpointDemandCoordinator({
    required this.attachmentId,
    required this.replace,
    this.onPublished,
  });

  final String attachmentId;
  final void Function(EndpointSupervisorDemandSnapshot) replace;
  final void Function(List<String> endpointIds)? onPublished;
  final Map<String, int> _references = {};
  int _revision = 0;
  bool _closed = false;

  List<String> get endpointIds =>
      List<String>.unmodifiable(_references.keys.toList()..sort());

  EndpointDemandLease retain(String endpointId) {
    if (_closed) throw StateError('Endpoint demand is closed');
    final normalized = endpointId.trim();
    if (normalized.isEmpty) {
      throw ArgumentError.value(endpointId, 'endpointId', 'must not be empty');
    }
    final previous = _references[normalized] ?? 0;
    _references[normalized] = previous + 1;
    if (previous == 0) {
      try {
        _publish();
      } catch (_) {
        _references.remove(normalized);
        rethrow;
      }
    }
    return EndpointDemandLease(() => _release(normalized));
  }

  void close() {
    if (_closed) return;
    try {
      if (_references.isNotEmpty) {
        _references.clear();
        _publish();
      }
    } finally {
      _closed = true;
    }
  }

  void _release(String endpointId) {
    if (_closed) return;
    final previous = _references[endpointId];
    if (previous == null) return;
    if (previous > 1) {
      _references[endpointId] = previous - 1;
      return;
    }
    _references.remove(endpointId);
    try {
      _publish();
    } catch (_) {
      _references[endpointId] = 1;
      rethrow;
    }
  }

  void _publish() {
    _revision += 1;
    final endpointIds = _references.keys.toList()..sort();
    replace(
      EndpointSupervisorDemandSnapshot(
        attachmentId: attachmentId,
        demandRevision: Int64(_revision),
        endpoints: endpointIds.map(
          (endpointId) => EndpointSupervisorDemand(
            endpointId: endpointId,
            mode: EndpointSupervisorMode.ENDPOINT_SUPERVISOR_MODE_TAKEOVER,
          ),
        ),
      ),
    );
    onPublished?.call(List<String>.unmodifiable(endpointIds));
  }
}

enum _PumpKind { events, platform }

enum _PumpSignal { next }

final class _PumpReady {
  const _PumpReady(this.control);

  final SendPort control;
}

final class _PumpStart {
  const _PumpStart(this.sendPort, this.engineHandle, this.kind);

  final SendPort sendPort;
  final int engineHandle;
  final _PumpKind kind;
}

final class _PumpFailure {
  const _PumpFailure(this.message);

  final String message;
}

Future<void> _runNativePump(_PumpStart start) async {
  final control = ReceivePort('${start.kind.name}-pump-control');
  final acknowledgements = StreamIterator<Object?>(control);
  try {
    final engine = AnyttyClientEngine.attach(start.engineHandle);
    start.sendPort.send(_PumpReady(control.sendPort));
    while (true) {
      final bytes = switch (start.kind) {
        _PumpKind.events => engine.nextEvent(),
        _PumpKind.platform => engine.nextPlatformRequest(),
      };
      if (bytes == null) return;
      start.sendPort.send(TransferableTypedData.fromList([bytes]));
      if (!await acknowledgements.moveNext() ||
          acknowledgements.current != _PumpSignal.next) {
        return;
      }
    }
  } catch (error) {
    start.sendPort.send(_PumpFailure(error.toString()));
  } finally {
    await acknowledgements.cancel();
    control.close();
  }
}
