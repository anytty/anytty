import 'package:flutter/services.dart';

final class BrowserProxyLease {
  const BrowserProxyLease({
    required this.leaseId,
    required this.sessionId,
    required this.endpointId,
    required this.routeId,
    required this.routeGeneration,
    required this.dnsProxied,
  });

  final String leaseId;
  final String sessionId;
  final String endpointId;
  final String routeId;
  final int routeGeneration;
  final bool dnsProxied;
}

final class BrowserProxyUnavailableException implements Exception {
  const BrowserProxyUnavailableException([
    this.message = 'Web tunnel is unavailable',
  ]);

  final String message;

  @override
  String toString() => message;
}

abstract interface class BrowserProxyPlatform {
  Future<BrowserProxyLease> open({
    required String sessionId,
    required String endpointId,
    String proxyHost = '127.0.0.1',
    int proxyPort = 0,
    String routeId = '',
    int routeGeneration = 0,
  });

  Future<void> close(BrowserProxyLease lease);

  /// Clears shared native WebView data before another session opens.
  ///
  /// Cold restore intentionally keeps only URL, scroll position, and a PNG
  /// snapshot. A platform that cannot clear its shared data store must fail
  /// closed and avoid creating the next WebView.
  Future<void> clearBrowserData();
}

final class MethodChannelBrowserProxyPlatform implements BrowserProxyPlatform {
  const MethodChannelBrowserProxyPlatform({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel('com.anytty.app/browser-proxy');

  static const instance = MethodChannelBrowserProxyPlatform();

  final MethodChannel _channel;

  @override
  Future<BrowserProxyLease> open({
    required String sessionId,
    required String endpointId,
    String proxyHost = '127.0.0.1',
    int proxyPort = 0,
    String routeId = '',
    int routeGeneration = 0,
  }) async {
    late final Object? value;
    try {
      value = await _channel.invokeMethod<Object?>('open', {
        'sessionId': sessionId,
        'endpointId': endpointId,
        'proxyHost': proxyHost,
        'proxyPort': proxyPort,
        'routeId': routeId,
        'routeGeneration': routeGeneration,
      });
    } on MissingPluginException {
      throw const BrowserProxyUnavailableException();
    } on PlatformException catch (error) {
      throw BrowserProxyUnavailableException(
        error.message == null || error.message!.trim().isEmpty
            ? 'Web tunnel is unavailable'
            : error.message!,
      );
    }
    if (value is! Map) {
      throw const BrowserProxyUnavailableException();
    }
    final data = value.cast<Object?, Object?>();
    final leaseId = _string(data['leaseId']);
    final responseSessionId = _string(data['sessionId']);
    final responseEndpointId = _string(data['endpointId']);
    final responseRouteId = _string(data['routeId']);
    final responseRouteGeneration = _int(data['routeGeneration']);
    if (leaseId.isEmpty ||
        responseSessionId != sessionId ||
        responseEndpointId != endpointId ||
        responseRouteId.isEmpty ||
        (routeId.isNotEmpty && responseRouteId != routeId) ||
        (routeGeneration > 0 && responseRouteGeneration != routeGeneration)) {
      throw const BrowserProxyUnavailableException(
        'Web tunnel returned an invalid session binding',
      );
    }
    return BrowserProxyLease(
      leaseId: leaseId,
      sessionId: responseSessionId,
      endpointId: responseEndpointId,
      routeId: responseRouteId,
      routeGeneration: responseRouteGeneration,
      dnsProxied: data['dnsProxied'] == true,
    );
  }

  @override
  Future<void> close(BrowserProxyLease lease) {
    return _channel.invokeMethod<void>('close', {'leaseId': lease.leaseId});
  }

  @override
  Future<void> clearBrowserData() async {
    try {
      await _channel.invokeMethod<void>('clearData');
    } on MissingPluginException {
      throw const BrowserProxyUnavailableException(
        'WebView data isolation is unavailable',
      );
    } on PlatformException catch (error) {
      throw BrowserProxyUnavailableException(
        error.message == null || error.message!.trim().isEmpty
            ? 'WebView data isolation is unavailable'
            : error.message!,
      );
    }
  }

  static String _string(Object? value) => value is String ? value.trim() : '';

  static int _int(Object? value) => value is num ? value.toInt() : 0;
}
