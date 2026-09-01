import 'package:anytty_native/src/native/external_uri_platform.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('test.external-uri');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test(
    'opens an allowlisted URI through the narrow platform channel',
    () async {
      MethodCall? received;
      messenger.setMockMethodCallHandler(channel, (call) async {
        received = call;
        return null;
      });
      final launcher = MethodChannelExternalUriLauncher(channel: channel);

      await launcher.open(Uri.parse('https://example.com/docs?q=terminal'));

      expect(received?.method, 'openExternalUri');
      expect(received?.arguments, {
        'uri': 'https://example.com/docs?q=terminal',
      });
    },
  );

  test('rejects a non-external scheme before calling the platform', () async {
    var calls = 0;
    messenger.setMockMethodCallHandler(channel, (call) async {
      calls += 1;
      return null;
    });
    final launcher = MethodChannelExternalUriLauncher(channel: channel);

    await expectLater(
      launcher.open(Uri.parse('file:///private/data.txt')),
      throwsA(isA<FormatException>()),
    );
    expect(calls, 0);
  });

  test('surfaces a native open failure without fallback behavior', () async {
    messenger.setMockMethodCallHandler(channel, (call) async {
      throw PlatformException(
        code: 'open_external_failed',
        message: 'No application can open the external URI',
      );
    });
    final launcher = MethodChannelExternalUriLauncher(channel: channel);

    await expectLater(
      launcher.open(Uri.parse('mailto:dev@example.com')),
      throwsA(
        isA<PlatformException>().having(
          (error) => error.code,
          'code',
          'open_external_failed',
        ),
      ),
    );
  });
}
