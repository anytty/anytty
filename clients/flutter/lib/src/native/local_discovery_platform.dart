import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/services.dart';

import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'client_credential_store.dart';

abstract interface class LocalDiscoveryPlatform {
  Future<LocalDiscoveryLookupResult> lookup(
    String deviceId,
    String deviceFingerprint,
  );
}

final class MethodChannelLocalDiscoveryPlatform
    implements LocalDiscoveryPlatform {
  MethodChannelLocalDiscoveryPlatform({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel(_channelName);

  static const _channelName = 'com.anytty.app/local-discovery';
  static const _maximumCandidates = 64;
  final MethodChannel _channel;

  @override
  Future<LocalDiscoveryLookupResult> lookup(
    String deviceId,
    String deviceFingerprint,
  ) async {
    final id = deviceId.trim();
    final fingerprint = deviceFingerprint.trim();
    if (id.isEmpty || fingerprint.isEmpty) {
      throw const ClientPlatformFailure(
        'protocol',
        'Local discovery identity is incomplete',
      );
    }
    final Object? raw;
    try {
      raw = await _channel.invokeMethod<Object?>('lookup', {
        'expectedKey': await localDiscoveryKey(id, fingerprint),
      });
    } on PlatformException catch (error) {
      throw ClientPlatformFailure(
        error.code.trim().isEmpty ? 'temporary' : error.code.trim(),
        error.message?.trim().isNotEmpty == true
            ? error.message!.trim()
            : 'Local discovery lookup failed',
      );
    }
    if (raw is! List || raw.length > _maximumCandidates) {
      throw const ClientPlatformFailure(
        'protocol',
        'Local discovery response is invalid',
      );
    }
    final nowUnixNano = DateTime.now().microsecondsSinceEpoch * 1000;
    final candidates = <LocalDiscoveryCandidate>[];
    for (final item in raw) {
      if (item is! Map) continue;
      final address = item['address'];
      final port = item['port'];
      final protocolVersion = item['protocolVersion'];
      final expiresAtUnixNano = item['expiresAtUnixNano'];
      final networkHandle = item['networkHandle'] ?? 0;
      if (address is! String ||
          address.trim().isEmpty ||
          address.length > 255 ||
          address.contains('\n') ||
          address.contains('\r') ||
          port is! int ||
          port <= 0 ||
          port > 65535 ||
          protocolVersion is! int ||
          protocolVersion <= 0 ||
          expiresAtUnixNano is! int ||
          expiresAtUnixNano <= nowUnixNano ||
          networkHandle is! int ||
          networkHandle < 0) {
        continue;
      }
      candidates.add(
        LocalDiscoveryCandidate(
          address: address.trim(),
          port: port,
          protocolVersion: protocolVersion,
          expiresAtUnixNano: Int64(expiresAtUnixNano),
          networkHandle: Int64(networkHandle),
        ),
      );
    }
    return LocalDiscoveryLookupResult(candidates: candidates);
  }
}

Future<String> localDiscoveryKey(
  String deviceId,
  String deviceFingerprint,
) async {
  final input = utf8.encode(
    'anytty-lan-discovery-v1\u0000${deviceId.trim()}\u0000${deviceFingerprint.trim()}',
  );
  final digest = await Sha256().hash(input);
  return digest.bytes
      .map((value) => value.toRadixString(16).padLeft(2, '0'))
      .join();
}
