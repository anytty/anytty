import 'dart:async';

import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    show CommandEnvelope;
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:anytty_native/src/native/binding_operation.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('cancels and releases an in-flight binding operation', () async {
    final runtime = _PendingRuntime();
    final cancellation = Completer<void>();
    final pending = runBindingOperation<int>(
      runtime: runtime,
      begin: () => 51,
      select: (event) => event.whichEvent() == EventEnvelope_Event.execute
          ? event.execute.operationHandle.toInt()
          : null,
      operationHandle: (value) => value,
      timeoutMessage: 'operation timed out',
      cancelWhen: cancellation.future,
    );

    await Future<void>.delayed(Duration.zero);
    cancellation.complete();

    await expectLater(
      pending,
      throwsA(isA<BindingOperationCancelledException>()),
    );
    expect(runtime.cancelled, [51]);
    expect(runtime.released, [51]);
  });
}

final class _PendingRuntime implements AnyttyEngineRuntime {
  final StreamController<EventEnvelope> _events =
      StreamController<EventEnvelope>.broadcast();
  final List<int> cancelled = [];
  final List<int> released = [];

  @override
  Stream<EventEnvelope> get events => _events.stream;

  @override
  Stream<int> get foregroundResumes => const Stream<int>.empty();

  @override
  void cancel(int operationHandle) {
    cancelled.add(operationHandle);
    _events.add(
      EventEnvelope(
        execute: ExecuteResult(operationHandle: Int64(operationHandle)),
      ),
    );
  }

  @override
  void release(int handle) => released.add(handle);

  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) =>
      EndpointDemandLease(() {});

  @override
  int command(EngineCommand command) => throw UnimplementedError();

  @override
  void closeSession(int sessionHandle) => throw UnimplementedError();

  @override
  int execute(int sessionHandle, CommandEnvelope request) =>
      throw UnimplementedError();

  @override
  int openSession(OpenSessionRequest request) => throw UnimplementedError();
}
