import 'dart:async';

import 'package:flutter/services.dart';

final class BackgroundEndpointProjection {
  const BackgroundEndpointProjection({
    required this.endpointId,
    required this.phase,
  });

  final String endpointId;
  final String phase;

  Map<String, Object> toMap() => <String, Object>{
    'endpointId': endpointId,
    'phase': phase,
  };
}

final class BackgroundNotification {
  const BackgroundNotification({
    required this.id,
    required this.title,
    required this.body,
    required this.route,
  });

  final String id;
  final String title;
  final String body;
  final String route;

  Map<String, Object> toMap() => <String, Object>{
    'id': id,
    'title': title,
    'body': body,
    'route': route,
  };
}

abstract interface class BackgroundPlatformHost {
  Stream<String> get routes;

  Future<void> initialize();

  Future<void> syncState({
    required bool foreground,
    required bool enabled,
    required List<BackgroundEndpointProjection> endpoints,
  });

  Future<bool> notificationsAuthorized();

  Future<bool> requestNotificationAuthorization();

  Future<void> showNotification(BackgroundNotification notification);
}

final class MethodChannelBackgroundPlatform implements BackgroundPlatformHost {
  MethodChannelBackgroundPlatform._() {
    _channel.setMethodCallHandler(_handleMethodCall);
  }

  static final instance = MethodChannelBackgroundPlatform._();
  static const _channel = MethodChannel('com.anytty.app/background');

  final StreamController<String> _routes = StreamController<String>.broadcast();
  bool _initialized = false;

  @override
  Stream<String> get routes => _routes.stream;

  @override
  Future<void> initialize() async {
    if (_initialized) return;
    _initialized = true;
    final pending = await _channel.invokeMethod<String>('takePendingRoute');
    _publishRoute(pending);
  }

  @override
  Future<void> syncState({
    required bool foreground,
    required bool enabled,
    required List<BackgroundEndpointProjection> endpoints,
  }) => _channel.invokeMethod<void>('syncState', <String, Object>{
    'foreground': foreground,
    'enabled': enabled,
    'endpoints': endpoints.map((endpoint) => endpoint.toMap()).toList(),
  });

  @override
  Future<bool> notificationsAuthorized() async =>
      await _channel.invokeMethod<bool>('notificationsAuthorized') ?? false;

  @override
  Future<bool> requestNotificationAuthorization() async =>
      await _channel.invokeMethod<bool>('requestNotificationAuthorization') ??
      false;

  @override
  Future<void> showNotification(BackgroundNotification notification) =>
      _channel.invokeMethod<void>('showNotification', notification.toMap());

  Future<void> _handleMethodCall(MethodCall call) async {
    if (call.method != 'openRoute') return;
    _publishRoute(call.arguments as String?);
  }

  void _publishRoute(String? route) {
    final normalized = route?.trim() ?? '';
    if (normalized.startsWith('/terminal/')) _routes.add(normalized);
  }
}
