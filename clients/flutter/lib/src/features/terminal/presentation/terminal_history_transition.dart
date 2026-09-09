import 'dart:async';

import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../shared/presentation/anytty_brand_mark.dart';
import 'terminal_canvas.dart';

/// Keeps history layout and positioning behind one stable entry surface.
final class TerminalHistoryTransition extends StatefulWidget {
  const TerminalHistoryTransition({
    super.key,
    required this.active,
    required this.ready,
    required this.background,
    required this.foreground,
    required this.fallback,
    required this.child,
    this.onCancel,
  });

  static const minimumDuration = Duration(milliseconds: 700);
  final bool active;
  final bool ready;
  final Color background;
  final Color foreground;
  final Widget fallback;
  final Widget child;
  final VoidCallback? onCancel;

  @override
  State<TerminalHistoryTransition> createState() =>
      _TerminalHistoryTransitionState();
}

final class _TerminalHistoryTransitionState
    extends State<TerminalHistoryTransition> {
  Timer? _minimumTimer;
  int _epoch = 0;
  bool _minimumElapsed = false;
  bool _overlayVisible = false;

  @override
  void initState() {
    super.initState();
    if (widget.active) _begin();
  }

  @override
  void didUpdateWidget(TerminalHistoryTransition oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.active == oldWidget.active) return;
    _minimumTimer?.cancel();
    _epoch++;
    if (widget.active) {
      _begin();
    } else {
      _overlayVisible = false;
      _minimumElapsed = false;
    }
  }

  void _begin() {
    _minimumElapsed = false;
    _overlayVisible = true;
    final epoch = _epoch;
    // Measure visible time, not time spent waiting for the first frame.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || epoch != _epoch || !widget.active) return;
      _minimumTimer = Timer(TerminalHistoryTransition.minimumDuration, () {
        if (!mounted || epoch != _epoch) return;
        setState(() => _minimumElapsed = true);
      });
    });
  }

  @override
  void dispose() {
    _epoch++;
    _minimumTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final reveal = widget.active && widget.ready && _minimumElapsed;
    final label = anyttyText(context, en: 'Entering history', zh: '正在进入历史模式');
    return Stack(
      fit: StackFit.expand,
      children: [
        TerminalHistoryPresentation(
          ready: reveal,
          fallbackInteractive: !widget.active,
          fallback: widget.fallback,
          child: widget.child,
        ),
        if (_overlayVisible && !(reveal && AnyttyMotion.disabled(context)))
          Positioned.fill(
            child: AnimatedOpacity(
              key: const ValueKey('history-entry-overlay'),
              opacity: reveal ? 0 : 1,
              duration: AnyttyMotion.disabled(context)
                  ? Duration.zero
                  : const Duration(milliseconds: 160),
              onEnd: () {
                if (mounted && reveal) {
                  setState(() => _overlayVisible = false);
                }
              },
              child: ColoredBox(
                color: widget.background,
                child: Stack(
                  children: [
                    Positioned.fill(
                      child: Center(
                        child: SingleChildScrollView(
                          padding: const EdgeInsets.all(24),
                          child: Semantics(
                            liveRegion: true,
                            label: label,
                            child: ExcludeSemantics(
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  const AnyttyBrandLoader(
                                    height: 48,
                                    scene: AnyttyMascotScene.history,
                                  ),
                                  const SizedBox(height: 12),
                                  Text(
                                    label,
                                    textAlign: TextAlign.center,
                                    style: TextStyle(
                                      color: widget.foreground,
                                      fontSize: 13,
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ),
                      ),
                    ),
                    if (widget.onCancel != null)
                      Positioned(
                        top: 4,
                        right: 4,
                        child: IconButton(
                          constraints: const BoxConstraints.tightFor(
                            width: 48,
                            height: 48,
                          ),
                          tooltip: anyttyText(
                            context,
                            en: 'Return to live',
                            zh: '返回实时模式',
                          ),
                          color: widget.foreground,
                          onPressed: widget.onCancel,
                          icon: const Icon(LucideIcons.x, size: 20),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),
      ],
    );
  }
}
