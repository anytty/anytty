import 'package:flutter/widgets.dart';

import '../../../shared/presentation/anytty_rive.dart';

/// Keep hands on the top edge and feet/tail on the actual bottom edge,
/// including accessibility text scaling, without stretching the artwork.
final class BrowserPerchedMascot extends StatelessWidget {
  const BrowserPerchedMascot({
    super.key,
    required this.visible,
    required this.child,
  });
  final bool visible;
  final Widget child;

  Widget _layer(String asset, Rect crop) => Visibility(
    visible: visible,
    maintainState: true,
    child: AnyttyRive(
      asset: asset,
      timeline: 'idle',
      seconds: 14,
      loop: false,
      visible: visible,
      crop: crop,
    ),
  );
  @override
  Widget build(BuildContext context) => Stack(
    children: [
      Positioned(
        top: 0,
        right: 16,
        width: 140,
        height: 128,
        child: _layer('browser-back', const Rect.fromLTWH(0, 0, 140, 128)),
      ),
      Positioned(
        bottom: 0,
        right: 16,
        width: 140,
        height: 46,
        child: _layer('browser-back', const Rect.fromLTWH(0, 128, 140, 46)),
      ),
      Padding(
        padding: const EdgeInsets.only(top: 70, bottom: 46),
        child: child,
      ),
      Positioned(
        top: 0,
        right: 16,
        width: 140,
        height: 128,
        child: _layer('browser-front', const Rect.fromLTWH(0, 0, 140, 128)),
      ),
    ],
  );
}
