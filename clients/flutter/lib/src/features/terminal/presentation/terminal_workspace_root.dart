part of 'terminal_workspace_screen.dart';

final class TerminalWorkspaceScreen extends ConsumerStatefulWidget {
  const TerminalWorkspaceScreen({
    super.key,
    required this.endpointId,
    this.terminalId,
    this.label,
  });

  final String endpointId;
  final String? terminalId;
  final String? label;

  @override
  ConsumerState<TerminalWorkspaceScreen> createState() =>
      _TerminalWorkspaceScreenState();
}

final class _TerminalWorkspaceScreenState
    extends ConsumerState<TerminalWorkspaceScreen>
    with WidgetsBindingObserver {
  final _surfaceKey = GlobalKey<_ActiveTerminalState>();
  late final ui.FlutterView _view;
  late final TerminalKeyboardInsetStabilizer _keyboardInsetStabilizer;
  StreamSubscription<double>? _nativeImeInsetSubscription;
  final ValueNotifier<double> _keyboardVisualInset = ValueNotifier(0);
  String? _activeTerminalId;
  double _keyboardInset = 0;
  bool _usingNativeKeyboardInsets = false;
  bool _browserOpened = false;
  bool _browserVisible = false;
  int _browserNavigationRequestId = 0;
  String? _browserNavigationUrl;

  @override
  void initState() {
    super.initState();
    // 保留连接：进入这台机器后，切换到其他机器不再默认断开它。
    unawaited(retainEndpointConnection(ref, widget.endpointId));
    _view = WidgetsBinding.instance.platformDispatcher.views.first;
    _keyboardInsetStabilizer = TerminalKeyboardInsetStabilizer(
      onInsetChanged: (inset) {
        if (mounted) setState(() => _keyboardInset = inset);
      },
    );
    WidgetsBinding.instance.addObserver(this);
    if (defaultTargetPlatform == TargetPlatform.android) {
      final platform = AndroidImeInsetPlatform.instance;
      _nativeImeInsetSubscription = platform.physicalInsets.listen(
        _updateNativeKeyboardInset,
      );
      unawaited(
        platform.currentPhysicalInset().then((inset) {
          if (inset != null) _updateNativeKeyboardInset(inset);
        }),
      );
    }
    WidgetsBinding.instance.addPostFrameCallback((_) => _updateKeyboardInset());
  }

  @override
  void didChangeMetrics() => _updateKeyboardInset();

  void _updateKeyboardInset() {
    if (_browserVisible) return;
    if (_usingNativeKeyboardInsets) return;
    final inset = _view.viewInsets.bottom / _view.devicePixelRatio;
    _keyboardVisualInset.value = inset;
    _keyboardInsetStabilizer.update(inset);
  }

  void _updateNativeKeyboardInset(double physicalInset) {
    if (!mounted || _browserVisible) return;
    _usingNativeKeyboardInsets = true;
    final inset = math.max(0.0, physicalInset / _view.devicePixelRatio);
    _keyboardInsetStabilizer.update(inset);
    _keyboardVisualInset.value = inset;
  }

  void _handleWorkspaceBack() {
    if (!mounted) return;
    if (_browserVisible) {
      setState(() => _browserVisible = false);
      return;
    }
    if (widget.terminalId == null) {
      context.go('/');
      return;
    }
    final endpointLabel = widget.label?.trim().isNotEmpty == true
        ? widget.label!.trim()
        : widget.endpointId;
    context.go(
      '/terminal/${Uri.encodeComponent(widget.endpointId)}'
      '?label=${Uri.encodeQueryComponent(endpointLabel)}',
    );
  }

  void _showBrowser() {
    if (!mounted) return;
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _browserOpened = true;
      _browserVisible = true;
    });
  }

  Future<void> _openBrowserUrl(Uri uri) async {
    if (!mounted) return;
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _browserOpened = true;
      _browserVisible = true;
      _browserNavigationRequestId += 1;
      _browserNavigationUrl = uri.toString();
    });
  }

  @override
  void didUpdateWidget(TerminalWorkspaceScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId) {
      _activeTerminalId = widget.terminalId;
      _browserOpened = false;
      _browserVisible = false;
      _browserNavigationRequestId = 0;
      _browserNavigationUrl = null;
    } else if (oldWidget.terminalId != widget.terminalId) {
      _activeTerminalId = widget.terminalId;
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_nativeImeInsetSubscription?.cancel());
    _keyboardVisualInset.dispose();
    _keyboardInsetStabilizer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.terminalId == null) return _buildWorkspace(context);
    final settings =
        ref.watch(terminalSettingsProvider).valueOrNull ??
        defaultTerminalSettings;
    return Theme(
      data: terminalUiThemeData(settings.theme),
      child: Builder(
        builder: (context) {
          final palette = AnyttyPalette.of(context);
          return AnnotatedRegion<SystemUiOverlayStyle>(
            value: anyttySystemUiOverlayStyle(
              Theme.of(context).brightness,
              palette: palette,
            ),
            child: TerminalKeyboardMediaQuery(child: _buildWorkspace(context)),
          );
        },
      ),
    );
  }

  Widget _buildWorkspace(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final endpointLabel = widget.label?.trim().isNotEmpty == true
        ? widget.label!.trim()
        : widget.endpointId;
    final selectedTerminal = widget.terminalId;
    final browserVisible = _browserVisible;
    final headerTerminalId = selectedTerminal == null
        ? null
        : _activeTerminalId ?? selectedTerminal;
    final terminalItems = ref
        .watch(terminalListProvider(widget.endpointId))
        .valueOrNull;
    final connectionDiagnostics = selectedTerminal == null
        ? ref.watch(connectionDiagnosticsProvider(widget.endpointId))
        : null;
    final activeTerminal = selectedTerminal == null
        ? null
        : terminalItems
              ?.where((terminal) => terminal.ref.terminalId == headerTerminalId)
              .firstOrNull;
    final title = activeTerminal == null
        ? endpointLabel
        : _terminalDisplayName(activeTerminal);
    final listResourceTotals = selectedTerminal == null && terminalItems != null
        ? terminalResourceTotals(terminalItems)
        : null;
    final listSubtitle = listResourceTotals == null
        ? anyttyText(context, en: 'Terminals', zh: '终端')
        : anyttyText(
            context,
            en: 'Terminals · ${listResourceTotals.runningCount} running · CPU ${(listResourceTotals.cpuX100 / 100).toStringAsFixed(1)}% · RAM ${formatResourceBytes(listResourceTotals.memoryBytes)}',
            zh: '终端 · ${listResourceTotals.runningCount} 运行 · CPU ${(listResourceTotals.cpuX100 / 100).toStringAsFixed(1)}% · 内存 ${formatResourceBytes(listResourceTotals.memoryBytes)}',
          );
    final cwd = activeTerminal == null
        ? ''
        : (activeTerminal.liveCwd.isNotEmpty
              ? activeTerminal.liveCwd
              : activeTerminal.cwd);
    final terminalReady = headerTerminalId != null
        ? ref
              .watch(
                terminalConnectionProvider((
                  endpointId: widget.endpointId,
                  terminalId: headerTerminalId,
                )),
              )
              .hasValue
        : false;
    final VoidCallback? terminalSwitcherAction =
        selectedTerminal != null &&
            terminalItems != null &&
            terminalItems.isNotEmpty
        ? () => _openTerminalSwitcher(
            context: context,
            endpointId: widget.endpointId,
            endpointLabel: endpointLabel,
            terminals: terminalItems,
            activeTerminalId: headerTerminalId,
          )
        : null;
    final VoidCallback? terminalSessionsAction =
        selectedTerminal != null &&
            terminalItems != null &&
            terminalItems.isNotEmpty
        ? () => _openTerminalSwitcher(
            context: context,
            endpointId: widget.endpointId,
            endpointLabel: endpointLabel,
            terminals: terminalItems,
            activeTerminalId: headerTerminalId,
            initialTab: terminalPreviewTabSessions,
          )
        : null;
    final workspace = PopScope<Object?>(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) return;
        if (selectedTerminal != null &&
            !browserVisible &&
            _surfaceKey.currentState?.handleBack() == true) {
          return;
        }
        _handleWorkspaceBack();
      },
      child: Scaffold(
        resizeToAvoidBottomInset: selectedTerminal == null && !browserVisible,
        backgroundColor: palette.background,
        appBar: AppBar(
          toolbarHeight: 36,
          leadingWidth: 40,
          automaticallyImplyLeading: false,
          leading: IconButton(
            tooltip: anyttyText(context, en: 'Back', zh: '返回'),
            constraints: const BoxConstraints.tightFor(width: 40, height: 36),
            padding: const EdgeInsets.all(9),
            onPressed: _handleWorkspaceBack,
            icon: const Icon(Icons.arrow_back_rounded, size: 17),
          ),
          titleSpacing: 0,
          actionsIconTheme: const IconThemeData(size: 17),
          backgroundColor: selectedTerminal == null
              ? palette.background
              : palette.surface,
          foregroundColor: palette.text,
          title: Tooltip(
            message: selectedTerminal == null
                ? endpointLabel
                : 'Switch terminal: $endpointLabel · $title${cwd.isEmpty ? '' : ' · $cwd'}',
            child: Semantics(
              button: terminalSwitcherAction != null,
              label: terminalSwitcherAction == null
                  ? null
                  : 'Switch terminal, $title, $endpointLabel',
              onTap: terminalSwitcherAction,
              excludeSemantics: terminalSwitcherAction != null,
              child: GestureDetector(
                // 标题区下拉 = 打开会话页签（设计稿的顶边下拉入口）。
                onVerticalDragEnd: (details) {
                  if ((details.primaryVelocity ?? 0) > 200) {
                    terminalSessionsAction?.call();
                  }
                },
                child: InkWell(
                  onTap: terminalSwitcherAction,
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(minHeight: 34),
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.text,
                              fontSize: 12,
                              height: 1,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          Text(
                            selectedTerminal == null
                                ? listSubtitle
                                : cwd.isEmpty
                                ? endpointLabel
                                : '$endpointLabel  ·  $cwd',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 8,
                              height: 1,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
          actions: [
            FileTransferCenterAction(
              controller: ref.read(fileTransferControllerProvider),
              dimension: 38,
              iconSize: 17,
            ),
            if (selectedTerminal == null)
              IconButton(
                tooltip: anyttyText(context, en: 'Files', zh: '文件管理'),
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 36,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: () => showAnyttyFileManager(
                  context: context,
                  endpointId: widget.endpointId,
                  endpointLabel: endpointLabel,
                  initialPath: '/',
                ),
                icon: const Icon(Icons.folder_outlined, size: 17),
              ),
            if (selectedTerminal == null)
              _WorkspaceNetworkAction(
                diagnostics: connectionDiagnostics!,
                onPressed: () => _showWorkspaceConnectionDetails(
                  context: context,
                  ref: ref,
                  endpointId: widget.endpointId,
                  endpointLabel: endpointLabel,
                ),
              ),
            if (selectedTerminal == null)
              IconButton(
                tooltip: anyttyText(context, en: 'Web', zh: 'Web 浏览器'),
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 36,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: _showBrowser,
                icon: const Icon(Icons.language_rounded, size: 17),
              ),
            if (selectedTerminal == null)
              IconButton(
                tooltip: anyttyText(context, en: 'Create terminal', zh: '新建终端'),
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: () => _createTerminal(
                  context: context,
                  ref: ref,
                  endpointId: widget.endpointId,
                  label: endpointLabel,
                ),
                icon: const Icon(Icons.add_rounded),
              ),
            if (selectedTerminal != null)
              IconButton(
                tooltip: anyttyText(context, en: 'Web', zh: 'Web 浏览器'),
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: _showBrowser,
                icon: const Icon(Icons.language_rounded, size: 17),
              ),
            if (selectedTerminal != null && activeTerminal != null)
              IconButton(
                tooltip: 'Files',
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: terminalReady
                    ? () => showAnyttyFileManager(
                        context: context,
                        endpointId: widget.endpointId,
                        endpointLabel: endpointLabel,
                        initialPath: cwd,
                      )
                    : null,
                icon: const Icon(Icons.folder_outlined, size: 17),
              ),
            if (selectedTerminal != null && activeTerminal != null)
              IconButton(
                tooltip: 'Split below',
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: terminalReady
                    ? () => _surfaceKey.currentState?.splitActiveBelow()
                    : null,
                icon: const Icon(Icons.horizontal_split_rounded, size: 17),
              ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Stack(
            fit: StackFit.expand,
            children: [
              selectedTerminal == null
                  ? _TerminalList(
                      endpointId: widget.endpointId,
                      label: endpointLabel,
                    )
                  : _ActiveTerminal(
                      endpointId: widget.endpointId,
                      endpointLabel: endpointLabel,
                      terminalId: selectedTerminal,
                      keyboardVisualInset: _keyboardVisualInset,
                      keyboardInset: _keyboardInset,
                      settledKeyboardInset: _keyboardInset,
                      surfaceKey: _surfaceKey,
                      onOpenPreview: () => _openTerminalSwitcher(
                        context: context,
                        endpointId: widget.endpointId,
                        endpointLabel: endpointLabel,
                        terminals: terminalItems ?? const <TerminalInfo>[],
                        activeTerminalId: headerTerminalId,
                      ),
                      onOpenBrowser: _openBrowserUrl,
                      onActiveTerminalChanged: (terminalId) {
                        if (mounted && _activeTerminalId != terminalId) {
                          setState(() => _activeTerminalId = terminalId);
                        }
                      },
                    ),
            ],
          ),
        ),
      ),
    );
    if (!_browserOpened) return workspace;
    return Stack(
      fit: StackFit.expand,
      children: [
        workspace,
        Positioned.fill(
          child: Offstage(
            offstage: !browserVisible,
            child: TickerMode(
              enabled: browserVisible,
              child: BrowserSessionScreen(
                endpointId: widget.endpointId,
                endpointLabel: endpointLabel,
                navigationRequestId: _browserNavigationRequestId,
                navigationUrl: _browserNavigationUrl,
                onExit: () {
                  if (mounted) setState(() => _browserVisible = false);
                },
              ),
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _openTerminalSwitcher({
    required BuildContext context,
    required String endpointId,
    required String endpointLabel,
    required List<TerminalInfo> terminals,
    required String? activeTerminalId,
    String initialTab = terminalPreviewTabTerminals,
  }) async {
    final currentTerminals = await _orderSwitcherTerminals(
      endpointId,
      terminals,
    );
    if (!context.mounted) return;

    final endpoints = <TerminalSwitcherEndpoint>[];
    try {
      final registry = await ref.read(endpointRegistryProvider.future);
      final saved = registry.endpoints
          .where(
            (endpoint) => endpoint.enabled || endpoint.endpointId == endpointId,
          )
          .toList(growable: false);
      final current = saved.where(
        (endpoint) => endpoint.endpointId == endpointId,
      );
      final others = saved.where(
        (endpoint) => endpoint.endpointId != endpointId,
      );
      for (final endpoint in [...current, ...others]) {
        final label = endpoint.label.trim().isEmpty
            ? endpoint.endpointId
            : endpoint.label.trim();
        endpoints.add(
          TerminalSwitcherEndpoint(
            endpointId: endpoint.endpointId,
            label: label,
            current: endpoint.endpointId == endpointId,
            terminals: endpoint.endpointId == endpointId
                ? currentTerminals
                : null,
            activeTerminalId: endpoint.endpointId == endpointId
                ? activeTerminalId
                : null,
          ),
        );
      }
    } catch (_) {
      // The current endpoint remains switchable if registry refresh is offline.
    }
    if (!endpoints.any((endpoint) => endpoint.current)) {
      endpoints.insert(
        0,
        TerminalSwitcherEndpoint(
          endpointId: endpointId,
          label: endpointLabel,
          current: true,
          terminals: currentTerminals,
          activeTerminalId: activeTerminalId,
        ),
      );
    }
    if (!context.mounted) return;
    // 组装全屏毛玻璃预览的数据：每个会话一组终端（当前会话带 active 标记）。
    // 并行拉取其它会话的终端列表（每个 3s 上限），避免逐个等待。
    final groups = await Future.wait([
      for (final endpoint in endpoints)
        () async {
          List<TerminalInfo> items;
          if (endpoint.terminals != null) {
            items = endpoint.terminals!;
          } else {
            try {
              items = await _orderSwitcherTerminals(
                endpoint.endpointId,
                await ref
                    .read(terminalListProvider(endpoint.endpointId).future)
                    .timeout(const Duration(seconds: 3)),
              );
            } catch (_) {
              items = const <TerminalInfo>[];
            }
          }
          return TerminalPreviewGroup(
            endpointId: endpoint.endpointId,
            label: endpoint.label,
            current: endpoint.current,
            activeId: endpoint.current ? activeTerminalId : null,
            items: [
              for (final terminal in items)
                TerminalPreviewItem(
                  id: terminal.ref.terminalId,
                  label: _terminalDisplayName(terminal),
                  meta: terminalUsesHistoryOnly(terminal.state)
                      ? '已退出'
                      : (terminal.liveCwd.isNotEmpty
                            ? terminal.liveCwd
                            : terminal.cwd),
                  exited: terminalUsesHistoryOnly(terminal.state),
                ),
            ],
          );
        }(),
    ]);
    if (!context.mounted) return;
    final settings =
        ref.read(terminalSettingsProvider).valueOrNull ??
        defaultTerminalSettings;
    final selection = await showTerminalPreviewOverlay(
      context: context,
      groups: groups,
      settings: settings,
      loadScreen: _loadPreviewScreen,
      initialTab: initialTab,
    );
    if (selection == null || !context.mounted) return;
    if (selection.endpointId == endpointId) {
      _surfaceKey.currentState?.switchTerminal(selection.terminalId);
      return;
    }
    context.go(
      '/terminal/${Uri.encodeComponent(selection.endpointId)}'
      '?terminalId=${Uri.encodeQueryComponent(selection.terminalId)}'
      '&label=${Uri.encodeQueryComponent(selection.endpointLabel)}',
    );
  }

  /// 全屏预览网格的缩略图加载：连上目标会话，取一帧首屏。
  Future<CanonicalLiveScreen?> _loadPreviewScreen(
    String endpointId,
    String terminalId,
  ) async {
    try {
      final session = await ref.read(
        endpointSessionProvider(endpointId).future,
      );
      final info = await session.getTerminal(
        TerminalRef(endpointId: endpointId, terminalId: terminalId),
      );
      return await _fetchSwipeScreen(session, info);
    } catch (_) {
      return null;
    }
  }

  Future<CanonicalLiveScreen?> _fetchSwipeScreen(
    EndpointSessionClient session,
    TerminalInfo terminal,
  ) async {
    try {
      final incoming = await session
          .nextScreen(terminal.ref, 0)
          .timeout(const Duration(milliseconds: 450));
      final outcome = mergeLiveScreen(
        current: null,
        incoming: incoming,
        connectionGeneration: session.stamp.generation,
        expectedTerminal: terminal.ref,
      );
      return outcome is LiveScreenMerged ? outcome.screen : null;
    } catch (_) {
      return null;
    }
  }

  Future<List<TerminalInfo>> _orderSwitcherTerminals(
    String endpointId,
    List<TerminalInfo> terminals,
  ) async {
    List<String> pinnedIds;
    try {
      pinnedIds = await const TerminalPinStore().load(endpointId);
    } catch (_) {
      pinnedIds = const [];
    }
    final running = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.running,
      ),
      pinnedIds,
    );
    final exited = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.exited,
      ),
      pinnedIds,
    );
    final known = {...running, ...exited};
    return [
      ...running,
      ...exited,
      ...terminals.where((terminal) => !known.contains(terminal)),
    ];
  }
}
