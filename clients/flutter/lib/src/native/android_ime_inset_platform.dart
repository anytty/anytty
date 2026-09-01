import 'dart:async';

import 'package:flutter/services.dart';

final class AndroidImeInsetPlatform {
  AndroidImeInsetPlatform._() {
    _channel.setMethodCallHandler(_handleMethodCall);
  }

  static final instance = AndroidImeInsetPlatform._();
  static const _channel = MethodChannel('com.anytty.app/ime-insets');

  final _insets = StreamController<double>.broadcast();

  Stream<double> get physicalInsets => _insets.stream;

  Future<double?> currentPhysicalInset() async {
    final value = await _channel.invokeMethod<num>('currentInset');
    return value?.toDouble();
  }

  Future<void> _handleMethodCall(MethodCall call) async {
    if (call.method != 'insetChanged') return;
    final inset = call.arguments;
    if (inset is num) _insets.add(inset.toDouble());
  }
}
