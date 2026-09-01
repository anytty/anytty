import '../../../generated/proto/apipb/common.pb.dart' show ApiError;
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../../native/anytty_runtime.dart';
import '../../../native/binding_operation.dart';
import '../../../native/request_id.dart';

final class AnyttyOperationException implements Exception {
  const AnyttyOperationException(this.message);

  final String message;

  @override
  String toString() => message;
}

final class EndpointRepository {
  const EndpointRepository(this._runtime);

  final AnyttyEngineRuntime _runtime;

  Future<EndpointRegistryV1> getRegistry() async {
    final result = await runBindingOperation<EndpointRegistryGetResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointRegistryGet: EndpointRegistryGetRequest(
            requestId: newRequestId(),
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointRegistryGet
          ? event.endpointRegistryGet.deepCopy()
          : null,
      operationHandle: (result) => result.operationHandle.toInt(),
      timeoutMessage: 'Endpoint registry request timed out',
      timeout: const Duration(seconds: 15),
    );
    if (result.hasError()) {
      throw AnyttyOperationException(
        result.error.message.isEmpty
            ? 'Endpoint registry request failed'
            : result.error.message,
      );
    }
    if (!result.hasRegistry()) {
      throw const AnyttyOperationException(
        'Endpoint registry response was incomplete',
      );
    }
    return result.registry.deepCopy();
  }

  Future<EndpointCloudPresenceGetResult> getCloudPresence(
    String endpointId,
  ) async {
    final id = _requireEndpointId(endpointId);
    final result = await runBindingOperation<EndpointCloudPresenceGetResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointCloudPresenceGet: EndpointCloudPresenceGetRequest(
            requestId: newRequestId(),
            endpointId: id,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointCloudPresenceGet
          ? event.endpointCloudPresenceGet.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Device presence request timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Device presence request failed',
    );
    return result.deepCopy();
  }

  Future<EndpointRegistryV1> upsertEndpoint(
    EndpointConfigV1 endpoint, {
    bool makeDefault = false,
  }) async {
    if (endpoint.endpointId.trim().isEmpty) {
      throw const AnyttyOperationException('Endpoint id is required');
    }
    final result = await runBindingOperation<EndpointUpsertResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointUpsert: EndpointUpsertRequest(
            requestId: newRequestId(),
            endpoint: endpoint,
            makeDefault: makeDefault,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointUpsert
          ? event.endpointUpsert.deepCopy()
          : null,
      operationHandle: (result) => result.operationHandle.toInt(),
      timeoutMessage: 'Endpoint update timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Endpoint update failed',
    );
    if (!result.hasRegistry()) {
      throw const AnyttyOperationException(
        'Endpoint update response was incomplete',
      );
    }
    return result.registry.deepCopy();
  }

  Future<void> disconnectEndpoint(String endpointId) async {
    final id = _requireEndpointId(endpointId);
    final result = await runBindingOperation<EndpointDisconnectResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointDisconnect: EndpointDisconnectRequest(
            requestId: newRequestId(),
            endpointId: id,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointDisconnect
          ? event.endpointDisconnect.deepCopy()
          : null,
      operationHandle: (result) => result.operationHandle.toInt(),
      timeoutMessage: 'Endpoint disconnect timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Endpoint disconnect failed',
    );
  }

  Future<EndpointRegistryV1> deleteEndpoint(String endpointId) async {
    final id = _requireEndpointId(endpointId);
    final result = await runBindingOperation<EndpointDeleteResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointDelete: EndpointDeleteRequest(
            requestId: newRequestId(),
            endpointId: id,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointDelete
          ? event.endpointDelete.deepCopy()
          : null,
      operationHandle: (result) => result.operationHandle.toInt(),
      timeoutMessage: 'Endpoint removal timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Endpoint removal failed',
    );
    if (!result.hasRegistry()) {
      throw const AnyttyOperationException(
        'Endpoint removal response was incomplete',
      );
    }
    return result.registry.deepCopy();
  }

  Future<EndpointSharePreview> receiveEndpointShare(
    String portableOffer,
  ) async {
    final offer = portableOffer.trim();
    if (offer.isEmpty) {
      throw const AnyttyOperationException('Endpoint share offer is required');
    }
    final result = await runBindingOperation<EndpointShareReceiveResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointShareReceive: EndpointShareReceiveRequest(
            requestId: newRequestId(),
            portableOffer: offer,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointShareReceive
          ? event.endpointShareReceive.deepCopy()
          : null,
      operationHandle: (result) => result.operationHandle.toInt(),
      timeoutMessage: 'Endpoint share preview timed out',
      timeout: const Duration(seconds: 45),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Endpoint share preview failed',
    );
    if (!result.hasPreview() || result.preview.importToken.trim().isEmpty) {
      throw const AnyttyOperationException(
        'Endpoint share preview was incomplete',
      );
    }
    return result.preview.deepCopy();
  }

  Future<EndpointShareCommitResult> commitEndpointShare(
    String importToken,
  ) async {
    final token = importToken.trim();
    if (token.isEmpty) {
      throw const AnyttyOperationException('Endpoint share token is required');
    }
    final result = await runBindingOperation<EndpointShareCommitResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          endpointShareCommit: EndpointShareCommitRequest(
            requestId: newRequestId(),
            importToken: token,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.endpointShareCommit
          ? event.endpointShareCommit.deepCopy()
          : null,
      operationHandle: (result) => result.operationHandle.toInt(),
      timeoutMessage: 'Endpoint share import timed out',
      timeout: const Duration(seconds: 30),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Endpoint share import failed',
    );
    if (!result.hasEndpoint() || !result.hasRegistry()) {
      throw const AnyttyOperationException(
        'Endpoint share import response was incomplete',
      );
    }
    return result.deepCopy();
  }

  Future<SSHCredentialProvisionResult> provisionSshCredential(
    String endpointId,
    String routeId,
  ) async {
    final id = _requireEndpointId(endpointId);
    final route = routeId.trim();
    if (route.isEmpty) {
      throw const AnyttyOperationException('Route id is required');
    }
    final result = await runBindingOperation<SSHCredentialProvisionResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          sshCredentialProvision: SSHCredentialProvisionRequest(
            requestId: newRequestId(),
            endpointId: id,
            routeId: route,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.sshCredentialProvision
          ? event.sshCredentialProvision.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'SSH key preparation timed out',
      timeout: const Duration(seconds: 30),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'SSH key preparation failed',
    );
    if (!result.hasEndpoint() ||
        !result.hasRegistry() ||
        result.authorizedKey.trim().isEmpty) {
      throw const AnyttyOperationException(
        'SSH key preparation response was incomplete',
      );
    }
    return result;
  }

  Future<ImportPairingResult> importPairing(
    String portablePayload, {
    String? expectedEndpointId,
  }) async {
    final payload = portablePayload.trim();
    if (payload.isEmpty) {
      throw const AnyttyOperationException('Pairing payload is required');
    }
    final expected = expectedEndpointId?.trim() ?? '';
    final result = await runBindingOperation<ImportPairingResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          importPairing: ImportPairingRequest(
            requestId: newRequestId(),
            portablePayload: payload,
            expectedEndpointId: expected,
          ),
        ),
      ),
      select: (event) => event.whichEvent() == EventEnvelope_Event.importPairing
          ? event.importPairing.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Pairing request timed out',
      timeout: const Duration(seconds: 45),
    );
    if (result.hasError()) {
      throw AnyttyOperationException(
        result.error.message.isEmpty
            ? 'Pairing request failed'
            : result.error.message,
      );
    }
    if (!result.hasEndpoint() || !result.hasRegistry()) {
      throw const AnyttyOperationException('Pairing response was incomplete');
    }
    return result;
  }

  String _requireEndpointId(String endpointId) {
    final id = endpointId.trim();
    if (id.isEmpty) {
      throw const AnyttyOperationException('Endpoint id is required');
    }
    return id;
  }

  void _throwResultError(bool hasError, ApiError error, String fallback) {
    if (!hasError) return;
    throw AnyttyOperationException(
      error.message.isEmpty ? fallback : error.message,
    );
  }
}
