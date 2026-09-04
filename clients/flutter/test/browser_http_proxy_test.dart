import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:anytty_native/src/features/browser/data/browser_http_proxy.dart';
import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    as application;
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/anytty_resource_stream.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late _FakeBrowserProxySession session;
  late BrowserHttpProxy proxy;

  setUp(() async {
    session = _FakeBrowserProxySession();
    proxy = await BrowserHttpProxy.start(session);
  });

  tearDown(() async {
    await proxy.close();
    await session.runtime.close();
  });

  test('forwards CONNECT leftover bytes exactly once', () async {
    final socket = await Socket.connect(
      InternetAddress.loopbackIPv4,
      proxy.port,
    );
    final response = socket.first;
    socket.add(
      ascii.encode(
        'CONNECT example.test:443 HTTP/1.1\r\n'
        'Host: example.test:443\r\n'
        '\r\n'
        'tls-client-hello',
      ),
    );

    final responseBytes = await response.timeout(const Duration(seconds: 1));
    expect(
      String.fromCharCodes(responseBytes),
      contains('200 Connection Established'),
    );
    await session.firstData.future.timeout(const Duration(seconds: 1));

    expect(session.sentPayloads, [
      <int>[
        116,
        108,
        115,
        45,
        99,
        108,
        105,
        101,
        110,
        116,
        45,
        104,
        101,
        108,
        108,
        111,
      ],
    ]);
    socket.destroy();
  });

  test('keeps a remote loopback CONNECT target intact', () async {
    session.expectedHost = '127.0.0.1';
    session.expectedPort = 8080;
    final socket = await Socket.connect(
      InternetAddress.loopbackIPv4,
      proxy.port,
    );
    final response = socket.first;
    socket.add(
      ascii.encode(
        'CONNECT 127.0.0.1:8080 HTTP/1.1\r\n'
        'Host: 127.0.0.1:8080\r\n'
        '\r\n',
      ),
    );

    expect(
      String.fromCharCodes(await response.timeout(const Duration(seconds: 1))),
      contains('200 Connection Established'),
    );
    socket.destroy();
  });

  test('closes a socket that never completes its request headers', () async {
    await proxy.close();
    proxy = await BrowserHttpProxy.start(
      session,
      headerTimeout: const Duration(milliseconds: 30),
    );
    final socket = await Socket.connect(
      InternetAddress.loopbackIPv4,
      proxy.port,
    );
    socket.add(ascii.encode('GET http://127.0.0.1:5175/ HTTP/1.1\r\n'));

    final response = await socket.first.timeout(const Duration(seconds: 1));
    expect(ascii.decode(response), contains('502 Bad Gateway'));
    socket.destroy();
  });

  test('forwards absolute-form HTTP requests to the remote target', () async {
    session.expectedHost = '127.0.0.1';
    session.expectedPort = 5175;
    final socket = await Socket.connect(
      InternetAddress.loopbackIPv4,
      proxy.port,
    );
    final response = socket.first;
    socket.add(
      ascii.encode(
        'GET http://127.0.0.1:5175/ HTTP/1.1\r\n'
        'Host: 127.0.0.1:5175\r\n'
        'Connection: keep-alive\r\n'
        '\r\n',
      ),
    );

    await session.firstData.future.timeout(const Duration(seconds: 1));
    expect(
      ascii.decode(session.sentPayloads.single),
      startsWith('GET / HTTP/1.1\r\n'),
    );
    expect(
      ascii.decode(session.sentPayloads.single),
      contains('Connection: close\r\n'),
    );
    expect(
      ascii.decode(session.sentPayloads.single),
      isNot(contains('Connection: keep-alive\r\n')),
    );
    session.emitRemoteData(
      ascii.encode('HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok'),
    );

    expect(
      ascii.decode(await response.timeout(const Duration(seconds: 1))),
      startsWith('HTTP/1.1 200 OK\r\n'),
    );
    socket.destroy();
  });

  test(
    'keeps WebSocket upgrade requests on the same remote resource',
    () async {
      session.expectedHost = '127.0.0.1';
      session.expectedPort = 5175;
      final socket = await Socket.connect(
        InternetAddress.loopbackIPv4,
        proxy.port,
      );
      final response = socket.first;
      socket.add(
        ascii.encode(
          'GET ws://127.0.0.1:5175/socket HTTP/1.1\r\n'
          'Host: 127.0.0.1:5175\r\n'
          'Connection: keep-alive, Upgrade\r\n'
          'Upgrade: websocket\r\n'
          'Sec-WebSocket-Version: 13\r\n'
          'Sec-WebSocket-Key: dGVzdA==\r\n'
          '\r\n',
        ),
      );

      await session.firstData.future.timeout(const Duration(seconds: 1));
      final request = ascii.decode(session.sentPayloads.single);
      expect(request, startsWith('GET /socket HTTP/1.1\r\n'));
      expect(request, contains('Connection: Upgrade\r\n'));
      expect(request, contains('Upgrade: websocket\r\n'));
      expect(request, isNot(contains('Connection: close\r\n')));

      session.emitRemoteData(
        ascii.encode(
          'HTTP/1.1 101 Switching Protocols\r\n'
          'Connection: Upgrade\r\n'
          'Upgrade: websocket\r\n'
          '\r\n',
        ),
      );
      expect(
        ascii.decode(await response.timeout(const Duration(seconds: 1))),
        startsWith('HTTP/1.1 101 Switching Protocols\r\n'),
      );
      socket.destroy();
    },
  );
}

final class _FakeBrowserProxySession implements BrowserProxySession {
  final runtime = _FakeResourceRuntime();
  String expectedHost = 'example.test';
  int expectedPort = 443;

  Completer<void> get firstData => runtime.firstData;

  List<List<int>> get sentPayloads => runtime.sentPayloads;

  void emitRemoteData(List<int> payload) => runtime.emitRemoteData(payload);

  @override
  Future<ResourceHandle> openBrowserProxy({
    required String host,
    required int port,
  }) async {
    expect(host, expectedHost);
    expect(port, expectedPort);
    return ResourceHandle(
      opaqueToken: <int>[1, 2, 3],
      kind: ResourceKind.RESOURCE_KIND_BROWSER_PROXY,
    );
  }

  @override
  Future<AnyttyResourceStream> openBrowserResourceStream(
    ResourceHandle resource,
  ) {
    return AnyttyResourceStream.open(
      runtime: runtime,
      sessionHandle: 1,
      request: OpenResourceStreamRequest(resource: resource),
    );
  }
}

final class _FakeResourceRuntime
    implements AnyttyEngineRuntime, AnyttyResourceStreamRuntime {
  final _events = StreamController<EventEnvelope>.broadcast(sync: true);
  final firstData = Completer<void>();
  final sentPayloads = <List<int>>[];
  final closed = <int>[];

  @override
  Stream<EventEnvelope> get events => _events.stream;

  @override
  Stream<int> get foregroundResumes => const Stream<int>.empty();

  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) =>
      EndpointDemandLease(() {});

  @override
  int command(EngineCommand command) => throw UnimplementedError();

  @override
  int openSession(OpenSessionRequest request) => throw UnimplementedError();

  @override
  int execute(int sessionHandle, application.CommandEnvelope request) =>
      throw UnimplementedError();

  @override
  void cancel(int operationHandle) => throw UnimplementedError();

  @override
  void release(int handle) {}

  @override
  void closeSession(int sessionHandle) {}

  @override
  int openResourceStream(
    int sessionHandle,
    OpenResourceStreamRequest request,
  ) => 41;

  @override
  void sendResourceStreamFrame(int streamHandle, ResourceStreamFrame frame) {
    if (frame.type ==
        ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_BROWSER_DATA) {
      sentPayloads.add(List<int>.of(frame.payload));
      if (!firstData.isCompleted) firstData.complete();
    }
  }

  void emitRemoteData(List<int> payload) {
    _events.add(
      EventEnvelope(
        resourceStreamFrame: ResourceStreamFrame(
          streamHandle: Int64(41),
          type: ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_BROWSER_DATA,
          payload: payload,
        ),
      ),
    );
  }

  @override
  void closeResourceStream(int streamHandle) {
    closed.add(streamHandle);
    _events.add(
      EventEnvelope(
        resourceStreamClosed: ResourceStreamClosedEvent(
          streamHandle: Int64(streamHandle),
        ),
      ),
    );
  }

  Future<void> close() => _events.close();
}
