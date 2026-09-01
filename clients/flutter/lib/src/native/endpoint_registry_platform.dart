import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../generated/proto/apipb/common.pb.dart';
import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'anytty_runtime.dart';
import 'client_credential_store.dart';
import 'local_discovery_platform.dart';
import 'ssh_credential_platform.dart';

abstract interface class EndpointRegistryBlobStore {
  Future<List<int>> load();

  Future<void> store(List<int> bytes);

  Future<void> clear();
}

final class PreferencesEndpointRegistryBlobStore
    implements EndpointRegistryBlobStore {
  static const _key = 'anytty.flutter.endpoint-registry.v1';
  static const _maximumBytes = 1 << 20;

  @override
  Future<List<int>> load() async {
    final preferences = await SharedPreferences.getInstance();
    final encoded = preferences.getString(_key);
    if (encoded == null) return const [];
    try {
      return base64Decode(encoded);
    } on FormatException {
      throw const FormatException('endpoint registry payload is malformed');
    }
  }

  @override
  Future<void> store(List<int> bytes) async {
    if (bytes.isEmpty || bytes.length > _maximumBytes) {
      throw const FormatException('endpoint registry payload size is invalid');
    }
    final preferences = await SharedPreferences.getInstance();
    final saved = await preferences.setString(_key, base64Encode(bytes));
    if (!saved) throw StateError('failed to persist endpoint registry');
  }

  @override
  Future<void> clear() async {
    final preferences = await SharedPreferences.getInstance();
    final cleared = await preferences.remove(_key);
    if (!cleared && preferences.containsKey(_key)) {
      throw StateError('failed to clear endpoint registry');
    }
  }
}

final class FlutterClientPlatform implements AnyttyPlatformHandler {
  FlutterClientPlatform({
    EndpointRegistryBlobStore? registryStore,
    ClientAccessCredentialStore? accessCredentials,
    LocalDiscoveryPlatform? localDiscovery,
    SSHCredentialPlatform? sshCredentials,
  }) : _registryStore = registryStore ?? PreferencesEndpointRegistryBlobStore(),
       _accessCredentials = accessCredentials ?? ClientAccessCredentialStore(),
       _localDiscovery =
           localDiscovery ?? MethodChannelLocalDiscoveryPlatform(),
       _sshCredentials = sshCredentials ?? MethodChannelSSHCredentialPlatform();

  static const _sshCredentialRefPrefix = 'ssh-platform-';
  final EndpointRegistryBlobStore _registryStore;
  final ClientAccessCredentialStore _accessCredentials;
  final LocalDiscoveryPlatform _localDiscovery;
  final SSHCredentialPlatform _sshCredentials;

  @override
  Future<PlatformResponse> handle(PlatformRequest request) async {
    final response = PlatformResponse(requestId: request.requestId);
    try {
      switch (request.whichRequest()) {
        case PlatformRequest_Request.credentialPrepare:
          response.credential = await _accessCredentials.prepareRecord(
            request.credentialPrepare.credentialRef,
            request.credentialPrepare.endpointId,
          );
        case PlatformRequest_Request.credentialResolve:
          response.credential = await _accessCredentials.resolveRecord(
            request.credentialResolve.credentialRef,
            request.credentialResolve.endpointId,
          );
        case PlatformRequest_Request.credentialDelete:
          await _accessCredentials.delete(
            request.credentialDelete.credentialRef,
          );
        case PlatformRequest_Request.credentialSign:
          response.credentialSign = CredentialSignResponse(
            signature: await _accessCredentials.sign(
              request.credentialSign.credentialRef,
              request.credentialSign.payload,
            ),
          );
        case PlatformRequest_Request.credentialBind:
          final value = request.credentialBind;
          response.credential = await _accessCredentials.bindRecord(
            credentialRef: value.credentialRef,
            endpointId: value.endpointId,
            capabilityGrant: value.capabilityGrant,
            cloudRouteGrant: value.cloudRouteGrant,
            cloudEdgeLocator: value.cloudEdgeLocator,
          );
        case PlatformRequest_Request.endpointRegistryLoad:
          response.endpointRegistry = EndpointRegistryLoaded(
            registryProto: await _registryStore.load(),
          );
        case PlatformRequest_Request.endpointRegistryStore:
          await _storeRegistry(request.endpointRegistryStore);
        case PlatformRequest_Request.localDiscoveryLookup:
          final lookup = request.localDiscoveryLookup;
          response.localDiscovery = await _localDiscovery.lookup(
            lookup.deviceId,
            lookup.deviceFingerprint,
          );
        case PlatformRequest_Request.cloudProfileResolve:
          throw const ClientPlatformFailure(
            'protocol',
            'Cloud profile resolution is owned by Go',
          );
        case PlatformRequest_Request.sshCredentialLookup:
          final lookup = request.sshCredentialLookup;
          response.sshCredential = await _sshCredentials.lookup(
            lookup.credentialRef,
            createIfMissing: lookup.createIfMissing,
          );
        case PlatformRequest_Request.sshCredentialSign:
          final sign = request.sshCredentialSign;
          response.sshCredentialSign = SSHCredentialSignResponse(
            signature: await _sshCredentials.sign(
              sign.credentialRef,
              sign.digest,
              sign.hash,
            ),
          );
        case PlatformRequest_Request.sshCredentialDelete:
          await _sshCredentials.delete(
            request.sshCredentialDelete.credentialRef,
          );
        case PlatformRequest_Request.notSet:
          response.error = _error(
            ApiErrorCode.API_ERROR_CODE_INVALID_REQUEST,
            'platform request payload is missing',
          );
      }
    } on ClientPlatformFailure catch (failure) {
      response.error = _platformError(failure.code, failure.message);
    } on FormatException catch (error) {
      response.error = _error(
        ApiErrorCode.API_ERROR_CODE_INVALID_REQUEST,
        error.message,
      );
    } catch (_) {
      response.error = _error(
        ApiErrorCode.API_ERROR_CODE_UNAVAILABLE,
        'Flutter platform request failed',
        retryable: true,
      );
    }
    return response;
  }

  Future<void> _storeRegistry(EndpointRegistryStoreRequest request) async {
    final sshRefs = request.deleteCredentialRefs
        .where((ref) => ref.startsWith(_sshCredentialRefPrefix))
        .toList(growable: false);
    final accessRefs = request.deleteCredentialRefs.where(
      (ref) => !ref.startsWith(_sshCredentialRefPrefix),
    );

    final previous = await _registryStore.load();
    await _registryStore.store(request.registryProto);
    try {
      await _accessCredentials.deleteMany(accessRefs);
      await _sshCredentials.deleteMany(sshRefs);
    } catch (_) {
      if (previous.isEmpty) {
        await _registryStore.clear();
      } else {
        await _registryStore.store(previous);
      }
      rethrow;
    }
  }

  ApiError _platformError(String failureCode, String message) {
    final code = switch (failureCode) {
      'protocol' => ApiErrorCode.API_ERROR_CODE_INVALID_REQUEST,
      'unauthenticated' ||
      'login_required' ||
      'capability_invalid' ||
      'capability_expired' ||
      'identity_conflict' => ApiErrorCode.API_ERROR_CODE_UNAUTHORIZED,
      'quota_exhausted' => ApiErrorCode.API_ERROR_CODE_CONFLICT,
      'entitlement_denied' => ApiErrorCode.API_ERROR_CODE_ENTITLEMENT_DENIED,
      'cancelled' => ApiErrorCode.API_ERROR_CODE_CANCELLED,
      'route_unavailable' ||
      'temporary' ||
      'companion_missing' ||
      'backpressure' => ApiErrorCode.API_ERROR_CODE_UNAVAILABLE,
      'unsupported' => ApiErrorCode.API_ERROR_CODE_UNSUPPORTED_CAPABILITY,
      _ => ApiErrorCode.API_ERROR_CODE_INTERNAL,
    };
    return _error(
      code,
      message,
      retryable:
          code == ApiErrorCode.API_ERROR_CODE_UNAVAILABLE ||
          failureCode == 'quota_exhausted',
    );
  }

  ApiError _error(ApiErrorCode code, String message, {bool retryable = false}) {
    return ApiError(
      code: code,
      message: message,
      retryable: retryable,
      attempted: true,
    );
  }
}
