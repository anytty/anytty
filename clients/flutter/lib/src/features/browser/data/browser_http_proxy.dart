import 'dart:async';
import 'dart:collection';
import 'dart:convert';
import 'dart:developer' as developer;
import 'dart:io';
import 'dart:typed_data';

import '../../../generated/proto/apipb/common.pb.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../native/anytty_resource_stream.dart';

abstract interface class BrowserProxySession {
  Future<ResourceHandle> openBrowserProxy({
    required String host,
    required int port,
  });

  Future<AnyttyResourceStream> openBrowserResourceStream(
    ResourceHandle resource,
  );
}

/// A loopback HTTP proxy that maps each local TCP connection to one
/// daemon-side browser resource. It never resolves a target hostname locally.
final class BrowserHttpProxy {
  BrowserHttpProxy._(
    this._session,
    this._server,
    this._headerTimeout,
    this._maximumSockets,
  );

  static const _maximumHeaderBytes = 64 * 1024;
  static const _maximumConcurrentResources = 16;
  static const _requestHeaderTimeout = Duration(seconds: 10);
  static const _maximumConcurrentSockets = 64;

  final BrowserProxySession _session;
  final ServerSocket _server;
  final Duration _headerTimeout;
  final int _maximumSockets;
  final Set<Socket> _sockets = <Socket>{};
  final Set<AnyttyResourceStream> _streams = <AnyttyResourceStream>{};
  final _AsyncSemaphore _resourceLimiter = _AsyncSemaphore(
    _maximumConcurrentResources,
  );
  bool _closed = false;

  int get port => _server.port;

  static Future<BrowserHttpProxy> start(
    BrowserProxySession session, {
    Duration headerTimeout = _requestHeaderTimeout,
    int maximumSockets = _maximumConcurrentSockets,
  }) async {
    if (headerTimeout <= Duration.zero) {
      throw ArgumentError.value(headerTimeout, 'headerTimeout');
    }
    if (maximumSockets < 1) {
      throw ArgumentError.value(maximumSockets, 'maximumSockets');
    }
    final server = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final proxy = BrowserHttpProxy._(
      session,
      server,
      headerTimeout,
      maximumSockets,
    );
    server.listen(proxy._accept, onError: (_) {});
    return proxy;
  }

  void _accept(Socket socket) {
    if (_closed) {
      socket.destroy();
      return;
    }
    if (_sockets.length >= _maximumSockets) {
      socket.destroy();
      return;
    }
    _sockets.add(socket);
    socket.setOption(SocketOption.tcpNoDelay, true);
    unawaited(_serve(socket).whenComplete(() => _sockets.remove(socket)));
  }

  Future<void> _serve(Socket socket) async {
    final input = _SocketInput(socket);
    AnyttyResourceStream? stream;
    StreamSubscription<Uint8List>? localSubscription;
    StreamSubscription<ResourceStreamFrame>? remoteSubscription;
    _SemaphoreLease? resourceLease;
    final done = Completer<void>();

    void finish() {
      if (done.isCompleted) return;
      done.complete();
      unawaited(localSubscription?.cancel() ?? Future<void>.value());
      unawaited(remoteSubscription?.cancel() ?? Future<void>.value());
      try {
        stream?.close();
      } catch (_) {}
      socket.destroy();
      input.close();
    }

    try {
      final request = await input.request.timeout(
        _headerTimeout,
        onTimeout: () =>
            throw TimeoutException('HTTP proxy request headers timed out'),
      );
      developer.log(
        'request target=${request.host}:${request.port} '
        'connect=${request.connect} websocket=${request.websocketUpgrade}',
        name: 'anytty.browser.proxy',
      );
      resourceLease = await _resourceLimiter.acquire();
      final resource = await _session.openBrowserProxy(
        host: request.host,
        port: request.port,
      );
      developer.log(
        'resource opened target=${request.host}:${request.port}',
        name: 'anytty.browser.proxy',
      );
      stream = await _session.openBrowserResourceStream(resource);
      _streams.add(stream);
      final activeStream = stream;

      if (request.connect) {
        socket.add(ascii.encode('HTTP/1.1 200 Connection Established\r\n\r\n'));
      }
      final initialPayload = <int>[
        if (!request.connect) ...request.forwardedHeader,
        ...request.leftover,
      ];
      if (initialPayload.isNotEmpty) {
        stream.send(
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_BROWSER_DATA,
          initialPayload,
        );
      }

      localSubscription = input.body.listen(
        (bytes) {
          try {
            stream!.send(
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_BROWSER_DATA,
              bytes,
            );
          } catch (_) {
            finish();
          }
        },
        onError: (Object error, StackTrace stackTrace) => finish(),
        onDone: finish,
        cancelOnError: true,
      );
      remoteSubscription = activeStream.frames.listen(
        (frame) {
          if (frame.type ==
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_BROWSER_DATA) {
            socket.add(frame.payload);
          } else if (frame.type ==
              ResourceStreamFrameType
                  .RESOURCE_STREAM_FRAME_TYPE_BROWSER_CLOSED) {
            finish();
          }
        },
        onError: (Object error, StackTrace stackTrace) => finish(),
        onDone: finish,
        cancelOnError: true,
      );
      unawaited(
        stream.closed.then(
          (_) => finish(),
          onError: (Object error, StackTrace stackTrace) => finish(),
        ),
      );
      await done.future;
    } catch (error, stackTrace) {
      developer.log(
        'request failed error=$error',
        name: 'anytty.browser.proxy',
        error: error,
        stackTrace: stackTrace,
      );
      if (!done.isCompleted) {
        try {
          socket.add(
            ascii.encode(
              'HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n',
            ),
          );
        } catch (_) {}
      }
      finish();
    } finally {
      resourceLease?.release();
      if (stream != null) _streams.remove(stream);
      input.close();
      socket.destroy();
    }
  }

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    _resourceLimiter.close();
    await _server.close();
    for (final stream in List<AnyttyResourceStream>.of(_streams)) {
      try {
        stream.close();
      } catch (_) {}
    }
    for (final socket in List<Socket>.of(_sockets)) {
      socket.destroy();
    }
    _streams.clear();
    _sockets.clear();
  }
}

final class _AsyncSemaphore {
  _AsyncSemaphore(int capacity) : _available = capacity;

  int _available;
  final Queue<Completer<_SemaphoreLease>> _waiters =
      Queue<Completer<_SemaphoreLease>>();
  bool _closed = false;

  Future<_SemaphoreLease> acquire() {
    if (_closed) {
      return Future<_SemaphoreLease>.error(
        StateError('browser proxy is closed'),
      );
    }
    if (_available > 0) {
      _available--;
      return Future<_SemaphoreLease>.value(_SemaphoreLease(this));
    }
    final completer = Completer<_SemaphoreLease>();
    _waiters.add(completer);
    return completer.future;
  }

  void release() {
    while (_waiters.isNotEmpty) {
      final waiter = _waiters.removeFirst();
      if (!waiter.isCompleted) {
        waiter.complete(_SemaphoreLease(this));
        return;
      }
    }
    _available++;
  }

  void close() {
    if (_closed) return;
    _closed = true;
    while (_waiters.isNotEmpty) {
      final waiter = _waiters.removeFirst();
      if (!waiter.isCompleted) {
        waiter.completeError(StateError('browser proxy is closed'));
      }
    }
  }
}

final class _SemaphoreLease {
  _SemaphoreLease(this._semaphore);

  final _AsyncSemaphore _semaphore;
  bool _released = false;

  void release() {
    if (_released) return;
    _released = true;
    _semaphore.release();
  }
}

final class _ProxyRequest {
  const _ProxyRequest({
    required this.host,
    required this.port,
    required this.connect,
    required this.websocketUpgrade,
    required this.forwardedHeader,
    required this.leftover,
  });

  final String host;
  final int port;
  final bool connect;
  final bool websocketUpgrade;
  final List<int> forwardedHeader;
  final List<int> leftover;
}

final class _SocketInput {
  _SocketInput(Socket socket) {
    _subscription = socket.listen(
      _onData,
      onError: _onError,
      onDone: _onDone,
      cancelOnError: true,
    );
  }

  final StreamController<Uint8List> _body = StreamController<Uint8List>();
  final List<int> _header = <int>[];
  late final StreamSubscription<Uint8List> _subscription;
  final Completer<_ProxyRequest> _request = Completer<_ProxyRequest>();
  bool _headerDone = false;
  bool _closed = false;
  int _headerBytes = 0;

  Future<_ProxyRequest> get request => _request.future;
  Stream<Uint8List> get body => _body.stream;

  void _onData(Uint8List bytes) {
    if (_headerDone) {
      _body.add(bytes);
      return;
    }
    _header.addAll(bytes);
    _headerBytes += bytes.length;
    if (_headerBytes > BrowserHttpProxy._maximumHeaderBytes) {
      _onError(const FormatException('HTTP proxy header is too large'));
      return;
    }
    final all = Uint8List.fromList(_header);
    final marker = _findHeaderEnd(all);
    if (marker < 0) return;
    _headerDone = true;
    final head = all.sublist(0, marker);
    final leftover = all.sublist(marker + 4);
    try {
      _request.complete(_parseRequest(head, leftover));
    } catch (error, stackTrace) {
      _request.completeError(error, stackTrace);
    }
  }

  void _onError(Object error, [StackTrace? stackTrace]) {
    if (!_request.isCompleted) _request.completeError(error, stackTrace);
    if (!_body.isClosed) {
      _body.addError(error, stackTrace ?? StackTrace.current);
    }
  }

  void _onDone() {
    if (!_request.isCompleted) {
      _request.completeError(
        const FormatException('HTTP proxy request ended early'),
      );
    }
    if (!_body.isClosed) unawaited(_body.close());
  }

  void close() {
    if (_closed) return;
    _closed = true;
    unawaited(_subscription.cancel());
    if (!_body.isClosed) unawaited(_body.close());
  }
}

int _findHeaderEnd(List<int> bytes) {
  for (var index = 0; index + 3 < bytes.length; index++) {
    if (bytes[index] == 13 &&
        bytes[index + 1] == 10 &&
        bytes[index + 2] == 13 &&
        bytes[index + 3] == 10) {
      return index;
    }
  }
  return -1;
}

_ProxyRequest _parseRequest(List<int> head, List<int> leftover) {
  final text = ascii.decode(head, allowInvalid: true);
  final lines = text.split('\r\n');
  if (lines.isEmpty) {
    throw const FormatException('HTTP request line is missing');
  }
  final requestLine = lines.first.split(' ');
  if (requestLine.length < 2) {
    throw const FormatException('HTTP request line is invalid');
  }
  final method = requestLine.first.toUpperCase();
  if (method == 'CONNECT') {
    final authority = _parseAuthority(requestLine[1], 443);
    return _ProxyRequest(
      host: authority.$1,
      port: authority.$2,
      connect: true,
      websocketUpgrade: false,
      forwardedHeader: const <int>[],
      leftover: leftover,
    );
  }
  final uri = Uri.tryParse(requestLine[1]);
  if (uri == null || uri.host.isEmpty) {
    throw const FormatException('HTTP proxy request must use an absolute URI');
  }
  if (requestLine.length < 3 || requestLine[2].isEmpty) {
    throw const FormatException('HTTP request version is missing');
  }
  final target = uri.path.isEmpty ? '/' : uri.path;
  final withQuery = uri.hasQuery ? '$target?${uri.query}' : target;
  final rewritten = <String>['$method $withQuery ${requestLine[2]}'];
  final connectionValues = <String>[];
  String? upgradeValue;
  for (final line in lines.skip(1)) {
    final lowerLine = line.toLowerCase();
    if (line.isNotEmpty && lowerLine.startsWith('proxy-connection:')) {
      continue;
    }
    if (line.isNotEmpty && lowerLine.startsWith('connection:')) {
      connectionValues.add(line.substring(line.indexOf(':') + 1).trim());
      continue;
    }
    if (line.isNotEmpty && lowerLine.startsWith('upgrade:')) {
      upgradeValue = line.substring(line.indexOf(':') + 1).trim();
      continue;
    }
    rewritten.add(line);
  }
  final websocketUpgrade =
      connectionValues
          .expand((value) => value.split(','))
          .map((value) => value.trim().toLowerCase())
          .contains('upgrade') &&
      upgradeValue?.toLowerCase() == 'websocket';
  if (websocketUpgrade) {
    // A WebSocket owns the resource until either peer closes. Closing the
    // upstream HTTP connection here would turn the handshake into a 502.
    rewritten.add('Connection: Upgrade');
    rewritten.add('Upgrade: websocket');
  } else {
    // Each local socket owns one daemon-side resource. Close the upstream HTTP
    // connection so WebView cannot reuse it for another request.
    rewritten.add('Connection: close');
  }
  rewritten.add('');
  rewritten.add('');
  return _ProxyRequest(
    host: uri.host,
    port: uri.hasPort ? uri.port : 80,
    connect: false,
    websocketUpgrade: websocketUpgrade,
    forwardedHeader: ascii.encode(rewritten.join('\r\n')),
    leftover: leftover,
  );
}

(String, int) _parseAuthority(String value, int defaultPort) {
  if (value.startsWith('[')) {
    final end = value.indexOf(']');
    if (end < 0) throw const FormatException('HTTP authority is invalid');
    final host = value.substring(1, end);
    final port = end + 1 < value.length && value[end + 1] == ':'
        ? int.tryParse(value.substring(end + 2))
        : defaultPort;
    if (port == null || port < 1 || port > 65535) {
      throw const FormatException('HTTP authority port is invalid');
    }
    return (host, port);
  }
  final separator = value.lastIndexOf(':');
  if (separator < 0) return (value, defaultPort);
  final port = int.tryParse(value.substring(separator + 1));
  if (port == null || port < 1 || port > 65535) {
    throw const FormatException('HTTP authority port is invalid');
  }
  return (value.substring(0, separator), port);
}
