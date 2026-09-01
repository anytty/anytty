import 'dart:async';

import '../app/background_preferences.dart';
import '../generated/proto/apipb/application.pb.dart' as application;
import '../generated/proto/apipb/common.pb.dart';
import '../generated/proto/apipb/terminal.pb.dart';
import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'anytty_runtime.dart';
import 'background_platform.dart';

final class AnyttyBackgroundRuntimeController {
  AnyttyBackgroundRuntimeController._(
    this._runtime,
    this._platform,
    this._preferences,
  );

  final AnyttyRuntime _runtime;
  final BackgroundPlatformHost _platform;
  BackgroundPreferences _preferences;
  StreamSubscription<EventEnvelope>? _eventSubscription;
  StreamSubscription<List<String>>? _demandSubscription;
  Future<void> _platformTail = Future.value();
  final Set<String> _notifiedEventIds = <String>{};
  final BackgroundSessionGenerationFence _generationFence =
      BackgroundSessionGenerationFence();
  bool _foreground = true;
  bool _closed = false;

  static Future<AnyttyBackgroundRuntimeController> start({
    required AnyttyRuntime runtime,
    required BackgroundPlatformHost platform,
    required BackgroundPreferences preferences,
  }) async {
    final controller = AnyttyBackgroundRuntimeController._(
      runtime,
      platform,
      preferences,
    );
    await platform.initialize();
    controller._eventSubscription = runtime.events.listen(
      controller._handleBindingEvent,
    );
    controller._demandSubscription = runtime.endpointDemands.listen((_) {
      controller._enqueuePlatform(controller._syncPlatform);
    });
    controller._enqueuePlatform(controller._syncPlatform);
    return controller;
  }

  void setForeground(bool foreground) {
    if (_closed || foreground == _foreground) return;
    _foreground = foreground;
    _enqueuePlatform(_syncPlatform);
  }

  void setPreferences(BackgroundPreferences preferences) {
    if (_closed || preferences == _preferences) return;
    _preferences = preferences;
    _enqueuePlatform(_syncPlatform);
  }

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    await _eventSubscription?.cancel();
    await _demandSubscription?.cancel();
    await _platformTail;
    await _platform.syncState(
      foreground: true,
      enabled: false,
      endpoints: const <BackgroundEndpointProjection>[],
    );
  }

  Future<void> _syncPlatform() async {
    if (_closed) return;
    final demanded = _runtime.demandedEndpointIds;
    final byEndpoint = <String, EndpointSupervisorProjection>{};
    try {
      for (final projection in _runtime.supervisorSnapshot().endpoints) {
        byEndpoint[projection.endpointId] = projection;
      }
    } catch (_) {
      // A demand snapshot remains enough to keep the host assertion accurate.
    }
    final endpoints = demanded
        .map(
          (endpointId) => BackgroundEndpointProjection(
            endpointId: endpointId,
            phase: byEndpoint[endpointId]?.phase ?? 'demanded',
          ),
        )
        .toList(growable: false);
    await _platform.syncState(
      foreground: _foreground,
      enabled: _preferences.keepConnections,
      endpoints: endpoints,
    );
  }

  void _handleBindingEvent(EventEnvelope event) {
    if (_closed) return;
    final applicationEvent = _generationFence.applicationEventForCurrentSession(
      event,
    );
    if (_foreground ||
        !_preferences.notifications ||
        applicationEvent == null) {
      return;
    }
    final notification = projectBackgroundNotification(applicationEvent);
    if (notification == null || !_notifiedEventIds.add(notification.id)) return;
    if (_notifiedEventIds.length > 256) {
      _notifiedEventIds.remove(_notifiedEventIds.first);
    }
    _enqueuePlatform(() => _platform.showNotification(notification));
  }

  void _enqueuePlatform(Future<void> Function() operation) {
    final next = _platformTail.then((_) => operation());
    _platformTail = next.catchError((Object _) {});
  }
}

final class BackgroundSessionGenerationFence {
  final Map<String, EndpointSessionStamp> _currentByEndpoint = {};
  final Map<int, EndpointSessionStamp> _openedByHandle = {};

  application.EventEnvelope? applicationEventForCurrentSession(
    EventEnvelope event,
  ) {
    if (event.whichEvent() == EventEnvelope_Event.openSession) {
      final opened = event.openSession;
      if (opened.hasSession() && opened.sessionHandle.toInt() != 0) {
        _observe(opened.sessionHandle.toInt(), opened.session);
      }
      return null;
    }
    if (event.whichEvent() != EventEnvelope_Event.application ||
        !event.application.hasEvent() ||
        !event.application.event.hasOriginSession()) {
      return null;
    }
    final applicationEvent = event.application.event;
    final origin = applicationEvent.originSession;
    final current = _currentByEndpoint[origin.endpointId];
    final opened = _openedByHandle[event.application.sessionHandle.toInt()];
    if (current == null ||
        opened == null ||
        !_sameSessionStamp(current, origin) ||
        !_sameSessionStamp(opened, origin)) {
      return null;
    }
    return applicationEvent.deepCopy();
  }

  void _observe(int sessionHandle, EndpointSessionStamp stamp) {
    final endpointId = stamp.endpointId.trim();
    if (endpointId.isEmpty ||
        stamp.routeId.trim().isEmpty ||
        stamp.generation.toInt() <= 0) {
      return;
    }
    _openedByHandle[sessionHandle] = stamp.deepCopy();
    final current = _currentByEndpoint[endpointId];
    if (current == null || stamp.generation > current.generation) {
      _currentByEndpoint[endpointId] = stamp.deepCopy();
    }
  }
}

bool _sameSessionStamp(EndpointSessionStamp left, EndpointSessionStamp right) {
  return left.endpointId == right.endpointId &&
      left.routeId == right.routeId &&
      left.generation == right.generation;
}

BackgroundNotification? projectBackgroundNotification(
  application.EventEnvelope event,
) {
  final eventId = event.eventId.trim();
  if (eventId.isEmpty) return null;
  switch (event.whichEvent()) {
    case application.EventEnvelope_Event.terminalLifecycle:
      final lifecycle = event.terminalLifecycle;
      if (!lifecycle.hasTerminal() ||
          lifecycle.terminal.state != TerminalState.TERMINAL_STATE_EXITED ||
          !lifecycle.terminal.hasRef()) {
        return null;
      }
      final terminal = lifecycle.terminal;
      final endpointId = terminal.ref.endpointId.trim();
      final terminalId = terminal.ref.terminalId.trim();
      if (endpointId.isEmpty || terminalId.isEmpty) return null;
      final label = terminal.name.trim().isEmpty
          ? terminalId
          : terminal.name.trim();
      final exitCode = terminal.hasExitCode() ? terminal.exitCode : null;
      return BackgroundNotification(
        id: eventId,
        title: 'Terminal finished',
        body: exitCode == null
            ? '$label exited'
            : '$label exited with code $exitCode',
        route: _terminalRoute(endpointId, terminalId),
      );
    case application.EventEnvelope_Event.fileTransferCompleted:
      final completed = event.fileTransferCompleted;
      final endpointId = event.hasOriginSession()
          ? event.originSession.endpointId.trim()
          : '';
      if (endpointId.isEmpty || !completed.hasTransfer()) return null;
      final path = completed.transfer.path.trim();
      return BackgroundNotification(
        id: eventId,
        title: 'Transfer complete',
        body: path.isEmpty ? 'File transfer completed' : path,
        route: '/terminal/${Uri.encodeComponent(endpointId)}',
      );
    default:
      return null;
  }
}

String _terminalRoute(String endpointId, String terminalId) =>
    '/terminal/${Uri.encodeComponent(endpointId)}/'
    '${Uri.encodeComponent(terminalId)}';
