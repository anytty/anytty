import 'dart:async';

import 'package:fixnum/fixnum.dart';

import '../generated/proto/bindingpb/client_binding.pb.dart';
import 'anytty_runtime.dart';

final class AnyttyResourceStream {
  AnyttyResourceStream._(
    this._runtime,
    this.handle,
    this._frames,
    this._closed,
  );

  final AnyttyEngineRuntime _runtime;
  final int handle;
  final StreamController<ResourceStreamFrame> _frames;
  final Completer<ResourceStreamClosedEvent> _closed;
  bool _closeRequested = false;
  bool _released = false;

  Stream<ResourceStreamFrame> get frames => _frames.stream;

  Future<ResourceStreamClosedEvent> get closed => _closed.future;

  static Future<AnyttyResourceStream> open({
    required AnyttyEngineRuntime runtime,
    required int sessionHandle,
    required OpenResourceStreamRequest request,
  }) async {
    if (runtime is! AnyttyResourceStreamRuntime) {
      throw UnsupportedError(
        'AnyTTY runtime does not support resource streams',
      );
    }
    final native = runtime as AnyttyResourceStreamRuntime;
    final frames = StreamController<ResourceStreamFrame>();
    final closed = Completer<ResourceStreamClosedEvent>();
    final earlyEvents = <EventEnvelope>[];
    int? streamHandle;
    var released = false;

    void releaseHandle() {
      final handle = streamHandle;
      if (handle == null || released) return;
      released = true;
      try {
        runtime.release(handle);
      } catch (_) {
        // Runtime shutdown or renderer replacement can revoke it first.
      }
    }

    void accept(EventEnvelope event) {
      final handle = streamHandle;
      if (handle == null) {
        earlyEvents.add(event.deepCopy());
        return;
      }
      switch (event.whichEvent()) {
        case EventEnvelope_Event.resourceStreamFrame:
          if (event.resourceStreamFrame.streamHandle.toInt() == handle &&
              !frames.isClosed) {
            frames.add(event.resourceStreamFrame.deepCopy());
          }
        case EventEnvelope_Event.resourceStreamClosed:
          if (event.resourceStreamClosed.streamHandle.toInt() == handle &&
              !closed.isCompleted) {
            closed.complete(event.resourceStreamClosed.deepCopy());
          }
        default:
          break;
      }
    }

    late final StreamSubscription<EventEnvelope> subscription;
    subscription = runtime.events.listen(
      accept,
      onError: (Object error, StackTrace stackTrace) {
        if (!closed.isCompleted) closed.completeError(error, stackTrace);
      },
      onDone: () {
        final handle = streamHandle;
        if (!closed.isCompleted) {
          closed.complete(
            ResourceStreamClosedEvent(streamHandle: Int64(handle ?? 0)),
          );
        }
      },
    );
    try {
      final handle = native.openResourceStream(sessionHandle, request);
      streamHandle = handle;
      for (final event in earlyEvents) {
        accept(event);
      }
      earlyEvents.clear();
      final stream = AnyttyResourceStream._(runtime, handle, frames, closed);
      unawaited(
        closed.future.then<void>(
          (_) async {
            stream._released = released;
            stream._release();
            await frames.close();
            await subscription.cancel();
          },
          onError: (Object _, StackTrace _) async {
            stream._released = released;
            stream._release();
            await frames.close();
            await subscription.cancel();
          },
        ),
      );
      return stream;
    } catch (_) {
      releaseHandle();
      await subscription.cancel();
      await frames.close();
      rethrow;
    }
  }

  void send(ResourceStreamFrameType type, List<int> payload) {
    if (_closeRequested || _closed.isCompleted) {
      throw StateError('AnyTTY resource stream is closed');
    }
    final native = _runtime as AnyttyResourceStreamRuntime;
    native.sendResourceStreamFrame(
      handle,
      ResourceStreamFrame(
        streamHandle: Int64(handle),
        type: type,
        payload: payload,
      ),
    );
  }

  void close() {
    if (_closeRequested || _closed.isCompleted) return;
    _closeRequested = true;
    try {
      (_runtime as AnyttyResourceStreamRuntime).closeResourceStream(handle);
    } catch (error, stackTrace) {
      _release();
      if (!_closed.isCompleted) {
        _closed.completeError(error, stackTrace);
      }
      rethrow;
    }
  }

  void _release() {
    if (_released) return;
    _released = true;
    try {
      _runtime.release(handle);
    } catch (_) {
      // Runtime shutdown or renderer replacement can revoke it first.
    }
  }
}
