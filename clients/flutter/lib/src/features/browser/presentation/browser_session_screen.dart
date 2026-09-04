import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/providers.dart';
import '../../../native/browser_proxy_platform.dart';
import '../data/browser_http_proxy.dart';
import '../data/browser_session_store.dart';
import '../data/browser_snapshot_store.dart';
import '../domain/browser_session.dart';
import '../../terminal/data/endpoint_session_client.dart';
import 'browser_endpoint_picker_sheet.dart';

final class BrowserSessionScreen extends ConsumerStatefulWidget {
  const BrowserSessionScreen({
    super.key,
    required this.endpointId,
    this.endpointLabel,
    this.proxyPlatform,
    this.sessionStore,
    this.snapshotStore,
  });

  final String endpointId;
  final String? endpointLabel;
  final BrowserProxyPlatform? proxyPlatform;
  final BrowserSessionStore? sessionStore;
  final BrowserSnapshotStore? snapshotStore;

  @override
  ConsumerState<BrowserSessionScreen> createState() =>
      _BrowserSessionScreenState();
}

final class _BrowserSessionScreenState
    extends ConsumerState<BrowserSessionScreen> {
  static const _sessionPreparationTimeout = Duration(seconds: 8);

  final _addressController = TextEditingController();
  final _addressFocusNode = FocusNode();
  final _webViewBoundaryKey = GlobalKey();
  final _stateMachine = BrowserSessionStateMachine();

  late final BrowserProxyPlatform _proxyPlatform;
  late final BrowserSessionStore _sessionStore;
  late final BrowserSnapshotStore _snapshotStore;

  String _activeEndpointId = '';
  String _activeEndpointLabel = '';
  BrowserSessionSnapshot? _snapshot;
  Uint8List? _snapshotBytes;
  BrowserProxyLease? _proxyLease;
  BrowserHttpProxy? _httpProxy;
  WebViewController? _webViewController;
  EndpointSessionClient? _activeEndpointSession;
  ProviderSubscription<AsyncValue<EndpointSessionClient>>?
  _endpointSessionSubscription;
  String? _error;
  bool _pageReady = false;
  bool _closing = false;
  bool _sessionRecoveryPending = false;
  String? _preparingEndpointLabel;
  Future<void> _transitionTail = Future<void>.value();

  @override
  void initState() {
    super.initState();
    _proxyPlatform =
        widget.proxyPlatform ?? MethodChannelBrowserProxyPlatform.instance;
    _sessionStore =
        widget.sessionStore ?? const SharedPreferencesBrowserSessionStore();
    _snapshotStore =
        widget.snapshotStore ?? const ApplicationBrowserSnapshotStore();
    _activeEndpointId = widget.endpointId;
    _activeEndpointLabel = _labelFor(widget.endpointId, widget.endpointLabel);
    _retainEndpointSession(_activeEndpointId);
    unawaited(_activateSession(_activeEndpointId, _activeEndpointLabel));
  }

  @override
  void dispose() {
    _closing = true;
    final controller = _webViewController;
    final lease = _proxyLease;
    final httpProxy = _httpProxy;
    final snapshot = _snapshot;
    _webViewController = null;
    _proxyLease = null;
    _httpProxy = null;
    _activeEndpointSession = null;
    _endpointSessionSubscription?.close();
    _endpointSessionSubscription = null;
    unawaited(
      _persistDetachedSession(
        sessionId: _activeEndpointId,
        endpointLabel: _activeEndpointLabel,
        controller: controller,
        lease: lease,
        httpProxy: httpProxy,
        previous: snapshot,
      ),
    );
    _addressController.dispose();
    _addressFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final state = _stateMachine.state;
    return PopScope<Object?>(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) unawaited(_closeScreen());
      },
      child: Scaffold(
        backgroundColor: palette.background,
        appBar: AppBar(
          toolbarHeight: 56,
          automaticallyImplyLeading: false,
          leading: IconButton(
            tooltip: anyttyText(context, en: 'Back', zh: '返回'),
            onPressed: _closing ? null : () => unawaited(_closeScreen()),
            icon: const Icon(Icons.arrow_back_rounded, size: 22),
          ),
          titleSpacing: 0,
          title: _BrowserToolbarTitle(
            endpointLabel: _activeEndpointLabel,
            preparingEndpointLabel: _preparingEndpointLabel,
            state: state,
            hasProxy: _proxyLease != null,
            dnsProxied: _proxyLease?.dnsProxied ?? false,
            addressController: _addressController,
            addressFocusNode: _addressFocusNode,
            controller: _webViewController,
            onNavigate: _navigate,
            onBack: () => _goBack(_webViewController),
            onForward: () => _goForward(_webViewController),
            onSwitchSession: () => unawaited(_openEndpointPicker()),
          ),
          actions: [
            IconButton(
              tooltip: anyttyText(context, en: 'Reload', zh: '重新加载'),
              onPressed: state.phase == BrowserSessionPhase.active
                  ? () => _reload(_webViewController)
                  : null,
              icon: const Icon(Icons.refresh_rounded, size: 20),
            ),
            _BrowserOverflowMenu(
              state: state,
              hasProxy: _proxyLease != null,
              dnsProxied: _proxyLease?.dnsProxied ?? false,
              controller: _webViewController,
              onBack: () => _goBack(_webViewController),
              onForward: () => _goForward(_webViewController),
              onSwitchSession: () => unawaited(_openEndpointPicker()),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Column(children: [Expanded(child: _buildContent(context))]),
        ),
      ),
    );
  }

  Widget _buildContent(BuildContext context) {
    final state = _stateMachine.state;
    final controller = _webViewController;
    final palette = AnyttyPalette.of(context);
    if (controller != null) {
      return Stack(
        fit: StackFit.expand,
        children: [
          RepaintBoundary(
            key: _webViewBoundaryKey,
            child: WebViewWidget(controller: controller),
          ),
          if (!_pageReady && _snapshotBytes != null)
            IgnorePointer(
              child: Image.memory(_snapshotBytes!, fit: BoxFit.cover),
            ),
          if (!_pageReady)
            Align(
              alignment: Alignment.topCenter,
              child: _BrowserLoadingMarker(
                label: anyttyText(
                  context,
                  en: 'Restoring session',
                  zh: '正在恢复会话',
                ),
              ),
            ),
        ],
      );
    }
    if (_snapshotBytes != null &&
        (state.phase == BrowserSessionPhase.blocked ||
            state.phase == BrowserSessionPhase.failed ||
            state.phase == BrowserSessionPhase.parked ||
            state.phase == BrowserSessionPhase.parking ||
            state.phase == BrowserSessionPhase.restoring)) {
      return Stack(
        fit: StackFit.expand,
        children: [
          Image.memory(_snapshotBytes!, fit: BoxFit.cover),
          Align(
            alignment: Alignment.bottomCenter,
            child: _BrowserLoadingMarker(
              label:
                  state.phase == BrowserSessionPhase.parking ||
                      state.phase == BrowserSessionPhase.restoring
                  ? anyttyText(context, en: 'Restoring session', zh: '正在恢复会话')
                  : anyttyText(context, en: 'Paused preview', zh: '已暂停预览'),
            ),
          ),
        ],
      );
    }
    final blocked = state.phase == BrowserSessionPhase.blocked;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                blocked ? Icons.route_rounded : Icons.language_rounded,
                size: 32,
                color: blocked ? palette.warning : palette.muted,
              ),
              const SizedBox(height: 14),
              Text(
                blocked
                    ? anyttyText(
                        context,
                        en: 'Web tunnel unavailable',
                        zh: 'Web 隧道不可用',
                      )
                    : anyttyText(
                        context,
                        en: 'Preparing web session',
                        zh: '正在准备 Web 会话',
                      ),
                textAlign: TextAlign.center,
                style: TextStyle(
                  color: palette.text,
                  fontSize: 17,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                blocked
                    ? anyttyText(
                        context,
                        en: 'The browser stays closed until this device has a session-bound route.',
                        zh: '当前设备的会话隧道建立后，浏览器才会启动。',
                      )
                    : anyttyText(
                        context,
                        en: 'The browser will open only after its route is ready.',
                        zh: '路由准备好后，浏览器才会打开。',
                      ),
                textAlign: TextAlign.center,
                style: TextStyle(color: palette.muted, height: 1.45),
              ),
              if (blocked) ...[
                const SizedBox(height: 20),
                FilledButton.icon(
                  onPressed: () => unawaited(
                    _activateSession(_activeEndpointId, _activeEndpointLabel),
                  ),
                  icon: const Icon(Icons.refresh_rounded, size: 18),
                  label: Text(
                    anyttyText(context, en: 'Retry tunnel', zh: '重试隧道'),
                  ),
                ),
              ],
              if (_error != null) ...[
                const SizedBox(height: 12),
                Text(
                  _error!,
                  textAlign: TextAlign.center,
                  style: TextStyle(color: palette.danger, fontSize: 12),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _activateSession(
    String endpointId,
    String endpointLabel, {
    bool forceReconnect = false,
  }) {
    final next = _transitionTail.then(
      (_) => _activateSessionNow(
        endpointId,
        endpointLabel,
        forceReconnect: forceReconnect,
      ),
      onError: (_, _) => _activateSessionNow(
        endpointId,
        endpointLabel,
        forceReconnect: forceReconnect,
      ),
    );
    _transitionTail = next.then<void>((_) {}, onError: (_, _) {});
    return next;
  }

  Future<void> _activateSessionNow(
    String endpointId,
    String endpointLabel, {
    bool forceReconnect = false,
  }) async {
    final previousEndpointId = _activeEndpointId;
    final previousEndpointLabel = _activeEndpointLabel;
    final hasLiveSession =
        _webViewController != null || _proxyLease != null || _httpProxy != null;
    if (hasLiveSession && endpointId == previousEndpointId && !forceReconnect) {
      return;
    }

    final switching = hasLiveSession && endpointId != previousEndpointId;
    if (mounted && switching) {
      setState(() {
        _preparingEndpointLabel = endpointLabel;
        _error = null;
      });
    }

    _PreparedBrowserSession prepared;
    try {
      prepared = await _prepareSession(
        endpointId: endpointId,
        endpointLabel: endpointLabel,
      );
    } catch (error) {
      if (mounted && switching) {
        setState(() => _preparingEndpointLabel = null);
        _showSwitchError(endpointLabel, error);
        return;
      }
      final operation = _stateMachine.begin(endpointId);
      final message = '$error';
      if (error is BrowserProxyUnavailableException) {
        _stateMachine.markBlocked(operation, endpointId, message);
      } else {
        _stateMachine.markFailed(operation, endpointId, error);
      }
      if (mounted) {
        setState(() {
          _preparingEndpointLabel = null;
          _error = message;
        });
      }
      return;
    }

    if (mounted) {
      setState(() {
        _preparingEndpointLabel = null;
        _error = null;
        _pageReady = false;
      });
    }
    final operation = _stateMachine.begin(endpointId);
    if (!await _parkLiveSession(captureSnapshot: !forceReconnect)) {
      await prepared.dispose();
      _stateMachine.markBlocked(
        operation,
        endpointId,
        'WebView data isolation is unavailable',
      );
      if (switching) {
        _restorePreviousSessionAfterFailure(
          previousEndpointId,
          previousEndpointLabel,
        );
      }
      return;
    }
    if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
      await prepared.dispose();
      return;
    }

    final restoredSnapshot = prepared.snapshot;
    final bytes = prepared.snapshotBytes;
    _stateMachine.markRestoring(operation, endpointId, restoredSnapshot);
    setState(() {
      _snapshot = restoredSnapshot;
      _snapshotBytes = bytes;
      _addressController.text = restoredSnapshot.url;
    });

    late final BrowserProxyLease lease;
    try {
      lease = await _proxyPlatform.open(
        sessionId: endpointId,
        endpointId: endpointId,
        proxyHost: '127.0.0.1',
        proxyPort: prepared.httpProxy.port,
        routeId: prepared.endpointSession.stamp.routeId,
        routeGeneration: prepared.endpointSession.stamp.generation.toInt(),
      );
    } on BrowserProxyUnavailableException catch (error) {
      await prepared.dispose();
      _stateMachine.markBlocked(operation, endpointId, error.message);
      if (mounted) {
        setState(() => _error = error.message);
        if (switching) _showSwitchError(endpointLabel, error);
      }
      if (switching) {
        _restorePreviousSessionAfterFailure(
          previousEndpointId,
          previousEndpointLabel,
        );
      }
      return;
    } on MissingPluginException {
      await prepared.dispose();
      const error = BrowserProxyUnavailableException();
      _stateMachine.markBlocked(operation, endpointId, error.message);
      if (mounted) {
        setState(() => _error = error.message);
        if (switching) _showSwitchError(endpointLabel, error);
      }
      if (switching) {
        _restorePreviousSessionAfterFailure(
          previousEndpointId,
          previousEndpointLabel,
        );
      }
      return;
    } catch (error) {
      await prepared.dispose();
      _stateMachine.markFailed(operation, endpointId, error);
      if (mounted) {
        setState(() => _error = '$error');
        if (switching) _showSwitchError(endpointLabel, error);
      }
      if (switching) {
        _restorePreviousSessionAfterFailure(
          previousEndpointId,
          previousEndpointLabel,
        );
      }
      return;
    }
    if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
      await _proxyPlatform.close(lease);
      await prepared.dispose();
      return;
    }

    final controller = WebViewController();
    var restoreScrollPending = restoredSnapshot.restorableUri != null;
    try {
      await controller.setJavaScriptMode(JavaScriptMode.unrestricted);
      await controller.setBackgroundColor(Colors.transparent);
      await controller.setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (url) {
            if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
              return;
            }
            setState(() => _pageReady = false);
          },
          onPageFinished: (url) {
            final restoreScroll = restoreScrollPending;
            restoreScrollPending = false;
            unawaited(
              _pageFinished(
                operation: operation,
                endpointId: endpointId,
                controller: controller,
                url: url,
                restoreScroll: restoreScroll,
              ),
            );
          },
          onUrlChange: (change) {
            if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
              return;
            }
            final url = change.url ?? '';
            if (!_addressFocusNode.hasFocus) _addressController.text = url;
          },
          onNavigationRequest: (request) {
            final uri = Uri.tryParse(request.url);
            if (uri == null || !_allowedUri(uri)) {
              return NavigationDecision.prevent;
            }
            return NavigationDecision.navigate;
          },
          onWebResourceError: (error) {
            if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
              return;
            }
            setState(() => _error = error.description);
          },
        ),
      );
      if (restoredSnapshot.restorableUri != null) {
        await controller.loadRequest(restoredSnapshot.restorableUri!);
      }
    } catch (error) {
      await _proxyPlatform.close(lease);
      await prepared.dispose();
      _stateMachine.markFailed(operation, endpointId, error);
      if (mounted) {
        setState(() => _error = '$error');
        if (switching) _showSwitchError(endpointLabel, error);
      }
      if (switching) {
        _restorePreviousSessionAfterFailure(
          previousEndpointId,
          previousEndpointLabel,
        );
      }
      return;
    }
    if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
      await _proxyPlatform.close(lease);
      await prepared.dispose();
      return;
    }
    final activeSnapshot = restoredSnapshot.copyWith(
      routeId: lease.routeId,
      routeGeneration: lease.routeGeneration,
    );
    if (prepared.endpointSubscription != null) {
      _adoptEndpointSession(
        prepared.endpointSubscription!,
        prepared.endpointSession,
      );
    }
    _proxyLease = lease;
    _httpProxy = prepared.httpProxy;
    _webViewController = controller;
    _activeEndpointId = endpointId;
    _activeEndpointLabel = endpointLabel;
    _activeEndpointSession = prepared.endpointSession;
    _watchEndpointSession(endpointId, prepared.endpointSession);
    _snapshot = activeSnapshot;
    _stateMachine.markActive(operation, endpointId, activeSnapshot);
    setState(() {
      _pageReady = activeSnapshot.restorableUri == null;
      _error = null;
    });
  }

  Future<_PreparedBrowserSession> _prepareSession({
    required String endpointId,
    required String endpointLabel,
  }) async {
    ProviderSubscription<AsyncValue<EndpointSessionClient>>?
    endpointSubscription;
    if (endpointId != _activeEndpointId) {
      endpointSubscription = ref
          .listenManual<AsyncValue<EndpointSessionClient>>(
            endpointSessionProvider(endpointId),
            (_, _) {},
          );
    }
    BrowserHttpProxy? httpProxy;
    try {
      final endpointSession = await ref
          .read(endpointSessionProvider(endpointId).future)
          .timeout(
            _sessionPreparationTimeout,
            onTimeout: () => throw const BrowserProxyUnavailableException(
              'The device session is not ready',
            ),
          );
      httpProxy = await BrowserHttpProxy.start(endpointSession);
      var snapshot = await _sessionStore.load(endpointId);
      snapshot ??= BrowserSessionSnapshot.empty(
        sessionId: endpointId,
        endpointId: endpointId,
        endpointLabel: endpointLabel,
      );
      snapshot = snapshot.copyWith(
        sessionId: endpointId,
        endpointId: endpointId,
        endpointLabel: endpointLabel,
      );
      final snapshotBytes = snapshot.snapshotPath == null
          ? null
          : await _snapshotStore.read(snapshot.snapshotPath!);
      return _PreparedBrowserSession(
        endpointSession: endpointSession,
        httpProxy: httpProxy,
        endpointSubscription: endpointSubscription,
        snapshot: snapshot,
        snapshotBytes: snapshotBytes,
      );
    } catch (_) {
      endpointSubscription?.close();
      await httpProxy?.close();
      rethrow;
    }
  }

  void _adoptEndpointSession(
    ProviderSubscription<AsyncValue<EndpointSessionClient>> subscription,
    EndpointSessionClient session,
  ) {
    _activeEndpointSession = session;
    _endpointSessionSubscription?.close();
    _endpointSessionSubscription = subscription;
  }

  void _restorePreviousSessionAfterFailure(
    String endpointId,
    String endpointLabel,
  ) {
    if (_closing || !mounted) return;
    unawaited(_activateSession(endpointId, endpointLabel));
  }

  void _showSwitchError(String endpointLabel, Object error) {
    if (!mounted) return;
    final detail = '$error'.trim();
    final message = anyttyText(
      context,
      en: 'Unable to switch to $endpointLabel${detail.isEmpty ? '' : ': $detail'}',
      zh: '无法切换到 $endpointLabel${detail.isEmpty ? '' : '：$detail'}',
    );
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(message)));
  }

  void _retainEndpointSession(String endpointId) {
    _endpointSessionSubscription?.close();
    _endpointSessionSubscription = ref
        .listenManual<AsyncValue<EndpointSessionClient>>(
          endpointSessionProvider(endpointId),
          (_, _) {},
        );
  }

  void _watchEndpointSession(String endpointId, EndpointSessionClient session) {
    unawaited(
      session.closed.then((_) {
        if (_closing || !mounted || _activeEndpointId != endpointId) return;
        if (!identical(_activeEndpointSession, session)) return;
        _scheduleSessionRecovery(endpointId);
      }),
    );
  }

  void _scheduleSessionRecovery(String endpointId) {
    if (_sessionRecoveryPending || _closing || !mounted) return;
    _sessionRecoveryPending = true;
    final label = _activeEndpointLabel;
    unawaited(
      _activateSession(endpointId, label, forceReconnect: true).whenComplete(
        () {
          _sessionRecoveryPending = false;
          final session = _activeEndpointSession;
          if (session != null &&
              session.isClosed &&
              !_closing &&
              mounted &&
              _activeEndpointId == endpointId) {
            _scheduleSessionRecovery(endpointId);
          }
        },
      ),
    );
  }

  Future<void> _pageFinished({
    required int operation,
    required String endpointId,
    required WebViewController controller,
    required String url,
    required bool restoreScroll,
  }) async {
    if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) return;
    try {
      final snapshot = _snapshot;
      if (restoreScroll &&
          snapshot != null &&
          (snapshot.scrollX != 0 || snapshot.scrollY != 0)) {
        await controller.scrollTo(snapshot.scrollX, snapshot.scrollY);
      }
      final title = await controller.getTitle() ?? '';
      final currentUrl = await controller.currentUrl() ?? url;
      final previous =
          _snapshot ??
          BrowserSessionSnapshot.empty(
            sessionId: endpointId,
            endpointId: endpointId,
            endpointLabel: _activeEndpointLabel,
          );
      final next = previous.copyWith(
        url: currentUrl,
        title: title,
        routeId: _proxyLease?.routeId,
        routeGeneration: _proxyLease?.routeGeneration,
        parkedAt: null,
      );
      _snapshot = next;
      _stateMachine.markActive(operation, endpointId, next);
      if (!_addressFocusNode.hasFocus) _addressController.text = currentUrl;
      await _sessionStore.save(next);
      if (mounted && _stateMachine.isCurrent(operation, endpointId)) {
        setState(() => _pageReady = true);
      }
    } catch (error) {
      if (mounted && _stateMachine.isCurrent(operation, endpointId)) {
        setState(() => _error = '$error');
      }
    }
  }

  Future<void> _navigate() async {
    final controller = _webViewController;
    if (controller == null || _proxyLease == null) return;
    final value = _addressController.text.trim();
    if (value.isEmpty) return;
    final uri = Uri.tryParse(value.contains('://') ? value : 'https://$value');
    if (uri == null || !_allowedUri(uri)) {
      setState(
        () => _error = anyttyText(
          context,
          en: 'Only HTTP and HTTPS pages can be opened.',
          zh: '只能打开 HTTP 和 HTTPS 页面。',
        ),
      );
      return;
    }
    _addressFocusNode.unfocus();
    setState(() {
      _error = null;
      _pageReady = false;
    });
    await controller.loadRequest(uri);
  }

  Future<bool> _parkLiveSession({bool captureSnapshot = true}) async {
    final controller = _webViewController;
    final lease = _proxyLease;
    final httpProxy = _httpProxy;
    if (controller == null && lease == null && httpProxy == null) {
      return _clearBrowserData();
    }
    if (mounted) {
      setState(() => _pageReady = false);
    }
    final parked = captureSnapshot
        ? await _captureSession(
            sessionId: _activeEndpointId,
            endpointLabel: _activeEndpointLabel,
            controller: controller,
            lease: lease,
            previous: _snapshot,
          )
        : _snapshot?.copyWith(parkedAt: DateTime.now());
    if (parked != null) {
      await _sessionStore.save(parked);
      _snapshot = parked;
      if (mounted) {
        final bytes = parked.snapshotPath == null
            ? null
            : await _snapshotStore.read(parked.snapshotPath!);
        if (mounted) {
          setState(() {
            _snapshot = parked;
            _snapshotBytes = bytes;
          });
        }
      }
    }
    _webViewController = null;
    _proxyLease = null;
    _httpProxy = null;
    if (mounted) {
      setState(() {});
      await WidgetsBinding.instance.endOfFrame;
    }
    if (lease != null) {
      try {
        await _proxyPlatform.close(lease);
      } catch (_) {}
    }
    if (httpProxy != null) {
      await httpProxy.close();
    }
    return _clearBrowserData();
  }

  Future<bool> _clearBrowserData() async {
    try {
      await _proxyPlatform.clearBrowserData();
      return true;
    } catch (error) {
      if (mounted) setState(() => _error = '$error');
      return false;
    }
  }

  Future<BrowserSessionSnapshot?> _captureSession({
    required String sessionId,
    required String endpointLabel,
    required WebViewController? controller,
    required BrowserProxyLease? lease,
    required BrowserSessionSnapshot? previous,
  }) async {
    var current =
        previous ??
        BrowserSessionSnapshot.empty(
          sessionId: sessionId,
          endpointId: sessionId,
          endpointLabel: endpointLabel,
        );
    current = current.copyWith(
      sessionId: sessionId,
      endpointId: sessionId,
      endpointLabel: endpointLabel,
      routeId: lease?.routeId ?? current.routeId,
      routeGeneration: lease?.routeGeneration ?? current.routeGeneration,
      parkedAt: DateTime.now(),
    );
    if (controller != null) {
      try {
        current = current.copyWith(
          url: await controller.currentUrl() ?? current.url,
          title: await controller.getTitle() ?? current.title,
        );
        final scroll = await controller.getScrollPosition();
        current = current.copyWith(
          scrollX: scroll.dx.round(),
          scrollY: scroll.dy.round(),
        );
      } catch (_) {}
      try {
        final boundary = _webViewBoundaryKey.currentContext?.findRenderObject();
        if (boundary is RenderRepaintBoundary) {
          final image = await boundary.toImage(pixelRatio: 1);
          final data = await image.toByteData(format: ui.ImageByteFormat.png);
          image.dispose();
          if (data != null) {
            final path = await _snapshotStore.save(
              sessionId,
              data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes),
            );
            current = current.copyWith(snapshotPath: path);
          }
        }
      } catch (_) {
        // Platform views may not be capturable on every renderer.
      }
    }
    return current;
  }

  Future<void> _persistDetachedSession({
    required String sessionId,
    required String endpointLabel,
    required WebViewController? controller,
    required BrowserProxyLease? lease,
    required BrowserHttpProxy? httpProxy,
    required BrowserSessionSnapshot? previous,
  }) async {
    final snapshot = await _captureSession(
      sessionId: sessionId,
      endpointLabel: endpointLabel,
      controller: controller,
      lease: lease,
      previous: previous,
    );
    if (snapshot != null) await _sessionStore.save(snapshot);
    if (lease != null) {
      try {
        await _proxyPlatform.close(lease);
      } catch (_) {}
    }
    if (httpProxy != null) {
      await httpProxy.close();
    }
    try {
      await _proxyPlatform.clearBrowserData();
    } catch (_) {}
  }

  Future<void> _closeScreen() async {
    if (_closing) return;
    _closing = true;
    await _parkLiveSession();
    if (mounted) Navigator.of(context).pop();
  }

  Future<void> _openEndpointPicker() async {
    List<BrowserEndpointOption> endpoints;
    try {
      final registry = await ref.read(endpointRegistryProvider.future);
      endpoints = registry.endpoints
          .where(
            (endpoint) =>
                endpoint.enabled || endpoint.endpointId == _activeEndpointId,
          )
          .map(
            (endpoint) => BrowserEndpointOption(
              endpointId: endpoint.endpointId,
              label: _labelFor(endpoint.endpointId, endpoint.label),
              current: endpoint.endpointId == _activeEndpointId,
            ),
          )
          .toList(growable: false);
    } catch (_) {
      endpoints = [
        BrowserEndpointOption(
          endpointId: _activeEndpointId,
          label: _activeEndpointLabel,
          current: true,
        ),
      ];
    }
    if (!mounted) return;
    final selected = await showAnyttyBrowserEndpointPicker(
      context: context,
      endpoints: endpoints,
    );
    if (!mounted || selected == null || selected == _activeEndpointId) return;
    final endpoint = endpoints.firstWhere(
      (item) => item.endpointId == selected,
    );
    await _activateSession(endpoint.endpointId, endpoint.label);
  }

  String _labelFor(String endpointId, String? label) {
    final value = label?.trim() ?? '';
    return value.isEmpty ? endpointId : value;
  }

  bool _allowedUri(Uri uri) =>
      uri.scheme == 'http' || uri.scheme == 'https' || uri.scheme == 'about';

  void _goBack(WebViewController? controller) {
    if (controller != null) unawaited(controller.goBack());
  }

  void _goForward(WebViewController? controller) {
    if (controller != null) unawaited(controller.goForward());
  }

  void _reload(WebViewController? controller) {
    if (controller != null) unawaited(controller.reload());
  }
}

final class _PreparedBrowserSession {
  _PreparedBrowserSession({
    required this.endpointSession,
    required this.httpProxy,
    required this.endpointSubscription,
    required this.snapshot,
    required this.snapshotBytes,
  });

  final EndpointSessionClient endpointSession;
  final BrowserHttpProxy httpProxy;
  final ProviderSubscription<AsyncValue<EndpointSessionClient>>?
  endpointSubscription;
  final BrowserSessionSnapshot snapshot;
  final Uint8List? snapshotBytes;

  Future<void> dispose() async {
    endpointSubscription?.close();
    await httpProxy.close();
  }
}

final class _BrowserToolbarTitle extends StatelessWidget {
  const _BrowserToolbarTitle({
    required this.endpointLabel,
    required this.preparingEndpointLabel,
    required this.state,
    required this.hasProxy,
    required this.dnsProxied,
    required this.addressController,
    required this.addressFocusNode,
    required this.controller,
    required this.onNavigate,
    required this.onBack,
    required this.onForward,
    required this.onSwitchSession,
  });

  final String endpointLabel;
  final String? preparingEndpointLabel;
  final BrowserSessionState state;
  final bool hasProxy;
  final bool dnsProxied;
  final TextEditingController addressController;
  final FocusNode addressFocusNode;
  final WebViewController? controller;
  final VoidCallback onNavigate;
  final VoidCallback onBack;
  final VoidCallback onForward;
  final VoidCallback onSwitchSession;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: addressFocusNode,
      builder: (context, _) => LayoutBuilder(
        builder: (context, constraints) {
          final showInlineNavigation = constraints.maxWidth >= 560;
          final addressFocused = addressFocusNode.hasFocus;
          return Row(
            children: [
              if (showInlineNavigation) ...[
                _BrowserIconButton(
                  tooltip: anyttyText(context, en: 'Back', zh: '后退'),
                  enabled: controller != null,
                  onPressed: onBack,
                  icon: Icons.arrow_back_rounded,
                ),
                _BrowserIconButton(
                  tooltip: anyttyText(context, en: 'Forward', zh: '前进'),
                  enabled: controller != null,
                  onPressed: onForward,
                  icon: Icons.arrow_forward_rounded,
                ),
                const SizedBox(width: 4),
              ],
              _BrowserSessionControl(
                endpointLabel: endpointLabel,
                state: state,
                hasProxy: hasProxy,
                dnsProxied: dnsProxied,
                compact: !showInlineNavigation,
                collapsed: addressFocused,
                preparingEndpointLabel: preparingEndpointLabel,
                onTap: onSwitchSession,
              ),
              const SizedBox(width: 6),
              Expanded(
                child: _BrowserAddressField(
                  addressController: addressController,
                  addressFocusNode: addressFocusNode,
                  controller: controller,
                  focused: addressFocused,
                  onNavigate: onNavigate,
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

final class _BrowserSessionControl extends StatelessWidget {
  const _BrowserSessionControl({
    required this.endpointLabel,
    required this.state,
    required this.hasProxy,
    required this.dnsProxied,
    required this.compact,
    required this.collapsed,
    required this.preparingEndpointLabel,
    required this.onTap,
  });

  final String endpointLabel;
  final BrowserSessionState state;
  final bool hasProxy;
  final bool dnsProxied;
  final bool compact;
  final bool collapsed;
  final String? preparingEndpointLabel;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final status = _browserStatus(
      context,
      state: state,
      hasProxy: hasProxy,
      dnsProxied: dnsProxied,
    );
    final switching = preparingEndpointLabel != null;
    final statusColor = switching ? palette.accent : status.color;
    return Tooltip(
      message: switching
          ? anyttyText(
              context,
              en: 'Preparing $preparingEndpointLabel',
              zh: '正在准备 $preparingEndpointLabel',
            )
          : '$endpointLabel\n${status.label}',
      child: InkWell(
        borderRadius: BorderRadius.circular(6),
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOutCubic,
          width: collapsed ? 36 : (compact ? 86 : 132),
          height: 48,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Row(
              children: [
                Icon(
                  switching ? Icons.sync_rounded : Icons.devices_rounded,
                  size: 18,
                  color: statusColor,
                ),
                if (!collapsed) ...[
                  const SizedBox(width: 4),
                  Expanded(
                    child: Text(
                      endpointLabel,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontSize: compact ? 11 : 12,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  const SizedBox(width: 4),
                ],
                Container(
                  width: 7,
                  height: 7,
                  decoration: BoxDecoration(
                    color: statusColor,
                    shape: BoxShape.circle,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _BrowserAddressField extends StatelessWidget {
  const _BrowserAddressField({
    required this.addressController,
    required this.addressFocusNode,
    required this.controller,
    required this.focused,
    required this.onNavigate,
  });

  final TextEditingController addressController;
  final FocusNode addressFocusNode;
  final WebViewController? controller;
  final bool focused;
  final VoidCallback onNavigate;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final enabled = controller != null;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      curve: Curves.easeOutCubic,
      height: focused ? 46 : 40,
      child: TextField(
        controller: addressController,
        focusNode: addressFocusNode,
        enabled: enabled,
        onSubmitted: (_) => onNavigate(),
        textInputAction: TextInputAction.go,
        keyboardType: TextInputType.url,
        maxLines: 1,
        style: TextStyle(color: palette.text, fontSize: 14),
        decoration: InputDecoration(
          hintText: anyttyText(context, en: 'Enter a URL', zh: '输入网址'),
          hintStyle: TextStyle(color: palette.faint, fontSize: 14),
          prefixIcon: Icon(
            Icons.lock_outline_rounded,
            size: 16,
            color: palette.muted,
          ),
          prefixIconConstraints: const BoxConstraints(
            minWidth: 38,
            minHeight: 40,
          ),
          suffixIcon: IconButton(
            tooltip: anyttyText(context, en: 'Open', zh: '打开'),
            onPressed: enabled ? onNavigate : null,
            icon: const Icon(Icons.arrow_forward_rounded, size: 18),
            color: palette.accent,
            constraints: const BoxConstraints.tightFor(width: 40, height: 40),
            padding: EdgeInsets.zero,
          ),
          filled: true,
          fillColor: palette.surfaceRaised,
          isDense: true,
          contentPadding: EdgeInsets.symmetric(
            horizontal: 8,
            vertical: focused ? 11 : 8,
          ),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(6),
            borderSide: BorderSide(color: palette.border),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(6),
            borderSide: BorderSide(color: palette.border),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(6),
            borderSide: BorderSide(color: palette.accent, width: 1.2),
          ),
        ),
      ),
    );
  }
}

final class _BrowserOverflowMenu extends StatelessWidget {
  const _BrowserOverflowMenu({
    required this.state,
    required this.hasProxy,
    required this.dnsProxied,
    required this.controller,
    required this.onBack,
    required this.onForward,
    required this.onSwitchSession,
  });

  final BrowserSessionState state;
  final bool hasProxy;
  final bool dnsProxied;
  final WebViewController? controller;
  final VoidCallback onBack;
  final VoidCallback onForward;
  final VoidCallback onSwitchSession;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final status = _browserStatus(
      context,
      state: state,
      hasProxy: hasProxy,
      dnsProxied: dnsProxied,
    );
    return MenuAnchor(
      style: MenuStyle(
        minimumSize: const WidgetStatePropertyAll(Size(220, 0)),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(vertical: 6),
        ),
      ),
      menuChildren: [
        MenuItemButton(
          leadingIcon: const Icon(Icons.arrow_back_rounded, size: 18),
          onPressed: controller != null ? onBack : null,
          child: Text(anyttyText(context, en: 'Back', zh: '后退')),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.arrow_forward_rounded, size: 18),
          onPressed: controller != null ? onForward : null,
          child: Text(anyttyText(context, en: 'Forward', zh: '前进')),
        ),
        const Divider(height: 1),
        MenuItemButton(
          leadingIcon: const Icon(Icons.devices_rounded, size: 18),
          onPressed: onSwitchSession,
          child: Text(anyttyText(context, en: 'Switch session', zh: '切换会话')),
        ),
        SizedBox(
          width: 220,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 6),
            child: Row(
              children: [
                Icon(status.icon, size: 18, color: status.color),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    status.label,
                    style: TextStyle(color: palette.muted, fontSize: 12),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
      builder: (context, menuController, child) => IconButton(
        tooltip: anyttyText(context, en: 'More browser actions', zh: '更多浏览器操作'),
        onPressed: () {
          if (menuController.isOpen) {
            menuController.close();
          } else {
            menuController.open();
          }
        },
        icon: const Icon(Icons.more_vert_rounded, size: 21),
        padding: EdgeInsets.zero,
        constraints: const BoxConstraints.tightFor(width: 44, height: 48),
      ),
    );
  }
}

final class _BrowserStatus {
  const _BrowserStatus({
    required this.label,
    required this.icon,
    required this.color,
  });

  final String label;
  final IconData icon;
  final Color color;
}

_BrowserStatus _browserStatus(
  BuildContext context, {
  required BrowserSessionState state,
  required bool hasProxy,
  required bool dnsProxied,
}) {
  final palette = AnyttyPalette.of(context);
  final active = state.phase == BrowserSessionPhase.active && hasProxy;
  final blocked = state.phase == BrowserSessionPhase.blocked;
  if (active) {
    return _BrowserStatus(
      label: dnsProxied
          ? anyttyText(context, en: 'Tunnel · DNS remote', zh: '隧道 · DNS 已代理')
          : anyttyText(context, en: 'Tunnel active', zh: '隧道已连接'),
      icon: Icons.shield_outlined,
      color: palette.success,
    );
  }
  return _BrowserStatus(
    label: blocked
        ? anyttyText(context, en: 'Route unavailable', zh: '路由不可用')
        : _browserPhaseText(context, state.phase),
    icon: blocked ? Icons.route_rounded : Icons.more_horiz_rounded,
    color: blocked ? palette.warning : palette.muted,
  );
}

String _browserPhaseText(BuildContext context, BrowserSessionPhase phase) {
  return switch (phase) {
    BrowserSessionPhase.parked => anyttyText(context, en: 'Paused', zh: '已暂停'),
    BrowserSessionPhase.restoring => anyttyText(
      context,
      en: 'Restoring',
      zh: '正在恢复',
    ),
    BrowserSessionPhase.active => anyttyText(
      context,
      en: 'Starting',
      zh: '正在启动',
    ),
    BrowserSessionPhase.parking => anyttyText(
      context,
      en: 'Saving session',
      zh: '正在保存会话',
    ),
    BrowserSessionPhase.blocked => anyttyText(
      context,
      en: 'Route unavailable',
      zh: '路由不可用',
    ),
    BrowserSessionPhase.failed => anyttyText(
      context,
      en: 'Browser error',
      zh: '浏览器错误',
    ),
  };
}

final class _BrowserIconButton extends StatelessWidget {
  const _BrowserIconButton({
    required this.tooltip,
    required this.enabled,
    required this.onPressed,
    required this.icon,
  });

  final String tooltip;
  final bool enabled;
  final VoidCallback onPressed;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Tooltip(
      message: tooltip,
      child: IconButton(
        constraints: const BoxConstraints.tightFor(width: 44, height: 44),
        padding: const EdgeInsets.all(11),
        onPressed: enabled ? onPressed : null,
        color: palette.muted,
        icon: Icon(icon, size: 18),
      ),
    );
  }
}

final class _BrowserLoadingMarker extends StatelessWidget {
  const _BrowserLoadingMarker({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      margin: const EdgeInsets.only(top: 12),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: palette.surface.withValues(alpha: 0.94),
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(5),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          SizedBox(
            width: 13,
            height: 13,
            child: CircularProgressIndicator(
              strokeWidth: 1.8,
              color: palette.accent,
            ),
          ),
          const SizedBox(width: 7),
          Text(label, style: TextStyle(color: palette.text, fontSize: 11)),
        ],
      ),
    );
  }
}
