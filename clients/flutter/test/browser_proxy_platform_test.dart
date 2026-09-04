import 'dart:async';

import 'package:anytty_native/src/native/browser_proxy_platform.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('test.browser-proxy');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test('returns the native session-bound lease', () async {
    MethodCall? received;
    messenger.setMockMethodCallHandler(channel, (call) async {
      received = call;
      return {
        'leaseId': 'lease-a',
        'sessionId': 'device-a',
        'endpointId': 'device-a',
        'routeId': 'direct-a',
        'routeGeneration': 12,
        'dnsProxied': true,
      };
    });
    const platform = MethodChannelBrowserProxyPlatform(channel: channel);

    final lease = await platform.open(
      sessionId: 'device-a',
      endpointId: 'device-a',
    );

    expect(received?.method, 'open');
    expect(received?.arguments, {
      'sessionId': 'device-a',
      'endpointId': 'device-a',
      'proxyHost': '127.0.0.1',
      'proxyPort': 0,
      'routeId': '',
      'routeGeneration': 0,
    });
    expect(lease.leaseId, 'lease-a');
    expect(lease.routeGeneration, 12);
    expect(lease.dnsProxied, isTrue);
  });

  test('rejects a lease bound to another endpoint', () async {
    messenger.setMockMethodCallHandler(channel, (call) async {
      return {
        'leaseId': 'lease-b',
        'sessionId': 'device-b',
        'endpointId': 'device-b',
        'routeId': 'direct-b',
      };
    });
    const platform = MethodChannelBrowserProxyPlatform(channel: channel);

    await expectLater(
      platform.open(sessionId: 'device-a', endpointId: 'device-a'),
      throwsA(isA<BrowserProxyUnavailableException>()),
    );
  });

  test('rejects a lease from a stale route generation', () async {
    messenger.setMockMethodCallHandler(channel, (call) async {
      return {
        'leaseId': 'lease-stale',
        'sessionId': 'device-a',
        'endpointId': 'device-a',
        'routeId': 'direct-a',
        'routeGeneration': 11,
      };
    });
    const platform = MethodChannelBrowserProxyPlatform(channel: channel);

    await expectLater(
      platform.open(
        sessionId: 'device-a',
        endpointId: 'device-a',
        routeId: 'direct-a',
        routeGeneration: 12,
      ),
      throwsA(isA<BrowserProxyUnavailableException>()),
    );
  });

  test(
    'maps a missing native implementation to an unavailable tunnel',
    () async {
      const platform = MethodChannelBrowserProxyPlatform(channel: channel);

      await expectLater(
        platform.open(sessionId: 'device-a', endpointId: 'device-a'),
        throwsA(isA<BrowserProxyUnavailableException>()),
      );
    },
  );

  test('clears browser data through the native channel', () async {
    final calls = <String>[];
    messenger.setMockMethodCallHandler(channel, (call) async {
      calls.add(call.method);
      return null;
    });
    const platform = MethodChannelBrowserProxyPlatform(channel: channel);

    await platform.clearBrowserData();

    expect(calls, ['clearData']);
  });

  test('bounds a native open callback that never returns', () async {
    messenger.setMockMethodCallHandler(channel, (call) {
      return Completer<Object?>().future;
    });
    const platform = MethodChannelBrowserProxyPlatform(
      channel: channel,
      operationTimeout: Duration(milliseconds: 20),
    );

    await expectLater(
      platform.open(sessionId: 'device-a', endpointId: 'device-a'),
      throwsA(
        isA<BrowserProxyUnavailableException>().having(
          (error) => error.message,
          'message',
          'Web tunnel setup timed out',
        ),
      ),
    );
  });
}
