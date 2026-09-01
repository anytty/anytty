import 'dart:async';

import 'package:anytty_native/src/features/endpoints/data/endpoint_repository.dart';
import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    show CommandEnvelope;
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'matches registry results by operation handle and releases them',
    () async {
      final runtime = _FakeRuntime();
      final result = await EndpointRepository(runtime).getRegistry();

      expect(result.defaultEndpointId, 'studio');
      expect(runtime.released, [41]);
      expect(
        runtime.lastCommand?.whichCommand(),
        EngineCommand_Command.endpointRegistryGet,
      );
    },
  );

  test('imports pairing through Go and returns the updated registry', () async {
    final runtime = _FakeRuntime();
    final result = await EndpointRepository(runtime)
        .importPairing('anytty-pairing-payload', expectedEndpointId: 'studio');

    expect(result.endpoint.endpointId, 'studio');
    expect(result.registry.defaultEndpointId, 'studio');
    expect(runtime.released, [73]);
    expect(
      runtime.lastCommand?.whichCommand(),
      EngineCommand_Command.importPairing,
    );
    expect(
      runtime.lastCommand?.importPairing.portablePayload,
      'anytty-pairing-payload',
    );
    expect(runtime.lastCommand?.importPairing.expectedEndpointId, 'studio');
  });

  test(
    'previews and commits endpoint shares through separate Go operations',
    () async {
      final runtime = _FakeRuntime();
      final repository = EndpointRepository(runtime);

      final preview = await repository.receiveEndpointShare(
        'anytty://share?payload=test-offer',
      );
      expect(preview.endpointId, 'shared-studio');
      expect(preview.routeDiffs.single.action, 'add');
      expect(runtime.released, [86]);
      expect(
        runtime.lastCommand?.whichCommand(),
        EngineCommand_Command.endpointShareReceive,
      );

      final imported = await repository.commitEndpointShare(
        preview.importToken,
      );
      expect(imported.endpoint.endpointId, 'shared-studio');
      expect(imported.authorizationRequired, isTrue);
      expect(runtime.released, [86, 87]);
      expect(
        runtime.lastCommand?.whichCommand(),
        EngineCommand_Command.endpointShareCommit,
      );
      expect(runtime.lastCommand?.endpointShareCommit.importToken, 'preview-1');
    },
  );

  test('probes Cloud presence without opening a terminal session', () async {
    final runtime = _FakeRuntime();
    final presence = await EndpointRepository(runtime)
        .getCloudPresence('studio');

    expect(presence.online, isTrue);
    expect(presence.edgeRegion, 'cn-east');
    expect(
      runtime.lastCommand?.whichCommand(),
      EngineCommand_Command.endpointCloudPresenceGet,
    );
    expect(runtime.released, [85]);
  });

  test('updates endpoint metadata and can make it the default', () async {
    final runtime = _FakeRuntime();
    final registry = await EndpointRepository(runtime).upsertEndpoint(
      EndpointConfigV1(endpointId: 'studio', label: 'Build Mac'),
      makeDefault: true,
    );

    expect(registry.defaultEndpointId, 'studio');
    expect(
      runtime.lastCommand?.whichCommand(),
      EngineCommand_Command.endpointUpsert,
    );
    expect(runtime.lastCommand?.endpointUpsert.endpoint.label, 'Build Mac');
    expect(runtime.lastCommand?.endpointUpsert.makeDefault, isTrue);
    expect(runtime.released, [81]);
  });

  test('disconnects and removes an endpoint through Go operations', () async {
    final runtime = _FakeRuntime();
    final repository = EndpointRepository(runtime);

    await repository.disconnectEndpoint('studio');
    expect(
      runtime.lastCommand?.whichCommand(),
      EngineCommand_Command.endpointDisconnect,
    );
    expect(runtime.released, [82]);

    final registry = await repository.deleteEndpoint('studio');
    expect(
      runtime.lastCommand?.whichCommand(),
      EngineCommand_Command.endpointDelete,
    );
    expect(registry.endpoints, isEmpty);
    expect(runtime.released, [82, 83]);
  });

  test(
    'prepares an SSH key through the platform-backed Go operation',
    () async {
      final runtime = _FakeRuntime();
      final result = await EndpointRepository(runtime)
          .provisionSshCredential('studio', 'ssh-office');

      expect(result.authorizedKey, 'ssh-ed25519 AAAATEST anytty');
      expect(result.keyFingerprint, 'SHA256:client');
      expect(
        runtime.lastCommand?.whichCommand(),
        EngineCommand_Command.sshCredentialProvision,
      );
      expect(runtime.released, [84]);
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
    if (command.whichCommand() == EngineCommand_Command.endpointShareReceive) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            endpointShareReceive: EndpointShareReceiveResult(
              requestId: command.endpointShareReceive.requestId,
              operationHandle: Int64(86),
              preview: EndpointSharePreview(
                importToken: 'preview-1',
                endpointId: 'shared-studio',
                label: 'Shared Studio',
                identity: EndpointDaemonIdentity(
                  deviceId: 'shared-studio',
                  deviceFingerprint: 'SHA256:shared',
                ),
                routeDiffs: [
                  EndpointShareRouteDiff(
                    routeId: 'direct-office',
                    routeKind: 'direct',
                    action: 'add',
                  ),
                ],
              ),
            ),
          ),
        );
      });
      return 86;
    }
    if (command.whichCommand() == EngineCommand_Command.endpointShareCommit) {
      scheduleMicrotask(() {
        final endpoint = EndpointConfigV1(
          endpointId: 'shared-studio',
          label: 'Shared Studio',
        );
        _events.add(
          EventEnvelope(
            endpointShareCommit: EndpointShareCommitResult(
              requestId: command.endpointShareCommit.requestId,
              operationHandle: Int64(87),
              endpoint: endpoint,
              registry: EndpointRegistryV1(
                schemaVersion: 1,
                endpoints: [endpoint],
              ),
              authorizationRequired: true,
            ),
          ),
        );
      });
      return 87;
    }
    if (command.whichCommand() ==
        EngineCommand_Command.endpointCloudPresenceGet) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            endpointCloudPresenceGet: EndpointCloudPresenceGetResult(
              requestId: command.endpointCloudPresenceGet.requestId,
              operationHandle: Int64(85),
              endpointId: command.endpointCloudPresenceGet.endpointId,
              online: true,
              edgeRegion: 'cn-east',
              locatorSource: 'cached_edge',
            ),
          ),
        );
      });
      return 85;
    }
    if (command.whichCommand() == EngineCommand_Command.endpointUpsert) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            endpointUpsert: EndpointUpsertResult(
              requestId: command.endpointUpsert.requestId,
              operationHandle: Int64(81),
              endpoint: command.endpointUpsert.endpoint.deepCopy(),
              registry: EndpointRegistryV1(
                schemaVersion: 1,
                defaultEndpointId: command.endpointUpsert.endpoint.endpointId,
                endpoints: [command.endpointUpsert.endpoint.deepCopy()],
              ),
            ),
          ),
        );
      });
      return 81;
    }
    if (command.whichCommand() == EngineCommand_Command.endpointDisconnect) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            endpointDisconnect: EndpointDisconnectResult(
              requestId: command.endpointDisconnect.requestId,
              operationHandle: Int64(82),
              endpointId: command.endpointDisconnect.endpointId,
            ),
          ),
        );
      });
      return 82;
    }
    if (command.whichCommand() == EngineCommand_Command.endpointDelete) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            endpointDelete: EndpointDeleteResult(
              requestId: command.endpointDelete.requestId,
              operationHandle: Int64(83),
              endpointId: command.endpointDelete.endpointId,
              registry: EndpointRegistryV1(schemaVersion: 1),
            ),
          ),
        );
      });
      return 83;
    }
    if (command.whichCommand() ==
        EngineCommand_Command.sshCredentialProvision) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            sshCredentialProvision: SSHCredentialProvisionResult(
              requestId: command.sshCredentialProvision.requestId,
              operationHandle: Int64(84),
              endpoint: EndpointConfigV1(
                endpointId: 'studio',
                routes: [
                  EndpointRouteConfigV1(
                    routeId: command.sshCredentialProvision.routeId,
                    sshWebrtcTcp: SSHWebRTCTCPRouteConfig(),
                  ),
                ],
              ),
              registry: EndpointRegistryV1(schemaVersion: 1),
              credentialRef: 'ssh-platform-test',
              authorizedKey: 'ssh-ed25519 AAAATEST anytty',
              keyFingerprint: 'SHA256:client',
            ),
          ),
        );
      });
      return 84;
    }
    if (command.whichCommand() == EngineCommand_Command.importPairing) {
      scheduleMicrotask(() {
        _events.add(
          EventEnvelope(
            importPairing: ImportPairingResult(
              requestId: command.importPairing.requestId,
              operationHandle: Int64(73),
              endpoint: EndpointConfigV1(endpointId: 'studio', label: 'Studio'),
              registry: EndpointRegistryV1(
                schemaVersion: 1,
                defaultEndpointId: 'studio',
              ),
            ),
          ),
        );
      });
      return 73;
    }
    scheduleMicrotask(() {
      _events.add(
        EventEnvelope(
          endpointRegistryGet: EndpointRegistryGetResult(
            requestId: command.endpointRegistryGet.requestId,
            operationHandle: Int64(41),
            registry: EndpointRegistryV1(
              schemaVersion: 1,
              defaultEndpointId: 'studio',
            ),
          ),
        ),
      );
    });
    return 41;
  }

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
