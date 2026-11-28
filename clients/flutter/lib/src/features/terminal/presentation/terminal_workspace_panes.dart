part of 'terminal_workspace_screen.dart';

enum _TerminalSplitTarget { left, right, above, below }

final class _TerminalInputOperation {
  const _TerminalInputOperation.text(this.text, this.modifiers)
    : hidUsage = null,
      unshiftedCodepoint = 0,
      paste = false,
      allowUnsafePaste = false;

  const _TerminalInputOperation.key({
    required this.hidUsage,
    required this.modifiers,
    this.unshiftedCodepoint = 0,
    this.text = '',
  }) : paste = false,
       allowUnsafePaste = false;

  const _TerminalInputOperation.paste(this.text, this.allowUnsafePaste)
    : hidUsage = null,
      modifiers = 0,
      unshiftedCodepoint = 0,
      paste = true;

  final int? hidUsage;
  final int modifiers;
  final int unshiftedCodepoint;
  final String text;
  final bool paste;
  final bool allowUnsafePaste;

  Future<void> send(TerminalConnection connection) {
    if (paste) {
      return connection.sendPaste(text, allowUnsafe: allowUnsafePaste);
    }
    final usage = hidUsage;
    if (usage != null) {
      return connection.sendKey(
        hidUsage: usage,
        modifiers: modifiers,
        unshiftedCodepoint: unshiftedCodepoint,
        text: text,
      );
    }
    return connection.sendText(text, modifiers: modifiers);
  }
}

final class _ActiveTerminal extends ConsumerStatefulWidget {
  const _ActiveTerminal({
    required this.endpointId,
    required this.endpointLabel,
    required this.terminalId,
    required this.keyboardVisualInset,
    required this.keyboardInset,
    required this.settledKeyboardInset,
    required this.surfaceKey,
    required this.onOpenPreview,
    required this.onOpenBrowser,
    required this.onActiveTerminalChanged,
  }) : super(key: surfaceKey);

  final String endpointId;
  final String endpointLabel;
  final String terminalId;
  final ValueListenable<double> keyboardVisualInset;
  final double keyboardInset;
  final double settledKeyboardInset;
  final GlobalKey<_ActiveTerminalState> surfaceKey;
  final VoidCallback? onOpenPreview;
  final Future<void> Function(Uri) onOpenBrowser;
  final ValueChanged<String> onActiveTerminalChanged;

  @override
  ConsumerState<_ActiveTerminal> createState() => _ActiveTerminalState();
}

final class _ActiveTerminalState extends ConsumerState<_ActiveTerminal> {
  static const _pinStore = TerminalPinStore();

  TerminalSplitNode _splitRoot = primaryTerminalPane;
  String _activePaneKey = primaryTerminalPaneKey;
  late String _primaryTerminalId;
  final Map<String, GlobalKey<_TerminalSurfaceState>> _surfaceKeys = {};
  var _splitSequence = 0;
  bool _syncInput = false;
  List<String> _pinnedIds = const [];

  @override
  void initState() {
    super.initState();
    _primaryTerminalId = widget.terminalId;
    unawaited(_loadPins());
  }

  @override
  void didUpdateWidget(_ActiveTerminal oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId ||
        oldWidget.terminalId != widget.terminalId) {
      _primaryTerminalId = widget.terminalId;
      _splitRoot = primaryTerminalPane;
      _activePaneKey = primaryTerminalPaneKey;
      _surfaceKeys.clear();
      _syncInput = false;
      unawaited(_loadPins());
    }
  }

  Future<void> _loadPins() async {
    try {
      final pinned = await _pinStore.load(widget.endpointId);
      if (mounted) setState(() => _pinnedIds = pinned);
    } catch (_) {
      // Pin order only affects which terminal is chosen for the next split.
    }
  }

  GlobalKey<_TerminalSurfaceState> _surfaceKey(String paneKey) {
    return _surfaceKeys.putIfAbsent(
      paneKey,
      () => GlobalKey<_TerminalSurfaceState>(),
    );
  }

  _TerminalSurfaceState? get _activeSurface =>
      _surfaceKeys[_activePaneKey]?.currentState;

  bool get _hasSplit => terminalPaneKeys(_splitRoot).length > 1;

  void splitActiveBelow() => _splitActive(_TerminalSplitTarget.below);

  bool handleBack() {
    if (_activeSurface?.handleBack() == true) return true;
    if (!_hasSplit) return false;
    _closeActiveSplit();
    return true;
  }

  void switchTerminal(String terminalId) {
    final paneKeys = terminalPaneKeys(_splitRoot);
    if (terminalId == _primaryTerminalId) {
      _activatePane(primaryTerminalPaneKey);
      return;
    }
    final existingPaneKey = terminalPaneKey(terminalId);
    if (paneKeys.contains(existingPaneKey)) {
      _activatePane(existingPaneKey);
      return;
    }
    if (isEmptyTerminalPaneKey(_activePaneKey)) {
      final emptyPaneKey = _activePaneKey;
      setState(() {
        _splitRoot = assignTerminalToPane(
          root: _splitRoot,
          targetPaneKey: emptyPaneKey,
          terminalId: terminalId,
        );
        _activePaneKey = existingPaneKey;
      });
      widget.onActiveTerminalChanged(terminalId);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _surfaceKeys[existingPaneKey]?.currentState?._showKeyboard();
        }
      });
      return;
    }
    setState(() {
      _primaryTerminalId = terminalId;
      _activePaneKey = primaryTerminalPaneKey;
    });
    widget.onActiveTerminalChanged(terminalId);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        _surfaceKeys[primaryTerminalPaneKey]?.currentState?._showKeyboard();
      }
    });
  }

  void _activatePane(String paneKey) {
    if (_activePaneKey == paneKey) return;
    HapticFeedback.selectionClick();
    setState(() => _activePaneKey = paneKey);
    final terminalId = terminalIdForPane(paneKey, _primaryTerminalId);
    if (terminalId != null) {
      widget.onActiveTerminalChanged(terminalId);
    } else {
      FocusManager.instance.primaryFocus?.unfocus();
    }
  }

  void _splitActive(_TerminalSplitTarget target) {
    final terminals = ref
        .read(terminalListProvider(widget.endpointId))
        .valueOrNull;
    if (terminals == null) return;
    final displayed = terminalPaneKeys(_splitRoot)
        .map((paneKey) => terminalIdForPane(paneKey, _primaryTerminalId))
        .whereType<String>()
        .toSet();
    if (!terminals.any(
      (terminal) => !displayed.contains(terminal.ref.terminalId),
    )) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No other terminal is available')),
      );
      return;
    }
    final horizontal =
        target == _TerminalSplitTarget.left ||
        target == _TerminalSplitTarget.right;
    final before =
        target == _TerminalSplitTarget.left ||
        target == _TerminalSplitTarget.above;
    _splitSequence += 1;
    final splitId = 'split-$_splitSequence';
    final paneKey = emptyTerminalPaneKey(splitId);
    setState(() {
      _splitRoot = splitTerminalPaneEmpty(
        root: _splitRoot,
        targetPaneKey: _activePaneKey,
        direction: horizontal
            ? TerminalSplitDirection.columns
            : TerminalSplitDirection.rows,
        splitId: splitId,
        placement: before
            ? TerminalSplitPlacement.before
            : TerminalSplitPlacement.after,
      );
      _activePaneKey = paneKey;
    });
    FocusManager.instance.primaryFocus?.unfocus();
  }

  Future<void> _chooseTerminalForPane(String paneKey) async {
    if (!isEmptyTerminalPaneKey(paneKey) ||
        !terminalPaneKeys(_splitRoot).contains(paneKey)) {
      return;
    }
    final terminals = ref
        .read(terminalListProvider(widget.endpointId))
        .valueOrNull;
    if (terminals == null) return;
    final displayed = terminalPaneKeys(_splitRoot)
        .map((key) => terminalIdForPane(key, _primaryTerminalId))
        .whereType<String>()
        .toSet();
    final available = _orderedSplitTerminals(terminals)
        .where((terminal) => !displayed.contains(terminal.ref.terminalId))
        .toList(growable: false);
    if (available.isEmpty) return;
    final selection = await showAnyttyTerminalSwitcher(
      context: context,
      endpoints: [
        TerminalSwitcherEndpoint(
          endpointId: widget.endpointId,
          label: widget.endpointLabel,
          current: true,
          terminals: available,
        ),
      ],
      loadTerminals: (_) async => available,
    );
    if (!mounted || selection == null) return;
    final terminalId = selection.terminalId;
    final newPaneKey = terminalPaneKey(terminalId);
    if (terminalPaneKeys(_splitRoot).contains(newPaneKey)) {
      _activatePane(newPaneKey);
      return;
    }
    setState(() {
      _splitRoot = assignTerminalToPane(
        root: _splitRoot,
        targetPaneKey: paneKey,
        terminalId: terminalId,
      );
      _activePaneKey = newPaneKey;
    });
    widget.onActiveTerminalChanged(terminalId);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        _surfaceKeys[newPaneKey]?.currentState?._showKeyboard();
      }
    });
  }

  List<TerminalInfo> _orderedSplitTerminals(List<TerminalInfo> terminals) {
    final running = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.running,
      ),
      _pinnedIds,
    );
    final exited = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.exited,
      ),
      _pinnedIds,
    );
    final known = {...running, ...exited};
    return [
      ...running,
      ...exited,
      ...terminals.where((terminal) => !known.contains(terminal)),
    ];
  }

  void _closeActiveSplit() {
    if (!_hasSplit) return;
    var paneKey = _activePaneKey;
    if (paneKey == primaryTerminalPaneKey) {
      paneKey = terminalPaneKeys(_splitRoot)
          .lastWhere((candidate) => candidate != primaryTerminalPaneKey);
    }
    setState(() {
      _splitRoot =
          removeTerminalPane(_splitRoot, paneKey) ?? primaryTerminalPane;
      _surfaceKeys.remove(paneKey);
      _activePaneKey = primaryTerminalPaneKey;
      _syncInput = false;
    });
    widget.onActiveTerminalChanged(_primaryTerminalId);
  }

  Future<void> _sendInput(_TerminalInputOperation operation) async {
    final paneKeys = _syncInput && _hasSplit
        ? terminalPaneKeys(_splitRoot)
        : [_activePaneKey];
    final targets = paneKeys
        .map((paneKey) => _surfaceKeys[paneKey]?.currentState)
        .whereType<_TerminalSurfaceState>()
        .toList(growable: false);
    if (targets.isEmpty) {
      throw const NativeSessionException('The active terminal is not ready');
    }
    await dispatchTerminalInput(
      targets,
      (target) => operation.send(target.widget.connection),
    );
  }

  void _removeUnavailablePanes(List<TerminalInfo> terminals) {
    final available = terminals
        .map((terminal) => terminal.ref.terminalId)
        .toSet();
    final invalid = terminalPaneKeys(_splitRoot)
        .where((paneKey) {
          if (paneKey == primaryTerminalPaneKey) return false;
          if (isEmptyTerminalPaneKey(paneKey)) return false;
          final terminalId = terminalIdForPane(paneKey, _primaryTerminalId);
          return terminalId == _primaryTerminalId ||
              !available.contains(terminalId);
        })
        .toList(growable: false);
    if (invalid.isEmpty) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      setState(() {
        for (final paneKey in invalid) {
          _splitRoot =
              removeTerminalPane(_splitRoot, paneKey) ?? primaryTerminalPane;
          _surfaceKeys.remove(paneKey);
          if (_activePaneKey == paneKey) {
            _activePaneKey = primaryTerminalPaneKey;
          }
        }
        _syncInput = false;
      });
      final activeTerminalId = terminalIdForPane(
        _activePaneKey,
        _primaryTerminalId,
      );
      if (activeTerminalId != null) {
        widget.onActiveTerminalChanged(activeTerminalId);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final terminalsAsync = ref.watch(terminalListProvider(widget.endpointId));
    final terminals = terminalsAsync.valueOrNull ?? const <TerminalInfo>[];
    _removeUnavailablePanes(terminals);
    final byId = {
      for (final terminal in terminals) terminal.ref.terminalId: terminal,
    };
    final keyboardFocusLayout =
        _hasSplit &&
        widget.keyboardInset > 100 &&
        MediaQuery.orientationOf(context) == Orientation.landscape;
    return TerminalPetalMenuOverlay(
      child: Stack(
        fit: StackFit.expand,
        children: [
          TerminalKeyboardWorkspace(
            visualInset: widget.keyboardVisualInset,
            translateVisualInset: true,
            child: Column(
              children: [
                Expanded(
                  child: Stack(
                    fit: StackFit.expand,
                    children: [
                      if (keyboardFocusLayout)
                        _buildKeyboardFocusedPanes(byId, widget.keyboardInset)
                      else
                        _buildSplitNode(_splitRoot, byId, widget.keyboardInset),
                      if (_activeSurface case final surface?)
                        ValueListenableBuilder<int>(
                          valueListenable: surface.controlRevision,
                          builder: (context, _, _) => Stack(
                            fit: StackFit.expand,
                            children: surface.buildExternalOverlays(),
                          ),
                        ),
                    ],
                  ),
                ),
                if (_activeSurface case final surface?)
                  ValueListenableBuilder<int>(
                    valueListenable: surface.controlRevision,
                    builder: (context, _, _) => surface.buildExternalKeyBar(),
                  ),
              ],
            ),
          ),
          if (_activeSurface case final surface?)
            ValueListenableBuilder<int>(
              valueListenable: surface.controlRevision,
              builder: (context, _, _) => Stack(
                fit: StackFit.expand,
                children: surface.buildHeaderOverlays(),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildSplitNode(
    TerminalSplitNode node,
    Map<String, TerminalInfo> terminals,
    double keyboardInset,
  ) {
    return switch (node) {
      TerminalPaneNode() => _buildTerminalPane(
        node.paneKey,
        terminals,
        keyboardInset: keyboardInset,
      ),
      TerminalSplitBranch() => LayoutBuilder(
        builder: (context, constraints) {
          final columns = node.direction == TerminalSplitDirection.columns;
          final extent = columns ? constraints.maxWidth : constraints.maxHeight;
          final divider = extent * node.ratio / 100;
          final first = columns
              ? Positioned(
                  left: 0,
                  top: 0,
                  bottom: 0,
                  width: math.max(0, divider - 0.5),
                  child: _buildSplitNode(node.first, terminals, keyboardInset),
                )
              : Positioned(
                  left: 0,
                  top: 0,
                  right: 0,
                  height: math.max(0, divider - 0.5),
                  child: _buildSplitNode(node.first, terminals, keyboardInset),
                );
          final second = columns
              ? Positioned(
                  left: divider + 0.5,
                  top: 0,
                  right: 0,
                  bottom: 0,
                  child: _buildSplitNode(node.second, terminals, keyboardInset),
                )
              : Positioned(
                  left: 0,
                  top: divider + 0.5,
                  right: 0,
                  bottom: 0,
                  child: _buildSplitNode(node.second, terminals, keyboardInset),
                );
          return Stack(
            fit: StackFit.expand,
            clipBehavior: Clip.hardEdge,
            children: [
              first,
              second,
              Positioned(
                left: columns ? divider - 10 : 0,
                top: columns ? 0 : divider - 10,
                right: columns ? null : 0,
                bottom: columns ? 0 : null,
                width: columns ? 20 : null,
                height: columns ? null : 20,
                child: _TerminalSplitDivider(
                  direction: node.direction,
                  onDrag: (delta) {
                    if (extent <= 0) return;
                    setState(() {
                      _splitRoot = updateTerminalSplitRatio(
                        _splitRoot,
                        node.id,
                        node.ratio + delta / extent * 100,
                      );
                    });
                  },
                ),
              ),
            ],
          );
        },
      ),
    };
  }

  Widget _buildKeyboardFocusedPanes(
    Map<String, TerminalInfo> terminals,
    double keyboardInset,
  ) {
    return Stack(
      fit: StackFit.expand,
      children: [
        for (final paneKey in terminalPaneKeys(_splitRoot))
          Positioned.fill(
            child: TickerMode(
              enabled: paneKey == _activePaneKey,
              child: Offstage(
                offstage: paneKey != _activePaneKey,
                child: _buildTerminalPane(
                  paneKey,
                  terminals,
                  keyboardInset: keyboardInset,
                  showPaneHeader: false,
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildTerminalPane(
    String paneKey,
    Map<String, TerminalInfo> terminals, {
    required double keyboardInset,
    bool showPaneHeader = true,
  }) {
    final terminalId = terminalIdForPane(paneKey, _primaryTerminalId);
    final terminal = terminalId == null ? null : terminals[terminalId];
    final active = paneKey == _activePaneKey;
    return GestureDetector(
      behavior: HitTestBehavior.translucent,
      excludeFromSemantics: true,
      onTapDown: (_) => _activatePane(paneKey),
      child: Stack(
        fit: StackFit.expand,
        children: [
          ColoredBox(
            color: AnyttyPalette.of(context).background,
            child: Column(
              children: [
                if (_hasSplit && showPaneHeader)
                  _TerminalPaneHeader(terminal: terminal, active: active),
                Expanded(
                  child: isEmptyTerminalPaneKey(paneKey)
                      ? _EmptyTerminalPane(
                          active: active,
                          onChoose: () => _chooseTerminalForPane(paneKey),
                        )
                      : terminal == null || terminalId == null
                      ? const _WaitingForSnapshot(label: 'Opening terminal')
                      : _TerminalPaneConnection(
                          endpointId: widget.endpointId,
                          endpointLabel: widget.endpointLabel,
                          terminal: terminal,
                          keyboardVisualInset: widget.keyboardVisualInset,
                          keyboardInset: keyboardInset,
                          settledKeyboardInset: widget.settledKeyboardInset,
                          surfaceKey: _surfaceKey(paneKey),
                          onOpenPreview: widget.onOpenPreview,
                          active: active,
                          onActivate: () => _activatePane(paneKey),
                          onOpenBrowser: widget.onOpenBrowser,
                          externalControls: true,
                          onSurfaceReady: () {
                            if (mounted) setState(() {});
                          },
                          onInput: _sendInput,
                          splitOpen: _hasSplit,
                          syncInput: _syncInput,
                          canSplit:
                              terminalPaneKeys(_splitRoot).length <
                              terminals.length,
                          onSplit: _splitActive,
                          onToggleSync: () =>
                              setState(() => _syncInput = !_syncInput),
                          onCloseSplit: _closeActiveSplit,
                        ),
                ),
              ],
            ),
          ),
          if (active && _hasSplit)
            IgnorePointer(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  border: Border.fromBorderSide(
                    BorderSide(color: AnyttyPalette.of(context).accent),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

final class _TerminalPaneConnection extends ConsumerWidget {
  const _TerminalPaneConnection({
    required this.endpointId,
    required this.endpointLabel,
    required this.terminal,
    required this.keyboardVisualInset,
    required this.keyboardInset,
    required this.settledKeyboardInset,
    required this.surfaceKey,
    required this.onOpenPreview,
    required this.active,
    required this.onActivate,
    required this.onOpenBrowser,
    required this.externalControls,
    required this.onSurfaceReady,
    required this.onInput,
    required this.splitOpen,
    required this.syncInput,
    required this.canSplit,
    required this.onSplit,
    required this.onToggleSync,
    required this.onCloseSplit,
  });

  final String endpointId;
  final String endpointLabel;
  final TerminalInfo terminal;
  final ValueListenable<double> keyboardVisualInset;
  final double keyboardInset;
  final double settledKeyboardInset;
  final GlobalKey<_TerminalSurfaceState> surfaceKey;
  final VoidCallback? onOpenPreview;
  final bool active;
  final VoidCallback onActivate;
  final Future<void> Function(Uri) onOpenBrowser;
  final bool externalControls;
  final VoidCallback onSurfaceReady;
  final Future<void> Function(_TerminalInputOperation) onInput;
  final bool splitOpen;
  final bool syncInput;
  final bool canSplit;
  final ValueChanged<_TerminalSplitTarget> onSplit;
  final VoidCallback onToggleSync;
  final VoidCallback onCloseSplit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final terminalId = terminal.ref.terminalId;
    final key = (endpointId: endpointId, terminalId: terminalId);
    final settings =
        ref.watch(terminalSettingsProvider).valueOrNull ??
        defaultTerminalSettings;
    final keyboardMode =
        ref.watch(terminalKeyboardModeProvider(key)).valueOrNull ??
        settings.keyboardMode;
    return ref
        .watch(terminalConnectionProvider(key))
        .when(
          loading: () => const _WaitingForSnapshot(label: 'Opening terminal'),
          error: (error, _) => _TerminalFailure(
            dark: true,
            message: error.toString(),
            onRetry: () => ref.invalidate(terminalConnectionProvider(key)),
          ),
          data: (connection) => _TerminalSurface(
            key: surfaceKey,
            connection: connection,
            onOpenPreview: onOpenPreview,
            keyboardMode: keyboardMode,
            keyboardVisualInset: keyboardVisualInset,
            keyboardInset: keyboardInset,
            settledKeyboardInset: settledKeyboardInset,
            settings: settings,
            active: active,
            onActivate: onActivate,
            externalControls: externalControls,
            manageKeyboardInset: false,
            onSurfaceReady: onSurfaceReady,
            onInput: onInput,
            onOpenBrowser: onOpenBrowser,
            onOpenFiles: () => showAnyttyFileManager(
              context: context,
              endpointId: endpointId,
              endpointLabel: endpointLabel,
              initialPath: terminal.liveCwd.isNotEmpty
                  ? terminal.liveCwd
                  : terminal.cwd,
            ),
            onShowResources: () =>
                showTerminalResourceDetails(context, terminal),
            splitOpen: splitOpen,
            syncInput: syncInput,
            canSplit: canSplit,
            onSplit: onSplit,
            onToggleSync: onToggleSync,
            onCloseSplit: onCloseSplit,
            onSettingsChanged: (next) =>
                ref.read(terminalSettingsProvider.notifier).save(next),
            onKeyboardModeChanged: (next) async {
              await const TerminalKeyboardModeStore().save(
                endpointId: endpointId,
                terminalId: terminalId,
                mode: next,
              );
              ref.invalidate(terminalKeyboardModeProvider(key));
            },
            onReconnect: () => ref.invalidate(terminalConnectionProvider(key)),
          ),
        );
  }
}

final class _EmptyTerminalPane extends StatelessWidget {
  const _EmptyTerminalPane({required this.active, required this.onChoose});

  final bool active;
  final VoidCallback onChoose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 230),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  LucideIcons.panelTopOpen,
                  size: 28,
                  color: active ? palette.accent : palette.muted,
                ),
                const SizedBox(height: 12),
                Text(
                  anyttyText(
                    context,
                    en: 'Choose a terminal for this pane',
                    zh: '为这个窗格选择终端',
                  ),
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: palette.text,
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 14),
                FilledButton.icon(
                  key: const ValueKey('choose-split-terminal'),
                  onPressed: onChoose,
                  icon: const Icon(LucideIcons.terminal, size: 16),
                  label: Text(
                    anyttyText(context, en: 'Choose terminal', zh: '选择终端'),
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

final class _TerminalPaneHeader extends StatelessWidget {
  const _TerminalPaneHeader({required this.terminal, required this.active});

  final TerminalInfo? terminal;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final title = terminal == null
        ? 'Terminal'
        : _terminalDisplayName(terminal!);
    final cwd = terminal == null
        ? ''
        : terminal!.liveCwd.isNotEmpty
        ? terminal!.liveCwd
        : terminal!.cwd;
    return Container(
      height: 28,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          Icon(
            Icons.terminal_rounded,
            size: 14,
            color: active ? palette.text : palette.faint,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: active ? palette.text : palette.muted,
                fontSize: 11,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
          if (cwd.isNotEmpty) ...[
            const SizedBox(width: 8),
            Flexible(
              child: Text(
                cwd,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: palette.faint,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 10,
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

final class _TerminalSplitDivider extends StatelessWidget {
  const _TerminalSplitDivider({required this.direction, required this.onDrag});

  final TerminalSplitDirection direction;
  final ValueChanged<double> onDrag;

  @override
  Widget build(BuildContext context) {
    final columns = direction == TerminalSplitDirection.columns;
    return MouseRegion(
      cursor: columns
          ? SystemMouseCursors.resizeColumn
          : SystemMouseCursors.resizeRow,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onHorizontalDragUpdate: columns
            ? (details) => onDrag(details.delta.dx)
            : null,
        onVerticalDragUpdate: columns
            ? null
            : (details) => onDrag(details.delta.dy),
        child: Center(
          child: ColoredBox(
            color: AnyttyPalette.of(context).borderStrong,
            child: SizedBox(
              width: columns ? 1 : double.infinity,
              height: columns ? double.infinity : 1,
            ),
          ),
        ),
      ),
    );
  }
}
