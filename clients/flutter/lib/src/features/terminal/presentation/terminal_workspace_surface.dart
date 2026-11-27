part of 'terminal_workspace_screen.dart';

final class _TerminalSurface extends StatefulWidget {
  const _TerminalSurface({
    super.key,
    required this.connection,
    required this.keyboardMode,
    required this.keyboardVisualInset,
    required this.keyboardInset,
    required this.settledKeyboardInset,
    required this.settings,
    required this.onOpenPreview,
    required this.active,
    required this.onActivate,
    required this.externalControls,
    required this.manageKeyboardInset,
    required this.onSurfaceReady,
    required this.onInput,
    required this.onOpenBrowser,
    required this.onOpenFiles,
    required this.onShowResources,
    required this.splitOpen,
    required this.syncInput,
    required this.canSplit,
    required this.onSplit,
    required this.onToggleSync,
    required this.onCloseSplit,
    required this.onSettingsChanged,
    required this.onKeyboardModeChanged,
    required this.onReconnect,
  });

  final TerminalConnection connection;
  final TerminalKeyboardMode keyboardMode;
  final ValueListenable<double> keyboardVisualInset;
  final double keyboardInset;
  final double settledKeyboardInset;
  final TerminalSettings settings;
  final VoidCallback? onOpenPreview;
  final bool active;
  final VoidCallback onActivate;
  final bool externalControls;
  final bool manageKeyboardInset;
  final VoidCallback onSurfaceReady;
  final Future<void> Function(_TerminalInputOperation) onInput;
  final Future<void> Function(Uri) onOpenBrowser;
  final Future<void> Function() onOpenFiles;
  final Future<void> Function() onShowResources;
  final bool splitOpen;
  final bool syncInput;
  final bool canSplit;
  final ValueChanged<_TerminalSplitTarget> onSplit;
  final VoidCallback onToggleSync;
  final VoidCallback onCloseSplit;
  final Future<void> Function(TerminalSettings) onSettingsChanged;
  final Future<void> Function(TerminalKeyboardMode) onKeyboardModeChanged;
  final VoidCallback onReconnect;

  @override
  State<_TerminalSurface> createState() => _TerminalSurfaceState();
}

final class _TerminalSurfaceState extends State<_TerminalSurface> {
  final ValueNotifier<int> controlRevision = ValueNotifier(0);
  final FocusNode _inputFocus = FocusNode(debugLabel: 'terminal-input');
  final FocusNode _searchFocus = FocusNode(debugLabel: 'terminal-search');
  final TextEditingController _inputController = TextEditingController();
  final TextEditingController _searchController = TextEditingController();
  final _historyScroll = TerminalHistoryScrollController();
  final TerminalHistoryHighlights _historyHighlights =
      TerminalHistoryHighlights();
  TerminalModifierState _modifiers = const TerminalModifierState();
  bool _clearingInput = false;
  int _keyboardRequestEpoch = 0;
  bool _fnOpen = false;
  bool _keyboardFocusLocked = false;
  bool _toolsOpen = false;
  late TerminalKeyboardMode _keyboardMode;
  bool _historyLoading = false;
  bool _historyLoadingVisible = false;
  bool _historyPositioned = false;
  bool _historyPresented = false;
  bool _selectionMode = false;
  bool _searchOpen = false;
  bool _searching = false;
  bool _searchScanActive = false;
  bool _copying = false;
  int _searchEpoch = 0;
  int _searchMatchCount = 0;
  FrozenHistory? _history;
  FrozenHistory? _historyLayoutSource;
  HistoryLayout? _historyLayoutCache;
  HistorySelection? _selection;
  HistoryRange? _searchMatch;
  List<HistoryRange> _searchMatches = const [];
  HistorySearchMode _searchMode = HistorySearchMode.HISTORY_SEARCH_MODE_TEXT;
  bool _searchWrapped = false;
  String? _historyError;
  bool _historyRequiresReload = false;
  late ResizeControl _resizeControl;
  StreamSubscription<ResizeControl>? _resizeSubscription;
  late TerminalState _terminalState;
  StreamSubscription<TerminalState>? _terminalStateSubscription;
  late TerminalDeliveryState _deliveryState;
  StreamSubscription<TerminalDeliveryState>? _deliveryStateSubscription;
  late final TerminalRecoveryNoticeGate _recoveryNoticeGate;
  bool _recoveryNoticeVisible = false;
  ({int cols, int rows})? _viewportCells;
  ({int cols, int rows})? _historyViewportCells;
  int? _historyProjectionFailedCols;
  double? _fullTerminalViewportHeight;
  bool _resizePending = false;
  double _liveDragDistance = 0;
  Offset _lastLiveDragPosition = Offset.zero;
  Timer? _momentumTimer;
  Timer? _historyLoadingDelay;
  int _historyRequestEpoch = 0;
  DateTime? _historyScrollSampleTime;
  double? _historyScrollSampleOffset;
  double _historyUpwardVelocityRows = 0;
  double _momentumVelocity = 0;
  int _momentumTicks = 0;
  bool _autoAcquireAttempted = false;
  late final String _androidInputOwner;
  StreamSubscription<AndroidTerminalInputEvent>? _androidInputSubscription;
  final ValueNotifier<({bool active, String text})> _androidComposition =
      ValueNotifier((active: false, text: ''));

  bool get _readOnly =>
      widget.connection.historyOnly || terminalUsesHistoryOnly(_terminalState);

  void _notifyControls() => controlRevision.value += 1;

  HistoryLayout _historyLayoutFor(FrozenHistory history) {
    if (identical(_historyLayoutSource, history) &&
        _historyLayoutCache != null) {
      return _historyLayoutCache!;
    }
    final layout = HistoryLayout.fromHistory(history);
    _historyLayoutSource = history;
    _historyLayoutCache = layout;
    return layout;
  }

  @override
  void initState() {
    super.initState();
    _keyboardMode = widget.keyboardMode;
    _androidInputOwner = 'terminal-input-${identityHashCode(this)}';
    if (defaultTargetPlatform == TargetPlatform.android) {
      _androidInputSubscription = AndroidTerminalInputPlatform.instance
          .eventsFor(_androidInputOwner)
          .listen(_handleAndroidTerminalInput);
    }
    _resetSoftKeyboardBuffer();
    _inputController.addListener(_handleTextInput);
    _inputFocus.addListener(_handleInputFocusChange);
    _historyScroll.addListener(_handleHistoryScroll);
    _recoveryNoticeGate = TerminalRecoveryNoticeGate(
      onVisibilityChanged: (visible) {
        if (!mounted || _recoveryNoticeVisible == visible) return;
        setState(() => _recoveryNoticeVisible = visible);
      },
    );
    widget.connection.updateInputMetrics(widget.settings.metrics);
    _bindResizeControl();
    _bindTerminalState();
    _bindDeliveryState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) widget.onSurfaceReady();
    });
  }

  @override
  void didUpdateWidget(_TerminalSurface oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.connection != widget.connection) {
      _clearAndroidComposition();
      _historyRequestEpoch += 1;
      _historyLoadingDelay?.cancel();
      if (_history case final history?) {
        _releaseHistory(oldWidget.connection, history);
      }
      _modifiers = const TerminalModifierState();
      _fnOpen = false;
      _toolsOpen = false;
      _history = null;
      _historyLoading = false;
      _historyLoadingVisible = false;
      _historyPositioned = false;
      _historyPresented = false;
      _resetHistoryInteraction();
      unawaited(_resizeSubscription?.cancel());
      unawaited(_terminalStateSubscription?.cancel());
      unawaited(_deliveryStateSubscription?.cancel());
      _recoveryNoticeGate.setRecovering(false);
      _viewportCells = null;
      _historyViewportCells = null;
      _historyProjectionFailedCols = null;
      _fullTerminalViewportHeight = null;
      _autoAcquireAttempted = false;
      widget.connection.updateInputMetrics(widget.settings.metrics);
      _bindResizeControl();
      _bindTerminalState();
      _bindDeliveryState();
    } else if (oldWidget.settings.metrics != widget.settings.metrics) {
      final firstHistoryRow = _historyScroll.hasClients
          ? _historyScroll.offset / oldWidget.settings.metrics.rowHeight
          : null;
      _viewportCells = null;
      _historyViewportCells = null;
      widget.connection.updateInputMetrics(widget.settings.metrics);
      if (firstHistoryRow != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (!mounted || !_historyScroll.hasClients) return;
          _historyScroll.jumpTo(
            (firstHistoryRow * widget.settings.metrics.rowHeight).clamp(
              0.0,
              _historyScroll.position.maxScrollExtent,
            ),
          );
        });
      }
    }
    if (oldWidget.keyboardMode != widget.keyboardMode) {
      _keyboardMode = widget.keyboardMode;
      _viewportCells = null;
    }
    if (!oldWidget.settings.autoAcquireResizeOwner &&
        widget.settings.autoAcquireResizeOwner) {
      _autoAcquireAttempted = false;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) unawaited(_autoAcquireResizeOwnership());
      });
    }
    if (oldWidget.settings.scrollInertia != widget.settings.scrollInertia) {
      _cancelMomentum();
    }
    if (oldWidget.active && !widget.active) {
      if (defaultTargetPlatform == TargetPlatform.android) {
        _clearAndroidComposition();
        unawaited(
          AndroidTerminalInputPlatform.instance.release(_androidInputOwner),
        );
      } else {
        _inputFocus.unfocus();
      }
    }
    if (oldWidget.active != widget.active ||
        oldWidget.splitOpen != widget.splitOpen ||
        oldWidget.syncInput != widget.syncInput ||
        oldWidget.canSplit != widget.canSplit) {
      _notifyControls();
    }
    // 重新进入这个终端时再试一次拿尺寸：另一个客户端（桌面）可能在这期间
    // 接走了 resize lease，只试过一次的话这里会一直停在 follower 的尺寸上。
    if (!oldWidget.active &&
        widget.active &&
        widget.settings.autoAcquireResizeOwner &&
        !_resizeControl.canResize &&
        _viewportCells != null) {
      _autoAcquireAttempted = false;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) unawaited(_autoAcquireResizeOwnership());
      });
    }
  }

  @override
  void dispose() {
    unawaited(_androidInputSubscription?.cancel());
    if (defaultTargetPlatform == TargetPlatform.android) {
      unawaited(
        AndroidTerminalInputPlatform.instance.release(_androidInputOwner),
      );
    }
    _inputController
      ..removeListener(_handleTextInput)
      ..dispose();
    _inputFocus.removeListener(_handleInputFocusChange);
    _inputFocus.dispose();
    _searchFocus.dispose();
    _searchController.dispose();
    _historyScroll.dispose();
    _historyHighlights.dispose();
    _androidComposition.dispose();
    _cancelMomentum();
    _historyLoadingDelay?.cancel();
    _recoveryNoticeGate.dispose();
    _historyRequestEpoch += 1;
    unawaited(_resizeSubscription?.cancel());
    unawaited(_terminalStateSubscription?.cancel());
    unawaited(_deliveryStateSubscription?.cancel());
    if (_history case final history?) {
      _releaseHistory(widget.connection, history);
    }
    controlRevision.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, outerConstraints) {
        final visibleHeight = widget.manageKeyboardInset
            ? math.max(0.0, outerConstraints.maxHeight - widget.keyboardInset)
            : outerConstraints.maxHeight;
        return Align(
          alignment: Alignment.topCenter,
          child: SizedBox(
            height: visibleHeight,
            child: Column(
              children: [
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      _scheduleHistoryProjectionFit(constraints.biggest);
                      return Focus(
                        canRequestFocus: widget.active,
                        onKeyEvent: _handleHardwareKey,
                        child: Stack(
                          fit: StackFit.expand,
                          children: [
                            _buildTerminalContent(),
                            if (defaultTargetPlatform != TargetPlatform.android)
                              Positioned(
                                left: 0,
                                bottom: 0,
                                width: 1,
                                height: 1,
                                child: ExcludeSemantics(
                                  child: ClipRect(
                                    child: EditableText(
                                      controller: _inputController,
                                      focusNode: _inputFocus,
                                      style: const TextStyle(
                                        color: Colors.transparent,
                                        fontSize: 1,
                                      ),
                                      cursorColor: Colors.transparent,
                                      backgroundCursorColor: Colors.transparent,
                                      keyboardType: TextInputType.multiline,
                                      textInputAction: TextInputAction.newline,
                                      autocorrect: false,
                                      enableSuggestions: false,
                                      smartDashesType: SmartDashesType.disabled,
                                      smartQuotesType: SmartQuotesType.disabled,
                                      onEditingComplete: () {},
                                      onSubmitted: (_) => _sendEnter(),
                                      maxLines: null,
                                    ),
                                  ),
                                ),
                              ),
                            if (defaultTargetPlatform == TargetPlatform.android)
                              Positioned.fill(
                                child: IgnorePointer(
                                  child:
                                      ValueListenableBuilder<
                                        ({bool active, String text})
                                      >(
                                        valueListenable: _androidComposition,
                                        builder: (context, composition, _) =>
                                            composition.active
                                            ? _TerminalCompositionOverlay(
                                                text: composition.text,
                                              )
                                            : const SizedBox.shrink(),
                                      ),
                                ),
                              ),
                            if (!widget.externalControls)
                              ...buildHeaderOverlays(),
                            if (!widget.externalControls)
                              ...buildExternalOverlays(),
                          ],
                        ),
                      );
                    },
                  ),
                ),
                if (!widget.externalControls) buildExternalKeyBar(),
              ],
            ),
          ),
        );
      },
    );
  }

  List<Widget> buildExternalOverlays() {
    return [
      if (_toolsOpen)
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: toggleTools,
          ),
        ),
      if (_toolsOpen)
        Positioned(
          left: 8,
          right: 8,
          bottom: 8,
          child: _TerminalToolsSheet(
            terminalState: _terminalState,
            terminalSize: widget.connection.terminalSize,
            resizeControl: _resizeControl,
            inputEnabled:
                !_readOnly &&
                _history == null &&
                !_historyLoading &&
                !_searchOpen &&
                _deliveryState == TerminalDeliveryState.ready,
            historyActive: _history != null || _historyLoading,
            historyActionEnabled: !_readOnly || _history == null,
            keyboardMode: _keyboardMode,
            searchActive: _searchOpen,
            resizeAvailable: !_readOnly,
            splitOpen: widget.splitOpen,
            syncInput: widget.syncInput,
            canSplit: widget.canSplit,
            onSplit: _requestSplit,
            onToggleSync: widget.onToggleSync,
            onCloseSplit: _requestCloseSplit,
            onKeyboardModeChanged: (mode) =>
                unawaited(_changeKeyboardMode(mode)),
            onClose: toggleTools,
            onSelected: _handleTerminalToolAction,
          ),
        ),
      if (_searchOpen)
        Positioned(
          left: 8,
          right: 8,
          bottom: 8,
          child: _HistorySearchBar(
            controller: _searchController,
            focusNode: _searchFocus,
            mode: _searchMode,
            searching: _searching,
            scanning: _searchScanActive,
            matchCount: _searchMatchCount,
            wrapped: _searchWrapped,
            onQueryChanged: _handleSearchQueryChanged,
            onModeChanged: _changeSearchMode,
            onPrevious: () => _navigateSearch(
              HistorySearchDirection.HISTORY_SEARCH_DIRECTION_BACKWARD,
            ),
            onNext: () => _navigateSearch(
              HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD,
            ),
            onClose: _closeSearch,
          ),
        ),
      if (_selectionMode)
        Positioned(
          left: 8,
          right: 8,
          bottom: 8,
          child: _TerminalSelectionToolbar(
            selectionAvailable: _selection != null,
            copyPending: _copying,
            onSelectAll: _selectAllLoaded,
            onSelectVisible: _selectVisible,
            onCopy: _copySelection,
            onClose: _closeSelection,
          ),
        ),
    ];
  }

  List<Widget> buildHeaderOverlays() {
    return [
      if (_fnOpen)
        TerminalHeaderQuickKeysLayer(
          child: Consumer(
            builder: (context, ref, _) => TerminalQuickKeysPanel(
              actions:
                  ref.watch(terminalQuickActionsProvider).valueOrNull ??
                  defaultTerminalQuickActions,
              inputEnabled:
                  !_readOnly &&
                  _history == null &&
                  !_historyLoading &&
                  _deliveryState == TerminalDeliveryState.ready,
              onClose: _toggleFn,
              onAction: _runQuickAction,
              attachedToHeader: true,
            ),
          ),
        ),
    ];
  }

  Widget buildExternalKeyBar() {
    final inputEnabled =
        !_readOnly &&
        _history == null &&
        !_historyLoading &&
        !_selectionMode &&
        !_searchOpen &&
        _deliveryState == TerminalDeliveryState.ready;
    return Consumer(
      builder: (context, ref, _) => TerminalCommandBar(
        actions:
            ref.watch(terminalQuickActionsProvider).valueOrNull ??
            defaultTerminalQuickActions,
        inputEnabled: inputEnabled,
        modifiers: _modifiers,
        keyboardControl: _KeyboardKeyButton(
          keyboardVisible: widget.keyboardInset > 100,
          focusLocked: _keyboardFocusLocked,
          onTap: inputEnabled ? _toggleKeyboardVisibility : null,
          onLongPress: inputEnabled ? _toggleKeyboardFocusLock : null,
        ),
        onAction: _runQuickAction,
        onConfigure: _openCommandBarEditor,
        functionKeysActive: _fnOpen,
        onFunctionKeys: _toggleFn,
      ),
    );
  }

  void _openCommandBarEditor() {
    final container = ProviderScope.containerOf(context);
    unawaited(
      showAnyttyCommandBarEditor(
        context: context,
        actions:
            container.read(terminalQuickActionsProvider).valueOrNull ??
            defaultTerminalQuickActions,
        onSave: container.read(terminalQuickActionsProvider.notifier).save,
      ),
    );
  }

  void _requestSplit(_TerminalSplitTarget target) {
    setState(() => _toolsOpen = false);
    _notifyControls();
    widget.onSplit(target);
  }

  void _requestCloseSplit() {
    setState(() => _toolsOpen = false);
    _notifyControls();
    widget.onCloseSplit();
  }

  Widget _buildTerminalContent() {
    if (_readOnly && _history == null && !_historyLoading) {
      if (_historyError case final historyError?) {
        return _TerminalFailure(
          dark: true,
          message: historyError,
          onRetry: () => unawaited(_ensureHistory()),
        );
      }
    }

    final history = _history;
    return TerminalHistoryTransition(
      key: ObjectKey(widget.connection),
      active: history != null || _historyLoading,
      ready: history != null && _historyPresented,
      background: _terminalThemeColor(widget.settings.theme.background),
      foreground: _terminalThemeColor(widget.settings.theme.foreground),
      onCancel: _readOnly ? null : () => unawaited(_toggleHistory()),
      fallback: _readOnly
          ? ColoredBox(
              color: _terminalThemeColor(widget.settings.theme.background),
            )
          : _buildLiveTerminalContent(),
      child: history == null
          ? const SizedBox.expand()
          : _buildHistoryContent(history),
    );
  }

  Widget _buildLiveTerminalContent() {
    return StreamBuilder<CanonicalLiveScreen>(
      stream: widget.connection.watchScreens(),
      initialData: widget.connection.current,
      builder: (context, snapshot) {
        final screen = snapshot.data ?? widget.connection.current;
        if (snapshot.hasError && screen == null) {
          return _TerminalFailure(
            dark: true,
            message: snapshot.error.toString(),
            onRetry: widget.onReconnect,
          );
        }
        if (screen == null) {
          if (_historyLoading) {
            if (_historyLoadingVisible) {
              return const _WaitingForSnapshot(label: 'Loading history');
            }
            return ColoredBox(
              color: _terminalThemeColor(widget.settings.theme.background),
            );
          }
          return const _WaitingForSnapshot(
            label: 'Waiting for screen snapshot',
          );
        }
        final terminalBody = _buildKeyboardAdjustedTerminal(
          screen: screen,
          child: TerminalKeyboardFrameFreeze(
            visualInset: widget.keyboardVisualInset,
            settledInset: widget.settledKeyboardInset,
            child: RepaintBoundary(
              child: Stack(
                fit: StackFit.expand,
                children: [
                  Consumer(
                    builder: (context, ref, _) {
                      final preferences =
                          ref
                              .watch(terminalPetalMenuPreferencesProvider)
                              .valueOrNull ??
                          TerminalPetalMenuPreferences.defaults;
                      final actions = _terminalPetalActions(preferences);
                      return TerminalPetalMenuRegion(
                        actions: actions,
                        enabled:
                            preferences.enabled &&
                            actions.isNotEmpty &&
                            !_toolsOpen &&
                            !_fnOpen &&
                            !_historyLoading &&
                            !_searchOpen &&
                            !_selectionMode,
                        hapticsEnabled: preferences.hapticsEnabled,
                        onOpened: () {
                          widget.onActivate();
                          _handleLiveInteractionCancel();
                        },
                        onSelected: (action) =>
                            unawaited(_handleTerminalPetalAction(action.id)),
                        child: _FitWidthCanvas(
                          screen: screen,
                          metrics: widget.settings.metrics,
                          background: widget.settings.theme.background,
                          child: TerminalCanvas(
                            screen: screen,
                            settings: widget.settings,
                            reserveLongPress: false,
                            onPresented:
                                widget.connection.acknowledgePresentation,
                            onLinkTap: _openTerminalLink,
                            onTerminalTap: (position) =>
                                _handleLiveTap(screen, position),
                            onInteractionStart: _handleLiveInteractionStart,
                            onInteractionCancel: _handleLiveInteractionCancel,
                            onVerticalDragStart: (details) =>
                                _lastLiveDragPosition = details.localPosition,
                            onVerticalDragUpdate: (details) =>
                                _handleLiveDrag(screen, details),
                            onVerticalDragEnd: (details) =>
                                _handleLiveDragEnd(screen, details),
                            onVerticalDragCancel: _handleLiveInteractionCancel,
                          ),
                        ),
                      );
                    },
                  ),
                  if (snapshot.hasError)
                    _TerminalDeliveryBanner(
                      message: snapshot.error.toString(),
                      onRetry: widget.onReconnect,
                      error: true,
                    )
                  else if (_deliveryState == TerminalDeliveryState.stalled)
                    _TerminalDeliveryBanner(
                      message: 'Terminal rendering is paused',
                      onRetry: widget.onReconnect,
                      error: false,
                    )
                  else if (_deliveryState == TerminalDeliveryState.recovering &&
                      _recoveryNoticeVisible)
                    const _TerminalDeliveryBanner(
                      message: 'Reconnecting terminal',
                      error: false,
                    ),
                ],
              ),
            ),
          ),
        );
        return terminalBody;
      },
    );
  }

  Widget _buildKeyboardAdjustedTerminal({
    required CanonicalLiveScreen screen,
    required Widget child,
  }) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final keyboardInset = widget.keyboardInset;
        final settledKeyboardInset = widget.settledKeyboardInset;
        final alternate =
            screen.alternateScreen || screen.modes?.alternateScreen == true;
        final mode = resolveTerminalKeyboardMode(
          _keyboardMode,
          alternateScreen: alternate,
        );
        final resizing =
            keyboardInset > 0 && mode == TerminalKeyboardMode.resize;
        final keyboardAnimating =
            (keyboardInset - settledKeyboardInset).abs() > 1;
        if (keyboardInset <= 1 && settledKeyboardInset <= 1) {
          _fullTerminalViewportHeight = constraints.maxHeight;
        }
        // The surrounding workspace follows the raw IME inset in a
        // composited layer. Keep only the settled viewport resize here so the
        // terminal does not receive a second visual translation.
        final preserveGrid = keyboardAnimating;
        final layoutHeight = resizing
            ? math.max(0.0, constraints.maxHeight - keyboardInset)
            : preserveGrid
            ? _fullTerminalViewportHeight ?? constraints.maxHeight
            : constraints.maxHeight;
        _scheduleViewportFit(Size(constraints.maxWidth, layoutHeight));
        if (!preserveGrid) {
          return Align(
            alignment: resizing ? Alignment.bottomLeft : Alignment.topLeft,
            child: SizedBox(
              width: constraints.maxWidth,
              height: layoutHeight,
              child: child,
            ),
          );
        }

        return ClipRect(
          child: OverflowBox(
            alignment: Alignment.topLeft,
            minWidth: constraints.maxWidth,
            maxWidth: constraints.maxWidth,
            minHeight: layoutHeight,
            maxHeight: layoutHeight,
            child: SizedBox(
              width: constraints.maxWidth,
              height: layoutHeight,
              child: child,
            ),
          ),
        );
      },
    );
  }

  void _bindResizeControl() {
    _resizeControl = widget.connection.resizeControl;
    _resizeSubscription = widget.connection.watchResizeControl().listen((
      control,
    ) {
      if (!mounted) return;
      setState(() => _resizeControl = control);
      _notifyControls();
      if (control.canResize && _viewportCells != null) {
        unawaited(_fitViewport());
      }
    });
  }

  void _bindTerminalState() {
    _terminalState = widget.connection.terminalState;
    _terminalStateSubscription = widget.connection.watchTerminalState().listen((
      state,
    ) {
      if (!mounted) return;
      final becameReadOnly =
          !terminalUsesHistoryOnly(_terminalState) &&
          terminalUsesHistoryOnly(state);
      setState(() => _terminalState = state);
      _notifyControls();
      if (becameReadOnly) {
        _cancelMomentum();
        FocusManager.instance.primaryFocus?.unfocus();
        setState(() {
          _modifiers = const TerminalModifierState();
          _fnOpen = false;
        });
        _notifyControls();
        _scheduleReadOnlyHistory();
      }
    });
    if (_readOnly) _scheduleReadOnlyHistory();
  }

  void _bindDeliveryState() {
    _deliveryState = widget.connection.deliveryState;
    _updateRecoveryNotice(_deliveryState);
    _deliveryStateSubscription = widget.connection.watchDeliveryState().listen((
      state,
    ) {
      if (mounted) {
        setState(() => _deliveryState = state);
        _updateRecoveryNotice(state);
        _notifyControls();
      }
    });
  }

  void _updateRecoveryNotice(TerminalDeliveryState state) {
    _recoveryNoticeGate.setRecovering(
      state == TerminalDeliveryState.recovering,
    );
  }

  void _scheduleReadOnlyHistory() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted &&
          _readOnly &&
          _historyViewportCells != null &&
          _history == null &&
          !_historyLoading) {
        unawaited(_ensureHistory());
      }
    });
  }

  void _scheduleHistoryProjectionFit(Size viewport) {
    if (viewport.isEmpty ||
        !viewport.width.isFinite ||
        !viewport.height.isFinite) {
      return;
    }
    final cells = (
      cols: (viewport.width / widget.settings.metrics.cellWidth).floor().clamp(
        20,
        500,
      ),
      rows: (viewport.height / widget.settings.metrics.rowHeight).floor().clamp(
        4,
        300,
      ),
    );
    if (_historyViewportCells?.cols != cells.cols) {
      _historyProjectionFailedCols = null;
    }
    _historyViewportCells = cells;
    final history = _history;
    if (history != null &&
        history.cols != cells.cols &&
        _historyProjectionFailedCols != cells.cols &&
        !_historyLoading) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        unawaited(
          _restartHistoryProjection(expected: history, cols: cells.cols),
        );
      });
    } else if (_readOnly && history == null && !_historyLoading) {
      _scheduleReadOnlyHistory();
    }
  }

  void _scheduleViewportFit(Size viewport) {
    if (_readOnly ||
        _history != null ||
        viewport.isEmpty ||
        !viewport.width.isFinite ||
        !viewport.height.isFinite) {
      return;
    }
    final cells = (
      cols: (viewport.width / widget.settings.metrics.cellWidth).floor().clamp(
        20,
        500,
      ),
      rows: (viewport.height / widget.settings.metrics.rowHeight).floor().clamp(
        4,
        300,
      ),
    );
    if (_viewportCells == cells) return;
    _viewportCells = cells;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (widget.settings.autoAcquireResizeOwner &&
          !_resizeControl.canResize &&
          !_autoAcquireAttempted) {
        unawaited(_autoAcquireResizeOwnership());
      } else {
        unawaited(_fitViewport());
      }
    });
  }

  Future<void> _autoAcquireResizeOwnership() async {
    final target = _viewportCells;
    if (target == null || _autoAcquireAttempted || _readOnly) return;
    _autoAcquireAttempted = true;
    try {
      final control = await widget.connection.requestResizeOwnership(
        cols: target.cols,
        rows: target.rows,
      );
      if (!mounted) return;
      setState(() => _resizeControl = control);
      await _fitViewport();
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not acquire terminal size: $error')),
      );
    }
  }

  Future<void> _fitViewport() async {
    if (_readOnly ||
        _resizePending ||
        _history != null ||
        !_resizeControl.canResize) {
      return;
    }
    _resizePending = true;
    try {
      while (mounted && _history == null && _resizeControl.canResize) {
        final target = _viewportCells;
        if (target == null) return;
        await widget.connection.fitViewport(
          cols: target.cols,
          rows: target.rows,
        );
        if (_viewportCells == target) return;
      }
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('Resize failed: $error')));
    } finally {
      _resizePending = false;
    }
  }

  Future<void> _showResizeControls() async {
    if (_readOnly) return;
    FocusManager.instance.primaryFocus?.unfocus();
    await showModalBottomSheet<void>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      backgroundColor: AnyttyPalette.of(context).surface,
      showDragHandle: true,
      builder: (context) => _ResizeControlSheet(
        connection: widget.connection,
        initialControl: _resizeControl,
        viewportCells: _viewportCells,
      ),
    );
  }

  Future<void> _showTerminalSettings() async {
    FocusManager.instance.primaryFocus?.unfocus();
    await showModalBottomSheet<void>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) => FractionallySizedBox(
        heightFactor: 0.86,
        child: _TerminalSettingsSheet(
          initialSettings: widget.settings,
          onChanged: widget.onSettingsChanged,
        ),
      ),
    );
  }

  Future<void> _changeKeyboardMode(TerminalKeyboardMode next) async {
    if (_keyboardMode == next) return;
    final previous = _keyboardMode;
    setState(() {
      _keyboardMode = next;
      _viewportCells = null;
    });
    _notifyControls();
    try {
      await widget.onKeyboardModeChanged(next);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _keyboardMode = previous;
        _viewportCells = null;
      });
      _notifyControls();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not save keyboard mode: $error')),
      );
    }
  }

  void toggleTools() {
    setState(() {
      _toolsOpen = !_toolsOpen;
      if (_toolsOpen) _fnOpen = false;
    });
    _notifyControls();
  }

  bool handleBack() {
    if (_fnOpen) {
      setState(() => _fnOpen = false);
      _notifyControls();
      return true;
    }
    if (_toolsOpen) {
      setState(() => _toolsOpen = false);
      _notifyControls();
      return true;
    }
    if (_selectionMode) {
      _closeSelection();
      return true;
    }
    if (_searchOpen) {
      _closeSearch();
      return true;
    }
    if (_history != null && !_readOnly) {
      unawaited(_toggleHistory());
      return true;
    }
    return false;
  }

  Future<void> _handleTerminalToolAction(_TerminalToolAction action) async {
    if (!mounted) return;
    setState(() => _toolsOpen = false);
    _notifyControls();
    switch (action) {
      case _TerminalToolAction.commandBar:
        _openCommandBarEditor();
      case _TerminalToolAction.quickKeys:
        _toggleFn();
      case _TerminalToolAction.keyboard:
        _toggleKeyboardVisibility();
      case _TerminalToolAction.enter:
        _sendEnter();
      case _TerminalToolAction.escape:
        _sendHid(_usbKeyboardPage | 0x29);
      case _TerminalToolAction.tab:
        _sendHid(_usbKeyboardPage | 0x2b);
      case _TerminalToolAction.backspace:
        _sendHid(_usbKeyboardPage | 0x2a);
      case _TerminalToolAction.delete:
        _sendHid(_usbKeyboardPage | 0x4c);
      case _TerminalToolAction.interrupt:
        _sendText('\x03');
      case _TerminalToolAction.eof:
        _sendText('\x04');
      case _TerminalToolAction.suspend:
        _sendText('\x1a');
      case _TerminalToolAction.clear:
        _sendText('\x0c');
      case _TerminalToolAction.arrowLeft:
        _sendHid(_usbKeyboardPage | 0x50);
      case _TerminalToolAction.arrowDown:
        _sendHid(_usbKeyboardPage | 0x51);
      case _TerminalToolAction.arrowUp:
        _sendHid(_usbKeyboardPage | 0x52);
      case _TerminalToolAction.arrowRight:
        _sendHid(_usbKeyboardPage | 0x4f);
      case _TerminalToolAction.home:
        _sendHid(_usbKeyboardPage | 0x4a);
      case _TerminalToolAction.end:
        _sendHid(_usbKeyboardPage | 0x4d);
      case _TerminalToolAction.pageUp:
        _sendHid(_usbKeyboardPage | 0x4b);
      case _TerminalToolAction.pageDown:
        _sendHid(_usbKeyboardPage | 0x4e);
      case _TerminalToolAction.paste:
        await _pasteClipboard();
      case _TerminalToolAction.history:
        await _toggleHistory();
      case _TerminalToolAction.search:
        await _toggleSearch();
      case _TerminalToolAction.selection:
        await _openSelection();
      case _TerminalToolAction.copyScreen:
        await _copyLiveScreen();
      case _TerminalToolAction.resources:
        await widget.onShowResources();
      case _TerminalToolAction.files:
        await widget.onOpenFiles();
      case _TerminalToolAction.split:
        _requestSplit(_TerminalSplitTarget.below);
      case _TerminalToolAction.splitRows:
        _requestSplit(_TerminalSplitTarget.below);
      case _TerminalToolAction.splitColumns:
        _requestSplit(_TerminalSplitTarget.right);
      case _TerminalToolAction.syncInput:
        widget.onToggleSync();
      case _TerminalToolAction.resize:
        await _showResizeControls();
      case _TerminalToolAction.reconnect:
        widget.onReconnect();
      case _TerminalToolAction.settings:
        await _showTerminalSettings();
    }
    _notifyControls();
  }

  List<TerminalPetalMenuItem> _terminalPetalActions(
    TerminalPetalMenuPreferences preferences,
  ) {
    final layout = preferences.visibleLayout;
    var cursor = 0;
    List<TerminalPetalMenuItem> buildLevel(int depth) {
      final actions = <TerminalPetalMenuItem>[];
      while (cursor < layout.length && layout[cursor].depth == depth) {
        final placement = layout[cursor];
        cursor += 1;
        final children = cursor < layout.length && layout[cursor].depth > depth
            ? buildLevel(depth + 1)
            : const <TerminalPetalMenuItem>[];
        final action = _terminalPetalAction(placement.id, children: children);
        if (action != null) actions.add(action);
      }
      return List.unmodifiable(actions);
    }

    return buildLevel(0);
  }

  TerminalPetalMenuItem? _terminalPetalAction(
    String id, {
    List<TerminalPetalMenuItem> children = const [],
  }) => switch (id) {
    'history' => TerminalPetalMenuItem(
      id: 'history',
      label: _history == null
          ? _terminalPetalLabel('History', '历史')
          : _terminalPetalLabel('Live', '实时'),
      icon: LucideIcons.history,
      enabled: !_readOnly || _history == null,
      children: children,
    ),
    'search' => TerminalPetalMenuItem(
      id: 'search',
      label: _terminalPetalLabel('Search', '搜索'),
      icon: LucideIcons.search,
      children: children,
    ),
    'selection' => TerminalPetalMenuItem(
      id: 'selection',
      label: _terminalPetalLabel('Select', '选择'),
      icon: LucideIcons.scanText,
      children: children,
    ),
    'preview' => TerminalPetalMenuItem(
      id: 'preview',
      label: _terminalPetalLabel('Preview', '多终端预览'),
      icon: LucideIcons.layoutGrid,
      children: children,
    ),
    'paste' => TerminalPetalMenuItem(
      id: 'paste',
      label: _terminalPetalLabel('Paste', '粘贴'),
      icon: LucideIcons.clipboardPaste,
      enabled:
          !_readOnly &&
          _history == null &&
          !_historyLoading &&
          _deliveryState == TerminalDeliveryState.ready,
      children: children,
    ),
    'enter' => TerminalPetalMenuItem(
      id: 'enter',
      label: _terminalPetalLabel('Enter', '回车'),
      icon: LucideIcons.cornerDownLeft,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'escape' => TerminalPetalMenuItem(
      id: 'escape',
      label: 'Esc',
      icon: LucideIcons.badgeX,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'resources' => TerminalPetalMenuItem(
      id: 'resources',
      label: _terminalPetalLabel('Resources', '资源'),
      icon: LucideIcons.activity,
      children: children,
    ),
    'more' => TerminalPetalMenuItem(
      id: 'more',
      label: _terminalPetalLabel('More', '更多'),
      icon: LucideIcons.ellipsis,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'input-tools' => TerminalPetalMenuItem(
      id: 'input-tools',
      label: _terminalPetalLabel('Input', '输入'),
      icon: LucideIcons.command,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'navigation-tools' => TerminalPetalMenuItem(
      id: 'navigation-tools',
      label: _terminalPetalLabel('Navigate', '导航'),
      icon: LucideIcons.navigation,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'session-tools' => TerminalPetalMenuItem(
      id: 'session-tools',
      label: _terminalPetalLabel('Layout', '布局'),
      icon: LucideIcons.panelsTopLeft,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'command-bar' => TerminalPetalMenuItem(
      id: 'command-bar',
      label: _terminalPetalLabel('Shortcut', '快捷栏'),
      icon: LucideIcons.slidersHorizontal,
      children: children,
    ),
    'copy-screen' => TerminalPetalMenuItem(
      id: 'copy-screen',
      label: _terminalPetalLabel('Copy', '复制'),
      icon: LucideIcons.copy,
      children: children,
    ),
    'quick-keys' => TerminalPetalMenuItem(
      id: 'quick-keys',
      label: _terminalPetalLabel('Quick Keys', '快捷键'),
      icon: LucideIcons.zap,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'keyboard' => TerminalPetalMenuItem(
      id: 'keyboard',
      label: _terminalPetalLabel('Keyboard', '键盘'),
      icon: LucideIcons.keyboard,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'tab' => TerminalPetalMenuItem(
      id: 'tab',
      label: 'Tab',
      icon: LucideIcons.arrowRightToLine,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'backspace' => TerminalPetalMenuItem(
      id: 'backspace',
      label: _terminalPetalLabel('Backspace', '退格'),
      icon: LucideIcons.delete,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'delete' => TerminalPetalMenuItem(
      id: 'delete',
      label: _terminalPetalLabel('Delete', '删除'),
      icon: LucideIcons.eraser,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'interrupt' => TerminalPetalMenuItem(
      id: 'interrupt',
      label: 'Ctrl+C',
      icon: LucideIcons.circleStop,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'eof' => TerminalPetalMenuItem(
      id: 'eof',
      label: 'Ctrl+D',
      icon: LucideIcons.logOut,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'suspend' => TerminalPetalMenuItem(
      id: 'suspend',
      label: 'Ctrl+Z',
      icon: LucideIcons.pause,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'clear' => TerminalPetalMenuItem(
      id: 'clear',
      label: _terminalPetalLabel('Clear', '清屏'),
      icon: LucideIcons.eraser,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-left' => TerminalPetalMenuItem(
      id: 'arrow-left',
      label: _terminalPetalLabel('Left', '左'),
      icon: LucideIcons.arrowLeft,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-down' => TerminalPetalMenuItem(
      id: 'arrow-down',
      label: _terminalPetalLabel('Down', '下'),
      icon: LucideIcons.arrowDown,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-up' => TerminalPetalMenuItem(
      id: 'arrow-up',
      label: _terminalPetalLabel('Up', '上'),
      icon: LucideIcons.arrowUp,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-right' => TerminalPetalMenuItem(
      id: 'arrow-right',
      label: _terminalPetalLabel('Right', '右'),
      icon: LucideIcons.arrowRight,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'home' => TerminalPetalMenuItem(
      id: 'home',
      label: _terminalPetalLabel('Home', '行首'),
      icon: LucideIcons.home,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'end' => TerminalPetalMenuItem(
      id: 'end',
      label: _terminalPetalLabel('End', '行尾'),
      icon: LucideIcons.arrowRightToLine,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'page-up' => TerminalPetalMenuItem(
      id: 'page-up',
      label: _terminalPetalLabel('PgUp', '上翻'),
      icon: LucideIcons.chevronsUp,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'page-down' => TerminalPetalMenuItem(
      id: 'page-down',
      label: _terminalPetalLabel('PgDn', '下翻'),
      icon: LucideIcons.chevronsDown,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'split' => TerminalPetalMenuItem(
      id: 'split',
      label: _terminalPetalLabel('Split', '分屏'),
      icon: LucideIcons.rows2,
      enabled: widget.canSplit,
      children: children,
    ),
    'split-rows' => TerminalPetalMenuItem(
      id: 'split-rows',
      label: _terminalPetalLabel('Rows', '上下'),
      icon: LucideIcons.rows2,
      enabled: widget.canSplit,
      children: children,
    ),
    'split-columns' => TerminalPetalMenuItem(
      id: 'split-columns',
      label: _terminalPetalLabel('Columns', '左右'),
      icon: LucideIcons.columns2,
      enabled: widget.canSplit,
      children: children,
    ),
    'sync-input' => TerminalPetalMenuItem(
      id: 'sync-input',
      label: widget.syncInput
          ? _terminalPetalLabel('Synced', '已同步')
          : _terminalPetalLabel('Sync', '同步'),
      icon: LucideIcons.gitCompareArrows,
      enabled: widget.splitOpen,
      children: children,
    ),
    'resize' => TerminalPetalMenuItem(
      id: 'resize',
      label: _terminalPetalLabel('Resize', '调整'),
      icon: LucideIcons.maximize2,
      enabled: !_readOnly,
      children: children,
    ),
    'files' => TerminalPetalMenuItem(
      id: 'files',
      label: _terminalPetalLabel('Files', '文件'),
      icon: LucideIcons.folderOpen,
      children: children,
    ),
    'reconnect' => TerminalPetalMenuItem(
      id: 'reconnect',
      label: _terminalPetalLabel('Reconnect', '重连'),
      icon: LucideIcons.refreshCw,
      children: children,
    ),
    'settings' => TerminalPetalMenuItem(
      id: 'settings',
      label: _terminalPetalLabel('Settings', '设置'),
      icon: LucideIcons.settings,
      children: children,
    ),
    _ => null,
  };

  String _terminalPetalLabel(String en, String zh) =>
      anyttyText(context, en: en, zh: zh);

  bool get _terminalPetalInputEnabled =>
      !_readOnly &&
      _history == null &&
      !_historyLoading &&
      !_searchOpen &&
      _deliveryState == TerminalDeliveryState.ready;

  Future<void> _handleTerminalPetalAction(String id) async {
    if (id == 'preview') {
      // 花瓣「多终端预览」：打开同一个全屏毛玻璃层。
      widget.onOpenPreview?.call();
      return;
    }
    final action = switch (id) {
      'paste' => _TerminalToolAction.paste,
      'history' => _TerminalToolAction.history,
      'search' => _TerminalToolAction.search,
      'selection' => _TerminalToolAction.selection,
      'copy-screen' => _TerminalToolAction.copyScreen,
      'command-bar' => _TerminalToolAction.commandBar,
      'quick-keys' => _TerminalToolAction.quickKeys,
      'keyboard' => _TerminalToolAction.keyboard,
      'enter' => _TerminalToolAction.enter,
      'escape' => _TerminalToolAction.escape,
      'tab' => _TerminalToolAction.tab,
      'backspace' => _TerminalToolAction.backspace,
      'delete' => _TerminalToolAction.delete,
      'interrupt' => _TerminalToolAction.interrupt,
      'eof' => _TerminalToolAction.eof,
      'suspend' => _TerminalToolAction.suspend,
      'clear' => _TerminalToolAction.clear,
      'arrow-left' => _TerminalToolAction.arrowLeft,
      'arrow-down' => _TerminalToolAction.arrowDown,
      'arrow-up' => _TerminalToolAction.arrowUp,
      'arrow-right' => _TerminalToolAction.arrowRight,
      'home' => _TerminalToolAction.home,
      'end' => _TerminalToolAction.end,
      'page-up' => _TerminalToolAction.pageUp,
      'page-down' => _TerminalToolAction.pageDown,
      'resources' => _TerminalToolAction.resources,
      'files' => _TerminalToolAction.files,
      'split' => _TerminalToolAction.split,
      'split-rows' => _TerminalToolAction.splitRows,
      'split-columns' => _TerminalToolAction.splitColumns,
      'sync-input' => _TerminalToolAction.syncInput,
      'resize' => _TerminalToolAction.resize,
      'reconnect' => _TerminalToolAction.reconnect,
      'settings' => _TerminalToolAction.settings,
      _ => null,
    };
    if (action != null) await _handleTerminalToolAction(action);
  }

  Widget _buildHistoryContent(FrozenHistory history) {
    final layout = _historyLayoutFor(history);
    final viewportRows =
        _historyViewportCells?.rows ??
        (history.anchor.screenRows > 0 ? history.anchor.screenRows : 24);
    final trailingRows = historyViewportTailRows(
      history: history,
      viewportRows: viewportRows,
    );
    return Stack(
      fit: StackFit.expand,
      children: [
        ScrollbarTheme(
          data: ScrollbarTheme.of(context).copyWith(
            thumbColor: WidgetStateProperty.resolveWith((states) {
              final foreground = _terminalThemeColor(
                widget.settings.theme.foreground,
              );
              return foreground.withValues(
                alpha: states.contains(WidgetState.dragged) ? 0.72 : 0.38,
              );
            }),
            thickness: WidgetStateProperty.resolveWith(
              (states) => states.contains(WidgetState.dragged) ? 8 : 3,
            ),
            radius: const Radius.circular(3),
            minThumbLength: 24,
          ),
          child: Scrollbar(
            controller: _historyScroll,
            interactive: true,
            child: TerminalHistoryCanvas(
              rows: history.rows,
              cols: history.cols,
              settings: widget.settings,
              scrollController: _historyScroll,
              layout: layout,
              highlights: _historyHighlights,
              searchMatches: _searchMatches,
              trailingRows: trailingRows,
              selectionEnabled: _selectionMode,
              canLoadOlder: history.hasMore || _historyLoading,
              onLinkTap: _openTerminalLink,
              onSelectionChanged: (selection) {
                final selectionBecameAvailable = _selection == null;
                _selection = selection;
                _syncHistoryHighlights();
                if (selectionBecameAvailable) _notifyControls();
              },
            ),
          ),
        ),
        Positioned(
          top: 8,
          left: 8,
          right: 8,
          child: _HistoryContextBar(
            liveEnabled: !_readOnly,
            searchActive: _searchOpen,
            selectionActive: _selectionMode,
            status: _historyLoadingVisible
                ? 'Loading older rows'
                : '${anyttyText(context, en: 'History', zh: '历史模式')} '
                      '${history.rows.length} / ${history.logicalTotal}',
            onLive: _toggleHistory,
            onSearch: _toggleSearch,
            onSelection: _openSelection,
          ),
        ),
        if (_historyError case final error?)
          Positioned(
            left: 8,
            right: 8,
            bottom: 8,
            child: Material(
              color: const Color(0xff7f1d1d),
              borderRadius: BorderRadius.circular(4),
              child: Row(
                children: [
                  Expanded(
                    child: Padding(
                      padding: const EdgeInsets.only(
                        left: 10,
                        top: 8,
                        bottom: 8,
                      ),
                      child: Text(
                        error,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 12,
                        ),
                      ),
                    ),
                  ),
                  if (_historyRequiresReload)
                    IconButton(
                      tooltip: 'Reload frozen history',
                      color: Colors.white,
                      onPressed: _reloadHistory,
                      icon: const Icon(Icons.refresh_rounded),
                    ),
                  IconButton(
                    tooltip: 'Dismiss history error',
                    color: Colors.white,
                    onPressed: () => setState(() {
                      _historyError = null;
                      _historyRequiresReload = false;
                    }),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _toggleHistory() async {
    if (_readOnly) {
      if (_history == null) await _ensureHistory();
      return;
    }
    if (_history != null) {
      final history = _history!;
      setState(() {
        _cancelHistoryRequest();
        _history = null;
        _historyError = null;
        _historyRequiresReload = false;
        _historyPositioned = false;
        _historyPresented = false;
        _resetHistoryInteraction();
      });
      _releaseHistory(widget.connection, history);
      return;
    }
    if (_historyLoading) {
      setState(_cancelHistoryRequest);
      return;
    }
    await _ensureHistory();
  }

  Future<FrozenHistory?> _ensureHistory() async {
    if (_history case final history?) return history;
    if (_historyLoading) return null;
    final connection = widget.connection;
    final projectionCols = _historyViewportCells?.cols;
    final requestEpoch = ++_historyRequestEpoch;
    setState(() {
      _historyLoading = true;
      _historyLoadingVisible = false;
      _historyError = null;
      _historyRequiresReload = false;
      _historyPresented = false;
      _fnOpen = false;
    });
    try {
      // Paint entry feedback before starting native history work.
      await WidgetsBinding.instance.endOfFrame;
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection) {
        return null;
      }
      final merged = await connection.openHistory(cols: projectionCols);
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection) {
        _releaseHistory(connection, merged.history);
        return null;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _history = merged.history;
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyRequiresReload = false;
      });
      _scheduleHistoryPosition(merged.history);
      return merged.history;
    } catch (error) {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection) {
        return null;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyError = error.toString();
        _historyRequiresReload = frozenHistoryRequiresReload(error.toString());
      });
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
      return null;
    }
  }

  Future<void> _restartHistoryProjection({
    required FrozenHistory expected,
    required int cols,
  }) async {
    if (_historyLoading || _history != expected || expected.cols == cols) {
      return;
    }
    final connection = widget.connection;
    final requestEpoch = ++_historyRequestEpoch;
    setState(() {
      _historyLoading = true;
      _historyLoadingVisible = false;
      _historyError = null;
      _historyRequiresReload = false;
    });
    _scheduleHistoryLoadingIndicator(requestEpoch);
    try {
      final merged = await connection.openHistory(cols: cols);
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        _releaseHistory(connection, merged.history);
        return;
      }
      if (_historyViewportCells?.cols != cols) {
        _releaseHistory(connection, merged.history);
        _historyLoadingDelay?.cancel();
        setState(() {
          _historyLoading = false;
          _historyLoadingVisible = false;
        });
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _history = merged.history;
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyRequiresReload = false;
        _historyProjectionFailedCols = null;
        _historyPositioned = false;
        _resetHistoryInteraction();
      });
      _scheduleHistoryPosition(merged.history);
      if (expected.token != merged.history.token ||
          expected.generation != merged.history.generation) {
        _releaseHistory(connection, expected);
      }
    } catch (error) {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyError = error.toString();
        _historyRequiresReload = true;
        _historyProjectionFailedCols = cols;
      });
    }
  }

  void _scheduleHistoryPosition(FrozenHistory expected) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || _history != expected || !_historyScroll.hasClients) {
        return;
      }
      final position = _historyScroll.position;
      final anchorRow = historyViewportAnchorRow(expected);
      final target = (anchorRow * widget.settings.metrics.rowHeight).clamp(
        0.0,
        position.maxScrollExtent,
      );
      _resetHistoryScrollSample(target.toDouble(), clearVelocity: true);
      _historyScroll.jumpTo(target);
      setState(() {
        _historyPositioned = true;
        _historyPresented = true;
      });
      _handleHistoryScroll();
    });
  }

  void _handleHistoryScroll() {
    _sampleHistoryScrollVelocity();
    final history = _history;
    final prefetch = adaptiveHistoryPrefetchPlan(
      baseThresholdRows: widget.settings.historyPrefetchThresholdRows,
      upwardVelocityRowsPerSecond: _historyUpwardVelocityRows,
    );
    if (_searchOpen ||
        !_historyPositioned ||
        _historyLoading ||
        history == null ||
        !history.hasMore ||
        !_historyScroll.hasClients ||
        _historyScroll.position.pixels >
            prefetch.thresholdRows * widget.settings.metrics.rowHeight) {
      return;
    }
    unawaited(_loadOlderHistory(history, limit: prefetch.requestRows));
  }

  void _sampleHistoryScrollVelocity() {
    if (!_historyScroll.hasClients) return;
    final now = DateTime.now();
    final offset = _historyScroll.position.pixels;
    final previousTime = _historyScrollSampleTime;
    final previousOffset = _historyScrollSampleOffset;
    _historyScrollSampleTime = now;
    _historyScrollSampleOffset = offset;
    if (previousTime == null || previousOffset == null) return;
    final elapsedMicros = now.difference(previousTime).inMicroseconds;
    if (elapsedMicros < 4000 || elapsedMicros > 500000) {
      _historyUpwardVelocityRows *= 0.6;
      return;
    }
    final upwardRows =
        (previousOffset - offset) / widget.settings.metrics.rowHeight;
    final instantaneous = math.max(
      0.0,
      upwardRows * Duration.microsecondsPerSecond / elapsedMicros,
    );
    _historyUpwardVelocityRows = instantaneous > 0
        ? math.max(instantaneous, _historyUpwardVelocityRows * 0.72)
        : _historyUpwardVelocityRows * 0.6;
  }

  void _resetHistoryScrollSample(double offset, {required bool clearVelocity}) {
    _historyScrollSampleTime = DateTime.now();
    _historyScrollSampleOffset = offset;
    if (clearVelocity) _historyUpwardVelocityRows = 0;
  }

  Future<void> _loadOlderHistory(
    FrozenHistory expected, {
    int limit = maximumHistoryWindowRequestRows,
  }) async {
    final connection = widget.connection;
    final requestEpoch = ++_historyRequestEpoch;
    setState(() {
      _historyLoading = true;
      _historyLoadingVisible = false;
      _historyError = null;
      _historyRequiresReload = false;
    });
    _scheduleHistoryLoadingIndicator(requestEpoch);
    try {
      final merged = await connection.loadOlderHistory(expected, limit: limit);
      if (merged.prependedRows == 0 && merged.history.hasMore) {
        throw const NativeSessionException('History request did not advance');
      }
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        return;
      }
      _historyLoadingDelay?.cancel();
      _historyScroll.preservePrepend(
        merged.prependedRows * widget.settings.metrics.rowHeight,
      );
      setState(() {
        _history = merged.history;
        _historyLoadingVisible = false;
        _historyRequiresReload = false;
      });
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted ||
            requestEpoch != _historyRequestEpoch ||
            connection != widget.connection ||
            _history != merged.history) {
          return;
        }
        setState(() => _historyLoading = false);
        if (_historyScroll.hasClients) {
          _resetHistoryScrollSample(
            _historyScroll.offset,
            clearVelocity: false,
          );
        }
        _handleHistoryScroll();
      });
    } catch (error) {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyError = error.toString();
        _historyRequiresReload = frozenHistoryRequiresReload(error.toString());
      });
    }
  }

  void _scheduleHistoryLoadingIndicator(int requestEpoch) {
    _historyLoadingDelay?.cancel();
    _historyLoadingDelay = Timer(const Duration(seconds: 2), () {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          !_historyLoading) {
        return;
      }
      setState(() => _historyLoadingVisible = true);
    });
  }

  void _cancelHistoryRequest() {
    _historyRequestEpoch += 1;
    _historyLoadingDelay?.cancel();
    _historyLoadingDelay = null;
    _historyLoading = false;
    _historyLoadingVisible = false;
  }

  Future<void> _reloadHistory() async {
    if (_historyLoading) return;
    final previous = _history;
    setState(() {
      _historyProjectionFailedCols = null;
      _history = null;
      _historyError = null;
      _historyRequiresReload = false;
      _historyPositioned = false;
      _historyPresented = false;
      _resetHistoryInteraction();
    });
    if (previous != null) _releaseHistory(widget.connection, previous);
    await _ensureHistory();
  }

  Future<void> _openSelection() async {
    _hideKeyboard();
    final history = await _ensureHistory();
    if (!mounted || history == null) return;
    setState(() {
      _selectionMode = true;
      _selection = null;
      _searchOpen = false;
      _searchEpoch += 1;
      _searching = false;
      _searchScanActive = false;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _closeSelection() {
    setState(() {
      _selectionMode = false;
      _selection = null;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _selectAllLoaded() {
    final history = _history;
    if (history == null || history.rows.isEmpty) return;
    _selection = selectAllHistory(_historyLayoutFor(history));
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _selectVisible() {
    final history = _history;
    if (history == null || history.rows.isEmpty || !_historyScroll.hasClients) {
      return;
    }
    final position = _historyScroll.position;
    final first = (position.pixels / widget.settings.metrics.rowHeight).floor();
    final last =
        ((position.pixels + position.viewportDimension) /
                widget.settings.metrics.rowHeight)
            .ceil()
            .clamp(1, history.rows.length) -
        1;
    _selection = selectVisibleHistory(
      layout: _historyLayoutFor(history),
      firstRow: first,
      lastRow: last,
    );
    _syncHistoryHighlights();
    _notifyControls();
  }

  Future<void> _copyLiveScreen() async {
    final screen = widget.connection.current;
    if (screen == null) {
      _showTerminalNotice(
        anyttyText(
          context,
          en: 'No terminal screen is available',
          zh: '当前没有可复制的终端画面',
        ),
      );
      return;
    }
    final text = screen.screenRows
        .map(
          (row) => terminalRowSemanticText(
            row,
            maxCharacters: math.max(512, screen.cols * 4),
          ),
        )
        .join('\n')
        .trimRight();
    if (text.isEmpty) {
      _showTerminalNotice(
        anyttyText(context, en: 'The terminal screen is empty', zh: '终端画面为空'),
      );
      return;
    }
    await Clipboard.setData(ClipboardData(text: text));
    if (!mounted) return;
    _showTerminalNotice(
      anyttyText(context, en: 'Terminal screen copied', zh: '已复制当前终端画面'),
    );
  }

  void _showTerminalNotice(String message) {
    final messenger = ScaffoldMessenger.maybeOf(context);
    if (messenger == null) return;
    messenger
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(message),
          duration: const Duration(milliseconds: 1500),
        ),
      );
  }

  Future<void> _copySelection() async {
    final history = _history;
    final selection = _selection;
    if (history == null || selection == null || _copying) return;
    setState(() => _copying = true);
    try {
      final range = normalizeHistorySelection(
        layout: _historyLayoutFor(history),
        selection: selection,
      );
      final text = await widget.connection.copyHistoryRange(history, range);
      await Clipboard.setData(ClipboardData(text: text));
      if (!mounted || !_sameFrozenHistory(history, _history)) return;
      setState(() {
        _copying = false;
        _selectionMode = false;
        _selection = null;
      });
      _syncHistoryHighlights();
      _notifyControls();
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Selection copied')));
    } catch (error) {
      if (!mounted) return;
      setState(() => _copying = false);
      _notifyControls();
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  Future<void> _toggleSearch() async {
    if (_searchOpen) {
      _closeSearch();
      return;
    }
    final history = await _ensureHistory();
    if (!mounted || history == null) return;
    setState(() {
      if (_historyLoading) _cancelHistoryRequest();
      _selectionMode = false;
      _selection = null;
      _searchOpen = true;
    });
    _syncHistoryHighlights();
    _notifyControls();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _searchFocus.requestFocus();
    });
  }

  void _closeSearch() {
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _searchOpen = false;
      _searching = false;
      _searchScanActive = false;
      _searchEpoch += 1;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _changeSearchMode(HistorySearchMode mode) {
    if (mode == _searchMode) return;
    setState(() {
      _searchEpoch += 1;
      _searchMode = mode;
      _searching = false;
      _searchScanActive = false;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
    if (_searchController.text.isNotEmpty) {
      unawaited(
        _runSearch(HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD),
      );
    }
  }

  void _handleSearchQueryChanged(String query) {
    setState(() {
      _searchEpoch += 1;
      _searching = false;
      _searchScanActive = false;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _navigateSearch(HistorySearchDirection direction) {
    if (_searching) return;
    final next = adjacentHistorySearchMatch(
      matches: _searchMatches,
      current: _searchMatch,
      forward:
          direction == HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD,
    );
    if (next == null) {
      if (_searchScanActive) return;
      unawaited(_runSearch(direction));
      return;
    }
    final history = _history;
    if (history == null) return;
    final targetRow = historyRowIndexForRange(
      layout: _historyLayoutFor(history),
      range: next.match,
    );
    if (targetRow == null) {
      unawaited(_loadAdjacentSearchMatch(direction));
      return;
    }
    final wrappedChanged = _searchWrapped != next.wrapped;
    _searchMatch = next.match;
    _searchWrapped = next.wrapped;
    _selection = null;
    _syncHistoryHighlights();
    _scrollToHistoryRange(history, next.match);
    if (wrappedChanged) _notifyControls();
  }

  Future<void> _loadAdjacentSearchMatch(
    HistorySearchDirection direction,
  ) async {
    final history = _history;
    final currentMatch = _searchMatch;
    final query = _searchController.text;
    if (history == null ||
        currentMatch == null ||
        query.isEmpty ||
        _searching) {
      return;
    }
    final epoch = _searchEpoch;
    final start =
        direction == HistorySearchDirection.HISTORY_SEARCH_DIRECTION_BACKWARD
        ? HistoryTextPosition(
            lineId: currentMatch.startLineId,
            col: currentMatch.startCol,
          )
        : HistoryTextPosition(
            lineId: currentMatch.endLineId,
            col: currentMatch.endCol,
          );
    setState(() {
      _searching = true;
      _historyError = null;
    });
    _notifyControls();
    try {
      final result = await widget.connection.searchHistory(
        history: history,
        query: query,
        direction: direction,
        mode: _searchMode,
        start: start,
      );
      if (!mounted ||
          epoch != _searchEpoch ||
          _searchController.text != query ||
          !_sameFrozenHistory(history, _history)) {
        return;
      }
      setState(() {
        _history = result.history;
        _searchMatch = result.match;
        _searchWrapped = result.wrapped;
        _searching = false;
        _selection = null;
      });
      _syncHistoryHighlights();
      if (result.match case final match?) {
        _scrollToHistoryRange(result.history, match);
      }
      _notifyControls();
    } catch (error) {
      if (!mounted || epoch != _searchEpoch) return;
      setState(() {
        _searching = false;
        _historyError = error.toString();
      });
      _notifyControls();
    }
  }

  Future<void> _runSearch(HistorySearchDirection direction) async {
    final history = _history;
    final query = _searchController.text;
    if (history == null || query.isEmpty || _searching) {
      if (query.isEmpty && mounted) {
        setState(() {
          _searchMatch = null;
          _searchMatches = const [];
          _searchMatchCount = 0;
          _searchWrapped = false;
          _searchScanActive = false;
        });
        _syncHistoryHighlights();
        _notifyControls();
      }
      return;
    }
    final epoch = ++_searchEpoch;
    final mode = _searchMode;
    final currentMatch = _searchMatch;
    final start = currentMatch == null
        ? null
        : direction == HistorySearchDirection.HISTORY_SEARCH_DIRECTION_BACKWARD
        ? HistoryTextPosition(
            lineId: currentMatch.startLineId,
            col: currentMatch.startCol,
          )
        : HistoryTextPosition(
            lineId: currentMatch.endLineId,
            col: currentMatch.endCol,
          );
    setState(() {
      _searching = true;
      _searchScanActive = false;
      _searchMatchCount = 0;
      _searchMatches = const [];
      _historyError = null;
    });
    _notifyControls();
    try {
      final result = await widget.connection.searchHistory(
        history: history,
        query: query,
        direction: direction,
        mode: mode,
        start: start,
      );
      if (!mounted ||
          epoch != _searchEpoch ||
          _searchController.text != query ||
          !_sameFrozenHistory(history, _history)) {
        return;
      }
      setState(() {
        _history = result.history;
        _searchMatch = result.match;
        _searchWrapped = result.wrapped;
        _searching = false;
        _searchScanActive = true;
        _selection = null;
      });
      _syncHistoryHighlights();
      if (result.match case final match?) {
        _scrollToHistoryRange(result.history, match);
      }
      _notifyControls();
      unawaited(
        _scanSearch(
          history: result.history,
          query: query,
          mode: mode,
          epoch: epoch,
        ),
      );
    } catch (error) {
      if (!mounted || epoch != _searchEpoch) return;
      setState(() {
        _searching = false;
        _searchScanActive = false;
        _historyError = error.toString();
      });
      _notifyControls();
    }
  }

  Future<void> _scanSearch({
    required FrozenHistory history,
    required String query,
    required HistorySearchMode mode,
    required int epoch,
  }) async {
    const maximumIndexedMatches = 512;
    final indexedMatches = <HistoryRange>[];
    var matchCount = 0;
    var publishedInitialIndex = false;
    HistoryTextPosition? start;
    try {
      while (true) {
        final batch = await widget.connection.scanHistory(
          history: history,
          query: query,
          mode: mode,
          start: start,
          maxMatches: 128,
        );
        if (!mounted ||
            epoch != _searchEpoch ||
            !_sameFrozenHistory(history, _history)) {
          return;
        }
        matchCount += batch.matches.length;
        final previousIndexedCount = indexedMatches.length;
        if (indexedMatches.length < maximumIndexedMatches) {
          indexedMatches.addAll(
            batch.matches.take(maximumIndexedMatches - indexedMatches.length),
          );
        }
        final indexChanged = indexedMatches.length != previousIndexedCount;
        final publishIndex =
            indexChanged &&
            (!publishedInitialIndex ||
                indexedMatches.length == maximumIndexedMatches ||
                batch.done);
        if (publishIndex) {
          setState(() {
            _searchMatches = List.unmodifiable(indexedMatches);
            _searchMatchCount = matchCount;
            _searchScanActive = !batch.done;
          });
          publishedInitialIndex = true;
        } else {
          _searchMatchCount = matchCount;
          _searchScanActive = !batch.done;
        }
        _notifyControls();
        if (batch.done) return;
        final next = batch.next;
        if (next == null || next.lineId == Int64.ZERO) {
          throw const NativeSessionException(
            'History search scan had no continuation',
          );
        }
        if (start != null &&
            next.lineId == start.lineId &&
            next.col == start.col) {
          throw const NativeSessionException(
            'History search scan did not advance',
          );
        }
        start = next;
      }
    } catch (error) {
      if (!mounted || epoch != _searchEpoch) return;
      setState(() {
        _searchScanActive = false;
        _historyError = error.toString();
      });
      _notifyControls();
    }
  }

  void _scrollToHistoryRange(FrozenHistory history, HistoryRange range) {
    final rowIndex = historyRowIndexForRange(
      layout: _historyLayoutFor(history),
      range: range,
    );
    if (rowIndex == null) return;
    void jumpToMatch() {
      if (!mounted || !_historyScroll.hasClients) return;
      final position = _historyScroll.position;
      if (historyRowIsVisibleInViewport(
        rowIndex: rowIndex,
        rowHeight: widget.settings.metrics.rowHeight,
        scrollOffset: position.pixels,
        viewportExtent: position.viewportDimension,
      )) {
        return;
      }
      final estimatedMax = math.max(
        position.maxScrollExtent,
        history.rows.length * widget.settings.metrics.rowHeight -
            position.viewportDimension,
      );
      final target =
          (rowIndex * widget.settings.metrics.rowHeight -
                  position.viewportDimension / 2)
              .clamp(0.0, estimatedMax);
      _historyScroll.jumpTo(target);
    }

    if (_historyScroll.hasClients) {
      jumpToMatch();
    } else {
      WidgetsBinding.instance.addPostFrameCallback((_) => jumpToMatch());
    }
  }

  Future<void> _pasteClipboard() async {
    if (_history != null || _historyLoading) return;
    try {
      final data = await Clipboard.getData(Clipboard.kTextPlain);
      final text = data?.text ?? '';
      if (!mounted || text.isEmpty) return;
      final unsafe =
          text.length > 1000 ||
          text.contains('\n') ||
          text.contains('\r') ||
          text.contains('\u001b');
      if (unsafe) {
        final accepted = await showDialog<bool>(
          context: context,
          builder: (context) => AlertDialog(
            title: const Text('Paste into terminal?'),
            content: Text(
              text,
              maxLines: 8,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Paste'),
              ),
            ],
          ),
        );
        if (accepted != true || !mounted) return;
      }
      _reportInput(widget.onInput(_TerminalInputOperation.paste(text, unsafe)));
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  Future<void> _openTerminalLink(TerminalLink link) async {
    final uri = externalTerminalLink(link.url);
    if (uri == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('This terminal link scheme is blocked')),
      );
      return;
    }
    final scheme = uri.scheme.toLowerCase();
    final opensInAnyttyBrowser = scheme == 'http' || scheme == 'https';
    FocusManager.instance.primaryFocus?.unfocus();
    final accepted = await showModalBottomSheet<bool>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      showDragHandle: true,
      builder: (context) => SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                anyttyText(
                  context,
                  en: opensInAnyttyBrowser
                      ? 'Open in AnyTTY browser?'
                      : 'Open terminal link?',
                  zh: opensInAnyttyBrowser ? '在 AnyTTY 浏览器中打开？' : '打开终端链接？',
                ),
                style: const TextStyle(
                  fontSize: 17,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 12),
              SelectableText(
                uri.toString(),
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
              if (link.params.trim().isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  link.params.trim(),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: Color(0xff71717a),
                    fontSize: 11,
                  ),
                ),
              ],
              const SizedBox(height: 18),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: () => Navigator.pop(context, false),
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton.icon(
                    onPressed: () => Navigator.pop(context, true),
                    icon: const Icon(Icons.open_in_new_rounded),
                    label: Text(
                      anyttyText(
                        context,
                        en: opensInAnyttyBrowser ? 'Open in browser' : 'Open',
                        zh: opensInAnyttyBrowser ? '浏览器打开' : '打开',
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
    if (accepted != true || !mounted) return;
    if (opensInAnyttyBrowser) {
      await widget.onOpenBrowser(uri);
      return;
    }
    try {
      await MethodChannelExternalUriLauncher.instance.open(uri);
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  void _handleLiveInteractionStart() {
    _cancelMomentum();
    _liveDragDistance = 0;
    widget.connection.cancelPendingMouseInput();
  }

  void _handleLiveInteractionCancel() {
    _cancelMomentum();
    _liveDragDistance = 0;
  }

  void _handleLiveTap(CanonicalLiveScreen screen, Offset position) {
    if (!widget.active ||
        _readOnly ||
        _history != null ||
        _historyLoading ||
        _deliveryState != TerminalDeliveryState.ready) {
      return;
    }
    if (_mouseTrackingEnabled(screen.modes)) {
      _dispatchModifiedInput(
        (modifiers) => widget.connection.sendPrimaryClick(
          x: position.dx,
          y: position.dy,
          modifiers: modifiers,
        ),
      );
    }
    _showKeyboard();
  }

  void _handleLiveDrag(CanonicalLiveScreen screen, DragUpdateDetails details) {
    if (_readOnly || _history != null || _historyLoading) return;
    _lastLiveDragPosition = details.localPosition;
    _liveDragDistance += details.delta.dy;
    final alternate =
        screen.alternateScreen || screen.modes?.alternateScreen == true;
    if (!alternate) {
      if (_liveDragDistance >= widget.settings.metrics.rowHeight * 2) {
        _liveDragDistance = 0;
        unawaited(_ensureHistory());
      } else if (_liveDragDistance < 0) {
        _liveDragDistance = 0;
      }
      return;
    }

    final modes = screen.modes;
    final mouseTracking = _mouseTrackingEnabled(modes);
    if (!mouseTracking && modes?.alternateScroll != true) {
      _liveDragDistance = 0;
      return;
    }

    var steps = 0;
    while (_liveDragDistance.abs() >= widget.settings.metrics.rowHeight &&
        steps < 8) {
      final up = _liveDragDistance > 0;
      _liveDragDistance += up
          ? -widget.settings.metrics.rowHeight
          : widget.settings.metrics.rowHeight;
      steps += 1;
      _sendAlternateScroll(
        up: up,
        position: details.localPosition,
        mouseTracking: mouseTracking,
      );
    }
  }

  void _handleLiveDragEnd(CanonicalLiveScreen screen, DragEndDetails details) {
    _liveDragDistance = 0;
    final alternate =
        screen.alternateScreen || screen.modes?.alternateScreen == true;
    final modes = screen.modes;
    final mouseTracking = _mouseTrackingEnabled(modes);
    if (!alternate ||
        (!mouseTracking && modes?.alternateScroll != true) ||
        MediaQuery.maybeOf(context)?.disableAnimations == true) {
      return;
    }
    final profile = resolveTerminalMomentumProfile(
      widget.settings.scrollInertia,
    );
    final velocity = details.velocity.pixelsPerSecond.dy;
    if (!profile.enabled || velocity.abs() < profile.minimumVelocity) return;
    _momentumVelocity = velocity;
    _momentumTicks = 0;
    _momentumTimer = Timer.periodic(const Duration(milliseconds: 50), (_) {
      _momentumTicks += 1;
      _momentumVelocity *= math.pow(profile.deceleration, 3).toDouble();
      if (!mounted ||
          _momentumTicks >= 120 ||
          _momentumVelocity.abs() < profile.minimumVelocity ||
          _history != null) {
        _cancelMomentum();
        return;
      }
      _liveDragDistance += _momentumVelocity * 0.05;
      if (_liveDragDistance.abs() < widget.settings.metrics.rowHeight) return;
      final up = _liveDragDistance > 0;
      _liveDragDistance += up
          ? -widget.settings.metrics.rowHeight
          : widget.settings.metrics.rowHeight;
      _sendAlternateScroll(
        up: up,
        position: _lastLiveDragPosition,
        mouseTracking: mouseTracking,
      );
    });
  }

  void _sendAlternateScroll({
    required bool up,
    required Offset position,
    required bool mouseTracking,
  }) {
    if (mouseTracking) {
      _dispatchModifiedInput(
        (modifiers) => widget.connection.sendScroll(
          up: up,
          x: position.dx,
          y: position.dy,
          modifiers: modifiers,
        ),
      );
    } else {
      _sendHid(_usbKeyboardPage | (up ? 0x52 : 0x51));
    }
  }

  void _cancelMomentum() {
    _momentumTimer?.cancel();
    _momentumTimer = null;
    _momentumVelocity = 0;
    _momentumTicks = 0;
  }

  void _resetHistoryInteraction() {
    _searchEpoch += 1;
    _selectionMode = false;
    _selection = null;
    _searchOpen = false;
    _searching = false;
    _searchScanActive = false;
    _copying = false;
    _searchMatch = null;
    _searchMatches = const [];
    _searchMatchCount = 0;
    _searchWrapped = false;
    _searchController.clear();
    _historyHighlights.clear();
  }

  void _syncHistoryHighlights() {
    final history = _history;
    _historyHighlights.update(
      selection: _selection,
      layout: history == null ? null : _historyLayoutFor(history),
      searchMatch: _searchMatch,
    );
  }

  bool _sameFrozenHistory(FrozenHistory left, FrozenHistory? right) {
    return right != null &&
        left.token == right.token &&
        left.generation == right.generation &&
        left.cols == right.cols;
  }

  void _releaseHistory(TerminalConnection connection, FrozenHistory history) {
    unawaited(connection.releaseHistory(history).catchError((_) {}));
  }

  void _handleInputFocusChange() {
    if (!_inputFocus.hasFocus) _keyboardRequestEpoch += 1;
    if (mounted) {
      setState(() {});
      _notifyControls();
    }
  }

  void _showKeyboard() {
    if (!widget.active || _keyboardFocusLocked) return;
    final requestEpoch = ++_keyboardRequestEpoch;
    if (defaultTargetPlatform == TargetPlatform.android) {
      unawaited(_requestAndroidKeyboard());
      return;
    }
    _inputFocus.requestFocus();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted ||
          requestEpoch != _keyboardRequestEpoch ||
          !widget.active ||
          _keyboardFocusLocked ||
          !_inputFocus.hasFocus) {
        return;
      }
      unawaited(SystemChannels.textInput.invokeMethod<void>('TextInput.show'));
    });
  }

  Future<void> _requestAndroidKeyboard() async {
    try {
      await AndroidTerminalInputPlatform.instance.show(_androidInputOwner);
    } catch (error, stackTrace) {
      if (kDebugMode) {
        debugPrint('AnyTTYTerminalInput platform error: $error');
        debugPrintStack(stackTrace: stackTrace);
      }
    }
  }

  void _hideKeyboard() {
    _keyboardRequestEpoch += 1;
    if (defaultTargetPlatform == TargetPlatform.android) {
      _clearAndroidComposition();
      unawaited(AndroidTerminalInputPlatform.instance.hide(_androidInputOwner));
      return;
    }
    unawaited(SystemChannels.textInput.invokeMethod<void>('TextInput.hide'));
    _inputFocus.unfocus();
  }

  void _toggleKeyboardVisibility() {
    if (_keyboardFocusLocked) {
      setState(() => _keyboardFocusLocked = false);
      _notifyControls();
      _showKeyboard();
      return;
    }
    if (widget.keyboardInset > 100) {
      _hideKeyboard();
    } else {
      _showKeyboard();
    }
  }

  void _toggleKeyboardFocusLock() {
    final locked = !_keyboardFocusLocked;
    setState(() => _keyboardFocusLocked = locked);
    _notifyControls();
    if (locked) _hideKeyboard();
  }

  void _toggleModifier(TerminalModifier modifier) {
    setState(() => _modifiers = _modifiers.cycle(modifier));
    _notifyControls();
  }

  void _toggleFn() {
    setState(() => _fnOpen = !_fnOpen);
    _notifyControls();
  }

  void _handleTextInput() {
    if (_clearingInput) return;
    final value = _inputController.value;
    final edit = decodeTerminalSoftInput(value.text);
    if (edit.isIdle) return;
    if (!edit.backspace &&
        value.composing.isValid &&
        !value.composing.isCollapsed) {
      return;
    }
    _resetSoftKeyboardBuffer();
    if (edit.backspace) {
      _sendHid(_usbKeyboardPage | 0x2a);
      return;
    }
    if (edit.text.isEmpty) return;
    _sendSoftInputText(edit.text);
  }

  void _handleAndroidTerminalInput(AndroidTerminalInputEvent event) {
    if (!mounted ||
        !widget.active ||
        _keyboardFocusLocked ||
        event.owner != _androidInputOwner) {
      return;
    }
    switch (event) {
      case AndroidTerminalCompositionInput(:final text, :final active):
        _updateAndroidComposition(active: active, text: text);
      case AndroidTerminalTextInput(:final text):
        _clearAndroidComposition();
        _sendSoftInputText(text);
      case AndroidTerminalBackspaceInput(:final count):
        for (var index = 0; index < count; index += 1) {
          _sendHid(_usbKeyboardPage | 0x2a);
        }
      case AndroidTerminalKeyInput(
        :final hidUsage,
        :final modifiers,
        :final unshiftedCodepoint,
        :final text,
      ):
        if (hidUsage case final usage?) {
          _dispatchModifiedInput(
            (combinedModifiers) => widget.onInput(
              _TerminalInputOperation.key(
                hidUsage: usage,
                modifiers: combinedModifiers,
                unshiftedCodepoint: unshiftedCodepoint,
                text: text,
              ),
            ),
            additionalModifiers: modifiers,
          );
        } else if (text.isNotEmpty) {
          _sendText(text);
        }
    }
  }

  void _updateAndroidComposition({required bool active, required String text}) {
    final next = (active: active, text: active ? text : '');
    if (_androidComposition.value == next) return;
    _androidComposition.value = next;
  }

  void _clearAndroidComposition() =>
      _updateAndroidComposition(active: false, text: '');

  void _sendSoftInputText(String text) {
    final parts = text
        .replaceAll('\r\n', '\n')
        .replaceAll('\r', '\n')
        .split('\n');
    for (var index = 0; index < parts.length; index += 1) {
      final part = parts[index];
      if (part.isNotEmpty) _sendText(part);
      if (index < parts.length - 1) {
        _sendHid(_usbKeyboardPage | 0x28);
      }
    }
  }

  void _runQuickAction(TerminalQuickAction action) {
    switch (action.kind) {
      case TerminalQuickActionKind.key:
        final key = action.key;
        if (key == null) return;
        if (key.modifier case final modifier?) {
          _toggleModifier(modifier);
          return;
        }
        if (key.text case final text?) {
          _sendText(text);
          return;
        }
        if (key.hidUsage case final usage?) {
          _sendHid(
            usage,
            unshiftedCodepoint: key.unshiftedCodepoint,
            text: key.text ?? '',
          );
        }
      case TerminalQuickActionKind.chord:
        final key = action.key;
        final usage = key?.hidUsage;
        if (key == null || usage == null) return;
        _sendHid(
          usage,
          additionalModifiers: action.modifiers,
          unshiftedCodepoint: key.unshiftedCodepoint,
          text: key.text ?? '',
        );
      case TerminalQuickActionKind.text:
        _sendSoftInputText(action.text);
        if (action.sendEnter) _sendHid(_usbKeyboardPage | 0x28);
    }
  }

  void _resetSoftKeyboardBuffer() {
    _clearingInput = true;
    _inputController.value = const TextEditingValue(
      text: terminalSoftInputSentinel,
      selection: TextSelection.collapsed(
        offset: terminalSoftInputSentinel.length,
      ),
    );
    _clearingInput = false;
  }

  KeyEventResult _handleHardwareKey(FocusNode node, KeyEvent event) {
    if (!widget.active) return KeyEventResult.ignored;
    if (event is KeyUpEvent) return KeyEventResult.ignored;
    if (_isPhysicalModifier(event.logicalKey)) return KeyEventResult.ignored;
    final keyboard = HardwareKeyboard.instance;
    var physicalModifiers = 0;
    if (keyboard.isShiftPressed) physicalModifiers |= terminalModifierShiftBit;
    if (keyboard.isControlPressed) {
      physicalModifiers |= terminalModifierControlBit;
    }
    if (keyboard.isAltPressed) physicalModifiers |= terminalModifierAltBit;
    if (keyboard.isMetaPressed) physicalModifiers |= terminalModifierSuperBit;
    final modifiers = _modifiers.bits | physicalModifiers;
    final hasCommandModifier =
        modifiers &
            (terminalModifierControlBit |
                terminalModifierAltBit |
                terminalModifierSuperBit) !=
        0;
    if (event.character != null && !hasCommandModifier) {
      return KeyEventResult.ignored;
    }
    final character = event.character ?? '';
    _dispatchModifiedInput(
      (combinedModifiers) => widget.onInput(
        _TerminalInputOperation.key(
          hidUsage: event.physicalKey.usbHidUsage,
          modifiers: combinedModifiers,
          unshiftedCodepoint: character.runes.firstOrNull ?? 0,
          text: character,
        ),
      ),
      additionalModifiers: physicalModifiers,
    );
    return KeyEventResult.handled;
  }

  void _sendHid(
    int usage, {
    int additionalModifiers = 0,
    int unshiftedCodepoint = 0,
    String text = '',
  }) {
    _dispatchModifiedInput(
      (modifiers) => widget.onInput(
        _TerminalInputOperation.key(
          hidUsage: usage,
          modifiers: modifiers,
          unshiftedCodepoint: unshiftedCodepoint,
          text: text,
        ),
      ),
      additionalModifiers: additionalModifiers,
    );
  }

  void _sendEnter() {
    _resetSoftKeyboardBuffer();
    _sendHid(_usbKeyboardPage | 0x28);
  }

  void _sendText(String text) {
    _dispatchModifiedInput(
      (modifiers) =>
          widget.onInput(_TerminalInputOperation.text(text, modifiers)),
    );
  }

  void _dispatchModifiedInput(
    Future<void> Function(int modifiers) send, {
    int additionalModifiers = 0,
  }) {
    if (_readOnly) return;
    final sent = _modifiers;
    final expectedConnection = widget.connection;
    if (sent.hasOnce) {
      setState(() => _modifiers = sent.consumeOnce());
      _notifyControls();
    }
    try {
      _reportInput(
        send(sent.bits | additionalModifiers),
        rejectedModifiers: sent,
        expectedConnection: expectedConnection,
      );
    } catch (error) {
      if (mounted && widget.connection == expectedConnection) {
        setState(() => _modifiers = _modifiers.restoreRejected(sent));
        _notifyControls();
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  void _reportInput(
    Future<void> pending, {
    TerminalModifierState? rejectedModifiers,
    TerminalConnection? expectedConnection,
  }) {
    unawaited(
      pending.catchError((Object error) {
        if (!mounted) return;
        if (rejectedModifiers != null &&
            widget.connection == expectedConnection) {
          setState(() {
            _modifiers = _modifiers.restoreRejected(rejectedModifiers);
          });
          _notifyControls();
        }
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(error.toString())));
      }),
    );
  }
}

bool _isPhysicalModifier(LogicalKeyboardKey key) {
  return key == LogicalKeyboardKey.shift ||
      key == LogicalKeyboardKey.shiftLeft ||
      key == LogicalKeyboardKey.shiftRight ||
      key == LogicalKeyboardKey.control ||
      key == LogicalKeyboardKey.controlLeft ||
      key == LogicalKeyboardKey.controlRight ||
      key == LogicalKeyboardKey.alt ||
      key == LogicalKeyboardKey.altLeft ||
      key == LogicalKeyboardKey.altRight ||
      key == LogicalKeyboardKey.meta ||
      key == LogicalKeyboardKey.metaLeft ||
      key == LogicalKeyboardKey.metaRight;
}
