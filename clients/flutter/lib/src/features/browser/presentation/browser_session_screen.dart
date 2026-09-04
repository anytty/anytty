import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/providers.dart';
import '../../../native/browser_proxy_platform.dart';
import '../data/browser_http_proxy.dart';
import '../data/browser_history_store.dart';
import '../data/browser_session_store.dart';
import '../domain/browser_load_progress.dart';
import '../domain/browser_session.dart';
import '../../terminal/data/endpoint_session_client.dart';
import '../../terminal/presentation/terminal_petal_menu.dart';
import 'browser_endpoint_picker_sheet.dart';

final class BrowserSessionScreen extends ConsumerStatefulWidget {
  const BrowserSessionScreen({
    super.key,
    required this.endpointId,
    this.endpointLabel,
    this.proxyPlatform,
    this.sessionStore,
    this.historyStore,
    this.onExit,
    this.navigationRequestId = 0,
    this.navigationUrl,
  });

  final String endpointId;
  final String? endpointLabel;
  final BrowserProxyPlatform? proxyPlatform;
  final BrowserSessionStore? sessionStore;
  final BrowserHistoryStore? historyStore;
  final VoidCallback? onExit;
  final int navigationRequestId;
  final String? navigationUrl;

  @override
  ConsumerState<BrowserSessionScreen> createState() =>
      _BrowserSessionScreenState();
}

final class _BrowserSessionScreenState
    extends ConsumerState<BrowserSessionScreen> {
  static const _sessionPreparationTimeout = Duration(seconds: 8);
  static const _desktopUserAgent =
      'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 '
      '(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36';

  final _addressController = TextEditingController();
  final _addressFocusNode = FocusNode();
  final _stateMachine = BrowserSessionStateMachine();

  late final BrowserProxyPlatform _proxyPlatform;
  late final BrowserSessionStore _sessionStore;
  late final BrowserHistoryStore _historyStore;

  String _activeEndpointId = '';
  String _activeEndpointLabel = '';
  BrowserSessionSnapshot? _snapshot;
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
  List<BrowserHistoryEntry> _history = const [];
  final _tabsByEndpoint = <String, List<BrowserTabSnapshot>>{};
  final _activeTabIds = <String, String?>{};
  bool _readerMode = false;
  bool _desktopMode = false;
  Timer? _loadProgressTimer;
  DateTime? _loadStartedAt;
  int _loadProgressGeneration = 0;
  double _loadProgress = 0;
  String? _pendingNavigationUrl;
  Future<void> _transitionTail = Future<void>.value();

  @override
  void initState() {
    super.initState();
    _proxyPlatform =
        widget.proxyPlatform ?? MethodChannelBrowserProxyPlatform.instance;
    _sessionStore =
        widget.sessionStore ?? const SharedPreferencesBrowserSessionStore();
    _historyStore =
        widget.historyStore ?? const SharedPreferencesBrowserHistoryStore();
    _activeEndpointId = widget.endpointId;
    _activeEndpointLabel = _labelFor(widget.endpointId, widget.endpointLabel);
    _pendingNavigationUrl = widget.navigationUrl;
    _retainEndpointSession(_activeEndpointId);
    unawaited(_loadHistory());
    unawaited(_activateSession(_activeEndpointId, _activeEndpointLabel));
  }

  @override
  void dispose() {
    _closing = true;
    _loadProgressTimer?.cancel();
    _loadProgressTimer = null;
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
  void didUpdateWidget(BrowserSessionScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.navigationRequestId == oldWidget.navigationRequestId ||
        widget.navigationUrl == null) {
      return;
    }
    _pendingNavigationUrl = widget.navigationUrl;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) unawaited(_navigate(_pendingNavigationUrl));
    });
  }

  Future<void> _loadHistory() async {
    try {
      final history = await _historyStore.load();
      if (mounted) setState(() => _history = history);
    } catch (_) {
      // History is an enhancement; a corrupt or unavailable store must not
      // prevent the remote browser from opening.
    }
  }

  List<BrowserTabSnapshot> _tabsFor(String endpointId) =>
      _tabsByEndpoint[endpointId] ?? const <BrowserTabSnapshot>[];

  String _tabIdFor(String endpointId, [int index = 0]) =>
      '$endpointId-tab-$index';

  String? _activeTabIdFor(String endpointId) => _activeTabIds[endpointId];

  BrowserTabSnapshot? _activeTabFor(String endpointId) {
    final tabs = _tabsFor(endpointId);
    final activeId = _activeTabIdFor(endpointId);
    for (final tab in tabs) {
      if (tab.id == activeId) return tab;
    }
    return tabs.isEmpty ? null : tabs.first;
  }

  void _installSessionTabs(BrowserSessionSnapshot snapshot) {
    final endpointId = snapshot.endpointId;
    final tabs = snapshot.tabs.isEmpty
        ? [
            BrowserTabSnapshot(
              id: snapshot.activeTabId ?? _tabIdFor(endpointId),
              url: snapshot.url,
              title: snapshot.title,
              scrollX: snapshot.scrollX,
              scrollY: snapshot.scrollY,
            ),
          ]
        : List<BrowserTabSnapshot>.of(snapshot.tabs);
    final activeId =
        snapshot.activeTabId != null &&
            tabs.any((tab) => tab.id == snapshot.activeTabId)
        ? snapshot.activeTabId!
        : tabs.first.id;
    _tabsByEndpoint[endpointId] = List<BrowserTabSnapshot>.unmodifiable(tabs);
    _activeTabIds[endpointId] = activeId;
  }

  void _replaceActiveTab(BrowserSessionSnapshot snapshot) {
    final endpointId = snapshot.endpointId;
    final tabs = List<BrowserTabSnapshot>.of(_tabsFor(endpointId));
    final activeId =
        _activeTabIdFor(endpointId) ??
        snapshot.activeTabId ??
        _tabIdFor(endpointId);
    final updated = BrowserTabSnapshot(
      id: activeId,
      url: snapshot.url,
      title: snapshot.title,
      scrollX: snapshot.scrollX,
      scrollY: snapshot.scrollY,
    );
    final index = tabs.indexWhere((tab) => tab.id == activeId);
    if (index == -1) {
      tabs.add(updated);
    } else {
      tabs[index] = updated;
    }
    _tabsByEndpoint[endpointId] = List<BrowserTabSnapshot>.unmodifiable(tabs);
    _activeTabIds[endpointId] = activeId;
  }

  BrowserSessionSnapshot _withTabState(BrowserSessionSnapshot snapshot) {
    final tabs = _tabsFor(snapshot.endpointId);
    return snapshot.copyWith(
      tabs: tabs,
      activeTabId: _activeTabIdFor(snapshot.endpointId),
    );
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final state = _stateMachine.state;
    final compact = MediaQuery.sizeOf(context).width < 600;
    final petalPreferences = ref
        .watch(terminalPetalMenuPreferencesProvider)
        .valueOrNull;
    final browser = PopScope<Object?>(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) unawaited(_closeScreen());
      },
      child: Scaffold(
        backgroundColor: palette.background,
        resizeToAvoidBottomInset: false,
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
            addressController: _addressController,
            addressFocusNode: _addressFocusNode,
            controller: _webViewController,
            onNavigate: _navigate,
            onBack: () => _goBack(_webViewController),
            onForward: () => _goForward(_webViewController),
            history: _history,
          ),
          actions: [
            if (compact)
              _BrowserTabCountButton(
                count: _tabsFor(_activeEndpointId).length,
                onPressed: _openTabSwitcher,
              )
            else
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
              onReload: () => _reload(_webViewController),
              onSwitchSession: () => unawaited(_openEndpointPicker()),
              onOpenTabs: _openTabSwitcher,
              onOpenHistory: _openHistory,
              onOpenSettings: _openBrowserSettings,
              readerMode: _readerMode,
              desktopMode: _desktopMode,
              onToggleReaderMode: _toggleReaderMode,
              onToggleDesktopMode: _toggleDesktopMode,
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Column(children: [Expanded(child: _buildContent(context))]),
        ),
      ),
    );
    return TerminalPetalMenuOverlay(
      child: TerminalPetalMenuRegion(
        actions: _browserPetalActions(context),
        enabled: petalPreferences?.enabled ?? true,
        hapticsEnabled: petalPreferences?.hapticsEnabled ?? true,
        onOpened: _addressFocusNode.unfocus,
        onSelected: (action) => unawaited(_handleBrowserPetalAction(action.id)),
        child: browser,
      ),
    );
  }

  List<TerminalPetalMenuItem> _browserPetalActions(BuildContext context) => [
    TerminalPetalMenuItem(
      id: 'browser-navigation',
      label: anyttyText(context, en: 'Navigate', zh: '导航'),
      icon: LucideIcons.navigation,
      enabled: _webViewController != null,
      children: [
        TerminalPetalMenuItem(
          id: 'browser-back',
          label: anyttyText(context, en: 'Back', zh: '后退'),
          icon: LucideIcons.arrowLeft,
          enabled: _webViewController != null,
        ),
        TerminalPetalMenuItem(
          id: 'browser-forward',
          label: anyttyText(context, en: 'Forward', zh: '前进'),
          icon: LucideIcons.arrowRight,
          enabled: _webViewController != null,
        ),
        TerminalPetalMenuItem(
          id: 'browser-reload',
          label: anyttyText(context, en: 'Reload', zh: '刷新'),
          icon: LucideIcons.refreshCw,
          enabled: _webViewController != null,
        ),
      ],
    ),
    TerminalPetalMenuItem(
      id: 'browser-tabs',
      label: anyttyText(context, en: 'Tabs', zh: '标签页'),
      icon: LucideIcons.panelsTopLeft,
      children: [
        TerminalPetalMenuItem(
          id: 'browser-new-tab',
          label: anyttyText(context, en: 'New tab', zh: '新建标签页'),
          icon: LucideIcons.plus,
        ),
        TerminalPetalMenuItem(
          id: 'browser-open-tabs',
          label: anyttyText(context, en: 'Switch tab', zh: '切换标签页'),
          icon: LucideIcons.listFilter,
        ),
      ],
    ),
    TerminalPetalMenuItem(
      id: 'browser-history',
      label: anyttyText(context, en: 'History', zh: '历史记录'),
      icon: LucideIcons.history,
    ),
    TerminalPetalMenuItem(
      id: 'browser-reader',
      label: anyttyText(context, en: 'Reader', zh: '阅读模式'),
      icon: LucideIcons.copy,
      enabled: _webViewController != null,
    ),
    TerminalPetalMenuItem(
      id: 'browser-desktop',
      label: anyttyText(context, en: 'Desktop site', zh: '电脑模式'),
      icon: LucideIcons.monitor,
      enabled: _webViewController != null,
    ),
    TerminalPetalMenuItem(
      id: 'browser-session',
      label: anyttyText(context, en: 'Session', zh: '会话'),
      icon: LucideIcons.gitCompareArrows,
      children: [
        TerminalPetalMenuItem(
          id: 'browser-switch-session',
          label: anyttyText(context, en: 'Switch session', zh: '切换会话'),
          icon: LucideIcons.gitCompareArrows,
        ),
        TerminalPetalMenuItem(
          id: 'browser-settings',
          label: anyttyText(context, en: 'Browser settings', zh: '浏览器设置'),
          icon: LucideIcons.settings,
        ),
        TerminalPetalMenuItem(
          id: 'browser-exit',
          label: anyttyText(context, en: 'Close browser', zh: '关闭浏览器'),
          icon: LucideIcons.x,
        ),
      ],
    ),
  ];

  Future<void> _handleBrowserPetalAction(String id) async {
    switch (id) {
      case 'browser-back':
        _goBack(_webViewController);
      case 'browser-forward':
        _goForward(_webViewController);
      case 'browser-reload':
        _reload(_webViewController);
      case 'browser-new-tab':
        await _createTab();
      case 'browser-open-tabs':
        await _openTabSwitcher();
      case 'browser-history':
        await _openHistory();
      case 'browser-reader':
        await _toggleReaderMode();
      case 'browser-desktop':
        await _toggleDesktopMode();
      case 'browser-switch-session':
        await _openEndpointPicker();
      case 'browser-settings':
        await _openBrowserSettings();
      case 'browser-exit':
        await _closeScreen();
    }
  }

  Widget _buildContent(BuildContext context) {
    final state = _stateMachine.state;
    final controller = _webViewController;
    final palette = AnyttyPalette.of(context);
    if (controller != null) {
      return Stack(
        fit: StackFit.expand,
        children: [
          WebViewWidget(controller: controller),
          if (!_pageReady)
            Positioned(
              top: 0,
              left: 0,
              right: 0,
              child: _BrowserLoadProgress(value: _loadProgress),
            ),
          if (_error != null)
            Align(
              alignment: Alignment.topCenter,
              child: _BrowserConnectionBanner(
                message: _error!,
                onRetry: () => unawaited(
                  _activateSession(
                    _activeEndpointId,
                    _activeEndpointLabel,
                    forceReconnect: true,
                  ),
                ),
              ),
            ),
        ],
      );
    }
    final blocked = state.phase == BrowserSessionPhase.blocked;
    if (!blocked && state.phase != BrowserSessionPhase.failed) {
      return Semantics(
        container: true,
        liveRegion: true,
        label: anyttyText(context, en: 'Loading page', zh: '正在加载页面'),
        child: _BrowserLoadingSurface(value: _loadProgress),
      );
    }
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
                        en: 'Browser unavailable',
                        zh: '浏览器不可用',
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
                        en: 'The page could not be opened. Check the connection and try again.',
                        zh: '网页暂时无法打开，请检查连接后重试。',
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
      setState(() => _error = null);
    }

    _PreparedBrowserSession prepared;
    try {
      prepared = await _prepareSession(
        endpointId: endpointId,
        endpointLabel: endpointLabel,
      );
    } catch (error) {
      if (!switching && _webViewController != null) {
        final operation = _stateMachine.begin(endpointId);
        final message = '$error';
        if (error is BrowserProxyUnavailableException) {
          _stateMachine.markBlocked(operation, endpointId, message);
        } else {
          _stateMachine.markFailed(operation, endpointId, error);
        }
        if (mounted) {
          setState(() {
            _error = message;
            _pageReady = true;
            _loadProgress = 0;
          });
        }
        return;
      }
      if (mounted && switching) {
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
        setState(() => _error = message);
      }
      return;
    }

    if (mounted) {
      setState(() {
        _error = null;
        _pageReady = false;
      });
    }
    final operation = _stateMachine.begin(endpointId);
    if (!await _parkLiveSession()) {
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
    _installSessionTabs(restoredSnapshot);
    _stateMachine.markRestoring(operation, endpointId, restoredSnapshot);
    setState(() {
      _snapshot = restoredSnapshot;
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
      await controller.setUserAgent(_desktopMode ? _desktopUserAgent : null);
      await controller.setBackgroundColor(Colors.transparent);
      await controller.setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (url) {
            if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) {
              return;
            }
            _beginPageLoad();
          },
          onPageFinished: (url) {
            final restoreScroll = restoreScrollPending;
            restoreScrollPending = false;
            unawaited(
              _finishPageLoad(
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
    _replaceActiveTab(activeSnapshot);
    _stateMachine.markActive(operation, endpointId, activeSnapshot);
    setState(() {
      _pageReady = activeSnapshot.restorableUri == null;
      _loadProgress = activeSnapshot.restorableUri == null
          ? 0
          : browserFakeLoadProgress(Duration.zero);
      _error = null;
    });
    final pendingUrl = _pendingNavigationUrl;
    if (pendingUrl != null) {
      _pendingNavigationUrl = null;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) unawaited(_navigate(pendingUrl));
      });
    }
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
      return _PreparedBrowserSession(
        endpointSession: endpointSession,
        httpProxy: httpProxy,
        endpointSubscription: endpointSubscription,
        snapshot: snapshot,
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

  void _beginPageLoad() {
    _loadProgressTimer?.cancel();
    final generation = ++_loadProgressGeneration;
    _loadStartedAt = DateTime.now();
    if (mounted) {
      setState(() {
        _pageReady = false;
        _loadProgress = browserFakeLoadProgress(Duration.zero);
      });
    }
    _loadProgressTimer = Timer.periodic(const Duration(milliseconds: 120), (_) {
      if (!mounted || generation != _loadProgressGeneration) {
        _loadProgressTimer?.cancel();
        return;
      }
      final startedAt = _loadStartedAt;
      if (startedAt == null) return;
      final progress = browserFakeLoadProgress(
        DateTime.now().difference(startedAt),
      );
      if (progress != _loadProgress) {
        setState(() => _loadProgress = progress);
      }
    });
  }

  Future<void> _finishPageLoad({
    required int operation,
    required String endpointId,
    required WebViewController controller,
    required String url,
    required bool restoreScroll,
  }) async {
    await _pageFinished(
      operation: operation,
      endpointId: endpointId,
      controller: controller,
      url: url,
      restoreScroll: restoreScroll,
    );
    if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) return;
    _loadProgressTimer?.cancel();
    _loadProgressTimer = null;
    setState(() {
      _loadProgress = 1;
      _pageReady = true;
    });
    await Future<void>.delayed(const Duration(milliseconds: 220));
    if (!_stateMachine.isCurrent(operation, endpointId) || !mounted) return;
    setState(() => _loadProgress = 0);
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
      _replaceActiveTab(next);
      _stateMachine.markActive(operation, endpointId, next);
      if (!_addressFocusNode.hasFocus) _addressController.text = currentUrl;
      await _sessionStore.save(_withTabState(next));
      unawaited(
        _recordHistory(
          BrowserHistoryEntry(
            url: currentUrl,
            title: title.isEmpty ? currentUrl : title,
          ),
        ),
      );
      if (_readerMode) await _applyReaderMode(controller, enabled: true);
    } catch (error) {
      if (mounted && _stateMachine.isCurrent(operation, endpointId)) {
        setState(() => _error = '$error');
      }
    }
  }

  Future<void> _applyReaderMode(
    WebViewController controller, {
    required bool enabled,
  }) async {
    final script = enabled
        ? '''(() => {
  const old = document.getElementById('anytty-reader-style');
  if (old) old.remove();
  const style = document.createElement('style');
  style.id = 'anytty-reader-style';
  style.textContent = 'body { max-width: 760px !important; margin: 0 auto !important; padding: 24px !important; background: #fffdf8 !important; color: #17201f !important; } nav, aside, header, footer, video, iframe, [role="banner"], [role="navigation"] { display: none !important; } p, li { font-size: 1.12em !important; line-height: 1.8 !important; }';
  document.head.appendChild(style);
})()'''
        : '''document.getElementById('anytty-reader-style')?.remove();''';
    try {
      await controller.runJavaScript(script);
    } catch (_) {}
  }

  Future<void> _setReaderMode(bool enabled) async {
    if (mounted) setState(() => _readerMode = enabled);
    final controller = _webViewController;
    if (controller != null) {
      await _applyReaderMode(controller, enabled: enabled);
    }
  }

  Future<void> _toggleReaderMode() => _setReaderMode(!_readerMode);

  Future<void> _setDesktopMode(bool enabled) async {
    if (mounted) {
      setState(() {
        _desktopMode = enabled;
        _pageReady = false;
      });
    }
    final controller = _webViewController;
    if (controller == null) return;
    try {
      await controller.setUserAgent(enabled ? _desktopUserAgent : null);
      _beginPageLoad();
      await controller.reload();
    } catch (error) {
      if (mounted) setState(() => _error = '$error');
    }
  }

  Future<void> _toggleDesktopMode() => _setDesktopMode(!_desktopMode);

  Future<void> _navigate([String? requestedValue]) async {
    final controller = _webViewController;
    final value = (requestedValue ?? _addressController.text).trim();
    if (value.isEmpty) return;
    if (controller == null || _proxyLease == null) {
      if (requestedValue != null) {
        _pendingNavigationUrl = requestedValue;
        if (_activeEndpointId.isNotEmpty) {
          unawaited(
            _activateSession(
              _activeEndpointId,
              _activeEndpointLabel,
              forceReconnect: true,
            ),
          );
        }
      }
      return;
    }
    if (requestedValue != null && _pendingNavigationUrl == requestedValue) {
      _pendingNavigationUrl = null;
    }
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
    _beginPageLoad();
    setState(() {
      _error = null;
      _pageReady = false;
    });
    _recordHistory(
      BrowserHistoryEntry(
        url: uri.toString(),
        title: uri.host.isEmpty ? uri.toString() : uri.host,
      ),
    );
    await controller.loadRequest(uri);
  }

  Future<void> _recordHistory(BrowserHistoryEntry entry) async {
    final next = <BrowserHistoryEntry>[
      entry,
      ..._history.where((item) => item.url != entry.url),
    ].take(20).toList(growable: false);
    if (mounted) setState(() => _history = next);
    try {
      await _historyStore.add(entry);
    } catch (_) {}
  }

  Future<bool> _parkLiveSession() async {
    final controller = _webViewController;
    final lease = _proxyLease;
    final httpProxy = _httpProxy;
    if (controller == null && lease == null && httpProxy == null) {
      return _clearBrowserData(notify: false);
    }
    if (mounted) {
      setState(() => _pageReady = false);
    }
    _loadProgressTimer?.cancel();
    _loadProgressTimer = null;
    var parked = await _captureSession(
      sessionId: _activeEndpointId,
      endpointLabel: _activeEndpointLabel,
      controller: controller,
      lease: lease,
      previous: _snapshot,
    );
    if (parked != null) {
      parked = _withTabState(parked);
      await _sessionStore.save(parked);
      _snapshot = parked;
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
    return _clearBrowserData(notify: false);
  }

  Future<bool> _clearBrowserData({bool notify = true}) async {
    try {
      await _proxyPlatform.clearBrowserData();
      if (notify && mounted && !_closing) {
        ScaffoldMessenger.of(context)
          ..hideCurrentSnackBar()
          ..showSnackBar(
            SnackBar(
              content: Text(
                anyttyText(
                  context,
                  en: 'Cache and site data cleared',
                  zh: '缓存和网站数据已清除',
                ),
              ),
            ),
          );
      }
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
    }
    _replaceActiveTab(current);
    return _withTabState(current);
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
    if (widget.onExit != null) {
      _addressFocusNode.unfocus();
      widget.onExit!();
      return;
    }
    if (_closing) return;
    _closing = true;
    await _parkLiveSession();
    if (mounted) Navigator.of(context).pop();
  }

  BrowserSessionSnapshot _snapshotForTab(BrowserTabSnapshot tab) {
    final base =
        _snapshot ??
        BrowserSessionSnapshot.empty(
          sessionId: _activeEndpointId,
          endpointId: _activeEndpointId,
          endpointLabel: _activeEndpointLabel,
        );
    return _withTabState(
      base.copyWith(
        url: tab.url,
        title: tab.title,
        scrollX: tab.scrollX,
        scrollY: tab.scrollY,
        parkedAt: null,
        activeTabId: tab.id,
      ),
    );
  }

  Future<void> _resumeTab(BrowserTabSnapshot tab) async {
    _activeTabIds[_activeEndpointId] = tab.id;
    final snapshot = _snapshotForTab(tab);
    _snapshot = snapshot;
    await _sessionStore.save(snapshot);
    if (mounted) {
      setState(() {
        _pageReady = false;
        _error = null;
      });
    }
    await _activateSession(_activeEndpointId, _activeEndpointLabel);
  }

  Future<void> _createTab() async {
    final endpointId = _activeEndpointId;
    await _parkLiveSession();
    if (_closing || !mounted) return;
    final tabs = List<BrowserTabSnapshot>.of(_tabsFor(endpointId));
    final tab = BrowserTabSnapshot.empty(
      id: '$endpointId-tab-${DateTime.now().microsecondsSinceEpoch}',
    );
    tabs.add(tab);
    _tabsByEndpoint[endpointId] = List<BrowserTabSnapshot>.unmodifiable(tabs);
    await _resumeTab(tab);
  }

  Future<void> _selectTab(String tabId) async {
    if (tabId == _activeTabIdFor(_activeEndpointId)) return;
    final tab = _tabsFor(_activeEndpointId)
        .cast<BrowserTabSnapshot?>()
        .firstWhere((item) => item?.id == tabId, orElse: () => null);
    if (tab == null) return;
    await _parkLiveSession();
    if (_closing || !mounted) return;
    await _resumeTab(tab);
  }

  Future<void> _closeTab(String tabId) async {
    final endpointId = _activeEndpointId;
    final tabs = List<BrowserTabSnapshot>.of(_tabsFor(endpointId));
    final index = tabs.indexWhere((tab) => tab.id == tabId);
    if (index == -1) return;
    final closingActive = tabId == _activeTabIdFor(endpointId);
    final fallback = closingActive
        ? tabs[index == 0 ? 1 : index - 1]
        : _activeTabFor(endpointId)!;
    await _parkLiveSession();
    if (_closing || !mounted) return;
    if (tabs.length == 1) {
      final blank = BrowserTabSnapshot.empty(id: tabId);
      _tabsByEndpoint[endpointId] = [blank];
      await _resumeTab(blank);
      return;
    }
    tabs.removeAt(index);
    _tabsByEndpoint[endpointId] = List<BrowserTabSnapshot>.unmodifiable(tabs);
    await _resumeTab(fallback);
  }

  Future<void> _openTabSwitcher() async {
    final action = await showModalBottomSheet<_BrowserTabAction>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      backgroundColor: AnyttyPalette.of(context).surface,
      builder: (context) => _BrowserTabsSheet(
        tabs: _tabsFor(_activeEndpointId),
        activeTabId: _activeTabIdFor(_activeEndpointId),
      ),
    );
    if (!mounted || action == null) return;
    switch (action.type) {
      case _BrowserTabActionType.newTab:
        await _createTab();
      case _BrowserTabActionType.select:
        await _selectTab(action.tabId!);
      case _BrowserTabActionType.close:
        await _closeTab(action.tabId!);
    }
  }

  Future<void> _openHistory() async {
    final entry = await showModalBottomSheet<BrowserHistoryEntry>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      backgroundColor: AnyttyPalette.of(context).surface,
      builder: (context) =>
          _BrowserHistorySheet(entries: _history, onClear: _clearHistory),
    );
    if (!mounted || entry == null) return;
    await _navigate(entry.url);
  }

  Future<void> _clearHistory() async {
    if (mounted) setState(() => _history = const []);
    try {
      await _historyStore.clear();
    } catch (_) {}
  }

  Future<bool> _confirmBrowserAction(
    BuildContext context, {
    required String title,
    required String message,
  }) async {
    return await showDialog<bool>(
          context: context,
          builder: (context) => AlertDialog(
            title: Text(title),
            content: Text(message),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(context).pop(false),
                child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
              ),
              FilledButton(
                onPressed: () => Navigator.of(context).pop(true),
                child: Text(anyttyText(context, en: 'Clear', zh: '清除')),
              ),
            ],
          ),
        ) ??
        false;
  }

  Future<void> _clearBrowserDataFromSettings() async {
    final endpointId = _activeEndpointId;
    final endpointLabel = _activeEndpointLabel;
    var cleared = true;
    if (_webViewController != null ||
        _proxyLease != null ||
        _httpProxy != null) {
      if (!await _parkLiveSession()) return;
      if (!mounted || _closing) return;
      await _activateSession(endpointId, endpointLabel, forceReconnect: true);
      cleared =
          _webViewController != null &&
          _stateMachine.state.phase == BrowserSessionPhase.active;
    } else {
      await _clearBrowserData(notify: false);
    }
    if (cleared && mounted && !_closing) {
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(
          SnackBar(
            content: Text(
              anyttyText(
                context,
                en: 'Cache and site data cleared',
                zh: '缓存和网站数据已清除',
              ),
            ),
          ),
        );
    }
  }

  Future<void> _openBrowserSettings() async {
    var readerMode = _readerMode;
    var desktopMode = _desktopMode;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      backgroundColor: AnyttyPalette.of(context).surface,
      builder: (context) => StatefulBuilder(
        builder: (context, setSheetState) => SafeArea(
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
                  child: Text(
                    anyttyText(context, en: 'Browser settings', zh: '浏览器设置'),
                    style: TextStyle(
                      color: AnyttyPalette.of(context).text,
                      fontSize: 18,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                SwitchListTile.adaptive(
                  secondary: const Icon(Icons.menu_book_outlined),
                  title: Text(
                    anyttyText(context, en: 'Reader mode', zh: '阅读模式'),
                  ),
                  subtitle: Text(
                    anyttyText(
                      context,
                      en: 'Reduce navigation and focus on page content.',
                      zh: '收起导航和装饰，只保留页面正文。',
                    ),
                  ),
                  value: readerMode,
                  onChanged: (value) {
                    setSheetState(() => readerMode = value);
                    unawaited(_setReaderMode(value));
                  },
                ),
                SwitchListTile.adaptive(
                  secondary: const Icon(Icons.desktop_windows_outlined),
                  title: Text(
                    anyttyText(context, en: 'Desktop site', zh: '电脑模式'),
                  ),
                  subtitle: Text(
                    anyttyText(
                      context,
                      en: 'Request the desktop layout for this tab.',
                      zh: '为当前标签页请求桌面布局。',
                    ),
                  ),
                  value: desktopMode,
                  onChanged: (value) {
                    setSheetState(() => desktopMode = value);
                    unawaited(_setDesktopMode(value));
                  },
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.code_rounded),
                  title: Text(
                    anyttyText(context, en: 'JavaScript', zh: 'JavaScript'),
                  ),
                  subtitle: Text(
                    anyttyText(
                      context,
                      en: 'Always enabled for modern web apps.',
                      zh: '为兼容现代网站保持开启。',
                    ),
                  ),
                  trailing: const Icon(Icons.check_circle_outline_rounded),
                ),
                ListTile(
                  leading: const Icon(Icons.history_rounded),
                  title: Text(
                    anyttyText(
                      context,
                      en: 'Clear browsing history',
                      zh: '清除浏览历史',
                    ),
                  ),
                  subtitle: Text(
                    anyttyText(
                      context,
                      en: '${_history.length} saved address${_history.length == 1 ? '' : 'es'}',
                      zh: '已保存 ${_history.length} 条地址',
                    ),
                  ),
                  onTap: () async {
                    final confirmed = await _confirmBrowserAction(
                      context,
                      title: anyttyText(
                        context,
                        en: 'Clear browsing history?',
                        zh: '清除浏览历史？',
                      ),
                      message: anyttyText(
                        context,
                        en: 'Saved addresses will be removed from this device.',
                        zh: '此设备上保存的地址将被移除。',
                      ),
                    );
                    if (!confirmed || !context.mounted) return;
                    Navigator.of(context).pop();
                    await _clearHistory();
                  },
                ),
                ListTile(
                  leading: const Icon(Icons.delete_sweep_outlined),
                  title: Text(
                    anyttyText(
                      context,
                      en: 'Clear cache and site data',
                      zh: '清除缓存和网站数据',
                    ),
                  ),
                  subtitle: Text(
                    anyttyText(
                      context,
                      en: 'Clear cookies, cache, storage, and HTTP auth.',
                      zh: '清除 Cookie、缓存、存储和 HTTP 登录信息。',
                    ),
                  ),
                  onTap: () async {
                    final confirmed = await _confirmBrowserAction(
                      context,
                      title: anyttyText(
                        context,
                        en: 'Clear cache and site data?',
                        zh: '清除缓存和网站数据？',
                      ),
                      message: anyttyText(
                        context,
                        en: 'You may need to sign in to websites again.',
                        zh: '清理后可能需要重新登录网站。',
                      ),
                    );
                    if (!confirmed || !context.mounted) return;
                    Navigator.of(context).pop();
                    await _clearBrowserDataFromSettings();
                  },
                ),
                const SizedBox(height: 8),
              ],
            ),
          ),
        ),
      ),
    );
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
  });

  final EndpointSessionClient endpointSession;
  final BrowserHttpProxy httpProxy;
  final ProviderSubscription<AsyncValue<EndpointSessionClient>>?
  endpointSubscription;
  final BrowserSessionSnapshot snapshot;

  Future<void> dispose() async {
    endpointSubscription?.close();
    await httpProxy.close();
  }
}

final class _BrowserToolbarTitle extends StatelessWidget {
  const _BrowserToolbarTitle({
    required this.addressController,
    required this.addressFocusNode,
    required this.controller,
    required this.onNavigate,
    required this.onBack,
    required this.onForward,
    required this.history,
  });

  final TextEditingController addressController;
  final FocusNode addressFocusNode;
  final WebViewController? controller;
  final Future<void> Function([String?]) onNavigate;
  final VoidCallback onBack;
  final VoidCallback onForward;
  final List<BrowserHistoryEntry> history;

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
              Expanded(
                child: _BrowserAddressField(
                  addressController: addressController,
                  addressFocusNode: addressFocusNode,
                  controller: controller,
                  focused: addressFocused,
                  onNavigate: onNavigate,
                  history: history,
                ),
              ),
            ],
          );
        },
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
    required this.history,
  });

  final TextEditingController addressController;
  final FocusNode addressFocusNode;
  final WebViewController? controller;
  final bool focused;
  final Future<void> Function([String?]) onNavigate;
  final List<BrowserHistoryEntry> history;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final enabled = controller != null;
    return RawAutocomplete<BrowserHistoryEntry>(
      key: const ValueKey('browser-address-autocomplete'),
      textEditingController: addressController,
      focusNode: addressFocusNode,
      displayStringForOption: (entry) => entry.url,
      optionsBuilder: (value) {
        final query = value.text.trim().toLowerCase();
        return history.where(
          (entry) =>
              query.isEmpty ||
              entry.url.toLowerCase().contains(query) ||
              entry.title.toLowerCase().contains(query),
        );
      },
      onSelected: (entry) => unawaited(onNavigate(entry.url)),
      optionsViewBuilder: (context, onSelected, options) {
        final entries = options.toList(growable: false);
        return Align(
          alignment: Alignment.topLeft,
          child: Material(
            color: palette.surface,
            elevation: 8,
            clipBehavior: Clip.antiAlias,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(6),
              side: BorderSide(color: palette.borderStrong),
            ),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 280),
              child: ListView.builder(
                padding: const EdgeInsets.symmetric(vertical: 6),
                shrinkWrap: true,
                itemCount: entries.length,
                itemBuilder: (context, index) {
                  final entry = entries[index];
                  return ListTile(
                    dense: true,
                    minVerticalPadding: 6,
                    leading: Icon(
                      Icons.history_rounded,
                      size: 18,
                      color: palette.muted,
                    ),
                    title: Text(
                      entry.title.isEmpty ? entry.url : entry.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    subtitle: Text(
                      entry.url,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.muted, fontSize: 11),
                    ),
                    trailing: index == 0
                        ? Text(
                            anyttyText(context, en: 'Tab', zh: 'Tab'),
                            style: TextStyle(
                              color: palette.faint,
                              fontSize: 10,
                            ),
                          )
                        : null,
                    onTap: () => onSelected(entry),
                  );
                },
              ),
            ),
          ),
        );
      },
      fieldViewBuilder: (context, controller, focusNode, onFieldSubmitted) =>
          AnimatedContainer(
            duration: const Duration(milliseconds: 180),
            curve: Curves.easeOutCubic,
            height: focused ? 46 : 40,
            child: TextField(
              controller: controller,
              focusNode: focusNode,
              enabled: enabled,
              onSubmitted: (_) {
                onFieldSubmitted();
                unawaited(onNavigate());
              },
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
                  onPressed: enabled ? () => unawaited(onNavigate()) : null,
                  icon: const Icon(Icons.arrow_forward_rounded, size: 18),
                  color: palette.accent,
                  constraints: const BoxConstraints.tightFor(
                    width: 40,
                    height: 40,
                  ),
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
    required this.onReload,
    required this.onSwitchSession,
    required this.onOpenTabs,
    required this.onOpenHistory,
    required this.onOpenSettings,
    required this.readerMode,
    required this.desktopMode,
    required this.onToggleReaderMode,
    required this.onToggleDesktopMode,
  });

  final BrowserSessionState state;
  final bool hasProxy;
  final bool dnsProxied;
  final WebViewController? controller;
  final VoidCallback onBack;
  final VoidCallback onForward;
  final VoidCallback onReload;
  final VoidCallback onSwitchSession;
  final VoidCallback onOpenTabs;
  final VoidCallback onOpenHistory;
  final VoidCallback onOpenSettings;
  final bool readerMode;
  final bool desktopMode;
  final VoidCallback onToggleReaderMode;
  final VoidCallback onToggleDesktopMode;

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
        MenuItemButton(
          leadingIcon: const Icon(Icons.refresh_rounded, size: 18),
          onPressed: controller != null ? onReload : null,
          child: Text(anyttyText(context, en: 'Reload', zh: '重新加载')),
        ),
        const Divider(height: 1),
        MenuItemButton(
          leadingIcon: const Icon(Icons.tab_rounded, size: 18),
          onPressed: onOpenTabs,
          child: Text(anyttyText(context, en: 'Tabs', zh: '标签页')),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.history_rounded, size: 18),
          onPressed: onOpenHistory,
          child: Text(anyttyText(context, en: 'History', zh: '历史记录')),
        ),
        MenuItemButton(
          leadingIcon: Icon(
            Icons.menu_book_outlined,
            color: readerMode ? palette.accent : null,
            size: 18,
          ),
          onPressed: onToggleReaderMode,
          child: Row(
            children: [
              Expanded(
                child: Text(anyttyText(context, en: 'Reader mode', zh: '阅读模式')),
              ),
              Text(
                readerMode
                    ? anyttyText(context, en: 'On', zh: '已开启')
                    : anyttyText(context, en: 'Off', zh: '关闭'),
                style: TextStyle(color: palette.muted, fontSize: 11),
              ),
            ],
          ),
        ),
        MenuItemButton(
          leadingIcon: Icon(
            Icons.desktop_windows_outlined,
            color: desktopMode ? palette.accent : null,
            size: 18,
          ),
          onPressed: onToggleDesktopMode,
          child: Row(
            children: [
              Expanded(
                child: Text(
                  anyttyText(context, en: 'Desktop site', zh: '电脑模式'),
                ),
              ),
              Text(
                desktopMode
                    ? anyttyText(context, en: 'On', zh: '已开启')
                    : anyttyText(context, en: 'Off', zh: '关闭'),
                style: TextStyle(color: palette.muted, fontSize: 11),
              ),
            ],
          ),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.settings_outlined, size: 18),
          onPressed: onOpenSettings,
          child: Text(anyttyText(context, en: 'Browser settings', zh: '浏览器设置')),
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

final class _BrowserTabCountButton extends StatelessWidget {
  const _BrowserTabCountButton({required this.count, required this.onPressed});

  final int count;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => IconButton(
    tooltip: anyttyText(context, en: 'Tabs', zh: '标签页'),
    onPressed: onPressed,
    icon: Container(
      width: 22,
      height: 22,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        border: Border.all(color: AnyttyPalette.of(context).muted),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        '$count',
        style: TextStyle(
          color: AnyttyPalette.of(context).text,
          fontSize: 11,
          fontWeight: FontWeight.w700,
        ),
      ),
    ),
  );
}

enum _BrowserTabActionType { newTab, select, close }

final class _BrowserTabAction {
  const _BrowserTabAction._(this.type, [this.tabId]);

  const _BrowserTabAction.newTab() : this._(_BrowserTabActionType.newTab);

  const _BrowserTabAction.select(String tabId)
    : this._(_BrowserTabActionType.select, tabId);

  const _BrowserTabAction.close(String tabId)
    : this._(_BrowserTabActionType.close, tabId);

  final _BrowserTabActionType type;
  final String? tabId;
}

final class _BrowserTabsSheet extends StatelessWidget {
  const _BrowserTabsSheet({required this.tabs, required this.activeTabId});

  final List<BrowserTabSnapshot> tabs;
  final String? activeTabId;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SafeArea(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: 620),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 4, 12, 10),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      anyttyText(context, en: 'Tabs', zh: '标签页'),
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 18,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  IconButton(
                    tooltip: anyttyText(context, en: 'New tab', zh: '新建标签页'),
                    onPressed: () =>
                        Navigator.of(context)
                            .pop(const _BrowserTabAction.newTab()),
                    icon: const Icon(Icons.add_rounded),
                  ),
                ],
              ),
            ),
            Flexible(
              child: ListView.separated(
                shrinkWrap: true,
                padding: const EdgeInsets.only(bottom: 12),
                itemCount: tabs.length,
                separatorBuilder: (_, _) =>
                    Divider(height: 1, indent: 72, color: palette.border),
                itemBuilder: (context, index) {
                  final tab = tabs[index];
                  final active = tab.id == activeTabId;
                  return ListTile(
                    minTileHeight: 68,
                    contentPadding: const EdgeInsets.symmetric(horizontal: 20),
                    leading: Container(
                      width: 36,
                      height: 36,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: active
                            ? palette.accent.withValues(alpha: 0.12)
                            : palette.surfaceRaised,
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        '${index + 1}',
                        style: TextStyle(
                          color: active ? palette.accent : palette.muted,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    title: Text(
                      tab.title.isEmpty
                          ? anyttyText(context, en: 'New tab', zh: '新标签页')
                          : tab.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    subtitle: Text(
                      tab.url.isEmpty ? 'about:blank' : tab.url,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                    trailing: IconButton(
                      tooltip: anyttyText(
                        context,
                        en: 'Close tab',
                        zh: '关闭标签页',
                      ),
                      onPressed: () =>
                          Navigator.of(context)
                              .pop(_BrowserTabAction.close(tab.id)),
                      icon: const Icon(Icons.close_rounded, size: 18),
                    ),
                    onTap: () =>
                        Navigator.of(context)
                            .pop(_BrowserTabAction.select(tab.id)),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _BrowserHistorySheet extends StatelessWidget {
  const _BrowserHistorySheet({required this.entries, required this.onClear});

  final List<BrowserHistoryEntry> entries;
  final Future<void> Function() onClear;

  @override
  Widget build(BuildContext context) {
    var currentEntries = entries;
    return StatefulBuilder(
      builder: (context, setState) {
        final palette = AnyttyPalette.of(context);
        return SafeArea(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 620),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 4, 12, 10),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          anyttyText(context, en: 'History', zh: '历史记录'),
                          style: TextStyle(
                            color: palette.text,
                            fontSize: 18,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      TextButton(
                        onPressed: currentEntries.isEmpty
                            ? null
                            : () {
                                setState(() => currentEntries = const []);
                                unawaited(onClear());
                              },
                        child: Text(anyttyText(context, en: 'Clear', zh: '清空')),
                      ),
                    ],
                  ),
                ),
                if (currentEntries.isEmpty)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 20, 20, 34),
                    child: Center(
                      child: Text(
                        anyttyText(
                          context,
                          en: 'No browsing history',
                          zh: '还没有浏览记录',
                        ),
                        style: TextStyle(color: palette.muted),
                      ),
                    ),
                  )
                else
                  Flexible(
                    child: ListView.separated(
                      shrinkWrap: true,
                      padding: const EdgeInsets.only(bottom: 12),
                      itemCount: currentEntries.length,
                      separatorBuilder: (_, _) =>
                          Divider(height: 1, indent: 68, color: palette.border),
                      itemBuilder: (context, index) {
                        final entry = currentEntries[index];
                        return ListTile(
                          minTileHeight: 62,
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 20,
                          ),
                          leading: Icon(
                            Icons.history_rounded,
                            color: palette.muted,
                          ),
                          title: Text(
                            entry.title.isEmpty ? entry.url : entry.title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.text,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          subtitle: Text(
                            entry.url,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 12,
                            ),
                          ),
                          onTap: () => Navigator.of(context).pop(entry),
                        );
                      },
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}

final class _BrowserConnectionBanner extends StatelessWidget {
  const _BrowserConnectionBanner({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 8, 12, 0),
      padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
      decoration: BoxDecoration(
        color: palette.warning.withValues(alpha: 0.94),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.warning),
      ),
      child: Row(
        children: [
          Icon(Icons.cloud_off_rounded, size: 18, color: palette.background),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: palette.background,
                fontSize: 12,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          TextButton.icon(
            onPressed: onRetry,
            icon: Icon(
              Icons.refresh_rounded,
              size: 16,
              color: palette.background,
            ),
            label: Text(
              anyttyText(context, en: 'Retry', zh: '重试'),
              style: TextStyle(color: palette.background, fontSize: 12),
            ),
            style: TextButton.styleFrom(
              minimumSize: const Size(0, 40),
              padding: const EdgeInsets.symmetric(horizontal: 8),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
          ),
        ],
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
      en: 'Loading page',
      zh: '正在加载页面',
    ),
    BrowserSessionPhase.active => anyttyText(
      context,
      en: 'Starting',
      zh: '正在启动',
    ),
    BrowserSessionPhase.parking => anyttyText(
      context,
      en: 'Closing page',
      zh: '正在关闭页面',
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

final class _BrowserLoadingSurface extends StatelessWidget {
  const _BrowserLoadingSurface({required this.value});

  final double value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: SizedBox(
          width: 220,
          child: Semantics(
            label: anyttyText(context, en: 'Loading page', zh: '正在加载页面'),
            value: '${(value * 100).round()}%',
            child: _BrowserLoadProgress(value: value),
          ),
        ),
      ),
    );
  }
}

final class _BrowserLoadProgress extends StatelessWidget {
  const _BrowserLoadProgress({required this.value});

  final double value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 2,
      child: LinearProgressIndicator(
        value: value <= 0 ? null : value.clamp(0, 1),
        backgroundColor: palette.border.withValues(alpha: 0.35),
        color: palette.accent,
        minHeight: 2,
      ),
    );
  }
}
