part of 'terminal_canvas.dart';

final class TerminalHistoryHighlights extends ChangeNotifier {
  HistorySelection? _selection;
  HistoryRange? _selectionRange;
  HistoryRange? _searchMatch;

  HistorySelection? get selection => _selection;
  HistoryRange? get selectionRange => _selectionRange;
  HistoryRange? get searchMatch => _searchMatch;

  void update({
    required HistorySelection? selection,
    required HistoryLayout? layout,
    required HistoryRange? searchMatch,
  }) {
    final selectionRange = selection == null || layout == null
        ? null
        : normalizeHistorySelection(layout: layout, selection: selection);
    if (_sameSelection(_selection, selection) &&
        _sameRange(_selectionRange, selectionRange) &&
        _sameRange(_searchMatch, searchMatch)) {
      return;
    }
    _selection = selection;
    _selectionRange = selectionRange;
    _searchMatch = searchMatch;
    notifyListeners();
  }

  void clear() => update(selection: null, layout: null, searchMatch: null);

  bool _sameSelection(HistorySelection? left, HistorySelection? right) {
    if (identical(left, right)) return true;
    if (left == null || right == null) return false;
    return left.anchor.lineId == right.anchor.lineId &&
        left.anchor.column == right.anchor.column &&
        left.focus.lineId == right.focus.lineId &&
        left.focus.column == right.focus.column;
  }

  bool _sameRange(HistoryRange? left, HistoryRange? right) {
    if (identical(left, right)) return true;
    if (left == null || right == null) return false;
    return sameHistoryRange(left, right);
  }
}

final class TerminalHistoryCanvas extends StatefulWidget {
  const TerminalHistoryCanvas({
    super.key,
    required this.rows,
    required this.cols,
    required this.scrollController,
    required this.layout,
    this.settings = defaultTerminalSettings,
    this.highlights,
    this.selection,
    this.searchMatch,
    this.searchMatches = const [],
    this.trailingRows = 0,
    this.selectionEnabled = false,
    this.canLoadOlder = false,
    this.onSelectionChanged,
    this.onLinkTap,
  });

  final List<HistoryRow> rows;
  final int cols;
  final ScrollController scrollController;
  final HistoryLayout layout;
  final TerminalSettings settings;
  final TerminalHistoryHighlights? highlights;
  final HistorySelection? selection;
  final HistoryRange? searchMatch;
  final List<HistoryRange> searchMatches;
  final int trailingRows;
  final bool selectionEnabled;
  final bool canLoadOlder;
  final ValueChanged<HistorySelection>? onSelectionChanged;
  final ValueChanged<TerminalLink>? onLinkTap;

  @override
  State<TerminalHistoryCanvas> createState() => _TerminalHistoryCanvasState();
}

final class TerminalHistoryPresentation extends StatelessWidget {
  const TerminalHistoryPresentation({
    super.key,
    required this.ready,
    required this.fallback,
    required this.child,
    this.fallbackInteractive = false,
  });

  final bool ready;
  final Widget fallback;
  final Widget child;
  final bool fallbackInteractive;

  @override
  Widget build(BuildContext context) {
    return Stack(
      fit: StackFit.expand,
      children: [
        if (!ready)
          Positioned.fill(
            key: const ValueKey('terminal-history-fallback-layer'),
            child: IgnorePointer(
              ignoring: !fallbackInteractive,
              child: ExcludeSemantics(
                excluding: !fallbackInteractive,
                child: fallback,
              ),
            ),
          ),
        Positioned.fill(
          key: const ValueKey('terminal-history-content-layer'),
          child: IgnorePointer(
            ignoring: !ready,
            child: Opacity(opacity: ready ? 1 : 0, child: child),
          ),
        ),
      ],
    );
  }
}

final class _TerminalHistoryCanvasState extends State<TerminalHistoryCanvas> {
  final ScrollController _horizontalScroll = ScrollController();
  HistoryCellPoint? _dragAnchor;
  bool _selectionGestureScrolled = false;
  Timer? _selectionAutoScrollTimer;
  final ValueNotifier<Offset?> _magnifierPosition = ValueNotifier(null);
  Offset? _selectionDragPosition;
  Size? _selectionViewport;
  TerminalCellMetrics? _selectionMetrics;
  HistoryLayout? _indexedSearchLayout;
  List<HistoryRange>? _indexedSearchMatches;
  Map<int, List<HistoryRowRange>> _searchMatchesByRow = const {};
  List<HistoryRow>? _measuredRows;
  int? _measuredBaseCols;
  int _measuredContentCols = 0;

  @override
  void didUpdateWidget(TerminalHistoryCanvas oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.selectionEnabled && !widget.selectionEnabled) {
      _resetSelectionGesture();
    }
  }

  @override
  void dispose() {
    _selectionAutoScrollTimer?.cancel();
    _magnifierPosition.dispose();
    _horizontalScroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final metrics = widget.settings.metrics;
    final theme = widget.settings.theme;
    final contentCols = _contentCols();
    final searchMatchesByRow = _indexedMatchesByRow();
    final fallbackSelectionRange = widget.selection == null
        ? null
        : normalizeHistorySelection(
            layout: widget.layout,
            selection: widget.selection!,
          );
    return LayoutBuilder(
      builder: (context, constraints) => ColoredBox(
        color: _color(theme.background),
        child: Stack(
          fit: StackFit.expand,
          children: [
            GestureDetector(
              behavior: HitTestBehavior.opaque,
              onScaleStart: widget.selectionEnabled
                  ? (details) => _startSelectionGesture(
                      details,
                      constraints.biggest,
                      metrics,
                    )
                  : null,
              onScaleUpdate: widget.selectionEnabled
                  ? (details) => _updateSelectionGesture(
                      details,
                      constraints.biggest,
                      metrics,
                    )
                  : null,
              onScaleEnd: widget.selectionEnabled
                  ? (_) => _resetSelectionGesture()
                  : null,
              onTapUp: widget.selectionEnabled
                  ? (details) => _tapSelection(
                      details.localPosition,
                      constraints.biggest,
                      metrics,
                    )
                  : _openLinkAt,
              child: IgnorePointer(
                ignoring: widget.selectionEnabled,
                child: SingleChildScrollView(
                  controller: _horizontalScroll,
                  scrollDirection: Axis.horizontal,
                  child: SizedBox(
                    width: contentCols * metrics.cellWidth,
                    child: ListView.builder(
                      key: const ValueKey('terminal-history-canvas'),
                      controller: widget.scrollController,
                      itemExtent: metrics.rowHeight,
                      itemCount: widget.rows.length + widget.trailingRows,
                      itemBuilder: (context, index) {
                        if (index >= widget.rows.length) {
                          return ColoredBox(color: _color(theme.background));
                        }
                        Widget buildRow(
                          HistoryRange? selectionRange,
                          HistoryRange? searchMatch,
                        ) {
                          final historyRow = widget.rows[index];
                          final selected = selectionRange == null
                              ? null
                              : projectHistoryRangeToRow(
                                  layout: widget.layout,
                                  rowIndex: index,
                                  range: selectionRange,
                                );
                          final currentSearchMatch = searchMatch == null
                              ? null
                              : projectHistoryRangeToRow(
                                  layout: widget.layout,
                                  rowIndex: index,
                                  range: searchMatch,
                                );
                          final searchMatches =
                              searchMatchesByRow[index] ??
                              const <HistoryRowRange>[];
                          return Semantics(
                            key: ValueKey('terminal-history-line-$index'),
                            container: true,
                            readOnly: true,
                            selected: selected != null,
                            label: 'History line ${index + 1}',
                            value: terminalRowSemanticText(historyRow.row),
                            child: CustomPaint(
                              painter: _HistoryRowPainter(
                                row: historyRow.row,
                                cols: historyRow.fixedGrid
                                    ? math.max(
                                        widget.cols,
                                        math.max(
                                          historyRow.screenCols,
                                          historyScreenRowWidth(historyRow.row),
                                        ),
                                      )
                                    : widget.cols,
                                metrics: metrics,
                                theme: theme,
                                fontFamily: widget.settings.fontFamily,
                                selection: selected,
                                searchMatch: currentSearchMatch,
                                searchMatches: searchMatches,
                              ),
                            ),
                          );
                        }

                        final highlights = widget.highlights;
                        if (highlights == null) {
                          return buildRow(
                            fallbackSelectionRange,
                            widget.searchMatch,
                          );
                        }
                        return ListenableBuilder(
                          listenable: highlights,
                          builder: (context, _) => buildRow(
                            highlights.selectionRange,
                            highlights.searchMatch,
                          ),
                        );
                      },
                    ),
                  ),
                ),
              ),
            ),
            ValueListenableBuilder<Offset?>(
              valueListenable: _magnifierPosition,
              builder: (context, position, _) =>
                  _buildMagnifier(position, constraints.biggest),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMagnifier(Offset? position, Size viewport) {
    if (position == null || viewport.width < 32 || viewport.height < 100) {
      return const SizedBox.shrink();
    }
    const margin = 8.0;
    const gap = 36.0;
    final size = Size(math.min(168, viewport.width - margin * 2), 64);
    final focus = Offset(
      position.dx.clamp(0, viewport.width - 1),
      position.dy.clamp(0, viewport.height - 1),
    );
    final left = (focus.dx - size.width / 2).clamp(
      margin,
      viewport.width - size.width - margin,
    );
    final above = focus.dy - gap - size.height;
    final top = (above >= margin ? above : focus.dy + gap).clamp(
      margin,
      viewport.height - size.height - margin,
    );
    final center = Offset(left + size.width / 2, top + size.height / 2);
    return Positioned(
      left: left,
      top: top,
      child: IgnorePointer(
        child: ExcludeSemantics(
          child: RawMagnifier(
            key: const ValueKey('terminal-selection-magnifier'),
            size: size,
            magnificationScale: 1.8,
            // Sample the already painted canvas, including selection and scroll
            // updates. Keep the source under the finger even at clamped edges.
            focalPointOffset: focus - center,
            decoration: MagnifierDecoration(
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(14),
                side: BorderSide(
                  color: _color(widget.settings.theme.foreground),
                ),
              ),
              shadows: const [
                BoxShadow(
                  color: Colors.black38,
                  blurRadius: 8,
                  blurStyle: BlurStyle.outer,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  int _contentCols() {
    if (identical(_measuredRows, widget.rows) &&
        _measuredBaseCols == widget.cols) {
      return _measuredContentCols;
    }
    var contentCols = widget.cols;
    for (final row in widget.rows) {
      if (!row.fixedGrid) continue;
      contentCols = math.max(
        contentCols,
        math.max(row.screenCols, historyScreenRowWidth(row.row)),
      );
    }
    _measuredRows = widget.rows;
    _measuredBaseCols = widget.cols;
    _measuredContentCols = contentCols;
    return contentCols;
  }

  Map<int, List<HistoryRowRange>> _indexedMatchesByRow() {
    if (identical(_indexedSearchLayout, widget.layout) &&
        identical(_indexedSearchMatches, widget.searchMatches)) {
      return _searchMatchesByRow;
    }
    _indexedSearchLayout = widget.layout;
    _indexedSearchMatches = widget.searchMatches;
    _searchMatchesByRow = projectHistoryRangesToRows(
      layout: widget.layout,
      ranges: widget.searchMatches,
    );
    return _searchMatchesByRow;
  }

  void _startSelection(
    Offset position,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    final point = _pointAt(position, viewport, metrics, extendCell: false);
    final focus = _pointAt(position, viewport, metrics, extendCell: true);
    final anchor =
        widget.highlights?.selection?.anchor ??
        widget.selection?.anchor ??
        point;
    _dragAnchor = anchor;
    widget.onSelectionChanged?.call(
      HistorySelection(anchor: anchor, focus: focus),
    );
  }

  void _startSelectionGesture(
    ScaleStartDetails details,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    _selectionGestureScrolled = details.pointerCount >= 2;
    if (_selectionGestureScrolled) return;
    _startSelection(details.localFocalPoint, viewport, metrics);
    _magnifierPosition.value = details.localFocalPoint;
  }

  void _updateSelectionGesture(
    ScaleUpdateDetails details,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    if (details.pointerCount >= 2 || _selectionGestureScrolled) {
      _selectionGestureScrolled = true;
      _magnifierPosition.value = null;
      _dragAnchor = null;
      _stopSelectionAutoScroll();
      _scrollSelectionViewport(-details.focalPointDelta.dy);
      return;
    }
    _updateSelection(details.localFocalPoint, viewport, metrics);
  }

  void _tapSelection(
    Offset position,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    _startSelection(position, viewport, metrics);
    _resetSelectionGesture();
  }

  void _resetSelectionGesture() {
    _magnifierPosition.value = null;
    _dragAnchor = null;
    _selectionGestureScrolled = false;
    _stopSelectionAutoScroll();
  }

  void _updateSelection(
    Offset position,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    final anchor = _dragAnchor;
    if (anchor == null) return;
    _magnifierPosition.value = position;
    _selectionDragPosition = position;
    _selectionViewport = viewport;
    _selectionMetrics = metrics;
    final shouldContinue = _autoScroll(position, viewport, metrics);
    _setSelectionAutoScrollEnabled(shouldContinue);
    _emitSelection(anchor, position, viewport, metrics);
  }

  void _emitSelection(
    HistoryCellPoint anchor,
    Offset position,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    widget.onSelectionChanged?.call(
      HistorySelection(
        anchor: anchor,
        focus: _pointAt(position, viewport, metrics, extendCell: true),
      ),
    );
  }

  HistoryCellPoint _pointAt(
    Offset position,
    Size viewport,
    TerminalCellMetrics metrics, {
    required bool extendCell,
  }) {
    final verticalOffset = widget.scrollController.hasClients
        ? widget.scrollController.offset
        : 0.0;
    final horizontalOffset = _horizontalScroll.hasClients
        ? _horizontalScroll.offset
        : 0.0;
    final row =
        ((position.dy.clamp(0, viewport.height - 1) + verticalOffset) /
                metrics.rowHeight)
            .floor();
    final column =
        ((position.dx.clamp(0, viewport.width - 1) + horizontalOffset) /
                metrics.cellWidth)
            .floor() +
        (extendCell ? 1 : 0);
    return historyPointForRowColumn(
      layout: widget.layout,
      rowIndex: row,
      localColumn: column,
    );
  }

  bool _autoScroll(
    Offset position,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    if (!widget.scrollController.hasClients) return false;
    final edge = math.min(
      viewport.height / 3,
      math.max(36.0, metrics.rowHeight * 2.5),
    );
    var delta = 0.0;
    if (position.dy < edge) {
      final intensity = ((edge - position.dy) / edge).clamp(0.0, 1.0);
      delta = -metrics.rowHeight * (1 + (intensity * 2).floor());
    } else if (position.dy > viewport.height - edge) {
      final intensity = ((position.dy - (viewport.height - edge)) / edge).clamp(
        0.0,
        1.0,
      );
      delta = metrics.rowHeight * (1 + (intensity * 2).floor());
    }
    if (delta == 0) return false;
    final target = (widget.scrollController.offset + delta).clamp(
      0.0,
      widget.scrollController.position.maxScrollExtent,
    );
    if (target == widget.scrollController.offset) {
      return delta < 0 && widget.canLoadOlder;
    }
    widget.scrollController.jumpTo(target);
    return true;
  }

  void _setSelectionAutoScrollEnabled(bool enabled) {
    if (!enabled) {
      _selectionAutoScrollTimer?.cancel();
      _selectionAutoScrollTimer = null;
      return;
    }
    _selectionAutoScrollTimer ??= Timer.periodic(
      const Duration(milliseconds: 60),
      (_) {
        final anchor = _dragAnchor;
        final position = _selectionDragPosition;
        final viewport = _selectionViewport;
        final metrics = _selectionMetrics;
        if (anchor == null ||
            position == null ||
            viewport == null ||
            metrics == null ||
            !_autoScroll(position, viewport, metrics)) {
          _setSelectionAutoScrollEnabled(false);
          return;
        }
        _emitSelection(anchor, position, viewport, metrics);
      },
    );
  }

  void _stopSelectionAutoScroll() {
    _selectionAutoScrollTimer?.cancel();
    _selectionAutoScrollTimer = null;
    _selectionDragPosition = null;
    _selectionViewport = null;
    _selectionMetrics = null;
  }

  void _scrollSelectionViewport(double delta) {
    if (!widget.scrollController.hasClients || delta == 0) return;
    final position = widget.scrollController.position;
    widget.scrollController.jumpTo(
      (position.pixels + delta).clamp(0.0, position.maxScrollExtent),
    );
  }

  void _openLinkAt(TapUpDetails details) {
    if (widget.onLinkTap == null) return;
    final metrics = widget.settings.metrics;
    final verticalOffset = widget.scrollController.hasClients
        ? widget.scrollController.offset
        : 0.0;
    final horizontalOffset = _horizontalScroll.hasClients
        ? _horizontalScroll.offset
        : 0.0;
    final row =
        ((details.localPosition.dy + verticalOffset) / metrics.rowHeight)
            .floor();
    final column =
        ((details.localPosition.dx + horizontalOffset) / metrics.cellWidth)
            .floor();
    if (row < 0 || row >= widget.rows.length) return;
    final screenRow = widget.rows[row].row;
    final link = terminalLinkAtColumn(screenRow, column);
    if (link != null && widget.onLinkTap != null) {
      widget.onLinkTap!(link);
    }
  }
}
