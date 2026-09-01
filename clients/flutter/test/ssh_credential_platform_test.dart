import 'package:anytty_native/src/native/client_credential_store.dart';
import 'package:anytty_native/src/native/ssh_credential_platform.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.anytty.app/ssh-credentials.test');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() {
    messenger.setMockMethodCallHandler(channel, null);
  });

  test('looks up, signs, and deletes through the native signer', () async {
    final calls = <MethodCall>[];
    messenger.setMockMethodCallHandler(channel, (call) async {
      calls.add(call);
      return switch (call.method) {
        'lookup' => <String, Object>{
          'credentialRef': 'ssh-platform-studio',
          'publicKeyPkix': Uint8List.fromList([1, 2, 3]),
          'newlyCreated': true,
        },
        'sign' => Uint8List.fromList([9, 8, 7]),
        'delete' => null,
        _ => throw MissingPluginException(),
      };
    });
    final platform = MethodChannelSSHCredentialPlatform(channel: channel);

    final record = await platform.lookup(
      'ssh-platform-studio',
      createIfMissing: true,
    );
    final signature = await platform.sign(
      'ssh-platform-studio',
      List<int>.filled(32, 4),
      'SHA-256',
    );
    await platform.delete('ssh-platform-studio');

    expect(record.credentialRef, 'ssh-platform-studio');
    expect(record.publicKeyPkix, [1, 2, 3]);
    expect(record.newlyCreated, isTrue);
    expect(signature, [9, 8, 7]);
    expect(calls.map((call) => call.method), ['lookup', 'sign', 'delete']);
    expect(calls.first.arguments, {
      'credentialRef': 'ssh-platform-studio',
      'createIfMissing': true,
    });
    expect((calls[1].arguments as Map)['hash'], 'SHA-256');
    expect((calls[1].arguments as Map)['digest'], isA<Uint8List>());
  });

  test('rejects malformed refs, digests, and native responses', () async {
    final platform = MethodChannelSSHCredentialPlatform(channel: channel);

    await expectLater(
      platform.lookup('access-key', createIfMissing: true),
      throwsA(isA<ClientPlatformFailure>()),
    );
    await expectLater(
      platform.sign('ssh-platform-studio', [1, 2, 3], 'SHA-256'),
      throwsA(isA<ClientPlatformFailure>()),
    );

    messenger.setMockMethodCallHandler(
      channel,
      (_) async => <String, Object>{
        'credentialRef': 'ssh-platform-other',
        'publicKeyPkix': Uint8List.fromList([1]),
        'newlyCreated': false,
      },
    );
    await expectLater(
      platform.lookup('ssh-platform-studio', createIfMissing: false),
      throwsA(isA<ClientPlatformFailure>()),
    );
  });

  test('maps native failures to client platform failures', () async {
    messenger.setMockMethodCallHandler(channel, (_) async {
      throw PlatformException(code: 'not_found', message: 'missing key');
    });
    final platform = MethodChannelSSHCredentialPlatform(channel: channel);

    await expectLater(
      platform.lookup('ssh-platform-studio', createIfMissing: false),
      throwsA(
        isA<ClientPlatformFailure>()
            .having((error) => error.code, 'code', 'not_found')
            .having((error) => error.message, 'message', 'missing key'),
      ),
    );
  });
}
