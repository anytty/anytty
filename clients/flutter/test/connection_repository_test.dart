import 'dart:async';

import 'package:anytty_native/src/features/endpoints/data/connection_repository.dart';
import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    show CommandEnvelope;
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('reads and applies Go-owned connection policy', () async {
    final runtime = _FakeRuntime();
    final repository = ConnectionRepository(runtime);

    final initial = await repository.getPolicy('studio');
    expect(
      initial.policy.routePreference,
      EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO,
    );
    expect(initial.routes.single.available, isTrue);
    expect(runtime.released, [91]);

    final updated = await repository.applyPolicy(
      'studio',
      ConnectionPolicy(
        routePreference:
            EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT,
        cloudRelayMode:
            ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY,
        relayTransport:
            ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
      ),
    );
    expect(
      updated.policy.routePreference,
      EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT,
    );
    expect(
      runtime.lastCommand?.whichCommand(),
      EngineCommand_Command.connectionPolicyApply,
    );
    expect(runtime.released, [91, 92]);
  });

  test(
    'reads a snapshot for the exact session and releases the handle',
    () async {
      final runtime = _FakeRuntime();
      final snapshot = await ConnectionRepository(runtime).getSnapshot(21);

      expect(snapshot.connected, isTrue);
      expect(snapshot.routeId, 'direct-test');
      expect(snapshot.roundTripNanos.toInt(), 12000000);
      expect(
        runtime.lastCommand?.connectionSnapshotGet.sessionHandle.toInt(),
        21,
      );
      expect(runtime.released, [93]);
    },
  );
}

final class _FakeRuntime implements AnyttyEngineRuntime {
  final StreamController<EventEnvelope> _events =
      StreamController<EventEnvelope>.broadcast();
  final List<int> released = [];
  EngineCommand? lastCommand;

  @override
  Stream<EventEnvelope> get events => _events.stream;

  @override
  Stream<int> get foregroundResumes => const Stream<int>.empty();

  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) =>
      EndpointDemandLease(() {});

  @override
  int command(EngineCommand command) {
    lastCommand = command.deepCopy();
    switch (command.whichCommand()) {
      case EngineCommand_Command.connectionPolicyGet:
        scheduleMicrotask(
          () => _events.add(
            EventEnvelope(
              connectionPolicyGet: ConnectionPolicyGetResult(
                requestId: command.connectionPolicyGet.requestId,
                operationHandle: Int64(91),
                state: _policyState(
                  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO,
                ),
              ),
            ),
          ),
        );
        return 91;
      case EngineCommand_Command.connectionPolicyApply:
        scheduleMicrotask(
          () => _events.add(
            EventEnvelope(
              connectionPolicyApply: ConnectionPolicyApplyResult(
                requestId: command.connectionPolicyApply.requestId,
                operationHandle: Int64(92),
                state: _policyState(
                  command.connectionPolicyApply.policy.routePreference,
                ),
              ),
            ),
          ),
        );
        return 92;
      case EngineCommand_Command.connectionSnapshotGet:
        scheduleMicrotask(
          () => _events.add(
            EventEnvelope(
              connectionSnapshotGet: ConnectionSnapshotGetResult(
                requestId: command.connectionSnapshotGet.requestId,
                operationHandle: Int64(93),
                sessionHandle: command.connectionSnapshotGet.sessionHandle,
                connection: ConnectionSnapshot(
                  connected: true,
                  routeId: 'direct-test',
                  routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
                  roundTripNanos: Int64(12000000),
                ),
              ),
            ),
          ),
        );
        return 93;
      default:
        throw StateError('Unexpected command ${command.whichCommand()}');
    }
  }

  ConnectionPolicyState _policyState(EndpointRoutePreference preference) =>
      ConnectionPolicyState(
        policy: ConnectionPolicy(
          routePreference: preference,
          cloudRelayMode: ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO,
          relayTransport:
              ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO,
        ),
        routes: [
          ConnectionPolicyRouteAvailability(
            routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
            available: true,
            reason: ConnectionPolicyAvailabilityReason
                .CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE,
          ),
        ],
      );

  @override
  void release(int handle) => released.add(handle);

  @override
  void cancel(int operationHandle) {}

  @override
  void closeSession(int sessionHandle) => throw UnimplementedError();

  @override
  int execute(int sessionHandle, CommandEnvelope request) =>
      throw UnimplementedError();

  @override
  int openSession(OpenSessionRequest request) => throw UnimplementedError();
}
