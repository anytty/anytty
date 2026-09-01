import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../generated/proto/bindingpb/client_binding.pb.dart';

final class ClientPlatformFailure implements Exception {
  const ClientPlatformFailure(this.code, this.message);

  final String code;
  final String message;

  @override
  String toString() => message;
}

abstract interface class SecureValueStore {
  Future<String?> read(String key);

  Future<void> write(String key, String value);

  Future<void> delete(String key);
}

final class FlutterSecureValueStore implements SecureValueStore {
  FlutterSecureValueStore({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);

  @override
  Future<void> delete(String key) => _storage.delete(key: key);
}

final class ClientAccessCredentialStore {
  ClientAccessCredentialStore({SecureValueStore? storage})
    : _storage = storage ?? FlutterSecureValueStore();

  static final _credentialRefPattern = RegExp(
    r'^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$',
  );
  static const _schemaVersion = 3;

  final SecureValueStore _storage;
  final Ed25519 _algorithm = Ed25519();

  Future<CredentialRecord> prepareRecord(
    String credentialRef,
    String endpointId,
  ) async {
    final ref = _validateRef(credentialRef);
    final endpoint = _validateEndpoint(endpointId);
    final existing = await _read(ref, requireGrant: false);
    if (existing != null) {
      _requireEndpoint(existing, endpoint);
      return _record(ref, existing, newlyCreated: false);
    }

    final keyPair = await _algorithm.newKeyPair();
    final seed = await keyPair.extractPrivateKeyBytes();
    final publicKey = await keyPair.extractPublicKey();
    final value = _StoredCredential(
      endpointId: endpoint,
      privateKeySeed: seed,
      publicKey: publicKey.bytes,
      keyFingerprint: await _fingerprint(publicKey.bytes),
      capabilityGrant: '',
      cloudRouteGrant: const [],
      cloudEdgeLocator: const [],
    );
    await _persist(ref, value);
    return _record(ref, value, newlyCreated: true);
  }

  Future<CredentialRecord> resolveRecord(
    String credentialRef,
    String endpointId,
  ) async {
    final ref = _validateRef(credentialRef);
    final endpoint = _validateEndpoint(endpointId);
    final value = await _read(ref, requireGrant: true);
    if (value == null) {
      throw const ClientPlatformFailure(
        'unauthenticated',
        'client access credential is missing',
      );
    }
    _requireEndpoint(value, endpoint);
    return _record(ref, value);
  }

  Future<CredentialRecord> bindRecord({
    required String credentialRef,
    required String endpointId,
    required String capabilityGrant,
    required List<int> cloudRouteGrant,
    required List<int> cloudEdgeLocator,
  }) async {
    final ref = _validateRef(credentialRef);
    final endpoint = _validateEndpoint(endpointId);
    final grant = capabilityGrant.trim();
    if (grant.isEmpty) {
      throw const ClientPlatformFailure(
        'protocol',
        'credential bind request is incomplete',
      );
    }
    final current = await _read(ref, requireGrant: false);
    if (current == null) {
      throw const ClientPlatformFailure(
        'unauthenticated',
        'client access identity is missing',
      );
    }
    _requireEndpoint(current, endpoint);
    final value = current.copyWith(
      capabilityGrant: grant,
      cloudRouteGrant: cloudRouteGrant,
      cloudEdgeLocator: cloudEdgeLocator,
    );
    await _persist(ref, value);
    return _record(ref, value);
  }

  Future<List<int>> sign(String credentialRef, List<int> payload) async {
    final ref = _validateRef(credentialRef);
    final value = await _read(ref, requireGrant: false);
    if (value == null) {
      throw const ClientPlatformFailure(
        'unauthenticated',
        'client access credential is missing',
      );
    }
    final keyPair = await _algorithm.newKeyPairFromSeed(value.privateKeySeed);
    final signature = await _algorithm.sign(payload, keyPair: keyPair);
    return signature.bytes;
  }

  Future<void> delete(String credentialRef) {
    final ref = _validateRef(credentialRef);
    return _storage.delete(_storageKey(ref));
  }

  Future<void> deleteMany(Iterable<String> credentialRefs) async {
    for (final ref in credentialRefs.map(_validateRef).toSet()) {
      await _storage.delete(_storageKey(ref));
    }
  }

  Future<_StoredCredential?> _read(
    String ref, {
    required bool requireGrant,
  }) async {
    final encoded = await _storage.read(_storageKey(ref));
    if (encoded == null) return null;
    try {
      final value = jsonDecode(encoded);
      if (value is! Map<String, dynamic> ||
          value.length != 6 ||
          value['version'] != _schemaVersion ||
          value['endpoint_id'] is! String ||
          value['private_key_seed'] is! String ||
          value['capability_grant'] is! String ||
          value['cloud_route_grant'] is! String ||
          value['cloud_edge_locator'] is! String) {
        throw const FormatException();
      }
      final seed = _decode(value['private_key_seed'] as String);
      if (seed.length != 32) throw const FormatException();
      final keyPair = await _algorithm.newKeyPairFromSeed(seed);
      final publicKey = await keyPair.extractPublicKey();
      final credential = _StoredCredential(
        endpointId: (value['endpoint_id'] as String).trim(),
        privateKeySeed: seed,
        publicKey: publicKey.bytes,
        keyFingerprint: await _fingerprint(publicKey.bytes),
        capabilityGrant: (value['capability_grant'] as String).trim(),
        cloudRouteGrant: _decode(value['cloud_route_grant'] as String),
        cloudEdgeLocator: _decode(value['cloud_edge_locator'] as String),
      );
      if (credential.endpointId.isEmpty ||
          (requireGrant && credential.capabilityGrant.isEmpty)) {
        throw const FormatException();
      }
      return credential;
    } catch (_) {
      throw const ClientPlatformFailure(
        'unauthenticated',
        'client access credential could not be decrypted',
      );
    }
  }

  Future<void> _persist(String ref, _StoredCredential value) {
    final payload = jsonEncode({
      'version': _schemaVersion,
      'endpoint_id': value.endpointId,
      'private_key_seed': _encode(value.privateKeySeed),
      'capability_grant': value.capabilityGrant,
      'cloud_route_grant': _encode(value.cloudRouteGrant),
      'cloud_edge_locator': _encode(value.cloudEdgeLocator),
    });
    return _storage.write(_storageKey(ref), payload);
  }

  CredentialRecord _record(
    String ref,
    _StoredCredential value, {
    bool newlyCreated = false,
  }) {
    return CredentialRecord(
      endpointId: value.endpointId,
      credentialRef: ref,
      publicKey: value.publicKey,
      keyFingerprint: value.keyFingerprint,
      capabilityGrant: value.capabilityGrant,
      newlyCreated: newlyCreated,
      cloudRouteGrant: value.cloudRouteGrant,
      cloudEdgeLocator: value.cloudEdgeLocator,
    );
  }

  String _validateRef(String value) {
    final normalized = value.trim();
    if (!_credentialRefPattern.hasMatch(normalized)) {
      throw const ClientPlatformFailure(
        'unauthenticated',
        'credential ref is invalid',
      );
    }
    return normalized;
  }

  String _validateEndpoint(String value) {
    final normalized = value.trim();
    if (normalized.isEmpty) {
      throw const ClientPlatformFailure(
        'protocol',
        'endpoint_id is required for ClientAccessIdentity',
      );
    }
    return normalized;
  }

  void _requireEndpoint(_StoredCredential value, String endpointId) {
    if (value.endpointId != endpointId) {
      throw const ClientPlatformFailure(
        'identity_conflict',
        'credential ref belongs to another endpoint',
      );
    }
  }

  Future<String> _fingerprint(List<int> publicKey) async {
    final digest = await Sha256().hash(publicKey);
    return 'ed25519-sha256:${_encode(digest.bytes)}';
  }

  String _storageKey(String ref) =>
      'anytty.client-access.v1.${_encode(utf8.encode(ref))}';

  String _encode(List<int> bytes) =>
      base64Url.encode(bytes).replaceAll('=', '');

  List<int> _decode(String value) => base64Url.decode(
    value.padRight(value.length + ((4 - value.length % 4) % 4), '='),
  );
}

final class _StoredCredential {
  const _StoredCredential({
    required this.endpointId,
    required this.privateKeySeed,
    required this.publicKey,
    required this.keyFingerprint,
    required this.capabilityGrant,
    required this.cloudRouteGrant,
    required this.cloudEdgeLocator,
  });

  final String endpointId;
  final List<int> privateKeySeed;
  final List<int> publicKey;
  final String keyFingerprint;
  final String capabilityGrant;
  final List<int> cloudRouteGrant;
  final List<int> cloudEdgeLocator;

  _StoredCredential copyWith({
    String? capabilityGrant,
    List<int>? cloudRouteGrant,
    List<int>? cloudEdgeLocator,
  }) {
    return _StoredCredential(
      endpointId: endpointId,
      privateKeySeed: privateKeySeed,
      publicKey: publicKey,
      keyFingerprint: keyFingerprint,
      capabilityGrant: capabilityGrant ?? this.capabilityGrant,
      cloudRouteGrant: cloudRouteGrant ?? this.cloudRouteGrant,
      cloudEdgeLocator: cloudEdgeLocator ?? this.cloudEdgeLocator,
    );
  }
}
