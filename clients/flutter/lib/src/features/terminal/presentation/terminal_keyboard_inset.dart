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

/// Keeps the IME-visible workspace together while the keyboard animates.
///
/// The terminal route uses [translateVisualInset] so the system keyboard and
/// its content move in the same composited layer. Smaller consumers can use
/// the settled [layoutInset] mode when they need a real bottom re-layout.
final class TerminalKeyboardWorkspace extends StatelessWidget {
  const TerminalKeyboardWorkspace({
    super.key,
    required this.visualInset,
    this.layoutInset,
    this.translateVisualInset = false,
    required this.child,
  });

  final ValueListenable<double> visualInset;

  /// Optional settled inset for expensive workspace layout changes.
  ///
  /// When provided, raw IME animation frames do not relayout the terminal.
  /// The parent updates this value after the inset stabilizer settles.
  final double? layoutInset;

  /// Moves the workspace in the compositor for every keyboard animation frame.
  final bool translateVisualInset;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final workspace = RepaintBoundary(
          child: SizedBox(
            width: constraints.hasBoundedWidth ? constraints.maxWidth : null,
            height: translateVisualInset && constraints.hasBoundedHeight
                ? constraints.maxHeight
                : null,
            child: child,
          ),
        );

        double boundedInset(double inset) {
          final normalizedInset = inset.isFinite ? math.max(0.0, inset) : 0.0;
          return constraints.hasBoundedHeight
              ? math.min(normalizedInset, constraints.maxHeight)
              : normalizedInset;
        }

        Widget withLayoutInset(double inset) {
          return Padding(
            padding: EdgeInsets.only(bottom: boundedInset(inset)),
            child: workspace,
          );
        }

        Widget withVisualInset(double inset) {
          return Transform.translate(
            offset: Offset(0, -boundedInset(inset)),
            child: workspace,
          );
        }

        final Widget content;
        if (translateVisualInset) {
          content = ValueListenableBuilder<double>(
            valueListenable: visualInset,
            child: workspace,
            builder: (context, inset, workspace) => withVisualInset(inset),
          );
        } else if (layoutInset == null) {
          content = ValueListenableBuilder<double>(
            valueListenable: visualInset,
            child: workspace,
            builder: (context, inset, workspace) => withLayoutInset(inset),
          );
        } else {
          content = withLayoutInset(layoutInset!);
        }

        return ClipRect(child: content);
      },
    );
  }
}

/// Keeps live terminal updates from invalidating the keyboard transition.
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
