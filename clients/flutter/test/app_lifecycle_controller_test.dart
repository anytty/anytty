import 'dart:async';

import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:anytty_native/src/native/app_lifecycle_controller.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'refreshes network detection after a suspended connectivity stream',
    () async {
      final runtime = _LifecycleRuntime();
      final network = _Connectivity();
      final controller = await AnyttyAppLifecycleController.start(
        runtime,
        connectivity: network,
      );
      await Future<void>.delayed(Duration.zero);
      controller.didChangeAppLifecycleState(AppLifecycleState.paused);
      network.current = [ConnectivityResult.none];
      controller.didChangeAppLifecycleState(AppLifecycleState.resumed);
      await Future<void>.delayed(Duration.zero);
      expect(runtime.resumes.last, isFalse);
      expect(network.checks, 2);
      await controller.close();
      await network.changes.close();
    },
  );

  test(
    'signals wifi to mobile changes even while both have connectivity',
    () async {
      final runtime = _LifecycleRuntime();
      final network = _Connectivity();
      final controller = await AnyttyAppLifecycleController.start(
        runtime,
        connectivity: network,
      );
      await Future<void>.delayed(Duration.zero);
      network.changes.add([ConnectivityResult.mobile]);
      await Future<void>.delayed(Duration.zero);
      expect(runtime.signals.last, (connected: true, reason: 'path_changed'));
      final count = runtime.signals.length;
      network.changes.add([ConnectivityResult.mobile]);
      await Future<void>.delayed(Duration.zero);
      expect(runtime.signals.length, count);
      network.changes.add([ConnectivityResult.none]);
      await Future<void>.delayed(Duration.zero);
      expect(runtime.signals.last, (connected: false, reason: 'offline'));
      await controller.close();
      await network.changes.close();
    },
  );
}

class _Connectivity implements Connectivity {
  List<ConnectivityResult> current = [ConnectivityResult.wifi];
  final changes = StreamController<List<ConnectivityResult>>.broadcast();
  int checks = 0;

  @override
  Future<List<ConnectivityResult>> checkConnectivity() async {
    checks++;
    return current;
  }

  @override
  Stream<List<ConnectivityResult>> get onConnectivityChanged => changes.stream;
}

class _LifecycleRuntime implements AnyttyLifecycleRuntime {
  final resumes = <bool>[];
  final signals = <({bool connected, String reason})>[];

  @override
  Future<void> resumeForeground({required bool connected}) async =>
      resumes.add(connected);

  @override
  void signalNetwork({required bool connected, required String reason}) =>
      signals.add((connected: connected, reason: reason));

  @override
  void suspendForeground({required bool connected}) {}
}
