import 'package:fixnum/fixnum.dart';

import '../../../generated/proto/apipb/common.pb.dart' show ApiError;
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../native/anytty_runtime.dart';
import '../../../native/binding_operation.dart';
import '../../../native/request_id.dart';

final class ConnectionOperationException implements Exception {
  const ConnectionOperationException(this.message);

  final String message;

  @override
  String toString() => message;
}

final class ConnectionRepository {
  const ConnectionRepository(this._runtime);

  final AnyttyEngineRuntime _runtime;

  Future<ConnectionPolicyState> getPolicy(String endpointId) async {
    final id = _requireEndpointId(endpointId);
    final result = await runBindingOperation<ConnectionPolicyGetResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          connectionPolicyGet: ConnectionPolicyGetRequest(
            requestId: newRequestId(),
            endpointId: id,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.connectionPolicyGet
          ? event.connectionPolicyGet.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Connection policy request timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Connection policy request failed',
    );
    if (!result.hasState() || !result.state.hasPolicy()) {
      throw const ConnectionOperationException(
        'Connection policy response was incomplete',
      );
    }
    return result.state.deepCopy();
  }

  Future<ConnectionPolicyState> applyPolicy(
    String endpointId,
    ConnectionPolicy policy,
  ) async {
    final id = _requireEndpointId(endpointId);
    final result = await runBindingOperation<ConnectionPolicyApplyResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          connectionPolicyApply: ConnectionPolicyApplyRequest(
            requestId: newRequestId(),
            endpointId: id,
            policy: policy,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.connectionPolicyApply
          ? event.connectionPolicyApply.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Connection policy update timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Connection policy update failed',
    );
    if (!result.hasState() || !result.state.hasPolicy()) {
      throw const ConnectionOperationException(
        'Connection policy update response was incomplete',
      );
    }
    return result.state.deepCopy();
  }

  Future<ConnectionSnapshot> getSnapshot(int sessionHandle) async {
    if (sessionHandle <= 0) {
      throw const ConnectionOperationException('Session handle is required');
    }
    final result = await runBindingOperation<ConnectionSnapshotGetResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          connectionSnapshotGet: ConnectionSnapshotGetRequest(
            requestId: newRequestId(),
            sessionHandle: Int64(sessionHandle),
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.connectionSnapshotGet
          ? event.connectionSnapshotGet.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Connection diagnostics request timed out',
      timeout: const Duration(seconds: 15),
    );
    _throwResultError(
      result.hasError(),
      result.error,
      'Connection diagnostics request failed',
    );
    if (result.sessionHandle.toInt() != sessionHandle ||
        !result.hasConnection()) {
      throw const ConnectionOperationException(
        'Connection diagnostics response was incomplete',
      );
    }
    return result.connection.deepCopy();
  }

  String _requireEndpointId(String endpointId) {
    final id = endpointId.trim();
    if (id.isEmpty) {
      throw const ConnectionOperationException('Endpoint id is required');
    }
    return id;
  }

  void _throwResultError(bool hasError, ApiError error, String fallback) {
    if (!hasError) return;
    throw ConnectionOperationException(
      error.message.isEmpty ? fallback : error.message,
    );
  }
}
