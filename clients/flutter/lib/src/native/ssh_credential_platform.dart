import 'package:flutter/services.dart';

import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'client_credential_store.dart';

abstract interface class SSHCredentialPlatform {
  Future<SSHCredentialRecord> lookup(
    String credentialRef, {
    required bool createIfMissing,
  });

  Future<List<int>> sign(String credentialRef, List<int> digest, String hash);

  Future<void> delete(String credentialRef);

  Future<void> deleteMany(Iterable<String> credentialRefs);
}

final class MethodChannelSSHCredentialPlatform
    implements SSHCredentialPlatform {
  MethodChannelSSHCredentialPlatform({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel(_channelName);

  static const _channelName = 'com.anytty.app/ssh-credentials';
  final MethodChannel _channel;

  @override
  Future<SSHCredentialRecord> lookup(
    String credentialRef, {
    required bool createIfMissing,
  }) async {
    final ref = _validateRef(credentialRef);
    final Object? raw;
    try {
      raw = await _channel.invokeMethod<Object?>('lookup', {
        'credentialRef': ref,
        'createIfMissing': createIfMissing,
      });
    } on PlatformException catch (error) {
      throw _failure(error, 'SSH credential lookup failed');
    }
    if (raw is! Map) {
      throw const ClientPlatformFailure(
        'protocol',
        'SSH credential response is incomplete',
      );
    }
    final returnedRef = raw['credentialRef'];
    final publicKey = _bytes(raw['publicKeyPkix']);
    final newlyCreated = raw['newlyCreated'];
    if (returnedRef != ref || publicKey.isEmpty || newlyCreated is! bool) {
      throw const ClientPlatformFailure(
        'protocol',
        'SSH credential response is invalid',
      );
    }
    return SSHCredentialRecord(
      credentialRef: ref,
      publicKeyPkix: publicKey,
      newlyCreated: newlyCreated,
    );
  }

  @override
  Future<List<int>> sign(
    String credentialRef,
    List<int> digest,
    String hash,
  ) async {
    final ref = _validateRef(credentialRef);
    if (hash != 'SHA-256' || digest.length != 32) {
      throw const ClientPlatformFailure(
        'protocol',
        'SSH signer only accepts SHA-256 digests',
      );
    }
    final Object? raw;
    try {
      raw = await _channel.invokeMethod<Object?>('sign', {
        'credentialRef': ref,
        'digest': Uint8List.fromList(digest),
        'hash': hash,
      });
    } on PlatformException catch (error) {
      throw _failure(error, 'SSH signature failed');
    }
    final signature = _bytes(raw);
    if (signature.isEmpty) {
      throw const ClientPlatformFailure(
        'protocol',
        'SSH signer returned an empty signature',
      );
    }
    return signature;
  }

  @override
  Future<void> delete(String credentialRef) async {
    final ref = _validateRef(credentialRef);
    try {
      await _channel.invokeMethod<void>('delete', {'credentialRef': ref});
    } on PlatformException catch (error) {
      throw _failure(error, 'SSH credential delete failed');
    }
  }

  @override
  Future<void> deleteMany(Iterable<String> credentialRefs) async {
    for (final ref in credentialRefs.toSet()) {
      await delete(ref);
    }
  }

  String _validateRef(String value) {
    final normalized = value.trim();
    if (!normalized.startsWith('ssh-platform-') ||
        !RegExp(r'^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$').hasMatch(normalized)) {
      throw const ClientPlatformFailure(
        'protocol',
        'SSH credential ref is invalid',
      );
    }
    return normalized;
  }

  List<int> _bytes(Object? value) {
    if (value is Uint8List) return List<int>.of(value);
    if (value is List) {
      final bytes = <int>[];
      for (final item in value) {
        if (item is! int || item < 0 || item > 255) return const [];
        bytes.add(item);
      }
      return bytes;
    }
    return const [];
  }

  ClientPlatformFailure _failure(PlatformException error, String fallback) {
    final code = error.code.trim().isEmpty ? 'temporary' : error.code.trim();
    final message = error.message?.trim();
    return ClientPlatformFailure(
      code,
      message == null || message.isEmpty ? fallback : message,
    );
  }
}
