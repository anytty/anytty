import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/platform_request_queue.dart';

void main() {
  test('reads overlap, mutation is a barrier, later reads wait', () async {
    final started = <String>[];
    final gates = <Completer<PlatformResponse>>[];
    final queue = PlatformRequestQueue((request) {
      started.add(request.whichRequest().name);
      final gate = Completer<PlatformResponse>();
      gates.add(gate);
      return gate.future;
    });
    final a = queue.submit(
      PlatformRequest(credentialResolve: CredentialResolveRequest()),
    );
    final b = queue.submit(
      PlatformRequest(credentialSign: CredentialSignRequest()),
    );
    final mutation = queue.submit(
      PlatformRequest(credentialDelete: CredentialDeleteRequest()),
    );
    final after = queue.submit(
      PlatformRequest(credentialResolve: CredentialResolveRequest()),
    );
    expect(started, ['credentialResolve', 'credentialSign']);
    gates[0].complete(PlatformResponse());
    await a;
    await Future<void>.delayed(Duration.zero);
    expect(started.length, 2);
    gates[1].complete(PlatformResponse());
    await b;
    await Future<void>.delayed(Duration.zero);
    expect(started.last, 'credentialDelete');
    gates[2].complete(PlatformResponse());
    await mutation;
    await Future<void>.delayed(Duration.zero);
    expect(started.last, 'credentialResolve');
    gates[3].complete(PlatformResponse());
    await after;
    queue.close();
  });

  test('read concurrency is bounded and errors release slots', () async {
    final gates = <Completer<PlatformResponse>>[];
    final queue = PlatformRequestQueue((_) {
      final gate = Completer<PlatformResponse>();
      gates.add(gate);
      return gate.future;
    });
    final requests = List.generate(
      5,
      (_) => queue.submit(
        PlatformRequest(credentialSign: CredentialSignRequest()),
      ),
    );
    expect(gates.length, 4);
    final failed = expectLater(requests[0], throwsStateError);
    gates[0].completeError(StateError('sign failed'));
    await failed;
    await Future<void>.delayed(Duration.zero);
    expect(gates.length, 5);
    for (final gate in gates.skip(1)) {
      gate.complete(PlatformResponse());
    }
    await Future.wait(requests.skip(1));
    queue.close();
  });
}
