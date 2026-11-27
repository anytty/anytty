part of 'terminal_workspace_screen.dart';

final class _TerminalList extends ConsumerStatefulWidget {
  const _TerminalList({required this.endpointId, required this.label});

  final String endpointId;
  final String label;

  @override
  ConsumerState<_TerminalList> createState() => _TerminalListState();
}

final class _TerminalListState extends ConsumerState<_TerminalList>
    with WidgetsBindingObserver {
  static const _pinStore = TerminalPinStore();
  static const _refreshInterval = Duration(seconds: 2);

  final _searchController = TextEditingController();
  final _searchFocusNode = FocusNode(debugLabel: 'terminal-list-search');
  TerminalStatusFilter _status = TerminalStatusFilter.running;
  Set<String> _selectedTagIds = const {};
  Set<String> _selectedTerminalIds = const {};
  List<String> _pinnedIds = const [];
  int _pinLoadEpoch = 0;
  Timer? _refreshTimer;
  bool _refreshInFlight = false;
  bool _searchOpen = false;
  bool _selectionMode = false;
  bool _batchBusy = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _searchController.addListener(_handleSearchChanged);
    unawaited(_loadPins());
    _scheduleRefresh();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _refreshTimer?.cancel();
    _searchController
      ..removeListener(_handleSearchChanged)
      ..dispose();
    _searchFocusNode.dispose();
    super.dispose();
  }

  void _handleSearchChanged() {
    setState(() {
      if (_selectionMode) _selectedTerminalIds = const {};
    });
  }

  void _openSearch() {
    if (_searchOpen) return;
    setState(() => _searchOpen = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _searchOpen) _searchFocusNode.requestFocus();
    });
  }

  void _closeSearch() {
    _searchFocusNode.unfocus();
    _searchController.clear();
    if (mounted) setState(() => _searchOpen = false);
  }

  void _enterSelection(String terminalId) {
    if (_selectionMode) return;
    _searchFocusNode.unfocus();
    if (mounted) {
      HapticFeedback.mediumImpact();
      setState(() {
        _selectionMode = true;
        _selectedTerminalIds = {terminalId};
      });
    }
  }

  void _closeSelection() {
    if (mounted) {
      setState(() {
        _selectionMode = false;
        _selectedTerminalIds = const {};
      });
    }
  }

  void _toggleTerminalSelection(String terminalId) {
    final next = {..._selectedTerminalIds};
    if (!next.add(terminalId)) next.remove(terminalId);
    setState(() => _selectedTerminalIds = Set.unmodifiable(next));
  }

  void _selectAllTerminals(List<TerminalInfo> terminals) {
    final ids = terminals.map((terminal) => terminal.ref.terminalId).toSet();
    setState(
      () => _selectedTerminalIds =
          ids.length == _selectedTerminalIds.length &&
              ids.every(_selectedTerminalIds.contains)
          ? const {}
          : Set.unmodifiable(ids),
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _scheduleRefresh(Duration.zero);
    } else {
      _refreshTimer?.cancel();
    }
  }

  @override
  void didUpdateWidget(_TerminalList oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId) {
      _refreshTimer?.cancel();
      _status = TerminalStatusFilter.running;
      _searchController.clear();
      _selectedTagIds = const {};
      _selectedTerminalIds = const {};
      _selectionMode = false;
      _searchOpen = false;
      _pinnedIds = const [];
      unawaited(_loadPins());
      _scheduleRefresh(Duration.zero);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final terminals = ref.watch(terminalListProvider(widget.endpointId));
    final content = terminals.when(
      loading: () => _TerminalListLoading(
        endpointId: widget.endpointId,
        label: widget.label,
      ),
      error: (error, _) => _TerminalFailure(
        message: error.toString(),
        collapseDetails: true,
        onRetry: () => ref.invalidate(terminalListProvider(widget.endpointId)),
      ),
      data: (items) {
        if (items.isEmpty) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.terminal_rounded, size: 36, color: palette.muted),
                const SizedBox(height: 12),
                Text(
                  anyttyText(context, en: 'No terminals', zh: '暂无终端'),
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          );
        }
        final tags = terminalTagOptions(items);
        final filtered = sortPinnedTerminals(
          filterTerminals(
            terminals: items,
            status: _status,
            tagIds: _selectedTagIds,
            query: _searchController.text,
          ),
          _pinnedIds,
        );
        final runningCount = filterTerminals(
          terminals: items,
          status: TerminalStatusFilter.running,
        ).length;
        final exitedCount = filterTerminals(
          terminals: items,
          status: TerminalStatusFilter.exited,
        ).length;
        return Column(
          children: [
            if (_selectionMode)
              _TerminalSelectionBar(
                selectedCount: _selectedTerminalIds.length,
                totalCount: filtered.length,
                canEnd: filtered.any(
                  (terminal) =>
                      _selectedTerminalIds.contains(terminal.ref.terminalId) &&
                      _terminalCanEnd(terminal),
                ),
                canDelete: filtered.any(
                  (terminal) =>
                      _selectedTerminalIds.contains(terminal.ref.terminalId) &&
                      _terminalCanDelete(terminal),
                ),
                busy: _batchBusy,
                onSelectAll: () => _selectAllTerminals(filtered),
                onEnd: () =>
                    unawaited(_runBulkAction(items, _TerminalAction.end)),
                onDelete: () =>
                    unawaited(_runBulkAction(items, _TerminalAction.remove)),
                onClose: _closeSelection,
              ),
            if (_searchOpen)
              _TerminalSearchField(
                controller: _searchController,
                focusNode: _searchFocusNode,
                onClose: _closeSearch,
              )
            else
              _TerminalFilterBar(
                status: _status,
                runningCount: runningCount,
                exitedCount: exitedCount,
                totalCount: items.length,
                selectedTagCount: _selectedTagIds.length,
                tagsAvailable: tags.isNotEmpty,
                onStatusChanged: (value) => setState(() {
                  _status = value;
                  _selectedTerminalIds = const {};
                }),
                onTags: () => _showTagFilters(tags, items),
                onSearch: _openSearch,
              ),
            Expanded(
              child: RefreshIndicator(
                onRefresh: () => _refreshInventory(force: true),
                child: filtered.isEmpty
                    ? ListView(
                        physics: const AlwaysScrollableScrollPhysics(),
                        children: [
                          const SizedBox(height: 120),
                          Icon(
                            Icons.filter_alt_off_rounded,
                            color: palette.muted,
                            size: 32,
                          ),
                          const SizedBox(height: 10),
                          Center(
                            child: Text(
                              anyttyText(
                                context,
                                en: 'No matching terminals',
                                zh: '没有符合条件的终端',
                              ),
                            ),
                          ),
                        ],
                      )
                    : LayoutBuilder(
                        builder: (context, constraints) {
                          const horizontalPadding = 12.0;
                          const gap = 8.0;
                          final columns = constraints.maxWidth >= 720 ? 2 : 1;
                          final contentWidth =
                              constraints.maxWidth - horizontalPadding * 2;
                          final itemWidth =
                              (contentWidth - gap * (columns - 1)) / columns;
                          return ListView(
                            physics: const AlwaysScrollableScrollPhysics(),
                            padding: const EdgeInsets.fromLTRB(
                              horizontalPadding,
                              8,
                              horizontalPadding,
                              32,
                            ),
                            children: [
                              Wrap(
                                spacing: gap,
                                runSpacing: gap,
                                children: [
                                  for (final terminal in filtered)
                                    SizedBox(
                                      width: itemWidth,
                                      child: _buildTerminalRow(terminal),
                                    ),
                                ],
                              ),
                            ],
                          );
                        },
                      ),
              ),
            ),
          ],
        );
      },
    );
    return PopScope<Object?>(
      canPop: !_searchOpen && !_selectionMode,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) return;
        if (_searchOpen) {
          _closeSearch();
        } else if (_selectionMode) {
          _closeSelection();
        }
      },
      child: content,
    );
  }

  Widget _buildTerminalRow(TerminalInfo terminal) {
    final terminalId = terminal.ref.terminalId;
    final pinIndex = _pinnedIds.indexOf(terminalId);
    return _TerminalRow(
      terminal: terminal,
      pinned: pinIndex >= 0,
      canMoveUp: pinIndex > 0,
      canMoveDown: pinIndex >= 0 && pinIndex < _pinnedIds.length - 1,
      searchQuery: _searchController.text,
      selected: _selectedTerminalIds.contains(terminalId),
      selectionMode: _selectionMode,
      onTap: () => _openTerminal(terminalId),
      onLongPress: () => _enterSelection(terminalId),
      onToggleSelected: () => _toggleTerminalSelection(terminalId),
      onAction: (action) => _handleRowAction(terminal, action),
    );
  }

  Future<void> _runBulkAction(
    List<TerminalInfo> terminals,
    _TerminalAction action,
  ) async {
    final selected = terminals
        .where(
          (terminal) => _selectedTerminalIds.contains(terminal.ref.terminalId),
        )
        .where(
          action == _TerminalAction.end ? _terminalCanEnd : _terminalCanDelete,
        )
        .toList(growable: false);
    if (selected.isEmpty || _batchBusy) return;
    final destructive = action == _TerminalAction.remove;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(
          destructive
              ? anyttyText(
                  context,
                  en: 'Delete terminal records?',
                  zh: '删除终端记录？',
                )
              : anyttyText(context, en: 'End processes?', zh: '结束进程？'),
        ),
        content: Text(
          destructive
              ? anyttyText(
                  context,
                  en: 'Delete ${selected.length} exited terminal records?',
                  zh: '删除 ${selected.length} 个已退出的终端记录？',
                )
              : anyttyText(
                  context,
                  en: 'End ${selected.length} terminal processes? Their records will remain.',
                  zh: '结束 ${selected.length} 个终端进程？终端记录会保留。',
                ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
          ),
          FilledButton(
            style: destructive
                ? FilledButton.styleFrom(
                    backgroundColor: const Color(0xffdc2626),
                  )
                : null,
            onPressed: () => Navigator.pop(context, true),
            child: Text(
              destructive
                  ? anyttyText(context, en: 'Delete', zh: '删除')
                  : anyttyText(context, en: 'End', zh: '结束'),
            ),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() => _batchBusy = true);
    final failures = <String>[];
    try {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      for (final terminal in selected) {
        try {
          if (action == _TerminalAction.end) {
            await session.killTerminal(terminal.ref);
          } else {
            await session.removeTerminal(terminal.ref);
          }
        } catch (_) {
          failures.add(terminal.ref.terminalId);
        }
      }
      ref.invalidate(terminalListProvider(widget.endpointId));
      if (!mounted) return;
      setState(() {
        _batchBusy = false;
        if (failures.isEmpty) {
          _selectionMode = false;
          _selectedTerminalIds = const {};
        } else {
          _selectedTerminalIds = Set.unmodifiable(failures.toSet());
        }
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            failures.isEmpty
                ? (destructive
                      ? anyttyText(
                          context,
                          en: 'Terminal records deleted',
                          zh: '终端记录已删除',
                        )
                      : anyttyText(
                          context,
                          en: 'Terminal processes ended',
                          zh: '终端进程已结束',
                        ))
                : anyttyText(
                    context,
                    en: '${selected.length - failures.length} completed, ${failures.length} failed',
                    zh: '已完成 ${selected.length - failures.length} 个，${failures.length} 个失败',
                  ),
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() => _batchBusy = false);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  void _scheduleRefresh([Duration delay = _refreshInterval]) {
    _refreshTimer?.cancel();
    _refreshTimer = Timer(delay, _refreshInventory);
  }

  Future<void> _refreshInventory({bool force = false}) async {
    if (!mounted || _refreshInFlight) return;
    final appActive =
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    final routeActive = ModalRoute.of(context)?.isCurrent ?? false;
    if (!force && (!appActive || !routeActive)) {
      _scheduleRefresh();
      return;
    }
    _refreshInFlight = true;
    try {
      await ref
          .refresh(terminalListProvider(widget.endpointId).future)
          .then<void>((_) {});
    } catch (_) {
      // The current snapshot stays visible and the next scheduled poll retries.
    } finally {
      _refreshInFlight = false;
      if (mounted) _scheduleRefresh();
    }
  }

  Future<void> _loadPins() async {
    final epoch = ++_pinLoadEpoch;
    try {
      final loaded = await _pinStore.load(widget.endpointId);
      if (!mounted || epoch != _pinLoadEpoch) return;
      setState(() => _pinnedIds = loaded);
    } catch (_) {
      // Pinning is optional local presentation state.
    }
  }

  Future<void> _savePins(List<String> next) async {
    final previous = _pinnedIds;
    setState(() => _pinnedIds = next);
    try {
      final saved = await _pinStore.save(widget.endpointId, next);
      if (mounted) setState(() => _pinnedIds = saved);
    } catch (error) {
      if (!mounted) return;
      setState(() => _pinnedIds = previous);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  void _openTerminal(String terminalId) {
    context.go(
      '/terminal/${Uri.encodeComponent(widget.endpointId)}'
      '?terminalId=${Uri.encodeQueryComponent(terminalId)}'
      '&label=${Uri.encodeQueryComponent(widget.label)}',
    );
  }

  void _handleRowAction(TerminalInfo terminal, _TerminalRowAction action) {
    final terminalId = terminal.ref.terminalId;
    switch (action) {
      case _TerminalRowAction.pin:
      case _TerminalRowAction.unpin:
        unawaited(_savePins(toggleTerminalPin(_pinnedIds, terminalId)));
      case _TerminalRowAction.moveUp:
        unawaited(_savePins(movePinnedTerminal(_pinnedIds, terminalId, -1)));
      case _TerminalRowAction.moveDown:
        unawaited(_savePins(movePinnedTerminal(_pinnedIds, terminalId, 1)));
      case _TerminalRowAction.rename:
      case _TerminalRowAction.restart:
      case _TerminalRowAction.end:
      case _TerminalRowAction.remove:
        final remoteAction = switch (action) {
          _TerminalRowAction.rename => _TerminalAction.rename,
          _TerminalRowAction.restart => _TerminalAction.restart,
          _TerminalRowAction.end => _TerminalAction.end,
          _TerminalRowAction.remove => _TerminalAction.remove,
          _ => throw StateError('Local terminal action was not handled'),
        };
        unawaited(
          _manageTerminal(
            context: context,
            ref: ref,
            endpointId: widget.endpointId,
            terminal: terminal,
            action: remoteAction,
          ),
        );
    }
  }

  Future<void> _showTagFilters(
    List<TerminalTagOption> options,
    List<TerminalInfo> terminals,
  ) async {
    final palette = AnyttyPalette.of(context);
    await showModalBottomSheet<void>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      showDragHandle: false,
      backgroundColor: palette.surface,
      barrierColor: palette.overlay,
      shape: RoundedRectangleBorder(
        borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
        side: BorderSide(color: palette.border),
      ),
      builder: (context) => _TerminalTagFilterSheet(
        options: options,
        selected: _selectedTagIds,
        totalCount: terminals.length,
        resultCount: (selected) => filterTerminals(
          terminals: terminals,
          status: _status,
          tagIds: selected,
          query: _searchController.text,
        ).length,
        onChanged: (selected) {
          if (mounted) {
            setState(() {
              _selectedTagIds = selected;
              _selectedTerminalIds = const {};
            });
          }
        },
      ),
    );
  }
}

final class _TerminalListLoading extends ConsumerStatefulWidget {
  const _TerminalListLoading({required this.endpointId, required this.label});

  final String endpointId;
  final String label;

  @override
  ConsumerState<_TerminalListLoading> createState() =>
      _TerminalListLoadingState();
}

final class _TerminalListLoadingState
    extends ConsumerState<_TerminalListLoading> {
  static const _progressRebuildInterval = Duration(milliseconds: 32);

  final ConnectionAttemptHistory _history = ConnectionAttemptHistory();
  bool _historyExpanded = false;
  ProviderSubscription<AsyncValue<EndpointConnectionEvent>>?
  _progressSubscription;
  AsyncValue<EndpointConnectionEvent> _progress = const AsyncLoading();
  Timer? _progressRebuildTimer;
  bool _applyingAuto = false;
  String? _actionError;

  @override
  void initState() {
    super.initState();
    _bindConnectionProgress();
  }

  @override
  void dispose() {
    _progressRebuildTimer?.cancel();
    _progressSubscription?.close();
    super.dispose();
  }

  @override
  void didUpdateWidget(_TerminalListLoading oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId) {
      _history.reset();
      _historyExpanded = false;
      _applyingAuto = false;
      _actionError = null;
      _bindConnectionProgress();
    }
  }

  void _bindConnectionProgress() {
    _progressSubscription?.close();
    final provider = endpointConnectionProgressProvider(widget.endpointId);
    _progress = ref.read(provider);
    _progressSubscription = ref
        .listenManual<AsyncValue<EndpointConnectionEvent>>(provider, (_, next) {
          if (!mounted) return;
          _progress = next;
          _captureProgress(next.valueOrNull);
          _scheduleProgressRebuild();
        });
  }

  void _scheduleProgressRebuild() {
    if (_progressRebuildTimer != null) return;
    _progressRebuildTimer = Timer(_progressRebuildInterval, () {
      _progressRebuildTimer = null;
      if (mounted) setState(() {});
    });
  }

  void _captureProgress(EndpointConnectionEvent? event) {
    if (!mounted || event == null) return;
    // 新一轮开始时归档上一轮，而不是清空：重试之后仍然能看到上次失败原因。
    _history.record(event);
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final progress = _progress;
    final policyState = ref
        .watch(connectionPolicyProvider(widget.endpointId))
        .valueOrNull;
    final directOnly =
        policyState?.policy.routePreference ==
        EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT;
    final phase =
        progress.valueOrNull?.phase ??
        EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING;
    final waitingForNetwork =
        phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_OFFLINE;
    final attempts = Map<ConnectionRouteKind, EndpointConnectionEvent>.of(
      _history.current,
    );
    final latest = progress.valueOrNull;
    if (latest != null &&
        !isSupersededConnectionAttempt(latest) &&
        latest.attemptedRouteKind !=
            ConnectionRouteKind.CONNECTION_ROUTE_KIND_UNSPECIFIED) {
      attempts[latest.attemptedRouteKind] = latest;
    }
    final attemptEvents = _orderedConnectionAttempts(attempts);
    final selectedKind = switch (policyState?.policy.routePreference) {
      EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT =>
        ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
      EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_SSH =>
        ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH,
      EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD =>
        ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
      _ => null,
    };
    // A losing AUTO route is not an overall failure. The runtime emits OFFLINE
    // when the planner has exhausted its routes, including routes not yet started.
    final selectedFailed =
        selectedKind != null &&
        attempts[selectedKind]?.connectionStage == 'attempt_failed';
    final connectionFailed =
        phase != EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_READY &&
        (waitingForNetwork || progress.hasError || selectedFailed);
    final errors = connectionFailed
        ? [
            for (final event in attemptEvents)
              if (event.hasError() &&
                  (selectedKind == null ||
                      event.attemptedRouteKind == selectedKind))
                _connectionAttemptLabel(context, event, includeError: true),
            if (latest != null &&
                latest.hasError() &&
                latest.attemptedRouteKind ==
                    ConnectionRouteKind.CONNECTION_ROUTE_KIND_UNSPECIFIED)
              latest.error.message,
            if (progress.hasError) progress.error.toString(),
          ]
        : <String>[];
    return LayoutBuilder(
      builder: (context, constraints) => Padding(
        // Anchor the mascot to the viewport, never to the changing route list.
        padding: EdgeInsets.fromLTRB(
          24,
          (constraints.maxHeight * 0.22).clamp(16.0, 160.0),
          24,
          16,
        ),
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 360),
            child: Column(
              children: [
                ExcludeSemantics(
                  child: AnyttyBrandLoader(
                    height: 96,
                    scene: connectionFailed
                        ? AnyttyMascotScene.failure
                        : AnyttyMascotScene.connecting,
                  ),
                ),
                Expanded(
                  child: SingleChildScrollView(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const SizedBox(height: 16),
                        Text(
                          widget.label,
                          textAlign: TextAlign.center,
                          style: TextStyle(
                            color: palette.text,
                            fontSize: 18,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 6),
                        Semantics(
                          liveRegion: true,
                          child: Text(
                            connectionFailed
                                ? anyttyText(
                                    context,
                                    en: 'Connection unavailable',
                                    zh: '暂时无法连接',
                                  )
                                : _connectionLoadingLabel(context, phase),
                            textAlign: TextAlign.center,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 13,
                            ),
                          ),
                        ),
                        const SizedBox(height: 20),
                        Text(
                          _connectionModeLabel(
                            context,
                            policyState?.policy.routePreference,
                          ),
                          textAlign: TextAlign.center,
                          style: TextStyle(color: palette.muted, fontSize: 11),
                        ),
                        if (attemptEvents.isNotEmpty) ...[
                          const SizedBox(height: 12),
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 14,
                              vertical: 8,
                            ),
                            decoration: BoxDecoration(
                              color: palette.surface,
                              borderRadius: BorderRadius.circular(16),
                              border: Border.all(color: palette.border),
                            ),
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                for (final event in attemptEvents)
                                  Padding(
                                    padding: const EdgeInsets.symmetric(
                                      vertical: 7,
                                    ),
                                    child: Row(
                                      children: [
                                        _ConnectionRouteDot(
                                          key: ValueKey(
                                            event.attemptedRouteKind,
                                          ),
                                          active:
                                              !connectionFailed &&
                                              event.connectionStage !=
                                                  'attempt_failed' &&
                                              event.phase !=
                                                  EndpointConnectionPhase
                                                      .ENDPOINT_CONNECTION_PHASE_READY,
                                          color:
                                              event.connectionStage ==
                                                  'attempt_failed'
                                              ? palette.faint
                                              : palette.accent,
                                        ),
                                        const SizedBox(width: 12),
                                        Expanded(
                                          child: Text(
                                            _connectionAttemptLabel(
                                              context,
                                              event,
                                            ),
                                            style: TextStyle(
                                              color: palette.muted,
                                              fontSize: 12,
                                              height: 1.5,
                                            ),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                              ],
                            ),
                          ),
                        ],
                        if (_history.archives.isNotEmpty) ...[
                          const SizedBox(height: 10),
                          _ConnectionAttemptHistoryPanel(
                            snapshots: _history.archives,
                            expanded: _historyExpanded,
                            onToggle: () => setState(
                              () => _historyExpanded = !_historyExpanded,
                            ),
                          ),
                        ],
                        if (connectionFailed && directOnly) ...[
                          const SizedBox(height: 20),
                          _DirectOnlyConnectionHelp(
                            applying: _applyingAuto,
                            error: _actionError,
                            onUseAuto: policyState == null
                                ? null
                                : () => _useAutomaticConnection(policyState),
                            onOpenSettings: _openConnectionSettings,
                          ),
                        ],
                        if (errors.isNotEmpty) ...[
                          const SizedBox(height: 12),
                          _ConnectionErrorDetails(message: errors.join('\n\n')),
                        ],
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _useAutomaticConnection(ConnectionPolicyState state) async {
    if (_applyingAuto) return;
    setState(() {
      _applyingAuto = true;
      _actionError = null;
    });
    var policySaved = false;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final policy = state.policy.deepCopy()
        ..routePreference =
            EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO;
      await ConnectionRepository(runtime)
          .applyPolicy(widget.endpointId, policy);
      policySaved = true;
      ref.invalidate(connectionPolicyProvider(widget.endpointId));
      await EndpointRepository(runtime).disconnectEndpoint(widget.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpointId));
      ref.invalidate(connectionDiagnosticsProvider(widget.endpointId));
      ref.invalidate(terminalListProvider(widget.endpointId));
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _actionError = policySaved
            ? anyttyText(
                context,
                en: 'Automatic mode was saved, but reconnecting failed. Try again.',
                zh: '已改为自动连接，但重新连接失败，请重试。',
              )
            : anyttyText(
                context,
                en: 'Could not change the connection mode. Try again.',
                zh: '无法更改连接模式，请重试。',
              );
      });
    } finally {
      if (mounted) setState(() => _applyingAuto = false);
    }
  }

  void _openConnectionSettings() {
    context.push(
      '/connection/${Uri.encodeComponent(widget.endpointId)}'
      '?label=${Uri.encodeQueryComponent(widget.label)}',
    );
  }
}
