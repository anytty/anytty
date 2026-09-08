import 'dart:async';
import 'dart:developer' as developer;
import 'dart:isolate';
import 'dart:typed_data';

import 'package:flutter/foundation.dart' show debugPrint;

import 'anytty_client_engine.dart';

/// Serializes blocking resource IO off the UI isolate with a bounded mailbox.
final class NativeResourceWriter {
  NativeResourceWriter._();

  static const maximumPending = 64;
  static const maximumFrameBytes = 128 * 1024;
  final _responses = ReceivePort('anytty-resource-writer-responses');
  final _errors = ReceivePort('anytty-resource-writer-errors');
  final _exits = ReceivePort('anytty-resource-writer-exits');
  final _ready = Completer<SendPort>();
  final _pending = <int, Completer<void>>{};
  final _timings = <int, _WriterTiming>{};
  Isolate? _isolate;
  SendPort? _commands;
  int _sequence = 0;
  bool _closed = false;

  static Future<NativeResourceWriter> start(int engineHandle) async {
    final writer = NativeResourceWriter._();
    writer._responses.listen(writer._accept);
    writer._errors.listen(
      (_) => writer._fail(StateError('resource writer failed')),
    );
    writer._exits.listen(
      (_) => writer._fail(StateError('resource writer exited')),
    );
    try {
      writer._isolate = await Isolate.spawn(
        _runWriter,
        _WriterStart(engineHandle, writer._responses.sendPort),
        onError: writer._errors.sendPort,
        onExit: writer._exits.sendPort,
        debugName: 'anytty-resource-writer',
      );
      writer._commands = await writer._ready.future.timeout(
        const Duration(seconds: 5),
      );
      return writer;
    } catch (_) {
      writer.dispose();
      rethrow;
    }
  }

  Future<void> send(int streamHandle, Uint8List frame) =>
      _submit(streamHandle, frame);
  Future<void> closeStream(int streamHandle) => _submit(streamHandle, null);

  Future<void> _submit(int handle, Uint8List? frame) {
    if (frame != null && frame.length > maximumFrameBytes) {
      return Future<void>.error(
        ArgumentError('resource writer frame exceeds bounded chunk size'),
      );
    }
    if (_closed || _commands == null) {
      return Future<void>.error(StateError('resource writer is closed'));
    }
    if (_pending.length >= maximumPending) {
      return Future<void>.error(StateError('resource writer queue is full'));
    }
    final id = ++_sequence;
    final done = Completer<void>();
    _pending[id] = done;
    _commands!.send(
      _WriterCommand(
        id,
        handle,
        frame == null ? null : TransferableTypedData.fromList([frame]),
        developer.Timeline.now,
      ),
    );
    return done.future;
  }

  void _accept(dynamic message) {
    if (message is SendPort) {
      if (!_ready.isCompleted) _ready.complete(message);
    } else if (message is _WriterResult) {
      final pending = _pending.remove(message.id);
      if (pending == null) return;
      // Remote closure may release a resource without a close command here.
      // Bound diagnostic state independently of resource lifetime.
      if (!_timings.containsKey(message.handle) &&
          _timings.length >= maximumPending) {
        _timings.remove(_timings.keys.first);
      }
      final timing = _timings.putIfAbsent(message.handle, _WriterTiming.new);
      timing.count++;
      timing.queueUs += message.queueUs;
      timing.nativeUs += message.nativeUs;
      timing.returnUs += developer.Timeline.now - message.completedUs;
      if (message.closing || message.error != null || timing.count >= 64) {
        final entry =
            'stream=${message.handle} writes=${timing.count} '
            'queue_us=${timing.queueUs} native_us=${timing.nativeUs} '
            'return_us=${timing.returnUs} closing=${message.closing} '
            'failed=${message.error != null}';
        developer.log(entry, name: 'anytty.resource.writer');
        debugPrint('anytty.resource.writer $entry');
        _timings.remove(message.handle);
      }
      if (message.error == null) {
        pending.complete();
      } else {
        pending.completeError(StateError(message.error!));
      }
    }
  }

  void _fail(Object error) {
    if (_closed) return;
    _closed = true;
    if (!_ready.isCompleted) _ready.completeError(error);
    for (final pending in _pending.values) {
      pending.completeError(error);
    }
    _pending.clear();
    _timings.clear();
  }

  void dispose() {
    _fail(StateError('resource writer disposed'));
    _isolate?.kill(priority: Isolate.immediate);
    _responses.close();
    _errors.close();
    _exits.close();
  }
}

final class _WriterStart {
  const _WriterStart(this.engineHandle, this.responses);
  final int engineHandle;
  final SendPort responses;
}

final class _WriterCommand {
  const _WriterCommand(this.id, this.handle, this.frame, this.submittedUs);
  final int id;
  final int handle;
  final TransferableTypedData? frame;
  final int submittedUs;
}

final class _WriterTiming {
  int count = 0;
  int queueUs = 0;
  int nativeUs = 0;
  int returnUs = 0;
}

final class _WriterResult {
  const _WriterResult(
    this.id,
    this.handle,
    this.closing,
    this.error,
    this.queueUs,
    this.nativeUs,
    this.completedUs,
  );
  final int id;
  final int handle;
  final bool closing;
  final String? error;
  final int queueUs;
  final int nativeUs;
  final int completedUs;
}

void _runWriter(_WriterStart start) {
  final engine = AnyttyClientEngine.attach(start.engineHandle);
  final commands = ReceivePort('anytty-resource-writer-commands');
  start.responses.send(commands.sendPort);
  commands.listen((dynamic message) {
    if (message is! _WriterCommand) return;
    final startedUs = developer.Timeline.now;
    String? error;
    try {
      final frame = message.frame;
      if (frame == null) {
        engine.closeResourceStream(message.handle);
      } else {
        engine.sendResourceStreamFrame(
          message.handle,
          frame.materialize().asUint8List(),
        );
      }
    } catch (failure) {
      error = failure.toString();
    }
    final completedUs = developer.Timeline.now;
    start.responses.send(
      _WriterResult(
        message.id,
        message.handle,
        message.frame == null,
        error,
        startedUs - message.submittedUs,
        completedUs - startedUs,
        completedUs,
      ),
    );
  });
}
