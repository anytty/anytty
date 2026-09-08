import 'package:anytty_native/src/features/browser/data/browser_http_proxy.dart';
import 'package:flutter_test/flutter_test.dart';

final class _Session implements BrowserProxySession {
  int port = 12345;
  int starts = 0;
  final stops = <int>[];
  @override
  Future<int> startBrowserProxy() async {
    starts++;
    return port;
  }

  @override
  Future<void> stopBrowserProxy(int port) async {
    stops.add(port);
  }
}

void main() {
  test(
    'Dart owns only native listener lifecycle and exact-port cleanup',
    () async {
      final session = _Session();
      final proxy = await BrowserHttpProxy.start(session);
      expect(proxy.port, 12345);
      expect(session.starts, 1);
      await Future.wait([proxy.close(), proxy.close()]);
      expect(session.stops, [12345]);
    },
  );
  test('invalid native listener port fails closed', () async {
    for (final port in [0, -1, 65536]) {
      await expectLater(
        BrowserHttpProxy.start(_Session()..port = port),
        throwsStateError,
      );
    }
  });
}
