part of 'terminal_workspace_screen.dart';

final class _ResizeControlSheet extends StatefulWidget {
  const _ResizeControlSheet({
    required this.connection,
    required this.initialControl,
    required this.viewportCells,
  });

  final TerminalConnection connection;
  final ResizeControl initialControl;
  final ({int cols, int rows})? viewportCells;

  @override
  State<_ResizeControlSheet> createState() => _ResizeControlSheetState();
}

final class _ResizeControlSheetState extends State<_ResizeControlSheet> {
  late ResizeControl _control;
  StreamSubscription<ResizeControl>? _subscription;
  bool _busy = false;
  String? _error;

  bool get _ownsResize =>
      _control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_OWNER ||
      _control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_SIZE_LOCKED;

  @override
  void initState() {
    super.initState();
    _control = widget.initialControl.deepCopy();
    _subscription = widget.connection.watchResizeControl().listen((control) {
      if (mounted) setState(() => _control = control);
    });
  }

  @override
  void dispose() {
    unawaited(_subscription?.cancel());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final size = _control.hasOwnership() && _control.ownership.hasSize()
        ? _control.ownership.size
        : widget.connection.terminalSize;
    final owner = _control.ownerSurfaceId.isNotEmpty
        ? _control.ownerSurfaceId
        : _control.hasOwnership()
        ? _control.ownership.ownerSurfaceId
        : '';
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Terminal size',
              style: TextStyle(
                color: palette.text,
                fontSize: 17,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 14),
            _ResizeStatusRow(
              icon: _control.sizeLocked
                  ? Icons.lock_outline_rounded
                  : _ownsResize
                  ? Icons.aspect_ratio_rounded
                  : Icons.visibility_outlined,
              label: _resizeControlStatus(_control),
              value: '${size.cols} x ${size.rows}',
            ),
            if (owner.isNotEmpty) ...[
              const SizedBox(height: 8),
              _ResizeStatusRow(
                icon: Icons.devices_other_rounded,
                label: 'Owner surface',
                value: owner,
              ),
            ],
            if (_error case final error?) ...[
              const SizedBox(height: 12),
              Text(
                error,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: palette.danger, fontSize: 12),
              ),
            ],
            const SizedBox(height: 18),
            if (_ownsResize) ...[
              FilledButton.icon(
                onPressed: _busy
                    ? null
                    : () => _run(
                        () => widget.connection.setResizeLock(
                          !_control.sizeLocked,
                        ),
                      ),
                icon: Icon(
                  _control.sizeLocked
                      ? Icons.lock_open_rounded
                      : Icons.lock_outline_rounded,
                ),
                label: Text(_control.sizeLocked ? 'Unlock size' : 'Lock size'),
              ),
              const SizedBox(height: 8),
              OutlinedButton.icon(
                onPressed: _busy
                    ? null
                    : () => _run(widget.connection.releaseResizeOwnership),
                icon: const Icon(Icons.logout_rounded),
                label: const Text('Release ownership'),
              ),
            ] else
              FilledButton.icon(
                onPressed: _busy ? null : _takeOwnership,
                icon: const Icon(Icons.aspect_ratio_rounded),
                label: const Text('Take resize ownership'),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _takeOwnership() async {
    final cells = widget.viewportCells;
    final size = widget.connection.terminalSize;
    await _run(
      () => widget.connection.requestResizeOwnership(
        cols: cells?.cols ?? size.cols,
        rows: cells?.rows ?? size.rows,
      ),
    );
  }

  Future<void> _run(Future<ResizeControl> Function() operation) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final control = await operation();
      if (mounted) setState(() => _control = control);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

final class _ResizeStatusRow extends StatelessWidget {
  const _ResizeStatusRow({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        Icon(icon, color: palette.muted, size: 20),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            label,
            style: TextStyle(color: palette.text, fontSize: 13),
          ),
        ),
        const SizedBox(width: 12),
        Flexible(
          child: Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: palette.text,
              fontFamily: 'monospace',
              fontSize: 12,
            ),
          ),
        ),
      ],
    );
  }
}

String _resizeControlStatus(ResizeControl control) {
  if (control.sizeLocked) {
    return control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER
        ? 'Follower / locked'
        : 'Owner / locked';
  }
  return switch (control.reason) {
    ResizeControlReason.RESIZE_CONTROL_REASON_OWNER => 'Resize owner',
    ResizeControlReason.RESIZE_CONTROL_REASON_OBSERVER => 'Observer',
    ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER => 'Follower',
    _ => control.canResize ? 'Resize owner' : 'Unavailable',
  };
}

final class _KeyboardKeyButton extends StatefulWidget {
  const _KeyboardKeyButton({
    required this.keyboardVisible,
    required this.focusLocked,
    required this.onTap,
    required this.onLongPress,
  });

  final bool keyboardVisible;
  final bool focusLocked;
  final VoidCallback? onTap;
  final VoidCallback? onLongPress;

  @override
  State<_KeyboardKeyButton> createState() => _KeyboardKeyButtonState();
}

final class _KeyboardKeyButtonState extends State<_KeyboardKeyButton> {
  static const _longPressDelay = Duration(milliseconds: 400);

  Timer? _longPressTimer;
  int? _pointer;
  bool _longPressTriggered = false;

  @override
  void dispose() {
    _cancelLongPress();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final enabled = widget.onTap != null && widget.onLongPress != null;
    final active = widget.focusLocked || widget.keyboardVisible;
    final tooltip = widget.focusLocked
        ? 'Allow the system keyboard to open'
        : widget.keyboardVisible
        ? 'Hide system keyboard'
        : 'Show system keyboard';
    final background = widget.focusLocked
        ? palette.warning
        : widget.keyboardVisible
        ? palette.text
        : palette.surfaceRaised;
    final foreground = active
        ? palette.background
        : enabled
        ? palette.muted
        : palette.faint;

    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: tooltip,
      onTap: enabled ? _handleSemanticTap : null,
      child: Tooltip(
        message: tooltip,
        child: Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: enabled ? _handlePointerDown : null,
          onPointerUp: enabled ? _handlePointerUp : null,
          onPointerCancel: enabled ? _handlePointerCancel : null,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: background,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Center(
              child: Icon(LucideIcons.keyboard, size: 18, color: foreground),
            ),
          ),
        ),
      ),
    );
  }

  void _handlePointerDown(PointerDownEvent event) {
    if (_pointer != null) return;
    _pointer = event.pointer;
    _longPressTriggered = false;
    _longPressTimer = Timer(_longPressDelay, () {
      _longPressTimer = null;
      _longPressTriggered = true;
      HapticFeedback.mediumImpact();
      widget.onLongPress?.call();
    });
  }

  void _handlePointerUp(PointerUpEvent event) {
    if (_pointer != event.pointer) return;
    _pointer = null;
    _cancelLongPress();
    if (_longPressTriggered) {
      _longPressTriggered = false;
      return;
    }
    HapticFeedback.selectionClick();
    widget.onTap?.call();
  }

  void _handlePointerCancel(PointerCancelEvent event) {
    if (_pointer != event.pointer) return;
    _pointer = null;
    _longPressTriggered = false;
    _cancelLongPress();
  }

  void _handleSemanticTap() {
    HapticFeedback.selectionClick();
    widget.onTap?.call();
  }

  void _cancelLongPress() {
    _longPressTimer?.cancel();
    _longPressTimer = null;
  }
}

bool _mouseTrackingEnabled(TerminalModes? modes) =>
    modes?.mouseTracking == true ||
    modes?.mouseX10 == true ||
    modes?.mouseNormal == true ||
    modes?.mouseButtonEvent == true ||
    modes?.mouseAnyEvent == true;

const _usbKeyboardPage = 0x00070000;

/// follower 的屏幕尺寸可能和本机视口对不上（例如桌面持有 resize lease）：
/// 比视口宽时按比例缩小到视口宽度，宁可字小一点，也不要把行尾裁掉。
///
/// 缩放后的布局尺寸必须跟着视觉尺寸走，否则父级仍按原始尺寸占位，右边和
/// 下方会留出大片没画背景的空白。背景色铺满整个可用区域，内容顶部对齐。
final class _FitWidthCanvas extends StatelessWidget {
  const _FitWidthCanvas({
    required this.screen,
    required this.metrics,
    required this.background,
    required this.child,
  });

  final CanonicalLiveScreen screen;
  final TerminalCellMetrics metrics;
  final int background;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final naturalWidth = screen.cols * metrics.cellWidth;
        final naturalHeight = screen.rows * metrics.rowHeight;
        final scale = naturalWidth <= 0 || constraints.maxWidth <= 0
            ? 1.0
            : math.min(1.0, constraints.maxWidth / naturalWidth);
        final scaledWidth = naturalWidth * scale;
        final scaledHeight = naturalHeight * scale;
        return ColoredBox(
          color: Color(0xff000000 | background),
          child: (naturalWidth <= 0 || naturalHeight <= 0)
              ? child
              : Align(
                  alignment: Alignment.topLeft,
                  child: SizedBox(
                    width: scaledWidth,
                    height: scaledHeight,
                    child: FittedBox(
                      fit: BoxFit.fill,
                      alignment: Alignment.topLeft,
                      child: SizedBox(
                        width: naturalWidth,
                        height: naturalHeight,
                        child: child,
                      ),
                    ),
                  ),
                ),
        );
      },
    );
  }
}
