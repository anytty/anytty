import 'package:flutter/widgets.dart';
import 'package:flutter/gestures.dart';

import '../../../shared/presentation/anytty_rive.dart';

/// Keep hands on the top edge and feet/tail on the actual bottom edge,
/// including accessibility text scaling, without stretching the artwork.
final class BrowserPerchedMascot extends StatefulWidget {
  const BrowserPerchedMascot({
    super.key,
    required this.visible,
    required this.child,
  });
  final bool visible;
  final Widget child;

  @override
  State<BrowserPerchedMascot> createState() => _BrowserPerchedMascotState();
}

final class _BrowserPerchedMascotState extends State<BrowserPerchedMascot> {
  Offset _gaze = Offset.zero;

  @override
  void initState() {
    super.initState();
    GestureBinding.instance.pointerRouter.addGlobalRoute(_followPointer);
  }

  void _followPointer(PointerEvent event) {
    if (!widget.visible ||
        !TickerMode.valuesOf(context).enabled ||
        (event is! PointerDownEvent &&
            event is! PointerMoveEvent &&
            event is! PointerHoverEvent)) {
      return;
    }
    final box = context.findRenderObject();
    if (box is! RenderBox || !box.hasSize) return;
    final point = box.globalToLocal(event.position);
    final next = Offset(
      ((point.dx - (box.size.width - 156 + 77)) / 130).clamp(-1.0, 1.0),
      ((point.dy - 36) / 130).clamp(-1.0, 1.0),
    );
    if (next != _gaze) setState(() => _gaze = next);
  }

  @override
  void dispose() {
    GestureBinding.instance.pointerRouter.removeGlobalRoute(_followPointer);
    super.dispose();
  }

  Widget _layer(String asset, Rect crop, Offset gaze) => Visibility(
    visible: widget.visible,
    maintainState: true,
    child: AnyttyRive(
      asset: asset,
      timeline: 'idle',
      seconds: 14,
      loop: true,
      visible: widget.visible,
      crop: crop,
      gaze: gaze,
    ),
  );
  @override
  Widget build(BuildContext context) => TweenAnimationBuilder<Offset>(
    tween: Tween(begin: Offset.zero, end: _gaze),
    duration: const Duration(milliseconds: 160),
    builder: (context, gaze, _) => Stack(
      children: [
        Positioned(
          top: 0,
          right: 16,
          width: 140,
          height: 128,
          child: _layer(
            'browser-back',
            const Rect.fromLTWH(0, 0, 140, 128),
            gaze,
          ),
        ),
        Positioned(
          bottom: 0,
          right: 16,
          width: 140,
          height: 46,
          child: _layer(
            'browser-back',
            const Rect.fromLTWH(0, 128, 140, 46),
            gaze,
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: 70, bottom: 46),
          child: widget.child,
        ),
        Positioned(
          top: 0,
          right: 16,
          width: 140,
          height: 128,
          child: _layer(
            'browser-front',
            const Rect.fromLTWH(0, 0, 140, 128),
            gaze,
          ),
        ),
      ],
    ),
  );
}
