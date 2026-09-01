import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/endpoint_registry_platform.dart';
import 'package:anytty_native/src/native/client_credential_store.dart';
import 'package:anytty_native/src/native/ssh_credential_platform.dart';
import 'package:anytty_native/src/native/local_discovery_platform.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('loads and stores the registry as an opaque blob', () async {
    final store = _MemoryBlobStore([1, 2, 3]);
    final platform = FlutterClientPlatform(registryStore: store);

    final loaded = await platform.handle(
      PlatformRequest(
        requestId: Int64.ONE,
        endpointRegistryLoad: EndpointRegistryLoadRequest(),
      ),
    );
    expect(loaded.endpointRegistry.registryProto, [1, 2, 3]);

    final stored = await platform.handle(
      PlatformRequest(
        requestId: Int64(2),
        endpointRegistryStore: EndpointRegistryStoreRequest(
          registryProto: [9, 8, 7],
        ),
      ),
    );
    expect(stored.hasError(), isFalse);
    expect(store.value, [9, 8, 7]);
  });

  test(
    'routes SSH lookup, sign, and delete through the secure signer',
    () async {
      final ssh = _MemorySSHCredentialPlatform();
      final platform = FlutterClientPlatform(
        registryStore: _MemoryBlobStore(const []),
        accessCredentials: ClientAccessCredentialStore(
          storage: _MemorySecureStore(),
        ),
        sshCredentials: ssh,
      );
      final lookup = await platform.handle(
        PlatformRequest(
          requestId: Int64.ONE,
          sshCredentialLookup: SSHCredentialLookupRequest(
            credentialRef: 'ssh-platform-key',
            createIfMissing: true,
          ),
        ),
      );
      expect(lookup.sshCredential.credentialRef, 'ssh-platform-key');
      expect(lookup.sshCredential.publicKeyPkix, [1, 2, 3]);
      expect(lookup.sshCredential.newlyCreated, isTrue);

      final sign = await platform.handle(
        PlatformRequest(
          requestId: Int64(2),
          sshCredentialSign: SSHCredentialSignRequest(
            credentialRef: 'ssh-platform-key',
            digest: List.filled(32, 7),
            hash: 'SHA-256',
          ),
        ),
      );
      expect(sign.sshCredentialSign.signature, [9, 8, 7]);

      final deleted = await platform.handle(
        PlatformRequest(
          requestId: Int64(3),
          sshCredentialDelete: SSHCredentialDeleteRequest(
            credentialRef: 'ssh-platform-key',
          ),
        ),
      );
      expect(deleted.hasError(), isFalse);
      expect(ssh.deleted, ['ssh-platform-key']);
    },
  );

  test('rolls back registry when credential cleanup fails', () async {
    final store = _MemoryBlobStore([1, 2, 3]);
    final secure = _MemorySecureStore(failDeletes: true);
    final platform = FlutterClientPlatform(
      registryStore: store,
      accessCredentials: ClientAccessCredentialStore(storage: secure),
    );
    final response = await platform.handle(
      PlatformRequest(
        requestId: Int64.ONE,
        endpointRegistryStore: EndpointRegistryStoreRequest(
          registryProto: [9, 8, 7],
          deleteCredentialRefs: ['access-key'],
        ),
      ),
    );

    expect(response.error.code, ApiErrorCode.API_ERROR_CODE_UNAVAILABLE);
    expect(store.value, [1, 2, 3]);
  });

  test('routes pinned identities through the platform DNS-SD cache', () async {
    final discovery = _MemoryLocalDiscoveryPlatform();
    final platform = FlutterClientPlatform(
      registryStore: _MemoryBlobStore(const []),
      localDiscovery: discovery,
    );
    final response = await platform.handle(
      PlatformRequest(
        requestId: Int64.ONE,
        localDiscoveryLookup: LocalDiscoveryLookupRequest(
          deviceId: 'device-1',
          deviceFingerprint: 'ed25519-sha256:test',
        ),
      ),
    );

    expect(discovery.identity, ('device-1', 'ed25519-sha256:test'));
    expect(response.localDiscovery.candidates.single.address, '192.168.1.8');
    expect(response.localDiscovery.candidates.single.networkHandle.toInt(), 42);
  });
}

final class _MemoryLocalDiscoveryPlatform implements LocalDiscoveryPlatform {
  (String, String)? identity;

  @override
  Future<LocalDiscoveryLookupResult> lookup(
    String deviceId,
    String deviceFingerprint,
  ) async {
    identity = (deviceId, deviceFingerprint);
    return LocalDiscoveryLookupResult(
      candidates: [
        LocalDiscoveryCandidate(
          address: '192.168.1.8',
          port: 41120,
          protocolVersion: 1,
          expiresAtUnixNano:
              Int64(DateTime.now().microsecondsSinceEpoch * 1000) +
              Int64(30 * 1000 * 1000 * 1000),
          networkHandle: Int64(42),
        ),
      ],
    );
  }
}

final class _MemorySSHCredentialPlatform implements SSHCredentialPlatform {
  final List<String> deleted = [];

  @override
  Future<void> delete(String credentialRef) async => deleted.add(credentialRef);

  @override
  Future<void> deleteMany(Iterable<String> credentialRefs) async {
    deleted.addAll(credentialRefs);
  }

  @override
  Future<SSHCredentialRecord> lookup(
    String credentialRef, {
    required bool createIfMissing,
  }) async => SSHCredentialRecord(
    credentialRef: credentialRef,
    publicKeyPkix: [1, 2, 3],
    newlyCreated: createIfMissing,
  );

  @override
  Future<List<int>> sign(
    String credentialRef,
    List<int> digest,
    String hash,
  ) async => [9, 8, 7];
}

final class _MemoryBlobStore implements EndpointRegistryBlobStore {
  _MemoryBlobStore(List<int> initial) : value = List.of(initial);

  List<int> value;

  @override
  Future<List<int>> load() async => List.of(value);

  @override
  Future<void> store(List<int> bytes) async => value = List.of(bytes);

  @override
  Future<void> clear() async => value = [];
}

final class _MemorySecureStore implements SecureValueStore {
  _MemorySecureStore({this.failDeletes = false});

  final bool failDeletes;
  final Map<String, String> values = {};

  @override
  Future<void> delete(String key) async {
    if (failDeletes) throw StateError('delete failed');
    values.remove(key);
  }

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;
}
