import 'dart:async';
import 'dart:collection';

import '../generated/proto/bindingpb/client_binding.pb.dart';

// Mutations are FIFO barriers; independent reads/signatures can overlap.
final class PlatformRequestQueue {
  PlatformRequestQueue(this.handle, {this.log});

  final Future<PlatformResponse> Function(PlatformRequest) handle;
  final void Function(String)? log;
  final Queue<_PendingRequest> _pending = Queue();
  int _active = 0;
  bool _exclusive = false;
  bool _closed = false;
  static const concurrency = 4;

  Future<PlatformResponse> submit(PlatformRequest request) {
    if (_closed) {
      return Future.error(StateError('Platform request queue is closed'));
    }
    final pending = _PendingRequest(request);
    _pending.add(pending);
    _drain();
    return pending.result.future;
  }

  void close() {
    _closed = true;
    while (_pending.isNotEmpty) {
      _pending.removeFirst().result.completeError(
        StateError('Platform request queue is closed'),
      );
    }
  }

  void _drain() {
    while (!_exclusive && _active < concurrency && _pending.isNotEmpty) {
      final next = _pending.first;
      final exclusive = !_canOverlap(next.request);
      if (exclusive && _active != 0) return;
      _pending.removeFirst();
      _active++;
      _exclusive = exclusive;
      final queueMs = next.clock.elapsedMilliseconds;
      final work = Stopwatch()..start();
      var success = false;
      unawaited(
        Future<PlatformResponse>.sync(() => handle(next.request))
            .then<void>(
              (response) {
                success = !response.hasError();
                next.result.complete(response);
              },
              onError: (Object error, StackTrace stack) {
                next.result.completeError(error, stack);
              },
            )
            .whenComplete(() {
              log?.call(
                'anytty connect component=platform_handler '
                'request_id=${next.request.requestId} '
                'operation=${next.request.whichRequest().name} '
                'queue_ms=$queueMs handler_ms=${work.elapsedMilliseconds} '
                'success=$success',
              );
              _active--;
              if (exclusive) _exclusive = false;
              _drain();
            }),
      );
    }
  }

  static bool _canOverlap(PlatformRequest request) =>
      switch (request.whichRequest()) {
        PlatformRequest_Request.credentialResolve ||
        PlatformRequest_Request.credentialSign ||
        PlatformRequest_Request.sshCredentialSign ||
        PlatformRequest_Request.localDiscoveryLookup ||
        PlatformRequest_Request.cloudProfileResolve => true,
        _ => false,
      };
}

final class _PendingRequest {
  _PendingRequest(this.request);
  final PlatformRequest request;
  final result = Completer<PlatformResponse>();
  final clock = Stopwatch()..start();
}
