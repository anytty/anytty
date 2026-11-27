part of 'terminal_canvas.dart';


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

/// 行级 `ui.Picture` 缓存。
///
/// merge 层保证「没有变化的行是同一个 ScreenRow 实例」，所以脏行判定
/// 直接用 `identical(row)`，不需要额外传递 damage 信息。
final class _RowPictureCache {
  String? _signature;
  final List<_CachedRowPicture?> _rows = [];

  /// 测试用：本进程累计「重新录制的行数」与「命中缓存的行数」。
  static int debugRecordedRows = 0;
  static int debugHitRows = 0;

  void sync(String signature, int count) {
    if (signature != _signature) {
      clear();
      _signature = signature;
    }
    if (_rows.length > count) {
      for (var index = count; index < _rows.length; index += 1) {
        _rows[index]?.dispose();
      }
      _rows.removeRange(count, _rows.length);
    } else if (_rows.length < count) {
      _rows.addAll(List<_CachedRowPicture?>.filled(count - _rows.length, null));
    }
  }

  ui.Picture pictureFor(
    int index,
    ScreenRow row,
    void Function(Canvas canvas) record,
  ) {
    final cached = _rows[index];
    if (cached != null && identical(cached.row, row)) {
      debugHitRows += 1;
      return cached.picture;
    }
    debugRecordedRows += 1;
    cached?.dispose();
    final recorder = ui.PictureRecorder();
    record(Canvas(recorder));
    final picture = recorder.endRecording();
    _rows[index] = _CachedRowPicture(row, picture);
    return picture;
  }

  void clear() {
    for (final cached in _rows) {
      cached?.dispose();
    }
    _rows.clear();
    _signature = null;
  }

  void dispose() => clear();
}

/// 测试用：读取行级画面缓存的累计统计并清零。
///
/// `recorded` 是真正重新录制的行数，`hits` 是直接复用缓存的点数；
/// 绘制成本与 `recorded` 成正比。
@visibleForTesting
({int recorded, int hits}) debugTakeTerminalPaintStats() {
  final stats = (
    recorded: _RowPictureCache.debugRecordedRows,
    hits: _RowPictureCache.debugHitRows,
  );
  _RowPictureCache.debugRecordedRows = 0;
  _RowPictureCache.debugHitRows = 0;
  return stats;
}

final class _CachedRowPicture {
  _CachedRowPicture(this.row, this.picture);

  final ScreenRow row;
  final ui.Picture picture;

  void dispose() => picture.dispose();
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
  final _rowCache = _RowPictureCache();

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
    _rowCache.dispose();
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
                rowCache: _rowCache,
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
