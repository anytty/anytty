import 'package:anytty_native/src/features/endpoints/domain/route_management.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('orders, moves, and removes routes with stable priorities', () {
    final endpoint = _endpoint();

    final moved = moveEndpointRoute(endpoint, 'ssh', -1);
    expect(
      orderedEndpointRoutes(moved)
          .map((route) => (route.routeId, route.priority)),
      [('ssh', 10), ('direct', 20)],
    );
    expect(
      moved.routes.every(
        (route) => route.policySource == EndpointSource.ENDPOINT_SOURCE_USER,
      ),
      isTrue,
    );

    final one = removeEndpointRoute(moved, 'direct');
    expect(one.routes.single.routeId, 'ssh');
    expect(removeEndpointRoute(one, 'ssh').routes.single.routeId, 'ssh');
  });

  test(
    'creates unique Direct and SSH drafts with safe credential metadata',
    () {
      final endpoint = _endpoint();
      final direct = newEndpointRoute(endpoint, EndpointRouteKind.direct);
      final ssh = newEndpointRoute(endpoint, EndpointRouteKind.ssh);

      expect(direct.routeId, 'direct-2');
      expect(direct.whichRoute(), EndpointRouteConfigV1_Route.directWebrtcTcp);
      expect(ssh.routeId, 'ssh-2');
      expect(ssh.sshWebrtcTcp.port, 22);
      expect(
        ssh.sshWebrtcTcp.credentialDescriptor.kind,
        EndpointCredentialKind.ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY,
      );
      expect(ssh.sshWebrtcTcp.sshCredentialRef, isEmpty);
    },
  );

  test('validates Direct and SSH fields before sending them to Go', () {
    final endpoint = _endpoint();
    final direct = newEndpointRoute(endpoint, EndpointRouteKind.direct);
    expect(
      validateEndpointRoute(endpoint, direct, isNew: true),
      contains('Direct routes require'),
    );
    direct.directWebrtcTcp
      ..signalingAddresses.add('direct.example.test:443')
      ..iceTcpAddresses.add('direct.example.test:443');
    expect(validateEndpointRoute(endpoint, direct, isNew: true), isNull);

    final ssh = newEndpointRoute(endpoint, EndpointRouteKind.ssh);
    expect(
      validateEndpointRoute(endpoint, ssh, isNew: true),
      contains('SSH routes require'),
    );
    ssh.sshWebrtcTcp
      ..host = 'ssh.example.test'
      ..user = 'anytty'
      ..hostKeyFingerprints.add('SHA256:test');
    expect(
      validateEndpointRoute(endpoint, ssh, isNew: true),
      contains('remote AnyTTY addresses'),
    );
    ssh.sshWebrtcTcp
      ..remoteSignalingAddress = '127.0.0.1:41120'
      ..remoteIceTcpAddress = '127.0.0.1:41120';
    expect(validateEndpointRoute(endpoint, ssh, isNew: true), isNull);
  });

  test(
    'splits route lists, removes blanks, and preserves first occurrence',
    () {
      expect(splitRouteValues('one:1, two:2\none:1\n\nthree:3'), [
        'one:1',
        'two:2',
        'three:3',
      ]);
    },
  );
}

EndpointConfigV1 _endpoint() => EndpointConfigV1(
  schemaVersion: 1,
  endpointId: 'studio',
  routes: [
    EndpointRouteConfigV1(
      schemaVersion: 1,
      routeId: 'direct',
      priority: 10,
      enabled: true,
      directWebrtcTcp: DirectWebRTCTCPRouteConfig(
        signalingAddresses: ['192.0.2.10:41120'],
        iceTcpAddresses: ['192.0.2.10:41120'],
      ),
    ),
    EndpointRouteConfigV1(
      schemaVersion: 1,
      routeId: 'ssh',
      priority: 20,
      enabled: true,
      sshWebrtcTcp: SSHWebRTCTCPRouteConfig(
        host: 'ssh.example.test',
        port: 22,
        user: 'anytty',
        hostKeyFingerprints: ['SHA256:host'],
      ),
    ),
  ],
);
