import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../generated/proto/apipb/history.pb.dart';
import '../domain/history_interaction.dart';
import '../domain/live_screen_store.dart';
import '../domain/terminal_links.dart';
import '../domain/terminal_metrics.dart';
import '../domain/terminal_settings.dart';

const terminalFontSize = 14.0;
const terminalCellWidth = 8.44;
const terminalRowHeight = 20.0;
const _terminalSemanticMaxLines = 32;
const _terminalSemanticMaxCharacters = 4096;
const _terminalSemanticOmission = 'Earlier visible output omitted.\n';
const terminalFontFamilyFallback = <String>[
  'JetBrainsMonoNerd',
  'Menlo',
  'Roboto Mono',
  'monospace',
];

String terminalRowSemanticText(ScreenRow row, {int maxCharacters = 512}) {
  if (maxCharacters <= 0) return '';
  final output = StringBuffer();
  for (final cell in row.cells) {
    if (cell.content.isNotEmpty) {
      output.write(cell.content);
    } else if (cell.width > 0) {
      output.write(''.padRight(cell.width.clamp(0, maxCharacters)));
    }
    if (output.length >= maxCharacters * 2) break;
  }
  final trimmed = output.toString().trimRight();
  final characters = trimmed.characters;
  if (characters.length <= maxCharacters) return trimmed;
  return '${characters.take(maxCharacters - 1)}\u2026';
}

String terminalScreenSemanticValue(
  CanonicalLiveScreen screen, {
  int maxLines = _terminalSemanticMaxLines,
  int maxCharacters = _terminalSemanticMaxCharacters,
}) {
  if (maxLines <= 0 || maxCharacters <= 0) return '';
  final lines = screen.screenRows
      .map(terminalRowSemanticText)
      .toList(growable: true);
  while (lines.isNotEmpty && lines.last.isEmpty) {
    lines.removeLast();
  }
  if (lines.isEmpty) return 'No visible output';

  var omitted = false;
  if (lines.length > maxLines) {
    lines.removeRange(0, lines.length - maxLines);
    omitted = true;
  }
  var output = lines.join('\n');
  final outputCharacters = output.characters;
  final omissionLength = _terminalSemanticOmission.characters.length;
  if (outputCharacters.length + (omitted ? omissionLength : 0) >
      maxCharacters) {
    omitted = true;
    if (maxCharacters <= omissionLength) {
      return outputCharacters
          .skip(math.max(0, outputCharacters.length - maxCharacters))
          .toString();
    }
    final budget = (maxCharacters - omissionLength).clamp(0, maxCharacters);
    output = outputCharacters
        .skip(math.max(0, outputCharacters.length - budget))
        .toString();
  }
  return omitted ? '$_terminalSemanticOmission$output' : output;
}

final class TerminalCanvas extends StatefulWidget {
  const TerminalCanvas({
    super.key,
    required this.screen,
    this.settings = defaultTerminalSettings,
    this.onLinkTap,
    this.onBlankTap,
    this.onTerminalTap,
    this.onInteractionStart,
    this.onInteractionCancel,
    this.onVerticalDragStart,
    this.onVerticalDragUpdate,
    this.onVerticalDragEnd,
    this.onVerticalDragCancel,
    this.reserveLongPress = true,
    this.onPresented,
  });

  final CanonicalLiveScreen screen;
  final TerminalSettings settings;
  final ValueChanged<TerminalLink>? onLinkTap;
  final VoidCallback? onBlankTap;
  final ValueChanged<Offset>? onTerminalTap;
  final VoidCallback? onInteractionStart;
  final VoidCallback? onInteractionCancel;
  final GestureDragStartCallback? onVerticalDragStart;
  final GestureDragUpdateCallback? onVerticalDragUpdate;
  final GestureDragEndCallback? onVerticalDragEnd;
  final GestureDragCancelCallback? onVerticalDragCancel;
  final bool reserveLongPress;
  final ValueChanged<int>? onPresented;

  @override
  State<TerminalCanvas> createState() => _TerminalCanvasState();
}

final class _TerminalCanvasState extends State<TerminalCanvas> {
  static const _semanticUpdateInterval = Duration(milliseconds: 100);

  Timer? _cursorTimer;
  Timer? _semanticUpdateTimer;
  final ValueNotifier<bool> _cursorPhase = ValueNotifier(true);
  final ValueNotifier<String> _semanticValue = ValueNotifier('');
  CanonicalLiveScreen? _scheduledPresentationScreen;
  int _scheduledPresentationRevision = -1;
  int? _scheduledPresentationGeneration;

  @override
  void initState() {
    super.initState();
    _semanticValue.value = terminalScreenSemanticValue(widget.screen);
    _syncCursorTimer();
  }

  @override
  void didUpdateWidget(TerminalCanvas oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.settings.cursorBlink != widget.settings.cursorBlink ||
        oldWidget.screen.cursor?.blink != widget.screen.cursor?.blink) {
      _cursorPhase.value = true;
      _syncCursorTimer();
    }
    if (!identical(oldWidget.screen, widget.screen)) {
      _scheduleSemanticUpdate();
    }
  }

  @override
  void dispose() {
    _cursorTimer?.cancel();
    _semanticUpdateTimer?.cancel();
    _cursorPhase.dispose();
    _semanticValue.dispose();
    super.dispose();
  }

  void _scheduleSemanticUpdate() {
    if (_semanticUpdateTimer != null) return;
    _semanticUpdateTimer = Timer(_semanticUpdateInterval, () {
      _semanticUpdateTimer = null;
      if (mounted) {
        final next = terminalScreenSemanticValue(widget.screen);
        if (_semanticValue.value != next) _semanticValue.value = next;
      }
    });
  }

  void _syncCursorTimer() {
    _cursorTimer?.cancel();
    _cursorTimer = null;
    if (!widget.settings.cursorBlink || widget.screen.cursor?.blink != true) {
      return;
    }
    _cursorTimer = Timer.periodic(const Duration(milliseconds: 500), (_) {
      if (mounted) _cursorPhase.value = !_cursorPhase.value;
    });
  }

  @override
  Widget build(BuildContext context) {
    final metrics = widget.settings.metrics;
    final theme = widget.settings.theme;
    final contentSize = Size(
      widget.screen.cols * metrics.cellWidth,
      widget.screen.rows * metrics.rowHeight,
    );
    final paintedCanvas = RepaintBoundary(
      child: Listener(
        behavior: HitTestBehavior.opaque,
        onPointerDown: (_) => widget.onInteractionStart?.call(),
        onPointerCancel: (_) => widget.onInteractionCancel?.call(),
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          excludeFromSemantics: true,
          onVerticalDragStart: widget.onVerticalDragStart,
          onVerticalDragUpdate: widget.onVerticalDragUpdate,
          onVerticalDragEnd: widget.onVerticalDragEnd,
          onVerticalDragCancel: widget.onVerticalDragCancel,
          onLongPress: widget.reserveLongPress ? () {} : null,
          onTapUp: (details) {
            final row = (details.localPosition.dy / metrics.rowHeight).floor();
            final column = (details.localPosition.dx / metrics.cellWidth)
                .floor();
            final screenRow = row >= 0 && row < widget.screen.screenRows.length
                ? widget.screen.screenRows[row]
                : null;
            final link = screenRow == null
                ? null
                : terminalLinkAtColumn(screenRow, column);
            if (link != null && widget.onLinkTap != null) {
              widget.onLinkTap!(link);
            } else if (widget.onTerminalTap case final onTerminalTap?) {
              onTerminalTap(details.localPosition);
            } else {
              widget.onBlankTap?.call();
            }
          },
          child: ValueListenableBuilder<bool>(
            valueListenable: _cursorPhase,
            builder: (context, cursorPhase, _) => CustomPaint(
              key: const ValueKey('terminal-canvas'),
              size: contentSize,
              painter: _TerminalPainter(
                screen: widget.screen,
                metrics: metrics,
                theme: theme,
                fontFamily: widget.settings.fontFamily,
                drawCursor:
                    !widget.settings.cursorBlink ||
                    widget.screen.cursor?.blink != true ||
                    cursorPhase,
              ),
            ),
          ),
        ),
      ),
    );
    _schedulePresentationAcknowledgement();
    return ValueListenableBuilder<String>(
      valueListenable: _semanticValue,
      builder: (context, semanticValue, child) => Semantics(
        key: const ValueKey('terminal-output-semantics'),
        excludeSemantics: true,
        readOnly: true,
        label:
            'Terminal output, ${widget.screen.cols} columns by '
            '${widget.screen.rows} rows',
        value: semanticValue,
        child: child,
      ),
      child: ColoredBox(
        color: _color(theme.background),
        child: LayoutBuilder(
          builder: (context, constraints) {
            if (contentSize.width <= constraints.maxWidth) {
              return Align(alignment: Alignment.topLeft, child: paintedCanvas);
            }
            return SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: paintedCanvas,
            );
          },
        ),
      ),
    );
  }

  void _schedulePresentationAcknowledgement() {
    final generation = widget.screen.connectionGeneration.toInt();
    if (generation != _scheduledPresentationGeneration) {
      _scheduledPresentationGeneration = generation;
      _scheduledPresentationScreen = null;
      _scheduledPresentationRevision = -1;
    }
    final revision = widget.screen.revision.toInt();
    if (identical(widget.screen, _scheduledPresentationScreen) ||
        revision < _scheduledPresentationRevision ||
        widget.onPresented == null) {
      return;
    }
    _scheduledPresentationScreen = widget.screen;
    _scheduledPresentationRevision = revision;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) widget.onPresented?.call(revision);
    });
  }
}

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
        child: GestureDetector(
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
  }

  void _updateSelectionGesture(
    ScaleUpdateDetails details,
    Size viewport,
    TerminalCellMetrics metrics,
  ) {
    if (details.pointerCount >= 2 || _selectionGestureScrolled) {
      _selectionGestureScrolled = true;
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

final class _TerminalPainter extends CustomPainter {
  const _TerminalPainter({
    required this.screen,
    required this.metrics,
    required this.theme,
    required this.fontFamily,
    required this.drawCursor,
  });

  final CanonicalLiveScreen screen;
  final TerminalCellMetrics metrics;
  final TerminalTheme theme;
  final String fontFamily;
  final bool drawCursor;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawRect(
      Offset.zero & size,
      Paint()..color = _color(theme.background),
    );
    for (var rowIndex = 0; rowIndex < screen.screenRows.length; rowIndex += 1) {
      _paintRow(canvas, rowIndex, screen.screenRows[rowIndex]);
    }
    if (screen.cursor case final cursor? when cursor.visible && drawCursor) {
      _paintCursor(canvas, cursor);
    }
  }

  void _paintRow(Canvas canvas, int rowIndex, ScreenRow row) {
    var column = 0;
    final top = rowIndex * metrics.rowHeight;
    for (final cell in row.cells) {
      final width = cell.width <= 0 ? 0 : cell.width;
      if (width == 0) {
        continue;
      }
      final style = cell.hasStyle() ? cell.style : CellStyle();
      var foreground = _terminalColor(
        style.foreground,
        _color(theme.foreground),
        theme,
      );
      var background = _terminalColor(
        style.background,
        _color(theme.background),
        theme,
      );
      if (style.reverse) {
        final swapped = foreground;
        foreground = background;
        background = swapped;
      }
      final bounds = Rect.fromLTWH(
        column * metrics.cellWidth,
        top,
        width * metrics.cellWidth,
        metrics.rowHeight,
      );
      if (background != _color(theme.background)) {
        canvas.drawRect(bounds, Paint()..color = background);
      }
      if (cell.content.isNotEmpty) {
        _paintTerminalText(
          canvas: canvas,
          content: cell.content,
          authoritativeWidth: width,
          bounds: bounds,
          metrics: metrics,
          foreground: foreground,
          fontFamily: fontFamily,
          style: style,
        );
      }
      column += width;
    }
    if (column < screen.cols && row.hasTailFill()) {
      final tailColor = _terminalColor(
        row.tailFill.background,
        _color(theme.background),
        theme,
      );
      if (tailColor != _color(theme.background)) {
        canvas.drawRect(
          Rect.fromLTWH(
            column * metrics.cellWidth,
            top,
            (screen.cols - column) * metrics.cellWidth,
            metrics.rowHeight,
          ),
          Paint()..color = tailColor,
        );
      }
    }
  }

  void _paintCursor(Canvas canvas, TerminalCursor cursor) {
    if (cursor.row < 0 ||
        cursor.row >= screen.rows ||
        cursor.col < 0 ||
        cursor.col >= screen.cols) {
      return;
    }
    final span = terminalCursorCellSpan(
      cursor.row < screen.screenRows.length
          ? screen.screenRows[cursor.row]
          : null,
      cursor.col,
    );
    final cursorWidth = math.min(span.width, screen.cols - span.column);
    final bounds = Rect.fromLTWH(
      span.column * metrics.cellWidth,
      cursor.row * metrics.rowHeight,
      cursorWidth * metrics.cellWidth,
      metrics.rowHeight,
    );
    final paint = Paint()..color = _color(theme.cursor).withValues(alpha: 0.82);
    switch (cursor.shape) {
      case CursorShape.CURSOR_SHAPE_UNDERLINE:
        canvas.drawRect(
          Rect.fromLTWH(bounds.left, bounds.bottom - 2, bounds.width, 2),
          paint,
        );
      case CursorShape.CURSOR_SHAPE_BAR:
        canvas.drawRect(
          Rect.fromLTWH(bounds.left, bounds.top, 2, bounds.height),
          paint,
        );
      default:
        canvas.drawRect(bounds, paint);
    }
  }

  @override
  bool shouldRepaint(_TerminalPainter oldDelegate) {
    return oldDelegate.screen.connectionGeneration !=
            screen.connectionGeneration ||
        oldDelegate.screen.revision != screen.revision ||
        oldDelegate.screen.cols != screen.cols ||
        oldDelegate.screen.rows != screen.rows ||
        oldDelegate.metrics != metrics ||
        oldDelegate.theme.id != theme.id ||
        oldDelegate.fontFamily != fontFamily ||
        oldDelegate.drawCursor != drawCursor;
  }
}

final class _HistoryRowPainter extends CustomPainter {
  const _HistoryRowPainter({
    required this.row,
    required this.cols,
    required this.metrics,
    required this.theme,
    required this.fontFamily,
    required this.selection,
    required this.searchMatch,
    required this.searchMatches,
  });

  final ScreenRow row;
  final int cols;
  final TerminalCellMetrics metrics;
  final TerminalTheme theme;
  final String fontFamily;
  final ({int start, int end})? selection;
  final ({int start, int end})? searchMatch;
  final List<({int start, int end})> searchMatches;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawRect(
      Offset.zero & size,
      Paint()..color = _color(theme.background),
    );
    var column = 0;
    var firstSearchMatch = 0;
    for (final cell in row.cells) {
      final width = cell.width <= 0 ? 0 : cell.width;
      if (width == 0) continue;
      final style = cell.hasStyle() ? cell.style : CellStyle();
      var foreground = _terminalColor(
        style.foreground,
        _color(theme.foreground),
        theme,
      );
      var background = _terminalColor(
        style.background,
        _color(theme.background),
        theme,
      );
      if (style.reverse) {
        final swapped = foreground;
        foreground = background;
        background = swapped;
      }
      final bounds = Rect.fromLTWH(
        column * metrics.cellWidth,
        0,
        width * metrics.cellWidth,
        metrics.rowHeight,
      );
      if (background != _color(theme.background)) {
        canvas.drawRect(bounds, Paint()..color = background);
      }
      while (firstSearchMatch < searchMatches.length &&
          searchMatches[firstSearchMatch].end <= column) {
        firstSearchMatch += 1;
      }
      for (
        var index = firstSearchMatch;
        index < searchMatches.length &&
            searchMatches[index].start < column + width;
        index += 1
      ) {
        final match = searchMatches[index];
        _paintHighlight(
          canvas,
          cellStart: column,
          cellEnd: column + width,
          range: match,
          color: const Color(0xfff59e0b).withValues(alpha: 0.22),
        );
      }
      if (searchMatch case final match?) {
        _paintHighlight(
          canvas,
          cellStart: column,
          cellEnd: column + width,
          range: match,
          color: const Color(0xfffbbf24).withValues(alpha: 0.52),
        );
      }
      if (selection case final selected?) {
        _paintHighlight(
          canvas,
          cellStart: column,
          cellEnd: column + width,
          range: selected,
          color: const Color(0xff22d3ee).withValues(alpha: 0.38),
        );
      }
      if (cell.content.isNotEmpty) {
        _paintTerminalText(
          canvas: canvas,
          content: cell.content,
          authoritativeWidth: width,
          bounds: bounds,
          metrics: metrics,
          foreground: foreground,
          fontFamily: fontFamily,
          style: style,
        );
      }
      column += width;
    }
    if (column < cols && row.hasTailFill()) {
      final tailColor = _terminalColor(
        row.tailFill.background,
        _color(theme.background),
        theme,
      );
      if (tailColor != _color(theme.background)) {
        canvas.drawRect(
          Rect.fromLTWH(
            column * metrics.cellWidth,
            0,
            (cols - column) * metrics.cellWidth,
            metrics.rowHeight,
          ),
          Paint()..color = tailColor,
        );
      }
    }
  }

  void _paintHighlight(
    Canvas canvas, {
    required int cellStart,
    required int cellEnd,
    required ({int start, int end}) range,
    required Color color,
  }) {
    final start = range.start.clamp(cellStart, cellEnd);
    final end = range.end.clamp(cellStart, cellEnd);
    if (end <= start) return;
    canvas.drawRect(
      Rect.fromLTWH(
        start * metrics.cellWidth,
        0,
        (end - start) * metrics.cellWidth,
        metrics.rowHeight,
      ),
      Paint()..color = color,
    );
  }

  @override
  bool shouldRepaint(_HistoryRowPainter oldDelegate) =>
      oldDelegate.row != row ||
      oldDelegate.cols != cols ||
      oldDelegate.metrics != metrics ||
      oldDelegate.theme.id != theme.id ||
      oldDelegate.fontFamily != fontFamily ||
      oldDelegate.selection != selection ||
      oldDelegate.searchMatch != searchMatch ||
      oldDelegate.searchMatches != searchMatches;
}

typedef TerminalGridTextRun = ({String text, int column, int width});
typedef TerminalCursorCellSpan = ({int column, int width});

TerminalCursorCellSpan terminalCursorCellSpan(
  ScreenRow? row,
  int cursorColumn,
) {
  if (row == null) return (column: cursorColumn, width: 1);
  var cellColumn = 0;
  for (final cell in row.cells) {
    final cellWidth = math.max(0, cell.width);
    final cellEnd = cellColumn + cellWidth;
    if (cursorColumn >= cellColumn && cursorColumn < cellEnd) {
      final relativeColumn = cursorColumn - cellColumn;
      for (final run in terminalGridTextRuns(cell.content, cellWidth)) {
        final runEnd = run.column + run.width;
        if (relativeColumn < run.column || relativeColumn >= runEnd) continue;
        final graphemeCount = run.text.characters.length;
        if (graphemeCount > 1) {
          return (column: cursorColumn, width: 1);
        }
        return (column: cellColumn + run.column, width: run.width);
      }
      return (column: cursorColumn, width: 1);
    }
    cellColumn = cellEnd;
  }
  return (column: cursorColumn, width: 1);
}

/// Splits fallback-font glyphs at terminal cell boundaries while retaining
/// efficient runs for ordinary single-column text.
List<TerminalGridTextRun> terminalGridTextRuns(
  String content,
  int authoritativeWidth,
) {
  if (content.isEmpty || authoritativeWidth <= 0) return const [];
  final graphemes = content.characters.toList(growable: false);
  if (graphemes.isEmpty) return const [];
  final equalWidth = authoritativeWidth % graphemes.length == 0
      ? authoritativeWidth ~/ graphemes.length
      : 0;
  final widths = [
    for (final grapheme in graphemes)
      equalWidth > 0 ? equalWidth : terminalGraphemeCellWidth(grapheme),
  ];

  final runs = <TerminalGridTextRun>[];
  var column = 0;
  var narrowStart = -1;
  var narrowWidth = 0;
  var narrowText = StringBuffer();

  void flushNarrowRun() {
    if (narrowStart < 0 || narrowText.isEmpty) return;
    runs.add((
      text: narrowText.toString(),
      column: narrowStart,
      width: narrowWidth,
    ));
    narrowStart = -1;
    narrowWidth = 0;
    narrowText = StringBuffer();
  }

  for (var index = 0; index < graphemes.length; index += 1) {
    final naturalWidth = widths[index];
    if (naturalWidth <= 0) {
      if (narrowStart >= 0) narrowText.write(graphemes[index]);
      continue;
    }
    if (column >= authoritativeWidth) break;
    final width = math.min(naturalWidth, authoritativeWidth - column);
    if (width == 1) {
      if (narrowStart < 0) narrowStart = column;
      narrowText.write(graphemes[index]);
      narrowWidth += 1;
    } else {
      flushNarrowRun();
      runs.add((text: graphemes[index], column: column, width: width));
    }
    column += width;
  }
  flushNarrowRun();
  return runs;
}

void _paintTerminalText({
  required Canvas canvas,
  required String content,
  required int authoritativeWidth,
  required Rect bounds,
  required TerminalCellMetrics metrics,
  required Color foreground,
  required String fontFamily,
  required CellStyle style,
}) {
  final textStyle = TextStyle(
    color: foreground,
    fontFamily: fontFamily,
    fontFamilyFallback: terminalFontFamilyFallback,
    fontSize: metrics.fontSize,
    height: 1,
    fontWeight: style.bold ? FontWeight.w700 : FontWeight.w400,
    fontStyle: style.italic ? FontStyle.italic : FontStyle.normal,
    decoration: TextDecoration.combine([
      if (style.underline) TextDecoration.underline,
      if (style.strikethrough) TextDecoration.lineThrough,
    ]),
    decorationColor: foreground,
  );
  final textTop = bounds.top + (metrics.rowHeight - metrics.fontSize) / 2;
  for (final run in terminalGridTextRuns(content, authoritativeWidth)) {
    final painter = TextPainter(
      text: TextSpan(text: run.text, style: textStyle),
      textDirection: TextDirection.ltr,
      maxLines: 1,
    )..layout(maxWidth: (run.width + 1) * metrics.cellWidth);
    painter.paint(
      canvas,
      Offset(bounds.left + run.column * metrics.cellWidth, textTop),
    );
  }
  paintTerminalGridGlyphsInRun(
    canvas,
    content,
    bounds,
    metrics.cellWidth,
    foreground,
  );
}

bool paintTerminalBlockElement(
  Canvas canvas,
  String content,
  Rect bounds,
  Color color,
) {
  final runes = content.runes;
  if (runes.length != 1) return false;
  final rects = terminalBlockElementRects(runes.single, bounds);
  if (rects == null) return false;
  final paint = Paint()
    ..color = color
    ..isAntiAlias = false;
  for (final rect in rects) {
    canvas.drawRect(rect, paint);
  }
  return true;
}

void paintTerminalGridGlyphsInRun(
  Canvas canvas,
  String content,
  Rect runBounds,
  double cellWidth,
  Color color,
) {
  var column = 0;
  for (final grapheme in content.characters) {
    final width = terminalGraphemeCellWidth(grapheme);
    final runes = grapheme.runes;
    if (runes.length == 1) {
      final bounds = Rect.fromLTWH(
        runBounds.left + column * cellWidth,
        runBounds.top,
        math.max(1, width) * cellWidth,
        runBounds.height,
      );
      final rects = terminalBlockElementRects(runes.single, bounds);
      if (rects != null) {
        final paint = Paint()
          ..color = color
          ..isAntiAlias = false;
        for (final rect in rects) {
          canvas.drawRect(rect, paint);
        }
      } else if (terminalShadeOpacity(runes.single) case final opacity?) {
        canvas.drawRect(
          bounds,
          Paint()
            ..color = color.withValues(alpha: opacity)
            ..isAntiAlias = false,
        );
      } else {
        paintTerminalBoxDrawingElement(canvas, runes.single, bounds, color);
      }
    }
    column += width;
  }
}

double? terminalShadeOpacity(int rune) => switch (rune) {
  0x2591 => 0.25,
  0x2592 => 0.5,
  0x2593 => 0.75,
  _ => null,
};

bool paintTerminalBoxDrawingElement(
  Canvas canvas,
  int rune,
  Rect bounds,
  Color color,
) {
  if (rune == 0x2571 || rune == 0x2572 || rune == 0x2573) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = math.max(1, math.min(bounds.width, bounds.height) * 0.06)
      ..strokeCap = StrokeCap.square
      ..isAntiAlias = false;
    if (rune == 0x2571 || rune == 0x2573) {
      canvas.drawLine(bounds.bottomLeft, bounds.topRight, paint);
    }
    if (rune == 0x2572 || rune == 0x2573) {
      canvas.drawLine(bounds.topLeft, bounds.bottomRight, paint);
    }
    return true;
  }

  final doubleLine = rune >= 0x2550 && rune <= 0x256c;
  final topology = _terminalBoxTopology(rune);
  if (topology == null) return false;
  final heavy = switch (rune) {
    0x2501 ||
    0x2503 ||
    0x2505 ||
    0x2507 ||
    0x2509 ||
    0x250b ||
    0x250f ||
    0x2513 ||
    0x2517 ||
    0x251b ||
    0x2523 ||
    0x252b ||
    0x2533 ||
    0x253b ||
    0x254b ||
    0x2578 ||
    0x2579 ||
    0x257a ||
    0x257b ||
    0x257e ||
    0x257f => true,
    _ => false,
  };
  final baseThickness = math.max(
    1.0,
    math.min(bounds.width, bounds.height) * 0.06,
  );
  final thickness = heavy ? baseThickness * 2 : baseThickness;
  final center = bounds.center;
  final paint = Paint()
    ..color = color
    ..isAntiAlias = false;

  void horizontal(double y, bool left, bool right) {
    if (left) {
      canvas.drawRect(
        Rect.fromLTRB(
          bounds.left,
          y - thickness / 2,
          center.dx,
          y + thickness / 2,
        ),
        paint,
      );
    }
    if (right) {
      canvas.drawRect(
        Rect.fromLTRB(
          center.dx,
          y - thickness / 2,
          bounds.right,
          y + thickness / 2,
        ),
        paint,
      );
    }
  }

  void vertical(double x, bool up, bool down) {
    if (up) {
      canvas.drawRect(
        Rect.fromLTRB(
          x - thickness / 2,
          bounds.top,
          x + thickness / 2,
          center.dy,
        ),
        paint,
      );
    }
    if (down) {
      canvas.drawRect(
        Rect.fromLTRB(
          x - thickness / 2,
          center.dy,
          x + thickness / 2,
          bounds.bottom,
        ),
        paint,
      );
    }
  }

  if (doubleLine) {
    final gap = baseThickness * 1.4;
    horizontal(center.dy - gap, topology.left, topology.right);
    horizontal(center.dy + gap, topology.left, topology.right);
    vertical(center.dx - gap, topology.up, topology.down);
    vertical(center.dx + gap, topology.up, topology.down);
  } else {
    horizontal(center.dy, topology.left, topology.right);
    vertical(center.dx, topology.up, topology.down);
  }
  return true;
}

({bool left, bool right, bool up, bool down})? _terminalBoxTopology(int rune) {
  if (rune == 0x2500 ||
      rune == 0x2501 ||
      rune == 0x2504 ||
      rune == 0x2505 ||
      rune == 0x2508 ||
      rune == 0x2509) {
    return (left: true, right: true, up: false, down: false);
  }
  if (rune == 0x2502 ||
      rune == 0x2503 ||
      rune == 0x2506 ||
      rune == 0x2507 ||
      rune == 0x250a ||
      rune == 0x250b) {
    return (left: false, right: false, up: true, down: true);
  }
  if (rune >= 0x250c && rune <= 0x250f) {
    return (left: false, right: true, up: false, down: true);
  }
  if (rune >= 0x2510 && rune <= 0x2513) {
    return (left: true, right: false, up: false, down: true);
  }
  if (rune >= 0x2514 && rune <= 0x2517) {
    return (left: false, right: true, up: true, down: false);
  }
  if (rune >= 0x2518 && rune <= 0x251b) {
    return (left: true, right: false, up: true, down: false);
  }
  if (rune >= 0x251c && rune <= 0x2523) {
    return (left: false, right: true, up: true, down: true);
  }
  if (rune >= 0x2524 && rune <= 0x252b) {
    return (left: true, right: false, up: true, down: true);
  }
  if (rune >= 0x252c && rune <= 0x2533) {
    return (left: true, right: true, up: false, down: true);
  }
  if (rune >= 0x2534 && rune <= 0x253b) {
    return (left: true, right: true, up: true, down: false);
  }
  if (rune >= 0x253c && rune <= 0x254b) {
    return (left: true, right: true, up: true, down: true);
  }
  if (rune == 0x2550) {
    return (left: true, right: true, up: false, down: false);
  }
  if (rune == 0x2551) {
    return (left: false, right: false, up: true, down: true);
  }
  if (rune >= 0x2552 && rune <= 0x2554) {
    return (left: false, right: true, up: false, down: true);
  }
  if (rune >= 0x2555 && rune <= 0x2557) {
    return (left: true, right: false, up: false, down: true);
  }
  if (rune >= 0x2558 && rune <= 0x255a) {
    return (left: false, right: true, up: true, down: false);
  }
  if (rune >= 0x255b && rune <= 0x255d) {
    return (left: true, right: false, up: true, down: false);
  }
  if (rune >= 0x255e && rune <= 0x2560) {
    return (left: false, right: true, up: true, down: true);
  }
  if (rune >= 0x2561 && rune <= 0x2563) {
    return (left: true, right: false, up: true, down: true);
  }
  if (rune >= 0x2564 && rune <= 0x2566) {
    return (left: true, right: true, up: false, down: true);
  }
  if (rune >= 0x2567 && rune <= 0x2569) {
    return (left: true, right: true, up: true, down: false);
  }
  if (rune >= 0x256a && rune <= 0x256c) {
    return (left: true, right: true, up: true, down: true);
  }
  return switch (rune) {
    0x256d => (left: false, right: true, up: false, down: true),
    0x256e => (left: true, right: false, up: false, down: true),
    0x256f => (left: true, right: false, up: true, down: false),
    0x2570 => (left: false, right: true, up: true, down: false),
    0x2574 || 0x2578 => (left: true, right: false, up: false, down: false),
    0x2575 || 0x2579 => (left: false, right: false, up: true, down: false),
    0x2576 || 0x257a => (left: false, right: true, up: false, down: false),
    0x2577 || 0x257b => (left: false, right: false, up: false, down: true),
    0x257c || 0x257e => (left: true, right: true, up: false, down: false),
    0x257d || 0x257f => (left: false, right: false, up: true, down: true),
    _ => null,
  };
}

int terminalGraphemeCellWidth(String grapheme) {
  if (grapheme.isEmpty) return 0;
  final runes = grapheme.runes.toList(growable: false);
  if (runes.every(_isZeroWidthCodepoint)) return 0;
  return runes.any(_isWideCodepoint) ? 2 : 1;
}

bool _isZeroWidthCodepoint(int rune) =>
    (rune >= 0x0300 && rune <= 0x036f) ||
    (rune >= 0x1ab0 && rune <= 0x1aff) ||
    (rune >= 0x1dc0 && rune <= 0x1dff) ||
    (rune >= 0x20d0 && rune <= 0x20ff) ||
    (rune >= 0xfe00 && rune <= 0xfe0f) ||
    (rune >= 0xfe20 && rune <= 0xfe2f) ||
    (rune >= 0xe0100 && rune <= 0xe01ef) ||
    rune == 0x200b ||
    rune == 0x200c ||
    rune == 0x200d;

bool _isWideCodepoint(int rune) =>
    (rune >= 0x1100 && rune <= 0x115f) ||
    rune == 0x2329 ||
    rune == 0x232a ||
    (rune >= 0x2e80 && rune <= 0xa4cf && rune != 0x303f) ||
    (rune >= 0xac00 && rune <= 0xd7a3) ||
    (rune >= 0xf900 && rune <= 0xfaff) ||
    (rune >= 0xfe10 && rune <= 0xfe19) ||
    (rune >= 0xfe30 && rune <= 0xfe6f) ||
    (rune >= 0xff00 && rune <= 0xff60) ||
    (rune >= 0xffe0 && rune <= 0xffe6) ||
    (rune >= 0x1f000 && rune <= 0x1faff) ||
    (rune >= 0x20000 && rune <= 0x3fffd);

List<Rect>? terminalBlockElementRects(int rune, Rect bounds) {
  if (rune == 0x2580) {
    return [
      Rect.fromLTWH(bounds.left, bounds.top, bounds.width, bounds.height / 2),
    ];
  }
  if (rune >= 0x2581 && rune <= 0x2588) {
    final height = bounds.height * (rune - 0x2580) / 8;
    return [
      Rect.fromLTWH(bounds.left, bounds.bottom - height, bounds.width, height),
    ];
  }
  if (rune >= 0x2589 && rune <= 0x258f) {
    final width = bounds.width * (0x2590 - rune) / 8;
    return [Rect.fromLTWH(bounds.left, bounds.top, width, bounds.height)];
  }
  if (rune == 0x2590) {
    return [
      Rect.fromLTWH(
        bounds.left + bounds.width / 2,
        bounds.top,
        bounds.width / 2,
        bounds.height,
      ),
    ];
  }
  if (rune == 0x2594) {
    return [
      Rect.fromLTWH(bounds.left, bounds.top, bounds.width, bounds.height / 8),
    ];
  }
  if (rune == 0x2595) {
    return [
      Rect.fromLTWH(
        bounds.right - bounds.width / 8,
        bounds.top,
        bounds.width / 8,
        bounds.height,
      ),
    ];
  }
  if (rune < 0x2596 || rune > 0x259f) return null;

  final left = bounds.left;
  final top = bounds.top;
  final halfWidth = bounds.width / 2;
  final halfHeight = bounds.height / 2;
  final quadrants = switch (rune) {
    0x2596 => const [false, false, true, false],
    0x2597 => const [false, false, false, true],
    0x2598 => const [true, false, false, false],
    0x2599 => const [true, false, true, true],
    0x259a => const [true, false, false, true],
    0x259b => const [true, true, true, false],
    0x259c => const [true, true, false, true],
    0x259d => const [false, true, false, false],
    0x259e => const [false, true, true, false],
    _ => const [false, true, true, true],
  };
  return [
    if (quadrants[0]) Rect.fromLTWH(left, top, halfWidth, halfHeight),
    if (quadrants[1])
      Rect.fromLTWH(left + halfWidth, top, halfWidth, halfHeight),
    if (quadrants[2])
      Rect.fromLTWH(left, top + halfHeight, halfWidth, halfHeight),
    if (quadrants[3])
      Rect.fromLTWH(left + halfWidth, top + halfHeight, halfWidth, halfHeight),
  ];
}

Color _terminalColor(String value, Color fallback, TerminalTheme theme) {
  if (value.isEmpty) {
    return fallback;
  }
  final hex = RegExp(r'^#([0-9a-fA-F]{6})$').firstMatch(value);
  if (hex != null) {
    return Color(0xff000000 | int.parse(hex.group(1)!, radix: 16));
  }
  final indexed = RegExp(r'^(?:ansi|idx):(\d{1,3})$').firstMatch(value);
  if (indexed == null) {
    return fallback;
  }
  final index = int.tryParse(indexed.group(1)!);
  if (index == null || index < 0 || index > 255) {
    return fallback;
  }
  return _ansiColor(index, theme);
}

Color _ansiColor(int index, TerminalTheme theme) {
  if (index < theme.ansi.length) return _color(theme.ansi[index]);
  if (index >= 232) {
    final level = 8 + (index - 232) * 10;
    return Color.fromARGB(255, level, level, level);
  }
  final cube = index - 16;
  int channel(int value) => value == 0 ? 0 : 55 + value * 40;
  return Color.fromARGB(
    255,
    channel(cube ~/ 36),
    channel((cube % 36) ~/ 6),
    channel(cube % 6),
  );
}

Color _color(int rgb) => Color(0xff000000 | rgb);
