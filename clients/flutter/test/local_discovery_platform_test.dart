import 'package:anytty_native/src/native/local_discovery_platform.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('com.anytty.test/local-discovery');

  tearDown(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test(
    'sends the pinned identity hash and accepts bounded fresh candidates',
    () async {
      MethodCall? call;
      final expiry =
          DateTime.now().microsecondsSinceEpoch * 1000 +
          30 * 1000 * 1000 * 1000;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (value) async {
            call = value;
            return [
              {
                'address': '192.168.1.8',
                'port': 41120,
                'protocolVersion': 1,
                'expiresAtUnixNano': expiry,
                'networkHandle': 42,
              },
            ];
          });

      final result = await MethodChannelLocalDiscoveryPlatform(channel: channel)
          .lookup('device-1', 'ed25519-sha256:test');

      expect(call?.method, 'lookup');
      expect(
        (call?.arguments as Map)['expectedKey'],
        matches(RegExp(r'^[0-9a-f]{64}$')),
      );
      expect(result.candidates.single.address, '192.168.1.8');
      expect(result.candidates.single.networkHandle.toInt(), 42);
    },
  );

  test('filters expired and structurally invalid native candidates', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (_) async {
          return [
            {
              'address': '',
              'port': 0,
              'protocolVersion': 0,
              'expiresAtUnixNano': 0,
            },
            {
              'address': '192.168.1.9',
              'port': 70000,
              'protocolVersion': 1,
              'expiresAtUnixNano': DateTime.now().microsecondsSinceEpoch * 1000,
            },
          ];
        });

    final result = await MethodChannelLocalDiscoveryPlatform(channel: channel)
        .lookup('device-1', 'fingerprint');
    expect(result.candidates, isEmpty);
  });
}
