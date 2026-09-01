import 'dart:ffi';
import 'dart:isolate';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';

import 'generated/anytty_client_bindings.dart';
import 'native_library_loader.dart';

final class AnyttyNativeException implements Exception {
  const AnyttyNativeException(this.operation, this.status);

  final String operation;
  final anytty_status_v1 status;

  @override
  String toString() => 'AnyTTY native $operation failed: ${status.name}';
}

final class AnyttyClientEngine {
  AnyttyClientEngine._(this._native, this._handle);

  final AnyttyClientNative _native;
  final int _handle;
  bool _closed = false;

  static AnyttyClientEngine open() {
    final native = _loadNative();
    final outHandle = calloc<anytty_handle_t>();
    try {
      final status = native.anytty_engine_create(outHandle);
      _check('engine_create', status);
      return AnyttyClientEngine._(native, outHandle.value);
    } finally {
      calloc.free(outHandle);
    }
  }

  /// Creates the process-wide Go engine without blocking Flutter's UI isolate.
  static Future<AnyttyClientEngine> openAsync() async {
    final handle = await Isolate.run(_openHandleForTransfer);
    return AnyttyClientEngine._(_loadNative(), handle);
  }

  static int _openHandleForTransfer() => open().handle;

  /// Attaches to a process-wide Go handle from a worker isolate.
  ///
  /// The attached wrapper never owns the handle and must not call [close].
  static AnyttyClientEngine attach(int handle) {
    if (handle == 0) throw ArgumentError.value(handle, 'handle');
    return AnyttyClientEngine._(_loadNative(), handle);
  }

  int get handle => _handle;

  int openSession(Uint8List request) {
    return _operation('open_session', request, (data, length, outHandle) {
      return _native.anytty_engine_open_session(
        _handle,
        data,
        length,
        outHandle,
      );
    });
  }

  int execute(int sessionHandle, Uint8List command) {
    return _operation('execute', command, (data, length, outHandle) {
      return _native.anytty_engine_execute(
        _handle,
        sessionHandle,
        data,
        length,
        outHandle,
      );
    });
  }

  int openResourceStream(int sessionHandle, Uint8List request) {
    return _operation('open_resource_stream', request, (
      data,
      length,
      outHandle,
    ) {
      return _native.anytty_engine_open_resource_stream(
        _handle,
        sessionHandle,
        data,
        length,
        outHandle,
      );
    });
  }

  void sendResourceStreamFrame(int streamHandle, Uint8List frame) {
    _payload('send_resource_stream_frame', frame, (data, length) {
      return _native.anytty_engine_send_resource_stream_frame(
        _handle,
        streamHandle,
        data,
        length,
      );
    });
  }

  void closeResourceStream(int streamHandle) {
    _ensureOpen();
    _check(
      'close_resource_stream',
      _native.anytty_engine_close_resource_stream(_handle, streamHandle),
    );
  }

  int command(Uint8List command) {
    return _operation('command', command, (data, length, outHandle) {
      return _native.anytty_engine_command(_handle, data, length, outHandle);
    });
  }

  void replaceSupervisorDemand(Uint8List demand) {
    _payload('supervisor_replace_demand', demand, (data, length) {
      return _native.anytty_supervisor_replace_demand(_handle, data, length);
    });
  }

  void signalSupervisor(Uint8List signal) {
    _payload('supervisor_signal', signal, (data, length) {
      return _native.anytty_supervisor_signal(_handle, data, length);
    });
  }

  void waitSupervisorReady({required int timeoutMillis}) {
    _ensureOpen();
    _check(
      'supervisor_wait_ready',
      _native.anytty_supervisor_wait_ready(_handle, timeoutMillis),
    );
  }

  Uint8List supervisorSnapshot() {
    _ensureOpen();
    final out = calloc<anytty_buffer_v1>();
    try {
      _check(
        'supervisor_snapshot',
        _native.anytty_supervisor_snapshot(_handle, out),
      );
      return _copyAndRelease(out.ref);
    } finally {
      calloc.free(out);
    }
  }

  Uint8List? nextEvent({int timeoutMillis = 0}) {
    _ensureOpen();
    final out = calloc<anytty_buffer_v1>();
    try {
      final status = _native.anytty_engine_next_event(
        _handle,
        timeoutMillis,
        out,
      );
      if (status == anytty_status_v1.ANYTTY_STATUS_CLOSED) return null;
      _check('next_event', status);
      return _copyAndRelease(out.ref);
    } finally {
      calloc.free(out);
    }
  }

  Uint8List? nextPlatformRequest({int timeoutMillis = 0}) {
    _ensureOpen();
    final out = calloc<anytty_buffer_v1>();
    try {
      final status = _native.anytty_platform_next_request(
        _handle,
        timeoutMillis,
        out,
      );
      if (status == anytty_status_v1.ANYTTY_STATUS_CLOSED) return null;
      _check('platform_next_request', status);
      return _copyAndRelease(out.ref);
    } finally {
      calloc.free(out);
    }
  }

  void completePlatformRequest(Uint8List response) {
    _ensureOpen();
    if (response.isEmpty) {
      throw ArgumentError.value(response, 'response', 'must not be empty');
    }
    final data = calloc<Uint8>(response.length);
    try {
      data.asTypedList(response.length).setAll(0, response);
      _check(
        'platform_complete',
        _native.anytty_platform_complete(_handle, data, response.length),
      );
    } finally {
      calloc.free(data);
    }
  }

  void cancel(int operationHandle) {
    _ensureOpen();
    _check('cancel', _native.anytty_engine_cancel(_handle, operationHandle));
  }

  void closeSession(int sessionHandle) {
    _ensureOpen();
    _check(
      'close_session',
      _native.anytty_engine_close_session(_handle, sessionHandle),
    );
  }

  void release(int handle) {
    _ensureOpen();
    _check('release', _native.anytty_engine_release(_handle, handle));
  }

  void close() {
    if (_closed) return;
    _closed = true;
    _check('engine_close', _native.anytty_engine_close(_handle));
  }

  int _operation(
    String name,
    Uint8List payload,
    anytty_status_v1 Function(
      Pointer<Uint8> data,
      int length,
      Pointer<anytty_handle_t> outHandle,
    )
    invoke,
  ) {
    _ensureOpen();
    if (payload.isEmpty) {
      throw ArgumentError.value(payload, 'payload', 'must not be empty');
    }
    final data = calloc<Uint8>(payload.length);
    final outHandle = calloc<anytty_handle_t>();
    try {
      data.asTypedList(payload.length).setAll(0, payload);
      final status = invoke(data, payload.length, outHandle);
      _check(name, status);
      return outHandle.value;
    } finally {
      calloc.free(outHandle);
      calloc.free(data);
    }
  }

  void _payload(
    String name,
    Uint8List payload,
    anytty_status_v1 Function(Pointer<Uint8> data, int length) invoke,
  ) {
    _ensureOpen();
    if (payload.isEmpty) {
      throw ArgumentError.value(payload, 'payload', 'must not be empty');
    }
    final data = calloc<Uint8>(payload.length);
    try {
      data.asTypedList(payload.length).setAll(0, payload);
      _check(name, invoke(data, payload.length));
    } finally {
      calloc.free(data);
    }
  }

  Uint8List _copyAndRelease(anytty_buffer_v1 buffer) {
    try {
      if (buffer.length == 0) return Uint8List(0);
      return Uint8List.fromList(buffer.data.asTypedList(buffer.length));
    } finally {
      if (buffer.buffer_handle != 0) {
        _check('buffer_free', _native.anytty_buffer_free(buffer.buffer_handle));
      }
    }
  }

  void _ensureOpen() {
    if (_closed) throw StateError('AnyTTY Client Engine is closed');
  }

  static void _check(String operation, anytty_status_v1 status) {
    if (status != anytty_status_v1.ANYTTY_STATUS_OK) {
      throw AnyttyNativeException(operation, status);
    }
  }

  static AnyttyClientNative _loadNative() {
    final native = AnyttyClientNative(loadAnyttyLibrary('libanytty_client.so'));
    final version = native.anytty_client_abi_version();
    if (version != ANYTTY_CLIENT_ABI_VERSION) {
      throw StateError(
        'AnyTTY Client ABI mismatch: native=$version dart=$ANYTTY_CLIENT_ABI_VERSION',
      );
    }
    return native;
  }
}
