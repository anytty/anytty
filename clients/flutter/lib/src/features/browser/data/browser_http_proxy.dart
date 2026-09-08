abstract interface class BrowserProxySession {
  Future<int> startBrowserProxy();
  Future<void> stopBrowserProxy(int port);
}

/// Controls a Go-owned loopback proxy. No browser payload crosses Dart/FFI.
final class BrowserHttpProxy {
  BrowserHttpProxy._(this._session, this.port);

  final BrowserProxySession _session;
  final int port;
  Future<void>? _closing;

  static Future<BrowserHttpProxy> start(BrowserProxySession session) async {
    final port = await session.startBrowserProxy();
    if (port < 1 || port > 65535) {
      throw StateError('Native browser proxy returned an invalid port');
    }
    return BrowserHttpProxy._(session, port);
  }

  Future<void> close() => _closing ??= _session.stopBrowserProxy(port);
}
