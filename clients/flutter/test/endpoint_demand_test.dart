import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('publishes sorted endpoint demand with reference-counted release', () {
    final snapshots = <EndpointSupervisorDemandSnapshot>[];
    final demand = EndpointDemandCoordinator(
      attachmentId: 'flutter-test',
      replace: (snapshot) => snapshots.add(snapshot.deepCopy()),
    );

    final beta = demand.retain('beta');
    final alpha = demand.retain('alpha');
    final alphaAgain = demand.retain('alpha');

    expect(snapshots, hasLength(2));
    expect(snapshots.last.endpoints.map((entry) => entry.endpointId), [
      'alpha',
      'beta',
    ]);
    expect(snapshots.last.endpoints.map((entry) => entry.mode).toSet(), {
      EndpointSupervisorMode.ENDPOINT_SUPERVISOR_MODE_TAKEOVER,
    });

    alpha.release();
    expect(snapshots, hasLength(2));
    alphaAgain.release();
    expect(snapshots.last.endpoints.map((entry) => entry.endpointId), ['beta']);
    beta.release();
    expect(snapshots.last.endpoints, isEmpty);
    expect(snapshots.map((snapshot) => snapshot.demandRevision.toInt()), [
      1,
      2,
      3,
      4,
    ]);
  });

  test('rolls back a demand mutation rejected by the native sink', () {
    var reject = true;
    final snapshots = <EndpointSupervisorDemandSnapshot>[];
    final demand = EndpointDemandCoordinator(
      attachmentId: 'flutter-test',
      replace: (snapshot) {
        if (reject) throw StateError('rejected');
        snapshots.add(snapshot.deepCopy());
      },
    );

    expect(() => demand.retain('alpha'), throwsStateError);
    reject = false;
    final lease = demand.retain('beta');

    expect(snapshots.single.endpoints.map((entry) => entry.endpointId), [
      'beta',
    ]);
    lease.release();
  });
}
