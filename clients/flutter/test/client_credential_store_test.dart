import 'package:anytty_native/src/native/client_credential_store.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'prepares a stable Ed25519 identity and signs without exposing seed',
    () async {
      final secure = _MemorySecureStore();
      final store = ClientAccessCredentialStore(storage: secure);

      final created = await store.prepareRecord('client-key', 'endpoint-a');
      final existing = await store.prepareRecord('client-key', 'endpoint-a');

      expect(created.newlyCreated, isTrue);
      expect(existing.newlyCreated, isFalse);
      expect(created.publicKey, hasLength(32));
      expect(existing.publicKey, created.publicKey);
      expect(created.keyFingerprint, startsWith('ed25519-sha256:'));
      expect(secure.values.values.single, isNot(contains(created.publicKey)));

      final payload = [1, 3, 3, 7];
      final signatureBytes = await store.sign('client-key', payload);
      final verified = await Ed25519().verify(
        payload,
        signature: Signature(
          signatureBytes,
          publicKey: SimplePublicKey(
            created.publicKey,
            type: KeyPairType.ed25519,
          ),
        ),
      );
      expect(verified, isTrue);
    },
  );

  test(
    'requires a grant before resolve and preserves bound route data',
    () async {
      final store = ClientAccessCredentialStore(storage: _MemorySecureStore());
      await store.prepareRecord('client-key', 'endpoint-a');

      await expectLater(
        store.resolveRecord('client-key', 'endpoint-a'),
        throwsA(
          isA<ClientPlatformFailure>().having(
            (error) => error.code,
            'code',
            'unauthenticated',
          ),
        ),
      );

      await store.bindRecord(
        credentialRef: 'client-key',
        endpointId: 'endpoint-a',
        capabilityGrant: 'grant',
        cloudRouteGrant: [4, 5],
        cloudEdgeLocator: [6, 7],
      );
      final resolved = await store.resolveRecord('client-key', 'endpoint-a');
      expect(resolved.capabilityGrant, 'grant');
      expect(resolved.cloudRouteGrant, [4, 5]);
      expect(resolved.cloudEdgeLocator, [6, 7]);
    },
  );

  test('rejects reusing one credential ref for another endpoint', () async {
    final store = ClientAccessCredentialStore(storage: _MemorySecureStore());
    await store.prepareRecord('client-key', 'endpoint-a');

    await expectLater(
      store.prepareRecord('client-key', 'endpoint-b'),
      throwsA(
        isA<ClientPlatformFailure>().having(
          (error) => error.code,
          'code',
          'identity_conflict',
        ),
      ),
    );
  });
}

final class _MemorySecureStore implements SecureValueStore {
  final Map<String, String> values = {};

  @override
  Future<void> delete(String key) async => values.remove(key);

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;
}
