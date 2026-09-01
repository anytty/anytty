import 'dart:async';

import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'anytty_runtime.dart';

final class BindingOperationCancelledException implements Exception {
  const BindingOperationCancelledException();

  @override
  String toString() => 'Binding operation was cancelled';
}

Future<T> runBindingOperation<T>({
  required AnyttyEngineRuntime runtime,
  required int Function() begin,
  required T? Function(EventEnvelope event) select,
  required int Function(T result) operationHandle,
  required String timeoutMessage,
  Duration timeout = const Duration(seconds: 30),
  Future<void>? cancelWhen,
}) async {
  final completer = Completer<T>();
  int? handle;
  final early = <T>[];
  Object? cancellationError;
  Timer? timeoutTimer;

  void accept(T result) {
    if (handle == null) {
      early.add(result);
      return;
    }
    if (operationHandle(result) == handle && !completer.isCompleted) {
      completer.complete(result);
    }
  }

  final subscription = runtime.events.listen((event) {
    final result = select(event);
    if (result != null) accept(result);
  }, onError: completer.completeError);

  try {
    handle = begin();
    for (final result in early) {
      accept(result);
    }

    void requestCancellation(Object error) {
      if (completer.isCompleted || cancellationError != null) return;
      cancellationError = error;
      try {
        runtime.cancel(handle!);
      } catch (_) {
        // A completed native operation can win the race with this callback.
        // Its terminal event still owns release ordering.
      }
    }

    if (cancelWhen case final cancellation?) {
      unawaited(
        cancellation.then(
          (_) =>
              requestCancellation(const BindingOperationCancelledException()),
        ),
      );
    }
    timeoutTimer = Timer(
      timeout,
      () => requestCancellation(TimeoutException(timeoutMessage, timeout)),
    );
    final result = await completer.future;
    if (cancellationError case final error?) throw error;
    return result;
  } finally {
    timeoutTimer?.cancel();
    await subscription.cancel();
    if (handle != null) runtime.release(handle);
  }
}
