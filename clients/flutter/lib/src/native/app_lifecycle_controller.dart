import 'dart:async';

import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter/foundation.dart';

import 'anytty_runtime.dart';

final class AnyttyAppLifecycleController with WidgetsBindingObserver {
  AnyttyAppLifecycleController._(
    this._runtime,
    this._connectivity,
    this._onForegroundChanged,
  );

  final AnyttyLifecycleRuntime _runtime;
  final Connectivity _connectivity;
  final void Function(bool foreground)? _onForegroundChanged;
  StreamSubscription<List<ConnectivityResult>>? _networkSubscription;
  Future<void> _signalTail = Future.value();
  bool _connected = true;
  bool _closed = false;
  Set<ConnectivityResult> _networkPaths = {};

  static Future<AnyttyAppLifecycleController> start(
    AnyttyLifecycleRuntime runtime, {
    Connectivity? connectivity,
    void Function(bool foreground)? onForegroundChanged,
  }) async {
    final controller = AnyttyAppLifecycleController._(
      runtime,
      connectivity ?? Connectivity(),
      onForegroundChanged,
    );
    WidgetsBinding.instance.addObserver(controller);
    controller._networkSubscription = controller
        ._connectivity
        .onConnectivityChanged
        .listen(controller._handleNetworkResults);
    try {
      final results = await controller._connectivity
          .checkConnectivity()
          .timeout(const Duration(seconds: 2));
      controller._networkPaths = results.toSet();
      controller._connected = _hasNetwork(results);
    } catch (_) {
      controller._connected = true;
    }
    controller._enqueue(() async {
      runtime.signalNetwork(
        connected: controller._connected,
        reason: controller._connected ? 'startup' : 'offline',
      );
      if (WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed) {
        await runtime.resumeForeground(connected: controller._connected);
      }
    });
    controller._onForegroundChanged?.call(
      WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed,
    );
    return controller;
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (_closed) return;
    if (state == AppLifecycleState.resumed) {
      _onForegroundChanged?.call(true);
      _enqueue(() async {
        // Connectivity callbacks can be suspended by the OS while locked.
        try {
          final results = await _connectivity.checkConnectivity().timeout(
            const Duration(seconds: 2),
          );
          if (_closed) return;
          _networkPaths = results.toSet();
          _connected = _hasNetwork(results);
        } catch (_) {
          // Let the real endpoint handshake decide when OS detection fails.
        }
        if (!_closed) await _runtime.resumeForeground(connected: _connected);
      });
      return;
    }
    if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.hidden ||
        state == AppLifecycleState.detached) {
      _onForegroundChanged?.call(false);
      _enqueue(() async {
        _runtime.suspendForeground(connected: _connected);
      });
    }
  }

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    WidgetsBinding.instance.removeObserver(this);
    await _networkSubscription?.cancel();
    await _signalTail;
  }

  void _handleNetworkResults(List<ConnectivityResult> results) {
    final connected = _hasNetwork(results);
    final paths = results.toSet();
    if (_closed || setEquals(paths, _networkPaths)) return;
    _networkPaths = paths;
    _connected = connected;
    _enqueue(() async {
      _runtime.signalNetwork(
        connected: connected,
        reason: connected ? 'path_changed' : 'offline',
      );
    });
  }

  void _enqueue(Future<void> Function() operation) {
    final next = _signalTail.then((_) => operation());
    _signalTail = next.catchError((Object error, StackTrace stackTrace) {
      if (!_closed) {
        FlutterError.reportError(
          FlutterErrorDetails(
            exception: error,
            stack: stackTrace,
            library: 'AnyTTY lifecycle',
          ),
        );
      }
    });
  }
}

bool _hasNetwork(List<ConnectivityResult> results) =>
    results.any((result) => result != ConnectivityResult.none);
