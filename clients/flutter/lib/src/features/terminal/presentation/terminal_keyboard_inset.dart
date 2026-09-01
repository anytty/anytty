import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

/// Prevents Flutter's duplicate IME inset from rebuilding the terminal route.
final class TerminalKeyboardMediaQuery extends StatelessWidget {
  const TerminalKeyboardMediaQuery({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return MediaQuery(
      data: MediaQuery.of(context).copyWith(viewInsets: EdgeInsets.zero),
      child: child,
    );
  }
}

/// Keeps the terminal viewport and its key bar in one IME-visible workspace.
final class TerminalKeyboardWorkspace extends StatelessWidget {
  const TerminalKeyboardWorkspace({
    super.key,
    required this.visualInset,
    required this.child,
  });

  final ValueListenable<double> visualInset;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) => ClipRect(
        child: ValueListenableBuilder<double>(
          valueListenable: visualInset,
          child: RepaintBoundary(
            child: SizedBox(
              width: constraints.hasBoundedWidth ? constraints.maxWidth : null,
              height: constraints.hasBoundedHeight
                  ? constraints.maxHeight
                  : null,
              child: child,
            ),
          ),
          builder: (context, inset, workspace) {
            final normalizedInset = inset.isFinite ? math.max(0.0, inset) : 0.0;
            final boundedInset = constraints.hasBoundedHeight
                ? math.min(normalizedInset, constraints.maxHeight)
                : normalizedInset;
            return Transform.translate(
              offset: Offset(0, -boundedInset),
              child: workspace,
            );
          },
        ),
      ),
    );
  }
}

/// Keeps live terminal updates from invalidating the translated workspace.
final class TerminalKeyboardFrameFreeze extends StatefulWidget {
  const TerminalKeyboardFrameFreeze({
    super.key,
    required this.visualInset,
    required this.settledInset,
    required this.child,
  });

  final ValueListenable<double> visualInset;
  final double settledInset;
  final Widget child;

  @override
  State<TerminalKeyboardFrameFreeze> createState() =>
      _TerminalKeyboardFrameFreezeState();
}

final class _TerminalKeyboardFrameFreezeState
    extends State<TerminalKeyboardFrameFreeze> {
  late Widget _visibleChild;

  bool get _keyboardAnimating =>
      (widget.visualInset.value - widget.settledInset).abs() > 1;

  @override
  void initState() {
    super.initState();
    _visibleChild = widget.child;
  }

  @override
  void didUpdateWidget(TerminalKeyboardFrameFreeze oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!_keyboardAnimating) _visibleChild = widget.child;
  }

  @override
  Widget build(BuildContext context) => _visibleChild;
}

double resolveTerminalKeyboardShift({
  required double keyboardInset,
  required double visibleHeight,
  required int? cursorRow,
  required double rowHeight,
}) {
  if (keyboardInset <= 0 ||
      visibleHeight <= 0 ||
      cursorRow == null ||
      cursorRow < 0 ||
      rowHeight <= 0) {
    return 0;
  }
  final cursorTop = cursorRow * rowHeight;
  if (cursorTop < visibleHeight) return 0;
  return math.min(keyboardInset, cursorTop - visibleHeight + rowHeight * 2);
}

final class TerminalKeyboardInsetStabilizer {
  TerminalKeyboardInsetStabilizer({
    required this.onInsetChanged,
    this.settleDelay = const Duration(milliseconds: 120),
  });

  final void Function(double inset) onInsetChanged;
  final Duration settleDelay;

  Timer? _timer;
  double _rawInset = 0;
  double _settledInset = 0;
  bool _disposed = false;

  double get settledInset => _settledInset;

  void update(double inset) {
    if (_disposed) return;
    final next = inset < 1 ? 0.0 : inset;
    _rawInset = next;
    _timer?.cancel();
    _timer = null;

    if (next == 0) _commit(0);
    if (next == 0) return;

    _timer = Timer(settleDelay, () {
      _timer = null;
      if (!_disposed) _commit(_rawInset);
    });
  }

  void _commit(double inset) {
    if ((_settledInset - inset).abs() < 1) return;
    _settledInset = inset;
    onInsetChanged(inset);
  }

  void dispose() {
    _disposed = true;
    _timer?.cancel();
    _timer = null;
  }
}
